# Quality Checklist: Multi-Agent Support (OpenCode + Codex)

**Change**: 260210-m3k7-multi-agent-support
**Generated**: 2026-02-10
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Always Create All Agent Integrations: `fab-setup.sh` creates symlinks for Claude Code, OpenCode, and Codex unconditionally on every run
- [x] CHK-002 Correct Symlink Formats: Claude Code uses `.claude/skills/<name>/SKILL.md`, OpenCode uses `.opencode/commands/<name>.md`, Codex uses `.agents/skills/<name>/SKILL.md`
- [x] CHK-003 Exclude Shared Preamble: `_context.md` has no symlinks in any agent directory
- [x] CHK-004 Include All User-Facing Skills: `retrospect.md` and all other non-`_context.md` skills have symlinks across all three agents
- [x] CHK-005 Symlink Reporting: Output shows per-agent counts (created, repaired, already valid)

## Scenario Coverage

- [x] CHK-006 Fresh Setup: Running on a project with no agent directories creates all three directory trees with correct symlinks
- [x] CHK-007 Re-run Idempotency: Running a second time shows "already valid" for all agents, no creates or repairs
- [x] CHK-008 OpenCode Flat File: `.opencode/commands/fab-new.md` exists as a flat symlink (not inside a subdirectory)
- [x] CHK-009 Codex Directory: `.agents/skills/fab-new/SKILL.md` exists inside a skill-named directory
- [x] CHK-010 Symlink Resolution: All symlinks across all agents resolve to existing `.kit/skills/*.md` files

## Edge Cases & Error Handling

- [x] CHK-011 Dangling Symlinks: Broken symlinks in any agent directory are repaired on re-run
- [x] CHK-012 Regular File Replacement: If a regular file exists where a symlink is expected, it is replaced with the correct symlink

## Documentation Accuracy

- [x] CHK-013 Kit Architecture Doc: `fab/docs/fab-workflow/kit-architecture.md` documents all three agent paths with correct examples
- [x] CHK-014 No Stale References: Old `.codex/skills/` reference in kit-architecture doc is corrected to `.agents/skills/`

## Cross References

- [x] CHK-015 Proposal-to-Spec Alignment: All "What Changes" items in proposal.md have corresponding spec requirements
- [x] CHK-016 Spec-to-Tasks Coverage: Every spec requirement has at least one task covering it

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
