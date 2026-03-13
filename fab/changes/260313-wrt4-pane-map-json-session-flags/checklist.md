# Quality Checklist: Pane Map JSON Session Flags

**Change**: 260313-wrt4-pane-map-json-session-flags
**Generated**: 2026-03-13
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 JSON Output Flag: `--json` produces valid JSON array with all specified fields
- [x] CHK-002 Session Targeting: `--session <name>` targets a specific tmux session and skips `$TMUX` check
- [x] CHK-003 All-Sessions: `--all-sessions` enumerates all sessions and includes panes from each
- [x] CHK-004 Window Index: `window_index` field present in both table and JSON output
- [x] CHK-005 Session Name: `session` field present in JSON; `Session` column in table only with `--all-sessions`

## Behavioral Correctness
- [x] CHK-006 Default table output: No regression — table output without new flags matches prior behavior (minus new WinIdx column)
- [x] CHK-007 Mutual exclusion: `--session` and `--all-sessions` together produce an error
- [x] CHK-008 TMUX guard: `$TMUX` check only applies when neither `--session` nor `--all-sessions` is set
- [x] CHK-009 JSON null semantics: em-dash and `(no change)` values map to `null` in JSON output
- [x] CHK-010 Agent state mapping: `agent_state` and `agent_idle_duration` correctly split from the combined table string

## Scenario Coverage
- [x] CHK-011 JSON output for active pane: stage, agent_state, agent_idle_duration correct
- [x] CHK-012 JSON null for non-fab pane: change, stage, agent_state are null
- [x] CHK-013 JSON idle agent: agent_state is "idle", agent_idle_duration is duration string
- [x] CHK-014 Session targeting from outside tmux: works without `$TMUX` set
- [x] CHK-015 All-sessions mode: panes from multiple sessions appear with correct session names

## Edge Cases & Error Handling
- [x] CHK-016 Invalid session name: tmux error propagated cleanly
- [x] CHK-017 No panes found: graceful "No tmux panes found." in both table and JSON modes
- [x] CHK-018 Agent state "?": maps to `"unknown"` in JSON

## Code Quality
- [x] CHK-019 Pattern consistency: New code follows naming and structural patterns of surrounding panemap.go code
- [x] CHK-020 No unnecessary duplication: Existing utilities (formatIdleDuration, worktreeDisplayPath, etc.) reused where applicable
- [x] CHK-021 Readability: Functions are focused and under ~50 lines where practical

## Documentation Accuracy
- [x] CHK-022 _scripts.md updated: `fab pane-map` section documents new flags and JSON schema

## Cross References
- [x] CHK-023 Memory alignment: kit-architecture.md pane-map section updated to reflect new capabilities

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
