# Spec: Slim Config & Decouple Naming

**Change**: 260226-jq7a-slim-config-decouple-naming
**Created**: 2026-02-26
**Affected memory**:
- `docs/memory/fab-workflow/configuration.md` (modify)
- `docs/memory/fab-workflow/change-lifecycle.md` (modify)
- `docs/memory/fab-workflow/templates.md` (modify)
- `docs/memory/fab-workflow/schemas.md` (modify)

## Non-Goals

- Making `change_name_format` truly configurable (parsed by `changeman.sh`) — the format remains hardcoded in the script; config documents it
- Changing worktree naming — remains independent (`adjective-noun` in `wt-create`)
- Changing backlog entry format — moves to the `idea` package, out of scope here

## Configuration: config.yaml Restructure

### Requirement: Remove `git` Section

`fab/project/config.yaml` SHALL NOT contain a `git` section. The `git.enabled` and `git.branch_prefix` fields are removed.

The scaffold template (`fab/.kit/scaffold/fab/project/config.yaml`) SHALL NOT contain a `git` section.

#### Scenario: Config Without Git Section
- **GIVEN** a project with `config.yaml` containing no `git` section
- **WHEN** any skill reads `config.yaml`
- **THEN** git integration is implicitly enabled
- **AND** branch names equal the change folder name with no prefix

### Requirement: Remove `naming` Section

`fab/project/config.yaml` SHALL NOT contain a `naming` section. Both `naming.format` and `naming.backlog_format` are removed.

The scaffold template SHALL NOT contain a `naming` section.

#### Scenario: Config Without Naming Section
- **GIVEN** a project with `config.yaml` containing no `naming` section
- **WHEN** `/fab-new` generates a change folder name
- **THEN** the name follows the `{YYMMDD}-{XXXX}-{slug}` pattern as implemented by `changeman.sh`

### Requirement: Rename `rules` to `stage_directives`

The `rules` key in `config.yaml` SHALL be renamed to `stage_directives`.

The `stage_directives` section SHALL contain explicit empty-array placeholders for all six pipeline stages: `intake`, `spec`, `tasks`, `apply`, `review`, `hydrate`.

#### Scenario: Stage Directives With All Placeholders
- **GIVEN** a project `config.yaml` with `stage_directives`
- **WHEN** a skill generates an artifact for a stage (e.g., `spec`)
- **THEN** the skill reads `stage_directives.{stage}` for additional generation instructions
- **AND** empty arrays (`[]`) are treated as no additional directives

#### Scenario: Scaffold Template Has All Stage Placeholders
- **GIVEN** a new project created via `/fab-setup`
- **WHEN** the scaffold `config.yaml` is generated
- **THEN** `stage_directives` contains all six stage keys with empty arrays
- **AND** the `spec` stage includes two default directives: GIVEN/WHEN/THEN and `[NEEDS CLARIFICATION]`

### Requirement: Reduce Config Verbosity

`config.yaml` SHALL use single-line comments per field. Multi-line `# Consumed by:` blocks, the file header comment block, and the `# ─────` separator SHALL be removed. Target: ~35 lines.

The scaffold template SHALL follow the same reduced-verbosity style.

#### Scenario: Slim Config Structure
- **GIVEN** the updated `config.yaml`
- **WHEN** a user reads it
- **THEN** it contains these top-level keys only: `project`, `source_paths`, `checklist`, `stage_directives`, `model_tiers`
- **AND** total line count is under 40

## Templates: Add `issue_id` to status.yaml

### Requirement: `issue_id` Field in status.yaml Template

`fab/.kit/templates/status.yaml` SHALL include an `issue_id` field initialized to `null`, placed after `change_type`.

#### Scenario: New Change Has Null issue_id
- **GIVEN** a new change created via `/fab-new`
- **WHEN** `.status.yaml` is initialized from the template
- **THEN** the file contains `issue_id: null`

#### Scenario: issue_id Hydrated From Linear Ticket
- **GIVEN** a change created via `/fab-new DEV-123`
- **WHEN** the Linear ticket is fetched successfully
- **THEN** `.status.yaml` contains `issue_id: DEV-123`
- **AND** the change folder slug does NOT contain the Linear ID

#### Scenario: issue_id Hydrated Later
- **GIVEN** a change with `issue_id: null`
- **WHEN** a user or skill writes an issue ID to `.status.yaml`
- **THEN** `issue_id` is updated (e.g., `issue_id: DEV-456`)
- **AND** downstream skills (e.g., `/git-pr`) can read it

## Skills: `/fab-new` Changes

### Requirement: Decouple Linear ID From Slug

`/fab-new` Step 1 (Generate Slug) SHALL NOT prefix the Linear issue ID into the slug. The slug SHALL contain only the descriptive portion (e.g., `add-oauth`, not `DEV-988-add-oauth`).

If a Linear ticket was detected in Step 0, `/fab-new` SHALL write the issue ID to `.status.yaml` as `issue_id: {ID}` after `changeman.sh new` completes.

