package main

import (
	"fmt"
	"strings"
	"testing"
)

// readWatches decodes the watches section into typed entries.
func readWatches(t *testing.T, path string) map[string]watchEntry {
	t.Helper()
	w := map[string]watchEntry{}
	if err := operatorSection(readStateFile(t, path), "watches", &w); err != nil {
		t.Fatalf("decode watches: %v", err)
	}
	return w
}

func addWatch(t *testing.T, extra ...string) error {
	t.Helper()
	args := append([]string{"linear-bugs", "--source", "linear", "--target-repo", "/home/u/foo"}, extra...)
	return runOperatorCmd(t, operatorWatchAddCmd(), args...)
}

func TestOperatorWatchAdd_CreatesWithDefaults(t *testing.T) {
	path := withOperatorState(t, "")

	err := addWatch(t, "--query", `{"project":"DEV","status":["Backlog","Todo"]}`,
		"--stop-stage", "intake", "--instructions", "Spawn for label bug.")
	if err != nil {
		t.Fatalf("watch add: %v", err)
	}
	w := readWatches(t, path)["linear-bugs"]
	if !w.Enabled {
		t.Error("enabled = false, want true")
	}
	if w.Source != "linear" || w.TargetRepo != "/home/u/foo" {
		t.Errorf("identity fields wrong: %+v", w)
	}
	if w.Query["project"] != "DEV" {
		t.Errorf("query.project = %v", w.Query["project"])
	}
	statuses, ok := w.Query["status"].([]interface{})
	if !ok || len(statuses) != 2 || statuses[0] != "Backlog" {
		t.Errorf("query.status nested list not preserved: %v", w.Query["status"])
	}
	if w.StopStage == nil || *w.StopStage != "intake" {
		t.Errorf("stop_stage = %v", w.StopStage)
	}
	if w.Instructions != "Spawn for label bug." {
		t.Errorf("instructions = %q", w.Instructions)
	}
	if w.Known == nil || len(w.Known) != 0 || w.Completed == nil || len(w.Completed) != 0 {
		t.Errorf("known/completed must be empty non-nil: %v / %v", w.Known, w.Completed)
	}
	if w.LastChecked != nil || w.LastError != nil {
		t.Errorf("last_checked/last_error must be null: %v / %v", w.LastChecked, w.LastError)
	}
}

func TestOperatorWatchAdd_Errors(t *testing.T) {
	withOperatorState(t, "")

	if err := runOperatorCmd(t, operatorWatchAddCmd(), "w", "--source", "pagerduty", "--target-repo", "/x"); err == nil {
		t.Error("invalid --source must error")
	}
	if err := runOperatorCmd(t, operatorWatchAddCmd(), "w", "--source", "linear"); err == nil {
		t.Error("missing --target-repo must error")
	}
	if err := addWatch(t, "--query", `{not json`); err == nil {
		t.Error("invalid --query JSON must error")
	}
	if err := addWatch(t, "--query", `["a","b"]`); err == nil {
		t.Error("non-object --query must error")
	}
	if err := addWatch(t, "--stop-stage", "bogus"); err == nil {
		t.Error("invalid --stop-stage must error")
	}
	if err := addWatch(t); err != nil {
		t.Fatalf("watch add: %v", err)
	}
	if err := addWatch(t); err == nil {
		t.Error("duplicate watch add must error")
	}
}

func TestOperatorWatchRm(t *testing.T) {
	path := withOperatorState(t, "")
	if err := addWatch(t); err != nil {
		t.Fatalf("watch add: %v", err)
	}
	if err := runOperatorCmd(t, operatorWatchRmCmd(), "linear-bugs"); err != nil {
		t.Fatalf("watch rm: %v", err)
	}
	if _, ok := readWatches(t, path)["linear-bugs"]; ok {
		t.Error("watch still present after rm")
	}
	if err := runOperatorCmd(t, operatorWatchRmCmd(), "ghost"); err == nil {
		t.Error("rm of unknown watch must error")
	}
}

