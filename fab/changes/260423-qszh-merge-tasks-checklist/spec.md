# Spec: Collapse Tasks Stage into Apply; Replace tasks.md + checklist.md with plan.md

**Change**: 260423-qszh-merge-tasks-checklist
**Created**: 2026-05-06
**Affected memory**: `docs/memory/fab-workflow/templates.md`, `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/schemas.md`, `docs/memory/fab-workflow/migrations.md`, `docs/memory/fab-workflow/clarify.md`, `docs/memory/fab-workflow/hydrate.md`

## Non-Goals

- Renaming the `apply` stage to `execute` or `implement` — see Design Decisions, alt R3 rejected
- Dropping the `spec` stage (collapse to `intake → apply → review …`) — see Design Decisions, alt R2 rejected
- Adding a `--pause-after-plan-gen` opt-in flag in this change — deferred to follow-up (R7)
- Teaching `fab score` to score `plan.md` — apply has no gate; future enhancement only
- Changing acceptance-item IDs for in-flight changes mid-flight — migration preserves `CHK-NNN` verbatim; only newly generated plans use `A-NNN`
- Deleting legacy `tasks.md` / `checklist.md` files in `fab/changes/` during migration — they get a "safe to delete" note instead

---

## artifacts: plan.md replaces tasks.md + checklist.md

### Requirement: plan.md is the canonical apply-stage artifact

The kit SHALL produce a single artifact `plan.md` at `fab/changes/{name}/plan.md` containing both the implementation task list and the declarative acceptance criteria. The kit SHALL NOT generate `tasks.md` or `checklist.md` for new changes. Apply parses the `## Tasks` section; review parses the `## Acceptance` section.

#### Scenario: New change generates plan.md only
- **GIVEN** a change with `spec.md` complete and no `plan.md`, `tasks.md`, or `checklist.md`
- **WHEN** `/fab-continue` advances from spec to apply
- **THEN** the apply skill writes `plan.md` with `## Tasks` and `## Acceptance` sections populated
- **AND** `tasks.md` and `checklist.md` are NOT created

#### Scenario: Resume mid-apply skips plan generation
- **GIVEN** a change at apply stage where `plan.md` already exists with some `[ ]` and some `[x]` tasks
- **WHEN** `/fab-continue` is re-run
- **THEN** the plan generation sub-step is skipped (idempotent on plan.md presence)
- **AND** task execution resumes from the first unchecked task

### Requirement: plan.md template structure

`plan.md` SHALL use a stable structure with two heading-keyed sections that downstream skills parse: `## Tasks` (apply consumer) and `## Acceptance` (review consumer). Optional `## Execution Order` and `## Notes` sections MAY follow `## Tasks` and `## Acceptance` respectively. Phase/category subheadings under each section are presentational and MAY vary per change. Task items under `## Tasks` MAY carry the existing `[P]` parallel-execution marker (e.g., `- [ ] T001 [P] {description}`) — the marker semantics are unchanged from the legacy `tasks.md` convention. The template SHALL live at `$(fab kit-path)/templates/plan.md`.
<!-- clarified: [P] parallel marker on task items is preserved from legacy tasks.md convention; fab-clarify plan target taxonomy explicitly scans [P] markers (see Requirement: fab-clarify target disambiguation) -->

#### Scenario: Section headings are the parser contract
- **GIVEN** a `plan.md` containing `## Tasks`, then `### Phase 1: Setup`, then `## Acceptance`, then `### Functional Completeness`
- **WHEN** apply parses the file
- **THEN** apply reads only content between `## Tasks` and `## Acceptance` (or `## Execution Order` / EOF if present)
- **AND** review reads only content between `## Acceptance` and `## Notes` / EOF

#### Scenario: Acceptance items use A-NNN prefix in newly generated plans
- **GIVEN** the unified Plan Generation Procedure runs for a new change
- **WHEN** acceptance items are emitted
- **THEN** their IDs SHALL be `A-001`, `A-002`, ... (sequential, zero-padded to 3 digits)

### Requirement: Remove legacy templates

The kit SHALL NOT ship `src/kit/templates/tasks.md` or `src/kit/templates/checklist.md`. The kit SHALL add `src/kit/templates/plan.md`.

#### Scenario: Templates directory after change
- **GIVEN** the kit is built from this change
- **WHEN** `ls $(fab kit-path)/templates/` is run
- **THEN** `plan.md` is present
- **AND** `tasks.md` is absent
- **AND** `checklist.md` is absent

---

## pipeline: 8 stages → 7 stages

### Requirement: Drop tasks stage from the pipeline

The pipeline SHALL be `intake → spec → apply → review → hydrate → ship → review-pr` (7 stages). The `tasks` stage SHALL be removed entirely from `StageOrder`, allowed-states map, and all skill dispatch tables. After `spec: done`, `finish` SHALL auto-activate `apply`.

#### Scenario: spec finish auto-activates apply
- **GIVEN** a change with `spec: active`
- **WHEN** `fab status finish <change> spec` runs
- **THEN** `progress.spec` becomes `done`
- **AND** `progress.apply` becomes `active`
- **AND** `progress.tasks` does NOT exist in the resulting `.status.yaml`

#### Scenario: StageOrder length is 7
- **GIVEN** the rebuilt `fab-go` binary
- **WHEN** `fab status all-stages` runs
- **THEN** stdout lists exactly 7 stages: `intake`, `spec`, `apply`, `review`, `hydrate`, `ship`, `review-pr` (in order)

### Requirement: Remove tasks from .status.yaml schema

The `.status.yaml` template SHALL drop the `progress.tasks` key entirely. The `checklist:` block SHALL be replaced by a `plan:` block with fields `generated: bool`, `task_count: int`, `acceptance_count: int`, `acceptance_completed: int`. The `path` field is removed (location is fixed at change root). Existing `prs`, `confidence`, `stage_metrics`, `issues`, `change_type`, `id`, `name`, `created`, `created_by`, `last_updated` fields remain unchanged.

