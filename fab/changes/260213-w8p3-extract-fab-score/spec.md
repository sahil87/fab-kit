# Spec: Extract confidence scoring into standalone script

**Change**: 260213-w8p3-extract-fab-score
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/planning-skills.md` (modify), `fab/memory/fab-workflow/change-lifecycle.md` (modify)

## Non-Goals

- Scanning `tasks.md` for assumptions — analysis of 48 archived changes shows only 6% contain Assumptions tables, all Confident-grade about task grouping. Not worth the complexity.
- Changing `/fab-ff` or `/fab-fff` behavior — they already don't recompute. Gate check reads stored score, unaffected.
- Altering the confidence formula or penalty weights — existing formula is unchanged.

## Script: `_fab-score.sh`

### Requirement: Script Interface

`fab/.kit/scripts/_fab-score.sh` SHALL accept a single positional argument (`$1`) containing the change directory path. It SHALL emit a YAML confidence block to stdout on success (exit 0) and an error message to stderr on failure (exit 1).

#### Scenario: Successful scoring

- **GIVEN** a change directory with `brief.md` and `spec.md` containing `## Assumptions` tables
- **WHEN** `_fab-score.sh $change_dir` is executed
- **THEN** stdout SHALL contain a YAML block with `certain`, `confident`, `tentative`, `unresolved` counts, computed `score`, and `delta` (difference from previous score)
- **AND** `.status.yaml` in the change directory SHALL be updated with the new confidence block
- **AND** exit code SHALL be 0

#### Scenario: Missing spec.md

- **GIVEN** a change directory with `brief.md` but no `spec.md`
- **WHEN** `_fab-score.sh $change_dir` is executed
- **THEN** stderr SHALL contain "spec.md required for scoring"
- **AND** exit code SHALL be 1
- **AND** `.status.yaml` SHALL NOT be modified

#### Scenario: Missing change directory

- **GIVEN** an invalid or nonexistent path as `$1`
- **WHEN** `_fab-score.sh` is executed
- **THEN** stderr SHALL contain an error message
- **AND** exit code SHALL be 1

### Requirement: Assumptions Table Scanning

The script SHALL scan `## Assumptions` tables in `brief.md` and `spec.md` only. For each file, it SHALL locate the `## Assumptions` heading, parse the markdown table rows (skipping header and separator rows), and extract the Grade column value from each data row.

#### Scenario: Both files have Assumptions tables

- **GIVEN** `brief.md` has an `## Assumptions` table with 3 Confident rows
- **AND** `spec.md` has an `## Assumptions` table with 1 Confident and 1 Tentative row
- **WHEN** the script scans both files
- **THEN** the table-sourced counts SHALL be: confident=4, tentative=1

#### Scenario: File has no Assumptions section

- **GIVEN** `brief.md` has no `## Assumptions` heading
- **AND** `spec.md` has an `## Assumptions` table with 2 Confident rows
- **WHEN** the script scans both files
- **THEN** `brief.md` SHALL contribute 0 counts
- **AND** the total table-sourced counts SHALL be: confident=2

#### Scenario: Grade matching is case-insensitive

- **GIVEN** an Assumptions table row with Grade value "confident" (lowercase)
- **WHEN** the script parses the row
- **THEN** it SHALL count as a Confident-graded decision

### Requirement: Carry-Forward for Implicit Certain Counts

The script SHALL preserve implicit Certain counts that are not represented in Assumptions tables. It SHALL read the existing `certain` count from `.status.yaml`, subtract the number of Certain-graded rows found in Assumptions tables, and add the remainder (implicit count) to the newly computed Certain total.

#### Scenario: Carry-forward with existing Certain count

- **GIVEN** `.status.yaml` has `certain: 8`
- **AND** `brief.md` Assumptions table has 0 Certain rows
- **AND** `spec.md` Assumptions table has 1 Certain row
- **WHEN** the script computes the Certain count
- **THEN** the result SHALL be `(8 - 1) + 1 = 8` (7 implicit + 1 explicit)

#### Scenario: No prior Certain count

- **GIVEN** `.status.yaml` has `certain: 0` (template default)
- **AND** no Certain-graded rows appear in any Assumptions table
- **WHEN** the script computes the Certain count
- **THEN** the result SHALL be 0

### Requirement: Formula Application

The script SHALL apply the existing confidence formula without modification:

```
if unresolved > 0:
  score = 0.0
else:
  score = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
```

The `unresolved` count SHALL always be 0 in script output because Unresolved decisions are asked interactively by skills and never appear in Assumptions tables.

#### Scenario: Standard score computation

- **GIVEN** counts of certain=8, confident=3, tentative=0, unresolved=0
- **WHEN** the formula is applied
- **THEN** score SHALL be `max(0.0, 5.0 - 0.9 - 0.0) = 4.1`

