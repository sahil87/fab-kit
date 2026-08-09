# Plan: Pane Readiness Gate + Send-Keys Delivery

**Change**: 260809-3oz7-pane-readiness-gate-sendkeys-delivery
**Intake**: `intake.md`

## Requirements

### Dispatch Runtime: Pane Entry Verb (`open`)

#### R1: `fab dispatch open` spawns a pane worker WITHOUT delivering the prompt
`fab dispatch open <change> <stage> [--server <name>]` SHALL spawn the pane exactly as
`fab dispatch start --pane` did — same `internal/spawn` composition of the resolved provider's
`interactive_command` with `{model}`/`{effort}` substitution, same two placement shapes (split into the
dispatcher's window with the stacked right column carved at `dispatch.column_width`, or the new-window
fallback), same record-keyed sibling detection, same `fab-{id}-{stage}` identity, same `--server`
handling, same refuse-if-running check, same stale exit/result/log clearing — **except that no prompt
pointer is appended to the composed command**. The full stage prompt SHALL still be read from stdin and
persisted to `.fab-dispatch/{id}/{stage}-prompt.md`. `open` is a **pane-only** verb: it forces pane mode
and hard-errors on either missing prerequisite (reachable tmux, `interactive_command`) rather than
descending.

- **GIVEN** a provider with an `interactive_command` and a reachable tmux server
- **WHEN** `fab dispatch open b91h apply < prompt.md` runs
- **THEN** a tmux pane is created running the composed command **verbatim** (no trailing pointer argument)
- **AND** `{stage}-prompt.md` holds the stdin bytes
- **AND** the record persists `pane`/`window`/`server` and reports `opened <id>/<stage> (pane %N, split, title fab-<id>-<stage>)` (or `… (pane %N, window fab-<id>-<stage>)` in the new-window shape)

#### R2: `fab dispatch ready` is a mechanical, purely echo-based readiness probe
`fab dispatch ready <change> <stage>` SHALL report exactly one of `ready`, `booting`, or `parked`,
derived **only** from a literal sentinel send plus pane captures — never from `@rk_agent_state`, never
from a pattern table of known dialogs. It SHALL send the sentinel with `send-keys -l` (no Enter), clear
it with `C-u` whether or not it echoed, and never submit anything. Every non-`ready` report SHALL carry
the pane id, the record's socket when non-empty, and a trailing capture snippet. All three
classifications exit **0** — the report string is the sole discriminator (the `fab dispatch wait`
precedent); non-zero is reserved for real errors (no record, non-pane record, dead pane). It SHALL be
idempotent and safe to re-run (Constitution III).

- **GIVEN** a live pane whose TUI accepts input
- **WHEN** `fab dispatch ready b91h apply` runs
- **THEN** stdout is exactly `ready\n` and the sentinel has been cleared with `C-u`
- **GIVEN** a pane parked at a trust dialog that swallows the sentinel and whose screen is stable
- **WHEN** the probe runs
- **THEN** stdout is `parked` followed by `pane: %N`, an optional `server: <name>`, and the capture snippet
- **GIVEN** a pane whose screen content is still changing between the two probe captures
- **WHEN** the probe runs and the sentinel did not echo
- **THEN** the report is `booting`

#### R3: `fab dispatch deliver` is the sole, verified prompt-delivery mechanism
`fab dispatch deliver <change> <stage> [--prompt-file <path>]` SHALL hand a pane worker its prompt via a
verified send-keys choreography: internal readiness probe → `C-u` → send the one-line pointer literally →
capture-verify the echoed line → `Enter` → confirm the worker went busy (the screen advanced) → **one
retry** on any verification failure → exit non-zero with the capture snippet on the second failure. With
no flag the pointer names `.fab-dispatch/{id}/{stage}-prompt.md`; `--prompt-file <path>` points it at a
continuation prompt instead — the **pane-arm resume** the apply-worker resume series' C2 item describes.
Before the first send attempt it SHALL clear the stale `{stage}-result.yaml` and `{stage}.exit` so the
dispatch reads `running` again after a continuation. On success it SHALL flip the record's `delivered`
marker; on failure the marker SHALL remain unset so no false state is written.

- **GIVEN** a pane opened by `open` with `delivered` unset
- **WHEN** `fab dispatch deliver b91h apply` runs and the pointer echoes and the screen advances after Enter
- **THEN** the record records `delivered: true` and stdout is `delivered <id>/<stage> (pane %N, prompt <repo-relative-path>)`
- **GIVEN** the pointer does not echo on the first attempt
- **WHEN** deliver retries once and the retry verifies
- **THEN** the delivery succeeds and a `warning: …; retrying` line is written to stderr
- **GIVEN** both attempts fail verification
- **THEN** deliver exits non-zero, carries the capture snippet, and leaves `delivered` unset

#### R4: `deliver` refuses to inject input into a worker that is mid-stage
`deliver` SHALL refuse when the record is headless (naming `fab dispatch start`), when the pane is dead,
and when the record is already `delivered: true` **and** the derived state is `running` — the last being
the code-level expression of the contract's no-input-injection rule (§ R11): a delivered worker executing
its stage is never re-delivered to. `delivered: true` with state `done` is the sanctioned continuation
case and SHALL proceed.

