# Tasks: Add Pipeline Orchestrator Tests

**Change**: 260221-8bs9-add-pipeline-orchestrator-tests
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add source guards to `fab/.kit/scripts/pipeline/run.sh` — wrap the bare `main` call at line 618 in `if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then ... fi`
- [x] T002 [P] Add source guards to `fab/.kit/scripts/pipeline/dispatch.sh` — wrap the bare `main` call at line 322 in `if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then ... fi`
- [x] T003 [P] Create test directory `src/scripts/pipeline/` and scaffold `test.bats` with setup/teardown, BATS load paths, REPO_ROOT, and `make_manifest` helper

## Phase 2: Core Implementation — run.sh Tests

- [x] T004 Add `validate_manifest` tests: valid manifest passes, missing base fails, empty changes fails, missing id fails, missing depends_on fails, dangling reference fails, multi-dependency marks invalid
- [x] T005 Add `detect_cycles` / `dfs_visit` tests: linear chain passes, direct cycle detected, indirect cycle detected, independent nodes pass
- [x] T006 [P] Add `is_terminal` tests: done/failed/invalid return 0, all other stages and empty return 1
- [x] T007 [P] Add `is_dispatchable` tests: no deps + non-terminal passes, self terminal fails, dep not done fails, dep done passes
- [x] T008 Add `find_next_dispatchable` tests: first dispatchable selected, all terminal returns 1, deps not met returns 1, skips terminal changes
- [x] T009 Add `get_parent_branch` tests: root node returns base, dependent node returns parent branch

## Phase 3: Core Implementation — dispatch.sh Tests

- [x] T010 Add `provision_artifacts` tests: first provision creates target, re-provision updates stale target, missing source fails
- [x] T011 Add `validate_prerequisites` tests: missing intake.md fails (returns 2), missing spec.md fails (returns 2), both present with passing gate succeeds

## Phase 4: Polish

- [x] T012 Run full test suite to verify all tests pass: `bats src/scripts/pipeline/test.bats`

---

## Execution Order

- T001 and T002 are independent (parallel)
- T003 is independent (parallel with T001/T002)
- T004-T011 depend on T001, T002, T003 (source guards + test scaffold must exist)
- T004-T005 are sequential (detect_cycles is called by validate_manifest)
- T006, T007 are parallel
- T008 depends on T006+T007 (uses is_terminal and is_dispatchable concepts)
- T009 is independent of T004-T008
- T010-T011 depend on T002 (dispatch.sh source guard)
- T012 runs after all tests are written
