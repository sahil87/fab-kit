---
name: _pipeline
description: "Shared ff/fff pipeline bracket — intake gate, inline plan co-gen at apply entry with the one-time light/full lane fork on task count (light lane runs apply/hydrate inline; review stays dispatched in both lanes), apply → review → hydrate steps, auto-rework loop with explicit per-cycle choreography (cycle cap from code-review.md Rework Budget, default 3), and the exhaustion stop. Parameterized by driver name and terminal stage. Full bracket used by /fab-ff and /fab-fff; /fab-adopt is a partial consumer (reuses the auto-rework loop + hydrate dispatch, not the full bracket)."
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
- Light Lane
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
- **`--light` / `--full`** *(optional, mutually exclusive)* — force the lane and
  skip the task-count check at the fork (Step 1). Passing both is a usage error.

The driver binds `{driver}` to its command name for status events and re-run
guidance, `{terminal}` to `hydrate` (`fab-ff`) or `review-pr` (`fab-fff`), and
`{confidence header}` to the exact header line its own § Output defines (the
`--force` suffix above appends to that line).
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

1. Run `fab agent <stage> -o yaml` immediately before dispatch and surface the resolved YAML — at minimum `provider`, `model`, `model_alias`, `effort`, and `dispatch:` presence. Branch only on `dispatch:` key presence: absence means the shared `dispatch.mode` capability ladder resolved native; presence carries the selected pane/headless `command` but is never executed by the skill.
2. Run the Dispatch Contract for `<stage>` per `_preamble.md` § CLI-Adapter Dispatch, including its native/CLI branch, recovery budget, pane-mode rules, result handling, reap, and no-state-cleanup rule.
3. In the worker prompt, name the `/fab-continue` Behavior section and carry `_preamble.md` § Dispatch-Prompt Obligations (standard context, result obligation, block-contract carve-out, and terminal refresh).
4. Read the returned result; the bracket remains the pure sequencer and owns every `finish`/`fail`/`reset` transition.
5. For review, resolve once for the single review worker and include `change_type` from `.status.yaml` in the prompt.

### Resumability

Check `progress` from preflight. Skip stages already `done`. If `{terminal}: done`, the pipeline is already complete. If `progress.review` is `failed` (a prior exhaustion stop or an interrupted fail→reset sequence), run `fab status start <change> review` first — the review-specific failed→active transition — then resume from Step 2.

**Lane re-derivation.** The lane is never persisted — on any resume where Step 1's co-gen is skipped because `plan.md` already exists (including `progress.apply: done`, where Step 1 is skipped entirely), RE-DERIVE the lane deterministically by the same rule: count the task entries in the existing `plan.md` `## Tasks` (all phases) against the ≤ 5 threshold. `--light` / `--full` are per-invocation flags and take precedence when re-passed. Step 3 and the Auto-Rework Loop rely on this re-derived lane.

### Step 1: Implementation (apply — inline plan generation, then the lane fork)

*(Skip if `progress.apply` is `done`.)* Since the intake gate already passed in pre-flight, if `progress.intake` is not `done`, finish intake: `fab status finish <change> intake {driver}` (auto-activates apply).

**Plan generation (inline, both lanes).** Co-generate `plan.md` inline in this orchestrator's own context per `_generation.md`'s Plan Generation Procedure — unless `plan.md` already exists (the resumability seam). **Large-scope guard**: when the intake's affected scope is obviously large, you MAY skip inline co-gen and dispatch apply-with-co-gen exactly as the FULL lane below (the plan-exists seam then never fires) — the criterion is orchestrator judgment; degrading is the shipped path, never a failure.

**Lane fork (one-time).** Immediately after co-gen, count the task entries in `plan.md` `## Tasks` (all phases): **≤ 5 → LIGHT lane**, **> 5 → FULL lane** (threshold hardcoded at 5). `--light` / `--full` (§ Driver Framing) skip the count check and force the lane. The lane is decided once and never revisited mid-run — there is no promotion valve; scope growth discovered during rework rides the rework backstop (§ Auto-Rework Loop); on a resumed run the same rule re-derives the lane from the existing plan (§ Resumability). Light-lane execution rules live in § Light Lane.

**LIGHT lane — task execution inline.** Execute `/fab-continue` Apply Behavior's Task Execution sub-step inline in this orchestrator's own context per § Light Lane (no dispatch, no `fab agent <stage> -o yaml` resolution).

**FULL lane — today's bracket verbatim, plan pre-exists.** Run the Stage Dispatch Procedure for `apply`, targeting `/fab-continue` Apply Behavior for `{id}`. On the native branch, **name the worker `apply-{id}`** per `_preamble.md` § Worker Continuation — that name is what a later rework cycle resumes. On the **pane** branch nothing extra is named: the pane itself is the handle, which is why `_preamble.md` § CLI-Adapter Dispatch step 3 does not reap the apply pane at done-read. The worker receives the finished plan through the plan-exists seam — its Plan Generation sub-step is skipped because `plan.md` already exists — so its invocation is Task Execution only (the dispatch prompt can carry the task list): execute unchecked `## Tasks` in dependency order, run tests, and mark each `[x]`. It returns completion status or a task ID and reason. When the large-scope guard fired, this same dispatch runs apply-with-co-gen exactly as before.

