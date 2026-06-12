package status

// Exhaustive state-machine transition tests (260612-tb6f, F39).
//
// The expected outcomes below are written out by hand — from the state-machine
// semantics documented in docs/memory/pipeline/schemas.md (transition events,
// forbidden state combinations) plus the shipped post-k4ge behavior
// (AllowedStates-enforced transition targets, review/review-pr failed→active
// start override) — deliberately NOT by referencing the implementation's own
// tables, so a table regression cannot silently rewrite the test's expectations.

import (
	"strings"
	"testing"

	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// specAllowedStates is the per-stage allowed-state table, written out by hand
// (NOT referencing the package's AllowedStates var).
var specAllowedStates = map[string][]string{
	"intake":    {"active", "ready", "done"},
	"apply":     {"pending", "active", "ready", "done", "skipped"},
	"review":    {"pending", "active", "ready", "done", "failed", "skipped"},
	"hydrate":   {"pending", "active", "ready", "done", "skipped"},
	"ship":      {"pending", "active", "done", "skipped"},
	"review-pr": {"pending", "active", "done", "failed", "skipped"},
}

// TestLookupTransition_ExhaustiveMatrix walks every stage × event × from-state
// cell (6 × 6 × 6 = 216) and asserts the outcome against the spec'd tables:
//
//   - start:   pending → active            (review/review-pr also failed → active)
//   - advance: active → ready
//   - finish:  active|ready → done
//   - reset:   done|ready|skipped → active
//   - skip:    pending|active → skipped
//   - fail:    active → failed             (review/review-pr ONLY)
//
// with the resolved target additionally validated against the stage's
// allowed states (so `advance ship`, `advance review-pr`, and `skip intake`
// are rejected even from their nominally-valid from-states).
func TestLookupTransition_ExhaustiveMatrix(t *testing.T) {
	allStates := []string{"pending", "active", "ready", "done", "failed", "skipped"}
	allEvents := []string{"start", "advance", "finish", "reset", "skip", "fail"}

	eventFrom := map[string][]string{
		"start":   {"pending"},
		"advance": {"active"},
		"finish":  {"active", "ready"},
		"reset":   {"done", "ready", "skipped"},
		"skip":    {"pending", "active"},
		"fail":    {"active"},
	}
	eventTarget := map[string]string{
		"start":   "active",
		"advance": "ready",
		"finish":  "done",
		"reset":   "active",
		"skip":    "skipped",
		"fail":    "failed",
	}
	failCapable := map[string]bool{"review": true, "review-pr": true}

	cells := 0
	for _, stage := range sf.StageOrder {
		for _, event := range allEvents {
			for _, from := range allStates {
				cells++

				// Spec: is this event defined for this stage at all?
				eventDefined := event != "fail" || failCapable[stage]

				// Spec: is the from-state in the event's From set?
				fromOK := contains(eventFrom[event], from)
				if event == "start" && failCapable[stage] && from == "failed" {
					fromOK = true // the failed→active start override
				}

				// Spec: is the resolved target allowed for the stage?
				target := eventTarget[event]
				targetAllowed := contains(specAllowedStates[stage], target)

				wantSuccess := eventDefined && fromOK && targetAllowed

				got, err := lookupTransition(event, stage, from)
				if wantSuccess {
					if err != nil {
						t.Errorf("%s %s from %s: want target %q, got error: %v", event, stage, from, target, err)
						continue
					}
					if got != target {
						t.Errorf("%s %s from %s = %q, want %q", event, stage, from, got, target)
					}
					continue
				}

				if err == nil {
					t.Errorf("%s %s from %s: want rejection, got target %q", event, stage, from, got)
					continue
				}
				if eventDefined && fromOK && !targetAllowed {
					if !strings.Contains(err.Error(), "not allowed for this stage") {
						t.Errorf("%s %s from %s: want forbidden-target error, got: %v", event, stage, from, err)
					}
				} else {
					if !strings.Contains(err.Error(), "no valid transition") {
						t.Errorf("%s %s from %s: want no-valid-transition error, got: %v", event, stage, from, err)
					}
				}
			}
		}
	}

	if cells != 216 {
		t.Fatalf("matrix walked %d cells, want 216 (6 stages × 6 events × 6 states)", cells)
	}
}

// --- Skip forward-cascade (pending → skipped) ---

func TestSkip_ForwardCascadeSkipsPendingDownstream(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := t.TempDir()

	statusFile.SetProgress("intake", "done")
	statusFile.SetProgress("apply", "active")
	// A downstream entry with iterations must survive the skipped-cascade
	// (same preservation contract as the reset cascade), a zero-iteration
	// entry must be deleted.
	statusFile.StageMetrics["review"] = &sf.StageMetric{Iterations: 2, StartedAt: "2026-03-10T12:00:00Z"}
	statusFile.StageMetrics["ship"] = &sf.StageMetric{CompletedAt: "2026-03-10T12:00:00Z"}

	if err := Skip(statusFile, path, dir, "apply", "test"); err != nil {
		t.Fatalf("Skip apply: %v", err)
	}

	if got := statusFile.GetProgress("apply"); got != "skipped" {
		t.Errorf("apply = %q, want skipped", got)
	}
	for _, stage := range []string{"review", "hydrate", "ship", "review-pr"} {
		if got := statusFile.GetProgress(stage); got != "skipped" {
			t.Errorf("downstream %s = %q, want skipped (forward cascade)", stage, got)
		}
	}
	if got := statusFile.GetProgress("intake"); got != "done" {
		t.Errorf("upstream intake = %q, want done (untouched)", got)
	}

	// stage_metrics: iterations preserved with timing cleared; zero-iteration deleted.
	if sm := statusFile.StageMetrics["review"]; sm == nil || sm.Iterations != 2 {
		t.Errorf("review metrics after skip cascade = %+v, want iterations 2 preserved", statusFile.StageMetrics["review"])
	} else if sm.StartedAt != "" {
		t.Errorf("review timing fields should be cleared, got %+v", sm)
	}
	if _, ok := statusFile.StageMetrics["ship"]; ok {
		t.Error("zero-iteration ship metrics entry should be deleted by the skip cascade")
	}

	// Cascade result must be persisted.
	reloaded, err := sf.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, stage := range []string{"apply", "review", "hydrate", "ship", "review-pr"} {
		if got := reloaded.GetProgress(stage); got != "skipped" {
			t.Errorf("reloaded %s = %q, want skipped", stage, got)
		}
	}
}

func TestSkip_CascadeLeavesNonPendingUntouched(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := t.TempDir()

	// review parked failed; hydrate already done (hand-built shape — the
	// cascade must only touch pending stages, whatever else it walks past).
	statusFile.SetProgress("intake", "done")
	statusFile.SetProgress("apply", "active")
	statusFile.SetProgress("review", "failed")
	statusFile.SetProgress("hydrate", "done")

	if err := Skip(statusFile, path, dir, "apply", "test"); err != nil {
		t.Fatalf("Skip apply: %v", err)
	}

	if got := statusFile.GetProgress("review"); got != "failed" {
		t.Errorf("review = %q, want failed (non-pending untouched)", got)
	}
	if got := statusFile.GetProgress("hydrate"); got != "done" {
		t.Errorf("hydrate = %q, want done (non-pending untouched)", got)
	}
	for _, stage := range []string{"ship", "review-pr"} {
		if got := statusFile.GetProgress(stage); got != "skipped" {
			t.Errorf("%s = %q, want skipped", stage, got)
		}
	}
}

func TestSkip_FromPendingStage(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := t.TempDir()

	statusFile.SetProgress("intake", "done")
	statusFile.SetProgress("apply", "done")
	statusFile.SetProgress("review", "done")
	statusFile.SetProgress("hydrate", "done")
	// ship and review-pr both pending; skipping ship cascades review-pr too.

	if err := Skip(statusFile, path, dir, "ship", "test"); err != nil {
		t.Fatalf("Skip ship: %v", err)
	}
	if got := statusFile.GetProgress("ship"); got != "skipped" {
		t.Errorf("ship = %q, want skipped", got)
	}
	if got := statusFile.GetProgress("review-pr"); got != "skipped" {
		t.Errorf("review-pr = %q, want skipped (cascade)", got)
	}
}

func TestSkip_InvalidCurrentStateErrors(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := t.TempDir()

	statusFile.SetProgress("apply", "done")
	err := Skip(statusFile, path, dir, "apply", "test")
	if err == nil {
		t.Fatal("expected Skip from done to error")
	}
	if !strings.Contains(err.Error(), "no valid transition") {
		t.Errorf("want no-valid-transition error, got: %v", err)
	}
}

func TestSkip_InvalidStageErrors(t *testing.T) {
	statusFile, path := loadFixture(t)
	dir := t.TempDir()

	if err := Skip(statusFile, path, dir, "bogus", "test"); err == nil {
		t.Fatal("expected Skip on invalid stage to error")
	}
}
