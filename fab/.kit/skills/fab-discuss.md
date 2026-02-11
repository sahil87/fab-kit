---
name: fab-discuss
description: "Discuss an idea or refine an existing proposal through conversation. Drives toward a solid, low-ambiguity proposal ready for /fab-fff."
---

# /fab-discuss [description]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Develop a proposal through free-form conversation. Unlike `/fab-new` (one-shot capture with max 3 questions), `/fab-discuss` is a back-and-forth exploration — it helps you figure out if a change is even needed, walks you through clarifying questions, and outputs a solid `proposal.md` with a high confidence score.

Two modes:
- **New change** (from scratch): gap analysis → clarifying questions → solid proposal
- **Refine existing** (on active change): load current proposal → discuss → improve → drive confidence up

Key difference from `/fab-new`: only offers to switch the active change when `fab/current` is empty (preserves current work context when another change is active). Key difference from `/fab-clarify`: not bounded to a fixed question cap, operates at the idea level rather than artifact gaps, and includes gap analysis.

---

## Pre-flight Check

Before doing anything else:

1. Check that `fab/config.yaml` exists and is readable
2. Check that `fab/constitution.md` exists and is readable

**If either check fails, STOP immediately.** Output this message and do nothing else:

> `fab/ is not initialized. Run /fab-init first to bootstrap the project.`

---

## Arguments

- **`[description]`** *(optional)* — a natural language description of what the user wants to discuss. Can be a vague idea ("better error handling"), a concrete feature ("add OAuth2 support"), or omitted entirely.

If no description is provided, ask the user: *"What would you like to discuss?"*

---

## Behavior

### Step 1: Determine Mode

Check whether `fab/current` exists and points to a valid change:

**If `fab/current` exists and points to a valid change:**

1. Read the active change's `proposal.md` (if it exists) and `.status.yaml`
2. Compare the user's description (if provided) against the active change's scope
3. **If the description is related** to the active change (or no description was provided): enter **Refine mode** — work on the active change's proposal
4. **If the description is significantly different** from the active change's scope: ask the user:
   > "You're currently working on {change-name} ({brief summary}). Is this about that change, or a new idea?"
   - If about the current change → **Refine mode**
   - If a new idea → **New change mode**

**If `fab/current` does not exist or is empty:**

Enter **New change mode** — start from scratch.

### Step 2: Load Context

Read these files for grounding (regardless of mode):
- `fab/config.yaml` — project name, tech stack, conventions
- `fab/constitution.md` — project principles and constraints
- `fab/docs/index.md` — understand the existing documentation landscape
- `fab/specs/index.md` — understand existing specs

**For Refine mode**, also load:
- `fab/changes/{name}/proposal.md` — the proposal to refine
- `fab/changes/{name}/.status.yaml` — current confidence score and stage

### Step 3: Gap Analysis (New Change Mode Only)

Before committing to a proposal, evaluate whether the change is needed:

1. **Check for existing mechanisms**: Does the current workflow, codebase, or docs already address what the user describes? Search centralized docs (`fab/docs/`) and skill definitions for relevant functionality.
2. **Evaluate scope**: Is the idea too broad (should be split)? Too narrow (could be part of something larger)?
3. **Consider alternatives**: Are there simpler approaches? Could an existing skill be extended rather than creating a new one?

Present your findings conversationally:

- If an existing mechanism covers the idea: explain what exists, ask if they still want to proceed or if the existing solution is sufficient
- If a gap is confirmed: acknowledge it and proceed to proposal development
- If the scope seems off: suggest adjustments and discuss

**This phase is conversational, not a checklist.** The goal is to help the user think through whether and how to proceed.

**Skip this step in Refine mode** — the change is already established; the user wants to improve it, not re-evaluate its existence.

### Step 4: Conversational Proposal Development

Develop the proposal through back-and-forth conversation. There is **no fixed question cap** — ask as many clarifying questions as needed to build a solid proposal.

**For each discussion round:**

1. Identify the most important unresolved aspect of the proposal
2. Ask a focused question about it, providing your recommendation and reasoning
3. Incorporate the user's answer into your evolving mental model of the proposal
4. Track SRAD grades for each decision point as you go:
   - Decisions the user explicitly answers → **Certain** (user signal is definitive)
   - Decisions clearly implied by the user's answers → **Confident**
   - Decisions you're making reasonable guesses about → **Tentative**
   - Decisions you haven't been able to resolve → ask about them next

**Conversation flow guidelines:**
- Start with the highest-impact decisions (lowest Reversibility + lowest Agent Competence)
- Build on previous answers — don't ask questions the user already answered
- Group related questions when natural ("While we're on the topic of X...")
- Offer recommendations with reasoning, not just open-ended questions
- After resolving a major ambiguity, briefly summarize your updated understanding

**For Refine mode:** Start by reviewing the existing proposal and identifying its weakest areas (Tentative assumptions, vague sections, gaps). Present these to the user and discuss.

