# Spec: wt create "Open Here" Option

**Change**: 260327-7rnu-wt-create-open-here
**Created**: 2026-03-27
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Auto-installing the shell wrapper into user's profile — documentation only
- Changing the default app detection logic — "open here" is opt-in from the menu
- Modifying `wt delete` or `wt list` — only `wt create` and `wt open` are affected

## App Menu: "Open here" Entry

### Requirement: "Open here" availability

`BuildAvailableApps()` SHALL include an `AppInfo{"Open here", "open_here"}` entry unconditionally (no platform detection required). The entry SHALL be placed as the **first** item in the returned list, before editors and terminals.

#### Scenario: App menu includes "Open here"

- **GIVEN** a user runs `wt create` or `wt open` in any terminal environment
- **WHEN** the app selection menu is displayed
- **THEN** "Open here" SHALL appear as the first option in the list
- **AND** all other detected apps SHALL follow in their existing order

#### Scenario: Non-interactive mode with --worktree-open=open_here

- **GIVEN** a user runs `wt create --worktree-open open_here`
- **WHEN** the worktree is created
- **THEN** the `open_here` handler is invoked directly (no menu)
- **AND** the behavior matches the interactive selection of "Open here"

### Requirement: "Open here" SHALL NOT be the default

`DetectDefaultApp()` SHALL NOT return the index of the "Open here" entry. The existing default detection logic (VSCode for `TERM_PROGRAM=vscode`, tmux window for tmux sessions, etc.) SHALL remain unchanged. "Open here" is an explicit user choice.

#### Scenario: Default detection ignores "Open here"

- **GIVEN** a user is in a tmux session
- **WHEN** `DetectDefaultApp()` is called
- **THEN** the default SHALL be "tmux window", not "Open here"
- **AND** "Open here" is available but not pre-selected

### Requirement: Last-app cache respects "Open here"

When a user selects "Open here" from the menu, `SaveLastApp("open_here")` SHALL be called. On subsequent `wt create` or `wt open` invocations where no context-based default applies, "Open here" SHALL be suggested as the default via the last-app cache.

#### Scenario: "Open here" remembered as last app

- **GIVEN** a user previously selected "Open here" from the menu
- **AND** the user is not in VSCode, Cursor, tmux, or byobu
- **WHEN** `DetectDefaultApp()` is called
- **THEN** the last-app cache returns `open_here`
- **AND** "Open here" is pre-selected in the menu

## App Handler: "open_here" Command

### Requirement: "open_here" prints cd command to stdout

`OpenInApp()` with `appCmd == "open_here"` SHALL print a single line to stdout: `cd <quoted-path>`, where `<quoted-path>` is the Go-quoted worktree path (via `%q` format verb). The function SHALL return `nil` (no error).

#### Scenario: open_here outputs cd command

- **GIVEN** a worktree at `/home/user/repo.worktrees/swift-fox`
- **WHEN** `OpenInApp("open_here", "/home/user/repo.worktrees/swift-fox", "repo", "swift-fox")` is called
- **THEN** stdout receives: `cd "/home/user/repo.worktrees/swift-fox"`
- **AND** the function returns `nil`

#### Scenario: Path with spaces is properly quoted

- **GIVEN** a worktree at `/home/user/my repo.worktrees/swift fox`
- **WHEN** `OpenInApp("open_here", ...)` is called
- **THEN** stdout receives: `cd "/home/user/my repo.worktrees/swift fox"`

### Requirement: create.go SHALL suppress final path line when open_here is selected

When the user selects "open_here" (either via menu or `--worktree-open`), `create.go` SHALL NOT print the final `fmt.Println(wtPath)` line. This prevents the shell wrapper from receiving two lines (the `cd` command and the raw path), which would break the `cd\ *` prefix check.

#### Scenario: stdout contains only cd line when open_here selected

- **GIVEN** a user selects "Open here" from the `wt create` menu
- **WHEN** the worktree is created and the app handler runs
- **THEN** stdout contains only the `cd "/path/to/worktree"` line
- **AND** the final `fmt.Println(wtPath)` is skipped

#### Scenario: stdout still shows path for other apps

- **GIVEN** a user selects "VSCode" from the `wt create` menu
- **WHEN** the worktree is created
- **THEN** stdout contains the worktree path as the last line (existing behavior preserved)

### Requirement: open.go SHALL handle open_here identically

