---
name: fab-clarify
description: "Refine the current stage artifact — resolve gaps, ambiguities, or [NEEDS CLARIFICATION] markers without advancing."
---

# /fab-clarify [<change-name>] [<target-artifact>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Deepen and refine the current stage artifact without advancing. Two modes:

- **Suggest mode** (user invocation) — interactive question flow with recommendations
- **Auto mode** (internal `fab-ff` call) — autonomous resolution, returns machine-readable result

Mode determined by `[AUTO-MODE]` prefix (see `_context.md` > Skill Invocation Protocol). Safe to call multiple times.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change (see `_context.md` > Change-name override). `fab/current` unchanged.
- **`<target-artifact>`** *(optional)* — `brief`, `spec`, or `tasks`. **Required** at post-planning stages. Defaults to current stage's artifact at planning stages.

Disambiguation: matches `brief`/`spec`/`tasks` → target artifact; anything else → change name. Both can be provided.

---

## Pre-flight & Stage Guard

Run preflight per `_context.md` §2.

- **Planning stages** (`brief`, `spec`, `tasks`) — defaults to current stage's artifact; `<target-artifact>` overrides.
- **Post-planning** (`apply`, `review`, `hydrate`) — requires `<target-artifact>`. If missing, prompt: "Which planning artifact to clarify? (1) spec, (2) tasks, (3) brief"

---

## Suggest Mode (User Invocation)

### Step 1: Read Target Artifact

Resolve file (`brief.md`, `spec.md`, or `tasks.md`). If missing: STOP with "No {artifact} found. Run /fab-continue to generate it first."

### Step 2: Taxonomy Scan

Scan for gaps, `[NEEDS CLARIFICATION]`, and `<!-- assumed: ... -->` markers. Categories by target:

- **Brief**: scope boundaries, affected areas, blocking questions, impact, memory coverage
- **Spec**: requirement precision (RFC 2119), scenario coverage (GIVEN/WHEN/THEN), edge cases, deprecated requirements, memory cross-references
- **Tasks**: completeness vs spec, granularity, dependencies, file paths, `[P]` markers

For `<!-- assumed: ... -->` markers, frame current assumption as recommended option with alternatives.

Build **prioritized question queue** (max 5). If zero gaps: "No gaps found — artifact looks solid." with Next line, stop.

### Step 3: Present Questions One at a Time

```
**Question {N} of {total}**: {question text}

Recommendation: {option} — {reasoning}

| # | Option | Description |
|---|--------|-------------|

Reply with a number, "yes"/"recommended" to accept, or your own answer.
```

Short-answer variant uses `Suggested answer:` instead of table.

### Step 4: Process Answer and Update

1. Interpret: `yes`/`y`/`recommended` → accept; number → select; `done`/`good`/`no more` → early termination; free text → custom
2. Update artifact in place: replace markers with resolved content, add `<!-- clarified: ... -->` for significant changes
3. Reclassify resolved entry to `Certain` in `## Assumptions` table
4. Present next question or proceed to Step 5 after queue exhaustion / 5th answer

### Step 5: Audit Trail

Append `## Clarifications > ### Session {YYYY-MM-DD}` with Q&A pairs. Append to existing section if present; create if not; skip if 0 answers.

### Step 6: Coverage Summary

```
Clarification complete.

| Category | Count |
|----------|-------|
| Resolved | {N} |
| Clear | {N} |
| Deferred | {N} |
| Outstanding | {N} |

Next: /fab-clarify or /fab-continue or /fab-ff
```

### Step 7: Recompute Confidence

Run `fab/.kit/scripts/_calc-score.sh $change_dir` if `spec.md` exists in the change directory. Skip this step if at brief stage (no spec yet). Auto mode does not invoke this script.

### Step 8: Do NOT Advance Stage

Only update `confidence` and `last_updated` in `.status.yaml`.

---

## Auto Mode (Internal fab-ff Call)

1. **Read target artifact** (same as Suggest Step 1)
2. **Autonomous gap resolution**: Same taxonomy scan. Resolvable from context → resolve + `<!-- clarified: ... -->`. Needs user input → `<!-- blocking: ... -->`. Minor → leave as-is.
3. **Return result**: `{resolved: N, blocking: N, non_blocking: N}`. If `blocking > 0`, include `blocking_issues: [...]`.
4. **Non-advancing**: Only update `last_updated`.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Post-planning stage, no `<target-artifact>` | Prompt for artifact selection |
| Artifact file missing | "No {artifact} found. Run /fab-continue to generate it first." |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Idempotent? | Yes |
| Modifies artifact? | Yes — edits in place |
| `.status.yaml` updates | `confidence` + `last_updated` only |
