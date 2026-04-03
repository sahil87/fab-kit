# Intake: Fab Pane Command Group

**Change**: 260403-tam1-pane-commands
**Created**: 2026-04-03
**Status**: Draft

## Origin

> Internalize tmux capabilities into the fab Go binary as a new fab pane command group replacing raw tmux calls with fab-aware equivalents. Add fab pane capture (structured pane content capture with fab context enrichment), fab pane send (safe send-keys with built-in pane existence and agent idle validation), and fab pane process (OS-level process state detection via /proc on Linux, ps/lsof on macOS). Rename existing pane-map to fab pane map as a subcommand. Update _cli-fab.md and _cli-external.md skill files accordingly.

One-shot natural language input. No prior discussion.

## Why

Today, operator skills and multi-agent coordination rely on raw `tmux capture-pane`, `tmux send-keys`, and ad-hoc process detection. These raw calls have no fab awareness — they don't know about changes, stages, agent idle state, or worktree context. Every caller must independently validate pane existence, check agent idle state via `.fab-runtime.yaml`, and parse/enrich captured output. This duplicates logic and creates fragile shell-level orchestration.

By internalizing these operations as `fab pane` subcommands:
- **Validation is built-in**: `fab pane send` refuses to send to a non-existent pane or a busy agent, preventing a class of operator bugs.
- **Context enrichment is automatic**: `fab pane capture` returns structured output with the pane's change, stage, and agent state already resolved — no post-processing needed.
- **Process detection is portable**: `fab pane process` provides OS-level ground truth about what's running in a pane (via `/proc` on Linux, `ps`/`lsof` on macOS), complementing the file-based agent state in `.fab-runtime.yaml`.
- **Command tree is consistent**: `fab pane-map` becomes `fab pane map`, grouping all pane operations under a single parent command.

If we don't do this, operator skills continue to shell out to raw `tmux` commands with hand-rolled validation, and process state remains a blind spot (only file-based idle timestamps, no actual process detection).

## What Changes

### 1. New `fab pane` Parent Command

Create a `pane` parent command in the Cobra command tree (`src/go/fab/cmd/fab/`). This groups all pane-related subcommands:

```
fab pane map       # existing pane-map, moved
fab pane capture   # new: structured pane capture
fab pane send      # new: safe send-keys
fab pane process   # new: OS-level process detection
```

The parent command itself (`fab pane` with no subcommand) shows help listing the available subcommands.

### 2. `fab pane map` — Rename of `pane-map`

Move the existing `paneMapCmd()` from a root-level command to a subcommand of the `pane` parent. The implementation in `panemap.go` is unchanged — only the registration point moves.

Current registration in `main.go`:
```go
root.AddCommand(paneMapCmd())  // fab pane-map
```

New registration:
```go
root.AddCommand(paneCmd())     // fab pane (parent)
// inside paneCmd():
//   cmd.AddCommand(paneMapCmd())    // fab pane map
//   cmd.AddCommand(paneCaptureCmd()) // fab pane capture
//   cmd.AddCommand(paneSendCmd())    // fab pane send
//   cmd.AddCommand(paneProcessCmd()) // fab pane process
```

The `paneMapCmd()` `Use` field changes from `"pane-map"` to `"map"`.

### 3. `fab pane capture` — Structured Pane Content Capture

```
fab pane capture <pane> [-l N] [--json] [--raw]
```

| Flag | Type | Default | Purpose |
|------|------|---------|---------|
| `<pane>` | positional | required | Tmux pane ID (e.g., `%5`) |
| `-l N` | int | 50 | Number of lines to capture (passed to `tmux capture-pane -l`) |
| `--json` | bool | false | Output as JSON with metadata |
| `--raw` | bool | false | Output raw captured text only (no enrichment) |

