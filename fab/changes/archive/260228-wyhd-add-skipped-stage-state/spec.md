# Spec: Add "skipped" Stage State

**Change**: 260228-wyhd-add-skipped-stage-state
**Created**: 2026-02-28
**Affected memory**: `docs/memory/fab-workflow/schemas.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Skill-level skip orchestration — no `/fab-skip` command or automatic skip triggers in this change. The `skip` event is a low-level statusman primitive; skill-level convenience comes later.
- Modifying `/fab-archive` preconditions — archive already checks `hydrate: done` OR equivalent; the interaction between skipped hydrate and archive eligibility is noted but not implemented here (intake says "fully-skipped change is directly archivable" but the archive skill change is out of scope).

## Workflow Schema: Skipped State Definition

### Requirement: Schema declares `skipped` as a valid state

The workflow schema (`fab/.kit/schemas/workflow.yaml`) SHALL define a `skipped` state in the `states` list with the following properties:

- `id`: `skipped`
- `symbol`: `⏭`
- `description`: `"Stage intentionally bypassed"`
- `terminal`: `true`

#### Scenario: State definition is present

- **GIVEN** `fab/.kit/schemas/workflow.yaml` is loaded
- **WHEN** the `states` list is queried for `id: skipped`
- **THEN** the entry exists with `symbol: "⏭"`, `description: "Stage intentionally bypassed"`, `terminal: true`

### Requirement: `skipped` is an allowed state for all stages except intake

The `allowed_states` arrays SHALL be updated as follows:

- `intake`: unchanged — `[active, ready, done]`
- `spec`: `[pending, active, ready, done, skipped]`
- `tasks`: `[pending, active, ready, done, skipped]`
- `apply`: `[pending, active, ready, done, skipped]`
- `review`: `[pending, active, ready, done, failed, skipped]`
- `hydrate`: `[pending, active, ready, done, skipped]`

#### Scenario: Intake rejects skipped

- **GIVEN** a `.status.yaml` with `progress.intake: skipped`
- **WHEN** `validate_stage_state "intake" "skipped"` is called
- **THEN** it returns non-zero (validation failure)

#### Scenario: Non-intake stages accept skipped

- **GIVEN** a `.status.yaml` with `progress.spec: skipped`
- **WHEN** `validate_stage_state "spec" "skipped"` is called
- **THEN** it returns 0 (validation success)

## Workflow Schema: Skip Transition

### Requirement: `skip` event transitions `pending` or `active` to `skipped`

A new `skip` transition SHALL be added to the `transitions.default` section:

```yaml
- event: skip
  from: [pending, active]
  to: skipped
```

The `skip` event SHALL NOT be added to the `transitions.review` override — the default rule applies to all stages including review.

#### Scenario: Skip a pending stage

- **GIVEN** a change with `progress.spec: pending`
- **WHEN** `statusman.sh skip <change> spec` is executed
- **THEN** `progress.spec` becomes `skipped`

#### Scenario: Skip an active stage

- **GIVEN** a change with `progress.spec: active`
- **WHEN** `statusman.sh skip <change> spec` is executed
- **THEN** `progress.spec` becomes `skipped`
- **AND** downstream pending stages cascade to `skipped`

#### Scenario: Skip rejects non-pending/active stages

- **GIVEN** a change with `progress.spec: ready`
- **WHEN** `statusman.sh skip <change> spec` is executed
- **THEN** the command exits non-zero with an error: `"Cannot skip stage 'spec' — current state is 'ready', no valid transition"`

#### Scenario: Skip rejects done stages

- **GIVEN** a change with `progress.intake: done`
- **WHEN** `statusman.sh skip <change> intake` is executed
- **THEN** the command exits non-zero (intake doesn't allow `skipped` in `allowed_states`)

### Requirement: `reset` transition accepts `skipped` as a source state

The `reset` transition in `transitions.default` SHALL be updated to:

```yaml
- event: reset
  from: [done, ready, skipped]
  to: active
