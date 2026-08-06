package dispatch

import (
	"fmt"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// This file holds the tmux side of PANE-MODE dispatch (`fab dispatch start
// --pane`): server reachability, worker-pane creation (split or new window),
// pane liveness, and pane kill.
//
// It lives in the platform-INDEPENDENT core rather than the build-tagged
// dispatch_posix.go / dispatch_windows.go split for the same reason WrapperArgv
// does: these are plain tmux SUBPROCESS calls with no syscall dependency, so the
// code compiles everywhere. Only the headless path's process launch/signal
// syscalls are platform-split. (Pane mode is still unusable where tmux is
// absent — but that surfaces as ServerReachable's actionable error, not a
// compile-time platform split.)
//
// Every tmux invocation goes through internal/pane's shared helpers — RunCmd
// (stdout/stderr/exec-error capture), WithServer (the `-L <name>` argv prefix),
// and StderrError (surface the child's diagnostic instead of a bare exit
// status) — so there is exactly one tmux argv builder and one stderr-enrichment
// convention in the binary.

// AutoReason names WHY a dispatch ended up in the mode it did — the selection
// SOURCE, distinct from the Mode itself. It exists so `fab dispatch start` can
// explain a surprising mode from its own output (the compliance-visibility
// principle) and so the auto-vs-explicit asymmetry in the tmux-reachability
// failure path (soft fallback vs hard error) keys on a named value rather than
// on re-reading the flags.
type AutoReason string

const (
	// ReasonExplicit: the caller named the mode (--pane / --headless), or named a
	// flag that only one mode can honor (--timeout ⇒ headless, --server ⇒ pane).
	// Auto did NOT fire, so the output line carries no `auto:` suffix and a
	// tmux-reachability failure stays a hard error.
	ReasonExplicit AutoReason = ""
	// ReasonAutoTmux: no explicit signal and $TMUX was set — pane mode was chosen
	// because a window would land on the server the caller is attached to.
	ReasonAutoTmux AutoReason = "auto: tmux"
	// ReasonAutoNoTmux: no explicit signal and $TMUX was unset — headless, the
	// pre-auto default, chosen because no pane would be visible to anyone.
	ReasonAutoNoTmux AutoReason = "auto: no tmux"
	// ReasonAutoUnreachable: auto selected pane from $TMUX, but the server did not
	// answer (a stale $TMUX inherited from a killed server), so the dispatch
	// SOFT-FELL-BACK to headless — fallback shape (a). Set by the caller after the
	// failed probe, never by SelectMode (which performs no I/O).
	ReasonAutoUnreachable AutoReason = "auto: tmux unreachable"
	// ReasonAutoNoSessionCommand: auto selected pane from a REACHABLE $TMUX, but
	// the resolved provider carries no session_command (a dispatch_command-only
	// provider — e.g. a headless codex tier), so the dispatch SOFT-FELL-BACK to
	// headless — fallback shape (b). Set by the caller after the composition
	// check, never by SelectMode.
	ReasonAutoNoSessionCommand AutoReason = "auto: no session_command"
)

// The two stderr notices an auto-selected pane dispatch prints when it
// soft-falls-back to headless — one per failure shape. Named so each message
// exists in exactly one place (the command prints it; tests assert it).
const (
	// FallbackNotice — shape (a): the tmux server did not answer.
	FallbackNotice = "pane auto-selection: tmux unreachable, falling back to headless"
	// FallbackNoticeNoSessionCommand — shape (b): tmux answered, but the resolved
	// provider has no session_command to open interactively. Falling back is what
	// keeps a dispatch_command-only provider working inside tmux (its stages were
	// headless before auto selection existed and must not start failing merely
	// because the caller sits in a tmux pane).
	FallbackNoticeNoSessionCommand = "pane auto-selection: provider has no session_command, falling back to headless"
)

// SelectMode resolves `fab dispatch start`'s launch mode from an
// EXPLICIT-SIGNAL-FIRST ladder, returning the mode plus the selection source.
//
// It is a PURE function — no environment reads, no tmux probe, no I/O — so the
// whole ladder is table-testable, matching DeriveState/DerivePaneState's shape in
// this package. The caller supplies tmuxEnv (os.Getenv("TMUX")) and the four
// explicit signals, each read as "was the flag SUPPLIED" (cobra's
// Flags().Changed) rather than by value, so an explicit `--timeout 0` or
// `--server ""` still counts as a signal.
//
// The ladder, in order:
//
//  1. paneFlag     → pane      (explicit)
//  2. headlessFlag → headless  (explicit)
//  3. timeoutSet   → headless  (explicit: --timeout is enforced by the headless
//     wrapper, so it can only mean headless)
//  4. serverSet    → pane      (explicit: --server exists solely to target a
//     pane's socket, so naming one means pane)
//  5. otherwise    → auto: tmuxEnv non-empty ⇒ pane, else headless
//
// paneFlag and headlessFlag are mutually exclusive at the cobra layer, so rung 1
// preceding rung 2 is a tie-break that can never be reached in practice.
//
// $TMUX is the DEFAULTING signal only — "a pane opened without -L lands on the
// server the caller is attached to". It never replaces ServerReachable, which
// stays the VALIDATION step once pane mode is chosen (and which is a real tmux
// query precisely so an explicit --pane works from a headless orchestrator, where
// $TMUX is unset). An empty $TMUX reads as unset: Go cannot distinguish the two,
// and tmux never exports an empty value.
func SelectMode(paneFlag, headlessFlag, timeoutSet, serverSet bool, tmuxEnv string) (Mode, AutoReason) {
	switch {
	case paneFlag:
		return ModePane, ReasonExplicit
	case headlessFlag:
		return ModeHeadless, ReasonExplicit
	case timeoutSet:
		return ModeHeadless, ReasonExplicit
	case serverSet:
		return ModePane, ReasonExplicit
	case tmuxEnv != "":
		return ModePane, ReasonAutoTmux
	default:
		return ModeHeadless, ReasonAutoNoTmux
	}
}

// PaneShape names WHERE a pane-mode worker is opened — the second decision pane
// mode makes, after SelectMode has already chosen pane over headless.
//
// It exists because the two-tier tmux hierarchy has two different callers of the
// same command. An OPERATOR spawns worktree agents as tmux WINDOWS; a worktree
// AGENT dispatching its own stage workers wants them beside it, as PANES
// splitting its own window (the Claude-teams layout), so a stage worker no longer
// costs a window in the operator's tab bar.
type PaneShape string

const (
	// ShapeSplit: split the dispatching agent's own window, so the worker appears
	// beside its dispatcher. Requires knowing WHICH pane the dispatcher is
	// ($TMUX_PANE) and that the pane id is meaningful on the target server.
	ShapeSplit PaneShape = "split"
	// ShapeWindow: open a new window named after the dispatch — the pre-split
	// behavior, and the fallback whenever a split cannot be placed.
	ShapeWindow PaneShape = "window"
)

// SelectPaneShape resolves WHERE a pane-mode worker opens, given the dispatching
// process's own tmux position. Like SelectMode it is a PURE function — no
// environment reads, no tmux probe, no I/O — so the whole table is testable; the
// cobra layer reads $TMUX_PANE and passes it down.
//
// The rules:
//
//  1. paneMode false     → ShapeWindow (vacuous: no pane worker is opened at all;
//     returning the pre-split shape keeps the zero value honest)
//  2. serverSet          → ShapeWindow. `--server <name>` targets a possibly
//     DIFFERENT tmux socket, on which the caller's own
//     $TMUX_PANE id is meaningless (pane ids are
//     server-global, not global) — splitting it would
//     target an unrelated pane or fail outright.
//  3. tmuxPane == ""     → ShapeWindow. A headless orchestrator dispatching with
//     an explicit --pane has no pane of its own to split.
//  4. otherwise          → ShapeSplit.
//
// Rungs 2 and 3 both reproduce the pre-split behavior byte-for-byte, which is
// what makes this change additive: only a dispatcher that IS a tmux pane on the
// target server gets the new shape.
func SelectPaneShape(paneMode, serverSet bool, tmuxPane string) PaneShape {
	switch {
	case !paneMode:
		return ShapeWindow
	case serverSet:
		return ShapeWindow
	case tmuxPane == "":
		return ShapeWindow
	default:
		return ShapeSplit
	}
}

// ServerReachable probes whether a tmux server is reachable, returning nil when
// it is and an ACTIONABLE error when it is not.
//
// The probe is a real tmux query (`tmux [-L <server>] list-sessions`), NOT an
// $TMUX environment read. That distinction is load-bearing: a dispatching
// orchestrator may itself be headless (the cross-harness case pane mode exists
// for), so an $TMUX-only gate would make --pane unusable from exactly those
// callers. With --server the probe targets that socket; without it, tmux's own
// default-socket resolution applies. This mirrors `fab resolve --pane`, where
// --server likewise replaces the $TMUX guard with socket-scoped discovery.
//
// What the probe actually distinguishes is "a server answered" from "nothing
// answered" — NOT "has sessions" from "has none". Under tmux's default
// `exit-empty on`, a server exits with its last session, so a zero-session
// server does not persist to be probed: list-sessions exits non-zero ("no server
// running") and --pane errors, correctly. Under `exit-empty off` a sessionless
// server DOES persist and list-sessions exits 0, so the probe passes; a
// subsequent new-window against it may still fail (tmux has no session to attach
// the window to), which OpenWindow surfaces as the child's own stderr via
// StderrError rather than a bare exit status. So the probe is a reachability
// gate, not a launch guarantee — the launch has its own actionable error.
//
// What the CALLER does with a non-nil return depends on how pane mode was
// selected (SelectMode's AutoReason): an EXPLICIT pane selection propagates this
// error (hard failure, nothing launched or persisted), while an AUTO selection
// soft-falls-back to headless with FallbackNotice — a stale $TMUX must not break
// a dispatch that never asked for a pane. (A missing session_command is the
// second, independent fallback shape, handled by the caller after this probe —
// see FallbackNoticeNoSessionCommand.)
func ServerReachable(server string) error {
	_, stderr, err := pane.RunCmd("tmux", pane.WithServer(server, "list-sessions")...)
	if err == nil {
		return nil
	}
	target := "the default tmux socket"
	if server != "" {
		target = fmt.Sprintf("tmux socket %q", server)
	}
	// The message names "pane mode" rather than `--pane`: pane can also be
	// selected explicitly by `--server` alone (SelectMode rung 4), and a caller
	// who passed only `--server` would be confused by guidance quoting a flag
	// they never supplied. The remedy clause still names both flags to drop.
	return pane.StderrError(fmt.Errorf(
		"pane mode requires a reachable tmux server, but %s is unreachable; start tmux (or pass --server <name>), or pass --headless (drop --pane/--server) to dispatch headless",
		target), stderr)
}

// OpenWindow creates a tmux window named name with cwd dir running cmd, and
// returns the new window's PANE ID.
//
// This is the FALLBACK pane shape (see SelectPaneShape): it is what a dispatcher
// with no pane of its own to split — a headless orchestrator passing `--pane`, or
// a caller naming another socket with `--server` — gets. A dispatcher that IS a
// tmux pane on the target server gets OpenSplitPane instead.
//
// The pane ID (not the window name) is the identity worth recording: it is
// server-global, stable for the pane's lifetime, and exempt from tmux's
// target-grammar prefix/glob resolution — so subsequent liveness probes and
// kills are exact, where a name-based target could resolve to a different
// window the user renamed into place. `-P -F '#{pane_id}'` asks new-window to
// print it, avoiding a follow-up lookup that could race a fast-exiting worker.
//
// cmd is passed as new-window's shell-command argument, so it is the WHOLE
// left-hand side including any shell expansions it carries (e.g.
// `$(basename "$(pwd)")` in the built-in claude session_command), which expand
// at invocation inside the new window — the `_cli-agents.md` § Spawn
// Composition contract.
func OpenWindow(server, name, dir, cmd string) (paneID string, err error) {
	return runPaneCreator(server, "new-window", name,
		"new-window", "-P", "-F", "#{pane_id}", "-n", name, "-c", dir, cmd)
}

// OpenSplitPane splits targetPane to create the worker's pane, titles it, and
// returns the new pane's ID. Unlike OpenWindow the worker lands in the SAME tmux
// window as targetPane — the two-tier hierarchy's inner tier.
//
// PLACEMENT is a stacked right column, matching the native agent-team layout:
//
//   - The first worker splits targetPane HORIZONTALLY (`-h`), carving the right
//     half of the dispatcher's window out as the worker column.
//   - Every later worker splits the LAST live worker pane VERTICALLY (`-v`), so
//     workers stack down that column instead of shrinking the dispatcher further.
//
// The caller supplies the already-chosen target and direction (see
// SplitTarget/SiblingDispatchPane); this function only executes the split, so the
// placement decision stays inspectable and this stays a thin tmux wrapper.
//
// TITLE: a split pane has no window name of its own — its window belongs to the
// dispatcher — so the dispatch's identity string rides the PANE TITLE instead
// (`select-pane -T`). A title-set failure is NON-FATAL: the worker is already
// running and its pane ID (the real identity, which every later probe keys on) is
// already in hand, so refusing the dispatch over a cosmetic label would be a
// strictly worse outcome. It is returned as titleErr for the caller to warn about.
//
// cmd is passed as split-window's shell-command argument, so — exactly as in
// OpenWindow — it is the WHOLE left-hand side including its own shell expansions,
// which expand at invocation inside the new pane.
func OpenSplitPane(server, targetPane, direction, title, dir, cmd string) (paneID string, titleErr error, err error) {
	paneID, err = runPaneCreator(server, "split-window", title,
		"split-window", direction, "-t", targetPane, "-P", "-F", "#{pane_id}", "-c", dir, cmd)
	if err != nil {
		return "", nil, err
	}
	if _, stderr, terr := pane.RunCmd("tmux", pane.WithServer(server,
		"select-pane", "-t", paneID, "-T", title)...); terr != nil {
		titleErr = pane.StderrError(fmt.Errorf("tmux select-pane -T: %w", terr), stderr)
	}
	return paneID, titleErr, nil
}

// runPaneCreator executes a tmux command that PRINTS the created pane's id
// (`-P -F '#{pane_id}'`) and returns that id, applying one convention to both
// creators: the child's own stderr is surfaced via pane.StderrError instead of a
// bare exit status, and an empty id is an error rather than a silently
// unidentifiable dispatch. verb/label only shape the diagnostics.
func runPaneCreator(server, verb, label string, args ...string) (string, error) {
	out, stderr, err := pane.RunCmd("tmux", pane.WithServer(server, args...)...)
	if err != nil {
		return "", pane.StderrError(fmt.Errorf("tmux %s: %w", verb, err), stderr)
	}
	paneID := strings.TrimSpace(out)
	if paneID == "" {
		return "", fmt.Errorf("tmux %s reported no pane id for %q", verb, label)
	}
	return paneID, nil
}

// DispatchTitlePrefix is the prefix every dispatch identity string carries
// ("fab-{id}-{stage}", see WindowName). SiblingDispatchPane matches on it to tell
// existing worker panes apart from the dispatcher's own pane and from any
// unrelated pane the user split off by hand.
const DispatchTitlePrefix = "fab-"

// Split direction arguments for OpenSplitPane, named so the two placement rules
// are not bare tmux flags at the call site.
const (
	// SplitRight carves the worker column out of the dispatcher's pane.
	SplitRight = "-h"
	// SplitBelow stacks a worker under the previous one, inside that column.
	SplitBelow = "-v"
)

// SiblingDispatchPane returns the LAST live dispatch worker pane in targetPane's
// window, or "" when there is none.
//
// It probes `tmux list-panes -t <targetPane> -F '#{pane_id} #{pane_title}'` —
// window-scoped, since `-t` on a pane resolves to that pane's window — and keeps
// the last row whose title carries DispatchTitlePrefix. "Last" is tmux's
// list-panes order, which is the pane-index order the stacked column was built
// in, so the newest worker is the one a further split lands under.
//
// A probe FAILURE returns ("", err) and the caller degrades to splitting the
// dispatcher's own pane: placement is cosmetic, so an unparseable or failing
// probe must never fail a dispatch that would otherwise launch fine.
func SiblingDispatchPane(server, targetPane string) (string, error) {
	out, stderr, err := pane.RunCmd("tmux", pane.WithServer(server,
		"list-panes", "-t", targetPane, "-F", "#{pane_id} #{pane_title}")...)
	if err != nil {
		return "", pane.StderrError(fmt.Errorf("tmux list-panes: %w", err), stderr)
	}
	return lastDispatchPane(out), nil
}

// lastDispatchPane is the pure parsing half of SiblingDispatchPane: it maps
// list-panes output ("<pane-id> <pane-title>" per line) to the last pane id whose
// title carries DispatchTitlePrefix, or "" when none does. Extracted so the row
// grammar is unit-testable without a tmux server.
func lastDispatchPane(out string) string {
	found := ""
	for _, line := range strings.Split(out, "\n") {
		id, title, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok || id == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(title), DispatchTitlePrefix) {
			found = id
		}
	}
	return found
}

