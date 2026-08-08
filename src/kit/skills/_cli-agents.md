---
name: _cli-agents
description: "Agent-CLI interaction reference — the generic spawn / pre-send-validation / peek / await procedures for driving another agent CLI in a tmux pane (extracted from fab-operator), plus a three-provider operational dictionary (claude, codex, gemini) carrying stable invocation grammar and model-discovery recipes rather than volatile model catalogs. Opt-in via `helpers:`; not part of the always-load layer."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Agent CLI Interaction Reference

> Loaded via a skill's `helpers: [_cli-agents]` frontmatter (or an in-body point-of-use read) — **not** part of the always-load layer. Any skill or ad-hoc session that needs to spawn another agent CLI, deliver a prompt into its TUI, peek at its output, or wait for it can load this file.

## Contents

- Scope Boundary
- Half A — Agent-Interaction Procedures
  - Spawn Composition
  - Pre-Send Validation
  - Delivery Probe (the printed-prompt trap)
  - Peek
  - Await
- Half B — Provider Dictionary
  - Dictionary Discipline
  - claude
  - codex
  - gemini
  - Codex MCP Bridge (recipe)

---

## Scope Boundary

This file carries **only the generic mechanics** of interacting with an agent CLI: composing its invocation, opening it in a pane, delivering a prompt reliably, reading its output and state, and waiting for it.

It deliberately carries **no orchestration policy**. Repo targeting, worktree creation, change-pointer activation, monitored-set enrollment, dependency resolution, autopilot, confirmation tiers, and bounded-retry budgets are **operator** concerns and live in `fab-operator.md` (with the fab-owned `wt`/tmux choreography in `_cli-external.md`). A consumer of this file supplies its own policy around these primitives.

The `fab` commands referenced here (`fab agent`, `fab pane capture`, `fab pane send`, `fab pane map`) are documented in `_cli-fab.md` — load it too when you need their exhaustive flag surface.

---

## Half A — Agent-Interaction Procedures

### Spawn Composition

An interactive agent session command is **never hand-assembled**. Ask `fab agent` to compose it, then open the composed command in a pane.

Two addressing forms — both print a fully **profile-resolved** command (`{model}`/`{effort}` substituted, or Claude-style flags appended for a non-templated `session_command`):

```sh
# Role-addressed — the resolved role supplies {provider, model, effort}.
fab agent --print                       # the default role, this repo
fab agent operator --print              # a named role
fab agent --print --repo <target-repo>  # another repo's config (never your own)

# Provider-addressed — bypasses role resolution entirely.
fab agent --provider codex --print                                  # bare invocation: the CLI's own default model
fab agent --provider codex --model <id> --effort <level> --print     # explicit profile
```

**Which form to use.** Use the **role** form when the spawn should inherit fab's role/budget policy (a pipeline-shaped worker, the operator's own coordinator) — including which provider the `agent.session` knob points Tier-1 agents at. Use the **provider** form when the question is mechanical — "give me a codex session right here" — with no role to speak of. `--provider` is mutually exclusive with the `[role]` positional, and `--model`/`--effort` are only valid alongside `--provider` (see `_cli-fab.md` § fab agent — note `fab resolve-agent` deliberately allows them bare, being a pure query). **No `providers:` block is needed for either form**: `claude`, `codex`, and `gemini` are built-in providers.

**Empty model/effort is a feature.** Omitting `--model`/`--effort` on the provider form leaves the value empty, and the composition rule drops the placeholder's token *and* a preceding `-`-flag while retaining fixed flags — so `fab agent --provider codex --print` against `codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` yields `codex --dangerously-bypass-approvals-and-sandbox`. The installed CLI's own default model applies while the built-in's deliberate full-auto posture remains. This is how you spawn a provider whose current model IDs you do not know.

**Provider-form caveat:** `fab agent --provider <name>` bypasses role resolution and both per-role fill sources, so its profile is exactly the passed flags; pass `--model` explicitly or use the role form when fills should apply. See `_cli-fab.md` § fab agent for the fill ladder and model-free Codex example.

**Open it in a pane:**

```sh
tmux new-window -n "<name>" -c "<dir>" "<composed-cmd> '<initial-prompt>'"
```

