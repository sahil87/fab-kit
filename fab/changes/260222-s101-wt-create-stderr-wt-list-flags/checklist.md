# Quality Checklist: wt-create stderr & wt-list flags

**Change**: 260222-s101-wt-create-stderr-wt-list-flags
**Generated**: 2026-02-22
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Stderr redirect: wt-create `--non-interactive` writes only path to stdout
- [ ] CHK-002 Stderr redirect: wt-pr `--non-interactive` writes only path to stdout
- [ ] CHK-003 Path flag: `wt-list --path <name>` outputs absolute path for existing worktree
- [ ] CHK-004 Path flag: `wt-list --path <name>` exits 1 for nonexistent worktree
- [ ] CHK-005 JSON flag: `wt-list --json` outputs valid JSON array with all required fields
- [ ] CHK-006 Status column: default wt-list shows `*` for dirty and `↑N` for unpushed

## Behavioral Correctness

- [ ] CHK-007 Interactive mode: wt-create without `--non-interactive` still writes all output to stdout
- [ ] CHK-008 Reuse path: `--reuse` codepath still works (already uses stderr)
- [ ] CHK-009 Batch callers: all 3 scripts capture path correctly without `| tail -1`

## Scenario Coverage

- [ ] CHK-010 Non-interactive creation captures path cleanly (spec scenario 1)
- [ ] CHK-011 Non-interactive with init script (spec scenario 2)
- [ ] CHK-012 JSON with dirty/unpushed detection (spec scenario)
- [ ] CHK-013 Flag mutual exclusivity: `--path` + `--json` errors (spec scenario)

## Edge Cases & Error Handling

- [ ] CHK-014 `--path` for main repo label "(main)" does not match
- [ ] CHK-015 `--json` with no worktrees returns array with 1 element (main)
- [ ] CHK-016 Status column handles worktree with no upstream (unpushed = 0, not error)

## Code Quality

- [ ] CHK-017 Pattern consistency: stderr redirects use `>&2` at call sites, not library function changes
- [ ] CHK-018 No unnecessary duplication: reuses existing `wt_has_uncommitted_changes`, `wt_get_unpushed_count` helpers

## Documentation Accuracy

- [ ] CHK-019 Help text for wt-create documents non-interactive output behavior
- [ ] CHK-020 Help text for wt-list documents `--path`, `--json`, and status indicators

## Cross References

- [ ] CHK-021 specs/packages.md: no update needed (describes behavior at concept level)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
