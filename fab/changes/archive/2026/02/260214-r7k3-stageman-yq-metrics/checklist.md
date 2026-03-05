# Quality Checklist: Stage Metrics, History Tracking & Stageman yq Migration

**Change**: 260214-r7k3-stageman-yq-metrics
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 yq dependency: stageman.sh emits error and exits 1 when yq is not on PATH
- [x] CHK-002 Accessor migration: get_progress_map output is identical to pre-migration (key:value format, stage order, defaults)
- [x] CHK-003 Accessor migration: get_checklist output is identical to pre-migration (generated/completed/total with defaults)
- [x] CHK-004 Accessor migration: get_confidence output is identical to pre-migration (all 5 fields with defaults)
- [x] CHK-005 Write migration: set_stage_state preserves validation, atomicity (temp-then-mv), last_updated refresh
- [x] CHK-006 Write migration: transition_stages preserves adjacency check, from-stage-active check, atomicity
- [x] CHK-007 Write migration: set_checklist_field preserves field/value validation
- [x] CHK-008 Write migration: set_confidence_block preserves count/score validation
- [x] CHK-009 Validation migration: validate_status_file checks valid states, active count, ignores stage_metrics
- [x] CHK-010 Schema queries: all awk-based schema functions produce identical output (not migrated)
- [x] CHK-011 stage_metrics: set_stage_state creates/updates metrics on active, sets completed_at on done, removes entry on pending
- [x] CHK-012 stage_metrics: transition_stages triggers both from→done and to→active metrics side-effects
- [x] CHK-013 stage_metrics: driver is required for active transitions, emits error if missing
- [x] CHK-014 stage_metrics: get_stage_metrics returns empty on missing/empty block
- [x] CHK-015 History: log_command appends valid JSON to .history.jsonl, creates file on first event
- [x] CHK-016 History: log_confidence appends confidence event with score/delta/trigger
- [x] CHK-017 History: log_review appends review event with result and optional rework
- [x] CHK-018 Template: status.yaml includes stage_metrics: {} between confidence and last_updated
- [x] CHK-019 calc-score.sh: reads previous values via get_confidence (no direct grep/sed)
- [x] CHK-020 calc-score.sh: calls log_confidence after score computation
- [x] CHK-021 Skill prompts: fab-continue.md passes driver on all transition/set-state calls
- [x] CHK-022 Skill prompts: fab-ff.md passes driver on all transition/set-state calls
- [x] CHK-023 Skill prompts: all updated skills call log_command after preflight

## Behavioral Correctness

- [x] CHK-024 Rework re-activation: set_stage_state increments iterations (not resets) on re-activation
- [x] CHK-025 Reset to pending: clears stage_metrics entry entirely, next activation starts at iterations=1
- [x] CHK-026 Failed state: no metrics side-effect (preserves timing data)

## Scenario Coverage

- [x] CHK-027 First activation scenario: stage_metrics entry created with started_at, driver, iterations=1
- [x] CHK-028 Stage completion scenario: completed_at set, other fields preserved
- [x] CHK-029 Rework scenario: started_at updated, iterations incremented, completed_at removed
- [x] CHK-030 Transition scenario: both stages get correct metrics updates in one call
- [x] CHK-031 CLI commands: set-state, transition, log-command, log-confidence, log-review all work from command line

## Edge Cases & Error Handling

- [x] CHK-032 yq not found: error message includes install URL, exits cleanly
- [x] CHK-033 Missing stage_metrics block: get_stage_metrics returns empty, set_stage_metric creates block
- [x] CHK-034 Driver missing for active: clear error message, no file modification
- [x] CHK-035 Old status files (pre-migration): accessors work with files lacking stage_metrics block

## Documentation Accuracy

- [x] CHK-036 Constitution v1.1.0: Principle I reflects single-binary utility allowance
- [x] CHK-037 CLI help text: --help output includes new commands and updated signatures

## Cross References

- [x] CHK-038 **N/A**: Affected memory files will be verified during hydrate stage
- [x] CHK-039 All existing tests pass after migration (no regressions)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
