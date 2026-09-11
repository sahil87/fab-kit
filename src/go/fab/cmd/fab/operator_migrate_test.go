package main

import (
	"os"
	"strings"
	"testing"
)

// --- legacy state-file conversion (R5) ----------------------------------------

// legacyV10State is a full legacy-shaped file: two monitored entries, one
// watch, an exhausted autopilot block (state/current null, queue/completed
// retained), and one open + one resolved note.
const legacyV10State = `tick_count: 9
monitored:
  ab12:
    pane: "%3"
    repo: /home/u/foo
    session: work
    stage: apply
    agent: active
    stop_stage: null
    spawned_by: null
    depends_on: []
    branch: 260909-ab12-x
    enrolled_at: "2026-09-09T00:00:00Z"
    last_transition: "2026-09-09T01:00:00Z"
  cd34:
    pane: "%4"
    repo: /home/u/bar
    session: ops
    branch: 260909-cd34-y
    enrolled_at: "2026-09-09T02:00:00Z"
    last_transition: "2026-09-09T02:30:00Z"
watches:
  linear-bugs:
    enabled: true
    source: linear
    query:
      project: DEV
    target_repo: /home/u/foo
    stop_stage: intake
    known: [DEV-1, DEV-2]
    completed: [DEV-2, DEV-3]
    last_checked: "2026-09-09T03:00:00Z"
    instructions: spawn each new bug
autopilot:
  queue: [ab12, cd34]
  current: null
  completed: [ab12, cd34]
  state: null
  mode: cherry-pick-ladder
notes:
  - id: n1
    kind: coordination
    text: merge sequence open
    refs: [ab12]
    created_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:10:00Z"
    resolved: false
  - id: n2
    kind: correction
    text: typo fix
    created_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:20:00Z"
    resolved: true
notes_seq: 2
branch_map:
  ab12: {branch: 260909-ab12-x, repo: /home/u/foo}
`

// readTracked decodes the tracked list from the state file, indexed by id.
func readTracked(t *testing.T, path string) map[string]trackedItem {
	t.Helper()
	items := []trackedItem{}
	if err := operatorSection(readStateFile(t, path), "tracked", &items); err != nil {
		t.Fatalf("decode tracked: %v", err)
	}
	byID := map[string]trackedItem{}
	for _, it := range items {
		byID[it.ID] = it
	}
	return byID
}

