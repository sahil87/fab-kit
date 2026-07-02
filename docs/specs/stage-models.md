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
non-overridable taxonomy), and a project overrides only *what each tier means* (the
`{provider, model, effort}` profile).

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

## Tiers are `{provider, model, effort}` profiles

A tier is a **named profile of `{provider, model, effort}`** — not a bare model. The invocation
**command** does NOT live on the tier: it lives on the **provider** the tier names (see
[§ Providers](#providers)), so a tier is pure budget/role policy. Effort is a first-class spend dial,
and what a user means by a tier is the provider, the model, *and* how hard it thinks. Bundling them
keeps the tier name honest.

Five **role tiers** form the vocabulary — concrete referents ("the operator", "the reviewer"), not
cognitive modes:

| Tier | Role |
|------|------|
| `default` | Spawned worker sessions (`fab batch`), `fab agent` with no tier, intake (advisory only — foreground). Also the **per-field fallback for every other tier**. |
| `operator` | The operator coordinator session (`fab operator`). |
| `doing` | **Execution that must not err** — apply writes the diff; review-pr fixes already-articulated feedback; hydrate writes memory. |
| `review` | **The critic** — review reads a diff and discovers what's wrong. Split from `doing` for author/critic separation (a different agent checks the work than does it). |
| `fast` | **Speed on near-mechanical work** — ship's commit/push/PR mechanics plus a faithful PR-description summary. |

`thinking` is **removed**, not split: with `review` its own tier, `thinking`'s only remaining stage
would be intake, which never dispatches (it is pre-boundary, foreground). Intake rides `default`,
honestly — it runs wherever the interactive session runs.

### Default tier profiles

fab-kit ships a default `{provider, model, effort}` per tier. This table is **owned by the Go binary**
(`internal/agent`), versioned with the kit, and the single place to bump when a new model ships.
Provider is written explicitly on every line (documented style — per-line readability; inheritance is
the safety net, not the style).

| Tier | Provider | Model | Effort |
|------|----------|-------|--------|
| `default` | `claude` | `claude-fable-5` | `xhigh` |
| `operator` | `claude` | `claude-sonnet-5` | `medium` |
| `doing` | `claude` | `claude-opus-4-8` | `xhigh` |
| `review` | `claude` | `claude-fable-5` | `xhigh` |
| `fast` | `claude` | `claude-sonnet-5` | `low` |

This is the verified mirror of the `defaultTiers` map in
`src/go/fab/internal/agent/agent.go`. A drift-guard test fails if the two disagree (see § Drift guard).

**Why these defaults.** `doing` runs Opus (the coupled apply/review-pr/hydrate work — see
§ apply↔review coupling); `default` and `review` run Fable at `xhigh` (the Fable upgrade curve);
`operator` runs Sonnet/medium (highest-volume coordinator, pattern-matching work, escalation
discipline makes the cheaper model safe); `fast` sits at the mechanical floor on Sonnet/low.
Cost-conscious projects opt any tier down themselves (see § Config schema).

---

## The fixed stage → tier mapping (fab-owned, NOT overridable)

fab owns which stage belongs to which tier. The mapping is **fixed and non-overridable** — it is fab's
considered judgment from a dimensional analysis (judgment density, cost-of-error, output volume,
determinism). Users override what a tier *costs* (budget), never which stages belong to it (taxonomy).

| Stage | Tier |
|-------|------|
| `intake` | `default` |
| `review` | `review` |
| `apply` | `doing` |
| `review-pr` | `doing` |
| `hydrate` | `doing` |
| `ship` | `fast` |

This is the verified mirror of the `stageTiers` map in `src/go/fab/internal/agent/agent.go`
(drift-guarded). The mapping is exhaustive — every one of the six pipeline stages belongs to exactly
one tier. `intake → default` is **advisory only**: intake runs foreground in the user's own session,
which fab cannot re-model (see § Foreground limitation).

**Critical distinction — `review` vs `review-pr`.** They share the word "review" but not the role.
`review` is **the critic** (reads a diff and discovers what's wrong from nothing → its own `review`
tier); `review-pr` is **responsive** (triages and fixes feedback someone else already generated →
`doing`). They are deliberately in **different tiers** — do not group them.

There is **no `stage_tiers` config** (stage→tier reassignment is not a user knob) and **no per-stage
escape hatch** (a stage cannot be pinned individually outside its tier). Disagreement with the tiering
is an upstream fab-kit issue, not a project knob.

---

## Providers

The invocation **command grammar** lives in a top-level `providers:` table, not on the tiers. Each
provider is an opaque, user-chosen name mapping to up to two command fields:

- **`session_command`** — opens an interactive agent **session** (`fab operator` / `fab batch` /
  `fab agent`). This is the relocated `agent.spawn_command`.
- **`dispatch_command`** — runs ONE headless **stage task** via `fab dispatch`. **ABSENT
  `dispatch_command` = native Agent-tool dispatch** (the default). There is **NO fallback** between the
  two fields — absence of `dispatch_command` signals native dispatch, never "use `session_command`".

The two fields are deliberately **not merged** into one `command`: session and dispatch are different
invocations of the same binary (claude interactive `-n` vs headless `-p`; codex TUI vs `codex exec`),
and no single template expresses both. fab-kit ships the **`claude` provider as the built-in default**
(session command shown below, no `dispatch_command` → native). A project extends/overrides via its own
`providers:` block, per-field merged over the built-in.

**Provider names are opaque — fab NEVER infers a provider from a model string** (`claude-*` → claude
would need a provider registry, which the no-validation/provider-neutrality contract refuses). The one
footgun is documented, not validated: **override a tier's `model` cross-provider ⇒ override its
`provider` too**.

## Config schema — `providers:` + `agent.tiers` (the override surfaces)

Both are optional maps in `fab/project/config.yaml`. The Go `Config` struct widens freely — yaml
unmarshalling ignores unknown keys, so existing configs are unaffected (the same property that made
`stage_hooks` free to add).

```yaml
providers:
  claude:
    session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")"'
    # dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'   # uncomment to flip claude's stages from native Agent-tool dispatch to headless CLI
  # codex:
  #   session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
  #   dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'
  # gemini:
  #   session_command: 'gemini -m {model}'
  #   dispatch_command: 'gemini -m {model}'   # no {effort} flag; no -p (fab dispatch pipes the prompt to stdin)

agent:
  # The stage→tier mapping is OWNED BY FAB-KIT and is NOT overridable — shown
  # here only as reference so you know which stages each tier governs:
  #   default:  intake (advisory), fab batch, fab agent   (+ per-field fallback)
  #   operator: fab operator (coordinator session)
  #   doing:    apply, review-pr, hydrate                 (execution that must not err)
  #   review:   review                                    (the critic)
  #   fast:     ship                                      (near-mechanical work)
  #
  # You override only WHAT EACH TIER MEANS (provider + model + effort). Omit any
  # tier to use fab-kit's built-in default. fab-kit defaults today are:
  #   default:  { provider: claude, model: claude-fable-5,  effort: xhigh }
  #   operator: { provider: claude, model: claude-sonnet-5, effort: medium }
  #   doing:    { provider: claude, model: claude-opus-4-8, effort: xhigh }
  #   review:   { provider: claude, model: claude-fable-5,  effort: xhigh }
  #   fast:     { provider: claude, model: claude-sonnet-5, effort: low }
  tiers:
    doing: { provider: claude, model: claude-sonnet-5, effort: medium }   # example: run doing cheaper
```

- Keys under `tiers:` are the five role-tier names: `default`, `operator`, `doing`, `review`, `fast`.
- Each value is a `{provider, model, effort}` object (the command lives on the provider). Any field MAY
  be set; an omitted field falls back to the project's `default` tier, then fab-kit's built-in for that
  tier (**per-field merge with default-tier inheritance**).
- A tier omitted entirely (or an absent `tiers:` block) uses fab-kit's built-in default for that tier.
- An **empty model** signals "inherit the session/orchestrator model" once resolution bottoms out.
- **Provider is written explicitly on every tier line** (documented style — per-line readability);
  inheritance is the safety net, not the style. Inheriting `{provider, model, effort}` is safe
  *because commands moved to `providers:`* — the dangerous cross-semantics command inheritance can no
  longer happen.
- The `{model}`/`{effort}` placeholders in a provider command are substituted at resolve time via the
  same `internal/spawn` template machinery. *This spec covers the config schema and the `dispatch=`
  resolution output; the dispatch that RUNS a `dispatch_command` (`fab dispatch`) and the skill
  dispatch-seam wiring share the cross-adapter contract fixed by
  [`harness-adapters.md`](harness-adapters.md).*

---

## Resolution — `fab resolve-agent <stage|tier>`

Resolution lives in **Go**, not in the prompt — the cascade is volatile logic that would drift across
skill files if reasoned about in markdown. A pure-query command returns the concrete
`{provider, model, effort}` for a stage (or tier); skills inject the result and reason about nothing.

```
fab resolve-agent <stage|tier> [--alias]
```

(Named `resolve-agent`, not `resolve-model`, because it resolves the provider, the model, and the
effort the agent dispatch needs.)

1. Take a **stage** name (`intake`/`apply`/`review`/`hydrate`/`ship`/`review-pr`) or a **role-tier**
   name (`default`/`operator`/`doing`/`review`/`fast`) — the two sets are disjoint, so the positional
   argument accepts either. A stage maps through the fixed stage→tier mapping; a tier resolves directly
   (the path `fab agent` and the operator launcher use).
2. Resolve the tier → `{provider, model, effort}`: the project's `agent.tiers.<tier>` override
   **per-field merged** over the project's `default` tier, over fab-kit's built-in. Any field wins in
   that order.
3. **Emit verbatim — NO validation** (see § No validation). fab does not check the provider, model, or
   effort against any provider's accepted set; it echoes the resolved strings as-is.
4. Output: a `model=<id>` line always, then optional `effort=<level>`, `provider=<name>`, and
   `dispatch=<command>` lines. The `effort=`/`provider=` lines are **omitted** when empty. An empty
   model emits an empty `model=` line (the "inherit" signal). The `dispatch=` line is emitted **ONLY
   when the resolved tier's provider carries a `dispatch_command`** (mirroring the effort-omit rule);
   its **absence signals native Agent-tool dispatch**, and there is **NO fallback to a session
   command**. The `dispatch=` command's `{model}`/`{effort}` placeholders are substituted via
   `internal/spawn`'s template resolution (reused, not reimplemented), using the tier's own resolved
   model/effort — and the `{model}` is **always the full model ID**, even under `--alias` (see
   § Harness-adapter boundary).
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
resolved `model=/effort=/provider=/dispatch=` lines (so a skipped or mis-resolved tier — or a CLI
dispatch — is visible in output rather than silent, the available stand-in for an enforcement guard
since dispatch is harness-internal), and apply the resolved **model AND effort** through their two
seams:

- **Model → the Agent tool's `model` param.** The Agent `model` param is a hard enum of short aliases (`opus`/`sonnet`/`haiku`/`fable`) that rejects full IDs, so the model half is resolved with `fab resolve-agent <stage> --alias` — the `--alias` flag emits the Agent-tool-valid short alias directly on the `model=` line (see § Harness-adapter boundary). Empty model → omit it (inherit session/orchestrator model — today's behavior).
- **Effort → an explicit instruction in the subagent prompt.** The Agent tool has no `effort` param, so the resolved effort is injected as an imperative line in the dispatched prompt (e.g., ``Operate at `xhigh` reasoning effort for this task.``) and the sub-agent self-selects. Empty effort → omit the instruction. (The effort half is therefore **no longer dropped** — earlier wiring had no seam for it; it now rides the prompt. The clean fix, a first-class per-sub-agent effort parameter on the Agent tool, is a harness ask outside fab's control — see § Foreground limitation's scope note.)

The **`review` stage resolves once** (on its own `review` tier) and applies the same
`{provider, model, effort}` to BOTH reviewer sub-agents (inward + outward) and the merge — the same
model param and the same effort-prompt instruction for all three; a consequence of
stage(/tier)-granularity, documented as a known tradeoff (the mechanical merge runs at the reviewer's
tier; acceptable for config simplicity).

`_cli-fab.md` documents the `fab resolve-agent` command signature (Constitution constraint: CLI changes
MUST update `_cli-fab.md`). `architecture.md` documents the `providers:` + `agent.tiers` config blocks
alongside the existing `stage_hooks` example.

### Harness-adapter boundary (the only Claude-Code-specific layer)

Per-stage selection is **provider-neutral by construction**, not Claude-locked:

- *Portable layers (no provider knowledge):* the `providers:` + `agent.tiers` config schema, and the
  entire `fab resolve-agent` resolution path (stage→tier→`{provider, model, effort}`). The resolver
  does no validation and echoes strings verbatim, so a project can switch agents by adding a provider
  and overriding `agent.tiers` with another provider's model IDs and effort vocabulary
  (`gpt-5 / reasoning_effort:high`, `gemini-* / <its-knob>`) and nothing in fab rejects it.
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
  resolves the **operator**-tier profile WITHOUT `--alias`, because `spawn.WithProfile` composes a
  `claude` CLI invocation, which accepts full IDs. `WithProfile` is grammar-forgiving: it **appends**
  `--model <full-id> --effort <level>` to a plain Claude `session_command` (no placeholder), and
  **substitutes** the resolved values into a `{model}`/`{effort}` **template** `session_command` (e.g. a
  codex command) instead — all-or-nothing, an empty value dropping the placeholder's token and a
  preceding `-`-flag — so a non-Claude worker CLI is configurable without the launcher emitting
  Claude-only flags; 260702-6tmi.)*
