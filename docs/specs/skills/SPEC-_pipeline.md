# _pipeline

## Summary

Shared pipeline bracket executed by `/fab-ff` and `/fab-fff` (added in 260611-szxd — f007/f071). `/fab-adopt` (260630-t54n) is a **partial consumer** — it declares `_pipeline` as a helper but runs its own Steps 0–6, reusing only the auto-rework loop (for its outward-only review verdict) and the hydrate dispatch, not the full apply → review → hydrate bracket; it is not a `{driver}` of the full sequence. The bracket is a **pure sequencer** (affirmed 260613-fgxx): dispatch block → read returned status/findings → decide proceed / loop / stop; it owns the `fab status` transitions and `fab resolve-agent <stage>` resolution and never reaches into block internals. The subagent dispatch note's "do NOT run `fab status`; return results only" instruction is the **universal block contract** for post-intake `/fab-continue` blocks (Apply/Review/Hydrate) — not an override this orchestrator imposes — because plain `/fab-continue` is itself a one-stage sequencer that dispatches those blocks identically and runs their transitions after they return (see `SPEC-fab-continue.md`). Single authoritative source for everything the two drivers share: pre-flight (intake prerequisite + the single intake confidence gate, flat 3.0), context loading, the subagent dispatch note ("do NOT run `fab status`; return results only"), resumability (skip `done` stages; review-`failed` recovery via `fab status start <change> review`), Steps 1–3 (apply with internal plan generation → review → hydrate — each preceded by `fab resolve-agent <stage> --alias` per-stage model resolution since 260613-l3ja (`--alias` since 260613-yky7 — emits the Agent-tool-valid short alias on the `model=` line), and since 260613-m3d4 each site **surfaces** the resolved `model=/effort=` (visibility — a skipped resolution is then detectable, not silent) and dispatches via two seams: model on the Agent `model` param, effort as an imperative instruction in the dispatch prompt (no Agent effort param; omitted when empty); review resolves once and applies the same model + same effort-prompt instruction to both reviewers + merge; rework items 3/4 re-resolve, re-surface, and re-inject before re-dispatch), the auto-rework loop with its **explicit per-cycle choreography**, the exhaustion stop, and the shared error rows. All user-facing stop/re-run guidance in the bracket (gate-fail, task-fail, exhaustion stop) names the change in the suggested commands — the run may be driving a non-active override, and argless re-runs would resolve the active change (260612-w7dp).

