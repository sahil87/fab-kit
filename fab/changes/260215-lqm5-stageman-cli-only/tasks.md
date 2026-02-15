# Tasks: Migrate Stageman to CLI-Only Interface

**Change**: 260215-lqm5-stageman-cli-only
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Add CLI Subcommands

<!-- Purely additive — no existing behavior changes. Zero regression risk. -->

- [x] T001 Add state query subcommands to CLI dispatch in `fab/.kit/scripts/lib/stageman.sh`: `all-states`, `validate-state <state>`, `state-symbol <state>`, `state-suffix <state>`, `is-terminal <state>`
- [x] T002 Add stage query subcommands to CLI dispatch in `fab/.kit/scripts/lib/stageman.sh`: `all-stages`, `validate-stage <stage>`, `stage-number <stage>`, `stage-name <stage>`, `stage-artifact <stage>`, `allowed-states <stage>`, `initial-state <stage>`, `is-required <stage>`, `has-auto-checklist <stage>`, `validate-stage-state <stage> <state>`
- [x] T003 Add .status.yaml accessor subcommands to CLI dispatch in `fab/.kit/scripts/lib/stageman.sh`: `progress-map <file>`, `checklist <file>`, `confidence <file>`
- [x] T004 Add stage metrics, progression, validation, and display subcommands to CLI dispatch in `fab/.kit/scripts/lib/stageman.sh`: `stage-metrics <file> [stage]`, `set-stage-metric <file> <stage> <field> <value>`, `current-stage <file>`, `next-stage <stage>`, `validate-status-file <file>`, `format-state <state>`
- [x] T005 Add `set-confidence-fuzzy` write subcommand to CLI dispatch in `fab/.kit/scripts/lib/stageman.sh`: `set-confidence-fuzzy <file> <certain> <confident> <tentative> <unresolved> <score> <mean_s> <mean_r> <mean_a> <mean_d>`

## Phase 2: Migrate Callers

<!-- Replace source/import with subprocess invocation. Each task independently testable. -->

- [x] T006 Migrate `fab/.kit/scripts/lib/preflight.sh` from `source stageman.sh` to `$STAGEMAN <subcommand>` subprocess calls. Replace 6 function calls: `validate_status_file`, `get_progress_map`, `get_current_stage`, `get_checklist`, `get_confidence`, `get_all_stages`. Keep `source resolve-change.sh` unchanged.
- [x] T007 Migrate `fab/.kit/scripts/lib/calc-score.sh` from `source stageman.sh` to `$STAGEMAN <subcommand>` subprocess calls. Replace 4 calls: `get_confidence`, `set_confidence_block`, `set_confidence_block_fuzzy`, `log_confidence`.
- [x] T008 Migrate `src/lib/stageman/test.sh` from `source stageman.sh` to `$STAGEMAN <subcommand>` CLI pattern. Convert all function calls to subprocess invocations. Preserve all existing assertions.
- [x] T009 Migrate `src/lib/stageman/test-simple.sh` from `source stageman.sh` to `$STAGEMAN <subcommand>` CLI pattern.

## Phase 3: Remove Dual-Mode Scaffolding

<!-- All callers migrated — safe to remove source-mode support. -->

- [x] T010 Remove dual-mode scaffolding from `fab/.kit/scripts/lib/stageman.sh`: delete `BASH_SOURCE[0]` guard (line 950) and closing `fi`, replace all `return 1 2>/dev/null || exit 1` patterns with `exit 1`, remove source-oriented comments from file header
- [x] T011 Rewrite `--help` output in `fab/.kit/scripts/lib/stageman.sh`: remove "As library" section, replace "AVAILABLE FUNCTIONS" with "SUBCOMMANDS" organized by category, list all ~35 subcommands with usage signatures

## Phase 4: Polish

- [x] T012 Update `src/lib/stageman/README.md`: remove "As Library" usage section, update API reference to show subcommand names, update all examples to CLI pattern

---

## Execution Order

- T001–T005 are independent, can be done sequentially or in any order within Phase 1
- T006–T009 each depend on T001–T005 (subcommands must exist before callers migrate)
- T008–T009 depend on T010 being NOT yet done (test migration should happen before guard removal so tests can validate the transition)
- T010–T011 depend on T006–T009 (all callers must be migrated before removing source support)
- T012 depends on T010–T011 (docs should reflect final state)
