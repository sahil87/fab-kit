# Spec: Fab Pane Command Group

**Change**: 260403-tam1-pane-commands
**Created**: 2026-04-03
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Migrating existing skill files (e.g., `fab-operator.md`) from raw tmux calls to `fab pane` subcommands — that is a follow-up change
- Internalizing `tmux new-window` — it remains in `_cli-external.md`
- Adding a backward-compatibility alias for `fab pane-map` — callers are in the same binary and updated atomically

## CLI: Parent Command

### Requirement: `fab pane` Parent Command

The `fab` binary SHALL provide a `pane` parent command that groups all pane-related subcommands. The parent command itself (invoked with no subcommand) SHALL display help listing the available subcommands. The parent command SHALL be registered in `main.go` replacing the current `paneMapCmd()` root-level registration.

#### Scenario: Invoke `fab pane` with no subcommand
- **GIVEN** the `fab` binary is built with the `pane` parent command
- **WHEN** a user runs `fab pane`
- **THEN** Cobra displays the help text listing `map`, `capture`, `send`, `process` subcommands

#### Scenario: Routing via fab-go
- **GIVEN** the `fab` router resolves a versioned `fab-go` binary
- **WHEN** a user runs `fab pane <subcommand>`
- **THEN** the router dispatches to `fab-go` (not `fab-kit`) because `pane` is not a workspace command

### Requirement: Command Registration

The `paneCmd()` function in a new file `pane.go` SHALL return a `*cobra.Command` with `Use: "pane"` and SHALL register four subcommands: `paneMapCmd()`, `paneCaptureCmd()`, `paneSendCmd()`, `paneProcessCmd()`. The `main.go` file SHALL replace `root.AddCommand(paneMapCmd())` with `root.AddCommand(paneCmd())`.

#### Scenario: Command tree structure
- **GIVEN** the `fab` binary is built
- **WHEN** a developer inspects the Cobra command tree
- **THEN** `pane` is a child of root, and `map`, `capture`, `send`, `process` are children of `pane`

## CLI: `fab pane map`

### Requirement: Rename from `pane-map`

The existing `paneMapCmd()` in `panemap.go` SHALL change its `Use` field from `"pane-map"` to `"map"`. All existing behavior, flags (`--json`, `--session`, `--all-sessions`), output formats, and tests SHALL remain unchanged. The command is now invoked as `fab pane map` instead of `fab pane-map`.

#### Scenario: Invoke `fab pane map`
- **GIVEN** a tmux session with panes
- **WHEN** a user runs `fab pane map`
- **THEN** the output is identical to the former `fab pane-map` command (aligned table with Pane, WinIdx, Tab, Worktree, Change, Stage, Agent columns)

#### Scenario: JSON output
- **GIVEN** a tmux session with panes
- **WHEN** a user runs `fab pane map --json`
- **THEN** the output is a JSON array identical to the former `fab pane-map --json`

## Shared: `internal/pane` Package

### Requirement: Pane Validation

The `internal/pane` package SHALL export a `ValidatePane(paneID string) error` function that verifies a tmux pane exists by running `tmux list-panes -a -F '#{pane_id}'` and checking that `paneID` appears in the output. If the pane does not exist, it SHALL return an error with the message `pane <id> not found`.

#### Scenario: Valid pane
- **GIVEN** a tmux session with pane `%5`
- **WHEN** `ValidatePane("%5")` is called
- **THEN** it returns `nil`

#### Scenario: Invalid pane
- **GIVEN** a tmux session without pane `%99`
- **WHEN** `ValidatePane("%99")` is called
- **THEN** it returns an error containing `pane %99 not found`

### Requirement: Pane Context Resolution

The `internal/pane` package SHALL export a `ResolvePaneContext(paneID string) (*PaneContext, error)` function that resolves the fab context for a pane. The `PaneContext` struct SHALL contain: `Pane string`, `CWD string`, `WorktreeRoot string`, `WorktreeDisplay string`, `Change *string` (nil if no active change), `Stage *string` (nil if no stage), `AgentState *string` (nil if not applicable), `AgentIdleDuration *string` (nil if not idle).

