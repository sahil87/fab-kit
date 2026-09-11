package main

import (
	"errors"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// --- operator clock sync test scaffolding ------------------------------------

// cronListBackoffJSON is a fixture `rk cron list --json` document (rk v3.19.46
// structured-schedule row shape) carrying one operator-tick entry on the
// derived pane/none schedule.
const cronListBackoffJSON = `[{"id":"cron-op","name":"operator tick","schedule":{"kind":"backoff","min":"3m0s","max":"24m0s"},"deliver":"skip-if-busy","target":"role:operator","pinned":true,"muted":false}]`

// cronListBackoffMutedJSON is cronListBackoffJSON with the entry muted (or
// leased — `muted` is the effective state).
const cronListBackoffMutedJSON = `[{"id":"cron-op","name":"operator tick","schedule":{"kind":"backoff","min":"3m0s","max":"24m0s"},"deliver":"skip-if-busy","target":"role:operator","pinned":true,"muted":true,"muted_until":null}]`

// cronListIdleEvery2mJSON carries the entry on the derived shell/agent
// schedule (epoch present): idle-every 2m, skip-if-busy.
const cronListIdleEvery2mJSON = `[{"id":"cron-op","name":"operator tick","schedule":{"kind":"idle-every","every":"2m0s"},"deliver":"skip-if-busy","target":"role:operator","pinned":true,"muted":false}]`

// stubRkCron replaces the rkCronRunner seam: `cron list --json` serves
// listJSON (or listErr), every call's argv is recorded, and any other call
// succeeds silently unless muteErr is set. Returns the recorded argv log.
func stubRkCron(t *testing.T, listJSON string, listErr, muteErr error) *[][]string {
	t.Helper()
	calls := [][]string{}
	prev := rkCronRunner
	rkCronRunner = func(args ...string) (string, error) {
		calls = append(calls, append([]string{}, args...))
		if len(args) == 3 && args[0] == "cron" && args[1] == "list" {
			if listErr != nil {
				return "", listErr
			}
			return listJSON, nil
		}
		if len(args) >= 3 && args[0] == "cron" && args[1] == "mute" && muteErr != nil {
			return "", muteErr
		}
		return "", nil
	}
	t.Cleanup(func() { rkCronRunner = prev })
	return &calls
}

// cronCalls filters the recorded argv log down to invocations of the given
// cron subcommand ("mute" / "edit"), each entry the argv after it.
func cronCalls(calls [][]string, sub string) [][]string {
	var out [][]string
	for _, c := range calls {
		if len(c) >= 3 && c[0] == "cron" && c[1] == sub {
			out = append(out, c[2:])
		}
	}
	return out
}

// wantMutes asserts the exact mute argv sequence (each entry the argv after
// "cron mute").
func wantMutes(t *testing.T, calls [][]string, want ...[]string) {
	t.Helper()
	wantCronCalls(t, calls, "mute", want...)
}

// wantEdits asserts the exact edit argv sequence.
func wantEdits(t *testing.T, calls [][]string, want ...[]string) {
	t.Helper()
	wantCronCalls(t, calls, "edit", want...)
}

func wantCronCalls(t *testing.T, calls [][]string, sub string, want ...[]string) {
	t.Helper()
	got := cronCalls(calls, sub)
	if len(got) != len(want) {
		t.Fatalf("cron %s calls = %v, want %v", sub, got, want)
	}
	for i, w := range want {
		if strings.Join(got[i], " ") != strings.Join(w, " ") {
			t.Errorf("cron %s call %d = %v, want argv %v", sub, i, got[i], w)
		}
	}
}

// stubEpoch pins operatorPaneEpoch's seams (the rk mux panes row and the
// operator-window role lookup) to the given verdict.
func stubEpoch(t *testing.T, epoch bool) {
	t.Helper()
	prevPanes, prevWin := rkPanesRunner, operatorWindowRoleRunner
	row := `null`
	if epoch {
		row = `"active"`
	}
	rkPanesRunner = func(server string) ([]byte, error) {
		return []byte(`[{"window_id":"@1","pane":"%1","agent_state":` + row + `}]`), nil
	}
	operatorWindowRoleRunner = func() (string, error) { return "@1\toperator\n", nil }
	t.Cleanup(func() { rkPanesRunner, operatorWindowRoleRunner = prevPanes, prevWin })
}

// seedState maps a seed YAML document to its data map.
func seedState(t *testing.T, seed string) map[string]interface{} {
	t.Helper()
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(seed), &data); err != nil {
		t.Fatalf("parse seed: %v", err)
	}
	return data
}

