---
name: fab-ff
description: "Fast-forward through all remaining planning stages in one pass to reach implementation quickly."
---

# /fab-ff

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Fast-forward through all remaining planning stages in one pass. Generates spec and tasks (with quality checklist) — all in a single invocation. Interleaves auto-clarify between stage generations to catch and resolve gaps before they compound downstream.

Interleaves auto-clarify between stage generations; stops if blocking issues are found that the agent cannot resolve autonomously. Resumable — re-running after a bail picks up from the first incomplete stage.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence`

Then verify the stage-specific precondition using the preflight output:

4. Verify that `brief.md` exists in the change directory (`fab/changes/{name}/brief.md`)

**If `brief.md` does not exist, STOP.** Output:

> `Brief not found. Run /fab-new to create the brief first, then run /fab-ff.`

---

## Context Loading

Load all context upfront since fast-forward traverses all planning stages:

1. `fab/config.yaml` — project config, tech stack
2. `fab/constitution.md` — project principles and constraints
3. `fab/changes/{name}/brief.md` — the completed brief
4. `fab/docs/index.md` — documentation landscape
5. Specific centralized docs referenced by the brief's **Affected Docs** section (read each `fab/docs/{domain}/{doc}.md` listed under New, Modified, or Removed)

---

## Behavior

### Resumability

On invocation, check the `progress` map from preflight output. **Skip stages already marked `done`**. This means:

- If `progress.spec` is already `done`, skip Step 1 (questions) and Step 2 (spec generation) and their auto-clarify
- If `progress.tasks` is already `done`, skip Step 3 (task generation) and its auto-clarify

This makes `/fab-ff` resumable after a bail — re-running picks up from the first incomplete stage.

### Step 1: Frontload All Questions

Find the `active` entry in the progress map and start from there, skipping stages already `done`. The pipeline covers 2 planning stages: `spec` and `tasks`.

Apply SRAD scoring across the brief for ambiguities spanning **all** planning stages (spec, tasks). Consider:

- **Spec ambiguities**: Are any requirements vague? Multiple interpretations? Missing acceptance criteria? Edge cases unaddressed?
- **Task ambiguities**: Is the scope unclear enough that task breakdown would require guessing?

Collect all **Unresolved** decisions (SRAD grade) into a **single batch of questions**. Ask once, then proceed without further interruption. Confident and Tentative decisions are assumed (tracked for the cumulative Assumptions summary).

**If there are questions:**

Present them all in a single numbered list. Wait for the user to answer all of them. Then proceed to Step 2 without asking anything else.

**If SRAD evaluation finds no Unresolved decisions:**

Skip questions entirely and proceed directly to Step 2.

**The goal: at most one Q&A round, then heads-down generation.**

### Step 2: Generate `spec.md`

*(Skip if `progress.spec` is already `done`.)*

Follow the **Spec Generation Procedure** defined in `fab/.kit/skills/_generation.md`.

Additionally: incorporate answers from Step 1 to resolve any ambiguities — the spec should have no `[NEEDS CLARIFICATION]` markers (unlike `/fab-continue`, which may leave markers for `/fab-clarify`).

Update `.status.yaml`:
- Set `progress.spec` to `done`
- Update `last_updated`

#### Auto-Clarify: Spec

Run auto-clarify on the generated spec: invoke `/fab-clarify` with the `[AUTO-MODE]` prefix (per the Skill Invocation Protocol in `_context.md`), with stage context set to `spec`. Interpret the result:

- **`blocking: 0`** → continue to Step 3
- **`blocking > 0`** → **BAIL**. Stop the pipeline, report blocking issues, and output:
  > `Auto-clarify found {N} blocking issue(s) in spec.md that cannot be resolved autonomously:`
  > `- {description of each blocking issue}`
  >
  > `Run /fab-clarify to resolve these interactively, then /fab-ff to resume.`
  >
  > Leave `.status.yaml` with `spec: done`, `tasks: pending`.

### Step 3: Generate `tasks.md`

*(Skip if `progress.tasks` is already `done`.)*

Follow the **Tasks Generation Procedure** defined in `fab/.kit/skills/_generation.md`.

#### Auto-Clarify: Tasks

Run auto-clarify on the generated tasks: invoke `/fab-clarify` with the `[AUTO-MODE]` prefix (per the Skill Invocation Protocol in `_context.md`), with stage context set to `tasks`. Interpret the result using the same bail/guess logic as above.

### Step 4: Auto-generate Quality Checklist

Follow the **Checklist Generation Procedure** defined in `fab/.kit/skills/_generation.md`.

### Step 5: Update `.status.yaml`

After all artifacts are generated:

1. Set `progress.tasks` to `done`
2. Set `progress.apply` to `active` (two-write transition)
3. Set `checklist.generated` to `true`
4. Set `checklist.total` to the number of checklist items generated
5. Set `checklist.completed` to `0`
6. Update `last_updated` to the current ISO 8601 timestamp

---

## Output

### Default Mode — Clean Fast-Forward (no questions, no blockers)

```
Fast-forwarding from brief...

