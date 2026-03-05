# Spec: Rename design/ → specs/ and docs/ → memory/

**Change**: 260213-1u9c-rename-specs-memory
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/*.md` (all hydrated docs reference these paths)

## Non-Goals

- Updating archived change artifacts (`fab/changes/archive/`) — they capture historical state
- Changing domain names or doc structure within the renamed folders (e.g., `fab-workflow/` stays as-is)
- Renaming the `fab/docs/` and `fab/design/` concepts in archived specs/briefs

## Folder Rename: Directory Structure

### Requirement: Rename design/ to specs/

The directory `fab/design/` SHALL be renamed to `fab/specs/`. All contents (files and subdirectories) SHALL be preserved exactly as they are, with only internal cross-references updated.

#### Scenario: Design folder is renamed
- **GIVEN** the directory `fab/design/` exists with its current contents
- **WHEN** the rename is applied
- **THEN** the directory exists at `fab/specs/` with identical contents
- **AND** `fab/design/` no longer exists

### Requirement: Rename docs/ to memory/

The directory `fab/docs/` SHALL be renamed to `fab/memory/`. All contents (files, domain subdirectories, and index files) SHALL be preserved exactly as they are, with only internal cross-references updated.

#### Scenario: Docs folder is renamed
- **GIVEN** the directory `fab/docs/` exists with its current contents
- **WHEN** the rename is applied
- **THEN** the directory exists at `fab/memory/` with identical contents
- **AND** `fab/docs/` no longer exists

## Cross-Reference Updates: Kit Skills

### Requirement: All skill files SHALL reference the new paths

Every file in `fab/.kit/skills/` that references `fab/docs/` SHALL be updated to `fab/memory/`. Every file that references `fab/design/` SHALL be updated to `fab/specs/`.

#### Scenario: Skill file references docs/
- **GIVEN** a skill file contains the string `fab/docs/`
- **WHEN** the rename is applied
- **THEN** the string is replaced with `fab/memory/`

#### Scenario: Skill file references design/
- **GIVEN** a skill file contains the string `fab/design/`
- **WHEN** the rename is applied
- **THEN** the string is replaced with `fab/specs/`

## Cross-Reference Updates: Kit Scripts

### Requirement: All shell scripts SHALL reference the new paths

Every file in `fab/.kit/scripts/` that references `fab/docs/` or `fab/design/` SHALL be updated to the new names.

#### Scenario: Script references docs/
- **GIVEN** a shell script contains `fab/docs/`
- **WHEN** the rename is applied
- **THEN** the string is replaced with `fab/memory/`

## Cross-Reference Updates: Templates

### Requirement: All templates SHALL reference the new paths

Every file in `fab/.kit/templates/` that references `fab/docs/` SHALL be updated to `fab/memory/`. Every file that references `fab/design/` SHALL be updated to `fab/specs/`.

#### Scenario: Template references docs path
- **GIVEN** a template contains `fab/docs/{domain}/{doc-name}.md`
- **WHEN** the rename is applied
- **THEN** it reads `fab/memory/{domain}/{doc-name}.md`

## Cross-Reference Updates: Configuration and Constitution

### Requirement: config.yaml SHALL reference the new paths

Any references to `fab/docs/` or `fab/design/` in `fab/config.yaml` SHALL be updated.

#### Scenario: Config references old paths
- **GIVEN** `fab/config.yaml` contains references to `fab/docs/` or `fab/design/`
- **WHEN** the rename is applied
- **THEN** references point to `fab/memory/` and `fab/specs/` respectively

### Requirement: constitution.md SHALL reference the new paths

All references to `fab/docs/` in `fab/constitution.md` SHALL be updated to `fab/memory/`. All references to `fab/design/` SHALL be updated to `fab/specs/`.

#### Scenario: Constitution references docs as source of truth
- **GIVEN** `fab/constitution.md` states "Centralized documentation in `fab/docs/`"
- **WHEN** the rename is applied
- **THEN** it reads "Centralized documentation in `fab/memory/`"

## Cross-Reference Updates: Internal References Within Renamed Folders

### Requirement: Files within renamed folders SHALL update their cross-references

Files in `fab/specs/` (formerly `fab/design/`) that reference `fab/docs/` SHALL be updated to `fab/memory/`. Files in `fab/memory/` (formerly `fab/docs/`) that reference `fab/design/` SHALL be updated to `fab/specs/`.

#### Scenario: Design index references docs index
- **GIVEN** `fab/specs/index.md` (formerly `fab/design/index.md`) contains a link to `../docs/index.md`
- **WHEN** the rename is applied
- **THEN** the link points to `../memory/index.md`

#### Scenario: Docs index references design index
- **GIVEN** `fab/memory/index.md` (formerly `fab/docs/index.md`) contains a link to `../design/index.md`
- **WHEN** the rename is applied
- **THEN** the link points to `../specs/index.md`

#### Scenario: Hydrated docs reference fab/docs/ paths internally
- **GIVEN** a file in `fab/memory/fab-workflow/` references `fab/docs/` or `fab/design/`
- **WHEN** the rename is applied
- **THEN** references point to `fab/memory/` and `fab/specs/` respectively

## Cross-Reference Updates: Prose and Headers

### Requirement: Index headers and prose SHOULD use new terminology

Where `fab/docs/index.md` and `fab/design/index.md` use the terms "docs" and "design" to describe their own folder, the prose SHOULD be updated to "memory" and "specs" respectively to match the new names.
<!-- assumed: Prose in index files updated to match new folder names — natural consequence of the rename, brief doesn't explicitly specify prose changes -->

#### Scenario: Docs index header updated
- **GIVEN** `fab/memory/index.md` header reads "Documentation Index"
- **WHEN** the rename is applied
- **THEN** the header SHOULD read "Memory Index" or equivalent reflecting the new name

#### Scenario: Design index header updated
- **GIVEN** `fab/specs/index.md` header reads "Design Index"
- **WHEN** the rename is applied
- **THEN** the header SHOULD read "Specs Index" or equivalent reflecting the new name

## Cross-Reference Updates: Scaffold and README

### Requirement: Scaffold script and README SHALL reference new paths

The scaffold script (`fab/.kit/scripts/_fab-scaffold.sh`) and any `README.md` files that reference `fab/docs/` or `fab/design/` SHALL be updated.

#### Scenario: Scaffold creates memory/ instead of docs/
- **GIVEN** the scaffold script creates `fab/docs/` during init
- **WHEN** the rename is applied
- **THEN** it creates `fab/memory/` instead

#### Scenario: Scaffold creates specs/ instead of design/
- **GIVEN** the scaffold script creates `fab/design/` during init
- **WHEN** the rename is applied
- **THEN** it creates `fab/specs/` instead

## Exclusions: Archived Changes

### Requirement: Archived change artifacts SHALL NOT be modified

Files in `fab/changes/archive/` SHALL NOT be modified. They capture historical state at the time each change was completed.

#### Scenario: Archived brief references old paths
- **GIVEN** an archived brief in `fab/changes/archive/` references `fab/docs/`
- **WHEN** the rename is applied
- **THEN** the archived brief is unchanged

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Update index prose/headers to match new folder names ("Memory Index", "Specs Index") | Natural consequence of rename — using old terminology in headers of renamed folders would be confusing. Brief doesn't explicitly specify but intent is clear. |

1 assumption made (1 confident, 0 tentative). Run /fab-clarify to review.
