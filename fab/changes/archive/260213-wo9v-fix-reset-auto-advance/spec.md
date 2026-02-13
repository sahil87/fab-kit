# Spec: Fix reset flow to stop at target stage

**Change**: 260213-wo9v-fix-reset-auto-advance
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/change-lifecycle.md`

## Non-Goals

- Changing the normal (non-reset) two-write stage transition — forward progression still uses `current: done` + `next: active` atomically
- Deleting downstream artifact files on reset — existing files stay in place, only their progress state changes to `pending` (already documented behavior)
- Adding a new state to the vocabulary — the fix uses the existing `pending`/`active`/`done`/`failed` states

## Fab Workflow: Reset Flow Stop Behavior

### Requirement: Reset SHALL stop at the target stage

When `/fab-continue` is invoked with a stage argument (reset flow), the target stage's artifact SHALL be regenerated and its progress set to `done`. The next stage in sequence SHALL remain `pending` — the reset flow MUST NOT auto-advance by setting the next stage to `active`.

Downstream stages (all stages after the target) SHALL be set to `pending`. Stages before the target SHALL be preserved as-is.

#### Scenario: Reset to spec from apply stage
- **GIVEN** a change with `brief: done`, `spec: done`, `tasks: done`, `apply: active`
- **WHEN** the user runs `/fab-continue spec`
- **THEN** spec.md is regenerated
- **AND** `.status.yaml` progress becomes: `brief: done`, `spec: done`, `tasks: pending`, `apply: pending`, `review: pending`, `archive: pending`
- **AND** no stage has `active` state

#### Scenario: Reset to brief
- **GIVEN** a change at any stage
- **WHEN** the user runs `/fab-continue brief`
- **THEN** brief.md is regenerated
- **AND** `.status.yaml` progress becomes: `brief: done`, all others `pending`
- **AND** no stage has `active` state

#### Scenario: Reset to tasks
- **GIVEN** a change with `brief: done`, `spec: done`, `tasks: done`, `apply: active`
- **WHEN** the user runs `/fab-continue tasks`
- **THEN** tasks.md is regenerated, task checkboxes are reset, checklist is regenerated
- **AND** `.status.yaml` progress becomes: `brief: done`, `spec: done`, `tasks: done`, `apply: pending`, `review: pending`, `archive: pending`
- **AND** no stage has `active` state

### Requirement: `/fab-continue` guard SHALL dispatch on preflight's derived stage

`/fab-continue` SHALL use the `stage` field from preflight output for its guard dispatch, regardless of whether that stage was found via an `active` entry or the pending-after-done fallback. When the derived stage's progress is `pending` (no `active` entry exists), `/fab-continue` SHALL first set it to `active`, then proceed with normal stage execution.
<!-- clarified: /fab-continue guard uses preflight's derived stage, sets pending→active before dispatch -->

This means the guard table in `fab-continue.md` does NOT need new rows — the existing dispatch logic works as-is because preflight's `stage` field already resolves to the correct stage name. The only addition is a pre-guard step: "if derived stage is `pending`, set it to `active`."

#### Scenario: Continue after spec reset
- **GIVEN** a change with `brief: done`, `spec: done`, `tasks: pending` (result of a spec reset)
- **WHEN** the user runs `/fab-continue`
- **THEN** preflight derives `tasks` as the current stage (first pending after last done)
- **AND** `/fab-continue` sets `tasks: active` in `.status.yaml`
- **AND** dispatches to the `spec` guard row (generate tasks.md), executing normally

#### Scenario: Continue when guard would previously block
- **GIVEN** a change with `brief: done`, all others `pending`, no `active` entry
- **WHEN** the user runs `/fab-continue`
- **THEN** preflight derives `spec` as the current stage
- **AND** `/fab-continue` sets `spec: active`, then generates spec.md
- **AND** does NOT output "Change is complete"

## Fab Workflow: Stage Derivation Fallback

### Requirement: Fallback SHALL find first pending stage after last done

When no stage has `active` state in the progress map, the stage derivation logic SHALL NOT unconditionally fall back to `archive`. Instead, it SHALL:

1. Walk stages in order and find the last stage with `done` state
2. Return the first `pending` stage after it as the derived "current stage"
3. Fall back to `archive` ONLY when all stages are `done` (workflow complete)

This logic applies identically in three locations: `fab-preflight.sh`, `fab-status.sh`, and `stageman.sh` (`get_current_stage` function).

#### Scenario: Brief done, no active stage (post-reset or initial transition gap)
- **GIVEN** `.status.yaml` with `brief: done`, `spec: pending`, `tasks: pending`, `apply: pending`, `review: pending`, `archive: pending`
- **WHEN** the preflight script or status script derives the current stage
- **THEN** the derived stage is `spec` (first pending after last done)

#### Scenario: Spec and brief done, no active stage
- **GIVEN** `.status.yaml` with `brief: done`, `spec: done`, `tasks: pending`, `apply: pending`, `review: pending`, `archive: pending`
- **WHEN** the stage derivation runs
- **THEN** the derived stage is `tasks`

#### Scenario: All stages done (workflow complete)
- **GIVEN** `.status.yaml` with all stages set to `done`
- **WHEN** the stage derivation runs
- **THEN** the derived stage is `archive` (existing behavior preserved)

#### Scenario: Review failed, apply re-activated
- **GIVEN** `.status.yaml` with `review: failed`, `apply: active`
- **WHEN** the stage derivation runs
- **THEN** the derived stage is `apply` (found via the `active` entry — fallback not triggered)

### Requirement: Schema SHALL document the updated fallback rule

`workflow.yaml` `progression.current_stage` SHALL be updated to reflect the new fallback behavior: "First stage with state=active; if no active, first pending stage after last done stage; if all done, archive."

#### Scenario: Schema rule matches implementation
- **GIVEN** the `workflow.yaml` schema
- **WHEN** a developer reads `progression.current_stage.rule`
- **THEN** the rule text describes the three-tier fallback (active → first-pending-after-done → archive)

## Fab Workflow: Status Display Consistency

### Requirement: `/fab-status` next command SHALL handle no-active state

The next-command suggestion in `fab-status.sh` SHALL produce a meaningful suggestion when the derived stage comes from the pending-after-done fallback (no `active` entry). The `case` statement SHALL include `{stage}:pending` entries for each planning stage that suggest `/fab-continue`.
<!-- clarified: fab-status adds explicit pending cases to the case statement -->

Specifically, these cases SHALL be added:
- `brief:pending` → `/fab-continue or /fab-clarify`
- `spec:pending` → `/fab-continue or /fab-clarify`
- `tasks:pending` → `/fab-continue`
- `apply:pending` → `/fab-continue`
- `review:pending` → `/fab-continue`

#### Scenario: Status display after spec reset
- **GIVEN** a change with `brief: done`, `spec: done`, `tasks: pending` (no active stage)
- **WHEN** the user runs `/fab-status`
- **THEN** the stage is displayed as `tasks (3/6)`
- **AND** the next command suggestion is `/fab-continue`

#### Scenario: Status display after brief done with no active
- **GIVEN** a change with `brief: done`, `spec: pending` (no active stage)
- **WHEN** the user runs `/fab-status`
- **THEN** the stage is displayed as `spec (2/6)`
- **AND** the next command suggestion is `/fab-continue or /fab-clarify`

## Design Decisions

1. **Stop-at-target, not stop-before-target**: After regeneration, the target stage is marked `done`, not left as `active`.
   - *Why*: The artifact has been regenerated successfully — marking it `done` accurately reflects that the work is complete. Leaving it `active` would require the user to run `/fab-continue` just to re-mark it `done`, adding a useless step.
   - *Rejected*: Leaving target as `active` — creates a confusing state where the artifact exists and is fresh but the stage says "in progress."

2. **Three-tier fallback in derivation, not a new status field**: The fix updates the fallback logic rather than adding a `next_stage:` field to `.status.yaml`.
   - *Why*: The progress map already contains all the information needed. Adding a field creates a second source of truth that can go stale. The derivation logic is simple (walk stages, find first pending after last done) and mirrors the existing "find first active" walk.
   - *Rejected*: Adding `next_stage:` to `.status.yaml` — second source of truth, stale risk. Adding a `ready` state — expands the state vocabulary for a narrow edge case.

3. **Fix in all three scripts, not just one**: The fallback logic is duplicated in `fab-preflight.sh`, `fab-status.sh`, and `stageman.sh`. All three MUST be updated.
   - *Why*: `stageman.sh` has `get_current_stage()` which should be the canonical implementation, but `fab-preflight.sh` and `fab-status.sh` have inline fallback logic that doesn't call `get_current_stage()`. All three need the fix for consistency. Refactoring to deduplicate is out of scope for this change.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Not refactoring inline fallback to call `get_current_stage()` | Deduplication is a separate concern; the three scripts already have independent fallback logic. Consolidation would be a separate change. |

1 assumption made (1 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-13

- **Q**: How should `/fab-continue`'s guard dispatch handle the "no active entry but stages not all done" case after the fallback fix?
  **A**: Use derived stage from preflight — `/fab-continue` dispatches on preflight's `stage` field regardless of whether it came from an `active` entry or the pending-after-done fallback. Sets pending→active before dispatch.
- **Q**: Should `fab-status.sh` add `{stage}:pending` cases to the next-command case statement?
  **A**: Yes, add explicit `pending` cases for each stage that suggest `/fab-continue`.