```

The `reset` transition in `transitions.review` SHALL be updated to:

```yaml
- event: reset
  from: [done, ready, skipped]
  to: active
```

#### Scenario: Reset a skipped stage

- **GIVEN** a change with `progress.spec: skipped`
- **WHEN** `statusman.sh reset <change> spec fab-continue` is executed
- **THEN** `progress.spec` becomes `active`
- **AND** all downstream stages (`tasks`, `apply`, `review`, `hydrate`) are set to `pending`

## Workflow Schema: Progression Rules

### Requirement: Progression rules treat `skipped` as resolved

The `progression.current_stage.rule` SHALL be updated to:

```
"First stage with state=active or state=ready; if no active/ready, first pending stage after last done or skipped; if all done/skipped, hydrate"
```

The `progression.next_stage.rule` SHALL be updated to:

```
"First stage in sequence where requires[] are all done or skipped and state=pending"
```

The `progression.completion.rule` SHALL be updated to:

```
"hydrate stage has state=done or state=skipped"
```

The `validation.prerequisites.rule` SHALL be updated to:

```
"Cannot set stage to active if requires[] stages are not done or skipped"
```

#### Scenario: Skipped stages don't block progression

- **GIVEN** a change with `progress.intake: done`, `progress.spec: skipped`, `progress.tasks: pending`
- **WHEN** `get_current_stage` is called
- **THEN** it returns `tasks`

#### Scenario: All non-intake stages skipped counts as complete

- **GIVEN** a change with `progress.intake: done`, all other stages `skipped`
- **WHEN** `get_current_stage` is called
- **THEN** it returns `hydrate` (the fallback for complete workflows)

## Statusman: `event_skip` Function

### Requirement: `event_skip` implements `{pending,active}` to `skipped` with forward cascade

The `event_skip` function SHALL:

1. Validate the status file exists and the stage is valid
2. Look up the `skip` transition via `lookup_transition`
3. Set the target stage to `skipped`
4. Call `_apply_metrics_side_effect` to clear metrics for the target stage
5. **Forward cascade**: iterate all stages after the target in pipeline order; for each stage whose current state is `pending`, set it to `skipped`
6. NOT auto-activate any next stage (unlike `event_finish`)
7. Update `last_updated` timestamp
8. Write atomically (tmpfile + mv pattern)

The function signature SHALL be: `event_skip <status_file> <stage> [driver]`

#### Scenario: Skip with forward cascade

- **GIVEN** a change with `progress.intake: done`, `progress.spec: pending`, `progress.tasks: pending`, `progress.apply: pending`, `progress.review: pending`, `progress.hydrate: pending`
- **WHEN** `event_skip` is called for `spec`
- **THEN** `progress.spec` becomes `skipped`
- **AND** `progress.tasks` becomes `skipped`
- **AND** `progress.apply` becomes `skipped`
- **AND** `progress.review` becomes `skipped`
- **AND** `progress.hydrate` becomes `skipped`

#### Scenario: Skip mid-pipeline cascades only downstream pending

- **GIVEN** a change with `progress.intake: done`, `progress.spec: done`, `progress.tasks: done`, `progress.apply: pending`, `progress.review: pending`, `progress.hydrate: pending`
- **WHEN** `event_skip` is called for `apply`
- **THEN** `progress.apply` becomes `skipped`
- **AND** `progress.review` becomes `skipped`
- **AND** `progress.hydrate` becomes `skipped`
- **AND** `progress.intake` remains `done`
- **AND** `progress.spec` remains `done`
- **AND** `progress.tasks` remains `done`

#### Scenario: Skip an active stage with cascade

- **GIVEN** a change with `progress.intake: done`, `progress.spec: active`, `progress.tasks: pending`, `progress.apply: pending`, `progress.review: pending`, `progress.hydrate: pending`
- **WHEN** `event_skip` is called for `spec`
- **THEN** `progress.spec` becomes `skipped`
- **AND** `progress.tasks` through `progress.hydrate` all become `skipped`

#### Scenario: Skip does not cascade to non-pending downstream stages

- **GIVEN** a change with stages in mixed states where some downstream stages are `done` or `active`
- **WHEN** `event_skip` is called for a pending or active stage
- **THEN** only downstream stages with `pending` state become `skipped`
- **AND** downstream stages with `done`, `active`, `ready`, or `failed` states are unchanged

## Statusman: CLI `skip` Subcommand

### Requirement: `skip` subcommand dispatches to `event_skip`

The CLI SHALL accept:

```
statusman.sh skip <change> <stage> [driver]
```

The subcommand SHALL:

1. Require at least 3 arguments (`skip`, `<change>`, `<stage>`) and accept an optional 4th (`[driver]`)
2. Resolve `<change>` via `resolve_to_status`
3. Dispatch to `event_skip`

The `--help` output SHALL include the new subcommand under "Event commands":

```
skip <change> <stage> [driver]            {pending,active} → skipped (+cascade)
```

#### Scenario: CLI invocation

- **GIVEN** a valid change `wyhd` with `progress.spec: pending`
- **WHEN** `statusman.sh skip wyhd spec fab-continue` is executed
- **THEN** `progress.spec` becomes `skipped` and downstream pending stages cascade to `skipped`

#### Scenario: CLI arg validation

- **GIVEN** `statusman.sh skip` is called with fewer than 3 arguments
- **WHEN** the command runs
- **THEN** it prints usage to stderr and exits 1

## Statusman: Updated Query Functions

### Requirement: `get_current_stage` treats `skipped` like `done` for progression

The fallback logic in `get_current_stage` SHALL find the first `pending` stage after the last `done` **or `skipped`** stage. Specifically:

1. First pass: find first `active` or `ready` stage (unchanged)
2. Fallback: track the last stage with state `done` or `skipped`, then find the first `pending` stage after it

#### Scenario: Current stage skips over skipped stages

- **GIVEN** `progress.intake: done`, `progress.spec: skipped`, `progress.tasks: skipped`, `progress.apply: pending`
- **WHEN** `get_current_stage` is called
- **THEN** it returns `apply`

### Requirement: `get_display_stage` treats `skipped` like `done` for display

The Tier 3 (last done) fallback in `get_display_stage` SHALL consider `skipped` alongside `done`. The "last done" tracking becomes "last done or skipped."

#### Scenario: Display stage shows last resolved stage

- **GIVEN** `progress.intake: done`, `progress.spec: skipped`, all others `pending`
- **WHEN** `get_display_stage` is called
- **THEN** it returns `spec:skipped`

### Requirement: `get_progress_line` renders `skipped` with `⏭` symbol

The `get_progress_line` function SHALL include a `skipped` case in the rendering loop:

```bash
skipped) parts+=("$stage ⏭") ;;
```

Skipped stages SHALL be included in the progress line output (not omitted like `pending`).

#### Scenario: Progress line includes skipped stages

- **GIVEN** `progress.intake: done`, `progress.spec: skipped`, `progress.tasks: done`, `progress.apply: active`
- **WHEN** `get_progress_line` is called
- **THEN** it outputs `intake → spec ⏭ → tasks → apply ⏳`

#### Scenario: All non-intake skipped shows completion marker

- **GIVEN** `progress.intake: done`, all other stages `skipped`
- **WHEN** `get_progress_line` is called
- **THEN** it outputs `intake → spec ⏭ → tasks ⏭ → apply ⏭ → review ⏭ → hydrate ⏭ ✓`
- **AND** the `✓` marker is appended because no stages are `active` or `pending`

### Requirement: `_apply_metrics_side_effect` deletes metrics for `skipped` state

The `_apply_metrics_side_effect` function SHALL handle the `skipped` case by deleting the stage's metrics (same as `pending` — nothing happened):

```bash
skipped)
  yq -i "del(.stage_metrics.${stage})" "$tmpfile"
  ;;
