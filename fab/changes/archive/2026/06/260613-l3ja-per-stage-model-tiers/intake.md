# Intake: Per-Stage Model Selection via Named Tiers

**Change**: 260613-l3ja-per-stage-model-tiers
**Created**: 2026-06-13

## Origin

> User: "In the earlier days of fab-kit, most stages used to be performed by the main agent in foreground. Only recently (since Opus 4.7) has fab-kit started running most stages via sub-agents. […] I want more control over which models get used for which lifecycle stages — eg: high-end mode for intake, sonnet for execution, high-end for code review, haiku for git stages. To be able to do this, we need sensible defaults, user overrides, project overrides etc."

This change is the product of an extended design conversation (conversational mode). The design space was explored thoroughly and converged through several iterations; the decisions below are **settled**, not open. Key turning points in the conversation:

1. **Why subagents now** — established that the move to sub-agents was driven by *context isolation* (a six-stage autonomous pipeline can't fit in one context window), with Opus 4.7+ as the *enabler* (reliable adherence to the subagent contract: return structured results, don't touch orchestrator-owned state). This established that the existing sub-agent dispatch seam is the natural injection point for a per-stage model.
2. **Granularity** — chose **stage-keyed** over role-keyed for config simplicity/onboarding, then refined to **tier-keyed** (named tiers) once it became clear stages cluster by cognitive mode.
3. **Layers** — dropped the user (`~/.fab-kit`) layer; only fab-kit defaults + project override.
4. **Tiers are `{model, effort}` profiles**, not bare models — effort is a first-class spend lever.
5. **Haiku excluded from the defaults** — it has no effort parameter (passing effort 400s), and the one stage that might want a fast/cheap model (ship) needs faithful PR-description comprehension that Haiku does unreliably. (A user *may* still override a tier to Haiku — pass-through doesn't forbid it; fab just doesn't ship it as a default.)
6. **Stage→tier taxonomy is fab's, fixed and non-overridable**; users override only what each tier *means*. This is the final inversion — the dimensional analysis that produced the taxonomy is fab's judgment to own, while *budget* (what a tier costs) is the user's to set.

A companion design doc already exists at `docs/specs/stage-models.md` (written during the conversation, partly mid-iteration); it must be reconciled to the final design described here.

## Why

