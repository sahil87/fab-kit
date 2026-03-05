# Spec: Expand fab-init Command Family

**Change**: 260212-h9k3-fab-init-family
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/init-family.md`, `fab/docs/fab-workflow/config-management.md`, `fab/docs/fab-workflow/constitution-governance.md`, `fab/docs/fab-workflow/init.md`, `fab/docs/fab-workflow/configuration.md`

## Non-Goals

- Semantic validation of constitution principles (e.g., "are these the right principles?") — only structural checks
- Auto-migration of config.yaml across kit versions — that remains `fab-update.sh`'s domain
- Implementing the `fab-hydrate-design` rename (tracked separately in backlog item [akhp])
- Adding a `fab-update` agent command — `fab-update.sh` remains script-only per user decision

---

## Init Family: fab-init-constitution

### Requirement: Dual-Mode Operation

`/fab-init-constitution` SHALL operate in two modes determined by file existence:

- **Create mode**: When `fab/constitution.md` does not exist, generate a new constitution interactively (identical behavior to current `/fab-init` step 1b)
- **Update mode**: When `fab/constitution.md` already exists, present the current constitution and offer guided amendment

#### Scenario: Create Constitution — Fresh Project
- **GIVEN** `fab/config.yaml` exists but `fab/constitution.md` does not
- **WHEN** the user runs `/fab-init-constitution`
- **THEN** the skill reads project context from `config.yaml`, README, and codebase
- **AND** generates `fab/constitution.md` with Core Principles, Additional Constraints, and Governance sections
- **AND** sets the Governance version to `1.0.0` with today's date as Ratified and Last Amended

#### Scenario: Create Constitution — No Config
- **GIVEN** `fab/config.yaml` does not exist
- **WHEN** the user runs `/fab-init-constitution`
- **THEN** the skill outputs: "fab/config.yaml not found. Run /fab-init first."

#### Scenario: Update Constitution — Existing File
- **GIVEN** `fab/constitution.md` exists with version `1.2.0`
- **WHEN** the user runs `/fab-init-constitution`
- **THEN** the skill displays the current constitution content
- **AND** asks what changes the user wants (add principle, modify principle, remove principle, add/modify constraint, update governance)
- **AND** applies the requested changes
- **AND** asks "Any other changes?" to allow additional amendments in the same session
- **AND** loops until the user declines

#### Scenario: Multiple Amendments — Version Precedence
- **GIVEN** `fab/constitution.md` exists with version `1.2.0`
- **WHEN** the user adds a new principle (MINOR) and removes an existing principle (MAJOR) in the same session
- **THEN** the version is bumped to `2.0.0` (MAJOR takes precedence)
- **AND** the output summarizes all amendments applied

### Requirement: Semantic Versioning for Amendments

Constitution version changes SHALL follow semantic versioning:

- **MAJOR** bump: Removing or fundamentally changing an existing principle
- **MINOR** bump: Adding a new principle or constraint
- **PATCH** bump: Clarifying wording without changing meaning

The skill SHALL automatically determine the correct bump level based on the type of change and update the Governance section accordingly. When multiple amendments are made in a single session, the highest-severity bump SHALL take precedence (MAJOR > MINOR > PATCH).
<!-- clarified: multiple amendments per session with highest-precedence version bump -->

#### Scenario: Add New Principle
- **GIVEN** `fab/constitution.md` exists with version `1.2.0`
- **WHEN** the user adds a new Core Principle "VII. Test Coverage"
- **THEN** the version is bumped to `1.3.0`
- **AND** the Last Amended date is updated to today

#### Scenario: Remove Existing Principle
- **GIVEN** `fab/constitution.md` exists with version `1.3.0`
- **WHEN** the user removes Core Principle "III. Idempotent Operations"
- **THEN** the version is bumped to `2.0.0`
- **AND** the Last Amended date is updated to today

#### Scenario: Clarify Wording
- **GIVEN** `fab/constitution.md` exists with version `1.3.0`
- **WHEN** the user rephrases a principle without changing its meaning
- **THEN** the version is bumped to `1.3.1`
- **AND** the Last Amended date is updated to today

### Requirement: Governance Audit Trail

When updating the constitution, the skill SHALL preserve an audit trail by including a brief summary of the amendment in the output. The constitution file itself does not need a changelog section — the git history serves as the authoritative record.

#### Scenario: Amendment Summary in Output
- **GIVEN** the user amends principle "II. Docs Are Source of Truth" to add a clarification
- **WHEN** the amendment is applied
- **THEN** the output includes: "Amended: II. Docs Are Source of Truth (clarification) — version 1.0.1"

### Requirement: Structural Preservation

When updating `fab/constitution.md`, the skill SHALL preserve:

- The overall document structure (heading hierarchy, section order)
- Roman numeral numbering of Core Principles (re-numbering when principles are removed)
- RFC 2119 keyword usage (MUST/SHALL/SHOULD/MAY) in principle text
- The Governance section format

#### Scenario: Principle Removal Re-numbers
- **GIVEN** constitution has principles I through VI
- **WHEN** principle III is removed
- **THEN** former principles IV, V, VI become III, IV, V
- **AND** version receives a MAJOR bump

### Requirement: Idempotency

`/fab-init-constitution` SHALL be safe to re-run. In update mode, running the skill and making no changes SHALL leave the file unchanged.

#### Scenario: No-Op Update
- **GIVEN** `fab/constitution.md` exists
- **WHEN** the user runs `/fab-init-constitution` and chooses not to make any changes
- **THEN** the file is not modified
- **AND** the version is not bumped

---

## Init Family: fab-init-config

### Requirement: Section-Based Interactive Updates

`/fab-init-config` SHALL present the user with a menu of config sections to update:

1. `project` — name and description
2. `context` — tech stack and conventions
3. `source_paths` — implementation code directories
4. `stages` — pipeline stage definitions
5. `rules` — per-stage generation rules
6. `checklist` — extra quality categories
7. `git` — branch integration settings
8. `naming` — change folder naming format

The skill SHALL accept an optional section argument (e.g., `/fab-init-config context`) to jump directly to editing that section, skipping the menu. When no argument is provided, the full menu is displayed. The user SHOULD be able to update multiple sections in a single invocation.
<!-- clarified: optional section argument supported for direct access -->

#### Scenario: Direct Section Access via Argument
- **GIVEN** `fab/config.yaml` exists
- **WHEN** the user runs `/fab-init-config context`
- **THEN** the skill skips the menu and shows the current `context` value directly
- **AND** asks for the updated value
- **AND** writes the update to `fab/config.yaml`

#### Scenario: Invalid Section Argument
- **GIVEN** `fab/config.yaml` exists
- **WHEN** the user runs `/fab-init-config foobar`
- **THEN** the skill outputs: "Unknown section 'foobar'. Valid sections: project, context, source_paths, stages, rules, checklist, git, naming"

#### Scenario: Update Single Section via Menu
- **GIVEN** `fab/config.yaml` exists
- **WHEN** the user runs `/fab-init-config` and selects "context"
- **THEN** the skill shows the current `context` value
- **AND** asks for the updated value
- **AND** writes the update to `fab/config.yaml`

#### Scenario: Update Multiple Sections
- **GIVEN** `fab/config.yaml` exists
- **WHEN** the user runs `/fab-init-config` and selects "source_paths" then "checklist"
- **THEN** both sections are updated in sequence
- **AND** the final file contains both changes

#### Scenario: No Config File
- **GIVEN** `fab/config.yaml` does not exist
- **WHEN** the user runs `/fab-init-config`
- **THEN** the skill outputs: "fab/config.yaml not found. Run /fab-init to create it."

### Requirement: YAML Validation After Edit

After each edit, the skill SHALL validate the resulting YAML:

- The file MUST be parseable as valid YAML
- Required fields (`project.name`, `project.description`, `stages`) MUST be present
- Stage `requires` references MUST point to existing stage IDs

If validation fails, the skill SHALL show the error and offer to revert the change.

#### Scenario: Invalid YAML After Edit
- **GIVEN** the user updates the `context` section
- **WHEN** the resulting YAML is invalid (e.g., broken indentation)
- **THEN** the skill reports: "Validation failed: {error details}"
- **AND** offers to revert to the previous content

#### Scenario: Broken Stage Reference
- **GIVEN** the user modifies the `stages` section
- **WHEN** a stage's `requires` references a non-existent stage ID
- **THEN** the skill reports: "Stage '{id}' requires non-existent stage '{ref}'"
- **AND** offers to fix or revert

### Requirement: Comment and Formatting Preservation

The skill SHOULD preserve existing YAML comments and formatting when updating sections. Since YAML parsers typically strip comments, the skill SHALL use targeted string replacement on the specific section being updated rather than full parse-and-rewrite.

#### Scenario: Comments Preserved
- **GIVEN** `fab/config.yaml` has inline comments on the `git` section
- **WHEN** the user updates the `git.branch_prefix` value
- **THEN** the surrounding comments are preserved in the output file

### Requirement: Idempotency

`/fab-init-config` SHALL be safe to re-run. Running the skill and making no changes SHALL leave the file unchanged.

#### Scenario: No-Op Run
- **GIVEN** `fab/config.yaml` exists
- **WHEN** the user runs `/fab-init-config` and exits without selecting any sections
- **THEN** the file is not modified

---

## Init Family: fab-init-validate

### Requirement: Config Structural Validation

`/fab-init-validate` SHALL check `fab/config.yaml` for structural correctness:

1. File exists and is parseable as valid YAML
2. Required top-level keys present: `project`, `context`, `stages`, `source_paths`
3. `project.name` and `project.description` are non-empty strings
4. `stages` is a non-empty list
5. Each stage has a required `id` field (string)
6. Stage `requires` references (if present) point to existing stage IDs
7. No circular dependencies in stage `requires` graph
8. Stage IDs are unique

#### Scenario: Valid Config
- **GIVEN** `fab/config.yaml` is structurally valid
- **WHEN** the user runs `/fab-init-validate`
- **THEN** the output shows all checks passing with a summary: "config.yaml: 8/8 checks passed"

#### Scenario: Missing Required Field
- **GIVEN** `fab/config.yaml` is missing `project.name`
- **WHEN** the user runs `/fab-init-validate`
- **THEN** the output reports: "FAIL: project.name is missing or empty"
- **AND** suggests: "Add `name: \"your-project\"` under the `project:` section"

