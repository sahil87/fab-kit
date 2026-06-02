# Tasks: Collapse Tasks Stage into Apply; Replace tasks.md + checklist.md with plan.md

**Change**: 260423-qszh-merge-tasks-checklist
**Spec**: `spec.md`
**Intake**: `intake.md`

<!--
  Phases:
    1. Setup — new template, new migration scaffold, new Go function shells
    2. Core — Go binary surgery (StageOrder, allowed-states, set-acceptance, hooklib, scoring), template/migration content
    3. Skills — _generation/_review/_preamble/_cli-fab/fab-continue/fab-ff/fab-fff/fab-clarify/fab-status text changes
    4. Tests + docs/specs — update Go tests + docs/specs/{overview,skills,templates,user-flow,architecture,glossary} + per-skill SPEC files
    5. Polish — cross-file consistency sweep, build/run full test suite, manual smoke

  Important: this change modifies the very pipeline that runs the change. The implementing agent
  is using the OLD 8-stage skill text (tasks stage exists, checklist.md is generated, etc.).
  All edits land on the source files; the running pipeline state catches up via migration on
  the next run. Do NOT delete legacy tasks.md/checklist.md from this change folder during this
  apply pass — let the agent's own checklist + tasks files persist (they document the work in
  progress on the old schema) and let the migration handle them on the next change.
-->

## Phase 1: Setup

- [x] T001 [P] Create `src/kit/templates/plan.md` template with `## Tasks` (with example phases) and `## Acceptance` (with example categories) heading-keyed sections; include guidance comments matching existing template style; preserve `[P]` marker docs
- [x] T002 [P] Update `src/kit/templates/status.yaml` template: drop `tasks: pending` from `progress`; replace `checklist:` block with `plan:` block (`generated: false`, `task_count: 0`, `acceptance_count: 0`, `acceptance_completed: 0`); remove `path:` field
- [x] T003 [P] Create migration file `src/kit/migrations/1.8.0-to-1.9.0.md` (Summary, Pre-check, Changes, Verification sections per migrations.md format) — flesh out content in T043 <!-- clarified: migration content finalized in T043, not T015; T015 is preflight YAML output -->
- [x] T003a [P] Delete legacy templates `src/kit/templates/tasks.md` and `src/kit/templates/checklist.md` (kit SHALL NOT ship these per spec §Remove legacy templates) <!-- clarified: spec mandates the kit no longer ship tasks.md/checklist.md templates; the original tasks.md added plan.md (T001) and updated status.yaml (T002) but did not explicitly delete the two legacy template files -->


## Phase 2: Core Implementation — fab-go binary

