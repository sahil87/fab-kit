# Plan: Pane Identity Keying Contract

**Change**: 260822-m461-pane-identity-keying-contract
**Intake**: `intake.md`

## Requirements

### Dispatch: Restart-alias liveness discriminator

#### R1: Record `pane_pid` at open-time
The `Dispatch` record (`src/go/fab/internal/dispatch/dispatch.go`) SHALL gain an optional field `PanePID int` (`yaml:"pane_pid,omitempty"`) — the pane's shell pid, read via the existing `pane.GetPanePID` (`src/go/fab/internal/pane/pane.go:284`, `#{pane_pid}`) — recorded by `launchPane` (`src/go/fab/cmd/fab/dispatch_start.go`) immediately after the pane ID is obtained, for both pane shapes (split and new-window). A failed pid read MUST NOT fail the launch: it warns on stderr and leaves the field zero (absent on disk via `omitempty`), which downgrades that dispatch to today's existence-only liveness.

- **GIVEN** a reachable tmux server and a provider with an `interactive_command`
- **WHEN** `fab dispatch open <change> <stage>` runs
- **THEN** `{stage}.yaml` carries `pane_pid: <N>` equal to the new pane's `#{pane_pid}`
- **AND** GIVEN the pid read fails, the launch still succeeds with a one-line stderr warning and the record carries no `pane_pid` key

#### R2: Identity-checked worker liveness is a pure, table-testable decision
A pure function in `internal/dispatch` (no I/O — the `SelectMode`/`DeriveState`/`DecideReap` precedent) SHALL decide worker identity: given (pane exists, recorded pid, current pid, pid-read-succeeded), the worker is alive iff the pane exists AND (no pid was recorded — legacy/degraded record — OR the pid read failed — cannot disprove identity, degrade to existence-only — OR current pid == recorded pid). A cmd-layer helper (`paneWorkerAlive(rec, dir…)` beside `observeDispatch` in `src/go/fab/cmd/fab/dispatch_status.go`) SHALL compose it from `pane.PaneAlive` + `pane.GetPanePID` and be the single shared implementation for every record-keyed consumer. `DerivePaneState` itself is NOT modified — it is a documented byte-stable cross-adapter contract; the identity check lives in the computation of its `paneAlive` input.

- **GIVEN** a record with `pane_pid: 100` and an existing pane whose current `#{pane_pid}` is 245
- **WHEN** the identity decision runs
- **THEN** the worker reads NOT alive (the pane is an impostor)
- **AND** GIVEN a record with no `pane_pid` (older binary), or a pid read that errors on an existing pane, the worker reads alive (existence-only back-compat)

#### R3: Observation seams treat a mismatched pane as a gone worker
Every liveness observation of a pane record SHALL use the R2 identity-checked helper in place of bare `pane.PaneAlive(rec.Pane, rec.Server)`: `observeDispatch` (`src/go/fab/cmd/fab/dispatch_status.go:112` — shared by `status` and `wait`), `runDispatchReap`'s state derivation (`src/go/fab/cmd/fab/dispatch_reap.go:68`), and `priorRunning` (`src/go/fab/cmd/fab/dispatch_start.go:675` — refuse-if-running for `start`/`open`/`restart`). A restart-aliased pane (result absent, pane ID exists, pid mismatch) therefore derives `orphaned` — the existing recovery path — instead of `running`, and a new launch over it is not refused.

- **GIVEN** a pane dispatch whose tmux server restarted mid-flight, where the recorded `%17` now names an unrelated pane and `{stage}-result.yaml` is absent
- **WHEN** `fab dispatch status` or `wait` observes it
- **THEN** the state is `orphaned`, not `running`
- **AND** WHEN `fab dispatch open`/`restart` runs for the same (change, stage), it overwrites the completed-or-orphaned attempt rather than refusing with "already running"

