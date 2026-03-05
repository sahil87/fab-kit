# Tasks: Fix reset flow to stop at target stage

**Change**: 260213-wo9v-fix-reset-auto-advance
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Fix Stage Derivation Fallback

<!-- Fix the "no active → archive" fallback in all three scripts to use "first pending after last done" -->

- [x] T001 [P] Update fallback logic in `fab/.kit/scripts/fab-preflight.sh` (lines 122-125): replace `stage="archive"` with a loop that walks stages in order, finds the last `done`, and returns the first `pending` after it. Fall back to `archive` only when all stages are `done`.
- [x] T002 [P] Update fallback logic in `fab/.kit/scripts/fab-status.sh` (lines 133-135): same three-tier fallback as T001 — walk stages, find first `pending` after last `done`, fall back to `archive` only if all `done`.
- [x] T003 [P] Update `get_current_stage()` in `fab/.kit/scripts/stageman.sh` (lines 211-227): replace the `echo "archive"` fallback with the same three-tier logic. Walk stages using `get_all_stages`, grep for states in the status file, return first `pending` after last `done`.

## Phase 2: Update Schema and Status Display

- [x] T004 [P] Update `progression.current_stage` in `fab/.kit/schemas/workflow.yaml` (lines 150-153): change `rule` text to "First stage with state=active; if no active, first pending stage after last done; if all done, archive" and update `fallback` field accordingly.
- [x] T005 [P] Add `{stage}:pending` cases to the next-command `case` statement in `fab/.kit/scripts/fab-status.sh` (lines 165-178): add `brief:pending`, `spec:pending`, `tasks:pending`, `apply:pending`, `review:pending` with appropriate `/fab-continue` suggestions per spec.

## Phase 3: Update Skill Definitions

- [x] T006 Update Reset Flow in `fab/.kit/skills/fab-continue.md`: modify Step 4 (Reset `.status.yaml`) to set target stage to `done` without setting the next stage to `active`. Downstream stages become `pending`. Add pre-guard step: when preflight's derived stage has `pending` progress, set it to `active` before dispatching. Update the guard table's "No active entry" row to reference the new fallback behavior.

## Execution Order

- T001, T002, T003 are independent — all fix the same fallback pattern in different files
- T004, T005 are independent of each other but logically follow T001-T003
- T006 depends on the fallback fix being defined (T001-T003) but doesn't depend on the code changes
