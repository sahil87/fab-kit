# Spec: Git PR Copilot Fix

**Change**: 260303-4ojc-git-pr-copilot-fix
**Created**: 2026-03-03
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Human review comment triage — this skill targets only `copilot-pull-request-reviewer[bot]` automated comments
- Multi-PR batch processing — operates on the single PR for the current branch
- Comment resolution via GitHub API — fixes are applied locally and pushed; Copilot comments are not programmatically dismissed

## Skills: `/git-pr-fix` (New Standalone Skill)

### Requirement: PR Resolution

The skill SHALL resolve the current branch's PR before any other action. If no PR exists on the current branch, the skill SHALL print "No PR found on this branch." and STOP.

#### Scenario: PR exists on current branch
- **GIVEN** the current branch has an open PR
- **WHEN** `/git-pr-fix` is invoked
- **THEN** the PR number and URL are captured from `gh pr view --json number,url`
- **AND** execution proceeds to Copilot review detection

#### Scenario: No PR on current branch
- **GIVEN** the current branch has no open PR
- **WHEN** `/git-pr-fix` is invoked
- **THEN** the skill prints "No PR found on this branch." and stops
- **AND** no polling or API calls occur

### Requirement: Copilot Review Detection

The skill SHALL detect the Copilot review by polling the GitHub reviews API for a review from `copilot-pull-request-reviewer[bot]`. The owner/repo SHALL be resolved via `gh repo view --json nameWithOwner -q '.nameWithOwner'`.

#### Scenario: Copilot review already present
- **GIVEN** a PR exists and a Copilot review has already been submitted
- **WHEN** the skill polls `gh api repos/{owner}/{repo}/pulls/{number}/reviews`
- **THEN** the review is found immediately on the first poll
- **AND** no further polling occurs

#### Scenario: No Copilot review on first poll
- **GIVEN** a PR exists but no Copilot review has been submitted yet
- **WHEN** the skill performs the first poll
- **THEN** the skill SHALL bail silently with "No Copilot review found — skipping."
- **AND** no further polling occurs (first-poll bail)

#### Scenario: Invoked from git-pr (wait mode)
- **GIVEN** `/git-pr-fix` behavior is invoked inline from `/git-pr` Step 6 (the PR was just created)
- **WHEN** no Copilot review is found on the first poll
- **THEN** the skill SHALL poll every 30 seconds for up to 12 attempts (6 minutes total)
- **AND** if the review arrives within the window, execution proceeds to comment triage
- **AND** if the review does not arrive, the skill prints "Copilot review did not arrive within 6 minutes." and stops

### Requirement: Comment Fetching and Triage

The skill SHALL fetch inline comments from the Copilot review and classify each as actionable or informational.

The skill SHALL fetch comments via `gh api repos/{owner}/{repo}/pulls/{number}/reviews/{review_id}/comments`. Each comment has `path`, `line` (nullable), and `body` fields.

A comment is **actionable** if its body suggests a concrete code change — identifying a specific issue with an implied or explicit fix (e.g., "this value is wrong", "this should be X instead of Y", "missing null check"). A comment is **informational** if it is a summary, praise, general observation, or question without a clear fix action.

#### Scenario: Mix of actionable and informational comments
- **GIVEN** a Copilot review with 6 comments: 4 actionable, 2 informational
- **WHEN** the skill triages comments
- **THEN** the 4 actionable comments are queued for fixing
- **AND** the 2 informational comments are skipped
- **AND** the output notes "{N} comments triaged: {A} actionable, {I} skipped"

#### Scenario: All comments informational
- **GIVEN** a Copilot review where all comments are informational (summary only, no code issues)
- **WHEN** the skill triages comments
- **THEN** no fixes are applied
- **AND** the skill prints "No actionable comments." and stops

### Requirement: Fix Application

For each actionable comment, the skill SHALL read the affected source file (identified by `path`), understand the issue described in `body`, and apply a targeted fix. The skill MUST NOT make unrelated changes to the file beyond what the comment addresses.

#### Scenario: Comment identifies a specific file and issue
- **GIVEN** a Copilot comment on `fab/.kit/skills/git-pr.md` line 42 saying "this reference is stale"
- **WHEN** the skill processes the comment
- **THEN** the skill reads `fab/.kit/skills/git-pr.md`
- **AND** applies a fix to the specific issue at or near line 42
- **AND** does not modify other parts of the file

