# Quality Checklist: Gemini CLI Agent-Aware Sync

**Change**: 260303-l6nk-gemini-cli-agent-aware-sync
**Generated**: 2026-03-04
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Gemini CLI skill deployment: `.gemini/skills/<name>/SKILL.md` created for every skill when `gemini` available
- [x] CHK-002 Agent detection: `command -v` used to check each agent's CLI command before syncing
- [x] CHK-003 Skip message: `Skipping {agent}: {command} not found in PATH` printed for absent agents
- [x] CHK-004 No-agent warning: warning printed when no agents detected, script continues (exit 0)
- [x] CHK-005 Gitignore entry: `/.gemini` present in `fab/.kit/scaffold/fragment-.gitignore`

## Behavioral Correctness

- [x] CHK-006 Existing agent sync unchanged: Claude Code, OpenCode, Codex sync behavior identical when their CLIs are present
- [x] CHK-007 Existing dot folders preserved: absent agents do not cause deletion of existing dot folders
- [x] CHK-008 Detection scope: scaffold tree-walk (section 2) remains unconditional

## Scenario Coverage

- [x] CHK-009 Gemini skills synced on first run: verified via test
- [x] CHK-010 Agent absent — sync skipped: verified via test
- [x] CHK-011 No agents — warning printed: verified via test
- [x] CHK-012 Partial availability — only detected agents synced: verified via test
- [x] CHK-013 Stale Gemini skills cleaned: `clean_stale_skills` called for `.gemini/skills/`

## Edge Cases & Error Handling

- [x] CHK-014 Script exits 0 even when no agents found
- [x] CHK-015 Idempotency: running sync twice with same agents produces same result

## Code Quality

- [x] CHK-016 Pattern consistency: agent block structure matches existing Claude/OpenCode/Codex pattern
- [x] CHK-017 No unnecessary duplication: reuses `sync_agent_skills` and `clean_stale_skills` (no new copy functions)
- [x] CHK-018 Readability: follows project principle of readability over cleverness
- [x] CHK-019 No god functions: helper function is small and focused
- [x] CHK-020 No magic strings: agent names and commands appear as clear literals

## Documentation Accuracy

- [x] CHK-021 Gitignore scaffold includes `/.gemini` entry

## Cross References

- [x] CHK-022 Spec matches implementation: all spec requirements have corresponding code

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
