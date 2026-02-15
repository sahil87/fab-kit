# Spec: Naming Linear ID & Drop Conventions

**Change**: 260215-w3n8-naming-linear-id-drop-conventions
**Created**: 2026-02-15
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/planning-skills.md`

## Non-Goals

- Migrating existing change folder names — new format is forward-only
- Adding Linear ID parsing to `resolve-change.sh` or other scripts — substring matching is format-agnostic
- Changing branch naming behavior — branch name already equals folder name when `branch_prefix` is `""`

## fab-workflow: Naming Convention Extension

### Requirement: Optional Linear Issue ID in Folder Name

The change folder naming format SHALL support an optional Linear issue ID component: `{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}`.

- When a Linear issue ID is available (from ticket input or backlog entry), the format SHALL be `{YYMMDD}-{XXXX}-{ISSUE}-{slug}` (e.g., `260115-a7k2-DEV-988-add-oauth`)
- When no Linear issue ID is available, the format SHALL remain `{YYMMDD}-{XXXX}-{slug}` (backward compatible)
- The `{ISSUE}` component SHALL be uppercase (`[A-Z]+-\d+`), breaking the all-lowercase convention for disambiguation — the uppercase pattern is distinct from the lowercase random token and slug
- The `{ISSUE}` component SHALL be placed after the 4-char random/backlog token and before the slug
- Hyphens SHALL be used as delimiters between all components (no alternate delimiter needed)

#### Scenario: Change created from Linear ticket

- **GIVEN** the user invokes `/fab-new` with a Linear ticket ID `DEV-988`
- **WHEN** the folder name is generated
- **THEN** the format SHALL be `{YYMMDD}-{XXXX}-DEV-988-{slug}` with the Linear ID in uppercase

#### Scenario: Change created from backlog entry with Linear ID

- **GIVEN** the user invokes `/fab-new` with a backlog ID that references a Linear ticket
- **WHEN** the Linear ticket is fetched successfully
- **THEN** the folder name SHALL include the Linear issue ID: `{YYMMDD}-{XXXX}-{ISSUE}-{slug}`

#### Scenario: Change created without Linear context

- **GIVEN** the user invokes `/fab-new` with a plain text description (no Linear ticket, no backlog entry with Linear ID)
- **WHEN** the folder name is generated
- **THEN** the format SHALL be `{YYMMDD}-{XXXX}-{slug}` (no `{ISSUE}` component)

#### Scenario: Linear fetch fails gracefully

- **GIVEN** the user invokes `/fab-new` with a Linear ticket ID
- **WHEN** the Linear API fetch fails
- **THEN** the folder name SHALL fall back to `{YYMMDD}-{XXXX}-{slug}` without the issue ID
- **AND** the slug SHALL be derived from the raw ticket ID text or user description

### Requirement: Config Naming Format Documentation

The `naming.format` field in `fab/config.yaml` SHALL be updated to document the extended format pattern `{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}` with:

- An explanation of the optional `{ISSUE}` component
- A note that `{ISSUE}` is uppercase (Linear issue ID, e.g., `DEV-988`)
- Updated example output showing both with and without the Linear ID

#### Scenario: Config reflects extended format

- **GIVEN** the change is applied
- **WHEN** a user reads `fab/config.yaml`'s `naming` section
- **THEN** the `format` value SHALL be `"{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}"`
- **AND** the inline comments SHALL explain the optional `{ISSUE}` component

### Requirement: Skill Prompt Updates for Naming

The `/fab-new` skill prompt (`fab/.kit/skills/fab-new.md`) SHALL reference the extended format in:

- Step 1 (Generate Folder Name) — updated format string and generation logic
- Step 0 (Parse Input) — Linear ID extraction already exists; the parsed ID SHALL be carried through to Step 1

The `/fab-init` skill prompt (`fab/.kit/skills/fab-init.md`) SHALL update the config template's `naming` section to match the extended format when generating new `config.yaml` files.

#### Scenario: fab-new generates folder with Linear ID from ticket input

- **GIVEN** `/fab-new` parses a Linear ticket ID in Step 0
- **WHEN** Step 1 generates the folder name
- **THEN** the `{ISSUE}` component SHALL be inserted between `{XXXX}` and `{slug}`

#### Scenario: fab-init template uses extended format

- **GIVEN** `/fab-init` generates a new `config.yaml`
- **WHEN** the `naming` section is written
- **THEN** the `format` field SHALL show `"{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}"`
- **AND** inline comments SHALL explain the optional Linear ID component

## fab-workflow: Drop Conventions Section

### Requirement: Remove `conventions` Section from Config Schema

The `conventions` section SHALL be removed from `fab/config.yaml`. This includes:

- The commented-out `conventions:` block and all its sub-keys (`branch_naming`, `pr_title`, `backlog`)
- All associated inline comments explaining the section

#### Scenario: Conventions block removed from config

- **GIVEN** the current `fab/config.yaml` contains a commented-out `conventions` section (lines 77-90)
- **WHEN** the change is applied
- **THEN** the entire `conventions` section (comments and commented-out YAML) SHALL be deleted
- **AND** no skill references to `conventions` SHALL remain

### Requirement: Remove `conventions` from Config Schema Documentation

The `conventions` section SHALL be removed from the memory file `docs/memory/fab-workflow/configuration.md`:

- Remove the `#### conventions` subsection under `config.yaml` Schema
- Remove `conventions` from the `/fab-init config` editable sections list (if present)

