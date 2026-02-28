# Spec: Richer Git PR Output

**Change**: 260227-8q33-richer-git-pr-output
**Created**: 2026-02-27
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## git-pr: Tier 1 Context Table

### Requirement: Fix Artifact Link Rows

The Tier 1 PR body template SHALL render Intake and Spec as separate rows in the `Field | Detail` column pattern, replacing the current single row that places both links as sibling cells.

Each artifact link row SHALL use `[{change_name}/{filename}]({blob_url})` as the Detail value (e.g., `| Intake | [260227-8q33-richer-git-pr-output/intake.md](https://github.com/...) |`).

If `spec.md` does not exist in the change folder, the Spec row SHALL be omitted entirely.

#### Scenario: Both artifacts exist
- **GIVEN** a Tier 1 PR (feat/fix/refactor) with an active change that has both `intake.md` and `spec.md`
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Context table contains separate `| Intake | [link] |` and `| Spec | [link] |` rows
- **AND** each link uses `{change_name}/{filename}` as the link text

#### Scenario: Only intake exists
- **GIVEN** a Tier 1 PR with an active change that has `intake.md` but no `spec.md`
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Context table contains an `| Intake | [link] |` row
- **AND** the Spec row is omitted entirely (no empty cell)

### Requirement: Add Confidence Row

The Tier 1 Context table SHALL include a `| Confidence | {score} / 5.0 |` row, reading `confidence.score` from `.status.yaml`.

#### Scenario: Confidence score present
- **GIVEN** a Tier 1 PR with `.status.yaml` containing `confidence.score: 3.5`
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Context table contains `| Confidence | 3.5 / 5.0 |`

#### Scenario: Default confidence (score 0.0)
- **GIVEN** a Tier 1 PR with `.status.yaml` containing `confidence.score: 0.0`
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Context table contains `| Confidence | 0.0 / 5.0 |`

### Requirement: Add Pipeline Row

The Tier 1 Context table SHALL include a `| Pipeline | {stages} |` row showing completed stages from the `progress` map in `.status.yaml`.

The pipeline value SHALL list stage names whose status is `done`, joined with ` → ` (e.g., `intake → spec → tasks → apply → review → hydrate`).

The stage order SHALL always be: intake, spec, tasks, apply, review, hydrate — regardless of the order in the YAML map.

#### Scenario: Full pipeline completed
- **GIVEN** `.status.yaml` with all six stages at `done`
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Pipeline row reads `intake → spec → tasks → apply → review → hydrate`

#### Scenario: Partial pipeline
- **GIVEN** `.status.yaml` with intake, spec, tasks, apply at `done` and review, hydrate at other states
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Pipeline row reads `intake → spec → tasks → apply`

### Requirement: Context Table Row Order

The Tier 1 Context table SHALL render rows in this fixed order:

1. `| Type | {type} |`
2. `| Change | \`{change_name}\` |`
3. `| Confidence | {score} / 5.0 |`
4. `| Pipeline | {stages} |`
5. `| Intake | [{change_name}/intake.md]({url}) |`
6. `| Spec | [{change_name}/spec.md]({url}) |` *(omit if spec.md absent)*

#### Scenario: Complete Tier 1 table
- **GIVEN** a Tier 1 PR with all data available and `spec.md` present
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Context table has exactly 6 rows in the specified order (after header + separator)

### Requirement: No Changes to Tier 2 Template

The Tier 2 (lightweight) PR body template SHALL remain unchanged. It has no `.status.yaml` to read.

#### Scenario: Lightweight PR unchanged
- **GIVEN** a Tier 2 PR (docs/test/ci/chore)
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Context table contains only `| Type | {type} |` and the "No design artifacts" note

### Requirement: Read Status via File Access

The skill SHALL read `.status.yaml` fields by reading the file directly (already accessible after `changeman.sh resolve`). No new scripts or `stageman.sh` subcommands are required.

#### Scenario: Status file available
- **GIVEN** `changeman.sh resolve` succeeds and `.status.yaml` exists
- **WHEN** the skill reads confidence and progress fields
- **THEN** values are extracted from the YAML file without invoking additional scripts

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only Tier 1 template changes | Confirmed from intake #1 — Tier 2 has no `.status.yaml` | S:90 R:95 A:95 D:95 |
| 2 | Certain | Read fields directly from `.status.yaml` | Confirmed from intake #2 — file available after resolve | S:85 R:90 A:90 D:90 |
| 3 | Confident | Omit Spec row when absent rather than empty cell | Confirmed from intake #3 — cleaner, consistent | S:70 R:90 A:80 D:75 |
| 4 | Confident | Confidence displayed as `{score} / 5.0` | Confirmed from intake #4 — matches fab-status convention | S:75 R:90 A:85 D:80 |
| 5 | Confident | Pipeline uses `→` separator | Confirmed from intake #5 — compact and readable | S:65 R:95 A:70 D:65 |
| 6 | Certain | Intake/Spec as separate Field/Detail rows | Confirmed from intake #6 — user-requested fix | S:95 R:95 A:95 D:95 |
| 7 | Confident | Link text uses `{change_name}/{filename}` | Confirmed from intake #7 — more informative than bare label | S:60 R:95 A:75 D:70 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
