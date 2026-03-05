# Brief: Rename design/ → specs/ and docs/ → memory/

**Change**: 260213-1u9c-rename-specs-memory
**Created**: 2026-02-13
**Status**: Draft

## Origin

> Rename fab/design to fab/specs and fab/docs to fab/memory. Update all cross-references across the codebase.

## Why

The current folder names (`design/` and `docs/`) are ambiguous — both sound like generic documentation. The rename makes the distinction between human-authored pre-implementation artifacts (`specs/`) and AI-generated post-implementation knowledge (`memory/`) immediately obvious from the folder name alone.

## What Changes

- Rename `fab/design/` → `fab/specs/`
- Rename `fab/docs/` → `fab/memory/`
- Update all cross-references in active source files: skills, scripts, config, constitution, templates, scaffold, README
- Update internal cross-references within the renamed folders themselves
- Leave archived change artifacts (`fab/changes/archive/`) untouched — they capture historical state

## Affected Docs

- `fab-workflow/*`: (modify) All hydrated docs reference `fab/docs/` and `fab/design/` paths

## Impact

- Every skill in `fab/.kit/skills/` references these paths
- Shell scripts in `fab/.kit/scripts/` reference these paths
- `fab/config.yaml` and `fab/constitution.md` reference these paths
- `_context.md` (shared context preamble) references these paths
- Templates in `fab/.kit/templates/` reference these paths
- 183 files total contain references (majority are archived changes — left as-is)

## Open Questions

- None — scope is clear and fully mechanical.
