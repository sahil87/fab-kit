# Finding: Per-stage model tiers are honored only on the subagent-dispatch seam

**Date**: 2026-06-13
**Area**: subagent dispatch, per-stage model resolution (`fab resolve-agent`)
**Status**: largely addressed — Gap 1a closed by `260613-fgxx` (intake-is-the-context-boundary); Gap 1b (compliance visibility) and Gap 2's effort half (effort injected via the subagent prompt) closed by `260613-m3d4` (uniform-stage-model-tier). **Residual**: a first-class per-sub-agent `effort` parameter on the Claude Code Agent tool (§ Suggested directions item 4) — a harness ask outside fab's control.
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

The resolved profile is applied at the Agent-tool dispatch seam. This finding originally identified two distinct gaps — one a *compliance* gap (orchestrator can skip the resolution step), one an *architectural* gap (the harness adapter has no effort knob). **All three sub-gaps below are now closed or addressed** (`260613-fgxx` closed Gap 1a; `260613-m3d4` addressed Gap 1b + Gap 2's effort half via visibility + prompt-injection); the lone residual is the harness ask in § Suggested directions item 4. Sections below are retained with their resolution annotations for the historical record.

---

## Gap 1 — Compliance: foreground stages and skipped resolution inherit the session model

### 1a. Foreground stages can't be tiered at all — CLOSED by `260613-fgxx`

**Historical (pre-`260613-fgxx`)**: `_preamble.md` § Per-Stage Model Resolution and `fab-continue.md`'s header note both stated that per-stage selection was honored **only on orchestrated/sub-agent dispatch**. When a stage ran **directly in the foreground** — i.e. plain `/fab-continue` for apply or hydrate — `fab` could not switch the session model mid-run, so the configured tier was **advisory only**.

Net effect *at the time*, by path:

| Stage | resolve-agent tier | Path A: `/fab-continue` (then-foreground) | Paths B/C/D: `/fab-ff`, `/fab-fff`, `/fab-proceed` |
|-------|--------------------|----------------------------------|-----------------------------------------------------|
| apply | opus + high | foreground → **session model** (advisory) | subagent → tier applied |
| review | opus + xhigh | subagents → tier applied | subagents → tier applied |
| hydrate | opus + high | foreground → **session model** (advisory) | subagent → tier applied |
| ship | sonnet + low | foreground → session model | subagent → tier applied |
| review-pr | opus + high | foreground → session model | subagent → tier applied |

> **CLOSED by [intake-is-the-context-boundary](intake-is-the-context-boundary.md), landed as `260613-fgxx`.** That change collapsed the post-intake dual execution mode: **every** post-intake stage now dispatches a sub-agent regardless of A/B/C/D — plain `/fab-continue` is a one-stage sequencer that resolves `fab resolve-agent <stage>` and dispatches its stage's block just like an orchestrator. With **no foreground execution path left**, `fab resolve-agent` applies uniformly across apply/review/hydrate and Gap 1a is gone — the per-path split above no longer exists (every cell is "subagent → tier applied"). The only residual "advisory" case is a stage skill genuinely run with no dispatch at all, which `fab` cannot switch mid-run by design. This finding's Gap 1b and Gap 2-effort were then closed by `260613-m3d4` (below).

### 1b. Orchestrators can silently skip the mandated `resolve-agent` call — ADDRESSED by `260613-m3d4`

`_pipeline.md` Steps 1/2/3 (and `fab-fff.md` Steps 4/5) each mandated (the original Gap-1b prose, since refined by `260613-m3d4` — see below):

> immediately **before** dispatching each stage's sub-agent, run `fab resolve-agent <stage>` and pass the resolved model AND effort into the Agent dispatch.

If an orchestrator agent **omits** that call and dispatches with no `model` param, the subagent inherits the **session** model/effort instead of being pinned to the stage tier. Nothing enforces the call — it's a prose instruction, not a code-level guard. Observed in the wild: an orchestrated run dispatched apply/review subagents with no model override and the subagents ran on the inherited session profile rather than `opus+high` / `opus+xhigh`. The agent's self-diagnosis correctly identified the symptom ("subagents ran on inherited session settings"), but mis-attributed cause to "in-process Agent dispatch doesn't route through resolve-agent" — the routing *is* defined; the run simply didn't execute the step.

> **ADDRESSED by `260613-m3d4` (compliance visibility).** A true code-level guard is impossible (dispatch is harness-internal — `fab` cannot observe Agent-tool calls). The available seam is **visibility**: every per-stage dispatch site now **surfaces** the resolved `model=/effort=` lines (carried into the dispatch prompt and/or echoed in the orchestrator's step output), so a *skipped* `fab resolve-agent` call — where the sub-agent silently inherits the session profile — is **visible in output rather than silent**. The canonical contract (`_preamble.md` § Per-Stage Model Resolution) also notes that an all-empty resolution is worth surfacing/asserting rather than dispatching blind. This does not *prevent* a skip (no enforcement seam exists) but makes one *detectable* — the cheap, prose-level fix the finding called for.

---

## Gap 2 — Architectural: the Claude Code Agent tool exposes `model` but no `effort`

Even a fully compliant orchestrator that calls `fab resolve-agent apply` and gets `model=claude-opus-4-8 effort=high` hits a wall: the **Claude Code Agent tool's parameters are `subagent_type`, `model`, `isolation`, `agentType`, … — there is no per-subagent `effort` parameter.**

`_preamble.md` names the Agent tool's `model` param as *the* Claude-Code harness adapter for injecting the resolved model. It is silent on effort because there is no seam for it. So:

- The **model** half of the tier can be pinned per-subagent. ✅
- The **effort** half (`xhigh` vs `high` vs `low`) **cannot** be injected per-subagent through the current Agent-tool adapter. ❌ The subagent runs at whatever effort the session/harness governs.

This is a real harness-adapter limitation, not a compliance miss. It means the `effort=` line `fab resolve-agent` emits is **unconsumable through a dispatch *parameter*** in Claude Code, regardless of orchestrator correctness.

> **Effort half ADDRESSED by `260613-m3d4` (effort-via-prompt); the clean fix remains the residual.** Since the Agent tool has no `effort` param, `260613-m3d4` injects the resolved effort into the **subagent prompt** as an explicit imperative instruction (e.g., ``Operate at `xhigh` reasoning effort for this task.``; omitted when the resolved effort is empty) at every per-stage dispatch site, so the sub-agent self-selects its reasoning effort. The model half stays on the Agent tool's `model` param (unchanged). This is **imperfect** — it relies on the sub-agent honoring the instruction rather than the harness enforcing it — but it is the only per-sub-agent effort seam available today. The clean fix — a first-class per-sub-agent `effort` parameter on the Agent tool (§ Suggested directions item 4) — is **out of fab's control** and remains the **residual** after this change.

---

## Why it matters

The whole point of `doing` vs `thinking` tiers is cost/quality calibration — apply on `high`, review on `xhigh`. If effort can't be pinned per-subagent (Gap 2) and the model isn't pinned when the call is skipped (Gap 1b), stages silently run at the session's effort. A session driven at `xhigh` over-spends on apply; a session at a low effort under-thinks review. The tier mapping looks authoritative in config but is only partially enforced at runtime.

---

## Suggested directions

1. **Close Gap 1b with a self-check.** ✅ **Done (`260613-m3d4`).** Orchestrators emit/surface the resolved `model=/effort=` lines (into the dispatch prompt and/or step output), so a skipped resolution is visible in output rather than silent. Cheap, prose-level — as proposed.
2. **Close Gap 2's effort half via the prompt.** ✅ **Done (`260613-m3d4`).** Since the Agent tool can't take an effort param, the resolved effort is injected into the **subagent prompt** as an explicit instruction (``Operate at `xhigh` reasoning effort for this task.``; omitted when empty), so the subagent self-selects. Imperfect (relies on the subagent honoring it) but it's the only seam available today.
3. **Document the foreground-advisory reality (Gap 1a).** ✅ **Superseded by `260613-fgxx`.** There is no longer a foreground-advisory path to document for apply/hydrate/ship — every post-intake stage dispatches a sub-agent and is tiered uniformly. The narrow residual ("a stage skill run with no dispatch at all") is captured in `docs/specs/stage-models.md` § Foreground limitation.
4. **Harness ask** *(residual — not built)*: a per-subagent `effort` parameter on the Claude Code Agent tool would make Gap 2 closable cleanly — injecting effort directly at the dispatch seam instead of via prose in the prompt. This is **out of fab's control** — flag upstream. It is the **only residual** after `260613-fgxx` + `260613-m3d4`; fab builds nothing for it.

---

## Evidence

- `fab resolve-agent {apply,review,ship}` outputs above — captured 2026-06-13 from `fab-kit` v2.3.1.
- `.claude/skills/_preamble/SKILL.md` § Subagent Dispatch → Per-Stage Model Resolution (the "foreground is advisory-only" + "Claude Code adapter is the Agent tool `model` parameter" contract).
- `.claude/skills/_pipeline/SKILL.md` Steps 1–3 (the mandated pre-dispatch `fab resolve-agent` call).
- `.claude/skills/fab-continue/SKILL.md` header note (foreground = advisory-only).
- `docs/specs/stage-models.md` (tier definitions and the provider-neutral / harness-adapter boundary).
