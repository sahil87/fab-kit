# Spec: fab-clarify Dual Modes + fab-ff Clarify Checkpoints

**Change**: 260207-m3qf-clarify-dual-modes
**Created**: 2026-02-07
**Affected docs**: `fab/docs/fab-workflow/clarify.md` (new), `fab/docs/fab-workflow/index.md` (modified)

## fab-clarify: Suggest Mode (User-Interactive)

### Requirement: Suggest Mode as Default for User Invocation

When a user invokes `/fab:clarify` directly, the skill SHALL operate in **suggest mode**. Suggest mode presents structured questions one at a time, with recommendations and options, allowing the user to drive ambiguity resolution interactively.

#### Scenario: User invokes /fab:clarify on a spec with NEEDS CLARIFICATION markers

- **GIVEN** the current stage is `specs` and `spec.md` contains `[NEEDS CLARIFICATION]` markers
- **WHEN** the user runs `/fab:clarify`
- **THEN** the skill performs a stage-scoped taxonomy scan and presents the highest-priority question with a recommendation, reasoning, and options table
- **AND** only one question is presented at a time (future queued questions are not revealed)

#### Scenario: User invokes /fab:clarify on a clean artifact

- **GIVEN** the current stage is `specs` and `spec.md` contains no explicit markers or detectable gaps
- **WHEN** the user runs `/fab:clarify`
- **THEN** the skill performs the taxonomy scan anyway and either surfaces implicit gaps or reports "No gaps found — artifact looks solid."

### Requirement: Stage-Scoped Taxonomy Scan

The suggest mode SHALL perform a taxonomy scan scoped to the current stage. The scan categories MUST vary by stage — there is no fixed universal category list. The scan identifies gaps, ambiguities, and `[NEEDS CLARIFICATION]` markers across all relevant categories for the current stage.

#### Scenario: Taxonomy scan on proposal stage

- **GIVEN** the current stage is `proposal`
- **WHEN** the taxonomy scan runs
- **THEN** categories include (but are not limited to): scope boundaries, affected areas, blocking questions, impact completeness, affected docs coverage

#### Scenario: Taxonomy scan on specs stage

- **GIVEN** the current stage is `specs`
- **WHEN** the taxonomy scan runs
- **THEN** categories include (but are not limited to): requirement precision, scenario coverage, edge cases, deprecated requirements, cross-references to centralized docs

#### Scenario: Taxonomy scan on plan stage

- **GIVEN** the current stage is `plan`
- **WHEN** the taxonomy scan runs
- **THEN** categories include (but are not limited to): assumption verification, research completeness, decision rationale, risk identification, file change coverage

#### Scenario: Taxonomy scan on tasks stage

- **GIVEN** the current stage is `tasks`
- **WHEN** the taxonomy scan runs
- **THEN** categories include (but are not limited to): task completeness, granularity, dependency ordering, file path accuracy, parallel markers

### Requirement: Structured Question Format

Each question presented in suggest mode SHALL include:
1. A recommendation with reasoning
2. An options table (for multiple-choice) or a suggested answer (for short-answer)

The user MAY accept the recommendation with "yes" / "recommended", pick a specific option, or provide their own answer.

#### Scenario: Multiple-choice question

- **GIVEN** the taxonomy scan identifies an ambiguity with discrete resolution options
- **WHEN** the question is presented
- **THEN** it includes a recommendation line with reasoning, plus an options table with at least 2 options
- **AND** the user can respond with the option number, "yes"/"recommended" to accept the recommendation, or free-text to provide a custom answer

#### Scenario: Short-answer question

- **GIVEN** the taxonomy scan identifies an ambiguity requiring free-form input
- **WHEN** the question is presented
- **THEN** it includes a suggested answer with reasoning
- **AND** the user can respond with "yes"/"recommended" to accept, or provide their own answer

### Requirement: Max 5 Questions Per Invocation

A single `/fab:clarify` invocation in suggest mode SHALL present at most 5 questions. If more gaps remain after 5 questions, the coverage summary SHALL indicate outstanding items, and the user can run `/fab:clarify` again (the taxonomy scan reprioritizes on each invocation).

#### Scenario: More than 5 gaps detected

- **GIVEN** the taxonomy scan identifies 8 gaps
- **WHEN** the skill runs in suggest mode
- **THEN** only the 5 highest-priority questions are presented (one at a time)
- **AND** the coverage summary reports 5 resolved, 3 outstanding
- **AND** re-running `/fab:clarify` addresses the remaining gaps

#### Scenario: Fewer than 5 gaps detected

- **GIVEN** the taxonomy scan identifies 2 gaps
- **WHEN** the skill runs in suggest mode
- **THEN** both questions are presented (one at a time)
- **AND** the coverage summary reports 2 resolved, 0 outstanding

### Requirement: Incremental Artifact Updates

After each user answer in suggest mode, the skill SHALL immediately update the artifact in place before presenting the next question. This ensures the artifact reflects all resolutions even if the user terminates early.

#### Scenario: User answers a question

