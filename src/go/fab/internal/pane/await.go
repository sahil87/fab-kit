package pane

import (
	"context"
	"time"
)

// AwaitTick is the await loop's in-process re-derivation cadence.
//
// `fab pane await` blocks by re-checking its signals on this tick — it is NOT
// a filesystem watcher. A watcher on the --file path could see a contract file
// appear, but it cannot see a pane DIE or an agent flip to idle: both are
// tmux-state reads, for which a periodic probe is required regardless. So a
// watcher would add a dependency without removing the tick (the dispatch
// package's TickInterval reasoning, applied to the record-free layer).
//
// Deliberately a package constant with no config field: the value is an
// implementation detail of the blocking contract, not a tuning surface (the
// user-visible knob is `--timeout`, which bounds the block).
const AwaitTick = 2 * time.Second

// AwaitReport is the one-word report `fab pane await` prints. The values are
// the report contract the caller branches on, so they are named constants,
// never inline literals (the Readiness and internal/dispatch State precedent).
type AwaitReport string

const (
	// AwaitIdle: the pane's @rk_agent_state resolved to `idle`.
	AwaitIdle AwaitReport = "idle"
	// AwaitFile: the --file path exists.
	AwaitFile AwaitReport = "file"
	// AwaitRunning: --timeout expired with no signal fired — the timeout bounds
	// the observer, never the pane (the `fab dispatch wait` precedent: exit 0).
	AwaitRunning AwaitReport = "running"
	// AwaitGone: the pane died mid-wait — the wait cannot complete, and the
	// caller must branch on cause (exit 2, the pane-family pane-missing code).
	AwaitGone AwaitReport = "gone"
)

// AwaitObserver derives which await signal, if any, has fired RIGHT NOW — the
// same signals `fab pane await` would check at that instant (pane liveness,
// --file existence, the @rk_agent_state read). It returns "" when nothing has
// fired yet.
//
// It is a function value rather than a concrete type so the loop stays a pure
// control structure with no I/O of its own — the caller supplies the tmux/stat
// composition, and tests supply a scripted sequence (the dispatch.Observer
// precedent).
type AwaitObserver func() (AwaitReport, error)

// Await blocks until observe reports a fired signal, then returns it.
//
// Semantics (mirroring dispatch.Wait, so the family's two blocking waits share
// one contract):
//
//   - The FIRST observation happens before any sleep, so an already-fired
//     signal (an idle agent, an existing file) returns immediately at zero
//     tick cost. This is what makes the verb idempotent and safe to re-run.
//   - timeout > 0 bounds the block. On expiry Await observes ONCE MORE and
//     returns that still-current report — AwaitRunning when still nothing has
//     fired — with a NIL error. Returning the report cached at the last tick
//     would miss a signal landing in the sub-tick window before the bound. So
//     a bounded await returns AwaitRunning only when nothing really has fired
//     AT the bound. The report string is the sole timeout discriminator; a
//     distinct error (or exit code) would add a second channel for information
//     the string already carries.
//   - timeout <= 0 waits indefinitely. An explicit `--timeout 0` therefore
//     reads as absent, matching "absent → wait indefinitely".
//   - An observe error aborts the wait and is returned as-is — a genuine read
//     failure must surface rather than be re-polled forever.
//   - ctx cancellation (e.g. SIGINT at the CLI) returns the last observed
//     report with ctx.Err(), so a cancelled await is distinguishable from a
//     timeout.
//
// tick is a parameter rather than a direct read of AwaitTick so tests can
// drive the loop in milliseconds — the same injectability every other
// derivation in this package has. Callers pass AwaitTick.
func Await(ctx context.Context, observe AwaitObserver, tick, timeout time.Duration) (AwaitReport, error) {
	report, err := observe()
	if err != nil {
		return report, err
	}
	if report != "" {
		return report, nil
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
			return report, ctx.Err()
		case <-deadline:
			// Bound reached: observe ONCE MORE before reporting. The cached
			// report is only as fresh as the last tick, so returning it would
			// let a signal landing in the sub-tick window before the bound go
			// unreported. A still-"" result IS the timeout return, reported as
			// AwaitRunning: the observer was bounded, not the pane.
			report, err := observe()
			if err != nil {
				return report, err
			}
			if report == "" {
				return AwaitRunning, nil
			}
			return report, nil
		case <-ticker.C:
			report, err = observe()
			if err != nil {
				return report, err
			}
			if report != "" {
				return report, nil
			}
		}
	}
}
