# Quality Checklist: Clarify Tentative First

**Change**: 260416-hyl6-clarify-tentative-first
**Generated**: 2026-04-16
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Taxonomy Scan Before Bulk Confirm: Step 1.5 is taxonomy scan, Step 2 is bulk confirm in `fab-clarify.md`
- [x] CHK-002 Question Budget Spans Both Phases: 5-question cap text applies across tentative + remaining taxonomy questions; bulk confirm excluded
- [x] CHK-003 Bulk Confirm Context Note Reversal: Step 2 note states it runs on artifact updated by Step 1.5
- [x] CHK-004 Auto Mode Unaffected: Auto mode section has no changes to behavior
- [x] CHK-005 Preamble Updated: `_preamble.md` Bulk Confirm text reflects Step 2 ordering
- [x] CHK-006 Spec Flow Diagram Updated: `SPEC-fab-clarify.md` shows taxonomy scan at Step 1.5, bulk confirm at Step 2
- [x] CHK-007 Memory File Updated: `clarify.md` Bulk Confirm section text reflects new ordering

## Behavioral Correctness
- [x] CHK-008 Tentative markers addressed before bulk confirm: Step ordering in `fab-clarify.md` presents `<!-- assumed: -->` questions before bulk confirm flow
- [x] CHK-009 Bulk confirm sees updated counts: Detection conditions evaluated after Step 1.5 resolution

## Scenario Coverage
- [x] CHK-010 Zero tentative scenario: When no `<!-- assumed: -->` markers exist, bulk confirm triggers immediately after empty scan
- [x] CHK-011 Question budget consumed by tentative: Tentative questions count toward 5-question cap

## Code Quality
- [x] CHK-012 Pattern consistency: Step numbering follows existing pattern (Step 1, Step 1.5, Step 2, Steps 3-8)
- [x] CHK-013 No unnecessary duplication: No duplicated step descriptions across files
- [x] CHK-014 Cross-references: All four files use consistent step numbering and ordering language

## Documentation Accuracy
- [x] CHK-015 No stale references: No remaining text referencing "Step 1.5: Bulk Confirm" or "Step 2: Taxonomy Scan" in old ordering
- [x] CHK-016 Changelog entries: Memory file changelog updated with this change

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
