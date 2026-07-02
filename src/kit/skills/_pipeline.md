---
name: _pipeline
description: "Shared ff/fff pipeline bracket — intake gate, apply → review → hydrate steps, auto-rework loop with explicit per-cycle choreography (cycle cap from code-review.md Rework Budget, default 3), and the exhaustion stop. Parameterized by driver name and terminal stage. Full bracket used by /fab-ff and /fab-fff; /fab-adopt is a partial consumer (reuses the auto-rework loop + hydrate dispatch, not the full bracket)."
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
> **Partial consumer**: `/fab-adopt` declares `_pipeline` as a helper but does NOT execute the
> full bracket — it runs its own Steps 0–6 and reuses only the § Auto-Rework Loop (for its
> outward-only review verdict) and Step 3's hydrate dispatch. It is not a `{driver}` of the full
> apply → review → hydrate sequence.
>
> - **`{driver}`** — the driver name passed to the `fab status` event commands this bracket
>   shows it on (the fail/recovery commands are deliberately driver-less — see the Behavior
>   note below) and used in re-run guidance: `fab-ff` or `fab-fff`
> - **`{terminal}`** — the bracket's terminal stage: `hydrate` for `/fab-ff` (the pipeline ends
>   after Step 3), or `review-pr` for `/fab-fff` (the fff-only Steps 4–5 — ship and review-pr —
>   live in `fab-fff.md` and run after this bracket's Step 3)
>
> Orchestration that differs between drivers (the fff-only ship/review-pr steps, driver-specific
> Output blocks and error rows) stays in each driver's own file. This partial is the single
> authoritative source for everything the two drivers share.

## Contents

- Pre-flight
- Context Loading
- Behavior
- Shared Error Handling

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

> **Note**: All `.status.yaml` mutations in this bracket use `fab status` event commands (`start`, `advance`, `finish`, `reset`, `fail`, `set-acceptance`) rather than direct file edits. The driver argument is optional in the CLI; this bracket passes `{driver}` wherever a command below shows it (the Resumability `fab status start <change> review` recovery, preserved verbatim from the pre-extraction drivers, passes none).
>
> **Dispatch**: All sub-skill invocations use the Agent tool (`general-purpose` subagent) per `_preamble.md` § Subagent Dispatch. Each subagent reads the target skill file, follows the specified behavior, and returns a structured result to the pipeline. Every `/fab-continue`-behavior subagent prompt MUST include the **block-contract carve-out**: **do NOT run `fab status` transition commands (`start`/`advance`/`finish`/`reset`/`fail`/`skip`); return results only** — the orchestrator runs those stages' transitions (finish/fail/reset) itself — **but the prompt DOES end with a terminal `fab status refresh`** (a pull-based recompute, not a transition, so the orchestrator still owns every transition; see `_preamble.md` § Dispatch-Prompt Obligations). This is the **universal block contract**, not an override this orchestrator imposes: post-intake `/fab-continue` blocks (Apply / Review / Hydrate Behavior) never own their transitions for **any** caller — plain `/fab-continue` is itself a one-stage sequencer that dispatches the block identically and runs the transition after it returns (see `fab-continue.md` Normal Flow Step 1). The orchestrator here is therefore a **pure sequencer**: dispatch block → read returned status/findings → decide proceed / loop / stop; it never reaches into block internals.
>
> **Per-stage model resolution + dispatch adapter** (see `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution for the canonical contract): immediately **before** dispatching each stage's sub-agent, run `fab resolve-agent <stage> --alias` and **surface** the resolved `model=/effort=/spawn=` lines (carry them into the dispatch prompt and/or echo them in this orchestrator's step output, so a skipped or mis-resolved tier — or a CLI dispatch — is visible rather than silent), then **branch on `spawn=` presence**: absent ⇒ **native Agent-tool dispatch** through the two seams — the **model** half via the Agent tool's `model` parameter (the `--alias` flag emits the Agent-tool-valid short alias directly on the `model=` line; empty model ⇒ omit/inherit) and the **effort** half via an explicit imperative instruction in the dispatch prompt — ``Operate at `<effort>` reasoning effort for this task.`` (empty effort ⇒ omit; the Agent tool has no effort param). `spawn=` present ⇒ the **CLI adapter** (`fab dispatch`) per `_preamble.md` § CLI-Adapter Dispatch (start-on-stdin → `sleep 30` poll → five-state handling; the profile rides the `spawn=` command, so the Agent-tool seams do not apply; NO fallback to `agent.spawn_command`; no cleanup after `done`). The Claude Code adapter is the Agent tool's `model` parameter; the resolution itself is provider-neutral. The `review` stage (Step 2) resolves **once** and applies the same profile (native: same model + same effort-prompt instruction to both reviewer sub-agents AND the merge; CLI: one `fab dispatch` for the review block).

### Resumability

Check `progress` from preflight. Skip stages already `done`. If `{terminal}: done`, the pipeline is already complete. If `progress.review` is `failed` (a prior exhaustion stop or an interrupted fail→reset sequence), run `fab status start <change> review` first — the review-specific failed→active transition — then resume from Step 2.

### Step 1: Implementation (apply, with internal plan generation)

*(Skip if `progress.apply` is `done`.)* Since the intake gate already passed in pre-flight, if `progress.intake` is not `done`, finish intake: `fab status finish <change> intake {driver}` (auto-activates apply).

Resolve the apply model + adapter: run `fab resolve-agent apply --alias` (the `--alias` flag emits the Agent-tool-valid short alias on the `model=` line), surface the resolved `model=/effort=/spawn=` (echo them and/or carry them into the dispatch prompt — a skipped resolution or a CLI dispatch is then visible), then **branch on `spawn=`** (per the Behavior dispatch note above + `_preamble.md` § CLI-Adapter Dispatch): absent ⇒ native dispatch (model via the Agent `model` param, empty ⇒ omit/inherit; effort via an imperative prompt instruction ``Operate at `<effort>` reasoning effort for this task.``, empty ⇒ omit); present ⇒ CLI dispatch via `fab dispatch` (the profile rides the `spawn=` command). Dispatch `/fab-continue` as subagent — Apply Behavior, change: `{id}` (prompt carries the block-contract carve-out: no `fab status` transition commands; terminal `fab status refresh`; return results only). The subagent runs both apply sub-steps in a single invocation: (1) Plan Generation — co-generate `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance`) from `intake.md` per **Plan Generation Procedure** (`_generation.md`), unless `plan.md` already exists; (2) Task Execution — parse unchecked tasks under `## Tasks`, execute in dependency order, run tests, mark `[x]` on completion. Returns completion status or failure with task ID and reason.

No `/fab-clarify` runs here. Under-specified requirements are resolved inline by the apply agent as graded SRAD assumptions in `plan.md` `## Assumptions` — not via any clarify ceremony.

**If task fails**: STOP with `Task {ID} failed: {reason}. Investigate and re-run /{driver} <change>.`

On success: run `fab status finish <change> apply {driver}`.

### Step 2: Review

*(Skip if `progress.review` is `done`.)*

Resolve the review model + adapter **once**: run `fab resolve-agent review --alias` (the `--alias` flag emits the Agent-tool-valid short alias on the `model=` line), surface the resolved `model=/effort=/spawn=`, then **branch on `spawn=`** (per the Behavior dispatch note above): absent ⇒ native dispatch — the same model AND the same effort-prompt instruction (``Operate at `<effort>` reasoning effort for this task.``) govern both reviewer sub-agents (inward + outward) and the merge (empty model ⇒ omit/inherit; empty effort ⇒ omit); present ⇒ one CLI dispatch of the review block via `fab dispatch` (the review worker runs the inward/outward/merge itself, degrading to sequential-inline when its harness lacks sub-agent support — `_review.md` § Parallel Dispatch → Nesting degradation). Dispatch `/fab-continue` as subagent — Review Behavior, change: `{id}` (prompt carries the block-contract carve-out: no `fab status` transition commands; terminal `fab status refresh`; return results only — verdict transitions belong to this orchestrator). The subagent reads `_review.md` for review dispatch instructions — both inward and outward sub-agents are defined there. It dispatches both sub-agents in parallel, merges their findings, and returns structured findings (must-fix / should-fix / nice-to-have) with pass/fail status.

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
3. **Re-dispatch apply**: re-run `fab resolve-agent apply --alias` (the `--alias` flag emits the Agent-tool-valid short alias on the `model=` line), surface the resolved `model=/effort=/spawn=`, then **branch on `spawn=`** (native two-seam dispatch when absent; CLI `fab dispatch` when present, per the Behavior dispatch note above), then dispatch `/fab-continue` as a subagent — Apply Behavior, same prompt contract as Step 1 (block-contract carve-out: no `fab status` transition commands; terminal `fab status refresh`; return results only). On success, run `fab status finish <change> apply {driver}` — this auto-activates review (review → `active`), the **one** counted transition that advances `stage_metrics.review.iterations` for this cycle (`status.go:627`). Re-entering review here via `finish apply` (not `reset review`, not any non-`active` path) is what makes the cycle count truthfully; this `finish apply` MUST run every cycle, even when item 2 was a trivial fix.
4. **Fresh re-review**: re-run `fab resolve-agent review --alias` (once, governing both reviewers + merge), surface the resolved `model=/effort=/spawn=`, then **branch on `spawn=`** (native: same model + same effort-prompt instruction to both reviewers and the merge, omitted when empty; CLI: one `fab dispatch` of the review block), then dispatch a **fresh** `/fab-continue` Review Behavior subagent, same prompt contract as Step 2. Never reuse a prior review subagent's context.
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

Resolve the hydrate model + adapter: run `fab resolve-agent hydrate --alias` (the `--alias` flag emits the Agent-tool-valid short alias on the `model=` line), surface the resolved `model=/effort=/spawn=`, then **branch on `spawn=`** (per the Behavior dispatch note above): absent ⇒ native dispatch (model via the Agent `model` param, empty ⇒ omit/inherit; effort via the imperative prompt instruction ``Operate at `<effort>` reasoning effort for this task.``, empty ⇒ omit); present ⇒ CLI dispatch via `fab dispatch`. Dispatch `/fab-continue` as subagent — Hydrate Behavior, change: `{id}` (prompt carries the block-contract carve-out: no `fab status` transition commands; terminal `fab status refresh`; return results only). The subagent validates review passed, hydrates into `docs/memory/`, and returns completion status.

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
