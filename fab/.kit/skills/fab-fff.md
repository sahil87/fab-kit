---
name: fab-fff
description: "Full autonomous pipeline — confidence gate, then planning → apply → review → hydrate with no interactive stops."
---

# /fab-fff [<change-name>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Run the entire Fab pipeline from planning through hydrate in a single invocation, gated on confidence against mode/type threshold policy. Unlike `/fab-ff`, never stops for interaction — bails immediately on review failure and auto-clarifies without user input.

---

## Arguments

- **`<change-name>`** *(optional)* — resolves per `_context.md` §2 (transient override). Defaults to active change.

---

## Pre-flight

1. Run preflight per `_context.md` §2
2. Verify `brief.md` exists. If not, STOP: `Brief not found. Run /fab-new first.`
3. **Confidence gate**: Read `confidence.score`, scoring mode, and change type.
   - legacy mode threshold: `3.0`
   - fuzzy mode thresholds:
     - `bugfix`: `2.7`
     - `refactor`: `3.0`
     - `feature`: `3.3`
     - `architecture`: `3.6`
   - unknown change type defaults to `feature`
   If score is below threshold or score is missing → STOP: `Confidence is {score} of 5.0 (need >= {threshold} for {change_type}). Run /fab-clarify to resolve, then retry.`

---

## Context Loading

Load per `_context.md` layers 1-3.

---

## Behavior

> **Note**: All `.status.yaml` transitions in this skill use `lib/stageman.sh` CLI commands (`transition`, `set-state`, `set-checklist`, `set-confidence`) rather than direct file edits. See `/fab-continue` and `/fab-ff` for specific invocations per step.

### Resumability

Skip stages already `done` or `skipped`. Re-running picks up from first incomplete stage.

### Step 1: Planning (fab-ff)

*(Skip if spec and tasks are `done`/`skipped`.)*

Execute `/fab-ff` planning behavior (Steps 1-5: frontload questions, interleaved auto-clarify, bail on blockers).

**If fab-ff bails**: STOP. `fab-ff bailed on blocking issues. Run /fab-clarify then /fab-fff.`

### Step 2: Implementation

*(Skip if `progress.apply` is `done`.)*

Execute apply behavior — parse unchecked tasks, execute in dependency order, run tests, mark complete.

### Step 3: Review

*(Skip if `progress.review` is `done`.)*

Execute review behavior. **If fails**: STOP immediately (no interactive rework menu). `Review failed. Run /fab-continue for rework options.`

### Step 4: Hydrate

*(Skip if `progress.hydrate` is `done`.)*

Execute hydrate behavior — validate review passed, hydrate into `docs/memory/`, set `hydrate: done`.

---

## Output

```
/fab-fff — confidence {score} of 5.0, gate passed.

--- Planning (fab-ff) ---
{fab-ff output}

--- Implementation ---
{apply output}

--- Review ---
{review output}

--- Hydrate ---
{hydrate output}

Pipeline complete. Change hydrated.

Next: /fab-archive
```

Skipped stages show `Skipping {stage} — already done.` Failures end at the failing stage with bail/failure message.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight fails | Abort with stderr message |
| `brief.md` missing | Abort: "Run /fab-new first." |
| Confidence < 3.0 or missing | Abort with score and guidance |
| fab-ff bails | Stop, report blocking issues |
| Review fails | Stop, report failure details |
