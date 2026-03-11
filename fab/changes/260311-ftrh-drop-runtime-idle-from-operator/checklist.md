# Quality Checklist: Drop runtime is-idle from Operator

**Change**: 260311-ftrh-drop-runtime-idle-from-operator
**Generated**: 2026-03-11
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Operator skill uses pane-map Agent column for idle detection: All six edit locations in `fab/.kit/skills/fab-operator1.md` reference pane-map instead of `fab runtime is-idle`
- [x] CHK-002 Operator spec mirrors skill changes: All seven distinct edit locations in `docs/specs/skills/SPEC-fab-operator1.md` updated to remove `fab runtime is-idle` (spec lists 8 but items 1 and 8 describe the same Summary line edit)

## Behavioral Correctness
- [x] CHK-003 Operator idle-checking policy unchanged: The operator still checks agent idle state before sending — only the data source changed from `fab runtime is-idle` to pane-map Agent column
- [x] CHK-004 No unintended removals: Sections that correctly reference `fab pane-map` (e.g., Discovery, Pane Map Structure) are not modified

## Removal Verification
- [x] CHK-005 No residual `runtime is-idle` in operator skill: `grep -c "runtime is-idle" fab/.kit/skills/fab-operator1.md` returns 0
- [x] CHK-006 No residual `fab runtime` in operator skill: `grep -c "fab runtime" fab/.kit/skills/fab-operator1.md` returns 0
- [x] CHK-007 No residual `runtime is-idle` in operator spec: `grep -c "runtime is-idle" docs/specs/skills/SPEC-fab-operator1.md` returns 0
- [x] CHK-008 No residual `fab runtime` in operator spec: `grep -c "fab runtime" docs/specs/skills/SPEC-fab-operator1.md` returns 0

## Scenario Coverage
- [x] CHK-009 State re-derivation scenario: State Re-derivation section lists only `fab pane-map`
- [x] CHK-010 Pre-send validation scenario: Pre-Send Validation uses pane-map Agent column only
- [x] CHK-011 Broadcast scenario: UC1 references Agent column in pane map
- [x] CHK-012 Unstick scenario: UC6 uses pane-map for idle confirmation
- [x] CHK-013 Autopilot scenario: Per-change loop polls `fab pane-map` only
- [x] CHK-014 Purpose scenario: Purpose statement says `fab pane-map` only

## Code Quality
- [x] CHK-015 Pattern consistency: Edited text follows the existing style and tone of surrounding content
- [x] CHK-016 No unnecessary duplication: No redundant pane-map references introduced

## Documentation Accuracy
- [x] CHK-017 Skill-spec consistency: Every change in the skill file has a corresponding change in the spec file
- [x] CHK-018 Cross-references intact: No broken cross-references or dangling links after edits

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