- **GIVEN** the skill has presented question 2 of 4
- **WHEN** the user provides an answer
- **THEN** the artifact is updated in place to reflect the resolution
- **AND** question 3 is presented

#### Scenario: User terminates early

- **GIVEN** the skill has presented question 2 of 4 and the user has answered questions 1 and 2
- **WHEN** the user says "done", "good", or "no more"
- **THEN** the skill stops presenting questions
- **AND** the artifact retains the updates from questions 1 and 2
- **AND** a coverage summary is displayed

### Requirement: Early Termination

The user MAY terminate a suggest-mode session early by responding with "done", "good", or "no more" (case-insensitive). The skill SHALL stop presenting further questions and proceed to the coverage summary.

#### Scenario: User terminates with "done"

- **GIVEN** the skill has presented question 1 of 5
- **WHEN** the user responds with "done"
- **THEN** no further questions are presented
- **AND** a coverage summary is displayed showing resolved, clear, deferred, and outstanding counts

### Requirement: Clarifications Audit Trail

Each suggest-mode session SHALL append an audit trail to the artifact under `## Clarifications > ### Session {YYYY-MM-DD}` with `Q:` / `A:` bullets for each resolved question.

#### Scenario: Audit trail after a complete session

- **GIVEN** the user answers 3 questions in a suggest-mode session
- **WHEN** the session completes
- **THEN** the artifact contains a `## Clarifications` section with a `### Session {date}` subsection
- **AND** the subsection contains 3 `Q:` / `A:` bullet pairs

#### Scenario: Multiple sessions accumulate

- **GIVEN** the artifact already has a `## Clarifications` section with a previous session
- **WHEN** a new suggest-mode session completes
- **THEN** a new `### Session {date}` subsection is appended (not replacing the previous one)

### Requirement: Coverage Summary

At the end of each suggest-mode session, the skill SHALL display a coverage summary table with four categories:

| Category | Meaning |
|----------|---------|
| Resolved | Gaps addressed in this session |
| Clear | Areas scanned with no gaps found |
| Deferred | Gaps the user chose not to address (early termination) |
| Outstanding | Gaps beyond the 5-question cap, awaiting next invocation |

#### Scenario: Coverage summary after full session

- **GIVEN** the taxonomy scan found 4 gaps and the user resolved all 4
- **WHEN** the coverage summary is displayed
- **THEN** it shows: Resolved: 4, Clear: {N categories with no gaps}, Deferred: 0, Outstanding: 0

## fab-clarify: Auto Mode (Internal)

### Requirement: Auto Mode Activation

When `/fab:clarify` is called internally by `fab-ff`, it SHALL operate in **auto mode**. Auto mode preserves the current autonomous behavior: the agent resolves gaps itself using context (config, constitution, centralized docs) and only escalates truly blocking issues.

#### Scenario: fab-ff calls clarify internally

- **GIVEN** fab-ff is generating artifacts and calls clarify internally between stages
- **WHEN** the clarify skill receives the internal call
- **THEN** it operates in auto mode — no user interaction, agent resolves gaps autonomously

### Requirement: Mode Selection by Call Context

Mode selection SHALL be determined by call context, not flags. `/fab:clarify` invoked by the user = suggest mode. `/fab:clarify` called internally by `fab-ff` = auto mode. There SHALL be no `--suggest` or `--auto` flags on the clarify skill.

#### Scenario: User invocation triggers suggest mode

- **GIVEN** the user types `/fab:clarify` in the CLI
- **WHEN** the skill executes
- **THEN** it operates in suggest mode

#### Scenario: Internal fab-ff call triggers auto mode

- **GIVEN** fab-ff's pipeline calls clarify between stage generations
- **WHEN** the skill executes
- **THEN** it operates in auto mode

### Requirement: Machine-Readable Auto Mode Result

Auto mode SHALL return a machine-readable result containing counts of: resolved issues, blocking issues (that the agent could not resolve autonomously), and non-blocking issues (minor gaps left as-is). This result is consumed by fab-ff to decide whether to continue or bail.

#### Scenario: Auto mode resolves all issues

- **GIVEN** the spec has 3 `[NEEDS CLARIFICATION]` markers
- **WHEN** auto-clarify runs
- **THEN** it returns `{resolved: 3, blocking: 0, non_blocking: 0}`
- **AND** all markers are resolved in the artifact

#### Scenario: Auto mode finds blocking issues

- **GIVEN** the spec has an ambiguity that cannot be resolved from available context
- **WHEN** auto-clarify runs
- **THEN** it returns `{resolved: 0, blocking: 1, non_blocking: 0}`
- **AND** the blocking issue is described in the result

## fab-ff: Default Mode (Clarify Checkpoints)

### Requirement: Interleaved Auto-Clarify

The default `/fab:ff` mode SHALL interleave auto-clarify between stage generations. The pipeline becomes: `spec → auto-clarify → plan → auto-clarify → tasks → auto-clarify`.

#### Scenario: Clean fast-forward with no issues

