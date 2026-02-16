---
name: fab-ff
description: "Fast-forward from spec — confidence-gated pipeline from current stage through hydrate, with bail on review failure."
---

# /fab-ff [<change-name>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Fast-forward from spec through hydrate: tasks → apply → review → hydrate. Gated on confidence score (dynamic per-type thresholds via `calc-score.sh --check-gate`). Minimal auto-clarify (tasks only). Bails immediately on review failure. Resumable — re-running picks up from the first incomplete stage.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of `fab/current`. Resolution per `_context.md` (Change-name override).

---

## Pre-flight

1. Run preflight per `_context.md` Section 2. Pass `<change-name>` if provided.
2. **Spec prerequisite**: Check that spec is `active` or later (not `pending`). If `spec: pending`, STOP: `Spec not started. Run /fab-continue to generate the spec first, or use /fab-fff for the full pipeline.`
3. **Confidence gate**: Run `lib/calc-score.sh --check-gate <change_dir>`. If the gate fails → STOP: `Confidence is {score} of 5.0 (need > {threshold} for {change_type}). Run /fab-clarify to resolve, then retry.`
4. Log invocation: `lib/stageman.sh log-command <change_dir> "fab-ff"`

---

## Context Loading

Load per `_context.md` Sections 1-3 (config, constitution, intake, memory index, affected memory files, all completed artifacts).

---

## Behavior

> **Note**: All `.status.yaml` transitions in this skill use `lib/stageman.sh` CLI commands (`transition`, `set-state`, `set-checklist`) rather than direct file edits. All `transition` calls pass `fab-ff` as the driver. All `set-state` calls pass `fab-ff` when setting state to `active`.

### Resumability

Check `progress` from preflight. Skip stages already `done`. If `hydrate: done`, pipeline is already complete.

### Step 1: Generate `tasks.md`

*(Skip if `progress.tasks` is `done`.)*

Follow **Tasks Generation Procedure** (`_generation.md`). No frontloaded questions — spec is already done.

**Auto-Clarify**: Invoke `/fab-clarify` with `[AUTO-MODE]` prefix on the generated tasks. If `blocking: 0` → continue. If `blocking > 0` → **BAIL**: report issues, suggest `/fab-clarify` then `/fab-ff`.

### Step 2: Generate Quality Checklist

*(Skip if checklist already generated.)*

Follow **Checklist Generation Procedure** (`_generation.md`).

### Step 3: Update `.status.yaml` (Planning Complete)

Run `lib/stageman.sh transition <file> tasks apply fab-ff`. Then set checklist fields via `lib/stageman.sh set-checklist <file> generated true`, `lib/stageman.sh set-checklist <file> total <count>`, `lib/stageman.sh set-checklist <file> completed 0`.

### Step 4: Implementation

*(Skip if `progress.apply` is `done`.)*

Execute apply behavior per `/fab-continue` — parse unchecked tasks, execute in dependency order, run tests, mark `[x]` on completion.

**If task fails**: STOP with `Task {ID} failed: {reason}. Investigate and re-run /fab-ff.`

On success: run `lib/stageman.sh transition <file> apply review fab-ff`.

### Step 5: Review

*(Skip if `progress.review` is `done`.)*

Execute review behavior per `/fab-continue` — validate tasks, checklist, tests, spec match, memory drift.

**Pass**: run `lib/stageman.sh transition <file> review hydrate fab-ff`. Run `lib/stageman.sh log-review <change_dir> "passed"`. Proceed to Step 6.

**Fail**: STOP immediately. `Review failed. Run /fab-continue for rework options.` Run `lib/stageman.sh log-review <change_dir> "failed"`. No interactive rework menu.

### Step 6: Hydrate

*(Skip if `progress.hydrate` is `done`.)*

Execute hydrate behavior per `/fab-continue` — validate review passed, hydrate into `docs/memory/`, run `lib/stageman.sh set-state <file> hydrate done`.

---

## Output

```
/fab-ff — confidence {score} of 5.0, gate passed.

--- Planning ---
{tasks + checklist output}

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

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight fails | Abort with stderr message |
| Spec not started (`spec: pending`) | Abort: "Spec not started. Run /fab-continue or use /fab-fff." |
| Confidence below threshold | Abort with score, threshold, and guidance |
| Auto-clarify bails | Stop, report blocking issues, suggest `/fab-clarify` then `/fab-ff` |
| Task fails | Stop: "Task {ID} failed: {reason}. Investigate and re-run /fab-ff." |
| Review fails | Stop: "Review failed. Run /fab-continue for rework options." |
