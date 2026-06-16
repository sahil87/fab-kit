---
name: _cli-external
description: "External CLI tool reference — wt (worktree manager), idea (backlog manager), tmux, rk (run-kit: context/iframe/proxy/visual-display + notify), and /loop. Loaded by operator skills only."
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
| `--reuse` | Reuse an existing worktree if one matches — requires `--worktree-name <name>` (the match key). Useful for autopilot respawns |
| `--base <ref>` | Branch from a specific ref instead of the default. Used for sequenced autopilot (branch from prior change) |
| `[branch]` | Positional — the git branch to create/checkout in the worktree |

**Example — known change**: `wt create --non-interactive --worktree-name <name> <change-folder-name>`
**Example — autopilot respawn**: `wt create --non-interactive --reuse --worktree-name <name> <branch> --base <prev-change>`

> **Repo-targeted spawning (operator).** `wt` operates on the **current working directory's** repo. For multi-repo coordination, the operator MUST run `wt create` **in the target repo's directory** (the agent's absolute main-worktree root), so the new worktree lands under `$(dirname <target-repo>)/<repo-name>.worktrees/` — not under the operator's own repo. The operator reads that target repo's spawn command separately via `fab spawn-command --repo <target-repo>` (see `_cli-fab.md`), never its own `config.yaml`.

### Operator Spawning Rules

When the operator creates a worktree for an agent, the naming strategy depends on whether the change already exists:

#### Known change (already exists)

Use the change folder name as the branch argument to `wt create`:

```
wt create --non-interactive --worktree-name <name> <change-folder-name>
```

The worktree gets a random name; the branch matches the change. No `/git-branch` needed.

#### New change (from backlog)

The change folder doesn't exist yet, so there's no branch name to use:

1. `wt create --non-interactive` — auto-generates worktree name, creates on default branch
2. Agent runs `/fab-new` to create the change folder — its Step 11 then renames the worktree's disposable branch to the change folder name inline (the rename guard passes: the `wt create` branch resolves to no change)
3. No operator action needed — the branch already matches the change; the operator does NOT send `/git-branch` (the former post-intake send predates fab-new's inline branch creation)

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

- **Pane mapping across sessions**: The operator's tick snapshots **all** sessions on its tmux server via `fab pane map --all-sessions --json` (see `_cli-fab.md`), not just the operator's own session. The `--json` output carries a per-row `repo` field (the pane's absolute main-worktree root, `null` when unresolved) used to group the status frame by repo then session.
- **Pane capture**: Use `fab pane capture` instead of raw `tmux capture-pane`. It provides fab context enrichment, validation, and structured output.
- **Send keys**: Use `fab pane send` instead of raw `tmux send-keys`. It includes built-in pane existence and agent idle validation.
- **`new-window`** is used for spawning new agent sessions: `tmux new-window -n "»<wt>" -c <worktree> "$SPAWN_CMD '<command>'"` where `<wt>` is the worktree name and `$SPAWN_CMD` is the target repo's spawn command (see the repo-targeted spawning note in the wt section above)

---

## rk (run-kit)

run-kit is the tmux session manager with a web UI that hosts the operator's session. All commands below are subject to the **detection / fail-silent rule** stated once in `_preamble.md` § Run-Kit (rk) Reference — check `command -v rk` first and skip silently when rk is absent (never error, never warn). This section is the full body the preamble points to.

### Notifications

`rk notify` sends a Web Push notification via the local run-kit server to every subscribed browser/device:

```sh
rk notify <message> [--title string]
```

- **Fail-silent by contract**: on any error (server unreachable, no subscriptions, bad request) `rk notify` exits 0 and prints nothing, so it never stalls a calling loop. This is run-kit's own guarantee — it composes with the preamble's detection rule for an end-to-end silent send.
- **Operator default channel**: the operator's non-blocking Strategic escalation (`fab-operator.md` §5) uses `rk notify` as its default out-of-band notification send, gated on `command -v rk`:

  ```sh
  command -v rk >/dev/null 2>&1 && rk notify "{change}: {summary} ({repo})" --title "Operator: strategic question"
  ```
- **Delivery model**: a real background mobile/desktop Web Push (run-kit holds the VAPID keypair and the device subscriptions). One user's subscriptions form a single feed across every operator on the box. `rk notify` itself reports nothing; the underlying `POST /api/notify` returns a `{"sent":N,"pruned":M}` summary if a caller needs delivery visibility (the operator does not — it relies on the fail-silent contract).

### Server URL Discovery

Discover the server URL at **use-time** by running:

```sh
rk context 2>/dev/null | grep 'Server URL' | awk '{print $NF}'
```

Never hardcode the server URL — it can change between sessions.

### Iframe Windows

Create a tmux window that displays a web page instead of a terminal:

```sh
tmux new-window -n <name>
tmux set-option -w @rk_type iframe
tmux set-option -w @rk_url <url>
```

Change the URL of an existing iframe window:

```sh
tmux set-option -w @rk_url <new-url>
```

The rk server detects `@rk_type` and `@rk_url` changes automatically via SSE polling — no manual refresh needed.

### Proxy

Access local services through the rk server using the proxy URL pattern:

```
{server_url}/proxy/{port}/...
```

For example, a service on port 8080 is available at `{server_url}/proxy/8080/`.

### Visual Display Recipe

The canonical recipe for displaying HTML content in an iframe window is documented by `rk context` — run-kit owns this workflow because every step (loopback HTTP server, relative `/proxy/<port>/...` path, `@rk_type`/`@rk_url` tmux options) is run-kit-specific. Keeping the recipe in one place eliminates drift between fab-kit and run-kit.

At use-time, call `rk context` and read the `### Visual Display Recipe` subsection of the output for the current 4-step flow (generate HTML → loopback HTTP server → iframe window with relative `@rk_url` → fail silently). Any step SHALL fail silently if its prerequisite is unavailable (rk missing, port in use, server start fails) — skip remaining steps without surfacing an error.

#### Visual-Explainer Integration

When the `visual-explainer` plugin is available, skills MAY delegate HTML generation to it (Step 1 of the `rk context` recipe), then follow the remaining steps to display the result. If `visual-explainer` is not available, skip the visual display entirely — no error, no fallback.

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
