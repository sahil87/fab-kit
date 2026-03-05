# Quality Checklist: Add --reuse flag to wt-create

**Change**: 260222-6ldg-wt-create-reuse-flag
**Generated**: 2026-02-22
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 --reuse returns existing path on collision: `wt-create --reuse --worktree-name X` exits 0 and prints existing path when worktree X exists
- [ ] CHK-002 --reuse creates normally when no collision: `wt-create --reuse --worktree-name X` creates worktree when X does not exist
- [ ] CHK-003 --reuse requires --worktree-name: `wt-create --reuse` without `--worktree-name` exits with code 2 and error message
- [ ] CHK-004 --reuse is blind to worktree state: orphaned directory (not git-registered) is returned without error
- [ ] CHK-005 --reuse in help output: `wt-create help` includes `--reuse` description
- [ ] CHK-006 batch-fab-switch-change.sh passes --reuse: `wt-create` invocation includes `--reuse` flag
- [ ] CHK-007 dispatch.sh passes --reuse and removes bespoke check: `wt-create` invocation includes `--reuse`, no `wt_get_worktree_path_by_name` call, no `wt-common.sh` source import

## Behavioral Correctness

- [ ] CHK-008 --reuse skips init script on collision: init marker file not created when reusing existing worktree
- [ ] CHK-009 --reuse skips app opening on collision: no app launch when reusing existing worktree
- [ ] CHK-010 Default behavior unchanged: `wt-create` without `--reuse` still errors on name collision
- [ ] CHK-011 batch-fab-new-backlog.sh unchanged: no `--reuse` flag in its `wt-create` invocation

## Scenario Coverage

- [ ] CHK-012 Test: reuse on collision returns success and same path
- [ ] CHK-013 Test: normal creation with --reuse when no collision
- [ ] CHK-014 Test: init script skipping on reuse
- [ ] CHK-015 Test: --reuse without --worktree-name validation
- [ ] CHK-016 Test: path as last line on reuse
- [ ] CHK-017 Test: orphaned directory edge case

## Edge Cases & Error Handling

- [ ] CHK-018 --reuse with other flags: `--reuse` combined with `--non-interactive`, `--worktree-open`, `--worktree-init` does not cause parse errors
- [ ] CHK-019 Error message format: `--reuse` without `--worktree-name` uses what/why/fix format

## Code Quality

- [ ] CHK-020 Pattern consistency: `--reuse` flag parsing follows existing flag patterns in wt-create
- [ ] CHK-021 No unnecessary duplication: reuse logic uses existing `wt_check_name_collision` helper

## Documentation Accuracy

- [ ] CHK-022 Help text: `--reuse` description is clear and accurate
- [ ] CHK-023 Memory update: `docs/memory/fab-workflow/execution-skills.md` documents `--reuse` flag

## Cross References

- [ ] CHK-024 Callers consistent: all resume-semantics callers pass `--reuse`, fresh-creation callers do not

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
