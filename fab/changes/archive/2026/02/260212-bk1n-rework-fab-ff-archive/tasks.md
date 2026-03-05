# Tasks: Rework fab-ff to go all the way to archive

**Change**: 260212-bk1n-rework-fab-ff-archive
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Implementation

- [x] T001 Rewrite `fab/.kit/skills/fab-ff.md` Behavior section to add execution stages (apply, review, archive) after the existing planning steps. Add Steps 6-8 for apply, review, and archive invocation. Include interactive review failure handling with rework menu. Include apply failure stop behavior. Extend resumability to cover all stages.
- [x] T002 Rewrite `fab/.kit/skills/fab-ff.md` Output section to include phased output with `--- Planning ---`, `--- Implementation (fab-apply) ---`, `--- Review (fab-review) ---`, `--- Archive (fab-archive) ---` section headers. Add output examples for: clean full pipeline, bail during planning, resume from mid-pipeline, review failure with rework.
- [x] T003 Update `fab/.kit/skills/fab-ff.md` comparison table (Key Difference section) to reflect that both fab-ff and fab-fff are now full-pipeline commands, differentiated by interaction model. Update Next Steps Reference.
- [x] T004 Update `fab/.kit/skills/fab-ff.md` Purpose, frontmatter description, and Error Handling table to reflect full pipeline scope.

## Phase 2: Sibling Skill Update

- [x] T005 Update `fab/.kit/skills/fab-fff.md` comparison table (Key Difference from Individual Skills) and description to reflect that fab-ff is now also a full-pipeline command. Clarify the differentiation: fab-fff has confidence gate + auto-clarify + immediate bail on review failure vs fab-ff's interactive stops.

## Phase 3: Documentation Updates

- [x] T006 [P] Update `fab/docs/fab-workflow/planning-skills.md` — update the `/fab-ff` section to describe it as a full-pipeline command (planning through archive), update the comparison table, update the Generation Flow, and add "When to Use" guidance differentiating from fab-fff.
- [x] T007 [P] Update `fab/docs/fab-workflow/execution-skills.md` — add a note in the Overview or a new section noting that `/fab-ff` can now invoke apply/review/archive internally as part of its extended pipeline.

---

## Execution Order

- T001 → T002 → T003 → T004 (sequential rewrite of fab-ff.md sections)
- T005 independent of T001-T004 but best done after T003 for comparison table consistency
- T006, T007 parallelizable, independent of each other
