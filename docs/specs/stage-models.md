# Per-Stage Model Selection via Named Tiers

> **Status:** Design intent (pre-implementation, now implemented in 260613-l3ja). This spec
> captures the design for letting a project run different pipeline stages on different model
> tiers. The canonical tables are the Go maps in `src/go/fab/internal/agent/agent.go`; the two
> tables in this doc are verified mirrors of them (drift-guarded — see § Drift guard).

Fab runs a six-stage pipeline (`intake → apply → review → hydrate → ship → review-pr`). Today every
stage runs on whatever model the session was launched with — the orchestrator's foreground model, or
the model a dispatched sub-agent inherits. This feature lets a **project** assign a model **tier** to
each stage, so judgment-heavy stages (intake, review) run at a high-end model + effort while the
mechanical stage (ship) runs on a cheaper, lower-effort tier.

The control surface is deliberately small: fab owns *which* stages cluster into *which* tier (a fixed,
non-overridable taxonomy), and a project overrides only *what each tier means* (the `{model, effort}`
profile).

---

## Why this is possible now

The pipeline already dispatches most post-intake stages as **sub-agents** (see `_preamble.md`
§ Subagent Dispatch). The move to sub-agents was driven by context isolation — a six-stage autonomous
pipeline cannot fit in one context window, so each stage runs in a fresh context and returns a
structured result. That same dispatch seam is the natural injection point for a per-stage model: the
orchestrator sets the sub-agent's model **at dispatch time**.

This makes per-stage model selection fundamentally a property of **dispatched sub-agent runs**. Since
260613-fgxx collapsed the post-intake dual execution mode, **every** post-intake stage dispatches a
sub-agent — including plain `/fab-continue`, which is now a one-stage sequencer that resolves the tier
and dispatches its stage's block (`/fab-ff`, `/fab-fff`, `/fab-proceed` orchestrate the same way). So
per-stage selection applies uniformly to apply/review/hydrate regardless of which command drove them.
See § Foreground limitation for the narrow case (a stage skill run with no dispatch at all) it cannot
cover.

---

## Tiers are `{model, effort}` profiles