The resolution logic SHALL be extracted from `panemap.go`'s `resolvePane` function. It SHALL: (1) get the pane CWD via `tmux display-message -t <pane> -p '#{pane_current_path}'`, (2) resolve the git worktree root, (3) check for `fab/` directory, (4) read `.fab-status.yaml` symlink, (5) read `.status.yaml` for stage, (6) read `.fab-runtime.yaml` for agent state.

#### Scenario: Pane in a fab worktree with active change
- **GIVEN** pane `%5` is in a directory that is a fab worktree with an active change at the `apply` stage
- **WHEN** `ResolvePaneContext("%5")` is called
- **THEN** `PaneContext.Change` is non-nil and contains the change folder name
- **AND** `PaneContext.Stage` is non-nil and contains `"apply"`

#### Scenario: Pane in a non-fab directory
- **GIVEN** pane `%5` is in `/tmp`
- **WHEN** `ResolvePaneContext("%5")` is called
- **THEN** `PaneContext.Change` is nil, `PaneContext.Stage` is nil, `PaneContext.AgentState` is nil

### Requirement: Pane PID Resolution

The `internal/pane` package SHALL export a `GetPanePID(paneID string) (int, error)` function that returns the shell PID of a tmux pane by running `tmux display-message -t <pane> -p '#{pane_pid}'` and parsing the integer result.

#### Scenario: Get PID of valid pane
- **GIVEN** a tmux session with pane `%5` running a shell with PID 12345
- **WHEN** `GetPanePID("%5")` is called
- **THEN** it returns `(12345, nil)`

### Requirement: Main Worktree Root Resolution

The `internal/pane` package SHALL export a `FindMainWorktreeRoot(paneIDs []string) string` function, extracted from `panemap.go`'s `findMainWorktreeRoot`. This accepts a slice of pane CWDs (not paneEntry structs) and returns the main worktree root path.

#### Scenario: Resolve main worktree
- **GIVEN** pane CWDs that include a directory within a git worktree
- **WHEN** `FindMainWorktreeRoot(cwds)` is called
- **THEN** it returns the path of the main worktree

## CLI: `fab pane capture`

### Requirement: Structured Pane Capture

The `fab pane capture` command SHALL capture terminal content from a tmux pane and enrich it with fab context. It SHALL accept a positional `<pane>` argument (required) and the following flags: `-l` (int, default 50, number of lines), `--json` (bool), `--raw` (bool). `--json` and `--raw` SHALL be mutually exclusive.

#### Scenario: Default output (human-readable)
- **GIVEN** pane `%5` in a fab worktree with active change `260306-r3m7-add-retry-logic` at stage `apply`
- **WHEN** a user runs `fab pane capture %5`
- **THEN** stdout contains a header block with pane metadata (pane ID, change, stage, agent state) followed by the captured terminal content

#### Scenario: JSON output
- **GIVEN** pane `%5` in a fab worktree
- **WHEN** a user runs `fab pane capture %5 --json`
- **THEN** stdout is a JSON object with fields: `pane` (string), `lines` (int), `content` (string), `worktree` (string), `change` (string|null), `stage` (string|null), `agent_state` (string|null), `agent_idle_duration` (string|null)

#### Scenario: Raw output
- **GIVEN** pane `%5`
- **WHEN** a user runs `fab pane capture %5 --raw`
- **THEN** stdout contains only the raw captured text, identical to `tmux capture-pane -t %5 -p -l 50`

#### Scenario: Custom line count
- **GIVEN** pane `%5`
- **WHEN** a user runs `fab pane capture %5 -l 20`
- **THEN** the capture uses `tmux capture-pane -t %5 -p -l 20`

#### Scenario: Invalid pane
- **GIVEN** no pane `%99` exists
- **WHEN** a user runs `fab pane capture %99`
- **THEN** stderr shows `Error: pane %99 not found` and exit code is 1

### Requirement: Capture Implementation

The capture command SHALL: (1) validate the pane exists via `pane.ValidatePane()`, (2) run `tmux capture-pane -t <pane> -p -l <N>` to capture content, (3) resolve fab context via `pane.ResolvePaneContext()`, (4) format output based on flags.

