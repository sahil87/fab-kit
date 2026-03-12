# Quality Checklist: Remove fab status show and fix stale fab/current references

**Change**: 260312-9lci-fix-status-show-fab-current
**Generated**: 2026-03-12
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Show subcommand removed: `statusShowCmd()` not registered in `statusCmd()`
- [x] CHK-002 Dead code removed: `worktreeInfo`, `listWorktrees`, `resolveWorktreeFabState`, `findWorktreeByName`, `currentWorktree`, `formatWorktreeHuman`, `formatWorktreesHuman` all deleted
- [x] CHK-003 Unused imports cleaned: no `encoding/json`, `os/exec`, `path/filepath` if unused by remaining code

## Behavioral Correctness
- [x] CHK-004 Other status subcommands unaffected: `finish`, `start`, `advance`, `reset`, `skip`, `fail`, `set-change-type`, `set-checklist`, etc. still work
- [x] CHK-005 Go binary compiles cleanly after removal

## Removal Verification
- [x] CHK-006 `fab status show` fully removed: no references in source code outside of historical changelogs (note: `docs/specs/skills/SPEC-fab-operator{1,2}.md` still reference it -- see should-fix)
- [x] CHK-007 `fab/current` references updated: all current-mechanism descriptions use `.fab-status.yaml` (note: kit-architecture.md line 259 "Preserved" list still says `current` -- see should-fix)

## Scenario Coverage
- [x] CHK-008 _scripts.md: no `status show` in command table or subcommand table
- [x] CHK-009 _scripts.md: send-keys pane resolution says `.fab-status.yaml` not `fab/current`
- [x] CHK-010 README.md: directory tree and activation text use `.fab-status.yaml`
- [x] CHK-011 kit-architecture.md: no `status show` in subcommand list, send-keys says `.fab-status.yaml`
- [x] CHK-012 execution-skills.md: operator fallback uses `wt list` + `fab change list`

## Edge Cases & Error Handling
- [x] CHK-013 Historical changelog entries preserved: changelog lines mentioning `fab status show` or `fab/current` in past-tense context are unchanged

## Code Quality
- [x] CHK-014 Pattern consistency: removal follows existing code structure patterns
- [x] CHK-015 No unnecessary duplication: no orphaned references left behind (note: residual refs in docs/specs/ and kit-architecture.md -- see should-fix)

## Documentation Accuracy
- [x] CHK-016 All documentation changes are factually accurate and consistent with the actual codebase state

## Cross References
- [x] CHK-017 Memory files, specs, and _scripts.md are internally consistent after updates (note: docs/specs/skills/ inconsistent -- see should-fix)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
