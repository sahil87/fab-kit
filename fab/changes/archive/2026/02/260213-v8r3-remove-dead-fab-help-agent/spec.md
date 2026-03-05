# Spec: Remove Dead fab-help Agent File

**Change**: 260213-v8r3-remove-dead-fab-help-agent
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/kit-architecture.md`

## Non-Goals

- Auditing other agent files for dead-code status — that is a separate change
- Modifying `fab-setup.sh` agent generation logic — the script can tolerate a missing source; it only generates agents for skills with `model_tier: fast` frontmatter

## Kit Architecture: Agent File Cleanup

### Requirement: Remove Unused fab-help Agent File

The file `.claude/agents/fab-help.md` SHALL be deleted. This agent file is never spawned by any skill or pipeline (`subagent_type.*fab-help` produces zero matches across the codebase). The `/fab-help` skill continues to function via its skill symlink path (`.claude/skills/fab-help/SKILL.md` → `fab/.kit/skills/fab-help.md`).

#### Scenario: Agent File Deleted

- **GIVEN** the file `.claude/agents/fab-help.md` exists in the repository
- **WHEN** this change is applied
- **THEN** `.claude/agents/fab-help.md` SHALL be deleted
- **AND** no other files in `.claude/agents/` SHALL be modified

#### Scenario: fab-help Skill Still Works

- **GIVEN** `.claude/agents/fab-help.md` has been deleted
- **WHEN** a user invokes `/fab-help`
- **THEN** the skill executes via `.claude/skills/fab-help/SKILL.md` (symlink to `fab/.kit/skills/fab-help.md`)
- **AND** behavior is identical to before the deletion

### Requirement: Update kit-architecture.md Directory Listing

The `fab/docs/fab-workflow/kit-architecture.md` document SHALL be updated to remove the `.claude/agents/fab-help.md` entry from the "Model Tier Agent Files" code block listing (currently line 104).

#### Scenario: Doc Listing Reflects Actual Agent Files

- **GIVEN** the kit-architecture.md contains a code block listing agent files under "Model Tier Agent Files (Dual Deployment)"
- **WHEN** this change is applied
- **THEN** the line `.claude/agents/fab-help.md    # Generated with model: haiku` SHALL be removed from the code block
- **AND** the remaining agent file entries (`fab-init.md`, `fab-status.md`, `fab-switch.md`) SHALL be preserved unchanged

## Deprecated Requirements

### fab-help Agent File Generation

**Reason**: The `fab-help.md` agent file was created by the model-tiers change (260212-k8m3) for pipeline invocation via the Task tool, but no skill or agent ever spawns it. The skill + script pair covers all actual usage.
**Migration**: N/A — the skill path (`.claude/skills/fab-help/`) is the sole invocation mechanism and remains unchanged.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Only fab-help agent needs removal | Brief explicitly scopes to fab-help; other agents may have similar status but that's a separate change |
| 2 | Confident | kit-architecture.md is the only doc needing update | Grep for `agents/fab-help` in fab/docs/ found only one match |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
