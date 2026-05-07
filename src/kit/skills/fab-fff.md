---
name: fab-fff
description: "Full pipeline — planning, implementation, sub-agent review, hydrate, ship, and PR review — confidence-gated, with auto-clarify and autonomous rework with bounded retry."
helpers: [_generation, _review]
---

# /fab-fff [<change-name>] [--force]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

Run the entire Fab pipeline from intake through PR review in a single invocation. Confidence-gated with identical gates to `/fab-ff`: (1) intake gate — indicative confidence >= 3.0, (2) spec gate — confidence >= per-type threshold. Interleaves auto-clarify between planning stages, and autonomously reworks on review failure with bounded retry (3 cycles max, escalation after 2 consecutive fix-code failures). Resumable — re-running picks up from the first incomplete stage. Compare with `/fab-ff`, which stops at hydrate. The difference is scope only: `/fab-fff` extends through ship and review-pr; `/fab-ff` stops at hydrate. Both have identical confidence gates.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one resolved via `.fab-status.yaml`. Resolution per `_preamble.md` (Change-name override).
- **`--force`** *(optional)* — bypass all confidence gates (intake gate and spec gate). All other behavior (auto-clarify, rework loop, etc.) is unchanged. Output header includes "(force mode -- gates bypassed)".

---

## Pre-flight

1. Run preflight per `_preamble.md` Section 2. Pass `<change-name>` if provided.
2. Verify `intake.md` exists. If not, STOP: `Intake not found. Run /fab-new to create the intake first, then run /fab-fff.`
3. **Intake gate** (skip if `--force`): Run `fab score --check-gate --stage intake <change>`. If the gate fails → STOP: `Indicative confidence is {score} of 5.0 (need >= 3.0). Run /fab-clarify to resolve, then retry.`

---

## Context Loading

Load per `_preamble.md` Sections 1-3 (config, constitution, intake, memory index, affected memory files, all completed artifacts).

---

## Behavior

> **Note**: All `.status.yaml` mutations in this skill use `fab status` event commands (`start`, `advance`, `finish`, `reset`, `fail`, `set-acceptance`) rather than direct file edits. Driver is optional in the CLI but this skill always passes `fab-fff`.
>
> **Dispatch**: All sub-skill invocations use the Agent tool (`general-purpose` subagent) per `_preamble.md` § Subagent Dispatch. Each subagent reads the target skill file, follows the specified behavior, and returns a structured result to the pipeline.

### Resumability

Check `progress` from preflight. Skip stages already `done`. If `review-pr: done`, pipeline is already complete.

### Step 1: Generate `spec.md`

*(Skip if `progress.spec` is `done`.)*

Follow **Spec Generation Procedure** (`_generation.md`). No frontloaded questions. Update `.status.yaml` via `fab status finish <change> intake fab-fff`.

**Spec gate** (skip if `--force`): After spec generation, run `fab score --check-gate <change>`. If the gate fails → **STOP**: `Confidence is {score} of 5.0 (need >= {threshold} for {change_type}). Run /fab-clarify to resolve, then retry /fab-fff.`

**Auto-Clarify**: Dispatch `/fab-clarify` as subagent — `[AUTO-MODE]`, target: `spec.md`, change: `{id}`. Returns `{resolved, blocking, non_blocking}`. If `blocking: 0` → continue. If `blocking > 0` → **BAIL**: report blocking issues, suggest `/fab-clarify` then `/fab-fff`.

### Step 2: Implementation (apply, with internal plan generation)

*(Skip if `progress.apply` is `done`.)*

Dispatch `/fab-continue` as subagent — Apply Behavior, change: `{id}`. The subagent runs both apply sub-steps in a single invocation: (1) Plan Generation — produce `plan.md` from `spec.md` per **Plan Generation Procedure** (`_generation.md`), unless `plan.md` already exists; (2) Task Execution — parse unchecked tasks under `## Tasks`, execute in dependency order, run tests, mark `[x]` on completion. Returns completion status or failure with task ID and reason.

**Auto-Clarify on plan** *(after plan generation, before task execution; only when `plan.md` was newly written this run)*: Dispatch `/fab-clarify` as subagent — `[AUTO-MODE]`, target: `plan`, change: `{id}`. Same bail logic as Step 1.

**If task fails**: STOP with `Task {ID} failed: {reason}. Investigate and re-run /fab-fff.`

On success: run `fab status finish <change> apply fab-fff`.

### Step 3: Review

*(Skip if `progress.review` is `done`.)*

Dispatch `/fab-continue` as subagent — Review Behavior, change: `{id}`. The subagent reads `_review.md` for review dispatch instructions — both inward and outward sub-agents are defined there. It dispatches both sub-agents in parallel, merges their findings, and returns structured findings (must-fix / should-fix / nice-to-have) with pass/fail status.

**Pass**: run `fab status finish <change> review fab-fff`. Proceed to Step 7.

