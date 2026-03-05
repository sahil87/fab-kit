# Quality Checklist: Fix calc-score.sh Short-Form Path References

**Change**: 260218-hpzb-fix-calc-score-path-refs
**Generated**: 2026-02-18
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Full repo-root-relative paths: All 5+1 backtick-enclosed `calc-score.sh` references use `fab/.kit/scripts/lib/calc-score.sh`
- [x] CHK-002 Shell script unchanged: `fab/.kit/scripts/lib/calc-score.sh` has no modifications

## Behavioral Correctness
- [x] CHK-003 Path consistency: Short forms `calc-score.sh` and `lib/calc-score.sh` no longer appear in backticks within skill files

## Scenario Coverage
- [x] CHK-004 LLM invocation scenario: Invocation-style references (fab-ff.md:28) resolve correctly from repo root
- [x] CHK-005 Descriptive reference scenario: Descriptive references (fab-ff.md:14, _context.md:151) use full path

## Edge Cases & Error Handling
- [x] CHK-006 No regressions: Already-correct references (_context.md:279 first ref, fab-clarify.md:95, fab-continue.md:70) remain unchanged

## Code Quality
- [x] CHK-007 Pattern consistency: Path format matches the convention established by stageman fix (260217-eywl)

## Documentation Accuracy
- [x] CHK-008 Surrounding text: Lines containing replaced paths still read correctly in context

## Cross References
- [x] CHK-009 Internal consistency: No dangling or inconsistent `calc-score.sh` references across skill files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
