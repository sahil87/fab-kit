# Quality Checklist: Rename Batch Scripts and Add Batch Archive

**Change**: 260213-v3rn-batch-commands
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Batch naming: `batch-new-backlog.sh` and `batch-switch-change.sh` exist at `fab/.kit/scripts/`
- [x] CHK-002 Old files deleted: `fab-batch-new.sh` and `fab-batch-switch.sh` no longer exist
- [x] CHK-003 Archive script: `batch-archive-change.sh` exists and is executable
- [x] CHK-004 Archive script options: `--list`, `--all`, `-h`/`--help` all function correctly

## Behavioral Correctness

- [x] CHK-005 Renamed scripts: content identical to originals except comment header

## Scenario Coverage

- [x] CHK-006 Archive single change: correct tmux tab name and `/fab-archive <change>` command
- [x] CHK-007 Archive --all: only `hydrate:done` changes selected, others skipped with warning
- [x] CHK-008 Archive --list: shows only archivable (hydrate:done) changes
- [x] CHK-009 No archivable changes: exits with code 1 and message

## Edge Cases & Error Handling

- [x] CHK-010 Missing `.status.yaml`: change skipped with warning
- [x] CHK-011 Substring matching: resolves change names correctly (single match)
- [x] CHK-012 Multiple substring matches: warning printed, change skipped

## Documentation Accuracy

- [x] CHK-013 Script comment headers reference correct new filenames

## Cross References

- [x] CHK-014 No stale references to old script names (`fab-batch-new`, `fab-batch-switch`) in codebase

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
