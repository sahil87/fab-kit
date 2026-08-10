---
name: _cli-agents
description: "Agent-CLI interaction reference — the generic spawn / pre-send-validation / peek / await procedures for driving another agent CLI in a tmux pane (extracted from fab-operator), plus a four-provider operational dictionary (claude, codex, agy, kimi) carrying stable invocation grammar and model-discovery recipes rather than volatile model catalogs. Opt-in via `helpers:`; not part of the always-load layer."
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
  - agy
  - kimi
  - Codex MCP Bridge (recipe)

---

## Scope Boundary

This file carries **only the generic mechanics** of interacting with an agent CLI: composing its invocation, opening it in a pane, delivering a prompt reliably, reading its output and state, and waiting for it.

It deliberately carries **no orchestration policy**. Repo targeting, worktree creation, change-pointer activation, monitored-set enrollment, dependency resolution, autopilot, confirmation tiers, and bounded-retry budgets are **operator** concerns and live in `fab-operator.md` (with the fab-owned `wt`/tmux choreography in `_cli-external.md`). A consumer of this file supplies its own policy around these primitives.

The `fab` commands referenced here (`fab agent`, `fab pane open`/`ready`/`deliver`, `fab pane capture`, `fab pane send`, `fab pane map`) are documented in `_cli-fab.md` — load it too when you need their exhaustive flag surface.

---

## Half A — Agent-Interaction Procedures

### Spawn Composition

An interactive agent session command is **never hand-assembled**. Ask `fab agent` to compose it, then open the composed command in a pane.

Two addressing forms — both print a fully **profile-resolved** command (`{model}`/`{effort}` substituted, or Claude-style flags appended for a non-templated `interactive_command`):

```sh
# Role-addressed — the resolved role supplies {provider, model, effort}.
fab agent --print                       # the default role, this repo
fab agent operator --print              # a named role
fab agent --print --repo <target-repo>  # another repo's config (never your own)

# Provider-addressed — bypasses role resolution entirely.
fab agent --provider codex --print                                  # bare invocation: the CLI's own default model
fab agent --provider codex --model <id> --effort <level> --print     # explicit profile
```

> **Not every provider has a session form.** Composition needs an `interactive_command`, and all four built-ins ship one. Either form errors actionably against a provider with none — a user-defined, dispatch-only `providers:` entry — naming the `providers.<name>.interactive_command` key to set.

**Which form to use.** Use the **role** form when the spawn should inherit fab's role/budget policy (a pipeline-shaped worker, the operator's own coordinator) — including which provider the `agent.session` knob points Tier-1 agents at. Use the **provider** form when the question is mechanical — "give me a codex session right here" — with no role to speak of. `--provider` is mutually exclusive with the `[role]` positional, and `--model`/`--effort` are only valid alongside `--provider` (see `_cli-fab.md` § fab agent — note `fab resolve-agent` deliberately allows them bare, being a pure query). **No `providers:` block is needed for either form**: `claude`, `codex`, `agy`, and `kimi` are built-in providers.

**Empty model/effort is a feature.** Omitting `--model`/`--effort` on the provider form leaves the value empty, and the composition rule drops the placeholder's token *and* a preceding `-`-flag while retaining fixed flags — so `fab agent --provider codex --print` against `codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` yields `codex --dangerously-bypass-approvals-and-sandbox`. The installed CLI's own default model applies while the built-in's deliberate full-auto posture remains. This is how you spawn a provider whose current model IDs you do not know.

**Provider-form caveat:** `fab agent --provider <name>` bypasses role resolution and both per-role fill sources, so its profile is exactly the passed flags; pass `--model` explicitly or use the role form when fills should apply. See `_cli-fab.md` § fab agent for the fill ladder and model-free Codex example.

**Open it in a pane** — the mechanized form composes *and* spawns in one step, with **no prompt attached**:

```sh
fab pane open --provider <name> [--role <role>] [-c <dir>]
```

