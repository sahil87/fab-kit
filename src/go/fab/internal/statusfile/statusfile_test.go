package statusfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testYAML = `id: te1t
name: 260305-test-1-sample-change
created: "2026-03-05T12:00:00+05:30"
created_by: test-user
change_type: feat
issues: []
progress:
  intake: done
  apply: active
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
  certain: 3
  confident: 1
  tentative: 0
  unresolved: 0
  score: 4.7
stage_metrics:
  intake: {started_at: "2026-03-05T12:00:00+05:30", driver: fab-new, iterations: 1, completed_at: "2026-03-05T12:01:00+05:30"}
  apply: {started_at: "2026-03-05T12:01:00+05:30", driver: fab-continue, iterations: 1}
prs: []
last_updated: "2026-03-05T12:01:00+05:30"
`

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	os.WriteFile(path, []byte(testYAML), 0644)

	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if sf.ID != "te1t" {
		t.Errorf("expected id 'te1t', got '%s'", sf.ID)
	}
	if sf.Name != "260305-test-1-sample-change" {
		t.Errorf("expected name '260305-test-1-sample-change', got '%s'", sf.Name)
	}
	if sf.ChangeType != "feat" {
		t.Errorf("expected change_type 'feat', got '%s'", sf.ChangeType)
	}
	if sf.GetProgress("intake") != "done" {
		t.Errorf("expected intake done, got '%s'", sf.GetProgress("intake"))
	}
	if sf.GetProgress("apply") != "active" {
		t.Errorf("expected apply active, got '%s'", sf.GetProgress("apply"))
	}
	if sf.GetProgress("review") != "pending" {
		t.Errorf("expected review pending, got '%s'", sf.GetProgress("review"))
	}
	if sf.Confidence.Score != 4.7 {
		t.Errorf("expected score 4.7, got %f", sf.Confidence.Score)
	}
	if sf.Plan.Generated != false {
		t.Error("expected generated false")
	}

	// Test stage metrics
	sm, ok := sf.StageMetrics["intake"]
	if !ok {
		t.Fatal("expected intake stage metrics")
	}
	if sm.Iterations != 1 {
		t.Errorf("expected iterations 1, got %d", sm.Iterations)
	}
	if sm.Driver != "fab-new" {
		t.Errorf("expected driver fab-new, got '%s'", sm.Driver)
	}

	// Test SetProgress
	sf.SetProgress("apply", "done")
	if sf.GetProgress("apply") != "done" {
		t.Errorf("expected apply done after set, got '%s'", sf.GetProgress("apply"))
	}

	// Test Save (round-trip)
	outPath := filepath.Join(dir, ".status-out.yaml")
	if err := sf.Save(outPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify
	sf2, err := Load(outPath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if sf2.ID != sf.ID {
		t.Errorf("round-trip id mismatch: %s vs %s", sf2.ID, sf.ID)
	}
	if sf2.Name != sf.Name {
		t.Errorf("round-trip name mismatch: %s vs %s", sf2.Name, sf.Name)
	}
	if sf2.GetProgress("apply") != "done" {
		t.Errorf("round-trip apply state mismatch: got '%s'", sf2.GetProgress("apply"))
	}
	if sf2.Confidence.Score != 4.7 {
		t.Errorf("round-trip score mismatch: %f", sf2.Confidence.Score)
	}
}

func TestGetProgressMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	os.WriteFile(path, []byte(testYAML), 0644)

	sf, _ := Load(path)
	pm := sf.GetProgressMap()

	if len(pm) != 6 {
		t.Errorf("expected 6 stages, got %d", len(pm))
	}

	// Verify pipeline order
	expected := []string{"intake", "apply", "review", "hydrate", "ship", "review-pr"}
	for i, ss := range pm {
		if ss.Stage != expected[i] {
			t.Errorf("stage %d: expected '%s', got '%s'", i, expected[i], ss.Stage)
		}
	}
}

