---
type: memory
description: "The providers & role-tiers model (agent config v3) — the `providers:` table (opaque names → `session_command`/`dispatch_command` + optional `model`/`effort` fill; three grammar-only built-ins; absent `dispatch_command` = native, no cross-fallback), the six role tiers with per-field inheritance, the fill precedence + cross-provider cutoff, the fixed stage→tier map, `fab resolve-agent` with its native-arm-only overrides and the `dispatch.watchable` opt-in, `fab agent`'s modes, and the consumers."
---
# Providers & Role Tiers

**Domain**: runtime

## Overview

Agent config v3 (260702-tykw) splits **provider mechanics** (how to invoke an agent) from **role/budget policy** (which model + effort a stage runs at). Providers live in a top-level `providers:` table carrying command grammar plus an optional default fill; tiers are `{provider, model, effort}` role profiles under `agent.tiers`. This file is the model — the three built-in providers, the six role tiers and their inheritance, the fill precedence and its cross-provider cutoff, the fixed stage→tier mapping, the `fab resolve-agent`/`fab agent` surfaces, and who consumes the resolution. The **config-schema authority** is [_shared/configuration.md](/_shared/configuration.md) § `providers` and § `agent`; the **dispatch-seam wiring** is [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution; the pre-implementation design intent is `docs/specs/stage-models.md` (drift-guarded against the Go maps). This file ties them together for the runtime reader.

## Requirements

### Requirement: Providers table — session vs dispatch

`fab/project/config.yaml` SHALL support a top-level `providers:` map keyed by **opaque, user-chosen provider names**. Each provider MAY carry two command fields, which SHALL NOT be merged into one:

- **`session_command`** — opens an interactive agent **session**. Consumed by `fab operator`, `fab batch new`/`batch switch`, `fab agent`, and `fab dispatch start --pane` (which runs a *stage* in an interactive session the user can watch and steer — see [dispatch.md](/runtime/dispatch.md)).
- **`dispatch_command`** — runs ONE headless **stage task** via `fab dispatch`. **ABSENT `dispatch_command` = native Agent-tool dispatch** (unless the `dispatch.watchable` opt-in applies inside tmux — § Watchable pane dispatch) — there is **NO fallback** to `session_command` for a headless dispatch.

A provider MAY additionally carry two optional **default-fill** fields, `model` and `effort` — the values that supply the command's `{model}`/`{effort}` placeholders when nothing more specific does (see § Fill precedence below). They are per-field merged over the built-in table exactly as the command fields are, and `providers` is `scope: both`, so a fill is settable once per machine in `~/.fab-kit/config.yaml`.

**The no-cross-fallback rule holds in both directions**: headless dispatch never substitutes `session_command`, and `--pane` never substitutes `dispatch_command` — each mode errors naming the field it needs. `--pane` composing `session_command` is **not** a fallback and not a resolver change: mode selection is per-invocation, `fab resolve-agent`'s output is byte-identical either way, and pane mode reads the provider table itself exactly as `fab agent` does. No third command field exists — the interactive stage invocation *is* the provider's session invocation.

**fab-kit ships THREE built-in providers — `claude`, `codex`, and `gemini` — as GRAMMAR ONLY** (`defaultProviders` in `internal/agent`, each command string a named Go constant `DefaultSessionCommand` / `DefaultCodex*` / `DefaultGemini*` that `internal/configref` interpolates, so no literal is duplicated):

| Built-in | `session_command` | `dispatch_command` | fill |
|----------|-------------------|--------------------|------|
| `claude` | templated default | **none** → native Agent-tool dispatch | none (claude's model/effort live on the built-in *tiers*) |
| `codex` | `codex -m {model} -c model_reasoning_effort={effort}` | `codex exec -m {model} -c model_reasoning_effort={effort}` | none |
| `gemini` | `gemini -m {model}` | `gemini -m {model}` | none |

**No built-in carries a model or effort fill.** Grammar changes at binary-release cadence and is safe to ship; non-claude model IDs rot in weeks, so they belong in user config (`providers.<name>.model`, a tier field, or an invocation flag) — never in a release. A project's `providers:` block per-field-merges over the built-ins via `agent.ResolveProvider(name)`, including the fill fields.

**Naming a built-in is sufficient — no `providers:` block required.** A tier override or an invocation flag that names `codex`/`gemini` resolves with zero config. Because codex/gemini DO carry a `dispatch_command` (claude carries none), naming one in a tier flips that tier's stages from native Agent-tool dispatch to headless CLI dispatch — exactly what selecting a non-claude provider means. A built-in provider is otherwise **inert**: adding a row changes no default behavior, which is why these rows live in Go while behavior-changing config still ships commented (see § Design Decisions → "Built-in Provider Grammar in Go, Fill Values in Config").

```yaml
providers:
  codex:
    model: gpt-5.3-codex     # the default fill — fab ships no model ID for codex
    effort: high
    # commands inherit the built-in grammar; override either to change it
```

The two command fields are deliberately unmerged because session and dispatch are **different invocations of the same binary** (claude interactive `-n <name>` vs headless `-p`; codex TUI vs `codex exec`) — no single template expresses both. `{model}`/`{effort}` placeholders in either command are substituted at resolve time via `spawn.WithProfile` (reused, not reimplemented — see [configuration.md](/_shared/configuration.md) § `providers` for the template/append modes and the empty-value token-drop rule).

Two per-provider grammar specifics: **gemini carries no `{effort}` placeholder** (the gemini CLI has no reasoning-effort flag, so a resolved effort has nowhere to go), and **gemini's `dispatch_command` is the bare `gemini -m {model}` with no `-p`** — `fab dispatch` pipes the prompt to gemini's stdin, which it reads as the prompt in non-TTY mode, whereas `-p` takes prompt TEXT appended after stdin. **codex reads its prompt from stdin** via the `exec` subcommand. `fab config reference` (and the `fab config upgrade` managed fence) presents all three as **commented reference-style built-in defaults** — the same presentation every other non-overridden default uses — with only `claude.session_command` shown live as the baseline example; see [configuration.md](/_shared/configuration.md) § `providers` for the rendered presentation and its parse-side guarantees.

**Provider names are opaque — fab NEVER infers a provider from a model string**, and the cutoff below is by provider **name**, not by vendor: two entries fronting the same vendor under different names are different providers.

#### Scenario: absent `dispatch_command` selects native dispatch

- **GIVEN** a stage whose tier points at the built-in `claude` provider (no `dispatch_command`), with `dispatch.watchable` off (the default)
- **WHEN** that stage is dispatched
- **THEN** it runs as a native Agent-tool sub-agent — `fab resolve-agent` emits no `dispatch=` line, and there is no fallback to `session_command`
- **AND** with `dispatch.watchable: true` and the orchestrator inside tmux, the same tier instead emits `dispatch=` from `session_command` (§ Watchable pane dispatch)

#### Scenario: provider `dispatch_command` drives CLI dispatch

- **GIVEN** `providers.codex.dispatch_command` and a tier `{ provider: codex, ... }`
- **WHEN** a stage on that tier is dispatched
- **THEN** `fab dispatch` runs the resolved `codex exec …` command (cross-harness), profile substituted

#### Scenario: naming a built-in provider with no `providers:` block

- **GIVEN** a config with `agent.tiers.review: { provider: codex, model: gpt-5.3-codex, effort: high }` and no `providers:` block at all
- **WHEN** `fab resolve-agent review` runs
- **THEN** it emits `provider=codex` plus `dispatch=codex exec -m gpt-5.3-codex -c model_reasoning_effort=high` — resolved entirely from the Go built-in grammar
- **AND** with no `agent.tiers` override, the same command emits no `dispatch=` line (the built-in claude provider carries none) — a built-in provider is inert until named

### Requirement: Six role tiers with per-field default-tier inheritance

`agent.tiers` keys SHALL be the six **role tiers** — `default`, `operator`, `doing`, `review`, `hydrate`, `fast`. Each tier value SHALL be `{provider, model, effort}` (no command — the command lives on the provider). A tier is **stage-named only where it maps 1:1 to a single referent** (`review`, `hydrate`); `default`, `doing`, and `fast` keep role names because each is **multi-referent** — `fast` governs the ship stage AND the `/fab-proceed` prefix-step dispatches (`/fab-switch`, `/git-branch`), and `default` governs intake, `fab batch`/`fab agent`, the `/fab-proceed` create-intake dispatch, and the per-field fallback. There is no `thinking` tier: with `review` split into its own tier, `thinking`'s only remaining stage would be intake, which never dispatches.

The six tiers and their fixed referents (the profiles themselves are owned by `defaultTiers` in `internal/agent` and drift-guarded against every doc that mirrors them):

| Tier | Role |
|------|------|
| `default` | intake (advisory, foreground); `fab batch` worker sessions; `fab agent` with no tier; the `/fab-proceed` create-intake dispatch; **per-field fallback for every other tier** |
| `operator` | the operator coordinator session (`fab operator`) |
| `doing` | `apply`, `review-pr` — execution that must not err |
| `review` | `review` — the critic (its own tier so its model/effort dial independently of the author's) |
| `hydrate` | `hydrate` — memory writing (its own tier so it runs on a different model/effort than apply) |
| `fast` | `ship` — near-mechanical work — plus the `/fab-proceed` prefix steps (`/fab-switch`, `/git-branch`) |

**Current profiles**: run `fab config reference` (renders live from `defaultTiers`) or see the drift-guarded table in [stage-models.md](../../specs/stage-models.md) § Default tier profiles. Not restated here — this table owns the *roles*, which change far less often than the models.

**Per-field inheritance**: any tier field left unset (provider, model, effort) inherits from the project's `default` tier, then from fab-kit's built-in for that tier (`ResolveTier` middle-layer merge). Inheriting `{provider, model, effort}` is safe *because commands moved to `providers:`* — the dangerous cross-semantics command inheritance cannot happen. **Documented style: write `provider:` explicitly on every tier line** even though inheritance makes it optional (per-line readability; inheritance is the safety net). Model IDs are written **versioned** (e.g. `claude-sonnet-5`, `claude-opus-4-8`) — bare family IDs (`claude-sonnet`) fail both dispatch seams.

#### Scenario: an unset field inherits the default tier

- **GIVEN** a project `agent.tiers.doing: { effort: high }` (no provider/model)
- **WHEN** the doing tier is resolved
- **THEN** provider+model come from the project's `default` tier (or its built-in), effort=high wins

### Requirement: Fill precedence and the per-field cross-provider cutoff

The `{model, effort}` any invocation resolves to SHALL be determined by, in order:

**invocation flag > explicit tier field > the named provider's default fill > empty**

An empty value keeps its established meaning: `spawn.WithProfile`'s token-drop (so the CLI's own default applies) and, on the `model=` line, "inherit the session/orchestrator model". The precedence is implemented **once** in `internal/agent` — `ResolveTier` covers the config rungs, `ApplyOverrides` adds the flag rung — so `cmd/fab` reimplements no part of it.

**Cross-provider cutoff.** Field inheritance is safe *within* a provider but not *across* one. When the config explicitly names a provider **differing from the built-in tier profile's**, every resolved `model`/`effort` **owned by another provider** refills from the resolved provider's default fill, then empty — never from the built-in tier's (or another provider's) values. A value's **owner** is the provider in effect at the layer that supplied it: the layer's own `provider:` when it names one, otherwise whatever the layers below resolved. `provider:` is applied *before* `model`/`effort` within a layer, so a value written beside a switch is owned by the new provider and survives, while a `model` on the project `default` tier with no `provider:` beside it is claude-owned and does not. Both the config path and a `--provider` flag swap run the *same* rule through one `cutForeignFields` helper, differing only in how each records owners (config-layer fold vs. flag layer).

The **gate** is the *net* configured provider vs. the built-in tier's, not a per-layer fold: a chain that ends back on the built-in's provider is not a switch at all (so an explicit `provider: claude` on a claude tier cuts nothing, and the all-claude default world is byte-unchanged — every built-in tier pins an explicit model). The cutoff keys on the provider **name**, so a second claude entry under another name still loses tier model/effort inheritance.

**Scope of ownership — a documented limitation.** Ownership is computed over the **merged** config: `config.LoadPath` deep-merges the system layer (`~/.fab-kit/config.yaml`) and the project layer per-key *before* resolution runs, so per-**scope** ownership is not tracked. When both scopes contribute to the same tier and name **different** providers, the merged tier reads as one layer and its values are attributed to the merged layer's `provider:` — the cutoff does not fire across that scope boundary. Concretely, a system-scope `agent.tiers.doing: {provider: codex, model: gpt-5.3-codex}` plus a project-scope `agent.tiers.doing: {provider: gemini}` resolves `model=gpt-5.3-codex` under `provider=gemini` (a codex model ID handed to the gemini CLI). This is **pinned, not endorsed** — `TestResolveCrossScopeCascadeLimitation` in `cmd/fab` (the layer that can compose both scopes) asserts the current bytes and marks them the documented limitation; cascade-aware ownership is deferred to a follow-up change. The workaround is to pin `model:`/`effort:` in the **same scope** as the `provider:` switch.

#### Scenario: a cross-provider tier fills from the provider, not the tier chain

- **GIVEN** `agent.tiers.default: { model: claude-fable-5, effort: medium }` + `agent.tiers.doing: { provider: codex }` + `providers.codex: { model: gpt-5.3-codex, effort: high }`
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `model=gpt-5.3-codex` and `effort=high` — the claude-owned `default`-tier values do not survive the switch

#### Scenario: a cross-provider tier with no fill resolves empty

- **GIVEN** `agent.tiers.doing: { provider: codex }` and no `providers.codex` fill
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `model=` is empty and no `effort=` line is emitted (never the claude-shaped built-in values), and `dispatch=codex exec` appears with both placeholder tokens dropped

### Requirement: Fixed, non-overridable stage → tier mapping

The stage→tier mapping is **fab-owned and NOT user-overridable** (`stageTiers` in `internal/agent`; no `stage_tiers` config, no per-stage escape hatch):

| Stage | Tier |
|-------|------|
| `intake` | `default` (advisory only — foreground) |
| `apply` | `doing` |
| `review` | `review` |
| `hydrate` | `hydrate` |
| `ship` | `fast` |
| `review-pr` | `doing` |

`review` and `review-pr` are deliberately in **different** tiers despite the shared word: `review` is the critic (discovers what's wrong from a diff); `review-pr` is responsive (fixes already-articulated feedback). `hydrate` is its own tier — memory writing runs on a different model/effort than apply's diff work. `agent.tiers` overrides *what a tier means* (budget), never *which stages belong to it* (taxonomy).

### Requirement: `fab resolve-agent <stage|tier> [--alias] [--provider <name>] [--model <id>] [--effort <level>]` resolution surface

`fab resolve-agent` SHALL accept a **stage** name OR a **role-tier** name positionally — a stage maps through the fixed mapping, a tier resolves directly, and **tier names are checked first**. The two name sets overlap only at **fixed points**: a name shared by a stage and a tier (`review`, `hydrate`) is one where the stage maps to that same-named tier (`stageTiers[name] == name`), so the tier-first check resolves such a name to the identical profile either interpretation would (`ship` is a stage but NOT a tier — it maps to `fast` — so `resolve-agent ship` and `resolve-agent fast` both resolve to the `fast` profile, one via the stage mapping, the other directly). It resolves the tier → `{provider, model, effort}` (project override per-field-merged over the `default` tier, over fab-kit's built-in) and emits, **verbatim, with NO validation**:

- `model=<id>` (always; empty = the inherit signal),
- `effort=<level>` (omitted when empty),
- `provider=<name>` (omitted when empty),
- `dispatch=<command>` — emitted when the resolved tier's provider carries a `dispatch_command`, **or** — with `dispatch.watchable: true` and the orchestrator inside tmux — for a `session_command`-only provider (the **watchable pane opt-in**, § Watchable pane dispatch below). Its absence signals native dispatch; a *headless* dispatch has no fallback to a session command. The command's `{model}`/`{effort}` are substituted via `spawn.WithProfile`, and the `{model}` is **ALWAYS the full model ID even under `--alias`** (an external CLI's `--model` flag takes a full ID; CLI dispatch never aliases).

`--alias` maps the `model=` line to the Claude-Code Agent-tool short alias (`opus`/`sonnet`/`haiku`/`fable`) — the Agent tool's `model` param is a hard enum that rejects full IDs; the `dispatch=` line is unaffected (full ID).

**Invocation-time overrides** are the fill precedence's top rung, applied to the resolved profile by `agent.ApplyOverrides`:

- **`--provider <name>`** swaps the provider and **re-derives `dispatch=` from the named provider's `dispatch_command`**, so the emitted `dispatch=` presence can differ from the stage's unoverridden one. A swap does **not** retain the tier's `model`/`effort` — those belong to the old provider, so they are foreign and refill from the new provider's default fill, then empty (the same cutoff § Fill precedence describes). One asymmetry when swapping *back*: `--provider claude` from a non-claude tier resolves an **empty** `model=`, because claude's fill lives on the built-in *tiers*, not on the provider, so there is no `providers.claude.model` rung to refill from — pair it with an explicit `--model` when the stage should run a specific claude model.
- **`--model <id>` / `--effort <level>`** override the corresponding field and are valid **without** `--provider` — a within-tier override of the profile this pure query would otherwise print. This is a deliberate, documented **asymmetry with `fab agent`**, where `--model`/`--effort` remain usage errors without `--provider` (a session launcher with two mutually exclusive addressing modes, where a bare `--model` would invent an undocumented tier-override surface).
- All three key on cobra's `Flag.Changed` — whether the flag was **supplied** — so `--model=` explicitly *clears* the tier's model (emitting the inherit signal) rather than being silently ignored, and `--provider=` is a lookup failure rather than a fall-through to the tier's provider.
- An **unknown `--provider` name** is a non-zero-exit **lookup** failure listing the resolvable names, byte-identical to `fab agent`'s because both call the shared `unknownProviderError(cfg, name)` helper in `cmd/fab`. Overrides themselves are applied with **no validation** (provider neutrality).

**Overrides bind the native Agent-tool arm only.** The emitted `dispatch=` line is a **query result**, not an adapter move: `fab dispatch start` takes no override flags and re-resolves the stage from config itself, so an overridden profile cannot ride the CLI adapter. Relocating a stage between native Agent-tool dispatch and CLI dispatch takes a **config/tier override**, never an invocation flag. See [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution for the dispatch-site wiring and the two-remedies rule.

#### Scenario: `--alias` aliases `model=` while `dispatch=` keeps the full ID

- **GIVEN** a tier resolving to a provider with a `dispatch_command`
- **WHEN** `fab resolve-agent <stage> --alias` runs
- **THEN** `model=` carries the short alias while `dispatch=` embeds the full model ID
- **AND** under `--provider codex --model gpt-5.3-codex --alias` the non-Claude model passes through **verbatim** on `model=` (no prefix matched) while `dispatch=` embeds the same full ID

#### Scenario: overriding a stage's provider on the resolve call

- **GIVEN** a default config (apply → `doing` → claude)
- **WHEN** `fab resolve-agent apply --provider codex` runs
- **THEN** `provider=codex`, `model=` is empty (no codex fill configured), no `effort=` line, and `dispatch=codex exec` appears with both placeholder tokens dropped
- **WHEN** `fab resolve-agent apply --effort high` runs (no `--provider`)
- **THEN** `provider=claude` with the tier's own model and `effort=high` — a within-tier override, not a usage error
- **WHEN** `fab resolve-agent apply --provider bogus` runs
- **THEN** it exits non-zero naming `bogus`, lists `claude, codex, gemini`, and prints no profile

### Requirement: `dispatch.watchable` — the watchable pane opt-in

`dispatch.watchable` (bool, default `false`, **scope `both`** — settable once machine-wide in `~/.fab-kit/config.yaml`; modeled on `Config.Dispatch.Watchable`, read via `GetDispatchWatchable()`) SHALL add a **second trigger** for the `dispatch=` line: when it is `true` **AND** `$TMUX` is set **AND** the resolved provider carries a `session_command` but **no** `dispatch_command`, `fab resolve-agent` emits `dispatch=` carrying the profile-substituted `session_command`.

- **Tmux presence decides pane vs native.** `$TMUX` unset ⇒ the line is omitted and the stage stays on **native Agent-tool dispatch**, never headless CLI (headless remains gated on a real `dispatch_command`). The env read lives in the cobra `RunE` layer; the emission rule itself is the pure `dispatchLineFor(prov, known, watchable, tmuxEnv)` helper (the `dispatch.SelectMode` precedent), so the whole matrix is table-testable.
- **A provider `dispatch_command` wins** — emission for a `dispatch_command`-carrying provider is unchanged; watchable only ADDS eligibility for providers that have none.
- **Why it exists.** Pane mode composes `session_command`, not `dispatch_command`, so pane *eligibility* was gated on a field pane mode never uses. Before the opt-in the only route to a watchable claude worker was uncommenting claude's `dispatch_command`, which ALSO flipped every out-of-tmux dispatch to **headless CLI** — a footgun disguised as a default.
- **No skill-wiring change.** The dispatch seam branches on the line's *presence* and never executes its value; `fab dispatch start` re-resolves internally and inside tmux its auto ladder selects pane mode, composing the same `session_command`. A `session_command`-only provider dispatches fine under pane mode (shipped zxe0/l9ng behavior).
- **`--alias` unaffected**: `dispatch=` always embeds the full model ID.
- **Known edge, documented not solved**: if tmux dies between the resolve and `fab dispatch start`, start's auto ladder soft-falls-back to headless and then errors on the missing `dispatch_command`. Rare, self-explaining at the CLI.

#### Scenario: watchable + tmux makes a session_command-only provider pane-eligible

- **GIVEN** `dispatch.watchable: true` and a tier resolving to the built-in `claude` (a `session_command`, no `dispatch_command`)
- **WHEN** `fab resolve-agent apply` runs with `$TMUX` set
- **THEN** the output carries a `dispatch=` line holding the substituted `session_command` (full model ID)
- **WHEN** the same command runs with `$TMUX` unset
- **THEN** no `dispatch=` line is emitted — the stage stays on native Agent-tool dispatch

#### Scenario: a provider dispatch_command outranks the opt-in

- **GIVEN** `dispatch.watchable: true`, `$TMUX` set, and a tier resolving to a provider carrying BOTH commands
- **WHEN** `fab resolve-agent <stage>` runs
- **THEN** `dispatch=` carries the **`dispatch_command`**, not the `session_command`

#### Scenario: the opt-in is settable once machine-wide

- **GIVEN** a `dispatch:` block setting `watchable: true` in `~/.fab-kit/config.yaml`, and a project config that never mentions `dispatch`
- **WHEN** `fab resolve-agent <stage>` runs inside tmux in that repo
- **THEN** the opt-in applies (scope `both` — the cascade honors it rather than pruning it)
- **AND** a project-level `dispatch.watchable: false` overrides it back off

### Requirement: `fab agent [tier] [--provider <name> [--model <id>] [--effort <level>]] [--print] [--repo <path>]` — session launcher

`fab agent` SHALL compose an interactive session command in one of **two mutually exclusive addressing modes** and **exec it in the current shell**:

- **Tier-addressed** (the `[tier]` positional) — resolve a tier profile (`default` when the tier is omitted; any of the six tier names accepted) and compose `providers.<profile.provider>.session_command` with the tier's `{model}`/`{effort}`. `fab agent` starts the default-tier agent right here; `fab agent operator` starts the coordinator profile.
- **Provider-addressed** (`--provider <name>`) — **bypass tier resolution entirely**: look up `providers.<name>` directly via `agent.ResolveProvider` (project config per-field-merged over the built-in table, exactly as the tier path's provider lookup does) and compose its `session_command` with the `--model`/`--effort` values. This is the "give me a codex session right here" form — no tier need name the provider first.

Both modes compose through the same `spawn.WithProfile` (template substitution or Claude-style flag append — see [configuration.md](/_shared/configuration.md) § `providers`) and share `--print`/`--repo`:

- `--print` prints the fully-resolved command instead of executing — the output is **profile-resolved** (model/effort substituted), so callers that spawn from the printed command get the profile.
- `--repo <path>` reads the target repo's config (the operator's fetch-another-repo's-command use case). Composes with either mode.
- `fab agent` exec does NOT TTY-guard — exec-and-let-the-CLI-fail is acceptable (the underlying agent CLI already handles no-TTY), matching the document-don't-validate contract.

Provider-mode rules:

- **Omitted `--model`/`--effort`** leave the value empty and follow `WithProfile`'s empty-value rule (template mode drops the placeholder's token plus a preceding `-`-flag; append mode omits the flag), so a bare provider invocation results and the installed CLI's own default model applies — the way to spawn a provider whose current model IDs the caller does not know. **This path bypasses the provider default fill too**: `providers.<name>.model`/`.effort` are deliberately NOT consulted by `fab agent --provider` (the bypass is total — it skips tier resolution *and* the fill), so an empty flag means empty, not the configured fill. The fill applies on the resolution surface (`fab resolve-agent`, tier resolution), not on this launcher.
- **`--model`/`--effort` require `--provider`** — supplying either alone is a usage error (non-zero exit).
- **`--provider` and the `[tier]` positional are mutually exclusive** — supplying both is a usage error naming the exclusion (a hand-written `RunE` check, since cobra's `MarkFlagsMutuallyExclusive` relates only flags and the tier is a positional).
- Both guards, and the mode selection itself, key on cobra's `Flag.Changed` — whether the flag was **supplied** — not on its value being non-empty, so `fab agent doing --provider=` and `fab agent --model= --print` still error rather than falling through to the tier path.
- **An unknown provider name** is a non-zero-exit **lookup** failure listing the available names (`agent.ProviderNames`: fab-kit's built-in table ∪ the project's `providers:` keys via `config.ProviderNames`, sorted). Listing resolvable *names* is not validation of a command's *content* — resolved strings still pass through verbatim.
- A provider that resolves but carries no `session_command` yields the `configure providers.<name>.session_command` hint error on either path.

The procedural knowledge for *using* a composed command — opening it in a tmux window, delivering a prompt, peeking, awaiting — plus the per-provider invocation grammar and model-discovery recipes live in the `_cli-agents` helper: see [agent-primitives.md](/runtime/agent-primitives.md).

#### Scenario: provider-addressed spawn with no model supplied

- **GIVEN** `providers.codex.session_command: 'codex -m {model} -c model_reasoning_effort={effort}'`
- **WHEN** `fab agent --provider codex --print` runs
- **THEN** stdout is a bare `codex` — both placeholder tokens and their preceding flags dropped — so the CLI's own default model applies
- **AND** `fab agent --provider codex --model gpt-5.3-codex --effort high --print` prints `codex -m gpt-5.3-codex -c model_reasoning_effort=high`

#### Scenario: unknown provider name

- **GIVEN** a config whose `providers:` block defines only `claude`
- **WHEN** `fab agent --provider bogus --print` runs
- **THEN** it exits non-zero, naming `bogus` and listing the available provider names, and prints no command

## Design Decisions

### Providers Extracted; Role Tiers; `fab agent` Retires `fab spawn-command` (tykw)
**Decision**: See the authoritative record in [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Providers Extracted from Tiers; Role Tiers; `review_tools` → `code-review.md`". In brief: extract a top-level `providers:` table (two unmerged command fields, claude built-in, absent `dispatch_command` = native, no cross-fallback); replace `thinking`/`doing`/`fast` with role tiers as `{provider, model, effort}` (dissolving `thinking`, splitting `review` out of `doing`); retire `review_tools` into `code-review.md` § Review Tools; add `fab agent` (retiring `fab spawn-command`); rename `resolve-agent`'s `spawn=`→`dispatch=` and add tier-name acceptance + a `provider=` line.
**Why**: The pre-v3 config conflated provider mechanics with role/budget policy and the names actively confused (two fields both named `spawn_command`; the `thinking` tier's referent was hidden — it "meant" review). Extraction + role naming attack the confusion at its source; commands leaving the tier make `{provider, model, effort}` inheritance safe (no cross-semantics command inheritance).
**Rejected**: Merging the two command fields; folding `agent.spawn_command` in as a `default`-tier command (implies the rejected 3a–3d fallback); keeping `thinking`; provider inference from model strings; a `fab spawn-command` deprecation alias.
*Introduced by*: 260702-tykw-agent-providers-role-tiers

### Positional Stage-or-Tier; `provider=` Line; No TTY Guard (tykw)
**Decision**: `fab resolve-agent` accepts a stage OR tier name positionally (tier names checked first; the only shared names are fixed points where either interpretation resolves identically, so no `--tier` flag is needed); its output gains a `provider=` line (needed by `fab agent`/operator for the session-command lookup, and it aids compliance visibility); `--alias` on a native (no-`dispatch_command`) tier aliases only the `model=` line; and `fab agent` exec does not TTY-guard.
**Why**: Reuse the existing positional surface rather than add flag surface for no disambiguation benefit; surface the provider rather than re-derive it downstream; keep the no-validation/document-don't-guard contract for TTY.
**Rejected**: A `--tier` flag (surface for no benefit); inferring provider downstream (re-does resolution); a TTY guard (the agent CLI already handles no-TTY).
*Introduced by*: 260702-tykw-agent-providers-role-tiers

### `hydrate` Is Its Own Tier; `fast` Keeps Its Role Name (Multi-Referent)
**Decision**: Split `hydrate` out of `doing` into its own tier, giving six role tiers (`default`/`operator`/`doing`/`review`/`hydrate`/`fast`); the hydrate stage→tier row is the only mapping change. A tier is stage-named only where it maps 1:1 to a single referent (`review`, `hydrate`); `default`, `doing`, and `fast` keep role names because each is multi-referent — `fast` in particular governs both the ship stage and the `/fab-proceed` prefix-step dispatches.
**Why**: Memory writing (hydrate) is knowledge work with a different cognitive profile than apply's diff work, so it deserves its own model/effort — grouped under `doing` it could never run cheaper or on a different model than apply. A stage name (`ship`) would misname `fast` once it also governs the prefix steps.
**Rejected**: Renaming `fast`→`ship` (misnames a multi-referent tier and would force an unnecessary carry-forward migration); six stage-named tiers / dissolving role tiers entirely (`default`/`doing`/`fast` are genuinely multi-referent and the role names carry the why); a user-overridable `stage_tiers:` mapping or per-stage escape hatch (taxonomy stays fab-owned).
*Introduced by*: 260719-g55d-stage-model-tier-defaults-v2

### `--provider` Is a Sibling Addressing Mode, Not a Tier Override
**Decision**: `fab agent --provider <name>` bypasses tier resolution entirely — a direct `providers.<name>` lookup with `--model`/`--effort` as the profile — rather than synthesizing an ad-hoc tier or overriding a named tier's fields. `--model`/`--effort` are usage errors without `--provider`, and `--provider` is mutually exclusive with the `[tier]` positional. All three guards key on cobra's `Flag.Changed` (was the flag supplied?) rather than value emptiness.
**Why**: A tier is a `{provider, model, effort}` role profile owned by fab-kit's fixed mapping; provider-addressed spawning answers a different, mechanical question ("give me a codex session"), and a tier already names a provider, so mixing them has no coherent semantics. Bypass leaves `ResolveTier` untouched, so no existing path changes behavior. On the tier path the profile IS the tier's, resolved through inheritance, so a bare `--model` would either invent an undocumented tier-override surface or be silently ignored — an explicit usage error is the only honest option and is trivially relaxable later. Emptiness-based guards would let `--provider=` and `--model=` fall silently through to the tier path, so supplied-ness is the correct test.
**Rejected**: A `--tier-provider`-style override (mutates role/budget policy to express a mechanics question); auto-creating a synthetic tier (invents state the config never declared); letting `--model` override a resolved tier's model (a second, undocumented tier-override surface); silently ignoring the flags (surprise-inducing CLI behavior); cobra's `MarkFlagsMutuallyExclusive` for the tier pairing (it relates flags only — the tier is a positional).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

### Built-in Provider Grammar in Go, Fill Values in Config
**Decision**: fab-kit's `defaultProviders` table carries the invocation *grammar* for three built-in providers (claude, codex, gemini) as named Go constants, and carries **no** model/effort fill for any of them. The volatile half lives in user config — `providers.<name>.model`/`.effort`, a tier field, or an invocation flag. This narrowly reverses ho9y's "no new built-in providers are added in Go — codex/gemini are template text only"; the reversal covers grammar strings only.
**Why**: Grammar changes at binary-release cadence and is safe to ship; model IDs rot in weeks and would make every fab release carry stale strings. Splitting them lets a fresh project name `codex` with zero config while the volatile half is settable once per machine via the `scope: both` system layer. ho9y's presence=intent reasoning is *preserved*, not contradicted: a built-in provider is inert until a tier or flag names it, so adding a row changes no default behavior. What ho9y additionally assumed — that a built-in row would need fill values — is exactly what the per-provider fill slot removes. Keeping codex/gemini as commented template text left cross-provider work config-gated at the moment it should be frictionless (mid-session "work with codex").
**Rejected**: Baking model IDs into Go (rot); leaving codex/gemini as commented template text (the ho9y state); inferring a provider from a model string (breaks provider neutrality); reversing ho9y silently (a reader of the config fence would meet a contradicted claim with no record).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

### An Explicit `provider:` Cuts Off Cross-Provider Field Inheritance, Per Field
**Decision**: When a tier's provider comes from an explicit config `provider:` differing from the built-in tier profile's, every `model`/`effort` **owned by another provider** refills from the resolved provider's default fill, then empty. Ownership is per field and bottom-up — the provider in effect at the layer that supplied the value, with `provider:` applied before `model`/`effort` within a layer. The gate stays anchored to the *net* configured provider vs. the built-in's, so a chain ending back on the built-in's provider is not a switch. A `--provider` swap on `fab resolve-agent` runs the same rule; both paths call one `cutForeignFields` helper, differing only in how they record owners.
**Why**: Per-field inheritance across a provider switch supplies another provider's model — a footgun only documentable while no correct fill value existed. The per-provider fill slot supplies one, so the rule closes it. Per-field ownership is what makes the rule hold across all three merged layers: a single flattened "the config set this" bit cannot distinguish a claude-owned `default`-tier model from a model written beside the switch, so it lets the former survive a codex switch. The net-provider gate (rather than a per-layer fold) keeps an intermediate switch that the top layer reverses from cutting values the reversal should preserve. The all-claude default world is byte-unchanged because every built-in tier pins an explicit model.
**Rejected**: Documenting the footgun instead of closing it (a correct fill value exists); a flattened "was this configured?" bit (lets foreign `default`-tier values survive a switch); a pure per-layer fold with no net gate (cuts on an intermediate switch the top layer reverses); erroring on a cross-provider tier with no fill (a resolvable empty model is the established inherit/CLI-default signal and is strictly more useful); validating provider/model compatibility (breaks provider neutrality); retaining the tier's model across a `--provider` swap (precisely the footgun the rule closes).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

### Ownership Sees the Merged Config; Cross-Scope Cascade Is a Documented Limitation
**Decision**: The cutoff's per-field ownership is computed over the **merged** config (`builtin ← project default tier ← requested tier`), not over the system↔project cascade, which `config.LoadPath` deep-merges per key before resolution. When both scopes name different providers for one tier, the cutoff does not fire across that boundary. The limitation is stated in `internal/agent`'s ownership doc comment (which deliberately does not overclaim about scopes it cannot see) and in `docs/specs/stage-models.md`, and pinned by `TestResolveCrossScopeCascadeLimitation` in `cmd/fab` — the layer that can compose two scopes — which asserts the current bytes with a comment marking them the limitation. No runtime warning ships.
**Why**: Detecting the condition requires the per-scope layers at resolution time, which is exactly the cascade-aware ownership being deferred — so a warning is not a cheaper subset of the fix, it *is* the fix's hard half. A warning would also fire on the legitimate case of deliberately re-pointing a tier machine-wide. Pinning the behavior by test makes the gap visible and makes a future fix a deliberate, reviewed change rather than an accidental one. The test lives in `cmd/fab` because `internal/agent` takes an already-merged config and structurally cannot compose scopes — reproducing it there would pin the merge's output rather than the cascade that produces it.
**Rejected**: Folding per-scope layers in `ResolveTier` now (the deferred follow-up — a larger change than the footgun fix it would ride along with); a `fab: warning:` on the divergent-scope condition (needs the deferred machinery; false-positives on deliberate machine-wide re-pointing); leaving the overclaiming doc comment in place (asserts a guarantee the code cannot make); hand-forging a merged tree in `internal/agent` to pin it there (pins the wrong seam).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

### Overrides Land on `resolve-agent`, Not New Dispatch Machinery — and Bind the Native Arm Only
**Decision**: Invocation-time provider/model/effort overrides are flags on `fab resolve-agent`, the single resolution call every dispatch site already makes, and the skill wiring is one passthrough paragraph. On that surface `--model`/`--effort` are valid without `--provider` (a within-tier override), while `fab agent` keeps its shipped requires-`--provider` rule. Because `fab dispatch start` carries no override surface — it re-resolves the stage from config — the overrides bind the **native Agent-tool arm** only, and every doc restating the surface carries that scope.
**Why**: Every dispatch site already makes exactly one `resolve-agent --alias` call and already branches on `dispatch=`, so overriding at that call reuses the whole seam with zero new machinery; a separate override channel would need its own precedence rules and its own compliance-visibility contract. `resolve-agent` is a pure query whose whole output is a profile, so overriding one field of it is unambiguous and useful; `fab agent` is a session launcher with two mutually exclusive addressing modes, where a bare `--model` would invent an undocumented tier-override surface. Scoping to the native arm keeps the docs describing what the code does instead of describing a workflow that errors on the headless arm and silently runs the wrong provider on the pane arm.
**Rejected**: Plumbing the flags through `fab dispatch start` (a second resolution surface with its own precedence and visibility contract — the thing this decision rejects); per-stage override config keys (persistent state for a per-run intent); named tier-profile sets `agent.profiles.*` (deferred — per-stage flags cover "use codex for the next N stages"); relaxing `fab agent` to match (re-litigates a deliberate shipped decision); forbidding bare `--model` on `resolve-agent` for symmetry (a usage error for an unambiguous query).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

### Unknown Provider Is a Lookup Failure That Names the Resolvable Set
**Decision**: An unknown `--provider` name exits non-zero listing the available provider names — the sorted union of fab-kit's built-in table and the project's `providers:` keys, exposed as `agent.ProviderNames(cfg)` over the nil-safe `config.ProviderNames()` accessor. The message is produced by a single `unknownProviderError(cfg, name)` helper in `cmd/fab`, shared verbatim by both flag-accepting commands (`fab agent`'s provider-addressed mode and `fab resolve-agent --provider`).
**Why**: The union is exactly the set `ResolveProvider` will accept, so it is the only set whose listing is actionable; sorting makes the message stable for tests and for readers. Naming resolvable *names* is not validation of command *content* — resolved command strings still pass through verbatim, preserving document-don't-validate (fab never infers a provider from a model string). One helper keeps the two commands' errors byte-identical, so a caller learns one phrasing.
**Rejected**: Listing only the project's configured providers (omits the built-in `claude`, which resolves fine); a bare "unknown provider" error (leaves the caller to guess the config surface); validating the resolved command string (breaks the document-don't-validate contract); a per-command copy of the formatter (two phrasings of one contract to drift).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

## Consumers

The provider/tier resolution feeds three runtime consumers:

- **The dispatch seam** (`/fab-ff`, `/fab-fff`, `/fab-proceed`, `/fab-adopt`, and `/fab-continue`'s one-stage sequencer) calls `fab resolve-agent <stage> --alias` before each post-intake stage's sub-agent and **branches on the resolved `dispatch=` line**: absent ⇒ native Agent-tool dispatch (model via the Agent `model` param, effort via a prompt instruction); present ⇒ the CLI adapter `fab dispatch` (the profile rides the `dispatch=` command). See [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution and [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership.
- **The operator launcher** (`fab operator`) resolves the **operator** tier in-process and composes its session command from that tier's provider `session_command` + profile. See [operator.md](/runtime/operator.md).
- **Batch worker spawns** (`fab batch new`/`switch` and the operator's repo-targeted worker spawns) compose from the **default-tier** provider `session_command` + the default profile — so workers spawn WITH a profile. See [operator.md](/runtime/operator.md) and [distribution/kit-architecture.md](/distribution/kit-architecture.md).

The `dispatch_command` a tier's provider carries is *run* by [`fab dispatch`](/runtime/dispatch.md) (the headless process manager); this file and `fab resolve-agent` only *resolve and emit* it. The same holds for a `session_command` emitted under the `dispatch.watchable` opt-in: `fab dispatch start` re-resolves and composes it itself under pane mode.
