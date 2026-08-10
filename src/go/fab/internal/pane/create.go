package pane

import (
	"fmt"
	"strconv"
	"strings"
)

// This file holds the tmux pane-CREATION and lifecycle mechanics of the shared
// pane layer: server reachability, the pane creators (split or new window),
// the placement EXECUTION half (SplitPlacement rendering), pane liveness, and
// pane kill — plus PointerPrompt, the shared pointer-line composer the verified
// delivery choreography types.
//
// The placement DECISION half — which pane a new worker splits, in which
// direction, and how wide the column is carved (SplitTarget, SelectPaneShape)
// — stays in internal/dispatch: it reads dispatch records and config, which is
// pipeline policy. What lives here is everything addressed by a pane id alone,
// so the provider-generic `fab pane` primitives and the dispatch bindings share
// exactly one copy of the tmux mechanics.
//
// It lives in the platform-INDEPENDENT core rather than a build-tagged split
// for the same reason internal/dispatch's WrapperArgv does: these are plain
// tmux SUBPROCESS calls with no syscall dependency, so the code compiles
// everywhere. (Pane work is still unusable where tmux is absent — but that
// surfaces as ServerReachable's actionable error, not a compile-time platform
// split.)
//
// Every tmux invocation goes through this package's shared helpers — RunCmd
// (stdout/stderr/exec-error capture), WithServer (the `-L <name>` argv prefix),
// and StderrError (surface the child's diagnostic instead of a bare exit
// status) — so there is exactly one tmux argv builder and one stderr-enrichment
// convention in the binary.

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
// selection re-runs the mode ladder with tmux-unreachable and descends to native
// or headless. Either outcome occurs before launch or persistence.
func ServerReachable(server string) error {
	_, stderr, err := RunCmd("tmux", WithServer(server, "list-sessions")...)
	if err == nil {
		return nil
	}
	target := "the default tmux socket"
	if server != "" {
		target = fmt.Sprintf("tmux socket %q", server)
	}
	// The message names "pane mode" rather than `--pane`: pane can also be
	// selected explicitly by `--server` alone (the dispatch mode ladder's rung
	// 4), and a caller who passed only `--server` would be confused by guidance
	// quoting a flag they never supplied. The remedy clause still names both
	// flags to drop.
	return StderrError(fmt.Errorf(
		"pane mode requires a reachable tmux server, but %s is unreachable; start tmux (or pass --server <name>), or pass --headless (drop --pane/--server) to dispatch headless",
		target), stderr)
}

// OpenWindow creates a tmux window named name with cwd dir running cmd, and
// returns the new window's PANE ID.
//
// This is the FALLBACK pane shape (see internal/dispatch's SelectPaneShape): it
// is what a dispatcher with no pane of its own to split — a headless
// orchestrator passing `--pane`, or a caller naming another socket with
// `--server` — gets. A dispatcher that IS a tmux pane on the target server gets
// OpenSplitPane instead.
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
// `$(basename "$(pwd)")` in the built-in claude interactive_command), which expand
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
// internal/dispatch's SiblingDispatchPane). A title-set failure is NON-FATAL: the
// worker is already running and its pane ID (the real identity, which every later
// probe keys on) is already in hand, so refusing the dispatch over a cosmetic label
// would be a strictly worse outcome.
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
			SizeFlag, place.SizePercent, err))
		unsized := place
		unsized.SizePercent = 0
		paneID, err = runPaneCreator(server, "split-window", title, splitArgs(unsized, dir, cmd)...)
	}
	if err != nil {
		return "", warnings, err
	}
	if _, stderr, terr := RunCmd("tmux", WithServer(server,
		"select-pane", "-t", paneID, "-T", title)...); terr != nil {
		warnings = append(warnings, StderrError(
			fmt.Errorf("could not set pane title %q: %w", title, terr), stderr))
	}
	return paneID, warnings, nil
}

