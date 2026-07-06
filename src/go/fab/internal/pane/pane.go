package pane

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// AgentStateOption is the tmux pane user option that carries an agent's
// lifecycle state, written by run-kit's `rk agent-setup` global agent-harness
// hooks and read by the fab pane commands. Its value is
// "<state>:<epoch_seconds>" where state is one of the AgentState* constants
// below. fab is a pure CONSUMER of this convention — it never writes the
// option (run-kit owns the schema); it reads it with plain tmux commands, so
// there is no dependency on run-kit software being installed.
const AgentStateOption = "@rk_agent_state"

// Agent lifecycle states carried by AgentStateOption:
//   - active  — turn in progress
//   - waiting — blocked on a human (permission prompt / menu / elicitation)
//   - idle    — turn complete
//
// An absent option, an unknown token, or a value without a parseable epoch
// suffix is treated as unknown (no state).
const (
	AgentStateActive  = "active"
	AgentStateWaiting = "waiting"
	AgentStateIdle    = "idle"
)

// WithServer prepends "-L <server>" to a tmux argument list when server is
// non-empty, and returns args unchanged otherwise. Callers use this to build
// the argv for `exec.Command("tmux", ...)` so the --server/-L CLI flag is
// plumbed through to every tmux invocation. The input args slice is never
// mutated; a new slice is allocated when a prefix is added.
//
// Exported (rather than unexported as originally drafted) so that the
// `cmd/fab` package — which owns the pane subcommand wiring — shares the
// single canonical argv builder instead of duplicating the logic.
func WithServer(server string, args ...string) []string {
	if server == "" {
		return args
	}
	return append([]string{"-L", server}, args...)
}

// RunCmd executes an external command, capturing stdout and stderr
// separately. Returns the raw stdout string (untrimmed — callers that need
// trimming do it themselves, so capture-style output is never altered), the
// raw stderr bytes, and the exec error. Generalizes the ReadWindowName
// capture pattern for any subprocess (tmux, git, wt) so call sites stop
// discarding the child's diagnostic.
func RunCmd(name string, args ...string) (string, []byte, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.Bytes(), err
}

// StderrError enriches err with the trimmed child stderr when present, so a
// subprocess failure surfaces the child's diagnostic (the self-correction
// signal for agent-facing CLI output) instead of a bare "exit status 1".
// Returns err unchanged when stderr is empty; the original error remains
// unwrappable via errors.Is/As.
func StderrError(err error, stderr []byte) error {
	msg := strings.TrimSpace(string(stderr))
	if msg == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, msg)
}

// IsPaneMissing reports whether tmux stderr indicates a missing pane (as
// opposed to other tmux failures such as a dead server or socket error).
// Matching is case-insensitive substring, mirroring tmux's "can't find pane"
// / "no such pane" wording across versions. Shared by ValidatePane and the
// window-name verbs' exit-code mapping.
func IsPaneMissing(stderr []byte) bool {
	s := strings.ToLower(string(stderr))
	return strings.Contains(s, "can't find pane") ||
		strings.Contains(s, "no such pane") ||
		(strings.Contains(s, "pane") && strings.Contains(s, "not found"))
}

// PaneNotFoundError reports a missing tmux pane. It carries the 2-vs-3
// exit-code classification for the pane-family scheme (2 = pane missing,
// 3 = other tmux failure) on the error VALUE — call sites detect it via
// errors.As instead of string matching. The message is byte-identical to the
// historical "pane <id> not found" string.
type PaneNotFoundError struct {
	Pane string
}

func (e *PaneNotFoundError) Error() string {
	return fmt.Sprintf("pane %s not found", e.Pane)
}

// PaneContext holds resolved fab context for a single tmux pane.
type PaneContext struct {
	Pane              string
	CWD               string
	WorktreeRoot      string
	WorktreeDisplay   string
	Change            *string // nil if no active change
	Stage             *string // nil if no stage
	AgentState        *string // nil if not applicable
	AgentIdleDuration *string // nil if not idle
}

// ValidatePane checks that a tmux pane exists and that the argument is its
// exact pane ID, via a single targeted probe:
//
//	tmux display-message -t <pane> -p '#{pane_id}'
//
// comparing the probe output to the argument. The comparison preserves
// ID-exactness: `-t` accepts the full tmux target grammar (window names,
// session:win.pane), but any such target resolves to a real pane ID that
// differs from the argument and is rejected — matching the previous
// `list-panes -a` exact-ID semantics without the O(server) enumeration.
//
// Error-path equivalence (verified on tmux 3.6a): a missing pane yields the
// same "pane <id> not found" error — on tmux ≥3.6 display-message exits 0
// with empty output for a missing pane (caught by the comparison); older
// tmux errors with "can't find pane" stderr (caught by the mapping). A dead
// server still fails with exit 1, now carrying tmux's connection diagnostic.
// If server is non-empty, the tmux invocation is scoped via `-L <server>`.
func ValidatePane(paneID, server string) error {
	out, stderr, err := RunCmd("tmux", WithServer(server, "display-message", "-t", paneID, "-p", "#{pane_id}")...)
	return validatePaneResult(paneID, out, stderr, err)
}