It resolves the fills through the standard precedence with the provider pinned (unlike `fab agent --provider`'s bypass), spawns a plain split of your window inside tmux (an unnamed new window otherwise), prints the new pane id, and writes no dispatch state. The prompt goes in afterwards, verified: loop `fab pane ready %N` until it reports `ready` — answering any wall the non-`ready` report's snippet shows between probes — then `fab pane deliver %N --text "<prompt>"`. This 3-command flow is also the **provider-probe recipe** (a pre-ship probe, a first-run wall discovery — the rpsr/ki9v flow): open, probe, answer walls, deliver a probe prompt, and record what the walls turned out to be.

The raw form — for when the prompt should ride the spawn itself:

```sh
tmux new-window -n "<name>" -c "<dir>" "<composed-cmd> '<initial-prompt>'"
```

- The composed command is the *whole* left-hand side — including any shell expansions it carries (e.g. `$(basename "$(pwd)")`), which expand at invocation inside the new window.
- The initial prompt is embedded **at spawn** as a single quoted argument. Shell-escape any user-supplied text before embedding it.
- **One prompt, one leading command.** The embedded string is delivered as a single prompt to the agent, where `&&` is *not* a shell operator and an agent harness reads at most one leading `/command` — so an `&&`-joined pair of slash commands does not run two commands; the tail is swallowed into the first command's argument. Embed exactly one command; if a second step is genuinely required, run it as a synchronous CLI call before opening the window, or as a separate Enter-terminated send afterwards.
- **Embedding at spawn also sidesteps the printed-prompt trap** (§ Delivery Probe): the trap needs a *pre-existing* input buffer to mistake printed output for, and a window created with its prompt already attached has none. Prefer spawn-embedding over "open the window, then send the prompt" whenever the prompt is known up front **and the send is one you can afford not to verify** — a one-shot argument either is or is not ingested, and you cannot tell which. Where that matters, verify instead of embedding: `fab pane ready` → `fab pane deliver --text` mechanizes exactly that sequence (echo-checked send, submit, screen-advance confirm — § Delivery Probe), with the manual recipe as the fallback for non-fab driving. That is exactly the trade the pane dispatch adapter makes (below).

> **Pipeline consumer**: `fab dispatch open <change> <stage>` (the interactive-pane dispatch adapter — `_cli-fab.md` § fab dispatch, contract in `docs/specs/harness-adapters.md`) is built on this procedure's LAUNCH half only. It composes the resolved provider's `interactive_command` and opens it **verbatim, with no prompt attached** — because a multi-thousand-token stage prompt cannot ride argv, and a positional one-shot cannot be verified. The prompt arrives afterwards as a **one-line pointer** typed by `fab dispatch deliver` behind the `fab dispatch ready` gate, which is the adapter's own answer to the printed-prompt trap: an echo-checked send beats an unverifiable spawn argument. Since 260810-1lah the dispatch verbs are thin record-keeping bindings over this section's own primitives — `fab pane open`/`ready`/`deliver` — so adapter and procedure share one gate and one delivery choreography. Completion detection follows § Await's "prefer asking for an artifact over a screen pattern" rule (the worker's `{stage}-result.yaml`).

### Pre-Send Validation

Before sending keys into an *existing* agent pane:

1. **Verify the pane exists** — refresh the pane map (`fab pane map [--all-sessions] [--json]`). A dead pane accepts keys silently into nothing, so a stale pane ID is a silent-failure hazard, not an error you will see.
2. **Verify the agent is idle** — read the pane's agent state. The state is the three-state `@rk_agent_state` convention plus unknown: `idle` is the only state safe to send to unattended; `active` (turn in progress) and `waiting` (blocked on a human) both risk corrupting work or cutting across a pending answer; `—`/unknown means the pane is uninstrumented. `fab pane send` enforces this same gate — `active`/`waiting` refuse without `--force`; unknown only **warns** (`warning: agent state unknown — sending anyway` on stderr) and sends, since an uninstrumented or foreign-agent pane is ordinary, not refused — so prefer `fab pane send` over raw `tmux send-keys` and let the binary hold the gate. The convention's exact semantics (value format, the mandatory epoch suffix, what counts as unknown, the no-staleness-heuristic rule) are in `_cli-fab.md` § fab pane → § agent state.

Anything beyond these two mechanics (whether to ask the user, how many times to retry, whether the target pane is on the right change or branch) is the consumer's policy.

### Delivery Probe (the printed-prompt trap)

**The trap.** A `/command` visible at an agent's `❯` prompt may be *printed output*, not a live input buffer. Pressing Enter then submits nothing — a **silent no-op**. The pane looks exactly like a pane that is about to run your command.

**The mechanized probe.** `fab pane ready <pane>` runs the classification half (typed sentinel → echo check → `ready`/`booting`/`parked`, snippet attached), and `fab pane deliver <pane> --text "<cmd>"` IS the probe-and-retype recovery mechanized: readiness probe → `C-u` clear → type → echo-verify → Enter → confirm the screen advanced, with one retry and the pane's snippet on stderr if nothing verifies. Reach for these first; the manual recipe below is the fallback for non-fab driving — and the explanation of what the binary is doing.

