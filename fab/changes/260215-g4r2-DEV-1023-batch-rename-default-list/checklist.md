# Quality Checklist: Batch Script Rename and Default List Behavior

**Change**: 260215-g4r2-DEV-1023-batch-rename-default-list
**Generated**: 2026-02-15
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 No-arg list behavior: All three scripts show `--list` output when invoked with no arguments
- [x] CHK-002 Help flags preserved: `-h` and `--help` display usage text on all three scripts
- [x] CHK-003 Script rename: Old filenames removed, new `batch-fab-*` filenames exist
- [x] CHK-004 Usage text updated: `usage()` in each script references the new filename

## Behavioral Correctness
- [x] CHK-005 No-arg fallthrough: `set -- --list` correctly rewrites positional params and falls through to `--list` case
- [x] CHK-006 Existing flags unaffected: `--all` and direct ID/name arguments still work as before

## Scenario Coverage
- [x] CHK-007 No-arg shows list (new-backlog): Verified `batch-fab-new-backlog.sh` with no args shows pending items
- [x] CHK-008 No-arg shows list (switch-change): Verified `batch-fab-switch-change.sh` with no args shows changes
- [x] CHK-009 No-arg shows list (archive-change): Verified `batch-fab-archive-change.sh` with no args shows archivable
- [x] CHK-010 Help flag scenario: Verified `-h`/`--help` displays usage with new script name

## Edge Cases & Error Handling
- [x] CHK-011 No-arg with empty data: Scripts handle no-arg when list is empty (no pending items / no changes / no archivable)

## Code Quality
- [x] CHK-012 Pattern consistency: `set -- --list` approach is consistent across all three scripts
- [x] CHK-013 No unnecessary duplication: Existing list helpers reused, no new code duplication

## Documentation Accuracy
- [x] CHK-014 Architecture spec: Prefix convention table shows `batch-fab-`, batch scripts table uses new names
- [x] CHK-015 Kit architecture memory: Directory tree and batch scripts section use new names and pattern

## Cross References
- [x] CHK-016 No stale references: No remaining references to old `batch-new-backlog.sh`, `batch-switch-change.sh`, `batch-archive-change.sh` names in memory or specs

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
