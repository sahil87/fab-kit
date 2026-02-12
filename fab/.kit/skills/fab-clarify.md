---
name: fab-clarify
description: "Refine the current stage artifact — resolve gaps, ambiguities, or [NEEDS CLARIFICATION] markers without advancing."
---

# /fab-clarify

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Deepen and refine the current stage artifact without advancing to the next stage. Operates in two modes:

- **Suggest mode** (user invocation) — interactive, structured question flow with recommendations
- **Auto mode** (internal `fab-ff` call) — autonomous resolution, returns machine-readable result

Safe to call multiple times — each invocation refines further.

---

## Mode Selection

Mode is determined by the **`[AUTO-MODE]` prefix** defined in the Skill Invocation Protocol section of `_context.md`. There are no `--suggest` or `--auto` flags.

**Detection logic**: Check the first line of the invocation context for the `[AUTO-MODE]` prefix.

| Invocation context | Mode |
|--------------------|------|
| `[AUTO-MODE]` prefix **present** (e.g., `/fab-ff` invoking internally) | **Auto** — autonomous, returns structured result |
| `[AUTO-MODE]` prefix **absent** (e.g., user invokes `/fab-clarify` directly) | **Suggest** — interactive, one question at a time |

See `_context.md` > Skill Invocation Protocol for the full protocol definition.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence`

Use the `stage` field from preflight output for the Stage Guard below (do not re-read `.status.yaml`).

---

## Stage Guard

The current stage must be one of:

- `brief`
- `spec`
- `tasks`

**If the stage is `apply`, `review`, or `archive`, STOP.** Output:

> `Stage is "{stage}" — clarify applies to planning artifacts only (brief, spec, tasks). Use /fab-review to validate implementation instead.`

---

## Context Loading

Context varies by the current stage. Load only what is relevant:

### Brief stage

- `fab/config.yaml` — project config, tech stack
- `fab/constitution.md` — project principles and constraints
- `fab/changes/{name}/brief.md` — the artifact to refine
- `fab/docs/index.md` — documentation landscape

### Spec stage

- Everything from brief context above, plus:
- `fab/changes/{name}/spec.md` — the artifact to refine (if exists)
- Specific centralized docs referenced by the brief's **Affected Docs** section

### Tasks stage

- Everything from spec context above, plus:
- `fab/changes/{name}/spec.md` — the completed spec (for reference)
- `fab/changes/{name}/tasks.md` — the artifact to refine

---

## Suggest Mode (User Invocation)

When the user invokes `/fab-clarify` directly, follow this flow.

### Step 1: Identify the Current Artifact

Based on the current stage, determine which artifact file to refine:

| Stage | Artifact file(s) |
|-------|-----------------|
| `brief` | `brief.md` |
| `spec` | `spec.md` (also scans brief.md for cross-stage gaps) |
| `tasks` | `tasks.md` |

Read the artifact file. If it does not exist, STOP with:

> `No {artifact} found for this stage. Run /fab-continue to generate it first.`

### Step 2: Stage-Scoped Taxonomy Scan

Perform a systematic scan of the artifact for gaps, ambiguities, and `[NEEDS CLARIFICATION]` markers. The scan categories vary by stage — there is no fixed universal list.

#### Brief categories

- Scope boundaries — are boundaries concrete or vague?
- Affected areas — are all impacted components identified?
- Blocking questions — any unresolved `[BLOCKING]` items?
- Impact completeness — does the Impact section cover all areas?
- Affected docs coverage — are all relevant docs listed under Affected Docs?

#### Spec categories

- Requirement precision — do requirements use RFC 2119 keywords with specific expected behaviors?
- Scenario coverage — does every requirement have at least one GIVEN/WHEN/THEN scenario?
- Edge cases — are error states, boundary conditions, and exceptional paths covered?
- Deprecated requirements — if behavior is removed, is it captured?
- Cross-references — do references to centralized docs match reality?

#### Tasks categories

- Task completeness — does every file and feature from the spec have a task?
- Granularity — is each task completable in one focused session?
- Dependency ordering — are non-obvious dependencies in the Execution Order section?
- File path accuracy — does each task reference exact file paths?
- Parallel markers — are independent tasks marked `[P]`?

Also scan for:
- `<!-- assumed: {description} -->` markers (left by any planning skill) — these are Tentative assumptions to confirm or override

When presenting a question derived from an `<!-- assumed: ... -->` marker, frame the current assumption as the **recommended option** and offer alternatives. For example, if the marker says `<!-- assumed: supplement existing auth rather than replace -->`, the recommendation should be "Supplement existing auth" with alternatives like "Replace existing auth" or "Both — configurable".

After scanning, build a **prioritized question queue** (highest-impact gaps first). Cap at **5 questions maximum** per invocation.

**If the scan finds zero gaps**, output:

```
Stage: {stage} (active). Reviewing {artifact} for gaps...

