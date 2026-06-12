# Plan: Operator/Pane/Tmux Surface — Perf + Error Surfacing (Binary-Review B5)

**Change**: 260612-pw3k-operator-pane-perf-error-surfacing
**Intake**: `intake.md`

## Requirements

### Agent-Spawn Surface: Launch-Failure Surfacing (F31)

#### R1: `fab batch new` surfaces per-item launch failures and exits non-zero
`runBatchNew` (`src/go/fab/cmd/fab/batch_new.go`) MUST capture the error from the `tmux new-window` launch. On failure it MUST print a per-item FAILED line naming the item **and the already-created worktree path** (recovery/cleanup hint), track a failure count, and return a non-nil error (→ central `ERROR: %s` + exit 1) when any item failed. The `wt create` failure branch MUST include the exec error (`%v`) plus the captured child stderr in its message, and counts toward the failure count (intake Impact: "non-zero exit on any launch failure"). Backlog not-found / empty-content items remain warn-and-skip (not failures). Pattern precedent: `batch_archive.go` archiveLoop.

- **GIVEN** a pending backlog item whose worktree is created but whose `tmux new-window` launch fails
- **WHEN** `fab batch new <id>` runs
- **THEN** stderr carries a FAILED line naming `[<id>]`, the tmux diagnostic, and the created worktree path
- **AND** the command exits non-zero (central `ERROR:` line), instead of today's silent exit 0

- **GIVEN** `wt create` fails for an item
- **WHEN** the loop processes that item
- **THEN** the warning includes the exec error and `wt`'s captured stderr, the item is counted as failed, and processing continues with the next item

### Agent-Spawn Surface: RunE Error Returns (F38)

#### R2: The three unpinned `os.Exit(1)` sites return errors instead
Exactly three sites MUST replace `Fprintln(errW)` + `os.Exit(1)` with `return fmt.Errorf(...)`, letting `main.go`'s central handler format (`ERROR: %s`) and exit: `batch_new.go` `$TMUX` guard, `batch_new.go` empty-pending-backlog guard, `operator.go` `$TMUX` guard. Deliberate user-visible stderr change (intake-pinned verbatim): `Error: not inside a tmux session.` → `ERROR: not inside a tmux session`; `No pending backlog items found.` → `ERROR: No pending backlog items found.`. No blanket sweep of the ~12 sibling sites; `pane_window_name.go`'s custom exit-code scheme stays the documented exception.

- **GIVEN** `$TMUX` is unset
- **WHEN** `fab batch new <id>` or `fab operator` runs
- **THEN** the RunE returns an error (testable in-process), stderr reads `ERROR: not inside a tmux session`, exit code 1

- **GIVEN** an empty pending backlog
- **WHEN** `fab batch new --all` runs inside tmux
- **THEN** the RunE returns an error, stderr reads `ERROR: No pending backlog items found.`, exit code 1

### Subprocess Errors: Child Stderr Surfacing (F35)

#### R3: A shared stderr-capturing helper enriches tmux/git/wt subprocess errors
`internal/pane` MUST gain a shared helper pair — `RunCmd(name, args...) (stdout string, stderr []byte, err error)` (generalizing the `ReadWindowName` capture pattern) and `StderrError(err, stderr) error` (appends the trimmed child stderr to the exec error when present) — applied at: `pane_capture.go` `capturePaneContent`, `pane_send.go` (both `send-keys` invocations), `operator.go` (tmux `new-window`, `gitRepoRoot`), and `batch_new.go` (`wt create`, tmux `new-window`, per R1). Errors MUST include the trimmed child message and the relevant identifier (pane ID / target). `--raw` capture output MUST remain byte-identical (no trimming of captured stdout).

- **GIVEN** a tmux/git subprocess fails with a diagnostic on stderr
- **WHEN** the wrapped command surfaces the error
- **THEN** the error text includes the child's trimmed stderr (e.g. `tmux send-keys to %5: exit status 1: can't find pane: %5`) instead of a bare `exit status 1`

### Operator Launcher: Exact Server-Wide Singleton (F33)

#### R4: Operator singleton check is an exact, server-wide window-name match
`runOperator` MUST replace the `tmux select-window -t operator` guard with enumeration of `tmux list-windows -a` (format `#{session_name}\t#{window_id}\t#{window_name}`) and an **exact** window-name comparison. On exact match it selects that window by its server-global window ID (`select-window -t <window_id>`), with a best-effort `switch-client -t <window_id>` so a cross-session match moves the user's client (failure ignored), and prints the existing `Switched to existing operator tab.` line. Window absent → launch; tmux enumeration error → surfaced error (not conflated with "absent"). This aligns code with the documented `_cli-fab.md` contract and the per-SERVER singleton.

