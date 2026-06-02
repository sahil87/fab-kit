# Spec: /fab-proceed — Prefer Conversation Context Over Stale Intakes

**Change**: 260419-1a98-fab-proceed-prefer-context
**Created**: 2026-04-19
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Introduce user prompts into `/fab-proceed` — the skill remains zero-prompt.
- Build new relevance-matching infrastructure (classifiers, embeddings, stored indexes). Relevance is judged inline by the invoking agent.
- Change behavior of `/fab-new`, `/fab-switch`, `/fab-draft`, `/fab-fff`, or the `fab` Go binary.
- Alter the semantics of `/fab-proceed` when an active change already exists. All active-change branches are unchanged.
- Introduce a word-count, turn-count, or terminology-match heuristic for conversation classification. The existing substantive-vs-empty test (Step 4 today) is the single classifier.

## /fab-proceed: State Detection

### Requirement: Detection Order

`/fab-proceed` SHALL execute state detection in the following order, with conversation context assessed before unactivated-intake lookup:

1. **Active change check** — run `fab resolve --folder 2>/dev/null`; on success, capture the folder name.
2. **Branch check** — if an active change was found, compare `git branch --show-current` to the resolved folder name.
3. **Conversation context assessment** — if no active change, classify the current conversation as *substantive* or *empty/thin* per the criterion in the next requirement.
4. **Unactivated-intake scan** — if no active change, enumerate candidate intakes via `ls -d fab/changes/*/intake.md 2>/dev/null | grep -v archive/ | sed 's|fab/changes/||;s|/intake.md||'`. Preserve the date-descending sort (`sort -t- -k1,1r`) for tiebreak use only.
5. **Dispatch decision** — combine the signals from 1–4 per the dispatch table in the next requirement.

The skill MUST NOT prompt the user at any step. Steps 3 and 4 MAY execute in either order; their outputs are independent.

#### Scenario: Active change short-circuits later steps

- **GIVEN** `fab resolve --folder` returns a folder name
- **WHEN** `/fab-proceed` runs state detection
- **THEN** Step 2 (branch check) runs
- **AND** Steps 3 and 4 SHALL NOT run
- **AND** dispatch uses only the active-change branches of the dispatch table

#### Scenario: No active change triggers context and intake assessment

- **GIVEN** `fab resolve --folder` exits non-zero
- **WHEN** `/fab-proceed` runs state detection
- **THEN** Step 3 (conversation context assessment) SHALL run
- **AND** Step 4 (unactivated-intake scan) SHALL run
- **AND** the dispatch decision in Step 5 uses both results

### Requirement: Conversation Classification

A conversation SHALL be classified as **substantive** when it contains at least one of:
- technical requirements,
- design decisions,
- specific values (config structures, API shapes, exact behaviors), or
- problem statements with enough detail to generate an intake.

All other conversations (greeting-only, chatty, literally empty) SHALL be classified as **empty/thin**. This is the single classifier used for `/fab-proceed` — no separate "thin but non-empty" tier, no word-count or terminology heuristic.

This definition is identical to the existing `/fab-proceed` Step 4 "substantive context" criterion and SHALL be reused unchanged.

#### Scenario: Substantive conversation

- **GIVEN** the prior conversation includes design decisions for a feature
- **WHEN** `/fab-proceed` classifies the conversation
- **THEN** the classification is `substantive`

#### Scenario: Empty-thin conversation

- **GIVEN** the prior conversation contains only greetings and a bare command invocation
- **WHEN** `/fab-proceed` classifies the conversation
- **THEN** the classification is `empty/thin`

### Requirement: Dispatch Table

`/fab-proceed` SHALL select the step sequence from the table below. This replaces the current 5-row dispatch table in `src/kit/skills/fab-proceed.md`.

| Active change? | Branch matches? | Conversation | Unactivated intake? | Relevant? | Steps to run |
|----------------|-----------------|--------------|---------------------|-----------|--------------|
| Yes | Yes | — | — | — | `/fab-fff` only |
| Yes | No | — | — | — | `/git-branch` → `/fab-fff` |
| No | — | Substantive | None | — | `/fab-new` → `/git-branch` → `/fab-fff` |
| No | — | Substantive | ≥1 | Clearly relevant | `/fab-switch` → `/git-branch` → `/fab-fff` |
| No | — | Substantive | ≥1 | Not clearly relevant | `/fab-new` → `/git-branch` → `/fab-fff` (bypass notes in output) |
| No | — | Empty/thin | ≥1 | — | `/fab-switch` → `/git-branch` → `/fab-fff` (pick by date-recency) |
| No | — | Empty/thin | None | — | Error — stop |

