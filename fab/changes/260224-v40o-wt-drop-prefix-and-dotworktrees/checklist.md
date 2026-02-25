# Quality Checklist: Drop wt/ Branch Prefix and Switch to .worktrees Directory

**Change**: 260224-v40o-wt-drop-prefix-and-dotworktrees
**Generated**: 2026-02-25
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Exploratory worktrees use unprefixed branch names: `wt_create_exploratory_worktree()` sets `branch="$name"` not `branch="wt/$name"`
- [x] CHK-002 Worktrees directory uses dot-suffix: `wt_get_repo_context()` sets `WT_WORKTREES_DIR` to `${WT_REPO_NAME}.worktrees`
- [x] CHK-003 Help text reflects new convention: dynamic path uses `.worktrees`, static text says `<random-name> branch` not `wt/<random-name> branch`
- [x] CHK-004 git-branch skill has no `wt/*` pattern matching: all non-main branches treated uniformly

## Behavioral Correctness
- [x] CHK-005 Branch-based worktrees unchanged: `wt_create_branch_worktree()` still uses user-specified branch name as-is
- [x] CHK-006 Exploratory branch name equals worktree directory name: for name `swift-fox`, branch is `swift-fox` and dir is `<repo>.worktrees/swift-fox/`

## Removal Verification
- [x] CHK-007 `wt/` prefix removed: no code path produces `wt/$name` branches
- [x] CHK-008 `wt/*` default override removed from git-branch skill: no branch-pattern-based default selection

## Scenario Coverage
- [x] CHK-009 Default name exploratory worktree: creates unprefixed branch with `.worktrees` directory
- [x] CHK-010 User-overridden name exploratory worktree: creates unprefixed branch
- [x] CHK-011 Non-interactive exploratory worktree: creates unprefixed branch
- [x] CHK-012 Branch-based worktree: uses branch as-is (unchanged)

## Edge Cases & Error Handling
- [x] CHK-013 Name collision detection works with new `.worktrees` directory path
- [x] CHK-014 Worktree directory auto-created on first use with `.worktrees` suffix

## Code Quality
- [x] CHK-015 Pattern consistency: changes follow existing naming and structural patterns
- [x] CHK-016 No unnecessary duplication: no leftover `-worktrees` references in modified files
- [x] CHK-017 Readability: changes are minimal and focused (code-quality.md principle)
- [x] CHK-018 **N/A**: `main()` in wt-create exceeds 50 lines but is pre-existing and not worsened by this change

## Documentation Accuracy
- [x] CHK-019 `docs/specs/packages.md` uses `.worktrees` convention throughout
- [x] CHK-020 No stale `wt/` prefix references in documentation

## Cross References
- [x] CHK-021 Test assertions updated to match new conventions: `test_helper.bash` and `wt-create.bats` use `.worktrees` and unprefixed branches
- [x] CHK-022 Memory file `distribution.md` references updated during hydrate

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