- **GIVEN** a window named `operator-logs` and no `operator` window
- **WHEN** `fab operator` runs
- **THEN** no false positive: a new `operator` window is launched (today the prefix match suppresses the launch and switches the user to the wrong window)

- **GIVEN** an `operator` window in a *different* session on the same server
- **WHEN** `fab operator` runs
- **THEN** no duplicate operator is spawned; the existing window is selected

### Pane Map: Subprocess Dedup (F32)

#### R5: One `git rev-parse` per distinct pane cwd
`runPaneMap` MUST resolve each pane's git worktree root exactly once per distinct cwd via a cwd-keyed cache (`map[cwd]wtRoot`), threading the resolved root into both `mainRootForPane` and `resolvePane` (which today each re-run `pane.GitWorktreeRoot`). The cache MUST carry the non-git signal as the `""` sentinel — `resolvePane`'s non-git branch keys off it.

- **GIVEN** a 10-pane server
- **WHEN** `fab pane map --all-sessions` runs
- **THEN** at most one `git rev-parse --show-toplevel` is spawned per distinct pane cwd (was 2 per pane), with identical row output

#### R6: Single `tmux list-panes -a` replaces the per-session loop
`discoverAllSessions` MUST run a single `tmux list-panes -a -F <tmuxPaneFormat>` (the format already carries `#{session_name}`) instead of `list-sessions` + one `list-panes -s -t <session>` per session. Incidentally fixes the latent prefix/glob target-resolution wrinkle of `-t <sess>`. The obsolete `listSessionsArgs` builder is removed; a `listAllPanesArgs` builder is added for argv unit tests.

- **GIVEN** a 4-session server
- **WHEN** `fab pane map --all-sessions` runs
- **THEN** exactly one tmux enumeration subprocess is spawned (was 5), and the parsed pane set is identical

### Pane Map: `display_state` JSON Field ([dkn3])

#### R7: `fab pane map --json` rows gain a nullable `display_state`
`resolvePane` MUST stop discarding the state half of `status.DisplayStage` and surface it as a **nullable, additive-only** `display_state` field on `--json` rows (placed immediately after `stage`), with value domain `active | ready | done | pending | skipped` — the set `status.DisplayStage` can actually emit. <!-- rework cycle 1, review MF-2: `failed` dropped from the documented domain — DisplayStage (internal/status/status.go:464-499) has no failed tier (first-active / first-ready / last-done-or-skipped / first-pending), so a review-failed change flattens to its last done stage and `failed` is structurally unreachable. internal/status is outside this change's wave-1 file scope, so all doc sites note the flattening and a follow-up backlog item is filed for adding a failed tier. --> It MUST be `null`/omitted exactly when the pane has no resolvable change or its `.status.yaml` fails to load (the same condition under which `stage` is null today). Table output MUST be unchanged (no new column). run-kit's own `api.md` update is out of scope.

- **GIVEN** a pane on a change whose apply stage is active
- **WHEN** `fab pane map --json` runs
- **THEN** the row carries `"stage": "apply"` and `"display_state": "active"`

- **GIVEN** a non-fab pane (or a failing `.status.yaml` load)
- **WHEN** `fab pane map --json` runs
- **THEN** `display_state` is `null`, exactly like `stage`

- **GIVEN** any pane set
- **WHEN** `fab pane map` renders the table
- **THEN** the table bytes are identical to the pre-change output

### Memory Index: Batched Git Log (F34)

