# Spec: Event-Driven Stageman

**Change**: 260226-6boq-event-driven-stageman
**Created**: 2026-02-26
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/schemas.md`, `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/planning-skills.md`

## Non-Goals

- Changing the state vocabulary (pending, active, ready, done, failed remain as-is)
- Changing the stage pipeline order or adding/removing stages
- Modifying non-stage-state write commands (set-change-type, set-checklist, set-confidence, ship, etc.)
- Changing read-only query commands (all-stages, progress-map, current-stage, display-stage, etc.)
- Adding prerequisite ordering validation (e.g., blocking `start spec` when intake is pending) — this is a future enhancement, not part of this change

## Stageman: Event Commands

### Requirement: Five Event CLI Commands

Stageman SHALL expose exactly 5 event commands that replace `set-state` and `transition`:

```
stageman.sh start   <change> <stage> [driver]
stageman.sh advance <change> <stage> [driver]
stageman.sh finish  <change> <stage> [driver]
stageman.sh reset   <change> <stage> [driver]
stageman.sh fail    <change> <stage> [driver]
```

Each command SHALL:
1. Resolve `<change>` to a `.status.yaml` path (see Change Identifier Resolution)
2. Read the current state of `<stage>` from `.status.yaml`
3. Look up the transition rule for the event + current_state pair in `workflow.yaml`
4. If the transition is legal, apply the new state atomically (temp file + mv)
5. If the transition is illegal, exit 1 with a diagnostic error to stderr

The state transition table SHALL be:

| From State | Event   | To State     | Scope       |
|------------|---------|--------------|-------------|
| pending    | start   | active       | default     |
| failed     | start   | active       | review only |
| active     | advance | ready        | default     |
| active     | finish  | done (+next) | default     |
| ready      | finish  | done (+next) | default     |
| ready      | reset   | active       | default     |
| done       | reset   | active       | default     |
| active     | fail    | failed       | review only |

All other (current_state, event) combinations SHALL be rejected.

#### Scenario: Start from pending
- **GIVEN** stage `spec` has state `pending`
- **WHEN** `stageman.sh start <change> spec fab-continue` is invoked
- **THEN** `spec` state becomes `active`
- **AND** `stage_metrics.spec` is created with `started_at`, `driver: fab-continue`, `iterations: 1`

#### Scenario: Start from failed (review only)
- **GIVEN** stage `review` has state `failed`
- **WHEN** `stageman.sh start <change> review fab-continue` is invoked
- **THEN** `review` state becomes `active`
- **AND** `stage_metrics.review.iterations` is incremented (not reset)

#### Scenario: Start from failed on non-review stage rejected
- **GIVEN** stage `spec` has state `pending` (never reaches `failed` — `failed` is not in `spec.allowed_states`)
- **WHEN** `stageman.sh start <change> spec` is invoked from any non-`pending` state
- **THEN** the command exits 1 with diagnostic error

#### Scenario: Advance from active
- **GIVEN** stage `spec` has state `active`
- **WHEN** `stageman.sh advance <change> spec` is invoked
- **THEN** `spec` state becomes `ready`
- **AND** stage_metrics are unchanged (advance is a no-op on metrics)

#### Scenario: Advance from non-active rejected
- **GIVEN** stage `spec` has state `pending`
- **WHEN** `stageman.sh advance <change> spec` is invoked
- **THEN** the command exits 1 with error: `ERROR: Cannot advance stage 'spec' — current state is 'pending', expected 'active'`

#### Scenario: Finish from active
- **GIVEN** stage `spec` has state `active`
- **AND** stage `tasks` has state `pending`
- **WHEN** `stageman.sh finish <change> spec fab-continue` is invoked
- **THEN** `spec` state becomes `done`
- **AND** `tasks` state becomes `active` (implicit start of next stage)
- **AND** `stage_metrics.spec.completed_at` is set
- **AND** `stage_metrics.tasks` is created with `started_at`, `driver: fab-continue`, `iterations: 1`

#### Scenario: Finish from ready
- **GIVEN** stage `spec` has state `ready`
- **AND** stage `tasks` has state `pending`
- **WHEN** `stageman.sh finish <change> spec fab-continue` is invoked
- **THEN** `spec` state becomes `done`
- **AND** `tasks` state becomes `active` (implicit start of next stage)
- **AND** same metrics side-effects as finishing from active

#### Scenario: Finish last stage (hydrate)
- **GIVEN** stage `hydrate` has state `active`
- **WHEN** `stageman.sh finish <change> hydrate fab-continue` is invoked
- **THEN** `hydrate` state becomes `done`
- **AND** no next stage is activated (hydrate is the last stage)
- **AND** `stage_metrics.hydrate.completed_at` is set

`reset` has an atomic cascade side-effect: when stage N is reset, all stages after N are set to `pending` and their `stage_metrics` entries are removed. This enables the `/fab-continue <stage>` reset flow without needing `set-state` — a single `reset` command handles the full multi-stage rollback. Stages before N are preserved.

#### Scenario: Finish when next stage is not pending
- **GIVEN** stage `spec` has state `active`
- **AND** stage `tasks` has state `done` (e.g., after a reset cycle)
- **WHEN** `stageman.sh finish <change> spec fab-continue` is invoked
- **THEN** `spec` state becomes `done`
- **AND** `tasks` state is NOT changed (already `done`, not `pending`)

#### Scenario: Reset from done
- **GIVEN** stage `spec` has state `done`
- **AND** stages `tasks`, `apply`, `review`, `hydrate` have various states
- **WHEN** `stageman.sh reset <change> spec fab-continue` is invoked
- **THEN** `spec` state becomes `active`
- **AND** all stages after `spec` (`tasks`, `apply`, `review`, `hydrate`) are set to `pending`
- **AND** `stage_metrics.spec` is recreated with new `started_at`, `driver`, incremented `iterations`
- **AND** `stage_metrics` entries for downstream stages are removed

#### Scenario: Reset from ready
- **GIVEN** stage `spec` has state `ready`
- **AND** no downstream stages have been started yet (all `pending`)
- **WHEN** `stageman.sh reset <change> spec fab-continue` is invoked
- **THEN** `spec` state becomes `active`
- **AND** downstream stages remain `pending` (cascade is a no-op since they're already pending)
- **AND** `stage_metrics.spec` is recreated with new `started_at`, `driver`, incremented `iterations`

#### Scenario: Reset cascade from deep in pipeline
- **GIVEN** stage `apply` has state `active`
- **AND** stages `intake`, `spec`, `tasks` are `done`
- **AND** stages `review`, `hydrate` are `pending`
- **WHEN** `stageman.sh reset <change> spec fab-continue` is invoked
- **THEN** `spec` state becomes `active`
- **AND** `tasks` state becomes `pending`
- **AND** `apply` state becomes `pending`
- **AND** `review` and `hydrate` remain `pending`
- **AND** `stage_metrics` for `tasks` and `apply` are removed
- **AND** `intake` state is preserved as `done` (before target, not affected)

#### Scenario: Fail from active (review only)
- **GIVEN** stage `review` has state `active`
- **WHEN** `stageman.sh fail <change> review` is invoked
- **THEN** `review` state becomes `failed`
- **AND** stage_metrics are unchanged (fail is a no-op on metrics)

#### Scenario: Fail on non-review stage rejected
- **GIVEN** stage `spec` has state `active`
- **WHEN** `stageman.sh fail <change> spec` is invoked
- **THEN** the command exits 1 with error: `ERROR: Event 'fail' is not valid for stage 'spec' — fail is only allowed on review`

### Requirement: Change Identifier Resolution

All stageman commands (event and non-event) SHALL accept either a raw file path or a change identifier as the first positional argument after the subcommand.

Resolution logic:
1. If the argument is an existing file path, use it directly
2. Otherwise, resolve via `changeman.sh resolve "$arg"`, then construct `fab/changes/{resolved-name}/.status.yaml`

This eliminates path construction boilerplate from every caller.

#### Scenario: File path argument
- **GIVEN** the file `fab/changes/260226-6boq-event-driven-stageman/.status.yaml` exists
- **WHEN** `stageman.sh start fab/changes/260226-6boq-event-driven-stageman/.status.yaml spec fab-continue` is invoked
- **THEN** the command operates on that exact file

#### Scenario: Change ID shortcut (4-char)
- **GIVEN** change `260226-6boq-event-driven-stageman` exists
- **WHEN** `stageman.sh start 6boq spec fab-continue` is invoked
- **THEN** stageman resolves `6boq` via `changeman.sh resolve` to `260226-6boq-event-driven-stageman`
- **AND** operates on `fab/changes/260226-6boq-event-driven-stageman/.status.yaml`

#### Scenario: Partial slug shortcut
- **GIVEN** change `260226-6boq-event-driven-stageman` exists
- **WHEN** `stageman.sh start event-driven spec fab-continue` is invoked
- **THEN** stageman resolves `event-driven` via `changeman.sh resolve`
- **AND** operates on the resolved change's `.status.yaml`

#### Scenario: Resolution failure
- **GIVEN** no change matches `nonexistent`
- **WHEN** `stageman.sh start nonexistent spec` is invoked
- **THEN** the command exits 1 with the error from `changeman.sh resolve`

#### Scenario: Non-event commands also resolve
- **GIVEN** change `260226-6boq-event-driven-stageman` exists
- **WHEN** `stageman.sh progress-map 6boq` is invoked
- **THEN** stageman resolves `6boq` and outputs the progress map from the resolved file

### Requirement: Driver Parameter

The `[driver]` parameter SHALL be optional on all 5 event commands. When provided, it is recorded in `stage_metrics` for traceability. When omitted, the driver field is left empty in metrics. No validation, no error on omission.

Skills are instructed to always pass the driver; manual/test invocations omit it.

#### Scenario: Driver provided
- **GIVEN** stage `spec` has state `pending`
- **WHEN** `stageman.sh start <change> spec fab-continue` is invoked
- **THEN** `stage_metrics.spec.driver` is set to `fab-continue`

#### Scenario: Driver omitted
- **GIVEN** stage `spec` has state `pending`
- **WHEN** `stageman.sh start <change> spec` is invoked (no driver)
- **THEN** `stage_metrics.spec.driver` is set to empty string
- **AND** no error is raised

### Requirement: Stage Metrics Side-Effects

Event commands SHALL apply stage_metrics side-effects consistent with the current implementation:

| Event   | Metrics Effect |
|---------|----------------|
| start   | Create/update entry: `started_at` = now, `driver` = arg, `iterations` += 1 |
| advance | No-op (preserve existing metrics from active phase) |
| finish  | Set `completed_at` = now on finished stage. If next stage activated: create entry on next stage |
| reset   | Create/update entry on target stage (same as start — `started_at`, `driver`, `iterations` += 1). Remove `completed_at` if present. **Cascade**: remove `stage_metrics` entries for all downstream stages (set to `pending`) |
| fail    | No-op (preserve timing data from active phase) |

#### Scenario: Start increments iterations
- **GIVEN** stage `review` was previously completed (iterations: 1) then reset
- **WHEN** `stageman.sh start <change> review fab-continue` is invoked
- **THEN** `stage_metrics.review.iterations` is `2` (incremented, not reset)

#### Scenario: Finish sets completed_at and activates next
- **GIVEN** stage `spec` is `active`
- **WHEN** `stageman.sh finish <change> spec fab-continue` is invoked
- **THEN** `stage_metrics.spec.completed_at` is set to current timestamp
- **AND** `stage_metrics.tasks.started_at` is set to current timestamp
- **AND** `stage_metrics.tasks.iterations` is `1`

## Stageman: Schema

### Requirement: Event-Keyed Transitions Format

The `transitions` section in `workflow.yaml` SHALL be restructured from `from→to` format to event-keyed format:

```yaml
transitions:
  default:
    - event: start
      from: [pending]
      to: active

    - event: advance
      from: [active]
      to: ready

    - event: finish
      from: [active, ready]
      to: done

    - event: reset
      from: [done, ready]
      to: active

  review:
    - event: start
      from: [pending, failed]
      to: active

    - event: advance
      from: [active]
      to: ready

    - event: finish
      from: [active, ready]
      to: done

    - event: reset
      from: [done, ready]
      to: active

    - event: fail
      from: [active]
      to: failed
