# Tasks: Extend Pipeline Through PR

**Change**: 260303-he6t-extend-pipeline-through-pr
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Schema & Template

- [x] T001 [P] Add `ship` and `review-pr` stages to `fab/.kit/schemas/workflow.yaml` — new stage entries after hydrate, `review-pr` transition overrides (same as review), update `stage_numbers` to 8 stages, update `progression.completion` rule and `current_stage.fallback` to `review-pr`
- [x] T002 [P] Add `ship: pending` and `review-pr: pending` to progress map in `fab/.kit/templates/status.yaml`

## Phase 2: Core Scripts

- [x] T003 Update `fab/.kit/scripts/lib/statusman.sh` — change `get_current_stage` final fallback from `hydrate` to `review-pr` (line 312), extend `event_finish` auto-log to include `review-pr` alongside `review`, extend `event_fail` auto-log to include `review-pr` alongside `review`
- [x] T004 Update `fab/.kit/scripts/lib/changeman.sh` — update `stage_number()` to add ship=7 and review-pr=8, update `next_stage()` to chain hydrate→ship→review-pr→"", update `default_command()` to add ship and review-pr routing, change display from `(N/6)` to `(N/8)`

## Phase 3: Skill Files

- [x] T005 Rename `fab/.kit/skills/git-review.md` to `fab/.kit/skills/git-pr-review.md` — update name in YAML frontmatter from `git-review` to `git-pr-review`, update heading from `/git-review` to `/git-pr-review`
- [x] T006 Add statusman integration to `fab/.kit/skills/git-pr.md` — add Step 0.5 (start ship stage before pipeline) and Step 4d (finish ship stage after PR sentinel), best-effort with silent failure, skip if no active change or ship already done
- [x] T007 Add statusman integration to `fab/.kit/skills/git-pr-review.md` — add start/finish/fail calls for `review-pr` stage, add phase sub-state tracking via yq writes to `stage_metrics.review-pr.phase`, add reviewer tracking
- [x] T008 Update `fab/.kit/skills/_preamble.md` — add `ship`, `review-pr (pass)`, `review-pr (fail)` rows to state table
- [x] T009 [P] Update `fab/.kit/skills/fab-ff.md` — add Step 8 (Ship: invoke git-pr behavior) and Step 9 (Review-PR: invoke git-pr-review behavior) after hydrate, update completion message
- [x] T010 [P] Update `fab/.kit/skills/fab-fff.md` — add Step 9 (Ship) and Step 10 (Review-PR) after hydrate, update completion message
- [x] T011 Update `fab/.kit/skills/fab-continue.md` — update "6-stage" to "8-stage", add ship and review-pr to dispatch table and valid reset targets, update error messages

## Phase 4: Cross-References

- [x] T012 Update all references from `/git-review` to `/git-pr-review` across skill files — grep for `git-review` in `fab/.kit/skills/` and update matches (excluding `git-pr-review.md` itself)
- [x] T013 Run `fab/.kit/scripts/lib/statusman.sh validate-status-file` against the current change's `.status.yaml` to verify schema compatibility

---

## Execution Order

- T001 and T002 are independent (parallel)
- T003 and T004 depend on T001 (schema must exist first)
- T005 must complete before T006, T007, T012 (file must be renamed first)
- T008 through T011 depend on T003 and T004 (scripts must be updated first)
- T009 and T010 are independent (parallel)
- T012 depends on T005
- T013 runs last (validation)