#### R4: Targeting verbs never touch a mismatched pane
The verbs that aim a keystroke, signal, or pointer at the recorded pane SHALL treat an identity mismatch as "the worker is gone", never acting on the impostor: `kill` (`src/go/fab/cmd/fab/dispatch_kill.go:48` — already-dead report, no `kill-pane` sent), `reap`'s already-gone check (`src/go/fab/cmd/fab/dispatch_reap.go:99`), `ready`/`deliver`'s dead-pane refusal (`loadPaneDispatch`, `src/go/fab/cmd/fab/dispatch_ready.go:109` — refuses naming `fab dispatch restart`, so no sentinel or pointer is ever typed into an unrelated pane), and `logs`' printed capture command (`src/go/fab/cmd/fab/dispatch_logs.go:44` — reports the worker gone instead of printing a `fab pane capture` hint that targets the impostor).

- **GIVEN** a pane record whose pane ID exists but whose `pane_pid` mismatches
- **WHEN** `fab dispatch kill` runs
- **THEN** it reports already-dead and sends no `tmux kill-pane`
- **AND** WHEN `ready` or `deliver` runs, it refuses naming `fab dispatch restart` and types nothing
- **AND** WHEN `logs` runs, its output does not name the impostor pane as a capture target

#### R5: `status --json` exposes the discriminator additively
`dispatchStatusJSON` SHALL gain `pane_pid` (`json:"pane_pid,omitempty"`, pane-only, absent when unrecorded) — the additive-evolution precedent (`mode`/`server`/`delivered`). `src/kit/skills/_cli-fab.md` § fab dispatch SHALL document the new record field, the `--json` key, and the identity-checked liveness sentence (constitution: CLI change ⇒ `_cli-fab.md` + tests).

- **GIVEN** a pane dispatch opened by the new binary
- **WHEN** `fab dispatch status --json` runs
- **THEN** the object carries `pane_pid`; a headless dispatch, or a pane record without the field, omits the key

#### R6: Back-compat — absent discriminator keeps today's behavior; no migration
Records written by older binaries (no `pane_pid`) SHALL behave exactly as today at every consumer (existence-only liveness). No `src/kit/migrations/` file ships: `.fab-dispatch/` is transient per-change runtime state (archive-time deletion + `fab dispatch clean`), and the field is additive `omitempty` — the migration rule covers config/`.status.yaml`/archive restructuring, not this.

- **GIVEN** a pre-existing pane record with no `pane_pid` key
- **WHEN** any of status/wait/kill/reap/ready/deliver/refuse-if-running observes it
- **THEN** behavior is byte-identical to the pre-change binary

### Pane map: identity-key contract declaration (docs)

#### R7: `fab pane map` help declares the identity-key contract
`src/go/fab/cmd/fab/panemap.go`'s command SHALL gain `Long` help text declaring: the row schema inherits rk's contract (`rk mux panes --json` is the primary declaration — a run-kit companion item); `pane` (with `server` context) and `window_id` are the identity keys; `session` and `window_index` are DISPLAY-ONLY and MUST NOT be used as join keys (positional, reassigned by `swap-window`/`move-window`/rename — the run-kit StatusDot swap-lag misattribution bug); consumers MUST tolerate `window_id: ""` (legacy line).

- **GIVEN** `fab pane map --help`
- **WHEN** a consumer author reads it
- **THEN** the identity-vs-display split, the misattribution citation, and the `window_id: ""` tolerance are all stated

#### R8: `_cli-fab.md` § fab pane map splits identity keys from display columns
The JSON field table at `src/kit/skills/_cli-fab.md:538` — which currently lumps `session`, `window_index`, `pane` … as "Table-equivalent identity/context fields" — SHALL be split: identity keys (`pane` + `server` scope, `window_id`) vs display-only fields (`session`, `window_index`, `tab`, …), with the same contract sentence, misattribution citation, and legacy-`window_id` tolerance as R7.

- **GIVEN** the updated § fab pane map field table
- **WHEN** a skill consumer decides how to join rows to a tmux snapshot
- **THEN** the table itself names `pane`/`window_id` as the only join keys and marks `session`/`window_index` display-only

