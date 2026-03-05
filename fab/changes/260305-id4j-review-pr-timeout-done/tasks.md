# Tasks: Review-PR Timeout Treated as Done

**Change**: 260305-id4j-review-pr-timeout-done
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Update Copilot polling window from 12 attempts / 6 minutes to 16 attempts / 8 minutes in `fab/.kit/skills/git-pr-review.md` (Step 2, Phase 3)
- [x] T002 [P] Update Step 6 stage routing in `fab/.kit/skills/git-pr-review.md` — move Copilot timeout from failure case to success case (call `finish` instead of `fail`)

## Phase 2: Memory Update

- [x] T003 Update `docs/memory/fab-workflow/execution-skills.md` — PR review handling section to reflect Copilot timeout as done and updated polling window

---

## Execution Order

- T001 and T002 are independent (different sections of same file), can run in parallel
- T003 depends on T001 + T002 (memory should reflect final skill behavior)
