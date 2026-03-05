# Spec: Add fab-fff Full-Pipeline Command with Confidence Gating

**Change**: 260208-k3m7-add-fab-fff
**Created**: 2026-02-08
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/change-lifecycle.md`, `fab/docs/fab-workflow/clarify.md`, `fab/docs/fab-workflow/execution-skills.md`

## Confidence Scoring: Schema and Formula

### Requirement: Confidence fields in `.status.yaml`

Every `.status.yaml` SHALL include a `confidence` block with the following fields:

- `certain` (integer) — count of Certain-graded SRAD decisions
- `confident` (integer) — count of Confident-graded SRAD decisions
- `tentative` (integer) — count of Tentative-graded SRAD decisions
- `unresolved` (integer) — count of Unresolved-graded SRAD decisions
- `score` (float, 1 decimal) — derived confidence score

#### Scenario: Initial confidence on new change

- **GIVEN** a user runs `/fab-new` to create a change
- **WHEN** the proposal is generated and SRAD decisions are evaluated
- **THEN** the `confidence` block is populated with counts for each grade and a derived score
- **AND** the score is written to `.status.yaml`

#### Scenario: All decisions are Certain

- **GIVEN** a change where all SRAD decisions are graded Certain
- **WHEN** the confidence score is computed
- **THEN** the score SHALL be `5.0`

#### Scenario: Unresolved decisions present

- **GIVEN** a change where `unresolved > 0`
- **WHEN** the confidence score is computed
- **THEN** the score SHALL be `0.0` regardless of other counts

### Requirement: Confidence score formula

The confidence score SHALL be computed as:

```
if unresolved > 0:
  score = 0.0
else:
  score = max(0.0, 5.0 - 0.1 * confident - 1.0 * tentative)
