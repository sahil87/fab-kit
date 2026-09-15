# Plan: Interactive Pane Stage Dispatch (Third Adapter)

**Change**: 260805-zxe0-interactive-pane-stage-dispatch
**Intake**: `intake.md`

## Requirements

### Dispatch Runtime: pane mode on `fab dispatch start`

#### R1: `--pane` opts a dispatch into an interactive tmux window
`fab dispatch start <change> <stage>` SHALL accept a `--pane` boolean flag. When set, the stage worker SHALL run **interactively in a tmux window** instead of as a detached headless `sh -c` wrapper. When unset, behavior SHALL be byte-identical to today's headless path (headless remains the default; the tmux-independence guarantee of the headless path is unchanged).

- **GIVEN** a change/stage whose resolved tier's provider carries a `session_command`
- **WHEN** `fab dispatch start <change> <stage> --pane` runs with a reachable tmux server
- **THEN** a tmux window is created running the composed interactive command, the dispatch record is persisted with the pane identity, and no `{stage}.exit` wrapper is involved
- **AND** GIVEN the same invocation without `--pane`, the headless detached wrapper launches exactly as before

#### R2: Pane mode composes `session_command`, never `dispatch_command`
Pane mode SHALL resolve the stage's tier → provider exactly as the headless path does, then compose the provider's **`session_command`** through `internal/spawn.WithProfile` (the same composition `fab agent` performs). It MUST NOT read, require, or fall back to `dispatch_command`; conversely the headless path MUST NOT fall back to `session_command` (the existing no-cross-fallback rule stands in both directions). A resolved provider with no `session_command` SHALL error naming the stage, the resolved tier, and the `providers.<name>.session_command` config key.

- **GIVEN** a tier whose provider carries a `session_command` but **no** `dispatch_command`
- **WHEN** `fab dispatch start <change> <stage> --pane` runs
- **THEN** the dispatch succeeds using the composed `session_command`
- **AND** GIVEN the same tier, a `--pane`-less `start` still errors with the `dispatch_command` config-key hint
- **AND** GIVEN a provider with no `session_command`, `--pane` errors with the `providers.<name>.session_command` hint

