# Plan: SRAD Resolution-Average Confidence Scoring

**Change**: 260615-tf5q-srad-resolution-average-scoring
**Intake**: `intake.md`

## Requirements

### Scoring: Resolution-Average Formula

#### R1: Per-row composite mean replaces grade-count penalty
`computeScore` SHALL compute the confidence score from the per-row S:R:A:D composite mean rather than from grade-count penalties. The per-row composite SHALL use the existing `_srad.md`/`srad.md` aggregation weights `0.25*S + 0.30*R + 0.25*A + 0.20*D`. The score SHALL be `round1( (mean_composite / 20.0) * cover )`, where `mean_composite` is the average composite over rows that have parseable dimensions (`DimCount`), and `cover` is the unchanged coverage term.

- **GIVEN** an intake whose Assumptions rows have parseable `Scores` (S:R:A:D) and no hard-fail rows
- **WHEN** `fab score` runs
- **THEN** the score is `round1( (mean(0.25*S+0.30*R+0.25*A+0.20*D) over DimCount rows) / 20.0 * cover )`
- **AND** the gate passes iff score >= 3.0 (a mean composite >= 60 at full coverage)

#### R2: Per-row Critical Rule hard-fail
`computeScore` SHALL return `0.0` when any Assumptions row has `R < 25 AND A < 25` on its raw per-row dimensions, evaluated before the mean is taken.

- **GIVEN** an intake with at least one row whose raw dimensions satisfy `R < 25 AND A < 25`
- **WHEN** `fab score` runs
- **THEN** the score is `0.0` regardless of the other rows' composites

#### R3: Unresolved hard-fail preserved (per-row + count)
`computeScore` SHALL return `0.0` when any Unresolved row exists. The existing `unresolved > 0 → 0.0` short-circuit semantics SHALL be preserved.

- **GIVEN** an intake with at least one Unresolved-graded row
- **WHEN** `fab score` runs
- **THEN** the score is `0.0`

#### R4: Coverage term and 0–5 scale preserved verbatim
The coverage term `cover = min(1.0, total_decisions / expectedMin)` SHALL be preserved unchanged, where `total_decisions = certain + confident + tentative + unresolved` (ALL graded rows, not only `DimCount` rows). The 0–5 scale, `roundTo1`, and `getGateThreshold` (flat 3.0) SHALL be unchanged.

- **GIVEN** an intake with fewer graded rows than `expectedMin` for its change type
- **WHEN** `fab score` runs
- **THEN** `cover < 1.0` attenuates the score proportionally, counting ALL graded rows in `total`

#### R5: Single parse pass — thread composite/flags out of countGrades
The per-row composite sum, `DimCount`, and the Critical-Rule / Unresolved hard-fail flags SHALL be produced by the existing single `countGrades` parse pass (by extending `GradeCount`) rather than by a second scan. `computeScore`'s internal signature MAY change to consume the `gc GradeCount`; the public `fab score` CLI signature SHALL NOT change.

- **GIVEN** the existing `countGrades` already parses S:R:A:D per row via `scoresRegex`
- **WHEN** the new formula needs per-row composites and hard-fail flags
- **THEN** they are accumulated inside `countGrades` and read from `GradeCount`, with no re-scan

#### R6: Remove dead penalty-weight constants
The `wCertain` / `wConfident` / `wTentative` const block SHALL be removed once the penalty arithmetic is gone.

- **GIVEN** the penalty arithmetic is replaced by the composite mean
- **WHEN** the rewrite is complete
- **THEN** `wCertain`/`wConfident`/`wTentative` are no longer declared or referenced

#### R7: Lower `fix` expectedMin 5→3
`expectedMin["fix"]` SHALL be lowered from `5` to `3`.

- **GIVEN** the `expectedMin` map in `score.go`
- **WHEN** the change is applied
- **THEN** `expectedMin["fix"] == 3` and `getExpectedMin("fix") == 3`

### Documentation: Spec and Skill-Source Conformance

#### R8: srad.md § Confidence Scoring rewritten
`docs/specs/srad.md` § Confidence Scoring SHALL restate the Resolution-Average formula, DROP the Penalty-Weights table, restate "What 3.0 Allows" in mean-composite terms (3.0 on 0–5 == mean composite >= 60, the Confident floor), and update Worked Example 2's arithmetic. The 0–5 scale, § Range, § Storage, § Gate Threshold table, § Coverage Factor, and § Critical Rule SHALL be kept.

