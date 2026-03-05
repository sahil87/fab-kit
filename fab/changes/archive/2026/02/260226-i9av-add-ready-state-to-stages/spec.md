# Spec: Add Ready State to Stage Lifecycle

**Change**: 260226-i9av-add-ready-state-to-stages
**Created**: 2026-02-26
**Affected memory**: `docs/memory/fab-workflow/schemas.md`, `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing the 6-stage pipeline order (intake → spec → tasks → apply → review → hydrate) — only state vocabulary within stages changes
- Modifying `changeman.sh` — all state logic is fully delegated to `stageman.sh`
- Adding new pipeline stages — `ready` is a state within existing stages, not a stage itself

## Workflow Schema: State Vocabulary

### Requirement: Add `ready` State

The `states` block in `fab/.kit/schemas/workflow.yaml` SHALL include a `ready` state with:
- `id: ready`
- `symbol: "◷"` (or equivalent pending-completion symbol)
- `description: "Stage work product exists, eligible for advancement or clarification"`
- `terminal: false`

The `ready` state SHALL be positioned between `active` and `done` in the states list.

#### Scenario: Schema lists ready state
- **GIVEN** `fab/.kit/schemas/workflow.yaml` is read by `stageman.sh get_all_states()`
- **WHEN** the states block is parsed
- **THEN** the returned list includes `ready` alongside `pending`, `active`, `done`, and `failed`

### Requirement: Remove `skipped` State

The `skipped` state definition SHALL be removed from the `states` block. All references to `skipped` in `transitions`, `allowed_states`, `progression`, and `validation` sections SHALL be removed.

#### Scenario: skipped no longer valid
- **GIVEN** `workflow.yaml` with `skipped` removed
- **WHEN** `stageman.sh validate_state "skipped"` is called
- **THEN** it returns exit code 1 (invalid state)

### Requirement: Update allowed_states Per Stage

Each stage's `allowed_states` SHALL include `ready`:

| Stage | allowed_states |
|-------|---------------|
| intake | `[active, ready, done]` |
| spec | `[pending, active, ready, done]` |
| tasks | `[pending, active, ready, done]` |
| apply | `[pending, active, ready, done]` |
| review | `[pending, active, ready, done, failed]` |
| hydrate | `[pending, active, ready, done]` |

No stage SHALL include `skipped` in `allowed_states`.

#### Scenario: ready is valid for spec stage
- **GIVEN** `workflow.yaml` with updated `allowed_states`
- **WHEN** `stageman.sh validate_stage_state "spec" "ready"` is called
- **THEN** it returns exit code 0 (valid)

#### Scenario: skipped is invalid for all stages
- **GIVEN** `workflow.yaml` with `skipped` removed
- **WHEN** `stageman.sh validate_stage_state "apply" "skipped"` is called
- **THEN** it returns exit code 1 (invalid)

### Requirement: Update Transition Rules

The default transitions SHALL be updated to:

```yaml
transitions:
  default:
    - from: pending
      to: [active]
      condition: "Prerequisites met"
    - from: active
      to: [ready, done]
      condition: "Stage work completed or advanced"
    - from: ready
      to: [done]
      condition: "Stage advanced to next"
    - from: done
      to: [active]
      condition: "Reset/rework requested"
```

The `active → ready` transition represents artifact generation completing. The `ready → done` transition represents advancement to the next stage. The `active → done` path is preserved for cases where `ready` is skipped (e.g., execution stages that auto-advance internally).

The review-specific transitions SHALL add:
```yaml
  review:
    - from: active
      to: [ready, done, failed]
    - from: ready
      to: [done]
    - from: failed
      to: [active]
    - from: done
      to: [active]
