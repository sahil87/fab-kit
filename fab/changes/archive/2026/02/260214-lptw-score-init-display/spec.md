# Spec: Confidence score initial value and display format

**Change**: 260214-lptw-score-init-display
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/planning-skills.md`, `fab/memory/fab-workflow/change-lifecycle.md`, `fab/memory/fab-workflow/templates.md`

## Non-Goals

- Changing the confidence scoring formula — only the initial value and display format change
- Migrating existing `.status.yaml` files in active changes — they retain their current scores

## Confidence Scoring: Initial Value

### Requirement: Initial confidence score SHALL be 0.0

The `.status.yaml` template SHALL initialize `confidence.score` to `0.0` instead of `5.0`. A new change with zero scored decisions has no assessed confidence — the score MUST reflect this by starting at zero rather than implying perfect confidence.

#### Scenario: New change creation

- **GIVEN** a user runs `/fab-new` to create a new change
- **WHEN** `.status.yaml` is initialized from the template
- **THEN** `confidence.score` SHALL be `0.0`
- **AND** `confidence.certain`, `confidence.confident`, `confidence.tentative`, and `confidence.unresolved` SHALL all be `0`

### Requirement: Fallback score in `_calc-score.sh` SHALL be 0.0

The `prev_score` fallback value in `_calc-score.sh` SHALL be `"0.0"` instead of `"5.0"`. This ensures that when no previous score exists in `.status.yaml`, the delta calculation uses 0.0 as the baseline.

#### Scenario: First score computation on a new change

- **GIVEN** a new change with `confidence.score: 0.0` in `.status.yaml`
- **WHEN** `_calc-score.sh` computes the first confidence score after spec generation
- **THEN** the `prev_score` fallback SHALL be `0.0`
- **AND** the delta SHALL be calculated relative to `0.0`

## Confidence Scoring: Display Format

### Requirement: Score display SHALL use "of 5.0" format

All confidence score displays SHALL use the format `{score} of 5.0` instead of `{score}/5.0`. This makes the scoring scale self-documenting.

#### Scenario: `/fab-status` confidence display

- **GIVEN** a change with `confidence.score: 2.4`
- **WHEN** `/fab-status` renders the confidence line
- **THEN** the display SHALL read `Confidence: 2.4 of 5.0 ({N} certain, {N} confident, {N} tentative)`

#### Scenario: `/fab-fff` confidence gate failure

- **GIVEN** a change with `confidence.score: 1.8`
- **WHEN** `/fab-fff` checks the confidence gate and the score is below 3.0
- **THEN** the message SHALL read `Confidence is 1.8 of 5.0 (need >= 3.0). Run /fab-clarify to resolve, then retry.`

#### Scenario: `/fab-fff` confidence gate pass

- **GIVEN** a change with `confidence.score: 4.1`
- **WHEN** `/fab-fff` passes the confidence gate
- **THEN** the output header SHALL read `/fab-fff — confidence 4.1 of 5.0, gate passed.`

## Documentation: Context Preamble Updates

### Requirement: `_context.md` template description SHALL reflect 0.0 default

The Confidence Scoring → Template section in `_context.md` SHALL describe the template default as "zero counts and score 0.0" instead of "zero counts and score 5.0".

#### Scenario: Reading `_context.md` template description

- **GIVEN** a developer reading the `_context.md` Confidence Scoring section
- **WHEN** they read the Template subsection
- **THEN** it SHALL state that `fab/.kit/templates/status.yaml` includes the confidence block initialized to zero counts and score 0.0
