# Quality Checklist: Operator Enforce Pipeline Routing

**Change**: 260326-u3un-operator-enforce-pipeline-routing
**Generated**: 2026-03-26
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Pipeline-first routing principle: §1 contains a new principle that mandates routing all new work through `/fab-new` then pipeline commands
- [x] CHK-002 §6 reinforcing note: A blockquote note appears at the top of "Working a Change" before the three work paths
- [x] CHK-003 Maintenance exemption: The §1 principle explicitly exempts operational maintenance commands

## Behavioral Correctness
- [x] CHK-004 Prohibition scope: The principle prohibits raw implementation instructions AND `/fab-continue` without prior `/fab-new` for new work
- [x] CHK-005 Exemption alignment: The exemption list aligns with the existing "Coordinate, don't execute" principle's carveout

## Scenario Coverage
- [x] CHK-006 Backlog routing scenario: New backlog items route through `/fab-new <id>`, not raw instructions
- [x] CHK-007 Raw text routing scenario: Free-form requests route through `idea add` → structured flow → `/fab-new`
- [x] CHK-008 Maintenance scenario: Archive, merge, rebase remain direct operator actions

## Documentation Accuracy
- [x] CHK-009 Memory hydration: `docs/memory/fab-workflow/execution-skills.md` updated with design decision and changelog entry

## Cross References
- [x] CHK-010 §6 note references §1 principle: The reinforcing note in §6 references the principle defined in §1

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
