package statusfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testYAML = `id: te1t
name: 260305-test-1-sample-change
created: "2026-03-05T12:00:00+05:30"
created_by: test-user
change_type: feat
issues: []
progress:
  intake: done
  spec: active
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
plan:
  generated: false
  task_count: 0
  acceptance_count: 0
  acceptance_completed: 0
confidence:
  certain: 3
  confident: 1
  tentative: 0
  unresolved: 0
  score: 4.7
stage_metrics:
  intake: {started_at: "2026-03-05T12:00:00+05:30", driver: fab-new, iterations: 1, completed_at: "2026-03-05T12:01:00+05:30"}
  spec: {started_at: "2026-03-05T12:01:00+05:30", driver: fab-continue, iterations: 1}
prs: []
last_updated: "2026-03-05T12:01:00+05:30"
`

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	os.WriteFile(path, []byte(testYAML), 0644)

	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if sf.ID != "te1t" {
		t.Errorf("expected id 'te1t', got '%s'", sf.ID)
	}
	if sf.Name != "260305-test-1-sample-change" {
		t.Errorf("expected name '260305-test-1-sample-change', got '%s'", sf.Name)
	}
	if sf.ChangeType != "feat" {
		t.Errorf("expected change_type 'feat', got '%s'", sf.ChangeType)
	}
	if sf.GetProgress("intake") != "done" {
		t.Errorf("expected intake done, got '%s'", sf.GetProgress("intake"))
	}
	if sf.GetProgress("spec") != "active" {
		t.Errorf("expected spec active, got '%s'", sf.GetProgress("spec"))
	}
	if sf.GetProgress("apply") != "pending" {
		t.Errorf("expected apply pending, got '%s'", sf.GetProgress("apply"))
	}
	if sf.Confidence.Score != 4.7 {
		t.Errorf("expected score 4.7, got %f", sf.Confidence.Score)
	}
	if sf.Plan.Generated != false {
		t.Error("expected generated false")
	}

	// Test stage metrics
	sm, ok := sf.StageMetrics["intake"]
	if !ok {
		t.Fatal("expected intake stage metrics")
	}
	if sm.Iterations != 1 {
		t.Errorf("expected iterations 1, got %d", sm.Iterations)
	}
	if sm.Driver != "fab-new" {
		t.Errorf("expected driver fab-new, got '%s'", sm.Driver)
	}

	// Test SetProgress
	sf.SetProgress("spec", "done")
	if sf.GetProgress("spec") != "done" {
		t.Errorf("expected spec done after set, got '%s'", sf.GetProgress("spec"))
	}

	// Test Save (round-trip)
	outPath := filepath.Join(dir, ".status-out.yaml")
	if err := sf.Save(outPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify
	sf2, err := Load(outPath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if sf2.ID != sf.ID {
		t.Errorf("round-trip id mismatch: %s vs %s", sf2.ID, sf.ID)
	}
	if sf2.Name != sf.Name {
		t.Errorf("round-trip name mismatch: %s vs %s", sf2.Name, sf.Name)
	}
	if sf2.GetProgress("spec") != "done" {
		t.Errorf("round-trip spec state mismatch: got '%s'", sf2.GetProgress("spec"))
	}
	if sf2.Confidence.Score != 4.7 {
		t.Errorf("round-trip score mismatch: %f", sf2.Confidence.Score)
	}
}

func TestGetProgressMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	os.WriteFile(path, []byte(testYAML), 0644)

	sf, _ := Load(path)
	pm := sf.GetProgressMap()

	if len(pm) != 7 {
		t.Errorf("expected 7 stages, got %d", len(pm))
	}

	// Verify pipeline order
	expected := []string{"intake", "spec", "apply", "review", "hydrate", "ship", "review-pr"}
	for i, ss := range pm {
		if ss.Stage != expected[i] {
			t.Errorf("stage %d: expected '%s', got '%s'", i, expected[i], ss.Stage)
		}
	}
}

func TestStageNumber(t *testing.T) {
	if StageNumber("intake") != 1 {
		t.Error("intake should be 1")
	}
	if StageNumber("review-pr") != 7 {
		t.Error("review-pr should be 7")
	}
	if StageNumber("bogus") != 0 {
		t.Error("bogus should be 0")
	}
}

func TestPlanRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	os.WriteFile(path, []byte(testYAML), 0644)

	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Mutate plan fields and save.
	sf.Plan.Generated = true
	sf.Plan.TaskCount = 5
	sf.Plan.AcceptanceCount = 9
	sf.Plan.AcceptanceCompleted = 3

	outPath := filepath.Join(dir, ".status-plan.yaml")
	if err := sf.Save(outPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	sf2, err := Load(outPath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if !sf2.Plan.Generated {
		t.Error("plan.generated round-trip lost")
	}
	if sf2.Plan.TaskCount != 5 {
		t.Errorf("plan.task_count round-trip: got %d, want 5", sf2.Plan.TaskCount)
	}
	if sf2.Plan.AcceptanceCount != 9 {
		t.Errorf("plan.acceptance_count round-trip: got %d, want 9", sf2.Plan.AcceptanceCount)
	}
	if sf2.Plan.AcceptanceCompleted != 3 {
		t.Errorf("plan.acceptance_completed round-trip: got %d, want 3", sf2.Plan.AcceptanceCompleted)
	}
}

// TestLegacyChecklistFileSavesPlanBlock guards a regression where saving a
// pre-1.9.0 .status.yaml (which has a `checklist:` block but no `plan:` block)
// silently dropped Plan struct mutations. The fix upgrades the raw schema on
// Load so syncToRaw has a `plan:` node to write into.
func TestLegacyChecklistFileSavesPlanBlock(t *testing.T) {
	const legacyYAML = `id: lgcy
name: 260423-legacy-fixture
created: "2026-04-23T05:02:32Z"
created_by: test-user
change_type: fix
issues: []
progress:
    intake: done
    spec: done
    tasks: done
    apply: active
    review: pending
    hydrate: pending
    ship: pending
    review-pr: pending
checklist:
    generated: true
    path: checklist.md
    completed: 5
    total: 9
confidence:
    certain: 1
    confident: 0
    tentative: 0
    unresolved: 0
    score: 4.7
stage_metrics: {}
prs: []
last_updated: "2026-04-23T05:02:32Z"
`

	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(legacyYAML), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	// 1. Load the legacy file.
	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Migration mapping should populate Plan from the checklist: block.
	if !sf.Plan.Generated {
		t.Error("expected Plan.Generated=true (migrated from checklist.generated)")
	}
	if sf.Plan.AcceptanceCompleted != 5 {
		t.Errorf("expected Plan.AcceptanceCompleted=5 (migrated from checklist.completed), got %d", sf.Plan.AcceptanceCompleted)
	}
	if sf.Plan.AcceptanceCount != 9 {
		t.Errorf("expected Plan.AcceptanceCount=9 (migrated from checklist.total), got %d", sf.Plan.AcceptanceCount)
	}

	// 2. Mutate plan fields and save.
	sf.Plan.TaskCount = 7
	sf.Plan.AcceptanceCount = 12

	if err := sf.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 3. Re-load — assert Plan fields persisted.
	sf2, err := Load(path)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if sf2.Plan.TaskCount != 7 {
		t.Errorf("Plan.TaskCount round-trip: got %d, want 7", sf2.Plan.TaskCount)
	}
	if sf2.Plan.AcceptanceCount != 12 {
		t.Errorf("Plan.AcceptanceCount round-trip: got %d, want 12", sf2.Plan.AcceptanceCount)
	}
	if sf2.Plan.AcceptanceCompleted != 5 {
		t.Errorf("Plan.AcceptanceCompleted round-trip: got %d, want 5", sf2.Plan.AcceptanceCompleted)
	}
	if !sf2.Plan.Generated {
		t.Error("Plan.Generated round-trip lost")
	}

	// 4. Inspect raw bytes — plan: block must be present, legacy checklist: must be absent.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	rawStr := string(raw)
	if !strings.Contains(rawStr, "plan:") {
		t.Errorf("saved file missing `plan:` block. content:\n%s", rawStr)
	}
	if strings.Contains(rawStr, "checklist:") {
		t.Errorf("saved file still contains legacy `checklist:` block. content:\n%s", rawStr)
	}
	if !strings.Contains(rawStr, "task_count: 7") {
		t.Errorf("saved file missing task_count: 7. content:\n%s", rawStr)
	}
}

func TestNextStage(t *testing.T) {
	if NextStage("intake") != "spec" {
		t.Error("after intake should be spec")
	}
	if NextStage("spec") != "apply" {
		t.Error("after spec should be apply")
	}
	if NextStage("hydrate") != "ship" {
		t.Error("after hydrate should be ship")
	}
	if NextStage("review-pr") != "" {
		t.Error("after review-pr should be empty")
	}
}

// TestPlanAndChecklistCoexistDropsChecklist guards against a partial-migration
// state where both `plan:` (new, authoritative) and `checklist:` (legacy,
// stale) coexist in the same .status.yaml. Load MUST drop the legacy
// `checklist:` block so it does not survive subsequent Save() calls.
func TestPlanAndChecklistCoexistDropsChecklist(t *testing.T) {
	const mixedYAML = `id: mxd1
name: 260423-mixed-fixture
created: "2026-04-23T05:02:32Z"
created_by: test-user
change_type: fix
issues: []
progress:
    intake: done
    spec: done
    apply: active
    review: pending
    hydrate: pending
    ship: pending
    review-pr: pending
plan:
    generated: true
    task_count: 4
    acceptance_count: 10
    acceptance_completed: 3
checklist:
    generated: true
    path: checklist.md
    completed: 99
    total: 99
confidence:
    certain: 1
    confident: 0
    tentative: 0
    unresolved: 0
    score: 4.7
stage_metrics: {}
prs: []
last_updated: "2026-04-23T05:02:32Z"
`

	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(mixedYAML), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	// 1. Load the mixed file.
	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Plan block is authoritative — its values must be preserved verbatim,
	// not overwritten by the stale checklist block.
	if !sf.Plan.Generated {
		t.Error("expected Plan.Generated=true (from plan: block, authoritative)")
	}
	if sf.Plan.TaskCount != 4 {
		t.Errorf("expected Plan.TaskCount=4 (from plan:), got %d", sf.Plan.TaskCount)
	}
	if sf.Plan.AcceptanceCount != 10 {
		t.Errorf("expected Plan.AcceptanceCount=10 (from plan:), got %d", sf.Plan.AcceptanceCount)
	}
	if sf.Plan.AcceptanceCompleted != 3 {
		t.Errorf("expected Plan.AcceptanceCompleted=3 (from plan:), got %d", sf.Plan.AcceptanceCompleted)
	}

	// 2. Save and re-read raw bytes — the legacy checklist: key MUST NOT
	//    survive.
	if err := sf.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	rawStr := string(raw)
	if !strings.Contains(rawStr, "plan:") {
		t.Errorf("saved file missing `plan:` block. content:\n%s", rawStr)
	}
	if strings.Contains(rawStr, "checklist:") {
		t.Errorf("saved file still contains stale `checklist:` block (should have been dropped on Load). content:\n%s", rawStr)
	}
	if !strings.Contains(rawStr, "task_count: 4") {
		t.Errorf("saved file lost authoritative plan values. content:\n%s", rawStr)
	}
}
