# Plan: Collapse Tasks Stage into Apply; Replace tasks.md + checklist.md with plan.md

**Change**: 260423-qszh-merge-tasks-checklist
**Status**: In Progress
**Intake**: `intake.md`
**Spec**: `spec.md`

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

## Tasks

### Phase 1: Setup

- [x] T001 [P] Create `src/kit/templates/plan.md` template with `## Tasks` (with example phases) and `## Acceptance` (with example categories) heading-keyed sections; include guidance comments matching existing template style; preserve `[P]` marker docs
- [x] T002 [P] Update `src/kit/templates/status.yaml` template: drop `tasks: pending` from `progress`; replace `checklist:` block with `plan:` block (`generated: false`, `task_count: 0`, `acceptance_count: 0`, `acceptance_completed: 0`); remove `path:` field
- [x] T003 [P] Create migration file `src/kit/migrations/1.8.0-to-1.9.0.md` (Summary, Pre-check, Changes, Verification sections per migrations.md format) — flesh out content in T043 <!-- clarified: migration content finalized in T043, not T015; T015 is preflight YAML output -->
- [x] T003a [P] Delete legacy templates `src/kit/templates/tasks.md` and `src/kit/templates/checklist.md` (kit SHALL NOT ship these per spec §Remove legacy templates) <!-- clarified: spec mandates the kit no longer ship tasks.md/checklist.md templates; the original tasks.md added plan.md (T001) and updated status.yaml (T002) but did not explicitly delete the two legacy template files -->


### Phase 2: Core Implementation — fab-go binary

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

### Phase 3: Skills text updates

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

### Phase 4: Tests + Docs/Specs

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


### Phase 5: Polish

- [x] T040 Cross-file consistency sweep: `grep -rn "tasks\|checklist" src/ docs/specs/` and audit each remaining hit for staleness; preserve historical changelog mentions and code-comments referring to the literal string `tasks.md` only when migration-relevant
- [x] T041 Build + test: `cd src/go/fab && go build ./... && go test ./...` — all tests pass; iterate on any remaining `tasks: pending` fixture stragglers from T029
- [x] T042 Smoke: `fab status all-stages` returns 7 stages; `fab status finish <some-test-change> tasks` returns the strict-error; `fab status set-checklist <ditto>` returns the pointer message; `fab status set-acceptance <ditto> task_count 5` works
- [x] T043 Final pass on the migration body in `1.8.0-to-1.9.0.md` (started in T003): write out the three-case logic (idempotent no-op / merge / pre-planning no-op), the `.status.yaml` rewrite rules, the legacy file annotation, the archived-folder skip; include the Verification checklist mirroring the spec scenarios

## Execution Order

