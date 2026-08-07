// Package dispatch is the headless, tmux-independent process manager backing
// the `fab dispatch` command family (start/restart/status/logs/kill/clean). It is
// the CLI adapter for cross-harness stage dispatch: it launches a stage's resolved
// spawn command DETACHED, tracks it via a per-change state directory, and
// exposes a byte-stable poll/logs/kill/clean surface. `restart` relaunches from the
// persisted {stage}-prompt.md over the same launch path, so it needs no state of
// its own here (a restart is a fresh attempt under last-attempt-only).
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
	"errors"
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

// Mode is the launch mechanism a dispatch used. It is DERIVED from the persisted
// record (see Dispatch.Mode), never stored, so a record written before pane mode
// existed reads as ModeHeadless without a migration.
type Mode string

const (
	// ModeHeadless: the detached `sh -c` wrapper — the tmux-independent path
	// observed via {stage}.exit + pid liveness (the five-state machine). Selected
	// by --headless/--timeout, or by auto outside tmux (see SelectMode).
	ModeHeadless Mode = "headless"
	// ModePane: an interactive worker in a tmux window, observed via
	// {stage}-result.yaml presence + pane liveness (the three-state subset — see
	// DerivePaneState). Selected by --pane/--server, or by auto inside tmux.
	ModePane Mode = "pane"
)

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
// worker. Timeout is the --timeout value in seconds, omitted (zero) when unset.
//
// The record covers BOTH launch modes with one loader, one save path, and one
// refuse-if-running check — the mode is DERIVED from which identity fields are
// populated (see Mode), so there is no stored mode discriminator to fall out of
// sync with the fields:
//
//	headless — PID/PGID identify the detached worker shell; Pane/Window/Server empty
//	pane     — Pane/Window(/Server) identify the tmux worker pane; PID/PGID unset
//	           (liveness is a pane property, not a pid property)
//
// Every mode-specific field is `omitempty`, so a headless record serializes
// byte-identically to before pane mode existed and a pane record carries no
// meaningless `pid: 0`.
type Dispatch struct {
	PID       int    `yaml:"pid,omitempty"`
	PGID      int    `yaml:"pgid,omitempty"`
	SpawnCmd  string `yaml:"spawn_cmd"`
	StartedAt string `yaml:"started_at"`
	Timeout   int    `yaml:"timeout,omitempty"`
	// Pane is the tmux pane ID (e.g. "%17") of an interactive pane dispatch. Its
	// presence IS the pane-mode signal (see Mode/IsPane).
	Pane string `yaml:"pane,omitempty"`
	// Window is the pane dispatch's IDENTITY STRING (WindowName(id, stage)) —
	// recorded for identification/debugging; the pane ID is what liveness and kill
	// target. Its MEANING depends on the pane shape the dispatch took: the tmux
	// WINDOW NAME for a new-window dispatch, or the tmux PANE TITLE for one that
	// split the dispatching agent's window (a split pane has no window name of its
	// own). Deliberately NOT split into two fields — the string is identical either
	// way, so a second field would carry no information and would break every
	// existing record's schema for nothing.
	Window string `yaml:"window,omitempty"`
	// Server is the tmux socket label (`tmux -L <name>`) the pane lives on, empty
	// for the default socket. Persisted so status/kill reach the same server the
	// start reached, without the caller re-supplying --server.
	Server string `yaml:"server,omitempty"`
}

// IsPane reports whether this record describes an interactive pane dispatch. The
// pane ID's presence is the signal — a record written before pane mode existed
// has none and therefore reads as headless.
func (d *Dispatch) IsPane() bool { return d.Pane != "" }

// Mode returns the derived launch mode (ModePane when a pane ID is recorded, else
// ModeHeadless).
func (d *Dispatch) Mode() Mode {
	if d.IsPane() {
		return ModePane
	}
	return ModeHeadless
}

// WindowName composes a pane dispatch's IDENTITY STRING:
// "fab-{4-char-change-id}-{stage}".
//
// One string, two carriers, by pane shape (see SelectPaneShape): it is the tmux
// WINDOW NAME when the worker opens in its own window (OpenWindow), and the tmux
// PANE TITLE when the worker splits the dispatching agent's window (OpenSplitPane,
// where the window is the dispatcher's and has its own name). The function name is
// kept for its call sites and its record field; what it composes is the same
// string either way.
//
// This is a DEDICATED dispatch convention and deliberately carries neither the
// operator's `»` (U+00BB) enrollment prefix nor its `›` (U+203A) done marker:
// those assert that a window is in the operator's monitored set and that the
// operator owns its lifecycle, which a pipeline dispatch does not have.
// Pre-marking would make the operator's tab bar misreport what it tracks. An
// operator that genuinely enrolls the window still adds the marker through its
// own idempotent `fab pane window-name ensure-prefix` primitive. The rule carries
// over to BOTH shapes — a split worker's pane title is unmarked for the same
// reason.
func WindowName(id, stage string) string { return "fab-" + id + "-" + stage }

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