## Spec: {Change Name}

{spec content}

Spec created.
Auto-clarify: spec — {resolved: N, blocking: 0, non_blocking: N}

## Tasks: {Change Name}

{tasks content}

Auto-clarify: tasks — {resolved: N, blocking: 0, non_blocking: N}

## Quality Checklist

Generated checklists/quality.md with {N} items.

Fast-forward complete — spec, tasks, and checklist generated.

## Assumptions (cumulative)

| # | Grade | Decision | Rationale | Artifact |
|---|-------|----------|-----------|----------|
| 1 | Confident | {decision} | {rationale} | spec.md |
| 2 | Tentative | {decision} | {rationale} | spec.md |
| 3 | Tentative | {decision} | {rationale} | tasks.md |

{N} assumptions made ({C} confident, {T} tentative). Run /fab-clarify to review.

Next: /fab-apply
```

### Default Mode — Bail on Blocking Issue

```
Fast-forwarding from brief...

## Spec: {Change Name}

{spec content}

Spec created.
Auto-clarify: spec — {resolved: 2, blocking: 1, non_blocking: 0}

⚠ Auto-clarify found 1 blocking issue in spec.md:
- The spec references "external auth provider" but available context doesn't specify which provider to use.

Run /fab-clarify to resolve this interactively, then /fab-ff to resume.
```

### Default Mode — Resume After Bail

```
Fast-forwarding from spec (resuming)...

Skipping spec — already done.

## Tasks: {Change Name}

{tasks content}

Auto-clarify: tasks — {resolved: 1, blocking: 0, non_blocking: 0}

## Quality Checklist

Generated checklists/quality.md with {N} items.

Fast-forward complete — spec, tasks, and checklist generated.

Next: /fab-apply
```

### Ambiguous Brief (questions first, then pipeline)

```
Fast-forwarding from brief...

Before I can generate all planning artifacts, I need to resolve a few ambiguities:

1. {question about spec scope}
2. {question about technical approach}
3. {question about edge case}

{user answers}

## Spec: {Change Name}

{spec content incorporating answers}

Spec created.
Auto-clarify: spec — {resolved: N, blocking: 0, non_blocking: N}

{... remainder of pipeline ...}

Fast-forward complete — spec, tasks, and checklist generated.

Next: /fab-apply
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight script exits non-zero | Abort with the stderr message from `fab-preflight.sh` |
| `brief.md` does not exist | Abort with: "Brief not found. Run /fab-new to create the brief first, then run /fab-ff." |
| Template file missing | Abort with: "Template not found at fab/.kit/templates/{file} — kit may be corrupted." |
| Spec already done (stage is `spec` or later) | Resume from current position — skip completed stages |
| Auto-clarify returns blocking issues | Bail — stop pipeline, report issues, suggest `/fab-clarify` then `/fab-ff` |

---

## Key Difference from `/fab-continue` and `/fab-fff`

| Behavior | `/fab-continue` | `/fab-ff` | `/fab-fff` |
|----------|-----------------|-----------|-----------|
| Questions | Asked per-stage as needed | Frontloaded: one batch upfront | Same as fab-ff (frontloaded) |
| Auto-clarify | None (manual `/fab-clarify`) | Between each stage; bails on blockers | Same as fab-ff |
| Stages per invocation | One planning stage | All planning stages (may bail mid-way) | Full pipeline: planning + apply + review + archive |
| Resumable? | N/A (one stage) | Yes — re-invoke after bail | Yes — skips completed stages |
| Confidence gate | None | None | Requires score >= 3.0 |
| Best for | Deliberate, step-by-step planning | Fast planning with quality gates | High-confidence changes, full autonomy |

---

## Next Steps Reference

After `/fab-ff` completes:

`Next: /fab-apply`
