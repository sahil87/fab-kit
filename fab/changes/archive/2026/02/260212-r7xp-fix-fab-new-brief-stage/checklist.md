# Quality Checklist: Fix fab-new premature brief completion

**Change**: 260212-r7xp-fix-fab-new-brief-stage
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Step 8 removed: `fab/.kit/skills/fab-new.md` no longer contains a "Mark Brief Complete" step or any `progress.brief: done` / `progress.spec: active` transition logic
- [x] CHK-002 Step renumbered: Former Step 9 ("Activate Change via `/fab-switch`") is now Step 8 with correct heading
- [x] CHK-003 Next Steps table updated: `_context.md` shows `brief active` for `/fab-new` row
- [x] CHK-004 Planning-skills doc updated: Change Initialization list has 4 steps without "mark brief complete"

## Behavioral Correctness

- [x] CHK-005 fab-new ends with brief active: After the fix, `/fab-new`'s behavior flow ends with `progress.brief: active` — no code path sets it to `done`
- [x] CHK-006 fab-continue handles transition: `/fab-continue` stage guard for `brief` active generates `spec.md` and sets `brief: done`, `spec: active` — this was already correct and remains unchanged

## Removal Verification

- [x] CHK-007 No remnants of Step 8: No references to the removed step's behavior (stage transition in fab-new) remain in the skill file
- [x] CHK-008 Step numbering consistent: No references to "Step 9" remain in `fab-new.md` (all renumbered to Step 8)

## Scenario Coverage

- [x] CHK-009 Normal brief generation scenario: Verified that `/fab-new` behavior ends with `brief: active`
- [x] CHK-010 Brief with --switch scenario: `/fab-switch` call in Step 8 (formerly Step 9) does not perform stage transitions
- [x] CHK-011 /fab-continue after /fab-new scenario: `/fab-continue` correctly detects `brief` as active and generates spec

## Documentation Accuracy

- [x] CHK-012 planning-skills.md consistency: `/fab-new` description matches actual skill behavior
- [x] CHK-013 _context.md Next Steps accuracy: Table entry matches actual `/fab-new` output

## Cross References

- [x] CHK-014 Output examples in fab-new.md: "Brief complete." text and Next lines are consistent with brief staying active
- [x] CHK-015 No stale "brief done" references: No other files in `fab/.kit/skills/` reference `/fab-new` marking brief as done

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
