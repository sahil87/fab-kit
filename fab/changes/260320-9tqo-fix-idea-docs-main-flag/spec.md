# Spec: Fix Idea Docs & --main Flag

**Change**: 260320-9tqo-fix-idea-docs-main-flag
**Created**: 2026-03-20
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Changing the `--file` flag behavior — it remains orthogonal to `--main`
- Adding `--main` to the `fab` dispatcher — only the standalone `idea` binary is affected
- Removing the `fab idea` alias in the Go backend (if one exists) — out of scope

## Documentation: Move Idea Section to `_cli-external.md`

### Requirement: Remove Backlog Section from `_cli-fab.md`

The entire `# Backlog` section (including `## fab idea` and all subsections) SHALL be removed from `fab/.kit/skills/_cli-fab.md`. The `fab idea` row SHALL be removed from the Command Reference table in that file.

#### Scenario: Agent reads `_cli-fab.md` after change

- **GIVEN** `_cli-fab.md` has been updated
- **WHEN** an agent reads the file for CLI invocation conventions
- **THEN** there is no mention of `idea` or `backlog` in the file
- **AND** the Command Reference table does not contain a `fab idea` row

### Requirement: Add Idea Section to `_cli-external.md`

`fab/.kit/skills/_cli-external.md` SHALL contain a new `## idea (Backlog Manager)` section documenting the standalone `idea` binary. The invocation MUST use `fab/.kit/bin/idea <subcommand>` (not `fab/.kit/bin/fab idea`). The section SHALL include the subcommand table, persistent flags (including `--main` and `--file`), query matching, backlog format, and output format — all carried over from the original `_cli-fab.md` content with corrections.

#### Scenario: Agent reads `_cli-external.md` for idea invocation

- **GIVEN** `_cli-external.md` has been updated
- **WHEN** an agent needs to invoke the idea command
- **THEN** the documented invocation is `fab/.kit/bin/idea <subcommand> [flags...]`
- **AND** the binary is described as a standalone binary shipped with fab-kit

#### Scenario: `_cli-external.md` frontmatter updated

- **GIVEN** `_cli-external.md` exists
- **WHEN** the file is updated
- **THEN** the frontmatter `description` field includes `idea` alongside `wt`, `tmux`, and `/loop`

### Requirement: Preserve Existing `_cli-external.md` Content

The existing `wt`, `tmux`, and `/loop` sections in `_cli-external.md` SHALL NOT be modified. The `idea` section SHALL be inserted as a new section (after `wt` and before `tmux`, since `idea` and `wt` are both fab-kit-shipped binaries).

#### Scenario: Existing wt documentation unchanged

- **GIVEN** `_cli-external.md` contains the `wt` section
- **WHEN** the `idea` section is added
- **THEN** the `wt` section content is identical to before the change

## CLI: Add `--main` Persistent Flag

### Requirement: Default Repo Root Resolution Uses Current Worktree

The `idea` binary SHALL resolve the repo root via `git rev-parse --show-toplevel` by default (without `--main`). This returns the current worktree's root directory, not the main worktree.

#### Scenario: Run `idea list` from a linked worktree without `--main`

- **GIVEN** a linked git worktree at `/repo.worktrees/alpha/`
- **AND** the worktree has its own `fab/backlog.md`
- **WHEN** the user runs `idea list` (no `--main` flag)
- **THEN** `idea` reads from `/repo.worktrees/alpha/fab/backlog.md`

#### Scenario: Run `idea list` from the main worktree without `--main`

- **GIVEN** the main git worktree at `/repo/`
- **WHEN** the user runs `idea list` (no `--main` flag)
- **THEN** `idea` reads from `/repo/fab/backlog.md`
- **AND** behavior is identical to `idea --main list` in this case

### Requirement: `--main` Flag Resolves to Main Worktree

When the `--main` flag is passed, the `idea` binary SHALL resolve the repo root via `git rev-parse --path-format=absolute --git-common-dir` and take its parent directory (current behavior). This always points to the main worktree regardless of which worktree the command is run from.

#### Scenario: Run `idea --main list` from a linked worktree

- **GIVEN** a linked git worktree at `/repo.worktrees/alpha/`
- **AND** the main repo is at `/repo/`
- **WHEN** the user runs `idea --main list`
- **THEN** `idea` reads from `/repo/fab/backlog.md`

### Requirement: `--main` Is a Persistent Flag

`--main` SHALL be a persistent flag on the root command (available to all subcommands), similar to `--file`. It is a boolean flag with no argument.

#### Scenario: `--main` works with any subcommand

- **GIVEN** the `idea` binary is installed
- **WHEN** the user runs `idea --main add "some idea"`
- **THEN** the idea is added to the main worktree's `fab/backlog.md`

### Requirement: `--file` Takes Precedence Over `--main`

When both `--file` and `--main` are specified, `--file` SHALL be resolved relative to the root determined by `--main`. The resolution priority remains: `--file` > `IDEAS_FILE` env > default `fab/backlog.md`, but the root directory is determined by `--main`.

#### Scenario: Both `--main` and `--file` specified

