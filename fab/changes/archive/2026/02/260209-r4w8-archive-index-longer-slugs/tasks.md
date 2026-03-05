# Tasks: Add archive index and allow longer folder slugs

**Change**: 260209-r4w8-archive-index-longer-slugs
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Slug Length Update

- [x] T001 [P] Update slug constraint in `fab/.kit/skills/fab-new.md` — change "2-4 words" to "2-6 words" in Step 1 (slug generation rules table and examples)
- [x] T002 [P] Update slug constraint in `fab/.kit/skills/fab-discuss.md` — change "2-4 words" references to "2-6 words" in Step 6 (folder name generation, same rules as `/fab-new`)

## Phase 2: Archive Index in fab-archive

- [x] T003 Add archive index maintenance step to `fab/.kit/skills/fab-archive.md` — insert a new step between Step 5 (Move Change Folder) and Step 6 (Clear Pointer) that appends an entry to `fab/changes/archive/index.md`. If `index.md` doesn't exist, create it with backfill of all existing archived changes. Entry format: `- **{folder-name}** — {1-2 sentence description from proposal Why section}`. New entries prepended at top (most-recent-first).

## Phase 3: Centralized Docs Update

- [x] T004 [P] Update `fab/docs/fab-workflow/planning-skills.md` — change slug constraint from "2-4 words" to "2-6 words" in the `/fab-new` Folder Name Generation section
- [x] T005 [P] Update `fab/docs/fab-workflow/change-lifecycle.md` — change slug constraint from "2-4 words" to "2-6 words" in the Folder Naming Convention table
- [x] T006 [P] Update `fab/docs/fab-workflow/configuration.md` — no change needed (naming format is `{YYMMDD}-{XXXX}-{slug}` which doesn't specify word count)
- [x] T007 [P] Update `fab/docs/fab-workflow/execution-skills.md` — add archive index maintenance to the `/fab-archive` Behavior section

---

## Execution Order

- T001 and T002 are independent (parallel)
- T003 depends on nothing (but logically follows Phase 1)
- T004-T007 are independent of each other (parallel), depend on T001-T003 being complete for consistency
