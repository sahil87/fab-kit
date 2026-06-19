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
	// Demerit model: 7 rows all at S:80 R:80 A:80 D:80 → composite
	// 0.20*80+0.30*80+0.30*80+0.20*80 = 80.0 each → penalty 0 (Certain, c>=80).
	// score = clamp(5.0 - 0, 0, 5) = 5.0. No coverage attenuation.
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

	assertApproxEqual(t, "Score", result.Score, 5.0)
	// Grades are derived from the composite: c=80 → Certain.
	if result.Certain != 7 {
		t.Errorf("Certain = %d, want 7", result.Certain)
	}
	if result.Confident != 0 {
		t.Errorf("Confident = %d, want 0", result.Confident)
	}
}

func TestCompute_PerfectDimensionsScoreFive(t *testing.T) {
	// All dimensions at 100 → composite 100.0 → penalty 0 each (Certain).
	// score = clamp(5.0 - 0, 0, 5) = 5.0 (the 0–5 ceiling)
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

func TestCompute_MixedRows_PenaltySum(t *testing.T) {
	// Mixed dimensions. Per-row composite = 0.20*S+0.30*R+0.30*A+0.20*D, then
	// the demerit penalty is summed (no mean, no coverage).
	//  r1 S:90 R:85 A:88 D:80: 18+25.5+26.4+16 = 85.9  → c>=80 → penalty 0
	//  r2 S:70 R:60 A:65 D:55: 14+18+19.5+11   = 62.5  → (80-62.5)/30*0.5 = 0.2917
	//  r3 S:80 R:75 A:78 D:70: 16+22.5+23.4+14 = 75.9  → (80-75.9)/30*0.5 = 0.0683
	// Σ penalty = 0.36 → score = round1(5.0 - 0.36) = 4.6
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

	assertApproxEqual(t, "Score", result.Score, 4.6)
	// Grades derived from composite: 85.9→Certain, 62.5/75.9→Confident.
	if result.Certain != 1 || result.Confident != 2 {
		t.Errorf("grades = %d certain / %d confident, want 1/2 (derived from composite)", result.Certain, result.Confident)
	}
}

func TestCompute_SingleUnresolvedBlocks(t *testing.T) {
	// No hard-fail short-circuit: a single genuinely-Unresolved row (composite
	// < 20) blocks the gate purely via the curve. Six strong rows penalize 0;
	// the weak row S:10 R:10 A:10 D:10 → composite 10 → penalty
	// 0.50 + (50-10)/50*2.50 = 0.50 + 2.0 = 2.5. score = 5.0 - 2.5 = 2.5 (fails).
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:90 A:90 D:90 |",
		"| 2 | Certain | D2 | R2 | S:90 R:90 A:90 D:90 |",
		"| 3 | Unresolved | D3 | R3 | S:10 R:10 A:10 D:10 |",
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

	assertApproxEqual(t, "Score", result.Score, 2.5)
	if result.Score >= 3.0 {
		t.Errorf("Score = %.1f, want < 3.0 (single Unresolved row must block)", result.Score)
	}
	// Grade is derived from composite: c=10 → Unresolved (no longer a hard fail).
	if result.Unresolved != 1 {
		t.Errorf("Unresolved = %d, want 1 (derived from composite < 20)", result.Unresolved)
	}
}

func TestCompute_NoCriticalRuleHardFail(t *testing.T) {
	// The old R<25 AND A<25 Critical Rule is removed. A row at S:40 R:20 A:20
	// D:40 → composite 0.20*40+0.30*20+0.30*20+0.20*40 = 8+6+6+8 = 28
	// (Tentative). penalty = 0.50 + (50-28)/50*2.50 = 0.50 + 1.10 = 1.60. The
	// six strong rows penalize 0. score = 5.0 - 1.60 = 3.4 — PASSES, where the
	// old Critical Rule would have hard-failed it to 0.0.
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

	assertApproxEqual(t, "Score", result.Score, 3.4)
	if result.Tentative != 1 {
		t.Errorf("Tentative = %d, want 1 (c=28 derived grade)", result.Tentative)
	}
}

func TestCompute_ThinButStrong(t *testing.T) {
	// Coverage / expected_min is dropped: a thin 2-row all-Certain intake is NOT
	// punished for being short. Both rows S:90 R:90 A:90 D:90 → composite 90 →
	// penalty 0. score = clamp(5.0 - 0, 0, 5) = 5.0 (feat had expectedMin 7).
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:90 A:90 D:90 |",
		"| 2 | Certain | D2 | R2 | S:90 R:90 A:90 D:90 |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	assertApproxEqual(t, "Score", result.Score, 5.0)
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

func TestCompute_DimensionlessRowIgnoredByDemerit(t *testing.T) {
	// Coverage is dropped, and grades are derived from the composite — so a
	// dimensionless row (no parseable Scores) has no composite, contributes no
	// penalty, and is not grade-counted. The two dimensioned rows S:80 R:80
	// A:80 D:80 → composite 80 → penalty 0 each. score = clamp(5.0 - 0, 0, 5) =
	// 5.0; only the two parseable rows count toward grades.
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

	assertApproxEqual(t, "Score", result.Score, 5.0)
	if result.Certain != 2 {
		t.Errorf("Certain = %d, want 2 (dimensionless row is not grade-counted)", result.Certain)
	}
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
	// Flat gate threshold 3.0 (all types). Strong dimensions over 3 rows.
	//  composite for S:90 R:88 A:90 D:85 = 18+26.4+27+17 = 88.4 each → c>=80 →
	//  penalty 0. score = clamp(5.0 - 0, 0, 5) = 5.0, threshold 3.0 => pass.
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
	// feat change type, threshold 3.0. Three deep-Tentative rows accumulate
	// enough penalty to fail (no coverage factor — the failure is the penalties).
	//  composite for S:30 R:30 A:30 D:30 = 6+9+9+6 = 30 each (Tentative) →
	//  penalty 0.50 + (50-30)/50*2.50 = 0.50 + 1.0 = 1.5 each.
	// Σ penalty = 4.5 → score = clamp(5.0 - 4.5, 0, 5) = 0.5 → fail.
	intake := specWithAssumptions(
		"| 1 | Tentative | D1 | R1 | S:30 R:30 A:30 D:30 |",
		"| 2 | Tentative | D2 | R2 | S:30 R:30 A:30 D:30 |",
		"| 3 | Tentative | D3 | R3 | S:30 R:30 A:30 D:30 |",
	)
	fabRoot := setupScoreFixture(t, "feat", intake)

	result, err := CheckGate(fabRoot, "abcd", "intake")
	if err != nil {
		t.Fatalf("CheckGate failed: %v", err)
	}

	if result.Gate != "fail" {
		t.Errorf("Gate = %q, want fail (score=%.1f, threshold=%.1f)", result.Gate, result.Score, result.Threshold)
	}
	assertApproxEqual(t, "Score", result.Score, 0.5)
}

func TestCheckGate_IntakeStage(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)

	statusYAML := strings.Replace(statusTemplate, "%s", "feat", 1)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(statusYAML), 0644)

	// Write intake.md with assumptions scoring below the flat 3.0 gate. Two
	// weak rows S:30 R:20 A:20 D:30 → composite 6+6+6+6 = 24 (Tentative) →
	// penalty 0.50 + (50-24)/50*2.50 = 0.50 + 1.30 = 1.80 each.
	// Σ penalty = 3.6 → score = clamp(5.0 - 3.6, 0, 5) = 1.4 => fail.
	intakeContent := specWithAssumptions(
		"| 1 | Tentative | D1 | R1 | S:30 R:20 A:20 D:30 |",
		"| 2 | Tentative | D2 | R2 | S:30 R:20 A:20 D:30 |",
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
	// row after it — including the Unresolved row whose penalty fails the gate,
	// so a failing intake could flip to gate: pass. All rows must still parse.
	// Rows carry dimensions (the v2 score derives grades from the composite and
	// drops dimensionless rows). Five strong rows + the oversized strong row →
	// penalty 0; the final Unresolved row S:10 R:10 A:10 D:10 (composite 10) →
	// penalty 2.5. score = 5.0 - 2.5 = 2.5 → fail.
	long := "| 6 | Certain | " + strings.Repeat("x", 70*1024) + " | R | S:90 R:90 A:90 D:90 |"
	intake := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:90 A:90 D:90 |",
		"| 2 | Certain | D2 | R2 | S:90 R:90 A:90 D:90 |",
		"| 3 | Certain | D3 | R3 | S:90 R:90 A:90 D:90 |",
		"| 4 | Certain | D4 | R4 | S:90 R:90 A:90 D:90 |",
		"| 5 | Certain | D5 | R5 | S:90 R:90 A:90 D:90 |",
		long,
		"| 7 | Unresolved | D7 | R7 | S:10 R:10 A:10 D:10 |",
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
	assertApproxEqual(t, "Score", result.Score, 2.5)
}

func TestCompute_OversizedLineInsideTableCountsAllRows(t *testing.T) {
	// The oversized strong row (composite 90 → Certain) and the rows around it
	// must all parse. r1/r2 Certain, r3 Tentative (S:30 R:30 A:30 D:30 →
	// composite 30).
	long := "| 2 | Certain | " + strings.Repeat("y", 70*1024) + " | R | S:90 R:90 A:90 D:90 |"
	intake := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:90 A:90 D:90 |",
		long,
		"| 3 | Tentative | D3 | R3 | S:30 R:30 A:30 D:30 |",
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
	// Verify the v2 SRAD composite weights match srad.md § The Composite
	// (0.20*S + 0.30*R + 0.30*A + 0.20*D — R and A up-weighted; MUST sum to 1.0).
	if wS != 0.20 {
		t.Errorf("wS = %f, want 0.20", wS)
	}
	if wR != 0.30 {
		t.Errorf("wR = %f, want 0.30", wR)
	}
	if wA != 0.30 {
		t.Errorf("wA = %f, want 0.30", wA)
	}
	if wD != 0.20 {
		t.Errorf("wD = %f, want 0.20", wD)
	}
	if sum := wS + wR + wA + wD; math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("composite weights sum = %f, want 1.0", sum)
	}

	// Verify the demerit penalty-curve constants (srad.md § Confidence Scoring).
	if freeKnee != 80.0 {
		t.Errorf("freeKnee = %f, want 80.0", freeKnee)
	}
	if confidentFloorPenalty != 0.50 {
		t.Errorf("confidentFloorPenalty = %f, want 0.50", confidentFloorPenalty)
	}
	if aggressiveSlopeCoeff != 2.50 {
		t.Errorf("aggressiveSlopeCoeff = %f, want 2.50", aggressiveSlopeCoeff)
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

// --- Demerit penalty curve + grade derivation (srad.md § Confidence Scoring,
// § Grades) — pure-function coverage of the four bands and the band joins. ---

func TestPenaltyCurve_Bands(t *testing.T) {
	cases := []struct {
		name string
		c    float64
		want float64
	}{
		{"Certain ceiling (c=100)", 100, 0.0},
		{"Certain knee (c=80)", 80, 0.0},
		{"Confident mid (c=65)", 65, 0.25},                // (80-65)/30*0.50
		{"Confident/Tentative join (c=50)", 50, 0.50},     // both slopes meet
		{"Tentative mid (c=35)", 35, 1.25},                // 0.50 + (50-35)/50*2.50
		{"Tentative/Unresolved boundary (c=20)", 20, 2.0}, // exactly 2.0
		{"Unresolved (c=10)", 10, 2.5},
		{"Unresolved floor (c=0)", 0, 3.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := penalty(tc.c)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("penalty(%.1f) = %.4f, want %.4f", tc.c, got, tc.want)
			}
		})
	}
	// Continuity at the joins: per-row penalty ∈ [0.0, 3.0], monotonically
	// decreasing as composite rises.
	if penalty(80) != 0.0 {
		t.Errorf("penalty(80) = %.4f, want 0.0 (Certain knee)", penalty(80))
	}
	if math.Abs(penalty(50)-0.50) > 1e-9 {
		t.Errorf("penalty(50) = %.4f, want 0.50 (slopes meet)", penalty(50))
	}
}

