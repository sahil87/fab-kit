# Spec: Replace .envrc Symlink with Line-Ensuring Sync

**Change**: 260216-ymvx-DEV-1043-envrc-line-sync
**Created**: 2026-02-16
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`

## fab-sync: .envrc Line-Ensuring

### Requirement: Line-Ensuring .envrc Sync

`fab-sync.sh` section 2 SHALL replace the current symlink-based `.envrc` management with a line-ensuring pattern identical to the `.gitignore` handling in section 7.

The script SHALL read lines from `fab/.kit/scaffold/envrc`, skip comments (`#`-prefixed) and empty lines, and ensure each remaining line exists in the project's `.envrc`. Missing lines SHALL be appended. If `.envrc` does not exist, it SHALL be created with the first required line, then subsequent lines appended.

The `fab/.kit/scaffold/envrc` file SHALL NOT be modified — it continues to serve as the source of required lines.

#### Scenario: New project — no .envrc exists
- **GIVEN** the project has no `.envrc` file
- **WHEN** `fab-sync.sh` runs
- **THEN** `.envrc` is created as a normal file containing all non-comment, non-empty lines from `scaffold/envrc`
- **AND** the script outputs `Created: .envrc (added <lines>)`

#### Scenario: Existing .envrc with all required lines present
- **GIVEN** `.envrc` exists as a normal file
- **AND** all required lines from `scaffold/envrc` are already present
- **WHEN** `fab-sync.sh` runs
- **THEN** `.envrc` is not modified
- **AND** the script outputs `.envrc: OK`

#### Scenario: Existing .envrc missing some required lines
- **GIVEN** `.envrc` exists as a normal file
- **AND** some lines from `scaffold/envrc` are not present
- **WHEN** `fab-sync.sh` runs
- **THEN** the missing lines are appended to `.envrc`
- **AND** the script outputs `Updated: .envrc (added <lines>)`
- **AND** existing user-added lines in `.envrc` are preserved

#### Scenario: scaffold/envrc not found
- **GIVEN** `fab/.kit/scaffold/envrc` does not exist
- **WHEN** `fab-sync.sh` runs
- **THEN** the `.envrc` section is skipped entirely (no error, no output for this section)

### Requirement: Symlink Migration

When `.envrc` is a symlink (from a previous fab version), `fab-sync.sh` SHALL migrate it to a real file before applying line-ensuring logic.

The migration SHALL resolve the symlink's content via `cat`, remove the symlink, and write the content to a real `.envrc` file. If the symlink target is unreadable (broken symlink), the symlink SHALL be removed and a new `.envrc` created from scratch by the line-ensuring logic.

#### Scenario: Existing .envrc is a valid symlink
- **GIVEN** `.envrc` is a symlink pointing to `fab/.kit/scaffold/envrc` (or any valid target)
- **WHEN** `fab-sync.sh` runs
- **THEN** the symlink content is read into memory
- **AND** the symlink is removed
- **AND** a real file is written with the resolved content
- **AND** the script outputs `.envrc: migrated from symlink to file`
- **AND** line-ensuring logic then runs on the real file (appending any missing lines)

#### Scenario: Existing .envrc is a broken symlink
- **GIVEN** `.envrc` is a symlink whose target does not exist
- **WHEN** `fab-sync.sh` runs
- **THEN** the broken symlink is removed
- **AND** line-ensuring logic creates a new `.envrc` from `scaffold/envrc`
- **AND** the script outputs `.envrc: migrated from symlink to file` followed by `Created: .envrc (added <lines>)`

### Requirement: Line Matching Semantics

Line matching SHALL use exact fixed-string matching (`grep -qxF`) — the same approach used by section 7 (.gitignore). Lines are compared verbatim, including leading/trailing whitespace. Partial matches do not count.

#### Scenario: Line exists with different whitespace
- **GIVEN** `.envrc` contains `PATH_add  fab/.kit/scripts` (double space)
- **AND** `scaffold/envrc` contains `PATH_add fab/.kit/scripts` (single space)
- **WHEN** `fab-sync.sh` runs
- **THEN** the scaffold line is appended (exact match not found)

## Deprecated Requirements

### Symlink-Based .envrc Management
**Reason**: Replaced by line-ensuring sync. The symlink approach prevented projects from adding their own `.envrc` lines alongside fab's entries.
**Migration**: Existing symlinks are automatically migrated to real files on the next `fab-sync.sh` run. Users may need to re-run `direnv allow` after migration since direnv tracks file identity.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use the same line-ensuring pattern as .gitignore section 7 | User explicitly requested; section 7 (lines 372-401) provides exact template. Confirmed from intake #1 | S:95 R:90 A:95 D:95 |
| 2 | Certain | Skip comments and empty lines from scaffold file | Consistent with .gitignore handling; comments are not functional direnv lines. Confirmed from intake #2 | S:85 R:95 A:90 D:90 |
| 3 | Confident | Migrate existing symlinks by resolving content to a real file | Prevents data loss during transition; scaffold content preserved. Confirmed from intake #3 | S:75 R:80 A:85 D:85 |
| 4 | Confident | Users may need to re-run `direnv allow` after migration | direnv tracks file identity; switching from symlink to real file may invalidate the allow. Confirmed from intake #4; documented in Deprecated Requirements | S:70 R:90 A:80 D:85 |

4 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).
