# Plan: Shared-State Concurrency & Hook Hot Path

**Change**: 260612-mz4q-shared-state-concurrency-hook-hot-path
**Intake**: `intake.md`

## Requirements

### Concurrency: Cross-Process Locking (F01, F03)

#### R1: Shared advisory-lock helper
A new small package `src/go/fab/internal/lockfile` SHALL provide exclusive advisory locking for fab's shared state files. The lock MUST be a sibling file (`<guarded-path>.lock`) opened with `os.OpenFile(O_CREATE)` and locked via `syscall.Flock(fd, LOCK_EX)` (works on both supported GOOS — linux and darwin; no build-tag split). Acquisition MUST be bounded (non-blocking flock with retry up to a timeout) so a pathological holder (e.g., a stage hook re-invoking `fab status` on the same change) produces a clear error instead of an indefinite deadlock. Lock files are never deleted — flock state, not file existence, carries the lock.

- **GIVEN** two processes calling `lockfile.WithLock` on the same path
- **WHEN** both attempt the load-mutate-save cycle concurrently
- **THEN** the cycles serialize — the second waits until the first releases
- **AND** an uncontended acquisition succeeds immediately (single syscall, no retry latency)

#### R2: `.fab-runtime.yaml` mutators are lock-serialized and write-skipping
All four `.fab-runtime.yaml` mutators (`WriteAgent`, `ClearAgent`, `ClearAgentIdle`, `GCIfDue`) MUST perform their load-mutate-save cycle while holding the `.fab-runtime.yaml.lock` exclusive lock, eliminating the lost-update race between concurrent hook invocations in the same worktree. The save MUST be skipped entirely when nothing changed — today's write-free paths (`ClearAgent` with no entry, `ClearAgentIdle` with no `idle_since`) stay write-free.

- **GIVEN** N concurrent `WriteAgent` calls for N distinct session IDs on one worktree
- **WHEN** all complete
- **THEN** `.fab-runtime.yaml` contains all N entries (no silently lost update)

- **GIVEN** a runtime file without an entry for session S
- **WHEN** `ClearAgent` runs for S (GC throttled)
- **THEN** the file content is byte-identical (no write occurred)

#### R3: `.status.yaml` load-mutate-save cycles are lock-serialized
The `.status.yaml` write paths MUST serialize under the same lock helper, using the sibling `fab/changes/<folder>/.status.yaml.lock`: (a) every mutating `fab status` subcommand entry point in `cmd/fab/status.go` (start, advance, finish, reset, skip, fail, set-change-type, set-acceptance, set-confidence, set-confidence-fuzzy, add-issue, add-pr), (b) the artifact-write hook's bookkeeping (load → mutate → single save), and (c) `score.Compute`'s load → set-confidence → save cycle. Read-only commands stay lock-free (rename atomicity already protects readers).

- **GIVEN** the artifact-write hook holding a loaded `.status.yaml` snapshot
- **WHEN** `fab status finish` runs concurrently in another pane on the same change
- **THEN** the two load-mutate-save cycles serialize and neither side's fields are reverted by a stale-snapshot overwrite

### Hook Hot Path: Single-Save Bookkeeping (F02)

#### R4: One `.status.yaml` save per artifact-write hook event
`status.SetAcceptance`/`SetChangeType` mutation logic SHALL be split from persistence via non-saving variants (`ApplyAcceptance`, `ApplyChangeType`) that keep validate-before-mutate semantics; the existing `Set*` functions remain as Apply+Save wrappers (CLI contract unchanged). `artifactBookkeeping` MUST mutate the in-memory `StatusFile` only and the hook MUST call `statusFile.Save` exactly once at the end, and only when something was mutated. External contracts MUST be unchanged: `additionalContext` JSON shape, final `.status.yaml` state, `fab status set-acceptance` CLI signature, exit-0 hook semantics.

- **GIVEN** a `plan.md` write with both `## Tasks` and `## Acceptance` present
- **WHEN** `fab hook artifact-write` processes it
- **THEN** all four plan fields are updated in memory and persisted by a single Save (one serialize+rename, one `last_updated` bump)

- **GIVEN** an invalid change type passed to `ApplyChangeType`
- **WHEN** the call validates
- **THEN** it errors without mutating the in-memory StatusFile (validate-before-mutate preserved)

