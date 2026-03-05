# Spec: wt-delete — Show "All" in Selection Menu

**Change**: 260305-38q7-wt-delete-show-all-in-menu
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## wt-delete: Interactive Selection Menu

### Requirement: "All" option in worktree selection menu

The `wt_select_worktree_menu` function SHALL prepend an "All ({N} worktrees)" option as menu item 1 before the individual worktree entries. When selected, it SHALL delegate to `wt_delete_all_worktrees` with the same parameters (`non_interactive`, `delete_branch`, `delete_remote`, `stash_flag`).

Individual worktree entries SHALL be shifted by +1 in the menu numbering. The default selection (most recently modified worktree) SHALL be shifted by +1 to account for the new entry.

#### Scenario: Menu displays "All" as first option

- **GIVEN** the user is in the main repo (not inside a worktree) and 2 worktrees exist (`alpha`, `beta`)
- **WHEN** the user runs `wt-delete` with no arguments
- **THEN** the menu displays:
  ```
  Select worktree to delete:
    1) All (2 worktrees)
    2) alpha (wt/alpha)
    3) beta (some-branch)
    0) Cancel
  ```

#### Scenario: Selecting "All" deletes all worktrees

- **GIVEN** the selection menu is displayed with 3 worktrees
- **WHEN** the user selects option 1 ("All")
- **THEN** `wt_delete_all_worktrees` is called with the current `non_interactive`, `delete_branch`, `delete_remote`, and `stash_flag` values
- **AND** all 3 worktrees are deleted

#### Scenario: Selecting an individual worktree still works

- **GIVEN** the selection menu is displayed with 2 worktrees
- **WHEN** the user selects option 2 (the first individual worktree)
- **THEN** `wt_delete_worktree_by_name` is called for that worktree
- **AND** only that worktree is deleted

#### Scenario: Default selection accounts for the new "All" entry

- **GIVEN** 3 worktrees exist and the most recently modified is `beta` (originally index 2)
- **WHEN** the menu is displayed
- **THEN** the default selection is 3 (index 2 + 1 for the "All" entry)

### Requirement: `--delete-all` flag preserved

The `--delete-all` CLI flag SHALL continue to work as before, directly invoking `wt_delete_all_worktrees` without the selection menu. This change SHALL NOT affect non-interactive mode or the `--delete-all` code path.

#### Scenario: --delete-all bypasses menu

- **GIVEN** 2 worktrees exist
- **WHEN** the user runs `wt-delete --delete-all --non-interactive`
- **THEN** all worktrees are deleted without displaying the selection menu

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | "All" appears as menu item 1 (first position) | Confirmed from intake #1 — user explicitly stated "Show All as the first option" | S:95 R:90 A:95 D:95 |
| 2 | Certain | `--delete-all` flag is preserved | Confirmed from intake #2 — non-interactive/scripted usage needs a direct flag | S:80 R:60 A:95 D:95 |
| 3 | Confident | "All" delegates to existing `wt_delete_all_worktrees` | Confirmed from intake #3 — reusing existing logic, no duplication | S:70 R:90 A:90 D:90 |
| 4 | Confident | Default selection shifts +1 for the "All" entry | Confirmed from intake #4 — MRU worktree stays pre-selected | S:60 R:90 A:85 D:80 |
| 5 | Certain | Menu label format is "All (N worktrees)" | Confirmed from intake #5 — consistent with existing patterns | S:70 R:95 A:85 D:85 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
