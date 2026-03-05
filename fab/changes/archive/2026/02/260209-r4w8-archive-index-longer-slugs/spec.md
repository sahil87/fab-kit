# Spec: Add archive index and allow longer folder slugs

**Change**: 260209-r4w8-archive-index-longer-slugs
**Created**: 2026-02-09
**Affected docs**: `fab/docs/fab-workflow/execution-skills.md`, `fab/docs/fab-workflow/change-lifecycle.md`, `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/configuration.md`

## Archive: Index Maintenance

### Requirement: Archive index file

`/fab-archive` SHALL maintain an `index.md` file at `fab/changes/archive/index.md` that provides a searchable summary of all archived changes.

#### Scenario: First archive with no existing index

- **GIVEN** no `fab/changes/archive/index.md` exists
- **WHEN** `/fab-archive` runs successfully
- **THEN** it SHALL create `fab/changes/archive/index.md` with a `# Archive Index` heading
- **AND** it SHALL backfill entries for all existing archived change folders by reading each folder's `proposal.md` Why section
- **AND** it SHALL append the newly archived change as an entry
- **AND** entries SHALL be ordered most-recent-first (newest at the top)

#### Scenario: Subsequent archive with existing index

- **GIVEN** `fab/changes/archive/index.md` already exists
- **WHEN** `/fab-archive` runs successfully
- **THEN** it SHALL prepend a new entry for the archived change at the top of the list (after the heading)
- **AND** it SHALL NOT modify existing entries

#### Scenario: Index entry format

- **GIVEN** a change is being archived
- **WHEN** its entry is written to `index.md`
- **THEN** the entry SHALL be a bullet list item in the format: `- **{folder-name}** — {description}`
- **AND** the description SHALL be 1-2 sentences extracted from the proposal's Why section
- **AND** the description SHALL capture the motivation/purpose, not just restate the folder name

### Requirement: Backfill reads proposal Why section

When backfilling existing archived changes, the agent SHALL read each `fab/changes/archive/{name}/proposal.md` and extract the first 1-2 sentences from the Why section as the description. If a `proposal.md` does not exist for an archived change, the entry SHALL use the folder name as a fallback description.

#### Scenario: Archived change missing proposal

- **GIVEN** an archived change folder exists without a `proposal.md`
- **WHEN** backfill runs
- **THEN** the entry SHALL use the folder slug (human-readable portion) as the description

## Naming: Slug Length

### Requirement: Expanded slug word count

The slug component of change folder names SHALL allow 2-6 words (expanded from 2-4 words). This applies to all skills that generate folder names: `/fab-new` and `/fab-discuss`.

#### Scenario: Slug generation with longer description

- **GIVEN** a user provides the description "add archive index and update slug length constraints"
- **WHEN** `/fab-new` generates the folder name
- **THEN** the slug MAY contain up to 6 words (e.g., `add-archive-index-update-slug-length`)
- **AND** the slug MUST contain at least 2 words

#### Scenario: Short description still works

- **GIVEN** a user provides the description "fix typo"
- **WHEN** `/fab-new` generates the folder name
- **THEN** the slug SHALL contain 2 words (e.g., `fix-typo`)

## Deprecated Requirements

_None._
