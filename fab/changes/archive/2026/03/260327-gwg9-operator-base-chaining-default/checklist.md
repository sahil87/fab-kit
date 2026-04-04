# Quality Checklist: Operator Base-Chaining Default

**Change**: 260327-gwg9-operator-base-chaining-default
**Generated**: 2026-03-27
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Stack-then-review default: Autopilot steps 5–9 describe stack-then-review behavior (record, dispatch next, report, summary)
- [ ] CHK-002 Implicit --base chaining: Queue ordering table states that user-provided ordering implies implicit depends_on for consecutive entries
- [ ] CHK-003 --merge-on-complete flag: Flag documented with description, natural language equivalents, and revert behavior
- [ ] CHK-004 Confirmation prompt: Default prompt says "creates PRs — merge after review"; merge-on-complete says "merges PRs on completion"
- [ ] CHK-005 Queue completion summary: After all changes complete, operator lists PR links with dependency annotations and merge order
- [ ] CHK-006 Ordered merge: "merge all" command documented with CI wait and failure halt behavior

## Behavioral Correctness
- [ ] CHK-007 Previous merge-as-you-go behavior is preserved under --merge-on-complete, not deleted
- [ ] CHK-008 Existing dependency resolution (cherry-pick mechanism) is unchanged — only the default chaining is new
- [ ] CHK-009 Confidence-based and hybrid strategies remain unchanged (stacking only applies to user-provided sequential ordering)

## Removal Verification
- [ ] CHK-010 Old steps 6–9 (merge/rebase/cleanup/report) are replaced, not left as dead text alongside new steps

## Scenario Coverage
- [ ] CHK-011 Default autopilot with three changes: implicit depends_on, PR-ready reports, final summary
- [ ] CHK-012 Single-item queue: no depends_on added, simple completion
- [ ] CHK-013 Explicit --merge-on-complete: merge/rebase behavior restored
- [ ] CHK-014 CI failure during ordered merge: halt and report

## Edge Cases & Error Handling
- [ ] CHK-015 Cherry-pick conflict in stacked queue escalates (not skips)
- [ ] CHK-016 Failure handling note updated: rebase conflict line qualified for merge-on-complete only

## Code Quality
- [ ] CHK-017 Pattern consistency: New text follows existing operator7 section style (numbered steps, table formats, scenario examples)
- [ ] CHK-018 No unnecessary duplication: New steps don't repeat dependency resolution details already documented earlier in §6

## Documentation Accuracy
- [ ] CHK-019 "Working a Change" subsections updated: "On completion" lines consistent with stack-then-review default
- [ ] CHK-020 No references to old default merge behavior remain outside of --merge-on-complete context

## Cross References
- [ ] CHK-021 Dependency Declaration section (§6) still accurately describes all three declaration paths including implicit --base default
- [ ] CHK-022 Failure handling section consistent with new default (no rebase conflicts in stack-then-review)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
