# Quality Checklist: Standardize Tmux Tab Naming for Spawned Agents

**Change**: 260328-iqt8-standardize-tmux-tab-naming
**Generated**: 2026-03-28
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Consistent tab name format: All `tmux new-window -n` invocations use `⚡<wt>` format
- [x] CHK-002 All spawn paths updated: All four spawn paths (generic, existing change, raw text, backlog) use the same naming pattern

## Behavioral Correctness
- [x] CHK-003 No `fab-<id>` remnants: No `tmux new-window -n "fab-<id>"` patterns remain in the skill file
- [x] CHK-004 No `fab-<wt>` remnants: No `tmux new-window -n "fab-<wt>"` patterns remain in the skill file

## Removal Verification
- [x] CHK-005 Deprecated `fab-<id>` format: Confirm all old tab name patterns are replaced, no dead references

## Scenario Coverage
- [x] CHK-006 Existing change spawn: Tab name uses worktree name with ⚡ prefix
- [x] CHK-007 Raw text spawn: Tab name uses worktree name with ⚡ prefix
- [x] CHK-008 Backlog spawn: Tab name uses worktree name with ⚡ prefix

## Code Quality
- [x] CHK-009 Pattern consistency: Tab name format is identical across all spawn paths
- [x] CHK-010 No unnecessary duplication: Single naming convention used throughout

## Documentation Accuracy
- [x] CHK-011 Memory file updated: `docs/memory/fab-workflow/execution-skills.md` reflects the new tab naming convention

## Cross References
- [x] CHK-012 No spec file for operator7 exists under `docs/specs/skills/` — no spec update needed

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
