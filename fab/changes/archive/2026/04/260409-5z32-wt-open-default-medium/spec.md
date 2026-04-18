# Spec: wt open — Support "default" as App Value

**Change**: 260409-5z32-wt-open-default-medium
**Created**: 2026-04-09
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## Worktree: Default App Resolution

### Requirement: ResolveDefaultApp Helper

A new exported function `ResolveDefaultApp(apps []AppInfo) (*AppInfo, error)` SHALL be added to `src/go/wt/internal/worktree/apps.go`. It SHALL call `DetectDefaultApp(apps)` and return the resolved `AppInfo`, or an error if the index is out of range (< 1 or > len(apps)).

#### Scenario: Default resolves in tmux session
- **GIVEN** the user is in a tmux session (not byobu)
- **AND** `BuildAvailableApps()` includes `tmux_window`
- **WHEN** `ResolveDefaultApp(apps)` is called
- **THEN** it returns the `AppInfo` with `Cmd == "tmux_window"` and no error

#### Scenario: No default detected
- **GIVEN** `DetectDefaultApp(apps)` returns -1
- **WHEN** `ResolveDefaultApp(apps)` is called
- **THEN** it returns nil and an error `"no default app detected"`

### Requirement: wt open --app default

When `wt open --app` receives the value `"default"`, the command SHALL resolve the app via `ResolveDefaultApp()` instead of `ResolveApp()`. On success, it SHALL call `SaveLastApp` and `OpenInApp` with the resolved app. On failure (no default detected), it SHALL exit with an error message indicating no default app could be detected.

The `"default"` check MUST occur before the `ResolveApp()` call in the `--app` code path (`src/go/wt/cmd/open.go` lines 81-101).

#### Scenario: --app default in VSCode terminal
- **GIVEN** the user is in a VSCode integrated terminal (`TERM_PROGRAM=vscode`)
- **AND** VSCode (`code`) is available
- **WHEN** the user runs `wt open --app default my-worktree`
- **THEN** the worktree opens in VSCode
- **AND** `SaveLastApp("code")` is called
- **AND** no interactive menu is shown

#### Scenario: --app default with no detectable medium
- **GIVEN** `TERM_PROGRAM` is unset, no tmux/byobu session, and no cached last-app
- **AND** no apps are available except `open_here`
- **WHEN** the user runs `wt open --app default my-worktree`
- **THEN** the command exits with an error: `"No default app detected"`
- **AND** the hint suggests using `wt open` (without `--app`) to see the menu

### Requirement: wt create --worktree-open default

When `wt create --worktree-open` receives the value `"default"`, the command SHALL resolve the app via `ResolveDefaultApp()` instead of `ResolveApp()`. On success, it SHALL call `SaveLastApp` and `OpenInApp`. On failure (no default detected), it SHALL print a warning to stderr and continue (non-fatal, matching the existing `--worktree-open` error pattern in `create.go` line 277).

The `"default"` check MUST be handled as a distinct branch alongside the existing `"prompt"` and `"skip"` checks in `create.go` (lines 253-287).

#### Scenario: --worktree-open default in tmux
- **GIVEN** the user is in a tmux session
- **WHEN** the user runs `wt create --worktree-open default feature-branch`
- **THEN** a worktree is created for `feature-branch`
- **AND** it opens in a new tmux window
- **AND** `SaveLastApp("tmux_window")` is called

#### Scenario: --worktree-open default falls back gracefully
- **GIVEN** `ResolveDefaultApp()` returns an error
- **WHEN** the user runs `wt create --worktree-open default`
- **THEN** a warning is printed to stderr: `"Warning: no default app detected"`
- **AND** worktree creation proceeds normally (non-fatal)
- **AND** the worktree path is still printed to stdout

### Requirement: Keyword Semantics

The keyword `"default"` SHALL be case-sensitive and lowercase, consistent with existing keywords `"skip"` and `"prompt"`. The value `"Default"`, `"DEFAULT"`, or any other casing SHALL be treated as an app name and resolved via `ResolveApp()` (which will fail, producing the standard "not found" error).

#### Scenario: Case sensitivity
- **GIVEN** the user passes `--app Default` (capitalized)
- **WHEN** the command processes the flag
- **THEN** it is NOT treated as the default keyword
- **AND** `ResolveApp("Default", apps)` is called (likely fails with "not found")

## Design Decisions

1. **Error behavior differs between `wt open` and `wt create`**: `wt open --app default` errors on no-default because the user explicitly asked to open something — a silent no-op would be confusing. `wt create --worktree-open default` warns and continues because the open step is secondary to worktree creation — failing the entire create for an open failure would be disruptive.
   - *Why*: Matches existing error patterns in each command — `open.go` uses `ExitWithError`, `create.go` uses `fmt.Fprintf(os.Stderr, "Warning: ...")`.
   - *Rejected*: Uniform error behavior — would either make `create` too strict or `open` too lenient.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Keyword is `"default"` (case-sensitive, lowercase) | Confirmed from intake #1 — consistent with `"skip"` and `"prompt"` | S:85 R:95 A:90 D:90 |
| 2 | Certain | Reuse existing `DetectDefaultApp()` logic | Confirmed from intake #2 — correct priority chain, no duplication | S:90 R:95 A:95 D:95 |
| 3 | Certain | `SaveLastApp` is called when `"default"` resolves | Confirmed from intake #3 — consistent with menu selection behavior | S:80 R:95 A:90 D:90 |
| 4 | Certain | `wt create` warns on no-default; `wt open` errors on no-default | Upgraded from intake #4 (Confident → Certain) — verified by reading both code paths: `create.go:277` uses warning pattern, `open.go:85` uses ExitWithError | S:90 R:90 A:95 D:90 |
| 5 | Certain | `--app` flag name is correct as-is | Confirmed from intake #5 — user-confirmed decision | S:95 R:95 A:95 D:95 |
| 6 | Certain | Extract `ResolveDefaultApp()` as shared helper in `apps.go` | Upgraded from intake #6 (Confident → Certain) — both callers need identical logic, single function avoids divergence | S:85 R:90 A:90 D:90 |

6 assumptions (6 certain, 0 confident, 0 tentative, 0 unresolved).
