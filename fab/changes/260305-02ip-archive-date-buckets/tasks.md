# Tasks: Archive Date Buckets

**Change**: 260305-02ip-archive-date-buckets
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add `parse_date_bucket` helper function to `fab/.kit/scripts/lib/archiveman.sh` — extract yyyy and mm from folder name YYMMDD prefix, return as `yyyy mm` pair

## Phase 2: Core Implementation

- [x] T002 Update `cmd_archive` in `fab/.kit/scripts/lib/archiveman.sh` — use `parse_date_bucket` to compute destination path `archive/yyyy/mm/{name}`, update `mkdir -p` and collision check to use bucketed path
- [x] T003 Update `resolve_archive` in `fab/.kit/scripts/lib/archiveman.sh` — scan `archive/*/*/*/` (or recursive find) instead of `archive/*/` to find change folders in nested yyyy/mm/ structure
- [x] T004 Update `cmd_list` in `fab/.kit/scripts/lib/archiveman.sh` — traverse `archive/yyyy/mm/` hierarchy, output folder names without path prefix
- [x] T005 Update `backfill_index` in `fab/.kit/scripts/lib/archiveman.sh` — scan nested `archive/yyyy/mm/` structure instead of `archive/*/`

## Phase 3: Integration & Edge Cases

- [x] T006 Add `cmd_migrate` function to `fab/.kit/scripts/lib/archiveman.sh` — scan for flat entries (directories directly under archive/ that aren't yyyy/ dirs or index.md), move each to `archive/yyyy/mm/` using `parse_date_bucket`, output summary
- [x] T007 Register `migrate` subcommand in CLI dispatch section of `fab/.kit/scripts/lib/archiveman.sh` and update `show_help`

## Phase 4: Polish

- [x] T008 Update `fab/.kit/skills/fab-archive.md` if it references flat archive paths — verify docs are consistent with new bucketed structure
- [x] T009 Fix `resolve_archive` subshell bug — restructure to use globals (`_ARCHIVE_RESOLVED_NAME`, `_ARCHIVE_RESOLVED_DIR`) instead of stdout, call directly without `$(...)` in `cmd_restore`
- [x] T010 Add collision check to `cmd_migrate` — skip entries already at destination to handle partial-failure re-runs
- [x] T011 Update `changeman.sh list --archive` to scan both flat and nested `archive/yyyy/mm/` structure

---

## Execution Order

- T001 blocks T002, T003, T004, T005, T006 (all depend on date parsing helper)
- T002-T005 are independent of each other (different functions)
- T006 depends on T001 (uses parse_date_bucket)
- T007 depends on T006 (registers its subcommand)
- T008 is independent
