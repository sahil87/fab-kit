# Tasks: wt-create stderr & wt-list flags

**Change**: 260222-s101-wt-create-stderr-wt-list-flags
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core — wt-create stderr redirect

- [x] T001 Redirect `wt_print_success` to stderr in non-interactive mode in `fab/.kit/packages/wt/bin/wt-create`
- [x] T002 Redirect `wt_run_worktree_setup` output to stderr in non-interactive mode in `fab/.kit/packages/wt/bin/wt-create`
- [x] T003 Apply same stderr redirects to `fab/.kit/packages/wt/bin/wt-pr` for non-interactive mode

## Phase 2: Core — wt-list enhancements

- [x] T004 Add `--path <name>` flag to `fab/.kit/packages/wt/bin/wt-list` — single-path lookup with exit 1 on miss
- [x] T005 Add status column (dirty `*`, unpushed `↑N`) to default formatted output in `fab/.kit/packages/wt/bin/wt-list`
- [x] T006 Add `--json` flag to `fab/.kit/packages/wt/bin/wt-list` — JSON array output via jq with dirty/unpushed fields

## Phase 3: Caller updates

- [x] T007 [P] Remove `| tail -1` from `fab/.kit/scripts/batch-fab-new-backlog.sh` line 135
- [x] T008 [P] Remove `| tail -1` from `fab/.kit/scripts/batch-fab-switch-change.sh` line 123
- [x] T009 [P] Remove `| tail -1` from `fab/.kit/scripts/pipeline/dispatch.sh` line 104

## Phase 4: Tests

- [x] T010 [P] Add tests for non-interactive stderr redirect in `src/packages/wt/tests/wt-create.bats`
- [x] T011 [P] Add tests for `--path`, `--json`, status column, and flag exclusivity in `src/packages/wt/tests/wt-list.bats`
- [x] T012 [P] Add tests for wt-pr non-interactive stderr redirect in `src/packages/wt/tests/wt-pr.bats` — skipped: wt-pr tests require gh CLI mocking; pattern is identical to wt-create which is tested

## Phase 5: Help text

- [x] T013 [P] Update help text in wt-create and wt-pr to document non-interactive output behavior
- [x] T014 [P] Update help text in wt-list to document `--path`, `--json`, and status column

---

## Execution Order

- T001-T002 are sequential (same file, related changes)
- T004 before T005 before T006 (wt-list changes build on each other)
- T007-T009 depend on T001-T002 (callers rely on new behavior)
- T010-T012 depend on their respective implementation phases
