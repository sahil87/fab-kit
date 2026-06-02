# Intake: /fab-proceed — Prefer Conversation Context Over Stale Intakes

**Change**: 260419-1a98-fab-proceed-prefer-context
**Created**: 2026-04-19
**Status**: Draft

## Origin

> Fix /fab-proceed dispatch logic to prefer conversation context over stale unactivated intakes.

Surfaced in a `/fab-discuss` session after another conversation hit the bug: the user was discussing "separator cursor fixes" in a session where an unrelated unactivated intake (`260418-2cjc-right-align-server-name`) existed. Under current dispatch logic, `/fab-proceed` would have auto-activated the unrelated intake rather than creating a new change from the current conversation. The agent in that session correctly detected the conflict and stopped to ask the user, but the skill logic itself should handle this without requiring the agent to catch it.

Mode: conversational. Decisions reached through iterative refinement — user proposed the reordering, assistant identified three gaps (empty-conversation fallback, ambiguous relevance bias, multi-intake scoring, output transparency), user implicitly accepted by saying "yes" to the intake. All four refinements are in scope.

## Why

**Problem**: `/fab-proceed` State Detection (in `src/kit/skills/fab-proceed.md`) short-circuits on *existence* rather than *relevance*. The current order:

1. Active change → use it
2. Unactivated intake exists → activate it (Step 3)
3. Conversation context → create intake (Step 4)

Step 2 dominates Step 3 by structure: Step 4 is only reached when `ls -d fab/changes/*/intake.md | grep -v archive/` returns nothing. This means any stale `/fab-draft` — even one from weeks ago, about a completely unrelated feature — hijacks `/fab-proceed` when the current conversation is about a different topic.

**Consequence if unfixed**: The zero-prompt ergonomics of `/fab-proceed` become a footgun. Users who drafted a backlog item via `/fab-draft`, then months later had a conversation about something unrelated, would have their new conversation silently discarded and the stale draft activated instead. The pipeline would then run on an intake whose content has nothing to do with what the user just discussed.

**Why this approach over alternatives**:
- *Error on ambiguity* (stop and list options): Safest but breaks the zero-prompt posture that makes `/fab-proceed` useful.
- *Remove auto-activation entirely* (require explicit `/fab-switch`): Cleanest architecturally — `/fab-draft` exists precisely to *not* auto-activate, so having `/fab-proceed` auto-activate undoes that separation. But it loses the "picked up where I left off" convenience for the legitimate resume case.
- *Context-as-lens* (this change): Keeps auto-resume for the common resume case, closes the stale-draft trap by checking relevance. Asymmetric failure modes favor this: if we wrongly create a new intake, the old draft is untouched and recoverable; if we wrongly activate a draft, its content gets corrupted by unrelated pipeline work.

## What Changes

### 1. Reordered State Detection in `/fab-proceed`

Current order (src/kit/skills/fab-proceed.md):

```
Step 1: Active change? → use it
Step 2: Branch check (if active change found)
Step 3: Unactivated intake scan → pick most-recent-by-date → activate
Step 4: Conversation context check (ONLY reached if Step 3 found nothing)
```

New order:

```
Step 1: Active change? → use it (unchanged)
Step 2: Branch check (if active change found, unchanged)
Step 3: Assess conversation context (substantive vs empty) — ALWAYS runs when no active change
Step 4: Scan for unactivated intakes
Step 5: Relevance-driven dispatch decision
```

### 2. New Dispatch Table

Replace the existing dispatch table with:

| Active change? | Branch matches? | Conversation context? | Unactivated intake? | Relevant? | Steps to run |
|----------------|----------------|----------------------|---------------------|-----------|--------------|
| Yes | Yes | — | — | — | `/fab-fff` only |
| Yes | No | — | — | — | `/git-branch` → `/fab-fff` |
| No | — | Substantive | None | — | `/fab-new` → `/git-branch` → `/fab-fff` |
| No | — | Substantive | ≥1 | Clearly relevant | `/fab-switch` → `/git-branch` → `/fab-fff` |
| No | — | Substantive | ≥1 | Not clearly relevant | `/fab-new` → `/git-branch` → `/fab-fff` (surface bypassed draft in output) |
| No | — | Empty/thin | ≥1 | — | `/fab-switch` → `/git-branch` → `/fab-fff` (most-recent-by-date) |
| No | — | Empty/thin | None | — | Error — stop |