No gaps found — artifact looks solid. Ready to proceed.

Next: /fab-clarify (refine further) or /fab-continue or /fab-ff
```

And stop (skip Steps 3-6).

### Step 3: Present Questions One at a Time

Present only the **first question** from the queue. Do not reveal queued questions.

Each question MUST include:

**For multiple-choice questions** (ambiguity with discrete resolution options):

```
**Question {N} of {total}**: {question text}

Recommendation: {recommended option} — {reasoning}

| # | Option | Description |
|---|--------|-------------|
| 1 | {option 1} | {description} |
| 2 | {option 2} | {description} |
| 3 | {option 3} | {description} |

Reply with a number, "yes"/"recommended" to accept, or your own answer.
```

**For short-answer questions** (free-form input needed):

```
**Question {N} of {total}**: {question text}

Suggested answer: {suggested answer} — {reasoning}

Reply with "yes"/"recommended" to accept, or provide your own answer.
```

### Step 4: Process Answer and Update Artifact

After the user responds:

1. Interpret the answer:
   - `"yes"`, `"recommended"`, `"y"` (case-insensitive) → accept the recommendation
   - A number → select that option from the table
   - Free text → use as the custom answer
   - `"done"`, `"good"`, `"no more"` (case-insensitive) → **early termination** (skip to Step 6)
2. **Immediately update the artifact in place** to reflect the resolution:
   - Replace `[NEEDS CLARIFICATION]` markers with concrete content
   - Replace `<!-- assumed: ... -->` markers with confirmed content (if user accepts) or updated content (if user overrides)
   - Add `<!-- clarified: {description} -->` HTML comment next to significant changes
3. Present the next question (return to Step 3)
4. After the 5th answer (or when the queue is exhausted), proceed to Step 5

### Step 5: Append Audit Trail

After all questions are answered (or on early termination), append an audit trail to the artifact:

```markdown
## Clarifications

### Session {YYYY-MM-DD}

- **Q**: {question text}
  **A**: {answer or "accepted recommendation: {description}"}
- **Q**: {question text}
  **A**: {answer}
```

**Rules:**
- If a `## Clarifications` section already exists, append a new `### Session {date}` subsection (do not replace previous sessions)
- If no `## Clarifications` section exists, create it at the end of the artifact
- Only include questions that were actually answered (not deferred/skipped)
- If 0 questions were answered (early termination on first question), skip the audit trail entirely — there is nothing to record

### Step 6: Display Coverage Summary

Display a summary table:

```
Clarification complete.

| Category | Count |
|----------|-------|
| Resolved | {N} — gaps addressed in this session |
| Clear | {N} — categories scanned with no gaps found |
| Deferred | {N} — gaps the user chose not to address (early termination) |
| Outstanding | {N} — gaps beyond the 5-question cap, awaiting next invocation |

{N} issues resolved. {M} items remain for further refinement.

Next: /fab-clarify (refine further) or /fab-continue or /fab-ff
```

### Step 7: Recompute Confidence Score

After resolving questions, recompute the confidence score:

1. Re-count SRAD grades across **all** artifacts in the change (brief, spec, tasks — whichever exist)
2. Apply the confidence formula (see `_context.md` Confidence Scoring section)
3. Write the updated `confidence` block to `.status.yaml`

This ensures the score reflects any resolved Tentative or Unresolved assumptions from this session.

### Step 8: Do NOT Advance Stage

**Critical**: Do NOT update the `stage` field in `.status.yaml`. Do NOT set any progress field to `done` that wasn't already `done`. The clarify skill is strictly non-advancing.

The only `.status.yaml` updates allowed are `confidence` (recomputed) and `last_updated` (to the current ISO 8601 timestamp).

---

## Auto Mode (Internal fab-ff Call)

When called internally by `fab-ff` between stage generations, the skill operates autonomously. No user interaction.

### Step 1: Identify and Read Artifact

Same as Suggest Mode Step 1 — determine the artifact file from the current stage and read it.

### Step 2: Autonomous Gap Analysis

Perform the same stage-scoped taxonomy scan as Suggest Mode Step 2, including scanning for `<!-- assumed: ... -->` markers. For each gap found, attempt autonomous resolution:

1. **Resolvable** — the gap can be resolved using available context (config, constitution, centralized docs, completed artifacts). Resolve it in place with a `<!-- clarified: {description} -->` marker. For `<!-- assumed: ... -->` markers that can be confirmed from context, remove the marker (assumption confirmed).
2. **Blocking** — the gap cannot be resolved from available context. It requires user input or external information that the agent does not have. Leave the gap in place with a `<!-- blocking: {description} -->` marker.
3. **Non-blocking** — a minor gap that does not materially affect downstream artifacts. Leave as-is with no marker.

### Step 3: Return Machine-Readable Result

Auto mode returns a structured result (not displayed to user — consumed by `fab-ff`):

```
{resolved: N, blocking: N, non_blocking: N}
```

Where:
- `resolved` — gaps the agent resolved autonomously
- `blocking` — gaps requiring user input (cannot proceed safely)
- `non_blocking` — minor gaps left as-is

If `blocking > 0`, include a description of each blocking issue:

```
{resolved: 2, blocking: 1, non_blocking: 0, blocking_issues: ["description of blocking issue"]}
```

### Step 4: Do NOT Advance Stage

Same as Suggest Mode — auto mode is non-advancing. Only update `last_updated` in `.status.yaml`.

---

## Output Examples

### Suggest Mode — Gaps Found

```
Stage: spec (active). Reviewing spec.md for gaps...

Found 3 gaps across 5 categories scanned.

**Question 1 of 3**: The spec mentions "handle authentication errors" but doesn't specify the response format. What should error responses look like?

Recommendation: JSON error body with `{error, message, code}` fields — consistent with existing API patterns in the codebase.

| # | Option | Description |
|---|--------|-------------|
| 1 | JSON error body | `{error: "auth_failed", message: "...", code: 401}` |
| 2 | Plain text | HTTP status code with text body |
| 3 | RFC 7807 Problem Details | Standard problem+json format |

Reply with a number, "yes"/"recommended" to accept, or your own answer.
```

### Suggest Mode — No Gaps

```
Stage: spec (active). Reviewing spec.md for gaps...

No gaps found — artifact looks solid. Ready to proceed.

Next: /fab-clarify (refine further) or /fab-continue or /fab-ff
```

### Suggest Mode — Early Termination

```
**Question 2 of 4**: ...

> done

Clarification complete.

| Category | Count |
|----------|-------|
| Resolved | 1 — gaps addressed in this session |
| Clear | 3 — categories scanned with no gaps found |
| Deferred | 3 — gaps the user chose not to address (early termination) |
| Outstanding | 0 — gaps beyond the 5-question cap, awaiting next invocation |

1 issue resolved. 3 items remain for further refinement.

Next: /fab-clarify (refine further) or /fab-continue or /fab-ff
```

### Suggest Mode — Coverage Summary After Full Session

```
Clarification complete.

| Category | Count |
|----------|-------|
| Resolved | 4 — gaps addressed in this session |
| Clear | 2 — categories scanned with no gaps found |
| Deferred | 0 — gaps the user chose not to address (early termination) |
| Outstanding | 0 — gaps beyond the 5-question cap, awaiting next invocation |

4 issues resolved. 0 items remain for further refinement.

Next: /fab-clarify (refine further) or /fab-continue or /fab-ff
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight script exits non-zero | Abort with the stderr message from `fab-preflight.sh` |
| Stage is `apply`, `review`, or `archive` | Abort with: "Stage is {stage} — clarify applies to planning artifacts only. Use /fab-review instead." |
| Artifact file missing for current stage | Abort with: "No {artifact} found. Run /fab-continue to generate it first." |
| Taxonomy scan finds zero gaps (suggest mode) | Output "No gaps found" message and stop |
| Early termination after 0 answered questions | Display coverage summary with 0 resolved, all deferred |
| Multiple `/fab-clarify` sessions | Audit trail entries accumulate — new session appended, previous sessions preserved |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — stage field in `.status.yaml` is never changed |
| Idempotent? | **Yes** — safe to call multiple times; each call refines further |
| Modifies artifact? | **Yes** — edits existing file in place |
| Creates new files? | **No** — only modifies the current stage artifact |
| Updates `.status.yaml`? | **Only** `confidence` (recomputed) and `last_updated` timestamp |
| Modes | **Suggest** (user invocation) and **Auto** (internal fab-ff call) |

---

## Next Steps Reference

After `/fab-clarify` completes:

`Next: /fab-clarify (refine further) or /fab-continue or /fab-ff`
