# Quality Checklist: Scoring Formula Produces Inflated Scores

**Change**: 260212-29xv-scoring-formula
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Increase Confident Penalty: `_context.md` formula reads `0.3 * confident`
- [x] CHK-002 Increase Confident Penalty: `srad.md` formula reads `0.3 * confident`
- [x] CHK-003 Increase Confident Penalty: Penalty weights table shows 0.3 for Confident in both `_context.md` and `srad.md`
- [x] CHK-004 Grade Reclassification: `fab-clarify.md` skill specifies Tentative → Certain on resolution
- [x] CHK-005 Grade Reclassification: `fab-clarify.md` skill specifies Confident → Certain on confirmation
- [x] CHK-006 Recount Mechanism: `fab-clarify.md` Step 7 specifies reading grades from Assumptions tables
- [x] CHK-007 Clarify Doc: `clarify.md` centralized doc describes grade reclassification in Confidence Recomputation section
- [x] CHK-008 Gate Threshold: Gate remains at 3.0 in all locations

## Behavioral Correctness

- [x] CHK-009 Formula Change: All-Certain change (11C, 0Co) still scores 5.0
- [x] CHK-010 Formula Change: Moderate-Confident change (3C, 4Co) scores 3.8 (not 4.6)
- [x] CHK-011 Formula Change: High-Confident change (8C, 9Co) scores 2.3 (not 4.1)
- [x] CHK-012 Gate Examples: "What 3.0 Allows" section in `srad.md` reflects 0.3 penalty math

## Removal Verification

- [x] CHK-013 Deprecated 0.1 Penalty: No remaining references to `0.1 * confident` in `_context.md`
- [x] CHK-014 Deprecated 0.1 Penalty: No remaining references to `0.1 * confident` in `srad.md`
- [x] CHK-015 Deprecated 0.1 Penalty: No remaining references to Confident penalty of 0.1 in `planning-skills.md`

## Scenario Coverage

- [x] CHK-016 Scenario: All-Certain change scores 5.0 per spec
- [x] CHK-017 Scenario: Mixed with Tentative (5C, 2Co, 1T) scores 3.4 per spec
- [x] CHK-018 Scenario: Gate passes at score 3.5 (5C, 5Co)
- [x] CHK-019 Scenario: Gate fails at score 2.6 (4C, 8Co)
- [x] CHK-020 Scenario: Score increases after clarify resolves Tentative assumptions

## Edge Cases & Error Handling

- [x] CHK-021 Zero Confident/Tentative: Score remains 5.0 when all decisions are Certain
- [x] CHK-022 High Confident: 7+ Confident decisions correctly fail gate (score < 3.0)
- [x] CHK-023 Clarify with no assumptions: Recomputation is a no-op when Assumptions table has no Tentative/Confident entries

## Documentation Accuracy

- [x] CHK-024 Formula consistency: All three locations (`_context.md`, `srad.md`, `planning-skills.md`) show the same formula or reference it consistently
- [x] CHK-025 Changelog entries: `clarify.md` and `planning-skills.md` have changelog entries. **N/A** for `srad.md`: design specs don't have changelog sections.

## Cross References

- [x] CHK-026 `_context.md` Confidence Scoring section is consistent with `srad.md` Formula section
- [x] CHK-027 `fab-clarify.md` skill Step 7 is consistent with `clarify.md` centralized doc Confidence Recomputation section

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
