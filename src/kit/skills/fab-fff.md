---
name: fab-fff
description: "Full pipeline — implementation, sub-agent review, hydrate, ship, and PR review — gated on the single intake confidence gate, with autonomous rework with bounded retry."
helpers: [_generation, _review, _srad]
---

# /fab-fff [<change-name>] [--force]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

Run the entire automated Fab pipeline — apply → review → hydrate → ship → review-pr — in a single invocation (everything after intake). Gated on the single intake confidence gate (>= 3.0, flat for all types), checked before the bracket. Autonomously reworks on review failure with bounded retry (3 cycles max, escalation after 2 consecutive fix-code failures). No `/fab-clarify` runs inside the bracket — clarification is intake-only. Resumable — re-running picks up from the first incomplete stage. Compare with `/fab-ff`, which stops at hydrate. The difference is scope only: `/fab-fff` extends through ship and review-pr; `/fab-ff` stops at hydrate. Both have the identical single intake gate.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one resolved via `.fab-status.yaml`. Resolution per `_preamble.md` (Change-name override).
- **`--force`** *(optional)* — bypass the intake confidence gate. All other behavior (rework loop, etc.) is unchanged. Output header includes "(force mode -- gate bypassed)".

---

## Pre-flight

1. Run preflight per `_preamble.md` Section 2. Pass `<change-name>` if provided.
2. Verify `intake.md` exists. If not, STOP: `Intake not found. Run /fab-new to create the intake first, then run /fab-fff.`
3. **Intake gate** (skip if `--force`): Run `fab score --check-gate --stage intake <change>`. If the gate fails → STOP: `Intake confidence is {score} of 5.0 (need >= 3.0). Run /fab-clarify to resolve, then retry.`

---

## Context Loading

Load per `_preamble.md` Sections 1-3 (config, constitution, intake, memory index, affected memory files, all completed artifacts).

---

## Behavior

> **Note**: All `.status.yaml` mutations in this skill use `fab status` event commands (`start`, `advance`, `finish`, `reset`, `fail`, `set-acceptance`) rather than direct file edits. Driver is optional in the CLI but this skill always passes `fab-fff`.
>
> **Dispatch**: All sub-skill invocations use the Agent tool (`general-purpose` subagent) per `_preamble.md` § Subagent Dispatch. Each subagent reads the target skill file, follows the specified behavior, and returns a structured result to the pipeline. Every `/fab-continue`-behavior subagent prompt MUST include: **"do NOT run `fab status` commands; return results only"** — the orchestrator runs those stages' transitions (finish/fail/reset) itself. (Ship and review-pr are the exception: `/git-pr` and `/git-pr-review` manage their own stage transitions internally — see Steps 4–5.)

### Resumability

Check `progress` from preflight. Skip stages already `done`. If `review-pr: done`, pipeline is already complete. If `progress.review` is `failed` (an interrupted fail→reset sequence), run `fab status start <change> review` first — the review-specific failed→active transition — then resume from Step 2.

### Step 1: Implementation (apply, with internal plan generation)

*(Skip if `progress.apply` is `done`.)* Since the intake gate already passed in pre-flight, if `progress.intake` is not `done`, finish intake: `fab status finish <change> intake fab-fff` (auto-activates apply).

Dispatch `/fab-continue` as subagent — Apply Behavior, change: `{id}` (prompt includes: do NOT run `fab status`; return results only). The subagent runs both apply sub-steps in a single invocation: (1) Plan Generation — co-generate `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance`) from `intake.md` per **Plan Generation Procedure** (`_generation.md`), unless `plan.md` already exists; (2) Task Execution — parse unchecked tasks under `## Tasks`, execute in dependency order, run tests, mark `[x]` on completion. Returns completion status or failure with task ID and reason.

No `/fab-clarify` runs here. Under-specified requirements are resolved inline by the apply agent as graded SRAD assumptions in `plan.md` `## Assumptions` — not via any clarify ceremony.

**If task fails**: STOP with `Task {ID} failed: {reason}. Investigate and re-run /fab-fff.`

On success: run `fab status finish <change> apply fab-fff`.

### Step 2: Review

*(Skip if `progress.review` is `done`.)*

Dispatch `/fab-continue` as subagent — Review Behavior, change: `{id}` (prompt includes: do NOT run `fab status`; return results only — verdict transitions belong to this orchestrator). The subagent reads `_review.md` for review dispatch instructions — both inward and outward sub-agents are defined there. It dispatches both sub-agents in parallel, merges their findings, and returns structured findings (must-fix / should-fix / nice-to-have) with pass/fail status.

**Pass**: run `fab status finish <change> review fab-fff`. Proceed to Step 3 (Hydrate).

**Fail**: Autonomous rework with bounded retry. Run `fab status fail <change> review` then `fab status reset <change> apply fab-fff`. The agent triages the sub-agent's prioritized findings and autonomously selects the rework path — no user interaction. Must-fix items are always addressed; should-fix items when clear and low-effort; nice-to-have items may be skipped.

