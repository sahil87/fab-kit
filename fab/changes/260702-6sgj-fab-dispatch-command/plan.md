# Plan: `fab dispatch` — headless process manager for CLI-dispatched pipeline stages

**Change**: 260702-6sgj-fab-dispatch-command
**Intake**: `intake.md`

## Requirements

### Dispatch: Command family

#### R1: `fab dispatch` command group
The `fab` binary SHALL expose a new top-level command group `fab dispatch` with five subcommands — `start`, `status`, `logs`, `kill`, `clean` — mirroring the multi-file `cmd/fab/pane*.go` structure and registered on the root command tree. Its top-level name MUST NOT collide with the router's workspace-command allowlist.

- **GIVEN** the assembled `fab` root command
- **WHEN** `fab dispatch --help` is run
- **THEN** the five subcommands are listed
- **AND** `dispatch` does not appear in the `_cli-fab.md` router allowlist (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `migrations-status`), so the collision test stays green

#### R2: POSIX-only guard
`fab dispatch start` SHALL error clearly on non-POSIX platforms (Windows) rather than half-working, with the message that dispatch requires a POSIX shell (`setsid`/`timeout`).

- **GIVEN** a Windows (`GOOS=windows`) build
- **WHEN** `fab dispatch start` is invoked
- **THEN** it returns an error naming the POSIX-shell requirement and does not launch anything

### Dispatch: State layout

#### R3: `.fab-dispatch/{id}/` state directory
Each dispatch's state SHALL live under `.fab-dispatch/{4-char-change-id}/` at the repository root, keyed by the stable 4-char change ID (not the slug). Per-stage files are `{stage}-prompt.md`, `{stage}.yaml`, `{stage}.log`, `{stage}.exit`, and `{stage}-result.yaml`. No gitignore/scaffold/migration work is required (the scaffold `.fab-*` pattern already matches).

- **GIVEN** a change resolving to ID `abcd` and stage `apply`
- **WHEN** `fab dispatch start abcd apply` runs
- **THEN** state is written under `.fab-dispatch/abcd/` with the `apply.yaml`, `apply-prompt.md` files present
- **AND** the directory is at the repo root (`filepath.Dir(fabRoot)`), one per worktree

### Dispatch: Start

#### R4: `fab dispatch start <change> <stage> [--timeout <secs>]`
`start` SHALL resolve `<change>` to its 4-char ID, read the stage prompt on stdin into `{stage}-prompt.md`, resolve the tier's spawn command via `internal/agent` + `internal/spawn`, launch it DETACHED via `sh -c '<cmd> < prompt > log 2>&1; echo $? > exit'` (launched with `setsid` semantics via `SysProcAttr{Setsid:true}` — the sole detach mechanism, so the recorded pid tracks the live worker shell) with cwd = repo root, and persist `{stage}.yaml` (pid, pgid, resolved spawn_cmd, started_at, timeout) via `internal/atomicfile` before returning.

- **GIVEN** a change/stage whose resolved tier carries a `spawn_command`
- **WHEN** `fab dispatch start <change> <stage>` is run with a prompt on stdin
- **THEN** the prompt is persisted, the command is launched detached in a new session/process group, and `{stage}.yaml` records the pid/pgid
- **AND** with `--timeout N`, the resolved command is wrapped in POSIX `timeout N <cmd>` inside the same wrapper (no Go timer, no daemon)

#### R5: No-spawn_command error (no fallback)
If the resolved tier has no `spawn_command`, `start` SHALL error clearly (naming the stage, the tier, and the `agent.tiers.<tier>.spawn_command` config key) and MUST NOT fall back to the top-level `agent.spawn_command`.

- **GIVEN** a stage resolving to a tier with no `spawn_command`
- **WHEN** `fab dispatch start` runs
- **THEN** it errors naming the tier and the config key to set, and launches nothing

#### R6: Refuse-if-running + last-attempt-only concurrency
`start` SHALL refuse if a dispatch for the exact `(change, stage)` pair is already `running` (reporting the live pid and directing to `fab dispatch kill`). A `start` over a completed prior attempt (done / failed / orphaned) SHALL overwrite its files — there is no per-attempt history.

- **GIVEN** a `(change, stage)` dispatch whose pid is alive and `{stage}.exit` is absent
- **WHEN** `fab dispatch start` is run again for the same pair
- **THEN** it refuses with a clear error and leaves the running dispatch untouched
- **AND** GIVEN a completed prior attempt, a new `start` overwrites the prior `{stage}.*` files

