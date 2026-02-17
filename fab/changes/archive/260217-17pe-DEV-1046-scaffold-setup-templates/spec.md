# Spec: Scaffold Setup Templates

**Change**: 260217-17pe-DEV-1046-scaffold-setup-templates
**Created**: 2026-02-17
**Affected memory**: `docs/memory/fab-workflow/setup.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Changing any behavioral output of `/fab-setup` — generated `config.yaml` and `constitution.md` content remains identical for users
- Modifying `fab-sync.sh` — it already reads from scaffold files for index templates
- Creating migration files — this change only affects new project setup, not existing projects

## Scaffold: Config Template

### Requirement: Config Template File

`fab/.kit/scaffold/` SHALL contain a `config.yaml` file serving as the canonical starting template for `/fab-setup config` in create mode. The file SHALL contain all config sections with placeholder values using `{PLACEHOLDER}` syntax and full inline comments matching the current config schema.

#### Scenario: Scaffold config.yaml exists after kit distribution

- **GIVEN** a fresh copy of `fab/.kit/`
- **WHEN** a user lists `fab/.kit/scaffold/`
- **THEN** `config.yaml` is present
- **AND** it contains all sections: `project`, `context`, `naming`, `git`, `stages`, `source_paths`, `checklist`, `rules`, `code_quality` (commented out)

#### Scenario: Placeholder syntax

- **GIVEN** `fab/.kit/scaffold/config.yaml`
- **WHEN** the file is read
- **THEN** placeholder values use `{PLACEHOLDER_NAME}` syntax (e.g., `{PROJECT_NAME}`, `{PROJECT_DESCRIPTION}`, `{TECH_STACK_AND_CONVENTIONS}`, `{SOURCE_PATHS}`)
- **AND** non-placeholder values (e.g., stage definitions, naming format, default git settings) contain their actual defaults

### Requirement: Config Template Content Match

The scaffold `config.yaml` SHALL be a structural superset of the inline YAML currently in `fab-setup.md` Config Create Mode (lines 192-259). The header comment, sections, default values, and commented-out `code_quality` block MUST all be present.

#### Scenario: Content equivalence with inline template

- **GIVEN** the current inline YAML in `fab-setup.md` Config Create Mode
- **WHEN** compared to `fab/.kit/scaffold/config.yaml`
- **THEN** every section, key, default value, and comment block from the inline version is present in the scaffold file
- **AND** the only differences are placeholder values where the inline version had `"{PROJECT_NAME}"` etc.

## Scaffold: Constitution Template

### Requirement: Constitution Template File

`fab/.kit/scaffold/` SHALL contain a `constitution.md` file serving as a minimal skeleton for `/fab-setup constitution` in create mode. The skeleton SHALL provide structural scaffolding (headings, sections, Governance block) without example principles — `/fab-setup` generates those dynamically from project context.

#### Scenario: Scaffold constitution.md exists

- **GIVEN** a fresh copy of `fab/.kit/`
- **WHEN** a user lists `fab/.kit/scaffold/`
- **THEN** `constitution.md` is present
- **AND** it contains the headings: `# {Project Name} Constitution`, `## Core Principles`, `## Additional Constraints`, `## Governance`

#### Scenario: Constitution skeleton is minimal

- **GIVEN** `fab/.kit/scaffold/constitution.md`
- **WHEN** the file is read
- **THEN** the `## Core Principles` section contains a single placeholder principle entry (`### I. {Principle Name}`) with instructional text
- **AND** the `## Governance` section contains the version/date template with `{DATE}` placeholders
- **AND** the file does NOT contain fully-written example principles

## Fab-Setup Skill: Config Create Mode

### Requirement: Config Create Reads from Scaffold

`fab-setup.md` Config Create Mode SHALL instruct the agent to read `fab/.kit/scaffold/config.yaml` as the starting template, substitute placeholders with user-provided values, and write the result to `fab/config.yaml`. The inline YAML block (currently lines 192-259) SHALL be replaced with a reference to the scaffold file.

#### Scenario: Config create mode uses scaffold

- **GIVEN** `fab/config.yaml` does not exist
- **WHEN** `/fab-setup` executes Config Create Mode
- **THEN** the agent reads `fab/.kit/scaffold/config.yaml`
- **AND** substitutes `{PROJECT_NAME}`, `{PROJECT_DESCRIPTION}`, `{TECH_STACK_AND_CONVENTIONS}`, `{SOURCE_PATHS}` with user-provided values
- **AND** writes the result to `fab/config.yaml`

#### Scenario: Generated config matches current output

