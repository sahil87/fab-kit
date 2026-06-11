---
name: fab-ff
description: "Fast-forward through hydrate — confidence-gated pipeline from intake through hydrate, with sub-agent review, auto-rework loop, and stop on exhaustion."
helpers: [_generation, _review]
---

# /fab-ff [<change-name>] [--force]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

Fast-forward through hydrate: apply → review → hydrate (everything after intake, stopping before the PR stages). Two gates where execution can stop: (1) intake gate — confidence >= 3.0 (flat, all types), checked before the bracket; (2) review gate — stops after 3 autonomous rework cycles. On any gate stop, the user can intervene then re-run. Resumable — re-running picks up from the first incomplete stage. No `/fab-clarify` runs inside the bracket — clarification is intake-only.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one resolved via `.fab-status.yaml`. Resolution per `_preamble.md` (Change-name override).
- **`--force`** *(optional)* — bypass the intake confidence gate. All other behavior (rework loop, etc.) is unchanged. Output header includes "(force mode -- gate bypassed)".

---

## Pre-flight

1. Run preflight per `_preamble.md` Section 2. Pass `<change-name>` if provided.
2. **Intake prerequisite**: Verify `intake.md` exists. If not, STOP: `Intake not found. Run /fab-new to create the intake first.`
3. **Intake gate** *(skip if `--force`)*: Run `fab score --check-gate --stage intake <change>`. If the gate fails → STOP: `Intake confidence is {score} of 5.0 (need >= 3.0). Run /fab-clarify to resolve, then retry.`

---

## Context Loading

Load per `_preamble.md` Sections 1-3 (config, constitution, intake, memory index, affected memory files, all completed artifacts).

---

## Behavior

> **Note**: All `.status.yaml` mutations in this skill use `fab status` event commands (`start`, `advance`, `finish`, `reset`, `fail`, `set-acceptance`) rather than direct file edits. The driver argument is optional, but this skill always passes `fab-ff`.
>
> **Dispatch**: All sub-skill invocations use the Agent tool (`general-purpose` subagent) per `_preamble.md` § Subagent Dispatch. Each subagent reads the target skill file, follows the specified behavior, and returns a structured result to the pipeline. Every `/fab-continue`-behavior subagent prompt MUST include: **"do NOT run `fab status` commands; return results only"** — the orchestrator runs those stages' transitions (finish/fail/reset) itself.

### Resumability

Check `progress` from preflight. Skip stages already `done`. If `hydrate: done`, pipeline is already complete. If `progress.review` is `failed` (an interrupted fail→reset sequence), run `fab status start <change> review` first — the review-specific failed→active transition — then resume from Step 2.

### Step 1: Implementation (apply, with internal plan generation)

*(Skip if `progress.apply` is `done`.)* Since the intake gate already passed in pre-flight, if `progress.intake` is not `done`, finish intake: `fab status finish <change> intake fab-ff` (auto-activates apply).

Dispatch `/fab-continue` as subagent — Apply Behavior, change: `{id}` (prompt includes: do NOT run `fab status`; return results only). The subagent runs both apply sub-steps in a single invocation: (1) Plan Generation — co-generate `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance`) from `intake.md` per **Plan Generation Procedure** (`_generation.md`), unless `plan.md` already exists; (2) Task Execution — parse unchecked tasks under `## Tasks`, execute in dependency order, run tests, mark `[x]` on completion. Returns completion status or failure with task ID and reason.

No `/fab-clarify` runs here. Under-specified requirements are resolved inline by the apply agent as graded SRAD assumptions in `plan.md` `## Assumptions` — not via any clarify ceremony.

**If task fails**: STOP with `Task {ID} failed: {reason}. Investigate and re-run /fab-ff.`

On success: run `fab status finish <change> apply fab-ff`.

### Step 2: Review

*(Skip if `progress.review` is `done`.)*

