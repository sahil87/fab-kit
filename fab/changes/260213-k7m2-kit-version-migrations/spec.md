# Spec: Kit Version Migrations

**Change**: 260213-k7m2-kit-version-migrations
**Created**: 2026-02-14
**Affected memory**:
- `fab/memory/fab-workflow/migrations.md` (new)
- `fab/memory/fab-workflow/kit-architecture.md` (modify)
- `fab/memory/fab-workflow/init.md` (modify)
- `fab/memory/fab-workflow/distribution.md` (modify)

## Non-Goals

- Automatic rollback of failed migrations — migrations are LLM instructions, not transactional scripts; partial state is handled by stopping and reporting
- Auto-invoking `/fab-update` from `fab-upgrade.sh` — the two-step flow keeps the mechanical swap separate from intelligent migration

## Version Tracking: Dual-Version Model

### Requirement: Engine Version File

`fab/.kit/VERSION` SHALL continue to track the kit engine version as a semver string (e.g., `0.1.11`). This file ships inside `.kit/` and is replaced on each `fab-upgrade.sh` run.

#### Scenario: Engine version is readable
- **GIVEN** a project with `fab/.kit/` installed
- **WHEN** any skill or script reads `fab/.kit/VERSION`
- **THEN** it SHALL contain a valid semver string (MAJOR.MINOR.PATCH)

### Requirement: Local Project Version File

`fab/VERSION` SHALL track the local project's kit version — the version its `config.yaml`, `.status.yaml`, and conventions were written for. This file lives outside `.kit/` and is NOT replaced on upgrades.

#### Scenario: New project initialization
- **GIVEN** a project without `fab/VERSION`
- **WHEN** the user runs `/fab-init` (bootstrap)
- **THEN** `fab/VERSION` SHALL be created with the value from `fab/.kit/VERSION`

#### Scenario: Re-run of `/fab-init` with existing `fab/VERSION`
- **GIVEN** a project with `fab/VERSION` already set to `0.1.0`
- **WHEN** the user runs `/fab-init` again
- **THEN** `fab/VERSION` SHALL NOT be overwritten
- **AND** the existing version SHALL be preserved

#### Scenario: Version comparison
- **GIVEN** `fab/VERSION` contains `0.1.0` and `fab/.kit/VERSION` contains `0.2.0`
- **WHEN** `/fab-update` or `/fab-status` reads both files
- **THEN** the system SHALL detect that the local version is behind the engine version

### Requirement: Version File Format

Both `fab/VERSION` and `fab/.kit/VERSION` SHALL contain a bare semver string (e.g., `0.2.0`) with no prefix, no trailing newline beyond the single line, and no additional content.

#### Scenario: Consistent format
- **GIVEN** both version files exist
- **WHEN** read by any script or skill
- **THEN** both SHALL parse identically as `MAJOR.MINOR.PATCH` integers

## Migration System: Files and Format

### Requirement: Migration Directory

`fab/.kit/migrations/` SHALL contain migration instruction files that ship with the kit. Each file covers a version range transition — the release author decides when a migration is needed based on what changed, regardless of bump type (patch, minor, or major).

#### Scenario: Migration directory exists after install
- **GIVEN** a freshly downloaded `fab/.kit/`
- **WHEN** the user inspects the directory structure
- **THEN** `fab/.kit/migrations/` SHALL exist (even if empty for the first release)

### Requirement: Migration File Naming

Migration files SHALL be named `{FROM}-to-{TO}.md` where `{FROM}` and `{TO}` are full semver strings (`MAJOR.MINOR.PATCH`).

The FROM version defines the inclusive lower bound: any project with `fab/VERSION` >= FROM is a candidate. The TO version defines the exclusive upper bound and the version the project will be set to after migration. A migration applies when `FROM <= fab/VERSION < TO`.
<!-- clarified: range-based applicability replaces exact-version or minor-only stepping — the release author decides what range each migration covers -->

#### Scenario: Patch release with project-level changes
- **GIVEN** a patch release `0.1.1` that changes `config.yaml` schema
- **WHEN** the release author creates a migration file
- **THEN** it SHALL be named `0.1.0-to-0.1.1.md`

#### Scenario: Minor version bump
- **GIVEN** a minor release `0.2.0` that introduces new conventions
- **WHEN** the release author creates a migration file
- **THEN** it SHALL be named `0.1.1-to-0.2.0.md` (FROM = last version that had a migration)

