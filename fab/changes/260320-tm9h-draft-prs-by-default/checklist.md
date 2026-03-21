# Quality Checklist: Draft PRs by Default

**Change**: 260320-tm9h-draft-prs-by-default
**Generated**: 2026-03-20
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Draft PR creation: `gh pr create` in Step 3c item 4 includes `--draft` flag
- [ ] CHK-002 Fallback draft: `gh pr create --fill` fallback also includes `--draft` flag
- [ ] CHK-003 SPEC-git-pr.md updated: Flow diagram Step 3c reflects `--draft` flag

## Behavioral Correctness
- [ ] CHK-004 Existing PR behavior unchanged: `/git-pr` does not modify draft/ready state of existing PRs
- [ ] CHK-005 Unconditional: No configuration toggle or conditional logic around `--draft`

## Scenario Coverage
- [ ] CHK-006 New PR scenario: Created PR is in draft state
- [ ] CHK-007 Existing PR scenario: Existing PRs are not modified
- [ ] CHK-008 Fallback scenario: Fallback path also creates draft PR

## Code Quality
- [ ] CHK-009 Pattern consistency: Change follows existing skill file formatting and structure
- [ ] CHK-010 No unnecessary duplication: Single `--draft` flag addition, not duplicated logic

## Documentation Accuracy
- [ ] CHK-011 Spec file accuracy: SPEC-git-pr.md flow diagram matches actual skill behavior

## Cross References
- [ ] CHK-012 Backlog cleanup: [m1ef] marked as done (duplicate item)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
