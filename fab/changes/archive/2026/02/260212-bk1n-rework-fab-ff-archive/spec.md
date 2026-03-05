# Spec: Rework fab-ff to go all the way to archive

**Change**: 260212-bk1n-rework-fab-ff-archive
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/execution-skills.md`

## Non-Goals

- Changing `/fab-fff` behavior — it remains the confidence-gated fully autonomous pipeline
- Adding a confidence gate to `/fab-ff` — it stays ungated
- Changing auto-clarify behavior during planning stages — existing spec/tasks auto-clarify continues as-is

## fab-ff: Extended Pipeline Scope

### Requirement: Full Pipeline Execution

`/fab-ff` SHALL execute the complete pipeline from the current stage through archive: planning (spec, tasks) → apply → review → archive. This extends the current behavior which stops after planning stages.

#### Scenario: Clean full pipeline from brief

- **GIVEN** a change with `brief: active` and no blocking issues
- **WHEN** the user invokes `/fab-ff`
- **THEN** the skill generates spec.md, runs auto-clarify on spec, generates tasks.md, runs auto-clarify on tasks, generates the quality checklist, executes tasks via apply, validates via review, hydrates docs via archive, and moves the change to archive
- **AND** the output includes section headers for each phase (Planning, Implementation, Review, Archive)

#### Scenario: Resume from mid-pipeline

- **GIVEN** a change with `spec: done`, `tasks: done`, `apply: done`, `review: pending`
- **WHEN** the user invokes `/fab-ff`
- **THEN** the skill skips planning and apply stages, executes review, then archive
- **AND** the output notes which stages were skipped

### Requirement: Execution Stage Invocation

`/fab-ff` SHALL invoke apply, review, and archive using the same behavior as their standalone skill invocations (`/fab-apply`, `/fab-review`, `/fab-archive`). The skill SHALL NOT inline or duplicate their logic.

#### Scenario: Apply phase uses standard fab-apply behavior

- **GIVEN** planning stages are complete (spec: done, tasks: done)
- **WHEN** `/fab-ff` reaches the apply phase
- **THEN** it executes tasks from tasks.md in dependency order, runs tests after each task, marks tasks `[x]` on completion, and updates `.status.yaml` — identical to standalone `/fab-apply`

#### Scenario: Archive phase uses standard fab-archive behavior

- **GIVEN** review has passed
- **WHEN** `/fab-ff` reaches the archive phase
- **THEN** it performs concurrent change check, hydrates docs, updates status, moves to archive, updates archive index, and clears `fab/current` — identical to standalone `/fab-archive`

### Requirement: No Confidence Gate

`/fab-ff` SHALL NOT perform a confidence gate check. Any change with a completed brief can be fast-forwarded regardless of confidence score.

#### Scenario: Low-confidence change runs successfully

- **GIVEN** a change with `confidence.score: 1.5`
- **WHEN** the user invokes `/fab-ff`
- **THEN** the skill proceeds normally (frontloads questions, generates artifacts, executes pipeline)
- **AND** no confidence-related warning or gate is displayed

## fab-ff: Interactive Clarification Stops

### Requirement: Bail on Blocking Issues During Planning

`/fab-ff` SHALL continue to bail when auto-clarify finds blocking issues during planning stages (spec, tasks). This preserves existing behavior.

#### Scenario: Blocking issue in spec auto-clarify

- **GIVEN** auto-clarify on spec.md finds 1 blocking issue
- **WHEN** the auto-clarify result is interpreted
- **THEN** the pipeline stops, reports the blocking issue(s), and suggests `Run /fab-clarify to resolve these interactively, then /fab-ff to resume.`
- **AND** `.status.yaml` reflects `spec: done`, `tasks: pending`

### Requirement: Interactive Review Failure Handling

On review failure, `/fab-ff` SHALL present the interactive rework options (fix code, revise tasks, revise spec) — the same options that standalone `/fab-review` offers. This is the key behavioral difference from `/fab-fff`, which bails immediately on review failure.

#### Scenario: Review fails during fab-ff pipeline

- **GIVEN** apply has completed and review finds failures
- **WHEN** `/fab-ff` reaches the review failure state
- **THEN** the skill presents the review failure details and the interactive rework menu:
  - **Fix code** → unchecks affected tasks, re-runs apply
  - **Revise tasks** → user edits tasks.md, re-runs apply
  - **Revise spec** → resets to spec stage via `/fab-continue spec`
- **AND** the user selects a rework option and the pipeline resumes accordingly

#### Scenario: Review passes during fab-ff pipeline

- **GIVEN** apply has completed successfully
- **WHEN** review validates all checks pass
- **THEN** the pipeline proceeds to archive without user interaction

### Requirement: Stop on Apply Failure

If a task fails during apply and cannot be resolved by the agent, `/fab-ff` SHALL stop and surface the failure to the user for interactive resolution.

#### Scenario: Unresolvable task failure during apply

- **GIVEN** the pipeline is in the apply phase
- **WHEN** a task fails and the agent cannot fix it after running tests
- **THEN** the pipeline stops, reports which task failed and why, and suggests the user investigate and re-run `/fab-ff` to resume

## fab-ff: Resumability

### Requirement: Full Pipeline Resumability

`/fab-ff` SHALL be resumable across all stages. Re-running after any stop (bail, failure, interruption) picks up from the first incomplete stage by checking the `progress` map.

#### Scenario: Resume after review failure rework

- **GIVEN** a previous `/fab-ff` run stopped at review failure, user chose "Fix code", tasks were re-applied
- **WHEN** the user invokes `/fab-ff` again
- **THEN** the skill checks progress, finds apply done, runs review, and continues to archive if review passes

#### Scenario: Resume after interruption during apply

- **GIVEN** a previous `/fab-ff` run was interrupted during apply (some tasks `[x]`, some `[ ]`)
- **WHEN** the user invokes `/fab-ff` again
- **THEN** the skill skips completed planning stages and resumes apply from the first unchecked task

## fab-ff: Output Format

### Requirement: Phased Output with Section Headers

`/fab-ff` SHALL use section headers to delineate each phase of the pipeline, consistent with the `/fab-fff` output format.

#### Scenario: Full pipeline output

- **GIVEN** a clean pipeline run with no issues
- **WHEN** all stages complete
- **THEN** the output uses these section headers in order:
  - `--- Planning ---` (contains spec, auto-clarify, tasks, auto-clarify, checklist)
  - `--- Implementation (fab-apply) ---`
  - `--- Review (fab-review) ---`
  - `--- Archive (fab-archive) ---`
- **AND** ends with `Pipeline complete. Change archived.`
- **AND** the next steps line is `Next: /fab-new <description> (start next change)`

#### Scenario: Partial pipeline output (planning only before bail)

- **GIVEN** auto-clarify finds blocking issues during spec
- **WHEN** the pipeline bails
- **THEN** only the `--- Planning ---` section header appears
- **AND** the bail message and next steps are shown

## fab-fff: Comparison Update

### Requirement: Updated Comparison Table

The comparison table in `/fab-ff` and `/fab-fff` skill files SHALL be updated to reflect that both are now full-pipeline commands, differentiated by their interaction model.

#### Scenario: Comparison table accurately reflects new behavior

- **GIVEN** the updated skill files
- **WHEN** a user reads the comparison table
- **THEN** the table shows:
  - `/fab-ff`: Full pipeline, stops for interactive clarification, no confidence gate, interactive rework on review failure
  - `/fab-fff`: Full pipeline, auto-clarifies without stopping, confidence gate >= 3.0, immediate bail on review failure

## Deprecated Requirements

### fab-ff Stops After Planning

**Reason**: `/fab-ff` previously stopped after generating spec and tasks (planning stages only). The new behavior extends through apply, review, and archive.
**Migration**: The pipeline now continues through all stages. Users who want planning-only can use `/fab-continue` repeatedly.

## Design Decisions

1. **Interactive rework on review failure**: fab-ff presents the same rework menu as standalone `/fab-review`
   - *Why*: fab-ff's identity is "full pipeline with interactive stops." Bailing like fab-fff would remove the key differentiator. Users chose fab-ff *because* they want the ability to intervene.
   - *Rejected*: Bail immediately like fab-fff — removes the interactive value proposition.

2. **Reuse execution skill behavior via internal invocation**: fab-ff invokes the same behavior as standalone apply/review/archive rather than inlining
   - *Why*: Consistent with fab-fff's existing pattern. Prevents drift between standalone and pipelined behavior. Single source of truth for each stage's behavior.
   - *Rejected*: Inlining execution logic — creates maintenance burden and inevitable drift.

3. **Phased output format with section headers**: Use `--- Phase (skill) ---` headers consistent with fab-fff
   - *Why*: Users scanning long output need clear phase boundaries. Consistency with fab-fff makes the mental model transferable.
   - *Rejected*: Flat output without headers — hard to navigate for multi-stage runs.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | fab-ff presents interactive rework menu on review failure | Brief says "stops for user clarification when needed" — interactive menu is the natural expression of this |
| 2 | Confident | Apply failure stops pipeline for user intervention | Brief says "stop and bail if clarifications are needed during execution stages" — consistent with interactive nature |
| 3 | Confident | Output format follows fab-fff section header pattern | No explicit format in brief, but consistency with the sibling command is the obvious choice |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.
