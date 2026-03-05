# Quality Checklist: Add fab-discuss Skill

**Change**: 260220-9ogw-add-fab-discuss
**Generated**: 2026-02-20
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Always-Load Context: Skill loads all 7 files from `_context.md` §1
- [x] CHK-002 No Active Change Required: Skill works without `fab/current` or with empty/missing changes directory
- [x] CHK-003 Orientation Summary: Output includes project identity, memory domains, specs landscape, active change (if any), and ready signal
- [x] CHK-004 Read-Only and Idempotent: Skill modifies no files; repeated invocation produces same output
- [x] CHK-005 Skill Frontmatter: Correct name, description, and model_tier in frontmatter block
- [x] CHK-006 fab-help Integration: `fab-discuss` appears in "Start & Navigate" group
- [x] CHK-007 skills.md Section: New `/fab-discuss` section with purpose, context, properties, output
- [x] CHK-008 Context Loading Fix: Always-loaded list in skills.md updated from 3 to 7 files
- [x] CHK-009 overview.md Row: `/fab-discuss` row in Quick Reference table
- [x] CHK-010 user-flow.md Node: `/fab-discuss` in "Utility (anytime)" subgraph
- [x] CHK-011 context-loading.md Update: Exception Skills section note + changelog entry

## Behavioral Correctness
- [x] CHK-012 Optional files gracefully skipped when missing (context.md, code-quality.md, code-review.md)
- [x] CHK-013 Active change display is light-touch — reads fab/current + .status.yaml only, no deep artifact loading
- [x] CHK-014 No `Next:` pipeline command in output — ends with discussion-mode ready signal

## Scenario Coverage
- [x] CHK-015 All 7 files present: orientation summary lists all loaded files
- [x] CHK-016 Optional files missing: summary notes missing files without error
- [x] CHK-017 No active change: shows "No active change"
- [x] CHK-018 Active change exists: shows change name and current stage

## Edge Cases & Error Handling
- [x] CHK-019 Missing `fab/changes/` directory does not cause skill failure
- [x] CHK-020 Empty `fab/current` file handled gracefully

## Code Quality
- [x] CHK-021 Pattern consistency: Skill file follows naming and structural patterns of existing fab skills
- [x] CHK-022 No unnecessary duplication: Existing conventions reused (e.g., _context.md references)
- [x] CHK-023 Readability: Skill body is clear and maintainable
- [x] CHK-024 No god functions: Skill logic is structured in clear sections

## Documentation Accuracy
- [x] CHK-025 skills.md `/fab-discuss` section matches actual skill behavior
- [x] CHK-026 overview.md row accurately describes the skill
- [x] CHK-027 user-flow.md diagram correctly places fab-discuss in utility subgraph

## Cross References
- [x] CHK-028 context-loading.md memory update is consistent with actual skill behavior
- [x] CHK-029 fab-help.sh group mapping matches where fab-discuss logically belongs

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
