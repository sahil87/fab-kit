# Intake: Extract PR Review Skill

**Change**: 260303-i58g-extract-pr-review-skill
**Created**: 2026-03-03
**Status**: Draft

## Origin

> Discussion during `/fab-discuss` session. User identified that `/git-pr` Step 6 (Copilot auto-fix) is non-generic — it assumes a specific external setup (GitHub Copilot reviewer) and is bolted onto an otherwise clean commit→push→PR pipeline. User decided to extract review handling into a standalone skill that supports both Copilot and manual reviews, replacing the existing `/git-pr-fix` skill.

## Why

The current architecture has two problems:

1. **`/git-pr` has a Copilot-specific workflow stapled onto it.** Step 6 is "best-effort" by its own admission, has its own wait/poll lifecycle, its own failure modes, and doesn't affect the exit status — all signs it doesn't belong in the PR creation skill. Users without Copilot get dead code on every invocation.

2. **`/git-pr-fix` only handles Copilot reviews.** Manual/human review comments use the same GitHub API shape (PR review comments) but there's no skill to triage and fix those. The polling/waiting logic is split awkwardly — wait mode lives in `/git-pr` Step 6, standalone mode lives in `/git-pr-fix`.

If we don't fix this: `/git-pr` remains bloated with external assumptions, human review comments require fully manual handling, and the wait/standalone mode split stays confusing.

## What Changes

### Remove `/git-pr-fix` skill

Delete the existing `.claude/skills/git-pr-fix/SKILL.md` skill entirely. It is replaced by the new skill below.

### Create `/git-fix-pr-reviews` skill

New standalone skill at `.claude/skills/git-fix-pr-reviews/SKILL.md` with these responsibilities:

- **Resolve PR** — same as current git-pr-fix Step 1 (detect current branch's PR via `gh pr view`)
- **Wait for reviews** — consolidated polling logic. Poll `GET /pulls/{number}/reviews` at intervals, looking for any review (not just Copilot). Configurable timeout. This absorbs the wait mode that currently lives in `/git-pr` Step 6
- **Fetch and triage comments** — from the most recent review. Classify as actionable vs informational. Support both Copilot bot reviews and human reviews (same API, different `user.login` values)
- **Fix actionable comments** — read file, understand issue, apply targeted fix
- **Commit and push** — stage only modified files, commit with descriptive message, push

Key design differences from current `git-pr-fix`:
- **Not Copilot-specific**: works with any GitHub PR review (Copilot, human, other bots)
- **Self-contained waiting**: the skill itself handles polling/waiting rather than relying on `/git-pr` to orchestrate wait mode
- **No wait/standalone mode split**: single behavior — poll for pending reviews, process what's there, exit
- **Review source awareness**: commit message reflects who reviewed (e.g., "fix: address copilot review feedback" vs "fix: address review feedback from @username")

### Remove Step 6 from `/git-pr`

Strip Step 6 (Auto-Fix Copilot Review) from `.claude/skills/git-pr/SKILL.md`. The skill ends at Step 5 ("Shipped."). The Step 4 series (record PR URL, commit status, write sentinel) remains unchanged.

Also remove Step 6 references from:
- The Rules section ("Step 6 (Copilot fix) is best-effort — never blocks shipping")
- Any other cross-references within git-pr

### Update cross-references in `/git-pr`

Remove any remaining references to `/git-pr-fix` or Step 6 within the git-pr skill file (Rules section, etc.). Pipeline orchestration changes (ff/fff extending through git-pr → git-fix-pr-reviews) are out of scope — tracked as a separate change.

## Affected Memory

No memory files affected — this is a skill-layer change with no spec-level behavior changes to document.

## Impact

- `.claude/skills/git-pr-fix/SKILL.md` — deleted (replaced)
- `.claude/skills/git-fix-pr-reviews/SKILL.md` — new skill file
- `.claude/skills/git-pr/SKILL.md` — modified (Step 6 removed, rules updated)
- `.claude/skills/git-pr-fix` referenced in `/git-pr` Step 6 — reference removed
- `fab/backlog.md` items [a4v0] and [9yvv] are related but not addressed by this change

## Open Questions

- Should the new skill request a Copilot review if none exists (current Phase 2 behavior), or only process reviews that are already there?
- What timeout/interval defaults for polling? Current: 30s intervals, 12 attempts (6 min). Is this appropriate for human reviews which take longer?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skill name is `git-fix-pr-reviews` | Discussed — user specified this name | S:95 R:90 A:95 D:95 |
| 2 | Certain | Remove Step 6 from git-pr entirely | Discussed — user explicitly decided this | S:95 R:70 A:90 D:95 |
| 3 | Certain | Replace git-pr-fix, not extend it | Discussed — user said "get rid of git-pr-fix" and replace with new skill | S:95 R:75 A:90 D:95 |
| 4 | Certain | Support manual/human reviews too | Discussed — user said "we can use it for manual review also" | S:90 R:80 A:85 D:90 |
| 5 | Confident | Polling logic moves entirely into new skill | Discussed — user said "move the waiting part into this skill itself" | S:85 R:75 A:80 D:80 |
| 6 | Confident | New skill is standalone, not a sub-skill | Discussed — user said "instead of being a subskill can be separate skill" | S:85 R:80 A:80 D:85 |
| 7 | Tentative | Keep Copilot review request (Phase 2 POST) in the new skill | Current behavior requests Copilot review if not present — unclear if user wants to keep this | S:50 R:70 A:55 D:50 |
| 8 | Tentative | Polling defaults stay at 30s/12 attempts for Copilot, but may need different defaults for human reviews | Human reviews take much longer than bot reviews — same timeout may not make sense | S:45 R:80 A:50 D:45 |

8 assumptions (4 certain, 2 confident, 2 tentative, 0 unresolved).