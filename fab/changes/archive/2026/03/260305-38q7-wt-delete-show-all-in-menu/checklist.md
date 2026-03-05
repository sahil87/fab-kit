# Quality Checklist: wt-delete — Show "All" in Selection Menu

**Change**: 260305-38q7-wt-delete-show-all-in-menu
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 "All" option appears as menu item 1 in `wt_select_worktree_menu`
- [ ] CHK-002 Selecting "All" delegates to `wt_delete_all_worktrees` with correct parameters
- [ ] CHK-003 `--delete-all` flag continues to work unchanged

## Behavioral Correctness
- [ ] CHK-004 Individual worktree indices shift by +1 (item that was 1 is now 2)
- [ ] CHK-005 Default selection (MRU worktree) shifts by +1 to account for "All" entry
- [ ] CHK-006 Selecting an individual worktree (index >= 2) correctly maps to the right worktree name

## Scenario Coverage
- [ ] CHK-007 Menu displays "All (N worktrees)" with correct count
- [ ] CHK-008 Selecting "All" deletes all worktrees
- [ ] CHK-009 Cancel (0) still cancels

## Edge Cases & Error Handling
- [ ] CHK-010 No worktrees: still shows "No worktrees found." (no "All" option with 0)

## Code Quality
- [ ] CHK-011 Pattern consistency: follows existing `wt_show_menu` calling conventions
- [ ] CHK-012 No unnecessary duplication: reuses `wt_delete_all_worktrees` instead of duplicating

## Documentation Accuracy
- [ ] CHK-013 Help text mentions the interactive "All" option

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
