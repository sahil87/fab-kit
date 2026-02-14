# Tasks: Relocate memory and specs to docs/

**Change**: 260214-m3v8-relocate-docs-dev-scripts
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Directory Relocation

- [x] T001 Create `docs/` directory, move `fab/memory/` → `docs/memory/` and `fab/specs/` → `docs/specs/`
- [x] T002 Update `fab/.kit/scripts/_init_scaffold.sh` — create `docs/memory/` and `docs/specs/` instead of `fab/memory/` and `fab/specs/`

## Phase 2: Core Reference Updates

- [x] T003 [P] Update `fab/.kit/skills/_context.md` — always-load paths (`docs/memory/index.md`, `docs/specs/index.md`), selective domain loading paths, memory file lookup paths
- [x] T004 [P] Update `fab/constitution.md` — Principle II (`docs/memory/`), Principle VI (`docs/specs/`)
- [x] T005 [P] Update templates `fab/.kit/templates/brief.md` and `fab/.kit/templates/spec.md` — affected memory path pattern
- [x] T006 [P] Update scaffold files `fab/.kit/scaffold/memory-index.md` and `fab/.kit/scaffold/specs-index.md` — boilerplate text and cross-references
- [x] T007 [P] Update `fab/.kit/scripts/fab-help.sh` and `fab/.kit/scripts/_stageman.sh` — path references
- [x] T008 [P] Update `README.md` — all `fab/memory/` and `fab/specs/` links and references

## Phase 3: Skill File Updates

- [x] T009 [P] Update `fab/.kit/skills/fab-new.md` — path references
- [x] T010 [P] Update `fab/.kit/skills/fab-init.md` — path references
- [x] T011 [P] Update `fab/.kit/skills/fab-continue.md` — path references
- [x] T012 [P] Update `fab/.kit/skills/fab-ff.md` — path references
- [x] T013 [P] Update `fab/.kit/skills/fab-fff.md` — path references
- [x] T014 [P] Update `fab/.kit/skills/fab-archive.md` — path references
- [x] T015 [P] Update `fab/.kit/skills/docs-hydrate-memory.md` — path references
- [x] T016 [P] Update `fab/.kit/skills/docs-hydrate-specs.md` — path references
- [x] T017 [P] Update `fab/.kit/skills/docs-reorg-memory.md` and `fab/.kit/skills/docs-reorg-specs.md` — path references
- [x] T018 [P] Update `fab/.kit/skills/internal-consistency-check.md` and `fab/.kit/skills/internal-retrospect.md` — path references

## Phase 4: Relocated File Updates + Migration

- [x] T019 [P] Update `docs/memory/index.md` and `docs/specs/index.md` — boilerplate text referencing old paths
- [x] T020 [P] Update individual memory files under `docs/memory/fab-workflow/` that reference `fab/memory/` or `fab/specs/` paths
- [x] T021 Create migration file `fab/.kit/migrations/0.2.0-to-0.3.0.md`

---

## Execution Order

- T001 blocks all other tasks (directories must exist at new locations first)
- T002 is independent of the move (scaffold script creates structure for new projects)
- T003–T018 are all independent of each other (different files)
- T019–T020 depend on T001 (files must be at new locations)
- T021 is independent
