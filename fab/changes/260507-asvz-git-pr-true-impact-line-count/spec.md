# Spec: git-pr true-impact line count

**Change**: 260507-asvz-git-pr-true-impact-line-count
**Created**: 2026-05-07
**Affected memory**: `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/migrations.md`

## Non-Goals

- File-count display (`+X / −Y across N files`) — deferred per intake Open Questions; not in scope.
- Configurable display label / wording — the format is fixed; only the exclusion list is configurable.
- Per-file or per-language breakdowns — out of scope. The impact block reports two aggregate `+/−` line counts only.
- Auto-detection of "noise" directories — projects opt in by populating the config field; no heuristics.

## git-pr: True-Impact Block

### Requirement: Optional `true_impact_exclude` config field

`fab/project/config.yaml` MAY contain a top-level `true_impact_exclude` key. When present, its value SHALL be a YAML sequence of pathspec exclusion patterns (typically directory prefixes ending in `/`, but any pattern accepted by `git diff` `:(exclude)<pattern>` syntax is valid).

When the field is absent, an empty sequence, or `null`, `/git-pr` SHALL behave exactly as it does today — the impact block is omitted from the PR body.

#### Scenario: Field present with directory list
- **GIVEN** `fab/project/config.yaml` contains `true_impact_exclude: [fab/, docs/]`
- **WHEN** `/git-pr` assembles the PR body
- **THEN** `/git-pr` SHALL emit a two-line impact block whose first line lists `fab/, docs/` in the exclusion clause
- **AND** the block SHALL appear at the bottom of the PR body, after every other section

#### Scenario: Field absent
- **GIVEN** `fab/project/config.yaml` has no `true_impact_exclude` key
- **WHEN** `/git-pr` assembles the PR body
- **THEN** `/git-pr` SHALL omit the impact block entirely
- **AND** the rest of the PR body SHALL match today's output byte-for-byte

#### Scenario: Field present but empty
- **GIVEN** `fab/project/config.yaml` contains `true_impact_exclude: []`
- **WHEN** `/git-pr` assembles the PR body
- **THEN** `/git-pr` SHALL omit the impact block entirely (treated identically to "field absent")

### Requirement: Impact block computation

When the impact block is emitted, `/git-pr` SHALL compute the two line-count pairs as follows.

Let `BASE` be the merge-base between `HEAD` and the resolved default branch (already computed in `/git-pr`). Let `EXCLUDES` be the value of `true_impact_exclude` from config.

The skill SHALL invoke `git diff --shortstat "$BASE...HEAD"` **twice** against the same `BASE`:

1. **True-impact pass** (with exclusions):
   ```bash
   git diff --shortstat "$BASE...HEAD" -- . \
     $(printf "':(exclude)%s' " "${EXCLUDES[@]}")
   ```

2. **Total pass** (no exclusions):
   ```bash
   git diff --shortstat "$BASE...HEAD"
   ```

The output of `git diff --shortstat` has the form `N files changed, A insertions(+), D deletions(-)` (with `insertions` and `deletions` clauses each independently optional when zero). The skill SHALL parse `A` (insertions) and `D` (deletions) from each pass; missing values default to `0`.

#### Scenario: Both passes return non-zero counts
- **GIVEN** the true-impact pass output is `4 files changed, 50 insertions(+), 10 deletions(-)`
- **AND** the total pass output is `42 files changed, 800 insertions(+), 50 deletions(-)`
- **WHEN** the impact block is rendered
- **THEN** the block SHALL be:
  ```
  **Impact (excluding fab/, docs/)**: +50 / −10
  **git diff total**: +800 / −50
  ```

#### Scenario: True-impact pass returns zero changes
- **GIVEN** every file modified in the diff falls inside an excluded path
- **AND** the true-impact pass output is empty (no `insertions` / `deletions` clause)
- **WHEN** the impact block is rendered
- **THEN** `/git-pr` SHALL omit the entire impact block
- **AND** the PR body SHALL be identical to today's output

#### Scenario: One side of the pair is zero
- **GIVEN** the true-impact pass output is `2 files changed, 50 insertions(+)` (no deletions clause)
- **WHEN** the impact block is rendered
- **THEN** the first line SHALL be `**Impact (excluding fab/, docs/)**: +50 / −0`
- **AND** the rendered numbers SHALL be `+0` / `−0` for any clause missing from `git diff --shortstat` output

### Requirement: Impact block format and placement

The impact block SHALL consist of exactly two markdown lines, in this order:

```markdown
**Impact (excluding {COMMA_LIST})**: +A / −D
**git diff total**: +A_total / −D_total
```

Where `{COMMA_LIST}` is the value of `true_impact_exclude` joined with `, ` (a literal comma followed by a space). The list SHALL reflect the actual config values verbatim — `/git-pr` SHALL NOT hardcode `fab/, docs/`.

The block SHALL be appended at the **bottom of the PR body**, after every other section (after `## Stats` and the optional pipeline progress line).

A single blank line SHALL separate the impact block from the preceding content. The block SHALL NOT be wrapped in a `## Impact` heading.

