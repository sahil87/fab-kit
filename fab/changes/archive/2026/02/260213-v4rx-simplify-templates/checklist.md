# Quality Checklist: Simplify Brief and Spec Templates

**Change**: 260213-v4rx-simplify-templates
**Generated**: 2026-02-13
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Flat Affected Docs List: brief.md template has a single flat list with `(new)`, `(modify)`, `(remove)` inline markers — no subsection headings
- [x] CHK-002 Plain Open Questions: brief.md template uses plain bullets without `[BLOCKING]`/`[DEFERRED]` labels
- [x] CHK-003 No Placeholder Content: spec.md template has no `## Non-Goals`, `## Design Decisions`, or `## Deprecated Requirements` headings with placeholder content
- [x] CHK-004 Guidance Comment: spec.md template includes a single comment block listing all three optional sections with their formats
- [x] CHK-005 Deprecated Requirements as Pattern: spec.md guidance comment documents Deprecated Requirements format (Reason + Migration) and when to use it
- [x] CHK-006 Context Loading Wording: `_context.md` Section 3 step 3 references flat list markers instead of "New, Modified, and Removed" subsections
- [x] CHK-007 Templates Doc Updated: `fab/docs/fab-workflow/templates.md` describes the new flat Affected Docs format and plain Open Questions

## Behavioral Correctness

- [x] CHK-008 SRAD Guidance: brief.md Open Questions comment explains that SRAD handles prioritization at spec time
- [x] CHK-009 Optional Section Patterns: spec.md guidance comment provides enough detail for agents to add Non-Goals, Design Decisions, or Deprecated Requirements when needed

## Scenario Coverage

- [x] CHK-010 Single Modified Doc: brief template supports the single-doc flat list scenario from spec
- [x] CHK-011 Mixed New and Modified: brief template supports multiple entries at the same list level
- [x] CHK-012 Agent Omits Optional Sections: spec template does not nudge agents to fill in optional sections

## Documentation Accuracy

- [x] CHK-013 Templates doc brief.md section: no mention of three headed subsections for Affected Docs
- [x] CHK-014 Templates doc spec.md section: optional sections described as patterns, not standing sections

## Cross References

- [x] CHK-015 `_context.md` wording consistent with brief template format
- [x] CHK-016 `_generation.md` spec procedure compatible with new template (no structural changes needed)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (archive)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
