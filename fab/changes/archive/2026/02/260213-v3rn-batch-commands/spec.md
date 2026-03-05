# Spec: Rename Batch Scripts and Add Batch Archive

**Change**: 260213-v3rn-batch-commands
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/kit-architecture.md` (modify)

## Scripts: Batch Naming Convention

### Requirement: Batch scripts SHALL follow `batch-{verb}-{entity}.sh` naming

All shell scripts in `fab/.kit/scripts/` that perform batch operations across multiple worktrees SHALL use the naming pattern `batch-{verb}-{entity}.sh`. The `fab-` prefix SHALL be removed since the scripts already reside in `fab/.kit/scripts/`.

#### Scenario: Renamed scripts maintain identical behavior

- **GIVEN** `fab-batch-new.sh` exists at `fab/.kit/scripts/`
- **WHEN** it is renamed to `batch-new-backlog.sh`
- **THEN** the script content is identical except for the comment header referencing the new name
- **AND** no other files reference the old name

#### Scenario: Renamed switch script maintains identical behavior

- **GIVEN** `fab-batch-switch.sh` exists at `fab/.kit/scripts/`
- **WHEN** it is renamed to `batch-switch-change.sh`
- **THEN** the script content is identical except for the comment header referencing the new name
- **AND** no other files reference the old name

### Requirement: Old script files SHALL be deleted after rename

The original `fab-batch-new.sh` and `fab-batch-switch.sh` files MUST be removed. They SHALL NOT remain as dead files alongside the renamed versions.

#### Scenario: No duplicate scripts after rename

- **GIVEN** the rename has been applied
- **WHEN** listing `fab/.kit/scripts/`
- **THEN** `fab-batch-new.sh` and `fab-batch-switch.sh` do not exist
- **AND** `batch-new-backlog.sh` and `batch-switch-change.sh` exist

## Scripts: Batch Archive

### Requirement: `batch-archive-change.sh` SHALL archive completed changes

A new script `batch-archive-change.sh` SHALL be created at `fab/.kit/scripts/`. It SHALL open a tmux tab per change, each running a Claude Code session that invokes `/fab-archive` on the target change.

<!-- clarified: Target hydrate:done with /fab-archive — confirmed by /fab-archive skill's Hydrate Guard (requires progress.hydrate: done) -->

#### Scenario: Archive a single completed change

- **GIVEN** change `260213-v3rn-batch-commands` has `hydrate: done` in its `.status.yaml`
- **AND** the user is inside a tmux session
- **WHEN** running `batch-archive-change.sh v3rn`
- **THEN** a new tmux tab opens with a Claude Code session running `/fab-archive v3rn`
- **AND** the tmux tab is named `fab-260213-v3rn-batch-commands`

#### Scenario: Archive multiple changes

- **GIVEN** three changes have `hydrate: done`
- **WHEN** running `batch-archive-change.sh --all`
- **THEN** three tmux tabs open, each running `/fab-archive <change-name>`

#### Scenario: Skip changes not ready for archive

- **GIVEN** change `260213-abc` has `review: done` (hydrate not yet done)
- **WHEN** running `batch-archive-change.sh --all`
- **THEN** change `260213-abc` is skipped
- **AND** a warning is printed: "Warning: '260213-abc' not ready for archive (hydrate not done), skipping"

### Requirement: Archive script SHALL follow existing batch script patterns

The new script SHALL reuse the same structural patterns as `batch-new-backlog.sh` and `batch-switch-change.sh`:

- `set -euo pipefail` and `SCRIPT_DIR`/`KIT_DIR`/`FAB_DIR` boilerplate
- `usage()` function with examples
- `--list` option to show archivable changes (hydrate:done)
- `--all` option to open tabs for all archivable changes
- `-h`/`--help` for usage
- Substring matching for change name resolution (same logic as `batch-switch-change.sh`)
- Worktree + tmux + Claude Code tab pattern via `wt-create`

#### Scenario: List archivable changes

- **GIVEN** two changes have `hydrate: done` and one has `review: done`
- **WHEN** running `batch-archive-change.sh --list`
- **THEN** only the two hydrate:done changes are listed

#### Scenario: No archivable changes exist

- **GIVEN** no changes have `hydrate: done`
- **WHEN** running `batch-archive-change.sh --all`
- **THEN** the script prints "No archivable changes found." and exits with code 1

### Requirement: Archive script SHALL detect hydrate:done from `.status.yaml`

The script SHALL read each change's `.status.yaml` to determine if `progress.hydrate` is `done`. It SHALL use grep/sed (not a YAML parser) consistent with how `batch-switch-change.sh` reads `config.yaml`.

#### Scenario: Parse hydrate status

- **GIVEN** a change's `.status.yaml` contains `hydrate: done`
- **WHEN** the script checks if the change is archivable
- **THEN** the change is included in the archivable set

#### Scenario: Missing `.status.yaml`

- **GIVEN** a change directory exists but has no `.status.yaml`
- **WHEN** the script checks if the change is archivable
- **THEN** the change is skipped with a warning

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Same worktree + tmux + Claude pattern | Both existing batch scripts use this pattern; consistency is a clear win |
| 2 | Confident | Substring matching for change resolution | Existing `batch-switch-change.sh` implements this; reuse for consistency |
| 3 | Certain | Target `hydrate:done` with `/fab-archive` | Confirmed: `/fab-archive` skill has Hydrate Guard requiring `progress.hydrate: done` |

2 assumptions made (2 confident, 0 tentative).

## Clarifications

### Session 2026-02-14

| # | Question | Resolution |
|---|----------|------------|
| 1 | Target `hydrate:done` + `/fab-archive` vs `review:done` + `/fab-continue`? | Resolved to Certain — `/fab-archive` enforces Hydrate Guard (`progress.hydrate: done` required) |
