# Tasks: Fix fab-new premature brief completion

**Change**: 260212-r7xp-fix-fab-new-brief-stage
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Fix

- [x] T001 Remove Step 8 ("Mark Brief Complete") from `fab/.kit/skills/fab-new.md` — delete the entire step (lines 206-213) including the heading, description, and both sub-items (status.yaml writes and progress transitions)
- [x] T002 Renumber Step 9 → Step 8 in `fab/.kit/skills/fab-new.md` — update the heading "Step 9: Activate Change via `/fab-switch`" to "Step 8"

## Phase 2: Reference Updates

- [x] T003 [P] Update Next Steps lookup table in `fab/.kit/skills/_context.md` — change `/fab-new` row from `brief done` to `brief active` and update the Next line to show the default (no-switch) command
- [x] T004 [P] Update Change Initialization list in `fab/docs/fab-workflow/planning-skills.md` — remove step 4 ("Mark brief complete once the user is satisfied") and renumber step 5 → step 4

---

## Execution Order

- T001 blocks T002 (renumbering depends on removal)
- T003 and T004 are independent of each other and of T001-T002 (different files)
