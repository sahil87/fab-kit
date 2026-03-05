# Tasks: Harden wt Package Resilience

**Change**: 260218-qcqx-harden-wt-resilience
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Shared Library Functions

<!-- Independent additions to wt-common.sh — all parallelizable -->

- [x] T001 [P] Add rollback stack + signal handler to `fab/.kit/packages/wt/lib/wt-common.sh`: implement `WT_ROLLBACK_STACK` array, `wt_register_rollback` (push command), `wt_rollback` (LIFO execution with `set +e`), `wt_disarm_rollback` (clear stack), and `wt_cleanup_on_signal` (newline + rollback + exit 130). Add unit tests to `src/packages/wt/tests/`.
- [x] T002 [P] Add hash-based stash functions to `fab/.kit/packages/wt/lib/wt-common.sh`: implement `wt_stash_create` (`git add -A` suppressed, `git stash create`, `git stash store` for reflog, `git reset --hard HEAD` + `git clean -fd`, echo hash) and `wt_stash_apply` (`git stash apply <hash>`, no-op if empty). Add unit tests to `src/packages/wt/tests/`.
- [x] T003 [P] Add branch name validation to `fab/.kit/packages/wt/lib/wt-common.sh`: implement `wt_validate_branch_name` rejecting empty, whitespace, `~^:?*[`, `..`, `.lock` suffix, `.` prefix, `/.` component. Return 0/1. Add unit tests to `src/packages/wt/tests/`.

## Phase 2: Command Integration

<!-- Integrate shared functions into existing commands. Sequential — wt-create first, then wt-delete. -->

- [x] T004 Integrate rollback, signal traps, validation, and dirty-state check into `fab/.kit/packages/wt/bin/wt-create`: add `trap wt_rollback EXIT` and `trap wt_cleanup_on_signal INT TERM` early in `main()`, register rollback in `wt_create_worktree` after `git worktree add` and branch creation, call `wt_disarm_rollback` before final output, add `wt_validate_branch_name` call on `$branch_arg` before git ops, add dirty-state check (uncommitted/untracked → yellow warning + 3-option `wt_show_menu`: continue/stash/abort; non-interactive defaults to continue). Update `src/packages/wt/tests/wt-create.bats`.
- [x] T005 Migrate `fab/.kit/packages/wt/bin/wt-delete` to hash-based stash + signal traps: replace `wt_stash_changes` function with calls to `wt_stash_create`/`wt_stash_apply` from wt-common.sh, capture stash hash and register with rollback stack, add `trap wt_cleanup_on_signal INT TERM`, update all stash callsites (`wt_handle_uncommitted_changes`, `wt_delete_worktree_by_name`). Update `src/packages/wt/tests/wt-delete.bats`.

## Phase 3: New Command

- [x] T006 Create `fab/.kit/packages/wt/bin/wt-pr`: source wt-common.sh, validate `gh` CLI (`command -v gh`), parse args (PR number positional, `--worktree-name`, `--worktree-init`, `--worktree-open`, `--non-interactive`, `help`), interactive mode shows PR list via `gh pr list --json number,title,headRefName` + `wt_show_menu`, non-interactive without PR number → error, resolve branch via `gh pr view $PR --json headRefName --jq .headRefName`, validate branch name, delegate to `wt_create_branch_worktree`, add rollback/signal traps, follow wt-create patterns for init/open. Add `src/packages/wt/tests/wt-pr.bats`. Make executable (`chmod +x`).

---

## Execution Order

- T001, T002, T003 are independent (parallel)
- T004 depends on T001 (rollback + signal) and T003 (validation)
- T005 depends on T002 (stash) and T001 (signal handler)
- T006 depends on T001, T002, T003, and T004 (reuses all shared functions + follows wt-create patterns)
