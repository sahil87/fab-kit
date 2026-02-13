# Tasks: Extract scaffold content into fab/.kit/scaffold/

**Change**: 260213-3njv-scaffold-dir
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/scaffold/` directory with four scaffold files: `envrc` (copy from `fab/.kit/envrc`), `gitignore-entries` (one line: `fab/current`), `docs-index.md` (extract heredoc from `fab/.kit/scripts/_fab-scaffold.sh` lines 101-109), `design-index.md` (extract heredoc from `fab/.kit/scripts/_fab-scaffold.sh` lines 115-131)

## Phase 2: Core Implementation

- [x] T002 Update `fab/.kit/scripts/_fab-scaffold.sh` section 2 (.envrc): change `envrc_target` from `fab/.kit/envrc` to `fab/.kit/scaffold/envrc`
- [x] T003 Update `fab/.kit/scripts/_fab-scaffold.sh` sections 3-4 (docs/index.md and design/index.md): replace `cat` heredocs with `cp` from `$kit_dir/scaffold/docs-index.md` and `$kit_dir/scaffold/design-index.md` (preserving the idempotent `if [ ! -f ... ]` guard)
- [x] T004 Update `fab/.kit/scripts/_fab-scaffold.sh` section 7 (.gitignore): replace hardcoded `grep -qx 'fab/current'` with a loop over lines in `$kit_dir/scaffold/gitignore-entries`, skipping comments (`#`) and empty lines, appending each missing entry
- [x] T005 Remove `fab/.kit/envrc` (replaced by `fab/.kit/scaffold/envrc`)

## Phase 3: Documentation

- [x] T006 [P] Update `fab/docs/fab-workflow/kit-architecture.md`: add `scaffold/` with its four files to the Directory Structure tree listing (as sibling of `templates/`, `scripts/`, `schemas/`)
- [x] T007 [P] Update `fab/docs/fab-workflow/init.md`: update Delegation Pattern table — `.envrc symlink` row to reference `fab/.kit/scaffold/envrc`, `.gitignore` row to mention `scaffold/gitignore-entries`, `Skeleton files` row to mention scaffold source files
- [x] T008 [P] Update `fab/docs/fab-workflow/distribution.md`: update Bootstrap scenario description to mention `_fab-scaffold.sh` reads from `scaffold/` files for index templates and gitignore entries

---

## Execution Order

- T001 blocks T002, T003, T004, T005 (scaffold files must exist before script reads them)
- T002, T003, T004 are sequential (same file: `_fab-scaffold.sh`)
- T005 is independent of T002-T004 but follows T001
- T006, T007, T008 are independent of each other (different files)