### Dispatch: Status

#### R7: Five-state status machine
`fab dispatch status <change> <stage> [--json]` SHALL read `{stage}.yaml`, `{stage}.exit`, and check pid liveness, then report exactly one of five byte-stable states: `running` (pid alive, no exit file), `done` (exit `0` AND `{stage}-result.yaml` present), `failed` (exit present and non-zero, including `124` timeout), `failed (no-result)` (exit `0` but result file absent — a contract violation, NOT done), `orphaned` (pid dead, no exit file).

- **GIVEN** a dispatch that exited `0` with a `{stage}-result.yaml` present
- **WHEN** `fab dispatch status` runs
- **THEN** it reports `done`
- **AND** GIVEN exit `0` with NO result file, it reports `failed (no-result)`
- **AND** GIVEN a non-zero exit (or `124`), it reports `failed`
- **AND** GIVEN pid dead with no exit file, it reports `orphaned`
- **AND** GIVEN pid alive with no exit file, it reports `running`

### Dispatch: Logs

#### R8: `fab dispatch logs <change> <stage> [--tail N]`
`logs` SHALL print `.fab-dispatch/{id}/{stage}.log`; `--tail N` prints the last N lines (implemented in Go, no external `tail`). A missing log SHALL produce a clear "no dispatch log" message.

- **GIVEN** a dispatch with a populated `{stage}.log`
- **WHEN** `fab dispatch logs <change> <stage> --tail 10` runs
- **THEN** the last 10 lines are printed
- **AND** a missing log yields the clear no-log message

### Dispatch: Kill

#### R9: `fab dispatch kill <change> <stage>`
`kill` SHALL terminate the process GROUP (`pgid` from `{stage}.yaml`) so the detached command and its children die together, and SHALL be idempotent — killing an already-dead dispatch is a benign no-op with a clear report.

- **GIVEN** a running dispatch with a recorded pgid
- **WHEN** `fab dispatch kill <change> <stage>` runs
- **THEN** the process group is signalled and the dispatch dies
- **AND** killing an already-dead dispatch reports a benign no-op

### Dispatch: Cleanup

#### R10: Two cleanup paths, no automatic GC
Cleanup SHALL happen at exactly two deterministic moments and never on a timer: (a) `fab change archive` deletes `.fab-dispatch/{id}/` as part of the archive move, and `fab change restore` does NOT recreate it; (b) `fab dispatch clean [<change>] [--orphans]` for manual cleanup.

- **GIVEN** a change with a `.fab-dispatch/{id}/` dir
- **WHEN** `fab change archive <change>` runs
- **THEN** `.fab-dispatch/{id}/` is deleted as part of the archive
- **AND** a subsequent `fab change restore` does not recreate it

