# Spec: Add Conventions Section to config.yaml

**Change**: 260213-r3m7-add-conventions-section
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/configuration.md`

## Non-Goals

- Updating `fab/design/architecture.md` — Constitution VI prohibits auto-generating or overwriting design docs. The design spec is human-curated; update it manually if desired.
- Adding validation rules to `fab-init-validate` — the new section is entirely optional with no required fields. Validation can be added later if conventions gain required keys.
- Migrating existing `naming` or `git` fields into `conventions` — those sections serve different purposes (folder naming and git integration toggles) and remain unchanged.

## Configuration: Conventions Section

### Requirement: Conventions Section Schema

`config.yaml` SHALL support an optional top-level `conventions:` section. The section SHALL contain string key-value pairs representing project-wide workflow conventions that skills can reference programmatically.

The initial keys SHALL be:
- `branch_naming` — pattern or description of branch naming convention (string)
- `pr_title` — PR title format pattern (string)
- `backlog` — URL or location of the project backlog (string)

All keys within `conventions` SHALL be optional. The `conventions` section itself SHALL be optional — omitting it entirely is valid.
<!-- assumed: All conventions keys are optional strings — consistent with existing config patterns where sections are additive, not required -->

#### Scenario: Config with conventions section

- **GIVEN** a `config.yaml` with a `conventions:` section containing `branch_naming`, `pr_title`, and `backlog`
- **WHEN** any skill loads `config.yaml` during context loading
- **THEN** the conventions values SHALL be available as part of the loaded config context
- **AND** skills MAY reference conventions values when generating artifacts (e.g., branch names, PR titles)

#### Scenario: Config without conventions section

- **GIVEN** a `config.yaml` that does not contain a `conventions:` section
- **WHEN** any skill loads `config.yaml`
- **THEN** the skill SHALL proceed normally without error
- **AND** no conventions-dependent behavior SHALL be triggered

#### Scenario: Partial conventions section

- **GIVEN** a `config.yaml` with a `conventions:` section containing only `backlog` (other keys omitted)
- **WHEN** a skill loads `config.yaml`
- **THEN** the present key (`backlog`) SHALL be available
- **AND** absent keys SHALL be treated as unset (no defaults injected)

### Requirement: Template Documentation

The `config.yaml` template comments SHALL document the `conventions` section with:
- A section header comment explaining the section's purpose
- Inline comments for each key describing its use
- Example values showing the expected format

The section SHALL be placed after `source_paths:` and before `stages:` in the template, grouping all project-context sections (project, context, naming, git, source_paths, conventions) before pipeline-definition sections (stages, checklist, rules).

#### Scenario: New project initialization

- **GIVEN** a new project running `/fab-init`
- **WHEN** `config.yaml` is generated from the template
- **THEN** the `conventions` section SHALL appear with commented-out example keys
- **AND** inline comments SHALL explain each key's purpose

### Requirement: Centralized Doc Update

The `fab/docs/fab-workflow/configuration.md` centralized doc SHALL be updated to document the `conventions` section under the `config.yaml` Schema requirements. The documentation SHALL include:
- Section description and purpose
- Key definitions (name, type, description for each)
- Relationship to existing `naming` and `git` sections (complementary, not overlapping)

#### Scenario: Configuration doc reflects conventions

- **GIVEN** the change is archived
- **WHEN** a user reads `fab/docs/fab-workflow/configuration.md`
- **THEN** the `conventions` section SHALL be documented alongside `project`, `context`, `naming`, `git`, `stages`, `checklist`, and `rules`
- **AND** the section SHALL clearly state that all keys are optional

## Design Decisions

1. **Flat string key-value pairs**: Conventions values are simple strings, not nested objects.
   - *Why*: Consistent with `naming` and `git` section patterns in config.yaml. Simple strings are easy to parse, display, and reference in prompts. Complex structure (e.g., `pr_title.format` + `pr_title.prefix`) can be added later if needed.
   - *Rejected*: Structured/nested convention entries — over-engineers the initial use case.

2. **Section is fully optional with no required keys**: Both the section and every key within it are optional.
   - *Why*: This is additive functionality. Existing projects work fine without it. Forcing conventions on projects that don't need them adds friction for no benefit.
   - *Rejected*: Making conventions required after init — breaks existing projects and adds migration burden.

3. **Complement, don't consolidate**: `conventions` sits alongside `naming` and `git` rather than absorbing them.
   - *Why*: `naming` controls folder name format (agent-consumed, structural). `git` controls git integration toggles (boolean flags). `conventions` captures human/workflow conventions (informational strings). Different purposes, different stability profiles.
   - *Rejected*: Merging all into one section — conflates structural config with informational conventions.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | All conventions keys are optional strings | Consistent with existing config patterns; brief lists these as informational values, not structural requirements |
| 2 | Confident | Place section after source_paths, before stages | Groups project-context sections together; YAML key ordering is non-functional so easily changed |
| 3 | Confident | Exclude fab/design/architecture.md from automated changes | Constitution VI explicitly prohibits auto-generating or overwriting design docs |
| 4 | Confident | No fab-init-validate changes needed | Section is fully optional; validation of optional empty sections adds no value |

4 assumptions made (4 confident, 0 tentative). Run /fab-clarify to review.
