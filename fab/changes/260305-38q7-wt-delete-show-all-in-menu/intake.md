# Intake: wt-delete — Show "All" in Selection Menu

**Change**: 260305-38q7-wt-delete-show-all-in-menu
**Created**: 2026-03-05
**Status**: Draft

## Origin

> User request: "wt-delete: the option 'wt-delete --all' should be absorbed into the default invocation of wt-delete (without arguments). How? Show All as the first option"

One-shot description. The user wants the most common `--delete-all` workflow to be reachable from the bare `wt-delete` invocation without needing to remember the flag.

## Why

1. **Problem**: Deleting all worktrees is the most frequent use of `wt-delete` when called from the main repo (not inside a worktree). Currently this requires the `--delete-all` flag, which is easy to forget. The interactive selection menu only shows individual worktrees, forcing users to either remember the flag or delete worktrees one by one.

2. **Consequence**: Users who forget `--delete-all` must either re-invoke with the flag or tediously pick worktrees one at a time from the menu.

3. **Approach**: Add an "All worktrees (N)" option as the **first item** in the `wt_select_worktree_menu` selection menu. When selected, it delegates to the existing `wt_delete_all_worktrees` function. The `--delete-all` flag remains available for non-interactive/scripted usage.

## What Changes

### `wt_select_worktree_menu` in `fab/.kit/packages/wt/bin/wt-delete`

The selection menu currently shows only individual worktrees:

```
Select worktree to delete:
  1) feature-a (wt/feature-a)
  2) feature-b (260305-38q7-some-change)
  0) Cancel
```

After this change, an "All" option appears as the first menu item:

```
Select worktree to delete:
  1) All (2 worktrees)
  2) feature-a (wt/feature-a)
  3) feature-b (260305-38q7-some-change)
  0) Cancel
```

When the user selects "All", the function delegates to `wt_delete_all_worktrees` with the same parameters (`non_interactive`, `delete_branch`, `delete_remote`, `stash_flag`). Individual worktree selections shift by one index.

The default selection (highlighted/pre-selected) remains on the most recently modified individual worktree (shifted by +1 to account for the new "All" entry at position 1).

### No changes to `--delete-all` flag

The `--delete-all` flag continues to work as before for non-interactive and scripted usage. This change only affects the interactive menu path.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update wt-delete menu behavior description if it documents the selection menu

## Impact

- **Code**: Single file change — `fab/.kit/packages/wt/bin/wt-delete`, specifically the `wt_select_worktree_menu` function
- **Tests**: `src/packages/wt/tests/wt-delete.bats` may need a new test case for the "All" menu option
- **UX**: Interactive users get a faster path to "delete all" without flags
- **Backward compatibility**: No breaking changes — `--delete-all` flag is preserved, non-interactive mode is unchanged

## Open Questions

- None

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | "All" appears as menu item 1 (first position) | User explicitly stated "Show All as the first option" | S:95 R:90 A:95 D:95 |
| 2 | Certain | `--delete-all` flag is preserved | Non-interactive/scripted usage needs a direct flag; removing it would be a breaking change | S:80 R:60 A:95 D:95 |
| 3 | Confident | "All" delegates to existing `wt_delete_all_worktrees` | Reusing existing logic is the obvious approach — no reason to duplicate | S:70 R:90 A:90 D:90 |
| 4 | Confident | Default selection shifts to account for the new item | The default (most recent worktree) should still be pre-selected, just at index+1 | S:60 R:90 A:85 D:80 |
| 5 | Certain | Menu label format is "All (N worktrees)" | Shows the count for clarity, consistent with the `wt_delete_all_worktrees` confirmation message pattern | S:70 R:95 A:85 D:85 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
