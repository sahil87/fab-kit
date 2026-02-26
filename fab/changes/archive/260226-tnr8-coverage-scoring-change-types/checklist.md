# Quality Checklist: Coverage-Weighted Confidence Scoring & Formalize Change Types

**Change**: 260226-tnr8-coverage-scoring-change-types
**Generated**: 2026-02-26
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Coverage-weighted formula: `calc-score.sh` computes `score = base * cover` with `cover = min(1.0, total / expected_min)`
- [x] CHK-002 Expected_min thresholds: embedded lookup table in `calc-score.sh` with correct values for all 7 types at both stages
- [x] CHK-003 Stage flag: `calc-score.sh` accepts `--stage <stage>` defaulting to `spec`
- [x] CHK-004 Read change_type: `calc-score.sh` reads `change_type` from `.status.yaml`, defaults to `feat`
- [x] CHK-005 Seven canonical types: all references use `feat`/`fix`/`refactor`/`docs`/`test`/`ci`/`chore`
- [x] CHK-006 Type inference in fab-new: keyword heuristic added, first-match-wins, writes via `stageman.sh set-change-type`
- [x] CHK-007 Indicative confidence in fab-new: display-only, uses coverage-weighted formula, not persisted
- [x] CHK-008 git-pr reads .status.yaml: resolution chain has new step 2 reading `change_type`
- [x] CHK-009 Gate thresholds: `--check-gate` uses 7-type taxonomy with correct thresholds
- [x] CHK-010 set-change-type subcommand: `stageman.sh set-change-type` validates type and writes atomically
- [x] CHK-011 Template default: `status.yaml` template uses `change_type: feat`
- [x] CHK-012 change-types.md spec: new document created with all required content sections

## Behavioral Correctness

- [x] CHK-013 Existing well-covered specs: score unchanged when `total_decisions >= expected_min`
- [x] CHK-014 Thin specs: score attenuated when `total_decisions < expected_min`
- [x] CHK-015 Old type values: `feature`/`bugfix`/`architecture` fall through to `feat` default threshold

## Removal Verification

- [x] CHK-016 Old 4-type gate mapping: `bugfix`/`feature`/`refactor`/`architecture` no longer referenced in `calc-score.sh`
- [x] CHK-017 Old default: `change_type: feature` no longer in template

## Scenario Coverage

- [x] CHK-018 Thin spec scenario: 2 Certain decisions + feat type → score ~1.7 (not 5.0)
- [x] CHK-019 Well-covered scenario: 10 decisions + feat type → score = base * 1.0
- [x] CHK-020 Unknown type fallback: null/missing change_type → `feat` thresholds
- [x] CHK-021 Invalid type rejection: `set-change-type architecture` → error

## Edge Cases & Error Handling

- [x] CHK-022 Missing change_type field: `calc-score.sh` defaults to `feat` without error
- [x] CHK-023 Zero decisions: `total_decisions = 0` → `cover = 0.0` → `score = 0.0`
- [x] CHK-024 Gate check with old type values: `feature` in .status.yaml → defaults to `feat` threshold

## Code Quality

- [x] CHK-025 Pattern consistency: new code follows existing `stageman.sh` patterns (validation, atomic writes, help text)
- [x] CHK-026 No unnecessary duplication: `expected_min` table defined once in `calc-score.sh`, not duplicated

## Documentation Accuracy

- [x] CHK-027 _preamble.md: confidence scoring section uses 7-type taxonomy and coverage-weighted formula
- [x] CHK-028 srad.md: gate threshold section uses 7-type taxonomy, formula updated
- [x] CHK-029 specs/index.md: includes change-types entry

## Cross References

- [x] CHK-030 All 7 types consistent across: calc-score.sh, stageman.sh, fab-new.md, git-pr.md, _preamble.md, srad.md, change-types.md
- [x] CHK-031 Gate threshold values consistent across: calc-score.sh, _preamble.md, srad.md, change-types.md

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
