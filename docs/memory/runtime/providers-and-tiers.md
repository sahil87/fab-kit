---
type: memory
description: "The providers & role-tiers model (agent config v3) — the top-level `providers:` command table (opaque names → `session_command`/`dispatch_command`, claude built-in, absent `dispatch_command` = native, NO cross-fallback), the six role tiers as `{provider, model, effort}` with per-field inheritance, the fixed stage→tier mapping + fixed-point name rule, `fab resolve-agent`, the `fab agent` launcher's two addressing modes (tier, or `--provider` bypassing tiers), and the resolution's consumers."
---
# Providers & Role Tiers

**Domain**: runtime

## Overview

Agent config v3 (260702-tykw) splits **provider mechanics** (how to invoke an agent) from **role/budget policy** (which model + effort a stage runs at). Providers live in a top-level `providers:` command table; tiers are `{provider, model, effort}` role profiles under `agent.tiers`. This file is the model — the provider table, the six role tiers and their inheritance, the fixed stage→tier mapping, the `fab resolve-agent`/`fab agent` surfaces, and who consumes the resolution. The **config-schema authority** is [_shared/configuration.md](/_shared/configuration.md) § `providers` and § `agent`; the **dispatch-seam wiring** is [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution; the pre-implementation design intent is `docs/specs/stage-models.md` (drift-guarded against the Go maps). This file ties them together for the runtime reader.

## Requirements

### Requirement: Providers table — session vs dispatch

`fab/project/config.yaml` SHALL support a top-level `providers:` map keyed by **opaque, user-chosen provider names**. Each provider MAY carry two command fields, which SHALL NOT be merged into one:

- **`session_command`** — opens an interactive agent **session**. Consumed by `fab operator`, `fab batch new`/`batch switch`, `fab agent`, and `fab dispatch start --pane` (which runs a *stage* in an interactive session the user can watch and steer — see [dispatch.md](/runtime/dispatch.md)).
- **`dispatch_command`** — runs ONE headless **stage task** via `fab dispatch`. **ABSENT `dispatch_command` = native Agent-tool dispatch** — there is **NO fallback** to `session_command`.

**The no-cross-fallback rule holds in both directions**: headless dispatch never substitutes `session_command`, and `--pane` never substitutes `dispatch_command` — each mode errors naming the field it needs. `--pane` composing `session_command` is **not** a fallback and not a resolver change: mode selection is per-invocation, `fab resolve-agent`'s output is byte-identical either way, and pane mode reads the provider table itself exactly as `fab agent` does. No third command field exists — the interactive stage invocation *is* the provider's session invocation.

fab-kit ships the **`claude` provider as the built-in default** (`defaultProviders` in `internal/agent`): the default `session_command`, no `dispatch_command` (native). A project's `providers:` block per-field-merges over the built-in via `agent.ResolveProvider(name)`.

```yaml
providers:
  claude:
    session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}'
    # no dispatch_command → claude's stages dispatch natively via the Agent tool
  codex:
    session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
    dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'
```

The two fields are deliberately unmerged because session and dispatch are **different invocations of the same binary** (claude interactive `-n <name>` vs headless `-p`; codex TUI vs `codex exec`) — no single template expresses both. `{model}`/`{effort}` placeholders in either command are substituted at resolve time via `spawn.WithProfile` (reused, not reimplemented — see [configuration.md](/_shared/configuration.md) § `providers` for the template/append modes and the empty-value token-drop rule).

**On-ramp for adding a provider — the three-provider starter template (ho9y).** A user does not compose these command strings from scratch. Both `fab config reference` and the new-project scaffold ship the `providers:` block as a **three-provider (claude/codex/gemini) starter template**: `claude.session_command` live, and claude's `dispatch_command` + the whole codex/gemini blocks commented, ready to uncomment-and-adapt. This is template TEXT only — the Go `defaultProviders` table stays claude-only; codex/gemini are guidance until a user uncomments them. Two shipped specifics worth carrying: gemini's strings omit `{effort}` (the gemini CLI has no reasoning-effort flag), and gemini's `dispatch_command` is the bare `gemini -m {model}` with **no `-p`** — `fab dispatch` pipes the prompt to gemini's stdin (which it reads as the prompt in non-TTY mode), whereas `-p` takes prompt TEXT appended after stdin. See [configuration.md](/_shared/configuration.md) § `providers` → "Three-provider starter template" for the full shipped strings and the parse-side/whole-block-uncomment guarantees. (The codex block in the schema snippet above is shown live purely as a schema illustration — the *shipped* reference/scaffold present codex commented.)

**Provider names are opaque — fab NEVER infers a provider from a model string.** The one documented footgun: override a tier's `model` cross-provider ⇒ override its `provider` too. fab documents this, it does not validate it.

#### Scenario: absent `dispatch_command` selects native dispatch

- **GIVEN** a stage whose tier points at the built-in `claude` provider (no `dispatch_command`)
- **WHEN** that stage is dispatched
- **THEN** it runs as a native Agent-tool sub-agent — `fab resolve-agent` emits no `dispatch=` line, and there is no fallback to `session_command`

#### Scenario: provider `dispatch_command` drives CLI dispatch

- **GIVEN** `providers.codex.dispatch_command` and a tier `{ provider: codex, ... }`
- **WHEN** a stage on that tier is dispatched
- **THEN** `fab dispatch` runs the resolved `codex exec …` command (cross-harness), profile substituted

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

### Requirement: `fab resolve-agent <stage|tier>` resolution surface

`fab resolve-agent` SHALL accept a **stage** name OR a **role-tier** name positionally — a stage maps through the fixed mapping, a tier resolves directly, and **tier names are checked first**. The two name sets overlap only at **fixed points**: a name shared by a stage and a tier (`review`, `hydrate`) is one where the stage maps to that same-named tier (`stageTiers[name] == name`), so the tier-first check resolves such a name to the identical profile either interpretation would (`ship` is a stage but NOT a tier — it maps to `fast` — so `resolve-agent ship` and `resolve-agent fast` both resolve to the `fast` profile, one via the stage mapping, the other directly). It resolves the tier → `{provider, model, effort}` (project override per-field-merged over the `default` tier, over fab-kit's built-in) and emits, **verbatim, with NO validation**:

- `model=<id>` (always; empty = the inherit signal),
- `effort=<level>` (omitted when empty),
- `provider=<name>` (omitted when empty),
- `dispatch=<command>` — emitted **ONLY when the resolved tier's provider carries a `dispatch_command`** (its absence signals native dispatch; NO fallback). The command's `{model}`/`{effort}` are substituted via `spawn.WithProfile`, and the `{model}` is **ALWAYS the full model ID even under `--alias`** (an external CLI's `--model` flag takes a full ID; CLI dispatch never aliases).

`--alias` maps the `model=` line to the Claude-Code Agent-tool short alias (`opus`/`sonnet`/`haiku`/`fable`) — the Agent tool's `model` param is a hard enum that rejects full IDs; the `dispatch=` line is unaffected (full ID).

#### Scenario: `--alias` aliases `model=` while `dispatch=` keeps the full ID

- **GIVEN** a tier resolving to a provider with a `dispatch_command`
- **WHEN** `fab resolve-agent <stage> --alias` runs
- **THEN** `model=` carries the short alias while `dispatch=` embeds the full model ID

### Requirement: `fab agent [tier] [--provider <name> [--model <id>] [--effort <level>]] [--print] [--repo <path>]` — session launcher

`fab agent` SHALL compose an interactive session command in one of **two mutually exclusive addressing modes** and **exec it in the current shell**:

- **Tier-addressed** (the `[tier]` positional) — resolve a tier profile (`default` when the tier is omitted; any of the six tier names accepted) and compose `providers.<profile.provider>.session_command` with the tier's `{model}`/`{effort}`. `fab agent` starts the default-tier agent right here; `fab agent operator` starts the coordinator profile.
- **Provider-addressed** (`--provider <name>`) — **bypass tier resolution entirely**: look up `providers.<name>` directly via `agent.ResolveProvider` (project config per-field-merged over the built-in table, exactly as the tier path's provider lookup does) and compose its `session_command` with the `--model`/`--effort` values. This is the "give me a codex session right here" form — no tier need name the provider first.

Both modes compose through the same `spawn.WithProfile` (template substitution or Claude-style flag append — see [configuration.md](/_shared/configuration.md) § `providers`) and share `--print`/`--repo`:

- `--print` prints the fully-resolved command instead of executing — the output is **profile-resolved** (model/effort substituted), so callers that spawn from the printed command get the profile.
- `--repo <path>` reads the target repo's config (the operator's fetch-another-repo's-command use case). Composes with either mode.
- `fab agent` exec does NOT TTY-guard — exec-and-let-the-CLI-fail is acceptable (the underlying agent CLI already handles no-TTY), matching the document-don't-validate contract.

Provider-mode rules:

- **Omitted `--model`/`--effort`** leave the value empty and follow `WithProfile`'s empty-value rule (template mode drops the placeholder's token plus a preceding `-`-flag; append mode omits the flag), so a bare provider invocation results and the installed CLI's own default model applies — the way to spawn a provider whose current model IDs the caller does not know.
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

### Unknown Provider Is a Lookup Failure That Names the Resolvable Set
**Decision**: An unknown `--provider` name exits non-zero listing the available provider names — the sorted union of fab-kit's built-in table and the project's `providers:` keys, exposed as `agent.ProviderNames(cfg)` over the nil-safe `config.ProviderNames()` accessor.
**Why**: The union is exactly the set `ResolveProvider` will accept, so it is the only set whose listing is actionable; sorting makes the message stable for tests and for readers. Naming resolvable *names* is not validation of command *content* — resolved command strings still pass through verbatim, preserving document-don't-validate (fab never infers a provider from a model string).
**Rejected**: Listing only the project's configured providers (omits the built-in `claude`, which resolves fine); a bare "unknown provider" error (leaves the caller to guess the config surface); validating the resolved command string (breaks the document-don't-validate contract).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

## Consumers

The provider/tier resolution feeds three runtime consumers:

- **The dispatch seam** (`/fab-ff`, `/fab-fff`, `/fab-proceed`, `/fab-adopt`, and `/fab-continue`'s one-stage sequencer) calls `fab resolve-agent <stage> --alias` before each post-intake stage's sub-agent and **branches on the resolved `dispatch=` line**: absent ⇒ native Agent-tool dispatch (model via the Agent `model` param, effort via a prompt instruction); present ⇒ the CLI adapter `fab dispatch` (the profile rides the `dispatch=` command). See [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution and [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership.
- **The operator launcher** (`fab operator`) resolves the **operator** tier in-process and composes its session command from that tier's provider `session_command` + profile. See [operator.md](/runtime/operator.md).
- **Batch worker spawns** (`fab batch new`/`switch` and the operator's repo-targeted worker spawns) compose from the **default-tier** provider `session_command` + the default profile — so workers spawn WITH a profile. See [operator.md](/runtime/operator.md) and [distribution/kit-architecture.md](/distribution/kit-architecture.md).

The `dispatch_command` a tier's provider carries is *run* by [`fab dispatch`](/runtime/dispatch.md) (the headless process manager); this file and `fab resolve-agent` only *resolve and emit* it.
