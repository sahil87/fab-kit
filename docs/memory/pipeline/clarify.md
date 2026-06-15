---
type: memory
description: "`/fab-clarify` skill — intake-only (j6cs), dual modes (suggest/auto), intake taxonomy scan, structured questions, coverage reports, audit trail, grade reclassification, always-recompute intake score; hosts the [AUTO-MODE] Skill Invocation Protocol and is sole bulk-confirm authority (zc9m); bulk confirm evaluated before the zero-gaps exit, confirmations graded by recomputed composite (no fiat Certain), unified audit-trail placement rules (c5tr)"
---
# Clarify Skill

**Domain**: pipeline

## Overview

The `/fab-clarify` skill deepens and refines the **intake** artifact (`intake.md`) without advancing. As of j6cs it is **intake-only**: clarification is the intake-time, human-facing activity where the developer's decisions and disambiguation happen, gated by the single intake confidence gate. There is no post-intake clarify — inside apply, the agent resolves ambiguity inline as graded SRAD assumptions in `plan.md`'s `## Assumptions`, not via this skill. It operates in two modes depending on call context: **suggest mode** for interactive user-driven clarification, and **auto mode** for autonomous resolution (machine-readable result; no orchestrator currently invokes it — the former `/fab-ff`/`/fab-fff` auto-clarify steps were removed in j6cs).

## Requirements

### Dual-Mode Operation

`/fab-clarify` SHALL support two modes, determined by the `[AUTO-MODE]` prefix defined in the Skill Invocation Protocol — since 260611-zc9m the protocol (prefix placement, detection, transitivity) is defined in `fab-clarify.md` § Skill Invocation Protocol itself, its sole referencer; `_preamble.md` carries only a 2-line pointer, and the preamble's live § Subagent Dispatch references the new location instead of restating it:

- **Suggest mode**: Activated when the `[AUTO-MODE]` prefix is **absent** (e.g., user invokes `/fab-clarify` directly). Interactive, presents structured questions one at a time with recommendations and options.
- **Auto mode**: Activated when the `[AUTO-MODE]` prefix is **present**. Autonomous, resolves gaps without user interaction and returns a machine-readable result. Retained for future use — no orchestrator currently invokes it (j6cs removed the `/fab-ff`/`/fab-fff` auto-clarify steps).

There SHALL be no `--suggest` or `--auto` flags on the clarify skill.

### Suggest Mode

#### Stage-Scoped Taxonomy Scan

The skill SHALL perform a systematic scan of `intake.md` for gaps, ambiguities, and `[NEEDS CLARIFICATION]` markers. There is a single taxonomy (intake-only as of j6cs):

- **Intake**: scope boundaries, affected areas, blocking questions, impact completeness, affected memory coverage, Origin section completeness

A passed `spec`, `plan`, or `tasks` argument is treated as a change name (those targets no longer exist — `spec` and `plan` were removed in j6cs; `tasks` in qszh). At apply or later, `/fab-clarify` STOPs with a pointer to `/fab-continue` for rework or editing `plan.md`'s `## Requirements` directly.

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

Each suggest-mode session SHALL append an audit trail to the artifact under `## Clarifications > ### Session {YYYY-MM-DD}` with `Q:` / `A:` entries for each resolved question. Multiple sessions accumulate — new sessions are appended, never replacing previous ones. Both audit-trail writers (Step 2 bulk confirm, Step 5 Q&A) state **identical placement/append rules** (c5tr): append to the existing `## Clarifications` section if present; create it immediately before `## Assumptions` if not; skip when the session resolved nothing.

#### Coverage Summary

At the end of each session, the skill SHALL display a coverage summary with four categories: Resolved (gaps addressed this session), Clear (categories with no gaps), Deferred (gaps skipped via early termination), Outstanding (gaps beyond the 5-question cap).

### Auto Mode

#### Autonomous Resolution

In auto mode, the skill SHALL resolve gaps in `intake.md` using available context (config, constitution, memory files). It classifies each gap as resolvable, blocking, or non-blocking. The scan includes `<!-- assumed: ... -->` markers — those confirmable from context are resolved (marker removed), others are classified as blocking or non-blocking. As of j6cs no orchestrator invokes auto mode; it is retained for future use and operates on `intake.md` only.

#### Grade Reclassification

When an assumption is resolved through the Step 3–4 Q&A path (a structured question answered directly by the user), the skill SHALL update the corresponding entry's Grade column in the artifact's `## Assumptions` table to `Certain` — answering the question eliminates the ambiguity entirely. This reclassification occurs immediately after each answer, before the next question is presented. **Bulk-confirmed rows are NOT relabeled by fiat** (c5tr): the bulk-confirm Artifact Update sets S → 95 (R, A, D unchanged), recomputes the composite per `_srad.md` § SRAD Scoring, and assigns the grade by its half-open thresholds — a confirmed row whose recomputed composite stays below the Certain band remains Confident, with the confirmation recorded in Rationale. The skill restates no weights or thresholds — the formula is referenced, not inlined (inlining it was exactly the drift class the c5tr batch fixed).