- **GIVEN** a pane dispatch that has been delivered and whose result file is absent (state `running`)
- **WHEN** `fab dispatch deliver` runs
- **THEN** it exits non-zero naming the mid-stage worker and sends nothing
- **GIVEN** the same dispatch after its result file appears (state `done`)
- **WHEN** `fab dispatch deliver … --prompt-file <continuation>` runs
- **THEN** delivery proceeds

#### R5: `fab dispatch start` becomes headless-only
`start` SHALL compose and launch only the headless arm. `--pane` and `--server` SHALL no longer select
pane mode on `start`; supplying either SHALL raise an actionable error naming `fab dispatch open`, before
any state write. `--timeout` and `--headless` stay. When automatic mode resolution lands on **pane**,
`start` SHALL error before writing state with guidance to use `open`, mirroring the existing native-result
error shape. The `--pane`/`--timeout` mutual-exclusion rule dissolves with the flag.

- **GIVEN** `dispatch.mode: pane`, tmux reachable, and a provider with an `interactive_command`
- **WHEN** `fab dispatch start b91h apply < prompt.md` runs
- **THEN** it exits non-zero naming `fab dispatch open` and writes no record
- **GIVEN** `fab dispatch start b91h apply --pane`
- **THEN** it exits non-zero with the same `open` guidance

#### R6: `restart`'s pane arm performs the spawn-only step and hands the gate back
`restart` SHALL keep re-deriving its mode from the current environment. When it lands on **headless** it
relaunches fully, as today. When it lands on **pane** it SHALL perform the `open` step only (spawn, no
delivery, `delivered` unset) and report on stderr that the readiness gate and `deliver` must follow —
Go cannot run the agent judgment the gate needs.

- **GIVEN** an orphaned pane dispatch and a configuration that still resolves to pane
- **WHEN** `fab dispatch restart b91h apply` runs
- **THEN** a fresh pane is opened with no prompt delivered, stdout carries the `opened …` line, and stderr names `fab dispatch ready` + `fab dispatch deliver`

#### R7: The record gains an additive `delivered` marker; the state machine is unchanged
`Dispatch` SHALL gain `delivered` (bool, `omitempty`, absent ⇒ not delivered), written by `deliver` only.
It is bookkeeping for `deliver`/`status` reporting, **not** a state: the five-state machine and the pane
three-state derivation (result present ⇒ `done`; result absent + pane alive ⇒ `running`; result absent +
pane dead ⇒ `orphaned`) SHALL be untouched. A headless record SHALL serialize byte-identically to before.
`fab dispatch status --json` SHALL expose `delivered` for pane records only.

- **GIVEN** a headless dispatch
- **WHEN** its record is written
- **THEN** the YAML carries no `delivered` key and `status --json` omits it
- **GIVEN** a pane dispatch that has been delivered
- **THEN** `status --json` carries `"delivered": true` alongside `pane`/`window`

#### R8: Pane I/O helpers are single-sourced in `internal/pane`
The tmux capture and send-keys argv builders and their runners SHALL live once in `internal/pane`
(`CaptureArgs`/`Capture`, `SendLiteralArgs`/`SendLiteral`, `SendKeyArgs`/`SendKey`), with
`cmd/fab/pane_capture.go` and `cmd/fab/pane_send.go` delegating to them, so the readiness gate and the
delivery choreography reuse the existing utilities instead of duplicating them.

- **GIVEN** the gate needs to capture a pane and send literal text
- **WHEN** it runs
- **THEN** it goes through the same `internal/pane` helpers `fab pane capture` / `fab pane send` use

#### R9: The gate and delivery choreography are unit-testable without a tmux server
The tmux surface the gate uses SHALL be an interface in `internal/dispatch` with a real tmux
implementation and a test fake, and the classification SHALL be a pure function over captures, matching
the package's existing pure-decision precedent (`SelectMode`, `DerivePaneState`, `splitPlacement`). The
inter-capture delay SHALL be a struct field so tests run at zero delay; it SHALL carry no flag and no
config field.

- **GIVEN** a scripted capture fixture
- **WHEN** the readiness derivation runs
- **THEN** `ready`/`booting`/`parked` are asserted with no tmux process involved

### Contract: `harness-adapters.md`

