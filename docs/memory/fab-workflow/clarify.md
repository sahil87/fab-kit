# Clarify Skill

**Domain**: fab-workflow

## Overview

The `/fab-clarify` skill deepens and refines the current stage artifact without advancing to the next stage. It operates in two modes depending on call context: **suggest mode** for interactive user-driven clarification, and **auto mode** for autonomous resolution when called internally by `fab-ff`.

## Requirements

### Dual-Mode Operation

`/fab-clarify` SHALL support two modes, determined by the `[AUTO-MODE]` prefix defined in the Skill Invocation Protocol (`_preamble.md`):

- **Suggest mode**: Activated when the `[AUTO-MODE]` prefix is **absent** (e.g., user invokes `/fab-clarify` directly). Interactive, presents structured questions one at a time with recommendations and options.
- **Auto mode**: Activated when the `[AUTO-MODE]` prefix is **present** (e.g., `/fab-ff` invokes clarify internally between stage generations). Autonomous, resolves gaps without user interaction and returns a machine-readable result.

There SHALL be no `--suggest` or `--auto` flags on the clarify skill.

### Suggest Mode

#### Stage-Scoped Taxonomy Scan

The skill SHALL perform a systematic scan of the current stage's artifacts for gaps, ambiguities, and `[NEEDS CLARIFICATION]` markers. Scan categories vary by stage, using per-artifact taxonomy:

- **Spec** (scans both `intake.md` and `spec.md`):
  - *Intake refinement*: scope boundaries, affected areas, blocking questions, impact completeness, affected memory coverage, Origin section completeness
  - *Spec refinement*: requirement precision, scenario coverage, edge cases, deprecated requirements, cross-references
- **Tasks**: task completeness, granularity, dependency ordering, file path accuracy, parallel markers

The scan also detects:
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

In auto mode, the skill SHALL resolve gaps using available context (config, constitution, memory files, completed artifacts). It classifies each gap as resolvable, blocking, or non-blocking. The scan includes `<!-- assumed: ... -->` markers — those confirmable from context are resolved (marker removed), others are classified as blocking or non-blocking.

#### Grade Reclassification

When a Tentative or Confident assumption is resolved or confirmed during a suggest-mode session, the skill SHALL update the corresponding entry's Grade column in the artifact's `## Assumptions` table to `Certain`. The user's confirmation eliminates ambiguity, making the decision deterministic. This reclassification occurs immediately after each answer, before the next question is presented.

#### Confidence Recomputation

After each suggest-mode session, the skill SHALL recompute the confidence score by re-counting SRAD grades across all artifacts in the change (scanning the `## Assumptions` tables) and applying the formula (see `_preamble.md` Confidence Scoring section). The updated `confidence` block is written to `.status.yaml`. Because reclassified grades (Tentative/Confident → Certain) reduce the penalty count, the score increases after clarification.

#### Machine-Readable Result

Auto mode SHALL return a structured result: `{resolved: N, blocking: N, non_blocking: N}`. If blocking issues exist, descriptions are included: `{..., blocking_issues: ["description"]}`. This result is consumed by `fab-ff` to decide whether to continue or bail.

### Bulk Confirm (Confident Assumptions)

When the confidence score is low primarily due to many Confident (not Tentative/Unresolved) assumptions, suggest mode SHALL offer a bulk confirm flow (Step 2) after the taxonomy scan and tentative resolution (Step 1.5). This displays all Confident assumptions in a numbered list and lets the user confirm, change, or request explanation in a single conversational turn.

#### Detection

Bulk confirm triggers when BOTH conditions are met:
- `confident >= 3` (enough to materially affect the score)
- `confident > tentative + unresolved` (Confident is the dominant drag)

When not triggered, the skill proceeds directly to Step 3 (remaining taxonomy questions).

#### Flow

1. Display all Confident assumptions using original `#` column from the Assumptions table, with Decision and Rationale. Do NOT use `AskUserQuestion` — the list is plain text, and the user's next conversational message is the response.
2. Parse the response: confirm (`✓`/`ok`/`yes`/bare number), change (free text), explain (`?`), range (`{start}-{end}`), or all (`all ✓`). Case-insensitive for keywords.
3. For explanation requests: provide a brief inline explanation, then re-prompt for only the unexplained items (one round max).
4. Update the Assumptions table in place: confirmed/changed items → Certain, Rationale updated (e.g., `Clarified — user confirmed`), S dimension → 95 (R, A, D unchanged). Unmentioned items stay Confident.
5. Append to Clarifications audit trail as `### Session {date} (bulk confirm)`.