// SplitTarget resolves WHICH pane a new worker splits and in WHICH direction,
// applying the stacked-right-column rule against the live window:
//
//	a live worker sibling exists → split THAT pane below (SplitBelow), stacking
//	                               the column
//	none (or the probe failed)   → split the dispatcher's own pane to the right
//	                               (SplitRight), creating the column
//
// The probe error is returned alongside the (already degraded) decision rather
// than replacing it, so the caller can warn without the dispatch failing.
func SplitTarget(server, dispatcherPane string) (target, direction string, probeErr error) {
	sibling, err := SiblingDispatchPane(server, dispatcherPane)
	if err != nil || sibling == "" {
		return dispatcherPane, SplitRight, err
	}
	return sibling, SplitBelow, nil
}

// PaneAlive reports whether a pane dispatch's tmux pane still exists — the
// pane-mode analogue of the headless path's kill(pid,0) liveness probe, and the
// signal that separates `running` from `orphaned` (see DerivePaneState).
//
// It delegates to pane.ValidatePane, which probes with a single targeted
// `display-message -t <pane> -p '#{pane_id}'` and compares the result to the
// argument. Any failure — pane gone, tmux server gone, socket error — reads as
// NOT alive, which is the correct conservative reading here: with no result
// file, an unobservable worker is orphaned, and `status` must report that rather
// than erroring out.
func PaneAlive(paneID, server string) bool {
	if paneID == "" {
		return false
	}
	return pane.ValidatePane(paneID, server) == nil
}

// KillPane kills a pane dispatch's tmux pane — the pane-mode analogue of
// KillGroup. Killing the pane takes the interactive worker down with it (the
// pane's process group is tmux's to reap), so there is no separate process
// signalling.
//
// It is IDEMPOTENT in the same shape as KillGroup: an already-gone pane is a
// benign no-op rather than an error, detected via pane.IsPaneMissing on tmux's
// stderr (and via the typed *pane.PaneNotFoundError path callers get from
// PaneAlive). No marker file is written — with no result file present the
// dispatch derives `orphaned` from the dead pane, per DerivePaneState.
func KillPane(paneID, server string) error {
	if paneID == "" {
		return fmt.Errorf("no pane recorded for this dispatch")
	}
	_, stderr, err := pane.RunCmd("tmux", pane.WithServer(server, "kill-pane", "-t", paneID)...)
	if err == nil || pane.IsPaneMissing(stderr) {
		return nil
	}
	return pane.StderrError(fmt.Errorf("tmux kill-pane: %w", err), stderr)
}
