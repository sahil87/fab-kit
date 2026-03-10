# Spec: Go wt Binary

**Change**: 260310-qbiq-go-wt-binary
**Created**: 2026-03-10
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Port `wt pr` — overlaps with `/git-pr` territory, shell script removed entirely
- Interactive TUI library (bubbletea/survey) — use stdlib `fmt.Scan*` and `os.Exec("fzf")` same as existing patterns in fab-go
- Config file for wt — wt operates purely from git state and conventions, no config needed

## Worktree Package: `internal/worktree/`

### Requirement: Repo Context Detection

The `internal/worktree/` package SHALL provide `RepoContext()` that returns the main repo root, repo name, and worktrees directory path (`<repo>.worktrees/`), derived from `git rev-parse --git-common-dir`.

#### Scenario: Run from main repo
- **GIVEN** the current directory is inside the main git repository
- **WHEN** `RepoContext()` is called
- **THEN** it returns the repo root, basename as repo name, and `<parent>/<repo>.worktrees/` as worktrees dir

#### Scenario: Run from a worktree
- **GIVEN** the current directory is inside a git worktree
- **WHEN** `RepoContext()` is called
- **THEN** it returns the same main repo root (via `--git-common-dir`), not the worktree root

#### Scenario: Not a git repo
- **GIVEN** the current directory is not inside any git repository
- **WHEN** `RepoContext()` is called
- **THEN** it returns an error

### Requirement: Random Name Generation

The package SHALL provide `GenerateRandomName()` returning an `adjective-noun` combo from embedded word lists (~50 adjectives, ~50 nouns). `GenerateUniqueName(worktreesDir string, maxRetries int)` SHALL retry until a non-colliding name is found, returning an error after `maxRetries` attempts.

#### Scenario: Unique name generated on first try
- **GIVEN** the worktrees directory has no existing worktrees
- **WHEN** `GenerateUniqueName` is called
- **THEN** it returns an `adjective-noun` name on the first attempt

#### Scenario: Retry exhaustion
- **GIVEN** all possible names collide with existing directories
- **WHEN** `GenerateUniqueName` is called with `maxRetries=10`
- **THEN** it returns an error after 10 attempts

### Requirement: Branch Validation

The package SHALL provide `ValidateBranchName(name string) error` that rejects names with spaces, `~`, `^`, `:`, `?`, `*`, `[`, `..`, `.lock` suffix, leading `.`, or `/.` — matching git ref naming rules.

#### Scenario: Valid branch name
- **GIVEN** a branch name `feature/auth-login`
- **WHEN** `ValidateBranchName` is called
- **THEN** it returns nil

#### Scenario: Invalid branch with `..`
- **GIVEN** a branch name `foo..bar`
- **WHEN** `ValidateBranchName` is called
- **THEN** it returns an error

### Requirement: Change Detection

The package SHALL provide functions to detect uncommitted changes, untracked files, and unpushed commits, matching the semantics of `wt_has_uncommitted_changes`, `wt_has_untracked_files`, and `wt_has_unpushed_commits` from `wt-common.sh`.

#### Scenario: Clean working tree
- **GIVEN** a worktree with no uncommitted changes and no untracked files
- **WHEN** `HasUncommittedChanges` and `HasUntrackedFiles` are called
- **THEN** both return false

### Requirement: Hash-Based Stash

The package SHALL provide `StashCreate(msg string) (string, error)` and `StashApply(hash string) error` using `git stash create` (hash-based, not index-based) for concurrency safety. `StashCreate` SHALL `git add -A`, create the stash, store it in the reflog, reset, and return the hash.

#### Scenario: Stash with changes
- **GIVEN** a worktree with uncommitted changes
- **WHEN** `StashCreate("pre-delete stash")` is called
- **THEN** it returns a non-empty hash and the working tree is clean

#### Scenario: Stash without changes
- **GIVEN** a worktree with no changes
- **WHEN** `StashCreate` is called
- **THEN** it returns an empty string and no error

### Requirement: Rollback Stack

The package SHALL provide a `Rollback` type with `Register(cmd string)`, `Execute()` (LIFO order), and `Disarm()` methods. `Execute` SHALL continue executing remaining commands even if individual commands fail.