**Decision heuristics** (applied to prioritized findings):
- **Must-fix: test failures, requirements mismatches, acceptance violations** → "Fix code" — uncheck affected tasks in `plan.md` `## Tasks` with `<!-- rework: reason -->`, re-run apply, then spawn a **fresh sub-agent** for re-review
- **Must-fix: missing functionality, incomplete coverage, wrong task breakdown** → "Revise plan" — edit `plan.md` (add/modify tasks under `## Tasks` and/or acceptance items under `## Acceptance`), re-run apply, then spawn a fresh sub-agent for re-review
- **Must-fix: requirements drift, requirements mismatch, fundamental approach issues** → "Revise requirements" — edit `plan.md` `## Requirements` plus the downstream `## Tasks`/`## Acceptance` it affects, re-run apply, then spawn a fresh sub-agent for re-review

**Retry cap**: Maximum **3 rework cycles** (each cycle = one rework action + one re-review by a fresh sub-agent). After 3 failed cycles, **BAIL** with:

```
Review failed after 3 rework attempts. Summary:
  Cycle 1: {action} — {what was done}
  Cycle 2: {action} — {what was done}
  Cycle 3: {action} — {what was done}
Run /fab-continue for manual rework options.
```

**Escalation rule**: If the agent chooses "Fix code" and the subsequent sub-agent review fails again on the same or similar issues, the agent MUST escalate to "Revise plan" or "Revise requirements" after **2 consecutive "fix code" attempts**. This is a hard rule — the agent SHALL NOT choose "Fix code" a third time in a row, even if it believes another code fix would work. Non-fix-code actions (revise plan, revise requirements) reset the consecutive counter.

### Step 3: Hydrate

*(Skip if `progress.hydrate` is `done`.)*

Dispatch `/fab-continue` as subagent — Hydrate Behavior, change: `{id}` (prompt includes: do NOT run `fab status`; return results only). The subagent validates review passed, hydrates into `docs/memory/`, and returns completion status.

On success: run `fab status finish <change> hydrate fab-fff`.

### Step 4: Ship

*(Skip if `progress.ship` is `done`.)*

Dispatch `/git-pr` as subagent — change: `{id}`. The subagent commits, pushes, and creates a GitHub PR. Handles `fab status` integration internally (start/finish ship stage). Returns PR URL or error.

**If git-pr fails**: STOP with the error from git-pr. The ship stage remains `active` for user retry.

On success: `progress.ship` becomes `done`, `progress.review-pr` auto-activates.

### Step 5: Review-PR

*(Skip if `progress.review-pr` is `done`.)*

Dispatch `/git-pr-review` as subagent — change: `{id}`. The subagent detects existing reviews, triages comments, applies fixes, and pushes. If no reviews exist, it requests a Copilot review and polls up to 10 minutes — see the timeout outcome below. Handles `fab status` integration internally (start/finish/fail review-pr stage). Returns completion status.

**If review-pr fails** (no PR found, processing error): STOP with the error.

**If no actionable reviews** (no automated reviewer available, or reviews with no inline comments to process): the stage completes as `done` — this is a successful no-op.

**If timeout** (Copilot review requested but not available within 10 minutes — git-pr-review's Step 6 timeout outcome): the subagent deliberately leaves `review-pr` `active` (no finish, no fail). Report `Review-PR pending (Copilot review requested, timed out waiting) — re-run /git-pr-review when ready` **instead of** `Pipeline complete.` and stop.

On success: `progress.review-pr` becomes `done`.

---

## Output

```
/fab-fff — intake confidence {score} of 5.0, gate passed.

--- Implementation ---
{apply output (plan generation — incl. ## Requirements — + task execution)}

## Assumptions (cumulative)
{table with Artifact column — apply-recorded assumptions from plan.md}

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

Resuming shows `(resuming)...` header and `Skipping {stage} — already done.` for completed stages. Bail/failure stops at the relevant stage with `Next:` derived from the state reached per state table in `_preamble.md`. On the Step 5 timeout outcome, the closing line is `Review-PR pending (Copilot review requested, timed out waiting) — re-run /git-pr-review when ready` instead of `Pipeline complete.`

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight fails | Abort with stderr message |
| `intake.md` missing | Abort: "Run /fab-new first." |
| Intake gate fails (confidence < 3.0) | Stop with score and guidance |
| Task fails | Stop: "Task {ID} failed: {reason}. Investigate and re-run /fab-fff." |
| Review fails | Autonomous rework: agent triages sub-agent's prioritized findings, selects path, 3-cycle retry cap (each re-review by fresh sub-agent), escalation after 2 consecutive fix-code. Bail after 3 cycles with summary. Escalation paths: revise plan or revise requirements (both in `plan.md`). |
| Ship fails | Stop with git-pr error. User retries /fab-fff or /git-pr. |
| Review-PR fails | Stop with git-pr-review error. User retries /fab-fff or /git-pr-review. |
| Review-PR timeout (Copilot review requested, not yet available) | Stage deliberately left `active`. Report `Review-PR pending (Copilot review requested, timed out waiting) — re-run /git-pr-review when ready` and stop — no finish, no fail. |
