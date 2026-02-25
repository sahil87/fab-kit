# Spec: Smart Change Resolution & PR Summary Generation

**Change**: 260224-1jkh-smart-resolve-and-pr-summary
**Created**: 2026-02-24
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Modifying the override mode of `cmd_resolve()` — the guessing fallback applies only to default mode (no `$1` argument)
- Auto-switching `fab/current` during the guess — the guess is transient, it resolves a name but does not write the pointer
- Generating PR body from `spec.md` content — only `intake.md` sections are extracted for the body; spec is linked but not inlined

## Changeman: Single-Change Guessing

### Requirement: Fallback Resolution When No Active Change

When `cmd_resolve()` is invoked in default mode (no override argument) and `fab/current` is missing or empty, the function SHALL enumerate non-archive folders in `fab/changes/` that contain a valid `.status.yaml` file. If exactly one candidate exists, the function SHALL return that candidate's folder name to stdout and emit a diagnostic note to stderr.

#### Scenario: Single Active Change — Successful Guess

- **GIVEN** `fab/current` does not exist or is empty
- **AND** `fab/changes/` contains exactly one non-archive folder with a `.status.yaml`
- **WHEN** `changeman.sh resolve` is invoked with no arguments
- **THEN** stdout contains the folder name of the single candidate
- **AND** stderr contains `(resolved from single active change)`

#### Scenario: No Active Changes

- **GIVEN** `fab/current` does not exist or is empty
- **AND** `fab/changes/` contains no non-archive folders with a `.status.yaml`
- **WHEN** `changeman.sh resolve` is invoked with no arguments
- **THEN** the function exits with code 1
- **AND** stderr contains `No active change.`

#### Scenario: Multiple Active Changes

- **GIVEN** `fab/current` does not exist or is empty
- **AND** `fab/changes/` contains two or more non-archive folders with `.status.yaml`
- **WHEN** `changeman.sh resolve` is invoked with no arguments
- **THEN** the function exits with code 1
- **AND** stderr contains `No active change (multiple changes exist — use /fab-switch).`

#### Scenario: Existing fab/current Still Takes Priority

- **GIVEN** `fab/current` exists and contains a valid change name
- **WHEN** `changeman.sh resolve` is invoked with no arguments
- **THEN** the function returns the name from `fab/current` (existing behavior, unchanged)
- **AND** no guessing fallback is triggered

### Requirement: Candidate Filtering

The guessing fallback SHALL exclude folders that do not contain a `.status.yaml` file. Folders without `.status.yaml` are considered corrupted or incomplete and MUST NOT be counted as candidates.

#### Scenario: Folder Without .status.yaml Excluded

- **GIVEN** `fab/current` does not exist
- **AND** `fab/changes/` contains two folders: `change-a/` (with `.status.yaml`) and `change-b/` (without `.status.yaml`)
- **WHEN** `changeman.sh resolve` is invoked with no arguments
- **THEN** stdout contains `change-a`
- **AND** stderr contains `(resolved from single active change)`

#### Scenario: Archive Folder Excluded

- **GIVEN** `fab/current` does not exist
- **AND** `fab/changes/` contains `archive/` and one change folder with `.status.yaml`
- **WHEN** `changeman.sh resolve` is invoked with no arguments
- **THEN** the `archive/` directory is excluded from candidates
- **AND** stdout contains the single change folder name

### Requirement: Downstream Propagation

All callers of `changeman.sh resolve` — including `preflight.sh`, `/git-branch`, and `/git-pr` — SHALL benefit from the guessing fallback automatically without code changes. The fallback is transparent to callers.

#### Scenario: Preflight Benefits From Guess

- **GIVEN** `fab/current` does not exist
- **AND** exactly one active change exists in `fab/changes/`
- **WHEN** `preflight.sh` is invoked
- **THEN** preflight succeeds using the guessed change name
- **AND** the stderr diagnostic note is visible but non-blocking

## Git-PR: Intake-Aware PR Summary

### Requirement: Intake-Derived PR Body

When creating a PR (Step 3c), `/git-pr` SHALL attempt to resolve the active change and, if successful, generate a PR with title and body derived from `intake.md`.

#### Scenario: Active Change With Intake — Rich PR