#### Scenario: Single-entry exclusion list
- **GIVEN** `true_impact_exclude: [vendor/]`
- **WHEN** the impact block is rendered
- **THEN** the first line SHALL be `**Impact (excluding vendor/)**: +X / −Y` (no trailing comma)

#### Scenario: Three-entry exclusion list
- **GIVEN** `true_impact_exclude: [fab/, docs/, vendor/]`
- **WHEN** the impact block is rendered
- **THEN** the first line SHALL be `**Impact (excluding fab/, docs/, vendor/)**: +X / −Y`

#### Scenario: Block placement
- **GIVEN** the PR body already contains `## Summary`, `## Changes`, `## Change`, `## Stats`, and the pipeline progress line
- **WHEN** the impact block is appended
- **THEN** the impact block SHALL be the last content in the PR body
- **AND** every preceding section SHALL retain its current ordering

### Requirement: Graceful degradation when fab context absent

The impact block depends on `fab/project/config.yaml`. When `/git-pr` is invoked from a directory that lacks fab project state (`fab/project/config.yaml` does not exist), the skill SHALL skip the impact block silently and proceed with PR body assembly. This preserves `/git-pr`'s existing fab-optional behavior (`{has_fab} = false`).

#### Scenario: No fab context
- **GIVEN** `fab/project/config.yaml` does not exist
- **WHEN** `/git-pr` runs
- **THEN** `/git-pr` SHALL omit the impact block
- **AND** the PR body SHALL match today's no-fab output byte-for-byte

## Config Template Defaults

### Requirement: New-project default in scaffold

The scaffold template at `src/kit/scaffold/fab/project/config.yaml` SHALL include `true_impact_exclude` populated with `[fab/, docs/]` so new fab-kit projects emit the impact block out of the box.

#### Scenario: Fresh project bootstrap
- **GIVEN** a directory with no existing fab state
- **WHEN** the user runs `/fab-setup` to bootstrap fab
- **THEN** the generated `fab/project/config.yaml` SHALL contain a `true_impact_exclude: [fab/, docs/]` entry

## Migration

### Requirement: Migration adds the field to existing configs

A migration file SHALL ship in `src/kit/migrations/` whose name follows the `{FROM}-to-{TO}.md` convention and targets the next kit version that includes this change. The migration SHALL:

1. Verify `fab/project/config.yaml` exists. If not, no-op (project not initialized).
2. If `fab/project/config.yaml` already contains a top-level `true_impact_exclude` key (with any value), no-op idempotently. Print: `Skipped: true_impact_exclude already present.`
3. Otherwise, append `true_impact_exclude: [fab/, docs/]` to `fab/project/config.yaml` as a top-level entry. Preserve all other config sections, comments, and formatting verbatim. Use atomic write (temp + rename).

The migration SHALL NOT alter any other fab files, change folders, or status state.

#### Scenario: Existing config without the field
- **GIVEN** `fab/project/config.yaml` exists and lacks `true_impact_exclude`
- **WHEN** the user runs `/fab-setup migrations`
- **THEN** the migration SHALL append `true_impact_exclude: [fab/, docs/]` to the config
- **AND** all other config sections SHALL remain unchanged

#### Scenario: Existing config with the field already populated
- **GIVEN** `fab/project/config.yaml` already contains `true_impact_exclude: [vendor/]`
- **WHEN** the migration runs
- **THEN** the migration SHALL be a no-op
- **AND** the existing `[vendor/]` value SHALL NOT be overwritten

#### Scenario: Re-running the migration
- **GIVEN** the migration has already been applied
- **WHEN** `/fab-setup migrations` re-runs
- **THEN** the migration SHALL detect the field and skip without error

## Documentation

### Requirement: Skill spec update

`docs/specs/skills/SPEC-git-pr.md` SHALL document the new sub-step in the `/git-pr` PR body assembly flow, including the config dependency and the graceful-degradation behavior when the field is absent or empty.

#### Scenario: Spec file contains the new behavior
- **GIVEN** the change has been applied
- **WHEN** a reader opens `docs/specs/skills/SPEC-git-pr.md`
- **THEN** the spec SHALL describe the impact-block step, its config dependency, and the omission rules

### Requirement: Memory hydration

`docs/memory/fab-workflow/configuration.md` SHALL document the new `true_impact_exclude` config field, its format (sequence of pathspec exclusion patterns), its default in new projects, and its semantics (omitted when absent or empty).

`docs/memory/fab-workflow/migrations.md` SHALL append a changelog entry for the new migration file.

#### Scenario: Configuration memory updated
- **GIVEN** the change has been hydrated
- **WHEN** a reader opens `docs/memory/fab-workflow/configuration.md`
- **THEN** the memory SHALL list `true_impact_exclude` under the `config.yaml` schema section
- **AND** SHALL describe the semantics of presence/absence/empty values

## Design Decisions

