# Quality Checklist: Redefine fab-ff and fab-fff Scope

**Change**: 260215-237b-DEV-1027-redefine-ff-fff-scope
**Generated**: 2026-02-16
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 fab-fff no gate: `/fab-fff` skill file has no confidence gate check in pre-flight
- [x] CHK-002 fab-fff frontloaded questions: `/fab-fff` skill file includes Step 1 frontloaded questions behavior
- [x] CHK-003 fab-fff auto-clarify: `/fab-fff` skill file includes interleaved auto-clarify between planning stages
- [x] CHK-004 fab-fff interactive rework: `/fab-fff` skill file presents interactive rework menu on review failure (not bail)
- [x] CHK-005 fab-ff confidence gate: `/fab-ff` skill file includes confidence gate via `calc-score.sh --check-gate` in pre-flight
- [x] CHK-006 fab-ff spec prerequisite: `/fab-ff` skill file requires spec `active` or later (not intake)
- [x] CHK-007 fab-ff no frontloaded questions: `/fab-ff` skill file does NOT include frontloaded questions step
- [x] CHK-008 fab-ff minimal auto-clarify: `/fab-ff` skill file runs auto-clarify only when tasks are generated
- [x] CHK-009 fab-ff bail on failure: `/fab-ff` skill file bails immediately on review failure (no interactive menu)
- [x] CHK-010 _context.md Next Steps: Table reflects new /fab-ff bail and /fab-fff interactive entries
- [x] CHK-011 _context.md Autonomy Levels: Table swaps /fab-ff and /fab-fff postures
- [x] CHK-012 _context.md Confidence Scoring: Gate reference points to /fab-ff, not /fab-fff
- [x] CHK-013 planning-skills.md: `/fab-ff` and `/fab-fff` sections rewritten to match new skill files
- [x] CHK-014 execution-skills.md: Pipeline invocation note reflects new behavioral differentiation
- [x] CHK-015 change-lifecycle.md: Full pipeline path references updated

## Behavioral Correctness
- [x] CHK-016 fab-fff callable from intake: Skill file allows invocation from any stage at or after intake
- [x] CHK-017 fab-ff callable from spec: Skill file allows invocation from any stage at or after spec
- [x] CHK-018 Driver identification: Both skill files pass their own name as driver to stageman.sh calls

## Removal Verification
- [x] CHK-019 fab-fff gate removed: No `confidence.score` check or `calc-score.sh --check-gate` in fab-fff.md
- [x] CHK-020 fab-ff questions removed: No frontloaded questions step in fab-ff.md
- [x] CHK-021 fab-ff interactive rework removed: No interactive rework menu in fab-ff.md review failure
- [x] CHK-022 fab-fff bail removed: No immediate bail on review failure in fab-fff.md

## Scenario Coverage
- [x] CHK-023 fab-ff gate pass scenario: Skill file describes behavior when confidence > threshold
- [x] CHK-024 fab-ff gate fail scenario: Skill file describes abort with score and guidance message
- [x] CHK-025 fab-ff spec-not-started scenario: Skill file describes abort when spec is pending
- [x] CHK-026 fab-fff full pipeline scenario: Skill file describes execution from intake through hydrate
- [x] CHK-027 fab-fff resume scenario: Skill file describes skipping done stages on re-invocation

## Edge Cases & Error Handling
- [x] CHK-028 fab-ff error table: Error handling table includes spec-not-started and gate-fail conditions
- [x] CHK-029 fab-fff error table: Error handling table includes preflight-fail and intake-missing conditions
- [x] CHK-030 Frontmatter accuracy: Both SKILL.md frontmatter descriptions match new purpose

## Code Quality
- [x] CHK-031 Pattern consistency: Skill files follow the same structure/patterns as other fab skill files
- [x] CHK-032 No unnecessary duplication: Shared generation procedures referenced (not inlined), no duplicated content between skills

## Documentation Accuracy
- [x] CHK-033 Cross-reference consistency: All references to /fab-ff and /fab-fff across _context.md, planning-skills.md, execution-skills.md, change-lifecycle.md are internally consistent
- [x] CHK-034 No stale references: No remaining references to old behavior (fab-fff gating, fab-ff interactive rework) — fixed confidence field in change-lifecycle.md and invocation protocol table in _context.md during review

## Cross References
- [x] CHK-035 Memory changelog entries: All three memory files have changelog entries for this change

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