#### Scenario: Circular Stage Dependencies
- **GIVEN** `fab/config.yaml` has stages where `spec` requires `tasks` and `tasks` requires `spec`
- **WHEN** the user runs `/fab-init-validate`
- **THEN** the output reports: "FAIL: Circular dependency detected: spec → tasks → spec"

### Requirement: Constitution Structural Validation

`/fab-init-validate` SHALL check `fab/constitution.md` for structural correctness:

1. File exists and is non-empty
2. Has a level-1 heading (`# ... Constitution`)
3. Has a `## Core Principles` section
4. Core Principles use Roman numeral headings (`### I.`, `### II.`, etc.)
5. Has a `## Governance` section
6. Governance section contains version in `MAJOR.MINOR.PATCH` format

#### Scenario: Valid Constitution
- **GIVEN** `fab/constitution.md` is structurally valid
- **WHEN** the user runs `/fab-init-validate`
- **THEN** the output shows all checks passing: "constitution.md: 6/6 checks passed"

#### Scenario: Missing Governance Section
- **GIVEN** `fab/constitution.md` has Core Principles but no Governance section
- **WHEN** the user runs `/fab-init-validate`
- **THEN** the output reports: "FAIL: Missing ## Governance section"
- **AND** suggests: "Add a Governance section with version, ratified date, and last amended date"

