# Quality Checklist: Interactive Pipeline Pane

**Change**: 260221-ay66-interactive-pipeline-pane
**Generated**: 2026-02-21
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 progress-line command: `stageman.sh progress-line` produces correct visual output for all stage combinations
- [ ] CHK-002 No startup log pane: `run.sh` no longer creates tmux split-window at startup
- [ ] CHK-003 Deferred pane creation: `dispatch.sh` creates interactive pane per-dispatch (horizontal first, vertical subsequent)
- [ ] CHK-004 Interactive Claude session: fab-ff uses `claude --dangerously-skip-permissions '/fab-ff'` (no `-p`)
- [ ] CHK-005 fab-switch unchanged: fab-switch still uses `claude -p --dangerously-skip-permissions`
- [ ] CHK-006 Ship via send-keys: `run.sh` pushes `/changes:ship pr` to interactive pane on `hydrate:done`
- [ ] CHK-007 Ship completion detection: `gh pr view` polling detects PR creation
- [ ] CHK-008 Unified polling loop: `poll_change()` polls every 5s with state machine transitions
- [ ] CHK-009 Pane tracking: `run.sh` maintains pane ID array for SIGINT cleanup
- [ ] CHK-010 Pane ID communication: `dispatch.sh` outputs worktree path + pane ID on stdout

## Behavioral Correctness

- [ ] CHK-011 ship() removed: No `ship()` function exists in `dispatch.sh`
- [ ] CHK-012 LOG_PANE_ID removed: No `LOG_PANE_ID` variable or startup pane creation in `run.sh`
- [ ] CHK-013 Dispatch returns immediately: `dispatch.sh` does not poll or wait after creating interactive pane

## Removal Verification

- [ ] CHK-014 ship() function: Fully removed from `dispatch.sh`, no dead code remains
- [ ] CHK-015 Startup log pane: `tail -f` pane creation removed from `run.sh`, `LOG_PANE_ID` cleanup in SIGINT handler removed

## Scenario Coverage

- [ ] CHK-016 progress-line: Fresh active (`intake ⏳`), mid-pipeline, failed stage, all-done, all-pending, single-done
- [ ] CHK-017 First dispatch creates horizontal split
- [ ] CHK-018 Subsequent dispatch creates vertical split (stacked)
- [ ] CHK-019 fab-ff completion triggers ship send-keys
- [ ] CHK-020 fab-ff failure stops polling
- [ ] CHK-021 Pane death stops polling with error
- [ ] CHK-022 Ship completion (PR detected) marks done

## Edge Cases & Error Handling

- [ ] CHK-023 Timeout: fab-ff timeout (30min) and ship timeout (5min) mark change as failed
- [ ] CHK-024 Pane dies mid-pipeline: Polling detects and transitions to failed
- [ ] CHK-025 SIGINT cleanup: All tracked panes killed, summary printed
- [ ] CHK-026 All-pending progress-line: Returns empty (no output)

## Code Quality

- [ ] CHK-027 Pattern consistency: New code follows existing stageman/pipeline naming and structural patterns
- [ ] CHK-028 No unnecessary duplication: `get_progress_line` reuses `get_progress_map`, `check_pane_alive` is a shared helper
- [ ] CHK-029 Readability: Functions focused, <50 lines where possible
- [ ] CHK-030 No magic strings: Timeouts use named constants/variables

## Documentation Accuracy

- [ ] CHK-031 Memory file updated: `docs/memory/fab-workflow/pipeline-orchestrator.md` reflects new architecture

## Cross References

- [ ] CHK-032 Stageman help text: `progress-line` listed in CLI help output
- [ ] CHK-033 Test coverage: `src/lib/stageman/test.bats` has test cases for all progress-line scenarios

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
