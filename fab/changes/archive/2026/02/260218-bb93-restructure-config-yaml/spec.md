# Spec: Restructure config.yaml

**Change**: 260218-bb93-restructure-config-yaml
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/model-tiers.md`, `docs/memory/fab-workflow/context-loading.md`

## Non-Goals

- Changing the content or semantics of `context:`, `code_quality:`, or `stages:` — only their location changes
- Modifying how `capable` tier model resolution works — only `fast` tier reads from the mapping file

## Configuration: Extract `context:` to `fab/context.md`

### Requirement: Context as standalone markdown file

The `context:` section SHALL be removed from `fab/config.yaml` and `fab/.kit/scaffold/config.yaml`. Project context SHALL instead live in `fab/context.md`, a plain markdown companion file alongside `fab/constitution.md`.

#### Scenario: New project bootstrap via /fab-setup

- **GIVEN** a project with no `fab/context.md`
- **WHEN** `/fab-setup` runs the bootstrap flow
- **THEN** `fab/context.md` is created from `fab/.kit/scaffold/context.md`
- **AND** the user is prompted for tech stack, conventions, and architecture context
- **AND** the content is written as free-form markdown (not YAML multi-line strings)

#### Scenario: Existing project with context: in config.yaml

- **GIVEN** a project whose `fab/config.yaml` contains a `context:` section
- **WHEN** the user reads `fab/config.yaml`
- **THEN** no `context:` section is present (it was extracted)
- **AND** a pointer comment in the config.yaml header references `fab/context.md`

### Requirement: Context file is optional

`fab/context.md` SHOULD exist but its absence MUST NOT cause any skill to error. Skills that load context SHALL treat a missing `fab/context.md` as an empty context (no content loaded, no warning).

#### Scenario: Skill runs without context.md

- **GIVEN** a project with no `fab/context.md`
- **WHEN** any skill runs the Always Load layer
- **THEN** the skill proceeds normally without error
- **AND** no warning is emitted about the missing file

### Requirement: Scaffold template

A scaffold template SHALL exist at `fab/.kit/scaffold/context.md` with a markdown structure that guides the user on what to include (tech stack, conventions, architecture, monorepo sections).

#### Scenario: Scaffold template content

- **GIVEN** the file `fab/.kit/scaffold/context.md`
- **WHEN** read by `/fab-setup` during bootstrap
- **THEN** it contains placeholder guidance for tech stack, conventions, and architecture
- **AND** it includes a monorepo example showing labeled sections

## Configuration: Extract `code_quality:` to `fab/code-quality.md`

### Requirement: Code quality as standalone markdown file

The `code_quality:` section (both the commented-out scaffold and any active configuration) SHALL be removed from `fab/config.yaml` and `fab/.kit/scaffold/config.yaml`. Code quality configuration SHALL instead live in `fab/code-quality.md`.

#### Scenario: Config.yaml no longer contains code_quality

- **GIVEN** the updated `fab/config.yaml` or scaffold
- **WHEN** the file is read
- **THEN** no `code_quality:` section exists (including commented-out examples)

### Requirement: Code quality file is optional

`fab/code-quality.md` SHOULD exist for projects that want coding standards but its absence MUST NOT cause any skill to error. When absent, apply and review stages SHALL use their built-in defaults (pattern consistency, no unnecessary duplication).

#### Scenario: Apply stage without code-quality.md

- **GIVEN** a project with no `fab/code-quality.md`
- **WHEN** `/fab-continue` runs the apply stage
- **THEN** implementation proceeds with built-in defaults only
- **AND** no error or warning about missing code-quality.md

#### Scenario: Review stage with code-quality.md present

- **GIVEN** a project with `fab/code-quality.md` containing principles and anti-patterns
- **WHEN** `/fab-continue` runs the review stage
- **THEN** the review sub-agent loads and checks against those principles and anti-patterns

### Requirement: Markdown format for code quality

The `fab/code-quality.md` file SHALL use markdown sections instead of YAML keys. The three former YAML fields map to sections:

- `principles:` → `## Principles` (bulleted list)
- `anti_patterns:` → `## Anti-Patterns` (bulleted list)
- `test_strategy:` → `## Test Strategy` (paragraph or single value)

