# Tasks: Fix stale shell script references

**Change**: 260306-7arg-fix-stale-shell-refs
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Cleanup

- [x] T001 [P] Delete all 20 orphaned files in `src/lib/` (9 bats tests, 5 SPECs, 5 test-simple/helper scripts, 1 sensitivity.sh)
- [x] T002 [P] Delete `src/sync/test-5-sync-hooks.bats`
- [x] T003 Remove empty directories: all 8 `src/lib/*/` subdirs, `src/lib/`, and `src/sync/`

## Phase 2: Core Implementation

- [x] T004 Add 4 missing subcommands (`add-issue`, `get-issues`, `add-pr`, `get-prs`) to `fab/.kit/skills/_scripts.md` status table, after `set-confidence-fuzzy` and before `progress-line`
- [x] T005 Fix `fab/.kit/skills/git-pr.md` Step 4 (lines 229-231): remove intermediate `.status.yaml` path derivation, pass `<name>` directly to `fab status add-pr`

## Phase 3: Verification

- [x] T006 Run `just test` and verify all tests pass after cleanup (also fixed stale calc-score.sh stub in `src/scripts/pipeline/test.bats` → fab dispatcher stub)

---

## Execution Order

- T001 and T002 are parallel (independent file deletions)
- T003 depends on T001 and T002 (directories must be empty before removal)
- T004 and T005 are independent of T001-T003
- T006 depends on T001-T003 (verifies the test pipeline after file deletion)
