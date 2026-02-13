# Tasks: Split Archive into Hydrate Stage and fab-archive Command

**Change**: 260213-jc0u-split-archive-hydrate
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Configuration & Templates

- [x] T001 [P] Update `fab/.kit/templates/status.yaml` — replace `archive: pending` with `hydrate: pending` in the progress map
- [x] T002 [P] Update `fab/.kit/schemas/workflow.yaml` — replace the `archive` stage with `hydrate` (id, name, description, commands), update `progression.fallback` and `completion.rule` to reference `hydrate`, update `stage_numbers` to `hydrate: 6`
- [x] T003 [P] Update `fab/config.yaml` — replace the `archive` stage entry (`id: archive`) with `id: hydrate` in the `stages` list

## Phase 2: Core Skill Files

- [x] T004 Rewrite Archive Behavior → Hydrate Behavior in `fab/.kit/skills/fab-continue.md` — replace the entire `## Archive Behavior` section with `## Hydrate Behavior` containing only steps 1-4 (final validation, concurrent check, doc hydration, status update). Remove folder move, archive index, backlog marking, and pointer clearing. Update all stage guard tables, transition tables, context loading, output templates, reset flow targets, error handling, and key properties to reference `hydrate` instead of `archive`. Update Next Steps after hydrate to `Next: /fab-archive`
- [x] T005 [P] Update `fab/.kit/skills/fab-ff.md` — replace archive as terminal step with hydrate. Update all references to archive stage/behavior. Change terminal output to suggest `/fab-archive` as next step
- [x] T006 [P] Update `fab/.kit/skills/fab-fff.md` — replace archive as terminal step with hydrate. Update all references to archive stage/behavior. Change terminal output to suggest `/fab-archive` as next step
- [x] T007 Create `fab/.kit/skills/fab-archive.md` — new standalone skill file. Includes: preflight check, `hydrate: done` guard, folder move, archive index update, backlog marking (exact-ID + keyword scan with interactive confirmation), conditional `fab/current` clearing (only for active change), fail-safe ordering, resumability for interrupted operations, `[change-name]` argument support

## Phase 3: Shared Context & Supporting Files

- [x] T008 Update `fab/.kit/skills/_context.md` — replace archive entries in the Next Steps Lookup Table with hydrate entries, add `/fab-archive` entry (`Next: /fab-archive`). Update any inline references to archive stage
- [x] T009 [P] Update `fab/.kit/skills/_generation.md` — replace references to "archive" (e.g., "All items MUST pass before /fab-continue (archive)") with "hydrate"
- [x] T010 [P] Audit and update `fab/.kit/scripts/stageman.sh` — search for `archive` references and replace with `hydrate` where they refer to the pipeline stage
- [x] T011 [P] Audit and update `fab/.kit/scripts/fab-preflight.sh` — search for `archive` references and replace with `hydrate` where they refer to the pipeline stage
- [x] T012 [P] Audit and update `fab/.kit/scripts/fab-status.sh` — search for `archive` references and replace with `hydrate` where they refer to the pipeline stage

## Phase 4: Peripheral Skills & Polish

- [x] T013 [P] Audit and update `fab/.kit/skills/fab-status.md` — replace any references to the archive stage with hydrate in stage lists, display examples, and output templates
- [x] T014 [P] Audit and update `fab/.kit/skills/fab-help.md` — replace any references to the archive stage with hydrate in pipeline descriptions and command summaries
- [x] T015 [P] Audit and update `fab/.kit/skills/fab-new.md` — replace any references to the archive stage (e.g., in initial `.status.yaml` generation or stage lists)
- [x] T016 [P] Audit and update `fab/.kit/skills/fab-switch.md` — replace any references to the archive stage (e.g., stage number mapping table)
- [x] T017 [P] Audit and update `fab/.kit/skills/fab-clarify.md` — replace any references to the archive stage in stage guard logic
- [x] T018 Audit and update `fab/.kit/skills/fab-hydrate.md`, `fab/.kit/skills/fab-hydrate-design.md`, `fab/.kit/skills/internal-consistency-check.md`, `fab/.kit/skills/internal-retrospect.md` — search for archive stage references and update to hydrate where applicable
- [x] T019 Audit and update `fab/.kit/scripts/fab-help.sh` — search for archive references and replace with hydrate where they refer to the pipeline stage

---

## Execution Order

- T001, T002, T003 are independent setup tasks (all [P])
- T004 should complete before T005/T006 since fab-ff and fab-fff reference fab-continue behavior patterns
- T005, T006, T007 are independent after T004
- T008 should follow T004 (next steps table references must align with fab-continue changes)
- T009-T012 are independent audit tasks (all [P])
- T013-T019 are independent audit tasks (all [P]), can run alongside Phase 3
