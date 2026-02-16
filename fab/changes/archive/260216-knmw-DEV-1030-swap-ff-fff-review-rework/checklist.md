# Quality Checklist: Swap fab-ff and fab-fff Review Failure Behavior

**Change**: 260216-knmw-DEV-1030-swap-ff-fff-review-rework
**Generated**: 2026-02-16
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 fab-ff interactive rework: `fab-ff.md` Step 5 Fail block describes 3 rework options (fix code, revise tasks, revise spec) presented to user
- [x] CHK-002 fab-ff no bail: `fab-ff.md` contains no reference to bailing on review failure
- [x] CHK-003 fab-fff autonomous rework: `fab-fff.md` Step 7 Fail block describes agent autonomously choosing rework path
- [x] CHK-004 fab-fff retry cap: `fab-fff.md` specifies 3-cycle maximum with bail message format
- [x] CHK-005 fab-fff escalation: `fab-fff.md` specifies forced escalation after 2 consecutive fix-code failures
- [x] CHK-006 fab-fff no interactive menu: `fab-fff.md` contains no reference to presenting interactive rework menu to user
- [x] CHK-007 Context autonomy table: `_context.md` escape valve entries swapped for fab-ff and fab-fff
- [x] CHK-008 Planning skills memory: `planning-skills.md` reflects swapped behavior in requirement sections and design decisions
- [x] CHK-009 Execution skills memory: `execution-skills.md` pipeline invocation note reflects swapped behavior

## Behavioral Correctness

- [x] CHK-010 fab-ff rework logging: `fab-ff.md` calls `log-review` with rework option after user selection
- [x] CHK-011 fab-fff rework logging: `fab-fff.md` calls `log-review` for each autonomous rework cycle
- [x] CHK-012 fab-ff no retry cap: `fab-ff.md` rework loop has no artificial bound (user controls iteration)

## Scenario Coverage

- [x] CHK-013 fab-ff fix code scenario: Step 5 describes unchecking affected tasks with `<!-- rework: reason -->` and re-running apply+review
- [x] CHK-014 fab-ff revise spec scenario: Step 5 describes resetting to spec stage and invalidating downstream
- [x] CHK-015 fab-fff escalation scenario: Escalation logic explicitly states agent MUST NOT choose fix-code after 2 consecutive attempts
- [x] CHK-016 fab-fff bail scenario: Bail message format includes per-cycle summary and suggests `/fab-continue`

## Edge Cases & Error Handling

- [x] CHK-017 fab-ff error table: Error Handling table row for "Review fails" updated to interactive rework
- [x] CHK-018 fab-fff error table: Error Handling table row for "Review fails" updated to autonomous rework with retry cap
- [x] CHK-019 Consecutive counter reset: `fab-fff.md` specifies that non-fix-code actions reset the consecutive counter

## Code Quality

- [x] CHK-020 Pattern consistency: Updated files follow existing formatting and structural patterns
- [x] CHK-021 No unnecessary duplication: Rework descriptions reference `/fab-continue` behavior where appropriate rather than duplicating

## Documentation Accuracy

- [x] CHK-022 Frontmatter descriptions: Both skill files have updated description frontmatter matching new behavior
- [x] CHK-023 Purpose sections: Both skill files have updated Purpose sections matching new behavior

## Cross References

- [x] CHK-024 Changelog entries: Memory files include changelog entries for this change
- [x] CHK-025 Specs consistency: `docs/specs/skills.md` and `docs/specs/user-flow.md` checked for stale references

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
