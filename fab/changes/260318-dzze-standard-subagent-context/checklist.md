# Quality Checklist: Standard Subagent Context Template

**Change**: 260318-dzze-standard-subagent-context
**Generated**: 2026-03-18
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Standard Subagent Context subsection exists in `_preamble.md` after dispatch pattern list
- [x] CHK-002 Dispatch pattern list includes item 6 referencing standard subagent context
- [x] CHK-003 All 5 `fab/project/**` files listed in the standard context subsection
- [x] CHK-004 `fab-continue.md` Review Behavior references `_preamble.md § Standard Subagent Context` instead of inline file list
- [x] CHK-005 `SPEC-preamble.md` created in `docs/specs/skills/`
- [x] CHK-006 `SPEC-fab-continue.md` updated to reflect standard subagent context reference

## Behavioral Correctness
- [x] CHK-007 Review Behavior still lists change-specific context (spec.md, tasks.md, checklist.md, source files, memory files) explicitly
- [x] CHK-008 Apply Behavior `code-quality.md` section reference (`## Principles`, `## Test Strategy`) preserved unchanged

## Scenario Coverage
- [x] CHK-009 Dispatching agent prompt construction: standard context files listed for subagent to read
- [x] CHK-010 Optional files missing: context.md, code-quality.md, code-review.md marked as skip-gracefully
- [x] CHK-011 Nested subagent dispatch: standard context required at every nesting level

## Code Quality
- [x] CHK-012 Pattern consistency: New `_preamble.md` subsection follows existing section formatting (headings, lists, notes)
- [x] CHK-013 No unnecessary duplication: No `fab/project/**` files listed inline in skills that should reference the standard context

## Documentation Accuracy
- [x] CHK-014 `SPEC-preamble.md` accurately reflects current `_preamble.md` structure
- [x] CHK-015 `SPEC-fab-continue.md` review sub-agent box matches updated `fab-continue.md`

## Cross References
- [x] CHK-016 `_preamble.md` standard context file list matches the 5 files in the always-load layer's `fab/project/**` subset
- [x] CHK-017 No stale references to individual `fab/project/**` files remain in `fab-continue.md` Review Behavior

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
