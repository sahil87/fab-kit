package dispatch

import (
	"context"
	"errors"
	"testing"
	"time"
)

// scriptedObserver returns an Observer that walks the given states, repeating the
// LAST one forever once exhausted — so a "stays running" script is written as a
// single-element slice and a transition as {running, running, done}.
func scriptedObserver(states []State, calls *int) Observer {
	return func() (State, error) {
		i := *calls
		*calls++
		if i >= len(states) {
			i = len(states) - 1
		}
		return states[i], nil
	}
}

// TestWaitAlreadyTerminal is the fast path: every non-`running` entry state
// returns immediately, on exactly ONE observation and zero ticks. This is what
// makes `wait` idempotent — safe to re-arm after a restart or re-run after an
// interruption.
func TestWaitAlreadyTerminal(t *testing.T) {
	for _, want := range []State{StateDone, StateFailed, StateFailedNoResult, StateOrphaned} {
		t.Run(string(want), func(t *testing.T) {
			calls := 0
			// A tick and timeout long enough that firing either would hang the test
			// well past its deadline — the fast path must not consult them at all.
			start := time.Now()
			got, err := Wait(context.Background(), scriptedObserver([]State{want}, &calls),
				time.Hour, time.Hour)
			if err != nil {
				t.Fatalf("Wait: %v", err)
			}
			if got != want {
				t.Errorf("state = %q, want %q", got, want)
			}
			if calls != 1 {
				t.Errorf("observations = %d, want 1 (no tick before the first observation)", calls)
			}
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Errorf("took %v — the already-terminal path must not sleep", elapsed)
			}
		})
	}
}

// TestWaitTimeoutReturnsRunning pins the timeout contract: expiry returns the
// still-current state (necessarily `running`) with a NIL error, so the STATE
// STRING is the timeout discriminator and the command exits 0.
func TestWaitTimeoutReturnsRunning(t *testing.T) {
	calls := 0
	start := time.Now()
	got, err := Wait(context.Background(), scriptedObserver([]State{StateRunning}, &calls),
		time.Millisecond, 40*time.Millisecond)
	if err != nil {
		t.Fatalf("timeout must not be an error, got %v", err)
	}
	if got != StateRunning {
		t.Errorf("state = %q, want %q", got, StateRunning)
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Errorf("returned after %v — expected to block until the bound", elapsed)
	}
	if calls < 2 {
		t.Errorf("observations = %d, want the loop to have ticked at least once", calls)
	}
}

// TestWaitTimeoutObservesAtTheBound is the timeout BOUNDARY case: a dispatch that
// turns terminal AFTER the last tick but BEFORE the bound must be reported
// terminal, not `running`.
//
// The tick is set longer than the whole test so NO tick can fire — the only
// observations are the entry one (`running`) and whatever the deadline arm makes.
// Returning the state cached at the last tick therefore yields `running`, which is
// exactly the disagreement R2 forbids: `status` would print `done` at that same
// instant. Only a final observation inside the deadline arm passes.
func TestWaitTimeoutObservesAtTheBound(t *testing.T) {
	flipAt := time.Now().Add(20 * time.Millisecond)
	calls := 0
	// Time-gated rather than call-gated: the state must change between the last
	// tick and the bound, which is a property of the CLOCK, not of the call count.
	observe := func() (State, error) {
		calls++
		if time.Now().Before(flipAt) {
			return StateRunning, nil
		}
		return StateDone, nil
	}

	got, err := Wait(context.Background(), observe, time.Hour, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if got != StateDone {
		t.Errorf("state = %q, want %q — the bound must report a FRESH observation, "+
			"not the state cached at the last tick", got, StateDone)
	}
	if calls != 2 {
		t.Errorf("observations = %d, want 2 (entry + one at the bound)", calls)
	}
}

// TestWaitPropagatesObserveErrorAtTheBound: the final observation at the bound is
// a real read, so its failure surfaces as an error — the same as a tick's would.
// Without this, a timeout could silently report a state derived from a failed read.
func TestWaitPropagatesObserveErrorAtTheBound(t *testing.T) {
	sentinel := errors.New("read exit file: boom")
	calls := 0
	observe := func() (State, error) {
		calls++
		if calls == 1 {
			return StateRunning, nil
		}
		return "", sentinel
	}

	// No tick fires, so the only observation after entry is the deadline arm's.
	_, err := Wait(context.Background(), observe, time.Hour, 20*time.Millisecond)
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want %v", err, sentinel)
	}
}

