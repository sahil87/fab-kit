# Spec: Copy-Template Skills, Drop Agents

**Change**: 260219-d2y2-copy-template-skills-drop-agents
**Created**: 2026-02-19
**Affected memory**: `docs/memory/fab-workflow/model-tiers.md`, `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Changing OpenCode or Codex deployment modes — they remain as-is (symlinks and copies respectively)
- Modifying the model tier classification system itself — `fast` and `capable` tiers are unchanged
- Altering canonical skill files in `fab/.kit/skills/` — they retain `model_tier:` for provider-agnosticism; only deployed copies get `model:`

## Sync Workspace: Claude Code Skill Deployment

### Requirement: Copy-with-Template Mode

`2-sync-workspace.sh` SHALL deploy Claude Code skills using copy mode instead of symlink mode. During the copy, skills with a `model_tier:` field in frontmatter SHALL have that field replaced with a provider-specific `model:` field.

- For skills with `model_tier: fast`: the line SHALL be replaced with `model: {resolved_model}` where `{resolved_model}` comes from `config.yaml` `model_tiers.fast.claude` (fallback: `haiku`)
- For skills without `model_tier`: the file SHALL be copied verbatim — no `model:` line injected

The `sync_agent_skills` call for Claude Code SHALL change from symlink mode to copy mode. The model tier templating MAY be implemented within `sync_agent_skills` or as a post-copy step — the implementation choice is left to the tasks stage.

#### Scenario: Fast-tier skill deployed with model substitution

- **GIVEN** `fab/.kit/skills/fab-help.md` contains `model_tier: fast` in frontmatter
- **AND** `config.yaml` maps `model_tiers.fast.claude` to `haiku`
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** `.claude/skills/fab-help/SKILL.md` is a regular file (not a symlink)
- **AND** the file contains `model: haiku` where the source had `model_tier: fast`
- **AND** all other content is identical to the source

#### Scenario: Capable-tier skill deployed verbatim

- **GIVEN** `fab/.kit/skills/fab-continue.md` has no `model_tier` field
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** `.claude/skills/fab-continue/SKILL.md` is a regular file (not a symlink)
- **AND** the file is an exact copy of the source

#### Scenario: Config missing model_tiers section

- **GIVEN** `config.yaml` has no `model_tiers:` section
- **AND** `fab/.kit/skills/fab-status.md` contains `model_tier: fast`
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** `.claude/skills/fab-status/SKILL.md` contains `model: haiku` (hardcoded fallback)

#### Scenario: Idempotent re-run with no changes

- **GIVEN** `.claude/skills/fab-help/SKILL.md` already exists with correct templated content
- **WHEN** `2-sync-workspace.sh` runs again
- **THEN** the file is NOT rewritten
- **AND** the skill is counted as "already valid" (not "repaired")

### Requirement: Agent File Generation Removal

`2-sync-workspace.sh` SHALL NOT generate `.claude/agents/` files for fast-tier skills. The entire "Section 4: Model tier agent files" block — including the generation loop and the stale agent cleanup logic — SHALL be removed.

#### Scenario: Sync produces no agent files

- **GIVEN** fast-tier skills `fab-help`, `fab-setup`, `fab-status`, `fab-switch` exist
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** no files are created or updated in `.claude/agents/`
- **AND** the "Agents:" output line is no longer printed

### Requirement: Transitional Agent Cleanup

`2-sync-workspace.sh` SHALL include a transitional cleanup step that removes `.claude/agents/` files whose basename (without `.md`) matches any entry in the `skills[]` array.

Files in `.claude/agents/` that do NOT match a known skill name SHALL be preserved (user-created agents).

This cleanup step is idempotent and SHOULD be removed in a future release once downstream projects have been updated.

#### Scenario: Stale agent files removed on first sync after upgrade

- **GIVEN** `.claude/agents/fab-help.md` and `.claude/agents/fab-status.md` exist from a previous version
- **WHEN** `2-sync-workspace.sh` runs (new version with this change)
- **THEN** both files are removed
- **AND** a message like "Cleaned: 2 stale agent files from .claude/agents/" is displayed

#### Scenario: User-created agent files preserved

- **GIVEN** `.claude/agents/my-custom-agent.md` exists (name does not match any skill)
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** `.claude/agents/my-custom-agent.md` is NOT removed

#### Scenario: No agents directory exists

- **GIVEN** `.claude/agents/` does not exist
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** no error occurs and no cleanup message is displayed

## Sync Workspace: Idempotency

### Requirement: Copy-with-Template Idempotency

The copy-with-template approach SHALL produce identical output files on repeated runs. The `sync_agent_skills` function's existing content comparison (`cmp -s`) MUST work correctly with templated copies — comparing the destination file (which has `model:`) against the expected output (not the source file which has `model_tier:`).

#### Scenario: Repeated sync counts files as "already valid"

- **GIVEN** `2-sync-workspace.sh` has already run once successfully
- **AND** no source files have changed
- **WHEN** `2-sync-workspace.sh` runs again
- **THEN** all skills are counted as "already valid"
- **AND** "repaired" count is 0

## Memory Updates

### Requirement: Model Tiers Documentation Update

`docs/memory/fab-workflow/model-tiers.md` SHALL be updated during hydrate:

- The "Dual Deployment for Fast-Tier" design decision SHALL be marked as **Superseded** — skills now support `model:` natively, so copy-with-template replaces the dual strategy
- The "Deployment: Dual Strategy" requirements section SHALL be replaced with documentation of the single-deployment approach (copy-with-template for Claude Code, copies for Codex, symlinks for OpenCode)
- References to `.claude/agents/` generation SHALL be removed

### Requirement: Distribution Documentation Update

`docs/memory/fab-workflow/distribution.md` SHALL be updated during hydrate:

- References to "symlinks" for Claude Code skills in `fab-sync.sh` description SHALL be updated to "copies" (or "copies with model templating")
- The "Symlink Repair After Update" section title and content SHALL reflect that Claude Code skills are now copies, not symlinks

## Deprecated Requirements

### Dual Deployment for Fast-Tier

**Reason**: Claude Code skills now support `model:` in frontmatter natively. The premise that only agents support `model:` is no longer true. Copy-with-template provides model selection within skills, eliminating the need for separate agent files.

**Migration**: Fast-tier skills are deployed via copy-with-template (single deployment). Existing `.claude/agents/` files are cleaned up by transitional cleanup logic in `2-sync-workspace.sh`.

## Design Decisions

### Copy-with-Template over Enhanced Function Signature

**Decision**: Model tier templating is applied to Claude Code copies, replacing `model_tier:` with `model:` during deployment.

*Why*: Same sed substitution that Section 4 already performs, but applied at copy time instead of in a separate agent file. Minimal code change, proven approach.

*Rejected*: Adding a new `sync_agent_skills` mode (`"copy-template"`) — would complicate the function signature for a single caller's needs. The templating can be handled alongside or after the existing copy logic.

### Transitional Cleanup over Immediate Removal

**Decision**: Keep a one-release transitional cleanup step that removes `.claude/agents/` files matching known skill names, rather than removing all cleanup logic immediately.

*Why*: Downstream projects that haven't yet upgraded will still have stale `.claude/agents/fab-*.md` files. The cleanup runs on every `fab-sync` (triggered by `direnv allow`), so projects get cleaned up on next directory entry. Cost is ~10 lines of bash that can be removed in a future release.

*Rejected*: Immediate removal of all cleanup — leaves stale agent files in downstream projects indefinitely. Manual cleanup — unnecessary burden on users.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Claude Code skills support `model:` frontmatter | Confirmed from intake #1 — Claude Code documentation confirms `model` is a supported optional field in SKILL.md frontmatter | S:95 R:90 A:95 D:95 |
| 2 | Certain | Copy mode already exists in `sync_agent_skills` | Confirmed from intake #2 — verified in source (lines 292-300 of `2-sync-workspace.sh`) | S:95 R:95 A:95 D:95 |
| 3 | Certain | `model_tier:` → `model:` sed substitution is sufficient | Upgraded from intake #3 Confident — code review confirms Section 4 does exactly this substitution, proven pattern | S:90 R:85 A:90 D:85 |
| 4 | Confident | Capable skills need no `model:` line | Confirmed from intake #4 — omission = platform default, intended behavior for capable tier | S:80 R:90 A:85 D:85 |
| 5 | Confident | Transitional cleanup preferable to immediate removal | Override from intake #5 Tentative — resolved via design decision: keep one-release cleanup, low cost, clean migration | S:70 R:75 A:65 D:60 |
| 6 | Certain | OpenCode and Codex deployment modes unchanged | Intake explicitly scopes changes to Claude Code only; other platforms unaffected | S:95 R:95 A:95 D:95 |
| 7 | Confident | Canonical skill files retain `model_tier:` | Provider-agnostic field stays in source; only deployed copies get provider-specific `model:` — preserves portability per constitution | S:85 R:90 A:90 D:85 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