#### Confidence Recomputation

After each suggest-mode session, the skill SHALL recompute the confidence score by re-running `fab score --stage intake <change>` (the recompute step was **inverted in j6cs**: instead of skipping at intake, it now always runs at intake — `intake.md`'s `## Assumptions` table is the sole scoring source). The updated `confidence` block is written to `.status.yaml`. Under the Resolution-Average formula (tf5q) the score is the mean of the per-row S:R:A:D composites rescaled onto 0–5; re-grading resolved assumptions raises their per-row composites (a resolved row's dimensions — typically S — go up), which raises the mean and thus the score. (There is no penalty count to reduce — the prior grade-count penalty ledger was replaced.) Auto mode also recomputes the intake score.

#### Machine-Readable Result

Auto mode SHALL return a structured result: `{resolved: N, blocking: N, non_blocking: N}`. If blocking issues exist, descriptions are included: `{..., blocking_issues: ["description"]}`. (Historically consumed by `/fab-ff`/`/fab-fff` to decide whether to continue or bail; those auto-clarify steps were removed in j6cs, so no current consumer reads this result.)

### Bulk Confirm (Confident Assumptions)

When the confidence score is low primarily due to many Confident (not Tentative/Unresolved) assumptions, suggest mode SHALL offer a bulk confirm flow (Step 2) after the taxonomy scan and tentative resolution (Step 1.5). This displays all Confident assumptions in a numbered list and lets the user confirm, change, or request explanation in a single conversational turn.

**Bulk confirm is evaluated before the zero-gaps exit** (c5tr): Step 1.5 builds the prioritized question queue **without stopping** — the "No gaps found — artifact looks solid." early exit moved into Step 2's not-triggered branch, emitted only when the bulk-confirm trigger did not fire AND the Step 1.5 queue is empty. This makes bulk confirm reachable in its primary scenario — a marker-free, Confident-only intake sitting below the 3.0 gate has zero gaps, so the old Step 1.5 exit dead-ended it at "artifact looks solid" with no path to raise the score.

`fab-clarify.md` (Step 2, Suggest Mode) is the **sole authority** for the bulk-confirm trigger and semantics (260611-zc9m): the former `_preamble.md` § Bulk Confirm subsection — which duplicated the trigger condition, the S → 95 upgrade semantics, and the internal step numbering verbatim — was cut to a one-sentence pointer.

#### Detection

Bulk confirm triggers when BOTH conditions are met:
- `confident >= 3` (enough to materially affect the score)
- `confident > tentative + unresolved` (Confident is the dominant drag)

When not triggered: if Step 1.5's question queue is also empty (zero gaps), the skill outputs "No gaps found — artifact looks solid." with the `Next:` line and stops — the zero-gaps exit lives here, in Step 2's not-triggered branch (c5tr), not in Step 1.5. Otherwise the skill proceeds directly to Step 3 (remaining taxonomy questions).

#### Flow

1. Display all Confident assumptions using original `#` column from the Assumptions table, with Decision and Rationale. Do NOT use `AskUserQuestion` — the list is plain text, and the user's next conversational message is the response.
2. Parse the response: confirm (`✓`/`ok`/`yes`/bare number), change (free text), explain (`?`), range (`{start}-{end}`), or all (`all ✓`). Case-insensitive for keywords.
3. For explanation requests: provide a brief inline explanation, then re-prompt for only the unexplained items (one round max).
4. Update the Assumptions table in place: confirmed/changed items set S → 95 (R, A, D unchanged), Rationale updated (e.g., `Clarified — user confirmed`), and the **grade is assigned by the recomputed composite, not by fiat** (c5tr) — recompute per `_srad.md` § SRAD Scoring and map via its half-open thresholds; a row whose recomputed composite stays < 85 remains Confident. Unmentioned items stay Confident.
5. Append to Clarifications audit trail as `### Session {date} (bulk confirm)` — same placement/append rules as Step 5's Q&A trail (c5tr): append to an existing `## Clarifications` section; create it immediately before `## Assumptions` if absent; skip when 0 items were resolved.

