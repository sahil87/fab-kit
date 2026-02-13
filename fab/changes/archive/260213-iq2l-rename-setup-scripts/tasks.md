# Tasks: Rename and Reorganize Bootstrap Scripts

**Change**: 260213-iq2l-rename-setup-scripts
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Script Renames and Content Updates

<!-- Rename the 3 script files and update their internal self-references. -->

- [x] T001 [P] Rename `fab/.kit/scripts/fab-setup.sh` → `fab/.kit/scripts/_fab-scaffold.sh` via `git mv`. Update the file's internal self-referencing comments (header comment line 4: `fab-setup.sh` → `_fab-scaffold.sh`, line 9: usage line).
- [x] T002 [P] Rename `fab/.kit/scripts/fab-update.sh` → `fab/.kit/scripts/fab-upgrade.sh` via `git mv`. Update internal references: header comment (line 4: `fab-update.sh` → `fab-upgrade.sh`), line 7 (`fab-setup.sh` → `_fab-scaffold.sh`), line 96 echo (`fab-setup.sh` → `_fab-scaffold.sh`), line 97 bash call (`fab-setup.sh` → `_fab-scaffold.sh`).
- [x] T003 [P] Rename `fab/.kit/worktree-init-common/2-rerun-fab-setup.sh` → `fab/.kit/worktree-init-common/2-rerun-fab-scaffold.sh` via `git mv`. Update content: line 2 call from `fab/.kit/scripts/fab-setup.sh` to `fab/.kit/scripts/_fab-scaffold.sh`.

## Phase 2: Internal Caller Updates

<!-- Update kit-internal files that reference the old script names. -->

- [x] T004 [P] Update `fab/.kit/skills/fab-init.md` — replace all `fab-setup.sh` occurrences with `_fab-scaffold.sh`.
- [x] T005 [P] Update `fab/.kit/model-tiers.yaml` — replace comment reference `fab-setup.sh` → `_fab-scaffold.sh` (line 4).
- [x] T006 [P] Update `README.md` — replace `fab-setup.sh` → `_fab-scaffold.sh` and `fab-update.sh` → `fab-upgrade.sh` in installation and upgrade instructions.

## Phase 3: Centralized Docs

<!-- Update all fab/docs/ and fab/design/ references to old script names. -->

- [x] T007 [P] Update `fab/docs/fab-workflow/kit-architecture.md` — directory listing tree (`fab-setup.sh` → `_fab-scaffold.sh`, `fab-update.sh` → `fab-upgrade.sh`), script description sections (rename headers and body text), bootstrap sequence references, all other occurrences.
- [x] T008 [P] Update `fab/docs/fab-workflow/distribution.md` — replace `fab-setup.sh` → `_fab-scaffold.sh` and `fab-update.sh` → `fab-upgrade.sh` throughout (bootstrap steps, update script section header and body, symlink repair references).
- [x] T009 [P] Update `fab/docs/fab-workflow/init.md` — replace all `fab-setup.sh` → `_fab-scaffold.sh` in delegation pattern, responsibility table, and bootstrap path description.
- [x] T010 [P] Update remaining fab-workflow docs: `model-tiers.md` (line 98: `fab-setup.sh` → `_fab-scaffold.sh`), `hydrate.md` (line 13: `fab-setup.sh` → `_fab-scaffold.sh`), `templates.md` (line 98: `fab-setup.sh` → `_fab-scaffold.sh`), `index.md` (doc descriptions mentioning `fab-setup.sh`).
- [x] T011 [P] Update design docs: `fab/design/architecture.md` (directory listing line 34, references lines 345, 379, 387) and `fab/design/glossary.md` (line 75) — replace `fab-setup.sh` → `_fab-scaffold.sh`.

---

## Execution Order

- Phase 1 tasks (T001-T003) are independent — all rename different files
- Phase 2 depends on Phase 1 (old files no longer exist after rename)
- Phase 3 depends on Phase 1 (references new names)
- Within Phase 2 and Phase 3, all tasks are independent
