# Tasks: Batch Pipeline — Single Change Support & Default Base Branch

**Change**: 260223-xiuk-batch-pipeline-single-change-and-base-branch
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Relax minimum change count from 2 to 1 in `fab/.kit/scripts/batch-pipeline-series.sh` — change guard on line 77 from `-lt 2` to `-lt 1`, update error message (line 78), update usage text: arguments line to `<change> [<change>...]`, remove "at least 2 required" from description, add single-change example
- [x] T002 [P] Default `base` to current branch in `fab/.kit/scripts/pipeline/run.sh` `validate_manifest()` (lines 87-91) — replace error-on-missing-base with `git branch --show-current` fallback (last-resort `main`), write resolved value back to manifest via `yq -i`
- [x] T003 [P] Update `fab/pipelines/example.yaml` — document `base` field as optional with comment explaining default behavior (current branch, `main` fallback)

## Phase 2: Tests

- [x] T004 Add git stub for `git branch --show-current` to `src/scripts/pipeline/test.bats` setup, replace "missing base field fails" test with "missing base resolves to current branch" test, add "empty base resolves to current branch" test

---

## Execution Order

- T001, T002, T003 are independent (different files), can run in parallel
- T004 depends on T002 conceptually (tests verify the new validate_manifest behavior)
