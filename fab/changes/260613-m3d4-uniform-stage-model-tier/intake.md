# Intake: Apply per-stage model tier uniformly + inject effort via subagent prompt

**Change**: 260613-m3d4-uniform-stage-model-tier
**Created**: 2026-06-13

## Origin

Natural-language change request (one-shot, no Linear/backlog ID). This is **"Change C"** in a
three-part coordinated refactor that emerged from an architecture discussion about fab-kit's skill
structure. The full analysis lives in two finding docs that the apply agent MUST read in full before
planning:

- `docs/findings/per-stage-model-tier-application.md` — the primary finding for this change (defines Gap 1a, Gap 1b, Gap 2).
- `docs/findings/intake-is-the-context-boundary.md` — the companion finding; its principle is **Change A**, on which this change depends.

The user's verbatim request:

> fix: Apply per-stage model tier uniformly + inject effort via subagent prompt
>
> fab defines per-stage model tiers in docs/specs/stage-models.md: thinking={opus,xhigh} for
> intake+review, doing={opus,high} for apply+review-pr+hydrate, fast={sonnet,low} for ship.
> `fab resolve-agent <stage>` ships and resolves these correctly. But the resolved profile is ONLY
> applied at the Agent-tool dispatch seam, producing two gaps (compliance + architectural). Close
> Gap 1b (compliance visibility) and Gap 2's effort half (inject effort into the subagent prompt),
> after Change A removes the foreground execution path (which closes Gap 1a). Document the harness ask
> for a per-subagent effort param. Skills/prose only — no Go change expected.

**Execution-plan sequencing** (from the discussion): three coordinated changes — A
(intake-is-the-context-boundary: make every post-intake stage dispatch a subagent, removing the
foreground path), B (independent), C (this change). Order: **B and A first; C LAST, gated on A.** This
change is scoped to assume Change A has already landed.

## Why

### The problem

fab assigns each pipeline stage a model **tier** (`docs/specs/stage-models.md`):

| Tier | Stages | Default profile |
|------|--------|-----------------|
| `thinking` | intake, review | `claude-opus-4-8` + `xhigh` |
| `doing` | apply, review-pr, hydrate | `claude-opus-4-8` + `high` |
| `fast` | ship | `claude-sonnet-4-6` + `low` |

