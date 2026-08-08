package dispatch

// This file holds the DONE-WORKER REAPING decision behind `fab dispatch reap` — the
// pane-hygiene verb that reclaims the column space a finished pane worker would
// otherwise hold for the rest of a run.
//
// A pane-mode worker never exits on completion: it writes {stage}-result.yaml and
// sits at its interactive prompt (deliberately, so the user can still steer it after
// it finishes). Across a multi-stage pipeline that leaves a stack of dead panes in
// the carved worker column, shrinking the panes the user actually watches with every
// completed stage. Reap kills such a pane at the one deterministic moment that
// already exists — right after the orchestrator reads a `done` result.
//
// REAP IS NOT KILL. `kill` is the RECOVERY verb and is valid in any state; reap is
// HYGIENE and fires only on `done`, only for a pane record, and only when policy
// allows it. Keeping them separate is why a config knob can gate reap without ever
// modulating a recovery command.
//
// It also cleans NO state: the record, result, prompt, and log files all survive, so
// a reaped dispatch still derives `done` forever (DerivePaneState gives result
// presence precedence over pane liveness) and the no-automatic-GC posture —
// archive-time deletion plus explicit `fab dispatch clean`, the only two cleanup
// moments — is untouched.

// ReapVerdict is the outcome of the reap guard: either "reap the pane" or the single
// reason the reap was skipped. Skips are NOT errors — every one of them is a normal,
// exit-0 no-op — so the verdict doubles as the reason the command reports, and the
// command layer never has to recompose a reason from raw booleans.
type ReapVerdict string

const (
	// ReapPane: all three conditions hold — a pane record, derived state `done`, and
	// the policy knob enabled. The only verdict that kills anything.
	ReapPane ReapVerdict = "reap"
	// ReapSkipHeadless: the record is headless. There is no pane to reclaim — the
	// detached worker process already exited — so reap has nothing visual to do.
	ReapSkipHeadless ReapVerdict = "headless"
	// ReapSkipNotDone: a pane record whose derived state is not `done` (running or
	// orphaned). This is the guard that makes reap ≠ kill: a live worker, or one that
	// died without producing a result, must never be terminated by a hygiene verb.
	ReapSkipNotDone ReapVerdict = "not-done"
	// ReapSkipDisabled: `dispatch.reap_done` resolved false — the user opted to keep
	// done-worker panes and their scrollback.
	ReapSkipDisabled ReapVerdict = "disabled"
)

// DecideReap is the whole reap guard, as a PURE function over its three inputs — no
// I/O, no config read, no tmux probe — so the mode × state × knob matrix is
// exhaustively table-testable, matching SelectMode / SelectPaneShape /
// DerivePaneState in this package.
//
// The order of the checks is deliberate and only the REPORTED reason depends on it:
// mode, then state, then policy. Putting the state check ahead of the knob keeps the
// "reap never terminates a live or failed dispatch" invariant independent of
// configuration — no value of dispatch.reap_done can reach a non-`done` dispatch.
//
// state is the DERIVED state (DerivePaneState for a pane record). It is passed in
// rather than derived here because deriving it needs I/O; a headless record's
// five-state value is accepted and simply never consulted, since the mode check
// short-circuits first.
func DecideReap(isPane bool, state State, reapDone bool) ReapVerdict {
	switch {
	case !isPane:
		return ReapSkipHeadless
	case state != StateDone:
		return ReapSkipNotDone
	case !reapDone:
		return ReapSkipDisabled
	default:
		return ReapPane
	}
}