- **GIVEN** the spec documents the old `base*cover` penalty formula
- **WHEN** the rewrite is complete
- **THEN** the spec describes the Resolution-Average formula self-consistently with no penalty references

#### R9: change-types.md expected_min table mirrors the map
`docs/specs/change-types.md` `expected_min` table SHALL show `fix` = `3`, mirroring the code map; `TestDocTablesMatchScoringMaps` SHALL stay green.

- **GIVEN** the doc table mirrors the `expectedMin` map
- **WHEN** the map's `fix` value changes to 3
- **THEN** the doc table shows `| fix | 3 |` and the drift test passes

#### R10: _cli-fab.md § fab score (extended) updated (constitution CLI doc rule)
`src/kit/skills/_cli-fab.md` § fab score (extended) SHALL update the formula pseudocode to Resolution-Average and the embedded `expected_min` text to `fix:3`. The canonical source SHALL be edited; deployed copies in `.claude/skills/` SHALL NOT be edited.

- **GIVEN** the constitution requires `_cli-fab.md` to document changed CLI behavior
- **WHEN** the formula changes
- **THEN** the canonical `_cli-fab.md` documents the new formula and `fix:3`

### Tests

#### R11: Tests conform to the new formula (Test Integrity)
`score_test.go`, `cmd/fab/score_test.go`, and `changetypes_doc_test.go` SHALL be updated to assert the Resolution-Average behavior with hand-computed expected values, including per-row Critical-Rule and Unresolved hard-fails, the `mean/20*cover` rescale, dimensionless-row handling, and `fix=3`. No implementation code SHALL be bent to fit a stale fixture.

- **GIVEN** existing tests assert the old penalty arithmetic and `fix=5`
- **WHEN** the formula and map change
- **THEN** the tests assert the new formula's correct expected values and pass

### Design Decisions

1. **Coverage `total` counts ALL graded rows**: the mean restricts to `DimCount` rows, but `cover`'s `total` counts `certain+confident+tentative+unresolved` — *Why*: the intake's recommended decide-at-apply default; matches today's `countGrades` total semantics; a dimensionless row is a malformed-intake edge case (the `Scores` column is REQUIRED). *Rejected*: counting only `DimCount` rows in `total` (would double-punish a malformed row and diverge from established coverage semantics).
2. **Critical-Rule + Unresolved evaluated inside `countGrades`**: per-row hard-fail flags are set during the single parse pass and read from `GradeCount` — *Why*: `countGrades` is already the single parse pass holding per-row S:R:A:D; avoids a second scan (code-quality: reuse, no duplication). *Rejected*: re-scanning the table in `computeScore` (duplicate parse).
3. **Glossary formula entry also updated**: `docs/specs/glossary.md`'s "Confidence score" definition restates the exact `base*cover` formula being removed — *Why*: leaving it would be a documented-accuracy inconsistency (config opts into `documentation_accuracy`). *Rejected*: leaving it stale (the intake's grep-for-old-formula step surfaced it).

### Non-Goals

- No change to the Assumptions table format, intake template, or skill behavior (the `Scores` column was already required and parsed).
- No migration — the score is recomputed, not persisted user-data shape.
- No change to the public `fab score` CLI signature, the gate threshold (3.0), or the SRAD grade-mapping thresholds.

## Tasks

### Phase 2: Core Implementation

- [x] T001 Extend `GradeCount` in `src/go/fab/internal/score/score.go` with a per-row composite accumulator (`SumComposite float64`) and hard-fail flags (`CriticalRowSeen bool`); accumulate them inside `countGrades` during the existing single parse pass <!-- R5 -->
- [x] T002 Rewrite `computeScore` in `src/go/fab/internal/score/score.go` to consume `gc GradeCount`: return 0.0 on Unresolved>0 or CriticalRowSeen; else `round1( (SumComposite/DimCount / 20.0) * cover )` with `cover = min(1.0, total/expectedMin)` over ALL graded rows; update both call sites (`CheckGate`, `buildResult`) to pass `gc` and `total` <!-- R1 R2 R3 R4 -->
- [x] T002b Remove the dead `wCertain`/`wConfident`/`wTentative` const block in `src/go/fab/internal/score/score.go` <!-- R6 -->
- [x] T003 Lower `expectedMin["fix"]` from 5 to 3 in `src/go/fab/internal/score/score.go` <!-- R7 -->