func TestGradeFromComposite_Bands(t *testing.T) {
	cases := []struct {
		c    float64
		want string
	}{
		{100, "Certain"}, {80, "Certain"},
		{79.9, "Confident"}, {50, "Confident"},
		{49.9, "Tentative"}, {20, "Tentative"},
		{19.9, "Unresolved"}, {0, "Unresolved"},
	}
	for _, tc := range cases {
		if got := gradeFromComposite(tc.c); got != tc.want {
			t.Errorf("gradeFromComposite(%.1f) = %q, want %q", tc.c, got, tc.want)
		}
	}
}

func TestComputeScore_C20BoundaryPasses(t *testing.T) {
	// srad.md edge case: a single row at exactly composite 20.000
	// (S:0 R:0 A:0 D:100 → 0+0+0+20 = 20) penalizes exactly 2.0, leaving a
	// one-row intake at exactly 3.0 — a pass (gate is >= 3.0).
	spec := specWithAssumptions(
		"| 1 | Tentative | D1 | R1 | S:0 R:0 A:0 D:100 |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := CheckGate(fabRoot, "abcd", "intake")
	if err != nil {
		t.Fatalf("CheckGate failed: %v", err)
	}
	assertApproxEqual(t, "Score", result.Score, 3.0)
	if result.Gate != "pass" {
		t.Errorf("Gate = %q, want pass (c=20 boundary scores exactly 3.0)", result.Gate)
	}
}

func TestComputeScore_SurviveOneBlockTwo(t *testing.T) {
	// One isolated shaky decision survives; two block (srad.md § What 3.0
	// Allows). A Tentative row S:30 R:30 A:30 D:30 → composite 30 → penalty 1.5.
	// One such row: score 3.5 (pass). Two: 2.0 (fail).
	one := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:90 A:90 D:90 |",
		"| 2 | Tentative | D2 | R2 | S:30 R:30 A:30 D:30 |",
	)
	fabRoot := setupScoreFixture(t, "feat", one)
	result, err := CheckGate(fabRoot, "abcd", "intake")
	if err != nil {
		t.Fatalf("CheckGate (one) failed: %v", err)
	}
	assertApproxEqual(t, "Score (survive one)", result.Score, 3.5)
	if result.Gate != "pass" {
		t.Errorf("one shaky row: Gate = %q, want pass (score %.1f)", result.Gate, result.Score)
	}

	two := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:90 A:90 D:90 |",
		"| 2 | Tentative | D2 | R2 | S:30 R:30 A:30 D:30 |",
		"| 3 | Tentative | D3 | R3 | S:30 R:30 A:30 D:30 |",
	)
	fabRoot2 := setupScoreFixture(t, "feat", two)
	result2, err := CheckGate(fabRoot2, "abcd", "intake")
	if err != nil {
		t.Fatalf("CheckGate (two) failed: %v", err)
	}
	assertApproxEqual(t, "Score (block two)", result2.Score, 2.0)
	if result2.Gate != "fail" {
		t.Errorf("two shaky rows: Gate = %q, want fail (score %.1f)", result2.Gate, result2.Score)
	}
}

