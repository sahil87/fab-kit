# Spec: Fix broken template links in fab-hydrate

**Change**: 260208-4wg3-fix-hydrate-links
**Created**: 2026-02-08
**Affected docs**: `fab/docs/fab-workflow/hydrate.md`

## Fab-Workflow: Hydrate Skill Template References

### Requirement: Correct Template Link Paths
The `/fab-hydrate` skill file (`fab/.kit/skills/fab-hydrate.md`) SHALL reference template formats using relative paths that resolve to `fab/specs/templates.md`. All three existing links that point to the old `doc/fab-spec/TEMPLATES.md` path MUST be updated to `../../specs/templates.md` with the correct anchor fragments.

#### Scenario: Domain Index Format Link (Line 97)
- **GIVEN** a user or agent reads fab-hydrate.md Step 3, bullet 2
- **WHEN** they follow the "Domain Index format" link
- **THEN** the link resolves to `fab/specs/templates.md#domain-index-fabdocsdomainindexmd`

#### Scenario: Individual Doc Format Link (Line 104)
- **GIVEN** a user or agent reads fab-hydrate.md Step 3, bullet 3
- **WHEN** they follow the "Centralized Doc" link
- **THEN** the link resolves to `fab/specs/templates.md#individual-doc-fabdomainnamemd`

#### Scenario: Top-Level Index Format Link (Line 124)
- **GIVEN** a user or agent reads fab-hydrate.md Step 5, bullet 5
- **WHEN** they follow the "Top-Level Index format" link
- **THEN** the link resolves to `fab/specs/templates.md#top-level-index-fabdocsindexmd`

### Requirement: Preserve Existing Behavior
The skill's operational behavior (ingest mode, generate mode, index maintenance) MUST NOT change. Only the markdown link targets are modified.

#### Scenario: No Behavioral Change
- **GIVEN** the three links are updated to the new paths
- **WHEN** `/fab-hydrate` is invoked with any valid arguments
- **THEN** skill behavior is identical to before the change — only documentation reference links differ

## Deprecated Requirements

(none)
