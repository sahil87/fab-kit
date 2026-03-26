# Tasks: Operator Enforce Pipeline Routing

**Change**: 260326-u3un-operator-enforce-pipeline-routing
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Add "Pipeline-first routing" principle to §1 Principles in `fab/.kit/skills/fab-operator7.md` — insert after "Self-manage context" as a new principle paragraph
- [x] T002 [P] Add reinforcing blockquote note at the top of §6 "Working a Change" subsection in `fab/.kit/skills/fab-operator7.md` — before the three work path descriptions

## Phase 2: Memory Hydration

- [x] T003 Update `docs/memory/fab-workflow/execution-skills.md` — add a design decision documenting the pipeline-first routing principle under the operator7 section, and add a changelog entry

---

## Execution Order

- T001 and T002 are independent ([P]) — both edit `fab-operator7.md` but in different sections
- T003 depends on T001 and T002 (needs final wording to document)