#### Scenario: fab-new With Linear Ticket
- **GIVEN** a user runs `/fab-new DEV-988`
- **WHEN** the Linear ticket is fetched and the slug is generated
- **THEN** the folder name is `{YYMMDD}-{XXXX}-{slug}` (no `DEV-988` in slug)
- **AND** `.status.yaml` contains `issue_id: DEV-988`

#### Scenario: fab-new Without Linear Ticket
- **GIVEN** a user runs `/fab-new add user authentication`
- **WHEN** the change is created
- **THEN** the folder name is `{YYMMDD}-{XXXX}-add-user-authentication`
- **AND** `.status.yaml` contains `issue_id: null`

## Skills: `/git-branch` Changes

### Requirement: Remove `git.enabled` Gate

`/git-branch` SHALL NOT check `git.enabled` from `config.yaml`. The skill SHALL always proceed if inside a git repository.

Step 1 (Read Config) SHALL be replaced with Step 1 (Check Git Repo) — verifying `git rev-parse --is-inside-work-tree`.

#### Scenario: git-branch Without Config git Section
- **GIVEN** `config.yaml` has no `git` section
- **WHEN** user runs `/git-branch`
- **THEN** the skill checks for a git repo and proceeds normally

### Requirement: Remove Branch Prefix

`/git-branch` Step 4 (Derive Branch Name) SHALL use the change name directly as the branch name, with no prefix. The `{git.branch_prefix}` reference is removed.

For standalone fallback (no matching change), the raw argument continues to be used as-is.

#### Scenario: Branch Name Equals Change Name
- **GIVEN** a resolved change `260226-jq7a-slim-config`
- **WHEN** `/git-branch` derives the branch name
- **THEN** branch name is `260226-jq7a-slim-config`

## Skills: `/git-pr` Changes

### Requirement: Read `issue_id` for PR Title

`/git-pr` SHALL read `issue_id` from `.status.yaml` when generating the PR title. If `issue_id` is non-null, the issue ID SHALL be included in the PR title for Linear auto-linking.

`/git-pr` SHALL NOT read `git.branch_prefix` from `config.yaml`.

#### Scenario: PR Title With issue_id
- **GIVEN** a change with `issue_id: DEV-123` in `.status.yaml`
- **WHEN** `/git-pr` generates the PR title
- **THEN** the title includes `DEV-123` (e.g., `feat: DEV-123 Add OAuth support`)

#### Scenario: PR Title Without issue_id
- **GIVEN** a change with `issue_id: null`
- **WHEN** `/git-pr` generates the PR title
- **THEN** the title omits the issue ID (e.g., `feat: Add OAuth support`)

## Skills: `/fab-switch` and `/fab-status` Changes

### Requirement: Always Show Git Hints

`/fab-switch` SHALL always show the `/git-branch` tip line after switching, without checking `git.enabled`.

`/fab-status` SHALL always show the current git branch via `git branch --show-current`, without checking `git.enabled`.

#### Scenario: fab-switch Shows git-branch Tip
- **GIVEN** `config.yaml` has no `git` section
- **WHEN** user runs `/fab-switch my-change`
- **THEN** output includes `Tip: run /git-branch to create or switch to the matching branch`

## Skills: `_generation.md` and Consuming Skills

### Requirement: Reference `stage_directives` Instead of `rules`

All skills that reference `config.rules` or `config.yaml` `rules` section SHALL reference `stage_directives` instead.

#### Scenario: Skill Reads Stage Directives
- **GIVEN** a project with `stage_directives.spec` containing directives
- **WHEN** `/fab-continue` generates a spec
- **THEN** the directives from `stage_directives.spec` are incorporated into the generation

## Scripts: Graceful Handling

### Requirement: Scripts Default on Missing Git Config

`dispatch.sh` `get_branch_prefix()` and `batch-fab-switch-change.sh` `get_branch_prefix()` SHALL continue to return `""` when the `git` section is absent from `config.yaml`. No script changes are required — existing fallback behavior is sufficient.

#### Scenario: dispatch.sh Without Git Config
- **GIVEN** `config.yaml` has no `git.branch_prefix`
- **WHEN** `dispatch.sh` calls `get_branch_prefix()`
- **THEN** the function returns `""`

## Specs: Create `naming.md`

### Requirement: Naming Conventions Spec

A new spec `docs/specs/naming.md` SHALL document five naming conventions:

1. **Change folder name** — pattern `{YYMMDD}-{XXXX}-{slug}`, encoded in `changeman.sh`
2. **Git branch** — equals change folder name, encoded in `/git-branch` skill
3. **Worktree** — `{adjective}-{noun}`, encoded in `wt-create`
4. **PR title** — `{type}: {title}`, encoded in `/git-pr` skill
5. **Backlog entry** — `- [ ] [{ID}] {YYYY-MM-DD}: {description}`, encoded in idea command

Each entry SHALL include: the pattern, an example, and where the convention is encoded (skill or script name).