#### R11: `fab dispatch clean [<change>] [--orphans]`
`clean` SHALL support: `clean <change>` (remove that change's dir), `clean` (remove all `.fab-dispatch/*/`), and `clean --orphans` (prune any `.fab-dispatch/{id}/` whose ID no longer resolves to a non-archived change).

- **GIVEN** several `.fab-dispatch/*/` dirs, one whose ID no longer resolves to an active change
- **WHEN** `fab dispatch clean --orphans` runs
- **THEN** only the orphaned dir is pruned
- **AND** `fab dispatch clean <change>` removes only the named change's dir, and `fab dispatch clean` removes all dirs

### Docs & Spec

#### R12: `_cli-fab.md` + SPEC mirror
The change SHALL add a `## fab dispatch` section to `src/kit/skills/_cli-fab.md` (each subcommand's signature/flags, the five states, POSIX-only) and add the matching row to its mirror `docs/specs/skills/SPEC-_cli-fab.md`.

- **GIVEN** the new CLI command family
- **WHEN** the change is reviewed
- **THEN** `_cli-fab.md` documents `fab dispatch` and `SPEC-_cli-fab.md` carries the corresponding inventory row

#### R13: `fab-archive` skill + SPEC archive/restore prose
The change SHALL update `src/kit/skills/fab-archive.md` and `docs/specs/skills/SPEC-fab-archive.md` so the archive mechanical-ops list gains the `.fab-dispatch/{id}/` deletion and the restore prose notes it is not recreated.

- **GIVEN** the archive-time deletion behavior
- **WHEN** the change is reviewed
- **THEN** both the skill and its SPEC mirror describe the `.fab-dispatch/` deletion on archive and the not-recreated-on-restore note

#### R14: Author `docs/specs/harness-adapters.md` + cross-references
The change SHALL author the new pre-implementation spec `docs/specs/harness-adapters.md` fixing the full cross-adapter dispatch protocol (both the native Agent-tool adapter and the CLI `fab dispatch` adapter; dispatch-prompt obligations incl. the `fab status refresh` epilogue; the five-state machine at the contract level; `review` nesting degradation; "hooks may enhance, never own"; and the "3d wires against this, amendments are explicit" marker). It SHALL add a cross-reference from `docs/specs/stage-models.md` § Harness-adapter boundary and a new row in `docs/specs/index.md`.

- **GIVEN** the shared dispatch contract 3c and 3d both depend on
- **WHEN** the change lands
- **THEN** `docs/specs/harness-adapters.md` exists with both adapters and the full protocol, `stage-models.md` points to it, and `index.md` lists it

### Non-Goals

- 3b's per-tier `spawn_command` config surface — consumed here, not defined here.
- 3d's skill-side dispatch-seam wiring, dispatch-prompt content, and nesting-degradation implementation — this change fixes the *contract* (spec), not the skill code.
- The `{stage}-result.yaml` content schema — only its path and presence obligation (via `failed (no-result)`) are fixed here.
- Any automatic garbage collection of `.fab-dispatch/`.
- `docs/memory/` edits — those are hydrate's job.

### Design Decisions

1. **Extract an `internal/dispatch` package**: state-dir read/write, wrapper composition, status-state derivation, and process-group signaling live in `internal/dispatch` so the logic is unit-testable independent of cobra wiring — *Why*: the `internal/pane`/`internal/archive` precedent; the status-state machine and wrapper composition are the testable core — *Rejected*: inline in `cmd/fab` (harder to table-test the state derivation without a launched process).
2. **Platform split for the launch/signal syscalls**: `dispatch_posix.go` (build tag `!windows`) owns `setsid`/`SysProcAttr.Setsid`, pgid liveness, and process-group kill; `dispatch_windows.go` (build tag `windows`) returns the POSIX-only error — *Why*: mirrors the `proc_{linux,darwin}.go` and `pane_process_{linux,darwin}.go` platform-split precedent; keeps the POSIX-only guard a compile-time reality, not a runtime string check — *Rejected*: a single file with a runtime `runtime.GOOS` check (would still compile Windows-incompatible syscalls).
3. **Reuse `syscall.Kill(pid, 0)` liveness probe**: the pid-liveness check reuses the exact POSIX-standard probe already in `internal/runtime.pidAlive` — *Why*: avoid duplicating a subtle EPERM/ESRCH probe — *Rejected*: `/proc` reads (Linux-only; the Kill probe is portable across POSIX).
4. **Archive deletion in `internal/archive.Archive`**: the `.fab-dispatch/{id}/` removal is added to `Archive()` (best-effort, after the folder move), computing the repo root as `filepath.Dir(fabRoot)` — *Why*: `Archive()` already owns the archive move and the repo-root derivation; keeping the deletion there keeps the two cleanup paths in the packages that own them — *Rejected*: a separate call from the cmd layer (splits the archive transaction).

## Tasks

### Phase 1: Setup

- [x] T001 Create `src/go/fab/internal/dispatch/dispatch.go` — package doc + core types (`Dispatch` state struct with `PID`/`PGID`/`SpawnCmd`/`StartedAt`/`Timeout` + file-path helpers), the `State` string constants (`running`/`done`/`failed`/`failed (no-result)`/`orphaned`), `DirFor(repoRoot, id)` and per-stage path helpers, YAML load/save via `internal/atomicfile`. <!-- R3 -->

### Phase 2: Core Implementation

- [x] T002 In `internal/dispatch/dispatch.go`, implement `DeriveState(d *Dispatch, exitPresent bool, exitCode int, resultPresent bool, alive bool) State` — the pure five-state derivation (R7), plus `ReadExit(path)` (returns present/code) and a `Tail(data []byte, n int)` helper for `logs`. <!-- R7 -->
- [x] T003 Add `src/go/fab/internal/dispatch/dispatch_posix.go` (build tag `!windows`) — `WrapperArgv(cmd, promptPath, logPath, exitPath string, timeoutSecs int) []string` composing `sh -c '...'` (with optional `timeout N`), `Launch(...)` using `exec.Command` + `SysProcAttr{Setsid:true}` (the setsid-semantics detach; returns pid/pgid on the live worker shell), `Alive(pid int) bool` (reuse the `syscall.Kill(pid,0)` EPERM/ESRCH probe), and `KillGroup(pgid int) error` (`syscall.Kill(-pgid, SIGTERM)`, ESRCH-benign). <!-- R4 -->
- [x] T004 Add `src/go/fab/internal/dispatch/dispatch_windows.go` (build tag `windows`) — the same function signatures returning the POSIX-only error (`Launch`/`KillGroup`), `Alive` conservatively false, so the package compiles on Windows and `start`/`kill` surface the clear error. <!-- R2 -->
- [x] T005 Create `src/go/fab/cmd/fab/dispatch.go` — the `dispatch` parent cobra command (Short/Long mirroring `pane.go`), adding the five subcommand constructors. <!-- R1 -->
- [x] T006 Create `src/go/fab/cmd/fab/dispatch_start.go` — `fab dispatch start <change> <stage> [--timeout N]`: resolve ID via `internal/resolve`, read stdin prompt, resolve tier via `internal/agent`/`internal/config` + `internal/spawn.WithProfile` (error on empty spawn_command per R5), refuse-if-running via `internal/dispatch` state (R6), launch detached and persist `{stage}.yaml`. <!-- R4 -->
- [x] T007 Register `dispatchCmd()` in `src/go/fab/cmd/fab/main.go` `newRootCmd()`. <!-- R1 -->
- [x] T008 Create `src/go/fab/cmd/fab/dispatch_status.go` — `fab dispatch status <change> <stage> [--json]`: load state, read exit, probe liveness, derive state, print byte-stable state string (or JSON). <!-- R7 -->
- [x] T009 Create `src/go/fab/cmd/fab/dispatch_logs.go` — `fab dispatch logs <change> <stage> [--tail N]`: print the log or the Go-side tail; clear missing-log message. <!-- R8 -->
- [x] T010 Create `src/go/fab/cmd/fab/dispatch_kill.go` — `fab dispatch kill <change> <stage>`: read pgid, kill the group idempotently, clear report. <!-- R9 -->
- [x] T011 Create `src/go/fab/cmd/fab/dispatch_clean.go` — `fab dispatch clean [<change>] [--orphans]`: remove named / all / orphaned `.fab-dispatch/*/` dirs (orphan = ID no longer resolves to a non-archived change). <!-- R11 -->
- [x] T012 In `src/go/fab/internal/archive/archive.go`, add `.fab-dispatch/{id}/` deletion to `Archive()` (best-effort, after the move) and confirm `Restore()` does not recreate it. <!-- R10 -->

### Phase 3: Tests

- [x] T013 [P] Add `src/go/fab/internal/dispatch/dispatch_test.go` — table-driven tests for `DeriveState` (all five states + edge combinations), `WrapperArgv` (with/without timeout), `Tail`, path helpers, and YAML round-trip. <!-- R7 -->
- [x] T014 [P] Add `src/go/fab/cmd/fab/dispatch_start_test.go` — refuse-if-running, no-spawn_command error, overwrite-completed-attempt, prompt persistence, timeout-in-wrapper composition. <!-- R4 --> <!-- R5 --> <!-- R6 -->
- [x] T015 [P] Add `src/go/fab/cmd/fab/dispatch_status_test.go`, `dispatch_logs_test.go`, `dispatch_clean_test.go` — status state derivation via the cmd layer, `--tail`, missing-log, and clean/`--orphans`. <!-- R7 --> <!-- R8 --> <!-- R11 -->
- [x] T016 Add an archive-side test in `src/go/fab/internal/archive/archive_test.go` asserting `Archive()` deletes a pre-existing `.fab-dispatch/{id}/` and `Restore()` does not recreate it. <!-- R10 -->
- [x] T017 Confirm `TestNoTopLevelCommandCollidesWithRouterAllowlist` stays green with the new `dispatch` command (no allowlist change needed). <!-- R1 -->

### Phase 4: Docs & Spec

- [x] T018 Add a `## fab dispatch` section to `src/kit/skills/_cli-fab.md` (mirroring `## fab pane`): the five subcommands with signatures/flags, the five states, refuse-if-running/last-attempt-only, timeout-in-wrapper, the two cleanup paths, POSIX-only. <!-- R12 -->
- [x] T019 Add the `fab dispatch` inventory row to `docs/specs/skills/SPEC-_cli-fab.md`. <!-- R12 -->
- [x] T020 Update `src/kit/skills/fab-archive.md` (archive mechanical-ops list gains `.fab-dispatch/{id}/` deletion; restore prose gains not-recreated note) and mirror in `docs/specs/skills/SPEC-fab-archive.md`. <!-- R13 -->
- [x] T021 Author `docs/specs/harness-adapters.md` — both dispatch adapters, the full dispatch protocol (prompt obligations + `fab status refresh` epilogue, five states, `review` nesting degradation, hooks-enhance-never-own), the "3d wires against this" marker. <!-- R14 -->
- [x] T022 Add the cross-reference from `docs/specs/stage-models.md` § Harness-adapter boundary to `harness-adapters.md`, and add the `harness-adapters` row to `docs/specs/index.md`. <!-- R14 -->

## Execution Order

- T001 blocks T002–T004 (same package types).
- T002–T004 block T005–T012 (cmd layer consumes the package).
- T007 depends on T005 (registers the parent).
- T013–T017 depend on their implementation tasks; T013/T014/T015 are `[P]` (different files).
- Phase 4 (T018–T022) is independent of the Go code and `[P]` among themselves.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab dispatch` exposes `start`/`status`/`logs`/`kill`/`clean` on the root command tree, split across `cmd/fab/dispatch*.go`. (dispatch.go adds all five subcommands; main.go registers `dispatchCmd()`.)
- [x] A-002 R2: `fab dispatch start` on Windows returns a clear POSIX-only error (the Windows-build stub returns it; not a runtime probe on POSIX). (`dispatch_windows.go` `errPOSIXOnly`; `GOOS=windows go build ./internal/dispatch/` compiles cleanly.)
- [x] A-003 R3: dispatch state is written under `.fab-dispatch/{4-char-id}/` at the repo root with the documented per-stage filenames. (`DirName=".fab-dispatch"`, `DirFor(repoRoot,id)`, path helpers; repoRoot = `filepath.Dir(fabRoot)`; TestPathHelpers.)
- [x] A-004 R4: `start` reads the prompt on stdin, resolves the tier spawn command, launches detached via `sh -c` with `setsid` semantics (`SysProcAttr{Setsid:true}`), and persists `{stage}.yaml` (pid/pgid/spawn_cmd/started_at/timeout); `--timeout N` wraps the command in POSIX `timeout N`. (dispatch_start.go + dispatch_posix.go WrapperArgv/Launch; TestDispatchStart_LaunchesAndPersistsState/_TimeoutWrapsCommand.)
- [x] A-005 R5: `start` errors (naming the tier + `agent.tiers.<tier>.spawn_command`) when the resolved tier has no spawn_command, with no fallback to `agent.spawn_command`. (dispatch_start.go:63-66; TestDispatchStart_NoSpawnCommandErrors.)
- [x] A-006 R6: `start` refuses when a `(change, stage)` dispatch is running and overwrites a completed prior attempt. (dispatch_start.go:72-106; TestDispatchStart_RefusesWhenRunning/_OverwritesCompletedAttempt.)
- [x] A-007 R7: `status` reports the correct one of the five states across all input combinations, byte-stable. (DeriveState + dispatch_status.go; TestDispatchStatus_States prints the bare state string.)
- [x] A-008 R8: `logs` prints the log and honors `--tail N` (Go-side), with a clear missing-log message. (dispatch_logs.go + Tail; TestDispatchLogs_PrintsAndTails/_MissingLogClearMessage.)
- [x] A-009 R9: `kill` signals the process group and is an idempotent benign no-op on a dead dispatch. (dispatch_kill.go + KillGroup `syscall.Kill(-pgid,SIGTERM)`, ESRCH-benign; TestDispatchKill_SignalsLiveGroup/_AlreadyDeadIsBenign.)
- [x] A-010 R10: `fab change archive` deletes `.fab-dispatch/{id}/` and `fab change restore` does not recreate it. (archive.go step 1b, best-effort after the move; TestArchive_DeletesDispatchState/TestRestore_DoesNotRecreateDispatchState.)
- [x] A-011 R11: `clean` handles the named-change, all-dirs, and `--orphans` modes correctly. (dispatch_clean.go; TestDispatchClean_NamedChange/_All/_Orphans/_NoState.)
- [x] A-012 R12: `_cli-fab.md` has a `## fab dispatch` section and `SPEC-_cli-fab.md` carries the matching row. (both present; row placed after `fab pane`, mirroring the skill section order.)
- [x] A-013 R13: `fab-archive.md` and `SPEC-fab-archive.md` describe the `.fab-dispatch/` archive-time deletion and not-recreated-on-restore note. (both updated; also swept in aggregate specs architecture.md + skills.md.)
- [x] A-014 R14: `docs/specs/harness-adapters.md` exists with both adapters + the full protocol; `stage-models.md` cross-references it; `docs/specs/index.md` lists it. (harness-adapters.md covers both adapters, prompt obligations incl. `fab status refresh` epilogue, five states, review nesting degradation, hooks-enhance-never-own, 3d-wires marker.)

### Behavioral Correctness

- [x] A-015 R7: the `failed (no-result)` state is distinct from `done` — exit `0` with no `{stage}-result.yaml` yields `failed (no-result)`, never `done`. (DeriveState:186-193; TestDeriveState "failed no-result" + TestDispatchStatus_States.)
- [x] A-016 R6: overwriting a completed prior attempt leaves no per-attempt history (files replaced in place). (dispatch_start.go:98-106 removes stale exit/result/log then Save overwrites {stage}.yaml; TestDispatchStart_OverwritesCompletedAttempt.)

### Scenario Coverage

- [x] A-017 R7: table-driven `DeriveState` test exercises all five states. (TestDeriveState — 11 cases spanning all five states + edge combos.)
- [x] A-018 R4: a test asserts `WrapperArgv` composes the `sh -c` form with and without `timeout N` (the detach is via the Setsid syscall attr, not a `setsid` binary prefix). (TestWrapperArgv asserts `[sh -c <script>]`, no `setsid` prefix; with/without timeout.)
- [x] A-019 R1: `TestNoTopLevelCommandCollidesWithRouterAllowlist` passes with the new `dispatch` command. (verified green; `dispatch` not in the router allowlist.)

### Edge Cases & Error Handling

- [x] A-020 R9: `kill` on an already-dead / missing dispatch is a benign no-op with a clear report (no error crash). (dispatch_kill.go:39-42 already-dead report; missing dispatch → clear "no dispatch" error; TestDispatchKill_AlreadyDeadIsBenign/_NoDispatchErrors.)
- [x] A-021 R8: `logs` on a missing log prints the clear "no dispatch log" message rather than erroring opaquely. (dispatch_logs.go:33-34; TestDispatchLogs_MissingLogClearMessage.)
- [x] A-022 R11: `clean --orphans` prunes only IDs that no longer resolve to a non-archived change, leaving live ones intact. (isOrphanedID via resolve.ToFolder which excludes archive/; TestDispatchClean_Orphans keeps the live dir, prunes the orphan.)

### Code Quality

- [x] A-023 Pattern consistency: new code follows the `pane*.go` / `internal/pane` split, cobra `RunE` conventions, and the platform-split (`_posix`/`_windows` build tags) precedent. (parent+child cmd files, `internal/dispatch` package, `dispatch_posix.go`/`dispatch_windows.go` build tags — mirrors pane/proc.)
- [x] A-024 No unnecessary duplication: the pid-liveness probe reuses the POSIX `syscall.Kill(pid,0)` pattern, spawn resolution reuses `internal/agent`/`internal/spawn`, and state writes reuse `internal/atomicfile` — no reimplementation. (spawn/agent/atomicfile all reused; `Alive` re-implements the probe because `internal/runtime.pidAlive` is unexported and importing runtime would be a heavy dependency — same logic, faithful copy, acknowledged in the plan/comments. See should-fix note.)
- [x] A-025 No magic strings: the five state strings and the `.fab-dispatch` dir/file-name components are named constants in `internal/dispatch`. (State consts + DirName + promptSuffix/yamlSuffix/logSuffix/exitSuffix/resultSuffix.)
- [x] A-026 Go changes ship tests: every new `.go` implementation file has table-driven `*_test.go` coverage (test-alongside). (dispatch_test.go + dispatch_{start,status,logs,kill,clean}_test.go + archive_test.go additions.)

### Documentation Accuracy

- [x] A-027 R12 R13: `_cli-fab.md`, `fab-archive.md`, and both SPEC mirrors accurately describe the shipped command signatures, states, and archive behavior (no drift between prose and code). (signatures, error strings, five states, timeout-in-wrapper, and archive deletion all match the code verbatim.)

### Cross References

- [x] A-028 R14: cross-references between `harness-adapters.md`, `stage-models.md`, and `docs/specs/index.md` resolve and are bidirectionally coherent (stage-models points to harness-adapters; index lists it). (stage-models.md § Harness-adapter boundary links harness-adapters.md; harness-adapters.md links back to stage-models.md; index.md row present.)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. The `fab pane` / `fab operator` interactive path is explicitly retained as a parallel surface (intake Why + § What Changes §1), and the new `internal/dispatch` package introduces a distinct runtime rather than superseding any existing one.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Extract an `internal/dispatch` package (state read/write, wrapper composition, five-state derivation, process signaling) with thin `cmd/fab/dispatch*.go` cobra wiring | Intake Open Question left this as an apply-time decision; the `internal/pane`/`internal/archive` precedent and the need to table-test the pure state machine make extraction the clear default | S:75 R:80 A:85 D:80 |
| 2 | Confident | Platform split via `dispatch_posix.go` (`!windows`) + `dispatch_windows.go` (`windows`) build tags for the launch/signal syscalls; Windows stub returns the POSIX-only error | Mirrors the `proc_{linux,darwin}.go` / `pane_process_{linux,darwin}.go` precedent; makes the POSIX-only guard a compile-time reality rather than a runtime string check | S:80 R:75 A:85 D:80 |
| 3 | Confident | Reuse the `syscall.Kill(pid,0)` EPERM/ESRCH liveness probe (as in `internal/runtime.pidAlive`) for dispatch liveness; kill via `syscall.Kill(-pgid, SIGTERM)` | Codebase already has the exact portable probe; duplicating a `/proc` reader would be Linux-only and against the reuse anti-pattern | S:80 R:75 A:90 D:85 |
| 4 | Confident | Archive-time `.fab-dispatch/{id}/` deletion lives in `internal/archive.Archive()` (best-effort, after the move), repo root = `filepath.Dir(fabRoot)` | `Archive()` owns the archive move and already derives repo root that way for the pointer clear; keeping deletion there keeps cleanup in the owning package | S:80 R:75 A:85 D:80 |
| 5 | Confident | `{stage}.yaml` fields = `pid`, `pgid`, `spawn_cmd`, `started_at`, `timeout` (secs, omitted when unset), plus derived file paths computed (not stored) — matching the intake's file table | Intake §2 specifies these fields verbatim; storing derived paths would duplicate the dir-convention | S:85 R:75 A:85 D:85 |
| 6 | Confident | `kill` sends SIGTERM (not SIGKILL) to the process group; ESRCH is treated as a benign already-dead no-op | Intake says "kills the process group"; SIGTERM is the graceful POSIX default and matches "die together"; idempotency requires ESRCH-benign handling | S:70 R:70 A:80 D:70 |
| 7 | Confident | `clean --orphans` resolves each dir's 4-char ID against non-archived changes via `internal/resolve` (ID→folder); a dir whose ID fails to resolve is pruned | Intake §7(b) defines orphan as "ID no longer resolves to a non-archived change"; `internal/resolve` is the established ID→change resolver | S:80 R:75 A:85 D:80 |
| 8 | Confident | The detach is done by Go's `SysProcAttr{Setsid:true}` on a plain `sh -c '...'` wrapper, NOT by prefixing the `setsid` binary — the intake's `setsid sh -c` string describes the *intent* (new session, survives orchestrator death), which the Setsid syscall attr delivers while keeping the recorded pid on the live worker shell | An end-to-end smoke test showed the `setsid` binary double-forks (its caller is already a process-group leader under Setsid), so the Go-recorded pid pointed at an immediately-exiting `setsid` process — breaking liveness/refuse-if-running/kill. One detach mechanism (the trackable one) is the correctness fix; the observable behavior (detached, resumable) matches the intake exactly | S:80 R:65 A:85 D:80 |

8 assumptions (0 certain, 8 confident, 0 tentative).
