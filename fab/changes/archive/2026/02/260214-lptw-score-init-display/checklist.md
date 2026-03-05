# Quality Checklist: Confidence score initial value and display format

**Change**: 260214-lptw-score-init-display
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Template initial score: `fab/.kit/templates/status.yaml` has `score: 0.0`
- [x] CHK-002 Calc-score fallback: `_calc-score.sh` uses `prev_score` fallback of `"0.0"` (both line 79 initial and line 84 parameter expansion)
- [x] CHK-003 Status display format: `fab-status.md` uses `{score} of 5.0` format
- [x] CHK-004 Fff gate display: `fab-fff.md` gate message (line 28) and output header (line 75) use `of 5.0` format
- [x] CHK-005 Context template description: `_context.md` describes template default as "score 0.0"

## Behavioral Correctness

- [x] CHK-006 New changes get `score: 0.0` instead of `5.0` in `.status.yaml` — template confirmed
- [x] CHK-007 First score computation delta is relative to 0.0 baseline — fallback is `"0.0"`

## Scenario Coverage

- [x] CHK-008 Scenario: new change creation — template produces `confidence.score: 0.0`
- [x] CHK-009 Scenario: `/fab-status` renders "{score} of 5.0" — instruction text confirmed
- [x] CHK-010 Scenario: `/fab-fff` gate failure shows "of 5.0" in message — line 28 confirmed
- [x] CHK-011 Scenario: `/fab-fff` gate pass header shows "of 5.0" — line 75 confirmed

## Edge Cases & Error Handling

- [x] CHK-012 Missing confidence block in `.status.yaml` still shows "not yet scored" — no change to that logic (fab-status.md line 48 still has the fallback)

## Documentation Accuracy

- [x] CHK-013 `_context.md` Template subsection matches actual template default — both say "score 0.0"
- [x] CHK-014 No stale "5.0" references remain in modified files — grep confirmed; remaining 5.0 refs are formula/display scale

## Cross References

- [x] CHK-015 All five modified files are internally consistent (same display format "of 5.0", same initial value 0.0)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
