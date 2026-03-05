# Spec: Smart Copilot Review Detection

**Change**: 260303-n30u-smart-copilot-review-detection
**Created**: 2026-03-03
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Copilot Review Detection

### Requirement: 3-Phase Copilot Review Detection

The Copilot review detection logic SHALL use a 3-phase approach (detect → request → poll) instead of blind polling. This logic is shared between `/git-pr` Step 6 and `/git-pr-fix` Step 2.

**Phase 1 — Check if already reviewed**: The agent SHALL query `GET /repos/{owner}/{repo}/pulls/{number}/reviews` and check for an entry where `user.login == "copilot-pull-request-reviewer[bot]"`. If found, the agent SHALL capture the review `.id` and skip directly to comment triage (Phase 3 is skipped entirely).

**Phase 2 — Request review**: If Phase 1 found no existing review, the agent SHALL attempt `POST /repos/{owner}/{repo}/pulls/{number}/requested_reviewers` with body `reviewers[]=copilot-pull-request-reviewer[bot]`. Any non-2xx response (including 422, 403, 404, network errors, or any other failure) SHALL be treated as "Copilot review not available." The agent SHALL print `Copilot review not available — skipping.` and exit the Copilot review flow. No polling SHALL occur.

**Phase 3 — Poll for completion**: If Phase 2 succeeded (2xx response), the agent SHALL poll `GET /repos/{owner}/{repo}/pulls/{number}/reviews` for `copilot-pull-request-reviewer[bot]` at the existing interval and attempt limits. On timeout, the agent SHALL print the existing timeout message and exit.

#### Scenario: Copilot review already completed
- **GIVEN** a PR exists and a Copilot review has already been submitted
- **WHEN** the Copilot review detection runs (Phase 1)
- **THEN** the review ID is captured from the reviews API
- **AND** Phases 2 and 3 are skipped entirely
- **AND** comment triage proceeds directly

#### Scenario: Copilot available but review not yet submitted
- **GIVEN** a PR exists and no Copilot review has been submitted yet
- **WHEN** Phase 1 finds no review and Phase 2 POSTs the review request
- **THEN** the POST returns 2xx
- **AND** Phase 3 polls for the review at 30s intervals, max 12 attempts

#### Scenario: Copilot not available on this repo
- **GIVEN** a PR exists on a repo without Copilot review enabled
- **WHEN** Phase 2 POSTs the review request
- **THEN** the POST returns a non-2xx status (e.g., 422)
- **AND** the agent prints `Copilot review not available — skipping.`
- **AND** no polling occurs
- **AND** the overall `/git-pr` flow is not affected (best-effort)

#### Scenario: Network or API error during review request
- **GIVEN** a PR exists and the GitHub API is unreachable or returns an unexpected error
- **WHEN** Phase 2 attempts the POST request
- **THEN** the error is treated identically to "Copilot not available"
- **AND** the agent prints `Copilot review not available — skipping.`
- **AND** no polling occurs

### Requirement: Mode-Specific Behavior

The 3-phase detection logic SHALL be applied consistently across all invocation modes, with Phase 3 behavior varying by mode:

- **Wait mode** (from `/git-pr` Step 6): Phase 3 polls every 30 seconds, max 12 attempts (6 minutes). This is unchanged from current behavior.
- **Standalone mode** (`/git-pr-fix` invoked directly): Phase 3 performs a single check (no polling loop). If no review is present after Phase 2 succeeds, the agent SHALL print `Copilot review requested but not yet available — re-run later.` and exit.

Phases 1 and 2 are identical in both modes.

#### Scenario: Standalone mode with Copilot available but slow
- **GIVEN** `/git-pr-fix` is invoked standalone (not from `/git-pr`)
- **AND** the repo has Copilot review enabled
- **WHEN** Phase 1 finds no existing review and Phase 2 succeeds
- **AND** Phase 3 single-check finds no review yet
- **THEN** the agent prints `Copilot review requested but not yet available — re-run later.`
- **AND** the agent exits without error

#### Scenario: Standalone mode with existing review
- **GIVEN** `/git-pr-fix` is invoked standalone
- **AND** a Copilot review already exists
- **WHEN** Phase 1 finds the review
- **THEN** the agent proceeds directly to comment triage
- **AND** Phases 2 and 3 are skipped

### Requirement: API Login Name Documentation

The skill files SHALL include a comment documenting the Copilot bot login name discrepancy across GitHub API endpoints:

- `GET /requested_reviewers` response: `"login": "Copilot"` (type: Bot)
- `GET /reviews` response: `"login": "copilot-pull-request-reviewer[bot]"`
- `POST /requested_reviewers` request body: `reviewers[]=copilot-pull-request-reviewer[bot]`

This comment SHALL be placed in `/git-pr-fix` Step 2 (the canonical location for detection logic) as an inline note.

#### Scenario: Maintainer reads detection logic
- **GIVEN** a maintainer is reading `/git-pr-fix` Step 2
- **WHEN** they encounter the different login strings in API calls
- **THEN** the inline comment explains the discrepancy
- **AND** no investigation is needed to understand why different strings are used

## Deprecated Requirements

### Blind Polling in `/git-pr-fix` Step 2

**Reason**: Replaced by 3-phase detection. The previous behavior of unconditionally polling (wait mode) or doing a single check (standalone mode) without first attempting to request a review is superseded.

**Migration**: The 3-phase flow covers all previous behavior — Phase 1 handles existing reviews, Phase 2 gates on availability, Phase 3 handles the polling.

### Unconditional Polling in `/git-pr` Step 6

**Reason**: Replaced by 3-phase detection invoked inline. `/git-pr` Step 6 previously described its own polling loop; it now delegates to `/git-pr-fix` Step 2 behavior which includes the 3-phase flow.

**Migration**: Same inline delegation pattern, but the underlying detection logic is now 3-phase.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use POST /requested_reviewers to detect Copilot availability | Confirmed from intake #1 — empirically validated on PR #192 | S:95 R:90 A:95 D:95 |
| 2 | Certain | Treat any non-2xx from POST as "not available" | Confirmed from intake #2 — user explicitly requested uniform error handling | S:95 R:90 A:90 D:95 |
| 3 | Certain | Keep same polling parameters (30s, 12 attempts) for Phase 3 | Confirmed from intake #3 — scope limited to detection gating, not polling tuning | S:90 R:95 A:90 D:95 |
| 4 | Confident | Standalone git-pr-fix attempts POST before single-check | Confirmed from intake #4 — consistent behavior across modes, POST is idempotent | S:70 R:85 A:80 D:75 |
| 5 | Certain | Document login name discrepancy in skill files | Confirmed from intake #5 — empirically confirmed different logins across endpoints | S:95 R:95 A:95 D:95 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).
