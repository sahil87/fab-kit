# Intake: wt-open cd in current shell

**Change**: 260223-ufk6-wt-open-cd-current-shell
**Created**: 2026-02-23
**Status**: Draft

## Origin

> "In wt-open, have an option to cd in the current shell session itself"

One-shot request. The user clarified that they mean specifically the "wt-open" part of `wt-create` — i.e., the app-selection step where you choose VSCode, Ghostty, tmux window, etc. They want a "cd here" option that changes the working directory of the **calling** shell, not just opening a new window or tab.

## Why

After creating or selecting a worktree, users often want to navigate to it in their current terminal session rather than spawning a new window/tab/app. Currently the closest option is "Copy path" (clipboard), which still requires manual `cd` + paste. This adds friction to the most common workflow: create a worktree, then start working in it from the same terminal.

`wt-create` already outputs the worktree path as its last stdout line, so `cd "$(wt-create --non-interactive)"` is technically possible — but it's clunky, requires `--non-interactive`, and doesn't work for `wt-open` (opening an existing worktree).

## What Changes

### 1. New `cd` app type in `wt-open`

Add "cd here" as a recognized app in `wt-open`'s app detection system. When selected (via menu or `--app cd`), `wt-open` prints the resolved worktree path to stdout and exits — it does NOT attempt to `cd` itself, because a child process cannot change the parent shell's working directory.

In the interactive menu, "cd here" should appear as an option (always available — no detection needed). When selected, it prints the path to stdout.

### 2. Shell function wrapper for sourced cd

Since the script itself can't `cd` the parent shell, provide a shell function that wraps `wt-open --app cd` and performs the actual `cd`. Two approaches to evaluate:

**Option A — function in `env-packages.sh`**: Add a `wt-cd()` function to the existing `env-packages.sh` that is already sourced during workspace setup:

```bash
wt-cd() {
    local path
    path=$(wt-open --app cd "$@") || return $?
    cd "$path" || return $?
}
```

**Option B — separate sourceable file**: Create `fab/.kit/packages/wt/lib/wt-functions.sh` that defines `wt-cd()`, sourced by `env-packages.sh`.

### 3. Integration with `wt-create`

`wt-create` already passes through to `wt-open` at the end (line 316–318 of `wt-create`). When `--worktree-open cd` is passed to `wt-create`, it should flow through to `wt-open --app cd`, and the printed path enables the same `wt-cd` function pattern:

```bash
wt-cd() {
    # If arguments look like wt-create args, delegate to wt-create
    # Otherwise delegate to wt-open
    ...
}
```

Or keep it simple: `wt-cd` only wraps `wt-open` for navigation to existing worktrees; for creation, users use `eval "$(wt-create --non-interactive)"` or the function handles it.

## Affected Memory

- `fab-workflow/distribution`: (modify) Document the new `wt-cd` shell function and how `env-packages.sh` provides it

## Impact

- **`fab/.kit/packages/wt/bin/wt-open`**: New `cd` app type in `wt_build_available_apps()` and `wt_open_in_app()`
- **`fab/.kit/scripts/lib/env-packages.sh`** (or new `wt-functions.sh`): Shell function definition
- **`src/packages/wt/tests/wt-open.bats`**: Tests for the new app type
- **`docs/specs/packages.md`**: Document the new option

## Open Questions

- Should `wt-cd` also handle worktree creation (wrapping `wt-create`), or stay focused on navigation only?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Child process cannot cd the parent — must use a shell function wrapper | Fundamental shell behavior; no alternative | S:90 R:95 A:95 D:95 |
| 2 | Confident | The function should be named `wt-cd` | Follows the `wt-*` naming convention; short, discoverable | S:70 R:90 A:80 D:65 |
| 3 | Confident | "cd here" option should always be available in the menu (no detection needed) | Unlike editors/terminals, cd doesn't require external tools — it works everywhere | S:75 R:90 A:85 D:80 |
| 4 | Tentative | Shell function goes in `env-packages.sh` rather than a separate file | Simplest approach — `env-packages.sh` is already sourced and handles PATH setup; but mixing PATH additions with function definitions may be a concern | S:55 R:85 A:60 D:50 |
<!-- assumed: Shell function location — env-packages.sh is already sourced, simplest integration point -->
| 5 | Confident | `wt-open --app cd` prints only the path to stdout (no decoration, no messages) | Follows the porcelain output pattern already used by `wt-create --non-interactive`; essential for eval/subshell usage | S:80 R:90 A:85 D:80 |

5 assumptions (1 certain, 3 confident, 1 tentative, 0 unresolved).
