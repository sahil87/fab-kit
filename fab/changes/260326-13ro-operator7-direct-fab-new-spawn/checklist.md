# Quality Checklist: Operator7 Direct fab-new Spawn

**Change**: 260326-13ro-operator7-direct-fab-new-spawn
**Generated**: 2026-03-26
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Direct fab-new invocation: "From raw text" section uses `/fab-new <description>` directly without `idea add`
- [ ] CHK-002 Updated explanatory paragraph: No `idea add` reference, explains `/fab-new` handles traceability
- [ ] CHK-003 Backlog/Linear path unchanged: "From backlog ID or Linear issue" section is unmodified

## Behavioral Correctness
- [ ] CHK-004 Spawn sequence includes dependency resolution: `depends_on` cherry-pick step is present in the raw text flow
- [ ] CHK-005 Spawn sequence includes worktree creation, agent tab, and enrollment steps

## Scenario Coverage
- [ ] CHK-006 Raw text spawn: The updated flow matches the spec's spawn sequence (worktree → deps → spawn → enroll → completion)
- [ ] CHK-007 No orphaned references: No remaining `idea add` references in the file

## Code Quality
- [ ] CHK-008 Pattern consistency: The raw text flow follows the same structure as the backlog/Linear flow
- [ ] CHK-009 No unnecessary duplication: Shared steps between flows are not duplicated unnecessarily

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
