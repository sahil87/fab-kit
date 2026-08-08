package dispatch

import "testing"

// TestDecideReap walks the whole mode × state × knob matrix. The two properties
// worth pinning are that a headless record short-circuits before either later input
// is consulted (so reap can never act on a mode with no pane), and — the load-bearing
// one — that NO value of the knob reaches a non-`done` dispatch: reap is not kill,
// and that invariant must hold independently of configuration.
func TestDecideReap(t *testing.T) {
	tests := []struct {
		name     string
		isPane   bool
		state    State
		reapDone bool
		want     ReapVerdict
	}{
		// The only verdict that kills anything.
		{"pane + done + enabled ⇒ reap", true, StateDone, true, ReapPane},

		// Policy: a done pane worker the user asked to keep.
		{"pane + done + disabled ⇒ skip (disabled)", true, StateDone, false, ReapSkipDisabled},

		// Reap is NOT kill — every reachable non-done pane state, under BOTH knob
		// values, so the never-terminate-a-live-worker rule is config-independent.
		{"pane + running + enabled ⇒ skip (not-done)", true, StateRunning, true, ReapSkipNotDone},
		{"pane + running + disabled ⇒ skip (not-done)", true, StateRunning, false, ReapSkipNotDone},
		{"pane + orphaned + enabled ⇒ skip (not-done)", true, StateOrphaned, true, ReapSkipNotDone},
		{"pane + orphaned + disabled ⇒ skip (not-done)", true, StateOrphaned, false, ReapSkipNotDone},
		// Unreachable on the pane path (no exit-code channel), but the guard must not
		// depend on that reachability to stay safe.
		{"pane + failed + enabled ⇒ skip (not-done)", true, StateFailed, true, ReapSkipNotDone},
		{"pane + failed (no-result) + enabled ⇒ skip (not-done)", true, StateFailedNoResult, true, ReapSkipNotDone},

		// Headless short-circuits on the mode, whatever the other two say — the
		// worker process already exited, so there is nothing visual to reclaim.
		{"headless + done + enabled ⇒ skip (headless)", false, StateDone, true, ReapSkipHeadless},
		{"headless + done + disabled ⇒ skip (headless)", false, StateDone, false, ReapSkipHeadless},
		{"headless + running + enabled ⇒ skip (headless)", false, StateRunning, true, ReapSkipHeadless},
		{"headless + failed + enabled ⇒ skip (headless)", false, StateFailed, true, ReapSkipHeadless},
		{"headless + failed (no-result) + enabled ⇒ skip (headless)", false, StateFailedNoResult, true, ReapSkipHeadless},
		{"headless + orphaned + enabled ⇒ skip (headless)", false, StateOrphaned, true, ReapSkipHeadless},
		// A headless record's state is never derived by the caller, so the zero value
		// reaches the guard — it must still report the mode, not an empty reason.
		{"headless + zero state ⇒ skip (headless)", false, "", true, ReapSkipHeadless},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DecideReap(tt.isPane, tt.state, tt.reapDone); got != tt.want {
				t.Errorf("DecideReap(pane=%v, state=%q, reapDone=%v) = %q, want %q",
					tt.isPane, tt.state, tt.reapDone, got, tt.want)
			}
		})
	}
}

// TestReapVerdictsAreDistinct: the verdicts double as the reason the command
// reports, so two colliding values would make one no-op explain itself as another.
func TestReapVerdictsAreDistinct(t *testing.T) {
	seen := map[ReapVerdict]string{}
	for name, v := range map[string]ReapVerdict{
		"ReapPane":         ReapPane,
		"ReapSkipHeadless": ReapSkipHeadless,
		"ReapSkipNotDone":  ReapSkipNotDone,
		"ReapSkipDisabled": ReapSkipDisabled,
	} {
		if v == "" {
			t.Errorf("%s is empty — every verdict must name itself in a report", name)
		}
		if prior, dup := seen[v]; dup {
			t.Errorf("%s and %s share the value %q", name, prior, v)
		}
		seen[v] = name
	}
}
