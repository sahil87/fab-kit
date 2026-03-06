# Quality Checklist: Fix stale shell script references

**Change**: 260306-7arg-fix-stale-shell-refs
**Generated**: 2026-03-06
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Orphaned file deletion: All 20 files listed in spec are deleted from the repository
- [x] CHK-002 Empty directory removal: `src/lib/` (all 8 subdirs + parent) and `src/sync/` are gone
- [x] CHK-003 Script invocation guide: 4 new subcommand rows present in `_scripts.md` status table
- [x] CHK-004 git-pr fix: Step 4 passes `<name>` directly, no `.status.yaml` path derivation

## Behavioral Correctness

- [x] CHK-005 Test pipeline: `just test` passes after file deletion
- [x] CHK-006 Hook tests: `src/hooks/test-on-session-start.bats` and `test-on-stop.bats` still run and pass

## Scenario Coverage

- [x] CHK-007 All orphaned files deleted: No files remain in `src/lib/` or `src/sync/`
- [x] CHK-008 New subcommands positioned correctly: Rows appear between `set-confidence-fuzzy` and `progress-line`
- [x] CHK-009 add-pr uses change reference: Step 4 has 3 items (not 4), no path construction

## Code Quality

- [x] CHK-010 Pattern consistency: `_scripts.md` table rows follow existing format (Subcommand | Usage | Purpose)
- [x] CHK-011 No unnecessary duplication: No redundant documentation of the same subcommands

## Documentation Accuracy

- [x] CHK-012 `_scripts.md` subcommand signatures match actual Go binary help output
- [x] CHK-013 `git-pr.md` Step 4 matches the `<change>` argument convention documented in `_scripts.md`

## Cross References

- [x] CHK-014 No remaining references to deleted shell scripts in `_scripts.md` or `git-pr.md`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
