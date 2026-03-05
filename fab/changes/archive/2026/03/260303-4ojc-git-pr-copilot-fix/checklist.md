# Quality Checklist: Git PR Copilot Fix

**Change**: 260303-4ojc-git-pr-copilot-fix
**Generated**: 2026-03-03
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 PR Resolution: `/git-pr-fix` resolves current branch PR via `gh pr view` and bails when no PR exists
- [x] CHK-002 Copilot Review Detection: Skill detects `copilot-pull-request-reviewer[bot]` reviews via GitHub reviews API
- [x] CHK-003 First-Poll Bail: Standalone mode bails silently when no Copilot review found on first poll
- [x] CHK-004 Wait Mode: Inline invocation (from git-pr) polls every 30s for up to 12 attempts
- [x] CHK-005 Comment Fetching: Comments fetched via reviews/{id}/comments endpoint with path, line, body fields
- [x] CHK-006 Triage Logic: Actionable vs informational classification implemented
- [x] CHK-007 Fix Application: Affected files read and targeted fixes applied per comment
- [x] CHK-008 Commit and Push: Single commit `fix: address copilot review feedback` with only modified files staged
- [x] CHK-009 git-pr Step 6: Auto-invokes git-pr-fix behavior inline after PR creation
- [x] CHK-010 git-pr Best-Effort: Step 6 failures do not fail the overall git-pr invocation

## Behavioral Correctness
- [x] CHK-011 Standalone vs Wait Mode: Standalone uses first-poll bail; inline (from git-pr) uses wait mode with polling
- [x] CHK-012 Autonomous Execution: No questions, no prompts — matches git-pr operational model

## Scenario Coverage
- [x] CHK-013 Scenario: PR exists, Copilot review already present → found immediately, no polling
- [x] CHK-014 Scenario: No PR on branch → "No PR found on this branch." and stop
- [x] CHK-015 Scenario: All comments informational → "No actionable comments." and stop
- [x] CHK-016 Scenario: Idempotent re-run → comments re-triaged but no modifications, clean exit

## Edge Cases & Error Handling
- [x] CHK-017 gh CLI not found → prints error and stops
- [x] CHK-018 API error → prints error and stops, no partial commits
- [x] CHK-019 Timeout (6 min, no review) → prints timeout message and stops
- [x] CHK-020 Null line number in comment → skill reads full file and locates issue from context

## Code Quality
- [x] CHK-021 Pattern consistency: Skill markdown follows existing git-pr.md structure (frontmatter, step numbering, output format)
- [x] CHK-022 No unnecessary duplication: gh API patterns reused consistently, no redundant resolution logic

## Documentation Accuracy
- [x] CHK-023 git-pr-fix.md: All steps, scenarios, and edge cases from spec are reflected in skill file
- [x] CHK-024 git-pr.md: Step 6 addition is consistent with existing step numbering and output format

## Cross References
- [x] CHK-025 git-pr.md Rules section includes the new best-effort rule
- [x] CHK-026 Bot name `copilot-pull-request-reviewer[bot]` used consistently across both skill files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
