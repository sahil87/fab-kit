# Quality Checklist: Configurable Agent Spawn Command

**Change**: 260320-t13m-configurable-agent-spawn-command
**Generated**: 2026-03-20
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Config key: `agent.spawn_command` is read from `config.yaml` by all 5 scripts
- [x] CHK-002 Helper function: `fab_spawn_cmd` in `lib/spawn.sh` exists and returns the correct value
- [x] CHK-003 Fallback: scripts fall back to `claude --dangerously-skip-permissions` when key is missing
- [x] CHK-004 Scaffold: `fab/.kit/scaffold/config.yaml` includes the `agent` section with default and comment
- [x] CHK-005 Migration: migration file exists and adds `agent` section to existing configs

## Behavioral Correctness
- [x] CHK-006 Operator scripts: tmux command string correctly concatenates spawn command with skill argument
- [x] CHK-007 Batch scripts: tmux invocations use `$SPAWN_CMD` instead of hardcoded command
- [x] CHK-008 Archive script: exec invocation correctly handles shell expansion via eval
- [x] CHK-009 Shell expansions in spawn_command expand at invocation time, not read time

## Scenario Coverage
- [x] CHK-010 Config present with custom value: scripts use the custom value
- [x] CHK-011 Config missing agent section: scripts use hardcoded fallback
- [x] CHK-012 Config agent section with null spawn_command: scripts use fallback
- [x] CHK-013 Migration skips when agent section already exists

## Edge Cases & Error Handling
- [x] CHK-014 yq not available: helper returns fallback default (stderr suppressed)
- [x] CHK-015 Config file not found: helper returns fallback default

## Code Quality
- [x] CHK-016 Pattern consistency: helper follows existing `lib/` conventions (env-packages.sh pattern)
- [x] CHK-017 No unnecessary duplication: spawn command reading is centralized in one helper, not repeated per script

## Documentation Accuracy
- [x] CHK-018 `_cli-external.md`: tmux new-window example references configurable spawn command
- [x] CHK-019 Migration file follows existing migration format (Summary, Pre-check, Changes, Verification)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
