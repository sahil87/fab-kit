# Tasks: README Quick Start Restructure + fab-sync Prerequisites Check

**Change**: 260217-zkah-readme-quickstart-prereqs-check
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Rename `fab/.kit/sync/1-direnv.sh` → `fab/.kit/sync/2-direnv.sh`
- [x] T002 Rename `fab/.kit/sync/2-sync-workspace.sh` → `fab/.kit/sync/3-sync-workspace.sh`

## Phase 2: Core Implementation

- [x] T003 Create `fab/.kit/sync/1-prerequisites.sh` — check for yq, jq, gh, direnv, bats using `command -v`; collect all missing tools; exit 1 with actionable error if any missing
- [x] T004 Restructure Quick Start in `README.md` — fold Initialize under Install as h4 sub-section, add "From a local clone" as h4 sub-section, add Updating as h4 sub-section under Install, renumber steps (Your first change → 2, Going parallel → 3)
- [x] T005 Remove standalone `## Updating` section from `README.md`
- [x] T006 Update Contents TOC line in `README.md` — remove `Updating` entry

## Phase 3: Integration & Edge Cases

- [x] T007 Update existing bats tests if any reference the old sync step filenames (`1-direnv.sh`, `2-sync-workspace.sh`)

---

## Execution Order

- T001 and T002 are independent, can run in parallel
- T003 is independent of T001/T002 (new file)
- T004 and T005 should be done together (both edit `README.md`)
- T006 depends on T004/T005 (needs final structure to set TOC)
- T007 runs last (depends on all renames being complete)
