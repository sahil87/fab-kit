# Spec: Migrate Scripts to Use Stage Manager

**Change**: 260212-4tw0-migrate-scripts-stageman
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/kit-architecture.md`, `fab/docs/fab-workflow/preflight.md`, `fab/docs/fab-workflow/schemas.md` (new)

## Non-Goals

- Changing stageman.sh API or workflow.yaml schema — this change is a consumer of the existing API, not a modification of it
- Migrating skills (markdown files) — only shell scripts are in scope
- Adding new stageman functions — use only existing API surface

## Script Migration: fab-status.sh

### Requirement: Source stageman.sh

`fab-status.sh` SHALL source `stageman.sh` at the top of the script, after resolving paths but before any stage/state logic.

#### Scenario: Stageman sourced successfully
- **GIVEN** `fab/.kit/scripts/stageman.sh` exists
- **WHEN** `fab-status.sh` is executed
- **THEN** stageman functions are available for use in the script

### Requirement: Replace hardcoded stage iteration

The hardcoded loop `for s in brief spec tasks apply review archive` SHALL be replaced with `for s in $(get_all_stages)`.

#### Scenario: Stage list matches schema
- **GIVEN** `workflow.yaml` defines stages `[brief, spec, tasks, apply, review, archive]`
- **WHEN** `fab-status.sh` iterates over stages
- **THEN** the iteration uses `get_all_stages` and matches the schema definition exactly

#### Scenario: Schema adds a new stage
- **GIVEN** a future `workflow.yaml` adds a stage between `review` and `archive`
- **WHEN** `fab-status.sh` iterates over stages
- **THEN** the new stage appears in the progress display without modifying `fab-status.sh`

### Requirement: Replace hardcoded stage number mapping

The hardcoded `case` statement mapping stage names to numbers (lines 93-97) SHALL be replaced with `get_stage_number "$stage"`.

#### Scenario: Stage number resolved dynamically
- **GIVEN** the active stage is `spec`
- **WHEN** `fab-status.sh` displays the stage number
- **THEN** `get_stage_number "spec"` returns `2` and the output shows `Stage: spec (2/6)`

### Requirement: Replace hardcoded state symbol mapping

The hardcoded `symbol()` function (lines 100-105) SHALL be replaced with calls to `get_state_symbol "$state"`.

#### Scenario: State symbols match schema
- **GIVEN** a stage has state `done`
- **WHEN** the progress line is rendered
- **THEN** `get_state_symbol "done"` returns `✓`

#### Scenario: Unknown state handled
- **GIVEN** a stage has an unrecognized state value
- **WHEN** the progress line is rendered
- **THEN** `get_state_symbol` returns its default symbol (same behavior as current fallback)

### Requirement: Replace hardcoded progress variable extraction

The per-stage progress variables (`p_brief`, `p_spec`, etc.) SHALL be replaced with a dynamic loop using `get_all_stages`.

#### Scenario: Progress extracted dynamically
- **GIVEN** `.status.yaml` has progress entries for all stages
- **WHEN** `fab-status.sh` parses progress
- **THEN** it loops over `get_all_stages` and extracts each stage's state dynamically
- **AND** no hardcoded variable names like `p_brief` appear in the script

### Requirement: Total stage count derived from schema

The hardcoded `/6` in the stage display SHALL be replaced with a count derived from `get_all_stages`.

#### Scenario: Stage count matches schema
- **GIVEN** `workflow.yaml` defines 6 stages
- **WHEN** `fab-status.sh` displays `Stage: spec (2/6)`
- **THEN** the `6` is computed from `get_all_stages | wc -l` (or equivalent), not hardcoded

### Requirement: Output format unchanged

The script's output format SHALL remain identical before and after migration. This is a refactor — no visible behavior change.

#### Scenario: Output parity
- **GIVEN** a change at stage `apply` with progress `brief:done spec:done tasks:done apply:active review:pending archive:pending`
- **WHEN** `fab-status.sh` runs before and after migration
- **THEN** the output is byte-identical

## Script Migration: fab-preflight.sh

### Requirement: Source stageman.sh

`fab-preflight.sh` SHALL source `stageman.sh` at the top of the script, after resolving paths.

#### Scenario: Stageman sourced successfully
- **GIVEN** `fab/.kit/scripts/stageman.sh` exists
- **WHEN** `fab-preflight.sh` is executed
- **THEN** stageman functions are available for use in the script

### Requirement: Replace hardcoded stage iteration

The hardcoded progress field extraction (lines 42-47) and stage derivation loop (line 58) SHALL use `get_all_stages` instead of hardcoded stage names.

#### Scenario: Stage iteration matches schema
- **GIVEN** `workflow.yaml` defines the stage list
- **WHEN** `fab-preflight.sh` extracts progress fields and derives the active stage
- **THEN** both loops use `get_all_stages`

### Requirement: Add status file validation

`fab-preflight.sh` SHALL call `validate_status_file "$status_file"` after confirming the file exists, to catch schema violations early.

#### Scenario: Valid status file passes
- **GIVEN** `.status.yaml` has valid stages and states per the schema
- **WHEN** `fab-preflight.sh` runs validation
- **THEN** validation passes and the script continues to emit YAML output

#### Scenario: Invalid status file caught
- **GIVEN** `.status.yaml` contains an invalid state like `progress: brief: bogus`
- **WHEN** `fab-preflight.sh` runs validation
- **THEN** the script exits non-zero with a diagnostic message on stderr
<!-- assumed: validate_status_file errors should cause preflight to fail — brief says "optional" but the purpose of preflight is to catch issues early -->

### Requirement: Output format unchanged

The YAML output format SHALL remain identical. Only internal implementation changes.

#### Scenario: Output parity
- **GIVEN** a valid active change
- **WHEN** `fab-preflight.sh` runs before and after migration
- **THEN** stdout YAML output is identical

## Script Migration: fab-help.sh

### Requirement: Keep static stage documentation

`fab-help.sh` SHALL keep its stage progression as a static documentation string. The help text is a user-facing reference that benefits from curated wording, not dynamic generation.

#### Scenario: Help output unchanged
- **GIVEN** `fab-help.sh` exists with the current help text
- **WHEN** the migration is complete
- **THEN** `fab-help.sh` output is identical — no changes to this script
<!-- assumed: Static help text preferred over dynamic generation — help text is curated documentation, not a runtime query. Dynamic generation adds complexity for no user benefit since stages rarely change. -->

## Documentation Deduplication

### Requirement: Move MIGRATION.md to change folder

`fab/.kit/schemas/MIGRATION.md` SHALL be moved to `fab/changes/260212-4tw0-migrate-scripts-stageman/MIGRATION.md`. It is change-specific content, not a permanent schema artifact.

#### Scenario: MIGRATION.md relocated
- **GIVEN** `fab/.kit/schemas/MIGRATION.md` exists
- **WHEN** the move is completed
- **THEN** the file exists at `fab/changes/260212-4tw0-migrate-scripts-stageman/MIGRATION.md`
- **AND** `fab/.kit/schemas/MIGRATION.md` no longer exists

### Requirement: Move and trim schemas README to fab/docs

`fab/.kit/schemas/README.md` SHALL be moved to `fab/docs/fab-workflow/schemas.md` with the stageman API section and bash usage examples removed (they duplicate `src/stageman/` content). The resulting doc SHALL focus on: what `workflow.yaml` defines, design principles, how to reference from skills vs scripts, and future enhancements.

#### Scenario: schemas.md created
- **GIVEN** `fab/.kit/schemas/README.md` exists with duplicated API docs
- **WHEN** the move and trim is completed
- **THEN** `fab/docs/fab-workflow/schemas.md` exists with schema-focused content
- **AND** `fab/.kit/schemas/README.md` no longer exists
- **AND** `fab/.kit/schemas/` contains only `workflow.yaml`

#### Scenario: No API duplication
- **GIVEN** `fab/docs/fab-workflow/schemas.md` is created
- **WHEN** its content is reviewed
- **THEN** it contains no stageman function signatures or bash usage examples
- **AND** it links to `src/stageman/README.md` for API details

### Requirement: Consolidate src/stageman/ to single README

`src/stageman/SUMMARY.md`, `src/stageman/SPEC.md`, and `src/stageman/CHANGELOG.md` SHALL be deleted. Their content SHALL be folded into a rewritten `src/stageman/README.md` covering: overview, sources-of-truth links, API reference (from SPEC.md), CLI interface, testing, and changelog.

#### Scenario: Files consolidated
- **GIVEN** `src/stageman/` contains `README.md`, `SUMMARY.md`, `SPEC.md`, `CHANGELOG.md`
- **WHEN** consolidation is completed
- **THEN** only `README.md`, `stageman.sh`, `test.sh`, and `test-simple.sh` remain in `src/stageman/`

#### Scenario: No content lost
- **GIVEN** `SPEC.md` documents API contracts
- **WHEN** it is deleted
- **THEN** its API reference content exists in the rewritten `src/stageman/README.md`

### Requirement: Fix dangling references

All references to deleted/moved files SHALL be updated across the codebase.

#### Scenario: Root README updated
- **GIVEN** `README.md` references `SPEC.md` and `schemas/README.md`
- **WHEN** the fix is applied
- **THEN** `SPEC.md` references point to `src/stageman/README.md`
- **AND** `schemas/README.md` references point to `fab/docs/fab-workflow/schemas.md`

#### Scenario: stageman.sh SEE ALSO updated
- **GIVEN** `stageman.sh` line 374 references old file paths in its help text
- **WHEN** the fix is applied
- **THEN** SEE ALSO paths reflect the new file locations

#### Scenario: docs/index.md updated
- **GIVEN** `fab/docs/fab-workflow/index.md` has no `schemas` entry
- **WHEN** the fix is applied
- **THEN** a `schemas` entry is added to the domain's doc table

#### Scenario: No dangling references remain
- **GIVEN** all fixes are applied
- **WHEN** `grep -r 'SPEC.md\|SUMMARY.md\|CHANGELOG.md\|schemas/README' --include='*.md' --include='*.sh'` is run from the project root
- **THEN** no matches are returned (excluding the change's own brief.md and this spec)

## Design Decisions

1. **Keep fab-help.sh static**: Do not dynamically generate stage lists in help text.
   - *Why*: Help text is curated user documentation. Dynamic generation adds shell complexity for negligible benefit — stages change rarely and help wording needs human curation.
   - *Rejected*: Dynamic `get_all_stages` in help — fragile formatting, worse readability, over-engineered.

2. **Validate status file in preflight**: Make `validate_status_file` a hard failure, not a warning.
   - *Why*: Preflight's purpose is to catch issues before skills run. A corrupt status file should stop execution early, not silently continue with bad data.
   - *Rejected*: Warning-only validation — defeats the purpose of preflight validation.

3. **Dynamic progress variables in fab-status.sh**: Replace per-stage named variables with a loop + associative array or indexed extraction.
   - *Why*: Eliminates all hardcoded stage names from the script. The `eval` pattern with named variables (`p_brief`, etc.) is fragile and doesn't scale with schema changes.
   - *Rejected*: Keeping named variables with a loop for assignment — still requires the variable names to exist somewhere.

## Deprecated Requirements

### Hardcoded stage lists in scripts
**Reason**: Replaced by `get_all_stages` from stageman.sh — schema becomes the single source of truth.
**Migration**: All scripts source stageman.sh and use its query functions.

### Hardcoded state symbol mapping in fab-status.sh
**Reason**: Replaced by `get_state_symbol` from stageman.sh.
**Migration**: Delete the `symbol()` function, call `get_state_symbol` instead.

### Hardcoded stage number mapping in fab-status.sh
**Reason**: Replaced by `get_stage_number` from stageman.sh.
**Migration**: Delete the `case` statement, call `get_stage_number` instead.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | fab-help.sh should remain static | Curated help text > dynamic generation; stages rarely change; complexity not justified |
| 2 | Tentative | validate_status_file should be a hard failure in preflight | Brief says "optional" but preflight exists to catch issues — silent continuation defeats the purpose |
| 3 | Confident | fab-status.sh progress variables should use dynamic loop | Eliminates all hardcoded stage names; the `eval` pattern is fragile |
| 4 | Confident | `fab/.kit/templates/status.yaml` does not need migration | It's a YAML template consumed by skills (markdown), not sourced by bash scripts |

4 assumptions made (3 confident, 1 tentative). Run /fab-clarify to review.
