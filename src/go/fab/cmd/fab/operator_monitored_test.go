package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// withOperatorState redirects operator state-file I/O to a temp file seeded
// with initial (empty string = missing file) and returns its path.
func withOperatorState(t *testing.T, initial string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "operator-state.yaml")
	if initial != "" {
		if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
			t.Fatalf("seed state file: %v", err)
		}
	}
	operatorStatePathOverride = path
	t.Cleanup(func() { operatorStatePathOverride = "" })
	return path
}

// runOperatorCmd executes a command with output silenced and returns its error.
func runOperatorCmd(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	return cmd.Execute()
}

// readStateFile parses the state file back into a raw map.
func readStateFile(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parse state file: %v", err)
	}
	return data
}

// readMonitored decodes the monitored section into typed entries.
func readMonitored(t *testing.T, path string) map[string]monitoredEntry {
	t.Helper()
	m := map[string]monitoredEntry{}
	if err := operatorSection(readStateFile(t, path), "monitored", &m); err != nil {
		t.Fatalf("decode monitored: %v", err)
	}
	return m
}

func enrollArgs(id string, extra ...string) []string {
	return append([]string{
		id, "--pane", "%3", "--repo", "/home/u/foo",
		"--session", "work", "--branch", "260819-" + id + "-x",
	}, extra...)
}

func TestOperatorEnroll_CreatesEntryAndBranchMap(t *testing.T) {
	path := withOperatorState(t, "")

	err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("r3m7", "--stage", "apply")...)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}

	m := readMonitored(t, path)
	e, ok := m["r3m7"]
	if !ok {
		t.Fatalf("monitored.r3m7 missing: %v", m)
	}
	if e.Pane != "%3" || e.Repo != "/home/u/foo" || e.Session != "work" || e.Branch != "260819-r3m7-x" {
		t.Errorf("entry identity fields wrong: %+v", e)
	}
	if e.Stage != "apply" {
		t.Errorf("stage = %q, want apply", e.Stage)
	}
	if e.StopStage != nil {
		t.Errorf("stop_stage = %v, want nil default", *e.StopStage)
	}
	if e.SpawnedBy != nil {
		t.Errorf("spawned_by = %v, want nil default", *e.SpawnedBy)
	}
	if e.DependsOn == nil || len(e.DependsOn) != 0 {
		t.Errorf("depends_on = %v, want empty (non-nil) list", e.DependsOn)
	}
	for _, ts := range []string{e.EnrolledAt, e.LastTransition} {
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			t.Errorf("timestamp %q not RFC3339: %v", ts, err)
		}
	}

	bm := map[string]branchMapEntry{}
	if err := operatorSection(readStateFile(t, path), "branch_map", &bm); err != nil {
		t.Fatalf("decode branch_map: %v", err)
	}
	if bm["r3m7"] != (branchMapEntry{Branch: "260819-r3m7-x", Repo: "/home/u/foo"}) {
		t.Errorf("branch_map.r3m7 = %+v", bm["r3m7"])
	}
}

func TestOperatorEnroll_OptionalFlagsAndDependsOn(t *testing.T) {
	path := withOperatorState(t, "")

	err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("r3m7",
		"--stage", "apply", "--agent", "active", "--stop-stage", "review",
		"--spawned-by", "linear-bugs", "--depends-on", "ab12,cd34")...)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	e := readMonitored(t, path)["r3m7"]
	if e.Agent != "active" || *e.StopStage != "review" || *e.SpawnedBy != "linear-bugs" {
		t.Errorf("optional flags not stored: %+v", e)
	}
	if len(e.DependsOn) != 2 || e.DependsOn[0] != "ab12" || e.DependsOn[1] != "cd34" {
		t.Errorf("depends_on = %v", e.DependsOn)
	}
}

func TestOperatorEnroll_ReenrollReplacesWholesale(t *testing.T) {
	path := withOperatorState(t, "")

	if err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("r3m7", "--stage", "apply", "--depends-on", "ab12")...); err != nil {
		t.Fatalf("first enroll: %v", err)
	}
	first := readMonitored(t, path)["r3m7"]
	if err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("r3m7", "--stage", "review")...); err != nil {
		t.Fatalf("re-enroll: %v", err)
	}
	e := readMonitored(t, path)["r3m7"]
	if e.Stage != "review" {
		t.Errorf("stage = %q, want review", e.Stage)
	}
	if len(e.DependsOn) != 0 {
		t.Errorf("depends_on survived wholesale replace: %v", e.DependsOn)
	}
	if e.EnrolledAt < first.EnrolledAt {
		t.Errorf("re-enroll did not refresh enrolled_at: %q vs %q", e.EnrolledAt, first.EnrolledAt)
	}
}

func TestOperatorEnroll_Errors(t *testing.T) {
	withOperatorState(t, "")

	if err := runOperatorCmd(t, operatorEnrollCmd(), "r3m7", "--repo", "/x", "--session", "s", "--branch", "b"); err == nil {
		t.Error("missing --pane must error")
	}
	if err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("r3m7", "--stage", "bogus")...); err == nil {
		t.Error("invalid --stage must error")
	}
	if err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("r3m7", "--stop-stage", "bogus")...); err == nil {
		t.Error("invalid --stop-stage must error")
	}
}

