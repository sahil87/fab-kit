# Quality Checklist: Redesign FF and FFF Pipeline Scopes

**Change**: 260314-q5p9-redesign-ff-fff-scopes
**Generated**: 2026-03-14
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 fab-ff scope: Pipeline stops at hydrate — no ship or review-pr steps in skill file
- [x] CHK-002 fab-fff scope: Pipeline extends through ship and review-pr
- [x] CHK-003 fab-fff gates: Both intake gate and spec gate present in pre-flight/behavior
- [x] CHK-004 fab-fff no frontloaded questions: Step 1 (Frontload All Questions) removed
- [x] CHK-005 --force flag: Both fab-ff.md and fab-fff.md document --force in Arguments
- [x] CHK-006 --force behavior: Gates skipped when --force present, other behavior unchanged
- [x] CHK-007 Preamble SRAD table: fab-ff and fab-fff columns updated, other columns unchanged
- [x] CHK-008 Preamble Confidence Scoring: Both skills described as having identical gates, --force documented
- [x] CHK-009 workflow.yaml: intake commands include fab-ff and fab-fff; tasks commands exclude fab-ff

## Behavioral Correctness
- [x] CHK-010 fab-ff resumability: Checks `hydrate: done` as terminal (not `review-pr: done`)
- [x] CHK-011 fab-ff output format: Ends at `--- Hydrate ---`, no Ship/Review-PR sections
- [x] CHK-012 fab-ff error handling: No ship/review-pr rows in error handling table
- [x] CHK-013 fab-fff output header: Reflects gate info (not "no gate")
- [x] CHK-014 Auto-rework identical: Both skills use same 3-cycle autonomous rework loop
- [x] CHK-015 Auto-clarify identical: Both skills interleave auto-clarify after spec and tasks

## Scenario Coverage
- [x] CHK-016 Diagram 2: fab-ff arrow → hydrate, fab-fff arrow → review-pr, both say "confidence-gated"
- [x] CHK-017 Diagram 3: FF node says "through hydrate", FFF node says "through review-pr"
- [x] CHK-018 Diagram 4: Transition labels match spec (fab-ff → hydrate, fab-fff → review-pr)
- [x] CHK-019 skills.md: Key difference text reflects scope-only differentiation
- [x] CHK-020 SPEC-fab-ff.md: Flow ends at Step 7 Hydrate, no Steps 8-9
- [x] CHK-021 SPEC-fab-fff.md: Gates in flow, no frontload step
- [x] CHK-022 srad.md: Autonomy table matches _preamble.md, gate section mentions both skills

## Code Quality
- [x] CHK-023 Pattern consistency: All edits follow existing naming and structural patterns
- [x] CHK-024 No unnecessary duplication: No duplicated text between fab-ff.md and fab-fff.md beyond inherent shared behavior

## Documentation Accuracy
- [x] CHK-025 planning-skills.md: Overview, requirements, and design decisions reflect new scopes
- [x] CHK-026 execution-skills.md: References to ff/fff pipeline scope updated

## Cross References
- [x] CHK-027 Internal consistency: All files that mention ff/fff scope agree (skill files, preamble, specs, memory, schema)
- [x] CHK-028 No stale references: No remaining mentions of "fff has no confidence gate" or "ff goes through review-pr"

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
