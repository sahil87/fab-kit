---
type: memory
description: "The `_cli-agents` helper — agent-CLI interaction primitives any skill opts into via `helpers:`: spawn composition from `fab agent --print` (role- or provider-addressed; the provider form bypasses role resolution AND the fills), the pane-exists/agent-idle pre-send gate, the printed-prompt probe, peek (capture + `@rk_agent_state`; uninstrumented panes read unknown), await (poll or announce), and a grammar + discovery-recipe dictionary for the four built-ins plus the `codex mcp-server` bridge."
---
# Agent Primitives (`_cli-agents`)

**Domain**: runtime

## Overview

`_cli-agents` is the kit's reusable agent-CLI interaction helper (`src/kit/skills/_cli-agents.md`): the generic mechanics of driving *another* agent CLI — composing its invocation, opening it in a tmux pane, delivering a prompt reliably, reading its output and state, and waiting for it — plus a per-provider operational dictionary. It is opt-in via a skill's frontmatter `helpers:` list, so any skill or ad-hoc session can load it; `/fab-operator` is its first consumer. The split against [operator.md](/runtime/operator.md) is **agent primitives vs. operator orchestration**: this helper owns *how* to talk to an agent CLI, the operator owns *when and whether* to. The `fab agent` / `fab pane` surfaces it drives are documented in [providers-and-profiles.md](/runtime/providers-and-profiles.md) and [pane-commands.md](/runtime/pane-commands.md); the `@rk_agent_state` convention it reads is [runtime-agents.md](/runtime/runtime-agents.md).

**Pipeline consumer — the interactive-pane dispatch adapter.** `fab dispatch start <change> <stage> --pane` (see [dispatch.md](/runtime/dispatch.md)) is the pipeline-side consumer of these procedures: it composes the provider's `interactive_command` and opens it in a tmux **pane** per § Spawn Composition — including the "embed the initial prompt at spawn as one shell-escaped quoted argument" form (in either delivery shape — substituted at `{prompt}` or appended positionally), which is *why* the pane adapter delivers a one-line pointer to a prompt file rather than sending the prompt afterwards (embedding at spawn sidesteps the printed-prompt trap entirely: there is no pre-existing buffer to probe). It applies the same composition form to `tmux split-window` (its default shape — splitting the dispatching agent's own window) as to `tmux new-window` (its fallback shape); the § Spawn Composition rules are verb-independent. Its completion detection likewise follows § Await's preference for an **artifact** over a screen pattern — the dispatch contract's `{stage}-result.yaml`. A consumer steering that pane afterwards uses § Pre-send validation and § Peek. The helper carries the primitives; the adapter's own contract (state derivation, record shape, pane placement and identity) is dispatch's.

## Requirements

### Requirement: An opt-in helper carrying only generic mechanics

`_cli-agents` SHALL be an internal helper (`user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`) declared per-skill via `helpers:` — it is NOT part of the always-load layer (see [_shared/context-loading.md](/_shared/context-loading.md) § Skill Helper Declaration, where it is one of the eight allowed values). It SHALL carry **no orchestration policy**: repo targeting, worktree creation, change-pointer activation, monitored-set enrollment, dependency resolution, autopilot, confirmation tiers, and bounded-retry budgets are operator concerns and stay in `fab-operator.md` (with the fab-owned `wt`/tmux choreography in `_cli-external.md`). A consumer supplies its own policy around the primitives.

#### Scenario: a session needs to drive another agent CLI

- **GIVEN** a skill or ad-hoc session that must spawn, prompt, peek at, or await an agent CLI
- **WHEN** it reads `.claude/skills/_cli-agents/SKILL.md`
- **THEN** it finds the five procedures plus the provider dictionary, and no operator-orchestration content

### Requirement: Five agent-interaction procedures

The helper SHALL carry exactly five procedural sections, each stated once and referenced (never restated) by consumers:

