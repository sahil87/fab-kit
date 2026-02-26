# Quality Checklist: Add Ready State to Stage Lifecycle

**Change**: 260226-i9av-add-ready-state-to-stages
**Generated**: 2026-02-26
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Add ready state: workflow.yaml contains `ready` state with symbol, description, terminal: false
- [ ] CHK-002 Remove skipped state: workflow.yaml no longer contains `skipped` state definition
- [ ] CHK-003 Updated allowed_states: all 6 stages include `ready` in allowed_states
- [ ] CHK-004 Updated transitions: `active→ready` and `ready→done` defined in default transitions; `skipped` removed from all transitions
- [ ] CHK-005 Progress line renders ready: `get_progress_line()` displays ready symbol for ready stages
- [ ] CHK-006 Current stage returns ready: `get_current_stage()` returns a ready stage as current
- [ ] CHK-007 Display stage shows ready: `get_display_stage()` returns ready stage with state `ready`
- [ ] CHK-008 Metrics no-op for ready: `_apply_metrics_side_effect()` preserves metrics when setting ready
- [ ] CHK-009 No driver for ready: `set_stage_state()` does not require driver for ready
- [ ] CHK-010 Validation excludes ready from active count: `validate_status_file()` counts only `active`, not `ready`
- [ ] CHK-011 Preflight outputs ready: preflight returns `display_state: ready` when stage is ready
- [ ] CHK-012 calc-score.sh --stage intake: reads intake.md Assumptions table with intake expected_min thresholds
- [ ] CHK-013 fab-continue split dispatch: describes generate (active→ready) and advance (ready→done) as separate actions
- [ ] CHK-014 fab-ff targeted edits: intake start, intake gate, spec gate (post-generation), review stop (no interactive fallback)
- [ ] CHK-015 fab-fff no gates: no confidence gates added to fab-fff
- [ ] CHK-016 fab-clarify accepts ready: stage guard allows `ready` state for taxonomy scanning
- [ ] CHK-017 _preamble.md updated: state vocabulary, gate thresholds reflect new /fab-ff intake gate

## Behavioral Correctness

- [ ] CHK-018 active→ready transition: setting a stage from active to ready works without error
- [ ] CHK-019 ready→done transition: setting a stage from ready to done works without error
- [ ] CHK-020 Progression rule updated: `current_stage` rule includes `ready` alongside `active`
- [ ] CHK-021 Validation rule updated: prerequisites rule no longer references `skipped`

## Removal Verification

- [ ] CHK-022 skipped state removed: no `skipped` references in workflow.yaml states, transitions, allowed_states, progression, or validation
- [ ] CHK-023 skipped in stageman: no hardcoded `skipped` references in stageman.sh (schema-driven functions adapt automatically; verify no leftover prose or comments)

## Scenario Coverage

- [ ] CHK-024 Scenario: ready is valid for spec stage (validate_stage_state returns 0)
- [ ] CHK-025 Scenario: skipped is invalid for all stages (validate_stage_state returns 1)
- [ ] CHK-026 Scenario: progress line with ready stage shows symbol
- [ ] CHK-027 Scenario: current-stage returns ready stage
- [ ] CHK-028 Scenario: set-state ready without driver succeeds
- [ ] CHK-029 Scenario: one active and one ready is valid (active_count=1)
- [ ] CHK-030 Scenario: preflight output for ready stage includes display_state: ready

## Edge Cases & Error Handling

- [ ] CHK-031 active→done path preserved: existing direct active→done transition still works (for execution stages)
- [ ] CHK-032 Review failed + ready interaction: after rework, stages go through ready→done consistently

## Code Quality

- [ ] CHK-033 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [ ] CHK-034 No unnecessary duplication: Existing utilities reused where applicable

## Documentation Accuracy

- [ ] CHK-035 Skill files accurately describe the ready state lifecycle
- [ ] CHK-036 _preamble.md state table and gate thresholds are consistent with implementation

## Cross References

- [ ] CHK-037 All affected memory files listed in intake are addressed during hydrate
- [ ] CHK-038 Spec requirements map to tasks (no uncovered requirements)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