- **GIVEN** an active change resolves successfully (via `changeman.sh resolve`)
- **AND** `fab/changes/{name}/intake.md` exists
- **WHEN** `/git-pr` reaches Step 3c (Create PR)
- **THEN** the PR title is derived from the intake's H1 heading with the `Intake: ` prefix stripped
- **AND** the PR body contains a `## Summary` section derived from the intake's `## Why` section
- **AND** the PR body contains a `## Changes` section derived from the `## What Changes` subsection headings
- **AND** the PR body contains a `## Context` section with a relative link to `fab/changes/{name}/intake.md`
- **AND** the PR body's Context section includes a link to `fab/changes/{name}/spec.md` if that file exists
- **AND** the PR is created via `gh pr create --title "<title>" --body "<body>"`

#### Scenario: Active Change Without Intake — Fallback

- **GIVEN** an active change resolves successfully
- **AND** `fab/changes/{name}/intake.md` does NOT exist
- **WHEN** `/git-pr` reaches Step 3c
- **THEN** the PR is created via `gh pr create --fill` (existing behavior)

#### Scenario: No Active Change — Fallback

- **GIVEN** `changeman.sh resolve` fails (no active change, multiple changes, etc.)
- **WHEN** `/git-pr` reaches Step 3c
- **THEN** the PR is created via `gh pr create --fill` (existing behavior)

### Requirement: PR Title Derivation

The PR title SHALL be derived from the intake's first H1 heading (`# ...`). The prefix `Intake: ` SHALL be stripped if present. The title SHOULD be concise (under 70 characters).

#### Scenario: Title With Intake Prefix

- **GIVEN** intake.md has `# Intake: Smart Change Resolution & PR Summary Generation`
- **WHEN** the PR title is derived
- **THEN** the title is `Smart Change Resolution & PR Summary Generation`

### Requirement: PR Body Structure

The PR body SHALL follow this structure:

```
## Summary
{1-3 sentences derived from intake ## Why section}

## Changes
{bulleted list of subsection headings from intake ## What Changes}

## Context
- [Intake](fab/changes/{name}/intake.md)
- [Spec](fab/changes/{name}/spec.md)  ← only if spec.md exists
```

#### Scenario: Full PR Body With Spec

- **GIVEN** intake.md and spec.md both exist for the active change
- **WHEN** the PR body is generated
- **THEN** the Context section contains links to both intake.md and spec.md

#### Scenario: PR Body Without Spec

- **GIVEN** intake.md exists but spec.md does not
- **WHEN** the PR body is generated
- **THEN** the Context section contains only a link to intake.md

### Requirement: Non-Blocking Resolution

The intake-aware PR generation MUST NOT block or fail the PR workflow. If resolution fails or intake parsing encounters errors, the skill SHALL fall back to `gh pr create --fill` silently.

#### Scenario: Resolution Error — Silent Fallback

- **GIVEN** `changeman.sh resolve` exits non-zero
- **WHEN** `/git-pr` reaches Step 3c
- **THEN** the skill proceeds with `gh pr create --fill`
- **AND** no error or warning is displayed about the resolution failure

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Guess only when exactly one candidate exists | Confirmed from intake #1 — user explicitly confirmed; multiple candidates should error with guidance | S:90 R:90 A:95 D:95 |
| 2 | Certain | Filter candidates by .status.yaml existence | Confirmed from intake #2 — folders without status are corrupted/incomplete | S:85 R:90 A:90 D:95 |
| 3 | Certain | Emit stderr note when guessing | Confirmed from intake #3 — visible but non-blocking diagnostic | S:90 R:95 A:90 D:95 |
| 4 | Certain | PR body includes intake summary + context links | Confirmed from intake #4 — user explicitly requested directing reviewers to intake | S:95 R:85 A:90 D:90 |
| 5 | Certain | Fall back to --fill when no active change or no intake | Confirmed from intake #5 — non-fab PRs and early-stage changes keep working | S:90 R:95 A:90 D:95 |
| 6 | Confident | PR title derived from intake H1 heading | Confirmed from intake #6 — strong template signal; easily changed if user prefers different source | S:70 R:90 A:80 D:75 |
| 7 | Confident | Include spec link only if spec.md exists | Confirmed from intake #7 — early-stage changes may not have a spec yet | S:75 R:90 A:85 D:80 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
