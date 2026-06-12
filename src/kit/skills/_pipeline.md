---
name: _pipeline
description: "Shared ff/fff pipeline bracket — intake gate, apply → review → hydrate steps, auto-rework loop with explicit per-cycle choreography (cycle cap from code-review.md Rework Budget, default 3), and the exhaustion stop. Parameterized by driver name and terminal stage."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Shared Pipeline Bracket

> This file defines the shared pipeline bracket used by `/fab-ff` and `/fab-fff`. The calling
> skill (the **driver**) declares two parameters before executing this bracket — read them from
> the driver's own file:
>
> - **`{driver}`** — the driver name passed to every `fab status` event command and used in
>   re-run guidance: `fab-ff` or `fab-fff`
> - **`{terminal}`** — the bracket's terminal stage: `hydrate` for `/fab-ff` (the pipeline ends
>   after Step 3), or `review-pr` for `/fab-fff` (the fff-only Steps 4–5 — ship and review-pr —
>   live in `fab-fff.md` and run after this bracket's Step 3)
>
> Orchestration that differs between drivers (the fff-only ship/review-pr steps, driver-specific
> Output blocks and error rows) stays in each driver's own file. This partial is the single
> authoritative source for everything the two drivers share.

---

## Pre-flight

1. Run preflight per `_preamble.md` Section 2. Pass `<change-name>` if provided.
2. **Intake prerequisite**: Verify `intake.md` exists. If not, STOP: `Intake not found. Run /fab-new to create the intake first.`
3. **Intake gate** *(skip if `--force`)*: Run `fab score --check-gate --stage intake <change>`. If the gate fails → STOP: `Intake confidence is {score} of 5.0 (need >= 3.0). Run /fab-clarify to resolve, then retry.`

This intake gate is the **single** confidence gate (flat 3.0 for all change types — see `_preamble.md` § Gate Threshold). There is no spec gate and no review gate; review failures are handled by the bounded auto-rework loop below, not by a gate.

---

## Context Loading

Load per `_preamble.md` Sections 1-3 (config, constitution, intake, memory index, affected memory files, all completed artifacts).

---

## Behavior

> **Note**: All `.status.yaml` mutations in this bracket use `fab status` event commands (`start`, `advance`, `finish`, `reset`, `fail`, `set-acceptance`) rather than direct file edits. The driver argument is optional in the CLI; this bracket passes `{driver}` wherever a command below shows it (the Resumability `fab status start <change> review` recovery, preserved verbatim from the pre-extraction drivers, passes none).
>
> **Dispatch**: All sub-skill invocations use the Agent tool (`general-purpose` subagent) per `_preamble.md` § Subagent Dispatch. Each subagent reads the target skill file, follows the specified behavior, and returns a structured result to the pipeline. Every `/fab-continue`-behavior subagent prompt MUST include: **"do NOT run `fab status` commands; return results only"** — the orchestrator runs those stages' transitions (finish/fail/reset) itself.

### Resumability

Check `progress` from preflight. Skip stages already `done`. If `{terminal}: done`, the pipeline is already complete. If `progress.review` is `failed` (a prior exhaustion stop or an interrupted fail→reset sequence), run `fab status start <change> review` first — the review-specific failed→active transition — then resume from Step 2.

### Step 1: Implementation (apply, with internal plan generation)

*(Skip if `progress.apply` is `done`.)* Since the intake gate already passed in pre-flight, if `progress.intake` is not `done`, finish intake: `fab status finish <change> intake {driver}` (auto-activates apply).

Dispatch `/fab-continue` as subagent — Apply Behavior, change: `{id}` (prompt includes: do NOT run `fab status`; return results only). The subagent runs both apply sub-steps in a single invocation: (1) Plan Generation — co-generate `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance`) from `intake.md` per **Plan Generation Procedure** (`_generation.md`), unless `plan.md` already exists; (2) Task Execution — parse unchecked tasks under `## Tasks`, execute in dependency order, run tests, mark `[x]` on completion. Returns completion status or failure with task ID and reason.

No `/fab-clarify` runs here. Under-specified requirements are resolved inline by the apply agent as graded SRAD assumptions in `plan.md` `## Assumptions` — not via any clarify ceremony.

**If task fails**: STOP with `Task {ID} failed: {reason}. Investigate and re-run /{driver}.`

On success: run `fab status finish <change> apply {driver}`.

### Step 2: Review

*(Skip if `progress.review` is `done`.)*

Dispatch `/fab-continue` as subagent — Review Behavior, change: `{id}` (prompt includes: do NOT run `fab status`; return results only — verdict transitions belong to this orchestrator). The subagent reads `_review.md` for review dispatch instructions — both inward and outward sub-agents are defined there. It dispatches both sub-agents in parallel, merges their findings, and returns structured findings (must-fix / should-fix / nice-to-have) with pass/fail status.

**Pass**: run `fab status finish <change> review {driver}`. Proceed to Step 3 (Hydrate).

**Fail**: enter the Auto-Rework Loop below.

#### Auto-Rework Loop (up to `{max_cycles}` cycles)

