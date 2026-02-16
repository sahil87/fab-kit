# Spec: Redefine fab-ff and fab-fff Scope

**Change**: 260215-237b-DEV-1027-redefine-ff-fff-scope
**Created**: 2026-02-16
**Affected memory**: `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/change-lifecycle.md`

## Non-Goals

- Changing `calc-score.sh` or any shell scripts — the gate mechanism is reused as-is, just invoked by a different skill
- Changing the confidence scoring formula or thresholds — the dynamic per-type thresholds (bugfix=2.0, feature/refactor=3.0, architecture=4.0) transfer unchanged
- Modifying `/fab-continue` or `/fab-clarify` behavior — these skills are unaffected
- Changing the SRAD framework or Assumptions table format

## Skill Files: `/fab-fff` Redefinition

### Requirement: `/fab-fff` SHALL be the full pipeline command with no confidence gate

`/fab-fff` SHALL run the entire Fab pipeline from the current stage through hydrate in a single invocation. The confidence gate SHALL be removed. The minimum prerequisite is that `intake.md` exists (spec pending or later). `/fab-fff` SHALL be callable from any stage at or after intake and SHALL skip stages already marked `done`.

#### Scenario: Full pipeline from intake
- **GIVEN** a change with `intake: active` and all other stages `pending`
- **WHEN** the user runs `/fab-fff`
- **THEN** the pipeline executes spec → tasks → apply → review → hydrate without gating on confidence
- **AND** intake is transitioned to `done` before spec generation begins

#### Scenario: Resuming mid-pipeline
- **GIVEN** a change with `spec: done`, `tasks: done`, `apply: active`
- **WHEN** the user runs `/fab-fff`
- **THEN** spec and tasks are skipped
- **AND** the pipeline resumes from apply through hydrate

#### Scenario: No confidence gate
- **GIVEN** a change with `confidence.score: 0.0` (no spec yet)
- **WHEN** the user runs `/fab-fff`
- **THEN** the pipeline proceeds without checking the confidence score

### Requirement: `/fab-fff` SHALL frontload questions before planning

`/fab-fff` SHALL scan the intake for ambiguities across all remaining planning stages, collect Unresolved decisions into a single batch, and ask once before generating any artifacts. At most one Q&A round.

#### Scenario: Unresolved decisions exist
- **GIVEN** the intake contains ambiguities that score as Unresolved via SRAD
- **WHEN** `/fab-fff` begins
- **THEN** all Unresolved decisions are presented as a numbered list
- **AND** the user's answers are incorporated into subsequent artifact generation

#### Scenario: No unresolved decisions
- **GIVEN** the intake is clear with no Unresolved decisions
- **WHEN** `/fab-fff` begins
- **THEN** planning proceeds immediately without questions

### Requirement: `/fab-fff` SHALL interleave auto-clarify between planning stages

`/fab-fff` SHALL invoke `/fab-clarify` with `[AUTO-MODE]` prefix between spec and tasks generation: `spec → auto-clarify → tasks → auto-clarify`. If auto-clarify finds blocking issues that cannot be resolved autonomously, the pipeline SHALL bail with a message suggesting `/fab-clarify` then `/fab-fff`.

#### Scenario: Auto-clarify finds no blockers
- **GIVEN** spec generation completed successfully
- **WHEN** auto-clarify runs on the spec
- **THEN** `blocking: 0` is returned
- **AND** the pipeline continues to tasks generation

#### Scenario: Auto-clarify finds blockers
- **GIVEN** spec generation completed with unresolvable gaps
- **WHEN** auto-clarify runs on the spec and returns `blocking > 0`
- **THEN** the pipeline stops
- **AND** the message `Run /fab-clarify to resolve these, then /fab-fff to resume.` is displayed

### Requirement: `/fab-fff` SHALL present interactive rework menu on review failure

When review fails during `/fab-fff`, the skill SHALL present the same interactive rework menu as the current `/fab-ff`: fix code (uncheck tasks with `<!-- rework: reason -->`), revise tasks, or revise spec (reset via `/fab-continue spec`).

#### Scenario: Review failure with interactive rework
- **GIVEN** the review step identifies spec mismatches or code quality issues
- **WHEN** the review verdict is "fail"
- **THEN** the user is presented with three rework options: fix code, revise tasks, revise spec
- **AND** the pipeline does NOT bail automatically

### Requirement: `/fab-fff` driver identification

