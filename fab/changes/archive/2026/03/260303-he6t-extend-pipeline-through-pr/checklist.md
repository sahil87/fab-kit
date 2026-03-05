# Quality Checklist: Extend Pipeline Through PR

**Change**: 260303-he6t-extend-pipeline-through-pr
**Generated**: 2026-03-04
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Ship stage definition: workflow.yaml contains `ship` stage with correct properties (id, name, allowed_states without failed)
- [ ] CHK-002 Review-PR stage definition: workflow.yaml contains `review-pr` stage with correct properties (id, name, allowed_states including failed)
- [ ] CHK-003 Review-PR transitions: workflow.yaml includes `review-pr` transition overrides matching review pattern (start from failed, fail from active)
- [ ] CHK-004 Stage numbering: stage_numbers maps ship=7, review-pr=8
- [ ] CHK-005 Completion rule: progression.completion references review-pr, not hydrate
- [ ] CHK-006 Template progress map: status.yaml template includes ship and review-pr as pending
- [ ] CHK-007 Statusman current-stage fallback: returns review-pr instead of hydrate
- [ ] CHK-008 Statusman auto-log for review-pr: finish and fail both trigger logman
- [ ] CHK-009 Finish hydrate auto-activates ship: event_finish chain continues past hydrate
- [ ] CHK-010 Changeman stage functions: stage_number, next_stage, default_command updated for all 8 stages
- [ ] CHK-011 Changeman display format: shows N/8 instead of N/6
- [ ] CHK-012 Skill rename: git-review.md renamed to git-pr-review.md with updated frontmatter
- [ ] CHK-013 Git-PR statusman integration: start/finish ship stage calls present, best-effort
- [ ] CHK-014 Git-PR-Review statusman integration: start/finish/fail review-pr calls present
- [ ] CHK-015 Git-PR-Review phase tracking: stage_metrics.review-pr.phase updates present
- [ ] CHK-016 Preamble state table: new rows for ship, review-pr (pass), review-pr (fail)
- [ ] CHK-017 fab-ff pipeline extension: Step 8 (Ship) and Step 9 (Review-PR) added
- [ ] CHK-018 fab-fff pipeline extension: Step 9 (Ship) and Step 10 (Review-PR) added
- [ ] CHK-019 fab-continue dispatch: ship and review-pr entries added to dispatch table
- [ ] CHK-020 All git-review references updated to git-pr-review

## Behavioral Correctness

- [ ] CHK-021 Ship rejects failed: `statusman.sh fail <change> ship` returns error
- [ ] CHK-022 Review-PR accepts failed: `statusman.sh fail <change> review-pr` succeeds
- [ ] CHK-023 Review-PR restarts from failed: `statusman.sh start <change> review-pr` succeeds from failed state
- [ ] CHK-024 Existing 6-stage changes still work: progress-map defaults missing stages to pending

## Scenario Coverage

- [ ] CHK-025 Full 8-stage pipeline completes: all stages reach done via statusman transitions
- [ ] CHK-026 Resumability: fab-ff skips completed ship/review-pr stages
- [ ] CHK-027 Git-PR without active change: statusman calls skipped silently, PR creation succeeds

## Edge Cases & Error Handling

- [ ] CHK-028 Old status.yaml without ship/review-pr: statusman operations work via yq defaults
- [ ] CHK-029 Git-PR statusman failure: silently ignored, PR creation not blocked
- [ ] CHK-030 Review-PR no reviews: treated as successful done (not skipped)

## Code Quality

- [ ] CHK-031 Pattern consistency: new stage entries follow existing patterns in workflow.yaml, statusman.sh, and changeman.sh
- [ ] CHK-032 No unnecessary duplication: review-pr transition overrides reuse the review pattern, not a third copy

## Documentation Accuracy

- [ ] CHK-033 Preamble state table matches implementation: state table entries produce correct routing
- [ ] CHK-034 fab-continue stage list matches implementation: all 8 stages accepted as valid targets

## Cross References

- [ ] CHK-035 No remaining references to `/git-review` (excluding git-pr-review.md deprecation section)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
