---
name: fab-ff
description: "Fast-forward through the entire pipeline — planning, implementation, review, and hydrate — with interactive clarification stops."
---

# /fab-ff [<change-name>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Fast-forward through the entire Fab pipeline in a single invocation: planning (spec, tasks) → apply → review → hydrate. Interleaves auto-clarify between planning stage generations and stops for interactive resolution when blocking issues arise at any phase.

Unlike `/fab-fff`, which requires a confidence gate and bails immediately on review failure, `/fab-ff` has no confidence gate and presents interactive rework options when review fails. This makes `/fab-ff` the "fast but interactive" pipeline and `/fab-fff` the "fully autonomous" pipeline.

Resumable — re-running after a bail or failure picks up from the first incomplete stage.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one in `fab/current`. Supports full folder names, partial slug matches, or 4-char IDs (e.g., `r3m7`). When provided, passed to the preflight script as `$1` for transient resolution — `fab/current` is **not** modified.

If no argument is provided, the skill operates on the active change in `fab/current`.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh [change-name]` via Bash — pass the change-name argument if one was provided
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence`

Then verify the stage-specific precondition using the preflight output:

4. Verify that `brief.md` exists in the change directory (`fab/changes/{name}/brief.md`)

**If `brief.md` does not exist, STOP.** Output:

> `Brief not found. Run /fab-new to create the brief first, then run /fab-ff.`

---

## Context Loading

Load all context upfront since fast-forward traverses all stages:

1. `fab/config.yaml` — project config, tech stack
2. `fab/constitution.md` — project principles and constraints
3. `fab/changes/{name}/brief.md` — the completed brief
4. `fab/memory/index.md` — memory landscape
5. Specific memory files referenced by the brief's **Affected Memory** section (read each `fab/memory/{domain}/{file}.md` listed as new, modify, or remove)

---

## Behavior

### Resumability

On invocation, check the `progress` map from preflight output. **Skip stages already marked `done`**. This means:

- If `progress.spec` is already `done`, skip Step 1 (questions) and Step 2 (spec generation) and their auto-clarify
- If `progress.tasks` is already `done`, skip Step 3 (task generation) and its auto-clarify
- If `progress.apply` is already `done`, skip Step 6 (implementation)
- If `progress.review` is already `done`, skip Step 7 (review)
- If `progress.hydrate` is already `done`, skip Step 8 (hydrate) — pipeline is complete

This makes `/fab-ff` resumable after a bail, failure, or interruption — re-running picks up from the first incomplete stage.

### Step 1: Frontload All Questions

Find the `active` entry in the progress map and start from there, skipping stages already `done`. The planning pipeline covers 3 stages: `brief`, `spec`, and `tasks`. Fast-forward can start from any of these stages based on the current `active` stage.

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

### Step 5: Update `.status.yaml` (Planning Complete)

After all planning artifacts are generated:

1. Set `progress.tasks` to `done`
2. Set `progress.apply` to `active` (two-write transition)
3. Set `checklist.generated` to `true`
4. Set `checklist.total` to the number of checklist items generated
5. Set `checklist.completed` to `0`
6. Update `last_updated` to the current ISO 8601 timestamp

### Step 6: Implementation (fab-apply)

*(Skip if `progress.apply` is already `done`.)*

Execute apply behavior — parse unchecked tasks from `tasks.md`, execute in dependency order, run tests after each completed task, mark tasks `[x]` on completion, update `.status.yaml` progress after each task.

**If a task fails and cannot be resolved by the agent**, **STOP** the pipeline. Report which task failed and why, and output:

> `Task {ID} failed during apply: {reason}`
>
> `Investigate the failure and re-run /fab-ff to resume from here.`

On successful completion of all tasks, update `.status.yaml`:
- Set `progress.apply` to `done`
- Set `progress.review` to `active`
- Update `last_updated`

### Step 7: Review (fab-review)

*(Skip if `progress.review` is already `done`.)*

Execute review behavior — validate implementation against specs and checklists:

1. All tasks in `tasks.md` marked `[x]`
2. All checklist items in `checklist.md` verified and checked off
3. Run tests affected by the change
4. Features match spec requirements (spot-check key scenarios)
5. No memory drift detected

**If review passes**: Update `.status.yaml`:
- Set `progress.review` to `done`
- Set `progress.hydrate` to `active`
- Update `last_updated`
- Proceed to Step 8.

**If review fails**: Present the interactive rework menu:

- **Fix code** → identify affected tasks, uncheck them in `tasks.md` (with `<!-- rework: reason -->` comment), re-run apply behavior from Step 6
- **Revise tasks** → user edits `tasks.md`, re-run apply behavior from Step 6
- **Revise spec** → reset to spec stage via `/fab-continue spec`, then the user can re-run `/fab-ff` to resume

