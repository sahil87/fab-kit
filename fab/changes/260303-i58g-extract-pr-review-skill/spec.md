# Spec: Extract PR Review Skill

**Change**: 260303-i58g-extract-pr-review-skill
**Created**: 2026-03-04
**Affected memory**: No memory files affected — skill-layer change only

## Non-Goals

- Pipeline orchestration changes (ff/fff extending through git-pr → git-review) — tracked separately
- Resolving review comments from non-GitHub platforms (e.g., Linear, Slack)
- Auto-requesting human reviewers — only Copilot is auto-requested as a fallback

## Skills: `/git-review` Skill

### Requirement: PR Resolution

The skill SHALL resolve the current branch's PR using `gh pr view`. If `gh` is not available, it SHALL print `gh CLI not found.` and STOP. If no PR exists on the current branch, it SHALL print `No PR found on this branch.` and STOP.

The skill SHALL extract `{number}`, `{url}`, `{owner}`, and `{repo}` from the resolved PR for use in subsequent steps.

#### Scenario: PR exists on current branch
- **GIVEN** the current branch has an open PR
- **WHEN** `/git-review` is invoked
- **THEN** the PR number, URL, owner, and repo are captured
- **AND** the skill proceeds to review detection

#### Scenario: No PR on branch
- **GIVEN** the current branch has no PR
- **WHEN** `/git-review` is invoked
- **THEN** the skill prints `No PR found on this branch.` and stops

#### Scenario: gh CLI not installed
- **GIVEN** `gh` is not on PATH
- **WHEN** `/git-review` is invoked
- **THEN** the skill prints `gh CLI not found.` and stops

### Requirement: Review Detection and Routing

The skill SHALL check for existing reviews with comments. If any reviewer (human or bot) has submitted a review with unresolved comments, the skill SHALL proceed directly to comment processing — it SHALL NOT request Copilot or poll.

If no reviews with comments exist, the skill SHALL attempt to assign Copilot as a reviewer via `POST /pulls/{number}/requested_reviewers` with `reviewers[]=copilot-pull-request-reviewer[bot]`. If the POST succeeds, the skill SHALL poll for review completion. If the POST fails (any non-2xx response), the skill SHALL print `No reviews found and Copilot not available — nothing to do.` and STOP.

#### Scenario: Human reviewer has left comments
- **GIVEN** a human reviewer has submitted a review with inline comments
- **WHEN** `/git-review` is invoked
- **THEN** the skill fetches all unresolved comments across all reviewers
- **AND** the skill does NOT request Copilot
- **AND** the skill proceeds to triage and fix

#### Scenario: Copilot review already submitted
- **GIVEN** `copilot-pull-request-reviewer[bot]` has submitted a review
- **WHEN** `/git-review` is invoked
- **THEN** the skill fetches comments from that review
- **AND** the skill does NOT re-request Copilot
- **AND** the skill proceeds to triage and fix

#### Scenario: No reviews — Copilot available
- **GIVEN** no reviews with comments exist on the PR
- **AND** the repo supports Copilot reviewer
- **WHEN** `/git-review` is invoked
- **THEN** the skill requests Copilot as reviewer via POST
- **AND** the skill polls for review completion
- **AND** upon completion, the skill proceeds to triage and fix

#### Scenario: No reviews — Copilot not available
- **GIVEN** no reviews with comments exist on the PR
- **AND** the Copilot reviewer POST returns a non-2xx response
- **WHEN** `/git-review` is invoked
- **THEN** the skill prints `No reviews found and Copilot not available — nothing to do.`
- **AND** the skill stops

### Requirement: Copilot Polling

When Copilot has been requested (and only then), the skill SHALL poll `GET /pulls/{number}/reviews` every 30 seconds for a maximum of 12 attempts (6 minutes total), checking for a review from `copilot-pull-request-reviewer[bot]`.

If a review arrives within the polling window, the skill SHALL capture the review ID and proceed to comment fetching. If the polling window expires without a review, the skill SHALL print `Copilot review did not arrive within 6 minutes.` and STOP.

<!-- Note on API login name discrepancy (empirically confirmed):
     - GET /requested_reviewers: "login": "Copilot" (type: Bot)
     - GET /reviews (once submitted): "login": "copilot-pull-request-reviewer[bot]"
     - POST /requested_reviewers body: reviewers[]=copilot-pull-request-reviewer[bot]
     These are different representations of the same bot across GitHub API endpoints. -->

#### Scenario: Copilot review arrives within polling window
- **GIVEN** Copilot was requested as reviewer
- **WHEN** the skill polls for reviews
- **AND** Copilot submits a review on attempt 4 (after ~90 seconds)
- **THEN** the skill captures the review ID
- **AND** the skill proceeds to comment fetching

#### Scenario: Copilot review times out
- **GIVEN** Copilot was requested as reviewer
- **WHEN** the skill polls for 12 attempts (6 minutes)
- **AND** no Copilot review arrives
- **THEN** the skill prints `Copilot review did not arrive within 6 minutes.`
- **AND** the skill stops

### Requirement: Comment Fetching