### Requirement: Combined Report

When both files exist, `/fab-init-validate` SHALL validate both and present a combined report.

#### Scenario: Both Files Valid
- **GIVEN** both `fab/config.yaml` and `fab/constitution.md` exist and are valid
- **WHEN** the user runs `/fab-init-validate`
- **THEN** the output shows:
  ```
  config.yaml:      8/8 checks passed ✓
  constitution.md:  6/6 checks passed ✓

  All validation checks passed.
  ```

#### Scenario: One File Missing
- **GIVEN** `fab/config.yaml` exists but `fab/constitution.md` does not
- **WHEN** the user runs `/fab-init-validate`
- **THEN** config.yaml is validated normally
- **AND** the output includes: "constitution.md: not found — run /fab-init or /fab-init-constitution to create it"

### Requirement: Actionable Fix Suggestions

Every validation failure SHALL include a concrete suggestion for how to fix the issue. Suggestions MUST reference the specific field, section, or structure that needs attention.

#### Scenario: Actionable Fix
- **GIVEN** `fab/config.yaml` has an empty `stages` list
- **WHEN** validation runs
- **THEN** the output reports: "FAIL: stages list is empty"
- **AND** suggests: "Add at least the default stages (brief, spec, tasks, apply, review, archive)"

### Requirement: Exit Status

`/fab-init-validate` SHALL communicate validation results clearly:

- All checks pass: output ends with "All validation checks passed."
- Any check fails: output ends with "{N} issue(s) found. Fix the issues above and re-run /fab-init-validate."

#### Scenario: Validation Fails
- **GIVEN** config.yaml has 2 issues
- **WHEN** validation completes
- **THEN** the output lists both issues with suggestions
- **AND** ends with: "2 issue(s) found. Fix the issues above and re-run /fab-init-validate."

---

## Init Family: fab-init.md Updates

### Requirement: Delegate Creation to Family Commands

`/fab-init` SHALL delegate artifact creation to the family commands rather than containing independent generation logic:

- Step 1a (`config.yaml`): If `fab/config.yaml` does not exist, `/fab-init` SHALL invoke `/fab-init-config` in create mode
- Step 1b (`constitution.md`): If `fab/constitution.md` does not exist, `/fab-init` SHALL invoke `/fab-init-constitution` in create mode