func TestOperatorWatchToggle(t *testing.T) {
	path := withOperatorState(t, "")
	if err := addWatch(t); err != nil {
		t.Fatalf("watch add: %v", err)
	}

	flip := func(args ...string) bool {
		t.Helper()
		full := append([]string{"linear-bugs"}, args...)
		if err := runOperatorCmd(t, operatorWatchToggleCmd(), full...); err != nil {
			t.Fatalf("watch toggle %v: %v", args, err)
		}
		return readWatches(t, path)["linear-bugs"].Enabled
	}

	if got := flip(); got {
		t.Error("toggle from true must flip to false")
	}
	if got := flip(); !got {
		t.Error("toggle from false must flip to true")
	}
	if got := flip("--off"); got {
		t.Error("--off must force false")
	}
	if got := flip("--off"); got {
		t.Error("--off must be idempotent")
	}
	if got := flip("--on"); !got {
		t.Error("--on must force true")
	}
	if err := runOperatorCmd(t, operatorWatchToggleCmd(), "linear-bugs", "--on", "--off"); err == nil {
		t.Error("--on --off together must error")
	}
	if err := runOperatorCmd(t, operatorWatchToggleCmd(), "ghost"); err == nil {
		t.Error("toggle of unknown watch must error")
	}
}

func TestOperatorWatchUpdate(t *testing.T) {
	path := withOperatorState(t, "")
	if err := addWatch(t, "--stop-stage", "intake", "--instructions", "old"); err != nil {
		t.Fatalf("watch add: %v", err)
	}

	err := runOperatorCmd(t, operatorWatchUpdateCmd(), "linear-bugs",
		"--target-repo", "/home/u/bar", "--instructions", "also limit to 2 concurrent agents",
		"--query", `{"status":["Todo"]}`)
	if err != nil {
		t.Fatalf("watch update: %v", err)
	}
	w := readWatches(t, path)["linear-bugs"]
	if w.TargetRepo != "/home/u/bar" || w.Instructions != "also limit to 2 concurrent agents" {
		t.Errorf("update fields wrong: %+v", w)
	}
	if _, ok := w.Query["status"]; !ok || len(w.Query) != 1 {
		t.Errorf("query not replaced: %v", w.Query)
	}
	if w.StopStage == nil || *w.StopStage != "intake" {
		t.Errorf("untouched stop_stage changed: %v", w.StopStage)
	}

	// Empty --stop-stage clears to null.
	if err := runOperatorCmd(t, operatorWatchUpdateCmd(), "linear-bugs", "--stop-stage", ""); err != nil {
		t.Fatalf("clear stop-stage: %v", err)
	}
	if w := readWatches(t, path)["linear-bugs"]; w.StopStage != nil {
		t.Errorf("stop_stage = %v, want cleared to null", *w.StopStage)
	}

	if err := runOperatorCmd(t, operatorWatchUpdateCmd(), "ghost", "--instructions", "x"); err == nil {
		t.Error("update of unknown watch must error")
	}
	if err := runOperatorCmd(t, operatorWatchUpdateCmd(), "linear-bugs", "--query", `nope`); err == nil {
		t.Error("invalid --query on update must error")
	}
}

func TestOperatorWatchChecked_SetAndClearError(t *testing.T) {
	path := withOperatorState(t, "")
	if err := addWatch(t); err != nil {
		t.Fatalf("watch add: %v", err)
	}

	if err := runOperatorCmd(t, operatorWatchCheckedCmd(), "linear-bugs", "--error", "linear API 502"); err != nil {
		t.Fatalf("watch checked --error: %v", err)
	}
	w := readWatches(t, path)["linear-bugs"]
	if w.LastChecked == nil {
		t.Error("last_checked not set")
	}
	if w.LastError == nil || *w.LastError != "linear API 502" {
		t.Errorf("last_error = %v", w.LastError)
	}

	// Absent --error clears last_error.
	if err := runOperatorCmd(t, operatorWatchCheckedCmd(), "linear-bugs"); err != nil {
		t.Fatalf("watch checked: %v", err)
	}
	w = readWatches(t, path)["linear-bugs"]
	if w.LastError != nil {
		t.Errorf("last_error = %v, want cleared to null", *w.LastError)
	}
	if w.LastChecked == nil {
		t.Error("last_checked must stay set")
	}

	if err := runOperatorCmd(t, operatorWatchCheckedCmd(), "ghost"); err == nil {
		t.Error("checked on unknown watch must error")
	}
}

