# Tasks: Add bulk confirm mode to fab-clarify

**Change**: 260302-c7is-fab-clarify-bulk-confirm
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add Step 1.5 (Bulk Confirm) section to `fab/.kit/skills/fab-clarify.md` — insert between Step 1 (Read Target Artifact) and Step 2 (Taxonomy Scan). Include: detection logic (`confident >= 3` AND `confident > tentative + unresolved`), display format (numbered list with original `#` column, no AskUserQuestion), response parsing (confirm/change/explain/range/all formats), artifact update (Grade/Rationale/Scores changes), re-prompt behavior (one round for explanation requests), and Clarifications audit trail format (`### Session {date} (bulk confirm)`)
- [x] T002 [P] Add `### Bulk Confirm (Confident Assumptions)` subsection to `fab/.kit/skills/_preamble.md` under `## Confidence Scoring` — document trigger conditions, flow location (Step 1.5 in Suggest Mode only), and update behavior. Keep concise (~10 lines of prose), reference `/fab-clarify` as implementing skill

## Phase 2: Integration

- [x] T003 Update Step 2 (Taxonomy Scan) heading/intro in `fab/.kit/skills/fab-clarify.md` to note it runs after bulk confirm when triggered — add a brief line clarifying the Step 1.5 → Step 2 sequencing
- [x] T004 Add explicit exclusion note to Auto Mode section in `fab/.kit/skills/fab-clarify.md` — confirm bulk confirm is Suggest Mode only, Auto Mode proceeds without it

---

## Execution Order

- T001 and T002 are independent (different files), can run in parallel
- T003 depends on T001 (needs the Step 1.5 content to reference)
- T004 depends on T001 (needs to verify what was added to Suggest Mode)
