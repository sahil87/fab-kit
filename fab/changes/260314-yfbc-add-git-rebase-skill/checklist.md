# Quality Checklist: Add Git Rebase Skill

**Change**: 260314-yfbc-add-git-rebase-skill
**Generated**: 2026-03-15
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Skill File Structure: frontmatter has name, description, allowed-tools per spec
- [x] CHK-002 Git Repository Guard: Step 1 checks `git rev-parse --is-inside-work-tree`
- [x] CHK-003 Branch Guard: Step 2 rejects main/master with appropriate error message
- [x] CHK-004 Uncommitted Changes Detection: Step 3 uses `git status --porcelain` and presents stash-or-abort options
- [x] CHK-005 Main Branch Auto-Detection: Step 4 uses `git rev-parse --verify main` with master fallback
- [x] CHK-006 Fetch and Rebase: Step 4 runs `git fetch origin {main_branch}` then `git rebase origin/{main_branch}`
- [x] CHK-007 Stash Safety: stash push before rebase, stash pop after, with error handling for both
- [x] CHK-008 No Fab State Modification: key properties confirm no .fab-status.yaml or .status.yaml changes
- [x] CHK-009 Skill Deployment: deployed copy exists at `.claude/skills/git-rebase/SKILL.md`

## Scenario Coverage
- [x] CHK-010 Not in git repo scenario: reports error and stops
- [x] CHK-011 On main/master scenario: reports error with branch name
- [x] CHK-012 Clean working tree scenario: proceeds directly to fetch/rebase
- [x] CHK-013 Uncommitted changes — stash flow: stash → rebase → pop
- [x] CHK-014 Uncommitted changes — abort flow: no git operations performed
- [x] CHK-015 Fetch failure scenario: reports error, pops stash if applicable
- [x] CHK-016 Rebase conflict scenario: reports conflict with resolution guidance
- [x] CHK-017 Stash pop conflict scenario: reports conflicts and stops

## Edge Cases & Error Handling
- [x] CHK-018 Fetch failure with stash: stash is popped before stopping
- [x] CHK-019 Rebase conflict with stash: notes stash pop needed after resolution

## Code Quality
- [x] CHK-020 Pattern consistency: skill structure matches git-branch and git-pr patterns
- [x] CHK-021 No unnecessary duplication: reuses standard git commands, no custom utilities

## Documentation Accuracy
- [x] CHK-022 Error handling table is complete and matches behavior sections
- [x] CHK-023 Key properties table accurately reflects skill behavior

## Cross References
- [x] CHK-024 Spec requirements all have corresponding checklist coverage
- [x] CHK-025 Skill description in frontmatter matches system reminder listing

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