// TestOrphanSpecKeyTolerated verifies R-STAGE-3: a .status.yaml carrying a
// leftover progress.spec key (un-migrated file) loads without error, and
// GetProgressMap omits the orphan key (it derives from StageOrder, which no
// longer contains spec).
func TestOrphanSpecKeyTolerated(t *testing.T) {
	const orphanYAML = `id: orph
name: 260601-orphan-spec-fixture
created: "2026-06-01T00:00:00Z"
created_by: test-user
change_type: refactor
issues: []
progress:
  intake: done
  spec: done
  apply: active
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
last_updated: "2026-06-01T00:00:00Z"
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(orphanYAML), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load of file with orphan progress.spec failed: %v", err)
	}

	pm := sf.GetProgressMap()
	if len(pm) != 6 {
		t.Errorf("GetProgressMap should omit orphan spec key: expected 6 stages, got %d", len(pm))
	}
	for _, ss := range pm {
		if ss.Stage == "spec" {
			t.Error("GetProgressMap should not include the orphan spec stage")
		}
	}

	// The raw key is still readable directly (passthrough) but not part of the
	// canonical pipeline view.
	if sf.GetProgress("spec") != "done" {
		t.Errorf("orphan spec key should still be readable via GetProgress, got '%s'", sf.GetProgress("spec"))
	}
}

func TestStageNumber(t *testing.T) {
	if StageNumber("intake") != 1 {
		t.Error("intake should be 1")
	}
	if StageNumber("review-pr") != 6 {
		t.Error("review-pr should be 6")
	}
	if StageNumber("apply") != 2 {
		t.Error("apply should be 2")
	}
	if StageNumber("bogus") != 0 {
		t.Error("bogus should be 0")
	}
}

func TestPlanRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	os.WriteFile(path, []byte(testYAML), 0644)

	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Mutate plan fields and save.
	sf.Plan.Generated = true
	sf.Plan.TaskCount = 5
	sf.Plan.AcceptanceCount = 9
	sf.Plan.AcceptanceCompleted = 3

	outPath := filepath.Join(dir, ".status-plan.yaml")
	if err := sf.Save(outPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	sf2, err := Load(outPath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if !sf2.Plan.Generated {
		t.Error("plan.generated round-trip lost")
	}
	if sf2.Plan.TaskCount != 5 {
		t.Errorf("plan.task_count round-trip: got %d, want 5", sf2.Plan.TaskCount)
	}
	if sf2.Plan.AcceptanceCount != 9 {
		t.Errorf("plan.acceptance_count round-trip: got %d, want 9", sf2.Plan.AcceptanceCount)
	}
	if sf2.Plan.AcceptanceCompleted != 3 {
		t.Errorf("plan.acceptance_completed round-trip: got %d, want 3", sf2.Plan.AcceptanceCompleted)
	}
}

// TestLegacyChecklistFileSavesPlanBlock guards a regression where saving a
// pre-1.9.0 .status.yaml (which has a `checklist:` block but no `plan:` block)
// silently dropped Plan struct mutations. The fix upgrades the raw schema on
// Load so syncToRaw has a `plan:` node to write into.
func TestLegacyChecklistFileSavesPlanBlock(t *testing.T) {
	const legacyYAML = `id: lgcy
name: 260423-legacy-fixture
created: "2026-04-23T05:02:32Z"
created_by: test-user
change_type: fix
issues: []
progress:
    intake: done
    spec: done
    tasks: done
    apply: active
    review: pending
    hydrate: pending
    ship: pending
    review-pr: pending
checklist:
    generated: true
    path: checklist.md
    completed: 5
    total: 9
confidence:
    certain: 1
    confident: 0
    tentative: 0
    unresolved: 0
    score: 4.7
stage_metrics: {}
prs: []
last_updated: "2026-04-23T05:02:32Z"
`

	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(legacyYAML), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	// 1. Load the legacy file.
	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Migration mapping should populate Plan from the checklist: block.
	if !sf.Plan.Generated {
		t.Error("expected Plan.Generated=true (migrated from checklist.generated)")
	}
	if sf.Plan.AcceptanceCompleted != 5 {
		t.Errorf("expected Plan.AcceptanceCompleted=5 (migrated from checklist.completed), got %d", sf.Plan.AcceptanceCompleted)
	}
	if sf.Plan.AcceptanceCount != 9 {
		t.Errorf("expected Plan.AcceptanceCount=9 (migrated from checklist.total), got %d", sf.Plan.AcceptanceCount)
	}

	// 2. Mutate plan fields and save.
	sf.Plan.TaskCount = 7
	sf.Plan.AcceptanceCount = 12

	if err := sf.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 3. Re-load — assert Plan fields persisted.
	sf2, err := Load(path)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if sf2.Plan.TaskCount != 7 {
		t.Errorf("Plan.TaskCount round-trip: got %d, want 7", sf2.Plan.TaskCount)
	}
	if sf2.Plan.AcceptanceCount != 12 {
		t.Errorf("Plan.AcceptanceCount round-trip: got %d, want 12", sf2.Plan.AcceptanceCount)
	}
	if sf2.Plan.AcceptanceCompleted != 5 {
		t.Errorf("Plan.AcceptanceCompleted round-trip: got %d, want 5", sf2.Plan.AcceptanceCompleted)
	}
	if !sf2.Plan.Generated {
		t.Error("Plan.Generated round-trip lost")
	}

	// 4. Inspect raw bytes — plan: block must be present, legacy checklist: must be absent.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	rawStr := string(raw)
	if !strings.Contains(rawStr, "plan:") {
		t.Errorf("saved file missing `plan:` block. content:\n%s", rawStr)
	}
	if strings.Contains(rawStr, "checklist:") {
		t.Errorf("saved file still contains legacy `checklist:` block. content:\n%s", rawStr)
	}
	if !strings.Contains(rawStr, "task_count: 7") {
		t.Errorf("saved file missing task_count: 7. content:\n%s", rawStr)
	}
}

// TestTrueImpactTestsRoundTrip verifies the new `tests` sub-block round-trips
// through Load→Save→Load and is positioned after `excluding` / before
// `computed_at` in the emitted YAML.
func TestTrueImpactTestsRoundTrip(t *testing.T) {
	const yamlWithTests = testYAML + `true_impact:
  added: 612
  deleted: 38
  net: 574
  excluding:
    added: 540
    deleted: 38
    net: 502
  tests:
    added: 400
    deleted: 0
    net: 400
  computed_at: "2026-05-30T00:00:00Z"
  computed_at_stage: apply
`

	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(yamlWithTests), 0644); err != nil {
		t.Fatal(err)
	}

	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if sf.TrueImpact == nil {
		t.Fatal("expected TrueImpact to be parsed")
	}
	if sf.TrueImpact.Tests == nil {
		t.Fatal("expected TrueImpact.Tests to be parsed")
	}
	if sf.TrueImpact.Tests.Added != 400 || sf.TrueImpact.Tests.Net != 400 || sf.TrueImpact.Tests.Deleted != 0 {
		t.Errorf("tests pair decoded wrong: %+v", sf.TrueImpact.Tests)
	}

	outPath := filepath.Join(dir, ".status-out.yaml")
	if err := sf.Save(outPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Re-load and verify the tests pair survived.
	sf2, err := Load(outPath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if sf2.TrueImpact == nil || sf2.TrueImpact.Tests == nil {
		t.Fatal("tests sub-block lost on round-trip")
	}
	if sf2.TrueImpact.Tests.Net != 400 {
		t.Errorf("round-trip tests.net = %d, want 400", sf2.TrueImpact.Tests.Net)
	}

	// Inspect raw bytes for ordering: tests must appear after excluding and
	// before computed_at.
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	rawStr := string(raw)
	exIdx := strings.Index(rawStr, "excluding:")
	tIdx := strings.Index(rawStr, "tests:")
	cIdx := strings.Index(rawStr, "computed_at:")
	if exIdx < 0 || tIdx < 0 || cIdx < 0 {
		t.Fatalf("missing expected keys in:\n%s", rawStr)
	}
	if !(exIdx < tIdx && tIdx < cIdx) {
		t.Errorf("expected ordering excluding < tests < computed_at, got %d < %d < %d in:\n%s", exIdx, tIdx, cIdx, rawStr)
	}
}

// TestTrueImpactTestsOmittedWhenNil verifies the lazy-omit posture: a
// TrueImpact with a nil Tests emits no `tests:` key.
func TestTrueImpactTestsOmittedWhenNil(t *testing.T) {
	const yamlNoTests = testYAML + `true_impact:
  added: 50
  deleted: 5
  net: 45
  computed_at: "2026-05-30T00:00:00Z"
  computed_at_stage: apply
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(yamlNoTests), 0644); err != nil {
		t.Fatal(err)
	}

	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if sf.TrueImpact == nil {
		t.Fatal("expected TrueImpact to be parsed")
	}
	if sf.TrueImpact.Tests != nil {
		t.Errorf("expected nil Tests, got %+v", sf.TrueImpact.Tests)
	}

	outPath := filepath.Join(dir, ".status-out.yaml")
	if err := sf.Save(outPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if strings.Contains(string(raw), "tests:") {
		t.Errorf("expected no tests: key when Tests is nil, got:\n%s", raw)
	}
}