#### Scenario: Pane exists but is not in a fab worktree
- **GIVEN** pane `%5` is in `/tmp` (no git repo)
- **WHEN** a user runs `fab pane capture %5 --json`
- **THEN** the JSON output has `change: null`, `stage: null`, `agent_state: null` and the `content` field contains the captured text

## CLI: `fab pane send`

### Requirement: Safe Send-Keys

The `fab pane send` command SHALL send keystrokes to a tmux pane with built-in validation. It SHALL accept two positional arguments: `<pane>` (required) and `<text>` (required), plus flags: `--no-enter` (bool, default false), `--force` (bool, default false).

#### Scenario: Send to idle agent
- **GIVEN** pane `%5` exists and the agent is idle (`.fab-runtime.yaml` has `idle_since` for the active change)
- **WHEN** a user runs `fab pane send %5 "/fab-status"`
- **THEN** `tmux send-keys -t %5 "/fab-status" Enter` is executed
- **AND** stdout shows `Sent to %5`

#### Scenario: Send without Enter
- **GIVEN** pane `%5` exists and the agent is idle
- **WHEN** a user runs `fab pane send %5 "some text" --no-enter`
- **THEN** `tmux send-keys -t %5 "some text"` is executed (no `Enter` appended)

#### Scenario: Reject send to busy agent
- **GIVEN** pane `%5` exists and the agent is active (no `idle_since` in `.fab-runtime.yaml`)
- **WHEN** a user runs `fab pane send %5 "/fab-status"`
- **THEN** stderr shows `Error: agent in pane %5 is not idle (state: active)` and exit code is 1
- **AND** no `tmux send-keys` is executed

#### Scenario: Force send to busy agent
- **GIVEN** pane `%5` exists and the agent is active
- **WHEN** a user runs `fab pane send %5 "/fab-status" --force`
- **THEN** the idle check is skipped, `tmux send-keys` is executed, and stdout shows `Sent to %5`

#### Scenario: Send to non-existent pane
- **GIVEN** no pane `%99` exists
- **WHEN** a user runs `fab pane send %99 "text"`
- **THEN** stderr shows `Error: pane %99 not found` and exit code is 1 (even with `--force`)

### Requirement: Send Validation Pipeline

The send command SHALL validate in order: (1) pane exists via `pane.ValidatePane()`, (2) resolve pane context via `pane.ResolvePaneContext()`, (3) check agent idle state unless `--force` is set. If the pane has no active change or no runtime file, the agent state is unknown — the command SHALL treat `unknown` as non-idle and reject (unless `--force`).

#### Scenario: Pane with no active change
- **GIVEN** pane `%5` is in a fab worktree with no active change (`.fab-status.yaml` absent)
- **WHEN** a user runs `fab pane send %5 "text"`
- **THEN** stderr shows `Error: agent in pane %5 is not idle (state: unknown)` and exit code is 1

#### Scenario: Pane in non-fab directory with force
- **GIVEN** pane `%5` is in `/tmp`
- **WHEN** a user runs `fab pane send %5 "text" --force`
- **THEN** the send proceeds (pane exists, force skips idle check)

## CLI: `fab pane process`

### Requirement: OS-Level Process Detection

The `fab pane process` command SHALL detect the process tree running in a tmux pane. It SHALL accept a positional `<pane>` argument (required) and a `--json` flag (bool, default false).

#### Scenario: Default output (human-readable)
- **GIVEN** pane `%5` with shell PID 12345 running a `claude` child process (PID 12350)
- **WHEN** a user runs `fab pane process %5`
- **THEN** stdout shows a tree-formatted process listing with PID, command name, and classification

#### Scenario: JSON output
- **GIVEN** pane `%5` with a process tree
- **WHEN** a user runs `fab pane process %5 --json`
- **THEN** stdout is a JSON object with fields: `pane` (string), `pane_pid` (int), `processes` (array of process nodes), `has_agent` (bool)

#### Scenario: Invalid pane
- **GIVEN** no pane `%99` exists
- **WHEN** a user runs `fab pane process %99`
- **THEN** stderr shows `Error: pane %99 not found` and exit code is 1

### Requirement: Process Tree Discovery (Linux)

