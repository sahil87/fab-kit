# Quality Checklist: Collapse Tasks Stage into Apply; Replace tasks.md + checklist.md with plan.md

**Change**: 260423-qszh-merge-tasks-checklist
**Generated**: 2026-05-06
**Spec**: `spec.md`

## Functional Completeness

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

## Behavioral Correctness

- [x] CHK-039 After `spec` is finished, `apply` is the next active stage (not `tasks`)
- [x] CHK-040 Apply preconditions on `spec.md` (not `tasks.md`); old "tasks.md MUST exist" precondition is removed from `fab-continue.md`
- [x] CHK-041 Hydrate preconditions: review must have passed AND all `## Acceptance` items in `plan.md` are `[x]` (replaces "all checklist items `[x]`")
- [x] CHK-042 `/fab-clarify plan` works post-apply-entry on `plan.md` and is non-advancing
- [x] CHK-043 Clarify pre-flight stage guard: planning stages list drops `tasks`; `plan` is valid when `plan.md` exists at apply or later

## Removal Verification

- [x] CHK-044 No live source code references the `tasks` stage (excluding migration content, changelog entries, and intentionally-historical references in comments)
- [x] CHK-045 No live skill text refers to `tasks.md` or `checklist.md` as currently-generated artifacts (changelog mentions allowed; migration text allowed)
- [x] CHK-046 The `Checklist` Go struct/type is removed (or repurposed only as a strict-error helper); replaced by `Plan` struct everywhere it was used

## Scenario Coverage

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

## Edge Cases & Error Handling

- [x] CHK-060 plan.md missing `## Acceptance` section at review time → review STOPs with clear message
- [x] CHK-061 plan.md malformed in mid-write (e.g., `## Tasks` exists but `## Acceptance` missing) → hook bookkeeping handles defensively (does not zero out fields)
- [x] CHK-062 Migration encounters a one-sided legacy state (only `tasks.md` or only `checklist.md` present, no `plan.md`) → handles gracefully without failing the entire migration

## Code Quality

- [x] CHK-063 Pattern consistency: New code follows naming and structural patterns of surrounding code (e.g., new `SetAcceptance` mirrors old `SetChecklist`; new `CountSectionItemsBounded` follows existing `CountUncheckedTasks`/`CountChecklistItems` style)
- [x] CHK-064 No unnecessary duplication: existing utilities (yaml.Node parsing, atomic-write helpers, `parseInt`) are reused where applicable

## Security

<!-- Not applicable — change has no security surface -->

## Documentation Accuracy

- [x] CHK-065 `docs/specs/overview.md` updated: stage count, mermaid diagram, stage details table all reflect 7 stages
- [x] CHK-066 `docs/specs/skills.md` per-skill flow updates reflect new pipeline
- [x] CHK-067 `docs/specs/templates.md` `tasks.md` + `checklist.md` entries replaced with single `plan.md` entry
- [x] CHK-068 `docs/specs/user-flow.md` diagrams updated to 7 stages
- [x] CHK-069 `docs/specs/architecture.md` `progress` map references updated if present
- [x] CHK-070 `docs/specs/glossary.md` updated: `tasks` stage entry removed, `plan.md` entry added/updated, `apply` mentions plan-generation sub-step
- [x] CHK-071 `docs/specs/skills/SPEC-fab-{continue,ff,fff,clarify}.md` flow diagrams reflect new 7-stage pipeline
- [x] CHK-072 **N/A**: Affected memory files (during hydrate) accurately reflect the implemented schema/skill/CLI changes — covered by hydrate stage validation (deferred to hydrate)

## Cross References

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
