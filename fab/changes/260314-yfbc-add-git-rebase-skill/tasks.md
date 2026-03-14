# Tasks: Add Git Rebase Skill

**Change**: 260314-yfbc-add-git-rebase-skill
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Verify `fab/.kit/skills/git-rebase.md` exists with correct frontmatter (`name`, `description`, `allowed-tools` fields per spec)

## Phase 2: Core Implementation

- [x] T002 Verify skill behavior sections: Step 1 (git repo check), Step 2 (branch guard for main/master), Step 3 (uncommitted changes detection with AskUserQuestion stash-or-abort flow), Step 4 (main branch auto-detection, fetch, rebase with stash safety), Step 5 (report) in `fab/.kit/skills/git-rebase.md`
- [x] T003 Verify error handling table covers all scenarios: not in git repo, on main/master, uncommitted changes, fetch failure, rebase conflicts, stash pop conflicts in `fab/.kit/skills/git-rebase.md`
- [x] T004 Verify key properties table matches spec: no stage advancement, no fab state modification, idempotent in `fab/.kit/skills/git-rebase.md`

## Phase 3: Integration & Edge Cases

- [x] T005 Verify skill deployment — confirm `.claude/skills/git-rebase/SKILL.md` exists as deployed copy after sync
- [x] T006 Verify skill appears in system reminder skill list with correct description

---

## Execution Order

- T002, T003, T004 can run in parallel (all read the same file)
- T005 and T006 are independent verification checks
