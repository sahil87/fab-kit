# Brief: Merge fab-init Subcommands into Single Skill

**Change**: 260213-3tyk-merge-fab-init-subcommands
**Created**: 2026-02-13
**Status**: Draft

## Origin

> merge the multiple fab-init commands into the main fab-init command - the variants (sub tasks of fab-init) can be invoked using fab-init <variantname> eg fab-init constitution.

## Why

The fab-init family currently consists of 4 separate slash commands (`/fab-init`, `/fab-init-config`, `/fab-init-constitution`, `/fab-init-validate`), each with its own skill file, symlink, and agent file. This fragments a cohesive feature across multiple entry points. Consolidating into a single `/fab-init [subcommand]` provides a cleaner UX — users discover all init capabilities under one command, and the skill list stays concise.

## What Changes

- **Merge 3 variant skill files into `fab-init.md`**: The content of `fab-init-config.md`, `fab-init-constitution.md`, and `fab-init-validate.md` is consolidated into a single `fab-init.md` with argument-based routing
- **Add subcommand routing**: `/fab-init` with no args → full bootstrap (unchanged); `/fab-init config [section]` → config create/update; `/fab-init constitution` → constitution create/amend; `/fab-init validate` → structural validation
- **Remove 3 variant skill files**: Delete `fab/.kit/skills/fab-init-config.md`, `fab-init-constitution.md`, `fab-init-validate.md`
- **Remove variant symlinks and agent files**: The setup script dynamically discovers skills, so removing the source files automatically eliminates symlinks and agents on next `fab-setup.sh` run. Existing stale symlinks/agents need manual cleanup.
- **Update cross-references**: ~10 files in `fab/docs/` and `fab/design/` reference `/fab-init-config`, `/fab-init-constitution`, or `/fab-init-validate` — update to new `/fab-init config`, `/fab-init constitution`, `/fab-init validate` syntax
<!-- assumed: Clean break, no backward-compatible aliases for old command names — this is an internal toolkit and the variant commands are recent additions -->

## Affected Docs

### New Docs
(none)

### Modified Docs
- `fab-workflow/init.md`: Update command references from `/fab-init-config` → `/fab-init config`, etc.
- `fab-workflow/init-family.md`: Update all variant command references and invocation patterns
- `fab-workflow/configuration.md`: Update `/fab-init-config` references
- `fab-workflow/constitution-governance.md`: Update `/fab-init-constitution` references
- `fab-workflow/config-management.md`: Update `/fab-init-config` references
- `fab-workflow/index.md`: Update command names in the domain index

### Removed Docs
(none)

## Impact

- **Skill files**: `fab/.kit/skills/fab-init.md` (major rewrite), 3 files deleted
- **Symlinks**: `.claude/skills/fab-init-config/`, `.claude/skills/fab-init-constitution/`, `.claude/skills/fab-init-validate/` — removed
- **Agent files**: `.claude/agents/fab-init-config.md`, `.claude/agents/fab-init-constitution.md`, `.claude/agents/fab-init-validate.md` — removed (setup script generates these; stale copies need cleanup)
- **Multi-agent symlinks**: `.opencode/commands/` and `.agents/skills/` also get variant symlinks from `fab-setup.sh` — stale copies need cleanup
- **Cross-references in docs**: 6 doc files + `fab/design/user-flow.md` reference variant command names
- **`_context.md`**: References to variant commands in the Next Steps convention table
- **`fab-setup.sh`**: No changes needed — it dynamically discovers `fab/.kit/skills/fab-*.md`, so removing variant files is sufficient. However, it won't clean up stale symlinks from removed skills — may want to add a cleanup step.

## Open Questions

(none — all decisions resolved via SRAD)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Single merged skill file rather than router-with-includes | Claude Code skills are single markdown files; the user said "merge into the main fab-init command" |
| 2 | Tentative | Clean break — no backward-compatible aliases for old `/fab-init-config` etc. | Internal toolkit, variant commands are recent additions (created 260212), and user explicitly wants consolidation |

2 assumptions made (1 confident, 1 tentative). Run /fab-clarify to review.
