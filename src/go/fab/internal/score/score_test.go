package score

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

const statusTemplate = `id: abcd
name: 260310-abcd-my-change
created: "2026-03-10T12:00:00Z"
created_by: test-user
change_type: %s
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

// setupScoreFixture creates a fab structure with a change directory and
// writes the given intake.md content (intake is the sole scoring source as of
// 1.10.0). Returns fabRoot.
func setupScoreFixture(t *testing.T, changeType, assumptionsContent string) string {
	t.Helper()
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)

	// Write .status.yaml
	statusYAML := strings.Replace(statusTemplate, "%s", changeType, 1)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(statusYAML), 0644)

	// Write intake.md (scoring reads intake.md)
	os.WriteFile(filepath.Join(changeDir, "intake.md"), []byte(assumptionsContent), 0644)

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

func TestCompute_AllStrongDimensions(t *testing.T) {
	// Resolution Average: 7 rows all at S:80 R:80 A:80 D:80 → composite 80.0
	// each → mean 80.0. total=7, expectedMin for feat=7, cover=1.0.
	// score = round1((80.0/20)*1.0) = 4.0
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:80 R:80 A:80 D:80 |",
		"| 2 | Certain | D2 | R2 | S:80 R:80 A:80 D:80 |",
		"| 3 | Certain | D3 | R3 | S:80 R:80 A:80 D:80 |",
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

	assertApproxEqual(t, "Score", result.Score, 4.0)
	if result.Certain != 7 {
		t.Errorf("Certain = %d, want 7", result.Certain)
	}
	if result.Confident != 0 {
		t.Errorf("Confident = %d, want 0", result.Confident)
	}
}

func TestCompute_PerfectDimensionsScoreFive(t *testing.T) {
	// All dimensions at 100 → composite 100.0 → mean 100.0, cover=1.0.
	// score = round1((100.0/20)*1.0) = 5.0 (the 0–5 ceiling)
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:100 R:100 A:100 D:100 |",
		"| 2 | Certain | D2 | R2 | S:100 R:100 A:100 D:100 |",
		"| 3 | Certain | D3 | R3 | S:100 R:100 A:100 D:100 |",
		"| 4 | Certain | D4 | R4 | S:100 R:100 A:100 D:100 |",
		"| 5 | Certain | D5 | R5 | S:100 R:100 A:100 D:100 |",
		"| 6 | Certain | D6 | R6 | S:100 R:100 A:100 D:100 |",
		"| 7 | Certain | D7 | R7 | S:100 R:100 A:100 D:100 |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	assertApproxEqual(t, "Score", result.Score, 5.0)
}

func TestCompute_MeanComposite_MixedRows(t *testing.T) {
	// Mixed dimensions. Per-row composite = 0.25*S+0.30*R+0.25*A+0.20*D.
	//  r1 S:90 R:85 A:88 D:80: 22.5+25.5+22.0+16.0 = 86.0
	//  r2 S:70 R:60 A:65 D:55: 17.5+18.0+16.25+11.0 = 62.75
	//  r3 S:80 R:75 A:78 D:70: 20.0+22.5+19.5+14.0 = 76.0
	// mean = (86.0+62.75+76.0)/3 = 224.75/3 = 74.9167
	// total=3, expectedMin for refactor=6, cover=3/6=0.5
	// score = round1((74.9167/20)*0.5) = round1(1.8729) = 1.9
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:85 A:88 D:80 |",
		"| 2 | Confident | D2 | R2 | S:70 R:60 A:65 D:55 |",
		"| 3 | Confident | D3 | R3 | S:80 R:75 A:78 D:70 |",
	)
	fabRoot := setupScoreFixture(t, "refactor", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	assertApproxEqual(t, "Score", result.Score, 1.9)
}

func TestCompute_UnresolvedZero(t *testing.T) {
	// Any Unresolved row hard-fails to 0.0, even with strong dimensions.
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:90 A:90 D:90 |",
		"| 2 | Certain | D2 | R2 | S:90 R:90 A:90 D:90 |",
		"| 3 | Unresolved | D3 | R3 | S:90 R:90 A:90 D:90 |",
		"| 4 | Certain | D4 | R4 | S:90 R:90 A:90 D:90 |",
		"| 5 | Certain | D5 | R5 | S:90 R:90 A:90 D:90 |",
		"| 6 | Certain | D6 | R6 | S:90 R:90 A:90 D:90 |",
		"| 7 | Certain | D7 | R7 | S:90 R:90 A:90 D:90 |",
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

func TestCompute_CriticalRuleHardFail(t *testing.T) {
	// A single row with R < 25 AND A < 25 (raw dimensions) hard-fails the whole
	// intake to 0.0 — even though every other row is perfect.
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:90 A:90 D:90 |",
		"| 2 | Certain | D2 | R2 | S:90 R:90 A:90 D:90 |",
		"| 3 | Tentative | D3 | R3 | S:40 R:20 A:20 D:40 |",
		"| 4 | Certain | D4 | R4 | S:90 R:90 A:90 D:90 |",
		"| 5 | Certain | D5 | R5 | S:90 R:90 A:90 D:90 |",
		"| 6 | Certain | D6 | R6 | S:90 R:90 A:90 D:90 |",
		"| 7 | Certain | D7 | R7 | S:90 R:90 A:90 D:90 |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	if result.Score != 0.0 {
		t.Errorf("Score = %.1f, want 0.0 (Critical Rule: R<25 AND A<25 row)", result.Score)
	}
}

func TestCompute_CriticalRuleBoundaryDoesNotFail(t *testing.T) {
	// R=25 and A=25 are NOT below the threshold (< 25 is strict), so the
	// Critical Rule does not fire. Mean is computed normally.
	//  composite = 0.25*60+0.30*25+0.25*25+0.20*60 = 15+7.5+6.25+12 = 40.75
	// 3 identical rows → mean 40.75, refactor expectedMin=6, cover=3/6=0.5
	// score = round1((40.75/20)*0.5) = round1(1.0188) = 1.0
	spec := specWithAssumptions(
		"| 1 | Tentative | D1 | R1 | S:60 R:25 A:25 D:60 |",
		"| 2 | Tentative | D2 | R2 | S:60 R:25 A:25 D:60 |",
		"| 3 | Tentative | D3 | R3 | S:60 R:25 A:25 D:60 |",
	)
	fabRoot := setupScoreFixture(t, "refactor", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	assertApproxEqual(t, "Score", result.Score, 1.0)
}

func TestCompute_CoverFactor(t *testing.T) {
	// Only 3 decisions for a feat change (expectedMin=7). Strong dimensions but
	// thin coverage attenuates the score.
	//  composite for S:80 R:80 A:80 D:80 = 80.0 each → mean 80.0
	// cover = 3/7 ≈ 0.4286, score = round1((80.0/20)*(3/7)) = round1(1.714) = 1.7
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:80 R:80 A:80 D:80 |",
		"| 2 | Certain | D2 | R2 | S:80 R:80 A:80 D:80 |",
		"| 3 | Certain | D3 | R3 | S:80 R:80 A:80 D:80 |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	assertApproxEqual(t, "Score", result.Score, 1.7)
}

func TestCompute_DimensionlessRowsScoreZero(t *testing.T) {
	// Rows with no parseable Scores column have no dimensions to average
	// (DimCount=0). The Scores column is required on every row, so a fully
	// dimensionless table is a malformed intake that scores 0.0.
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

	if result.Score != 0.0 {
		t.Errorf("Score = %.1f, want 0.0 (no parseable dimensions)", result.Score)
	}
}

func TestCompute_DimensionlessRowStillCountsTowardCoverage(t *testing.T) {
	// The mean restricts to DimCount rows, but coverage's total counts ALL
	// graded rows. Here 2 rows have dimensions and 1 is dimensionless.
	//  composite for S:80 R:80 A:80 D:80 = 80.0 each → mean over 2 rows = 80.0
	// total = 3 (all graded rows), feat expectedMin=7, cover=3/7≈0.4286
	// score = round1((80.0/20)*(3/7)) = round1(1.714) = 1.7
	// If the dimensionless row were dropped from total, cover would be 2/7 and
	// the score would be lower — this asserts it is NOT dropped.
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:80 R:80 A:80 D:80 |",
		"| 2 | Certain | D2 | R2 | S:80 R:80 A:80 D:80 |",
		"| 3 | Certain | D3 | R3 | |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	assertApproxEqual(t, "Score", result.Score, 1.7)
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
	// fix change type now has expectedMin=3 (lowered from 5) and the flat gate
	// threshold 3.0 (1.10.0). Strong dimensions over 3 rows.
	//  composite for S:90 R:88 A:90 D:85 = 22.5+26.4+22.5+17.0 = 88.4 each
	// mean=88.4, total=3, expectedMin for fix=3, cover=1.0
	// score = round1((88.4/20)*1.0) = round1(4.42) = 4.4, threshold 3.0 => pass
	intake := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:88 A:90 D:85 |",
		"| 2 | Certain | D2 | R2 | S:90 R:88 A:90 D:85 |",
		"| 3 | Certain | D3 | R3 | S:90 R:88 A:90 D:85 |",
	)
	fabRoot := setupScoreFixture(t, "fix", intake)

	result, err := CheckGate(fabRoot, "abcd", "intake")
	if err != nil {
		t.Fatalf("CheckGate failed: %v", err)
	}

	if result.Gate != "pass" {
		t.Errorf("Gate = %q, want pass (score=%.1f)", result.Gate, result.Score)
	}
	if result.Threshold != 3.0 {
		t.Errorf("Threshold = %.1f, want 3.0", result.Threshold)
	}
}

func TestCheckGate_Fail(t *testing.T) {
	// feat change type, threshold 3.0, only 3 decisions (cover factor low).
	//  composite for S:70 R:70 A:70 D:70 = 70.0 each → mean 70.0
	// cover = 3/7 ≈ 0.4286, score = round1((70.0/20)*(3/7)) = round1(1.5) = 1.5
	// threshold for feat = 3.0 => fail
	intake := specWithAssumptions(
		"| 1 | Confident | D1 | R1 | S:70 R:70 A:70 D:70 |",
		"| 2 | Confident | D2 | R2 | S:70 R:70 A:70 D:70 |",
		"| 3 | Confident | D3 | R3 | S:70 R:70 A:70 D:70 |",
	)
	fabRoot := setupScoreFixture(t, "feat", intake)

	result, err := CheckGate(fabRoot, "abcd", "intake")
	if err != nil {
		t.Fatalf("CheckGate failed: %v", err)
	}

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

	// Write intake.md with assumptions scoring below 3.0 (thin coverage).
	//  composite for S:80 R:80 A:80 D:80 = 80.0 → mean 80.0
	// total=2, expectedMin for feat=7, cover=2/7≈0.2857
	// score = round1((80.0/20)*(2/7)) = round1(1.143) = 1.1 => fail
	intakeContent := specWithAssumptions(
		"| 1 | Confident | D1 | R1 | S:80 R:80 A:80 D:80 |",
		"| 2 | Confident | D2 | R2 | S:80 R:80 A:80 D:80 |",
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

// --- Scanner-truncation and error-surfacing coverage (hv7t) ---

func TestCheckGate_OversizedLineInsideTableCountsAllRows(t *testing.T) {
	// The old default-buffer scanner aborted on a >64KB line, dropping every
	// row after it — including the Unresolved row that forces score 0.0, so
	// a hard-fail intake could flip to gate: pass. All rows must count.
	long := "| 6 | Certain | " + strings.Repeat("x", 70*1024) + " | R | |"
	intake := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | |",
		"| 2 | Certain | D2 | R2 | |",
		"| 3 | Certain | D3 | R3 | |",
		"| 4 | Certain | D4 | R4 | |",
		"| 5 | Certain | D5 | R5 | |",
		long,
		"| 7 | Unresolved | D7 | R7 | |",
	)
	fabRoot := setupScoreFixture(t, "feat", intake)

	result, err := CheckGate(fabRoot, "abcd", "intake")
	if err != nil {
		t.Fatalf("CheckGate failed: %v", err)
	}

	if result.Unresolved != 1 {
		t.Errorf("Unresolved = %d, want 1 (row after oversized line must be counted)", result.Unresolved)
	}
	if result.Certain != 6 {
		t.Errorf("Certain = %d, want 6 (oversized row itself must be counted)", result.Certain)
	}
	if result.Gate != "fail" {
		t.Errorf("Gate = %q, want fail — truncation must not flip the gate", result.Gate)
	}
	if result.Score != 0.0 {
		t.Errorf("Score = %.1f, want 0.0 (unresolved present)", result.Score)
	}
}

func TestCompute_OversizedLineInsideTableCountsAllRows(t *testing.T) {
	long := "| 2 | Certain | " + strings.Repeat("y", 70*1024) + " | R | |"
	intake := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | |",
		long,
		"| 3 | Tentative | D3 | R3 | |",
	)
	fabRoot := setupScoreFixture(t, "feat", intake)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}
	if result.Certain != 2 || result.Tentative != 1 {
		t.Errorf("counts = %d certain / %d tentative, want 2/1 (no truncation)", result.Certain, result.Tentative)
	}
}

func TestCompute_StatusLoadFailureReturnsError(t *testing.T) {
	fabRoot := setupScoreFixture(t, "feat", specWithAssumptions("| 1 | Certain | D | R | |"))

	// Corrupt .status.yaml: previously Compute silently skipped the
	// write-back, defaulted change_type to feat, and reported success.
	statusPath := filepath.Join(fabRoot, "changes", "260310-abcd-my-change", ".status.yaml")
	os.WriteFile(statusPath, []byte("not: [valid: yaml"), 0644)

	if _, err := Compute(fabRoot, "abcd", ""); err == nil {
		t.Fatal("expected error for unloadable .status.yaml, got nil")
	}
}

func TestCompute_MissingStatusFileReturnsError(t *testing.T) {
	fabRoot := setupScoreFixture(t, "feat", specWithAssumptions("| 1 | Certain | D | R | |"))

	statusPath := filepath.Join(fabRoot, "changes", "260310-abcd-my-change", ".status.yaml")
	os.Remove(statusPath)

	if _, err := Compute(fabRoot, "abcd", ""); err == nil {
		t.Fatal("expected error for missing .status.yaml, got nil")
	}
}

func TestCompute_PersistFailureReturnsError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-denied semantics do not apply to root")
	}

	fabRoot := setupScoreFixture(t, "feat", specWithAssumptions("| 1 | Certain | D | R | |"))

	// A read-only change directory lets .status.yaml load but makes the
	// atomic save's CreateTemp fail — previously discarded via `_ =`.
	// Pre-create the sibling lock file and the history log so Compute's lock
	// acquisition and ComputeWithStatus's .history.jsonl append (both only
	// need to open an existing writable file) succeed, and the failure
	// surfaces from the Save itself.
	changeDir := filepath.Join(fabRoot, "changes", "260310-abcd-my-change")
	if err := os.WriteFile(filepath.Join(changeDir, ".status.yaml.lock"), nil, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, ".history.jsonl"), nil, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.Chmod(changeDir, 0o555); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { os.Chmod(changeDir, 0o755) })

	_, err := Compute(fabRoot, "abcd", "")
	if err == nil {
		t.Fatal("expected persistence error, got nil")
	}
	if !strings.Contains(err.Error(), "persist confidence") {
		t.Errorf("error = %q, want it to mention confidence persistence", err.Error())
	}
}

func TestScore_ReadFailureDistinguishableFromEmptyTable(t *testing.T) {
	// countGrades parses caller-read content and cannot fail; read-failure
	// surfacing lives in CheckGate/Compute's os.ReadFile (mz4q F06 posture).
	// An unreadable intake.md must be an error — never a zero-count result.
	if os.Geteuid() == 0 {
		t.Skip("permission-denied fixtures do not bind as root")
	}
	fabRoot := setupScoreFixture(t, "feat", specWithAssumptions("| 1 | Certain | D | R | |"))
	intakePath := filepath.Join(fabRoot, "changes", "260310-abcd-my-change", "intake.md")
	if err := os.Chmod(intakePath, 0o000); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { os.Chmod(intakePath, 0o644) })

	if _, err := CheckGate(fabRoot, "abcd", "intake"); err == nil {
		t.Fatal("expected read error from CheckGate, got nil")
	}
	if _, err := Compute(fabRoot, "abcd", ""); err == nil {
		t.Fatal("expected read error from Compute, got nil")
	}

	// No Assumptions table → zero GradeCount (the legitimate empty case).
	if gc := countGrades([]byte("# Intake\n\nNo table here.\n")); gc != (GradeCount{}) {
		t.Errorf("gc = %+v, want zero GradeCount for table-less intake", gc)
	}
}

func TestConstants(t *testing.T) {
	// Verify the SRAD composite weights match srad.md's aggregation formula
	// (0.25*S + 0.30*R + 0.25*A + 0.20*D — they MUST sum to 1.0).
	if wS != 0.25 {
		t.Errorf("wS = %f, want 0.25", wS)
	}
	if wR != 0.30 {
		t.Errorf("wR = %f, want 0.30", wR)
	}
	if wA != 0.25 {
		t.Errorf("wA = %f, want 0.25", wA)
	}
	if wD != 0.20 {
		t.Errorf("wD = %f, want 0.20", wD)
	}
	if sum := wS + wR + wA + wD; math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("composite weights sum = %f, want 1.0", sum)
	}

	// The 0–100 composite mean is rescaled onto the 0–5 scale via /20, so a 3.0
	// gate equals a mean composite of 60 (the Confident floor).
	if compositeToScore != 20.0 {
		t.Errorf("compositeToScore = %f, want 20.0", compositeToScore)
	}
	if criticalDim != 25 {
		t.Errorf("criticalDim = %d, want 25", criticalDim)
	}

	// Verify gate thresholds — flat 3.0 for all types (1.10.0)
	if gateThresholds["feat"] != 3.0 {
		t.Errorf("feat threshold = %f, want 3.0", gateThresholds["feat"])
	}
	if gateThresholds["fix"] != 3.0 {
		t.Errorf("fix threshold = %f, want 3.0", gateThresholds["fix"])
	}

	// Verify single expectedMin table (fix lowered 5→3)
	if expectedMin["feat"] != 7 {
		t.Errorf("expectedMin[feat] = %d, want 7", expectedMin["feat"])
	}
	if expectedMin["refactor"] != 6 {
		t.Errorf("expectedMin[refactor] = %d, want 6", expectedMin["refactor"])
	}
	if expectedMin["fix"] != 3 {
		t.Errorf("expectedMin[fix] = %d, want 3", expectedMin["fix"])
	}
	if getExpectedMin("fix") != 3 {
		t.Errorf("getExpectedMin(fix) = %d, want 3", getExpectedMin("fix"))
	}
	if getExpectedMin("docs") != 3 {
		t.Errorf("getExpectedMin(docs) = %d, want 3 (default)", getExpectedMin("docs"))
	}
}

// --- ComputeWithStatus (mz4q F02): single-load entry point — mutates the
// loaded StatusFile in memory, never saves; the caller owns persistence. ---

func TestComputeWithStatus_MutatesInMemoryWithoutSaving(t *testing.T) {
	// 5 rows all at S:88 R:90 A:92 D:90 → composite
	// 0.25*88+0.30*90+0.25*92+0.20*90 = 22+27+23+18 = 90.0 each → mean 90.0.
	// fix expectedMin=3, total=5 → cover=1.0. score = round1((90.0/20)*1.0) = 4.5
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:88 R:90 A:92 D:90 |",
		"| 2 | Confident | D2 | R2 | S:88 R:90 A:92 D:90 |",
		"| 3 | Certain | D3 | R3 | S:88 R:90 A:92 D:90 |",
		"| 4 | Certain | D4 | R4 | S:88 R:90 A:92 D:90 |",
		"| 5 | Certain | D5 | R5 | S:88 R:90 A:92 D:90 |",
	)
	fabRoot := setupScoreFixture(t, "fix", spec)
	changeDir := filepath.Join(fabRoot, "changes", "260310-abcd-my-change")
	statusPath := filepath.Join(changeDir, ".status.yaml")

	statusFile, err := sf.Load(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(changeDir, "intake.md"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := ComputeWithStatus(fabRoot, changeDir, content, statusFile)
	if err != nil {
		t.Fatalf("ComputeWithStatus failed: %v", err)
	}

	assertApproxEqual(t, "Score", result.Score, 4.5)
	assertApproxEqual(t, "Confidence.Score (in memory)", statusFile.Confidence.Score, 4.5)
	if statusFile.Confidence.Certain != 4 || statusFile.Confidence.Confident != 1 {
		t.Errorf("in-memory confidence counts = %+v", statusFile.Confidence)
	}

	// No save: .status.yaml on disk is untouched.
	after, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("ComputeWithStatus must not save — caller owns persistence")
	}

	// The confidence event is logged against the resolved changeDir.
	if _, err := os.Stat(filepath.Join(changeDir, ".history.jsonl")); err != nil {
		t.Errorf("expected confidence event logged to .history.jsonl: %v", err)
	}
}

func TestCheckGate_UnreadableIntakeClassified(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root — permission bits are not enforced")
	}
	fabRoot := setupScoreFixture(t, "feat", specWithAssumptions("| 1 | Certain | D1 | R1 | |"))
	intakePath := filepath.Join(fabRoot, "changes", "260310-abcd-my-change", "intake.md")
	if err := os.Chmod(intakePath, 0o000); err != nil {
		t.Fatal(err)
	}

	_, err := CheckGate(fabRoot, "abcd", "intake")
	if err == nil {
		t.Fatal("expected error for unreadable intake.md")
	}
	if strings.Contains(err.Error(), "not found") {
		t.Errorf("permission failure must not masquerade as absence: %v", err)
	}
	if !strings.Contains(err.Error(), "read") || !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected cause-bearing read error, got: %v", err)
	}
}

func TestCompute_UnreadableIntakeClassified(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root — permission bits are not enforced")
	}
	fabRoot := setupScoreFixture(t, "feat", specWithAssumptions("| 1 | Certain | D1 | R1 | |"))
	intakePath := filepath.Join(fabRoot, "changes", "260310-abcd-my-change", "intake.md")
	if err := os.Chmod(intakePath, 0o000); err != nil {
		t.Fatal(err)
	}

	_, err := Compute(fabRoot, "abcd", "intake")
	if err == nil {
		t.Fatal("expected error for unreadable intake.md")
	}
	if strings.Contains(err.Error(), "required for scoring") {
		t.Errorf("permission failure must not masquerade as a missing intake: %v", err)
	}
	if !strings.Contains(err.Error(), "read") || !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected cause-bearing read error, got: %v", err)
	}
}
