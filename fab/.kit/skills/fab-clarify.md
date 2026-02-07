---
name: fab-clarify
description: "Refine the current stage artifact — resolve gaps, ambiguities, or [NEEDS CLARIFICATION] markers without advancing."
---

# /fab:clarify

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Deepen and refine the current stage artifact without advancing to the next stage. Use this skill when the current artifact has gaps, ambiguities, or `[NEEDS CLARIFICATION]` markers that should be resolved before moving forward. Safe to call multiple times — each invocation refines further.

---

## Pre-flight Check

Before doing anything else:

1. Check that `fab/current` exists and is readable
2. Read the change name from `fab/current`
3. Verify `fab/changes/{name}/` directory exists
4. Read `fab/changes/{name}/.status.yaml`

**If `fab/current` does not exist, STOP immediately.** Output:

> `No active change. Run /fab:new <description> to start one.`

**If the change directory or `.status.yaml` is missing, STOP.** Output:

> `Active change "{name}" is corrupted — .status.yaml not found. Run /fab:new to start a fresh change.`

**If `fab/config.yaml` or `fab/constitution.md` is missing, STOP.** Output:

> `fab/ is not initialized. Run /fab:init first.`

---

## Stage Guard

Read the `stage` field from `.status.yaml`. The current stage must be one of:

- `proposal`
- `specs`
- `plan`
- `tasks`

**If the stage is `apply`, `review`, or `archive`, STOP.** Output:

> `Stage is "{stage}" — clarify applies to planning artifacts only (proposal, specs, plan, tasks). Use /fab:review to validate implementation instead.`

---

## Context Loading

Context varies by the current stage. Load only what is relevant:

### Proposal stage

- `fab/config.yaml` — project config, tech stack
- `fab/constitution.md` — project principles and constraints
- `fab/changes/{name}/proposal.md` — the artifact to refine

### Specs stage

- Everything from proposal context above, plus:
- `fab/changes/{name}/proposal.md` — the completed proposal (for reference)
- `fab/changes/{name}/spec.md` — the artifact to refine
- `fab/docs/index.md` — documentation landscape
- Specific centralized docs referenced by the proposal's **Affected Docs** section

### Plan stage

- Everything from specs context above, plus:
- `fab/changes/{name}/spec.md` — the completed spec (for reference)
- `fab/changes/{name}/plan.md` — the artifact to refine

### Tasks stage

- Everything from plan context above, plus:
- `fab/changes/{name}/plan.md` — the completed plan (for reference, if it exists; skip if plan was `skipped`)
- `fab/changes/{name}/tasks.md` — the artifact to refine

---

## Behavior

### Step 1: Identify the Current Artifact

Based on the current stage from `.status.yaml`, determine which artifact file to refine:

| Stage | Artifact file |
|-------|--------------|
| `proposal` | `proposal.md` |
| `specs` | `spec.md` |
| `plan` | `plan.md` |
| `tasks` | `tasks.md` |

Read the artifact file. If it does not exist, STOP with:

> `No {artifact} found for this stage. Run /fab:continue to generate it first.`

### Step 2: Analyze for Gaps

Examine the artifact for gaps, ambiguities, and opportunities to deepen. The analysis focus varies by stage:

#### Proposal

- **Unresolved `[BLOCKING]` questions** — these prevent moving to specs. Attempt to resolve each one by researching context (config, constitution, existing docs) or asking the user for clarification.
- **Vague scope** — look for overly broad or undefined boundaries. Sharpen the "What Changes" section with concrete boundaries.
- **Missing impact analysis** — verify the "Impact" section covers all affected areas. Cross-reference against `fab/docs/index.md` to ensure no domains are overlooked.
- **Incomplete affected docs** — verify the "Affected Docs" section lists all docs that will be created, modified, or removed.

#### Specs

- **`[NEEDS CLARIFICATION]` markers** — resolve each one by researching centralized docs, asking the user, or making a well-reasoned decision (document the reasoning inline).
- **Missing scenarios** — for each requirement, verify there is at least one GIVEN/WHEN/THEN scenario. Add scenarios for edge cases, error states, and boundary conditions.
- **Underspecified requirements** — look for requirements that use vague language (e.g., "should handle errors appropriately") and make them concrete with specific expected behaviors.
- **Missing deprecated requirements** — if the change removes or replaces existing behavior, verify the Deprecated Requirements section captures what is being removed.

#### Plan

