# Spec: Consolidate Status Field Naming

**Change**: 260227-gasp-consolidate-status-field-naming
**Created**: 2026-02-27
**Affected memory**: `docs/memory/fab-workflow/templates.md`, `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/pipeline-orchestrator.md`, `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/configuration.md`

## Non-Goals

- Migrating archived changes — old field names in `fab/changes/archive/` are left as-is
- Adding new capabilities beyond rename + array conversion — no new query semantics
- Changing `changeman.sh` — it does not own `.status.yaml` field mutations

## Status Template: Field Renames

### Requirement: issues field replaces issue_id

The `.status.yaml` template (`fab/.kit/templates/status.yaml`) SHALL replace `issue_id: null` with `issues: []`. The `issues` field is a YAML array of external tracker ID strings (e.g., `"DEV-123"`). It SHALL be initialized as an empty array for new changes.

#### Scenario: New change with no tracker
- **GIVEN** a new change is created via `/fab-new` with natural language input
- **WHEN** `.status.yaml` is initialized from the template
- **THEN** the `issues` field SHALL be `[]`
- **AND** no `issue_id` field SHALL exist

#### Scenario: New change from Linear ticket
- **GIVEN** a new change is created via `/fab-new` from a Linear ticket `DEV-988`
- **WHEN** `.status.yaml` is initialized
- **THEN** `/fab-new` SHALL call `stageman.sh add-issue <change> DEV-988`
- **AND** the `issues` field SHALL be `["DEV-988"]`

### Requirement: prs field replaces shipped

The `.status.yaml` template SHALL replace `shipped: []` with `prs: []`. The `prs` field is a YAML array of PR URL strings. It SHALL be initialized as an empty array for new changes.

#### Scenario: Template initialization
- **GIVEN** a new change is created
- **WHEN** `.status.yaml` is initialized from the template
- **THEN** the `prs` field SHALL be `[]`
- **AND** no `shipped` field SHALL exist

## Workflow Schema: Field Declarations

### Requirement: workflow.yaml declares issues and prs

`fab/.kit/schemas/workflow.yaml` SHALL replace the `shipped:` section with two field declarations:

```yaml
issues:
  type: "string[]"
  description: "External tracker IDs associated with this change (e.g., DEV-123). Append-only."
  initial_value: "[]"
  managed_by: "stageman.sh add-issue / get-issues"

prs:
  type: "string[]"
  description: "PR URLs associated with this change. Append-only list of bare URL strings."
  initial_value: "[]"
  managed_by: "stageman.sh add-pr / get-prs"
```

#### Scenario: Schema reflects new field names
- **GIVEN** `fab/.kit/schemas/workflow.yaml` is read
- **WHEN** a consumer looks up side-band field declarations
- **THEN** `issues` and `prs` sections SHALL exist
- **AND** no `shipped` section SHALL exist

## Stageman: Symmetric Array Operations

### Requirement: add_issue and get_issues functions

`stageman.sh` SHALL provide `add_issue()` and `get_issues()` internal functions, exposed as `add-issue` and `get-issues` CLI subcommands.

`add_issue(status_file, id)`:
- SHALL append `id` to the `issues` YAML array
- SHALL skip silently if `id` already exists in the array (idempotent, exact-match dedup via `grep -qxF`)
- SHALL update `last_updated`
- SHALL use the atomic write pattern (tmpfile copy → `yq -i` → `mv`)

`get_issues(status_file)`:
- SHALL emit each issue ID on its own line via `yq '.issues // [] | .[]'`
- SHALL produce empty output when the array is empty or missing

#### Scenario: Adding a new issue ID
- **GIVEN** a `.status.yaml` with `issues: []`
- **WHEN** `stageman.sh add-issue <change> DEV-123` is called
- **THEN** `issues` SHALL be `["DEV-123"]`
- **AND** `last_updated` SHALL be refreshed

