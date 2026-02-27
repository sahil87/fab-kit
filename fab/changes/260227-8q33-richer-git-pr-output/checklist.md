# Quality Checklist: Richer Git PR Output

**Change**: 260227-8q33-richer-git-pr-output
**Generated**: 2026-02-27
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Fix Artifact Link Rows: Intake and Spec are separate `Field | Detail` rows (not sibling cells)
- [x] CHK-002 Add Confidence Row: Context table includes `| Confidence | {score} / 5.0 |`
- [x] CHK-003 Add Pipeline Row: Context table includes `| Pipeline | {stages} |` with `→`-joined done stages
- [x] CHK-004 Context Table Row Order: Rows appear in fixed order (Type, Change, Confidence, Pipeline, Intake, Spec)
- [x] CHK-005 No Changes to Tier 2: Lightweight template unchanged
- [x] CHK-006 Read Status via File Access: No new scripts required — reads `.status.yaml` directly

## Behavioral Correctness
- [x] CHK-007 Spec row omitted when `spec.md` absent (not an empty cell)
- [x] CHK-008 Pipeline shows only `done` stages in canonical order

## Scenario Coverage
- [x] CHK-009 Both artifacts exist: 6-row Context table with all fields
- [x] CHK-010 Only intake exists: 5-row Context table, no Spec row
- [x] CHK-011 Full pipeline completed: `intake → spec → tasks → apply → review → hydrate`
- [x] CHK-012 Partial pipeline: only done stages shown
- [x] CHK-013 Default confidence (0.0): row still rendered as `0.0 / 5.0`

## Edge Cases & Error Handling
- [x] CHK-014 Tier 2 PR: no Confidence/Pipeline/Intake/Spec rows — only Type row + housekeeping note

## Code Quality
- [x] CHK-015 Pattern consistency: New template prose follows existing skill file conventions
- [x] CHK-016 No unnecessary duplication: No duplicated logic between Tier 1 and Tier 2

## Documentation Accuracy
- [x] CHK-017 Memory file updated: execution-skills.md reflects expanded Tier 1 Context table

## Cross References
- [x] CHK-018 git-pr.md internal consistency: instruction text matches template example

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