```

The `condition` field is removed — transition legality is fully determined by the `(current_state, event)` pair.

#### Scenario: Schema parsed for transition lookup
- **GIVEN** workflow.yaml contains the event-keyed transitions above
- **WHEN** stageman looks up event `finish` for stage `spec` with current state `ready`
- **THEN** it finds the `default` section (spec has no override), matches `event: finish` with `from: [active, ready]`, and resolves `to: done`

#### Scenario: Review-specific override
- **GIVEN** workflow.yaml contains the `review` override section
- **WHEN** stageman looks up event `fail` for stage `review` with current state `active`
- **THEN** it finds the `review` section, matches `event: fail` with `from: [active]`, and resolves `to: failed`

#### Scenario: No matching transition
- **GIVEN** workflow.yaml contains the event-keyed transitions
- **WHEN** stageman looks up event `advance` for stage `spec` with current state `pending`
- **THEN** no match is found (pending is not in `from: [active]`), and the command is rejected

## Stageman: Removed Commands

### Deprecated: set-state

**Reason**: Replaced by the 5 event commands. `set-state` allows arbitrary state writes that bypass the transition graph.
**Migration**: Map each `set-state` call to the appropriate event command:
- `set-state <f> <stage> active <driver>` → `start <change> <stage> <driver>`
- `set-state <f> <stage> done` → `finish <change> <stage> <driver>`
- `set-state <f> <stage> ready` → `advance <change> <stage>`
- `set-state <f> <stage> pending` → no direct replacement (reset flow handles this internally)
- `set-state <f> review failed` → `fail <change> review`

### Deprecated: transition

**Reason**: Replaced by `finish`, which subsumes `transition`'s role and additionally handles the `ready` from-state.
**Migration**: `transition <f> <from> <to> <driver>` → `finish <change> <from> <driver>`

## Skills: Event API Migration

### Requirement: All Skill Files Updated

All skill files that currently call `stageman.sh set-state` or `stageman.sh transition` SHALL be updated to use the event commands. The mapping is:

| Current call pattern | New call pattern |
|---------------------|-----------------|
| `set-state <f> intake active fab-new` | `start <f> intake fab-new` |
| `set-state <f> <stage> active <driver>` | `start <change> <stage> <driver>` |
| `set-state <f> <stage> done` | `finish <change> <stage> <driver>` |
| `set-state <f> <stage> ready` | `advance <change> <stage>` |
| `set-state <f> review failed` | `fail <change> review` |
| `set-state <f> <stage> pending` | (handled internally by reset flow) |
| `transition <f> <from> <to> <driver>` | `finish <change> <from> <driver>` |

With the change-ID shortcut, skills MAY pass the change identifier directly instead of constructing file paths — except when the caller already has the full path (e.g., `changeman.sh` which constructs the path internally).

#### Scenario: changeman.sh updated
- **GIVEN** `changeman.sh` contains `set-state "$status_file" intake active fab-new`
- **WHEN** the migration is applied
- **THEN** it becomes `start "$status_file" intake fab-new` (keeps raw path since changeman already has it)

#### Scenario: fab-continue.md updated
- **GIVEN** `fab-continue.md` references `stageman.sh set-state` and `stageman.sh transition`
- **WHEN** the migration is applied
- **THEN** all references use event commands: `start`, `advance`, `finish`, `reset`, `fail`
- **AND** instructions reference `<change>` instead of `<file>` where the skill has the change identifier

#### Scenario: fab-ff.md and fab-fff.md updated
- **GIVEN** `fab-ff.md` and `fab-fff.md` reference `stageman.sh set-state` and `stageman.sh transition`
- **WHEN** the migration is applied
- **THEN** all references use event commands with the same mapping

### Requirement: SPEC-changeman.md Updated

The changeman spec file SHALL be updated to reference the new event API instead of `set-state`.

#### Scenario: SPEC reference updated
- **GIVEN** `src/lib/changeman/SPEC-changeman.md` references `set-state`
- **WHEN** the migration is applied
- **THEN** the reference uses the equivalent event command

## Tests: Event-Based Test Suite

### Requirement: Comprehensive Event-Based Tests

`src/lib/stageman/test.bats` SHALL be rewritten to test the event-based API. Tests for `set-state` and `transition` SHALL be removed and replaced with:

1. **Happy path per event**: Each event from each valid from-state produces the correct to-state
2. **Rejection per event**: Each event from each invalid from-state exits 1 with diagnostic error
3. **`finish` side-effect**: Finishing stage N atomically activates stage N+1 when N+1 is pending; finishing hydrate marks done with no side-effect; finishing when next stage is not pending does not change next stage
4. **`reset` cascade**: Resetting stage N sets all downstream stages to `pending` and removes their `stage_metrics`; stages before N are preserved; resetting intake cascades all 5 downstream stages; resetting hydrate (last stage) has no cascade
5. **Review-specific**: `fail` only works on review stage; `start` from `failed` only works on review stage
5. **Stage metrics**: `start` increments iterations and sets started_at; `finish` sets completed_at and creates metrics on next; `reset` recreates metrics with incremented iterations; `advance` and `fail` are no-ops on metrics
6. **Change-ID resolution**: Event commands accept change identifiers (partial slug, 4-char ID) and resolve to correct `.status.yaml` path
7. **Error cases**: Missing file, invalid stage name, invalid event name

#### Scenario: Happy path test coverage
- **GIVEN** the test suite runs
- **WHEN** each (valid_from_state, event) pair is tested
- **THEN** the resulting state matches the transition table
- **AND** all 8 valid transitions are covered

#### Scenario: Rejection test coverage
- **GIVEN** the test suite runs
- **WHEN** each (invalid_from_state, event) pair is tested
- **THEN** stageman exits 1 with a diagnostic error message
- **AND** `.status.yaml` is unchanged

## Memory: Documentation Updates

### Requirement: Memory Files Updated

The following memory files SHALL be updated to reflect the event-driven API:

1. **`change-lifecycle.md`**: Add the state transition table. Update "Two-write transitions" to reference `finish` instead of `transition`. Update `stage_metrics` description to reference event commands instead of `set_stage_state` and `transition_stages`. Remove references to `set-state` and `transition` CLI commands.

2. **`schemas.md`**: Update the transitions section description to reference the event-keyed format. Note that transitions are now keyed by event name rather than from→to pairs.

3. **`execution-skills.md`**: Update "Status mutations" overview to reference event commands instead of `transition`/`set-state`. Update all stageman CLI references throughout.

4. **`planning-skills.md`**: Update "Shared Generation Partial" notes about `set-state`/`transition` calls. Update all stage transition references to use event commands.

#### Scenario: State transition table in change-lifecycle.md
- **GIVEN** the hydrate stage completes
- **WHEN** `change-lifecycle.md` is updated
- **THEN** it contains the complete 8-row state transition table with FROM STATE, EVENT, TO STATE, and REVIEW ONLY columns

#### Scenario: execution-skills.md status mutations
- **GIVEN** the hydrate stage completes
- **WHEN** `execution-skills.md` is updated
- **THEN** the "Status mutations" paragraph references `start`, `advance`, `finish`, `reset`, `fail` instead of `transition` and `set-state`
- **AND** all inline CLI examples use event commands

## Design Decisions

1. **Event-driven over validated-setter**: The event-driven pattern (trigger event → machine resolves state) is chosen over the validated-setter pattern (caller names target state → machine validates). The event-driven pattern centralizes all transition logic in stageman — callers don't need to know which state transitions are valid, only which semantic event occurred.
   - *Why*: Eliminates distributed state machine logic across ~10 skill files. The `set-state` backdoor allowed any skill to write any allowed state, defeating the transition graph.
   - *Rejected*: Keeping `set-state` as a backstop with `transition` as convenience — preserves the bypass that caused the problem.

2. **`finish` subsumes `transition` with ready-state support**: The `finish` event replaces both `transition` and `set-state <stage> done`. It works from both `active` and `ready` states, and atomically activates the next pending stage.
   - *Why*: `transition` was hardcoded to require `active` as from-state. After adding `ready`, advancing a ready stage required two raw `set-state` calls. `finish` handles both cases in one command.
   - *Rejected*: Extending `transition` to accept `ready` — doesn't address the fundamental `set-state` bypass issue.

3. **Change identifier resolution applied universally**: All stageman commands (event and non-event) accept change identifiers, not just event commands. Resolution is applied at the CLI dispatch layer before delegating to the function.
   - *Why*: Consistency — callers shouldn't have to remember which commands accept IDs and which require paths. The resolution logic is a one-time cost at the CLI entry point.
   - *Rejected*: Event commands only — would create inconsistent calling conventions.

4. **No prerequisite ordering validation**: This change does NOT add validation that e.g. `start spec` requires `intake: done`. The `allowed_states` per-stage validation is sufficient for now; prerequisite enforcement is a future enhancement.
   - *Why*: Scope control. The transition graph (current_state × event → new_state) is the primary correctness mechanism. Prerequisite ordering is a separate concern that can be added incrementally.
   - *Rejected*: Adding prerequisite validation in this change — increases scope and test surface significantly.

5. **`reset` cascades downstream stages to `pending`**: When stage N is reset, all stages after N are atomically set to `pending` with their `stage_metrics` removed. This mirrors the `finish` side-effect pattern (which activates the next stage) and replaces the multi-call `set-state` pattern used by the `/fab-continue <stage>` reset flow.
   - *Why*: Without `set-state`, the reset flow has no way to set downstream stages to `pending`. Making cascade a side-effect of `reset` keeps the event API self-contained — one command, one atomic operation, full rollback. This is the natural semantic: "reset this stage" implies "everything after it is invalidated."
   - *Rejected*: Compound `reset-to` command — adds a 6th command for one use case. Keeping a restricted `set-state` for pending-only — preserves the backdoor pattern we're eliminating.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | 5 events: start, advance, finish, reset, fail | Confirmed from intake #1 — user designed and confirmed the event vocabulary | S:95 R:70 A:95 D:95 |
| 2 | Certain | Remove `set-state` entirely — no backdoor | Confirmed from intake #2 — user explicitly chose event-driven, no backstop | S:90 R:75 A:90 D:90 |
| 3 | Certain | Remove `transition` — replaced by `finish` | Confirmed from intake #3 — `finish` subsumes transition's role with ready-state support | S:85 R:75 A:90 D:90 |
| 4 | Certain | `finish` works from both active and ready | Confirmed from intake #4 — "no matter where we were in this stage" | S:90 R:75 A:90 D:90 |
| 5 | Certain | `advance` only goes active → ready | Confirmed from intake #5 — "advance could mean transition from active to ready" | S:90 R:80 A:90 D:95 |
| 6 | Certain | Memory docs updated with state transition table | Confirmed from intake #6 — user explicitly requested this | S:95 R:90 A:90 D:95 |
| 7 | Confident | Non-event commands unchanged | Confirmed from intake #7 — read-only or non-stage-state writes, no reason to change | S:50 R:85 A:85 D:85 |
| 8 | Confident | changeman.sh keeps raw path (not change ID shortcut) | Confirmed from intake #8 — changeman already constructs the path internally | S:70 R:80 A:85 D:90 |
| 9 | Confident | All skill .md files updated to use event commands | Confirmed from intake #9 — mandatory since set-state/transition are removed | S:70 R:75 A:85 D:85 |
| 10 | Certain | All commands accept change ID or file path | Confirmed from intake #10 — applies universally to event and non-event commands | S:90 R:80 A:90 D:90 |
| 11 | Certain | Driver is always optional on all event commands | Confirmed from intake #11 — no validation needed, empty when omitted | S:90 R:85 A:90 D:90 |
| 12 | Certain | `finish` side-effect only activates next stage when next is `pending` | Derived from spec analysis — if next stage is already done/active (reset scenario), don't overwrite | S:85 R:70 A:85 D:90 |
| 13 | Confident | `reset` recreates metrics (same as start) rather than clearing entirely | Derived from spec analysis — reset returns to active, which needs fresh metrics with incremented iterations | S:60 R:75 A:80 D:80 |
| 14 | Certain | No prerequisite ordering validation in this change | Scope control — transition graph is the primary correctness mechanism | S:80 R:85 A:90 D:90 |
| 15 | Confident | Transition lookup uses stage-specific override when available, falls back to default | Standard pattern from current workflow.yaml — review has overrides, others use default | S:65 R:80 A:90 D:85 |
| 16 | Certain | `reset` cascades downstream stages to `pending` | Discussed — user chose option 1 (cascade side-effect) over compound command or restricted set-state | S:90 R:75 A:90 D:95 |

16 assumptions (10 certain, 6 confident, 0 tentative, 0 unresolved).
