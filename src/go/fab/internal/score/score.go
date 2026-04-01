package score

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/log"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// Penalty weights per decision grade.
const (
	wCertain    = 0.0
	wConfident  = 0.3
	wTentative  = 1.0
)

// Expected minimum decisions by {stage, type}.
var expectedMinIntake = map[string]int{
	"feat": 5, "refactor": 4, "fix": 3,
}
var expectedMinSpec = map[string]int{
	"feat": 7, "refactor": 6, "fix": 5,
}

// Gate thresholds by change type.
var gateThresholds = map[string]float64{
	"fix": 2.0, "feat": 3.0, "refactor": 3.0,
	"docs": 2.0, "test": 2.0, "ci": 2.0, "chore": 2.0,
}

var scoresRegex = regexp.MustCompile(`S:(\d+)\s+R:(\d+)\s+A:(\d+)\s+D:(\d+)`)

// GradeCount holds parsed assumption counts.
type GradeCount struct {
	Certain    int
	Confident  int
	Tentative  int
	Unresolved int
	HasFuzzy   bool
	DimCount   int
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
	Certain    int
	Confident  int
	Tentative  int
	Unresolved int
	Score      float64
	Delta      string
	HasFuzzy   bool
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

	var scoreFile string
	var threshold float64
	if stage == "intake" {
		scoreFile = filepath.Join(changeDir, "intake.md")
		threshold = 3.0
	} else {
		scoreFile = filepath.Join(changeDir, "spec.md")
		threshold = getGateThreshold(changeType)
	}

	if _, err := os.Stat(scoreFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s not found in %s", filepath.Base(scoreFile), changeDir)
	}

	gc := countGrades(scoreFile)
	total := gc.Certain + gc.Confident + gc.Tentative + gc.Unresolved
	expectedMin := getExpectedMin(stage, changeType)
	score := computeScore(gc.Certain, gc.Confident, gc.Tentative, gc.Unresolved, total, expectedMin)

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

// Compute runs the normal scoring mode.
func Compute(fabRoot, changeArg, stage string) (*ScoreResult, error) {
	changeDir, err := resolve.ToAbsDir(fabRoot, changeArg)
	if err != nil {
		return nil, err
	}

	statusPath := filepath.Join(changeDir, ".status.yaml")

	var scoreFile string
	if stage == "intake" {
		scoreFile = filepath.Join(changeDir, "intake.md")
		if _, err := os.Stat(scoreFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("intake.md required for scoring at intake stage")
		}
	} else {
		scoreFile = filepath.Join(changeDir, "spec.md")
		if _, err := os.Stat(scoreFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("spec.md required for scoring")
		}
	}

	// Load status file once for change type, previous score, and writing back
	statusFile, loadErr := sf.Load(statusPath)

	changeType := "feat"
	prevScore := 0.0
	if loadErr == nil {
		if ct := statusFile.ChangeType; ct != "" && ct != "null" {
			changeType = ct
		}
		prevScore = statusFile.Confidence.Score
	}

	expectedMin := getExpectedMin(stage, changeType)

	gc := countGrades(scoreFile)
	total := gc.Certain + gc.Confident + gc.Tentative + gc.Unresolved
	score := computeScore(gc.Certain, gc.Confident, gc.Tentative, gc.Unresolved, total, expectedMin)

	// Compute dimension means
	var meanS, meanR, meanA, meanD float64
	if gc.DimCount > 0 {
		meanS = roundTo1(float64(gc.SumS) / float64(gc.DimCount))
		meanR = roundTo1(float64(gc.SumR) / float64(gc.DimCount))
		meanA = roundTo1(float64(gc.SumA) / float64(gc.DimCount))
		meanD = roundTo1(float64(gc.SumD) / float64(gc.DimCount))
	}

	delta := score - prevScore
	deltaStr := fmt.Sprintf("%+.1f", delta)

	// Write to .status.yaml
	if loadErr == nil {
		indicative := stage == "intake"
		if gc.HasFuzzy {
			_ = status.SetConfidenceFuzzy(statusFile, statusPath, gc.Certain, gc.Confident, gc.Tentative, gc.Unresolved, score, meanS, meanR, meanA, meanD, indicative)
		} else {
			_ = status.SetConfidence(statusFile, statusPath, gc.Certain, gc.Confident, gc.Tentative, gc.Unresolved, score, indicative)
		}

		folder := filepath.Base(changeDir)
		_ = log.ConfidenceLog(fabRoot, folder, score, deltaStr, "calc-score")
	}

	return &ScoreResult{
		Certain:    gc.Certain,
		Confident:  gc.Confident,
		Tentative:  gc.Tentative,
		Unresolved: gc.Unresolved,
		Score:      score,
		Delta:      deltaStr,
		HasFuzzy:   gc.HasFuzzy,
		MeanS:      meanS,
		MeanR:      meanR,
		MeanA:      meanA,
		MeanD:      meanD,
	}, nil
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

func countGrades(file string) GradeCount {
	f, err := os.Open(file)
	if err != nil {
		return GradeCount{}
	}
	defer f.Close()

	gc := GradeCount{}
	inSection := false
	headerSeen := false
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

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

func getExpectedMin(stage, changeType string) int {
	if stage == "intake" {
		if v, ok := expectedMinIntake[changeType]; ok {
			return v
		}
		return 2 // default
	}
	if v, ok := expectedMinSpec[changeType]; ok {
		return v
	}
	return 3 // default
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