#### R5: Intake branch resolves once, loads once, reads once
The intake.md hook branch MUST NOT re-resolve the change folder or re-load `.status.yaml` or re-read `intake.md`. A new `score.ComputeWithStatus(fabRoot, changeDir, intakeContent, statusFile)` entry point SHALL reuse the already-resolved folder, already-read intake content, and already-loaded `StatusFile`; it mutates `statusFile.Confidence` in memory (caller owns persistence and locking) and logs the confidence event. `log.ConfidenceLog` SHALL accept the already-resolved absolute change dir instead of re-resolving (callers use `filepath.Join`/existing resolution; the `fab log confidence` CLI resolves once in the cmd layer — CLI signature unchanged).

- **GIVEN** an `intake.md` write event
- **WHEN** the hook runs
- **THEN** `fab/changes` is directory-scanned at most once (the existing hook-entry resolution), `.status.yaml` is loaded once and saved once, and `intake.md` is read once
- **AND** the computed score and change type land in `.status.yaml` identical to today's final state

### Runtime File: Merged Mutation + GC, Durability Posture (F04, F03)

#### R6: `UpdateAgent` merges entry mutation and GC into one load/save
A new `runtime.UpdateAgent(fabRoot, createIfMissing, mutate, gcInterval)` SHALL load `.fab-runtime.yaml` once (under the lock), apply the entry mutation, run the GC sweep inline when due (`now - last_run_gc >= interval`; sweep is pure in-memory + `kill(pid,0)`), and save once. GC MUST run even when the mutation half is a no-op (GC-on-no-op semantics). The save MUST be skipped when neither the entry nor GC changed anything. Hook handlers (`stop`, `session-start`, `user-prompt`) MUST make a single runtime call each (mutation + GC folded), replacing the current mutator+`GCIfDue` pairs. When `createIfMissing` is false and the file is absent, the call is a complete no-op (today's ClearAgent/ClearAgentIdle/GCIfDue posture); the stop path creates the file as today.

- **GIVEN** a stop hook event with the GC throttle expired and a dead-PID entry present
- **WHEN** the handler runs
- **THEN** one load and one save occur, the new entry is written, the dead entry is swept, and `last_run_gc` is updated — all in the same write

- **GIVEN** a user-prompt event for a session with no `idle_since` and GC throttled
- **WHEN** the handler runs
- **THEN** no write occurs at all

- **GIVEN** a session-start event for an absent session ID while GC is due
- **WHEN** the handler runs
- **THEN** the GC sweep still executes (GC-on-no-op) and the file is saved once because `last_run_gc` changed

#### R7: Durability follows criticality
`runtime.SaveFile` SHALL drop its per-write `fsync` (`tmpFile.Sync()`) — the file is ephemeral and fully re-derivable, and the fsync sits on every hook event's latency path. `statusfile.Save` SHALL gain `tmp.Sync()` before Close — `.status.yaml` is the pipeline state machine's source of truth (constitution Principle III).

- **GIVEN** a crash between temp-write and rename of `.status.yaml`
- **WHEN** `statusfile.Save` has synced the temp file before rename
- **THEN** the visible `.status.yaml` is never empty/torn — it is either the old or the new complete document

### Active Pointer: Atomic Swap + Target Validation (F05, F08)

#### R8: Atomic `.fab-status.yaml` pointer swap
A shared `setActivePointer` helper in `internal/change` SHALL create the symlink at a unique temp name in the repo root and `os.Rename` it over `.fab-status.yaml` — rename atomically replaces the old link on POSIX, eliminating both the empty-pointer window and the concurrent-Switch EEXIST race. Both `Switch` and `Rename` MUST use it (Rename keeps its best-effort posture; on failure the old pointer now remains in place instead of no pointer at all).

- **GIVEN** an existing `.fab-status.yaml` pointing at change A
- **WHEN** `fab change switch B` runs while a concurrent reader resolves the pointer
- **THEN** the reader sees either A's or B's pointer — never a missing pointer, and the switch never fails with EEXIST

#### R9: Dangling pointer target is detected and falls through
`resolveFromCurrent` SHALL `os.Stat` the target `fab/changes/<folder>/.status.yaml` after extracting the folder name from the symlink. On failure it MUST treat the pointer as stale and fall through to the existing no-active-change/single-change-fallback logic (actionable `/fab-switch` guidance). The stale link is NOT removed — `fab resolve` is documented as a pure query with no side effects.

- **GIVEN** a `.fab-status.yaml` pointing at an archived/deleted change folder and two other changes present
- **WHEN** `fab resolve` runs
- **THEN** it exits non-zero with `No active change (multiple changes exist — use /fab-switch).` instead of silently printing the stale folder

- **GIVEN** the same dangling pointer with exactly one valid change present
- **WHEN** `fab resolve` runs
- **THEN** the single change resolves with the `(resolved from single active change)` stderr note

#### R10: Archive captures the pointer before the folder rename (prerequisite to R9)
`archive.Archive` MUST determine whether the archived change is the active one by reading the symlink directly (`os.Readlink` + `ExtractFolderFromSymlink`) BEFORE renaming the folder into `archive/`, then clear the pointer after a successful move. Without this reorder, R9's stat-and-fall-through would make the post-rename resolution miss the dangling pointer and `fab-archive`'s `pointer:` output would regress from `cleared` to `skipped`.

- **GIVEN** the active change being archived
- **WHEN** `fab change archive <change>` completes
- **THEN** the result reports `pointer: cleared` and the symlink is removed (same observable behavior as today, now via pre-rename capture)

### Error Truth-Telling (F06, F07)

#### R11: Classified `.status.yaml` read errors
`statusfile.Load` MUST keep the friendly `status file not found: <path>` text only for `os.IsNotExist` failures and wrap every other read failure with its cause (`read status file <path>: <err>`), so permission-denied/is-a-directory/I/O errors stop masquerading as absence. The `change.ListWithOptions` warning MUST echo the actual load error so corruption (e.g., git merge-conflict markers) is distinguishable from absence. No `_cli-fab.md` change needed (its error table documents resolve.go strings only).

- **GIVEN** an existing but unreadable (permission-denied) `.status.yaml`
- **WHEN** any status-touching command loads it
- **THEN** the error names the real cause and does not claim the file is missing

- **GIVEN** a `.status.yaml` containing merge-conflict markers
- **WHEN** `fab change list` scans it
- **THEN** the stderr warning carries the YAML parse error, not "not found"

#### R12: No silent dropped `.status.yaml` writes
`syncToRaw` SHALL insert absent keys on write (generalizing the `insertTrueImpact` pattern) for every command-mutated key — `issues`, `prs`, `plan`, `confidence`, `stage_metrics`, `last_updated` always; `name`/`change_type` when the struct value is non-empty — so writes to well-formed-but-sparse documents persist. `SetProgress` SHALL create a missing stage key in an existing `progress:` mapping, and SHALL return an error when the document shape is malformed (`progress:` absent or not a mapping); the transition functions in `internal/status` (Start/Advance/Finish/Reset/Skip/Fail) MUST propagate that error so callers can't exit 0 on a dropped transition. This is the write-time-insertion variant — NOT validate-at-Load refusal, which would conflict with the documented legacy tolerance posture.

- **GIVEN** a restored pre-0.24.0 `.status.yaml` missing the `prs:` key
- **WHEN** `fab status add-pr` runs
- **THEN** the URL is persisted (key inserted) and the command exits 0 truthfully

- **GIVEN** a `.status.yaml` whose `progress:` mapping lacks the `apply` key
- **WHEN** `fab status start <change> apply` runs
- **THEN** the `apply: active` entry is created and persisted

- **GIVEN** a `.status.yaml` with no `progress:` mapping at all
- **WHEN** any transition command runs
- **THEN** it exits non-zero with a malformed-shape error and does not write a half-consistent state

### Repo Plumbing & Docs

#### R13: `.gitignore` entries for lock siblings — dev repo AND distribution surface
The repo `.gitignore` SHALL gain explicit entries for the new lock siblings: `.fab-runtime.yaml.lock` (already matched by the existing `.fab-*` pattern, made explicit per the binding verifier correction) and `.status.yaml.lock` (matches the per-change siblings under `fab/changes/`). The scaffold gitignore fragment shipped to user projects (`src/kit/scaffold/fragment-.gitignore`) SHALL likewise cover `.status.yaml.lock` — its existing `.fab-*` line covers the runtime lock but NOT the per-change status lock, which `withStatusLock`, `score.Compute`, and the artifact-write hook create in every user project. <!-- rework cycle 1: R13 originally specified only the dev repo .gitignore; review found the user-project distribution surface uncovered -->

- **GIVEN** lock files created by normal operation
- **WHEN** `git status` runs (dev repo or a user project scaffolded/synced by fab)
- **THEN** no `.lock` sibling appears as untracked

#### R14: `_cli-fab.md` error-table rows updated for dangling pointers
`src/kit/skills/_cli-fab.md`'s Common Error Messages rows that currently say "symlink absent" SHALL be extended to cover the dangling-target case (symlink present but target `.status.yaml` missing → same fall-through behavior). No other documented message changes (F06's strings are not in the table). The deployed copy under `.claude/skills/` is NOT edited.

