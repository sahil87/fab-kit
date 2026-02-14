# Quality Checklist: Add dev folder and tests for _calc-score.sh

**Change**: 260214-mgh5-calc-score-dev-setup
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Directory layout: `src/calc-score/` contains exactly `_calc-score.sh`, `README.md`, `test-simple.sh`, `test.sh`
- [x] CHK-002 Symlink: `_calc-score.sh` points to `../../fab/.kit/scripts/_calc-score.sh` and resolves correctly
- [x] CHK-003 README: follows established pattern (title, sources, usage, API reference, requirements, testing, changelog)
- [x] CHK-004 test-simple.sh: executable, self-contained smoke test with temp fixtures and cleanup
- [x] CHK-005 test.sh: executable, comprehensive suite covering all 6 spec areas

## Behavioral Correctness
- [x] CHK-006 test-simple.sh passes with exit code 0
- [x] CHK-007 test.sh passes with exit code 0 — all assertions green

## Scenario Coverage
- [x] CHK-008 Grade counting: test exercises parsing Assumptions tables from brief.md and spec.md
- [x] CHK-009 Score formula: test verifies `max(0.0, 5.0 - 0.3*confident - 1.0*tentative)`
- [x] CHK-010 Carry-forward: test verifies implicit Certain counts from previous .status.yaml
- [x] CHK-011 Status update: test verifies confidence block replacement in .status.yaml
- [x] CHK-012 Delta computation: test verifies `+X.X`/`-X.X` format
- [x] CHK-013 Combined brief+spec: test verifies grades from both files are counted

## Edge Cases & Error Handling
- [x] CHK-014 Missing change directory: exits 1 with appropriate stderr
- [x] CHK-015 Missing spec.md: exits 1 with appropriate stderr
- [x] CHK-016 No arguments: exits 1 with usage message

## Documentation Accuracy
- [x] CHK-017 README documents all arguments, output format, exit codes, and side effects
- [x] CHK-018 README testing section shows correct commands

## Cross References
- [x] CHK-019 Pattern consistency: file structure matches `src/stageman/`, `src/resolve-change/`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
