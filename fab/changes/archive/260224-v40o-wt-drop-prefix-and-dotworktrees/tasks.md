# Tasks: Drop wt/ Branch Prefix and Switch to .worktrees Directory

**Change**: 260224-v40o-wt-drop-prefix-and-dotworktrees
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Remove `wt/` branch prefix in `fab/.kit/packages/wt/bin/wt-create` — change `local branch="wt/$name"` to `local branch="$name"` in `wt_create_exploratory_worktree()` (line 60)
- [x] T002 [P] Switch worktree directory convention in `fab/.kit/packages/wt/lib/wt-common.sh` — change `${WT_REPO_NAME}-worktrees` to `${WT_REPO_NAME}.worktrees` in `wt_get_repo_context()` (line 304)
- [x] T003 [P] Update help text in `fab/.kit/packages/wt/bin/wt-create` — change dynamic path `${repo_name}-worktrees/` to `${repo_name}.worktrees/` in `wt_show_help()` (line 79), and static text `creates wt/<random-name> branch` to `creates <random-name> branch` (line 89), and file header comment `<repo>-worktrees/` to `<repo>.worktrees/` (line 8)

## Phase 2: Skill Update

- [x] T004 Remove `wt/*` special-casing in `fab/.kit/skills/git-branch.md` — remove the `wt/*` mention from Step 5 line 94 and the default override for `wt/*` branches in line 96; all non-main, non-target branches should present "Adopt this branch" as default

## Phase 3: Test Updates

- [x] T005 Update test helper `src/packages/wt/tests/test_helper.bash` — change `-worktrees` to `.worktrees` in `assert_worktree_exists()` (line 96), `assert_worktree_not_exists()` (line 115), and `cleanup_test_repo()` (line 312)
- [x] T006 Update test assertions in `src/packages/wt/tests/wt-create.bats` — (a) rename test "creates wt/<name> branch" to verify unprefixed branch: change `local expected_branch="wt/${wt_name}"` to `local expected_branch="${wt_name}"` (line 56); (b) update directory structure test: change `${repo_name}-worktrees` to `${repo_name}.worktrees` (line 480); (c) update mock verification patterns from `-worktrees` to `.worktrees` (lines 293, 301)

## Phase 4: Documentation

- [x] T007 Update `docs/specs/packages.md` — change `<repo>-worktrees/` to `<repo>.worktrees/` in wt section description (line 11) and common workflows example (line 41)

---

## Execution Order

- T001, T002, T003 are independent (all [P])
- T004 is independent of T001-T003
- T005 should complete before or alongside T006 (shared test infrastructure)
- T007 is independent