func TestMigrate_V10FileConvertsOnFirstTouch(t *testing.T) {
	path := withOperatorState(t, legacyV10State)

	if err := runOperatorCmd(t, operatorStateCmd()); err != nil {
		t.Fatalf("state: %v", err)
	}

	state := readStateFile(t, path)
	for _, k := range []string{"monitored", "watches", "autopilot", "notes", "notes_seq"} {
		if _, ok := state[k]; ok {
			t.Errorf("legacy key %q survived conversion", k)
		}
	}
	if _, ok := state["branch_map"]; !ok {
		t.Error("branch_map lost in conversion")
	}

	items := readTracked(t, path)
	// R5's scenario text says "five" but its own breakdown (2 pane,
	// 1 linear, 1 note, 0 from the exhausted queue) sums to 4 — the breakdown
	// is the contract (flagged as a review finding).
	if len(items) != 4 {
		t.Fatalf("tracked items = %d, want 4 (2 pane, 1 linear, 1 note; 0 from the exhausted queue): %v", len(items), items)
	}

	ab12 := items["ab12"]
	if ab12.Kind != kindPane || ab12.Probe.Mode != probePane {
		t.Errorf("ab12 kind/probe = %s/%s", ab12.Kind, ab12.Probe.Mode)
	}
	for k, want := range map[string]string{"pane": "%3", "change": "ab12", "repo": "/home/u/foo", "session": "work", "branch": "260909-ab12-x", "stage": "apply", "agent": "active"} {
		if got := scopeString(ab12.Scope, k); got != want {
			t.Errorf("ab12 scope.%s = %q, want %q", k, got, want)
		}
	}
	if v, present := ab12.Scope["pane_pid"]; !present || v != nil {
		t.Errorf("ab12 scope.pane_pid = %v (present %v), want key present with null (no live fingerprint)", v, present)
	}
	if ab12.CheckedAt == nil || *ab12.CheckedAt != "2026-09-09T01:00:00Z" {
		t.Errorf("ab12 checked_at = %v, want last_transition", ab12.CheckedAt)
	}
	if ab12.AddedAt != "2026-09-09T00:00:00Z" || ab12.UpdatedAt != "2026-09-09T01:00:00Z" {
		t.Errorf("ab12 timestamps = %s/%s, want enrolled_at/last_transition", ab12.AddedAt, ab12.UpdatedAt)
	}

	lw := items["linear-bugs"]
	if lw.Kind != kindLinear || lw.Probe.Mode != probeAgent || lw.Probe.Instruction == "" {
		t.Errorf("linear-bugs kind/probe = %+v", lw.Probe)
	}
	if lw.CheckEvery == nil || *lw.CheckEvery != "5m" {
		t.Errorf("linear-bugs check_every = %v, want 5m", lw.CheckEvery)
	}
	if got := strings.Join(lw.Seen, ","); got != "DEV-1,DEV-2,DEV-3" {
		t.Errorf("linear-bugs seen = %v, want known ∪ completed deduped", lw.Seen)
	}
	if lw.Then == nil || *lw.Then != "spawn each new bug" {
		t.Errorf("linear-bugs then = %v, want instructions", lw.Then)
	}
	if got := scopeString(lw.Scope, "repo"); got != "/home/u/foo" {
		t.Errorf("linear-bugs scope.repo = %q", got)
	}
	if got := scopeString(lw.Scope, "stop_stage"); got != "intake" {
		t.Errorf("linear-bugs scope.stop_stage = %q", got)
	}
	if lw.Scope["query"] == nil {
		t.Error("linear-bugs scope.query missing")
	}
	if lw.CheckedAt == nil || *lw.CheckedAt != "2026-09-09T03:00:00Z" {
		t.Errorf("linear-bugs checked_at = %v, want last_checked", lw.CheckedAt)
	}

	n1 := items["n1"]
	if n1.Kind != kindNote || n1.Text != "merge sequence open" {
		t.Errorf("n1 = %+v, want the open note as a note item", n1)
	}
	if _, ok := items["n2"]; ok {
		t.Error("resolved note n2 must be dropped")
	}

	// Idempotence: the second run changes nothing (state is a read verb on a
	// converted file — byte-stable).
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := runOperatorCmd(t, operatorStateCmd()); err != nil {
		t.Fatalf("state (second run): %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("second run rewrote the file:\n%s\n---\n%s", first, second)
	}
}

func TestMigrate_RunningAutopilotRefuses(t *testing.T) {
	seed := legacyV10State + ""
	seed = strings.Replace(seed, "  state: null", "  state: running", 1)
	seed = strings.Replace(seed, "  current: null", "  current: ab12", 1)

	verbs := []struct {
		name string
		run  func(t *testing.T) error
	}{
		{"state", func(t *testing.T) error { return runOperatorCmd(t, operatorStateCmd()) }},
		{"tick-start", func(t *testing.T) error { return runOperatorCmd(t, operatorTickStartCmd()) }},
		{"branch-map rm --all", func(t *testing.T) error {
			return runOperatorCmd(t, operatorBranchMapRmCmd(), "--all")
		}},
	}
	for _, v := range verbs {
		t.Run(v.name, func(t *testing.T) {
			path := withOperatorState(t, seed)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			err = v.run(t)
			if err == nil {
				t.Fatal("verb succeeded, want the running-autopilot refusal")
			}
			if err.Error() != operatorLegacyRunningRefusal {
				t.Errorf("error = %q, want exactly %q", err.Error(), operatorLegacyRunningRefusal)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Error("refusal must write nothing — the file changed")
			}
		})
	}
}

func TestMigrate_AutopilotQueueChainsItems(t *testing.T) {
	// A-024: a paused queue's not-completed entries convert to pane-less
	// pane-kind items chained by depends_on (nearest same-repo predecessor;
	// cross-repo → immediate predecessor) with scope.merge_mode set and
	// scope.change seeded from the id.
	seed := `autopilot:
  queue: [a1, b1, a2, done0]
  current: a1
  completed: [done0]
  state: paused
  mode: stacked-prs
branch_map:
  a1: {branch: br-a1, repo: /r/a}
  b1: {branch: br-b1, repo: /r/b}
  a2: {branch: br-a2, repo: /r/a}
  done0: {branch: br-d0, repo: /r/a}
`
	path := withOperatorState(t, seed)
	if err := runOperatorCmd(t, operatorStateCmd()); err != nil {
		t.Fatalf("state: %v", err)
	}
	items := readTracked(t, path)
	if len(items) != 3 {
		t.Fatalf("tracked items = %d, want 3 (completed entries convert nothing): %v", len(items), items)
	}
	a1 := items["a1"]
	if a1.Kind != kindPane {
		t.Errorf("a1 kind = %s, want pane", a1.Kind)
	}
	if got := scopeString(a1.Scope, "pane"); got != "" {
		t.Errorf("a1 scope.pane = %q, want null (pending)", got)
	}
	if got := scopeString(a1.Scope, "change"); got != "a1" {
		t.Errorf("a1 scope.change = %q, want seeded from the id", got)
	}
	if len(a1.DependsOn) != 0 {
		t.Errorf("a1 depends_on = %v, want []", a1.DependsOn)
	}
	if got := scopeString(a1.Scope, "merge_mode"); got != "stacked-prs" {
		t.Errorf("a1 scope.merge_mode = %q, want autopilot.mode", got)
	}
	if got := scopeString(a1.Scope, "branch"); got != "br-a1" {
		t.Errorf("a1 scope.branch = %q, want branch_map's", got)
	}
	if got := strings.Join(items["b1"].DependsOn, ","); got != "a1" {
		t.Errorf("b1 depends_on = %v, want [a1] (cross-repo → immediate predecessor)", items["b1"].DependsOn)
	}
	if got := strings.Join(items["a2"].DependsOn, ","); got != "a1" {
		t.Errorf("a2 depends_on = %v, want [a1] (nearest same-repo predecessor)", items["a2"].DependsOn)
	}
}

func TestMigrate_DisabledWatchConvertsPaused(t *testing.T) {
	seed := `watches:
  w1:
    enabled: false
    source: slack
    target_repo: /home/u/foo
    known: []
    completed: []
`
	path := withOperatorState(t, seed)
	if err := runOperatorCmd(t, operatorStateCmd()); err != nil {
		t.Fatalf("state: %v", err)
	}
	w1 := readTracked(t, path)["w1"]
	if w1.Kind != kindSlack || !w1.Paused {
		t.Errorf("w1 = kind %s paused %v, want slack/paused (disabled watch)", w1.Kind, w1.Paused)
	}
}

func TestMigrate_FabChangeItemsConvertWithTrackedPresent(t *testing.T) {
	// R11: a file with `tracked` present holding a retired kind: fab-change
	// item converts on ANY verb's read-modify-write — kind: pane,
	// scope.change seeded from the item id, scope.pane_pid null — in the same
	// atomic write; a stray legacy key survives as an unknown top-level key
	// (A-034); a second run is a byte-stable no-op (idempotent).
	seed := `tracked:
  - id: ab12
    kind: fab-change
    probe: {mode: pane}
    depends_on: []
    scope: {pane: "%3", repo: /home/u/foo}
    last: {}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
notes:
  - id: n1
    kind: correction
    text: stray legacy note
    created_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
    resolved: false
`

	t.Run("read verb converts and saves", func(t *testing.T) {
		path := withOperatorState(t, seed)
		if err := runOperatorCmd(t, operatorStateCmd()); err != nil {
			t.Fatalf("state: %v", err)
		}
		it := readTracked(t, path)["ab12"]
		if it.Kind != kindPane {
			t.Errorf("kind = %s, want pane", it.Kind)
		}
		if got := scopeString(it.Scope, "change"); got != "ab12" {
			t.Errorf("scope.change = %q, want seeded from the item id", got)
		}
		if v, present := it.Scope["pane_pid"]; !present || v != nil {
			t.Errorf("scope.pane_pid = %v (present %v), want key present with null", v, present)
		}
		if got := scopeString(it.Scope, "pane"); got != "%3" {
			t.Errorf("scope.pane = %q, want preserved %%3", got)
		}
		if _, ok := readStateFile(t, path)["notes"]; !ok {
			t.Error("stray legacy key lost — unknown top-level keys survive the conversion")
		}

		// Idempotent: the second run rewrites nothing.
		first, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := runOperatorCmd(t, operatorStateCmd()); err != nil {
			t.Fatalf("state (second run): %v", err)
		}
		second, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(first) != string(second) {
			t.Errorf("second run rewrote the file:\n%s\n---\n%s", first, second)
		}
	})

	t.Run("write verb converts in its own atomic write", func(t *testing.T) {
		path := withOperatorState(t, seed)
		stubQuietClock(t)
		if err := runOperatorCmd(t, operatorTrackClockCmd(), "--every", "10m", "--for", "1h"); err != nil {
			t.Fatalf("track clock: %v", err)
		}
		it := readTracked(t, path)["ab12"]
		if it.Kind != kindPane || scopeString(it.Scope, "change") != "ab12" {
			t.Errorf("after a write verb: kind %s scope.change %q, want pane / ab12", it.Kind, scopeString(it.Scope, "change"))
		}
		if readClockOverride(readStateFile(t, path)) == nil {
			t.Error("the verb's own mutation lost in the converting write")
		}
	})
}