1. **Field name `true_impact_exclude` (top-level)**:
   - *Why*: Echoes the user-coined "true impact" framing from the original conversation; flat top-level matches the existing `source_paths` convention; simple to consume in shell with `yq '.true_impact_exclude'`.
   - *Rejected*: `pr_impact_exclude` (less evocative); nested `git_pr.impact_exclude` (premature namespacing for a single option); `pr.impact_exclude` (same — and "pr" is fab-kit-internal jargon, not a project-level concept).

2. **Two `git diff` invocations rather than one**:
   - *Why*: Producing both the true-impact and the total counts requires the diff with and without exclusions. Two `--shortstat` invocations are trivially fast (< 100ms typical) and let each line of the block trace to a single, parseable command output. No state shared between them other than `BASE`.
   - *Rejected*: Single `git diff --numstat` summed in shell (more brittle parser, doesn't materially reduce work); subtracting excluded paths from the total (would require parsing every excluded file's diff individually).

3. **Three-dot range `$BASE...HEAD`**:
   - *Why*: Matches the "changes on this branch" semantics that reviewers expect, even if the base branch advanced after the feature branch was cut. Two-dot would understate impact when `main` has moved.
   - *Rejected*: Two-dot range `$BASE..HEAD` (different semantics; can produce confusing numbers when base moves).

4. **Block placement at bottom of PR body**:
   - *Why*: Footer-style metadata; reviewers see the prose first, then scope. Confirmed by user during clarify after considering top and middle placements.
   - *Rejected*: Top placement (visually dominant, but pushes the user's actual summary below the fold); middle placement between Summary and Test plan (the current `/git-pr` body has no Test plan section, so this would land between Summary/Changes and Change/Stats — awkward).

5. **Migration over lazy default**:
   - *Why*: Constitution's user-data-restructuring rule (per the `migrations` memory) mandates a migration file for any change that adds fields to user-owned `config.yaml`. Lazy in-skill defaults would drift from what the user sees in their config and violate the "config says what you use" principle (memory: configuration § Config for Facts).
   - *Rejected*: Built-in `[fab/, docs/]` fallback inside `/git-pr` (drift between visible config and effective behavior); opt-in only (defeats the goal of every project benefiting).

6. **Two-line block (true-impact + git diff total) rather than single-line true-impact**:
   - *Why*: User confirmed during clarify that contrast between filtered and unfiltered numbers is the primary signal — a single number doesn't communicate the `fab/`+`docs/` overhead, which is the motivating problem.
   - *Rejected*: True-impact only (loses the contrast); inline `(GitHub total: ...)` (less readable, and "GitHub" mislabels — the source is `git`).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Impact block emitted in PR body, never in title | Confirmed from intake #1 | S:95 R:90 A:90 D:95 |
| 2 | Certain | Default exclusion list `[fab/, docs/]` for new projects | Confirmed from intake #2 | S:95 R:85 A:90 D:95 |
| 3 | Certain | Top-level `true_impact_exclude` in `fab/project/config.yaml` | Confirmed from intake #3+#4 (clarified value) | S:95 R:75 A:85 D:80 |
| 4 | Certain | Two `git diff --shortstat` calls, three-dot range vs. merge-base | Confirmed from intake #5 (clarified) | S:95 R:80 A:90 D:75 |
| 5 | Certain | Migration `1.9.1-to-1.9.2.md` adds field idempotently | Confirmed from intake #6 (clarified); filename uses canonical `{FROM}-to-{TO}.md` convention against pre-bump `src/kit/VERSION` | S:95 R:70 A:85 D:80 |
| 6 | Certain | Block format: `**Impact (excluding ...)**: +X / −Y` then `**git diff total**: +A / −B` | Confirmed from intake #7+#8 (clarified) | S:95 R:85 A:60 D:55 |
| 7 | Certain | Block placed at bottom of PR body, after `## Stats` and pipeline line | Confirmed from intake #9 (clarified) | S:95 R:90 A:60 D:55 |
| 8 | Certain | Empty-list and missing-field treated identically (block omitted) | Spec-stage decision: removes a degenerate-state ambiguity | S:90 R:90 A:90 D:90 |
| 9 | Certain | Missing `insertions` / `deletions` clauses in `--shortstat` rendered as `+0` / `−0` | Spec-stage decision: defines parser fallback explicitly | S:90 R:90 A:90 D:90 |
| 10 | Certain | Migration target version is `1.9.2` (next patch after 1.9.1) | Spec-stage decision: matches the post-apply `src/kit/VERSION` bump; canonical migration filename `1.9.1-to-1.9.2.md` | S:95 R:90 A:85 D:85 |
| 11 | Certain | Empty-list, null, and missing field treated identically by `/git-pr` | Spec-stage decision: removes ambiguity for users who explicitly opt out | S:90 R:90 A:90 D:90 |
| 12 | Certain | True-impact pass with zero changes outside exclusions → block omitted entirely | Spec-stage decision: avoids a misleading `+0 / −0` line | S:85 R:90 A:90 D:90 |

12 assumptions (12 certain, 0 confident, 0 tentative, 0 unresolved).
