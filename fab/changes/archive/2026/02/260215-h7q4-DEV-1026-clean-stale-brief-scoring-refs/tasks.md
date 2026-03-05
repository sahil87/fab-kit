# Tasks: Clean Stale Brief Scoring Refs

**Change**: 260215-h7q4-DEV-1026-clean-stale-brief-scoring-refs
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Fixes

- [x] T001 [P] Fix `src/lib/calc-score/README.md` — replace line 3 (description mentioning brief.md) and line 18 (brief.md optional claim) with spec-only language
- [x] T002 [P] Fix `docs/specs/srad.md` — update Confidence Lifecycle table line 205: change `_calc-score.sh` to `calc-score.sh` and `intake + spec` to `spec`
- [x] T003 [P] Remove stale "Combined grades from brief and spec" test (lines 143–168) from `src/lib/calc-score/test.sh`

## Phase 2: Verification

- [x] T004 Run `src/lib/calc-score/test.sh` and confirm all remaining tests pass (26 pre-existing failures in carry-forward/fuzzy tests — no regressions from this change)

---

## Execution Order

- T001, T002, T003 are independent — all [P]
- T004 depends on T003 (test removal must happen before running suite)
