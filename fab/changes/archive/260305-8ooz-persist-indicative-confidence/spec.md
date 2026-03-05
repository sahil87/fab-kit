# Spec: Persist Indicative Confidence

**Change**: 260305-8ooz-persist-indicative-confidence
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/kit-scripts.md`, `docs/memory/fab-workflow/planning-skills.md`

## Non-Goals

- Changing the `--check-gate` mode behavior — it remains read-only
- Adding a separate `indicative_confidence` block in `.status.yaml` — the agreed approach reuses the existing `confidence` block with an `indicative: true` flag
- Modifying how spec-stage confidence scoring works — only the persistence and consumer display changes

## Scripts: calc-score.sh

### Requirement: Indicative flag in normal-mode intake scoring

When `calc-score.sh` runs in normal mode (not `--check-gate`) with `--stage intake`, it SHALL write the confidence block to `.status.yaml` with `confidence.indicative: true`. When `calc-score.sh` runs in normal mode without `--stage intake` (spec scoring), it SHALL ensure `confidence.indicative` is absent from the written block (clearing the flag if present from a prior intake scoring).

#### Scenario: Intake-stage normal scoring persists indicative flag
- **GIVEN** a change at the intake stage with `intake.md` containing an Assumptions table
- **WHEN** `calc-score.sh --stage intake <change>` runs in normal mode
- **THEN** the confidence block in `.status.yaml` includes `indicative: true`
- **AND** the score, grade counts, and dimensions are written as before

#### Scenario: Spec-stage normal scoring clears indicative flag
- **GIVEN** a change with `confidence.indicative: true` in `.status.yaml` (from prior intake scoring)
- **WHEN** `calc-score.sh <change>` runs in normal mode (spec scoring)
- **THEN** the confidence block in `.status.yaml` does NOT contain `indicative: true`
- **AND** the score is computed from `spec.md` Assumptions table as before

### Requirement: Check-gate mode unchanged

The `--check-gate` mode SHALL remain read-only. It MUST NOT write to `.status.yaml` regardless of `--stage` flag.

#### Scenario: Check-gate with intake stage is read-only
- **GIVEN** a change with `intake.md`
- **WHEN** `calc-score.sh --check-gate --stage intake <change>` runs
- **THEN** `.status.yaml` is not modified
- **AND** gate result is output to stdout

## Scripts: statusman.sh

### Requirement: Extended confidence accessor

`get_confidence` SHALL additionally output `indicative:{true|false}`, reading `confidence.indicative` from `.status.yaml` and defaulting to `false` when the key is absent.

#### Scenario: Reading confidence with indicative flag present
- **GIVEN** `.status.yaml` contains `confidence.indicative: true`
- **WHEN** `statusman.sh confidence <change>` is called
- **THEN** the output includes `indicative:true` alongside existing fields (certain, confident, tentative, unresolved, score)

#### Scenario: Reading confidence without indicative flag
- **GIVEN** `.status.yaml` has no `confidence.indicative` key
- **WHEN** `statusman.sh confidence <change>` is called
- **THEN** the output includes `indicative:false`

### Requirement: Extended confidence writers with --indicative flag

`set_confidence_block` and `set_confidence_block_fuzzy` SHALL accept an optional `--indicative` trailing flag. When `--indicative` is passed, the written confidence block SHALL include `indicative: true`. When `--indicative` is not passed, the `indicative` key SHALL be absent from the written block.

The CLI forms become:
- `set-confidence <change> <certain> <confident> <tentative> <unresolved> <score> [--indicative]`
- `set-confidence-fuzzy <change> <certain> <confident> <tentative> <unresolved> <score> <mean_s> <mean_r> <mean_a> <mean_d> [--indicative]`

#### Scenario: Writing confidence with --indicative flag
- **GIVEN** a valid `.status.yaml`
- **WHEN** `statusman.sh set-confidence <change> 6 3 0 0 4.1 --indicative` is called
- **THEN** `.status.yaml` contains `confidence.indicative: true`

#### Scenario: Writing confidence without --indicative flag
- **GIVEN** a `.status.yaml` that currently has `confidence.indicative: true`
- **WHEN** `statusman.sh set-confidence <change> 9 0 0 0 5.0` is called (no `--indicative`)
- **THEN** `.status.yaml` does NOT contain `confidence.indicative`

## Scripts: changeman.sh

### Requirement: List output includes confidence

`changeman.sh list` SHALL output lines in the format `name:display_stage:display_state:score:indicative`, extending the current `name:display_stage:display_state` format. The `score` is the confidence score from `.status.yaml` (e.g., `4.1` or `0.0`). The `indicative` field is `true` or `false`. Read via `statusman.sh confidence` for each change.

#### Scenario: List with indicative confidence
- **GIVEN** a change at intake stage with `confidence.indicative: true` and `confidence.score: 4.1`
- **WHEN** `changeman.sh list` is called
- **THEN** the output line for that change is `{name}:intake:ready:4.1:true`

#### Scenario: List with spec-persisted confidence
- **GIVEN** a change at spec stage with no `confidence.indicative` and `confidence.score: 3.5`
- **WHEN** `changeman.sh list` is called
- **THEN** the output line is `{name}:spec:ready:3.5:false`

### Requirement: Switch output includes confidence line

`changeman.sh switch` SHALL include a `Confidence:` line in its output between the `Stage:` and `Next:` lines. The format is `Confidence:  {score} of 5.0{indicative_suffix}`, where `{indicative_suffix}` is ` (indicative)` when `confidence.indicative` is true, empty otherwise. When score is `0.0` and no assumptions exist (template default), display `not yet scored`.

#### Scenario: Switch to change with indicative score
- **GIVEN** a change with `confidence.score: 4.1` and `confidence.indicative: true`
- **WHEN** `changeman.sh switch <name>` is called
- **THEN** the output includes `Confidence:  4.1 of 5.0 (indicative)`

#### Scenario: Switch to change with spec score
- **GIVEN** a change with `confidence.score: 3.5` and no `confidence.indicative`
- **WHEN** `changeman.sh switch <name>` is called
- **THEN** the output includes `Confidence:  3.5 of 5.0`

#### Scenario: Switch to change with no scored confidence
- **GIVEN** a change with `confidence.score: 0.0` (template default) and all grade counts at 0
- **WHEN** `changeman.sh switch <name>` is called
- **THEN** the output includes `Confidence:  not yet scored`

## Scripts: preflight.sh

### Requirement: Indicative field in confidence output

The preflight YAML output SHALL include `indicative: {true|false}` in the confidence section, reading from `statusman.sh confidence` output.

#### Scenario: Preflight output with indicative score
- **GIVEN** a change with `confidence.indicative: true`
- **WHEN** `preflight.sh` runs
- **THEN** the YAML output includes `indicative: true` under the `confidence:` block

#### Scenario: Preflight output without indicative flag
- **GIVEN** a change without `confidence.indicative` in `.status.yaml`
- **WHEN** `preflight.sh` runs
- **THEN** the YAML output includes `indicative: false` under the `confidence:` block

## Skills: fab-new.md

### Requirement: Step 7 persists indicative score via script

`/fab-new` Step 7 ("Indicative Confidence") SHALL call `calc-score.sh --stage intake <change>` in normal mode (not `--check-gate`) to persist the indicative score to `.status.yaml` with `indicative: true`. This replaces the current inline computation and display-only behavior.

#### Scenario: fab-new persists indicative confidence
- **GIVEN** a newly created change with `intake.md` containing an Assumptions table
- **WHEN** `/fab-new` completes Step 7
- **THEN** `.status.yaml` contains the confidence block with computed score and `indicative: true`
- **AND** the score is displayed to the user

## Skills: fab-status.md

### Requirement: Uniform confidence display from .status.yaml

`/fab-status` SHALL remove the intake-stage special case that calls `calc-score.sh --check-gate --stage intake` live. Instead, it SHALL uniformly read the confidence block from `.status.yaml` (via preflight output) for all stages.

Display rules:
- **Score > 0.0 with `indicative: true`**: `Indicative confidence: {score} of 5.0 ({breakdown})`
- **Score > 0.0 without `indicative`**: `Confidence: {score} of 5.0 ({breakdown})`
- **Score = 0.0 (template default, pre-intake)**: `Confidence: not yet scored`

#### Scenario: fab-status shows indicative confidence at intake
- **GIVEN** a change at intake stage with `confidence.score: 4.1` and `confidence.indicative: true`
- **WHEN** `/fab-status` runs
- **THEN** it displays `Indicative confidence: 4.1 of 5.0 (6 certain, 3 confident)`
- **AND** it does NOT call `calc-score.sh`

#### Scenario: fab-status shows spec confidence
- **GIVEN** a change at spec stage with `confidence.score: 3.5` and no `indicative` flag
- **WHEN** `/fab-status` runs
- **THEN** it displays `Confidence: 3.5 of 5.0 (...)`

## Skills: fab-switch.md

### Requirement: Confidence visible in switch output

After the `changeman.sh switch` output change (which now includes the Confidence line), `/fab-switch` displays it without additional skill-level confidence logic.

For the no-argument flow (listing changes), the skill reads `changeman.sh list` output (which now includes `:score:indicative`) and displays it alongside stage info in the numbered list.

#### Scenario: fab-switch shows confidence in list
- **GIVEN** two active changes, one with indicative score 4.1 and one with spec score 3.5
- **WHEN** `/fab-switch` is called with no arguments
- **THEN** the numbered list shows confidence for each change (e.g., `4.1 (indicative)` and `3.5`)

## Skills: _preamble.md

### Requirement: Update Confidence Scoring documentation

The Confidence Scoring section SHALL document:
- The `indicative: true` flag in `.status.yaml`
- That `/fab-new` persists indicative scores (no longer display-only)
- That consumers read uniformly from `.status.yaml`

#### Scenario: Updated schema example
- **GIVEN** the `_preamble.md` Confidence Scoring section
- **WHEN** a reader examines the schema
- **THEN** the example YAML includes `indicative: true` as an optional field

## Backward Compatibility

### Requirement: Missing indicative key defaults to false

All consumers (statusman, changeman, preflight, skills) SHALL treat a missing `confidence.indicative` key as `false`. This ensures existing `.status.yaml` files work without migration.

#### Scenario: Legacy status file without indicative key
- **GIVEN** a `.status.yaml` created before this change (no `confidence.indicative` field)
- **WHEN** any consumer reads the confidence block
- **THEN** `indicative` is treated as `false`
- **AND** confidence is displayed as a normal (non-indicative) score

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Same `confidence` block with `indicative: true` flag (not separate block) | Confirmed from intake #1 — user chose Option A; uniform consumer reads | S:95 R:90 A:95 D:95 |
| 2 | Certain | `/fab-new` calls `calc-score.sh --stage intake` in normal mode to persist | Confirmed from intake #2 — user agreed to persist at intake finish | S:95 R:85 A:90 D:95 |
| 3 | Certain | Spec-stage scoring clears the `indicative` flag | Confirmed from intake #3 — spec overwrites with real score | S:90 R:90 A:95 D:95 |
| 4 | Certain | `changeman.sh list` appends `:score:indicative` to output format | Confirmed from intake #4 — user agreed to script-level output changes | S:90 R:85 A:90 D:90 |
| 5 | Certain | `changeman.sh switch` adds Confidence line | Confirmed from intake #5 — user agreed | S:90 R:85 A:90 D:90 |
| 6 | Certain | `fab-status` removes live calc-score call, reads .status.yaml uniformly | Confirmed from intake #6 — this is the core simplification | S:95 R:80 A:95 D:95 |
| 7 | Certain | `statusman.sh` CLI uses `--indicative` flag (not positional arg) | Upgraded from intake Confident #7 — strong convention match with existing CLI flags | S:85 R:90 A:90 D:85 |
| 8 | Certain | Pre-intake changes show "not yet scored" (not "0.0") | Upgraded from intake Confident #8 — current fab-status already does this | S:85 R:90 A:85 D:85 |
| 9 | Certain | No migration needed — missing `indicative` defaults to `false` | Upgraded from intake Confident #9 — backward compat is clean | S:85 R:85 A:90 D:85 |
| 10 | Confident | `calc-score.sh` passes `--indicative` to statusman for intake scoring | Logical consequence of #2 and #7 — calc-score calls statusman with the flag | S:80 R:90 A:85 D:80 |

10 assumptions (9 certain, 1 confident, 0 tentative, 0 unresolved).
