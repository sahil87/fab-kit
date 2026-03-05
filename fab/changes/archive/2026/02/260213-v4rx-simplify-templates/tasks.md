# Tasks: Simplify Brief and Spec Templates

**Change**: 260213-v4rx-simplify-templates
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Template Changes

- [x] T001 [P] Flatten Affected Docs in `fab/.kit/templates/brief.md` — replace 3 headed subsections (New Docs / Modified Docs / Removed Docs) with a single flat list using inline `(new)`, `(modify)`, `(remove)` markers
- [x] T002 [P] Simplify Open Questions in `fab/.kit/templates/brief.md` — remove `[BLOCKING]`/`[DEFERRED]` labels, add guidance comment explaining SRAD handles prioritization
- [x] T003 [P] Simplify optional sections in `fab/.kit/templates/spec.md` — remove `## Non-Goals`, `## Design Decisions`, and `## Deprecated Requirements` headings and their placeholder content; replace with a single guidance comment listing all three as optional patterns with their formats

## Phase 2: Shared Context Update

- [x] T004 Update `_context.md` Section 3 step 3 — change parenthetical "(the New, Modified, and Removed entries)" to reference the flat list format with inline markers

## Phase 3: Centralized Doc Update

- [x] T005 Update `fab/docs/fab-workflow/templates.md` — revise `brief.md` section to describe flat Affected Docs list and plain Open Questions; revise `spec.md` section to describe optional sections as patterns rather than standing template sections

---

## Execution Order

- T001, T002, T003 are independent (parallel within Phase 1)
- T004 is independent of Phase 1 but logically follows
- T005 depends on T001-T004 being complete (documents the new structure)
