# Quality Checklist: Document the SRAD Framework

**Change**: 260210-k7p2-document-srad-framework
**Generated**: 2026-02-10
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 SRAD spec existence: `fab/specs/srad.md` exists and is self-contained (readable without `_context.md`)
- [x] CHK-002 Acronym expansion: All four dimensions defined with one-sentence descriptions (S, R, A, D)
- [x] CHK-003 Dimension evaluation criteria: High vs low table present for each dimension, matching `_context.md`
- [x] CHK-004 Confidence grades: All four grades (Certain, Confident, Tentative, Unresolved) documented with meaning, marker, and visibility
- [x] CHK-005 Confidence scoring formula: Formula present, penalty weights explained (Certain=0, Confident=0.1, Tentative=1.0, Unresolved=hard zero)
- [x] CHK-006 Gate threshold: `/fab-fff` >= 3.0 requirement documented with practical implications
- [x] CHK-007 Confidence lifecycle: Table present showing compute/recompute/consume per skill, including `/fab-discuss`
- [x] CHK-008 Critical Rule: Documented that Unresolved + low R + low A must always be asked; `/fab-clarify` is not an escape valve
- [x] CHK-009 Worked examples: Two proposal-level examples present (high-ambiguity near 0.0, low-ambiguity near 5.0) with SRAD evaluations, counts, and scores
- [x] CHK-010 Skill-specific autonomy levels: Table covering posture, interruption budget, output format, escape valve per skill
- [x] CHK-011 Specs index entry: `fab/specs/index.md` contains row for `[srad](srad.md)` with appropriate description

## Behavioral Correctness
- [x] CHK-012 Formula accuracy: Formula in `srad.md` matches the formula in `_context.md` exactly
- [x] CHK-013 Dimension criteria accuracy: High/low criteria in `srad.md` match `_context.md` definitions
- [x] CHK-014 Grade definitions accuracy: Grade meanings and markers in `srad.md` match `_context.md`

## Scenario Coverage
- [x] CHK-015 Reader discovers SRAD via index: Verified link from `fab/specs/index.md` to `srad.md` works
- [x] CHK-016 Computing a score: Example calculation in worked examples matches formula output (Example 2: 8C, 2Co, 0T, 0U → 4.8)
- [x] CHK-017 Unresolved hard zero: Example 1 shows score = 0.0 when Unresolved exists
- [x] CHK-018 Floor clamping: Range section explains clamping with 6T example (5.0 - 6.0 = -1.0 → 0.0)
- [x] CHK-019 Gate blocks low-confidence: Example 1 shows score 0.0 < 3.0, gate blocks
- [x] CHK-020 Gate allows exact threshold: "What 3.0 Allows" section shows 2T/0Co = 3.0 passes
- [x] CHK-021 Gate allows high-confidence: Example 2 shows score 4.8 >= 3.0, gate passes

## Edge Cases & Error Handling
- [x] CHK-022 Score boundary: `max(0.0, ...)` floor clamping explained in Range section with concrete example
- [x] CHK-023 Unresolved override: Critical Rule section explains override of question budget

## Documentation Accuracy
- [x] CHK-024 No stale references: All cross-references to `_context.md`, `fab-fff`, skills, and `.status.yaml` are accurate
- [x] CHK-025 Consistent terminology: Terms in `srad.md` match glossary definitions (minor casing variance: glossary uses "Signal strength", srad.md uses "Signal Strength" — not contradictory)

## Cross References
- [x] CHK-026 Index consistency: `fab/specs/index.md` description matches actual content of `srad.md`
- [x] CHK-027 Glossary alignment: SRAD section in `fab/specs/glossary.md` does not contradict `srad.md`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
