# Tasks: Add Non-Goals and Design Decisions to Spec Template

**Change**: 260211-r4w8-spec-template-sections
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Template Update

- [x] T001 Add `## Non-Goals` and `## Design Decisions` optional sections to `fab/.kit/templates/spec.md` — Non-Goals after metadata/comments and before `## {Domain}: {Topic}`; Design Decisions after domain sections and before `## Deprecated Requirements`; both with HTML comment guidance explaining format, when to include, and when to omit

## Phase 2: Skill Awareness

- [x] T002 Add Non-Goals generation step to `fab/.kit/skills/_generation.md` Spec Generation Procedure — insert a new step between current step 4 and step 5 covering Non-Goals population from brief context, using the same optional/omit pattern as existing step 5b (Design Decisions)

## Phase 3: Documentation

- [x] T003 Update `fab/docs/fab-workflow/templates.md` — add Non-Goals and Design Decisions to the `spec.md` subsection, documenting section placement, format, and optionality; consistent with the template and generation procedure

---

## Execution Order

- T001 first (template is source of truth for all downstream references)
- T002 after T001
- T003 after T002 (documentation should reflect final skill behavior)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Single task for both template sections (T001) | Both sections are in the same file; splitting would create unnecessary coordination |
| 2 | Confident | `_generation.md` covers fab-continue and fab-ff awareness | Both skills delegate spec generation to `_generation.md`'s Spec Generation Procedure |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-12

- **Q**: Task IDs skip T003 (removed with fab-discuss). Should the remaining tasks be renumbered for sequential consistency?
  **A**: Accepted recommendation: Renumber T004 → T003 and update Execution Order references.
