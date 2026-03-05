# Tasks: Scoring Formula Produces Inflated Scores

**Change**: 260212-29xv-scoring-formula
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Formula Update

<!-- Change the Confident penalty from 0.1 to 0.3 in the two authoritative formula locations -->

- [x] T001 Update confidence formula in `fab/.kit/skills/_context.md` — change `0.1 * confident` to `0.3 * confident` in the Confidence Scoring section (line ~257). Update the Penalty Weights table: Confident row from `0.1` to `0.3` with rationale "Moderate — strong signal but still an assumption; accumulates meaningfully". Update the "What 3.0 Allows" commentary in the Gate Threshold subsection.
- [x] T002 Update confidence formula in `fab/design/srad.md` — change formula (line ~85), penalty weights table (line ~93), and the "What 3.0 Allows" section (line ~126) to reflect 0.3 penalty. Update all worked examples that show score calculations.

## Phase 2: Clarify Grade Reclassification

<!-- Add the mechanism for resolved assumptions to change grades in the Assumptions table -->

- [x] T003 Update `fab/.kit/skills/fab-clarify.md` — in the Suggest Mode section, after the artifact update step (Step 5), add grade reclassification: when a Tentative or Confident assumption is resolved/confirmed, update the `## Assumptions` table grade to `Certain`. Update Step 7 (Recompute Confidence Score) to specify the recount reads grades from Assumptions tables across all artifacts.
- [x] T004 Update `fab/docs/fab-workflow/clarify.md` — add grade reclassification behavior to the Confidence Recomputation subsection under Requirements. Specify that resolved Tentative/Confident assumptions become Certain in the Assumptions table before recomputation.

## Phase 3: Documentation Consistency

<!-- Ensure all documentation references are consistent with the new formula -->

- [x] T005 [P] Review and update `fab/docs/fab-workflow/planning-skills.md` — verify confidence scoring references. The file references the formula indirectly ("Recompute confidence score...apply formula"). No formula is inline, but verify no stale 0.1 references exist. Add changelog entry for this change.
- [x] T006 [P] Add changelog entries to `fab/docs/fab-workflow/clarify.md` and `fab/design/srad.md` for this change.

---

## Execution Order

- T001 and T002 are independent (parallel OK) — both update the formula in different files
- T003 depends on T001 (clarify skill references the formula from `_context.md`)
- T004 depends on T003 (doc mirrors the skill behavior)
- T005, T006 are independent polish tasks, can run after T001-T004
