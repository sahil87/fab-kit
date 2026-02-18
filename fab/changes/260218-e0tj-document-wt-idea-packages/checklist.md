# Quality Checklist: Document wt and idea packages

**Change**: 260218-e0tj-document-wt-idea-packages
**Generated**: 2026-02-18
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Packages section in help output: `fab-help.sh` outputs a PACKAGES section after TYPICAL FLOW
- [x] CHK-002 wt-pr included: All 6 wt commands listed in the PACKAGES section (wt-create, wt-list, wt-open, wt-delete, wt-init, wt-pr)
- [x] CHK-003 idea listed: idea command appears in PACKAGES section with description
- [x] CHK-004 Help hint: PACKAGES section ends with "Run <command> help for details."
- [x] CHK-005 packages.md created: `docs/specs/packages.md` exists with all 4 required sections (Overview, wt, idea, Architecture)
- [x] CHK-006 Specs index updated: `docs/specs/index.md` has a row for packages.md

## Behavioral Correctness

- [x] CHK-007 Existing help output preserved: WORKFLOW, COMMANDS, and TYPICAL FLOW sections unchanged
- [x] CHK-008 Formatting consistency: PACKAGES section uses same formatting style (uppercase header, 4-space indent) as other sections

## Scenario Coverage

- [x] CHK-009 User runs fab-help.sh: Output includes PACKAGES after TYPICAL FLOW with correct content
- [x] CHK-010 wt section covers assembly-line: packages.md references assembly-line.md and batch scripts
- [x] CHK-011 idea section covers fab integration: packages.md explains backlog → /fab-new flow

## Code Quality

- [x] CHK-012 Pattern consistency: New code in fab-help.sh follows existing echo/formatting patterns
- [x] CHK-013 No unnecessary duplication: packages.md does not duplicate inline help text

## Documentation Accuracy

- [x] CHK-014 Command descriptions match reality: One-liners in fab-help.sh and packages.md align with actual command behavior
- [x] CHK-015 File paths accurate: All referenced paths (fab/.kit/packages/, fab/backlog.md, etc.) are correct

## Cross References

- [x] CHK-016 assembly-line.md reference: packages.md references docs/specs/assembly-line.md correctly
- [x] CHK-017 Specs index consistency: packages.md description in index matches the page content

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
