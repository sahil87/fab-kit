# Tasks: Add .gitkeep to fab/changes/archive/

**Change**: 260216-pr1u-DEV-1017-add-archive-gitkeep
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add `"$fab_dir/changes/archive"` to the directory creation `for` loop in `fab/.kit/scripts/fab-sync.sh` (line 74)
- [x] T002 Add conditional `.gitkeep` creation for `fab/changes/archive/.gitkeep` after the existing `fab/changes/.gitkeep` block in `fab/.kit/scripts/fab-sync.sh` (after line 83)

## Phase 2: Documentation & Tests

- [x] T003 [P] Update `src/lib/fab-sync/SPEC-fab-sync.md` Section "1. Directory Creation" to mention `fab/changes/archive/` and `fab/changes/archive/.gitkeep`
- [x] T004 [P] Add bats test case for `fab/changes/archive/.gitkeep` creation in `src/lib/fab-sync/test.bats`, following the existing `"creates fab/changes/.gitkeep"` test pattern

---

## Execution Order

- T001 blocks T002 (directory must exist before .gitkeep)
- T003 and T004 are independent of each other and can run after T001+T002
