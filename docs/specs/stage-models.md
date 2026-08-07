# Per-Stage Model Selection via Named Tiers

> **Status:** Design intent (pre-implementation, now implemented in 260613-l3ja). This spec
> captures the design for letting a project run different pipeline stages on different model
> tiers. The canonical data is `src/go/fab/internal/agent/defaults.yaml` (the embedded default tier
> profiles and provider grammars) plus the `stageTiers` map in
> `src/go/fab/internal/agent/agent.go`; the two tables in this doc are verified mirrors of them
> (drift-guarded — see § Drift guard).

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

Six **role tiers** form the vocabulary — concrete referents ("the operator", "the reviewer"), not
cognitive modes. A tier is **stage-named only where it maps 1:1 to a single referent** (`review`,
`hydrate`); `default`, `doing`, and `fast` keep role names because each is **multi-referent** (`fast`
governs the ship stage *and* the `/fab-proceed` prefix-step dispatches — see § Skill wiring):

| Tier | Role |
|------|------|
| `default` | Spawned worker sessions (`fab batch`), `fab agent` with no tier, intake (advisory only — foreground), and the `/fab-proceed` create-intake dispatch. Also the **per-field fallback for every other tier**. *(`fab agent --provider <name>` deliberately resolves NO tier — it is a sibling addressing mode that bypasses this taxonomy entirely; see § Providers.)* |
| `operator` | The operator coordinator session (`fab operator`). |
| `doing` | **Execution that must not err** — apply writes the diff; review-pr fixes already-articulated feedback. |
| `review` | **The critic** — review reads a diff and discovers what's wrong. Its own tier (not folded into `doing`) so the critic's model and effort can be dialed independently of the author's; the separation that matters is the fresh context and adversarial framing, not a different model family. |
| `hydrate` | **Memory writing** — hydrate merges the change into `docs/memory/` as current truth. Its own tier so it can run on a different model/effort than apply. |
| `fast` | **Speed on near-mechanical work** — ship's commit/push/PR mechanics plus a faithful PR-description summary, and the `/fab-proceed` prefix steps (`/fab-switch`, `/git-branch`). Multi-referent, so it keeps its role name. |

There is no `thinking` tier: with `review` its own tier, `thinking`'s only remaining stage would be
intake, which never dispatches (it is pre-boundary, foreground). Intake rides `default`, honestly — it
runs wherever the interactive session runs.

### Default tier profiles

fab-kit ships a default `{provider, model, effort}` per tier. These profiles live in
**`src/go/fab/internal/agent/defaults.yaml`** — a data file **embedded into the binary** (`go:embed`,
never read from the kit cache at runtime) and shaped as a config-file fragment, so it is versioned
with the kit and is the single file to edit when a new model ships. Provider is written explicitly on
every line (documented style — per-line readability; inheritance is the safety net, not the style).

| Tier | Provider | Model | Effort |
|------|----------|-------|--------|
| `default` | `claude` | `claude-fable-5` | `high` |
| `operator` | `claude` | `claude-sonnet-5` | `medium` |
| `doing` | `claude` | `claude-opus-5` | `high` |
| `review` | `claude` | `claude-opus-5` | `high` |
| `hydrate` | `claude` | `claude-opus-5` | `high` |
| `fast` | `claude` | `claude-sonnet-5` | `medium` |

This is the verified mirror of the `agent.tiers` block of
`src/go/fab/internal/agent/defaults.yaml` (parsed into the `defaultTiers` map at package
initialization). A drift-guard test fails if the two disagree (see § Drift guard).

