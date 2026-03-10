package score

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const statusTemplate = `id: abcd
name: 260310-abcd-my-change
created: "2026-03-10T12:00:00Z"
created_by: test-user
change_type: %s
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
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 0.0
stage_metrics: {}
prs: []
last_updated: "2026-03-10T12:00:00Z"
`

// setupScoreFixture creates a fab structure with a change directory and
// writes the given spec.md content. Returns fabRoot.
func setupScoreFixture(t *testing.T, changeType, specContent string) string {
	t.Helper()
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)

	// Write .status.yaml
	statusYAML := strings.Replace(statusTemplate, "%s", changeType, 1)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(statusYAML), 0644)

	// Write spec.md
	os.WriteFile(filepath.Join(changeDir, "spec.md"), []byte(specContent), 0644)

	// Create project config — required by status.SetConfidence/SetConfidenceFuzzy
	// which reads project config to locate the status file during YAML writes
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0755)
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte("project:\n  name: test\n"), 0644)

	return fabRoot
}

func specWithAssumptions(rows ...string) string {
	var b strings.Builder
	b.WriteString("# Spec\n\n## Assumptions\n\n")
	b.WriteString("| # | Grade | Decision | Rationale | Scores |\n")
	b.WriteString("|---|-------|----------|-----------|--------|\n")
	for _, row := range rows {
		b.WriteString(row + "\n")
	}
	b.WriteString("\n## Next Section\n")
	return b.String()
}

func assertApproxEqual(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.05 {
		t.Errorf("%s = %.2f, want %.2f", name, got, want)
	}
}

func TestCompute_AllCertain(t *testing.T) {
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | |",
		"| 2 | Certain | D2 | R2 | |",
		"| 3 | Certain | D3 | R3 | |",
		"| 4 | Certain | D4 | R4 | |",
		"| 5 | Certain | D5 | R5 | |",
		"| 6 | Certain | D6 | R6 | |",
		"| 7 | Certain | D7 | R7 | |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	// 7 certain, 0 penalty, total=7, expectedMin for feat spec=7, cover=1.0
	// score = (5.0 - 0*7) * 1.0 = 5.0
	assertApproxEqual(t, "Score", result.Score, 5.0)
	if result.Certain != 7 {
		t.Errorf("Certain = %d, want 7", result.Certain)
	}
	if result.Confident != 0 {
		t.Errorf("Confident = %d, want 0", result.Confident)
	}
}

func TestCompute_ConfidentPenalties(t *testing.T) {
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | |",
		"| 2 | Certain | D2 | R2 | |",
		"| 3 | Certain | D3 | R3 | |",
		"| 4 | Confident | D4 | R4 | |",
		"| 5 | Confident | D5 | R5 | |",
		"| 6 | Certain | D6 | R6 | |",
		"| 7 | Certain | D7 | R7 | |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	// 5 certain, 2 confident, total=7, expectedMin=7, cover=1.0
	// base = 5.0 - 0.0*5 - 0.3*2 = 5.0 - 0.6 = 4.4
	// score = 4.4 * 1.0 = 4.4
	assertApproxEqual(t, "Score", result.Score, 4.4)
}

func TestCompute_UnresolvedZero(t *testing.T) {
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | |",
		"| 2 | Certain | D2 | R2 | |",
		"| 3 | Unresolved | D3 | R3 | |",
		"| 4 | Certain | D4 | R4 | |",
		"| 5 | Certain | D5 | R5 | |",
		"| 6 | Certain | D6 | R6 | |",
		"| 7 | Certain | D7 | R7 | |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	if result.Score != 0.0 {
		t.Errorf("Score = %.1f, want 0.0 (unresolved present)", result.Score)
	}
	if result.Unresolved != 1 {
		t.Errorf("Unresolved = %d, want 1", result.Unresolved)
	}
}

func TestCompute_CoverFactor(t *testing.T) {
	// Only 3 decisions for a feat change (expectedMin=7 for spec)
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | |",
		"| 2 | Certain | D2 | R2 | |",
		"| 3 | Certain | D3 | R3 | |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	// base = 5.0, cover = 3/7 ~= 0.4286
	// score = 5.0 * (3/7) ~= 2.1
	assertApproxEqual(t, "Score", result.Score, 2.1)
}

func TestCompute_DimensionParsing(t *testing.T) {
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:80 R:90 A:70 D:85 |",
		"| 2 | Certain | D2 | R2 | S:90 R:80 A:80 D:75 |",
		"| 3 | Certain | D3 | R3 | S:70 R:70 A:90 D:90 |",
		"| 4 | Certain | D4 | R4 | S:80 R:80 A:80 D:80 |",
		"| 5 | Certain | D5 | R5 | S:80 R:80 A:80 D:80 |",
		"| 6 | Certain | D6 | R6 | S:80 R:80 A:80 D:80 |",
		"| 7 | Certain | D7 | R7 | S:80 R:80 A:80 D:80 |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	if !result.HasFuzzy {
		t.Error("HasFuzzy should be true when dimensions are present")
	}

	// MeanS = (80+90+70+80+80+80+80)/7 = 560/7 = 80.0
	assertApproxEqual(t, "MeanS", result.MeanS, 80.0)
	// MeanR = (90+80+70+80+80+80+80)/7 = 560/7 = 80.0
	assertApproxEqual(t, "MeanR", result.MeanR, 80.0)
	// MeanA = (70+80+90+80+80+80+80)/7 = 560/7 = 80.0
	assertApproxEqual(t, "MeanA", result.MeanA, 80.0)
	// MeanD = (85+75+90+80+80+80+80)/7 = 570/7 = 81.4
	assertApproxEqual(t, "MeanD", result.MeanD, 81.4)
}

