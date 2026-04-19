# Pane Commands

**Domain**: fab-workflow

## Overview

`fab pane` is the parent command grouping four tmux-pane operations: `map`, `capture`, `send`, and `process`. All four subcommands shell out to `tmux` to query or manipulate panes, combining raw tmux output with fab-specific enrichment (worktree, change, stage, agent state resolved from per-pane CWD).

The command group is special in the router: `pane` is the sole entry in the `fab` router's `fabGoNoConfigArgs` allowlist, meaning it runs from any directory — including outside a fab-managed repo (scratch tmux tabs, cross-repo orchestration, non-fab daemons). See `kit-architecture.md` for the router-level exemption.

This doc covers the four subcommands, the `--server` / `-L` persistent flag, and the semantic invariants that govern how pane IDs and server selection interact with tmux's own socket model.

## Requirements

### Parent Command: `fab pane`

`fab pane` is a cobra command group with four subcommands (`map`, `capture`, `send`, `process`) and one persistent flag (`--server` / `-L`). Invoking `fab pane` with no subcommand prints the standard cobra help listing the four subcommands. Source: `src/go/fab/cmd/fab/pane.go`.

### Subcommand: `fab pane map`

`fab pane map [--json] [--session <name>] [--all-sessions]` combines tmux pane introspection with worktree/change/runtime state into a unified view. Source: `src/go/fab/cmd/fab/panemap.go`.

**Flags**:

| Flag | Type | Purpose |
|------|------|---------|
| `--json` | bool | Output as JSON array instead of aligned table |
| `--session <name>` | string | Target a specific tmux session by name (skips `$TMUX` check) |
| `--all-sessions` | bool | Query all tmux sessions (skips `$TMUX` check) |
| `--server <name>` | string | Persistent flag — see §`--server` flag below |

`--session` and `--all-sessions` are mutually exclusive. When neither is set, discovery runs against the current tmux session only (`tmux list-panes -s`) and requires `$TMUX` to be set.

**Table columns**: `Session` (only with `--all-sessions`), `Pane`, `WinIdx`, `Tab`, `Worktree`, `Change`, `Stage`, `Agent`. The `Worktree` column displays `(main)` for the main worktree, a relative path from the main repo's parent for other git worktrees, or `basename/` for non-git panes. Non-fab panes render em-dash fallbacks for `Change`, `Stage`, `Agent`.

**JSON fields** (snake_case): `session`, `window_index`, `pane`, `tab`, `worktree`, `change`, `stage`, `agent_state`, `agent_idle_duration`. `change` and `stage` are `null` when no active change exists on the pane's worktree. `agent_state` and `agent_idle_duration` populate whenever an `_agents` entry matches the pane — independent of `change` / `stage`.

**Three-axis model**: The map resolves three orthogonal axes independently — **Change** (from `.fab-status.yaml`), **Agent** (from `_agents` in `.fab-runtime.yaml`), and **Process** (opt-in via `fab pane process`, not in `map` output). See [runtime-agents.md](runtime-agents.md) for the full model.

**Agent state resolution**: The Agent column is resolved by scanning `_agents` in the pane's worktree's `.fab-runtime.yaml` for an entry whose `tmux_pane` equals the pane ID AND whose `tmux_server` is either empty or matches the current server. A matched entry without `idle_since` renders as `active`; with `idle_since` renders as `idle (<duration>)` (e.g., `idle (2m)`). No match renders as `—`. This resolution is **independent of whether the pane has an active change** — agents running in discussion mode populate the Agent column just like change-associated agents. See [runtime-agents.md](runtime-agents.md) for the matching rule and schema details.

**Display scenarios**:

| Scenario | Change | Stage | Agent |
|----------|--------|-------|-------|
| Change active, agent idle | `260417-...` | `spec` | `idle (2m)` |
| Change active, agent active | `260417-...` | `spec` | `active` |
| Discussion mode (fab worktree), agent idle | `(no change)` | `—` | `idle (2m)` |
| Discussion mode (fab worktree), agent active | `(no change)` | `—` | `active` |
| Change active, no agent matched | `260417-...` | `spec` | `—` |
| Fab worktree, no change, no agent | `(no change)` | `—` | `—` |
| Non-fab pane (no `fab/` dir) | `—` | `—` | `—` |

