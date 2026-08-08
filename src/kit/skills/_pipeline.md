---
name: _pipeline
description: "Shared ff/fff pipeline bracket — intake gate, apply → review → hydrate steps, auto-rework loop with explicit per-cycle choreography (cycle cap from code-review.md Rework Budget, default 3), and the exhaustion stop. Parameterized by driver name and terminal stage. Full bracket used by /fab-ff and /fab-fff; /fab-adopt is a partial consumer (reuses the auto-rework loop + hydrate dispatch, not the full bracket)."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Shared Pipeline Bracket

> This file defines the shared bracket used by `/fab-ff` and `/fab-fff`; each
> driver binds the parameters in § Driver Framing.
>
> `/fab-adopt` is a partial consumer: it reuses only § Auto-Rework Loop and
> Step 3's hydrate dispatch, not the full bracket.
>
> Driver-specific ship/review-pr steps, output deltas, and errors stay local.

## Contents

- Driver Framing
- Pre-flight
- Context Loading
- Behavior
- Shared Error Handling

---

## Driver Framing

Both drivers run the confidence-gated, resumable apply → review → hydrate
bracket, with bounded auto-rework and no in-bracket `/fab-clarify` invocation.
They accept the same arguments:

- **`<change-name>`** *(optional)* — target a change instead of the active one;
  resolve it per `_preamble.md` § Change-name override.
- **`--force`** *(optional)* — bypass only the intake confidence gate and add
  `(force mode -- gate bypassed)` to the output header.

The driver binds `{driver}` to its command name for status events and re-run
guidance, and `{terminal}` to `hydrate` (`fab-ff`) or `review-pr` (`fab-fff`).
Every output includes the driver header, Implementation, Review, and Hydrate
sections, then `Pipeline complete.` and the state-table `Next:` guidance.
Resumes add `(resuming)` and report each completed stage as skipped; failures
stop at the reached state and derive `Next:` from `_preamble.md`.

```
/{driver} — {confidence header}

--- Implementation ---
{apply output}

--- Review ---
{review output}

--- Hydrate ---
{hydrate output}

Pipeline complete.

Next: {per state table}
```

---

## Pre-flight

1. Run preflight per `_preamble.md` Section 2. Pass `<change-name>` if provided.
2. **Intake prerequisite**: Verify `intake.md` exists. If not, STOP: `Intake not found. Run /fab-new to create the intake first.`
3. **Intake gate** *(skip if `--force`)*: Run `fab score --check-gate --stage intake <change>`. If the gate fails → STOP: `Intake confidence is {score} of 5.0 (need >= 3.0). Run /fab-clarify <change> to resolve, then re-run /{driver} <change>.` (Both commands name the change — the run may be driving a non-active override.)

This intake gate is the **single** confidence gate (flat 3.0 for all change types — see `_preamble.md` § Gate Threshold). There is no spec gate and no review gate; review failures are handled by the bounded auto-rework loop below, not by a gate.

---

## Context Loading

Load per `_preamble.md` Sections 1-3 (config, constitution, intake, memory index, affected memory files, all completed artifacts).

---

## Behavior

All `.status.yaml` mutations in this bracket use the `fab status` event commands shown below rather than direct file edits. The bracket passes `{driver}` wherever a command shows it; the Resumability recovery intentionally passes none.

### Stage Dispatch Procedure

For every apply, review, and hydrate dispatch below:

1. Run `fab resolve-agent <stage> --alias` immediately before dispatch and surface its `model=/effort=/provider=/dispatch=` lines.
2. Run the Dispatch Contract for `<stage>` per `_preamble.md` § CLI-Adapter Dispatch, including its native/CLI branch, recovery budget, pane-mode rules, result handling, reap, and no-state-cleanup rule.
3. In the worker prompt, name the `/fab-continue` Behavior section and carry `_preamble.md` § Dispatch-Prompt Obligations (standard context, result obligation, block-contract carve-out, and terminal refresh).
4. Read the returned result; the bracket remains the pure sequencer and owns every `finish`/`fail`/`reset` transition.
5. For review, resolve once for the single review worker and include `change_type` from `.status.yaml` in the prompt.

### Resumability

Check `progress` from preflight. Skip stages already `done`. If `{terminal}: done`, the pipeline is already complete. If `progress.review` is `failed` (a prior exhaustion stop or an interrupted fail→reset sequence), run `fab status start <change> review` first — the review-specific failed→active transition — then resume from Step 2.