func TestNextStage(t *testing.T) {
	if NextStage("intake") != "apply" {
		t.Error("after intake should be apply")
	}
	if NextStage("apply") != "review" {
		t.Error("after apply should be review")
	}
	if NextStage("hydrate") != "ship" {
		t.Error("after hydrate should be ship")
	}
	if NextStage("review-pr") != "" {
		t.Error("after review-pr should be empty")
	}
}

// TestPlanAndChecklistCoexistDropsChecklist guards against a partial-migration
// state where both `plan:` (new, authoritative) and `checklist:` (legacy,
// stale) coexist in the same .status.yaml. Load MUST drop the legacy
// `checklist:` block so it does not survive subsequent Save() calls.
func TestPlanAndChecklistCoexistDropsChecklist(t *testing.T) {
	const mixedYAML = `id: mxd1
name: 260423-mixed-fixture
created: "2026-04-23T05:02:32Z"
created_by: test-user
change_type: fix
issues: []
progress:
    intake: done
    spec: done
    apply: active
    review: pending
    hydrate: pending
    ship: pending
    review-pr: pending
plan:
    generated: true
    task_count: 4
    acceptance_count: 10
    acceptance_completed: 3
checklist:
    generated: true
    path: checklist.md
    completed: 99
    total: 99
confidence:
    certain: 1
    confident: 0
    tentative: 0
    unresolved: 0
    score: 4.7
stage_metrics: {}
prs: []
last_updated: "2026-04-23T05:02:32Z"
`

	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(mixedYAML), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	// 1. Load the mixed file.
	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Plan block is authoritative — its values must be preserved verbatim,
	// not overwritten by the stale checklist block.
	if !sf.Plan.Generated {
		t.Error("expected Plan.Generated=true (from plan: block, authoritative)")
	}
	if sf.Plan.TaskCount != 4 {
		t.Errorf("expected Plan.TaskCount=4 (from plan:), got %d", sf.Plan.TaskCount)
	}
	if sf.Plan.AcceptanceCount != 10 {
		t.Errorf("expected Plan.AcceptanceCount=10 (from plan:), got %d", sf.Plan.AcceptanceCount)
	}
	if sf.Plan.AcceptanceCompleted != 3 {
		t.Errorf("expected Plan.AcceptanceCompleted=3 (from plan:), got %d", sf.Plan.AcceptanceCompleted)
	}

	// 2. Save and re-read raw bytes — the legacy checklist: key MUST NOT
	//    survive.
	if err := sf.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	rawStr := string(raw)
	if !strings.Contains(rawStr, "plan:") {
		t.Errorf("saved file missing `plan:` block. content:\n%s", rawStr)
	}
	if strings.Contains(rawStr, "checklist:") {
		t.Errorf("saved file still contains stale `checklist:` block (should have been dropped on Load). content:\n%s", rawStr)
	}
	if !strings.Contains(rawStr, "task_count: 4") {
		t.Errorf("saved file lost authoritative plan values. content:\n%s", rawStr)
	}
}