func TestOperatorUpdate_LastTransitionOnlyOnStageChange(t *testing.T) {
	path := withOperatorState(t, `monitored:
  r3m7:
    pane: "%3"
    repo: /home/u/foo
    session: work
    stage: apply
    agent: active
    stop_stage: null
    spawned_by: null
    depends_on: []
    branch: 260819-r3m7-x
    enrolled_at: "2026-01-01T00:00:00Z"
    last_transition: "2026-01-01T00:00:00Z"
`)

	if err := runOperatorCmd(t, operatorUpdateCmd(), "r3m7", "--agent", "idle"); err != nil {
		t.Fatalf("update --agent: %v", err)
	}
	e := readMonitored(t, path)["r3m7"]
	if e.Agent != "idle" {
		t.Errorf("agent = %q, want idle", e.Agent)
	}
	if e.LastTransition != "2026-01-01T00:00:00Z" {
		t.Errorf("last_transition moved on non-stage update: %q", e.LastTransition)
	}

	if err := runOperatorCmd(t, operatorUpdateCmd(), "r3m7", "--stage", "apply"); err != nil {
		t.Fatalf("update --stage (same): %v", err)
	}
	if e := readMonitored(t, path)["r3m7"]; e.LastTransition != "2026-01-01T00:00:00Z" {
		t.Errorf("last_transition moved on unchanged stage: %q", e.LastTransition)
	}

	if err := runOperatorCmd(t, operatorUpdateCmd(), "r3m7", "--stage", "review"); err != nil {
		t.Fatalf("update --stage review: %v", err)
	}
	e = readMonitored(t, path)["r3m7"]
	if e.Stage != "review" {
		t.Errorf("stage = %q, want review", e.Stage)
	}
	if e.LastTransition == "2026-01-01T00:00:00Z" {
		t.Error("last_transition not touched on stage change")
	}
}

func TestOperatorUpdate_Errors(t *testing.T) {
	path := withOperatorState(t, "")
	if err := runOperatorCmd(t, operatorUpdateCmd(), "ghost", "--agent", "idle"); err == nil {
		t.Error("unknown change-id must error")
	}
	if err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("r3m7")...); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if err := runOperatorCmd(t, operatorUpdateCmd(), "r3m7", "--stop-stage", "bogus"); err == nil {
		t.Error("invalid --stop-stage must error")
	}
	// stop-stage "" clears to null.
	if err := runOperatorCmd(t, operatorUpdateCmd(), "r3m7", "--stop-stage", ""); err != nil {
		t.Fatalf("clear stop-stage: %v", err)
	}
	if e := readMonitored(t, path)["r3m7"]; e.StopStage != nil {
		t.Errorf("stop_stage = %v, want cleared to null", *e.StopStage)
	}
}

func TestOperatorRemove_RetainsBranchMap(t *testing.T) {
	path := withOperatorState(t, "")
	if err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs("r3m7")...); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if err := runOperatorCmd(t, operatorRemoveCmd(), "r3m7"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, ok := readMonitored(t, path)["r3m7"]; ok {
		t.Error("monitored.r3m7 still present after remove")
	}
	bm := map[string]branchMapEntry{}
	if err := operatorSection(readStateFile(t, path), "branch_map", &bm); err != nil {
		t.Fatalf("decode branch_map: %v", err)
	}
	if _, ok := bm["r3m7"]; !ok {
		t.Error("branch_map.r3m7 must be retained after remove")
	}
	if err := runOperatorCmd(t, operatorRemoveCmd(), "r3m7"); err == nil {
		t.Error("removing an unknown id must error")
	}
}

// TestOperatorMutation_TolerantReadTypedWrite pins the IO posture: unknown
// TOP-LEVEL keys survive any mutation; an invented field inside an OWNED
// section is dropped when that section is rewritten.
func TestOperatorMutation_TolerantReadTypedWrite(t *testing.T) {
	path := withOperatorState(t, `tick_count: 47
custom_top: keep-me
watches:
  linear-bugs:
    enabled: true
    source: linear
    target_repo: /home/u/foo
    stop_stage: null
    known: []
    completed: []
    last_checked: null
    last_error: null
    note: trigger armed
`)

	if err := runOperatorCmd(t, operatorWatchToggleCmd(), "linear-bugs"); err != nil {
		t.Fatalf("watch toggle: %v", err)
	}
	data := readStateFile(t, path)
	if data["custom_top"] != "keep-me" {
		t.Errorf("unknown top-level key not preserved: %v", data)
	}
	if data["tick_count"] != 47 {
		t.Errorf("tick_count not preserved: %v", data["tick_count"])
	}
	watches, ok := data["watches"].(map[string]interface{})
	if !ok {
		t.Fatalf("watches section missing: %v", data)
	}
	entry, ok := watches["linear-bugs"].(map[string]interface{})
	if !ok {
		t.Fatalf("watch entry missing: %v", watches)
	}
	if _, drifted := entry["note"]; drifted {
		t.Errorf("invented in-section field survived its section's mutation: %v", entry)
	}
	if entry["enabled"] != false {
		t.Errorf("enabled = %v, want false (toggled)", entry["enabled"])
	}
}
