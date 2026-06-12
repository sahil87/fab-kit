package score

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
	"github.com/sahil87/fab-kit/src/go/fab/internal/lockfile"
	"github.com/sahil87/fab-kit/src/go/fab/internal/log"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// Penalty weights per decision grade.
const (
	wCertain   = 0.0
	wConfident = 0.3
	wTentative = 1.0
)

// Expected minimum decisions by change type. Single table seeded from the old
// spec-gate values; types without an explicit entry (docs/test/ci/chore) use
// the default of 3. The intake gate is now the sole authoritative gate, so it
// demands spec-level decision coverage.
var expectedMin = map[string]int{
	"feat": 7, "refactor": 6, "fix": 5,
}

// Gate thresholds by change type. Flat 3.0 for all seven types (1.10.0). The
// per-type map is retained so future divergence is a data-only change.
var gateThresholds = map[string]float64{
	"fix": 3.0, "feat": 3.0, "refactor": 3.0,
	"docs": 3.0, "test": 3.0, "ci": 3.0, "chore": 3.0,
}

var scoresRegex = regexp.MustCompile(`S:(\d+)\s+R:(\d+)\s+A:(\d+)\s+D:(\d+)`)

// GradeCount holds parsed assumption counts.
type GradeCount struct {
	Certain                int
	Confident              int
	Tentative              int
	Unresolved             int
	HasFuzzy               bool
	DimCount               int
	SumS, SumR, SumA, SumD int
}

// GateResult holds the gate check output.
type GateResult struct {
	Gate       string
	Score      float64
	Threshold  float64
	ChangeType string
	Certain    int
	Confident  int
	Tentative  int
	Unresolved int
}

// ScoreResult holds the normal scoring output.
type ScoreResult struct {
	Certain                    int
	Confident                  int
	Tentative                  int
	Unresolved                 int
	Score                      float64
	Delta                      string
	HasFuzzy                   bool
	MeanS, MeanR, MeanA, MeanD float64
}

// CheckGate runs the gate check mode.
func CheckGate(fabRoot, changeArg, stage string) (*GateResult, error) {
	changeDir, err := resolve.ToAbsDir(fabRoot, changeArg)
	if err != nil {
		return nil, err
	}

	statusPath := filepath.Join(changeDir, ".status.yaml")
	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		return nil, fmt.Errorf(".status.yaml not found in %s", changeDir)
	}

	changeType := "feat"
	if statusFile, err := sf.Load(statusPath); err == nil {
		if ct := statusFile.ChangeType; ct != "" && ct != "null" {
			changeType = ct
		}
	}

	// The intake gate is the sole confidence gate (1.10.0). Scoring always
	// reads intake.md; the threshold routes through the per-type table (flat
	// 3.0 today) so future per-type divergence is a one-line data change.
	scoreFile := filepath.Join(changeDir, "intake.md")
	threshold := getGateThreshold(changeType)

	content, err := os.ReadFile(scoreFile)
	if err != nil {
		// Friendly not-found text only for genuine absence; other read
		// failures (permission, I/O) keep their cause (mz4q F06 posture).
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s not found in %s", filepath.Base(scoreFile), changeDir)
		}
		return nil, fmt.Errorf("read %s: %w", scoreFile, err)
	}

	gc := countGrades(content)
	total := gc.Certain + gc.Confident + gc.Tentative + gc.Unresolved
	score := computeScore(gc.Certain, gc.Confident, gc.Tentative, gc.Unresolved, total, getExpectedMin(changeType))

	gate := "pass"
	if score < threshold {
		gate = "fail"
	}

	return &GateResult{
		Gate:       gate,
		Score:      score,
		Threshold:  threshold,
		ChangeType: changeType,
		Certain:    gc.Certain,
		Confident:  gc.Confident,
		Tentative:  gc.Tentative,
		Unresolved: gc.Unresolved,
	}, nil
}

