# Spec: wt open — Add tmux session option

**Change**: 260403-o8eg-wt-open-tmux-session-option
**Created**: 2026-04-03
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## wt open: tmux session app entry

### Requirement: App Detection for tmux session

The `BuildAvailableApps` function SHALL include a `tmux session` entry with command key `tmux_session` when `IsTmuxSession()` returns true. The entry MUST appear immediately after the existing `tmux_window` entry.

#### Scenario: tmux session option appears in tmux
- **GIVEN** the user is running inside a plain tmux session (`TMUX` env var set, not byobu)
- **WHEN** `BuildAvailableApps` is called
- **THEN** the returned list SHALL contain `AppInfo{"tmux session", "tmux_session"}`
- **AND** the `tmux_session` entry SHALL appear immediately after the `tmux_window` entry

#### Scenario: tmux session option absent outside tmux
- **GIVEN** the user is not running inside a tmux session (`TMUX` env var unset)
- **WHEN** `BuildAvailableApps` is called
- **THEN** the returned list SHALL NOT contain a `tmux_session` entry

#### Scenario: tmux session option absent in byobu
- **GIVEN** the user is running inside a byobu session
- **WHEN** `BuildAvailableApps` is called
- **THEN** the returned list SHALL NOT contain a `tmux_session` entry

### Requirement: Open in tmux session

The `OpenInApp` function SHALL handle the `tmux_session` command key by creating a new detached tmux session named `{repoName}-{wtName}` with its start directory set to `path`.

The implementation MUST use `tmux new-session -d -s {sessionName} -c {path}`:
- `-d` — create detached (MUST NOT attempt to attach, as this would cause a nested session error)
- `-s` — set the session name
- `-c` — set the start directory

#### Scenario: Successful session creation
- **GIVEN** the user selects "tmux session" from the app menu
- **AND** tmux is available on the system
- **WHEN** `OpenInApp` is called with `appCmd="tmux_session"`
- **THEN** a new detached tmux session SHALL be created
- **AND** the session name SHALL be `{repoName}-{wtName}`
- **AND** the session start directory SHALL be `path`
- **AND** the function SHALL return `nil`

#### Scenario: tmux not available
- **GIVEN** the user selects "tmux session"
- **AND** tmux is not on the PATH
- **WHEN** `OpenInApp` is called with `appCmd="tmux_session"`
- **THEN** the function SHALL return an error: `"tmux is not available on this system"`

#### Scenario: tmux new-session fails
- **GIVEN** the user selects "tmux session"
- **AND** tmux is available
- **WHEN** `tmux new-session` exits non-zero (e.g., duplicate session name)
- **THEN** the function SHALL return an error: `"tmux new-session failed: {stderr}"`

### Requirement: Exit code reuse

Error handling for the `tmux_session` app SHALL reuse `ExitTmuxWindowError` (exit code 6). The existing error path in `open.go` uses `strings.Contains(resolved.Cmd, "tmux")`, which naturally matches both `tmux_window` and `tmux_session`.

No new exit code constant SHALL be introduced.

#### Scenario: Exit code on tmux session failure
- **GIVEN** `OpenInApp` returns an error for `tmux_session`
- **WHEN** the error is handled in `openCmd` or `handleAppMenu`
- **THEN** the exit code SHALL be `ExitTmuxWindowError` (6)

### Requirement: Default app unchanged

The `DetectDefaultApp` function SHALL NOT be modified. When `IsTmuxSession()` is true, the default app SHALL remain `tmux_window`. The `tmux_session` option is opt-in only.

#### Scenario: Default remains tmux_window in tmux
- **GIVEN** the user is in a plain tmux session
- **AND** no `TERM_PROGRAM` env var is set
- **WHEN** `DetectDefaultApp` is called
- **THEN** the returned default SHALL be `tmux_window`, not `tmux_session`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `-d` (detached) flag for `tmux new-session` | Confirmed from intake #1 — `tmux new-session` without `-d` inside tmux causes "sessions should be nested" error. `-d` is mandatory | S:85 R:90 A:95 D:95 |
| 2 | Certain | Session name follows `repoName-wtName` pattern | Confirmed from intake #2 — consistent with `tmux_window` tab naming | S:90 R:95 A:95 D:95 |
| 3 | Certain | Same `IsTmuxSession()` guard for detection | Confirmed from intake #3 — matching the `tmux_window` guard is the only correct approach | S:90 R:90 A:95 D:95 |
| 4 | Certain | Reuse `ExitTmuxWindowError` exit code | Confirmed from intake #4 — `strings.Contains` already matches; no over-engineering | S:80 R:95 A:90 D:90 |
| 5 | Certain | Default app remains `tmux_window` when in tmux | Upgraded from intake Confident #5 — codebase confirms `DetectDefaultApp` has no reason to change; `tmux_session` is strictly opt-in | S:85 R:90 A:90 D:90 |
| 6 | Certain | Place `tmux_session` entry immediately after `tmux_window` in menu | Upgraded from intake Confident #6 — logical grouping, consistent with byobu_tab/tmux_window adjacency | S:80 R:95 A:85 D:85 |

6 assumptions (6 certain, 0 confident, 0 tentative, 0 unresolved).
