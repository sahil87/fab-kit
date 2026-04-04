# Tasks: Remove Old Operator Skills

**Change**: 260331-eeso-remove-old-operator-skills
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Delete Files

- [x] T001 [P] Delete `fab/.kit/skills/fab-operator5.md`
- [x] T002 [P] Delete `fab/.kit/skills/fab-operator6.md`
- [x] T003 [P] Delete `fab/.kit/scripts/fab-operator5.sh`
- [x] T004 [P] Delete `fab/.kit/scripts/fab-operator6.sh`
- [x] T005 [P] Delete `fab/.kit/scripts/fab-operator4.sh` <!-- N/A: file doesn't exist on main -->
- [x] T006 [P] Delete `docs/specs/skills/SPEC-fab-operator5.md`

## Phase 2: Update Memory Files

- [x] T007 Remove the `/fab-operator5` section (lines ~352-374) from `docs/memory/fab-workflow/execution-skills.md`. Update the "Dependency-Aware Agent Spawning (operator7)" design decision to remove "extends operator6" reference. Preserve all changelog entries unchanged.
- [x] T008 Update `docs/memory/fab-workflow/kit-architecture.md`: remove `fab-operator4.sh` and `fab-operator5.sh` from directory tree and launcher descriptions. Update `lib/spawn.sh` description to reference `fab-operator7.sh` only. Add `fab-operator7.sh` launcher description.
- [x] T009 Update `docs/memory/fab-workflow/index.md`: change execution-skills row description to reference `/fab-operator7` instead of `/fab-operator4` and `/fab-operator5`.

## Phase 3: Update Spec Files

- [x] T010 Update `docs/specs/superpowers-comparison.md`: simplify "operator5/6/7" references to "operator" or "operator7".

---

## Execution Order

- Phase 1 tasks (T001-T006) are all independent and parallel
- Phase 2 tasks (T007-T009) are independent of each other but depend on Phase 1
- Phase 3 (T010) is independent of Phase 2