// validatePaneResult is the pure decision half of ValidatePane: it maps the
// probe's (stdout, stderr, exec error) to the validation outcome. Extracted
// for unit-testability without a tmux server.
func validatePaneResult(paneID, out string, stderr []byte, err error) error {
	if err != nil {
		if IsPaneMissing(stderr) {
			return &PaneNotFoundError{Pane: paneID}
		}
		return StderrError(fmt.Errorf("tmux display-message: %w", err), stderr)
	}
	if strings.TrimSpace(out) != paneID {
		return &PaneNotFoundError{Pane: paneID}
	}
	return nil
}

// ReadWindowName returns the current window name for a tmux pane via
// `tmux display-message -p -t <pane> '#W'`. Returns the trimmed name, the
// tmux stderr bytes (useful for exit-code mapping — callers can distinguish
// "pane missing" from other tmux errors by inspecting stderr), and any exec
// error. If server is non-empty, the tmux invocation is scoped to that server
// via `-L <server>`.
func ReadWindowName(paneID, server string) (string, []byte, error) {
	out, stderr, err := RunCmd("tmux", WithServer(server, "display-message", "-p", "-t", paneID, "#W")...)
	return strings.TrimSpace(out), stderr, err
}

// GetPanePID returns the shell PID of a tmux pane. If server is non-empty, the
// tmux invocation is scoped to that server via `-L <server>`.
func GetPanePID(paneID, server string) (int, error) {
	out, err := exec.Command("tmux", WithServer(server, "display-message", "-t", paneID, "-p", "#{pane_pid}")...).Output()
	if err != nil {
		return 0, fmt.Errorf("tmux display-message: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("parsing pane PID: %w", err)
	}
	return pid, nil
}

// ResolvePaneContext resolves the fab context for a given tmux pane.
// mainRoot is the main worktree root used for computing relative display paths.
// Pass "" if unknown — WorktreeDisplay will fall back to filepath.Base.
// If server is non-empty, the tmux invocation is scoped to that server via
// `-L <server>`; file reads and git-worktree detection are independent of the
// tmux server.
//
// Agent-state resolution is independent of whether a change is active — a pane
// running any instrumented agent in "discussion mode" (no change), a git repo
// without a fab/ directory, or a non-git directory still populates AgentState
// when the pane carries the @rk_agent_state option. It is resolved before the
// git/fab early returns for exactly that reason (see the resolution block), so
// send/map/capture agree on every pane class.
func ResolvePaneContext(paneID, mainRoot, server string) (*PaneContext, error) {
	// Get pane CWD
	out, err := exec.Command("tmux", WithServer(server, "display-message", "-t", paneID, "-p", "#{pane_current_path}")...).Output()
	if err != nil {
		return nil, fmt.Errorf("tmux display-message: %w", err)
	}
	cwd := strings.TrimSpace(string(out))

	ctx := &PaneContext{
		Pane: paneID,
		CWD:  cwd,
	}

	// Agent resolution — the AGENT axis is fully independent of the CHANGE
	// axis, so it MUST be resolved BEFORE the not-a-git-repo / no-fab-dir
	// early returns below. Otherwise a non-fab pane carrying @rk_agent_state
	// would resolve to unknown here while `pane map` (which reads the option
	// off every pane's list-panes row) shows its real state — the two readers
	// would disagree, and `pane send` would refuse (as unknown) a pane the map
	// reports as idle. Reading the option first keeps all three readers
	// (send/map/capture) in agreement for every pane class: non-git, git but
	// no fab/, and fab. Reads the @rk_agent_state pane option (written by
	// run-kit's rk agent-setup), so discussion-mode panes and non-Claude
	// agents (codex/copilot/gemini) are covered uniformly; an absent or
	// unparseable option leaves AgentState nil (unknown).
	state, idleDur := AgentDisplayFromOption(ReadAgentStateOption(paneID, server))
	if state != "" {
		ctx.AgentState = &state
	}
	if idleDur != "" {
		ctx.AgentIdleDuration = &idleDur
	}

	// Resolve git worktree root
	wtRoot, err := GitWorktreeRoot(cwd)
	if err != nil {
		// Not in a git repo
		ctx.WorktreeRoot = cwd
		ctx.WorktreeDisplay = filepath.Base(cwd) + "/"
		return ctx, nil
	}
	ctx.WorktreeRoot = wtRoot

	// Set worktree display using the same logic as pane map
	ctx.WorktreeDisplay = WorktreeDisplayPath(wtRoot, mainRoot)

	// Check for fab/ directory
	fabDir := filepath.Join(wtRoot, "fab")
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		return ctx, nil
	}

	// Read .fab-status.yaml symlink for the active change (independent axis).
	_, folderName := ReadFabCurrent(wtRoot)
	if folderName != "" {
		ctx.Change = &folderName

		// Read stage from .status.yaml
		statusPath := filepath.Join(fabDir, "changes", folderName, ".status.yaml")
		if statusFile, err := sf.Load(statusPath); err == nil {
			stage, _ := status.DisplayStage(statusFile)
			ctx.Stage = &stage
		}
	}

	return ctx, nil
}

