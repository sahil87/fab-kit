# Spec: Add --base flag to wt create

**Change**: 260312-wrk6-add-wt-create-base-flag
**Created**: 2026-03-12
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Change ID resolution within `wt create` itself — resolution uses the existing change folder name as a branch name (folder == branch by naming convention), not `fab resolve`
- Modifying `wt delete` or other wt subcommands — `--base` only applies to creation
- Automatic dependency detection — the operator must explicitly specify `--base`; no DAG inference

## wt: `--base` Flag for Branch Start-Point

### Requirement: `--base` flag on `wt create`

`wt create` SHALL accept a `--base <ref>` flag that specifies the git start-point when creating a new branch. When `--base` is provided and a new branch is being created, the underlying `git worktree add -b <branch> <path> <start-point>` command SHALL use the `--base` value as the start-point.

#### Scenario: New branch with --base

- **GIVEN** a repository with branch `feature-A` containing a marker commit
- **WHEN** `wt create feature-B --base feature-A --worktree-name bravo` is run
- **THEN** the new worktree is created with branch `feature-B` based on `feature-A`'s tip
- **AND** the worktree contains `feature-A`'s commits (marker file present)
- **AND** the worktree's HEAD matches `feature-A`'s tip, not main's HEAD

#### Scenario: Exploratory worktree with --base

- **GIVEN** a repository with branch `feature-A` containing a marker commit
- **WHEN** `wt create --base feature-A --worktree-name explore` is run (no branch arg)
- **THEN** an exploratory worktree is created with a branch based on `feature-A`'s tip
- **AND** the worktree contains `feature-A`'s commits

#### Scenario: New branch without --base (regression guard)

- **GIVEN** a repository on `main`
- **WHEN** `wt create new-feature --worktree-name wt-new` is run without `--base`
- **THEN** the branch is created from HEAD (current behavior unchanged)

### Requirement: `--base` ignored for existing branches

When `--base` is provided but the target branch already exists (locally or remotely), the flag SHALL be ignored with a warning on stderr.

#### Scenario: --base with existing local branch

- **GIVEN** branch `existing-branch` exists locally
- **WHEN** `wt create existing-branch --base some-ref --worktree-name wt-exist` is run
- **THEN** the worktree checks out `existing-branch` at its current tip
- **AND** stderr contains `--base ignored: branch already exists locally`
- **AND** `--base` has no effect on the checked-out state

#### Scenario: --base with existing remote branch

- **GIVEN** branch `remote-branch` exists only on the remote
- **WHEN** `wt create remote-branch --base some-ref --worktree-name wt-remote` is run
- **THEN** the remote branch is fetched and checked out normally
- **AND** stderr contains `--base ignored: fetching existing remote branch`

### Requirement: `--base` with invalid ref errors

When `--base` is provided with a ref that does not exist as a git ref (branch, tag, or commit), the command SHALL exit with a non-zero exit code and an error message.

#### Scenario: Invalid --base ref

- **GIVEN** no branch, tag, or commit named `nonexistent-ref` exists
- **WHEN** `wt create new-branch --base nonexistent-ref --worktree-name wt-bad` is run
- **THEN** the command exits with a non-zero exit code
- **AND** stderr contains an error about the unresolvable base ref
- **AND** no worktree or branch is created

### Requirement: `--base` with `--reuse` defers to reuse

When both `--base` and `--reuse` are provided and the worktree already exists, `--reuse` SHALL take precedence — the existing worktree path is returned and `--base` has no effect.

#### Scenario: --base with --reuse on existing worktree

- **GIVEN** worktree `reuse-base` already exists
- **WHEN** `wt create --base some-ref --reuse --worktree-name reuse-base` is run
- **THEN** the existing worktree path is returned
- **AND** `--base` has no effect

## wt: `CreateWorktree` Start-Point Parameter

### Requirement: `CreateWorktree` accepts a start-point

The `CreateWorktree` function in `src/go/wt/internal/worktree/crud.go` SHALL accept a `startPoint string` parameter. When `startPoint` is non-empty and `newBranch` is true, the git command SHALL be `git worktree add -b <branch> <path> <startPoint>`. When `startPoint` is empty, behavior SHALL be unchanged.

#### Scenario: CreateWorktree with startPoint and newBranch

- **GIVEN** `startPoint` is `"feature-A"` and `newBranch` is `true`
- **WHEN** `CreateWorktree(path, "feature-B", true, "feature-A")` is called
- **THEN** the git command executed is `git worktree add -b feature-B <path> feature-A`

#### Scenario: CreateWorktree with empty startPoint

- **GIVEN** `startPoint` is `""`
- **WHEN** `CreateWorktree(path, branch, true, "")` is called
- **THEN** the git command is `git worktree add -b <branch> <path>` (unchanged)

#### Scenario: CreateWorktree with startPoint and existing branch

- **GIVEN** `startPoint` is `"feature-A"` and `newBranch` is `false`
- **WHEN** `CreateWorktree(path, branch, false, "feature-A")` is called
- **THEN** `startPoint` is ignored — command is `git worktree add <path> <branch>`

### Requirement: Higher-level functions pass startPoint through

`CreateBranchWorktree` and `CreateExploratoryWorktree` SHALL accept a `startPoint string` parameter and pass it through to `CreateWorktree` when creating new branches.