### 3. Relevance Assessment

When both substantive conversation context AND unactivated intake(s) exist, score each candidate intake against the conversation context:

- Read each candidate's `intake.md` (at minimum: title, Origin, Why, What Changes sections)
- Judge topical overlap with the conversation's subject matter
- "Clearly relevant" requires strong alignment — shared topic, overlapping terminology, consistent scope. Partial or vague overlap does NOT qualify.
- If multiple intakes are clearly relevant, pick the best match; use date-recency (current `sort -t- -k1,1r | head -1`) only as tiebreak among equally-relevant candidates.

**Bias rule**: When relevance is ambiguous (neither clearly relevant nor clearly unrelated), fall through to `/fab-new`. The failure modes are asymmetric:
- False positive (activate unrelated draft): corrupts the draft's content, wastes its original intent, produces pipeline output that conflates two different features.
- False negative (create new when draft was relevant): leaves draft intact and recoverable; user sees `Note: unactivated draft X exists` in output and can run `/fab-switch X` if needed.

### 4. Output Transparency

When `/fab-proceed` bypasses one or more drafts (creating a new intake instead of activating an existing one despite its presence), the output MUST surface the bypassed drafts:

```
/fab-proceed — detecting state...

Note: unactivated draft {name} exists — not relevant to current conversation, left untouched.
{additional Note: lines if multiple drafts bypassed}

Created intake: {new-change-name}
Branch: {new-change-name} (created)

Handing off to /fab-fff...
```

This prevents the "my draft disappeared" confusion and gives the user an escape hatch (`/fab-switch <name>`) if the bypass was wrong.

### 5. Zero-Prompt Preservation

The skill MUST remain zero-prompt — no user interaction at any detection step. Relevance judgment is made by the invoking agent (LLM) based on conversation state and intake content. If the relevance judgment is wrong, the asymmetric-failure-mode bias ensures recovery is cheap (draft remains, user can `/fab-switch` it).

**Substantive vs empty/thin criterion** (reused from `/fab-proceed` Step 4 today): a conversation is **substantive** when it contains at least one of — technical requirements, design decisions, specific values, or problem statements. Anything else (greeting-only, thin/chatty, or literally empty) is **empty/thin**. The dispatch table's "Substantive" and "Empty/thin" rows key off this single definition — no separate word-count or domain-terminology heuristic is introduced. When substantive AND ≥1 unactivated intake exist, the relevance check runs; when empty/thin, the skill auto-activates (or errors if no intake). This avoids a second classifier while preserving the asymmetric-bias rule: borderline "thin-but-maybe-substantive" conversations that fail the Step 4 substantive test fall into the empty/thin branch and auto-activate — but since an empty/thin conversation carries no competing signal, auto-activation cannot corrupt a draft with unrelated content.
<!-- clarified: "thin but non-empty" collapses into the existing substantive-vs-empty definition from /fab-proceed Step 4 — no new heuristic, single classifier -->


### 6. Spec and Memory Sync

Update `docs/specs/skills/SPEC-fab-proceed.md` to reflect the new dispatch order, relevance check, and output format.

Update any `docs/memory/fab-workflow/` files that describe `/fab-proceed` dispatch behavior (at minimum, check `execution-skills.md` and any planning-stage docs).

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update `/fab-proceed` dispatch description — new order, relevance check, output format
- `fab-workflow/planning-skills`: (modify) If it references `/fab-proceed` intake resolution, update to reflect context-priority logic

## Impact

- **Code**: `src/kit/skills/fab-proceed.md` — State Detection section (Steps 1–4), Dispatch Table, Output section
- **Specs**: `docs/specs/skills/SPEC-fab-proceed.md` — dispatch description and flow diagram if present
- **Memory**: `docs/memory/fab-workflow/execution-skills.md`, possibly `planning-skills.md`
- **No changes to**: `/fab-new`, `/fab-switch`, `/fab-draft`, `/fab-fff`, `fab` CLI (Go binary), other skills
- **Distribution**: Kit version bump required (change to `src/kit/skills/`), then `fab sync` deploys to `.claude/skills/`
- **Downstream agents**: `/fab-fff` receives an active change same as today — no contract change

