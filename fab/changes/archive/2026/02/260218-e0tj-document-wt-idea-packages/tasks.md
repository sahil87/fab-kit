# Tasks: Document wt and idea packages

**Change**: 260218-e0tj-document-wt-idea-packages
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Add PACKAGES footer section to `fab/.kit/scripts/fab-help.sh` — append static block after TYPICAL FLOW listing wt commands (wt-create, wt-list, wt-open, wt-delete, wt-init, wt-pr) and idea with one-liner descriptions, ending with "Run <command> help for details."
- [x] T002 [P] Create `docs/specs/packages.md` — new spec page with Overview, wt (Worktree Management), idea (Backlog Management), and Package Architecture sections per spec requirements
- [x] T003 [P] Add packages.md entry to `docs/specs/index.md` — new row in the specs table

---

## Execution Order

- T001, T002, T003 are all independent and parallelizable
