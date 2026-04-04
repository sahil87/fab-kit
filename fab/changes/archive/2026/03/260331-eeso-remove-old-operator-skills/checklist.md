# Quality Checklist: Remove Old Operator Skills

**Change**: 260331-eeso-remove-old-operator-skills
**Generated**: 2026-03-31
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Source skills deleted: `fab-operator5.md` and `fab-operator6.md` no longer exist in `fab/.kit/skills/`
- [ ] CHK-002 Launcher scripts deleted: `fab-operator4.sh`, `fab-operator5.sh`, `fab-operator6.sh` no longer exist in `fab/.kit/scripts/`
- [ ] CHK-003 Spec file deleted: `SPEC-fab-operator5.md` no longer exists in `docs/specs/skills/`
- [ ] CHK-004 execution-skills.md updated: no `/fab-operator5` section remains
- [ ] CHK-005 kit-architecture.md updated: directory tree and descriptions reference only `fab-operator7.sh`
- [ ] CHK-006 index.md updated: execution-skills row references `/fab-operator7`
- [ ] CHK-007 superpowers-comparison.md updated: no "operator5/6" version range references

## Behavioral Correctness
- [ ] CHK-008 operator7 files untouched: `fab-operator7.md`, `fab-operator7.sh`, `.claude/skills/fab-operator7/` unchanged
- [ ] CHK-009 Deployed copies untouched: `.claude/skills/fab-operator5/` and `.claude/skills/fab-operator6/` NOT deleted

## Removal Verification
- [ ] CHK-010 No dangling references: grep for `operator5\.md` and `operator6\.md` across repo returns no results (excluding changelog/historical entries)

## Scenario Coverage
- [ ] CHK-011 Memory file coherence: execution-skills.md reads coherently with operator5 section removed — no broken transitions or orphaned references
- [ ] CHK-012 Kit-architecture coherence: directory tree accurately reflects remaining scripts

## Edge Cases & Error Handling
- [ ] CHK-013 Changelog entries preserved: no changelog rows in execution-skills.md deleted
- [ ] CHK-014 operator7 design decisions: "extends operator6" references updated to standalone phrasing

## Code Quality
- [ ] CHK-015 Pattern consistency: memory file updates follow existing formatting and structure
- [ ] CHK-016 No unnecessary duplication: no redundant operator references introduced

## Documentation Accuracy
- [ ] CHK-017 lib/spawn.sh sourcer list: references `fab-operator7.sh` only (not operator4/5)

## Cross References
- [ ] CHK-018 All affected files from spec accounted for in implementation

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
