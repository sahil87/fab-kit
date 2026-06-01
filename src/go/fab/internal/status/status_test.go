package status

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

const setAcceptanceFixture = `id: abcd
name: 260310-abcd-my-change
created: "2026-03-10T12:00:00Z"
created_by: test-user
change_type: feat
issues: []
progress:
  intake: active
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
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 0.0
stage_metrics: {}
prs: []
last_updated: "2026-03-10T12:00:00Z"
`

func loadFixture(t *testing.T) (*sf.StatusFile, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(setAcceptanceFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	statusFile, err := sf.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return statusFile, path
}

func TestSetAcceptance_Generated(t *testing.T) {
	statusFile, path := loadFixture(t)
	priorUpdated := statusFile.LastUpdated

	if err := SetAcceptance(statusFile, path, "generated", "true"); err != nil {
		t.Fatalf("SetAcceptance: %v", err)
	}
	if !statusFile.Plan.Generated {
		t.Error("Plan.Generated not updated")
	}
	if statusFile.LastUpdated == priorUpdated {
		t.Error("last_updated should be refreshed")
	}

	// Round-trip
	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if !reloaded.Plan.Generated {
		t.Error("Plan.Generated not persisted")
	}
}

func TestSetAcceptance_TaskCount(t *testing.T) {
	statusFile, path := loadFixture(t)
	if err := SetAcceptance(statusFile, path, "task_count", "12"); err != nil {
		t.Fatalf("SetAcceptance: %v", err)
	}
	if statusFile.Plan.TaskCount != 12 {
		t.Errorf("TaskCount = %d, want 12", statusFile.Plan.TaskCount)
	}
}

func TestSetAcceptance_AcceptanceCount(t *testing.T) {
	statusFile, path := loadFixture(t)
	if err := SetAcceptance(statusFile, path, "acceptance_count", "8"); err != nil {
		t.Fatalf("SetAcceptance: %v", err)
	}
	if statusFile.Plan.AcceptanceCount != 8 {
		t.Errorf("AcceptanceCount = %d, want 8", statusFile.Plan.AcceptanceCount)
	}
}

func TestSetAcceptance_AcceptanceCompleted(t *testing.T) {
	statusFile, path := loadFixture(t)
	if err := SetAcceptance(statusFile, path, "acceptance_completed", "5"); err != nil {
		t.Fatalf("SetAcceptance: %v", err)
	}
	if statusFile.Plan.AcceptanceCompleted != 5 {
		t.Errorf("AcceptanceCompleted = %d, want 5", statusFile.Plan.AcceptanceCompleted)
	}
}

func TestSetAcceptance_InvalidField(t *testing.T) {
	statusFile, path := loadFixture(t)
	err := SetAcceptance(statusFile, path, "unknown", "1")
	if err == nil {
		t.Fatal("expected error for invalid field")
	}
	if !strings.Contains(err.Error(), "Invalid plan field 'unknown'") {
		t.Errorf("error should mention invalid plan field, got: %v", err)
	}
	expectedFields := []string{"generated", "task_count", "acceptance_count", "acceptance_completed"}
	for _, field := range expectedFields {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("error should list valid field %q, got: %v", field, err)
		}
	}
}

func TestSetAcceptance_InvalidGeneratedValue(t *testing.T) {
	statusFile, path := loadFixture(t)
	err := SetAcceptance(statusFile, path, "generated", "maybe")
	if err == nil {
		t.Fatal("expected error for invalid generated value")
	}
	if !strings.Contains(err.Error(), "expected true/false") {
		t.Errorf("error should mention true/false, got: %v", err)
	}
}

func TestSetAcceptance_InvalidIntValue(t *testing.T) {
	statusFile, path := loadFixture(t)
	err := SetAcceptance(statusFile, path, "task_count", "abc")
	if err == nil {
		t.Fatal("expected error for non-numeric task_count")
	}
}

func TestSetChecklistRemovedError_HasPointer(t *testing.T) {
	err := SetChecklistRemovedError()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "\"set-checklist\" is now \"set-acceptance\"") {
		t.Errorf("error should contain pointer to set-acceptance, got: %v", err)
	}
}