- **GIVEN** a user provides the same inputs as before
- **WHEN** `/fab-setup` generates `fab/config.yaml` using the scaffold template
- **THEN** the output is structurally identical to what the inline template would have produced

## Fab-Setup Skill: Constitution Create Mode

### Requirement: Constitution Create Reads from Scaffold

`fab-setup.md` Constitution Create Mode SHALL instruct the agent to read `fab/.kit/scaffold/constitution.md` as the starting skeleton, generate principles based on project context filling the skeleton structure, and write the result to `fab/constitution.md`. The inline markdown block (currently lines 337-356) SHALL be replaced with a reference to the scaffold file.

#### Scenario: Constitution create mode uses scaffold

- **GIVEN** `fab/constitution.md` does not exist
- **WHEN** `/fab-setup` executes Constitution Create Mode
- **THEN** the agent reads `fab/.kit/scaffold/constitution.md`
- **AND** generates principles based on project context, filling in the skeleton structure
- **AND** writes the result to `fab/constitution.md`

## Fab-Setup Skill: Index Template References

### Requirement: Memory Index Uses Scaffold Reference

`fab-setup.md` step 1c (memory index creation) SHALL replace the inline markdown template with an instruction to copy from `fab/.kit/scaffold/memory-index.md`. This aligns with what `fab-sync.sh` already does.

#### Scenario: Memory index step references scaffold

- **GIVEN** `docs/memory/index.md` does not exist
- **WHEN** `/fab-setup` executes step 1c
- **THEN** it reads `fab/.kit/scaffold/memory-index.md` and writes the content to `docs/memory/index.md`

### Requirement: Specs Index Uses Scaffold Reference

`fab-setup.md` step 1d (specs index creation) SHALL replace the inline markdown template with an instruction to copy from `fab/.kit/scaffold/specs-index.md`. This aligns with what `fab-sync.sh` already does.

#### Scenario: Specs index step references scaffold

- **GIVEN** `docs/specs/index.md` does not exist
- **WHEN** `/fab-setup` executes step 1d
- **THEN** it reads `fab/.kit/scaffold/specs-index.md` and writes the content to `docs/specs/index.md`

#### Scenario: Inline template divergence eliminated

- **GIVEN** the scaffold files `memory-index.md` and `specs-index.md` exist in `fab/.kit/scaffold/`
- **WHEN** `/fab-setup` runs step 1c or 1d
- **THEN** both `/fab-setup` and `fab-sync.sh` use the same source file for each index template
- **AND** there is no inline template in `fab-setup.md` that could diverge from the scaffold

## Design Decisions

1. **Scaffold files use placeholder syntax, not Mustache/Handlebars**: Placeholders use `{NAME}` format consistent with existing templates in `fab/.kit/templates/`. No templating engine involved — the LLM agent performs substitution directly.
   - *Why*: Maintains Pure Prompt Play principle (Constitution I). No build tools or template engines needed.
   - *Rejected*: Mustache/Handlebars templates — would require a rendering engine, violating the constitution.

2. **Constitution scaffold is structural skeleton only**: The scaffold provides headings and a single placeholder principle entry, not populated example principles.
   - *Why*: `/fab-setup constitution` already generates principles dynamically from project context. A populated example would be misleading — it might get left in verbatim instead of being replaced.
   - *Rejected*: Full example constitution with realistic principles — risk of users adopting boilerplate principles that don't fit their project.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scaffold files use `{PLACEHOLDER}` syntax | Confirmed from intake #1 — consistent with `fab/.kit/templates/` convention | S:90 R:90 A:95 D:90 |
| 2 | Certain | `fab-sync.sh` is not modified | Confirmed from intake #2 — it already copies from scaffold files for index templates | S:95 R:95 A:95 D:95 |
| 3 | Confident | Constitution scaffold is a minimal skeleton, not a full example | Confirmed from intake #3 — `/fab-setup` generates principles dynamically from project context | S:80 R:85 A:80 D:70 |
| 4 | Certain | No migration file needed | Confirmed from intake #4 — only affects new project setup | S:90 R:95 A:90 D:95 |
| 5 | Certain | Inline templates in fab-setup.md steps 1c/1d are fully replaced by scaffold references | Constitution V (Portability) — `.kit/` owns its templates as files; aligns fab-setup.md with fab-sync.sh | S:90 R:90 A:90 D:90 |
| 6 | Confident | Minor text differences between scaffold index files and inline versions resolve in favor of scaffold | The scaffold files become the single source of truth; any small wording differences (e.g., attribution text) are acceptable since the scaffold is what `fab-sync.sh` already uses | S:75 R:85 A:80 D:70 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