After bulk confirm completes, proceed to Step 3 (remaining taxonomy questions from Step 1.5's queue).

#### Auto Mode Exclusion

Bulk confirm is Suggest Mode only. Auto Mode skips it — there is no user to confirm with.

### Non-Advancing Property

The clarify skill SHALL never advance the stage in `.status.yaml`. It only updates the `last_updated` timestamp. The user explicitly advances via `/fab-continue`.

### Stage Guard

The skill SHALL only operate on planning stages (`intake`, `spec`, `tasks`). At the `intake` stage, the taxonomy scan covers intake artifact refinement (scope boundaries, affected areas, blocking questions, impact, memory coverage). If the stage is `apply`, `review`, or `hydrate`, the skill aborts with a suggestion to use `/fab-continue` instead.

## Design Decisions

### Mode Selection by `[AUTO-MODE]` Prefix
**Decision**: Mode is determined by the `[AUTO-MODE]` prefix defined in the Skill Invocation Protocol (`_preamble.md`). Prefix present = auto mode; absent = suggest mode. No flags.
**Why**: Makes the contract explicit and testable rather than relying on implicit call-context interpretation. Avoids a confusing `--suggest`/`--auto` flag pair with no clear use case for user-invoked auto mode.
**Rejected**: Flag-based mode selection — adds complexity, no user scenario requires it. Implicit call-context detection — unreliable, not testable.
*Updated by*: 260210-nan4-define-auto-mode-signaling

### Max 5 Questions Per Invocation
**Decision**: Cap suggest mode at 5 questions per invocation. Re-run for more.
**Why**: Beyond 5 questions, diminishing returns and user fatigue. The skill is idempotent — running it again is free and reprioritizes.
**Rejected**: Unlimited questions — leads to marathon sessions. Fixed question count regardless of gaps — too rigid.

### Incremental Updates (Not Batched)
**Decision**: Update the artifact after each answer, not at the end of the session.
**Why**: If the user terminates early or the session is interrupted, all answered questions are already reflected in the artifact. No work is lost.
**Rejected**: Batch updates at session end — risks losing all clarifications on interruption.

### Grade Reclassification in Assumptions Table
**Decision**: When `/fab-clarify` resolves a Tentative or Confident assumption, the grade is reclassified to Certain in-place in the artifact's `## Assumptions` table. The confidence recount then reads the updated table, producing a higher score.
**Why**: Keeps the source of truth co-located with the artifact. The recount reads the Assumptions table directly, so in-place updates make the recount naturally correct. This ensures scores increase after clarification.
**Rejected**: Separate resolution tracking file — adds complexity, risks drift between the table and the tracker. Removing entries instead of reclassifying — loses the decision record.
*Introduced by*: 260212-29xv-scoring-formula

### Bulk Confirm over AskUserQuestion
**Decision**: The bulk confirm flow uses plain text display + conversational message parsing instead of per-item `AskUserQuestion` tool calls.
**Why**: The motivating session proved conversational bulk response is ~10x faster. `AskUserQuestion` forces per-item round-trips that defeat the purpose of bulk confirmation. `multiSelect: true` caps at 4 options per question and still requires structured tool-call interaction.
**Rejected**: Per-item `AskUserQuestion` — too slow. Multi-select `AskUserQuestion` — capped at 4 options.
*Introduced by*: 260302-c7is-fab-clarify-bulk-confirm

### Audit Trail in Artifact (Not Separate File)
**Decision**: Append clarification history directly to the artifact under a `## Clarifications` section.
**Why**: Keeps the audit trail with the artifact it describes. No separate files to track. Sessions accumulate naturally.
**Rejected**: Separate `clarifications.md` file — adds file management overhead, loses co-location benefit.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260416-hyl6-clarify-tentative-first | 2026-04-16 | Reordered suggest mode flow — taxonomy scan (Step 1.5) now runs before bulk confirm (Step 2), so tentative assumptions are addressed before confident ones |
| 260302-c7is-fab-clarify-bulk-confirm | 2026-03-02 | Added bulk confirm mode (Step 1.5) to suggest mode — detects when Confident assumptions dominate the confidence drag (`confident >= 3` AND `confident > tentative + unresolved`), displays numbered list for conversational bulk response, supports confirm/change/explain/range/all formats, one re-prompt round for explanations. Added to `_preamble.md` Confidence Scoring section. Auto Mode excluded. |
| 260221-5tj7-rename-context-to-preamble | 2026-02-21 | Renamed shared skill preamble from `_context.md` to `_preamble.md`. Updated all references in dual-mode operation and design decisions sections. |
| 260216-7ltw-DEV-1038-standardize-state-keyed-suggestions | 2026-02-16 | Extended stage guard to include `intake` as valid planning stage. Added intake taxonomy scan categories (scope boundaries, affected areas, blocking questions, impact, memory coverage). All `Next:` lines now derived from canonical state table in `_preamble.md`. |
| 260215-v4n7-DEV-1025-rename-brief-to-intake | 2026-02-15 | Renamed `brief` stage/artifact to `intake` throughout — stage identifiers, artifact filenames, YAML keys, prose references |
| 260212-29xv-scoring-formula | 2026-02-12 | Added grade reclassification: resolved Tentative/Confident assumptions become Certain in Assumptions table before confidence recomputation, so scores increase after clarification |
| 260212-v5p2-simplify-stages-entry-paths | 2026-02-12 | Added intake refinement capability at spec stage with per-artifact taxonomy, removed intake from valid stages |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Updated stage names to intake/spec/tasks in guard logic and output examples |
| 260210-nan4-define-auto-mode-signaling | 2026-02-10 | Updated dual-mode operation to use explicit `[AUTO-MODE]` prefix protocol; updated "Mode Selection" design decision |
| 260208-k3m7-add-fab-fff | 2026-02-08 | Added confidence recomputation after suggest-mode sessions, removed `<!-- auto-guess -->` scanning references |
| 260207-09sj-autonomy-framework | 2026-02-08 | Added `<!-- assumed: ... -->` marker scanning to both suggest and auto modes; assumed markers framed as recommendations with alternatives |
| 260207-m3qf-clarify-dual-modes | 2026-02-07 | Initial doc — dual-mode clarify skill (suggest + auto), taxonomy scan, structured questions, coverage reports, audit trail |
