---
type: memory
description: "The `_cli-agents` helper — agent-CLI interaction primitives any skill opts into via `helpers:`: spawn composition from `fab agent --print` (tier- or provider-addressed), the pane-exists/agent-idle pre-send gate, the printed-prompt probe, peek (capture + `@rk_agent_state`; uninstrumented panes read unknown, so capture is the fallback), await (no completion callback — poll or announce), and a claude/codex/gemini dictionary of stable grammar + discovery recipes plus the `codex mcp-server` bridge."
---
# Agent Primitives (`_cli-agents`)

**Domain**: runtime

## Overview

`_cli-agents` is the kit's reusable agent-CLI interaction helper (`src/kit/skills/_cli-agents.md`): the generic mechanics of driving *another* agent CLI — composing its invocation, opening it in a tmux pane, delivering a prompt reliably, reading its output and state, and waiting for it — plus a per-provider operational dictionary. It is opt-in via a skill's frontmatter `helpers:` list, so any skill or ad-hoc session can load it; `/fab-operator` is its first consumer. The split against [operator.md](/runtime/operator.md) is **agent primitives vs. operator orchestration**: this helper owns *how* to talk to an agent CLI, the operator owns *when and whether* to. The `fab agent` / `fab pane` surfaces it drives are documented in [providers-and-tiers.md](/runtime/providers-and-tiers.md) and [pane-commands.md](/runtime/pane-commands.md); the `@rk_agent_state` convention it reads is [runtime-agents.md](/runtime/runtime-agents.md).

## Requirements

### Requirement: An opt-in helper carrying only generic mechanics

`_cli-agents` SHALL be an internal helper (`user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`) declared per-skill via `helpers:` — it is NOT part of the always-load layer (see [_shared/context-loading.md](/_shared/context-loading.md) § Skill Helper Declaration, where it is one of the eight allowed values). It SHALL carry **no orchestration policy**: repo targeting, worktree creation, change-pointer activation, monitored-set enrollment, dependency resolution, autopilot, confirmation tiers, and bounded-retry budgets are operator concerns and stay in `fab-operator.md` (with the fab-owned `wt`/tmux choreography in `_cli-external.md`). A consumer supplies its own policy around the primitives.

#### Scenario: a session needs to drive another agent CLI

- **GIVEN** a skill or ad-hoc session that must spawn, prompt, peek at, or await an agent CLI
- **WHEN** it reads `.claude/skills/_cli-agents/SKILL.md`
- **THEN** it finds the four procedures plus the provider dictionary, and no operator-orchestration content

### Requirement: Four agent-interaction procedures

The helper SHALL carry exactly four procedural sections, each stated once and referenced (never restated) by consumers:

1. **Spawn composition** — a session command is never hand-assembled. `fab agent --print` composes it profile-resolved, in either addressing form (tier: `fab agent [tier] --print [--repo <path>]`; provider: `fab agent --provider <name> --print [--model <id>] [--effort <level>]`), and the composed command is opened with `tmux new-window -n "<name>" -c "<dir>" "<composed-cmd> '<initial-prompt>'"`. The initial prompt is embedded at spawn as one shell-escaped quoted argument, and it carries **exactly one leading command**: an agent harness reads at most one leading `/command` and `&&` is not a shell operator in a prompt, so an `&&`-joined pair does not run two commands.
2. **Pre-send validation** — a two-step gate before sending keys into an existing pane: the pane exists (refreshed pane map — a dead pane swallows keys silently, so a stale pane ID is a silent-failure hazard, not a visible error), and the agent is `idle` per the three-state `@rk_agent_state` read. `fab pane send` enforces the same gate (refusing `active`/`waiting`/unknown without `--force`), so consumers prefer it over raw `tmux send-keys` and let the binary hold the gate. Everything beyond the two mechanics — whether to ask the user, how many times to retry, whether the pane is on the right change or branch — is consumer policy.
3. **Delivery probe** — the recovery for the printed-prompt trap (below).
4. **Await** — a poll loop or a worker-announced signal (below).