#### Scenario: Rollback on failure
- **GIVEN** two rollback commands are registered: "git branch -D foo" and "git worktree remove /tmp/wt"
- **WHEN** `Execute()` is called
- **THEN** "git worktree remove /tmp/wt" runs first, then "git branch -D foo"

### Requirement: Default Branch Detection

The package SHALL provide `DefaultBranch() string` that checks `refs/remotes/origin/HEAD` first, falls back to checking `refs/heads/main` then `refs/heads/master`, and finally falls back to `HEAD`.

#### Scenario: Default branch is main
- **GIVEN** a repo with `refs/heads/main`
- **WHEN** `DefaultBranch()` is called
- **THEN** it returns `"main"`

### Requirement: Worktree CRUD Operations

The package SHALL provide `CreateWorktree(path, branch string, newBranch bool) error` and `RemoveWorktree(path string, force bool) error` wrapping `git worktree add` and `git worktree remove`. `CreateWorktree` with `newBranch=true` SHALL use `-b` flag. `RemoveWorktree` with `force=true` SHALL use `--force` flag, followed by `git worktree prune`.

#### Scenario: Create exploratory worktree
- **GIVEN** a valid repo context and a name "swift-fox"
- **WHEN** `CreateWorktree("<worktrees>/swift-fox", "swift-fox", true)` is called
- **THEN** a new worktree is created at `<repo>.worktrees/swift-fox/` on branch `swift-fox`

#### Scenario: Create worktree for existing branch
- **GIVEN** a local branch `feature/auth` exists
- **WHEN** `CreateWorktree("<worktrees>/auth", "feature/auth", false)` is called
- **THEN** a worktree is created checking out the existing branch

### Requirement: Interactive Menu

The package SHALL provide `ShowMenu(prompt string, options []string, defaultIdx int) (int, error)` that displays a numbered menu with a Cancel option (0), reads user input from stdin, validates numeric input, and returns the selected index. The `defaultIdx` is used when the user presses Enter without input.

#### Scenario: User selects option
- **GIVEN** a menu with options ["Stash changes", "Discard changes"]
- **WHEN** the user enters "1"
- **THEN** `ShowMenu` returns 1

#### Scenario: User cancels
- **GIVEN** a menu with options
- **WHEN** the user enters "0"
- **THEN** `ShowMenu` returns 0

### Requirement: OS and Session Detection

The package SHALL provide `DetectOS() string` (returns "macos", "linux", or "unknown"), `IsByobuSession() bool`, and `IsTmuxSession() bool` matching the semantics of the bash equivalents.

#### Scenario: Inside tmux but not byobu
- **GIVEN** `$TMUX` is set and `$BYOBU_TTY` is not set
- **WHEN** `IsTmuxSession()` is called
- **THEN** it returns true

### Requirement: Worktree Name Derivation

The package SHALL provide `DeriveWorktreeName(branch string) string` that extracts the last segment after slashes and replaces non-alphanumeric characters (except `-` and `_`) with `-`.

#### Scenario: Branch with slashes
- **GIVEN** a branch name `feature/login`
- **WHEN** `DeriveWorktreeName` is called
- **THEN** it returns `"login"`

## wt create

### Requirement: Create Subcommand

`wt create [flags] [branch]` SHALL create a git worktree at `<repo>.worktrees/<name>/`. When `branch` is omitted, it creates an exploratory worktree with a random name as the branch name. When `branch` is provided, it checks out that branch (fetching from remote if needed) or creates a new branch.

Flags:
- `--worktree-name <name>` — explicit worktree name (skips name prompt)
- `--worktree-init <bool>` — run init script (default: true)
- `--worktree-open <app>` — open in app after creation, or `skip`
- `--reuse` — reuse existing worktree if name collides (requires `--worktree-name`)
- `--non-interactive` — no prompts, porcelain output (only path on stdout, messages on stderr)

#### Scenario: Exploratory worktree (interactive)
- **GIVEN** user runs `wt create` with no arguments
- **WHEN** a unique random name "swift-fox" is generated
- **THEN** a worktree is created at `<repo>.worktrees/swift-fox/` on branch `swift-fox`
- **AND** the user is prompted for name confirmation, init, and app selection

