# Spec: Fix stage guard to check progress value instead of stage name

**Change**: 260210-0p4e-fix-stage-guard-progress-check
**Created**: 2026-02-10
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`

## Fab Workflow: Stage Guard Logic

### Requirement: Guard SHALL Check Progress Value

The `/fab-continue` stage guard logic (Step 1: Determine Current Stage, Normal Flow section) SHALL check the `progress` map value for the current stage, not just the `stage` field name, when determining whether to allow continuation.

#### Scenario: Tasks Active — Allow Resume

- **GIVEN** a change with `stage: tasks` and `progress.tasks: active` (generation interrupted mid-way)
- **WHEN** `/fab-continue` is invoked with no argument
- **THEN** the guard SHALL allow task generation to resume
- **AND** SHALL NOT block with "Planning is complete. Run /fab-apply"

#### Scenario: Tasks Done — Block

- **GIVEN** a change with `stage: tasks` and `progress.tasks: done` (generation complete)
- **WHEN** `/fab-continue` is invoked with no argument
- **THEN** the guard SHALL block with "Planning is complete. Run /fab-apply to begin implementation."

#### Scenario: Specs Active — Allow Resume

- **GIVEN** a change with `stage: specs` and `progress.specs: active` (generation interrupted)
- **WHEN** `/fab-continue` is invoked with no argument
- **THEN** the guard SHALL allow spec generation to resume
- **AND** SHALL NOT block

#### Scenario: Apply or Later — Block Regardless

- **GIVEN** a change with `stage: apply` (or review, archive) and `progress.apply: active` or `done`
- **WHEN** `/fab-continue` is invoked with no argument
- **THEN** the guard SHALL block with "Implementation is underway. Use /fab-apply, /fab-review, or /fab-archive as appropriate."
- **AND** this behavior is unchanged (apply/review/archive stages never resume via /fab-continue)

### Requirement: Guard Logic SHALL Use Preflight Output

The guard logic SHALL use the `progress` map from `fab-preflight.sh` stdout (parsed in Pre-flight Check step), not re-read `.status.yaml` directly, to determine the current stage's progress value.

#### Scenario: Consistent Progress Source

- **GIVEN** `/fab-continue` has run the preflight script and parsed its YAML output
- **WHEN** the guard logic needs to check progress values
- **THEN** it SHALL use the `progress` map from preflight output (already parsed)
- **AND** SHALL NOT re-read `.status.yaml` separately

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Fix applies only to fab-continue.md Normal Flow guard | Other skills like /fab-ff already use progress-based resumability checks; the bug is specific to /fab-continue's guard conditions |
| 2 | Confident | Progress value check distinguishes `done` from `active` | The progress map uses these two values to distinguish completed vs interrupted states (plus `pending` for not-yet-started); checking `== 'done'` is the correct gate condition |

2 assumptions made (2 confident, 0 tentative).