### Phase 3: Tests

- [x] T004 Update `src/go/fab/internal/score/score_test.go`: replace penalty-arithmetic assertions with Resolution-Average; add per-row Critical-Rule hard-fail, per-row Unresolved hard-fail, `mean/20*cover` rescale, dimensionless-row, and the rescued-strong-intake cases; update `TestConstants` (drop penalty consts, `fix` expectedMin 5→3) <!-- R11 -->
- [x] T005 [P] Update `src/go/fab/cmd/fab/score_test.go`: replace old `fix=5`/penalty end-to-end values with Resolution-Average values <!-- R11 -->
- [x] T006 [P] Verify `src/go/fab/internal/score/changetypes_doc_test.go` (`TestDocTablesMatchScoringMaps`) stays green after the map+doc `fix` 5→3 change; no separate hard-coded `fix:5` <!-- R11 R9 -->
- [x] T006b Update `src/go/fab/cmd/fab/hook_test.go` `TestHookArtifactWrite_IntakeSingleLoadAndSave`: its dimensionless fixture now scores 0.0 under the new formula — add perfect dimensions (S:100 R:100 A:100 D:100) so the 5.0 assertion stays meaningful (Test Integrity; not in the intake's explicit list but a consequence of the formula change) <!-- R11 -->

### Phase 4: Docs & Skill Source

- [x] T007 [P] Rewrite `docs/specs/srad.md` § Confidence Scoring: new formula, drop Penalty-Weights table, restate "What 3.0 Allows" in mean-composite terms, update Worked Example 2 arithmetic; keep 0–5 scale / Range / Storage / Gate table / Coverage / Critical Rule <!-- R8 -->
- [x] T008 [P] Update `docs/specs/change-types.md` expected_min table `fix` 5→3 <!-- R9 -->
- [x] T009 [P] Update `docs/specs/glossary.md` "Confidence score" definition to the Resolution-Average formula <!-- R8 -->
- [x] T010 [P] Update canonical `src/kit/skills/_cli-fab.md` § fab score (extended): formula pseudocode → Resolution-Average, embedded `expected_min` text `fix:5`→`fix:3` (edit canonical source only) <!-- R10 -->

## Execution Order

- T001 blocks T002 (computeScore reads the new GradeCount fields)
- T002, T002b, T003 are in the same file; apply sequentially
- T004–T006 depend on T001–T003 (tests assert the new behavior)
- T007–T010 are independent docs/skill edits ([P])

## Acceptance

### Functional Completeness

- [x] A-001 R1: `computeScore` returns `round1( (mean per-row composite / 20.0) * cover )` over `DimCount` rows using weights `0.25*S+0.30*R+0.25*A+0.20*D`
- [x] A-002 R5: per-row composite sum, `DimCount`, and hard-fail flags are produced by the single `countGrades` pass; no second scan; public CLI signature unchanged
- [x] A-003 R6: `wCertain`/`wConfident`/`wTentative` are removed and unreferenced
- [x] A-004 R7: `expectedMin["fix"] == 3` and `getExpectedMin("fix") == 3`
- [x] A-005 R8: `srad.md` § Confidence Scoring states the Resolution-Average formula; Penalty-Weights table dropped; Worked Example 2 recomputed
- [x] A-006 R9: `change-types.md` expected_min table shows `fix` = 3
- [x] A-007 R10: canonical `_cli-fab.md` documents the new formula and `fix:3`; deployed copies untouched

### Behavioral Correctness

- [x] A-008 R2: an intake with any row `R<25 AND A<25` scores `0.0`
- [x] A-009 R3: an intake with any Unresolved row scores `0.0`
- [x] A-010 R4: `cover` counts ALL graded rows in `total`; the mean restricts to `DimCount` rows; 3.0 gate preserved

### Edge Cases & Error Handling

- [x] A-011 R4: a dimensionless row (no parseable `Scores`) is excluded from the mean but still counted in coverage `total`

### Removal Verification

- [x] A-012 R6: no penalty arithmetic or penalty constants remain anywhere in `score.go`

### Code Quality

- [x] A-013 Pattern consistency: new code follows naming and structural patterns of surrounding `score.go` code
- [x] A-014 No unnecessary duplication: per-row dimensions are reused from `countGrades`, not re-parsed (anti-pattern: duplicating existing utilities)
- [x] A-015 No magic numbers: the `0.25/0.30/0.25/0.20` weights and `/20.0` rescale are documented (named or commented) per the magic-numbers anti-pattern

### Documentation Accuracy (config: documentation_accuracy)

- [x] A-016 R8: no doc, spec, or skill file references the removed `5.0 - 0.3*confident - 1.0*tentative` penalty formula

### Cross-References (config: cross_references)

- [x] A-017 R9: the `change-types.md` table and the `expectedMin` map agree (`TestDocTablesMatchScoringMaps` passes)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Deletion Candidates

None — this change adds new functionality without making existing code redundant.

<!-- Review note: the `wCertain`/`wConfident`/`wTentative` penalty-weight const block (R6/T002b) is already removed in this diff — already-deleted dead code, not a remaining candidate. The Resolution-Average rewrite is a swap (old penalty arithmetic → composite mean), not an addition: it reuses the existing single `countGrades` parse pass via extended `GradeCount` fields (no second scan, no duplicated parse), and both call sites (`CheckGate`, `buildResult`) already held the `gc`/`total` values now threaded in. No surrounding code (`SumS/R/A/D`, `MeanS/R/A/D`, `roundTo1`, `getExpectedMin`, `getGateThreshold`, `scoresRegex`) became redundant. -->

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Coverage `total` counts ALL graded rows (certain+confident+tentative+unresolved); only the mean restricts to `DimCount` rows | Intake's explicit decide-at-apply default (What-Changes §4, Open Questions); matches today's `countGrades` total semantics; dimensionless row is a malformed-intake edge case (Scores column REQUIRED) | S:80 R:72 A:82 D:78 |
| 2 | Confident | Extend `GradeCount` with `SumComposite float64` + `CriticalRowSeen bool` and accumulate both inside `countGrades`; `computeScore(gc, total, expectedMin)` consumes them | Intake permits this exact shape; `countGrades` is the single parse pass; both call sites already hold `gc`; code-quality reuse/no-duplication | S:82 R:70 A:84 D:75 |
| 3 | Confident | Reuse the per-row Unresolved hard-fail via the existing `gc.Unresolved>0` count (no separate per-row Unresolved flag needed) since the count already captures it | Unresolved is a grade-count, parsed per row already; a count>0 short-circuit is equivalent to a per-row flag and avoids a redundant field | S:85 R:80 A:85 D:80 |
| 4 | Confident | Also update `docs/specs/glossary.md` "Confidence score" definition (restates the exact removed formula) — not in the intake's explicit file list but a documentation-accuracy consequence surfaced by the old-formula grep | config opts into `documentation_accuracy`; leaving it stale would be a doc inconsistency; the intake's §Impact grep step is the trigger | S:80 R:88 A:85 D:85 |
| 5 | Confident | Name the composite weights and `/20.0` rescale with documenting comments/consts in `score.go` to satisfy the magic-numbers anti-pattern while matching the existing `score.go` style (which uses inline numeric weights with explanatory comments, e.g. gateThresholds) | code-quality anti-pattern (magic numbers); existing `score.go` favors commented inline literals over a proliferation of consts; keep consistency | S:78 R:85 A:80 D:72 |
| 6 | Confident | Also update `cmd/fab/hook_test.go` `TestHookArtifactWrite_IntakeSingleLoadAndSave` — its dimensionless fixture scored 5.0 under the old penalty formula but scores 0.0 under Resolution Average (DimCount=0); add perfect dimensions so the 5.0 assertion stays meaningful | Test Integrity (tests conform to spec); discovered at apply via the full test run; the artifact-write hook routes through the changed scoring path so its fixture must carry dimensions | S:85 R:85 A:88 D:82 |

6 assumptions (0 certain, 6 confident, 0 tentative).
