# Spec: Extract scaffold content into fab/.kit/scaffold/

**Change**: 260213-3njv-scaffold-dir
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/kit-architecture.md`, `fab/docs/fab-workflow/init.md`, `fab/docs/fab-workflow/distribution.md`

## Non-Goals

- Changing the `.envrc` symlink mechanism itself (stays as a symlink, just new target path)
- Adding new scaffold content beyond what currently exists (new content is a separate change)
- Modifying `config.yaml` or `constitution.md` generation (those live in `/fab-init`, not in scaffold)

## fab-workflow: Scaffold Directory

### Requirement: Scaffold Directory Structure

`fab/.kit/` SHALL contain a `scaffold/` subdirectory with the following files:

| File | Content source |
|------|---------------|
| `scaffold/envrc` | Moved from `fab/.kit/envrc` |
| `scaffold/gitignore-entries` | One `.gitignore` entry per line (initially `fab/current`) |
| `scaffold/docs-index.md` | Initial `fab/docs/index.md` content (extracted from heredoc) |
| `scaffold/design-index.md` | Initial `fab/design/index.md` content (extracted from heredoc) |

Each scaffold file SHALL contain the exact content currently hardcoded in `_fab-scaffold.sh` (heredocs) or stored at `fab/.kit/envrc`. No content changes — this is a pure extraction refactor.

#### Scenario: Fresh kit installation includes scaffold directory
- **GIVEN** a user downloads `kit.tar.gz` or copies `.kit/` via `cp -r`
- **WHEN** they inspect the `.kit/` directory contents
- **THEN** `scaffold/` exists containing `envrc`, `gitignore-entries`, `docs-index.md`, and `design-index.md`

#### Scenario: Scaffold files match current hardcoded content
- **GIVEN** the scaffold files have been extracted from `_fab-scaffold.sh`
- **WHEN** comparing scaffold file content against the current heredoc/file content
- **THEN** they are identical (no content drift)

### Requirement: gitignore-entries Format

`scaffold/gitignore-entries` SHALL use a one-entry-per-line format. Each non-empty, non-comment line represents a single `.gitignore` pattern to ensure is present.

Lines beginning with `#` SHALL be treated as comments and ignored. Empty lines SHALL be ignored.

#### Scenario: Single entry file
- **GIVEN** `scaffold/gitignore-entries` contains one line: `fab/current`
- **WHEN** `_fab-scaffold.sh` processes the file
- **THEN** only `fab/current` is checked/appended to `.gitignore`

#### Scenario: Multiple entries
- **GIVEN** `scaffold/gitignore-entries` contains multiple lines (e.g., `fab/current` and a future entry)
- **WHEN** `_fab-scaffold.sh` processes the file
- **THEN** each entry is independently checked and appended if missing

#### Scenario: Comment lines ignored
- **GIVEN** `scaffold/gitignore-entries` contains `# This is a comment` followed by `fab/current`
- **WHEN** `_fab-scaffold.sh` processes the file
- **THEN** only `fab/current` is processed; the comment is skipped

## fab-workflow: Script Updates

### Requirement: _fab-scaffold.sh reads from scaffold files

`_fab-scaffold.sh` SHALL read content from `fab/.kit/scaffold/` files instead of using hardcoded heredocs or referencing `fab/.kit/envrc` directly.

The following sections of the script SHALL change:

| Section | Current behavior | New behavior |
|---------|-----------------|-------------|
| `.envrc` symlink (section 2) | Target: `fab/.kit/envrc` | Target: `fab/.kit/scaffold/envrc` |
| `docs/index.md` (section 3) | `cat` heredoc to create file | `cp` from `scaffold/docs-index.md` if target missing |
| `design/index.md` (section 4) | `cat` heredoc to create file | `cp` from `scaffold/design-index.md` if target missing |
| `.gitignore` (section 7) | Hardcoded `grep -qx 'fab/current'` | Loop over lines in `scaffold/gitignore-entries` |

#### Scenario: .envrc symlink points to scaffold/envrc
- **GIVEN** a project without an `.envrc` file
- **WHEN** `_fab-scaffold.sh` runs
- **THEN** `.envrc` is created as a symlink to `fab/.kit/scaffold/envrc`
- **AND** the output says `.envrc: created symlink → fab/.kit/scaffold/envrc`

#### Scenario: Existing .envrc symlink repaired to new target
- **GIVEN** `.envrc` is a broken symlink (e.g., pointing to old `fab/.kit/envrc`)
- **WHEN** `_fab-scaffold.sh` runs
- **THEN** the symlink is replaced with one pointing to `fab/.kit/scaffold/envrc`