The user selects a rework option and the pipeline handles it accordingly. This interactive stop is the key behavioral difference from `/fab-fff`, which bails immediately.

### Step 8: Hydrate

*(Skip if `progress.hydrate` is already `done`.)*

Execute hydrate behavior:

1. Final validation — review must have passed
2. Concurrent change check — warn about other active changes modifying the same memory files
3. Hydrate into `fab/memory/` — integrate new/changed requirements from `spec.md`
4. Update `.status.yaml` to `hydrate: done`

---

## Output

### Clean Full Pipeline (no questions, no blockers)

```
Fast-forwarding from brief...

--- Planning ---

## Spec: {Change Name}

{spec content}

Spec created.
Auto-clarify: spec — {resolved: N, blocking: 0, non_blocking: N}

## Tasks: {Change Name}

{tasks content}

Auto-clarify: tasks — {resolved: N, blocking: 0, non_blocking: N}

## Quality Checklist

Generated checklist.md with {N} items.

## Assumptions (cumulative)

| # | Grade | Decision | Rationale | Artifact |
|---|-------|----------|-----------|----------|
| 1 | Confident | {decision} | {rationale} | spec.md |

{N} assumptions made ({C} confident, {T} tentative). Run /fab-clarify to review.

--- Implementation ---

{apply output — task execution details}

--- Review ---

{review output — validation results}

--- Hydrate ---

{hydrate output — validation and memory hydration}

Pipeline complete. Change hydrated.

Next: /fab-archive (archive change)
```

### Bail on Blocking Issue (Planning)

```
Fast-forwarding from brief...

--- Planning ---

## Spec: {Change Name}

{spec content}

Spec created.
Auto-clarify: spec — {resolved: 2, blocking: 1, non_blocking: 0}

⚠ Auto-clarify found 1 blocking issue in spec.md:
- The spec references "external auth provider" but available context doesn't specify which provider to use.

Run /fab-clarify to resolve this interactively, then /fab-ff to resume.
```

### Resume After Bail or Failure

```
Fast-forwarding (resuming)...

Skipping planning — all stages done.
Skipping implementation — already done.

--- Review ---

{review output}

--- Hydrate ---

{hydrate output}

Pipeline complete. Change hydrated.

Next: /fab-archive (archive change)
```

### Review Failure with Interactive Rework

```
Fast-forwarding from brief...

--- Planning ---

{planning output}

--- Implementation ---

{apply output}

--- Review ---

Review found {N} issue(s):
- {issue description}

Rework options:
1. **Fix code** — uncheck affected tasks, re-run apply
2. **Revise tasks** — edit tasks.md, re-run apply
3. **Revise spec** — reset to spec stage via /fab-continue spec

Which option? (1-3)
```

### Ambiguous Brief (questions first, then pipeline)

```
Fast-forwarding from brief...

Before I can generate all planning artifacts, I need to resolve a few ambiguities:

1. {question about spec scope}
2. {question about technical approach}

{user answers}

--- Planning ---

{... pipeline continues ...}
```

### Apply Failure

```
Fast-forwarding from brief...

--- Planning ---

{planning output}

--- Implementation ---

{partial apply output}

Task {ID} failed during apply: {reason}

Investigate the failure and re-run /fab-ff to resume from here.
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
| Task fails during apply | Stop pipeline, report which task failed and why, suggest re-running `/fab-ff` |
| Review fails | Present interactive rework menu (fix code, revise tasks, revise spec) |
| Hydrate fails | Report error with details |

---

## Key Difference from `/fab-continue` and `/fab-fff`

| Behavior | `/fab-continue` | `/fab-ff` | `/fab-fff` |
|----------|-----------------|-----------|-----------|
| Questions | Asked per-stage as needed | Frontloaded: one batch upfront | Same as fab-ff (frontloaded) |
| Auto-clarify | None (manual `/fab-clarify`) | Between each planning stage; bails on blockers | Same as fab-ff |
| Stages per invocation | One stage | Full pipeline: planning + apply + review + hydrate | Full pipeline: planning + apply + review + hydrate |
| On review failure | Rework options | Interactive rework menu | Immediate bail |
| Resumable? | Yes — re-invoke to resume current stage | Yes — re-invoke after bail or failure | Yes — skips completed stages |
| Confidence gate | None | None | Requires score >= 3.0 |
| Best for | Step-by-step progression | Fast full pipeline with interactive safety net | High-confidence changes, full autonomy |

---

## Next Steps Reference

After `/fab-ff` completes (hydrate):

`Next: /fab-archive (archive change)`

After `/fab-ff` bails (planning):

`Next: /fab-clarify (resolve issues) then /fab-ff (resume)`

After `/fab-ff` stops (apply failure):

`Next: Investigate failure, then /fab-ff (resume)`
