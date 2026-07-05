# Divest agent active/idle state production from fab-kit

> Backlog detail doc — written 2026-07-06 after a run-kit discussion session that audited
> agent-status detection across the competitive landscape (Claude Squad, Webmux, guppi,
> Agent of Empires, amux, agentdock, Codeman, Happy/Omnara/Vibe Kanban) and both kits'
> current mechanisms. Companion run-kit work is tracked in run-kit's backlog; the shared
> design decision is recorded in run-kit's constitution (Principle X, v1.4.0: "Hooks Carry
> Only the Underivable").

## Goal

fab-kit stops **producing** agent lifecycle state (active/idle) and becomes a pure
**consumer** of a shared tmux pane-option convention. The `.fab-runtime.yaml` `_agents`
pipeline is deleted wholesale.

**Independence principle** (the "why", per Sahil): fab-kit must function fully wherever it
runs — with or without tmux, with or without run-kit. Agent-state detection was never core
fab: it is a tmux-context observation feature that got bolted onto fab because no owner
existed. run-kit (`rk agent-setup`) is that owner now.

## The convention (read-side contract)

run-kit's `rk agent-setup` installs **global** agent-harness hooks (Claude Code, Codex,
Copilot, Gemini, OpenCode — per-agent registry) whose hook commands self-locate via
`$TMUX_PANE` and write a tmux **pane user option**:

```
@rk_agent_state = "<state>:<epoch_seconds>"     # state ∈ active | waiting | idle
```

- `active` — turn in progress (UserPromptSubmit/PreToolUse fired, no terminal event since)
- `waiting` — blocked on a human (Notification: permission_prompt | elicitation_dialog |
  agent_needs_input; PermissionRequest for agents that have it)
- `idle` — turn complete (Stop; idle_prompt as backstop)
- Option **absent** — no instrumented agent in this pane (render `—`, treat as unknown)
- The epoch suffix is mandatory — consumers compute idle duration from it and can apply
  staleness heuristics (an Esc-interrupted agent can leave a stale `active`).

The exact value schema is owned by the run-kit side — **coordinate before implementing**;
treat the above as the current draft. The convention is a data format in tmux, not a
dependency on run-kit software: fab reads it with plain tmux commands.

## Verified audit (2026-07-06, fab-kit HEAD)

Everything the `fab hook` pipeline writes into `.fab-runtime.yaml` `_agents`, and who reads it:

| Field | Readers | Verdict |
|---|---|---|
| `idle_since` (active/idle) | `fab pane send` gate, `fab pane map` Agent column, `fab pane capture` header | All tmux-scoped `fab pane *` consumers |
| `tmux_pane` / `tmux_server` | `findAgentByPane` matching | Exists only to join state onto panes |
| `pid` | GC `kill(pid,0)` liveness | Bookkeeping for the state file itself |
| `change` | **nobody** (write-only) | Dead weight — deletable regardless |
| `transcript_path` | **nobody** (write-only) | Dead weight — deletable regardless |

Key facts:
- `fab pane map`'s Change/Stage/display_state/pr_url columns come from the pane's cwd →
  `.fab-status.yaml` → `.status.yaml` (`internal/pane/pane.go` `ResolvePaneContext`), NOT
  from `_agents`. They are untouched by this change.
- Outside tmux, the hooks fire and write entries **nothing ever reads** (no `tmux_pane` to
  match) — pure dead weight until GC'd.
- The pipeline is Claude-only today: `fab pane send`'s idle gate is blind to codex/copilot/
  gemini agents. Reading the shared convention fixes that for free.

## Subtraction list

Delete:
1. `_agents` write pipeline: the state-tracking purpose of `fab hook stop`,
   `fab hook user-prompt`, `fab hook session-start` (`cmd/fab/hook.go`), including
   `WriteAgent`/`ClearAgent`/`ClearAgentIdle`/`UpdateAgent`, the GC sweep + `last_run_gc`
   throttle, the grandparent PID walker (`internal/proc/`), and the flock serialization
   (`internal/lockfile` — check for other consumers before deleting the package).
