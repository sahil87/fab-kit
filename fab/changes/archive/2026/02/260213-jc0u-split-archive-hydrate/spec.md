# Spec: Split Archive into Hydrate Stage and fab-archive Command

**Change**: 260213-jc0u-split-archive-hydrate
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/change-lifecycle.md`, `fab/docs/fab-workflow/execution-skills.md`, `fab/docs/fab-workflow/templates.md`, `fab/docs/fab-workflow/configuration.md`, `fab/docs/fab-workflow/schemas.md`

## Non-Goals

- Migrating existing archived changes — changes already in `fab/changes/archive/` with `archive: done` keep their old `.status.yaml` format
- Adding a progress map entry for fab-archive — it is a one-shot housekeeping action, not a tracked pipeline stage
- Changing any behavior of planning stages (brief, spec, tasks) or execution stages (apply, review)

## Pipeline: Stage Progression

### Requirement: Hydrate Replaces Archive in Progress Map

The pipeline SHALL use `hydrate` as the terminal tracked stage, replacing `archive`. The progress map in `.status.yaml` SHALL contain these 6 stages:

```
brief → spec → tasks → apply → review → hydrate
```

The `archive` key SHALL NOT appear in the progress map of newly created changes. The pipeline phases remain:
- **Planning** (1-3): brief, spec, tasks
- **Execution** (4-5): apply, review
- **Completion** (6): hydrate

#### Scenario: New change creation

- **GIVEN** a user runs `/fab-new` to create a new change
- **WHEN** the `.status.yaml` template is applied
- **THEN** the progress map SHALL contain `hydrate: pending` instead of `archive: pending`
- **AND** all other stage keys remain unchanged

#### Scenario: Stage number display

- **GIVEN** a change with `hydrate` as the 6th stage
- **WHEN** `/fab-status` or any skill displays stage position
- **THEN** hydrate SHALL display as `(6/6)`

### Requirement: Stage Numbering

The stage-to-number mapping SHALL be:

| Stage | Number |
|-------|--------|
| brief | 1 |
| spec | 2 |
| tasks | 3 |
| apply | 4 |
| review | 5 |
| hydrate | 6 |

#### Scenario: Hydrate stage position

- **GIVEN** a change at the hydrate stage
- **WHEN** displaying status
- **THEN** the output SHALL show `hydrate (6/6)`

## Execution: Hydrate Behavior

### Requirement: Hydrate as Terminal Pipeline Stage

`/fab-continue` SHALL dispatch to hydrate behavior when the active stage is `review` (with `review: done`) or `hydrate`. Hydrate behavior is the terminal behavior in the pipeline — after hydrate completes, the change folder remains in `fab/changes/` (not moved to archive).

#### Scenario: Normal advance after review passes

- **GIVEN** a change with `review: done`
- **WHEN** `/fab-continue` is invoked
- **THEN** the stage SHALL advance to `hydrate: active` and execute hydrate behavior

#### Scenario: Change folder stays after hydrate

- **GIVEN** hydrate behavior completes successfully
- **WHEN** `.status.yaml` is updated
- **THEN** `hydrate` SHALL be `done`
- **AND** the change folder SHALL remain at `fab/changes/{name}/` (NOT moved)
- **AND** `fab/current` SHALL NOT be cleared

### Requirement: Hydrate Behavior Steps

Hydrate behavior SHALL perform these steps in order:

1. **Final validation** — verify all tasks in `tasks.md` are `[x]` and all checklist items in `checklist.md` are `[x]` (including N/A items)
2. **Concurrent change check** — scan `fab/changes/` for other active changes whose specs reference the same centralized doc paths. If overlap found: warn (not block)
3. **Hydrate into `fab/docs/`** — integrate learnings from `spec.md` into centralized docs:
   - New doc: create domain folder if needed, create from template, populate from spec, update indexes
   - Existing doc: update Requirements section (add new, update changed, remove deprecated), update Design Decisions, add Changelog row, update indexes
   - Extract durable design decisions from spec; skip tactical details
4. **Update `.status.yaml`** — set `hydrate: done`, update `last_updated`

#### Scenario: Hydrate with concurrent changes

- **GIVEN** a change at hydrate stage
- **AND** another active change references the same centralized doc
- **WHEN** hydrate behavior runs
- **THEN** a warning SHALL be displayed: "Change {name} also modifies {doc}."
- **AND** hydrate SHALL proceed (not block)

#### Scenario: Incomplete tasks block hydrate

- **GIVEN** a change with `review: done` but unchecked tasks in `tasks.md`
- **WHEN** hydrate behavior attempts final validation
- **THEN** it SHALL STOP with: "{N} of {total} tasks are incomplete."

### Requirement: Hydrate Preconditions

Hydrate behavior SHALL require `review: done`. If review has not passed, it SHALL STOP with: "Review has not passed. Run /fab-continue to validate implementation first."

#### Scenario: Hydrate without review

- **GIVEN** a change with `review: pending`
- **WHEN** `/fab-continue` attempts to advance to hydrate
- **THEN** it SHALL NOT execute hydrate behavior
- **AND** it SHALL display an error message

### Requirement: Hydrate Context Loading

Hydrate behavior SHALL load:
- `fab/config.yaml`, `fab/constitution.md`, `fab/design/index.md`
- `fab/changes/{name}/spec.md`, `fab/changes/{name}/brief.md`
- `fab/docs/index.md` and specific docs referenced by the brief's Affected Docs section
- Target centralized doc(s) from `fab/docs/`

#### Scenario: Context for hydration

- **GIVEN** a change at hydrate stage with Affected Docs listing `fab-workflow/execution-skills`
- **WHEN** hydrate loads context
- **THEN** it SHALL read `fab/docs/fab-workflow/execution-skills.md` and `fab/docs/fab-workflow/index.md`

## Execution: fab-archive Skill

### Requirement: Standalone Archive Skill

A new `/fab-archive` skill SHALL exist as a standalone command. Its ONLY responsibilities are housekeeping operations:

1. **Move change folder** — `fab/changes/{name}/` → `fab/changes/archive/{name}/`. Create `archive/` if needed. Do NOT rename.
2. **Update archive index** — prepend entry to `fab/changes/archive/index.md` (create with backfill if missing). Format: `- **{folder-name}** — {1-2 sentence description}`. Most-recent-first.
3. **Mark backlog items done** — exact-ID check (always), then keyword scan with interactive confirmation (since fab-archive is always user-invoked)
4. **Clear pointer** — delete `fab/current`

#### Scenario: Successful archive

- **GIVEN** a change with `hydrate: done`
- **WHEN** the user runs `/fab-archive`
- **THEN** the change folder SHALL be moved to `fab/changes/archive/{name}/`
- **AND** `fab/changes/archive/index.md` SHALL be updated
- **AND** `fab/current` SHALL be deleted

#### Scenario: Archive with backlog match

- **GIVEN** a change whose brief contains a backlog ID
- **AND** `fab/backlog.md` has a matching unchecked item
- **WHEN** `/fab-archive` runs
- **THEN** the exact-match item SHALL be marked done automatically
- **AND** keyword-scan candidates SHALL be presented for interactive confirmation

### Requirement: fab-archive Guard

`/fab-archive` SHALL require `hydrate: done` in `.status.yaml`. If hydrate is not done, it SHALL STOP with: "Hydrate has not completed. Run /fab-continue to hydrate docs first."

#### Scenario: Archive before hydrate

- **GIVEN** a change with `review: done` but `hydrate: pending`
- **WHEN** the user runs `/fab-archive`
- **THEN** it SHALL display: "Hydrate has not completed. Run /fab-continue to hydrate docs first."

#### Scenario: Archive before review

- **GIVEN** a change with `review: pending`
- **WHEN** the user runs `/fab-archive`
- **THEN** it SHALL display: "Hydrate has not completed. Run /fab-continue to hydrate docs first."

### Requirement: fab-archive Fail-Safe Order

The steps SHALL execute in this order for safety: folder move first, index update second, backlog third, pointer last. If interrupted mid-operation, the state is recoverable.

#### Scenario: Interrupted archive

- **GIVEN** `/fab-archive` moves the folder but crashes before clearing the pointer
- **WHEN** the user re-runs `/fab-archive`
- **THEN** it SHALL detect the folder is already in archive
- **AND** it SHALL complete the remaining steps (index, backlog, pointer)

### Requirement: fab-archive Does Not Modify .status.yaml Progress

`/fab-archive` SHALL NOT add an `archive` entry to the progress map or modify progress fields. It MAY update `last_updated`.

#### Scenario: Status after archive

- **GIVEN** a change archived by `/fab-archive`
- **WHEN** examining the archived `.status.yaml`
- **THEN** the progress map SHALL show `hydrate: done` as the terminal entry
- **AND** no `archive` key SHALL exist in the progress map

### Requirement: fab-archive Arguments

`/fab-archive` SHALL accept an optional `[change-name]` argument for targeting a specific change. If no argument is provided, it SHALL operate on the active change in `fab/current`. Supports full folder names, partial slug matches, or 4-char IDs.

#### Scenario: Archive non-active change

- **GIVEN** `fab/current` points to change A
- **AND** change B has `hydrate: done`
- **WHEN** the user runs `/fab-archive B`
- **THEN** change B SHALL be archived
- **AND** `fab/current` SHALL be cleared (since it pointed to a different change, clearing is skipped — only clear if the archived change was the active one)

#### Scenario: Archive active change

- **GIVEN** `fab/current` points to change A with `hydrate: done`
- **WHEN** the user runs `/fab-archive`
- **THEN** change A SHALL be archived
- **AND** `fab/current` SHALL be deleted

## Configuration: Stage Definitions

### Requirement: Update config.yaml Stages

`fab/config.yaml` SHALL replace the `archive` stage with `hydrate` in the stages list:

```yaml
stages:
  - id: hydrate
    requires: [review]
