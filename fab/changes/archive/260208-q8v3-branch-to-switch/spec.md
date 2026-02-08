# Spec: Move Branch Integration from fab-new to fab-switch

**Change**: 260208-q8v3-branch-to-switch
**Created**: 2026-02-09
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/change-lifecycle.md`

## fab-switch: Branch Integration

### Requirement: fab-switch SHALL offer branch integration after writing fab/current

When `git.enabled` is `true` in `fab/config.yaml` and the working directory is inside a git repository, `/fab-switch` SHALL perform branch integration after writing the change name to `fab/current`.

#### Scenario: Switch while on main/master

- **GIVEN** `git.enabled` is `true` in config
- **AND** the current git branch is `main` or `master`
- **WHEN** `/fab-switch` activates a change
- **THEN** a new branch SHALL be auto-created named `{branch_prefix}{change-name}` using `git.branch_prefix` from config
- **AND** no user prompt SHALL be shown (Certain grade — SRAD: high R, high A, high D)

#### Scenario: Switch while on a feature branch (not wt/*)

- **GIVEN** `git.enabled` is `true` in config
- **AND** the current git branch is not `main`, `master`, or a `wt/*` branch
- **WHEN** `/fab-switch` activates a change
- **THEN** the user SHALL be presented with three options: **Adopt this branch**, **Create new branch**, **Skip**
- **AND** "Adopt this branch" SHALL be the default

#### Scenario: Switch while on a wt/* branch

- **GIVEN** `git.enabled` is `true` in config
- **AND** the current git branch matches `wt/*`
- **WHEN** `/fab-switch` activates a change
- **THEN** the user SHALL be presented with the same three options
- **AND** "Create new branch" SHALL be the default (wt branches are worktree base branches, not feature branches)

#### Scenario: Explicit --branch flag

- **GIVEN** `git.enabled` is `true` in config
- **WHEN** `/fab-switch --branch <name>` is invoked
- **THEN** the specified branch SHALL be used directly (created if new, checked out if existing)
- **AND** the interactive prompt SHALL be skipped

#### Scenario: Git disabled or not a git repo

- **GIVEN** `git.enabled` is `false` OR the working directory is not a git repo
- **WHEN** `/fab-switch` activates a change
- **THEN** branch integration SHALL be skipped entirely

#### Scenario: Branch creation fails

- **GIVEN** branch integration is attempted
- **WHEN** the git operation fails (e.g., invalid branch name, checkout conflict)
- **THEN** the error SHALL be reported to the user
- **AND** the switch SHALL still complete (fab/current is written, status is displayed)
- **AND** no branch information is recorded

### Requirement: fab-switch SHALL load config.yaml for git settings

`/fab-switch` SHALL read `fab/config.yaml` to check `git.enabled` and `git.branch_prefix`. This changes `/fab-switch` from a "minimal context" skill to one that loads config.

#### Scenario: Config loading for branch integration

- **GIVEN** `/fab-switch` is invoked
- **WHEN** the skill loads context
- **THEN** it SHALL read `fab/config.yaml` for `git.enabled` and `git.branch_prefix`
- **AND** it SHALL still NOT load `fab/constitution.md` (not needed for switch behavior)

### Requirement: fab-switch SHALL accept --branch flag

`/fab-switch` SHALL accept an optional `--branch <name>` argument that specifies an explicit branch name, bypassing the interactive prompt.

#### Scenario: --branch with new branch name

- **GIVEN** `/fab-switch --branch feature/my-branch` is invoked
- **AND** no git branch named `feature/my-branch` exists
- **WHEN** the switch completes
- **THEN** a new branch `feature/my-branch` SHALL be created via `git checkout -b`

#### Scenario: --branch with existing branch name

- **GIVEN** `/fab-switch --branch feature/my-branch` is invoked
- **AND** a git branch named `feature/my-branch` already exists
- **WHEN** the switch completes
- **THEN** the existing branch SHALL be checked out (if not already on it)

## fab-new: Remove Branch Integration

### Requirement: fab-new SHALL NOT perform branch integration

`/fab-new` SHALL no longer create, adopt, or prompt about git branches. Step 4 (Git Integration) SHALL be removed entirely. `/fab-new` SHALL call `/fab-switch` internally after generating the proposal to activate the change.

#### Scenario: fab-new creates change and delegates to fab-switch

- **GIVEN** a user invokes `/fab-new <description>`
- **WHEN** the proposal is generated
- **THEN** `/fab-new` SHALL invoke the `/fab-switch` flow internally to activate the change (writing `fab/current` and performing branch integration)
- **AND** the user experience SHALL be transparent — from their perspective, `/fab-new` still results in an active change with a branch

#### Scenario: fab-new no longer accepts --branch flag

- **GIVEN** `/fab-new` is invoked
- **WHEN** arguments are parsed
- **THEN** the `--branch` flag SHALL NOT be recognized by `/fab-new`
- **AND** users wanting explicit branch names SHALL use `/fab-switch --branch <name>` after `/fab-new`

### Requirement: fab-new SHALL NOT write branch to .status.yaml

`/fab-new` SHALL no longer include a `branch:` field in the `.status.yaml` it creates. The template and initialization logic SHALL omit this field.

#### Scenario: .status.yaml created without branch field

- **GIVEN** `/fab-new` creates a new change
- **WHEN** `.status.yaml` is initialized
- **THEN** no `branch:` field SHALL be present in the file

## .status.yaml: Remove branch field

### Requirement: branch field SHALL be removed from .status.yaml

The `branch:` field SHALL be removed from `.status.yaml`. No skill SHALL write a `branch:` field to `.status.yaml`. The `branch:` field in the status template SHALL be removed.

#### Scenario: Template no longer contains branch

- **GIVEN** `fab/.kit/templates/status.yaml` is the template for new changes
- **WHEN** the template is read
- **THEN** it SHALL NOT contain a `branch:` line

#### Scenario: Existing archived changes are unaffected

- **GIVEN** archived changes in `fab/changes/archive/` may have `branch:` in their `.status.yaml`
- **WHEN** any skill reads archived change status
- **THEN** the presence of a `branch:` field SHALL NOT cause errors (graceful handling)

## fab-status: Live git branch display

### Requirement: fab-status SHALL use git branch --show-current for branch display

`/fab-status` (and its backing script `fab-status.sh`) SHALL replace the `.status.yaml` `branch:` field lookup with a live `git branch --show-current` call. The branch SHALL only be displayed when `git.enabled` is `true` and the working directory is inside a git repo.

#### Scenario: Display live branch in git-enabled project

- **GIVEN** `git.enabled` is `true` in config
- **AND** the working directory is inside a git repo
- **WHEN** `/fab-status` runs
- **THEN** the Branch line SHALL show the output of `git branch --show-current`

#### Scenario: No branch display when git disabled

- **GIVEN** `git.enabled` is `false`
- **WHEN** `/fab-status` runs
- **THEN** the Branch line SHALL be omitted entirely (not shown as "(none)")

#### Scenario: No branch display outside git repo

- **GIVEN** the working directory is not inside a git repo
- **WHEN** `/fab-status` runs
- **THEN** the Branch line SHALL be omitted entirely

## fab-preflight.sh: Remove branch from YAML output

### Requirement: fab-preflight.sh SHALL NOT emit branch in YAML output

The preflight script SHALL remove the `branch` field from its stdout YAML output. It SHALL no longer parse the `branch:` field from `.status.yaml`.

#### Scenario: Preflight YAML output without branch

- **GIVEN** the preflight script runs successfully
- **WHEN** it emits structured YAML to stdout
- **THEN** the output SHALL NOT contain a `branch:` line

## _context.md: Remove branch from preflight fields list

### Requirement: _context.md SHALL remove branch from preflight YAML fields

The shared context preamble SHALL update its documentation of preflight output fields to remove `branch` from the list. The "Change Context" section SHALL list only `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence` as fields parsed from preflight.

#### Scenario: Updated preflight field documentation

- **GIVEN** a developer reads `_context.md` for preflight output format
- **WHEN** they check the "Parse stdout YAML" step
- **THEN** `branch` SHALL NOT be listed among the parsed fields

## fab-discuss: Remove branch field references

### Requirement: fab-discuss SHALL not reference branch field omission

`/fab-discuss` currently notes "no `branch:` field — git integration is deferred to `/fab-switch` or `/fab-new`" when describing its `.status.yaml` output. Since no skill writes `branch:` anymore, this note SHALL be removed. The `.status.yaml` example in `/fab-discuss` SHALL simply omit the `branch:` line without comment.

#### Scenario: fab-discuss .status.yaml example without branch note

- **GIVEN** the `/fab-discuss` skill definition includes a `.status.yaml` example
- **WHEN** a developer reads the skill
- **THEN** no `branch:` field or commentary about branch omission SHALL be present

## config.yaml: branch_prefix remains under git section

### Requirement: branch_prefix SHALL remain in config.yaml under git section

The `git.branch_prefix` setting SHALL remain in `fab/config.yaml` since it is a project-level naming convention. It is now consumed by `/fab-switch` instead of `/fab-new`.

#### Scenario: branch_prefix consumed by fab-switch

- **GIVEN** `git.branch_prefix` is set to `feature/` in config
- **AND** `/fab-switch` auto-creates a branch on main
- **WHEN** the branch is named
- **THEN** the name SHALL be `feature/{change-name}`

## Deprecated Requirements

### Branch field in .status.yaml
**Reason**: The `branch:` field was purely ceremonial — no skill consumed it for logic, and it went stale if the user manually switched branches. Live git queries are more accurate.
**Migration**: `/fab-status` uses `git branch --show-current` for display. No other skill needs branch information.

### --branch flag on /fab-new
**Reason**: Branch integration moved to `/fab-switch`, which now accepts `--branch <name>`.
**Migration**: Use `/fab-switch --branch <name>` after `/fab-new`, or let `/fab-switch` handle it interactively when called internally by `/fab-new`.

### Step 4 (Git Integration) in /fab-new
**Reason**: Branch handling consolidated into `/fab-switch` for consistency between `/fab-new` and `/fab-discuss` entry points.
**Migration**: `/fab-new` calls `/fab-switch` internally, which handles branch integration.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | fab-status.sh reads config.yaml for git.enabled to decide whether to show branch | The script currently only reads .status.yaml; this adds a config read, but it's minimal and consistent with the proposal |
| 2 | Confident | fab-new still calls fab-switch internally rather than requiring user to call it separately | Proposal explicitly states this; preserves existing UX where /fab-new results in an active change |
| 3 | Confident | The _context.md Skill-Specific Autonomy table removes "0 for branch-on-main" from fab-new's interruption budget | Branch-on-main is no longer a fab-new concern; the row simplifies |

3 assumptions made (3 confident, 0 tentative).
