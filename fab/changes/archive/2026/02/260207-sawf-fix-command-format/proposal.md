# Proposal: Fix command format in Next suggestions — fab:xxx to /fab-xxx

**Change**: 260207-sawf-fix-command-format
**Created**: 2026-02-07
**Status**: Draft

## Why

All "Next:" command suggestions across fab skill files, shell scripts, templates, and centralized docs reference commands using the `/fab:xxx` colon format (e.g., `/fab:continue`). However, the actual Claude Code skill names use the `/fab-xxx` hyphen format (e.g., `/fab-continue`). Users following these suggestions get invalid command references.

## What Changes

- Update all `/fab:xxx` references to `/fab-xxx` across:
  - Skill instruction files (`fab/.kit/skills/*.md` and `_context.md`)
  - Shell scripts (`fab/.kit/scripts/*.sh`)
  - Templates (`fab/.kit/templates/*.md`)
  - Centralized docs (`fab/docs/**/*.md`)
- Archived changes (`fab/changes/archive/`) are left untouched as historical artifacts

## Affected Docs

### New Docs
_(none)_

### Modified Docs
- `fab-workflow/context-loading.md`: Command references in Next Steps convention
- `fab-workflow/planning-skills.md`: Command references throughout
- `fab-workflow/execution-skills.md`: Command references throughout
- `fab-workflow/change-lifecycle.md`: Command references throughout
- `fab-workflow/hydrate.md`: Command references
- `fab-workflow/hydrate-generate.md`: Command references
- `fab-workflow/init.md`: Command references
- `fab-workflow/index.md`: Command references
- `fab-workflow/configuration.md`: Command references
- `fab-workflow/templates.md`: Command references
- `fab-workflow/kit-architecture.md`: Command references

### Removed Docs
_(none)_

## Impact

- All skill files in `fab/.kit/skills/` (13 files, ~223 occurrences)
- Shell scripts in `fab/.kit/scripts/` (4 files)
- Templates in `fab/.kit/templates/` (2 files)
- Centralized docs in `fab/docs/` (12 files, ~98 occurrences)
- No code logic changes — purely text/documentation corrections

## Open Questions

_(none — scope is clear: find-and-replace `/fab:` with `/fab-` in all non-archived files)_
