# Quality Checklist: git-branch Standalone Fallback

**Change**: 260225-jwa3-git-branch-standalone-fallback
**Generated**: 2026-02-25
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Fallback to literal branch name: Skill uses raw argument as branch name when `changeman.sh resolve` fails
- [x] CHK-002 Skip branch prefix: Standalone branches do not have `git.branch_prefix` applied
- [x] CHK-003 Existing branch switch: Skill switches to existing standalone branch instead of failing
- [x] CHK-004 Context-dependent action: Standalone branches use same create/adopt/skip flow as change branches
- [x] CHK-005 Feedback message: Skill prints `No matching change found — using standalone branch '{name}'` when fallback activates

## Behavioral Correctness
- [x] CHK-006 Change resolution precedence: `changeman.sh resolve` is tried first; fallback only on failure
- [x] CHK-007 No-argument behavior unchanged: Omitting argument still resolves from `fab/current`, no fallback attempted

## Scenario Coverage
- [x] CHK-008 Scenario: argument matches fab change → resolved normally, no fallback
- [x] CHK-009 Scenario: argument matches no change, on main → standalone branch created
- [x] CHK-010 Scenario: argument matches no change, branch exists → switched to existing
- [x] CHK-011 Scenario: already on standalone branch → no-op, "already active"

## Edge Cases & Error Handling
- [x] CHK-012 Error handling table updated with standalone fallback row

## Code Quality
- [x] CHK-013 Pattern consistency: New skill markdown follows the existing step/section structure of `git-branch.md`
- [x] CHK-014 No unnecessary duplication: Standalone path reuses existing Step 5 logic, no copy-paste

## Documentation Accuracy
- [x] CHK-015 Error handling table accurately describes the fallback behavior

## Cross References
- [x] CHK-016 Spec requirements map 1:1 to skill behavior — no requirement left unimplemented

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