The skill SHALL fetch all unresolved review comments on the PR. The fetching strategy depends on the detection path:

**Path A — Existing reviews found**: Fetch all review comments across all reviewers via `GET /pulls/{number}/comments`. This captures comments from all submitted reviews regardless of reviewer.

**Path B — Copilot just requested and reviewed**: Fetch comments from the specific Copilot review via `GET /pulls/{number}/reviews/{review_id}/comments`.

In both paths, comments that are part of a resolved conversation (where the thread has been marked as resolved in the GitHub UI) SHOULD be skipped. The skill SHALL use the `GET /pulls/{number}/comments` endpoint which includes `subject_type` and thread resolution state when available.
<!-- assumed: GitHub API exposes thread resolution state via the pull request comments endpoint — needs verification against actual API response shape -->

#### Scenario: Multiple reviewers with comments
- **GIVEN** both a human reviewer and Copilot have submitted reviews with comments
- **WHEN** the skill fetches comments (Path A)
- **THEN** comments from all reviewers are collected
- **AND** resolved conversation threads are excluded

#### Scenario: Single Copilot review after polling
- **GIVEN** Copilot was just requested and submitted a review
- **WHEN** the skill fetches comments (Path B)
- **THEN** only comments from the Copilot review are fetched

### Requirement: Comment Triage

The skill SHALL classify each fetched comment as **actionable** or **informational**.

- **Actionable**: The comment identifies a specific code issue with an implied or explicit fix (e.g., "This variable is unused", "Missing null check here", "Should use `const` instead of `let`")
- **Informational**: Summary comments, praise, general observations, questions without a clear fix action (e.g., "Looks good overall", "Why was this approach chosen?", "Nice refactor")

The skill SHALL skip informational comments. If all comments are informational, the skill SHALL print `No actionable comments.` and STOP.

The skill SHALL print a triage summary: `{N} comments triaged: {A} actionable, {I} skipped`

#### Scenario: Mix of actionable and informational comments
- **GIVEN** a review contains 5 comments: 3 identify code issues, 2 are general observations
- **WHEN** the skill triages comments
- **THEN** it classifies 3 as actionable and 2 as informational
- **AND** it prints `5 comments triaged: 3 actionable, 2 skipped`

#### Scenario: All comments informational
- **GIVEN** a review contains only summary/praise comments
- **WHEN** the skill triages comments
- **THEN** it prints `No actionable comments.` and stops

### Requirement: Fix Application

For each actionable comment, the skill SHALL:

1. Read the file at `{path}`
2. Understand the issue described in `{body}`
3. If `{line}` is non-null, focus on that area of the file
4. If `{line}` is null, locate the issue from context in the body
5. Apply a targeted fix — the skill SHALL NOT make unrelated changes beyond what the comment addresses

#### Scenario: Comment with line reference
- **GIVEN** an actionable comment on `src/utils.js` at line 42 saying "Missing null check"
- **WHEN** the skill applies a fix
- **THEN** it reads `src/utils.js`, adds a null check at or near line 42
- **AND** it does not modify other parts of the file

#### Scenario: Comment without line reference
- **GIVEN** an actionable comment on `src/config.js` with no line number, body says "The timeout constant should be configurable"
- **WHEN** the skill applies a fix
- **THEN** it reads `src/config.js`, locates the timeout constant from context
- **AND** applies the fix described in the comment

### Requirement: Commit and Push

After all actionable comments are processed, the skill SHALL:

1. Check for modifications via `git status --porcelain`
2. If no modifications exist, print `No changes needed.` and STOP
3. Stage only the specific modified files (`git add {file1} {file2} ...` — NOT `git add -A`)
4. Generate a commit message reflecting the review source:
   - If comments came from `copilot-pull-request-reviewer[bot]`: `fix: address copilot review feedback`
   - If comments came from a human reviewer: `fix: address review feedback from @{username}`
   - If comments came from multiple reviewers: `fix: address PR review feedback`
5. Commit and push
6. If commit or push fails, run `git reset` to clear staged changes, print the error, and STOP

The skill SHALL print `Fixed {N} comment(s) across {M} file(s)` on success.

#### Scenario: Fixes applied from Copilot review
- **GIVEN** 3 actionable Copilot comments were fixed across 2 files
- **WHEN** the skill commits
- **THEN** it stages only the 2 modified files
- **AND** commits with message `fix: address copilot review feedback`
- **AND** pushes to origin
- **AND** prints `Fixed 3 comment(s) across 2 file(s)`

#### Scenario: Fixes applied from human review
- **GIVEN** 2 actionable comments from `@alice` were fixed in 1 file
- **WHEN** the skill commits
- **THEN** commits with message `fix: address review feedback from @alice`
- **AND** prints `Fixed 2 comment(s) across 1 file(s)`

#### Scenario: Commit failure
- **GIVEN** the commit fails (e.g., pre-commit hook rejects)
- **WHEN** the skill attempts to commit
- **THEN** it runs `git reset` to clear staged changes
- **AND** prints the error and stops

## Skills: Remove `/git-pr-fix`

### Requirement: Delete git-pr-fix Skill