#### Scenario: Comment has null line number
- **GIVEN** a Copilot comment on `src/lib/resolve.sh` with `line: null` and body describing an issue
- **WHEN** the skill processes the comment
- **THEN** the skill reads the full file and locates the issue from context in the comment body
- **AND** applies the fix

### Requirement: Commit and Push

After all actionable comments are processed, the skill SHALL stage only the modified files, create a single commit, and push. The commit message SHALL be `fix: address copilot review feedback`.

#### Scenario: Fixes applied successfully
- **GIVEN** 3 files were modified to address Copilot comments
- **WHEN** all fixes are complete
- **THEN** only the 3 modified files are staged (not `git add -A`)
- **AND** a single commit is created with message `fix: address copilot review feedback`
- **AND** the commit is pushed to the remote
- **AND** the output shows `✓ Fixed {N} copilot comment(s) across {M} file(s)`

#### Scenario: No fixes needed (idempotent re-run)
- **GIVEN** `/git-pr-fix` was already run and all issues were fixed
- **WHEN** the skill is invoked again
- **THEN** comments are fetched and triaged but no modifications are made
- **AND** the skill prints "No actionable comments." and stops
- **AND** no commit or push occurs

### Requirement: Skill Frontmatter

The skill file SHALL use the following frontmatter:

```yaml
---
name: git-pr-fix
description: "Wait for Copilot review comments on the current PR, then triage and fix them."
allowed-tools: Bash(git:*), Bash(gh:*)
---
```

### Requirement: Autonomous Execution

The skill SHALL be fully autonomous — no questions, no user prompts, no interactive choices. On any error, the skill SHALL print the error and STOP. This matches the `/git-pr` operational model.

#### Scenario: gh CLI not available
- **GIVEN** `gh` is not installed or not in PATH
- **WHEN** the skill attempts to resolve the PR
- **THEN** the skill prints "gh CLI not found." and stops

#### Scenario: API rate limit or network error
- **GIVEN** a `gh api` call fails with a non-zero exit code
- **WHEN** the skill encounters the error
- **THEN** the skill prints the error output and stops
- **AND** no partial commits are made

## Skills: `/git-pr` Step 6 Addition

### Requirement: Auto-Invoke Copilot Fix

After Step 5 (Report), `/git-pr` SHALL execute a Step 6 that invokes `/git-pr-fix` behavior inline. Step 6 SHALL only run when a PR URL is known (from Step 3c creation or from Step 1 existing-PR detection).

Step 6 is **best-effort**: any failure (no Copilot review, API error, fix failure) SHALL NOT fail the overall `/git-pr` invocation. The skill SHALL print the outcome and proceed to the final "Shipped." output.

#### Scenario: PR just created, Copilot review arrives
- **GIVEN** `/git-pr` just created a new PR in Step 3c
- **WHEN** Step 6 executes and waits for the Copilot review
- **THEN** the wait mode is used (polling every 30s, up to 6 minutes)
- **AND** if the review arrives, comments are triaged and fixed
- **AND** "Shipped." is printed after Step 6 completes

#### Scenario: PR just created, no Copilot review arrives
- **GIVEN** `/git-pr` just created a new PR in Step 3c
- **WHEN** Step 6 polls for 6 minutes with no Copilot review
- **THEN** the timeout message is printed
- **AND** "Shipped." is still printed (best-effort, not blocking)

#### Scenario: Existing PR, Copilot review already present
- **GIVEN** `/git-pr` found an existing PR in Step 1 with a Copilot review already submitted
- **WHEN** Step 6 executes
- **THEN** the review is found immediately, comments are triaged and fixed if actionable
- **AND** "Shipped." is printed

#### Scenario: Step 6 encounters an error
- **GIVEN** any error occurs during Step 6 (API failure, commit failure, etc.)
- **WHEN** the error is caught
- **THEN** the error is printed
- **AND** "Shipped." is still printed (best-effort)
- **AND** the exit code of `/git-pr` is unaffected

### Requirement: Wait Mode vs Standalone Mode

