# Quality Checklist: Extract PR Review Skill

**Change**: 260303-i58g-extract-pr-review-skill
**Generated**: 2026-03-04
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 PR Resolution: Skill resolves PR number, URL, owner, repo from current branch via `gh pr view`
- [x] CHK-002 Review Detection: Skill checks for existing reviews with comments before attempting Copilot
- [x] CHK-003 Copilot Fallback: When no reviews exist, skill POSTs to request Copilot and handles failure gracefully
- [x] CHK-004 Copilot Polling: Polls every 30s for up to 12 attempts (6 min) when Copilot requested
- [x] CHK-005 Comment Fetching: Path A fetches all unresolved comments; Path B fetches from specific Copilot review
- [x] CHK-006 Comment Triage: Classifies comments as actionable vs informational with summary output
- [x] CHK-007 Fix Application: Reads file, applies targeted fix per comment body/line, no unrelated changes
- [x] CHK-008 Commit and Push: Stages specific files, reviewer-aware commit message, push, git reset on failure
- [x] CHK-009 git-pr Step 6 Removed: No Step 6 section, no Step 6 references in Rules or elsewhere
- [x] CHK-010 git-pr-fix Deleted: `.claude/skills/git-pr-fix/SKILL.md` no longer exists

## Behavioral Correctness

- [x] CHK-011 Human review priority: When human reviews exist, Copilot is NOT requested
- [x] CHK-012 Copilot only as fallback: POST to request Copilot only when no reviews with comments found
- [x] CHK-013 All unresolved comments: Fetches comments across all reviewers, not just most recent review
- [x] CHK-014 Reviewer-aware commit: Message reflects source (copilot, @username, or generic "PR review")

## Removal Verification

- [x] CHK-015 git-pr-fix skill file deleted: `.claude/skills/git-pr-fix/SKILL.md` does not exist
- [x] CHK-016 git-pr Step 6 removed: No "Step 6", "Auto-Fix Copilot", or "git-pr-fix" text in git-pr skill
- [x] CHK-017 git-pr Rules cleaned: "Step 6 (Copilot fix) is best-effort" line removed from Rules section

## Scenario Coverage

- [x] CHK-018 No PR scenario: Skill prints "No PR found on this branch." and stops
- [x] CHK-019 No gh CLI scenario: Skill prints "gh CLI not found." and stops
- [x] CHK-020 Copilot not available scenario: Skill prints "No reviews found and Copilot not available" and stops
- [x] CHK-021 Copilot timeout scenario: Skill prints timeout message after 6 minutes
- [x] CHK-022 All informational scenario: Skill prints "No actionable comments." and stops
- [x] CHK-023 Commit failure scenario: Runs `git reset`, prints error, stops

## Edge Cases & Error Handling

- [x] CHK-024 No modifications after fixes: Prints "No changes needed." and stops
- [x] CHK-025 Multiple reviewers: Comments from all reviewers collected in Path A
- [x] CHK-026 Comment without line reference: Skill locates issue from body context

## Code Quality

- [x] CHK-027 Pattern consistency: Skill file follows structure and conventions of existing skills (git-pr, git-pr-fix)
- [x] CHK-028 No unnecessary duplication: Common patterns (gh commands, error handling) consistent with git-pr

## Documentation Accuracy

- [x] CHK-029 Skill frontmatter: name, description, allowed-tools correctly set
- [x] CHK-030 Rules section: Matches behavioral requirements (autonomous, fail-fast, idempotent, targeted)

## Cross References

- [x] CHK-031 No stale references: No remaining references to git-pr-fix or Step 6 across modified files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