#### Scenario: Deduplication
- **GIVEN** a `.status.yaml` with `issues: ["DEV-123"]`
- **WHEN** `stageman.sh add-issue <change> DEV-123` is called
- **THEN** `issues` SHALL remain `["DEV-123"]` (no duplicate)
- **AND** `last_updated` SHALL be refreshed

#### Scenario: Multiple issues
- **GIVEN** a `.status.yaml` with `issues: ["DEV-123"]`
- **WHEN** `stageman.sh add-issue <change> DEV-456` is called
- **THEN** `issues` SHALL be `["DEV-123", "DEV-456"]`

#### Scenario: Getting issues from empty array
- **GIVEN** a `.status.yaml` with `issues: []`
- **WHEN** `stageman.sh get-issues <change>` is called
- **THEN** stdout SHALL be empty (exit 0)

#### Scenario: Getting issues from populated array
- **GIVEN** a `.status.yaml` with `issues: ["DEV-123", "DEV-456"]`
- **WHEN** `stageman.sh get-issues <change>` is called
- **THEN** stdout SHALL contain `DEV-123` and `DEV-456`, one per line

### Requirement: add_pr and get_prs functions

`stageman.sh` SHALL provide `add_pr()` and `get_prs()` internal functions, exposed as `add-pr` and `get-prs` CLI subcommands. These replace `ship_url()` / `is_shipped()` and their `ship` / `is-shipped` CLI routes.

`add_pr(status_file, url)`:
- SHALL append `url` to the `prs` YAML array
- SHALL skip silently if `url` already exists (idempotent, exact-match dedup)
- SHALL update `last_updated`
- SHALL use the atomic write pattern

`get_prs(status_file)`:
- SHALL emit each PR URL on its own line via `yq '.prs // [] | .[]'`
- SHALL produce empty output when the array is empty or missing

#### Scenario: Adding a PR URL
- **GIVEN** a `.status.yaml` with `prs: []`
- **WHEN** `stageman.sh add-pr <change> https://github.com/org/repo/pull/42` is called
- **THEN** `prs` SHALL contain the URL
- **AND** `last_updated` SHALL be refreshed

#### Scenario: Getting PRs (replaces is-shipped boolean)
- **GIVEN** a `.status.yaml` with `prs: ["https://github.com/org/repo/pull/42"]`
- **WHEN** `stageman.sh get-prs <change>` is called
- **THEN** stdout SHALL contain the URL
- **AND** callers determine "has PRs" by checking for non-empty output

### Requirement: Remove old functions and CLI routes

`stageman.sh` SHALL remove:
- Internal functions: `ship_url()`, `is_shipped()`
- CLI routes: `ship`, `is-shipped`
- Section header comment: `# Shipped Tracking` → `# Issues & PRs`
- Help text entries for `ship` and `is-shipped`

The help text SHALL add:
```
add-issue <change> <id>            Append issue ID to issues array (idempotent)
get-issues <change>                List issue IDs (one per line)
add-pr <change> <url>              Append PR URL to prs array (idempotent)
get-prs <change>                   List PR URLs (one per line)
```

#### Scenario: Old CLI routes rejected
- **GIVEN** the updated `stageman.sh`
- **WHEN** `stageman.sh ship <change> <url>` is called
- **THEN** it SHALL exit with an unknown-subcommand error

## Skills: Stageman API Enforcement

### Requirement: fab-new uses stageman for issue IDs

`fab/.kit/skills/fab-new.md` Step 3 SHALL replace the raw `yq` call:
```
yq -i '.issue_id = "DEV-988"' fab/changes/{name}/.status.yaml
```
with:
```
fab/.kit/scripts/lib/stageman.sh add-issue fab/changes/{name}/.status.yaml DEV-988
```

#### Scenario: Linear ticket creates change with issue
- **GIVEN** `/fab-new DEV-988` is invoked
- **WHEN** the change folder is created
- **THEN** `/fab-new` SHALL call `stageman.sh add-issue` (not raw `yq`)

