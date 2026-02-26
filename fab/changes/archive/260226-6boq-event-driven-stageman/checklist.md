# Quality Checklist: Event-Driven Stageman

**Change**: 260226-6boq-event-driven-stageman
**Generated**: 2026-02-26
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Five event commands: `start`, `advance`, `finish`, `reset`, `fail` all exist as CLI subcommands and produce correct state transitions
- [x] CHK-002 Change identifier resolution: all stageman commands (event and non-event) accept change IDs, partial slugs, and file paths
- [x] CHK-003 Finish side-effect: finishing stage N activates stage N+1 when pending; finishing hydrate has no side-effect
- [x] CHK-004 Reset cascade: resetting stage N sets all downstream stages to pending and removes their stage_metrics
- [x] CHK-005 Stage metrics: start/reset increment iterations and set started_at; finish sets completed_at; advance/fail are no-ops
- [x] CHK-006 Driver parameter: optional on all event commands, recorded in stage_metrics when provided, empty when omitted

## Behavioral Correctness
- [x] CHK-007 Event-keyed workflow.yaml: transitions section uses event/from/to format with default and review sections
- [x] CHK-008 Transition lookup: stage-specific override (review) takes precedence over default section
- [x] CHK-009 Review-only events: `fail` only works on review stage; `start` from `failed` only works on review stage
- [x] CHK-010 Illegal transitions rejected: all invalid (current_state, event) combinations exit 1 with diagnostic error

## Removal Verification
- [x] CHK-011 set-state removed: `set_stage_state()` function and `set-state` CLI dispatch case are deleted
- [x] CHK-012 transition removed: `transition_stages()` function and `transition` CLI dispatch case are deleted
- [x] CHK-013 No lingering callers: no skill files or scripts reference `set-state` or `transition` stageman commands

## Scenario Coverage
- [x] CHK-014 Start from pending: verified via test
- [x] CHK-015 Start from failed (review): verified via test
- [x] CHK-016 Advance from active: verified via test
- [x] CHK-017 Finish from active and ready: verified via test
- [x] CHK-018 Reset from done and ready with cascade: verified via test
- [x] CHK-019 Fail from active (review only): verified via test
- [x] CHK-020 Deep pipeline cascade (reset spec while at apply): verified via test

## Edge Cases & Error Handling
- [x] CHK-021 Missing .status.yaml: event commands exit 1 with file-not-found error
- [x] CHK-022 Invalid stage name: event commands exit 1 with invalid-stage error
- [x] CHK-023 Change resolution failure: event commands exit 1 with changeman error
- [x] CHK-024 Finish when next stage is not pending: next stage state preserved (not overwritten)

## Code Quality
- [x] CHK-025 Pattern consistency: new event functions follow the naming and structure patterns of existing stageman functions (atomic writes, validation-first, metrics side-effects)
- [x] CHK-026 No unnecessary duplication: shared logic (resolution, lookup, metrics) extracted into helper functions

## Documentation Accuracy
- [x] CHK-027 change-lifecycle.md: contains state transition table and references event commands (not set-state/transition)
- [x] CHK-028 schemas.md: describes event-keyed transition format
- [x] CHK-029 execution-skills.md: status mutations overview uses event commands
- [x] CHK-030 planning-skills.md: shared generation partial notes use event commands

## Cross References
- [x] CHK-031 Skill files consistent: fab-continue.md, fab-ff.md, fab-fff.md all reference only event commands
- [x] CHK-032 SPEC-changeman.md: references event command instead of set-state

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
