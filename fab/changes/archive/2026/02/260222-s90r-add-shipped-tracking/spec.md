# Spec: Add Shipped Tracking

**Change**: 260222-s90r-add-shipped-tracking
**Created**: 2026-02-22
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Adding a 7th pipeline stage — `shipped` is a side-band field, not a stage in the `progress` map
- Gating `/fab-archive` on shipped status — archive guard remains `hydrate: done` only
- Recording metadata per PR URL (timestamps, titles) — bare URL strings only

## stageman.sh: Shipped Subcommands

### Requirement: `ship` subcommand appends PR URL

`stageman.sh ship <file> <url>` SHALL append a PR URL string to the `shipped` array in `.status.yaml`. The command SHALL create the `shipped` key if it does not exist. The command SHALL skip the append silently if the URL already exists in the array (deduplication). The command SHALL update `last_updated` to the current ISO timestamp. The command SHALL use the atomic write pattern (temp file → `mv`) consistent with all other write subcommands.

#### Scenario: First ship call on a fresh status file

- **GIVEN** a `.status.yaml` with `shipped: []`
- **WHEN** `stageman.sh ship <file> "https://github.com/org/repo/pull/42"` is invoked
- **THEN** `.status.yaml` contains `shipped: ["https://github.com/org/repo/pull/42"]`
- **AND** `last_updated` is refreshed

#### Scenario: Append second URL

- **GIVEN** a `.status.yaml` with `shipped: ["https://github.com/org/repo/pull/42"]`
- **WHEN** `stageman.sh ship <file> "https://github.com/org/repo/pull/43"` is invoked
- **THEN** `.status.yaml` contains both URLs in order
- **AND** `last_updated` is refreshed

#### Scenario: Duplicate URL is skipped

- **GIVEN** a `.status.yaml` with `shipped: ["https://github.com/org/repo/pull/42"]`
- **WHEN** `stageman.sh ship <file> "https://github.com/org/repo/pull/42"` is invoked
- **THEN** `.status.yaml` still contains exactly one entry
- **AND** `last_updated` is refreshed
- **AND** exit code is 0 (no error)

#### Scenario: Missing shipped key

- **GIVEN** a `.status.yaml` with no `shipped` key (e.g., a pre-existing status file from before this change)
- **WHEN** `stageman.sh ship <file> <url>` is invoked
- **THEN** the `shipped` key is created with the URL as its first entry
- **AND** `last_updated` is refreshed

#### Scenario: Missing status file

- **GIVEN** a non-existent file path
- **WHEN** `stageman.sh ship <file> <url>` is invoked
- **THEN** exit code is 1
- **AND** stderr contains "ERROR: Status file not found"

### Requirement: `is-shipped` subcommand queries shipped state

`stageman.sh is-shipped <file>` SHALL exit 0 if the `shipped` array contains one or more entries, and exit 1 otherwise. The command SHALL produce no stdout output. The command SHALL treat a missing `shipped` key as empty (exit 1).

#### Scenario: Change has been shipped

- **GIVEN** a `.status.yaml` with `shipped: ["https://github.com/org/repo/pull/42"]`
- **WHEN** `stageman.sh is-shipped <file>` is invoked
- **THEN** exit code is 0
- **AND** stdout is empty

#### Scenario: Change has not been shipped

- **GIVEN** a `.status.yaml` with `shipped: []`
- **WHEN** `stageman.sh is-shipped <file>` is invoked
- **THEN** exit code is 1
- **AND** stdout is empty

#### Scenario: Missing shipped key

- **GIVEN** a `.status.yaml` with no `shipped` key
- **WHEN** `stageman.sh is-shipped <file>` is invoked
- **THEN** exit code is 1

#### Scenario: Missing status file

- **GIVEN** a non-existent file path
- **WHEN** `stageman.sh is-shipped <file>` is invoked
- **THEN** exit code is 1
- **AND** stderr contains "ERROR: Status file not found"

### Requirement: CLI dispatch and help text

The `ship` and `is-shipped` subcommands SHALL be registered in the CLI dispatch (`case` block) and documented in the `show_help` output. Help text SHALL follow the existing format: subcommand name, argument signature, and one-line description.

#### Scenario: Help text includes new subcommands

- **GIVEN** `stageman.sh --help` is invoked
- **WHEN** output is inspected
- **THEN** `ship <file> <url>` appears under Write commands
- **AND** `is-shipped <file>` appears under .status.yaml accessors

## status.yaml Template

### Requirement: Template includes shipped field

The `fab/.kit/templates/status.yaml` template SHALL include `shipped: []` as an initialized field. The field SHALL be positioned after `stage_metrics` and before `last_updated`.

#### Scenario: New change has empty shipped array

- **GIVEN** a new change is created via `changeman.sh new`
- **WHEN** `.status.yaml` is read
- **THEN** the `shipped` key exists with value `[]`

