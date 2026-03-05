# Tasks: Multi-Agent Support (OpenCode + Codex)

**Change**: 260210-m3k7-multi-agent-support
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Core Implementation

- [x] T001 Refactor `fab/.kit/scripts/fab-setup.sh` symlink section into a reusable function that takes agent name, target directory, and symlink format (directory-based vs flat file) as parameters. Extract the existing Claude Code loop into this function.
- [x] T002 Add OpenCode commands symlink loop to `fab/.kit/scripts/fab-setup.sh` — create `.opencode/commands/` directory, then for each skill create a flat symlink: `.opencode/commands/<name>.md` → `../../fab/.kit/skills/<name>.md`. Exclude `_context.md`. Use the same create/repair/valid counting as Claude Code.
- [x] T003 Add Codex skills symlink loop to `fab/.kit/scripts/fab-setup.sh` — create `.agents/skills/` directory, then for each skill create a directory-based symlink: `.agents/skills/<name>/SKILL.md` → `../../../fab/.kit/skills/<name>.md`. Exclude `_context.md`. Use the same create/repair/valid counting.
- [x] T004 Update the output reporting in `fab/.kit/scripts/fab-setup.sh` to show per-agent counts (Claude Code, OpenCode, Codex) instead of a single aggregate line.

## Phase 2: Verification

- [x] T005 Run `fab/.kit/scripts/fab-setup.sh` and verify all symlinks are created correctly: check that `.claude/skills/`, `.opencode/commands/`, and `.agents/skills/` all contain the expected entries, symlinks resolve, and `_context.md` is excluded from all three.
- [x] T006 Run `fab/.kit/scripts/fab-setup.sh` a second time to verify idempotency — counts should show "already valid" for all agents, no new creates or repairs.

## Phase 3: Documentation

- [x] T007 Update `fab/docs/fab-workflow/kit-architecture.md` — expand the "Agent Integration via Symlinks" section to document all three agent paths (Claude Code, OpenCode commands, Codex) with examples, and correct the outdated `.codex/skills/` reference to `.agents/skills/`.

---

## Execution Order

- T001 blocks T002, T003, T004 (refactor must happen first)
- T002, T003 are independent of each other [P]
- T004 depends on T002 and T003 (needs all loops to exist for reporting)
- T005 depends on T004
- T006 depends on T005
- T007 is independent of all other tasks [P]