// runPaneCreator executes a tmux command that PRINTS the created pane's id
// (`-P -F '#{pane_id}'`) and returns that id, applying one convention to both
// creators: the child's own stderr is surfaced via StderrError instead of a
// bare exit status, and an empty id is an error rather than a silently
// unidentifiable dispatch. verb/label only shape the diagnostics.
func runPaneCreator(server, verb, label string, args ...string) (string, error) {
	out, stderr, err := RunCmd("tmux", WithServer(server, args...)...)
	if err != nil {
		return "", StderrError(fmt.Errorf("tmux %s: %w", verb, err), stderr)
	}
	paneID := strings.TrimSpace(out)
	if paneID == "" {
		return "", fmt.Errorf("tmux %s reported no pane id for %q", verb, label)
	}
	return paneID, nil
}

// OpenPlainPane spawns cmd in a plain tmux pane with cwd dir and returns the
// new pane's ID — the provider-generic primitive under `fab pane open`, with
// NO size, NO title, and NO placement logic (the worker-column carving is
// dispatch policy and stays behind OpenSplitPane).
//
// When split is true the pane is a plain split of the caller's own window
// (split-window with no -t: tmux resolves the current pane from the invoking
// client, which is exactly the $TMUX_PANE case the cmd layer gates on);
// otherwise it is a new UNNAMED window. Both print the pane id, so no
// follow-up lookup can race a fast-exiting command.
//
// cmd is passed as the shell-command argument, so — exactly as in OpenWindow —
// it is the WHOLE left-hand side including its own shell expansions, which
// expand at invocation inside the new pane.
func OpenPlainPane(server string, split bool, dir, cmd string) (paneID string, err error) {
	verb := "new-window"
	if split {
		verb = "split-window"
	}
	return runPaneCreator(server, verb, dir, plainPaneArgs(split, dir, cmd)...)
}

// plainPaneArgs composes the spawn argv for OpenPlainPane (without the
// `-L <server>` prefix, which WithServer adds). Extracted so both shapes are
// unit-testable without a tmux server.
func plainPaneArgs(split bool, dir, cmd string) []string {
	verb := "new-window"
	if split {
		verb = "split-window"
	}
	return []string{verb, "-P", "-F", "#{pane_id}", "-c", dir, cmd}
}

// Split argv flags for OpenSplitPane, named so the placement rules are not bare
// tmux flags at the call site. They are exported for exactly ONE cross-package
// reader: internal/dispatch's placement DECISION (SplitTarget/splitPlacement)
// constructs SplitPlacement values, while the rendering (splitArgs) and the
// description (SplitPlacement.Describe) stay here — so no caller ever handles
// a raw tmux flag as a flag.
const (
	// SplitRight carves the worker column out of the dispatcher's pane.
	SplitRight = "-h"
	// SplitBelow stacks a worker under the previous one, inside that column.
	SplitBelow = "-v"
	// SizeFlag sizes a split (`-l <n>%`, tmux ≥ 3.1). Only the column-carving
	// SplitRight is ever sized; see SplitPlacement.
	SizeFlag = "-l"
)

// SplitPlacement is a resolved worker-pane placement: WHICH pane to split, in WHICH
// direction, and — for the column-carving split only — how wide to carve the column.
//
// The three travel together because they are one decision (SplitTarget's), and
// bundling them is what keeps the "size the carving split, never a stacking split"
// rule in exactly one place: the degraded branch is the same SplitRight decision, so
// it inherits the size instead of needing its own copy of the rule.
type SplitPlacement struct {
	// Target is the pane to split — an existing worker pane when stacking, the
	// dispatcher's own pane when carving the column.
	Target string
	// Direction is SplitRight (carve) or SplitBelow (stack).
	Direction string
	// SizePercent is the new pane's width as a percent of the window, rendered as
	// `-l <n>%`. Zero means UNSIZED (tmux even-splits), which is always the case for
	// a SplitBelow: sizing a stacking split would fight the user's own resizes
	// inside the column, and the left/right separator must never be re-touched.
	SizePercent int
}

// Describe renders the placement in the stacked-column vocabulary the rule is
// documented in ("carving a new worker column" / "stacking under") — the human half
// of the cobra layer's degraded-probe warning.
//
// It lives here, as a method, so the bare tmux `-h`/`-v` flag the placement carries
// is never RENDERED outside this package: dispatch's decision constructs the
// placement, but the only cross-package reader of Direction is this vocabulary —
// splitArgs alone turns the flag into argv.
func (p SplitPlacement) Describe() string {
	if p.Direction == SplitRight {
		return fmt.Sprintf("carving a new worker column off pane %s", p.Target)
	}
	return fmt.Sprintf("stacking the worker under pane %s", p.Target)
}