1. **Spawn composition** — a session command is never hand-assembled. `fab agent --print` composes it profile-resolved, in either addressing form (role: `fab agent [role] --print [--repo <path>]`; provider: `fab agent --provider <name> --print [--model <id>] [--effort <level>]`), and the composed command is opened with `tmux new-window -n "<name>" -c "<dir>" "<composed-cmd> '<initial-prompt>'"`. The initial prompt is embedded at spawn as one shell-escaped quoted argument, and it carries **exactly one leading command**: an agent harness reads at most one leading `/command` and `&&` is not a shell operator in a prompt, so an `&&`-joined pair does not run two commands. The trailing-positional form above is one of **two** delivery shapes — the other substitutes the prompt at a `{prompt}` placeholder ([_shared/configuration.md](/_shared/configuration.md) § `providers`), which is what a value-taking initial-prompt flag requires.
2. **Pre-send validation** — a two-step gate before sending keys into an existing pane: the pane exists (refreshed pane map — a dead pane swallows keys silently, so a stale pane ID is a silent-failure hazard, not a visible error), and the agent is `idle` per the three-state `@rk_agent_state` read. `fab pane send` enforces the same gate (refusing `active`/`waiting`/unknown without `--force`), so consumers prefer it over raw `tmux send-keys` and let the binary hold the gate. Everything beyond the two mechanics — whether to ask the user, how many times to retry, whether the pane is on the right change or branch — is consumer policy.
3. **Delivery probe** — the recovery for the printed-prompt trap (below).
4. **Peek** — output and agent state read as two independent axes, with capture as the universal fallback (below).
5. **Await** — a poll loop or a worker-announced signal (below).

#### Scenario: the operator sends to a non-idle pane

- **GIVEN** a monitored pane whose agent state reads `waiting`
- **WHEN** the operator runs the pre-send gate
- **THEN** the gate reports not-idle and the operator applies its own policy — explicit user confirmation before sending

### Requirement: Spawn composition covers the empty-profile case

Omitting `--model`/`--effort` on the provider-addressed form SHALL leave the value empty and drop the placeholder's token plus a preceding `-`-flag, so a profile-free provider invocation results and the installed CLI's own default model applies. Provider-level flags without placeholders remain in the command. This is the documented way to spawn a provider whose current model IDs the caller does not know.

**`fab agent --provider` bypasses the provider's per-role fills too.** Its bypass of role resolution is total: `providers.<name>.profiles` is deliberately not consulted on this path, so an omitted flag means empty, not the configured fill. The fills apply on the resolution surface (`fab resolve-agent`, role resolution) — see [providers-and-profiles.md](/runtime/providers-and-profiles.md) § `fab agent`. Pass `--model` explicitly when a specific model is wanted from this launcher.

#### Scenario: spawning codex without knowing its model IDs

- **GIVEN** `providers.codex.interactive_command: 'codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'`
- **WHEN** `fab agent --provider codex --print` runs
- **THEN** it prints `codex --dangerously-bypass-approvals-and-sandbox`, and the session launches on the codex CLI's own default model under the provider's full-auto policy

### Requirement: A `{prompt}`-carrying provider is spawned by a fab launcher, or not at all

`fab agent --print` composes a **session** command, so it has already resolved `{prompt}` to empty and dropped the placeholder's flag pair. Appending a positional prompt to that output delivers nothing on a provider whose CLI takes its initial prompt through a **value-taking flag** — agy among the built-ins, whose `-i` discards a positional. For such a provider the spawn SHALL be composed by a **fab launcher** that delivers its own prompt through the shared two-shape rule while the placeholder is still standing: `fab dispatch start --pane` for a stage worker, `fab batch new`/`switch` for a change worker, `fab operator` for the operator session itself.

**Where no launcher covers the spawn, the spawn is UNSUPPORTED for such a provider.** The operator's per-change worker spawns are that case: they hand-assemble a `tmux new-window` around an arbitrary `/command` prompt, and no launcher composes it. The remedy is to point the target repo's `agent.session` at a placeholder-free provider (claude, codex) for those spawns, or to skip the spawn and tell the user — never to hand-assemble a substitute. See [operator.md](/runtime/operator.md) § Repo-targeted spawning.

#### Scenario: an operator spawn targets a repo whose session provider carries `{prompt}`

- **GIVEN** a target repo with `agent.session: agy` and an operator asked to spawn a change worker there
- **WHEN** it reaches the spawn step
- **THEN** it does not hand-assemble the window command from `fab agent --print` output — the printed command has no placeholder left to receive the prompt — and instead repoints the provider or reports the gap to the user

### Requirement: The printed-prompt trap has a probe-and-retype recovery

