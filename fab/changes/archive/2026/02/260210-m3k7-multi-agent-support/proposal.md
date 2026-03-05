# Proposal: Multi-Agent Support (OpenCode + Codex)

**Change**: 260210-m3k7-multi-agent-support
**Created**: 2026-02-10
**Status**: Draft

## Why

The fab kit's architecture was designed for multi-agent portability (Constitution I: "any AI agent that can read markdown and execute shell commands can drive the workflow"), but `fab-setup.sh` only creates Claude Code symlinks. Users working with OpenCode or Codex have to manually wire up skill/command files. This change closes the gap between design intent and implementation.

## What Changes

- **`fab-setup.sh`**: Extend the symlink creation loop to always create integrations for all three agents:
  - **Claude Code** (existing): `.claude/skills/<name>/SKILL.md` → `../../../fab/.kit/skills/<name>.md`
  - **OpenCode commands** (new): `.opencode/commands/<name>.md` → `../../fab/.kit/skills/<name>.md`
  - **Codex skills** (new): `.agents/skills/<name>/SKILL.md` → `../../../fab/.kit/skills/<name>.md`
- All three agent directories are always created — no auto-detection, no config gating
- The `_context.md` exclusion rule applies to all agents (it's a shared preamble, not a command)
- `retrospect.md` is included as a user-facing command for all agents
- No changes to `fab/config.yaml` — no new config knobs needed
- No changes to skill file content — existing YAML frontmatter (`name` + `description`) is already compatible with all three agents

## Affected Docs

### Modified Docs
- `fab-workflow/kit-architecture`: Update "Agent Integration via Symlinks" section to document OpenCode and Codex paths alongside Claude Code. Update the example to show all three patterns.

### New Docs
_(none)_

### Removed Docs
_(none)_

## Impact

- **`fab/.kit/scripts/fab-setup.sh`**: Primary change target. Add two new symlink loops (OpenCode commands, Codex skills) parallel to the existing Claude Code loop.
- **New directories created by setup**: `.opencode/commands/`, `.agents/skills/` — harmless if the user doesn't use those agents.
- **`.gitignore`**: May need entries for `.opencode/` and `.agents/` if they contain generated content (symlinks are fine to commit).
- **Existing Claude Code integration**: Untouched — this is purely additive.

## Open Questions

_(none — all decisions resolved during discussion)_

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Codex uses `.agents/skills/<name>/SKILL.md` path | Documented at developers.openai.com/codex/skills/ |
| 2 | Confident | No skill content changes needed | Verified: existing frontmatter (`name` + `description`) satisfies OpenCode and Codex requirements |
| 3 | Confident | Kit-architecture doc should be updated | Natural follow-through — documents the new multi-agent paths |
| 4 | Confident | Symlink relative paths are correct for each agent's directory depth | Based on directory structure: `.opencode/commands/` is 2 levels deep, `.claude/skills/<name>/` and `.agents/skills/<name>/` are 3 levels deep |

4 assumptions made (4 confident, 0 tentative).