- *Cross-harness stage dispatch (the `dispatch=` adapter):* a provider's optional `dispatch_command` is
  the seam for handing one stage to a **different CLI harness** (e.g. `codex exec …`) instead of a
  native Agent-tool sub-agent. When a resolved tier's provider carries it, `fab resolve-agent` emits a
  `dispatch=<command>` line — its `{model}`/`{effort}` substituted via `internal/spawn`. This adapter is
  the **inverse aliasing rule** from the Agent-tool `model` param: the `dispatch=` command **ALWAYS
  embeds the FULL model ID, never an alias**, because an external CLI's `--model` flag takes a full ID
  — CLI dispatch never aliases. So under `--alias` the `model=` line is aliased (Agent-tool half) while
  the `dispatch=` line carries the full ID (CLI half). The field is **independent of** a provider's
  `session_command` (which opens whole sessions) with **no cross-fallback** — absence of a resolved
  provider `dispatch_command` is the native-dispatch signal. *`fab resolve-agent` emits the line; the
  dispatch that RUNS it (`fab dispatch`) and the skill dispatch-seam wiring that consumes it both
  shipped.* **The
  native Agent-tool adapter described in this section is now one of *two* dispatch adapters catalogued
  in [`harness-adapters.md`](harness-adapters.md)** — the CLI adapter (`fab dispatch`, 3c) is the
  other, and that spec fixes the cross-adapter dispatch protocol (dispatch-prompt obligations, the
  five-state machine, `review` nesting degradation, hooks-enhance-never-own) both share; the skill
  dispatch-seam wiring against it lives in `_preamble.md` § CLI-Adapter Dispatch + § Dispatch-Prompt
  Obligations (3d).
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

Fable has landed: `default` and `review` now run `claude-fable-5`/`xhigh`, and `doing` runs
Opus/`xhigh` (its effort rose to `xhigh` on the Fable curve). fab bumps the default tier→profile table
in **one place** (the `defaultTiers` map) each release, and every non-overriding project upgrades for
free. The tier→profile table is fab's curated judgment per release, not a fixed effort-per-tier-rank
rule. A project that overrides a tier opts **out** of fab's upgrade curve for that tier (correct
behavior — naming it here).

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
