# Spec: Consolidate worktree-init into fab-sync

**Change**: 260218-bx4d-consolidate-worktree-init-into-sync
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Changing the external `wt-init` binary — it reads `$WORKTREE_INIT_SCRIPT` and that contract is preserved via env var update
- Adding new sync functionality — this is a pure restructure of existing behavior

## Bootstrap: Single Entry Point

### Requirement: fab-sync.sh as thin orchestrator

`fab/.kit/scripts/fab-sync.sh` SHALL be rewritten as a thin orchestrator that:

1. Derives `kit_dir`, `fab_dir`, and `repo_root` from its own location (same as current)
2. Iterates `fab/.kit/sync/*.sh` in sorted order, executing each with `bash "$script"`
3. Then iterates `fab/sync/*.sh` in sorted order (if the directory exists), executing each with `bash "$script"`

The orchestrator MUST set `set -euo pipefail`. It MUST print a header line identifying itself. It MUST print the script name before executing each script (matching the current `worktree-init.sh` pattern: `→ basename`).

The orchestrator MUST NOT contain any workspace sync logic itself — all sync logic moves into the iterated scripts.

#### Scenario: Normal execution with both directories

- **GIVEN** `fab/.kit/sync/` contains `1-direnv.sh` and `2-sync-workspace.sh`
- **AND** `fab/sync/` contains `1-symlink-backlog.sh`
- **WHEN** `fab-sync.sh` is executed
- **THEN** scripts execute in order: `1-direnv.sh`, `2-sync-workspace.sh`, `1-symlink-backlog.sh`
- **AND** each script name is printed before execution

#### Scenario: No project-specific sync directory

- **GIVEN** `fab/.kit/sync/` contains scripts
- **AND** `fab/sync/` does not exist
- **WHEN** `fab-sync.sh` is executed
- **THEN** only kit-level scripts execute
- **AND** no error is emitted for the missing project directory

#### Scenario: A script fails

- **GIVEN** `set -euo pipefail` is active
- **WHEN** any iterated script exits non-zero
- **THEN** the orchestrator halts immediately (no subsequent scripts run)

### Requirement: Workspace sync logic in 2-sync-workspace.sh

The current 470-line content of `fab/.kit/scripts/fab-sync.sh` SHALL be moved to `fab/.kit/sync/2-sync-workspace.sh` as-is, with only the path resolution adjusted.

The script MUST adjust its path derivation to account for its new location (`fab/.kit/sync/` instead of `fab/.kit/scripts/`):

- Current: `scripts_dir → kit_dir = dirname "$scripts_dir"`
- New: `sync_dir → kit_dir = dirname "$sync_dir"`

All remaining logic (sections 1 through 8, helper functions, pre-flight checks) SHALL remain unchanged.

#### Scenario: Path resolution correctness

- **GIVEN** `2-sync-workspace.sh` is located at `fab/.kit/sync/2-sync-workspace.sh`
- **WHEN** the script derives `kit_dir` from its own location
- **THEN** `kit_dir` resolves to `fab/.kit/`
- **AND** `fab_dir` resolves to `fab/`
- **AND** `repo_root` resolves to the repository root

#### Scenario: Idempotent execution

- **GIVEN** `2-sync-workspace.sh` has already been run
- **WHEN** it is run again
- **THEN** the output reports "OK" for already-configured items
- **AND** no data is lost or corrupted

### Requirement: direnv script in sync directory

`fab/.kit/worktree-init-common/1-direnv.sh` SHALL be moved to `fab/.kit/sync/1-direnv.sh` with identical content (`direnv allow`).

#### Scenario: direnv allow on every sync

- **GIVEN** `1-direnv.sh` exists in `fab/.kit/sync/`
- **WHEN** the orchestrator iterates kit-level scripts
- **THEN** `direnv allow` runs (idempotent, no guard needed)

## Directory Renames

### Requirement: Kit-level directory rename

`fab/.kit/worktree-init-common/` SHALL be renamed to `fab/.kit/sync/`. The contents after rename:

- `1-direnv.sh` — unchanged
- `2-sync-workspace.sh` — renamed from `2-rerun-sync-workspace.sh`, content replaced with the moved workspace sync logic

#### Scenario: Old directory removed

- **GIVEN** `fab/.kit/worktree-init-common/` exists
- **WHEN** the rename is applied
- **THEN** `fab/.kit/sync/` contains `1-direnv.sh` and `2-sync-workspace.sh`
- **AND** `fab/.kit/worktree-init-common/` no longer exists

### Requirement: Project-level directory rename

`fab/worktree-init/` SHALL be renamed to `fab/sync/`. During the rename:

- `1-claude-settings.sh` SHALL be deleted (section 8 of `2-sync-workspace.sh` already handles `settings.local.json`)
- `assets/` directory (containing `settings.local.json`) SHALL be deleted
- `2-symlink-backlog.sh` SHALL be renumbered to `1-symlink-backlog.sh` (content unchanged)

#### Scenario: Project sync directory after rename

- **GIVEN** `fab/worktree-init/` contains `1-claude-settings.sh`, `2-symlink-backlog.sh`, and `assets/`
- **WHEN** the rename is applied
- **THEN** `fab/sync/` contains only `1-symlink-backlog.sh`
- **AND** `fab/worktree-init/` no longer exists

## Deletions

### Requirement: Remove worktree-init.sh

`fab/.kit/worktree-init.sh` SHALL be deleted. Its orchestration role is replaced by the rewritten `fab-sync.sh`.

#### Scenario: Entry point replaced