```

The score SHALL be clamped to the range `[0.0, 5.0]`.

#### Scenario: Mixed grades without Unresolved

- **GIVEN** `certain: 10, confident: 5, tentative: 1, unresolved: 0`
- **WHEN** the score is computed
- **THEN** the score SHALL be `max(0, 5.0 - 0.5 - 1.0)` = `3.5`

#### Scenario: Many Tentative decisions

- **GIVEN** `certain: 5, confident: 0, tentative: 5, unresolved: 0`
- **WHEN** the score is computed
- **THEN** the score SHALL be `max(0, 5.0 - 0 - 5.0)` = `0.0`

### Requirement: Confidence schema definition location

The canonical schema for the `confidence` block (field names, types, formula, gate threshold) SHALL be defined in `fab/.kit/skills/_context.md`.

#### Scenario: Agent needs to understand confidence fields

- **GIVEN** a fab skill needs to read or write confidence data
- **WHEN** the agent loads `_context.md` as part of its preamble
- **THEN** the confidence schema and formula are available in that document

### Requirement: Status template with confidence block

`fab/.kit/templates/status.yaml` SHALL exist as a stamp-out template for new changes, including the `confidence` block initialized to zero values.

#### Scenario: New change created from template

- **GIVEN** `/fab-new` creates a new change
- **WHEN** it initializes `.status.yaml`
- **THEN** it SHALL use the template, which includes `confidence: {certain: 0, confident: 0, tentative: 0, unresolved: 0, score: 5.0}`

## Confidence Scoring: Lifecycle

### Requirement: Manual skills recompute confidence

`/fab-new`, `/fab-continue`, and `/fab-clarify` SHALL recompute the confidence score after each invocation by re-counting SRAD grades across all artifacts in the change and applying the formula.

#### Scenario: fab-continue generates spec with new assumptions

- **GIVEN** a change at proposal stage with confidence score 4.8
- **WHEN** `/fab-continue` generates `spec.md` introducing 1 Tentative and 2 Confident assumptions
- **THEN** the confidence block in `.status.yaml` SHALL be updated with the new cumulative counts and recomputed score

#### Scenario: fab-clarify resolves a Tentative assumption

- **GIVEN** a change with `tentative: 2` and `score: 3.0`
- **WHEN** `/fab-clarify` resolves one Tentative assumption (promoting it to Certain or Confident)
- **THEN** the confidence block SHALL be recalculated with `tentative: 1` and the score updated accordingly

### Requirement: Autonomous skills do NOT recompute confidence

`/fab-ff` and `/fab-fff` SHALL NOT update the `confidence` block in `.status.yaml`. The gate check in `/fab-fff` uses the score from the last manual step.

#### Scenario: fab-ff generates artifacts with new assumptions

- **GIVEN** a change with confidence score 4.5
- **WHEN** `/fab-ff` runs and generates spec, plan, and tasks with new Tentative assumptions
- **THEN** the `confidence` block in `.status.yaml` SHALL remain unchanged at 4.5

## New Skill: `/fab-fff`

### Requirement: fab-fff chains the full pipeline

`/fab-fff` SHALL be a thin wrapper that chains: `/fab-ff` → `/fab-apply` → `/fab-review` → `/fab-archive`. Each stage uses the same behavior as its standalone invocation.

#### Scenario: Successful full pipeline

- **GIVEN** a change at proposal stage with confidence score >= 3.0
- **WHEN** the user runs `/fab-fff`
- **THEN** the skill SHALL execute fab-ff, fab-apply, fab-review, and fab-archive in sequence
- **AND** the change SHALL end in `archive: done` state

#### Scenario: fab-ff bails on blocking issues

- **GIVEN** a change where `/fab-ff` encounters blocking auto-clarify issues
- **WHEN** `/fab-fff` is running
- **THEN** `/fab-fff` SHALL stop at the point where `/fab-ff` bails
- **AND** report the blocking issues with suggestion: `Run /fab-clarify to resolve these, then /fab-fff to retry.`

### Requirement: Confidence gate on fab-fff

`/fab-fff` SHALL check `confidence.score >= 3.0` from `.status.yaml` before proceeding. If the score is below the threshold, the skill SHALL abort.

#### Scenario: Confidence below threshold

- **GIVEN** a change with `confidence.score: 2.5`
- **WHEN** the user runs `/fab-fff`
- **THEN** the skill SHALL abort with: `Confidence is 2.5 (need >= 3.0). Run /fab-clarify to resolve tentative/unresolved decisions, then retry.`

#### Scenario: Confidence at threshold

- **GIVEN** a change with `confidence.score: 3.0`
- **WHEN** the user runs `/fab-fff`
- **THEN** the confidence gate passes and the pipeline proceeds

### Requirement: fab-fff bails on review failure

If `/fab-review` fails during a `/fab-fff` run, the pipeline SHALL stop immediately. It SHALL NOT offer the interactive rework menu that standalone `/fab-review` provides.
<!-- assumed: fab-fff bails immediately on review failure rather than offering rework options — autonomous pipeline should not pause for interactive choices -->

#### Scenario: Review fails during fab-fff

- **GIVEN** `/fab-fff` has completed fab-ff and fab-apply
- **WHEN** `/fab-review` reports validation failures
- **THEN** `/fab-fff` SHALL stop and output the review failure details
- **AND** suggest: `Review failed. Run /fab-review to see rework options, or /fab-clarify to refine artifacts.`

### Requirement: fab-fff is resumable

`/fab-fff` SHALL be resumable by checking the progress map in `.status.yaml`. On re-invocation, it skips stages already marked `done` and continues from the first incomplete stage.

#### Scenario: Resume after interruption at apply stage

- **GIVEN** a previous `/fab-fff` run was interrupted during `/fab-apply` (specs and plan are `done`, tasks are `done`, apply is `active`)
- **WHEN** the user runs `/fab-fff` again
- **THEN** the skill SHALL skip fab-ff (all planning stages done) and resume from `/fab-apply`

### Requirement: fab-fff skill registration

A Claude Code skill registration file SHALL exist at `.claude/skills/fab-fff/prompt.md` to make `/fab-fff` available as a slash command.

#### Scenario: User invokes fab-fff

- **GIVEN** the skill registration file exists
- **WHEN** the user types `/fab-fff`
- **THEN** the skill is loaded and executed

### Requirement: fab-fff preflight emits confidence

`fab-preflight.sh` SHALL extract and emit the `confidence` fields from `.status.yaml` in its YAML output so that `/fab-fff` can read the score without separately parsing `.status.yaml`.

#### Scenario: Preflight output includes confidence

- **GIVEN** a `.status.yaml` with `confidence: {certain: 10, confident: 3, tentative: 1, unresolved: 0, score: 3.9}`
- **WHEN** `fab-preflight.sh` runs
- **THEN** the stdout YAML SHALL include the `confidence` block with all five fields

## Removals: `fab-ff --auto` Mode

### Requirement: Remove --auto flag from fab-ff

`/fab-ff` SHALL NOT accept a `--auto` flag. The full-auto mode is removed entirely. Only the default mode (frontload questions, interleaved auto-clarify, bail on blockers) SHALL remain.

#### Scenario: User attempts --auto flag

- **GIVEN** the updated `/fab-ff` skill
- **WHEN** a user or agent invokes `/fab-ff --auto`
- **THEN** the skill SHALL ignore the flag and run in default mode (or warn that `--auto` is no longer supported)

## Removals: `<!-- auto-guess -->` Marker System

### Requirement: Remove auto-guess marker production

No fab skill SHALL produce `<!-- auto-guess: {description} -->` markers. The `<!-- assumed: ... -->` (Tentative) markers SHALL remain in use.

#### Scenario: fab-ff encounters an Unresolved decision

- **GIVEN** `/fab-ff` (without --auto) encounters an Unresolved SRAD decision during auto-clarify
- **WHEN** the decision cannot be resolved autonomously
- **THEN** the pipeline SHALL bail (existing default behavior) rather than producing an auto-guess marker

### Requirement: Remove auto-guess references from _context.md

The Artifact Markers table and Skill-Specific Autonomy Levels table in `_context.md` SHALL be updated to remove all `<!-- auto-guess -->` references and the `fab-ff --auto` column.

#### Scenario: Agent reads _context.md

- **GIVEN** the updated `_context.md`
- **WHEN** an agent reads the Artifact Markers table
- **THEN** only `<!-- assumed: ... -->` and `<!-- clarified: ... -->` markers SHALL be listed

### Requirement: Remove auto-guess scanning from fab-clarify

`/fab-clarify` (both suggest and auto modes) SHALL no longer scan for `<!-- auto-guess: ... -->` markers. It SHALL continue scanning for `<!-- assumed: ... -->` markers.

#### Scenario: Clarify scans a spec artifact

- **GIVEN** an artifact containing only `<!-- assumed: ... -->` markers (no auto-guess markers exist)
- **WHEN** `/fab-clarify` performs its taxonomy scan
- **THEN** the scan detects and processes `<!-- assumed: ... -->` markers as before

## Removals: Auto-Guess Soft Gate from `/fab-apply`

### Requirement: Remove auto-guess soft gate from fab-apply

`/fab-apply` SHALL NOT scan for `<!-- auto-guess: ... -->` markers before beginning implementation. The soft gate section is removed entirely.

#### Scenario: fab-apply starts implementation

- **GIVEN** the updated `/fab-apply` skill
- **WHEN** `/fab-apply` passes preflight checks
- **THEN** it proceeds directly to context loading and task execution without any marker scan

## Documentation Updates

### Requirement: Update _context.md with fab-fff and confidence schema

`_context.md` SHALL be updated to:

1. Add `/fab-fff` to the Next Steps Lookup Table
2. Replace the `fab-ff --auto` column in the Skill-Specific Autonomy Levels table with a `/fab-fff` column
3. Remove the `<!-- auto-guess -->` row from the Artifact Markers table
4. Add a Confidence Scoring section defining the schema, formula, gate threshold, and lifecycle (which skills recompute, which don't)

#### Scenario: Agent reads updated _context.md

- **GIVEN** the updated `_context.md`
- **WHEN** an agent reads the Next Steps table
- **THEN** `/fab-fff` appears with: `Next: /fab-new <description> (start next change)` (same as archive)

### Requirement: Update centralized docs

The following centralized docs SHALL be updated during `/fab-archive` hydration:

- `fab-workflow/planning-skills.md` — add `/fab-fff` documentation, remove `/fab-ff --auto` documentation, document `/fab-continue` confidence recomputation
- `fab-workflow/change-lifecycle.md` — add the fab-fff path and confidence fields to `.status.yaml` schema
- `fab-workflow/clarify.md` — remove `<!-- auto-guess -->` references, add confidence recomputation behavior
- `fab-workflow/execution-skills.md` — remove auto-guess soft gate from `/fab-apply` documentation

#### Scenario: Centralized docs reflect new workflow

- **GIVEN** this change has been archived
- **WHEN** a user reads `planning-skills.md`
- **THEN** `/fab-fff` is documented alongside `/fab-ff`, and `/fab-ff --auto` is no longer mentioned

## Deprecated Requirements

### Auto-Guess Soft Gate (from execution-skills.md)
**Reason**: The `<!-- auto-guess -->` marker system is being removed. With `/fab-ff --auto` gone, no skill produces these markers, so scanning for them is unnecessary.
**Migration**: Replaced by confidence gating on `/fab-fff`. Users raise confidence via `/fab-clarify` before running the autonomous pipeline.

### Full-Auto Mode for fab-ff (from planning-skills.md)
**Reason**: `/fab-ff --auto` defers human interaction rather than eliminating it — auto-guess markers still need `/fab-clarify` resolution. The replacement workflow (`/fab-clarify` → `/fab-fff`) addresses the root problem.
**Migration**: Use `/fab-clarify` to raise confidence, then `/fab-fff` for full autonomous pipeline.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | `/fab-fff` bails immediately on review failure without offering the interactive rework menu | Autonomous pipeline should not pause for interactive choices; user can run `/fab-review` standalone for rework options |

1 assumption made (1 confident, 0 tentative).