**Why these defaults.** `doing` runs Opus at `high` — Opus is the strongest coding/agentic model, and
`high` is the recommended effort sweet spot (`xhigh` buys marginal gains at disproportionate
latency/cost); a strong author minimizes rework cycles per the apply↔review coupling (see §
apply↔review coupling). `review` runs Opus/`high` — code review is a named Opus strength, and the critic gets the same
top-tier model as the author so it can actually catch what the author missed (author/critic separation
is enforced by the *fresh context and adversarial framing*, not by a weaker model). `hydrate` runs
Opus/`high` — knowledge work and memory writing are named Opus strengths, and `high` is the
recommended default for intelligence-sensitive-but-not-hardest work. `default` runs
Fable/`high` — interactive sessions want the quicker working style (Anthropic guidance: `high` is the
sweet spot, and Fable at lower efforts still exceeds prior models' `xhigh`). `operator` runs
Sonnet/`medium` (highest-volume coordinator, pattern-matching work, escalation discipline makes the
cheaper model safe). `fast` sits at the mechanical floor on Sonnet/`medium` — effort at `medium` (not
`low`) buys margin for faithful PR-description comprehension. Cost-conscious projects opt any tier down
themselves (see § Config schema).

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
| `hydrate` | `hydrate` |
| `ship` | `fast` |

This is the verified mirror of the `stageTiers` map in `src/go/fab/internal/agent/agent.go`
(drift-guarded). The mapping is exhaustive — every one of the six pipeline stages belongs to exactly
one tier. `intake → default` is **advisory only**: intake runs foreground in the user's own session,
which fab cannot re-model (see § Foreground limitation). Where a stage and a tier share a name
(`review`, `hydrate`), the stage maps to that same-named tier — see § Resolution's fixed-point rule.
(`ship` is a stage but not a tier — it maps to the multi-referent `fast` tier.)

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
  `fab agent`). This is the relocated `agent.spawn_command`. It is reachable **two ways**: through a
  tier (the tier names a provider, and its `{model, effort}` are substituted) or **directly**, via
  `fab agent --provider <name> [--model <id>] [--effort <level>]`, which bypasses tier resolution
  entirely — a provider-addressed spawn for the "give me a codex session right here" case, where no
  tier need name the provider first. The direct form is a **lookup**, not a new validation surface: an
  unknown name errors listing the available providers, while resolved command strings still pass
  through verbatim. See `_cli-fab.md` § fab agent.
- **`dispatch_command`** — runs ONE headless **stage task** via `fab dispatch`. **ABSENT
  `dispatch_command` = native Agent-tool dispatch** (the default, unless the `dispatch.watchable`
  opt-in applies — § Watchable pane dispatch below). There is **NO fallback** between the
  two fields — absence of `dispatch_command` never means "use `session_command`" for a *headless*
  dispatch.

The two fields are deliberately **not merged** into one `command`: session and dispatch are different
invocations of the same binary (claude interactive `-n` vs headless `-p`; codex TUI vs `codex exec`),
and no single template expresses both.

A provider MAY also carry two optional **default-fill** fields:

- **`model`** / **`effort`** — this provider's default values for the `{model}`/`{effort}`
  placeholders. They sit third in the fill precedence (below), so they answer "what model does *this*
  provider run when nothing more specific says". Scope `both`, so a machine-wide fill is settable once
  in `~/.fab-kit/config.yaml`.

### Three built-in providers, grammar only

fab-kit ships **three built-in providers** — `claude` (the default), `codex`, and `gemini` — in the
`providers:` block of `internal/agent`'s embedded `defaults.yaml`. They carry **grammar only**: the
command templates ship inside the binary, and **no built-in carries a `model`/`effort` fill**. The
split is deliberate — invocation *grammar* changes at binary-release cadence and is safe to ship;
non-claude *model IDs* rot in weeks and must never be baked into a release.

Consequences:

- **Naming `codex`/`gemini` resolves with zero `providers:` config** — a tier override, or
  `fab agent --provider codex` / `fab resolve-agent <stage> --provider codex`, works on a fresh
  project. A `providers:` block is for *overriding* a grammar or *supplying fill*, not for registering
  these providers.
- **A built-in provider is inert until named.** Adding the rows changed no default behavior, which is
  why presence=intent (the rule that keeps behavior-changing config commented) does not force the
  grammar out of Go.
- **codex/gemini DO carry a `dispatch_command`; claude does not.** So naming one in a tier flips that
  tier's stages from native Agent-tool dispatch to CLI dispatch — exactly what selecting a non-claude
  provider means.
- The reference/scaffold render the codex/gemini blocks **commented**, like every other non-overridden
  default (a commented block registers no project override).

