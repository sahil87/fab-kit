# Quality Checklist: Fix Pipeline Ship Timing

**Change**: 260221-6ljc-fix-pipeline-ship-timing
**Generated**: 2026-02-21
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Delay before ship: `poll_change()` sleeps ~8s after detecting `hydrate:done` before sending the ship command
- [ ] CHK-002 Split send-keys: Ship command text and Enter are sent as two separate `tmux send-keys` calls with 0.5s gap
- [ ] CHK-003 Log messages: "waiting for Claude to finish turn..." logged before delay, "Sending /changes:ship pr" logged after delay

## Behavioral Correctness
- [ ] CHK-004 State transition preserved: After delay+send, state still transitions to `shipping` correctly
- [ ] CHK-005 No other polling logic changed: `PIPELINE_FF_TIMEOUT`, `PIPELINE_SHIP_TIMEOUT`, pane-alive checks, progress rendering unchanged

## Scenario Coverage
- [ ] CHK-006 hydrate:done → delay → split send-keys → shipping state: Full happy path matches spec scenario
- [ ] CHK-007 Ship timeout still applies: `PIPELINE_SHIP_TIMEOUT` still triggers failure after delayed send

## Code Quality
- [ ] CHK-008 Pattern consistency: New code follows naming and structural patterns of surrounding code in `run.sh`
- [ ] CHK-009 No unnecessary duplication: Existing log/sleep patterns reused

## Documentation Accuracy
- [ ] CHK-010 Memory update: `pipeline-orchestrator.md` Shipping section updated to document delay and split send-keys

## Cross References
- [ ] CHK-011 Spec alignment: Implementation matches all GIVEN/WHEN/THEN scenarios in spec.md

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
