# Spec: Pane Map JSON Session Flags

**Change**: 260313-wrt4-pane-map-json-session-flags
**Created**: 2026-03-13
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Pane Map: New Flags and Output Modes

### Requirement: JSON Output Flag

`fab pane-map` SHALL accept a `--json` boolean flag that switches output from the aligned table to a JSON array.

Each JSON element SHALL contain the fields: `session`, `window_index`, `pane`, `tab`, `worktree`, `change`, `stage`, `agent_state`, `agent_idle_duration`.

Fields that are `—` (em dash) in table mode SHALL be `null` in JSON output. `(no change)` SHALL also be `null` for the `change` field. `agent_idle_duration` SHALL be `null` when the agent is not idle, and a string duration (e.g., `"2m"`) when idle.

When `--json` is not set, output SHALL remain the current aligned table format (no regression).

#### Scenario: JSON output for active pane
- **GIVEN** a tmux session with one pane running a fab worktree at apply stage with an active agent
- **WHEN** `fab pane-map --json` is invoked
- **THEN** output is a valid JSON array with one element
- **AND** the element contains `"stage": "apply"`, `"agent_state": "active"`, `"agent_idle_duration": null`

#### Scenario: JSON null semantics for non-fab pane
- **GIVEN** a tmux pane in a non-git directory (e.g., `/tmp`)
- **WHEN** `fab pane-map --json` is invoked
- **THEN** the pane's element has `"change": null`, `"stage": null`, `"agent_state": null`

#### Scenario: JSON output with idle agent
- **GIVEN** a pane with an idle agent (idle for 5 minutes)
- **WHEN** `fab pane-map --json` is invoked
- **THEN** `"agent_state": "idle"` and `"agent_idle_duration": "5m"`

#### Scenario: Default table output unchanged
- **GIVEN** any tmux session
- **WHEN** `fab pane-map` is invoked without `--json`
- **THEN** output is the aligned table format (existing behavior, no regression)

### Requirement: Session Targeting Flag

`fab pane-map` SHALL accept a `--session <name>` string flag that targets a specific tmux session by name.

When `--session` is provided, the command SHALL use `tmux list-panes -s -t <name>` to discover panes, and the `$TMUX` environment variable check SHALL be skipped.

#### Scenario: Target specific session by name
- **GIVEN** two tmux sessions named "runK" and "dev"
- **WHEN** `fab pane-map --session runK` is invoked from outside tmux
- **THEN** output includes only panes from the "runK" session
- **AND** the command succeeds even though `$TMUX` is not set

#### Scenario: Invalid session name
- **GIVEN** no tmux session named "nonexistent"
- **WHEN** `fab pane-map --session nonexistent` is invoked
- **THEN** the command exits with an error from tmux (session not found)

### Requirement: All-Sessions Flag

`fab pane-map` SHALL accept an `--all-sessions` boolean flag that queries all tmux sessions.

When `--all-sessions` is set, the command SHALL enumerate sessions via `tmux list-sessions`, then run `tmux list-panes -s -t <session>` for each. The `$TMUX` check SHALL be skipped.

`--session` and `--all-sessions` SHALL be mutually exclusive — the command SHALL exit with an error if both are provided.

When neither `--session` nor `--all-sessions` is set, current behavior SHALL be preserved: use `$TMUX` to discover the current session only.

#### Scenario: Query all sessions
- **GIVEN** two tmux sessions each with panes
- **WHEN** `fab pane-map --all-sessions` is invoked
- **THEN** output includes panes from both sessions, each with a `session` field identifying its session

#### Scenario: Mutual exclusion
- **GIVEN** any tmux state
- **WHEN** `fab pane-map --session runK --all-sessions` is invoked
- **THEN** the command exits with an error indicating the flags are mutually exclusive

### Requirement: Window Index Field

`fab pane-map` output SHALL include a `window_index` integer field.

In table mode, a `WinIdx` column SHALL appear between `Pane` and `Tab`.

In JSON mode, the field SHALL be `window_index` (integer).

The value SHALL come from the tmux format variable `#{window_index}`.

