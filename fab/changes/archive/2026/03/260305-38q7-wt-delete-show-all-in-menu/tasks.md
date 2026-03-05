# Tasks: wt-delete — Show "All" in Selection Menu

**Change**: 260305-38q7-wt-delete-show-all-in-menu
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Modify `wt_select_worktree_menu` in `fab/.kit/packages/wt/bin/wt-delete` to prepend "All ({N} worktrees)" as menu item 1, shift individual worktree indices by +1, shift default selection by +1, and delegate to `wt_delete_all_worktrees` when "All" is selected

## Phase 2: Tests

- [x] T002 Add test to `src/packages/wt/tests/wt-delete.bats` verifying the menu includes "All" as the first option when worktrees exist

## Phase 3: Documentation

- [x] T003 Update help text in `wt_show_help` in `fab/.kit/packages/wt/bin/wt-delete` — add a note that the interactive menu includes an "All" option

---

## Execution Order

- T001 is the sole prerequisite — T002 and T003 depend on it