#### R8: One batched `git log` pass replaces per-file `git log -1` spawns
`internal/memoryindex` `Gather` MUST run a single batched `git -c core.quotepath=off log --date=short --format=%x00%ad --name-only -- docs/memory` pass (keyed by paths relative to `git rev-parse --show-toplevel`), streaming the output and recording the **first (most recent) date seen per path**, then look dates up from that map in `gatherFiles`. The per-file `gitLastUpdated` call is kept **only as fallback** when the batched call fails (not for per-path misses — a missing path means uncommitted, matching the current `""` behavior). Rendered index output MUST be byte-identical.

- **GIVEN** a git repo with committed `docs/memory` files
- **WHEN** `fab memory-index` runs
- **THEN** dates come from one batched git pass and equal the per-file `git log -1 --date=short` dates

- **GIVEN** a non-git directory (batched call fails)
- **WHEN** `Gather` runs
- **THEN** the per-file fallback path is used and dates degrade to `""`/`—` exactly as today

### Pane Validation: Targeted Probe (F36)

#### R9: `ValidatePane` uses a targeted `display-message` probe, not server-wide enumeration
`pane.ValidatePane` MUST replace the `tmux list-panes -a` enumeration with a single `tmux display-message -t <arg> -p '#{pane_id}'` probe, comparing the trimmed output to the argument (**ID-exactness**: a window-name/target-grammar arg resolves to some pane ID ≠ arg → rejected). "Can't find pane" stderr maps to the existing `pane %s not found` error (reusing the pane-missing matcher shared with `pane_window_name.go`'s `tmuxExitCode`); other tmux failures surface stderr per R3. Error-path equivalence (re-verified empirically on tmux 3.6a, per the intake's binding re-verification step): missing pane → same `Error: pane <id> not found` + exit 1 (on 3.6a `display-message` exits 0 with empty output for a missing pane, so the output==arg comparison is the load-bearing check; the stderr branch covers tmux versions that error); dead server → exit 1 (unchanged) with the stderr detail now included (deliberate, per R3). The `fab pane send --force` "still validates pane existence" contract and the three call sites (capture/send/process) are unchanged.

- **GIVEN** a missing pane `%99`
- **WHEN** `fab pane send %99 hi --force` runs
- **THEN** stderr is `Error: pane %99 not found` and exit code is 1 (unchanged contract)

- **GIVEN** an existing window name passed as the pane argument
- **WHEN** `ValidatePane` probes it
- **THEN** the probe output (a real pane ID) differs from the argument and validation fails — no target-grammar loosening

- **GIVEN** a dead/unreachable tmux server
- **WHEN** `ValidatePane` probes
- **THEN** exit code 1 with the tmux connection diagnostic included in the error

### Darwin Pane Process: Batched `ps` (F37)