### Step 5: Monitor Confidence Score

Throughout the conversation, maintain a running count of SRAD grades:

- **After each answer**, mentally update the counts (certain, confident, tentative, unresolved)
- **When the computed score crosses 3.0** (the `/fab-fff` gate threshold), proactively inform the user:

> "Confidence is now {score}/5.0 — high enough for `/fab-fff`. Want to finalize the proposal, or keep refining?"

- **If the user wants to continue**, keep going — there's no upper limit
- **If the user wants to finalize**, proceed to Step 6

The user may also end the discussion early at any time (e.g., "done", "looks good", "that's enough") regardless of the current score.

### Step 6: Generate or Update Proposal

**New change mode:**

1. Generate a folder name using the format `{YYMMDD}-{XXXX}-{slug}` (same rules as `/fab-new`)
2. Create `fab/changes/{name}/` and `fab/changes/{name}/checklists/`
3. **Do NOT write to `fab/current` yet** — activation is offered after the summary (see Step 8)
4. **Do NOT create or adopt a git branch yet** — branch integration is handled via `/fab-switch` if the user accepts activation
5. Initialize `fab/changes/{name}/.status.yaml`:

```yaml
name: {name}
created: {ISO 8601 timestamp}
created_by: {git config user.name, or "unknown" if unset}
stage: proposal
progress:
  proposal: done
  specs: pending
  plan: pending
  tasks: pending
  apply: pending
  review: pending
  archive: pending
checklist:
  generated: false
  path: checklists/quality.md
  completed: 0
  total: 0
confidence:
  certain: {count}
  confident: {count}
  tentative: {count}
  unresolved: 0
  score: {computed score}
last_updated: {ISO 8601 timestamp}
```

6. Generate `proposal.md` from the template at `fab/.kit/templates/proposal.md`, incorporating all discussion outcomes:
   - Fill in all sections: Why, What Changes, Affected Docs, Impact, Open Questions
   - Mark any remaining Tentative decisions with `<!-- assumed: {description} -->`
   - Append the `## Assumptions` section
7. Set `progress.proposal` to `done`

**Refine mode:**

1. Update the existing `proposal.md` in place — incorporate discussion outcomes, resolve `<!-- assumed: ... -->` markers, update sections
2. Recompute the confidence score
3. Update `.status.yaml`: confidence block + `last_updated`
4. Do NOT change the `stage` or any `progress` fields beyond what's already set

### Step 7: Compute Confidence Score

Count SRAD grades across the final proposal:

1. **Certain**: decisions explicitly answered by the user or deterministically answered by config/constitution
2. **Confident**: decisions with strong signal and one obvious interpretation
3. **Tentative**: decisions marked with `<!-- assumed: ... -->` in the artifact
4. **Unresolved**: should be 0 after a thorough discussion (if any remain, they become Tentative with a best guess)

Apply the confidence formula: `max(0.0, 5.0 - 0.1 * confident - 1.0 * tentative)` if `unresolved == 0`, else `0.0`.

Write the `confidence` block to `.status.yaml`.

### Step 8: Display Summary

After finalizing, display:

- The confidence score and what it means for the pipeline
- If score >= 3.0: note that `/fab-fff` is available for full autonomous pipeline
- If score < 3.0: note the score and suggest `/fab-clarify` or another `/fab-discuss` session to raise it

### Step 9: Offer Activation (New Change Mode Only)

**This step only applies in new change mode.** Skip entirely in refine mode.

After displaying the summary, check whether `fab/current` exists and is non-empty:

**If `fab/current` does not exist or is empty:**

Prompt the user:

> "{name} — no active change. Set as active?"

- **If the user accepts**: Call `/fab-switch` internally — write `fab/current` with the new change name and handle git branch integration (same pattern as `/fab-new`). Then show the "Next:" line without the `/fab-switch` step:
  > `Next: /fab-continue or /fab-ff (fast-forward all planning)`

- **If the user declines**: Leave `fab/current` untouched. Show the standard "Next:" line:
  > `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff`

**If `fab/current` exists and points to a valid (different) change:**

Do NOT prompt for activation — the user's current work context is preserved. Show the standard "Next:" line:
> `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff`

**If `/fab-switch` fails** (e.g., git branch creation error):

The proposal is already saved. Report the error and suggest manual activation:
> "Activation failed: {error}. Run `/fab-switch {name}` manually."

---

## Output

### New Change Mode (Activated)

```
Created fab/changes/260208-x7k2-better-errors/

## Proposal: Better Error Handling

{filled proposal content}

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | JSON error format | Config shows REST API stack |

1 assumption made (1 confident, 0 tentative).

Confidence: 4.8/5.0 — ready for /fab-fff.

260208-x7k2-better-errors — no active change. Set as active?

{user accepts}

Active change set to 260208-x7k2-better-errors on branch 260208-x7k2-better-errors.

Next: /fab-continue or /fab-ff (fast-forward all planning)
```

