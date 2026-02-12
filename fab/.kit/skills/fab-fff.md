---
name: fab-fff
description: "Full autonomous pipeline — confidence gate, then planning → apply → review → archive with no interactive stops."
---

# /fab-fff

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Run the entire Fab pipeline from planning through archive in a single invocation, gated on confidence score >= 3.0. Each stage uses the same behavior as its standalone invocation. Unlike `/fab-ff`, which also runs the full pipeline but stops for interactive clarification, `/fab-fff` never stops — it bails immediately on review failure and auto-clarifies without user input.

Use this when confidence is high and you want to go from brief to archived change without manual intervention.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence`

Then verify preconditions:

4. Verify that `brief.md` exists in the change directory (`fab/changes/{name}/brief.md`)

**If `brief.md` does not exist, STOP.** Output:

> `Brief not found. Run /fab-new to create the brief first, then run /fab-fff.`

### Confidence Gate

Read `confidence.score` from the preflight output (or from `.status.yaml` if preflight does not yet emit it).

**If `confidence.score < 3.0`, STOP.** Output:

> `Confidence is {score} (need >= 3.0). Run /fab-clarify to resolve tentative/unresolved decisions, then retry.`

**If the `confidence` block is missing entirely** (legacy change without confidence tracking), treat as score 0 and refuse to run with the same message above.

**If `confidence.score >= 3.0`, proceed.**

---

## Context Loading

Same as `/fab-ff` — load all context upfront:

1. `fab/config.yaml` — project config, tech stack
2. `fab/constitution.md` — project principles and constraints
3. `fab/changes/{name}/brief.md` — the completed brief
4. `fab/docs/index.md` — documentation landscape
5. Specific centralized docs referenced by the brief's **Affected Docs** section

---

## Behavior

### Resumability

On invocation, check the `progress` map from preflight output. **Skip stages already marked `done` or `skipped`.** This means:

- If all planning stages are `done` → skip fab-ff, start at fab-apply
- If `progress.apply` is `done` → skip fab-apply, start at fab-review
- If `progress.review` is `done` → skip fab-review, start at fab-archive

This makes `/fab-fff` resumable after interruption or failure — re-running picks up from the first incomplete stage.

### Step 1: Planning (fab-ff)

*(Skip if all planning stages — spec, tasks — are `done` or `skipped`.)*

Execute `/fab-ff` planning behavior (Steps 1-5: frontload questions, interleaved auto-clarify, bail on blockers). This generates spec and tasks with quality checklist. Only the planning portion of fab-ff is used here — fab-fff handles apply/review/archive in its own subsequent steps.

**If fab-ff bails on blocking issues**, the `/fab-fff` pipeline stops. Output:

> `fab-ff bailed on blocking issues. Run /fab-clarify to resolve these, then /fab-fff to retry.`

Do not proceed to fab-apply.

### Step 2: Implementation (fab-apply)

*(Skip if `progress.apply` is `done`.)*

Execute `/fab-apply` behavior — parse unchecked tasks, execute in dependency order, run tests after each, mark tasks complete.

### Step 3: Review (fab-review)

*(Skip if `progress.review` is `done`.)*

Execute `/fab-review` behavior — validate implementation against specs and checklists.

**If review fails**, the `/fab-fff` pipeline stops immediately. Do NOT offer the interactive rework menu that standalone `/fab-review` provides. Output the review failure details and:

> `Review failed. Run /fab-review to see rework options, or /fab-clarify to refine artifacts.`

### Step 4: Archive (fab-archive)

*(Skip if `progress.archive` is `done`.)*

Execute `/fab-archive` behavior — validate review passed, hydrate learnings into centralized docs, move change to archive, clear pointer.

---

## Output

### Clean Full Pipeline

```
/fab-fff — confidence {score}, gate passed.

--- Planning (fab-ff) ---

{fab-ff output}

--- Implementation (fab-apply) ---

{fab-apply output}

--- Review (fab-review) ---

{fab-review output}

--- Archive (fab-archive) ---

{fab-archive output}

Pipeline complete. Change archived.

Next: /fab-new <description> (start next change)
```

### Confidence Gate Failure

```
Confidence is 2.5 (need >= 3.0). Run /fab-clarify to resolve tentative/unresolved decisions, then retry.
```

### fab-ff Bail

```
/fab-fff — confidence {score}, gate passed.

--- Planning (fab-ff) ---

{fab-ff output including bail message}

fab-ff bailed on blocking issues. Run /fab-clarify to resolve these, then /fab-fff to retry.
```

### Review Failure

```
/fab-fff — confidence {score}, gate passed.

--- Planning (fab-ff) ---

{fab-ff output}

--- Implementation (fab-apply) ---

{fab-apply output}

--- Review (fab-review) ---

{review failure details}

Review failed. Run /fab-review to see rework options, or /fab-clarify to refine artifacts.
```

### Resume After Interruption

```
/fab-fff — confidence {score}, gate passed.

Skipping planning — all stages done.
Skipping implementation — already done.

--- Review (fab-review) ---

{fab-review output}

--- Archive (fab-archive) ---

{fab-archive output}

Pipeline complete. Change archived.

Next: /fab-new <description> (start next change)
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight script exits non-zero | Abort with the stderr message from `fab-preflight.sh` |
| `brief.md` does not exist | Abort with: "Brief not found. Run /fab-new first." |
| `confidence.score < 3.0` | Abort with: "Confidence is {score} (need >= 3.0)." |
| `confidence` block missing | Treat as score 0, abort with confidence message |
| fab-ff bails on blocking issues | Stop pipeline, report blocking issues |
| fab-review fails | Stop pipeline, report failure details |
| Any stage already `done` | Skip it and continue to next stage |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **Yes** — progresses through all stages to `archive: done` |
| Idempotent? | **Yes** — safe to re-invoke; skips completed stages |
| Modifies artifacts? | **Yes** — generates planning artifacts, implements tasks, hydrates docs |
| Updates `.status.yaml`? | **Yes** — each sub-skill updates as it completes |
| Recomputes confidence? | **No** — reads the score but does not update it |

---

## Key Difference from `/fab-ff` and Individual Skills

| Behavior | Individual skills | `/fab-ff` | `/fab-fff` |
|----------|-------------------|-----------|-----------|
| Invocations | One per stage | Single invocation, full pipeline | Single invocation, full pipeline |
| On review failure | Interactive rework menu | Interactive rework menu | Immediate bail |
| Confidence gate | None | None | Requires score >= 3.0 |
| User interaction | Per-skill | Interactive stops when needed | Minimal — frontloads questions, rest is autonomous |
| Best for | Step-by-step control | Fast pipeline with safety net | High-confidence, full autonomy |

---

## Next Steps Reference

After `/fab-fff` completes:

`Next: /fab-new <description> (start next change)`

After `/fab-fff` bails:

`Next: /fab-clarify (resolve issues) then /fab-fff (retry)`