- **GIVEN** the updated error table
- **WHEN** an agent reads the `No active change...` rows
- **THEN** the documented causes include both the absent and the dangling pointer

### Non-Goals

- `change.Rename`'s own `.status.yaml` name update and other low-frequency writers (prmeta, operator state) are not lock-wrapped — scope is the F01/F03 surfaces (runtime mutators, status CLI mutators, hook, score)
- No validate-at-Load schema refusal (explicitly rejected by the verifier as conflicting with legacy tolerance)
- No memory-doc edits (`runtime-agents.md`, `change-lifecycle.md`) — that is hydrate-stage work
- `lookupTransition`/`AllowedStates` in `internal/status/status.go` are untouched (k4ge seam)

### Design Decisions

1. **Bounded lock acquisition (LOCK_NB + retry, ~10s timeout)** — *Why*: a blocking `LOCK_EX` held across `status.Start/Finish`'s stage-hook execution could deadlock forever if a configured stage hook invokes `fab status` on the same change; a bounded wait converts that to a clear error while the uncontended path stays a single syscall. — *Rejected*: unbounded blocking flock (silent deadlock risk in unattended pipelines).
2. **`ComputeWithStatus` does not save** — *Why*: the hook's "exactly one Save" mandate requires the caller to own persistence; standalone `Compute` wraps lock+load+save itself. — *Rejected*: a saving variant (would re-introduce the double save).
3. **Per-op runtime functions gain a `gcInterval` parameter (≤0 = no GC) atop the exported `UpdateAgent` core** — *Why*: hooks make one typed call each without duplicating mutation logic; tests keep targeted per-op entry points. — *Rejected*: exporting raw map mutators (leaks schema internals into cmd).
4. **Stale pointer never removed by resolve** — *Why*: `fab resolve` is documented as a pure query with no side effects; fall-through alone restores actionable guidance. — *Rejected*: the intake's optional remove-on-stale.

