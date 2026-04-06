# Tasks: Operator tick-start and time subcommands for fab-go

**Change**: 260405-px5o-operator-tick-start-time-subcommands
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

## Phase 1: Refactor — operator parent command

<!-- Remove cobra.NoArgs, convert operatorCmd() to a parent command that registers subcommands. No new behavior yet. -->

- [x] T001 Refactor `operatorCmd()` in `src/go/fab/cmd/fab/operator.go`: remove `Args: cobra.NoArgs`, keep `RunE: runOperator` as default, add `cmd.AddCommand(...)` stubs for `tick-start` and `time` subcommands (stubs may return `nil` or a not-yet-implemented error — real logic added in Phase 2). Verify `go build ./...` passes.

## Phase 2: Core Implementation

<!-- Primary functionality. T002 (tick-start) and T003 (time) are independent and can run in parallel once T001 is done. -->

- [x] T002 [P] Implement `fab operator tick-start` subcommand in `src/go/fab/cmd/fab/operator.go`: resolve repo root via `gitRepoRoot()`, read `.fab-operator.yaml` into `map[string]interface{}` using `gopkg.in/yaml.v3` (treat missing file as empty map with `tick_count: 0`), increment `tick_count` by 1, set `last_tick_at` using `time.Now().UTC().Format(time.RFC3339)`, write updated map back to `.fab-operator.yaml` preserving all other fields, output `tick: N\nnow: HH:MM` to stdout using `time.Now().Format("15:04")`. Exit 1 with stderr error on write failure.

- [x] T003 [P] Implement `fab operator time` subcommand in `src/go/fab/cmd/fab/operator.go`: define `--interval` flag (string, no default), output `now: HH:MM` using `time.Now().Format("15:04")`, if `--interval` is provided parse with `time.ParseDuration()` (invalid → stderr error + exit 1), compute `next = time.Now().Add(parsedDuration)` and output `next: HH:MM` as a second line. No file I/O.

## Phase 3: Tests

<!-- Five new tests in operator_test.go. T004 and T005 (tick-start) depend on T002; T006–T008 (time) depend on T003. T004–T005 and T006–T008 sub-groups are independent of each other. -->

- [x] T004 [P] Add `TestOperatorTickStart_IncrementsCount` to `src/go/fab/cmd/fab/operator_test.go`: create a `t.TempDir()`, write `.fab-operator.yaml` with `tick_count: 5` and a dummy extra field (e.g. `monitored: {}`), invoke the `tick-start` command with the temp dir as repo root, verify stdout contains `tick: 6` and `now: \d\d:\d\d`, read back the YAML file and verify `tick_count == 6`, `last_tick_at` is a non-empty RFC3339 string, and the extra field is preserved.

- [x] T005 [P] Add `TestOperatorTickStart_MissingFile` to `src/go/fab/cmd/fab/operator_test.go`: use a `t.TempDir()` with no `.fab-operator.yaml`, invoke `tick-start`, verify stdout contains `tick: 1`, read back the created `.fab-operator.yaml` and verify `tick_count == 1` and `last_tick_at` is set.

- [x] T006 [P] Add `TestOperatorTime_NoInterval` to `src/go/fab/cmd/fab/operator_test.go`: invoke `fab operator time` with no flags, capture stdout, verify it contains exactly one line matching `now: \d\d:\d\d` and no `next:` line.

- [x] T007 [P] Add `TestOperatorTime_WithInterval` to `src/go/fab/cmd/fab/operator_test.go`: invoke `fab operator time --interval 3m`, capture stdout, verify it contains `now: \d\d:\d\d` and `next: \d\d:\d\d` (both lines present, both match HH:MM pattern).

- [x] T008 [P] Add `TestOperatorTime_InvalidInterval` to `src/go/fab/cmd/fab/operator_test.go`: invoke `fab operator time --interval notaduration`, verify the command exits with error (non-nil err or exit code 1) and writes an error to stderr.

- [x] T009 Update `TestOperatorCmd_Structure` in `src/go/fab/cmd/fab/operator_test.go` to also assert that the operator command has two registered subcommands named `tick-start` and `time` (verify via `cmd.Commands()` slice).

## Phase 4: Skill and memory updates

<!-- Documentation updates. All tasks in this phase are independent of each other [P]. -->

- [x] T010 [P] Update `src/kit/skills/fab-operator.md` — Tick Behavior step 1: replace the manual `increment tick_count` instruction with `run fab operator tick-start (increments tick_count, writes last_tick_at, outputs tick: N and now: HH:MM). Parse stdout for the tick number and current time. Then run fab pane map and read .fab-operator.yaml.`

- [x] T011 [P] Update `src/kit/skills/fab-operator.md` — Idle Message paragraph: replace the current "The time is HH:MM in the operator's local timezone..." sentence with the instruction to run `fab operator time --interval {interval}` (where `{interval}` is the current loop interval, e.g. `3m`) to obtain the `now:` and `next:` values for the idle message. Preserve the example idle message line `Waiting for next tick. Time: 08:26 · next tick: 08:29`.

- [x] T012 [P] Update `src/kit/skills/_cli-fab.md` — add `### fab operator tick-start` and `### fab operator time` subsections under the existing `## fab operator` section. Document: usage, output format, flags (`--interval` for `time`), side effects (`.fab-operator.yaml` for `tick-start`, none for `time`), and error behavior (invalid interval → exit 1). Preserve all existing `## fab operator` prose.

- [x] T013 [P] Update `docs/memory/fab-workflow/execution-skills.md` — in the operator section, add documentation for `fab operator tick-start` (purpose: start-of-tick atomic state update; output: `tick: N\nnow: HH:MM`; side effect: updates `.fab-operator.yaml` `tick_count` and `last_tick_at`) and `fab operator time` (purpose: pure clock query; flag: `--interval <duration>`; output: `now: HH:MM` and optionally `next: HH:MM`). Note their use in tick step 1 and idle message respectively. Note the separation of concerns: `tick-start` has side effects, `time` is pure.

- [x] T014 [P] Update `docs/memory/fab-workflow/kit-architecture.md` — update the `fab operator` entry in the Command Reference (line ~144 and ~312) to note that `fab operator` now has two subcommands: `tick-start` (start-of-tick state update + time output) and `time` (pure clock query with optional `--interval`). Also update the Testing entry (line ~319) to include `tick-start` and `time` in the tested subcommand list for `cmd/fab`.

---

## Execution Order

- T001 (refactor to parent command) blocks T002 and T003 — subcommand stubs must exist before implementation
- T002 blocks T004 and T005 — tick-start implementation must exist before its tests pass
- T003 blocks T006, T007, and T008 — time implementation must exist before its tests pass
- T009 (structure test update) can run any time after T001
- T010–T014 are independent of the Go implementation phases and of each other; they can be written in parallel or after Phase 3
- Run `go test ./src/go/fab/cmd/fab/...` after T009 to verify all tests pass before proceeding to Phase 4