All `.status.yaml` transitions within `/fab-fff` SHALL pass `fab-fff` as the driver parameter. `log-command` SHALL record `fab-fff`. Planning stages internally use the same generation procedures as `/fab-ff` (frontloaded questions, auto-clarify interleaving).

#### Scenario: Status tracking
- **GIVEN** `/fab-fff` is executing the pipeline
- **WHEN** a stage transition occurs (e.g., spec → tasks)
- **THEN** `lib/stageman.sh transition` is called with `fab-fff` as driver
- **AND** `stage_metrics` records `fab-fff` as the driver for each stage

## Skill Files: `/fab-ff` Redefinition

### Requirement: `/fab-ff` SHALL be the fast-forward-from-spec command with confidence gate

`/fab-ff` SHALL run the pipeline from the current stage through hydrate, with a minimum prerequisite that spec is `active` or later (spec.md and confidence score already exist). The confidence gate SHALL use the dynamic threshold from `calc-score.sh --check-gate` (bugfix=2.0, feature/refactor=3.0, architecture=4.0; default 3.0).

#### Scenario: Gate passes
- **GIVEN** a change with `spec: active`, `confidence.score: 3.5`, `change_type: feature`
- **WHEN** the user runs `/fab-ff`
- **THEN** the gate passes (3.5 > 3.0 for feature)
- **AND** the pipeline proceeds from spec through hydrate

#### Scenario: Gate fails
- **GIVEN** a change with `spec: active`, `confidence.score: 2.5`, `change_type: feature`
- **WHEN** the user runs `/fab-ff`
- **THEN** the pipeline aborts
- **AND** a message is displayed: `Confidence is 2.5 of 5.0 (need > 3.0 for feature). Run /fab-clarify to resolve, then retry.`

#### Scenario: Spec not yet started
- **GIVEN** a change with `intake: active`, `spec: pending`
- **WHEN** the user runs `/fab-ff`
- **THEN** the pipeline aborts
- **AND** a message is displayed: `Spec not started. Run /fab-continue to generate the spec first, or use /fab-fff for the full pipeline.`

#### Scenario: Resuming from tasks or later
- **GIVEN** a change with `spec: done`, `tasks: active`, `confidence.score: 4.0`
- **WHEN** the user runs `/fab-ff`
- **THEN** the gate passes
- **AND** the pipeline resumes from tasks through hydrate, skipping spec

### Requirement: `/fab-ff` SHALL NOT frontload questions

Since the spec is already done (or at least active with a confidence score), `/fab-ff` SHALL NOT perform a frontloaded question batch. Planning stages (if tasks are not yet done) proceed directly.

#### Scenario: No frontloaded questions
- **GIVEN** a change with `spec: done`, `tasks: pending`
- **WHEN** the user runs `/fab-ff`
- **THEN** tasks generation begins without a preliminary Q&A round

### Requirement: `/fab-ff` SHALL do minimal auto-clarify

`/fab-ff` SHALL interleave auto-clarify only if tasks generation occurs (tasks not yet `done`). If the pipeline starts at apply or later, no auto-clarify runs. Auto-clarify uses `[AUTO-MODE]` prefix and bails on blocking issues.

#### Scenario: Tasks not yet done
- **GIVEN** `spec: done`, `tasks: pending`
- **WHEN** `/fab-ff` generates tasks
- **THEN** auto-clarify runs after tasks generation
- **AND** bails if blocking issues are found

#### Scenario: Tasks already done
- **GIVEN** `spec: done`, `tasks: done`, `apply: active`
- **WHEN** `/fab-ff` resumes from apply
- **THEN** no auto-clarify is invoked

### Requirement: `/fab-ff` SHALL bail immediately on review failure

Unlike `/fab-fff` which presents interactive rework, `/fab-ff` SHALL stop immediately on review failure with an actionable message and no interactive menu.

#### Scenario: Review failure bail
- **GIVEN** the review step identifies issues
- **WHEN** the review verdict is "fail"
- **THEN** the pipeline stops immediately
- **AND** the message `Review failed. Run /fab-continue for rework options.` is displayed
- **AND** no interactive rework menu is presented

### Requirement: `/fab-ff` driver identification

All `.status.yaml` transitions within `/fab-ff` SHALL pass `fab-ff` as the driver parameter. `log-command` SHALL record `fab-ff`.

#### Scenario: Status tracking
- **GIVEN** `/fab-ff` is executing the pipeline
- **WHEN** a stage transition occurs
- **THEN** `lib/stageman.sh transition` is called with `fab-ff` as driver

