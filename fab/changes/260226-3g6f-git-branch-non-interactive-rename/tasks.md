# Tasks: Non-Interactive Branch Rename for /git-branch

**Change**: 260226-3g6f-git-branch-non-interactive-rename
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Rewrite Step 4 in `fab/.kit/skills/git-branch.md` — replace the 3-option interactive menu with deterministic upstream-tracking logic (rename if no upstream, create if has upstream)
- [x] T002 Update Step 5 report format in `fab/.kit/skills/git-branch.md` — add `renamed from {old_branch}` and `created, leaving {old_branch} intact` verbs, remove `adopted` verb

## Phase 2: Documentation Updates

- [x] T003 [P] Update `docs/specs/skills.md` `/git-branch` section — replace "prompt: create new, adopt current, or skip" with new deterministic logic
- [x] T004 [P] Update `docs/memory/fab-workflow/change-lifecycle.md` — replace the branch management options table and `/git-branch` behavior description with new rename/create logic
- [x] T005 [P] Update `docs/memory/fab-workflow/execution-skills.md` — add changelog entry for this change

---

## Execution Order

- T001 blocks T002 (report format depends on the rewritten Step 4)
- T003, T004, T005 are independent of each other (all [P])
- T003-T005 can start after T001-T002 complete (they document the new behavior)
