# Tasks: Operator Never-Ask Monitor Fix

**Change**: 260331-mvhj-operator-never-ask-monitor
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Add never-ask admonition and step 4 annotation to `fab/.kit/skills/fab-operator7.md` "Spawning an Agent" subsection (lines 297–306): insert blockquote after step 4, expand step 4 text
- [x] T002 [P] Add never-ask admonition to `fab/.kit/skills/fab-operator6.md` "Spawning an Agent" subsection (lines 253–259): insert blockquote after the tmux command block
- [x] T003 Add "Never ask whether to monitor a spawned agent" sentence to `fab/.kit/skills/fab-operator6.md` §1 "Automate the routine" principle (line 20), matching operator7's wording

## Phase 2: Documentation

- [x] T004 Update `docs/memory/fab-workflow/execution-skills.md` operator section to document the never-ask-monitor reinforcement pattern

---

## Execution Order

- T001 and T002 are independent (different files), can run in parallel
- T003 depends on T002 being complete (same file)
- T004 depends on T001–T003 (documents the completed changes)
