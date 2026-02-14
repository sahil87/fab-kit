# Quality Checklist: Extract confidence scoring into standalone script

**Change**: 260213-w8p3-extract-fab-score
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Script Interface: `_fab-score.sh` accepts `$1` change dir, emits YAML stdout, exit 0/1
- [x] CHK-002 Assumptions Table Scanning: Scans `## Assumptions` in `brief.md` + `spec.md` only
- [x] CHK-003 Carry-Forward: Implicit Certain counts preserved from `.status.yaml`
- [x] CHK-004 Formula Application: Confidence formula applied correctly (0.3 confident, 1.0 tentative, hard zero on unresolved)
- [x] CHK-005 Status File Update: `confidence:` block in `.status.yaml` replaced via awk without corrupting other fields
- [x] CHK-006 Delta Output: stdout includes `delta` showing score change from previous
- [x] CHK-007 `/fab-new` scoring removed: No confidence computation in `fab-new.md`
- [x] CHK-008 `/fab-continue` spec-only scoring: `_fab-score.sh` invoked after spec generation only, not at other stages
- [x] CHK-009 `/fab-clarify` suggest-mode scoring: `_fab-score.sh` invoked in suggest mode when spec exists; skipped at brief stage and in auto mode

## Behavioral Correctness

- [x] CHK-010 Template defaults persist: New changes from `/fab-new` keep score 5.0 until spec stage
- [x] CHK-011 Reset flow scores: `/fab-continue spec` (reset) invokes `_fab-score.sh` after regeneration

## Removal Verification

- [x] CHK-012 No inline scoring in `fab-new.md`: grep for "confidence", "score", "SRAD grade" returns zero matches
- [x] CHK-013 No inline scoring in `fab-continue.md`: Step 3 and Step 4 references to "recompute confidence" removed
- [x] CHK-014 No inline scoring in `fab-clarify.md`: Step 7 replaced with script invocation
- [x] CHK-015 Lifecycle table removed from `_context.md`: No 5-row lifecycle table in Confidence Scoring section

## Scenario Coverage

- [x] CHK-016 Missing spec.md: Script exits 1 with "spec.md required for scoring"
- [x] CHK-017 Missing change directory: Script exits 1 with error
- [x] CHK-018 No Assumptions section in file: Treated as 0 counts (count_grades returns empty for missing file)
- [x] CHK-019 Case-insensitive grade matching: tr '[:upper:]' '[:lower:]' in grade parsing loop

## Edge Cases & Error Handling

- [x] CHK-020 Score clamped at zero: awk formula has `if (s < 0.0) s = 0.0`
- [x] CHK-021 No prior Certain count: certain=0 in template → 0 carry-forward (default fallback)
- [x] CHK-022 Auto-clarify does not score: `[AUTO-MODE]` invocations skip `_fab-score.sh` (per fab-clarify.md Step 7)

## Documentation Accuracy

- [x] CHK-023 `_context.md` updated: Lifecycle table → one-liner, Autonomy table row updated
- [x] CHK-024 `srad.md` updated: 3-row lifecycle table, autonomy table row updated
- [x] CHK-025 `planning-skills.md` updated: `/fab-new` paragraph removed, `/fab-continue` step 6 updated, `/fab-fff` note updated
- [x] CHK-026 `change-lifecycle.md` updated: confidence description references `_fab-score.sh`

## Cross References

- [x] CHK-027 All references to `fab-score.sh` use underscore prefix `_fab-score.sh` (brief.md retains original pre-rename text — historical artifact, not implementation)
- [x] CHK-028 Script path consistent: `fab/.kit/scripts/_fab-score.sh` everywhere

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
