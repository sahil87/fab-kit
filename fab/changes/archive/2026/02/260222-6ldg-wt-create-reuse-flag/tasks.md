# Tasks: Add --reuse flag to wt-create

**Change**: 260222-6ldg-wt-create-reuse-flag
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add `--reuse` flag parsing to `fab/.kit/packages/wt/bin/wt-create` — add `--reuse` case to argument parser, store in `reuse` variable, add validation that `--reuse` requires `--worktree-name`, add to help text
- [x] T002 Implement reuse-on-collision logic in `fab/.kit/packages/wt/bin/wt-create` — after name collision check, if `--reuse` is set and collision detected, set `WT_PATH` to existing path, skip creation/init/open, print path as last line and exit success

## Phase 2: Caller Updates

- [x] T003 [P] Update `fab/.kit/scripts/batch-fab-switch-change.sh` line 123 — add `--reuse` to the `wt-create` invocation
- [x] T004 [P] Update `fab/.kit/scripts/pipeline/dispatch.sh` — remove `wt-common.sh` source import and `wt_get_worktree_path_by_name` pre-check (lines 94-99), add `--reuse` to the `wt-create` invocation (line 112)

## Phase 3: Tests

- [x] T005 [P] Add `--reuse Flag Tests` section to `src/packages/wt/tests/wt-create.bats` — 5 test cases: reuse on collision (success + same path), normal creation with --reuse (no collision), init script skipping on reuse, --reuse requires --worktree-name, path as last line on reuse
- [x] T006 [P] Add orphaned directory edge case to `src/packages/wt/tests/edge-cases.bats` — create bare directory at worktree path, verify `--reuse` returns it successfully

---

## Execution Order

- T001 blocks T002 (parsing must exist before collision logic)
- T002 blocks T003, T004, T005, T006 (callers and tests depend on the flag existing)
- T003 and T004 are independent of each other
- T005 and T006 are independent of each other
