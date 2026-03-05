# Spec: Fix command format — /fab:xxx to /fab-xxx

**Change**: 260207-sawf-fix-command-format
**Created**: 2026-02-07
**Affected docs**: `fab-workflow/context-loading.md`, `fab-workflow/planning-skills.md`, `fab-workflow/execution-skills.md`, `fab-workflow/change-lifecycle.md`, `fab-workflow/hydrate.md`, `fab-workflow/hydrate-generate.md`, `fab-workflow/init.md`, `fab-workflow/index.md`, `fab-workflow/configuration.md`, `fab-workflow/templates.md`, `fab-workflow/kit-architecture.md`

## Command References: Colon-to-Hyphen Update

### Requirement: All command references MUST use hyphen format

All references to fab commands across skill files, scripts, templates, and centralized docs SHALL use the `/fab-xxx` hyphen format (e.g., `/fab-continue`) instead of the `/fab:xxx` colon format (e.g., `/fab:continue`). This applies to:
- "Next:" suggestion lines
- Inline references to commands in prose
- Table cells referencing commands
- Code blocks and backtick-quoted command names
- Shell script output strings

#### Scenario: Next line in skill output
- **GIVEN** a skill file contains a "Next:" suggestion line
- **WHEN** the line references a fab command
- **THEN** the command MUST use hyphen format (e.g., `Next: /fab-continue`)

#### Scenario: Inline prose reference
- **GIVEN** a markdown file references a fab command in running text
- **WHEN** the reference uses `/fab:xxx` colon format
- **THEN** it SHALL be updated to `/fab-xxx` hyphen format

#### Scenario: Shell script output
- **GIVEN** a shell script (`fab/.kit/scripts/*.sh`) emits command suggestions
- **WHEN** those suggestions reference fab commands
- **THEN** they SHALL use `/fab-xxx` hyphen format

### Requirement: Archived changes SHALL NOT be modified

Historical artifacts in `fab/changes/archive/` SHALL remain untouched. These are completed records and do not affect runtime behavior.

#### Scenario: Archive directory is excluded
- **GIVEN** the `fab/changes/archive/` directory contains completed change artifacts
- **WHEN** the replacement is applied across the project
- **THEN** files under `fab/changes/archive/` SHALL NOT be modified

### Requirement: Replacement MUST be scoped to command references only

The replacement of `/fab:` with `/fab-` SHALL only affect command references. It MUST NOT alter:
- File paths (none exist with this pattern)
- YAML keys or values that aren't command references
- Any content outside the `fab/.kit/` and `fab/docs/` directories (except `fab/backlog.md` if it contains references)

#### Scenario: Non-command colon usage preserved
- **GIVEN** a file contains a colon in non-command context (e.g., `domain: fab-workflow`)
- **WHEN** the replacement is applied
- **THEN** only `/fab:` patterns are replaced, not arbitrary colons

## Deprecated Requirements

_(none)_
