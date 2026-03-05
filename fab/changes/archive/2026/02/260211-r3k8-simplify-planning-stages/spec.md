# Spec: Simplify Planning Stages

**Change**: 260211-r3k8-simplify-planning-stages
**Created**: 2026-02-11
**Affected docs**: `fab-workflow/planning-skills`, `fab-workflow/change-lifecycle`, `fab-workflow/configuration`, `fab-workflow/templates`, `fab-workflow/kit-architecture`, `fab-workflow/specs-index`, `fab-workflow/context-loading`, `fab-workflow/clarify`

## Stage Pipeline: Reduction from 7 to 6 Stages

### Requirement: 6-Stage Pipeline

The Fab pipeline SHALL consist of 6 stages: `brief → spec → tasks → apply → review → archive`. The `plan` stage is removed. The `proposal` stage is renamed to `brief`. The `specs` stage is renamed to `spec`.

#### Scenario: Pipeline Progression

- **GIVEN** a new change is created
- **WHEN** the change progresses through all stages
- **THEN** the stages are: brief, spec, tasks, apply, review, archive
- **AND** no plan stage exists in the progression

#### Scenario: Stage Numbering

- **GIVEN** a change at any stage
- **WHEN** `/fab-status` displays stage position
- **THEN** it shows `(N/6)` instead of `(N/7)`
- **AND** stage numbers are: brief=1, spec=2, tasks=3, apply=4, review=5, archive=6

### Requirement: Stage ID Renames

All stage identifiers SHALL use the new names: `brief` (was `proposal`), `spec` (was `specs`). The `plan` stage ID SHALL be removed entirely.

#### Scenario: .status.yaml Progress Keys

- **GIVEN** a new change is initialized
- **WHEN** `.status.yaml` is created
- **THEN** the progress map SHALL contain keys: `brief`, `spec`, `tasks`, `apply`, `review`, `archive`
- **AND** SHALL NOT contain keys: `proposal`, `specs`, `plan`

#### Scenario: Config Stage Definitions

- **GIVEN** `fab/config.yaml` defines the stage pipeline
- **WHEN** a user or skill reads the `stages:` block
- **THEN** stage IDs are `brief`, `spec`, `tasks`, `apply`, `review`, `archive`
- **AND** no `plan` stage entry exists
- **AND** `spec` requires `[brief]`
- **AND** `tasks` requires `[spec]`

## Plan Stage: Removal

### Requirement: Plan Stage Eliminated

The `plan` stage SHALL be removed from the pipeline. There SHALL be no `plan.md` artifact, no Plan Generation Procedure, and no plan-skip decision logic.

#### Scenario: /fab-continue No Longer Offers Plan Skip

- **GIVEN** spec stage is done
- **WHEN** `/fab-continue` advances to the next stage
- **THEN** it proceeds directly to tasks generation
- **AND** does not prompt "skip plan?"

#### Scenario: /fab-ff No Longer Generates Plan

- **GIVEN** `/fab-ff` is fast-forwarding through planning stages
- **WHEN** it finishes generating spec
- **THEN** it proceeds directly to tasks (after auto-clarify on spec)
- **AND** does not evaluate whether a plan is warranted

### Requirement: Spec Absorbs Design Decisions

The spec artifact (`spec.md`) MAY include a `## Design Decisions` section when the change involves architectural choices. The agent SHALL include this section when:
- The change involves choosing between technologies or patterns
- The approach is non-obvious and the rationale should be captured
- Technical research was needed to make the decision

The agent SHALL omit this section for straightforward changes.

#### Scenario: Complex Change With Design Decisions

- **GIVEN** a change that requires choosing between approaches (e.g., WebSockets vs SSE)
- **WHEN** the agent generates `spec.md`
- **THEN** `spec.md` includes a `## Design Decisions` section
- **AND** each decision includes: decision, rationale, rejected alternatives

#### Scenario: Straightforward Change Without Design Decisions

- **GIVEN** a change with an obvious implementation approach
- **WHEN** the agent generates `spec.md`
- **THEN** `spec.md` does NOT include a `## Design Decisions` section

## Skill Landing Points

### Requirement: /fab-new Lands on Brief

`/fab-new` SHALL produce `brief.md` (was `proposal.md`) and mark `progress.brief` as `done`. The artifact content and structure remain the same as the current proposal template — only the filename and stage ID change.

#### Scenario: /fab-new Creates Brief

- **GIVEN** a user invokes `/fab-new "add dark mode"`
- **WHEN** the change folder is created
- **THEN** the artifact is `brief.md` (not `proposal.md`)
- **AND** `.status.yaml` shows `stage: brief` and `progress.brief: done`

