# Per-Stage Model Selection via Agent Roles

> **Status:** Design intent (pre-implementation, now implemented — 260613-l3ja, reshaped in
> 260806-j9nh). This spec captures the design for letting a project run different pipeline stages
> on different models. The canonical data is `src/go/fab/internal/agent/defaults.yaml` (the embedded
> depth knobs and provider table, including claude's per-role fills) plus the `stageRoles` and
> `roleDepth` maps in `src/go/fab/internal/agent/agent.go`; the two tables in this doc are verified
> mirrors of them (drift-guarded — see § Drift guard).

Fab runs a six-stage pipeline (`intake → apply → review → hydrate → ship → review-pr`). Without this
feature every stage runs on whatever model the session was launched with — the orchestrator's
foreground model, or the model a dispatched sub-agent inherits. This feature lets a **project**
assign a provider and model to each stage, so judgment-heavy stages (review) run at a high-end model
+ effort while the mechanical stage (ship) runs cheaper.

The control surface is deliberately small, and it is **two knobs**: pick a provider for the agents
you talk to, and a provider for the agents the pipeline dispatches. Everything beneath that — which
stages cluster into which role, which model each role gets on each provider — is fab-owned default
or a sparse escape hatch.

---

## Vocabulary

Four words, each with exactly one meaning:

| Term | Meaning |
|------|---------|
| **role** | One of six fixed slot names — `default`, `operator`, `doing`, `review`, `hydrate`, `fast`. A role is *what an agent is for*. |
| **profile** | A concrete `{provider, model, effort}` value. What a role resolves *to*. |
| **provider** | A named set of independent launch capabilities (`session_command`, `dispatch_command`, and `native`) plus its per-role fills. Capability presence says how a rung runs; `dispatch.mode` chooses the preference ceiling and descends `pane → native → headless`. |
| **Tier 1 / Tier 2** | Agent **depth**: Tier 1 is the agents a user talks to (a `fab agent` session, the operator, a `fab batch` worker); Tier 2 is the agents pipeline stages dispatch to. |

"Tier" means depth and *only* depth. A watchable pane worker is still Tier 2 — the defining property
is "owes a result artifact and owns no transitions", not "never spoken to".

---

## Why this is possible now

The pipeline already dispatches most post-intake stages as **sub-agents** (see `_preamble.md`
§ Subagent Dispatch). The move to sub-agents was driven by context isolation — a six-stage autonomous
pipeline cannot fit in one context window, so each stage runs in a fresh context and returns a
structured result. That same dispatch seam is the natural injection point for a per-stage model: the
orchestrator sets the sub-agent's model **at dispatch time**.

This makes per-stage model selection fundamentally a property of **dispatched sub-agent runs**. Since
260613-fgxx collapsed the post-intake dual execution mode, **every** post-intake stage dispatches a
sub-agent — including plain `/fab-continue`, which is now a one-stage sequencer that resolves the role
and dispatches its stage's block (`/fab-ff`, `/fab-fff`, `/fab-proceed` orchestrate the same way). So
per-stage selection applies uniformly to apply/review/hydrate regardless of which command drove them.
See § Foreground limitation for the narrow case (a stage skill run with no dispatch at all) it cannot
cover.

---

## The two knobs — `agent.session` and `agent.workers`

The advertised config surface is exactly two scalars, each naming a provider:

```yaml
agent:
  session: claude    # Tier 1 — what you talk to: fab agent, fab operator, fab batch
  workers: gemini    # Tier 2 — what pipeline stages dispatch to
```

Both default to `claude`, so a fresh install writes nothing. The **role → depth partition** is fixed
and fab-owned:

| Depth | Knob | Roles |
|-------|------|-------|
| Tier 1 (session) | `agent.session` | `default`, `operator` |
| Tier 2 (workers) | `agent.workers` | `doing`, `review`, `hydrate`, `fast` |

**The split is mechanically real, not cosmetic.** `agent.session` applies at **launch time** — fab
cannot switch a running session's provider — while `agent.workers` applies at **every stage dispatch**
(`fab resolve-agent`). `intake` rides `default` and is therefore a session role for exactly that
reason: it runs foreground in the user's own session.

Both keys are scope `both`, so "claude for what I talk to, gemini for the workers" is settable once
per machine in `~/.fab-kit/config.yaml`.

---

## Roles are the referent taxonomy; profiles are what they resolve to

A profile is a **named triple of `{provider, model, effort}`** — not a bare model. The invocation
**command** does NOT live on the role: it lives on the **provider** (see [§ Providers](#providers)),
so a role is pure budget/referent policy. Effort is a first-class spend dial, and what a user means by
a role is the provider, the model, *and* how hard it thinks.

Six **roles** form the vocabulary — concrete referents ("the operator", "the reviewer"), not
cognitive modes. A role is **stage-named only where it maps 1:1 to a single referent** (`review`,
`hydrate`); `default`, `doing`, and `fast` keep role names because each is **multi-referent** (`fast`
governs the ship stage *and* the `/fab-proceed` prefix-step dispatches — see § Skill wiring):

| Role | Referent |
|------|----------|
| `default` | Spawned worker sessions (`fab batch`), `fab agent` with no role, intake (advisory only — foreground), and the `/fab-proceed` create-intake dispatch. *(`fab agent --provider <name>` deliberately resolves NO role — it is a sibling addressing mode that bypasses this taxonomy entirely; see § Providers.)* |
| `operator` | The operator coordinator session (`fab operator`). |
| `doing` | **Execution that must not err** — apply writes the diff; review-pr fixes already-articulated feedback. |
| `review` | **The critic** — review reads a diff and discovers what's wrong. Its own role (not folded into `doing`) so the critic's model and effort can be dialed independently of the author's; the separation that matters is the fresh context and adversarial framing, not a different model family. |
| `hydrate` | **Memory writing** — hydrate merges the change into `docs/memory/` as current truth. Its own role so it can run on a different model/effort than apply. |
| `fast` | **Speed on near-mechanical work** — ship's commit/push/PR mechanics plus a faithful PR-description summary, and the `/fab-proceed` prefix steps (`/fab-switch`, `/git-branch`). Multi-referent, so it keeps its role name. |

There is no `thinking` role: with `review` its own role, `thinking`'s only remaining stage would be
intake, which never dispatches (it is pre-boundary, foreground). Intake rides `default`, honestly — it
runs wherever the interactive session runs.

### Default role profiles

fab-kit ships a default `{provider, model, effort}` per role. The provider comes from the built-in
depth knobs (both `claude`); the model and effort come from **`providers.claude.profiles.<role>`**.
Both live in **`src/go/fab/internal/agent/defaults.yaml`** — a data file **embedded into the binary**
(`go:embed`, never read from the kit cache at runtime) and shaped as a config-file fragment, so it is
versioned with the kit and is the single file to edit when a new model ships.

| Role | Provider | Model | Effort |
|------|----------|-------|--------|
| `default` | `claude` | `claude-fable-5` | `high` |
| `operator` | `claude` | `claude-sonnet-5` | `medium` |
| `doing` | `claude` | `claude-opus-5` | `high` |
| `review` | `claude` | `claude-opus-5` | `high` |
| `hydrate` | `claude` | `claude-opus-5` | `high` |
| `fast` | `claude` | `claude-sonnet-5` | `medium` |

This is the verified mirror of what `defaults.yaml` composes. A drift-guard test fails if the two
disagree (see § Drift guard).

**Why these defaults.** `doing` runs Opus at `high` — Opus is the strongest coding/agentic model, and
`high` is the recommended effort sweet spot (`xhigh` buys marginal gains at disproportionate
latency/cost); a strong author minimizes rework cycles per the apply↔review coupling (see §
apply↔review coupling). `review` runs Opus/`high` — code review is a named Opus strength, and the
critic gets the same top-end model as the author so it can actually catch what the author missed
(author/critic separation is enforced by the *fresh context and adversarial framing*, not by a weaker
model). `hydrate` runs Opus/`high` — knowledge work and memory writing are named Opus strengths. Note
that on the native dispatch arm the effort half is advisory anyway (§ Effort asymmetry), so the model
choice is what actually carries. `default` runs
Fable/`high` — interactive sessions want the quicker working style (Anthropic guidance: `high` is the
sweet spot, and Fable at lower efforts still exceeds prior models' `xhigh`). `operator` runs
Sonnet/`medium` (highest-volume coordinator, pattern-matching work, escalation discipline makes the
cheaper model safe). `fast` sits at the mechanical floor on Sonnet/`medium` — effort at `medium` (not
`low`) buys margin for faithful PR-description comprehension. Cost-conscious projects opt any role down
themselves (see § Config schema).

---

## The fixed stage → role mapping (fab-owned, NOT overridable)

fab owns which stage belongs to which role. The mapping is **fixed and non-overridable** — it is fab's
considered judgment from a dimensional analysis (judgment density, cost-of-error, output volume,
determinism). Users override what a role *costs* (budget), never which stages belong to it (taxonomy).

| Stage | Role |
|-------|------|
| `intake` | `default` |
| `review` | `review` |
| `apply` | `doing` |
| `review-pr` | `doing` |
| `hydrate` | `hydrate` |
| `ship` | `fast` |

This is the verified mirror of the `stageRoles` map in `src/go/fab/internal/agent/agent.go`
(drift-guarded). The mapping is exhaustive — every one of the six pipeline stages belongs to exactly
one role. `intake → default` is **advisory only**: intake runs foreground in the user's own session,
which fab cannot re-model (see § Foreground limitation). Where a stage and a role share a name
(`review`, `hydrate`), the stage maps to that same-named role — see § Resolution's fixed-point rule.
(`ship` is a stage but not a role — it maps to the multi-referent `fast` role.)

**Critical distinction — `review` vs `review-pr`.** They share the word "review" but not the role.
`review` is **the critic** (reads a diff and discovers what's wrong from nothing → its own `review`
role); `review-pr` is **responsive** (triages and fixes feedback someone else already generated →
`doing`). They are deliberately in **different roles** — do not group them.

There is **no `stage_roles` config** (stage→role reassignment is not a user knob), **no per-stage
escape hatch** (a stage cannot be pinned individually outside its role), and **no user-overridable
role→depth partition**. Disagreement with the taxonomy is an upstream fab-kit issue, not a project
knob.

---

## Providers

The launch **capability grammar** lives in a top-level `providers:` table, not on the roles. Each
provider is an opaque, user-chosen name mapping to three independent dispatch capabilities plus a per-role fill map:

- **`session_command`** — opens an interactive agent **session** (`fab operator` / `fab batch` /
  `fab agent`). It is reachable **two ways**: through a role (the role names a provider, and its
  `{model, effort}` are substituted) or **directly**, via
  `fab agent --provider <name> [--model <id>] [--effort <level>]`, which bypasses role resolution
  entirely — a provider-addressed spawn for the "give me a codex session right here" case, where no
  role need name the provider first. The direct form is a **lookup**, not a new validation surface: an
  unknown name errors listing the available providers, while resolved command strings still pass
  through verbatim. See `_cli-fab.md` § fab agent.
- **`dispatch_command`** — runs ONE headless **stage task** via `fab dispatch`.
- **`native`** — boolean capability for the in-harness Agent-tool adapter. Provider names are opaque,
  so fab never infers native support from a name or model.
- **`profiles.<role>`** — the provider's **per-role fill**: `{model, effort}` for "when this provider
  plays this role". Keyed by role name; the `default` entry doubles as the provider's **cross-role
  fallback**. Scope `both`, so a machine-wide fill is settable once in `~/.fab-kit/config.yaml`.

The capabilities are deliberately independent. Session and headless dispatch are different
invocations of the same binary (claude interactive `-n` vs headless `-p`; codex TUI vs `codex exec`),
and native is not a command at all. Presence describes **how** a rung runs; it never chooses policy.
`dispatch.mode` owns preference, and no command field substitutes for another.

**Per-role fills, not one fill per provider.** A single `{model, effort}` per provider would resolve
the *same* model for every role, so swapping `agent.workers` to another provider would flatten the
whole role taxonomy. Keyed by role, the swap keeps the differentiation: `apply` gets that provider's
strongest model, `ship` its cheapest.

### Three built-in providers

fab-kit ships **three built-in providers** — `claude` (the default), `codex`, and `gemini` — in the
`providers:` block of `internal/agent`'s embedded `defaults.yaml`:

```yaml
providers:
  claude:
    native: true
    session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}'
    dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'
    profiles:
      default:  { model: claude-fable-5,  effort: high }
      operator: { model: claude-sonnet-5, effort: medium }
      doing:    { model: claude-opus-5,   effort: high }
      review:   { model: claude-opus-5,   effort: high }
      hydrate:  { model: claude-opus-5,   effort: high }
      fast:     { model: claude-sonnet-5, effort: medium }
  codex:
    session_command: 'codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
    dispatch_command: 'codex exec --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
    profiles:                                 # sparse — an absent role takes `default`
      default: { model: gpt-5.6-sol,  effort: high }
      doing:   { effort: xhigh }
      review:  { effort: xhigh }
      fast:    { model: gpt-5.6-luna, effort: low }
  gemini:
    session_command: 'gemini --approval-mode=yolo -m {model}'
    dispatch_command: 'gemini --approval-mode=yolo -m {model}'
    profiles:                                 # model-only: no reasoning-effort flag
      default: { model: pro }
      fast:    { model: flash }
```

**All three carry fills.** The maps are SPARSE for the non-claude providers: a role absent from a
provider's map resolves that provider's `default` entry, so codex's `hydrate` and `operator` land on
codex's `default` model and effort, and gemini's non-`fast` roles on gemini's `default` model, without
a row of their own. (The merge is per FIELD, so codex's `doing`/`review` rows — effort only — take
their model from `default` too.)

**All three also ship a full-auto posture.** Claude uses `--dangerously-skip-permissions`, Codex uses
`--dangerously-bypass-approvals-and-sandbox`; Gemini uses the non-deprecated
`--approval-mode=yolo` spelling. Pipeline workers are unattended and have no channel for answering
approval prompts, so both the interactive session command (used by pane mode) and headless dispatch
command carry the provider's bypass flag. Users who require approval-gated workers override the
corresponding `providers.<name>.session_command` or `.dispatch_command`.

Two per-provider shapes are load-bearing:

- **gemini's fills carry no `effort`** — that CLI has no reasoning-effort flag, the same reason its
  command grammars carry no `{effort}` placeholder. A gemini-resolved role emits no `effort=` line.
- **gemini's fills are the CLI's own stable ALIASES (`pro`, `flash`), not versioned IDs.** `gemini -m
  pro` resolves to whatever that CLI's current best pro model is for the caller's entitlement, and
  degrades gracefully without preview access — so those two rows do not rot at all. The codex CLI
  exposes no alias mechanism (`-m` takes a slug), so its rows are pinned IDs and are the ones the
  refresh policy below is about.
  The alias mechanism is a **gemini-CLI version floor**, not a wire-protocol feature: a CLI older than
  its `resolveModel()` alias support passes `pro`/`flash` through as an unknown model ID and fails
  there. That is the same loud, one-line-fixable failure as a stale slug — override
  `providers.gemini.profiles.<role>.model` with a full versioned ID, or upgrade the CLI.

Consequences:

- **Naming `codex`/`gemini` resolves with zero `providers:` config** — `agent.workers: codex`, an
  `agent.profiles.<role>.provider`, or `fab agent --provider codex` / `fab resolve-agent <stage>
  --provider codex` all work on a fresh project. A `providers:` block is for *overriding* a grammar or
  a fill, not for registering these providers.
- **Role differentiation survives a provider swap.** `agent.workers: codex` resolves `xhigh` for
  apply/review and codex's cheaper `fast` model at `low` for ship — which is the whole point of keying
  fills by role. Shipping no fills resolved an *empty* model identically for all four, silently
  flattening the taxonomy.
- **A built-in provider is inert until named.** Adding the rows changes no default behavior (both
  depth knobs ship `claude`), which is why presence=intent — the rule that keeps behavior-changing
  config commented — does not force the table out of Go.
- **Claude carries all three capabilities; codex/gemini carry both command fields and are non-native.**
  Under the default `dispatch.mode: native`, claude resolves native while codex/gemini descend to
  headless. Adding a command never changes policy by itself.
- The reference renders the codex/gemini blocks **commented**, like every other non-overridden
  default (a commented block registers no project override).

### Refreshing the non-claude fills

Built-in non-claude fills are **refreshed at kit-release cadence**, by editing `defaults.yaml`. There
is no staleness automation: fab runs no CI check against provider APIs and no model-catalog fetch —
resolution passes every string through **verbatim** (the no-validation contract; compatibility is the
harness's concern).

That is safe because of the failure shape on each side:

- **A stale ID fails loudly and cheaply.** The provider CLI rejects it immediately, and the fix is one
  config line: `providers.<name>.profiles.<role>.model`. Because `providers` is scope `both`, that
  line is settable once machine-wide in `~/.fab-kit/config.yaml`.
- **Shipping nothing failed silently.** An empty model resolved the CLI's own default identically for
  every role, with no error at all — worse than a loud stale ID, because it defeats role
  differentiation exactly where a user first exercises the knob.
- **A bump is a data diff.** The fills are rows in an embedded YAML file, reviewable at a glance, and
  pinned by `TestNonClaudeProviderFillsArePinned` so the edit is always test-acknowledged.

Verify a fill against the **installed** binary rather than from memory — `_cli-agents.md` § discovery
recipes records what to run per CLI. The fills are never seeded into a user's `config.yaml`: they live
in the binary, so an upgrade refreshes them and no project pins rot in place.

> **Decision lineage — `260731-ho9y` → `260805-j3cm` → `260806-ywkx`.** ho9y shipped codex/gemini as
> *uncomment-to-opt-in template text* and recorded "no new built-in providers are added in Go". j3cm
> reversed that narrowly, for **grammar strings only**, explicitly keeping model IDs out ("non-claude
> model IDs rot at CLI cadence, so they belong in config, not in a release"). ywkx completes the
> reversal: fills ship too. What survives from ho9y is its actual rule — presence implies intent, so
> anything whose presence changes behavior ships commented — and it is *preserved*, because a built-in
> provider is inert until a knob, role override, or flag names it. What is retired is the rot argument,
> on the four grounds above plus release cadence: fab-kit ships every few days, so users see refreshed
> suggestions at kit cadence rather than CLI cadence.

**Provider names are opaque — fab NEVER infers a provider from a model string** (`claude-*` → claude
would need a provider registry, which the no-validation/provider-neutrality contract refuses).

### Fill precedence

```
provider:       invocation --provider
                >  agent.profiles.<role>.provider
                >  the role's depth knob (agent.session | agent.workers)
                >  the built-in claude

model / effort  invocation flag
(per field):    >  agent.profiles.<role>.<field>
                >  providers.<p>.profiles.<role>.<field>
                >  providers.<p>.profiles.default.<field>
                >  empty
```

`<p>` is the **resolved** provider, so a provider swap re-derives model and effort from the new
provider's own fills. `empty` keeps its existing meaning: an empty `model=` line is the "inherit the
session model" signal, and on a command the placeholder's token (plus a preceding `-`-flag) is dropped
by `spawn.WithProfile`, so the CLI's own default applies.

**There is exactly ONE cross-role fallback chain, and it lives on the provider side.** `agent.profiles`
is sparse: an unset field is simply not an override, and `agent.profiles.default` is the `default`
**role's** own override — never a fallback source for the other five. Cross-role fallback is
`providers.<p>.profiles.default`.

**Why one chain.** Two competing chains are what made a cross-provider *cutoff rule* necessary in the
first place: with agent-side inheritance, `{provider: codex}` on a role would inherit a **claude**
model through the same per-field merge, and fab had to detect and cut that. With model/effort always
sourced from the resolved provider's own fills, **no value can be foreign to the provider that will run
it**, so the cutoff rule, its per-field ownership tracking, and its cross-scope limitation are all
gone.

The one value that *does* survive a provider swap is an explicit `agent.profiles.<role>.model` — a pin
the user wrote by hand. That is not inheritance; it is the user's own escape hatch, and removing it on
a swap would make the field unusable.

> **Retired with the cutoff: the cross-scope ownership limitation.** The cutoff computed per-field
> ownership over the *merged* config, so it could not fire across the system↔project scope boundary —
> a documented limitation pinned by `TestResolveCrossScopeCascadeLimitation`. Both the rule and the
> pin are deleted: with no ownership computation there is no scope-boundary gap to document.

## Config schema — the two knobs, `agent.profiles`, and `providers:`

All three are optional in `fab/project/config.yaml`. The Go `Config` struct widens freely — yaml
unmarshalling ignores unknown keys, so existing configs are unaffected (the same property that made
`stage_hooks` free to add).

```yaml
agent:
  session: claude          # Tier 1 — fab agent / fab operator / fab batch   (roles: default, operator)
  workers: claude          # Tier 2 — the pipeline's stage workers            (roles: doing, review, hydrate, fast)

  # The sparse per-role escape hatch. Every field optional; a set field beats
  # the knob (provider) or the provider's own fill (model/effort). NO cross-role
  # inheritance — `default` here is the default ROLE, not a fallback.
  profiles:
    review: { provider: codex }                       # e.g. run just the critic elsewhere
    fast:   { model: claude-haiku-4-5, effort: low }  # e.g. run ship cheaper

providers:
  claude:
    native: true
    session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}'
    dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'
  # codex and gemini are BUILT-IN providers carrying grammar AND fills — these
  # blocks merely restate a built-in default, so they ship commented like every
  # other default. Uncomment only to OVERRIDE a grammar or pin a newer model.
  # (Shape only below; § Three built-in providers above carries the live values,
  # and `fab config explain` prints what your binary actually ships.)
  # codex:
  #   session_command: 'codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
  #   dispatch_command: 'codex exec --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
  #   profiles:
  #     default: { model: <codex-model-id>, effort: high }   # e.g. — pin a newer model here
  # gemini:
  #   session_command: 'gemini --approval-mode=yolo -m {model}'
  #   dispatch_command: 'gemini --approval-mode=yolo -m {model}'   # no {effort} flag; no -p (fab dispatch pipes the prompt to stdin)
  #   profiles:
  #     default: { model: <gemini-model-id> }  # e.g. — no effort: the gemini CLI has no such flag

dispatch:
  mode: native              # preference ceiling: pane → native → headless
```

- Keys under `agent.profiles:` are the six role names: `default`, `operator`, `doing`, `review`,
  `hydrate`, `fast`. Keys under `providers.<name>.profiles:` are the same six.
- Each `agent.profiles` value is a `{provider, model, effort}` object (the command lives on the
  provider). Any field MAY be set; an unset field is not an override and resolution continues down the
  fill precedence.
- A role omitted entirely (or an absent `profiles:` block) uses the knob's provider and that
  provider's own fills.
- An **empty model** signals "inherit the session/orchestrator model" once resolution bottoms out.
- The `{model}`/`{effort}` placeholders in a provider command are substituted at resolve time via the
  same `internal/spawn` template machinery. *This spec covers the config schema and the `dispatch=`
  resolution output; the dispatch that RUNS a `dispatch_command` (`fab dispatch`) and the skill
  dispatch-seam wiring share the cross-adapter contract fixed by
  [`harness-adapters.md`](harness-adapters.md).*

### Advertised surface vs. full schema

The **managed fence** in every project's `config.yaml` scaffolds only the advertised fields, and on the
agent side that is exactly the two knobs (plus `dispatch.mode` / `dispatch.column_width` /
`dispatch.reap_done`).
`agent.profiles` and the whole `providers:` table are `advertise: false`: still documented in
`fab config explain` (YAML + `--json`) and in [`config.md`](config.md), but no longer ~90 commented
lines in every repo. Users pick a provider per depth; the machinery is there when they need it.

### Migrating from `agent.tiers`

`agent.tiers` was renamed `agent.profiles`, and the flat `providers.<name>.model`/`.effort` fill folded
into `providers.<name>.profiles.default`. Both old spellings are **still read** — per role for
`agent.tiers`, and as an alias for `profiles.default` for the flat fill — so a config keeps resolving
before it is rewritten. The `2.16.19-to-2.17.0` migration performs the on-disk rewrite in both the
project config and `~/.fab-kit/config.yaml`.

One semantic that the rename does **not** preserve, and the migration cannot: the old `agent.tiers`
re-based every unset field from its `default` tier. A project that relied on `agent.tiers.default.model`
to set every tier's model must now either set `agent.session`/`agent.workers` (if the intent was a
provider choice) or write the model on `providers.<name>.profiles.default` (if the intent was a model
choice).

One pre-migration wrinkle: the legacy alias resolves **after** the scope cascade, so while a config is
half-migrated the *spelling* can outrank the scope — a system-scope `agent.profiles.<role>` beats a
project-scope `agent.tiers.<role>` for that role, inverting the usual `project > system` precedence
(pinned by `TestResolveCrossScopeLegacyAliasPrecedence`). The migration sweeps both scopes, so the
window closes as soon as it runs; the retired-cutoff note above is about the *ownership* computation,
not this alias.

---

## Resolution — `fab resolve-agent <stage|role>`

Resolution lives in **Go**, not in the prompt — the cascade is volatile logic that would drift across
skill files if reasoned about in markdown. A pure-query command returns the concrete
`{provider, model, effort}` for a stage (or role); skills inject the result and reason about nothing.

```
fab resolve-agent <stage|role> [--alias] [--provider <name>] [--model <id>] [--effort <level>]
```

(Named `resolve-agent`, not `resolve-model`, because it resolves the provider, the model, and the
effort the agent dispatch needs.)

1. Take a **stage** name (`intake`/`apply`/`review`/`hydrate`/`ship`/`review-pr`) or a **role**
   name (`default`/`operator`/`doing`/`review`/`hydrate`/`fast`) — the positional argument accepts
   either. A stage maps through the fixed stage→role mapping; a role resolves directly (the path
   `fab agent` and the operator launcher use). **Role names are checked first.** The two name sets
   overlap only at **fixed points**: a name shared by a stage and a role (`review`, `hydrate`) is one
   where the stage maps to that same-named role (`stageRoles[name] == name`), so the role-first check
   resolves such a name to the same profile either interpretation would — the order is immaterial for
   results. (A drift-guard test in `internal/agent` asserts every collision is a fixed point.) `ship`
   is a stage but NOT a role — it maps to the `fast` role — so `resolve-agent ship` resolves the stage
   while `resolve-agent fast` resolves the role, both to the same profile.
2. Resolve the role → `{provider, model, effort}` through the fill precedence above: the provider from
   an `agent.profiles.<role>.provider` override, else the role's depth knob, else the built-in claude;
   then model and effort per field from an `agent.profiles.<role>` field, else that provider's
   `profiles.<role>` fill, else its `profiles.default` fill, else empty.
2a. **Apply invocation-time overrides** (`--provider`/`--model`/`--effort`) — the top rung, riding the
   same single resolution call. `--provider` swaps the provider and **re-derives `dispatch=` from
   `dispatch.mode` plus the NAMED provider's capabilities** — so the selected rung and emitted-line presence can differ from the
   stage's unoverridden one, but that is a **query result, not an adapter move**: `fab dispatch`
   re-resolves from config and accepts no overrides, so only a config override actually relocates a
   stage between the native and CLI adapters (§ Skill wiring → User-directed overrides). A swap
   re-derives an unoverridden model/effort from the new provider's own per-role fills (an explicit
   `agent.profiles.<role>` pin still wins). `--model`/`--effort` are valid **without** `--provider` (a
   within-role override) — the documented asymmetry with `fab agent`, where they stay a usage error
   without `--provider`, because `resolve-agent` is a pure query whose whole output is a profile while
   `fab agent` is a launcher with two mutually exclusive addressing modes. All three key on whether the
   flag was *supplied* (cobra's `Flag.Changed`), not on value emptiness — so an explicitly-empty
   `--provider=` resolves an empty provider (a **lookup** failure: non-zero exit naming the resolvable
   set — built-in table ∪ the project's `providers:` keys, sorted — mirroring `fab agent`'s error)
   rather than falling back to the depth knob.
3. **Emit verbatim — NO validation** (see § No validation). fab does not check the provider, model, or
   effort against any provider's accepted set; it echoes the resolved strings as-is.
4. Output: a `model=<id>` line always, then optional `effort=<level>`, `provider=<name>`, and
   `dispatch=<command>` lines. The `effort=`/`provider=` lines are **omitted** when empty. An empty
   model emits an empty `model=` line (the "inherit" signal). `dispatch=` is derived by the
   `dispatch.mode` descent ladder: native omits it; pane emits `session_command`; headless emits
   `dispatch_command`. Its presence remains the only skill-side branch, and skills never execute its
   value. The command's `{model}`/`{effort}` placeholders are substituted via
   `internal/spawn`'s template resolution (reused, not reimplemented), using the role's own resolved
   model/effort — and the `{model}` is **always the full model ID**, even under `--alias` (see
   § Harness-adapter boundary).
5. **Byte-stable** for the same config (like other `fab resolve` queries). Non-zero exit only on a
   real error: an unreadable/malformed config, an unknown stage/role name, a supplied `--provider`
   that resolves to no provider, or a configured provider with no reachable dispatch capability.
   A stage that resolves to a default is success, not an error.

### Dispatch preference — `dispatch.mode`

`dispatch.mode` is a `pane | native | headless` preference ceiling (**default `native`**, **scope
`both`** — settable once machine-wide). Resolution starts at that rung and descends only through
`pane → native → headless`, selecting the first possible adapter:

- **Pane** requires tmux (`$TMUX` at the resolver seam; a real probe at start) and `session_command`.
- **Native** requires `native: true`.
- **Headless** requires `dispatch_command`.
- Missing prerequisites skip rungs; no selection ever ascends. Thus `pane` is the watchable preference,
  `native` is quiet/in-context, and `headless` is detached-only.
- Capability presence says how, never whether. Each mode composes only its own field, and adding
  claude's headless command does not move it off native under the default preference.
- **No skill-wiring change:** absence iff native, presence for pane/headless; branch on presence and
  never execute the value. `fab dispatch start` re-resolves internally from current config/environment.
- **`--alias` is unaffected**: the `dispatch=` line always embeds the full model ID.
- If start-time re-resolution lands on native (for example tmux died after pane resolution),
  `fab dispatch` errors before writing state and tells the caller to re-run `fab resolve-agent`.

---

## No validation — verbatim pass-through (provider-neutral)

`fab resolve-agent` does **NOT** validate the model or effort against any provider's accepted set. It
maps stage→role→`{model, effort}` and **echoes both strings verbatim**, whatever they are — `xhigh`
for an Opus model, `high` for Sonnet, `reasoning_effort`-style values for a non-Claude model a project
might configure, or an empty effort. fab has no provider-specific knowledge in the resolution path.

**Rationale (Constitution Principle I — provider neutrality):** validating against Claude's effort
enum (`low/medium/high/xhigh/max`, Opus-only `xhigh`, Haiku-rejects-all) would hard-code Claude into
the resolver and bolt the door on other agents. Keeping it open — verbatim pass-through — is what lets
a project switch the underlying agent by pointing a knob at another provider and giving it per-role
fills in that provider's model IDs and effort vocabulary, with nothing in fab rejecting it. The
**safety net moves from fab to the runtime/harness**: a misconfigured pair (e.g. Claude
`{model: claude-sonnet-4-6, effort: xhigh}`, which Sonnet rejects with a 400) is *not* corrected by fab
— it surfaces as a dispatch-time error. This is the accepted tradeoff for portability. fab does **not**
"degrade gracefully", drop an incompatible effort, or warn on one — earlier design iterations proposed
that; it was removed.

**For reference only** (NOT enforced by fab) — Claude's effort validity, which is why the fab-kit
*defaults* are chosen to be valid:

| Model family | Accepts effort? | Valid values |
|--------------|-----------------|--------------|
| Opus / Fable | Yes | `low`, `medium`, `high`, `xhigh`, `max` |
| Sonnet | Yes | `low`, `medium`, `high`, `max` (no `xhigh` — Opus-family only) |
| Haiku | **No** | — (effort param returns HTTP 400) |

This table explains *why fab-kit's shipped defaults are what they are* — but it is documentation of
fab's default choices, not a rule the resolver enforces on user overrides.

### Haiku excluded from the defaults (not forbidden)

Haiku is **absent from the default fills**, for two reasons: it has no effort parameter (passing
effort 400s), and the one stage that might want a fast/cheap model (the `ship` stage, governed by the
`fast` role) needs faithful PR-description comprehension that Haiku does unreliably — so the `fast`
default is Sonnet/`medium`. This is **exclusion from the defaults, not a prohibition**: a user MAY
still point a role at Haiku (pass-through doesn't forbid it); fab just doesn't ship it as a default.

---

## Skill wiring — orchestrator/dispatch consume `fab resolve-agent`

The orchestrators (`/fab-ff`, `/fab-fff`, `/fab-proceed`, `/fab-adopt`) and `/fab-continue`'s sub-agent dispatch call
`fab resolve-agent <stage>` immediately before dispatching each stage's sub-agent, **surface** the
resolved `model=/effort=/provider=/dispatch=` lines (so a skipped or mis-resolved role — or a CLI
dispatch — is visible in output rather than silent, the available stand-in for an enforcement guard
since dispatch is harness-internal), and apply the resolved **model AND effort** through their two
seams:

- **Model → the Agent tool's `model` param.** The Agent `model` param is a hard enum of short aliases (`opus`/`sonnet`/`haiku`/`fable`) that rejects full IDs, so the model half is resolved with `fab resolve-agent <stage> --alias` — the `--alias` flag emits the Agent-tool-valid short alias directly on the `model=` line (see § Harness-adapter boundary). Empty model → omit it (inherit session/orchestrator model — today's behavior).
- **Effort → an explicit instruction in the subagent prompt.** The Agent tool has no `effort` param, so the resolved effort is injected as an imperative line in the dispatched prompt (e.g., ``Operate at `high` reasoning effort for this task.``) and the sub-agent self-selects. Empty effort → omit the instruction. See § Effort asymmetry for how far that actually carries.

### Effort asymmetry — the two arms are not equally reliable

**Model differentiation works on both dispatch arms. Effort differentiation does not.**

- On the **native Agent-tool arm**, effort rides an *instruction in the prompt* — there is no `effort`
  parameter to set. That instruction is **not reliably honored**: the session-level reasoning effort
  dominates, so a sub-agent dispatched from a `high`-effort session generally keeps running at the
  session's effort no matter what its prompt asks for. This is a known Claude Code limitation (GitHub
  issues #64033 and #39220), not a fab bug, and fab has no other seam to reach.
- On the **CLI arms** — `fab dispatch` headless and pane, and the operator launcher — effort rides
  `--effort` (or the provider's own equivalent) inside a *composed command line*, which the harness
  reads as configuration rather than as a request. Effort differentiation is trustworthy there.

Practical consequence: treat a per-role `effort` as **advisory on the native arm and binding on the CLI
arms**. A project that genuinely needs a cheaper-thinking stage should reach for a cheaper *model* (or
run that stage through `fab dispatch`), not a lower effort on a natively-dispatched stage. The clean
fix — a first-class per-sub-agent `effort` parameter on the Agent tool — is a harness ask outside fab's
control (§ Foreground limitation's scope note).

### User-directed overrides

When the user directs a provider/model for specific stages ("run review on codex"), the dispatch site
adds the override flags to its **existing single** `fab resolve-agent <stage> --alias` call
(§ Resolution step 2a). Nothing else about the seam changes — one resolve call per stage, the same two
seams, the same branch on `dispatch=` presence, the same compliance-visibility obligation. There is
**no new dispatch machinery and no persistent state**: an override is per-invocation, so "use codex for
the next N stages" means the same flags on those N resolve calls. The load-bearing caveat: **an
invocation-time override binds the native Agent-tool arm only.** `fab dispatch start` takes no override
flags — it re-resolves the stage from config itself (`agent.Resolve`) — so an overridden profile never
reaches either `fab dispatch` mode, and a `dispatch=` line that appears *only* because of a
`--provider` swap is **not actionable**. The two remedies are **not interchangeable**. Dispatching the
stage natively with the overridden model/effort is executable only for a **within-claude**
`--model`/`--effort` override: the native adapter's model seam is the Agent tool's `model` param, a hard
**Claude-alias enum** (`opus`/`sonnet`/`haiku`/`fable` — § Harness-adapter boundary), so a non-Claude
model has no native seam to ride. For a **cross-provider `--provider` override** the **config override**
(`agent.workers`, or `agent.profiles.<role>.provider`) that `dispatch start`'s own re-resolution will
see is therefore the **sole executable path** — the invocation flag can only report the mismatch. Sites
still re-read the resolved `dispatch=` after an override rather than assuming the stage's unoverridden
adapter — the branch rule is unchanged — but they read it to *notice* the mismatch, not to act on it.
See [`harness-adapters.md`](harness-adapters.md) § Relationship to `stage-models.md`.

The **`review` stage resolves once** (on its own `review` role) and applies the resolved
`{provider, model, effort}` to its **single** review sub-agent — exactly like every other stage
(260704-pag2 collapsed review to one sub-agent; there is no longer a second reviewer or a merge to
resolve, so review carries no special resolution rule).

### Caller-invariant resolution — every dispatched seam resolves a role

**A stage (or dispatched step) resolves the same role regardless of which caller drives it** —
`/fab-continue`, `/fab-ff`, `/fab-fff`, or `/fab-proceed`. Two seams that formerly dispatched at the
inherited session model now resolve a role like every other:

- **`/fab-continue`'s ship and review-pr rows.** These delegate to `/git-pr` and `/git-pr-review`, and
  resolve `fab resolve-agent ship --alias` / `fab resolve-agent review-pr --alias` before
  dispatching that sub-agent — surfacing `model=/effort=` and applying the two seams — **mirroring
  `/fab-fff` Steps 4–5 exactly**. This closes the caller asymmetry where `/fab-fff` resolved a role for
  ship/review-pr but plain `/fab-continue` did not. `/git-pr` and `/git-pr-review` still self-manage their own
  `fab status` transitions — only the model/effort seam is added.
- **`/fab-proceed`'s prefix steps.** The prefix-step dispatches were previously exempt ("no
  `fab resolve-agent` — they dispatch at the inherited model"). They now resolve a **role by name**
  (the resolver accepts a role name positionally, the same path `fab agent <role>` uses — no Go change):
  `/fab-switch` and `/git-branch` resolve `fab resolve-agent fast --alias`; the `_intake` create-intake
  dispatch resolves `fab resolve-agent default --alias`. (Intake itself remains advisory-only on the
  foreground `/fab-new` path, which no resolution can govern.) Both surface `model=/effort=` and dispatch
  through the two seams (empty ⇒ omit). This is why `fast` is multi-referent — it governs the ship stage
  *and* these prefix-step dispatches.

`_cli-fab.md` documents the `fab resolve-agent` command signature (Constitution constraint: CLI changes
MUST update `_cli-fab.md`). `architecture.md` documents the `agent:` + `providers:` config blocks
alongside the existing `stage_hooks` example.

### Harness-adapter boundary (the only Claude-Code-specific layer)

Per-stage selection is **provider-neutral by construction**, not Claude-locked:

- *Portable layers (no provider knowledge):* the `agent:` + `providers:` config schema, and the
  entire `fab resolve-agent` resolution path (stage→role→`{provider, model, effort}`). The resolver
  does no validation and echoes strings verbatim, so a project can switch agents by pointing a depth
  knob at another provider and giving it per-role fills in that provider's model IDs and effort
  vocabulary (`gpt-5 / reasoning_effort:high`, `gemini-* / <its-knob>`) and nothing in fab rejects it.
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
  instruction in the subagent prompt** (the Agent tool exposes no effort parameter) — with the
  reliability caveat in § Effort asymmetry. The skill wiring
  names both explicitly as the Claude-Code adapter, not as universal truth. This coupling is **not
  introduced by this feature** — fab's entire existing subagent-dispatch design (`_preamble.md` §
  Subagent Dispatch) is already Claude-Code-shaped. Per-stage selection is exactly as portable as fab's
  existing dispatch: no more, no less. *(The operator launcher path is the deliberate exception — it
  resolves the **operator**-role profile WITHOUT `--alias`, because `spawn.WithProfile` composes a
  `claude` CLI invocation, which accepts full IDs. `WithProfile` is grammar-forgiving: it
  **substitutes** the resolved values into a `{model}`/`{effort}` **template** `session_command` —
  including the built-in claude default, which is templated, and a codex command — all-or-nothing (any
  placeholder disables the append entirely); an empty value drops the placeholder's token and a
  preceding `-`-flag — and,
  for a command carrying **no placeholder** (a plain-form config carried forward from before the
  templated default), it instead **appends** `--model <full-id> --effort <level>` at the END (last-wins;
  each flag omitted independently when its value is empty, per the `empty ⇒ omit` convention). Placing
  the default's placeholders last makes substitution byte-identical to the former append — so a
  non-Claude worker CLI is configurable without the launcher emitting Claude-only flags; 260702-6tmi,
  templated default 260703-gvxd.)*
- *Cross-harness stage dispatch (the `dispatch=` adapter):* the resolved `dispatch.mode` + capability
  ladder is the seam for handing one stage to a native Agent-tool, pane, or headless CLI adapter.
  Pane/headless resolution emits `dispatch=<command>` with `{model}`/`{effort}` substituted via
  `internal/spawn`; native resolution omits it. This adapter is
  the **inverse aliasing rule** from the Agent-tool `model` param: the `dispatch=` command **ALWAYS
  embeds the FULL model ID, never an alias**, because an external CLI's `--model` flag takes a full ID
  — CLI dispatch never aliases. So under `--alias` the `model=` line is aliased (Agent-tool half) while
  the `dispatch=` line carries the full ID (CLI half). The field is **independent of** a provider's
  `session_command` (which opens whole sessions); each ladder rung requires its own capability and
  command fields never substitute for one another. *`fab resolve-agent` emits the line; the
  dispatch that RUNS it (`fab dispatch`) and the skill dispatch-seam wiring that consumes it both
  shipped.* **The
  native Agent-tool adapter described in this section is one of *three* dispatch adapters catalogued
  in [`harness-adapters.md`](harness-adapters.md)** — the two `fab dispatch` modes are the others:
  **headless CLI** (3c) and **interactive pane** (`fab dispatch start --pane`, 260805-zxe0, a worker in a
  tmux pane the user can watch and steer — split into the dispatching agent's own window, or a new window
  when there is no pane to split). That spec fixes the cross-adapter dispatch protocol
  (dispatch-prompt obligations, the five-state machine plus each adapter's reachable subset,
  hooks-enhance-never-own) all three share; the skill
  dispatch-seam wiring against it lives in `_preamble.md` § CLI-Adapter Dispatch + § Dispatch-Prompt
  Obligations (3d).
  **Mode resolution is shared**: `resolve-agent` and `fab dispatch start` consume the same pure
  selector. Explicit flags precede automatic `dispatch.mode` descent; pane composes the resolved
  provider's `session_command`, native uses the Agent-tool capability, and headless composes
  `dispatch_command`. A missing capability skips its automatic rung, never substitutes a field, and
  never ascends. Start performs a real pane probe and re-descends on `tmux unreachable`; landing on
  native yields re-resolution guidance before state writes.
- *Claude-flavored data (overridable):* the `claude` provider's shipped fills use Claude model
  IDs/effort — the fills that apply while both depth knobs sit on their `claude` default; `codex` and
  `gemini` ship their own model IDs (§ Three built-in providers). Every provider's fills are
  fab-kit's defaults for that provider, fully replaceable via the depth knobs plus
  `providers.<name>.profiles`.
- *v1 scope is architecture-neutral + documented — NOT shipped/tested against a non-Claude harness.* No
  provider-detection, no non-Claude integration test. The acceptance proof is
  "a non-Claude project can point a knob elsewhere and nothing in fab rejects it",
  not "we ran it on a non-Claude harness." Shipped+tested multi-provider support is explicitly out of
  scope.

---

## apply↔review coupling: why apply is `doing`, not cheaper

The apply stage produces the diff the review stage critiques, so the two are **economically coupled**:
if `apply` runs on a cheaper model than `review`, a sharper reviewer bounces the cheaper executor's
work more often, driving **more rework cycles** (capped at 3 per `code-review.md`). Three expensive
review rounds can cost more than running `apply` on the capable model once. "Cheaper apply = cheaper
pipeline" is therefore *not* strictly true.

This is why `apply` stays in `doing` on a top-end author rather than dropping to a cheap model: apply
has the highest output volume (which argues for the cheaper model), but the coupling argues louder — a
strong author minimizes the rework cycles a sharp reviewer would otherwise trigger.

`doing` and `review` are **separate roles that currently resolve to the same profile** (see
§ Default role profiles for the shipped values). The separation is structural, not a model-diversity
claim: keeping two roles lets a project dial the critic independently of the author without touching
the taxonomy, and the author/critic separation that actually does the work is the **fresh context and
adversarial framing**, not a different model. A project that wants a genuinely different critic sets
`agent.profiles.review`; fab does not ship that as the default.

---

## Fable upgrade path

Fable has landed, and the defaults moved with it — the shipped assignment is the table in
§ Default role profiles, which is the drift-guarded mirror of `defaults.yaml` and the only place this
spec states model IDs. (Deliberate: prose that restates the IDs rots silently between bumps, which is
exactly what happened to this section before.) The durable shape is the rationale in that section:
Fable for the quicker interactive working style, Opus where the named strengths are code review,
agentic execution, and memory writing, Sonnet at the mechanical floor.

fab bumps the values in **one place** (the `providers.claude.profiles` block of `defaults.yaml`) each
release, and every non-overriding project upgrades for free. The per-role fill table is fab's curated
judgment per release, not a fixed effort-per-role-rank rule — a new top model does not mechanically
promote every role, and two roles resolving to the same profile in a given release is a legitimate
outcome, not a bug. A project that pins a role opts **out** of fab's upgrade curve for that role
(correct behavior — naming it here).

### Upgrade note — the hydrate split (no migration)

Before 260719-g55d, `hydrate` mapped to `doing`, so a project carrying a `doing` override governed its
hydrate stage through that override. After the split, `hydrate` is its own role: a project with a
`doing` override but **no** `hydrate` override resolves the `hydrate` kit default
(§ Default role profiles) for the hydrate stage, not its `doing` value. **No config key changed
meaning or went inert** — the `doing` override still governs apply and review-pr exactly as before, and
`hydrate` is a newly-recognized key that was simply ignored before. Because nothing was restructured,
that shipped as an **upgrade note, not a migration**: a project that wants hydrate to keep tracking its
old `doing` value adds an `agent.profiles.hydrate:` override with that value.

---

## Foreground limitation (advisory only)

A sub-agent's model is set at dispatch time by the orchestrator. Per-stage model selection is honored
on dispatched sub-agent runs.

**Post-intake stages no longer have a foreground path (260613-fgxx).** The post-intake dual execution
mode was collapsed: apply/review/hydrate always dispatch a sub-agent, and plain `/fab-continue` is a
one-stage sequencer that resolves `fab resolve-agent <stage>` and dispatches the stage's block just
like an orchestrator. So `fab resolve-agent` applies uniformly across those stages regardless of
caller — this closes **Gap 1a** of the per-stage-model finding (foreground stages can't resolve a
role). Intake is pre-boundary: it runs in the main session and resolves no role.

The residual advisory-only case is narrow: a stage skill genuinely run with **no dispatch at all**.
There fab cannot switch the session model mid-run, so the configured profile is **advisory only** — the
skill MAY note "this stage is configured for X; you're on Y" but MUST NOT attempt to switch models.

> **Scope note**: this section reconciles the foreground limitation with the single post-intake
> execution mode (260613-fgxx, Change A). The **effort half** of per-stage selection — injected into the
> subagent prompt as an explicit instruction (since the Claude Code Agent tool has no effort parameter)
> — and the **compliance-visibility** behavior (surfacing the resolved `model=/effort=` at each
> dispatch site) are written in by 260613-m3d4 (Change C); see § Skill wiring above, and § Effort
> asymmetry for how far the prompt seam actually carries. The **lone
> residual** is a first-class per-sub-agent `effort` parameter on the Agent tool (the
> per-stage-model finding's Gap 2 clean fix) — a harness ask outside fab's control, deliberately not built here.

---

## Drift guard

The two tables above (§ Default role profiles and § The fixed stage → role mapping) are verified
mirrors of `src/go/fab/internal/agent/defaults.yaml` (§ Default role profiles, via the knobs and
`providers.claude.profiles` it composes) and the `stageRoles` map in
`src/go/fab/internal/agent/agent.go` (§ The fixed stage → role mapping). The code side is canonical. A
test in that package (`TestDocTablesMatchAgentMaps`) parses both tables from this doc and fails if
either disagrees with the code — same pattern as `TestDocTablesMatchScoringMaps` for
`docs/specs/change-types.md`. `TestMirrorDocsMatchDefaultProfiles` covers the second shape this doc
carries: the inline-YAML `providers.<name>.profiles` samples in § Three built-in providers — every
fill line is checked against the provider whose block it sits in, so all three built-ins' samples are
guarded, not just claude's.

The embedded data file has its own guard: a YAML typo is no longer a compile error, so
`TestDefaultsFileIsWellFormed` (and its siblings in `defaults_test.go`) parse `defaults.yaml`
independently and assert both knobs are set, every role's claude fill is present and populated, all
three built-ins carry per-role fills whose `default` entry names a model (with none using the
deprecated flat spelling, and gemini's carrying no effort), and the package's tables and exported
command values are wired to the file's keys.

---

## Out of scope (deferred)

- **User (`~/.fab-kit`) config layer** — was dropped here; subsequently shipped as the system rung
  (`260708-lpb5`) of the current four-layer cascade (environment > project > system
  `~/.fab-kit/config.yaml` > defaults). `agent` and
  `providers` are both scope `both`, so the depth knobs, a per-role provider fill, and a role override
  are all settable once per machine.
- **Non-claude default fills** — *no longer deferred*: shipped by `260806-ywkx`. See § Three built-in
  providers and § Refreshing the non-claude fills.
- **Role-granular reviewer keys** — obsolete: review is now a single sub-agent (260704-pag2), so there are no per-role reviewer/merge profiles to key on; the stage/role is the unit.
- **Per-invocation `--model-<stage>` flags** on the orchestrators — still deferred as an
  *orchestrator* surface. The equivalent capability exists one level down, on the resolution
  surface itself: `fab resolve-agent <stage> [--provider] [--model] [--effort]` (`260805-j3cm`), which
  every dispatch site already calls exactly once per stage — so a per-stage override needs no new
  orchestrator flag surface and no new dispatch machinery.
- **Cost/latency telemetry** per role — out of scope; this is selection only.
- **Shipped/tested multi-provider support** — out of scope; v1 proves architecture-neutrality only.
