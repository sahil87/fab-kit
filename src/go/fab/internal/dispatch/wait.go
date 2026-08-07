package dispatch

import (
	"context"
	"time"
)

// TickInterval is the wait loop's in-process re-derivation cadence.
//
// `fab dispatch wait` blocks by re-deriving the dispatch's state on this tick —
// it is NOT a filesystem watcher. A watcher on {stage}-result.yaml could see a
// worker finish, but it cannot see a worker DIE: the `orphaned` state is derived
// from pid/pane liveness, for which a periodic probe is required regardless. So a
// watcher would add a dependency without removing the tick.
//
// The cost being eliminated by `wait` is INFERENCE TURNS (an orchestrator waking
// every 30s to run `fab dispatch status`), not filesystem syscalls: a 2s stat +
// liveness probe inside one Go process is free. 2s is chosen so an orphaned
// worker surfaces in ~2s instead of at the next 30s poll, which is what lets the
// recovery policy's single automatic restart fire almost immediately.
//
// Deliberately a package constant with no config field and no flag: the value is
// an implementation detail of the blocking contract, not a tuning surface (the
// user-visible knob is `--timeout`, which bounds the block).
const TickInterval = 2 * time.Second

// Observer derives the CURRENT state of a dispatch, exactly as
// `fab dispatch status` would derive it at that instant.
//
// It is a function value rather than a concrete type so the wait loop stays a
// pure control structure with no I/O of its own — the caller supplies the same
// loader + DeriveState/DerivePaneState composition `status` uses (see
// cmd/fab/dispatch_status.go's observeDispatch), and tests supply a scripted
// sequence. That is what makes `wait` and `status` structurally incapable of
// disagreeing about state.
type Observer func() (State, error)

// Wait blocks until observe reports a state other than StateRunning, then
// returns that state.
//
// Semantics (the `fab dispatch wait` contract):
//
//   - The FIRST observation happens before any sleep, so an already-terminal
//     dispatch returns immediately at zero tick cost. This is what makes the verb
//     idempotent and safe to re-arm after a restart or re-run after an
//     interruption.
//   - timeout > 0 bounds the block. On expiry Wait observes ONCE MORE and returns
//     that still-current state with a NIL error — it does NOT return the state
//     cached at the last tick, which would miss a transition landing in the
//     sub-tick window before the bound and let `wait` print `running` while
//     `status` prints `done` at the same instant. So a bounded wait returns
//     StateRunning only when the dispatch really is still running AT the bound.
//     The state string is the sole timeout discriminator; a distinct error (or
//     exit code) would add a second channel for information the string already
//     carries, and 124 in this family already means "the WORKER was killed by its
//     own --timeout wrapper".
//   - timeout <= 0 waits indefinitely. An explicit `--timeout 0` therefore reads
//     as absent, matching "absent → wait indefinitely": zero as an instant
//     timeout has no consumer and would silently no-op the skill wiring if a
//     variable expanded empty.
//   - An observe error aborts the wait and is returned as-is — a genuine read
//     failure must surface rather than be re-polled forever.
//   - ctx cancellation (e.g. SIGINT at the CLI) returns the last observed state
//     with ctx.Err(), so a cancelled wait is distinguishable from a timeout.
//
// tick is a parameter rather than a direct read of TickInterval so tests can
// drive the loop in milliseconds — the same injectability every other derivation
// in this package has. Callers pass TickInterval.
func Wait(ctx context.Context, observe Observer, tick, timeout time.Duration) (State, error) {
	state, err := observe()
	if err != nil {
		return state, err
	}
	if state != StateRunning {
		return state, nil
	}

	var deadline <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		deadline = timer.C
	}

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		case <-deadline:
			// Bound reached: observe ONCE MORE before reporting. The cached `state`
			// is only as fresh as the last tick, so returning it would let a
			// transition landing in the sub-tick window before the bound go
			// unreported — `wait` would print `running` while `status` prints `done`
			// at the same instant, breaking the "wait and status cannot disagree"
			// contract. The final observation also settles the coincident-readiness
			// case, where select may pick this arm over an equally ready tick.
			//
			// A still-`running` result IS the timeout return: the caller treats it
			// as its peek-on-suspicion moment.
			return observe()
		case <-ticker.C:
			state, err = observe()
			if err != nil {
				return state, err
			}
			if state != StateRunning {
				return state, nil
			}
		}
	}
}
