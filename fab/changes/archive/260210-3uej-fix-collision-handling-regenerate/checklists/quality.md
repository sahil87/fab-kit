# Quality Checklist: Fix collision handling in fab-new to regenerate 4-char component

**Change**: 260210-3uej-fix-collision-handling-regenerate
**Generated**: 2026-02-10
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Collision Retry SHALL Regenerate Random Component: `fab/.kit/skills/fab-new.md` Step 2 item 3 describes regenerating the entire 4-character random component (not appending)
- [x] CHK-002 Documentation SHALL Reflect Regeneration Behavior: Step 2 item 3 explicitly states "regenerate the 4-character random component (`{XXXX}`) and retry"

## Behavioral Correctness

- [x] CHK-003 Instruction changed from append to regenerate: The text no longer mentions "append an additional random character" — verify old wording is completely replaced

## Removal Verification

- [x] CHK-004 Append behavior removed from documentation: Confirm no mention of "append" or "additional character" remains in the collision handling section

## Scenario Coverage

- [x] CHK-005 Collision During Folder Creation: Documentation describes what happens when a collision is detected (regenerate, retry with new name)
- [x] CHK-006 Format Invariant Preserved: Documentation makes clear that the retry maintains exactly 4 characters in the `{XXXX}` component
- [x] CHK-007 Skill Documentation Accuracy: Step 2 item 3 matches the requirement exactly — "regenerate the 4-character random component (`{XXXX}`) and retry"

## Edge Cases & Error Handling

- [x] CHK-008 Multiple consecutive collisions: Documentation implies regeneration can be repeated (doesn't suggest append after first collision)

## Documentation Accuracy

- [x] CHK-009 No stale references: Ensure no other sections of fab-new.md reference the old "append" behavior
- [x] CHK-010 Symlink consistency: `.agents/skills/fab-new/SKILL.md` reflects the fix (symlink to updated file)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
