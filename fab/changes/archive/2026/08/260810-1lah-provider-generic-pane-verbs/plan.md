# Plan: Provider-Generic Pane Verbs — Extract open/ready/deliver Primitives from Dispatch

**Change**: 260810-1lah-provider-generic-pane-verbs
**Intake**: `intake.md`

## Requirements

### Runtime: Pane-Addressed Primitives (`fab pane`)

#### R1: `fab pane open` — provider-generic pane spawn
`fab pane open --provider <name> [--role <role>] [-c <dir>] [--server <name>]` SHALL resolve the named provider's `interactive_command` (project config per-field merged over the built-in table, exactly as `fab agent` resolves it), substitute `{model}`/`{effort}` via the standard fill precedence with the provider pinned at invocation time (`agent.ResolveRoleWith` with the provider override — role defaulting to `default` when `--role` is omitted), spawn the command in a tmux pane, and print the new pane id (plus the socket label when `--server` is non-default). It SHALL write no dispatch record and no `.fab-dispatch/` state. The spawn SHALL be a plain split of the current window when the invoker is a tmux pane on the target server (`$TMUX_PANE` set, no `--server`), and a new (unnamed) window otherwise — the worker-column carving/placement logic stays dispatch-owned. An unknown provider SHALL be a lookup failure naming the available providers (the shared `unknownProviderError`); a provider with no `interactive_command` SHALL be a hard error naming the provider (explicit-pane posture: no descent); an unreachable tmux server SHALL be a hard error before anything is spawned. Provider resolution SHALL work outside a fab repo (empty config → the built-in provider table applies), matching the pane family's no-`FabRoot`-guard posture.

- **GIVEN** a built-in provider with an `interactive_command` (e.g. kimi) and a reachable tmux server
- **WHEN** `fab pane open --provider kimi` runs inside a tmux pane
- **THEN** the composed interactive command runs in a plain split of the current window and stdout prints the new pane id
- **AND** no `.fab-dispatch/` entry is created

- **GIVEN** a provider with no `interactive_command` (e.g. agy)
- **WHEN** `fab pane open --provider agy` runs
- **THEN** the command exits non-zero naming the provider and the missing `interactive_command`, spawning nothing

#### R2: `fab pane ready` — pane-addressed readiness probe
`fab pane ready <pane>` SHALL run the relocated gate classifier against the named pane id: type the sentinel literally, check the wrap-tolerant echo, clear with C-u, and report `ready` / `booting` / `parked`. Non-ready reports SHALL carry the pane id, the socket label (when non-default), and a trailing capture snippet, mirroring `fab dispatch ready`'s report form. All three classifications SHALL exit 0 — the report string is the sole discriminator. The pane SHALL be validated first (`pane.ValidatePane`): a missing pane exits 2, any other tmux failure exits 3 (the pane-family scheme). The help text SHALL document that the probe types into the target pane (cleared before return) and is not run against panes the caller does not own.

- **GIVEN** a live pane sitting at an idle shell prompt
- **WHEN** `fab pane ready %N` runs
- **THEN** stdout is exactly `ready\n` and the exit code is 0

- **GIVEN** a pane parked behind a stable wall screen
- **WHEN** `fab pane ready %N` runs
- **THEN** stdout reports `parked` plus `pane:` and the capture snippet, exiting 0

#### R3: `fab pane deliver` — pane-addressed verified delivery
`fab pane deliver <pane> (--prompt-file <path> | --text <string>)` SHALL run the verified typed-delivery choreography against the named pane: readiness probe → C-u → type → wrap-tolerant echo-verify → Enter → confirm the screen advanced, with exactly one retry. `--prompt-file` SHALL type the pointer line (`Read <path> and execute it.` — dispatch parity, via the shared pointer composer) after verifying the file exists; `--text` SHALL type the literal text. The two flags SHALL be mutually exclusive and exactly one SHALL be required (usage error otherwise). Retry warnings SHALL go to stderr; a second failed attempt SHALL report the trailing pane snippet on stderr and exit 1. Pane validation failures follow the 2/3 scheme.

- **GIVEN** a ready pane
- **WHEN** `fab pane deliver %N --text "echo hi"` runs
- **THEN** the text is typed, echo-verified, submitted, the screen advance is confirmed, and stdout reports the delivery

- **GIVEN** a missing prompt file
- **WHEN** `fab pane deliver %N --prompt-file nope.md` runs
- **THEN** the command exits 1 naming the missing file, typing nothing

