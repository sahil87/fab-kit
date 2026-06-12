package statusfile

// Golden byte-stability tests for the yaml-mediated .status.yaml write path
// (260612-tb6f, F41). The .status.yaml emit format is a documented stability
// surface: skills and orchestrators parse it, and `fab memory-index`-style
// byte-stability is the contract that keeps diffs quiet. These tests pin the
// exact bytes yaml.v3 produces for a representative, fully-populated document
// so that any yaml-library change (e.g., a goccy/go-yaml evaluation) has an
// objective parity arbiter: the swap is admissible only if these tests still
// pass byte-for-byte.

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// goldenInputYAML is a representative fully-evolved .status.yaml exercising
// every encoder in this package: block sequences (issues, prs), the progress
// mapping, plan, fuzzy confidence with dimensions, flow-style stage_metrics
// (full and sparse entries), and true_impact with both optional sub-blocks.
const goldenInputYAML = `id: ab12
name: 260601-ab12-golden-fixture
created: "2026-06-01T08:00:00Z"
created_by: fab-new
change_type: feat
issues:
  - DEV-1042
progress:
  intake: done
  apply: done
  review: done
  hydrate: done
  ship: done
  review-pr: active
plan:
  generated: true
  task_count: 7
  acceptance_count: 12
  acceptance_completed: 12
confidence:
  certain: 5
  confident: 2
  tentative: 1
  unresolved: 0
  score: 3.4
  fuzzy: true
  dimensions:
    signal: 82.5
    reversibility: 74.0
    competence: 88.0
    disambiguation: 71.5
stage_metrics:
  intake: {started_at: "2026-06-01T08:00:00Z", driver: fab-new, iterations: 1, completed_at: "2026-06-01T08:30:00Z"}
  apply: {started_at: "2026-06-01T08:30:00Z", driver: fab-continue, iterations: 2, completed_at: "2026-06-01T10:00:00Z"}
  review: {iterations: 3}
prs:
  - https://github.com/example/repo/pull/123
true_impact:
  added: 120
  deleted: 30
  net: 90
  excluding:
    added: 100
    deleted: 20
    net: 80
  tests:
    added: 60
    deleted: 5
    net: 55
  computed_at: "2026-06-01T10:00:00Z"
  computed_at_stage: apply
last_updated: "2026-06-01T10:00:00Z"
`

// goldenSavedYAML is the exact byte output of Load(goldenInputYAML) → Save,
// with the freshly-written last_updated value normalized to a fixed token.
// Captured against gopkg.in/yaml.v3 v3.0.1 — the current pinned library.
// Do NOT regenerate this constant to make a failing test pass after a yaml
// library change: a mismatch means the candidate library is NOT byte-parity
// compatible (constitution VII — the pinned format is the spec).
// Note the 4-space indentation: that is yaml.v3's default emit style and is
// what every .status.yaml in the wild carries after its first Save.
const goldenSavedYAML = `id: ab12
name: 260601-ab12-golden-fixture
created: "2026-06-01T08:00:00Z"
created_by: fab-new
change_type: feat
issues:
    - DEV-1042
progress:
    intake: done
    apply: done
    review: done
    hydrate: done
    ship: done
    review-pr: active
plan:
    generated: true
    task_count: 7
    acceptance_count: 12
    acceptance_completed: 12
confidence:
    certain: 5
    confident: 2
    tentative: 1
    unresolved: 0
    score: 3.4
    fuzzy: true
    dimensions:
        signal: 82.5
        reversibility: 74.0
        competence: 88.0
        disambiguation: 71.5
stage_metrics:
    intake: {started_at: "2026-06-01T08:00:00Z", driver: fab-new, iterations: 1, completed_at: "2026-06-01T08:30:00Z"}
    apply: {started_at: "2026-06-01T08:30:00Z", driver: fab-continue, iterations: 2, completed_at: "2026-06-01T10:00:00Z"}
    review: {iterations: 3}
prs:
    - https://github.com/example/repo/pull/123
true_impact:
    added: 120
    deleted: 30
    net: 90
    excluding:
        added: 100
        deleted: 20
        net: 80
    tests:
        added: 60
        deleted: 5
        net: 55
    computed_at: "2026-06-01T10:00:00Z"
    computed_at_stage: apply
last_updated: <TIMESTAMP>
`

// lastUpdatedRe matches the freshly-stamped last_updated line. The value is
// non-deterministic (Save stamps now) but its STYLE is part of the contract:
// a double-quoted RFC3339 UTC timestamp.
var lastUpdatedRe = regexp.MustCompile(`(?m)^last_updated: "\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z"$`)

// normalizeTimestamp replaces the (style-validated) last_updated line value
// with a fixed token so the remaining bytes can be compared exactly.
func normalizeTimestamp(t *testing.T, data []byte) string {
	t.Helper()
	if !lastUpdatedRe.Match(data) {
		t.Fatalf("last_updated line missing or not a double-quoted RFC3339 UTC timestamp:\n%s", data)
	}
	return lastUpdatedRe.ReplaceAllString(string(data), "last_updated: <TIMESTAMP>")
}

func writeGoldenFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".status.yaml")
	if err := os.WriteFile(path, []byte(goldenInputYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestSave_GoldenByteStability pins the exact byte output of a load→save
// round trip for a fully-populated document.
func TestSave_GoldenByteStability(t *testing.T) {
	path := writeGoldenFixture(t)
	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := sf.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := normalizeTimestamp(t, data)
	if got != goldenSavedYAML {
		t.Errorf("Save output deviates from golden bytes.\n--- got ---\n%s\n--- want ---\n%s", got, goldenSavedYAML)
	}
}

// TestSave_RoundTripIdempotent asserts that re-loading the saved file and
// saving again produces byte-identical output (modulo the timestamp value):
// the emit format is a fixed point of load→save.
func TestSave_RoundTripIdempotent(t *testing.T) {
	path := writeGoldenFixture(t)

	sf1, err := Load(path)
	if err != nil {
		t.Fatalf("Load 1: %v", err)
	}
	if err := sf1.Save(path); err != nil {
		t.Fatalf("Save 1: %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	sf2, err := Load(path)
	if err != nil {
		t.Fatalf("Load 2: %v", err)
	}
	if err := sf2.Save(path); err != nil {
		t.Fatalf("Save 2: %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if normalizeTimestamp(t, first) != normalizeTimestamp(t, second) {
		t.Errorf("load→save is not a fixed point.\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// TestSave_GoldenMinimalDocument pins the emit format for the minimal/empty
// shapes: empty flow-style sequences and maps ([], {}) and absent optional
// blocks (no fuzzy confidence, no true_impact).
func TestSave_GoldenMinimalDocument(t *testing.T) {
	const minimalInput = `id: cd34
name: 260601-cd34-minimal
created: "2026-06-01T08:00:00Z"
created_by: test
change_type: docs
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
last_updated: "2026-06-01T08:00:00Z"
`
	const minimalGolden = `id: cd34
name: 260601-cd34-minimal
created: "2026-06-01T08:00:00Z"
created_by: test
change_type: docs
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
last_updated: <TIMESTAMP>
`
	path := filepath.Join(t.TempDir(), ".status.yaml")
	if err := os.WriteFile(path, []byte(minimalInput), 0o644); err != nil {
		t.Fatal(err)
	}
	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := sf.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := normalizeTimestamp(t, data)
	if got != minimalGolden {
		t.Errorf("minimal Save output deviates from golden bytes.\n--- got ---\n%s\n--- want ---\n%s", got, minimalGolden)
	}
}
