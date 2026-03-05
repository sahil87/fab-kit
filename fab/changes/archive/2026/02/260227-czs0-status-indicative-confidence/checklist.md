# Quality Checklist: Status Indicative Confidence

**Change**: 260227-czs0-status-indicative-confidence
**Generated**: 2026-02-27
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Intake --check-gate count emission: `calc-score.sh --check-gate --stage intake` output includes `certain`, `confident`, `tentative`, `unresolved` fields
- [x] CHK-002 Spec --check-gate count emission: `calc-score.sh --check-gate` (spec branch) output includes count fields from `.status.yaml`
- [x] CHK-003 Indicative confidence display: `/fab-status` at intake stage shows `Indicative confidence: {score} (fab-ff gate: {threshold}) — {total} assumptions (...)`
- [x] CHK-004 Persisted confidence display: `/fab-status` at spec+ stage shows `Confidence: {score} of 5.0 (...)`
- [x] CHK-005 Fallback display: `/fab-status` at non-intake stage with no confidence data shows `Confidence: not yet scored`
- [x] CHK-006 Skill file sync: `fab-status.md` and `SKILL.md` have identical confidence display behavior

## Behavioral Correctness
- [x] CHK-007 No side effects in --check-gate mode: `.status.yaml` not modified when running `calc-score.sh --check-gate`
- [x] CHK-008 Unresolved suffix: `, {N} unresolved` appended only when unresolved > 0

## Scenario Coverage
- [x] CHK-009 Intake gate with all-certain assumptions: score computed correctly, counts emitted
- [x] CHK-010 Intake gate with unresolved: score is 0.0, gate fails, unresolved count emitted
- [x] CHK-011 Spec gate with persisted counts: counts read from .status.yaml and emitted

## Edge Cases & Error Handling
- [x] CHK-012 Missing confidence block at non-intake stage: falls back to "not yet scored"

## Code Quality
- [x] CHK-013 Pattern consistency: New code follows naming and structural patterns of surrounding code in calc-score.sh
- [x] CHK-014 No unnecessary duplication: Existing variables reused (local_certain etc. already computed in intake branch)

## Documentation Accuracy
- [x] CHK-015 fab-status.md: Confidence display section accurately describes the three-case logic
- [x] CHK-016 SKILL.md: Mirrors fab-status.md confidence display behavior exactly

## Cross References
- [x] CHK-017 Intake references calc-score.sh output format: intake.md "What Changes" section matches actual output

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
