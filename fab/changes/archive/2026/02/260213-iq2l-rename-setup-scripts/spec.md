# Spec: Rename and Reorganize Bootstrap Scripts

**Change**: 260213-iq2l-rename-setup-scripts
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/kit-architecture.md`, `fab/docs/fab-workflow/distribution.md`, `fab/docs/fab-workflow/init.md`, `fab/docs/fab-workflow/model-tiers.md`, `fab/docs/fab-workflow/hydrate.md`, `fab/docs/fab-workflow/index.md`

## Non-Goals

- Modifying archived changes — archives are historical records and SHALL NOT be touched
- Changing the functional behavior of any script — this is a pure rename/reorganize, not a refactor
- Restructuring the `scripts/` directory layout (e.g., adding a `lib/` subdirectory)

## Scripts: File Renames

### Requirement: Rename fab-setup.sh to _fab-scaffold.sh

The bootstrap script `fab/.kit/scripts/fab-setup.sh` SHALL be renamed to `fab/.kit/scripts/_fab-scaffold.sh`. The underscore prefix signals that this script is internal plumbing, not user-facing — consistent with the `_context.md` naming convention already used in skills. The name "scaffold" replaces the synonymous "setup" to break confusion with "init".

#### Scenario: Script file renamed
- **GIVEN** the file `fab/.kit/scripts/fab-setup.sh` exists
- **WHEN** the rename is applied
- **THEN** the file SHALL exist at `fab/.kit/scripts/_fab-scaffold.sh`
- **AND** `fab/.kit/scripts/fab-setup.sh` SHALL no longer exist

#### Scenario: Script content preserved
- **GIVEN** `_fab-scaffold.sh` has been renamed from `fab-setup.sh`
- **WHEN** the script is executed
- **THEN** it SHALL produce identical behavior to the original `fab-setup.sh`

### Requirement: Rename fab-update.sh to fab-upgrade.sh

The update script `fab/.kit/scripts/fab-update.sh` SHALL be renamed to `fab/.kit/scripts/fab-upgrade.sh`. "Upgrade" better describes the action (fetch a new kit version) and matches developer vocabulary (npm upgrade, brew upgrade).

#### Scenario: Script file renamed
- **GIVEN** the file `fab/.kit/scripts/fab-update.sh` exists
- **WHEN** the rename is applied
- **THEN** the file SHALL exist at `fab/.kit/scripts/fab-upgrade.sh`
- **AND** `fab/.kit/scripts/fab-update.sh` SHALL no longer exist

#### Scenario: Script content updated
- **GIVEN** `fab-upgrade.sh` exists (renamed from `fab-update.sh`)
- **WHEN** the script internally calls `fab-setup.sh` (e.g., for symlink repair after update)
- **THEN** that internal reference SHALL be updated to `_fab-scaffold.sh`

### Requirement: Rename worktree init script

The worktree init script `fab/.kit/worktree-init-common/2-rerun-fab-setup.sh` SHALL be renamed to `fab/.kit/worktree-init-common/2-rerun-fab-scaffold.sh`.

#### Scenario: Worktree script renamed
- **GIVEN** `fab/.kit/worktree-init-common/2-rerun-fab-setup.sh` exists
- **WHEN** the rename is applied
- **THEN** the file SHALL exist at `fab/.kit/worktree-init-common/2-rerun-fab-scaffold.sh`
- **AND** the old file SHALL no longer exist

#### Scenario: Worktree script content updated
- **GIVEN** `2-rerun-fab-scaffold.sh` calls `fab-setup.sh` internally
- **WHEN** the rename is applied
- **THEN** the internal reference SHALL be updated to `_fab-scaffold.sh`

## Callers: Internal Reference Updates

### Requirement: Update fab-init.md skill references

The skill file `fab/.kit/skills/fab-init.md` SHALL have all references to `fab-setup.sh` updated to `_fab-scaffold.sh`.

#### Scenario: Skill references updated
- **GIVEN** `fab/.kit/skills/fab-init.md` contains references to `fab-setup.sh`
- **WHEN** the rename is applied
- **THEN** all occurrences of `fab-setup.sh` SHALL be replaced with `_fab-scaffold.sh`
- **AND** the skill's delegation behavior SHALL remain functionally identical

### Requirement: Update fab-upgrade.sh internal references

The renamed `fab-upgrade.sh` script SHALL reference `_fab-scaffold.sh` instead of `fab-setup.sh` in any internal calls (e.g., post-update symlink repair).

#### Scenario: Internal call updated
- **GIVEN** `fab-upgrade.sh` calls the scaffold script for symlink repair
- **WHEN** the upgrade process runs
- **THEN** it SHALL invoke `_fab-scaffold.sh` (not `fab-setup.sh`)

### Requirement: Update model-tiers.yaml if applicable

If `fab/.kit/model-tiers.yaml` references `fab-setup.sh` or `fab-update.sh`, those references SHALL be updated to the new names.

#### Scenario: Model tiers checked
- **GIVEN** `fab/.kit/model-tiers.yaml` exists
- **WHEN** the rename is applied
- **THEN** any references to old script names SHALL be updated to new names

### Requirement: Update fab-help.sh if applicable

If `fab/.kit/scripts/fab-help.sh` references `fab-setup.sh` or `fab-update.sh`, those references SHALL be updated.

#### Scenario: Help script checked
- **GIVEN** `fab/.kit/scripts/fab-help.sh` exists
- **WHEN** the rename is applied
- **THEN** any references to old script names SHALL be updated to new names

### Requirement: Update README.md

`README.md` at the project root SHALL have all references to `fab-setup.sh` updated to `_fab-scaffold.sh` and all references to `fab-update.sh` updated to `fab-upgrade.sh`. Installation and upgrade instructions MUST reflect the new names.

#### Scenario: Installation instructions updated
- **GIVEN** README.md contains `fab-setup.sh` in bootstrap instructions
- **WHEN** the rename is applied
- **THEN** the instructions SHALL reference `_fab-scaffold.sh`

#### Scenario: Upgrade instructions updated
- **GIVEN** README.md contains `fab-update.sh` in upgrade instructions
- **WHEN** the rename is applied
- **THEN** the instructions SHALL reference `fab-upgrade.sh`

## Docs: Cross-Reference Updates

### Requirement: Update centralized docs

All centralized docs in `fab/docs/` that reference `fab-setup.sh` or `fab-update.sh` SHALL be updated to the new names. This includes but is not limited to:

- `fab/docs/fab-workflow/kit-architecture.md` — directory listing, script descriptions, bootstrap sequence
- `fab/docs/fab-workflow/distribution.md` — bootstrap and update instructions
- `fab/docs/fab-workflow/init.md` — delegation pattern references
- `fab/docs/fab-workflow/model-tiers.md` — deployment script references
- `fab/docs/fab-workflow/hydrate.md` — any setup script references
- `fab/docs/fab-workflow/index.md` — doc descriptions mentioning script names

#### Scenario: kit-architecture.md updated
- **GIVEN** `kit-architecture.md` lists `fab-setup.sh` in the directory tree and script descriptions
- **WHEN** the rename is applied
- **THEN** all occurrences SHALL be updated: `fab-setup.sh` → `_fab-scaffold.sh`, `fab-update.sh` → `fab-upgrade.sh`
- **AND** script description text SHALL reflect the new names and purpose

#### Scenario: distribution.md updated
- **GIVEN** `distribution.md` references `fab-setup.sh` in bootstrap steps and `fab-update.sh` in update sections
- **WHEN** the rename is applied
- **THEN** all occurrences SHALL be updated to the new names

#### Scenario: init.md updated
- **GIVEN** `init.md` references `fab-setup.sh` in its delegation pattern
- **WHEN** the rename is applied
- **THEN** all occurrences SHALL be updated to `_fab-scaffold.sh`

#### Scenario: Remaining docs checked
- **GIVEN** any other doc in `fab/docs/fab-workflow/` references old script names
- **WHEN** the rename is applied
- **THEN** those references SHALL be updated

### Requirement: Update design docs

Design docs in `fab/design/` that reference the old script names SHALL be updated.

#### Scenario: Design docs updated
- **GIVEN** `fab/design/architecture.md` or `fab/design/glossary.md` references old script names
- **WHEN** the rename is applied
- **THEN** those references SHALL be updated to new names

## Design Decisions

1. **Underscore prefix for internal scripts**: `_fab-scaffold.sh` uses underscore prefix rather than a `lib/` subdirectory
   - *Why*: Consistent with existing `_context.md` convention in skills; avoids restructuring the `scripts/` directory
   - *Rejected*: `lib/` subdirectory — would change directory structure and require updating all path references more broadly
<!-- clarified: Underscore prefix convention confirmed — matches existing _context.md pattern in skills/ -->

## Deprecated Requirements

### `fab-setup.sh` as a named script
**Reason**: Renamed to `_fab-scaffold.sh` to signal internal-only usage and eliminate confusion with `/fab-init`
**Migration**: All callers updated to reference `_fab-scaffold.sh`

### `fab-update.sh` as a named script
**Reason**: Renamed to `fab-upgrade.sh` to better match developer vocabulary
**Migration**: All callers and docs updated to reference `fab-upgrade.sh`

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | `fab-upgrade.sh` over alternatives (e.g., keeping `fab-update.sh`) | "Upgrade" matches developer vocabulary (npm/brew); user explicitly stated preference in brief |
| 2 | Tentative | Underscore prefix (`_fab-scaffold.sh`) rather than `lib/` subdirectory | Matches existing `_context.md` convention; avoids restructuring `scripts/` directory |

2 assumptions made (1 confident, 1 tentative). Run /fab-clarify to review.
