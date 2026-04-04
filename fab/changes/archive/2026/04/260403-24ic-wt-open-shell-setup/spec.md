# Spec: wt open Shell Setup

**Change**: 260403-24ic-wt-open-shell-setup
**Created**: 2026-04-03
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Fish, tcsh, or other non-bash/zsh shell support — the existing wrapper uses bash/zsh process substitution syntax and fish would require a different function body
- Suppressing the `cd` stdout output when wrapper is absent — the cd line must always be printed for compatibility with manual wrapper setups

## wt CLI: Shell Setup Subcommand

### Requirement: wt shell-setup subcommand

The `wt` binary SHALL provide a `shell-setup` subcommand that outputs a shell wrapper function to stdout. The output SHALL be suitable for `eval` in the user's shell profile (e.g., `eval "$(wt shell-setup)"`).

The output MUST include:
1. A shell function named `wt` that wraps the real `wt` binary
2. An `export WT_WRAPPER=1` statement

The wrapper function SHALL:
- Capture all stdout from `command wt "$@"` line-by-line
- Print each captured line to stdout
- If the last line of output starts with `cd `, eval it in the calling shell context
- Preserve the exit code from the underlying `wt` binary

#### Scenario: Default output (bash/zsh)
- **GIVEN** the user's `$SHELL` basename is `bash` or `zsh`
- **WHEN** `wt shell-setup` is executed
- **THEN** stdout contains the wrapper function using process substitution (`< <(command wt "$@")`)
- **AND** stdout ends with `export WT_WRAPPER=1`
- **AND** exit code is 0

#### Scenario: Unrecognized shell
- **GIVEN** the user's `$SHELL` basename is not `bash` or `zsh` (e.g., `fish`, `tcsh`)
- **WHEN** `wt shell-setup` is executed
- **THEN** stdout contains the bash/zsh wrapper function (same as default)
- **AND** stderr contains a warning: `warning: unsupported shell "{shell}" — outputting bash/zsh wrapper`
- **AND** exit code is 0

#### Scenario: SHELL unset
- **GIVEN** `$SHELL` is empty or unset
- **WHEN** `wt shell-setup` is executed
- **THEN** stdout contains the bash/zsh wrapper function (same as default)
- **AND** no warning is printed to stderr
- **AND** exit code is 0

### Requirement: Wrapper function correctness

The wrapper function output by `wt shell-setup` SHALL produce the following exact text:

```bash
wt() {
  local line last rc
  while IFS= read -r line; do
    printf '%s\n' "$line"
    last=$line
  done < <(command wt "$@")
  rc=$?
  if [[ "$last" == cd\ * ]]; then
    eval "$last"
  fi
  return $rc
}
export WT_WRAPPER=1
```

This matches the function already documented in the `wt` root command's long help text, with the addition of `export WT_WRAPPER=1`.

#### Scenario: Function wraps binary correctly
- **GIVEN** the wrapper function is sourced in a bash/zsh shell
- **WHEN** the user runs `wt open` and selects "Open here"
- **THEN** the `cd` command is eval'd in the calling shell
- **AND** the shell's working directory changes to the worktree path

## wt CLI: Wrapper Detection

### Requirement: WT_WRAPPER environment variable detection

When `open_here` is selected in the app menu (via `OpenInApp`), the function SHALL check `os.Getenv("WT_WRAPPER")`. If the value is not `"1"`, a hint SHALL be printed to stderr before the `cd` command is printed to stdout.

#### Scenario: Wrapper not installed
- **GIVEN** `WT_WRAPPER` is not set (or set to a value other than `"1"`)
- **WHEN** `OpenInApp` is called with `appCmd == "open_here"`
- **THEN** stderr contains: `hint: "Open here" requires the shell wrapper to cd. Run: eval "$(wt shell-setup)"`
- **AND** stderr contains: `      Add it to your ~/.zshrc or ~/.bashrc to make it permanent.`
- **AND** stdout contains the `cd -- <path>` command as before
- **AND** the function returns nil

