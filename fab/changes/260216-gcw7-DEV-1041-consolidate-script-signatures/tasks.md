# Tasks: Consolidate Script Signatures

**Change**: 260216-gcw7-DEV-1041-consolidate-script-signatures
**Spec**: `spec.md`
**Intake**: `intake.md`

<!--
  TASK FORMAT: - [ ] {ID} [{markers}] {Description with file paths}

  Markers (optional, combine as needed):
    [P]   — Parallelizable (different files, no dependencies on other [P] tasks in same group)

  IDs are sequential: T001, T002, ...
  Include exact file paths in descriptions.
  Each task should be completable in one focused session.

  Tasks are grouped by phase. Phases execute sequentially.
  Within a phase, [P] tasks can execute in parallel.
-->

## Phase 1: Remove Dead Code from stageman.sh

<!-- Remove unused functions, CLI dispatch entries, and self-test infrastructure from `fab/.kit/scripts/lib/stageman.sh`. -->

- [x] T001 Remove 12 dead function definitions from `fab/.kit/scripts/lib/stageman.sh`: `is_terminal_state` (lines 78-86), `get_stage_number` (108-115), `get_stage_name` (118-128), `get_stage_artifact` (131-142), `get_initial_state` (171-180), `is_required_stage` (183-191), `has_auto_checklist` (194-202), `get_state_symbol` (52-62), `get_state_suffix` (65-75), `format_state` (686-694), `get_stage_metrics` (247-267), `set_stage_metric` (272-292). Keep all other functions intact — especially `_apply_metrics_side_effect` (297-322).
- [x] T002 Remove 18 CLI dispatch `case` arms from `fab/.kit/scripts/lib/stageman.sh`: dead function entries (`all-states`, `validate-state`, `state-symbol`, `state-suffix`, `is-terminal`, `validate-stage`, `stage-number`, `stage-name`, `stage-artifact`, `allowed-states`, `initial-state`, `is-required`, `has-auto-checklist`, `validate-stage-state`, `next-stage`, `format-state`, `stage-metrics`, `set-stage-metric`) and meta entries (`--version`/`-v`, `--test`/`-t`, `""` empty-arg). Update the empty-arg case to show help text instead. Remove the section comment headers for "State Queries", "Stage Metrics", and "Display" dispatch blocks.
- [x] T003 Delete `run_tests()` (lines 877-916) and `show_version()` (lines 872-875) functions from `fab/.kit/scripts/lib/stageman.sh`. Rewrite `show_help()` to list only the 14 retained subcommands grouped into 4 categories: Stage query (`all-stages`), .status.yaml accessors (`progress-map`, `checklist`, `confidence`), Progression (`current-stage`), Validation (`validate-status-file`), Write commands (`set-state`, `transition`, `set-checklist`, `set-confidence`, `set-confidence-fuzzy`), History (`log-command`, `log-confidence`, `log-review`). Remove `--test | --version` from the USAGE line. Update the file header comment (lines 7-12) to remove references to deleted subcommands.

## Phase 2: Update Test Suites

<!-- Remove tests for deleted or CLI-removed subcommands. Keep all tests for retained subcommands. -->

- [x] T004 [P] Remove tests from `src/lib/stageman/test.bats` for removed CLI subcommands: "all-states includes all state values" (line 49), "validate-state accepts/rejects" (lines 58-66), "state-symbol returns correct symbols" (line 68), "is-terminal returns true/false" (lines 75-83), "validate-stage accepts/rejects" (lines 102-110), "stage-number returns correct numbers" (line 112), "stage-name returns display name" (line 119), "stage-artifact returns correct filename" (line 124), "allowed-states for review" (line 131), "initial-state returns correct state" (line 137), "has-auto-checklist returns true/false" (lines 144-152), "next-stage after spec/intake/hydrate" (lines 158-171), "stage-metrics: empty on empty block" (line 667), "stage-metrics: single stage returns fields" (line 728), "stage-metrics: all stages returns correct entry count" (line 735), "stage-metrics: handles missing block" (line 744). Also remove the now-empty "State Tests" and "Progression" section headers. Keep all stage-metrics tests that verify `set-state`/`transition` side-effects (lines 673-726) and all other retained subcommand tests.
- [x] T005 [P] Remove 2 test blocks from `src/lib/stageman/test-simple.sh`: "Testing all-states..." (lines 7-14) and "Testing stage-number..." (lines 25-32). Keep tests for `all-stages`, `progress-map`, `checklist`, `confidence`.

## Phase 3: Verify & Document

<!-- Run tests, update SPEC and memory files to match the reduced CLI. -->

- [x] T006 Run `bats src/lib/stageman/test.bats` and `bash src/lib/stageman/test-simple.sh` to verify all remaining tests pass. Fix any failures caused by the removals.
- [x] T007 [P] Update `src/lib/stageman/SPEC-stageman.md`: remove "State Queries" table (lines 37-43), "Stage Metrics" table (lines 71-73), "Display" table (lines 91-92); reduce "Stage Queries" section to only `all-stages`; reduce "Progression" section to only `current-stage`; remove `next-stage` row; update Usage code block to remove deleted subcommand examples; remove `--version` and `--test` from Usage section; update Testing section to remove self-test reference; add v3.0.0 changelog entry.
- [x] T008 [P] Update `docs/memory/fab-workflow/kit-architecture.md`: change "~35 CLI subcommands" to "14 CLI subcommands"; replace the 6-category subcommand listing with 4 categories (Schema/accessor, Write, History, plus note about internal helpers retained); update test count if it changed.

---

## Execution Order

- T001 blocks T002 (function removals before dispatch removals to avoid referencing deleted functions)
- T002 blocks T003 (dispatch cleanup before help text rewrite, since show_help covers the final set)
- T004, T005 are independent of each other, both can start after Phase 1
- T006 blocks T007, T008 (verify tests pass before updating docs)
- T007, T008 are independent of each other