> **Explicit reversal (`260805-j3cm` over `260731-ho9y`).** ho9y shipped codex/gemini as
> *uncomment-to-opt-in template text* and recorded "no new built-in providers are added in Go". j3cm
> reverses that, narrowly: **grammar strings only**. ho9y's reasoning (presence implies intent, so
> anything whose presence changes behavior ships commented) still holds and is *preserved* — a built-in
> provider is inert until named, so the rows change no default behavior. What ho9y additionally assumed
> — that a built-in row would need fill values — is exactly what the per-provider fill fields remove.

**Provider names are opaque — fab NEVER infers a provider from a model string** (`claude-*` → claude
would need a provider registry, which the no-validation/provider-neutrality contract refuses).

### Fill precedence

```
invocation flag  >  explicit tier field  >  named provider's default fill  >  empty
```

`empty` keeps its existing meaning: an empty `model=` line is the "inherit the session model" signal,
and on a command the placeholder's token (plus a preceding `-`-flag) is dropped by
`spawn.WithProfile`, so the CLI's own default applies.

**Cross-provider fill** closes the former footgun ("override a tier's `model` cross-provider ⇒
override its `provider` too", which fab documented rather than fixed because there was no correct
value to fill with). When a tier **explicitly sets `provider:`** to a provider **NAME** differing from
the built-in tier profile's, its unset `model`/`effort` fill from **that provider's** default fill and
then empty — **never** from the built-in/`default`-tier values, which belong to the other provider
name. A tier that sets no `provider:` inherits exactly as before, and the all-claude default world is
byte-unchanged (every built-in tier pins an explicit model). The residual guidance shrinks to: *pin
`model:` on the tier to be explicit.* The same rule governs a `--provider` swap on
`fab resolve-agent` (§ Resolution).

The cutoff is **by provider name, not by vendor.** The rule is a string comparison of two provider
names (`configured.Provider != builtin.Provider`), and fab knows nothing about which vendor a name
fronts — provider names are opaque, user-chosen strings and fab never infers a provider from a model
string (§ No validation). So a *same-vendor* rename is still a cutoff: a second entry
`providers.claude-alt` naming the claude grammar under a different name, set as a tier's `provider:`,
loses that tier's `model`/`effort` inheritance exactly as `codex` would. This is deliberate — a
name-keyed rule needs no vendor table to maintain, and the two escapes are the ones already
documented: pin `model:` on the tier, or give the named provider a `model`/`effort` fill.

