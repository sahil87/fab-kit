# Quality Checklist: Archive Date Buckets

**Change**: 260305-02ip-archive-date-buckets
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Date-bucketed archive path: `cmd_archive` moves to `archive/yyyy/mm/{name}` with correct year/month derivation
- [x] CHK-002 Recursive archive resolution: `resolve_archive` finds folders at `archive/yyyy/mm/` depth
- [x] CHK-003 Nested list traversal: `cmd_list` outputs folder names from nested structure without path prefix
- [x] CHK-004 Nested backfill: `backfill_index` scans nested structure for unindexed folders
- [x] CHK-005 Migration subcommand: `cmd_migrate` moves flat entries into yyyy/mm buckets

## Behavioral Correctness

- [x] CHK-006 Collision detection uses bucketed path: error when `archive/yyyy/mm/{name}` already exists
- [x] CHK-007 Archive output YAML unchanged: action, name, clean, move, index, pointer fields still present
- [x] CHK-008 Restore output YAML unchanged: action, name, move, index, pointer fields still present
- [x] CHK-009 List output contract preserved: one folder name per line, no path prefix

## Scenario Coverage

- [x] CHK-010 Archive a change verifies correct yyyy/mm destination
- [x] CHK-011 Restore resolves from nested archive path
- [x] CHK-012 Partial name match works with nested archive
- [x] CHK-013 Multiple match error works with nested archive
- [x] CHK-014 Migration is idempotent — re-running on already-bucketed archive is safe
- [x] CHK-015 Mixed flat/bucketed migration only moves flat entries

## Edge Cases & Error Handling

- [x] CHK-016 Empty archive directory: list and resolve handle gracefully
- [x] CHK-017 Date parsing for boundary values: month 01, month 12, year 25, year 26

## Code Quality

- [x] CHK-018 Pattern consistency: new functions follow naming and structural patterns of existing archiveman.sh functions
- [x] CHK-019 No unnecessary duplication: date parsing extracted to shared helper, reused by archive and migrate

## Documentation Accuracy

- [x] CHK-020 `fab-archive.md` skill file references consistent with new bucketed archive paths

## Cross References

- [x] CHK-021 **N/A**: Memory file updates happen during hydrate phase, not apply/review

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
