# Quality Checklist: Add --base flag to wt create

**Change**: 260312-wrk6-add-wt-create-base-flag
**Generated**: 2026-03-12
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 --base flag: `wt create` accepts `--base <ref>` flag
- [x] CHK-002 New branch with --base: new branch is created from the specified start-point, not HEAD
- [x] CHK-003 Exploratory with --base: exploratory worktree branches from the specified start-point
- [x] CHK-004 CreateWorktree startPoint: function accepts and uses startPoint parameter
- [x] CHK-005 CreateBranchWorktree startPoint: passes through to CreateWorktree for new branches
- [x] CHK-006 CreateExploratoryWorktree startPoint: passes through to CreateWorktree
- [x] CHK-007 Operator2 autopilot: uses --base for user-provided ordering dependencies
- [x] CHK-008 packages.md: documents --base flag behavior

## Behavioral Correctness
- [x] CHK-009 Existing local branch: --base is ignored with warning on stderr
- [x] CHK-010 Existing remote branch: --base is ignored with warning on stderr
- [x] CHK-011 --reuse precedence: --reuse takes precedence over --base when worktree exists
- [x] CHK-012 No --base regression: creating branches without --base still works from HEAD

## Scenario Coverage
- [x] CHK-013 TestCreate_BaseNewBranch: marker file present, HEAD matches base tip
- [x] CHK-014 TestCreate_BaseExploratoryWorktree: branches from base tip
- [x] CHK-015 TestCreate_BaseWithExistingLocalBranch: warning emitted, branch unchanged
- [x] CHK-016 TestCreate_BaseWithExistingRemoteBranch: warning emitted, remote branch fetched
- [x] CHK-017 TestCreate_BaseInvalidRef: non-zero exit, no worktree created
- [x] CHK-018 TestCreate_BaseWithReuse: reuse takes precedence
- [x] CHK-019 TestCreate_BaseDoesNotAffectExistingBehavior: HEAD-based branching preserved

## Edge Cases & Error Handling
- [x] CHK-020 Invalid ref validation: git rev-parse --verify run before worktree creation; clear error on failure
- [x] CHK-021 No partial state on invalid ref: no worktree directory or branch left behind
- [x] CHK-022 Confidence-based ordering: operator does NOT use --base

## Code Quality
- [x] CHK-023 Pattern consistency: new code follows naming and structural patterns of surrounding code in wt package
- [x] CHK-024 No unnecessary duplication: existing utilities (CreateWorktree, branch checks) reused

## Documentation Accuracy
- [x] CHK-025 packages.md accuracy: --base documentation matches implementation behavior
- [x] CHK-026 operator2 skill accuracy: autopilot --base usage matches implementation

## Cross References
- [x] CHK-027 All callers of CreateWorktree updated: no compile errors from signature change
- [x] CHK-028 Spec-code alignment: all spec requirements have corresponding implementation

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