The `Relevant?` column SHALL be evaluated only when the Conversation column is Substantive AND the Unactivated intake column is ≥1.

#### Scenario: Active change with matching branch

- **GIVEN** active change `AAAA` exists and the current branch equals the change folder name
- **WHEN** `/fab-proceed` runs
- **THEN** only `/fab-fff` is dispatched

#### Scenario: Substantive conversation with relevant draft

- **GIVEN** no active change
- **AND** the conversation includes problem statements about feature `X`
- **AND** an unactivated intake `bbbb` exists whose content is clearly about feature `X`
- **WHEN** `/fab-proceed` runs
- **THEN** the skill dispatches `/fab-switch` → `/git-branch` → `/fab-fff` against intake `bbbb`

#### Scenario: Substantive conversation with unrelated draft

- **GIVEN** no active change
- **AND** the conversation includes problem statements about feature `Y`
- **AND** an unactivated intake `cccc` exists whose content is about feature `Z` (unrelated to `Y`)
- **WHEN** `/fab-proceed` runs
- **THEN** the skill dispatches `/fab-new` → `/git-branch` → `/fab-fff` using the conversation about `Y`
- **AND** intake `cccc` is left untouched
- **AND** the output includes a bypass note naming `cccc`

#### Scenario: Empty conversation with existing draft

- **GIVEN** no active change
- **AND** the conversation is empty/thin
- **AND** one or more unactivated intakes exist
- **WHEN** `/fab-proceed` runs
- **THEN** the skill dispatches `/fab-switch` → `/git-branch` → `/fab-fff` against the most-recent-by-date-prefix intake
- **AND** the skill SHALL NOT produce a bypass note

#### Scenario: Empty conversation with no draft

- **GIVEN** no active change
- **AND** the conversation is empty/thin
- **AND** no unactivated intakes exist
- **WHEN** `/fab-proceed` runs
- **THEN** the skill exits with the error message: `Nothing to proceed with — start a discussion or run /fab-new (or /fab-draft) first.`

### Requirement: Relevance Assessment

When both a substantive conversation AND ≥1 unactivated intake exist, `/fab-proceed` SHALL score each candidate intake for topical relevance to the conversation using the following procedure:

1. For each non-archived candidate folder, read at minimum the intake's title heading, `## Origin` section, `## Why` section, and `## What Changes` section.
2. Judge topical overlap between each intake and the conversation subject matter. Overlap SHALL require shared topic, overlapping terminology, and consistent scope — partial or vague overlap MUST NOT qualify as "clearly relevant."
3. Classify each candidate as **clearly relevant** or **not clearly relevant**.
4. If ≥1 candidate is clearly relevant: select the best match. If multiple candidates are equally clearly relevant, use the date-descending prefix tiebreak (`sort -t- -k1,1r | head -1`).
5. If no candidate is clearly relevant: fall through to `/fab-new`; all non-matching drafts SHALL be surfaced as bypass notes in the output.

**Asymmetric-bias rule**: When a candidate's relevance is genuinely ambiguous (neither clearly relevant nor clearly unrelated), it MUST be classified as *not clearly relevant*. This biases toward creating a new intake (recoverable failure: draft remains intact) over activating an existing draft (unrecoverable failure: draft content conflates with unrelated pipeline work).

Relevance judgment is performed by the invoking agent inline — no external classifier, embedding index, or fab-binary subcommand is added.

#### Scenario: Single clearly-relevant candidate

- **GIVEN** substantive conversation about feature `X`
- **AND** one unactivated intake clearly about feature `X`
- **WHEN** relevance assessment runs
- **THEN** the intake is classified `clearly relevant`
- **AND** it is selected for activation

#### Scenario: Multiple clearly-relevant candidates

- **GIVEN** substantive conversation about feature `X`
- **AND** two unactivated intakes both clearly about feature `X`
- **WHEN** relevance assessment runs
- **THEN** both are classified `clearly relevant`
- **AND** the one with the higher `YYMMDD` prefix is selected for activation
- **AND** the other is left untouched
- **AND** the output SHALL NOT contain a bypass Note for the non-selected candidate (activation path emits no Notes — see Requirement: Output Format)

#### Scenario: Ambiguous candidate resolves to not-relevant

- **GIVEN** substantive conversation about feature `X`
- **AND** an unactivated intake that partially overlaps feature `X` but also covers unrelated concerns
- **WHEN** relevance assessment runs
- **THEN** the intake is classified `not clearly relevant`
- **AND** the skill dispatches `/fab-new` instead
- **AND** the intake is surfaced as a bypass note

