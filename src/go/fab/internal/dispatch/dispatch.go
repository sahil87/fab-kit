// Package dispatch is the headless, tmux-independent process manager backing
// the `fab dispatch` command family (start/status/logs/kill/clean). It is the
// CLI adapter for cross-harness stage dispatch: it launches a stage's resolved
// spawn command DETACHED, tracks it via a per-change state directory, and
// exposes a byte-stable poll/logs/kill/clean surface.
//
// State layout — .fab-dispatch/{4-char-change-id}/ at the repository root
// (alongside .fab-status.yaml, already gitignored via the scaffold `.fab-*`
// pattern). The 4-char change ID keys the dir so it is stable
// across `fab change rename`; each git worktree gets its own dir (repo-root
// relative). Per-stage files:
//
//	{stage}-prompt.md   — the stage prompt piped to the dispatched command's stdin
//	{stage}.yaml        — pid/pgid/spawn_cmd/started_at/timeout (this package)
//	{stage}.log         — combined stdout+stderr of the dispatched command
//	{stage}.exit        — the exit code (`echo $? > ...`); its presence = "finished"
//	{stage}-result.yaml — the dispatched agent's result (contract; 3d owns content)
//
// The launch is supervisor-free: `sh -c '<cmd> < prompt > log 2>&1; echo $? >
// exit'` makes the SHELL the supervisor (records the exit code itself), and
// Launch's SysProcAttr{Setsid:true} detaches that shell into its own session/
// process group so the dispatch survives the orchestrator dying — no Go process
// remains in the loop. (The intake's `setsid sh -c` form describes the intent;
// the detach is done by the Setsid syscall attr, not a `setsid` binary prefix,
// so the recorded pid tracks the live worker — see dispatch_posix.WrapperArgv.)
// The process-launch and process-group-signal syscalls are POSIX-only and live in
// the build-tagged dispatch_posix.go / dispatch_windows.go split (mirroring
// cmd/fab/pane_process_{linux,darwin}.go). This file holds the
// platform-independent core: state types, the five-state derivation, path
// helpers, and YAML load/save.
package dispatch

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/atomicfile"
	"gopkg.in/yaml.v3"
)

// State is one of the five byte-stable status states reported by
// `fab dispatch status`. The values are the exact strings printed — they are
// the cross-adapter contract (docs/specs/harness-adapters.md § five-state
// machine), so they are named constants, never inline literals.
type State string

const (
	// StateRunning: pid alive AND the exit file is absent.
	StateRunning State = "running"
	// StateDone: exit == 0 AND {stage}-result.yaml present.
	StateDone State = "done"
	// StateFailed: exit present AND != 0 (includes 124, the POSIX timeout code).
	StateFailed State = "failed"
	// StateFailedNoResult: exit == 0 BUT {stage}-result.yaml absent — a contract
	// violation, NOT done: the process exited clean but never wrote its result.
	StateFailedNoResult State = "failed (no-result)"
	// StateOrphaned: pid dead AND the exit file is absent — reboot / kill -9 /
	// crash, so no exit code was ever recorded.
	StateOrphaned State = "orphaned"
)

// DirName is the repo-root-relative state directory name. Every .fab-dispatch/
// dir lives directly under the repository root (filepath.Dir(fabRoot)).
const DirName = ".fab-dispatch"

// File-suffix components. Each per-stage file is "{stage}" + suffix.
const (
	promptSuffix = "-prompt.md"
	yamlSuffix   = ".yaml"
	logSuffix    = ".log"
	exitSuffix   = ".exit"
	resultSuffix = "-result.yaml"
)

// Dispatch is the persisted state of one (change, stage) dispatch — the content
// of {stage}.yaml. File paths are derived from the dir + stage (see the Path
// helpers), never stored, so the record stays a pure descriptor of the launched
// process. Timeout is the --timeout value in seconds, omitted (zero) when unset.
type Dispatch struct {
	PID       int    `yaml:"pid"`
	PGID      int    `yaml:"pgid"`
	SpawnCmd  string `yaml:"spawn_cmd"`
	StartedAt string `yaml:"started_at"`
	Timeout   int    `yaml:"timeout,omitempty"`
}

// DirFor returns the .fab-dispatch/{id}/ directory for a change ID, rooted at
// repoRoot (the repository root, i.e. filepath.Dir(fabRoot)).
func DirFor(repoRoot, id string) string {
	return filepath.Join(repoRoot, DirName, id)
}

// WrapperArgv composes the detached-launch argv:
//
//	sh -c '<cmd> < {prompt} > {log} 2>&1; echo $? > {exit}'
//
// With timeoutSecs > 0 the resolved command is wrapped in POSIX `timeout`:
//
//	sh -c 'timeout <secs> <cmd> < {prompt} > {log} 2>&1; echo $? > {exit}'
//
// The whole pipeline is a single `sh -c` script string so the SHELL is the
// supervisor (it records $? itself) — no Go supervisor process remains in the
// loop. The session detach the intake's `setsid sh -c` form describes is
// performed by Launch via SysProcAttr{Setsid:true}, NOT by prefixing the
// `setsid` binary: prefixing it would double-fork (setsid forks when its caller
// is already a process-group leader, which SysProcAttr.Setsid makes the child),
// leaving the Go-recorded pid pointing at a `setsid` process that exits
// immediately while the real worker runs under an untracked pid — breaking
// liveness/refuse-if-running/kill. One detach mechanism, the trackable one.
// Timeout is enforced entirely inside the wrapper (no Go timer, no daemon); a
// timed-out command exits 124 (POSIX convention), surfacing as `failed` via the
// normal exit-code path. Paths are single-quoted defensively; cmd is the
// resolved spawn command inserted verbatim (its own quoting is the
// resolver's/user's concern, per the verbatim pass-through philosophy).
//
// This is pure string composition (no syscalls), so it lives in the
// platform-independent core rather than the build-tagged launch files — the
// argv contract is identical on every platform, even where Launch is
// unsupported (Windows). Only the process launch/signal syscalls are split.
func WrapperArgv(cmd, promptPath, logPath, exitPath string, timeoutSecs int) []string {
	inner := cmd
	if timeoutSecs > 0 {
		inner = "timeout " + strconv.Itoa(timeoutSecs) + " " + cmd
	}
	script := fmt.Sprintf("%s < %s > %s 2>&1; echo $? > %s",
		inner, shellQuote(promptPath), shellQuote(logPath), shellQuote(exitPath))
	return []string{"sh", "-c", script}
}

