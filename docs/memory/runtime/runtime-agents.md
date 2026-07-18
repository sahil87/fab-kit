---
type: memory
description: "Agent-state divestment: fab reads the `@rk_agent_state` tmux pane option (`state:epoch_seconds`; states active/waiting/idle, absent = unknown) via plain tmux commands — a data convention, NOT a run-kit software dependency; run-kit's `rk agent-setup` is the writer. The former `.fab-runtime.yaml` `_agents` PRODUCER pipeline (hooks/GC/PID-walker/flock/`internal/runtime`+`internal/proc`) is deleted; fab is now a pure consumer, via the pure `parseAgentState` parser + surviving `FormatIdleDuration`."
---
# Runtime Agents

**Domain**: runtime

## Overview

fab determines an agent's lifecycle state by **reading** a tmux pane user option, `@rk_agent_state`, with plain tmux commands. It does **not** produce that state, and it does **not** depend on run-kit software being installed — the option is a data convention in tmux, read with `tmux show-options`/`list-panes`, so fab reads it whether or not run-kit is present.

This is a divestment (ioku, 2026-07): fab-kit **stopped producing** agent active/idle lifecycle state and became a **pure consumer** of a shared convention. Agent-state detection was never core fab — it is a tmux-context observation feature that got bolted onto fab because no owner existed. run-kit is that owner now: its `rk agent-setup` global agent-harness hooks write `@rk_agent_state` for Claude Code, Codex, Copilot, Gemini, and OpenCode. fab reads it in three places — `fab pane map` (Agent column), `fab pane send` (idle gate), and `fab pane capture` (header). See [pane-commands.md](/runtime/pane-commands.md) for those readers and [hooks-may-enhance-never-own.md](/pipeline/hooks-may-enhance-never-own.md) for the principle this strengthens.

> **Release gate.** The writer does not exist yet. When ioku shipped, the installed `rk` had no `agent-setup` command and no pane carried `@rk_agent_state`. The change's PR is therefore **explicitly held** from merge/release until `rk agent-setup` exists on Sahil's machines — otherwise `fab pane send` gating and the operator's Agent column go blind everywhere (pane map all `—`, send refuses without `--force`). fab is the consumer; the schema draft below is the working contract, and fab reading a not-yet-written option degrades to "unknown" — the correct fallback.

## Requirements

### Read Contract: `@rk_agent_state`

fab reads the tmux **pane user option** `@rk_agent_state`, whose value is `"<state>:<epoch_seconds>"`:

```
@rk_agent_state = "idle:1751800000"      # state ∈ active | waiting | idle
```

| State | Meaning | Trigger (run-kit writer) |
|-------|---------|--------------------------|
| `active` | Turn in progress | UserPromptSubmit / PreToolUse fired, no terminal event since |
| `waiting` | Blocked on a human | Notification: permission_prompt / elicitation_dialog / agent_needs_input; PermissionRequest for agents that have it |
| `idle` | Turn complete | Stop; idle_prompt as a backstop |
| *(option absent)* | No instrumented agent in this pane | render `—`, treat as **unknown** |

- The **epoch suffix is mandatory.** Consumers compute idle duration from it (`now - epoch`, formatted by `FormatIdleDuration`) and can apply staleness heuristics (an Esc-interrupted agent can leave a stale `active`). A value without a parseable `:epoch` suffix is **unknown**, not a bare state.
- **Absent / unparseable / unknown-token → unknown.** An absent option, an unknown state token (outside `{active, waiting, idle}`), or a missing/non-integer epoch all resolve to unknown — displayed as `—`, gated by `fab pane send` as a distinct "unknown" refusal (see [pane-commands.md](/runtime/pane-commands.md)).

**Schema ownership is run-kit's.** The `"<state>:<epoch_seconds>"` grammar above is the current working contract (drafted in the ioku pickup doc, recorded in run-kit constitution Principle X "Hooks Carry Only the Underivable", v1.4.0). If run-kit changes the format later, adapting fab's reader is a follow-up change — the divergence risk is accepted.

**No run-kit software dependency.** fab reads the option with `tmux show-options -pv -t <pane> @rk_agent_state` (send/capture) and `#{@rk_agent_state}` in the `list-panes -F` format string (map). These are plain tmux commands against a pane option — a *data* convention, not a link against run-kit. fab works identically whether run-kit wrote the option or nobody did (nobody → unknown everywhere, the honest fallback). All commands behave identically **outside tmux** too: with no tmux server there is no pane to read, so there is simply no agent state — no runtime file is written or read anywhere.

#### Scenario: idle pane resolves to a duration

