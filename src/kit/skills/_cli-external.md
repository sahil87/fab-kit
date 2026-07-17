---
name: _cli-external
description: "External CLI tool reference — wt (worktree manager), idea (backlog manager), hop (multi-repo navigator), tmux, rk (run-kit: context/iframe/proxy/visual-display + notify), and /loop. Hand-authored gist (operator-critical commands/flags + integration semantics) per tool; the exhaustive command/flag surface is delegated to each tool's `help-dump` at use-time. Loaded by operator skills only."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# External CLI Tool Reference

> Loaded by operator skills only (not part of the always-load layer). Documents non-fab CLI tools used for multi-agent coordination.

## Contents

- Reference Model
- wt (Worktree Manager)
- idea (Backlog Manager)
- hop (Multi-Repo Navigator)
- tmux
- rk (run-kit)
- /loop

---

## Reference Model

This file documents a hand-authored **gist** per tool — what each tool *is*, the
commands and flags the operator's correctness depends on, and the integration
semantics that tie the tools to fab. It is deliberately **not** an exhaustive
command reference: the full command/flag surface is delegated to each tool's
`help-dump` (below), so this file never goes stale against a tool's release cadence.

### The `help-dump` contract

Each owned tool — `wt`, `idea`, `rk`, `hop` (and `fab` itself) — exposes a **hidden
but stable** `help-dump` subcommand that emits its entire command tree as JSON. It
is not listed in `--help`, but the shape is stable across all of them:

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

Fields: `tool`, `version`, `schema_version` (currently `1`), and a recursive
`root` → `commands[]` tree where each node carries `name`, `path`, `short`,
`usage`, and `text`. The tree is recursive — `commands[]` nests for subcommand
groups. Per the shll v0.0.23 help-dump standard the envelope carries **no
`captured_at`**: the capture timestamp is owned by shll.ai's puller (a tool
cannot know its own capture time — it is stamped after capture), so emitting it
is forbidden toolkit-wide. `fab` and `wt` already omit it; any peer tool still
emitting `captured_at` (empty or otherwise) drops it on its own release cadence.

**Use it at use-time.** For any flag or subcommand **not** in the gist below, run
`<tool> help-dump` (or `<tool> <cmd> --help`) and treat *that*, not this file, as
authoritative for the exhaustive surface. The inlined gist tables are a deliberately
curated subset (the operator-critical commands/flags + integration semantics).

> The `help-dump` delegation scopes to the **four owned binaries** (`wt`, `idea`,
> `rk`, `hop`). `tmux` is third-party and has no `help-dump`; `/loop` is a Claude
> Code skill, not a binary — neither is covered by the instruction above.

### Absent-binary discipline (two install classes)

The four owned binaries fall into two classes by install guarantee, and the
fail-silent rule applies **asymmetrically**:

- **Assumed-present — `wt`, `idea`.** These are Homebrew `depends_on` of `fab-kit`
  (they land together via `brew install fab-kit`). Invoke them **bare**; no
  `command -v` gate is required.
- **Genuinely-optional — `rk`, `hop`.** Each is a separate sibling formula the user
  may or may not have installed (`rk` is run-kit — formula `sahil87/tap/run-kit`
  since run-kit v3.0.0, with `rk` kept as a symlink alias; `hop` is the multi-repo
  navigator). **Every `rk`/`hop` invocation — including `help-dump` — MUST be
  `command -v`-gated and fail silently** (never surface `command not found` or any
  error/warning when the tool is absent). Do NOT generalize this gate to `wt`/`idea`.

```sh
command -v hop >/dev/null 2>&1 && hop help-dump   # rk/hop: gated, fail silently
command -v rk  >/dev/null 2>&1 && rk help-dump    # rk/hop: gated, fail silently
wt help-dump                                       # wt/idea: assumed present, bare
idea help-dump                                     # wt/idea: assumed present, bare
```

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
| `--base <ref>` | Branch from a specific ref instead of the default. Used for sequenced autopilot (branch from prior change). Conflicts with `--checkout` (both exit 2) |
| `--checkout <branch>` | Put the worktree on an EXISTING local/remote branch (fetches remote-only branches). Conflicts with `--base` and with the positional (both exit 2) |
| `[branch]` | Positional — name for a NEW branch only; exits 2 if the branch already exists locally or remotely (use `--checkout` for an existing branch) |

**Example — known change** (the change branch may already exist — created by `/fab-new` Step 11 in the original checkout — so probe and route): branch exists → `wt create --non-interactive --worktree-name <name> --checkout <change-folder-name>`; missing → `wt create --non-interactive --worktree-name <name> <change-folder-name>`.
**Example — autopilot respawn**: branch exists → `wt create --non-interactive --reuse --worktree-name <name> --checkout <branch>`; missing → `wt create --non-interactive --reuse --worktree-name <name> <branch> --base <prev-change>`. (`--base` rides only the positional/new-branch arm — `--checkout`+`--base` is a hard exit-2 conflict, and an existing branch already embodies its start-point.)

> The gist above is the operator-used subset. The full `wt` surface (e.g. `init`, `open`, `shell-init`, `update`) and the complete flag set for each command are available via `wt help-dump` (assumed present — bare, per § Reference Model).

> **Repo-targeted spawning (operator).** `wt` operates on the **current working directory's** repo. For multi-repo coordination, the operator MUST run `wt create` **in the target repo's directory** (the agent's absolute main-worktree root), so the new worktree lands under `$(dirname <target-repo>)/<repo-name>.worktrees/` — not under the operator's own repo. The operator reads that target repo's session command separately via `fab agent --print --repo <target-repo>` (see `_cli-fab.md`), never its own `config.yaml`.

### Operator Spawning Rules

When the operator creates a worktree for an agent, the naming strategy depends on whether the change already exists:

#### Known change (already exists)

The change's branch usually already exists (created by `/fab-new` Step 11 in the original checkout), so **probe branch existence and route** per wt's new-branch-only positional contract — passing an existing branch positionally exits 2. Probe local first (`git show-ref --verify --quiet refs/heads/<change-folder-name>`), then remote (`git ls-remote --heads origin <change-folder-name>`):

```
# branch exists (the common case) → put the worktree ON the existing branch
wt create --non-interactive --worktree-name <name> --checkout <change-folder-name>

# branch missing → create it (new-branch positional)
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

> The verbs above are the operator-used subset. The full `idea` surface (e.g. `fmt`, `prune`, `shell-init`, `update`) is available via `idea help-dump` (assumed present — bare, per § Reference Model).

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

## hop (Multi-Repo Navigator)

`hop` is a **genuinely-optional** binary — a separate sibling formula, not a `fab-kit` Homebrew dependency, so it can legitimately be absent. Every `hop` invocation (including `hop help-dump`) MUST be `command -v hop`-gated and skip silently when absent (per § Reference Model — never surface `command not found` or any error/warning). This mirrors the `rk` fail-silent discipline.

```sh
command -v hop >/dev/null 2>&1 && hop ls   # gate every hop call, fail silently
```

**What it is.** `hop` is the **discovery front-end to the same repo/worktree space `wt` operates on**: it locates, opens, and operates on repos registered in a `hop.yaml` registry (default `~/.config/hop/hop.yaml`). The grammar is `hop <selection> <action>` — selection is a repo name (substring → fzf on ambiguity), a `repo/worktree`, a group, or `--all`; action is a builtin verb (`cd`/`open`/`where`), a batch verb (`pull`/`push`/`sync`), or any PATH binary. Where `wt` enumerates the worktrees *within* a repo, `hop` enumerates the *repos* themselves.

### Repo/worktree discovery (the operator-relevant subset)

| Command | Usage | Purpose |
|---------|-------|---------|
| `ls` | `hop ls` | List all registered repos as aligned name/path columns — the most useful command for discovering where sibling repos live on disk (`--json` for machine-readable output) |
| `ls --trees` | `hop ls --trees` | List repos **with worktree summaries**, fanning out to `wt list --json` per repo. This is the explicit `hop`↔`wt` seam: `hop` enumerates repos, `wt` enumerates each repo's worktrees |
| `where` | `hop <name> where` | Echo the absolute path of a matching repo. `hop <name>/<wt> where` resolves a specific worktree (via `wt list --json`) |

**Why it matters to the operator.** Multi-repo coordination needs the absolute main-worktree root of a *sibling* repo — e.g. to spawn an agent into it (see the **Repo-targeted spawning** note in the `wt` section, which requires running `wt create` in the target repo's directory and reading `fab agent --print --repo <target-repo>`). `hop ls` / `hop <name> where` is how an agent **discovers** those locations rather than hardcoding paths.

**Full surface.** The rest of `hop` — `add`, `clone`, `rm`, `config` (`init`/`where`/`print`), `shell-init`, `update`, the batch verbs (`pull`/`push`/`sync`), and `--all` fan-out — is available via `command -v hop >/dev/null 2>&1 && hop help-dump`. The gist above covers only discovery.

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
- **`new-window`** is used for spawning new agent sessions: `tmux new-window -n "»<wt>" -c <worktree> "$SPAWN_CMD '<command>'"` where `<wt>` is the worktree name and `$SPAWN_CMD` is the target repo's session command (see the repo-targeted spawning note in the wt section above)

---

## rk (run-kit)

run-kit is the tmux session manager with a web UI that hosts the operator's session. Since run-kit v3.0.0 the Homebrew formula and primary binary are named `run-kit` (`sahil87/tap/run-kit`); `rk` is kept as a symlink alias and remains the invocation form used throughout fab skills. All commands below are subject to the **detection / fail-silent rule** stated once in `_preamble.md` § Run-Kit (rk) Reference — check `command -v rk` first and skip silently when rk is absent (never error, never warn). This section is the full operator-facing body the preamble points to; the exhaustive command surface is delegated to `rk help-dump` (see below).

The gist below is the operator-used subset. The full `rk` surface (`daemon`, `doctor`, `serve`, `reaper`, `riff`, `init-conf`, `status`, `update`, …) is available via `command -v rk >/dev/null 2>&1 && rk help-dump` — gated and fail-silent like every other `rk` invocation (per § Reference Model and the `_preamble.md` rule above).

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