### Runtime: Relocation & Dispatch Bindings

#### R4: Gate/delivery/tmux mechanics relocated to `internal/pane`
The pane-addressed machinery SHALL move from `internal/dispatch` into `internal/pane` (no import cycle: `internal/pane` imports `internal/resolve`/`internal/status`/`internal/statusfile` only, and the gate needs nothing from `internal/dispatch`): the whole gate (`Gate`, `NewGate`, `Probe`, `Deliver`, `DeriveReadiness`, the `Readiness` constants, `ReadySentinel`, `SnippetLines`, `Snippet`, `PaneIO`, `newlyEchoed`, `countWrapped`, `squeeze`, timings, retry count), the tmux pane creators (`OpenWindow`, `OpenSplitPane`, `SplitPlacement` + `Describe`, `splitArgs` + split-flag constants, `runPaneCreator`), `ServerReachable`, `PaneAlive`, `KillPane`, `Tail`, and the pointer-line composer `PointerPrompt`. Placement *policy* — `SelectMode`, `SelectPaneShape`, `SplitTarget`, `SiblingDispatchPane`, `recordedPanes` — SHALL stay in `internal/dispatch` (`SplitTarget` then returns `pane.SplitPlacement`). All behavior SHALL be byte-identical: this is a relocation, not a behavior change.

- **GIVEN** the relocated packages
- **WHEN** `go build ./... && go test ./internal/pane/... ./internal/dispatch/... ./cmd/fab/...` run
- **THEN** everything compiles and the moved tests pass under their new package with no behavior edits

#### R5: Dispatch open/ready/deliver as thin bindings with a byte-identical contract
`fab dispatch open`, `fab dispatch ready`, and `fab dispatch deliver` SHALL keep their exact external contract — flags, output forms (`opened <id>/<stage> (...)`, `delivered <id>/<stage> (pane %N, prompt <path>)`, readiness reports with pane/server/snippet), and exit behavior — while delegating the mechanics to `internal/pane` and retaining only dispatch record bookkeeping (record load/save, refuse-if-running, mid-stage guard, stash/restore of completion signals, result-file clearing). Existing dispatch tests SHALL pass unmodified except import/package moves (`dispatch.PaneAlive` → `pane.PaneAlive`, `dispatch.ReadyReady` → `pane.ReadyReady`, and the like).

- **GIVEN** the pre-change dispatch test suite
- **WHEN** the relocation lands
- **THEN** every dispatch test passes with no edits beyond symbol-package updates

### Runtime: `fab pane send` Foreign-Pane Posture

#### R6: Unknown agent state warns and proceeds
`fab pane send` SHALL stop refusing on an unknown `@rk_agent_state` (option absent or unparseable): it SHALL print a stderr warning (`agent state unknown — sending anyway`) and send. Refusal SHALL remain only for a *parseable* non-idle state (`active`/`waiting` → the three-state-aware refusal). `--force` SHALL retain its skip-everything meaning (idle validation skipped entirely; pane existence still enforced, exit 2/3 intact).

- **GIVEN** a pane with no `@rk_agent_state` option (a foreign-agent pane)
- **WHEN** `fab pane send %N "hi"` runs without `--force`
- **THEN** the warning is printed on stderr and the keys are sent (exit 0)

- **GIVEN** a pane whose `@rk_agent_state` parses to `active`
- **WHEN** `fab pane send %N "hi"` runs without `--force`
- **THEN** the send is refused with `agent in pane %N is not idle (state: active)` (exit 1)

### Runtime: Exit Codes

#### R7: The pane-family 2/3 scheme extends to the new verbs
Across `fab pane open` / `ready` / `deliver`: pane missing SHALL exit 2, other tmux failures (dead server, bad socket, failed spawn/probe) SHALL exit 3 — classification riding the typed `*pane.PaneNotFoundError`, no string matching. Verification failures (`deliver` echo/submit checks exhausted) and non-tmux operational errors (unknown provider, missing `interactive_command`, missing prompt file, flag misuse) SHALL exit 1 through RunE, leaving the binary-wide usage-error exit 2 coexistence documented in pane-commands memory undisturbed.

- **GIVEN** a dead tmux socket named via `--server`
- **WHEN** `fab pane ready %N --server nosuch` runs
- **THEN** the exit code is 3

### Documentation: Skills, SPEC Mirrors, Specs

