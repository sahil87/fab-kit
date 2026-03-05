# Tasks: Docs Skills Housekeeping

**Change**: 260214-r8kv-docs-skills-housekeeping
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Removal & Renames

- [x] T001 Delete `fab/.kit/scripts/fab-status.sh`
- [x] T002 [P] Rename `fab/.kit/skills/fab-hydrate-specs.md` → `fab/.kit/skills/docs-hydrate-specs.md`; update frontmatter `name` and heading to `docs-hydrate-specs`
- [x] T003 [P] Rename `fab/.kit/skills/fab-hydrate.md` → `fab/.kit/skills/docs-hydrate-memory.md`; update frontmatter `name` and heading to `docs-hydrate-memory`
- [x] T004 [P] Rename `fab/.kit/skills/fab-reorg-specs.md` → `fab/.kit/skills/docs-reorg-specs.md`; update frontmatter `name` and heading to `docs-reorg-specs`

## Phase 2: New Skill

- [x] T005 Create `fab/.kit/skills/docs-reorg-memory.md` — mirror `docs-reorg-specs.md` structure but target `fab/memory/` domain directories instead of `fab/specs/`

## Phase 3: Symlink Cleanup & Regeneration

- [x] T006 Remove stale symlink directories for old skill names: `.claude/skills/fab-hydrate-specs/`, `.claude/skills/fab-hydrate/`, `.claude/skills/fab-reorg-specs/`, `.opencode/commands/fab-hydrate-specs.md`, `.opencode/commands/fab-hydrate.md`, `.opencode/commands/fab-reorg-specs.md`, `.agents/skills/fab-hydrate-specs/`, `.agents/skills/fab-hydrate/`, `.agents/skills/fab-reorg-specs/`
- [x] T007 Run `fab/.kit/scripts/_fab-scaffold.sh` to regenerate symlinks for renamed and new skills

## Phase 4: Cross-Reference Updates

- [x] T008 [P] Update `fab/.kit/scripts/fab-help.sh` — replace old skill names with new names, remove `fab-status.sh` references
- [x] T009 [P] Update `fab/.kit/skills/fab-status.md` — remove delegation to `fab-status.sh`; note that the skill uses `fab-preflight.sh` + `stageman.sh`
- [x] T010 [P] Update `fab/memory/fab-workflow/kit-architecture.md` — remove `fab-status.sh` from directory listing and Shell Scripts section; update skill file names in the skills/ listing
- [x] T011 [P] Update `fab/memory/fab-workflow/hydrate.md` — rename `/fab-hydrate` skill references to `/docs-hydrate-memory`
- [x] T012 [P] Update `fab/memory/fab-workflow/execution-skills.md` — remove `fab-status.sh` references
- [x] T013 [P] Update `fab/memory/fab-workflow/init.md` — rename `/fab-hydrate` references to `/docs-hydrate-memory`
- [x] T014 [P] Update `fab/memory/fab-workflow/context-loading.md` — rename `/fab-hydrate` references to `/docs-hydrate-memory`
- [x] T015 [P] Update `fab/memory/fab-workflow/hydrate-generate.md` — rename `/fab-hydrate` references to `/docs-hydrate-memory`
- [x] T016 [P] Update `fab/memory/fab-workflow/model-tiers.md` — rename skill references (`fab-hydrate`, `fab-hydrate-specs`, `fab-status.sh`)
- [x] T017 [P] Update `fab/memory/fab-workflow/index.md` — rename skill references
- [x] T018 [P] Update `fab/memory/fab-workflow/change-lifecycle.md` — remove `fab-status.sh` reference
- [x] T019 [P] Update `README.md` — rename `/fab-hydrate-specs` → `/docs-hydrate-specs`, `/fab-hydrate` → `/docs-hydrate-memory`
- [x] T020 [P] Update `fab/specs/skills.md` — rename `fab-hydrate-specs` references to `docs-hydrate-specs`
- [x] T021 [P] Update `fab/specs/glossary.md` — rename `fab-hydrate-specs` and remove `fab-status.sh` references
- [x] T022 [P] Update `fab/specs/architecture.md` — remove `fab-status.sh` references
- [x] T023 [P] Update `fab/specs/overview.md` — remove `fab-status.sh` references
- [x] T024 [P] Update `fab/specs/user-flow.md` — rename `fab-hydrate-specs` → `docs-hydrate-specs`
- [x] T025 [P] Update `.claude/agents/fab-init.md` — rename `/fab-hydrate` → `/docs-hydrate-memory`
- [x] T026 [P] Update `fab/memory/fab-workflow/hydrate-specs.md` — rename `fab-hydrate-specs` references to `docs-hydrate-specs`
- [x] T027 [P] Update `fab/memory/fab-workflow/specs-index.md` — rename `fab-hydrate-specs` references to `docs-hydrate-specs`

---

## Execution Order

- T001-T004 are independent (Phase 1)
- T005 depends on T004 (mirrors the renamed docs-reorg-specs)
- T006 depends on T001-T004 (removes stale symlinks for old names)
- T007 depends on T006 (scaffold creates new symlinks after stale ones removed)
- T008-T027 are all independent [P] tasks, depend on T001-T004 (need to know final names)
