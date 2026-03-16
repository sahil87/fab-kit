# Quality Checklist: Multi-Worktree Delete

**Change**: 260316-euw2-multi-worktree-delete
**Generated**: 2026-03-16
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Positional args: `wt delete name1 name2` deletes both worktrees
- [x] CHK-002 Fail-fast validation: invalid names abort before any deletion
- [x] CHK-003 Deduplication: repeated names are handled without error
- [x] CHK-004 Single confirmation: interactive multi-delete shows one prompt
- [x] CHK-005 Continue-on-error: per-worktree failure doesn't abort remaining
- [x] CHK-006 Branch cleanup: `--delete-branch` / `--delete-remote` applied per worktree
- [x] CHK-007 Stash: `--stash` stashes changes per worktree before deletion
- [x] CHK-008 Resolution order: `--delete-all` > positional args > `--worktree-name` > current > menu
- [x] CHK-009 Deprecation: `--worktree-name` prints deprecation warning but still works
- [x] CHK-010 Mix error: positional args + `--worktree-name` produces clear error

## Behavioral Correctness
- [x] CHK-011 Single positional arg behaves identically to `--worktree-name`
- [x] CHK-012 No args, no flags: existing resolution order unchanged (current worktree or menu)
- [x] CHK-013 `--delete-all` with positional args: --delete-all wins
- [x] CHK-014 Non-interactive mode: no prompts for multi-delete

## Scenario Coverage
- [x] CHK-015 Multiple valid names (spec scenario: delete 2 of 3)
- [x] CHK-016 One invalid name in batch (spec scenario: fail-fast)
- [x] CHK-017 All names invalid (spec scenario: multiple not found)
- [x] CHK-018 Duplicate names (spec scenario: deduplicate)
- [x] CHK-019 Multi-delete with stash (spec scenario)
- [x] CHK-020 Multi-delete with branch preservation (spec scenario: --delete-branch false)

## Edge Cases & Error Handling
- [x] CHK-021 Zero positional args falls through to existing handlers
- [x] CHK-022 Non-interactive + no target + no args still errors correctly
- [x] CHK-023 Error message for invalid names includes all unresolved names

## Code Quality
- [x] CHK-024 Pattern consistency: `handleDeleteMultiple` follows existing handler patterns (handleDeleteAll, handleDeleteByName)
- [x] CHK-025 No unnecessary duplication: reuses existing helpers (listWorktreeEntries, handleBranchCleanup, handleStashInDir, wt.RemoveWorktree)

## Documentation Accuracy
- [x] CHK-026 `docs/specs/packages.md` updated with new `wt delete` positional args syntax
- [x] CHK-027 Deprecation note for `--worktree-name` in docs

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
