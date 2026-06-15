package status

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lockfile"
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

// --- AllowedStates enforcement on transitions (k4ge) ---

func TestLookupTransition_RejectsForbiddenTargets(t *testing.T) {
	cases := []struct {
		event, stage, from string
	}{
		{"advance", "ship", "active"},      // would write ready — forbidden for ship
		{"advance", "review-pr", "active"}, // would write ready — forbidden for review-pr
		{"skip", "intake", "active"},       // would write skipped — forbidden for intake
		{"skip", "intake", "pending"},
	}
	for _, c := range cases {
		_, err := lookupTransition(c.event, c.stage, c.from)
		if err == nil {
			t.Errorf("%s %s from %s: expected rejection, got nil", c.event, c.stage, c.from)
			continue
		}
		if !strings.Contains(err.Error(), "not allowed for this stage") {
			t.Errorf("%s %s: error should mention forbidden target, got: %v", c.event, c.stage, err)
		}
	}
}

func TestLookupTransition_AllowedTargetsUnchanged(t *testing.T) {
	cases := []struct {
		event, stage, from, want string
	}{
		{"start", "ship", "pending", "active"},
		{"start", "review", "failed", "active"},
		{"advance", "apply", "active", "ready"},
		{"advance", "intake", "active", "ready"},
		{"finish", "ship", "active", "done"},
		{"finish", "review-pr", "active", "done"},
		{"skip", "apply", "pending", "skipped"},
		{"fail", "review", "active", "failed"},
		{"fail", "review-pr", "active", "failed"},
		{"reset", "apply", "done", "active"},
	}
	for _, c := range cases {
		got, err := lookupTransition(c.event, c.stage, c.from)
		if err != nil {
			t.Errorf("%s %s from %s: unexpected error: %v", c.event, c.stage, c.from, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s %s from %s = %q, want %q", c.event, c.stage, c.from, got, c.want)
		}
	}
}

func TestAdvanceShip_RejectedAndFileUntouched(t *testing.T) {
	statusFile, path := loadFixture(t)
	for _, s := range []string{"intake", "apply", "review", "hydrate"} {
		statusFile.SetProgress(s, "done")
	}
	statusFile.SetProgress("ship", "active")
	if err := statusFile.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	err := Advance(statusFile, path, "ship", "test")
	if err == nil {
		t.Fatal("expected advance ship to be rejected")
	}
	if !strings.Contains(err.Error(), "not allowed for this stage") {
		t.Errorf("error should mention forbidden target, got: %v", err)
	}

	reloaded, loadErr := sf.Load(path)
	if loadErr != nil {
		t.Fatalf("reload: %v", loadErr)
	}
	if got := reloaded.GetProgress("ship"); got != "active" {
		t.Errorf("ship state on disk = %q, want active (unmodified)", got)
	}
	if err := Validate(reloaded); err != nil {
		t.Errorf("status file should still validate, got: %v", err)
	}
}

func TestSkipIntake_Rejected(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := filepath.Dir(path)

	err := Skip(statusFile, path, dir, "intake", "test")
	if err == nil {
		t.Fatal("expected skip intake to be rejected")
	}
	if !strings.Contains(err.Error(), "not allowed for this stage") {
		t.Errorf("error should mention forbidden target, got: %v", err)
	}
	if got := statusFile.GetProgress("intake"); got != "active" {
		t.Errorf("intake state = %q, want active (unmodified)", got)
	}
}

// --- stage_metrics iterations survive the fail+reset cascade (k4ge) ---

func TestResetCascade_PreservesReviewIterations(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := filepath.Dir(path)

	// intake → apply → review (review iterations = 1)
	if err := Finish(statusFile, path, dir, "intake", "test"); err != nil {
		t.Fatalf("Finish intake: %v", err)
	}
	if err := Finish(statusFile, path, dir, "apply", "test"); err != nil {
		t.Fatalf("Finish apply: %v", err)
	}
	if sm := statusFile.StageMetrics["review"]; sm == nil || sm.Iterations != 1 {
		t.Fatalf("review iterations after first activation = %v, want 1", statusFile.StageMetrics["review"])
	}

	// The rework choreography: fail review, then reset apply (cascades review → pending).
	if err := Fail(statusFile, path, dir, "review", "test", ""); err != nil {
		t.Fatalf("Fail review: %v", err)
	}
	if err := Reset(statusFile, path, dir, "apply", "test", "", ""); err != nil {
		t.Fatalf("Reset apply: %v", err)
	}

	sm := statusFile.StageMetrics["review"]
	if sm == nil {
		t.Fatal("stage_metrics.review was deleted by the reset cascade — iterations counter lost")
	}
	if sm.Iterations != 1 {
		t.Errorf("review iterations after fail+reset = %d, want 1 (preserved)", sm.Iterations)
	}
	if sm.StartedAt != "" || sm.Driver != "" || sm.CompletedAt != "" {
		t.Errorf("timing fields should be cleared, got %+v", sm)
	}

	// Preservation must survive the save/load round trip.
	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if rm := reloaded.StageMetrics["review"]; rm == nil || rm.Iterations != 1 {
		t.Fatalf("reloaded review iterations = %v, want 1", reloaded.StageMetrics["review"])
	}

	// Re-finishing apply re-activates review as a re-entry: iterations = 2.
	if err := Finish(statusFile, path, dir, "apply", "test"); err != nil {
		t.Fatalf("re-Finish apply: %v", err)
	}
	if sm := statusFile.StageMetrics["review"]; sm == nil || sm.Iterations != 2 {
		t.Fatalf("review iterations after rework re-entry = %v, want 2", statusFile.StageMetrics["review"])
	}
}

func TestResetCascade_DeletesZeroIterationEntries(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := filepath.Dir(path)

	statusFile.SetProgress("intake", "done")
	statusFile.SetProgress("apply", "done")
	// An entry that was never activated (no iterations) must not linger.
	statusFile.StageMetrics["hydrate"] = &sf.StageMetric{CompletedAt: "2026-03-10T12:00:00Z"}

	if err := Reset(statusFile, path, dir, "apply", "test", "", ""); err != nil {
		t.Fatalf("Reset apply: %v", err)
	}
	if _, ok := statusFile.StageMetrics["hydrate"]; ok {
		t.Error("zero-iteration stage_metrics entry should be deleted by the cascade")
	}
}

// displayStageFixture builds a .status.yaml in a temp dir with the given
// progress block and loads it, following the loadFixture YAML+Load pattern
// (StatusFile.Progress is a raw-backed yaml.Node, not directly constructible).
func displayStageFixture(t *testing.T, progress string) *sf.StatusFile {
	t.Helper()
	yaml := `id: dkn3
name: 260612-dkn3-pane-map-display-state
created: "2026-06-12T12:00:00Z"
created_by: test-user
change_type: feat
issues: []
progress:
` + progress + `plan:
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
last_updated: "2026-06-12T12:00:00Z"
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	statusFile, err := sf.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return statusFile
}

// TestDisplayStage_FailedTier verifies the failed tier inserted between the
// active and ready tiers: precedence is active → failed → ready → last
// done/skipped → first pending.
func TestDisplayStage_FailedTier(t *testing.T) {
	t.Run("review failed with nothing active returns review/failed", func(t *testing.T) {
		statusFile := displayStageFixture(t, `  intake: done
  apply: done
  review: failed
  hydrate: pending
  ship: pending
  review-pr: pending
`)
		stage, state := DisplayStage(statusFile)
		if stage != "review" || state != "failed" {
			t.Errorf("DisplayStage = (%q, %q), want (\"review\", \"failed\")", stage, state)
		}
	})

	t.Run("failed plus a later active stage returns the active stage", func(t *testing.T) {
		statusFile := displayStageFixture(t, `  intake: done
  apply: done
  review: failed
  hydrate: active
  ship: pending
  review-pr: pending
`)
		stage, state := DisplayStage(statusFile)
		if stage != "hydrate" || state != "active" {
			t.Errorf("DisplayStage = (%q, %q), want (\"hydrate\", \"active\")", stage, state)
		}
	})

	t.Run("failed outranks ready", func(t *testing.T) {
		statusFile := displayStageFixture(t, `  intake: done
  apply: ready
  review: failed
  hydrate: pending
  ship: pending
  review-pr: pending
`)
		stage, state := DisplayStage(statusFile)
		if stage != "review" || state != "failed" {
			t.Errorf("DisplayStage = (%q, %q), want (\"review\", \"failed\")", stage, state)
		}
	})

	t.Run("no failed stage preserves pre-change derivation", func(t *testing.T) {
		// ready wins over done when nothing is active or failed.
		statusFile := displayStageFixture(t, `  intake: done
  apply: ready
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
`)
		stage, state := DisplayStage(statusFile)
		if stage != "apply" || state != "ready" {
			t.Errorf("DisplayStage = (%q, %q), want (\"apply\", \"ready\")", stage, state)
		}

		// Last done wins when nothing is active, failed, or ready.
		statusFile = displayStageFixture(t, `  intake: done
  apply: done
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
`)
		stage, state = DisplayStage(statusFile)
		if stage != "apply" || state != "done" {
			t.Errorf("DisplayStage = (%q, %q), want (\"apply\", \"done\")", stage, state)
		}
	})
}

// --- Non-saving Apply variants (mz4q F02): mutate in memory only,
// validate-before-mutate preserved. ---

func TestApplyAcceptance_MutatesWithoutSaving(t *testing.T) {
	statusFile, path := loadFixture(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := ApplyAcceptance(statusFile, "task_count", "7"); err != nil {
		t.Fatalf("ApplyAcceptance: %v", err)
	}
	if statusFile.Plan.TaskCount != 7 {
		t.Errorf("TaskCount = %d, want 7", statusFile.Plan.TaskCount)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("ApplyAcceptance must not write to disk — caller owns persistence")
	}
}

func TestApplyAcceptance_ValidatesBeforeMutate(t *testing.T) {
	statusFile, _ := loadFixture(t)
	prior := statusFile.Plan.TaskCount

	if err := ApplyAcceptance(statusFile, "task_count", "not-a-number"); err == nil {
		t.Fatal("expected error for invalid value")
	}
	if statusFile.Plan.TaskCount != prior {
		t.Error("invalid value must not mutate the in-memory StatusFile")
	}
}

func TestApplyChangeType_MutatesWithoutSaving(t *testing.T) {
	statusFile, path := loadFixture(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := ApplyChangeType(statusFile, "fix"); err != nil {
		t.Fatalf("ApplyChangeType: %v", err)
	}
	if statusFile.ChangeType != "fix" {
		t.Errorf("ChangeType = %q, want fix", statusFile.ChangeType)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("ApplyChangeType must not write to disk — caller owns persistence")
	}
}

func TestApplyChangeType_ValidatesBeforeMutate(t *testing.T) {
	statusFile, _ := loadFixture(t)

	if err := ApplyChangeType(statusFile, "bogus"); err == nil {
		t.Fatal("expected error for invalid change type")
	}
	if statusFile.ChangeType != "feat" {
		t.Errorf("invalid type must not mutate ChangeType, got %q", statusFile.ChangeType)
	}
}

// TestSetChangeType_MarksExplicit covers jznd (2/a): a human running
// set-change-type marks the source explicit and persists it, so the
// PostToolUse intake-write hook stops re-inferring/overwriting the type.
func TestSetChangeType_MarksExplicit(t *testing.T) {
	statusFile, path := loadFixture(t)

	if err := SetChangeType(statusFile, path, "feat"); err != nil {
		t.Fatalf("SetChangeType: %v", err)
	}
	if statusFile.ChangeTypeSource != sf.SourceExplicit {
		t.Errorf("ChangeTypeSource = %q, want %q", statusFile.ChangeTypeSource, sf.SourceExplicit)
	}

	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.ChangeType != "feat" || reloaded.ChangeTypeSource != sf.SourceExplicit {
		t.Errorf("explicit marker not persisted: type=%q source=%q", reloaded.ChangeType, reloaded.ChangeTypeSource)
	}
}

// TestApplyChangeType_DoesNotMarkExplicit covers jznd (2/a): the hook's
// inference path (ApplyChangeType) must NOT set the explicit marker — only
// set-change-type does — so re-inference stays allowed on inferred changes.
func TestApplyChangeType_DoesNotMarkExplicit(t *testing.T) {
	statusFile, _ := loadFixture(t)

	if err := ApplyChangeType(statusFile, "fix"); err != nil {
		t.Fatalf("ApplyChangeType: %v", err)
	}
	if statusFile.ChangeTypeSource == sf.SourceExplicit {
		t.Error("ApplyChangeType must not mark source explicit (only set-change-type does)")
	}
}

func TestApplyConfidence_MutatesWithoutSaving(t *testing.T) {
	statusFile, path := loadFixture(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	ApplyConfidence(statusFile, 3, 2, 1, 0, 3.7)
	if statusFile.Confidence.Score != 3.7 || statusFile.Confidence.Certain != 3 {
		t.Errorf("Confidence not applied: %+v", statusFile.Confidence)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("ApplyConfidence must not write to disk — caller owns persistence")
	}
}

// --- F07 (mz4q): transitions on sparse/malformed progress maps either
// persist (missing stage key created) or fail loudly (malformed shape) —
// never an exit-0 dropped write. ---

func TestStart_CreatesMissingStageKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	sparse := `id: sp4r
name: 260310-sp4r-sparse-change
created: "2026-03-10T12:00:00Z"
progress:
  intake: done
last_updated: "2026-03-10T12:00:00Z"
`
	if err := os.WriteFile(path, []byte(sparse), 0o644); err != nil {
		t.Fatal(err)
	}
	statusFile, err := sf.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// apply is absent from the progress map: GetProgress defaults it to
	// pending, the transition validates, and the key must be created and
	// persisted (previously a silent no-op while stage_metrics and
	// .history.jsonl did persist — an inconsistent state).
	if err := Start(statusFile, path, dir, "apply", "test", "", ""); err != nil {
		t.Fatalf("Start: %v", err)
	}

	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := reloaded.GetProgress("apply"); got != "active" {
		t.Errorf("apply = %q after Start on sparse file, want active (dropped write)", got)
	}
}

func TestStart_MalformedProgressErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	malformed := `id: sp4r
name: 260310-sp4r-no-progress
created: "2026-03-10T12:00:00Z"
last_updated: "2026-03-10T12:00:00Z"
`
	if err := os.WriteFile(path, []byte(malformed), 0o644); err != nil {
		t.Fatal(err)
	}
	statusFile, err := sf.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	err = Start(statusFile, path, dir, "apply", "test", "", "")
	if err == nil {
		t.Fatal("expected malformed-shape error when progress: is absent")
	}
	if !strings.Contains(err.Error(), "progress is missing or not a mapping") {
		t.Errorf("unexpected error text: %v", err)
	}

	// Nothing persisted — no half-consistent state.
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("malformed-shape failure must not persist anything")
	}
}

// --- Lock-serialized load-mutate-save (mz4q F03): concurrent writers running
// the full cycle under lockfile.WithLock (the composition used by
// withStatusLock in cmd/fab and by the artifact-write hook) cannot lose each
// other's updates to the shared document. ---

func TestConcurrentLockedMutatorsNoLostUpdates(t *testing.T) {
	statusFile, path := loadFixture(t)
	_ = statusFile

	const writers = 10
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			err := lockfile.WithLock(path, func() error {
				st, err := sf.Load(path)
				if err != nil {
					return err
				}
				return AddPR(st, path, fmt.Sprintf("https://github.com/o/r/pull/%d", n))
			})
			if err != nil {
				t.Errorf("locked AddPR cycle %d failed: %v", n, err)
			}
		}(i)
	}
	wg.Wait()

	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.PRs) != writers {
		t.Errorf("lost updates: expected %d PRs, got %d (%v)", writers, len(reloaded.PRs), reloaded.PRs)
	}
}
