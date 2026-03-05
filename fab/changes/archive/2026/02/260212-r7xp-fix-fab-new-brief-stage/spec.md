# Spec: Fix fab-new premature brief completion

**Change**: 260212-r7xp-fix-fab-new-brief-stage
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/context-loading.md`

## Non-Goals

- Changing `/fab-continue`'s brief → spec transition logic — it already handles this correctly
- Modifying `/fab-ff` or `/fab-fff` behavior — they invoke `/fab-continue` internally and inherit the fix
- Altering brief artifact content or structure — only the stage lifecycle is affected

## fab-new: Remove Stage Transition

### Requirement: fab-new SHALL NOT transition brief stage

`/fab-new` SHALL end with `progress.brief` set to `active`. It MUST NOT set `progress.brief` to `done` or `progress.spec` to `active`. The stage transition from brief to spec is the responsibility of `/fab-continue`.

#### Scenario: Normal brief generation

- **GIVEN** a user runs `/fab-new "some description"`
- **WHEN** the brief artifact is generated and written to `fab/changes/{name}/brief.md`
- **THEN** `.status.yaml` SHALL have `progress.brief: active` and all other stages `pending`
- **AND** the confidence block SHALL be written with actual SRAD counts (Step 7 unchanged)
- **AND** `last_updated` SHALL be set to the current timestamp

#### Scenario: Brief generation with --switch flag

- **GIVEN** a user runs `/fab-new "some description" --switch`
- **WHEN** the brief is generated and `/fab-switch` is called internally
- **THEN** `.status.yaml` SHALL still have `progress.brief: active`
- **AND** `/fab-switch` SHALL write `fab/current` and handle branch integration
- **AND** the stage transition SHALL NOT occur as part of `/fab-new`

#### Scenario: User runs /fab-continue after /fab-new

- **GIVEN** `/fab-new` has completed with `progress.brief: active`
- **WHEN** the user runs `/fab-continue`
- **THEN** `/fab-continue` SHALL detect `brief` as the active stage
- **AND** generate `spec.md`
- **AND** transition `progress.brief` to `done` and `progress.spec` to `active`

### Requirement: Step 8 SHALL be removed and Step 9 renumbered

The "Mark Brief Complete" step (current Step 8 in `fab/.kit/skills/fab-new.md`) SHALL be removed entirely. The current Step 9 ("Activate Change via `/fab-switch`") SHALL be renumbered to Step 8.

#### Scenario: Step removal in fab-new.md

- **GIVEN** the current `fab/.kit/skills/fab-new.md` contains Step 8 "Mark Brief Complete" (lines 206-213)
- **WHEN** the fix is applied
- **THEN** Step 8 ("Mark Brief Complete") SHALL be removed — including the stage transition logic (`progress.brief: done`, `progress.spec: active`) and the `last_updated` write
- **AND** the current Step 9 ("Activate Change via `/fab-switch`") SHALL be renumbered to Step 8
- **AND** all internal references to step numbers SHALL remain consistent

## _context.md: Update Next Steps Table

### Requirement: Next Steps table SHALL reflect brief active state

The Next Steps lookup table in `fab/.kit/skills/_context.md` SHALL show `brief active` (not `brief done`) as the stage reached after `/fab-new`.

#### Scenario: Next Steps table entry for /fab-new

- **GIVEN** the Next Steps table in `fab/.kit/skills/_context.md`
- **WHEN** the entry for `/fab-new` is read
- **THEN** the "Stage reached" column SHALL show `brief active`
- **AND** the "Next line" column SHALL show the default (no-switch) next command: `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff`

## planning-skills.md: Update /fab-new Documentation

### Requirement: Centralized doc SHALL reflect new behavior

The `/fab-new` section in `fab/docs/fab-workflow/planning-skills.md` SHALL describe the updated change initialization sequence without the "Mark brief complete" step.

#### Scenario: Change Initialization list in planning-skills.md

- **GIVEN** the "Change Initialization" numbered list in `fab/docs/fab-workflow/planning-skills.md`
- **WHEN** the list is read
- **THEN** it SHALL NOT contain a step about marking the brief complete or transitioning stage progress
- **AND** it SHALL describe: create directory, initialize `.status.yaml` (with `brief: active`), generate `brief.md`, conditionally call `/fab-switch`

## Deprecated Requirements

### Mark Brief Complete in /fab-new

**Reason**: Step 8 prematurely transitions `progress.brief` to `done` within `/fab-new`, bypassing the intended user review flow. The transition belongs in `/fab-continue`, which already implements it.
**Migration**: `/fab-continue` handles the `brief → spec` transition (stage guard table: `brief` active → generate `spec.md` → set `brief: done`, `spec: active`). No new code needed.

## Design Decisions

1. **Remove Step 8 entirely rather than making it conditional**: The step's logic (stage transition) is already fully implemented in `/fab-continue`. Keeping a conditional version in `/fab-new` would create two code paths for the same transition, violating DRY and increasing drift risk.
   - *Why*: Single responsibility — `/fab-new` creates artifacts, `/fab-continue` manages stage transitions.
   - *Rejected*: Making Step 8 conditional on user confirmation — adds complexity for no benefit since `/fab-continue` already exists for this purpose.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Output examples in fab-new.md keep "Brief complete." text unchanged | "Brief complete." refers to the artifact being generated, not the stage being marked done — consistent with the wording used by other skills for artifact creation |
| 2 | Confident | Step 7 (confidence score computation) is unaffected | Step 7 writes the `confidence` block to `.status.yaml` and does not touch `progress.*` fields — orthogonal to stage transitions |
| 3 | Confident | `fab/docs/fab-workflow/context-loading.md` does not need direct content changes | The Next Steps table lives in `fab/.kit/skills/_context.md` (source), not in the centralized doc; the doc describes the loading convention in general terms that aren't affected by this change |

3 assumptions made (3 confident, 0 tentative).