// shellQuote wraps s in single quotes, escaping any embedded single quote via
// the '\” idiom. State-dir paths are fab-controlled (repo root + .fab-dispatch
// + stage name), so this is defensive rather than adversarial.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// PromptPath / YAMLPath / LogPath / ExitPath / ResultPath return the per-stage
// file paths inside a dispatch dir. dir is a DirFor result.
func PromptPath(dir, stage string) string { return filepath.Join(dir, stage+promptSuffix) }
func YAMLPath(dir, stage string) string   { return filepath.Join(dir, stage+yamlSuffix) }
func LogPath(dir, stage string) string    { return filepath.Join(dir, stage+logSuffix) }
func ExitPath(dir, stage string) string   { return filepath.Join(dir, stage+exitSuffix) }
func ResultPath(dir, stage string) string { return filepath.Join(dir, stage+resultSuffix) }

// Save writes the Dispatch record to {stage}.yaml atomically (via
// internal/atomicfile), creating the dispatch dir if needed.
func Save(dir, stage string, d *Dispatch) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dispatch dir: %w", err)
	}
	data, err := yaml.Marshal(d)
	if err != nil {
		return fmt.Errorf("marshal dispatch state: %w", err)
	}
	if err := atomicfile.WriteFile(YAMLPath(dir, stage), data, 0o644); err != nil {
		return fmt.Errorf("write dispatch state: %w", err)
	}
	return nil
}

// Load reads and parses {stage}.yaml. os.IsNotExist(err) distinguishes "no such
// dispatch" from a genuine read/parse failure.
func Load(dir, stage string) (*Dispatch, error) {
	data, err := os.ReadFile(YAMLPath(dir, stage))
	if err != nil {
		return nil, err
	}
	var d Dispatch
	if err := yaml.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("parse dispatch state: %w", err)
	}
	return &d, nil
}

// ReadExit reads {stage}.exit. It returns (present, code): present is false when
// the file is absent (the "process still running / no code recorded" signal);
// when present, code is the parsed integer (an unparseable/empty file reads as
// code 0 present=true — a written-but-garbage exit file is still "finished",
// and a clean-exit-no-result is the more conservative reading). A non-IsNotExist
// read error is returned so callers can surface it rather than mis-derive a
// state.
func ReadExit(dir, stage string) (present bool, code int, err error) {
	data, err := os.ReadFile(ExitPath(dir, stage))
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, fmt.Errorf("read exit file: %w", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return true, 0, nil
	}
	code, convErr := strconv.Atoi(trimmed)
	if convErr != nil {
		// A garbage exit file still means the process finished; treat as code 0
		// so a missing result surfaces as failed (no-result) rather than done.
		return true, 0, nil
	}
	return true, code, nil
}

// ResultPresent reports whether {stage}-result.yaml exists.
func ResultPresent(dir, stage string) bool {
	_, err := os.Stat(ResultPath(dir, stage))
	return err == nil
}

// DeriveState computes the reported status from the observed signals — the pure
// five-state machine (kept free of I/O so it is exhaustively table-testable):
//
//	exit absent, pid alive        → running
//	exit absent, pid dead         → orphaned
//	exit present, code != 0       → failed  (includes 124 timeout)
//	exit present, code == 0, result present → done
//	exit present, code == 0, result absent  → failed (no-result)
//
// A clean exit (code 0) is necessary but NOT sufficient for done — the result
// file must exist too. That is the crux distinguishing a well-behaved success
// from an agent that exited 0 without honoring the result contract.
func DeriveState(exitPresent bool, exitCode int, resultPresent, alive bool) State {
	if !exitPresent {
		if alive {
			return StateRunning
		}
		return StateOrphaned
	}
	if exitCode != 0 {
		return StateFailed
	}
	if resultPresent {
		return StateDone
	}
	return StateFailedNoResult
}

// Tail returns the last n lines of data (Go-side, no external `tail`). n <= 0
// returns the whole content unchanged. A trailing newline is treated as a line
// terminator, not an empty final line, so `Tail(data, 1)` on "a\nb\n" yields
// "b\n".
func Tail(data []byte, n int) []byte {
	if n <= 0 || len(data) == 0 {
		return data
	}
	s := string(data)
	// Split off a single trailing newline so it doesn't count as a blank line.
	trailing := ""
	if strings.HasSuffix(s, "\n") {
		trailing = "\n"
		s = s[:len(s)-1]
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return data
	}
	return []byte(strings.Join(lines[len(lines)-n:], "\n") + trailing)
}
