# Tasks: Unify Pipeline Commands into fab-continue

**Change**: 260212-a4bd-unify-fab-continue
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Skill Rewrite

- [x] T001 Rewrite `fab/.kit/skills/fab-continue.md` — extend stage guard table to dispatch all 6 stages (brief→spec→tasks→apply→review→archive), add Apply Behavior section (absorbed from `fab-apply.md`), add Review Behavior section with rework options (absorbed from `fab-review.md`), add Archive Behavior section (absorbed from `fab-archive.md`), extend Reset Flow to accept all 6 stages as targets, update context loading per stage, update output examples and Next Steps references

## Phase 2: Skill Ecosystem Updates

- [x] T002 [P] Delete standalone execution skill files: `fab/.kit/skills/fab-apply.md`, `fab/.kit/skills/fab-review.md`, `fab/.kit/skills/fab-archive.md`
- [x] T003 [P] Update `fab/.kit/skills/fab-ff.md` — replace all `/fab-apply`, `/fab-review`, `/fab-archive` invocable command references with `fab-continue` equivalents. Update Steps 6-8 headings/descriptions, review rework options, error handling table, and comparison table
- [x] T004 [P] Update `fab/.kit/skills/fab-fff.md` — replace all `/fab-apply`, `/fab-review`, `/fab-archive` references with `fab-continue` equivalents. Update Steps 2-4 descriptions, error handling table, and comparison table
- [x] T005 [P] Update `fab/.kit/skills/_context.md` — rewrite the Next Steps Convention lookup table per spec (remove `/fab-apply`, `/fab-review`, `/fab-archive` rows; add `/fab-continue` → apply/review/archive rows)
- [x] T006 [P] Update remaining skill files with reference fixes: `fab/.kit/skills/fab-new.md`, `fab/.kit/skills/fab-clarify.md`, `fab/.kit/skills/fab-status.md`, `fab/.kit/skills/fab-switch.md`, `fab/.kit/skills/fab-init.md`, `fab/.kit/skills/fab-hydrate-design.md` — replace all `/fab-apply`, `/fab-review`, `/fab-archive` references with `/fab-continue`
- [x] T007 [P] Update `fab/.kit/templates/checklist.md` — replace `/fab-review` and `/fab-archive` references with `/fab-continue`

## Phase 3: Docs & Design Updates

- [x] T008 [P] Update `fab/docs/fab-workflow/planning-skills.md` — expand `/fab-continue` section to document execution stage behavior (apply, review, archive). Update stage guard, reset behavior, context loading. Update changelog
- [x] T009 [P] Update `fab/docs/fab-workflow/execution-skills.md` — restructure to note behavior is now accessed via `/fab-continue`. Remove standalone skill sections, replace with references to unified command. Update changelog
- [x] T010 [P] Update `fab/docs/fab-workflow/change-lifecycle.md` — replace all `/fab-apply`, `/fab-review`, `/fab-archive` command references with `/fab-continue`. Update `fab/current` lifecycle, stage transitions, and `/fab-switch` sections. Update changelog
- [x] T011 [P] Update remaining `fab/docs/` files with reference fixes: `fab/docs/fab-workflow/clarify.md`, `fab/docs/fab-workflow/configuration.md`, `fab/docs/fab-workflow/templates.md`, `fab/docs/fab-workflow/index.md`, `fab/docs/index.md`
- [x] T012 [P] Update design specs with reference fixes: `fab/design/skills.md`, `fab/design/user-flow.md`, `fab/design/overview.md`, `fab/design/architecture.md`, `fab/design/glossary.md`, `fab/design/templates.md`, `fab/design/index.md`
- [x] T013 Update `README.md` — replace all `/fab-apply`, `/fab-review`, `/fab-archive` command references with `/fab-continue`

## Phase 4: Verification

- [x] T014 Search codebase for remaining dangling references to `/fab-apply`, `/fab-review`, `/fab-archive` outside of `fab/changes/archive/` and changelog tables. Fix any found

---

## Execution Order

- T001 blocks all other tasks (core rewrite establishes the new behavior that others reference)
- T002-T007 are parallelizable (independent file updates in Phase 2)
- T008-T013 are parallelizable (independent file updates in Phase 3)
- T014 must run after all other tasks complete (verification sweep)
