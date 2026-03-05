# Spec: Unified PR Template

**Change**: 260305-b0xs-unified-pr-template
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing the PR type resolution chain (4-step chain remains as-is)
- Modifying `/git-pr-review` or any other skill
- Changing how blob URLs are constructed (same `https://github.com/{owner_repo}/blob/{branch}/...` pattern)

## PR Template: Unified Template

### Requirement: Single Template Replaces Two Tiers

The `/git-pr` skill Step 3c SHALL use a single PR body template for all PR types. The template SHALL conditionally populate fab-linked fields based on artifact availability (whether `changeman.sh resolve` succeeds and `intake.md` exists), NOT based on the resolved PR type.

#### Scenario: Full fab pipeline change of type `test`

- **GIVEN** a change of type `test` with completed intake, spec, tasks, apply, review, and hydrate stages
- **WHEN** `/git-pr` generates the PR body
- **THEN** the body includes a Stats table with populated Confidence, Checklist, Tasks, and Review columns
- **AND** the body includes a Pipeline progress line with intake and spec as hyperlinks

#### Scenario: Feat change with no fab artifacts

- **GIVEN** a change of type `feat` where `changeman.sh resolve` fails (no active fab change)
- **WHEN** `/git-pr` generates the PR body
- **THEN** the body includes a Stats table with only Type populated and all other columns showing `—`
- **AND** the Pipeline progress line is omitted

#### Scenario: Fix change with intake but no spec

- **GIVEN** a change of type `fix` with `intake.md` present but no `spec.md`
- **WHEN** `/git-pr` generates the PR body
- **THEN** the body includes a Stats table with Type populated and other columns showing `—` (since no `.status.yaml` confidence/checklist data)
- **AND** the Pipeline progress line shows "intake" as a hyperlink and "spec" as plain text

### Requirement: Horizontal Stats Table

The PR body SHALL include a `## Stats` section with a horizontal table containing five columns: Type, Confidence, Checklist, Tasks, and Review.

Column population rules:
- **Type**: MUST always be populated from the resolved PR type
- **Confidence**: MUST show `{confidence.score} / 5.0` from `.status.yaml`. SHALL show `—` if no fab change resolves or confidence data is absent
- **Checklist**: MUST show `{checklist.completed}/{checklist.total}` from `.status.yaml`. SHALL append ` ✓` when `completed == total` AND `total > 0`. SHALL show `—` if data not available
- **Tasks**: MUST parse `tasks.md` for checkbox counts (`- [x]` vs `- [ ]`), formatted as `{done}/{total}`. SHALL show `—` if `tasks.md` doesn't exist
- **Review**: MUST derive from `.status.yaml` `progress.review` state and `stage_metrics.review.iterations`. SHALL show `Pass ({N} iterations)` if review is `done`, `Fail ({N} iterations)` if review is `failed`, `—` if review not yet reached. If `iterations` is not populated, the parenthetical SHALL be omitted

#### Scenario: All stats populated

- **GIVEN** a change with `confidence.score: 3.5`, `checklist.completed: 21`, `checklist.total: 21`, `tasks.md` with 13/13 checked, `progress.review: done`, `stage_metrics.review.iterations: 2`
- **WHEN** the Stats table is generated
- **THEN** the table renders as:
  ```
  | Type | Confidence | Checklist | Tasks | Review |
  |------|-----------|-----------|-------|--------|
  | feat | 3.5 / 5.0 | 21/21 ✓ | 13/13 | Pass (2 iterations) |
  ```

#### Scenario: No fab change resolved

- **GIVEN** `changeman.sh resolve` fails
- **WHEN** the Stats table is generated
- **THEN** the table renders as:
  ```
  | Type | Confidence | Checklist | Tasks | Review |
  |------|-----------|-----------|-------|--------|
  | chore | — | — | — | — |
  ```

#### Scenario: Review not yet reached