A `/command` visible at an agent's `❯` prompt MAY be *printed output* rather than a live input buffer, in which case Enter submits nothing — a silent no-op indistinguishable on screen from a pane about to run the command. The recovery SHALL be: send a literal harmless sentinel (e.g. `XYZTEST`) with no Enter → re-capture (sentinel appended ⇒ the buffer is live; absent ⇒ printed output) → clear the line with `C-u`, retype, send Enter → **confirm delivery** by re-capturing for a working indicator (spinner / thinking line / the agent's own turn start). "The send call succeeded" and "the agent received the command" are independent facts; only the post-send capture establishes the second.

#### Scenario: a send that appears to land but does nothing

- **GIVEN** a pane displaying a `/command` at its prompt with no spinner after Enter
- **WHEN** the consumer applies the delivery probe
- **THEN** it re-establishes a live buffer via `C-u` + retype and confirms delivery from a working indicator, instead of re-sending blind

### Requirement: Peek reads two independent axes; capture is the universal fallback

Peeking SHALL read output and agent state as separate axes: output via `fab pane capture <pane> [-l N] [--json]` (or `--raw` for a plain `tmux capture-pane -p`), state via the pane's `@rk_agent_state` option (surfaced by `fab pane map`'s Agent column and `fab pane capture --json`'s `agent_state`/`agent_idle_duration`). A wide capture window compensates for line wrapping when scanning for a prompt.

**State-writer caveat — uninstrumented panes read unknown.** fab is a pure *consumer* of `@rk_agent_state`; the writer is run-kit's `rk agent-setup` global agent-harness hooks, which cover **Claude Code, Codex, Copilot, Gemini, and OpenCode**. The unknown case is therefore an **uninstrumented pane** — no `rk agent-setup` in that environment, or a harness those hooks do not cover — not a non-claude one. Such a pane reports `—` (unknown), indistinguishable from "no state". Instrumentation is a property of the environment, not of the provider, so a claude pane is no exception. State reads are an optimization on instrumented panes, not a portable signal: **capture is the universal fallback**, and a flow MUST NOT be gated on a non-unknown state read.

#### Scenario: peeking at a pane in an environment without `rk agent-setup`

- **GIVEN** a pane running any instrumentable harness on a machine where `rk agent-setup` never ran
- **WHEN** a consumer reads its agent state
- **THEN** the state reads `—` (unknown) and the consumer falls back to capture rather than treating unknown as a failure

### Requirement: Await has no completion callback — poll or have the worker announce

There SHALL be no cross-provider equivalent of the Agent-tool "sub-agent finished, here is its result" callback when driving a CLI in a pane. The two available signals are (1) **poll** — loop capture (plus the state read, where instrumented) on a fixed unhurried cadence until a completion signal the consumer *defined* appears, and (2) **worker announcement** — instruct the worker in its prompt to run a notification as its last act (e.g. `rk notify`, `command -v rk`-gated and fail-silent), converting polling into an event at the cost of depending on the worker honoring the instruction. Asking for an **artifact** (a file at a named path) is preferred over a screen pattern: a file's presence is unambiguous, survives scrollback, and is readable without the pane.

### Requirement: The provider dictionary carries stable grammar and discovery recipes, never catalogs

Half B SHALL carry one entry per provider (claude, codex, agy, kimi), each recording **stable invocation grammar** — interactive vs. headless entry point, whether the prompt arrives by stdin or argument, the structured-output flag, profile-flag grammar, session/resume semantics where they exist — and a **model-discovery recipe**: what to run against the *installed* binary to learn which models it accepts. It MUST NOT bake model-ID catalogs, which rot in weeks. Interactive quirks SHALL be recorded only once actually hit and confirmed; speculating about an uninstalled CLI reads as verified and is worse than silence.

