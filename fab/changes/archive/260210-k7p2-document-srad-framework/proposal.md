# Proposal: Document the SRAD Framework

**Change**: 260210-k7p2-document-srad-framework
**Created**: 2026-02-10
**Status**: Draft

## Why

SRAD is the autonomy framework that governs how every planning skill decides whether to ask the user or assume. It gates `/fab-fff` via the confidence score, and every proposal, spec, and plan artifact uses SRAD grades and markers. But the framework itself is only defined in `fab/.kit/skills/_context.md` — an internal skill context file that no human reader would naturally find. The glossary has terse one-line definitions, and the centralized docs reference SRAD in passing without explaining it. Anyone who didn't build the system has no way to understand what the acronym stands for, how ambiguity scores work, or why `/fab-fff` refuses to run. This change creates a dedicated spec that makes the framework self-documenting.

## What Changes

- **New spec**: `fab/specs/srad.md` — standalone specification covering:
  - Acronym expansion (Signal strength, Reversibility, Agent competence, Disambiguation type)
  - The four scoring dimensions with evaluation criteria (high vs low for each)
  - The four confidence grades (Certain, Confident, Tentative, Unresolved) with their artifact markers and visibility rules
  - The confidence scoring formula and its 0.0–5.0 range
  - The `/fab-fff` gate threshold (>= 3.0) and what it allows/blocks
  - The confidence lifecycle (which skills compute/consume the score)
  - Worked examples: one high-ambiguity proposal (score near 0) vs one low-ambiguity proposal (score near 5)
  - The Critical Rule (Unresolved + low R + low A must always be asked)
- **Updated index**: `fab/specs/index.md` — add `srad.md` to the specs table
<!-- assumed: Standalone spec rather than a section in skills.md — SRAD is cross-cutting across all planning skills and has enough substance for its own document -->

## Affected Docs

### New Docs
- (none — this is a specs-only change; centralized docs already reference SRAD in context)

### Modified Docs
- (none)

### Removed Docs
- (none)

## Impact

- **Specs**: New file `fab/specs/srad.md`, updated `fab/specs/index.md`
- **Skills**: No changes — `_context.md` remains the canonical runtime reference; the new spec documents the design intent
- **Docs**: No changes — existing SRAD references in `fab/docs/fab-workflow/` are sufficient for post-implementation documentation
- **Config/Scripts**: No changes

## Open Questions

(none — all decisions are Certain or Confident)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Standalone `fab/specs/srad.md` rather than a section in `skills.md` | SRAD is cross-cutting across all planning skills (fab-new, fab-continue, fab-ff, fab-fff, fab-clarify); it has enough distinct substance (acronym, dimensions, scoring formula, gating, examples) to warrant its own file rather than being buried in per-skill behavior |

1 assumption made (1 confident, 0 tentative). Run /fab-clarify to review.
