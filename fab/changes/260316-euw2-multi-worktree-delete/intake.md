# Intake: Multi-Worktree Delete

**Change**: 260316-euw2-multi-worktree-delete
**Created**: 2026-03-16
**Status**: Draft

## Origin

> Allow deleting multiple worktrees at once by allowing specifying multiple worktree names. Please explore.

One-shot request with "explore" qualifier — user wants the design explored before committing to an approach.

## Why

Currently `wt delete` can remove exactly one worktree at a time (via `--worktree-name`) or all of them (`--delete-all`). There's no middle ground — if you have 5 worktrees and want to delete 3 specific ones, you must run `wt delete` three times. This is friction in the common cleanup scenario where multiple completed changes need to be torn down.

If unchanged, users continue running `wt delete --worktree-name X` in a loop or resorting to `--delete-all` when they only want a subset removed.

## What Changes

### Positional arguments for worktree names

Change the Cobra command from `cobra.NoArgs` to accept positional arguments as worktree names:

```
wt delete [name1] [name2] [name3] [flags]
```

When one or more positional args are provided, they are treated as worktree names to delete. This replaces the `--worktree-name` flag as the primary way to specify targets.

### `--worktree-name` flag behavior

Deprecate `--worktree-name` in favor of positional args. For backward compatibility, `--worktree-name X` continues to work and is equivalent to `wt delete X`. Mixing `--worktree-name` with positional args is an error.

### Multi-delete flow

When multiple names are provided:

1. **Validate all names upfront** — resolve each name against `listWorktreeEntries()`. If any name doesn't match, print all unresolved names and exit with an error (fail-fast, no partial deletes from typos).
2. **Display summary** — list all worktrees that will be deleted (name, branch, path).
3. **Single confirmation prompt** (interactive mode) — "Delete these N worktrees?" rather than prompting per-worktree.
4. **Sequential deletion** — delete each worktree in order. On per-worktree failure, print a warning and continue (same pattern as `handleDeleteAll`).
5. **Branch cleanup** — per-worktree, controlled by existing `--delete-branch` / `--delete-remote` flags.

### Non-interactive mode

`wt delete --non-interactive name1 name2 name3` skips the confirmation prompt and proceeds directly with deletion, same as the single-name case today.

### `--stash` flag interaction

When `--stash` is set, stash uncommitted changes in each worktree before deletion (same per-worktree stash logic as `handleDeleteAll` should use).

### Resolution order update

The resolution order in `RunE` becomes:

1. `--delete-all` → `handleDeleteAll()` (unchanged)
2. Positional args (1+) → new `handleDeleteMultiple()` (also used for single positional arg)
3. `--worktree-name` set → `handleDeleteByName()` (deprecated path, unchanged behavior)
4. In a worktree → `handleDeleteCurrent()` (unchanged)
5. Non-interactive, no name → error (unchanged)
6. Interactive, no name → `handleDeleteMenu()` (unchanged)

### Source files affected

- `src/go/wt/cmd/delete.go` — command definition, new `handleDeleteMultiple()`, deprecation of `--worktree-name`
- `src/go/wt/cmd/delete_test.go` — tests for multi-name deletion, backward compat for `--worktree-name`

## Affected Memory

- `fab-workflow/distribution`: (modify) Update wt package command signatures if documented

## Impact

- **wt binary** — `delete` subcommand argument parsing changes. Existing `--worktree-name` usage continues to work (backward compatible).
- **Operator scripts** — `batch-fab-archive-change` and similar may call `wt delete --worktree-name`. These continue working via the deprecated flag path.
- **docs/specs/packages.md** — needs updated `wt delete` documentation.

## Open Questions

- Should the interactive menu (`handleDeleteMenu`) support multi-select (e.g., checkboxes) in addition to the positional-arg approach? This would be a larger UX change.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use positional args (not repeated `--worktree-name` flags) | Positional args are the standard Cobra pattern for multiple values of the same type. Consistent with `git worktree remove` which accepts a path argument | S:70 R:85 A:90 D:90 |
| 2 | Certain | Fail-fast on unresolved names before deleting anything | Partial deletes from typos would be confusing and hard to recover from. Validate-then-execute is the safe pattern | S:75 R:70 A:95 D:95 |
| 3 | Confident | Deprecate `--worktree-name` but keep it working | Breaking existing scripts is worse than carrying a deprecated flag. The flag can be removed in a future major version | S:60 R:60 A:80 D:80 |
| 4 | Confident | Single confirmation prompt for multi-delete (not per-worktree) | Per-worktree prompts defeat the purpose of batch deletion. `handleDeleteAll` already uses a single prompt for N worktrees | S:65 R:80 A:85 D:85 |
| 5 | Confident | Sequential deletion with continue-on-error | Matches `handleDeleteAll` behavior. Parallel deletion would complicate error handling and output for minimal time savings | S:60 R:85 A:80 D:75 |
| 6 | Tentative | Do not add multi-select to the interactive menu | Multi-select checkbox UI is a significant UX addition beyond the scope of "allow specifying multiple names." Can be added later if needed | S:50 R:80 A:60 D:55 |
<!-- assumed: Interactive menu multi-select deferred — user said "specifying multiple worktree names" which implies CLI args, not menu UX -->

6 assumptions (2 certain, 3 confident, 1 tentative, 0 unresolved).