### Step 1: Implementation (apply, with internal plan generation)

*(Skip if `progress.apply` is `done`.)* Since the intake gate already passed in pre-flight, if `progress.intake` is not `done`, finish intake: `fab status finish <change> intake {driver}` (auto-activates apply).

Run the Stage Dispatch Procedure for `apply`, targeting `/fab-continue` Apply Behavior for `{id}`. The worker runs both apply sub-steps in one invocation: (1) Plan Generation — co-generate `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance`) from `intake.md` per `_generation.md` unless `plan.md` exists; (2) Task Execution — execute unchecked `## Tasks` in dependency order, run tests, and mark each `[x]`. It returns completion status or a task ID and reason.

No `/fab-clarify` runs here. Under-specified requirements are resolved inline by the apply agent as graded SRAD assumptions in `plan.md` `## Assumptions` — not via any clarify ceremony.

**If task fails**: STOP with `Task {ID} failed: {reason}. Investigate and re-run /{driver} <change>.`

On success: run `fab status finish <change> apply {driver}`.

### Step 2: Review

*(Skip if `progress.review` is `done`.)*

Run the Stage Dispatch Procedure for `review`, targeting `/fab-continue` Review Behavior for `{id}`. The single worker reads `_review.md`, runs both checklists inline, and returns one structured findings set (must-fix / should-fix / nice-to-have) with a pass/fail verdict.

**Pass**: run `fab status finish <change> review {driver}`. Proceed to Step 3 (Hydrate).

**Fail**: enter the Auto-Rework Loop below.

#### Auto-Rework Loop (up to `{max_cycles}` cycles)

> **`{max_cycles}`** — the rework-cycle cap: the integer from the `Max cycles: {N}` line under `## Rework Budget` in `fab/project/code-review.md` (already loaded via the always-load layer). Default **3** when the file, the section, or the line is absent. Only the cycle cap is configurable — the escalation threshold (2 consecutive fix-code attempts) is fixed.

The agent triages the sub-agent's prioritized findings and autonomously selects the rework path — no user interaction. Must-fix items are always addressed; should-fix items when clear and low-effort; nice-to-have items may be skipped.