### Sweep: index-join audit

#### R9: No fab-kit surface correlates panes by `session:window_index` or assumes session/index stability
A repo-wide sweep (Go, `src/kit/skills/`, `docs/`) SHALL find and fix-or-annotate (a) any consumer correlating panes by `session` + `window_index` instead of `%pane_id`/`window_id`, and (b) any assumption that a pane's session or index is stable across its lifetime (run-kit's planned `_rk-operator` relocation via `move-window` violates both). Planning-time grounding found: `_cli-fab.md:538` (fixed by R8); `src/kit/skills/fab-operator.md` `(session, repo, pane)` tuple (§1 row ~line 34, §4 ~line 201) — pane ID is already documented as the server-global primary key (conforms), so ANNOTATE that `session` is a display/context dimension, never a join key, and that a monitored agent's session can change mid-lifetime; Go verified clean (`panemap.go` keeps `window_index` as a display/JSON column only; `operator.go:116` deliberately carries no session field). The sweep MUST re-run at apply per the code-quality sweep classes: `*_test.go` comments, contrastive phrases, and user-facing string literals included.

- **GIVEN** the apply-time sweep greps (`session:window`, `window_index`, `#{session_name}:#{window_index}`, index/session-stability phrases) over `src/go/`, `src/kit/skills/`, `docs/`
- **WHEN** the results are triaged
- **THEN** every join-key use or stability assumption is fixed or annotated against the R7/R8 contract, and display-column uses are left as-is (Non-Goals)

### Cross-repo: `.status.yaml` disk contract (hydrate-executed)

