# Spec: Streamline Planning Stage Dispatch

**Change**: 260227-ijql-streamline-planning-dispatch
**Created**: 2026-02-27
**Affected memory**: `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/templates.md`

## Non-Goals

- Changing the state machine in `workflow.yaml` — all transitions already exist
- Changing `stageman.sh` — event commands are already correct
- Modifying apply, review, or hydrate behavior — they already work in single invocations
- Changing `/fab-ff` or `/fab-fff` pipeline behavior — they have their own dispatch logic

## Template: Status Initialization

### Requirement: Template SHALL initialize all stages as `pending`

The `fab/.kit/templates/status.yaml` template SHALL set `intake: pending` (not `intake: active`). All six stages SHALL start as `pending` in the template. The `changeman.sh` script is responsible for transitioning intake from `pending` to `active` via `stageman.sh start`.

#### Scenario: New change creation via changeman.sh

- **GIVEN** a user runs `/fab-new` which calls `changeman.sh new`
- **WHEN** `changeman.sh` initializes `.status.yaml` from the template
- **THEN** all stages are `pending` in the template output
- **AND** `changeman.sh` calls `stageman.sh start <file> intake fab-new` to transition intake to `active`
- **AND** `stage_metrics.intake` is populated with `started_at`, `driver: "fab-new"`, `iterations: 1`

#### Scenario: Responsibility separation across layers

- **GIVEN** the three layers (template, script, skill) each have distinct responsibilities
- **WHEN** a new change is created end-to-end
- **THEN** template provides `pending` (blank slate)
- **AND** `changeman.sh` transitions to `active` (with metrics)
- **AND** `/fab-new` generates `intake.md` and calls `advance` to reach `ready`

## Skill: `/fab-new` Intake Advancement

### Requirement: `/fab-new` SHALL leave intake as `ready` after generating `intake.md`

After generating the intake artifact, `/fab-new` SHALL call `fab/.kit/scripts/lib/stageman.sh advance <change_dir> intake` to transition the intake stage from `active` to `ready`. The skill SHALL NOT call `finish` — the stage remains open for `/fab-clarify` refinement.

#### Scenario: Normal `/fab-new` invocation

- **GIVEN** a user runs `/fab-new <description>`
- **WHEN** `/fab-new` completes intake generation
- **THEN** `intake.md` exists in the change folder
- **AND** `.status.yaml` shows `intake: ready`
- **AND** `stage_metrics.intake` has `started_at` and `driver: "fab-new"`

#### Scenario: User wants to clarify intake before proceeding

- **GIVEN** `/fab-new` has completed and intake is `ready`
- **WHEN** the user runs `/fab-clarify`
- **THEN** `/fab-clarify` refines `intake.md` in place
- **AND** intake remains `ready` (not advanced or reset)

#### Scenario: User proceeds immediately after `/fab-new`

- **GIVEN** `/fab-new` has completed and intake is `ready`
- **WHEN** the user runs `/fab-continue`
- **THEN** `/fab-continue` finishes intake and generates spec in one invocation

## Skill: `/fab-continue` Consolidated Planning Dispatch

### Requirement: `/fab-continue` SHALL collapse finish + generate into a single invocation for planning stages

When the current stage is a planning stage (intake, spec, or tasks) in the `ready` state, `/fab-continue` SHALL finish the current stage, start the next stage, generate its artifact, and advance the next stage to `ready` — all in a single invocation. The single-dispatch rule SHALL be removed.

#### Scenario: Intake ready, generate spec

- **GIVEN** intake is `ready` and spec is `pending`
- **WHEN** the user runs `/fab-continue`
- **THEN** `/fab-continue` runs `finish intake` → intake becomes `done`, spec auto-activates
- **AND** generates `spec.md`
- **AND** runs `advance spec` → spec becomes `ready`
- **AND** outputs the spec content and assumptions summary

#### Scenario: Spec ready, generate tasks

- **GIVEN** spec is `ready` (or `done` with tasks `pending`)
- **WHEN** the user runs `/fab-continue`
- **THEN** `/fab-continue` runs `finish spec` → spec becomes `done`, tasks auto-activates
- **AND** generates `tasks.md` and `checklist.md`
- **AND** runs `advance tasks` → tasks becomes `ready`

#### Scenario: Tasks ready, begin apply

- **GIVEN** tasks is `ready`
- **WHEN** the user runs `/fab-continue`
- **THEN** `/fab-continue` runs `finish tasks` → tasks becomes `done`, apply auto-activates
- **AND** begins task execution (apply behavior)
- **AND** on completion runs `finish apply` (apply goes directly to `done`, no `ready` state)

### Requirement: Planning stages in `active` state SHALL generate and advance to `ready`

When a planning stage is `active` (backward compatibility for interrupted generations), `/fab-continue` SHALL generate the stage's artifact and advance it to `ready`. This is the same generate-then-advance behavior as before, but the `ready` → `done` transition no longer requires a separate invocation.

#### Scenario: Spec active (interrupted generation)

- **GIVEN** spec is `active` (e.g., previous generation was interrupted)
- **WHEN** the user runs `/fab-continue`
- **THEN** `/fab-continue` generates `spec.md`
- **AND** runs `advance spec` → spec becomes `ready`
- **AND** does NOT finish spec or start tasks (user runs `/fab-continue` again to proceed)

