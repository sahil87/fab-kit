# Plan: fab dispatch wait — event-driven blocking wait replaces the 30s poll loop

**Change**: 260806-mkfj-dispatch-wait-event-driven
**Intake**: `intake.md`

## Requirements

### CLI: `fab dispatch wait`

#### R1: A seventh `fab dispatch` subcommand `wait <change> <stage>` blocks until the dispatch leaves `running`

The `fab` binary SHALL expose `fab dispatch wait <change> <stage> [--timeout <secs>] [--json]`, registered on the existing `dispatchCmd()` parent alongside `start`/`restart`/`status`/`logs`/`kill`/`clean`. It SHALL block while the derived dispatch state is `running`, re-deriving that state on a fixed internal tick, and return as soon as the state is any non-`running` value. It SHALL be purely additive — no existing subcommand's behavior, flags, output, or record schema changes.

- **GIVEN** a dispatch for `abcd/apply` whose state is `running`
- **WHEN** `fab dispatch wait abcd apply` runs and the worker subsequently writes its exit + result files
- **THEN** the command returns shortly after the transition, prints `done`, and exits 0

#### R2: The wait loop re-derives state through the exact functions `status` uses

The wait loop SHALL obtain state via the same record loader (`dispatch.Load`) and the same pure derivations `fab dispatch status` uses — `DeriveState` for a headless record (`ReadExit` + `ResultPresent` + `Alive`) and `DerivePaneState` for a pane record (`ResultPresent` + `PaneAlive`) — selected by the record's derived mode (`Dispatch.IsPane()`). It MUST NOT introduce a second, parallel derivation. Consequently `wait` and `status` cannot report different states for the same on-disk signals. The per-observation logic SHALL live in `internal/dispatch` as a reusable, table-testable seam rather than in the cobra layer.

- **GIVEN** any on-disk dispatch record and signal set
- **WHEN** both `fab dispatch status` and `fab dispatch wait` observe it at the same instant
- **THEN** both report the identical state string

#### R3: An already-terminal dispatch returns immediately

If the derived state at entry is any value other than `running`, `wait` SHALL print it and exit 0 without sleeping. This makes the verb idempotent and safe to re-arm after a restart or re-run after an interruption.

- **GIVEN** a dispatch whose state is already `done` (or `failed` / `failed (no-result)` / `orphaned`)
- **WHEN** `fab dispatch wait <change> <stage>` runs
- **THEN** it prints that state and exits 0 without waiting a tick

#### R4: `--timeout <secs>` bounds the block; expiry prints the still-current state and exits 0

`--timeout <secs>` SHALL be an optional upper bound on the block. On expiry `wait` SHALL print the still-current state — necessarily `running`, since any other state would already have ended the wait — and exit 0. Absent (or `0`), the wait SHALL be unbounded. The printed state string is the sole discriminator: a consumer reading `running` knows the wait timed out; every other string is a terminal state reached. No distinct exit code is introduced for timeout.

- **GIVEN** a dispatch that stays `running`
- **WHEN** `fab dispatch wait <change> <stage> --timeout 1` runs
- **THEN** it returns after roughly one second, prints `running`, and exits 0

#### R5: The internal tick is a package constant of ~2 seconds, not fsnotify and not configurable

State re-derivation SHALL happen on a fixed in-process tick exposed as a named constant in `internal/dispatch` (`TickInterval = 2 * time.Second`). It MUST NOT use a filesystem watcher: a watcher cannot observe pid or pane death, which is the `orphaned` trigger, so a periodic liveness probe is required regardless. There SHALL be no config field and no flag for the tick. The wait core SHALL accept the tick as a parameter so tests can drive it fast.

- **GIVEN** a headless dispatch whose worker process dies without writing an exit file
- **WHEN** `fab dispatch wait` is blocking on it
- **THEN** the loop's next liveness probe (within ~2s) derives `orphaned` and the command returns

#### R6: Errors and `--json` mirror `status` exactly

`wait` SHALL share `status`'s error surface: an unresolvable change reference, or no dispatch record for the `(change, stage)` pair, SHALL exit non-zero with the same `no dispatch for <change>/<stage> (run \`fab dispatch start\` first)` message; only real errors are non-zero. `--json` SHALL emit the same object `fab dispatch status --json` emits — the `mode` discriminator plus that mode's identity keys — rendered through the **same** code path, not a re-declared struct. Human output is the bare state string on stdout, exactly as `status` prints it.

