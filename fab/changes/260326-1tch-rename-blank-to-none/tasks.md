# Tasks: Rename --blank to --none in fab-switch

**Change**: 260326-1tch-rename-blank-to-none
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Rename `SwitchBlank` → `SwitchNone` in `src/go/fab/internal/change/change.go`, update output string from `already blank` to `already deactivated`
- [x] T002 [P] Rename `--blank` flag to `--none` in `src/go/fab/cmd/fab/change.go`, update error message from `switch requires <name> or --blank` to `switch requires <name> or --none`
- [x] T003 Update `change.SwitchBlank(fabRoot)` → `change.SwitchNone(fabRoot)` in `src/go/fab/internal/archive/archive.go`
- [x] T004 Update tests in `src/go/fab/internal/change/change_test.go`: rename `TestSwitchBlank` → `TestSwitchNone`, rename `TestSwitchBlank_AlreadyBlank` → `TestSwitchNone_AlreadyDeactivated`, update expected output assertion from `already blank` to `already deactivated`

## Phase 2: Documentation

- [x] T005 [P] Replace all `--blank` with `--none` in `fab/.kit/skills/fab-switch.md` (heading, arguments, deactivation flow, output sections)
- [x] T006 [P] Update switch row in `fab/.kit/skills/_cli-fab.md` from `--blank` to `--none`
- [x] T007 [P] Replace all `--blank` with `--none` in `docs/specs/skills/SPEC-fab-switch.md`
- [x] T008 [P] Replace all `--blank` with `--none` and `already blank` with `already deactivated` in `docs/memory/fab-workflow/change-lifecycle.md`

## Phase 3: Verification

- [x] T009 Run `go test ./...` from `src/go/fab/` to verify all tests pass
- [x] T010 Run `grep -r "blank" src/go/fab/` and `grep -r "\-\-blank" fab/.kit/skills/ docs/` to verify no stale references remain

---

## Execution Order

- T001 and T002 are independent, can run in parallel
- T003 depends on T001 (needs `SwitchNone` to exist)
- T004 depends on T001 (needs `SwitchNone` and new output string)
- T005-T008 are independent, can run in parallel
- T009 depends on T001-T004
- T010 depends on T001-T008
