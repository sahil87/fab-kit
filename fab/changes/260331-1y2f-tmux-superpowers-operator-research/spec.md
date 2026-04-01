# Spec: Tmux Superpowers — Operator Research

**Change**: 260331-1y2f-tmux-superpowers-operator-research
**Created**: 2026-04-01
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md` (modify)

## Non-Goals

- `fab pane spawn` — wrapping `tmux new-window` moves complexity without reducing it; spawning is a one-shot operation with complex argument construction that remains raw tmux
- Adopting an external tmux MCP server — violates Constitution §I (Pure Prompt Play) and §V (Portability)
- `tmux wait-for` integration — blocking primitives do not help LLM-driven agents that work via tool calls and polling ticks
- Backward-compatible `fab pane-map` alias — explicitly dropped per user confirmation

## CLI: `pane` Command Group

### Requirement: Parent Command Registration

The `fab` binary SHALL register a `pane` parent command group. The existing `pane-map` top-level command SHALL be removed and re-registered as a `map` subcommand under `pane`. The `pane` parent command itself SHALL have no `RunE` — invoking `fab pane` with no subcommand MUST print help listing all subcommands.

#### Scenario: Parent command help

- **GIVEN** the `fab` binary is built with the `pane` command group
- **WHEN** a user runs `fab pane`
- **THEN** the output MUST list `map`, `capture`, `send`, and `process` as available subcommands

#### Scenario: Old pane-map command removed

- **GIVEN** the `fab` binary is built with the `pane` command group
- **WHEN** a user runs `fab pane-map`
- **THEN** the command MUST fail with an unknown command error
- **AND** `fab pane map` MUST produce the same output that `fab pane-map` previously produced

### Requirement: `fab pane map` (renamed)

`fab pane map` SHALL provide identical behavior to the former `fab pane-map` command. All existing flags (`--json`, `--session`, `--all-sessions`) and output formats MUST be preserved unchanged. The implementation MAY reuse the existing `paneMapCmd()` function by re-registering it under the `pane` parent with `Use: "map"`.

#### Scenario: Flags preserved

- **GIVEN** a tmux session with multiple panes in fab worktrees
- **WHEN** a user runs `fab pane map --json`
- **THEN** the output MUST be a JSON array with the same schema as the former `fab pane-map --json`

#### Scenario: Session targeting preserved

- **GIVEN** multiple tmux sessions
- **WHEN** a user runs `fab pane map --all-sessions`
- **THEN** panes from all sessions MUST be included in the output
- **AND** `--session` and `--all-sessions` MUST remain mutually exclusive

## CLI: `pane capture`

### Requirement: Raw Text Capture

`fab pane capture <pane>` SHALL capture the visible content of the specified tmux pane and write it to stdout. The `<pane>` argument is required (positional, tmux pane ID like `%3`). The command MUST accept a `-l N` flag specifying the number of lines to capture (default: all visible lines). Default output mode SHALL be raw text, equivalent to `tmux capture-pane -t <pane> -p [-l N]`.

#### Scenario: Raw capture with line limit

- **GIVEN** tmux pane `%3` exists with 50+ lines of content
- **WHEN** a user runs `fab pane capture %3 -l 20`
- **THEN** stdout MUST contain the last 20 lines of pane `%3` as plain text
- **AND** exit code MUST be 0

#### Scenario: Pane does not exist

- **GIVEN** tmux pane `%99` does not exist
- **WHEN** a user runs `fab pane capture %99`
- **THEN** the command MUST exit with code 1
- **AND** stderr MUST contain a message indicating the pane was not found

### Requirement: JSON Capture with Fab Context

When the `--json` flag is provided, `fab pane capture` SHALL output a JSON object enriched with fab context. The JSON object MUST include: `pane` (string, pane ID), `lines` (int, number of lines captured), `content` (string, raw captured text), `change` (string or null, active change folder name resolved from the pane's worktree), `stage` (string or null, current pipeline stage), and `agent_state` (string or null, agent idle/active state from `.fab-runtime.yaml`). Fab context resolution SHALL reuse the same logic as `fab pane map` (worktree root detection, `.fab-status.yaml` symlink reading, `.fab-runtime.yaml` agent state).

#### Scenario: JSON capture with active change

- **GIVEN** tmux pane `%3` is in a worktree with active change `260331-r3m7-add-retry-logic` at stage `apply` with agent `idle`
- **WHEN** a user runs `fab pane capture %3 -l 20 --json`
- **THEN** stdout MUST be a JSON object with `pane: "%3"`, `lines: 20`, `content` containing the captured text, `change: "260331-r3m7-add-retry-logic"`, `stage: "apply"`, and `agent_state: "idle"`

#### Scenario: JSON capture with no fab context

- **GIVEN** tmux pane `%5` is in a directory with no fab project
- **WHEN** a user runs `fab pane capture %5 --json`
- **THEN** stdout MUST be a JSON object with `change: null`, `stage: null`, and `agent_state: null`

## CLI: `pane send`

### Requirement: Safe Send with Validation

`fab pane send <pane> <text>` SHALL send the specified text to the tmux pane, followed by Enter (via `tmux send-keys -t <pane> "<text>" Enter`). The `<pane>` and `<text>` arguments are both required (positional). By default (without `--force`), the command MUST perform two validation checks before sending:

1. **Pane existence**: verify the pane exists. If the pane does not exist, the command MUST exit with code 1 and print `"pane <pane> not found"` to stderr.
2. **Agent idle check**: read `.fab-runtime.yaml` from the pane's worktree root and verify the agent is idle (has `idle_since` set). If the agent is active (no `idle_since`), the command MUST exit with code 1 and print `"agent in <pane> is active, use --force to override"` to stderr.

If both checks pass, the command SHALL send the text and exit with code 0.

#### Scenario: Successful send to idle agent

- **GIVEN** tmux pane `%3` exists in a worktree where `.fab-runtime.yaml` shows the agent is idle
- **WHEN** a user runs `fab pane send %3 "/fab-continue"`
- **THEN** `tmux send-keys -t %3 "/fab-continue" Enter` MUST be executed
- **AND** exit code MUST be 0

#### Scenario: Send blocked by active agent

- **GIVEN** tmux pane `%3` exists in a worktree where `.fab-runtime.yaml` shows the agent is active (no `idle_since`)
- **WHEN** a user runs `fab pane send %3 "/fab-continue"`
- **THEN** the command MUST exit with code 1
- **AND** stderr MUST contain `"agent in %3 is active, use --force to override"`
- **AND** no `tmux send-keys` MUST be executed

#### Scenario: Send to non-existent pane

- **GIVEN** tmux pane `%99` does not exist
- **WHEN** a user runs `fab pane send %99 "/fab-continue"`
- **THEN** the command MUST exit with code 1
- **AND** stderr MUST contain `"pane %99 not found"`

### Requirement: Force Mode

When the `--force` flag is provided, `fab pane send` SHALL skip the agent idle check. Pane existence validation MUST still be performed even with `--force`. This mode is intended for sending responses to agent prompts (e.g., `fab pane send %3 "y" --force`).

#### Scenario: Force send to active agent

- **GIVEN** tmux pane `%3` exists with an active agent
- **WHEN** a user runs `fab pane send %3 "y" --force`
- **THEN** `tmux send-keys -t %3 "y" Enter` MUST be executed
- **AND** exit code MUST be 0

#### Scenario: Force send to non-existent pane still fails

- **GIVEN** tmux pane `%99` does not exist
- **WHEN** a user runs `fab pane send %99 "y" --force`
- **THEN** the command MUST exit with code 1
- **AND** stderr MUST contain `"pane %99 not found"`

### Requirement: Send to Non-Fab Panes

When a pane is not in a fab worktree (no `fab/` directory or no `.fab-runtime.yaml`), `fab pane send` without `--force` SHOULD treat the pane as idle (no runtime state to check) and proceed with the send. The command SHALL NOT require fab context to function — it MUST work for any tmux pane.
<!-- assumed: Non-fab panes treated as idle — the operator may need to send commands to non-fab panes (e.g., utility shells), and requiring --force for every non-fab pane would be unnecessarily cumbersome -->

#### Scenario: Send to non-fab pane

- **GIVEN** tmux pane `%7` exists in a directory with no fab project
- **WHEN** a user runs `fab pane send %7 "ls -la"`
- **THEN** `tmux send-keys -t %7 "ls -la" Enter` MUST be executed
- **AND** exit code MUST be 0

## CLI: `pane process`

### Requirement: Process State Detection

`fab pane process <pane>` SHALL detect the OS-level state of the foreground process in the specified tmux pane. The command MUST resolve the pane's foreground process by: (1) getting the pane's PID via `tmux display-message -t <pane> -p '#{pane_pid}'`, (2) walking the process tree to find the foreground process group leader via the pane's tty. The command MUST report one of these states:

| State | Meaning |
|---|---|
| `running` | Foreground process is actively executing (R state in `/proc/stat` on Linux) |
| `waiting-for-input` | Foreground process is blocked on tty read (`S` state + wchan indicates read/poll on tty fd) |
| `sleeping` | Foreground process is sleeping but not on tty read (e.g., `sleep`, network I/O) |
| `stopped` | Foreground process is stopped (SIGSTOP/SIGTSTP — T state) |
| `exited` | No foreground process beyond the pane's shell |

Default output SHALL be a single word on stdout (one of the five states above). Exit code SHALL be 0 on success.

#### Scenario: Detect running process

- **GIVEN** tmux pane `%3` has a foreground process actively executing (CPU-bound)
- **WHEN** a user runs `fab pane process %3`
- **THEN** stdout MUST contain exactly `running` (followed by a newline)
- **AND** exit code MUST be 0

#### Scenario: Detect waiting-for-input

- **GIVEN** tmux pane `%3` has a foreground process (e.g., `claude`) blocked waiting for user input on the tty
- **WHEN** a user runs `fab pane process %3`
- **THEN** stdout MUST contain exactly `waiting-for-input`

#### Scenario: Detect exited (shell only)

- **GIVEN** tmux pane `%3` shows only the shell prompt (no foreground process running)
- **WHEN** a user runs `fab pane process %3`
- **THEN** stdout MUST contain exactly `exited`

#### Scenario: Pane does not exist

- **GIVEN** tmux pane `%99` does not exist
- **WHEN** a user runs `fab pane process %99`
- **THEN** the command MUST exit with code 1
- **AND** stderr MUST contain a message indicating the pane was not found

### Requirement: JSON Process Output

When the `--json` flag is provided, `fab pane process` SHALL output a JSON object with: `pane` (string, pane ID), `pid` (int, foreground process PID), `state` (string, one of the five states), `process_name` (string, name of the foreground process), and `change` (string or null, active change folder name resolved from the pane's worktree). When the state is `exited`, `pid` SHALL be the shell PID and `process_name` SHALL be the shell name.

#### Scenario: JSON output for waiting process

- **GIVEN** tmux pane `%3` has `claude` (PID 12345) waiting for input, in a worktree with change `260331-r3m7-add-retry-logic`
- **WHEN** a user runs `fab pane process %3 --json`
- **THEN** stdout MUST be a JSON object with `pane: "%3"`, `pid: 12345`, `state: "waiting-for-input"`, `process_name: "claude"`, and `change: "260331-r3m7-add-retry-logic"`

### Requirement: Cross-Platform Support

Process state detection MUST support both Linux and macOS. On Linux, the implementation SHALL use `/proc/{pid}/stat` for process state and `/proc/{pid}/wchan` for wait channel detection. On macOS, the implementation SHALL use `ps -o stat= -p {pid}` for coarse state and `lsof` for tty-read detection. The platform abstraction SHALL be implemented in Go with build tags or runtime detection. <!-- assumed: Runtime GOOS detection rather than build tags — simpler for a single binary, and the platform-specific code is limited to process introspection helpers -->

#### Scenario: Linux process detection

- **GIVEN** the `fab` binary is running on Linux
- **WHEN** `fab pane process %3` needs to determine process state
- **THEN** it MUST read `/proc/{pid}/stat` and `/proc/{pid}/wchan`
- **AND** it MUST NOT shell out to `ps` or `lsof`

#### Scenario: macOS process detection

- **GIVEN** the `fab` binary is running on macOS
- **WHEN** `fab pane process %3` needs to determine process state
- **THEN** it MUST use `ps -o stat= -p {pid}` and `lsof`
- **AND** it MUST NOT attempt to read `/proc/`

### Requirement: Graceful Degradation

When process state detection cannot definitively distinguish `waiting-for-input` from `sleeping` (e.g., ambiguous wchan values, unexpected `/proc` format), the command SHOULD fall back to reporting `sleeping`. The command MUST NOT fail or return an error for ambiguous states — the existing capture-and-regex heuristic in the operator serves as a downstream fallback.

#### Scenario: Ambiguous sleep state

- **GIVEN** tmux pane `%3` has a foreground process in sleep state with an unrecognized wchan value
- **WHEN** a user runs `fab pane process %3`
- **THEN** stdout MUST contain `sleeping` (not an error)
- **AND** exit code MUST be 0

## Downstream: Operator and Documentation Impact

### Requirement: CLI Documentation Updates

Per Constitution constraint (CLI changes MUST update `_cli-fab.md`), `fab/.kit/skills/_cli-fab.md` SHALL be updated to document all four `fab pane` subcommands (`map`, `capture`, `send`, `process`) with their full signatures, flags, and output formats.

`fab/.kit/skills/_cli-external.md` SHALL have its tmux section updated: raw `tmux capture-pane` and `tmux send-keys` entries SHALL be replaced by references to `fab pane capture` and `fab pane send`. `tmux new-window` SHALL remain as the only raw tmux command.

#### Scenario: CLI docs reflect new commands

- **GIVEN** the change is complete
- **WHEN** an agent reads `_cli-fab.md`
- **THEN** it MUST find documentation for `fab pane map`, `fab pane capture`, `fab pane send`, and `fab pane process`
- **AND** `_cli-external.md` MUST NOT document `tmux capture-pane` or `tmux send-keys` as direct-use commands

### Requirement: Operator Skill Simplification

`fab/.kit/skills/fab-operator7.md` SHALL be updated:

- **Section 3 (Pre-Send Validation)**: steps 1-2 (pane exists + agent idle) SHALL be replaced by a single `fab pane send` call. Steps 3-4 (change active, branch aligned) SHALL remain as operator-level logic.
- **Section 5 (Question Detection)**: SHALL gain a `fab pane process` pre-filter step before the capture-and-regex step. The flow becomes: (1) `fab pane map` to find idle agents, (2) `fab pane process <pane>` to check for `waiting-for-input`, (3) only then `fab pane capture <pane> -l 20` for regex matching.

#### Scenario: Operator pre-send validation simplified

- **GIVEN** the operator needs to send a command to pane `%3`
- **WHEN** it follows the updated Section 3
- **THEN** it MUST use `fab pane send %3 "<command>"` for existence + idle validation
- **AND** it MUST separately verify change-active and branch-aligned (steps 3-4)

#### Scenario: Operator question detection with process pre-filter

- **GIVEN** the operator is checking for agent questions
- **WHEN** it follows the updated Section 5
- **THEN** it MUST run `fab pane process <pane>` before `fab pane capture`
- **AND** it MUST only proceed to capture-and-regex when the process state is `waiting-for-input`

## Design Decisions

1. **Internalize into `fab-go` rather than adopt external MCP server**
   - *Why*: Constitution §I (no runtime frameworks) and §V (portability) prohibit external MCP dependencies. The `fab-go` binary already has shared plumbing (change resolution, runtime state, tmux integration) that new pane commands need.
   - *Rejected*: `MadAppGang/tmux-mcp` as an external dependency — adds runtime dep, creates conditional operator code paths.

2. **`pane` parent command with subcommands, not flat `pane-*` commands**
   - *Why*: Groups related functionality under a single namespace. Consistent with `fab status`, `fab runtime`, `fab change` patterns in the existing CLI.
   - *Rejected*: Individual top-level commands (`fab pane-capture`, `fab pane-send`) — inconsistent with existing grouping pattern, clutters `fab --help`.

3. **`pane send` defaults to safe mode (idle + exists check)**
   - *Why*: Mirrors operator Section 3 behavior — safe by default prevents accidental sends to active agents. `--force` provides escape hatch for intentional sends (answering prompts).
   - *Rejected*: Default unsafe (always send) — would replicate the manual validation the operator currently does.

4. **`pane process` falls back to `sleeping` on ambiguous state**
   - *Why*: False negatives (reporting `sleeping` when actually waiting) are safe — the operator falls back to capture-and-regex. False positives (`waiting-for-input` when not) would cause the operator to try answering non-questions.
   - *Rejected*: Returning an error on ambiguous state — would break the operator's flow and provide no actionable information.

5. **Runtime `GOOS` detection rather than build tags for platform abstraction**
   - *Why*: The platform-specific code is limited to process introspection helpers (read `/proc` vs shell out to `ps`/`lsof`). Build tags add compilation complexity for minimal benefit. A single binary with runtime detection is simpler.
   - *Rejected*: Build tags with separate `_linux.go` / `_darwin.go` files — warranted for larger platform layers, overkill here.

## Deprecated Requirements

### `fab pane-map` Top-Level Command

**Reason**: Renamed to `fab pane map` under the new `pane` parent command group. All functionality preserved; only the command path changes.
**Migration**: Replace `fab pane-map` with `fab pane map` in all scripts and documentation. No alias provided (confirmed by user).

## Assumptions

<!-- SCORING SOURCE: fab score reads only this table — intake.md assumptions are
     state transfer, not scored. This is the authoritative decision record for confidence
     scoring.

     The spec-stage agent reads intake.md's Assumptions as a starting point, then
     confirms, upgrades, or overrides each assumption based on spec-level analysis.
     New assumptions discovered during spec generation are added here.

     All four SRAD grades (Certain, Confident, Tentative, Unresolved) are recorded.
     Scores column is required for every row.
     Unresolved rows must include status context in Rationale (e.g., "Asked — user undecided"). -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Internalize into `fab-go` rather than adopt external MCP server | Confirmed from intake #1 — Constitution §I and §V, user confirmed | S:95 R:90 A:95 D:95 |
| 2 | Certain | `wait-for` recommendations dropped — not useful for LLM-driven operator | Confirmed from intake #2 — blocking primitives don't help agents | S:95 R:95 A:90 D:95 |
| 3 | Certain | `fab pane-map` renamed to `fab pane map`, no alias for old name | Confirmed from intake #3 — user explicitly confirmed | S:95 R:85 A:95 D:95 |
| 4 | Certain | All pane commands live in existing `fab-go` binary, no separate binary | Confirmed from intake #4 — shared plumbing requires same binary | S:95 R:90 A:95 D:90 |
| 5 | Certain | `tmux new-window` stays as raw tmux (no `fab pane` equivalent) | Upgraded from intake #8 Confident — spawning is clearly a different concern (multi-step sequence with worktree + deps + tmux) | S:85 R:85 A:85 D:85 |
| 6 | Confident | `fab pane send` defaults to safe mode, `--force` skips idle check | Confirmed from intake #5 — mirrors operator §3, safe-by-default is obvious | S:75 R:85 A:80 D:75 |
| 7 | Confident | `fab pane process` uses `/proc` on Linux, `ps`/`lsof` on macOS | Confirmed from intake #6 — standard approach, no alternatives | S:70 R:80 A:85 D:80 |
| 8 | Confident | `fab pane process` can reliably distinguish `waiting-for-input` from sleep | Confirmed from intake #7 — binary distinction is straightforward, graceful fallback to `sleeping` | S:70 R:85 A:70 D:70 |
| 9 | Confident | Non-fab panes treated as idle by `pane send` (no `--force` required) | Clarified — idle check is about fab agent state; non-fab panes have no runtime entry, treating them as idle avoids needless friction for utility shells | S:75 R:85 A:75 D:70 |
| 10 | Confident | Runtime `GOOS` detection rather than build tags for platform abstraction | Clarified — platform-specific code is a single function with small branches; standard Go pattern for minimal platform differences | S:70 R:90 A:75 D:70 |
| 11 | Confident | `pane capture` default line count is all visible lines (no `-l` means full pane) | New — mirrors `tmux capture-pane` default behavior; agents typically want all visible content unless explicitly limiting | S:70 R:90 A:80 D:75 |

11 assumptions (5 certain, 6 confident, 0 tentative, 0 unresolved).
