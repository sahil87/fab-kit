# Tasks: Scriptify Fab-Archive

**Change**: 260303-hcq9-scriptify-fab-archive
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/scripts/lib/archiveman.sh` with boilerplate: shebang, `set -euo pipefail`, `LIB_DIR`/`FAB_ROOT` resolution, CLI dispatch skeleton (`archive`, `restore`, `list`, `--help`, error for unknown subcommands), and `--help` output
- [x] T002 Create `src/lib/archiveman/` test directory with `test.bats` containing setup/teardown fixtures: temp dir, copy real archiveman.sh, copy resolve.sh, stub changeman.sh (handle `resolve`, `switch --blank`, `switch <name>`)

## Phase 2: Core Implementation

- [x] T003 Implement `cmd_archive` in `fab/.kit/scripts/lib/archiveman.sh`: parse `<change>` and `--description` args, resolve via `resolve.sh --folder`, clean `.pr-done`, move folder to `fab/changes/archive/` (create dir if needed), output YAML with `action`, `name`, `clean`, `move` fields. Do not implement index or pointer yet.
- [x] T004 Implement archive index management in `cmd_archive`: create `index.md` with `# Archive Index` header if missing, prepend `- **{folder-name}** — {description}` entry after header, output `index: created` or `index: updated`
- [x] T005 Implement archive backfill in `cmd_archive`: scan `fab/changes/archive/` for folders not present in `index.md`, append missing entries with `(no description — pre-index archive)` placeholder, run on every invocation (no-op when all indexed)
- [x] T006 Implement archive pointer clearing in `cmd_archive`: check active change via `changeman.sh resolve`, if matches → `changeman.sh switch --blank` and output `pointer: cleared`, else output `pointer: skipped`
- [x] T007 Implement `resolve_archive` helper function in `fab/.kit/scripts/lib/archiveman.sh`: case-insensitive substring matching against `fab/changes/archive/` folder names (same logic as `resolve.sh` but scanning archive), handle exact/single/multiple/no match with appropriate exit codes and error messages
- [x] T008 Implement `cmd_restore` in `fab/.kit/scripts/lib/archiveman.sh`: parse `<change>` and `--switch` args, resolve via `resolve_archive`, move folder from `archive/` to `fab/changes/`, remove index entry, optionally call `changeman.sh switch {name}`, output YAML with `action`, `name`, `move`, `index`, `pointer` fields
- [x] T009 Implement `cmd_list` in `fab/.kit/scripts/lib/archiveman.sh`: list archived folder names (one per line), exclude `index.md`, handle missing/empty archive gracefully (exit 0)

## Phase 3: Integration & Tests

- [x] T010 Write BATS tests for `archive` subcommand in `src/lib/archiveman/test.bats`: happy path (move + index + pointer), `.pr-done` cleanup, no `.pr-done`, creates archive dir, pointer skip when not active, missing `--description`, YAML output validation
- [x] T011 Write BATS tests for archive index management in `src/lib/archiveman/test.bats`: creates new index with header, prepends to existing index, entry format, backfill for unindexed folders, backfill no-op when all indexed
- [x] T012 Write BATS tests for `restore` subcommand in `src/lib/archiveman/test.bats`: happy path (move + index remove + pointer), `--switch` flag, no `--switch`, resolution (exact, substring, 4-char ID, multiple match error, no match error), already-in-changes resumability, index entry removal, missing index entry
- [x] T013 Write BATS tests for `list` subcommand and CLI edge cases in `src/lib/archiveman/test.bats`: list with folders, empty archive, missing archive dir, excludes `index.md`, `--help` output, no subcommand error, unknown subcommand error

## Phase 4: Skill Update

- [x] T014 Update `.claude/skills/fab-archive/SKILL.md`: replace Steps 1-3 and Step 5 with single `archiveman.sh archive <change> --description "..."` call, replace restore Steps 1-3 with `archiveman.sh restore <change> [--switch]` call, keep backlog matching (Step 4) as-is, update skill to construct user-facing report from YAML output

---

## Execution Order

- T001 blocks T003–T009 (script skeleton needed before subcommands)
- T002 blocks T010–T013 (test fixtures needed before test cases)
- T003 blocks T004 (move logic needed before index)
- T004 blocks T005 (index creation needed before backfill)
- T003–T006 block T010 (archive implementation needed before archive tests)
- T007 blocks T008 (resolve_archive needed before restore)
- T007–T008 block T012 (restore implementation needed before restore tests)
- T009 blocks T013 (list implementation needed before list tests)
- T010–T013 block T014 (tests passing before skill update)
