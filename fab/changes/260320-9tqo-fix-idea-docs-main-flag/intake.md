# Intake: Fix Idea Docs & --main Flag

**Change**: 260320-9tqo-fix-idea-docs-main-flag
**Created**: 2026-03-20
**Status**: Draft

## Origin

> Backlog item [9tqo] 2026-03-18: The idea section on _cli-fab.md is factually incorrect. Move it to _cli_external - the whole Backlog section, and correct it. idea is a standalone binary that is shipped with fab-kit. Also make it explicit in the help text that the idea command operates on the backlog.md of the main worktree only if --main modifier is passed (make relevant code changes) - right now this behaviour is default (not obvious).

## Why

`_cli-fab.md` documents `idea` as `fab idea` — a subcommand of the `fab` dispatcher. This is factually wrong: `idea` is a standalone binary shipped at `fab/.kit/bin/idea`, completely separate from the `fab` Go backend. Agents reading `_cli-fab.md` will try to invoke `fab/.kit/bin/fab idea ...` which is incorrect.

Additionally, the `idea` binary currently always resolves to the **main worktree's** `fab/backlog.md` (via `git rev-parse --git-common-dir`), even when run from a linked worktree. This behavior is non-obvious — users in a worktree expect the tool to operate locally. The `--main` modifier should be required to opt into the main-worktree behavior, making the default operate on the current worktree.

## What Changes

### 1. Move Backlog section from `_cli-fab.md` to `_cli-external.md`

Remove the entire `# Backlog` section (including `## fab idea`) from `fab/.kit/skills/_cli-fab.md`. Move it to `fab/.kit/skills/_cli-external.md`, corrected:

- Invocation: `fab/.kit/bin/idea <subcommand>` (not `fab/.kit/bin/fab idea`)
- Description: standalone binary shipped with fab-kit, not a fab subcommand
- Remove `fab idea` row from the Command Reference table in `_cli-fab.md`

### 2. Add `--main` persistent flag to the `idea` CLI

In `src/go/idea/cmd/main.go`, add a `--main` persistent flag. When set, `idea` resolves the repo root via `git rev-parse --git-common-dir` (current behavior). Without `--main`, resolve via `git rev-parse --show-toplevel` (current worktree).

Code changes:
- `src/go/idea/cmd/main.go`: add `mainFlag` bool persistent flag
- `src/go/idea/cmd/resolve.go`: pass `mainFlag` to select resolution strategy
- `src/go/idea/internal/idea/idea.go`: add `WorktreeRoot()` function using `--show-toplevel`, rename current `GitRepoRoot()` to `MainRepoRoot()` or parameterize
- Update root command help text to explicitly mention `--main` behavior

### 3. Update help text

The root command `Short` and flag descriptions should make the worktree behavior explicit:
- Root short: "Backlog idea management — CRUD for fab/backlog.md (current worktree; use --main for main worktree)"
- `--main` flag: "Operate on the main worktree's backlog instead of the current worktree"

## Affected Memory

- `fab-workflow/distribution`: (modify) Update to note `idea` is a standalone binary alongside `wt`, not a fab subcommand

## Impact

- **`fab/.kit/skills/_cli-fab.md`**: Remove `# Backlog` section and `fab idea` from Command Reference table
- **`fab/.kit/skills/_cli-external.md`**: Add corrected Backlog/idea section; update frontmatter description
- **`src/go/idea/cmd/main.go`**: Add `--main` persistent flag
- **`src/go/idea/cmd/resolve.go`**: Branch on `mainFlag` for repo root resolution
- **`src/go/idea/internal/idea/idea.go`**: Add worktree-local root resolution function
- **`src/go/idea/internal/idea/idea_test.go`**: Update tests for new resolution behavior
- **`docs/specs/packages.md`**: May need update to reflect `--main` flag

## Open Questions

(none — scope is well-defined)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `idea` is standalone binary at `fab/.kit/bin/idea` | Verified: separate Go module at `src/go/idea/`, binary exists at `fab/.kit/bin/idea` | S:95 R:90 A:95 D:95 |
| 2 | Certain | Remove `fab idea` row from `_cli-fab.md` Command Reference table | Description explicitly says "move" and the entry is factually incorrect there | S:90 R:85 A:90 D:95 |
| 3 | Certain | Default (no flag) uses `--show-toplevel` for current worktree | User says "right now this behaviour is default (not obvious)" — wants main-worktree behavior gated behind `--main` | S:90 R:70 A:85 D:90 |
| 4 | Confident | Flag name is `--main` (not `--main-worktree` or `-m`) | User explicitly used "--main modifier" in the description; short enough, clear enough | S:85 R:90 A:80 D:70 |
| 5 | Confident | Keep `--file` flag as-is — `--main` and `--file` are orthogonal | `--file` overrides the path relative to the resolved root; `--main` changes which root is resolved. Both can coexist | S:60 R:85 A:80 D:75 |
| 6 | Confident | Update `_cli-external.md` frontmatter description to include `idea` | Currently says "wt (worktree manager), tmux, and /loop" — adding `idea` keeps it accurate | S:70 R:90 A:85 D:85 |
| 7 | Certain | `_cli-external.md` is the correct destination | User explicitly says "Move it to _cli_external" and the file exists at `fab/.kit/skills/_cli-external.md` | S:95 R:85 A:95 D:95 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