// recordedPanes returns the set of tmux pane IDs recorded by every dispatch record
// in the checkout that lives on the given tmux SERVER — the identity source
// pane-worker PLACEMENT keys on (see SiblingDispatchPane).
//
// It walks .fab-dispatch/*/ under repoRoot and loads each {stage}.yaml through the
// package's own Load, so the record schema has one reader. Records with no Pane
// (every headless dispatch) contribute nothing.
//
// The SERVER FILTER is exact equality against Dispatch.Server, because a tmux pane
// ID is per-SOCKET, not global: a `%17` recorded by a `--server work` dispatch names
// a completely different pane from the `%17` in a default-socket window, so an
// unfiltered set could false-match and stack a worker onto an unrelated pane.
// Default-socket dispatches record Server "" and are therefore matched by a
// default-socket probe (server ""), which is the same equality test — no special
// case.
//
// Scope is ALL record dirs, not just the active change's: nothing stops one tmux
// window from hosting two changes' workers, and an extra directory listing is
// cheaper than a misplaced pane. Over-collecting is safe because the caller
// INTERSECTS this set with one window's live pane list — a pane belonging to
// another window (or a dead one) simply never matches.
//
// Errors are REPORTED but never fatal, and the set returned alongside them is the
// partial one collected so far: an absent .fab-dispatch/ tree is the benign
// first-dispatch case (empty set, nil error), while an unreadable dir entry or a
// corrupt record is a real failure the caller surfaces as a warning (see
// SiblingDispatchPane) without failing a dispatch that would otherwise launch —
// placement is cosmetic (§ SplitTarget). Returning the partial set rather than
// discarding it is strictly safer: a missing record can only fail to FIND a
// sibling (degrading to the first-worker carve), never invent one, since the
// caller still intersects with the window's live pane list.
func recordedPanes(repoRoot, server string) (map[string]bool, error) {
	panes := map[string]bool{}
	root := filepath.Join(repoRoot, DirName)
	dirs, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return panes, nil
		}
		return panes, fmt.Errorf("read dispatch state dir: %w", err)
	}
	var errs []error
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		dir := filepath.Join(root, d.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("read dispatch dir %s: %w", d.Name(), err))
			continue
		}
		for _, f := range files {
			name := f.Name()
			// {stage}.yaml only — {stage}-result.yaml is the WORKER's result file,
			// which also ends in .yaml but is not a dispatch record.
			if !strings.HasSuffix(name, yamlSuffix) || strings.HasSuffix(name, resultSuffix) {
				continue
			}
			rec, err := Load(dir, strings.TrimSuffix(name, yamlSuffix))
			if err != nil {
				errs = append(errs, fmt.Errorf("read dispatch record %s/%s: %w", d.Name(), name, err))
				continue
			}
			if rec.IsPane() && rec.Server == server {
				panes[rec.Pane] = true
			}
		}
	}
	return panes, errors.Join(errs...)
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

// DerivePaneState computes the reported status of an INTERACTIVE PANE dispatch —
// the pure three-state subset (kept free of I/O so it is exhaustively
// table-testable, exactly like DeriveState):
//
//	result present            → done
//	result absent, pane alive → running
//	result absent, pane dead  → orphaned
//
// Two properties are load-bearing:
//
//   - RESULT PRESENCE WINS over pane liveness. An interactive worker never exits
//     on task completion — it finishes and sits at its prompt — so a
//     liveness-first rule would report `running` forever. The result file is
//     already the contract's success token for the other adapters, so reusing it
//     keeps ONE success definition across all three.
//   - StateFailed and StateFailedNoResult are UNREACHABLE here. There is no
//     exit-code channel in pane mode, so a crashed or killed worker collapses
//     into orphaned. The five state STRINGS stay byte-stable (they are the
//     cross-adapter contract); a pane dispatch simply emits a subset of three.
//
// This is deliberately a separate function rather than extra parameters on
// DeriveState: the headless five-state machine is a byte-stable documented
// contract, and threading pane liveness through it would couple two different
// observation models in one function.
func DerivePaneState(resultPresent, paneAlive bool) State {
	if resultPresent {
		return StateDone
	}
	if paneAlive {
		return StateRunning
	}
	return StateOrphaned
}

// PointerPrompt composes the one-line prompt a pane worker receives at spawn.
//
// The FULL stage prompt is never delivered through tmux: a multi-thousand-token
// prompt cannot ride send-keys or argv reliably, so `start` persists it to
// {stage}-prompt.md (the same path and writer the headless path uses) and the
// worker gets a pointer to that path instead. Embedding the pointer at window
// creation — as the interactive command's single quoted prompt argument, per
// `_cli-agents.md` § Spawn Composition — also sidesteps the printed-prompt trap
// entirely: there is no pre-existing input buffer to probe when the window is
// created with its prompt already attached.
//
// promptPath should be REPO-RELATIVE: the window's cwd is the repo root, and a
// relative path keeps the pointer readable and portable across worktrees.
func PointerPrompt(promptPath string) string {
	return "Read " + promptPath + " and execute it."
}

// WindowCommand composes the tmux new-window shell-command argument for a pane
// dispatch: the resolved session command followed by the pointer prompt as its
// single QUOTED argument (the `_cli-agents.md` § Spawn Composition form).
//
// The pointer is shell-quoted rather than wrapped in bare single quotes, because
// the prompt path is derived from the repository path and a repo checked out
// under a directory containing a single quote (`/home/me/sahil's-repo/...`) would
// otherwise terminate the quoted argument early — breaking the new-window command
// and letting the remainder of the path be interpreted by the window's shell.
// § Spawn Composition states the rule directly ("shell-escape any user-supplied
// text before embedding it"); this is the one place pane mode embeds such text.
//
// resolvedCmd is inserted VERBATIM, per the resolver's pass-through philosophy:
// it is the provider's own session_command and carries deliberate shell
// expansions (e.g. `$(basename "$(pwd)")` in the built-in claude default) that
// must expand inside the new window.
func WindowCommand(resolvedCmd, pointer string) string {
	return resolvedCmd + " " + shellQuote(pointer)
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