## Tasks

### Phase 1: Setup

- [x] T001 Create `src/go/fab/internal/lockfile/lockfile.go` (`Lock(path) (unlock, error)` + `WithLock(path, fn)`, sibling `<path>.lock`, `O_CREATE`+`Flock(LOCK_EX|LOCK_NB)` retry loop with bounded timeout) with `lockfile_test.go` covering acquisition, release, mutual exclusion between two open descriptors, and timeout error <!-- R1 -->
- [x] T002 [P] Add explicit `.gitignore` entries for `.fab-runtime.yaml.lock` and `.status.yaml.lock` with a lock-siblings comment <!-- R13 -->

### Phase 2: Core Implementation

- [x] T003 Rework `src/go/fab/internal/runtime/runtime.go`: locked single load/save core `UpdateAgent(fabRoot, createIfMissing, mutate, gcInterval)`, pure in-memory `gcSweepIfDue`, re-express `WriteAgent`/`ClearAgent`/`ClearAgentIdle` as wrappers taking `gcInterval` (≤0 disables GC) and `GCIfDue` over the core, skip-save-when-unchanged, drop `tmpFile.Sync()` from `SaveFile`; update `runtime_test.go` (signature updates + new tests: GC-on-no-op, write-skip, merged single-write, concurrent no-lost-update) <!-- R2, R6, R7 -->
- [x] T004 Update `src/go/fab/cmd/fab/hook.go` session hooks (`stop`, `session-start`, `user-prompt`) to a single merged runtime call each (mutator + 180s GC), replacing the `GCIfDue` pairs <!-- R6 -->
- [x] T005 [P] Add `tmp.Sync()` before Close in `src/go/fab/internal/statusfile/statusfile.go` `Save` <!-- R7 -->
- [x] T006 Add non-saving `ApplyChangeType`/`ApplyAcceptance` (and `ApplyConfidence`/`ApplyConfidenceFuzzy`) to `src/go/fab/internal/status/status.go`, with `SetChangeType`/`SetAcceptance`/`SetConfidence`/`SetConfidenceFuzzy` becoming Apply+Save wrappers (validate-before-mutate preserved); tests in `status_test.go` <!-- R4 -->
- [x] T007 Add `score.ComputeWithStatus(fabRoot, changeDir, intakeContent, statusFile)` to `src/go/fab/internal/score/score.go` (in-memory confidence mutation + confidence log, no save), refactor `countGrades` to scan from bytes, restructure `Compute` as resolve → read intake → lock → load → ComputeWithStatus → save; change `log.ConfidenceLog` to take the resolved absolute change dir (update `cmd/fab/log.go` to resolve once); update `score_test.go`/`log_test.go` <!-- R5, R3 -->
- [x] T008 Rework `src/go/fab/cmd/fab/hook.go` artifact-write: wrap load→bookkeeping→save in `lockfile.WithLock(statusPath, ...)`, `artifactBookkeeping` returns `(contextParts, dirty)` using Apply variants + `ComputeWithStatus` (single resolution, single intake read), hook saves exactly once when dirty; update `hook_test.go` (existing behavior preserved + single-save coverage) <!-- R3, R4, R5 -->
- [x] T009 Add `withStatusLock` helper in `src/go/fab/cmd/fab/status.go` and route the 12 mutating subcommands through it (read-only commands keep `loadStatus`) <!-- R3 -->
- [x] T010 Extract `setActivePointer(repoRoot, target)` in `src/go/fab/internal/change/change.go` (temp symlink + `os.Rename`), use it in `Switch` and `Rename`; test that switching over an existing pointer works and replaces atomically (no Remove-then-Symlink) in `change_test.go` <!-- R8 -->
- [x] T011 Reorder `src/go/fab/internal/archive/archive.go` `Archive` to capture the active pointer via direct `os.Readlink`+`ExtractFolderFromSymlink` before the folder rename, clearing it after a successful move; keep `pointer: cleared|skipped` outputs (existing `archive_test.go` pointer tests must stay green) <!-- R10 -->
- [x] T012 Add dangling-target validation in `src/go/fab/internal/resolve/resolve.go` `resolveFromCurrent` (stat target `.status.yaml`, fall through on failure, no removal); tests for dangling+multiple, dangling+single, dangling+zero in `resolve_test.go` <!-- R9 -->
- [x] T013 Classify read errors in `src/go/fab/internal/statusfile/statusfile.go` `Load` (IsNotExist keeps friendly text, others wrapped with `%w`) and echo the actual load error in `src/go/fab/internal/change/change.go` `ListWithOptions` warning; tests for permission-denied and parse-error paths <!-- R11 -->
- [x] T014 Generalize key insertion in `syncToRaw` (insert absent `issues`/`prs`/`plan`/`confidence`/`stage_metrics`/`last_updated`; `name`/`change_type` when non-empty), make `SetProgress` create missing stage keys and return a malformed-shape error, propagate it through Start/Advance/Finish/Reset/Skip/Fail in `internal/status/status.go`; tests: sparse-file add-pr persists, missing-stage start persists, missing-progress errors <!-- R12 -->

