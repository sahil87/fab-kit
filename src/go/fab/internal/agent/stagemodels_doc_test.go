package agent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
)

const (
	stageModelsDocPath = "docs/specs/stage-models.md"
	defaultTierHeading = "### Default tier profiles"
	stageTierHeading   = "## The fixed stage → tier mapping"
)

// TestDocTablesMatchAgentMaps guards against drift between the canonical Go maps
// (defaultTiers, stageTiers) and their mirror tables in
// docs/specs/stage-models.md. The code maps are canonical; this test fails if a
// doc table disagrees with the resolved profile/tier for any tier or stage, or
// if a doc table covers a different set than the code knows about. Mirrors
// internal/score's TestDocTablesMatchScoringMaps (change-types.md ↔ score.go).
func TestDocTablesMatchAgentMaps(t *testing.T) {
	docPath := findDocFile(t, stageModelsDocPath)

	// Direction 1 — the default tier table: { tier → (model, effort) }.
	tierTable := parseTierProfileTable(t, docPath, defaultTierHeading)
	assertCoversSet(t, "Default tier profiles", keys(tierTable), TierNames())
	for _, tier := range TierNames() {
		t.Run("tier/"+tier, func(t *testing.T) {
			want, _ := DefaultTier(tier)
			got := tierTable[tier]
			if got.Model != want.Model || got.Effort != want.Effort {
				t.Errorf("stage-models.md default[%s] = {%s, %s}, code defaultTiers = {%s, %s} (doc drifted)",
					tier, got.Model, got.Effort, want.Model, want.Effort)
			}
		})
	}

	// Direction 2 — the stage→tier table: { stage → tier }.
	stageTable := parse2ColTable(t, docPath, stageTierHeading)
	assertCoversSet(t, "Stage → tier mapping", keys2(stageTable), StageNames())
	for _, stage := range StageNames() {
		t.Run("stage/"+stage, func(t *testing.T) {
			want, _ := TierForStage(stage)
			if got := stageTable[stage]; got != want {
				t.Errorf("stage-models.md mapping[%s] = %q, code stageTiers = %q (doc drifted)", stage, got, want)
			}
		})
	}
}

func keys(m map[string]Profile) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func keys2(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// assertCoversSet fails if got and want are not the same set (order-independent).
func assertCoversSet(t *testing.T, tableName string, got, want []string) {
	t.Helper()
	g := append([]string(nil), got...)
	w := append([]string(nil), want...)
	sort.Strings(g)
	sort.Strings(w)
	if strings.Join(g, ",") != strings.Join(w, ",") {
		t.Errorf("%s table covers %v, want exactly %v (set drifted)", tableName, g, w)
	}
}

// parse2ColTable extracts the first pipe-delimited 2-column table under the given
// heading: { col1 → col2 }. Line-based, no markdown library (Constitution I).
func parse2ColTable(t *testing.T, docPath, heading string) map[string]string {
	t.Helper()
	rows := tableRowsUnder(t, docPath, heading)
	result := make(map[string]string)
	for _, cols := range rows {
		if len(cols) < 2 {
			continue
		}
		key := cleanCell(cols[0])
		if key == "" {
			continue
		}
		result[key] = cleanCell(cols[1])
	}
	if len(result) == 0 {
		t.Fatalf("no 2-column table rows found under heading %q in %s", heading, docPath)
	}
	return result
}

// parseTierProfileTable extracts the first pipe-delimited 3-column table
// (Tier | Model | Effort) under the given heading: { tier → Profile }.
func parseTierProfileTable(t *testing.T, docPath, heading string) map[string]Profile {
	t.Helper()
	rows := tableRowsUnder(t, docPath, heading)
	result := make(map[string]Profile)
	for _, cols := range rows {
		if len(cols) < 3 {
			continue
		}
		tier := cleanCell(cols[0])
		if tier == "" {
			continue
		}
		result[tier] = Profile{Model: cleanCell(cols[1]), Effort: cleanCell(cols[2])}
	}
	if len(result) == 0 {
		t.Fatalf("no 3-column table rows found under heading %q in %s", heading, docPath)
	}
	return result
}

// tableRowsUnder returns the data rows (header + separator stripped) of the FIRST
// markdown pipe-table appearing under the given heading. Each row is returned as
// its interior cells (the leading/trailing empty splits dropped). The section
// ends at the next heading of the same-or-shallower level, or the first blank gap
// after the table — we stop at the next "## "/"### " heading, which is sufficient
// because each anchored section's first table is the one we want.
func tableRowsUnder(t *testing.T, docPath, heading string) [][]string {
	t.Helper()
	body, err := lines.ReadFileLines(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}

	var rows [][]string
	inSection := false
	headerSeen := false
	tableStarted := false

	for _, line := range body {
		if strings.HasPrefix(line, heading) {
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		// A new heading ends the section.
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			break
		}

		trimmed := strings.TrimSpace(line)
		isPipeRow := strings.HasPrefix(trimmed, "|")

		if !isPipeRow {
			// A blank line after the table has started ends the table; before
			// the table starts, blank/prose lines are skipped.
			if tableStarted {
				break
			}
			continue
		}
		tableStarted = true

		// First pipe row is the column header; skip it.
		if !headerSeen {
			headerSeen = true
			continue
		}
		// Skip the |---|---| separator row.
		if isTableSeparator(trimmed) {
			continue
		}

		parts := strings.Split(trimmed, "|")
		// A pipe-bounded row splits into ["", c1, c2, ..., ""]; drop the
		// leading/trailing empties.
		if len(parts) < 3 {
			continue
		}
		rows = append(rows, parts[1:len(parts)-1])
	}

	if len(rows) == 0 {
		t.Fatalf("no table rows found under heading %q in %s", heading, docPath)
	}
	return rows
}

// findDocFile resolves a repo-relative path by walking up from the test's working
// directory until the file is found (Go runs tests with cwd = package dir). Same
// helper shape as internal/score's findDocFile.
func findDocFile(t *testing.T, relPath string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		candidate := filepath.Join(dir, relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate %q by walking up to the filesystem root", relPath)
		}
		dir = parent
	}
}

// isTableSeparator reports whether a trimmed markdown line is a |---|---| rule.
func isTableSeparator(trimmed string) bool {
	for _, r := range trimmed {
		if r != '|' && r != '-' && r != ':' && r != ' ' {
			return false
		}
	}
	return strings.Contains(trimmed, "-")
}

// cleanCell strips surrounding whitespace and backticks from a table cell.
func cleanCell(s string) string {
	return strings.Trim(strings.TrimSpace(s), "`")
}
