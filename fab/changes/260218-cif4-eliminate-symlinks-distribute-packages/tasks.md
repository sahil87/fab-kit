# Tasks: Eliminate Symlinks, Distribute Packages via Kit

**Change**: 260218-cif4-eliminate-symlinks-distribute-packages
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/scripts/env-packages.sh` — PATH setup script that iterates `fab/.kit/packages/*/bin` and exports each to PATH
- [x] T002 Create `fab/.kit/packages/` directory structure — `mkdir -p fab/.kit/packages/idea/bin fab/.kit/packages/wt/bin fab/.kit/packages/wt/lib`

## Phase 2: Core Implementation

- [x] T003 Move package production code via `git mv` — `src/packages/idea/bin/idea` → `fab/.kit/packages/idea/bin/idea`, `src/packages/wt/bin/*` → `fab/.kit/packages/wt/bin/`, `src/packages/wt/lib/wt-common.sh` → `fab/.kit/packages/wt/lib/wt-common.sh`
- [x] T004 [P] Update `src/lib/stageman/test.bats` preamble (lines 7-8) — replace `SCRIPT_DIR`/`readlink -f` with `REPO_ROOT`/direct path to `fab/.kit/scripts/lib/stageman.sh`
- [x] T005 [P] Update `src/lib/changeman/test.bats` preamble (lines 7-8) — replace `SCRIPT_DIR`/`readlink -f` with `REPO_ROOT`/direct path to `fab/.kit/scripts/lib/changeman.sh`
- [x] T006 [P] Update `src/lib/calc-score/test.bats` preamble (lines 8-9) — replace `SCRIPT_DIR`/`readlink -f` with `REPO_ROOT`/direct path to `fab/.kit/scripts/lib/calc-score.sh`
- [x] T007 Delete 5 symlinks via `git rm` — `src/lib/stageman/stageman.sh`, `src/lib/changeman/changeman.sh`, `src/lib/calc-score/calc-score.sh`, `src/lib/preflight/preflight.sh`, `src/lib/sync-workspace/fab-sync.sh`

## Phase 3: Integration & Edge Cases

- [x] T008 [P] Update `src/packages/idea/tests/setup_suite.bash` — replace `${BATS_TEST_DIRNAME}/../bin` PATH with `$REPO_ROOT/fab/.kit/packages/idea/bin`
- [x] T009 [P] Update `src/packages/wt/tests/setup_suite.bash` — add explicit `$REPO_ROOT/fab/.kit/packages/wt/bin` to PATH
- [x] T010 Update `src/packages/rc-init.sh` — simplify to delegate to `fab/.kit/scripts/env-packages.sh`
- [x] T011 Add `source fab/.kit/scripts/env-packages.sh` line to `fab/.kit/scaffold/envrc`
- [x] T012 Run all tests — `src/lib/stageman/`, `src/lib/changeman/`, `src/lib/calc-score/`, `src/lib/preflight/`, `src/lib/sync-workspace/`, `src/packages/idea/tests/`, `src/packages/wt/tests/`

## Phase 4: Polish

- [x] T013 [P] Update stale symlink references in comments — `fab/.kit/scripts/lib/stageman.sh` (line 15), `fab/.kit/scripts/lib/changeman.sh` (line 16), `fab/.kit/scripts/fab-upgrade.sh` (line 96), `fab/.kit/scripts/fab-help.sh` (line 138)
- [x] T014 [P] Update `README.md` — package location references (`src/packages/` → `fab/.kit/packages/` for production code), `rc-init.sh` delegation note

---

## Execution Order

- T002 blocks T003 (directories must exist before `git mv`)
- T003 blocks T007 (move files before deleting symlinks, so tests can still resolve during transition)
- T003 blocks T008, T009 (binaries must be at new location before test setup is updated)
- T004, T005, T006 are independent of each other, can run in parallel
- T007 blocks T012 (symlinks deleted before final test run)
- T012 blocks T013, T014 (tests pass before polish)