#### Scenario: docs/index.md created from scaffold file
- **GIVEN** `fab/docs/index.md` does not exist
- **WHEN** `_fab-scaffold.sh` runs
- **THEN** `fab/docs/index.md` is created with content identical to `scaffold/docs-index.md`

#### Scenario: Existing docs/index.md not overwritten
- **GIVEN** `fab/docs/index.md` already exists with project-specific content
- **WHEN** `_fab-scaffold.sh` runs
- **THEN** the file is not modified (idempotency preserved)

#### Scenario: design/index.md created from scaffold file
- **GIVEN** `fab/design/index.md` does not exist
- **WHEN** `_fab-scaffold.sh` runs
- **THEN** `fab/design/index.md` is created with content identical to `scaffold/design-index.md`

#### Scenario: .gitignore entries loop
- **GIVEN** `.gitignore` exists but is missing one of the entries in `scaffold/gitignore-entries`
- **WHEN** `_fab-scaffold.sh` runs
- **THEN** only the missing entry is appended
- **AND** existing entries are not duplicated

#### Scenario: .gitignore does not exist
- **GIVEN** no `.gitignore` file in the repo root
- **WHEN** `_fab-scaffold.sh` runs
- **THEN** `.gitignore` is created containing all entries from `scaffold/gitignore-entries` (one per line)

### Requirement: fab/.kit/envrc removed

The file `fab/.kit/envrc` SHALL be removed. It is replaced by `fab/.kit/scaffold/envrc` with identical content.

#### Scenario: envrc file location after change
- **GIVEN** the change is applied
- **WHEN** inspecting `fab/.kit/`
- **THEN** `fab/.kit/envrc` does not exist
- **AND** `fab/.kit/scaffold/envrc` exists with the same content

## fab-workflow: Documentation Updates

### Requirement: Kit architecture doc updated

The kit-architecture centralized doc SHALL be updated to include `scaffold/` in the `.kit/` directory structure listing.
<!-- assumed: Add scaffold/ as a sibling of templates/, scripts/, etc. in the directory tree — follows the existing listing pattern -->

#### Scenario: Directory listing includes scaffold
- **GIVEN** the updated `fab/docs/fab-workflow/kit-architecture.md`
- **WHEN** reading the Directory Structure section
- **THEN** `scaffold/` appears in the tree with its four files listed

### Requirement: Init doc updated

The init centralized doc SHALL update references to scaffold content sources. Specifically, the Delegation Pattern table row for `.envrc` symlink SHALL reference `fab/.kit/scaffold/envrc` instead of `fab/.kit/envrc`.
<!-- assumed: Only the .envrc row needs updating in the delegation table — other rows reference script behavior, not file paths -->

#### Scenario: Delegation table reflects new path
- **GIVEN** the updated `fab/docs/fab-workflow/init.md`
- **WHEN** reading the Delegation Pattern table
- **THEN** the `.envrc symlink` row says "Links to `fab/.kit/scaffold/envrc`"
- **AND** the `.gitignore` row says "Appends entries from `scaffold/gitignore-entries`"
- **AND** the skeleton files rows mention scaffold source files

### Requirement: Distribution doc updated

The distribution centralized doc SHALL update the bootstrap scenario description to mention that `_fab-scaffold.sh` reads scaffold files for index templates and gitignore entries.

#### Scenario: Bootstrap description mentions scaffold
- **GIVEN** the updated `fab/docs/fab-workflow/distribution.md`
- **WHEN** reading the Bootstrap section
- **THEN** it mentions that `_fab-scaffold.sh` reads from `scaffold/` files for index creation and gitignore entries

## Design Decisions

### Content extraction over template engine
**Decision**: Extract exact current content into plain files (no variable substitution, no template syntax).
**Why**: Scaffold files are direct copies of what the script currently produces. Users edit them as plain markdown/text. No template engine complexity needed — `cp` and line-by-line reads are sufficient.
**Rejected**: Mustache/envsubst templates — adds a dependency and complexity for zero benefit since the content doesn't vary per-project.

### Comment support in gitignore-entries
**Decision**: Support `#` comment lines and empty lines in `gitignore-entries`.
**Why**: Makes the file self-documenting. Users can annotate why entries exist. Zero-cost in bash (one `grep -v` or conditional).
**Rejected**: Raw lines only — functional but less maintainable as the list grows.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | No migration script for existing `.envrc` symlinks — repaired on next `_fab-scaffold.sh` run | Script already handles broken symlinks (replaces them); old `fab/.kit/envrc` disappearing just makes the symlink broken, which triggers repair |
| 2 | Confident | Init doc delegation table: update `.envrc`, `.gitignore`, and skeleton file rows | These are the rows that reference content sources; other rows reference script behavior not file paths |
| 3 | Confident | Comment and blank line support in `gitignore-entries` | Standard convention for line-list config files; trivial to implement in bash |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.
