# Spec: Harden wt Package Resilience

**Change**: 260218-qcqx-harden-wt-resilience
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- GitLab MR support for `wt-pr` — no GitLab usage detected in the project; GitHub-only for now
- Changes to `wt-open`, `wt-list`, or `wt-init` — these commands have no resilience gaps

## wt Package: Atomic Rollback

### Requirement: Rollback Stack in Shared Library

The `wt-common.sh` library SHALL provide a rollback stack mechanism (`WT_ROLLBACK_STACK` array) with three functions: `wt_register_rollback` (push), `wt_rollback` (execute LIFO), and `wt_disarm_rollback` (clear).

`wt_rollback` SHALL disable `set -e` (`set +e`) at entry and restore it at exit, ensuring all entries execute even if individual commands fail. Each command's failure SHALL be suppressed (stderr redirected to /dev/null) so that one rollback step failing does not prevent subsequent rollbacks.
<!-- clarified: wt_rollback disables set -e to prevent trap handler from exiting early on rollback command failures -->

`wt_disarm_rollback` SHALL clear the stack to prevent rollback on successful completion.

#### Scenario: Successful creation disarms rollback
- **GIVEN** `wt-create` has registered rollback entries for worktree add and branch creation
- **WHEN** `wt-create` completes successfully
- **THEN** `wt_disarm_rollback` is called
- **AND** the EXIT trap fires with an empty stack (no-op)

#### Scenario: Mid-creation failure triggers rollback
- **GIVEN** `wt-create` has registered `git worktree remove --force <path>` after a successful `git worktree add`
- **WHEN** a subsequent step fails (e.g., init script errors with `set -e` active)
- **THEN** the EXIT trap fires `wt_rollback`
- **AND** the orphaned worktree is removed
- **AND** the orphaned local branch is deleted

### Requirement: Rollback Integration in wt-create

`wt-create` SHALL register rollback entries at each creation checkpoint:

1. After `git worktree add` succeeds → register `git worktree remove --force <path>`
2. After new branch creation (mode "new") → register `git branch -D <branch>`

The registration points SHALL be inside the existing `wt_create_worktree` function or immediately after each call to it. An `EXIT` trap SHALL invoke `wt_rollback`.

On successful completion (just before final output), `wt-create` SHALL call `wt_disarm_rollback`.

#### Scenario: git worktree add succeeds but init script fails
- **GIVEN** a valid branch and worktree name
- **WHEN** `git worktree add` succeeds but `wt_run_worktree_setup` fails
- **THEN** `wt_rollback` removes the worktree via `git worktree remove --force`
- **AND** if a new branch was created, it is deleted via `git branch -D`
- **AND** the exit code is non-zero

#### Scenario: Rollback tolerates already-cleaned state
- **GIVEN** rollback entries are registered
- **WHEN** the worktree directory was already removed by another process
- **THEN** `wt_rollback` runs all entries without error (failures suppressed)

## wt Package: Signal Handling

### Requirement: SIGINT/SIGTERM Trap Handlers

`wt-common.sh` SHALL provide a `wt_cleanup_on_signal` function that:
1. Prints a newline (to separate from `^C` display)
2. Calls `wt_rollback`
3. Exits with code 130 (standard SIGINT exit code)

Commands that source `wt-common.sh` and perform multi-step destructive operations (`wt-create`, `wt-delete`) SHALL set `trap wt_cleanup_on_signal INT TERM` after sourcing the library.

#### Scenario: Ctrl-C during wt-create
- **GIVEN** `wt-create` is mid-execution after `git worktree add` succeeded
- **WHEN** the user sends SIGINT (Ctrl-C)
- **THEN** `wt_cleanup_on_signal` fires
- **AND** `wt_rollback` removes the partial worktree
- **AND** the process exits with code 130

#### Scenario: SIGTERM during wt-delete
- **GIVEN** `wt-delete` is mid-execution
- **WHEN** the process receives SIGTERM
- **THEN** `wt_cleanup_on_signal` fires
- **AND** `wt_rollback` executes any registered cleanup
- **AND** the process exits with code 130

### Requirement: Signal Trap in wt-delete

`wt-delete` SHALL register signal traps (`INT TERM`) after sourcing `wt-common.sh`. When stash operations are in progress, the signal handler SHALL NOT leave the repository in a state where a stash was created but the worktree was not yet removed (the hash-based stash approach makes this safer since the stash is referenced by hash, not stack position).

