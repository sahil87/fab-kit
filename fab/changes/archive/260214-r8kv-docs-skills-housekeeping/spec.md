# Spec: Docs Skills Housekeeping

**Change**: 260214-r8kv-docs-skills-housekeeping
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/hydrate.md`, `fab/memory/fab-workflow/kit-architecture.md`, `fab/memory/fab-workflow/execution-skills.md`

## Script Removal: fab-status.sh

### Requirement: Remove fab-status.sh

The script `fab/.kit/scripts/fab-status.sh` SHALL be deleted. It is redundant with `fab-preflight.sh` (which sources `stageman.sh`) and the `/fab-status` skill.

#### Scenario: Script deletion

- **GIVEN** `fab/.kit/scripts/fab-status.sh` exists
- **WHEN** the change is applied
- **THEN** the file SHALL be removed from `fab/.kit/scripts/`
- **AND** any references to `fab-status.sh` in active (non-archived) files SHALL be removed or updated

### Requirement: Update references to fab-status.sh

All references to `fab-status.sh` in non-archived files SHALL be updated. Specifically:

- `fab/.kit/skills/fab-status.md` — remove delegation to `fab-status.sh` and note that the skill handles status display directly (or delegates to `fab-preflight.sh` + `stageman.sh`)
- `fab/memory/fab-workflow/kit-architecture.md` — remove `fab-status.sh` from directory listing and Shell Scripts section
- `fab/memory/fab-workflow/execution-skills.md` — remove references to `fab-status.sh`
- `fab/specs/architecture.md`, `fab/specs/overview.md`, `fab/specs/glossary.md` — update references

Archived changes (`fab/changes/archive/`) SHALL NOT be modified — they are historical records.

#### Scenario: Status skill still works after removal

- **GIVEN** `fab-status.sh` has been deleted
- **WHEN** a user invokes `/fab-status`
- **THEN** the skill SHALL still function correctly using `fab-preflight.sh` and `stageman.sh`

## Skill Renames

### Requirement: Rename fab-hydrate-specs to docs-hydrate-specs

The skill file `fab/.kit/skills/fab-hydrate-specs.md` SHALL be renamed to `fab/.kit/skills/docs-hydrate-specs.md`. Internal references within the file (frontmatter `name`, heading) SHALL be updated to `docs-hydrate-specs`.

#### Scenario: Renamed skill file

- **GIVEN** `fab/.kit/skills/fab-hydrate-specs.md` exists
- **WHEN** the rename is applied
- **THEN** the file SHALL exist at `fab/.kit/skills/docs-hydrate-specs.md`
- **AND** the old file SHALL not exist
- **AND** the frontmatter `name` field SHALL read `docs-hydrate-specs`

### Requirement: Rename fab-hydrate to docs-hydrate-memory

The skill file `fab/.kit/skills/fab-hydrate.md` SHALL be renamed to `fab/.kit/skills/docs-hydrate-memory.md`. Internal references within the file SHALL be updated to `docs-hydrate-memory`.

#### Scenario: Renamed skill file

- **GIVEN** `fab/.kit/skills/fab-hydrate.md` exists
- **WHEN** the rename is applied
- **THEN** the file SHALL exist at `fab/.kit/skills/docs-hydrate-memory.md`
- **AND** the old file SHALL not exist

### Requirement: Rename fab-reorg-specs to docs-reorg-specs

The skill file `fab/.kit/skills/fab-reorg-specs.md` SHALL be renamed to `fab/.kit/skills/docs-reorg-specs.md`. Internal references within the file SHALL be updated to `docs-reorg-specs`.

#### Scenario: Renamed skill file

- **GIVEN** `fab/.kit/skills/fab-reorg-specs.md` exists
- **WHEN** the rename is applied
- **THEN** the file SHALL exist at `fab/.kit/skills/docs-reorg-specs.md`
- **AND** the old file SHALL not exist
- **AND** the frontmatter `name` field SHALL read `docs-reorg-specs`

## Symlink Updates

### Requirement: Regenerate symlinks for renamed skills

After renaming skill files, `_fab-scaffold.sh` SHALL be re-run to regenerate symlinks. The old symlink directories SHALL be removed, and new ones created:

| Old symlink path | New symlink path |
|---|---|
| `.claude/skills/fab-hydrate-specs/` | `.claude/skills/docs-hydrate-specs/` |
| `.claude/skills/fab-hydrate/` | `.claude/skills/docs-hydrate-memory/` |
| `.claude/skills/fab-reorg-specs/` | `.claude/skills/docs-reorg-specs/` |
| `.opencode/commands/fab-hydrate-specs.md` | `.opencode/commands/docs-hydrate-specs.md` |
| `.opencode/commands/fab-hydrate.md` | `.opencode/commands/docs-hydrate-memory.md` |
| `.opencode/commands/fab-reorg-specs.md` | `.opencode/commands/docs-reorg-specs.md` |
| `.agents/skills/fab-hydrate-specs/` | `.agents/skills/docs-hydrate-specs/` |
| `.agents/skills/fab-hydrate/` | `.agents/skills/docs-hydrate-memory/` |
| `.agents/skills/fab-reorg-specs/` | `.agents/skills/docs-reorg-specs/` |

#### Scenario: Scaffold regenerates cleanly

- **GIVEN** skill files have been renamed
- **WHEN** `_fab-scaffold.sh` is run
- **THEN** symlinks for the new names SHALL be created
- **AND** the old symlink directories/files SHALL be removed (manually before re-running scaffold, since scaffold only creates — it does not clean stale symlinks)

## New Skill: docs-reorg-memory

### Requirement: Create docs-reorg-memory skill

A new skill `fab/.kit/skills/docs-reorg-memory.md` SHALL be created, mirroring `docs-reorg-specs` but targeting `fab/memory/` instead of `fab/specs/`.

The skill SHALL:
- Scan all memory files across all domains in `fab/memory/`
- Identify themes and organizational patterns (up to 10)
- Diagnose the current structure
- Propose reorganization with a migration map
- Be read-only by default — execute migrations only with explicit user approval

#### Scenario: Memory reorganization analysis

- **GIVEN** `fab/memory/` contains at least one domain with memory files
- **WHEN** a user invokes `/docs-reorg-memory`
- **THEN** the skill SHALL read all memory files, present themes, and propose reorganization
- **AND** no files SHALL be modified without explicit user confirmation

#### Scenario: No memory files

- **GIVEN** `fab/memory/` is empty or contains only `index.md`
- **WHEN** a user invokes `/docs-reorg-memory`
- **THEN** the skill SHALL abort with "Nothing to reorganize."

## Cross-Reference Updates

### Requirement: Update all non-archived references

All references to old skill names in non-archived files SHALL be updated:

| Old name | New name | Files to update |
|----------|----------|-----------------|
| `fab-hydrate-specs` | `docs-hydrate-specs` | README.md, `fab/specs/glossary.md`, `fab/specs/skills.md`, `fab/specs/user-flow.md`, `fab/.kit/scripts/fab-help.sh`, `fab/memory/fab-workflow/hydrate-specs.md`, `fab/memory/fab-workflow/model-tiers.md`, `fab/memory/fab-workflow/index.md` |
| `fab-hydrate` (the skill) | `docs-hydrate-memory` | README.md, `fab/.kit/scripts/fab-help.sh`, `fab/memory/fab-workflow/hydrate.md`, `fab/memory/fab-workflow/init.md`, `fab/memory/fab-workflow/context-loading.md`, `fab/memory/fab-workflow/hydrate-generate.md`, `fab/memory/fab-workflow/kit-architecture.md`, `fab/memory/fab-workflow/model-tiers.md`, `fab/memory/fab-workflow/index.md`, `.claude/agents/fab-init.md` |
| `fab-reorg-specs` | `docs-reorg-specs` | (no external references found beyond the skill file itself) |
| `fab-status.sh` | *(removed)* | `fab/.kit/skills/fab-status.md`, `fab/memory/fab-workflow/kit-architecture.md`, `fab/memory/fab-workflow/execution-skills.md`, `fab/specs/architecture.md`, `fab/specs/overview.md`, `fab/specs/glossary.md`, `fab/.kit/scripts/fab-help.sh`, `fab/memory/fab-workflow/model-tiers.md`, `fab/memory/fab-workflow/change-lifecycle.md` |

When updating `fab-hydrate` references, care MUST be taken to distinguish:
- **Skill references** (`/fab-hydrate`, `fab-hydrate.md`) → rename to `docs-hydrate-memory`
- **Pipeline behavior references** ("hydrate behavior", "hydration", "hydrate stage") → leave unchanged, as these refer to the pipeline stage in `/fab-continue`, not the standalone skill

Archived changes (`fab/changes/archive/`) and the current change folder (`fab/changes/260214-r8kv-docs-skills-housekeeping/`) SHALL NOT be modified.

#### Scenario: Distinguish skill from pipeline references

- **GIVEN** a file contains both `/fab-hydrate` (skill invocation) and "hydrate behavior" (pipeline stage)
- **WHEN** cross-references are updated
- **THEN** only the skill invocation SHALL be renamed to `/docs-hydrate-memory`
- **AND** pipeline stage references SHALL remain unchanged

## Deprecated Requirements

### fab-status.sh Script

**Reason**: Redundant with `fab-preflight.sh` + `stageman.sh` for data retrieval and `/fab-status` skill for user-facing display. The script's functionality is fully covered by these two components.
**Migration**: No migration needed — `/fab-status` skill already works without `fab-status.sh` via preflight.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | `docs-reorg-memory` mirrors `docs-reorg-specs` structure and behavior | User explicitly requested it; naming symmetry makes the pattern obvious |
| 2 | Confident | Archived changes are not modified | Archive is a historical record; modifying it would violate its purpose |
| 3 | Confident | Old symlinks must be manually removed before re-running scaffold | `_fab-scaffold.sh` creates symlinks but does not clean stale ones — this is existing behavior |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.
