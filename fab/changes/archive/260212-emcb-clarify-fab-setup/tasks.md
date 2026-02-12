# Tasks: Clarify fab-setup Responsibilities and Initialize fab/design Folder

**Change**: 260212-emcb-clarify-fab-setup
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Implementation

- [x] T001 Add `fab/design/` directory and `fab/design/index.md` skeleton creation to `fab/.kit/scripts/fab-setup.sh` — insert new section between existing section 3 (Docs index) and section 4 (Skill symlinks). Follow the same idempotent pattern: check `[ ! -f "$fab_dir/design/index.md" ]`, create directory with `mkdir -p`, write skeleton matching `/fab-init` step 1d content.

## Phase 2: Documentation

- [x] T002 [P] Add delegation pattern documentation to `fab/.kit/skills/fab-init.md` — add a note near the top of the Behavior section explaining that `/fab-init` delegates structural setup to `fab-setup.sh` (step 1f) and only adds interactive configuration (config.yaml, constitution.md) on top. Include the responsibility split table from the spec.
- [x] T003 [P] Add "Delegation Pattern" section to `fab/docs/fab-workflow/init.md` — new section explaining the relationship between `/fab-init` and `fab-setup.sh`: what each creates, that fab-setup.sh can be run independently, and that `/fab-init` invokes it as step 1f. Add changelog entry.
- [x] T004 [P] Update bootstrap scenarios in `fab/docs/fab-workflow/distribution.md` — update the `fab-setup.sh` description and bootstrap scenario to include `fab/design/` directory and `fab/design/index.md` in the list of structural artifacts created. Add changelog entry.

---

## Execution Order

- T001 is independent — no dependencies
- T002, T003, T004 are all independent of each other (can run in parallel) and independent of T001 (they document existing behavior + the new behavior, but don't depend on the script change being applied first)