- [x] T004 Update `src/go/fab/internal/statusfile/statusfile.go`: remove `"tasks"` from `StageOrder`. (T001/T002 not blocking — Go work is independent.)
- [x] T005 Update `src/go/fab/internal/statusfile/statusfile.go`: replace `Checklist` struct + field with `Plan` struct (`Generated bool`, `TaskCount int`, `AcceptanceCount int`, `AcceptanceCompleted int`) and YAML tags (`generated`, `task_count`, `acceptance_count`, `acceptance_completed`); update `Load()` switch case from `"checklist"` to `"plan"`; update `syncToRaw` accordingly; remove `path` from struct entirely
- [x] T006 Update `src/go/fab/internal/status/status.go`: remove `"tasks"` key from `allowedStates` map; ensure `isValidStage("tasks")` returns false
- [x] T007 Update `src/go/fab/internal/status/status.go`: replace `SetChecklist` function with `SetAcceptance(statusFile, statusPath, field, value)` — supports fields `generated`/`task_count`/`acceptance_count`/`acceptance_completed`; same validation pattern as old SetChecklist
- [x] T008 Update `src/go/fab/internal/status/status.go`: add a stub `SetChecklist` (or new helper) that returns the strict-error message `"set-checklist" is now "set-acceptance" — run fab status set-acceptance instead.` for the Cobra layer to surface (T013 wires the Cobra command)
- [x] T009 Update `src/go/fab/internal/status/status.go`: in `Start`/`Advance`/`Finish`/`Reset`/`Skip`/`Fail`, change the early `Invalid stage` error path so that `stage == "tasks"` returns the dedicated message `"tasks" stage was removed — run "fab status <event> <change> apply" instead. plan.md is now generated at apply entry.` (rather than the generic `Invalid stage 'tasks'`)
- [x] T010 Update `src/go/fab/internal/change/change.go` `defaultCommand`: drop `"tasks"` from the case list (becomes `case "intake", "spec", "apply", "review":`)
- [x] T011 Update `src/go/fab/internal/score/score.go`: verify `expectedMinSpec` thresholds are unchanged (no tasks reference present); add a test or assertion if `tasks.md`/`checklist.md`/`plan.md` are referenced (none should be)
- [x] T012 Update `src/go/fab/internal/hooklib/artifact.go` `MatchArtifactPath`: change the known-artifact switch from `intake.md, spec.md, tasks.md, checklist.md` to `intake.md, spec.md, plan.md`
- [x] T013 Update `src/go/fab/cmd/fab/hook.go` `artifactBookkeeping`: replace the `tasks.md` and `checklist.md` cases with a single `plan.md` case that (a) reads the file, (b) parses tasks count between `## Tasks` and the next `^##\s` heading, (c) parses acceptance count between `## Acceptance` and the next `^##\s` heading or EOF, (d) calls `SetAcceptance(generated=true)`, `SetAcceptance(task_count=<N>)`, `SetAcceptance(acceptance_count=<M>)`, `SetAcceptance(acceptance_completed=<count of [x] under ## Acceptance>)`. **Defensive behavior**: if `plan.md` lacks the `## Tasks` heading, do NOT call `SetAcceptance(task_count, ...)` (avoid overwriting valid values with zero on a malformed in-progress write); same for missing `## Acceptance` heading and the two acceptance fields. Always set `generated=true` if the file exists with at least the `## Tasks` heading. Add helper functions in `hooklib/artifact.go` (e.g., `CountSectionItemsBounded`, `CountCompletedSectionItemsBounded`) following existing `CountUncheckedTasks`/`CountChecklistItems` style. <!-- clarified: defensive parsing rules per spec §hooklib MatchArtifactPath; on every write task_count and acceptance_count are recomputed (stable property), and acceptance_completed is recomputed too so review's mark-in-place flows into .status.yaml -->
- [x] T014 Update `src/go/fab/cmd/fab/status.go`: replace `statusSetChecklistCmd()` registration with `statusSetAcceptanceCmd()` (Use: `set-acceptance <change> <field> <value>`, calls `status.SetAcceptance`); add a separate `statusSetChecklistRemovedCmd()` (Use: `set-checklist`, returns the strict-error message via `cmd.SilenceUsage = true; return fmt.Errorf(...)`) so users get the pointer message; both registered in `statusCmd()`
- [x] T015 Update `src/go/fab/internal/preflight/preflight.go`: change YAML output to emit `plan:` block with the four fields (replace the old `checklist:` emission); ensure the `progress:` map only emits 7 keys (intake, spec, apply, review, hydrate, ship, review-pr) — derived from `StageOrder` so this is automatic

## Phase 3: Skills text updates

