# Plan: SRAD v2 Demerit Confidence Scoring

**Change**: 260618-4yi8-srad-v2-demerit-scoring
**Intake**: `intake.md`

## Requirements

The authoritative design lives in `docs/specs/srad.md` (§ The Composite, § Grades,
§ Confidence Scoring, § No hard-fail rules, § The Critical Rule) and the decision record
`docs/specs/srad-scoring-rationale-v1-to-v2.md`. These requirements derive the implementation
surface from that design.

### Scoring: Composite & Penalty Formula

#### R1: Composite weights up-weight R and A
`fab score` SHALL compute each Assumptions row's composite as
`c = 0.20·S + 0.30·R + 0.30·A + 0.20·D` (weights summing to 1.0), replacing the v1
`0.25/0.30/0.25/0.20` weights.

- **GIVEN** a row with `S:100 R:0 A:0 D:100`
- **WHEN** `fab score` computes its composite
- **THEN** the composite is `40.0` (0.20·100 + 0.30·0 + 0.30·0 + 0.20·100)
- **AND** the four weight constants `wS,wR,wA,wD` sum to exactly `1.0`

#### R2: Per-row penalty is the piecewise demerit curve
`fab score` SHALL compute a per-row penalty via a `penalty(c float64) float64` helper:
`0` for `c ≥ 80`; `(80 − c)/30 · 0.50` for `50 ≤ c < 80`; `0.50 + (50 − c)/50 · 2.50` for
`c < 50`. The curve uses named constants (free-knee `80`, confident-floor penalty `0.50`,
aggressive slope coefficient `2.50`).

- **GIVEN** composites at the band boundaries
- **WHEN** `penalty(c)` is evaluated
- **THEN** `penalty(80) = 0.0`, `penalty(50) = 0.50` (both slopes meet), `penalty(20) = 2.0`,
  `penalty(0) = 3.0`
- **AND** the curve is continuous at both joins and per-row penalty ∈ `[0.0, 3.0]`

#### R3: Score is 5.0 minus the sum of penalties, clamped
`fab score` SHALL compute `score = clamp(5.0 − Σ penalty(c), 0.0, 5.0)`, summed over all rows
with parseable dimensions, replacing the v1 `(meanComposite / 20) · cover`. The
`compositeToScore` rescale constant SHALL be retired.

- **GIVEN** an intake whose rows' penalties sum to `2.0`
- **WHEN** `fab score` computes the score
- **THEN** the score is exactly `3.0`
- **AND** an intake whose penalties sum to ≥ `5.0` clamps to `0.0`; an all-Certain intake
  (penalties `0.0`) scores `5.0`

### Scoring: Removal of Hard-Fail Rules

#### R4: No hard-fail short-circuits — blocking is emergent
`fab score` SHALL NOT short-circuit to `0.0` on the presence of an Unresolved row, and SHALL
NOT apply the `R<25 ∧ A<25` Critical Rule. Blocking is emergent from the penalty curve: a row
at `composite < 20` penalizes ≥ `2.0`. The `CriticalRowSeen` flag and the `criticalDim`
constant SHALL be removed; the Unresolved grade count SHALL cease to gate the score (counts
remain surfaced for display).