### Requirement: git-pr reads issues via stageman

`fab/.kit/skills/git-pr.md` SHALL replace direct `.status.yaml` reads of `issue_id` with `stageman.sh get-issues <change>`.

- Step 1: Run `stageman.sh get-issues <status_file>` and capture output into a variable
- Step 3c PR title: If issues output is non-empty, join with space and prepend to title: `{type}: {issues} {title}` (e.g., `feat: DEV-123 DEV-456 Add OAuth support`). If empty, omit: `{type}: {title}`.
- All references to `issue_id` in the skill SHALL be replaced with `issues`

#### Scenario: PR title with multiple issues
- **GIVEN** a change with `issues: ["DEV-123", "DEV-456"]` and type `feat`
- **WHEN** `/git-pr` generates the PR title
- **THEN** the title SHALL be `feat: DEV-123 DEV-456 {title}`

#### Scenario: PR title with no issues
- **GIVEN** a change with `issues: []` and type `fix`
- **WHEN** `/git-pr` generates the PR title
- **THEN** the title SHALL be `fix: {title}` (no issue prefix)

### Requirement: git-pr uses add-pr instead of ship

`fab/.kit/skills/git-pr.md` Step 4a SHALL replace `stageman.sh ship <status_file> <pr_url>` with `stageman.sh add-pr <status_file> <pr_url>`.

Step 4b commit message SHALL change from `"Record shipped URL in .status.yaml"` to `"Record PR URL in .status.yaml"`.

#### Scenario: PR URL recorded after creation
- **GIVEN** `/git-pr` successfully creates a PR
- **WHEN** Step 4a records the URL
- **THEN** it SHALL call `stageman.sh add-pr` (not `stageman.sh ship`)

## Sentinel File Rename

### Requirement: .shipped sentinel becomes .pr-done

The `.shipped` sentinel file (gitignored, written by `/git-pr` after all git ops complete) SHALL be renamed to `.pr-done` across all consumers:

1. `fab/.kit/skills/git-pr.md` — write `echo "$PR_URL" > "$change_dir/.pr-done"`
2. `fab/.kit/scripts/pipeline/run.sh` — check `local pr_sentinel="$wt_path/fab/changes/$resolved_id/.pr-done"` and update log message to `"Done: $resolved_id — pr complete"`
3. `fab/.kit/skills/fab-archive.md` — Step 1 cleanup references `.pr-done`
4. `.gitignore` (repo root) — `fab/changes/**/.shipped` → `fab/changes/**/.pr-done`
5. `fab/.kit/scaffold/fragment-.gitignore` — same pattern update

#### Scenario: Pipeline runner detects completion
- **GIVEN** `/git-pr` has written `.pr-done` in a change folder
- **WHEN** `pipeline/run.sh` polls for completion
- **THEN** it SHALL check for `.pr-done` (not `.shipped`)

#### Scenario: Archive cleans up sentinel
- **GIVEN** a change folder contains `.pr-done`
- **WHEN** `/fab-archive` runs Step 1
- **THEN** it SHALL delete `.pr-done` (not `.shipped`)

## Naming Spec Updates

### Requirement: naming.md reflects new field names

`docs/specs/naming.md` SHALL update:
- PR title pattern: `{type}: {issue_id} {title}` → `{type}: {issues} {title}` (space-joined)
- Backlog entry pattern: `[{issue_id}]` → `[{issue_ids}]` (optional, may contain multiple)
- All prose references from `issue_id` to `issues`

#### Scenario: Naming spec consistency
- **GIVEN** a reader consults `docs/specs/naming.md`
- **WHEN** they look up the PR title pattern
- **THEN** the pattern SHALL reference `{issues}` (plural, space-joined)

## Migration

### Requirement: Migration for active changes

