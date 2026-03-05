# Tasks: Consolidate .status.yaml Ownership into Stageman

**Change**: 260213-puow-consolidate-status-reads
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Rename `fab/.kit/scripts/stageman.sh` → `fab/.kit/scripts/_stageman.sh`
- [x] T002 Update dev symlink: remove `src/stageman/stageman.sh`, create `src/stageman/_stageman.sh` → `../../fab/.kit/scripts/_stageman.sh`
- [x] T003 Create skeleton `fab/.kit/scripts/_resolve-change.sh` with header, `set -euo pipefail`, and empty `resolve_change()` function stub
- [x] T004 Create `src/resolve-change/` directory with symlink `_resolve-change.sh` → `../../fab/.kit/scripts/_resolve-change.sh`

## Phase 2: Core Implementation

- [x] T005 [P] Add `get_progress_map` function to `fab/.kit/scripts/_stageman.sh` — accepts status file path, outputs `stage:state` lines, defaults missing stages to `pending`
- [x] T006 [P] Add `get_checklist` function to `fab/.kit/scripts/_stageman.sh` — accepts status file path, outputs `generated:{val}`, `completed:{val}`, `total:{val}` lines with defaults
- [x] T007 [P] Add `get_confidence` function to `fab/.kit/scripts/_stageman.sh` — accepts status file path, outputs `certain:{val}`, `confident:{val}`, `tentative:{val}`, `unresolved:{val}`, `score:{val}` lines with defaults
- [x] T008 Refactor `get_current_stage` in `fab/.kit/scripts/_stageman.sh` to use `get_progress_map` internally instead of raw `grep | sed`
- [x] T009 Implement `resolve_change` function in `fab/.kit/scripts/_resolve-change.sh` — accepts `fab_root` and optional `override`, sets `RESOLVED_CHANGE_NAME`, handles all scenarios (exact match, substring, multiple, no match, no fab/current, missing changes dir)

## Phase 3: Integration

- [x] T010 Refactor `fab/.kit/scripts/fab-preflight.sh` — source `_resolve-change.sh`, replace change resolution block (lines ~17-83) with `resolve_change` call, replace inline progress/stage/checklist/confidence extraction with stageman accessor calls, update `source` line to `_stageman.sh`
- [x] T011 Refactor `fab/.kit/scripts/fab-status.sh` — source `_resolve-change.sh`, replace change resolution block (lines ~21-87) with `resolve_change` call, remove `get_field`/`get_nested` helpers, replace inline progress/stage/checklist/confidence extraction with stageman accessor calls, update `source` line to `_stageman.sh`
- [x] T012 Update `src/stageman/test-simple.sh` — update `source` line from `stageman.sh` to `_stageman.sh`, add smoke tests for `get_progress_map`, `get_checklist`, `get_confidence`
- [x] T013 Update `src/stageman/test.sh` — update `source` line, add comprehensive tests for the three new accessor functions and refactored `get_current_stage`
- [x] T014 Update `src/preflight/test-simple.sh` — update `cp` line from `stageman.sh` to `_stageman.sh`

## Phase 4: Polish

- [x] T015 [P] Update `src/stageman/README.md` — add `get_progress_map`, `get_checklist`, `get_confidence` to API Reference tables
- [x] T016 [P] Create `src/resolve-change/README.md` — API docs for `resolve_change` function (signature, arguments, return values, error messages, examples)
- [x] T017 [P] Create `src/resolve-change/test-simple.sh` — smoke test: source library, call `resolve_change` with a valid fab root, verify `RESOLVED_CHANGE_NAME` is set
- [x] T018 [P] Create `src/resolve-change/test.sh` — comprehensive test suite: exact match, substring single, substring multiple, no match, no override (fab/current), no active change, missing changes dir

---

## Execution Order

- T001 blocks T002, T005-T008, T010-T014 (rename must happen first)
- T003 blocks T009 (skeleton before implementation)
- T004 blocks T016-T018 (dev folder before its contents)
- T005-T007 block T008 (accessors before get_current_stage refactor)
- T008 blocks T010, T011 (stageman complete before consumer refactor)
- T009 blocks T010, T011 (resolve-change complete before consumer refactor)
- T010, T011 block T014 (preflight refactored before its test updates)
