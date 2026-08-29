# Plan: Pane readiness gate — require agent takeover before probing; wait-timeout stall guard

**Change**: 260829-57mp-pane-readiness-agent-takeover
**Intake**: `intake.md`

## Requirements

> Source of truth for shapes and file lists: `intake.md` § What Changes (all decisions user-confirmed or clarified 2026-08-29). This section states the contracts review verifies; the intake carries the sketches.

### Runtime: the readiness gate's agent-takeover precondition

#### R1: Probe checks who owns the pane before typing anything
`Gate.Probe` MUST read the pane's foreground command before sending the sentinel. While the foreground command's basename is a known shell (the provider binary has not taken the tty), `Probe` MUST return `booting` with a capture snippet and MUST send **no keystroke at all** — neither the sentinel nor `C-u`. The echo is not consulted on that path.

- **GIVEN** a pane whose `#{pane_current_command}` is `zsh` (or `bash`, `/usr/bin/fish`, …)
- **WHEN** `Probe` runs
- **THEN** it reports `booting`
- **AND** the fake `PaneIO` recorded zero sends

- **GIVEN** a pane whose foreground command is `kimi` (any non-shell)
- **WHEN** `Probe` runs
- **THEN** the existing sentinel → echo → stability choreography runs unchanged and classifies `ready` / `booting` / `parked` exactly as today

- **GIVEN** the foreground-command read fails
- **WHEN** `Probe` runs
- **THEN** the error propagates and nothing is typed

#### R2: Delivery inherits the guard
`Deliver` MUST NOT be able to false-verify into a pre-takeover pane. Because `deliverOnce` opens with `Probe`, a shell-foreground pane fails both delivery attempts with the ordinary `pane is booting, not ready` error and the snippet attached; `Deliver`'s body is not otherwise changed.

- **GIVEN** a pane whose foreground command is a shell
- **WHEN** `Deliver` runs
- **THEN** it returns the two-attempt error carrying the snippet, and no pointer or Enter was ever sent

#### R3: `PaneIO` gains `CurrentCommand`; the tmux helper lives in `internal/pane`
`PaneIO` MUST gain `CurrentCommand(paneID string) (string, error)`. `tmuxPaneIO` MUST delegate to a new exported helper `CurrentCommand(server, paneID string) (string, error)` in `src/go/fab/internal/pane/pane.go` that runs `tmux [-L <server>] display-message -p -t <pane> '#{pane_current_command}'`, following the existing `Capture`/`SendLiteral`/`SendKey`/`ReadWindowName` conventions (server-first argument order, `RunCmd` + `StderrError`, trimmed output).

- **GIVEN** a live tmux pane running `sleep`
- **WHEN** `CurrentCommand(server, pane)` is called
- **THEN** it returns `sleep`

#### R4: One shared shell-name predicate
`shellCommands` / `isShellCommand` MUST move from `src/go/fab/cmd/fab/operator_tick_start.go` to `src/go/fab/internal/pane` as exported `IsShellCommand` (same nine basenames, case-sensitive basename match, empty string never matches). The operator's `agent_exited` predicate MUST call the relocated function. `operator_tick_diff_test.go` MUST pass byte-unchanged.

- **GIVEN** the relocated predicate
- **WHEN** `go test ./src/go/fab/cmd/fab -run 'TestOperatorTickDiff|TestOperatorTick.*AgentExited'` runs
- **THEN** every pre-existing `agent_exited` row passes unchanged

#### R5: `DeriveReadiness` is untouched
`DeriveReadiness(echoed bool, first, second string)` MUST keep its signature and table; the precondition sits in front of it.

- **GIVEN** the existing `DeriveReadiness` table tests
- **WHEN** the suite runs
- **THEN** they pass unmodified

### Tests: the "ready" integration fixture stops being a shell

#### R6: Integration fixtures use a non-shell echoing foreground
Every `newTmuxPane(t, server, "", …)` site that stands in for a **live, ready agent** in `dispatch_ready_test.go`, `pane_ready_test.go`, `dispatch_deliver_test.go`, and `pane_deliver_test.go` MUST use a non-shell command that still echoes typed text and reacts to Enter (`cat`). The `parked`/`booting` fixtures (`sleep` foreground) are unchanged. Each edited site carries a one-line comment that the fixture changed **because the gate's spec changed** (Constitution VII), not to accommodate the implementation. `pane_kill_test.go` sites are liveness-only and are verified, not necessarily changed.

