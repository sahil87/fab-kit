# Quality Checklist: Extract Agent Runtime to Gitignored File

**Change**: 260306-1lwf-extract-agent-runtime-file
**Generated**: 2026-03-06
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 on-stop.sh writes to .fab-runtime.yaml: `agent.idle_since` is written to `.fab-runtime.yaml` keyed by change folder name
- [ ] CHK-002 on-session-start.sh clears from .fab-runtime.yaml: `agent` block is deleted from `.fab-runtime.yaml` for the current change
- [ ] CHK-003 .fab-runtime.yaml is gitignored: Entry exists in `.gitignore` and file does not appear in `git status`
- [ ] CHK-004 schemas.md updated: Ephemeral Runtime State section references `.fab-runtime.yaml` with keyed structure
- [ ] CHK-005 pipeline-orchestrator.md updated: Agent idle signal paragraph references `.fab-runtime.yaml`

## Behavioral Correctness

- [ ] CHK-006 on-stop.sh no longer writes to .status.yaml: The `agent` block in `.status.yaml` is not modified by the hook
- [ ] CHK-007 on-session-start.sh no longer writes to .status.yaml: The `agent` block in `.status.yaml` is not modified by the hook
- [ ] CHK-008 File creation on first write: When `.fab-runtime.yaml` does not exist, `on-stop.sh` creates it with `{}` seed before writing
- [ ] CHK-009 Graceful missing file: `on-session-start.sh` exits 0 when `.fab-runtime.yaml` does not exist

## Scenario Coverage

- [ ] CHK-010 No active change: Both hooks exit 0 without error when `fab/current` is missing or empty
- [ ] CHK-011 Multi-change isolation: Each change's entry is independent — writing one does not affect another
- [ ] CHK-012 yq not installed: Both hooks exit 0 without error when `yq` is not available

## Edge Cases & Error Handling

- [ ] CHK-013 Empty fab/current: Hooks exit cleanly with empty or whitespace-only `fab/current`
- [ ] CHK-014 Missing change directory: Hooks exit cleanly when resolved change directory doesn't exist

## Code Quality

- [ ] CHK-015 Pattern consistency: Hook scripts follow the same guard-clause structure as existing hooks
- [ ] CHK-016 No unnecessary duplication: Common resolution logic is not duplicated between the two hooks

## Documentation Accuracy

- [ ] CHK-017 schemas.md accuracy: The `.fab-runtime.yaml` structure documented matches actual implementation
- [ ] CHK-018 pipeline-orchestrator.md accuracy: The location reference is consistent with schemas.md

## Cross References

- [ ] CHK-019 No stale references: No remaining references to `.status.yaml` agent block in hook scripts or updated docs

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