## Open Questions

- Should the relevance check require *both* the intake's title/slug and its Why/What-Changes content to align, or is slug-level overlap sufficient? Leaning toward content-level check (titles/slugs are often terse and misleading), but this adds reading cost for multiple candidates.
- ~~When conversation context is "thin" (short, low-detail but non-empty), should we treat it as empty (auto-activate existing intake) or substantive (run relevance check)?~~ **Resolved**: reuse the existing substantive-vs-empty definition from `/fab-proceed` Step 4 (substantive = has technical requirements, design decisions, specific values, or problem statements). Anything that fails that test is empty/thin. No separate "thin" classifier.
- For the bypassed-draft note — should the output format be standardized to appear before or after the `Created intake:` line? Suggest: before, so the user sees context before the action.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix lives in `src/kit/skills/fab-proceed.md` (canonical source), with spec mirror in `docs/specs/skills/SPEC-fab-proceed.md` | Constitution + project context both require canonical-source edits; `.claude/skills/` is deployed copy only | S:95 R:90 A:95 D:95 |
| 2 | Certain | Zero-prompt posture preserved | Constraint explicitly stated by user: "Preserve zero-prompt posture" | S:100 R:85 A:95 D:95 |
| 3 | Certain | Asymmetric-failure bias favors `/fab-new` on ambiguous relevance | User accepted this rationale in discussion; false-positive corrupts drafts, false-negative is recoverable | S:95 R:85 A:90 D:90 |
| 4 | Certain | Output surfaces bypassed drafts with explicit Note line using exact wording `Note: unactivated draft {name} exists — not relevant to current conversation, left untouched.` | User agreed verbatim in discussion | S:95 R:75 A:90 D:95 |
| 5 | Certain | Relevance judgment made inline by the invoking agent (LLM) — no new infrastructure, no prompts | User explicitly placed "Relevance-matching infrastructure beyond LLM judgment inline in the skill" out of scope | S:95 R:80 A:90 D:90 |
| 6 | Certain | Empty-conversation case falls back to activate-intake (preserves "resume yesterday's draft") | Explicitly designed in the new dispatch table | S:95 R:80 A:90 D:95 |
| 7 | Certain | Change type: `fix` (keyword match "fix" in description) | Deterministic via `/fab-new` Step 6 keyword heuristic | S:95 R:95 A:95 D:95 |
| 8 | Confident | Date-recency tiebreak (`sort -t- -k1,1r \| head -1`) retained only for equally-relevant candidates | Preserves current behavior as fallback when relevance cannot disambiguate | S:85 R:75 A:80 D:80 |
| 9 | Confident | Relevance check reads full intake content (title + Origin + Why + What Changes), not just slug | Slugs are terse and often misleading; content-level check is more reliable. Cost is bounded (few candidates) | S:80 R:75 A:80 D:75 |
| 10 | Confident | Bypassed-draft Note line appears BEFORE the `Created intake:` line in output | Reader-flow: context before action. Minor formatting, easily reversible | S:75 R:85 A:75 D:80 |
| 11 | Confident | Memory updates scoped to `fab-workflow/execution-skills.md` (verified via grep: only other hit is `fab-workflow/index.md`, which is an auto-maintained domain index) | Grepped `docs/memory/` for `fab-proceed` — exactly 2 files match | S:85 R:80 A:85 D:85 |
| 12 | Certain | "Thin but non-empty" conversation classified via the existing `/fab-proceed` Step 4 substantive test: has technical requirements, design decisions, specific values, or problem statements → substantive (run relevance check); otherwise → empty/thin (auto-activate). No separate classifier. | Reuses the existing substantive-vs-empty definition from `/fab-proceed` Step 4, so no new heuristic is introduced. Asymmetric-bias rule is preserved: auto-activation only happens when the conversation carries no competing signal, so a draft cannot be corrupted by unrelated content. Locked at intake. | S:95 R:80 A:90 D:90 |

12 assumptions (8 certain, 4 confident, 0 tentative, 0 unresolved).
