# Spec: Port wt Tests & Cleanup Legacy

**Change**: 260310-8m3k-port-wt-tests-cleanup-legacy
**Created**: 2026-03-10
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Porting `wt pr` tests — `wt pr` has no Go implementation yet; those behavioral expectations are captured as comments for future work
- Modifying the Go implementation to pass tests — tests SHALL match the existing Go behavior
- Adding new test behaviors beyond what the bats tests cover

## Test Porting: wt init

### Requirement: Init command tests SHALL cover script execution and error handling

The `wt init` Go command (`src/go/wt/cmd/init.go`) SHALL have tests covering: running the init script when it exists, friendly message when script doesn't exist, running from a worktree, idempotency, and error when not in a git repo.

#### Scenario: Init runs script when it exists
- **GIVEN** a git repo with an init script at `fab/.kit/worktree-init.sh`
- **WHEN** `wt init` is executed
- **THEN** the script runs successfully
- **AND** output contains "Running worktree init" and "Worktree init complete"

#### Scenario: Init shows guidance when no script exists
- **GIVEN** a git repo without an init script
- **WHEN** `wt init` is executed
- **THEN** output contains "No init script found" and instructions for creating one
- **AND** the command exits successfully

#### Scenario: Init errors outside git repo
- **GIVEN** a directory that is not a git repository
- **WHEN** `wt init` is executed
- **THEN** the command fails with "Not a git repository"

## Test Porting: wt create

### Requirement: Create command tests SHALL cover worktree creation, naming, branching, and collision handling

Tests for `src/go/wt/cmd/create.go` SHALL cover: exploratory worktree creation with random names, branch-based creation (local, remote, new), name derivation from branches, `--worktree-name` override, name collision errors, `--reuse` flag, init script integration, `--worktree-open` flag, porcelain output contract (stdout = path only, stderr = human messages), and branch-off-HEAD behavior.

#### Scenario: Exploratory worktree creation
- **GIVEN** a git repo on the main branch
- **WHEN** `wt create --non-interactive` is executed
- **THEN** a worktree is created with a random name in `{repo}.worktrees/`
- **AND** stderr contains "Created worktree:"
- **AND** stdout is exactly one line containing the worktree path

#### Scenario: Name collision with --reuse
- **GIVEN** a worktree named "reuse-test" already exists
- **WHEN** `wt create --non-interactive --reuse --worktree-name reuse-test` is executed
- **THEN** the command succeeds
- **AND** stdout contains the path to the existing worktree

#### Scenario: Name collision without --reuse
- **GIVEN** a worktree named "collision" already exists
- **WHEN** `wt create --non-interactive --worktree-name collision` is executed
- **THEN** the command fails with "already exists"

#### Scenario: Exploratory worktree branches off current branch
- **GIVEN** a git repo on a feature branch with a unique commit
- **WHEN** `wt create --non-interactive` is executed
- **THEN** the new worktree's HEAD matches the feature branch commit (not main)

#### Scenario: Branch name derivation
- **GIVEN** a branch `feature/login`
- **WHEN** `wt create --non-interactive feature/login` is executed
- **THEN** the worktree name is derived as "login"

## Test Porting: wt delete

### Requirement: Delete command tests SHALL cover deletion modes, branch cleanup, stashing, and delete-all

Tests for `src/go/wt/cmd/delete.go` SHALL cover: deleting by name, deleting current worktree, `--delete-branch` and `--delete-remote` flags, `--stash` flag for uncommitted changes, `--delete-all`, error for non-existent worktree, and directory cleanup verification.

#### Scenario: Delete by name removes worktree and branch
- **GIVEN** a worktree "test-wt" exists
- **WHEN** `wt delete --non-interactive --worktree-name test-wt` is executed
- **THEN** the worktree directory is removed
- **AND** the local branch is deleted
- **AND** output contains "Deleted worktree"

#### Scenario: Delete with --stash preserves changes
- **GIVEN** a worktree with uncommitted staged changes
- **WHEN** `wt delete --non-interactive --worktree-name <name> --stash` is executed
- **THEN** changes are stashed before deletion
- **AND** output contains "Stashing changes"

