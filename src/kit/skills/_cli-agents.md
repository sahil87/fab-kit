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
    - Skill Prompts
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

It deliberately carries **no orchestration policy**. Repo targeting, worktree creation, change-pointer activation, tracked-item enrollment, dependency resolution, queue choreography, confirmation tiers, and bounded-retry budgets are **operator** concerns and live in `fab-operator.md` (with the fab-owned `wt`/tmux choreography in `_cli-external.md`). A consumer of this file supplies its own policy around these primitives.

The `fab` commands referenced here (`fab agent`, `fab pane open`/`ready`/`deliver`, `fab pane map`) are documented in `_cli-fab-operator.md` (`fab agent`) and `_cli-fab-pane.md` (`fab pane …`) — load them too when you need their exhaustive flag surface. Messaging into an existing agent pane rides `rk mux send`/`rk mux await`, and pane peek, removal, and process-tree inspection ride the substrate twins `rk mux capture`/`rk mux kill`/`rk mux process` (tool-owned contracts: `rk skill`; fab-owned usage summarized in § Pre-Send Validation, § Peek, and § Await below). fab's own `fab pane capture`/`kill`/`process` are dispatch-internal — kept for the rk-less pane arm per cli-layering Part 7 (`_cli-fab-pane.md` § fab pane).

---

## Half A — Agent-Interaction Procedures

### Spawn Composition

An interactive agent session command is **never hand-assembled**. Ask `fab agent` to compose it, then open the composed command in a pane.

Two addressing forms — both print a fully **profile-resolved** command (`{model}`/`{effort}` substituted, or Claude-style flags appended for a non-templated `interactive_command`). The positional also accepts a **stage** name (`apply`, `review`, `hydrate`, `ship`, `intake`, `review-pr`), which maps to its fixed role; `-t` prints the raw template, `--headless` picks the `headless_command` slot, `-o yaml` emits the resolution as a structured document:

```sh
# Selector-addressed — the resolved role (or stage→role) supplies {provider, model, effort}.
fab agent --print                       # the default role, this repo
fab agent operator --print              # a named role
fab agent apply --print                 # a stage → its mapped role (byte-identical to `fab agent doing --print`)
fab agent --print --repo <target-repo>  # another repo's config (never your own)

# Provider-addressed — bypasses role resolution entirely.
fab agent --provider codex --print                                  # bare invocation: the CLI's own default model
fab agent --provider codex --model <id> --effort <level> --print     # explicit profile

# Inspection taps.
fab agent apply -t                    # the unsubstituted template (placeholders intact)
fab agent doing --headless --print    # the headless_command slot instead of interactive_command
fab agent apply -o yaml               # full structured resolution; schema: _cli-fab-operator.md § fab agent
```

> **Every built-in provider has a session form.** Composition needs an `interactive_command`, and `claude`, `codex`, `agy`, and `kimi` all ship one. A wholly user-defined provider may deliberately omit the field; either form then errors actionably, naming the `providers.<name>.interactive_command` key to set.

