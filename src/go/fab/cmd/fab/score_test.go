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

	hookTestEnv(t, root, map[string]string{})
	return root
}

func TestScoreCmd_CheckGateFail_ReturnsError(t *testing.T) {
	// 3 confident decisions on a fix change: base = 5.0 - 0.3*3 = 4.1,
	// cover = 3/5 → score 2.5 < 3.0 threshold → gate fail.
	setupScoreCmdFixture(t,
		"| 1 | Confident | D1 | R1 | |",
		"| 2 | Confident | D2 | R2 | |",
		"| 3 | Confident | D3 | R3 | |",
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
	// 5 certain decisions on a fix change: score 5.0 >= 3.0 → gate pass.
	setupScoreCmdFixture(t,
		"| 1 | Certain | D1 | R1 | |",
		"| 2 | Certain | D2 | R2 | |",
		"| 3 | Certain | D3 | R3 | |",
		"| 4 | Certain | D4 | R4 | |",
		"| 5 | Certain | D5 | R5 | |",
	)

	cmd := scoreCmd()
	cmd.SetArgs([]string{"--check-gate", "abcd"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected nil error (exit 0) on gate pass, got %v", err)
	}
}