- **GIVEN** a 7-row intake: 6 rows at `composite ≥ 80` plus one Unresolved row at `composite < 20`
- **WHEN** `fab score` computes the score
- **THEN** the result is the curve-derived value (≤ 3.0 driven by the weak row's ≥ 2.0 penalty),
  NOT a hard `0.0`
- **AND** a strong intake containing a single `R:20 A:20` row is no longer hard-failed by a
  Critical Rule — its penalty is purely curve-derived

#### R5: Drop coverage / expected_min from the score path
`fab score` SHALL NOT apply a coverage factor (`cover = min(1, rows/expected_min)`) to the
score. A thin-but-strong intake SHALL score on its rows' penalties alone. The `expectedMin`
map and `getExpectedMin` MAY remain in the package for the `change-types.md` doc-drift guard
(`changetypes_doc_test.go`) but SHALL NOT participate in `computeScore`.

- **GIVEN** a 2-row intake, both rows at `composite ≥ 80`, change_type `feat` (v1 expected_min 7)
- **WHEN** `fab score` computes the score
- **THEN** the score is `5.0` (no coverage attenuation)

### Scoring: Grade Derivation

#### R6: Grade label is derived from composite, not read from the Grade column
`fab score` SHALL derive each row's grade from its composite via the bands Certain ≥ 80,
Confident 50–80, Tentative 20–50, Unresolved < 20 — replacing the parse of the hand-written
`Grade` column for grade counting. The parser SHALL keep reading the `Scores` column for
dimensions.

- **GIVEN** a row whose hand-written Grade column says `Certain` but whose dimensions yield
  `composite = 45`
- **WHEN** `fab score` counts grades
- **THEN** the row is counted as Tentative (derived from composite), not Certain
- **AND** the `Scores` column remains the sole dimension input

### Documentation: Skill & Spec Conformance

#### R7: `_srad.md` reflects the v2 formula, bands, and Critical Rule
`src/kit/skills/_srad.md` SHALL state the v2 composite weights (`0.20/0.30/0.30/0.20`) and
bands (80/50/20), and SHALL rewrite the Critical Rule so a genuine unknown is surfaced and
blocks via `composite < 20` (NOT a hard-fail or `R<25 ∧ A<25` override). The corresponding
SPEC mirror `docs/specs/skills/SPEC-_srad.md` SHALL be updated to match (constitution: skill
changes update the SPEC mirror).

- **GIVEN** a reader consulting `_srad.md` for the scoring formula
- **WHEN** they read the SRAD Scoring and Critical Rule sections
- **THEN** the weights, bands, and emergent-blocking contract match `docs/specs/srad.md`
- **AND** `SPEC-_srad.md`'s summary and flow diagram are consistent

#### R8: `_cli-fab.md` `fab score` formula description is the demerit model
`src/kit/skills/_cli-fab.md` § fab score (extended) SHALL describe the demerit formula
(composite weights, penalty curve, `5.0 − Σ penalty`, no hard-fail, no coverage) replacing
the Resolution-Average description.

- **GIVEN** a reader consulting `_cli-fab.md` for the `fab score` formula
- **WHEN** they read the § Formula block
- **THEN** it states the v2 demerit model and no longer asserts coverage / Critical-Rule / mean

#### R9: Promptless-defer backstop wording is the v2 contract in all five files
`src/kit/skills/_intake.md`, `src/kit/skills/fab-proceed.md`, and the SPEC mirrors
`docs/specs/skills/SPEC-fab-proceed.md`, `docs/specs/skills/SPEC-_intake.md`,
`docs/specs/skills/SPEC-_srad.md` SHALL replace every "`fab score` returns 0.0 whenever any
Unresolved row exists" assertion with the v2 contract: a deferred/unresolved decision blocks
the gate only when its composite is below 20 (emergent from the curve, no special gate), and
the agent must score genuine unknowns with honestly-low dimensions so they land there.

- **GIVEN** any of the five files describing the promptless-defer backstop
- **WHEN** the backstop is read
- **THEN** it states "blocks only when composite < 20", not "returns 0.0 on any Unresolved row"
- **AND** the defer-and-surface MUST-ask contract is otherwise unchanged

### Non-Goals

- Reintroducing the v1 Resolution-Average scheme (`docs/specs/srad-v1.md` is the superseded record).
- Changing the flat 3.0 gate threshold (unchanged, all change types) or the `getGateThreshold` map.
- Changing the `## Assumptions` table format or the required `Scores` column.
- Editing `docs/specs/change-types.md` (the `expected_min` concept stays documented).
- Restructuring `.status.yaml` confidence schema or shipping a migration (computed-score change only).
- Editing the design specs `srad.md` / `srad-scoring-rationale-v1-to-v2.md` / `srad-v1.md`
  (authored in the design session; verify-only).
- Editing files under `.claude/skills/` (gitignored deployed copies — canonical source is `src/kit/skills/`).

### Design Decisions

1. **Keep `expectedMin` / `getExpectedMin` in the package** — *Why*: `changetypes_doc_test.go`'s
   `TestDocTablesMatchScoringMaps` asserts `getExpectedMin` mirrors `change-types.md`, and the intake
   says leave `change-types.md` intact. *Rejected*: removing them — would break the doc-drift guard and
   require deleting a doc table the intake says to keep. They simply stop feeding `computeScore`.
2. **`penalty(c)` as a standalone package-level helper** — *Why*: the curve is a pure function
   reused by `computeScore` and directly unit-testable per the spec's band/boundary cases. *Rejected*:
   inlining — would hide the curve from focused tests and bury the named constants.
3. **Grade derivation centralized in the parser (`countGrades`)** — *Why*: the parser already
   computes each row's composite; deriving the grade from that same composite keeps one source of truth
   and matches the spec ("the label can never contradict its own dimensions"). *Rejected*: a separate
   post-pass — redundant second classification.

### Deprecated Requirements

#### v1 Resolution-Average score
**Reason**: mean dilution hid the single dangerous decision; coverage punished thin-but-strong intakes
(see `docs/specs/srad-scoring-rationale-v1-to-v2.md`).
**Migration**: replaced by the demerit model (R1–R6). Existing `.status.yaml` confidence values
recompute on the next `fab score`; no data migration.

#### v1 hard-fail short-circuits (`Unresolved → 0.0`, `R<25 ∧ A<25` Critical Rule)
**Reason**: inconsistent (three grades cosmetic, one load-bearing); blocking now emergent from the curve.
**Migration**: removed; a `composite < 20` row penalizes ≥ 2.0 and blocks via the curve.

## Tasks

### Phase 1: Core Formula (score.go)

- [x] T001 Update composite weight constants in `src/go/fab/internal/score/score.go` to `wS=0.20 wR=0.30 wA=0.30 wD=0.20` and refresh their doc comment to the v2 aggregation; verify the sum stays 1.0 <!-- R1 -->
- [x] T002 Add a `penalty(c float64) float64` helper to `src/go/fab/internal/score/score.go` implementing the piecewise curve, with named constants `freeKnee = 80.0`, `confidentFloorPenalty = 0.50`, `aggressiveSlopeCoeff = 2.50`; retire the `compositeToScore` constant <!-- R2 -->
- [x] T003 Replace `computeScore` body in `src/go/fab/internal/score/score.go` with `clamp(5.0 − Σ penalty(composite), 0, 5)` over per-row composites; remove the `Unresolved>0 || CriticalRowSeen → 0.0` short-circuit and the `cover`/`expectedMin` arithmetic; drop the now-unused `total`/`expectedMin` parameters from its signature and update both call sites <!-- R3 -->
- [x] T004 Remove the `CriticalRowSeen` field and the `criticalDim` constant from `src/go/fab/internal/score/score.go`, and delete the per-row `R<criticalDim && A<criticalDim` tracking in `countGrades`; keep the per-row composite accumulation needed for grade derivation and scoring <!-- R4 -->
- [x] T005 Demote `expectedMin` / `getExpectedMin` in `src/go/fab/internal/score/score.go` to documentation-only: update their comments to note they no longer feed `computeScore` and exist solely for the `change-types.md` doc-drift guard <!-- R5 -->
- [x] T006 Change `countGrades` in `src/go/fab/internal/score/score.go` to derive the grade (Certain/Confident/Tentative/Unresolved) from each row's composite via the 80/50/20 bands instead of reading the hand-written Grade column; keep reading the `Scores` column for dimensions; add a `gradeFromComposite(c float64) string` helper (or equivalent) using shared band constants <!-- R6 -->

### Phase 2: Tests (score package — conformance to the v2 spec)

- [x] T007 Rewrite affected fixtures in `src/go/fab/internal/score/score_test.go` to the v2 formula: update `TestCompute_AllStrongDimensions`, `TestCompute_PerfectDimensionsScoreFive`, `TestCompute_MeanComposite_MixedRows`, `TestCompute_CoverFactor`, `TestCheckGate_Pass`, `TestCheckGate_Fail`, `TestCheckGate_IntakeStage`, `TestComputeWithStatus_MutatesInMemoryWithoutSaving`, and `TestCompute_DimensionlessRowStillCountsTowardCoverage`; replace `TestCompute_UnresolvedZero` and `TestCompute_CriticalRuleHardFail`/`TestCompute_CriticalRuleBoundaryDoesNotFail` (no longer hard-fails) with curve-derived expectations; update `TestConstants` for the new weights and removed constants <!-- R3 -->
- [x] T008 Add new test cases to `src/go/fab/internal/score/score_test.go`: four-band penalties (Certain/Confident/Tentative/Unresolved), the `c=20` boundary (penalty exactly 2.0 → one-row score 3.0 → passes), survive-one/block-two (one Tentative passes, two block), single-Unresolved-blocks (`c<20` alone fails the gate), and thin-but-strong (2-row all-Certain → 5.0) <!-- R3 R4 R5 -->
- [x] T009 Verify `src/go/fab/internal/score/changetypes_doc_test.go` still compiles and passes against the retained `getExpectedMin`/`getGateThreshold`; adjust only if it asserts the removed score-path behavior (it should not — it guards the doc tables, not the formula) <!-- R5 -->

### Phase 3: Skill & SPEC Documentation

- [x] T010 [P] Update `src/kit/skills/_srad.md`: SRAD Scoring aggregation to `0.20/0.30/0.30/0.20`, bands to 80/50/20, rewrite the Critical Rule (surface genuine unknowns; block via composite<20, no hard-fail / no `R<25 ∧ A<25` override), update the promptless-dispatch carve-out and Example 1 to the emergent-blocking wording <!-- R7 R9 -->
- [x] T011 [P] Update `docs/specs/skills/SPEC-_srad.md` to mirror `_srad.md`: summary line (weights, bands, Critical Rule) and the flow diagram's SRAD Scoring / Confidence Grades / Critical Rule boxes <!-- R7 R9 -->
- [x] T012 [P] Update `src/kit/skills/_cli-fab.md` § fab score (extended) § Formula to the demerit model (composite weights, penalty curve, `5.0 − Σ penalty`, clamp; remove the Critical-Rule / Unresolved-hard-fail / coverage / `expected_min` description) <!-- R8 -->
- [x] T013 [P] Replace the promptless-defer backstop wording in `src/kit/skills/_intake.md` and `src/kit/skills/fab-proceed.md` with the v2 contract (blocks only when composite < 20; score genuine unknowns with honestly-low dimensions) <!-- R9 -->
- [x] T014 [P] Replace the promptless-defer backstop wording in `docs/specs/skills/SPEC-fab-proceed.md` and `docs/specs/skills/SPEC-_intake.md` with the v2 contract, mirroring the skill edits <!-- R9 -->

### Phase 4: Build & Verify

- [x] T015 Run `cd src/go/fab && go build ./... && go vet ./internal/score/... && go test ./internal/score/...`; fix failures; then run `go test ./...` and report any pre-existing failures not caused by this change <!-- R1 R2 R3 R4 R5 R6 -->

## Execution Order

- Phase 1 (T001–T006) precedes Phase 2 tests (the tests assert the new behavior).
- T003 depends on T002 (uses `penalty`) and on T001 (new weights via `countGrades`); T004 and T003 both touch the short-circuit area — do T002→T001→T004→T003, then T005, T006.
- Phase 3 (T010–T014, all `[P]` — distinct files) is independent of Phases 1–2.
- T015 runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `score.go` composite weights are `0.20/0.30/0.30/0.20`, sum to 1.0, and `countGrades` computes the composite with them
- [x] A-002 R2: a `penalty(c float64) float64` helper exists with named free-knee/confident-floor/aggressive-slope constants and `compositeToScore` is gone
- [x] A-003 R3: `computeScore` returns `clamp(5.0 − Σ penalty(c), 0, 5)` and no longer takes/uses `total`/`expectedMin`
- [x] A-004 R4: `CriticalRowSeen` and `criticalDim` are removed; no `Unresolved>0`/`CriticalRowSeen` short-circuit remains
- [x] A-005 R5: `computeScore` has no coverage factor; `expectedMin`/`getExpectedMin` remain only for the doc-drift guard
- [x] A-006 R6: grade counts are derived from composite bands (80/50/20), not the hand-written Grade column; the `Scores` column is still parsed
- [x] A-007 R7: `_srad.md` states v2 weights/bands and the emergent-blocking Critical Rule; `SPEC-_srad.md` matches
- [x] A-008 R8: `_cli-fab.md` § fab score Formula describes the v2 demerit model
- [x] A-009 R9: all five promptless-defer files state "blocks only when composite < 20" (no "returns 0.0 on any Unresolved row")

### Behavioral Correctness

- [x] A-010 R3: a penalty sum of exactly 2.0 yields score 3.0; all-Certain yields 5.0; penalty sum ≥ 5.0 clamps to 0.0
- [x] A-011 R2: `penalty(80)=0`, `penalty(50)=0.50`, `penalty(20)=2.0`, `penalty(0)=3.0`, continuous at joins
- [x] A-012 R4: an intake with one Unresolved (`c<20`) row plus strong rows scores the curve value (≤3.0), not a hard 0.0
- [x] A-013 R6: a row whose hand-written grade contradicts its composite is counted by its composite-derived grade

### Scenario Coverage

- [x] A-014 R3 R4 R5: score_test.go covers four bands, the c=20 boundary (score 3.0, passes), survive-one/block-two, single-Unresolved-blocks, and thin-but-strong (5.0)
- [x] A-015 R5: `changetypes_doc_test.go` compiles and passes unchanged in intent

### Edge Cases & Error Handling

- [x] A-016 R3: a fully-dimensionless Assumptions table (DimCount 0) still scores 5.0 only if there are zero penalties — confirm the no-parseable-rows path is intentional and tested (a 0-row table yields 5.0 under pure demerit; verify against spec intent and keep the existing malformed-table guard if the spec requires it)
- [x] A-017 R3: the oversized-line truncation guard tests still hold under the new formula (counts unaffected; gate outcome recomputed)

### Code Quality

- [x] A-018 Pattern consistency: new code follows the score package's existing naming, error-handling, and comment style; no magic numbers (curve coefficients are named constants)
- [x] A-019 No unnecessary duplication: grade-band thresholds and weight constants are defined once and reused by both the parser and any helper
- [x] A-020 Readability: `penalty` and `computeScore` stay focused (well under the 50-line god-function bound)

### Documentation Accuracy

- [x] A-021 R7 R8 R9: skill/SPEC edits match the authoritative `docs/specs/srad.md` and `srad-scoring-rationale-v1-to-v2.md`; no v1 (`srad-v1.md`) wording is reintroduced

### Cross-References

- [x] A-022 R7 R9: each edited `src/kit/skills/*.md` has its corresponding `docs/specs/skills/SPEC-*.md` updated (constitution constraint); `.claude/skills/` is not edited

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- The installed `fab` binary (2.6.2) does NOT reflect these source edits — verify via `go test`/`go build` against the worktree, never `fab score`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Keep `expectedMin`/`getExpectedMin` in the package (doc-only) rather than deleting | `changetypes_doc_test.go` asserts `getExpectedMin` mirrors `change-types.md`; intake says leave the doc intact, and removing them breaks the doc-drift guard | S:95 R:80 A:95 D:90 |
| 2 | Certain | `penalty(c)` is a standalone package-level helper with named constants | Intake/spec § Implementation surface name the helper and constants explicitly; directly unit-testable | S:98 R:85 A:95 D:95 |
| 3 | Certain | Drop the `total`/`expectedMin` params from `computeScore` and fix both call sites | They become unused once coverage is removed; leaving dead params is a lint/style smell flagged by code-quality | S:90 R:80 A:90 D:85 |
| 4 | Confident | Grade derivation lives in `countGrades` via a `gradeFromComposite` helper sharing band constants | Parser already computes the composite; one source of truth matches spec intent; exact helper name is an implementation detail | S:80 R:75 A:85 D:75 |
| 5 | Confident | A 0-row / fully-dimensionless table scores 5.0 under pure demerit unless an explicit malformed-table guard is kept | Pure demerit has nothing to subtract; the existing DimCount==0 guard returns 0.0 — verify spec intent and preserve the malformed-intake guard so a dimensionless table does not pass | S:70 R:70 A:75 D:70 |

5 assumptions (3 certain, 2 confident, 0 tentative).
