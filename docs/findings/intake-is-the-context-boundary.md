# Finding: Intake is the sole context boundary — post-intake stages should have one execution mode

**Date**: 2026-06-13
**Area**: pipeline architecture, subagent dispatch, context model
**Status**: open
**Severity**: medium — design simplification; collapses a pervasive dual-mode seam and closes Gap 1a of [per-stage-model-tier-application](per-stage-model-tier-application.md)

---

## The principle

There is exactly **one context-bearing boundary** in the whole pipeline: **intake**.

- **Up to and including intake creation** → runs in the **main session context**, because it needs the live conversation. This is `/fab-new`, `/fab-draft`, the intake-creation prefix of `/fab-proceed`, and `/fab-clarify` (interactive intake refinement). These have *zero context breakage* — the conversation is right there.
- **After intake** → the intake artifact **IS the context**. Every subsequent stage (apply → review → hydrate → ship → review-pr) runs as a **dispatched subagent** that reconstructs its context from artifacts (`intake.md` + `plan.md` + `docs/memory/`), never from conversation history.

Stated as a rule:

> **Main context ≤ intake. Dispatched, artifact-fed blocks > intake. The intake is the handoff payload across the boundary.**

This is the design principle the system has been operating *near* but never named. Naming it resolves several tangles at once.

---

## What it fixes

### 1. The dual-mode `do NOT run fab status` conditional collapses

Today every post-intake block in `fab-continue.md` carries a caller-aware conditional (three instances — lines 100, 156, 176):

> *"When invoked as a subagent (dispatched by `/fab-ff`/`/fab-fff`): do NOT run any `fab status` command … the orchestrator owns the transitions."*

That conditional exists because a block has **two execution modes**: foreground (Path A — `/fab-continue` runs the stage in-session and owns its own transitions) and subagent (Paths B/C/D — orchestrator dispatches the block and owns transitions itself). A Lego brick that behaves differently depending on which baseplate it's snapped onto is not a Lego brick.

Under the principle, **post-intake stages have exactly one execution mode: dispatched.** Plain `/fab-continue apply` *also* dispatches a subagent — there is no in-session execution path for apply/review/hydrate. The conditional doesn't get "deleted"; it never has a second branch to express. Whether the block or the orchestrator owns the `fab status` call becomes a free implementation choice (recommend: orchestrator-as-sequencer), not a forked behavior baked into the block.

### 2. It closes Gap 1a of the model-tier finding (not just narrows it)

[per-stage-model-tier-application](per-stage-model-tier-application.md) Gap 1a: foreground stages can't be tiered because `fab` can't switch the session model mid-run, so `fab resolve-agent` is "advisory only" for manual `/fab-continue` apply/hydrate/ship.

If **every** post-intake stage dispatches a subagent regardless of A/B/C/D, then **`fab resolve-agent` applies uniformly on every post-intake stage** — there is no non-dispatch path left to be the exception. Gap 1a is fully closed. Only **Gap 2** (the Claude Code Agent tool exposes `model` but no per-subagent `effort` knob) survives, and that is a harness limitation, not a fab design choice.

### 3. It makes "the block is a Lego" a testable invariant

A block is correct **iff** it produces the same result given the same `intake.md` + `plan.md` + memory, *regardless of what conversation preceded it*. That is verifiable: dispatch a change's review stage from a cold session vs. mid-conversation — identical output, or the block leaked conversation context. The principle turns "Lego-ness" from a vibe into a property you can test.

---

## What it forces (the load-bearing consequence)

