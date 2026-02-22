# Quality Checklist: git-pr Shipped Sentinel and Status Commit

**Change**: 260222-trdc-git-pr-shipped-sentinel-and-status-commit
**Generated**: 2026-02-22
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Second commit+push after recording PR URL: `.status.yaml` changes are staged, committed, and pushed after `stageman ship` writes the URL
- [x] CHK-002 Sentinel file creation: `.shipped` file is created with PR URL content after all git ops complete
- [x] CHK-003 Gitignore patterns: `.shipped` pattern added and prevents tracking

## Behavioral Correctness

- [x] CHK-004 git-pr skill documentation: Steps 4b and 4c are documented in `fab/.kit/skills/git-pr.md`
- [x] CHK-005 Sentinel polling: `run.sh` checks for `.shipped` file existence instead of `stageman is-shipped`

## Scenario Coverage

- [x] CHK-006 Manual git-pr flow: Running `/git-pr` creates sentinel file and branch is clean
- [x] CHK-007 Orchestrator flow: `run.sh` detects sentinel and marks change `done`
- [x] CHK-008 Dependent change branching: Next change branches from parent's clean tip (no dirty state leaks)

## Edge Cases & Error Handling

- [x] CHK-009 Sentinel persistence: File remains after worktree operations (used by orchestrator polling)
- [x] CHK-010 Cleanup: Sentinel file is removed when worktree/change folder is deleted
- [x] CHK-011 Git operation failure: If second commit+push fails, error is reported and git-pr stops (no sentinel written). Step 4b uses `git diff --cached --quiet` guard for idempotent case.

## Code Quality

- [x] CHK-012 Pattern consistency: git-pr skill additions follow existing step numbering and conditional structure
- [x] CHK-013 No unnecessary duplication: Simplified to plain `echo` write (dropped unnecessary atomic mv pattern)

## Documentation

- [x] CHK-014 Memory hydration: `execution-skills.md` documents git-pr's new sentinel write
- [x] CHK-015 Memory hydration: `pipeline-orchestrator.md` documents sentinel-based ship detection in `run.sh`
- [x] CHK-016 Cross-references: Memory files reference each other and the change name for traceability

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-001 **N/A**: {reason}`
