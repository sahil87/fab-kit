package dispatch

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// This file holds the POLICY half of PANE-MODE dispatch (`fab dispatch open`):
// the launch-mode ladder (SelectMode), the pane-shape decision
// (SelectPaneShape), and worker-COLUMN placement (SplitTarget — which pane a
// new worker splits, in which direction, and how wide the column is carved).
//
// The MECHANICS half — server reachability, the pane creators (OpenWindow,
// OpenSplitPane, SplitPlacement rendering), pane liveness, pane kill, the
// readiness gate, and verified delivery — lives in internal/pane, the shared
// pane layer the provider-generic `fab pane` primitives and the dispatch
// bindings both build on. This file's SplitTarget/splitPlacement therefore
// PRODUCE pane.SplitPlacement values (constructed with pane.SplitRight /
// pane.SplitBelow); only pane's splitArgs ever renders one into tmux argv.
//
// It lives in the platform-INDEPENDENT core rather than the build-tagged
// dispatch_posix.go / dispatch_windows.go split for the same reason WrapperArgv
// does: these are pure decisions plus plain tmux SUBPROCESS calls with no
// syscall dependency, so the code compiles everywhere. Only the headless path's
// process launch/signal syscalls are platform-split. (Pane mode is still
// unusable where tmux is absent — but that surfaces as pane.ServerReachable's
// actionable error, not a compile-time platform split.)
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
	native, interactiveCommand, headlessCommand bool, tmux TmuxAvailability,
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
		case !interactiveCommand:
			skipped = append(skipped, "pane unavailable: no interactive_command")
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

	if headlessCommand {
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

// SplitTarget resolves a new worker's pane.SplitPlacement, applying the
// stacked-column rule against the live window:
//
//	a live recorded worker sibling → split THAT pane below (pane.SplitBelow),
//	                                 stacking the column; unsized
//	none (or the probe failed)     → split the dispatcher's own pane to the
//	                                 right (pane.SplitRight), CARVING the column
//	                                 at columnWidth percent
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
func SplitTarget(server, dispatcherPane, repoRoot string, columnWidth int) (pane.SplitPlacement, error) {
	sibling, err := SiblingDispatchPane(server, dispatcherPane, repoRoot)
	return splitPlacement(sibling, dispatcherPane, columnWidth), err
}

// splitPlacement is the pure decision half of SplitTarget: it maps the probe's
// answer to a placement. An empty sibling (none found, or the probe failed) is the
// first-worker/degraded case, which carves a sized column off the dispatcher.
func splitPlacement(sibling, dispatcherPane string, columnWidth int) pane.SplitPlacement {
	if sibling == "" {
		return pane.SplitPlacement{Target: dispatcherPane, Direction: pane.SplitRight, SizePercent: columnWidth}
	}
	return pane.SplitPlacement{Target: sibling, Direction: pane.SplitBelow}
}