The existing `.claude/skills/git-pr-fix/SKILL.md` SHALL be deleted entirely. It is fully replaced by `/git-review`.

#### Scenario: git-pr-fix no longer exists
- **GIVEN** the change has been applied
- **WHEN** a user looks for `/git-pr-fix`
- **THEN** the skill file does not exist
- **AND** `/git-review` provides equivalent and expanded functionality

## Skills: Modify `/git-pr`

### Requirement: Remove Step 6 from git-pr

Step 6 (Auto-Fix Copilot Review) SHALL be removed from `.claude/skills/git-pr/SKILL.md`. The skill SHALL end at Step 5 ("Shipped."). Steps 4, 4b, and 4c (record PR URL, commit status, write sentinel) remain unchanged.

#### Scenario: git-pr ends at Step 5
- **GIVEN** a user runs `/git-pr`
- **WHEN** the PR is created successfully
- **THEN** the skill prints "Shipped." and stops
- **AND** no Copilot review detection or fixing occurs

### Requirement: Remove Step 6 References

All references to Step 6, `/git-pr-fix`, and Copilot auto-fix SHALL be removed from the git-pr skill file, including:

- The Rules section line: "Step 6 (Copilot fix) is best-effort — never blocks shipping"
- Any other cross-references to Step 6 or `/git-pr-fix` within the file

#### Scenario: No residual references
- **GIVEN** the change has been applied
- **WHEN** the git-pr skill file is searched for "Step 6", "git-pr-fix", or "Copilot"
- **THEN** no matches are found

## Deprecated Requirements

### git-pr Step 6 (Auto-Fix Copilot Review)
**Reason**: Copilot-specific workflow bolted onto an otherwise clean commit→push→PR pipeline. Review handling is now a separate concern in `/git-review`.
**Migration**: Use `/git-review` after `/git-pr` to process review comments.

### git-pr-fix Skill
**Reason**: Only handled Copilot reviews with an awkward wait/standalone mode split. Replaced by `/git-review` which handles all reviewer types with a single unified flow.
**Migration**: Use `/git-review` instead.

## Design Decisions

1. **Skill name: `git-review`**
   - *Why*: Short, memorable, pairs naturally with `git-pr`. Describes the action (review handling) rather than the mechanism (fix PR reviews).
   - *Rejected*: `git-fix-pr-reviews` (verbose, overspecific), `git-pr-review` (sounds like it creates a review)

2. **All unresolved comments, not just most recent review**
   - *Why*: User explicitly chose this — avoids missing comments from earlier review rounds that were never addressed. Stateless in the sense that "unresolved" is tracked by GitHub, not by the skill.
   - *Rejected*: Most recent review only (simpler but risks missing older unresolved feedback)

3. **Human reviews take priority over Copilot fallback**
   - *Why*: If a human has already reviewed, that's the review cycle. Requesting Copilot on top would be noise. Copilot is a fallback for when no one has reviewed yet.
   - *Rejected*: Always check both (wasteful, potentially conflicting fixes)

4. **Copilot request as feature probe**
   - *Why*: The POST to request Copilot acts as a feature probe — if the repo doesn't support it, the POST fails and the skill exits gracefully. No configuration needed.
   - *Rejected*: Config flag for Copilot support (adds maintenance burden, stale config risk)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skill name is `git-review` | User explicitly specified during spec discussion | S:95 R:90 A:95 D:95 |
| 2 | Certain | Remove Step 6 from git-pr entirely | Confirmed from intake #2 — user explicitly decided | S:95 R:70 A:90 D:95 |
| 3 | Certain | Replace git-pr-fix with git-review | Confirmed from intake #3 — user said replace | S:95 R:75 A:90 D:95 |
| 4 | Certain | Support all reviewer types (human + bot) | Confirmed from intake #4 — user clarified | S:90 R:80 A:85 D:90 |
| 5 | Certain | Process all unresolved comments, not just most recent review | User explicitly chose (b) during spec discussion | S:95 R:75 A:95 D:95 |
| 6 | Certain | Human reviews found → skip Copilot request | User explicitly confirmed during spec discussion | S:95 R:80 A:95 D:95 |
| 7 | Certain | Copilot request as fallback when no reviews exist | User described this flow explicitly during spec discussion | S:95 R:75 A:90 D:90 |
| 8 | Confident | Polling logic moves entirely into git-review | Upgraded from intake #5 — follows naturally from user's described flow | S:85 R:75 A:80 D:80 |
| 9 | Confident | Standalone skill (not sub-skill of git-pr) | Confirmed from intake #6 — user said separate skill | S:85 R:80 A:80 D:85 |
| 10 | Tentative | GitHub API exposes thread resolution state for filtering resolved comments | PR comments endpoint may not directly expose resolution state — needs verification | S:50 R:80 A:45 D:50 |
| 11 | Confident | Polling defaults: 30s interval, 12 attempts (6 min) | User confirmed these defaults in the flow description | S:85 R:85 A:80 D:85 |

11 assumptions (7 certain, 3 confident, 1 tentative, 0 unresolved).