#### Scenario: Code quality file structure

- **GIVEN** the file `fab/code-quality.md`
- **WHEN** read by any skill
- **THEN** it contains `## Principles`, `## Anti-Patterns`, and `## Test Strategy` sections
- **AND** each section uses natural markdown formatting (bullet lists, paragraphs)

### Requirement: Scaffold template for code quality

A scaffold template SHALL exist at `fab/.kit/scaffold/code-quality.md` with example content that mirrors the current scaffold's commented-out `code_quality:` section.

#### Scenario: Scaffold template provides examples

- **GIVEN** the file `fab/.kit/scaffold/code-quality.md`
- **WHEN** read by `/fab-setup` during bootstrap
- **THEN** it contains example principles, anti-patterns, and test strategy
- **AND** the examples match the current scaffold's commented-out content

## Configuration: Delete `stages:` from config.yaml

### Requirement: Remove stages section

The `stages:` section SHALL be removed from `fab/config.yaml` and `fab/.kit/scaffold/config.yaml`. `fab/.kit/schemas/workflow.yaml` is the sole authoritative source for stage definitions.

#### Scenario: No stages in config.yaml

- **GIVEN** the updated `fab/config.yaml` or scaffold
- **WHEN** the file is read
- **THEN** no `stages:` section exists
- **AND** no comments referencing `stages:` as a consumed section remain

#### Scenario: No consumers of stages from config

- **GIVEN** the codebase after this change
- **WHEN** searching for code that reads `stages:` from config.yaml
- **THEN** no such code exists
- **AND** `fab/.kit/schemas/workflow.yaml` remains the only stages definition

### Requirement: Remove stages from /fab-setup config menu

The `stages` entry SHALL be removed from `/fab-setup`'s valid config sections list and the config update menu.

#### Scenario: Config menu no longer lists stages

- **GIVEN** a user runs `/fab-setup config`
- **WHEN** the section menu is displayed
- **THEN** `stages` is not listed as an option
- **AND** `/fab-setup config stages` returns an invalid section error

## Configuration: Merge `model-tiers.yaml` into config.yaml

### Requirement: model_tiers section in config.yaml

A `model_tiers:` section SHALL be added to `fab/config.yaml` and `fab/.kit/scaffold/config.yaml` containing the tier-to-model mappings currently in `fab/.kit/model-tiers.yaml`.

```yaml
model_tiers:
  fast:
    claude: haiku
  capable:
    claude: null    # null = use platform default
```

#### Scenario: Scaffold includes model_tiers

- **GIVEN** the file `fab/.kit/scaffold/config.yaml`
- **WHEN** read
- **THEN** it contains a `model_tiers:` section with `fast.claude: haiku` and `capable.claude: null`
- **AND** it includes comments explaining the tier system and null semantics

### Requirement: Delete standalone model-tiers.yaml

The file `fab/.kit/model-tiers.yaml` SHALL be deleted. All references to it SHALL be updated to point to `fab/config.yaml` instead.

#### Scenario: model-tiers.yaml no longer exists

- **GIVEN** the `.kit/` directory after this change
- **WHEN** listing its contents
- **THEN** `model-tiers.yaml` does not exist

### Requirement: Update sync-workspace.sh

`fab/.kit/sync/2-sync-workspace.sh` SHALL be updated to:

1. Remove the pre-flight check for `model-tiers.yaml` existence (lines 20-23)
2. Read fast-tier model from `fab/config.yaml` `model_tiers.fast.claude` instead of from `model-tiers.yaml`
3. Remove the override logic (since config.yaml is now the single source)
4. Fall back to hardcoded default (`haiku`) if `config.yaml` has no `model_tiers` section or the project has no `config.yaml` yet

#### Scenario: Sync reads model tiers from config.yaml

- **GIVEN** a project with `model_tiers.fast.claude: sonnet` in config.yaml
- **WHEN** `2-sync-workspace.sh` generates agent files for fast-tier skills
- **THEN** the generated agent files use `model: sonnet`

#### Scenario: Sync falls back when no model_tiers in config

- **GIVEN** a project whose config.yaml has no `model_tiers:` section
- **WHEN** `2-sync-workspace.sh` generates agent files for fast-tier skills
- **THEN** the generated agent files use `model: haiku` (hardcoded default)