### Requirement: Output Format

When `/fab-proceed` bypasses one or more unactivated intakes (i.e., runs `/fab-new` despite their presence), the output SHALL include one Note line per bypassed draft, using the exact wording:

```
Note: unactivated draft {name} exists — not relevant to current conversation, left untouched.
```

Each bypass Note line SHALL appear BEFORE the `Created intake:` line so readers see the context before the action. When multiple drafts are bypassed, Note lines SHALL be emitted in the same order as the intake scan (date-descending).

When no drafts are bypassed (or the skill is activating an existing draft), no Note lines SHALL be emitted.

The rest of the output format (step reports, handoff line) is unchanged from today.

#### Scenario: Single bypass

- **GIVEN** substantive conversation, one unactivated intake `260110-abcd-xyz` not clearly relevant
- **WHEN** `/fab-proceed` runs
- **THEN** the output contains the line `Note: unactivated draft 260110-abcd-xyz exists — not relevant to current conversation, left untouched.`
- **AND** that line appears before the `Created intake: {new-change-name}` line

#### Scenario: Multiple bypasses

- **GIVEN** substantive conversation, two unactivated intakes `260120-aaaa-p` and `260115-bbbb-q`, neither clearly relevant
- **WHEN** `/fab-proceed` runs
- **THEN** the output contains two Note lines, in date-descending order (260120 first, then 260115)
- **AND** both appear before the `Created intake:` line

#### Scenario: Activation path emits no bypass notes

- **GIVEN** substantive conversation matches an unactivated intake clearly
- **WHEN** `/fab-proceed` activates that intake
- **THEN** the output SHALL NOT contain any `Note: unactivated draft` lines

### Requirement: Zero-Prompt Preservation

`/fab-proceed` SHALL NOT prompt the user at any step of state detection, relevance assessment, or dispatch. All decisions are made by the invoking agent based on conversation content, intake content, and git/fab state.

If a decision is genuinely irresolvable (e.g., malformed intake files), the skill SHALL error with a descriptive message rather than prompt.

#### Scenario: Ambiguous relevance does not prompt

- **GIVEN** substantive conversation and a partially-overlapping unactivated intake
- **WHEN** `/fab-proceed` runs
- **THEN** the skill SHALL NOT ask the user to disambiguate
- **AND** the asymmetric-bias rule resolves the ambiguity (classify as not clearly relevant)

## Design Decisions

1. **Decision**: Reorder state detection so conversation context assessment runs before unactivated-intake lookup, and combine both signals in a relevance-driven dispatch table.
   - *Why*: The current short-circuit (intake exists → activate) ignores conversation relevance. A stale `/fab-draft` hijacks `/fab-proceed` when the current conversation is about a different topic. Reordering makes conversation context the interpretive lens for intakes.
   - *Rejected*: *Error on ambiguity* — breaks the zero-prompt posture that makes `/fab-proceed` ergonomic. *Remove auto-activation entirely* — cleanest architecturally but loses the "resume yesterday's draft" convenience; `/fab-switch` would become mandatory.

2. **Decision**: When relevance is ambiguous, classify as not clearly relevant (bias toward `/fab-new`).
   - *Why*: Asymmetric failure modes. False positive (activate unrelated draft) corrupts the draft's content and conflates two features; recovery requires manual rollback. False negative (create new when draft was relevant) leaves the draft intact; user sees the bypass note and can run `/fab-switch` to recover.
   - *Rejected*: *Coin-flip between activation and creation* — treats recoverable and unrecoverable failures symmetrically. *Prompt the user* — breaks zero-prompt posture.

3. **Decision**: Reuse the existing Step 4 substantive-vs-empty criterion as the single conversation classifier; no separate "thin but non-empty" tier.
   - *Why*: Introducing a second classifier (word-count, terminology-match) adds a new heuristic to reason about and maintain. The existing criterion is already used for the "no intake" branch today and handles the edge cases acceptably: borderline conversations that fail the substantive test fall into the empty/thin branch and auto-activate, but since the conversation carries no competing signal, auto-activation cannot conflict with unrelated content.
   - *Rejected*: *Word/turn-count threshold* — arbitrary threshold, drifts with conversation style. *Domain-terminology match against intake* — circular: requires the relevance check to drive conversation classification.

4. **Decision**: Relevance check reads title + Origin + Why + What Changes sections of each candidate intake.
   - *Why*: Slug/folder-name alone is terse and often misleading. Reading the substantive body sections gives reliable topical signal. Cost is bounded — few candidates typically exist (single-digit folder count in active use).
   - *Rejected*: *Slug-only match* — too brittle (terse slugs routinely misrepresent content). *Full intake read* — diminishing returns; the late sections (Assumptions, Open Questions) don't improve topical judgment.

