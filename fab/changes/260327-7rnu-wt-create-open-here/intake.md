# Intake: wt create "Open Here" Option

**Change**: 260327-7rnu-wt-create-open-here
**Created**: 2026-03-27
**Status**: Draft

## Origin

> Backlog item [7rnu]: "Earlier wt couldn't show 'open here' as one of the options because of a shell limitation. Now wt-create is no longer a shell script but a go binary. Can this be done now?"

One-shot request from backlog. The original shell-based `wt-create` script couldn't offer an "open here" option because a child shell process cannot change the parent shell's working directory. Now that `wt` is a Go binary, the question is whether this limitation can be overcome.

## Why

When a user runs `wt create`, they're presented with a menu of apps to open the new worktree in (VSCode, Cursor, Ghostty, tmux window, etc.). A common desire is to simply `cd` into the new worktree in the **current** terminal session — "open here." The old shell script architecture made this impossible because `wt-open` ran as a subprocess and couldn't modify the parent shell's working directory.

If we don't add this, users who just want to work in their current terminal must manually copy/type the worktree path and `cd` into it after creation. This is friction in the most basic workflow.

The Go binary has the same fundamental POSIX limitation — a child process cannot change the parent shell's `$PWD`. However, Go can work around this by printing a `cd` command that the user's shell evaluates, using the standard shell-function wrapper pattern (e.g., `eval "$(wt create ...)"` or a shell function that captures stdout). The worktree path is already printed as the last line of stdout — the mechanism is nearly in place.

## What Changes

### 1. Add "Open here" to the app menu

In `src/go/wt/internal/worktree/apps.go`, add an `AppInfo{"Open here", "open_here"}` entry to `BuildAvailableApps()`. This option should always be available (no detection needed — it works in any terminal). Place it at the top of the list before editors, since it's the most lightweight option.

### 2. Handle "open_here" in `OpenInApp()`

Add a case for `"open_here"` in the `OpenInApp()` switch in `apps.go`. The behavior: print a shell-evaluable `cd` command to stdout. The exact mechanism:

```go
case "open_here":
    // Print cd command for shell function wrapper to eval
    fmt.Printf("cd %q\n", path)
    return nil
```

This relies on the caller having a shell wrapper (see below). Without the wrapper, the `cd` line simply prints to the terminal — harmless but ineffective. The worktree path is still printed on the final line by `create.go`, so non-wrapper users still get the path.

### 3. Shell function wrapper

Provide a shell function (documented in README or `wt --help`) that users add to their shell profile:

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

This is the standard pattern used by tools like `nvm`, `direnv`, and `z`. The function intercepts stdout: if it starts with `cd `, it evaluates it (changing the shell's directory); otherwise it passes through.

### 4. Default app detection for "open_here"

In `DetectDefaultApp()`, "open here" should NOT be the default — users who are in tmux/VSCode/Cursor likely want those as defaults. The existing detection logic is fine. "Open here" is an opt-in choice from the menu.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the shell wrapper pattern and "open here" option in the wt packages section

## Impact

- **`src/go/wt/internal/worktree/apps.go`** — new AppInfo entry, new OpenInApp case
- **`src/go/wt/cmd/create.go`** — stdout handling may need care: the `cd` line from "open_here" and the final `fmt.Println(wtPath)` both go to stdout. The shell wrapper needs to handle both lines, or the "open_here" case should suppress the final path print.
- **`src/go/wt/cmd/open.go`** — `wt open` also uses `OpenInApp`, so "open here" would appear there too (desirable — `wt open` in an existing worktree should also support `cd`-ing to it)
- **Tests** — `apps_test.go` (if exists) needs update; new test for "open_here" case
- **Documentation** — shell wrapper instructions

## Open Questions

- Should the shell wrapper be auto-installed (e.g., via `wt init-shell`) or just documented?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Go binary has same POSIX limitation as shell — child cannot change parent's PWD | Fundamental OS constraint, not language-dependent | S:90 R:95 A:95 D:95 |
| 2 | Certain | Shell function wrapper pattern is the solution | Well-established pattern (nvm, direnv, z); only known way to change parent shell directory | S:85 R:90 A:95 D:90 |
| 3 | Confident | "Open here" should always be available (no detection) | Unlike editors/terminals, `cd` works everywhere — no binary to detect | S:75 R:90 A:85 D:80 |
| 4 | Confident | Place "Open here" at the top of the menu but NOT as default | Top position for discoverability; existing context-sensitive defaults (tmux/VSCode) are better for most users | S:70 R:85 A:75 D:70 |
| 5 | Confident | stdout `cd` prefix is the communication mechanism between binary and shell wrapper | Standard pattern; alternatives (temp files, env vars) are more complex for no benefit | S:75 R:80 A:80 D:75 |
| 6 | Tentative | Shell wrapper should be documented, not auto-installed | Auto-install modifies user's shell profile — invasive. But `/wt init-shell` is common (e.g., `rbenv init`) | S:50 R:70 A:50 D:50 |
<!-- assumed: Shell wrapper documentation-only — user can opt in, no auto-modification of shell profile -->

6 assumptions (2 certain, 3 confident, 1 tentative, 0 unresolved). Run /fab-clarify to review.
