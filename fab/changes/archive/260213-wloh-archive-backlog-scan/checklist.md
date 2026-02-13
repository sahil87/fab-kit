# Quality Checklist: Broaden Archive Backlog Scanning

**Change**: 260213-wloh-archive-backlog-scan
**Generated**: 2026-02-13
**Spec**: `spec.md`

## Functional Completeness
<!-- Every requirement in spec.md has working implementation -->
- [x] CHK-001 Preserve Exact-ID Auto-Marking: Existing Step 7 exact-ID behavior is preserved unchanged in the rewritten step
- [x] CHK-002 Secondary Keyword Scan: Step 7 includes keyword extraction from brief title + Why, stop word filtering, normalization, and 2-keyword threshold matching
- [x] CHK-003 Interactive Confirmation: Batch prompt format matches spec (numbered list, comma-separated selection, "none" option)
- [x] CHK-004 Backlog File Mutation: Done items are moved from Backlog section to Done section with checkbox changed to `[x]`

## Behavioral Correctness
<!-- Changed requirements behave as specified, not as before -->
- [x] CHK-005 Auto-Mode Skip: Keyword scan is explicitly skipped when archive runs via `/fab-ff` or `/fab-fff`; only exact-ID runs
- [x] CHK-006 Already-Marked Exclusion: Items marked by exact-ID check are excluded from keyword scan candidates

## Scenario Coverage
<!-- Key scenarios from spec.md have been exercised -->
- [x] CHK-007 Scenario: Keywords Match a Backlog Item — matching logic described with concrete example
- [x] CHK-008 Scenario: No Keyword Matches — silent proceed behavior documented
- [x] CHK-009 Scenario: Auto Mode Skips Keyword Scan — auto-mode guard documented
- [x] CHK-010 Scenario: User Confirms/Declines — batch selection and "none" behavior documented

## Edge Cases & Error Handling
<!-- Error states, boundary conditions, failure modes -->
- [x] CHK-011 Edge case: `fab/backlog.md` missing — keyword scan skips silently
- [x] CHK-012 Edge case: No unchecked items in backlog — no candidates, proceeds silently
- [x] CHK-013 Edge case: All candidates declined by user — proceeds normally

## Documentation Accuracy
<!-- Project-specific: from config.yaml extra_categories -->
- [x] CHK-014 Archive output template updated to show keyword scan results
- [x] CHK-015 Error handling table updated with new edge cases

## Cross References
<!-- Project-specific: from config.yaml extra_categories -->
- [x] CHK-016 Step references in "Order of Operations" section remain consistent with new sub-step structure

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (archive)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