On Linux, the command SHALL discover child processes by reading `/proc/<pid>/task/<tid>/children` recursively to build the full process tree. For each discovered PID, it SHALL read `/proc/<pid>/comm` for the command name and `/proc/<pid>/cmdline` (NUL-separated) for the full command line.

#### Scenario: Linux process tree
- **GIVEN** a Linux host with pane PID 12345 and child PID 12350 (`claude`)
- **WHEN** process discovery runs
- **THEN** `/proc/12345/task/12345/children` is read, then `/proc/12350/comm` and `/proc/12350/cmdline` are read
- **AND** the resulting tree has 12345 as root with 12350 as a child

### Requirement: Process Tree Discovery (macOS)

On macOS, the command SHALL discover child processes by running `ps -o pid,ppid,comm -ax` and filtering by PPID traversal from the pane PID. For the full command line, it SHALL use `ps -o args= -p <pid>`.

#### Scenario: macOS process tree
- **GIVEN** a macOS host with pane PID 12345
- **WHEN** process discovery runs
- **THEN** `ps -o pid,ppid,comm -ax` output is parsed, and all PIDs with PPID chain leading to 12345 are included

### Requirement: Process Classification

The command SHALL classify each process based on its `comm` name (case-insensitive):
- `claude` or `claude-code` → `"agent"`
- `node` → `"node"`
- `git` or `gh` → `"git"`
- All others → `"other"`

The `has_agent` field in JSON output SHALL be `true` if any process in the tree is classified as `"agent"`.

#### Scenario: Agent detected
- **GIVEN** the process tree contains a process with comm `claude`
- **WHEN** classification runs
- **THEN** that process is classified as `"agent"` and `has_agent` is `true`

#### Scenario: No agent
- **GIVEN** the process tree contains only `zsh` and `node`
- **WHEN** classification runs
- **THEN** `has_agent` is `false`

### Requirement: Process Node Schema

Each process node in JSON output SHALL have fields: `pid` (int), `ppid` (int), `comm` (string), `cmdline` (string), `classification` (string), `children` (array of process nodes, recursive).

#### Scenario: Nested process tree
- **GIVEN** PID 100 (zsh) → PID 200 (claude) → PID 300 (node)
- **WHEN** JSON output is generated
- **THEN** the tree is nested: 100 contains 200, 200 contains 300

## Skills: `_cli-fab.md` Update

### Requirement: New `fab pane` Section

The `_cli-fab.md` skill file SHALL replace the existing `## fab pane-map` section with a new `## fab pane` section documenting the parent command and all four subcommands: `map`, `capture`, `send`, `process`. Each subcommand SHALL include its usage signature, flags table, behavior description, and output format.

#### Scenario: Skill file reflects new command tree
- **GIVEN** the updated `_cli-fab.md`
- **WHEN** an agent reads the file to invoke pane commands
- **THEN** it finds `fab pane capture`, `fab pane send`, `fab pane process`, and `fab pane map` documented under a single `## fab pane` parent section

### Requirement: Command Reference Table Update

The Command Reference table at the top of `_cli-fab.md` SHALL replace the `fab pane-map` entry with `fab pane` and update the description to reflect the command group.

#### Scenario: Command reference
- **GIVEN** the updated `_cli-fab.md`
- **WHEN** an agent reads the Command Reference table
- **THEN** the entry reads `fab pane` with purpose "Tmux pane operations: map, capture, send, process"

## Skills: `_cli-external.md` Update

### Requirement: Reduced tmux Section

The `_cli-external.md` skill file SHALL remove the `capture-pane` and `send-keys` rows from the tmux Commands table, as these operations are now provided by `fab pane capture` and `fab pane send`. The `new-window` row SHALL remain. The Usage Notes SHALL be updated to reference `fab pane capture` and `fab pane send` instead of raw tmux commands.

#### Scenario: Reduced tmux section
- **GIVEN** the updated `_cli-external.md`
- **WHEN** an agent reads the `## tmux` section
- **THEN** only `new-window` appears in the Commands table
- **AND** Usage Notes reference `fab pane capture` and `fab pane send` for the internalized operations

## Deprecated Requirements

### `fab pane-map` as Root-Level Command

**Reason**: Replaced by `fab pane map` subcommand under the new `pane` parent.
**Migration**: All callers use `fab pane map`. No backward-compatibility alias.