#### Scenario: Ctrl-C after stash but before worktree removal
- **GIVEN** `wt-delete` has stashed changes (hash captured)
- **WHEN** the user sends SIGINT before worktree removal completes
- **THEN** the stash entry is preserved (recoverable via `git stash list`)
- **AND** the worktree is NOT removed (incomplete operation aborted)

## wt Package: Hash-Based Stash

### Requirement: Replace Index-Based Stash with Hash-Based Stash

`wt-common.sh` SHALL provide `wt_stash_create` and `wt_stash_apply` functions that use `git stash create` (hash-based) instead of `git stash push`/`git stash pop` (index-based).

`wt_stash_create` SHALL:
1. Accept a message argument
2. Stage all files including untracked (`git add -A`), suppressing all output (`>/dev/null 2>&1`)
3. Create a stash object via `git stash create`
4. If a hash is returned (changes exist):
   a. Store the stash in the reflog via `git stash store <hash> -m <msg>` (makes it visible in `git stash list`)
   b. Reset the working tree (`git reset --hard HEAD` + `git clean -fd`)
5. Echo the hash to stdout (must be the only stdout output for clean capture by callers)
6. If no hash is returned (no changes), output nothing
<!-- clarified: git add output suppressed to prevent stdout contamination; git stash store added for reflog recovery -->

`wt_stash_apply` SHALL:
1. Accept a hash argument
2. Apply the stash via `git stash apply <hash>`
3. Be a no-op if the hash is empty

#### Scenario: Concurrent stash safety
- **GIVEN** `wt-delete --stash` captures stash hash `abc123`
- **WHEN** another terminal runs `git stash push` before restore
- **THEN** `wt_stash_apply abc123` still restores the correct changes
- **AND** the other terminal's stash is unaffected

#### Scenario: No changes to stash
- **GIVEN** the worktree has no uncommitted changes and no untracked files
- **WHEN** `wt_stash_create` is called
- **THEN** `git stash create` returns empty
- **AND** `wt_stash_create` outputs nothing (no hash)
- **AND** no `git reset` or `git clean` is executed

### Requirement: Migrate wt-delete to Hash-Based Stash

`wt-delete` SHALL replace its existing `wt_stash_changes` usage with the new `wt_stash_create`/`wt_stash_apply` pattern. The stash hash SHALL be captured and registered with the rollback stack so that signal interruption can attempt restore.

The existing `wt_stash_changes` function in `wt-delete` SHALL be removed or replaced. The shared `wt_stash_create`/`wt_stash_apply` in `wt-common.sh` SHALL be the sole stash mechanism.

#### Scenario: wt-delete --stash preserves and restores changes
- **GIVEN** a worktree with uncommitted changes
- **WHEN** `wt-delete --stash` is run
- **THEN** changes are stashed via `wt_stash_create` (hash captured)
- **AND** the worktree is removed
- **AND** the stash hash is printed for recovery

## wt Package: Branch Name Validation

### Requirement: Pre-Validation of Branch Names

`wt-common.sh` SHALL provide a `wt_validate_branch_name` function that validates branch names against git ref naming rules. The function SHALL reject:

- Empty strings
- Names containing whitespace, `~`, `^`, `:`, `?`, `*`, `[`
- Names containing `..` (double dot)
- Names ending with `.lock`
- Names starting with `.`
- Names containing `/.` (hidden component)

The function SHALL return 0 on valid input, 1 on invalid input.

#### Scenario: Valid branch name accepted
- **GIVEN** a branch name `feature/add-auth`
- **WHEN** `wt_validate_branch_name` is called
- **THEN** it returns 0

#### Scenario: Branch name with invalid characters rejected
- **GIVEN** a branch name `feature~bad`
- **WHEN** `wt_validate_branch_name` is called
- **THEN** it returns 1

#### Scenario: Branch name with double dot rejected
- **GIVEN** a branch name `feature..branch`
- **WHEN** `wt_validate_branch_name` is called
- **THEN** it returns 1

#### Scenario: Branch name ending in .lock rejected
- **GIVEN** a branch name `feature.lock`
- **WHEN** `wt_validate_branch_name` is called
- **THEN** it returns 1

### Requirement: Validation Integrated in wt-create and wt-pr

`wt-create` SHALL call `wt_validate_branch_name` on the branch argument (when provided) before any git operations. On failure, it SHALL exit with `wt_error "Invalid branch name" "Branch name '<name>' contains invalid characters" "Use alphanumeric characters, hyphens, and single slashes"`.

