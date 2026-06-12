package score

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
)

// canonicalChangeTypes is the authoritative set of change types. It reuses
// status.ValidChangeTypes (the same list the change-type validation layer
// enforces) rather than redeclaring the 7 types here — keeping a single source
// so adding a type updates validation and this drift test together.
var canonicalChangeTypes = status.ValidChangeTypes

const (
	docRelPath           = "docs/specs/change-types.md"
	expectedMinHeading   = "## Expected Minimum Decisions"
	gateThresholdHeading = "## Gate Thresholds"
)

// TestDocTablesMatchScoringMaps guards against drift between the canonical Go
// maps (expectedMin, gateThresholds) and their mirror tables in
// docs/specs/change-types.md. The code maps are canonical; this test fails if
// the doc table disagrees with the resolved getExpectedMin/getGateThreshold
// values for any of the 7 canonical change types, or if the doc covers a
// different set of types than the code knows about.
func TestDocTablesMatchScoringMaps(t *testing.T) {
	docPath := findDocFile(t, docRelPath)

	expMinDoc := parseChangeTypeTable(t, docPath, expectedMinHeading)
	gateDoc := parseChangeTypeTable(t, docPath, gateThresholdHeading)

	// Direction 1: the doc must cover exactly the canonical type set in each
	// table — catches a type added/removed/renamed in only one place.
	assertCoversCanonicalTypes(t, "Expected Minimum Decisions", expMinDoc)
	assertCoversCanonicalTypes(t, "Gate Thresholds", gateDoc)

	// Direction 2: per-type values must match the resolved code values. We
	// compare against getExpectedMin/getGateThreshold (not raw map membership)
	// because the maps omit default-valued types while the doc lists all 7.
	for _, ct := range canonicalChangeTypes {
		t.Run(ct, func(t *testing.T) {
			gotMin, err := strconv.Atoi(expMinDoc[ct])
			if err != nil {
				t.Fatalf("change-types.md expected_min[%s]=%q is not an int: %v", ct, expMinDoc[ct], err)
			}
			if want := getExpectedMin(ct); gotMin != want {
				t.Errorf("change-types.md expected_min[%s]=%d, code getExpectedMin=%d (doc drifted)", ct, gotMin, want)
			}

			gotGate, err := strconv.ParseFloat(gateDoc[ct], 64)
			if err != nil {
				t.Fatalf("change-types.md gate[%s]=%q is not a float: %v", ct, gateDoc[ct], err)
			}
			if want := getGateThreshold(ct); gotGate != want {
				t.Errorf("change-types.md gate[%s]=%.1f, code getGateThreshold=%.1f (doc drifted)", ct, gotGate, want)
			}
		})
	}
}

// assertCoversCanonicalTypes fails if the parsed table covers a different set of
// change types than canonicalChangeTypes (order-independent).
func assertCoversCanonicalTypes(t *testing.T, tableName string, parsed map[string]string) {
	t.Helper()

	got := make([]string, 0, len(parsed))
	for ct := range parsed {
		got = append(got, ct)
	}
	want := append([]string(nil), canonicalChangeTypes...)
	sort.Strings(got)
	sort.Strings(want)

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("%s table covers types %v, want exactly %v (type set drifted)", tableName, got, want)
	}
}

// findDocFile resolves a repo-relative path (e.g. docs/specs/change-types.md) by
// walking up from the test's working directory until the file is found. Go runs
// tests with the working directory set to the package dir, so the repo root is
// several levels up; walking up is robust to layout changes (unlike a fixed
// ../../../../../ depth count). Fails with a clear message if not found.
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
			// Reached the filesystem root without finding the file.
			t.Fatalf("could not locate %q by walking up from %q to the filesystem root", relPath, mustGetwd(t))
		}
		dir = parent
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		return "(unknown)"
	}
	return wd
}

// parseChangeTypeTable scans change-types.md line-by-line (bufio.Scanner +
// pipe-split — no markdown library, per Constitution Principle I) and
// extracts the pipe-delimited table under the given
// section heading. It anchors on the heading rather than the column names because
// both tables share a "Type" first column. Returns {type → raw value string}; the
// caller converts to int/float64 at comparison time.
func parseChangeTypeTable(t *testing.T, docPath, heading string) map[string]string {
	t.Helper()

	f, err := os.Open(docPath)
	if err != nil {
		t.Fatalf("open %s: %v", docPath, err)
	}
	defer f.Close()

	result := make(map[string]string)
	inSection := false
	headerSeen := false
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, heading) {
			inSection = true
			headerSeen = false
			continue
		}
		if !inSection {
			continue
		}
		// A new heading ends the section.
		if strings.HasPrefix(line, "## ") {
			break
		}

		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}

		// First pipe row is the column header (| Type | ... |); skip it.
		if !headerSeen {
			headerSeen = true
			continue
		}
		// Skip the |---|---| separator row.
		if isTableSeparator(trimmed) {
			continue
		}

		cols := strings.Split(trimmed, "|")
		// A pipe-bounded row splits into: ["", col1, col2, ""], so a 2-column
		// table needs at least 4 parts.
		if len(cols) < 4 {
			continue
		}

		changeType := cleanCell(cols[1])
		value := cleanCell(cols[2])
		if changeType == "" {
			continue
		}
		result[changeType] = value
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", docPath, err)
	}
	if len(result) == 0 {
		t.Fatalf("no table rows found under heading %q in %s", heading, docPath)
	}

	return result
}

// cleanCell strips surrounding whitespace and backticks from a markdown table cell.
func cleanCell(s string) string {
	return strings.Trim(strings.TrimSpace(s), "`")
}
