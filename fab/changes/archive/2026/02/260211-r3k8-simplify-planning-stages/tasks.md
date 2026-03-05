# Tasks: Simplify Planning Stages

**Change**: 260211-r3k8-simplify-planning-stages
**Spec**: `spec.md`
**Brief**: `proposal.md`

## Phase 1: Setup — Config, Templates, Directory Rename

- [ ] T001 Rename `fab/.kit/templates/proposal.md` → `fab/.kit/templates/brief.md` (keep content identical)
- [ ] T002 Delete `fab/.kit/templates/plan.md`
- [ ] T003 Update `fab/.kit/templates/tasks.md` — change `**Proposal**: \`proposal.md\`` reference to `**Brief**: \`brief.md\``; remove `**Plan**: \`plan.md\`` line
- [ ] T004 Update `fab/.kit/templates/status.yaml` — replace progress keys `proposal`/`specs`/`plan` with `brief`/`spec`, remove `plan` key entirely. Update stage field default
- [ ] T005 Update `fab/config.yaml` — rename stage IDs (`proposal` → `brief`, `specs` → `spec`), remove `plan` stage entry, update `requires` chains (`spec` requires `[brief]`, `tasks` requires `[spec]`)
- [ ] T006 Rename directory `fab/specs/` → `fab/design/` — update `fab/design/index.md` terminology from "specs" to "design"
- [ ] T007 Update `fab/constitution.md` — replace `fab/specs/` references with `fab/design/`

## Phase 2: Core — Shared Partials and Skill Files

- [ ] T008 Update `fab/.kit/skills/_context.md` — Context Loading: `fab/specs/index.md` → `fab/design/index.md`; Next Steps table: `proposal` → `brief`, `specs` → `spec`, remove plan rows; SRAD skill table: update stage names; Confidence Scoring lifecycle table: update stage names
- [ ] T009 Update `fab/.kit/skills/_generation.md` — remove Plan Generation Procedure section entirely; update Spec Generation Procedure to include optional `## Design Decisions` section guidance
- [ ] T010 Update `fab/.kit/skills/fab-new.md` — `proposal` → `brief` throughout, artifact output `proposal.md` → `brief.md`, stage references, status.yaml template example
- [ ] T011 Update `fab/.kit/skills/fab-discuss.md` — `proposal` → `brief` throughout; new change mode: produce both `brief.md` + `spec.md`, mark both stages done, set stage to `spec`; update `.status.yaml` template example; update key differences table
- [ ] T012 Update `fab/.kit/skills/fab-continue.md` — `proposal` → `brief`, `specs` → `spec` throughout; remove plan stage logic (plan-skip decision, plan generation, specs→plan transition); update stage progression graph; update stage guard logic; update reset targets; update context loading sections; update output examples; update stage transition table
- [ ] T013 Update `fab/.kit/skills/fab-ff.md` — `proposal` → `brief`, `specs` → `spec` throughout; remove plan generation step and plan-decision logic; update pipeline: `spec → auto-clarify → tasks → auto-clarify`; update output examples
- [ ] T014 [P] Update `fab/.kit/skills/fab-fff.md` — `proposal` → `brief` in precondition check (`progress.brief`); update stage references
- [ ] T015 [P] Update `fab/.kit/skills/fab-clarify.md` — `proposal` → `brief`, `specs` → `spec` throughout; remove `plan` from stage guard list and context loading sections; update stage-scoped taxonomy categories (rename "Proposal categories" to "Brief categories"); update artifact file mapping table
- [ ] T016 [P] Update `fab/.kit/skills/fab-init.md` — `fab/specs/` → `fab/design/`; update stage references if any
- [ ] T017 [P] Update `fab/.kit/skills/fab-switch.md` — update stage number mapping table (6 stages, brief=1, spec=2, tasks=3, apply=4, review=5, archive=6); update suggested next commands table
- [ ] T018 [P] Update `fab/.kit/skills/fab-apply.md` — update stage references (`proposal` → `brief`, `specs` → `spec`, remove plan references)
- [ ] T019 [P] Update `fab/.kit/skills/fab-review.md` — update stage references
- [ ] T020 [P] Update `fab/.kit/skills/fab-archive.md` — update stage references; update hydration to extract Design Decisions from spec (not plan)
- [ ] T021 [P] Update `fab/.kit/skills/fab-backfill.md` — `fab/specs/` → `fab/design/`
- [ ] T022 [P] Update `fab/.kit/skills/retrospect.md` — `fab/specs/` → `fab/design/` if referenced
- [ ] T023 [P] Update `fab/.kit/skills/fab-help.md` — update stage references if any

## Phase 3: Shell Scripts

