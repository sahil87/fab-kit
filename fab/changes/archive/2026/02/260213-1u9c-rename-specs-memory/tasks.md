# Tasks: Rename design/ → specs/ and docs/ → memory/

**Change**: 260213-1u9c-rename-specs-memory
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Directory Renames

- [x] T001 Rename `fab/design/` → `fab/specs/` via `git mv fab/design fab/specs`
- [x] T002 Rename `fab/docs/` → `fab/memory/` via `git mv fab/docs fab/memory`

## Phase 2: Kit Source Updates

<!-- All [P] tasks are independent find-and-replace within different directories -->

- [x] T003 [P] Update all skill files in `fab/.kit/skills/` — replace `fab/docs/` → `fab/memory/`, `fab/design/` → `fab/specs/`, `../docs/` → `../memory/`, `../design/` → `../specs/`. Files: `_context.md`, `fab-continue.md`, `fab-ff.md`, `fab-new.md`, `fab-init.md`, `fab-switch.md`, `fab-clarify.md`, `fab-hydrate-design.md`, `fab-hydrate.md`, `internal-consistency-check.md`, `internal-retrospect.md`
- [x] T004 [P] Update all scripts in `fab/.kit/scripts/` — replace `fab/docs/` → `fab/memory/`, `fab/design/` → `fab/specs/`, `docs/index.md` → `memory/index.md`, `design/index.md` → `specs/index.md`. Files: `_fab-scaffold.sh`, `fab-help.sh`, `stageman.sh`
- [x] T005 [P] Update templates in `fab/.kit/templates/` — replace `fab/docs/` → `fab/memory/`. Files: `brief.md`, `spec.md`
- [x] T006 [P] Update scaffold files in `fab/.kit/scaffold/` — (a) rename `design-index.md` → `specs-index.md`, (b) update `../docs/index.md` → `../memory/index.md` inside the renamed file, (c) rename `docs-index.md` → `memory-index.md`, (d) update header "Documentation Index" → "Memory Index" inside the renamed file

## Phase 3: Project Config and Root Files

- [x] T007 [P] Update `fab/config.yaml` — replace `fab/docs/` → `fab/memory/`, `fab/design/` → `fab/specs/` in comments
- [x] T008 [P] Update `fab/constitution.md` — replace `fab/docs/` → `fab/memory/`, `fab/design/` → `fab/specs/`
- [x] T009 [P] Update `README.md` — replace `fab/docs/` → `fab/memory/`, `fab/design/` → `fab/specs/`, and update any prose references to "docs" and "design" that refer to these folders

## Phase 4: Internal Cross-References Within Renamed Folders

- [x] T010 [P] Update all files in `fab/memory/` (formerly `fab/docs/`) — replace `fab/docs/` → `fab/memory/`, `fab/design/` → `fab/specs/`, `../design/` → `../specs/`. Files: `index.md`, `fab-workflow/index.md`, `fab-workflow/planning-skills.md`, `fab-workflow/distribution.md`, `fab-workflow/templates.md`, `fab-workflow/kit-architecture.md`, `fab-workflow/hydrate-generate.md`, `fab-workflow/init.md`, `fab-workflow/design-index.md`, `fab-workflow/context-loading.md`, `fab-workflow/hydrate-design.md`, `fab-workflow/hydrate.md`, `fab-workflow/execution-skills.md`
- [x] T011 [P] Update all files in `fab/specs/` (formerly `fab/design/`) — replace `fab/docs/` → `fab/memory/`, `fab/design/` → `fab/specs/`, `../docs/` → `../memory/`. Files: `index.md`, `architecture.md`, `templates.md`, `overview.md`, `glossary.md`, `skills.md`

## Phase 5: Headers and Prose

- [x] T012 Update index headers — `fab/memory/index.md`: "Documentation Index" → "Memory Index"; `fab/specs/index.md`: "Design Index" → "Specs Index". Update prose in both files to use new terminology consistently.

## Phase 6: Verification

- [x] T013 Run `grep -r 'fab/docs/' fab/.kit/ fab/memory/ fab/specs/ fab/config.yaml fab/constitution.md README.md` and `grep -r 'fab/design/' fab/.kit/ fab/memory/ fab/specs/ fab/config.yaml fab/constitution.md README.md` to verify no stale references remain in source files (excluding `fab/changes/`)

---

## Execution Order

- T001 and T002 MUST complete before T003–T013 (directories must exist at new paths)
- T003–T011 are independent and parallelizable
- T012 depends on T010 and T011 (files must have paths updated before header/prose changes)
- T013 depends on all preceding tasks (verification runs last)
