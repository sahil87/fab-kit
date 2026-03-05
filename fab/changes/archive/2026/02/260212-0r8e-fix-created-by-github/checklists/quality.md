# Quality Checklist: Fix created_by Format to Use GitHub ID

**Change**: 260212-0r8e-fix-created-by-github
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 GitHub ID as Primary: `fab-new.md` instructs agent to use `gh api user --jq .login` as the primary source for `created_by`
- [x] CHK-002 Graceful Fallback: `fab-new.md` instructs fallback to `git config user.name` when `gh` fails
- [x] CHK-003 Ultimate Fallback: `fab-new.md` instructs `"unknown"` when both `gh` and `git config` fail
- [x] CHK-004 Backward Compatibility: No existing `.status.yaml` files are modified by this change

## Behavioral Correctness

- [x] CHK-005 Fallback is Silent: Instructions specify no error or warning to user when `gh` fails
- [x] CHK-006 Raw Login String: Instructions specify the GitHub username is written as-is with no formatting or prefix

## Removal Verification

- [x] CHK-007 Deprecated Primary: `git config user.name` is no longer the primary source — only used as fallback

## Scenario Coverage

- [x] CHK-008 Scenario: gh available and authenticated — instructions cover this path
- [x] CHK-009 Scenario: gh not installed — instructions cover fallback
- [x] CHK-010 Scenario: gh not authenticated — instructions cover fallback
- [x] CHK-011 Scenario: gh API error — instructions cover fallback
- [x] CHK-012 Scenario: both gh and git config fail — instructions cover "unknown" fallback

## Documentation Accuracy

- [x] CHK-013 planning-skills.md reflects the new `created_by` behavior (primary: gh, fallback: git config, then "unknown")
- [x] CHK-014 planning-skills.md changelog includes entry for this change

## Cross References

- [x] CHK-015 fab-new.md Step 4 YAML template block matches the updated instructions
- [x] CHK-016 fab-new.md explanation text matches the YAML template block

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
