# Quality Checklist: Worktree Status Command

**Change**: 260305-7zq4-worktree-status-command
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Single worktree status: `wt_get_fab_status` resolves change name, stage, and state from a worktree path
- [ ] CHK-002 Default mode: no args shows current worktree's fab status
- [ ] CHK-003 Named mode: `wt-status <name>` shows the named worktree's fab status
- [ ] CHK-004 All mode: `--all` iterates all worktrees and displays formatted table
- [ ] CHK-005 Help: `wt-status help` displays usage information
- [ ] CHK-006 Script infrastructure: sources `wt-common.sh`, uses `set -euo pipefail`, validates git repo

## Behavioral Correctness
- [ ] CHK-007 statusman invocation: passes absolute .status.yaml path to statusman.sh display-stage
- [ ] CHK-008 fab/current parsing: reads two-line format correctly (line 1 = ID, line 2 = folder name)
- [ ] CHK-009 Current worktree marker: `*` prefix on current worktree in `--all` mode
- [ ] CHK-010 Main repo label: displays `(main)` for the main repo entry

## Scenario Coverage
- [ ] CHK-011 Worktree with active fab change shows change name, stage, and state
- [ ] CHK-012 Worktree with no fab directory shows `(no fab)` fallback
- [ ] CHK-013 Worktree with fab but no active change shows `(no change)` fallback
- [ ] CHK-014 Worktree with stale fab/current pointer shows `(stale)` fallback
- [ ] CHK-015 Invalid worktree name shows error message

## Edge Cases & Error Handling
- [ ] CHK-016 Missing fab/current file handled gracefully (no crash)
- [ ] CHK-017 Empty fab/current file handled gracefully
- [ ] CHK-018 Missing .status.yaml for referenced change handled gracefully
- [ ] CHK-019 Non-git-repo invocation shows error and exits

## Code Quality
- [ ] CHK-020 Pattern consistency: follows naming and structural patterns of existing wt-* commands (wt-list, wt-delete)
- [ ] CHK-021 No unnecessary duplication: reuses wt-common.sh helpers
- [ ] CHK-022 No god functions: each function has a focused responsibility

## Documentation Accuracy
- [ ] CHK-023 Help text covers all three invocation modes with examples

## Cross References
- [ ] CHK-024 statusman.sh display-stage output format matches what wt-status parses

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