> **`{max_cycles}`** — the rework-cycle cap: the integer from the `Max cycles: {N}` line under `## Rework Budget` in `fab/project/code-review.md` (already loaded via the always-load layer). Default **3** when the file, the section, or the line is absent. Only the cycle cap is configurable — the escalation threshold (2 consecutive fix-code attempts) is fixed.

The agent triages the sub-agent's prioritized findings and autonomously selects the rework path — no user interaction. Must-fix items are always addressed; should-fix items when clear and low-effort; nice-to-have items may be skipped.

**Per-cycle choreography** — every cycle runs this exact sequence (a cycle begins in response to a failed review verdict, whether the initial Step 2 review or a later re-review):

1. **Status pair**: run `fab status fail <change> review` then `fab status reset <change> apply {driver}`. This fail+reset pair repeats on **every** failed review verdict that starts a new cycle — not just the first failure — so every conforming run leaves the same `.status.yaml` history shape.
2. **Triage + rework action**: triage the prioritized findings, select exactly one path per the decision heuristics below, and apply its edits (uncheck tasks / edit `plan.md` / edit `## Requirements`).
3. **Re-dispatch apply**: dispatch `/fab-continue` as a subagent — Apply Behavior, same prompt contract as Step 1 (do NOT run `fab status`; return results only). On success, run `fab status finish <change> apply {driver}` (auto-activates review).
4. **Fresh re-review**: dispatch a **fresh** `/fab-continue` Review Behavior subagent, same prompt contract as Step 2. Never reuse a prior review subagent's context.
5. **Verdict**: pass → run `fab status finish <change> review {driver}` and proceed to Step 3. Fail → if fewer than `{max_cycles}` cycles have run, start the next cycle at item 1 (the fail+reset pair fires again); after the `{max_cycles}`-th failed cycle, stop per **Stop** below.

**Decision heuristics** (applied at item 2 of each cycle):
- **Must-fix: test failures, requirements mismatches, acceptance violations** → "Fix code" — uncheck affected tasks in `plan.md` `## Tasks` with `<!-- rework: reason -->`
- **Must-fix: missing functionality, incomplete coverage, wrong task breakdown** → "Revise plan" — edit `plan.md` (add/modify tasks under `## Tasks` and/or acceptance items under `## Acceptance`)
- **Must-fix: requirements drift, requirements mismatch, fundamental approach issues** → "Revise requirements" — edit `plan.md` `## Requirements` plus the downstream `## Tasks`/`## Acceptance` it affects

**Escalation rule**: If the agent chooses "Fix code" and the subsequent sub-agent review fails again on the same or similar issues, the agent MUST escalate to "Revise plan" or "Revise requirements" after **2 consecutive "fix code" attempts**. This is a hard rule — the agent SHALL NOT choose "Fix code" a third time in a row, even if it believes another code fix would work. Non-fix-code actions (revise plan, revise requirements) reset the consecutive counter.

#### Stop (after `{max_cycles}` failed cycles)

After the `{max_cycles}`-th cycle's re-review fails, run `fab status fail <change> review` only — **no reset**. The exact terminal state at exhaustion is `review: failed` (apply remains `done`); this is the resting state `/fab-continue`'s review-failed dispatch handles. Then **STOP** with a per-cycle summary:

```
Review failed after {max_cycles} rework attempts. Summary:
  Cycle 1: {action} — {what was done}
  ...
  Cycle {max_cycles}: {action} — {what was done}
Run /fab-continue for manual rework options.
```

`/fab-continue` will detect the `failed` review state, reset apply, and present the rework menu (fix code / revise plan / revise requirements) directly for the user to choose from. Alternatively, the user can run `/fab-clarify intake` to deepen the intake (the apply-entry requirements regenerate from it) before re-running `/{driver}`.

### Step 3: Hydrate

*(Skip if `progress.hydrate` is `done`.)*

Dispatch `/fab-continue` as subagent — Hydrate Behavior, change: `{id}` (prompt includes: do NOT run `fab status`; return results only). The subagent validates review passed, hydrates into `docs/memory/`, and returns completion status.

On success: run `fab status finish <change> hydrate {driver}`.

When `{terminal}` is `hydrate`, the pipeline is complete here. When `{terminal}` is `review-pr`, continue with the driver's own Steps 4–5 (`fab-fff.md`).

---

## Shared Error Handling

These rows apply to both drivers; each driver's own file adds any driver-specific rows.

| Condition | Action |
|-----------|--------|
| Preflight fails | Abort with stderr message |
| `intake.md` missing | Abort: "Intake not found. Run /fab-new first." |
| Intake gate fails (confidence < 3.0) | Stop with score and guidance |
| Task fails | Stop: "Task {ID} failed: {reason}. Investigate and re-run /{driver}." |
| Review fails | Auto-rework loop: `{max_cycles}` cycles (default 3), each per the per-cycle choreography (fail+reset pair, one rework action, re-apply, fresh re-review), escalation after 2 consecutive fix-code. After `{max_cycles}` failed cycles: `fail` review (no reset) and stop with summary. Escalation paths: revise plan or revise requirements (both in `plan.md`). |
