# Quality Checklist: PR Change Metadata

**Change**: 260312-9r3t-pr-change-metadata
**Generated**: 2026-03-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Change Table in PR Body: git-pr skill generates a "Change" section above "Stats" with ID, Name, Issue columns
- [x] CHK-002 Change Table Format: Table uses exact markdown format specified in spec (## Change header, 3-column table)
- [x] CHK-003 Optional linear_workspace Field: `fab/project/config.yaml` supports the `linear_workspace` field under `project:` block
- [x] CHK-004 URL Construction: When `linear_workspace` is configured, issues render as `[{ID}](https://linear.app/{workspace}/issue/{ID})`
- [x] CHK-005 Migration File: `fab/.kit/migrations/0.34.0-to-0.37.0.md` exists with correct structure
- [x] CHK-006 SPEC-git-pr.md Update: Flow diagram and tools table reflect the new Change section

## Behavioral Correctness

- [x] CHK-007 has_fab gating: Change section is omitted entirely when no active fab change resolves
- [x] CHK-008 Bare ID fallback: Issues show as bare text when `linear_workspace` is absent
- [x] CHK-009 Em-dash fallback: Missing fields (id, name, issues) show `—` not blank or omitted

## Scenario Coverage

- [x] CHK-010 Full metadata with Linear workspace: ID + name + hyperlinked issue renders correctly
- [x] CHK-011 Multiple issues: Comma-separated hyperlinked issues in Issue column
- [x] CHK-012 No issues: Issue column shows `—`
- [x] CHK-013 No active fab change: Change section omitted entirely
- [x] CHK-014 Migration skip when field exists: Migration detects existing `linear_workspace` and skips
- [x] CHK-015 Migration adds commented-out field: Adds `# linear_workspace: "your-workspace"` under `project:`

## Edge Cases & Error Handling

- [x] CHK-016 ID/name unavailable: Missing `.status.yaml` fields result in `—` columns, not errors
- [x] CHK-017 No config.yaml for migration: Migration handles missing config gracefully

## Code Quality

- [x] CHK-018 Pattern consistency: New skill instructions follow existing git-pr patterns (conditional sections, field population, fallbacks)
- [x] CHK-019 No unnecessary duplication: Reuses existing `{has_fab}`, `{name}`, issues resolution from Step 1

## Documentation Accuracy

- [x] CHK-020 Spec alignment: SPEC-git-pr.md accurately reflects the implemented skill changes
- [x] CHK-021 Migration instructions: Migration file follows the established format (Summary, Pre-check, Changes, Verification)

## Cross References

- [x] CHK-022 Config field documented: `linear_workspace` appears in config.yaml and migration references it
- [x] CHK-023 Constitution compliance: Skill file change has corresponding spec update (per constitution Additional Constraints)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