#### Scenario: Memory no longer documents conventions

- **GIVEN** `docs/memory/fab-workflow/configuration.md` currently documents a `conventions` section
- **WHEN** the change is hydrated
- **THEN** the `#### conventions` subsection SHALL be absent from the config schema documentation
- **AND** the changelog SHALL record the removal

### Requirement: Remove `conventions` from Fab-Init Valid Sections

The `/fab-init` skill prompt SHALL remove `conventions` from its valid config sections list if it appears there. The `config [section]` argument help and section menu SHALL not list `conventions`.

#### Scenario: fab-init no longer offers conventions editing

- **GIVEN** a user runs `/fab-init config`
- **WHEN** the section menu is displayed
- **THEN** `conventions` SHALL NOT appear as an option

## Design Decisions

1. **Uppercase Linear ID in otherwise-lowercase name**: The Linear issue ID stays uppercase (e.g., `DEV-988`) despite the all-lowercase convention for other components.
   - *Why*: Makes regex parsing unambiguous — `[A-Z]+-\d+` is distinct from the lowercase random token (`[a-z0-9]{4}`) and slug. No script currently parses name components, but the visual distinction aids human scanning.
   - *Rejected*: Lowercasing the issue ID — would make `dev-988` ambiguous with slug words like `dev-tools`. Alternate delimiters (underscores, dots) — unnecessary complexity when casing already disambiguates.

2. **Hyphen delimiter throughout**: All components remain hyphen-separated, including the Linear ID.
   - *Why*: Consistent with existing format. The uppercase pattern provides sufficient disambiguation without a different separator.
   - *Rejected*: Using `/` or `_` around the Linear ID — filesystem issues with `/`, visual noise with `_`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Linear ID placed after random token, before slug | Confirmed from brief #1 — user explicitly chose Option C | S:95 R:90 A:95 D:95 |
| 2 | Certain | Linear ID stays uppercase in folder name | Confirmed from brief #2 — user explicitly stated preference | S:95 R:85 A:90 D:95 |
| 3 | Certain | Branch name = folder name (no separate convention) | Confirmed from brief #3 — already the default with empty `branch_prefix` | S:95 R:90 A:95 D:95 |
| 4 | Certain | Delete conventions section entirely | Confirmed from brief #4 — user explicitly requested; no skill consumes it | S:95 R:70 A:90 D:95 |
| 5 | Confident | Hyphen delimiter sufficient (no alternate delimiter) | Confirmed from brief #5 — uppercase Linear ID makes parsing unambiguous | S:80 R:75 A:85 D:70 |
| 6 | Confident | No migration for existing folder names | Confirmed from brief #6 — forward-only; `resolve-change.sh` uses substring matching | S:70 R:85 A:80 D:75 |
| 7 | Confident | 6 files in scope (config.yaml, fab-new.md, fab-init.md, change-lifecycle.md, configuration.md, planning-skills.md) | Confirmed from brief #7 — audit found no additional references | S:75 R:85 A:75 D:70 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