#### R8: CLI reference, agent primitives, and adapter spec record the layering
`src/kit/skills/_cli-fab.md` SHALL document the three new § fab pane verbs (signatures, flags, exit codes, the ready probe's side-effect caveat), `pane send`'s new unknown-state posture, and a dispatch § note that `open`/`ready`/`deliver` are thin record-keeping bindings over the pane primitives. `src/kit/skills/_cli-agents.md` SHALL rewrite the probe recipes (rpsr pre-ship probe, first-run wall discovery) onto the 3-command flow `fab pane open --provider X` → `fab pane ready` → `fab pane deliver --text`, keeping the printed-prompt-trap prose honest about what the binary now mechanizes. `docs/specs/harness-adapters.md` SHALL record the primitives-vs-bindings layering. Every touched `src/kit/skills/*.md` SHALL carry its `docs/specs/skills/SPEC-*.md` mirror update in this change (the whole mirror class, per code-quality.md § Sibling & Mirror Sweeps). `src/kit/skills/_preamble.md` SHALL be touched only if its pane-readiness-gate / CLI-Adapter-Dispatch prose needs a pointer at the primitives — owner-or-pointer, no owned-rule duplication. `.claude/skills/` SHALL NOT be edited (canonical sources only).

- **GIVEN** the change's skill edits
- **WHEN** reviewing `docs/specs/skills/SPEC-_cli-fab.md`, `SPEC-_cli-agents.md`, and `SPEC-_preamble.md` (if the skill changed)
- **THEN** each mirror restates the new contract accurately and no stale claim about the old send refusal or dispatch-locked gate survives repo-wide

### Non-Goals

- Worker-column placement for `fab pane open` — placement is pipeline policy and stays dispatch-owned; the primitive does a plain split.
- Changing the dispatch record schema, the five-state machine, or any dispatch output string — the binding contract is byte-identical.
- Renumbering or unifying the pane-family exit codes with the binary-wide usage-error 2 — the documented coexistence stands.
- Memory updates (`docs/memory/`) — hydrate's stage, not apply's.

### Design Decisions

#### Primitives in `internal/pane`, Policy in `internal/dispatch`
**Decision**: The gate, the verified-delivery choreography, the tmux pane creators/liveness/kill helpers, `Tail`, and `PointerPrompt` relocate to `internal/pane`; mode/shape/placement *decisions* and all record bookkeeping stay in `internal/dispatch`.
**Why**: The new verbs need the mechanics addressed by pane id with no dispatch record; the import graph allows it cleanly (`internal/pane` has no dispatch dependency, and the gate needs none), so the sibling-package fallback the intake allowed is unnecessary. One home for tmux mechanics means echo-verify cannot grow a divergent second copy.
**Rejected**: Ad-hoc `--no-record`/`--pane-id` flags on the dispatch verbs (muddies the five-state contract — rejected at intake); a new sibling package (unneeded — no import cycle); moving `SplitTarget`/`SelectPaneShape` (they read dispatch records and config — pipeline policy).
*Introduced by*: 260810-1lah-provider-generic-pane-verbs

#### `fab pane open` Resolves Fills with the Provider Pinned, Not the `fab agent --provider` Bypass
**Decision**: `--provider` pins the provider at the top of the standard fill precedence (`agent.ResolveRoleWith` with the provider override set); `--role` selects whose fills apply, defaulting to the `default` role.
**Why**: The intake specifies "role/profile fills via the standard precedence; `--role` optional, default role's fill otherwise" — the opposite of `fab agent --provider`'s deliberate fill bypass. A probe spawn wants the same resolved command a pipeline worker would get.
**Rejected**: Reusing `fab agent --provider`'s bypass semantics (contradicts the intake; composes a profile-free invocation the probe then has to hand-tune).
*Introduced by*: 260810-1lah-provider-generic-pane-verbs

## Tasks

### Phase 1: Relocation to `internal/pane`

- [x] T001 Move `src/go/fab/internal/dispatch/gate.go` → `src/go/fab/internal/pane/gate.go` (package `pane`; drop the now-local `pane.` qualifiers; update the file-header comment's framing from dispatch to the shared pane layer) and `gate_test.go` → `src/go/fab/internal/pane/gate_test.go` with the same package move. Move `Tail` from `src/go/fab/internal/dispatch/dispatch.go` into `internal/pane` (gate's `Snippet` uses it; `cmd/fab/dispatch_logs.go:55` switches to `pane.Tail`; move its tests to `internal/pane/pane_test.go` or the new gate test file as fits) <!-- R4 -->
- [x] T002 Move the tmux mechanics out of `src/go/fab/internal/dispatch/pane_mode.go` into a new `src/go/fab/internal/pane/create.go`: `OpenWindow`, `OpenSplitPane`, `runPaneCreator`, `SplitPlacement` (+`Describe`), `splitArgs`, the split-flag constants, `ServerReachable`, `PaneAlive`, `KillPane`. Move `PointerPrompt` from `dispatch.go` there as well. Keep `SelectMode`, `SelectPaneShape`, `SplitTarget`, `splitPlacement`, `SiblingDispatchPane`, `recordedPanes` in `internal/dispatch` (`SplitTarget`/`splitPlacement` now produce `pane.SplitPlacement`); rewrite `pane_mode.go`'s header comment to describe the remaining policy half. Update dispatch-side callers: `cmd/fab/dispatch_start.go` (`OpenSplitPane`/`OpenWindow`/`ServerReachable`/`SplitTarget` sites), `cmd/fab/dispatch_ready.go`, `cmd/fab/dispatch_deliver.go` (`NewGate`, `PointerPrompt`, `SnippetLines`, `ReadyReady`), `cmd/fab/dispatch_reap.go`, `cmd/fab/dispatch_status.go`, `cmd/fab/dispatch_kill.go` <!-- R4, R5 -->
- [x] T003 Update the test files for the package moves and run the scoped suites: `cd src/go/fab && go build ./... && go test ./internal/pane/... ./internal/dispatch/... ./cmd/fab/...` — dispatch tests green with import/package-move edits only (`dispatch.PaneAlive`→`pane.PaneAlive`, `dispatch.ReadyReady`→`pane.ReadyReady`, `dispatch.SnippetLines`→`pane.SnippetLines` in `dispatch_open_test.go`, `dispatch_ready_test.go`, `dispatch_deliver_test.go`, `dispatch_reap_test.go`, `dispatch_kill_test.go`, `dispatch_restart_test.go`; relocation of moved-function unit tests from `internal/dispatch/dispatch_test.go` to the matching `internal/pane` test files) <!-- R4, R5 -->

### Phase 2: `fab pane send` Posture Fix

- [x] T004 In `src/go/fab/cmd/fab/pane_send.go` change `idleGate`'s unknown case from refusal to warn-and-proceed: return a warning (printed by the caller as `warning: agent state unknown — sending anyway` on stderr) instead of an error, keeping the parseable non-idle refusal and `--force` semantics; update `src/go/fab/cmd/fab/pane_send_test.go` (the five-case table + the pinned message contracts) to the new posture and run `go test ./cmd/fab/ -run 'PaneSend'` <!-- R6 -->

### Phase 3: New `fab pane` Verbs

- [x] T005 Add the plain-spawn creator to `src/go/fab/internal/pane/create.go` (reusing `runPaneCreator`): a plain `tmux split-window -P -F '#{pane_id}' -c <dir> <cmd>` when splitting, an unnamed `tmux new-window` otherwise — no size, no title, no placement logic. Unit-test the argv shapes in `internal/pane` <!-- R1 -->
- [x] T006 Add `src/go/fab/cmd/fab/pane_open.go`: `fab pane open --provider <name> [--role <role>] [-c <dir>]` (inherits the parent's persistent `--server`). Resolution: config via fab-root when resolvable, empty config otherwise (built-in table); `agent.ResolveRoleWith(cfg, role, Overrides{Provider: name, ProviderSet: true})` for fills; `agent.ResolveProvider` for the command; unknown provider → shared `unknownProviderError`; missing `interactive_command` → hard error naming the provider; `pane.ServerReachable` probe before spawn (exit 3 on tmux failure); split when `$TMUX_PANE` set and no `--server`, else unnamed window via T005's creator; print the pane id and, when non-default, the socket. Tests in `src/go/fab/cmd/fab/pane_open_test.go` <!-- R1, R7 -->
- [x] T007 Add `src/go/fab/cmd/fab/pane_ready.go`: `fab pane ready <pane>` — `pane.ValidatePane` (exit 2/3 via `paneValidationExitCode`), then `pane.NewGate(server).Probe(pane)`; print the classification; for non-`ready` print `pane:`, `server:` (when set), and the snippet under the `--- last N lines ---` header (dispatch-ready report parity); probe tmux failures exit 3; classifications all exit 0. Long help documents the typed-sentinel side effect. Tests in `src/go/fab/cmd/fab/pane_ready_test.go` (mirror `dispatch_ready_test.go`'s live-tmux table where the harness allows) <!-- R2, R7 -->
- [x] T008 Add `src/go/fab/cmd/fab/pane_deliver.go`: `fab pane deliver <pane> (--prompt-file <path> | --text <string>)` — exactly-one-of usage guard; `pane.ValidatePane` (2/3); `--prompt-file` existence check then `pane.PointerPrompt(path)` as typed payload, `--text` literal payload; `pane.NewGate(server).Deliver`; warnings to stderr, failure snippet to stderr, exit 1 on unverified delivery; success prints a `delivered` line naming the pane and payload source. Tests in `src/go/fab/cmd/fab/pane_deliver_test.go` <!-- R3, R7 -->
- [x] T009 Wire the three verbs into `paneCmd()` in `src/go/fab/cmd/fab/pane.go` (and its Long listing), then run the full affected suites: `cd src/go/fab && go test ./internal/pane/... ./internal/dispatch/... ./cmd/fab/...` (watch `helpdump_test.go` / `fabhelp_test.go` / `examples_test.go` / `pane_exitcode_test.go` for command-enumeration breakage and update them to the new surface). Finish with `gofmt -l src/go` clean <!-- R1, R2, R3, R7 -->

### Phase 4: Documentation & Mirrors

- [x] T010 [P] Update `src/kit/skills/_cli-fab.md`: § fab pane gains `open` / `ready` / `deliver` entries (signatures, flags, exit codes, ready's typed-probe caveat); the `send` entry's unknown-state paragraph flips to warn-and-proceed; § fab dispatch gains a layering note (open/ready/deliver are thin record-keeping bindings over the pane primitives). Mirror the changes in `docs/specs/skills/SPEC-_cli-fab.md` <!-- R8 -->
- [x] T011 [P] Update `src/kit/skills/_cli-agents.md`: rewrite the probe recipes (the rpsr pre-ship probe, first-run wall discovery) onto `fab pane open --provider X` → `fab pane ready` → `fab pane deliver --text`; adjust § Pre-Send Validation for send's new unknown posture. Mirror in `docs/specs/skills/SPEC-_cli-agents.md` <!-- R8 -->
- [x] T012 [P] Record the primitives-vs-bindings layering in `docs/specs/harness-adapters.md` (§ 3 interactive-pane adapter); check `src/kit/skills/_preamble.md` § pane readiness gate / CLI-Adapter Dispatch — touch only if a pointer at the primitives is needed (owner-or-pointer, no owned-rule duplication), and if touched update `docs/specs/skills/SPEC-_preamble.md`. Sweep the mirror class: grep repo-wide for the old claims (unknown-state refusal text, gate-lives-in-dispatch framing) and update every occurrence in `docs/specs/` and `src/kit/skills/` <!-- R8 -->

## Execution Order

- T001 → T002 → T003 are strictly sequential (one relocation, verified green before anything new is built on it)
- T004 is independent of Phase 1 and may run alongside it
- T005 → T006; T007 and T008 depend on T001–T002 (the relocated gate) but not on T005/T006
- T009 caps Phase 3
- Phase 4 (T010–T012) depends on the final command surface (T006–T009) for exact signatures

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab pane open --provider <name>` spawns the resolved interactive command in a plain split (or unnamed window fallback), prints the pane id (+ socket when non-default), and writes no `.fab-dispatch/` state
- [x] A-002 R2: `fab pane ready <pane>` reports `ready`/`booting`/`parked` with pane, socket, and snippet on non-ready, all classifications exiting 0
- [x] A-003 R3: `fab pane deliver <pane> --prompt-file|--text` performs the probe→clear→type→echo-verify→Enter→confirm choreography with one retry; `--prompt-file` types the dispatch-parity pointer line
- [x] A-004 R4: Gate, creators, liveness/kill, `Tail`, and `PointerPrompt` live in `internal/pane`; placement policy and record bookkeeping stay in `internal/dispatch`
- [x] A-005 R5: `fab dispatch open/ready/deliver` delegate to `internal/pane` with byte-identical flags, output strings, and exit behavior
- [x] A-006 R6: `fab pane send` warns (`agent state unknown — sending anyway`) and sends on unknown state; parseable non-idle still refuses; `--force` unchanged
- [x] A-007 R7: The three new verbs exit 2 on missing pane and 3 on other tmux failures via the typed error classification
- [x] A-008 R8: `_cli-fab.md`, `_cli-agents.md`, `harness-adapters.md`, and every touched skill's SPEC mirror reflect the new verbs, the send posture, and the primitives/bindings layering

### Behavioral Correctness

- [x] A-009 R5: The pre-existing dispatch test suite passes with import/package-move edits only — no expectation strings changed
- [x] A-010 R6: `pane_send_test.go`'s updated table proves unknown→send+warning, idle→send, active/waiting→refusal, `--force`→skip

### Scenario Coverage

- [x] A-011 R1: Unknown provider and missing-`interactive_command` hard errors are covered by cmd-level tests
- [x] A-012 R2,R3: Live-tmux tests (or scripted-IO gate tests) cover ready's three classifications and deliver's retry/failure paths for the new verbs
- [x] A-013 R7: Exit-code tests cover 2 (missing pane) and 3 (dead socket) for the new verbs, extending `pane_exitcode_test.go`'s scheme

### Edge Cases & Error Handling

- [x] A-014 R1: `fab pane open` outside a fab repo resolves built-in providers (empty config) and outside tmux falls back to an unnamed window
- [x] A-015 R3: Missing `--prompt-file` target exits 1 typing nothing; the two payload flags are mutually exclusive with exactly one required
- [x] A-016 R5: A failed dispatch delivery still restores the stashed completion signals (existing behavior preserved through the relocation)

### Code Quality

- [x] A-017 Readability/maintainability: relocated code moves verbatim apart from package qualifiers; new code matches the surrounding comment density and naming
- [x] A-018 No duplicated utilities: one gate, one pointer composer, one argv builder — dispatch and the new verbs share them from `internal/pane`
- [x] A-019 Mirror sweep: every touched `src/kit/skills/*.md` carries its SPEC-*.md update; no stale restatement of the old send refusal or dispatch-locked gate survives in `docs/specs/` or `src/kit/skills/`
- [x] A-020 Canonical sources only: no edit under `.claude/skills/`; `gofmt -l src/go` is clean; Go changes ship tests in this change
- [x] A-021 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [x] A-022 No unnecessary duplication: Existing utilities reused where applicable

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — the relocation deleted the dispatch-side copies of the gate, creators, `Tail`, and `PointerPrompt` in this same change; every remaining `internal/dispatch` symbol retains call sites, and no existing code was made redundant without being removed.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Relocation target is `internal/pane` (no sibling package) | Import graph read at apply: `internal/pane` imports only resolve/status/statusfile; the gate needs nothing from dispatch — no cycle possible | S:80 R:75 A:90 D:85 |
| 2 | Confident | Placement *execution* (`SplitPlacement`/`splitArgs`/`OpenSplitPane`) moves with the mechanics; placement *decision* (`SplitTarget`/`SelectPaneShape`) stays dispatch | The decision reads dispatch records + config (policy); the execution is pure tmux argv work both layers share | S:65 R:70 A:75 D:65 |
| 3 | Confident | `fab pane open` resolves config when a fab root is resolvable and falls back to empty config (built-in providers) otherwise | The pane family carries no FabRoot guard; probes run from scratch dirs against built-in providers | S:60 R:80 A:70 D:65 |
| 4 | Confident | `fab pane open` spawns an unnamed new window when `$TMUX_PANE` is unset or `--server` is given | Mirrors `SelectPaneShape`'s rules (a pane id is meaningless cross-socket) minus the dispatch titling convention, which is dispatch's identity, not a primitive's | S:55 R:75 A:70 D:60 |
| 5 | Confident | `deliver` verification failures exit 1; the 2/3 scheme binds to pane validation and tmux I/O (ready's probe errors are tmux I/O → 3) | The scheme names pane-missing/tmux-failure; a verified-but-refused delivery is operational, and gate errors are not type-classified today | S:55 R:80 A:65 D:55 |
| 6 | Confident | `PointerPrompt` moves to `internal/pane` as the shared pointer-line composer | `pane deliver --prompt-file` is defined as dispatch parity on the pointer line; one composer prevents the divergent-copy failure | S:60 R:75 A:75 D:65 |
| 7 | Confident | `pane deliver --prompt-file` types the path as supplied (no repo-relative rewrite) | A generic pane's cwd is unknown to fab; the dispatch repo-relative rule rests on the worker cwd being the repo root, which only dispatch guarantees | S:50 R:80 A:65 D:60 |

7 assumptions (1 certain, 6 confident, 0 tentative).