#### R10: Full cmdlines come from a second single `ps` pass joined by PID
`pane_process_darwin.go` MUST replace the per-node `ps -o args= -p <pid>` spawns with one `ps -axo pid=,args=` pass parsed into a `map[pid]cmdline` (pid is numeric-first, remainder is args — robust against comm-with-spaces) and joined by PID during the tree walk. Output shape unchanged; a process exiting between passes yields cmdline `""` (same degraded value as today's per-pid failure). The pure parser lives in the un-tagged `pane_process.go` so it is unit-testable on Linux; the darwin file is verified via `GOOS=darwin go build ./... && GOOS=darwin go vet ./...`.

- **GIVEN** a pane process tree of N nodes on macOS
- **WHEN** `fab pane process <pane>` runs
- **THEN** exactly 2 `ps` subprocesses are spawned (was N+1), with identical output shape

### Docs: CLI Reference and Skill Mirrors (constitution :31-32)

#### R11: Kit-skill docs reflect every changed mechanism/contract
`src/kit/skills/_cli-fab.md` MUST be updated at: the pane map `--json` row (add `display_state`), :208 send validation (probe replaces `list-panes -a`), :212 darwin ps mechanism, :363-366 memory-index date sourcing (batched pass + fallback), :444 operator singleton (exact, server-wide), :492 batch new exit semantics. `src/kit/skills/fab-operator.md` gets the pane-map output-shape touch (tick snapshot `--json` field mention), mirrored in `docs/specs/skills/SPEC-fab-operator.md`. Memory docs (`docs/memory/**`) are NOT edited at apply — hydrate's job (intake Affected Memory queues `runtime/pane-commands`, `runtime/operator`, `memory-docs/hydrate`, `memory-docs/templates`).

- **GIVEN** the implemented mechanism changes
- **WHEN** an agent reads `_cli-fab.md`
- **THEN** the documented mechanisms (validation probe, darwin ps, memory-index dates, operator singleton, batch-new exit codes, pane map JSON shape) match the code

### Non-Goals

- `batch_switch.go`'s identical discarded errors — [ye8r]/F29 surface (backlog wave-1 scope constraint)
- `hook.go` / `runtime.go` — owned by parallel change [mz4q]
- Blanket sweep of the ~12 sibling `os.Exit(1)` sites — `resolve --pane`/`pane map` stderr semantics are pinned in live memory docs
- run-kit's consumption of `display_state` and its `docs/specs/api.md` — separate repo, out of scope
- `docs/memory/**` edits — deferred to hydrate
- PaneContext (`pane capture --json` enrichment) gaining `display_state` — [dkn3] scopes the field to pane map rows only

### Design Decisions

1. **Helpers live in `internal/pane`**: `RunCmd`/`StderrError`/`IsPaneMissing` are exported from `internal/pane/pane.go` — *Why*: the memory-documented convention ("cross-package argv builders … are exported from `internal/pane` when consumed outside the pane package; future tmux-helper additions should follow the same pattern"), and the file-scope boundary forbids new files in `cmd/fab`. — *Rejected*: a new `internal/tmuxutil` package (over-engineering for three helpers; same reasoning as the `WithServer` decision).
2. **F36 probe keeps both detection branches**: output==arg comparison (load-bearing on tmux ≥3.6 where `display-message` exits 0/empty for a missing pane) plus the "can't find pane" stderr mapping (older tmux). — *Why*: empirical verification on tmux 3.6a contradicted the verifier's assumed stderr path; both branches make the probe version-robust.
3. **F33 selects by `#{window_id}`**: window IDs are server-global and exempt from target-grammar prefix/glob resolution; `switch-client` is best-effort for cross-session jumps. — *Rejected*: `select-window -t '=operator'` alone (stays session-scoped, misses the per-server invariant).
4. **F31 failure-path test fakes `wt`/`tmux` via `$PATH`**: integration-style test with stub scripts in a temp dir prepended to `PATH`. — *Why*: exercises the real loop (print order, failure count, exit semantics) without tmux; precedent for env-keyed test seams exists in `operator_test.go` (`operatorStatePathOverride`, `t.Setenv`).

## Tasks

### Phase 1: Setup

- [x] T001 Add `RunCmd(name string, args ...string) (string, []byte, error)`, `StderrError(err error, stderr []byte) error`, and `IsPaneMissing(stderr []byte) bool` to `src/go/fab/internal/pane/pane.go`; unit tests in `src/go/fab/internal/pane/pane_test.go` <!-- R3 -->

### Phase 2: Core Implementation

- [x] T002 Rewrite `ValidatePane` in `src/go/fab/internal/pane/pane.go` as the targeted `display-message -t <arg> -p '#{pane_id}'` probe with an extracted pure comparator (`validatePaneResult`); rewire `tmuxExitCode` in `src/go/fab/cmd/fab/pane_window_name.go` onto `pane.IsPaneMissing`; tests for comparator branches (missing-pane stderr, empty-output mismatch, window-name mismatch, dead-server stderr, exact match) <!-- R9 -->
- [x] T003 Apply the stderr-surfacing helpers at `src/go/fab/cmd/fab/pane_capture.go` (`capturePaneContent` — raw stdout untrimmed), `src/go/fab/cmd/fab/pane_send.go` (both send-keys sites, errors name the pane), `src/go/fab/cmd/fab/operator.go` (`tmux new-window`, `gitRepoRoot`) <!-- R3 -->
- [x] T004 Replace the three `os.Exit(1)` sites with returned errors in `src/go/fab/cmd/fab/batch_new.go` (×2) and `src/go/fab/cmd/fab/operator.go` (×1); RunE-level tests in `batch_new_test.go` ($TMUX unset; empty pending backlog) and `operator_test.go` ($TMUX unset) <!-- R2 -->
- [x] T005 Surface launch failures in `src/go/fab/cmd/fab/batch_new.go`: capture `wt create` stderr into the warning, capture `tmux new-window` error into a per-item FAILED line naming the worktree path, count failures, return non-nil error when `failed > 0`; PATH-stubbed integration test in `src/go/fab/cmd/fab/batch_new_test.go` (tmux-fails → FAILED line + worktree path + error; all-succeed → nil) <!-- R1 -->
- [x] T006 Exact server-wide singleton in `src/go/fab/cmd/fab/operator.go`: `list-windows -a -F '#{session_name}\t#{window_id}\t#{window_name}'`, pure `findWindowExact` parser, select by window ID + best-effort `switch-client`; parser tests in `src/go/fab/cmd/fab/operator_test.go` (exact match only — `operator-logs` no match; cross-session match; tab-in-name tolerance) <!-- R4 -->
- [x] T007 Dedupe `git rev-parse` in `src/go/fab/cmd/fab/panemap.go`: cwd-keyed `worktreeRootForPane` cache ("" non-git sentinel), thread `wtRoot` through `mainRootForPane(cwd, wtRoot, cache)` and `resolvePane(p, wtRoot, mainRoot, server, runtimeCache)`; update `TestMainRootForPane` and `TestResolvePanePRURL` call shapes in `src/go/fab/cmd/fab/panemap_test.go`, add `worktreeRootForPane` cache tests <!-- R5 -->
- [x] T008 Single-call discovery in `src/go/fab/cmd/fab/panemap.go`: `discoverAllSessions` → one `tmux list-panes -a -F <tmuxPaneFormat>` via new `listAllPanesArgs` builder; drop `listSessionsArgs`; replace `TestListSessionsArgs` with `TestListAllPanesArgs` in `panemap_test.go` <!-- R6 -->
- [x] T009 Add `display_state` to `src/go/fab/cmd/fab/panemap.go`: `paneRow.displayState`, `paneJSON.DisplayState *string` after `Stage`, populate from `status.DisplayStage`'s second return in `resolvePane`; tests in `panemap_test.go` (populated + null JSON, field name present, table output byte-identical/no leak) <!-- R7 -->
- [x] T010 Batched git log in `src/go/fab/internal/memoryindex/memoryindex.go`: `gitDates` struct (`loadGitDates(repoRoot)`, nil on failure → per-file fallback via nil-receiver `lookup`), parse `%x00`-delimited newest-first stream, thread through `Gather`/`gatherFiles`/`gatherSubDomains`; tests in `memoryindex_test.go` (pure parser; pinned-date git-repo integration asserting batch == per-file; non-git fallback) <!-- R8 -->
- [x] T011 Two-pass darwin ps in `src/go/fab/cmd/fab/pane_process_darwin.go`: single `ps -axo pid=,args=` pass joined by PID, per-node `getPSCmdline` removed; pure `parsePSCmdlines` parser in un-tagged `src/go/fab/cmd/fab/pane_process.go` with tests in `pane_process_test.go`; verify `GOOS=darwin go build ./... && GOOS=darwin go vet ./...` <!-- R10 -->

### Phase 3: Integration & Edge Cases

- [x] T012 Full verification from `src/go/fab`: `gofmt -l`, `go vet ./...`, `go test ./... -count=1` (plus `just test` at repo root), and the darwin cross-compile check; fix any fallout <!-- R1 R2 R3 R4 R5 R6 R7 R8 R9 R10 -->

### Phase 4: Polish

- [x] T013 Update `src/kit/skills/_cli-fab.md`: pane map `--json` row (+`display_state`), :208 send validation probe, :212 darwin ps mechanism, :363-366 memory-index date sourcing, :444 operator singleton mechanism, :492 batch new exit semantics <!-- R11 -->
- [x] T014 Touch `src/kit/skills/fab-operator.md` tick-snapshot output-shape mention (+nullable `display_state`) and mirror in `docs/specs/skills/SPEC-fab-operator.md` <!-- R11 -->

### Phase 5: Rework (cycle 1)

- [x] T015 Delegate `pane.ReadWindowName` (`src/go/fab/internal/pane/pane.go:141-148`) onto `RunCmd` — two-line body (`RunCmd` + `strings.TrimSpace`), behavior-identical; rewire the companion stderr-only capture in `renameWindow` (`src/go/fab/cmd/fab/pane_window_name.go:119-125`) onto the shared helper; drop the satisfied `## Deletion Candidates` entry <!-- R3 --> <!-- rework: review MF-1 / A-022 — RunCmd was added but its motivating duplication was never removed -->
- [x] T016 Drop `failed` from the documented `display_state` value domain at all four sites — `src/go/fab/cmd/fab/panemap.go` doc comment (:373), `src/kit/skills/_cli-fab.md` pane map row, `src/kit/skills/fab-operator.md`, `docs/specs/skills/SPEC-fab-operator.md` — noting that DisplayStage flattens failed states (no failed tier); append a follow-up backlog item to `fab/backlog.md` (new unique 4-char ID, dated 2026-06-12) for adding a `failed` tier to `status.DisplayStage`, referencing review MF-2 of this change and noting internal/status is outside pw3k's wave-1 scope <!-- R7 R11 --> <!-- rework: review MF-2 — docs promised a value the producer cannot emit -->
- [x] T017 Tighten `findWindowExact` in `src/go/fab/cmd/fab/operator.go`: format `'#{window_id}\t#{window_name}'` with `strings.SplitN(line, "\t", 2)` (drop the unused leading `#{session_name}` field — a tab inside a session name shifts columns and silently misses the existing operator window); extend `TestFindWindowExact` with a tab-in-session-name case <!-- R4 --> <!-- rework: review SF-1 — clear and low-effort -->

## Execution Order

- T001 blocks T002, T003, T005, T006 (helpers consumed everywhere)
- T007 blocks T009 only in that both edit `resolvePane` — execute sequentially (T007 → T009)
- T010, T011 are independent of the pane/operator chain ([P]-equivalent)
- T013/T014 last, after behavior is final

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab batch new` prints a per-item FAILED line (item + worktree path) on tmux launch failure, counts failures, and exits non-zero when any item failed; `wt create` failures carry the child stderr
- [x] A-002 R2: The three sites (`batch_new.go` ×2, `operator.go` ×1) return errors through RunE; no `os.Exit` remains in `batch_new.go`/`operator.go`; stderr strings match the intake-pinned `ERROR: …` forms
- [x] A-003 R3: `RunCmd`/`StderrError` exist in `internal/pane` and are applied at capture/send/operator/batch-new subprocess sites; failures include trimmed child stderr and the relevant identifier
- [x] A-004 R4: Operator singleton check enumerates `tmux list-windows -a` and compares names exactly; exact match selects by window ID; absence launches; tmux error is surfaced distinctly
- [x] A-005 R5: Pane map spawns at most one `git rev-parse` per distinct pane cwd; non-git sentinel cached and honored by `resolvePane`
- [x] A-006 R6: `--all-sessions` discovery is a single `tmux list-panes -a` call; `listSessionsArgs` removed
- [x] A-007 R7: `pane map --json` rows carry nullable `display_state` (after `stage`, additive-only); table output unchanged
- [x] A-008 R8: Memory-index dates come from one batched `git log --name-only` pass with per-file fallback only on batch failure; index output byte-identical
- [x] A-009 R9: `ValidatePane` is a single targeted `display-message` probe with output==arg ID-exactness; the three callers and `--force` existence contract unchanged
- [x] A-010 R10: Darwin `pane process` spawns exactly two `ps` passes; cmdlines joined by PID; output shape unchanged
- [x] A-011 R11: All six `_cli-fab.md` rows, the `fab-operator.md` output-shape mention, and the `SPEC-fab-operator.md` mirror reflect the new mechanisms

### Behavioral Correctness

- [x] A-012 R9: Error-path equivalence re-verified (binding intake step): missing pane → `Error: pane <id> not found` + exit 1; dead server → exit 1 with stderr detail; window-name args rejected (no target-grammar loosening)
- [x] A-013 R2: `$TMUX`-unset and empty-backlog paths are exercised in-process by unit tests (previously untestable due to `os.Exit`)
- [x] A-014 R7: `display_state` is null exactly when `stage` is null (no resolvable change / status load failure)

### Scenario Coverage

- [x] A-015 R1: PATH-stubbed test covers both the tmux-failure FAILED path (error returned, worktree path printed) and the all-success path (nil error)
- [x] A-016 R8: Git-repo integration test pins commit dates and asserts batched dates equal per-file `git log -1` dates; non-git fixture exercises the fallback
- [x] A-017 R4: Parser tests cover exact-match-only (prefix name `operator-logs` rejected), cross-session match, and absence
- [x] A-030 R4: `findWindowExact` parses `window_id`/`window_name` only via `SplitN(..., 2)`; a tab inside a session name cannot shift columns (covered by a parser test)

### Edge Cases & Error Handling

- [x] A-018 R9: Probe handles tmux 3.6a's exit-0/empty-output missing-pane behavior (comparison branch) AND older tmux's "can't find pane" stderr (mapping branch)
- [x] A-019 R3: Empty child stderr degrades to the bare exec error (no trailing colon/garbage); `--raw` capture output is untrimmed
- [x] A-020 R8: Uncommitted file (present on disk, absent from batch map) renders `—` exactly as before; exotic-path quoting disabled via `core.quotepath=off`

### Code Quality

- [x] A-021 Pattern consistency: New code follows naming and structural patterns of surrounding code (argv-builder helpers, `internal/pane` export convention, archiveLoop failure-count precedent)
- [x] A-022 No unnecessary duplication: `pane_window_name.go`'s pane-missing matching is unified onto `pane.IsPaneMissing`; one shared `RunCmd` instead of per-site capture buffers
- [x] A-023 No god functions: `runBatchNew`/`runOperator`/`runPaneMap` stay focused; parsing extracted into pure helpers (`findWindowExact`, `parsePSCmdlines`, `validatePaneResult`, `parseGitDates`)
- [x] A-024 No magic strings: tmux formats, sentinels, and pane-missing markers named/centralized where shared

### Documentation Accuracy

- [x] A-025: `_cli-fab.md` rows match the implemented mechanisms exactly (probe, darwin ps two-pass, batched git log + fallback, exact server-wide singleton, batch-new exit semantics, pane map JSON shape)
- [x] A-026: No `docs/memory/**` file was edited at apply (hydrate owns those updates)
- [x] A-029 R7: The documented `display_state` value domain (R7 + all four doc sites) contains only values `status.DisplayStage` can emit (no `failed`); each site notes the failed-state flattening; a follow-up backlog item for a DisplayStage `failed` tier exists in `fab/backlog.md`

### Cross References

- [x] A-027: `fab-operator.md` and `docs/specs/skills/SPEC-fab-operator.md` stay consistent with each other and with `_cli-fab.md`'s pane map row
- [x] A-028: Excluded files untouched: `hook.go`, `runtime.go` (mz4q), `batch_switch.go` (ye8r)

## Notes

- **Rebase reconciliation (2026-06-12, onto origin/main after PRs #394–#397 merged)**: [dkn3] shipped standalone as `260612-dkn3-pane-map-display-state` (PR #394) **including the DisplayStage `failed` tier** this change's review (MF-2) had identified as out-of-scope. Consequences folded in during the rebase: pw3k's duplicate `display_state` implementation/tests/docs yielded to main's (R7/T009/T016's display_state portions are upstream-superseded; the perf work R5/R6 and everything else stands); the `failed`-flattening caveats were removed from `fab-operator.md`/`SPEC-fab-operator.md` (the full domain incl. `failed` is now correct per main's `_cli-fab.md` row); follow-up backlog item [fj3d] was withdrawn as moot; [dkn3] marked shipped in `fab/backlog.md`.
- Apply-time verification: full `just test` green (26 packages, both modules); `gofmt`/`go vet` clean on linux and `GOOS=darwin`; live smoke on a scratch tmux 3.6a server confirmed `display_state` end-to-end, all three ValidatePane error paths, the operator false-positive fix, the cross-session singleton match, and `fab memory-index --check` byte-equivalence of batched vs per-file dates on this repo's real history
- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None outstanding — all claims re-verified at re-review (rework cycle 1). Everything this change made redundant was deleted in the same diff:

- `listSessionsArgs` + the per-session discovery loop in `panemap.go` — replaced by `listAllPanesArgs` / one `tmux list-panes -a` call (zero references remain; verified by grep)
- `getPSCmdline` (per-PID `ps` spawn) in `pane_process_darwin.go` — replaced by the batched `getPSCmdlines` + pure `parsePSCmdlines` pair (zero references remain)
- The `tmux list-panes -a` enumeration body of `pane.ValidatePane` — replaced by the targeted `display-message` probe + pure `validatePaneResult` (the only remaining `list-panes -a` mention in `internal/pane/pane.go` is a doc-comment reference to the previous semantics)
- The inline pane-missing matcher in `tmuxExitCode` (`pane_window_name.go`) — unified onto `pane.IsPaneMissing`
- The hand-rolled stdout/stderr capture blocks in `pane.ReadWindowName` and `renameWindow` — delegated onto `pane.RunCmd` in rework cycle 1 (T015); `RunCmd` itself is now the single capture implementation in non-test code

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | F36 probe verified empirically on tmux 3.6a: `display-message -t <missing-pane>` exits **0 with empty output** (not "can't find pane" stderr as the verifier assumed) — the output==arg comparison is the load-bearing check; the stderr-mapping branch is retained for tmux versions that do error | Intake mandated re-verification before removing the old path; both branches make the probe version-robust with identical user-visible behavior | S:90 R:85 A:90 D:90 |
| 2 | Confident | F33 selects the matched window by `#{window_id}` (server-global, grammar-exempt) and adds best-effort `switch-client -t <window-id>` for cross-session jumps (error ignored) | Intake says "select that window" without specifying cross-session mechanics; `select-window` alone doesn't move the client between sessions; verified both commands accept window IDs on tmux 3.6a | S:70 R:85 A:80 D:70 |
| 3 | Certain | F38 strings follow the intake verbatim: `not inside a tmux session` (lowercase, no period) and `No pending backlog items found.` (capital + period preserved), despite the intra-change style inconsistency | Intake pins both resulting stderr lines exactly; verifier accepted them as the deliberate output change | S:85 R:90 A:85 D:85 |
| 4 | Confident | `wt create` failures count toward F31's failure count / non-zero exit (alongside tmux launch failures); backlog not-found/empty-content stay warn-and-skip | Intake Impact says "non-zero exit on **any launch failure**"; a failed worktree creation is a failed launch; skip semantics for bad input mirror `batch archive`'s resolve warnings | S:65 R:85 A:80 D:70 |
| 5 | Certain | `display_state` placed immediately after `stage` in the JSON struct; null exactly when `stage` is null | Intake pins the null condition ("same condition under which `stage` is absent today"); adjacency is the natural additive placement | S:85 R:90 A:90 D:90 |
| 6 | Confident | F34 batch keys paths relative to `git rev-parse --show-toplevel` (one extra spawn) with `-c core.quotepath=off`; per-file fallback fires only when the batched call fails, not on per-path misses | `git log --name-only` prints repo-root-relative paths, which only equal `repoRoot`-relative paths when `fab/`'s parent is the git root — resolving the real top-level removes that silent assumption; quotepath guards exotic filenames | S:70 R:85 A:85 D:75 |
| 7 | Confident | F37's pure `ps` args parser lives in the un-tagged `pane_process.go` so it is unit-testable on Linux; the darwin file is gated by `GOOS=darwin go build`/`go vet` only | Constitution mandates tests with CLI changes; darwin-tagged test files would never run in CI on this Linux machine; the parser is platform-independent string logic | S:70 R:90 A:85 D:75 |
| 8 | Certain | `docs/memory/**` (pane-commands, operator, hydrate:63, templates:115) untouched at apply despite intake F34/F36/F37 listing them — they are queued in Affected Memory for hydrate | Orchestrator hard rule ("do NOT edit docs/memory — hydrate's job") + intake Affected Memory section already routes them to hydrate | S:90 R:95 A:95 D:90 |
| 9 | Confident | `fab-operator.md` gets a one-clause `display_state` mention inside tick step 1's `--json` parenthetical; `SPEC-fab-operator.md` mirrors at its tick-snapshot description | Intake says "check if fab-operator.md needs a pane-map output-shape touch" — the tick step is the only place the JSON shape is described; minimal touch keeps the skill's frame logic unchanged | S:65 R:90 A:80 D:75 |

9 assumptions (4 certain, 5 confident, 0 tentative).