// FindMainWorktreeRoot returns the main worktree root by parsing
// `git worktree list --porcelain`. Derives the root from one of the
// provided pane CWDs so the command works even outside the repo.
func FindMainWorktreeRoot(cwds []string) string {
	for _, cwd := range cwds {
		out, err := exec.Command("git", "-C", cwd, "worktree", "list", "--porcelain").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "worktree ") {
				return strings.TrimPrefix(line, "worktree ")
			}
		}
	}
	return ""
}

// GitWorktreeRoot returns the git worktree root for a given path.
func GitWorktreeRoot(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// WorktreeDisplayPath computes the display path for a worktree.
// Main worktree shows "(main)", others show path relative to main's parent.
func WorktreeDisplayPath(wtRoot, mainRoot string) string {
	if mainRoot != "" && wtRoot == mainRoot {
		return "(main)"
	}
	if mainRoot != "" {
		parent := filepath.Dir(mainRoot)
		rel, err := filepath.Rel(parent, wtRoot)
		if err == nil {
			return rel + "/"
		}
	}
	return filepath.Base(wtRoot) + "/"
}

// ReadFabCurrent reads .fab-status.yaml symlink and returns (displayName, folderName).
func ReadFabCurrent(wtRoot string) (string, string) {
	symlinkPath := filepath.Join(wtRoot, ".fab-status.yaml")
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return "(no change)", ""
	}
	folderName := resolve.ExtractFolderFromSymlink(target)
	if folderName == "" {
		return "(no change)", ""
	}
	return folderName, folderName
}

// parseAgentState parses the raw value of the @rk_agent_state pane option
// ("<state>:<epoch_seconds>") into its state token and epoch. It returns
// ok=false when the raw value is empty, has no ":epoch" suffix, carries a
// non-integer epoch, or names a state token outside {active, waiting, idle}
// — every "unknown" case collapses to ok=false so callers render a single
// unknown sentinel. Pure (no tmux), so the grammar is unit-testable without
// a tmux server.
func parseAgentState(raw string) (state string, epoch int64, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, false
	}
	idx := strings.LastIndex(raw, ":")
	if idx < 0 {
		return "", 0, false
	}
	state = raw[:idx]
	switch state {
	case AgentStateActive, AgentStateWaiting, AgentStateIdle:
	default:
		return "", 0, false
	}
	epoch, err := strconv.ParseInt(raw[idx+1:], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return state, epoch, true
}

// AgentDisplayFromOption converts a raw @rk_agent_state option value into a
// display state and (for idle only) an idle-duration string. It returns
// ("", "") for the unknown case (unparseable / absent / unknown token), which
// callers map to the em-dash / JSON-null sentinel. Only the idle state
// carries a duration — active/waiting describe an in-progress state with no
// meaningful "idle for" measure, mirroring the pre-divestment active-has-no-
// duration semantics.
func AgentDisplayFromOption(raw string) (state, idleDuration string) {
	st, epoch, ok := parseAgentState(raw)
	if !ok {
		return "", ""
	}
	if st != AgentStateIdle {
		return st, ""
	}
	elapsed := time.Now().Unix() - epoch
	if elapsed < 0 {
		elapsed = 0
	}
	return AgentStateIdle, FormatIdleDuration(elapsed)
}

// ReadAgentStateOption reads the raw @rk_agent_state pane user option for a
// single pane via `tmux [-L <server>] show-options -pv -t <pane>
// @rk_agent_state`. The `-v` flag returns the bare value when the option is
// SET. An *unset* option is not an empty-stdout success: on tmux 3.6a
// `show-options -pv` for a missing option exits 1 with an error on stderr and
// no stdout — so the common "no state written for this pane" case surfaces as
// a non-zero exit, handled by the error branch below (which also absorbs a
// missing pane or dead server). The error branch therefore exists to map ALL
// of these — unset option, missing pane, dead server — to the unknown state:
// send/capture validate pane existence separately, so an option-read failure
// degrades to unknown rather than erroring out.
// If server is non-empty, the invocation is scoped via `-L <server>`.
//
// An empty paneID returns "" (unknown) without touching tmux: `show-options
// -pv -t ""` would silently target the client's CURRENT pane, reading a
// wrong-pane state rather than failing — so the empty case is refused up front.
func ReadAgentStateOption(paneID, server string) string {
	if paneID == "" {
		return ""
	}
	out, _, err := RunCmd("tmux", WithServer(server, "show-options", "-pv", "-t", paneID, AgentStateOption)...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// FormatIdleDuration formats elapsed seconds into a human-readable duration.
// Uses floor division: <60s -> Ns, 60s-3599s -> Nm, >=3600s -> Nh. Formats
// the epoch-derived idle durations of the @rk_agent_state readers.
func FormatIdleDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%dh", seconds/3600)
}
