# Quality Checklist: Orchestrator Idle Hooks

**Change**: 260305-bs5x-orchestrator-idle-hooks
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Stop hook writes timestamp: `on-stop.sh` writes `agent.idle_since` as unix integer to `.status.yaml`
- [ ] CHK-002 SessionStart hook clears state: `on-session-start.sh` removes entire `agent` block from `.status.yaml`
- [ ] CHK-003 Sync script registers hooks: `5-sync-hooks.sh` merges hook entries into `.claude/settings.local.json`
- [ ] CHK-004 Hook discovery: sync script discovers `on-*.sh` files from `fab/.kit/hooks/`
- [ ] CHK-005 Filename-to-event mapping: `on-session-start.sh`→`SessionStart`, `on-stop.sh`→`Stop`
- [ ] CHK-006 Hook entry format: entries use `{"type":"command","command":"bash fab/.kit/hooks/{filename}"}`

## Behavioral Correctness

- [ ] CHK-007 Idle semantics: `agent` block present = idle, absent = active
- [ ] CHK-008 SessionStart idempotent: clearing when no `agent` block exists is a no-op
- [ ] CHK-009 Sync idempotent: re-running sync with already-registered hooks produces no changes
- [ ] CHK-010 User hooks preserved: sync appends fab hooks without removing existing user-defined hooks
- [ ] CHK-011 Duplicate detection: sync compares `command` field value, no duplicate entries added

## Scenario Coverage

- [ ] CHK-012 Stop hook: active change writes timestamp (spec scenario)
- [ ] CHK-013 Stop hook: no fab/current exits 0 silently
- [ ] CHK-014 Stop hook: missing change dir exits 0 silently
- [ ] CHK-015 Stop hook: missing .status.yaml exits 0 without creating file
- [ ] CHK-016 SessionStart: active change clears agent block
- [ ] CHK-017 SessionStart: no fab/current exits 0 silently
- [ ] CHK-018 SessionStart: missing .status.yaml exits 0 without modifying files
- [ ] CHK-019 Sync: first sync creates hooks key in settings
- [ ] CHK-020 Sync: preserves user hooks when adding fab hooks
- [ ] CHK-021 Sync: no hook scripts = silent skip
- [ ] CHK-022 Sync: missing settings.local.json creates file
- [ ] CHK-023 Sync: unknown hook script (non-`on-*.sh`) is ignored

## Edge Cases & Error Handling

- [ ] CHK-024 Hooks exit 0 when yq not available
- [ ] CHK-025 Hooks exit 0 when fab dispatcher not available
- [ ] CHK-026 Hooks exit 0 when fab/current contains stale/invalid name
- [ ] CHK-027 Sync warns and skips when jq not available
- [ ] CHK-028 Path resolution works from worktree subdirectory (git rev-parse --show-toplevel)

## Code Quality

- [ ] CHK-029 Pattern consistency: hook scripts follow existing shell script patterns (set -euo pipefail pattern adapted for exit-0 requirement, quoting)
- [ ] CHK-030 No unnecessary duplication: shared resolution logic between the two hook scripts uses similar structure
- [ ] CHK-031 Readability: sync script merge logic is clear and maintainable

## Documentation Accuracy

- [ ] CHK-032 Template not modified: `fab/.kit/templates/status.yaml` has no `agent` block
- [ ] CHK-033 run.sh/dispatch.sh not modified: no changes to pipeline orchestrator scripts

## Cross References

- [ ] CHK-034 Memory files reference new hooks directory and agent schema
- [ ] CHK-035 Setup memory references new sync step

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