func TestValidateStage_TasksReturnsStrictError(t *testing.T) {
	err := validateStage("finish", "tasks")
	if err == nil {
		t.Fatal("expected strict-error for tasks stage")
	}
	msg := err.Error()
	if !strings.Contains(msg, "\"tasks\" stage was removed") {
		t.Errorf("error should contain '\"tasks\" stage was removed', got: %v", err)
	}
	if !strings.Contains(msg, "fab status finish <change> apply") {
		t.Errorf("error should suggest finish <change> apply, got: %v", err)
	}
	if !strings.Contains(msg, "plan.md is now generated at apply entry") {
		t.Errorf("error should mention plan.md generation, got: %v", err)
	}
}

func TestValidateStage_BogusReturnsGenericError(t *testing.T) {
	err := validateStage("finish", "bogus")
	if err == nil {
		t.Fatal("expected error for bogus stage")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Invalid stage 'bogus'") {
		t.Errorf("error should be generic for non-tasks unknown stage, got: %v", err)
	}
}

func TestValidateStage_ValidStagesAccepted(t *testing.T) {
	for _, stage := range []string{"intake", "apply", "review", "hydrate", "ship", "review-pr"} {
		if err := validateStage("finish", stage); err != nil {
			t.Errorf("validateStage(\"finish\", %q) returned error: %v", stage, err)
		}
	}
}

func TestValidateStage_SpecReturnsStrictError(t *testing.T) {
	for _, event := range []string{"start", "advance", "finish", "reset", "skip", "fail"} {
		err := validateStage(event, "spec")
		if err == nil {
			t.Fatalf("expected strict-error from validateStage(%q, \"spec\")", event)
		}
		if !strings.Contains(err.Error(), "\"spec\" stage was removed") {
			t.Errorf("validateStage(%q, \"spec\") error should mention spec removed, got: %v", event, err)
		}
	}
}

func TestStartFinishOnSpecReturnsStrictError(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := filepath.Dir(path)

	err := Start(statusFile, path, dir, "spec", "test", "", "")
	if err == nil {
		t.Fatal("expected strict-error from Start on spec stage")
	}
	if !strings.Contains(err.Error(), "\"spec\" stage was removed") {
		t.Errorf("Start error should mention spec removed, got: %v", err)
	}

	err = Finish(statusFile, path, dir, "spec", "test")
	if err == nil {
		t.Fatal("expected strict-error from Finish on spec stage")
	}
	if !strings.Contains(err.Error(), "\"spec\" stage was removed") {
		t.Errorf("Finish error should mention spec removed, got: %v", err)
	}
}

func TestStartFinishOnTasksReturnsStrictError(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := filepath.Dir(path)

	err := Start(statusFile, path, dir, "tasks", "test", "", "")
	if err == nil {
		t.Fatal("expected strict-error from Start on tasks stage")
	}
	if !strings.Contains(err.Error(), "\"tasks\" stage was removed") {
		t.Errorf("Start error should mention tasks removed, got: %v", err)
	}

	err = Finish(statusFile, path, dir, "tasks", "test")
	if err == nil {
		t.Fatal("expected strict-error from Finish on tasks stage")
	}
	if !strings.Contains(err.Error(), "\"tasks\" stage was removed") {
		t.Errorf("Finish error should mention tasks removed, got: %v", err)
	}
}

// TestIntakeFinishAutoActivatesApply verifies the six-stage transition: with
// spec removed, apply is the stage immediately after intake, so finishing
// intake auto-activates apply.
func TestIntakeFinishAutoActivatesApply(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := filepath.Dir(path)

	if err := Finish(statusFile, path, dir, "intake", "test"); err != nil {
		t.Fatalf("Finish intake: %v", err)
	}

	if statusFile.GetProgress("intake") != "done" {
		t.Errorf("intake should be done, got %q", statusFile.GetProgress("intake"))
	}
	if statusFile.GetProgress("apply") != "active" {
		t.Errorf("apply should auto-activate to active, got %q", statusFile.GetProgress("apply"))
	}
	// Neither progress.spec nor progress.tasks should exist in the canonical map.
	for _, ss := range statusFile.GetProgressMap() {
		if ss.Stage == "spec" || ss.Stage == "tasks" {
			t.Errorf("progress.%s should not exist, but found state %q", ss.Stage, ss.State)
		}
	}
}
