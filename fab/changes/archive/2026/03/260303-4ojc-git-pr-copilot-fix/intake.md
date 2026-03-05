# Intake: Git PR Copilot Fix

**Change**: 260303-4ojc-git-pr-copilot-fix
**Created**: 2026-03-03
**Status**: Draft

## Origin

> Add a companion skill /git-pr-fix that waits for Copilot review comments on a PR, triages them, and auto-fixes them. Also modify /git-pr to auto-invoke git-pr-fix as a best-effort final step after PR creation.

Conversational — originated from a `/fab-discuss` session exploring whether to add a "poll and fix review comments" instruction to `/git-pr`. After discussion, the approach was refined: separate companion skill + auto-invoke, wait for Copilot check (not blind polling), best-effort from git-pr.

Key decisions from discussion:
- Bot identity verified: `copilot-pull-request-reviewer[bot]` (confirmed via `gh api` across 5 recent PRs)
- Review latency measured: 3-6 minutes consistently (PRs 187-191)
- Copilot review surfaces as a PR review (via reviews API), NOT as a check run (`gh pr checks` does not list it)
- Comments have `path`, `line` (sometimes null), and `body` fields
- Polling approach: 30s intervals, bail after ~6 minutes
- First-poll bail: if no Copilot review appears on first check AND repo has no history of Copilot reviews, skip silently (repo doesn't have it enabled)

## Why

After `/git-pr` creates a PR, GitHub Copilot's automated reviewer posts inline comments within 3-6 minutes. These comments flag real issues (spec mismatches, potential bugs, stale references) but currently require manual triage — opening the PR, reading each comment, making fixes, committing, and pushing. This interrupts flow and adds ~10-15 minutes of mechanical work per PR.

Automating this triage-and-fix loop saves time on every PR and keeps the developer in the CLI workflow. Since Copilot comments are automated and concrete (not conversational like human reviews), auto-fixing is low-risk — each fix is a single additional commit that's easy to review or revert.

## What Changes

### New skill: `/git-pr-fix` (`fab/.kit/skills/git-pr-fix.md`)

A standalone autonomous skill that waits for Copilot review comments on the current branch's PR, triages them, and fixes actionable ones.

**Step 1: Resolve PR**
- Get current branch via `git branch --show-current`
- Find existing PR via `gh pr view --json number,url`
- If no PR → print "No PR found on this branch" and STOP

**Step 2: Wait for Copilot Review**
- Get `{owner}/{repo}` via `gh repo view --json nameWithOwner -q '.nameWithOwner'`
- Poll `gh api repos/{owner}/{repo}/pulls/{number}/reviews` looking for a review where `user.login == "copilot-pull-request-reviewer[bot]"`
- **First-check bail**: If no Copilot review on first poll, bail silently — repo likely doesn't have Copilot reviews enabled. No waiting, no error.
- **Polling**: If Copilot review not yet present but repo is expected to have it (determined by presence of a pending/in-progress state or other signal), poll every 30 seconds, max 12 attempts (6 minutes total)
- **Timeout**: If review doesn't arrive within 6 minutes → print message and STOP

**Step 3: Fetch and Triage Comments**
- Get the Copilot review ID from the reviews response
- Fetch inline comments: `gh api repos/{owner}/{repo}/pulls/{number}/reviews/{review_id}/comments`
- For each comment with `path` and `body`:
  - Read the affected source file
  - Determine if the comment is actionable (suggests a concrete fix) vs. informational (summary, praise, observations with no clear action)
  - For actionable comments: apply the suggested fix
  - For informational: skip
- Track which files were modified and which comments were addressed

**Step 4: Commit and Push**
- If no fixes were made → print "No actionable comments" and STOP
- Stage only the specific modified files (not `git add -A`)
- Commit with message: `fix: address copilot review feedback`
- Push
- Print summary: `✓ Fixed {N} copilot comment(s) across {M} file(s)`

**Idempotency**: On re-run, Step 2 finds the existing Copilot review immediately (no waiting). Step 3 re-reads comments but the underlying issues are already fixed → no modifications → Step 4 prints "No actionable comments" and exits cleanly.

**Frontmatter**:
```yaml
---
name: git-pr-fix
description: "Wait for Copilot review comments on the current PR, then triage and fix them."
allowed-tools: Bash(git:*), Bash(gh:*)
---
```

### Modify `/git-pr` (`fab/.kit/skills/git-pr.md`)

Add a **Step 6: Auto-Fix Copilot Review** after Step 5 (Report):

- Only runs if a PR URL is known (from Step 3c creation or Step 1 existing PR)
- Executes the `/git-pr-fix` behavior inline (Steps 1-4 from above, skipping Step 1 since PR is already known)
- **Best-effort**: failures in Step 6 do NOT fail the overall `/git-pr` invocation. Any error → print the outcome and proceed to "Shipped."
- Update the Rules section to document this behavior

### Update `/git-pr` Rules section

Add: "Step 6 (Copilot fix) is best-effort — never blocks shipping"

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document new `/git-pr-fix` skill and the `/git-pr` Step 6 addition

## Impact

- **`fab/.kit/skills/git-pr-fix.md`** — new file
- **`fab/.kit/skills/git-pr.md`** — modified (Step 6 + Rules update)
- **No script changes** — uses existing `gh` CLI and `git` commands only
- **No template changes** — skill is autonomous, no artifact templates needed
- **Distribution**: Both files are inside `fab/.kit/`, so changes ship with the kit

## Open Questions

- Should `/git-pr-fix` attempt to detect whether the repo has Copilot reviews enabled (e.g., by checking recent PRs for the bot) or just bail on first-poll empty? First-poll bail is simpler and sufficient — if the review isn't there immediately and we have no reason to wait, don't wait.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Bot name is `copilot-pull-request-reviewer[bot]` | Discussed — verified via `gh api` across 5 recent PRs (187-191) | S:95 R:90 A:95 D:95 |
| 2 | Certain | Copilot review appears as PR review, not check run | Discussed — confirmed `gh pr checks` does not list it, reviews API does | S:95 R:90 A:95 D:95 |
| 3 | Certain | Review latency is 3-6 minutes | Discussed — measured across PRs 187-191 | S:95 R:85 A:90 D:95 |
| 4 | Certain | Comments have path, line (nullable), body fields | Discussed — verified via reviews comments API on PR 191 | S:95 R:90 A:95 D:95 |
| 5 | Certain | Standalone skill + auto-invoke from git-pr | Discussed — user chose this over embedding directly in git-pr | S:95 R:90 A:90 D:95 |
| 6 | Certain | Single commit for all fixes, not per-comment | Discussed — user agreed | S:90 R:90 A:85 D:90 |
| 7 | Certain | Best-effort from git-pr — failures don't block shipping | Discussed — user agreed | S:95 R:95 A:90 D:95 |
| 8 | Confident | Poll every 30s, max 12 attempts (6 min) | Discussed — covers observed 3-6 min window with margin. Could need adjustment if Copilot latency changes | S:80 R:85 A:75 D:80 |
| 9 | Confident | First-poll bail when no review found immediately | Discussed — simplest detection for repos without Copilot. Could miss slow-starting reviews on first PR | S:75 R:85 A:70 D:75 |
| 10 | Confident | Commit message: `fix: address copilot review feedback` | Reasonable default matching conventional commits. User may prefer different wording | S:70 R:90 A:75 D:75 |
| 11 | Confident | Skill frontmatter allows `Bash(git:*), Bash(gh:*)` only | Matches git-pr pattern. Copilot fix needs file reads too — but Read tool access is implicit | S:80 R:90 A:80 D:85 |

11 assumptions (7 certain, 4 confident, 0 tentative, 0 unresolved).
