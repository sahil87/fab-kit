package main

import (
	"errors"
	"strings"
	"testing"
)

// --- operator clock sync test scaffolding ------------------------------------

// cronListJSON is a fixture `rk cron list --json` document (rk v3.19.37 row
// shape) carrying one operator-tick entry.
const cronListJSON = `[{"id":"cron-op","name":"operator tick","schedule":"backoff","target":"role:operator","deliver":"immediate","pinned":true,"muted":false,"last_fired":null,"orphaned_since":null,"expires_at":null}]`

// cronListJSONMuted is cronListJSON with the entry already muted (the
// reconcile's not-drifted case).
const cronListJSONMuted = `[{"id":"cron-op","name":"operator tick","schedule":"backoff","target":"role:operator","deliver":"immediate","pinned":true,"muted":true,"muted_until":null,"last_fired":null,"orphaned_since":null,"expires_at":null}]`

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

// muteCalls filters the recorded argv log down to `cron mute` invocations.
func muteCalls(calls [][]string) [][]string {
	var out [][]string
	for _, c := range calls {
		if len(c) >= 3 && c[0] == "cron" && c[1] == "mute" {
			out = append(out, c)
		}
	}
	return out
}

// wantMutes asserts the exact mute argv sequence (each entry the argv after
// "cron mute").
func wantMutes(t *testing.T, calls [][]string, want ...[]string) {
	t.Helper()
	got := muteCalls(calls)
	if len(got) != len(want) {
		t.Fatalf("mute calls = %v, want %v", got, want)
	}
	for i, w := range want {
		if strings.Join(got[i][2:], " ") != strings.Join(w, " ") {
			t.Errorf("mute call %d = %v, want argv %v", i, got[i], w)
		}
	}
}

const clockSeedMonitored = `monitored:
  ab12:
    pane: "%3"
    repo: /home/u/foo
    session: work
    branch: 260909-ab12-x
    enrolled_at: "2026-09-09T00:00:00Z"
    last_transition: "2026-09-09T00:00:00Z"
`

const clockSeedWatch = `watches:
  w1:
    enabled: false
    source: linear
    target_repo: /home/u/foo
    known: []
    completed: []
`

const clockSeedAutopilot = `autopilot:
  queue: [ab12]
  current: ab12
  completed: []
  state: running
  mode: cherry-pick-ladder
`

const clockSeedCoordNote = `notes:
  - id: n1
    kind: coordination
    text: merge sequence open
    created_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
    resolved: false
notes_seq: 1
`

// --- R1: tracked predicate ---------------------------------------------------