#### Scenario: Score clamped at zero

- **GIVEN** counts of certain=2, confident=0, tentative=6, unresolved=0
- **WHEN** the formula is applied
- **THEN** score SHALL be `max(0.0, 5.0 - 6.0) = 0.0`

### Requirement: Status File Update

The script SHALL write the computed confidence block to `.status.yaml` in the change directory, replacing the existing `confidence:` block. The write mechanism SHALL use sed (consistent with other `.kit/scripts/`).

#### Scenario: Update existing confidence block

- **GIVEN** `.status.yaml` contains a `confidence:` block with previous values
- **WHEN** the script completes scoring
- **THEN** the `confidence:` block SHALL be replaced with updated counts and score
- **AND** no other fields in `.status.yaml` SHALL be modified

### Requirement: Delta Output

The script's stdout YAML SHALL include a `delta` field showing the difference between the newly computed score and the previous score from `.status.yaml`.

#### Scenario: Score improved after clarification

- **GIVEN** `.status.yaml` has `score: 3.8`
- **AND** the newly computed score is `4.1`
- **WHEN** the script emits stdout
- **THEN** `delta` SHALL be `+0.3`

#### Scenario: First computation replaces template default

- **GIVEN** `.status.yaml` has `score: 5.0` (template default)
- **AND** the newly computed score is `4.1`
- **WHEN** the script emits stdout
- **THEN** `delta` SHALL be `-0.9`

## Skill Modifications: Remove Inline Scoring

### Requirement: Remove scoring from `/fab-new`

`fab/.kit/skills/fab-new.md` SHALL NOT contain any confidence score computation logic. The template defaults in `.status.yaml` (score 5.0, zero counts) SHALL persist until `/fab-continue` generates the spec and invokes `_fab-score.sh`.

#### Scenario: New change created

- **GIVEN** a user runs `/fab-new "description"`
- **WHEN** the brief is generated and `.status.yaml` initialized
- **THEN** the confidence block SHALL contain template defaults: `certain: 0, confident: 0, tentative: 0, unresolved: 0, score: 5.0`
- **AND** no scoring computation SHALL occur

### Requirement: Invoke `_fab-score.sh` from `/fab-continue` at spec stage only

`fab/.kit/skills/fab-continue.md` SHALL invoke `_fab-score.sh` after spec generation completes (both normal flow and reset flow when resetting to spec). The script SHALL NOT be invoked at any other stage (brief, tasks, apply, review, hydrate).

#### Scenario: Normal spec generation

- **GIVEN** a change at the brief stage
- **WHEN** `/fab-continue` generates `spec.md`
- **THEN** `_fab-score.sh` SHALL be invoked with the change directory path
- **AND** the updated confidence score SHALL be written to `.status.yaml`

#### Scenario: Reset to spec stage

- **GIVEN** a change at any later stage
- **WHEN** `/fab-continue spec` resets and regenerates the spec
- **THEN** `_fab-score.sh` SHALL be invoked after regeneration

#### Scenario: Tasks stage does not trigger scoring

- **GIVEN** a change at the spec stage (done)
- **WHEN** `/fab-continue` generates `tasks.md`
- **THEN** `_fab-score.sh` SHALL NOT be invoked
- **AND** the confidence block in `.status.yaml` SHALL be unchanged

### Requirement: Invoke `_fab-score.sh` from `/fab-clarify` in suggest mode only

`fab/.kit/skills/fab-clarify.md` SHALL invoke `_fab-score.sh` after completing a suggest-mode session, provided `spec.md` exists. Auto mode (invoked internally by `/fab-ff`) SHALL NOT invoke the script.

#### Scenario: Suggest mode with spec present

- **GIVEN** a change at spec stage with `spec.md` present
- **WHEN** the user runs `/fab-clarify` and completes a suggest-mode session
- **THEN** `_fab-score.sh` SHALL be invoked
- **AND** the updated confidence SHALL be reported

#### Scenario: Suggest mode at brief stage (no spec)

- **GIVEN** a change at brief stage with no `spec.md`
- **WHEN** the user runs `/fab-clarify` and completes a suggest-mode session
- **THEN** the scoring step SHALL be skipped (no `_fab-score.sh` invocation)

#### Scenario: Auto mode does not score

- **GIVEN** `/fab-ff` invokes `/fab-clarify` with `[AUTO-MODE]` prefix
- **WHEN** auto-clarify completes
- **THEN** `_fab-score.sh` SHALL NOT be invoked

## Documentation Updates

### Requirement: Simplify `_context.md` Confidence Scoring section

The `_context.md` Confidence Scoring section SHALL have the Lifecycle table removed. A one-liner SHALL replace it stating that `_fab-score.sh` is invoked by `/fab-continue` (spec stage) and `/fab-clarify` (suggest mode), and that autonomous skills do not recompute. The Template note SHALL be updated to reflect that template defaults persist until `/fab-continue` invokes `_fab-score.sh`.