- [x] T016 Update `src/kit/skills/_generation.md`: remove **Tasks Generation Procedure** and **Checklist Generation Procedure**; add a new **Plan Generation Procedure** that walks `spec.md` once and emits Task entries (under `## Tasks` with phased subheadings) + Acceptance entries (under `## Acceptance` with category subheadings, IDs `A-NNN`); document the optional cross-linking note
- [x] T017 Update `src/kit/skills/_review.md`: change Preconditions from "tasks.md and checklist.md MUST exist" to "plan.md MUST exist with both ## Tasks and ## Acceptance sections populated"; change inward sub-agent step 2 to inspect `plan.md ## Acceptance` items in place; keep three-tier severity scheme + Findings Merge unchanged
- [x] T018 Update `src/kit/skills/fab-continue.md`: dispatch table — remove `spec ready → tasks` row, remove both `tasks ready` and `tasks active` rows, change spec-ready dispatch to `finish spec → start apply → execute apply (which generates plan.md then runs tasks)`; add an Apply Behavior subsection describing the plan-generation entry sub-step and the resumability skip; update Preconditions for apply (remove `tasks.md MUST exist`, add `spec.md MUST exist` if not already present); change reset target list to drop `tasks` and add the strict-error path with the exact message `"tasks" stage was removed — use /fab-continue apply (regenerates plan.md and re-runs) or /fab-clarify spec.` (per spec scenario "Reset to tasks errors"); rewrite Hydrate Behavior preconditions to read "plan.md ## Acceptance items all [x]"; replace any `set-checklist` invocations in Review Behavior with `set-acceptance`; update Error Handling rows accordingly (drop tasks.md/checklist.md missing rows, add plan.md missing rows) <!-- clarified: exact reset-error message text quoted from spec; set-acceptance replaces set-checklist in Review Behavior status calls -->
- [x] T019 Update `src/kit/skills/fab-ff.md`: remove old Step 2 (Generate tasks.md) and Step 3 (Generate Quality Checklist); renumber so spec gen is Step 1, implementation (apply) is Step 2, review Step 3, hydrate Step 4; remove auto-clarify on tasks; ensure auto-clarify still runs after spec generation; update header narrative + Output template
- [x] T020 Update `src/kit/skills/fab-fff.md`: same changes as T019 plus extend through Step 5 (ship via /git-pr) and Step 6 (review-pr via /git-pr-review); update header narrative + Output template
- [x] T021 Update `src/kit/skills/fab-clarify.md`: change `<target-artifact>` valid values from `intake|spec|tasks` to `intake|spec|plan`; add tasks→error mapping; update Pre-flight & Stage Guard so post-planning targets accept `plan` (when `plan.md` exists at apply or later); update taxonomy scan categories for `plan` target (task completeness/granularity/dependencies/file paths/[P] markers + acceptance coverage of spec requirements); update Suggest Mode Step 7 confidence recompute trigger condition to remain spec-based
- [x] T022 Update `src/kit/skills/_preamble.md`: drop the `tasks` row from State Table; update narrative stage counts from "8 stages" to "7 stages" wherever they appear; verify Section 2 § Common fab Commands references stay valid
- [x] T023 Update `src/kit/skills/_cli-fab.md`: update `Side effects of finish` line from `intake→spec, spec→tasks, tasks→apply, apply→review, …` to `intake→spec, spec→apply, apply→review, review→hydrate, hydrate→ship, ship→review-pr`; rename `set-checklist` row to `set-acceptance` in the status subcommands table with updated field list; add a row for the now-removed `set-checklist` showing the strict-error stance
- [x] T024 Update `src/kit/skills/fab-status.md`: change description and prose from "checklist counts" to "plan: tasks/acceptance counts"; update progress-table row count expectations from 8 stages to 7; update default fallback "checklist not yet generated" → "plan not yet generated"
- [x] T025 Update `src/kit/skills/fab-proceed.md` and `src/kit/skills/fab-discuss.md`: remove any "tasks artifact" references; ensure their dispatch tables reflect the 7-stage pipeline (only minor copy edits expected)
- [x] T026 Update `src/kit/skills/git-pr.md`: change Step 2 logic — replace "Check if `tasks.md` exists → `{has_tasks}`" with "Check if `plan.md` exists → `{has_plan}`"; replace `.status.yaml` reads from `checklist` → `plan`; rename Stats table column "Checklist" → "Acceptance" (display `{plan.acceptance_completed}/{plan.acceptance_count}`); rename Stats column "Tasks" derivation: parse `plan.md` `## Tasks` checkbox counts; update Pipeline progress line stage list to 7 stages
- [x] T027 Update `src/kit/skills/fab-operator.md`: change the inline pipeline diagram `intake → spec → tasks → apply → review → hydrate → ship` to the 7-stage form (remove `tasks`)
- [x] T028 Update `src/kit/skills/fab-setup.md`: remove `tasks` from any stage-name lists; update `checklist` config-section help text to clarify it now configures plan-acceptance categories (or rename internally to `plan` if clean — but renaming the config key is out of scope; just update help text)

## Phase 4: Tests + Docs/Specs

- [x] T029 Update Go test fixtures: `src/go/fab/internal/{change,preflight,score,statusfile,status,hooklib}/` test files — remove `tasks: pending` rows from inline YAML fixtures; replace any `checklist:` blocks with `plan:` blocks; update expectations that assert 8-element StageOrder or `tasks` allowed states
- [x] T030 Add Go tests: `src/go/fab/internal/status/status_test.go` (or near) — verify `Start/Finish/etc.(stage="tasks")` returns the strict-error message; `src/go/fab/internal/hooklib/artifact_test.go` — `MatchArtifactPath` recognizes `plan.md`, no longer recognizes `tasks.md`/`checklist.md`; `src/go/fab/internal/statusfile/statusfile_test.go` — Plan struct round-trips through Load/Save preserving fields
- [x] T031 Add Go tests for `SetAcceptance`: valid fields update correctly, invalid field returns descriptive error, atomic write refreshes `last_updated`
- [x] T032 Add a CLI integration test (or equivalent unit test scoped to status.go Cobra wiring) that verifies `fab status set-checklist` exits 1 with stderr containing `"set-checklist" is now "set-acceptance"` (exact pointer message text per spec); add a parallel test that `fab status finish <change> tasks` exits 1 with stderr containing `"tasks" stage was removed` <!-- clarified: assert exact stderr substrings to lock down the strict-error contract -->
- [x] T033 [P] Update `docs/specs/overview.md`: change "6 Stages" / "8 Stages" wording to 7 stages; update stage list table; update mermaid diagram (remove the `T["3 TASKS"]` node and rewire `S → A`); update stage details table (drop the tasks row, update apply row to mention plan.md generation)
- [x] T034 [P] Update `docs/specs/skills.md`: per-skill flow updates (`fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify`) reflecting new pipeline; remove tasks-stage flow descriptions
- [x] T035 [P] Update `docs/specs/templates.md`: replace `tasks.md` and `checklist.md` entries with a single `plan.md` entry describing the merged template
- [x] T036 [P] Update `docs/specs/user-flow.md`: pipeline diagrams updated to 7 stages; remove tasks gate transitions
- [x] T037 [P] Update `docs/specs/architecture.md`: scan for and update any `progress:` map references; update if it lists the 8 keys
- [x] T038 [P] Update `docs/specs/glossary.md`: remove `tasks` stage entry; add `plan.md` entry; ensure `apply` entry mentions plan-generation sub-step
- [x] T039 [P] Update `docs/specs/skills/SPEC-fab-continue.md`, `SPEC-fab-ff.md`, `SPEC-fab-fff.md`, `SPEC-fab-clarify.md`: flow diagrams reflect new 7-stage pipeline (drop tasks node) <!-- rework: SPEC sync per constitution — additionally synced SPEC-fab-status.md, SPEC-git-pr.md, SPEC-preamble.md, SPEC-hooks.md (all had stale tasks.md/checklist.md/set-checklist references) -->

