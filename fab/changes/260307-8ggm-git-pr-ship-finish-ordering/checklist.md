# Quality Checklist: Fix git-pr Ship Finish Ordering

**Change**: 260307-8ggm-git-pr-ship-finish-ordering
**Generated**: 2026-03-07
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Step ordering: Status mutations (add-pr, finish ship) occur before commit+push in `git-pr.md`
- [x] CHK-002 Staging scope: Step 4c stages both `.status.yaml` and `.history.jsonl`
- [x] CHK-003 Step renumbering: Post-PR steps are numbered 4a/4b/4c/4d

## Behavioral Correctness
- [x] CHK-004 Best-effort finish: Step 4b preserves `2>/dev/null || true` error suppression
- [x] CHK-005 No-op guard: Step 4c checks `git diff --cached --quiet` before committing
- [x] CHK-006 Sentinel position: `.pr-done` write remains the final step (4d)

## Scenario Coverage
- [x] CHK-007 Normal flow: All mutations committed atomically before sentinel write
- [x] CHK-008 Already-shipped path: Early-exit references updated to use new step numbering (4a–4d)

## Edge Cases & Error Handling
- [x] CHK-009 History missing: `git add .history.jsonl` is a no-op when file doesn't exist
- [x] CHK-010 Finish fails: Pipeline continues when `fab status finish` errors (best-effort)

## Code Quality
- [x] CHK-011 Pattern consistency: New step ordering follows existing git-pr.md patterns and markdown style
- [x] CHK-012 No unnecessary duplication: No duplicate commit+push cycles introduced

## Documentation Accuracy
- [x] CHK-013 Spec file: `SPEC-git-pr.md` flow diagram reflects new 4a/4b/4c/4d ordering
- [x] CHK-014 Cross-reference: No stale step number references remain in `git-pr.md`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