#### Scenario: New change template
- **GIVEN** `fab change new --slug example`
- **WHEN** `.status.yaml` is initialized
- **THEN** the file contains `progress:` with 7 keys (intake, spec, apply, review, hydrate, ship, review-pr)
- **AND** the file contains `plan:` with `generated: false`, `task_count: 0`, `acceptance_count: 0`, `acceptance_completed: 0`
- **AND** the file does NOT contain a `checklist:` block
- **AND** the file does NOT contain `progress.tasks`

### Requirement: Strict-error stance for legacy tasks references

`fab` CLI commands referencing the `tasks` stage SHALL error immediately. Specifically: `fab status start|advance|finish|reset|skip|fail <change> tasks` SHALL return exit code 1 with the message `"tasks" stage was removed — run ... apply instead. plan.md is now generated at apply entry.` Similarly, `fab status set-checklist` SHALL error with `"set-checklist" is now "set-acceptance" — run fab status set-acceptance instead.` No alias window.

#### Scenario: Legacy tasks event errors
- **GIVEN** the rebuilt CLI
- **WHEN** `fab status finish <change> tasks` is invoked
- **THEN** exit code is 1
- **AND** stderr contains `"tasks" stage was removed`
- **AND** stderr suggests running `apply` instead

#### Scenario: set-checklist errors with pointer
- **GIVEN** the rebuilt CLI
- **WHEN** `fab status set-checklist <change> total 5` is invoked
- **THEN** exit code is 1
- **AND** stderr contains `"set-checklist" is now "set-acceptance"`

### Requirement: New set-acceptance CLI command

`fab status set-acceptance <change> <field> <value>` SHALL update fields in the `plan:` block of `.status.yaml`. Valid fields: `generated` (bool), `task_count` (int), `acceptance_count` (int), `acceptance_completed` (int). Validation rules and atomic-write semantics MUST match the prior `set-checklist` behavior.

#### Scenario: set-acceptance updates plan block
- **GIVEN** a change with `plan.acceptance_completed: 0`
- **WHEN** `fab status set-acceptance <change> acceptance_completed 5` runs
- **THEN** exit code is 0
- **AND** `.status.yaml` `plan.acceptance_completed` is `5`
- **AND** `last_updated` is refreshed

---

## skills: apply absorbs plan generation

### Requirement: Apply skill generates plan.md at entry

The Apply Behavior in `src/kit/skills/fab-continue.md` SHALL be restructured into two sub-steps that run in a single skill invocation: (1) **Plan Generation sub-step** — read `spec.md`, run the Plan Generation Procedure, write `plan.md`. Skipped on resume when `plan.md` already exists. (2) **Task Execution sub-step** — parse `plan.md` `## Tasks`, execute unchecked items, mark `[x]` on completion. Apply MUST ignore the `## Acceptance` section.

#### Scenario: First-time apply generates and executes
- **GIVEN** `spec.md` exists, `plan.md` does not
- **WHEN** apply behavior runs
- **THEN** `plan.md` is created before any task is executed
- **AND** task execution begins with the first `## Tasks` item

#### Scenario: Apply preconditions
- **GIVEN** apply behavior is invoked at the apply stage
- **WHEN** preconditions are checked
- **THEN** `spec.md` MUST exist (used to generate plan)
- **AND** the prior `tasks.md MUST exist` precondition is removed

### Requirement: Unified Plan Generation Procedure in _generation.md

`src/kit/skills/_generation.md` SHALL replace the **Tasks Generation Procedure** and **Checklist Generation Procedure** with a single **Plan Generation Procedure**. The procedure SHALL enumerate spec requirements once and emit, for each requirement, a Task entry (imperative work item with file path) and an Acceptance entry (declarative outcome that review verifies). Cross-linking between Task and Acceptance IDs is OPTIONAL — the co-generation invariant (single skill call, single context window) is the alignment guarantee.

#### Scenario: Spec → plan single-pass generation
- **GIVEN** a `spec.md` with N requirements
- **WHEN** the Plan Generation Procedure runs
- **THEN** `plan.md` `## Tasks` contains tasks covering each requirement
- **AND** `plan.md` `## Acceptance` contains acceptance items covering each requirement
- **AND** Acceptance items use `A-NNN` IDs

### Requirement: Review reads ## Acceptance section

`src/kit/skills/_review.md` SHALL replace its precondition "tasks.md and checklist.md MUST exist" with "`plan.md` MUST exist with both `## Tasks` and `## Acceptance` sections populated". The inward sub-agent's quality checklist step SHALL inspect items in `plan.md` `## Acceptance` and mark them in place. Pass/fail logic and three-tier finding scheme are unchanged.

#### Scenario: Review precondition checks plan.md
- **GIVEN** apply has completed but `plan.md` is missing the `## Acceptance` section
- **WHEN** review behavior runs
- **THEN** review STOPs with `plan.md missing Acceptance section.`

### Requirement: fab-continue dispatch table