- **GIVEN** a change with valid confidence and checklist but `progress.review: pending`
- **WHEN** the Stats table is generated
- **THEN** the Review column shows `—`

#### Scenario: Review passed without iteration count

- **GIVEN** a change with `progress.review: done` but no `stage_metrics.review.iterations` field
- **WHEN** the Stats table is generated
- **THEN** the Review column shows `Pass` (no parenthetical)

### Requirement: Pipeline Progress Line

Below the Stats table, the PR body SHALL include a pipeline progress line showing stages with `done` status from `.status.yaml`'s `progress` map.

Stages MUST be listed in fixed order: intake, spec, tasks, apply, review, hydrate, ship, review-pr. Only stages with `done` status SHALL be included.

The words "intake" and "spec" SHALL be hyperlinks to GitHub blob URLs when the corresponding artifact files exist. All other stage names SHALL be plain text.

#### Scenario: Full pipeline completed

- **GIVEN** a change with all stages `done`, `intake.md` exists, `spec.md` exists
- **WHEN** the pipeline line is generated
- **THEN** it renders as: `[intake]({blob_url}) → [spec]({blob_url}) → tasks → apply → review → hydrate → ship → review-pr`

#### Scenario: Partial pipeline

- **GIVEN** a change with intake, spec, and tasks `done`, apply `active`
- **WHEN** the pipeline line is generated
- **THEN** it renders as: `[intake]({blob_url}) → [spec]({blob_url}) → tasks`

#### Scenario: Spec file missing

- **GIVEN** a change with intake and spec `done`, but `spec.md` file doesn't exist on disk
- **WHEN** the pipeline line is generated
- **THEN** "spec" is plain text (not a link): `[intake]({blob_url}) → spec`

#### Scenario: No fab change

- **GIVEN** `changeman.sh resolve` fails
- **WHEN** the PR body is generated
- **THEN** the pipeline progress line is omitted entirely

### Requirement: PR Summary Section

The PR body SHALL include a `## Summary` section.

When an active fab change resolves and `intake.md` exists, the summary SHALL be derived from the intake's `## Why` section (1-3 sentences).

When no fab change resolves or no intake exists, the summary SHALL be auto-generated from commit messages or `git diff --stat`.

#### Scenario: Fab-linked summary

- **GIVEN** an active fab change with `intake.md` containing a `## Why` section
- **WHEN** the PR body is generated
- **THEN** `## Summary` contains 1-3 sentences derived from the Why section

#### Scenario: Fallback summary

- **GIVEN** no active fab change
- **WHEN** the PR body is generated
- **THEN** `## Summary` contains auto-generated content from commits/diff

### Requirement: Changes Section

When an active fab change resolves and `intake.md` exists, the PR body SHALL include a `## Changes` section with a bulleted list of subsection headings from the intake's `## What Changes` section.

When no fab change resolves or no intake exists, the Changes section SHALL be omitted.

#### Scenario: Fab-linked changes

- **GIVEN** an active fab change with intake containing subsections "Single Template Replaces Two Tiers", "Horizontal Stats Table", "Pipeline Line"
- **WHEN** the PR body is generated
- **THEN** `## Changes` lists: `- Single Template Replaces Two Tiers`, `- Horizontal Stats Table`, `- Pipeline Line`

#### Scenario: No fab change

- **GIVEN** no active fab change
- **WHEN** the PR body is generated
- **THEN** the `## Changes` section is omitted

## PR Title: Simplified Derivation

### Requirement: Unified Title Derivation

The PR title derivation in Step 3c step 2 SHALL use intake heading when intake exists and commit subject otherwise, regardless of PR type. The current type-based branching (fab-linked types vs lightweight types) SHALL be removed.

When `changeman.sh resolve` succeeds AND `intake.md` exists: `{title}` = first `# ` heading from `intake.md`, stripping `Intake: ` prefix if present.

Otherwise: `{title}` = commit message subject line from `git log -1 --format=%s`.

The `{pr_title}` format (`{type}: {issues} {title}` or `{type}: {title}`) remains unchanged.