#### Scenario: CreateBranchWorktree with startPoint on new branch

- **GIVEN** branch `new-feature` does not exist locally or remotely
- **WHEN** `CreateBranchWorktree("new-feature", "wt-name", ctx, rb, "feature-A")` is called
- **THEN** `CreateWorktree` is called with `startPoint="feature-A"`

#### Scenario: CreateBranchWorktree with startPoint on existing branch

- **GIVEN** branch `existing` exists locally
- **WHEN** `CreateBranchWorktree("existing", "wt-name", ctx, rb, "feature-A")` is called
- **THEN** `CreateWorktree` is called with `startPoint=""` (ignored for existing branches)

## wt: `--base` Ref Validation

### Requirement: `--base` value validated as git ref

When `--base <ref>` is provided, the command SHALL validate that `<ref>` resolves to a valid git object (via `git rev-parse --verify <ref>`) before creating the worktree. This applies only when the ref will actually be used (new branch creation).

#### Scenario: Valid branch as --base

- **GIVEN** branch `feature-A` exists
- **WHEN** `--base feature-A` is provided for a new branch
- **THEN** validation passes and the branch is created from `feature-A`

#### Scenario: Valid commit SHA as --base

- **GIVEN** a valid commit SHA exists
- **WHEN** `--base <sha>` is provided
- **THEN** validation passes and the branch is created from that commit

## Operator2: `--base` Usage in Autopilot

### Requirement: Operator2 autopilot uses `--base` for user-provided ordering

The operator2 skill's autopilot per-change loop SHALL use `--base <previous-change-id>` when creating worktrees for sequentially dependent changes in user-provided ordering. The previous change in the queue provides the base.

#### Scenario: Sequential dependency in autopilot queue

- **GIVEN** the autopilot queue contains [change-A, change-B] with user-provided ordering
- **WHEN** creating the worktree for change-B
- **THEN** `wt create --non-interactive --reuse --worktree-name <B-name> <B-branch> --base <A-folder-name>` is used
- **AND** `<A-folder-name>` is the full folder name of change-A (which equals its branch name)

#### Scenario: Confidence-based ordering (no dependency)

- **GIVEN** the autopilot queue is sorted by confidence score
- **WHEN** creating worktrees for any change
- **THEN** `--base` is NOT used (changes are independent)

## Spec: packages.md Update

### Requirement: packages.md documents `--base`

The `docs/specs/packages.md` wt section SHALL document the `--base` flag with its behavior for new branches, existing branches (warn-and-ignore), and exploratory worktrees.

#### Scenario: Documentation accuracy

- **GIVEN** the `--base` flag is implemented
- **WHEN** a user reads `docs/specs/packages.md`
- **THEN** the documentation accurately describes `--base` behavior, including the warn-and-ignore for existing branches

## Design Decisions

1. **Branch name == change folder name for `--base` resolution**: The `--base` flag accepts raw git refs directly. When the operator passes a change folder name as `--base`, it works because fab's naming convention means the folder name IS the branch name. No separate fab resolve step is needed inside `wt create`.
   - *Why*: Keeps `wt` package independent of fab internals — `wt` knows nothing about `fab/changes/` or `fab resolve`. The naming convention is the bridge.
   - *Rejected*: Adding `fab resolve` as a dependency inside `wt create` — would couple the generic worktree tool to fab's change management, violating the packages' independence.

2. **Warn-and-ignore for `--base` with existing branches**: Rather than erroring when `--base` is provided with an existing branch, the command warns and proceeds.
   - *Why*: Reduces friction in automation (operator doesn't need to check branch existence before passing `--base`). The warning makes the no-op visible for debugging.
   - *Rejected*: Hard error — would require the operator to conditionally construct the command, adding complexity.

3. **Validate `--base` ref before creating worktree**: The ref is validated with `git rev-parse --verify` before the worktree is created.
   - *Why*: Provides a clear error message instead of a cryptic git failure. Also prevents partial state (worktree dir created but branch creation fails).
   - *Rejected*: Letting git fail naturally — error messages are less clear and cleanup is messier.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Go implementation in existing wt package | Confirmed from intake #1 — wt is a Go binary, this extends it | S:95 R:90 A:95 D:95 |
| 2 | Certain | Use git worktree add start-point argument | Confirmed from intake #2 — this is the direct git primitive | S:90 R:95 A:95 D:95 |
| 3 | Certain | Branch name == change folder name for resolution | Upgraded from intake #3 Confident — spec clarifies wt has no fab dependency; naming convention is the bridge | S:90 R:85 A:90 D:90 |
| 4 | Confident | Warn-and-ignore for --base with existing branches | Confirmed from intake #4 — existing branches have a history, --base is meaningless | S:85 R:90 A:80 D:75 |
| 5 | Confident | Operator uses --base only for user-provided ordering | Confirmed from intake #5 — confidence-based ordering implies independence | S:75 R:80 A:70 D:70 |
| 6 | Certain | Validate --base ref with git rev-parse before use | Decided during spec — prevents partial state and gives clear errors | S:85 R:90 A:90 D:90 |
| 7 | Confident | startPoint parameter ignored for existing branch checkout | Follows from warn-and-ignore design — existing branches don't need a start-point | S:80 R:85 A:80 D:80 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
