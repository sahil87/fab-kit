# Tasks: Rename "brief" to "intake" + Add Intake Generation Rule

**Change**: 260215-v4n7-DEV-1025-rename-brief-to-intake
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

<!-- Foundation: template, schema, config, status template. These are the source-of-truth files that other files reference. -->

- [x] T001 Rename `fab/.kit/templates/brief.md` → `fab/.kit/templates/intake.md` and strengthen structural cues in HTML comments (What Changes, Origin, Why sections)
- [x] T002 [P] Update `fab/.kit/schemas/workflow.yaml` — replace `brief` with `intake` in stage enum and all references (4 occurrences)
- [x] T003 [P] Update `fab/.kit/templates/status.yaml` — replace `brief:` with `intake:` in progress map (1 occurrence)
- [x] T004 [P] Update `fab/config.yaml` — replace `id: brief` → `id: intake`, `generates: brief.md` → `generates: intake.md`, `requires: [brief]` → `requires: [intake]` (3 occurrences)

## Phase 2: Core Implementation

<!-- Skill files and generation procedure. Order: shared partials first, then individual skills. -->

- [x] T005 Add Intake Generation Procedure to `fab/.kit/skills/_generation.md` (new section before Spec Generation Procedure) and update existing `brief.md`/`brief` references in Spec and Tasks procedures (7 occurrences total)
- [x] T006 Update `fab/.kit/skills/_context.md` — replace `brief` references in context loading layers, memory file lookup instructions (3 occurrences)
- [x] T007 Update `fab/.kit/skills/fab-new.md` — artifact generation target, status initialization, Step 5 to reference Intake Generation Procedure (6 occurrences)
- [x] T008 Update `fab/.kit/skills/fab-continue.md` — stage dispatch table, context loading, transition calls (5 occurrences)
- [x] T009 [P] Update `fab/.kit/skills/fab-clarify.md` — taxonomy scan references, stage guard, artifact references (7 occurrences)
- [x] T010 [P] Update `fab/.kit/skills/fab-ff.md` — stage references, pipeline flow, pre-flight check (3 occurrences)
- [x] T011 [P] Update `fab/.kit/skills/fab-fff.md` — stage references (2 occurrences)
- [x] T012 [P] Update `fab/.kit/skills/fab-archive.md` — artifact references (5 occurrences)
- [x] T013 [P] Update `fab/.kit/skills/fab-init.md` — valid sections, bootstrap references (3 occurrences)
- [x] T014 [P] Update `fab/.kit/skills/fab-switch.md` — stage display references (2 occurrences)
- [x] T015 [P] Update remaining skill files: `docs-reorg-specs.md` (1), `docs-reorg-memory.md` (1), `docs-hydrate-specs.md` (1), `internal-skill-optimize.md` (2) — no changes needed (all occurrences are English adjective "brief", not the stage name)
- [x] T016 [P] Update `fab/.kit/templates/spec.md` and `fab/.kit/templates/tasks.md` — replace `brief.md` cross-references (3 occurrences total)

## Phase 3: Integration — Documentation

<!-- Specs and memory files. These are documentation updates, not behavioral. -->

- [x] T017 [P] Update `docs/specs/` files: `skills.md` (22), `architecture.md` (9), `glossary.md` (7), `user-flow.md` (6), `templates.md` (5), `overview.md` (3), `srad.md` (2), `index.md` (1) — replace `brief` stage/artifact references, preserve English adjective "brief" (55 occurrences across 8 files)
- [x] T018 [P] Update `docs/memory/fab-workflow/` files: `planning-skills.md` (26), `templates.md` (12), `change-lifecycle.md` (11), `configuration.md` (5), `kit-architecture.md` (3), `clarify.md` (3), `execution-skills.md` (2), `index.md` (2), `context-loading.md` (1), `hydrate-generate.md` (1) — replace `brief` stage/artifact references, preserve English adjective "brief" (66 occurrences across 10 files)

---

## Execution Order

- T001–T004 (Phase 1) before Phase 2 — templates and schema define the canonical names
- T005–T006 before T007–T008 — shared partials (`_generation.md`, `_context.md`) before individual skills that reference them
- T007–T008 are sequential (fab-new and fab-continue are the primary pipeline drivers)
- T009–T016 are parallelizable (independent skill files and templates)
- T017–T018 are parallelizable and independent of Phase 2