```

All references to `skipped` in transitions SHALL be removed.

#### Scenario: active to ready is valid default transition
- **GIVEN** a stage with state `active`
- **WHEN** `stageman.sh set-state <file> spec ready` is called
- **THEN** the state changes to `ready` without error

#### Scenario: ready to done is valid
- **GIVEN** a stage with state `ready`
- **WHEN** `stageman.sh set-state <file> spec done` is called
- **THEN** the state changes to `done` without error

### Requirement: Update Progression Rules

The `progression.current_stage.rule` SHALL be updated to:
`"First stage with state=active or state=ready; if no active/ready, first pending stage after last done; if all done, hydrate"`

The `progression.next_stage.rule` SHALL remove `skipped` references:
`"First stage in sequence where requires[] are all done and state=pending"`

#### Scenario: ready stage is current stage
- **GIVEN** `.status.yaml` with `spec: ready`, all others `pending` or `done`
- **WHEN** `stageman.sh current-stage` is called
- **THEN** it returns `spec`

### Requirement: Update Validation Rules

The `validation.prerequisites.rule` SHALL remove `skipped` references. The `validation.terminal_states.rule` SHALL be updated to: `"done stages can only transition back to active via explicit reset"`.

## Stage Manager: stageman.sh

### Requirement: get_progress_line() Renders Ready Symbol

`get_progress_line()` SHALL render stages with `ready` state using the `ready` symbol from `workflow.yaml` (e.g., `◷`). A `ready` stage SHALL appear in the visual progress chain (it is a visible state, not hidden like `pending`).

#### Scenario: progress line with ready stage
- **GIVEN** `.status.yaml` with `intake: done`, `spec: ready`, `tasks: pending`
- **WHEN** `stageman.sh progress-line <file>` is called
- **THEN** the output includes both `intake` and `spec` with the `ready` symbol (e.g., `intake → spec ◷`)

### Requirement: get_current_stage() Treats Ready as Routable

`get_current_stage()` SHALL return a `ready` stage as the current stage. The three-tier fallback becomes:
1. First stage with `active` or `ready` state — return it
2. First `pending` stage after last `done` — return it
3. `hydrate` if all done

When a stage is `ready`, the routing semantic is "advance this stage" (not "generate its artifact"). Skills use the state value to distinguish: `active` = generate, `ready` = advance.

#### Scenario: current-stage returns ready stage
- **GIVEN** `.status.yaml` with `intake: done`, `spec: ready`
- **WHEN** `stageman.sh current-stage <file>` is called
- **THEN** it returns `spec`

### Requirement: get_display_stage() Shows Ready

`get_display_stage()` SHALL return a `ready` stage as the display stage with state `ready`. The tier hierarchy becomes:
1. First `active` stage
2. First `ready` stage
3. Last `done` stage
4. First `pending` stage

#### Scenario: display-stage shows ready
- **GIVEN** `.status.yaml` with `intake: done`, `spec: ready`
- **WHEN** `stageman.sh display-stage <file>` is called
- **THEN** it returns `spec:ready`

### Requirement: _apply_metrics_side_effect() is No-Op for Ready

Setting a stage to `ready` SHALL NOT modify `stage_metrics`. The existing metrics from the `active` phase (started_at, driver, iterations) SHALL be preserved. The `completed_at` timestamp is NOT set until the stage reaches `done`.

#### Scenario: set-state ready preserves metrics
- **GIVEN** `.status.yaml` with `spec: active` and `stage_metrics.spec.started_at` set
- **WHEN** `stageman.sh set-state <file> spec ready` is called
- **THEN** `stage_metrics.spec.started_at` is unchanged
- **AND** `stage_metrics.spec.completed_at` is NOT set

### Requirement: set_stage_state() Does Not Require Driver for Ready

Setting a stage to `ready` SHALL NOT require a `driver` parameter. The `driver` requirement applies only when setting state to `active`.

#### Scenario: set-state ready without driver
- **GIVEN** a valid `.status.yaml`
- **WHEN** `stageman.sh set-state <file> spec ready` is called (no driver argument)
- **THEN** the command succeeds (exit 0)

#### Scenario: set-state active still requires driver
- **GIVEN** a valid `.status.yaml`
- **WHEN** `stageman.sh set-state <file> spec active` is called (no driver argument)
- **THEN** the command fails with "ERROR: driver required when setting state to 'active'"

### Requirement: validate_status_file() Does Not Count Ready as Active

The active-count validation (0 or 1 active stages) SHALL count only stages with state `active`. Stages with state `ready` SHALL NOT be included in the active count.

#### Scenario: one active and one ready is valid
- **GIVEN** `.status.yaml` with `spec: active`, `tasks: ready`
- **WHEN** `stageman.sh validate-status-file <file>` is called
- **THEN** validation passes (active_count = 1, which is within [0,1])

### Requirement: transition_stages() Behavior Unchanged

The `transition_stages()` function's signature and behavior SHALL remain unchanged. It still performs the two-write transition: `from_stage → done`, `to_stage → active`. Skills that use the `ready` intermediate SHALL call `set-state <stage> ready` first, then call `transition` when advancing. The `ready → done` step is NOT handled by `transition_stages()` — it happens implicitly because `transition` sets `from_stage` to `done`.
<!-- assumed: transition_stages keeps its current from→done, to→active contract rather than being extended to handle ready→done explicitly -->

#### Scenario: transition from ready stage
- **GIVEN** `.status.yaml` with `spec: ready`
- **WHEN** skill calls `set-state spec done` then `set-state tasks active fab-continue`
- **THEN** both writes succeed and `spec: done`, `tasks: active`

## Preflight: preflight.sh

### Requirement: Preflight Routing Includes Ready

The `stage` field (routing) SHALL return a `ready` stage as the current stage. Skills distinguish `active` vs `ready` using the `display_state` field to determine whether to generate or advance.

The `display_stage` and `display_state` fields SHALL reflect `ready` state:
- `display_stage: spec`, `display_state: ready` when `spec: ready`

#### Scenario: preflight output for ready stage
- **GIVEN** `.status.yaml` with `spec: ready`
- **WHEN** `preflight.sh` is run
- **THEN** output includes `stage: spec`, `display_stage: spec`, `display_state: ready`

## Skills: fab-continue

### Requirement: Split Generate and Advance Dispatch

`/fab-continue` SHALL distinguish between `active` and `ready` states for dispatch:

| State | Dispatch |
|-------|----------|
| `active` | Generate the stage's artifact, then set to `ready` |
| `ready` | Advance to the next stage (`set-state <stage> done`, `set-state <next> active`) |

This replaces the current single-dispatch where generation and advancement happen in one step.

#### Scenario: fab-continue on active spec generates
- **GIVEN** `spec: active`, `spec.md` does not exist
- **WHEN** `/fab-continue` is invoked
- **THEN** `spec.md` is generated
- **AND** `spec` is set to `ready` (not `done`)

#### Scenario: fab-continue on ready spec advances
- **GIVEN** `spec: ready`, `spec.md` exists
- **WHEN** `/fab-continue` is invoked
- **THEN** `spec` is set to `done`, `tasks` is set to `active`
- **AND** no new artifact is generated

### Requirement: Single-Dispatch Rule Still Applies

Each `/fab-continue` invocation SHALL still execute exactly ONE action: either generate (active → ready) OR advance (ready → done + next → active). The user runs `/fab-continue` again to proceed.

#### Scenario: generate does not auto-advance
- **GIVEN** `spec: active`
- **WHEN** `/fab-continue` generates spec.md and sets `spec: ready`
- **THEN** execution STOPS — it does NOT also advance to tasks

## Skills: fab-ff (Targeted Edits)

The existing `fab-ff.md` structure is mostly preserved (~70%). Changes are: new starting point (intake), insert spec generation step + spec gate, and swap review fallback from interactive to stop. The auto-rework loop, decision heuristics, escalation rule, apply/hydrate behavior, resumability, output format, and context loading all stay as-is.

### Requirement: Start from Intake

`/fab-ff` SHALL accept invocation from the `intake` stage (was: spec-only). The minimum prerequisite changes from "spec must be active or later" to "intake must be active or later."

#### Scenario: fab-ff starts from intake
- **GIVEN** `intake: active` or `intake: ready`
- **WHEN** `/fab-ff` is invoked
- **THEN** the pipeline begins (generating spec, then tasks, then apply, review, hydrate)

### Requirement: Intake Gate

`/fab-ff` SHALL refuse to start if the indicative confidence score at the intake stage is < 3.0. The indicative score is computed using `calc-score.sh --stage intake <change_dir>`.

#### Scenario: intake gate rejects low confidence
- **GIVEN** intake with 2 tentative assumptions and indicative score 1.5
- **WHEN** `/fab-ff` is invoked
- **THEN** it STOPS with: `Indicative confidence is {score} of 5.0 (need >= 3.0). Run /fab-clarify to resolve, then retry.`

#### Scenario: intake gate passes
- **GIVEN** intake with mostly certain assumptions and indicative score 4.2
- **WHEN** `/fab-ff` is invoked
- **THEN** the pipeline proceeds past the intake gate

### Requirement: Spec Gate

`/fab-ff` SHALL stop after spec generation if the confidence score is below the type-specific threshold (dynamic per-type: fix=2.0, feat/refactor=3.0, docs/test/ci/chore=2.0, default=3.0). The gate is checked via `calc-score.sh --check-gate <change_dir>`.

#### Scenario: spec gate stops low confidence
- **GIVEN** spec generated with confidence score 2.5 and change type `feat` (threshold 3.0)
- **WHEN** the spec gate check runs
- **THEN** `/fab-ff` STOPS with: `Confidence is {score} of 5.0 (need > {threshold} for {type}). Run /fab-clarify to resolve, then retry.`

### Requirement: Review Gate (3-Cycle Cap)

`/fab-ff` SHALL use autonomous review rework with a 3-cycle cap. On exhaustion after 3 cycles, `/fab-ff` SHALL stop (not fall back to interactive). The escalation rule applies: after 2 consecutive "fix code" attempts, escalate to "revise tasks" or "revise spec."

#### Scenario: review gate stops after 3 cycles
- **GIVEN** review fails 3 times despite autonomous rework
- **WHEN** the 3rd rework cycle fails
- **THEN** `/fab-ff` STOPS with a per-cycle summary and suggests `/fab-continue` for manual rework

### Requirement: No Frontloaded Questions

`/fab-ff` SHALL NOT frontload questions (unchanged from current behavior concept — since spec may not exist yet, it generates without batched Q&A). SRAD-driven questions during spec generation follow the normal `/fab-continue` budget (1-2 per stage).

### Requirement: Resumability

`/fab-ff` SHALL be resumable — re-running after a gate stop picks up from the first incomplete stage. Stages already `done` or `ready` are skipped.

#### Scenario: resume after spec gate stop
- **GIVEN** `spec: ready` (gate stopped here), `tasks: pending`
- **WHEN** user runs `/fab-clarify` to raise confidence, then re-runs `/fab-ff`
- **THEN** the spec gate is re-checked; if passing, pipeline continues from tasks generation

## Skills: fab-fff (Clarifications)

### Requirement: No Gates

`/fab-fff` SHALL NOT check any confidence gates — no intake gate, no spec gate. It forces through all stages regardless of confidence scores (unchanged).

### Requirement: 3-Cycle Review Cap Preserved

`/fab-fff` SHALL retain the 3-cycle autonomous review rework cap with bail on exhaustion. The cap is a practical safety limit, not a confidence gate.
<!-- assumed: fab-fff's 3-cycle review cap stays as-is — "forces through" means no confidence gates, not no safety limits -->

#### Scenario: fab-fff bails after 3 review failures
- **GIVEN** review fails 3 times despite autonomous rework
- **WHEN** the 3rd rework cycle fails
- **THEN** `/fab-fff` bails with a per-cycle summary (unchanged behavior)

## Skills: fab-clarify

### Requirement: Accept Ready State for Scanning

`/fab-clarify` SHALL accept stages in `ready` state (artifact exists) for taxonomy scanning. The stage guard becomes: stage MUST be `active` or `ready` for planning stages (intake, spec, tasks).

#### Scenario: clarify on ready spec
- **GIVEN** `spec: ready`, `spec.md` exists
- **WHEN** `/fab-clarify` is invoked
- **THEN** it scans `spec.md` for gaps and presents questions (stays `ready` throughout)

## Skills: _preamble.md

### Requirement: Update State Table

The State Table in `_preamble.md` SHALL be updated to reflect that `ready` stages route the same as their corresponding state but with the "advance" semantic. No new rows needed — the existing state-keyed routing works because skills check `display_state` for the active/ready distinction.

### Requirement: Update Confidence Gate Thresholds Section

The gate thresholds section SHALL note that `/fab-ff` now has an additional intake gate (indicative score >= 3.0) alongside the existing spec gate.

## Confidence Scoring: calc-score.sh

### Requirement: Support --stage intake

`calc-score.sh` SHALL accept `--stage intake` to compute the indicative confidence score from `intake.md`'s Assumptions table (instead of the default `spec.md`). The `expected_min` thresholds for intake stage are read from the existing per-type tables (already defined in `change-types.md`).

#### Scenario: calc-score at intake stage
- **GIVEN** `intake.md` with 10 certain, 4 confident, 2 tentative assumptions and change type `feat`
- **WHEN** `calc-score.sh --stage intake <change_dir>` is run
- **THEN** it reads `intake.md` Assumptions table, uses `expected_min` intake threshold for `feat` (4), and outputs the computed score

#### Scenario: check-gate at intake stage
- **GIVEN** `calc-score.sh --check-gate --stage intake <change_dir>`
- **WHEN** the indicative score is 2.5
- **THEN** the gate fails with exit code 1 and message showing score vs threshold (3.0 for intake gate)

## Deprecated Requirements

### `skipped` State

**Reason**: The `skipped` state is defined in `workflow.yaml` but never set by any script or skill. No stage has ever been marked `skipped` in any `.status.yaml`. Dead code.

**Migration**: Remove from `workflow.yaml` states, transitions, allowed_states, progression rules, and validation rules. No `.status.yaml` migration needed — no existing files contain `skipped` values.

## Design Decisions

1. **`ready` is not a work-in-progress state**: `ready` signals that generation is complete and the artifact exists. It does NOT require a `driver`, does NOT trigger metrics updates, and does NOT count toward the active-stage limit. Only `active` is a work-in-progress state.
   - *Why*: Keeps the semantic clean — `active` = agent is working, `ready` = agent is done, human decides next step.
   - *Rejected*: Treating `ready` like a second `active` — would conflate work-in-progress with awaiting-advancement.

2. **`transition_stages()` contract unchanged**: Rather than extending the transition function to handle `ready → done`, skills call `set-state` to move through `ready → done` before (or instead of) calling `transition`. This preserves the simple two-write contract.
   - *Why*: The existing transition function is well-tested and understood. Adding a three-write path increases complexity for little benefit since skills already manage state explicitly.
   - *Rejected*: Extending `transition` to accept a `from_state` parameter (e.g., `transition --from-state ready`) — API change with backward compatibility concerns.

3. **`/fab-ff` targeted edits, not full rewrite**: `/fab-ff` starts from intake with 3 safety gates. ~70% of the existing skill file stays (auto-rework loop, decision heuristics, escalation rule, apply/hydrate behavior, resumability, output format, context loading). Changes: new starting point, insert spec generation step + spec gate, swap review fallback from interactive to stop.
   - *Why*: The distinction between `/fab-ff` (gated, can stop) and `/fab-fff` (ungated, forces through) is cleaner and more intuitive than the previous "start point + rework style" differentiation. Most pipeline mechanics are unchanged.
   - *Rejected*: Full rewrite — unnecessary given the high overlap with existing structure. Keeping `/fab-ff` as spec-only — unnecessarily limits the command's utility.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Five states: `pending → active → ready → done` + `failed` | Confirmed from intake #1 — user-discussed, naming settled | S:95 R:70 A:90 D:95 |
| 2 | Certain | `ready` = artifact exists, eligible for advance/clarify | Confirmed from intake #2 — user-specified semantic | S:95 R:70 A:90 D:95 |
| 3 | Certain | `active` = work in progress, generation/execution ongoing | Confirmed from intake #3 — user-specified semantic | S:95 R:70 A:90 D:95 |
| 4 | Certain | `transition_stages()` contract unchanged — skills use `set-state` for ready→done | Upgraded from intake #4 (Confident→Certain) — analysis shows extension adds complexity with no benefit; existing two-write contract works | S:90 R:60 A:90 D:90 |
| 5 | Confident | New `stageman.sh complete` subcommand: `complete <file> <stage>` = `set-state <stage> ready` | Upgraded from intake #5 (Tentative→Confident) — syntactic sugar over `set-state`, but clarifies intent and reduces error-prone direct calls | S:70 R:90 A:75 D:65 |
| 6 | Certain | Pipelines auto-advance through `ready` | Confirmed from intake #6 | S:95 R:60 A:90 D:95 |
| 7 | Certain | `failed` unchanged; rework stages go through `ready` | Confirmed from intake #7 | S:95 R:55 A:90 D:95 |
| 8 | Certain | `/fab-continue` auto-advances from `ready` | Confirmed from intake #8 | S:95 R:70 A:95 D:95 |
| 9 | Certain | Drop `skipped` — dead code cleanup | Confirmed from intake #9 — codebase analysis verified no usage | S:95 R:85 A:95 D:95 |
| 10 | Certain | `/fab-ff` starts from intake with 3 gates | Confirmed from intake #11-14 — user-specified design | S:95 R:45 A:90 D:90 |
| 11 | Confident | `/fab-ff` spec gate keeps dynamic per-type thresholds | Confirmed from intake #13 — consistent with existing calc-score.sh | S:80 R:55 A:85 D:75 |
| 12 | Certain | `/fab-ff` intake gate threshold: 3.0 (fixed, not per-type) | From intake #12 — user specified "< 3" as a fixed threshold | S:90 R:50 A:85 D:85 |
| 13 | Confident | `/fab-fff` 3-cycle review cap preserved | From intake #15 (Tentative→Confident) — "forces through" means no gates, not no safety limits | S:75 R:45 A:70 D:65 |
| 14 | Certain | `ready` state: no driver, no metrics side-effect, not counted as active | Confirmed from intake #17 — `ready` is completion signal, not work-in-progress | S:90 R:75 A:85 D:90 |
| 15 | Certain | `changeman.sh` needs no changes | Confirmed from intake #16 — full delegation to stageman | S:95 R:90 A:95 D:95 |
| 16 | Confident | `ready` symbol: `◷` | Chosen for visual consistency — not `active` (●) or `pending` (○), suggests "waiting/ready" | S:70 R:95 A:70 D:60 |
| 17 | Confident | `active → done` path preserved alongside `active → ready → done` | Execution stages may skip `ready` for internal auto-advance; keeping both paths avoids forcing all flows through `ready` | S:75 R:65 A:80 D:70 |

17 assumptions (10 certain, 7 confident, 0 tentative, 0 unresolved).