**Problem.** Every fab pipeline stage currently runs on whatever model the session was launched with (the orchestrator's foreground model, or the model a dispatched sub-agent inherits). There is no way to spend model quality where judgment lives (intake, review) and economize where work is mechanical (ship). A user running the whole pipeline on `--effort xhigh` Opus pays maximum cost on stages that don't need it; a user economizing globally under-powers the stages where bug-catching and requirement capture actually matter.

**Consequence of not fixing.** Pipeline cost and latency are an all-or-nothing dial. As model tiers diverge in price (Opus $5/$25 vs Sonnet $3/$15 per MTok) and capability, the inability to match model+effort to the cognitive demand of each stage means either overspending on mechanical stages or under-powering judgment stages — and no clean upgrade path when a new top model (e.g. Fable) lands.

**Why this approach (named tiers) over alternatives.**
- *Over per-stage `{model, effort}` config*: six stages × two dials is too many knobs and forces users to re-derive each stage's judgment profile. Named tiers (3) group stages by cognitive mode and hide the per-stage reasoning.
- *Over concrete model IDs in config*: abstract tiers survive model churn — when Fable lands, fab bumps the default tier→profile table in ONE place and every non-overriding project upgrades for free. A pinned `claude-opus-4-8` string would rot.
- *Over stage-reassignment overrides*: the stage→tier mapping is a considered judgment (derived from a dimensional analysis); letting users reassign stages invites undoing reasoning they haven't done. Users legitimately disagree about *budget*, not *taxonomy* — so the override surface is tier *redefinition* (what a tier costs), not tier *reassignment* (which stages belong to it).

## What Changes

### 1. The three tiers and the fixed stage→tier mapping

A **tier** is a named `{model, effort}` profile. Three tiers, grouped by cognitive mode:

| Tier | Stages (FIXED — fab-owned, not user-overridable) | Default profile (today) | Cognitive mode |
|------|---------------------------------------------------|--------------------------|----------------|
| `thinking` | `intake`, `review` | Opus 4.8 / xhigh | **Generative judgment** — intake *discovers* requirements; review *discovers* bugs. Deliberation directly buys quality. |
| `doing` | `apply`, `review-pr`, `hydrate` | Opus 4.8 / high | **Execution that must not err** — apply writes the diff; review-pr fixes already-articulated feedback (responsive, not generative); hydrate writes memory. |
| `ship` | `ship` | Sonnet 4.6 / low | **Speed on near-mechanical work** — commit/push/PR mechanics + a faithful PR-description summary. |

The six pipeline stages are `intake → apply → review → hydrate → ship → review-pr`. The mapping above is exhaustive (every stage belongs to exactly one tier).

**Critical distinction — review vs review-pr:** they share the word "review" but not the cognitive mode. `review` is **generative** (reads a diff and discovers what's wrong from nothing → `thinking`); `review-pr` is **responsive** (triages and fixes feedback someone else already generated → `doing`). Do not group them together.

### 2. Config schema — `agent.tiers` (the ONLY override surface)

New optional `agent.tiers` map in `fab/project/config.yaml`, under the existing `agent:` block (where `spawn_command` lives). The Go `Config` struct widens freely — yaml unmarshalling ignores unknown keys, the same property that made `stage_hooks` free to add (`internal/config/config.go`).

```yaml
agent:
  spawn_command: claude --dangerously-skip-permissions --effort xhigh -n "$(basename "$(pwd)")"

  # The stage→tier mapping below is OWNED BY FAB-KIT and is NOT overridable —
  # shown here only as reference so you know which stages each tier governs:
  #   thinking: intake, review        (generative judgment)
  #   doing:    apply, review-pr, hydrate   (execution that must not err)
  #   ship:     ship                  (speed on near-mechanical work)
  #
  # You override only WHAT EACH TIER MEANS (model + effort). Omit any tier to
  # use fab-kit's built-in default. fab-kit defaults today are:
  #   thinking: { model: claude-opus-4-8,   effort: xhigh }
  #   doing:    { model: claude-opus-4-8,   effort: high  }
  #   ship:     { model: claude-sonnet-4-6, effort: low   }
  tiers:
    doing: { model: claude-sonnet-4-6, effort: medium }   # example: run the doing tier cheaper
```

- Keys under `tiers:` are tier names: `thinking`, `doing`, `ship`.
- Each value is a `{model, effort}` object. Either field MAY be set; an omitted field falls back to the fab-kit default for that tier.
- A tier omitted entirely from `agent.tiers` (or an absent `tiers:` block) uses fab-kit's built-in default profile for that tier.
- There is **no `stage_tiers` map** (stage→tier reassignment is not a user knob) and **no per-stage escape hatch** (a stage cannot be individually pinned to a model/effort outside its tier). Disagreement with the tiering is an upstream fab-kit issue, not a project knob.

### 3. fab-kit owns two tables in the Go binary

1. **Default tier → `{model, effort}` table** (today: thinking=opus-4-8/xhigh, doing=opus-4-8/high, ship=sonnet-4-6/low).
2. **Fixed stage → tier mapping** (thinking={intake, review}, doing={apply, review-pr, hydrate}, ship={ship}).

Both get a **drift-guard test** mirroring the docs — same pattern as `TestDocTablesMatchScoringMaps` for the `expected_min` table (`docs/specs/change-types.md` ↔ `internal/score/score.go`). The doc tables in `docs/specs/stage-models.md` and the Go maps must not drift.

### 4. New command: `fab resolve-agent <stage>`

A pure-query command (no side effects), in the same family as `fab resolve`. Behavior:

1. Take a stage name (`intake`/`apply`/`review`/`hydrate`/`ship`/`review-pr`).
2. Map the stage → its tier via the fixed stage→tier mapping.
3. Resolve the tier → `{model, effort}`: project's `agent.tiers.<tier>` override if present (per-field merge over the default), else fab-kit's built-in default for that tier.
4. **Emit verbatim — NO validation** (see §5). fab does not check the model or effort against any provider's accepted set; it echoes the resolved strings as-is. Compatibility is the runtime/harness's concern, not fab's.
5. Emit the concrete `{model, effort}` on stdout. **Output shape:** two lines, `model=<id>` and `effort=<level>` (effort line omitted when the tier has no effort configured — e.g. an empty/absent effort). An **empty model** signals "inherit the session/orchestrator model" (today's behavior) — used when a tier is intentionally configured with no model.
6. Byte-stable for the same config (like other `fab resolve` queries). Non-zero exit only on a real error (unreadable/malformed config); an unknown stage name is an error. A stage that resolves to a default is success, not an error.

Named `resolve-agent` (not `resolve-model`) because it resolves both the model AND the effort the agent dispatch needs.

Per the constitution, this CLI addition MUST be documented in `src/kit/skills/_cli-fab.md` (and surfaced in the Common fab Commands table in `_preamble.md` if warranted).

### 5. No validation — verbatim pass-through (provider-neutral)

`fab resolve-agent` does **NOT** validate the model or effort against any provider's accepted set. It maps stage→tier→`{model, effort}` and **echoes both strings verbatim**, whatever they are — `xhigh` for an Opus model, `high` for Sonnet, `reasoning_effort`-style values for a non-Claude model a project might configure, or an empty effort. fab has no provider-specific knowledge in the resolution path.

**Rationale (Constitution Principle I — provider neutrality):** validating against Claude's effort enum (`low/medium/high/xhigh/max`, Opus-only `xhigh`, Haiku-rejects-all) would hard-code Claude into the resolver and bolt the door on other agents. Keeping it open — verbatim pass-through — is what lets a project switch the underlying agent by overriding `agent.tiers` with that provider's model IDs and effort vocabulary, with nothing in fab rejecting it. The **safety net moves from fab to the runtime/harness**: a misconfigured pair (e.g. Claude `{model: claude-sonnet-4-6, effort: xhigh}`, which Sonnet rejects with a 400) is *not* corrected by fab — it surfaces as a dispatch-time error. This is the accepted tradeoff for portability.

**For reference only** (NOT enforced by fab) — Claude's effort validity, so the fab-kit *defaults* are chosen to be valid:

| Model | Accepts effort? | Valid values |
|-------|-----------------|--------------|
| Opus 4.8 | Yes | `low`, `medium`, `high`, `xhigh`, `max` |
| Sonnet 4.6 | Yes | `low`, `medium`, `high`, `max` (no `xhigh` — Opus-family only) |
| Haiku 4.5 | **No** | — (effort param returns HTTP 400) |

This table explains *why fab-kit's shipped defaults are what they are* (e.g. `ship` is Sonnet/low not Sonnet/xhigh; Haiku is absent from the defaults because it can't carry effort) — but it is documentation of fab's default choices, not a rule the resolver enforces on user overrides.

### 6. Skill wiring — orchestrator/dispatch consume `fab resolve-agent`

The orchestrators (`/fab-ff`, `/fab-fff`, `/fab-proceed`) and `/fab-continue`'s sub-agent dispatch call `fab resolve-agent <stage>` immediately before dispatching each stage's sub-agent, and pass the resolved **model AND effort** to the Agent dispatch:
- Empty model → omit the `model` param (inherit session/orchestrator model — today's behavior).
- Empty effort → omit the effort flag.

The **`review` stage resolves once** and applies the same `{model, effort}` to BOTH reviewer sub-agents (inward + outward) and the merge — a consequence of stage(/tier)-granularity, documented as a known tradeoff (the mechanical merge runs at the reviewer's tier; acceptable for config simplicity).

Affected skill source files (canonical sources at `src/kit/skills/` — never edit deployed `.claude/skills/` copies): `fab-ff.md`, `fab-fff.md`, `fab-proceed.md`, `fab-continue.md`, and likely the dispatch contract in `_preamble.md` (§ Subagent Dispatch) and/or `_pipeline.md`. Per the constitution, each changed skill MUST update its corresponding `docs/specs/skills/SPEC-*.md`.

### 7. Migration

Ship a migration file in `src/kit/migrations/` (per `docs/memory/distribution/migrations.md` format + versioning) that adds a **FULLY COMMENTED** `agent.tiers` block to existing `fab/project/config.yaml` files. The block:
- Documents the fixed stage→tier mapping as reference comments (non-overridable).
- Documents fab-kit's built-in default profiles per tier.
- Shows the override shape (commented example), but is **entirely commented out** by default — fab-kit uses its built-in defaults; the user opts in by uncommenting/editing.

The migration is idempotent (safe to re-run; doesn't duplicate the block if already present).

### 8. Documentation reconciliation

- `docs/specs/stage-models.md` — already exists (written mid-conversation, partly stale); reconcile to this final design: tiers as `{model, effort}`, the three named tiers with the final stage groupings, `agent.tiers` as the sole override, `fab resolve-agent`, Haiku exclusion rationale, the Fable upgrade-for-free property, the apply↔review coupling rationale, the foreground-advisory-only limitation.
- `docs/specs/architecture.md` — document the `agent.tiers` config block alongside the existing `stage_hooks` example (~line 217-232).
- `docs/specs/index.md` — entry already added.

### 9. Design properties to capture as rationale (settled — not open questions)

- **Foreground limitation:** per-stage selection is fundamentally a property of orchestrated/sub-agent runs. A user running `/fab-continue` directly in the foreground cannot have the session model switched mid-run; for those runs the configured tier is **advisory only** (the skill MAY note "this stage is configured for X; you're on Y" but MUST NOT attempt to switch). By design.
- **apply↔review coupling:** apply produces the diff review critiques. Keeping apply on the capable model (`doing` = Opus/high) rather than a cheaper one reduces rework cycles (capped at 3 per `code-review.md`); three expensive review rounds can cost more than one capable apply. This is why apply is `doing`, not a cheaper tier. The savings on the doing tier come from *effort* (high, not xhigh), not a model downgrade.
- **Fable upgrade path:** when Fable access lands, fab bumps the default table in ONE place — `thinking` → Fable/xhigh, `doing` → Opus/xhigh — and every non-overriding project upgrades for free. A project that overrides a tier opts OUT of fab's upgrade curve for that tier (correct behavior; name it in the docs). Note the doing tier's effort also rises (high→xhigh) under Fable — the tier→profile table is fab's curated judgment per release, not a fixed effort-per-tier-rank rule.
- **Provider neutrality (Constitution Principle I):** the feature is **provider-neutral by construction**, not Claude-locked, and v1 states this as a requirement.
  - *Portable layers (no provider knowledge):* the `agent.tiers` config schema, and the entire `fab resolve-agent` resolution path (stage→tier→`{model, effort}`). The resolver does no validation and echoes strings verbatim (§5), so a project can switch agents by overriding `agent.tiers` with another provider's model IDs and effort vocabulary (`gpt-5 / reasoning_effort:high`, `gemini-* / <its-knob>`, etc.) and nothing in fab rejects it.
  - *Harness-specific layer (the adapter):* injecting the resolved model+effort into the actual sub-agent dispatch is harness behavior. For Claude Code that is the Agent tool's `model` parameter; the skill wiring names this explicitly as the Claude-Code adapter, not as universal truth. This coupling is **not introduced by this feature** — fab's entire existing subagent-dispatch design (`_preamble.md` § Subagent Dispatch) is already Claude-Code-shaped. Per-stage selection is exactly as portable as fab's existing dispatch: no more, no less.
  - *Claude-flavored data (overridable):* fab-kit's shipped default table uses Claude model IDs/effort. These are documented as "fab-kit's Claude defaults," fully replaceable via `agent.tiers`.
  - *v1 scope is architecture-neutral + documented — NOT shipped/tested against a non-Claude harness.* No per-provider default tables, no provider-detection, no non-Claude integration test. The acceptance proof is "a non-Claude project can override the tiers and nothing in fab rejects it," not "we ran it on a non-Claude harness." Shipped+tested multi-provider support is explicitly out of scope (a far larger change).

## Affected Memory

- `pipeline/change-lifecycle.md`: (modify) The stage lifecycle gains a per-stage model-resolution step at sub-agent dispatch (orchestrators call `fab resolve-agent <stage>`).
- `_shared/configuration.md`: (modify) Documents the `agent` config section (currently `spawn_command`); add `agent.tiers` and the tier semantics.
- `distribution/migrations.md`: (modify) A new migration is added; if the memory file enumerates migrations or version transitions, record this one.
- `pipeline/schemas.md`: (modify, if applicable) If `.status.yaml`/config schemas are catalogued here, note the `agent.tiers` addition (config-only; `.status.yaml` is unchanged by this feature).

(Final Affected Memory set is reconciled during hydrate against the actual diff.)

## Impact

- **Go binary (`src/go/fab/`):** new `internal/config` fields (`AgentConfig.Tiers`), a new tier/stage-mapping package or additions to an existing one, the `fab resolve-agent` cobra command (`cmd/fab/`), and tests (including the drift-guard test). Per the constitution, Go CLI changes MUST include test updates and update `_cli-fab.md`.
- **Skills (`src/kit/skills/`):** `_cli-fab.md` (new command), dispatch wiring in `fab-ff.md`/`fab-fff.md`/`fab-proceed.md`/`fab-continue.md`, dispatch contract in `_preamble.md` and/or `_pipeline.md`. Each changed skill updates its `docs/specs/skills/SPEC-*.md`.
- **Migration (`src/kit/migrations/`):** one new migration file; version bump per the migration versioning model.
- **Docs (`docs/specs/`):** `stage-models.md` (reconcile), `architecture.md` (config block).
- **No `.status.yaml` schema change** — this is config-only; the existing pipeline state machine is untouched.
- **Backward compatibility:** existing configs without `agent.tiers` are fully supported (fab-kit defaults apply). Existing foreground behavior is preserved (empty resolution = inherit).

## Open Questions

(None blocking — the design is fully settled. The items below are implementation-level details to decide during apply, recorded as graded assumptions rather than blockers.)

- Exact `fab resolve-agent` stdout shape (`model=`/`effort=` lines vs a single launch-args fragment) — recorded as a Confident assumption; the consuming skills only need a documented, stable contract.
- Whether the tier/stage-mapping tables live in a new Go package or extend `internal/config` — an implementation-structure decision for apply.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Tiers are `{model, effort}` profiles; three tiers `thinking`/`doing`/`ship` | Settled explicitly in conversation; effort is a first-class lever (spawn_command already uses --effort) | S:100 R:70 A:90 D:95 |
| 2 | Certain | Stage→tier mapping (thinking={intake,review}, doing={apply,review-pr,hydrate}, ship={ship}) is fab-owned and NOT user-overridable | Derived from an explicit dimensional analysis in conversation; user confirmed taxonomy is fab's, budget is the user's | S:100 R:55 A:90 D:90 |
| 3 | Certain | The only override surface is `agent.tiers` (tier redefinition); no `stage_tiers`, no per-stage escape hatch | User confirmed: "What the users can override is tiers… there's no need to let users override [the mapping]" | S:100 R:65 A:95 D:95 |
| 4 | Certain | Default profiles: thinking=opus-4-8/xhigh, doing=opus-4-8/high, ship=sonnet-4-6/low | User stated verbatim: "doing = opus+high. thinking=opus+xhigh. ship=sonnet+low" | S:100 R:75 A:95 D:100 |
| 5 | Certain | Haiku excluded from the default tiers (no effort param; ship needs PR-description comprehension) — not forbidden, just not a default | Confirmed against the Claude API docs (effort 400s on Haiku); ship default is Sonnet/low for diff-comprehension quality; pass-through (#9) still allows a Haiku override | S:95 R:80 A:100 D:90 |
| 6 | Certain | New `fab resolve-agent <stage>` pure-query command; orchestrators/fab-continue dispatch consume it; foreground is advisory-only | Established as the dispatch-seam injection point; foreground asymmetry is by design | S:90 R:60 A:85 D:85 |
| 7 | Certain | Migration ships a FULLY COMMENTED `agent.tiers` block; fab-kit uses built-in defaults | User: "let it be fully commented, so users know they can override. But by default, fab-kit uses its own defaults" | S:100 R:80 A:95 D:95 |
| 8 | Confident | `review` (generative→thinking) and `review-pr` (responsive→doing) are split across tiers | Reasoned distinction (discovering bugs vs fixing articulated feedback); user agreed to move review-pr to doing | S:90 R:55 A:80 D:80 |
| 9 | Certain | `resolve-agent` does NO validation — echoes model+effort verbatim; runtime/harness is the safety net | User decided explicitly: "don't validate. Keep it open so we can switch agents easily" | S:100 R:70 A:90 D:95 |
| 10 | Certain | Provider-neutral by construction in v1: config + resolution are provider-agnostic; dispatch injection is a harness-specific adapter (Claude Code = Agent tool `model` param); defaults are Claude-flavored but overridable. v1 = architecture-neutral + documented, NOT shipped/tested against a non-Claude harness | User decided explicitly ("yes" to multi-provider in v1, scoped as architecture-neutral not shipped-tested); honors Constitution Principle I | S:95 R:60 A:90 D:90 |
| 11 | Confident | `fab resolve-agent` output shape = `model=`/`effort=` lines on stdout | Consistent with byte-stable `fab resolve` query family; exact shape is an apply-time detail behind a stable contract | S:65 R:75 A:80 D:75 |
| 12 | Confident | Drift-guard test mirrors the `expected_min` `TestDocTablesMatchScoringMaps` pattern | Established project pattern for doc↔Go table parity (change-types.md) | S:80 R:80 A:90 D:85 |

12 assumptions (10 certain, 2 confident, 0 tentative, 0 unresolved).