func TestComputeScore_FourBandsPenalties(t *testing.T) {
	// One row per band, demonstrating the penalty each band contributes:
	//  Certain    S:90 R:90 A:90 D:90 → composite 90 → penalty 0
	//  Confident  S:65 R:65 A:65 D:65 → composite 65 → (80-65)/30*0.50 = 0.25
	//  Tentative  S:35 R:35 A:35 D:35 → composite 35 → 0.50 + (50-35)/50*2.50 = 1.25
	//  Unresolved S:10 R:10 A:10 D:10 → composite 10 → 0.50 + (50-10)/50*2.50 = 2.50
	// Σ penalty = 0 + 0.25 + 1.25 + 2.50 = 4.0 → score = 5.0 - 4.0 = 1.0.
	spec := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:90 A:90 D:90 |",
		"| 2 | Confident | D2 | R2 | S:65 R:65 A:65 D:65 |",
		"| 3 | Tentative | D3 | R3 | S:35 R:35 A:35 D:35 |",
		"| 4 | Unresolved | D4 | R4 | S:10 R:10 A:10 D:10 |",
	)
	fabRoot := setupScoreFixture(t, "feat", spec)

	result, err := Compute(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}
	assertApproxEqual(t, "Score", result.Score, 1.0)
	if result.Certain != 1 || result.Confident != 1 || result.Tentative != 1 || result.Unresolved != 1 {
		t.Errorf("derived grades = %d/%d/%d/%d, want 1/1/1/1 (one per band)",
			result.Certain, result.Confident, result.Tentative, result.Unresolved)
	}
}

