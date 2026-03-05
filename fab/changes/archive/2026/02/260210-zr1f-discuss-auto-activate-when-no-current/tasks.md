# Tasks: Auto-activate after /fab-discuss when no current change

**Change**: 260210-zr1f-discuss-auto-activate-when-no-current
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Core Skill Change

- [x] T001 Add conditional activation logic to `fab/.kit/skills/fab-discuss.md` — insert a new Step 7.5 (between Display Summary and Next Steps) that checks `fab/current`, prompts the user, and calls `/fab-switch` internally if accepted. Update Step 6 items 3-4 to reference the new conditional behavior instead of blanket "do NOT write".
- [x] T002 Update Key Properties table in `fab/.kit/skills/fab-discuss.md` — change "Switches active change?" to conditional, "Creates git branch?" to conditional.
- [x] T003 Update Key Differences table in `fab/.kit/skills/fab-discuss.md` — change "Sets active change" and "Git integration" rows for fab-discuss to reflect conditional behavior.
- [x] T004 Update Output section examples in `fab/.kit/skills/fab-discuss.md` — add a "New Change Mode (Activated)" example showing the offer accepted flow with adjusted Next line. Update existing "New Change Mode" example header to clarify it's the declined/existing-active case.
- [x] T005 Update Next Steps Reference section in `fab/.kit/skills/fab-discuss.md` — add the activated case.
- [x] T006 Update Error Handling table in `fab/.kit/skills/fab-discuss.md` — add row for `/fab-switch` failure during activation.
- [x] T007 Update Purpose paragraph in `fab/.kit/skills/fab-discuss.md` — remove "does NOT switch the active change" absolute statement, replace with conditional description.

## Phase 2: Context and Docs Updates

- [x] T008 [P] Update Next Steps lookup table in `fab/.kit/skills/_context.md` — add `/fab-discuss` (new, activated) row.
- [x] T009 [P] Update `fab/docs/fab-workflow/change-lifecycle.md` — replace the "Not written by `/fab-discuss`" bullet with conditional description, add changelog entry.
- [x] T010 [P] Update `fab/docs/fab-workflow/planning-skills.md` — update `/fab-discuss` Proposal Output subsection, Key Differences table, and add changelog entry.

## Phase 3: Backlog Cleanup

- [x] T011 Mark backlog item `[s3d6]` as done in `fab/backlog.md`.

---

## Execution Order

- T001 through T007 are sequential (all modify the same file — `fab-discuss.md`)
- T008, T009, T010 are parallel (different files)
- T011 is independent