`src/kit/skills/fab-continue.md` dispatch table SHALL drop the `spec ready → tasks` row, drop the `tasks ready → apply` row, and drop the `tasks active → generate tasks.md + checklist` row. The new spec-ready row SHALL be `spec | ready | finish spec → start apply → execute apply` (apply's entry sub-step generates plan.md). The reset target list SHALL drop `tasks` and accept `intake`, `spec`, `apply`, `review`, `hydrate`, `ship`, `review-pr` only.

#### Scenario: spec ready dispatches to apply
- **GIVEN** a change with `progress.spec: ready`
- **WHEN** `/fab-continue` runs (no stage argument)
- **THEN** spec is finished, apply is started, `plan.md` is generated, then task execution begins

#### Scenario: Reset to tasks errors
- **GIVEN** a user runs `/fab-continue tasks`
- **WHEN** the reset target is validated
- **THEN** the skill responds with `"tasks" stage was removed — use /fab-continue apply (regenerates plan.md and re-runs) or /fab-clarify spec.`

#### Scenario: Reset to apply preserves plan.md
- **GIVEN** a change at apply stage with `plan.md` present (some tasks checked) and `fab status reset apply` is invoked
- **WHEN** `/fab-continue` re-runs after the reset
- **THEN** `plan.md` is preserved on disk (reset operates on `.status.yaml` state only — artifact files persist per existing reset semantics)
- **AND** the apply skill's plan-generation sub-step is skipped (plan.md exists)
- **AND** task execution resumes from the first unchecked task
- **AND** to force plan regeneration the user MUST delete `plan.md` before re-running `/fab-continue`
<!-- clarified: reset semantics for plan.md follow existing artifact-file convention (reset modifies .status.yaml progress only; artifacts persist); resumability is intentional — forcing regen requires explicit file deletion -->

### Requirement: fab-ff and fab-fff pipeline shape

`src/kit/skills/fab-ff.md` and `src/kit/skills/fab-fff.md` SHALL drop the separate Step 2 ("Generate `tasks.md`") and Step 3 ("Generate Quality Checklist"). The new flow SHALL be: Step 1 = Generate spec.md → spec gate → auto-clarify spec; Step 2 = Implementation (apply, which internally generates plan.md then executes tasks); Step 3 = Review; Step 4 = Hydrate; (fff continues with Ship, Review-PR). Auto-clarify on tasks SHALL be replaced with auto-clarify on `plan` (target=`plan`) invoked once after plan generation but before execution, and only when this auto-clarify hook is enabled — see Plan Generation Procedure for placement details.

#### Scenario: fab-ff has 4 step blocks (intake → hydrate)
- **GIVEN** the rebuilt `fab-ff.md`
- **WHEN** the file is read
- **THEN** the Behavior section contains exactly four numbered Step blocks: spec, apply, review, hydrate
- **AND** no Step block references `tasks.md` or `checklist.md`

#### Scenario: fab-fff has 6 step blocks (intake → review-pr)
- **GIVEN** the rebuilt `fab-fff.md`
- **WHEN** the file is read
- **THEN** the Behavior section contains exactly six numbered Step blocks: spec, apply, review, hydrate, ship, review-pr

### Requirement: fab-clarify target disambiguation

`src/kit/skills/fab-clarify.md` SHALL accept `intake`, `spec`, and `plan` as valid `<target-artifact>` values. `tasks` SHALL error with `"tasks" target was removed — use plan (post-apply-entry) or spec (pre-apply).` The pre-flight stage guard SHALL allow planning stages `intake` and `spec` only; for `plan` target, the change MUST be at `apply` or later AND `plan.md` MUST exist. The taxonomy scan for `plan` target SHALL cover task completeness, granularity, dependencies, file paths, `[P]` markers (existing tasks taxonomy), plus acceptance coverage of spec requirements.

#### Scenario: clarify plan after apply entry
- **GIVEN** a change at apply stage with `plan.md` present
- **WHEN** `/fab-clarify plan` is invoked
- **THEN** the skill scans `plan.md` `## Tasks` and `## Acceptance` for gaps
- **AND** the stage stays at apply (clarify is non-advancing)

#### Scenario: clarify tasks errors
- **GIVEN** any change state
- **WHEN** `/fab-clarify tasks` is invoked
- **THEN** the skill stops with the tasks-removed error message

### Requirement: _preamble.md updates

`src/kit/skills/_preamble.md` State Table SHALL drop the `tasks` row. The `intake → apply → review → hydrate` narrative count SHALL update from 8 stages to 7. The `Side effects of finish` line in `_cli-fab.md` SHALL update from `intake→spec, spec→tasks, tasks→apply, apply→review, …` to `intake→spec, spec→apply, apply→review, review→hydrate, hydrate→ship, ship→review-pr`.

#### Scenario: State Table after change
- **GIVEN** the rebuilt `_preamble.md`
- **WHEN** the State Table is read
- **THEN** no row has `State = tasks`
- **AND** the spec row's available commands are `/fab-continue, /fab-ff, /fab-clarify`

---

## binary: fab-go updates

### Requirement: StageOrder constant

`src/go/fab/internal/statusfile/statusfile.go` `StageOrder` SHALL be `[]string{"intake", "spec", "apply", "review", "hydrate", "ship", "review-pr"}`. `StageNumber` and `NextStage` SHALL operate on this 7-element list.

#### Scenario: NextStage(spec) returns apply
- **GIVEN** the rebuilt binary
- **WHEN** `NextStage("spec")` is called
- **THEN** the return value is `"apply"`

### Requirement: Allowed states map drops tasks

`src/go/fab/internal/status/status.go` `allowedStates` SHALL drop the `"tasks"` key. `isValidStage("tasks")` SHALL return false.

#### Scenario: tasks is not a valid stage
- **GIVEN** the rebuilt binary
- **WHEN** any event command is invoked with stage `tasks`
- **THEN** an error is returned containing the strict-error message specified above

### Requirement: defaultCommand routing

`src/go/fab/internal/change/change.go` `defaultCommand` SHALL drop the `tasks` case from its `intake|spec|tasks|apply|review` branch.

#### Scenario: Routing for spec ready
- **GIVEN** a change at spec ready
- **WHEN** `defaultCommand("spec")` is called
- **THEN** the return value is `/fab-continue`

### Requirement: Score expected_min for spec stage

`src/go/fab/internal/score/score.go` `expectedMinSpec` thresholds SHALL be unchanged — scoring reads only `spec.md`, never `tasks.md` or `plan.md`. Threshold values for `feat`/`refactor`/`fix` remain `7`/`6`/`5`. The scoring code SHALL contain no reference to `tasks.md`, `checklist.md`, or `plan.md`.

#### Scenario: Score reads only spec.md
- **GIVEN** a change with `spec.md` and `plan.md`
- **WHEN** `fab score <change>` runs
- **THEN** only `spec.md` is opened/parsed for grade counts

### Requirement: hooklib MatchArtifactPath supports plan.md

`src/go/fab/internal/hooklib/artifact.go` `MatchArtifactPath` SHALL replace its known-artifact list `intake.md, spec.md, tasks.md, checklist.md` with `intake.md, spec.md, plan.md`. The PostToolUse hook bookkeeping SHALL, on each `plan.md` write, parse the file by section heading and update `.status.yaml` as follows:

- `plan.task_count` = count of `- [ ]` + `- [x]` items in the `## Tasks` section (between `## Tasks` and the next `##`-level heading or EOF). Updated on every write — total task count is a stable property of the plan.
- `plan.acceptance_count` = count of `- [ ]` + `- [x]` items in the `## Acceptance` section (between `## Acceptance` and the next `##`-level heading or EOF). Updated on every write.
- `plan.acceptance_completed` = count of `- [x]` items in the `## Acceptance` section. Updated on every write — increments as review marks acceptance items complete.
- `plan.generated` = `true` on first write and remains `true` thereafter (never reset by the hook).

If `plan.md` lacks either heading, the corresponding count fields are not modified (defensive: avoid overwriting valid values with zero on a malformed in-progress write). `plan.generated` is still set to `true` if the file exists and contains any `## Tasks` heading.
<!-- clarified: hooklib counts items section-by-section using heading-bounded parse, not whole-file checkbox count — required because plan.md mixes Task and Acceptance items in one file. acceptance_completed is updated on every write (not "unchanged") so review's mark-in-place updates flow into .status.yaml. -->

#### Scenario: plan.md initial write triggers bookkeeping
- **GIVEN** an agent writes `fab/changes/{name}/plan.md` with 5 unchecked tasks under `## Tasks` and 8 unchecked acceptance items under `## Acceptance`
- **WHEN** the PostToolUse hook fires
- **THEN** `.status.yaml` `plan.generated` becomes `true`
- **AND** `plan.task_count` becomes `5`
- **AND** `plan.acceptance_count` becomes `8`
- **AND** `plan.acceptance_completed` becomes `0`

#### Scenario: plan.md re-write updates acceptance_completed
- **GIVEN** an existing `plan.md` with 8 acceptance items, 0 currently checked
- **WHEN** review marks 3 acceptance items `[x]` and the file is written
- **THEN** the PostToolUse hook updates `plan.acceptance_completed` to `3`
- **AND** `plan.acceptance_count` remains `8`
- **AND** `plan.task_count` is unchanged

#### Scenario: tasks.md / checklist.md writes are no-ops
- **GIVEN** an agent writes `fab/changes/{name}/tasks.md` (legacy in-flight migration leftover)
- **WHEN** the PostToolUse hook fires
- **THEN** the hook returns without modifying `.status.yaml`

### Requirement: SetAcceptance Go function

A new `status.SetAcceptance(statusFile, statusPath, field, value)` function SHALL update the `plan:` block fields. The existing `status.SetChecklist` function SHALL be removed. The `statusfile.StatusFile` Go struct SHALL replace the `Checklist` field/struct with a `Plan` struct: `{Generated bool, TaskCount int, AcceptanceCount int, AcceptanceCompleted int}` (with corresponding YAML tags `generated`, `task_count`, `acceptance_count`, `acceptance_completed`).

#### Scenario: SetAcceptance with valid field
- **GIVEN** a `.status.yaml` with `plan.acceptance_count: 10`
- **WHEN** `SetAcceptance(sf, path, "acceptance_completed", "10")` is called
- **THEN** `sf.Plan.AcceptanceCompleted` is `10`
- **AND** the file is saved with `last_updated` refreshed

#### Scenario: SetAcceptance with invalid field
- **GIVEN** any status file
- **WHEN** `SetAcceptance(sf, path, "unknown", "1")` is called
- **THEN** an error is returned: `Invalid plan field 'unknown' (expected: generated, task_count, acceptance_count, acceptance_completed)`

### Requirement: status set-acceptance Cobra command

`src/go/fab/cmd/fab/status.go` SHALL register `statusSetAcceptanceCmd()` (`Use: "set-acceptance <change> <field> <value>"`). The existing `statusSetChecklistCmd()` SHALL be replaced by `statusSetChecklistRemovedCmd()` that exits 1 with the strict-error message `"set-checklist" is now "set-acceptance" — run fab status set-acceptance instead.`

#### Scenario: set-acceptance command works
- **GIVEN** the rebuilt binary
- **WHEN** `fab status set-acceptance qszh task_count 12` runs
- **THEN** exit code is 0
- **AND** `.status.yaml` `plan.task_count` becomes `12`

### Requirement: preflight YAML output

`src/go/fab/internal/preflight/preflight.go` SHALL emit a `plan:` block in its YAML output (replacing the prior `checklist:` block). Fields: `generated`, `task_count`, `acceptance_count`, `acceptance_completed`. The `progress:` block in the output SHALL contain 7 keys.

#### Scenario: preflight output schema
- **GIVEN** an active change with planning complete
- **WHEN** `fab preflight` runs
- **THEN** stdout YAML contains `plan:` with the four fields
- **AND** stdout YAML does NOT contain `checklist:`
- **AND** stdout YAML `progress:` contains no `tasks` key

### Requirement: changeman.sh / fab change list

`fab change list` and other `fab change` outputs that reference stages SHALL operate over the 7-stage list. No literal stage-count constants outside `StageOrder` MAY remain in the codebase.

#### Scenario: fab change list output
- **GIVEN** a change at apply stage
- **WHEN** `fab change list` runs
- **THEN** the displayed `display_stage` for that change reports `apply` (3/7) — never `tasks` (3/8)

---

## migration: 1.8.0 → next

### Requirement: New migration file

A new migration file at `src/kit/migrations/1.8.0-to-1.9.0.md` SHALL be added (or to whatever the next minor release is — see Design Decisions). The migration SHALL be idempotent and SHALL handle three cases per change folder under `fab/changes/` (excluding `archive/`):

1. **`plan.md` already exists** → no-op for that change.
2. **Legacy `tasks.md` and/or `checklist.md` present, no `plan.md`** → produce `plan.md` by concatenating the body content of each legacy file under unified headings:
   - Start with the standard `plan.md` frontmatter (`# Plan: {CHANGE_NAME}`, metadata block, links to `intake.md` and `spec.md`).
   - Emit a `## Tasks` heading, then the body of `tasks.md` (everything below its `# {title}` H1 and any `**Change**:` / `**Status**:` metadata block, stripped) — task subheadings (`### Phase 1: Setup`, etc.), `T001` IDs, and `[P]` markers are preserved verbatim.
   - Emit a `## Acceptance` heading, then the body of `checklist.md` (everything below its H1 and metadata, stripped) — category subheadings and `CHK-NNN` IDs are preserved verbatim.
   - If only one of the two legacy files is present, the corresponding section in `plan.md` SHALL contain a single placeholder note (`<!-- {tasks|acceptance} content not migrated — original {tasks.md|checklist.md} was missing -->`) and the migration SHALL log a warning for that change.
   - Append `<!-- Migrated to plan.md on {DATE} — safe to delete. -->` to both legacy files (whichever exist). Do NOT delete legacy files.
3. **Neither `plan.md` nor legacy files** → no-op (change is pre-planning).
<!-- clarified: "verbatim modulo heading" means strip the H1 title and frontmatter metadata block but preserve all subheadings, item IDs (T-NNN, CHK-NNN), [P] markers, and body content; one-sided legacy state (tasks.md only or checklist.md only) writes a placeholder + warning rather than failing -->

For all non-no-op cases, the migration SHALL rewrite `.status.yaml`:
- Drop `progress.tasks` key. If its value was `done` or `skipped`, leave `progress.apply` at its current state. If its value was `active` or `ready`, set `progress.apply` to that state (the change was mid-planning and is now mid-apply with plan already generated).
- Replace the `checklist:` block with a `plan:` block: `plan.generated = true`, `plan.task_count` = count of `- [ ]` + `- [x]` items in the merged `## Tasks` section, `plan.acceptance_count` = old `checklist.total`, `plan.acceptance_completed` = old `checklist.completed`.

Archived changes (`fab/changes/archive/**`) SHALL be left untouched — they are historical and not subject to this migration. The migration file SHALL include a Verification section that lists post-condition checks.

#### Scenario: Idempotency
- **GIVEN** a change folder where `plan.md` already exists
- **WHEN** the migration runs
- **THEN** `plan.md` is unchanged
- **AND** `tasks.md` / `checklist.md` (if present) are unchanged
- **AND** `.status.yaml` is unchanged

#### Scenario: In-flight change at tasks: ready
- **GIVEN** a change folder with `tasks.md`, `checklist.md`, `progress.tasks: ready`, `checklist: {generated: true, total: 8, completed: 0}`
- **WHEN** the migration runs
- **THEN** `plan.md` is created with `## Tasks` from `tasks.md` content and `## Acceptance` from `checklist.md` content (CHK-001..CHK-008 preserved)
- **AND** `.status.yaml` `progress` no longer has a `tasks` key
- **AND** `.status.yaml` `progress.apply` is `ready`
- **AND** `.status.yaml` `plan.generated` is true
- **AND** `.status.yaml` `plan.task_count` matches the `- [ ]` + `- [x]` count in `## Tasks`
- **AND** `.status.yaml` `plan.acceptance_count` is `8`
- **AND** `.status.yaml` `plan.acceptance_completed` is `0`
- **AND** `tasks.md` and `checklist.md` end with the migration note

#### Scenario: In-flight change at tasks: done
- **GIVEN** a change folder with `progress.tasks: done`, `progress.apply: active`
- **WHEN** the migration runs
- **THEN** `progress.tasks` is removed
- **AND** `progress.apply` remains `active`

#### Scenario: Archived change
- **GIVEN** an archived folder under `fab/changes/archive/2026/04/{name}/`
- **WHEN** the migration runs
- **THEN** the folder's files are unchanged

### Requirement: Status template update

`src/kit/templates/status.yaml` SHALL be updated to drop `progress.tasks` and replace the `checklist` block with a `plan` block, matching the schema spec above. Initial values: `plan: {generated: false, task_count: 0, acceptance_count: 0, acceptance_completed: 0}`.

#### Scenario: Template after change
- **GIVEN** the kit at this version
- **WHEN** `cat $(fab kit-path)/templates/status.yaml` is run
- **THEN** the file contains 7 progress keys and a `plan:` block with the four fields

---

## docs: specs and changelog

### Requirement: Update docs/specs

The following pre-implementation spec files SHALL be updated to reflect the 7-stage pipeline and the `plan.md` artifact:

- `docs/specs/overview.md` — stage list, mermaid diagram, stage details table
- `docs/specs/skills.md` — per-skill flow updates (`fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify`)
- `docs/specs/templates.md` — replace `tasks.md` + `checklist.md` entries with a single `plan.md` entry
- `docs/specs/user-flow.md` — pipeline diagrams updated to 7 stages
- `docs/specs/architecture.md` — if it references progress map keys, update
- `docs/specs/glossary.md` — remove `tasks` stage entry; add/update `plan.md` entry
- `docs/specs/skills/SPEC-fab-continue.md`, `SPEC-fab-ff.md`, `SPEC-fab-fff.md`, `SPEC-fab-clarify.md` — flow diagrams reflect new pipeline

#### Scenario: Specs no longer mention tasks stage
- **GIVEN** the rebuilt repo
- **WHEN** `grep -r "tasks stage" docs/specs/` runs
- **THEN** no matches are found describing `tasks` as an active pipeline stage (historical mentions in changelog-style sections are exempt)

### Requirement: docs/memory hydration

After this change ships, hydrate SHALL update the memory files listed in Affected Memory above to reflect the new 7-stage pipeline, plan.md artifact, removed `tasks` stage references, renamed `set-checklist` → `set-acceptance` CLI surface, schema changes, new migration, and clarify target list. The hydrate step is part of this change's lifecycle (review must pass first).

#### Scenario: Memory consistency
- **GIVEN** hydrate has completed for this change
- **WHEN** the affected memory files are read
- **THEN** all references to the `tasks` stage as a separate pipeline gate are removed (or moved to changelog-style historical notes)
- **AND** `plan.md` is documented as the apply-stage artifact

---

## coordination: 260423-xvaz becomes obsolete

### Requirement: Note xvaz as obsolete in this change's hydrate

The `260423-xvaz-skip-tasks-simple-types` draft proposed a per-type skip policy for the tasks stage. Once `qszh` ships, that draft is obsolete by construction (no `tasks` stage to skip). This spec SHALL include a note in `docs/memory/fab-workflow/planning-skills.md` (during hydrate) that the xvaz approach was superseded by `qszh` collapsing the gate. The xvaz folder itself SHALL NOT be archived by this change — that is a separate, user-initiated action via `/fab-archive 260423-xvaz...` once qszh ships.

#### Scenario: Memory references xvaz superseding
- **GIVEN** hydrate has completed for qszh
- **WHEN** `docs/memory/fab-workflow/planning-skills.md` is read
- **THEN** a Design Decision or Changelog row documents that the simple-type skip policy proposal (xvaz) was superseded by collapsing the tasks stage into apply

---

## Design Decisions

### 1. Drop tasks stage entirely (not rename to plan)

**Decision**: Remove the `tasks` stage from the pipeline; do NOT introduce a `plan` stage. `plan.md` is an apply-internal artifact, not a stage's output.

**Why**: A separate stage adds a transition gate that has no decision content. The xvaz draft (per-type skip policy) exists only to paper over this gate's emptiness — collapsing eliminates both the gate and xvaz. Keeping `apply` as the only post-spec stage means the schema simplifies: 7 progress keys instead of 8, no `progress.plan` to confuse with `plan.md`. (Clarification R1, R4)

**Rejected**: (a) Keep tasks stage, merge artifacts only — fixes drift but leaves the no-decision gate. (b) Rename `tasks` → `plan` stage — same gate, new name. (c) Drop both `spec` and `tasks` — loses the per-type spec gate and `/fab-clarify spec` checkpoint, breaking review's behavioral reference (R2).

### 2. Apply skill absorbs plan generation (not a new dedicated skill)

**Decision**: The apply skill (in `fab-continue.md` § Apply Behavior) gains plan generation as an entry sub-step. No new skill is introduced; no new dedicated `/fab-plan` command.

**Why**: The two sub-steps share context (spec, memory, code-quality config), execute back-to-back without user intervention, and have no decision boundary between them. Splitting them into separate skills would duplicate context loading and add a redundant pause point. Keeping the stage name `apply` (not renaming to `execute` or `implement`) avoids migration churn across state table, `.status.yaml`, all skills, and user muscle memory for marginal semantic gain. (Clarification R3)

**Rejected**: New `/fab-plan` skill — adds a command surface for what is mechanically a single autonomous step. Dedicated `progress.plan` stage with separate `active`/`ready`/`done` tracking — recreates the no-decision gate this change exists to remove.

### 3. plan.md is the parser contract; section headings are stable

**Decision**: `## Tasks` and `## Acceptance` are the stable headings that apply (read `## Tasks`) and review (read `## Acceptance`) parse on. Phase/category subheadings underneath are presentational.

**Why**: Heading-based parsing is what the existing skills already use implicitly for `tasks.md` (Phase 1/2/3) and `checklist.md` (category sections). Promoting two parent headings to a contract codifies the boundary the parsers actually need without locking down the ergonomic subheadings.

**Rejected**: Section markers like `<!-- TASKS-START -->` / `<!-- ACCEPTANCE-START -->` — uglier, identical guarantees. Single mixed list with item-type markers — destroys the imperative-vs-declarative framing reviewers rely on.

### 4. A-NNN for new plans; CHK-NNN preserved for in-flight changes

**Decision**: Newly generated `plan.md` files use `A-001`, `A-002`, ... for acceptance items. The migration preserves `CHK-NNN` IDs verbatim for in-flight changes — no mid-change ID swap.

**Why**: A clean prefix for new artifacts signals the new model. Rewriting IDs mid-change risks breaking outstanding references or rework annotations. Mixed prefix coexistence in `fab/changes/` is acceptable because in-flight changes are short-lived (days to weeks) and migrate away naturally as they complete. (Clarification #9)

**Rejected**: Rename all CHK-NNN to A-NNN during migration — risk of breaking existing rework `<!-- rework: ... -->` annotations or open clarifications. Keep CHK-NNN for new plans too — confuses the new-vs-old boundary; A-NNN better matches the new section name `## Acceptance`.

### 5. Strict-error stance for legacy CLI references

**Decision**: All `tasks` stage references and the old `set-checklist` command error immediately with helpful pointer messages. No alias window.

**Why**: The migration rewrites every in-flight `.status.yaml` so no live change carries a `tasks` key after upgrade. With no live tasks state to support, an alias adds maintenance burden for zero user benefit. Strict errors with pointer messages are self-documenting. (Clarification #12, R6)

**Rejected**: Phased deprecation (alias for one release, error in next) — no in-flight `.status.yaml` carries `tasks` after migration, so phasing buys nothing. Silent renaming — leaves users uncertain whether a command did what they expected.

### 6. Migration leaves legacy files on disk with a "safe to delete" note

**Decision**: Migration appends `<!-- Migrated to plan.md on {DATE} — safe to delete. -->` to legacy `tasks.md` and `checklist.md` rather than deleting them.

**Why**: User-content files (under `fab/changes/`, not templates) deserve a safety margin. Users may want to verify the merge before deletion. Cleanup is cheap later; data loss from an aggressive migration is expensive. (Clarification #15)

**Rejected**: Delete legacy files in migration — risks losing content if the migration concatenated incorrectly. Leave files with no annotation — leaves users confused about whether the file is current or legacy.

### 7. Migration version target

**Decision**: The migration file SHALL be named for the next release this change is included in. The current `fab_version` is `1.8.0`. The implementing agent will check existing `src/kit/migrations/` for the highest version and pick the next minor (`1.8.0-to-1.9.0.md` is the expected name absent a competing change). The release author MAY rename if a coordinated bundle ships first.

**Why**: Migration filenames pin to release boundaries, not change-folder dates. Naming up-front keeps the migration discoverable; the renaming-on-coordination escape hatch handles release-bundling without churn.

**Rejected**: Pin to a hardcoded version pre-coordination — risks collision with parallel changes. Defer naming until release time — the migration file needs a name to land in code review.

---

## Clarifications

### Session 2026-05-06 (auto mode)

| Topic | Resolution |
|-------|------------|
| `[P]` parallel marker on plan.md tasks | Preserved verbatim from legacy `tasks.md` convention; `fab-clarify plan` taxonomy scans `[P]` markers (template structure requirement updated) |
| Hooklib bookkeeping algorithm | Section-bounded parse: count `- [ ]`/`- [x]` between `## Tasks` and next `##` for `task_count`; same between `## Acceptance` and next `##` for `acceptance_count`; `acceptance_completed` counts `- [x]` in `## Acceptance` and updates on every write so review's mark-in-place flows into `.status.yaml` |
| `fab status reset apply` artifact handling | `plan.md` preserved on disk (reset is a state-machine op only); to force plan regeneration user must delete `plan.md` before `/fab-continue` |
| Migration concatenation semantics | "Verbatim modulo heading" = strip legacy file H1 + frontmatter; preserve subheadings, T-NNN/CHK-NNN IDs, `[P]` markers, body content. One-sided legacy state (only tasks.md or only checklist.md) writes a placeholder note + logs a warning instead of failing |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Drop `tasks` stage entirely; fold plan generation into apply entry sub-step | Confirmed from intake #4 — no-decision gate, schema simplifies to 7 keys; supersedes xvaz | S:95 R:60 A:80 D:75 |
| 2 | Certain | Keep `apply` as the post-spec stage name (no rename to `execute`/`implement`) | Confirmed from intake #5 — semantic gain doesn't justify migration churn | S:95 R:70 A:80 D:75 |
| 3 | Certain | Drop `progress.tasks` key from `.status.yaml` (not rename to `progress.plan`) | Confirmed from intake #6 — with no separate stage, no key needed; 7 progress keys total | S:95 R:70 A:85 D:80 |
| 4 | Certain | `## Tasks` and `## Acceptance` are the stable parser contract; phase/category subheadings under each are presentational | Confirmed from intake #13 — heading-based parsing is what existing skills use implicitly | S:95 R:80 A:90 D:85 |
| 5 | Certain | Newly generated plans use `A-NNN` acceptance IDs; migration preserves `CHK-NNN` verbatim for in-flight | Confirmed from intake #7 — avoids mid-change ID rewrite risk | S:95 R:80 A:65 D:60 |
| 6 | Certain | Remove `src/kit/templates/tasks.md` and `src/kit/templates/checklist.md` immediately (no deprecation window) | Confirmed from intake #8 — templates aren't read by in-flight changes; migration handles in-flight artifacts | S:95 R:60 A:65 D:55 |
| 7 | Certain | Migration is idempotent no-op when `plan.md` already exists | Confirmed from intake #9 — standard idempotency contract | S:95 R:75 A:75 D:65 |
| 8 | Certain | Strict-error stance: all `tasks` references error immediately across `fab status`, `fab-continue`, `fab-clarify`; `set-checklist` errors with pointer to `set-acceptance` | Confirmed from intake #10 — no live `.status.yaml` carries `tasks` after migration, so alias window adds zero benefit | S:95 R:55 A:60 D:50 |
| 9 | Certain | Constitution § Additional Constraints mandates a migration for this user-data schema change | Direct constitution quote (`progress.tasks` removal + checklist→plan rename qualifies as schema change) | S:95 R:95 A:95 D:95 |
| 10 | Certain | Apply and review consume different sub-sections of `plan.md` (Tasks vs Acceptance) | Confirmed from intake #3 — apply = imperative, review = declarative; framing preserved by section split | S:95 R:85 A:90 D:90 |
| 11 | Certain | xvaz draft becomes obsolete; archived later by user, not in this change | Confirmed from intake #11 — out-of-scope housekeeping, separate `/fab-archive` action | S:95 R:90 A:90 D:90 |
| 12 | Certain | `--pause-after-plan-gen` flag deferred to follow-up | Confirmed from intake #17 — plan-level clarification still possible via `/fab-clarify spec` (upstream) or `/fab-clarify plan` (post-apply-entry) | S:95 R:80 A:70 D:65 |
| 13 | Certain | No external tooling reads `tasks.md` / `checklist.md` directly | Confirmed from intake #18 — user verified no consumer scripts, dashboards, or projects depend on these files | S:95 R:55 A:90 D:90 |
| 14 | Certain | Acceptance-section bookkeeping moves from `checklist:` block to `plan:` block in `.status.yaml`; field rename `total`→`acceptance_count`, `completed`→`acceptance_completed`, plus new `task_count` | Schema-level decision: tracking task count was previously hook-driven on `tasks.md` writes; folding it under `plan:` colocates plan-related metadata | S:90 R:65 A:80 D:75 |
| 15 | Certain | Removing `path` field from the new `plan:` block (location is fixed at change root) | Old `checklist.path` defaulted to `checklist.md` and was never overridden; YAGNI | S:90 R:80 A:90 D:85 |
| 16 | Certain | Hooklib `MatchArtifactPath` recognizes `plan.md` and stops recognizing `tasks.md` / `checklist.md` (legacy artifacts on in-flight changes are bookkeeping no-ops) | Confirmed from intake #18 (no external consumers) + intake §6 (migration rewrites `.status.yaml`); recognizing legacy files would re-populate a `checklist:` block that no longer exists | S:90 R:70 A:80 D:80 |
| 17 | Certain | Apply preserves resumability: when `plan.md` exists with mixed `[ ]`/`[x]`, plan generation sub-step is skipped and execution resumes from first unchecked task | Confirmed from intake #16 — standard resumability pattern | S:95 R:75 A:90 D:85 |
| 18 | Certain | `/fab-clarify plan` is a valid post-apply-entry target (in addition to `intake` and `spec`) | Confirmed from intake #17 — preserves a plan-level clarification path for users who want to review/refine plan.md before task execution proceeds | S:90 R:80 A:75 D:70 |
| 19 | Confident | Migration filename `1.8.0-to-1.9.0.md` (next minor after current 1.8.0) | Current `fab_version: 1.8.0`; this is a feature-level schema/CLI change warranting a minor bump. The release author may rename if a different bundle ships first | S:75 R:70 A:80 D:75 |
| 20 | Confident | The Plan Generation Procedure walks spec requirements once and emits paired Task + Acceptance per requirement, optional cross-linking | Matches how `_generation.md` already structures Tasks and Checklist procedures (both walk spec); folding into one walk preserves coverage | S:80 R:80 A:85 D:80 |
| 21 | Confident | `_review.md` outward sub-agent prompt requires no plan.md-specific updates (it operates on diff + repo, not artifact-by-artifact) | The outward sub-agent reads diff and full repo per its existing description; it doesn't parse `tasks.md`/`checklist.md` directly | S:80 R:85 A:80 D:75 |
| 22 | Confident | `git-pr.md` Stats table's "Tasks" column derives from `plan.md` `## Tasks` checkbox counts (replaces today's `tasks.md` parse); "Checklist" column is renamed to "Acceptance" or unified into "Tasks/Acceptance" — implementer chooses the cleaner UX, both supported by the new schema | The PR template's purpose is reviewer signal; a single "Tasks (N/M)" + "Acceptance (N/M)" pair is the most readable mapping | S:75 R:80 A:75 D:65 |
| 23 | Confident | Schemas memory file (`docs/memory/fab-workflow/schemas.md`) drops `tasks` from the `allowed_states` example and updates Stage numbering | The existing doc lists `tasks` as a stage with allowed states; this needs editing for consistency | S:80 R:85 A:90 D:85 |
| 24 | Confident | Tests in `src/go/fab/internal/{change,preflight,score,statusfile}/...` are updated to drop `tasks: pending` from their fixture YAMLs | These tests currently include `tasks: pending` in their expected progress maps (lines flagged in repo); they will fail after schema change without updates | S:85 R:60 A:90 D:90 |
| 25 | Certain | Hooklib bookkeeping uses heading-bounded section parse (count `- [ ]`/`- [x]` between `## Tasks` and next `##`-level heading; same for `## Acceptance`); `acceptance_completed` updates on every write so review's mark-in-place flows into `.status.yaml` | Clarified — required because plan.md mixes Task and Acceptance items in one file; a whole-file count would conflate the two; matches existing `CountUncheckedTasks`/`CountChecklistItems` patterns adapted to section scope | S:90 R:80 A:90 D:85 |
| 26 | Certain | `[P]` parallel-execution marker is preserved on plan.md task items (e.g., `- [ ] T001 [P] {description}`) — semantics unchanged from legacy `tasks.md` convention | Clarified — fab-clarify plan target taxonomy explicitly scans `[P]` markers; preserving the marker maintains existing parallel-execution UX | S:95 R:90 A:95 D:90 |
| 27 | Certain | `fab status reset apply` preserves `plan.md` on disk (reset modifies `.status.yaml` state only; artifact files persist) — to force plan regeneration the user MUST delete `plan.md` before `/fab-continue` | Clarified — matches existing reset semantics across all stages (artifacts are user-owned files, reset is a state-machine operation only); preserves Constitution III (idempotency) | S:90 R:80 A:90 D:85 |
| 28 | Certain | Migration concatenation strips legacy file H1 + metadata frontmatter but preserves subheadings, T-NNN/CHK-NNN IDs, `[P]` markers, and body content; one-sided legacy state (only tasks.md or only checklist.md) writes a placeholder + warning rather than failing | Clarified — "verbatim modulo heading" disambiguated; one-sided handling is graceful-degradation pattern consistent with idempotency | S:90 R:75 A:85 D:80 |

28 assumptions (22 certain, 6 confident, 0 tentative, 0 unresolved).
