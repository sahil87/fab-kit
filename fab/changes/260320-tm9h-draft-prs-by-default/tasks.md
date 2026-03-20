# Tasks: Draft PRs by Default

**Change**: 260320-tm9h-draft-prs-by-default
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Add `--draft` flag to `gh pr create` command in `fab/.kit/skills/git-pr.md` Step 3c item 4
- [x] T002 [P] Add `--draft` flag to `gh pr create --fill` fallback in `fab/.kit/skills/git-pr.md` Step 3c item 4

## Phase 2: Spec Update

- [x] T003 Update `docs/specs/skills/SPEC-git-pr.md` flow diagram Step 3c to show `--draft` flag

## Phase 3: Backlog Cleanup

- [x] T004 Mark backlog item `[m1ef]` as done via `fab/.kit/bin/fab idea done m1ef`

---

## Execution Order

- T001 and T002 are independent (parallel)
- T003 is independent
- T004 is independent