When `/git-pr-fix` behavior is invoked from `/git-pr` Step 6, it SHALL use **wait mode** (poll every 30s, max 12 attempts). When `/git-pr-fix` is invoked standalone, it SHALL use **first-poll bail** (check once, bail if not found).

The distinction is: `/git-pr` just created the PR, so the review hasn't had time to arrive. Standalone invocation assumes the PR has existed long enough for the review to have completed.

#### Scenario: Standalone invocation finds review immediately
- **GIVEN** a PR has existed for 10 minutes with a completed Copilot review
- **WHEN** `/git-pr-fix` is invoked standalone
- **THEN** the review is found on the first poll and processed immediately

#### Scenario: Standalone invocation finds no review
- **GIVEN** a PR has existed for 10 minutes but no Copilot review was submitted
- **WHEN** `/git-pr-fix` is invoked standalone
- **THEN** the skill bails on the first poll with "No Copilot review found — skipping."

### Requirement: Rules Section Update

The `/git-pr` Rules section SHALL include: "Step 6 (Copilot fix) is best-effort — never blocks shipping."

## Design Decisions

### Companion Skill Over Embedded Logic
**Decision**: `/git-pr-fix` is a standalone skill, with `/git-pr` auto-invoking its behavior inline.
**Why**: Keeps `/git-pr` focused on shipping. The standalone skill is independently re-runnable (idempotent) for cases where the user wants to run it manually after seeing Copilot comments.
**Rejected**: Embedding all logic directly in `/git-pr` — would bloat a clean shipping skill with polling and fix logic.

### First-Poll Bail for Standalone, Wait for Inline
**Decision**: Standalone mode checks once and bails; inline mode (from `/git-pr`) polls with retry.
**Why**: When invoked standalone, the user is explicitly asking to fix existing comments — if none exist, bail fast. When invoked inline from `/git-pr`, the PR was just created and the review needs time to arrive.
**Rejected**: Always polling (wastes time when review is already present or repo doesn't use Copilot). Always first-poll bail (would never catch a review when auto-invoked from `/git-pr`).

### Single Commit for All Fixes
**Decision**: All fixes go into one commit (`fix: address copilot review feedback`), not per-comment commits.
**Why**: Keeps git history clean. Copilot comments are a single review pass — one response commit is the natural unit.
**Rejected**: Per-comment commits — noisy history, no traceability benefit since the PR links to the review.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Bot login is `copilot-pull-request-reviewer[bot]` | Confirmed from intake #1 — verified via API across 5 PRs | S:95 R:90 A:95 D:95 |
| 2 | Certain | Reviews surface via PR reviews API, not checks API | Confirmed from intake #2 — `gh pr checks` does not list Copilot review | S:95 R:90 A:95 D:95 |
| 3 | Certain | Review comments have path, line (nullable), body fields | Confirmed from intake #4 — verified via reviews/comments endpoint | S:95 R:90 A:95 D:95 |
| 4 | Certain | Standalone skill + auto-invoke from git-pr | Confirmed from intake #5 — user chose separation of concerns | S:95 R:90 A:90 D:95 |
| 5 | Certain | Single commit for all fixes | Confirmed from intake #6 — user agreed | S:90 R:90 A:85 D:90 |
| 6 | Certain | Best-effort from git-pr — failures don't block | Confirmed from intake #7 — user agreed | S:95 R:95 A:90 D:95 |
| 7 | Certain | Commit message: `fix: address copilot review feedback` | Upgraded from intake #10 — conventional commits prefix, clear intent | S:85 R:90 A:80 D:85 |
| 8 | Confident | Poll 30s intervals, max 12 attempts (6 min) for wait mode | Confirmed from intake #8 — covers observed 3-6 min latency window | S:80 R:85 A:75 D:80 |
| 9 | Confident | First-poll bail for standalone, wait mode for inline | Spec-level refinement of intake #9 — two distinct invocation contexts justify different behavior | S:80 R:85 A:75 D:80 |
| 10 | Confident | Actionable vs informational triage heuristic | Agent competence — concrete code suggestions vs summaries/praise is a clear distinction for an LLM | S:75 R:85 A:80 D:75 |

10 assumptions (7 certain, 3 confident, 0 tentative, 0 unresolved).
