# Spec: Add brief as a formal pipeline stage

**Change**: 260212-v5p2-brief-pipeline-stage
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/configuration.md`, `fab/docs/fab-workflow/change-lifecycle.md`, `fab/docs/fab-workflow/planning-skills.md`

## Non-Goals

- Changing how `/fab-new` generates `brief.md` — that behavior is already correct
- Adding brief-specific rules to `config.yaml` `rules:` section — no generation rules needed since brief is created conversationally by `/fab-new`, not by template-driven generation
- Modifying the brief template itself — template content is out of scope
- Changing `/fab-new` `.status.yaml` initialization — the template already has `brief: active` and fab-new uses it; this change is a config alignment, not a fab-new fix
<!-- clarified: fab-new initialization is already correct via template; no skill change needed -->

## Configuration: Brief Stage Definition

### Requirement: Config SHALL include brief as the first pipeline stage

`fab/config.yaml` `stages` list SHALL include a `brief` stage entry as the **first** element, before `spec`. The entry SHALL have:
- `id: brief`
- `generates: brief.md`
- `required: true`
- No `requires` field (brief is the pipeline entry point with no prerequisites)

The total stage count becomes 6: brief, spec, tasks, apply, review, archive.

#### Scenario: Config stages list after change

- **GIVEN** the updated `fab/config.yaml`
- **WHEN** a skill reads the `stages` list
- **THEN** the first entry SHALL have `id: brief` and `generates: brief.md`
- **AND** the `spec` stage SHALL have `requires: [brief]` added to its prerequisites

#### Scenario: Stage count in configuration doc

- **GIVEN** the centralized doc `fab/docs/fab-workflow/configuration.md`
- **WHEN** it describes the `stages` section
- **THEN** it SHALL reference 6 stages and list `brief` as a valid stage ID alongside spec, tasks, apply, review, archive

## Planning Skills: Brief Stage Handling

### Requirement: fab-continue SHALL handle brief as a valid active stage

When `fab-continue` runs with `brief` as the active stage (no argument), it SHALL generate `spec.md` and transition progress from `brief: done` to `spec: active`. This is the standard forward flow from brief to spec.

#### Scenario: Normal forward from brief

- **GIVEN** a change with `progress.brief: active`
- **WHEN** the user runs `/fab-continue`
- **THEN** the skill SHALL generate `spec.md`
- **AND** set `progress.brief: done` and `progress.spec: active`

### Requirement: fab-continue reset SHALL accept brief as a valid target

When called as `/fab-continue brief`, the skill SHALL reset to the brief stage: set `progress.brief: active`, mark all subsequent stages as `pending`, and regenerate `brief.md` in place.

#### Scenario: Reset to brief

- **GIVEN** a change at any stage past brief
- **WHEN** the user runs `/fab-continue brief`
- **THEN** `progress.brief` SHALL be set to `active`
- **AND** all stages after brief (spec, tasks, apply, review, archive) SHALL be set to `pending`
- **AND** `brief.md` SHALL be regenerated in place

### Requirement: fab-ff SHALL skip brief stage

`/fab-ff` SHALL skip the brief stage when fast-forwarding through planning. Since brief is always created by `/fab-new` before `/fab-ff` runs, the fast-forward pipeline starts from spec (or wherever the active stage is).

#### Scenario: Fast-forward starting from brief active

- **GIVEN** a change with `progress.brief: active`
- **WHEN** the user runs `/fab-ff`
- **THEN** the skill SHALL treat brief as the starting point and generate spec onward
- **AND** set `progress.brief: done` before generating spec

#### Scenario: Fast-forward starting from spec active

- **GIVEN** a change with `progress.brief: done` and `progress.spec: active`
- **WHEN** the user runs `/fab-ff`
- **THEN** the skill SHALL start generating from spec (brief already done, no action needed)

### Requirement: fab-fff SHALL skip brief stage

`/fab-fff` SHALL handle the brief stage identically to `/fab-ff` — it is already completed by `/fab-new` and skipped during the pipeline run.

#### Scenario: Full pipeline with brief done

- **GIVEN** a change with `progress.brief: done`
- **WHEN** the user runs `/fab-fff`
- **THEN** the skill SHALL skip brief and proceed with the normal pipeline (ff → apply → review → archive)

### Requirement: fab-clarify SHALL support brief as a clarifiable stage

`/fab-clarify` SHALL accept `brief` as a valid active stage. When the current stage is brief, the skill SHALL scan `brief.md` using the brief-appropriate taxonomy (scope, affected docs, impact, open questions).

#### Scenario: Clarify during brief stage

- **GIVEN** a change with `progress.brief: active`
- **WHEN** the user runs `/fab-clarify`
- **THEN** the skill SHALL scan `brief.md` for gaps and ambiguities
- **AND** present structured questions to refine the brief

### Requirement: fab-switch SHALL update stage number mapping to 6 stages

`/fab-switch` skill's Stage Number Mapping table SHALL be updated from 5 stages to 6 stages, with brief as position 1. The display format changes from `(N/5)` to `(N/6)`.
<!-- clarified: fab-switch stage mapping gap identified during clarify — added requirement -->

#### Scenario: fab-switch stage display after change

- **GIVEN** a change with `progress.brief: active`
- **WHEN** the user runs `/fab-switch` and the change is selected
- **THEN** the status display SHALL show `brief (1/6)` as the stage

#### Scenario: fab-switch stage display for later stages

- **GIVEN** a change with `progress.apply: active`
- **WHEN** the user runs `/fab-switch` and the change is selected
- **THEN** the status display SHALL show `apply (4/6)` as the stage

### Requirement: fab-status SHALL display brief stage correctly

`/fab-status` (and its backing script `fab-status.sh`) SHALL display the brief stage in the progress bar and handle it as a valid stage position.

#### Scenario: Status display during brief stage

- **GIVEN** a change with `progress.brief: active`
- **WHEN** the user runs `/fab-status`
- **THEN** the output SHALL show brief as the current active stage
- **AND** the progress indicators SHALL show 6 stages total

## Change Lifecycle: Stage Count and Migration

### Requirement: Change lifecycle doc SHALL document 6 stages consistently

The `fab/docs/fab-workflow/change-lifecycle.md` document SHALL consistently reference 6 stages throughout. The "Migration Note" section SHALL be updated to remove the instruction to delete `brief:` from the progress map — that instruction is now incorrect since brief IS a formal pipeline stage.

#### Scenario: Migration note after change

- **GIVEN** the updated `fab/docs/fab-workflow/change-lifecycle.md`
- **WHEN** a user reads the Migration Note section
- **THEN** the note SHALL NOT instruct removal of `brief:` from the progress map
- **AND** the note SHALL only address removal of the legacy `stage:` field

### Requirement: Configuration doc SHALL document 6 stages

The `fab/docs/fab-workflow/configuration.md` document SHALL update the `stages` section to reference 6 stages and include `brief` as a valid stage ID.

#### Scenario: Configuration doc stages description

- **GIVEN** the updated `fab/docs/fab-workflow/configuration.md`
- **WHEN** it describes the `stages` config section
- **THEN** it SHALL list the stage IDs as: brief, spec, tasks, apply, review, archive

## Preflight: Brief Stage Recognition

### Requirement: fab-preflight.sh SHALL recognize brief as a valid stage

The preflight script SHALL recognize `brief` as a valid stage when parsing the progress map from `.status.yaml`. When `progress.brief: active`, the script SHALL output `stage: brief`.

#### Scenario: Preflight with brief active

- **GIVEN** a `.status.yaml` with `progress.brief: active`
- **WHEN** `fab-preflight.sh` runs
- **THEN** the output SHALL include `stage: brief`

### Requirement: Preflight SHALL tolerate missing brief entry for backward compatibility

For `.status.yaml` files created before this change (which may lack a `brief:` entry in the progress map), the preflight script SHALL treat the missing entry as implicitly `done`. This ensures existing in-flight changes continue to work without manual migration.
<!-- clarified: tolerance-based migration confirmed — infer brief: done when entry missing, no file surgery -->

#### Scenario: Preflight with legacy status file (no brief entry)

- **GIVEN** a `.status.yaml` with no `brief:` key in the progress map and `spec: active`
- **WHEN** `fab-preflight.sh` runs
- **THEN** the output SHALL include `stage: spec`
- **AND** the progress map in output SHALL include `brief: done` (inferred)

## Deprecated Requirements

### Migration Note: Remove `brief:` from progress map

**Reason**: The instruction in `change-lifecycle.md` to "Remove `brief:` from the progress map — brief is no longer a pipeline stage" is now incorrect. Brief IS a formal pipeline stage.
**Migration**: Replace with guidance that `brief:` SHOULD be present in all `.status.yaml` files. For legacy files missing it, the preflight script infers `brief: done`.

## Design Decisions

1. **Tolerance-based backward compatibility**: Treat missing `brief:` entry as implicitly `done` in preflight
   - *Why*: Least disruptive — existing changes work without manual migration or scripts. Standard defensive programming pattern.
   - *Rejected*: Migration script (requires manual invocation, could fail). Retroactive file updates (modifies files the user hasn't touched, confusing in git history).

2. **`spec` gains `requires: [brief]`**: Adding an explicit dependency from spec to brief in the stages config
   - *Why*: Makes the dependency graph explicit and machine-readable. Skills that traverse the stage graph from config will correctly understand brief→spec ordering.
   - *Rejected*: Implicit ordering by position only (loses the explicit dependency signal that `requires` provides).

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Mark brief as `required: true` | Always created by fab-new, never optional — brief says this explicitly |
| 2 | Confident | Skills should recognize but not generate brief (except reset) | Brief is always created by fab-new; fab-continue transitions from it but doesn't create it |
| 3 | Confident | Tolerance-based migration (missing brief = done) | Least disruptive, standard defensive pattern, avoids file surgery on existing changes |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-12

- **Q**: The brief lists fab-new as an affected file but the spec has no requirement for it. Should the spec include a fab-new initialization requirement?
  **A**: No — covered via Non-Goals clarification. The template already has `brief: active` and fab-new uses it; this is a config alignment, not a fab-new fix.
- **Q**: The spec covers fab-status but not fab-switch. fab-switch has a hardcoded stage number mapping (5 stages). Should the spec include a requirement for updating it?
  **A**: Yes — added requirement for fab-switch to update stage number mapping to 6 stages (brief=1 through archive=6).
- **Q**: Confirm tolerance-based migration approach (missing brief entry = implicitly done)?
  **A**: Accepted recommendation: tolerance approach confirmed — preflight infers `brief: done` when entry is missing, no file surgery needed.
