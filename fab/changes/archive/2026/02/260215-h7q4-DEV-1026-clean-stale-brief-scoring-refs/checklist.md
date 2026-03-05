# Quality Checklist: Clean Stale Brief Scoring Refs

**Change**: 260215-h7q4-DEV-1026-clean-stale-brief-scoring-refs
**Generated**: 2026-02-15
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 README spec-only: `src/lib/calc-score/README.md` states directory MUST contain `spec.md` with `## Assumptions` table, no mention of `brief.md`
- [x] CHK-002 SRAD lifecycle: `docs/specs/srad.md` Confidence Lifecycle Computation row says `calc-score.sh` scans spec only
- [x] CHK-003 Test removal: `src/lib/calc-score/test.sh` contains no "Combined grades from brief and spec" test

## Behavioral Correctness

- [x] CHK-004 SRAD script name: Confidence Lifecycle table uses `calc-score.sh` (no underscore prefix `_calc-score.sh`)

## Removal Verification

- [x] CHK-005 No brief.md test fixtures: No test in `test.sh` creates a `brief.md` file
- [x] CHK-006 No brief.md grade assertions: No test asserts grade counting from `brief.md`

## Scenario Coverage

- [x] CHK-007 README readthrough: Reading the README gives no impression that brief.md is scanned
- [x] CHK-008 Test suite passes: 26 pre-existing failures in carry-forward/fuzzy tests — no regressions from this change

## Documentation Accuracy

- [x] CHK-009 README description line: Opening description references `spec.md` only
- [x] CHK-010 Recomputation row: SRAD spec Recomputation row also uses `calc-score.sh` (not `_calc-score.sh`)

## Cross References

- [x] CHK-011 Memory consistency: `docs/memory/fab-workflow/change-lifecycle.md:59` correctly documents spec-only scoring (unchanged)

## Code Quality

- [x] CHK-012 Pattern consistency: Edits follow surrounding markdown/script style
- [x] CHK-013 No unnecessary duplication: No redundant content introduced
