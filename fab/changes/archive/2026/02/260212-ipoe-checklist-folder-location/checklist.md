# Quality Checklist: Re-evaluate Checklist Folder Location

**Change**: 260212-ipoe-checklist-folder-location
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Status template: `fab/.kit/templates/status.yaml` has `checklist.path` set to `checklist.md`
- [x] CHK-002 `/fab-new`: No longer creates `checklists/` subdirectory
- [x] CHK-003 `_generation.md`: References `fab/changes/{name}/checklist.md` for checklist output path
- [x] CHK-004 `/fab-continue`: Reset path references `checklist.md` (not `checklists/quality.md`)
- [x] CHK-005 `/fab-ff`: All checklist references use `checklist.md`
- [x] CHK-006 `/fab-review`: All checklist references use `checklist.md` (7 references verified)
- [x] CHK-007 `/fab-archive`: All checklist references use `checklist.md` (3 references verified)

## Behavioral Correctness

- [x] CHK-008 No skill file contains the string `checklists/quality.md` after changes — grep confirmed 0 matches in fab/.kit/
- [x] CHK-009 No skill file creates or references a `checklists/` directory — grep confirmed 0 matches in fab/.kit/

## Scenario Coverage

- [x] CHK-010 **N/A**: No active changes had `checklists/quality.md` — migration was a no-op
- [x] CHK-011 Active changes without checklists are untouched by migration — verified, no changes modified
- [x] CHK-012 Archived changes are not modified by migration — verified, 37 archived changes still have `checklists/quality.md` intact

## Edge Cases & Error Handling

- [x] CHK-013 `/fab-review` missing checklist error message references `checklist.md` (not `quality.md`) — verified at line 40 and 290

## Documentation Accuracy

- [x] CHK-014 Centralized docs will be updated during archive hydration — spec correctly identifies all 5 affected docs

## Cross References

- [x] CHK-015 All affected centralized docs (`templates.md`, `planning-skills.md`, `execution-skills.md`, `change-lifecycle.md`, `kit-architecture.md`) listed in spec Affected Docs — archive hydration will update them

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
