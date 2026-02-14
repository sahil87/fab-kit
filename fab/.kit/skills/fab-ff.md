---
name: fab-ff
description: "Fast-forward through the entire pipeline — planning, implementation, review, and hydrate — with interactive clarification stops."
---

# /fab-ff [<change-name>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Fast-forward through the entire Fab pipeline: planning (spec, tasks) → apply → review → hydrate. Interleaves auto-clarify between planning stages and stops for interactive resolution on blockers. No confidence gate (unlike `/fab-fff`). Resumable — re-running picks up from the first incomplete stage.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of `fab/current`. Resolution per `_context.md` (Change-name override).

---

## Pre-flight

1. Run preflight per `_context.md` Section 2. Pass `<change-name>` if provided.
2. Verify `brief.md` exists. If not, STOP: `Brief not found. Run /fab-new to create the brief first, then run /fab-ff.`

---

## Context Loading

Load per `_context.md` Sections 1-3 (config, constitution, brief, memory index, affected memory files, all completed artifacts).

---

## Behavior

### Resumability

Check `progress` from preflight. Skip stages already `done`. If `hydrate: done`, pipeline is already complete.

### Step 1: Frontload All Questions

Apply SRAD across the brief for all planning stages. Collect **Unresolved** decisions into a single batch. Confident/Tentative decisions are assumed (tracked in cumulative Assumptions summary).

- **Unresolved exist**: Present as numbered list, wait for answers, then proceed.
- **None**: Skip to Step 2.

At most one Q&A round.

### Step 2: Generate `spec.md`

*(Skip if `progress.spec` is `done`.)*

Follow **Spec Generation Procedure** (`_generation.md`). Incorporate answers from Step 1 — no `[NEEDS CLARIFICATION]` markers. Update `.status.yaml` via `lib/stageman.sh set-state <file> spec done`.

**Auto-Clarify**: Invoke `/fab-clarify` with `[AUTO-MODE]` prefix. If `blocking: 0` → continue. If `blocking > 0` → **BAIL**: report issues, suggest `/fab-clarify` then `/fab-ff`.

### Step 3: Generate `tasks.md`

*(Skip if `progress.tasks` is `done`.)*

Follow **Tasks Generation Procedure** (`_generation.md`). Auto-clarify with same bail logic.

### Step 4: Generate Quality Checklist

Follow **Checklist Generation Procedure** (`_generation.md`).

### Step 5: Update `.status.yaml` (Planning Complete)

Run `lib/stageman.sh transition <file> tasks apply`. Then set checklist fields via `lib/stageman.sh set-checklist <file> generated true`, `lib/stageman.sh set-checklist <file> total <count>`, `lib/stageman.sh set-checklist <file> completed 0`.

### Step 6: Implementation

*(Skip if `progress.apply` is `done`.)*

Execute apply behavior per `/fab-continue` — parse unchecked tasks, execute in dependency order, run tests, mark `[x]` on completion.

**If task fails**: STOP with `Task {ID} failed: {reason}. Investigate and re-run /fab-ff.`

On success: run `lib/stageman.sh transition <file> apply review`.

### Step 7: Review

*(Skip if `progress.review` is `done`.)*

Execute review behavior per `/fab-continue` — validate tasks, checklist, tests, spec match, memory drift.

**Pass**: run `lib/stageman.sh transition <file> review hydrate`. Proceed to Step 8.

**Fail**: Present interactive rework menu: fix code (uncheck tasks with `<!-- rework: reason -->`), revise tasks, or revise spec (reset via `/fab-continue spec`).

### Step 8: Hydrate

*(Skip if `progress.hydrate` is `done`.)*

Execute hydrate behavior per `/fab-continue` — validate review passed, hydrate into `docs/memory/`, run `lib/stageman.sh set-state <file> hydrate done`.

---

## Output

```
Fast-forwarding from {starting stage}...

--- Planning ---
{spec + tasks + checklist output, with auto-clarify results}

## Assumptions (cumulative)
{table with Artifact column}

--- Implementation ---
{apply output}

--- Review ---
{review output}

--- Hydrate ---
{hydrate output}

Pipeline complete. Change hydrated.

Next: /fab-archive
```

Resuming shows `(resuming)...` header and `Skipping {stage} — already done.` for completed stages. Bail/failure stops at the relevant stage with contextual Next line.
