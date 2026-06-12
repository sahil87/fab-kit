# _pipeline

## Summary

Shared pipeline bracket executed by `/fab-ff` and `/fab-fff` (added in 260611-szxd — f007/f071). Single authoritative source for everything the two drivers share: pre-flight (intake prerequisite + the single intake confidence gate, flat 3.0), context loading, the subagent dispatch note ("do NOT run `fab status`; return results only"), resumability (skip `done` stages; review-`failed` recovery via `fab status start <change> review`), Steps 1–3 (apply with internal plan generation → review → hydrate), the auto-rework loop with its **explicit per-cycle choreography**, the exhaustion stop, and the shared error rows.

**Parameters** (declared by each driver's own file):

| Parameter | `/fab-ff` | `/fab-fff` |
|-----------|-----------|------------|
| `{driver}` — passed to every `fab status` event command and used in re-run guidance | `fab-ff` | `fab-fff` |
| `{terminal}` — the bracket's terminal stage | `hydrate` (pipeline ends after Step 3) | `review-pr` (fff-only Steps 4–5 live in `fab-fff.md`) |

This is an internal partial (`user-invocable: false`) — never invoked directly. Drivers declare it via `helpers: [_pipeline]` frontmatter.

## Per-Cycle Rework Choreography (f071)

Stated exactly once, in the Auto-Rework Loop. Every cycle (the initial Step 2 failure and each later re-review failure alike):

1. **Status pair**: `fab status fail <change> review` then `fab status reset <change> apply {driver}` — repeats on **every** failed verdict that starts a cycle, so conforming runs leave identical `.status.yaml` histories (`stage_metrics.review.iterations` feeds PR meta)
2. **Triage + one rework action**: fix code / revise plan / revise requirements per the decision heuristics
3. **Re-dispatch apply**: fresh `/fab-continue` Apply Behavior subagent (no-`fab status` prompt contract); on success `fab status finish <change> apply {driver}`
4. **Fresh re-review**: a new `/fab-continue` Review Behavior subagent — never reuse a prior review subagent's context
5. **Verdict**: pass → `finish review {driver}`, proceed to Step 3; fail → next cycle at item 1, or stop after the 3rd failed cycle

**Exhaustion terminal state** (f019/f071): after the 3rd cycle's re-review fails, run `fab status fail <change> review` only — **no reset**. Terminal state is `review: failed` (apply `done`), the resting state `/fab-continue`'s review-failed dispatch row handles (reset apply + rework menu). The stop message tells the user exactly that, plus the `/fab-clarify intake` alternative.

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
├─ Step 1: Apply (subagent: /fab-continue Apply Behavior — plan co-gen + tasks)
│  ├─ fab status finish <change> intake {driver}  (if intake not done)
│  └─ fab status finish <change> apply {driver}   (on success)
│
├─ Step 2: Review (subagent: /fab-continue Review Behavior → _review.md
│  │        inward + outward dispatch, merged findings, pass/fail)
│  ├─ Pass: fab status finish <change> review {driver} → Step 3
│  └─ Fail: Auto-Rework Loop (≤3 cycles, per-cycle choreography above)
│     └─ Exhaustion: fab status fail <change> review (no reset) → STOP
│
├─ Step 3: Hydrate (subagent: /fab-continue Hydrate Behavior)
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
