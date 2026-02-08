# Clarify Skill

**Domain**: fab-workflow

## Overview

The `/fab-clarify` skill deepens and refines the current stage artifact without advancing to the next stage. It operates in two modes depending on call context: **suggest mode** for interactive user-driven clarification, and **auto mode** for autonomous resolution when called internally by `fab-ff`.

## Requirements

### Dual-Mode Operation

`/fab-clarify` SHALL support two modes, determined by call context (not flags):

- **Suggest mode**: Activated when the user invokes `/fab-clarify` directly. Interactive, presents structured questions one at a time with recommendations and options.
- **Auto mode**: Activated when `fab-ff` calls clarify internally between stage generations. Autonomous, resolves gaps without user interaction and returns a machine-readable result.

There SHALL be no `--suggest` or `--auto` flags on the clarify skill.

### Suggest Mode

#### Stage-Scoped Taxonomy Scan

The skill SHALL perform a systematic scan of the current artifact for gaps, ambiguities, and `[NEEDS CLARIFICATION]` markers. Scan categories vary by stage:

- **Proposal**: scope boundaries, affected areas, blocking questions, impact completeness, affected docs coverage
- **Specs**: requirement precision, scenario coverage, edge cases, deprecated requirements, cross-references
- **Plan**: assumption verification, research completeness, decision rationale, risk identification, file change coverage
- **Tasks**: task completeness, granularity, dependency ordering, file path accuracy, parallel markers

The scan also detects:
- `<!-- auto-guess: ... -->` markers left by `/fab-ff --auto` — Unresolved decisions to resolve interactively
- `<!-- assumed: ... -->` markers left by any planning skill — Tentative assumptions to confirm or override

When presenting questions from `<!-- assumed: ... -->` markers, the current assumption is framed as the recommended option with alternatives offered.

#### Structured Question Format

Each question SHALL include either:
- A **recommendation with options table** (for multiple-choice questions with discrete resolution options)
- A **suggested answer with reasoning** (for short-answer questions requiring free-form input)

The user MAY accept the recommendation ("yes"/"recommended"), pick a numbered option, or provide a custom answer.

#### One Question at a Time

Questions SHALL be presented one at a time. Future queued questions are not revealed until the current one is answered.

#### Max 5 Questions Per Invocation

A single invocation SHALL present at most 5 questions. If more gaps remain, the coverage summary indicates outstanding items. Re-running `/fab-clarify` addresses remaining gaps (the taxonomy scan reprioritizes on each invocation).

#### Incremental Artifact Updates

After each user answer, the skill SHALL immediately update the artifact in place before presenting the next question. This ensures the artifact reflects all resolutions even if the user terminates early.

#### Early Termination

The user MAY terminate early by responding with "done", "good", or "no more" (case-insensitive). The skill stops presenting questions and proceeds to the coverage summary.

#### Clarifications Audit Trail

Each suggest-mode session SHALL append an audit trail to the artifact under `## Clarifications > ### Session {YYYY-MM-DD}` with `Q:` / `A:` entries for each resolved question. Multiple sessions accumulate — new sessions are appended, never replacing previous ones.

#### Coverage Summary

At the end of each session, the skill SHALL display a coverage summary with four categories: Resolved (gaps addressed this session), Clear (categories with no gaps), Deferred (gaps skipped via early termination), Outstanding (gaps beyond the 5-question cap).

### Auto Mode

#### Autonomous Resolution

In auto mode, the skill SHALL resolve gaps using available context (config, constitution, centralized docs, completed artifacts). It classifies each gap as resolvable, blocking, or non-blocking. The scan includes `<!-- assumed: ... -->` markers — those confirmable from context are resolved (marker removed), others are classified as blocking or non-blocking.

#### Machine-Readable Result

Auto mode SHALL return a structured result: `{resolved: N, blocking: N, non_blocking: N}`. If blocking issues exist, descriptions are included: `{..., blocking_issues: ["description"]}`. This result is consumed by `fab-ff` to decide whether to continue or bail.

### Non-Advancing Property

The clarify skill SHALL never advance the stage in `.status.yaml`. It only updates the `last_updated` timestamp. The user explicitly advances via `/fab-continue`.

### Stage Guard

The skill SHALL only operate on planning stages (`proposal`, `specs`, `plan`, `tasks`). If the stage is `apply`, `review`, or `archive`, the skill aborts with a suggestion to use `/fab-review` instead.

## Design Decisions

### Mode Selection by Call Context
**Decision**: Mode is determined by how the skill is invoked (user = suggest, fab-ff internal = auto), not by flags.
**Why**: Avoids a confusing `--suggest`/`--auto` flag pair with no clear use case for user-invoked auto mode. The call context naturally maps to the right behavior.
**Rejected**: Flag-based mode selection — adds complexity, no user scenario requires it.

### Max 5 Questions Per Invocation
**Decision**: Cap suggest mode at 5 questions per invocation. Re-run for more.
**Why**: Beyond 5 questions, diminishing returns and user fatigue. The skill is idempotent — running it again is free and reprioritizes.
**Rejected**: Unlimited questions — leads to marathon sessions. Fixed question count regardless of gaps — too rigid.

### Incremental Updates (Not Batched)
**Decision**: Update the artifact after each answer, not at the end of the session.
**Why**: If the user terminates early or the session is interrupted, all answered questions are already reflected in the artifact. No work is lost.
**Rejected**: Batch updates at session end — risks losing all clarifications on interruption.

### Audit Trail in Artifact (Not Separate File)
**Decision**: Append clarification history directly to the artifact under a `## Clarifications` section.
**Why**: Keeps the audit trail with the artifact it describes. No separate files to track. Sessions accumulate naturally.
**Rejected**: Separate `clarifications.md` file — adds file management overhead, loses co-location benefit.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260207-09sj-autonomy-framework | 2026-02-08 | Added `<!-- assumed: ... -->` marker scanning to both suggest and auto modes; assumed markers framed as recommendations with alternatives |
| 260207-m3qf-clarify-dual-modes | 2026-02-07 | Initial doc — dual-mode clarify skill (suggest + auto), taxonomy scan, structured questions, coverage reports, audit trail |
