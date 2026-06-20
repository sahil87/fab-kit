---
name: fab-clarify
description: "Refine the intake artifact — resolve gaps, ambiguities, or [NEEDS CLARIFICATION] markers without advancing."
helpers: [_srad]
---

# /fab-clarify [<change-name>]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Contents

- [Purpose](#purpose)
- [Arguments](#arguments)
- [Pre-flight & Stage Guard](#pre-flight--stage-guard)
- [Suggest Mode (User Invocation)](#suggest-mode-user-invocation)
- [Skill Invocation Protocol](#skill-invocation-protocol)
- [Auto Mode](#auto-mode)
- [Error Handling](#error-handling)
- [Key Properties](#key-properties)

---

## Purpose

Deepen and refine the **intake** artifact (`intake.md`) without advancing. Clarification is an intake-only, human-facing activity: it is where the developer's decisions and disambiguation happen, gated by the single intake confidence gate. There is no post-intake clarify — inside apply, the agent resolves ambiguity inline as graded SRAD assumptions in `plan.md`, not via this skill. Two modes:

- **Suggest mode** (user invocation) — interactive question flow with recommendations
- **Auto mode** — autonomous resolution, returns machine-readable result (retained for future use; see § Skill Invocation Protocol → Currently Applicable)

Mode determined by `[AUTO-MODE]` prefix (see § Skill Invocation Protocol below). Safe to call multiple times.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change (see `_preamble.md` > Change-name override). `.fab-status.yaml` unchanged.

`/fab-clarify` operates only on `intake.md`; the legacy `spec`/`plan`/`tasks` targets were removed (the spec stage is gone; under-specified requirements at apply become inline SRAD assumptions). Any positional argument is treated as a change name.

---

## Pre-flight & Stage Guard

Run preflight per `_preamble.md` §2.

- **Intake** is the only stage `/fab-clarify` operates at. With `intake` state `active` or `ready`, scan `intake.md` (scope boundaries, affected areas, blocking questions, impact, memory coverage).
- **Post-intake stages** (`apply`, `review`, `hydrate`, `ship`, `review-pr`): `/fab-clarify` does not apply — STOP (see Error Handling for the message). If `intake.md` is missing entirely, STOP with the missing-intake message (Error Handling).

---

## Suggest Mode (User Invocation)

### Step 1: Read Target Artifact

Read `intake.md`. If missing: STOP (missing-intake message, see Error Handling).

### Step 1.5: Taxonomy Scan

Scan `intake.md` for gaps, `[NEEDS CLARIFICATION]`, and `<!-- assumed: ... -->` markers. Categories:

- **Intake**: scope boundaries, affected areas, blocking questions, impact, memory coverage

For `<!-- assumed: ... -->` markers, frame current assumption as recommended option with alternatives.

Build **prioritized question queue** (max 5). Present tentative assumption questions (from `<!-- assumed: ... -->` markers) first. Do NOT stop here when the queue is empty — bulk confirm (Step 2) is evaluated first; the zero-gaps early exit lives in Step 2's not-triggered branch. (A below-gate, Confident-only intake has zero gaps but still needs the bulk-confirm flow.)

### Step 2: Bulk Confirm (Confident Assumptions)

> **Note**: If Step 1.5 (Taxonomy Scan) presented tentative questions, this flow runs on the already-updated artifact. Some gaps may have been resolved by tentative resolution.

After the taxonomy scan, parse the `## Assumptions` table and count assumptions by grade. Trigger bulk confirm when BOTH:

1. `confident >= 3`
2. `confident > tentative + unresolved`

If NOT triggered: when Step 1.5's queue is also empty (zero gaps), output "No gaps found — artifact looks solid." with the Next line and stop; otherwise skip to Step 3.

#### Display

Present all Confident assumptions as a numbered list using the original `#` column from the Assumptions table:

```
## Confident Assumptions ({N} items — primary confidence drag)

Review each and respond with: ✓ (confirm), a new value, or ? (explain).

{original_#}. {Decision} — {Rationale}
...
```

Do NOT use `AskUserQuestion`. Display as plain text and read the user's next conversational message as the response.

#### Response Parsing

Recognize these formats (case-insensitive for keywords):

| Format | Meaning |
|--------|---------|
| `{#}. ✓` or `{#}. ok` or `{#}. yes` | Confirm |
| `{#}.` (bare number with period) | Confirm |
| `{#}. {free text}` | Change value |
| `{#}. ?` or `{#}. explain` | Request explanation |
| `{start}-{end}. ✓` or `{start}-{end}. ok` | Confirm range |
| `all ✓` or `all ok` or `all yes` | Confirm all |

Items not mentioned remain Confident (unchanged).

#### Explanation Re-prompt

For items marked `?` or `explain`:

1. Provide a brief inline explanation of the assumption's reasoning and implications
2. Re-prompt for ONLY the unexplained items: `Still pending: #{#}. {Decision} — respond with ✓ or a new value`
3. Accept the same response formats

At most one round of re-prompting. After the re-prompt response, unresolved items remain Confident.

#### Artifact Update

For each resolved item, update the `## Assumptions` table in place:

| Action | Scores | Grade | Rationale |
|--------|--------|-------|-----------|
| Confirmed | S → 95 | recompute composite, grade by threshold | `Clarified — user confirmed` |
| Changed | S → 95 | recompute composite, grade by threshold | `Clarified — user changed to {value}` |
| Explained then confirmed | S → 95 | recompute composite, grade by threshold | `Clarified — user confirmed after explanation` |

For changed items, also update the Decision column with the user's new value. Only the S dimension changes to 95; R, A, D remain unchanged. **Grade by the recomputed composite, not by fiat**: recompute the composite per `_srad.md` § SRAD Scoring (with the new S) and grade by its half-open thresholds. A confirmed row whose recomputed composite still falls short of the Certain band remains Confident — the Rationale still records the confirmation.

#### Audit Trail

Append a `### Session {YYYY-MM-DD} (bulk confirm)` block under `## Clarifications`. **Placement/append rule** (shared with Step 5): append to the existing `## Clarifications` section if present; create it (immediately before `## Assumptions`) if not; skip if 0 items were resolved.

```markdown
### Session {YYYY-MM-DD} (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| {#} | Confirmed | — |
| {#} | Changed | "{new value}" |
| {#} | Confirmed | After explanation |
```

After bulk confirm completes (including any re-prompts), proceed to Step 3.

### Step 3: Ask Questions One at a Time

For each remaining non-tentative question from the Step 1.5 queue, present:
- The question text with its position in the queue (e.g., 1 of 3)
- A recommended option with brief reasoning
- Alternatives (if applicable)

Allow the user to accept the recommendation, pick an alternative, provide a free-text answer, or stop early. Use whatever interaction method is natural for your environment.

### Step 4: Process Answer and Update

1. Update artifact in place: replace markers with resolved content, add `<!-- clarified: ... -->` for significant changes
2. Re-grade the resolved entry's row in the `## Assumptions` table **by recomputed composite, not by fiat** — the same rule as Step 2's Artifact Update (set S → 95, R/A/D unchanged; recompute per `_srad.md` § SRAD Scoring; grade by the half-open thresholds). A direct answer typically lands the row in Certain, but a row whose recomputed composite stays below the Certain band keeps its banded grade.
3. Present next question or proceed to Step 5 after queue exhaustion / 5th answer / early termination

### Step 5: Audit Trail

Append `## Clarifications > ### Session {YYYY-MM-DD}` with Q&A pairs — same placement/append rule as Step 2's bulk-confirm trail (skip if 0 answers).

### Step 6: Coverage Summary

```
Clarification complete.

| Category | Count |
|----------|-------|
| Resolved | {N} |
| Clear | {N} |
| Deferred | {N} |
| Outstanding | {N} |

Next: {per state table — current state, since clarify is non-advancing}
```

### Step 7: Recompute Confidence

Always run `fab score --stage intake <change>` after resolving assumptions — intake is the sole scoring source, and clarify operates only at intake. This re-persists the authoritative intake confidence. Both Suggest and Auto modes recompute (Auto Mode step 4).

### Step 8: Do NOT Advance Stage

Only update `confidence` and `last_updated` in `.status.yaml`.

---

## Skill Invocation Protocol

When one skill invokes another internally, the calling skill MUST signal the invocation mode explicitly using an instruction prefix. This makes the contract between skills explicit and testable rather than relying on implicit "call context" interpretation. (No live flow currently does this — see Currently Applicable below.)

### Protocol

1. **Prefix**: `[AUTO-MODE]`
2. **Placement**: The calling skill includes `[AUTO-MODE]` as the **first line** of the invocation prompt / instruction to the called skill.
3. **Detection**: The called skill checks for the `[AUTO-MODE]` prefix at the start of its invocation context.
   - **If present**: Enter autonomous mode (no user interaction, machine-readable result).
   - **If absent**: Enter default/interactive mode (user-facing, structured questions).
4. **Transitivity**: When skills chain, each link applies the prefix independently.

### Currently Applicable

No skill currently invokes another with the `[AUTO-MODE]` prefix. The former
`/fab-fff` → `/fab-clarify` and `/fab-ff` → `/fab-clarify` auto-invocations were
removed in 1.10.0: clarification is an intake-only, human-facing activity, so no
clarify step runs inside the automated post-intake bracket (apply → review →
hydrate → ship → review-pr). The protocol itself remains defined for future use.

User-invoked skills never carry the `[AUTO-MODE]` prefix, so called skills default to interactive mode.

To add new mode signals, define new bracketed prefixes (e.g., `[BATCH-MODE]`) here. Pattern: one prefix per mode, first-line placement, absence means default.

---

## Auto Mode

> **Note**: Bulk confirm (Step 2) is Suggest Mode only. Auto Mode skips it — there is no user to confirm with. Retained for future use only (see § Skill Invocation Protocol → Currently Applicable); operates on `intake.md` only.

1. **Read `intake.md`** (same as Suggest Step 1)
2. **Autonomous gap resolution**: Same intake taxonomy scan. Resolvable from context → resolve + `<!-- clarified: ... -->`. Needs user input → `<!-- blocking: ... -->`. Minor → leave as-is.
3. **Return result**: `{resolved: N, blocking: N, non_blocking: N}`. If `blocking > 0`, include `blocking_issues: [...]`.
4. **Non-advancing**: recompute the intake score (`fab score --stage intake <change>`) and update `last_updated`.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Stage is post-intake (apply/review/hydrate/ship/review-pr) | "Clarification is intake-only. Run /fab-continue for rework, or edit plan.md `## Requirements`. Reset via /fab-continue intake to re-clarify the intake." |
| `intake.md` missing | "No intake.md found. Run /fab-new to create the intake first." |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Idempotent? | Yes |
| Modifies artifact? | Yes — edits in place |
| `.status.yaml` updates | `confidence` + `last_updated` only |