After bulk confirm completes, proceed to Step 3 (remaining taxonomy questions from Step 1.5's queue).

#### Auto Mode Exclusion

Bulk confirm is Suggest Mode only. Auto Mode skips it — there is no user to confirm with.

### Non-Advancing Property

The clarify skill SHALL never advance the stage in `.status.yaml`. It only updates the `last_updated` timestamp. The user explicitly advances via `/fab-continue`.

### Stage Guard

`/fab-clarify` operates only at the **intake** stage:

- **`intake`** — operates on the intake stage (`progress.intake` in `{active, ready, done}`), scanning `intake.md`.

The pre-flight stage guard MUST allow only `intake`. At any post-intake stage (`apply`, `review`, `hydrate`, `ship`, `review-pr`), `/fab-clarify` does not apply and STOPs with: "Clarification is intake-only. At apply or later, run /fab-continue for rework, or edit plan.md `## Requirements` directly. To re-clarify the intake, reset with /fab-continue intake first." If `intake.md` is missing entirely, it STOPs with "No intake.md found. Run /fab-new to create the intake first." The former `spec` and `plan` targets were removed in j6cs (and `tasks` in qszh); any such positional argument is treated as a change name.

## Design Decisions

### Clarify is Intake-Only (j6cs)
**Decision**: `/fab-clarify` accepts only the `intake` target. The former `spec` and `plan` targets were removed when the spec stage was merged into apply. The recompute-confidence step was inverted: instead of skipping at intake, it now always runs `fab score --stage intake <change>`.
**Why**: With one manual stage (intake) and one confidence gate, clarification is the intake-time activity where the developer's decisions happen. After intake, the agent runs unattended — under-specified requirements encountered inside apply are resolved as graded SRAD assumptions in `plan.md`'s `## Assumptions`, not via a clarify session. The intake gate is the guard: a sub-threshold intake never reaches `done`, so gate-checking orchestrators can't enter apply. The SRAD Critical Rule (Unresolved must be asked/bailed) therefore applies at intake-time skills only (`/fab-new`, `/fab-clarify`).
**Rejected**: Keeping a `plan` target for post-apply-entry requirement clarification — re-adds an interactive checkpoint to the unattended segment; rework already flows through `/fab-continue` editing `plan.md` `## Requirements`.
*Introduced by*: 260601-j6cs-merge-spec-into-apply

### Mode Selection by `[AUTO-MODE]` Prefix
**Decision**: Mode is determined by the `[AUTO-MODE]` prefix defined in the Skill Invocation Protocol (in `fab-clarify.md` itself since zc9m — see the next decision). Prefix present = auto mode; absent = suggest mode. No flags.
**Why**: Makes the contract explicit and testable rather than relying on implicit call-context interpretation. Avoids a confusing `--suggest`/`--auto` flag pair with no clear use case for user-invoked auto mode.
**Rejected**: Flag-based mode selection — adds complexity, no user scenario requires it. Implicit call-context detection — unreliable, not testable.
*Updated by*: 260210-nan4-define-auto-mode-signaling; 260611-zc9m-preamble-context-diet (protocol definition relocated into `fab-clarify.md`)

### `[AUTO-MODE]` Protocol Co-located with Its Sole Referencer (zc9m)
**Decision**: The `[AUTO-MODE]` Skill Invocation Protocol (prefix, placement, detection, transitivity) moved from `_preamble.md` into `fab-clarify.md` § Skill Invocation Protocol — its sole referencer since j6cs removed the orchestrator auto-clarify steps. `_preamble.md` keeps a 2-line pointer, and its live § Subagent Dispatch section references the new location. `fab-clarify`'s Auto Mode is **retained** — zero behavior change.
**Why**: The protocol is dormant (no skill currently invokes another with the prefix) yet was paid for by every skill on every invocation via the always-load preamble. Co-locating it with the only skill that consumes it keeps the contract fully specified while removing it from the universal tax. The user explicitly chose **move over delete** at intake: deleting both dormant halves (protocol + Auto Mode) would have been a behavior decision, and retention preserves the machine-readable auto-mode path for future orchestrators at zero cost.
**Rejected**: Deleting the protocol and fab-clarify's Auto Mode (user decision — loses retained-for-future-use behavior). Leaving the protocol in the preamble (every skill pays for a protocol nothing live uses).
*Introduced by*: 260611-zc9m-preamble-context-diet

### Bulk Confirm Before the Zero-Gaps Exit; Grade by Recomputed Composite (c5tr)
**Decision**: Step 1.5 (taxonomy scan) builds the prioritized question queue without stopping; the "No gaps found — artifact looks solid." early exit moved into Step 2's not-triggered branch — emitted only when bulk confirm did not trigger AND the queue is empty. The bulk-confirm Artifact Update grades by the recomputed composite (S → 95, then recompute per `_srad.md` § SRAD Scoring and map via its half-open thresholds — no weights or thresholds restated in `fab-clarify.md`), and both audit-trail writers (Step 2 bulk confirm, Step 5 Q&A) state identical placement/append rules.
**Why**: The old order made bulk confirm unreachable in its primary scenario — a marker-free, Confident-only intake below the 3.0 gate has zero gaps, so Step 1.5 dead-ended at "artifact looks solid" with no path to raise the score. Relocating the exit (rather than deleting it) is the smallest reorder that preserves the solid-artifact UX for genuinely clean intakes. Grading by recomputed composite stops the S→95 upgrade from labeling rows Certain whose composite still sits below 85 — `fab score` reads the table, so fiat labels misstated the gate input. Referencing (not inlining) the formula removes a duplicate of `_srad.md`'s aggregation line — the exact drift class the c5tr batch fixes.
**Rejected**: Deleting the early exit (forces empty question rounds on genuinely clean artifacts). Labeling confirmed rows Certain by fiat (misstates the composite; corrupts scoring inputs). Inlining the composite formula in the update table (re-creates the `_srad` duplication).
*Introduced by*: 260612-c5tr-scaffold-config-truth-srad-coherence

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