5. **Decision**: Bypass Note lines appear BEFORE the `Created intake:` line in output.
   - *Why*: Reader-flow — users see the context (why a new intake was created despite existing drafts) before the action line. Helps the user form the right mental model and reach for `/fab-switch` if the bypass was wrong.
   - *Rejected*: *Notes after `Created intake:`* — buries context after the primary action line.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Edits in `src/kit/skills/fab-proceed.md` (canonical source) + `docs/specs/skills/SPEC-fab-proceed.md` mirror | Constitution + context explicit; `.claude/skills/` is deployed copy only | S:95 R:90 A:95 D:95 |
| 2 | Certain | Zero-prompt posture preserved at every detection, relevance, and dispatch step | User constraint explicit; spec Requirement §/fab-proceed Zero-Prompt | S:100 R:85 A:95 D:95 |
| 3 | Certain | Asymmetric-bias rule: ambiguous relevance → not clearly relevant → `/fab-new` | Confirmed from intake #3; codified in spec Requirement §Relevance Assessment | S:95 R:85 A:90 D:90 |
| 4 | Certain | Exact Note wording: `Note: unactivated draft {name} exists — not relevant to current conversation, left untouched.` | User agreed verbatim; pinned as Requirement §Output Format | S:95 R:80 A:90 D:95 |
| 5 | Certain | Relevance judgment is inline LLM judgment, no new fab-binary subcommand or infrastructure | User explicitly scoped out; Requirement §Relevance Assessment | S:95 R:80 A:90 D:90 |
| 6 | Certain | Empty-thin + ≥1 intake → activate most-recent-by-date (preserves resume flow) | Confirmed from intake #6; Requirement §Dispatch Table row 6 | S:95 R:80 A:90 D:95 |
| 7 | Certain | "Thin but non-empty" collapses into the existing Step 4 substantive test — single classifier | Clarified from intake Tentative #12; Requirement §Conversation Classification | S:95 R:85 A:90 D:95 |
| 8 | Certain | Relevance scans title + Origin + Why + What Changes (not slug, not full intake) | Upgraded from intake Confident #9; locked in Requirement §Relevance Assessment; tradeoff analyzed in Design Decision #4 | S:90 R:80 A:85 D:90 |
| 9 | Certain | Bypass Note line placement: BEFORE `Created intake:` line | Upgraded from intake Confident #10; Requirement §Output Format and Design Decision #5 | S:90 R:85 A:85 D:90 |
| 10 | Certain | Memory scope: `docs/memory/fab-workflow/execution-skills.md` only (verified: `planning-skills.md` contains no `/fab-proceed` dispatch content) | Confirmed via grep during spec generation; the `execution-skills.md` section on `/fab-proceed` is the authoritative dispatch description | S:95 R:90 A:95 D:95 |
| 11 | Certain | Date-recency tiebreak used (a) to pick among equally-relevant candidates in the substantive branch and (b) to pick the sole candidate in the empty/thin branch | Confirmed from intake Confident #8; codified in Requirement §Relevance Assessment step 4 and Dispatch Table row 6 | S:95 R:90 A:90 D:95 |
| 12 | Certain | Change type: `fix` (keyword heuristic, persisted to `.status.yaml`) | Deterministic; carried over from intake #7 | S:95 R:95 A:95 D:95 |
| 13 | Certain | Steps 3 (conversation classification) and 4 (intake scan) MAY run in either order — their outputs are independent | Inherent from the redesign; called out explicitly in Requirement §Detection Order to avoid implementation ambiguity | S:90 R:85 A:90 D:90 |
| 14 | Confident | Multiple bypass Notes are emitted in date-descending order (matching the intake scan order) | Consistent with `sort -t- -k1,1r` already used; Requirement §Output Format locks the ordering | S:85 R:85 A:85 D:85 |
| 15 | Confident | `/fab-proceed` output for the activation path (single clearly-relevant candidate) continues to report the activated name and `/fab-switch` step — no new fields added | Preserves existing output for the common resume case; only the bypass branch adds output | S:80 R:85 A:80 D:85 |
| 16 | Confident | Error-branch message (empty/thin + no intake) is unchanged from today: `Nothing to proceed with — start a discussion or run /fab-new (or /fab-draft) first.` | Preserves current error wording; no user agreement sought since it's a strict preservation | S:85 R:90 A:85 D:85 |

16 assumptions (13 certain, 3 confident, 0 tentative, 0 unresolved).