**Error behavior**: Unset `$TMUX` with neither session flag → `Error: not inside a tmux session` (exit 1). No panes found → `No tmux panes found.` (exit 0).

### Subcommand: `fab pane capture`

`fab pane capture <pane> [-l N] [--json] [--raw]` captures terminal content from a tmux pane with fab context enrichment. Source: `src/go/fab/cmd/fab/pane_capture.go`.

**Flags**: `<pane>` (required tmux pane ID, e.g. `%5`); `-l`/`--lines` (int, default 50); `--json` (structured output with pane metadata); `--raw` (plain captured text, no header, no enrichment). `--json` and `--raw` are mutually exclusive.

**Default output**: Header block (pane ID, worktree, change, stage, agent state) followed by the captured text.

**JSON output**: `pane`, `lines`, `content`, `worktree`, `change`, `stage`, `agent_state`, `agent_idle_duration`. The four fab-context fields are `null` when the pane is not in a fab worktree or has no active change.

**Error behavior**: Pane not found → `Error: pane <id> not found` (exit 1). `--lines < 1` → `Error: --lines must be >= 1` (exit 1).

### Subcommand: `fab pane send`

`fab pane send <pane> <text> [--no-enter] [--force]` sends keystrokes to a tmux pane with built-in pane-existence and agent-idle validation. Source: `src/go/fab/cmd/fab/pane_send.go`.