// seedOnePaneItem is a state file with a single live pane item.
const seedOnePaneItem = `tracked:
  - id: ab12
    kind: pane
    probe: {mode: pane}
    depends_on: []
    scope: {pane: "%3", repo: /r/a}
    last: {}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
`

// seedTwoShellItems carries two not-done shell items at 2m and 5m cadences.
const seedTwoShellItems = `tracked:
  - id: pr-2m
    kind: shell
    probe: {mode: shell, argv: [p], fields: [state]}
    check_every: 2m
    done_when: 'state == "MERGED"'
    depends_on: []
    scope: {}
    last: {state: OPEN}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
  - id: pr-5m
    kind: shell
    probe: {mode: shell, argv: [p], fields: [state]}
    check_every: 5m
    done_when: 'state == "MERGED"'
    depends_on: []
    scope: {}
    last: {state: OPEN}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
`

// seedDoneItem carries a single item whose done_when fires on last.
const seedDoneItem = `tracked:
  - id: pr-9
    kind: github-pr
    probe: {mode: shell, argv: [p], fields: [state]}
    check_every: 2m
    done_when: 'state == "MERGED"'
    depends_on: []
    scope: {}
    last: {state: MERGED}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
`

// seedDonePlusTask carries one done item and one open (none-probe) item.
const seedDonePlusTask = seedDoneItem + `  - id: other
    kind: task
    probe: {mode: none}
    depends_on: []
    scope: {}
    last: {}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
`

// --- R6: tracked predicate ---------------------------------------------------

