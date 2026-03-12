# Spec: PR Change Metadata

**Change**: 260312-9r3t-pr-change-metadata
**Created**: 2026-03-12
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/migrations.md`

## git-pr: Change Section

### Requirement: Change Table in PR Body

The `/git-pr` skill SHALL generate a "Change" section in the PR body, placed **above** the existing "Stats" section. The section contains a three-column table: ID, Name, Issue.

The entire "Change" section SHALL be conditional on `{has_fab}` — if no active fab change resolves, the section MUST be omitted entirely (same gating as the pipeline progress line).

#### Scenario: Full metadata with Linear workspace configured

- **GIVEN** an active fab change with `id: r3m7`, `name: 260312-r3m7-add-oauth-support`, issues `["DEV-123"]`, and `linear_workspace: "weaver-ai"` in `fab/project/config.yaml`
- **WHEN** `/git-pr` generates the PR body
- **THEN** the body contains a "## Change" section above "## Stats"
- **AND** the table row shows `r3m7 | 260312-r3m7-add-oauth-support | [DEV-123](https://linear.app/weaver-ai/issue/DEV-123)`

#### Scenario: Multiple issues with Linear workspace

- **GIVEN** an active fab change with issues `["DEV-123", "DEV-456"]` and `linear_workspace: "weaver-ai"`
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Issue column shows `[DEV-123](https://linear.app/weaver-ai/issue/DEV-123), [DEV-456](https://linear.app/weaver-ai/issue/DEV-456)`

#### Scenario: Issues present but no linear_workspace

- **GIVEN** an active fab change with issues `["DEV-123"]` and no `linear_workspace` field in config.yaml
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Issue column shows bare IDs: `DEV-123`

#### Scenario: No issues linked

- **GIVEN** an active fab change with no issues (empty array or `fab status get-issues` returns empty)
- **WHEN** `/git-pr` generates the PR body
- **THEN** the Issue column shows `—`

#### Scenario: ID or name unavailable

- **GIVEN** an active fab change where `.status.yaml` lacks an `id` or `name` field
- **WHEN** `/git-pr` generates the PR body
- **THEN** the missing field's column shows `—`

#### Scenario: No active fab change

- **GIVEN** no active fab change (`fab change resolve` fails)
- **WHEN** `/git-pr` generates the PR body
- **THEN** the "Change" section is omitted entirely

### Requirement: Change Table Format

The Change table SHALL use this exact markdown format:

```markdown
## Change
| ID | Name | Issue |
|----|------|-------|
| {id} | {name} | {issue_display} |
```

Where:
- `{id}` — from `.status.yaml` `id` field (4-char), or `—` if unavailable
- `{name}` — from `.status.yaml` `name` field (full folder name), or `—` if unavailable
- `{issue_display}` — formatted per the linear_workspace rules below, or `—` if no issues

#### Scenario: Table renders as valid markdown

- **GIVEN** an active fab change with all fields populated
- **WHEN** the PR body is rendered on GitHub
- **THEN** the Change section renders as a properly formatted table with three columns

## Configuration: linear_workspace

### Requirement: Optional linear_workspace Field

`fab/project/config.yaml` SHALL support an optional `linear_workspace` field under the `project:` block.

```yaml
project:
  name: "fab-kit"
  description: "FAB Kit — Tooling for Specification-Driven Development"
  linear_workspace: "weaver-ai"  # optional — enables Linear issue links in PRs
```

The field:
- SHALL be optional — absence is not an error
- SHALL be project-level — set once per project
- SHALL be used exclusively by `/git-pr` for URL construction

#### Scenario: Config with linear_workspace present

- **GIVEN** `fab/project/config.yaml` contains `linear_workspace: "weaver-ai"` under `project:`
- **WHEN** `/git-pr` reads the config
- **THEN** issue IDs are rendered as `[{ID}](https://linear.app/weaver-ai/issue/{ID})`

#### Scenario: Config without linear_workspace

- **GIVEN** `fab/project/config.yaml` has no `linear_workspace` field
- **WHEN** `/git-pr` reads the config
- **THEN** issue IDs are rendered as bare text (e.g., `DEV-123`)

### Requirement: URL Construction

When `linear_workspace` is configured, `/git-pr` SHALL construct issue URLs using the pattern:

```
https://linear.app/{linear_workspace}/issue/{ISSUE_ID}
```

Where `{ISSUE_ID}` is the value returned by `fab status get-issues <change>` (e.g., `DEV-123`).

#### Scenario: URL follows Linear convention

- **GIVEN** `linear_workspace: "weaver-ai"` and issue ID `DEV-123`
- **WHEN** the URL is constructed
- **THEN** the result is `https://linear.app/weaver-ai/issue/DEV-123`

## Migration: linear_workspace Discoverability

### Requirement: Migration File

A migration file SHALL be created at `fab/.kit/migrations/0.34.0-to-0.37.0.md` to surface the new `linear_workspace` field to existing users.
<!-- clarified: Migration range 0.34.0-to-0.37.0 — FROM=0.34.0 matches last migration TO (0.32.0-to-0.34.0.md), TO=0.37.0 is next minor after current 0.36.0 -->

The migration:
1. SHALL check if `fab/project/config.yaml` already has a `linear_workspace` field — if so, skip
2. SHALL add a commented-out `linear_workspace` field under the `project:` block
3. SHALL print a note explaining the new field and how to enable it

This is a **discoverability migration** — it MUST NOT change behavior, only surface the new option.

#### Scenario: Fresh config without linear_workspace

- **GIVEN** `fab/project/config.yaml` exists and does not contain `linear_workspace`
- **WHEN** `/fab-setup migrations` applies this migration
- **THEN** a commented-out line `# linear_workspace: "your-workspace"  # optional — enables Linear issue links in PRs` is added under the `project:` block
- **AND** a note is printed explaining the field

#### Scenario: Config already has linear_workspace

- **GIVEN** `fab/project/config.yaml` already contains `linear_workspace:`
- **WHEN** `/fab-setup migrations` applies this migration
- **THEN** the migration is skipped with a note that the field already exists

#### Scenario: No config.yaml

- **GIVEN** `fab/project/config.yaml` does not exist
- **WHEN** `/fab-setup migrations` runs
- **THEN** this migration is skipped (config.yaml absence is a pre-existing issue, not this migration's concern)

## Skill Spec Update

### Requirement: Update SPEC-git-pr.md

`docs/specs/skills/SPEC-git-pr.md` SHALL be updated to reflect the new "Change" section in the flow diagram, adding a Read step for `config.yaml` (linear_workspace) in Step 3c and reflecting the new body section.

#### Scenario: Spec reflects new behavior

- **GIVEN** the git-pr skill now generates a Change section
- **WHEN** a reader consults `docs/specs/skills/SPEC-git-pr.md`
- **THEN** the flow diagram includes the Change section generation step
- **AND** the tools table includes `config.yaml` as a Read source

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Separate "Change" section above Stats, not merged into Stats table | Confirmed from intake #1 — user explicitly chose separate section | S:95 R:90 A:90 D:95 |
| 2 | Certain | Use `linear_workspace` config field for URL construction | Confirmed from intake #2 — user confirmed option D over alternatives | S:95 R:85 A:90 D:95 |
| 3 | Certain | Link format: `https://linear.app/{workspace}/issue/{ID}` | Confirmed from intake #3 — standard Linear URL pattern | S:90 R:90 A:95 D:95 |
| 4 | Confident | Fall back to bare issue IDs when `linear_workspace` is absent | Confirmed from intake #4 — graceful degradation, still shows IDs | S:70 R:90 A:85 D:85 |
| 5 | Confident | Show `—` for empty fields rather than omitting columns | Confirmed from intake #5 — consistent with existing Stats table convention | S:75 R:90 A:85 D:80 |
| 6 | Certain | Whole "Change" section gated on `{has_fab}` | Confirmed from intake #6 — same pattern as pipeline progress line | S:85 R:90 A:90 D:95 |
| 7 | Certain | Migration adds commented-out field, not auto-populated | Confirmed from intake #7 — discoverability only | S:90 R:95 A:85 D:90 |
| 8 | Certain | Migration version range 0.34.0-to-0.37.0 | Clarified — FROM=0.34.0 matches last migration TO (0.32.0-to-0.34.0.md), TO=0.37.0 is next minor after current 0.36.0 | S:90 R:85 A:60 D:55 |
| 9 | Certain | git-pr reads linear_workspace from config.yaml already loaded in context | Config is already loaded during PR body generation — no new file reads needed | S:90 R:95 A:90 D:90 |

9 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).