> **Cycle-count invariant** (pin against the Go contract — do NOT change `internal/status`). `stage_metrics.review.iterations` is the number `fab pr-meta` renders as "{N} cycle(s)" (`prmeta.go` `reviewCell`). It is incremented by **exactly one event**: a review transition to `state == "active"` (`status.go:627` `Iterations++` fires **only** on `active`). The `reset apply` in item 1 cascades review → `pending`, which the Go layer treats as iterations-**preserving** — it clears only the timing fields and never increments or zeroes the counter (`status.go:646–660`). Therefore the **only** thing that advances the counter is the `finish apply` auto-activation of review at item 3. The choreography below MUST drive **exactly one** review `→ active` re-entry per rework cycle (via item 3's `finish apply`) and MUST NOT re-enter review by any other path, and MUST NOT rely on `reset` to bump or zero the counter. Re-entering review by a non-`active` path (or skipping the `finish apply` after a trivial fix) is the under-count bug: the counter stays at its prior value and `pr-meta` collapses a multi-cycle run to "1 cycle".
>
> **Baseline convention** (the Go regression test is the oracle — `TestStageMetrics_IterationsAccumulateAcrossReworkCycles`): `iterations` counts the **initial** review entry **plus** each rework re-entry — i.e. `iterations` == the total number of review `→ active` transitions. The initial `finish apply` in Step 1 activates review once (`iterations` = 1); each rework cycle's item-3 `finish apply` adds one. So a run with an **initial review attempt + N rework cycles** leaves `iterations == N + 1` and `fab pr-meta` renders "{N+1} cycle(s)". Example: an initial review fail followed by 2 rework cycles (final pass) → `iterations` 3 → "✓ 3 cycles", **never** "✓ 1 cycle".

**Per-cycle choreography** — every cycle runs this exact sequence (a cycle begins in response to a failed review verdict, whether the initial Step 2 review or a later re-review). Each conforming cycle drives **exactly one** counted review `→ active` re-entry (at item 3), so N rework cycles add N to `iterations` per the invariant above:

1. **Status pair**: run `fab status fail <change> review` then `fab status reset <change> apply {driver}`. This fail+reset pair repeats on **every** failed review verdict that starts a new cycle — not just the first failure — so every conforming run leaves the same `.status.yaml` history shape. The `reset apply` cascade drives review → `pending`, which **preserves** `stage_metrics.review.iterations` (timing fields cleared, counter untouched per `status.go:646–660`) — it never advances the counter; only item 3 does.
2. **Triage + rework action**: triage the prioritized findings, select exactly one path per the decision heuristics below, and apply its edits (uncheck tasks / edit `plan.md` / edit `## Requirements`).
3. **Re-dispatch apply**: run the Stage Dispatch Procedure for `apply` with the Step 1 target. On success, run `fab status finish <change> apply {driver}` — this auto-activates review (review → `active`), the **one** counted transition that advances `stage_metrics.review.iterations` for this cycle (`status.go:627`). Re-entering review here via `finish apply` (not `reset review`, not any non-`active` path) is what makes the cycle count truthfully; this `finish apply` MUST run every cycle, even when item 2 was a trivial fix.
4. **Fresh re-review**: run the Stage Dispatch Procedure for `review` with the Step 2 target in a **fresh** worker. Never reuse a prior review worker's context.
5. **Verdict**: pass → run `fab status finish <change> review {driver}` and proceed to Step 3. Fail → if fewer than `{max_cycles}` cycles have run, start the next cycle at item 1 (the fail+reset pair fires again); after the `{max_cycles}`-th failed cycle, stop per **Stop** below.

**Decision heuristics** (applied at item 2 of each cycle — disjoint: each failure description routes to exactly one path):
- **Must-fix: test failures, code that fails a correct requirement, acceptance violations** → "Fix code" — uncheck affected tasks in `plan.md` `## Tasks` with `<!-- rework: reason -->`
- **Must-fix: missing functionality, incomplete coverage, wrong task breakdown** → "Revise plan" — edit `plan.md` (add/modify tasks under `## Tasks` and/or acceptance items under `## Acceptance`)
- **Must-fix: the requirement itself is wrong or has drifted, fundamental approach issues** → "Revise requirements" — edit `plan.md` `## Requirements` plus the downstream `## Tasks`/`## Acceptance` it affects

**Escalation rule**: If the agent chooses "Fix code" and the subsequent sub-agent review fails again on the same or similar issues, the agent MUST escalate to "Revise plan" or "Revise requirements" after **2 consecutive "fix code" attempts**. This is a hard rule — the agent SHALL NOT choose "Fix code" a third time in a row, even if it believes another code fix would work. Non-fix-code actions (revise plan, revise requirements) reset the consecutive counter.

#### Stop (after `{max_cycles}` failed cycles)

After the `{max_cycles}`-th cycle's re-review fails, run `fab status fail <change> review` only — **no reset**. The exact terminal state at exhaustion is `review: failed` (apply remains `done`); this is the resting state `/fab-continue`'s review-failed dispatch handles. Then **STOP** with a per-cycle summary:

```
Review failed after {max_cycles} rework attempts. Summary:
  Cycle 1: {action} — {what was done}
  ...
  Cycle {max_cycles}: {action} — {what was done}
Run /fab-continue <change> for manual rework options.
```

`/fab-continue <change>` will detect the `failed` review state, reset apply, and present the rework menu (fix code / revise plan / revise requirements) directly for the user to choose from. Alternatively, the user can deepen the intake: run `/fab-continue <change> intake` then `/fab-clarify <change>`, and delete `plan.md` (the documented force-regeneration mechanism — it is otherwise preserved on reset) so the apply-entry requirements regenerate from the deepened intake before re-running `/{driver}`. **Name the change in every command here** — this run may have been driving a non-active override, and an argless invocation would resolve the ACTIVE change instead (fab-continue accepts both arguments in any order; fab-clarify accepts a `<change-name>` override; the intake reset regenerates the intake).

### Step 3: Hydrate

*(Skip if `progress.hydrate` is `done`.)*

Run the Stage Dispatch Procedure for `hydrate`, targeting `/fab-continue` Hydrate Behavior for `{id}`. The worker validates review passed, hydrates `docs/memory/`, and returns completion status.

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
| Task fails | Stop: "Task {ID} failed: {reason}. Investigate and re-run /{driver} <change>." |
| Review fails | Auto-rework loop: `{max_cycles}` cycles (default 3), each per the per-cycle choreography (fail+reset pair, one rework action, re-apply, fresh re-review), escalation after 2 consecutive fix-code. After `{max_cycles}` failed cycles: `fail` review (no reset) and stop with summary. Escalation paths: revise plan or revise requirements (both in `plan.md`). |
