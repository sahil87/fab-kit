# Tasks: Remove .pr-done Sentinel

**Change**: 260409-2v5s-remove-pr-done-sentinel
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Go Code

- [x] T001 Remove `.pr-done` cleanup from `src/go/fab/internal/archive/archive.go`: delete the `Clean` field from `ArchiveResult` struct, remove the `.pr-done` stat+remove block (lines 66-72), remove `clean` from `FormatArchiveYAML` output
- [x] T002 Update `src/go/fab/internal/archive/archive_test.go`: remove `Clean` field references from `TestFormatArchiveYAML`, verify no test asserts `.pr-done` behavior

## Phase 2: Skill Files

- [x] T003 [P] Remove Step 4d from `src/kit/skills/git-pr.md`: delete the "Step 4d: Write PR Sentinel" section (lines 273-281)
- [x] T004 [P] Update `src/kit/skills/fab-archive.md`: remove "Clean" bullet from Step 2 description, remove `clean:` rows from report format table, remove `Cleaned:` line from output example, update Purpose paragraph to remove "clean" reference
- [x] T005 [P] Update `src/kit/skills/_cli-fab.md`: change `archive` row description from "Clean .pr-done, move to archive/, update index, clear pointer" to "Move to archive/, update index, clear pointer"

## Phase 3: Spec Diagrams

- [x] T006 [P] Update `docs/specs/skills/SPEC-git-pr.md`: remove Step 4d from flow diagram
- [x] T007 [P] Update `docs/specs/skills/SPEC-fab-archive.md`: remove `.pr-done` reference from archive flow step

---

## Execution Order

- T001 blocks T002 (struct change must exist before test update)
- T003-T007 are independent of each other and T001-T002