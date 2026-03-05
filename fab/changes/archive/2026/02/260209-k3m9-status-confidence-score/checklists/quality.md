# Quality Checklist: Show confidence score in fab-status

**Change**: 260209-k3m9-status-confidence-score
**Generated**: 2026-02-09
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Confidence parsing: script parses `score`, `certain`, `confident`, `tentative`, `unresolved` from `.status.yaml`
- [x] CHK-002 Confidence rendering: output includes `Confidence:` line between Checklist and Next
- [x] CHK-003 Skill docs: `fab-status.md` documents the Confidence line with all three variants
- [x] CHK-004 Centralized docs: `change-lifecycle.md` mentions the Confidence line in `/fab-status` description

## Scenario Coverage

- [x] CHK-005 Normal display: with `unresolved: 0`, line shows score and counts without unresolved
- [x] CHK-006 Unresolved display: with `unresolved > 0`, line appends `, {N} unresolved`
- [x] CHK-007 Missing confidence: without confidence block, line shows `not yet scored`

## Edge Cases & Error Handling

- [x] CHK-008 Partial confidence block: script handles missing individual fields gracefully (defaults to empty/0)

## Documentation Accuracy

- [x] CHK-009 Output format in fab-status.md matches actual script output
- [x] CHK-010 Change-lifecycle.md description is consistent with actual behavior

## Cross References

- [x] CHK-011 Changelog entry added to change-lifecycle.md referencing this change

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
