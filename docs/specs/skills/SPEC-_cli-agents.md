# _cli-agents

## Contents

- Summary
- Section Structure
- Half A — Agent-Interaction Procedures
- Half B — Provider Dictionary
- Extraction Boundary
- Key Properties
- Resolved Design Decisions

## Summary

Agent-CLI interaction reference — the generic **procedures** for driving another agent CLI in a tmux pane (spawn composition, pre-send validation, the delivery probe, peek, await) plus a three-provider **operational dictionary** (claude, codex, gemini) carrying stable invocation grammar and model-*discovery recipes* rather than volatile model catalogs.

The procedures are **extracted from `fab-operator.md`** (moved, not rewritten): before this change, the knowledge of how to spawn an agent CLI, deliver a prompt reliably into its TUI, peek at its output, and wait for it lived only inside a ~700-line orchestrator skill that no other skill (and no ad-hoc session) could load, and was written for a single provider by convention. Extraction makes the primitives loadable by any consumer and gives the drafted Phase 2 (interactive stage dispatch) a primitive layer to build on rather than a re-extraction.

This is an internal partial (`user-invocable: false`, `disable-model-invocation: true`, `metadata: internal: true`) — never invoked directly. It is **opt-in** via a consumer's `helpers:` frontmatter (or an in-body point-of-use read), **not** part of the always-load layer: `/fab-operator` declares `helpers: [_cli-agents, _cli-fab, _cli-external]`. Canonical source is the flat `src/kit/skills/_cli-agents.md`; `fab sync` deploys it to `.claude/skills/_cli-agents/SKILL.md`.

> **Unlike `_cli-fab.md` / `_cli-external.md`, this partial carries a SPEC.** `docs/specs/skills.md` § New Skill Checklist item 6 excludes those two as **pure-reference** partials whose content mirrors an external command surface (a SPEC would be a third copy of the same tables). `_cli-agents.md` defines *procedures* — behavior — so it falls on the "every other behavioral partial gets a SPEC" side of that policy. The exclusion policy is worded to name the two files explicitly rather than keying on the `_cli-` prefix.

## Section Structure

| Section | Covers |
|---------|--------|
| `## Scope Boundary` | States that only generic mechanics live here, and names the operator concerns that do not (repo targeting, worktrees, pointer activation, enrollment, dependency resolution, autopilot, confirmation tiers, retry budgets). Points at `_cli-fab.md` for the referenced `fab` commands' exhaustive flag surface |
| `## Half A — Agent-Interaction Procedures` | The five procedures: Spawn Composition, Pre-Send Validation, Delivery Probe, Peek, Await |
| `## Half B — Provider Dictionary` | Dictionary Discipline (the stable-grammar / discovery-recipe / confirmed-quirks-only rules) then one section per provider (claude, codex, gemini) plus the Codex MCP Bridge recipe |

## Half A — Agent-Interaction Procedures