#### Scenario: Branch worktree (non-interactive)
- **GIVEN** user runs `wt create --non-interactive feature/auth`
- **WHEN** branch `feature/auth` exists locally
- **THEN** a worktree is created with name derived from branch (`auth`)
- **AND** only the worktree path is printed to stdout
- **AND** human-readable messages go to stderr

#### Scenario: Reuse existing worktree
- **GIVEN** user runs `wt create --reuse --worktree-name alpha feature/auth`
- **WHEN** worktree `alpha` already exists
- **THEN** the existing path is returned without creating a new worktree

#### Scenario: Dirty working tree warning
- **GIVEN** the main repo has uncommitted changes
- **WHEN** user runs `wt create` interactively
- **THEN** a menu offers: Continue, Stash changes, Abort

#### Scenario: Remote branch checkout
- **GIVEN** branch `feature/auth` exists on remote but not locally
- **WHEN** user runs `wt create feature/auth`
- **THEN** the branch is fetched from origin and the worktree is created

### Requirement: Worktree Init Script

After creating a worktree, if `$WORKTREE_INIT_SCRIPT` (default: `fab/.kit/worktree-init.sh`) exists in the main repo, it SHALL be executed in the new worktree directory. In non-interactive mode, init runs automatically. In interactive mode for exploratory worktrees, init runs automatically. For branch worktrees interactively, the user is prompted.

#### Scenario: Init script exists
- **GIVEN** `fab/.kit/worktree-init.sh` exists in the main repo
- **WHEN** a worktree is created with `--worktree-init true`
- **THEN** the init script runs in the new worktree directory

### Requirement: Rollback on Failure

If any step during worktree creation fails (worktree add, init script, etc.), all previous steps SHALL be rolled back in reverse order: remove worktree, delete branch (if newly created). Signal handlers (SIGINT, SIGTERM) SHALL trigger rollback.

#### Scenario: Init script fails
- **GIVEN** the worktree was created and the init script fails
- **WHEN** rollback executes
- **THEN** the worktree is removed and the branch is deleted (if newly created)

## wt list

### Requirement: List Subcommand

`wt list [flags]` SHALL list all git worktrees with status information.

Flags:
- `--path <name>` — output just the absolute path for a named worktree
- `--json` — output worktree data as a JSON array
- `--path` and `--json` are mutually exclusive

#### Scenario: Default formatted list
- **GIVEN** the repo has 3 worktrees (main + 2 created)
- **WHEN** user runs `wt list`
- **THEN** a formatted table shows: name, branch, status indicators (dirty `*`, unpushed `↑N`), path
- **AND** the current worktree is marked with a green `*`

#### Scenario: JSON output
- **GIVEN** the repo has worktrees
- **WHEN** user runs `wt list --json`
- **THEN** a JSON array is output with objects containing `name`, `branch`, `path`, `is_main`, `is_current`, `dirty`, `unpushed`

#### Scenario: Path lookup
- **GIVEN** a worktree named "swift-fox" exists
- **WHEN** user runs `wt list --path swift-fox`
- **THEN** the absolute path is printed to stdout

#### Scenario: Path lookup — not found
- **GIVEN** no worktree named "missing" exists
- **WHEN** user runs `wt list --path missing`
- **THEN** exits with error code 1 and message suggesting `wt list`

## wt open

### Requirement: Open Subcommand

`wt open [flags] [name|path]` SHALL open a worktree in a detected application. When called without arguments from a worktree, opens the current worktree. When called without arguments from the main repo, shows a worktree selection menu then an app selection menu.

Flags:
- `--app <name>` — open in specified app, skipping the app menu. Accepts command keys (`code`, `cursor`, `ghostty`, `tmux_window`, `byobu_tab`) or display names (case-insensitive).

#### Scenario: Open in tmux (from worktree)
- **GIVEN** user is in worktree "swift-fox" inside a tmux session
- **WHEN** user runs `wt open --app tmux_window`
- **THEN** a new tmux window named `<repo>-swift-fox` opens in the worktree directory

#### Scenario: Interactive worktree + app selection
- **GIVEN** user is in the main repo with worktrees available
- **WHEN** user runs `wt open`
- **THEN** a worktree selection menu appears (defaulting to most recently modified)
- **AND** after selection, an app menu appears (detecting available apps)

### Requirement: Application Detection

