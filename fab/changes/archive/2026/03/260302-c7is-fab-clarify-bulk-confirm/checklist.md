# Quality Checklist: Add bulk confirm mode to fab-clarify

**Change**: 260302-c7is-fab-clarify-bulk-confirm
**Generated**: 2026-03-02
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Bulk confirm detection: Step 1.5 checks `confident >= 3` AND `confident > tentative + unresolved` from Assumptions table
- [ ] CHK-002 Bulk confirm display: Confident assumptions shown as numbered list using original `#` column with Decision and Rationale
- [ ] CHK-003 No AskUserQuestion: Bulk confirm uses plain text output + conversational response, not per-item tool calls
- [ ] CHK-004 Response parsing: All 6 formats recognized — confirm (✓/ok/yes), bare number, free text change, explain (?), range, all
- [ ] CHK-005 Artifact update: Confirmed items upgraded to Certain with correct Rationale and S:95 in Scores
- [ ] CHK-006 Explanation re-prompt: One round of re-prompting for `?` items, unresolved items stay Confident
- [ ] CHK-007 Audit trail: Bulk confirm results appended as `### Session {date} (bulk confirm)` in Clarifications section
- [ ] CHK-008 Preamble documentation: `### Bulk Confirm (Confident Assumptions)` subsection added under `## Confidence Scoring` in `_preamble.md`
- [ ] CHK-009 Auto Mode exclusion: Bulk confirm NOT triggered in Auto Mode, only Suggest Mode

## Behavioral Correctness

- [ ] CHK-010 Step ordering: Step 1.5 executes after Step 1 (read artifact) and before Step 2 (taxonomy scan)
- [ ] CHK-011 Existing flow preserved: When bulk confirm is NOT triggered (below threshold), existing taxonomy scan runs as before
- [ ] CHK-012 Unmentioned items: Items not addressed in user response remain Confident (no forced confirmation)

## Scenario Coverage

- [ ] CHK-013 Many Confident dominate: 11 Confident, 0 Tentative/Unresolved → triggers bulk confirm
- [ ] CHK-014 Below threshold: 2 Confident → does NOT trigger
- [ ] CHK-015 Tentative/Unresolved dominate: 4 Confident, 4 Tentative+Unresolved → does NOT trigger
- [ ] CHK-016 Equal split: 3 Confident, 3 Tentative+Unresolved → does NOT trigger (strictly greater)
- [ ] CHK-017 Mixed response: Confirms, changes, and explanations in single response correctly parsed
- [ ] CHK-018 Confirm-all shorthand: `all ok` confirms every Confident item

## Edge Cases & Error Handling

- [ ] CHK-019 Explanation with no follow-up: Items not re-confirmed after explanation remain Confident
- [ ] CHK-020 Changed item: Decision column updated with user's new value, not just confirmed
- [ ] CHK-021 Bulk confirm then taxonomy: After bulk confirm, taxonomy scan operates on updated artifact

## Code Quality

- [ ] CHK-022 Pattern consistency: New skill sections follow existing fab-clarify.md structure (heading levels, formatting, instruction style)
- [ ] CHK-023 No unnecessary duplication: Preamble documentation doesn't repeat full flow details from fab-clarify.md — references the skill instead

## Documentation Accuracy

- [ ] CHK-024 Preamble content matches spec: Trigger conditions, flow location, and update behavior documented accurately
- [ ] CHK-025 Step numbering consistent: All references to Step 1.5, Step 1, Step 2 are internally consistent

## Cross References

- [ ] CHK-026 Preamble → fab-clarify link: Preamble subsection references `/fab-clarify` as implementing skill
- [ ] CHK-027 Step 2 note: Taxonomy scan section acknowledges Step 1.5 precedence

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
