# Spec: Add fab/specs/index.md to context loading in apply, review, and archive

**Change**: 260210-7wxx-add-specs-index-context-loading
**Created**: 2026-02-10
**Affected docs**: `fab/docs/fab-workflow/execution-skills.md`

## Context Loading: Specs Index

### Requirement: fab-apply MUST load fab/specs/index.md

The `/fab-apply` skill's Context Loading section MUST include `fab/specs/index.md` as a numbered item, described as "specifications landscape (pre-implementation design intent, human-curated)". This aligns `/fab-apply` with the always-load protocol defined in `_context.md`.

#### Scenario: fab-apply context loading includes specs index
- **GIVEN** an agent reads the `/fab-apply` skill definition
- **WHEN** it processes the Context Loading section
- **THEN** `fab/specs/index.md` SHALL appear as a numbered list item
- **AND** the description SHALL read "specifications landscape (pre-implementation design intent, human-curated)"

### Requirement: fab-review MUST load fab/specs/index.md

The `/fab-review` skill's Context Loading section MUST include `fab/specs/index.md` as a numbered item, described as "specifications landscape (pre-implementation design intent, human-curated)". This aligns `/fab-review` with the always-load protocol defined in `_context.md`.

#### Scenario: fab-review context loading includes specs index
- **GIVEN** an agent reads the `/fab-review` skill definition
- **WHEN** it processes the Context Loading section
- **THEN** `fab/specs/index.md` SHALL appear as a numbered list item
- **AND** the description SHALL read "specifications landscape (pre-implementation design intent, human-curated)"

### Requirement: fab-archive MUST load fab/specs/index.md

The `/fab-archive` skill's Context Loading section MUST include `fab/specs/index.md` as a numbered item, described as "specifications landscape (pre-implementation design intent, human-curated)". This aligns `/fab-archive` with the always-load protocol defined in `_context.md`.

#### Scenario: fab-archive context loading includes specs index
- **GIVEN** an agent reads the `/fab-archive` skill definition
- **WHEN** it processes the Context Loading section
- **THEN** `fab/specs/index.md` SHALL appear as a numbered list item
- **AND** the description SHALL read "specifications landscape (pre-implementation design intent, human-curated)"

## Deprecated Requirements

None.