A migration file SHALL be created at `fab/.kit/migrations/{from}-to-{to}.md` to update active (non-archived) changes' `.status.yaml` files.

The migration SHALL:
1. Find all `fab/changes/*/.status.yaml` files (excluding `fab/changes/archive/`)
2. For each file:
   - If `issue_id` exists and is non-null: set `issues` to a single-element array `["{value}"]`
   - If `issue_id` exists and is null: set `issues` to `[]`
   - If `issue_id` does not exist: set `issues` to `[]`
   - Rename `shipped` to `prs` (preserve array contents)
   - Remove `issue_id` and `shipped` fields
3. Rename any `fab/changes/*/.shipped` sentinel files to `.pr-done`
4. Bump `fab/.kit-migration-version`

#### Scenario: Scalar issue_id migrated to array
- **GIVEN** a `.status.yaml` with `issue_id: "DEV-123"` and `shipped: ["https://..."]`
- **WHEN** the migration runs
- **THEN** the file SHALL have `issues: ["DEV-123"]` and `prs: ["https://..."]`
- **AND** no `issue_id` or `shipped` fields SHALL remain

#### Scenario: Null issue_id migrated
- **GIVEN** a `.status.yaml` with `issue_id: null` and `shipped: []`
- **WHEN** the migration runs
- **THEN** the file SHALL have `issues: []` and `prs: []`

#### Scenario: No active changes
- **GIVEN** no files exist in `fab/changes/` (excluding archive)
- **WHEN** the migration runs
- **THEN** it SHALL complete without error (no-op for status files, still bumps version)

## Deprecated Requirements

### ship_url and is_shipped functions
**Reason**: Replaced by `add_pr`/`get_prs` with consistent naming
**Migration**: `ship` CLI → `add-pr`, `is-shipped` CLI → `get-prs` (check output emptiness)

### issue_id scalar field
**Reason**: Replaced by `issues` array to support multi-issue changes
**Migration**: Scalar `"DEV-123"` → array `["DEV-123"]`, null → `[]`

### .shipped sentinel file
**Reason**: Renamed to `.pr-done` for consistency with field rename
**Migration**: Rename file, update `.gitignore` pattern

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Field names: `issues: []` and `prs: []` | Confirmed from intake #1 — user chose plural nouns over verb forms | S:95 R:90 A:95 D:95 |
| 2 | Certain | Four symmetric operations: add-issue, get-issues, add-pr, get-prs | Confirmed from intake #2 — user confirmed symmetric pattern | S:95 R:85 A:90 D:95 |
| 3 | Certain | fab-new uses stageman API, not raw yq | Confirmed from intake #3 — user identified raw yq as a bug | S:90 R:80 A:90 D:95 |
| 4 | Certain | PR title joins issues with space | Confirmed from intake #4 — Linear picks up space-separated IDs | S:85 R:85 A:85 D:90 |
| 5 | Certain | No migration for archived changes | Confirmed from intake #5 — user explicitly confirmed active-only | S:80 R:95 A:90 D:90 |
| 6 | Confident | get-prs replaces is-shipped (no dedicated boolean) | Confirmed from intake #6 — caller checks empty output | S:85 R:90 A:80 D:75 |
| 7 | Certain | Sentinel renamed .shipped → .pr-done | Consistent with field rename; sentinel is an implementation detail | S:80 R:95 A:90 D:90 |
| 8 | Certain | Migration required for active changes | User confirmed — new stageman functions read new field names | S:90 R:70 A:90 D:95 |
| 9 | Certain | .gitignore and scaffold fragment updated | Follows from sentinel rename; both locations confirmed by grep | S:90 R:95 A:95 D:95 |
| 10 | Confident | git-pr reads issues via stageman CLI, not direct YAML | Consistent with the "skills MUST NOT touch .status.yaml directly" principle; git-pr currently tells agent to "read issue_id from .status.yaml" | S:80 R:85 A:80 D:80 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).