#### Scenario: Sync works without config.yaml (new project)

- **GIVEN** a new project with no `fab/config.yaml`
- **WHEN** `2-sync-workspace.sh` runs during bootstrap
- **THEN** it uses the hardcoded default `haiku` for fast-tier models
- **AND** does not error about missing config

### Requirement: Add model_tiers to /fab-setup config menu

The `model_tiers` entry SHALL be added to `/fab-setup`'s valid config sections list and the config update menu.

#### Scenario: Config menu lists model_tiers

- **GIVEN** a user runs `/fab-setup config`
- **WHEN** the section menu is displayed
- **THEN** `model_tiers` is listed as an option
- **AND** `/fab-setup config model_tiers` edits the model tier mappings

## Context Loading: Update always-load list

### Requirement: Add context.md and code-quality.md to Layer 1

`fab/.kit/skills/_context.md` Section 1 ("Always Load") SHALL be updated to include:

1. `fab/context.md` — free-form project context (optional, no error if missing)
2. `fab/code-quality.md` — coding standards for apply/review (optional, no error if missing)

Both are listed after `fab/constitution.md` in the loading order.

#### Scenario: Always Load file list

- **GIVEN** the updated `_context.md`
- **WHEN** reading the Always Load layer description
- **THEN** it lists 6 files: `fab/config.yaml`, `fab/constitution.md`, `fab/context.md`, `fab/code-quality.md`, `docs/memory/index.md`, `docs/specs/index.md`
- **AND** `fab/context.md` and `fab/code-quality.md` are marked as optional

### Requirement: Config.yaml header update

The companion files comment block at the top of `fab/config.yaml` and the scaffold SHALL be updated to reference all companion files:

```yaml
# Companion files (not configured here):
#   fab/constitution.md    — project principles & constraints (MUST/SHOULD rules)
#   fab/context.md         — free-form project context (tech stack, conventions)
#   fab/code-quality.md    — coding standards for apply/review (optional)
#   docs/memory/index.md   — centralized documentation index
```

#### Scenario: Config header references all companions

- **GIVEN** the updated `fab/config.yaml` or scaffold
- **WHEN** reading the header comment block
- **THEN** it lists `constitution.md`, `context.md`, `code-quality.md`, and `docs/memory/index.md`

## Skill References: Update cross-references

### Requirement: Update fab-continue code_quality references

`fab/.kit/skills/fab-continue.md` SHALL update all references to `code_quality` from config.yaml to point to `fab/code-quality.md`:

- Apply stage: pattern extraction reads `fab/code-quality.md` for `principles` and `test_strategy`
- Review stage: code quality check reads `fab/code-quality.md` for `principles` and `anti_patterns`

#### Scenario: Apply stage loads code-quality.md

- **GIVEN** a project with `fab/code-quality.md`
- **WHEN** `/fab-continue` runs the apply stage
- **THEN** it reads `fab/code-quality.md` for coding principles and test strategy
- **AND** does not reference `config.yaml` for `code_quality`

### Requirement: Update _generation.md checklist references

`fab/.kit/skills/_generation.md` Checklist Generation Procedure SHALL update the `code_quality` reference to read from `fab/code-quality.md` instead of `fab/config.yaml`.

#### Scenario: Checklist generation reads code-quality.md

- **GIVEN** a project with `fab/code-quality.md`
- **WHEN** the checklist is generated
- **THEN** Code Quality items are derived from `fab/code-quality.md`
- **AND** the procedure does not reference `config.yaml` for `code_quality`

### Requirement: Update fab-setup valid sections

`fab/.kit/skills/fab-setup.md` SHALL:

1. Remove `stages` from the valid config sections list
2. Remove `code_quality` from the valid config sections list
3. Add `model_tiers` to the valid config sections list
4. Remove `context` from the valid config sections list (context is now a separate file, not a config section)
5. Update the config update menu numbering accordingly

#### Scenario: Updated valid sections list

- **GIVEN** the updated `fab-setup.md`
- **WHEN** reading the valid sections list
- **THEN** it contains: `project`, `source_paths`, `rules`, `checklist`, `git`, `naming`, `model_tiers`
- **AND** `stages`, `code_quality`, and `context` are absent