- **GIVEN** a proposal with no ambiguities
- **WHEN** the user runs `/fab:ff`
- **THEN** the pipeline runs: generate spec → auto-clarify (0 issues) → plan decision → auto-clarify (0 issues) → generate tasks → auto-clarify (0 issues)
- **AND** all artifacts are generated without interruption

#### Scenario: Auto-clarify resolves issues mid-pipeline

- **GIVEN** a proposal that produces a spec with 2 resolvable ambiguities
- **WHEN** the user runs `/fab:ff`
- **THEN** after generating spec, auto-clarify resolves the 2 ambiguities in the spec
- **AND** the pipeline continues to plan/tasks using the clarified spec

### Requirement: Bail on Blocking Issues

If auto-clarify returns blocking issues (count > 0) during the default `/fab:ff` pipeline, the skill SHALL stop generation, report the blocking issues, and suggest the user run `/fab:clarify` to resolve them interactively, then `/fab:ff` to resume.

#### Scenario: Blocking issue stops fast-forward

- **GIVEN** auto-clarify after spec generation returns `{blocking: 1}`
- **WHEN** the blocking result is received
- **THEN** fab-ff stops the pipeline
- **AND** reports the blocking issue to the user
- **AND** outputs: "Run `/fab:clarify` to resolve these, then `/fab:ff` to resume."
- **AND** `.status.yaml` reflects: specs `done`, plan/tasks still `pending`

### Requirement: Resumability After Bail

When `/fab:ff` is re-invoked after a bail, it SHALL resume from the current position — skipping stages already marked `done` and generating only the remaining artifacts.

#### Scenario: Resume after blocking bail

- **GIVEN** fab-ff previously bailed after specs (specs: done, plan: pending, tasks: pending)
- **AND** the user ran `/fab:clarify` to resolve the blocking issues
- **WHEN** the user runs `/fab:ff` again
- **THEN** fab-ff skips spec generation (already done)
- **AND** continues from the plan decision onward

## fab-ff: Full-Auto Mode (`--auto`)

### Requirement: Full-Auto Flag

`/fab:ff --auto` SHALL run the same interleaved pipeline as default mode but never stop for blocking issues. Instead, it makes best-guess decisions on blockers, marks them with `<!-- auto-guess: {description} -->` HTML comment markers in the artifact, and warns the user in output.

#### Scenario: Full-auto with blocking issue

- **GIVEN** auto-clarify after spec generation finds a blocking ambiguity
- **WHEN** running `/fab:ff --auto`
- **THEN** the agent makes a best-guess resolution
- **AND** the guess is marked in the artifact with `<!-- auto-guess: {description} -->`
- **AND** the pipeline continues without stopping
- **AND** the output includes a warning listing all auto-guesses made

#### Scenario: Full-auto with no issues

- **GIVEN** a clean proposal with no ambiguities
- **WHEN** running `/fab:ff --auto`
- **THEN** the pipeline runs identically to default `/fab:ff` (no auto-guess markers needed)
- **AND** output confirms no guesses were necessary

### Requirement: Auto-Guess Visibility

All `<!-- auto-guess: {description} -->` markers SHALL be:
1. Visible in the raw markdown of the artifact
2. Detectable by `/fab:review` during validation (review SHOULD flag them as warnings)
3. Resolvable by subsequent `/fab:clarify` (suggest mode) which can find and interactively resolve them

#### Scenario: Review detects auto-guess markers

- **GIVEN** a spec with `<!-- auto-guess: assumed OAuth2 for auth provider -->` markers
- **WHEN** `/fab:review` runs
- **THEN** the review flags each auto-guess marker as a warning
- **AND** the review output lists them for the user's attention

#### Scenario: Clarify resolves auto-guess markers

- **GIVEN** a spec with `<!-- auto-guess: assumed OAuth2 for auth provider -->` markers
- **WHEN** the user runs `/fab:clarify`
- **THEN** the taxonomy scan includes auto-guess markers as gaps to resolve
- **AND** the user can confirm or override each guess interactively

## Context and Next Steps Updates

### Requirement: _context.md Next Steps Update

The `_context.md` Next Steps table SHALL be updated to include the `/fab:ff --auto` variant. After `/fab:ff --auto` completes, the next step is `Next: /fab:apply` (same as default `/fab:ff`).

#### Scenario: Next Steps table includes --auto variant

- **GIVEN** the `_context.md` file
- **WHEN** a developer reads the Next Steps table
- **THEN** there is an entry for `/fab:ff --auto` with the same next step as `/fab:ff`

### Requirement: No Changes to Other Skills

This change SHALL NOT modify the behavior of `/fab:continue`, `/fab:new`, `/fab:apply`, `/fab:review`, `/fab:archive`, templates, or the `.status.yaml` schema. The only skill files modified are `fab-clarify.md`, `fab-ff.md`, and `_context.md`.

#### Scenario: /fab:continue behavior unchanged

- **GIVEN** a user runs `/fab:continue` between stages
- **WHEN** the stage advances
- **THEN** no auto-clarify is interleaved (clarify is manual in the continue path)

## Deprecated Requirements

(none — this change extends existing skills without removing any current behavior)
