# Tasks: Simplify Stages and Entry Paths

**Change**: 260212-v5p2-simplify-stages-entry-paths
**Spec**: `spec.md`
**Brief**: `brief.md`

<!--
  TASK FORMAT: - [ ] {ID} [{markers}] {Description with file paths}

  Markers (optional, combine as needed):
    [P]   — Parallelizable (different files, no dependencies on other [P] tasks in same group)

  IDs are sequential: T001, T002, ...
  Include exact file paths in descriptions.
  Each task should be completable in one focused session.

  Tasks are grouped by phase. Phases execute sequentially.
  Within a phase, [P] tasks can execute in parallel.
-->

## Phase 1: Setup

<!-- Templates and config — foundation changes that later tasks reference. -->

- [x] T001 [P] Update status template `fab/.kit/templates/status.yaml`: remove `stage:` field, remove `brief:` from progress map, set `spec: active` as first entry, all others `pending`
- [x] T002 [P] Add Origin section to brief template `fab/.kit/templates/brief.md`: new `## Origin` section between metadata and Why, captures user's raw input/prompt
- [x] T003 [P] Update `fab/config.yaml` stages section: remove `brief` entry, remove `brief` from `spec.requires` (spec becomes first stage with no prerequisites)

## Phase 2: Core Implementation

<!-- Shell scripts first (shared infrastructure), then skill definitions. -->

- [x] T004 Update `fab/.kit/scripts/fab-preflight.sh`: derive `stage:` output from the `active` entry in progress map instead of reading `stage:` field; remove `brief` progress extraction (`p_brief`); output only `spec`, `tasks`, `apply`, `review`, `archive` progress fields
- [x] T005 Update `fab/.kit/scripts/fab-status.sh`: derive current stage from `active` progress entry instead of `stage:` field; change stage numbering to spec=1 through archive=5 (total `/5`); remove `brief` progress display line
- [x] T006 [P] Update `fab/.kit/scripts/fab-help.sh`: change "Planning stages: brief → spec → tasks" to "Planning stages: spec → tasks"; remove any `/fab-discuss` references from skill catalog
- [x] T007 [P] Rewrite `fab/.kit/skills/fab-new.md`: replace fixed 3-question cap with SRAD-driven questioning; add gap analysis before folder creation; add conversational mode for vague inputs (low Signal Strength); ensure output is brief-only (no spec generation); add Origin section population in brief; update output format and next-steps line
- [x] T008 Update `fab/.kit/skills/fab-continue.md`: replace `stage:` field checks with `active` entry lookup in progress map; update action table (spec/tasks/apply/review/archive); change reset-to-brief rejection message to "Cannot reset to brief — brief is not a pipeline stage. Edit brief.md directly or use /fab-clarify."; update stage numbering references to 5-stage model; update `.status.yaml` examples to omit `stage:` and `brief:`
- [x] T009 [P] Update `fab/.kit/skills/fab-clarify.md`: remove `brief` from stage guard valid list; add brief refinement when current stage is `spec` — scan both `brief.md` (brief taxonomy) and `spec.md` (spec taxonomy) using per-artifact categories; update context loading for spec stage to include `brief.md`
- [x] T010 [P] Update `fab/.kit/skills/fab-ff.md`: change pipeline from 3-stage (brief→spec→tasks) to 2-stage (spec→tasks); start from `active` entry; update stage counting and output
- [x] T011 [P] Update `fab/.kit/skills/fab-fff.md`: replace brief-stage-done gate with brief-exists check (`brief.md` must exist in change folder); keep `confidence.score >= 3.0` gate
- [x] T012 [P] Update `fab/.kit/skills/fab-switch.md`: change stage numbering from 6-stage to 5-stage (spec=1, tasks=2, apply=3, review=4, archive=5); remove brief from stage number mapping table
- [x] T013 Update `fab/.kit/skills/_context.md`: remove all `/fab-discuss` rows from Next Steps Lookup Table; update `/fab-init` next line to exclude `/fab-discuss`; update `/fab-hydrate` next line; update pipeline stage references from 6 to 5

## Phase 3: Integration & Edge Cases

<!-- Delete fab-discuss, update centralized docs. All doc updates are parallel. -->

- [x] T014 Delete `/fab-discuss` skill and symlinks: remove `fab/.kit/skills/fab-discuss.md`; remove `.claude/skills/fab-discuss/` directory; remove `.opencode/commands/fab-discuss.md` (if exists); remove `.agents/skills/fab-discuss/` (if exists); check `fab/.kit/scripts/fab-setup.sh` — if discuss is hardcoded, remove it; otherwise deletion of source file is sufficient
- [x] T015 [P] Update `fab/docs/fab-workflow/planning-skills.md`: remove `/fab-discuss` section entirely; rewrite `/fab-new` section to describe adaptive SRAD-driven behavior, gap analysis, conversational mode, Origin section, brief-only output
- [x] T016 [P] Update `fab/docs/fab-workflow/change-lifecycle.md`: change pipeline from 6 to 5 stages; remove brief as a stage; update state machine description (remove `stage:` field, document `active` marker as single source of truth, state vocabulary, two-write transitions, review failure backward movement); add migration note for old-format `.status.yaml` files
- [x] T017 [P] Update `fab/docs/fab-workflow/clarify.md`: add brief refinement capability documentation (scanning both `brief.md` and `spec.md` at spec stage with per-artifact taxonomy); remove `brief` from valid stages; update examples
- [x] T018 [P] Update `fab/docs/fab-workflow/configuration.md`: update stages pipeline documentation to 5 stages; remove brief entry from documented config example
- [x] T019 [P] Update `fab/docs/fab-workflow/templates.md`: update `.status.yaml` template documentation (no `stage:` field, no `brief:` progress, `spec: active` as initial); document brief template Origin section addition
- [x] T020 [P] Update `fab/docs/fab-workflow/context-loading.md`: remove `/fab-discuss` references from next-steps and exception skills list
- [x] T021 [P] Update `fab/docs/fab-workflow/kit-architecture.md`: remove `fab-discuss.md` from skill catalog listing; verify skill count

## Phase 4: Polish

<!-- Design docs and cross-reference sweep. -->

- [x] T022 Update design docs in `fab/design/`: update stage counts from 6 to 5, flow diagrams, and skill references in affected files (glossary.md, overview.md, skills.md, user-flow.md, templates.md, architecture.md); remove `/fab-discuss` references; preserve changelog entries as historical record

---

## Execution Order

- Phase 1 (T001-T003) all parallel — no internal dependencies
- T004 (preflight) should precede T005 (status) — same stage-derivation pattern, preflight is canonical
- T006-T007, T009-T012 are parallel within Phase 2
- T008 (fab-continue) after T004 — references same active-lookup approach
- T013 (_context.md) after T007 — fab-new next-steps line must be finalized
- T014 (delete discuss) before T015-T021 — ensures no stale references during doc updates
- T015-T021 all parallel
- T022 last — references all prior changes for consistency