**The probe-and-retype recovery (manual fallback):**

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

- **Built-ins:** `claude`, `codex`, `agy`, and `kimi` use the independent `interactive_command` / `headless_command` / `native` capability grammar in `internal/agent`'s embedded `defaults.yaml`; all resolve without a `providers:` block. Claude ships all three capabilities; codex, agy and kimi are non-native (pane + headless). `dispatch.mode` chooses the starting rung of the descending `pane → native → headless` ladder.
- **Fill-consuming paths:** depth knobs (`agent.session` / `agent.workers`), `agent.profiles.<role>.provider`, and `fab resolve-agent --provider` consume the built-in per-role fills and resolve a real model for every role — except on `kimi`, which ships none deliberately and resolves an empty model so the CLI's own `default_model` applies.
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
| Session naming | `-n <name>` names the session (the built-in `interactive_command` uses `-n "$(basename "$(pwd)")"`) |
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

**Why independent provider capabilities matter here.** `codex` (TUI) and `codex exec` (headless) are different invocations of the same binary, which is exactly why `providers.<name>` carries unmerged `interactive_command` and `headless_command` fields; the separate `native` boolean records an Agent-tool seam when one exists. These fields say how a rung runs, never which dispatch mode to prefer.

### agy

The Antigravity CLI. Verified against v1.1.11.

| Aspect | Value |
|--------|-------|
| Interactive | `agy --dangerously-skip-permissions --model {model}` — the shipped built-in `interactive_command` |
| Headless | `agy -p "<prompt>"` — `-p` takes the prompt as an **ARGUMENT** and **ignores stdin** |
| stdin delivery | Requires a **nested shell**: `sh -c 'agy … -p "$(cat)"'`. POSIX expands `$(cat)` *before* the outer `< prompt.md` redirect applies, so the un-nested form reads the *outer* stdin and the worker gets an empty prompt. The inner `sh`'s stdin is the redirected file — which is why the built-in `headless_command` nests |
| Timeout | `--print-timeout <dur>` — the 5m default kills long stage workers, so the built-in `headless_command` raises it to `120m` |
| Profile flags | `--model <id>`. **No effort flag is used** — agy's model IDs *embed* the reasoning level as a suffix (`gemini-3.1-pro-high`), so a separate `--effort` would fight the suffix; the built-in grammar omits `{effort}` entirely |
| Approvals | `--dangerously-skip-permissions` on both forms — unattended stage workers cannot answer approval prompts |
| Model discovery | `agy models` on the installed binary lists the accepted `--model` values (effort-suffixed families plus cross-vendor entries). fab ships per-role agy fills (`default`/`fast`) and validates neither, so the recipe is how you verify one. Pin a discovered ID as `providers.agy.profiles.<role>.model` — the modern spelling — or pass `--model` per invocation. The flat `providers.agy.model` is an **alias for `profiles.default`** and is outranked by the shipped `fast` role fill |

**Template consequence.** Because the agy grammar carries no `{effort}` placeholder, the substitution is all-or-nothing per placeholder present: a resolved effort simply has nowhere to go and is not injected. That is the intended behavior — provider grammar is the provider's, and fab never appends a flag the CLI does not have.

**Pane-capable.** The agy built-in ships an `interactive_command` — `agy --dangerously-skip-permissions --model {model}`, the same full-auto posture flag as its headless form — so like every built-in it is eligible for **pane-mode dispatch**. What had to be answered first is agy's FIRST-RUN behavior, not its prompt grammar (fab appends nothing to `interactive_command`; the prompt is typed post-spawn — § Spawn Composition): agy gates a fresh workspace behind an interactive **trust prompt** even under `--dangerously-skip-permissions`, and worktree-per-change makes every dispatch a fresh workspace. That wall is an ordinary readiness-gate **judgment round** — the pane reads `parked` until the prompt is answered, one answer clears it, and the answer is remembered per folder, so it amortizes across a checkout (the kimi precedent, probed live 2026-08-10). The trust store (`~/.gemini/antigravity-cli/settings.json`, exact-match paths) is additionally user-seedable — a user-side optimization, nothing fab ships. An absent `interactive_command` in a user's own `providers:` block remains a valid configuration the ladder handles (`descended: pane unavailable: no interactive_command`, and an explicit `fab dispatch open` hard-errors actionably) — it is just no longer a shipped state.