The intake (plus what apply co-generates into `plan.md`'s `## Requirements`) **must be context-complete.** Any decision, constraint, or rejected alternative that lived only in conversation has to be written into the intake *before* the boundary — otherwise the subagent cannot reconstruct it and the block silently degrades.

The mechanism for this **already exists** and is reframed by the principle:

- `/fab-new` **Step 4 (Conversation Context Mining)** and `/fab-proceed`'s **Conversation Context Synthesis** extract "Decisions made / Alternatives rejected / Constraints identified / Specific values agreed" from the conversation into the intake's Assumptions table.
- Today these read as **SRAD-scoring optimizations**. Under the principle they are the **context-flush at the boundary** — the load-bearing mechanism that makes the boundary watertight. This is not a nice-to-have; it is *why* the boundary holds. That reframing alone is worth capturing.

`/fab-clarify` sits cleanly on the **pre-boundary (main-context)** side: it refines the intake interactively (needs the human), and `_pipeline.md` already forbids it inside the post-intake bracket ("clarification is intake-only"). It is part of *getting the intake right*, not a pipeline stage.

---

## Why this is a small refactor, not a large one (Go-side evidence)

The skill layer manufactures the dual mode; **the Go state machine has none.** Verified 2026-06-13 in `src/go/fab/internal/status/status.go`:

- `Start`, `Advance`, `Finish`, `Reset`, `Skip`, `Fail` take a `driver` string, but it flows **only** into `applyMetricsSideEffect` (status.go:617) — it sets `sm.Driver` (stage-metrics) and the transition log (status.go:629, 638). **No state transition reads `driver`.**
- `Finish`'s auto-activate-next (status.go:179–189) and `Reset`'s downstream cascade (status.go:228–240) are computed purely from `StageOrder` / `NextStage` and the current progress map — caller-identity-blind.

**Consequence**: whether `/fab-continue` (foreground) or `_pipeline.md` (orchestrator) calls `fab status finish apply`, the resulting `.status.yaml` *state* is byte-identical; only the recorded `driver` name differs. The CLI is already caller-agnostic. The duality is entirely a skill-prose artifact.

This also settles the **history-shape invariant** that `_pipeline.md` guards ("every conforming run leaves the same `.status.yaml` history shape" — the fail+reset pair must fire every rework cycle). That invariant is enforced by the **call sequence**, not by *who* calls. A uniformly-dispatched apply does not threaten it: the orchestrator still issues the same fail→reset→finish sequence; it just always issues it (no foreground branch where the block self-issues a different sequence).

---

## Open seams (lower confidence — flag before implementing)

1. **The autonomous rework loop is the highest-risk seam.** It's the one place post-intake work makes *decisions* (triage findings → pick fix-code / revise-plan / revise-requirements → edit `plan.md`). Under the principle it must run dispatched, fed by the review block's **structured returned findings** + `plan.md` — never by conversation. Findings are a return value, not conversation, so this is achievable, but it's the spot most likely to leak context if implemented carelessly. It is also legitimately **invocation-level policy**: *who decides on failure* (a human menu in Path A vs. autonomous triage in B/C/D) is a property of how the pipeline was invoked, not of what the review block does. The review block's job is identical in both modes — review the diff, return pass/fail + prioritized findings. Keep the policy in the orchestrator; keep findings as the block's return value; never give the block a `skip §Verdict when subagent` flag.

2. **The interactive-rework menu in `fab-continue.md` § Verdict (Path A).** If post-intake stages always dispatch, the manual rework menu still needs a human — but the human now reads the *dispatched* review's returned findings rather than findings produced in-session. Resolvable (it's the same findings payload), but the UX seam (surfacing a subagent's findings back to the foreground user for a menu choice) needs deliberate design.

3. **`resolve-agent` ownership.** With one execution mode, where does the `fab resolve-agent <stage>` call live — orchestrator (sequencer) or block? Recommend orchestrator, consistent with "invocations control order/grouping/model-tier; blocks control work." This keeps the tier decision out of the block, matching the Lego model.

---

## Relationship to the Lego model

The user's framing: *every stage is a Lego block; invocations (A/B/C/D) control only order and grouping, never what the block does or how.*

- The **work layer is already Lego** — `_generation.md`, `_review.md`, hydrate behavior are clean, caller-agnostic procedures. The hard decomposition is done.
- What's tangled is the **bookkeeping + context skin** around each block. This principle removes the context skin (one boundary, one mode) and the model-tier finding removes the model skin (uniform tier application).
- The one irreducible invocation-level difference is **autonomy / failure policy** (interactive menu vs. bounded auto-rework). That is *correctly* invocation-level under the Lego model — it's about who decides on failure, not what the block does. Keep it in the orchestrator.

---

## Pre-boundary de-duplication: extract `_intake.md`

The principle splits the skill set into two families (see *Relationship to the Lego model*). The post-boundary family's shared orchestration was already extracted into `_pipeline.md`. The **pre-boundary family has the symmetric duplication, and it is not yet cleanly extracted** — it's managed today by two *inconsistent* reuse mechanisms both pointing at `fab-new.md`.

### What's already shared vs. what isn't

The artifact-*generation* mechanics are extracted: `_generation.md` § **Intake Generation Procedure** (read template → fill metadata → write sections → append `## Assumptions` → write file) is consumed by `/fab-new`, `/fab-draft`, and `/fab-continue`'s intake-regen row. That part is done.

But "create an intake" as a skill is `/fab-new` **Steps 0–9** — generation is only Step 5. The surrounding orchestration is the duplication:

| Step | Work | Shared across new/draft/proceed? |
|------|------|----------------------------------|
| 0 | Parse input (Linear ID / backlog ID / NL) | ✅ |
| 1 | Generate slug | ✅ |
| 2 | Gap analysis | ✅ |
| 3 | Create change + re-run/collision check | ✅ |
| 4 | Conversation context mining (**the boundary context-flush**) | ✅ |
| 5 | Generate intake → `_generation.md` | ✅ already extracted |
| 6 | Verify change type | ✅ |
| 7 | Confidence score | ✅ |
| 8 | SRAD question selection | ⚠️ **the one fork**: interactive (new/draft) vs. promptless-defer (proceed) |
| 9 | Advance intake → ready | ✅ |
| 10 | Activate | ❌ `/fab-new` only |
| 11 | Git branch | ❌ `/fab-new` only |

### The two inconsistent reuse mechanisms today

- **`/fab-draft`** is a **prose delta over `fab-new.md`**: "execute its Pre-flight, Arguments, and Steps 0–9 exactly as written … read self-name mentions as `/fab-draft`," then skip 10–11. Fragile — it carries an explicit warning about the "run activation by momentum" failure mode, *because* the steps it must not run live in the same body it's executing.
- **`/fab-proceed`** dispatches `/fab-new` **as a subagent** with a promptless defer-and-surface contract that replaces Step 8's interactive questioning.

So `fab-new.md` is simultaneously *a skill* and *the de-facto shared library* the other two reach into — by two different routes.

### The extraction (mirrors `_pipeline.md`)

Lift Steps 0–9 into a shared helper **`_intake.md`** — the missing pre-boundary peer of `_generation`/`_review`/`_pipeline` — parameterized by the single real fork:

```
_intake.md — "Create-Intake Procedure": Steps 0–9, parameterized by
               {questioning-mode}: interactive | promptless-defer   (Step 8 only)

fab-new.md     = _intake (interactive) + Step 10 activate + Step 11 branch  (tail)
fab-draft.md   = _intake (interactive)                                       (stop at ready)
fab-proceed    = dispatch with _intake (promptless-defer), then /fab-fff
```

`{questioning-mode}` is the *only* behavioral fork in intake creation, and it's legitimately invocation-level under the Lego model — "who resolves ambiguity: the human now, or defer-and-surface" is a property of how you invoked, not of what intake-creation does. Exactly parallel to the post-boundary autonomy fork (interactive menu vs. auto-rework).

### Why it's strictly better than today

1. **`/fab-draft` stops being a prose delta over a sibling skill** — it becomes `_intake(interactive)` + nothing. The "don't run Step 10–11 by momentum" warning evaporates: those steps live in `fab-new.md`, not in the shared body draft executes.
2. **`fab-new.md` stops being a library masquerading as a skill** — it shrinks to `_intake(interactive)` + the activate/branch tail.
3. **It mirrors the proven `_pipeline.md` shape** — shared body parameterized by one knob, call-site-specific tail stays in the call-site file (just as `fab-fff`'s ship/review-pr Steps 4–5 stay in `fab-fff.md`).

### Extraction boundary — do NOT over-extract

Two things stay at the call site, or the extraction recreates the dual-mode problem on the intake side:

- **Activate (10) + branch (11)** are a *different responsibility* (make the change active + checked out vs. queue it), **not** a questioning-mode parameter. They stay as a tail in `fab-new.md`.
- **`/fab-proceed`'s state detection + relevance assessment** (its dispatch table, asymmetric-bias rule, bypass notes — the bulk of its 241 lines) are *not* intake creation. They decide *whether* to call `_intake` at all (create new vs. activate an existing draft). They stay in `fab-proceed.md`.

The boundary: **`_intake.md` = "given I've decided to create an intake, do it (Steps 0–9), with `{questioning-mode}` as the one knob."** Whether to create one, and what to do after (activate/branch), stays at the call site.

### Completes the helper symmetry

This is the natural completion of the extraction pattern already in place — one shared helper per phase:

| Phase | Helper | Consumers |
|-------|--------|-----------|
| artifact mechanics | `_generation.md` | new, draft, continue, ff, fff |
| review mechanics | `_review.md` | continue, ff, fff |
| post-intake orchestration | `_pipeline.md` | ff, fff |
| **pre-intake orchestration** | **`_intake.md`** *(proposed)* | **new, draft, proceed** |

`_intake.md` is also, concretely, *the code that runs the one main-context phase this finding blesses* — the pre-boundary side of the boundary.

---

## Suggested directions (not yet decided)

1. **Make post-intake `/fab-continue` dispatch a subagent** for its stage, the same way orchestrators do — removing the foreground execution path and the three dual-mode conditionals.
2. **Demote orchestrators to pure sequencers**: dispatch block → read returned status/findings → decide proceed/loop/stop. They own `fab status` transitions and `resolve-agent`; they never reach into block internals.
3. **Reframe `/fab-new` Step 4 and `/fab-proceed` synthesis** in docs as the context-flush mechanism, and consider a context-completeness check at the boundary.
4. **Extract `_intake.md`** (pre-boundary de-duplication — see section above): lift `/fab-new` Steps 0–9 into a shared helper parameterized by `{questioning-mode}`, reducing `fab-new`/`fab-draft`/`fab-proceed` to their genuinely-distinct tails.
5. **Sequence with the model-tier finding** — adopting this principle closes that finding's Gap 1a, so they should be planned together.

---

## Evidence

- `.claude/skills/fab-continue/SKILL.md` lines 100/156/176 — the three `do NOT run fab status` dual-mode conditionals.
- `.claude/skills/_pipeline/SKILL.md` line 48 + Steps 1–3 — orchestrator owns transitions; "clarification is intake-only".
- `.claude/skills/fab-new/SKILL.md` Step 4 (Conversation Context Mining); `.claude/skills/fab-proceed/SKILL.md` (Conversation Context Synthesis).
- `.claude/skills/fab-draft/SKILL.md` (prose delta over `fab-new.md` Steps 0–9); `.claude/skills/_generation/SKILL.md` § Intake Generation Procedure (generation mechanics already shared) — evidence for the pre-boundary `_intake.md` extraction.
- `src/go/fab/internal/status/status.go` — `Finish` (160–209), `Reset` (211–243), `applyMetricsSideEffect` (617) — `driver` is metrics-only; state transitions are caller-agnostic. Verified 2026-06-13, fab-kit v2.3.1.
- Companion: [per-stage-model-tier-application](per-stage-model-tier-application.md) (this finding closes its Gap 1a).