**Behavior**:
1. Validate the pane exists (via `tmux list-panes`)
2. Run `tmux capture-pane -t <pane> -p -l <N>` to capture content
3. Resolve the pane's fab context: worktree root, active change, current stage, agent state (reusing resolution logic from `panemap.go`)
4. **Default output** (human-readable): captured text with a header block showing pane metadata
5. **`--json` output**: JSON object with `pane`, `content` (the captured text), `change`, `stage`, `agent_state`, `worktree` fields
6. **`--raw` output**: plain captured text only, identical to raw `tmux capture-pane -p` (for backward compatibility in scripts that parse raw output)

**JSON schema**:
```json
{
  "pane": "%5",
  "lines": 50,
  "content": "...",
  "worktree": "myrepo.worktrees/alpha/",
  "change": "260306-r3m7-add-retry-logic",
  "stage": "apply",
  "agent_state": "idle",
  "agent_idle_duration": "2m"
}
```

### 4. `fab pane send` — Safe Send-Keys

```
fab pane send <pane> <text> [--no-enter] [--force]
```

| Flag | Type | Default | Purpose |
|------|------|---------|---------|
| `<pane>` | positional | required | Tmux pane ID (e.g., `%5`) |
| `<text>` | positional | required | Text to send |
| `--no-enter` | bool | false | Don't append Enter keystroke |
| `--force` | bool | false | Skip idle validation (still validates pane existence) |

**Behavior**:
1. **Validate pane exists**: check via `tmux list-panes` that the target pane ID is present. If not, exit 1 with `Error: pane <id> not found`
2. **Validate agent idle**: resolve the pane's worktree and active change, read `.fab-runtime.yaml` to check agent idle state (reusing `resolveAgentState` logic). If the agent is not idle (i.e., `active` or `unknown`), exit 1 with `Error: agent in pane <id> is not idle (state: <state>)`. The `--force` flag bypasses this check.
3. **Send keys**: run `tmux send-keys -t <pane> "<text>" Enter` (or without `Enter` if `--no-enter`)
4. **Output on success**: `Sent to <pane>` to stderr (or silent — TBD)

### 5. `fab pane process` — OS-Level Process Detection

```
fab pane process <pane> [--json]
```

| Flag | Type | Default | Purpose |
|------|------|---------|---------|
| `<pane>` | positional | required | Tmux pane ID (e.g., `%5`) |
| `--json` | bool | false | Output as JSON |

**Behavior**:
1. **Get pane PID**: Use `tmux display-message -t <pane> -p '#{pane_pid}'` to get the shell PID of the pane
2. **Discover child processes**:
   - **Linux**: Read `/proc/<pid>/task/*/children` recursively to build the process tree. Read `/proc/<pid>/comm` and `/proc/<pid>/cmdline` for process names and arguments
   - **macOS**: Use `ps -o pid,ppid,comm -ax` to find child processes by PPID traversal, or `pgrep -P <pid>` for direct children
3. **Classify processes**: Detect known process types:
   - `claude` / `claude-code` → agent process
   - `node` → Node.js runtime (likely agent backend)
   - `git`, `gh` → git operations
   - Other → generic process entry
4. **Default output** (human-readable): process tree with PID, command, and classification
5. **`--json` output**: JSON object with process tree

**JSON schema**:
```json
{
  "pane": "%5",
  "pane_pid": 12345,
  "processes": [
    {
      "pid": 12345,
      "ppid": 1,
      "comm": "zsh",
      "cmdline": "/bin/zsh",
      "children": [
        {
          "pid": 12350,
          "ppid": 12345,
          "comm": "claude",
          "cmdline": "claude --dangerously-skip-permissions ...",
          "children": []
        }
      ]
    }
  ],
  "has_agent": true
}
```

### 6. Shared Pane Utilities

Extract common pane resolution logic from `panemap.go` into a shared internal package (e.g., `internal/pane/`) so that `capture`, `send`, `process`, and `map` can all reuse:

- `ValidatePane(paneID string) error` — check pane exists via `tmux list-panes`
- `ResolvePaneContext(paneID string) (*PaneContext, error)` — resolve worktree, change, stage, agent state for a pane
- `GetPanePID(paneID string) (int, error)` — get the shell PID for a pane

### 7. Skill File Updates

