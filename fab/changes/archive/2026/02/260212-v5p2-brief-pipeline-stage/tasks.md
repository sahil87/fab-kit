# Tasks: Add brief as a formal pipeline stage

**Change**: 260212-v5p2-brief-pipeline-stage
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Configuration

- [x] T001 Add `brief` stage definition to `fab/config.yaml` as the first entry in the `stages` list with `id: brief`, `generates: brief.md`, `required: true`, and no `requires` field. Add `requires: [brief]` to the existing `spec` stage entry.

## Phase 2: Core Skill Updates

- [x] T002 [P] Fix `fab/.kit/skills/fab-new.md` Step 4: remove the hardcoded `.status.yaml` YAML block that shows `spec: active` (no brief entry) and replace with a reference to use the template at `fab/.kit/templates/status.yaml` (which already has `brief: active`). Also fix Step 8 text that says "The brief is an input artifact, not a pipeline stage" — brief IS now a pipeline stage.
- [x] T003 [P] Fix `fab/.kit/skills/fab-continue.md` Error Handling table: remove the row `Reset target is brief → Abort with "Cannot reset to brief"`. The Normal Flow guard table and Reset Flow already correctly handle brief; only the Error Handling row contradicts them.
- [x] T004 [P] Update `fab/.kit/skills/fab-switch.md` Stage Number Mapping table: add `brief` as position 1, shift spec→2, tasks→3, apply→4, review→5, archive→6. Change the display format description from `(N/5)` to `(N/6)`.

## Phase 3: Documentation Updates

- [x] T005 [P] Update `fab/docs/fab-workflow/configuration.md`: change "5 stages" to "6 stages" in the `stages` section description. Add `brief` to the list of stage IDs (`brief, spec, tasks, apply, review, archive`).
- [x] T006 [P] Update `fab/docs/fab-workflow/change-lifecycle.md` Migration Note section: remove instruction "Remove `brief:` from the progress map — brief is no longer a pipeline stage". Replace with guidance that `brief:` SHOULD be present in all `.status.yaml` files; for legacy files missing it, the preflight script infers `brief: done`.

## Phase 4: Verification

- [x] T007 Run `fab/.kit/scripts/fab-preflight.sh` against a change with `brief: active` in `.status.yaml` and verify it outputs `stage: brief`. Confirm existing migration shim (missing brief → inferred done) works.
- [x] T008 Run `fab/.kit/scripts/fab-status.sh` and verify it displays `brief (1/6)` correctly for the current change.

---

## Execution Order

- T001 is the foundation — config change should land first
- T002, T003, T004 are independent skill updates (parallel)
- T005, T006 are independent doc updates (parallel)
- T007 blocks T008 (verify preflight before status)
- T007, T008 can run after all other tasks complete

## Already Correct (No Changes Needed)

These files were audited and already handle brief correctly:

- `fab/.kit/scripts/fab-preflight.sh` — iterates through `brief spec tasks apply review archive`, has migration shim for missing brief entry
- `fab/.kit/scripts/fab-status.sh` — extracts `p_brief`, maps brief→1 in stage numbering, displays `(N/6)`, includes brief in progress line output
- `fab/.kit/skills/fab-ff.md` — says "Can start from brief, spec, or tasks stage", handles brief active correctly
- `fab/.kit/skills/fab-fff.md` — delegates to fab-ff which handles brief
- `fab/.kit/skills/fab-clarify.md` — stage guard already lists brief as valid
- `fab/.kit/skills/fab-status.md` — delegates to shell script which handles brief
- `fab/docs/fab-workflow/planning-skills.md` — already describes 6-stage pipeline with brief first
