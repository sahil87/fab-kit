# Intake: wt open Shell Setup

**Change**: 260403-24ic-wt-open-shell-setup
**Created**: 2026-04-03
**Status**: Draft

## Origin

> Fix wt open Open Here cd mechanism — the Open here option prints a cd command as text instead of actually changing the shell directory. Add a wt shell-setup subcommand that outputs a shell wrapper function for eval in the user's shell profile, following the direnv/rbenv/mise pattern. Detect wrapper presence via WT_WRAPPER=1 env var, show stderr hint when wrapper not detected.

## Why

The `wt open` "Open here" option currently prints `cd -- "/path/to/worktree"` to stdout, which the user sees as text output rather than the shell actually changing directories. This is a fundamental Unix constraint — a child process cannot modify its parent shell's working directory. The workaround is a shell wrapper function that intercepts the `cd` output and evals it, but today this wrapper is only documented in `wt --help` long text and requires manual copy-paste into the user's shell profile.

Tools like direnv, rbenv, and mise solve the same problem with a `shell-setup` or `init` subcommand that outputs a shell function for `eval`. This is a well-understood pattern that users expect. Without it, "Open here" appears broken on first use — users select it, see a `cd` command printed, and nothing happens.

## What Changes

### 1. New `wt shell-setup` subcommand

Add a new cobra subcommand `shell-setup` to the wt binary that outputs a shell wrapper function suitable for `eval` in the user's shell profile.

**Usage pattern** (added to `~/.zshrc` or `~/.bashrc`):
```bash
eval "$(wt shell-setup)"
```

**Output** (the shell function printed to stdout):
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

The function wraps the real `wt` binary: it captures all stdout output line-by-line, prints it through, and if the last line starts with `cd `, evals it in the calling shell context. The `export WT_WRAPPER=1` line allows the binary to detect that the wrapper is active.

Shell detection: read `$SHELL` basename to determine `bash` vs `zsh`. Both use the same function body (bash 4+ and zsh both support process substitution). If the shell is unrecognized, output the bash/zsh version with a stderr warning.

### 2. WT_WRAPPER environment variable detection

When `open_here` is selected (in both `wt open` and `wt create`) and `WT_WRAPPER` is not set to `"1"`, print a hint to stderr:

```
hint: "Open here" requires the shell wrapper to cd. Run: eval "$(wt shell-setup)"
      Add it to your ~/.zshrc or ~/.bashrc to make it permanent.
```

This hint goes to stderr so it doesn't interfere with the `cd` command on stdout (which still works if the user has manually set up the wrapper or is piping output).

When `WT_WRAPPER=1` is set, no hint is printed — the wrapper is in place and will eval the `cd` command.

### 3. Update help text in main.go

Replace the current inline shell function in the `wt` root command's `Long` description with a reference to `wt shell-setup`:

```
Shell wrapper (recommended):
  To enable the "Open here" menu option (cd into a worktree in the current
  shell), add this to your shell profile (~/.bashrc or ~/.zshrc):

    eval "$(wt shell-setup)"
```

### 4. Update docs/specs/packages.md

Update the "Why wt-open Cannot cd" section to reference `wt shell-setup` as the recommended solution, replacing the manual function copy-paste.

## Affected Memory

- `fab-workflow/distribution`: (modify) Document `wt shell-setup` as part of wt setup instructions

## Impact

- **`src/go/wt/cmd/`** — new `shell_setup.go` file for the subcommand
- **`src/go/wt/cmd/main.go`** — add `shellSetupCmd()` to root command, update help text
- **`src/go/wt/internal/worktree/apps.go`** — add `WT_WRAPPER` check in `open_here` case
- **`src/go/wt/internal/worktree/apps_test.go`** — new tests for wrapper detection hint
- **`docs/specs/packages.md`** — update "Open here" documentation

## Open Questions

(none — the description is specific and the direnv/rbenv/mise pattern is well-established)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use bash/zsh process substitution syntax for the wrapper | Both are the only shells documented in the existing help text; the current wrapper already uses this syntax | S:90 R:90 A:95 D:95 |
| 2 | Certain | Output the same wrapper function body already in main.go help | The existing function is tested and correct; no reason to change it | S:95 R:95 A:95 D:95 |
| 3 | Certain | Use `WT_WRAPPER=1` as the env var name and value | Explicitly specified in the user's description | S:95 R:90 A:90 D:95 |
| 4 | Certain | Hint goes to stderr, cd command stays on stdout | Specified in description; required so wrapper eval still works | S:90 R:85 A:95 D:95 |
| 5 | Confident | Detect shell via `$SHELL` basename, fall back to bash/zsh syntax with warning | direnv/rbenv pattern; fish/tcsh could differ but user specified bash/zsh pattern | S:70 R:85 A:80 D:70 |
| 6 | Confident | Show hint on every open_here invocation without wrapper (no "show once" suppression) | Keeps implementation simple; hint is short and goes to stderr | S:65 R:90 A:75 D:70 |
| 7 | Certain | Add shell-setup as a new cobra subcommand (not a flag) | User specified "subcommand"; matches direnv/rbenv/mise pattern | S:95 R:90 A:90 D:95 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
