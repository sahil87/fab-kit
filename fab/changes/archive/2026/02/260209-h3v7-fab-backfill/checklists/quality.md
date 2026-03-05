# Quality Checklist: Add `/fab-backfill` Command

**Change**: 260209-h3v7-fab-backfill
**Generated**: 2026-02-09
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Skill file exists at `fab/.kit/skills/fab-backfill.md` with complete definition
- [x] CHK-002 Symlink exists at `.claude/skills/fab-backfill/SKILL.md` and resolves correctly
- [x] CHK-003 Gap detection cross-references at section level, not just file level (Step 3 in skill: checks headings AND inline mentions)
- [x] CHK-004 Output capped at top 3 gaps with "{N} additional gaps" note when applicable (Step 4 + Step 7 in skill)
- [x] CHK-005 Each gap shows exact markdown preview before confirmation (Step 5 in skill with format template)

## Behavioral Correctness
- [x] CHK-006 Per-gap interactive confirmation: confirm/reject/skip flow described (Step 6: yes/no/done)
- [x] CHK-007 No `fab/current` required — skill works without an active change (Context Loading section: "does not require fab/current")
- [x] CHK-008 Pre-flight checks: abort if `fab/docs/index.md` or `fab/specs/index.md` missing (Pre-flight Check section)

## Scenario Coverage
- [x] CHK-009 No-gaps scenario: "No structural gaps found" message specified (Step 7 + Output section)
- [x] CHK-010 Fewer-than-3 scenario: only existing gaps shown (Step 4: "Take the top 3 gaps" — naturally caps at available count)
- [x] CHK-011 Skip-remaining scenario: user can stop early (Step 6: "done" / "skip rest" option)

## Documentation Accuracy
- [x] CHK-012 `fab/docs/fab-workflow/specs-index.md` updated to reference `/fab-backfill` (Human-Curated Ownership section updated)
- [x] CHK-013 `fab/specs/skills.md` includes `/fab-backfill` entry matching existing style (added after `/fab:status`)
- [x] CHK-014 `fab-help.sh` includes `/fab-backfill` in catalog (new "Maintenance" section)

## Cross References
- [x] CHK-015 Skill definition references `_context.md` preamble (line 8: "Read and follow the instructions in fab/.kit/skills/_context.md")
- [x] CHK-016 Key properties table present and accurate (13 properties listed at end of skill file)

## Notes
- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