- **Untested assumptions** — identify assumptions about the codebase, libraries, or APIs that haven't been verified. Add research notes or flag for investigation.
- **Missing research** — if the plan references libraries, APIs, or patterns, verify they exist and are suitable. Add findings to the Research section.
- **Weak decision rationale** — for each decision in the Decisions section, ensure the "Why" is compelling and at least one "Rejected alternative" is listed.
- **Incomplete file changes** — cross-reference the File Changes section against the spec requirements to ensure all necessary files are listed.
- **Missing risks** — identify risks not captured in the Risks / Trade-offs section.

#### Tasks

- **Missing tasks** — cross-reference tasks against the plan's File Changes (or spec requirements if plan was skipped) to ensure every file and feature is covered.
- **Wrong granularity** — tasks should be completable in one focused session. Split tasks that are too large; merge tasks that are trivially small.
- **Unclear dependencies** — verify the Execution Order section captures non-obvious dependencies between tasks.
- **Missing file paths** — each task should reference exact file paths. Add paths where missing.
- **Missing parallel markers** — identify tasks that can run in parallel and add `[P]` markers.

### Step 3: Refine In Place

Edit the existing artifact file directly. Do NOT regenerate from scratch or create a new file.

**Rules for in-place refinement:**
- Preserve the overall structure and sections of the artifact
- Add new content (scenarios, requirements, tasks) inline where they belong
- Resolve markers (`[BLOCKING]`, `[NEEDS CLARIFICATION]`) by replacing them with concrete content
- If asking the user a question to resolve an ambiguity, present the question, wait for the answer, then edit the artifact with the resolution
- Add a brief `<!-- clarified: {description} -->` HTML comment next to each significant change so subsequent `/fab:clarify` calls can see what was already refined

### Step 4: Report Changes

After refining, output a summary of what was changed:

```
Stage: {stage} (active). Reviewing {artifact} for gaps...

Changes made:
- {description of change 1}
- {description of change 2}
- {description of change 3}

{N} issues resolved. {M} items remain for further refinement.
```

If no gaps were found:

```
Stage: {stage} (active). Reviewing {artifact} for gaps...

No gaps found — artifact looks solid. Ready to proceed.
```

### Step 5: Do NOT Advance Stage

**Critical**: Do NOT update the `stage` field in `.status.yaml`. Do NOT set any progress field to `done` that wasn't already `done`. The clarify skill is strictly non-advancing — it only deepens the current artifact.

The only `.status.yaml` update allowed is `last_updated` (to the current ISO 8601 timestamp).

---

## Output

### Gaps Found and Resolved

```
Stage: specs (active). Reviewing spec.md for gaps...

Changes made:
- Resolved [NEEDS CLARIFICATION] on authentication timeout (set to 30 minutes based on constitution security policy)
- Added 3 missing GIVEN/WHEN/THEN scenarios for error states
- Sharpened requirement R-AUTH-003 from "handle errors" to "return 401 with JSON error body"

3 issues resolved. 0 items remain for further refinement.

Next: /fab:clarify (refine further) or /fab:continue or /fab:ff
```

### Gaps Found, User Input Needed

```
Stage: proposal (active). Reviewing proposal.md for gaps...

Found 1 [BLOCKING] question that needs your input:

1. The proposal mentions "support for external providers" — which providers specifically? (Google, GitHub, SAML, all of the above?)

{user answers}

Changes made:
- Resolved [BLOCKING]: scoped to Google and GitHub OAuth2 only
- Sharpened scope in "What Changes" to list specific endpoints
- Added "Affected Docs: auth/authentication.md (Modified)"

1 issue resolved. 0 items remain for further refinement.

Next: /fab:clarify (refine further) or /fab:continue or /fab:ff
```

### No Gaps Found

```
Stage: plan (active). Reviewing plan.md for gaps...

No gaps found — artifact looks solid. Ready to proceed.

Next: /fab:clarify (refine further) or /fab:continue or /fab:ff
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/current` missing | Abort with: "No active change. Run /fab:new \<description\> to start one." |
| `.status.yaml` missing or corrupted | Abort with: "Active change is corrupted — .status.yaml not found." |
| Stage is `apply`, `review`, or `archive` | Abort with: "Stage is {stage} — use /fab:review instead." |
| Artifact file missing for current stage | Abort with: "No {artifact} found. Run /fab:continue to generate it first." |
| `fab/config.yaml` or `fab/constitution.md` missing | Abort with: "fab/ is not initialized. Run /fab:init first." |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — stage field in `.status.yaml` is never changed |
| Idempotent? | **Yes** — safe to call multiple times; each call refines further |
| Modifies artifact? | **Yes** — edits existing file in place |
| Creates new files? | **No** — only modifies the current stage artifact |
| Updates `.status.yaml`? | **Only** `last_updated` timestamp |

---

## Next Steps Reference

After `/fab:clarify` completes:

`Next: /fab:clarify (refine further) or /fab:continue or /fab:ff`
