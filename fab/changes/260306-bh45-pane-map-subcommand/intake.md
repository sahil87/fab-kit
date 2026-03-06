# Intake: Add `fab pane-map` Subcommand

**Change**: 260306-bh45-pane-map-subcommand
**Created**: 2026-03-06
**Status**: Draft

## Origin

> Discussed during `/fab-discuss` session exploring a `fab-conductor` skill (multi-agent coordination via tmux). The pane map emerged as a standalone primitive — useful independently of the conductor for debugging, manual tab management, and scripting.

User requested it be created as its own change, separate from the conductor spec.

## Why

1. **No unified view exists** — today you need three separate commands to understand the full picture: `fab status show --all` (change state), `fab runtime` (agent idle/active), and manual `tmux list-panes` inspection (which pane runs which change). There's no single command that connects these.

2. **Without it** — users managing multiple parallel agents in tmux tabs must mentally map pane numbers to worktrees to changes. This is error-prone during the assembly-line workflow with 5+ concurrent agents.

3. **Why a CLI subcommand** — the Go binary already has `fab status`, `fab runtime`, and change resolution. Adding `pane-map` keeps the observation layer in one place. The conductor skill (future) will consume this output rather than reimplementing the discovery logic.

## What Changes

### New subcommand: `fab pane-map`

A new `fab pane-map` subcommand in the Go binary that produces a table combining four data sources:

#### Data sources

1. **tmux pane introspection** — `tmux list-panes -a -F '#{pane_id} #{pane_current_path}'` to get all panes and their CWDs
2. **Worktree resolution** — for each pane CWD, `git -C <path> rev-parse --show-toplevel` to identify worktree roots, then check for `fab/current` to find the active change
3. **`fab runtime`** — read `.fab-runtime.yaml` to determine agent idle state per change (idle_since timestamp → "idle (Nm)" or "active")
4. **`fab status show --all`** — read `.status.yaml` files for stage and progress

#### Output format

```
Pane   Worktree                       Change                              Stage     Agent
%3     .worktrees/alpha/              260306-r3m7-add-retry-logic         apply     active
%7     .worktrees/bravo/              260306-k8ds-ship-wt-binary          review    idle (2m)
%12    .worktrees/charlie/            260306-ab12-refactor-auth           hydrate   idle (8m)
```

Key behaviors:
- Worktree paths are displayed relative to the repo parent (e.g., `.worktrees/alpha/` not full absolute path)
- Idle duration is shown in parentheses: seconds, minutes, or hours as appropriate (e.g., `idle (30s)`, `idle (5m)`, `idle (2h)`)
- Panes that are worktrees but have no active change show `(no change)` in the Change column
- Non-fab panes (e.g., plain shell tabs) are excluded from the output

#### Error handling

- If not inside a tmux session (`$TMUX` unset), print a clear error and exit non-zero
- If `.fab-runtime.yaml` is missing, treat all agents as state unknown (show `?` in Agent column)
- If a worktree's `fab/current` is empty or missing, show `(no change)` for that pane

#### Flags (future consideration)

- `--json` — machine-readable output for scripting and conductor consumption
- `--watch` — refresh on interval (useful as a dashboard)

These flags are noted for design awareness but are NOT required for the initial implementation.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the new `pane-map` subcommand alongside existing `status`, `runtime`, `change` commands

## Impact

- **Go binary** (`src/`): New subcommand registration, tmux interaction logic, worktree resolution, output formatting
- **External dependencies**: Requires `tmux` at runtime (graceful error if not in tmux session), `git` for worktree resolution
- **No impact on existing commands** — purely additive

## Open Questions

- ~~Should the main repo worktree (not a `.worktrees/` child) also appear in the map if it has an active change?~~ **Resolved**: Yes — include it, shown as `(main)` or `.` in the Worktree column.
- ~~Should pane-map detect Claude Code sessions specifically, or any process in a worktree pane?~~ **Resolved**: No process inspection — Agent column is derived entirely from `.fab-runtime.yaml` (set/cleared by hooks). No runtime entry → `?`.

## Clarifications

### Session 2026-03-06 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 4 | Confirmed | — |
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |

### Session 2026-03-06 (suggest)

| # | Question | Answer |
|---|----------|--------|
| 1 | Main repo worktree inclusion? | Yes — show as `(main)` or `.` |
| 2 | Agent column: detect processes or use runtime file? | Runtime file only, no process inspection |
| 3 | Orphan change discovery scope? | Skip orphan discovery — only show tmux panes |
| 4 | Affected Memory: keep schemas entry? | Remove — pane-map only reads, doesn't extend schema |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Subcommand lives in Go binary as `fab pane-map` | Discussed — consistent with existing `fab status`, `fab runtime` subcommands | S:95 R:90 A:95 D:95 |
| 2 | Certain | Output is a formatted table with Pane, Worktree, Change, Stage, Agent columns | Discussed — specific format agreed upon in conversation | S:90 R:85 A:90 D:95 |
| 3 | Certain | No orphan discovery — only show panes that exist in tmux | Clarified — user chose to skip orphan discovery entirely | S:95 R:90 A:90 D:95 |
| 4 | Certain | Worktree paths shown relative to repo parent, not absolute | Clarified — user confirmed | S:95 R:85 A:80 D:80 |
| 5 | Certain | Idle duration shown as human-readable relative time | Clarified — user confirmed | S:95 R:90 A:85 D:80 |
| 6 | Certain | Non-fab panes excluded from output | Clarified — user confirmed | S:95 R:85 A:75 D:70 |
| 7 | Certain | tmux required at runtime — graceful error if not in tmux | Clarified — user confirmed | S:95 R:90 A:70 D:85 |
| 8 | Tentative | `--json` and `--watch` flags deferred to future iteration | Reasonable for v1 scope — but conductor may need `--json` sooner than expected | S:60 R:85 A:65 D:60 |
<!-- assumed: JSON and watch flags deferred — v1 focuses on human-readable table output, conductor can parse text initially -->

8 assumptions (7 certain, 0 confident, 1 tentative, 0 unresolved).
