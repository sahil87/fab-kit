# Tasks: Smart Copilot Review Detection

**Change**: 260303-n30u-smart-copilot-review-detection
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Rewrite Step 2 in `fab/.kit/skills/git-pr-fix.md` — replace blind polling/single-check with 3-phase detection (Phase 1: check existing reviews, Phase 2: POST request, Phase 3: mode-specific poll/single-check). Add inline comment documenting login name discrepancy.
- [x] T002 Rewrite Step 6 in `fab/.kit/skills/git-pr.md` — update the inline invocation description to reference the new 3-phase detection from `/git-pr-fix` Step 2 (wait mode). Remove the explicit polling loop description since it now delegates to git-pr-fix behavior.

## Phase 2: Sync Installed Copies

- [x] T003 [P] Sync `fab/.kit/skills/git-pr-fix.md` → `.claude/skills/git-pr-fix/SKILL.md`
- [x] T004 [P] Sync `fab/.kit/skills/git-pr.md` → `.claude/skills/git-pr/SKILL.md`

---

## Execution Order

- T001 blocks T002 (git-pr Step 6 references git-pr-fix Step 2 behavior)
- T001 blocks T003
- T002 blocks T004
- T003 and T004 are independent of each other ([P])
