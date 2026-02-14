# Tasks: Add dev folder and tests for _calc-score.sh

**Change**: 260214-mgh5-calc-score-dev-setup
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Create `src/calc-score/` directory and symlink `_calc-score.sh → ../../fab/.kit/scripts/_calc-score.sh`

## Phase 2: Core Implementation

- [x] T002 [P] Create `src/calc-score/README.md` documenting API, usage, testing, and changelog
- [x] T003 [P] Create `src/calc-score/test-simple.sh` — quick smoke test with temp fixtures
- [x] T004 [P] Create `src/calc-score/test.sh` — comprehensive test suite covering grade counting, score formula, carry-forward, status updates, delta computation, and error cases

## Phase 3: Verification

- [x] T005 Run `src/calc-score/test-simple.sh` and `src/calc-score/test.sh` — verify all tests pass

---

## Execution Order

- T001 blocks T002, T003, T004 (directory must exist first)
- T002, T003, T004 are independent of each other
- T005 requires T001-T004 complete