The binary SHALL detect available applications: VSCode (`code`), Cursor (`cursor`), Ghostty, iTerm2 (macOS), Terminal.app (macOS), GNOME Terminal (Linux), Konsole (Linux), Finder/Nautilus/Dolphin, clipboard copy, byobu tab, tmux window. Detection uses CLI availability, macOS bundle IDs, or Linux .desktop files.

#### Scenario: Detected apps on Linux with tmux
- **GIVEN** the system has `code` and `cursor` in PATH, and is inside tmux
- **WHEN** the app list is built
- **THEN** VSCode, Cursor, tmux window, and Copy path are available

### Requirement: Default App Detection

The binary SHALL suggest a default app based on context: `$TERM_PROGRAM=vscode` → VSCode, `$TERM_PROGRAM=cursor` → Cursor, byobu session → byobu tab, tmux session → tmux window, else check `~/.cache/wt/last-app`.

#### Scenario: Inside VS Code terminal
- **GIVEN** `$TERM_PROGRAM` is `vscode`
- **WHEN** the app menu is shown
- **THEN** VSCode is the default selection

### Requirement: Last-Used App Cache

After opening in an app, the selected app command key SHALL be saved to `~/.cache/wt/last-app` and used as the default on subsequent invocations (when no context-based default applies).

#### Scenario: Cache persistence
- **GIVEN** the user previously opened a worktree in Cursor
- **WHEN** the user runs `wt open` again (no context-based default)
- **THEN** Cursor is the default selection

## wt delete

### Requirement: Delete Subcommand

`wt delete [flags]` SHALL delete a git worktree with optional branch and remote cleanup.

Flags:
- `--worktree-name <name>` — worktree to delete
- `--delete-branch <bool>` — delete local branch (default: true)
- `--delete-remote <bool>` — delete remote branch (default: true)
- `--delete-all` — delete all worktrees for the repo
- `--stash`, `-s` — stash uncommitted changes before deleting
- `--non-interactive` — no prompts, use defaults

Resolution order: `--worktree-name` → current worktree (if in one) → interactive selection menu.

#### Scenario: Delete current worktree (interactive)
- **GIVEN** user is inside worktree "swift-fox"
- **WHEN** user runs `wt delete`
- **THEN** worktree info is displayed, and prompts appear for uncommitted changes (if any) and confirmation
- **AND** on confirmation, worktree is removed, branch deleted locally and on remote

#### Scenario: Delete by name (non-interactive)
- **GIVEN** worktree "swift-fox" exists
- **WHEN** user runs `wt delete --non-interactive --worktree-name swift-fox`
- **THEN** worktree is removed with force, branch deleted, no prompts

#### Scenario: Stash before delete
- **GIVEN** worktree "swift-fox" has uncommitted changes
- **WHEN** user runs `wt delete --stash --worktree-name swift-fox`
- **THEN** changes are stashed (hash-based), hash printed for recovery, then worktree removed

#### Scenario: Delete all worktrees
- **GIVEN** the repo has 3 worktrees (excluding main)
- **WHEN** user runs `wt delete --delete-all`
- **THEN** confirmation prompt lists all 3, then deletes each sequentially

#### Scenario: Unpushed commits warning
- **GIVEN** worktree branch has 3 unpushed commits
- **WHEN** user runs `wt delete` interactively
- **THEN** a warning shows the commit list and asks for confirmation

### Requirement: Branch Cleanup

When `--delete-branch true` (default), deletion SHALL:
1. Force-delete the local branch (`git branch -D`)
2. If `--delete-remote true` (default), delete the remote branch (`git push origin --delete`)
3. Also delete any orphaned original `wt/<name>` branch if the worktree switched to a different branch

#### Scenario: Orphaned wt branch cleanup
- **GIVEN** worktree "swift-fox" was created with branch `swift-fox` then switched to `feature/auth`
- **WHEN** deletion with branch cleanup occurs
- **THEN** both `feature/auth` and `swift-fox` branches are deleted

### Requirement: Selection Menu

When no worktree is specified and not in a worktree, the binary SHALL show a selection menu with an "All (N worktrees)" option at the top, defaulting to the most recently modified worktree.

#### Scenario: Menu with "All" option
- **GIVEN** 2 worktrees exist: "swift-fox" (older) and "calm-owl" (newer)
- **WHEN** the selection menu appears
- **THEN** options are: [1] All (2 worktrees), [2] swift-fox, [3] calm-owl (default)