### New Change Mode (Not Activated)

```
Created fab/changes/260208-x7k2-better-errors/

## Proposal: Better Error Handling

{filled proposal content}

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | JSON error format | Config shows REST API stack |

1 assumption made (1 confident, 0 tentative).

Confidence: 4.8/5.0 — ready for /fab-fff.

Next: /fab-switch 260208-x7k2-better-errors to make it active, then /fab-continue or /fab-ff
```

### Refine Mode

```
## Proposal: Add OAuth2 Support (Updated)

{updated proposal content}

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Google + GitHub providers | User confirmed during discussion |

1 assumption made (1 confident, 0 tentative).

Confidence: 3.8 → 4.5/5.0 — improved from discussion.

Next: /fab-continue or /fab-ff (fast-forward all planning)
```

### Early Termination (Score Below Threshold)

```
Created fab/changes/260208-x7k2-payment-flow/ (not set as active)

## Proposal: Payment Flow Redesign (Draft)

{partially filled proposal content}

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Stripe over Braintree | User preference |
| 2 | Tentative | Webhook-based reconciliation | Most common pattern, not confirmed |
| 3 | Tentative | Single-currency initially | Scope reduction guess |

3 assumptions made (1 confident, 2 tentative).

Confidence: 2.9/5.0 — below the /fab-fff threshold (3.0). Run /fab-discuss or /fab-clarify to resolve tentative assumptions.

Next: /fab-switch 260208-x7k2-payment-flow to make it active, then /fab-clarify or /fab-discuss
```

### Gap Analysis Finds Existing Mechanism

```
Before we create a new change, let me check if this is already covered...

It looks like `/fab-continue <stage>` already provides the ability to reset to an earlier stage (see fab-workflow/planning-skills.md). The reset behavior includes invalidating downstream artifacts and regenerating from the target stage.

Does this cover what you need, or are you looking for something different?

{user decides to not proceed}

No change created. The existing mechanism covers this use case.
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing | Abort with: "fab/ is not initialized. Run /fab-init first to bootstrap the project." |
| `fab/constitution.md` missing | Abort with same message as above |
| No description provided | Ask: "What would you like to discuss?" |
| `fab/.kit/templates/proposal.md` missing | Abort with: "Proposal template not found at fab/.kit/templates/proposal.md — kit may be corrupted." |
| `fab/changes/{name}/` already exists (new change) | Regenerate the random component (`XXXX`) and retry |
| Active change's `proposal.md` missing (refine mode) | Note the absence; offer to create a proposal from scratch for this change or start a new change |
| Gap analysis finds the idea is already covered | Present findings; let the user decide whether to proceed |
| User ends discussion with zero decisions made | Do not create a change folder; output "No change created." |
| `/fab-switch` fails during activation | Proposal is already saved; report error and suggest manual `/fab-switch` |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **Yes** — sets `progress.proposal` to `done` when finalizing |
| Switches active change? | **Conditionally** — offers when `fab/current` is empty (calls `/fab-switch` internally) |
| Creates git branch? | **Conditionally** — via internal `/fab-switch` when activation is accepted |
| Idempotent? | **Yes** — safe to call multiple times; refine mode updates in place |
| Modifies artifact? | **Yes** — creates or updates `proposal.md` |
| Creates new files? | **Yes** (new change mode) — change folder, `.status.yaml`, `proposal.md` |
| Updates `.status.yaml`? | **Yes** — progress, confidence, and `last_updated` |
| Question cap? | **None** — conversational by design |

---

## Key Differences from Related Skills

| Aspect | fab-discuss | fab-new | fab-clarify |
|--------|-------------|---------|-------------|
| **Purpose** | Explore & develop proposal through conversation | Capture clear description as proposal | Refine existing artifact's gaps |
| **Input** | Vague idea or existing proposal | Clear change description | Existing artifact with gaps |
| **Gap analysis** | Yes — "is this change even needed?" | No — assumes the change is needed | No — assumes the artifact exists |
| **Interaction style** | Free-form conversation, unlimited questions | One-shot generation, max 3 SRAD questions | Structured Q&A, max 5 per session |
| **Sets active change** | Conditionally — offers when no active change | Yes | N/A (operates on active change) |
| **Creates change folder** | Yes (new change mode) | Yes | No |
| **Git integration** | Conditionally — via internal `/fab-switch` when activation accepted | Yes (branch create/adopt) | None |
| **Confidence goal** | Drive score high for `/fab-fff` | Compute initial score | Recompute after refinements |

---

## Next Steps Reference

After `/fab-discuss` completes:

- New change created (activated): `Next: /fab-continue or /fab-ff (fast-forward all planning)`
- New change created (not activated): `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff`
- Existing proposal refined: `Next: /fab-continue or /fab-ff (fast-forward all planning)`
- No change created (gap analysis): no Next line