| Procedure | Contract |
|-----------|----------|
| **Spawn Composition** | The invocation is never hand-assembled — `fab agent --print` composes it, in either the **tier-addressed** form (`fab agent [tier] --print [--repo <path>]`, inherits fab's role/budget policy) or the **provider-addressed** form (`fab agent --provider <name> [--model <id>] [--effort <level>] --print`, mechanical "give me a codex session"). Documents that omitting `--model`/`--effort` on the provider form is a *feature* (the empty-value token-drop yields a bare invocation, so the installed CLI's own default model applies — usable without knowing current model IDs). Opening it: `tmux new-window -n <name> -c <dir> "<cmd> '<prompt>'"`, with shell-escaping of user text and the **one-prompt / no-`&&`-chaining** rule (the embedded string is one prompt; `&&` is not a shell operator there and an agent reads at most one leading `/command`, so a chained tail is absorbed into the first command's argument) |
| **Pre-Send Validation** | The two-step gate before sending keys into an existing pane: (1) pane exists via a refreshed pane map (a dead pane swallows keys silently — a silent-failure hazard, not a visible error); (2) agent state is `idle` per the three-state `@rk_agent_state` read (`active` = turn in progress, `waiting` = blocked on a human, `—` = uninstrumented) — the same gate `fab pane send` enforces, so prefer `fab pane send` over raw `tmux send-keys` and let the binary hold it. Explicitly states that everything beyond these two mechanics (whether to ask the user, retry counts, whether the pane is on the right change/branch) is the **consumer's policy** |
| **Delivery Probe** | The printed-prompt trap and its recovery: a `/command` visible at an agent's `❯` prompt may be *printed output* rather than a live buffer, so Enter submits nothing (a silent no-op) and the pane looks identical either way. Recovery: send a literal sentinel with no Enter → re-capture to see whether it appended → `C-u`, retype, Enter → **confirm via a working indicator**. States the underlying rule: "the send call succeeded" and "the agent received the command" are independent facts, and only the post-send capture establishes the second |
| **Peek** | Two independent axes: output via `fab pane capture` (enriched; `--raw` for a plain `capture-pane`, a wide window to survive line wrapping) and agent state via `@rk_agent_state`. Carries the **state-writer caveat**: fab is a pure consumer and the writer is run-kit's `rk agent-setup` global agent-harness hooks, which cover Claude Code, Codex, Copilot, Gemini, and OpenCode (per `_cli-fab.md` § fab pane → § agent state) — so the unknown case is an **uninstrumented pane** (no `rk agent-setup` run, or an uncovered harness), *not* a non-claude one. Such a pane reports `—` indistinguishably from "no state", making **capture the universal fallback**; no flow may gate on a non-unknown state read |
| **Await** | States honestly that **no adapter recovers Agent-tool-style completion notifications** when driving a CLI in a pane. Two available signals: (1) **poll** capture (+ state where instrumented) on an unhurried fixed cadence until a completion signal *you defined*; (2) have the worker **announce itself** via a notification instructed in its prompt (e.g. `command -v rk`-gated fail-silent `rk notify`). Prefers an **artifact** (a file at a path you name — unambiguous, survives scrollback, readable without the pane) over a screen pattern, which is the fallback when the worker's output contract cannot be dictated |

## Half B — Provider Dictionary

**Dictionary discipline** (the invariant that keeps the dictionary from rotting):

- **Stable grammar only** — whether headless mode is a flag or a subcommand, stdin-vs-argument prompt delivery, the structured-output flag's name. These move on major-version boundaries, not weekly.
- **No baked model catalogs.** Each entry instead carries a **discovery recipe**: what to run against the *installed* binary to learn which models it accepts. Matches fab's document-don't-validate posture; the alternative rots in weeks.
- **Quirks accrete from confirmed encounters only.** An interactive quirk (first-run trust prompt, submit-key behavior) is recorded only once actually hit; speculating about an uninstalled CLI reads as verified and is worse than silence.

| Provider | Recorded facts |
|----------|----------------|
| claude | Interactive `claude` / headless `claude -p` (prompt on **stdin**); `--output-format stream-json` for structured output; `--model <id> --effort <level>` (the CLI accepts full IDs *and* short aliases); `-n <name>` session naming (used by the built-in `session_command`); discovery via the installed `claude --help`. Notes that claude panes are the case `rk agent-setup` most reliably instruments — the inverse of the § Peek caveat, and explicitly not to be generalized |
| codex | Interactive `codex` (TUI) / headless `codex exec` — a **subcommand**, not a flag, prompt on **stdin**; `codex exec --json`; `-m <id>` for the model and `-c model_reasoning_effort=<level>` for effort; `codex mcp-server` for MCP-server mode (explicitly distinguished from `codex mcp`, which manages the *external* MCP servers codex connects to); discovery via `codex --version` then `codex --help`/`codex exec --help`. Names codex as the concrete reason `providers.<name>` carries `session_command` and `dispatch_command` **unmerged** — TUI vs `exec` are different invocations of one binary |
| gemini | Interactive `gemini` / headless `gemini` with the prompt on **stdin** (read as the prompt in non-TTY mode); the **`-p` caveat** — `-p` takes prompt TEXT appended after stdin, so it is not the piping flag and the shipped template omits it; `-m <id>`, and **no effort flag** at all (the shipped template omits `{effort}`); discovery via `gemini --help`. Draws the template consequence: a resolved effort has nowhere to go and is not injected, which is intended — provider grammar lives in the config and fab never appends a flag the CLI lacks |
| Codex MCP Bridge | A **recipe only**, no fab machinery: capability-probe the installed codex (`codex --version`, confirm `codex mcp-server` in `--help`) → register `codex mcp-server` as a stdio MCP server in the claude session's own MCP config → converse through the resulting tools. Names the `codex mcp` / `codex mcp-server` distinction explicitly so the wrong subcommand is not reached for. States the choice rule: MCP bridge for a *conversation* (asking codex about a diff, iterating), pane-driving (Half A) for a *long-running autonomous task* to watch and steer |

The command strings are consistent with the `providers:` three-provider starter template fab-kit ships (`fab config reference`), whose codex/gemini blocks are commented until a user opts in. **The Go `defaultProviders` table stays claude-only** — this dictionary is markdown, so it updates per kit release with no behavior risk.

## Extraction Boundary

The boundary is **agent primitives vs. operator orchestration**:

| Moved to `_cli-agents.md` | Stays in `fab-operator.md` |
|---------------------------|----------------------------|
| Pane-exists + agent-idle gate mechanics (§3 items 1–2's mechanics) | The operator's policy on the gate's outcome (the "{change} is {state}. Send anyway?" confirmation, the pane-gone report), plus the change-active (item 3) and branch-alignment (item 4) checks |
| Session-command composition + tmux window-open steps (§6 steps 5–6's mechanics) | The **always pass `--repo <target-repo>`** rule (never the operator's own config), the `»<wt>` window-marker name, repo targeting, worktree creation, existence-guarded pointer activation, dependency resolution, enrollment |
| Capture / state-read mechanics and the uninstrumented-pane state-writer caveat | The operator's question-detection patterns and guards over that capture, the answer model, non-blocking strategic handling, notification send, idle auto-default, logging |
| The delivery probe | When to reach for it (a send that appears to land but leaves the agent idle), referenced from §3 and § Sending Auto-Answers |
| — | Confirmation tiers, bounded retries, autopilot, watches, the status frame, configuration |

`fab-operator.md` §2 Context Loading states the split in one sentence so a reader of either file knows which owns what.

## Key Properties

| Property | Value |
|----------|-------|
| User-invocable? | No — internal partial (`user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`) |
| In the always-load layer? | No — opt-in via `helpers: [_cli-agents]` or an in-body point-of-use read |
| Declared by | `/fab-operator` (`helpers: [_cli-agents, _cli-fab, _cli-external]`); any skill or ad-hoc session may load it |
| Defines a flow? | No — it is a procedure + reference file consumed by other skills |
| Records model IDs? | **No** — discovery recipes only, by design |
| Adds Go behavior? | No — the only Go surface it references is the `fab agent --provider` form added in the same change |

### Tools used

None directly — `_cli-agents.md` documents procedures its consumers execute (`fab agent`, `fab pane map/capture/send`, `tmux new-window`, `tmux send-keys` via Bash). The file itself runs nothing.

### Sub-agents

None.

## Resolved Design Decisions

1. **Opt-in helper, not always-load.** `_cli-*` helpers are opt-in by convention and the always-load layer stays lean; a cross-provider spawn is not something every skill invocation pays for. Consumers declare `helpers: [_cli-agents]` (or read it at point of use).

2. **Stable grammar + discovery recipes, never model catalogs.** Model catalogs rot monthly (the shll-conformance precedent: a live contract's details go stale, so re-fetch rather than cache). A recipe against the *installed* binary cannot go stale, and it matches fab's document-don't-validate posture. Rejected: shipping current model IDs per provider (stale within weeks, and wrong in a way that reads authoritative).

3. **Extraction moves, it does not copy.** Each relocated procedure is stated **once**, in `_cli-agents.md`; `fab-operator.md` keeps its operator-specific wrapper text and references the mechanics. Rejected: leaving a copy in the operator "for convenience" — the duplicate-truth failure mode the project's sibling/mirror-sweep rules exist to prevent (a canonical sentence with several homes drifts).

4. **Markdown dictionary, claude-only Go defaults.** The Go `defaultProviders` table stays claude-only and codex/gemini remain uncomment-to-opt-in config template text; the operational knowledge lives in markdown. This keeps the dictionary updatable per kit release with zero behavior risk and preserves the shipped opt-in posture for non-claude providers.

5. **MCP bridge is recipe text, not a phase.** Registering `codex mcp-server` in a claude session is session configuration, not fab machinery — it earns a subsection, not its own change. Paired with an explicit choice rule (bridge for conversation, pane-driving for watchable autonomous work) so a reader knows which primitive to reach for.

6. **The await section is honest about the missing primitive.** No adapter recovers Agent-tool-style completion notifications for a pane-driven CLI, so the file says so and names the two real signals (polling; a worker-announced notification) plus the artifact-over-screen-pattern preference. Rejected: presenting polling as equivalent to a completion callback — it would hide the latency/reliability cost a consumer must design around.