```

#### Scenario: Metrics deleted on skip

- **GIVEN** a stage with existing `stage_metrics` from a prior active phase
- **WHEN** the stage transitions to `skipped` via reset→pending→skip path
- **THEN** the `stage_metrics` entry for that stage is removed

## Design Decisions

1. **Forward cascade on skip mirrors backward cascade on reset**
   - *Why*: Both are directional state propagation. Reset goes backward (downstream → pending); skip goes forward (downstream pending → skipped). The symmetry makes the mental model consistent: skip "resolves" everything ahead as intentionally bypassed.
   - *Rejected*: Per-stage skip without cascade — forces the user to skip each stage individually, tedious and error-prone for the "intake-only" use case.

2. **`skipped` is terminal like `done`**
   - *Why*: Both represent resolved end states. A skipped stage is intentionally complete (by decision, not by work). Requiring explicit `reset` to un-skip prevents accidental re-activation.
   - *Rejected*: Non-terminal `skipped` — would allow implicit transitions, undermining the explicit intent signal.

3. **Skip allowed from `active` but not `ready`**
   - *Why*: `active` means "work in progress" — the user may realize mid-stage they don't need it. Without this, there's no clean path from `active` to `skipped` (no transition leads back to `pending`). `ready` means an artifact exists — discarding it should be a deliberate two-step (`reset` → `active` → `skip`) to acknowledge the thrown-away work.
   - *Rejected*: Skip from all non-terminal states including `ready` — too easy to accidentally discard completed artifacts. Skip only from `pending` — leaves no escape from `active` to `skipped`.

4. **No metrics for skipped stages**
   - *Why*: No work was performed — no `started_at`, no `completed_at`, no `driver`, no `iterations` to record. Deleting metrics (like `pending`) keeps the metrics block clean.
   - *Rejected*: Recording a `skipped_at` timestamp — adds complexity for minimal value; the `last_updated` on `.status.yaml` already captures when the skip happened.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `skip` event from `pending` or `active` states | Revised from intake #1 — skipping an active stage is valid ("started but don't need it"); `ready` excluded (deliberate reset-then-skip) | S:90 R:85 A:90 D:85 |
| 2 | Certain | Intake cannot be skipped | Confirmed from intake #2 — intake is the entry point, always required | S:95 R:85 A:95 D:95 |
| 3 | Certain | `skip` cascades downstream pending stages to `skipped` | Confirmed from intake #3 — user agreed with cascade over per-stage | S:95 R:85 A:90 D:90 |
| 4 | Certain | No auto-activate after skip | Confirmed from intake #4 — skip is terminal intent, not "move to next" | S:95 R:90 A:90 D:95 |
| 5 | Certain | Skipped stages satisfy prerequisites (like `done`) | Confirmed from intake #5 — skipped = resolved for progression | S:90 R:85 A:90 D:90 |
| 6 | Certain | `reset` from `skipped` → `active` with normal downstream cascade to `pending` | Confirmed from intake #6 — reuse existing reset mechanics for un-skip | S:90 R:90 A:90 D:95 |
| 7 | Certain | Symbol for skipped is `⏭` | Confirmed from intake #8 — consistent with emoji-style state symbols | S:80 R:95 A:90 D:85 |
| 8 | Confident | `skipped` state has `terminal: true` in schema | Confirmed from intake #9 — analogous to `done`, resettable via explicit reset | S:80 R:90 A:85 D:80 |
| 9 | Confident | No metrics for skipped stages (delete like `pending`) | Confirmed from intake #10 — nothing happened, no timing data to record | S:75 R:90 A:85 D:85 |
| 10 | Confident | Forward cascade only affects `pending` downstream stages | Codebase signal — `event_reset` cascade sets all downstream to `pending` regardless of state, but skip cascade should be conservative: only change `pending` stages to avoid overwriting `done`/`active` work | S:80 R:75 A:85 D:75 |
| 11 | Certain | Archive eligibility with skipped stages is out of scope | Intake mentions it but explicitly says user-flow.md is already updated; the archive precondition change is separate | S:90 R:90 A:90 D:90 |

11 assumptions (9 certain, 2 confident, 0 tentative, 0 unresolved).