// --- Write-time key insertion + SetProgress shape errors (mz4q F07):
// mutations against sparse legacy documents must persist (keys created on
// write), and a malformed progress shape must error instead of silently
// dropping the transition. ---

// sparseYAML mimics a restored pre-0.24.0 archive / hand-edited file: no
// prs:, stage_metrics:, confidence:, plan:, change_type:, and a progress map
// missing the apply stage key.
const sparseYAML = `id: sp4r
name: 260310-sp4r-sparse-change
created: "2026-03-10T12:00:00Z"
progress:
  intake: done
last_updated: "2026-03-10T12:00:00Z"
`

func loadSparse(t *testing.T) (*StatusFile, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(sparseYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return sf, path
}

func TestSparseFile_AddPRPersists(t *testing.T) {
	sf, path := loadSparse(t)

	sf.PRs = append(sf.PRs, "https://github.com/o/r/pull/1")
	if err := sf.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.PRs) != 1 || reloaded.PRs[0] != "https://github.com/o/r/pull/1" {
		t.Errorf("PR write dropped on sparse file: %v", reloaded.PRs)
	}
}

func TestSparseFile_ChangeTypeAndConfidencePersist(t *testing.T) {
	sf, path := loadSparse(t)

	sf.ChangeType = "fix"
	sf.Confidence.Certain = 3
	sf.Confidence.Score = 4.2
	if err := sf.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.ChangeType != "fix" {
		t.Errorf("change_type dropped: %q", reloaded.ChangeType)
	}
	if reloaded.Confidence.Certain != 3 || reloaded.Confidence.Score != 4.2 {
		t.Errorf("confidence dropped: %+v", reloaded.Confidence)
	}
}