// splitArgs composes the `tmux split-window` argv for a placement (without the
// `-L <server>` prefix, which WithServer adds), printing the new pane's id so
// no follow-up lookup can race a fast-exiting worker.
//
// The size argument is emitted only when the placement carries one — which, by
// SplitPlacement's contract, means only for a column-carving split. It is rendered
// as a PERCENTAGE (`-l 35%`) so the column scales with the window rather than
// pinning a cell count that would be wrong on the next resize.
func splitArgs(place SplitPlacement, dir, cmd string) []string {
	args := []string{"split-window", place.Direction, "-t", place.Target}
	if place.SizePercent > 0 {
		args = append(args, SizeFlag, strconv.Itoa(place.SizePercent)+"%")
	}
	return append(args, "-P", "-F", "#{pane_id}", "-c", dir, cmd)
}

// PaneAlive reports whether a pane dispatch's tmux pane still exists — the
// pane-mode analogue of the headless path's kill(pid,0) liveness probe, and the
// signal that separates `running` from `orphaned` (see internal/dispatch's
// DerivePaneState).
//
// It delegates to ValidatePane, which probes with a single targeted
// `display-message -t <pane> -p '#{pane_id}'` and compares the result to the
// argument. Any failure — pane gone, tmux server gone, socket error — reads as
// NOT alive, which is the correct conservative reading here: with no result
// file, an unobservable worker is orphaned, and `status` must report that rather
// than erroring out.
func PaneAlive(paneID, server string) bool {
	if paneID == "" {
		return false
	}
	return ValidatePane(paneID, server) == nil
}

// KillPane kills a pane dispatch's tmux pane — the pane-mode analogue of the
// headless path's KillGroup. Killing the pane takes the interactive worker down
// with it (the pane's process group is tmux's to reap), so there is no separate
// process signalling.
//
// It is IDEMPOTENT in the same shape as KillGroup: an already-gone pane is a
// benign no-op rather than an error, detected via IsPaneMissing on tmux's
// stderr (and via the typed *PaneNotFoundError path callers get from
// PaneAlive). No marker file is written — with no result file present the
// dispatch derives `orphaned` from the dead pane, per DerivePaneState.
func KillPane(paneID, server string) error {
	if paneID == "" {
		return fmt.Errorf("no pane recorded for this dispatch")
	}
	_, stderr, err := RunCmd("tmux", WithServer(server, "kill-pane", "-t", paneID)...)
	if err == nil || IsPaneMissing(stderr) {
		return nil
	}
	return StderrError(fmt.Errorf("tmux kill-pane: %w", err), stderr)
}

// PointerPrompt composes the one-line prompt a pane worker is handed by
// `fab pane deliver --prompt-file` and `fab dispatch deliver`.
//
// The FULL stage prompt is never delivered through tmux: a multi-thousand-token
// prompt cannot ride send-keys or argv reliably, so a caller persists it to a
// file (dispatch uses {stage}-prompt.md, the same path and writer the headless
// path uses) and the worker is later typed a pointer to that path instead.
//
// The pointer is DELIVERED POST-SPAWN, not embedded in the launch command. A
// spawn-time positional argument is fire-and-forget and unverifiable — a
// provider CLI that silently drops it leaves a worker sitting at an empty prompt
// while the dispatch reads `running` — and requiring one made pane capability
// hostage to whether a provider's CLI happens to accept a positional prompt.
// Typing it through the gate's send-keys choreography makes delivery a VERIFIED
// step (echo-checked, submit-confirmed, retried once) and leaves
// interactive_command as pure launch grammar. See docs/specs/harness-adapters.md
// § 3.
//
// promptPath should be relative to the pane's cwd: dispatch types it
// REPO-RELATIVE because the worker pane's cwd is the repo root, which keeps the
// pointer readable and portable across worktrees.
func PointerPrompt(promptPath string) string {
	return "Read " + promptPath + " and execute it."
}
