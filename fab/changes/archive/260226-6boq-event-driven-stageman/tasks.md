# Tasks: Event-Driven Stageman

**Change**: 260226-6boq-event-driven-stageman
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Schema & Infrastructure

- [x] T001 Update `fab/.kit/schemas/workflow.yaml` transitions section from `from→to` format to event-keyed format (`event`, `from`, `to` fields). Remove `condition` fields. Keep `default` and `review` sections.

- [x] T002 Add `resolve_change_arg()` function to `fab/.kit/scripts/lib/stageman.sh` — accepts a positional argument, returns the resolved `.status.yaml` path. If the argument is an existing file, use it directly. Otherwise call `changeman.sh resolve "$arg"` and append `/.status.yaml`. Wire into CLI dispatch so ALL commands (event and non-event) resolve through it.

- [x] T003 Add `lookup_transition()` function to `fab/.kit/scripts/lib/stageman.sh` — given an event name and stage, read the event-keyed transitions from `workflow.yaml`. Check stage-specific override first (e.g., `review`), fall back to `default`. Return the matching `from` list and `to` state, or exit 1 if no match.

## Phase 2: Core Event Implementation

- [x] T004 Implement `event_start()` in `fab/.kit/scripts/lib/stageman.sh` — validates current state is in `from` list for `start` event via `lookup_transition`, writes new state atomically, applies `_apply_metrics_side_effect` (active: started_at, driver, iterations++).

- [x] T005 Implement `event_advance()` in `fab/.kit/scripts/lib/stageman.sh` — validates current state is `active` (via lookup), writes `ready` atomically. No metrics side-effect.

- [x] T006 Implement `event_finish()` in `fab/.kit/scripts/lib/stageman.sh` — validates current state is in `[active, ready]`, writes `done` atomically, sets `completed_at` via metrics. Side-effect: if next stage exists and is `pending`, set it to `active` with metrics (started_at, driver, iterations=1) in the same atomic write.

- [x] T007 Implement `event_reset()` in `fab/.kit/scripts/lib/stageman.sh` — validates current state is in `[done, ready]`, writes `active` atomically with metrics (started_at, driver, iterations++). Cascade side-effect: set all stages after the target to `pending` and remove their `stage_metrics` entries in the same atomic write.

- [x] T008 Implement `event_fail()` in `fab/.kit/scripts/lib/stageman.sh` — validates stage is `review` (by checking that `fail` event exists in workflow.yaml for this stage), validates current state is `active`, writes `failed`. No metrics side-effect.

- [x] T009 Add CLI dispatch cases for `start`, `advance`, `finish`, `reset`, `fail` in `fab/.kit/scripts/lib/stageman.sh`. Each case: parse args (`<change> <stage> [driver]`), call `resolve_change_arg` on the change argument, delegate to the corresponding `event_*` function. Update `show_help()` with new commands.

## Phase 3: Caller Migration

- [x] T010 Update `fab/.kit/scripts/lib/changeman.sh` — replace `set-state "$status_file" intake active fab-new` with `start "$status_file" intake fab-new` (keeps raw path since changeman already has it). Single call site.

- [x] T011 [P] Update `fab/.kit/skills/fab-continue.md` — replace all `stageman.sh set-state` and `stageman.sh transition` references with event commands. Map per spec §Skills migration table. Update reset flow to use `reset` (cascade handles downstream).

- [x] T012 [P] Update `fab/.kit/skills/fab-ff.md` — replace all `stageman.sh set-state` and `stageman.sh transition` references with event commands.

- [x] T013 [P] Update `fab/.kit/skills/fab-fff.md` — replace all `stageman.sh set-state` and `stageman.sh transition` references with event commands.

- [x] T014 Update `src/lib/changeman/SPEC-changeman.md` — replace `set-state` reference with equivalent event command.

## Phase 4: Removal

- [x] T015 Remove `set_stage_state()`, `transition_stages()` functions from `fab/.kit/scripts/lib/stageman.sh`. Remove `set-state` and `transition` CLI dispatch cases. Remove from `show_help()`.

## Phase 5: Tests & Documentation

- [x] T016 Rewrite `src/lib/stageman/test.bats` — remove all `set-state` and `transition` tests. Add event-based tests: happy paths (8 valid transitions), rejections (invalid from-state per event), finish side-effect (next stage activation, hydrate no side-effect, non-pending next stage), reset cascade (downstream to pending, metrics removed, before-target preserved), review-specific (fail only review, start from failed only review), stage metrics per event, change-ID resolution, error cases (missing file, invalid stage, invalid event).

- [x] T017 [P] Update `docs/memory/fab-workflow/change-lifecycle.md` — add state transition table, update "Two-write transitions" to reference `finish` and `reset`, update `stage_metrics` description to reference event commands, remove `set-state`/`transition` CLI references.

- [x] T018 [P] Update `docs/memory/fab-workflow/schemas.md` — update transitions section description to reference event-keyed format.

- [x] T019 [P] Update `docs/memory/fab-workflow/execution-skills.md` — update "Status mutations" overview and all inline CLI examples to use event commands.

- [x] T020 [P] Update `docs/memory/fab-workflow/planning-skills.md` — update "Shared Generation Partial" notes and all stage transition references to use event commands.

---

## Execution Order

- T001 blocks T003 (transition lookup reads the new schema format)
- T002 blocks T009 (CLI dispatch uses resolution function)
- T003 blocks T004-T008 (event functions use transition lookup)
- T004-T008 block T009 (CLI dispatch delegates to event functions)
- T009 blocks T010-T014 (callers need the new CLI commands)
- T010-T014 block T015 (removal requires all callers migrated)
- T015 blocks T016 (tests cover the final API, not the transitional state)
- T016 is independent of T017-T020 (tests vs docs)