- **T001/T002/T003/T003a** are independent: parallel.
- **Phase 2 (Go binary)**: T004 and T005 are independent (different functions in same file — sequential is fine to avoid merge friction). T006 follows T004 conceptually but is in a different file. T007 → T008 → T009 are sequential within `status.go`. T010, T011, T012 independent. T013 depends on T012 (relies on `MatchArtifactPath` recognizing `plan.md`) and on T007 (uses `SetAcceptance`). T014 depends on T007 and T008. T015 depends on T005.
- **Phase 3 (Skills)**: T016 → T017 (review references generation contract) → T018/T019/T020 (those reference `_generation.md`'s new procedure name). T021–T028 are independent of each other but reference Phase 1+2 outputs.
- **Phase 4 (Tests + Docs)**: T029 depends on Phase 2 binary changes complete; T030–T032 follow T029. T033–T039 are all `[P]` — independent docs work on different files.
- **Phase 5**: T040 sweep runs after all skill/code edits; T041 build runs after T040; T042 smoke after T041; T043 finalizes the migration content (the scaffold in T003 is fleshed out here using all the spec details). <!-- clarified: T003 scaffolds the migration; T043 fleshes out the body. Earlier task description in T003 mistakenly referenced T015 (preflight) — corrected. -->

## Acceptance

### Functional Completeness

- [x] CHK-001 plan.md template: `src/kit/templates/plan.md` exists and contains `## Tasks` and `## Acceptance` heading-keyed sections with example phases/categories and guidance comments
- [x] CHK-002 plan.md is generated for new changes by the apply skill at entry; tasks.md and checklist.md are NOT created for new changes
- [x] CHK-003 plan.md has a stable parser contract: `## Tasks` (apply consumer), `## Acceptance` (review consumer), optional `## Execution Order` and `## Notes`; phase/category subheadings are presentational
- [x] CHK-004 Newly generated plans use `A-001`, `A-002`, ... acceptance IDs (zero-padded to 3 digits)
- [x] CHK-005 `[P]` parallel marker on tasks is preserved/supported per legacy convention
- [x] CHK-006 Legacy templates removed: `src/kit/templates/tasks.md` and `src/kit/templates/checklist.md` no longer exist in the kit
- [x] CHK-007 Pipeline reduces from 8 stages → 7 stages: `intake → spec → apply → review → hydrate → ship → review-pr`
- [x] CHK-008 `progress.tasks` key is removed entirely from `.status.yaml` template (not renamed)
- [x] CHK-009 `checklist:` block in `.status.yaml` template replaced by `plan:` block with fields `generated`, `task_count`, `acceptance_count`, `acceptance_completed`; `path` field removed
- [x] CHK-010 `fab status start|advance|finish|reset|skip|fail <change> tasks` returns exit 1 with the strict-error message including `"tasks" stage was removed` and a pointer to `apply`
- [x] CHK-011 `fab status set-checklist <change> <field> <value>` returns exit 1 with the strict-error message including `"set-checklist" is now "set-acceptance"`
- [x] CHK-012 `fab status set-acceptance <change> <field> <value>` updates the `plan:` block correctly for valid fields (generated, task_count, acceptance_count, acceptance_completed)
- [x] CHK-013 Apply skill: plan generation sub-step runs at entry, before any task execution; sub-step is skipped when `plan.md` already exists (resumability)
- [x] CHK-014 Apply skill: task execution parses only `## Tasks` section; `## Acceptance` is ignored by apply
- [x] CHK-015 Unified Plan Generation Procedure replaces both Tasks Generation Procedure and Checklist Generation Procedure in `_generation.md`
- [x] CHK-016 Review Behavior precondition checks for `plan.md` with both `## Tasks` and `## Acceptance` sections (not tasks.md/checklist.md)
- [x] CHK-017 Inward sub-agent quality checklist step inspects items in `plan.md ## Acceptance` and marks them in place
- [x] CHK-018 `fab-continue.md` dispatch table no longer has `tasks` rows; `spec ready` dispatches to apply (which generates plan.md and runs tasks)
- [x] CHK-019 `fab-ff.md` and `fab-fff.md` no longer contain Step 2 (Generate tasks.md) or Step 3 (Generate Quality Checklist) as separate steps
- [x] CHK-020 `fab-clarify.md` accepts `intake|spec|plan` as `<target-artifact>`; `tasks` target returns the strict-error pointer to `plan` or `spec`
- [x] CHK-021 `_preamble.md` State Table no longer has a `tasks` row; narrative stage count updated to 7
- [x] CHK-022 `_cli-fab.md` `Side effects of finish` line lists the new 7-stage cascade; `set-checklist` row replaced by `set-acceptance` row
- [x] CHK-023 `StageOrder` in `src/go/fab/internal/statusfile/statusfile.go` is exactly `["intake", "spec", "apply", "review", "hydrate", "ship", "review-pr"]`
- [x] CHK-024 `statusfile.StatusFile` Go struct has a `Plan` struct (replacing `Checklist`) with the four documented fields and YAML tags
- [x] CHK-025 `allowedStates` in `status.go` no longer has a `"tasks"` key; `isValidStage("tasks")` returns false
- [x] CHK-026 `defaultCommand` in `change.go` no longer references `"tasks"`
- [x] CHK-027 `expectedMinSpec` in `score.go` is unchanged; scoring code references neither `tasks.md`, `checklist.md`, nor `plan.md`
- [x] CHK-028 `MatchArtifactPath` in `hooklib/artifact.go` recognizes `intake.md`, `spec.md`, `plan.md`; no longer recognizes `tasks.md` or `checklist.md`
- [x] CHK-029 PostToolUse `artifactBookkeeping` for `plan.md` writes correctly populates `plan.generated=true`, `plan.task_count`, `plan.acceptance_count`, and `plan.acceptance_completed`; defensive: skips updates for missing sections
- [x] CHK-030 `status.SetAcceptance` Go function exists and updates `Plan.{Generated,TaskCount,AcceptanceCount,AcceptanceCompleted}` correctly; `status.SetChecklist` is removed (or returns strict error)
- [x] CHK-031 `statusSetAcceptanceCmd()` Cobra command is registered; `statusSetChecklistRemovedCmd()` (or equivalent error stub) is registered with the strict-error message
- [x] CHK-032 `preflight.go` YAML output emits a `plan:` block (not `checklist:`); `progress:` block emits exactly 7 keys
- [x] CHK-033 `fab change list` output reflects 7-stage progression (no `tasks` stage in any displayed change)
- [x] CHK-034 Migration file `src/kit/migrations/1.8.0-to-1.9.0.md` exists with Summary, Pre-check, Changes, and Verification sections
- [x] CHK-035 Migration is idempotent: re-running on a change with `plan.md` already present is a no-op
- [x] CHK-036 Migration handles in-flight `tasks.md`/`checklist.md`: produces `plan.md` (preserving CHK-NNN IDs verbatim), drops `progress.tasks` from `.status.yaml`, replaces `checklist:` block with `plan:` block, appends "safe to delete" note to legacy files
- [x] CHK-037 Migration handles archived changes: folders under `fab/changes/archive/**` are left untouched
- [x] CHK-038 `.status.yaml` template at `src/kit/templates/status.yaml` matches the new schema (7 progress keys, `plan:` block with four zero-initialized fields)

### Behavioral Correctness

- [x] CHK-039 After `spec` is finished, `apply` is the next active stage (not `tasks`)
- [x] CHK-040 Apply preconditions on `spec.md` (not `tasks.md`); old "tasks.md MUST exist" precondition is removed from `fab-continue.md`
- [x] CHK-041 Hydrate preconditions: review must have passed AND all `## Acceptance` items in `plan.md` are `[x]` (replaces "all checklist items `[x]`")
- [x] CHK-042 `/fab-clarify plan` works post-apply-entry on `plan.md` and is non-advancing
- [x] CHK-043 Clarify pre-flight stage guard: planning stages list drops `tasks`; `plan` is valid when `plan.md` exists at apply or later

### Removal Verification

- [x] CHK-044 No live source code references the `tasks` stage (excluding migration content, changelog entries, and intentionally-historical references in comments)
- [x] CHK-045 No live skill text refers to `tasks.md` or `checklist.md` as currently-generated artifacts (changelog mentions allowed; migration text allowed)
- [x] CHK-046 The `Checklist` Go struct/type is removed (or repurposed only as a strict-error helper); replaced by `Plan` struct everywhere it was used

### Scenario Coverage

- [x] CHK-047 Spec scenario "Section headings are the parser contract" — apply correctly reads only `## Tasks` content; review correctly reads only `## Acceptance` content
- [x] CHK-048 Spec scenario "spec finish auto-activates apply" — verified by inspection of test fixtures and/or runtime behavior
- [x] CHK-049 Spec scenario "Legacy tasks event errors" — `fab status finish <change> tasks` exits 1 with the documented stderr substrings
- [x] CHK-050 Spec scenario "set-checklist errors with pointer" — exit 1 with `"set-checklist" is now "set-acceptance"` in stderr
- [x] CHK-051 Spec scenario "set-acceptance updates plan block" — exit 0; `plan.acceptance_completed` updated; `last_updated` refreshed
- [x] CHK-052 Spec scenario "First-time apply generates and executes" — plan.md is created before any task execution begins
- [x] CHK-053 Spec scenario "Resume mid-apply skips plan generation" — apply detects existing plan.md and skips generation
- [x] CHK-054 Spec scenario "Migration idempotency" — running migration on a change with plan.md already is a no-op
- [x] CHK-055 Spec scenario "Migration: in-flight change at tasks: ready" — produces correct plan.md and rewrites .status.yaml per spec
- [x] CHK-056 Spec scenario "Migration: in-flight change at tasks: done" — preserves `progress.apply` state correctly when removing `progress.tasks`
- [x] CHK-057 Spec scenario "Migration: archived change" — archive folders untouched
- [x] CHK-058 Spec scenario "plan.md write triggers bookkeeping" — PostToolUse hook correctly populates plan block fields after a plan.md write
- [x] CHK-059 Spec scenario "tasks.md / checklist.md writes are no-ops" — PostToolUse hook returns without modifying .status.yaml when those legacy files are written

### Edge Cases & Error Handling

- [x] CHK-060 plan.md missing `## Acceptance` section at review time → review STOPs with clear message
- [x] CHK-061 plan.md malformed in mid-write (e.g., `## Tasks` exists but `## Acceptance` missing) → hook bookkeeping handles defensively (does not zero out fields)
- [x] CHK-062 Migration encounters a one-sided legacy state (only `tasks.md` or only `checklist.md` present, no `plan.md`) → handles gracefully without failing the entire migration

### Code Quality

- [x] CHK-063 Pattern consistency: New code follows naming and structural patterns of surrounding code (e.g., new `SetAcceptance` mirrors old `SetChecklist`; new `CountSectionItemsBounded` follows existing `CountUncheckedTasks`/`CountChecklistItems` style)
- [x] CHK-064 No unnecessary duplication: existing utilities (yaml.Node parsing, atomic-write helpers, `parseInt`) are reused where applicable

### Security

<!-- Not applicable — change has no security surface -->

### Documentation Accuracy

- [x] CHK-065 `docs/specs/overview.md` updated: stage count, mermaid diagram, stage details table all reflect 7 stages
- [x] CHK-066 `docs/specs/skills.md` per-skill flow updates reflect new pipeline
- [x] CHK-067 `docs/specs/templates.md` `tasks.md` + `checklist.md` entries replaced with single `plan.md` entry
- [x] CHK-068 `docs/specs/user-flow.md` diagrams updated to 7 stages
- [x] CHK-069 `docs/specs/architecture.md` `progress` map references updated if present
- [x] CHK-070 `docs/specs/glossary.md` updated: `tasks` stage entry removed, `plan.md` entry added/updated, `apply` mentions plan-generation sub-step
- [x] CHK-071 `docs/specs/skills/SPEC-fab-{continue,ff,fff,clarify}.md` flow diagrams reflect new 7-stage pipeline
- [x] CHK-072 **N/A**: Affected memory files (during hydrate) accurately reflect the implemented schema/skill/CLI changes — covered by hydrate stage validation (deferred to hydrate)

### Cross References

- [x] CHK-073 All inter-skill references updated consistently (e.g., `fab-continue.md` references to `_review.md` plan.md preconditions; `fab-clarify.md` references to plan target)
- [x] CHK-074 All `fab status set-checklist` callers internal to the kit (e.g., review skill, hooks) are updated to `fab status set-acceptance` in this same change <!-- rework cycle 2: re-verified — only remaining `set-checklist` reference in src/kit/ is the strict-error stub row in _cli-fab.md (intentional, documents the pointer) -->
- [x] CHK-077 SPEC-fab-status.md synced to fab-status skill: narrative reflects plan progress (tasks + acceptance counts) and 7-stage pipeline (rework cycle 2)
- [x] CHK-078 SPEC-git-pr.md synced to git-pr skill: Stats columns derive from plan.md ## Tasks + .status.yaml plan block; "Read: ... plan.md ..." replaces "Read: ... tasks.md ..." (rework cycle 2)
- [x] CHK-079 SPEC-preamble.md synced to _preamble skill: Layer 2 Change Context reads (intake, spec, plan) (rework cycle 2)
- [x] CHK-080 SPEC-hooks.md synced: PostToolUse entries for tasks.md/checklist.md replaced with plan.md; set-checklist references updated to set-acceptance (rework cycle 2)
- [x] CHK-081 _review.md rework-loop note points to fab-ff.md Step 3 / fab-fff.md Step 3 (was Step 6) (rework cycle 2)
- [x] CHK-082 Migration prunes `stage_directives.tasks` from fab/project/config.yaml with idempotency (rework cycle 2)
- [x] CHK-083 statusfile Load() drops legacy `checklist:` block when both `plan:` and `checklist:` coexist; test `TestPlanAndChecklistCoexistDropsChecklist` covers (rework cycle 2)
- [x] CHK-075 `git-pr.md` Stats table column derivations updated: "Checklist" → "Acceptance" and/or "Tasks" derivation updated to read `plan.md`
- [x] CHK-076 No dangling references to removed templates (`templates/tasks.md`, `templates/checklist.md`) in any kit file

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`

<!-- Migrated from legacy tasks.md + checklist.md on 2026-05-06. The legacy files
     remain in place per the original intent (see top-of-tasks.md comment); they
     are now annotated with "safe to delete" markers, mirroring how the migration
     would treat any other in-flight change. -->
