# Tasks: Stageman Write API

**Change**: 260214-w3r8-stageman-write-api
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Write Functions

<!-- All functions go in fab/.kit/scripts/_stageman.sh under a new "Write Functions" section after the existing Accessors section. Each uses: validate inputs → awk transform (mutation + last_updated) → mktemp in same dir → mv. -->

- [x] T001 Implement `set_stage_state` in `fab/.kit/scripts/_stageman.sh` — validate file exists, `validate_stage`, `validate_stage_state`; awk to replace `^  {stage}: {old}` with new state + update `last_updated` to `date -Iseconds`; temp-file-then-mv; return 0/1
- [x] T002 Implement `transition_stages` in `fab/.kit/scripts/_stageman.sh` — validate file exists, both stages via `validate_stage`, `done` allowed for from + `active` allowed for to via `validate_stage_state`, read current state of from_stage and reject if not `active`, `get_next_stage` adjacency check; single awk pass setting from→done + to→active + last_updated; temp-file-then-mv
- [x] T003 [P] Implement `set_checklist_field` in `fab/.kit/scripts/_stageman.sh` — validate file exists, field is `generated|completed|total`, value type (bool for generated, non-negative int for others); awk to replace field in checklist block + update last_updated; temp-file-then-mv
- [x] T004 [P] Implement `set_confidence_block` in `fab/.kit/scripts/_stageman.sh` — validate file exists, all counts are non-negative integers, score is non-negative float; awk to replace entire `confidence:` block through next top-level key + update last_updated; temp-file-then-mv

## Phase 2: CLI & Integration

- [x] T005 Add CLI write subcommands to `fab/.kit/scripts/_stageman.sh` — extend the `case` statement in the CLI section with `set-state`, `transition`, `set-checklist`, `set-confidence`; validate argument count per subcommand; delegate to corresponding function; exit with function's return code
- [x] T006 Update `--help` output in `fab/.kit/scripts/_stageman.sh` to list write commands under a "Write commands" section; update `--test` self-test to exercise write functions with a temp status file
- [x] T007 Refactor `fab/.kit/scripts/_calc-score.sh` — source `_stageman.sh` at top (after set -euo pipefail); replace inline awk write block (lines ~117-141: tmpfile, awk confidence replacement, mv) with `set_confidence_block "$status_file" "$total_certain" "$table_confident" "$table_tentative" "$unresolved" "$score"`; keep stdout YAML emit block unchanged

## Phase 3: Skill Prompt Updates

- [x] T008 [P] Update `fab/.kit/skills/fab-continue.md` — replace ad-hoc `.status.yaml` edit instructions with `_stageman.sh` CLI: Step 4 two-write transition → `_stageman.sh transition`; Apply preconditions/completion → `_stageman.sh transition`; Review pass → `_stageman.sh transition`; Review fail → `_stageman.sh set-state` for each; Hydrate → `_stageman.sh set-state`; Reset flow → `_stageman.sh set-state` for each stage
- [x] T009 [P] Update `fab/.kit/skills/fab-ff.md` — replace status update instructions with `_stageman.sh` CLI: Step 2 spec→done, Step 5 tasks→done/apply→active + checklist fields, Step 6 apply→done/review→active, Step 7 review→done/hydrate→active, Step 8 hydrate→done
- [x] T010 [P] Update `fab/.kit/skills/fab-fff.md` — add note that all `.status.yaml` transitions use `_stageman.sh` CLI (Steps 2-4 reference fab-ff/fab-continue behavior)
- [x] T011 [P] Update `fab/.kit/skills/_generation.md` — replace Checklist Generation Procedure Step 6 `.status.yaml` update with `_stageman.sh set-checklist` CLI calls for `generated`, `total`, `completed` fields

## Phase 4: Tests

- [x] T012 Add write function tests to `src/stageman/test.sh` — test all 4 write functions: valid operations (state change, transition, checklist field, confidence block), validation failures (invalid stage, invalid state for stage, file not found, non-adjacent transition, from_stage not active, invalid checklist field, negative count, non-numeric score), verify file unchanged on validation error, verify last_updated refreshed on success

---

## Execution Order

- T001 establishes the write pattern → T002 follows same pattern with additional checks
- T003, T004 are independent of T002, can run in parallel
- T005, T006 depend on T001-T004 (CLI dispatches to write functions)
- T007 depends on T004 (`set_confidence_block` must exist)
- T008-T011 depend on T005 (skill prompts reference CLI commands)
- T012 depends on T001-T004 (tests exercise write functions)
