package statusfile

import (
	"os"
	"path/filepath"
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
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
checklist:
  generated: false
  path: checklist.md
  completed: 0
  total: 0
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
	if sf.GetProgress("tasks") != "pending" {
		t.Errorf("expected tasks pending, got '%s'", sf.GetProgress("tasks"))
	}
	if sf.Confidence.Score != 4.7 {
		t.Errorf("expected score 4.7, got %f", sf.Confidence.Score)
	}
	if sf.Checklist.Generated != false {
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

	if len(pm) != 8 {
		t.Errorf("expected 8 stages, got %d", len(pm))
	}

	// Verify pipeline order
	expected := []string{"intake", "spec", "tasks", "apply", "review", "hydrate", "ship", "review-pr"}
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
	if StageNumber("review-pr") != 8 {
		t.Error("review-pr should be 8")
	}
	if StageNumber("bogus") != 0 {
		t.Error("bogus should be 0")
	}
}

func TestNextStage(t *testing.T) {
	if NextStage("intake") != "spec" {
		t.Error("after intake should be spec")
	}
	if NextStage("hydrate") != "ship" {
		t.Error("after hydrate should be ship")
	}
	if NextStage("review-pr") != "" {
		t.Error("after review-pr should be empty")
	}
}