- **GIVEN** `fab/.kit/worktree-init.sh` existed as the bootstrap entry point
- **WHEN** the change is applied
- **THEN** the file no longer exists
- **AND** `$WORKTREE_INIT_SCRIPT` points to `fab/.kit/scripts/fab-sync.sh` instead

## Configuration Updates

### Requirement: envrc scaffold update

`fab/.kit/scaffold/envrc` SHALL update the `WORKTREE_INIT_SCRIPT` line:

- Before: `export WORKTREE_INIT_SCRIPT=fab/.kit/worktree-init.sh`
- After: `export WORKTREE_INIT_SCRIPT=fab/.kit/scripts/fab-sync.sh`

All other lines in the scaffold SHALL remain unchanged.

#### Scenario: New worktrees get correct entry point

- **GIVEN** a new worktree is created
- **AND** `.envrc` is generated from the scaffold
- **WHEN** the `$WORKTREE_INIT_SCRIPT` env var is read
- **THEN** it points to `fab/.kit/scripts/fab-sync.sh`

### Requirement: Sync README scaffold

A new scaffold file `fab/.kit/scaffold/sync-readme.md` SHALL be created with content explaining that project-specific sync scripts go in `fab/sync/`, with naming convention guidance (numbered `*.sh` files, executed in sorted order after kit-level scripts).

`2-sync-workspace.sh` SHALL scaffold this README into `fab/sync/README.md` if the file does not exist (similar to how it scaffolds other files). This SHOULD be added as a new section in the sync-workspace script, after the existing sections.
<!-- assumed: README scaffolding added as a new section in 2-sync-workspace.sh — follows existing scaffold pattern (conditional create), minimal additions -->

#### Scenario: README created on first sync

- **GIVEN** `fab/sync/` exists
- **AND** `fab/sync/README.md` does not exist
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** `fab/sync/README.md` is created from the scaffold template

#### Scenario: README preserved on re-sync

- **GIVEN** `fab/sync/README.md` already exists
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** the existing file is not overwritten

## Documentation Updates

### Requirement: README.md references

`README.md` SHALL be updated to replace all references to `worktree-init.sh` and the old directory names with the new structure. Specifically:

- Any mention of `worktree-init.sh` → `fab-sync.sh` (as the bootstrap entry point)
- Any mention of `worktree-init-common/` or `worktree-init/` → `sync/` (where applicable)

#### Scenario: README accuracy

- **GIVEN** the change is complete
- **WHEN** a user reads `README.md`
- **THEN** all bootstrap flow references point to `fab-sync.sh` as the entry point
- **AND** no references to `worktree-init` remain

## Deprecated Requirements

### worktree-init.sh orchestrator

**Reason**: Replaced by `fab-sync.sh` as single entry point. The two-step orchestration (`worktree-init.sh` → `2-rerun-sync-workspace.sh` → `fab-sync.sh`) is collapsed into a direct iteration model.
**Migration**: `$WORKTREE_INIT_SCRIPT` env var now points to `fab/.kit/scripts/fab-sync.sh`.

### worktree-init-common/ directory

**Reason**: Renamed to `fab/.kit/sync/` to reflect its new role as the kit-level sync scripts directory.
**Migration**: Contents moved to `fab/.kit/sync/`.

### worktree-init/ directory

**Reason**: Renamed to `fab/sync/` to reflect its new role as the project-level sync scripts directory.
**Migration**: Contents moved to `fab/sync/` (minus deleted files).

### 1-claude-settings.sh

**Reason**: Duplicates functionality already in section 8 of the workspace sync script (`settings.local.json` JSON merge).
**Migration**: N/A — removed without replacement.

## Design Decisions

1. **Scripts self-locate via `$0`**: Each script in `fab/.kit/sync/` and `fab/sync/` derives its own paths from `$0`. The orchestrator does not pass context variables.
   - *Why*: Matches existing pattern. Scripts remain independently runnable for debugging. No coupling between orchestrator and script internals.
   - *Rejected*: Passing `$repo_root` as `$1` — would require all scripts to accept and parse arguments, breaking the current contract.

2. **Keep `WORKTREE_INIT_SCRIPT` env var name unchanged**: The env var name stays despite the script it points to changing.
   - *Why*: External `wt-init` binary reads this var. Renaming would require changes outside this repo.
   - *Rejected*: Renaming to `FAB_SYNC_SCRIPT` — would break external tooling.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `1-direnv.sh` runs on every sync without guard | Confirmed from intake #1 — `direnv allow` is idempotent | S:95 R:95 A:90 D:95 |
| 2 | Certain | Delete `1-claude-settings.sh` and `assets/` | Confirmed from intake #2 — section 8 of sync-workspace already handles `settings.local.json` via JSON merge | S:95 R:85 A:95 D:95 |
| 3 | Certain | Keep `WORKTREE_INIT_SCRIPT` env var name | Confirmed from intake #3 — external `wt-init` binary reads this var | S:95 R:90 A:90 D:95 |
| 4 | Certain | Kit-level scripts run before project-level scripts | Confirmed from intake #4 — project scripts depend on workspace being set up first | S:90 R:80 A:90 D:90 |
| 5 | Confident | Path resolution changes from `scripts_dir` to `sync_dir` | Confirmed from intake #5 — verified by reading source: `dirname` call goes from sync/ up to .kit/ instead of scripts/ up to .kit/ | S:85 R:90 A:85 D:80 |
| 6 | Confident | README scaffold added as new section in sync-workspace | Follows existing conditional-create scaffold pattern; minimal addition | S:75 R:90 A:80 D:75 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
