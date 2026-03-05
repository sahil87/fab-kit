# Spec: Rename /fab-backfill to /fab-hydrate-design

**Change**: 260212-akhp-rename-fab-backfill
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/backfill.md` (renamed to `hydrate-design.md`)

## Non-Goals

- Changing any behavior or logic of the skill — pure rename
- Updating archived change files (historical records preserved as-is)

## Skill Rename: File and Directory Moves

### Requirement: Rename Kit Skill File

The skill file `fab/.kit/skills/fab-backfill.md` SHALL be renamed to `fab/.kit/skills/fab-hydrate-design.md`. All internal references within the file (frontmatter `name`, heading, prose) SHALL be updated to use `/fab-hydrate-design`.

#### Scenario: Kit skill file renamed

- **GIVEN** the skill file exists at `fab/.kit/skills/fab-backfill.md`
- **WHEN** the rename is applied
- **THEN** `fab/.kit/skills/fab-hydrate-design.md` exists with updated internal references
- **AND** `fab/.kit/skills/fab-backfill.md` no longer exists

### Requirement: Rename Claude Skill Directory and Symlink

The Claude skill directory `.claude/skills/fab-backfill/` SHALL be renamed to `.claude/skills/fab-hydrate-design/`. The `SKILL.md` symlink inside SHALL point to the new kit skill path `../../../fab/.kit/skills/fab-hydrate-design.md`.

#### Scenario: Claude skill directory and symlink updated

- **GIVEN** `.claude/skills/fab-backfill/SKILL.md` is a symlink to `../../../fab/.kit/skills/fab-backfill.md`
- **WHEN** the rename is applied
- **THEN** `.claude/skills/fab-hydrate-design/SKILL.md` is a symlink to `../../../fab/.kit/skills/fab-hydrate-design.md`
- **AND** `.claude/skills/fab-backfill/` no longer exists

## Documentation Updates: Centralized Docs

### Requirement: Rename Centralized Doc File

`fab/docs/fab-workflow/backfill.md` SHALL be renamed to `fab/docs/fab-workflow/hydrate-design.md`. All internal references within the file SHALL be updated from `/fab-backfill` to `/fab-hydrate-design`.

#### Scenario: Doc file renamed with updated content

- **GIVEN** `fab/docs/fab-workflow/backfill.md` exists with references to `/fab-backfill`
- **WHEN** the rename is applied
- **THEN** `fab/docs/fab-workflow/hydrate-design.md` exists with all references updated to `/fab-hydrate-design`
- **AND** the `# Backfill` heading is updated to `# Hydrate Design`
- **AND** `fab/docs/fab-workflow/backfill.md` no longer exists

### Requirement: Update Documentation Indexes

`fab/docs/fab-workflow/index.md` and `fab/docs/index.md` SHALL be updated to reference `hydrate-design` instead of `backfill`.

#### Scenario: Domain index updated

- **GIVEN** `fab/docs/fab-workflow/index.md` lists `[backfill](backfill.md)`
- **WHEN** the rename is applied
- **THEN** the entry references `[hydrate-design](hydrate-design.md)` with the updated skill name `/fab-hydrate-design`

#### Scenario: Top-level docs index updated

- **GIVEN** `fab/docs/index.md` lists `backfill` in the fab-workflow domain doc list
- **WHEN** the rename is applied
- **THEN** the entry reads `hydrate-design` instead of `backfill`

## Reference Updates: Cross-Cutting Files

### Requirement: Update All Active References to /fab-backfill

Every occurrence of `/fab-backfill` or `fab-backfill` (as a command name reference) in non-archived files SHALL be updated to `/fab-hydrate-design` or `fab-hydrate-design` respectively.

The following files contain references that MUST be updated:

| File | Reference type |
|------|---------------|
| `fab/docs/fab-workflow/model-tiers.md` | Skill name in tier listing |
| `fab/docs/fab-workflow/design-index.md` | Skill name in description |
| `fab/design/glossary.md` | Term definition |
| `fab/design/skills.md` | Skill entry |
| `fab/design/user-flow.md` | Command in flow diagrams |
| `fab/.kit/scripts/fab-help.sh` | Help text output |
| `README.md` | Command listing |
| `fab/backlog.md` | Backlog item reference |

#### Scenario: All active references updated

- **GIVEN** 8+ non-archived files contain references to `fab-backfill`
- **WHEN** the rename is applied
- **THEN** all references in non-archived files use `fab-hydrate-design`
- **AND** no non-archived file contains `fab-backfill` as a command name (excluding historical context in this change's own artifacts)

### Requirement: Preserve Archived Change References

Files under `fab/changes/archive/` SHALL NOT be modified. These are historical records of past changes and SHOULD retain their original command references.

#### Scenario: Archive files unchanged

- **GIVEN** archived changes reference `/fab-backfill`
- **WHEN** the rename is applied
- **THEN** all files under `fab/changes/archive/` remain unmodified

### Requirement: Update Active Change References

Active changes (non-archived) that reference `/fab-backfill` SHOULD be updated where the reference is a command name rather than historical attribution.

#### Scenario: Active change brief updated

- **GIVEN** `fab/changes/260212-h9k3-fab-init-family/brief.md` references `/fab-backfill`
- **WHEN** the rename is applied
- **THEN** the reference is updated to `/fab-hydrate-design`

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Archived changes are not updated | Archives are historical records of completed work; modifying them would alter the historical record without functional benefit |
| 2 | Confident | Active change references are updated | Active changes may still be consulted; stale command names would cause confusion |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