Dispatch `/fab-continue` as subagent — Review Behavior, change: `{id}` (prompt includes: do NOT run `fab status`; return results only — verdict transitions belong to this orchestrator). The subagent reads `_review.md` for review dispatch instructions — both inward and outward sub-agents are defined there. It dispatches both sub-agents in parallel, merges their findings, and returns structured findings (must-fix / should-fix / nice-to-have) with pass/fail status.

**Pass**: run `fab status finish <change> review fab-ff`. Proceed to Step 3 (Hydrate).

**Fail**: Auto-rework loop with bounded retry, then interactive fallback. Run `fab status fail <change> review` then `fab status reset <change> apply fab-ff`.

#### Auto-Rework Loop (up to 3 cycles)

The agent triages the sub-agent's prioritized findings and autonomously selects the rework path — no user interaction. Must-fix items are always addressed; should-fix items when clear and low-effort; nice-to-have items may be skipped.

**Decision heuristics** (applied to prioritized findings):
- **Must-fix: test failures, requirements mismatches, acceptance violations** → "Fix code" — uncheck affected tasks in `plan.md` `## Tasks` with `<!-- rework: reason -->`, re-run apply, then spawn a **fresh sub-agent** for re-review
- **Must-fix: missing functionality, incomplete coverage, wrong task breakdown** → "Revise plan" — edit `plan.md` (add/modify tasks under `## Tasks` and/or acceptance items under `## Acceptance`), re-run apply, then spawn a fresh sub-agent for re-review
- **Must-fix: requirements drift, requirements mismatch, fundamental approach issues** → "Revise requirements" — edit `plan.md` `## Requirements` plus the downstream `## Tasks`/`## Acceptance` it affects, re-run apply, then spawn a fresh sub-agent for re-review

**Escalation rule**: If the agent chooses "Fix code" and the subsequent sub-agent review fails again on the same or similar issues, the agent MUST escalate to "Revise plan" or "Revise requirements" after **2 consecutive "fix code" attempts**. This is a hard rule — the agent SHALL NOT choose "Fix code" a third time in a row, even if it believes another code fix would work. Non-fix-code actions (revise plan, revise requirements) reset the consecutive counter.

#### Stop (after 3 failed cycles)

After 3 auto-rework cycles fail, **STOP** with a per-cycle summary:

```
Review failed after 3 rework attempts. Summary:
  Cycle 1: {action} — {what was done}
  Cycle 2: {action} — {what was done}
  Cycle 3: {action} — {what was done}
Run /fab-continue for manual rework options.
```

The user can run `/fab-continue` for interactive rework, or `/fab-clarify intake` to deepen the intake (then the apply-entry requirements regenerate from it) before re-running `/fab-ff`.

### Step 3: Hydrate

*(Skip if `progress.hydrate` is `done`.)*

Dispatch `/fab-continue` as subagent — Hydrate Behavior, change: `{id}` (prompt includes: do NOT run `fab status`; return results only). The subagent validates review passed, hydrates into `docs/memory/`, and returns completion status.

On success: run `fab status finish <change> hydrate fab-ff`.

---

## Output

```
/fab-ff — confidence {score} of 5.0, gate passed.

--- Implementation ---
{apply output (plan generation + task execution)}

--- Review ---
{review output}

--- Hydrate ---
{hydrate output}

Pipeline complete.

Next: {per state table}
```

Resuming shows `(resuming)...` header and `Skipping {stage} — already done.` for completed stages. Bail/failure stops at the relevant stage with `Next:` derived from the state reached per state table in `_preamble.md`.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight fails | Abort with stderr message |
| `intake.md` missing | Abort: "Intake not found. Run /fab-new first." |
| Intake gate fails (confidence < 3.0) | Stop with score and guidance |
| Task fails | Stop: "Task {ID} failed: {reason}. Investigate and re-run /fab-ff." |
| Review fails | Auto-rework loop: 3 cycles (each re-review by fresh sub-agent), escalation after 2 consecutive fix-code. Stops after 3 cycles with summary. Escalation paths: revise plan or revise requirements (both in `plan.md`). |