### Requirement: /fab-discuss Produces Both Brief and Spec

`/fab-discuss` SHALL produce both `brief.md` and `spec.md` when creating a new change, marking both `progress.brief` and `progress.spec` as `done`. The brief is a summary snapshot of the conversation; the spec is the full structured requirements.

#### Scenario: /fab-discuss New Change Mode

- **GIVEN** a user invokes `/fab-discuss` with no active change
- **WHEN** the conversation concludes and the proposal is finalized
- **THEN** both `brief.md` and `spec.md` are written to the change folder
- **AND** `.status.yaml` shows `stage: spec`, `progress.brief: done`, `progress.spec: done`

#### Scenario: /fab-discuss Next Step After New Change

- **GIVEN** `/fab-discuss` has created a new change with both brief and spec
- **WHEN** the user runs `/fab-continue`
- **THEN** it advances to tasks generation (not spec generation, since spec is already done)

#### Scenario: /fab-discuss Refine Mode Unchanged

- **GIVEN** a user invokes `/fab-discuss` on an active change with an existing brief
- **WHEN** the conversation concludes
- **THEN** only `brief.md` is updated (same behavior as current refine mode, adapted to the new artifact name)
- **AND** no `spec.md` is generated or modified

## Root Folder Rename: fab/specs/ → fab/design/

### Requirement: Specs Directory Renamed to Design

The project-level pre-implementation design directory SHALL be renamed from `fab/specs/` to `fab/design/`. All references throughout the codebase SHALL be updated.

#### Scenario: Constitution References

- **GIVEN** `fab/constitution.md` references `fab/specs/`
- **WHEN** the rename is applied
- **THEN** all references read `fab/design/`

#### Scenario: Context Loading

- **GIVEN** `_context.md` lists `fab/specs/index.md` in the "Always Load" layer
- **WHEN** the rename is applied
- **THEN** the reference reads `fab/design/index.md`

#### Scenario: Index File Content

- **GIVEN** `fab/specs/index.md` contains boilerplate about "pre-implementation specs"
- **WHEN** the directory is renamed
- **THEN** `fab/design/index.md` uses consistent terminology ("design" not "specs")
- **AND** cross-references to `fab/docs/index.md` remain valid

#### Scenario: Centralized Docs References

- **GIVEN** the docs `specs-index.md` describes the `fab/specs/` directory
- **WHEN** the rename is applied
- **THEN** the doc is updated to reference `fab/design/` throughout
- **AND** the doc filename MAY be renamed to `design-index.md` for consistency

## Template Changes

### Requirement: Proposal Template Renamed to Brief

The template file `fab/.kit/templates/proposal.md` SHALL be renamed to `fab/.kit/templates/brief.md`. The template content remains structurally identical.

#### Scenario: Template Exists at New Path

- **GIVEN** a fresh `.kit/` installation
- **WHEN** a skill looks for the brief template
- **THEN** it finds `fab/.kit/templates/brief.md`
- **AND** `fab/.kit/templates/proposal.md` does not exist

### Requirement: Plan Template Removed

The template file `fab/.kit/templates/plan.md` SHALL be removed. No plan template is needed.

#### Scenario: No Plan Template

- **GIVEN** a fresh `.kit/` installation
- **WHEN** the templates directory is listed
- **THEN** it contains: `brief.md`, `spec.md`, `tasks.md`, `checklist.md`, `status.yaml`
- **AND** `plan.md` does not exist

## Shared Generation Partial

### Requirement: Plan Generation Procedure Removed

The Plan Generation Procedure in `fab/.kit/skills/_generation.md` SHALL be removed entirely.

#### Scenario: Generation Partial Contents

- **GIVEN** `_generation.md` defines shared generation procedures
- **WHEN** the change is applied
- **THEN** it contains: Spec Generation Procedure, Tasks Generation Procedure, Checklist Generation Procedure
- **AND** does not contain a Plan Generation Procedure

### Requirement: Spec Generation Procedure Updated

The Spec Generation Procedure SHALL be updated to optionally include a `## Design Decisions` section. The procedure SHALL instruct the agent to evaluate whether design decisions are warranted and include the section when appropriate.

## Downstream Skill Updates

### Requirement: /fab-continue Stage Guards Updated

`/fab-continue` stage guard logic SHALL use the new stage IDs. The plan-skip decision logic SHALL be removed. After spec is done, `/fab-continue` SHALL proceed directly to tasks.

#### Scenario: Continue After Spec