- The composed command is the *whole* left-hand side — including any shell expansions it carries (e.g. `$(basename "$(pwd)")`), which expand at invocation inside the new window.
- The initial prompt is embedded **at spawn** as a single quoted argument. Shell-escape any user-supplied text before embedding it.
- **One prompt, one leading command.** The embedded string is delivered as a single prompt to the agent, where `&&` is *not* a shell operator and an agent harness reads at most one leading `/command` — so an `&&`-joined pair of slash commands does not run two commands; the tail is swallowed into the first command's argument. Embed exactly one command; if a second step is genuinely required, run it as a synchronous CLI call before opening the window, or as a separate Enter-terminated send afterwards.
- **Embedding at spawn also sidesteps the printed-prompt trap** (§ Delivery Probe): the trap needs a *pre-existing* input buffer to mistake printed output for, and a window created with its prompt already attached has none. Prefer spawn-embedding over "open the window, then send the prompt" whenever the prompt is known up front.

> **Pipeline consumer**: `fab dispatch start <change> <stage> --pane` (the interactive-pane dispatch adapter — `_cli-fab.md` § fab dispatch, contract in `docs/specs/harness-adapters.md`) is built on exactly this procedure. It composes the resolved provider's `session_command`, opens it with the form above, and — because a multi-thousand-token stage prompt cannot ride argv or send-keys — embeds a **one-line pointer** to a prompt file as the single quoted argument (shell-escaped per the rule above — the pointer names a repo-derived path), applying § Await's "prefer asking for an artifact over a screen pattern" rule to completion detection too (the worker's `{stage}-result.yaml`).

### Pre-Send Validation

Before sending keys into an *existing* agent pane:

1. **Verify the pane exists** — refresh the pane map (`fab pane map [--all-sessions] [--json]`). A dead pane accepts keys silently into nothing, so a stale pane ID is a silent-failure hazard, not an error you will see.
2. **Verify the agent is idle** — read the pane's agent state. The state is the three-state `@rk_agent_state` convention plus unknown: `idle` is the only state safe to send to unattended; `active` (turn in progress) and `waiting` (blocked on a human) both risk corrupting work or cutting across a pending answer; `—`/unknown means the pane is uninstrumented. `fab pane send` enforces this same gate — it refuses `active`/`waiting`/unknown without `--force` — so prefer `fab pane send` over raw `tmux send-keys` and let the binary hold the gate. The convention's exact semantics (value format, the mandatory epoch suffix, what counts as unknown, the no-staleness-heuristic rule) are in `_cli-fab.md` § fab pane → § agent state.

Anything beyond these two mechanics (whether to ask the user, how many times to retry, whether the target pane is on the right change or branch) is the consumer's policy.

### Delivery Probe (the printed-prompt trap)

**The trap.** A `/command` visible at an agent's `❯` prompt may be *printed output*, not a live input buffer. Pressing Enter then submits nothing — a **silent no-op**. The pane looks exactly like a pane that is about to run your command.

**The probe-and-retype recovery:**

1. Send a literal, harmless sentinel (e.g. `XYZTEST`) with **no** Enter.
2. Re-capture the pane. If the sentinel appears appended to the visible prompt text, the buffer is live. If it does not appear, what you are looking at is printed output.
3. Either way, clear the line with `C-u`, retype the intended command, then send Enter.
4. **Confirm delivery** — re-capture and look for a working indicator (a spinner / "thinking" line / the agent's own turn start). Absence of a working indicator after a send means the command did not land; do not assume success from the send call's exit status.

Treat "the send call succeeded" and "the agent received the command" as independent facts. Only the post-send capture establishes the second.

### Peek

Two independent axes, read separately:

- **Output** — `fab pane capture <pane> [-l N] [--json]` for enriched capture, `--raw` for a plain `tmux capture-pane -p`. A wide window (`-l 50`+, or `capture-pane -S -20` for the last 20 lines) compensates for line wrapping when scanning for a prompt.
- **Agent state** — the pane's `@rk_agent_state` option (surfaced by `fab pane map`'s Agent column and by `fab pane capture --json`'s `agent_state`/`agent_idle_duration` fields).

> **State-writer caveat — uninstrumented panes read unknown.** fab is a pure *consumer* of `@rk_agent_state`; the writer is run-kit's `rk agent-setup` global agent-harness hooks, which cover **Claude Code, Codex, Copilot, Gemini, and OpenCode** — not just Claude (see `_cli-fab.md` § fab pane → § agent state). The unknown case is therefore an **uninstrumented pane**, not a non-claude one: no `rk agent-setup` was ever run in that environment, or the pane runs a harness those hooks do not cover. Such a pane reports `—` (unknown) — indistinguishable from "no state". So **capture is the universal fallback**: state reads are an optimization on instrumented panes, not a portable signal. Never gate a flow on a non-unknown state read.

### Await

There is **no cross-provider completion notification.** The Agent-tool style "sub-agent finished, here is its result" callback has no equivalent when you drive a CLI in a pane. The available signals are:

1. **Poll** — loop `capture` (+ the state read, where instrumented) on an interval until a completion signal appears. Use a fixed, unhurried cadence; a completion signal is a *pattern you defined*, e.g. the agent's own summary line, a sentinel string you asked for in the prompt, or a file the worker was told to write.
2. **Have the worker announce itself** — instruct the worker in its prompt to run a notification as its last act (e.g. `rk notify` — `command -v rk`-gated and fail-silent per `_preamble.md` § Run-Kit (rk) Reference). This converts polling into an event, at the cost of depending on the worker honoring the instruction.

Prefer asking for an **artifact** (a file at a path you name) over a screen pattern: a file's presence is unambiguous, survives scrollback, and is readable without the pane. A screen pattern is the fallback when you cannot dictate the worker's output contract.

---

## Half B — Provider Dictionary

### Dictionary Discipline

Each entry below carries only **stable invocation grammar** and **discovery recipes**:

- **Grammar is stable** — whether a CLI's headless mode is a flag or a subcommand, whether it reads the prompt from stdin or an argument, what its structured-output flag is called. These change on major-version boundaries, not weekly.
- **Model IDs are NOT recorded *here*.** Model catalogs rot in weeks, and this dictionary is read by an agent that may be reasoning about an uninstalled CLI. Instead each entry carries a *discovery recipe*: what to run against the **installed** binary to learn which models it accepts. Never assume a model ID from memory; run the recipe. *(fab-kit does ship per-role model fills in the binary — `providers.<name>.profiles` — but those are DATA refreshed at kit-release cadence and overridden by one config line, not knowledge an agent should carry. The recipe is still how you verify one.)*
- **Quirks accrete from real encounters only.** An entry records an interactive quirk (first-run trust prompt, submit-key behavior) only once it has actually been hit and confirmed. Speculating about an uninstalled CLI's behavior is worse than silence — it reads as verified.

- **Built-ins:** `claude`, `codex`, and `gemini` use the independent `session_command` / `dispatch_command` / `native` capability grammar in `internal/agent`'s embedded `defaults.yaml`; all resolve without a `providers:` block. Claude ships all three capabilities, while codex/gemini are non-native; `dispatch.mode` chooses the starting rung of the descending `pane → native → headless` ladder.
- **Fill-consuming paths:** depth knobs (`agent.session` / `agent.workers`), `agent.profiles.<role>.provider`, and `fab resolve-agent --provider` consume the built-in per-role fills and resolve a real model for every role.
- **Provider-addressed sessions:** `fab agent --provider` bypasses both fill sources and stays bare unless `--model`/`--effort` is passed.
- **Freshness and overrides:** non-Claude fills are release-cadence, unvalidated data. Discover current IDs with each entry's recipe; override with `providers.<name>.profiles.<role>.{model,effort}` (including in `~/.fab-kit/config.yaml`) or invocation flags. A `providers:` block overrides capabilities/fills; it does not register built-ins.

### claude

| Aspect | Value |
|--------|-------|
| Interactive | `claude` |
| Headless | `claude -p` — reads the prompt from **stdin** |
| Native capability | `native: true` — the Claude Agent-tool seam; independent of both command fields |
| Structured output | `claude -p --output-format stream-json` |
| Profile flags | `--model <id> --effort <level>` (the CLI accepts full IDs *and* short aliases) |
| Session naming | `-n <name>` names the session (the built-in `session_command` uses `-n "$(basename "$(pwd)")"`) |
| Model discovery | The installed CLI's own help is authoritative: `claude --help` for the `--model` flag's accepted forms. fab's own alias mapping (`opus`/`sonnet`/`haiku`/`fable`) is an Agent-tool adapter, not a claude-CLI constraint — see `_cli-fab.md` § fab resolve-agent |

**Agent-state instrumentation.** Claude Code is one of the harnesses `rk agent-setup` instruments (alongside Codex, Copilot, Gemini, and OpenCode); instrumentation is an environment property, not a provider guarantee.

### codex

| Aspect | Value |
|--------|-------|
| Interactive | `codex` (TUI) |
| Headless | `codex exec` — a **subcommand**, not a flag; reads the prompt from **stdin** |
| Structured output | `codex exec --json` |
| Profile flags | `-m <id>` for the model; reasoning effort rides a config override: `-c model_reasoning_effort=<level>` |
| MCP server mode | `codex mcp-server` — starts codex AS a stdio MCP server (distinct from `codex mcp`, which *manages* the external MCP servers codex itself connects to); see § Codex MCP Bridge below |
| Model discovery | Capability-probe the installed binary: `codex --version`, then `codex --help` / `codex exec --help` for the `-m` flag's accepted values. fab ships per-role codex fills (`default`/`doing`/`review`/`fast`) and validates none of them, so the recipe is how you verify one. Pin a discovered ID as `providers.codex.profiles.<role>.model` — the modern spelling — or pass `--model` per invocation. The flat `providers.codex.model` is an **alias for `profiles.default`** and is outranked by the shipped role fills, so it does not reach `doing`/`review`/`fast` |

**Why independent provider capabilities matter here.** `codex` (TUI) and `codex exec` (headless) are different invocations of the same binary, which is exactly why `providers.<name>` carries unmerged `session_command` and `dispatch_command` fields; the separate `native` boolean records an Agent-tool seam when one exists. These fields say how a rung runs, never which dispatch mode to prefer.

### gemini

| Aspect | Value |
|--------|-------|
| Interactive | `gemini` |
| Headless | `gemini` with the prompt on **stdin** — in a non-TTY context it reads stdin as the prompt |
| `-p` caveat | `gemini -p` takes prompt **TEXT** (appended after stdin), so it is **not** the headless flag you want when piping — the built-in `gemini` `dispatch_command` deliberately omits `-p` |
| Profile flags | `-m <id>` for the model. **No effort flag** — the gemini CLI has no reasoning-effort knob, so the built-in grammar omits `{effort}` entirely |
| Model discovery | `gemini --help` on the installed binary for the `-m` flag's accepted values. fab ships per-role gemini fills (`default`/`fast`) as the CLI's own stable aliases `pro`/`flash`, and validates neither, so the recipe is how you verify one. Pin a discovered ID as `providers.gemini.profiles.<role>.model` — the modern spelling — or pass `--model` per invocation. The flat `providers.gemini.model` is an **alias for `profiles.default`** and is outranked by the shipped `fast` role fill |

**Template consequence.** Because the gemini grammar carries no `{effort}` placeholder, the substitution is all-or-nothing per placeholder present: a resolved effort simply has nowhere to go and is not injected. That is the intended behavior — provider grammar is the provider's, and fab never appends a flag the CLI does not have.

### Codex MCP Bridge (recipe)

For tool-mediated, multi-turn cross-provider conversation (rather than one-shot output scraping), register the codex CLI as an **MCP server** inside a claude session. This is **session configuration, not fab machinery** — fab ships nothing for it.

Recipe:

1. **Capability-probe the installed codex** for MCP-server support before relying on it: `codex --version`, then confirm `codex mcp-server` is present in `codex --help`. An older binary may not expose it. Do **not** reach for `codex mcp` — that subcommand *manages the external MCP servers codex connects to* (list/add/remove/login); `codex mcp-server` is the one that starts codex itself as a stdio MCP server.
2. **Register `codex mcp-server` as an MCP server** in the claude session's MCP configuration (the harness's own MCP server config — a stdio server whose command is `codex mcp-server`).
3. **Converse through the resulting tools** — the codex-side tools appear in the claude session's tool list, so a multi-turn exchange becomes ordinary tool use with full context retained on both sides, instead of piping temp files and reading back terminal scrollback.

Prefer this over pane-driving whenever the goal is a *conversation* (asking codex about a diff, iterating on its answer). Prefer pane-driving (Half A) when the goal is a *long-running autonomous task* you want to watch and steer.