**Flags**: `<pane>` (required); `<text>` (required); `--no-enter` (don't append Enter); `--force` (skip idle validation — still validates pane existence).

**Validation pipeline**:

1. Pane exists: `tmux list-panes -a`. If not found → exit 1 with `Error: pane <id> not found` (even with `--force`).
2. Agent idle: resolves pane fab context and checks agent state. Rejects `active` or `unknown` states with `Error: agent in pane <id> is not idle (state: <state>)`. `--force` bypasses only this check.
3. Send keys: `tmux send-keys -t <pane> -l <text>` (literal text), optionally followed by a separate `tmux send-keys -t <pane> Enter`.

**Why two send-keys invocations**: The `-l` flag sends `<text>` literally so tmux does not interpret key names like `"Enter"`, `"Space"`, `"C-c"` embedded in the text itself. The trailing Enter keystroke is sent as a separate non-literal command.

**Unknown state**: A pane with no matching `_agents` entry (no `.fab-runtime.yaml`, or no entry whose `tmux_pane` matches this pane) is treated as `unknown` (non-idle). Discussion-mode panes with a live Claude session resolve to `idle`/`active` via `_agents` matching and are accepted without `--force`. Use `--force` to override the idle check for any non-idle state. See [runtime-agents.md](runtime-agents.md) for the matching rule.

### Subcommand: `fab pane process`

`fab pane process <pane> [--json]` detects the process tree running in a tmux pane via OS-level process inspection. Source: `src/go/fab/cmd/fab/pane_process.go` (plus platform-specific `pane_process_linux.go` / `pane_process_darwin.go`).

**Discovery**: Linux reads `/proc/<pid>/task/<tid>/children` recursively; macOS uses `ps -o pid,ppid,comm -ax` with PPID traversal. Platform selection via Go build tags.

**Classification** (based on process comm name): `claude`/`claude-code` → `agent`; `node` → `node`; `git`/`gh` → `git`; all others → `other`.

**Default output**: Tree-formatted process listing with PID, command name, and classification.

**JSON output**: `pane`, `pane_pid`, `processes` (tree of `{pid, ppid, comm, cmdline, classification, children}`), `has_agent` (true if any process classified as `agent`).

Platform-specific process discovery is tmux-server-independent — once the pane's shell PID has been resolved via `GetPanePID`, the `/proc` walk or `ps` traversal operates on the OS process table, not tmux.

### `--server` / `-L` Flag

**Registration**: `paneCmd` registers a persistent string flag `--server` (short `-L`) with default `""`. Because it is a persistent flag on the parent, it is automatically visible on all four subcommands' `--help`. Source: `src/go/fab/cmd/fab/pane.go:14`.

**Help text**: `Target tmux socket label (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket.`

**Behavior**:

- When the flag is **absent or empty**, every `exec.Command("tmux", ...)` invocation in the pane call tree runs with no `-L` argument. Tmux inherits socket selection from `$TMUX` (when set) or falls back to its default socket. This is byte-for-byte identical to pre-flag behavior.
- When the flag is **non-empty**, every `exec.Command("tmux", ...)` invocation in the pane call tree is prepended with `-L <value>`. The flag is passed to tmux verbatim — fab does not inspect, validate, or normalize the server name. Tmux owns the semantics; any error (e.g., `no server running on /tmp/tmux-1001/nonexistent`) is propagated to stderr.

**Short form**: `fab pane map -L runKit` is identical to `fab pane map --server runKit`.

**Motivating use case**: The run-kit daemon runs inside a tmux session named `rk-daemon` (so its `$TMUX` points to one socket) while the user's sessions it is inspecting live on a different socket (`runKit`). Without `--server`, `fab pane map --json --all-sessions` invoked by `rk serve` enumerates panes from the wrong socket — the one in its own `$TMUX` — and every key lookup misses. With `fab pane map --json --all-sessions --server runKit`, every internal tmux invocation runs with `-L runKit` and the correct pane set is returned. More generally, the flag enables any programmatic caller that needs to inspect a tmux server different from the one it inherits.

**Workarounds that don't work** (and why the flag is the right fix): Setting `$TMUX` as a socket selector is incorrect — `$TMUX` means `socket,pid,pane_id`, not a socket path, and some tmux code paths behave differently when `$TMUX` is set (e.g., refusing nested `attach`). `$TMUX_TMPDIR` only helps for default-named sockets in a dedicated tmpdir. Unsetting `$TMUX` and relying on the default socket only works when the target is in fact the default.

### Semantic Invariants

**Pane IDs are per-server.** Tmux allocates pane IDs (e.g., `%3`, `%5`) within each tmux server's own scope. The same `%3` can exist on two different servers and refer to unrelated panes. When `--server <S>` is passed with a pane ID argument, the ID is interpreted in the context of server `<S>`. Callers are responsible for pairing the correct pane ID with the correct server.

**`--server` takes precedence over `$TMUX`.** When both are set, the explicit CLI flag wins. This matches tmux's own behavior — `tmux -L <label>` explicitly selects a socket, overriding any inherited selection.

**Non-tmux operations are unaffected by `--server`.** File reads (`.fab-runtime.yaml`, `.status.yaml`), git-worktree detection (`git rev-parse --show-toplevel`, `git worktree list`), and OS-level process discovery (`/proc` on Linux, `ps` on macOS) key off the pane's CWD or the resolved folder name, not the tmux server. The `--server` value is never used as a filesystem lookup key.

### Shared Pane Package (`internal/pane`)

Shared pane-resolution logic lives in `src/go/fab/internal/pane/pane.go`:

- `ValidatePane(paneID, server string) error` — runs `tmux list-panes -a` and checks for the pane ID
- `GetPanePID(paneID, server string) (int, error)` — resolves shell PID via `tmux display-message`
- `ResolvePaneContext(paneID, mainRoot, server string) (*PaneContext, error)` — resolves worktree, change, stage, and agent state from the pane's CWD
- `FindMainWorktreeRoot(cwds []string) string` — derives the main worktree root from pane CWDs via `git worktree list --porcelain`
- `WithServer(server string, args ...string) []string` — the canonical argv-building helper (see Design Decisions)

All tmux-invoking functions accept a trailing `server string` parameter and build their argv via `WithServer`. Callers in `cmd/fab/pane*.go` read the flag via `cmd.Flags().GetString("server")` and thread the value through.

## Design Decisions

### Persistent Flag on the Parent, Not Per-Subcommand
**Decision**: `--server` is registered as a persistent flag on `paneCmd` via `cmd.PersistentFlags().StringP("server", "L", "", "...")`, visible on all four subcommands. Each subcommand reads the value via `cmd.Flags().GetString("server")`.
**Why**: Cobra idiom for a flag that applies uniformly across a command group. Single registration point, single help-text location, zero chance of per-subcommand drift.
**Rejected**: Per-subcommand registration — four copies of the same flag, four places to update if the description changes.
*Source*: 260417-2fbb-pane-server-flag

### `WithServer` Helper in `internal/pane/pane.go`
**Decision**: A single argv-building helper `WithServer(server string, args ...string) []string` lives in `src/go/fab/internal/pane/pane.go`. It returns `args` unchanged when `server == ""` and `append([]string{"-L", server}, args...)` otherwise. Every `exec.Command("tmux", ...)` site in the pane call tree builds its argv via this helper.
**Why**: `WithServer` is a short pure function that eliminates per-file conditional logic and ensures the `-L` prepend is identical at every call site. Scope is exactly one helper for one flag; introducing an `internal/tmuxutil/` package or a `TmuxClient` struct type would be over-engineering for a single-flag change and can be promoted later if tmux-helper surface grows.
**Exported** (rather than unexported as drafted in the spec): the helper is used from the `cmd/fab` package (e.g., inside `sendTextArgs`, `listPanesArgs`, `capturePaneArgs`) to keep a single canonical argv builder across packages. Cross-package argv builders in this codebase are exported from `internal/pane` when consumed outside the pane package — future tmux-helper additions should follow the same pattern.
*Source*: 260417-2fbb-pane-server-flag

### Helper Named `WithServer`, Not `tmuxArgs`
**Decision**: The helper is named `WithServer`. The pre-existing local variable `tmuxArgs` in `pane_send.go:58` is preserved.
**Why**: `tmuxArgs` was already a local variable name; a free function of the same name would shadow or collide. Renaming the local variable creates churn outside the flag's scope. `WithServer` also reads naturally at call sites: `exec.Command("tmux", WithServer(server, "list-panes", "-a")...)`.
*Source*: 260417-2fbb-pane-server-flag

### Pass the Server Name Verbatim to Tmux
**Decision**: The `--server` value is passed to tmux without fab-side validation, escaping, or normalization.
**Why**: Tmux owns the semantics of socket labels. Any pre-validation in fab (e.g., `tmux -L <server> has-session`) would duplicate tmux's own error handling and introduce race conditions (socket created/destroyed between check and use). Propagating tmux's native error is simpler and more accurate.
**Rejected**: Pre-check via `tmux has-session` — extra subprocess, and fab would still need to handle the real tmux error from the actual command anyway.
*Source*: 260417-2fbb-pane-server-flag

### `-L <name>` Only — No `-S <path>` in First Cut
**Decision**: Only `--server <name>` (maps to `tmux -L <name>`) is exposed. A `--socket-path` / `-S` equivalent is a non-goal for the first cut.
**Why**: `-L` covers the motivating run-kit case and every named-socket scenario. Callers that truly need a full path rather than a label are rare; adding `-S` later is cheap and non-breaking.
**Rejected**: Env-var alternative (`FAB_TMUX_SERVER`) — adds hidden env coupling; CLI flag is more discoverable via `--help` and easier to plumb through subprocess-style callers that already build argv slices.
*Source*: 260417-2fbb-pane-server-flag

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260419-o5ej-agents-runtime-unified | 2026-04-19 | Rewrote the `fab pane map` Agent column resolution: entries are matched by `_agents[*].tmux_pane` in `.fab-runtime.yaml`, independent of active-change state. Discussion-mode panes (no active change) now render `idle (<dur>)` or `active` in the Agent column instead of `—`. Added the three-axis model (Change / Agent / Process) and display-scenario table. `fab pane send` inherits the fix — previously rejected discussion-mode panes as `unknown` now resolve correctly and accept sends when idle. Cross-referenced the new [runtime-agents.md](runtime-agents.md) for the full `.fab-runtime.yaml` schema, hook write pipeline, GC design, and grandparent PID walker. |
| 260417-2fbb-pane-server-flag | 2026-04-17 | Created `pane-commands.md` from the per-subcommand detail previously in `kit-architecture.md`. Added persistent `--server` / `-L` flag on the parent `paneCmd` (default `""`; when non-empty, every internal `exec.Command("tmux", ...)` is prepended with `-L <server>`). New `WithServer(server string, args ...string) []string` helper exported from `internal/pane/pane.go`, consumed by all tmux exec sites across `cmd/fab/pane*.go` and `internal/pane/pane.go`. `ValidatePane`, `GetPanePID`, `ResolvePaneContext`, `discoverPanes`/`discoverSessionPanes`/`discoverAllSessions` all gained a trailing `server string` parameter; test-only argv builders (`listPanesArgs`, `listSessionsArgs`, `capturePaneArgs`, `sendTextArgs`, `sendEnterArgs`) extracted for pure-function argv-capture tests. Motivating use case: run-kit daemon in `rk-daemon` tmux socket inspecting panes on the `runKit` socket. |
