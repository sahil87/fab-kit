// Package refresh recomputes the artifact-derived .status.yaml fields from the
// on-disk change artifacts (intake.md, plan.md). It is the pull-based successor
// to the removed artifact-write PostToolUse hook: the same recompute logic the
// hook ran on every Write/Edit now runs on demand (fab status refresh) and is
// self-healed at the transition seams (fab status advance/finish, fab
// preflight), so a hook-bypassing edit (sed, direct write) or a non-Claude
// agent that never fires the hook can no longer leave the derived fields stale.
//
// Governing principle (3a of the cross-harness dispatch series): hooks may
// enhance, never own — artifact-derived pipeline state is correctness-critical
// and MUST be pull-based, not written only behind a Claude Code hook.
package refresh

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/sahil87/fab-kit/src/go/fab/internal/hooklib"
	"github.com/sahil87/fab-kit/src/go/fab/internal/score"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// Refresh recomputes artifact-derived fields on the in-memory StatusFile from
// the on-disk intake.md/plan.md under changeDir. It inspects BOTH artifacts
// (not scoped to a single written file like the old hook's match.Artifact),
// since a transition-time refresh is not tied to a single write. The caller
// owns persistence and locking — Refresh mutates sf in memory and reports
// whether anything changed via dirty, and the caller Saves exactly once when
// dirty (the single-load/single-Save discipline the hook followed, mz4q F02).
//
// It respects change_type_source: explicit — when a human ran fab status
// set-change-type the type is kept and re-inference is skipped (the jznd
// guard). It is missing-artifact tolerant: an absent intake.md or plan.md
// recomputes only what it can read and is never an error.
//
// No .history.jsonl append: the confidence recompute uses score.ApplyToStatus
// (not ComputeWithStatus), because refresh runs on every self-healing
// transition/preflight — far more often than an explicit `fab score` — and a
// no-delta re-log would spam the history file. Refresh therefore has no I/O of
// its own beyond reading the artifacts, so it returns an error only for
// forward-compatibility (there is no genuine failure path today).
//
// Idempotent (Constitution III): running twice against unchanged artifacts
// mutates sf to the same values AND reports dirty=false on the second run.
// Refresh sets dirty only when a recompute actually changed a field: it
// compares the recomputed values against the persisted ones (the confidence
// block via confidenceEqual, the plan block via a struct != on the pre/post
// snapshot), so the caller's dirty-guarded Save avoids a spurious write and
// last_updated bump when nothing changed.
func Refresh(fabRoot, changeDir string, sfile *sf.StatusFile) (dirty bool, err error) {
	if refreshFromIntake(changeDir, sfile) {
		dirty = true
	}
	if refreshFromPlan(changeDir, sfile) {
		dirty = true
	}
	return dirty, nil
}

// refreshFromIntake recomputes change_type (respecting the explicit guard) and
// the confidence block from intake.md. A missing intake.md is a no-op.
func refreshFromIntake(changeDir string, sfile *sf.StatusFile) (dirty bool) {
	intakePath := filepath.Join(changeDir, "intake.md")
	content, readErr := os.ReadFile(intakePath)
	if readErr != nil {
		// Missing (or unreadable) intake.md: recompute nothing from it. The
		// intake artifact is legitimately absent at some points (and refresh
		// must be a safe no-op then), so this is not an error.
		return false
	}

	// Respect an explicitly-set change_type: when a human ran fab status
	// set-change-type, change_type_source is "explicit" and refresh must NOT
	// re-infer/overwrite it (the F02 re-clobber guard). Absent or "inferred"
	// source = re-inference allowed (back-compat default). Compare before/after
	// so re-inferring the same type does not report dirty (dirty-idempotency).
	if sfile.ChangeTypeSource != sf.SourceExplicit {
		changeType := hooklib.InferChangeType(string(content))
		beforeType := sfile.ChangeType
		if applyErr := status.ApplyChangeType(sfile, changeType); applyErr == nil && sfile.ChangeType != beforeType {
			dirty = true
		}
	}

	// Recompute the authoritative intake confidence in memory (no history
	// append — see the Refresh doc). The caller owns the Save. Snapshot the
	// pre-recompute confidence and compare after so an unchanged intake does
	// NOT report dirty — otherwise every refresh (and every self-healing
	// preflight/advance/finish) would re-Save and bump last_updated with no
	// real delta, breaking dirty-idempotency (Constitution III).
	before := sfile.Confidence
	score.ApplyToStatus(content, sfile)
	if !confidenceEqual(before, sfile.Confidence) {
		dirty = true
	}

	return dirty
}

// confidenceEqual reports whether two confidence blocks are value-equal,
// including the optional fuzzy Dimensions pointer (a shallow == would compare
// pointer identity, not the pointed-to means). Indicative is a decode-only
// legacy field never written by the recompute, so it is not compared.
func confidenceEqual(a, b sf.Confidence) bool {
	if a.Certain != b.Certain || a.Confident != b.Confident ||
		a.Tentative != b.Tentative || a.Unresolved != b.Unresolved ||
		a.Score != b.Score {
		return false
	}
	if !boolPtrEqual(a.Fuzzy, b.Fuzzy) {
		return false
	}
	return dimensionsEqual(a.Dimensions, b.Dimensions)
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func dimensionsEqual(a, b *sf.Dimensions) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// refreshFromPlan recomputes plan.generated and the three plan counters from
// plan.md, each guarded by HasSectionHeading so a missing section leaves its
// field untouched (never zeroing a valid value). A missing plan.md is a no-op.
func refreshFromPlan(changeDir string, sfile *sf.StatusFile) (dirty bool) {
	planPath := filepath.Join(changeDir, "plan.md")
	data, readErr := os.ReadFile(planPath)
	if readErr != nil {
		return false // missing/unreadable plan.md: recompute nothing from it
	}
	content := string(data)

	hasTasks := hooklib.HasSectionHeading(content, hooklib.SectionTasks)
	hasAcceptance := hooklib.HasSectionHeading(content, hooklib.SectionAcceptance)

	// Snapshot the plan block so we report dirty only when a recompute
	// actually changed a value — ApplyAcceptance always mutates and returns
	// nil, so an err==nil check alone would flag dirty on every run and defeat
	// dirty-idempotency (Constitution III). A missing section is never applied
	// below, so the snapshot preserves those fields untouched.
	before := sfile.Plan

	// generated=true when the file exists with at least a ## Tasks heading.
	if hasTasks {
		_ = status.ApplyAcceptance(sfile, "generated", "true")
		taskCount := hooklib.CountSectionItemsBounded(content, hooklib.SectionTasks)
		_ = status.ApplyAcceptance(sfile, "task_count", strconv.Itoa(taskCount))
	}

	// Only update acceptance counts when ## Acceptance is present.
	if hasAcceptance {
		acceptanceCount := hooklib.CountSectionItemsBounded(content, hooklib.SectionAcceptance)
		acceptanceCompleted := hooklib.CountCompletedSectionItemsBounded(content, hooklib.SectionAcceptance)
		_ = status.ApplyAcceptance(sfile, "acceptance_count", strconv.Itoa(acceptanceCount))
		_ = status.ApplyAcceptance(sfile, "acceptance_completed", strconv.Itoa(acceptanceCompleted))
	}

	return sfile.Plan != before
}