**Skill discovery.** agy reads `<workspace>/.agents/skills/<skill>/SKILL.md` natively — which is why `fab sync` deploys **one** skill set to `.agents/skills/` and no `.agy/skills/` directory. One target per skill set is what makes duplicate-skill conflicts impossible.

### kimi

The kimi-code CLI (Moonshot's Kimi K3 agent). Verified against v0.34.0.

| Aspect | Value |
|--------|-------|
| Interactive | `kimi --auto -m {model}` — the shipped `interactive_command` (`--auto` is the full-auto posture the headless form rejects; `--yolo` is the same idea and equally interactive-only) |
| Headless | `kimi -p "<prompt>"` — like agy, `-p` takes the prompt as an **ARGUMENT** and ignores stdin, so stdin delivery uses the same nested-shell `"$(cat)"` idiom |
| Approvals | `-p` is **already** non-interactive and auto-approves tool calls. It **REJECTS** `--yolo`/`--auto` (`Cannot combine --prompt with --yolo`), so the built-in `headless_command` carries no approval flag at all — the full-auto flag is meaningful only on an interactive invocation, which is why the built-in `interactive_command` is the one carrying `--auto` |
| Profile flags | `-m <alias>` — note this is a **user-config model alias**, not a vendor catalog ID |
| Fills | **fab ships NONE**, deliberately: the `-m` alias set differs per install (managed installs expose `kimi-code`/`k3`; custom providers differ), so a pinned value would break non-managed setups. The empty `{model}` drops `-m` entirely and kimi falls back to the user's configured `default_model` |
| Model discovery | Read the installed CLI's own model configuration (its config file / `kimi --help`) for the aliases *that install* accepts — there is no portable catalog to quote. Pin one per role with `providers.kimi.profiles.<role>.model` if you want role differentiation; otherwise every role inherits `default_model` |

**Pane-capable.** kimi ships **both** command fields, so it is eligible for interactive sessions and for pane-mode dispatch. Its lack of an interactive-initial-prompt flag (`-p` is non-interactive; upstream issue #2240 tracks the gap) does not matter: delivery moved off the launch command, and fab types the pointer into the running TUI itself. The two things that had to be established were probed live on 2026-08-10 against v0.34.0:

- **First run**: kimi gates a fresh folder behind a `Trust this folder?` wall. That is an ordinary readiness-gate **judgment round** — the pane reads `parked` until the prompt is answered, one answer clears it, and the answer is remembered per folder — so it needs no code and amortizes across every later pane worker in the same checkout.
- **Input echo**: kimi draws **vertical side rules** down its input box, so a wrapped line arrives in a capture with `││` interleaved between the halves. `fab dispatch deliver`'s echo verification ignores box-drawing runes as well as whitespace, so the pointer verifies.

**Caveat**: the trust wall is the one interactive step a first pane worker in a fresh worktree spends a judgment round on. Budget is 2 rounds, so it costs one and leaves one.

**Skill discovery.** kimi reads the generic workspace `.agents/skills/` directory alongside its brand group, merged by priority — which is why `fab sync` deploys **one** skill set to `.agents/skills/` and no `.kimi/skills/` directory. One target per skill set is what makes duplicate-skill conflicts impossible.

### Codex MCP Bridge (recipe)

For tool-mediated, multi-turn cross-provider conversation (rather than one-shot output scraping), register the codex CLI as an **MCP server** inside a claude session. This is **session configuration, not fab machinery** — fab ships nothing for it.

Recipe:

1. **Capability-probe the installed codex** for MCP-server support before relying on it: `codex --version`, then confirm `codex mcp-server` is present in `codex --help`. An older binary may not expose it. Do **not** reach for `codex mcp` — that subcommand *manages the external MCP servers codex connects to* (list/add/remove/login); `codex mcp-server` is the one that starts codex itself as a stdio MCP server.
2. **Register `codex mcp-server` as an MCP server** in the claude session's MCP configuration (the harness's own MCP server config — a stdio server whose command is `codex mcp-server`).
3. **Converse through the resulting tools** — the codex-side tools appear in the claude session's tool list, so a multi-turn exchange becomes ordinary tool use with full context retained on both sides, instead of piping temp files and reading back terminal scrollback.

Prefer this over pane-driving whenever the goal is a *conversation* (asking codex about a diff, iterating on its answer). Prefer pane-driving (Half A) when the goal is a *long-running autonomous task* you want to watch and steer.