A tier is a **named profile of `{model, effort}`** — not a bare model. Effort is a first-class spend
dial (the project's current `spawn_command` already runs `--effort xhigh`), and what a user means by a
tier is the model *and* how hard it thinks. Bundling them keeps the tier name honest.

Three tiers form the vocabulary, grouped by **cognitive mode**:

| Tier | Cognitive mode |
|------|----------------|
| `thinking` | **Generative judgment** — intake *discovers* requirements; review *discovers* bugs. Deliberation directly buys quality. |
| `doing` | **Execution that must not err** — apply writes the diff; review-pr fixes already-articulated feedback (responsive, not generative); hydrate writes memory. |
| `fast` | **Speed on near-mechanical work** — commit/push/PR mechanics plus a faithful PR-description summary. |

### Default tier profiles

fab-kit ships a default `{model, effort}` per tier. This table is **owned by the Go binary**
(`internal/agent`), versioned with the kit, and the single place to bump when a new model ships.

| Tier | Model | Effort |
|------|-------|--------|
| `thinking` | `claude-opus-4-8` | `xhigh` |
| `doing` | `claude-opus-4-8` | `high` |
| `fast` | `claude-sonnet-4-6` | `low` |

This is the verified mirror of the `defaultTiers` map in
`src/go/fab/internal/agent/agent.go`. A drift-guard test fails if the two disagree (see § Drift guard).

**Why these defaults.** Two of three tiers are Opus; the differentiation between `thinking` and
`doing` is **effort** (`xhigh` → `high`), not model. The single model boundary sits at the bottom
(`fast` → Sonnet). Five of six stages run on Opus — quality-first, with the primary spend lever being
effort and one model downgrade at the mechanical floor. Cost-conscious projects opt the `doing` tier
down to Sonnet themselves (see § Config schema).

---

## The fixed stage → tier mapping (fab-owned, NOT overridable)

fab owns which stage belongs to which tier. The mapping is **fixed and non-overridable** — it is fab's
considered judgment from a dimensional analysis (judgment density, cost-of-error, output volume,
determinism). Users override what a tier *costs* (budget), never which stages belong to it (taxonomy).

| Stage | Tier |
|-------|------|
| `intake` | `thinking` |
| `review` | `thinking` |
| `apply` | `doing` |
| `review-pr` | `doing` |
| `hydrate` | `doing` |
| `ship` | `fast` |

This is the verified mirror of the `stageTiers` map in `src/go/fab/internal/agent/agent.go`
(drift-guarded). The mapping is exhaustive — every one of the six pipeline stages belongs to exactly
one tier.

**Critical distinction — `review` vs `review-pr`.** They share the word "review" but not the cognitive
mode. `review` is **generative** (reads a diff and discovers what's wrong from nothing → `thinking`);
`review-pr` is **responsive** (triages and fixes feedback someone else already generated → `doing`).
They are deliberately in **different tiers** — do not group them.

There is **no `stage_tiers` config** (stage→tier reassignment is not a user knob) and **no per-stage
escape hatch** (a stage cannot be pinned individually outside its tier). Disagreement with the tiering
is an upstream fab-kit issue, not a project knob.

---

## Config schema — `agent.tiers` (the ONLY override surface)

A new optional `agent.tiers` map in `fab/project/config.yaml`, under the existing `agent:` block. The
Go `Config` struct widens freely — yaml unmarshalling ignores unknown keys, so existing configs are
unaffected (the same property that made `stage_hooks` free to add).

```yaml
agent:
  spawn_command: claude --dangerously-skip-permissions --effort xhigh -n "$(basename "$(pwd)")"

  # The stage→tier mapping is OWNED BY FAB-KIT and is NOT overridable — shown
  # here only as reference so you know which stages each tier governs:
  #   thinking: intake, review            (generative judgment)
  #   doing:    apply, review-pr, hydrate  (execution that must not err)
  #   fast:     ship                       (speed on near-mechanical work)
  #
  # You override only WHAT EACH TIER MEANS (model + effort). Omit any tier to
  # use fab-kit's built-in default. fab-kit defaults today are:
  #   thinking: { model: claude-opus-4-8,   effort: xhigh }
  #   doing:    { model: claude-opus-4-8,   effort: high  }
  #   fast:     { model: claude-sonnet-4-6, effort: low   }
  tiers:
    doing: { model: claude-sonnet-4-6, effort: medium }   # example: run the doing tier cheaper
```

- Keys under `tiers:` are tier names: `thinking`, `doing`, `fast`.
- Each value is a `{model, effort, spawn_command}` object. Any field MAY be set; an omitted field
  falls back to the fab-kit default for that tier (**per-field merge**).
- A tier omitted entirely (or an absent `tiers:` block) uses fab-kit's built-in default for that tier.
- An **empty model** signals "inherit the session/orchestrator model" (today's behavior).
- **`spawn_command` (per-tier, opt-in) — the cross-harness stage-dispatch knob.** PRESENT on a
  resolved tier → that tier's stages are dispatched by **running the command** (cross-harness, e.g.
  `codex exec …`); ABSENT → **native Agent-tool dispatch** (the default). fab-kit's built-in default
  tiers carry **no** `spawn_command` — the field is populated **exclusively** from user config, so the
  default behavior is unchanged. It is **INDEPENDENT of `agent.spawn_command`** (which opens whole
  agent *sessions* for `fab operator` / `fab batch` / `fab spawn-command`): **there is NO fallback
  from a tier to `agent.spawn_command`** — the absence of a resolved tier `spawn_command` is itself
  the native-dispatch signal. (Mental model: `agent.spawn_command` hires an employee; a tier
  `spawn_command` outsources one task.) The `{model}`/`{effort}` placeholders are substituted at
  resolve time via the same `internal/spawn` template machinery. *This spec covers only the config
  schema and the `spawn=` resolution output; the dispatch that RUNS the command (`fab dispatch`) and
  the skill dispatch-seam wiring are separate follow-up changes (3c/3d).*

---

## Resolution — `fab resolve-agent <stage>`

Resolution lives in **Go**, not in the prompt — the cascade is volatile logic that would drift across
skill files if reasoned about in markdown. A pure-query command returns the concrete `{model, effort}`
for a stage; skills inject the result and reason about nothing.

```
fab resolve-agent <stage>
```

(Named `resolve-agent`, not `resolve-model`, because it resolves both the model and the effort the
agent dispatch needs.)

1. Take a stage name (`intake`/`apply`/`review`/`hydrate`/`ship`/`review-pr`).
2. Map the stage → its tier via the fixed stage→tier mapping.
3. Resolve the tier → `{model, effort, spawn_command}`: the project's `agent.tiers.<tier>` override
   **per-field merged** over fab-kit's default if present, else the default. `spawn_command` merges on
   the same footing as model/effort (override wins when set); defaults carry none.
4. **Emit verbatim — NO validation** (see § No validation). fab does not check the model, effort, or
   spawn_command against any provider's accepted set; it echoes the resolved strings as-is.
5. Output: two stdout lines, `model=<id>` and `effort=<level>`, **plus an optional third line
   `spawn=<command>`**. The `effort=` line is **omitted** when the resolved tier has no effort. An
   empty model emits an empty `model=` line (the "inherit" signal). The `spawn=` line is emitted
   **ONLY when the resolved tier carries a `spawn_command`** (mirroring the effort-omit rule); its
   **absence signals native Agent-tool dispatch**, and there is **NO fallback to
   `agent.spawn_command`** — `resolve-agent` never consults the whole-session field. The `spawn=`
   command's `{model}`/`{effort}` placeholders are substituted via `internal/spawn`'s template
   resolution (reused, not reimplemented), using the tier's own resolved model/effort — and the
   `{model}` is **always the full model ID**, even under `--alias` (see § Harness-adapter boundary).
6. **Byte-stable** for the same config (like other `fab resolve` queries). Non-zero exit only on a
   real error: an unreadable/malformed config, or an unknown stage name. A stage that resolves to a
   default is success, not an error.

---

## No validation — verbatim pass-through (provider-neutral)

`fab resolve-agent` does **NOT** validate the model or effort against any provider's accepted set. It
maps stage→tier→`{model, effort}` and **echoes both strings verbatim**, whatever they are — `xhigh`
for an Opus model, `high` for Sonnet, `reasoning_effort`-style values for a non-Claude model a project
might configure, or an empty effort. fab has no provider-specific knowledge in the resolution path.

**Rationale (Constitution Principle I — provider neutrality):** validating against Claude's effort
enum (`low/medium/high/xhigh/max`, Opus-only `xhigh`, Haiku-rejects-all) would hard-code Claude into
the resolver and bolt the door on other agents. Keeping it open — verbatim pass-through — is what lets
a project switch the underlying agent by overriding `agent.tiers` with that provider's model IDs and
effort vocabulary, with nothing in fab rejecting it. The **safety net moves from fab to the
runtime/harness**: a misconfigured pair (e.g. Claude `{model: claude-sonnet-4-6, effort: xhigh}`,
which Sonnet rejects with a 400) is *not* corrected by fab — it surfaces as a dispatch-time error.
This is the accepted tradeoff for portability. fab does **not** "degrade gracefully", drop an
incompatible effort, or warn on one — earlier design iterations proposed that; it was removed.

**For reference only** (NOT enforced by fab) — Claude's effort validity, which is why the fab-kit
*defaults* are chosen to be valid:

| Model | Accepts effort? | Valid values |
|-------|-----------------|--------------|
| Opus 4.8 | Yes | `low`, `medium`, `high`, `xhigh`, `max` |
| Sonnet 4.6 | Yes | `low`, `medium`, `high`, `max` (no `xhigh` — Opus-family only) |
| Haiku 4.5 | **No** | — (effort param returns HTTP 400) |

This table explains *why fab-kit's shipped defaults are what they are* — but it is documentation of
fab's default choices, not a rule the resolver enforces on user overrides.

### Haiku excluded from the defaults (not forbidden)

Haiku is **absent from the default tiers**, for two reasons: it has no effort parameter (passing
effort 400s), and the one stage that might want a fast/cheap model (the `ship` stage, governed by the
`fast` tier) needs faithful PR-description comprehension that Haiku does unreliably — so the `fast`
default is Sonnet/low. This is **exclusion from the defaults, not a prohibition**: a user MAY still
override a tier to Haiku (pass-through doesn't forbid it); fab just doesn't ship it as a default.

---

## Skill wiring — orchestrator/dispatch consume `fab resolve-agent`

The orchestrators (`/fab-ff`, `/fab-fff`, `/fab-proceed`, `/fab-adopt`) and `/fab-continue`'s sub-agent dispatch call
`fab resolve-agent <stage>` immediately before dispatching each stage's sub-agent, **surface** the
resolved `model=/effort=` lines (so a skipped or mis-resolved tier is visible in output rather than
silent — the available stand-in for an enforcement guard, since dispatch is harness-internal), and
apply the resolved **model AND effort** through their two seams:

- **Model → the Agent tool's `model` param.** The Agent `model` param is a hard enum of short aliases (`opus`/`sonnet`/`haiku`/`fable`) that rejects full IDs, so the model half is resolved with `fab resolve-agent <stage> --alias` — the `--alias` flag emits the Agent-tool-valid short alias directly on the `model=` line (see § Harness-adapter boundary). Empty model → omit it (inherit session/orchestrator model — today's behavior).
- **Effort → an explicit instruction in the subagent prompt.** The Agent tool has no `effort` param, so the resolved effort is injected as an imperative line in the dispatched prompt (e.g., ``Operate at `xhigh` reasoning effort for this task.``) and the sub-agent self-selects. Empty effort → omit the instruction. (The effort half is therefore **no longer dropped** — earlier wiring had no seam for it; it now rides the prompt. The clean fix, a first-class per-sub-agent effort parameter on the Agent tool, is a harness ask outside fab's control — see § Foreground limitation's scope note.)

The **`review` stage resolves once** and applies the same `{model, effort}` to BOTH reviewer sub-agents
(inward + outward) and the merge — the same model param and the same effort-prompt instruction for all
three; a consequence of stage(/tier)-granularity, documented as a known tradeoff (the mechanical merge
runs at the reviewer's tier; acceptable for config simplicity).

`_cli-fab.md` documents the `fab resolve-agent` command signature (Constitution constraint: CLI changes
MUST update `_cli-fab.md`). `architecture.md` documents the `agent.tiers` config block alongside the
existing `stage_hooks` example.

### Harness-adapter boundary (the only Claude-Code-specific layer)

Per-stage selection is **provider-neutral by construction**, not Claude-locked:

- *Portable layers (no provider knowledge):* the `agent.tiers` config schema, and the entire
  `fab resolve-agent` resolution path (stage→tier→`{model, effort}`). The resolver does no validation
  and echoes strings verbatim, so a project can switch agents by overriding `agent.tiers` with another
  provider's model IDs and effort vocabulary (`gpt-5 / reasoning_effort:high`, `gemini-* / <its-knob>`)
  and nothing in fab rejects it.
- *Harness-specific layer (the adapter):* injecting the resolved model+effort into the actual
  sub-agent dispatch is harness behavior, and the two halves use **two different seams** in Claude
  Code. **The model rides the Agent tool's `model` parameter** — a hard enum that takes a short alias
  (`opus`/`sonnet`/`haiku`/`fable`), not the full versioned id the plain resolver emits — so the model
  half is resolved with **`fab resolve-agent <stage> --alias`**, the deterministic Agent-tool adapter:
  the `--alias` flag maps the resolved full ID to its short alias on the `model=` line (prefix-matched,
  so dated variants like `claude-haiku-4-5-20251001` resolve to `haiku`; empty ⇒ empty inherit-signal;
  a non-Claude override passes through verbatim). This replaces the earlier prompt-side hand-mapping
  instruction (where the orchestrator was told to translate the id by hand on every dispatch — brittle
  and easy to fumble) with a Go-side translation that cannot be skipped. **The effort rides an
  instruction in the subagent prompt** (the Agent tool exposes no effort parameter). The skill wiring
  names both explicitly as the Claude-Code adapter, not as universal truth. This coupling is **not
  introduced by this feature** — fab's entire existing subagent-dispatch design (`_preamble.md` §
  Subagent Dispatch) is already Claude-Code-shaped. Per-stage selection is exactly as portable as fab's
  existing dispatch: no more, no less. *(The operator launcher path is the deliberate exception — it
  resolves the doing-tier profile WITHOUT `--alias`, because `spawn.WithProfile` composes a `claude`
  CLI invocation, which accepts full IDs. `WithProfile` is grammar-forgiving: it **appends**
  `--model <full-id> --effort <level>` to a plain Claude `spawn_command` (no placeholder), and
  **substitutes** the resolved values into a `{model}`/`{effort}` **template** `spawn_command` (e.g. a
  codex command) instead — all-or-nothing, an empty value dropping the placeholder's token and a
  preceding `-`-flag — so a non-Claude worker CLI is configurable without the launcher emitting
  Claude-only flags; 260702-6tmi.)*
- *Cross-harness stage dispatch (the `spawn=` adapter):* a tier's optional `spawn_command` is the
  seam for handing one stage to a **different CLI harness** (e.g. `codex exec …`) instead of a native
  Agent-tool sub-agent. When a resolved tier carries it, `fab resolve-agent` emits a third
  `spawn=<command>` line — its `{model}`/`{effort}` substituted via `internal/spawn`. This adapter is
  the **inverse aliasing rule** from the Agent-tool `model` param: the `spawn=` command **ALWAYS
  embeds the FULL model ID, never an alias**, because an external CLI's `--model` flag takes a full ID
  — CLI dispatch never aliases. So under `--alias` the `model=` line is aliased (Agent-tool half) while
  the `spawn=` line carries the full ID (CLI half). The field is **independent of** `agent.spawn_command`
  (the whole-session boundary) with **no cross-fallback** — absence of a resolved tier `spawn_command`
  is the native-dispatch signal. *v1 emits the line only; the dispatch that RUNS it (`fab dispatch`)
  and the skill wiring are separate follow-ups (3c/3d).*
- *Claude-flavored data (overridable):* fab-kit's shipped default table uses Claude model IDs/effort.
  These are documented as "fab-kit's Claude defaults," fully replaceable via `agent.tiers`.
- *v1 scope is architecture-neutral + documented — NOT shipped/tested against a non-Claude harness.* No
  per-provider default tables, no provider-detection, no non-Claude integration test. The acceptance
  proof is "a non-Claude project can override the tiers and nothing in fab rejects it," not "we ran it
  on a non-Claude harness." Shipped+tested multi-provider support is explicitly out of scope.

---

## apply↔review coupling: why apply is `doing`, not cheaper

The apply stage produces the diff the review stage critiques, so the two are **economically coupled**:
if `apply` runs on a cheaper model than `review`, a sharper reviewer bounces the cheaper executor's
work more often, driving **more rework cycles** (capped at 3 per `code-review.md`). Three expensive
review rounds can cost more than running `apply` on the capable model once. "Cheaper apply = cheaper
pipeline" is therefore *not* strictly true.

This is why `apply` stays in `doing` (Opus/high) rather than dropping to Sonnet: apply has the highest
output volume (which argues for the cheaper model), but the coupling argues louder. The savings on the
`doing` tier come from **effort** (`high`, not `xhigh`), not a model downgrade.

---

## Fable upgrade path

When Fable access lands, fab bumps the default tier→profile table in **one place** — `thinking` →
Fable/xhigh, `doing` → Opus/xhigh — and every non-overriding project upgrades for free. Note the
`doing` tier's effort also rises (`high` → `xhigh`) under Fable: the tier→profile table is fab's
curated judgment per release, not a fixed effort-per-tier-rank rule. A project that overrides a tier
opts **out** of fab's upgrade curve for that tier (correct behavior — naming it here).

---

## Foreground limitation (advisory only)

A sub-agent's model is set at dispatch time by the orchestrator. Per-stage model selection is honored
on dispatched sub-agent runs.

**Post-intake stages no longer have a foreground path (260613-fgxx).** The post-intake dual execution
mode was collapsed: apply/review/hydrate always dispatch a sub-agent, and plain `/fab-continue` is a
one-stage sequencer that resolves `fab resolve-agent <stage>` and dispatches the stage's block just
like an orchestrator. So `fab resolve-agent` applies uniformly across those stages regardless of
caller — this closes **Gap 1a** of the model-tier finding (foreground stages can't be tiered). Intake
is pre-boundary: it runs in the main session and is not tiered.

The residual advisory-only case is narrow: a stage skill genuinely run with **no dispatch at all**.
There fab cannot switch the session model mid-run, so the configured tier is **advisory only** — the
skill MAY note "this stage is configured for X; you're on Y" but MUST NOT attempt to switch models.

> **Scope note**: this section reconciles the foreground limitation with the single post-intake
> execution mode (260613-fgxx, Change A). The **effort half** of per-stage tiering — injected into the
> subagent prompt as an explicit instruction (since the Claude Code Agent tool has no effort parameter)
> — and the **compliance-visibility** behavior (surfacing the resolved `model=/effort=` at each
> dispatch site) are written in by 260613-m3d4 (Change C); see § Skill wiring above. The **lone
> residual** is a first-class per-sub-agent `effort` parameter on the Agent tool (the model-tier
> finding's Gap 2 clean fix) — a harness ask outside fab's control, deliberately not built here.

---

## Drift guard

The two tables above (§ Default tier profiles and § The fixed stage → tier mapping) are verified
mirrors of the `defaultTiers` and `stageTiers` maps in `src/go/fab/internal/agent/agent.go`. The Go
maps are canonical. A test in that package (`TestDocTablesMatchAgentMaps`) parses both tables from this
doc and fails if either disagrees with the code — same pattern as `TestDocTablesMatchScoringMaps` for
`docs/specs/change-types.md`.

---

## Out of scope (deferred)

- **User (`~/.fab-kit`) config layer** — explicitly dropped.
- **Role-granular keys** (`review.inward`, `review.merge`) — deferred; the stage/tier is the unit.
- **Per-invocation `--model-<stage>` flags** on the orchestrators — deferred.
- **Cost/latency telemetry** per tier — out of scope; this is selection only.
- **Shipped/tested multi-provider support** — out of scope; v1 proves architecture-neutrality only.
