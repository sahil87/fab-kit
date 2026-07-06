package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const scoreCmdStatusYAML = `id: abcd
name: 260310-abcd-my-change
created: "2026-03-10T12:00:00Z"
created_by: test-user
change_type: fix
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

// setupScoreCmdFixture creates a repo root with a fab change whose intake.md
// carries the given Assumptions rows, and chdirs into it so resolve.FabRoot()
// finds it. Returns the repo root.
func setupScoreCmdFixture(t *testing.T, assumptionRows ...string) string {
	t.Helper()
	root := t.TempDir()
	changeDir := filepath.Join(root, "fab", "changes", "260310-abcd-my-change")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(scoreCmdStatusYAML), 0o644)

	var b strings.Builder
	b.WriteString("# Intake\n\n## Assumptions\n\n")
	b.WriteString("| # | Grade | Decision | Rationale | Scores |\n")
	b.WriteString("|---|-------|----------|-----------|--------|\n")
	for _, row := range assumptionRows {
		b.WriteString(row + "\n")
	}
	os.WriteFile(filepath.Join(changeDir, "intake.md"), []byte(b.String()), 0o644)

	chdirTestEnv(t, root, map[string]string{})
	return root
}

func TestScoreCmd_CheckGateFail_ReturnsError(t *testing.T) {
	// Demerit model, flat 3.0 gate. Weak dimensions: composite for
	// S:30 R:30 A:30 D:30 = 6+9+9+6 = 30 (Tentative) → penalty
	// 0.50 + (50-30)/50*2.50 = 1.5 each. Σ penalty = 4.5 → score = 0.5 < 3.0
	// → gate fail.
	setupScoreCmdFixture(t,
		"| 1 | Tentative | D1 | R1 | S:30 R:30 A:30 D:30 |",
		"| 2 | Tentative | D2 | R2 | S:30 R:30 A:30 D:30 |",
		"| 3 | Tentative | D3 | R3 | S:30 R:30 A:30 D:30 |",
	)

	cmd := scoreCmd()
	cmd.SetArgs([]string{"--check-gate", "abcd"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected non-nil error (non-zero exit) on gate fail, got nil")
	}
	if !strings.Contains(err.Error(), "intake gate failed") {
		t.Errorf("error = %q, want it to mention the failed intake gate", err.Error())
	}
}

func TestScoreCmd_CheckGatePass_ExitsZero(t *testing.T) {
	// Demerit model, flat 3.0 gate. Strong dimensions: composite for
	// S:95 R:95 A:95 D:95 = 95.0 (Certain) → penalty 0 each. Σ penalty = 0 →
	// score = clamp(5.0 - 0, 0, 5) = 5.0 >= 3.0 → gate pass.
	setupScoreCmdFixture(t,
		"| 1 | Certain | D1 | R1 | S:95 R:95 A:95 D:95 |",
		"| 2 | Certain | D2 | R2 | S:95 R:95 A:95 D:95 |",
		"| 3 | Certain | D3 | R3 | S:95 R:95 A:95 D:95 |",
	)

	cmd := scoreCmd()
	cmd.SetArgs([]string{"--check-gate", "abcd"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected nil error (exit 0) on gate pass, got %v", err)
	}
}