- **GIVEN** a pane whose `@rk_agent_state` is `idle:1751800000`
- **WHEN** any reader (`pane send`/`map`/`capture`) resolves that pane's state
- **THEN** the state is `idle` and the idle duration is `now - 1751800000`, formatted via `FormatIdleDuration`
- **AND** a pane with no `@rk_agent_state` option resolves to unknown (`—` / em-dash in displays)

### The Parser (`parseAgentState`) and Display Helper

The `"<state>:<epoch>"` parse lives in a **single pure function** `parseAgentState(raw string) (state string, epoch int64, ok bool)` in `src/go/fab/internal/pane/pane.go`, reused by all three readers so there is one authority for the grammar and it is unit-testable without a tmux server. `ok` is false for an empty value, a missing `:epoch` suffix, a non-integer epoch, or a state token outside `{active, waiting, idle}`. State tokens and the option name are named constants (`AgentStateActive`/`AgentStateWaiting`/`AgentStateIdle`, `AgentStateOption = "@rk_agent_state"`) — no magic strings.

`AgentDisplayFromOption(raw) (state, idleDuration string)` maps a raw option value to a display state plus an idle duration string (populated **only** for `idle`). `ResolvePaneContext` reads the raw option via `ReadAgentStateOption(paneID, server)` (a targeted `show-options -pv`, guarded against an empty paneID) and sets `AgentState` (`active`/`waiting`/`idle`, nil when unknown) + `AgentIdleDuration` (only for idle). Agent state is resolved for **every** pane class — before the not-a-git-repo / no-fab-dir early returns — so `send`/`map`/`capture` agree on non-fab panes.

**`FormatIdleDuration` survives** in `internal/pane/pane.go` — it formats the epoch-derived idle durations of the new readers (the one piece of the old code that carries forward, because idle duration is still meaningful).

#### Scenario: only a well-formed value parses

- **GIVEN** raw values `""`, `"active"`, `"idle:notanum"`, `"bogus:123"`, `"idle:1751800000"`
- **WHEN** `parseAgentState` runs on each
- **THEN** only `"idle:1751800000"` returns `ok=true` (`idle`, `1751800000`); all others return `ok=false`

### The Deleted Producer Pipeline

The entire `.fab-runtime.yaml` `_agents` producer subsystem was **deleted wholesale** in ioku. What is gone:

- **The hook write pipeline** (`cmd/fab/hook.go`): the whole `fab hook` command family — `fab hook stop|session-start|user-prompt`, plus `artifact-write` and `sync` — was **removed outright** (no shim period; see [hooks-may-enhance-never-own.md](/pipeline/hooks-may-enhance-never-own.md)), including `WriteAgent`/`ClearAgent`/`ClearAgentIdle`/`UpdateAgent`, the throttled GC sweep + `last_run_gc`, and the grandparent PID walker. `cmd/fab/hook.go` and `internal/hooklib/sync.go` are deleted; the plan-parsing helpers in `internal/hooklib/artifact.go` (change-type inference, section counting) survive — they feed `fab status refresh`, not any hook.
- **`internal/runtime/`** — the whole `_agents` map and `.fab-runtime.yaml` read/write. Nothing else lived in the file (only `_agents` + top-level `last_run_gc`), so the file concept died wholesale.
- **`internal/proc/`** — the grandparent PID walker (`proc_linux.go`/`proc_darwin.go`). Its sole importer was `cmd/fab/hook.go`; the comment-only reference in `internal/dispatch/dispatch_posix.go` was swept.
- **The `_agents` resolvers in `internal/pane/pane.go`**: `ResolveAgentState`, `ResolveAgentStateWithCache`, `findAgentByPane`, `loadRuntimeForCache`, `LoadRuntimeFile`, and the per-worktree runtime cache in pane map, plus the `_agents`/`idle_since`/`tmux_pane`/`tmux_server` schema-key constants.
- **The three hook settings entries** (`SessionStart`/`Stop`/`UserPromptSubmit`) from `.claude/settings.local.json`, removed by the `2.13.6-to-2.14.0` migration (for the checkout it runs in) — later re-swept across **every** worktree, main checkout included, by the `2.15.7-to-2.15.8` migration (weoh), since the committed `fab/.kit-migration-version` gate meant the per-checkout gitignored settings copies in sibling worktrees never re-ran the original edit (see [distribution/migrations.md](/distribution/migrations.md) § `2.15.7-to-2.15.8`) — plus deletion of any lingering `.fab-runtime.yaml`/`.fab-runtime.yaml.lock` across worktrees.

**`internal/lockfile` STAYS.** It is consumed by `cmd/fab/status.go`, `cmd/fab/preflight.go`, and `internal/score/score.go` for `.status.yaml` serialization. Only the **runtime** lock usage (`.fab-runtime.yaml.lock` in the deleted `internal/runtime`) went away with the runtime package.

#### Scenario: hook commands are gone, readers agree everywhere