**Which form to use.** Use the **selector** form when the spawn should inherit fab's role/budget policy (a pipeline-shaped worker, the operator's own coordinator) — including which provider the `agent.session` knob points Tier-1 agents at; a stage name is just shorthand for its mapped role. Use the **provider** form when the question is mechanical — "give me a codex session right here" — with no role to speak of. `--model`/`--effort` are legal on ANY form as final post-refill overrides (a selector combined with `--provider` re-resolves the role's fills from that provider instead — see `_cli-fab-operator.md` § fab agent). **No `providers:` block is needed for either form**: `claude`, `codex`, `agy`, and `kimi` are built-in providers.

**Empty model/effort is a feature.** Omitting `--model`/`--effort` on the provider form leaves the value empty, and the composition rule drops the placeholder's token *and* a preceding `-`-flag while retaining fixed flags — so `fab agent --provider codex --print` against `codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` yields `codex --dangerously-bypass-approvals-and-sandbox`. The installed CLI's own default model applies while the built-in's deliberate full-auto posture remains. This is how you spawn a provider whose current model IDs you do not know.

**Provider-form caveat:** the bare `fab agent --provider <name>` form bypasses role resolution and both per-role fill sources, so its profile is exactly the passed flags; when fills should apply, use a selector (optionally with `--provider` to re-resolve them from the named provider) or pass `--model` explicitly. See `_cli-fab-operator.md` § fab agent for the fill ladder and model-free Codex example.

**Open it in a pane** — the mechanized form composes *and* spawns in one step, with **no prompt attached**:

```sh
fab pane open --provider <name> [--role <role>] [-c <dir>]
```

It resolves the fills through the standard precedence with the provider pinned (unlike `fab agent --provider`'s bypass), spawns a plain split of your window inside tmux (an unnamed new window otherwise), prints the new pane id, and writes no dispatch state. The prompt goes in afterwards, verified: loop `fab pane ready %N` until it reports `ready` — answering any wall the non-`ready` report's snippet shows between probes — then `fab pane deliver %N --text "<prompt>"`. This 3-command flow is also the **provider-probe recipe** (a pre-ship probe, a first-run wall discovery — the rpsr/ki9v flow): open, probe, answer walls, deliver a probe prompt, and record what the walls turned out to be.

The raw form — for when the prompt should ride the spawn itself:

```sh
rk tab new --session =<session> --cwd <dir> --name <name> --ready --json -- <composed-argv…> ["<initial-prompt>"]
```

- The composed command goes after `--` as **argv tokens, never a composed shell string** — each token is its own argv element, so no shell quoting, escaping, or `;`-chaining is applied (or needed). An initial prompt the provider's interactive command accepts rides as one trailing argv token.
- **`--session`, `--cwd`, and `--name` pin where the tab lands** — there is no ambient-session guesswork. Which session is the right target is the caller's policy, not this file's.
- **`--ready` makes rk probe readiness before returning**; read the `--json` report's `ready:` verdict — the report is the `result` object of run-kit's `{"ok":true,"result":{…}}` envelope, and its `window_id`/`pane_id` live there too. A non-ready verdict is an ordinary judgment round — answer the wall the snippet shows, then probe again — the same loop `fab pane ready` mechanizes (§ Delivery Probe).
- **rk appends the interactive shell fallback itself**: when the agent exits for any reason (`/exit`, crash, a `C-c` too many), a fresh interactive shell takes over the tab — same cwd, scrollback and position survive. Never append your own `; exec …`; a composed shell string is exactly what the argv form exists to prevent. **Scope rule: interactive spawns only.** `fab dispatch open` pane workers and `fab pane open` are excluded — their `running`/`done`/`orphaned` state machine treats pane death as the worker's terminal event, so a surviving fallback shell would read as a live worker. The full `rk tab new` contract is tool-owned (`rk skill`).
- **One prompt, one leading command.** The embedded prompt token is delivered as a single prompt to the agent, where `&&` is *not* a shell operator and a skill prompt carries one leading invocation rendered per § Skill Prompts — so an `&&`-joined pair of skill invocations does not run two commands; the tail is swallowed into the first command's argument. Embed exactly one command; if a second step is genuinely required, run it as a synchronous CLI call before opening the tab, or as a separate Enter-terminated send afterwards.
- **Embedding at spawn also sidesteps the printed-prompt trap** (§ Delivery Probe): the trap needs a *pre-existing* input buffer to mistake printed output for, and a tab created with its prompt already attached has none. Prefer spawn-embedding over "open the tab, then send the prompt" whenever the prompt is known up front **and the send is one you can afford not to verify** — a one-shot argument either is or is not ingested, and you cannot tell which. Where that matters, verify instead of embedding: `fab pane ready` → `fab pane deliver --text` mechanizes exactly that sequence (echo-checked send, submit, screen-advance confirm — § Delivery Probe), with the manual recipe as the fallback for non-fab driving. That is exactly the trade the pane dispatch adapter makes (below).

> **Pipeline consumer**: `fab dispatch open <change> <stage>` (the interactive-pane dispatch adapter — `_cli-fab-pane.md` § fab dispatch, contract in `_preamble.md` § CLI-Adapter Dispatch) is built on this procedure's LAUNCH half only. It composes the resolved provider's `interactive_command` and opens it **verbatim, with no prompt attached** — because a multi-thousand-token stage prompt cannot ride argv, and a positional one-shot cannot be verified. The prompt arrives afterwards as a **one-line pointer** typed by `fab dispatch deliver` behind the `fab dispatch ready` gate, which is the adapter's own answer to the printed-prompt trap: an echo-checked send beats an unverifiable spawn argument. Since 260810-1lah the dispatch verbs are thin record-keeping bindings over this section's own primitives — `fab pane open`/`ready`/`deliver` — so adapter and procedure share one gate and one delivery choreography. Completion detection follows § Await's "prefer asking for an artifact over a screen pattern" rule (the worker's `{stage}-result.yaml`).

#### Skill Prompts

An explicit skill invocation is `<prefix><skill>[ <arguments>]`, where the prefix belongs to the **receiving** harness: `codex` ⇒ `$`, every other or unknown provider ⇒ `/`. `internal/agent.SkillPrefix` owns the rule; `fab agent … -o yaml` exposes it as `skill_prefix` (`_cli-fab-operator.md` § fab agent). This applies to initial prompts and to skill commands routed into existing panes.

1. **Identify the receiver.**
   - *Fresh default-role spawn*: resolve the target repo once with `fab agent default -o yaml --repo <target-repo>` and read both `command` (the session command — byte-identical to what `--print` would emit) and `skill_prefix` from that one document. Do not run `--print` separately or resolve the provider a second way.
   - *Other fresh roles*: the `skill_prefix` of the resolution already made for that launch (`fab agent <role> -o yaml`).
   - *Existing pane*: the live agent process per § Peek's process-tree command; the interactive harness executable — including a child beneath a wrapper shell — is the receiver. Never identify it from prompt text, a nested tool worker, the operator's own provider, a model ID, or the repo's current default. Unknown ⇒ `/`. This does not bypass the caller's state/confirmation gate.
2. **Compose and quote once.** Render `<prefix><skill>[ <arguments>]` with the arguments verbatim, then deliver the whole prompt as **one token** exactly like any other initial prompt — one trailing argv token for a new tab (§ Spawn Composition's spawn-embedding rule, no shell quoting), the normal send mechanism (`rk mux send <pane> "<prompt>"`) plus the pre-send gate for an existing pane. Never embed a literal `$`-prefixed skill inside an outer double-quoted shell string.
3. **Keep message types distinct.** Only explicit skill invocations take a prefix. Ordinary prompts, answers, keys, `Read <path> and execute it.` pointers, and native TUI controls such as `/loop` or `/clear` use their own contracts and are not transformed.

### Pre-Send Validation

Before sending keys into an *existing* agent pane:

1. **Verify the pane exists** — refresh the pane map (`fab pane map [--all-sessions] [--json]`). A dead pane accepts keys silently into nothing, so a stale pane ID is a silent-failure hazard, not an error you will see.
2. **Verify the agent's state fits the send intent** — read the pane's agent state. The state is the three-state `@rk_pane_agent_state` convention plus unknown: `idle` is the only state safe to route a *command* to unattended; `active` (turn in progress) risks corrupting work; `waiting` (blocked on a human) means the pane wants an *answer*, not a command — sending anything else cuts across the pending prompt (classify it first: `rk mux capture <pane> --lines 40 --classify --json` names the pending-prompt class and the matched line — § Peek); `—`/unknown means the pane is uninstrumented. `rk mux send` enforces this gate with a mode per intent: plain send for command routing (`active`/`waiting` refuse), `--answer` for delivering a detected prompt's answer to a `waiting` agent (still refuses `active`), `--force` as the deliberate skip-everything override. Unknown only **warns** and sends in both non-force modes, since an uninstrumented or foreign-agent pane is ordinary, not refused. So prefer `rk mux send` over raw `tmux send-keys` and let the binary hold the gate — the flags map one-to-one (plain, `--answer`, `--force`, `--no-enter`, `-L/--server`), probe-verified delivery is built in (§ Delivery Probe), and `--key Enter|Up|C-c` covers the key-name input a literal-text send cannot express, closing the old raw-keys carve-out. The convention's exact semantics (value format, the mandatory epoch suffix, what counts as unknown, the no-staleness-heuristic rule) are in `_cli-fab-pane.md` § fab pane → § agent state; the full `rk mux send` contract is tool-owned (`rk skill`).

3. **Raw sends only — clear any pane mode first.** Keys sent into a pane that is in a tmux mode (`#{pane_in_mode}` = 1 — copy-mode from a human scrolling up, choose-tree, …) are consumed as mode key bindings: `send-keys` exits 0 and **nothing reaches the application** (verified tmux 3.7c) — a silent no-op, not an error you will see. Before a raw `tmux send-keys` (the pre-delivery judgment rounds, any manual driving), probe `tmux [-L <server>] display-message -p -t <pane> '#{pane_in_mode}'` and, only when it prints `1`, run `tmux [-L <server>] send-keys -X -t <pane> cancel` (unconditional cancel errors outside a mode). fab's own senders (`fab pane deliver`/`ready`, `fab dispatch deliver` — everything riding the Go send seam) apply this guard automatically; `rk mux send` does not yet carry it, so probe/cancel first when the target pane may have been scrolled. Mode commands (`send-keys -X`) are not blocked by a read-only tmux client, so the guard still works in environments where ordinary sends are blocked.

Anything beyond these mechanics (whether to ask the user, how many times to retry, whether the target pane is on the right change or branch) is the consumer's policy.

### Delivery Probe (the printed-prompt trap)

**The trap.** A `/command` visible at an agent's `❯` prompt may be *printed output*, not a live input buffer. Pressing Enter then submits nothing — a **silent no-op**. The pane looks exactly like a pane that is about to run your command.

**The mechanized probe.** `fab pane ready <pane>` runs the classification half (agent-takeover precondition first — a shell foreground reports `booting` with NOTHING typed, since a cooked-mode shell echoes the sentinel by itself; the classification itself, including its rk delegation, is owned by `_cli-fab-pane.md` § fab pane ready), and `fab pane deliver <pane> --text "<cmd>"` IS the probe-and-retype recovery mechanized: readiness probe → `C-u` clear → type → echo-verify → Enter → confirm the screen advanced, with one retry and the pane's snippet on stderr if nothing verifies. For message sends into an existing pane, `rk mux send` mechanizes the same probe with delivery verification built in — a probe failure stages the text, warns on stderr, and exits 1: re-capture and decide, never blind-resend. Reach for these first; the manual recipe below is the fallback for non-fab/raw-tmux driving and the explanation of what the binaries are doing.

**The probe-and-retype recovery (manual fallback):**

1. Send a literal, harmless sentinel (e.g. `XYZTEST`) with **no** Enter.
2. Re-capture the pane. If the sentinel appears appended to the visible prompt text, the buffer is live. If it does not appear, what you are looking at is printed output.
3. Either way, clear the line with `C-u`, retype the intended command, then send Enter.
4. **Confirm delivery** — re-capture and look for a working indicator (a spinner / "thinking" line / the agent's own turn start). Absence of a working indicator after a send means the command did not land; do not assume success from the send call's exit status.

Treat "the send call succeeded" and "the agent received the command" as independent facts. Only the post-send capture establishes the second.

### Peek

Two independent axes, read separately:

- **Output** — `rk mux capture <pane> [-l N] [--json|--raw]` (every `rk … --json` report in this section is the `result` of run-kit's `{"ok":true,"result":…}` envelope; an `{"ok":false,"error":{…}}` document is the failure form): substrate-enriched capture — `-l N` returns the last N lines (blank screen-padding stripped, tailed internally), `--raw` the bare captured text, `--json` the metadata wrapper carrying the pane's reconciled agent state. A wide window (`-l 50`+) compensates for line wrapping when scanning for a prompt.
- **Agent state** — the pane's `@rk_pane_agent_state` option (surfaced by `fab pane map`'s Agent column and by `rk mux capture --json`'s reconciled agent-state fields — under `result` in run-kit's `{"ok":true,"result":{…}}` envelope form, an object for this verb).
- **Process tree** (when the question is what is *running*, not what is on screen) — `rk mux process <pane> [--json]`: the pane's process tree, with a live agent pid from `@rk_pane_agent_state` classifying its node `agent` authoritatively. Removing a pane rides `rk mux kill <pane>` — agent-state-gated (refuses `active`/`waiting`; `--force` skips the gate).

> **State-writer caveat — uninstrumented panes read unknown.** fab is a pure *consumer* of `@rk_pane_agent_state`; the writer is HexoKit's `rk agent setup` global agent-harness hooks, which cover **Claude Code, Codex, Copilot, Gemini, and OpenCode** — not just Claude (see `_cli-fab-pane.md` § fab pane → § agent state). The unknown case is therefore an **uninstrumented pane**, not a non-claude one: no `rk agent setup` was ever run in that environment, or the pane runs a harness those hooks do not cover. Such a pane reports `—` (unknown) — indistinguishable from "no state". So **capture is the universal fallback**: state reads are an optimization on instrumented panes, not a portable signal. Never gate a flow on a non-unknown state read.

**Question detection rides `rk mux capture --classify`.** When the question is "is this pane blocked on a prompt, and what shape of answer does it want?", run `rk mux capture <pane> --lines 40 --classify --json`: the report — the `result` object of run-kit's `{"ok":true,"result":{…}}` envelope — carries a pending-prompt **class** — yes/no, numbered menu, colon prompt, open question, press-key, or `none` with a reason — plus the **matched line**, which together feed the caller's answer model. The classification contract is tool-owned (`rk skill`); plain capture remains the fallback for reading the screen itself.

### Await

There is **no cross-provider completion notification.** The Agent-tool style "sub-agent finished, here is its result" callback has no equivalent when you drive a CLI in a pane. The available signals are:

1. **`rk mux await`** — `rk mux await <pane> [--until <states>] [--file <path>] [--timeout <dur>]` is the preferred mechanized wait, and the composed `rk mux send --await` covers send-then-wait. It reports one of `idle`/`waiting`/`file`/`running`/`gone` — `gone` = exit 1 (rk's toolkit codes, not fab's old 2/3 scheme); the full contract is tool-owned (`rk skill`).
2. **Poll** (when no `await` signal fits — e.g. a completion signal that is a screen pattern) — loop `capture` (+ the state read, where instrumented) on an interval until a completion signal appears. Use a fixed, unhurried cadence; a completion signal is a *pattern you defined*, e.g. the agent's own summary line, a sentinel string you asked for in the prompt, or a file the worker was told to write.
3. **Have the worker announce itself** — instruct the worker in its prompt to run a notification as its last act (e.g. `rk notify` — fail-silent per `_preamble.md` § HexoKit (rk) Reference). This converts polling into an event, at the cost of depending on the worker honoring the instruction.

Prefer asking for an **artifact** (a file at a path you name) over a screen pattern: a file's presence is unambiguous, survives scrollback, and is readable without the pane. A screen pattern is the fallback when you cannot dictate the worker's output contract.

---

## Half B — Provider Dictionary

### Dictionary Discipline

Each entry below carries only **stable invocation grammar** and **discovery recipes**:

- **Grammar is stable** — whether a CLI's headless mode is a flag or a subcommand, whether it reads the prompt from stdin or an argument, what its structured-output flag is called. These change on major-version boundaries, not weekly.
- **Model IDs are NOT recorded *here*.** Model catalogs rot in weeks, and this dictionary is read by an agent that may be reasoning about an uninstalled CLI. Instead each entry carries a *discovery recipe*: what to run against the **installed** binary to learn which models it accepts. Never assume a model ID from memory; run the recipe. *(fab-kit does ship per-role model fills in the binary — `providers.<name>.profiles` — but those are DATA refreshed at kit-release cadence and overridden by one config line, not knowledge an agent should carry. The recipe is still how you verify one.)*
- **Quirks accrete from real encounters only.** An entry records an interactive quirk (first-run trust prompt, submit-key behavior) only once it has actually been hit and confirmed. Speculating about an uninstalled CLI's behavior is worse than silence — it reads as verified.

- **Built-ins:** `claude`, `codex`, `agy`, and `kimi` use the independent `interactive_command` / `headless_command` / `native` capability grammar in the module-root embedded `defaults.yaml`; all resolve without a `providers:` block. Claude ships all three capabilities; codex, agy, and kimi are non-native (pane + headless). `dispatch.mode` chooses the starting rung of the descending `pane → native → headless` ladder.
- **Exec contract** (the owner statement — point here, never restate): a provider's `interactive_command` MUST **exec its binary**, so the binary — not a wrapper shell — owns the pane's foreground; all four built-ins already do. The readiness gate requires exactly that takeover (a shell foreground reports `booting` and is typed into never), so a wrapper that keeps a shell in the foreground is **unsupported by design** and fails OBSERVABLY, never silently: every probe reports `booting` with the shell prompt in the snippet until the wiring's consecutive-booting allowance is spent and the run escalates. No time bound, no `spawn_cmd` comparison — the contract is the command's shape.
- **Fill-consuming paths:** depth knobs (`agent.session` / `agent.workers`), `agent.profiles.<role>.provider`, and selector-addressed `fab agent <role|stage> --provider <name> -o yaml` consume the built-in per-role fills and resolve a real model for every role — except on `kimi`, which ships none deliberately and resolves an empty model so the CLI's own `default_model` applies.
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
| Model discovery | The installed CLI's own help is authoritative: `claude --help` for the `--model` flag's accepted forms. fab's own `model_alias` mapping (`opus`/`sonnet`/`haiku`/`fable`) is an Agent-tool adapter, not a claude-CLI constraint — see `_cli-fab-operator.md` § fab agent |

**Agent-state instrumentation.** Claude Code is one of the harnesses `rk agent setup` instruments (alongside Codex, Copilot, Gemini, and OpenCode); instrumentation is an environment property, not a provider guarantee.

### codex

| Aspect | Value |
|--------|-------|
| Interactive | `codex` (TUI) |
| Headless | `codex exec` — a **subcommand**, not a flag; reads the prompt from **stdin** |
| Structured output | `codex exec --json` |
| Profile flags | `-m <id>` for the model; reasoning effort rides a config override: `-c model_reasoning_effort=<level>` |
| MCP server mode | `codex mcp-server` — starts codex AS a stdio MCP server (distinct from `codex mcp`, which *manages* the external MCP servers codex itself connects to); see § Codex MCP Bridge below |
| Model discovery | Capability-probe the installed binary: `codex --version`, then `codex --help` / `codex exec --help` for the `-m` flag's accepted values; the installed CLI also caches its live catalog at `~/.codex/models_cache.json` (per model: slug, priority, description, supported reasoning levels, visibility) — the verified source the shipped fills were read from. fab ships per-role codex fills for all six roles (`default`, `operator`, `doing`, `review`, `hydrate`, `fast`) and validates none of them, so the recipe is how you verify one. Pin a discovered ID as `providers.codex.profiles.<role>.model` — the modern spelling — or pass `--model` per invocation. The flat `providers.codex.model` is an **alias for `profiles.default`** and is outranked by the shipped role fills, so it reaches only the `default` role |

**Why independent provider capabilities matter here.** `codex` (TUI) and `codex exec` (headless) are different invocations of the same binary, which is exactly why `providers.<name>` carries unmerged `interactive_command` and `headless_command` fields; the separate `native` boolean records an Agent-tool seam when one exists. These fields say how a rung runs, never which dispatch mode to prefer.

### agy

The Antigravity CLI. Verified against v1.1.11.

| Aspect | Value |
|--------|-------|
| Interactive | `agy --dangerously-skip-permissions --model {model}` — the shipped `interactive_command` |
| Headless | `agy -p "<prompt>"` — `-p` takes the prompt as an **ARGUMENT** and **ignores stdin** |
| stdin delivery | Requires a **nested shell**: `sh -c 'agy … -p "$(cat)"'`. POSIX expands `$(cat)` *before* the outer `< prompt.md` redirect applies, so the un-nested form reads the *outer* stdin and the worker gets an empty prompt. The inner `sh`'s stdin is the redirected file — which is why the built-in `headless_command` nests |
| Timeout | `--print-timeout <dur>` — the 5m default kills long stage workers, so the built-in `headless_command` raises it to `120m` |
| Profile flags | `--model <id>`. **No effort flag is used** — agy's model IDs *embed* the reasoning level as a suffix (`gemini-3.1-pro-high`), so a separate `--effort` would fight the suffix; the built-in grammar omits `{effort}` entirely |
| Approvals | `--dangerously-skip-permissions` on both forms — unattended stage workers cannot answer approval prompts |
| Model discovery | `agy models` on the installed binary lists the accepted `--model` values (effort-suffixed families plus cross-vendor entries). fab ships per-role agy fills (`default`/`fast`) and validates neither, so the recipe is how you verify one. Pin a discovered ID as `providers.agy.profiles.<role>.model` — the modern spelling — or pass `--model` per invocation. The flat `providers.agy.model` is an **alias for `profiles.default`** and is outranked by the shipped `fast` role fill |

**Template consequence.** Because the agy grammar carries no `{effort}` placeholder, the substitution is all-or-nothing per placeholder present: a resolved effort simply has nowhere to go and is not injected. That is the intended behavior — provider grammar is the provider's, and fab never appends a flag the CLI does not have.

**First-run trust wall.** agy ships both command fields, so it is eligible for interactive sessions and pane-mode dispatch. A fresh workspace can park at agy's interactive trust prompt even under `--dangerously-skip-permissions`; treat that as an ordinary readiness-gate judgment round, answer it, then probe again before delivery. The answer is remembered for the exact workspace path. Operators may pre-seed that exact path in `trustedWorkspaces` in `~/.gemini/antigravity-cli/settings.json`; fab neither guesses paths nor writes the provider's trust store. Backlog `[agik]` retains the live open → ready → deliver verification after the current agy quota resets; the command grammar and gate behavior are fixture-pinned meanwhile.

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

- **First run**: kimi gates a fresh folder behind a `Trust this folder?` wall. That is an ordinary readiness-gate **judgment round** — the pane reads `parked`, one Enter clears it, and the answer is remembered per folder — so it needs no code and amortizes across every later pane worker in the same checkout.
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
