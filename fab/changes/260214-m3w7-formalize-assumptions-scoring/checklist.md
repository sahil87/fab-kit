# Quality Checklist: Formalize Assumptions Tables & Fix Scoring Pipeline

**Change**: 260214-m3w7-formalize-assumptions-scoring
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 All Four Grades: `_context.md` Confidence Grades table includes updated Output Visibility for Certain and Unresolved
- [x] CHK-002 All Four Grades: `_context.md` Assumptions Summary Block rules specify all four grades (not just Confident/Tentative)
- [x] CHK-003 Required Scores Column: `_context.md` marks Scores column as mandatory (not optional)
- [x] CHK-004 Summary Line Format: `_context.md` uses `{N} assumptions ({Ce} certain, {Co} confident, {T} tentative, {U} unresolved)`
- [x] CHK-005 Unresolved Status Context: `_context.md` specifies `Asked — {outcome}` / `Deferred — {reason}` patterns
- [x] CHK-006 Brief Template: `fab/.kit/templates/brief.md` has `## Assumptions` with 5-column header and state-transfer HTML comment
- [x] CHK-007 Spec Template: `fab/.kit/templates/spec.md` has `## Assumptions` with 5-column header and sole-scoring-source HTML comment
- [x] CHK-008 calc-score.sh: `brief_file` and `parse_assumptions "$brief_file"` removed — only `spec.md` parsed
- [x] CHK-009 calc-score.sh: AWK uses `cols[6]` (not `cols[4]`) for Scores column
- [x] CHK-010 calc-score.sh: `has_scores` detection logic removed — always extracts scores
- [x] CHK-011 calc-score.sh: Unresolved grade counted (not hardcoded to 0)
- [x] CHK-012 calc-score.sh: Implicit Certain carry-forward block removed
- [x] CHK-013 fab-new.md: Assumptions step specifies all four grades with required Scores
- [x] CHK-014 _generation.md: Spec Procedure Step 6 specifies brief-as-starting-point (confirm/upgrade/override)
- [x] CHK-015 fab-ff.md: Cumulative summary tracks all four grades

## Behavioral Correctness

- [x] CHK-016 calc-score.sh: With spec containing 1 Unresolved row, score outputs `0.0`
- [x] CHK-017 calc-score.sh: With spec containing 5 Certain + 2 Confident, score is `4.4`
- [x] CHK-018 calc-score.sh: Dimension scores correctly parsed from actual Scores column (not Decision column)
- [x] CHK-019 calc-score.sh: `total_certain = table_certain` (no carry-forward inflation)

## Scenario Coverage

- [x] CHK-020 Score from spec only: brief.md with 10 assumptions + spec.md with 8 → counts 8 total
- [x] CHK-021 Missing brief.md: scoring succeeds from spec.md alone
- [x] CHK-022 Old-format spec (no Scores column): grade counting works, dimension aggregation skipped

## Edge Cases & Error Handling

- [x] CHK-023 Empty Assumptions table: calc-score.sh handles zero rows (all counts 0, score 5.0)
- [x] CHK-024 Transition: existing specs with 4-column tables don't crash calc-score.sh

## Documentation Accuracy

- [x] CHK-025 planning-skills.md: calc-score reference says "spec Assumptions table" not "brief + spec"
- [x] CHK-026 change-lifecycle.md: confidence field notes spec-only computation
- [x] CHK-027 planning-skills.md: SRAD/Assumptions references mention all four grades
- [x] CHK-028 Both memory files have changelog entries for this change

## Cross References

- [x] CHK-029 All files referencing "optional Scores column" or "has_scores" are updated
- [x] CHK-030 No remaining references to "brief + spec" scoring in any skill or memory file

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
