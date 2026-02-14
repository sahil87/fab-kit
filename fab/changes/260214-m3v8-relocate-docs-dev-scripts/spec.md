# Spec: Relocate memory and specs to docs/

**Change**: 260214-m3v8-relocate-docs-dev-scripts
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/kit-architecture.md`, `fab/memory/fab-workflow/init.md`, `fab/memory/fab-workflow/context-loading.md`, `fab/memory/fab-workflow/specs-index.md`, `fab/memory/fab-workflow/hydrate.md`, `fab/memory/fab-workflow/hydrate-specs.md`

## Non-Goals

- Updating archived changes (68 folders are frozen records)
- Reorganizing `src/` (tracked separately in 260214-q7f2-reorganize-src)
- Changing memory or specs file format — only paths change

## Directory Structure

### Requirement: docs/ as canonical documentation root

The project SHALL use `docs/` as the top-level directory for reference documentation. `docs/memory/` SHALL replace `fab/memory/` and `docs/specs/` SHALL replace `fab/specs/` as their canonical locations.

#### Scenario: New project bootstrap
- **GIVEN** a new project running `_init_scaffold.sh`
- **WHEN** the scaffold creates documentation directories
- **THEN** `docs/memory/` and `docs/specs/` SHALL be created (not `fab/memory/` or `fab/specs/`)
- **AND** `docs/memory/index.md` and `docs/specs/index.md` SHALL be populated from scaffold templates

#### Scenario: Existing project after migration
- **GIVEN** an existing project with `fab/memory/` and `fab/specs/`
- **WHEN** the user runs the migration
- **THEN** `fab/memory/` SHALL be moved to `docs/memory/`
- **AND** `fab/specs/` SHALL be moved to `docs/specs/`
- **AND** all content and relative links within the moved directories SHALL be preserved

### Requirement: Cross-links between memory and specs preserved

Relative links between memory and specs files (e.g., `../specs/index.md` from memory, `../memory/index.md` from specs) SHALL remain valid after the move, since both directories relocate together under `docs/`.

#### Scenario: Memory-to-specs cross-reference
- **GIVEN** `docs/memory/index.md` contains a relative link `../specs/index.md`
- **WHEN** a reader follows the link
- **THEN** the link SHALL resolve to `docs/specs/index.md`

## Context Loading

### Requirement: Always-load layer uses docs/ paths

The four always-load files in `_context.md` SHALL reference the new locations:

1. `fab/config.yaml` (unchanged)
2. `fab/constitution.md` (unchanged)
3. `docs/memory/index.md` (was `fab/memory/index.md`)
4. `docs/specs/index.md` (was `fab/specs/index.md`)

#### Scenario: Skill loads baseline context
- **GIVEN** any skill that follows the always-load convention
- **WHEN** it reads the four baseline files
- **THEN** it SHALL read `docs/memory/index.md` and `docs/specs/index.md`

### Requirement: Selective domain loading uses docs/ paths

When skills load memory files for an active change, they SHALL read from `docs/memory/{domain}/` instead of `fab/memory/{domain}/`.

#### Scenario: Spec generation loads affected memory
- **GIVEN** a brief with Affected Memory referencing `docs/memory/fab-workflow/kit-architecture.md`
- **WHEN** `/fab-continue` generates a spec
- **THEN** it SHALL read the memory file from `docs/memory/fab-workflow/kit-architecture.md`

## Init and Scaffold

### Requirement: _init_scaffold.sh creates docs/ structure

`_init_scaffold.sh` SHALL create `docs/memory/` and `docs/specs/` directories with their index files. It SHALL NOT create `fab/memory/` or `fab/specs/`.

#### Scenario: Fresh scaffold run
- **GIVEN** a project with `fab/.kit/` but no `docs/` directory
- **WHEN** `_init_scaffold.sh` runs
- **THEN** `docs/memory/index.md` SHALL be created from `scaffold/memory-index.md`
- **AND** `docs/specs/index.md` SHALL be created from `scaffold/specs-index.md`
- **AND** `fab/memory/` and `fab/specs/` SHALL NOT be created

### Requirement: Scaffold template content references docs/ paths

The scaffold files (`scaffold/memory-index.md`, `scaffold/specs-index.md`) SHALL reference `docs/` paths in their boilerplate text and cross-links.

#### Scenario: Scaffold memory-index.md content
- **GIVEN** a fresh project
- **WHEN** `docs/memory/index.md` is created from the scaffold template
- **THEN** cross-references to specs SHALL use `../specs/index.md` (sibling directories under `docs/`)

## Skills and Templates

### Requirement: All skill files use docs/ paths

Every skill file in `fab/.kit/skills/` that references `fab/memory/` or `fab/specs/` SHALL be updated to use `docs/memory/` and `docs/specs/` respectively.

#### Scenario: Skill references memory path
- **GIVEN** a skill file (e.g., `_context.md`, `fab-new.md`, `fab-continue.md`)
- **WHEN** it references the memory or specs directory
- **THEN** it SHALL use `docs/memory/` or `docs/specs/` paths

### Requirement: Artifact templates use docs/ paths

The templates in `fab/.kit/templates/` (brief.md, spec.md) that reference `fab/memory/` paths in their Affected Memory metadata SHALL use `docs/memory/` instead.

#### Scenario: Spec template affected memory path
- **GIVEN** the spec template `fab/.kit/templates/spec.md`
- **WHEN** it specifies the affected memory path pattern
- **THEN** it SHALL show `docs/memory/{domain}/{file-name}.md`

## Scripts

### Requirement: Shell scripts use docs/ paths

All shell scripts in `fab/.kit/scripts/` that reference `fab/memory/` or `fab/specs/` SHALL be updated to `docs/memory/` and `docs/specs/`.

#### Scenario: fab-help.sh path references
- **GIVEN** `fab-help.sh` displays directory structure or path information
- **WHEN** it references memory or specs locations
- **THEN** it SHALL use `docs/memory/` and `docs/specs/`

## Constitution

### Requirement: Constitution principles reference docs/ paths

Constitution Principle II ("Docs Are Source of Truth") SHALL reference `docs/memory/` instead of `fab/memory/`. Principle VI ("Specs Are Pre-Implementation Design Intent") SHALL reference `docs/specs/` instead of `fab/specs/`.

#### Scenario: Constitution Principle II
- **GIVEN** `fab/constitution.md` Principle II
- **WHEN** it describes the authoritative source for system behavior
- **THEN** it SHALL reference `docs/memory/` as the canonical location

#### Scenario: Constitution Principle VI
- **GIVEN** `fab/constitution.md` Principle VI
- **WHEN** it describes pre-implementation specifications
- **THEN** it SHALL reference `docs/specs/` as the canonical location

## Config

### Requirement: Preserved files list updated

The conceptual "Preserved" list (files outside `.kit/` that survive upgrades) SHALL list `docs/memory/` and `docs/specs/` instead of `memory/` and `specs/` under `fab/`.

#### Scenario: Kit upgrade preserves docs/
- **GIVEN** a user runs `fab-upgrade.sh`
- **WHEN** `.kit/` is atomically replaced
- **THEN** `docs/memory/` and `docs/specs/` SHALL be untouched (they live outside `fab/`)

## Migration

### Requirement: Migration entry for directory relocation

A migration file SHALL be added to `fab/.kit/migrations/` targeting the appropriate version range. The migration SHALL include instructions and a shell snippet to move `fab/memory/` → `docs/memory/` and `fab/specs/` → `docs/specs/`.

#### Scenario: User runs /fab-update after upgrading
- **GIVEN** an existing project with `fab/memory/` and `fab/specs/`
- **WHEN** the user runs `/fab-update` after a kit upgrade
- **THEN** the migration SHALL instruct moving `fab/memory/` to `docs/memory/`
- **AND** the migration SHALL instruct moving `fab/specs/` to `docs/specs/`
- **AND** the migration SHALL note that `fab/constitution.md` path references need manual review

## Design Decisions

1. **`docs/` over `doc/`**: Chose `docs/` as the top-level directory name.
   - *Why*: `docs/` is the GitHub Pages convention and the most common documentation directory name across open-source projects. More discoverable.
   - *Rejected*: `doc/` — less conventional, though shorter.

2. **Move both memory and specs together**: Both directories relocate to `docs/` in a single change.
   - *Why*: Moving them together preserves all relative cross-links (`../specs/`, `../memory/`) without requiring link rewrites within the directories. Splitting into two changes would require temporary broken links or two rounds of link updates.
   - *Rejected*: Sequential moves — would break cross-links during the interim.

3. **Frozen archived changes**: The 68 archived change folders are not updated.
   - *Why*: Archived changes are historical records. Their path references pointed to the correct locations at time of completion. Updating them provides no value (they're never re-executed) and risks corrupting frozen artifacts.
   - *Rejected*: Bulk-updating archives — high effort, no benefit, violates "frozen record" principle.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Migration includes shell snippet for `mv` commands | Users need a concrete command to relocate directories; pure prose instructions are error-prone for file moves |
| 2 | Confident | Scaffold cross-link from memory index to specs index uses `../specs/index.md` | Both directories are siblings under `docs/`, so relative path is `../specs/index.md` — same pattern as current `fab/memory/` → `fab/specs/` |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
