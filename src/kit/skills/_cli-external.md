---
name: _cli-external
description: "External CLI tool reference — wt (worktree manager), idea (backlog manager), tmux, and /loop. Loaded by operator skills only."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# External CLI Tool Reference

> Loaded by operator skills only (not part of the always-load layer). Documents non-fab CLI tools used for multi-agent coordination.

---

## wt (Worktree Manager)

`wt` manages git worktrees for parallel development. Installed system-wide via `brew install fab-kit`.

### Commands

| Command | Usage | Purpose |
|---------|-------|---------|
| `list` | `wt list` | List all worktrees: names, branches, paths |
| `list --path` | `wt list --path <name>` | Check if a worktree exists. Exit 0 = exists (prints path), exit 1 = not found |
| `create` | `wt create --non-interactive [flags] [branch]` | Create a new worktree (see flags below) |
| `delete` | `wt delete <name>` | Delete a worktree. Destructive — confirm first |

### `wt create` Flags

| Flag | Purpose |
|------|---------|
| `--non-interactive` | Required for operator use — suppresses prompts |
| `--worktree-name <name>` | Override the auto-generated worktree directory name |
| `--reuse` | Reuse an existing worktree if one matches. Useful for autopilot respawns |
| `--base <ref>` | Branch from a specific ref instead of the default. Used for sequenced autopilot (branch from prior change) |
| `[branch]` | Positional — the git branch to create/checkout in the worktree |

**Example — known change**: `wt create --non-interactive --worktree-name <name> <change-folder-name>`
**Example — autopilot respawn**: `wt create --non-interactive --reuse --worktree-name <name> <branch> --base <prev-change>`

---

## idea (Backlog Manager)

Standalone binary for backlog idea management — CRUD for `fab/backlog.md`. Installed system-wide via `brew install fab-kit` (not a `fab` subcommand).

```
idea <subcommand> [flags...]
```

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| *(bare)* | `idea <text>` | Shorthand for `idea add <text>` (no `--id`/`--date` flags) |
| `add` | `add <text> [--id <4char>] [--date <YYYY-MM-DD>]` | Add a new idea |
| `list` | `list [-a] [--done] [--json] [--sort <id\|date>] [--reverse]` | List ideas |
| `show` | `show <query> [--json]` | Show a single idea |
| `done` | `done <query>` | Mark an idea as done |
| `reopen` | `reopen <query>` | Reopen a completed idea |
| `edit` | `edit <query> <new-text> [--id <4char>] [--date <YYYY-MM-DD>]` | Modify an idea |
| `rm` | `rm <query> --force` | Delete an idea (requires --force) |

### Persistent Flags

| Flag | Purpose |
|------|---------|
| `--file <path>` | Override backlog file path (relative to git root). Also respects `IDEAS_FILE` env var. Priority: `--file` > `IDEAS_FILE` > default `fab/backlog.md` |
| `--main` | Operate on the main worktree's backlog instead of the current worktree |

By default, `idea` operates on the **current worktree's** `fab/backlog.md` (resolved via `git rev-parse --show-toplevel`). Pass `--main` to target the main worktree's backlog instead (resolved by running `git rev-parse --path-format=absolute --git-common-dir` and taking its parent directory as the main worktree root). In the main worktree, both behave identically.

**Query matching**: Case-insensitive substring match on both the idea ID and text fields. Commands that modify a single idea (`show`, `done`, `reopen`, `edit`, `rm`) require exactly one match; zero matches returns "No idea matching", multiple matches returns disambiguation guidance.

**Backlog format**:

```
- [ ] [a7k2] 2025-06-15: Add dark mode to settings page
- [ ] [c3d4] 2025-06-10: DES-123 Link to a Linear ticket
- [x] [e5f6] 2025-06-08: Fix login redirect bug
```

**Output format**:
- Add: `Added: [{id}] {date}: {text}`
- Done: `Done: - [x] [{id}] {date}: {text}`
- Reopen: `Reopened: - [ ] [{id}] {date}: {text}`
- Edit: `Updated: - [{status}] [{id}] {date}: {text}`
- Rm: `Removed: - [{status}] [{id}] {date}: {text}`

---

## tmux

Terminal multiplexer commands used by the operator for agent observation and interaction.

### Commands

| Command | Usage | Purpose |
|---------|-------|---------|
| `new-window` | `tmux new-window -n <name> -c <dir> "<cmd>"` | Open a new tmux tab with a command running in a specific directory |

### Usage Notes

- **Pane capture**: Use `fab pane capture` instead of raw `tmux capture-pane`. It provides fab context enrichment, validation, and structured output.
- **Send keys**: Use `fab pane send` instead of raw `tmux send-keys`. It includes built-in pane existence and agent idle validation.
- **`new-window`** is used for spawning new agent sessions: `tmux new-window -n "fab-<id>" -c <worktree> "$SPAWN_CMD '<command>'"` where `$SPAWN_CMD` is read from `config.yaml` `agent.spawn_command` (see `lib/spawn.sh`)

---

## /loop

Recurring check skill — invokes a prompt at a regular interval.

### Usage

```
/loop <interval> "<prompt>"
```

- **`<interval>`** — duration between ticks (e.g., `5m`, `2m`)
- **`<prompt>`** — the instruction to execute on each tick

### Constraints

- **One loop at a time** — there SHALL be at most one active `/loop` in a session
- **Start**: when the first change is enrolled in monitoring and no loop is running
- **Stop**: when the monitored set becomes empty, or on explicit user command
- **Autopilot override**: autopilot uses its own cadence (default 2m); replaces any existing monitoring loop