#### Scenario: Delete-all removes all worktrees
- **GIVEN** three worktrees exist
- **WHEN** `wt delete --non-interactive --delete-all` is executed
- **THEN** all three worktrees are removed

#### Scenario: Delete-all with no worktrees
- **GIVEN** no worktrees exist (only main repo)
- **WHEN** `wt delete --non-interactive --delete-all` is executed
- **THEN** output contains "No worktrees found"

## Test Porting: wt list

### Requirement: List command tests SHALL cover output formatting, --path, --json, and status indicators

Tests for `src/go/wt/cmd/list.go` SHALL cover: showing repo name and location, marking current worktree, total count, `--path` flag for single worktree lookup, `--json` flag with all required fields (name, branch, path, is_main, is_current, dirty, unpushed), mutual exclusivity of `--path` and `--json`, dirty/unpushed indicators, and `NO_COLOR` support.

#### Scenario: List shows all worktrees
- **GIVEN** two worktrees "wt1" and "wt2" exist
- **WHEN** `wt list` is executed
- **THEN** output contains both "wt1" and "wt2"
- **AND** output contains "Total: 3 worktree(s)"

#### Scenario: --path returns absolute path
- **GIVEN** a worktree "path-test" exists
- **WHEN** `wt list --path path-test` is executed
- **THEN** output is a single line containing the absolute path
- **AND** the path ends with "/path-test"

#### Scenario: --json outputs valid JSON with all fields
- **GIVEN** a worktree exists
- **WHEN** `wt list --json` is executed
- **THEN** output is valid JSON
- **AND** each entry has name, branch, path, is_main, is_current, dirty, unpushed fields

#### Scenario: --path and --json are mutually exclusive
- **GIVEN** a git repo
- **WHEN** `wt list --path foo --json` is executed
- **THEN** the command fails with "mutually exclusive"

## Test Porting: wt open

### Requirement: Open command tests SHALL cover target resolution and app launching

Tests for `src/go/wt/cmd/open.go` SHALL cover: opening current worktree, opening by name, opening by path, error for non-existent worktree, error for unknown app, and error from main repo without target.

#### Scenario: Open by name
- **GIVEN** a worktree "named-open" exists
- **WHEN** `wt open --app code named-open` is executed
- **THEN** the command succeeds
- **AND** the code application is invoked with the correct path

#### Scenario: Error for non-existent worktree
- **GIVEN** no worktree "nonexistent" exists
- **WHEN** `wt open --app code nonexistent` is executed
- **THEN** the command fails with "not found"

## Test Porting: Common utilities

### Requirement: Rollback, branch validation, and stash tests SHALL be ported from wt-common.bats

The existing Go tests already cover rollback (`rollback_test.go`) and names (`names_test.go`). Any additional behavioral expectations from `wt-common.bats` not already covered SHALL be added — specifically hash-based stash create/apply and menu display behavior.

#### Scenario: Branch name validation rejects invalid characters
- **GIVEN** various invalid branch names (tilde, caret, colon, space, double dot, .lock suffix)
- **WHEN** `ValidateBranchName` is called
- **THEN** it returns an error for each

#### Scenario: Branch name validation accepts valid names
- **GIVEN** valid branch names ("feature-auth", "feature/add-auth", "release/v1.2.3")
- **WHEN** `ValidateBranchName` is called
- **THEN** it returns nil

## Test Porting: Edge cases

### Requirement: Edge case tests SHALL cover corrupted state, special characters, detached HEAD, and multiple worktrees

Tests SHALL cover: worktree deleted outside git (prune recovery), creating worktrees from detached HEAD, creating multiple worktrees with unique names, invalid branch names fail cleanly without leftover directories, and branch names with slashes.

#### Scenario: Create with invalid branch name leaves no partial state
- **GIVEN** a git repo
- **WHEN** `wt create --non-interactive --worktree-name bad-branch "refs/invalid..name"` is executed
- **THEN** the command fails
- **AND** no worktree directory is left behind

#### Scenario: Multiple worktree creation generates unique names
- **GIVEN** a git repo
- **WHEN** 5 exploratory worktrees are created
- **THEN** all 5 have unique names

## Test Porting: Integration

### Requirement: Integration tests SHALL cover full lifecycle workflows

