# Quality Checklist: Rename --blank to --none in fab-switch

**Change**: 260326-1tch-rename-blank-to-none
**Generated**: 2026-03-26
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 --none flag: `fab change switch --none` deactivates the current change (removes symlink)
- [x] CHK-002 Error message: `fab change switch` with no args outputs `switch requires <name> or --none`
- [x] CHK-003 Go function rename: `SwitchBlank` renamed to `SwitchNone` with no dead references
- [x] CHK-004 Archive integration: `archive.go` calls `change.SwitchNone(fabRoot)` instead of `SwitchBlank`

## Behavioral Correctness
- [x] CHK-005 Deactivation output: Active change deactivation outputs `No active change.`
- [x] CHK-006 Already deactivated: When no symlink exists, outputs `No active change (already deactivated).`

## Scenario Coverage
- [x] CHK-007 TestSwitchNone: Test verifies deactivation removes symlink and outputs correct message
- [x] CHK-008 TestSwitchNone_AlreadyDeactivated: Test verifies already-deactivated message contains `already deactivated`

## Documentation Accuracy
- [x] CHK-009 fab-switch.md: All `--blank` replaced with `--none`, heading updated
- [x] CHK-010 _cli-fab.md: Switch row shows `--none`
- [x] CHK-011 SPEC-fab-switch.md: All `--blank` replaced with `--none`
- [x] CHK-012 change-lifecycle.md: All `--blank` replaced with `--none`, `already blank` → `already deactivated`

## Cross References
- [x] CHK-013 No stale references: `grep -r "blank"` in Go source and `grep -r "\-\-blank"` in skills/docs returns no hits

## Code Quality
- [x] CHK-014 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [x] CHK-015 No unnecessary duplication: Existing utilities reused where applicable

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