## wt init

### Requirement: Init Subcommand

`wt init` SHALL run the worktree init script (`$WORKTREE_INIT_SCRIPT`, default: `fab/.kit/worktree-init.sh`) from the main repo root in the current worktree directory. If the init script does not exist, print guidance and exit 0.

#### Scenario: Init script exists
- **GIVEN** `fab/.kit/worktree-init.sh` exists
- **WHEN** user runs `wt init`
- **THEN** the script runs in the current worktree directory

#### Scenario: No init script
- **GIVEN** no init script at the expected path
- **WHEN** user runs `wt init`
- **THEN** a message shows the expected path and instructions to create it, exit 0

## Binary & Build Integration

### Requirement: Separate Binary

The `wt` binary SHALL be built from `src/fab-go/cmd/wt/main.go` as a separate binary from `fab`. It uses cobra for subcommand dispatch. It SHALL NOT depend on any `fab`-specific packages — only `internal/worktree/` and shared utilities in `internal/`.

#### Scenario: Build wt binary
- **GIVEN** the Go module at `src/fab-go/`
- **WHEN** `go build ./cmd/wt` is run
- **THEN** a `wt` binary is produced

### Requirement: Exit Codes

The `wt` binary SHALL use these exit codes matching the bash scripts:
- `0` — success
- `1` — general error
- `2` — invalid arguments
- `3` — git operation failed
- `4` — retry exhausted (name generation)
- `5` — byobu tab error
- `6` — tmux window error

#### Scenario: Invalid argument
- **GIVEN** user runs `wt create --unknown-flag`
- **WHEN** cobra parses arguments
- **THEN** exits with code 2

### Requirement: Error Format

All errors SHALL use the structured format: `Error: {what}\n  Why: {why}\n  Fix: {fix}` matching the bash `wt_error` function. Colors SHALL be disabled when `$NO_COLOR` is set.

#### Scenario: Error with NO_COLOR
- **GIVEN** `$NO_COLOR` is set
- **WHEN** an error occurs
- **THEN** the error message has no ANSI color codes

### Requirement: Justfile Build Recipes

The justfile SHALL include recipes for building the wt binary:
- `build-wt` — current platform at `fab/.kit/bin/wt`
- `build-wt-target os arch` — cross-compile to `.release-build/wt-<os>-<arch>`
- `build-wt-all` — cross-compile for all 4 release targets

`build-all` SHALL include `build-wt-all`. `package-kit` SHALL include the `wt` binary in per-platform archives at `.kit/bin/wt`.

#### Scenario: Local dev build
- **GIVEN** the Go module compiles
- **WHEN** `just build-wt` is run
- **THEN** `fab/.kit/bin/wt` is produced

#### Scenario: Cross-compile all
- **GIVEN** Go cross-compilation toolchain available
- **WHEN** `just build-wt-all` is run
- **THEN** 4 binaries are produced in `.release-build/`

### Requirement: Release Archive Integration

Per-platform archives (`kit-{os}-{arch}.tar.gz`) SHALL include the `wt` binary at `.kit/bin/wt` alongside `fab-go` and `fab-rust`. The generic archive (`kit.tar.gz`) SHALL NOT include any binaries.

#### Scenario: Platform archive contents
- **GIVEN** a release build is packaged
- **WHEN** `kit-linux-arm64.tar.gz` is extracted
- **THEN** `.kit/bin/wt`, `.kit/bin/fab-go`, and `.kit/bin/fab-rust` are present

### Requirement: Test Coverage

Go tests SHALL provide same or greater coverage than the shell scripts being replaced. At minimum, test:
- Name generation (randomness, collision retry)
- Branch validation (valid/invalid patterns)
- Worktree name derivation
- Repo context detection
- JSON output formatting (for `wt list --json`)

#### Scenario: Go tests pass
- **GIVEN** the `internal/worktree/` package and `cmd/wt/` tests
- **WHEN** `go test ./...` runs
- **THEN** all tests pass

## Deprecated Requirements

### Shell Script Removal

**Reason**: All 6 wt-* shell scripts and `wt-common.sh` are replaced by the Go binary (5 ported commands) and removed (`wt-pr` dropped entirely).

