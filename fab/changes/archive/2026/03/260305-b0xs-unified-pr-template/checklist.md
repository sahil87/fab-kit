# Quality Checklist: Unified PR Template

**Change**: 260305-b0xs-unified-pr-template
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Single Template: Step 3c uses one template path, no Tier 1 / Tier 2 branching
- [x] CHK-002 Stats Table: `## Stats` section present with Type, Confidence, Checklist, Tasks, Review columns
- [x] CHK-003 Pipeline Line: Pipeline progress line present below Stats table with done stages in fixed order
- [x] CHK-004 Unified Title: Title derivation uses intake heading when available, commit subject otherwise, regardless of type
- [x] CHK-005 PR Type Reference Cleanup: Table has only Type and Description columns

## Behavioral Correctness

- [x] CHK-006 Confidence Column: Shows `{score} / 5.0` from `.status.yaml`, `—` when unavailable
- [x] CHK-007 Checklist Column: Shows `{completed}/{total}`, appends ` ✓` when complete, `—` when unavailable
- [x] CHK-008 Tasks Column: Parses `tasks.md` checkbox counts, `—` when file missing
- [x] CHK-009 Review Column: Shows `Pass/Fail ({N} iterations)`, omits parenthetical when iterations absent, `—` when not reached
- [x] CHK-010 Pipeline Links: "intake" and "spec" are hyperlinks when files exist, plain text otherwise
- [x] CHK-011 Changes Section: Bulleted list from intake subsections when available, omitted otherwise

## Removal Verification

- [x] CHK-012 Tier 1 template block removed — no "Fab-Linked" conditional path
- [x] CHK-013 Tier 2 template block removed — no "Lightweight" conditional path, no "housekeeping change" footer
- [x] CHK-014 "Fab Pipeline?" column removed from PR Type Reference table
- [x] CHK-015 "Template Tier" column removed from PR Type Reference table
- [x] CHK-016 Type-gated title derivation removed — no branching on fab-linked vs lightweight types

## Scenario Coverage

- [x] CHK-017 Full fab pipeline test-type change: all Stats columns populated, pipeline line with links
- [x] CHK-018 No fab change: Stats shows only Type, all others `—`, pipeline line omitted, Changes omitted
- [x] CHK-019 Partial pipeline: only done stages in pipeline line, incomplete columns show `—`

## Edge Cases & Error Handling

- [x] CHK-020 Missing spec.md: "spec" in pipeline line is plain text, Spec blob URL not generated
- [x] CHK-021 Missing tasks.md: Tasks column shows `—`
- [x] CHK-022 Review iterations not populated: Review column shows `Pass` or `Fail` without parenthetical

## Code Quality

- [x] CHK-023 Pattern consistency: Markdown template structure follows existing git-pr.md conventions
- [x] CHK-024 No unnecessary duplication: Reuses existing blob URL construction, changeman resolve, statusman reads
- [x] CHK-025 Readability: Template logic is linear and readable, no deeply nested conditionals
- [x] CHK-026 No god functions: Template generation instructions are broken into clear steps
- [x] CHK-027 No magic strings: Column headers and field names are clearly documented

## Documentation Accuracy

- [x] CHK-028 Memory update: execution-skills.md "Two-Tier PR Templates" decision revised to reflect unified template
- [x] CHK-029 Changelog: New entry added to execution-skills.md changelog

## Cross References

- [x] CHK-030 Deployed copy synced: `.claude/skills/git-pr.md` matches `fab/.kit/skills/git-pr.md`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
