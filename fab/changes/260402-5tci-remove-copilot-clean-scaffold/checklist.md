# Quality Checklist: Remove Copilot Integration and Clean Stale Scaffold

**Change**: 260402-5tci-remove-copilot-clean-scaffold
**Generated**: 2026-04-02
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Copilot scaffold deleted: `src/kit/scaffold/.github/copilot-code-review.yml` does not exist
- [x] CHK-002 .gitignore fragment cleaned: `src/kit/scaffold/fragment-.gitignore` contains no `/.ralph` or `fab/changes/**/.pr-done` lines
- [x] CHK-003 Phase 2 removed: `src/kit/skills/git-pr-review.md` contains no Copilot review request (`POST /requested_reviewers`)
- [x] CHK-004 Phase 3 removed: No polling loop for `copilot-pull-request-reviewer[bot]` in skill
- [x] CHK-005 Path B removed: Step 3 only contains Path A (fetch all comments)
- [x] CHK-006 Commit message simplified: No Copilot-specific branch in Step 5 commit logic
- [x] CHK-007 Phase tracking updated: `waiting` removed from sub-state table, `received` description updated
- [x] CHK-008 Skill metadata updated: Description says "human or bot", not "human or Copilot"
- [x] CHK-009 Migration step 5: Remove `.github/copilot-code-review.yml` with exists/absent handling
- [x] CHK-010 Migration step 6: Remove `/.ralph` and `fab/changes/**/.pr-done` from `.gitignore`
- [x] CHK-011 Migration step 7: Delete `.pr-done` files under `fab/changes/` with count reporting
- [x] CHK-012 Migration verification: Updated to cover all three new steps

## Behavioral Correctness

- [x] CHK-013 No-reviews stop: When no reviews exist, skill prints stop message and completes as `done`
- [x] CHK-014 Existing reviews processed: Phase 1 / Path A still handles all reviewer types normally

## Removal Verification

- [x] CHK-015 No Copilot references: No `copilot-pull-request-reviewer` anywhere in skill file
- [x] CHK-016 No API login comment: HTML comment about Copilot login name discrepancy removed
- [x] CHK-017 No Phase 2/3 remnants: No `requested_reviewers` POST or polling loop remains

## Scenario Coverage

- [x] CHK-018 No reviews scenario: Skill flow terminates after Phase 1 with stop message
- [x] CHK-019 Reviews with comments scenario: Path A fetch and triage works as before
- [x] CHK-020 Migration file exists scenario: Steps handle presence/absence gracefully

## Code Quality

- [x] CHK-021 Pattern consistency: Skill markdown follows existing section structure and formatting
- [x] CHK-022 No unnecessary duplication: Migration steps follow same pattern as existing steps 1-4

## Documentation Accuracy

- [x] CHK-023 Step 2 routing comment: HTML comment about body-only reviews updated (no Copilot fall-through reference)
- [x] CHK-024 Step 6 stage outcomes: Copilot-specific references removed from stage completion logic

## Cross References

- [x] CHK-025 fab-fff Step 9 description: Verify skill description in `src/kit/skills/fab-fff.md` still compatible (references git-pr-review behavior) — reworked: lines 132 and 136 now describe generic detect/triage/stop behavior with no Copilot references

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