**Migration**: Users invoke `wt <subcommand>` instead of `wt-<subcommand>`. No shim layer — direct cutover.

### Legacy PATH for `wt/bin`

**Reason**: The wt binary lives at `fab/.kit/bin/wt` which is already on PATH via `env-packages.sh`. The `fab/.kit/packages/wt/bin` PATH entry is no longer needed after the package directory is removed.

**Migration**: `env-packages.sh` loop over `packages/*/bin` will naturally stop including wt once the directory is removed.

## Design Decisions

1. **Separate binary, not fab subcommand**: `wt` operates on any git repo, not just fab-initialized projects. Different concern domain — worktree management vs workflow pipeline. Users type `wt create`, not `fab wt create`.
   - *Why*: Clean separation of concerns, simpler mental model, works without `fab/` directory
   - *Rejected*: `fab wt` subcommand — would require fab init, conflates worktree management with workflow

2. **Shell out to git rather than using go-git**: The wt commands shell out to `git` CLI for all git operations (worktree add/remove, branch operations, stash, etc.) rather than using a Go git library.
   - *Why*: Matches the bash scripts exactly, avoids large dependency, git CLI handles edge cases (worktree locking, reflog, etc.)
   - *Rejected*: go-git — large dependency, incomplete worktree support, behavior drift risk

3. **No shim layer**: Direct cutover from shell scripts to Go binary. No backward compatibility shim in the old scripts.
   - *Why*: User confirmed — clean break, less code to maintain
   - *Rejected*: Shim delegation — adds complexity for minimal transition benefit

4. **Existing `internal/worktree/` package extended**: The existing `worktree.go` (used by `fab pane-map`) already has worktree listing and state detection. The wt binary adds new files alongside it for CRUD, names, stash, rollback, etc.
   - *Why*: Reuses existing worktree list parsing and state detection. Single package for all worktree operations.
   - *Rejected*: Separate `internal/wt/` package — would duplicate worktree listing logic

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Go, same module as fab (`src/fab-go/`) | Confirmed from intake #1 — shared internal packages, single go.mod | S:95 R:85 A:90 D:90 |
| 2 | Certain | Separate `wt` binary at `cmd/wt/` | Confirmed from intake #2 — different concern domains, works without fab init | S:95 R:85 A:90 D:90 |
| 3 | Certain | wt-common.sh → extend `internal/worktree/` | Confirmed from intake #3, leverages existing worktree.go | S:95 R:85 A:90 D:95 |
| 4 | Certain | Exclude wt pr, remove shell script | Confirmed from intake #4 — overlaps /git-pr, dropped entirely | S:95 R:85 A:80 D:75 |
| 5 | Certain | Preserve interactive TUI (menus, fzf) | Confirmed from intake #5 — use stdlib + os.Exec for fzf | S:95 R:70 A:75 D:70 |
| 6 | Certain | No shim — direct cutover | Confirmed from intake #6 — clean break | S:95 R:90 A:80 D:75 |
| 7 | Certain | Both binaries in same per-platform archive | Confirmed from intake #7 | S:85 R:85 A:85 D:90 |
| 8 | Certain | PATH already covered by env-packages.sh | env-packages.sh already adds `$KIT_DIR/bin` to PATH (line 5) — no change needed | S:95 R:90 A:95 D:95 |
| 9 | Certain | Remove all wt shell scripts including wt-pr | Confirmed from intake #9 — user confirmed full removal | S:95 R:70 A:80 D:75 |
| 10 | Certain | Shell out to git CLI, not go-git | Matches bash behavior exactly, avoids large dependency | S:85 R:85 A:90 D:90 |
| 11 | Certain | Extend existing `internal/worktree/` package | Existing worktree.go has list/parse logic, extend rather than duplicate | S:90 R:85 A:90 D:90 |
| 12 | Certain | Test coverage >= shell scripts | Confirmed from intake #8 — user required same or more coverage | S:95 R:85 A:85 D:90 |
| 13 | Confident | `wt open` detects apps via CLI check + platform-specific methods | Matching bash: `command -v` + macOS `mdfind` + Linux .desktop files. Go exec.LookPath for CLI, os-specific for rest | S:80 R:75 A:75 D:70 |

13 assumptions (12 certain, 1 confident, 0 tentative, 0 unresolved).