`wt-pr` SHALL call `wt_validate_branch_name` on the PR's head branch after fetching it from GitHub. On failure (unexpected, since GitHub enforces its own rules), it SHALL exit with a descriptive error.

#### Scenario: wt-create with invalid branch name
- **GIVEN** a user runs `wt-create "bad~branch"`
- **WHEN** argument parsing completes
- **THEN** `wt_validate_branch_name` returns 1
- **AND** `wt-create` exits with a clear error message before any git operations

## wt Package: Dirty-State Check

### Requirement: Uncommitted Changes Warning in wt-create

`wt-create` SHALL check for uncommitted changes and untracked files in the current repository before creating a worktree. If either exists, it SHALL display a warning and present a menu:

1. **Continue anyway** — proceed with creation (worktree gets committed state, not dirty state)
2. **Stash changes first** — stash via `wt_stash_create` then proceed
3. **Abort** — exit 0

In `--non-interactive` mode, the default SHALL be "Continue anyway" (creating a worktree from dirty state is not broken — the worktree gets the committed state regardless).

#### Scenario: Interactive mode with dirty state
- **GIVEN** the main repo has uncommitted changes
- **WHEN** `wt-create` is run interactively
- **THEN** a yellow warning is displayed
- **AND** a 3-option menu is presented
- **AND** selecting "Stash changes first" stashes via `wt_stash_create`

#### Scenario: Non-interactive mode with dirty state
- **GIVEN** the main repo has untracked files
- **WHEN** `wt-create --non-interactive` is run
- **THEN** creation proceeds silently (no warning, no prompt)
- **AND** the worktree is created from the committed state

#### Scenario: Clean state skips check
- **GIVEN** the main repo has no uncommitted changes and no untracked files
- **WHEN** `wt-create` is run
- **THEN** no dirty-state warning is shown
- **AND** creation proceeds normally

## wt Package: PR Worktree Command

### Requirement: wt-pr Command

A new command `wt-pr` SHALL be created at `fab/.kit/packages/wt/bin/wt-pr` that creates a worktree from a GitHub PR.

`wt-pr` SHALL:
1. Source `wt-common.sh` for shared functions
2. Validate `gh` CLI is available (exit with `wt_error` if missing)
3. Accept an optional PR number as a positional argument
4. Accept the same `--worktree-name`, `--worktree-init`, `--worktree-open`, and `--non-interactive` flags as `wt-create`
5. Accept `help`, `--help`, `-h` to display usage

#### Scenario: wt-pr with PR number
- **GIVEN** `gh` CLI is installed and authenticated
- **WHEN** `wt-pr 42` is run
- **THEN** `gh pr view 42 --json headRefName --jq .headRefName` retrieves the branch name
- **AND** `git fetch origin` fetches the PR's branch
- **AND** a worktree is created for that branch (delegating to `wt_create_branch_worktree`)
- **AND** init and open follow the same logic as `wt-create`

#### Scenario: wt-pr without PR number (interactive)
- **GIVEN** `gh` CLI is installed
- **WHEN** `wt-pr` is run without arguments in interactive mode
- **THEN** `gh pr list --json number,title,headRefName` retrieves open PRs
- **AND** a menu is presented via `wt_show_menu`
- **AND** the selected PR's branch is used for worktree creation

#### Scenario: wt-pr without gh CLI
- **GIVEN** `gh` CLI is not installed
- **WHEN** `wt-pr` is run
- **THEN** it exits with `wt_error "gh CLI not found" ...`

#### Scenario: wt-pr non-interactive without PR number
- **GIVEN** `--non-interactive` is set and no PR number is provided
- **WHEN** `wt-pr --non-interactive` is run
- **THEN** it exits with `wt_error "No PR number specified" "PR number is required in non-interactive mode" "Example: wt-pr --non-interactive 42"`
<!-- clarified: wt-pr --non-interactive requires explicit PR number to avoid menu hang -->

### Requirement: wt-pr Fetch Strategy

`wt-pr` SHALL fetch the PR branch using `git fetch origin <branch>:<branch>` for remote branches not yet local. If the branch already exists locally, it SHALL use the local branch directly (same logic as `wt_create_branch_worktree`).

`wt-pr` SHALL call `wt_validate_branch_name` on the retrieved branch name before proceeding.