- [ ] T024 Update `fab/.kit/scripts/fab-status.sh` — stage names in progress display, stage numbering (N/6 not N/7), `proposal` → `brief`, `specs` → `spec`, remove `plan` handling
- [ ] T025 [P] Update `fab/.kit/scripts/fab-preflight.sh` — progress key names if hardcoded (`proposal` → `brief`, `specs` → `spec`, remove `plan`)
- [ ] T026 [P] Update `fab/.kit/scripts/fab-help.sh` — stage references in help text
- [ ] T027 [P] Update `fab/.kit/scripts/fab-setup.sh` — `fab/specs/` → `fab/design/` if referenced in directory creation

## Phase 4: Centralized Docs

- [ ] T028 Update `fab/docs/fab-workflow/planning-skills.md` — all stage name references, remove plan-related design decisions and requirements, update `/fab-discuss` entry for dual artifact output
- [ ] T029 [P] Update `fab/docs/fab-workflow/change-lifecycle.md` — stage names in all sections (7→6 stages, progress keys, stage field values, stage graph), remove `plan` from state vocabulary `skipped` usage
- [ ] T030 [P] Update `fab/docs/fab-workflow/configuration.md` — stage IDs in `stages` schema, update `rules` example (remove `plan:` key)
- [ ] T031 [P] Update `fab/docs/fab-workflow/templates.md` — rename `proposal.md` section to `brief.md`, remove `plan.md` section, update spec section to mention optional Design Decisions
- [ ] T032 [P] Update `fab/docs/fab-workflow/kit-architecture.md` — directory structure listing (`proposal.md` → `brief.md`, remove `plan.md`), `fab/specs/` references in Preserved list
- [ ] T033 [P] Rename `fab/docs/fab-workflow/specs-index.md` → `fab/docs/fab-workflow/design-index.md` — update content: `fab/specs/` → `fab/design/`, "specs" terminology → "design"
- [ ] T034 [P] Update `fab/docs/fab-workflow/context-loading.md` — `fab/specs/index.md` → `fab/design/index.md`, update stage name references
- [ ] T035 [P] Update `fab/docs/fab-workflow/clarify.md` — remove `plan` from stage lists, `proposal` → `brief`, `specs` → `spec`
- [ ] T036 [P] Update `fab/docs/fab-workflow/execution-skills.md` — `fab/specs/` → `fab/design/`, stage references
- [ ] T037 [P] Update `fab/docs/fab-workflow/init.md` — `fab/specs/` → `fab/design/`
- [ ] T038 [P] Update `fab/docs/fab-workflow/backfill.md` — `fab/specs/` → `fab/design/`
- [ ] T039 [P] Update `fab/docs/fab-workflow/hydrate.md` — `fab/specs/` → `fab/design/`
- [ ] T040 [P] Update `fab/docs/fab-workflow/distribution.md` — `fab/specs/` → `fab/design/`
- [ ] T041 Update `fab/docs/fab-workflow/index.md` — update `specs-index` entry to `design-index`, update descriptions mentioning stages
- [ ] T042 Update `fab/docs/index.md` — update `specs-index` reference if present in doc list
- [ ] T043 [P] Update `fab/specs/glossary.md` (now `fab/design/glossary.md`) — update stage terminology, remove plan stage entries, rename proposal → brief, specs → spec
- [ ] T044 [P] Update `fab/specs/overview.md` (now `fab/design/overview.md`) — update 7-stage → 6-stage references, mermaid diagrams, stage details table, example workflows, quick reference
- [ ] T045 [P] Update `fab/specs/skills.md` (now `fab/design/skills.md`) — stage references if any
- [ ] T046 [P] Update `fab/specs/templates.md` (now `fab/design/templates.md`) — stage references if any
- [ ] T047 [P] Update `fab/specs/user-flow.md` (now `fab/design/user-flow.md`) — stage references, flow diagrams
- [ ] T048 [P] Update `fab/specs/srad.md` (now `fab/design/srad.md`) — stage references if any
- [ ] T049 [P] Update remaining `fab/design/*.md` files — `fab/specs/` self-references → `fab/design/`

---

## Execution Order

- T001-T007 (Phase 1) are foundational — must complete before Phase 2
- T008-T009 (shared partials) should complete before other Phase 2 skill files
- T010-T013 are the major skill rewrites (sequential, each informed by prior)
- T014-T023 are independent minor skill updates ([P] marked)
- T024-T027 (Phase 3) are independent of each other ([P] where marked)
- T028-T049 (Phase 4) are mostly independent docs updates ([P] where marked), except T041-T042 (indexes) should follow T033 (specs-index rename)