No `/fab-clarify` runs here. Under-specified requirements are resolved inline as graded SRAD assumptions in `plan.md` `## Assumptions` — not via any clarify ceremony.

**If task fails**: STOP with `Task {ID} failed: {reason}. Investigate and re-run /{driver} <change>.`

On success (either lane): run `fab status finish <change> apply {driver}`.

### Light Lane

The lane decision is Step 1's one-time fork; this section owns the light-lane execution-locus rules. v1 is skill-prose only: the lane lives in the orchestrator's context for the run — zero new states, transitions, `.status.yaml` schema fields, or config knobs, and both lanes fire the same `finish`/`fail`/`reset` choreography in the same order (the review cycle-count invariant in § Auto-Rework Loop holds verbatim).

- **Inline**: apply task execution (Step 1) and hydrate (Step 3), plus — for `{terminal} = review-pr` only — ship and review-pr (the driver's Steps 4–5; its Step 3.5 Linear link is inline in BOTH lanes — see `fab-fff.md`). An inline stage runs the same `/fab-continue` Behavior section (or `/git-pr` / `/git-pr-review` behavior) a dispatched worker would, in the orchestrator's own context: no dispatch, no `fab agent <stage> -o yaml` resolution, session model throughout (per `_preamble.md` § Per-Stage Model Resolution, an undispatched stage MAY report the configured profile but MUST NOT switch the session model). Inline ship/review-pr are today's standalone path — those skills keep managing their own stage transitions exactly as standalone. Inline review-pr also removes the yield-seam hazard `fab-fff.md` Step 5's synchronous-poll directive exists to fight: the Copilot poll runs in the main context with no subagent yield risk.
- **Dispatched**: review (Step 2 and every re-review) stays a fresh dispatched worker in BOTH lanes — reviewer independence is the pipeline's highest-value dispatch; author self-review would share the author's blind spots, and a fresh reviewer over a tiny diff is cheap anyway.
- **Rework**: light-lane rework stays inline under the same `{max_cycles}` budget and the same per-cycle fail+reset choreography (§ Auto-Rework Loop item 3 runs the rework inline instead of re-dispatching); exhaustion parks `review: failed` exactly as in the full lane, and a parked light run re-enters however the user chooses, including `--full`. Worker continuation (`_preamble.md` § Worker Continuation) is a FULL-lane-only concern — in the light lane the orchestrator IS the apply author and remembers what the reviewer rejected.

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
3. **Re-dispatch apply (resume-first)** — **FULL lane**: reach the existing apply worker when there is one, by the arm's own mechanism — the native handle `apply-{id}` from **this** orchestrator session, or the still-live apply **pane** delivered into with `fab dispatch deliver <change> apply --prompt-file …`. Both are owned by `_preamble.md` § Worker Continuation (naming, the continuation prompt's content rules, block-contract restatement, profile fixity, and the fallback). Otherwise run the Stage Dispatch Procedure for `apply` with the Step 1 target — today's behavior verbatim, and the mandatory fallback in every unreachable case § Worker Continuation enumerates. `/fab-adopt`'s partial consumption of this loop always runs this FULL branch (adoption has no lane fork). **LIGHT lane**: run the rework inline in this orchestrator's own context per § Light Lane (no dispatch, no continuation — the orchestrator is the apply author). **Either way**, on success run `fab status finish <change> apply {driver}` — this auto-activates review (review → `active`), the **one** counted transition that advances `stage_metrics.review.iterations` for this cycle (`status.go:627`). Re-entering review here via `finish apply` (not `reset review`, not any non-`active` path) is what makes the cycle count truthfully; this `finish apply` MUST run every cycle, even when item 2 was a trivial fix.
4. **Fresh re-review**: run the Stage Dispatch Procedure for `review` with the Step 2 target in a **fresh** worker. Never reuse a prior review worker's context.
5. **Verdict**: pass → run `fab status finish <change> review {driver}` and proceed to Step 3. Fail → if fewer than `{max_cycles}` cycles have run, start the next cycle at item 1 (the fail+reset pair fires again); after the `{max_cycles}`-th failed cycle, stop per **Stop** below.

**Worker release** (FULL lane only — the light lane has no apply worker): once review passes (item 5's `finish review`) or the loop stops at exhaustion, the orchestrator STOPS continuing the apply worker — never message it again. On the **native** arm release is passive: no teardown call exists or is needed, the handle is simply never used. On the **pane** arm this moment is the one `_preamble.md` § CLI-Adapter Dispatch step 3 deferred the apply reap to precisely so the pane could be resumed — take the reap action there, including its arm gate, rather than any restatement here. Hydrate (Step 3) and every later stage always dispatch a fresh worker.

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

**LIGHT lane**: run `/fab-continue` Hydrate Behavior for `{id}` inline per § Light Lane (no dispatch, no `fab agent hydrate -o yaml` resolution). **FULL lane**: run the Stage Dispatch Procedure for `hydrate`, targeting `/fab-continue` Hydrate Behavior for `{id}`. Either way the behavior validates review passed, hydrates `docs/memory/`, and returns completion status.

On success: run `fab status finish <change> hydrate {driver}`.

When `{terminal}` is `hydrate`, the pipeline is complete here. When `{terminal}` is `review-pr`, continue with the driver's own Steps 3.5–5 (`fab-fff.md`).

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
