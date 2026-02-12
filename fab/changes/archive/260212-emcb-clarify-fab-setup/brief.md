# Proposal: Clarify fab-setup Responsibilities and Initialize fab/design Folder

**Change**: 260212-ulxn-clarify-fab-setup
**Created**: 2026-02-12
**Status**: Draft

## Origin

**Backlog**: [emcb]
**Linear**: DEV-1008
**User requested**: `/fab-new emcb`

> Clarify fab-setup responsibilities and initialize fab/design folder (merged with DEV-1009)

## Why

There's unclear boundary between `/fab-init` and `fab-setup.sh` responsibilities. Both tools manage overlapping concerns:
- Both create `fab/changes/` directory
- Both manage `fab/docs/index.md`
- Both handle skill symlinks
- Both manage `.gitignore` entries

Additionally, `fab-setup.sh` doesn't initialize `fab/design/` directory despite `/fab-init` expecting `fab/design/index.md` to exist. This causes inconsistency in bootstrap workflows.

## What Changes

1. **Clarify distinct responsibilities** between `/fab-init` skill and `fab-setup.sh` script through updated documentation
2. **Modify `fab-setup.sh`** to initialize `fab/design/` folder with index.md skeleton (matching the pattern used for `fab/docs/`)
3. **Update documentation** in `fab/docs/fab-workflow/` to clearly document the boundary:
   - `fab-setup.sh`: Pure structural setup (directories, symlinks, skeleton files)
   - `/fab-init`: Interactive configuration generation (config.yaml, constitution.md) + delegates to fab-setup.sh
4. **Consider consolidation** if the overlap proves too significant after clarification

## Affected Docs

### Modified Docs
- `fab-workflow/init`: Update to clarify responsibilities vs fab-setup.sh and document the delegation pattern
- `fab-workflow/distribution`: Update bootstrap instructions to reflect that fab-setup.sh now initializes fab/design/ folder

## Impact

**Code Changes**:
- `fab/.kit/scripts/fab-setup.sh`: Add fab/design/ directory creation and index.md skeleton
- `fab/.kit/skills/fab-init.md`: May need updates to document delegation pattern more clearly

**Workflow Impact**:
- Bootstrap workflow for new projects (both curl one-liner and manual copy paths)
- Users following distribution.md instructions will see updated behavior
- Existing projects re-running fab-setup.sh will get fab/design/ created if missing (idempotent)

**Documentation Impact**:
- Two docs in fab/docs/fab-workflow/ require updates

## Open Questions

<!-- No unresolved questions — proceeding with confident assumptions below. -->

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Clarify responsibilities first, defer consolidation decision | Linear issue uses exploratory language ("consider consolidating if..."), suggesting clarification is the primary goal; consolidation can be evaluated after boundaries are clear |
| 2 | Confident | fab-setup.sh creates fab/design/ with index.md skeleton | Follows established pattern from fab/docs/ initialization; maintains symmetry between docs/ and design/ setup |
| 3 | Confident | Keep fab-setup.sh as pure structural setup, /fab-init as interactive configuration | Separation of concerns: setup script is scriptable/automatable, init skill is interactive and project-specific |

3 assumptions made (3 confident, 0 tentative). Run `/fab-clarify` to review if needed.