#### Scenario: Test change with intake

- **GIVEN** a change of type `test` with `intake.md` starting with `# Intake: Add Widget Tests`
- **WHEN** the PR title is derived
- **THEN** `{title}` = `Add Widget Tests` (stripped prefix, regardless of type)

#### Scenario: Feat change without intake

- **GIVEN** a change of type `feat` where `changeman.sh resolve` fails
- **WHEN** the PR title is derived
- **THEN** `{title}` = commit subject line

## PR Type Reference: Cleanup

### Requirement: Remove Obsolete Columns

The PR Type Reference table at the bottom of `git-pr.md` SHALL remove the "Fab Pipeline?" and "Template Tier" columns. The table SHALL retain only "Type" and "Description" columns.

#### Scenario: Reference table after cleanup

- **GIVEN** the PR Type Reference table in `git-pr.md`
- **WHEN** the cleanup is applied
- **THEN** the table has two columns: Type and Description
- **AND** all 7 types are listed with descriptions preserved

## Deprecated Requirements

### Two-Tier Template System

**Reason**: Replaced by unified template. The distinction between Tier 1 (fab-linked) and Tier 2 (lightweight) based on PR type is removed. All PR types now use the same template with conditional field population.

**Migration**: Single unified template (this spec's "Single Template Replaces Two Tiers" requirement).

### Type-Gated Title Derivation

**Reason**: Title derivation no longer branches on whether the type is fab-linked vs lightweight. Simplified to: intake heading when available, commit subject otherwise.

**Migration**: "Unified Title Derivation" requirement in this spec.

## Design Decisions

### 1. Conditional Population by Artifact Availability, Not PR Type

**Decision**: Template fields are populated based on whether fab artifacts exist (changeman resolves, intake.md exists, .status.yaml has data), not based on the PR type.

*Why*: A `test` or `docs` change that went through the full fab pipeline deserves the same quality signals as a `feat` change. Gating on type hides real work.

*Rejected*: Keep type-gating but extend Tier 1 to all types — still requires two templates and the conditional logic is backwards (types don't predict artifact availability).

### 2. Dashes for Unavailable Fields, Not Column Omission

**Decision**: Show `—` for unavailable fields rather than omitting columns or sections.

*Why*: Keeps table shape consistent across all PRs. Reviewers always see the same columns, reducing cognitive overhead. Consistent with the mockup approved in discussion.

*Rejected*: Omit columns when empty — table structure varies per PR, harder to scan. Show "N/A" — less clean than `—`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Single template with conditional field population | Confirmed from intake #1 — user explicitly chose unified over two-tier | S:95 R:80 A:90 D:95 |
| 2 | Certain | Horizontal stats table with Type/Confidence/Checklist/Tasks/Review columns | Confirmed from intake #2 — user approved mockup | S:95 R:85 A:85 D:95 |
| 3 | Certain | Pipeline line below stats with intake/spec as links | Confirmed from intake #3 — user refined in discussion | S:95 R:90 A:90 D:95 |
| 4 | Certain | Review column shows Pass/Fail with iteration count | Confirmed from intake #4 — user approved | S:90 R:85 A:85 D:90 |
| 5 | Certain | Tasks column parsed from tasks.md checkboxes | Confirmed from intake #5 — user approved | S:90 R:85 A:80 D:90 |
| 6 | Confident | Show `—` for unavailable fields rather than omitting columns | Confirmed from intake #6 — strong convention, keeps table consistent | S:75 R:90 A:80 D:75 |
| 7 | Certain | Remove "Fab Pipeline?" and "Template Tier" columns from PR Type Reference | Confirmed from intake #7 — columns meaningless with unified template | S:90 R:90 A:90 D:90 |
| 8 | Certain | PR title uses intake heading when available, commit subject otherwise, regardless of type | Confirmed from intake #8 — user approved simplification | S:90 R:85 A:85 D:90 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