### Requirement: Update fab-setup bootstrap to create companion files

`fab/.kit/skills/fab-setup.md` bootstrap behavior SHALL add steps to create `fab/context.md` and `fab/code-quality.md` from their scaffold templates (if they don't exist), following the same pattern as `constitution.md`.

#### Scenario: Bootstrap creates context.md

- **GIVEN** a project with no `fab/context.md`
- **WHEN** `/fab-setup` runs the bootstrap flow
- **THEN** `fab/context.md` is created from the scaffold template
- **AND** creation is reported in the bootstrap output

#### Scenario: Bootstrap creates code-quality.md

- **GIVEN** a project with no `fab/code-quality.md`
- **WHEN** `/fab-setup` runs the bootstrap flow
- **THEN** `fab/code-quality.md` is created from the scaffold template
- **AND** creation is reported in the bootstrap output

## Migration: Automated upgrade for existing projects

### Requirement: Migration file for config restructuring

A migration file SHALL be created at `fab/.kit/migrations/{FROM}-to-{TO}.md` that automates the restructuring for existing projects. The migration SHALL handle:

1. Extract `context:` value from `fab/config.yaml` → write to `fab/context.md`
2. Extract `code_quality:` value from `fab/config.yaml` (if present, including commented-out) → write to `fab/code-quality.md` with markdown sections
3. Remove `stages:` section from `fab/config.yaml`
4. Add `model_tiers:` section to `fab/config.yaml` (using defaults from the deleted `model-tiers.yaml`)
5. Remove `context:`, `code_quality:`, and `stages:` sections from `fab/config.yaml`
6. Update the config.yaml header comment to list new companion files

<!-- clarified: migration included in this change per user decision — not deferred to follow-up -->

#### Scenario: Migration extracts context to markdown

- **GIVEN** an existing project with `context: |` in `fab/config.yaml`
- **WHEN** the migration runs
- **THEN** `fab/context.md` is created with the extracted content
- **AND** the `context:` section is removed from `fab/config.yaml`

#### Scenario: Migration extracts active code_quality

- **GIVEN** an existing project with an active `code_quality:` section in `fab/config.yaml`
- **WHEN** the migration runs
- **THEN** `fab/code-quality.md` is created with `## Principles`, `## Anti-Patterns`, and `## Test Strategy` sections
- **AND** the `code_quality:` section is removed from `fab/config.yaml`

#### Scenario: Migration handles commented-out code_quality

- **GIVEN** an existing project with only commented-out `code_quality:` in `fab/config.yaml`
- **WHEN** the migration runs
- **THEN** `fab/code-quality.md` is created from the scaffold template (not from commented YAML)
- **AND** the commented-out `code_quality:` block is removed from `fab/config.yaml`

#### Scenario: Migration adds model_tiers

- **GIVEN** an existing project without `model_tiers:` in `fab/config.yaml`
- **WHEN** the migration runs
- **THEN** a `model_tiers:` section is added with the default mappings (`fast.claude: haiku`, `capable.claude: null`)

#### Scenario: Migration removes stages

- **GIVEN** an existing project with `stages:` in `fab/config.yaml`
- **WHEN** the migration runs
- **THEN** the entire `stages:` section is removed
- **AND** no replacement section is added

#### Scenario: Migration is idempotent

- **GIVEN** a project that has already been migrated
- **WHEN** the migration runs again
- **THEN** no changes are made (all steps detect existing state and skip)

### Requirement: Migration pre-checks

The migration file SHALL include pre-checks that verify:

1. `fab/config.yaml` exists
2. The project version is within the expected range

If pre-checks fail, the migration MUST abort with an actionable error message.

### Requirement: Migration verification

The migration file SHALL include verification steps that confirm:

1. `fab/context.md` exists (or was already present)
2. `fab/config.yaml` no longer contains `context:`, `stages:`, or `code_quality:` sections
3. `fab/config.yaml` contains a `model_tiers:` section

## Deprecated Requirements

### `context:` in config.yaml

**Reason**: Extracted to `fab/context.md` — free-form prose is better authored in markdown than YAML multi-line strings.
**Migration**: Content moves to `fab/context.md`.

### `code_quality:` in config.yaml

**Reason**: Extracted to `fab/code-quality.md` — principles and anti-patterns are prose guidelines, not structured configuration.
**Migration**: Content moves to `fab/code-quality.md` with YAML keys becoming markdown sections.

### `stages:` in config.yaml

**Reason**: Dead code — `fab/.kit/schemas/workflow.yaml` is the sole source of truth for stage definitions. No skill or script reads `stages:` from config.yaml.
**Migration**: N/A — `workflow.yaml` already provides complete stage definitions.

### `fab/.kit/model-tiers.yaml`

**Reason**: Collapsed into `config.yaml` as `model_tiers:` section. The kit-default → project-override dual-file pattern added complexity with minimal benefit.
**Migration**: Content moves to `model_tiers:` in config.yaml.

## Design Decisions

1. **context.md and code-quality.md are optional, constitution.md remains required**
   - *Why*: Constitution defines immutable principles that all skills depend on. Context and code quality are supplementary — new projects work without them, and existing projects shouldn't break during migration.
   - *Rejected*: Making all companion files required — would break existing projects without migration.

2. **Hardcoded fallback in sync-workspace.sh instead of requiring model-tiers.yaml**
   - *Why*: With model-tiers.yaml deleted, the script needs a fallback for projects without `model_tiers` in config.yaml (or before config.yaml exists during bootstrap). Hardcoding `haiku` as the fast-tier default is simpler than maintaining a separate defaults file.
   - *Rejected*: Keeping model-tiers.yaml as a defaults file — defeats the purpose of consolidation.

3. **Markdown sections instead of YAML keys for code-quality.md**
   - *Why*: The whole point of extraction is freeing prose from YAML constraints. Using `## Principles` / `## Anti-Patterns` / `## Test Strategy` as sections lets users write naturally with examples, code blocks, and multi-paragraph explanations.
   - *Rejected*: YAML frontmatter with markdown body — adds parsing complexity for no benefit.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab/context.md` as filename | Mirrors `fab/constitution.md` naming; confirmed by user in discussion. Confirmed from intake #1 | S:90 R:90 A:95 D:90 |
| 2 | Certain | `fab/code-quality.md` as filename | Natural name following companion-file pattern; confirmed by user. Confirmed from intake #2 | S:85 R:85 A:90 D:85 |
| 3 | Certain | Delete `stages:` entirely (no move/replacement) | Zero consumers confirmed — workflow.yaml is complete superset. Confirmed from intake #3 | S:95 R:95 A:95 D:95 |
| 4 | Certain | `model_tiers:` goes into config.yaml (not a separate file) | User explicitly chose this to simplify mental model. Confirmed from intake #4 | S:95 R:85 A:90 D:90 |
| 5 | Confident | context.md and code-quality.md are optional (no error if missing) | Constitution is required; these are supplementary. New projects get them from scaffold. Confirmed from intake #5 | S:70 R:90 A:80 D:75 |
| 6 | Confident | Hardcoded `haiku` fallback in sync-workspace.sh | Maintains "just works" behavior. Simple and explicit — no hidden file dependencies. Upgraded from intake #6 (was about fab-sync.sh, now about sync-workspace.sh) | S:75 R:85 A:80 D:70 |
| 7 | Confident | `context` removed from /fab-setup config sections (not just a rename) | Context is now a standalone file, not a config section. Editing it is a file-level operation, not a config-section edit | S:80 R:80 A:75 D:70 |
| 8 | Certain | Migration included in this change as a migration file | User chose to include migration rather than defer. Clarified from intake Tentative #7 | S:95 R:70 A:95 D:95 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-02-18

1. **Migration scope**: Should migration be deferred to a follow-up change or included here?
   → **Include migration file** — add a migration markdown file that automates extraction of context, code_quality, stages removal, and model_tiers addition. Assumption #8 upgraded from Tentative to Certain.

2. **fab-help.md / fab-new.md references**: Intake mentioned updating these but spec didn't address them.
   → **Verify during tasks** — these likely don't need changes (context.md is optional, fab-help delegates to shell script). Will verify during task generation and add tasks if needed.