`docs/specs/index.md` SHALL be updated with a row for `naming.md`.

#### Scenario: Naming Spec Content
- **GIVEN** the `naming.md` spec exists
- **WHEN** a developer reads it
- **THEN** they can identify all five naming conventions, their patterns, examples, and encoding locations

## Migration: `0.10.0-to-0.20.0.md`

### Requirement: Config Migration

`fab/.kit/migrations/0.10.0-to-0.20.0.md` SHALL migrate existing projects:

1. Remove the `git:` block from `config.yaml`
2. Remove the `naming:` block from `config.yaml`
3. Rename `rules:` to `stage_directives:` and add empty placeholders for missing stages
4. Strip verbose comments by referring to the scaffold at `fab/.kit/scaffold/fab/project/config.yaml`
5. Bump `fab/project/VERSION` to `0.20.0`

#### Scenario: Migration Applied to Pre-0.20.0 Project
- **GIVEN** a project at version 0.15.0 with `git:` and `naming:` in config
- **WHEN** the migration runs
- **THEN** `config.yaml` no longer contains `git:` or `naming:` sections
- **AND** `rules:` is renamed to `stage_directives:` with all six stage placeholders
- **AND** `fab/project/VERSION` contains `0.20.0`

## Design Decisions

1. **`issue_id` in status.yaml, not folder name**: Decouples tracker ID from immutable identity. The ID can be discovered at any point in the lifecycle.
   - *Why*: Folder names are created once and can't change (prefix is immutable). Issue IDs may not be known at `/fab-new` time.
   - *Rejected*: Embedding in slug (current approach) — requires knowing the ID upfront, pollutes the naming pattern.

2. **Git always enabled, no prefix**: Removes configuration that doesn't match real workflows.
   - *Why*: `branch_prefix` assumes you know the work type (feat/fix) before starting. You usually don't. Disabling git via config is an edge case not worth the config surface area.
   - *Rejected*: Moving `branch_prefix` to git-branch skill — still unnecessary indirection.

3. **`stage_directives` over `rules`**: More descriptive name for per-stage artifact generation instructions.
   - *Why*: "Rules" is too generic. "Stage directives" scopes to the mechanism (per-stage, directive-style instructions for generation).
   - *Rejected*: `stage_skill_overrides` (they add, not override), `stage_instructions` (confuses with skill instructions).

4. **naming.md as project spec, not .kit content**: Naming conventions are documented in `docs/specs/` (project-level), not `fab/.kit/` (portable).
   - *Why*: `.kit/` is shipped to all projects. Naming documentation is project-specific reference material. Skills remain self-contained — they carry their own naming logic.
   - *Rejected*: Adding to `.kit/` — violates portability principle (naming docs reference project-specific patterns).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove `git` section entirely | Confirmed from intake #1 — user explicitly requested. Scripts default gracefully. | S:95 R:85 A:90 D:95 |
| 2 | Certain | Remove `naming` section entirely | Confirmed from intake #2 — format never parsed by code, backlog_format has zero refs. | S:95 R:90 A:90 D:95 |
| 3 | Certain | Rename `rules` → `stage_directives` | Confirmed from intake #3 — user agreed after evaluating alternatives. | S:95 R:90 A:95 D:95 |
| 4 | Certain | All six stage placeholders in `stage_directives` | Confirmed from intake #4 — user: "or it won't get used." | S:90 R:95 A:85 D:90 |
| 5 | Certain | `issue_id` in status.yaml, not folder name | Confirmed from intake #5 — user proposed the design. | S:95 R:85 A:90 D:95 |
| 6 | Certain | Key name is `issue_id` | Confirmed from intake #6 — user used this term directly. | S:85 R:95 A:85 D:80 |
| 7 | Certain | naming.md covers 5 items only | Confirmed from intake #7 — user explicitly scoped. | S:95 R:90 A:90 D:95 |
| 8 | Certain | naming.md is project-level spec | Confirmed from intake #8 — user clarified portability constraint. | S:95 R:90 A:95 D:95 |
| 9 | Certain | `/git-pr` is sole consumer of `issue_id` | Confirmed from intake #9 — only PR title needs it for Linear auto-linking. | S:90 R:85 A:85 D:85 |
| 10 | Certain | Git always enabled after removal | Confirmed from intake #10 — user chose "Always enabled, no prefix." | S:90 R:80 A:85 D:90 |
| 11 | Confident | Migration range `0.10.0-to-0.20.0` | Last migration TO was 0.10.0; catches all current projects. | S:75 R:85 A:80 D:75 |
| 12 | Confident | `issue_id` in PR title (not just description) | Linear auto-links from either; title is more visible. Intake said "title or description." | S:70 R:90 A:75 D:70 |
| 13 | Confident | No script changes needed for dispatch.sh / batch-fab-switch-change.sh | Both `get_branch_prefix()` already return `""` on missing field — verified in code. | S:80 R:90 A:85 D:85 |

13 assumptions (10 certain, 3 confident, 0 tentative, 0 unresolved).