**Fail**: Autonomous rework with bounded retry. Run `fab status fail <change> review` then `fab status reset <change> apply fab-fff`. The agent triages the sub-agent's prioritized findings and autonomously selects the rework path — no user interaction. Must-fix items are always addressed; should-fix items when clear and low-effort; nice-to-have items may be skipped.

**Decision heuristics** (applied to prioritized findings):
- **Must-fix: test failures, spec mismatches, acceptance violations** → "Fix code" — uncheck affected tasks in `plan.md` `## Tasks` with `<!-- rework: reason -->`, re-run apply, then spawn a **fresh sub-agent** for re-review
- **Must-fix: missing functionality, incomplete coverage, wrong task breakdown** → "Revise plan" — edit `plan.md` (add/modify tasks under `## Tasks` and/or acceptance items under `## Acceptance`), re-run apply, then spawn a fresh sub-agent for re-review
- **Must-fix: spec drift, requirements mismatch, fundamental approach issues** → "Revise spec" — reset to spec stage, regenerate downstream, re-run apply, then spawn a fresh sub-agent for re-review

**Retry cap**: Maximum **3 rework cycles** (each cycle = one rework action + one re-review by a fresh sub-agent). After 3 failed cycles, **BAIL** with:

```
Review failed after 3 rework attempts. Summary:
  Cycle 1: {action} — {what was done}
  Cycle 2: {action} — {what was done}
  Cycle 3: {action} — {what was done}
Run /fab-continue for manual rework options.
```

**Escalation rule**: If the agent chooses "Fix code" and the subsequent sub-agent review fails again on the same or similar issues, the agent MUST escalate to "Revise plan" or "Revise spec" after **2 consecutive "fix code" attempts**. This is a hard rule — the agent SHALL NOT choose "Fix code" a third time in a row, even if it believes another code fix would work. Non-fix-code actions (revise plan, revise spec) reset the consecutive counter.

### Step 4: Hydrate

*(Skip if `progress.hydrate` is `done`.)*

Dispatch `/fab-continue` as subagent — Hydrate Behavior, change: `{id}`. The subagent validates review passed, hydrates into `docs/memory/`, and runs `fab status finish <change> hydrate fab-fff`. Returns completion status.

### Step 5: Ship

*(Skip if `progress.ship` is `done`.)*

Dispatch `/git-pr` as subagent — change: `{id}`. The subagent commits, pushes, and creates a GitHub PR. Handles statusman integration internally (start/finish ship stage). Returns PR URL or error.

**If git-pr fails**: STOP with the error from git-pr. The ship stage remains `active` for user retry.

On success: `progress.ship` becomes `done`, `progress.review-pr` auto-activates.

### Step 6: Review-PR

*(Skip if `progress.review-pr` is `done`.)*

Dispatch `/git-pr-review` as subagent — change: `{id}`. The subagent detects existing reviews, triages comments, applies fixes, and pushes. If no reviews exist, prints a stop message and completes. Handles statusman integration internally (start/finish/fail review-pr stage). Returns completion status.

**If review-pr fails** (no PR found, processing error): STOP with the error.

**If no reviews found**: the stage completes as `done` — this is a successful no-op.

On success: `progress.review-pr` becomes `done`.

---

## Output

```
/fab-fff — confidence {score} of 5.0, gate passed.

--- Planning ---
{spec output, with auto-clarify results}

## Assumptions (cumulative)
{table with Artifact column}

--- Implementation ---
{apply output (plan generation + task execution)}

--- Review ---
{review output}

--- Hydrate ---
{hydrate output}

--- Ship ---
{git-pr output}

--- Review-PR ---
{git-pr-review output}

Pipeline complete.

Next: {per state table}
```

Resuming shows `(resuming)...` header and `Skipping {stage} — already done.` for completed stages. Bail/failure stops at the relevant stage with `Next:` derived from the state reached per state table in `_preamble.md`.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight fails | Abort with stderr message |
| `intake.md` missing | Abort: "Run /fab-new first." |
| Intake gate fails (indicative < 3.0) | Stop with score and guidance |
| Spec gate fails (confidence < threshold) | Stop with score, threshold, and guidance |
| Auto-clarify bails | Stop, report blocking issues, suggest `/fab-clarify` then `/fab-fff` |
| Task fails | Stop: "Task {ID} failed: {reason}. Investigate and re-run /fab-fff." |
| Review fails | Autonomous rework: agent triages sub-agent's prioritized findings, selects path, 3-cycle retry cap (each re-review by fresh sub-agent), escalation after 2 consecutive fix-code. Bail after 3 cycles with summary. Escalation paths: revise plan (`plan.md`) or revise spec (`spec.md`). |
| Ship fails | Stop with git-pr error. User retries /fab-fff or /git-pr. |
| Review-PR fails | Stop with git-pr-review error. User retries /fab-fff or /git-pr-review. |