- **GIVEN** no dispatch record for `abcd/apply`
- **WHEN** `fab dispatch wait abcd apply` runs
- **THEN** it exits non-zero naming the missing dispatch, identically to `fab dispatch status`
- **AND** GIVEN a pane dispatch that is already `done`, `fab dispatch wait abcd apply --json` emits `{change, stage, state, mode: "pane", pane, window}` with no `pid`/`pgid`/`exit`

#### R7: Go tests cover the timeout path, the already-terminal fast path, and a state transition waking the wait

The change SHALL ship table tests in `internal/dispatch` for the wait core plus command-level tests in `cmd/fab`, covering at minimum: (a) an already-terminal state returning immediately, (b) timeout expiry returning `running` with a nil error, and (c) a state that changes from `running` to terminal mid-wait waking the loop and returning the new state. Tests conform to these requirements, never the reverse (Constitution VII).

- **GIVEN** the shipped test suite
- **WHEN** `go test ./internal/dispatch/... ./cmd/fab/...` runs
- **THEN** all three cases above are exercised and pass

### Skill wiring: push replaces the poll loop

#### R8: `_preamble.md` § CLI-Adapter Dispatch step 2 becomes a background blocking wait

`src/kit/skills/_preamble.md` § CLI-Adapter Dispatch step 2 ("Poll") SHALL be rewritten from `fab dispatch status` + `sleep 30` to: **preferred (push)** — run `fab dispatch wait <change> <stage> --timeout 300` as a **background command** (the harness's notify-on-exit seam; in Claude Code, Bash `run_in_background`), so the orchestrator spends zero turns until the command exits; and **fallback (degraded)** — the same `wait --timeout 300` as a plain foreground blocking call for harnesses without notify-on-exit background commands, still a 10× turn reduction. A terminal state SHALL route into step 3's unchanged five-state handling; a `running` return (the timeout) IS the peek-on-suspicion moment — peek, classify (a/b/c), re-arm a fresh `wait`.

- **GIVEN** the deployed `_preamble.md`
- **WHEN** a dispatch-seam skill reaches step 2 of § CLI-Adapter Dispatch
- **THEN** it is instructed to run `fab dispatch wait … --timeout 300` in the background, not `fab dispatch status` on a `sleep 30` loop

#### R9: The recovery policy's cadence carrier changes; its semantics do not

The § Recovery policy text SHALL replace "every 10th result-less poll" with "every timeout-return of `fab dispatch wait`" (the same ~5-minute cadence: `--timeout 300` = 10 polls × 30s). Everything else about the policy SHALL remain byte-equivalent in meaning: the single restart budget held in orchestrator context with nothing on disk, the automatic restart on `orphaned` (after which the orchestrator re-arms `wait`), no automatic restart on `failed`, never restarting `failed (no-result)`, the three-way peek classification (a/b/c), escalation via per-mode evidence + gated `rk notify`, and the pipeline's verb set of peek / kill / restart / notify / stop with **no send-keys, ever**. The pane-mode bullet asserting "the same fixed `sleep 30` poll applies" SHALL reference the same wait wiring instead.

- **GIVEN** the rewritten § Recovery policy
- **WHEN** the restart budget, classification, escalation, and never-send-keys rules are compared to the prior text
- **THEN** they are unchanged in substance — only the observation mechanism and cadence carrier differ

#### R10: The restating dispatch-adapter passages in `_pipeline.md` and `fab-continue.md` update in step

`src/kit/skills/_pipeline.md` and `src/kit/skills/fab-continue.md` each restate the CLI-adapter poll contract inline; both SHALL be updated to the wait wiring with the same cadence-carrier wording. `src/kit/skills/fab-adopt.md` references § CLI-Adapter Dispatch without restating the cadence and SHALL be verified to need no edit.

- **GIVEN** a repo-wide grep for `sleep 30` and "10th result-less poll" under `src/kit/skills/`
- **WHEN** the change is complete
- **THEN** no occurrence remains outside the unrelated `/git-pr-review` Copilot wait, and `fab-adopt.md` is confirmed unchanged

### Documentation & spec surfaces

#### R11: `_cli-fab.md` § fab dispatch documents the `wait` subcommand

`src/kit/skills/_cli-fab.md` § fab dispatch SHALL gain a `### wait` subsection documenting the signature, the blocking contract, `--timeout` semantics (expiry prints the current state and exits 0), the already-terminal fast path, the shared derivation with `status`, `--json` parity, and the error surface. The family sentence and header line naming six subcommands SHALL be updated to seven. This satisfies the constitution's "CLI change ⇒ `_cli-fab.md` + tests" constraint.

- **GIVEN** the constitution's CLI constraint
- **WHEN** `fab dispatch wait` ships
- **THEN** `_cli-fab.md` documents its signature and semantics, and the family enumeration reads `start|restart|status|wait|logs|kill|clean`

#### R12: `harness-adapters.md` records the wait/push observation model including the cross-harness fallback

`docs/specs/harness-adapters.md` SHALL replace the "fixed `sleep 30` cadence" claim with the wait/push observation model, and SHALL state the cross-harness fallback explicitly: a harness without notify-on-exit background commands runs the same bounded `wait` in the foreground. The spec MUST NOT assume a Claude-Code-only seam. The five-state machine, the result-file contract, and the dispatch-prompt obligations remain untouched.

- **GIVEN** a harness with no background-command facility
- **WHEN** its orchestrator consumes the spec
- **THEN** it finds the foreground blocking `wait --timeout 300` documented as the supported degraded path

#### R13: The SPEC mirror class is swept whole

Per Constitution Additional Constraints and `code-quality.md` § Sibling & Mirror Sweeps, every edited `src/kit/skills/*.md` file SHALL carry its `docs/specs/skills/SPEC-*.md` mirror update in this change: `SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-fab-continue.md`, `SPEC-_cli-fab.md`. The old claims ("sleep 30", "every 30s", "10th result-less poll") SHALL be grepped repo-wide and every occurrence in the class updated. `docs/specs/skills/SPEC-git-pr-review.md` contains an unrelated `sleep 30` (the Copilot-review wait) and MUST NOT be touched.

- **GIVEN** a repo-wide grep for the old cadence claims after the change
- **WHEN** the results are inspected
- **THEN** only the `/git-pr-review` Copilot wait and historical findings documents remain, and all four SPEC mirrors carry the wait wiring

### Non-Goals

- No worker-initiated push channel and no tmux send-keys in either direction — the cross-adapter contract stays observation-only.
- No change to `status`, `start`, `restart`, `kill`, or `clean`; `status` remains the one-shot probe.
- No new dispatch state, no `{stage}.yaml` schema change, no migration file.
- No backoff or cadence configuration: the internal tick is a package constant and `--timeout 300` is a skill-text constant, overridable per invocation like any flag.
- No edits under `.claude/skills/` (gitignored deployed copies).
- Memory files are not edited by apply — `docs/memory/runtime/dispatch.md`, `docs/memory/pipeline/execution-skills.md`, and `docs/memory/_shared/context-loading.md` are hydrate's responsibility.

### Design Decisions

#### Internal derivation tick rather than an fsnotify watch

**Decision**: `wait` re-derives state on a fixed ~2s in-process tick over the existing loader + `DeriveState`/`DerivePaneState`, exposed as `dispatch.TickInterval`.
**Why**: The efficiency being bought is *inference-turn* elimination, not filesystem-poll elimination — a 2s stat loop inside one Go process is free. It reuses the derivation `status` already owns, so the two verbs cannot disagree by construction, and it needs no new dependency.
**Rejected**: An fsnotify watch on `{stage}-result.yaml`. A file watcher cannot observe pid or pane death — the `orphaned` trigger — so a periodic liveness probe would still be needed alongside it, adding a dependency without removing the tick.
*Introduced by*: 260806-mkfj-dispatch-wait-event-driven

#### Timeout expiry exits 0 and discriminates by the printed state string

**Decision**: On `--timeout` expiry `wait` prints the still-current state (`running`) and exits 0; non-zero exits are reserved for real errors, mirroring `status`.
**Why**: The consuming skill reads stdout either way, and the state string already carries the information. A distinct exit code would add a second channel for the same fact, and would make the common "timed out, go peek" path look like a failure to any shell-level `set -e` wrapper.
**Rejected**: A dedicated timeout exit code (e.g. 124, by analogy with POSIX `timeout`). That code already means "the dispatched worker was killed by its own `--timeout` wrapper" in this family, so reusing it for the observer's bound would be actively confusing.
*Introduced by*: 260806-mkfj-dispatch-wait-event-driven

#### `--timeout 300` on the wiring preserves the peek-on-suspicion cadence exactly

**Decision**: The skill wiring passes `--timeout 300`, and a timeout return is defined as the peek-on-suspicion moment.
**Why**: The prior rule was "peek every 10th result-less poll" at a fixed 30s cadence — exactly 5 minutes. Making the timeout the cadence carrier keeps the recovery policy's timing identical while removing the poll turns that used to count it.
**Rejected**: A shorter or unbounded wait. Unbounded would drop peek-on-suspicion entirely (a worker parked at an error banner reads `running` forever); shorter would reintroduce turn cost for no policy benefit.
*Introduced by*: 260806-mkfj-dispatch-wait-event-driven

## Tasks

### Phase 1: Core Implementation

- [x] T001 Add the wait core to `src/go/fab/internal/dispatch/` — a `TickInterval` constant (2s) plus an `Observe`-style seam that derives one state from a loaded record + dir/stage, and a `Wait` loop that returns as soon as the observation is non-`running` or the bound expires; parameterize the tick and the bound so tests drive it fast <!-- R1 R2 R3 R4 R5 --> <!-- rework: on --timeout expiry the deadline arm returns the last-TICKED state without a final observe(), so wait can disagree with status in the sub-tick window before the bound (empirically confirmed); call observe() in the deadline arm before returning and propagate its error -->
- [x] T002 Add `src/go/fab/cmd/fab/dispatch_wait.go` — thin cobra `wait <change> <stage> [--timeout <secs>] [--json]`, resolving the dir via `resolveDispatchDir`, mirroring `status`'s error surface, and rendering through the **same** output path as `status` (shared render helper, single `dispatchStatusJSON` struct) <!-- R1 R3 R4 R6 -->
- [x] T003 Register `dispatchWaitCmd()` on the `dispatch` parent in `src/go/fab/cmd/fab/dispatch.go` and update the parent's `Short`/`Long` help to name seven subcommands <!-- R1 -->

### Phase 2: Refactor for single-sourcing

- [x] T004 Extract `status`'s state-derivation + render into a shared helper in `src/go/fab/cmd/fab/dispatch_status.go` (or the wait file) so `status` and `wait` provably share one derivation and one `--json` shape; leave `status`'s observable behavior byte-identical <!-- R2 R6 -->

### Phase 3: Tests

- [x] T005 [P] Table tests in `src/go/fab/internal/dispatch/dispatch_test.go` for the wait core: already-terminal fast path, timeout expiry returning `running`, and a state transition mid-wait waking the loop <!-- R7 --> <!-- rework: add the boundary case the suite misses — a scripted observer that turns terminal between the last tick and the bound must be reported terminal, not running (pins the deadline-arm observe fix); also add a golden-string assertion on raw `fab dispatch status` stdout for a headless and a pane record so the byte-identical claim (A-013) is actually pinned (should-fix folded in) -->
- [x] T006 [P] Command tests in `src/go/fab/cmd/fab/dispatch_wait_test.go`: already-terminal `done` returns immediately, `--timeout` expiry prints `running` and exits 0, `--json` parity with `status --json` (both modes), and the no-dispatch error <!-- R6 R7 -->
- [x] T007 Run `gofmt -l` on the new/changed Go files and `go test ./internal/dispatch/... ./cmd/fab/...`, fixing failures <!-- R7 -->

### Phase 4: Skill wiring

- [x] T008 Rewrite `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch step 2 to the background `fab dispatch wait --timeout 300` push wiring plus the foreground fallback, and update step 3's `running` row to the timeout-return framing <!-- R8 -->
- [x] T009 Update `src/kit/skills/_preamble.md` § Recovery policy — "every 10th result-less poll" → "every timeout-return of `fab dispatch wait`" (same ~5-min cadence) — and the pane-mode bullets that restate the `sleep 30` poll and the peek cadence, leaving budget/classification/escalation/never-send-keys semantics unchanged <!-- R9 -->
- [x] T010 [P] Update the restating dispatch-adapter passages in `src/kit/skills/_pipeline.md` and `src/kit/skills/fab-continue.md` to the wait wiring; verify `src/kit/skills/fab-adopt.md` needs no edit <!-- R10 -->
- [x] T011 [P] Add a `### wait` subsection to `src/kit/skills/_cli-fab.md` § fab dispatch and update the family enumeration from six to seven subcommands <!-- R11 -->

### Phase 5: Specs & mirror sweep

- [x] T012 Update `docs/specs/harness-adapters.md` — replace the fixed-`sleep 30`-cadence claim with the wait/push observation model, including the cross-harness foreground-blocking fallback <!-- R12 -->
- [x] T013 Update the SPEC mirrors `docs/specs/skills/SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-fab-continue.md`, and `SPEC-_cli-fab.md` for the wait wiring and the new verb <!-- R13 -->
- [x] T014 Grep the repo for `sleep 30` / `every 30s` / `10th result-less poll` and confirm every remaining occurrence is out of class (the `/git-pr-review` Copilot wait, historical findings docs, and memory files owned by hydrate) <!-- R13 -->

## Execution Order

- T001 blocks T002; T002 and T004 are interdependent (do T004 as part of / immediately after T002) and both block T006
- T003 follows T002
- T005 follows T001; T006 follows T002–T004; T007 follows T005 and T006
- Phase 4 tasks depend on nothing in Phases 1–3 but SHOULD land after the verb exists so the documented signature matches the code
- T013 follows T008–T011 (mirrors reflect the final skill text); T014 is last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab dispatch wait <change> <stage>` exists on the `dispatch` parent and blocks while the state is `running`, returning on the first non-`running` observation
- [x] A-002 R2: The wait path loads the record and derives state through `dispatch.Load` + `DeriveState`/`DerivePaneState` selected by `Dispatch.IsPane()` — no second derivation exists in the tree
- [x] A-003 R3: An already-terminal dispatch returns immediately without sleeping
- [x] A-004 R4: `--timeout <secs>` bounds the block; expiry prints `running` and exits 0; absent/zero waits indefinitely — *the deadline arm now observes once more and returns THAT result (propagating its error), so expiry reports the genuinely still-current state rather than the last-ticked one, closing the sub-tick disagreement with `status` and the coincident-readiness race; pinned by `TestWaitTimeoutObservesAtTheBound` + `TestWaitPropagatesObserveErrorAtTheBound`, both verified to FAIL against the pre-fix arm*
- [x] A-005 R5: `dispatch.TickInterval` is a named 2s package constant, the wait core takes the tick as a parameter, and no filesystem watcher is used
- [x] A-006 R6: `wait --json` emits the same object shape as `status --json` through the same code path, and the no-dispatch error message is identical to `status`'s
- [x] A-007 R8: `_preamble.md` § CLI-Adapter Dispatch step 2 instructs a background `fab dispatch wait --timeout 300` with a documented foreground fallback
- [x] A-008 R11: `_cli-fab.md` § fab dispatch documents `wait` and enumerates seven subcommands
- [x] A-009 R12: `harness-adapters.md` describes the wait/push observation model including the cross-harness foreground fallback
- [x] A-010 R13: All four SPEC mirrors (`SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-fab-continue.md`, `SPEC-_cli-fab.md`) carry the wait wiring — *plus `SPEC-fab-adopt.md`, correctly swept alongside the `fab-adopt.md` edit. All five carry it; one residual clause inside `SPEC-_preamble.md` ("then resumes polling") was missed by the rewrite of that same line — see A-011 and the review findings*

### Behavioral Correctness

- [x] A-011 R9: The recovery policy's restart budget, `orphaned`/`failed`/`failed (no-result)` rules, three-way peek classification, escalation, and never-send-keys rule are unchanged in substance — only the cadence carrier is now the `wait` timeout return — *verified clause-by-clause against the canonical `_preamble.md`: budget, the three state rules, the (a)/(b)/(c) classification, escalation-with-gated-`rk notify`, and peek/kill/restart/notify/stop-with-no-send-keys all stand. Cycle-2 residual: the `SPEC-_preamble.md:11` mirror still renders the `orphaned` bullet's tail as "then resumes polling" where the canonical file now reads "after which the orchestrator re-arms `wait`" — the SEMANTICS are unchanged (that is exactly what A-011 asserts), but the mirror's cadence CARRIER wording is stale (see review findings)*
- [x] A-012 R10: `_pipeline.md` and `fab-continue.md` restate the wait wiring, and `fab-adopt.md` is verified unchanged — *the plan's prediction was wrong: `fab-adopt.md` DID carry a stale "whose poll" phrase and was correctly updated, with its `SPEC-fab-adopt.md` mirror swept alongside. Implementation is more correct than the plan predicted*
- [x] A-013 R6: `fab dispatch status` output (text and `--json`) is byte-identical to before the shared-render refactor — *`dispatch_status_test.go`'s six pre-existing tests are untouched by the diff and pass against the refactor; the claim is now PINNED for the future by `TestDispatchStatus_GoldenOutput`, which asserts raw stdout as exact bytes (no TrimSpace, no unmarshal-then-compare) for headless and pane records in both text and `--json` form, so a changed trailing newline, JSON indent, or key order would fail*

### Scenario Coverage

- [x] A-014 R7: Tests exercise the already-terminal fast path, the timeout path, and a mid-wait state transition, and `go test ./internal/dispatch/... ./cmd/fab/...` passes — *all pass; `gofmt -l` and `go vet` clean. The timeout-boundary case is now covered too: `TestWaitTimeoutObservesAtTheBound` drives a time-gated observer that turns terminal after the last tick but before the bound (tick set longer than the test so only the entry and deadline observations occur), and `TestWaitPropagatesObserveErrorAtTheBound` pins the deadline arm's error propagation*
- [x] A-015 R13: A repo-wide grep for `sleep 30` / `10th result-less poll` returns no in-class occurrence; `SPEC-git-pr-review.md` is untouched — *verified: remaining hits are the `/git-pr-review` Copilot wait (out of class, untouched), historical findings docs, deliberate "no longer a sleep 30 poll" framing, `dispatch_start/restart/kill_test.go` fixture commands, and hydrate-owned `docs/memory/`*

### Edge Cases & Error Handling

- [x] A-016 R6: An unresolvable change reference or a missing dispatch record exits non-zero with `status`'s message; timeout expiry does not — *shares `resolveDispatchDir` + `loadDispatchRecord`; pinned by `TestDispatchWait_NoDispatchErrors` comparing both error strings*
- [x] A-017 R5: A headless worker dying with no exit file is observed as `orphaned` by the liveness probe within one tick, ending the wait

### Code Quality

- [x] A-018 Pattern consistency: The new files follow the `internal/dispatch` pure-function + thin-cobra split, named constants over magic numbers, and the surrounding doc-comment style; `gofmt` is clean
- [x] A-019 No unnecessary duplication: `wait` reuses the existing loader, derivations, liveness probes, `resolveDispatchDir`, and `status`'s JSON struct/render path rather than reimplementing any of them — *verified: `DeriveState`/`DerivePaneState` have exactly one non-test call site each (`observeDispatch`)*
- [x] A-020 Canonical source only: No file under `.claude/skills/` is edited; every skill change lands in `src/kit/skills/`
- [x] A-021 SPEC-mirror sync: Every edited `src/kit/skills/*.md` carries its `docs/specs/skills/SPEC-*.md` update in this same change — *all five: `_preamble`, `_pipeline`, `fab-continue`, `_cli-fab`, `fab-adopt`*
- [x] A-022 CLI ⇒ docs + tests: The command-signature change updates `_cli-fab.md` and ships Go tests
- [x] A-023 No user-data restructuring: No `.status.yaml` / config / archive-layout change, so no `src/kit/migrations/` file is required

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab/internal/dispatch/wait.go:90-91` (the `case <-ctx.Done():` arm of `Wait`) — dead in production: `cmd/fab/main.go:109` calls `root.Execute()`, never `ExecuteContext`, so cobra fills `cmd.Context()` with a plain `context.Background()` that is never cancelled (confirmed at `cobra@v1.8.1/command.go:1055-1056`), and `dispatch_wait.go:77` is the tree's ONLY `cmd.Context()` consumer. Reachable only from `TestWaitHonorsContextCancellation`. Keep it if the root is ever upgraded to signal-aware `ExecuteContext` (a reasonable near-term move given this is now the binary's only long-blocking command); delete the arm and its test otherwise.
- No other candidates — `wait` is purely additive. Re-verified this cycle: every new symbol (`TickInterval`, `Observer`, `Wait`, `loadDispatchRecord`, `observeDispatch`, `renderDispatchState`, `dispatchWaitCmd`, `runDispatchWait`) has live non-declaration call sites, and `status`, `start`, `restart`, `kill`, `clean`, `logs`, `DeriveState`, `DerivePaneState`, and `Load` all retain theirs. The shared-helper extraction in `dispatch_status.go` replaced duplicated inline code rather than leaving an orphan behind. The `sleep 30` poll wiring it supersedes was prose in skill files, already rewritten in place (not deletable code).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Package-level API is `dispatch.TickInterval` (2s constant) plus an exported `Wait` loop taking an observation function, a tick, and a bound — mirroring the package's existing pure-function-plus-thin-cobra split | The intake fixes the behavior and the tick value; the shape follows `SelectMode`/`DeriveState`/`SelectPaneShape` precedent in the same file, so it is the only idiomatic choice here | S:85 R:90 A:95 D:90 |
| 2 | Confident | `status`'s per-mode derivation + JSON render is extracted into a shared helper that both `status` and `wait` call, rather than duplicating the branch in `dispatch_wait.go` | R2/R6 demand the two verbs cannot disagree and that `--json` is single-sourced; the code-quality anti-pattern list explicitly names duplicating existing utilities. `status`'s observable output stays byte-identical | S:75 R:85 A:95 D:85 |
| 3 | Confident | `--timeout 0` (explicitly zero) is treated the same as absent — an unbounded wait — rather than as an instant timeout | The intake says "absent → wait indefinitely" and defines timeout as an upper bound; an instant-return-on-zero reading has no consumer and would silently no-op the wiring if a variable expanded empty | S:65 R:90 A:85 D:80 |
| 4 | Confident | The wait loop derives state once at entry before its first sleep, so the already-terminal fast path costs zero ticks | R3 requires immediate return; deriving-then-sleeping is the only ordering that satisfies it, and it also makes the common re-arm-after-restart case free | S:80 R:95 A:95 D:90 |
| 5 | Confident | The wait core takes the tick interval as a parameter (defaulting to `TickInterval` at the cobra layer) so tests run in milliseconds instead of real seconds | Without it, the transition test would need multi-second sleeps; parameterizing is the package's existing testability idiom (every derivation is pure and injectable) | S:70 R:95 A:95 D:85 |
| 6 | Confident | The pane-mode `PaneAlive` probe runs on every tick just as the headless `Alive` probe does, with no extra rate limiting | Symmetry with `status`, which probes on every invocation; a `tmux list-panes` every 2s is negligible next to the inference turns being eliminated, and rate-limiting it would delay `orphaned` detection — the intake's stated latency win | S:60 R:85 A:85 D:75 |
| 7 | Confident | `_cli-fab.md`'s `wait` subsection is placed after `status` (the verb it shares its derivation and output with) rather than at the end of the family | The family enumeration is start/restart/status/logs/kill/clean; `wait` reads as `status`'s blocking sibling, so adjacency aids the reader. Purely presentational | S:60 R:100 A:90 D:85 |
| 8 | Tentative | Memory files listed in the intake's Affected Memory are left untouched by apply | The apply block's scope is code + skills + specs; `docs/memory/` is written at hydrate per the pipeline contract, and the invoking prompt says so explicitly. Recorded as an assumption because their stale "sleep 30" text will show in a repo-wide grep until hydrate runs | S:70 R:95 A:80 D:70 |

8 assumptions (1 certain, 6 confident, 1 tentative).
