package status

// Tests for the previously-untested persisting mutators and read-side helpers
// (260612-tb6f, F39): SetChangeType, AddIssue, ProgressMap, ProgressLine,
// AllStages, SetConfidence/SetConfidenceFuzzy, CurrentStage, and Advance's
// remaining branches — plus the stage_metrics iterations accumulation
// regression across repeated rework cycles.

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// --- SetChangeType ---

func TestSetChangeType_PersistsValidType(t *testing.T) {
	statusFile, path := loadFixture(t)

	if err := SetChangeType(statusFile, path, "refactor"); err != nil {
		t.Fatalf("SetChangeType: %v", err)
	}
	if statusFile.ChangeType != "refactor" {
		t.Errorf("ChangeType = %q, want refactor", statusFile.ChangeType)
	}

	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.ChangeType != "refactor" {
		t.Errorf("persisted ChangeType = %q, want refactor", reloaded.ChangeType)
	}
}

func TestSetChangeType_InvalidTypeErrorsWithoutWrite(t *testing.T) {
	statusFile, path := loadFixture(t)
	priorUpdated := statusFile.LastUpdated

	err := SetChangeType(statusFile, path, "banana")
	if err == nil {
		t.Fatal("expected invalid change type to error")
	}
	if !strings.Contains(err.Error(), "Invalid change type") {
		t.Errorf("want invalid-type error, got: %v", err)
	}

	// Validation happens before mutation/persist — file untouched.
	reloaded, loadErr := sf.Load(path)
	if loadErr != nil {
		t.Fatalf("reload: %v", loadErr)
	}
	if reloaded.ChangeType != "feat" {
		t.Errorf("on-disk ChangeType = %q, want feat (unchanged)", reloaded.ChangeType)
	}
	if reloaded.LastUpdated != priorUpdated {
		t.Error("last_updated should not be refreshed on a rejected mutation")
	}
}

// --- SetSummary ---

func TestSetSummary_PersistsAndRoundTrips(t *testing.T) {
	statusFile, path := loadFixture(t)
	priorUpdated := statusFile.LastUpdated

	if err := SetSummary(statusFile, path, "added the summary field"); err != nil {
		t.Fatalf("SetSummary: %v", err)
	}
	if statusFile.Summary != "added the summary field" {
		t.Errorf("Summary = %q, want \"added the summary field\"", statusFile.Summary)
	}
	if statusFile.LastUpdated == priorUpdated {
		t.Error("last_updated should be refreshed")
	}

	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Summary != "added the summary field" {
		t.Errorf("persisted Summary = %q, want \"added the summary field\"", reloaded.Summary)
	}
}

func TestSetSummary_EmptyClearsKey(t *testing.T) {
	statusFile, path := loadFixture(t)
	if err := SetSummary(statusFile, path, "temporary"); err != nil {
		t.Fatalf("SetSummary: %v", err)
	}
	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if err := SetSummary(reloaded, path, ""); err != nil {
		t.Fatalf("SetSummary (clear): %v", err)
	}
	final, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload after clear: %v", err)
	}
	if final.Summary != "" {
		t.Errorf("cleared Summary should be empty, got %q", final.Summary)
	}
}

// --- AddIssue ---

func TestAddIssue_AppendsAndPersists(t *testing.T) {
	statusFile, path := loadFixture(t)

	if err := AddIssue(statusFile, path, "DEV-101"); err != nil {
		t.Fatalf("AddIssue: %v", err)
	}
	if err := AddIssue(statusFile, path, "DEV-102"); err != nil {
		t.Fatalf("AddIssue: %v", err)
	}

	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Issues) != 2 || reloaded.Issues[0] != "DEV-101" || reloaded.Issues[1] != "DEV-102" {
		t.Errorf("persisted issues = %v, want [DEV-101 DEV-102]", reloaded.Issues)
	}
}