## workflow.yaml Schema

### Requirement: Schema documents shipped field

`fab/.kit/schemas/workflow.yaml` SHALL include a `shipped` section documenting the field's purpose, type (`string[]`), and semantics (append-only, managed by `stageman.sh ship`). This is descriptive documentation, not a structural change to the stage/state system.

#### Scenario: Schema describes shipped

- **GIVEN** `workflow.yaml` is read
- **WHEN** the `shipped` section is inspected
- **THEN** it describes the field as a string array of PR URLs
- **AND** it notes the field is managed by `stageman.sh ship`

## /git-pr Skill: Shipped Integration

### Requirement: Record PR URL after creation

After successful PR creation (step 3c) or when a PR already exists, `/git-pr` SHALL attempt to record the PR URL via `stageman.sh ship`. The skill SHALL resolve the active change by running `changeman.sh resolve` to obtain the change name, then derive the `.status.yaml` path as `fab/changes/{name}/.status.yaml`.

#### Scenario: PR created with active change

- **GIVEN** an active change exists (`fab/current` is set)
- **AND** `/git-pr` successfully creates a PR
- **WHEN** the skill completes step 3c
- **THEN** `stageman.sh ship` is called with the status file path and PR URL
- **AND** the `shipped` array in `.status.yaml` contains the PR URL

#### Scenario: PR already exists with active change

- **GIVEN** an active change exists
- **AND** a PR already exists for the current branch
- **WHEN** `/git-pr` detects the existing PR
- **THEN** `stageman.sh ship` is called with the existing PR URL
- **AND** the call is idempotent (no duplicate if URL already recorded)

### Requirement: Graceful degradation without active change

If `changeman.sh resolve` fails (no active change, no `fab/current`, or `fab/changes/` doesn't exist), `/git-pr` SHALL skip the `stageman.sh ship` call silently and proceed normally. The PR creation workflow MUST NOT be blocked by fab pipeline state.

#### Scenario: No active change

- **GIVEN** `fab/current` does not exist
- **WHEN** `/git-pr` runs
- **THEN** PR creation proceeds normally
- **AND** no `stageman.sh ship` call is made
- **AND** no error is displayed

#### Scenario: changeman.sh resolve fails

- **GIVEN** `changeman.sh resolve` exits non-zero (for any reason)
- **WHEN** `/git-pr` runs
- **THEN** PR creation proceeds normally
- **AND** no `stageman.sh ship` call is made

## _preamble.md: State Table Update

### Requirement: Hydrate state routes to /git-pr

The State Table in `_preamble.md` SHALL update the `hydrate` row to list `/git-pr` as the default command, with `/fab-archive` as an alternative:

| State | Available commands | Default |
|-------|-------------------|---------|
| hydrate | `/git-pr`, `/fab-archive` | `/git-pr` |

#### Scenario: Hydrate complete shows git-pr as next

- **GIVEN** a change with `hydrate: done`
- **WHEN** any skill outputs a `Next:` line
- **THEN** the line shows `/git-pr` as the default
- **AND** `/fab-archive` as an alternative

## changeman.sh: Default Command Update

### Requirement: Hydrate maps to /git-pr

The `default_command` function in `changeman.sh` SHALL map `hydrate` to `/git-pr` instead of `/fab-archive`.

#### Scenario: Switch to completed change

- **GIVEN** a change with `hydrate: done`
- **WHEN** `changeman.sh switch` outputs the Next line
- **THEN** the default command shown is `/git-pr`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Simple string array for shipped field | Confirmed from intake #1 — user explicitly chose bare URLs over structured objects | S:95 R:90 A:90 D:95 |
| 2 | Certain | Extend stageman.sh rather than separate script | Confirmed from intake #2 — single-writer principle for .status.yaml | S:90 R:85 A:90 D:90 |
| 3 | Certain | Built into /git-pr skill | Confirmed from intake #3 — natural integration point | S:90 R:90 A:85 D:95 |
| 4 | Certain | Archive guard unchanged (hydrate:done only) | Confirmed from intake #4 — user explicitly stated no dependency on shipped | S:95 R:95 A:90 D:95 |
| 5 | Certain | List by default (not single value) | Confirmed from intake #5 — supports multiple PRs per change | S:95 R:90 A:85 D:95 |
| 6 | Confident | is-shipped uses exit codes only (no stdout) | Confirmed from intake #6 — grep -q convention; consistent with CLI query patterns | S:70 R:90 A:80 D:75 |
| 7 | Confident | git-pr silently skips ship call when no active change | Confirmed from intake #7 — graceful degradation; git-pr works outside fab pipeline | S:70 R:85 A:80 D:75 |
| 8 | Confident | shipped field positioned after stage_metrics in template | Confirmed from intake #8 — logical post-pipeline grouping | S:55 R:95 A:70 D:80 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