func TestSparseFile_StageMetricsAndPlanPersist(t *testing.T) {
	sf, path := loadSparse(t)

	sf.Plan.Generated = true
	sf.Plan.TaskCount = 5
	sf.StageMetrics["apply"] = &StageMetric{StartedAt: "2026-03-10T13:00:00Z", Iterations: 1}
	if err := sf.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !reloaded.Plan.Generated || reloaded.Plan.TaskCount != 5 {
		t.Errorf("plan dropped: %+v", reloaded.Plan)
	}
	sm, ok := reloaded.StageMetrics["apply"]
	if !ok || sm.Iterations != 1 {
		t.Errorf("stage_metrics dropped: %+v", reloaded.StageMetrics)
	}
}

func TestSetProgress_CreatesMissingStageKey(t *testing.T) {
	sf, path := loadSparse(t)

	if err := sf.SetProgress("apply", "active"); err != nil {
		t.Fatalf("SetProgress: %v", err)
	}
	if err := sf.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := reloaded.GetProgress("apply"); got != "active" {
		t.Errorf("missing stage key not created: apply = %q, want active", got)
	}
	if got := reloaded.GetProgress("intake"); got != "done" {
		t.Errorf("existing stage disturbed: intake = %q, want done", got)
	}
}

func TestSetProgress_MalformedProgressErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	noProgress := "id: ab12\nname: x\nlast_updated: \"2026-03-10T12:00:00Z\"\n"
	if err := os.WriteFile(path, []byte(noProgress), 0o644); err != nil {
		t.Fatal(err)
	}
	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	err = sf.SetProgress("apply", "active")
	if err == nil {
		t.Fatal("expected malformed-shape error when progress: is absent")
	}
	if !strings.Contains(err.Error(), "progress is missing or not a mapping") {
		t.Errorf("unexpected error text: %v", err)
	}
}

func TestSparseFile_LastUpdatedInserted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	noLastUpdated := "id: ab12\nname: x\nprogress:\n  intake: done\n"
	if err := os.WriteFile(path, []byte(noLastUpdated), 0o644); err != nil {
		t.Fatal(err)
	}
	sf, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := sf.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.LastUpdated == "" {
		t.Error("last_updated not inserted on a file missing the key")
	}
}

// --- Classified read errors (mz4q F06): only genuine absence reports "not
// found"; other read failures carry the real cause. ---

func TestLoad_NotFoundKeepsFriendlyText(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), ".status.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "status file not found:") {
		t.Errorf("expected friendly not-found text, got: %v", err)
	}
}

func TestLoad_PermissionDeniedClassified(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root — permission bits are not enforced")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.WriteFile(path, []byte(testYAML), 0o000); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
	if strings.Contains(err.Error(), "not found") {
		t.Errorf("permission failure must not masquerade as absence: %v", err)
	}
	if !strings.Contains(err.Error(), "read status file") || !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected cause-bearing read error, got: %v", err)
	}
}

func TestLoad_IsADirectoryClassified(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for directory path")
	}
	if strings.Contains(err.Error(), "not found") {
		t.Errorf("is-a-directory failure must not masquerade as absence: %v", err)
	}
}

func TestLoad_ParseErrorDistinctFromAbsence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".status.yaml")
	// Git merge-conflict markers — the file is git-tracked, so this happens.
	conflicted := "id: ab12\n<<<<<<< HEAD\nname: a\n=======\nname: b\n>>>>>>> theirs\n"
	if err := os.WriteFile(path, []byte(conflicted), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error for conflicted file")
	}
	if strings.Contains(err.Error(), "not found") {
		t.Errorf("corruption must be distinguishable from absence: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid YAML") {
		t.Errorf("expected invalid-YAML classification, got: %v", err)
	}
}