### Requirement: The revised dispatch table SHALL be the authoritative dispatch reference

The dispatch table in `fab-continue.md` SHALL be updated to reflect the new behavior:

| Derived stage | State | Action |
|---|---|---|
| `intake` | `ready` | finish intake → start spec → generate `spec.md` → advance spec to `ready` |
| `intake` | `active` | generate intake if missing → advance to `ready` |
| `spec` | `ready` | finish spec → start tasks → generate `tasks.md` + checklist → advance tasks to `ready` |
| `spec` | `active` | generate `spec.md` → advance to `ready` |
| `tasks` | `ready` | finish tasks → start apply → execute tasks → finish apply |
| `tasks` | `active` | generate `tasks.md` + checklist → advance to `ready` |
| `apply` | `active`/`ready` | execute apply → finish apply |
| `review` | `active`/`ready` | execute review → pass: finish / fail: rework |
| `hydrate` | `active`/`ready` | execute hydrate → finish hydrate |
| all `done` | — | Block: "Change is complete." |

#### Scenario: Complete planning flow with three invocations

- **GIVEN** a freshly created change with intake `ready`
- **WHEN** the user runs `/fab-continue` three times
- **THEN** first invocation: finishes intake, generates spec, spec is `ready`
- **AND** second invocation: finishes spec, generates tasks + checklist, tasks is `ready`
- **AND** third invocation: finishes tasks, executes apply, apply is `done`

### Requirement: The single-dispatch rule SHALL be removed from `fab-continue.md`

The line "**Single-dispatch rule**: Execute exactly ONE action per invocation..." (currently at line 59 of `fab-continue.md`) SHALL be removed. It is replaced by the consolidated dispatch behavior described above.

## Skill: Reset Flow Preservation

### Requirement: Reset flow SHALL continue to use `advance` (not `finish`) for planning stages

The existing reset flow behavior (§ Reset Flow in `fab-continue.md`) SHALL remain unchanged. After regenerating an artifact during a planning reset, `/fab-continue` SHALL use `advance` (not `finish`) to move the stage to `ready` — preventing auto-activation of the next stage. This preserves the user's ability to `/fab-clarify` after a reset.

#### Scenario: Reset to spec stage

- **GIVEN** the change is at tasks or later
- **WHEN** the user runs `/fab-continue spec`
- **THEN** spec is reset to `active`, downstream stages cascade to `pending`
- **AND** spec is regenerated
- **AND** spec advances to `ready` (not `done`)
- **AND** the user can run `/fab-clarify` or `/fab-continue` to proceed

## Spec: User-Flow Diagrams

### Requirement: Section 2 ("The Same Flow, With Fab") SHALL reflect the new invocation count

The diagram in `docs/specs/user-flow.md` Section 2 SHALL show that each `/fab-continue` transition covers a full stage (not half a stage). The transitions remain `intake →|"/fab-continue"| spec →|"/fab-continue"| tasks`, etc.

### Requirement: Section 5 ("Per-Stage State Machine") SHALL be unchanged

The per-stage state machine (`pending → active → ready → done`) is unchanged. The transitions and side-effects are unchanged. Only the skill dispatch behavior changes, not the state machine itself.

## Spec: Skills Reference

### Requirement: The `/fab-continue` section in `docs/specs/skills.md` SHALL reflect the new dispatch model

The Next Steps table and `/fab-continue` description in `docs/specs/skills.md` SHALL be updated to show the consolidated dispatch. The `/fab-new` description SHALL note that intake ends as `ready`.

#### Scenario: Next Steps table after `/fab-new`

- **GIVEN** the user has run `/fab-new`
- **WHEN** they consult the Next Steps table
- **THEN** intake state is `ready` (not `active`)
- **AND** suggested next is `/fab-continue` or `/fab-clarify`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `/fab-new` leaves intake as `ready` | Confirmed from intake #1 — user explicitly chose `ready` to preserve `/fab-clarify` checkpoint | S:95 R:85 A:95 D:95 |
| 2 | Certain | `/fab-continue` collapses finish + generate into single invocation for planning stages | Confirmed from intake #2 — user explicitly confirmed this behavior | S:95 R:80 A:90 D:95 |
| 3 | Certain | `ready` state is the refinement checkpoint for `/fab-clarify` | Confirmed from intake #3 — user rejected `done` because it requires reset to clarify | S:90 R:90 A:95 D:95 |
| 4 | Certain | No changes to `workflow.yaml` or `stageman.sh` | Confirmed from intake #4 — existing state machine supports all needed transitions | S:90 R:95 A:95 D:90 |
| 5 | Certain | Template initializes `intake: pending`, `changeman.sh start` transitions to `active` | Confirmed from intake #8 — fixes bug where `start` fails on `active` template default | S:95 R:90 A:95 D:95 |
| 6 | Certain | Apply, review, hydrate behavior unchanged | Confirmed from intake #5 — already work in single invocations | S:90 R:95 A:95 D:95 |
| 7 | Confident | `active` planning rows are backward-compat only | Confirmed from intake #6 — for interrupted generations; normal path uses `ready` | S:70 R:90 A:85 D:80 |
| 8 | Certain | Reset flow preserved — uses `advance` not `finish` | Codebase confirms: reset stops at target stage per existing design decision | S:90 R:85 A:95 D:90 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
