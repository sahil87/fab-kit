# Brief: Docs Skills Housekeeping

**Change**: 260214-r8kv-docs-skills-housekeeping
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Housekeeping tasks:
> 1. No need of fab-status.sh — can remove it
> 2. Rename fab-hydrate-specs to docs-hydrate-specs
> 3. Rename fab-hydrate to docs-hydrate-memory
> 4. Rename fab-reorg-specs to docs-reorg-specs
> 5. Create a docs-reorg-memory

## Why

The documentation-related skills (`fab-hydrate-specs`, `fab-hydrate`, `fab-reorg-specs`) are currently namespaced under `fab-*` alongside workflow skills. Renaming them to `docs-*` creates a clear namespace boundary: `fab-*` = change workflow, `docs-*` = documentation management. Additionally, `fab-status.sh` is redundant now that `fab-preflight.sh` and the `/fab-status` skill handle status queries, and a `docs-reorg-memory` skill is missing as the counterpart to `docs-reorg-specs`.

## What Changes

- **Remove** `fab/.kit/scripts/fab-status.sh` — redundant with preflight + skill-based status
- **Rename** `fab/.kit/skills/fab-hydrate-specs.md` → `docs-hydrate-specs.md`
- **Rename** `fab/.kit/skills/fab-hydrate.md` → `docs-hydrate-memory.md`
- **Rename** `fab/.kit/skills/fab-reorg-specs.md` → `docs-reorg-specs.md`
- **Create** `fab/.kit/skills/docs-reorg-memory.md` — analyze memory files for organizational improvements, mirroring `docs-reorg-specs` for the memory domain
- **Update** all cross-references in skills, scripts, config, templates, and memory files that mention the old names

## Affected Memory

- `fab-workflow/hydrate`: (modify) references to skill names change
- `fab-workflow/kit-architecture`: (modify) script inventory and skill namespace documentation
- `fab-workflow/execution-skills`: (modify) skill names and references

## Impact

- Skill files in `fab/.kit/skills/` — renamed files + internal cross-references
- `fab/.kit/skills/_context.md` — may reference old skill names
- Memory files referencing old skill names
- `fab/specs/skills.md` — skill catalog references
- No external API or config.yaml changes needed

## Open Questions

- None — all 5 tasks are specific and mechanical.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | `docs-reorg-memory` mirrors `docs-reorg-specs` behavior for the memory domain | Pattern is obvious from naming symmetry; user explicitly requested it alongside the other renames |

1 assumption made (1 confident, 0 tentative). Run /fab-clarify to review.
