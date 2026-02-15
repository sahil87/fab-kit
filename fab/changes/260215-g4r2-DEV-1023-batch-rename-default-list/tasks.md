# Tasks: Batch Script Rename and Default List Behavior

**Change**: 260215-g4r2-DEV-1023-batch-rename-default-list
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Rename Scripts

- [x] T001 [P] Rename `fab/.kit/scripts/batch-new-backlog.sh` → `batch-fab-new-backlog.sh` via `git mv`
- [x] T002 [P] Rename `fab/.kit/scripts/batch-switch-change.sh` → `batch-fab-switch-change.sh` via `git mv`
- [x] T003 [P] Rename `fab/.kit/scripts/batch-archive-change.sh` → `batch-fab-archive-change.sh` via `git mv`

## Phase 2: Core Implementation

- [x] T004 [P] In `fab/.kit/scripts/batch-fab-new-backlog.sh`: change no-arg block (currently `usage; exit 1`) to `set -- --list`; update comment header, `usage()` Usage line and Examples to use `batch-fab-new-backlog`
- [x] T005 [P] In `fab/.kit/scripts/batch-fab-switch-change.sh`: change no-arg block (currently `usage; exit 1`) to `set -- --list`; update comment header, `usage()` Usage line and Examples to use `batch-fab-switch-change`
- [x] T006 [P] In `fab/.kit/scripts/batch-fab-archive-change.sh`: change no-arg block (currently `usage; exit 1`) to `set -- --list`; update comment header, `usage()` Usage line and Examples to use `batch-fab-archive-change`

## Phase 3: Documentation

- [x] T007 [P] Update `docs/specs/architecture.md`: change `batch-` prefix row to `batch-fab-` in Script Naming Convention table; update all three script names in Batch Scripts table
- [x] T008 [P] Update `docs/memory/fab-workflow/kit-architecture.md`: update directory tree listing with new `batch-fab-*` names; update Batch Scripts section naming pattern from `batch-{verb}-{entity}.sh` to `batch-fab-{verb}-{entity}.sh`; update all script name references

---

## Execution Order

- T001-T003 are independent (parallel renames)
- T004-T006 depend on T001-T003 respectively (rename must happen before editing)
- T007-T008 are independent of each other but depend on T001-T003 (new names must exist for reference accuracy)