This ensures a single source of truth for generation logic. `/fab-init` retains ownership of the overall orchestration (phase ordering, structural bootstrap, symlinks, `.gitignore`) but defers config and constitution generation to the specialized commands.
<!-- clarified: fab-init delegates to family commands rather than duplicating generation logic -->

#### Scenario: Init Delegates Config Creation
- **GIVEN** `fab/config.yaml` does not exist
- **WHEN** the user runs `/fab-init`
- **THEN** `/fab-init` invokes `/fab-init-config` in create mode for step 1a
- **AND** the resulting `config.yaml` is identical to what `/fab-init-config` would produce standalone

#### Scenario: Init Delegates Constitution Creation
- **GIVEN** `fab/constitution.md` does not exist
- **WHEN** the user runs `/fab-init`
- **THEN** `/fab-init` invokes `/fab-init-constitution` in create mode for step 1b
- **AND** the resulting `constitution.md` is identical to what `/fab-init-constitution` would produce standalone

#### Scenario: Init Skips Delegation When Files Exist
- **GIVEN** both `fab/config.yaml` and `fab/constitution.md` already exist
- **WHEN** the user runs `/fab-init`
- **THEN** neither family command is invoked
- **AND** `/fab-init` reports both files as "already exists — skipping"

### Requirement: Cross-Reference to Family Commands

`/fab-init` skill file SHALL include a "Related Commands" section after the Error Handling table, listing the init family commands with one-line descriptions.

#### Scenario: Related Commands Section
- **GIVEN** the updated `/fab-init` skill file
- **WHEN** a user reads the skill documentation
- **THEN** they find a "Related Commands" section listing:
  - `/fab-init-constitution` — Create or amend project constitution
  - `/fab-init-config` — Interactive config.yaml updates
  - `/fab-init-validate` — Structural validation for config and constitution

---

## Design Decisions

1. **Separate skills per concern, not a single `fab-init --subcommand`**
   - *Why*: Follows the existing Fab pattern of one skill file per command (e.g., `fab-init.md`, `fab-switch.md`). Each skill gets its own `model_tier`, description, and frontmatter. Subcommand routing would add parsing complexity with no benefit.
   - *Rejected*: Single `fab-init` with `--constitution`, `--config`, `--validate` flags — requires argument parsing logic, muddies the simple "one skill = one action" pattern.

2. **Constitution has no embedded changelog — git history is the record**
   - *Why*: The constitution is version-controlled in git. Adding a changelog section inside the markdown file would duplicate what git already tracks and add maintenance burden. The Governance version number provides the semantic signal.
   - *Rejected*: Inline changelog table (like centralized docs have) — centralized docs need changelogs because they're hydrated by `/fab-archive` and the changelog tracks which changes contributed. Constitution amendments are direct edits, not hydrated.

3. **Config updates use string replacement, not full YAML parse-and-rewrite**
   - *Why*: Standard YAML parsers strip comments. Config files rely heavily on comments for documentation. Targeted string replacement on the section being updated preserves comments and formatting at the cost of slightly less structural safety.
   - *Rejected*: Full parse-and-rewrite — loses all comments, which degrades the config's self-documenting nature.

4. **Validate checks structural correctness only**
   - *Why*: Structural checks (required fields, valid YAML, no circular deps) are deterministic and automatable. Semantic checks ("are these the right principles?") require human judgment and don't belong in a validation tool.
   - *Rejected*: Semantic validation or "lint" rules — scope creep, hard to define correctness criteria for free-form content like principles.

---

## Deprecated Requirements

*None — this change only adds new functionality.*

---

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | No changelog section inside constitution.md — git history suffices | Existing constitution format has no changelog; centralized docs need changelogs because of archive hydration, but constitution amendments are direct edits |
| 2 | Confident | String-based section replacement for config edits rather than full YAML parse-and-rewrite | Constitution Principle IV (Markdown-Only Artifacts) and existing heavily-commented config suggest preserving comments is important |
| 3 | Confident | Each init-family command gets its own skill file (not subcommand routing) | Matches existing one-skill-one-file pattern across all fab commands |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-12

- **Q**: Should `/fab-init` delegate constitution/config creation to the family commands, or keep independent logic?
  **A**: Delegate — `/fab-init` invokes `/fab-init-constitution` and `/fab-init-config` for steps 1b and 1a. Single source of truth for generation logic.
- **Q**: Should users be able to make multiple amendments per `/fab-init-constitution` session, with highest-precedence version bump?
  **A**: Yes, allow multiple amendments per session. Version bump uses highest precedence (MAJOR > MINOR > PATCH).
- **Q**: Should `/fab-init-config` accept an optional section argument to skip the menu?
  **A**: Yes, support optional section argument (e.g., `/fab-init-config context`) for direct access.