`wt open` uses the same `OpenInApp()` function. When "open_here" is selected via `wt open`, the `cd` command SHALL be printed to stdout. Since `open.go` does not print a trailing path line, no suppression logic is needed there.

#### Scenario: wt open with open_here

- **GIVEN** a user runs `wt open` from within a worktree
- **WHEN** the user selects "Open here" from the menu
- **THEN** stdout receives: `cd "/path/to/worktree"`

## Shell Function Wrapper

### Requirement: Shell wrapper documentation

The project SHALL document a shell function wrapper that enables "Open here" to change the current shell's working directory. The wrapper SHALL be documented in the wt package README or help output.

#### Scenario: User installs wrapper and uses "Open here"

- **GIVEN** the user has added the shell wrapper function to their profile
- **WHEN** the user runs `wt create` and selects "Open here"
- **THEN** the wrapper captures the `cd` command from stdout
- **AND** evaluates it, changing the shell's working directory to the new worktree

#### Scenario: User without wrapper selects "Open here"

- **GIVEN** the user has NOT installed the shell wrapper
- **WHEN** the user runs `wt create` and selects "Open here"
- **THEN** the `cd` command is printed to the terminal as text
- **AND** no error occurs — the behavior is harmless but ineffective

### Requirement: Wrapper function specification

The shell wrapper SHALL follow this pattern:

```bash
wt() {
    local output
    output="$(command wt "$@")"
    local rc=$?
    if [[ "$output" == cd\ * ]]; then
        eval "$output"
    else
        printf '%s\n' "$output"
    fi
    return $rc
}
```

The wrapper SHALL:
- Capture all stdout from the `wt` binary
- Check if stdout starts with `cd ` (literal prefix)
- If yes: `eval` the output (executing the `cd` command in the current shell)
- If no: pass through stdout unchanged
- Preserve the exit code of the `wt` binary

## Design Decisions

1. **Shell function wrapper over other IPC mechanisms**
   - *Why*: Only known way to change parent shell's working directory. Standard pattern used by `nvm`, `direnv`, `z`, `rbenv`. Zero dependencies, universal compatibility.
   - *Rejected*: Temp files (`~/.cache/wt/last-cd`) — adds complexity, race conditions. Environment variables — cannot be set cross-process. Named pipes — overcomplicated for a `cd`.

2. **`cd` prefix as the communication protocol**
   - *Why*: Simple, unambiguous, human-readable. The wrapper can detect it with a single string prefix check.
   - *Rejected*: JSON envelope — requires parsing. Exit code signaling — too fragile, exit codes have other meanings. Stderr channel — stderr is for errors/warnings, not data.

3. **Suppress final path line in create.go when open_here is active**
   - *Why*: The shell wrapper checks `output == cd\ *`. If the output contains both `cd "/path"` and `/path` on a second line, the prefix check fails. Suppressing the path line keeps the protocol clean.
   - *Rejected*: Making the wrapper handle multi-line output — adds complexity to the wrapper for no benefit. The path is already embedded in the `cd` command.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Go binary has same POSIX limitation — child cannot change parent's PWD | Confirmed from intake #1 — fundamental OS constraint | S:90 R:95 A:95 D:95 |
| 2 | Certain | Shell function wrapper is the only viable mechanism | Confirmed from intake #2 — well-established pattern, no alternatives | S:85 R:90 A:95 D:90 |
| 3 | Certain | "Open here" always available, no detection needed | Upgraded from intake #3 — `cd` is a shell builtin, always works | S:85 R:90 A:90 D:90 |
| 4 | Confident | Place "Open here" first in menu but not as default | Confirmed from intake #4 — existing defaults are context-aware | S:70 R:85 A:75 D:70 |
| 5 | Certain | stdout `cd` prefix as communication protocol | Upgraded from intake #5 — spec-level analysis confirms simplicity | S:80 R:85 A:85 D:85 |
| 6 | Certain | Suppress final path line when open_here is active | New — spec analysis revealed stdout conflict between cd line and path line | S:85 R:80 A:90 D:90 |
| 7 | Confident | Shell wrapper documented, not auto-installed | Confirmed from intake #6 — documentation is sufficient, auto-install is invasive | S:60 R:75 A:60 D:65 |
| 8 | Certain | open.go needs no suppression — it doesn't print a trailing path | New — code review of open.go confirms no trailing path output | S:90 R:95 A:95 D:95 |

8 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