Every dictionary provider is a **built-in provider** resolvable by name — no `providers:` block and no uncommenting is needed to spawn one (`fab agent --provider codex`, or a depth knob naming it). The binary also ships **per-role fills** for claude, codex and agy (kimi deliberately carries none — its `-m` takes a user-config alias, so the empty model drops the flag and the CLI's own `default_model` applies), so a *fill-consuming* path (a depth knob, `agent.profiles.<role>.provider`, `fab resolve-agent --provider`) resolves a real model for every role; `fab agent --provider` is **not** one — it bypasses role resolution *and* the provider's fills, so it composes a profile-free invocation unless `--model`/`--effort` are passed (see [providers-and-profiles.md](/runtime/providers-and-profiles.md) § `fab agent`). Those fills are refreshed at kit-release cadence and unvalidated, so the dictionary still carries a recipe rather than a catalog: verify against the installed binary, then pin what you find with `providers.<name>.profiles.<role>.model` (the modern spelling — a flat `providers.<name>.model` is an alias for `profiles.default` and is outranked per field wherever the provider ships a role fill) or pass `--model` per invocation. The dictionary's command strings stay consistent with that built-in grammar; being markdown, it updates with the kit and carries no behavior risk.

The grammar facts each entry pins today: claude is headless via the `-p` **flag** reading stdin (`--output-format stream-json` for structured output, `-n <name>` for session naming); codex is headless via the `exec` **subcommand** reading stdin (`--json`, `--dangerously-bypass-approvals-and-sandbox`, `-m <id>`, effort via `-c model_reasoning_effort=<level>`); agy and kimi are both headless via a `-p` **flag whose argument IS the prompt** — neither reads stdin — so both shipped templates nest a shell (`sh -c '… -p "$(cat)"'`). agy adds `--dangerously-skip-permissions`, `--print-timeout` (raised from its 5m default), and **no effort placeholder** — its model IDs embed the reasoning level, and fab never appends a flag the CLI lacks. Interactively, agy takes its initial prompt through the **value-taking** `-i`/`--prompt-interactive` flag (a bare positional is discarded, a bare `-i` exits 2), which is why fab ships it pane-capable through the `{prompt}` placeholder — its entry carries the once-per-worktree trust-dialog caveat with that grammar. kimi's `-p` already auto-approves tools and *rejects* `--yolo`/`--auto`, its `-m` names a **user-config alias** rather than a catalog ID, and its entry carries the pane caveat behind fab shipping it **dispatch-only** (no `interactive_command`, and no interactive-initial-prompt flag for one to target).

#### Scenario: learning which models the installed codex accepts

- **GIVEN** a session that needs a valid `--model` value for codex
- **WHEN** it reads the codex dictionary entry
- **THEN** it finds a discovery recipe to run against the installed binary (`codex --version`, then `codex --help` / `codex exec --help` for `-m`), not a hardcoded list

### Requirement: The codex MCP bridge is `codex mcp-server`

The dictionary SHALL carry the codex MCP-bridge recipe as session configuration only — fab ships nothing for it. Registering codex as a stdio MCP server inside a claude session turns cross-provider work into ordinary tool use with full context retained on both sides, instead of piping temp files and reading terminal scrollback. The server subcommand is **`codex mcp-server`** (starts codex *as* a stdio MCP server); **`codex mcp` is a different subcommand** that *manages* the external MCP servers codex itself connects to (list/add/remove/login). The recipe capability-probes the installed binary (`codex --version`, then confirm `codex mcp-server` in `codex --help`) before relying on it. Bridging is preferred when the goal is a *conversation* (asking codex about a diff, iterating on its answer); pane-driving is preferred for a *long-running autonomous task* to watch and steer.

## Design Decisions

### Agent Primitives Extracted into an Opt-In `_cli-agents` Helper
**Decision**: The generic spawn / pre-send-validation / delivery-probe / peek / await mechanics live in `_cli-agents.md`, declared opt-in via `helpers:` (making the allowed set eight), and `fab-operator.md` references those sections instead of restating them — keeping its own confirmation tiers, retry budgets, change-active/branch-alignment checks, repo targeting, pointer activation, enrollment, dependency resolution, and autopilot verbatim.
**Why**: How to spawn an agent CLI, deliver a prompt into its TUI, peek at it, and wait for it is generic knowledge with many potential consumers, so it belongs somewhere loadable rather than inside a 700-line orchestrator only that orchestrator can read — otherwise every cross-provider moment reinvents it, and the printed-prompt/send-keys trap in particular is a recurring, expensive failure to rediscover. A helper is the cheapest home: markdown only, no new binary, no behavior change to any existing spawn or dispatch path, and it gives an interactive stage-dispatch adapter a primitive layer to build on rather than forcing a second extraction.
**Rejected**: Adding the mechanics to the always-load layer (every skill would pay for content few need — `_cli-*` helpers are opt-in and always-load stays lean); copying the procedures into a second file (the duplicate-truth failure the sweep rules exist to prevent); a standalone agent-orchestration tool (deferred — the one-helper-plus-markdown-dictionary shape keeps a later move cheap).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

### The Placeholder Gap Is Named, Not Papered Over With a New `--print` Mode
**Decision**: A hand-assembled spawn (the operator's per-change worker spawns) is documented **UNSUPPORTED** for a provider whose `interactive_command` carries `{prompt}`, with a repoint-or-skip remedy. `fab agent --print` gains no mode that emits the command with the placeholder left undropped.
**Why**: `--print` composes a *session* command, and a session command with a live `{prompt}` in it is not runnable — printing one would hand every caller a string that only some callers know how to finish, re-creating the two-implementations-of-one-grammar hazard the single delivery helper removes. Naming the gap keeps the failure loud and one config line from fixed, and the gap is closing from the other side anyway: the pane-delivery redesign removes spawn-time prompt embedding entirely, so a new print mode would be surface built for a window that is already shutting.
**Rejected**: A `--print --with-prompt-placeholder` mode (an unrunnable command on the launcher's only output surface); teaching the operator to substitute the placeholder itself (a second implementation of the delivery grammar, in prose, in the one consumer least able to test it); silently appending the positional anyway (the prompt vanishes and the worker sits at an empty TUI — the silent per-spawn failure the pane-eligibility rules exist to prevent).
*Introduced by*: 260809-agik-agy-interactive-command

### The Provider Dictionary Ships Discovery Recipes, Not Model Catalogs
**Decision**: Each provider entry records stable invocation grammar plus a recipe for discovering models on the *installed* CLI, and records interactive quirks only once confirmed by a real encounter. **No model-ID catalog appears anywhere in the dictionary** — the split is dictionary-vs-binary, not grammar-vs-fill: `internal/agent`'s built-in table does ship per-role fills for most providers, but those are **data** refreshed at kit-release cadence and overridden by one config line, not knowledge an agent should carry in prose. Each entry names that fills exist for its provider and points at the recipe as the way to verify one.
**Why**: A model catalog written into a markdown dictionary is read by an agent that may be reasoning about an *uninstalled* CLI, so it rots into confident misinformation where a recipe stays correct; this matches fab's document-don't-validate posture, which never infers a provider from a model string. Shipped fills are a different artifact with a different failure shape — one line of embedded YAML, refreshed on every kit release, rejected loudly by the CLI when stale — so they can carry values the dictionary must not. Keeping the dictionary in markdown means it updates with the kit at zero behavior risk, and asserting unverified behavior about an uninstalled CLI is worse than silence because it reads as verified. A built-in provider is inert until named, so the dictionary can point at `--provider codex` as a working invocation rather than at a config-editing prerequisite.
**Rejected**: Baked model-ID catalogs in the dictionary (rot into misinformation an agent will act on); enumerating the binary's shipped fills here (a second copy of a drift-guarded table, in the one place with no guard); speculative quirk entries for CLIs never actually driven (unverifiable claims presented as fact); describing a config-editing prerequisite the resolvable provider set does not impose (the dictionary's whole value is that its recipes work as written).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn; *Updated by*: 260805-j3cm-builtin-provider-templates-and-fill, 260806-ywkx-ship-codex-gemini-fills (the binary ships fills; the dictionary still ships recipes only)

### The MCP Bridge Rides in the Dictionary as Recipe Text
**Decision**: Registering codex as an MCP server inside a claude session is documented as a recipe subsection in the dictionary — capability-probe, register `codex mcp-server`, converse through the resulting tools — with no fab machinery behind it.
**Why**: It is session configuration, not fab behavior: nothing in the kit needs to run, resolve, or validate it, so a recipe is the whole deliverable. Naming the correct subcommand explicitly is load-bearing because `codex mcp` and `codex mcp-server` are adjacent and only one starts codex as a stdio server.
**Rejected**: A fab subcommand or config surface for the bridge (machinery for something the harness's own MCP config already expresses); its own change phase (nothing to build).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn
