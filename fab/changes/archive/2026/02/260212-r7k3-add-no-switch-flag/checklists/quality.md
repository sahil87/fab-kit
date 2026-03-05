# Quality Checklist: Add --no-switch Flag to fab-new

**Change**: 260212-r7k3-add-no-switch-flag
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Optional --no-switch argument: `fab-new.md` Arguments section documents `--no-switch` as an optional flag
- [x] CHK-002 Step 9 conditional: Step 9 includes conditional logic that skips `/fab-switch` when `--no-switch` is present
- [x] CHK-003 Contextual Next line: Output section shows different Next line when `--no-switch` is used vs not used

## Behavioral Correctness
- [x] CHK-004 Default behavior unchanged: When `--no-switch` is NOT used, the existing Step 9 behavior (call `/fab-switch`) is preserved exactly
- [x] CHK-005 No fab/current write: When `--no-switch` is used, the skill explicitly states `fab/current` is NOT modified
- [x] CHK-006 No branch integration: When `--no-switch` is used, no git branch is created or checked out

## Scenario Coverage
- [x] CHK-007 Scenario: --no-switch present: Skill instructions cover the case where `--no-switch` is provided — folder, status, and brief are created; fab/current and branch are not touched
- [x] CHK-008 Scenario: --no-switch absent: Skill instructions confirm existing behavior is unmodified when flag is not present
- [x] CHK-009 Next line with --no-switch: Output shows `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff`
- [x] CHK-010 Next line without --no-switch: Output shows `Next: /fab-continue or /fab-ff (fast-forward all planning)`

## Edge Cases & Error Handling
- [x] CHK-011 No output Branch line: When `--no-switch` is used, output does not include a `Branch:` line

## Documentation Accuracy
- [x] CHK-012 Skill file only: Only `fab/.kit/skills/fab-new.md` is modified — no other files in `fab/.kit/` are changed

## Cross References
- [x] CHK-013 Consistent with fab-discuss pattern: The `--no-switch` Next line matches the existing "not activated" pattern from `/fab-discuss`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
