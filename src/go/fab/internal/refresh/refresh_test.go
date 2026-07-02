package refresh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// statusFixture is a minimal .status.yaml matching the hook_test fixture shape.
// %s is the change folder name.
const statusFixture = `id: abcd
name: %s
created: "2026-07-02T12:00:00Z"
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
last_updated: "2026-07-02T12:00:00Z"
`

// intakeFix is an intake whose prose infers "fix" and whose Assumptions table
// yields a full 5.0 confidence score (five Certain rows, all dimensions 100).
const intakeFix = `# Intake: Fix the broken widget

This is a fix for a bug.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | D1 | R1 | S:100 R:100 A:100 D:100 |
| 2 | Certain | D2 | R2 | S:100 R:100 A:100 D:100 |
| 3 | Certain | D3 | R3 | S:100 R:100 A:100 D:100 |
| 4 | Certain | D4 | R4 | S:100 R:100 A:100 D:100 |
| 5 | Certain | D5 | R5 | S:100 R:100 A:100 D:100 |
`

const planBoth = `# Plan

## Tasks

- [ ] T001 first
- [x] T002 second
- [ ] T003 third

## Acceptance

- [x] A-001 done thing
- [ ] A-002 open thing
`

// setupChange writes a change dir with a .status.yaml fixture under a temp
// fabRoot. Returns (fabRoot, changeDir).
func setupChange(t *testing.T) (fabRoot, changeDir string) {
	t.Helper()
	root := t.TempDir()
	fabRoot = filepath.Join(root, "fab")
	folder := "260702-abcd-refresh-test"
	changeDir = filepath.Join(fabRoot, "changes", folder)
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	statusYAML := strings.Replace(statusFixture, "%s", folder, 1)
	if err := os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(statusYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	return fabRoot, changeDir
}

func loadStatus(t *testing.T, changeDir string) *sf.StatusFile {
	t.Helper()
	st, err := sf.Load(filepath.Join(changeDir, ".status.yaml"))
	if err != nil {
		t.Fatalf("load status: %v", err)
	}
	return st
}

func writeArtifact(t *testing.T, changeDir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(changeDir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRefresh_IntakeOnly(t *testing.T) {
	fabRoot, changeDir := setupChange(t)
	writeArtifact(t, changeDir, "intake.md", intakeFix)

	st := loadStatus(t, changeDir)
	dirty, err := Refresh(fabRoot, changeDir, st)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if !dirty {
		t.Error("expected dirty=true when intake.md recomputes change_type + confidence")
	}
	if st.ChangeType != "fix" {
		t.Errorf("change_type = %q, want fix", st.ChangeType)
	}
	if st.Confidence.Score != 5.0 {
		t.Errorf("confidence.score = %v, want 5.0", st.Confidence.Score)
	}
	if st.Confidence.Certain != 5 {
		t.Errorf("confidence.certain = %d, want 5", st.Confidence.Certain)
	}
	// plan fields must be untouched (no plan.md on disk).
	if st.Plan.Generated {
		t.Error("plan.generated should stay false with no plan.md")
	}
}

func TestRefresh_PlanOnly(t *testing.T) {
	fabRoot, changeDir := setupChange(t)
	writeArtifact(t, changeDir, "plan.md", planBoth)

	st := loadStatus(t, changeDir)
	dirty, err := Refresh(fabRoot, changeDir, st)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if !dirty {
		t.Error("expected dirty=true when plan.md recomputes counts")
	}
	if !st.Plan.Generated {
		t.Error("plan.generated not set")
	}
	if st.Plan.TaskCount != 3 {
		t.Errorf("task_count = %d, want 3", st.Plan.TaskCount)
	}
	if st.Plan.AcceptanceCount != 2 {
		t.Errorf("acceptance_count = %d, want 2", st.Plan.AcceptanceCount)
	}
	if st.Plan.AcceptanceCompleted != 1 {
		t.Errorf("acceptance_completed = %d, want 1", st.Plan.AcceptanceCompleted)
	}
	// intake-derived fields untouched (no intake.md): change_type stays feat.
	if st.ChangeType != "feat" {
		t.Errorf("change_type = %q, want feat (unchanged, no intake.md)", st.ChangeType)
	}
}

func TestRefresh_BothPresent(t *testing.T) {
	fabRoot, changeDir := setupChange(t)
	writeArtifact(t, changeDir, "intake.md", intakeFix)
	writeArtifact(t, changeDir, "plan.md", planBoth)

	st := loadStatus(t, changeDir)
	dirty, err := Refresh(fabRoot, changeDir, st)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if !dirty {
		t.Error("expected dirty=true")
	}
	if st.ChangeType != "fix" {
		t.Errorf("change_type = %q, want fix", st.ChangeType)
	}
	if st.Confidence.Score != 5.0 {
		t.Errorf("confidence.score = %v, want 5.0", st.Confidence.Score)
	}
	if st.Plan.TaskCount != 3 || st.Plan.AcceptanceCount != 2 || st.Plan.AcceptanceCompleted != 1 {
		t.Errorf("plan counts = %d/%d/%d, want 3/2/1", st.Plan.TaskCount, st.Plan.AcceptanceCount, st.Plan.AcceptanceCompleted)
	}
}

func TestRefresh_MissingArtifactsNoOp(t *testing.T) {
	fabRoot, changeDir := setupChange(t)
	// No intake.md, no plan.md on disk.
	st := loadStatus(t, changeDir)
	dirty, err := Refresh(fabRoot, changeDir, st)
	if err != nil {
		t.Fatalf("Refresh must not error on missing artifacts: %v", err)
	}
	if dirty {
		t.Error("expected dirty=false when no artifacts are present")
	}
	if st.ChangeType != "feat" {
		t.Errorf("change_type = %q, want feat (unchanged)", st.ChangeType)
	}
}

func TestRefresh_RespectsExplicitChangeType(t *testing.T) {
	fabRoot, changeDir := setupChange(t)
	// intakeFix prose would infer "fix", but the type is explicitly feat.
	writeArtifact(t, changeDir, "intake.md", intakeFix)

	st := loadStatus(t, changeDir)
	st.ChangeType = "feat"
	st.ChangeTypeSource = sf.SourceExplicit

	if _, err := Refresh(fabRoot, changeDir, st); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if st.ChangeType != "feat" {
		t.Errorf("change_type = %q, want feat (explicit must survive refresh)", st.ChangeType)
	}
	if st.ChangeTypeSource != sf.SourceExplicit {
		t.Errorf("change_type_source = %q, want explicit", st.ChangeTypeSource)
	}
	// Confidence still recomputes even under the explicit guard.
	if st.Confidence.Score != 5.0 {
		t.Errorf("confidence.score = %v, want 5.0 (recomputed regardless of explicit type)", st.Confidence.Score)
	}
}

func TestRefresh_InferredChangeTypeReinfers(t *testing.T) {
	fabRoot, changeDir := setupChange(t)
	writeArtifact(t, changeDir, "intake.md", intakeFix)

	st := loadStatus(t, changeDir)
	// Default source (absent/inferred): re-inference allowed.
	if _, err := Refresh(fabRoot, changeDir, st); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if st.ChangeType != "fix" {
		t.Errorf("change_type = %q, want fix (inferred source re-infers)", st.ChangeType)
	}
}

func TestRefresh_MissingSectionLeavesFieldsUntouched(t *testing.T) {
	fabRoot, changeDir := setupChange(t)
	// plan.md with ## Tasks but NO ## Acceptance.
	planTasksOnly := `# Plan

## Tasks

- [ ] T001 only task
`
	writeArtifact(t, changeDir, "plan.md", planTasksOnly)

	st := loadStatus(t, changeDir)
	// Seed a non-zero acceptance count that MUST be preserved (no ## Acceptance).
	st.Plan.AcceptanceCount = 7
	st.Plan.AcceptanceCompleted = 3

	if _, err := Refresh(fabRoot, changeDir, st); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if !st.Plan.Generated {
		t.Error("plan.generated should be set (## Tasks present)")
	}
	if st.Plan.TaskCount != 1 {
		t.Errorf("task_count = %d, want 1", st.Plan.TaskCount)
	}
	// Acceptance fields must be left untouched — no ## Acceptance section.
	if st.Plan.AcceptanceCount != 7 {
		t.Errorf("acceptance_count = %d, want 7 (untouched — no ## Acceptance)", st.Plan.AcceptanceCount)
	}
	if st.Plan.AcceptanceCompleted != 3 {
		t.Errorf("acceptance_completed = %d, want 3 (untouched — no ## Acceptance)", st.Plan.AcceptanceCompleted)
	}
}

// TestRefresh_Idempotent verifies a second refresh against unchanged artifacts
// yields the same field values AND reports dirty=false (so the caller's
// dirty-guarded Save writes nothing new and does not bump last_updated). The
// first run against a fresh fixture is expected to report dirty=true; the
// second run must be a value- AND dirty-idempotent no-op (A-005).
func TestRefresh_Idempotent(t *testing.T) {
	fabRoot, changeDir := setupChange(t)
	writeArtifact(t, changeDir, "intake.md", intakeFix)
	writeArtifact(t, changeDir, "plan.md", planBoth)

	st := loadStatus(t, changeDir)
	firstDirty, err := Refresh(fabRoot, changeDir, st)
	if err != nil {
		t.Fatalf("first Refresh: %v", err)
	}
	if !firstDirty {
		t.Error("expected dirty=true on the first refresh against a fresh fixture")
	}
	first := st.Plan
	firstType := st.ChangeType
	firstScore := st.Confidence.Score

	secondDirty, err := Refresh(fabRoot, changeDir, st)
	if err != nil {
		t.Fatalf("second Refresh: %v", err)
	}
	if secondDirty {
		t.Error("expected dirty=false on a second refresh against unchanged artifacts (no spurious Save / last_updated bump)")
	}
	if st.Plan != first {
		t.Errorf("plan changed on second refresh: %+v != %+v", st.Plan, first)
	}
	if st.ChangeType != firstType {
		t.Errorf("change_type changed on second refresh: %q != %q", st.ChangeType, firstType)
	}
	if st.Confidence.Score != firstScore {
		t.Errorf("confidence.score changed on second refresh: %v != %v", st.Confidence.Score, firstScore)
	}
}
