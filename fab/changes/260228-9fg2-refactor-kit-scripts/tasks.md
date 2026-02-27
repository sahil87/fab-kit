# Tasks: Refactor Kit Scripts Into Atomic Responsibilities

**Change**: 260228-9fg2-refactor-kit-scripts
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup — New Scripts

- [x] T001 Create `fab/.kit/scripts/lib/resolve.sh` — extract resolution logic from `changeman.sh cmd_resolve()`, add `--id` (default), `--folder`, `--dir`, `--status` output flags, add change ID extraction from folder name
- [x] T002 Create `fab/.kit/scripts/lib/logman.sh` — three subcommands (`command`, `confidence`, `review`), resolve via `resolve.sh --dir`, append JSON to `.history.jsonl`

## Phase 2: Core Implementation — Script Refactoring

- [x] T003 Rename `fab/.kit/scripts/lib/stageman.sh` → `fab/.kit/scripts/lib/statusman.sh` — update internal variable names (`STAGEMAN_DIR` → `STATUSMAN_DIR`), update `show_help()`, update CLI header comment
- [x] T004 Remove `resolve_change_arg()` from `statusman.sh` — replace all CLI dispatch resolution with calls to `resolve.sh --status`
- [x] T005 Remove `log_command`, `log_confidence`, `log_review` functions and their CLI dispatch cases from `statusman.sh` — remove the "History Logging" section and "History Commands" CLI section
- [x] T006 Add auto-log review outcomes to `statusman.sh` — `event_finish` for review stage calls `logman.sh review "passed"`, `event_fail` for review stage calls `logman.sh review "failed" [rework]`, update `fail` CLI dispatch to accept optional `[rework]` argument for review stage
- [x] T007 Update `fab/.kit/scripts/lib/changeman.sh` — replace `cmd_resolve()` with calls to `resolve.sh`, update `STAGEMAN` variable to `STATUSMAN` referencing `statusman.sh`, update `log-command` calls in `cmd_new` and `cmd_rename` to use `logman.sh command`
- [x] T008 DRY `fab/.kit/scripts/lib/calc-score.sh` — extract `count_grades()` and `compute_score()` helper functions, unify gate-check and normal-scoring paths, switch input from `<change-dir>` to `<change>` via `resolve.sh --dir`, update `STAGEMAN` → `STATUSMAN`, update `log-confidence` call to use `logman.sh confidence`
- [x] T009 Update `fab/.kit/scripts/lib/preflight.sh` — add `--driver <skill-name>` flag that calls `logman.sh command` after validation, update `STAGEMAN` → `STATUSMAN` variable reference

## Phase 3: Reference Updates — Skills & Docs

- [x] T010 [P] Update shared skill files — `fab/.kit/skills/_preamble.md`: stageman → statusman references; `fab/.kit/skills/_scripts.md`: rewrite for 5-script architecture; `fab/.kit/skills/_generation.md`: stageman → statusman references
- [x] T011 [P] Update pipeline skill files — `fab/.kit/skills/fab-continue.md`, `fab/.kit/skills/fab-ff.md`, `fab/.kit/skills/fab-fff.md`: stageman → statusman, remove manual `log-command` and `log-review` lines, update preflight call to include `--driver`
- [x] T012 [P] Update remaining skill files — `fab/.kit/skills/fab-clarify.md`, `fab/.kit/skills/fab-new.md`, `fab/.kit/skills/fab-status.md`, `fab/.kit/skills/fab-archive.md`, `fab/.kit/skills/git-pr.md`: stageman → statusman, remove manual `log-command` where present, update preflight calls

## Phase 4: Test Suite & Memory

- [x] T013 Rename `src/lib/stageman/` → `src/lib/statusman/` — rename directory, rename `SPEC-stageman.md` → `SPEC-statusman.md`, update all internal test references from stageman to statusman, remove tests for `log-command`, `log-confidence`, `log-review` subcommands
- [x] T014 [P] Create `src/lib/resolve/` test directory — test all four output modes, substring/exact matching, `fab/current` fallback, single-change guessing, error cases
- [x] T015 [P] Create `src/lib/logman/` test directory — test each subcommand, verify JSON structure, append-only behavior
- [x] T016 [P] Update `src/lib/changeman/test.bats` — update stageman → statusman references, adjust resolve-related tests
- [x] T017 [P] Update `src/lib/calc-score/test.bats` — update for `<change>` input convention, verify DRY'd helpers
- [x] T018 [P] Update `src/lib/preflight/test.bats` — add tests for `--driver` flag
- [x] T019 Update memory files — `docs/memory/fab-workflow/kit-architecture.md` and `docs/memory/fab-workflow/execution-skills.md`: update script references, call graph, logging behavior. Search all memory files for "stageman" and update.

---

## Execution Order

- T001 (resolve.sh) blocks T002, T004, T007, T008, T009 (all depend on resolve.sh existing)
- T002 (logman.sh) blocks T005, T006, T007, T008, T009 (all depend on logman.sh existing)
- T003 (rename) blocks T004, T005, T006 (operate on statusman.sh)
- T004, T005 can run in parallel after T001, T003
- T006 depends on T002, T005 (logman exists, log functions removed)
- T007 depends on T001, T002 (uses resolve.sh and logman.sh)
- T008 depends on T001, T002 (uses resolve.sh and logman.sh)
- T009 depends on T002 (uses logman.sh)
- T010-T012 depend on T003 (need the new name to reference)
- T013-T018 depend on T001-T009 (test the implemented scripts)
- T019 depends on T003 (references the new script names)
