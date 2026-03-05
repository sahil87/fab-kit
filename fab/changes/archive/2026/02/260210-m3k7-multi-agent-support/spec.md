# Spec: Multi-Agent Support (OpenCode + Codex)

**Change**: 260210-m3k7-multi-agent-support
**Created**: 2026-02-10
**Affected docs**: `fab-workflow/kit-architecture.md`

## Kit Architecture: Multi-Agent Symlinks

### Requirement: Always Create All Agent Integrations

`fab-setup.sh` SHALL create symlinks for all three supported agents on every invocation, unconditionally. There SHALL be no configuration gating, no auto-detection of installed agents, and no user prompt to select agents.

The three agent integrations are:

1. **Claude Code skills**: `.claude/skills/<name>/SKILL.md` → `../../../fab/.kit/skills/<name>.md`
2. **OpenCode commands**: `.opencode/commands/<name>.md` → `../../fab/.kit/skills/<name>.md`
3. **Codex skills**: `.agents/skills/<name>/SKILL.md` → `../../../fab/.kit/skills/<name>.md`

#### Scenario: Fresh Setup

- **GIVEN** a project with `fab/.kit/` present and no existing agent directories
- **WHEN** `fab-setup.sh` is executed
- **THEN** `.claude/skills/`, `.opencode/commands/`, and `.agents/skills/` directories are created
- **AND** each directory contains symlinks for every skill in `fab/.kit/skills/` (excluding `_context.md`)

#### Scenario: Re-run with Existing Symlinks

- **GIVEN** a project where `fab-setup.sh` has already been run
- **WHEN** `fab-setup.sh` is executed again
- **THEN** existing valid symlinks are preserved (not recreated)
- **AND** dangling or broken symlinks are repaired
- **AND** missing symlinks for any agent are created

#### Scenario: New Skill Added to Kit

- **GIVEN** a new `.md` file is added to `fab/.kit/skills/`
- **WHEN** `fab-setup.sh` is executed
- **THEN** the new skill gets symlinks across all three agent directories

### Requirement: Exclude Shared Preamble

`_context.md` SHALL be excluded from symlink creation for all agents. It is a shared context file, not a user-facing skill or command.

#### Scenario: _context.md Not Symlinked

- **GIVEN** `fab/.kit/skills/_context.md` exists
- **WHEN** `fab-setup.sh` is executed
- **THEN** no symlink is created for `_context.md` in any agent directory

### Requirement: Include All User-Facing Skills

All `.md` files in `fab/.kit/skills/` except `_context.md` SHALL receive symlinks across all agents. This includes `retrospect.md` and any future skill files.

#### Scenario: retrospect.md Included

- **GIVEN** `fab/.kit/skills/retrospect.md` exists
- **WHEN** `fab-setup.sh` is executed
- **THEN** symlinks for `retrospect` are created in `.claude/skills/retrospect/SKILL.md`, `.opencode/commands/retrospect.md`, and `.agents/skills/retrospect/SKILL.md`

### Requirement: Correct Symlink Formats Per Agent

Each agent has a different directory structure convention. The symlink format SHALL match each agent's expectations:

- **Claude Code**: Directory-based — `.claude/skills/<name>/SKILL.md` (file named `SKILL.md` inside a skill-named directory)
- **OpenCode**: Flat file — `.opencode/commands/<name>.md` (file named after the skill, directly in commands/)
- **Codex**: Directory-based — `.agents/skills/<name>/SKILL.md` (same shape as Claude Code, different root)

#### Scenario: OpenCode Flat File Format

- **GIVEN** skill `fab-new` exists in `.kit/skills/`
- **WHEN** `fab-setup.sh` creates the OpenCode symlink
- **THEN** the symlink is at `.opencode/commands/fab-new.md` (not `.opencode/commands/fab-new/SKILL.md`)
- **AND** the relative path `../../fab/.kit/skills/fab-new.md` resolves correctly

#### Scenario: Codex Directory Format

- **GIVEN** skill `fab-new` exists in `.kit/skills/`
- **WHEN** `fab-setup.sh` creates the Codex symlink
- **THEN** the symlink is at `.agents/skills/fab-new/SKILL.md`
- **AND** the relative path `../../../fab/.kit/skills/fab-new.md` resolves correctly

### Requirement: Symlink Reporting

`fab-setup.sh` SHALL report symlink counts per agent, following the same format as the existing Claude Code reporting (created, repaired, already valid).

#### Scenario: Status Output

- **GIVEN** `fab-setup.sh` runs with 15 skills
- **WHEN** all three agent integrations are processed
- **THEN** output includes a line per agent, e.g.:
  ```
  Claude Code: 15/15 (created 0, repaired 0, already valid 15)
  OpenCode:    15/15 (created 15, repaired 0, already valid 0)
  Codex:       15/15 (created 15, repaired 0, already valid 0)
  ```

## Deprecated Requirements

_(none)_

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Symlink relative path depth is 2 for OpenCode, 3 for Claude Code and Codex | OpenCode uses flat files (`.opencode/commands/foo.md`), others use subdirectories (`.claude/skills/foo/SKILL.md`) |
| 2 | Confident | No `.gitignore` changes needed for `.opencode/` or `.agents/` | Symlinks should be committed so other team members get them; no generated content in these directories |

2 assumptions made (2 confident, 0 tentative).
