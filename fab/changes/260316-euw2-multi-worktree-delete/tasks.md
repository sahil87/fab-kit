# Tasks: Multi-Worktree Delete

**Change**: 260316-euw2-multi-worktree-delete
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Change Cobra command definition from `cobra.NoArgs` to `cobra.ArbitraryArgs` in `src/go/wt/cmd/delete.go:33`
- [x] T002 Mark `--worktree-name` flag as deprecated via Cobra's `cmd.Flags().MarkDeprecated("worktree-name", "use positional arguments instead")` in `src/go/wt/cmd/delete.go` after flag registration

## Phase 2: Core Implementation

- [x] T003 Add `handleDeleteMultiple(names []string, nonInteractive bool, deleteBranch, deleteRemote, stashMode string)` function in `src/go/wt/cmd/delete.go` — validate all names upfront (deduplicate, resolve each against `listWorktreeEntries()`), collect unresolved names, exit non-zero if any unresolved, display summary, single confirmation prompt, sequential deletion with continue-on-error, per-worktree branch cleanup and stash handling
- [x] T004 Update `RunE` resolution order in `src/go/wt/cmd/delete.go:34-86` — after `deleteAll` check: if `len(args) > 0 && worktreeName != ""` → error (cannot mix); if `len(args) > 0` → `handleDeleteMultiple(args, ...)`; existing `worktreeName` / current-worktree / menu paths unchanged

## Phase 3: Integration & Edge Cases

- [x] T005 [P] Add test `TestDelete_MultipleByPositionalArgs` — create 3 worktrees, delete 2 by positional args, verify third remains. In `src/go/wt/cmd/delete_test.go`
- [x] T006 [P] Add test `TestDelete_MultipleFailFastOnInvalidName` — create 1 worktree, pass valid + invalid name, verify exit non-zero and valid worktree still exists. In `src/go/wt/cmd/delete_test.go`
- [x] T007 [P] Add test `TestDelete_MultipleDeduplication` — create 1 worktree, pass same name twice, verify deleted once without error. In `src/go/wt/cmd/delete_test.go`
- [x] T008 [P] Add test `TestDelete_MultipleBranchCleanup` — create 2 worktrees, delete both, verify branches deleted. In `src/go/wt/cmd/delete_test.go`
- [x] T009 [P] Add test `TestDelete_MixPositionalAndFlagError` — pass both positional arg and `--worktree-name`, verify error. In `src/go/wt/cmd/delete_test.go`
- [x] T010 [P] Add test `TestDelete_SinglePositionalArg` — verify single positional arg works identically to `--worktree-name`. In `src/go/wt/cmd/delete_test.go`
- [x] T011 [P] Add test `TestDelete_DeprecatedFlagStillWorks` — verify `--worktree-name` continues to work (backward compat). In `src/go/wt/cmd/delete_test.go`
- [x] T012 [P] Add test `TestDelete_MultipleWithStash` — create 2 worktrees with uncommitted changes, delete with `--stash`, verify stashes exist. In `src/go/wt/cmd/delete_test.go`

---

## Execution Order

- T001 and T002 are independent setup tasks
- T003 depends on T001 (needs ArbitraryArgs)
- T004 depends on T003 (needs handleDeleteMultiple to exist)
- T005-T012 depend on T004 (need full implementation) but are independent of each other [P]
