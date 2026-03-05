# Quality Checklist: Add fab-switch --blank to deactivate the current change

**Change**: 260212-egqa-switch-return-main
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 `--blank` flag: fab-switch.md Arguments section documents `--blank` with description and behavior
- [x] CHK-002 Deactivation flow: fab-switch.md Behavior section includes a Deactivation Flow subsection describing `fab/current` deletion
- [x] CHK-003 No git operations: `--blank` alone is documented as having no git side effects
- [x] CHK-004 Composable with `--branch`: Documented that `--blank --branch main` combines deactivation with branch checkout
- [x] CHK-005 Output format: Deactivation output examples present in Output section (blank-only, with branch, branch failed, already blank)
- [x] CHK-006 Change lifecycle updated: `change-lifecycle.md` lists `/fab-switch --blank` in fab/current lifecycle and /fab-switch section

## Behavioral Correctness
- [x] CHK-007 Existing flows unchanged: fab-switch.md Argument Flow and No Argument Flow sections are NOT modified by this change
- [x] CHK-008 Key Properties updated: Key Properties table reflects that `fab/current` may now be deleted (not just written)

## Scenario Coverage
- [x] CHK-009 Deactivate with `--blank`: Scenario documented — delete fab/current, no git ops
- [x] CHK-010 Already deactivated (idempotent): Scenario documented — "No active change (already blank)."
- [x] CHK-011 Combine `--blank` with `--branch`: Scenario documented — delete fab/current + checkout branch
- [x] CHK-012 Branch checkout fails: Scenario documented — deactivation succeeds, stays on current branch
- [x] CHK-013 Preflight after deactivation: Scenario documented — preflight exits non-zero

## Edge Cases & Error Handling
- [x] CHK-014 Error handling table: Deactivation-specific error cases added (fab/current missing, git checkout failure with --branch)
- [x] CHK-015 `--no-branch-change` with `--blank`: Documented as redundant but harmless

## Documentation Accuracy
- [x] CHK-016 No references to old keyword design: No mention of `main`/`master` as deactivation keywords in the skill file
- [x] CHK-017 Consistent terminology: All references use `--blank` flag (not "deactivate keyword" or similar)

## Cross References
- [x] CHK-018 change-lifecycle.md `/fab-switch` section matches fab-switch.md skill file
- [x] CHK-019 Affected docs correct: Only `change-lifecycle.md` is modified (not `planning-skills.md`)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
