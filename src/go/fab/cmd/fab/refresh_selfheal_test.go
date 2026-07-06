package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// selfHealStatusYAML is a change fixture whose derived fields are STALE relative
// to the artifacts written below (change_type feat vs. an intake that infers
// fix; zeroed plan counts vs. a plan.md with 3 tasks / 2 acceptance). %s is the
// stage that starts `active` (so a forward transition is valid).
const selfHealStatusYAML = `id: abcd
name: 260702-abcd-refresh-heal
created: "2026-07-02T12:00:00Z"
created_by: test-user
change_type: feat
issues: []
progress:
  intake: %s
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

const healIntakeFix = `# Intake: Fix the broken widget

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

const healPlanBoth = `# Plan

## Tasks

- [ ] T001 first
- [x] T002 second
- [ ] T003 third

## Acceptance

- [x] A-001 done thing
- [ ] A-002 open thing
`

// setupSelfHealFixture creates a repo root with a stale change fixture plus the
// intake.md/plan.md artifacts, and chdirs into it so resolve.FabRoot() finds
// it. intakeStage is the state of the `intake` stage in the fixture. Returns
// the change dir.
func setupSelfHealFixture(t *testing.T, intakeStage string) string {
	t.Helper()
	root := t.TempDir()
	changeDir := filepath.Join(root, "fab", "changes", "260702-abcd-refresh-heal")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	statusYAML := strings.Replace(selfHealStatusYAML, "%s", intakeStage, 1)
	if err := os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(statusYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "intake.md"), []byte(healIntakeFix), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "plan.md"), []byte(healPlanBoth), 0o644); err != nil {
		t.Fatal(err)
	}
	// project files so preflight passes its init check.
	projDir := filepath.Join(root, "fab", "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "config.yaml"), []byte("project:\n  name: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "constitution.md"), []byte("# Constitution\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdirTestEnv(t, root, map[string]string{})
	return changeDir
}

func reloadHealStatus(t *testing.T, changeDir string) *sf.StatusFile {
	t.Helper()
	st, err := sf.Load(filepath.Join(changeDir, ".status.yaml"))
	if err != nil {
		t.Fatalf("reload status: %v", err)
	}
	return st
}

func assertHealed(t *testing.T, st *sf.StatusFile) {
	t.Helper()
	if st.ChangeType != "fix" {
		t.Errorf("change_type = %q, want fix (healed from stale feat)", st.ChangeType)
	}
	if st.Confidence.Score != 5.0 {
		t.Errorf("confidence.score = %v, want 5.0 (healed from stale 0.0)", st.Confidence.Score)
	}
	if !st.Plan.Generated {
		t.Error("plan.generated should be healed to true")
	}
	if st.Plan.TaskCount != 3 {
		t.Errorf("task_count = %d, want 3 (healed)", st.Plan.TaskCount)
	}
	if st.Plan.AcceptanceCount != 2 || st.Plan.AcceptanceCompleted != 1 {
		t.Errorf("acceptance = %d/%d, want 1/2 (healed)", st.Plan.AcceptanceCompleted, st.Plan.AcceptanceCount)
	}
}

func TestRefreshCmd_HealsStaleStatus(t *testing.T) {
	changeDir := setupSelfHealFixture(t, "active")

	cmd := statusRefreshCmd()
	cmd.SetArgs([]string{"abcd"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fab status refresh: %v", err)
	}
	assertHealed(t, reloadHealStatus(t, changeDir))
}

func TestAdvance_SelfHeals(t *testing.T) {
	changeDir := setupSelfHealFixture(t, "active")

	cmd := statusAdvanceCmd()
	cmd.SetArgs([]string{"abcd", "intake"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fab status advance: %v", err)
	}
	st := reloadHealStatus(t, changeDir)
	assertHealed(t, st)
	// The transition also happened in the same Save.
	if got := st.GetProgress("intake"); got != "ready" {
		t.Errorf("intake state = %q, want ready (advance transition persisted alongside the heal)", got)
	}
}

func TestFinish_SelfHeals(t *testing.T) {
	changeDir := setupSelfHealFixture(t, "active")

	cmd := statusFinishCmd()
	cmd.SetArgs([]string{"abcd", "intake"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fab status finish: %v", err)
	}
	st := reloadHealStatus(t, changeDir)
	assertHealed(t, st)
	if got := st.GetProgress("intake"); got != "done" {
		t.Errorf("intake state = %q, want done (finish transition persisted alongside the heal)", got)
	}
}

func TestPreflight_SelfHeals(t *testing.T) {
	changeDir := setupSelfHealFixture(t, "active")

	cmd := preflightCmd()
	cmd.SetArgs([]string{"abcd"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fab preflight: %v", err)
	}
	assertHealed(t, reloadHealStatus(t, changeDir))
}

// TestStart_DoesNotSelfHeal confirms non-forward/non-orient seams do NOT
// refresh: start moves a pending stage to active without recomputing
// artifact-derived fields (R6). We start `apply` (pending → active) and assert
// the stale change_type/plan counts are untouched.
func TestStart_DoesNotSelfHeal(t *testing.T) {
	changeDir := setupSelfHealFixture(t, "active")

	cmd := statusStartCmd()
	cmd.SetArgs([]string{"abcd", "apply"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fab status start: %v", err)
	}
	st := reloadHealStatus(t, changeDir)
	if st.ChangeType != "feat" {
		t.Errorf("change_type = %q, want feat (start must NOT self-heal)", st.ChangeType)
	}
	if st.Plan.Generated || st.Plan.TaskCount != 0 {
		t.Errorf("plan fields changed (%v/%d); start must NOT self-heal", st.Plan.Generated, st.Plan.TaskCount)
	}
	if got := st.GetProgress("apply"); got != "active" {
		t.Errorf("apply state = %q, want active (start transition still happened)", got)
	}
}

// TestReset_DoesNotSelfHeal confirms reset does not refresh either (R6). We
// reset `intake` from a done state and assert the stale fields are untouched.
func TestReset_DoesNotSelfHeal(t *testing.T) {
	// intake must be resettable: reset's From-set is {done, ready, skipped}.
	changeDir := setupSelfHealFixture(t, "done")

	cmd := statusResetCmd()
	cmd.SetArgs([]string{"abcd", "intake"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fab status reset: %v", err)
	}
	st := reloadHealStatus(t, changeDir)
	if st.ChangeType != "feat" {
		t.Errorf("change_type = %q, want feat (reset must NOT self-heal)", st.ChangeType)
	}
	if st.Plan.Generated || st.Plan.TaskCount != 0 {
		t.Errorf("plan fields changed (%v/%d); reset must NOT self-heal", st.Plan.Generated, st.Plan.TaskCount)
	}
}
