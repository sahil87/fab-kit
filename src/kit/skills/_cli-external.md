---
name: _cli-external
description: "External CLI tool reference — wt (worktree manager), idea (backlog manager), hop (multi-repo navigator), tmux, and rk (HexoKit). Carries only fab-owned integration content; each tool's usage knowledge is delegated to `<tool> skill` at use-time and its exhaustive command tree to `<tool> help-dump`."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# External CLI Tool Reference

> Loaded on demand (not part of the always-load layer). Documents non-fab CLI tools used by fab workflows.

## Contents

- Reference Model
- wt (Worktree Manager)
- idea (Backlog Manager)
- hop (Multi-Repo Navigator)
- tmux
- rk (HexoKit)

---

## Reference Model

This file documents only **fab-owned** content — what each tool *is* in one line,
and the fab-specific integration knowledge that no tool's own documentation
carries (the pipeline escalation `rk notify` usage and the pane-substrate
delegation notes — skill-facing capture/kill/process ride rk's `mux` twins, with
fab's copies dispatch-internal). It deliberately does **not** restate any
tool-owned usage knowledge: that is delegated to each owned tool's own bundle at
use-time, so this file never goes stale against a tool's release cadence.

Each owned tool exposes two version-locked surfaces: `<tool> skill` for its usage
briefing and `<tool> help-dump` for its exhaustive command tree and flags.

### Delegation and binary gate

`wt`, `idea`, `rk`, and `hop` are optional sibling binaries. Gate every
informational delegation on `command -v` and fail silently when the binary is
absent:

```sh
command -v wt   >/dev/null 2>&1 && wt skill        # gated, fail silently
command -v idea >/dev/null 2>&1 && idea skill
command -v rk   >/dev/null 2>&1 && rk skill
command -v hop  >/dev/null 2>&1 && hop skill
```

Per the shll toolkit `skill` standard, `<tool> skill` prints a static, ≤150-line,
agent-optimized usage briefing as raw markdown to stdout (exit 0, stderr empty),
byte-stable per release.

**Version-skew fallback (required).** An installed tool may predate its `skill`
subcommand. The invocation MUST **capability-probe** it — `<tool> skill` failing
(non-zero exit, or no output) is the probe — and fall back **silently** to the
shll.ai bundle-page pointer `https://shll.ai/<tool>/skill`. This composes with
the `command -v` gate: **absent** binary → skip entirely; **present but old** →
the fallback pointer.

### The `help-dump` contract

For any specific flag or subcommand, run the gated `<tool> help-dump` (or
`<tool> <cmd> --help`) and treat that output as authoritative. The JSON envelope
contains `tool`, `version`, `schema_version`, and a recursive `root` command tree;
it does not contain `captured_at`.

```json
{
  "tool": "idea",
  "version": "v0.0.13",
  "schema_version": 1,
  "root": {
    "name": "idea",
    "path": "idea",
    "short": "Backlog idea management (current worktree; use --main for main worktree)",
    "usage": "idea [flags]",
    "text": "...full help text...",
    "commands": [
      { "name": "add", "path": "idea add", "short": "...", "usage": "idea add <text> [flags]", "text": "...", "commands": [] }
    ]
  }
}
```

### Functional entry points

The fail-silent rule governs informational delegations. `wt` remains
**functionally required** for `fab batch new`/`switch` (an upfront
`exec.LookPath("wt")` guard in the binary). Those entry points do NOT silently
skip: they stop with an actionable install hint
(`… install it via: brew install sahil87/tap/wt`).

---

## wt (Worktree Manager)

`wt` manages git worktrees for parallel development. Install it with
`brew install sahil87/tap/wt`; required worktree entry points stop with that hint
when it is absent (per § Functional entry points).

`wt`'s command set (`list`/`create`/`delete`/…), the `wt create` flags
(`--non-interactive`/`--name`/`--reuse`/`--base`/`--checkout` + the positional
`[branch]`), and its branch-selection contract (positional is new-branch-only,
exit 2 on an existing branch; `--checkout <branch>` is the existing-branch opt-in
and conflicts with both `--base` and the positional) are **tool-owned** — read
them at use-time via `wt skill` (usage) / `wt help-dump` (flags), gated per §
Delegation and binary gate.

---

## idea (Backlog Manager)

Standalone binary for backlog idea management — CRUD for `fab/backlog.md` (the
inbox that feeds `/fab-new <id>`). Install it with
`brew install sahil87/tap/idea`; `/fab-new <id>` can resolve backlog IDs directly
when it is absent.

`idea`'s verbs, flags, matching rules, and backlog line format are tool-owned;
delegate per § Delegation and binary gate.

---

## hop (Multi-Repo Navigator)

`hop` is the **repo locator** for the same space `wt` operates on: `wt`
enumerates worktrees within a repo, while `hop` enumerates registered repos.
Its discovery grammar is tool-owned; delegate per § Delegation and binary gate.

---

## tmux

Terminal multiplexer commands used for agent observation and interaction.

### Commands

| Command | Usage | Purpose |
|---------|-------|---------|
| `new-window` | `tmux new-window [-t '<session>:'] -n <name> -c <dir> "<cmd>"` | Open a new tmux tab with a command running in a specific directory. Without `-t` the window lands in the **ambient** session — the session of the pane running the command |

### Usage Notes

- **Pane mapping across sessions**: `fab pane map --all-sessions --json` is the on-demand surface. Rows carry a per-row `repo` field (the pane's absolute main-worktree root, `null` when unresolved).
- **Pane capture**: Use `rk mux capture` (substrate-enriched capture — last-N tail, `--raw`/`--json`, reconciled agent state). Usage ownership is in `_cli-agents.md` § Peek. (`fab pane capture` is dispatch-internal — kept for the rk-less pane arm; see `_cli-fab-pane.md` § fab pane.)
- **Send keys**: Use `rk mux send` (it carries built-in pane-existence and agent-state validation with probe-verified delivery). Usage ownership is in `_cli-agents.md` § Pre-Send Validation.
- **Agent-session spawning** rides `rk tab new` (argv tokens after `--`, the rk-appended shell fallback, the `--ready` verdict) — the fab-owned usage is owned by `_cli-agents.md` § Spawn Composition ("Open it in a pane"). The rk verbs' contracts are tool-owned (`rk skill`).

---

## rk (HexoKit)

HexoKit is the tmux session manager with a web UI that may host agent sessions.
The Homebrew formula and primary binary are `run-kit`
(`sahil87/tap/run-kit`); `rk` is the invocation used throughout fab skills.

`rk`'s command surface is tool-owned; delegate per § Delegation and binary gate.
This includes `rk notify`, `rk context`, iframe windows, proxy URLs, and the
visual display recipe.

The **dynamic** environment (current server URL, session, pane) stays in `rk context` — run at use-time, never hardcoded. `rk skill` is the static usage briefing; `rk context` reports the live environment (the two are distinct per `shll standards skill`).

### Pipeline escalation send (fab-owned)

A pipeline's non-blocking escalation uses `rk notify` as its out-of-band
notification send, relying on HexoKit's fail-silent-by-contract guarantee:

```sh
rk notify "{change}: {summary} ({repo})" --title "Fab: strategic question"
```

This is fab's *usage* of the tool, not the `rk notify` contract itself (that is
tool-owned — see `rk skill`).

### Agent messaging (fab-owned — pointer)

Agent messaging via `rk mux send`/`rk mux await` is owned by `_cli-agents.md` §
Pre-Send Validation / § Await. The verbs' full contract (gate matrix,
probe-verified delivery, report words) is tool-owned; see `rk skill`.

### Pane peek/kill/process (fab-owned — pointer)

Pane peek via `rk mux capture`, pane removal via the agent-state-gated `rk mux
kill`, and process-tree inspection via `rk mux process` are owned by
`_cli-agents.md` § Peek. The verbs' full contracts are tool-owned; see `rk skill`.
fab's own `fab pane capture`/`kill`/`process` remain dispatch-internal for the
rk-less pane arm (`_cli-fab-pane.md` § fab pane).
