# Quality Checklist: Add `fab pane-map` Subcommand

**Change**: 260306-bh45-pane-map-subcommand
**Generated**: 2026-03-06
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Subcommand Registration: `fab pane-map` appears in `fab --help` and executes without error
- [x] CHK-002 Tmux Pane Discovery: all tmux panes are enumerated via `tmux list-panes -a`
- [x] CHK-003 Worktree Resolution: pane CWDs correctly resolve to git worktree roots
- [x] CHK-004 Runtime State Correlation: `.fab-runtime.yaml` is read and agent state rendered per change
- [x] CHK-005 Output Format: aligned table with Pane, Worktree, Change, Stage, Agent columns
- [x] CHK-006 Main Worktree Inclusion: main repo worktree shown as `(main)` in Worktree column
- [x] CHK-007 Non-Fab Pane Exclusion: panes not in git repos or without `fab/` are excluded
- [x] CHK-008 Tmux Session Guard: error message and exit 1 when `$TMUX` is unset

## Behavioral Correctness

- [x] CHK-009 Idle Duration Format: `{N}s` / `{N}m` / `{N}h` with floor division at boundaries
- [x] CHK-010 Relative Paths: worktree paths are relative to main repo parent, not absolute
- [x] CHK-011 No-Change Panes: worktrees without `fab/current` show `(no change)` and `—` for Stage
- [x] CHK-012 Multiple Panes Same Worktree: each pane gets its own row (no dedup)
- [x] CHK-013 Runtime File Missing: Agent column shows `?` when `.fab-runtime.yaml` absent
- [x] CHK-014 Empty Result: prints `No fab worktrees found in tmux panes.` when no fab panes exist

## Scenario Coverage

- [x] CHK-015 Scenario: pane in worktree subdirectory resolves to correct worktree root
- [x] CHK-016 Scenario: agent idle for 300s shows `idle (5m)`
- [x] CHK-017 Scenario: agent active (no idle_since) shows `active`

## Edge Cases & Error Handling

- [x] CHK-018 Pane CWD in non-git directory: excluded silently
- [x] CHK-019 `fab/current` is empty file: treated as `(no change)`
- [x] CHK-020 `.status.yaml` missing for an active change: stage shows `—`

## Code Quality

- [x] CHK-021 Pattern consistency: follows existing command file patterns (runtime.go, status.go)
- [x] CHK-022 No unnecessary duplication: reuses `loadRuntimeFile`, `statusfile.Load`, `status.DisplayStage` where applicable <!-- note: loadPaneMapRuntimeFile is a justified variant of loadRuntimeFile (needs to distinguish missing vs empty); readFabCurrent duplicates worktree.resolveFabState but per spec Decision #3 this is intentional -->

## Documentation Accuracy

- [x] CHK-023 `_scripts.md` updated: `fab pane-map` documented in Command Reference table and has its own section

## Cross References

- [x] CHK-024 Constitution compliance: Go CLI change includes test updates and `_scripts.md` update

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