- **GIVEN** the reworked fixtures
- **WHEN** `go test ./src/go/fab/cmd/fab -run 'Ready|Deliver|PaneKill'` runs with a tmux server available
- **THEN** all ready/deliver integration tests pass regardless of the host's login shell

### CLI: help text reflects the precondition

#### R7: `fab dispatch ready` and `fab pane ready` help describe the precondition
Both `Long` strings MUST state, ahead of the "purely MECHANICAL" sentence, that the probe first checks who owns the pane and types NOTHING while the foreground is a shell, and the `booting` line MUST name the shell-foreground cause. `fab pane ready`'s SIDE EFFECT paragraph MUST be narrowed accordingly (nothing is typed on the shell path). Report words, `--json` shape `{state, pane, server, snippet}`, and exit codes are unchanged.

- **GIVEN** `fab pane ready --help` and `fab dispatch ready --help`
- **WHEN** read
- **THEN** both name the foreground-command precondition and the no-keystroke property

### Kit prose: `_cli-fab.md`, `_cli-agents.md`, `_preamble.md` stall guard

#### R8: `_cli-fab.md` restates the new contract in its two owner sections
§ `fab pane ready` and § `fab dispatch ready` (incl. the three-row classification table's `booting` row) MUST carry the foreground-command precondition and the types-nothing-while-a-shell property. (Constitution: CLI behaviour change ⇒ `_cli-fab.md` + tests.)

- **GIVEN** `grep -n "purely echo-based\|purely MECHANICAL" src/kit/skills/_cli-fab.md`
- **WHEN** read after the edit
- **THEN** every hit is qualified by the precondition

#### R9: `_cli-agents.md` — parenthetical + the exec contract (owner)
The `fab pane ready <pane>` parenthetical MUST mention the precondition. `_cli-agents.md` MUST state once — as the owner — the **exec contract**: a provider's `interactive_command` MUST exec its binary; a wrapper that keeps a shell in the foreground is unsupported by design and fails observably (`booting` → promoted `parked` with the shell prompt in the snippet → escalation), never silently. No time bound, no `spawn_cmd` comparison.

- **GIVEN** a user reading the provider-authoring guidance
- **WHEN** they define a provider via a shell wrapper
- **THEN** the documented contract tells them why the gate will never read `ready`

#### R10: `_preamble.md` § The pane readiness gate owns the stall guard
`_preamble.md` § The pane readiness gate MUST gain (a) the **stall guard**: when a `fab dispatch wait` round returns `running` (bound expired) AND there is no `{stage}-result.yaml` AND the `fab pane capture` screen is unchanged against the previous round, the orchestrator MUST judge the captured screen before re-arming — a first-run wall or bare shell prompt means the delivery never happened and the judgment rounds re-enter; otherwise re-arm. It is capture-based, read-only, and MUST NOT re-run `fab dispatch ready` (which refuses `Delivered && !ResultPresent`) or `fab pane ready` (a sender into a possibly-live worker). (b) The closing "From successful delivery onward the ordinary rule applies again" sentence MUST be qualified: a delivery that never reached a worker leaves no stage context, so the judgment rounds stay legal. The numbered CLI-Adapter Dispatch procedure's step 2 (`wait`) and the timeout-return peek table MUST **point at** the owner section (a fourth bucket row or pointer line), not restate it.

- **GIVEN** an orchestrator whose `wait` round times out with no result file and an unchanged screen
- **WHEN** it follows `_preamble.md`
- **THEN** it peeks with `fab pane capture`, judges the screen, and either re-enters the judgment rounds or re-arms — never types a probe into the worker

#### R11: Sibling sweep
`docs/specs/harness-adapters.md` (verb-table row + the "gate is MECHANICAL … ⇒ `booting`" passage) MUST be updated. `fab-fff.md`, `fab-ff.md`, `fab-operator.md`, `fab-continue.md`, `fab-adopt.md`, `_pipeline.md` were grepped at intake with zero restatements — re-verify at apply (`grep -rn "readiness gate\|dispatch ready\|echo-based\|purely MECHANICAL" src/kit/skills docs/specs`). Memory files are hydrate's, not apply's.

- **GIVEN** the sweep grep
- **WHEN** run after apply
- **THEN** no hit describes the gate as echo-only without the precondition

### Non-Goals

- Reading `@rk_pane_agent_state` in the gate — explicitly rejected (decision 2); `runtime/dispatch.md`'s "never from `@rk_pane_agent_state`" stays true.
- A table of known dialog texts, any new config field, flag, report word, or `--json` field — none.
- Time-bounding the precondition or comparing against `spawn_cmd` — rejected in favour of the exec contract.
- Raising the `_preamble.md` 5-consecutive-`booting` allowance — stays at 5.
- Memory edits — hydrate's job.

### Design Decisions

#### The precondition sits in front of DeriveReadiness, not inside it
**Decision**: Check `IsShellCommand(CurrentCommand(pane))` in `Probe` before the sentinel; `DeriveReadiness` keeps its `(echoed, first, second)` signature.
**Why**: Two unrelated axes — who owns the pane vs. what the screen shows — should not share one classifier table; the package's pure-classifier precedent is one decision per function.
**Rejected**: A fourth `DeriveReadiness` parameter — spreads the ownership question across every existing table row.
*Introduced by*: 260829-57mp-pane-readiness-agent-takeover

#### The stall guard is capture-based judgment, not a probe re-run
**Decision**: On a no-progress `wait` timeout the orchestrator judges `fab pane capture` output; no readiness verb is re-run.
**Why**: `fab dispatch ready` refuses exactly `Delivered && !ResultPresent` (the stall state), and `fab pane ready` would type a sentinel into a possibly-live worker — violating "the pipeline NEVER sends keys to a WORKER". With R1 in place a bare shell prompt on screen is conclusive.
**Rejected**: Carving out the `dispatch ready` refusal (weakens a guard that exists for a reason); naming `fab pane ready` (reintroduces mid-stage typing).
*Introduced by*: 260829-57mp-pane-readiness-agent-takeover

#### Exec contract for providers
**Decision**: A provider's `interactive_command` must exec its binary; documented once in `_cli-agents.md`.
**Why**: All four built-ins already do; a wrapper provider fails observably (parked + shell-prompt snippet + escalation), so no escape hatch is needed and no new field is added.
**Rejected**: Time-bounding the precondition (reopens the cooked-tty false-ready window after N seconds); comparing against `spawn_cmd` (brittle across `bash -c` argv shapes).
*Introduced by*: 260829-57mp-pane-readiness-agent-takeover

## Tasks

### Phase 1: Setup

- [x] T001 Add `CurrentCommand(server, paneID string) (string, error)` to `src/go/fab/internal/pane/pane.go` (tmux `display-message -p -t <pane> '#{pane_current_command}'`, server-first, `RunCmd` + `StderrError`, trimmed) with a pure argv/result test in `src/go/fab/internal/pane/pane_test.go` <!-- R3 -->
- [x] T002 [P] Move `shellCommands` + `isShellCommand` from `src/go/fab/cmd/fab/operator_tick_start.go` into `src/go/fab/internal/pane` as exported `IsShellCommand` (byte-identical semantics), repoint the `agent_exited` call site, add `IsShellCommand` table tests in `pane_test.go`; run `go test ./src/go/fab/cmd/fab -run OperatorTick` to prove `operator_tick_diff_test.go` passes unchanged <!-- R4 -->

### Phase 2: Core Implementation

- [x] T003 In `src/go/fab/internal/pane/gate.go`: add `CurrentCommand` to `PaneIO`, implement `tmuxPaneIO.CurrentCommand`, and add the agent-takeover precondition at the top of `Probe` (shell foreground ⇒ `ReadyBooting` + `Snippet(capture)`, zero sends; error ⇒ propagate); rewrite the "Why the gate is mechanical" doc block to name the cooked-tty false-echo cause. `DeriveReadiness` untouched <!-- R1 -->
- [x] T004 In `src/go/fab/internal/pane/gate_test.go`: extend `fakePaneIO` with a scripted `CurrentCommand` value + `failOn["command"]` arm (existing tests default to a non-shell command so they keep passing); add the precondition table (shell names ⇒ `booting` with `len(io.sends)==0`; `kimi`/`claude`/`node` + echo ⇒ `ready`; `kimi` + no-echo rows unchanged; `CurrentCommand` error ⇒ error, nothing typed) and a `Deliver` row proving a shell-foreground pane fails both attempts with the snippet and no pointer/Enter sent <!-- R1 R2 R5 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Replace the bare-shell "ready" fixture (`newTmuxPane(t, server, "", …)`) with a `cat` foreground at every live-agent stand-in site in `src/go/fab/cmd/fab/dispatch_ready_test.go`, `pane_ready_test.go`, `dispatch_deliver_test.go`, `pane_deliver_test.go` (prefer a named helper/default so the intent is stated once); each edited site (or the helper) carries a Constitution VII note that the spec changed; verify `pane_kill_test.go` sites are liveness-only; run `go test ./src/go/fab/cmd/fab -run 'Ready|Deliver|PaneKill'` and `go test ./src/go/fab/internal/pane/...` <!-- R6 -->
- [x] T006 [P] Update `Long` help text in `src/go/fab/cmd/fab/dispatch_ready.go` and `src/go/fab/cmd/fab/pane_ready.go` (precondition before the "purely MECHANICAL" sentence; `booting` line gains the shell-foreground cause; `pane ready` SIDE EFFECT paragraph narrowed); adjust any help-text tests <!-- R7 -->

### Phase 4: Polish

- [x] T007 [P] Update `src/kit/skills/_cli-fab.md` § `fab pane ready` and § `fab dispatch ready` (incl. the classification table's `booting` row) with the precondition and types-nothing property; `--json`/exit codes noted unchanged <!-- R8 -->
- [x] T008 [P] Update `src/kit/skills/_cli-agents.md`: the `fab pane ready <pane>` parenthetical, plus the exec-contract sentence in the provider-authoring guidance (single owner statement) <!-- R9 -->
- [x] T009 Edit `src/kit/skills/_preamble.md` § The pane readiness gate: add the capture-based stall guard (a), qualify the "From successful delivery onward" sentence (b); add a pointer row/line in the numbered procedure's step 2 (`wait`) and the timeout-return peek table — pointers only, no restatement <!-- R10 -->
- [x] T010 [P] Update `docs/specs/harness-adapters.md` (verb-table row + "gate is MECHANICAL … ⇒ `booting`" passage); run the sibling-sweep grep across `src/kit/skills` and `docs/specs` and fix any remaining echo-only description <!-- R11 -->
- [x] T011 Run `fab sync` (deployed copies), `gofmt -l src/go`, `go vet ./src/go/...`, and the full `go test ./src/go/...`; fix anything that trips <!-- R1 R6 -->

## Execution Order

- T001, T002 are independent and precede T003
- T003 blocks T004 (fake needs the new interface method) and T005 (fixtures fail only once the precondition lands)
- T006–T010 are independent of each other; T009 depends on nothing but should follow T007/T008 so pointers name final wording
- T011 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `Probe` returns `booting` and records zero sends when the foreground command is a shell; the sentinel path is unchanged for a non-shell foreground
- [x] A-002 R2: `Deliver` against a shell-foreground pane returns the two-attempt error with snippet and never sends the pointer or Enter
- [x] A-003 R3: `PaneIO.CurrentCommand` exists; `pane.CurrentCommand(server, pane)` runs `display-message -p -t <pane> '#{pane_current_command}'` via `RunCmd`/`StderrError`
- [x] A-004 R4: `IsShellCommand` is exported from `internal/pane`; `cmd/fab` no longer defines its own copy; `operator_tick_diff_test.go` passes unchanged
- [x] A-005 R6: No `newTmuxPane(t, server, "", …)` remains as a live-agent stand-in in the four ready/deliver test files; the `cat` fixture is used and each edit is annotated as spec-driven (Constitution VII)
- [x] A-006 R7: Both `Long` help strings describe the precondition and the no-keystroke property; `pane ready`'s SIDE EFFECT paragraph no longer claims an unconditional type
- [x] A-007 R8: `_cli-fab.md`'s two sections carry the precondition; the classification table's `booting` row names the shell-foreground cause
- [x] A-008 R9: `_cli-agents.md` states the exec contract once and the parenthetical is updated
- [x] A-009 R10: `_preamble.md` § The pane readiness gate owns the stall guard (capture-based, no probe re-run) and the qualified carve-out; step 2 and the peek table point at it

### Behavioral Correctness

- [x] A-010 R1: A pre-takeover pane (shell foreground, cooked tty) can no longer classify `ready` — verified by the gate unit table
- [x] A-011 R5: `DeriveReadiness`'s signature and its existing table tests are unmodified
- [x] A-012 R10: The stall guard text does not instruct re-running `fab dispatch ready` or `fab pane ready`

### Scenario Coverage

- [x] A-013 R1: Gate table rows cover `zsh`, `bash`, `/usr/bin/fish` (booting, no sends), `kimi`/`claude`/`node` + echo (ready), `kimi` no-echo changed/stable/blank (booting/parked/booting), and `CurrentCommand` error
- [x] A-014 R6: `go test ./src/go/fab/cmd/fab -run 'Ready|Deliver'` (from `src/go/fab`) passes on the host regardless of login shell

### Edge Cases & Error Handling

- [x] A-015 R1: A `CurrentCommand` error propagates from `Probe` with nothing typed
- [x] A-016 R6: `parked`/`booting` integration fixtures (`sleep` foreground) still classify as before

### Code Quality

- [x] A-017 Pattern consistency: `CurrentCommand` follows the `Capture`/`SendLiteral`/`ReadWindowName` helper shape; test fakes follow the existing `captures`/`failOn` scripting style
- [x] A-018 No unnecessary duplication: exactly one shell-name predicate exists in the Go tree
- [x] A-019 Canonical source only: kit edits are under `src/kit/skills/`, none under `.claude/skills/`
- [x] A-020 CLI ⇒ docs + tests: `_cli-fab.md` updated and Go tests accompany every `.go` change
- [x] A-021 Owner-or-pointer: the stall guard is stated once (`_preamble.md` gate section) and pointed at elsewhere; the exec contract is stated once (`_cli-agents.md`)
- [x] A-022 No magic strings: report words remain the `Readiness` constants; no inline `"ready"`/`"booting"` literals added

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Constitution VII: the integration-fixture edits (T005) are spec-driven — the definition of "ready" changed — never implementation-accommodating.

## Deletion Candidates

- None — this change adds the takeover precondition and relocates the shell predicate; the old `shellCommands`/`isShellCommand` copy in `cmd/fab/operator_tick_start.go` was already removed by the planned R4 move itself, leaving no discovered redundancy.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The `cat` fixture is introduced through a named helper/default in the test package rather than 13 literal edits | Intake §5 leaves this as an apply-time detail; a named helper states the Constitution VII rationale once | S:70 R:90 A:85 D:75 |
| 2 | Confident | `Probe`'s shell-foreground path returns `Snippet(capture)` so the orchestrator sees the shell prompt in a `booting` report | Every non-ready report already carries a snippet; a blank one would read as truncated | S:75 R:90 A:85 D:80 |
| 3 | Confident | `fakePaneIO` defaults `CurrentCommand` to a non-shell value so pre-existing gate tests need no per-test edit | Keeps the diff to the new rows; existing rows test the post-takeover path by definition | S:70 R:90 A:85 D:75 |
| 4 | Confident | `readyPaneCommand` is `echo READY-PANE; exec cat`, not bare `cat`, and lives as a named constant in `dispatch_ready_test.go` (shared across the `cmd/fab` test package) rather than a helper | Recorded at apply: bare `cat` draws a blank screen until typed at, which broke the four refusal tests' settled-screen baseline diff (`settledPane`/`settledCapture` require a non-blank settled screen); the banner echo fixes that while `exec` keeps `cat` — a non-shell basename — as the pane's foreground command. A constant beats a helper because the "ready" sites need no per-site parameterization | S:80 R:85 A:85 D:70 |

4 assumptions (0 certain, 4 confident, 0 tentative).
