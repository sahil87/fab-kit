# Quality Checklist: Absorb Ship Command

**Change**: 260222-n811-absorb-ship-command
**Generated**: 2026-02-22
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Skill File: `fab/.kit/skills/git-pr.md` exists with correct frontmatter (name, description, model_tier: fast, allowed-tools)
- [x] CHK-002 Autonomous Execution: Skill prompt instructs commit → push → PR with no user interaction
- [x] CHK-003 Branch Guard: Skill includes guard against running on main/master
- [x] CHK-004 Commit Message Generation: Skill instructs commit message from diff + existing style, no Co-Authored-By
- [x] CHK-005 Error Handling: Skill instructs fail-fast on each step
- [x] CHK-006 Progress Output: Skill specifies pipeline-style output with checkmarks
- [x] CHK-007 Replace Ship Command: `run.sh` sends `/git-pr` instead of `/changes:ship pr`

## Behavioral Correctness

- [x] CHK-008 Skip Logic: Skill handles no-uncommitted-changes, PR-already-exists, and already-fully-shipped cases
- [x] CHK-009 Push Strategy: Skill uses `-u` flag when no upstream is set

## Scenario Coverage

- [x] CHK-010 Clean Pipeline Run: Skill covers the full commit+push+PR path
- [x] CHK-011 No Uncommitted Changes: Skill skips commit step when nothing to commit
- [x] CHK-012 PR Already Exists: Skill reports existing PR URL without creating a new one
- [x] CHK-013 Already Fully Shipped: Skill reports "Already shipped" when nothing to do
- [x] CHK-014 Invoked on Main: Skill errors with clear message

## Edge Cases & Error Handling

- [x] CHK-015 Push Rejected: Skill stops without attempting PR creation
- [x] CHK-016 gh CLI Missing: Skill checks for gh availability

## Code Quality

- [x] CHK-017 Pattern consistency: Skill frontmatter follows same format as existing skills; preamble reference included
- [x] CHK-018 No unnecessary duplication: Skill doesn't duplicate patterns already handled by git/gh defaults

## Documentation Accuracy

- [x] CHK-019 Pipeline log message updated from "Sending /changes:ship pr" to "Sending /git-pr"

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