#### R10: The run-kit disk-read contract is recorded in memory at hydrate
Hydrate SHALL record (per the intake's Affected Memory list — no apply task; memory edits are hydrate's locus): in `docs/memory/pipeline/schemas.md`, that run-kit's `fabstate.go` parses `.status.yaml`'s `progress` map (stage states → change/stage/display-state) directly from disk, making those fields a public cross-repo API — schema changes are breaking for run-kit, coordinate before changing; in `docs/memory/pipeline/change-lifecycle.md`, that the `.fab-status.yaml` worktree-root symlink convention (target `fab/changes/<name>/.status.yaml`; change name = target's parent dir basename) is an external consumer contract for the same reader. Cross-linked, following the `runtime-agents.md` cross-repo-convention precedent. The other three Affected Memory files carry the R1–R8 outcomes.

- **GIVEN** the hydrated memory
- **WHEN** a future change edits the `progress` map schema or the symlink convention
- **THEN** the owning memory file warns that run-kit reads it from disk and requires coordination

### Non-Goals

- No change to panemap's cwd-based change/stage resolution (already correct).
- No removal of `session`/`window_index` from table or JSON output — they stay as display columns.
- No fab enrichment (change/stage/PR) moved into `rk mux panes`: rk owns rows, fab owns meaning, disk is the interface.
- No rk-side code work (the `rk mux panes` contract declaration is a run-kit backlog item).
- No change to `SiblingDispatchPane` placement — its live-pane intersection already filters aliased records.
- No change to `DerivePaneState`'s signature or derivation table (byte-stable contract; identity rides its input).

### Design Decisions

#### Discriminator is the pane shell pid, not a start time
**Decision**: Record `#{pane_pid}` at open-time as the liveness discriminator.
**Why**: tmux 3.7c exposes no `pane_start_time` format variable (verified via man page — only `pane_pid`/`pane_start_command`/`pane_start_path` exist); the shell pid is tmux-native, stable for the pane's lifetime, and `GetPanePID` already exists.
**Rejected**: `pane_start_time` (does not exist in tmux); recording the spawn timestamp fab-side (proves nothing about the pane the ID currently names).
*Introduced by*: 260822-m461-pane-identity-keying-contract

#### Identity check computes the `paneAlive` input; the state machine is untouched
**Decision**: `DerivePaneState` stays pure and byte-stable; identity-checked liveness replaces the bare `PaneAlive` call at each consumer via one shared helper.
**Why**: The three-state derivation is a documented cross-adapter contract with in-code contract comments; changing its signature would ripple through the harness-adapters spec for zero semantic gain.
**Rejected**: A fourth parameter on `DerivePaneState` (couples identity into a documented byte-stable machine); per-call-site inline checks (six sites — drift liability).
*Introduced by*: 260822-m461-pane-identity-keying-contract

#### A failed pid read degrades to existence-only, never to orphaned
**Decision**: Both at record time (open) and at check time, a pid-read failure downgrades that observation to today's existence-only behavior (with a warning at record time).
**Why**: An unreadable pid is not a mismatch — treating it as one would false-orphan live workers on transient tmux errors and burn the one-restart recovery budget; the degraded mode is exactly the legacy-record mode, so it adds no new behavior class.
**Rejected**: Conservative not-alive on read failure (false orphans); failing the open (a cosmetic-adjacent read must not kill a launch that already succeeded).
*Introduced by*: 260822-m461-pane-identity-keying-contract

#### No migration for the new record field
**Decision**: Ship `pane_pid` as additive `omitempty` with no `src/kit/migrations/` file.
**Why**: `.fab-dispatch/` is transient per-change runtime state (archive-deleted, `clean`-able), not user data being restructured; absent field = today's behavior by construction.
**Rejected**: A migration stamping pids into existing records (a stored pid for a pane opened before the migration is unverifiable and could itself alias).
*Introduced by*: 260822-m461-pane-identity-keying-contract

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add `PanePID int` (`yaml:"pane_pid,omitempty"`) to `Dispatch` in `src/go/fab/internal/dispatch/dispatch.go` with a doc comment stating the restart-alias rationale; add the pure identity-liveness decision function beside `DerivePaneState` and exhaustive table tests in `src/go/fab/internal/dispatch/dispatch_test.go` (gone / legacy-zero / pid-read-failed / match / mismatch) <!-- R1, R2 -->
- [x] T002 Record the discriminator in `launchPane` (`src/go/fab/cmd/fab/dispatch_start.go`): after the pane ID is obtained, `pane.GetPanePID(paneID, server)` → `rec.PanePID`; warn-nonfatal on error; cover in `src/go/fab/cmd/fab/dispatch_open_test.go` (field present and matching; warning path leaves it absent) <!-- R1 -->
- [x] T003 Add the shared cmd-layer helper `paneWorkerAlive` beside `observeDispatch` in `src/go/fab/cmd/fab/dispatch_status.go` (composes `pane.PaneAlive` + `pane.GetPanePID` + the T001 pure function); switch `observeDispatch` and `priorRunning` (`dispatch_start.go`) onto it; tests in `dispatch_status_test.go` / `dispatch_start_test.go` for the mismatch→`orphaned` and mismatch→overwritable paths <!-- R2, R3 -->
- [x] T004 Switch `runDispatchReap` (both call sites, `src/go/fab/cmd/fab/dispatch_reap.go`) and `runDispatchKill`'s already-dead gate (`src/go/fab/cmd/fab/dispatch_kill.go`) onto `paneWorkerAlive`; tests: mismatched pane → already-gone report, no `kill-pane` sent <!-- R3, R4 -->
- [x] T005 Switch `loadPaneDispatch` (`src/go/fab/cmd/fab/dispatch_ready.go`) onto `paneWorkerAlive` (mismatch → the existing gone/`restart` refusal) and guard `logs`' capture hint (`src/go/fab/cmd/fab/dispatch_logs.go`): identity-mismatched worker → report gone instead of printing the capture command; tests in `dispatch_ready_test.go` / `dispatch_deliver_test.go` / `dispatch_logs_test.go` <!-- R4 -->
- [x] T006 Add `PanePID *int`-or-`int` (`json:"pane_pid,omitempty"`, pane-only) to `dispatchStatusJSON` and populate it in `observeDispatch`; extend the `--json` shape test in `src/go/fab/cmd/fab/dispatch_status_test.go` (present for a new pane record, absent for headless and legacy records) <!-- R5 -->

### Phase 3: Integration & Edge Cases

- [x] T007 Add `Long` help to `fab pane map` in `src/go/fab/cmd/fab/panemap.go` declaring the identity-key contract (inherits rk's contract; `pane`+`server`/`window_id` identity; `session`/`window_index` display-only, never join keys — StatusDot swap-lag citation; `window_id: ""` legacy tolerance) <!-- R7 -->
- [x] T008 Update `src/kit/skills/_cli-fab.md`: split the § fab pane map JSON field table (line ~538) into identity keys vs display-only with the contract sentence; document in § fab dispatch the `pane_pid` record field, the `status --json` key, and identity-checked pane liveness (mismatch ⇒ orphaned; kill/ready/deliver/logs never target a mismatched pane) <!-- R8, R5 -->
- [x] T009 Annotate `src/kit/skills/fab-operator.md`'s `(session, repo, pane)` addressing tuple (§1 Multi-repo row ~line 34 and §4 ~line 201): `session` is a display/context dimension, never a join key; a monitored agent's session can change mid-lifetime (rk operator-relocation direction); pane ID remains the server-global primary key <!-- R9 -->
- [x] T010 Re-run the index-join sweep repo-wide (`session:window`, `window_index`, `#{session_name}:#{window_index}`, session/index-stability phrases over `src/go/`, `src/kit/skills/`, `docs/` — including `*_test.go` comments, contrastive phrases, and user-facing string literals per code-quality § Sibling Sweeps); fix or annotate any site T007–T009 did not already cover; record `None found` in `## Notes` if clean <!-- R9 -->

### Phase 4: Polish

- [x] T011 Run the affected Go tests: `go test ./internal/dispatch/... ./internal/pane/... ./cmd/fab/` (scoped first; widen if cross-cutting failures appear) <!-- R1, R2, R3, R4, R5, R6 -->

## Execution Order

- T001 blocks T002–T006 (field + pure function first)
- T003 blocks T004–T005 (shared helper first)
- T007–T010 are independent of the Go chain and of each other ([P]-equivalent; kept sequential for sweep coherence: T008/T009 land before T010's re-grep)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab dispatch open` writes `pane_pid` matching the pane's `#{pane_pid}` in both pane shapes; a failed pid read warns and leaves the key absent while the launch succeeds
- [x] A-002 R2: the pure identity decision exists in `internal/dispatch` with table tests covering gone / legacy-zero / pid-read-failed / match / mismatch; `DerivePaneState` is unmodified
- [x] A-003 R3: `status` and `wait` report `orphaned` for a record whose pane exists but whose current pid mismatches the recorded `pane_pid`
- [x] A-004 R4: `kill` and `reap` on a mismatched pane report already-gone and send no `tmux kill-pane`; `ready`/`deliver` refuse naming `fab dispatch restart`; `logs` prints no capture command targeting the impostor
- [x] A-005 R5: `status --json` carries `pane_pid` for a new pane record and omits it for headless and legacy records; `_cli-fab.md` § fab dispatch documents the field
- [x] A-006 R7: `fab pane map` `Long` help states the identity-vs-display contract, the misattribution citation, and the `window_id: ""` tolerance
- [x] A-007 R8: the `_cli-fab.md` § fab pane map field table separates identity keys from display-only fields with the contract sentence
- [x] A-008 R9: `fab-operator.md` carries the session-is-display-context annotation; the repo-wide sweep ran with its classes and every hit is fixed, annotated, or recorded clean

### Behavioral Correctness

- [x] A-009 R6: a record with no `pane_pid` behaves byte-identically to the pre-change binary at every consumer (status/wait/kill/reap/ready/deliver/refuse-if-running)
- [x] A-010 R3: refuse-if-running does not refuse over a mismatched (aliased) prior pane record — a fresh launch overwrites it

### Scenario Coverage

- [x] A-011 R3: a test exercises the restart-alias scenario end-to-end at the derivation seam (result absent + pane ID exists + pid mismatch ⇒ `orphaned`)

### Edge Cases & Error Handling

- [x] A-012 R2: a pid-read failure on an existing pane degrades to existence-only liveness (no false `orphaned`), matching the legacy-record mode

### Code Quality

- [x] A-013 Pattern consistency: new code follows the package's pure-decision + thin-cobra-wiring precedent (`SelectMode`/`DecideReap` shape); naming and comment density match surrounding code
- [x] A-014 No unnecessary duplication: reuses `GetPanePID`, `PaneAlive`, `loadDispatchRecord`; one shared liveness helper, no per-site copies
- [x] A-015 CLI ⇒ docs + tests: the `fab` binary change updates `src/kit/skills/_cli-fab.md` and ships test updates alongside (constitution Additional Constraints)
- [x] A-016 Canonical source only: all skill edits land under `src/kit/skills/`, none under `.claude/skills/`
- [x] A-017 **N/A**: the new field is additive `omitempty` on transient `.fab-dispatch/` state — no user-data restructuring, so no migration (rationale recorded in Design Decisions)
- [x] A-018 Owner-or-pointer: skill/doc edits state the contract at its owner or point to it, never both

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- R10 has no task or acceptance item by design: memory edits are hydrate's execution locus (review runs pre-hydrate); the requirement text is what hydrate consumes.
- **T010 sweep record (apply, 2026-08-22)**: greps `session:window`, `window_index`, `#{session_name}:#{window_index}`, and session/index-stability phrases over `src/go/`, `src/kit/skills/`, `docs/` (incl. `*_test.go` comments and user-facing literals). Result: **no unconforming site**. `_cli-fab.md:538` fixed by R8; `fab-operator.md` annotated by R9. All remaining hits are conforming: `panemap.go` carries `session`/`window_index` as display/JSON columns only (rk-row mapping + tmux format string, no joins); test files assert field presence/position only; `docs/memory/runtime/pane-commands.md:46` already documents the index-join misattribution rationale (contrastive); archived/closed change docs and the sibling plan doc are historical records; `fab memory-index` hits are an unrelated word sense.

## Deletion Candidates

- None — this change replaces the six bare `pane.PaneAlive(rec…)` liveness reads in place with the shared `paneWorkerAlive` helper; `pane.PaneAlive` itself stays live (used by the helper, `dispatch_logs.go`'s alias guard, and tests), and no file, function, branch, or config was made redundant or unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Pure identity function in `internal/dispatch` + one shared cmd-layer `paneWorkerAlive` helper (six call sites, five files) | Matches the package's pure-decision precedent; cmd layer already composes `PaneAlive` per record; single helper kills drift | S:70 R:85 A:85 D:75 |
| 2 | Confident | Pid-read failure degrades to existence-only (alive), never orphaned | An unreadable pid is not a mismatch; false orphans would burn the one-restart budget; degraded mode = legacy mode, no new behavior class | S:60 R:80 A:80 D:70 |
| 3 | Confident | `logs`' capture hint is in scope as a targeting surface (R4) | Intake names "any peek/capture pointer the record produces, e.g. `dispatch logs`' printed capture command" explicitly | S:80 R:90 A:85 D:85 |
| 4 | Certain | R10 (cross-repo contract docs) executes at hydrate via Affected Memory, with no apply task or review acceptance item | Pipeline convention: memory edits are hydrate's locus; review precedes hydrate so an acceptance item would be unverifiable | S:85 R:90 A:90 D:85 |
| 5 | Confident | `status --json` gains `pane_pid` (additive) even though no consumer needs it yet | The record surface and its JSON mirror have moved together every time (`mode`/`server`/`delivered` precedent); constitution's CLI⇒docs rule wants the observable surface honest | S:65 R:85 A:80 D:70 |

5 assumptions (1 certain, 4 confident, 0 tentative).
