# Quality Checklist: Operator Observation Fixes

**Change**: 260310-b8ff-operator-observation-fixes
**Generated**: 2026-03-10
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Session-scoped pane discovery: `discoverPanes()` uses `tmux list-panes -s` not `-a`
- [x] CHK-002 Tab column in pane map: Output includes 6 columns (Pane, Tab, Worktree, Change, Stage, Agent) with Tab between Pane and Worktree
- [x] CHK-003 Send-keys session scoping: `send-keys` discovers panes via the same session-scoped `discoverPanes()` — no separate `-a` call
- [x] CHK-004 Runtime is-idle subcommand: `fab runtime is-idle <change>` exists and is registered in `runtimeCmd()`
- [x] CHK-005 Is-idle output contract: Prints `idle {duration}`, `active`, or `unknown` per the three conditions; always exits 0
- [x] CHK-006 Operator skill updated: All `status show --all` references replaced with `pane-map` (except outside-tmux fallback)
- [x] CHK-007 Operator spec updated: Primitives table, Discovery section, Use Cases, and Guardrails reference `pane-map` and `runtime is-idle`
- [x] CHK-008 Scripts reference updated: `_scripts.md` reflects Tab column, `is-idle` subcommand, and session-scoped send-keys

## Behavioral Correctness

- [x] CHK-009 Pane map excludes other sessions: In a multi-session environment, only current session panes appear
- [x] CHK-010 Duration format consistency: `is-idle` uses the same `formatIdleDuration` as pane-map (`Ns`, `Nm`, `Nh`)
- [x] CHK-011 Change resolution: `is-idle` uses `resolve.ToFolder` like other runtime subcommands (4-char ID, substring, full name)

## Scenario Coverage

- [x] CHK-012 Agent idle scenario: `fab runtime is-idle` with idle_since 90s ago → `idle 1m`
- [x] CHK-013 Agent active scenario: No agent block → `active`
- [x] CHK-014 Runtime file missing scenario: No `.fab-runtime.yaml` → `unknown`
- [x] CHK-015 Operator outside tmux: Operator fallback uses `fab status show --all` when `$TMUX` unset

## Edge Cases & Error Handling

- [x] CHK-016 Empty window name: `discoverPanes()` handles panes where `#{window_name}` is empty string
- [x] CHK-017 Tab column width calculation: `printPaneTable()` correctly computes max width for the Tab column

## Code Quality

- [x] CHK-018 Pattern consistency: New `runtimeIsIdleCmd()` follows same structure as `runtimeSetIdleCmd()` and `runtimeClearIdleCmd()`
- [x] CHK-019 No unnecessary duplication: `formatIdleDuration` reused from panemap.go (not duplicated in runtime.go)
- [x] CHK-020 Readability: Code follows existing naming conventions and struct patterns

## Documentation Accuracy

- [x] CHK-021 _scripts.md pane-map section: Column table shows 6 columns with Tab; example output includes Tab column
- [x] CHK-022 _scripts.md runtime section: `is-idle` row in subcommand table with correct usage and purpose
- [x] CHK-023 _scripts.md send-keys section: Pane resolution description notes `-s` session scope

## Cross References

- [x] CHK-024 Operator skill ↔ operator spec: Both files reference the same observation commands consistently
- [x] CHK-025 _scripts.md ↔ Go implementation: Documented columns and subcommands match actual code

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