**`src/kit/skills/_cli-fab.md`**: Add a new `## fab pane` section documenting the parent command and all four subcommands (`map`, `capture`, `send`, `process`). Replace the existing `## fab pane-map` section. This is required by the constitution: "Changes to the `fab` CLI (Go binary) MUST include corresponding test updates and MUST update `src/kit/skills/_cli-fab.md`."

**`src/kit/skills/_cli-external.md`**: Remove the `capture-pane` and `send-keys` entries from the `## tmux` section since they are now internalized as `fab pane capture` and `fab pane send`. The `new-window` entry stays — it is not being internalized in this change.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the new `internal/pane` package and `fab pane` command group in the three-binary architecture context

## Impact

- **Go source**: New files in `src/go/fab/cmd/fab/` (pane parent, capture, send, process subcommands) and `src/go/fab/internal/pane/` (shared utilities). Modified `main.go` (command registration) and `panemap.go` (Use field change).
- **Skill files**: `src/kit/skills/_cli-fab.md` (new section), `src/kit/skills/_cli-external.md` (reduced tmux section)
- **Tests**: New test files for each subcommand and the shared pane package. Existing `panemap_test.go` tests remain valid (behavior unchanged, only command path changes).
- **Operator skill**: `fab-operator.md` callers can migrate from raw `tmux capture-pane`/`tmux send-keys` to `fab pane capture`/`fab pane send` in a follow-up change. This change only adds the Go commands — skill migration is out of scope.

## Open Questions

- None identified. The description is specific about all four subcommands and their capabilities.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab pane` parent groups `map`, `capture`, `send`, `process` subcommands | Explicit in description: "fab pane command group" | S:95 R:85 A:90 D:95 |
| 2 | Certain | Routed via fab-go (workflow engine), not fab-kit | All non-workspace commands route to fab-go per three-binary architecture | S:85 R:90 A:95 D:95 |
| 3 | Certain | `fab pane map` replaces `fab pane-map` with identical behavior | Explicit: "Rename existing pane-map to fab pane map as a subcommand" | S:95 R:80 A:90 D:95 |
| 4 | Certain | `fab pane send` validates pane existence and agent idle before sending | Explicit: "safe send-keys with built-in pane existence and agent idle validation" | S:95 R:75 A:85 D:90 |
| 5 | Certain | `fab pane process` uses /proc on Linux, ps/lsof on macOS | Explicit: "OS-level process state detection via /proc on Linux, ps/lsof on macOS" | S:95 R:80 A:85 D:90 |
| 6 | Certain | `_cli-fab.md` updated with new `fab pane` section per constitution requirement | Constitution: "Changes to the fab CLI MUST update src/kit/skills/_cli-fab.md" | S:90 R:85 A:95 D:90 |
| 7 | Confident | Shared pane resolution logic extracted to `internal/pane/` package | Strong signal from code duplication across subcommands; codebase uses internal packages for shared logic | S:70 R:80 A:80 D:65 |
| 8 | Confident | `fab pane capture --json` returns enriched JSON with change/stage/agent metadata | "fab context enrichment" implies metadata; consistent with `pane-map --json` pattern | S:75 R:75 A:70 D:60 |
| 9 | Confident | Agent idle validation in `send` reuses `.fab-runtime.yaml` approach from `resolveAgentState` | Existing idle detection pattern is file-based; natural to reuse | S:70 R:80 A:85 D:70 |
| 10 | Confident | `_cli-external.md` tmux section reduced (capture-pane and send-keys removed, new-window stays) | Capture and send are internalized; new-window is not part of this change | S:70 R:70 A:75 D:60 |
| 11 | Tentative | No backward-compatibility alias for `fab pane-map` → `fab pane map` | Not mentioned in description; callers (operator, batch) are in the same binary and can be updated atomically. But external scripts may call `fab pane-map` | S:30 R:60 A:50 D:40 |

11 assumptions (6 certain, 4 confident, 1 tentative, 0 unresolved). Run /fab-clarify to review.
