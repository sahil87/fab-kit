# Spec: Fix created_by Format to Use GitHub ID

**Change**: 260212-0r8e-fix-created-by-github
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`

## Non-Goals

- Updating `created_by` in existing `.status.yaml` files — historical changes retain their current values
- Validating or normalizing existing `created_by` values across the project

## fab-workflow: `/fab-new` created_by Field

### Requirement: GitHub ID as Primary Identifier

The `/fab-new` skill SHALL use `gh api user --jq .login` as the primary method for populating the `created_by` field in `.status.yaml` when creating a new change.

#### Scenario: GitHub CLI Available and Authenticated

- **GIVEN** the `gh` CLI is installed and the user is authenticated
- **WHEN** `/fab-new` initializes `.status.yaml` for a new change
- **THEN** the `created_by` field SHALL contain the GitHub username returned by `gh api user --jq .login`

#### Scenario: GitHub CLI Output Format

- **GIVEN** the `gh` CLI returns a GitHub username (e.g., `sahilahuja`)
- **WHEN** the value is written to `created_by`
- **THEN** the value SHALL be the raw login string with no additional formatting or prefix

### Requirement: Graceful Fallback to git config

The `/fab-new` skill SHALL fall back to `git config user.name` when the `gh` CLI is unavailable or fails for any reason.

#### Scenario: gh CLI Not Installed

- **GIVEN** the `gh` command is not found in the system PATH
- **WHEN** `/fab-new` initializes `.status.yaml`
- **THEN** the `created_by` field SHALL contain the value from `git config user.name`
- **AND** no error or warning SHALL be displayed to the user

#### Scenario: gh CLI Not Authenticated

- **GIVEN** the `gh` CLI is installed but the user is not authenticated (`gh auth status` would fail)
- **WHEN** `/fab-new` attempts `gh api user --jq .login`
- **THEN** the command exits non-zero
- **AND** the `created_by` field SHALL fall back to `git config user.name`
- **AND** no error or warning SHALL be displayed to the user

#### Scenario: gh API Error (Network, Rate Limit, etc.)

- **GIVEN** the `gh` CLI is installed and authenticated but the API call fails (network error, rate limit, server error)
- **WHEN** `/fab-new` attempts `gh api user --jq .login`
- **THEN** the command exits non-zero or returns empty output
- **AND** the `created_by` field SHALL fall back to `git config user.name`

#### Scenario: Both gh and git config Unavailable

- **GIVEN** `gh api user --jq .login` fails AND `git config user.name` returns empty or exits non-zero
- **WHEN** `/fab-new` initializes `.status.yaml`
- **THEN** the `created_by` field SHALL be set to `"unknown"`

### Requirement: Backward Compatibility

Existing `.status.yaml` files SHALL NOT be modified by this change. The updated behavior applies only to newly created changes going forward.

#### Scenario: Existing Change Unaffected

- **GIVEN** an existing change with `created_by: "Sahil Ahuja"` in its `.status.yaml`
- **WHEN** any Fab skill reads or processes this change
- **THEN** the `created_by` field SHALL remain `"Sahil Ahuja"` unchanged

## Deprecated Requirements

### git config user.name as Primary Identifier

**Reason**: Replaced by `gh api user --jq .login` as the primary source for `created_by`. `git config user.name` becomes the fallback.
**Migration**: Automatic — the fallback chain (`gh` → `git config` → `"unknown"`) preserves existing behavior when `gh` is unavailable.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Only future changes affected, not historical | Updating historical `.status.yaml` files is out of scope — backward-compatible, easily reversed if the user later wants a migration script |

1 assumptions made (1 confident, 0 tentative). Run /fab-clarify to review.
