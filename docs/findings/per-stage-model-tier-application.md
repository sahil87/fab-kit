# Finding: Per-stage model tiers are honored only on the subagent-dispatch seam

**Date**: 2026-06-13
**Area**: subagent dispatch, per-stage model resolution (`fab resolve-agent`)
**Status**: open
**Severity**: medium — correctness-of-execution gap (stages may silently run on the wrong model/effort), not a crash

---

## Summary

The fab pipeline defines a per-stage model tier mapping (`docs/specs/stage-models.md`):

| Tier | Stages | Default profile |
|------|--------|-----------------|
| `thinking` | intake, review | `opus + xhigh` |
| `doing` | apply, review-pr, hydrate | `opus + high` |
| `fast` | ship | `sonnet + low` |

`fab resolve-agent <stage>` ships and resolves these correctly (verified 2026-06-13):

```
$ fab resolve-agent apply   →  model=claude-opus-4-8  effort=high
$ fab resolve-agent review  →  model=claude-opus-4-8  effort=xhigh
$ fab resolve-agent ship    →  model=claude-sonnet-4-6 effort=low
```

But the resolved profile is **only ever applied at the Agent-tool dispatch seam**. This produces two distinct gaps — one a *compliance* gap (orchestrator can skip the resolution step), one an *architectural* gap (the harness adapter has no effort knob).

---

## Gap 1 — Compliance: foreground stages and skipped resolution inherit the session model

### 1a. Foreground stages can't be tiered at all (by design, but easy to forget)

`_preamble.md` § Per-Stage Model Resolution and `fab-continue.md`'s header note both state that per-stage selection is honored **only on orchestrated/sub-agent dispatch**. When a stage runs **directly in the foreground** — i.e. plain `/fab-continue` for apply or hydrate — `fab` cannot switch the session model mid-run, so the configured tier is **advisory only**. The skill "MAY note 'this stage is configured for X; you're on Y' but MUST NOT attempt to switch."

Net effect, by path:

| Stage | resolve-agent tier | Path A: `/fab-continue` (manual) | Paths B/C/D: `/fab-ff`, `/fab-fff`, `/fab-proceed` |
|-------|--------------------|----------------------------------|-----------------------------------------------------|
| apply | opus + high | foreground → **session model** (advisory) | subagent → tier applied |
| review | opus + xhigh | subagents → tier applied | subagents → tier applied |
| hydrate | opus + high | foreground → **session model** (advisory) | subagent → tier applied |
| ship | sonnet + low | foreground → session model | subagent → tier applied |
| review-pr | opus + high | foreground → session model | subagent → tier applied |

(In Path A, `review` is the lone exception: `_review.md` mandates inward+outward **subagents**, so the tier *is* applied there even in the manual path.)

This is intended behavior — but it means "did apply run on opus+high?" has a different answer depending on whether the user ran `/fab-continue` or an orchestrator. Worth stating plainly in user-facing docs.

> **Closed by [intake-is-the-context-boundary](intake-is-the-context-boundary.md)**: if every post-intake stage dispatches a subagent regardless of A/B/C/D (the principle in that finding), there is no foreground execution path left to be the exception — `fab resolve-agent` applies uniformly and Gap 1a disappears. Only Gap 2 (below) survives. The two findings should be planned together.

### 1b. Orchestrators can silently skip the mandated `resolve-agent` call

`_pipeline.md` Steps 1/2/3 (and `fab-fff.md` Steps 4/5) each mandate:

> immediately **before** dispatching each stage's sub-agent, run `fab resolve-agent <stage>` and pass the resolved model AND effort into the Agent dispatch.

If an orchestrator agent **omits** that call and dispatches with no `model` param, the subagent inherits the **session** model/effort instead of being pinned to the stage tier. Nothing enforces the call — it's a prose instruction, not a code-level guard. Observed in the wild: an orchestrated run dispatched apply/review subagents with no model override and the subagents ran on the inherited session profile rather than `opus+high` / `opus+xhigh`. The agent's self-diagnosis correctly identified the symptom ("subagents ran on inherited session settings"), but mis-attributed cause to "in-process Agent dispatch doesn't route through resolve-agent" — the routing *is* defined; the run simply didn't execute the step.

---

## Gap 2 — Architectural: the Claude Code Agent tool exposes `model` but no `effort`

Even a fully compliant orchestrator that calls `fab resolve-agent apply` and gets `model=claude-opus-4-8 effort=high` hits a wall: the **Claude Code Agent tool's parameters are `subagent_type`, `model`, `isolation`, `agentType`, … — there is no per-subagent `effort` parameter.**

`_preamble.md` names the Agent tool's `model` param as *the* Claude-Code harness adapter for injecting the resolved model. It is silent on effort because there is no seam for it. So:

- The **model** half of the tier can be pinned per-subagent. ✅
- The **effort** half (`xhigh` vs `high` vs `low`) **cannot** be injected per-subagent through the current Agent-tool adapter. ❌ The subagent runs at whatever effort the session/harness governs.

This is a real harness-adapter limitation, not a compliance miss. It means the `effort=` line `fab resolve-agent` emits is currently **unconsumable** on the per-subagent dispatch path in Claude Code, regardless of orchestrator correctness.

---

## Why it matters

The whole point of `doing` vs `thinking` tiers is cost/quality calibration — apply on `high`, review on `xhigh`. If effort can't be pinned per-subagent (Gap 2) and the model isn't pinned when the call is skipped (Gap 1b), stages silently run at the session's effort. A session driven at `xhigh` over-spends on apply; a session at a low effort under-thinks review. The tier mapping looks authoritative in config but is only partially enforced at runtime.

---

## Suggested directions (not yet decided)

1. **Close Gap 1b with a self-check.** Have orchestrators emit the resolved `model=/effort=` lines into the dispatch prompt and/or log them, so a skipped resolution is visible in output rather than silent. Cheap, prose-level.
2. **Close Gap 2's effort half via the prompt.** Since the Agent tool can't take an effort param, inject the resolved effort into the **subagent prompt** as an explicit instruction ("operate at `xhigh` reasoning effort"), so the subagent self-selects. Imperfect (relies on the subagent honoring it) but it's the only seam available today.
3. **Document the foreground-advisory reality (Gap 1a)** in user-facing docs so users know manual `/fab-continue` does not tier apply/hydrate/ship.
4. **Harness ask**: a per-subagent `effort` parameter on the Agent tool would make Gap 2 closable cleanly. Out of fab's control — flag upstream.

---

## Evidence

- `fab resolve-agent {apply,review,ship}` outputs above — captured 2026-06-13 from `fab-kit` v2.3.1.
- `.claude/skills/_preamble/SKILL.md` § Subagent Dispatch → Per-Stage Model Resolution (the "foreground is advisory-only" + "Claude Code adapter is the Agent tool `model` parameter" contract).
- `.claude/skills/_pipeline/SKILL.md` Steps 1–3 (the mandated pre-dispatch `fab resolve-agent` call).
- `.claude/skills/fab-continue/SKILL.md` header note (foreground = advisory-only).
- `docs/specs/stage-models.md` (tier definitions and the provider-neutral / harness-adapter boundary).