#### Scenario: Wrapper installed
- **GIVEN** `WT_WRAPPER` is set to `"1"`
- **WHEN** `OpenInApp` is called with `appCmd == "open_here"`
- **THEN** stdout contains only the `cd -- <path>` command
- **AND** no hint is printed to stderr
- **AND** the function returns nil

#### Scenario: Wrapper detection in wt create
- **GIVEN** `WT_WRAPPER` is not set
- **WHEN** the user selects "Open here" during `wt create`'s open menu
- **THEN** the same stderr hint is printed (detection is in `OpenInApp`, shared by both commands)

## wt CLI: Help Text Update

### Requirement: Updated root command help

The `wt` root command's `Long` description SHALL replace the inline shell function with a reference to `wt shell-setup`.

The help text SHALL read:

```
Shell wrapper (recommended):
  To enable the "Open here" menu option (cd into a worktree in the current
  shell), add this to your shell profile (~/.bashrc or ~/.zshrc):

    eval "$(wt shell-setup)"
```

#### Scenario: Help text shows eval instruction
- **GIVEN** the user runs `wt --help`
- **WHEN** the long help text is displayed
- **THEN** the help mentions `eval "$(wt shell-setup)"` as the setup instruction
- **AND** the help does NOT contain the inline function body

## Documentation: packages.md Update

### Requirement: Updated workaround documentation

The "Why wt-open Cannot cd" section in `docs/specs/packages.md` SHALL be updated to reference `wt shell-setup` as the recommended solution.

The "Option 3" workaround SHALL be replaced with the `eval "$(wt shell-setup)"` pattern. The explanation of why shell functions work SHALL be preserved.

#### Scenario: Documentation references shell-setup
- **GIVEN** a user reads `docs/specs/packages.md`
- **WHEN** they reach the "Workarounds" section
- **THEN** the recommended approach is `eval "$(wt shell-setup)"` in the shell profile
- **AND** the Unix process model constraint explanation is preserved

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use bash/zsh process substitution syntax for the wrapper | Confirmed from intake #1 — both shells support this; existing help text uses this syntax | S:90 R:90 A:95 D:95 |
| 2 | Certain | Output the same wrapper function body already in main.go help | Confirmed from intake #2 — function is battle-tested | S:95 R:95 A:95 D:95 |
| 3 | Certain | Use `WT_WRAPPER=1` as the env var name and value | Confirmed from intake #3 — explicitly specified by user | S:95 R:90 A:90 D:95 |
| 4 | Certain | Hint goes to stderr, cd command stays on stdout | Confirmed from intake #4 — required for wrapper eval to work | S:90 R:85 A:95 D:95 |
| 5 | Confident | Detect shell via `$SHELL` basename, fall back to bash/zsh syntax with warning | Confirmed from intake #5 — direnv/rbenv pattern; fish support explicitly excluded as non-goal | S:75 R:85 A:80 D:70 |
| 6 | Confident | Show hint on every open_here invocation without wrapper | Confirmed from intake #6 — simple, low-friction, stderr-only | S:65 R:90 A:75 D:70 |
| 7 | Certain | Add shell-setup as a new cobra subcommand | Confirmed from intake #7 — matches direnv/rbenv/mise pattern | S:95 R:90 A:90 D:95 |
| 8 | Certain | Wrapper detection is in OpenInApp, shared by wt open and wt create | Codebase shows both commands call OpenInApp for open_here — single detection point | S:85 R:90 A:95 D:90 |
| 9 | Certain | SHELL unset falls back silently to bash/zsh output | bash/zsh is the overwhelmingly common default; no point warning when there's nothing to warn about | S:80 R:95 A:85 D:90 |

9 assumptions (7 certain, 2 confident, 0 tentative, 0 unresolved).
