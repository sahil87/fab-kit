# Quality Checklist: Archive Restore Mode

**Change**: 260214-v7k3-archive-restore-mode
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness
<!-- Every requirement in spec.md has working implementation -->
- [x] CHK-001 Restore Subcommand: `/fab-archive restore <change-name>` syntax documented with required argument
- [x] CHK-002 Artifact Preservation: Skill explicitly states no status reset, no artifact modification during restore
- [x] CHK-003 Archive Index Cleanup: Entry removal from `archive/index.md` documented, empty index preserved
- [x] CHK-004 Idempotent Restore: Re-run detection (folder already in changes) documented with skip-and-complete behavior
- [x] CHK-005 Restore Output Format: Structured summary with Moved/Index/Pointer lines and Next suggestion
- [x] CHK-006 Lifecycle Transition: `archived → active` transition documented as inverse of archive

## Behavioral Correctness
<!-- Changed requirements behave as specified, not as before -->
- [x] CHK-007 Existing archive behavior unchanged: Original `/fab-archive` flow (move, index, backlog, pointer) not altered
- [x] CHK-008 Name resolution uses archive folder: Restore matches against `fab/changes/archive/`, not `fab/changes/`

## Scenario Coverage
<!-- Key scenarios from spec.md have been exercised -->
- [x] CHK-009 Successful restore scenario: Folder move + index removal + no pointer change
- [x] CHK-010 Restore with --switch: Folder move + index removal + pointer written
- [x] CHK-011 Ambiguous match: Lists matches and asks user to pick
- [x] CHK-012 No match found: Lists all archived changes, informs user
- [x] CHK-013 Re-run after partial restore: Skip move, complete remaining steps
- [x] CHK-014 Lifecycle round-trip: Restored change visible in `/fab-switch` listings

## Edge Cases & Error Handling
<!-- Error states, boundary conditions, failure modes -->
- [x] CHK-015 No archived changes exist: Outputs "No archived changes found."
- [x] CHK-016 Archive folder missing: Graceful error handling documented
- [x] CHK-017 Empty index after removal: Index file preserved (not deleted)

## Documentation Accuracy
<!-- Extra category from config.yaml -->
- [x] CHK-018 Restore section consistent with existing archive section conventions (step numbering, resumability pattern)
- [x] CHK-019 Arguments section reflects both archive and restore modes

## Cross References
<!-- Extra category from config.yaml -->
- [x] CHK-020 Memory files referenced in spec (execution-skills, change-lifecycle) match actual affected areas

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