func TestOperatorTracked(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		want bool
	}{
		{"empty skeleton", emptyOperatorState(), false},
		{"missing sections", map[string]interface{}{}, false},
		{"one pane item", seedState(t, seedOnePaneItem), true},
		{"only done items is untracked", seedState(t, seedDoneItem), false},
		{"done plus open is tracked", seedState(t, seedDonePlusTask), true},
		{"undecodable tracked section counts as empty", map[string]interface{}{"tracked": 5}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := operatorTracked(tc.data); got != tc.want {
				t.Errorf("operatorTracked = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- R4: edge-triggered flips through the track verbs, then reconcile ---------

func TestClockSync_UntrackedToTracked(t *testing.T) {
	t.Run("pane item unmutes; derived schedule matches the row → no edit", func(t *testing.T) {
		withOperatorState(t, "")
		stubQuietClock(t)
		calls := stubRkCron(t, cronListBackoffJSON, nil, nil)
		if err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("ab12", kindPane,
			"--pane", "%3", "--repo", "/r/a", "--branch", "b")...); err != nil {
			t.Fatalf("track add: %v", err)
		}
		wantMutes(t, *calls, []string{"cron-op", "--off"})
		wantEdits(t, *calls)
	})

	t.Run("shell item unmutes, then the reconcile edits the schedule (R4 order)", func(t *testing.T) {
		withOperatorState(t, "")
		stubQuietClock(t) // panes seam errors → no epoch → --every
		stubGHNameWithOwner(t, "o/r", nil)
		calls := stubRkCron(t, cronListBackoffJSON, nil, nil)
		if err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("pr-913", kindGitHubPR,
			"--scope", `{"repo":"/x","pr":913}`, "--check-every", "2m")...); err != nil {
			t.Fatalf("track add: %v", err)
		}
		wantMutes(t, *calls, []string{"cron-op", "--off"})
		wantEdits(t, *calls, []string{"cron-op", "--every", "2m", "--deliver", "skip-if-busy"})
		// R4: the mute is issued, THEN the schedule reconcile runs.
		var order []string
		for _, c := range *calls {
			if len(c) >= 2 && c[0] == "cron" && (c[1] == "mute" || c[1] == "edit") {
				order = append(order, c[1])
			}
		}
		if strings.Join(order, ",") != "mute,edit" {
			t.Errorf("call order = %v, want mute then edit", order)
		}
	})
}

func TestClockSync_TrackedToUntracked(t *testing.T) {
	// Removing the last not-done item mutes; the reconcile derives "muted —
	// unchanged" and issues no edit.
	withOperatorState(t, seedOnePaneItem)
	stubQuietClock(t)
	calls := stubRkCron(t, cronListBackoffJSON, nil, nil)
	if err := runOperatorCmd(t, operatorTrackRmCmd(), "ab12"); err != nil {
		t.Fatalf("track rm: %v", err)
	}
	wantMutes(t, *calls, []string{"cron-op"})
	wantEdits(t, *calls)
}

func TestClockSync_NonFlippingMutationsStayQuiet(t *testing.T) {
	withOperatorState(t, seedOnePaneItem)
	stubQuietClock(t)
	calls := stubRkCron(t, cronListBackoffJSON, nil, nil)
	if err := runOperatorCmd(t, operatorTrackUpdateCmd(), "ab12", "--then", "report"); err != nil {
		t.Fatalf("track update: %v", err)
	}
	wantMutes(t, *calls)
	wantEdits(t, *calls) // derived backoff/skip-if-busy equals the row
}

// --- R12: derived schedule reconcile -------------------------------------------

func TestReconcile_DerivedSchedule(t *testing.T) {
	tests := []struct {
		name     string
		seed     string
		row      string
		epoch    bool
		wantEdit []string // nil → no edit
	}{
		{"pane-only set derives backoff (A-031 idle-every→backoff)", seedOnePaneItem, cronListIdleEvery2mJSON, true,
			[]string{"cron-op", "--backoff", "--min", "3m", "--max", "24m", "--deliver", "skip-if-busy"}},
		{"R12: shell items with epoch derive idle-every min(check_every)", seedTwoShellItems, cronListBackoffJSON, true,
			[]string{"cron-op", "--idle-every", "2m", "--deliver", "skip-if-busy"}},
		{"shell items without epoch derive every", seedTwoShellItems, cronListBackoffJSON, false,
			[]string{"cron-op", "--every", "2m", "--deliver", "skip-if-busy"}},
		{"R12: equal row (2m0s == 2m) issues nothing", seedTwoShellItems, cronListIdleEvery2mJSON, true, nil},
		{"backoff row equal to derived backoff issues nothing", seedOnePaneItem, cronListBackoffJSON, false, nil},
		{"A-023: a muted/leased entry is still edited", seedTwoShellItems, cronListBackoffMutedJSON, true,
			[]string{"cron-op", "--idle-every", "2m", "--deliver", "skip-if-busy"}},
		{"all items done → muted, unchanged (no edit)", seedDoneItem, cronListBackoffJSON, true, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stubEpoch(t, tc.epoch)
			calls := stubRkCron(t, tc.row, nil, nil)
			reconcileOperatorSchedule(seedState(t, tc.seed))
			wantMutes(t, *calls)
			if tc.wantEdit == nil {
				wantEdits(t, *calls)
			} else {
				wantEdits(t, *calls, tc.wantEdit)
			}
		})
	}
}

func TestReconcile_Override(t *testing.T) {
	overrideSeed := seedTwoShellItems + `clock_override:
  schedule: {kind: every, every: 10m}
  deliver: skip-if-busy
  until: "2999-01-01T00:00:00Z"
`
	t.Run("a live override applies instead of the derived value (R13)", func(t *testing.T) {
		stubEpoch(t, true)
		calls := stubRkCron(t, cronListBackoffJSON, nil, nil)
		reconcileOperatorSchedule(seedState(t, overrideSeed))
		wantEdits(t, *calls, []string{"cron-op", "--every", "10m", "--deliver", "skip-if-busy"})
	})

	t.Run("an expired override reverts to the derived value and leaves the file (R13)", func(t *testing.T) {
		expired := strings.Replace(overrideSeed, "2999-01-01", "2020-01-01", 1)
		path := withOperatorState(t, expired)
		stubQuietClock(t) // no epoch → --every
		calls := stubRkCron(t, cronListBackoffJSON, nil, nil)
		// Any mutation runs the pre-save expiry + the reconcile.
		if err := runOperatorCmd(t, operatorTrackUpdateCmd(), "pr-2m", "--pause"); err != nil {
			t.Fatalf("track update: %v", err)
		}
		if _, ok := readStateFile(t, path)["clock_override"]; ok {
			t.Error("expired clock_override must be removed from the file")
		}
		wantEdits(t, *calls, []string{"cron-op", "--every", "2m", "--deliver", "skip-if-busy"})
	})
}

// --- epoch detection -------------------------------------------------------------

func TestOperatorPaneEpoch(t *testing.T) {
	tests := []struct {
		name     string
		panesOut string
		panesErr error
		windows  string
		winErr   error
		tmuxPane string
		want     bool
	}{
		{"role-marked window with agent_state", `[{"window_id":"@1","pane":"%1","agent_state":"active"}]`, nil, "@1\toperator\n@2\t\n", nil, "", true},
		{"role-marked window with null agent_state", `[{"window_id":"@1","pane":"%1","agent_state":null}]`, nil, "@1\toperator\n", nil, "", false},
		{"no role window falls back to $TMUX_PANE", `[{"window_id":"@9","pane":"%7","agent_state":"idle"}]`, nil, "", nil, "%7", true},
		{"$TMUX_PANE row with null agent_state", `[{"window_id":"@9","pane":"%7","agent_state":null}]`, nil, "", nil, "%7", false},
		{"rk failure degrades to no epoch", "", errors.New("no rk"), "@1\toperator\n", nil, "", false},
		{"no matching row", `[{"window_id":"@2","pane":"%2","agent_state":"active"}]`, nil, "@1\toperator\n", nil, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prevPanes, prevWin := rkPanesRunner, operatorWindowRoleRunner
			rkPanesRunner = func(server string) ([]byte, error) {
				if tc.panesErr != nil {
					return nil, tc.panesErr
				}
				return []byte(tc.panesOut), nil
			}
			operatorWindowRoleRunner = func() (string, error) { return tc.windows, tc.winErr }
			t.Cleanup(func() { rkPanesRunner, operatorWindowRoleRunner = prevPanes, prevWin })
			if tc.tmuxPane != "" {
				t.Setenv("TMUX_PANE", tc.tmuxPane)
			}
			if got := operatorPaneEpoch(); got != tc.want {
				t.Errorf("operatorPaneEpoch = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- tick-start reconcile ---------------------------------------------------------

func TestTickStartDiff_ClockReconcile(t *testing.T) {
	t.Run("untracked state issues exactly one mute and emits the tick doc", func(t *testing.T) {
		withOperatorState(t, "")
		stubQuietClock(t)
		calls := stubRkCron(t, cronListBackoffJSON, nil, nil)
		out, err := runTickDiffArgs(t, "--diff", "--quiet")
		if err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		if !strings.Contains(out, "tick: 1\n") {
			t.Errorf("stdout = %q, want the tick document", out)
		}
		wantMutes(t, *calls, []string{"cron-op"})
		wantEdits(t, *calls)
	})
	t.Run("already-muted entry is left alone", func(t *testing.T) {
		withOperatorState(t, "")
		stubQuietClock(t)
		calls := stubRkCron(t, cronListBackoffMutedJSON, nil, nil)
		if _, err := runTickDiffArgs(t, "--diff", "--quiet"); err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		wantMutes(t, *calls)
		wantEdits(t, *calls)
	})
	t.Run("tracked state issues no mute; derived schedule edits when drifted", func(t *testing.T) {
		withOperatorState(t, seedOnePaneItem)
		stubQuietClock(t)
		stubSnapshot(t, []paneRow{snapRow("%3", "ab12", "apply", "active", "active", "")})
		calls := stubRkCron(t, cronListIdleEvery2mJSON, nil, nil)
		if _, err := runTickDiffArgs(t, "--diff", "--quiet"); err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		wantMutes(t, *calls)
		// The pane-only set derives backoff/skip-if-busy; the row is on
		// idle-every → the end-of-tick reconcile converges it.
		wantEdits(t, *calls, []string{"cron-op", "--backoff", "--min", "3m", "--max", "24m", "--deliver", "skip-if-busy"})
	})
	t.Run("R6: an all-done tracked set mutes the entry", func(t *testing.T) {
		withOperatorState(t, seedDoneItem)
		stubQuietClock(t)
		calls := stubRkCron(t, cronListBackoffJSON, nil, nil)
		if _, err := runTickDiffArgs(t, "--diff", "--quiet"); err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		wantMutes(t, *calls, []string{"cron-op"})
		wantEdits(t, *calls)
	})
}

// --- entry resolution and fail-silent degradation --------------------------------

func TestClockSync_FailSilentDegradation(t *testing.T) {
	tests := []struct {
		name     string
		listJSON string
		listErr  error
	}{
		{"rk absent / list fails", "", errors.New("exec: rk not found")},
		{"malformed JSON", "{not json", nil},
		{"zero candidate rows", `[]`, nil},
		{"two role:operator rows with no name tiebreak", `[{"id":"a","name":"x","target":"role:operator"},{"id":"b","name":"y","target":"role:operator"}]`, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withOperatorState(t, "")
			stubQuietClock(t)
			calls := stubRkCron(t, tc.listJSON, tc.listErr, nil)
			if err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("ab12", kindPane,
				"--pane", "%3", "--repo", "/r/a", "--branch", "b")...); err != nil {
				t.Fatalf("track add must succeed with a degraded rk: %v", err)
			}
			wantMutes(t, *calls)
			wantEdits(t, *calls)
		})
	}

	t.Run("non-zero mute call leaves the verb unchanged", func(t *testing.T) {
		withOperatorState(t, "")
		stubQuietClock(t)
		calls := stubRkCron(t, cronListBackoffJSON, nil, errors.New("exit status 1"))
		if err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("ab12", kindPane,
			"--pane", "%3", "--repo", "/r/a", "--branch", "b")...); err != nil {
			t.Fatalf("track add must succeed when the mute call fails: %v", err)
		}
		wantMutes(t, *calls, []string{"cron-op", "--off"})
	})
}

func TestResolveOperatorCronRow_Tiebreak(t *testing.T) {
	doc := `[{"id":"other","name":"not-the-tick","target":"role:operator"},{"id":"cron-op","name":"operator tick","target":"role:operator"}]`
	stubRkCron(t, doc, nil, nil)
	row, ok := resolveOperatorCronRow()
	if !ok {
		t.Fatal("resolveOperatorCronRow ok = false, want true")
	}
	if row.ID != "cron-op" {
		t.Errorf("row.ID = %q, want %q (name tiebreak)", row.ID, "cron-op")
	}
}

// --- no clock call when the save fails --------------------------------------------

func TestClockSync_SaveFailureIssuesNoCall(t *testing.T) {
	// Point state I/O at a path inside a nonexistent directory so the atomic
	// save fails after the mutation.
	path := strings.Join([]string{t.TempDir(), "missing", "operator-state.yaml"}, "/")
	operatorStatePathOverride = path
	t.Cleanup(func() { operatorStatePathOverride = "" })
	stubQuietClock(t)
	calls := stubRkCron(t, cronListBackoffJSON, nil, nil)
	if err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("ab12", kindPane,
		"--pane", "%3", "--repo", "/r/a", "--branch", "b")...); err == nil {
		t.Fatal("track add = nil error, want a save failure")
	}
	if len(*calls) != 0 {
		t.Errorf("rk calls = %v, want none (state not persisted)", *calls)
	}
}
