# Tasks: Re-evaluate Checklist Folder Location

**Change**: 260212-ipoe-checklist-folder-location
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Update `.status.yaml` template default path from `checklists/quality.md` to `checklist.md` in `fab/.kit/templates/status.yaml`

## Phase 2: Core Implementation

- [x] T002 [P] Remove `checklists/` directory creation from `/fab-new` in `fab/.kit/skills/fab-new.md` (line 121)
- [x] T003 [P] Update Checklist Generation Procedure in `fab/.kit/skills/_generation.md` — change path from `fab/changes/{name}/checklists/quality.md` to `fab/changes/{name}/checklist.md` and remove `checklists/` directory existence check (lines 57-58)
- [x] T004 [P] Update `/fab-continue` checklist paths in `fab/.kit/skills/fab-continue.md` — reset path (line 198) and output template (line 244)
- [x] T005 [P] Update `/fab-ff` checklist paths in `fab/.kit/skills/fab-ff.md` — review validation path (line 161) and output template (line 220)
- [x] T006 [P] Update `/fab-review` checklist paths in `fab/.kit/skills/fab-review.md` — all references to `checklists/quality.md` (lines 30, 40, 54, 83, 90, 290, 308)
- [x] T007 [P] Update `/fab-archive` checklist paths in `fab/.kit/skills/fab-archive.md` — all references to `checklists/quality.md` (lines 30, 40, 67)

## Phase 3: Integration & Edge Cases

- [x] T008 One-time migration: scan `fab/changes/` for active (non-archived) change folders containing `checklists/quality.md`, move to `checklist.md` at change root, remove empty `checklists/` directory, update `.status.yaml` `checklist.path` in each migrated change (no-op: no active changes had checklists/quality.md)

---

## Execution Order

- T001 is independent setup
- T002-T007 are all independent (different files), can run in parallel
- T008 (migration) can run after T001 since it references the new path convention
