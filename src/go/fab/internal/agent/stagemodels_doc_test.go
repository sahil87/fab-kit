package agent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
)

const (
	stageModelsDocPath = "docs/specs/stage-models.md"
	defaultRoleHeading = "### Default role profiles"
	stageRoleHeading   = "## The fixed stage → role mapping"
	presetHeading      = "### Agent presets"
)

// TestDocTablesMatchAgentMaps guards against drift between the canonical built-in
// data (defaults.yaml, composed by DefaultProfile) and the fab-owned stageRoles map
// on one side, and their mirror tables in docs/specs/stage-models.md on the other.
// The code side is canonical; this test fails if a doc table disagrees with the
// resolved profile/role for any role or stage, or if a doc table covers a different
// set than the code knows about. Mirrors internal/score's
// TestDocTablesMatchScoringMaps (change-types.md ↔ score.go).
func TestDocTablesMatchAgentMaps(t *testing.T) {
	docPath := findDocFile(t, stageModelsDocPath)

	// Direction 1 — the default role table: { role → (CLI, model, effort) }.
	roleTable := parseRoleProfileTable(t, docPath, defaultRoleHeading)
	assertCoversSet(t, "Default role profiles", keys(roleTable), RoleNames())
	for _, role := range RoleNames() {
		t.Run("role/"+role, func(t *testing.T) {
			want, _ := DefaultProfile(role)
			got := roleTable[role]
			if got.CLI != want.CLI || got.Model != want.Model || got.Effort != want.Effort {
				t.Errorf("stage-models.md default[%s] = {%s, %s, %s}, code resolves {%s, %s, %s} (doc drifted)",
					role, got.CLI, got.Model, got.Effort, want.CLI, want.Model, want.Effort)
			}
		})
	}

	// Direction 2 — the stage→role table: { stage → role }.
	stageTable := parse2ColTable(t, docPath, stageRoleHeading)
	assertCoversSet(t, "Stage → role mapping", keys2(stageTable), StageNames())
	for _, stage := range StageNames() {
		t.Run("stage/"+stage, func(t *testing.T) {
			want, _ := RoleForStage(stage)
			if got := stageTable[stage]; got != want {
				t.Errorf("stage-models.md mapping[%s] = %q, code stageRoles = %q (doc drifted)", stage, got, want)
			}
		})
	}

	// Direction 3 — the built-in preset table: { preset → fields it sets }, against
	// config.BuiltinAgentPresets (pushed from defaults.yaml).
	presetTable := parsePresetTable(t, docPath, presetHeading)
	builtin := make([]string, 0, len(config.BuiltinAgentPresets))
	for name := range config.BuiltinAgentPresets {
		builtin = append(builtin, name)
	}
	assertCoversSet(t, "Built-in presets", mapKeys(presetTable), builtin)
	for _, name := range builtin {
		t.Run("preset/"+name, func(t *testing.T) {
			want := presetFieldSet(config.BuiltinAgentPresets[name])
			got := presetTable[name]
			if len(got) != len(want) {
				t.Errorf("stage-models.md preset[%s] sets %v, code sets %v (doc drifted)", name, got, want)
				return
			}
			for k, v := range want {
				if got[k] != v {
					t.Errorf("stage-models.md preset[%s] %s = %q, code %q (doc drifted)", name, k, got[k], v)
				}
			}
		})
	}
}

// presetFieldSet flattens a preset to { dotted field → value } (description excluded:
// the doc table lists only what a preset sets on the agent.* tier).
func presetFieldSet(p config.AgentPreset) map[string]string {
	out := map[string]string{}
	if p.Session != "" {
		out["session"] = p.Session
	}
	if p.Workers != "" {
		out["workers"] = p.Workers
	}
	for role, rp := range p.Profiles {
		for leaf, v := range map[string]string{"cli": rp.CLI, "model": rp.Model, "effort": rp.Effort} {
			if v != "" {
				out["profiles."+role+"."+leaf] = v
			}
		}
	}
	return out
}

// parsePresetTable reads the `| Preset | Sets |` table: the name from column one and
// every backticked `field: value` token of column two.
func parsePresetTable(t *testing.T, docPath, heading string) map[string]map[string]string {
	t.Helper()
	out := map[string]map[string]string{}
	for _, row := range tableRowsUnder(t, docPath, heading) {
		if len(row) < 2 {
			continue
		}
		fields := map[string]string{}
		for _, tok := range strings.Split(row[1], "`") {
			if k, v, ok := strings.Cut(tok, ": "); ok {
				fields[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
		out[cleanCell(row[0])] = fields
	}
	return out
}

func mapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
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

// parseRoleProfileTable extracts the first pipe-delimited 4-column table
// (Role | CLI | Model | Effort) under the given heading: { role → Profile }.
func parseRoleProfileTable(t *testing.T, docPath, heading string) map[string]Profile {
	t.Helper()
	rows := tableRowsUnder(t, docPath, heading)
	result := make(map[string]Profile)
	for _, cols := range rows {
		if len(cols) < 4 {
			continue
		}
		role := cleanCell(cols[0])
		if role == "" {
			continue
		}
		result[role] = Profile{
			CLI:    cleanCell(cols[1]),
			Model:  cleanCell(cols[2]),
			Effort: cleanCell(cols[3]),
		}
	}
	if len(result) == 0 {
		t.Fatalf("no 4-column table rows found under heading %q in %s", heading, docPath)
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
