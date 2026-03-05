# Quality Checklist: Swap batch-fab-archive-change defaults

**Change**: 260305-obib-swap-batch-archive-defaults
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 No-argument default: Running with no args triggers `--all` behavior (archive all eligible)
- [x] CHK-002 --list flag: Running with `--list` shows archivable changes without archiving
- [x] CHK-003 Usage text: Help output documents new default and `--list` as preview option
- [x] CHK-004 Existing flags: `--all`, `-h`/`--help`, and positional arguments still work

## Behavioral Correctness
- [x] CHK-005 No-args with no archivable changes exits 1 with "No archivable changes found."
- [x] CHK-006 --list with no archivable changes shows "(none)" and exits 0

## Scenario Coverage
- [x] CHK-007 No arguments with archivable changes: archives all
- [x] CHK-008 Explicit --list: lists without archiving
- [x] CHK-009 Explicit --all: archives all (same as no-args)
- [x] CHK-010 Positional arguments: archives only specified changes

## Code Quality
- [x] CHK-011 Pattern consistency: Change follows existing script structure and conventions
- [x] CHK-012 No unnecessary duplication: No redundant code introduced

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
