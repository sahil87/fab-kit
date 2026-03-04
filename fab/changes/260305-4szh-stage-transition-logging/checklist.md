# Quality Checklist: Stage Transition Logging

**Change**: 260305-4szh-stage-transition-logging
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 `logman.sh transition` subcommand: Accepts `<change> <stage> <action> [from] [reason] [driver]` and appends correct JSON to `.history.jsonl`
- [ ] CHK-002 `_apply_metrics_side_effect` emission: Calls `logman.sh transition` in the `active` case after incrementing `iterations`
- [ ] CHK-003 Enter vs re-entry: `iterations==1` produces `action=enter`, `iterations>1` produces `action=re-entry`
- [ ] CHK-004 From/reason passthrough: `event_start` and `event_reset` propagate `from`/`reason` to `_apply_metrics_side_effect`
- [ ] CHK-005 Forward flow isolation: `event_finish` auto-activate passes empty `from`/`reason` (always `enter`)
- [ ] CHK-006 `_scripts.md` updated: `transition` subcommand documented in logman section, `[from] [reason]` added to start/reset
- [ ] CHK-007 `change-lifecycle.md` updated: `stage-transition` event documented, `iterations` semantics clarified, canonical review values noted
- [ ] CHK-008 `kit-scripts.md` updated: `transition` subcommand, callers table, `iterations` semantics, canonical review values

## Behavioral Correctness
- [ ] CHK-009 Conditional field omission: `from`, `reason`, `driver` fields omitted from JSON when empty (not present as empty strings)
- [ ] CHK-010 Permissive validation: Non-canonical `action` values accepted without error
- [ ] CHK-011 Backward compatibility: Existing `start` and `reset` calls without `from`/`reason` still work (defaults to empty)

## Scenario Coverage
- [ ] CHK-012 First entry scenario: Stage with no prior metrics produces `enter` event
- [ ] CHK-013 Re-entry scenario: Stage with prior metrics produces `re-entry` event with `from`/`reason`
- [ ] CHK-014 Change resolution failure: Invalid change reference exits with code 1 and stderr error
- [ ] CHK-015 Best-effort logging: Logman call in statusman uses `2>/dev/null || true`

## Edge Cases & Error Handling
- [ ] CHK-016 Missing history file: First transition event creates `.history.jsonl` (append mode)
- [ ] CHK-017 Non-active states: `done`, `pending`, `skipped` states do not emit transition events

## Code Quality
- [ ] CHK-018 Pattern consistency: New `transition` case follows the same structure as existing `command`/`confidence`/`review` cases in logman.sh
- [ ] CHK-019 No unnecessary duplication: Reuses `resolve_change_dir()` helper for change resolution

## Documentation Accuracy
- [ ] CHK-020 Event count accuracy: "Three event types" updated to "Four event types" in change-lifecycle.md
- [ ] CHK-021 Help text accuracy: Both logman.sh and statusman.sh `show_help()` reflect new signatures

## Cross References
- [ ] CHK-022 Callers table consistency: `_scripts.md` callers table matches actual call sites in statusman.sh
