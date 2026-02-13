# Spec: Merge fab-init Subcommands into Single Skill

**Change**: 260213-3tyk-merge-fab-init-subcommands
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/init.md`, `fab/docs/fab-workflow/init-family.md`, `fab/docs/fab-workflow/configuration.md`, `fab/docs/fab-workflow/constitution-governance.md`, `fab/docs/fab-workflow/config-management.md`, `fab/docs/fab-workflow/index.md`

## Non-Goals

- Changing the behavioral logic of config creation/update, constitution create/amend, or validate — only the entry point changes
- Adding new subcommands beyond the existing three (config, constitution, validate)
- Modifying `fab-setup.sh` discovery logic — it already globs `fab-*.md`, so removing files is sufficient
- Updating `fab/design/` specs — these are human-curated pre-implementation artifacts and out of scope for this change

## Skill File: Subcommand Routing

### Requirement: Argument-Based Routing

`/fab-init` SHALL route to the appropriate behavior based on the first positional argument:

| Invocation | Behavior |
|------------|----------|
| `/fab-init` (no args) | Full structural bootstrap (existing behavior, unchanged) |
| `/fab-init config [section]` | Config create/update (currently `/fab-init-config [section]`) |
| `/fab-init constitution` | Constitution create/amend (currently `/fab-init-constitution`) |
| `/fab-init validate` | Structural validation (currently `/fab-init-validate`) |

The skill MUST recognize `config`, `constitution`, and `validate` as subcommand keywords. Any other argument that is not a recognized subcommand SHALL trigger the existing redirect message: "Did you mean /fab-hydrate? /fab-init no longer accepts source arguments."

#### Scenario: No arguments — full bootstrap
- **GIVEN** `/fab-init` is invoked with no arguments
- **WHEN** the skill processes the invocation
- **THEN** the full structural bootstrap behavior executes (Phase 1: steps 1a–1g)

#### Scenario: Config subcommand with section argument
- **GIVEN** `/fab-init config context` is invoked
- **WHEN** the skill parses the arguments
- **THEN** the config update behavior executes with `context` as the section argument

#### Scenario: Config subcommand without section argument
- **GIVEN** `/fab-init config` is invoked
- **WHEN** the skill parses the arguments
- **THEN** the config behavior executes in menu mode (no section pre-selected)

#### Scenario: Constitution subcommand
- **GIVEN** `/fab-init constitution` is invoked
- **WHEN** the skill parses the arguments
- **THEN** the constitution create/amend behavior executes

#### Scenario: Validate subcommand
- **GIVEN** `/fab-init validate` is invoked
- **WHEN** the skill parses the arguments
- **THEN** the structural validation behavior executes

#### Scenario: Unrecognized argument
- **GIVEN** `/fab-init https://example.com` is invoked (not a recognized subcommand)
- **WHEN** the skill parses the arguments
- **THEN** the skill outputs: "Did you mean /fab-hydrate? /fab-init no longer accepts source arguments."
- **AND** no bootstrap or subcommand behavior executes

### Requirement: Merged Skill File Structure

The consolidated `fab/.kit/skills/fab-init.md` SHALL contain all behavior currently spread across 4 files. The file MUST use a clear section structure:

1. **Frontmatter** — single `name: fab-init` entry with updated description
2. **Purpose & Arguments** — documents the subcommand routing
3. **Pre-flight Check** — unified (kit existence check, argument classification)
4. **Bootstrap Behavior** — current `/fab-init` Phase 1 (steps 1a–1g), unchanged
5. **Config Behavior** — full content of current `/fab-init-config` (create mode, update mode, edit section flow)
6. **Constitution Behavior** — full content of current `/fab-init-constitution` (create mode, update mode, amendment flow)
7. **Validate Behavior** — full content of current `/fab-init-validate` (all 14 checks)

Each behavior section SHALL preserve the complete logic from its source file. No behavioral changes — this is a structural reorganization only.