- [x] T044 [rework] SPEC-fab-status.md, SPEC-git-pr.md, SPEC-preamble.md, SPEC-hooks.md synced to skill changes per constitution (Changes to skill files MUST update the corresponding SPEC). Includes: status spec narrative refers to plan progress (tasks + acceptance counts) and 7-stage pipeline; git-pr spec references plan.md (not tasks.md) and Stats columns derive from plan.md ## Tasks + .status.yaml plan; preamble spec change-context reads (intake, spec, plan); hooks spec replaces tasks.md/checklist.md PostToolUse entries with plan.md and renames set-checklist → set-acceptance. <!-- rework: cycle 2 of 3 -->
- [x] T045 [rework] _review.md note updated: rework loop reference points to fab-ff.md Step 3 / fab-fff.md Step 3 (was incorrectly Step 6). <!-- rework: cycle 2 of 3 -->
- [x] T046 [rework] Migration 1.8.0-to-1.9.0.md adds Step 6: prune `stage_directives.tasks: []` from `fab/project/config.yaml`, with idempotency. Verification step 4 added. <!-- rework: cycle 2 of 3 -->
- [x] T047 [rework] statusfile.go Load(): when both `plan:` and `checklist:` coexist, drop the stale `checklist:` key from the raw mapping (`plan:` is authoritative). New helper `dropChecklistRaw`. Test added: `TestPlanAndChecklistCoexistDropsChecklist` in statusfile_test.go. <!-- rework: cycle 2 of 3 -->


## Phase 5: Polish

- [x] T040 Cross-file consistency sweep: `grep -rn "tasks\|checklist" src/ docs/specs/` and audit each remaining hit for staleness; preserve historical changelog mentions and code-comments referring to the literal string `tasks.md` only when migration-relevant
- [x] T041 Build + test: `cd src/go/fab && go build ./... && go test ./...` — all tests pass; iterate on any remaining `tasks: pending` fixture stragglers from T029
- [x] T042 Smoke: `fab status all-stages` returns 7 stages; `fab status finish <some-test-change> tasks` returns the strict-error; `fab status set-checklist <ditto>` returns the pointer message; `fab status set-acceptance <ditto> task_count 5` works
- [x] T043 Final pass on the migration body in `1.8.0-to-1.9.0.md` (started in T003): write out the three-case logic (idempotent no-op / merge / pre-planning no-op), the `.status.yaml` rewrite rules, the legacy file annotation, the archived-folder skip; include the Verification checklist mirroring the spec scenarios

---

## Execution Order

- **T001/T002/T003/T003a** are independent: parallel.
- **Phase 2 (Go binary)**: T004 and T005 are independent (different functions in same file — sequential is fine to avoid merge friction). T006 follows T004 conceptually but is in a different file. T007 → T008 → T009 are sequential within `status.go`. T010, T011, T012 independent. T013 depends on T012 (relies on `MatchArtifactPath` recognizing `plan.md`) and on T007 (uses `SetAcceptance`). T014 depends on T007 and T008. T015 depends on T005.
- **Phase 3 (Skills)**: T016 → T017 (review references generation contract) → T018/T019/T020 (those reference `_generation.md`'s new procedure name). T021–T028 are independent of each other but reference Phase 1+2 outputs.
- **Phase 4 (Tests + Docs)**: T029 depends on Phase 2 binary changes complete; T030–T032 follow T029. T033–T039 are all `[P]` — independent docs work on different files.
- **Phase 5**: T040 sweep runs after all skill/code edits; T041 build runs after T040; T042 smoke after T041; T043 finalizes the migration content (the scaffold in T003 is fleshed out here using all the spec details). <!-- clarified: T003 scaffolds the migration; T043 fleshes out the body. Earlier task description in T003 mistakenly referenced T015 (preflight) — corrected. -->

<!-- Migrated to plan.md on 2026-05-06 — safe to delete. -->
