# Spec: Gemini CLI Agent-Aware Sync

**Change**: 260303-l6nk-gemini-cli-agent-aware-sync
**Created**: 2026-03-03
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Modifying the scaffold tree-walk (section 2) to be agent-aware — scaffold files like `.claude/settings.local.json` continue to deploy unconditionally. Agent-awareness applies only to skill deployment (section 3).
- Adding GitHub Copilot as an agent target — `.github/` is a shared namespace requiring separate analysis.
- Changing the `sync_agent_skills` or `clean_stale_skills` function signatures — the existing abstraction handles Gemini's format natively.

## Sync: Gemini CLI Agent Target

### Requirement: Gemini CLI Skill Deployment

The sync script (`fab/.kit/sync/2-sync-workspace.sh`) SHALL deploy skills to Gemini CLI using the existing `sync_agent_skills` function with:

- **Agent label**: `Gemini`
- **Base directory**: `$repo_root/.gemini/skills`
- **Format**: `directory` (creates `.gemini/skills/<name>/SKILL.md`)
- **Mode**: `copy` (plain copy, no sed templating, no rel_prefix)

The sync script SHALL also call `clean_stale_skills` for the Gemini skills directory to remove stale entries.

#### Scenario: Gemini CLI skills synced on first run

- **GIVEN** Gemini CLI (`gemini`) is available in PATH
- **AND** `.gemini/skills/` does not exist
- **WHEN** `fab-sync.sh` runs
- **THEN** `.gemini/skills/<name>/SKILL.md` is created for every skill in `fab/.kit/skills/`
- **AND** the output includes a `Gemini:` summary line with created/repaired/valid counts

#### Scenario: Gemini CLI skills updated on re-run

- **GIVEN** Gemini CLI is available in PATH
- **AND** `.gemini/skills/` exists with previously synced skills
- **WHEN** `fab-sync.sh` runs after a kit update
- **THEN** modified skills are overwritten (repaired count > 0)
- **AND** stale skill directories not in the current skills list are removed

#### Scenario: Gemini CLI stale skills cleaned

- **GIVEN** `.gemini/skills/old-removed-skill/` exists from a previous sync
- **AND** `old-removed-skill` is no longer in `fab/.kit/skills/`
- **WHEN** `fab-sync.sh` runs
- **THEN** `.gemini/skills/old-removed-skill/` is removed
- **AND** the output includes a "Cleaned: N stale entries" message

### Requirement: Gemini CLI Placement in Sync Script

The Gemini CLI `sync_agent_skills` and `clean_stale_skills` calls SHALL be placed in section 3 ("Skill deployment") of `2-sync-workspace.sh`, after the existing Codex block and before the transitional agent cleanup (section 4).

#### Scenario: Sync order preserved

- **GIVEN** the sync script runs section 3
- **WHEN** all four agents are available
- **THEN** skills are synced in order: Claude Code, OpenCode, Codex, Gemini

## Sync: Agent-Aware Conditional Sync

### Requirement: Agent Detection Before Skill Deployment

The sync script SHALL check whether each agent's CLI command is available in PATH before calling `sync_agent_skills` and `clean_stale_skills` for that agent. Detection SHALL use `command -v <cli-command> >/dev/null 2>&1`.

The agent-to-command mapping SHALL be:

| Agent | CLI command |
|-------|-----------|
| Claude Code | `claude` |
| OpenCode | `opencode` |
| Codex CLI | `codex` |
| Gemini CLI | `gemini` |

#### Scenario: Agent present — skills synced normally

- **GIVEN** `claude` is available in PATH
- **WHEN** `fab-sync.sh` runs
- **THEN** `sync_agent_skills` and `clean_stale_skills` are called for Claude Code
- **AND** output shows the normal summary line (e.g., `Claude Code: 30/30 (created 0, repaired 0, already valid 30)`)

#### Scenario: Agent absent — sync skipped with message

- **GIVEN** `codex` is NOT available in PATH
- **WHEN** `fab-sync.sh` runs
- **THEN** `sync_agent_skills` is NOT called for Codex
- **AND** `clean_stale_skills` is NOT called for Codex
- **AND** the output includes: `Skipping Codex: codex not found in PATH`

#### Scenario: Agent absent — existing dot folder preserved

- **GIVEN** `opencode` is NOT available in PATH
- **AND** `.opencode/commands/` exists with previously synced skills
- **WHEN** `fab-sync.sh` runs
- **THEN** `.opencode/commands/` and its contents are NOT deleted
- **AND** the skip message is printed

### Requirement: No-Agent Warning

When no agents are detected in PATH, the sync script SHALL print a warning but MUST NOT exit with a non-zero code. The warning message SHALL be: `Warning: No agent CLIs found in PATH. Skills were not deployed to any agent.`

#### Scenario: No agents available

- **GIVEN** none of `claude`, `opencode`, `codex`, `gemini` are available in PATH
- **WHEN** `fab-sync.sh` runs
- **THEN** section 3 completes without calling any `sync_agent_skills`
- **AND** the output includes the no-agent warning
- **AND** the script continues to section 4 and beyond (exit code 0)

#### Scenario: Partial agent availability

