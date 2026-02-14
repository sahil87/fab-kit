# Tasks: Stage Metrics, History Tracking, and stageman yq Migration

**Change**: 260214-r7k3-stageman-yq-metrics
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Amend governance text in `fab/constitution.md` to explicitly allow single-binary runtime tooling (e.g., `yq`) while preserving the no-package-manager/no-build-step principles.
- [x] T002 Add `stage_metrics: {}` to `fab/.kit/templates/status.yaml` and document metrics schema expectations in `fab/.kit/schemas/workflow.yaml`.
- [x] T003 [P] Update runtime/developer docs to reflect new `yq` dependency and metrics/history capabilities (`src/lib/stageman/README.md`, `src/lib/calc-score/README.md`).

## Phase 2: Core Implementation

- [x] T004 Refactor `fab/.kit/scripts/lib/stageman.sh` status accessors (`get_progress_map`, `get_checklist`, `get_confidence`) to use `yq` queries.
- [x] T005 Refactor `fab/.kit/scripts/lib/stageman.sh` write functions (`set_stage_state`, `transition_stages`, `set_checklist_field`, `set_confidence_block`) to use `yq` mutations with atomic writes and `last_updated` refresh.
- [x] T006 Implement stage metrics + history helpers in `fab/.kit/scripts/lib/stageman.sh`: `get_stage_metrics`, `set_stage_metric`, `log_command`, `log_confidence`, `log_review`.
- [x] T007 Integrate automatic `stage_metrics` side-effects into `set_stage_state` and `transition_stages` in `fab/.kit/scripts/lib/stageman.sh` (`started_at`, `completed_at`, `driver`, `iterations`).
- [x] T008 Add fail-fast `yq` runtime checks with clear install guidance in `fab/.kit/scripts/lib/stageman.sh` and ensure dependent scripts inherit this behavior.
- [x] T009 Refactor `fab/.kit/scripts/lib/calc-score.sh` to read prior confidence via stageman accessors and append confidence history events via `log_confidence` after successful score writes.

## Phase 3: Integration & Edge Cases

- [x] T010 Update stageman test fixtures and assertions for `stage_metrics`, `.history.jsonl`, and new helper behavior in `src/lib/stageman/test.sh` and `src/lib/stageman/test-simple.sh`.
- [x] T011 Update calc-score tests to cover confidence history logging integration and `yq`-backed behavior in `src/lib/calc-score/test.sh` and `src/lib/calc-score/test-simple.sh`.
- [x] T012 Update skill prompts to pass driver identity and call command logging (`fab/.kit/skills/fab-new.md`, `fab/.kit/skills/fab-continue.md`, `fab/.kit/skills/fab-ff.md`, `fab/.kit/skills/fab-fff.md`, `fab/.kit/skills/fab-clarify.md`, `fab/.kit/skills/fab-switch.md`).

## Phase 4: Polish

- [x] T013 [P] Run targeted validation for touched modules (`src/lib/stageman/test.sh`, `src/lib/calc-score/test.sh`) and fix regressions.
- [x] T014 Sync memory docs for hydrate targets with shipped behavior updates (`docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`).

---

## Execution Order

- T001-T003 establish policy + schema + docs prerequisites before implementation starts.
- T004-T009 are core script changes; T006/T007 depend on T004/T005 foundations.
- T010-T011 depend on T004-T009.
- T012 depends on stageman logging API shape from T006.
- T013 requires T010-T012 complete.
- T014 runs after implementation and tests to hydrate memory with final shipped behavior.