#### Scenario: the operator sends to a non-idle pane

- **GIVEN** a monitored pane whose agent state reads `waiting`
- **WHEN** the operator runs the pre-send gate
- **THEN** the gate reports not-idle and the operator applies its own policy — explicit user confirmation before sending

### Requirement: Spawn composition covers the empty-profile case

Omitting `--model`/`--effort` on the provider-addressed form SHALL leave the value empty and drop the placeholder's token plus a preceding `-`-flag, so a bare provider invocation results and the installed CLI's own default model applies. This is the documented way to spawn a provider whose current model IDs the caller does not know.

#### Scenario: spawning codex without knowing its model IDs

- **GIVEN** `providers.codex.session_command: 'codex -m {model} -c model_reasoning_effort={effort}'`
- **WHEN** `fab agent --provider codex --print` runs
- **THEN** it prints a bare `codex`, and the session launches on the codex CLI's own default model

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

Half B SHALL carry one entry per provider (claude, codex, gemini), each recording **stable invocation grammar** — interactive vs. headless entry point, whether the prompt arrives by stdin or argument, the structured-output flag, profile-flag grammar, session/resume semantics where they exist — and a **model-discovery recipe**: what to run against the *installed* binary to learn which models it accepts. It MUST NOT bake model-ID catalogs, which rot in weeks. Interactive quirks SHALL be recorded only once actually hit and confirmed; speculating about an uninstalled CLI reads as verified and is worse than silence. The command strings stay consistent with the shipped `providers:` starter template, whose codex/gemini blocks are commented until a user opts in — the dictionary is markdown, so it updates with the kit and carries no behavior risk, while the Go built-in provider table stays claude-only.

The grammar facts each entry pins today: claude is headless via the `-p` **flag** reading stdin (`--output-format stream-json` for structured output, `-n <name>` for session naming); codex is headless via the `exec` **subcommand** reading stdin (`--json`, `-m <id>`, effort via `-c model_reasoning_effort=<level>`); gemini reads stdin as the prompt in a non-TTY context and its `-p` takes prompt **TEXT** appended after stdin (so the shipped template omits `-p`), and it has **no reasoning-effort flag** at all (so the template omits `{effort}`, and a resolved effort simply has nowhere to go — fab never appends a flag the CLI lacks).

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

### The Provider Dictionary Ships Discovery Recipes, Not Model Catalogs
**Decision**: Each provider entry records stable invocation grammar plus a recipe for discovering models on the *installed* CLI, and records interactive quirks only once confirmed by a real encounter. No model-ID catalog appears anywhere in it, and no Go built-in provider is added — codex and gemini stay uncomment-to-opt-in config template text plus markdown dictionary entries.
**Why**: Model catalogs rot within weeks while grammar changes on major-version boundaries, so a recipe stays correct where a list silently misleads; this matches fab's document-don't-validate posture, which never infers a provider from a model string. Keeping the dictionary in markdown means it updates with the kit at zero behavior risk, and asserting unverified behavior about an uninstalled CLI is worse than silence because it reads as verified.
**Rejected**: Baked model IDs per provider (rot monthly); speculative quirk entries for CLIs never actually driven (unverifiable claims presented as fact); promoting codex/gemini to Go built-ins (changes default behavior for a markdown-only need).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

### The MCP Bridge Rides in the Dictionary as Recipe Text
**Decision**: Registering codex as an MCP server inside a claude session is documented as a recipe subsection in the dictionary — capability-probe, register `codex mcp-server`, converse through the resulting tools — with no fab machinery behind it.
**Why**: It is session configuration, not fab behavior: nothing in the kit needs to run, resolve, or validate it, so a recipe is the whole deliverable. Naming the correct subcommand explicitly is load-bearing because `codex mcp` and `codex mcp-server` are adjacent and only one starts codex as a stdio server.
**Rejected**: A fab subcommand or config surface for the bridge (machinery for something the harness's own MCP config already expresses); its own change phase (nothing to build).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn
