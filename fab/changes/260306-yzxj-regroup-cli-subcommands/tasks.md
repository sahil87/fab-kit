# Tasks: Regroup CLI Subcommands

**Change**: 260306-yzxj-regroup-cli-subcommands
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Go Implementation

- [x] T001 Refactor `src/fab-go/cmd/fab/archive.go` — rename `archiveCmd` → `changeArchiveCmd`, `archiveRestoreCmd` → `changeRestoreCmd`, `archiveListCmd` → `changeArchiveListCmd`. Update Cobra `Use` for archive-list to `archive-list`. Remove the `cmd.AddCommand` call inside `changeArchiveCmd` (restore and archive-list are now siblings, not children)
- [x] T002 Update `src/fab-go/cmd/fab/change.go` — add `changeArchiveCmd()`, `changeRestoreCmd()`, `changeArchiveListCmd()` to the `cmd.AddCommand` call in `changeCmd()`
- [x] T003 Update `src/fab-go/cmd/fab/main.go` — remove `archiveCmd()` from the root `AddCommand` list

## Phase 2: Test Updates

- [x] T004 Update `src/fab-go/test/parity/archive_test.go` — change `runGo` calls from `"archive", ...` to `"change", "archive", ...` and `"change", "archive-list"` respectively

## Phase 3: Skill and Script Documentation

- [x] T005 [P] Update `fab/.kit/skills/_scripts.md` — reorganize into three sections (Change Lifecycle, Pipeline & Status, Plumbing), move archive docs into `fab change` section, remove standalone `## fab archive` section, update Command Reference table
- [x] T006 [P] Update `fab/.kit/skills/fab-archive.md` — replace all `fab archive` CLI invocations with `fab change archive`, `fab change restore`, `fab change archive-list`

## Phase 4: Specs and Memory Updates

- [x] T007 [P] Update `docs/specs/skills/SPEC-fab-archive.md` — replace `fab archive` CLI references with `fab change archive`/`restore`/`archive-list` in flow diagram and tools table
- [x] T008 [P] Update `docs/memory/fab-workflow/kit-architecture.md` — update command reference line from `fab archive` to reflect new paths under `fab change`

## Phase 5: Build and Verify

- [x] T009 Build Go binary — run `go build` in `src/fab-go/` to verify compilation
- [x] T010 Run parity tests — run `go test ./test/parity/ -run TestArchive` to verify updated tests pass

---

## Execution Order

- T001 blocks T002, T003 (archive functions must be renamed before re-wiring)
- T002 + T003 block T009 (build needs both wiring changes)
- T009 blocks T010 (tests need the built binary)
- T005, T006, T007, T008 are independent of each other and of Go changes