2. `internal/runtime/` `_agents` schema handling (`runtime.go`) — the whole `_agents` map,
   `.fab-runtime.yaml` read/write. If nothing else lives in `.fab-runtime.yaml`, the file
   itself is gone (check scaffolding/gitignore references).
3. `internal/pane/pane.go`: `ResolveAgentState`, `ResolveAgentStateWithCache`,
   `findAgentByPane`, `loadRuntimeForCache`, `LoadRuntimeFile` (+ the per-worktree cache in
   pane map).
4. Hook **settings entries** for Stop/UserPromptSubmit/SessionStart in the deployed
   `.claude/settings.json` template — follow the `fab hook artifact-write` removal
   precedent (y022): one-release no-op shims for un-migrated settings + a version
   migration that removes the entries. (SessionStart may have non-`_agents` uses — verify
   before removing; at audit time its only action was deleting the `_agents` entry.)

## What stays (rewritten as convention readers)

1. **`fab pane send` idle gate** (`cmd/fab/pane_send.go`, currently hard-refuses when
   state ≠ "idle"): read `@rk_agent_state` via
   `tmux [-L <server>] show-options -pv -t <pane> @rk_agent_state`.
   - `idle` → send. `active`/`waiting` → refuse (same error shape, now three-state aware).
   - Absent/unparseable → "unknown": refuse with a distinct message telling the caller to
     use `--force` (today's `--force` semantics unchanged).
2. **`fab pane map` Agent column** (`cmd/fab/panemap.go`): add `#{@rk_agent_state}` to the
   existing `list-panes` format string — zero extra subprocesses, and the `tmux_server`
   disambiguation problem evaporates (a pane option lives on exactly one server's pane).
   Column values: `active` / `waiting` / `idle (<duration>)` / `—`. Duration formatted from
   the epoch suffix with the existing `FormatIdleDuration`.
3. **`fab pane capture` header** (`cmd/fab/pane_capture.go`): same read, same display.
4. **JSON schema**: keep `agent_state` / `agent_idle_duration` field names in
   `pane map --json` / `pane capture --json` for consumer compatibility (run-kit still
   joins them during its own migration); `agent_state` gains the `waiting` value — note it
   in the schema docs.

## Operator skill impact

`fab-operator` (deployed skill) keys question detection, stuck detection (>15m idle), and
its pre-send checks off the pane-map Agent column — interface unchanged, data richer:
`waiting` makes menu/permission-blocked agents event-visible (today a mid-turn
AskUserQuestion/permission prompt fires no Stop, so the agent reads `active` and the
idle-only §5 sweep likely never probes it). Update the skill text where it assumes the
two-state active/idle vocabulary; consider tightening cadence on `waiting` directly.

## Sequencing

1. Convention schema agreed with run-kit (blocker — schema draft above).
2. run-kit ships `rk agent-setup` (the writer). Without a writer, fab's readers see
   "unknown" everywhere — which is why the reader rewrite and the pipeline deletion should
   land only after the writer exists on Sahil's machines.
3. fab-kit change (this doc): rewrite the three readers, delete the pipeline, migrate
   settings, update `docs/memory/runtime/runtime-agents.md` (describes the deleted
   system — hydrate will rewrite it to document the convention-reader model instead) and
   `SPEC-hooks.md`.

## Acceptance

- `rg "idle_since|_agents|fab-runtime" src/` → only convention-reader code and migration
  shims remain.
- `fab pane send` refuses on `active`/`waiting`/unknown, sends on `idle`, `--force`
  bypasses — covering a codex pane (previously invisible).
- `fab pane map` shows `waiting` for a Claude sitting on a permission prompt (manual probe).
- All fab commands behave identically outside tmux (no runtime-file writes anywhere).
- Version migration removes the three hook settings entries; a stale settings file
  invoking removed hooks gets no-op exit-0 shims for one release.