func TestCheckGate_Pass(t *testing.T) {
	// fix change type has threshold 2.0
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | |",
		"| 2 | Certain | D2 | R2 | |",
		"| 3 | Certain | D3 | R3 | |",
		"| 4 | Certain | D4 | R4 | |",
		"| 5 | Certain | D5 | R5 | |",
	)
	fabRoot := setupScoreFixture(t, "fix", spec)

	result, err := CheckGate(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("CheckGate failed: %v", err)
	}

	// 5 certain, total=5, expectedMin for fix spec=5, cover=1.0
	// score = 5.0, threshold = 2.0 => pass
	if result.Gate != "pass" {
		t.Errorf("Gate = %q, want pass", result.Gate)
	}
	if result.Threshold != 2.0 {
		t.Errorf("Threshold = %.1f, want 2.0", result.Threshold)
	}
}

func TestCheckGate_Fail(t *testing.T) {
	// feat change type has threshold 3.0, but only 3 decisions (cover factor low)
	spec := specWithAssumptions(
		"| 1 | Confident | D1 | R1 | |",
		"| 2 | Confident | D2 | R2 | |",
		"| 3 | Confident | D3 | R3 | |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := CheckGate(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("CheckGate failed: %v", err)
	}

	// base = 5.0 - 0.3*3 = 4.1, cover = 3/7, score = 4.1 * 3/7 ~= 1.8
	// threshold for feat = 3.0 => fail
	if result.Gate != "fail" {
		t.Errorf("Gate = %q, want fail (score=%.1f, threshold=%.1f)", result.Gate, result.Score, result.Threshold)
	}
}

func TestCheckGate_IntakeStage(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)

	statusYAML := strings.Replace(statusTemplate, "%s", "feat", 1)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(statusYAML), 0644)

	// Write intake.md with assumptions scoring below 3.0
	intakeContent := specWithAssumptions(
		"| 1 | Confident | D1 | R1 | |",
		"| 2 | Confident | D2 | R2 | |",
	)
	os.WriteFile(filepath.Join(changeDir, "intake.md"), []byte(intakeContent), 0644)

	result, err := CheckGate(fabRoot, "abcd", "intake")
	if err != nil {
		t.Fatalf("CheckGate intake failed: %v", err)
	}

	// Intake gate threshold is always 3.0
	if result.Threshold != 3.0 {
		t.Errorf("Threshold = %.1f, want 3.0", result.Threshold)
	}

	// base = 5.0 - 0.3*2 = 4.4, total=2, expectedMin for feat intake=5, cover=2/5=0.4
	// score = 4.4 * 0.4 = 1.8 => fail
	if result.Gate != "fail" {
		t.Errorf("Gate = %q, want fail (score=%.1f)", result.Gate, result.Score)
	}
}

func TestFormatGateYAML(t *testing.T) {
	r := &GateResult{
		Gate:       "pass",
		Score:      4.5,
		Threshold:  3.0,
		ChangeType: "feat",
		Certain:    5,
		Confident:  1,
		Tentative:  0,
		Unresolved: 0,
	}
	output := FormatGateYAML(r)

	for _, want := range []string{"gate: pass", "score: 4.5", "threshold: 3.0", "change_type: feat"} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatGateYAML missing %q in output: %s", want, output)
		}
	}
}

func TestFormatScoreYAML(t *testing.T) {
	r := &ScoreResult{
		Certain:    5,
		Confident:  1,
		Tentative:  0,
		Unresolved: 0,
		Score:      4.7,
		Delta:      "+0.3",
		HasFuzzy:   true,
		MeanS:      80.0,
		MeanR:      85.0,
		MeanA:      75.0,
		MeanD:      90.0,
	}
	output := FormatScoreYAML(r)

	for _, want := range []string{"confidence:", "certain: 5", "score: 4.7", "delta: +0.3", "fuzzy: true", "signal: 80.0", "disambiguation: 90.0"} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatScoreYAML missing %q in output: %s", want, output)
		}
	}
}

func TestConstants(t *testing.T) {
	// Verify the penalty constants match the spec
	if wCertain != 0.0 {
		t.Errorf("wCertain = %f, want 0.0", wCertain)
	}
	if wConfident != 0.3 {
		t.Errorf("wConfident = %f, want 0.3", wConfident)
	}
	if wTentative != 1.0 {
		t.Errorf("wTentative = %f, want 1.0", wTentative)
	}

	// Verify gate thresholds
	if gateThresholds["feat"] != 3.0 {
		t.Errorf("feat threshold = %f, want 3.0", gateThresholds["feat"])
	}
	if gateThresholds["fix"] != 2.0 {
		t.Errorf("fix threshold = %f, want 2.0", gateThresholds["fix"])
	}
}