#### Scenario: Bootstrap delegates to internal config section
- **GIVEN** `/fab-init` runs the full bootstrap and `fab/config.yaml` does not exist
- **WHEN** step 1a executes
- **THEN** the config create mode behavior from the Config Behavior section executes (previously delegated to `/fab-init-config`)
- **AND** the behavior is identical to the current `/fab-init-config` create mode

#### Scenario: Bootstrap delegates to internal constitution section
- **GIVEN** `/fab-init` runs the full bootstrap and `fab/constitution.md` does not exist
- **WHEN** step 1b executes
- **THEN** the constitution create mode behavior from the Constitution Behavior section executes (previously delegated to `/fab-init-constitution`)
- **AND** the behavior is identical to the current `/fab-init-constitution` create mode

### Requirement: Frontmatter Update

The merged skill MUST have a single frontmatter block:

```yaml
---
name: fab-init
description: "Bootstrap fab/ directory structure, or manage config/constitution/validation. Safe to re-run."
model_tier: fast
---
```

The `description` SHALL reflect the expanded scope while remaining concise.

#### Scenario: Skill discovery
- **GIVEN** `fab-setup.sh` globs `fab/.kit/skills/fab-*.md`
- **WHEN** it discovers `fab-init.md`
- **THEN** one symlink is created: `.claude/skills/fab-init/SKILL.md`
- **AND** no symlinks are created for the removed variant files

## Skill File: Removal of Variant Files

### Requirement: Delete Variant Skill Files

The following files SHALL be deleted:
- `fab/.kit/skills/fab-init-config.md`
- `fab/.kit/skills/fab-init-constitution.md`
- `fab/.kit/skills/fab-init-validate.md`

#### Scenario: Variant files removed
- **GIVEN** the change is applied
- **WHEN** the three variant skill files are deleted
- **THEN** `fab/.kit/skills/` contains `fab-init.md` but not `fab-init-config.md`, `fab-init-constitution.md`, or `fab-init-validate.md`

### Requirement: Stale Symlink and Agent Cleanup

After variant skill files are deleted, the following stale artifacts SHALL be manually removed:

**Symlink directories** (created by `fab-setup.sh`):
- `.claude/skills/fab-init-config/`
- `.claude/skills/fab-init-constitution/`
- `.claude/skills/fab-init-validate/`

**Agent files** (generated by `fab-setup.sh`):
- `.claude/agents/fab-init-config.md`
- `.claude/agents/fab-init-constitution.md`
- `.claude/agents/fab-init-validate.md`

**Multi-agent symlinks** (if present):
- `.opencode/commands/fab-init-config*`
- `.opencode/commands/fab-init-constitution*`
- `.opencode/commands/fab-init-validate*`
- `.agents/skills/fab-init-config*`
- `.agents/skills/fab-init-constitution*`
- `.agents/skills/fab-init-validate*`

<!-- clarified: Manual stale artifact cleanup confirmed — keeps scope focused on the merge; automated cleanup can be a follow-up -->

#### Scenario: Stale symlinks removed
- **GIVEN** the variant skill files have been deleted
- **WHEN** the stale artifact cleanup executes
- **THEN** no symlinks, agent files, or command files referencing `fab-init-config`, `fab-init-constitution`, or `fab-init-validate` remain

#### Scenario: fab-setup.sh re-run after cleanup
- **GIVEN** variant skill files are deleted and stale artifacts are cleaned up
- **WHEN** `fab-setup.sh` is re-run
- **THEN** only `fab-init` symlink/agent is created (no variant entries)
- **AND** no errors are produced

## Cross-References: Documentation Updates

### Requirement: Update fab/docs/ References

All references to `/fab-init-config`, `/fab-init-constitution`, and `/fab-init-validate` in `fab/docs/fab-workflow/` SHALL be updated to the new subcommand syntax:

| Old | New |
|-----|-----|
| `/fab-init-config` | `/fab-init config` |
| `/fab-init-config <section>` | `/fab-init config <section>` |
| `/fab-init-constitution` | `/fab-init constitution` |
| `/fab-init-validate` | `/fab-init validate` |

