# Spec: Re-evaluate Checklist Folder Location

**Change**: 260212-ipoe-checklist-folder-location
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/templates.md`, `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/execution-skills.md`, `fab/docs/fab-workflow/change-lifecycle.md`, `fab/docs/fab-workflow/kit-architecture.md`

## Non-Goals

- Migrating existing archived changes — archived changes are historical records; their `checklists/` subfolder remains as-is
- Splitting checklist into multiple files (e.g., `security.md`, `performance.md`) — permanently dismissed; all checklist items go in one file, reintroduce only if concrete need arises
<!-- clarified: expansion alternative permanently dismissed per user confirmation -->

## Change Folder Structure: Checklist Co-location

### Requirement: Checklist at Change Root

The checklist SHALL be stored at `fab/changes/{name}/checklist.md` (change root), co-located with `brief.md`, `spec.md`, `tasks.md`, and `.status.yaml`. All checklist items (functional, behavioral, security, etc.) go into this single file.

The `checklists/` subdirectory SHALL NOT be created for new changes.

#### Scenario: New Change Creation

- **GIVEN** a user runs `/fab-new` to create a new change
- **WHEN** the change folder is initialized
- **THEN** only the change root directory `fab/changes/{name}/` is created
- **AND** no `checklists/` subdirectory is created

#### Scenario: Checklist Generation

- **GIVEN** a change at the tasks stage
- **WHEN** `/fab-continue` or `/fab-ff` generates the quality checklist
- **THEN** the checklist is written to `fab/changes/{name}/checklist.md`
- **AND** `.status.yaml` `checklist.path` reads `checklist.md`

### Requirement: Status Template Default Path

The `.status.yaml` template (`fab/.kit/templates/status.yaml`) SHALL set `checklist.path` to `checklist.md`.

#### Scenario: Template Default

- **GIVEN** a fresh `.status.yaml` initialized from the template
- **WHEN** the `checklist.path` field is read
- **THEN** it contains the value `checklist.md`

## Skill Path Updates

### Requirement: `/fab-new` Directory Creation

`/fab-new` SHALL NOT create a `checklists/` subdirectory when initializing a new change folder. The line creating `fab/changes/{name}/checklists/` SHALL be removed.

#### Scenario: `/fab-new` Initialization

- **GIVEN** a user creates a new change with `/fab-new`
- **WHEN** the change folder structure is created
- **THEN** the folder contains only `fab/changes/{name}/` (no `checklists/` subdirectory)

### Requirement: Generation Partial Path Update

`_generation.md` SHALL reference `fab/changes/{name}/checklist.md` instead of `fab/changes/{name}/checklists/quality.md` in the Checklist Generation Procedure. The directory existence check for `checklists/` SHALL be removed.

#### Scenario: Checklist Generation Procedure

- **GIVEN** the Checklist Generation Procedure in `_generation.md`
- **WHEN** the procedure directs where to write the checklist
- **THEN** it specifies `fab/changes/{name}/checklist.md`
- **AND** does not mention the `checklists/` directory

### Requirement: `/fab-continue` Reset Path Update

When `/fab-continue` resets to the tasks stage, it SHALL regenerate the checklist at `fab/changes/{name}/checklist.md` (not `checklists/quality.md`).

#### Scenario: Tasks Reset

- **GIVEN** a user runs `/fab-continue tasks` to reset the tasks stage
- **WHEN** the checklist is regenerated
- **THEN** it is written to `fab/changes/{name}/checklist.md`

### Requirement: `/fab-review` Checklist Validation Path

`/fab-review` SHALL read and validate the checklist at `fab/changes/{name}/checklist.md`. All references to `checklists/quality.md` SHALL be updated.

#### Scenario: Review Reads Checklist

- **GIVEN** a change at the review stage
- **WHEN** `/fab-review` loads the quality checklist
- **THEN** it reads from `fab/changes/{name}/checklist.md`

#### Scenario: Missing Checklist Error

- **GIVEN** a change at the review stage with no checklist file
- **WHEN** `/fab-review` checks for the checklist
- **THEN** it reports: "No quality checklist found at checklist.md."

### Requirement: `/fab-archive` Checklist Verification Path

`/fab-archive` SHALL verify checklist completion at `fab/changes/{name}/checklist.md`. All references to `checklists/quality.md` SHALL be updated.

#### Scenario: Archive Verifies Checklist

- **GIVEN** a change ready for archiving
- **WHEN** `/fab-archive` performs final validation
- **THEN** it reads checklist items from `fab/changes/{name}/checklist.md`

### Requirement: `/fab-ff` Checklist References

`/fab-ff` SHALL reference `checklist.md` (not `checklists/quality.md`) in all checklist generation and validation paths.

#### Scenario: `/fab-ff` Pipeline Checklist

- **GIVEN** a change running through the `/fab-ff` pipeline
- **WHEN** the tasks stage generates a checklist
- **THEN** it is written to `fab/changes/{name}/checklist.md`
- **AND** the review stage reads from `fab/changes/{name}/checklist.md`

## Active Change Migration

### Requirement: One-Time Migration of Active Changes

During implementation of this change, all active (non-archived) change folders in `fab/changes/` that contain a `checklists/quality.md` file SHALL have that file moved to `checklist.md` at the change root. The empty `checklists/` directory SHALL be removed after the move. This is a one-time migration task, not a runtime fallback.
<!-- clarified: one-time migration for active changes per user request — no fallback lookup in skills -->

#### Scenario: Active Change With Existing Checklist

- **GIVEN** an active change folder `fab/changes/{name}/` containing `checklists/quality.md`
- **WHEN** this change is implemented
- **THEN** `checklists/quality.md` is moved to `fab/changes/{name}/checklist.md`
- **AND** the `checklists/` directory is removed
- **AND** `.status.yaml` `checklist.path` is updated to `checklist.md`

#### Scenario: Active Change Without Checklist

- **GIVEN** an active change folder with no `checklists/quality.md` (checklist not yet generated)
- **WHEN** this change is implemented
- **THEN** no migration action is taken for that change

#### Scenario: Archived Changes Untouched

- **GIVEN** an archived change in `fab/changes/archive/` with `checklists/quality.md`
- **WHEN** this change is implemented
- **THEN** the archived change is not modified

## Design Decisions

1. **Move checklist to change root as `checklist.md`**: Co-locate `checklist.md` alongside other change artifacts at the change root. Renamed from `quality.md` to `checklist.md` — more general, matches the template filename, and reflects that all checklist item types belong in one file.
   - *Why*: All other artifacts (`brief.md`, `spec.md`, `tasks.md`, `.status.yaml`) live at the change root. A `checklists/` subdirectory for a single file violates YAGNI and reduces discoverability. Consistency trumps speculative extensibility.
   - *Rejected*: Keep `checklists/` subfolder — no evidence of multiple types needed. Keep `quality.md` name — too narrow; `checklist.md` is more accurate and matches the template.

2. **Forward-only change**: Only new changes use the root path. Existing archived changes retain their `checklists/` subfolder.
   - *Why*: Archived changes are historical records. Retroactively restructuring them adds risk and complexity with no practical benefit — no skill reads archived checklists.

3. **Strict root lookup with one-time migration**: Skills only look at `checklist.md` at root — no fallback to `checklists/quality.md`. Active changes with existing checklists are migrated once during implementation.
   - *Why*: Fallback lookup adds permanent complexity for a transient problem. A one-time migration cleanly moves active changes to the new path, after which only one path exists.
   - *Rejected*: Fallback lookup (`checklist.md` first, then `checklists/quality.md`) — adds permanent branching logic to every skill for a problem that only exists during the transition.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Move checklist to change root | Strong consistency signal — all other artifacts at root, single-file subfolder violates YAGNI. Brief and user confirmation both lean this way |
| 2 | Certain | Reject expansion alternative (multiple checklist types) | Confirmed by user — permanently dismissed, reintroduce only if concrete need arises |

1 assumption remaining (1 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-12

- **Q**: Should the `checklists/` subfolder concept be permanently dismissed, or kept as a documented future option?
  **A**: Permanently dismiss — reintroduce only if concrete need arises.
- **Q**: Should skills handle both paths as a fallback, or strictly only look at `checklist.md` at root?
  **A**: Strictly root only. During implementation, scan `fab/changes/` and do a one-time move for active changes that have `checklists/quality.md`.
- **Q**: (User-initiated) Rename from `quality.md` to `checklist.md`.
  **A**: All references updated. `checklist.md` is more general, matches the template filename, and reflects that all checklist item types belong in one file.