func TestOperatorWatchSeen_IdempotentAndCapped(t *testing.T) {
	// Seed a watch whose known list already holds the 200-cap's worth of items.
	var sb strings.Builder
	sb.WriteString("watches:\n  linear-bugs:\n    enabled: true\n    source: linear\n    target_repo: /home/u/foo\n    known:\n")
	for i := 1; i <= knownCap; i++ {
		fmt.Fprintf(&sb, "      - DEV-%03d\n", i)
	}
	sb.WriteString("    completed: []\n    last_checked: null\n    last_error: null\n")
	path := withOperatorState(t, sb.String())

	// Idempotent: re-seeing an existing item changes nothing.
	if err := runOperatorCmd(t, operatorWatchSeenCmd(), "linear-bugs", "DEV-001"); err != nil {
		t.Fatalf("seen existing: %v", err)
	}
	w := readWatches(t, path)["linear-bugs"]
	if len(w.Known) != knownCap || w.Known[0] != "DEV-001" {
		t.Errorf("idempotent seen mutated known: len=%d first=%v", len(w.Known), w.Known[0])
	}

	// Cap: the 201st item prunes the oldest, newest lands last.
	if err := runOperatorCmd(t, operatorWatchSeenCmd(), "linear-bugs", "DEV-NEW"); err != nil {
		t.Fatalf("seen new: %v", err)
	}
	w = readWatches(t, path)["linear-bugs"]
	if len(w.Known) != knownCap {
		t.Errorf("known len = %d, want %d", len(w.Known), knownCap)
	}
	if w.Known[0] != "DEV-002" {
		t.Errorf("oldest not pruned first: known[0] = %v", w.Known[0])
	}
	if w.Known[len(w.Known)-1] != "DEV-NEW" {
		t.Errorf("newest item not last: %v", w.Known[len(w.Known)-1])
	}

	if err := runOperatorCmd(t, operatorWatchSeenCmd(), "ghost", "X-1"); err == nil {
		t.Error("seen on unknown watch must error")
	}
}

func TestOperatorWatchComplete(t *testing.T) {
	path := withOperatorState(t, "")
	if err := addWatch(t); err != nil {
		t.Fatalf("watch add: %v", err)
	}
	for _, id := range []string{"DEV-1", "DEV-2"} {
		if err := runOperatorCmd(t, operatorWatchSeenCmd(), "linear-bugs", id); err != nil {
			t.Fatalf("seen %s: %v", id, err)
		}
	}

	// Known item moves to completed.
	if err := runOperatorCmd(t, operatorWatchCompleteCmd(), "linear-bugs", "DEV-1"); err != nil {
		t.Fatalf("complete DEV-1: %v", err)
	}
	w := readWatches(t, path)["linear-bugs"]
	if len(w.Known) != 1 || w.Known[0] != "DEV-2" {
		t.Errorf("known = %v, want [DEV-2]", w.Known)
	}
	if len(w.Completed) != 1 || w.Completed[0] != "DEV-1" {
		t.Errorf("completed = %v, want [DEV-1]", w.Completed)
	}

	// Late completion (absent from known) is still recorded, never duplicated.
	if err := runOperatorCmd(t, operatorWatchCompleteCmd(), "linear-bugs", "DEV-9"); err != nil {
		t.Fatalf("complete DEV-9: %v", err)
	}
	if err := runOperatorCmd(t, operatorWatchCompleteCmd(), "linear-bugs", "DEV-9"); err != nil {
		t.Fatalf("re-complete DEV-9: %v", err)
	}
	w = readWatches(t, path)["linear-bugs"]
	if len(w.Completed) != 2 {
		t.Errorf("completed = %v, want [DEV-1 DEV-9] (no duplicate)", w.Completed)
	}

	if err := runOperatorCmd(t, operatorWatchCompleteCmd(), "ghost", "X-1"); err == nil {
		t.Error("complete on unknown watch must error")
	}
}