### Phase 3: Integration & Edge Cases

- [x] T015 Concurrency integration tests: parallel `UpdateAgent` writers produce no lost entries (`internal/runtime`); parallel status mutators through the lock produce a consistent final document (`internal/lockfile` or cmd-level test) <!-- R2, R3 -->
- [x] T016 Run full module tests (`go test ./...` in `src/go/fab`, then `src/go/fab-kit`) and `go vet ./...`; fix regressions <!-- R1-R12 -->

### Phase 4: Polish

- [x] T017 Update `src/kit/skills/_cli-fab.md` Common Error Messages rows for the dangling-pointer fall-through (absent **or dangling** symlink) — canonical kit source only, not deployed copies <!-- R14 -->
- [x] T018 Add `.status.yaml.lock` to `src/kit/scaffold/fragment-.gitignore` so user projects ignore the per-change lock siblings (the fragment's `.fab-*` line already covers the runtime lock; propagates via the existing `fab sync` line-ensure merge, no migration) <!-- R13 --> <!-- rework: review cycle 1 must-fix — fragment missed, lock files would be committed by autonomous ship flows in user projects -->
- [x] T019 Fix goimports import grouping in `src/go/fab/internal/status/status_test.go` and `src/go/fab/internal/score/score_test.go` (project imports interleaved in the stdlib group); re-run `gofmt -l` + tests for the two packages <!-- R1-R12 --> <!-- rework: review cycle 1 should-fix — clear and low-effort, pattern consistency -->

## Execution Order

- T001 blocks T003, T007, T008, T009 (all consume `lockfile`)
- T006 blocks T007 and T008 (Apply variants consumed by score/hook)
- T007 blocks T008 (hook consumes `ComputeWithStatus`)
- T003 blocks T004
- T011 (archive reorder) must land with or before T012 (dangling-target validation)
- T002, T005, T010, T013, T014, T017 are independent

## Acceptance

### Functional Completeness

- [x] A-001 R1: `internal/lockfile` exists with sibling-`.lock` exclusive flock, bounded acquisition, and unit tests
- [x] A-002 R2: all four runtime mutators run their load-mutate-save under the lock and skip the save when nothing changed
- [x] A-003 R3: mutating `fab status` subcommands, the artifact-write hook, and `score.Compute` serialize `.status.yaml` cycles via the lock helper
- [x] A-004 R4: a plan.md hook event performs exactly one `.status.yaml` Save; Apply variants validate before mutating
- [x] A-005 R5: the intake hook branch performs one folder resolution, one status load, one intake read; `ComputeWithStatus` and dir-taking `ConfidenceLog` exist
- [x] A-006 R6: `UpdateAgent` merges mutation+GC into one load/save with GC-on-no-op and skip-save-when-unchanged; hooks make one runtime call each
- [x] A-007 R7: `runtime.SaveFile` no longer fsyncs; `statusfile.Save` syncs the temp file before rename
- [x] A-008 R8: `setActivePointer` (temp symlink + rename) is used by both `Switch` and `Rename`
- [x] A-009 R9: `resolveFromCurrent` stats the pointer target and falls through to fallback logic on a dangling pointer without removing the link
- [x] A-010 R10: `Archive` captures/clears the active pointer before the folder rename; `pointer: cleared` still reported for the active change
- [x] A-011 R11: `statusfile.Load` wraps non-IsNotExist errors with their cause; `change list` warning echoes the actual load error
- [x] A-012 R12: writes to sparse documents persist via key insertion; `SetProgress` creates missing stage keys and errors on malformed `progress:`; transition commands propagate the error
- [x] A-013 R13: `.gitignore` covers both lock-sibling patterns
- [x] A-032 R13: the scaffold gitignore fragment (`src/kit/scaffold/fragment-.gitignore`) shipped to user projects covers `.status.yaml.lock`, so user-project lock siblings are never committed
- [x] A-014 R14: `_cli-fab.md` error-table rows document the dangling-pointer case

### Behavioral Correctness

- [x] A-015 R4: external contracts unchanged — `additionalContext` JSON shape, final `.status.yaml` state, `fab status set-acceptance` signature, exit-0 hook semantics
- [x] A-016 R6: today's write-free hook paths remain write-free (no new writes on no-op clear with throttled GC)
- [x] A-017 R9: `fab resolve` no longer prints a stale folder for a dangling pointer (silent-wrong-answer fixed)
- [x] A-018 R12: a missing-stage `fab status start` no longer persists `stage_metrics`/`.history.jsonl` while dropping the progress transition (inconsistent-state bug gone)

### Scenario Coverage

- [x] A-019 R2: concurrent-writer test demonstrates no lost `_agents` entries under parallel mutation
- [x] A-020 R6: merged-path test demonstrates a due GC and an entry write landing in a single save
- [x] A-021 R9: dangling-pointer tests cover zero/one/multiple candidate fall-through outcomes

### Edge Cases & Error Handling

- [x] A-022 R1: lock acquisition timeout yields a clear error (no indefinite hang)
- [x] A-023 R11: permission-denied and YAML-parse failures produce cause-bearing errors distinct from absence
- [x] A-024 R12: malformed `progress:` shape errors instead of silently dropping the transition

### Code Quality

- [x] A-025 Pattern consistency: new code follows naming and structural patterns of surrounding code (temp+rename convention, swallow-on-error hook posture, table-driven tests, goimports import grouping) <!-- rework: unchecked in cycle 1 — import grouping finding; re-verify after T019 --> <!-- review cycle 2: T019 verified — gofmt -l clean, stdlib-then-project groups in status_test.go/score_test.go and all other touched files -->
- [x] A-026 No unnecessary duplication: one lock helper serves runtime and statusfile; Apply/Set split reuses single mutation bodies; `insertTrueImpact` generalized rather than cloned
- [x] A-027 No god functions: the runtime core, syncToRaw insertion, and hook bookkeeping stay focused; no function balloons past ~50 lines without clear reason
- [x] A-028 No magic numbers: lock timeout/retry interval and GC sentinel are named constants

### Documentation Accuracy

- [x] A-029: code comments describing locking, GC piggyback, and durability posture match actual behavior; `_cli-fab.md` rows match resolve.go behavior verbatim
- [x] A-030: no edits to deployed `.claude/skills/` copies or to `docs/memory/` (hydrate-stage work)

### Cross References

- [x] A-031: task/requirement trace annotations (`<!-- R# -->`) are consistent, and the k4ge seam is respected — `lookupTransition`/`AllowedStates` untouched

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Memory-doc impacts (runtime-agents.md:75 atomicity claim, change-lifecycle.md pointer semantics) are recorded in the intake's Affected Memory and land at hydrate, not here

## Deletion Candidates

- `src/go/fab/internal/runtime/runtime.go:330 runtime.GCIfDue` — zero production call sites after T004 folded GC into the hook mutator calls; survives only as a one-line `UpdateAgent` wrapper exercised by runtime_test.go's 5 targeted GC tests (unexport or remove with tests rewritten against `UpdateAgent`)
- `src/go/fab/cmd/fab/status.go:52 loadStatus` statusPath/fabRoot return values — all 9 surviving call sites are read-only subcommands that discard them (`sf, _, _, err :=`); the mutating callers that consumed them moved to `withStatusLock`, so the signature can shrink to `(*sf.StatusFile, error)`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Lock acquisition is bounded (LOCK_NB + retry, ~10s) rather than blocking indefinitely | Stage hooks run inside `status.Start`/`Finish` while the cmd-level lock is held; a child `fab status` on the same change would deadlock forever under blocking flock — bounded wait fails loud instead; uncontended cost is one syscall | S:60 R:85 A:80 D:70 |
| 2 | Confident | `ComputeWithStatus` mutates confidence in memory and does not save; callers own persistence and locking | The binding "exactly one Save per hook event" forces caller-owned persistence; standalone `Compute` keeps today's persist behavior by saving under its own lock | S:75 R:85 A:85 D:75 |
| 3 | Confident | Runtime API shape: exported `UpdateAgent(fabRoot, createIfMissing, mutate, gcInterval)` core + per-op wrappers gaining a `gcInterval` param (≤0 = no GC); hooks call the wrappers | Intake sketches `UpdateAgent(... mutate func ...)`; binding constraints are single load/save, GC-on-no-op, skip-save — wrappers keep call sites typed without duplicating mutation logic; internal API only, no CLI change | S:70 R:85 A:85 D:65 |
| 4 | Confident | syncToRaw insertion covers all command-mutated keys (issues/prs/plan/confidence/stage_metrics/last_updated always; name/change_type when non-empty), beyond the three keys the report names | The report's fix says "generalize the insert" — the same dropped-write class hits `AddIssue`, `Rename`'s name update, and `SetChangeType` on sparse files; uniform insertion matches the `TestLegacyChecklistFileSavesPlanBlock` posture | S:65 R:80 A:85 D:70 |
| 5 | Confident | Stale pointers are never removed by resolve (intake's "optionally remove it" declined) | `_preamble.md`/`_cli-fab.md` document `fab resolve` as a pure query with no side effects; fall-through alone restores the actionable guidance | S:70 R:90 A:85 D:75 |
| 6 | Certain | `Rename` keeps best-effort pointer update (error not propagated); atomic swap means failure now leaves the old pointer rather than none | Existing contract discards the error; with R8 the failure mode improves from "permanently no pointer" to "old pointer remains", which R9's fall-through then handles | S:80 R:90 A:90 D:85 |
| 7 | Confident | `.status.yaml` locks live at `fab/changes/<folder>/.status.yaml.lock`; gitignore uses the basename pattern `.status.yaml.lock` plus an explicit `.fab-runtime.yaml.lock` entry (redundant with `.fab-*` but explicit per the binding correction) | Sibling placement is the report's stated design; a slash-less gitignore pattern matches at any depth, covering every change folder with one line | S:70 R:90 A:85 D:80 |

7 assumptions (1 certain, 6 confident, 0 tentative).
