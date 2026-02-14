# Tasks: Rename Batch Scripts and Add Batch Archive

**Change**: 260213-v3rn-batch-commands
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Renames

- [x] T001 [P] Rename `fab/.kit/scripts/fab-batch-new.sh` → `fab/.kit/scripts/batch-new-backlog.sh` via `git mv`; update comment header to reference new name
- [x] T002 [P] Rename `fab/.kit/scripts/fab-batch-switch.sh` → `fab/.kit/scripts/batch-switch-change.sh` via `git mv`; update comment header to reference new name

## Phase 2: Core Implementation

- [x] T003 Create `fab/.kit/scripts/batch-archive-change.sh` — new batch archive script following the worktree + tmux + Claude pattern from `batch-new-backlog.sh` and `batch-switch-change.sh`. Includes: `set -euo pipefail` boilerplate, `usage()`, `--list`/`--all`/`-h`/`--help`, substring matching for change resolution, `hydrate: done` detection via grep on `.status.yaml`, `wt-create` + tmux tab + `claude --dangerously-skip-permissions '/fab-archive <change>'`

## Phase 3: Verification

- [x] T004 Verify no references to old script names (`fab-batch-new.sh`, `fab-batch-switch.sh`) remain anywhere in the codebase

---

## Execution Order

- T001 and T002 are independent (parallel)
- T003 depends on T001 and T002 (uses renamed scripts as pattern reference)
- T004 depends on T001 and T002
