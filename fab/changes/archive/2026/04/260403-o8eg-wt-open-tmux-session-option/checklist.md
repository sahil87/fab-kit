# Quality Checklist: wt open — Add tmux session option

**Change**: 260403-o8eg-wt-open-tmux-session-option
**Generated**: 2026-04-03
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 App detection: `tmux_session` entry appears in `BuildAvailableApps` when `IsTmuxSession()` is true
- [x] CHK-002 App detection ordering: `tmux_session` appears immediately after `tmux_window` in the app list
- [x] CHK-003 Open in tmux session: `OpenInApp("tmux_session", ...)` runs `tmux new-session -d -s {name} -c {path}`
- [x] CHK-004 Session naming: session name follows `{repoName}-{wtName}` pattern
- [x] CHK-005 Exit code reuse: tmux_session errors produce `ExitTmuxWindowError` (6)
- [x] CHK-006 Default app unchanged: `DetectDefaultApp` still returns `tmux_window` in tmux (not `tmux_session`)

## Scenario Coverage
- [x] CHK-007 tmux session option appears in tmux: menu shows "tmux session" when TMUX is set
- [x] CHK-008 tmux session option absent outside tmux: no "tmux session" entry when TMUX is unset
- [x] CHK-009 tmux session option absent in byobu: no "tmux session" entry in byobu sessions
- [x] CHK-010 tmux not available: returns descriptive error when tmux binary is not on PATH
- [x] CHK-011 tmux new-session fails: returns descriptive error with stderr content

## Code Quality
- [x] CHK-012 Pattern consistency: new code follows the same structure as existing `tmux_window` and `byobu_tab` entries
- [x] CHK-013 No unnecessary duplication: reuses existing patterns (LookPath check, CombinedOutput, error formatting)

## Documentation Accuracy
- [x] CHK-014 **N/A**: Memory file update happens during hydrate stage

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
