# Tasks: Extract shared generation logic from fab-continue and fab-ff

**Change**: 260210-wpay-extract-shared-generation-logic
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/skills/_generation.md` with header and purpose section explaining it is a shared partial for artifact generation

## Phase 2: Core Implementation

- [x] T002 Extract spec generation procedure from `fab-continue.md` (lines 97-111, "Generating `spec.md`" section) and `fab-ff.md` (lines 82-98, "Generate `spec.md`" Step 2) into `_generation.md` as "## Spec Generation Procedure"
- [x] T003 Extract plan generation procedure from `fab-continue.md` (lines 132-148, "Generating `plan.md`" section) and `fab-ff.md` (lines 137-156, plan generation within Step 3) into `_generation.md` as "## Plan Generation Procedure"
- [x] T004 Extract tasks generation procedure from `fab-continue.md` (lines 150-169, "Generating `tasks.md`" section) and `fab-ff.md` (lines 164-185, "Generate `tasks.md`" Step 4) into `_generation.md` as "## Tasks Generation Procedure"
- [x] T005 Extract checklist generation procedure from `fab-continue.md` (lines 173-194, "Auto-generate Checklist" section) and `fab-ff.md` (lines 191-207, "Auto-generate Quality Checklist" Step 5) into `_generation.md` as "## Checklist Generation Procedure"

## Phase 3: Integration & Edge Cases

- [x] T006 Replace inline spec generation steps in `.agents/skills/fab-continue/SKILL.md` with a reference to `_generation.md` "Spec Generation Procedure", preserving the orchestration context (SRAD questions, plan decision, reset flow)
- [x] T007 Replace inline plan generation steps in `.agents/skills/fab-continue/SKILL.md` with a reference to `_generation.md` "Plan Generation Procedure"
- [x] T008 Replace inline tasks generation steps in `.agents/skills/fab-continue/SKILL.md` with a reference to `_generation.md` "Tasks Generation Procedure"
- [x] T009 Replace inline checklist generation steps in `.agents/skills/fab-continue/SKILL.md` with a reference to `_generation.md` "Checklist Generation Procedure"
- [x] T010 Replace inline spec generation steps in `.agents/skills/fab-ff/SKILL.md` with a reference to `_generation.md` "Spec Generation Procedure", preserving auto-clarify and bail logic
- [x] T011 Replace inline plan generation steps in `.agents/skills/fab-ff/SKILL.md` with a reference to `_generation.md` "Plan Generation Procedure"
- [x] T012 Replace inline tasks generation steps in `.agents/skills/fab-ff/SKILL.md` with a reference to `_generation.md` "Tasks Generation Procedure"
- [x] T013 Replace inline checklist generation steps in `.agents/skills/fab-ff/SKILL.md` with a reference to `_generation.md` "Checklist Generation Procedure"

## Phase 4: Polish

- [x] T014 Verify that both `fab-continue.md` and `fab-ff.md` still read coherently end-to-end after the extraction — ensure reference text flows naturally with surrounding orchestration logic

---

## Execution Order

- T001 blocks T002-T005 (partial must exist before content is added)
- T002-T005 can run sequentially (each adds a section to `_generation.md`)
- T006-T009 depend on T002-T005 (shared content must exist before referencing it)
- T010-T013 depend on T002-T005 (same reason)
- T006-T009 and T010-T013 are independent groups (different files)
- T014 depends on all prior tasks
