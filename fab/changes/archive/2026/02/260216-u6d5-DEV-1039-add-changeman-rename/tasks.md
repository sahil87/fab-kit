# Tasks: Add Rename Subcommand to changeman.sh

**Change**: 260216-u6d5-DEV-1039-add-changeman-rename
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add `cmd_rename` function to `fab/.kit/scripts/lib/changeman.sh` — parse `--folder` and `--slug` flags, validate slug format (reuse `new`'s regex), verify source folder exists under `fab/changes/`, extract `{YYMMDD}-{XXXX}` prefix (first two hyphen-separated segments), construct new folder name, check same-name and destination collision, `mv` folder, update `.status.yaml` `name` field via `sed`, conditionally update `fab/current`, call `$STAGEMAN log-command`, print new folder name to stdout
- [x] T002 [P] Update `show_help()` in `fab/.kit/scripts/lib/changeman.sh` — add rename subcommand usage line, flags section, and example
- [x] T003 [P] Add `rename` case to CLI dispatch in `fab/.kit/scripts/lib/changeman.sh` — route to `cmd_rename`, update header comment with new usage line

## Phase 2: Tests

- [x] T004 Add rename test section to `src/lib/changeman/test.bats` — happy path (folder renamed, .status.yaml updated, output correct), fab/current updated when active, fab/current unchanged when different, fab/current not created when absent, slug validation (leading/trailing hyphen rejected, uppercase accepted), missing source folder error, destination collision error, same-name error, missing --folder error, missing --slug error, stageman log-command called

## Phase 3: Documentation

- [x] T005 Update `src/lib/changeman/SPEC-changeman.md` — add rename to Subcommands table, add `### rename Subcommand` section with arguments, behavior steps, error cases; update description and Usage section

---

## Execution Order

- T002, T003 are independent of each other, can run alongside T001
- T004 depends on T001-T003 (needs working rename to test)
- T005 is independent, can run alongside T004
