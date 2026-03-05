# Tasks: Stage Metrics, History Tracking & Stageman yq Migration

**Change**: 260214-r7k3-stageman-yq-metrics
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Install yq v4 binary for linux/arm64, add yq detection guard at top of `fab/.kit/scripts/lib/stageman.sh` (check on source, emit error + exit if missing), add `stage_metrics: {}` block to `fab/.kit/templates/status.yaml` between confidence and last_updated

## Phase 2: Core Migration — Accessors & Building Blocks

- [x] T002 [P] Migrate `get_progress_map`, `get_checklist`, `get_confidence` from awk/grep/sed to yq in `fab/.kit/scripts/lib/stageman.sh` — preserve `key:value` output format, handle missing blocks with defaults
- [x] T003 [P] Add `get_stage_metrics` and `set_stage_metric` functions to `fab/.kit/scripts/lib/stageman.sh` — yq-based, flow-style YAML, handle empty/missing `stage_metrics` block gracefully (return empty)

## Phase 3: Write Function Migration + Side-Effects

- [x] T004 Migrate `set_stage_state` from awk to yq in `fab/.kit/scripts/lib/stageman.sh` — add optional `[driver]` parameter (required for `active`, error if missing), add stage_metrics side-effects (active: set started_at/driver/iterations, done: set completed_at, pending: remove entry, failed: no-op), update CLI `set-state` command signature
- [x] T005 Migrate `transition_stages` from awk to yq in `fab/.kit/scripts/lib/stageman.sh` — add optional `[driver]` parameter (required, applied to to_stage), add stage_metrics side-effects for both from→done and to→active, update CLI `transition` command signature
- [x] T006 [P] Migrate `set_checklist_field` and `set_confidence_block` from awk to yq in `fab/.kit/scripts/lib/stageman.sh` — preserve validation, temp-file-then-mv, last_updated refresh
- [x] T007 Migrate `validate_status_file` from grep/awk to yq in `fab/.kit/scripts/lib/stageman.sh` — skip stage_metrics validation, preserve valid-states-per-stage and active-count checks

## Phase 4: History Logging & Integration

- [x] T008 Add `log_command`, `log_confidence`, `log_review` functions to `fab/.kit/scripts/lib/stageman.sh` — append JSON to `<change_dir>/.history.jsonl`, create file on first event, add CLI commands `log-command`, `log-confidence`, `log-review` to dispatcher
- [x] T009 Migrate `fab/.kit/scripts/lib/calc-score.sh` — replace direct grep/sed reads with `get_confidence` accessor, add `log_confidence` call after score computation
- [x] T010 Update skill prompts to pass driver on transitions and call `log_command` at invocation start: `fab/.kit/skills/fab-continue.md` (all transition/set-state calls + log_command), `fab/.kit/skills/fab-ff.md` (same), `fab/.kit/skills/fab-fff.md` (same), `fab/.kit/skills/_generation.md` (no driver needed for set-checklist), `fab/.kit/skills/fab-new.md` (set-state brief active with driver + log_command after folder creation), `fab/.kit/skills/fab-clarify.md` (log_command only)

## Phase 5: Testing

- [x] T011 Update `src/lib/stageman/test.sh` and `test-simple.sh` for new function signatures (driver param on set_stage_state/transition_stages), add tests for: stage_metrics side-effects (activation, completion, rework, reset-to-pending), get_stage_metrics (all/single/empty), log functions (command/confidence/review events, file creation), validate_status_file ignoring stage_metrics. Run `src/lib/stageman/test.sh`, `src/lib/calc-score/test.sh`, `src/lib/preflight/test.sh`

---

## Execution Order

- T001 blocks all other tasks (yq must be available)
- T003 blocks T004, T005 (set_stage_metric is used by set_stage_state side-effects)
- T004 blocks T005 (transition_stages reuses set_stage_state pattern)
- T004, T005 block T010 (skill prompts need updated signatures)
- T008 blocks T009 (calc-score needs log_confidence)
- T002, T003 are parallel after T001
- T006, T007 are independent of T003-T005 (no metrics involvement)