## Context File: `_context.md` Updates

### Requirement: Next Steps table SHALL reflect new skill scopes

The Next Steps lookup table in `_context.md` SHALL update the `/fab-ff` and `/fab-fff` entries to reflect the new scope definitions:
- `/fab-ff` bail → `Next: /fab-continue for rework options` (no interactive rework)
- `/fab-fff` bail → contextual based on what stage failed (same as current `/fab-ff` bail behavior)

#### Scenario: Next Steps table accuracy
- **GIVEN** a user reads the Next Steps table in `_context.md`
- **WHEN** looking up the next command after `/fab-ff` or `/fab-fff`
- **THEN** the table reflects the correct behavior: `/fab-fff` with interactive options, `/fab-ff` with bail

### Requirement: Skill-Specific Autonomy Levels table SHALL swap behaviors

The Autonomy Levels table SHALL update to reflect:
- `/fab-ff`: posture is "gated on confidence > 3, no frontloaded questions, minimal auto-clarify, bail on failure"
- `/fab-fff`: posture is "full pipeline, frontloaded questions, interleaved auto-clarify, interactive on failure"

#### Scenario: Autonomy table accuracy
- **GIVEN** a skill reads the Autonomy Levels table
- **WHEN** determining how to behave for `/fab-ff` vs `/fab-fff`
- **THEN** `/fab-ff` shows gated, minimal interaction, bail behavior
- **AND** `/fab-fff` shows no gate, frontloaded questions, interactive failure handling

### Requirement: Confidence Scoring gate reference SHALL point to `/fab-ff`

The Confidence Scoring section in `_context.md` SHALL reference `/fab-ff` (not `/fab-fff`) as the skill that gates on confidence score. The threshold description and `calc-score.sh --check-gate` reference SHALL be updated accordingly.

#### Scenario: Gate reference accuracy
- **GIVEN** a user or skill reads the Confidence Scoring section
- **WHEN** looking for which skill is gated
- **THEN** the section references `/fab-ff` as the gated skill
- **AND** `/fab-fff` is described as ungated

## Memory Files: Cross-Reference Updates

### Requirement: `planning-skills.md` SHALL rewrite `/fab-ff` and `/fab-fff` sections

The `/fab-ff` section SHALL describe the new fast-forward-from-spec behavior with confidence gate, minimal auto-clarify, and bail-on-failure. The `/fab-fff` section SHALL describe the full pipeline with no gate, frontloaded questions, interleaved auto-clarify, and interactive rework menu. The overview paragraph noting `/fab-ff` as documented here SHALL be updated to reflect its new scope. Design decisions SHALL be updated or added to capture the redefinition rationale.

#### Scenario: Planning skills accuracy
- **GIVEN** a user reads the planning-skills.md memory file
- **WHEN** looking up `/fab-ff` or `/fab-fff` behavior
- **THEN** the descriptions match the new skill files exactly

### Requirement: `execution-skills.md` SHALL update pipeline invocation note

The overview paragraph referencing `/fab-ff` and `/fab-fff` pipeline invocation SHALL be updated to reflect the new behavioral differentiation: `/fab-fff` presents interactive rework on review failure; `/fab-ff` bails immediately.

#### Scenario: Execution skills accuracy
- **GIVEN** a user reads the execution-skills.md memory file
- **WHEN** looking at the pipeline invocation note
- **THEN** the note correctly describes `/fab-fff` as interactive on failure and `/fab-ff` as bail-on-failure

### Requirement: `change-lifecycle.md` SHALL update full pipeline path

The "Full pipeline path" description SHALL reference `/fab-fff` as the full pipeline command (no confidence gate) and `/fab-ff` as the fast-forward-from-spec command (gated). The existing reference to `/fab-fff` gating on confidence >= 3.0 SHALL be corrected.

#### Scenario: Change lifecycle accuracy
- **GIVEN** a user reads the change-lifecycle.md memory file
- **WHEN** looking at the full pipeline path
- **THEN** `/fab-fff` is described as the ungated full pipeline command
- **AND** `/fab-ff` is described as the confidence-gated from-spec command

## Deprecated Requirements

### `/fab-fff` Confidence Gate

**Reason**: The confidence gate is moved from `/fab-fff` to `/fab-ff`. `/fab-fff` no longer checks `confidence.score` before proceeding.
**Migration**: `/fab-ff` now performs the same gate check using `calc-score.sh --check-gate`.

