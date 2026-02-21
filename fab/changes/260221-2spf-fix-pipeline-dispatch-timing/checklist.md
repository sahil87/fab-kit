# Quality Checklist: Fix Pipeline Dispatch Visibility

**Change**: 260221-2spf-fix-pipeline-dispatch-timing
**Generated**: 2026-02-21
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Interactive pane created first: bare `claude --dangerously-skip-permissions` session starts before any fab-switch execution
- [ ] CHK-002 fab-switch via send-keys: `/fab-switch $CHANGE_ID --no-branch-change` sent to pane via `tmux send-keys` (split: text then Enter)
- [ ] CHK-003 fab/current polling: dispatch.sh polls `$wt_path/fab/current` until content matches `$CHANGE_ID`
- [ ] CHK-004 fab-ff via send-keys: `/fab-ff` sent after switch confirmation via `tmux send-keys` (split: text then Enter)
- [ ] CHK-005 Output contract: dispatch.sh stdout emits exactly two lines — worktree path (line 1) and pane ID (line 2)

## Behavioral Correctness
- [ ] CHK-006 No claude -p call: the previous `claude -p --dangerously-skip-permissions "/fab-switch ..."` call is fully removed
- [ ] CHK-007 First dispatch uses horizontal split (`-h`), subsequent uses vertical split (`-v -t "$LAST_PANE_ID"`)
- [ ] CHK-008 Startup delay before sending fab-switch allows Claude to initialize

## Removal Verification
- [ ] CHK-009 Deprecated `claude -p` fab-switch: no print-mode Claude invocation remains in `run_pipeline()`

## Scenario Coverage
- [ ] CHK-010 First dispatch horizontal split: pane created with `-h -d -P -F '#{pane_id}'`
- [ ] CHK-011 Subsequent dispatch vertical split: pane created with `-v -t "$LAST_PANE_ID" -d -P -F '#{pane_id}'`
- [ ] CHK-012 fab/current matches expected change ID: polling stops, fab-ff sent
- [ ] CHK-013 fab/current polling times out: change marked `failed`, dispatch returns pane ID
- [ ] CHK-014 Pane dies during fab-switch polling: change marked `failed`, dispatch returns

## Edge Cases & Error Handling
- [ ] CHK-015 Polling timeout uses configurable 60s default
- [ ] CHK-016 Pane alive check uses `check_pane_alive()` (already defined in dispatch.sh)
- [ ] CHK-017 Failed change still returns pane ID (for run.sh tracking)

## Code Quality
- [ ] CHK-018 Pattern consistency: send-keys pattern matches existing tmux usage in the codebase
- [ ] CHK-019 No unnecessary duplication: reuses existing `check_pane_alive()`, `write_stage()`, `log()` helpers

## Documentation Accuracy
- [ ] CHK-020 Memory hydration reflects the new dispatch flow accurately

## Cross References
- [ ] CHK-021 run.sh's `poll_change()` and stdout parsing remain compatible with dispatch.sh's output

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
