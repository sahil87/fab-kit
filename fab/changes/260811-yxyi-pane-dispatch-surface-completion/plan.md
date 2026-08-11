# Plan: Pane/Dispatch Surface Completion

**Change**: 260811-yxyi-pane-dispatch-surface-completion
**Intake**: `intake.md`

## Requirements

### CLI: `fab pane kill` Verb

#### R1: `fab pane kill <pane>` exposes the existing `pane.KillPane` helper
The `fab pane` command group SHALL gain a ninth subcommand `kill <pane>` (new `src/go/fab/cmd/fab/pane_kill.go`, registered on `paneCmd` in `src/go/fab/cmd/fab/pane.go`, inheriting the persistent `--server`/`-L` flag). It SHALL validate with `pane.ValidatePane` first — a missing pane prints `Error: pane <id> not found` and exits **2** in-handler via the existing `*PaneNotFoundError`/`paneValidationExitCode` path; any other tmux failure exits **3**. On success it SHALL print `killed <pane>` plus a `server: <name>` line when the socket is non-default (matching `open`'s form). It SHALL be record-free: no dispatch-record interaction, no `.fab-dispatch/` state. `fab dispatch kill` (record-keyed, ungated recovery) is unaffected.

- **GIVEN** a live tmux pane `%5` on the default socket
- **WHEN** `fab pane kill %5` runs
- **THEN** the pane is killed, stdout reads `killed %5`, and the exit code is 0
- **AND** **GIVEN** no pane `%999`, **WHEN** `fab pane kill %999` runs, **THEN** stderr reads `Error: pane %999 not found` and the exit code is 2; a dead socket exits 3

### CLI: `fab pane await` Verb

#### R2: `fab pane await <pane> [--file <path>] [--timeout <secs>]` is the record-free blocking wait
The `fab pane` group SHALL gain a tenth subcommand `await` (new `src/go/fab/cmd/fab/pane_await.go`) that blocks until ANY waitable signal fires, prints a one-word report, and exits 0 — the report string is the discriminator (the `fab dispatch wait` precedent):

| Report | Fired by |
|--------|----------|
| `idle` | the pane's `@rk_agent_state` resolves to `idle` (the shared parser, same read as `map`/`send`/`capture`) |
| `file` | the `--file <path>` file exists |
| `running` | `--timeout` expired with neither signal — **exit 0** (timeout bounds the observer, not the worker) |
| `gone` | the pane died mid-wait — **exit 2** (the wait cannot complete; the caller branches on cause) |

Signals SHALL compose as OR (with both `--file` and an instrumented pane, whichever fires first wins). An uninstrumented pane (agent state unknown/absent) with NO `--file` SHALL be an immediate error (exit 1 via RunE) — there is nothing observable to wait on. Cadence/bounds SHALL mirror `dispatch wait`'s observer conventions: an internal ~2s re-derive tick (no flag, no config) and `--timeout` default 300 (0 or negative = unbounded); the FIRST observation happens before any sleep. Pane missing at entry → exit 2; other tmux failure → exit 3. The control loop SHALL live in `internal/pane` as a pure, observer-injected loop (the `dispatch.Wait` precedent) so it is unit-testable without tmux.

- **GIVEN** a pane whose `@rk_agent_state` is `idle:<epoch>`
- **WHEN** `fab pane await <pane>` runs
- **THEN** it prints `idle` and exits 0 immediately (no tick consumed)
- **AND** **GIVEN** an uninstrumented pane and no `--file`, **WHEN** `fab pane await <pane>` runs, **THEN** it errors immediately (exit 1) instead of blocking

### CLI: `fab dispatch status --json` Server Field

#### R3: `dispatchStatusJSON` gains a `server` field
`src/go/fab/cmd/fab/dispatch_status.go`'s `dispatchStatusJSON` SHALL gain `Server string \`json:"server,omitempty"\``, sourced from the record's `Server` field (populated for pane dispatches that carried a socket; omitted when empty). The change is additive — existing consumers ignore unknown keys (the `repo`/`window_id`/`pr_url` precedent). Human output SHALL be unchanged.

- **GIVEN** a pane dispatch started with `--server work`
- **WHEN** `fab dispatch status <change> <stage> --json` runs
- **THEN** the object carries `"server": "work"`
- **AND** **GIVEN** a default-socket or headless dispatch, **THEN** the `server` key is absent from the JSON

### CLI: `--json` on `fab pane open`/`ready`/`deliver`

#### R4: the three provider-generic verbs emit a single JSON object under `--json`
`src/go/fab/cmd/fab/pane_open.go`, `pane_ready.go`, and `pane_deliver.go` SHALL each gain a `--json` bool emitting one JSON object on stdout (always an object, including for every classification — the `window-name` precedent):

- `open`: `{"pane": "%12", "provider": "claude", "server": null}` — empty server → `null` via the existing `toNullable` contract
- `ready`: `{"state": "parked", "pane": "%12", "server": null, "snippet": "…"}` — `state` ∈ `{ready, booting, parked}`; `snippet` is the same trailing-blank-trimmed 20-line capture the text report carries (`""` when empty); all three states still exit 0
- `deliver`: `{"pane": "%12", "source": "prompt", "path": ".fab-dispatch/yxyi/apply-prompt.md"}` — `source` ∈ `{prompt, text}`; `path` present only for `prompt`; JSON is emitted only on verified delivery (failures keep the existing stderr + non-zero contract)

Plain-text output SHALL remain byte-identical to today in all three verbs. Scope is the `fab pane` primitives only — the `fab dispatch open/ready/deliver` bindings keep their text-only output.

- **GIVEN** a live pane at an idle shell prompt
- **WHEN** `fab pane ready <pane> --json` runs
- **THEN** stdout is exactly one JSON object with `"state": "ready"` and exit code is 0
- **AND** **GIVEN** no `--json`, **WHEN** the same verb runs, **THEN** stdout is byte-identical to the pre-change text report

### CLI: `fab pane process` Exit-Code Alignment

#### R5: `process` adopts the pane-family 2/3 scheme
`src/go/fab/cmd/fab/pane_process.go` SHALL map `pane.ValidatePane` failures through the existing `paneValidationExitCode` in-handler exit path: pane missing → **exit 2** (`Error: pane <id> not found`), any other tmux validation failure → **exit 3**. The current flat exit-1-through-RunE routing for validation failures is removed. The `map` subcommand is deliberately untouched (multi-pane discovery, no single target pane). `pane_exitcode_test.go` SHALL gain `process` rows.

- **GIVEN** no pane `%999` on a live server
- **WHEN** `fab pane process %999` runs
- **THEN** stderr reads `Error: pane %999 not found` and the exit code is 2
- **AND** **GIVEN** a dead socket, **WHEN** `fab pane process %1 -L nosuch-dead-sock` runs, **THEN** the exit code is 3

### Kit Prose & Doc Sweep

#### R6: `_cli-fab.md`, `_preamble.md`, and stale memory claims are updated in the same change
Per the constitution's CLI constraint and code-quality.md § Sibling & Mirror Sweeps:

- `src/kit/skills/_cli-fab.md` SHALL gain `kill` + `await` subcommand sections under § fab pane, document `--json` on `open`/`ready`/`deliver`, document the `server` field under § fab dispatch status, and its pane-family exit-code paragraph SHALL drop the `process` exception (process joins the 2/3 scheme; `map` remains the exit-1 case).
- `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch's pane-output bullet SHALL drop the "obtain the exact socket-included command from `fab dispatch logs` … because `status --json` does not carry the socket" workaround wording (the socket now rides `status --json`).
- The stale claims in `docs/memory/runtime/dispatch.md` (`server` "deliberately absent from the JSON surface", "status --json carries the pane but not the socket", the matching Design Decisions entry) SHALL be rewritten to current truth so no repo prose contradicts the shipped behavior; the fuller memory hydration (new verb sections, agent-primitives await reference) lands at the hydrate stage.
- No `docs/specs/skills/SPEC-*.md` mirror updates: no skill flow/tool/sub-agent structure changes (constitution v1.5.0).

- **GIVEN** the change is applied
- **WHEN** the repo is grepped for the old socket-lookup workaround claim
- **THEN** no surviving prose asserts that `status --json` lacks the socket

### Non-Goals

- `fab pane send --answer` mode — contract change, deferred to backlog `[answ]`
- A `fab dispatch await` binding — `fab dispatch wait` owns the record-keyed side
- Operator-skill / `_cli-agents.md` rewiring onto `kill`/`await` — follow-up once the verbs are proven
- `--json` on the `fab dispatch open/ready/deliver` bindings — their consumer reads the report words
- `fab pane map` exit-code changes — multi-pane discovery has no single target pane to be "missing"
- Any migration — no user-data restructuring; all additions are additive surface

### Design Decisions

#### Bundle the Five Additive Items into One Change
**Decision**: kill, await, the `server` JSON field, the three `--json` flags, and the process exit-code alignment ship together.
**Why**: They share a single mechanical sweep class (`pane-commands.md`, `dispatch.md`, `_cli-fab.md`, tests) — paying that sweep once; all five are additive, so no existing caller's behavior changes.
**Rejected**: Five separate changes (five identical sweep passes); extracting the pane/dispatch surface into a separate binary (rejected in the intake discussion — the completion signal is convention-bound and the tmux choreography is a depreciating workaround for missing provider APIs).
*Introduced by*: 260811-yxyi-pane-dispatch-surface-completion

#### Await's Control Loop Mirrors `dispatch.Wait`
**Decision**: the await tick loop lives in `internal/pane` as a pure, observer-injected control structure (`Await(ctx, observe, tick, timeout)`) with a package `AwaitTick` constant — the same shape as `internal/dispatch`'s `Wait`/`TickInterval`.
**Why**: the observer conventions (first observation before any sleep, timeout bounds the observer not the worker, observe-once-more at the bound) are already the family's documented blocking-wait contract; reusing the shape keeps the two waits incapable of drifting on semantics, and observer injection makes the loop unit-testable without tmux.
**Rejected**: a hand-rolled `for`/`sleep` poll in the cobra layer (untestable, re-derives solved edge cases); a filesystem watcher (cannot see a pane die — liveness needs a periodic probe regardless).
*Introduced by*: 260811-yxyi-pane-dispatch-surface-completion

## Tasks

### Phase 1: Setup

- [x] T001 Add the pure await control loop to `src/go/fab/internal/pane/await.go` — `AwaitReport` constants (`idle`/`file`/`running`/`gone`), `AwaitTick = 2s`, and `Await(ctx, observe, tick, timeout)` mirroring `internal/dispatch.Wait` semantics (first observation before sleep; timeout > 0 bounds and observes once more at expiry returning the still-current report with nil error; timeout <= 0 unbounded; observe error aborts; ctx cancel returns last report + ctx.Err()) <!-- R2 -->

### Phase 2: Core Implementation

- [x] T002 New `src/go/fab/cmd/fab/pane_kill.go` (`fab pane kill <pane>` — ValidatePane first, in-handler exit 2/3 via `paneValidationExitCode`, then `pane.KillPane`; success prints `killed <pane>` + `server:` line when non-default) and register it on `paneCmd` in `src/go/fab/cmd/fab/pane.go` (subcommand list + help string) <!-- R1 -->
- [x] T003 New `src/go/fab/cmd/fab/pane_await.go` (`fab pane await <pane> [--file <path>] [--timeout 300]` — ValidatePane first (2/3), immediate exit-1 error when the pane is uninstrumented AND no `--file` given, observer = pane-liveness (`gone`) + `--file` existence (`file`) + `@rk_agent_state` idle (`idle`), loop via `pane.Await` from T001, report word printed and exit 0 except `gone` → exit 2) and register it on `paneCmd` <!-- R2 -->
- [x] T004 Add `Server string \`json:"server,omitempty"\`` to `dispatchStatusJSON` in `src/go/fab/cmd/fab/dispatch_status.go`, populated from `rec.Server` in the pane branch of `observeDispatch` <!-- R3 -->
- [x] T005 [P] Align `src/go/fab/cmd/fab/pane_process.go` exit codes: route `pane.ValidatePane` failure through the in-handler `Error:` print + `os.Exit(paneValidationExitCode(err))` instead of returning it through RunE <!-- R5 -->
- [x] T006 [P] Add `--json` to `src/go/fab/cmd/fab/pane_open.go` (`{"pane","provider","server"}`), `pane_ready.go` (`{"state","pane","server","snippet"}`), and `pane_deliver.go` (`{"pane","source","path?"}`) — nullability via `toNullable`, JSON only on success, plain-text output byte-identical <!-- R4 -->

### Phase 3: Integration & Edge Cases

- [x] T007 Unit tests for the await control loop in `src/go/fab/internal/pane/await_test.go` (scripted observers: immediate fire, fire on tick, timeout returns `running` with a final re-observation, unbounded wait, observe error aborts, ctx cancel) <!-- R2 -->
- [x] T008 Command tests in `src/go/fab/cmd/fab/`: `pane_kill_test.go` (kill a live pane via private-socket tmux; missing-pane/dead-socket exit rows added to `pane_exitcode_test.go`'s helper-driven table), `pane_await_test.go` (idle-signal via `set-option -p` writer simulation, `--file` signal, unwaitable immediate error, missing-pane exit 2, `gone` mid-wait), `process` exit-2/exit-3 rows in `pane_exitcode_test.go`, JSON-shape tests for `open`/`ready`/`deliver` (stub-tmux + real-pane precedents) and for `dispatchStatusJSON.server` in `dispatch_status_test.go` <!-- R1 R2 R3 R4 R5 -->

### Phase 4: Polish

- [x] T009 Update `src/kit/skills/_cli-fab.md`: `### kill` + `### await` sections under § fab pane, `--json` on the open/ready/deliver sections, `server` field under § fab dispatch status, exit-code paragraph drops the `process` exception, and the § logs "status --json exposes pane but not server" tail updated <!-- R6 -->
- [x] T010 Update `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch pane-output bullet (drop the logs-lookup workaround), and rewrite the stale claims in `docs/memory/runtime/dispatch.md` (the "deliberately absent" requirement text, the matching Design Decisions entry, and § ready's parenthetical) to current truth <!-- R6 -->

## Execution Order

- T001 blocks T003 and T007 (the loop is their shared core)
- T002–T006 are independent of each other once T001 exists; T005/T006 marked [P]
- T008 follows the implementation tasks it covers
- T009/T010 follow all code tasks (they document the final surface)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab pane kill <pane>` kills a live pane (stdout `killed <pane>`, exit 0), exits 2 on a missing pane, 3 on other tmux failure, and writes no `.fab-dispatch/` state
- [x] A-002 R2: `fab pane await <pane>` reports `idle`/`file`/`running` (exit 0) per the signal table, `gone` (exit 2) when the pane dies mid-wait, and errors immediately (exit 1) on an uninstrumented pane with no `--file`
- [x] A-003 R3: `fab dispatch status --json` carries `"server"` for a socket-scoped pane dispatch and omits it otherwise; human output is byte-identical
- [x] A-004 R4: `fab pane open`/`ready`/`deliver --json` each emit exactly one JSON object of the specified shape on success; plain-text output is byte-identical to pre-change
- [x] A-005 R5: `fab pane process` exits 2 on a missing pane and 3 on other tmux validation failure, matching the family scheme; `map` is untouched

### Behavioral Correctness

- [x] A-006 R2: Await observes once before any sleep (an already-idle pane returns immediately), honors `--timeout` as observer-bound-only (expiry prints `running`, exit 0), and treats `--timeout 0` as unbounded
- [x] A-007 R5: `pane_exitcode_test.go` pins the new `process` rows and the pre-existing rows stay green unmodified (the coexistence of parse-time usage-error 2 with in-handler pane-missing 2 is preserved)

### Scenario Coverage

- [x] A-008 R1 R2: Integration tests run against a verified private tmux socket (the `newTmuxPane`/`tmuxSocketDir` precedent) and use the `set-option -p` writer simulation for agent-state signals
- [x] A-009 R4: A `ready --json` test covers all three classifications (`ready`/`booting`/`parked`) exiting 0 with the object shape — `TestPaneReady_JSON` covers `ready` (twice, incl. the null-server case), `parked`, and `booting` ("booting carries a blank snippet" subtest, via the existing `bootingPaneCommand` harness)

### Edge Cases & Error Handling

- [x] A-010 R2: Await on a missing pane exits 2 at entry; a dead socket exits 3; signals compose as OR (first fire wins) when both `--file` and an instrumented pane are watched
- [x] A-011 R4: `--json` failures (missing pane, refused delivery, missing prompt file) keep the existing stderr + non-zero contract and emit no JSON

### Code Quality

- [x] A-012 Pattern consistency: new verbs reuse `paneValidationExitCode`, `toNullable`, `NewGate`, and the child-re-exec / stub-tmux test patterns of the surrounding pane code
- [x] A-013 No unnecessary duplication: the await loop is the single control structure in `internal/pane`; no second tick loop in the cobra layer
- [x] A-014 Test-alongside: every `.go` change ships tests in this change, and the affected packages' test suites pass
- [x] A-015 No `.claude/skills/` edits: kit-prose changes land in `src/kit/skills/` only

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/kit/skills/fab-operator.md` raw `tmux kill-pane` removal paths and hand-rolled completion poll loops (also `_cli-agents.md`'s prose-level "artifact-oriented await") — `fab pane kill` / `fab pane await` now provide the gated binary verbs for exactly these; rewiring is a declared non-goal of this change (intake § Non-Goals) deferred to the `[answ]` follow-up, so this is informational only — do not delete here

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Await's report constants and tick live in `internal/pane` (`AwaitReport`, `AwaitTick = 2s`), not in the cobra layer | The `dispatch.TickInterval`/`Wait` precedent pins the placement; mechanics not policy | S:85 R:85 A:90 D:85 |
| 2 | Confident | `await` decides "instrumented" once at entry via the existing `AgentDisplayFromOption`-style read; a state appearing mid-wait on an initially-uninstrumented pane is not watched | Intake's error-immediately posture (assumption 4) implies the instrumented check is an entry gate; re-checking each tick would resurrect the rejected warn-and-wait shape by another name | S:65 R:75 A:70 D:60 |
| 3 | Confident | `--json` encoding uses the same indented `json.NewEncoder` form as `dispatch status --json`/`process --json` | Family precedent; exact whitespace is internal to the JSON surface (consumers parse, not diff) | S:70 R:80 A:75 D:65 |
| 4 | Certain | Memory claims in `docs/memory/runtime/dispatch.md` that contradict the shipped `server` field are corrected at apply; the fuller memory hydration (kill/await sections, agent-primitives await reference) is left to the hydrate stage | The sweep rule (code-quality.md) puts stale claims in-scope for apply; the intake's Affected Memory list is hydrate's work queue | S:80 R:80 A:80 D:75 |

4 assumptions (2 certain, 2 confident, 0 tentative).