- **GIVEN** the `fab hook` command family was removed (no `stop`/`user-prompt`/`session-start`/`artifact-write`/`sync` subcommands)
- **WHEN** an un-migrated `.claude/settings.local.json` still fires `fab hook <x>` (before the settings-cleanup migrations run)
- **THEN** it errors with a cobra unknown-command message on stderr and a non-zero exit — no `.fab-runtime.yaml` is created (nothing writes it anymore); the `2.13.6-to-2.14.0` migration removes the entry in the checkout it runs in, and the `2.15.7-to-2.15.8` migration (weoh) re-sweeps it out of every worktree — including the main checkout, whose settings file every worktree session resolves through (see [distribution/migrations.md](/distribution/migrations.md) § `2.15.7-to-2.15.8`)
- **AND** all three pane readers resolve agent state from `@rk_agent_state`, so a codex/copilot/gemini pane (previously invisible to the Claude-only pipeline) is now covered once its option is set

## Design Decisions

### Consumer-not-producer: read a shared convention, own nothing
**Decision**: fab-kit stops producing agent active/idle lifecycle state and becomes a pure **consumer** of the `@rk_agent_state` tmux pane-option convention. run-kit's `rk agent-setup` is the sole writer; fab reads with plain tmux commands and depends on no run-kit software.
**Why**: **Independence** — fab-kit must function fully wherever it runs, with or without tmux, with or without run-kit. Agent-state detection was never core fab; it is a tmux-context observation feature bolted on because no owner existed. run-kit is that owner now. Reading a shared convention (a) drops a whole producer subsystem (hooks, GC, PID walker, flock, runtime file) that was **dead weight outside tmux** — hooks fired and wrote `_agents` entries nothing ever read (no `tmux_pane` to match) — and (b) fixes **Claude-only blindness for free**: the old pipeline tracked only Claude Code's hooks, so `fab pane send`'s gate was blind to codex/copilot/gemini agents; run-kit's harness hooks cover them all. It also adds a **richer `waiting` state** (blocked on a human) that the Stop-only pipeline could not observe (a mid-turn permission prompt fires no Stop). The read is a *data* convention in tmux, not a software link — so the independence principle holds: no run-kit binary need exist for fab to read (or degrade to unknown).
**Rejected**: Keeping the producer subsystem (two writers' worth of drift risk, a per-hook-event latency tax, and permanent Claude-only blindness). Making run-kit a hard dependency (violates independence — fab must work with run-kit absent). Waiting for the writer before landing the reader (accepted: fab reads a not-yet-written option and degrades to unknown, the honest fallback; the PR is procedurally held from merge until the writer ships).
*Introduced by*: 260705-ioku-divest-agent-state-production

### Epoch suffix mandatory; unknown is a first-class outcome
**Decision**: The read contract is `"<state>:<epoch_seconds>"` — the epoch is **required**, and an absent option / unknown token / missing-or-bad epoch all collapse to a single **unknown** outcome (rendered `—`, gated by a distinct `pane send` refusal).
**Why**: The epoch lets consumers compute idle duration and apply staleness heuristics (an Esc-interrupted agent can leave a stale `active`), so a value without it carries no usable duration and is treated as unparseable rather than a bare state. Collapsing absent/unknown/unparseable into one "unknown" keeps the reader a simple, correct pure function (`parseAgentState`) and gives the operator/`pane send` one clean "no instrumented agent here" signal. No staleness heuristic ships in v1 — a stale `active` still refuses sends; `--force` is the escape hatch, and heuristics are a consumer follow-up.
**Rejected**: Optional epoch (loses duration and the staleness signal). Distinct outcomes for absent-vs-malformed (no reader acts on the distinction; one "unknown" is enough). A v1 staleness heuristic (simplest correct reader wins; `--force` already covers the stuck-active case).
*Introduced by*: 260705-ioku-divest-agent-state-production

### Single pure `parseAgentState`, reused by all three readers
**Decision**: The `"<state>:<epoch>"` parse is one pure function in `internal/pane/pane.go`, consumed by `pane send`/`map`/`capture`; `map` reads via the `list-panes -F` format string, `send`/`capture` via a targeted `show-options -pv`.
**Why**: A single grammar authority is tmux-free unit-testable and eliminates the three-drifting-copies anti-pattern. `map` already runs `list-panes -F`, so adding `#{@rk_agent_state}` is zero extra subprocesses (and the tmux-server disambiguation problem evaporates — a pane option lives on exactly one server's pane); `send`/`capture` operate on a single pane they already probe, so a targeted `show-options -pv` is the minimal read.
**Rejected**: Parsing inline at each reader (three drifting copies). A `show-options` per pane in `map` (extra subprocess per pane — explicitly forbidden).
*Introduced by*: 260705-ioku-divest-agent-state-production
