# Spec: Add --reuse flag to wt-create

**Change**: 260222-6ldg-wt-create-reuse-flag
**Created**: 2026-02-22
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Branch verification on reuse — `--reuse` returns the path blindly without checking which branch the worktree is on
- Modifying `batch-fab-new-backlog.sh` — fresh-creation scripts keep the hard error on collision
- Changing default `wt-create` behavior — `--reuse` is opt-in only

## wt-create: --reuse Flag

### Requirement: --reuse returns existing worktree path on name collision

When `--reuse` is passed and the target worktree directory already exists at `$WT_WORKTREES_DIR/$final_name`, `wt-create` SHALL return success (exit 0) and print the existing worktree path as the last line of stdout. It SHALL skip worktree creation, init script execution, and app opening.

#### Scenario: Reuse existing worktree

- **GIVEN** a worktree named `my-feature` already exists at `$WT_WORKTREES_DIR/my-feature`
- **WHEN** `wt-create --non-interactive --reuse --worktree-name my-feature some-branch` is invoked
- **THEN** the command exits 0
- **AND** the last line of stdout is the full path to the existing worktree
- **AND** no `git worktree add` command is executed
- **AND** no init script is run

#### Scenario: No collision — normal creation

- **GIVEN** no worktree named `new-feature` exists
- **WHEN** `wt-create --non-interactive --reuse --worktree-name new-feature some-branch` is invoked
- **THEN** the command creates the worktree normally (same behavior as without `--reuse`)
- **AND** the init script runs if configured
- **AND** the last line of stdout is the full path to the newly created worktree

### Requirement: --reuse MUST require --worktree-name

`--reuse` without `--worktree-name` SHALL produce an error and exit with `WT_EXIT_INVALID_ARGS` (exit code 2). The error message SHALL use the standard what/why/fix format.

#### Scenario: --reuse without --worktree-name

- **GIVEN** no `--worktree-name` flag is provided
- **WHEN** `wt-create --non-interactive --reuse` is invoked
- **THEN** the command exits with code 2
- **AND** stderr contains "Error: --reuse requires --worktree-name"

### Requirement: --reuse SHALL be blind to worktree state

`--reuse` SHALL NOT verify whether the existing directory is a registered git worktree, whether it is on the expected branch, or whether it is in a clean state. If the directory exists, the path is returned.

#### Scenario: Orphaned directory (not a git worktree)

- **GIVEN** a directory exists at `$WT_WORKTREES_DIR/orphaned` but is not registered as a git worktree
- **WHEN** `wt-create --non-interactive --reuse --worktree-name orphaned some-branch` is invoked
- **THEN** the command exits 0
- **AND** the last line of stdout is the path to the orphaned directory

### Requirement: --reuse argument parsing

`--reuse` SHALL be parsed as a boolean flag (no argument). It MAY appear in any position relative to other flags. It SHALL be included in `wt-create help` output.

#### Scenario: --reuse in help output

- **GIVEN** the user runs `wt-create help`
- **THEN** the output includes `--reuse` with a description

## Callers: batch-fab-switch-change.sh

### Requirement: batch-fab-switch-change.sh SHALL pass --reuse

The `wt-create` invocation in `batch-fab-switch-change.sh` SHALL include `--reuse` so that re-running the script after a partial failure reuses existing worktrees instead of erroring.

#### Scenario: Re-run after interruption

- **GIVEN** `batch-fab-switch-change.sh` was interrupted after creating worktrees for changes A and B
- **WHEN** the script is re-run with changes A, B, C, D
- **THEN** worktrees for A and B are reused (no error)
- **AND** worktrees for C and D are created normally

## Callers: dispatch.sh

### Requirement: dispatch.sh SHALL use --reuse and remove bespoke reuse check

The `create_worktree()` function in `dispatch.sh` SHALL pass `--reuse` to `wt-create` and SHALL remove the `wt_get_worktree_path_by_name` pre-check. The `wt-common.sh` source import for the pre-check SHALL also be removed.

#### Scenario: Pipeline resumes with existing worktree

- **GIVEN** a pipeline run was interrupted after creating a worktree for change X
- **WHEN** the pipeline is re-run
- **THEN** `dispatch.sh` reuses the existing worktree for change X via `--reuse`
- **AND** no `wt_get_worktree_path_by_name` call is made

## Callers: batch-fab-new-backlog.sh (no change)

### Requirement: batch-fab-new-backlog.sh SHALL NOT pass --reuse

`batch-fab-new-backlog.sh` creates worktrees for new backlog items. Name collisions indicate a genuine conflict (backlog ID already in-flight). The hard error SHALL be preserved.

#### Scenario: Collision on fresh creation

- **GIVEN** a worktree named `90g5` already exists (from a previous backlog run)
- **WHEN** `batch-fab-new-backlog.sh 90g5` is invoked
- **THEN** the command errors with "already exists" (unchanged behavior)

## Tests: wt-create.bats

### Requirement: New test section for --reuse flag

`src/packages/wt/tests/wt-create.bats` SHALL include a new `# --reuse Flag Tests` section with tests covering: reuse on collision, normal creation with --reuse, init script skipping, --reuse without --worktree-name validation, and output contract (path as last line).

#### Scenario: Test coverage

- **GIVEN** the test suite runs
- **WHEN** the `--reuse Flag Tests` section executes
- **THEN** all 5 test cases pass

### Requirement: Edge case test for orphaned directory

`src/packages/wt/tests/edge-cases.bats` SHALL include a test that creates a bare directory (not a git worktree) at the expected worktree path and verifies `--reuse` returns it successfully.

#### Scenario: Orphaned directory test

- **GIVEN** a directory is created manually at `$WT_WORKTREES_DIR/orphaned`
- **WHEN** `wt-create --non-interactive --reuse --worktree-name orphaned some-branch` is invoked
- **THEN** the command exits 0
- **AND** the last line of stdout is the path to the orphaned directory

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `--reuse` returns path blindly without branch verification | Confirmed from intake #1 — user explicitly chose blind reuse | S:95 R:80 A:95 D:95 |
| 2 | Certain | `--reuse` requires `--worktree-name` | Confirmed from intake #2 — random names wouldn't collide intentionally | S:90 R:90 A:90 D:90 |
| 3 | Certain | `batch-fab-switch-change.sh` and `dispatch.sh` get `--reuse`; `batch-fab-new-backlog.sh` does not | Confirmed from intake #3 — resume vs fresh-creation semantics | S:95 R:85 A:95 D:95 |
| 4 | Certain | `--reuse` skips init script and app opening on collision | Confirmed from intake #4 — re-running init on existing worktree would be surprising | S:85 R:85 A:85 D:90 |
| 5 | Confident | Remove bespoke reuse check and wt-common.sh import from `dispatch.sh` | Confirmed from intake #5 — `wt_get_worktree_path_by_name` becomes redundant; removing the import keeps dispatch.sh clean | S:80 R:70 A:80 D:85 |
| 6 | Confident | Orphaned directory edge case tested | Confirmed from intake #6 — blind reuse means directory existence is the only check | S:75 R:75 A:80 D:80 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
