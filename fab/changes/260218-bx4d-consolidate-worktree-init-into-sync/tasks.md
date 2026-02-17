# Tasks: Consolidate worktree-init into fab-sync

**Change**: 260218-bx4d-consolidate-worktree-init-into-sync
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup & Scaffolds

- [x] T001 Create `fab/.kit/sync/` directory
- [x] T002 [P] Create `fab/.kit/scaffold/sync-readme.md` — scaffold content explaining project-specific sync scripts in `fab/sync/`, naming convention (numbered `*.sh` files, sorted order)

## Phase 2: Core — Move & Rewrite

- [x] T003 Move current `fab/.kit/scripts/fab-sync.sh` content to `fab/.kit/sync/2-sync-workspace.sh` — adjust path resolution: rename `scripts_dir` to `sync_dir`, `kit_dir` derivation changes from `dirname "$scripts_dir"` to `dirname "$sync_dir"`
- [x] T004 [P] Copy `fab/.kit/worktree-init-common/1-direnv.sh` to `fab/.kit/sync/1-direnv.sh` (identical content)
- [x] T005 Rewrite `fab/.kit/scripts/fab-sync.sh` as thin orchestrator — `set -euo pipefail`, derive paths, iterate `fab/.kit/sync/*.sh` then `fab/sync/*.sh` (if exists), print script names before execution
- [x] T006 Add sync README scaffold section to `fab/.kit/sync/2-sync-workspace.sh` — conditional create of `fab/sync/README.md` from `fab/.kit/scaffold/sync-readme.md`, after existing section 8

## Phase 3: Config & Renames

- [x] T007 [P] Update `fab/.kit/scaffold/envrc` — change `WORKTREE_INIT_SCRIPT=fab/.kit/worktree-init.sh` to `WORKTREE_INIT_SCRIPT=fab/.kit/scripts/fab-sync.sh`
- [x] T008 [P] Create `fab/sync/` from `fab/worktree-init/` — move `2-symlink-backlog.sh` → `1-symlink-backlog.sh` (content unchanged), delete `1-claude-settings.sh` and `assets/` directory
- [x] T009 [P] Delete `fab/.kit/worktree-init.sh`
- [x] T010 [P] Delete `fab/.kit/worktree-init-common/` (contents already moved by T003, T004)

## Phase 4: Documentation

- [x] T011 Update `README.md` — replace all references to `worktree-init.sh` and old directory names with new structure (no references found — already up to date)

---

## Execution Order

- T001 blocks T003, T004 (sync directory must exist before files are placed in it)
- T003 blocks T005 (content must be moved out before fab-sync.sh is rewritten)
- T003 blocks T006 (2-sync-workspace.sh must exist before adding the README section)
- T003, T004 block T010 (content must be moved before old directory is deleted)