- **GIVEN** a change with `stage: spec` and `progress.spec: done`
- **WHEN** `/fab-continue` is invoked
- **THEN** it generates `tasks.md` directly
- **AND** does not ask about skipping plan

### Requirement: /fab-ff Updated Pipeline

`/fab-ff` SHALL generate `spec → tasks` (with auto-clarify between). The plan generation step and plan-decision logic SHALL be removed.

#### Scenario: Fast-Forward Pipeline

- **GIVEN** a change with a completed brief
- **WHEN** `/fab-ff` is invoked
- **THEN** it generates: spec → auto-clarify → tasks → auto-clarify
- **AND** no plan step exists in the pipeline

### Requirement: Shared Preamble (_context.md) Updated

`fab/.kit/skills/_context.md` SHALL be updated to reflect new stage names throughout:
- Context Loading section: `fab/specs/index.md` → `fab/design/index.md`
- Next Steps Convention table: all `proposal`/`specs` references → `brief`/`spec`, remove plan-related rows
- SRAD Autonomy Framework: skill table and examples updated to new stage names
- Confidence Scoring lifecycle table: updated to new stage names

#### Scenario: Next Steps Table References

- **GIVEN** `_context.md` contains the Next Steps Convention table
- **WHEN** the change is applied
- **THEN** rows reference `brief` and `spec` (not `proposal` or `specs`)
- **AND** no rows reference `plan`

### Requirement: /fab-clarify Stage Validation Updated

`/fab-clarify` SHALL accept stages: `brief`, `spec`, `tasks`. It SHALL NOT accept `proposal`, `specs`, or `plan`.

#### Scenario: Clarify on Brief Stage

- **GIVEN** a change at `stage: brief`
- **WHEN** `/fab-clarify` is invoked
- **THEN** it scans `brief.md` for gaps
- **AND** does not reject the stage

### Requirement: /fab-continue Reset Targets Updated

`/fab-continue <stage>` reset targets SHALL be: `spec`, `tasks`. Resetting to `brief` SHALL be rejected with "Cannot reset to brief. Run /fab-new to start a new change instead."

#### Scenario: Reset to Spec

- **GIVEN** a change at tasks stage
- **WHEN** `/fab-continue spec` is invoked
- **THEN** spec is regenerated, tasks are invalidated
- **AND** no plan invalidation occurs (plan does not exist)

## Migration

### Requirement: No Backward Compatibility

This change SHALL NOT include migration logic for existing in-progress changes that use old stage names. Any existing changes with old stage names (`proposal`, `specs`, `plan`) SHALL be manually updated or abandoned.

#### Scenario: Old Change With Old Stage Names

- **GIVEN** an existing change folder with `stage: proposal` in `.status.yaml`
- **WHEN** the updated skills are deployed
- **THEN** the change is non-functional until `.status.yaml` is manually edited to use new stage names
- **AND** no automated migration is provided

## Deprecated Requirements

### `plan` Stage

**Reason**: Plan stage provides insufficient value for most changes. Architectural decisions are absorbed into the spec's optional `## Design Decisions` section. File change lists and execution ordering belong in tasks.
**Migration**: Spec generation optionally includes `## Design Decisions` when warranted. Tasks generation handles file change lists and dependency ordering.

### `proposal` and `specs` Stage IDs

**Reason**: Renamed to `brief` and `spec` respectively for clarity and to resolve naming collisions with `fab/specs/` (now `fab/design/`).
**Migration**: All stage ID references updated throughout config, status files, skill files, docs, and templates.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | /fab-discuss refine mode updates brief.md only (not spec.md) | Refine mode operates on the entry artifact; spec refinement would use /fab-clarify |
| 2 | Confident | spec.md Design Decisions section uses same format as current plan.md Decisions section | Reuses existing proven format (decision, rationale, rejected alternatives) |
| 3 | Confident | Tasks generation references spec directly (no plan fallback logic needed) | Plan no longer exists; tasks always derive from spec |
| 4 | Confident | specs-index doc renamed to design-index for consistency | Follows the directory rename; avoids stale naming |

4 assumptions made (4 confident, 0 tentative).

Confidence: 4.4/5.0

## Clarifications

### Session 2026-02-11

- **Q**: Migration path for existing in-progress changes with old stage names?
  **A**: No migration needed — old changes manually updated or abandoned. Young project, no external users.
- **Q**: Missing affected docs: context-loading and clarify?
  **A**: Accepted recommendation: add both to affected docs list.
- **Q**: Should _context.md be explicitly called out in Impact?
  **A**: Yes — add to Impact section explicitly alongside _generation.md.
