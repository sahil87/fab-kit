# Tasks: Enhance SRAD Confidence Scoring with Fuzzy Dimensions

**Change**: 260212-f9m3-enhance-srad-fuzzy
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Add `change_type` field support to `.status.yaml` template at `fab/.kit/templates/status.yaml` — add `change_type: feature` as default field after `created_by`
- [x] T002 [P] Create `src/lib/calc-score/sensitivity.sh` scaffold with argument parsing, archive scanning, and output formatting (empty analysis functions)

## Phase 2: Core Implementation

- [x] T003 Extend `fab/.kit/scripts/lib/calc-score.sh` to detect and parse optional `Scores` column (`S:nn R:nn A:nn D:nn` format) from Assumptions tables in brief.md and spec.md — extract per-dimension scores alongside existing grade counting
- [x] T004 Add fuzzy dimension aggregation to `fab/.kit/scripts/lib/calc-score.sh` — compute mean S, R, A, D scores across all rows with Scores data, and output `fuzzy: true` + `dimensions:` block in YAML stdout
- [x] T005 Extend `set_confidence_block` in `fab/.kit/scripts/lib/stageman.sh` to accept optional fuzzy parameters and write `fuzzy:` flag + `dimensions:` sub-block to `.status.yaml`
- [x] T006 Implement `--check-gate` flag in `fab/.kit/scripts/lib/calc-score.sh` — read `change_type` from `.status.yaml`, look up per-type threshold (bugfix=2.0, feature/refactor=3.0, architecture=4.0), output `gate: pass/fail` with threshold info
- [x] T007 Implement Domain 1 (formula penalty weights) sensitivity analysis in `src/lib/calc-score/sensitivity.sh` — scan archived `.status.yaml` files, vary Confident penalty [0.1–0.5] and Tentative penalty [0.5–1.5], recompute scores, measure discrimination against review outcomes
- [x] T008 Implement Domain 2 (dimension aggregation weights) sensitivity analysis in `src/lib/calc-score/sensitivity.sh` — vary w_R [0.20–0.40] with others proportional, compute composite scores for spec worked examples, verify Critical Rule override, report sensitivity

## Phase 3: Integration & Edge Cases

- [x] T009 Extend `src/lib/calc-score/test.sh` with fuzzy dimension parsing tests — Scores column extraction, mean computation, mixed rows (some with Scores, some without), malformed `S:nn` input
- [x] T010 [P] Extend `src/lib/calc-score/test.sh` with `--check-gate` tests — each change type threshold, missing `change_type` defaults to 3.0, edge cases at exact boundary values
- [x] T011 [P] Extend `src/lib/calc-score/test.sh` with backward compatibility tests — legacy tables without Scores column produce identical results to current behavior (regression guard)
- [x] T012 [P] Extend `src/lib/calc-score/test.sh` with edge cases — empty Scores field, partial dimension data (e.g., only `S:50 R:60`), all-zero dimensions, all-100 dimensions, single-row tables
- [x] T013 Run full test suite `src/lib/calc-score/test.sh` and fix any failures

## Phase 4: Polish

- [x] T014 Update `docs/specs/srad.md` — add fuzzy dimension evaluation methodology, aggregation formula + default weights, grade threshold table, dynamic threshold table by change type, updated worked examples with numeric dimension scores
- [x] T015 [P] Update `fab/.kit/skills/_context.md` SRAD Scoring section — change evaluation criteria table from binary high/low to continuous 0–100, add aggregation formula reference, note fuzzy flag in `.status.yaml` schema

---

## Execution Order

- T001 is independent setup (template change)
- T002 is independent setup (script scaffold)
- T003 blocks T004 (dimension parsing before aggregation)
- T004 blocks T005 (aggregation output format before stageman integration)
- T005 blocks T006 (status writing before gate check reads it)
- T007 and T008 are independent of T003–T006 (analysis script is separate)
- T009 blocks T013 (tests must exist before running suite)
- T010, T011, T012 are parallelizable (independent test sections), all block T013
- T014 and T015 are parallelizable (independent doc files)