func TestAddIssue_IdempotentOnDuplicate(t *testing.T) {
	statusFile, path := loadFixture(t)

	if err := AddIssue(statusFile, path, "DEV-101"); err != nil {
		t.Fatalf("AddIssue: %v", err)
	}
	// Duplicate: no second entry, but the call still succeeds (refreshes
	// last_updated like the other idempotent mutators).
	if err := AddIssue(statusFile, path, "DEV-101"); err != nil {
		t.Fatalf("duplicate AddIssue: %v", err)
	}

	if len(statusFile.Issues) != 1 {
		t.Errorf("issues = %v, want exactly one DEV-101 entry", statusFile.Issues)
	}
	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Issues) != 1 {
		t.Errorf("persisted issues = %v, want exactly one entry", reloaded.Issues)
	}
}

// --- ProgressMap / ProgressLine / AllStages / CurrentStage ---

func TestProgressMap_OrderedPairs(t *testing.T) {
	statusFile, _ := loadFixture(t)
	statusFile.SetProgress("intake", "done")
	statusFile.SetProgress("apply", "active")

	pm := ProgressMap(statusFile)
	if len(pm) != len(sf.StageOrder) {
		t.Fatalf("ProgressMap returned %d pairs, want %d", len(pm), len(sf.StageOrder))
	}
	for i, ss := range pm {
		if ss.Stage != sf.StageOrder[i] {
			t.Errorf("pair %d stage = %q, want %q (pipeline order)", i, ss.Stage, sf.StageOrder[i])
		}
	}
	if pm[0].State != "done" || pm[1].State != "active" || pm[2].State != "pending" {
		t.Errorf("states = %v, want done/active/pending...", pm)
	}
}

func TestProgressLine_RendersEachStateGlyph(t *testing.T) {
	statusFile, _ := loadFixture(t)
	statusFile.SetProgress("intake", "done")
	statusFile.SetProgress("apply", "ready")
	statusFile.SetProgress("review", "failed")
	statusFile.SetProgress("hydrate", "skipped")
	statusFile.SetProgress("ship", "active")
	// review-pr stays pending → omitted from the line, suppresses the ✓.

	got := ProgressLine(statusFile)
	want := "intake → apply ◷ → review ✗ → hydrate ⏭ → ship ⏳"
	if got != want {
		t.Errorf("ProgressLine = %q, want %q", got, want)
	}
}

func TestProgressLine_AllPendingIsEmpty(t *testing.T) {
	statusFile, _ := loadFixture(t)
	statusFile.SetProgress("intake", "pending") // fixture starts intake active

	if got := ProgressLine(statusFile); got != "" {
		t.Errorf("ProgressLine for all-pending = %q, want empty", got)
	}
}

func TestProgressLine_CompleteGetsCheckmark(t *testing.T) {
	statusFile, _ := loadFixture(t)
	for _, stage := range sf.StageOrder {
		statusFile.SetProgress(stage, "done")
	}

	got := ProgressLine(statusFile)
	if !strings.HasSuffix(got, " ✓") {
		t.Errorf("ProgressLine for all-done = %q, want trailing ✓", got)
	}
}