- **GIVEN** `claude` and `gemini` are available in PATH
- **AND** `opencode` and `codex` are NOT available in PATH
- **WHEN** `fab-sync.sh` runs
- **THEN** skills are synced for Claude Code and Gemini only
- **AND** skip messages are printed for OpenCode and Codex
- **AND** no no-agent warning is printed

### Requirement: Detection Scope

Agent detection SHALL apply only to section 3 ("Skill deployment") of `2-sync-workspace.sh`. The scaffold tree-walk (section 2), directory creation (section 1), transitional agent cleanup (section 4), and sync version stamp (section 5) SHALL remain unconditional.

#### Scenario: Scaffold files deploy regardless of agent availability

- **GIVEN** `claude` is NOT available in PATH
- **AND** `fab/.kit/scaffold/.claude/fragment-settings.local.json` exists
- **WHEN** `fab-sync.sh` runs
- **THEN** `.claude/settings.local.json` is still created/merged by the scaffold tree-walk
- **AND** the Claude Code skill sync is skipped

## Scaffold: Gemini CLI Support

### Requirement: Gitignore Entry for Gemini

The scaffold file `fab/.kit/scaffold/fragment-.gitignore` SHALL include `/.gemini` in the agent-specific section, ensuring the Gemini skills directory is gitignored in downstream projects.

#### Scenario: Gitignore updated with Gemini entry

- **GIVEN** a project runs `fab-sync.sh`
- **WHEN** the scaffold tree-walk processes `fragment-.gitignore`
- **THEN** `.gitignore` contains `/.gemini`
- **AND** the entry is alongside the other agent entries (`.agents`, `.claude`, `.opencode`, etc.)

### Requirement: No Gemini-Specific Scaffold Files

This change SHALL NOT add Gemini-specific scaffold files (e.g., permissions or settings files under `fab/.kit/scaffold/.gemini/`). Gemini CLI does not require equivalent configuration to Claude Code's `settings.local.json`.
<!-- assumed: Gemini CLI needs no scaffold permissions file — no evidence of equivalent config in Gemini CLI docs -->

#### Scenario: No .gemini scaffold directory

- **GIVEN** `fab/.kit/scaffold/` is inspected
- **THEN** no `.gemini/` subdirectory exists within it

## Design Decisions

1. **Detection at section 3 only, not the scaffold walk**: The scaffold tree-walk (section 2) handles many non-agent concerns (gitignore, envrc, memory/specs indexes) and its agent-specific files (`.claude/settings.local.json`) are small configuration files that don't cause harm if the agent is absent. Making the scaffold walk agent-aware would require path-based filtering that adds complexity for minimal benefit.
   - *Why*: Minimal blast radius — agent detection touches only the skill deployment block.
   - *Rejected*: Full agent-awareness in scaffold walk — over-scoped, complicates the generic tree-walk with agent-specific logic.

2. **Individual if-blocks per agent, not a data-driven loop**: Each agent has different parameters (format, mode, rel_prefix). An array-of-structs approach in bash would be less readable than explicit blocks.
   - *Why*: Readability over cleverness (constitution principle). Each block is self-contained and easy to modify independently.
   - *Rejected*: Associative array mapping — bash associative arrays are fragile and harder to read for this use case.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Gemini uses directory-based SKILL.md format | Confirmed from intake #1 — verified via Gemini CLI docs | S:95 R:85 A:95 D:95 |
| 2 | Certain | Agent detection uses `command -v` | Confirmed from intake #2 — standard POSIX, already used in sync prerequisites | S:90 R:90 A:95 D:95 |
| 3 | Certain | Gemini dot folder is `.gemini/skills/` | Confirmed from intake #3 — verified via Gemini CLI docs | S:95 R:85 A:95 D:95 |
| 4 | Certain | Copy mode (not symlink) for Gemini | Upgraded from intake #4 Confident — all directory-based agents (Claude, Codex) use copy mode; consistent pattern | S:85 R:85 A:90 D:95 |
| 5 | Certain | Skip-don't-delete for missing agents | Upgraded from intake #5 Confident — destructive deletion of user state on temporary absence is unacceptable | S:80 R:60 A:90 D:95 |
| 6 | Confident | GitHub Copilot deferred to separate change | Confirmed from intake #6 — `.github/` is shared namespace, warrants separate analysis | S:50 R:80 A:60 D:55 |
| 7 | Certain | Detection scope limited to section 3 (skill deployment) | Scaffold walk is generic and agent-specific files there are harmless; minimal blast radius | S:85 R:85 A:90 D:90 |
| 8 | Confident | Individual if-blocks per agent (not data-driven loop) | Readability principle; each agent has different params; 4 agents doesn't warrant abstraction | S:70 R:90 A:80 D:75 |
| 9 | Tentative | Gemini CLI needs no scaffold permissions file | No evidence of equivalent config in Gemini CLI docs; may need revision if discovered during implementation | S:55 R:80 A:50 D:60 |
<!-- assumed: Gemini CLI needs no scaffold permissions file — no evidence of equivalent config in Gemini CLI docs -->

9 assumptions (5 certain, 2 confident, 1 tentative, 0 unresolved). Run /fab-clarify to review.