#### Scenario: Lifecycle table removed

- **GIVEN** the current `_context.md` with a 5-row Lifecycle table under Confidence Scoring
- **WHEN** the change is applied
- **THEN** the Lifecycle table SHALL be replaced by a single paragraph
- **AND** the Skill-Specific Autonomy table's "Recomputes confidence?" row SHALL show: `/fab-new`: No, `/fab-continue`: Spec stage only, `/fab-ff`: No, `/fab-fff`: No

### Requirement: Simplify `srad.md` Confidence Lifecycle section

`fab/specs/srad.md` SHALL replace its 5-row Confidence Lifecycle table with a simplified 3-row table covering computation (by `/fab-continue`), recomputation (by `/fab-clarify`), and gate check (by `/fab-fff`). The Skill-Specific Autonomy table SHALL receive the same "Recomputes confidence?" row update.

#### Scenario: Lifecycle table simplified

- **GIVEN** the current `srad.md` with 5 lifecycle rows
- **WHEN** the change is applied
- **THEN** the Lifecycle table SHALL have exactly 3 rows: Computation, Recomputation, Gate check

### Requirement: Update memory file references

`fab/memory/fab-workflow/planning-skills.md` SHALL remove the `/fab-new` Confidence Scoring paragraph, update the `/fab-continue` forward flow step to reference `_fab-score.sh`, and update the `/fab-fff` confidence recomputation note. `fab/memory/fab-workflow/change-lifecycle.md` SHALL update the confidence field description.

#### Scenario: planning-skills.md updated

- **GIVEN** the current planning-skills.md with a Confidence Scoring paragraph under `/fab-new`
- **WHEN** the change is applied
- **THEN** the paragraph SHALL be removed
- **AND** `/fab-continue` step 6 SHALL reference `_fab-score.sh` at spec stage only

#### Scenario: change-lifecycle.md updated

- **GIVEN** the current change-lifecycle.md with "Computed by `/fab-new`, recomputed by `/fab-continue` and `/fab-clarify`"
- **WHEN** the change is applied
- **THEN** it SHALL read "Computed by `_fab-score.sh`, invoked at spec stage by `/fab-continue` and by `/fab-clarify`"

## Design Decisions

1. **Script-only, no skill wrapper**: `_fab-score.sh` is a standalone script, not wrapped by a `/fab-score` skill.
   - *Why*: Users see scores via `/fab-status`. No interactive use case for a scoring skill. Scripts are simpler, faster, and testable from the terminal.
   - *Rejected*: Skill wrapper — adds unnecessary indirection for a non-interactive operation.

2. **Scan brief + spec only, exclude tasks.md**: The script scans `## Assumptions` tables in `brief.md` and `spec.md`, not `tasks.md`.
   - *Why*: Analysis of 48 archived changes shows Assumptions tables in brief (46%), spec (81%), tasks (6%). The 3 tasks.md instances contain only Confident-grade assumptions about task grouping — not design decisions. Scanning brief + spec captures effectively all meaningful signal.
   - *Rejected*: Scanning all artifacts — added complexity for negligible signal.

3. **No scoring at brief stage**: `/fab-new` no longer computes confidence. Template defaults persist until spec stage.
   - *Why*: Brief-stage scoring is premature — real decisions emerge at spec stage. Removing it simplifies `/fab-new` and makes the scoring trigger model clearer: score appears only when there's a spec to score.
   - *Rejected*: Keep brief-stage scoring — premature, adds complexity to `/fab-new` for minimal value.

## Deprecated Requirements

### Inline Confidence Computation in `/fab-new`
**Reason**: Replaced by centralized `_fab-score.sh`. Brief-stage scoring was premature.
**Migration**: Template defaults (score 5.0) persist until `/fab-continue` generates spec.

### Inline Confidence Recomputation in `/fab-continue` Step 3b
**Reason**: Replaced by `_fab-score.sh` invocation at spec stage only. Other planning stages no longer trigger scoring.
**Migration**: Script invocation after spec generation.

### Inline Confidence Recomputation in `/fab-clarify` Step 7
**Reason**: Replaced by `_fab-score.sh` invocation in suggest mode.
**Migration**: Script invocation after suggest-mode session.

### Confidence Lifecycle Table in `_context.md`
**Reason**: Redundant with the Skill-Specific Autonomy table's "Recomputes confidence?" row. Replaced by a one-liner.
**Migration**: N/A — informational documentation change.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Include `delta` field in stdout showing score change | Plan specifies it; useful for skill output to report score movement to the user |

1 assumption made (1 confident, 0 tentative). Run /fab-clarify to review.
