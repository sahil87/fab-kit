# Tasks: Clarify Tentative First

**Change**: 260416-hyl6-clarify-tentative-first
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Skill Reorder

- [x] T001 Reorder suggest mode steps in `src/kit/skills/fab-clarify.md` — move taxonomy scan to Step 1.5, bulk confirm to Step 2, update step numbering and cross-references within the file
- [x] T002 [P] Update `src/kit/skills/_preamble.md` Bulk Confirm subsection — replace "This flow runs as Step 1.5 in Suggest Mode, before the standard taxonomy scan (Step 2)" with "This flow runs as Step 2 in Suggest Mode, after the taxonomy scan and tentative resolution (Step 1.5)"

## Phase 2: Documentation Updates

- [x] T003 [P] Update `docs/specs/skills/SPEC-fab-clarify.md` flow diagram — swap Step 1.5 (Bulk Confirm) and Step 2 (Taxonomy Scan) in the suggest mode flow
- [x] T004 [P] Update `docs/memory/fab-workflow/clarify.md` — replace ordering text in the Bulk Confirm section: "before the taxonomy scan (Step 1.5)" → after, and "proceed to Step 2 (taxonomy scan)" → proceed to remaining taxonomy questions

---

## Execution Order

- T001 and T002 are independent (different files), can run in parallel
- T003 and T004 are independent, can run in parallel, and independent of T001/T002
