# Tasks: Worktree Status Command

**Change**: 260305-7zq4-worktree-status-command
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/packages/wt/bin/wt-status` with shebang, `set -euo pipefail`, source `wt-common.sh`, and skeleton `main()` function with argument parsing for `help`, `--all`, and positional `<name>`

## Phase 2: Core Implementation

- [x] T002 Implement `wt_get_fab_status()` function — the atomic single-worktree status resolver. Accepts a worktree path, reads `fab/current`, resolves `.status.yaml`, calls `statusman.sh display-stage`, sets output variables (`WT_FAB_CHANGE`, `WT_FAB_STAGE`, `WT_FAB_STATE`) or fallback labels (`(no fab)`, `(no change)`, `(stale)`)
- [x] T003 Implement `wt_show_single_status()` — formats and prints the status for one worktree. Used by both default (current) and `<name>` modes. Calls `wt_get_fab_status` and prints the compact one-line format
- [x] T004 Implement default mode (no args) — detect current worktree path via `pwd -P`, resolve name (basename or `(main)` if at repo root), call `wt_show_single_status`
- [x] T005 Implement `<name>` mode — resolve worktree path via `wt_get_worktree_path_by_name`, error if not found, call `wt_show_single_status`
- [x] T006 Implement `--all` mode — iterate `wt_list_worktrees`, call `wt_get_fab_status` for each, format as aligned table with header, current-worktree marker, and total count

## Phase 3: Integration & Edge Cases

- [x] T007 Add help text following `wt-list` help pattern — usage, options, examples for all three modes
- [x] T008 Make `wt-status` executable (`chmod +x`) and verify it runs from a worktree and from the main repo

## Phase 4: Tests

- [x] T009 Create `src/packages/wt/tests/wt-status.bats` with tests covering: help output, current worktree status, named worktree, `--all` mode, no-fab fallback, no-change fallback, stale pointer fallback, invalid name error

---

## Execution Order

- T001 blocks T002-T007
- T002 blocks T003-T006
- T003 blocks T004, T005
- T004, T005, T006 are independent of each other once T003 exists
- T007 is independent of T002-T006
- T008 depends on T001-T007
- T009 depends on T008