// Compute runs the normal scoring mode. The .status.yaml load-mutate-save
// cycle runs under the cross-process status lock so concurrent writers (the
// artifact-write hook, fab status in other panes) serialize (mz4q F03).
func Compute(fabRoot, changeArg, stage string) (*ScoreResult, error) {
	changeDir, err := resolve.ToAbsDir(fabRoot, changeArg)
	if err != nil {
		return nil, err
	}

	statusPath := filepath.Join(changeDir, ".status.yaml")

	// Scoring always reads intake.md (1.10.0): intake is the sole scoring
	// source now that the spec stage and spec.md are retired.
	scoreFile := filepath.Join(changeDir, "intake.md")
	content, err := os.ReadFile(scoreFile)
	if err != nil {
		// Friendly guidance only for genuine absence; other read failures
		// (permission, I/O) keep their cause (mz4q F06 posture).
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("intake.md required for scoring")
		}
		return nil, fmt.Errorf("read %s: %w", scoreFile, err)
	}

	var result *ScoreResult
	err = lockfile.WithLock(statusPath, func() error {
		// Load status file once for change type, previous score, and writing
		// back. A load failure is a hard error for explicit scoring: the
		// documented contract is "compute, write .status.yaml" — silently
		// skipping the write-back (and defaulting change_type to feat) would
		// report success while persisting nothing (hv7t F11). The
		// artifact-write hook keeps its best-effort posture by calling
		// ComputeWithStatus directly under its own err == nil guard.
		statusFile, loadErr := sf.Load(statusPath)
		if loadErr != nil {
			return loadErr
		}

		r, err := ComputeWithStatus(fabRoot, changeDir, content, statusFile)
		if err != nil {
			return err
		}
		// Persistence failures propagate — printing a score while silently
		// persisting nothing would leave stale confidence for preflight,
		// fab status confidence, fab change view/list, and pr-meta (hv7t F11).
		if err := statusFile.Save(statusPath); err != nil {
			return fmt.Errorf("persist confidence: %w", err)
		}
		result = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ComputeWithStatus is the single-load scoring entry point (mz4q F02): it
// reuses the caller's already-resolved changeDir, already-read intake.md
// content, and already-loaded StatusFile. It mutates statusFile.Confidence in
// memory and appends the confidence event to .history.jsonl, but does NOT
// save — the caller owns persistence and locking. Used by the artifact-write
// hook (inside its lock+single-Save cycle) and by Compute.
func ComputeWithStatus(fabRoot, changeDir string, intakeContent []byte, statusFile *sf.StatusFile) (*ScoreResult, error) {
	changeType := "feat"
	if ct := statusFile.ChangeType; ct != "" && ct != "null" {
		changeType = ct
	}
	prevScore := statusFile.Confidence.Score

	result := buildResult(intakeContent, changeType, prevScore)

	if result.HasFuzzy {
		status.ApplyConfidenceFuzzy(statusFile, result.Certain, result.Confident, result.Tentative, result.Unresolved, result.Score, result.MeanS, result.MeanR, result.MeanA, result.MeanD)
	} else {
		status.ApplyConfidence(statusFile, result.Certain, result.Confident, result.Tentative, result.Unresolved, result.Score)
	}

	// The history append is part of this entry point's contract; callers
	// choose the posture — Compute propagates (hv7t F11), the hook guards
	// with err == nil and stays best-effort.
	if err := log.ConfidenceLog(changeDir, result.Score, result.Delta, "calc-score"); err != nil {
		return nil, fmt.Errorf("log confidence: %w", err)
	}

	return result, nil
}

// buildResult counts grades in the intake content and assembles a ScoreResult
// for the given change type and previous score. Pure — no I/O, no mutation.
func buildResult(intakeContent []byte, changeType string, prevScore float64) *ScoreResult {
	gc := countGrades(intakeContent)
	total := gc.Certain + gc.Confident + gc.Tentative + gc.Unresolved
	score := computeScore(gc.Certain, gc.Confident, gc.Tentative, gc.Unresolved, total, getExpectedMin(changeType))

	// Compute dimension means
	var meanS, meanR, meanA, meanD float64
	if gc.DimCount > 0 {
		meanS = roundTo1(float64(gc.SumS) / float64(gc.DimCount))
		meanR = roundTo1(float64(gc.SumR) / float64(gc.DimCount))
		meanA = roundTo1(float64(gc.SumA) / float64(gc.DimCount))
		meanD = roundTo1(float64(gc.SumD) / float64(gc.DimCount))
	}

	return &ScoreResult{
		Certain:    gc.Certain,
		Confident:  gc.Confident,
		Tentative:  gc.Tentative,
		Unresolved: gc.Unresolved,
		Score:      score,
		Delta:      fmt.Sprintf("%+.1f", score-prevScore),
		HasFuzzy:   gc.HasFuzzy,
		MeanS:      meanS,
		MeanR:      meanR,
		MeanA:      meanA,
		MeanD:      meanD,
	}
}

// FormatGateYAML formats a GateResult as YAML.
func FormatGateYAML(r *GateResult) string {
	return fmt.Sprintf("gate: %s\nscore: %.1f\nthreshold: %.1f\nchange_type: %s\ncertain: %d\nconfident: %d\ntentative: %d\nunresolved: %d",
		r.Gate, r.Score, r.Threshold, r.ChangeType, r.Certain, r.Confident, r.Tentative, r.Unresolved)
}

// FormatScoreYAML formats a ScoreResult as YAML.
func FormatScoreYAML(r *ScoreResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "confidence:\n")
	fmt.Fprintf(&b, "  certain: %d\n", r.Certain)
	fmt.Fprintf(&b, "  confident: %d\n", r.Confident)
	fmt.Fprintf(&b, "  tentative: %d\n", r.Tentative)
	fmt.Fprintf(&b, "  unresolved: %d\n", r.Unresolved)
	fmt.Fprintf(&b, "  score: %.1f\n", r.Score)
	fmt.Fprintf(&b, "  delta: %s\n", r.Delta)

	if r.HasFuzzy {
		fmt.Fprintf(&b, "  fuzzy: true\n")
		fmt.Fprintf(&b, "  dimensions:\n")
		fmt.Fprintf(&b, "    signal: %.1f\n", r.MeanS)
		fmt.Fprintf(&b, "    reversibility: %.1f\n", r.MeanR)
		fmt.Fprintf(&b, "    competence: %.1f\n", r.MeanA)
		fmt.Fprintf(&b, "    disambiguation: %.1f\n", r.MeanD)
	}

	return b.String()
}

// countGrades scans already-read intake content for the ## Assumptions table.
// Taking the content (not a path) lets the artifact-write hook reuse its
// single intake.md read (mz4q F02). Lines come from lines.Split — not a
// bufio.Scanner — so an over-long line can never silently truncate the table
// and flip the gate by dropping graded rows (hv7t F09); a partial count must
// never be scored.
func countGrades(content []byte) GradeCount {
	gc := GradeCount{}
	inSection := false
	headerSeen := false

	for _, line := range lines.Split(string(content)) {
		if strings.HasPrefix(line, "## Assumptions") {
			inSection = true
			headerSeen = false
			continue
		}
		if inSection && strings.HasPrefix(line, "## ") {
			break
		}

		if !inSection {
			continue
		}

		if strings.HasPrefix(line, "| #") || strings.HasPrefix(line, "| # ") {
			headerSeen = true
			continue
		}

		// Skip separator lines
		trimmed := strings.TrimSpace(line)
		if headerSeen && isTableSeparator(trimmed) {
			continue
		}

		if headerSeen && strings.HasPrefix(line, "|") {
			cols := strings.Split(line, "|")
			if len(cols) < 6 {
				continue
			}

			grade := strings.TrimSpace(cols[2])
			gradeLower := strings.ToLower(grade)

			switch gradeLower {
			case "certain":
				gc.Certain++
			case "confident":
				gc.Confident++
			case "tentative":
				gc.Tentative++
			case "unresolved":
				gc.Unresolved++
			}

			scoresCol := ""
			if len(cols) >= 6 {
				scoresCol = strings.TrimSpace(cols[5])
			}

			if m := scoresRegex.FindStringSubmatch(scoresCol); len(m) == 5 {
				gc.HasFuzzy = true
				gc.DimCount++
				s, _ := strconv.Atoi(m[1])
				r, _ := strconv.Atoi(m[2])
				a, _ := strconv.Atoi(m[3])
				d, _ := strconv.Atoi(m[4])
				gc.SumS += s
				gc.SumR += r
				gc.SumA += a
				gc.SumD += d
			}
		}
	}

	return gc
}

func isTableSeparator(line string) bool {
	// Lines like |---|---|---|---|---|---|
	if !strings.HasPrefix(line, "|") {
		return false
	}
	for _, c := range line {
		if c != '|' && c != '-' && c != ' ' {
			return false
		}
	}
	return true
}

func computeScore(certain, confident, tentative, unresolved, total, expectedMin int) float64 {
	if unresolved > 0 {
		return 0.0
	}

	base := 5.0 - wCertain*float64(certain) - wConfident*float64(confident) - wTentative*float64(tentative)
	if base < 0 {
		base = 0
	}

	cover := 1.0
	if expectedMin > 0 {
		cover = float64(total) / float64(expectedMin)
		if cover > 1.0 {
			cover = 1.0
		}
	}

	return roundTo1(base * cover)
}

func getExpectedMin(changeType string) int {
	if v, ok := expectedMin[changeType]; ok {
		return v
	}
	return 3 // default for docs/test/ci/chore
}

func getGateThreshold(changeType string) float64 {
	if v, ok := gateThresholds[changeType]; ok {
		return v
	}
	return 3.0 // default
}

func roundTo1(f float64) float64 {
	return math.Round(f*10) / 10
}
