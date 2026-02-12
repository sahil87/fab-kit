# Brief: Expand fab-init Command Family

**Change**: 260212-h9k3-fab-init-family
**Created**: 2026-02-12
**Status**: Draft

## Origin

User requested:
> "Expand fab-init family with constitution, config, and validate commands. Clarify fab-update architecture (script vs agent). Reference backlog: 90g5 (constitution), akhp (hydrate-design)."

Context from discussion:
- Backlog item [90g5] (DEV-988): Add constitution command
- Backlog item [akhp]: Rename fab-backfill to fab-hydrate-design (separate change)
- Discussion identified that fab-init creates several artifacts that may need updating after initial creation
- fab-update.sh script exists (DEV-990) but unclear if agent command is also needed

## Why

fab-init currently creates config.yaml and constitution.md once, but offers no mechanism to update them as projects evolve. Tech stacks change, new stages are added, and constitutional principles need amendments. Users must manually edit these files without validation or guidance, risking structural errors.

This change introduces focused commands for managing initialization artifacts lifecycle, following the pattern of specialized tools for specific concerns.

## What Changes

Add three new fab-init family commands:

1. **fab-init-constitution** - Constitutional management
   - Create initial constitution (same as current fab-init behavior)
   - Update existing constitution with versioning (MAJOR.MINOR.PATCH)
   - Amend principles with governance tracking
   - Follow patterns from `references/speckit/constitution.md`

2. **fab-init-config** - Interactive config.yaml updates
   - Guided updates for key sections: context, stages, rules, source_paths, checklist categories
   - Validate YAML structure after edits
   - Preserve comments and formatting

3. **fab-init-validate** - Structural validation
   - Check config.yaml against schema (required fields, valid stage dependencies, etc.)
   - Check constitution.md structure (principles, governance section, versioning)
   - Report issues with actionable fixes
   - Can be used before commits or after manual edits

Additionally: Document the relationship between fab-update.sh (script) and potential fab-update agent command.

## Affected Docs

### New Docs
- `fab-workflow/init-family`: Documents the fab-init-* command family (constitution, config, validate)
- `fab-workflow/config-management`: Guide for maintaining config.yaml over project lifecycle
- `fab-workflow/constitution-governance`: Constitutional amendment workflow and versioning

### Modified Docs
- `fab-workflow/init`: Add references to the new fab-init-* commands for post-initialization management
- `fab-workflow/configuration`: Expand with validation patterns and update workflows

## Impact

**New Skills**:
- `fab/.kit/skills/fab-init-constitution.md`
- `fab/.kit/skills/fab-init-config.md`
- `fab/.kit/skills/fab-init-validate.md`

**Modified Skills**:
- `fab/.kit/skills/fab-init.md` - Reference the new family members for post-init updates

**Scripts**:
- May need validation helper script for fab-init-validate to share with fab-init

**Backlog Items**:
- Resolves [90g5] (DEV-988)
- References but doesn't implement [akhp] (fab-hydrate-design - separate change)

## Open Questions

- [BLOCKING] **fab-update architecture**: Should we have both `fab-update.sh` (script) and `/fab-update` (agent command), or just the script? What are the distinct use cases for each?
  - Script use case: Automated kit updates in CI, quick repairs of symlinks
  - Agent command use case: Interactive migration with validation, config schema updates, explaining what changed
  - If both: How do they divide responsibilities?
  - If one: Which one, and why?