func TestAllStages_PipelineOrder(t *testing.T) {
	got := AllStages()
	want := []string{"intake", "apply", "review", "hydrate", "ship", "review-pr"}
	if len(got) != len(want) {
		t.Fatalf("AllStages = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllStages[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCurrentStage_Tiers(t *testing.T) {
	statusFile, _ := loadFixture(t)

	// Tier 1: first active/ready.
	if got := CurrentStage(statusFile); got != "intake" {
		t.Errorf("CurrentStage with intake active = %q, want intake", got)
	}
	statusFile.SetProgress("intake", "done")
	statusFile.SetProgress("apply", "ready")
	if got := CurrentStage(statusFile); got != "apply" {
		t.Errorf("CurrentStage with apply ready = %q, want apply", got)
	}

	// Tier 2: first pending after the last done/skipped.
	statusFile.SetProgress("apply", "done")
	statusFile.SetProgress("review", "skipped")
	if got := CurrentStage(statusFile); got != "hydrate" {
		t.Errorf("CurrentStage fallback = %q, want hydrate", got)
	}

	// Tier 3: all done → review-pr (the documented routing fallback).
	for _, stage := range sf.StageOrder {
		statusFile.SetProgress(stage, "done")
	}
	if got := CurrentStage(statusFile); got != "review-pr" {
		t.Errorf("CurrentStage all-done = %q, want review-pr", got)
	}
}

// --- SetConfidence / SetConfidenceFuzzy ---

func TestSetConfidence_Persists(t *testing.T) {
	statusFile, path := loadFixture(t)

	if err := SetConfidence(statusFile, path, 5, 2, 1, 0, 3.4); err != nil {
		t.Fatalf("SetConfidence: %v", err)
	}
	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	c := reloaded.Confidence
	if c.Certain != 5 || c.Confident != 2 || c.Tentative != 1 || c.Unresolved != 0 || c.Score != 3.4 {
		t.Errorf("persisted confidence = %+v, want 5/2/1/0 score 3.4", c)
	}
	if c.Fuzzy != nil {
		t.Errorf("non-fuzzy SetConfidence should not set fuzzy, got %v", *c.Fuzzy)
	}
}

func TestSetConfidenceFuzzy_PersistsDimensions(t *testing.T) {
	statusFile, path := loadFixture(t)

	if err := SetConfidenceFuzzy(statusFile, path, 5, 2, 1, 0, 3.4, 82.5, 74.0, 88.0, 71.5); err != nil {
		t.Fatalf("SetConfidenceFuzzy: %v", err)
	}
	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	c := reloaded.Confidence
	if c.Fuzzy == nil || !*c.Fuzzy {
		t.Fatal("fuzzy flag not persisted")
	}
	if c.Dimensions == nil {
		t.Fatal("dimensions block not persisted")
	}
	d := c.Dimensions
	if d.Signal != 82.5 || d.Reversibility != 74.0 || d.Competence != 88.0 || d.Disambiguation != 71.5 {
		t.Errorf("persisted dimensions = %+v, want 82.5/74.0/88.0/71.5", d)
	}
}

// --- Advance remaining branches ---

func TestAdvance_ActiveToReadyPersists(t *testing.T) {
	statusFile, path := loadFixture(t)

	if err := Advance(statusFile, path, "intake", "test"); err != nil {
		t.Fatalf("Advance intake: %v", err)
	}
	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := reloaded.GetProgress("intake"); got != "ready" {
		t.Errorf("intake = %q, want ready", got)
	}
}

func TestAdvance_FromPendingErrors(t *testing.T) {
	statusFile, path := loadFixture(t)

	err := Advance(statusFile, path, "apply", "test")
	if err == nil {
		t.Fatal("expected Advance from pending to error")
	}
	if !strings.Contains(err.Error(), "no valid transition") {
		t.Errorf("want no-valid-transition error, got: %v", err)
	}
}

func TestAdvance_InvalidStageErrors(t *testing.T) {
	statusFile, path := loadFixture(t)

	if err := Advance(statusFile, path, "bogus", "test"); err == nil {
		t.Fatal("expected Advance on invalid stage to error")
	}
	if err := Advance(statusFile, path, "tasks", "test"); err == nil || !strings.Contains(err.Error(), "removed") {
		t.Fatal("expected the removed-stage strict error for tasks")
	}
}

// --- stage_metrics iterations accumulation regression (F39 / intake assumption #6) ---

// TestStageMetrics_IterationsAccumulateAcrossReworkCycles drives the full
// fail→reset→re-finish rework choreography TWICE after the initial activation
// and asserts stage_metrics.review.iterations reads 3 — in memory and on
// disk. This is the spec'd behavior #395 (k4ge) implemented ("incremented,
// not reset — tracks rework cycles"); the observed PR-meta "1 cycle for a
// 3-cycle review" anomaly (2026-06-12) would show here as a reset to 1 if it
// lived in the state machine.
func TestStageMetrics_IterationsAccumulateAcrossReworkCycles(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := t.TempDir()

	// intake → apply → review active (iterations = 1)
	if err := Finish(statusFile, path, dir, "intake", "test"); err != nil {
		t.Fatalf("Finish intake: %v", err)
	}
	if err := Finish(statusFile, path, dir, "apply", "test"); err != nil {
		t.Fatalf("Finish apply: %v", err)
	}

	for cycle := 2; cycle <= 3; cycle++ {
		if err := Fail(statusFile, path, dir, "review", "test", ""); err != nil {
			t.Fatalf("cycle %d Fail review: %v", cycle, err)
		}
		if err := Reset(statusFile, path, dir, "apply", "test", "", ""); err != nil {
			t.Fatalf("cycle %d Reset apply: %v", cycle, err)
		}
		if err := Finish(statusFile, path, dir, "apply", "test"); err != nil {
			t.Fatalf("cycle %d re-Finish apply: %v", cycle, err)
		}
		if sm := statusFile.StageMetrics["review"]; sm == nil || sm.Iterations != cycle {
			t.Fatalf("after rework cycle %d: review iterations = %v, want %d (accumulated, not reset)",
				cycle, statusFile.StageMetrics["review"], cycle)
		}
	}

	// The accumulated count must be what a fresh reader (fab pr-meta) sees.
	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if sm := reloaded.StageMetrics["review"]; sm == nil || sm.Iterations != 3 {
		t.Fatalf("persisted review iterations = %v, want 3", reloaded.StageMetrics["review"])
	}
}

// --- history-shape is identical regardless of driver (260613-fgxx) ---

// loadFixtureInFabRoot lays out a real fab/changes/{name}/ tree so that
// log.Transition (which resolves the change dir via resolve.ToAbsDir(fabRoot,
// statusFile.Name) = {fabRoot}/changes/{folder}) actually writes
// .history.jsonl where the test can read it back. The plain loadFixture helper
// puts .status.yaml in a bare temp dir, so the best-effort transition log
// silently no-ops there — that shape cannot assert on history.
func loadFixtureInFabRoot(t *testing.T) (statusFile *sf.StatusFile, statusPath, fabRoot, changeDir string) {
	t.Helper()
	fabRoot = t.TempDir()
	// statusFile.Name in the fixture is "260310-abcd-my-change".
	changeDir = filepath.Join(fabRoot, "changes", "260310-abcd-my-change")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	statusPath = filepath.Join(changeDir, ".status.yaml")
	if err := os.WriteFile(statusPath, []byte(setAcceptanceFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	var err error
	statusFile, err = sf.Load(statusPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return statusFile, statusPath, fabRoot, changeDir
}

// transitionEntry is the caller-identity-blind subset of a .history.jsonl
// stage-transition entry: everything EXCEPT the timestamp (ts) and the
// intentionally-driver-dependent driver field. Two conforming runs must agree
// on every field here.
type transitionEntry struct {
	Stage  string `json:"stage"`
	Action string `json:"action"`
	From   string `json:"from"`
	Reason string `json:"reason"`
}

// readTransitions parses {changeDir}/.history.jsonl and returns, in order, the
// caller-blind subset of every stage-transition entry plus the parallel slice
// of recorded driver strings (empty when the entry omitted the optional field).
func readTransitions(t *testing.T, changeDir string) (entries []transitionEntry, drivers []string) {
	t.Helper()
	f, err := os.Open(filepath.Join(changeDir, ".history.jsonl"))
	if err != nil {
		t.Fatalf("open .history.jsonl: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			t.Fatalf("unmarshal history line %q: %v", line, err)
		}
		if raw["event"] != "stage-transition" {
			continue
		}
		var te transitionEntry
		if err := json.Unmarshal([]byte(line), &te); err != nil {
			t.Fatalf("unmarshal transition entry %q: %v", line, err)
		}
		driver, _ := raw["driver"].(string) // absent ⇒ "" (the optional-field contract)
		entries = append(entries, te)
		drivers = append(drivers, driver)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan .history.jsonl: %v", err)
	}
	return entries, drivers
}

// driveReworkSequence runs the canonical rework choreography
//
//	Finish(intake) → Finish(apply) → [Fail(review) → Reset(apply) → Finish(apply)]×cycles
//
// with the given driver string, returning the recorded .history.jsonl
// transitions (caller-blind subset) and the parallel recorded-driver slice.
func driveReworkSequence(t *testing.T, driver string, cycles int) (entries []transitionEntry, drivers []string) {
	t.Helper()
	statusFile, statusPath, fabRoot, changeDir := loadFixtureInFabRoot(t)

	if err := Finish(statusFile, statusPath, fabRoot, "intake", driver); err != nil {
		t.Fatalf("Finish intake: %v", err)
	}
	if err := Finish(statusFile, statusPath, fabRoot, "apply", driver); err != nil {
		t.Fatalf("Finish apply: %v", err)
	}
	for c := 1; c <= cycles; c++ {
		if err := Fail(statusFile, statusPath, fabRoot, "review", driver, ""); err != nil {
			t.Fatalf("cycle %d Fail review: %v", c, err)
		}
		if err := Reset(statusFile, statusPath, fabRoot, "apply", driver, "", ""); err != nil {
			t.Fatalf("cycle %d Reset apply: %v", c, err)
		}
		if err := Finish(statusFile, statusPath, fabRoot, "apply", driver); err != nil {
			t.Fatalf("cycle %d re-Finish apply: %v", c, err)
		}
	}
	return readTransitions(t, changeDir)
}

// TestHistoryShape_IdenticalRegardlessOfDriver pins the load-bearing invariant
// behind collapsing the post-intake dual execution mode (260613-fgxx): the
// foreground/manual path (driver="") and the dispatched orchestrator path
// (driver="fab-fff") issue the SAME fab status call sequence, so the
// .history.jsonl transition entries they leave agree on every
// caller-blind field (stage/action/from/reason) — equal modulo the per-run
// ts timestamp and the optional driver annotation. The Go state machine is already caller-agnostic
// (driver flows only into applyMetricsSideEffect, never into a transition
// decision — status.go), so this holds today; the test guards against a future
// skills-layer regression that diverges the two call sequences.
func TestHistoryShape_IdenticalRegardlessOfDriver(t *testing.T) {
	const cycles = 2

	manualEntries, manualDrivers := driveReworkSequence(t, "", cycles)
	dispatchEntries, dispatchDrivers := driveReworkSequence(t, "fab-fff", cycles)

	// Identical in count.
	if len(manualEntries) != len(dispatchEntries) {
		t.Fatalf("transition count differs: manual=%d dispatched=%d", len(manualEntries), len(dispatchEntries))
	}
	if len(manualEntries) == 0 {
		t.Fatal("no stage-transition entries recorded — the history log did not resolve/write")
	}

	// Identical in stage, action, from, reason — for every entry, in order.
	for i := range manualEntries {
		if manualEntries[i] != dispatchEntries[i] {
			t.Errorf("transition %d differs (stage/action/from/reason):\n  manual     = %+v\n  dispatched = %+v",
				i, manualEntries[i], dispatchEntries[i])
		}
	}

	// The ONLY permitted difference: the recorded driver. Manual records no
	// driver (the optional field is omitted ⇒ ""); the dispatched run records
	// "fab-fff" on each driver-carrying entry. Where the dispatched run records
	// a driver, the manual run records none on the same entry.
	sawDispatchDriver := false
	for i := range manualDrivers {
		if manualDrivers[i] != "" {
			t.Errorf("manual (driver=\"\") run unexpectedly recorded driver %q on transition %d", manualDrivers[i], i)
		}
		if dispatchDrivers[i] != "" {
			sawDispatchDriver = true
			if dispatchDrivers[i] != "fab-fff" {
				t.Errorf("dispatched transition %d recorded driver %q, want fab-fff", i, dispatchDrivers[i])
			}
		}
	}
	if !sawDispatchDriver {
		t.Error("dispatched run recorded no driver on any transition — the driver annotation is not being written")
	}
}