func TestCountGrades_GradeDerivedNotReadFromColumn(t *testing.T) {
	// The hand-written Grade column is ignored — the grade is derived from the
	// composite. Here every row is LABELLED "Certain" but the dimensions place
	// each in a different band.
	content := specWithAssumptions(
		"| 1 | Certain | D1 | R1 | S:90 R:90 A:90 D:90 |", // c=90 → Certain
		"| 2 | Certain | D2 | R2 | S:65 R:65 A:65 D:65 |", // c=65 → Confident
		"| 3 | Certain | D3 | R3 | S:35 R:35 A:35 D:35 |", // c=35 → Tentative
		"| 4 | Certain | D4 | R4 | S:10 R:10 A:10 D:10 |", // c=10 → Unresolved
	)
	gc := countGrades([]byte(content))
	if gc.Certain != 1 || gc.Confident != 1 || gc.Tentative != 1 || gc.Unresolved != 1 {
		t.Errorf("counts = %d/%d/%d/%d, want 1/1/1/1 — grade must derive from composite, not the Grade column",
			gc.Certain, gc.Confident, gc.Tentative, gc.Unresolved)
	}
}

// --- ComputeWithStatus (mz4q F02): single-load entry point — mutates the
// loaded StatusFile in memory, never saves; the caller owns persistence. ---

func TestComputeWithStatus_MutatesInMemoryWithoutSaving(t *testing.T) {
	// 5 rows all at S:88 R:90 A:92 D:90 → composite
	// 0.20*88+0.30*90+0.30*92+0.20*90 = 17.6+27+27.6+18 = 90.2 each → c>=80 →
	// penalty 0. score = clamp(5.0 - 0, 0, 5) = 5.0. Grades are derived from the
	// composite, so all five count as Certain regardless of the Grade column.
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

	assertApproxEqual(t, "Score", result.Score, 5.0)
	assertApproxEqual(t, "Confidence.Score (in memory)", statusFile.Confidence.Score, 5.0)
	if statusFile.Confidence.Certain != 5 || statusFile.Confidence.Confident != 0 {
		t.Errorf("in-memory confidence counts = %+v, want 5 certain / 0 confident (derived from composite)", statusFile.Confidence)
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