// TestWaitWakesOnStateTransition is the core event-driven property: a dispatch
// that is `running` at entry and becomes terminal mid-wait wakes the loop and
// returns the NEW state, without reaching the timeout.
func TestWaitWakesOnStateTransition(t *testing.T) {
	tests := []struct {
		name   string
		script []State
		want   State
	}{
		{"running → done", []State{StateRunning, StateRunning, StateDone}, StateDone},
		{"running → failed", []State{StateRunning, StateFailed}, StateFailed},
		{"running → failed (no-result)", []State{StateRunning, StateFailedNoResult}, StateFailedNoResult},
		// The orphan-latency win: a worker that dies without recording an exit is
		// noticed by the liveness probe on the very next tick, not at a 30s poll.
		{"running → orphaned", []State{StateRunning, StateRunning, StateOrphaned}, StateOrphaned},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			got, err := Wait(context.Background(), scriptedObserver(tt.script, &calls),
				time.Millisecond, 10*time.Second)
			if err != nil {
				t.Fatalf("Wait: %v", err)
			}
			if got != tt.want {
				t.Errorf("state = %q, want %q", got, tt.want)
			}
			if calls != len(tt.script) {
				t.Errorf("observations = %d, want %d (the loop must stop at the first non-running state)",
					calls, len(tt.script))
			}
		})
	}
}

// TestWaitUnboundedWaitsPastAnyTimeout: timeout <= 0 means "wait indefinitely",
// so an explicit `--timeout 0` reads as absent rather than as an instant timeout.
func TestWaitUnboundedWaitsPastAnyTimeout(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second} {
		calls := 0
		// Stays running for several ticks, then finishes. With a zero/negative
		// timeout misread as "expire immediately", this would return `running`.
		script := []State{StateRunning, StateRunning, StateRunning, StateDone}
		got, err := Wait(context.Background(), scriptedObserver(script, &calls),
			time.Millisecond, timeout)
		if err != nil {
			t.Fatalf("timeout %v: %v", timeout, err)
		}
		if got != StateDone {
			t.Errorf("timeout %v: state = %q, want %q", timeout, got, StateDone)
		}
	}
}

// TestWaitPropagatesObserveError: a genuine read failure aborts the wait rather
// than being re-polled forever — the CLI then exits non-zero, as `status` would.
func TestWaitPropagatesObserveError(t *testing.T) {
	sentinel := errors.New("read exit file: boom")

	t.Run("on the first observation", func(t *testing.T) {
		_, err := Wait(context.Background(), func() (State, error) { return "", sentinel },
			time.Millisecond, time.Second)
		if !errors.Is(err, sentinel) {
			t.Errorf("err = %v, want %v", err, sentinel)
		}
	})

	t.Run("on a later tick", func(t *testing.T) {
		calls := 0
		_, err := Wait(context.Background(), func() (State, error) {
			calls++
			if calls == 1 {
				return StateRunning, nil
			}
			return "", sentinel
		}, time.Millisecond, time.Second)
		if !errors.Is(err, sentinel) {
			t.Errorf("err = %v, want %v", err, sentinel)
		}
	})
}

// TestWaitHonorsContextCancellation: a cancelled wait (SIGINT at the CLI) returns
// ctx.Err(), so it is distinguishable from a timeout — which returns nil.
func TestWaitHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	got, err := Wait(ctx, scriptedObserver([]State{StateRunning}, &calls),
		time.Millisecond, time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if got != StateRunning {
		t.Errorf("state = %q, want the last observed state %q", got, StateRunning)
	}
}

// TestTickIntervalIsTheDocumentedConstant pins the ~2s cadence the intake fixed:
// short enough that an orphaned worker surfaces in ~2s (vs. the former 30s poll),
// with no config field or flag to tune it.
func TestTickIntervalIsTheDocumentedConstant(t *testing.T) {
	if TickInterval != 2*time.Second {
		t.Errorf("TickInterval = %v, want 2s", TickInterval)
	}
}
