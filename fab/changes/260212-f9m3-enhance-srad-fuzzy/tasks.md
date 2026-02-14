# Tasks: Enhance SRAD Confidence Scoring

**Change**: 260212-f9m3-enhance-srad-fuzzy
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Add fuzzy scoring mode, weighted composite helpers, and backward-compatible defaults in `fab/.kit/scripts/lib/calc-score.sh`.
- [x] T002 Add change-type threshold mapping and legacy fallback helpers for `/fab-fff` gate consumers in `fab/.kit/scripts/lib/calc-score.sh`.
- [x] T003 [P] Extend calculator usage and behavior documentation in `src/lib/calc-score/README.md`.

## Phase 2: Core Implementation

- [x] T004 Update SRAD specification to document 0-100 dimension scoring, weighted composite, fuzzy memberships, calibrated penalties, and dynamic thresholds in `docs/specs/srad.md`.
- [x] T005 Update skills reference behavior for fuzzy SRAD scoring and dynamic `/fab-fff` thresholding in `docs/specs/skills.md`.
- [x] T006 Update runtime skill context for fuzzy SRAD scoring model, threshold table, and legacy/fuzzy mode semantics in `fab/.kit/skills/_context.md`.
- [x] T007 Update `/fab-fff` operational gate instructions to consume type-aware thresholds (with legacy fallback) in `fab/.kit/skills/fab-fff.md`.

## Phase 3: Integration & Edge Cases

- [x] T008 Expand coverage for fuzzy score computation, weighted composite behavior, dynamic threshold mapping, and legacy compatibility in `src/lib/calc-score/test.sh`.
- [x] T009 Run scoped validation for calculator behavior using `src/lib/calc-score/test.sh` and fix any regressions.

## Phase 4: Polish

- [x] T010 Perform consistency pass across changed docs and scripts (`docs/specs/srad.md`, `docs/specs/skills.md`, `fab/.kit/skills/_context.md`, `fab/.kit/skills/fab-fff.md`, `src/lib/calc-score/README.md`) to ensure terminology and formulas match exactly.

---

## Execution Order

- T001 blocks T002 and T008 (core scoring helpers and outputs must exist first).
- T004, T005, T006, and T007 depend on the final behavior from T001/T002.
- T008 blocks T009 (tests must exist before execution).
- T010 runs last as the final consistency sweep.