#### R10: The adapter-3 prompt-delivery row records verified post-spawn delivery
The spec's adapter table row for the interactive pane SHALL change from "prompt file + one-line pointer
embedded at spawn" to "prompt file + pointer delivered post-spawn via the verified send-keys choreography
(`fab dispatch deliver`), behind an agent-driven readiness gate", and the superseded rationale sentence
claiming spawn-time embedding "sidesteps the printed-prompt trap entirely" SHALL be explicitly recorded as
amended (the spec's own Amendments-are-explicit rule). Adapter-3 mechanics sections SHALL name `open`
where they named `start --pane`; the three-state pane derivation table SHALL be unchanged.

- **GIVEN** a reader of `harness-adapters.md`
- **WHEN** they read § 3 and the amendment header block
- **THEN** the delivery mechanism is post-spawn send-keys and the change is recorded as an explicit amendment naming this change

#### R11: The no-input-injection rule gains a bounded pre-delivery carve-out
The spec SHALL state that between `open` and a successful `deliver` the pane is **not yet a dispatched
worker** — it holds no stage context to corrupt — so the orchestrator MAY send keys to it (the gate's
judgment rounds), with `fab dispatch ready`/`deliver` as the sanctioned mechanical senders. From
successful delivery onward the existing rule applies unchanged: peek, kill, restart, notify, stop, reap —
never input injection; mid-stage walls still escalate.

- **GIVEN** a pane parked at a trust dialog before delivery
- **THEN** the orchestrator MAY answer it
- **GIVEN** the same wall appearing mid-stage after delivery
- **THEN** the orchestrator MUST escalate instead

#### R12: Pane resume and stage-aware reap are recorded in the contract
The spec SHALL record pane resume as the pane-adapter counterpart of the native Worker Continuation
amendment — same apply-only scope, same mandatory fresh-dispatch fallback, same profile fixity (no
re-resolution on resume), reviewer independence untouched — and SHALL state that the headless adapter
remains deliberately non-resumable. The reap section SHALL record that *when* the orchestrator invokes
reap is stage-aware, while invoking it at all remains the orchestrator's choice and the Go guard is
unchanged.

- **GIVEN** the tv3g amendment's statement that pane resume is "a separate follow-up change"
- **WHEN** this change lands
- **THEN** that statement is replaced by the pane-resume capability itself, recorded as an amendment

### Capability Grammar: `interactive_command`

#### R13: `interactive_command` is pure launch grammar
No code path SHALL append a prompt argument to `interactive_command`. Its documented meaning SHALL become
*how to launch an interactive session — prompt delivery is fab's, not the command's*, across
`defaults.yaml`, `internal/configref`, `docs/specs/stage-models.md`, `src/kit/skills/_cli-agents.md`, and
`_cli-fab.md`. The stale rationale that `agy`/`kimi` carry no `interactive_command` **because they cannot
accept a positional prompt** SHALL be rewritten to name the real remaining consideration (first-run walls
and echo behavior, probed per provider before shipping the field), pointing at backlog `[agik]` as the
owner of the roster flip. The shipped built-in `interactive_command` values SHALL NOT change and no
provider SHALL gain or lose one in this change.

- **GIVEN** a provider whose CLI rejects a positional prompt
- **WHEN** an `interactive_command` is configured for it
- **THEN** nothing in fab appends a prompt to that command, so pane capability no longer depends on positional-prompt support

### Orchestrator Wiring

#### R14: The pane branch of the CLI-adapter dispatch procedure becomes open → gate → deliver → wait
`_preamble.md` § CLI-Adapter Dispatch SHALL replace single-shot `start --pane` with: `open` → readiness
gate loop → `deliver` → `wait`. The gate loop: `ready` ⇒ deliver; `booting` ⇒ wait briefly and re-probe
(booting re-probes do **not** consume judgment rounds, under a bounded consecutive-boot allowance after
which the pane is treated as `parked`); `parked` ⇒ a judgment round in which the orchestrator reads the
snippet and MAY answer the wall with raw `tmux send-keys` — **maximum 2 rounds per gate**, then escalate.
**Login/credential walls escalate immediately and are never answered.** Escalation surfaces the capture
evidence, sends `rk notify` behind the fail-silent `command -v rk` gate, and stops on the existing failure
path, leaving the pane alive for the human. Gate exhaustion SHALL NOT auto-descend to headless. The gate
MAY also be run ahead of apply as a warm-up, with no special casing.

- **GIVEN** a fresh worktree whose provider trust-prompts on first launch
- **WHEN** the gate probes and reads `parked`
- **THEN** the orchestrator answers the dialog, re-probes, reads `ready`, and delivers
- **GIVEN** two judgment rounds that still do not reach `ready`
- **THEN** the orchestrator escalates and does not descend to headless

#### R15: Reap timing becomes stage-aware in the wiring only
The **apply** pane SHALL NOT be reaped when its `done` result is read — it is the resume target across
rework cycles. It SHALL be reaped when **review passes**, or when the pipeline stops/escalates past apply
for good. Every other stage's pane SHALL still be reaped immediately on done-read. The call stays
unconditional and dumb (reap is a reported no-op for a headless record, a non-`done` state, or a disabled
knob), so the Go guard in `dispatch.DecideReap` SHALL be untouched.

- **GIVEN** a pane-dispatched apply worker whose result was just read
- **WHEN** the orchestrator processes the `done` result
- **THEN** it does not reap, and reaps only after `fab status finish <change> review`

#### R16: The pane arm of the auto-rework loop resumes via `deliver --prompt-file`
`_pipeline.md` § Auto-Rework Loop item 3 SHALL gain the pane-arm counterpart of the native resume-first
rule: when the apply dispatch record is a pane dispatch whose pane is still alive, the orchestrator writes
the continuation prompt (triaged findings + rework action + the re-read-from-disk instruction, carrying
obligations 1 and 3 but not the context files) to `.fab-dispatch/{id}/apply-continuation.md` and delivers
it with `fab dispatch deliver <change> apply --prompt-file …`. Any failure falls back to a fresh
open → gate → deliver dispatch — resume stays an optimization, never correctness-bearing. Scope stays
apply-only; review workers are never resumed.

- **GIVEN** a failed review verdict and a live, un-reaped apply pane
- **WHEN** the loop reaches item 3
- **THEN** it delivers the continuation prompt into that pane instead of opening a new one
- **GIVEN** the deliver call fails
- **THEN** the loop dispatches a fresh pane worker with the full obligations

#### R17: `_cli-fab.md` documents the new verb surface, and every touched skill gets its SPEC mirror
`_cli-fab.md` § fab dispatch SHALL document `open`, `ready`, and `deliver` (arguments, flags, outputs,
exit codes, report strings), narrow `start` to headless, record `restart`'s pane arm, and note the
stage-aware reap timing. Every edited `src/kit/skills/*.md` SHALL be paired with its
`docs/specs/skills/SPEC-*.md` mirror in the same change (Constitution Additional Constraints), and the
old claims SHALL be grep-swept repo-wide so no restatement site keeps the superseded contract.

- **GIVEN** the constitution's skill/SPEC mirror rule
- **WHEN** `_preamble.md`, `_pipeline.md`, `_cli-fab.md`, and `_cli-agents.md` are edited
- **THEN** `SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-_cli-fab.md`, and `SPEC-_cli-agents.md` are updated in the same change

#### R18: Go tests cover every new and narrowed surface
Per the project test strategy (test-alongside), `internal/dispatch` and `cmd/fab` SHALL gain tests for:
`open` (record shape, no-pointer composition, prerequisites), the readiness derivation against scripted
capture fixtures, the delivery verify/retry/fail choreography plus marker flip and continuation mode,
`deliver`'s refusal guards, `start`'s pane rejection, `restart`'s pane arm, and the unchanged reap guard.
Existing `start --pane` tests SHALL migrate to `open`. New report strings SHALL be pinned.

- **GIVEN** the changed packages
- **WHEN** `go test ./internal/dispatch/... ./cmd/fab/... ./internal/pane/...` runs
- **THEN** it passes with the new coverage in place

### Non-Goals

- Provider roster data — giving `agy`/`kimi` an `interactive_command` is backlog `[agik]`, run in parallel. This change ships provider-neutral machinery only and changes no shipped provider value.
- Go-side dialog pattern matching. Wall classification is the orchestrator's judgment, never a table in the binary.
- Any change to the native or headless adapters: native Worker Continuation stands as-is, headless keeps stdin delivery and stays non-resumable by decision.
- New config fields, new dispatch states, record-path changes, or a migration. The gate budget is contract prose, not config; `delivered` is an additive transient key in `.fab-dispatch/`.

### Design Decisions

#### Send-keys is the only pane delivery mechanism
**Decision**: The spawn-time positional pointer is removed outright for every provider, including claude; `fab dispatch deliver` is the single delivery engine for both initial dispatch and rework continuation.
**Why**: The one-shot positional path is fire-and-forget and unverifiable — a provider that silently drops it (observed with agy) leaves a worker at an empty prompt while the dispatch reads `running`. One engine, hardened by every dispatch, also gives resume for free.
**Rejected**: A hybrid preferring the positional one-shot where supported — it needs a per-provider "accepts positional prompt" capability bit, which is exactly the presence-as-policy coupling the dispatch-mode ladder change eliminated, and it keeps an unverifiable arm.
*Introduced by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

#### First-run walls are answered by agent judgment, not Go
**Decision**: `fab dispatch ready` classifies mechanically (echo / screen-stability) and reports; the orchestrator reads the snippet and decides, with a 2-round budget and immediate escalation for login walls.
**Why**: Dialogs are a version treadmill and provider-specific; a half-matched pattern pressing Enter into an unknown screen is worse than stalling. An agent already reads screens for a living.
**Rejected**: A Go pattern table of known dialogs; and provider-specific trust-store pre-seeding (probed and working for agy, but undocumented-format provider machinery inside a provider-neutral binary).
*Introduced by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

#### The pre-delivery pane is not yet a worker
**Decision**: The no-input-injection rule carves out the window between `open` and successful `deliver`.
**Why**: A pane holding no stage context cannot have that context corrupted by a keystroke, so the rule's whole purpose is absent there. Bounding the carve-out at successful delivery keeps the mid-stage guarantee exactly as strong as before.
**Rejected**: Keeping the rule absolute — that leaves first-run walls with no recovery story at all, since the only outcome is a never-`done` escalation after the fact.
*Introduced by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

#### Stage-aware reap is wiring, not a Go guard change
**Decision**: `dispatch.DecideReap` is untouched; only the moment the orchestrator calls reap moves (apply after review passes, everything else on done-read).
**Why**: Reap is orchestrator-invoked by design, so "when" is already a wiring concern; touching the guard would put pipeline policy into the binary.
**Rejected**: A `--keep-for-resume` flag or a stage list in Go — both duplicate a decision the caller already owns.
*Introduced by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery

## Tasks

### Phase 1: Setup

- [x] T001 [P] Add single-sourced tmux pane I/O to `src/go/fab/internal/pane/pane.go` — `CaptureArgs`/`Capture`, `SendLiteralArgs`/`SendLiteral`, `SendKeyArgs`/`SendKey` — and make `cmd/fab/pane_capture.go` (`capturePaneArgs`, `capturePaneContent`) and `cmd/fab/pane_send.go` (`sendTextArgs`, `sendEnterArgs`) delegate to them without changing their behavior or existing tests <!-- R8 -->
- [x] T002 [P] Add the additive `Delivered bool \`yaml:"delivered,omitempty"\`` field to `dispatch.Dispatch` in `src/go/fab/internal/dispatch/dispatch.go`, documenting that absence means "not delivered" and that headless records stay byte-identical <!-- R7 -->

### Phase 2: Core Implementation

- [x] T003 Add `src/go/fab/internal/dispatch/gate.go`: the `PaneIO` interface (capture / send literal / send key), its tmux implementation, the `Readiness` string type with `ReadyReady`/`ReadyBooting`/`ReadyParked` constants, the pure `DeriveReadiness(echoed bool, first, second string) Readiness` classifier, the sentinel constant, and `Gate.Probe(paneID)` running the send → capture → `C-u` choreography <!-- R2 R9 -->
- [x] T004 Add `Gate.Deliver` to `src/go/fab/internal/dispatch/gate.go`: readiness precondition → `C-u` → literal pointer send → capture-verify echo → `Enter` → busy confirmation, with exactly one retry and a structured failure carrying the last capture <!-- R3 R9 --> <!-- rework c4 MUST-FIX: Gate.Deliver retry is dead for the busy-failure class — deliverOnce compares typed-count against the PRE-C-u probe capture, so attempt 2 baselines against attempt 1's leftover pointer (1 > 1 = false) and reports the WRONG cause ("did not echo" instead of "did not react to Enter"). Fix: take the baseline capture immediately AFTER deliverOnce's own C-u (gate.go:266) and compare against that; anti-false-verify property strictly strengthens (scrollback pointer appears post-clear too). ALSO close review c4 SF2 with probe evidence, no code change: live-tested 2026-08-09 — claude @50 and @30 cols (mid-word wrap) and agy @50 cols all countWrapped=1; both TUIs draw horizontal-rule input boxes with NO side borders, wraps insert only whitespace; extend countWrapped/squeeze doc comment with this and note kimi is covered by [agik]'s pre-shipping echo probe (review c4 MF1, SF2) -->
- [x] T005 Drop the pointer from pane spawn composition in `src/go/fab/internal/dispatch/dispatch.go` — `WindowCommand` composes the launch grammar alone; keep `PointerPrompt` as the delivery-side pointer composer and rewrite both doc comments to state that delivery is post-spawn <!-- R1 R13 -->
- [x] T006 Add `src/go/fab/cmd/fab/dispatch_open.go`: the `open` verb (pane forced, `--server` only), reusing the shared resolve → validate → refuse-if-running → persist-prompt → stale-clear → `launchPane` path, reporting `opened <id>/<stage> (…)` <!-- R1 -->
- [x] T007 Add `src/go/fab/cmd/fab/dispatch_ready.go`: the `ready` verb — load the record, require pane mode and a live pane, run `Gate.Probe`, print the classification plus (for non-`ready`) `pane:`/`server:`/snippet, exit 0 for all three <!-- R2 --> <!-- rework c4: the `--- last N lines ---` header prints even when the snippet is empty (ordinary booting-on-blank-screen case) — skip the header when snippet == "" (review c4 NTH3) -->
- [x] T008 Add `src/go/fab/cmd/fab/dispatch_deliver.go`: the `deliver` verb with `--prompt-file`, the headless/dead-pane/mid-stage refusal guards, stale result+exit clearing before the first send, `Gate.Deliver`, the `delivered` marker flip, and the `delivered <id>/<stage> (pane %N, prompt <path>)` report <!-- R3 R4 --> <!-- rework c4: stashSignals removes files as it goes but returns nil,err on a later failure, discarding already-removed entries — a partial failure loses the previous cycle's result file and wedges the dispatch (delivered:true + no result). Return the partial stash alongside the error and restore it on the error path (review c4 SF1) -->
- [x] T009 Narrow `start` to headless in `src/go/fab/cmd/fab/dispatch_start.go`: split the launch flag surface so `start` registers `--timeout`/`--headless` plus hidden `--pane`/`--server` that raise the `fab dispatch open` guidance before any work, and error with the same guidance when automatic resolution lands on pane <!-- R5 --> <!-- rework c4: dispatch_start.go:600 `windowCmd := resolvedCmd` is vestigial post-WindowCommand-removal — inline resolvedCmd at both use sites (review c4 NTH2) -->
- [x] T010 Give `restart`'s pane arm the spawn-only behavior in `src/go/fab/cmd/fab/dispatch_restart.go` and the shared launch path: on a pane landing, open without delivering and print the `fab dispatch ready` + `fab dispatch deliver` hand-back note on stderr <!-- R6 -->
- [x] T011 Register `open`/`ready`/`deliver` in `src/go/fab/cmd/fab/dispatch.go` (and refresh the family docstring), and expose `delivered` for pane records in `src/go/fab/cmd/fab/dispatch_status.go` `--json` <!-- R1 R2 R3 R7 -->

### Phase 3: Tests

- [x] T012 Extend `src/go/fab/internal/dispatch/dispatch_test.go` (or a new `gate_test.go`) with: the `DeriveReadiness` table against scripted captures, `Gate.Probe` against a fake `PaneIO` (echo / no-echo-stable / no-echo-changing), `Gate.Deliver`'s success, one-retry, and both-failed paths, the `delivered` marshal/unmarshal shape, and the unchanged headless record bytes <!-- R18 --> <!-- rework c4: add the retry-after-busy-failure regression test (a1 echo ok / a1 Enter ignored / a2 retype succeeds — expected pass, currently fails with "did not echo"); TestDeliverVerifiesEchoAndSubmission pinned ops gains one capture; fix TestDeliverPropagatesIOFailure doc comment overstating "never a silently-retried verification failure" (the retry IS spent on IO failure; the assertion is right, the comment wrong) (review c4 MF1, NTH1) -->
- [x] T013 Add `src/go/fab/cmd/fab/dispatch_open_test.go`, `dispatch_ready_test.go`, `dispatch_deliver_test.go` and migrate the `start --pane` cases in `dispatch_start_test.go`/`dispatch_restart_test.go`: pane rejection on `start`, restart's pane hand-back, deliver's refusal guards, and the pinned `opened …` / `delivered …` / readiness report strings <!-- R18 -->

### Phase 4: Contract & Wiring

- [x] T014 Amend `docs/specs/harness-adapters.md`: adapter-table delivery row, the superseded spawn-time rationale, § 3 mechanics renamed from `start --pane` to `open` (+ the two new verbs), the no-input-injection carve-out, pane resume, stage-aware reap timing, and a dated amendment note in the header block <!-- R10 R11 R12 -->
- [x] T015 Rewrite the `interactive_command` semantics as pure launch grammar in `src/go/fab/internal/agent/defaults.yaml`, `src/go/fab/internal/configref/configref.go`, `docs/specs/stage-models.md`, and `src/kit/skills/_cli-agents.md` — replacing the positional-prompt rationale for agy/kimi with the first-run-wall/echo consideration and the `[agik]` pointer, changing no shipped provider value <!-- R13 -->
- [x] T016 Rewrite `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch's pane branch (open → gate → deliver → wait, gate budget, escalation classes, stage-aware reap timing) and extend § Worker Continuation with the pane arm <!-- R14 R15 R16 -->
- [x] T017 Wire the pane-arm resume and reap timing into `src/kit/skills/_pipeline.md` § Auto-Rework Loop item 3 and the worker-release paragraph <!-- R15 R16 -->
- [x] T018 Document `open`/`ready`/`deliver`, the narrowed `start`, the pane-arm `restart`, and the reap-timing note in `src/kit/skills/_cli-fab.md` § fab dispatch <!-- R17 -->
- [x] T019 Update the SPEC mirrors for every touched skill source: `docs/specs/skills/SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-_cli-fab.md`, `SPEC-_cli-agents.md`, `SPEC-fab-continue.md` (and `SPEC-fab-adopt.md`, which restated the reap call) <!-- R17 -->
- [x] T020 Grep-sweep the repo for the superseded claims (`start --pane`, "embedded at spawn", "positional argument", the pane-resume-is-a-follow-up statement) and update every restatement site outside `docs/memory/` — including `docs/specs/architecture.md`, `docs/specs/glossary.md`, `docs/specs/skills.md`, and any `SPEC-fab-*.md` carrying them <!-- R17 -->

### Phase 5: Verification

- [x] T021 Run `go build ./...`, `go vet ./...`, and `go test ./internal/dispatch/... ./internal/pane/... ./cmd/fab/...` from `src/go/fab`, then widen to the full Go suite; fix any failure at its source <!-- R18 -->
- [x] T022 Add a one-clause pointer in `src/kit/skills/fab-continue.md` § Review Behavior → Verdict (Pass) to `_preamble.md` § CLI-Adapter Dispatch step 3's stage-aware reap timing (manual Path-A under pane mode currently leaks the apply pane's column slice), updating its SPEC mirror only if the mirror restates the sequence <!-- R15, review c3 NTH3 -->

## Execution Order

- T001 and T002 are independent and precede everything in Phase 2.
- T003 blocks T004, T007, and T008.
- T005 blocks T006 (open composes the pointer-free command).
- T009 and T010 share `dispatch_start.go`'s launch path — do T009 first.
- T011 lands after T006–T008 exist.
- Phase 3 follows Phase 2; Phase 4 is independent of Phase 3 but its `_cli-fab.md` / spec text must match the report strings pinned in T013.
- T021 runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab dispatch open <change> <stage>` spawns a pane running the composed `interactive_command` with **no** pointer argument, persists the stdin prompt to `{stage}-prompt.md`, records the pane identity, and reports `opened …`
- [x] A-002 R2: `fab dispatch ready` reports exactly one of `ready`/`booting`/`parked` from a sentinel echo plus captures, always clears the sentinel, never sends Enter, and exits 0 in all three cases
- [x] A-003 R3: `fab dispatch deliver` verifies echo, submits, confirms the worker went busy, retries once, flips `delivered` on success, and leaves it unset on failure — the c4 retry-baseline defect is fixed: `deliverOnce` now takes `newlyEchoed`'s baseline from a capture made after its OWN `C-u` (`gate.go:266–270`), so a retry after an ignored Enter verifies the pointer it re-typed instead of comparing 1 against 1. Pinned by `TestDeliverRetriesAfterAnIgnoredEnter`, with `TestDeliverDoesNotVerifyAPointerAlreadyOnScreen` confirming the anti-false-verify property survives the move and `TestDeliverVerifiesEchoAndSubmission` pinning the capture's position in the op sequence.
- [x] A-004 R3: `--prompt-file <path>` delivers a continuation prompt into a live pane — the pane-arm resume — and clears the stale result/exit files so the dispatch reads `running` again
- [x] A-005 R4: `deliver` refuses a headless record, a dead pane, and a `delivered`+`running` (mid-stage) worker, each with an actionable message and no keystrokes sent
- [x] A-006 R5: `fab dispatch start` launches only headless; `--pane`/`--server` and a pane-resolving configuration each error with `fab dispatch open` guidance before any state write
- [x] A-007 R6: `fab dispatch restart` landing on pane opens without delivering and names `ready` + `deliver` on stderr
- [x] A-008 R7: the record carries `delivered` only when true; headless record bytes are unchanged; `status --json` exposes it for pane records only
- [x] A-009 R8: `internal/pane` owns the capture/send argv builders and runners, and the two `cmd/fab` call sites delegate rather than duplicate
- [x] A-010 R10: `harness-adapters.md`'s adapter-3 delivery row names post-spawn verified send-keys behind the readiness gate, with the superseded spawn-time rationale explicitly recorded as amended
- [x] A-011 R11: the spec states the pre-delivery carve-out and that the no-input-injection rule applies unchanged from successful delivery onward
- [x] A-012 R12: the spec records pane resume (apply-only, fresh-dispatch fallback, profile fixity) and stage-aware reap timing, and no longer calls pane resume a future follow-up
- [x] A-013 R13: no code path appends a prompt to `interactive_command`, and its documented meaning is pure launch grammar everywhere it is described
- [x] A-014 R14: `_preamble.md`'s pane branch is open → gate → deliver → wait with the 2-round judgment budget, the immediate login-wall escalation, the bounded boot allowance, and no auto-descent on exhaustion
- [x] A-015 R15: the wiring reaps every non-apply stage's pane on done-read and apply's only after review passes, with `dispatch.DecideReap` unchanged
- [x] A-016 R16: `_pipeline.md` item 3 carries the pane-arm resume via `deliver --prompt-file` with the mandatory fresh-dispatch fallback and apply-only scope
- [x] A-017 R17: `_cli-fab.md` documents the three new verbs and the narrowed `start`/`restart`, and every touched skill file has its SPEC mirror updated in this change

### Behavioral Correctness

- [x] A-018 R1: the composed pane command is byte-identical to the resolved `interactive_command` — a diff against the previous `WindowCommand` output shows only the removed pointer argument
- [x] A-019 R5: the previously documented `--pane`/`--timeout` mutual-exclusion error is gone from `start` along with the flag, and no path silently reinterprets `--pane`
- [x] A-020 R7: the pane three-state derivation and the five-state machine behave exactly as before; `delivered` changes no state anywhere
- [x] A-034 R15: the deferred apply-pane reap call at review-pass runs ONLY when a pane-mode dispatch record exists for apply (the pane arm); the native and headless arms run no reap call there, and no skill prose claims `fab dispatch reap` no-ops on a missing record (it errors — `_cli-fab.md` § reap, TestDispatchReap_NoDispatchErrors)

### Removal Verification

- [x] A-021 R1: `PointerPrompt` is no longer reachable from the spawn path, and no test or doc still asserts a pointer inside the composed window command
- [x] A-022 R13: no remaining prose in `src/kit/`, `docs/specs/`, or `internal/` claims a provider lacks `interactive_command` *because* it cannot accept a positional prompt

### Scenario Coverage

- [x] A-023 R2: scripted-capture tests cover echo → `ready`, no-echo + stable screen → `parked`, and no-echo + changing screen → `booting`
- [x] A-024 R3: tests cover deliver success, first-attempt-fails-then-retry-succeeds, and both-attempts-fail
- [x] A-025 R18: `go test ./internal/dispatch/... ./internal/pane/... ./cmd/fab/...` passes, and the full Go suite is green

### Edge Cases & Error Handling

- [x] A-026 R2: `ready` against a headless record, a missing record, or a dead pane exits non-zero with an actionable message rather than reporting a classification
- [x] A-027 R3: a delivery that fails both attempts exits non-zero, surfaces the capture snippet, and leaves `delivered` unset so the orchestrator can tell delivery from execution failure
- [x] A-028 R14: gate exhaustion escalates with capture evidence and a fail-silent `rk notify`, leaves the pane alive, and never descends to headless

### Code Quality

- [x] A-029 Pattern consistency: new Go code follows the package's pure-decision + thin-wrapper split, named constants over magic strings, and the existing error/warning conventions (`pane.StderrError`, stderr warnings for non-fatal degradation)
- [x] A-030 No unnecessary duplication: the gate and delivery paths reuse `internal/pane`'s tmux helpers and `internal/dispatch`'s existing loaders/derivations instead of reimplementing them
- [x] A-031 Owner-or-pointer: no skill file both states a rule it does not own and points at its owner — the gate procedure is owned by `_preamble.md` § CLI-Adapter Dispatch and referenced by pointer elsewhere
- [x] A-032 Sibling & mirror sweeps: every edited `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated, and the CLI-signature change updates `_cli-fab.md` plus tests, per the constitution
- [x] A-033 No `.claude/skills/` file was edited directly; all skill edits are in `src/kit/skills/`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

- 2026-08-09 live probe (orchestrator session, isolated tmux sockets): the review-c4 frame-tolerance concern does not materialize — claude @50 cols, claude @30 cols (mid-word wrap `ap`/`ply-prompt.md`), and agy @50 cols all render the typed pointer with countWrapped == 1 under the exact gate.go squeeze logic; both TUIs use horizontal-rule input boxes with no side borders. Closed with evidence, no verification redesign; kimi is covered by backlog [agik]'s pre-shipping echo probe.

## Deletion Candidates

- `src/go/fab/cmd/fab/pane_capture.go` `capturePaneArgs` / `capturePaneContent` — now one-line pass-throughs to `pane.CaptureArgs` / `pane.Capture`, retained only because `pane_capture_test.go` names them; inline the `internal/pane` calls and retarget the tests
- `src/go/fab/cmd/fab/pane_send.go` `sendTextArgs` / `sendEnterArgs` — same shape: pure delegations to `pane.SendLiteralArgs` / `pane.SendKeyArgs` kept alive by their own tests
- `src/go/fab/cmd/fab/dispatch_start.go` hidden `--pane` / `--server` flags on `start` (+ `paneFlagRetiredError`) — deliberately retained this release so the retired spelling answers with the `open` route; deletable once that guidance has shipped for a version
- `src/go/fab/internal/dispatch/dispatch.go` `shellQuote` is **NOT** a candidate — `WindowCommand` was its pane caller, but the headless `sh -c` wrapper still uses it

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `open` forces pane mode and hard-errors on a missing prerequisite rather than descending | The intake fixes `open` as "pane mode's entry"; a descending `open` would silently become a headless launcher under a different verb name | S:90 R:80 A:90 D:90 |
| 2 | Confident | `start` keeps `--pane`/`--server` registered but **hidden**, with a pre-work RunE guard raising the `fab dispatch open` guidance | The intake requires an error "with guidance pointing at `open`", which a removed flag cannot give (cobra would emit a bare `unknown flag`); hiding keeps the help surface clean | S:75 R:90 A:85 D:75 |
| 3 | Confident | `deliver` clears the stale `{stage}-result.yaml` and `{stage}.exit` **before** the first send attempt | Without it a continuation delivery leaves the dispatch reading `done`, so `wait` returns immediately on the previous cycle's result; clearing before the send matches the launch path's stale-clearing and keeps the record honest once keys have been typed | S:80 R:75 A:85 D:80 |
| 4 | Confident | `deliver` refuses when the record is `delivered: true` and the state is `running` | Encodes the contract's post-delivery no-input-injection rule in code at the only place that could violate it; the sanctioned continuation case (`done`) is unaffected | S:80 R:80 A:85 D:80 |
| 5 | Confident | The continuation prompt is written to `.fab-dispatch/{id}/{stage}-continuation.md` | Keeps the resume prompt inside the dir `clean`/archive already own, and the record walker ignores non-`.yaml` files, so no lifecycle work is needed | S:70 R:90 A:85 D:70 |
| 6 | Confident | All three `ready` classifications exit 0; the report string is the sole discriminator | Mirrors `fab dispatch wait`'s documented timeout-is-not-an-error rule, so the family has one convention for "a probe answered" | S:80 R:85 A:90 D:80 |
| 7 | Confident | Non-`ready` reports carry `pane:` and (when non-empty) `server:` lines above the snippet | The judgment rounds send raw `tmux send-keys`, which needs both the pane id and the socket; `status --json` exposes the pane but not the socket, so the probe is the natural carrier | S:70 R:90 A:85 D:70 |
| 8 | Confident | The bounded boot allowance is **5 consecutive `booting` probes**, after which the pane is treated as `parked` | The intake fixes only that the allowance is bounded; five probes at the wiring's brief wait is comfortably longer than any observed TUI cold start and still terminates | S:60 R:95 A:75 D:65 |
| 9 | Confident | The tmux surface used by the gate is an interface with a test fake, and the inter-capture delay is a struct field | The package's existing tests only exercise pure functions; the verify-retry choreography cannot be covered that way, and the intake requires choreography tests | S:75 R:90 A:90 D:80 |
| 10 | Tentative | Probe mechanics: sentinel `FAB-READY-PROBE`, 500 ms inter-capture tick, 300 ms echo settle, 800 ms busy settle, 20-line snippet | Pure implementation detail, explicitly graded Tentative in the intake; all are named package constants so a field revision is a one-line change | S:50 R:95 A:70 D:55 |
| 11 | Tentative | Report strings: `opened <id>/<stage> (<identity>)` and `delivered <id>/<stage> (pane %N, prompt <path>)`, extending the `dispatched …` family | The intake defers exact wording to apply; these mirror the existing line's shape verbatim and are pinned by tests | S:55 R:90 A:80 D:60 |
| 12 | Tentative | `restart`'s pane hand-back note goes to **stderr** while stdout keeps the pinned `opened …` line | Keeps stdout machine-readable and matches the family's existing use of stderr for `dispatch selection:` and placement warnings | S:60 R:90 A:80 D:65 |

12 assumptions (1 certain, 8 confident, 3 tentative).
