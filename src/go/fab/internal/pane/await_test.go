package pane

import (
	"context"
	"errors"
	"testing"
	"time"
)

// scriptedAwaitObserver returns an AwaitObserver that walks the given reports,
// repeating the LAST one forever once exhausted — so a "nothing fires" script
// is written as a single-element slice and a signal firing on the third
// observation as {"", "", AwaitIdle}. (The scriptedObserver precedent in
// internal/dispatch/wait_test.go.)
func scriptedAwaitObserver(reports []AwaitReport, calls *int) AwaitObserver {
	return func() (AwaitReport, error) {
		i := *calls
		*calls++
		if i >= len(reports) {
			i = len(reports) - 1
		}
		return reports[i], nil
	}
}

// TestAwaitAlreadyFired is the fast path: a signal fired at entry returns
// immediately, on exactly ONE observation and zero ticks. This is what makes
// `await` idempotent — re-running it against an already-idle agent or an
// already-written contract file never sleeps.
func TestAwaitAlreadyFired(t *testing.T) {
	for _, want := range []AwaitReport{AwaitIdle, AwaitFile, AwaitGone} {
		t.Run(string(want), func(t *testing.T) {
			calls := 0
			// A tick and timeout long enough that firing either would hang the
			// test well past its deadline — the fast path must not consult them.
			start := time.Now()
			got, err := Await(context.Background(), scriptedAwaitObserver([]AwaitReport{want}, &calls),
				time.Hour, time.Hour)
			if err != nil {
				t.Fatalf("Await: %v", err)
			}
			if got != want {
				t.Errorf("report = %q, want %q", got, want)
			}
			if calls != 1 {
				t.Errorf("observations = %d, want 1 (no tick before the first observation)", calls)
			}
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Errorf("took %v — the already-fired path must not sleep", elapsed)
			}
		})
	}
}

// TestAwaitSignalOnTick: a signal that fires after the entry observation is
// reported on the tick that sees it.
func TestAwaitSignalOnTick(t *testing.T) {
	calls := 0
	got, err := Await(context.Background(), scriptedAwaitObserver([]AwaitReport{"", "", AwaitFile}, &calls),
		time.Millisecond, time.Hour)
	if err != nil {
		t.Fatalf("Await: %v", err)
	}
	if got != AwaitFile {
		t.Errorf("report = %q, want %q", got, AwaitFile)
	}
	if calls != 3 {
		t.Errorf("observations = %d, want 3 (entry + two ticks)", calls)
	}
}

// TestAwaitTimeoutReturnsRunning pins the timeout contract: expiry returns
// `running` with a NIL error, so the REPORT STRING is the timeout
// discriminator and the command exits 0 — the timeout bounds the observer,
// never the pane.
func TestAwaitTimeoutReturnsRunning(t *testing.T) {
	calls := 0
	start := time.Now()
	got, err := Await(context.Background(), scriptedAwaitObserver([]AwaitReport{""}, &calls),
		time.Millisecond, 40*time.Millisecond)
	if err != nil {
		t.Fatalf("timeout must not be an error, got %v", err)
	}
	if got != AwaitRunning {
		t.Errorf("report = %q, want %q", got, AwaitRunning)
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Errorf("returned after %v — expected to block until the bound", elapsed)
	}
	if calls < 2 {
		t.Errorf("observations = %d, want the loop to have ticked at least once", calls)
	}
}

// TestAwaitTimeoutObservesAtTheBound is the timeout BOUNDARY case: a signal
// that fires AFTER the last tick but BEFORE the bound must be reported, not
// drowned out by `running` — the deadline arm re-observes rather than
// returning the report cached at the last tick.
func TestAwaitTimeoutObservesAtTheBound(t *testing.T) {
	calls := 0
	// The tick is set longer than the whole test so NO tick can fire — the
	// only observations are the entry one ("") and the deadline one (idle).
	got, err := Await(context.Background(), scriptedAwaitObserver([]AwaitReport{"", AwaitIdle}, &calls),
		time.Hour, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("Await: %v", err)
	}
	if got != AwaitIdle {
		t.Errorf("report = %q, want %q (a signal landing in the sub-tick window before the bound)", got, AwaitIdle)
	}
	if calls != 2 {
		t.Errorf("observations = %d, want 2 (entry + the bound's re-observation)", calls)
	}
}

// TestAwaitUnbounded pins the absent/0-timeout contract: the wait is unbounded
// and returns only when a signal fires.
func TestAwaitUnbounded(t *testing.T) {
	calls := 0
	got, err := Await(context.Background(), scriptedAwaitObserver([]AwaitReport{"", "", AwaitGone}, &calls),
		time.Millisecond, 0)
	if err != nil {
		t.Fatalf("Await: %v", err)
	}
	if got != AwaitGone {
		t.Errorf("report = %q, want %q", got, AwaitGone)
	}
}

// TestAwaitObserveError: a genuine read failure aborts the wait and surfaces
// — it is never re-polled forever.
func TestAwaitObserveError(t *testing.T) {
	want := errors.New("tmux read failed")
	observe := func() (AwaitReport, error) { return "", want }
	got, err := Await(context.Background(), observe, time.Millisecond, time.Hour)
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if got != "" {
		t.Errorf("report = %q, want empty on an observe error", got)
	}
}

// TestAwaitCancelled: ctx cancellation returns the last observed report with
// ctx.Err(), so a cancelled await is distinguishable from a timeout.
func TestAwaitCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	got, err := Await(ctx, scriptedAwaitObserver([]AwaitReport{""}, &calls), time.Millisecond, time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if got != "" {
		t.Errorf("report = %q, want the last observed (unfired) report", got)
	}
}