### `/fab-ff` Frontloaded Questions

**Reason**: `/fab-ff` no longer scans the intake for all-stage ambiguities since the spec already exists when `/fab-ff` is invoked.
**Migration**: Frontloaded questions move to `/fab-fff`, which now handles the full planning pipeline.

### `/fab-ff` Interactive Rework on Review Failure

**Reason**: `/fab-ff` now bails immediately on review failure instead of presenting an interactive rework menu.
**Migration**: Interactive rework is now in `/fab-fff`. Users can run `/fab-continue` for rework options after `/fab-ff` bails.

### `/fab-fff` Bail on Review Failure

**Reason**: `/fab-fff` now presents interactive rework options instead of bailing immediately.
**Migration**: Bail-on-failure behavior moves to `/fab-ff`.

## Design Decisions

1. **Scope differentiation by prerequisite**: `/fab-fff` starts from intake (full pipeline), `/fab-ff` starts from spec (post-spec pipeline).
   - *Why*: The chicken-and-egg problem — confidence scores only exist after spec generation, so gating a command that generates the spec is paradoxical. Moving the gate to the post-spec command resolves this naturally.
   - *Rejected*: Computing confidence from intake alone — intake assumptions are state transfer, not scored, and would give unreliable scores.

2. **Gate on `/fab-ff`, not `/fab-fff`**: The confidence gate makes practical sense on the shorter pipeline that assumes planning is already done well.
   - *Why*: When spec and score exist, the gate validates that the spec is solid enough for autonomous execution. When starting from intake, there's no score to gate on.
   - *Rejected*: Keeping the gate on `/fab-fff` — perpetuates the chicken-and-egg problem.

3. **Interactive rework on `/fab-fff`, bail on `/fab-ff`**: The longer pipeline (`/fab-fff`) gets interactive rework; the gated pipeline (`/fab-ff`) bails.
   - *Why*: `/fab-fff` runs the full pipeline including planning — if review fails, the user invested more time and should be able to course-correct. `/fab-ff` has a confidence gate that implies high confidence, so failure is unexpected and warrants investigation.
   - *Rejected*: Both bail — loses the course-correction value for long pipelines. Both interactive — makes `/fab-ff` too similar to `/fab-fff`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `/fab-fff` inherits current `/fab-ff`'s interactive rework menu | Confirmed from intake #1 — explicitly decided in conversation | S:95 R:70 A:90 D:95 |
| 2 | Certain | `/fab-ff` inherits current `/fab-fff`'s bail-on-failure behavior | Confirmed from intake #2 — explicitly decided in conversation | S:95 R:70 A:90 D:95 |
| 3 | Certain | `/fab-ff` minimum prerequisite is spec `active` or later | Confirmed from intake #3 — score exists when spec stage is active | S:95 R:80 A:95 D:95 |
| 4 | Certain | Confidence gate threshold stays > 3 (moved to `/fab-ff`) | Confirmed from intake #4 — dynamic per-type thresholds preserved | S:95 R:85 A:90 D:95 |
| 5 | Certain | No script changes needed — `calc-score.sh` reused as-is | Confirmed from intake #5 — gate logic is identical | S:90 R:90 A:95 D:95 |
| 6 | Certain | `/fab-ff` does minimal auto-clarify (only between tasks if tasks not done) | Upgraded from intake #6 Confident — spec is already scored and solid, auto-clarify at tasks level only is the clear intent | S:85 R:80 A:85 D:85 |
| 7 | Certain | Dynamic gate thresholds (bugfix=2.0, feature=3.0, arch=4.0) move to `/fab-ff` | Upgraded from intake #7 Confident — same `calc-score.sh --check-gate` mechanism, just invoked by `/fab-ff` | S:85 R:80 A:90 D:85 |
| 8 | Certain | `/fab-fff` callable from any stage at or after intake | Intake states "Callable from: Any stage at or after intake" — explicit | S:95 R:85 A:90 D:95 |
| 9 | Certain | `/fab-ff` callable from any stage at or after spec | Intake states "Callable from: Any stage at or after spec" — explicit | S:95 R:85 A:90 D:95 |
| 10 | Certain | `/fab-fff` description/frontmatter updates to reflect new purpose | Required for accurate skill triggering — description must match new behavior | S:90 R:90 A:95 D:95 |

10 assumptions (10 certain, 0 confident, 0 tentative, 0 unresolved).
