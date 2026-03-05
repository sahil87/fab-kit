# Tasks: Scaffold Overlay Tree

**Change**: 260218-09fa-scaffold-overlay-tree
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create scaffold overlay tree structure — create subdirectories (`scaffold/.claude/`, `scaffold/docs/memory/`, `scaffold/docs/specs/`, `scaffold/fab/`, `scaffold/fab/sync/`), move and rename all 11 files per the spec mapping table in `fab/.kit/scaffold/`

## Phase 2: Core Implementation

- [x] T002 Add `line_ensure_merge` and `json_merge_permissions` helper functions to `fab/.kit/sync/3-sync-workspace.sh` — extract from current sections 2/7 (line-ensuring) and section 8 (JSON merge), absorb `.envrc` symlink migration into `line_ensure_merge`
- [x] T003 Replace bespoke scaffold sections (2, 3, 4, 7, 8, 9) with a generic scaffold tree-walk in `fab/.kit/sync/3-sync-workspace.sh` — walk `scaffold/` recursively, compute destination by stripping `scaffold/` prefix and `fragment-` filename prefix, dispatch to helper functions or copy-if-absent

## Phase 3: Reference Updates

- [x] T004 [P] Update `fab/.kit/skills/fab-setup.md` — change all scaffold path references to new overlay paths (7 paths per spec table), and change bootstrap detection for `config.yaml` (step 1a) and `constitution.md` (step 1b) from existence checks to placeholder checks (`{PROJECT_NAME}`, `{Project Name}`)
- [x] T005 [P] Update `fab/.kit/migrations/0.7.0-to-0.8.0.md` — change `fab/.kit/scaffold/code-quality.md` → `fab/.kit/scaffold/fab/code-quality.md`

## Phase 4: Verification

- [x] T006 Run `fab-sync.sh` end-to-end to verify tree-walk produces correct output — check that all scaffold files are processed, fragment merges work, copy-if-absent works, and no regressions in skill symlinks or agent files

---

## Execution Order

- T001 → T002 → T003 (sequential: directory restructure before helpers, helpers before tree-walk)
- T004, T005 independent of T002-T003 but require T001 (new paths must exist)
- T006 depends on all prior tasks
