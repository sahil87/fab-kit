# Spec: Remove Copilot Integration and Clean Stale Scaffold

**Change**: 260402-5tci-remove-copilot-clean-scaffold
**Created**: 2026-04-02
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/migrations.md`

## Scaffold: Copilot Code Review Config

### Requirement: Remove Copilot scaffold file
The scaffold SHALL NOT include `.github/copilot-code-review.yml`. The file `src/kit/scaffold/.github/copilot-code-review.yml` MUST be deleted.

#### Scenario: New project scaffolded after this change
- **GIVEN** a user runs `fab sync` on a new project
- **WHEN** the scaffold tree is walked
- **THEN** no `.github/copilot-code-review.yml` is created in the project

## Scaffold: .gitignore Fragment

### Requirement: Remove stale .gitignore entries
The `.gitignore` scaffold fragment (`src/kit/scaffold/fragment-.gitignore`) MUST NOT contain the `fab/changes/**/.pr-done` or `/.ralph` entries.

#### Scenario: Fragment after cleanup
- **GIVEN** the scaffold fragment at `src/kit/scaffold/fragment-.gitignore`
- **WHEN** a user runs `fab sync`
- **THEN** neither `fab/changes/**/.pr-done` nor `/.ralph` are added to the project's `.gitignore`

#### Scenario: Existing entries in scaffold
- **GIVEN** the fragment currently contains lines for `.fab-*`, agent-specific folders (`.agents`, `.claude`, `.cursor`, `.opencode`, `.codex`, `.gemini`)
- **WHEN** the stale lines are removed
- **THEN** the fragment retains only the `.fab-*` pattern and the agent-specific folder ignores

## Skill: git-pr-review

### Requirement: Remove Copilot auto-request and polling
The `/git-pr-review` skill MUST NOT request Copilot as a reviewer (Phase 2) or poll for Copilot review completion (Phase 3). When Phase 1 finds no existing reviews with comments, the skill SHALL print `No reviews found — nothing to do.` and STOP.
<!-- clarified: The existing HTML comment in Step 2 about body-only reviews falling through to "Copilot request" must be updated — the fall-through target is now the stop message, not Phase 2. Covered by the broader "remove Copilot-specific references" requirement. -->

#### Scenario: No reviews exist on PR
- **GIVEN** a PR with no reviews and no inline comments
- **WHEN** `/git-pr-review` runs Phase 1
- **THEN** it prints `No reviews found — nothing to do.` and stops
- **AND** the review-pr stage completes as `done` (successful no-op)

#### Scenario: Existing reviews with comments
- **GIVEN** a PR with reviews that have inline comments (from any source: human, Copilot, bot)
- **WHEN** `/git-pr-review` runs Phase 1
- **THEN** it proceeds to Step 3 Path A (fetch all comments) and processes them normally

#### Scenario: Reviews exist but no inline comments
- **GIVEN** a PR with reviews but no inline comments
- **WHEN** `/git-pr-review` runs Phase 1
- **THEN** it prints `No reviews found — nothing to do.` and stops
- **AND** the review-pr stage completes as `done`

### Requirement: Remove Path B from comment fetching
Step 3 SHALL only include Path A (fetch all review comments). Path B (Copilot-specific review comments by `review_id`) MUST be removed since no `review_id` is captured without Phase 2/3.

#### Scenario: Comment fetch always uses Path A
- **GIVEN** existing reviews with inline comments
- **WHEN** Step 3 fetches comments
- **THEN** it uses `GET /pulls/{number}/comments` with `--paginate` (Path A)
- **AND** no `review_id`-specific endpoint is called

### Requirement: Simplify commit message logic
Step 5 commit message logic MUST remove the Copilot-specific branch. The logic SHALL be: single reviewer → `fix: address review feedback from @{username}`, multiple reviewers → `fix: address PR review feedback`.

#### Scenario: Single reviewer commit message
- **GIVEN** all processed comments are from a single reviewer `@alice`
- **WHEN** the agent commits fixes
- **THEN** the commit message is `fix: address review feedback from @alice`

#### Scenario: Multiple reviewer commit message
- **GIVEN** processed comments are from multiple reviewers
- **WHEN** the agent commits fixes
- **THEN** the commit message is `fix: address PR review feedback`

### Requirement: Remove Copilot-specific phase tracking
The Phase Sub-State Tracking table MUST remove the `waiting` entry. The `received` entry description SHALL change to `Reviews detected`.

#### Scenario: Phase tracking after change
- **GIVEN** `/git-pr-review` detects existing reviews
- **WHEN** it updates phase sub-state
- **THEN** it sets `received` (not `waiting`) as the first phase value

### Requirement: Update skill metadata
The skill description frontmatter MUST change from "human or Copilot" to "human or bot". The introductory paragraph and Step 6 MUST remove Copilot-specific references.

#### Scenario: Skill description
- **GIVEN** the `git-pr-review.md` frontmatter
- **WHEN** read by `fab fab-help` or other tools
- **THEN** the description reads "Process PR review comments — triage and fix feedback from any reviewer (human or bot)."

### Requirement: Remove Copilot API login name comment
The HTML comment block documenting the `Copilot` vs `copilot-pull-request-reviewer[bot]` login name discrepancy MUST be removed.

#### Scenario: No Copilot API comments in skill
- **GIVEN** the updated `git-pr-review.md`
- **WHEN** searched for `copilot-pull-request-reviewer`
- **THEN** no matches are found

## Migration: 0.46.0 to 0.47.0 Additions

### Requirement: Remove Copilot code review config
The migration MUST add a step to delete `.github/copilot-code-review.yml` if it exists.

#### Scenario: File exists
- **GIVEN** an existing project with `.github/copilot-code-review.yml`
- **WHEN** the migration runs
- **THEN** the file is deleted
- **AND** a status message is printed

#### Scenario: File does not exist
- **GIVEN** an existing project without `.github/copilot-code-review.yml`
- **WHEN** the migration runs
- **THEN** it prints that the file is already absent and skips

### Requirement: Clean stale .gitignore entries
The migration MUST add a step to remove `/.ralph` and `fab/changes/**/.pr-done` lines from `.gitignore` if present.

#### Scenario: Both entries present
- **GIVEN** a `.gitignore` containing both `/.ralph` and `fab/changes/**/.pr-done`
- **WHEN** the migration runs
- **THEN** both lines are removed
- **AND** a status message is printed

#### Scenario: Neither entry present
- **GIVEN** a `.gitignore` without either entry
- **WHEN** the migration runs
- **THEN** it prints that `.gitignore` is already clean

### Requirement: Delete .pr-done files
The migration MUST add a step to find and delete any `.pr-done` files under `fab/changes/`.

#### Scenario: .pr-done files exist
- **GIVEN** `fab/changes/some-change/.pr-done` exists
- **WHEN** the migration runs
- **THEN** all `.pr-done` files under `fab/changes/` are deleted
- **AND** a count and status message are printed

#### Scenario: No .pr-done files
- **GIVEN** no `.pr-done` files exist under `fab/changes/`
- **WHEN** the migration runs
- **THEN** it prints that no `.pr-done` files were found

### Requirement: Update migration verification
The Verification section MUST include checks for all three new steps.

#### Scenario: Verification completeness
- **GIVEN** the migration has completed
- **WHEN** the verification steps are checked
- **THEN** they confirm: `.github/copilot-code-review.yml` does not exist, `.gitignore` does not contain stale entries, no `.pr-done` files exist under `fab/changes/`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete copilot-code-review.yml scaffold file | Confirmed from intake #1 — user explicitly requested | S:95 R:90 A:95 D:95 |
| 2 | Certain | Strip Phase 2 and Phase 3 from git-pr-review | Confirmed from intake #2 — user specified stop behavior | S:95 R:85 A:90 D:95 |
| 3 | Certain | Keep Phase 1 / Path A for all reviewer types | Confirmed from intake #3 — existing reviews still processed | S:95 R:90 A:95 D:95 |
| 4 | Certain | Remove .pr-done and .ralph from .gitignore scaffold | Confirmed from intake #4 — user listed both | S:95 R:95 A:90 D:95 |
| 5 | Certain | Append to existing 0.46.0-to-0.47.0.md migration | Confirmed from intake #5 — user specified appending | S:95 R:85 A:95 D:95 |
| 6 | Certain | Remove Path B from Step 3 | Upgraded from intake Confident #6 — unreachable without Phase 2/3 | S:90 R:85 A:95 D:95 |
| 7 | Certain | Update skill description to "human or bot" | Upgraded from intake Confident #7 — Copilot no longer special-cased | S:90 R:95 A:95 D:95 |
| 8 | Certain | Delete existing .pr-done files in migration | Discussed — user explicitly confirmed deletion | S:95 R:90 A:95 D:95 |

8 assumptions (8 certain, 0 confident, 0 tentative, 0 unresolved).
