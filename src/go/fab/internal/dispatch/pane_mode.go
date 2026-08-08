package dispatch

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// This file holds the tmux side of PANE-MODE dispatch (`fab dispatch start
// --pane`): server reachability, worker-pane creation (split or new window),
// worker-COLUMN placement (which pane a new worker splits, in which direction, and
// how wide the column is carved), pane liveness, and pane kill.
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

// AutoReason names the chosen automatic rung and any prerequisites that forced a
// descent. It is distinct from Mode so successful launch output can explain the
// choice without re-deriving policy.
type AutoReason string

const (
	// ReasonExplicit means an explicit flag selected the mode. Explicit output
	// carries no suffix and missing prerequisites remain hard errors.
	ReasonExplicit AutoReason = ""
)

// ErrNoMode means no rung at or below the configured preference is possible.
var ErrNoMode = errors.New("no dispatch mode is available")

// TmuxAvailability is the caller-observed pane environment. Resolve-agent can
// distinguish only presence/absence; dispatch start upgrades a failed probe to
// TmuxUnreachable so output names the real descent cause.
type TmuxAvailability string

const (
	TmuxAvailable   TmuxAvailability = "available"
	TmuxAbsent      TmuxAvailability = "no tmux"
	TmuxUnreachable TmuxAvailability = "tmux unreachable"
)

// SelectMode resolves the explicit-signal-first launch ladder. When no explicit
// signal is present, preference is a ceiling and selection descends only through
// pane → native → headless. A missing prerequisite skips its rung; no rung yields
// ErrNoMode.
//
// It is a PURE function — no environment reads, no tmux probe, no I/O — so the
// whole ladder is table-testable, matching DeriveState/DerivePaneState's shape in
// this package. The caller supplies tmuxEnv (os.Getenv("TMUX")) and the four
// explicit signals, each read as "was the flag SUPPLIED" (cobra's
// Flags().Changed) rather than by value, so an explicit `--timeout 0` or
// `--server ""` still counts as a signal.
//
// Explicit signals, in order:
//
//  1. paneFlag     → pane      (explicit)
//  2. headlessFlag → headless  (explicit)
//  3. timeoutSet   → headless  (explicit: --timeout is enforced by the headless
//     wrapper, so it can only mean headless)
//  4. serverSet    → pane      (explicit: --server exists solely to target a
//     pane's socket, so naming one means pane)
//  5. otherwise    → configured preference + capability descent
//
// paneFlag and headlessFlag are mutually exclusive at the cobra layer, so rung 1
// preceding rung 2 is a tie-break that can never be reached in practice.
//
// tmux is caller-supplied availability: resolve-agent uses $TMUX
// presence, while dispatch start replaces it with a real reachability result
// before launching. That keeps environment/probe I/O outside this pure function.
func SelectMode(paneFlag, headlessFlag, timeoutSet, serverSet bool, preference string,
	native, sessionCommand, dispatchCommand bool, tmux TmuxAvailability,
) (Mode, AutoReason, error) {
	switch {
	case paneFlag:
		return ModePane, ReasonExplicit, nil
	case headlessFlag:
		return ModeHeadless, ReasonExplicit, nil
	case timeoutSet:
		return ModeHeadless, ReasonExplicit, nil
	case serverSet:
		return ModePane, ReasonExplicit, nil
	}

	var skipped []string
	if preference == string(ModePane) {
		switch {
		case tmux != TmuxAvailable:
			skipped = append(skipped, "pane unavailable: "+string(tmux))
		case !sessionCommand:
			skipped = append(skipped, "pane unavailable: no session_command")
		default:
			return ModePane, preferredReason(ModePane), nil
		}
	}

	if preference == string(ModePane) || preference == string(ModeNative) {
		if native {
			return ModeNative, selectionReason(ModeNative, skipped), nil
		}
		skipped = append(skipped, "native unavailable")
	}

	if dispatchCommand {
		return ModeHeadless, selectionReason(ModeHeadless, skipped), nil
	}
	return "", "", ErrNoMode
}

func preferredReason(mode Mode) AutoReason {
	return AutoReason(fmt.Sprintf("mode: %s (preferred)", mode))
}