## Design Decisions

1. **No backward-compat alias for `pane-map`**: All callers (`operator.go`, `batch_new.go`, `batch_switch.go`, skill files) are in the same repository and updated atomically in this change. No external consumers are known.
   - *Rejected*: Adding a Cobra alias — adds maintenance burden for zero external users.

2. **Shared `internal/pane` package rather than keeping logic in command files**: Four subcommands need pane validation, context resolution, and PID resolution. Duplicating this across command files violates DRY and the existing codebase pattern (15 internal packages already exist).
   - *Rejected*: Inline logic per command file — would triple the code for validation/resolution.

3. **`unknown` agent state treated as non-idle in `send`**: Conservative default prevents sending to a pane whose state can't be determined. `--force` overrides for deliberate use.
   - *Rejected*: Treating unknown as idle — would risk sending to an active agent when runtime file is missing.

4. **Platform-specific process discovery via build tags**: Linux uses `/proc` (zero subprocess overhead), macOS uses `ps` (standard POSIX tool). Separated via Go build tags (`//go:build linux` / `//go:build darwin`) for clean platform isolation.
   - *Rejected*: Using `ps` on both platforms — loses the performance and detail advantage of `/proc` on Linux.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab pane` parent groups `map`, `capture`, `send`, `process` subcommands | Confirmed from intake #1 — explicit in description | S:95 R:85 A:90 D:95 |
| 2 | Certain | Routed via fab-go (workflow engine), not fab-kit | Confirmed from intake #2 — three-binary architecture | S:85 R:90 A:95 D:95 |
| 3 | Certain | `fab pane map` replaces `fab pane-map` with identical behavior | Confirmed from intake #3 — explicit rename | S:95 R:80 A:90 D:95 |
| 4 | Certain | `fab pane send` validates pane existence and agent idle before sending | Confirmed from intake #4 — explicit requirement | S:95 R:75 A:85 D:90 |
| 5 | Certain | `fab pane process` uses /proc on Linux, ps on macOS | Confirmed from intake #5 — explicit platforms | S:95 R:80 A:85 D:90 |
| 6 | Certain | `_cli-fab.md` updated with `fab pane` section | Confirmed from intake #6 — constitution mandate | S:90 R:85 A:95 D:90 |
| 7 | Certain | Shared logic in `internal/pane/` package | Upgraded from intake #7 Confident — codebase has 15 internal packages following this pattern | S:85 R:80 A:90 D:85 |
| 8 | Certain | `fab pane capture --json` returns enriched JSON with change/stage/agent metadata | Upgraded from intake #8 Confident — identical pattern to `pane-map --json` in panemap.go | S:85 R:75 A:85 D:80 |
| 9 | Certain | Agent idle validation reuses `.fab-runtime.yaml` approach | Confirmed from intake #9 — resolveAgentState already implements this | S:80 R:80 A:90 D:80 |
| 10 | Certain | `_cli-external.md` reduced: capture-pane and send-keys removed, new-window stays | Confirmed from intake #10 — new-window not in scope | S:80 R:70 A:80 D:75 |
| 11 | Confident | No backward-compatibility alias for `fab pane-map` | Maintained from intake #11 Tentative, upgraded — all callers are in-repo, updated atomically | S:50 R:65 A:70 D:55 |
| 12 | Certain | `--raw` and `--json` are mutually exclusive in `capture` | Standard CLI convention; Cobra `MarkFlagsMutuallyExclusive` | S:90 R:90 A:95 D:95 |
| 13 | Certain | Process classification is best-effort — known types classified, rest are `"other"` | Cannot enumerate all processes; extensible classification is sufficient | S:85 R:90 A:85 D:90 |
| 14 | Certain | Platform-specific process discovery via Go build tags | Standard Go pattern for platform code; clean separation | S:85 R:85 A:90 D:90 |
| 15 | Certain | `panemap.go` resolution logic extracted (not copied) to `internal/pane/` | Extract-and-import avoids duplication; panemap.go becomes a thin caller | S:80 R:80 A:90 D:85 |

15 assumptions (14 certain, 1 confident, 0 tentative, 0 unresolved).
