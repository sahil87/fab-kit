# Quality Checklist: Add archive index and allow longer folder slugs

**Change**: 260209-r4w8-archive-index-longer-slugs
**Generated**: 2026-02-09
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Archive index file: `/fab-archive` skill includes a step to create/update `fab/changes/archive/index.md`
- [x] CHK-002 Backfill behavior: When `index.md` doesn't exist, the skill specifies scanning all existing archived folders and reading their `proposal.md` Why sections
- [x] CHK-003 Entry format: Index entries use bullet list format `- **{folder-name}** — {description}` with 1-2 sentence description
- [x] CHK-004 Slug word count: `/fab-new` specifies 2-6 words for the slug component
- [x] CHK-005 Slug word count (discuss): `/fab-discuss` references `/fab-new` rules ("same rules as `/fab-new`") which now specify 2-6 words

## Behavioral Correctness

- [x] CHK-006 Most-recent-first ordering: New entries are prepended at the top of the list, not appended at the bottom
- [x] CHK-007 Existing entries preserved: Subsequent archives don't modify existing index entries

## Scenario Coverage

- [x] CHK-008 First archive scenario: Skill definition covers the case where no `index.md` exists yet (backfill flow)
- [x] CHK-009 Subsequent archive scenario: Skill definition covers prepending to existing `index.md`
- [x] CHK-010 Missing proposal fallback: Skill handles archived changes that lack a `proposal.md` (uses folder slug as fallback)

## Edge Cases & Error Handling

- [x] CHK-011 Short slug still valid: 2-word slugs remain valid under the new 2-6 range (minimum is still 2)

## Documentation Accuracy

- [x] CHK-012 Centralized docs consistency: `planning-skills.md` and `change-lifecycle.md` reflect the 2-6 word slug constraint
- [x] CHK-013 Execution skills doc: `execution-skills.md` mentions archive index maintenance in `/fab-archive` behavior

## Cross References

- [x] CHK-014 No stale "2-4 words" references: Grep confirms zero matches for "2-4" across all skill and doc files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