#### Scenario: Wide range migration
- **GIVEN** several releases occurred without project-level changes, then `0.4.0` introduces schema changes
- **WHEN** the release author creates a migration file with FROM = `0.2.0` (last migration's TO)
- **THEN** it SHALL be named `0.2.0-to-0.4.0.md`
- **AND** a project on version `0.3.5` SHALL match this migration because `0.2.0 <= 0.3.5 < 0.4.0`

### Requirement: Migration File Structure

Each migration file SHALL be a markdown document with structured agent instructions following this format:

```markdown
# Migration: {FROM} to {TO}

## Summary
{One-paragraph description of what changed and why migration is needed.}

## Pre-check
{Conditions to verify before applying. Each item is a bullet describing what to check.}

## Changes
{Ordered list of changes to apply. Each item describes what to check, what to modify, and the expected result. Uses RFC 2119 keywords where appropriate.}

## Verification
{Steps to confirm the migration succeeded. Each item describes what to validate.}
```

#### Scenario: Agent reads and applies a migration
- **GIVEN** `/fab-update` reads `fab/.kit/migrations/0.1.0-to-0.2.0.md`
- **WHEN** the skill processes the file
- **THEN** it SHALL execute the Pre-check steps first
- **AND** it SHALL apply the Changes in order
- **AND** it SHALL run the Verification steps after all changes are applied

#### Scenario: Pre-check failure
- **GIVEN** a migration file with a Pre-check step that fails
- **WHEN** `/fab-update` processes the migration
- **THEN** it SHALL STOP and report which pre-check failed
- **AND** it SHALL NOT apply any Changes from that migration
- **AND** `fab/VERSION` SHALL NOT be updated

### Requirement: Pure Prompt Play Compliance

Migration files SHALL contain only markdown instructions interpretable by an LLM agent. They SHALL NOT contain executable scripts, shell commands to run blindly, or references to external tools beyond what the agent already has access to (file read/write, shell execution).
<!-- assumed: migration files are pure markdown instructions — Constitution I mandates all logic in markdown and shell; migration instructions follow the same pattern as skill files -->

#### Scenario: Any AI agent can apply migrations
- **GIVEN** a migration file following the required structure
- **WHEN** any LLM agent with file read/write and shell access processes it
- **THEN** it SHALL be able to understand and apply the migration without additional tooling

## `/fab-update` Skill: Migration Runner

### Requirement: Skill File Location

`/fab-update` SHALL be defined as a skill at `fab/.kit/skills/fab-update.md`. It SHALL follow standard skill conventions (frontmatter, context loading).

#### Scenario: Skill is discoverable
- **GIVEN** a project with `fab/.kit/` installed
- **WHEN** `_init_scaffold.sh` runs
- **THEN** a symlink SHALL be created at `.claude/skills/fab-update/SKILL.md` pointing to `fab/.kit/skills/fab-update.md`

### Requirement: Range-Based Migration Discovery

`/fab-update` SHALL compare `fab/VERSION` (local) to `fab/.kit/VERSION` (engine), then scan `fab/.kit/migrations/` to discover applicable migrations using range-based matching. A migration file `{FROM}-to-{TO}.md` applies when `FROM <= fab/VERSION < TO`.

#### Scenario: Versions are equal
- **GIVEN** `fab/VERSION` = `0.2.0` and `fab/.kit/VERSION` = `0.2.0`
- **WHEN** the user runs `/fab-update`
- **THEN** it SHALL report "Already up to date (0.2.0)."
- **AND** no migrations SHALL be applied

#### Scenario: Single migration covers current version
- **GIVEN** `fab/VERSION` = `0.1.5` and `fab/.kit/VERSION` = `0.2.0`
- **AND** `fab/.kit/migrations/0.1.0-to-0.2.0.md` exists
- **WHEN** the user runs `/fab-update`
- **THEN** the migration SHALL apply because `0.1.0 <= 0.1.5 < 0.2.0`
- **AND** after successful migration, `fab/VERSION` SHALL be set to `0.2.0` (the engine version)

#### Scenario: Version falls in the middle of a wide range
- **GIVEN** `fab/VERSION` = `3.4.0` and `fab/.kit/VERSION` = `4.3.0`
- **AND** `fab/.kit/migrations/2.1.0-to-4.3.0.md` exists
- **WHEN** the user runs `/fab-update`
- **THEN** the migration SHALL apply because `2.1.0 <= 3.4.0 < 4.3.0`
- **AND** `fab/VERSION` SHALL be set to `4.3.0`

#### Scenario: Multiple chained migrations
- **GIVEN** `fab/VERSION` = `0.1.0` and `fab/.kit/VERSION` = `0.4.0`
- **AND** migrations `0.1.0-to-0.2.0.md` and `0.2.0-to-0.4.0.md` exist
- **WHEN** the user runs `/fab-update`
- **THEN** it SHALL apply `0.1.0-to-0.2.0.md` first (because `0.1.0 <= 0.1.0 < 0.2.0`), set `fab/VERSION` to `0.2.0`
- **AND** then apply `0.2.0-to-0.4.0.md` (because `0.2.0 <= 0.2.0 < 0.4.0`), set `fab/VERSION` to `0.4.0`

#### Scenario: Gap with no migration needed
- **GIVEN** `fab/VERSION` = `0.3.0` and `fab/.kit/VERSION` = `0.5.0`
- **AND** only migration `0.4.0-to-0.5.0.md` exists (no migration covers 0.3.0)
- **WHEN** the user runs `/fab-update`
- **THEN** it SHALL skip to `0.4.0` (no migration needed for 0.3.0 → 0.4.0)
- **AND** apply `0.4.0-to-0.5.0.md`, set `fab/VERSION` to `0.5.0`
- **AND** log: "No migration needed for 0.3.0 → 0.4.0, skipping."

#### Scenario: No migrations exist at all
- **GIVEN** `fab/VERSION` = `0.1.0` and `fab/.kit/VERSION` = `0.2.0`
- **AND** `fab/.kit/migrations/` is empty
- **WHEN** the user runs `/fab-update`
- **THEN** it SHALL set `fab/VERSION` to `0.2.0` (engine version)
- **AND** report "No migrations found. fab/VERSION updated to 0.2.0."

#### Scenario: Local version ahead of engine
- **GIVEN** `fab/VERSION` = `0.3.0` and `fab/.kit/VERSION` = `0.2.0`
- **WHEN** the user runs `/fab-update`
- **THEN** it SHALL report "Local version (0.3.0) is ahead of engine version (0.2.0). This is unexpected — check your fab/.kit/ installation."

### Requirement: Non-Overlapping Migration Ranges

Migration file ranges MUST NOT overlap. Two migration files overlap if their FROM-TO ranges cover any common version (i.e., for files A and B: `A.FROM < B.TO AND B.FROM < A.TO`). `/fab-update` SHALL validate this before applying any migrations.
<!-- clarified: non-overlapping constraint — overlaps indicate authoring mistakes; validated at both apply time and release time -->

#### Scenario: Overlapping ranges detected
- **GIVEN** `fab/.kit/migrations/` contains `0.1.0-to-0.3.0.md` and `0.1.0-to-0.2.0.md`
- **WHEN** the user runs `/fab-update`
- **THEN** it SHALL STOP with: "Overlapping migration ranges detected: 0.1.0-to-0.3.0.md and 0.1.0-to-0.2.0.md. Fix the migrations directory."
- **AND** no migrations SHALL be applied

### Requirement: Migration Discovery Algorithm

`/fab-update` SHALL discover and apply migrations using this algorithm:

1. Read `fab/VERSION` (current) and `fab/.kit/VERSION` (target)
2. If current >= target: report already up to date (or ahead), stop
3. Scan `fab/.kit/migrations/` and parse FROM/TO from each filename
4. Validate no overlapping ranges (stop with error if detected)
5. Sort migrations by FROM ascending
6. **Loop**:
   a. Find the first migration where `FROM <= current < TO`
   b. If found: apply it, set current = TO, repeat from (a)
   c. If not found but a migration exists with `FROM > current`: skip to that FROM (log the skip), repeat from (a)
   d. If not found and no later migrations exist: set `fab/VERSION` to engine version, done

#### Scenario: Algorithm handles interleaved gaps and migrations
- **GIVEN** `fab/VERSION` = `0.1.0`, `fab/.kit/VERSION` = `1.0.0`
- **AND** migrations: `0.1.0-to-0.3.0.md`, `0.5.0-to-0.8.0.md`, `0.8.0-to-1.0.0.md`
- **WHEN** the user runs `/fab-update`
- **THEN** it SHALL apply `0.1.0-to-0.3.0.md` → version becomes `0.3.0`
- **AND** skip `0.3.0 → 0.5.0` (no migration, log skip)
- **AND** apply `0.5.0-to-0.8.0.md` → version becomes `0.8.0`
- **AND** apply `0.8.0-to-1.0.0.md` → version becomes `1.0.0`

### Requirement: Migration Failure Handling

If a migration fails (pre-check, change application, or verification), `/fab-update` SHALL stop immediately. It SHALL NOT proceed to subsequent migrations. `fab/VERSION` SHALL remain at its value before the failed migration.

#### Scenario: Mid-sequence failure
- **GIVEN** migrations `0.1.0-to-0.2.0.md` and `0.2.0-to-0.3.0.md` are needed
- **WHEN** `0.1.0-to-0.2.0` succeeds but `0.2.0-to-0.3.0` fails at a verification step
- **THEN** `fab/VERSION` SHALL be `0.2.0` (updated after first migration succeeded)
- **AND** the skill SHALL report which migration failed and at which step
- **AND** the skill SHALL suggest: "Fix the issue and re-run /fab-update to continue from 0.2.0."

### Requirement: Pre-flight Checks

`/fab-update` SHALL verify prerequisites before attempting any migration.

#### Scenario: `fab/VERSION` missing
- **GIVEN** a project without `fab/VERSION`
- **WHEN** the user runs `/fab-update`
- **THEN** it SHALL STOP with: "fab/VERSION not found. Run /fab-init to create it."

#### Scenario: `fab/.kit/VERSION` missing
- **GIVEN** a project without `fab/.kit/VERSION`
- **WHEN** the user runs `/fab-update`
- **THEN** it SHALL STOP with: "fab/.kit/VERSION not found — kit may be corrupted."

### Requirement: Output Format

`/fab-update` SHALL produce clear, structured output showing the migration plan and progress.

#### Scenario: Successful multi-step migration
- **GIVEN** `fab/VERSION` = `0.1.0`, engine = `0.4.0`, migrations `0.1.0-to-0.2.0.md` and `0.2.0-to-0.4.0.md` exist
- **WHEN** both migrations succeed
- **THEN** the output SHALL follow this format:
  ```
  Local version:  0.1.0
  Engine version: 0.4.0
  Migrations found: 2

  [1/2] Applying 0.1.0 → 0.2.0...
  {migration output}
  ✓ fab/VERSION updated to 0.2.0

  [2/2] Applying 0.2.0 → 0.4.0...
  {migration output}
  ✓ fab/VERSION updated to 0.4.0

  All migrations complete. fab/VERSION: 0.1.0 → 0.4.0
  ```

#### Scenario: Migration with gap skip
- **GIVEN** `fab/VERSION` = `0.1.0`, engine = `0.5.0`, only migration `0.3.0-to-0.5.0.md` exists
- **WHEN** the migration succeeds
- **THEN** the output SHALL include:
  ```
  Local version:  0.1.0
  Engine version: 0.5.0
  Migrations found: 1

  No migration needed for 0.1.0 → 0.3.0, skipping.

  [1/1] Applying 0.3.0 → 0.5.0...
  {migration output}
  ✓ fab/VERSION updated to 0.5.0

  All migrations complete. fab/VERSION: 0.1.0 → 0.5.0
  ```

## `/fab-init` Changes: VERSION Creation

### Requirement: Create `fab/VERSION` During Bootstrap

`/fab-init` SHALL create `fab/VERSION` during the structural bootstrap phase via `_init_scaffold.sh`. The version assigned depends on whether the project is new (no `config.yaml`) or existing (has `config.yaml`).

#### Scenario: First-time init (new project)
- **GIVEN** a project without `fab/VERSION` or `fab/config.yaml`, and `fab/.kit/VERSION` = `0.2.0`
- **WHEN** the user runs `/fab-init`
- **THEN** `fab/VERSION` SHALL be created with contents `0.2.0` (engine version)
- **AND** the bootstrap output SHALL include "Created: fab/VERSION (0.2.0)"

#### Scenario: First-time init (existing project)
- **GIVEN** a project without `fab/VERSION` but with `fab/config.yaml`, and `fab/.kit/VERSION` = `0.2.0`
- **WHEN** the user runs `/fab-init`
- **THEN** `fab/VERSION` SHALL be created with contents `0.1.0` (base version)
- **AND** the output SHALL include "Created: fab/VERSION (0.1.0 — existing project, run /fab-update to migrate)"

#### Scenario: Re-init with existing VERSION
- **GIVEN** `fab/VERSION` already exists with value `0.1.0`
- **WHEN** the user runs `/fab-init`
- **THEN** `fab/VERSION` SHALL NOT be modified
- **AND** the output SHALL report "fab/VERSION: OK (0.1.0)"

### Requirement: `_init_scaffold.sh` Creates `fab/VERSION`

The structural bootstrap script `_init_scaffold.sh` SHALL handle `fab/VERSION` creation, consistent with its role as the structural bootstrap owner. It SHALL distinguish new projects from existing ones to assign the correct initial version.

#### Scenario: New project (no `fab/config.yaml`)
- **GIVEN** `fab/.kit/VERSION` = `0.2.0` and no `fab/VERSION` exists and no `fab/config.yaml` exists
- **WHEN** `_init_scaffold.sh` runs
- **THEN** it SHALL copy `fab/.kit/VERSION` to `fab/VERSION` (value: `0.2.0`)
- **AND** output "Created: fab/VERSION (0.2.0)"
<!-- clarified: new projects get the engine version since /fab-init will generate config/constitution matching that version -->

#### Scenario: Existing project (has `fab/config.yaml`, no `fab/VERSION`)
- **GIVEN** `fab/.kit/VERSION` = `0.2.0` and no `fab/VERSION` exists but `fab/config.yaml` exists
- **WHEN** `_init_scaffold.sh` runs
- **THEN** it SHALL create `fab/VERSION` with the base version `0.1.0`
- **AND** output "Created: fab/VERSION (0.1.0 — existing project, run /fab-update to migrate)"
<!-- clarified: existing projects get the pre-migration base version so /fab-update runs all needed migrations -->

#### Scenario: Scaffold script preserves existing VERSION
- **GIVEN** `fab/VERSION` already exists
- **WHEN** `_init_scaffold.sh` runs
- **THEN** it SHALL NOT overwrite `fab/VERSION`

## `fab-upgrade.sh` Changes: Post-Upgrade Guidance

### Requirement: Version Drift Reminder

After a successful `.kit/` upgrade, `fab-upgrade.sh` SHALL check whether `fab/VERSION` exists and whether it differs from the new `fab/.kit/VERSION`. If `fab/VERSION` < engine version, it SHALL print a reminder to run `/fab-update`.

#### Scenario: Upgrade introduces version drift
- **GIVEN** `fab/VERSION` = `0.1.0` and the upgrade installs engine version `0.2.0`
- **WHEN** `fab-upgrade.sh` completes successfully
- **THEN** it SHALL print: "Note: fab/VERSION (0.1.0) is behind engine (0.2.0). Run /fab-update to apply migrations."

#### Scenario: Upgrade with no drift
- **GIVEN** `fab/VERSION` = `0.2.0` and the upgrade installs engine version `0.2.0`
- **WHEN** `fab-upgrade.sh` completes successfully
- **THEN** no migration reminder SHALL be printed (versions match)

#### Scenario: No `fab/VERSION` file
- **GIVEN** a project without `fab/VERSION`
- **WHEN** `fab-upgrade.sh` completes successfully
- **THEN** it SHALL print: "Note: fab/VERSION not found. Run /fab-init to create it, then /fab-update for migrations."

## `fab-release.sh` Changes: Migration Chain Validation

### Requirement: Migration Chain Continuity Check

`fab-release.sh` SHOULD warn (not block) when the new release version is not covered as the TO of any existing migration file. This helps release authors remember to create migration files when project-level changes occur.

#### Scenario: Release version is reachable via migration chain
- **GIVEN** a release bumping from `0.1.5` to `0.2.0`
- **WHEN** `fab/.kit/migrations/` contains a file with TO = `0.2.0` (e.g., `0.1.0-to-0.2.0.md`)
- **THEN** the release SHALL proceed without warning

#### Scenario: Release version has no migration targeting it
- **GIVEN** a release bumping from `0.1.5` to `0.2.0`
- **WHEN** no migration file in `fab/.kit/migrations/` has TO = `0.2.0`
- **THEN** the script SHALL print: "Note: No migration targets version 0.2.0. If this release changes project-level files, consider adding a migration."
- **AND** the release SHALL proceed (warning only, not blocking)

#### Scenario: Overlapping migration ranges detected during release
- **GIVEN** `fab/.kit/migrations/` contains files with overlapping FROM-TO ranges
- **WHEN** `fab-release.sh` runs
- **THEN** the script SHALL print: "Warning: Overlapping migration ranges detected — this will cause /fab-update to error."
- **AND** the release SHALL proceed (warning only, not blocking)

## Version Drift Display

### Requirement: Status Display Shows Version Drift

`/fab-status` (or the underlying status script) SHOULD display a version drift warning when `fab/VERSION` < `fab/.kit/VERSION`.

#### Scenario: Version drift detected
- **GIVEN** `fab/VERSION` = `0.1.0` and `fab/.kit/VERSION` = `0.2.0`
- **WHEN** the user runs `/fab-status`
- **THEN** the output SHALL include: "⚠ Version drift: local 0.1.0, engine 0.2.0 — run /fab-update"

#### Scenario: No drift
- **GIVEN** `fab/VERSION` = `0.2.0` and `fab/.kit/VERSION` = `0.2.0`
- **WHEN** the user runs `/fab-status`
- **THEN** no drift warning SHALL appear

## Design Decisions

### Two-Step Update Flow
**Decision**: `fab-upgrade.sh` handles the mechanical `.kit/` swap; `/fab-update` (skill) handles intelligent migration execution. They are separate operations.
**Why**: Migrations are LLM instruction files — they need an agent to interpret. The shell script handles download/swap (no LLM needed); the skill handles reading instructions and applying changes (LLM needed). This preserves Constitution I (Pure Prompt Play) while keeping the mechanical operation scriptable.
**Rejected**: Single combined script — would require embedding LLM invocation in a shell script, or making migration files executable scripts (violates pure prompt play).

### Range-Based Migration Applicability
**Decision**: Migration files define a FROM-TO version range. A migration applies when `FROM <= fab/VERSION < TO`. Any release (patch, minor, or major) can ship a migration file if it changes project-level files. The release author decides — the system does not impose rules based on bump type.
**Why**: Avoids hardcoding assumptions about which version types need migrations. Allows sparse migration files (no empty placeholders for releases that don't change project-level files). Supports wide-range migrations that cover multiple intermediate releases.
**Rejected**: Minor-only stepping — forced every minor release to have a migration file, even when no project-level changes occurred. Exact-version chaining — required unbroken linked list of migration files, creating maintenance burden.

### Warning-Only Release Validation
**Decision**: `fab-release.sh` warns but does not block releases without a migration file targeting the new version.
**Why**: Not every release changes project-level files. Blocking would create friction. The warning serves as a reminder for the release author.
**Rejected**: Hard block — too restrictive; empty migration files would become boilerplate.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Certain | Migration files named with full semver range (`0.1.0-to-0.2.0.md`) | Clarified: range-based model with full semver, release author decides when migrations are needed |
| 2 | Confident | Two-step update flow: `fab-upgrade.sh` (script) then `/fab-update` (skill) | Brief explicitly describes this separation; migrations need an LLM to interpret |
| 3 | Certain | Any version bump (patch/minor/major) can have a migration file | Clarified: determined by whether project-level files change, not by bump type |
| 4 | Confident | Migration files follow pure markdown instruction format (not executable scripts) | Constitution I mandates all logic in markdown and shell; migration instructions follow the same pattern as skill files |
| 5 | Confident | `_init_scaffold.sh` owns `fab/VERSION` creation (not just the skill) | Consistent with existing delegation pattern — scaffold script handles structural files, skill adds interactive parts |
| 6 | Confident | `fab-release.sh` warns (not blocks) on missing migration files | Not every release changes project-level files; blocking would create unnecessary friction |
| 7 | Certain | Existing projects get base version `0.1.0`; new projects get engine version | Clarified: `_init_scaffold.sh` checks for `config.yaml` to distinguish new vs existing projects |

7 assumptions made (4 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-14

| # | Question | Answer | Impact |
|---|----------|--------|--------|
| 1 | When an existing project (pre-migration era) runs `/fab-init`, what version should `fab/VERSION` get? | Base version `0.1.0` — ensures `/fab-update` runs all needed migrations. New projects get the engine version since their config is freshly generated. | Added new/existing project distinction to `_init_scaffold.sh` and `/fab-init` requirements. Heuristic: `config.yaml` exists → existing project. |
| 2 | Should patch versions skip migrations (only minor+ bumps)? | No — any release can have a migration file. Whether a migration is needed is determined by the release author (creation side) and by file existence (receiving side), not by version bump type. | Replaced minor-only stepping with range-based model. Migration files use full semver (`FROM-to-TO.md`). Algorithm uses `FROM <= version < TO` for applicability. Removed tentative assumption #3, added range-based design decision. |
| 3 | What if two migration files both match the current version (overlapping ranges)? | Non-overlapping constraint — migration ranges MUST NOT overlap. `/fab-update` validates before applying. `fab-release.sh` warns if overlap detected. | Added non-overlapping requirement with validation scenario. Added overlap detection to `fab-release.sh` warnings. |