```

The `hydrate` stage SHALL have the same position and prerequisites as the former `archive` stage.

#### Scenario: Config stage list

- **GIVEN** a project with `fab/config.yaml`
- **WHEN** reading the stages list
- **THEN** the terminal stage SHALL be `hydrate` (not `archive`)

### Requirement: Update workflow.yaml Schema

`fab/.kit/schemas/workflow.yaml` SHALL:

1. Replace the `archive` stage definition with `hydrate` (id, name, description, commands updated)
2. Update `progression.current_stage.fallback` from `archive` to `hydrate`
3. Update `progression.completion.rule` from `archive` to `hydrate`
4. Update `stage_numbers` to map `hydrate: 6` instead of `archive: 6`

#### Scenario: Workflow schema hydrate stage

- **GIVEN** the `workflow.yaml` schema
- **WHEN** querying stage definitions
- **THEN** `hydrate` SHALL appear as the terminal stage
- **AND** `archive` SHALL NOT appear in the stages list

### Requirement: Update status.yaml Template

`fab/.kit/templates/status.yaml` SHALL use `hydrate: pending` in the progress map instead of `archive: pending`.

#### Scenario: New change from template

- **GIVEN** the status.yaml template
- **WHEN** `/fab-new` creates a new `.status.yaml`
- **THEN** the progress map SHALL contain `hydrate: pending`
- **AND** `archive` SHALL NOT appear

## Cascade: Skill Updates

### Requirement: Update fab-continue

`/fab-continue` (skill file `fab-continue.md`) SHALL:

1. Replace all references to "archive" stage/behavior with "hydrate" in stage guard tables, context loading, output templates, and stage progression
2. Replace the Archive Behavior section with Hydrate Behavior (steps 1-4 only — no folder move, index, backlog, or pointer operations)
3. Update the stage transition table: `review (pass)` → `hydrate: active`, `hydrate (done)` → change complete
4. Remove archive from the Reset Flow valid targets and add hydrate
5. Update Next Steps: after hydrate → `Next: /fab-archive` (instead of `Next: /fab-new`)

#### Scenario: fab-continue after review pass

- **GIVEN** a change with `review: done`
- **WHEN** `/fab-continue` is invoked
- **THEN** it SHALL execute hydrate behavior (not archive behavior)
- **AND** the next step suggestion SHALL be `/fab-archive`

### Requirement: Update fab-ff Terminal Step

`/fab-ff` SHALL use hydrate as its terminal step. After hydrate completes, `/fab-ff` SHALL stop. The change folder remains in `fab/changes/`.

#### Scenario: fab-ff completes at hydrate

- **GIVEN** a change processed by `/fab-ff`
- **WHEN** hydrate completes
- **THEN** `/fab-ff` SHALL stop and output: `Next: /fab-archive`
- **AND** the change folder SHALL remain in `fab/changes/`

### Requirement: Update fab-fff Terminal Step

`/fab-fff` SHALL use hydrate as its terminal step. After hydrate completes, `/fab-fff` SHALL stop. The change folder remains in `fab/changes/`.

#### Scenario: fab-fff completes at hydrate

- **GIVEN** a change processed by `/fab-fff`
- **WHEN** hydrate completes
- **THEN** `/fab-fff` SHALL stop
- **AND** the change folder SHALL remain in `fab/changes/`

### Requirement: Update _context.md

`fab/.kit/skills/_context.md` SHALL:

1. Update the Next Steps Lookup Table: replace archive entries with hydrate entries, add `/fab-archive` entry
2. Update any inline references to the "6-stage pipeline" or "archive" stage to reflect hydrate as the terminal stage

#### Scenario: Next steps after hydrate

- **GIVEN** the Next Steps Lookup Table in `_context.md`
- **WHEN** looking up the step after hydrate
- **THEN** the table SHALL show `Next: /fab-archive`

### Requirement: Update _generation.md

`fab/.kit/skills/_generation.md` SHALL update any references to archive (e.g., "All items MUST pass before /fab-continue (archive)") to reference hydrate instead.

#### Scenario: Checklist generation references

- **GIVEN** the checklist generation procedure in `_generation.md`
- **WHEN** referencing the stage that requires all checklist items to pass
- **THEN** it SHALL reference hydrate (not archive)

## Deprecated Requirements

### Archive as Pipeline Stage

**Reason**: Archive is being removed from the tracked pipeline. Its doc-hydration responsibilities move to the new `hydrate` stage; its housekeeping responsibilities move to the standalone `/fab-archive` skill.

**Migration**: Replace `archive: pending` with `hydrate: pending` in `.status.yaml`. Use `/fab-archive` for manual housekeeping after hydrate completes.

## Design Decisions

1. **Hydrate is a full pipeline stage, archive is not**
   - *Why*: Doc hydration is the logical completion of the agent's work — it closes the feedback loop from implementation back to centralized docs. Folder housekeeping is a user-triggered cleanup action with no bearing on artifact quality.
   - *Rejected*: Both as pipeline stages — would add a 7th stage for marginal benefit. Neither as pipeline stages — would lose the doc hydration automation.

2. **fab-ff and fab-fff stop at hydrate**
   - *Why*: Hydrate represents the meaningful work boundary. Archiving is a housekeeping decision the user makes when they're ready to clean up. Auto-archiving would move folders before the user has reviewed the hydrated docs or decided they're done with the change.
   - *Rejected*: Auto-archive after hydrate — removes user control over when changes leave the active workspace.

3. **fab-archive clears fab/current only for the active change**
   - *Why*: If archiving a non-active change (via change-name argument), clearing the pointer would disrupt the user's active work context. Only clear when the archived change IS the active one.
   - *Rejected*: Always clear — would lose active change context when archiving a different change. Never clear — would leave stale pointer after archiving the active change.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Existing archived changes keep their old `.status.yaml` format (no migration) | Historical data is already archived — no active skill reads these for progression logic |
| 2 | Confident | fab-archive's backlog marking includes both exact-ID check and keyword scan with interactive confirmation | fab-archive is always user-invoked (never auto), so interactive keyword scan is natural and consistent with the existing interactive-only policy |

2 assumptions made (2 confident, 0 tentative).
