# Brief: Simplify Brief and Spec Templates

**Change**: 260213-v4rx-simplify-templates
**Created**: 2026-02-13
**Status**: Draft

## Origin

> Simplify brief and spec templates: (1) Flatten Affected Docs into a single list with inline markers instead of 3 subsections, (2) Remove BLOCKING/DEFERRED from brief Open Questions — let SRAD handle prioritization at spec time, (3) Remove placeholder content from optional spec sections (Non-Goals, Design Decisions, Deprecated Requirements) so they're only added when needed rather than nudging fill-in-the-blank, (4) Drop Deprecated Requirements as a standing template section — document as a pattern to use when needed instead.

## Why

The current templates include structural overhead that doesn't pay for itself in the common case. The brief's Affected Docs section has 3 headed subsections (New/Modified/Removed) when most changes only touch 1-2 docs, leaving empty headings. The Open Questions section duplicates SRAD's prioritization work. The spec template includes placeholder content for optional sections (Non-Goals, Design Decisions, Deprecated Requirements) that nudges the agent to fill them in even when they should be omitted entirely.

## What Changes

- **brief.md template**: Flatten Affected Docs from 3 subsections into a single list with inline `(new)`, `(modify)`, `(remove)` markers
- **brief.md template**: Simplify Open Questions to a plain list — remove BLOCKING/DEFERRED labels (SRAD handles prioritization at spec generation time)
- **spec.md template**: Remove placeholder content from optional sections; replace with a single comment listing optional sections to add when needed
- **spec.md template**: Drop Deprecated Requirements as a standing section — document it as a pattern in the template comment instead

## Affected Docs

- `fab-workflow/templates`: (modify) template structure rules changing for brief.md and spec.md

## Impact

- `fab/.kit/templates/brief.md` — structural change to Affected Docs and Open Questions sections
- `fab/.kit/templates/spec.md` — removal of optional section placeholders, removal of Deprecated Requirements section
- All downstream skills that consume these templates (`/fab-new`, `/fab-continue`) are unaffected — they fill templates dynamically, not by pattern-matching section headings
- `_context.md` reference to "New, Modified, and Removed entries" in Centralized Doc Lookup (Section 3) may need a wording update to match the new flat list format

## Open Questions

- Should `_context.md` Section 3 (Centralized Doc Lookup) wording be updated to reference the new flat marker format, or is the current wording generic enough?