- **GIVEN** a linked worktree at `/repo.worktrees/alpha/`
- **AND** the main repo is at `/repo/`
- **WHEN** the user runs `idea --main --file custom/ideas.md list`
- **THEN** `idea` reads from `/repo/custom/ideas.md`

#### Scenario: `IDEAS_FILE` env with `--main`
<!-- clarified: IDEAS_FILE + --main interaction — same resolution chain as --file, root determined by --main -->

- **GIVEN** a linked worktree at `/repo.worktrees/alpha/`
- **AND** the main repo is at `/repo/`
- **AND** `IDEAS_FILE=custom/ideas.md` is set
- **WHEN** the user runs `idea --main list` (no `--file` flag)
- **THEN** `idea` reads from `/repo/custom/ideas.md`
- **AND** the resolution priority remains `--file` > `IDEAS_FILE` > default, with root from `--main`

### Requirement: Update Root Command Help Text

The root command's `Short` description SHALL explicitly mention the worktree behavior: `"Backlog idea management (current worktree; use --main for main worktree)"`. The `--main` flag description SHALL be `"Operate on the main worktree's backlog instead of the current worktree"`.

#### Scenario: `idea --help` shows worktree guidance

- **GIVEN** the `idea` binary is installed
- **WHEN** the user runs `idea --help`
- **THEN** the output contains "current worktree" and "--main"
- **AND** the `--main` flag is listed in the flags section

## Go Implementation

### Requirement: Add `WorktreeRoot()` Function

`src/go/idea/internal/idea/idea.go` SHALL export a new function `WorktreeRoot() (string, error)` that uses `git rev-parse --show-toplevel` to return the current worktree's root directory. The existing `GitRepoRoot()` function (using `--git-common-dir`) SHALL be renamed to `MainRepoRoot()` for clarity.

#### Scenario: `WorktreeRoot()` in a linked worktree

- **GIVEN** the current directory is inside a linked git worktree at `/repo.worktrees/alpha/`
- **WHEN** `WorktreeRoot()` is called
- **THEN** it returns `/repo.worktrees/alpha/`

#### Scenario: `MainRepoRoot()` in a linked worktree

- **GIVEN** the current directory is inside a linked git worktree
- **AND** the main worktree is at `/repo/`
- **WHEN** `MainRepoRoot()` is called
- **THEN** it returns `/repo/`

### Requirement: Update `resolveFile()` to Accept `mainFlag`

`src/go/idea/cmd/resolve.go`'s `resolveFile()` function SHALL branch on the `mainFlag` boolean: if true, call `idea.MainRepoRoot()`; if false, call `idea.WorktreeRoot()`.

#### Scenario: `resolveFile()` without `--main`

- **GIVEN** `mainFlag` is `false`
- **WHEN** `resolveFile()` is called
- **THEN** it calls `WorktreeRoot()` and resolves the file path relative to the current worktree

#### Scenario: `resolveFile()` with `--main`

- **GIVEN** `mainFlag` is `true`
- **WHEN** `resolveFile()` is called
- **THEN** it calls `MainRepoRoot()` and resolves the file path relative to the main worktree

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `idea` is standalone binary at `fab/.kit/bin/idea` | Confirmed from intake #1 — verified in codebase | S:95 R:90 A:95 D:95 |
| 2 | Certain | Remove `fab idea` row from `_cli-fab.md` Command Reference table | Confirmed from intake #2 — factually incorrect entry | S:90 R:85 A:90 D:95 |
| 3 | Certain | Default uses `--show-toplevel` for current worktree | Confirmed from intake #3 — user explicitly wants main-worktree gated behind `--main` | S:90 R:70 A:85 D:90 |
| 4 | Certain | Flag name is `--main` | Upgraded from intake Confident #4 — user explicitly wrote "--main modifier" | S:90 R:90 A:85 D:90 |
| 5 | Confident | `--main` and `--file` are orthogonal — `--file` resolved relative to `--main`-determined root | Confirmed from intake #5 — consistent with existing resolution chain | S:60 R:85 A:80 D:75 |
| 6 | Certain | `_cli-external.md` frontmatter description includes `idea` | Upgraded from intake Confident #6 — straightforward accuracy fix | S:80 R:90 A:90 D:90 |
| 7 | Certain | `_cli-external.md` is the correct destination | Confirmed from intake #7 | S:95 R:85 A:95 D:95 |
| 8 | Certain | Rename `GitRepoRoot()` to `MainRepoRoot()` | Codebase shows single function; renaming clarifies intent alongside new `WorktreeRoot()` | S:80 R:85 A:90 D:85 |
| 9 | Confident | Place idea section after wt and before tmux in `_cli-external.md` | Both `idea` and `wt` are fab-kit-shipped binaries; logical grouping | S:65 R:95 A:80 D:70 |
| 10 | Confident | Help text: "Backlog idea management (current worktree; use --main for main worktree)" | Concise, follows existing Cobra short-description patterns | S:70 R:90 A:75 D:75 |

10 assumptions (7 certain, 3 confident, 0 tentative, 0 unresolved).