**Parameters** (declared by each driver's own file):

| Parameter | `/fab-ff` | `/fab-fff` |
|-----------|-----------|------------|
| `{driver}` — passed to the `fab status` event commands the bracket shows it on (the fail/recovery commands are deliberately driver-less — 260612-w7dp scoped the wrappers' claim) and used in re-run guidance | `fab-ff` | `fab-fff` |
| `{terminal}` — the bracket's terminal stage | `hydrate` (pipeline ends after Step 3) | `review-pr` (fff-only Steps 4–5 live in `fab-fff.md`) |

A third value, **`{max_cycles}`**, is defined by the bracket itself (260612-c5tr — the formerly dead "Max cycles" knob is wired): the integer from the `Max cycles: {N}` line under `## Rework Budget` in `fab/project/code-review.md` (always-load layer), defaulting to **3** when the file, section, or line is absent. Only the cycle cap is configurable; the escalation threshold (2 consecutive fix-code) stays fixed.

This is an internal partial (`user-invocable: false`) — never invoked directly. Drivers declare it via `helpers: [_pipeline]` frontmatter.

**Prose optimization** (260620-skop): a `## Contents` TOC added to `_pipeline.md` (structural check, file >100 lines); no prose trimmed and no behavioral change (Per-Cycle Rework Choreography / Flow unchanged).

## Per-Cycle Rework Choreography (f071)

Stated exactly once, in the Auto-Rework Loop. Every cycle (the initial Step 2 failure and each later re-review failure alike):

1. **Status pair**: `fab status fail <change> review` then `fab status reset <change> apply {driver}` — repeats on **every** failed verdict that starts a cycle, so conforming runs leave identical `.status.yaml` histories (`stage_metrics.review.iterations` feeds PR meta). The `reset apply` cascade drives review → `pending`, which **preserves** `iterations` (timing fields cleared, counter untouched per `status.go:646–660`); it never advances the counter — only item 3 does.
2. **Triage + one rework action**: fix code / revise plan / revise requirements per the decision heuristics (disjoint since 260612-w7dp: code-fails-a-correct-requirement → fix code; the-requirement-itself-wrong-or-drifted → revise requirements — each failure description appears exactly once)
3. **Re-dispatch apply**: fresh `/fab-continue` Apply Behavior subagent (no-`fab status` prompt contract); on success `fab status finish <change> apply {driver}` — the auto-activation of review here (review → `active`) is the **one** counted transition that advances `iterations` for this cycle (`status.go:627`, `Iterations++` only on `active`). It MUST run every cycle, even after a trivial fix.
4. **Fresh re-review**: a new `/fab-continue` Review Behavior subagent — never reuse a prior review subagent's context
5. **Verdict**: pass → `finish review {driver}`, proceed to Step 3; fail → next cycle at item 1, or stop after the `{max_cycles}`-th failed cycle

**Cycle-count invariant (260615-qg64)**: `stage_metrics.review.iterations` is the number `fab pr-meta` renders as "{N} cycle(s)" (`prmeta.go` `reviewCell`). It is advanced by **exactly one** event — a review transition to `state == "active"` (`status.go:627`) — so the choreography MUST drive **exactly one** review `→ active` re-entry per rework cycle, via item 3's `finish apply` auto-activation, and MUST NOT re-enter review by any non-`active` path nor rely on `reset` to bump/zero the counter (the `reset` cascade preserves-without-incrementing, `status.go:646–660`). Re-entering review by a non-counting path (or skipping the per-cycle `finish apply`) is the under-count bug that collapses a multi-cycle run to "1 cycle". **Baseline convention** (the Go regression test `TestStageMetrics_IterationsAccumulateAcrossReworkCycles` is the oracle): `iterations` counts the **initial** review entry **plus** each rework re-entry — i.e. the total number of review `→ active` transitions. The Step-1 `finish apply` activates review once (iterations = 1); each rework cycle adds one. So an **initial review attempt + N rework cycles** leaves `iterations == N + 1`, rendered "{N+1} cycle(s)" — e.g. an initial fail + 2 rework cycles (final pass) → iterations 3 → "✓ 3 cycles", never "✓ 1 cycle". This is a **choreography property, not a state-machine one**: the Go layer (`internal/status`, `internal/prmeta`) is correct as-is and is NOT changed — the fix lives in this prose.

**Exhaustion terminal state** (f019/f071): after the `{max_cycles}`-th cycle's re-review fails, run `fab status fail <change> review` only — **no reset**. Terminal state is `review: failed` (apply `done`), the resting state `/fab-continue`'s review-failed dispatch row handles (reset apply + rework menu). The stop message tells the user exactly that, plus the executable intake-deepening alternative (260612-w7dp): `/fab-continue <change> intake`, then `/fab-clarify <change>`, then delete `plan.md` so the apply-entry requirements regenerate — replacing the unexecutable `/fab-clarify intake` pointer. Every command in the stop guidance (including the rework-menu line `Run /fab-continue <change> for manual rework options.`) names the change: the run may have been driving a non-active override, and an argless invocation would resolve the ACTIVE change — refining the wrong intake or tripping clarify's stage guard. (fab-continue's own internal recovery messages stay argless — the active change is implied there.)

**Escalation rule**: after 2 consecutive "fix code" attempts, MUST escalate to "Revise plan" or "Revise requirements"; non-fix-code actions reset the counter.

## Flow

```
Driver (fab-ff / fab-fff) reads _pipeline.md with {driver}/{terminal} bound
│
├─ Pre-flight
│  ├─ Bash: fab preflight [change-name]
│  ├─ Verify intake.md exists (STOP if missing)
│  └─ Gate (skip if --force): fab score --check-gate --stage intake <change>
│     └─ STOP if < 3.0 (the single confidence gate — no spec/review gate)
│
├─ Resumability: skip done stages; {terminal}: done → already complete;
│  review failed → fab status start <change> review, resume from Step 2
│
│  (each stage dispatch first runs `fab resolve-agent <stage> --alias`, surfaces
│   the resolved model=/effort= (visibility — 260613-m3d4), then dispatches via two
│   seams: model → Agent `model` param (empty ⇒ omit/inherit), effort → imperative
│   instruction in the dispatch prompt (no Agent effort param; empty ⇒ omit;
│   260613-m3d4) — 260613-l3ja established the resolve call, 260613-yky7 added
│   `--alias` (emits the Agent-tool-valid short alias on the model= line); see
│   _preamble.md § Subagent Dispatch → Per-Stage Model Resolution)
│
├─ Step 1: Apply (fab resolve-agent apply --alias → subagent: /fab-continue Apply Behavior — plan co-gen + tasks)
│  ├─ fab status finish <change> intake {driver}  (if intake not done)
│  └─ fab status finish <change> apply {driver}   (on success)
│
├─ Step 2: Review (fab resolve-agent review --alias ONCE for both reviewers + merge →
│  │        subagent: /fab-continue Review Behavior → _review.md
│  │        inward + outward dispatch, merged findings, pass/fail)
│  ├─ Pass: fab status finish <change> review {driver} → Step 3
│  └─ Fail: Auto-Rework Loop (≤{max_cycles} cycles — code-review.md
│     Rework Budget knob, default 3; per-cycle choreography above;
│     items 3/4 re-resolve apply/review before re-dispatch)
│     └─ Exhaustion: fab status fail <change> review (no reset) → STOP
│
├─ Step 3: Hydrate (fab resolve-agent hydrate --alias → subagent: /fab-continue Hydrate Behavior)
│  └─ fab status finish <change> hydrate {driver}
│
└─ {terminal} = hydrate → complete; {terminal} = review-pr → driver Steps 4–5
```

### Sub-agents

| Agent | Step | Purpose |
|-------|------|---------|
| /fab-continue (Apply) | 1, rework item 3 | Plan co-generation + task execution; no `fab status` calls |
| /fab-continue (Review) | 2, rework item 4 | Reads `_review.md`; dispatches inward + outward sub-agents in parallel; merges findings |
| /fab-continue (Hydrate) | 3 | Memory hydration; no `fab status` calls |

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| pre | `fab score --check-gate --stage intake` | Before the bracket (intake gate) |
| 1 | PostToolUse hook recomputes plan counts; sets `plan.generated=true` | After plan.md write/edit |
| rework | `fail` + `reset` pair, `finish apply`, `finish review` | Per cycle, per the choreography |
