# Spec: Add `created_by` Attribution to Changes

**Change**: 260211-endg-add-created-by-field
**Created**: 2026-02-11
**Affected docs**: `fab/docs/fab-workflow/templates.md`, `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/execution-skills.md`

## Templates: `.status.yaml` Schema

### Requirement: `created_by` Field in `.status.yaml`

The `.status.yaml` template SHALL include a `created_by` field that records the identity of the person who initiated the change. The field SHALL be placed immediately after the `created:` field. The field is write-once — once set at creation time, it SHALL NOT be modified by any subsequent skill invocation.

#### Scenario: New `.status.yaml` includes `created_by`
- **GIVEN** a new change is being created by `/fab-new` or `/fab-discuss`
- **WHEN** `.status.yaml` is initialized
- **THEN** the file SHALL include a `created_by:` field immediately after `created:`
- **AND** the value SHALL be the output of `git config user.name`

#### Scenario: `created_by` field format
- **GIVEN** a `.status.yaml` file with a `created_by` field
- **WHEN** the field is read by any skill
- **THEN** the value SHALL be a plain string (no quoting required unless the name contains YAML special characters)

### Requirement: Fallback When Git Config Unset

If `git config user.name` returns empty or errors, the `created_by` field SHALL be set to `"unknown"`.

#### Scenario: Git user name not configured
- **GIVEN** `git config user.name` returns an empty string or exits non-zero
- **WHEN** a new change is created
- **THEN** `created_by` SHALL be set to `"unknown"`
- **AND** change creation SHALL NOT be blocked

### Requirement: Backward Compatibility

Skills that read `created_by` SHALL tolerate its absence. Archived changes created before this feature SHALL NOT be backfilled. A missing `created_by` field SHALL be treated as "not available" — not as an error.

#### Scenario: Existing change missing `created_by`
- **GIVEN** a `.status.yaml` file that does not contain a `created_by` field
- **WHEN** `/fab-status` reads the file
- **THEN** the `Created by:` line SHALL be omitted from output (not shown as "unknown")

## Planning Skills: Change Creation

### Requirement: `/fab-new` Sets `created_by`

`/fab-new` SHALL populate the `created_by` field when initializing `.status.yaml` for a new change. The value SHALL be obtained by running `git config user.name`, with fallback to `"unknown"`.

#### Scenario: `/fab-new` creates change with attribution
- **GIVEN** a user runs `/fab-new "add login page"`
- **AND** `git config user.name` returns `"Sahil Ahuja"`
- **WHEN** `.status.yaml` is initialized
- **THEN** the file SHALL contain `created_by: Sahil Ahuja`

### Requirement: `/fab-discuss` Sets `created_by`

`/fab-discuss` SHALL populate the `created_by` field when initializing `.status.yaml` for a new change (new change mode only). The value SHALL be obtained by running `git config user.name`, with fallback to `"unknown"`. In refine mode, the existing `created_by` value SHALL NOT be modified.

#### Scenario: `/fab-discuss` new change mode sets attribution
- **GIVEN** a user runs `/fab-discuss` in new change mode
- **AND** `git config user.name` returns `"Sahil Ahuja"`
- **WHEN** `.status.yaml` is initialized
- **THEN** the file SHALL contain `created_by: Sahil Ahuja`

#### Scenario: `/fab-discuss` refine mode preserves attribution
- **GIVEN** a user runs `/fab-discuss` in refine mode on an existing change
- **AND** the existing `.status.yaml` contains `created_by: Jane Doe`
- **WHEN** `.status.yaml` is updated
- **THEN** `created_by` SHALL remain `Jane Doe`

## Execution Skills: Status Display

### Requirement: `/fab-status` Displays `created_by`

The `fab-status.sh` script SHALL read the `created_by` field from `.status.yaml` and display it in the status output. The display line SHALL appear after `Change:` and before `Branch:` (or before `Stage:` when git is disabled).
<!-- assumed: Display placement after Change: line — follows .status.yaml field ordering, consistent with other metadata lines -->

#### Scenario: Status output includes `created_by`
- **GIVEN** a `.status.yaml` with `created_by: Sahil Ahuja`
- **WHEN** the user runs `/fab-status`
- **THEN** the output SHALL include `Created by: Sahil Ahuja` between the `Change:` and `Branch:` lines

#### Scenario: Status output omits `created_by` when missing
- **GIVEN** a `.status.yaml` without a `created_by` field
- **WHEN** the user runs `/fab-status`
- **THEN** the `Created by:` line SHALL be omitted entirely

## Deprecated Requirements

(none)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Display placement: `Created by:` line between `Change:` and `Branch:` | Follows `.status.yaml` field ordering; consistent with existing metadata display pattern |
