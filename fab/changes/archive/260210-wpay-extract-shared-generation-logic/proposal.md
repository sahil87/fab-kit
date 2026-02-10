# Proposal: Extract shared generation logic from fab-continue and fab-ff

**Change**: 260210-wpay-extract-shared-generation-logic
**Created**: 2026-02-10
**Status**: Draft

## Why

`fab-continue.md` and `fab-ff.md` duplicate spec, plan, tasks, and checklist generation logic nearly verbatim. This means every bug fix or behavior change to artifact generation must be applied in two places, and they inevitably drift. Extracting the shared logic into a `_generation.md` partial that both skills reference eliminates this duplication and makes generation behavior authoritative in one location.

## What Changes

- Create a new `fab/.kit/skills/_generation.md` partial containing the shared artifact generation logic (spec, plan, tasks, checklist generation steps)
- Update `fab-continue.md` to reference `_generation.md` instead of inlining the generation steps
- Update `fab-ff.md` to reference `_generation.md` instead of inlining the generation steps
- Each skill retains its own orchestration logic (stage guards, question handling, auto-clarify, resumability) — only the generation mechanics are shared

## Affected Docs

### New Docs
- None

### Modified Docs
- `fab-workflow/planning-skills`: Update to document the `_generation.md` partial and how fab-continue/fab-ff delegate to it

### Removed Docs
- None

## Impact

- `fab/.kit/skills/fab-continue.md` — generation sections replaced with references to `_generation.md`
- `fab/.kit/skills/fab-ff.md` — generation sections replaced with references to `_generation.md`
- `fab/.kit/skills/_generation.md` — new file containing extracted generation logic
- `fab/specs/skills.md` — may need a note about the partial pattern

## Open Questions

- None — the duplicated sections are clearly identifiable and the extraction boundary is unambiguous.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Use `_generation.md` naming convention (underscore prefix) | Consistent with existing `_context.md` partial pattern already established in the skills directory |

1 assumption made (1 confident, 0 tentative).