The following 6 doc files contain references (33 total occurrences):
1. `fab/docs/fab-workflow/index.md` (1 occurrence)
2. `fab/docs/fab-workflow/init.md` (6 occurrences)
3. `fab/docs/fab-workflow/init-family.md` (8 occurrences)
4. `fab/docs/fab-workflow/configuration.md` (3 occurrences)
5. `fab/docs/fab-workflow/constitution-governance.md` (4 occurrences)
6. `fab/docs/fab-workflow/config-management.md` (11 occurrences)

#### Scenario: Doc reference updated
- **GIVEN** `fab/docs/fab-workflow/config-management.md` contains `/fab-init-config context`
- **WHEN** the cross-reference update is applied
- **THEN** the reference reads `/fab-init config context`
- **AND** surrounding text and formatting are preserved

#### Scenario: init-family.md overview updated
- **GIVEN** `fab/docs/fab-workflow/init-family.md` describes separate commands
- **WHEN** the cross-reference update is applied
- **THEN** command names in the overview table and description text use subcommand syntax
- **AND** the document accurately describes the consolidated command structure

### Requirement: Update _context.md References

If `fab/.kit/skills/_context.md` contains references to variant command names, they SHALL be updated to the new subcommand syntax.

#### Scenario: _context.md has no variant references
- **GIVEN** `_context.md` is searched for `/fab-init-config`, `/fab-init-constitution`, `/fab-init-validate`
- **WHEN** no matches are found (confirmed: 0 occurrences)
- **THEN** no changes needed for `_context.md`

## Design Decisions

1. **Single merged file rather than router-with-includes**: All subcommand behavior lives in one `fab-init.md` file.
   - *Why*: Claude Code skills are single markdown files with no include mechanism. The user explicitly asked to "merge into the main fab-init command."
   - *Rejected*: A routing file that references other files — not supported by the skill format.

2. **Clean break, no backward-compatible aliases**: Old `/fab-init-config`, `/fab-init-constitution`, `/fab-init-validate` commands cease to exist immediately.
   <!-- clarified: Clean break confirmed — internal toolkit, variant commands are 1 day old, no external consumers -->
   - *Why*: Internal toolkit with no external consumers. The variant commands were created one day ago (260212). Adding aliases adds complexity for zero benefit.
   - *Rejected*: Redirect aliases that catch old command names — unnecessary engineering for an internal tool.

3. **Manual stale artifact cleanup rather than automated cleanup in fab-setup.sh**: Stale symlinks/agents from removed skills are cleaned up as explicit tasks in this change.
   <!-- clarified: Manual cleanup confirmed — keeps scope focused; automated cleanup in fab-setup.sh can be a separate change -->
   - *Why*: Keeps this change focused on the merge. Adding automated cleanup logic to `fab-setup.sh` (detecting orphaned symlinks) is a separate concern.
   - *Rejected*: Adding orphan detection to `fab-setup.sh` — scope creep, can be a follow-up change.

## Deprecated Requirements

### Separate Skill Files for Init Variants
**Reason**: Consolidated into subcommands of `/fab-init`. Three separate skill files (`fab-init-config.md`, `fab-init-constitution.md`, `fab-init-validate.md`) are replaced by sections within `fab-init.md`.
**Migration**: Use `/fab-init config`, `/fab-init constitution`, `/fab-init validate` respectively.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Single merged skill file rather than router-with-includes | Claude Code skills are single markdown files; user said "merge into the main fab-init command" |
| 2 | Certain | Clean break — no backward-compatible aliases for old command names | Confirmed: internal toolkit, variant commands created 260212 (1 day ago), no external consumers |
| 3 | Certain | Manual stale artifact cleanup rather than adding automated orphan detection to fab-setup.sh | Confirmed: keeps scope focused on the merge; automated cleanup can be a follow-up |

3 assumptions made (1 confident, 2 certain — clarified). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-13

- **Q**: Clean break (no backward-compatible aliases) or redirect aliases for old command names?
  **A**: Clean break — old commands cease to exist immediately
- **Q**: Manual stale artifact cleanup or automated orphan detection in fab-setup.sh?
  **A**: Manual cleanup — keeps scope focused on the merge
