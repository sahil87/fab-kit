# Quality Checklist: Pane Capture Tmux Flag Fix

**Change**: 260406-x33u-pane-capture-tmux-flag
**Generated**: 2026-04-06
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Correct tmux flag in capturePaneContent: `src/go/fab/cmd/fab/pane_capture.go` uses `-S -N` (not `-l N`) when invoking `tmux capture-pane`
- [ ] CHK-002 capturePaneArgs extracted: A standalone `capturePaneArgs(paneID string, lines int) []string` function exists in `pane_capture.go`
- [ ] CHK-003 Operator skill doc fixed: `src/kit/skills/fab-operator.md` Question Detection step 1 shows `tmux capture-pane -t <pane> -p -S -20`
- [ ] CHK-004 Operator spec doc fixed: `docs/specs/skills/SPEC-fab-operator.md` shows `-S -20` at both the feature summary and the Question Detection detail

## Behavioral Correctness

- [ ] CHK-005 capturePaneContent delegates to capturePaneArgs: `capturePaneContent` calls `capturePaneArgs` and spreads the result into `exec.Command("tmux", ...)`
- [ ] CHK-006 Negative line offset: `capturePaneArgs("%5", 50)` returns `["capture-pane", "-t", "%5", "-p", "-S", "-50"]` — the offset is negative

## Scenario Coverage

- [ ] CHK-007 Default 50-line capture: `TestCapturePaneArgs` covers `capturePaneArgs("%5", 50)` returning correct slice
- [ ] CHK-008 Custom line count: `TestCapturePaneArgs` covers `capturePaneArgs("%3", 20)` returning `"-S", "-20"`

## Edge Cases & Error Handling

- [ ] CHK-009 Line count format: Negative offset is formatted as `fmt.Sprintf("-%d", lines)` — no double-negative for any valid positive input

## Code Quality

- [ ] CHK-010 Pattern consistency: `capturePaneArgs` follows the same naming and signature style as other helpers in `pane_capture.go`
- [ ] CHK-011 No unnecessary duplication: Existing `exec.Command` pattern is preserved; `capturePaneArgs` is used exactly once (in `capturePaneContent`)
- [ ] CHK-012 Readability: The extraction makes the intent of the tmux invocation clearer than inline string construction

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-001 **N/A**: {reason}`
