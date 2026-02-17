---
name: fab-ff
description: "Fast-forward from spec — confidence-gated pipeline from current stage through hydrate, with sub-agent review, auto-rework loop, and interactive fallback."
---

# /fab-ff [<change-name>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Fast-forward from spec through hydrate: tasks → apply → review → hydrate. Gated on confidence score (dynamic per-type thresholds via `calc-score.sh --check-gate`). Minimal auto-clarify (tasks only). On review failure, auto-loops between apply and review (sub-agent review, prioritized findings, comment triage) for up to 3 cycles, then falls back to interactive rework options. Resumable — re-running picks up from the first incomplete stage.

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

Dispatch review to a **sub-agent** per `/fab-continue` Review Behavior — the sub-agent runs in a separate execution context, performs all validation checks, and returns structured findings with three-tier priority (must-fix / should-fix / nice-to-have).

**Pass**: run `lib/stageman.sh transition <file> review hydrate fab-ff`. Run `lib/stageman.sh log-review <change_dir> "passed"`. Proceed to Step 6.

**Fail**: Auto-rework loop with bounded retry, then interactive fallback. Run `lib/stageman.sh set-state <file> review failed` then `lib/stageman.sh set-state <file> apply active fab-ff`.

#### Auto-Rework Loop (up to 3 cycles)

The agent triages the sub-agent's prioritized findings and autonomously selects the rework path — no user interaction. Must-fix items are always addressed; should-fix items when clear and low-effort; nice-to-have items may be skipped.

**Decision heuristics** (applied to prioritized findings):
- **Must-fix: test failures, spec mismatches, checklist violations** → "Fix code" — uncheck affected tasks with `<!-- rework: reason -->`, re-run apply, then spawn a **fresh sub-agent** for re-review
- **Must-fix: missing functionality, incomplete coverage, wrong task breakdown** → "Revise tasks" — add/modify tasks in `tasks.md`, re-run apply, then spawn a fresh sub-agent for re-review
- **Must-fix: spec drift, requirements mismatch, fundamental approach issues** → "Revise spec" — reset to spec stage, regenerate downstream, re-run apply, then spawn a fresh sub-agent for re-review

Run `lib/stageman.sh log-review <change_dir> "failed" "<rework-option>"` for each rework cycle.

**Escalation rule**: If the agent chooses "Fix code" and the subsequent sub-agent review fails again on the same or similar issues, the agent MUST escalate to "Revise tasks" or "Revise spec" after **2 consecutive "fix code" attempts**. This is a hard rule — the agent SHALL NOT choose "Fix code" a third time in a row, even if it believes another code fix would work. Non-fix-code actions (revise tasks, revise spec) reset the consecutive counter.

#### Interactive Fallback (after 3 failed cycles)

After 3 auto-rework cycles fail, fall back to interactive rework — present the user with the same 3 rework options as `/fab-continue`:

- **Fix code** — the agent identifies affected tasks, unchecks them in `tasks.md` with `<!-- rework: reason -->` annotations, re-runs apply for unchecked tasks, then spawns a fresh sub-agent for re-review
- **Revise tasks** — the user edits `tasks.md` (add/modify tasks), then the agent re-runs apply for unchecked tasks and spawns a fresh sub-agent for re-review
- **Revise spec** — resets to spec stage, regenerates downstream, re-runs apply, then spawns a fresh sub-agent for re-review

Once in interactive fallback, there is no further retry cap — the user is in the loop. If review fails again after user-directed rework, present the interactive menu again.

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

Next: {per state table}
```

Resuming shows `(resuming)...` header and `Skipping {stage} — already done.` for completed stages. Bail/failure stops at the relevant stage with `Next:` derived from the state reached per state table in `_context.md`.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight fails | Abort with stderr message |
| Spec not started (`spec: pending`) | Abort: "Spec not started. Run /fab-continue or use /fab-fff." |
| Confidence below threshold | Abort with score, threshold, and guidance |
| Auto-clarify bails | Stop, report blocking issues, suggest `/fab-clarify` then `/fab-ff` |
| Task fails | Stop: "Task {ID} failed: {reason}. Investigate and re-run /fab-ff." |
| Review fails | Auto-rework loop: agent triages sub-agent's prioritized findings, selects path, up to 3 cycles (each re-review by fresh sub-agent), escalation after 2 consecutive fix-code. Falls back to interactive rework after 3 cycles. |
