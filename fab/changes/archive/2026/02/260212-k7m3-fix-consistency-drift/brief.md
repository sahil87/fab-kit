# Brief: Fix consistency drift between design, docs, and implementation

**Change**: 260212-k7m3-fix-consistency-drift
**Created**: 2026-02-12
**Status**: Draft

## Origin

> Fix consistency drift between design specs, docs, and implementation — address the 11 critical findings from the internal consistency check: rename "Proposal" to "Brief" in templates, align DEFERRED timing, add missing Origin section to design template spec, fix confidence score default, update slug word count in design, move branch integration docs from fab-new to fab-switch, update .status.yaml schema in design, add archive index maintenance to design, add created_by field to design, fix stale "specs" terminology in backfill index, and replace "proposal" references in _generation.md

## Why

The internal consistency check (`/internal-consistency-check`) revealed 11 critical inconsistencies across the three sources of truth: design specs (`fab/design/`), centralized docs (`fab/docs/`), and implementation (`fab/.kit/`). These inconsistencies cause agents to receive contradictory instructions depending on which layer they consult — leading to wrong artifact headers, incorrect timing assumptions, and stale schema references. This is a maintenance pass to bring all three layers back into alignment.

## What Changes

- **Fix "Proposal" → "Brief"**: Update `fab/.kit/templates/brief.md` header and `fab/.kit/skills/_generation.md` references to use "Brief" consistently
- **Align DEFERRED timing**: Reconcile "during spec" (design) vs "before tasks" (implementation) for `[DEFERRED]` question resolution window
- **Add Origin section to design**: Document the `## Origin` section in `fab/design/templates.md` brief template spec
- **Fix confidence score default**: Clarify the `score: 5.0` default in `fab/.kit/templates/status.yaml` vs the "all zeros" claim in docs
- **Update slug word count**: Change "2-4 words" → "2-6 words" in `fab/design/glossary.md` and `fab/design/architecture.md`
- **Move branch integration in design**: Update `fab/design/architecture.md` to reflect branch integration in `/fab-switch` (not `/fab-new`)
- **Update .status.yaml schema in design**: Replace `stage:` field with `active` marker structure in `fab/design/architecture.md`
- **Add archive index maintenance**: Document the archive index step in `fab/design/templates.md`
- **Add created_by field to design**: Include `created_by` in `.status.yaml` template documentation in `fab/design/templates.md`
- **Fix backfill index terminology**: Change "docs and specs" → "docs and design" in `fab/docs/fab-workflow/index.md`

## Affected Docs

### New Docs
(none)

### Modified Docs
- `fab-workflow/index.md`: Fix stale "specs" terminology in backfill entry
- `fab-workflow/planning-skills.md`: Clarify "all zeros" language re: confidence score default

### Removed Docs
(none)

## Impact

- **`fab/.kit/templates/brief.md`** — header text change
- **`fab/.kit/templates/status.yaml`** — confidence score documentation clarification
- **`fab/.kit/skills/_generation.md`** — terminology update (proposal → brief)
- **`fab/design/templates.md`** — Origin section, archive index, created_by field additions
- **`fab/design/glossary.md`** — slug word count update
- **`fab/design/architecture.md`** — branch integration, .status.yaml schema, slug word count updates
- **`fab/docs/fab-workflow/index.md`** — backfill entry terminology fix
- **`fab/docs/fab-workflow/planning-skills.md`** — confidence score default clarification

All changes are documentation/template corrections. No behavioral logic changes — this aligns what the docs say with what the code does (or vice versa, depending on which is authoritative).

## Open Questions

(none — all 11 findings have clear, unambiguous fixes identified by the consistency check)