`fab resolve-agent <stage>` ships and resolves these correctly. **Verified 2026-06-13 on fab-kit
v2.3.1** (re-confirmed in this intake's gap analysis):

```
$ fab resolve-agent apply   →  model=claude-opus-4-8   effort=high
$ fab resolve-agent review  →  model=claude-opus-4-8   effort=xhigh
$ fab resolve-agent ship    →  model=claude-sonnet-4-6 effort=low
```

But the resolved profile is **only applied at the Agent-tool dispatch seam**. This produces three
gaps (the model-tier finding's taxonomy):

**GAP 1 (compliance) — two sub-cases:**

- **1a — Foreground stages can't be tiered at all.** When a stage runs directly in the foreground
  (plain `/fab-continue apply`/`hydrate`/`ship`), fab cannot switch the session model mid-run, so the
  tier is "advisory only". `_preamble.md` § Per-Stage Model Resolution ("Foreground is advisory-only")
  and `fab-continue.md`'s header note ("Per-stage model (foreground = advisory-only)") both say so.
  `docs/specs/stage-models.md` § Foreground limitation documents it as by-design.
- **1b — Orchestrators can SILENTLY SKIP the mandated `fab resolve-agent <stage>` call.**
  `_pipeline.md` Steps 1/2/3 (and `fab-fff.md` Steps 4/5) mandate: *"immediately before dispatching
  each stage's sub-agent, run `fab resolve-agent <stage>` and pass the resolved model AND effort into
  the Agent dispatch."* Nothing enforces this — it is prose, not a code-level guard. **Observed in the
  wild**: a real orchestrated run dispatched apply/review subagents with NO model override; they
  inherited the session profile instead of being pinned to `opus+high` / `opus+xhigh`. (The agent's
  self-diagnosis correctly identified the symptom — "subagents ran on inherited session settings" —
  but mis-attributed the cause to "in-process Agent dispatch doesn't route through resolve-agent": the
  routing IS defined; the run simply didn't execute the step.)

**GAP 2 (architectural) — the Claude Code Agent tool exposes a `model` parameter but NO per-subagent
`effort` parameter.** Even a fully compliant orchestrator that calls `fab resolve-agent apply` and
gets `model=claude-opus-4-8 effort=high` can pin the MODEL (via the Agent tool's `model` param) but
**cannot inject the EFFORT half** of the tier — there is no `effort` seam on the Agent tool. So the
`effort=` line that `fab resolve-agent` emits is currently **UNCONSUMABLE** on the per-subagent
dispatch path in Claude Code, regardless of orchestrator correctness. The subagent runs at whatever
effort the session/harness governs. This is a real harness-adapter limitation, not a compliance miss.

### Why it matters

The whole point of `doing` vs `thinking` tiers is cost/quality calibration — apply on `high`, review
on `xhigh`. If effort can't be pinned per-subagent (Gap 2) and the model isn't pinned when the call is
skipped (Gap 1b), stages silently run at the session's effort. A session driven at `xhigh` over-spends
on apply; a session at a low effort under-thinks review. The tier mapping looks authoritative in config
but is only partially enforced at runtime.

### Why this approach over alternatives

- For Gap 1b: a true code-level guard would require fab to observe/intercept Agent-tool dispatches,
  which it cannot (dispatch is harness-internal). The cheap, available seam is **prose-level
  visibility** — emit the resolved `model=/effort=` lines into the dispatch prompt and/or log, so a
  skipped or mis-resolved tier is VISIBLE in output rather than silent. (A lightweight guard MAY be
  considered if feasible — see Open Questions.)
- For Gap 2: the Agent tool has no effort param, so the only seam available **today** is the
  **subagent prompt** — inject the resolved effort as an explicit instruction (e.g., "operate at
  `xhigh` reasoning effort"), so the subagent self-selects. Imperfect (relies on the subagent honoring
  it) but it is the only seam available. The clean fix (a per-subagent `effort` param on the Agent
  tool) is **out of fab's control** — document it as a harness ask, do not attempt to build it.

## What Changes

> **CRITICAL DEPENDENCY — this change is scoped to land AFTER Change A.** Change A
> (`intake-is-the-context-boundary`) makes **every** post-intake stage dispatch a subagent regardless
> of invocation path (`/fab-continue` foreground, `/fab-ff`, `/fab-fff`, `/fab-proceed`). Once there is
> **no foreground execution path left**, `fab resolve-agent` applies uniformly and **Gap 1a
> DISAPPEARS** — there is no exception remaining. Therefore this change's scope is **reduced to Gap 1b
> + Gap 2 + doc updates**. **Do NOT duplicate Change A's work** (do not implement the
> "every-stage-dispatches-a-subagent" mechanic here — that is A's deliverable; this change assumes it
> is already in place). If, at apply time, Change A has NOT yet landed (the foreground advisory caveats
> still exist and `/fab-continue` still runs apply/hydrate/ship in-session), STOP and surface that the
> dependency is unmet — see Open Questions.

All edits are to **canonical source** under `src/kit/skills/*.md` and `docs/`. Per the constitution,
`.claude/skills/` is gitignored deployed copies — **never edit those directly**.

### 1. Close Gap 1b — compliance visibility at every per-stage dispatch site

Have orchestrators **emit the resolved `model=/effort=` lines into the dispatch prompt and/or log
them**, so a skipped or mis-resolved tier is VISIBLE in output rather than silent. This is cheap,
prose-level. Apply at every per-stage dispatch site:

- **`src/kit/skills/_pipeline.md`** — the Behavior "Per-stage model resolution" note (around line 50)
  and Steps 1 (apply, ~line 60), 2 (review, ~line 72), 3 (hydrate, ~line 117), and the Auto-Rework
  Loop items 3 (re-dispatch apply, ~line 88) and 4 (fresh re-review, ~line 89). Each of these already
  instructs `fab resolve-agent <stage>` immediately before dispatch — extend each so the resolved
  `model=/effort=` is surfaced (echoed into the dispatch prompt and/or logged) rather than only
  silently consumed.
- **`src/kit/skills/fab-fff.md`** — Steps 4 (ship, ~line 47) and 5 (review-pr, ~line 57), which run
  `fab resolve-agent ship` / `fab resolve-agent review-pr` before dispatching `/git-pr` / `/git-pr-review`.

**Consider whether a lightweight guard is feasible** (e.g., the orchestrator asserting the resolved
lines are non-empty / match the expected stage before dispatching, or a post-hoc visibility log line).
Keep it prose-level; do not add Go. If no guard is cleanly feasible, visibility-in-output alone
satisfies this item.

### 2. Close Gap 2's effort half — inject the resolved effort into the SUBAGENT PROMPT

Since the Claude Code Agent tool has **no effort param**, inject the resolved **effort** into the
**subagent prompt** as an explicit instruction so the subagent self-selects its reasoning effort.
Example phrasing (final wording is the apply agent's choice — keep it imperative and unambiguous):

> "Operate at `xhigh` reasoning effort for this task."

Where the resolved effort is empty, omit the instruction (mirroring the existing "empty effort ⇒ omit"
contract). Apply this at **every per-stage dispatch site**:

- **`src/kit/skills/_pipeline.md`** — Steps 1–3 dispatch prompts and the Auto-Rework Loop's re-apply
  (item 3) and re-review (item 4) dispatch prompts. The `review` stage resolves **once** and applies
  the same effort instruction to both reviewer sub-agents (inward + outward) and the merge — preserve
  the existing "review resolves once" contract.
- **`src/kit/skills/fab-fff.md`** — Steps 4 (ship) and 5 (review-pr) dispatch prompts. (Note: the
  description names "fab-fff.md (Steps 4–5)" specifically as a dispatch site for the effort injection.)

This is **imperfect** (relies on the subagent honoring the instruction) but it is the only seam
available today. Keep the model half on the Agent tool's `model` param (unchanged — already wired); add
the effort half via the prompt.

> **Where the resolved model+effort already flow** (existing wiring, do not remove — extend): the
> dispatch contract today is "run `fab resolve-agent <stage>`, pass resolved model to the Agent tool's
> `model` param (empty ⇒ omit/inherit), omit effort (no seam)." This change keeps the model wiring and
> **adds** the effort-into-prompt + visibility behavior. The canonical contract lives in `_preamble.md`
> § Subagent Dispatch → Per-Stage Model Resolution — update it to reflect the new effort-via-prompt
> seam (see item 3).

### 3. Update docs to reflect the new uniform behavior

- **`docs/findings/per-stage-model-tier-application.md`** — Gap 1a's `> **Closed by ...**` note
  (~line 52) currently reads as a forward-looking "if Change A lands" projection. Update it to reflect
  that Gap 1a is **closed** by the now-landed Change A (uniform subagent dispatch). Update the finding's
  `**Status:**` and the per-path table (~lines 40–46) as appropriate to reflect that there is no
  longer a foreground-vs-subagent split. Reflect that Gap 1b and Gap 2's effort-half are addressed by
  this change (visibility + prompt-injection), with the residual being the harness ask.
- **`docs/specs/stage-models.md`** — § Foreground limitation (~lines 263–272) and § Skill wiring
  describe the now-removed foreground exception and the "empty effort ⇒ omit" behavior. Reconcile them
  with the new uniform behavior: no foreground advisory path (post-Change-A), and effort is now
  injected via the subagent prompt rather than dropped. (`docs/specs/` is human-curated pre-impl design
  intent per Constitution VI — update it to keep design intent accurate, do not auto-generate.)
- **`src/kit/skills/_preamble.md`** § Subagent Dispatch → Per-Stage Model Resolution — update the
  "Foreground is advisory-only" paragraph (Gap 1a is gone post-A) and the harness-adapter boundary note
  to document the **effort-via-prompt** seam (the Agent tool injects model via its `model` param;
  effort is injected via the subagent prompt instruction).
- **`src/kit/skills/fab-continue.md`** header note (~line 19, "Per-stage model (foreground =
  advisory-only)") — the foreground advisory caveat is obsolete once Change A removes the foreground
  path. Reconcile it (Gap 1a deletion).

> **Note on overlap with Change A**: items that delete the *foreground advisory caveats* themselves are
> partly Change A's territory (A removes the foreground path). This change owns the **model-tier-specific
> reconciliation** of those caveats (the "advisory-only because tier can't apply" language) and the
> finding-doc/spec updates tied to the tier mechanism. If Change A has already removed a given caveat at
> apply time, do not re-remove it; reconcile only what remains. Treat the boundary pragmatically: the
> goal is that after both changes land, no doc claims a foreground stage is "tier-advisory-only" and
> every dispatch site injects effort via prompt. Avoid edit collisions on shared lines (see Open
> Questions on coordination).

### 4. HARNESS ASK (document only — out of fab's control)

A **per-subagent `effort` parameter on the Claude Code Agent tool** would close Gap 2 cleanly (inject
effort directly at the dispatch seam instead of via prose in the prompt). This is **out of fab's
control** — flag it upstream, **do NOT attempt to build it**. Document it in
`docs/findings/per-stage-model-tier-application.md` (its § Suggested directions item 4 already names
this; ensure it remains and is marked as the residual after this change). Do not add any fab Go code or
skill mechanism for it.

### Constraints (from the constitution + the request)

- `src/kit/` is canonical; `.claude/skills/` is gitignored deployed copies — **edit ONLY source**.
- Skill-file changes **MUST** update the corresponding `docs/specs/skills/SPEC-*.md`. Affected SPEC
  files: **`SPEC-_pipeline.md`**, **`SPEC-_preamble.md`**, **`SPEC-fab-fff.md`**,
  **`SPEC-fab-continue.md`** (each mirrors a skill file edited above). Update each to reflect the
  effort-via-prompt + visibility behavior and the Gap 1a removal.
- If `fab resolve-agent`'s signature changes (**NOT expected** — it already resolves model+effort
  correctly), update `src/kit/skills/_cli-fab.md`. This change is **expected to be skills/prose +
  docs only — NO Go change**. (Gap analysis confirmed `fab resolve-agent` already emits the correct
  `model=`/`effort=` lines; this change consumes them differently, it does not change the resolver.)
- Markdown-only artifacts; CommonMark syntax (Constitution IV + Additional Constraints).

## Affected Memory

- `pipeline/execution-skills`: (modify) `_pipeline` orchestration — record that per-stage dispatch
  sites now emit/log the resolved `model=/effort=` (Gap 1b visibility) and inject effort into the
  subagent prompt (Gap 2 effort half); `fab-fff` Steps 4–5 likewise.
- `pipeline/planning-skills`: (modify) only if the `_preamble` Per-Stage Model Resolution contract
  reframing (effort-via-prompt; Gap 1a removed) touches planning-skill-documented behavior — likely a
  light touch or none; confirm during hydrate.
- `distribution/distribution` or `runtime/*`: (modify, **only if relevant**) the request flags
  "distribution/runtime if relevant to resolve-agent". `fab resolve-agent` itself is unchanged, so a
  runtime-memory touch is unlikely; include only if hydrate finds the per-stage-model-resolution
  behavior is documented there and needs reconciliation. <!-- assumed: runtime/distribution memory likely untouched since resolve-agent's signature/behavior is unchanged — flagged as conditional per the request's "if relevant" wording -->

## Impact

**Files expected to change (all canonical source / docs — no Go):**

- `src/kit/skills/_pipeline.md` — per-stage dispatch notes + Steps 1–3 + Auto-Rework items 3/4: emit/log resolved tier; inject effort into dispatch prompts.
- `src/kit/skills/fab-fff.md` — Steps 4–5 (ship, review-pr): emit/log resolved tier; inject effort into dispatch prompts.
- `src/kit/skills/_preamble.md` — § Subagent Dispatch → Per-Stage Model Resolution: document effort-via-prompt seam; remove/ reconcile the foreground-advisory paragraph (Gap 1a gone post-A).
- `src/kit/skills/fab-continue.md` — header per-stage-model note: reconcile the foreground-advisory caveat.
- `docs/findings/per-stage-model-tier-application.md` — Gap 1a "Closed by" note → closed; status/table updates; harness-ask residual.
- `docs/specs/stage-models.md` — § Foreground limitation + § Skill wiring: reconcile with uniform behavior + effort-via-prompt.
- `docs/specs/skills/SPEC-_pipeline.md`, `SPEC-_preamble.md`, `SPEC-fab-fff.md`, `SPEC-fab-continue.md` — mirror the skill-file edits (constitution requirement).

**Not expected to change:** any Go (`src/go/**`), `src/kit/skills/_cli-fab.md` (resolver signature
unchanged), templates, migrations. **No migration needed** — this change does not restructure user
data (config/`.status.yaml`/archive layout); it is skills + docs only.

**Dependency / sequencing impact:** gated on Change A (`intake-is-the-context-boundary`). Change B is
independent and may land in any order relative to this. Coordinate edits to shared files (`_pipeline.md`,
`_preamble.md`, `fab-continue.md`) with Change A to avoid collisions (see Open Questions).

**Verification:** since no behavior is encoded in Go, the proof is documentary + consistency: after both
A and C land, no doc/skill claims a foreground stage is "tier-advisory-only", and every per-stage
dispatch site (`_pipeline` Steps 1–3 + rework items 3/4; `fab-fff` Steps 4–5) injects the resolved
effort into its subagent prompt and surfaces the resolved tier. Run `internal-consistency-check`-style
cross-checks between `_pipeline.md`/`_preamble.md`/`fab-fff.md` and their SPEC mirrors + the
finding/stage-models docs.

## Open Questions

- **Is Change A already landed at apply time?** This change is gated on Change A removing the foreground
  execution path (which closes Gap 1a). If A has NOT landed when apply runs, the apply agent should STOP
  and surface the unmet dependency rather than (a) duplicating A's work or (b) editing around a
  foreground path that still exists.
- **Lightweight guard for Gap 1b — feasible or not?** The request says "consider whether a lightweight
  guard is feasible." Decide at apply time: a prose-level assertion (orchestrator checks the resolved
  lines are non-empty before dispatch) vs. visibility-in-output only. No Go.
- **Exact effort-injection wording.** The example is "operate at `xhigh` reasoning effort"; the apply
  agent chooses final phrasing (imperative, unambiguous, omitted when effort is empty).
- **Edit-collision coordination with Change A** on shared files (`_pipeline.md`, `_preamble.md`,
  `fab-continue.md`). Since C is gated on A, A's edits should already be in place; reconcile rather than
  re-introduce.

## Assumptions

<!-- STATE TRANSFER: see template note. All four SRAD grades recorded; Scores required on every row. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skills/prose + docs only — no Go change; `fab resolve-agent` signature is unchanged (gap analysis re-verified it already emits correct `model=`/`effort=`), so `_cli-fab.md` is untouched | Stated explicitly in the request and constitution constraint; verified live in gap analysis | S:95 R:80 A:95 D:95 |
| 2 | Certain | Edit ONLY `src/kit/skills/*` (canonical), never `.claude/skills/*` (gitignored deployed copies) | Constitution Principle V + Additional Constraints; project context.md | S:100 R:70 A:100 D:100 |
| 3 | Certain | Skill-file edits MUST update their `SPEC-*.md` mirrors: `SPEC-_pipeline.md`, `SPEC-_preamble.md`, `SPEC-fab-fff.md`, `SPEC-fab-continue.md` | Constitution Additional Constraint (skill changes → SPEC update); files confirmed to exist | S:90 R:75 A:95 D:90 |
| 4 | Certain | Scope is Gap 1b + Gap 2-effort + doc updates only; Gap 1a is closed by Change A (not duplicated here) | Request "IMPORTANT DEPENDENCY" section + both finding docs state Gap 1a disappears under A | S:95 R:70 A:90 D:90 |
| 5 | Confident | Gap 1b visibility = emit/log resolved `model=/effort=` at each dispatch site; Gap 2 effort = inject "operate at X effort" into the subagent prompt | Request items 1–2 prescribe these seams explicitly; the only available seam for effort (no Agent-tool effort param) | S:90 R:70 A:85 D:80 |
| 6 | Confident | Per-stage dispatch sites are `_pipeline.md` Steps 1–3 + Auto-Rework items 3/4 (review resolves once → both reviewers + merge) and `fab-fff.md` Steps 4–5 | Located them directly in source during context loading; request names `_pipeline.md` and `fab-fff.md Steps 4–5` | S:85 R:75 A:90 D:85 |
| 7 | Confident | Harness ask (per-subagent `effort` param on Agent tool) is document-only — do NOT build | Request item 4 + finding § Suggested directions item 4; out of fab's control | S:95 R:90 A:90 D:90 |
| 8 | Tentative | runtime/distribution memory is likely untouched (resolver unchanged); include only if hydrate finds documented per-stage-model behavior there needing reconciliation | Request says "distribution/runtime if relevant"; conditional — defer the call to hydrate | S:55 R:75 A:60 D:60 |
| 9 | Tentative | A lightweight Gap 1b guard (prose assertion that resolved lines are non-empty before dispatch) MAY be added if cleanly feasible; otherwise visibility-in-output alone suffices | Request says "consider whether a lightweight guard is feasible"; no Go, prose-level only — decision deferred to apply | S:50 R:70 A:60 D:55 |

9 assumptions (4 certain, 3 confident, 2 tentative, 0 unresolved).