#### R3: `--pane` without a reachable tmux server is a hard error
Pane mode SHALL require a reachable tmux server and SHALL fail with a non-zero exit and an actionable stderr message when one is not reachable, launching nothing and persisting no dispatch record. Reachability SHALL be established by an actual tmux query (not merely an `$TMUX` environment read), so a `--server`-targeted invocation from outside tmux works. `fab dispatch start` SHALL accept `--server <name>` / `-L <name>` (mirroring the `fab pane` family's persistent flag) to target a specific tmux socket; when `--server` is empty and `$TMUX` is unset, the error names the requirement.

- **GIVEN** no reachable tmux server (or an unreachable `--server` socket)
- **WHEN** `fab dispatch start <change> <stage> --pane` runs
- **THEN** it exits non-zero with a message naming tmux reachability and the headless alternative, and creates no `{stage}.yaml`
- **AND** GIVEN `--pane` is not set, no tmux probe occurs at all (headless stays tmux-independent)

#### R4: `--pane` and `--timeout` are mutually exclusive
`--timeout` enforces its limit inside the headless `sh -c` wrapper via POSIX `timeout`; pane mode has no such wrapper. Supplying both SHALL be a usage error (non-zero exit) naming the exclusion rather than silently ignoring `--timeout`.

- **GIVEN** `fab dispatch start <change> <stage> --pane --timeout 600`
- **WHEN** it runs
- **THEN** it exits non-zero naming the `--pane`/`--timeout` exclusion, and launches nothing

#### R5: Prompt delivery is a prompt file plus a one-line pointer send
Pane mode SHALL persist the stage prompt read from stdin to `.fab-dispatch/{4-char-change-id}/{stage}-prompt.md` (the same path and writer the headless path uses), then deliver to the pane worker a **one-line pointer** naming that path — not the prompt body. The pointer SHALL be embedded at spawn as the interactive command's single quoted prompt argument (the `_cli-agents` § Spawn Composition form), so no `send-keys` delivery and no printed-prompt probe is required for the initial delivery.

- **GIVEN** a multi-thousand-token stage prompt on stdin
- **WHEN** `fab dispatch start <change> <stage> --pane` runs
- **THEN** the full prompt is written to `{stage}-prompt.md` and the window's command carries only the one-line pointer referencing that path
- **AND** the pointer names the repo-relative prompt path so the worker can read it from the window's cwd (the repo root)

#### R6: Pane dispatch records pane identity in its state file
The `Dispatch` record persisted at `{stage}.yaml` SHALL carry the pane mode's identity fields — the tmux pane ID, the window name, and the tmux server label — alongside the existing fields, each omitted when empty so a headless record is byte-identical to today's. Pane mode SHALL record `spawn_cmd` as the composed interactive command. `pid`/`pgid` SHALL be omitted for a pane dispatch (liveness is a pane property, not a pid property).

- **GIVEN** a successful `--pane` start
- **WHEN** `{stage}.yaml` is read back
- **THEN** it carries `pane`, `window`, and (when set) `server`, plus `spawn_cmd`/`started_at`
- **AND** GIVEN a headless start, the serialized record contains none of the pane fields

### Dispatch Runtime: pane-mode state, kill, and the read surfaces

#### R7: Pane-mode completion detection keys on the result file plus pane liveness
`fab dispatch status <change> <stage>` SHALL derive a pane dispatch's state from **result-file presence** and **pane liveness**, never from an exit file:

| Condition | State |
|-----------|-------|
| `{stage}-result.yaml` present | `done` |
| result absent, pane alive | `running` |
| result absent, pane dead (or tmux server gone) | `orphaned` |

The five state strings SHALL remain byte-stable; a pane dispatch simply emits a **subset** of three. Result-file presence SHALL take precedence over pane liveness, so a worker that produced its result and still sits at its prompt reads `done`.

- **GIVEN** a pane dispatch whose window is alive with no result file
- **WHEN** `fab dispatch status` runs
- **THEN** it reports `running`
- **AND** GIVEN the result file is then written while the pane still lives, it reports `done`
- **AND** GIVEN the pane is killed with no result file, it reports `orphaned`

#### R8: `failed` and `failed (no-result)` are unreachable on the pane path
Pane mode has no exit-code channel, so the two exit-code-derived states SHALL be unreachable for a pane dispatch: a crashed or killed pane worker collapses into `orphaned`. The `DeriveState` five-state machine for the headless path SHALL be unchanged; pane derivation SHALL be a separate pure function so both are independently table-testable.

- **GIVEN** any pane dispatch in any observable condition
- **WHEN** its state is derived
- **THEN** the result is one of `running` / `done` / `orphaned` — never `failed` or `failed (no-result)`
- **AND** GIVEN a headless dispatch, all five states remain reachable exactly as before

#### R9: `fab dispatch kill` has a pane-mode implementation
`kill` SHALL detect a pane dispatch from its record and kill the **tmux pane** (`tmux kill-pane -t <pane>`) rather than signalling a process group. It SHALL stay idempotent: killing an already-dead (or already-gone) pane is a benign no-op with a clear report, and a missing dispatch record is a clear error. After a successful kill with no result file present, the dispatch reads `orphaned` (per R7) — no separate marker file is written.

- **GIVEN** a live pane dispatch with no result file
- **WHEN** `fab dispatch kill <change> <stage>` runs
- **THEN** the pane is killed, a killed report is printed, and a subsequent `status` reports `orphaned`
- **AND** GIVEN the pane is already gone, `kill` exits 0 with an already-dead report

#### R10: `status --json`, `logs`, and `clean` have defined pane-mode behavior
- `status --json` SHALL carry the pane identity: a `mode` field (`headless` | `pane`), and `pane`/`window` fields populated for a pane dispatch. `pid`/`pgid` SHALL be omitted for a pane dispatch, and `exit` remains omitted (no exit file exists). Existing headless JSON keys SHALL be unchanged for a headless dispatch.
- `logs` SHALL report that a pane dispatch keeps no log file, naming `fab pane capture <pane>` as the pane-mode equivalent, rather than printing the generic missing-log message. (An interactive worker's output lives in tmux scrollback, not in `{stage}.log`.)
- `clean` SHALL require no pane-mode change — it removes state dirs and never inspects a record's mode; a `clean` over a live pane dispatch removes the state dir without killing the pane (the same non-guarantee the headless path already carries for a live process).

- **GIVEN** a pane dispatch
- **WHEN** `fab dispatch status --json` runs
- **THEN** the object carries `mode: "pane"` with `pane`/`window` populated and no `pid`/`pgid`/`exit`
- **AND** GIVEN `fab dispatch logs` on that dispatch, it reports the no-log-file fact and points at `fab pane capture`
- **AND** GIVEN a headless dispatch, `status --json` carries `mode: "headless"` with `pid`/`pgid` as before

#### R11: Pane windows carry a distinct dispatch-window name, not the operator's `»` marker
A pane dispatch's tmux window SHALL be named from a dedicated convention — `fab-{id}-{stage}` — and MUST NOT carry the operator's `»` (U+00BB) enrollment prefix or the `›` (U+203A) done marker. Those markers signal operator ownership and monitored-set enrollment, which a pipeline dispatch does not have.

- **GIVEN** `fab dispatch start abcd apply --pane`
- **WHEN** the window is created
- **THEN** its name is `fab-abcd-apply` and carries no `»`/`›` prefix
- **AND** an operator running on the same server may still enroll the window by its own rules, which is what adds the marker

### Contract & Documentation: the third adapter

#### R12: `harness-adapters.md` documents three adapters and the pane protocol variant
`docs/specs/harness-adapters.md` SHALL be amended from a two-adapter to a **three-adapter** catalog (native Agent-tool / headless CLI / interactive pane), with the shared protocol expressing per-adapter observation: prompt-file + pointer delivery, result-file-only detection, and the reachable-state subset. The dispatch-prompt obligations SHALL be stated as binding **all three** adapters. The five-state machine section SHALL state which states each adapter can observe.

- **GIVEN** a reader of `harness-adapters.md`
- **WHEN** they look for the adapter catalog
- **THEN** they find three adapters, with the pane adapter's delivery mechanism, detection mechanism, and reachable-state subset stated
- **AND** no sentence still counts the adapters as two

#### R13: The kit's dispatch-seam documentation carries the pane option
`src/kit/skills/_preamble.md` § CLI-Adapter Dispatch SHALL document the pane mode as an option **within** the CLI-adapter branch (skills pass `--pane` when the user asked for a watchable stage; the branch itself stays keyed on `dispatch=` presence), and § Dispatch-Prompt Obligations SHALL note the prompt-file delivery variant plus the state-subset consequence. `src/kit/skills/_cli-fab.md` § fab dispatch SHALL document the `--pane` and `--server` flags, the `session_command` composition, the mutual exclusion with `--timeout`, the pane-mode state derivation, and the pane-mode `kill`/`logs`/`clean` behavior.

- **GIVEN** a dispatch-seam skill following `_preamble.md`
- **WHEN** the user asks for a watchable stage on a CLI-dispatched tier
- **THEN** the documented procedure tells it to pass `--pane` inside the existing `dispatch=` branch and what state subset to expect
- **AND** `_cli-fab.md` § fab dispatch documents every new flag and behavior (constitution: CLI change ⇒ `_cli-fab.md` + tests)

#### R14: Steering is documented as contract-neutral
The spec amendment SHALL state that a user MAY converse with a pane worker mid-stage and that this changes no contract: the worker still owes `{stage}-result.yaml`, still ends with the terminal `fab status refresh` epilogue, and still never runs `fab status` **transition** commands — the orchestrator owns all transitions. A steered worker redirected away from producing a result surfaces as a never-`done` dispatch the orchestrator escalates on. This is documentation only — no code enforces or detects steering.

- **GIVEN** a user who types into a running pane worker's window
- **WHEN** the worker finishes
- **THEN** the same result-file + refresh-epilogue obligations apply, and the orchestrator's transition ownership is unchanged
- **AND** GIVEN the worker never produces a result, the dispatch stays `running`/`orphaned` and the orchestrator escalates by its existing never-`done` path

#### R15: Every swept sibling and mirror is updated in this change
The declared sweep class SHALL be updated in full: the SPEC mirrors of every edited skill (`SPEC-_preamble.md`, `SPEC-_cli-fab.md`, plus any dispatch-site skill actually edited and its mirror), and the aggregate specs (`skills.md`, `glossary.md`, `architecture.md`) and dispatch-site skills (`_pipeline.md`, `fab-continue.md`, `fab-adopt.md`) **where and only where** they restate the two-adapter model or enumerate adapter modes — verified by grep, not assumed.

- **GIVEN** the constitution's rule that a `src/kit/skills/*.md` edit requires its `docs/specs/skills/SPEC-*.md` update
- **WHEN** apply finishes
- **THEN** every edited skill has its mirror updated in the same change
- **AND** a repo-wide grep for two-adapter counting language returns no stale occurrence

### Non-Goals

- No change to native Agent-tool dispatch, `fab resolve-agent` output, or the `providers:` schema — no new provider config field in v1 (the interactive invocation is the provider's existing `session_command`).
- No completion notification — polling stands; an `rk notify` line in the worker prompt remains the opt-in signal.
- No auto-GC change for `.fab-dispatch/` — prompt files ride the two existing cleanup paths (archive-time deletion, explicit `fab dispatch clean`).
- Not a replacement for the headless CLI adapter — `--pane` is opt-in per invocation; unattended pipelines keep the tmux-independent default.
- No `docs/memory/` edits (hydrate's job) and no migration (`.fab-dispatch/` is transient state, not user data).
- No pane-mode `--timeout` analogue and no pane-mode log file.

### Design Decisions

#### Pane mode is a flag on `fab dispatch start`, not a new command family
**Decision**: Interactive dispatch is `fab dispatch start <change> <stage> --pane`, sharing the resolution, state directory, refuse-if-running concurrency, and status/kill/logs/clean surfaces with the headless path.
**Why**: The sequencer wiring already branches at `fab dispatch`, and the five-state machine plus `.fab-dispatch/{id}/` layout are exactly the machinery a pane dispatch needs; a parallel command family would duplicate all of it and force every dispatch site to learn a second grammar.
**Rejected**: A new `fab pane-dispatch` family (duplicated state machine); a headless mode on `fab pane` (the inverse conflation `fab dispatch` was created to avoid).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

#### Detection keys on the result file, with pane liveness only distinguishing running from orphaned
**Decision**: A pane dispatch is `done` when `{stage}-result.yaml` exists, `running` when it does not and the pane lives, `orphaned` when it does not and the pane is gone. Result presence wins over liveness.
**Why**: An interactive worker never exits on task completion — it finishes and sits at its prompt — so an exit-code channel does not exist to key on. The result file is already the contract's success token for both existing adapters, so reusing it keeps one success definition across all three.
**Rejected**: Screen-pattern detection via `tmux capture-pane` (scrollback-dependent, ambiguous, and the `_cli-agents` § Await guidance explicitly prefers an artifact); requiring the worker to exit after writing its result (throws away the steer-after-finish property that motivates the adapter).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

#### Prompt file plus one-line pointer, embedded at spawn
**Decision**: The full stage prompt is written to `{stage}-prompt.md` (the path the headless path already uses) and the pane worker receives a one-line pointer to it, embedded as the interactive command's single quoted prompt argument at window creation.
**Why**: A multi-thousand-token stage prompt cannot ride `send-keys` or argv reliably, and embedding the pointer at spawn sidesteps the printed-prompt trap entirely — there is no existing buffer to probe when the window is created with its prompt already attached. The file doubles as a debugging artifact and rides the existing cleanup paths.
**Rejected**: Sending the whole prompt via `fab pane send` (the printed-prompt trap plus send-keys length limits); passing the prompt on the interactive command's stdin (an interactive TUI reads stdin as keystrokes, not as a prompt).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

#### Pane windows get a `fab-{id}-{stage}` name, not the operator's `»` marker
**Decision**: A dispatch window is named `fab-{id}-{stage}` and carries no `»`/`›` prefix.
**Why**: The `»` prefix is the operator's enrollment marker — it asserts that the window is in the operator's monitored set and that the operator owns its lifecycle. A pipeline dispatch has neither property, so pre-marking would make the operator's tab bar lie about what it tracks. A distinct, greppable name convention gives the same at-a-glance identification without the false claim, and an operator that genuinely enrolls the window still adds the marker through its own idempotent primitive.
**Rejected**: Prefixing `»` at creation (falsely signals operator ownership); leaving the window unnamed (indistinguishable from an ad-hoc shell tab).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

#### `--pane` is mutually exclusive with `--timeout`
**Decision**: Supplying both flags is a usage error.
**Why**: `--timeout` is implemented as POSIX `timeout N` inside the headless `sh -c` wrapper, which pane mode does not construct. Silently ignoring the flag would let an orchestrator believe a bound is enforced when nothing enforces it — precisely the class of silent-nonenforcement the `failed (no-result)` state exists to prevent elsewhere.
**Rejected**: Silently ignoring `--timeout` under `--pane` (a false guarantee); implementing a pane-side timer (re-introduces a supervisor process the dispatch design deliberately has none of).
*Introduced by*: 260805-zxe0-interactive-pane-stage-dispatch

## Tasks

### Phase 1: Core state + derivation (`internal/dispatch`)

- [x] T001 Extend the `Dispatch` record in `src/go/fab/internal/dispatch/dispatch.go` with pane-mode fields (`Pane`, `Window`, `Server`, all `yaml:",omitempty"`) and a `Mode()`/`IsPane()` accessor deriving the mode from `Pane` being non-empty; make `PID`/`PGID` `omitempty` so a pane record serializes without them and a headless record stays byte-identical. Add named mode constants (`ModeHeadless`, `ModePane`) — no magic strings. <!-- R6 -->
- [x] T002 Add the pure pane-state derivation `DerivePaneState(resultPresent, paneAlive bool) State` to `src/go/fab/internal/dispatch/dispatch.go`, returning `done` / `running` / `orphaned` with result-presence taking precedence; leave `DeriveState` (the headless five-state machine) untouched. <!-- R7 R8 -->
- [x] T003 Add tmux pane primitives to the build-tagged split (`src/go/fab/internal/dispatch/dispatch_posix.go` / `dispatch_windows.go`): `ServerReachable(server string) error`, `OpenWindow(server, name, dir, cmd string) (paneID string, err error)`, `PaneAlive(paneID, server string) bool`, and `KillPane(paneID, server string) error` — each delegating to `internal/pane`'s `RunCmd`/`WithServer`/`StderrError` helpers rather than re-implementing tmux invocation. <!-- R3 R7 R9 R11 -->
- [x] T004 [P] Extend `src/go/fab/internal/dispatch/dispatch_test.go` with table tests for `DerivePaneState` (all four result×liveness combinations), the `Mode()`/`IsPane()` accessors, and round-trip YAML assertions proving a headless record serializes with no pane keys and a pane record with no `pid`/`pgid`. <!-- R6 R7 R8 -->

### Phase 2: Command wiring (`cmd/fab`)

- [x] T005 Add `--pane` and `--server`/`-L` flags to `dispatchStartCmd()` in `src/go/fab/cmd/fab/dispatch_start.go`, with the `--pane`/`--timeout` mutual-exclusion guard (keyed on `Flags().Changed`, mirroring `agent.go`'s guard style) and a `--server`-without-`--pane` no-op note. <!-- R1 R4 -->
- [x] T006 Split `runDispatchStart` in `src/go/fab/cmd/fab/dispatch_start.go` into the shared prologue (resolve change → ID → dir, load config, resolve tier profile, refuse-if-running, persist the stdin prompt, clear stale files) plus two mode-specific launch tails: the existing headless tail and a new pane tail that probes tmux reachability, composes the provider's `session_command` via `spawn.WithProfile`, opens the window with the pointer prompt, and persists the pane record. <!-- R1 R2 R3 R5 R6 R11 -->
- [x] T007 Implement pane-mode `status` in `src/go/fab/cmd/fab/dispatch_status.go`: branch on the loaded record's mode, use `DerivePaneState` with `PaneAlive`, and extend `dispatchStatusJSON` with `mode` plus `omitempty` `pane`/`window` and `omitempty` `pid`/`pgid`. <!-- R7 R8 R10 -->
- [x] T008 Implement pane-mode `kill` in `src/go/fab/cmd/fab/dispatch_kill.go`: branch on the record's mode, kill the tmux pane idempotently (already-gone pane = benign no-op report), leaving the headless process-group path unchanged. <!-- R9 -->
- [x] T009 Implement pane-mode `logs` in `src/go/fab/cmd/fab/dispatch_logs.go`: for a pane dispatch report that no log file exists and point at `fab pane capture <pane>`, instead of the generic missing-log message. <!-- R10 --> <!-- rework: the suggested command must include the server socket when the record carries one — emit `fab pane capture -L <server> <pane>` when rec.Server is non-empty (dispatch_logs.go:39); mirror the same fix in _preamble.md § CLI-Adapter Dispatch logs bullet -->

### Phase 3: Tests

- [x] T010 Extend `src/go/fab/cmd/fab/dispatch_start_test.go`: `--pane` + `--timeout` is a usage error; `--pane` with no reachable tmux server errors and persists no record; `--pane` against a provider with no `session_command` errors with the config-key hint; `--pane` succeeds against a provider with only a `session_command` (no `dispatch_command`) when tmux is reachable, persisting the pane record and writing the full prompt file with the window command carrying only the pointer. <!-- R1 R2 R3 R4 R5 R6 R11 -->
- [x] T011 [P] Extend `src/go/fab/cmd/fab/dispatch_status_test.go`: pane-mode `running`/`done`/`orphaned` across pane-alive × result-present, `done` winning while the pane still lives, and the `--json` shape (`mode: "pane"`, `pane`/`window` present, `pid`/`pgid`/`exit` absent; `mode: "headless"` on the headless path). <!-- R7 R8 R10 -->
- [x] T012 [P] Extend `src/go/fab/cmd/fab/dispatch_kill_test.go` and `dispatch_logs_test.go`: pane-mode kill of a live pane reports killed and leaves the dispatch `orphaned`, pane-mode kill of an already-gone pane is a benign no-op, and pane-mode `logs` reports the no-log fact naming `fab pane capture`. <!-- R9 R10 -->

### Phase 4: Spec, kit docs, and mirror sweep

- [x] T013 Amend `docs/specs/harness-adapters.md` from two adapters to three: add the interactive-pane adapter section, express the shared protocol per-adapter (prompt-file + pointer delivery, result-file-only detection, reachable-state subset), state the obligations as binding all three adapters, add the per-adapter state-reachability note, and add the steering-is-contract-neutral paragraph. <!-- R12 R14 -->
- [x] T014 Amend `src/kit/skills/_preamble.md`: § CLI-Adapter Dispatch gains the pane-mode option inside the `dispatch=` branch (with the state-subset consequence), and § Dispatch-Prompt Obligations notes the prompt-file delivery variant and that the obligations bind the pane adapter identically. <!-- R13 R14 -->
- [x] T015 Amend `src/kit/skills/_cli-fab.md` § fab dispatch: `--pane`/`--server` flags on `start`, the `session_command` composition and no-cross-fallback statement, the `--pane`/`--timeout` exclusion, the pane-mode state table and unreachable-state note, and pane-mode `kill`/`logs`/`clean` behavior. <!-- R13 -->
- [x] T016 Grep-verify the dispatch-site skills (`src/kit/skills/_pipeline.md`, `fab-continue.md`, `fab-adopt.md`) and `_cli-agents.md` for adapter-mode enumeration or two-adapter counting; update only where they actually enumerate modes, and record the verification outcome in `## Notes`. <!-- R15 -->
- [x] T017 Update the SPEC mirrors of every skill edited above (`docs/specs/skills/SPEC-_preamble.md`, `SPEC-_cli-fab.md`, plus any dispatch-site mirror whose skill T016 changed) per the constitution's skill⇒SPEC rule. <!-- R15 -->
- [x] T018 Grep-verify and update the aggregate specs (`docs/specs/skills.md`, `glossary.md`, `architecture.md`) wherever they restate the two-adapter model or the `fab dispatch` flag surface. <!-- R15 --> <!-- rework: MUST-FIX — docs/specs/index.md:29 (hand-curated, editable) still reads "the two dispatch adapters (native Agent-tool + CLI `fab dispatch`)"; rewrite to name three adapters (native Agent-tool / headless CLI / interactive pane --pane) + the pane three-state subset. The prior grep patterns missed the literal phrase "two dispatch adapters" — re-sweep with broader patterns (`two dispatch`, `both adapters`, `two-adapter`, `pair of adapters`). ALSO should-fix: docs/specs/skills.md:642 calls .fab-dispatch/{id}/ the "headless-dispatch state dir" — drop the mode-specific adjective (pane dispatches share the dir) -->
- [x] T019 Correct the `ServerReachable` doc comment in `src/go/fab/internal/dispatch/pane_mode.go:37-39`: the "zero-session server is still reachable" sentence describes a case that does not exist as written (default `exit-empty` kills the server with its last session; with `exit-empty off` list-sessions exits 0 but new-window fails gracefully via StderrError). Rewrite the comment to describe actual behavior; no code change required. <!-- R3 -->
- [x] T020 Run the affected Go packages' tests (`internal/dispatch`, `cmd/fab`), then the whole `src/go/fab` module, and fix any failures. <!-- R1 R7 R9 R10 --> <!-- renumbered from T019 during rework cycle 1: the orchestrator's added doc-comment task claimed T019, colliding with this pre-existing task; both are preserved -->

## Execution Order

- Phase 1 (T001–T003) blocks Phase 2 — the command wiring consumes the record fields, the derivation, and the tmux primitives.
- T004 is independent of Phase 2 and can run alongside it.
- T006 depends on T005 (flags) and on T001/T002/T003.
- T010–T012 depend on their respective Phase 2 tasks.
- Phase 4 documentation (T013–T018) is independent of the Go work and can run in parallel with Phase 3, except T016–T018, which depend on T014/T015 being written (the mirrors describe them).
- T019 (comment-only) is independent of everything else.
- T020 (the test run) runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab dispatch start <change> <stage> --pane` runs the stage worker in a tmux window; without `--pane` the headless detached path is unchanged.
- [x] A-002 R2: Pane mode composes the provider's `session_command` (never `dispatch_command`), and a provider with only a `session_command` dispatches successfully under `--pane` while still erroring without it.
- [x] A-003 R3: `--pane` against an unreachable tmux server exits non-zero with an actionable message and persists no dispatch record; the headless path performs no tmux probe.
- [x] A-004 R4: `--pane --timeout N` is a usage error naming the exclusion.
- [x] A-005 R5: The full stage prompt lands in `{stage}-prompt.md` and the window command carries only a one-line pointer to that path.
- [x] A-006 R6: `{stage}.yaml` carries `pane`/`window`/`server` for a pane dispatch and none of them (with `pid`/`pgid` present) for a headless one.
- [x] A-007 R7: Pane-mode `status` reports `done` on result presence, `running` on alive-and-no-result, `orphaned` on dead-and-no-result.
- [x] A-008 R9: `fab dispatch kill` kills the tmux pane for a pane dispatch, is idempotent, and leaves the dispatch reading `orphaned`.
- [x] A-009 R10: `status --json` carries `mode` plus pane identity; `logs` reports the pane-mode no-log fact naming `fab pane capture`; `clean` needed no change.
- [x] A-010 R11: A dispatch window is named `fab-{id}-{stage}` and carries no `»`/`›` prefix.
- [x] A-011 R12: `docs/specs/harness-adapters.md` catalogs three adapters with the pane adapter's delivery, detection, and state-subset stated.
- [x] A-012 R13: `_preamble.md` § CLI-Adapter Dispatch and § Dispatch-Prompt Obligations carry the pane option and prompt-file variant; `_cli-fab.md` § fab dispatch documents every new flag and behavior.
- [x] A-013 R14: Steering is documented as contract-neutral (result file + refresh epilogue + orchestrator-owned transitions unchanged) with no code enforcing it.

### Behavioral Correctness

- [x] A-014 R8: `failed` and `failed (no-result)` are unreachable for a pane dispatch (crash/kill collapse to `orphaned`), while all five states remain reachable for a headless dispatch; the state strings are byte-identical to before.
- [x] A-015 R7: Result-file presence takes precedence over pane liveness, so a finished worker still sitting at its prompt reads `done`, not `running`.
- [x] A-016 R1: A headless `fab dispatch start`'s persisted record and printed output are byte-identical to the pre-change behavior (no pane keys, same dispatched line).

### Scenario Coverage

- [x] A-017 R3: A test covers the unreachable-tmux hard error including the no-record-persisted assertion.
- [x] A-018 R7: Tests cover the pane state matrix across pane-alive × result-present, including the `done`-wins case.
- [x] A-019 R9: A test covers pane-mode kill of a live pane and of an already-gone pane.
- [x] A-020 R5: A test asserts the prompt file holds the full prompt while the window command carries only the pointer.

### Edge Cases & Error Handling

- [x] A-021 R2: A provider with no `session_command` under `--pane` errors naming the `providers.<name>.session_command` key; the headless no-`dispatch_command` error is unchanged.
- [x] A-022 R4: The `--pane`/`--timeout` exclusion is enforced before any launch or file write.
- [x] A-023 R7: A vanished tmux server (not merely a dead pane) reads `orphaned` rather than erroring out of `status`.
- [x] A-024 R9: A `kill` on a record with no pane and no pid, or on a missing record, produces a clear error rather than a panic or a signal to pid 0.

### Code Quality

- [x] A-025 Pattern consistency: New code follows the surrounding `internal/dispatch` + `cmd/fab/dispatch*.go` patterns — named constants over magic strings, pure functions separated from I/O, build-tagged platform split for syscalls/tmux, cobra `Flags().Changed` guards.
- [x] A-026 No unnecessary duplication: tmux invocation reuses `internal/pane`'s `RunCmd`/`WithServer`/`StderrError`; command composition reuses `internal/spawn.WithProfile`; the shared `start` prologue is not duplicated across the two launch tails.
- [x] A-027 Composition over inheritance / no god functions: `runDispatchStart` is decomposed rather than grown into a branching monolith exceeding the package's typical function size.
- [x] A-028 Canonical sources only: every kit edit lands in `src/kit/skills/`, never `.claude/skills/`; `docs/memory/` is untouched (hydrate's job).
- [x] A-029 Mirror sweep complete (R15): every edited skill has its `docs/specs/skills/SPEC-*.md` mirror updated in this change, and a repo-wide grep finds no stale two-adapter counting language. — **MET in rework cycle 1**: `docs/specs/index.md:29` rewritten to three adapters + the pane three-state subset, and the broader re-sweep (`two dispatch`, `both adapters`, `two-adapter`, `pair of adapters`, `one of two`, `two non-native`, `second adapter`) over `docs/specs/` + `src/kit/` + `src/go/` + `scripts/` + `README.md` leaves only correct post-amendment usages (see § Sweep verification). Skill⇒SPEC mirror half: 7 skills, 7 mirrors (`fab-archive.md` added this cycle; its mirror was already mode-neutral).
- [x] A-030 Tests conform to the spec (Constitution VII): no implementation was bent to satisfy a fixture, and `_cli-fab.md` + tests accompany the CLI signature change.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

### Sweep verification (T016/T018)

Recorded during apply — grep results and the edit/no-edit decision per file.

- `src/kit/skills/_pipeline.md` — **updated**. Its per-stage dispatch-adapter note (line ~62) and Behavior dispatch note (line ~60) enumerate the adapter modes on the `dispatch=` branch; added the pane-mode option pointer to `_preamble.md` § CLI-Adapter Dispatch. Per-stage Step 1/2/3 + Auto-Rework bullets reference the Behavior note rather than re-enumerating, so they needed no edit.
- `src/kit/skills/fab-continue.md` — **updated**. Normal Flow Step 1's sub-agent dispatch contract and the dispatch-shorthand note enumerate the branch outcomes; added the pane-mode pointer in both.
- `src/kit/skills/fab-adopt.md` — **updated** (Step 3 review dispatch enumerates `native ⇒ Agent tool / CLI ⇒ fab dispatch`); added the pane pointer. Step 4 hydrate reuses `_pipeline.md` Step 3 by reference, so no separate edit.
- `src/kit/skills/_cli-agents.md` — **updated**. § Spawn Composition's "Open it in a pane" is the primitive the pane adapter consumes; added a cross-reference noting `fab dispatch start --pane` as its pipeline consumer. No adapter-mode enumeration to correct.
- `docs/specs/skills.md` — **no edit needed** (grep-verified). It carries no adapter-model restatement: its only `fab dispatch` mention is the `/fab-archive` step-2 line about deleting `.fab-dispatch/{id}/` state, which is mode-independent and still accurate. Its `helpers:`/authoring content is unaffected.
- `docs/specs/glossary.md` — **updated**: the `providers` row's `session_command` consumer list now includes `fab dispatch start --pane` and the no-cross-fallback rule is stated in both directions; a new **Dispatch adapter** term was added to the *Core Concepts* table (beside **Sub-agent**) naming all three adapters and the pane mode's reachable-state subset.
- `docs/specs/architecture.md` — **updated**: the `providers:` config-illustration comment block now names `fab dispatch start --pane` as a `session_command` consumer and states the no-fallback rule in both directions. (Not a byte-pinned copy of `fab config reference` — no config-surface change was made in this change, and `internal/agent`'s mirrors test deliberately excludes this file.)
- `docs/specs/stage-models.md` — **updated**: § Harness-adapter boundary's cross-harness-dispatch bullet said the native adapter is "one of *two* dispatch adapters catalogued in harness-adapters.md" → now *three*, naming both `fab dispatch` modes, plus an explicit note that resolution is unchanged by pane mode (no new resolver line, no new provider field, per-invocation selection, and that composing `session_command` is not a cross-fallback).
- SPEC mirrors updated for every edited skill (T017): `SPEC-_preamble.md` (Summary paragraph + the § Subsection Inventory tree's CLI-Adapter Dispatch / Dispatch-Prompt Obligations nodes), `SPEC-_cli-fab.md` (the `fab dispatch` row rewritten as two modes), `SPEC-_pipeline.md` (the resolve/branch tree node), `SPEC-fab-continue.md` (the Dispatch annotation), `SPEC-fab-adopt.md` (the Step 3 flow node), `SPEC-_cli-agents.md` (the Spawn Composition row's pipeline-consumer pointer).
- Repo-wide grep for two-adapter counting language (`two adapters`, `two-adapter`, `BOTH adapters`, `one of two dispatch`, `one of *two*`) over `docs/specs/` + `src/kit/` returns only two hits, both **correct** post-amendment usages inside `harness-adapters.md` ("the *other two* adapters" of three). `docs/memory/pipeline/execution-skills.md` still carries the old phrasing — deliberately untouched, as `docs/memory/` is hydrate's job.
- Canonical-source discipline verified: zero edits under `.claude/skills/` and zero under `docs/memory/`.

#### Rework cycle 1 re-sweep (T018)

The cycle-0 patterns were too narrow — all keyed on the noun `adapters`, so the literal `docs/specs/index.md:29` phrase **"the two dispatch adapters"** (noun *dispatch*, plural adapters as a separate word) escaped, and no pattern covered the `docs/` index tier at all. Re-swept with the broadened set `two dispatch|both adapters|two-adapter|pair of adapters|one of two|two adapters|BOTH adapter|two non-native|second adapter`, case-insensitive, over `docs/specs/` + `src/kit/` + `src/go/` + `scripts/` + `README.md` (excluding the frozen `docs/specs/findings/` and `fab/changes/archive/` records):

- `docs/specs/index.md:29` — **fixed**. Rewritten to name three adapters (native Agent-tool / headless CLI `fab dispatch start` / interactive pane `fab dispatch start --pane`), state that the obligations bind all three, and add the pane adapter's three-state subset (`running`/`done`/`orphaned`) beside the five-state machine. The file is hand-curated (its own header says so) and carries no generator, so it is directly editable.
- `docs/specs/harness-adapters.md:151,201` — **correct as-is**: "the *other two* adapters" of three, a post-amendment usage.
- `src/kit/skills/_cli-fab.md:466` + `docs/specs/skills/SPEC-_cli-fab.md:28` — **correct as-is**: "the two *non-native* adapters" (of three total, native excluded), and both already name all three explicitly.
- `docs/specs/srad.md:261`, `src/go/fab/internal/spawn/spawn.go:52`, `src/go/fab/cmd/fab/dispatch_start.go:58` — **out of class**: "one of two ways/modes" about SRAD handling and `WithProfile`/launch modes, nothing to do with the adapter catalog.
- `docs/memory/` and the zxe0/j3cm change folders — deliberately untouched (hydrate's job; own-change artifacts describing the two→three transformation).

Should-fix, same cycle: the mode-specific adjective on the shared state dir was swept as a class, not just at the flagged line — `docs/specs/skills.md:642` and its canonical kit source `src/kit/skills/fab-archive.md:76` both called `.fab-dispatch/{id}/` the "headless-dispatch state dir"; both now read "dispatch state dir (shared by both `fab dispatch` modes — headless and `--pane`)". `docs/specs/skills/SPEC-fab-archive.md` was already mode-neutral ("the dispatch state dir"), so the skill⇒SPEC mirror requirement is satisfied with no mirror edit; its "one of the two `fab dispatch` cleanup paths" is about *cleanup* paths (archive + `clean`), not adapters, and stays.

T009's server fix propagated as a class too: the message-carrying code (`dispatch_logs.go`), the kit seam docs (`_preamble.md` § CLI-Adapter Dispatch `logs` bullet, `_cli-fab.md` § dispatch → logs), and both SPEC mirrors (`SPEC-_preamble.md` summary + subsection tree, `SPEC-_cli-fab.md` row) now all state that the suggested capture command carries `-L <server>` when the record has one. Because `status --json` exposes `pane` but not `server`, the docs additionally point the reader at the `fab dispatch logs` report as the copy-pasteable source rather than telling them to hand-assemble from `--json`.

## Deletion Candidates

None — this change adds a second launch mode to an existing command family and reuses the existing state directory, loader, save path, refuse-if-running check, and state-string vocabulary. Verified by grep: `WrapperArgv`, `dispatch.Launch`, `KillGroup`, and `DeriveState` all retain live non-test call sites (the headless path is untouched), the pre-change `dispatch_command`-absent error string was rewritten in place rather than duplicated (`modeCommand` now serves both modes), and no `internal/pane` helper was re-implemented. The one file the plan expected not to exist (`internal/dispatch/pane_mode.go`) is net-new, not a duplicate of the build-tagged split.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Completion detection = result-file presence; pane liveness only distinguishes `running` from `orphaned`, and result presence wins over liveness | Carried from intake assumption 1 (Certain, user-discussed); the precedence rule follows necessarily — an interactive worker sits at its prompt after finishing, so liveness-first would report `running` forever | S:80 R:80 A:90 D:90 |
| 2 | Certain | Prompt delivery = full prompt at `.fab-dispatch/{id}/{stage}-prompt.md` + a one-line pointer embedded at spawn as the interactive command's quoted prompt argument | Carried from intake assumption 2 (Certain); embedding at spawn (rather than a post-spawn `fab pane send`) follows `_cli-agents` § Spawn Composition and sidesteps the printed-prompt trap entirely — there is no pre-existing buffer to probe | S:80 R:85 A:90 D:85 |
| 3 | Certain | Mode selection = the per-invocation `--pane` flag reusing the provider's `session_command`; no new provider config field in v1 | Intake assumption 3, explicitly user-confirmed at activation; not reopened | S:85 R:80 A:90 D:90 |
| 4 | Certain | `failed` / `failed (no-result)` are unreachable on the pane path; crash and kill collapse into `orphaned`, and the five state strings stay byte-stable | Intake assumption 5 (Confident) plus the code reality that no exit-code channel exists in pane mode — a subset, not new strings | S:80 R:85 A:90 D:85 |
| 5 | Confident | Pane windows are named `fab-{id}-{stage}` and carry NO operator `»`/`›` marker | The intake's Open Question, decided per the recommended default: `»` asserts operator ownership and monitored-set enrollment, which a pipeline dispatch does not have — pre-marking would make the operator's tab bar misreport what it tracks. A distinct name gives identification without the false claim, and the operator's own idempotent `ensure-prefix` still adds the marker if it genuinely enrolls the window | S:65 R:85 A:80 D:75 |
| 6 | Confident | `--pane` and `--timeout` are mutually exclusive (usage error), rather than `--timeout` being silently ignored | `--timeout` is implemented as POSIX `timeout` inside the headless `sh -c` wrapper that pane mode never constructs; silently accepting the flag would advertise a bound nothing enforces — the same false-success class `failed (no-result)` exists to prevent | S:55 R:85 A:85 D:80 |
| 7 | Confident | `fab dispatch start` gains a `--server`/`-L` flag (mirroring the `fab pane` family) and tmux reachability is established by a real tmux query, not an `$TMUX` env read | The intake fixes "requires a reachable tmux server" but not the mechanism; a dispatching orchestrator may itself be headless, so an `$TMUX`-only gate would make `--pane` unusable from exactly the cross-harness callers the adapter exists for. `fab resolve --pane` sets the precedent (`--server` present ⇒ skip the `$TMUX` guard, query the socket server-wide) | S:60 R:80 A:85 D:75 |
| 8 | Confident | Pane identity (`pane`, `window`, `server`) is persisted into the existing `{stage}.yaml` record as `omitempty` fields, with `pid`/`pgid` becoming `omitempty` too — no second state file, no new file type beyond the already-specified prompt file | The intake says "records pane identity in `.fab-dispatch/{id}/`" without fixing the file; extending the existing record keeps one loader, one save path, and one refuse-if-running check. `omitempty` on both sides keeps a headless record byte-identical, so the change is additive on disk | S:70 R:80 A:85 D:80 |
| 9 | Confident | Pane-mode `logs` reports the no-log fact and points at `fab pane capture <pane>` rather than printing the generic missing-log message; `clean` needs no pane-mode change | The intake requires `status`/`logs`/`clean` behavior to be "defined and tested" without fixing it. An interactive worker's output is tmux scrollback, so there is no log file to print and the actionable answer is the pane-capture equivalent. `clean` never inspects a record's mode, so it is correct as-is (documented, tested by its existing coverage) | S:60 R:85 A:85 D:80 |
| 10 | Confident | `status --json` gains a `mode` field (`headless`\|`pane`) plus `omitempty` `pane`/`window`, keeping headless JSON byte-identical | The intake requires pane `status` behavior but not the JSON shape. The `--json` surface's documented contract is additive evolution with no `schema_version`, so adding fields is the sanctioned move; a `mode` discriminator is what lets a consumer know which state subset to expect | S:65 R:85 A:85 D:80 |
| 11 | Confident | Pane state derivation is a separate pure function (`DerivePaneState`) rather than extra parameters on `DeriveState` | The headless five-state machine is a byte-stable documented contract with exhaustive table tests; threading pane liveness through it would couple two different observation models in one function. Two pure functions stay independently table-testable, matching the package's existing "pure core, table-tested" pattern | S:60 R:85 A:90 D:80 |
| 12 | Confident | Steering stays documentation-only — no code detects, gates, or reports a steered worker | Intake assumption 6 (Confident) plus §5's explicit "documented contract, not code". The never-`done` escalation path the orchestrator already owns covers a worker redirected away from producing a result | S:75 R:80 A:85 D:80 |

12 assumptions (4 certain, 8 confident, 0 tentative).