#### Scenario: Window index in table output
- **GIVEN** a pane in window index 3
- **WHEN** `fab pane-map` is invoked
- **THEN** the `WinIdx` column shows `3` for that row

#### Scenario: Window index in JSON output
- **GIVEN** a pane in window index 0
- **WHEN** `fab pane-map --json` is invoked
- **THEN** the element contains `"window_index": 0`

### Requirement: Session Name Field

`fab pane-map` output SHALL include the session name.

In table mode, a `Session` column SHALL appear as the first column ONLY when `--all-sessions` is used. In single-session mode (default or `--session`), the `Session` column SHALL be omitted to avoid noise.

In JSON mode, the `session` field SHALL always be present regardless of session targeting mode.

#### Scenario: Session column omitted in single-session table mode
- **GIVEN** the default single-session mode
- **WHEN** `fab pane-map` is invoked
- **THEN** the table header does NOT contain a "Session" column

#### Scenario: Session column present in all-sessions table mode
- **GIVEN** `--all-sessions` mode
- **WHEN** `fab pane-map --all-sessions` is invoked
- **THEN** the table header includes "Session" as the first column

#### Scenario: Session in JSON always present
- **GIVEN** default single-session mode
- **WHEN** `fab pane-map --json` is invoked
- **THEN** every element contains a `session` field with the session name

### Requirement: Struct and Discovery Changes

The `paneEntry` struct SHALL be extended with `session` (string) and `index` (int) fields.

The `paneRow` struct SHALL be extended with `session` (string) and `windowIndex` (int) fields.

The `discoverPanes()` function SHALL accept a session targeting parameter and extend the tmux format string to include `#{session_name}` and `#{window_index}`.

The `$TMUX` guard in `runPaneMap()` SHALL only apply when neither `--session` nor `--all-sessions` is provided.

#### Scenario: discoverPanes with explicit session
- **GIVEN** a session name "runK"
- **WHEN** `discoverPanes` is called with session targeting for "runK"
- **THEN** it executes `tmux list-panes -s -t runK` with the extended format string

#### Scenario: discoverPanes in default mode
- **GIVEN** no session targeting flags
- **WHEN** `discoverPanes` is called
- **THEN** it executes `tmux list-panes -s` (current session, existing behavior) with the extended format string

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `encoding/json` for `--json` output | Confirmed from intake #1 — Go stdlib, no external dependency needed | S:90 R:95 A:95 D:95 |
| 2 | Certain | `--session` and `--all-sessions` are mutually exclusive | Confirmed from intake #2 — clear semantics, standard CLI pattern | S:85 R:90 A:90 D:95 |
| 3 | Certain | Null fields in JSON for em-dash/no-change values | Confirmed from intake #3 — clean machine-readable semantics | S:85 R:90 A:90 D:90 |
| 4 | Certain | `window_index` from `#{window_index}` tmux format var | Confirmed from intake #4 — standard tmux variable | S:80 R:85 A:85 D:90 |
| 5 | Confident | Session column only in table mode with `--all-sessions` | Confirmed from intake #5 — avoids noise in default output | S:70 R:90 A:75 D:80 |
| 6 | Confident | JSON field names use snake_case | Confirmed from intake #6 — matches Go JSON tag conventions | S:75 R:85 A:80 D:80 |
| 7 | Certain | `agent_idle_duration` as separate field from `agent_state` | Confirmed from intake #7 — run-kit needs separate fields | S:90 R:85 A:90 D:90 |
| 8 | Certain | JSON `agent_state` values are `"active"`, `"idle"`, `null`, `"unknown"` | Maps from table values: `active` → `"active"`, `idle (Nm)` → `"idle"`, `—` → `null`, `?` → `"unknown"` | S:80 R:85 A:85 D:80 |
| 9 | Confident | `discoverPanes` accepts a mode parameter rather than separate functions | Single function with mode keeps logic centralized; current code already has one discovery function | S:70 R:85 A:80 D:75 |

9 assumptions (6 certain, 3 confident, 0 tentative, 0 unresolved).
