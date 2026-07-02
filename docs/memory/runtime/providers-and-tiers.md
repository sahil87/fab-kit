---
type: memory
description: "The providers & role-tiers model (agent config v3, tykw) — the top-level `providers:` command table (opaque names → `session_command`/`dispatch_command`, the session-vs-dispatch split, claude built-in, absent `dispatch_command` = native, NO cross-fallback), the three-provider (claude/codex/gemini) starter template shipped by `fab config reference` + the scaffold as the on-ramp for adding a provider (template text only, gemini no `{effort}` + bare `gemini -m {model}` dispatch — ho9y), the five role tiers `default`/`operator`/`doing`/`review`/`fast` as `{provider, model, effort}` with per-field default-tier inheritance, the fixed non-overridable stage→tier mapping, `fab resolve-agent <stage|tier>` (spawn=→dispatch=, `provider=` line, tier-name acceptance, full-ID dispatch under --alias), the `fab agent [tier] [--print] [--repo]` launcher (retiring `fab spawn-command`), and where the provider/tier resolution is consumed (dispatch seam, operator launcher, batch worker spawns)"
---
# Providers & Role Tiers

**Domain**: runtime

## Overview

Agent config v3 (260702-tykw) splits **provider mechanics** (how to invoke an agent) from **role/budget policy** (which model + effort a stage runs at). Providers live in a top-level `providers:` command table; tiers are `{provider, model, effort}` role profiles under `agent.tiers`. This file is the model — the provider table, the five role tiers and their inheritance, the fixed stage→tier mapping, the `fab resolve-agent`/`fab agent` surfaces, and who consumes the resolution. The **config-schema authority** is [_shared/configuration.md](/_shared/configuration.md) § `providers` and § `agent`; the **dispatch-seam wiring** is [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution; the pre-implementation design intent is `docs/specs/stage-models.md` (drift-guarded against the Go maps). This file ties them together for the runtime reader.

## Requirements

### Requirement: Providers table — session vs dispatch

`fab/project/config.yaml` SHALL support a top-level `providers:` map keyed by **opaque, user-chosen provider names**. Each provider MAY carry two command fields, which SHALL NOT be merged into one:

- **`session_command`** — opens an interactive agent **session** (the relocated `agent.spawn_command`). Consumed by `fab operator`, `fab batch new`/`batch switch`, and `fab agent`.
- **`dispatch_command`** — runs ONE headless **stage task** via `fab dispatch` (the relocated per-tier `spawn_command`). **ABSENT `dispatch_command` = native Agent-tool dispatch** — there is **NO fallback** to `session_command`.

fab-kit ships the **`claude` provider as the built-in default** (`defaultProviders` in `internal/agent`): the default `session_command`, no `dispatch_command` (native). A project's `providers:` block per-field-merges over the built-in via `agent.ResolveProvider(name)`.

```yaml
providers:
  claude:
    session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")"'
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

### Requirement: Five role tiers with per-field default-tier inheritance

`agent.tiers` keys SHALL be the five **role tiers** — `default`, `operator`, `doing`, `review`, `fast` — replacing the former `thinking`/`doing`/`fast` cognitive-mode vocabulary. Each tier value SHALL be `{provider, model, effort}` (no command — the command lives on the provider). `thinking` is removed entirely: with `review` split into its own tier, `thinking`'s only remaining stage would be intake, which never dispatches.

fab-kit's built-in default profiles (owned by `defaultTiers` in `internal/agent`, drift-guarded against `docs/specs/stage-models.md`):

| Tier | Role | Built-in default profile |
|------|------|--------------------------|
| `default` | intake (advisory, foreground); `fab batch` worker sessions; `fab agent` with no tier; **per-field fallback for every other tier** | `claude` / `claude-fable-5` / `xhigh` |
| `operator` | the operator coordinator session (`fab operator`) | `claude` / `claude-sonnet-5` / `medium` |
| `doing` | `apply`, `review-pr`, `hydrate` — execution that must not err | `claude` / `claude-opus-4-8` / `xhigh` |
| `review` | `review` — the critic (author/critic separation) | `claude` / `claude-fable-5` / `xhigh` |
| `fast` | `ship` — near-mechanical work | `claude` / `claude-sonnet-5` / `low` |

**Per-field inheritance**: any tier field left unset (provider, model, effort) inherits from the project's `default` tier, then from fab-kit's built-in for that tier (`ResolveTier` middle-layer merge). Inheriting `{provider, model, effort}` is safe *because commands moved to `providers:`* — the dangerous cross-semantics command inheritance can no longer happen. **Documented style: write `provider:` explicitly on every tier line** even though inheritance makes it optional (per-line readability; inheritance is the safety net). Model IDs are written **versioned** (`claude-opus-4-8`) — bare family IDs fail both dispatch seams.

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
| `hydrate` | `doing` |
| `ship` | `fast` |
| `review-pr` | `doing` |

`review` and `review-pr` are deliberately in **different** tiers despite the shared word: `review` is the critic (discovers what's wrong from a diff); `review-pr` is responsive (fixes already-articulated feedback). `agent.tiers` overrides *what a tier means* (budget), never *which stages belong to it* (taxonomy).

### Requirement: `fab resolve-agent <stage|tier>` resolution surface

`fab resolve-agent` SHALL accept a **stage** name OR a **role-tier** name positionally (the two sets are disjoint — a stage maps through the fixed mapping, a tier resolves directly). It resolves the tier → `{provider, model, effort}` (project override per-field-merged over the `default` tier, over fab-kit's built-in) and emits, **verbatim, with NO validation**:

- `model=<id>` (always; empty = the inherit signal),
- `effort=<level>` (omitted when empty),
- `provider=<name>` (omitted when empty),
- `dispatch=<command>` — emitted **ONLY when the resolved tier's provider carries a `dispatch_command`** (its absence signals native dispatch; NO fallback). The command's `{model}`/`{effort}` are substituted via `spawn.WithProfile`, and the `{model}` is **ALWAYS the full model ID even under `--alias`** (an external CLI's `--model` flag takes a full ID; CLI dispatch never aliases).

`--alias` maps the `model=` line to the Claude-Code Agent-tool short alias (`opus`/`sonnet`/`haiku`/`fable`) — the Agent tool's `model` param is a hard enum that rejects full IDs; the `dispatch=` line is unaffected (full ID). This renamed the pre-tykw optional third line `spawn=` → `dispatch=` to match the provider field.

#### Scenario: `--alias` aliases `model=` while `dispatch=` keeps the full ID

- **GIVEN** a tier resolving to a provider with a `dispatch_command`
- **WHEN** `fab resolve-agent <stage> --alias` runs
- **THEN** `model=` carries the short alias while `dispatch=` embeds the full model ID

### Requirement: `fab agent [tier] [--print] [--repo <path>]` — session launcher

`fab agent` SHALL resolve a tier profile (`default` when the tier is omitted; any of the five tier names accepted), compose `providers.<provider>.session_command` with `{model}`/`{effort}` substituted (or Claude-style flags appended for a non-templated command via `spawn.WithProfile`), and **exec it in the current shell** — `fab agent` starts the default-tier agent right here; `fab agent operator` starts the coordinator profile.

- `--print` prints the fully-resolved command instead of executing — **this replaces `fab spawn-command`**, with a semantic upgrade: the output is **profile-resolved** (model/effort substituted), not placeholder-stripped as `fab spawn-command` was, so callers that spawn from the printed command finally get the tier profile.
- `--repo <path>` reads the target repo's config (the operator's fetch-another-repo's-command use case, carried over from `fab spawn-command --repo`).
- `fab agent` exec does NOT TTY-guard — exec-and-let-the-CLI-fail is acceptable (the underlying agent CLI already handles no-TTY), matching the document-don't-validate contract.

`fab spawn-command` is **removed in the same release** (no deprecation alias — its only CLI consumer, the operator skill, ships and is updated in the same kit).

## Design Decisions

### Providers Extracted; Five Role Tiers; `fab agent` Retires `fab spawn-command` (tykw)
**Decision**: See the authoritative record in [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Providers Extracted from Tiers; Five Role Tiers; `review_tools` → `code-review.md`". In brief: extract a top-level `providers:` table (two unmerged command fields, claude built-in, absent `dispatch_command` = native, no cross-fallback); replace `thinking`/`doing`/`fast` with the five role tiers as `{provider, model, effort}` (dissolving `thinking`, splitting `review` out of `doing`); retire `review_tools` into `code-review.md` § Review Tools; add `fab agent` (retiring `fab spawn-command`); rename `resolve-agent`'s `spawn=`→`dispatch=` and add tier-name acceptance + a `provider=` line.
**Why**: The pre-v3 config conflated provider mechanics with role/budget policy and the names actively confused (two fields both named `spawn_command`; the `thinking` tier's referent was hidden — it "meant" review). Extraction + role naming attack the confusion at its source; commands leaving the tier make `{provider, model, effort}` inheritance safe (no cross-semantics command inheritance).
**Rejected**: Merging the two command fields; folding `agent.spawn_command` in as a `default`-tier command (implies the rejected 3a–3d fallback); keeping `thinking`; provider inference from model strings; a `fab spawn-command` deprecation alias.
*Introduced by*: 260702-tykw-agent-providers-role-tiers

### Positional Stage-or-Tier; `provider=` Line; No TTY Guard (tykw)
**Decision**: `fab resolve-agent` accepts a stage OR tier name positionally (disjoint name sets make it unambiguous — no `--tier` flag); its output gains a `provider=` line (needed by `fab agent`/operator for the session-command lookup, and it aids compliance visibility); `--alias` on a native (no-`dispatch_command`) tier aliases only the `model=` line; and `fab agent` exec does not TTY-guard.
**Why**: Reuse the existing positional surface rather than add flag surface for no disambiguation benefit; surface the provider rather than re-derive it downstream; keep the no-validation/document-don't-guard contract for TTY.
**Rejected**: A `--tier` flag (surface for no benefit); inferring provider downstream (re-does resolution); a TTY guard (the agent CLI already handles no-TTY).
*Introduced by*: 260702-tykw-agent-providers-role-tiers

## Consumers

The provider/tier resolution feeds three runtime consumers:

- **The dispatch seam** (`/fab-ff`, `/fab-fff`, `/fab-proceed`, `/fab-adopt`, and `/fab-continue`'s one-stage sequencer) calls `fab resolve-agent <stage> --alias` before each post-intake stage's sub-agent and **branches on the resolved `dispatch=` line**: absent ⇒ native Agent-tool dispatch (model via the Agent `model` param, effort via a prompt instruction); present ⇒ the CLI adapter `fab dispatch` (the profile rides the `dispatch=` command). See [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution and [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership.
- **The operator launcher** (`fab operator`) resolves the **operator** tier in-process (previously it borrowed the doing tier via `fab resolve-agent apply`) and composes its session command from that tier's provider `session_command` + profile. See [operator.md](/runtime/operator.md).
- **Batch worker spawns** (`fab batch new`/`switch` and the operator's repo-targeted worker spawns) compose from the **default-tier** provider `session_command` + the default profile — so workers spawn WITH a profile (the pre-tykw placeholder-stripping print path disappeared with `fab spawn-command`). See [operator.md](/runtime/operator.md) and [distribution/kit-architecture.md](/distribution/kit-architecture.md).

The `dispatch_command` a tier's provider carries is *run* by [`fab dispatch`](/runtime/dispatch.md) (the headless process manager); this file and `fab resolve-agent` only *resolve and emit* it.