func TestOperatorTracked(t *testing.T) {
	running, paused := "running", "paused"
	tests := []struct {
		name string
		data map[string]interface{}
		want bool
	}{
		{"empty skeleton", emptyOperatorState(), false},
		{"missing sections", map[string]interface{}{}, false},
		{"one monitored entry", map[string]interface{}{
			"monitored": map[string]monitoredEntry{"ab12": {Pane: "%3"}},
		}, true},
		{"autopilot running", map[string]interface{}{
			"autopilot": &autopilotState{Queue: []string{"ab12"}, State: &running},
		}, true},
		{"autopilot paused", map[string]interface{}{
			"autopilot": &autopilotState{Queue: []string{"ab12"}, State: &paused},
		}, true},
		{"autopilot exhausted (state null, queue retained)", map[string]interface{}{
			"autopilot": &autopilotState{Queue: []string{"ab12"}, Completed: []string{"ab12"}},
		}, false},
		{"one enabled watch", map[string]interface{}{
			"watches": map[string]watchEntry{"w1": {Enabled: true, Source: "linear"}},
		}, true},
		{"one disabled watch still counts", map[string]interface{}{
			"watches": map[string]watchEntry{"w1": {Enabled: false, Source: "linear"}},
		}, true},
		{"open coordination note", map[string]interface{}{
			"notes": []noteEntry{{ID: "n1", Kind: "coordination"}},
		}, true},
		{"resolved coordination note", map[string]interface{}{
			"notes": []noteEntry{{ID: "n1", Kind: "coordination", Resolved: true}},
		}, false},
		{"open correction note does not count", map[string]interface{}{
			"notes": []noteEntry{{ID: "n1", Kind: "correction"}},
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := operatorTracked(tc.data); got != tc.want {
				t.Errorf("operatorTracked = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- R2: edge-triggered flips through the verbs ------------------------------

func TestClockSync_UntrackedToTracked(t *testing.T) {
	tests := []struct {
		name string
		seed string
		run  func(t *testing.T) error
	}{
		{"enroll first monitored entry", "", func(t *testing.T) error {
			return runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("ab12")...)
		}},
		{"watch add first watch", "", func(t *testing.T) error {
			return runOperatorCmd(t, operatorWatchAddCmd(), "w1", "--source", "linear", "--target-repo", "/home/u/foo")
		}},
		{"autopilot start", "", func(t *testing.T) error {
			return runOperatorCmd(t, operatorAutopilotCmd(), "start", "--queue", "ab12")
		}},
		{"note add --kind coordination", "", func(t *testing.T) error {
			return runOperatorCmd(t, operatorNoteAddCmd(), "merge sequence open", "--kind", "coordination")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withOperatorState(t, tc.seed)
			calls := stubRkCron(t, cronListJSON, nil, nil)
			if err := tc.run(t); err != nil {
				t.Fatalf("verb: %v", err)
			}
			wantMutes(t, *calls, []string{"cron-op", "--off"})
		})
	}
}

func TestClockSync_TrackedToUntracked(t *testing.T) {
	tests := []struct {
		name string
		seed string
		run  func(t *testing.T) error
	}{
		{"remove last monitored entry", clockSeedMonitored, func(t *testing.T) error {
			return runOperatorCmd(t, operatorRemoveCmd(), "ab12")
		}},
		{"watch rm last watch", clockSeedWatch, func(t *testing.T) error {
			return runOperatorCmd(t, operatorWatchRmCmd(), "w1")
		}},
		{"autopilot stop", clockSeedAutopilot, func(t *testing.T) error {
			return runOperatorCmd(t, operatorAutopilotCmd(), "stop")
		}},
		{"note resolve last coordination note", clockSeedCoordNote, func(t *testing.T) error {
			return runOperatorCmd(t, operatorNoteResolveCmd(), "n1")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withOperatorState(t, tc.seed)
			calls := stubRkCron(t, cronListJSON, nil, nil)
			if err := tc.run(t); err != nil {
				t.Fatalf("verb: %v", err)
			}
			wantMutes(t, *calls, []string{"cron-op"})
		})
	}
}

func TestClockSync_NonFlippingMutationsStayQuiet(t *testing.T) {
	tests := []struct {
		name string
		seed string
		run  func(t *testing.T) error
	}{
		{"second enroll (tracked stays tracked)", clockSeedMonitored, func(t *testing.T) error {
			return runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("cd34")...)
		}},
		{"update observed fields", clockSeedMonitored, func(t *testing.T) error {
			return runOperatorCmd(t, operatorUpdateCmd(), "ab12", "--stage", "review")
		}},
		{"note add --kind correction (untracked stays untracked)", "", func(t *testing.T) error {
			return runOperatorCmd(t, operatorNoteAddCmd(), "typo fix", "--kind", "correction")
		}},
		{"coordination note while already tracked", clockSeedMonitored, func(t *testing.T) error {
			return runOperatorCmd(t, operatorNoteAddCmd(), "merge sequence open", "--kind", "coordination")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withOperatorState(t, tc.seed)
			calls := stubRkCron(t, cronListJSON, nil, nil)
			if err := tc.run(t); err != nil {
				t.Fatalf("verb: %v", err)
			}
			wantMutes(t, *calls)
		})
	}
}

// --- R2: autopilot advance to exhaustion -------------------------------------

func TestClockSync_AutopilotAdvanceExhaustion(t *testing.T) {
	t.Run("mutes when nothing else tracked", func(t *testing.T) {
		withOperatorState(t, clockSeedAutopilot)
		calls := stubRkCron(t, cronListJSON, nil, nil)
		if err := runOperatorCmd(t, operatorAutopilotCmd(), "advance"); err != nil {
			t.Fatalf("advance: %v", err)
		}
		wantMutes(t, *calls, []string{"cron-op"})
	})
	t.Run("open coordination note keeps it tracked", func(t *testing.T) {
		withOperatorState(t, clockSeedAutopilot+"\n"+clockSeedCoordNote)
		calls := stubRkCron(t, cronListJSON, nil, nil)
		if err := runOperatorCmd(t, operatorAutopilotCmd(), "advance"); err != nil {
			t.Fatalf("advance: %v", err)
		}
		wantMutes(t, *calls)
	})
}

// --- R6: tick-start reconcile -------------------------------------------------

func TestTickStartDiff_ClockReconcile(t *testing.T) {
	t.Run("untracked state issues exactly one mute and emits the tick doc", func(t *testing.T) {
		withOperatorState(t, "")
		calls := stubRkCron(t, cronListJSON, nil, nil)
		out, err := runTickDiffArgs(t, "--diff", "--quiet")
		if err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		if !strings.Contains(out, "tick: 1\n") {
			t.Errorf("stdout = %q, want the tick document", out)
		}
		wantMutes(t, *calls, []string{"cron-op"})
	})
	t.Run("already-muted entry is left alone", func(t *testing.T) {
		withOperatorState(t, "")
		calls := stubRkCron(t, cronListJSONMuted, nil, nil)
		if _, err := runTickDiffArgs(t, "--diff", "--quiet"); err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		wantMutes(t, *calls)
	})
	t.Run("tracked state issues no call", func(t *testing.T) {
		withOperatorState(t, clockSeedMonitored)
		stubSnapshot(t, []paneRow{snapRow("%3", "ab12", "apply", "active", "active", "")})
		calls := stubRkCron(t, cronListJSON, nil, nil)
		if _, err := runTickDiffArgs(t, "--diff", "--quiet"); err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		wantMutes(t, *calls)
	})
}

// --- R3/R4: entry resolution and fail-silent degradation ----------------------

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
			calls := stubRkCron(t, tc.listJSON, tc.listErr, nil)
			if err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("ab12")...); err != nil {
				t.Fatalf("enroll must succeed with a degraded rk: %v", err)
			}
			wantMutes(t, *calls)
		})
	}

	t.Run("non-zero mute call leaves the verb unchanged", func(t *testing.T) {
		withOperatorState(t, "")
		calls := stubRkCron(t, cronListJSON, nil, errors.New("exit status 1"))
		if err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("ab12")...); err != nil {
			t.Fatalf("enroll must succeed when the mute call fails: %v", err)
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

// --- R2: no clock call when the save fails ------------------------------------

func TestClockSync_SaveFailureIssuesNoCall(t *testing.T) {
	// Point state I/O at a path inside a nonexistent directory so the atomic
	// save fails after the mutation.
	path := strings.Join([]string{t.TempDir(), "missing", "operator-state.yaml"}, "/")
	operatorStatePathOverride = path
	t.Cleanup(func() { operatorStatePathOverride = "" })
	calls := stubRkCron(t, cronListJSON, nil, nil)
	if err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("ab12")...); err == nil {
		t.Fatal("enroll = nil error, want a save failure")
	}
	wantMutes(t, *calls)
	if len(*calls) != 0 {
		t.Errorf("rk calls = %v, want none (state not persisted)", *calls)
	}
}