Tests SHALL cover: create → list → delete lifecycle, create multiple → delete all, non-interactive automation workflow, and git state integrity after create-delete cycles.

#### Scenario: Full create-list-delete lifecycle
- **GIVEN** a git repo
- **WHEN** a worktree is created, listed, then deleted
- **THEN** the worktree appears in list after creation
- **AND** disappears from list after deletion

#### Scenario: Git state integrity after create-delete
- **GIVEN** a git repo
- **WHEN** a worktree is created and then deleted
- **THEN** git fsck reports no errors

## Legacy Cleanup: Remove src/packages/

### Requirement: `src/packages/` SHALL be deleted entirely

The directory contains only legacy shell infrastructure: `rc-init.sh` (env setup), `wt/` (old shell wt package with tests and temp artifacts), and `tests/` (bats submodule libs). All superseded by the Go rewrite.

#### Scenario: src/packages/ removed
- **GIVEN** the bats tests have been ported to Go
- **WHEN** the cleanup is applied
- **THEN** `src/packages/` no longer exists

## Legacy Cleanup: Remove src/tests/

### Requirement: `src/tests/` SHALL be deleted entirely

Contains only `libs/` which holds bats submodule checkouts (bats-core, bats-support, bats-assert, bats-file).

#### Scenario: src/tests/ removed
- **GIVEN** bats is no longer used
- **WHEN** the cleanup is applied
- **THEN** `src/tests/` no longer exists

## Legacy Cleanup: Remove .gitmodules

### Requirement: `.gitmodules` SHALL be deleted

Contains only 4 bats-related submodule entries. No other submodules exist.

#### Scenario: .gitmodules removed
- **GIVEN** `.gitmodules` contains only bats submodule refs
- **WHEN** the cleanup is applied
- **THEN** `.gitmodules` no longer exists
- **AND** git submodule references are cleaned up

## Legacy Cleanup: Reference cleanup

### Requirement: References to removed paths SHALL be updated

Any references to `src/packages/`, `src/tests/`, or bats in documentation, scripts, or config files SHALL be removed or updated.

#### Scenario: No dangling references
- **GIVEN** the directories have been removed
- **WHEN** the codebase is searched for "src/packages" and "src/tests"
- **THEN** no stale references remain (except in git history)

## Deprecated Requirements

### wt pr tests
**Reason**: `wt pr` has no Go implementation — only `init`, `create`, `delete`, `list`, `open` exist in `src/go/wt/cmd/`
**Migration**: Test expectations captured as TODO comments in test files for when `wt pr` is implemented

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Port tests to Go test files in src/go/wt/ | Confirmed from intake #1 — Go module has test infra | S:90 R:90 A:95 D:95 |
| 2 | Certain | Remove src/packages/ entirely | Confirmed from intake #2 — all legacy shell | S:85 R:85 A:90 D:95 |
| 3 | Certain | Remove src/tests/ entirely | Confirmed from intake #3 — only bats libs | S:85 R:85 A:90 D:95 |
| 4 | Certain | Remove .gitmodules | Confirmed from intake #4 — only bats entries | S:95 R:85 A:95 D:95 |
| 5 | Certain | Tests target cmd/ layer as integration tests | Confirmed — bats tests are CLI-level; existing Go tests cover internal/ | S:85 R:85 A:90 D:90 |
| 6 | Certain | No CI references to bats tests | Confirmed — searched .github/ for bats/packages/tests, no matches | S:95 R:90 A:95 D:95 |
| 7 | Certain | wt pr not implemented in Go — skip those tests | Confirmed — only init/create/delete/list/open in cmd/ | S:90 R:90 A:90 D:95 |
| 8 | Certain | .gitmodules has no non-bats entries | Confirmed from intake #8 — file read verified | S:95 R:90 A:95 D:95 |
| 9 | Certain | Tests use Go's testing package with exec.Command for CLI testing | Standard Go CLI test pattern; matches existing test style | S:90 R:90 A:95 D:95 |
| 10 | Confident | Some bats test behaviors already covered by existing Go unit tests | Rollback and names tests exist; need to check overlap and only port gaps | S:75 R:85 A:80 D:80 |

10 assumptions (9 certain, 1 confident, 0 tentative, 0 unresolved).
