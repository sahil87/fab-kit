# Tasks: Fix git-pr Ship Finish Ordering

**Change**: 260307-8ggm-git-pr-ship-finish-ordering
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Reorder and renumber post-PR steps in `fab/.kit/skills/git-pr.md`: move Step 4d (finish ship stage) to immediately after Step 4 (record PR URL), renumber as 4a/4b/4c/4d, update Step 4c (commit+push) to stage both `.status.yaml` and `.history.jsonl`
- [x] T002 Update `docs/specs/skills/SPEC-git-pr.md` flow diagram to reflect new 4a/4b/4c/4d step ordering and descriptions

## Phase 2: Memory

- [x] T003 Update `docs/memory/fab-workflow/execution-skills.md` PR shipping section to document the corrected step ordering

---

## Execution Order

- T001 is the primary implementation task
- T002 depends on T001 (spec reflects what the skill says)
- T003 depends on T001 (memory reflects what shipped)