**Scope of ownership — a documented limitation (not a supported behavior).** The cutoff decides which
values are foreign by their *owner* — the provider in effect at the config layer that supplied each
value. Those layers are `built-in tier ← project default tier ← requested tier override`, and they are
the **only** ones ownership can see: `internal/config.LoadPath` resolves the
`project > system (~/.fab-kit/config.yaml) > built-in` cascade by **deep-merging the two files per key
BEFORE** `internal/agent` resolves anything, so resolution receives one merged tree and per-SCOPE
ownership is not tracked. Consequence: when the system scope and the project scope both contribute to
the **same tier** and name **different** providers, the merged tier reads as one layer and its
`model`/`effort` are attributed to the merged layer's `provider:` — the cutoff does **not** fire across
that scope boundary. Concretely, a system-scope `agent.tiers.doing: {provider: codex, model:
gpt-5.3-codex, effort: high}` under a project-scope `agent.tiers.doing: {provider: gemini}` resolves
`model=gpt-5.3-codex` with `provider=gemini` — a codex model ID handed to the gemini CLI. This is
**pinned by test, not endorsed** (`TestResolveCrossScopeCascadeLimitation` in `cmd/fab`, the layer that
can compose both scopes; the same limitation is stated in `internal/agent`'s `ResolveTier` doc comment).
The workaround is to pin `model:`/`effort:` in the **same scope** as the `provider:` switch.
Cascade-aware ownership — folding the per-scope layers inside `ResolveTier` instead of consuming a
pre-merged tree — is **deferred to a follow-up change**.

One asymmetry follows from claude's fill living on the **built-in tiers** rather than on the provider:
a swap **to** `claude` has no `providers.claude.model` rung to refill from, so `--provider claude`
from a non-claude tier resolves an **empty** `model=` (the inherit-the-session-model signal) unless an
explicit `--model` accompanies it or a `providers.claude.model` fill is configured. (`--provider claude`
against a tier that already resolves to claude is not a swap at all, so nothing refills.)

## Config schema — `providers:` + `agent.tiers` (the override surfaces)

Both are optional maps in `fab/project/config.yaml`. The Go `Config` struct widens freely — yaml
unmarshalling ignores unknown keys, so existing configs are unaffected (the same property that made
`stage_hooks` free to add).

```yaml
providers:
  claude:
    session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}'
    # dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'   # uncomment to flip claude's stages from native Agent-tool dispatch to headless CLI
  # codex and gemini are BUILT-IN providers (grammar only) — these blocks merely
  # restate the built-in default, so they ship commented like every other default.
  # Uncomment only to OVERRIDE a grammar; set model/effort to supply the fill fab
  # deliberately ships none of.
  # codex:
  #   session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
  #   dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'
  #   model: gpt-5.3-codex                    # example fill — fab ships no codex model ID
  #   effort: high
  # gemini:
  #   session_command: 'gemini -m {model}'
  #   dispatch_command: 'gemini -m {model}'   # no {effort} flag; no -p (fab dispatch pipes the prompt to stdin)
  #   model: gemini-2.5-pro                   # example fill — fab ships no gemini model ID

agent:
  # The stage→tier mapping is OWNED BY FAB-KIT and is NOT overridable — shown
  # here only as reference so you know which stages each tier governs:
  #   default:  intake (advisory), fab batch, fab agent, /fab-proceed create-intake  (+ per-field fallback)
  #   operator: fab operator (coordinator session)
  #   doing:    apply, review-pr                          (execution that must not err)
  #   review:   review                                    (the critic)
  #   hydrate:  hydrate                                   (memory writing)
  #   fast:     ship, /fab-proceed prefix steps           (near-mechanical work)
  #
  # You override only WHAT EACH TIER MEANS (provider + model + effort). Omit any
  # tier to use fab-kit's built-in default. fab-kit defaults today are:
  #   default:  { provider: claude, model: claude-fable-5,  effort: high }
  #   operator: { provider: claude, model: claude-sonnet-5, effort: medium }
  #   doing:    { provider: claude, model: claude-opus-5,   effort: high }
  #   review:   { provider: claude, model: claude-opus-5,   effort: high }
  #   hydrate:  { provider: claude, model: claude-opus-5,   effort: high }
  #   fast:     { provider: claude, model: claude-sonnet-5, effort: medium }
  tiers:
    doing: { provider: claude, model: claude-sonnet-5, effort: medium }   # example: run doing cheaper
```

- Keys under `tiers:` are the six role-tier names: `default`, `operator`, `doing`, `review`, `hydrate`, `fast`.
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
fab resolve-agent <stage|tier> [--alias] [--provider <name>] [--model <id>] [--effort <level>]
```

(Named `resolve-agent`, not `resolve-model`, because it resolves the provider, the model, and the
effort the agent dispatch needs.)

1. Take a **stage** name (`intake`/`apply`/`review`/`hydrate`/`ship`/`review-pr`) or a **role-tier**
   name (`default`/`operator`/`doing`/`review`/`hydrate`/`fast`) — the positional argument accepts
   either. A stage maps through the fixed stage→tier mapping; a tier resolves directly (the path
   `fab agent` and the operator launcher use). **Tier names are checked first.** The two name sets
   overlap only at **fixed points**: a name shared by a stage and a tier (`review`, `hydrate`) is one
   where the stage maps to that same-named tier (`stageTiers[name] == name`), so the tier-first check
   resolves such a name to the same profile either interpretation would — the order is immaterial for
   results. (A drift-guard test in `internal/agent` asserts every collision is a fixed point.) `ship`
   is a stage but NOT a tier — it maps to the `fast` tier — so `resolve-agent ship` resolves the stage
   while `resolve-agent fast` resolves the tier, both to the same profile.
2. Resolve the tier → `{provider, model, effort}`: the project's `agent.tiers.<tier>` override
   **per-field merged** over the project's `default` tier, over fab-kit's built-in. Any field wins in
   that order — **except across a provider switch**: a tier that explicitly sets `provider:` to a
   provider NAME differing from the built-in tier's fills its unset `model`/`effort` from that
   provider's default fill, then empty (§ Fill precedence → Cross-provider fill — name-based, so a
   same-vendor rename cuts inheritance too).
2a. **Apply invocation-time overrides** (`--provider`/`--model`/`--effort`) — the top rung of the fill
   precedence. `--provider` swaps the provider and **re-derives `dispatch=` from the NAMED provider's
   `dispatch_command`** — so the emitted `dispatch=` presence can differ from the stage's unoverridden
   one, but that is a **query result, not an adapter move**: `fab dispatch` re-resolves from config and
   accepts no overrides, so only a config/tier override actually relocates a stage between the native and
   CLI adapters (§ Skill wiring → User-directed overrides). A swap does not retain the tier's
   `model`/`effort` (an unoverridden field refills from
   the new provider's fill, then empty — the same name-based cross-provider rule, including the
   swap-back-to-claude case that lands on an empty `model=`). `--model`/`--effort` are valid
   **without** `--provider` (a within-tier override) — the documented asymmetry with `fab agent`, where
   they stay a usage error without `--provider`, because `resolve-agent` is a pure query whose whole
   output is a profile while `fab agent` is a launcher with two mutually exclusive addressing modes.
   All three key on whether the flag was *supplied* (cobra's `Flag.Changed`), not on value emptiness.
   A supplied-but-unresolvable `--provider` is a **lookup** failure: non-zero exit naming the
   resolvable set (built-in table ∪ the project's `providers:` keys, sorted), mirroring `fab agent`'s
   error — naming resolvable *names* is not validating a command's *content*.
3. **Emit verbatim — NO validation** (see § No validation). fab does not check the provider, model, or
   effort against any provider's accepted set; it echoes the resolved strings as-is.
4. Output: a `model=<id>` line always, then optional `effort=<level>`, `provider=<name>`, and
   `dispatch=<command>` lines. The `effort=`/`provider=` lines are **omitted** when empty. An empty
   model emits an empty `model=` line (the "inherit" signal). The `dispatch=` line is emitted when the
   resolved tier's provider carries a `dispatch_command` (mirroring the effort-omit rule) — or, when
   `dispatch.watchable: true` and the orchestrator sits inside tmux, for a `session_command`-only
   provider (the **watchable pane opt-in**, § Watchable pane dispatch below); its **absence signals
   native Agent-tool dispatch**, and a **headless** dispatch has **NO fallback to a session command**.
   The `dispatch=` command's `{model}`/`{effort}` placeholders are substituted via
   `internal/spawn`'s template resolution (reused, not reimplemented), using the tier's own resolved
   model/effort — and the `{model}` is **always the full model ID**, even under `--alias` (see
   § Harness-adapter boundary).
6. **Byte-stable** for the same config (like other `fab resolve` queries). Non-zero exit only on a
   real error: an unreadable/malformed config, or an unknown stage name. A stage that resolves to a
   default is success, not an error.

### Watchable pane dispatch — `dispatch.watchable`

`dispatch.watchable` is a bool config field (**default `false`**, **scope `both`** — settable once
machine-wide in `~/.fab-kit/config.yaml`) that adds a **second trigger** for the `dispatch=` line: when
it is `true` **AND** `$TMUX` is set **AND** the resolved provider carries a `session_command` but **no**
`dispatch_command`, the line is emitted carrying the profile-substituted **`session_command`**.

- **Tmux presence decides pane vs native.** With `$TMUX` unset the line is omitted and the stage stays
  on **native Agent-tool dispatch** — never headless CLI. Headless remains gated on a real
  `dispatch_command`, so the no-cross-fallback rule is intact for the headless adapter.
- **A provider `dispatch_command` wins.** Watchable only ADDS eligibility for providers that have none;
  emission for a `dispatch_command`-carrying provider is unchanged.
- **Why this exists.** Pane mode composes `session_command`, not `dispatch_command` — so pane
  *eligibility* was gated on a field pane mode never uses. Before the opt-in, the only way to get a
  watchable claude worker was to uncomment claude's `dispatch_command`, which also flipped every
  out-of-tmux dispatch to **headless CLI** — a footgun disguised as a default.
- **No skill-wiring change.** The dispatch seam branches on the line's *presence* and never executes its
  value; `fab dispatch start` re-resolves internally, and inside tmux its auto ladder selects pane mode
  and composes the same `session_command`. A `session_command`-only provider dispatches fine under pane
  mode (shipped 260805-zxe0 / l9ng behavior).
- **`--alias` is unaffected**: the `dispatch=` line always embeds the full model ID.
- **Known edge (documented, not solved)**: if tmux dies between the resolve and `fab dispatch start`,
  start's auto ladder soft-falls-back to headless and then errors on the missing `dispatch_command`.
  Rare, and self-explaining at the CLI.

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

| Model family | Accepts effort? | Valid values |
|--------------|-----------------|--------------|
| Opus / Fable | Yes | `low`, `medium`, `high`, `xhigh`, `max` |
| Sonnet | Yes | `low`, `medium`, `high`, `max` (no `xhigh` — Opus-family only) |
| Haiku | **No** | — (effort param returns HTTP 400) |

This table explains *why fab-kit's shipped defaults are what they are* — but it is documentation of
fab's default choices, not a rule the resolver enforces on user overrides.

### Haiku excluded from the defaults (not forbidden)

Haiku is **absent from the default tiers**, for two reasons: it has no effort parameter (passing
effort 400s), and the one stage that might want a fast/cheap model (the `ship` stage, governed by the
`fast` tier) needs faithful PR-description comprehension that Haiku does unreliably — so the `fast`
default is Sonnet/`medium`. This is **exclusion from the defaults, not a prohibition**: a user MAY
still override a tier to Haiku (pass-through doesn't forbid it); fab just doesn't ship it as a default.

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

**User-directed overrides ride the same single call.** When the user directs a provider/model for
specific stages ("run review on codex"), the dispatch site adds the override flags to its **existing
single** `fab resolve-agent <stage> --alias` call (§ Resolution step 2a). Nothing else about the seam
changes — one resolve call per stage, the same two seams, the same branch on `dispatch=` presence, the
same compliance-visibility obligation. There is **no new dispatch machinery and no persistent state**:
an override is per-invocation, so "use codex for the next N stages" means the same flags on those N
resolve calls. The load-bearing caveat: **an invocation-time override binds the native Agent-tool arm
only.** `fab dispatch start` takes no override flags — it re-resolves the stage from config itself
(`agent.Resolve`) — so an overridden profile never reaches either `fab dispatch` mode, and a `dispatch=`
line that appears *only* because of a `--provider` swap is **not actionable**. The two remedies are **not
interchangeable**. Dispatching the stage natively with the overridden model/effort is executable only for
a **within-claude** `--model`/`--effort` override: the native adapter's model seam is the Agent tool's
`model` param, a hard **Claude-alias enum** (`opus`/`sonnet`/`haiku`/`fable` — § Harness-adapter
boundary), so a non-Claude model has no native seam to ride. For a **cross-provider `--provider`
override** the **config/tier override** (`agent.tiers.<tier>.provider` plus a `providers:` entry) that
`dispatch start`'s own re-resolution will see is therefore the **sole executable path** — the invocation
flag can only report the mismatch. Sites still re-read the resolved `dispatch=` after an override rather
than assuming the stage's unoverridden adapter — the branch rule is unchanged — but they read it to
*notice* the mismatch, not to act on it. See [`harness-adapters.md`](harness-adapters.md) § Relationship to `stage-models.md`.

The **`review` stage resolves once** (on its own `review` tier) and applies the resolved
`{provider, model, effort}` to its **single** review sub-agent — exactly like every other stage
(260704-pag2 collapsed review to one sub-agent; there is no longer a second reviewer or a merge to
resolve, so review carries no special resolution rule).

### Caller-invariant resolution — every dispatched seam is tiered

**A stage (or dispatched step) resolves the same tier regardless of which caller drives it** —
`/fab-continue`, `/fab-ff`, `/fab-fff`, or `/fab-proceed`. Two seams that formerly dispatched at the
inherited session model are now tiered like every other:

- **`/fab-continue`'s ship and review-pr rows.** These delegate to `/git-pr` and `/git-pr-review`, and
  now resolve `fab resolve-agent ship --alias` / `fab resolve-agent review-pr --alias` before
  dispatching that sub-agent — surfacing `model=/effort=` and applying the two seams — **mirroring
  `/fab-fff` Steps 4–5 exactly**. This closes the caller asymmetry where `/fab-fff` tiered ship/review-pr
  but plain `/fab-continue` did not. `/git-pr` and `/git-pr-review` still self-manage their own
  `fab status` transitions — only the model/effort seam is added.
- **`/fab-proceed`'s prefix steps.** The prefix-step dispatches were previously exempt ("no
  `fab resolve-agent` — they dispatch at the inherited model"). They now resolve a **tier by name**
  (the resolver accepts a tier name positionally, the same path `fab agent <tier>` uses — no Go change):
  `/fab-switch` and `/git-branch` resolve `fab resolve-agent fast --alias`; the `_intake` create-intake
  dispatch resolves `fab resolve-agent default --alias`. (Intake itself remains advisory-only on the
  foreground `/fab-new` path, which no resolution can govern.) Both surface `model=/effort=` and dispatch
  through the two seams (empty ⇒ omit). This is why `fast` is multi-referent — it governs the ship stage
  *and* these prefix-step dispatches.

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
- *Cross-harness stage dispatch (the `dispatch=` adapter):* a provider's optional `dispatch_command` is
  the seam for handing one stage to a **different CLI harness** (e.g. `codex exec …`) instead of a
  native Agent-tool sub-agent. When a resolved tier's provider carries it, `fab resolve-agent` emits a
  `dispatch=<command>` line — its `{model}`/`{effort}` substituted via `internal/spawn`. This adapter is
  the **inverse aliasing rule** from the Agent-tool `model` param: the `dispatch=` command **ALWAYS
  embeds the FULL model ID, never an alias**, because an external CLI's `--model` flag takes a full ID
  — CLI dispatch never aliases. So under `--alias` the `model=` line is aliased (Agent-tool half) while
  the `dispatch=` line carries the full ID (CLI half). The field is **independent of** a provider's
  `session_command` (which opens whole sessions) with **no cross-fallback** for a headless dispatch —
  absence of a resolved provider `dispatch_command` is the native-dispatch signal, unless the
  `dispatch.watchable` opt-in makes a `session_command`-only provider pane-eligible inside tmux
  (§ Watchable pane dispatch). *`fab resolve-agent` emits the line; the
  dispatch that RUNS it (`fab dispatch`) and the skill dispatch-seam wiring that consumes it both
  shipped.* **The
  native Agent-tool adapter described in this section is now one of *three* dispatch adapters catalogued
  in [`harness-adapters.md`](harness-adapters.md)** — the two `fab dispatch` modes are the others:
  **headless CLI** (3c) and **interactive pane** (`fab dispatch start --pane`, 260805-zxe0, a worker in a
  tmux pane the user can watch and steer — split into the dispatching agent's own window, or a new window
  when there is no pane to split). That spec fixes the cross-adapter dispatch protocol
  (dispatch-prompt obligations, the five-state machine plus each adapter's reachable subset,
  hooks-enhance-never-own) all three share; the skill
  dispatch-seam wiring against it lives in `_preamble.md` § CLI-Adapter Dispatch + § Dispatch-Prompt
  Obligations (3d).
  **Resolution is unchanged by the pane mode**: it emits no new resolver line and needs no new provider
  field — pane mode composes the resolved provider's existing `session_command` (the same field
  `fab agent` and the operator launcher compose, through the same `internal/spawn` substitution), and mode
  selection is **per-invocation** (an explicit-first ladder over `--pane`/`--headless`/`--timeout`/`--server`
  ending in auto — pane inside tmux, headless outside), never a property of a tier or provider. This is not
  a cross-fallback: the no-fallback rule governs what `resolve-agent` emits for *dispatch*, and pane mode
  reads the provider table itself rather than consuming a `dispatch=` line. Nor does the auto default
  weaken it: an auto-selected pane whose provider carries no `session_command` **soft-falls-back to
  headless** (re-composing from `dispatch_command`), so a tier resolved to a `dispatch_command`-only
  provider dispatches identically inside and outside tmux.
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

This is why `apply` stays in `doing` on a top-end author rather than dropping to a cheap model: apply
has the highest output volume (which argues for the cheaper model), but the coupling argues louder — a
strong author minimizes the rework cycles a sharp reviewer would otherwise trigger.

`doing` and `review` are **separate tiers that currently resolve to the same profile** (see
§ Default tier profiles for the shipped values). The separation is structural, not a model-diversity
claim: keeping two tiers lets a project dial the critic independently of the author without touching
the taxonomy, and the author/critic separation that actually does the work is the **fresh context and
adversarial framing**, not a different model. A project that wants a genuinely different critic
overrides `agent.tiers.review`; fab does not ship that as the default.

---

## Fable upgrade path

Fable has landed, and the defaults moved with it — the shipped assignment is the table in
§ Default tier profiles, which is the drift-guarded mirror of `defaultTiers` and the only place this
spec states model IDs. (Deliberate: prose that restates the IDs rots silently between bumps, which is
exactly what happened to this section before.) The durable shape is the rationale in that section:
Fable for the quicker interactive working style, Opus where the named strengths are code review,
agentic execution, and memory writing, Sonnet at the mechanical floor.

fab bumps the table in **one place** (the `agent.tiers` block of `defaults.yaml`) each release, and
every non-overriding project upgrades for free. The tier→profile table is fab's curated judgment per
release, not a fixed effort-per-tier-rank rule — a new top model does not mechanically promote every
tier, and two tiers resolving to the same profile in a given release is a legitimate outcome, not a
bug. A project that overrides a tier opts **out** of fab's upgrade curve for that tier (correct
behavior — naming it here).

### Upgrade note — the hydrate split (no migration)

Before this change, `hydrate` mapped to `doing`, so a project carrying an `agent.tiers.doing` override
governed its hydrate stage through that override. After the split, `hydrate` is its own tier: a project
with a `doing` override but **no** `hydrate` override now resolves the `hydrate` kit default
(§ Default tier profiles) for the hydrate stage, not its `doing` value. **No config key changes
meaning or goes inert** — `agent.tiers.doing` still governs apply and review-pr exactly as before, and
`agent.tiers.hydrate` is a newly-recognized key that was simply ignored before. Because nothing is
restructured, this ships as an **upgrade note, not a migration**: a project that wants hydrate to keep
tracking its old `doing` value adds an `agent.tiers.hydrate:` override with that value.

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
mirrors of `src/go/fab/internal/agent/defaults.yaml` (§ Default tier profiles, via the `defaultTiers`
map it is parsed into) and the `stageTiers` map in `src/go/fab/internal/agent/agent.go`
(§ The fixed stage → tier mapping). The code side is canonical. A test in that package
(`TestDocTablesMatchAgentMaps`) parses both tables from this doc and fails if either disagrees with
the code — same pattern as `TestDocTablesMatchScoringMaps` for `docs/specs/change-types.md`.

The embedded data file has its own guard: a YAML typo is no longer a compile error, so
`TestDefaultsFileIsWellFormed` (and its siblings in `defaults_test.go`) parse `defaults.yaml`
independently and assert every tier and provider is present and populated, that no built-in carries a
`model`/`effort` fill, and that the package's tables and exported command values are wired to the
file's keys.

---

## Out of scope (deferred)

- **User (`~/.fab-kit`) config layer** — was dropped here; subsequently shipped as the three-layer
  config cascade (project > system `~/.fab-kit/config.yaml` > defaults, `260708-lpb5`). `providers`
  and `agent.tiers` are both scope `both`, so a per-provider fill or a tier override is settable
  once per machine.
- **Role-granular keys** — obsolete: review is now a single sub-agent (260704-pag2), so there are no per-role reviewer/merge tiers to key on; the stage/tier is the unit.
- **Per-invocation `--model-<stage>` flags** on the orchestrators — still deferred as an
  *orchestrator* surface. The equivalent capability now exists one level down, on the resolution
  surface itself: `fab resolve-agent <stage> [--provider] [--model] [--effort]` (`260805-j3cm`), which
  every dispatch site already calls exactly once per stage — so a per-stage override needs no new
  orchestrator flag surface and no new dispatch machinery.
- **Named tier-profile sets** (`agent.profiles.*` — switching a whole tier map per run) — deferred
  until per-stage overrides prove insufficient (`260805-j3cm`).
- **Cost/latency telemetry** per tier — out of scope; this is selection only.
- **Shipped/tested multi-provider support** — out of scope; v1 proves architecture-neutrality only.
