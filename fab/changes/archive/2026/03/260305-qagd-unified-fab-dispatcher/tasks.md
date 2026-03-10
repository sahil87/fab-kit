# Tasks: Unified Fab Dispatcher

**Change**: 260305-qagd-unified-fab-dispatcher
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Rename Go binary: update `justfile` `build-go` target to output to `fab/.kit/bin/fab-go` instead of `fab/.kit/bin/fab`
- [x] T002 [P] Update `.gitignore`: replace `fab/.kit/bin/fab` with `fab/.kit/bin/fab-go` and `fab/.kit/bin/fab-rust`
- [x] T003 [P] Rename existing Go binary from `fab/.kit/bin/fab` to `fab/.kit/bin/fab-go` (so the dispatcher can take its place)

## Phase 2: Core Implementation

- [x] T004 Create shell dispatcher at `fab/.kit/bin/fab`: POSIX-compatible script with `--version` handling, backend priority chain (fab-rust → fab-go → shell), stderr diagnostic `[fab] using shell backend` on fallback, and `case` routing table for 7 commands (resolve, status, log, preflight, change, score, archive with arg injection)
- [x] T005 Remove `_fab_bin` shim blocks from all 7 shell scripts: `fab/.kit/scripts/lib/resolve.sh` (lines 13-17), `statusman.sh` (16-20), `logman.sh` (16-20), `preflight.sh` (4-8), `changeman.sh` (17-21), `calc-score.sh` (4-8), `archiveman.sh` (16-24)

## Phase 3: Integration & Edge Cases

- [x] T006 Update `fab/.kit/scripts/batch-fab-switch-change.sh`: replace `CHANGEMAN="${SCRIPT_DIR}/lib/changeman.sh"` with `FAB_BIN="$KIT_DIR/bin/fab"`, update resolve call at line 112 to `"$FAB_BIN" change resolve "$change"`
- [x] T007 [P] Update `fab/.kit/scripts/batch-fab-archive-change.sh`: replace `CHANGEMAN="${SCRIPT_DIR}/lib/changeman.sh"` with `FAB_BIN="$KIT_DIR/bin/fab"`, update resolve call at line 113 to `"$FAB_BIN" change resolve "$change"`
- [x] T008 Update parity tests: change `fabBinary()` in `src/go/fab/test/parity/parity_test.go` line 41 to reference `fab-go` instead of `fab`

## Phase 4: Polish

- [x] T009 Update `fab/.kit/skills/_scripts.md`: reframe from "Go binary primary + shell shim fallback" to "shell dispatcher + backend priority chain + pure shell implementations". Remove legacy calling convention section, update command mapping table
- [x] T010 Verify end-to-end: run `fab/.kit/bin/fab preflight` and `fab/.kit/bin/fab --version` to confirm dispatcher works with the Go backend present

---

## Execution Order

- T001 and T003 must complete before T004 (binary must be renamed before dispatcher takes its path)
- T004 must complete before T005 (dispatcher must exist before removing shims, to avoid breaking the entry point)
- T005 is independent of T006-T008
- T006 and T007 are independent of each other [P]
- T009 can run after T004-T005
- T010 runs last (end-to-end verification)