#### Scenario: PR branch not yet local
- **GIVEN** PR #42 has branch `feature/login` which is not local
- **WHEN** `wt-pr 42` is run
- **THEN** `git fetch origin feature/login:feature/login` is run
- **AND** a worktree is created for `feature/login`

#### Scenario: PR branch already local
- **GIVEN** PR #42 has branch `feature/login` which already exists locally
- **WHEN** `wt-pr 42` is run
- **THEN** no fetch is needed
- **AND** a worktree is created for the existing local branch

## Design Decisions

1. **Rollback functions in `wt-common.sh`, not per-command**: The rollback stack is a generic mechanism useful for any multi-step command. Placing it in the shared library follows the existing pattern (all `wt_*` utilities live in `wt-common.sh`) and enables reuse by `wt-pr` and future commands.
   - *Why*: Single source of truth, consistent behavior across commands
   - *Rejected*: Per-command rollback logic — duplicates code, higher maintenance burden

2. **`git stash create` (hash-based) over `git stash push` (index-based)**: The hash-based approach produces a stable reference unaffected by concurrent stash operations in other terminals. `git stash create` generates a commit object without adding to the stash reflog; `git stash apply <hash>` restores from that object.
   - *Why*: Eliminates race condition with concurrent terminal sessions
   - *Rejected*: Named stash with `git stash push -m` + search by name — fragile, requires parsing `git stash list`

3. **`wt-pr` delegates to `wt_create_branch_worktree` rather than reimplementing**: The PR command's unique responsibility is resolving a PR to a branch name. All worktree creation logic (name prompting, collision detection, init, open) is reused from existing functions.
   - *Why*: Avoids duplicating ~100 lines of creation logic; bug fixes propagate automatically
   - *Rejected*: Standalone implementation — higher maintenance, inconsistent behavior risk

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rollback functions go in `wt-common.sh` | Confirmed from intake #1 — `wt-common.sh` is the existing shared library; all bin scripts source it | S:90 R:95 A:95 D:95 |
| 2 | Certain | `wt-pr` follows existing command patterns (same flag style, sourcing, error functions) | Confirmed from intake #2 — 5 existing commands establish a clear pattern | S:85 R:90 A:95 D:95 |
| 3 | Confident | Hash-based stash uses `git stash create` + `git reset --hard` | Confirmed from intake #3 — avoids index-based fragility, well-documented git internals | S:80 R:70 A:80 D:85 |
| 4 | Confident | `wt-pr` requires `gh` CLI (no REST API fallback) | Confirmed from intake #4 — project already uses `gh`; REST fallback adds complexity for a niche case | S:70 R:80 A:75 D:70 |
| 5 | Confident | Dirty-state check defaults to "continue" in non-interactive mode | Confirmed from intake #5 — worktree creation from dirty state is safe (committed state is used) | S:75 R:90 A:80 D:75 |
| 6 | Certain | GitLab MR support deferred — GitHub-only | User confirmed: no GitLab usage, no plans to add. Upgraded from Tentative | S:90 R:85 A:90 D:95 |
| 7 | Confident | EXIT trap used for rollback (not only INT/TERM) | EXIT trap catches all exit paths including `set -e` failures, which INT/TERM alone would miss | S:75 R:85 A:85 D:80 |
| 8 | Certain | Tests go in `src/packages/wt/tests/` using bats-core | Existing test infrastructure uses bats at that path; 6+ test files already exist | S:95 R:95 A:95 D:95 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
<!-- clarified: GitHub-only for wt-pr — user confirmed, upgraded from Tentative to Certain -->

## Clarifications

### Session 2026-02-18

1. **Q: Should `wt_rollback` disable `set -e` to prevent trap handler from exiting early?**
   A: Yes — `wt_rollback` will `set +e` at entry, restore at exit. Guarantees all rollback entries execute.

2. **Q: What should `wt-pr --non-interactive` do without a PR number?**
   A: Error and exit. Consistent with `wt-delete`'s non-interactive behavior — explicit argument required.

3. **Q: Should `wt_stash_create` call `git stash store` to make stash visible in `git stash list`?**
   A: Yes — `git stash store <hash> -m <msg>` after create. Safety net for hash loss.

4. **Q: Should `git add -A` output be suppressed in `wt_stash_create`?**
   A: Yes — redirect all output (`>/dev/null 2>&1`) to keep stdout clean for hash capture.
