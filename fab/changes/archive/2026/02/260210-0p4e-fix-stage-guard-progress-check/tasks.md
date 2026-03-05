# Tasks: Fix stage guard to check progress value instead of stage name

**Change**: 260210-0p4e-fix-stage-guard-progress-check
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Setup

No setup tasks required — change is contained to existing skill file.

## Phase 2: Core Implementation

- [x] T001 Update stage guard in `fab/.kit/skills/fab-continue.md` Step 1 (Determine Current Stage) to check `progress.{stage} == 'done'` instead of just checking `stage` field name for planning stages (specs, plan, tasks)

- [x] T002 Update guard condition for `tasks` stage in `fab/.kit/skills/fab-continue.md` line 52: change from "If the current stage is `tasks` (done) or later" to "If the current stage is `tasks` AND `progress.tasks == 'done'`, or stage is later than `tasks`"

- [x] T003 Update guard condition for `apply` or later stages in `fab/.kit/skills/fab-continue.md` line 56: ensure this condition remains unchanged (apply/review/archive never resume via /fab-continue, regardless of progress value)

## Phase 3: Integration & Edge Cases

- [x] T004 Verify guard logic handles all state combinations: `{stage: specs, progress.specs: active}` (allow), `{stage: specs, progress.specs: done}` (advance), `{stage: tasks, progress.tasks: active}` (allow), `{stage: tasks, progress.tasks: done}` (block), `{stage: apply}` (block)

## Phase 4: Polish

No polish tasks needed — documentation update will be handled by `/fab-archive` hydration.

---

## Execution Order

- T001 establishes the overall pattern
- T002 and T003 are specific guard condition updates (can run after T001)
- T004 is verification across all cases (runs after T002-T003)