func selectionReason(mode Mode, skipped []string) AutoReason {
	if len(skipped) == 0 {
		return preferredReason(mode)
	}
	return AutoReason(fmt.Sprintf("mode: %s (descended: %s)", mode, strings.Join(skipped, "; ")))
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
// What the caller does with a non-nil return depends on how pane mode was
// selected: an explicit pane selection propagates this error, while an automatic
// selection re-runs SelectMode with TmuxUnreachable and descends to native or
// headless. Either outcome occurs before launch or persistence.
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

// OpenSplitPane executes an already-resolved SplitPlacement to create the worker's
// pane, titles it, and returns the new pane's ID. Unlike OpenWindow the worker lands
// in the SAME tmux window as the split target — the two-tier hierarchy's inner tier.
//
// The placement decision (which pane, which direction, how wide) is SplitTarget's;
// this function only executes it, so the decision stays inspectable and pure while
// this stays a thin tmux wrapper.
//
// SIZE DEGRADATION: a size is only ever requested for the column-carving split, and
// tmux may refuse it — `-l <n>%` needs tmux ≥ 3.1, and a window can be too narrow
// for the requested percentage. Either way the split is retried UNSIZED rather than
// failing: placement is cosmetic, so the worker must still launch. The refusal is
// returned as a warning instead of an error. Retrying beats probing `tmux -V` first:
// it costs no extra round-trip on the happy path, needs no version-string parsing
// (`3.1a`, `next-3.4`, distro forks), and covers the too-narrow case a version
// check would miss.
//
// TITLE: a split pane has no window name of its own — its window belongs to the
// dispatcher — so the dispatch's identity string rides the PANE TITLE instead
// (`select-pane -T`). Titles are still set for IDENTIFICATION only; placement no
// longer reads them (harnesses running inside the worker rewrite them — see
// SiblingDispatchPane). A title-set failure is NON-FATAL: the worker is already
// running and its pane ID (the real identity, which every later probe keys on) is
// already in hand, so refusing the dispatch over a cosmetic label would be a
// strictly worse outcome.
//
// Both non-fatal outcomes come back as warnings for the caller to surface; only a
// split that cannot be placed at all is an error.
//
// cmd is passed as split-window's shell-command argument, so — exactly as in
// OpenWindow — it is the WHOLE left-hand side including its own shell expansions,
// which expand at invocation inside the new pane.
func OpenSplitPane(server string, place SplitPlacement, title, dir, cmd string) (paneID string, warnings []error, err error) {
	paneID, err = runPaneCreator(server, "split-window", title, splitArgs(place, dir, cmd)...)
	if err != nil && place.SizePercent > 0 {
		// The parenthetical names BOTH refusal causes: a pre-3.1 tmux has no
		// percentage size at all, and any tmux refuses one a too-narrow window
		// cannot satisfy. Naming only the version would send a user on a
		// tmux-upgrade hunt for a window-geometry problem.
		warnings = append(warnings, fmt.Errorf(
			"tmux rejected the sized split (%s %d%%): %w; retrying unsized (a percentage size needs tmux 3.1+ and a window wide enough for it)",
			sizeFlag, place.SizePercent, err))
		unsized := place
		unsized.SizePercent = 0
		paneID, err = runPaneCreator(server, "split-window", title, splitArgs(unsized, dir, cmd)...)
	}
	if err != nil {
		return "", warnings, err
	}
	if _, stderr, terr := pane.RunCmd("tmux", pane.WithServer(server,
		"select-pane", "-t", paneID, "-T", title)...); terr != nil {
		warnings = append(warnings, pane.StderrError(
			fmt.Errorf("could not set pane title %q: %w", title, terr), stderr))
	}
	return paneID, warnings, nil
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

// Split argv flags for OpenSplitPane, named so the placement rules are not bare
// tmux flags at the call site. They are PACKAGE-SCOPE: the placement is decided
// (SplitTarget), rendered (splitArgs), and described (SplitPlacement.Describe) here,
// so no caller outside this package ever handles a raw tmux flag.
const (
	// splitRight carves the worker column out of the dispatcher's pane.
	splitRight = "-h"
	// splitBelow stacks a worker under the previous one, inside that column.
	splitBelow = "-v"
	// sizeFlag sizes a split (`-l <n>%`, tmux ≥ 3.1). Only the column-carving
	// splitRight is ever sized; see SplitPlacement.
	sizeFlag = "-l"
)

// SplitPlacement is a resolved worker-pane placement: WHICH pane to split, in WHICH
// direction, and — for the column-carving split only — how wide to carve the column.
//
// The three travel together because they are one decision (SplitTarget's), and
// bundling them is what keeps the "size the carving split, never a stacking split"
// rule in exactly one place: the degraded branch is the same splitRight decision, so
// it inherits the size instead of needing its own copy of the rule.
type SplitPlacement struct {
	// Target is the pane to split — an existing worker pane when stacking, the
	// dispatcher's own pane when carving the column.
	Target string
	// Direction is splitRight (carve) or splitBelow (stack).
	Direction string
	// SizePercent is the new pane's width as a percent of the window, rendered as
	// `-l <n>%`. Zero means UNSIZED (tmux even-splits), which is always the case for
	// a splitBelow: sizing a stacking split would fight the user's own resizes
	// inside the column, and the left/right separator must never be re-touched.
	SizePercent int
}

// Describe renders the placement in the stacked-column vocabulary the rule is
// documented in ("carving a new worker column" / "stacking under") — the human half
// of the cobra layer's degraded-probe warning.
//
// It lives here, as a method, so the bare tmux `-h`/`-v` flag the placement carries
// never leaves this package: the direction constants stay package-scope, and the only
// cross-package reader of Direction is this vocabulary rather than the flag.
func (p SplitPlacement) Describe() string {
	if p.Direction == splitRight {
		return fmt.Sprintf("carving a new worker column off pane %s", p.Target)
	}
	return fmt.Sprintf("stacking the worker under pane %s", p.Target)
}

// SiblingDispatchPane returns the LAST live dispatch worker pane in targetPane's
// window, or "" when there is none.
//
// Detection keys on DISPATCH RECORDS, not pane titles. It intersects the pane IDs
// recorded across the checkout's dispatch records for THIS server (recordedPanes —
// pane IDs are per-socket, so records from another socket are filtered out before
// the intersection) with `tmux list-panes -t <targetPane> -F '#{pane_id}'` —
// window-scoped, since `-t` on a pane resolves to that pane's window — and keeps the
// last row present in both.
//
// The record is the right identity source because a pane ID is server-global and
// stable for the pane's lifetime, which is already why status/kill/capture key on
// it. A pane TITLE is not: a harness running inside the worker pane (Claude Code and
// friends) rewrites it via terminal escapes within seconds of spawn, so a
// title-keyed probe finds nothing and every subsequent worker takes the
// no-sibling branch — re-splitting the dispatcher and carving yet another
// full-height column until the dispatching agent is a sliver. Titles are still SET
// at spawn, for identification only.
//
// The intersection is also the whole filter: a dead pane and a pane in another
// window are both absent from this window's live list-panes output, so no separate
// liveness probe or window lookup is needed — and "last" stays tmux's list-panes
// order, i.e. the pane-index order the column was built in, so a further split lands
// under the newest worker.
//
// One consequence is deliberate: when the DISPATCHER is itself a pane worker (a
// stage worker dispatching a stage of its own), its pane is in the record set, so it
// can be its own "sibling" and the new worker stacks under it rather than carving a
// second column. That is the wanted outcome — the dispatcher already lives in a
// worker column, and stacking keeps that column — and it is self-limiting: the next
// dispatch finds the child below it and stacks under that instead.
//
// BOTH halves of the probe can fail, and NEITHER failure is fatal — placement is
// cosmetic, so a failing probe must never fail a dispatch that would otherwise launch
// fine. The two degrade differently only because they can:
//
//   - A failing `list-panes` leaves no window to intersect against, so it returns
//     ("", err) and the caller carves a column off the dispatcher's own pane.
//   - A failing RECORD READ (an unreadable dir, a corrupt {stage}.yaml) still leaves
//     the panes that WERE read, so the partial intersection is returned ALONGSIDE the
//     error. A missing record can only fail to find a sibling, never invent one, so
//     the answer is either the same one a clean read would give or the first-worker
//     carve — and the caller warns either way rather than silently placing blind.
func SiblingDispatchPane(server, targetPane, repoRoot string) (string, error) {
	out, stderr, err := pane.RunCmd("tmux", pane.WithServer(server,
		"list-panes", "-t", targetPane, "-F", "#{pane_id}")...)
	if err != nil {
		return "", pane.StderrError(fmt.Errorf("tmux list-panes: %w", err), stderr)
	}
	recorded, recErr := recordedPanes(repoRoot, server)
	return lastRecordedPane(out, recorded), recErr
}

// lastRecordedPane is the pure half of SiblingDispatchPane: given list-panes output
// (one pane id per line, in pane-index order) and the set of pane ids the dispatch
// records claim, it returns the LAST id present in both, or "" when none is.
// Extracted so the row grammar and the intersection are unit-testable without a tmux
// server or a record tree.
func lastRecordedPane(out string, recorded map[string]bool) string {
	found := ""
	for _, line := range strings.Split(out, "\n") {
		id := strings.TrimSpace(line)
		if id == "" {
			continue
		}
		if recorded[id] {
			found = id
		}
	}
	return found
}

// SplitTarget resolves a new worker's SplitPlacement, applying the stacked-column
// rule against the live window:
//
//	a live recorded worker sibling → split THAT pane below (splitBelow), stacking
//	                                 the column; unsized
//	none (or the probe failed)     → split the dispatcher's own pane to the right
//	                                 (splitRight), CARVING the column at
//	                                 columnWidth percent
//
// This is the column INVARIANT: the vertical left/right separator is created exactly
// once, by the carving split, and never touched again — fab issues no select-layout,
// never rearranges user-made panes, and never fights a manual resize. It is a
// creation-time rule, not an enforcement loop: an already-mangled window is left
// alone until its panes die.
//
// columnWidth is the resolved dispatch.column_width (percent). It applies to the
// carving split ONLY — including the DEGRADED carve below, since a fallback column
// that halved the dispatcher would reintroduce exactly the squeeze the width exists
// to prevent.
//
// A probe error is returned ALONGSIDE a usable decision rather than replacing it, so
// the caller can warn without the dispatch failing. Which decision depends on how much
// the probe still managed to answer: a failing list-panes (or an unreadable record
// tree) leaves no sibling and lands on the degraded carve, while a single unreadable
// record leaves the others intact and may still stack. Either way the caller warns.
func SplitTarget(server, dispatcherPane, repoRoot string, columnWidth int) (SplitPlacement, error) {
	sibling, err := SiblingDispatchPane(server, dispatcherPane, repoRoot)
	return splitPlacement(sibling, dispatcherPane, columnWidth), err
}

// splitPlacement is the pure decision half of SplitTarget: it maps the probe's
// answer to a placement. An empty sibling (none found, or the probe failed) is the
// first-worker/degraded case, which carves a sized column off the dispatcher.
func splitPlacement(sibling, dispatcherPane string, columnWidth int) SplitPlacement {
	if sibling == "" {
		return SplitPlacement{Target: dispatcherPane, Direction: splitRight, SizePercent: columnWidth}
	}
	return SplitPlacement{Target: sibling, Direction: splitBelow}
}

// splitArgs composes the `tmux split-window` argv for a placement (without the
// `-L <server>` prefix, which pane.WithServer adds), printing the new pane's id so
// no follow-up lookup can race a fast-exiting worker.
//
// The size argument is emitted only when the placement carries one — which, by
// SplitPlacement's contract, means only for a column-carving split. It is rendered
// as a PERCENTAGE (`-l 35%`) so the column scales with the window rather than
// pinning a cell count that would be wrong on the next resize.
func splitArgs(place SplitPlacement, dir, cmd string) []string {
	args := []string{"split-window", place.Direction, "-t", place.Target}
	if place.SizePercent > 0 {
		args = append(args, sizeFlag, strconv.Itoa(place.SizePercent)+"%")
	}
	return append(args, "-P", "-F", "#{pane_id}", "-c", dir, cmd)
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
