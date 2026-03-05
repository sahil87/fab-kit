# Tasks: Gemini CLI Agent-Aware Sync

**Change**: 260303-l6nk-gemini-cli-agent-aware-sync
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 [P] Add `/.gemini` entry to `fab/.kit/scaffold/fragment-.gitignore` in the agent-specific section (alongside `/.agents`, `/.claude`, etc.)

## Phase 2: Core Implementation

- [x] T002 Add `agent_available()` helper function (using `command -v`) and an `agents_found` counter variable to section 3 of `fab/.kit/sync/2-sync-workspace.sh`, placed before the first `sync_agent_skills` call
- [x] T003 Wrap existing Claude Code, OpenCode, and Codex `sync_agent_skills` + `clean_stale_skills` calls in `agent_available` conditionals in `fab/.kit/sync/2-sync-workspace.sh` — each block prints `Skipping {agent}: {command} not found in PATH` when the agent is absent, and increments `agents_found` when present
- [x] T004 Add Gemini CLI sync block after Codex in section 3 of `fab/.kit/sync/2-sync-workspace.sh`: `sync_agent_skills "Gemini" "$repo_root/.gemini/skills" "directory" "copy"` + `clean_stale_skills "$repo_root/.gemini/skills" "directory"`, wrapped in `agent_available "gemini"` conditional
- [x] T005 Add no-agent warning after all agent sync blocks in section 3 of `fab/.kit/sync/2-sync-workspace.sh`: if `agents_found` is 0, print `Warning: No agent CLIs found in PATH. Skills were not deployed to any agent.`

## Phase 3: Integration & Edge Cases

- [x] T006 Update test setup in `src/lib/sync-workspace/test.bats` — create mock agent binaries (`claude`, `opencode`, `codex`, `gemini`) in a temp `$mock_bin` directory, prepend to PATH, so existing tests pass with detection enabled
- [x] T007 Add new test cases to `src/lib/sync-workspace/test.bats`: (a) Gemini skills created with directory format, (b) agent skipped when CLI not in PATH, (c) no-agent warning when none available, (d) existing dot folder preserved when agent absent, (e) skip message printed for absent agent

---

## Execution Order

- T002 blocks T003, T004, T005 (helper function must exist first)
- T003 is independent of T004 (different code blocks in same file)
- T005 depends on T003+T004 (counter logic references all blocks)
- T006 blocks T007 (test infrastructure must exist for new tests)
- T001 is independent of all other tasks
