# Tasks: Richer Git PR Output

**Change**: 260227-8q33-richer-git-pr-output
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Update Tier 1 PR body template in `fab/.kit/skills/git-pr.md` Step 3c: replace the combined `| [Intake](url) | [Spec](url) |` row with separate `| Intake | [{change_name}/intake.md]({url}) |` and `| Spec | [{change_name}/spec.md]({url}) |` rows following the Field|Detail pattern. Omit Spec row when `spec.md` absent (remove the "emit empty cell" fallback).

- [x] T002 Add Confidence and Pipeline rows to the Tier 1 template in `fab/.kit/skills/git-pr.md` Step 3c: insert `| Confidence | {score} / 5.0 |` and `| Pipeline | {stages} |` rows between the Change row and the Intake row. Read `confidence.score` and `progress` map from `.status.yaml`. Pipeline value = stage names with `done` status joined with ` → `, in fixed order: intake, spec, tasks, apply, review, hydrate.

- [x] T003 Update the instruction text surrounding the Tier 1 template in `fab/.kit/skills/git-pr.md` Step 3c to document the new field reading behavior: read `.status.yaml` via the agent (file is available after `changeman.sh resolve`), extract `confidence.score` and iterate `progress` keys. Remove the old "If spec.md does not exist, still emit a two-column row with the Spec cell empty" instruction.

## Phase 2: Memory Update

- [x] T004 Update `docs/memory/fab-workflow/execution-skills.md` — update the PR type system paragraph to mention the expanded Tier 1 Context table fields (confidence, pipeline, separate artifact link rows).

---

## Execution Order

- T001 and T002 both modify the same template block in `fab/.kit/skills/git-pr.md` — execute T001 first, then T002 builds on the updated structure.
- T003 depends on T001 + T002 (updates surrounding prose).
- T004 is independent of T001–T003 but logically comes after.
