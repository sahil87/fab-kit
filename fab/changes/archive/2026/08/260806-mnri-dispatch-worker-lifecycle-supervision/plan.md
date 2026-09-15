# Plan: Dispatch Worker Lifecycle Supervision

**Change**: 260806-mnri-dispatch-worker-lifecycle-supervision
**Intake**: `intake.md`

## Requirements

### Go CLI: `fab dispatch restart`

#### R1: A `restart` subcommand relaunches a non-running dispatch from the persisted prompt
`fab dispatch restart <change> <stage>` SHALL relaunch a stage dispatch by reading the stage prompt from `.fab-dispatch/{id}/{stage}-prompt.md` instead of stdin. It SHALL accept the same mode flags as `start` (`--pane`, `--headless`, `--timeout`, `--server`) with the same mutual exclusions, and SHALL run `start`'s prologue unchanged: change resolution, config + tier→provider resolution, pane validation, refuse-if-running, stale-file clearing, and the mode-selection ladder.

- **GIVEN** a completed (`done`/`failed`/`orphaned`) dispatch for `abcd/apply` whose `apply-prompt.md` is on disk
- **WHEN** the operator runs `fab dispatch restart abcd apply`
- **THEN** a new worker is launched with the persisted prompt as its input
- **AND** the persisted prompt file is left byte-identical (it is the restart's source, not a fresh write)
- **AND** stdout carries a `dispatched abcd/apply (…)` line shaped exactly like `start`'s

#### R2: A missing prompt file is a clear, actionable error
`restart` MUST error when `.fab-dispatch/{id}/{stage}-prompt.md` is absent, naming the change/stage and pointing at `fab dispatch start` with a prompt on stdin. Nothing SHALL be launched and no state written.

- **GIVEN** no `apply-prompt.md` in `.fab-dispatch/abcd/`
- **WHEN** the operator runs `fab dispatch restart abcd apply`
- **THEN** the command exits non-zero with a message naming the missing prompt and the `fab dispatch start` remedy
- **AND** no `apply.yaml` record exists afterwards

#### R3: `restart` refuses a genuinely running dispatch
`restart` SHALL apply the same refuse-if-running check `start` applies — the **prior** record's own mode's finished signal (headless: `{stage}.exit` absent AND pid alive; pane: `{stage}-result.yaml` absent AND pane alive) — and MUST point at `fab dispatch kill`.

- **GIVEN** a live headless dispatch for `abcd/apply` (pid alive, no exit file)
- **WHEN** the operator runs `fab dispatch restart abcd apply`
- **THEN** the command errors with the same `already running (…); run \`fab dispatch kill\` first` refusal `start` raises
- **AND** the existing record is left untouched

#### R4: Mode is re-derived from the current environment, not inherited from the prior attempt
`restart` SHALL resolve its launch mode through the same explicit-first ladder ending in auto (`$TMUX` set ⇒ pane, unset ⇒ headless), with the same explicit/auto asymmetry on pane-prerequisite failures (hard error when explicit, soft fallback to headless when auto). The prior attempt's mode SHALL have no influence.

- **GIVEN** an `orphaned` **pane** dispatch for `abcd/apply` and no reachable tmux server (`$TMUX` unset)
- **WHEN** the operator runs `fab dispatch restart abcd apply`
- **THEN** auto selects **headless** and the new record is headless-shaped (pid/pgid, no pane identity)
- **AND** GIVEN `$TMUX` set and a reachable server with a provider `session_command`, the same command instead opens a pane dispatch

#### R5: `restart`'s output and record are byte-shaped like `start`'s
`restart` SHALL emit the same `dispatched <id>/<stage> (<report>)` line — including the `, auto: …` selection-source suffix rules — and persist the same `{stage}.yaml` shape. It SHALL introduce no new state string, no attempt counter, no attempt history, and no `restarted:` marker.

- **GIVEN** a restart that auto-selects headless outside tmux
- **WHEN** it succeeds
- **THEN** stdout is `dispatched abcd/apply (pid N, pgid N, auto: no tmux)` — indistinguishable from the equivalent `start`
- **AND** `apply.yaml` carries only the keys a `start` record carries

#### R6: `start` and `restart` share one launch path
The prompt-acquisition seam in `runDispatchStart` SHALL be refactored so `start` (stdin) and `restart` (state-dir file) differ **only** in how the prompt bytes are obtained, sharing one prologue, one launch, one save, and one report. There SHALL be no duplicated launch tail.

- **GIVEN** the two subcommands
- **WHEN** either runs
- **THEN** both execute the same resolution → validate → refuse → persist → launch → save → report sequence
- **AND** a change to that sequence takes effect for both without a second edit

### Skill wiring: poll-loop recovery policy

#### R7: `orphaned` gets one automatic restart per stage dispatch, then escalates
`_preamble.md` § CLI-Adapter Dispatch's five-state handling SHALL replace the unconditional "surface and stop" row for `orphaned` with: run `fab dispatch restart <change> <stage>` once (the single per-stage-dispatch budget) and resume polling. If the restarted attempt also fails to reach `done`, the orchestrator SHALL escalate — surface the evidence per mode (`fab dispatch logs --tail N` headless / `fab pane capture [-L <server>] <pane>` pane), send a gated `rk notify` (`command -v rk`, fail-silent per the rk universal rule), and stop per the stage's existing failure path.

- **GIVEN** a CLI-dispatched apply stage whose worker died with no recorded exit
- **WHEN** the orchestrator's poll observes `orphaned` with the restart budget unspent
- **THEN** it restarts once and keeps polling
- **AND** WHEN a second `orphaned` (or a `failed`) is observed with the budget spent, it surfaces evidence, notifies, and stops

#### R8: `failed` gets no automatic restart; `failed (no-result)` always escalates
`failed` SHALL carry **no automatic** restart (a deterministic failure — bad config, a real test failure, a `124` timeout — would loop). The orchestrator MAY judge a clearly-transient failure from the log tail (provider 5xx/overload signatures) worth one restart, spending the **same single** budget; anything else stops as today. `failed (no-result)` is a contract violation and SHALL always escalate, never restart.

- **GIVEN** a `failed` dispatch whose log tail shows a provider overload signature and an unspent budget
- **WHEN** the orchestrator classifies it as transient
- **THEN** it MAY restart once, spending the shared budget
- **AND** GIVEN `failed (no-result)`, it escalates unconditionally regardless of budget

#### R9: Peek-on-suspicion classifies a result-less dispatch three ways
Every 10th result-less poll (~5 min at the fixed 30s cadence) the orchestrator SHALL take a **read-only** peek — `fab pane capture [-L <server>] <pane>` (pane) or `fab dispatch logs --tail 40` (headless) — and classify: (a) visibly progressing ⇒ keep polling; (b) parked at an error banner / dead-ended ⇒ `fab dispatch kill` then restart within the same budget (escalate if spent); (c) waiting on genuine human input ⇒ escalate via gated `rk notify` **without killing**.

- **GIVEN** a pane dispatch that has been `running` for 10 polls with no result file
- **WHEN** the orchestrator peeks and sees a permission prompt awaiting an answer
- **THEN** it notifies a human and neither kills nor restarts nor types into the pane
- **AND** GIVEN the peek instead shows a dead-ended error banner, it kills and restarts within the single budget

#### R10: The pipeline never sends keys to a worker
No pipeline path SHALL send keystrokes to a dispatched worker. The pipeline's verbs are exactly: peek (read-only), kill, restart, notify, stop. Nudging and answering remain the operator's and the user's affordances.

- **GIVEN** any recovery classification
- **WHEN** the orchestrator acts
- **THEN** it never issues `send-keys` or any other input-injection command against the worker

#### R11: The restart budget lives in orchestrator context, not on disk
The single-restart budget SHALL be tracked per stage dispatch in the orchestrator's own context. No attempt counter or history SHALL be added to `{stage}.yaml` — last-attempt-only is preserved. The documented worst case after orchestrator context loss is one extra restart.

- **GIVEN** a stage dispatch that has consumed its restart
- **WHEN** the orchestrator's context is lost and rebuilt
- **THEN** at most one extra restart may occur — no on-disk counter exists to consult
- **AND** `{stage}.yaml` carries no attempt/restart key

#### R12: Pane mode's three-state subset carries the same recovery policy
The pane-subset row SHALL state that `orphaned` there also gets the one automatic restart, that `failed`/`failed (no-result)` remain unreachable rather than newly-handled, and that evidence surfacing on the pane path uses `fab pane capture` (socket-included when the dispatch used `--server`) rather than `fab dispatch logs`.

- **GIVEN** an `orphaned` pane dispatch
- **WHEN** the orchestrator recovers it
- **THEN** it restarts once — mode re-derived, so a dead tmux server soft-falls-back to headless
- **AND** the evidence it surfaces on escalation is a pane capture, not a log tail

### Documentation, spec, and memory

#### R13: `_cli-fab.md` documents `restart` and the updated family headline
`src/kit/skills/_cli-fab.md` § fab dispatch SHALL gain a `### restart` subsection (signature, prompt-file source, refuse-if-running, flag set, shared prologue) and SHALL update every family enumeration to `start`/`restart`/`status`/`logs`/`kill`/`clean`.

- **GIVEN** the new subcommand
- **WHEN** an agent reads `_cli-fab.md` § fab dispatch
- **THEN** the family headline and the per-subcommand reference both name `restart`

#### R14: `harness-adapters.md` gains a recovery/supervision paragraph
`docs/specs/harness-adapters.md` SHALL document recovery as **orchestrator policy over the existing states** in its shared-protocol section — the states, the result-file contract, and the prompt obligations all unchanged.

- **GIVEN** the spec's five-state machine
- **WHEN** the recovery paragraph is added
- **THEN** no state is added, renamed, or re-tabled — the paragraph only says what an orchestrator does with `orphaned`/`failed`/never-`done`

#### R15: The mirror class is swept whole
Every touched `src/kit/skills/*.md` SHALL have its `docs/specs/skills/SPEC-*.md` mirror updated in the same change, and the sibling dispatch-seam surfaces (`_pipeline.md`, `fab-continue.md`, `fab-adopt.md` + their mirrors) SHALL be grep-swept for restated five-state "surface and stop" rows and corrected or left correct by delegation.

- **GIVEN** an edit to `_cli-fab.md` and `_preamble.md`
- **WHEN** apply finishes
- **THEN** `SPEC-_cli-fab.md` and `SPEC-_preamble.md` carry the same facts
- **AND** no sibling surface restates a now-stale unconditional "surface and stop"

#### R16: Memory records the shipped behavior
`docs/memory/runtime/dispatch.md` SHALL document the `restart` requirement, the shared prompt-acquisition seam, and the recovery-policy pointer; `docs/memory/_shared/context-loading.md`'s CLI-Adapter Dispatch mirror SHALL carry the bounded-recovery policy; `docs/memory/pipeline/execution-skills.md` SHALL be updated only if it restates the poll-loop/five-state handling (grep-verified).

- **GIVEN** the shipped change
- **WHEN** the memory files are read
- **THEN** they describe restart + bounded recovery as present truth, with no transition narration

### Non-Goals

- No orchestrator→worker send-keys/nudge channel — operator territory; it would fork the cross-adapter contract.
- No supervisor daemon, timer, or background sweep — polling remains the only clock.
- No on-disk attempt history or counter — last-attempt-only preserved.
- No new dispatch states and no change to the five/three-state machines, the result-file contract, or the prompt obligations.
- No auto-answer of worker prompts — classification (c) notifies a human, never types.
- No `--force` / `--kill-first` convenience flag on `restart` — killing first is `fab dispatch kill`, explicitly.

### Design Decisions

#### Restart, not nudge, is tier 2's recovery verb
**Decision**: The pipeline recovers a stuck stage worker by kill+restart, never by sending keystrokes. Stage state lives in artifacts (plan.md task checkboxes, the result-file contract, idempotent stages), so a restarted worker resumes from the last `[x]`.
**Why**: A dispatch is not a session — it carries no irreplaceable conversational state, so restart is deterministic and cheap. Send-keys delivery is documented-flaky (the printed-prompt trap, probe-then-retype choreography) and a pipeline nudge channel would fork the cross-adapter contract (native workers have no such channel).
**Rejected**: An orchestrator→worker send-keys channel — fragile TUI delivery, contract fork, and a duplicate of the operator's job.
*Introduced by*: 260806-mnri-dispatch-worker-lifecycle-supervision

#### `restart` reads the persisted prompt file rather than stdin
**Decision**: `restart` acquires its prompt from `.fab-dispatch/{id}/{stage}-prompt.md`; a missing file is a clear error naming `fab dispatch start`.
**Why**: The orchestrator that needs to restart may have lost the multi-thousand-token block prompt to compaction. The prompt is already persisted for both modes (headless stdin redirect, pane pointer target), so the file is an existing, authoritative source — restart adds a source, not a mechanism.
**Rejected**: Requiring the prompt on stdin again (defeats the purpose — the caller may no longer have it); persisting a second copy under a restart-specific name (a second source of truth for the same bytes).
*Introduced by*: 260806-mnri-dispatch-worker-lifecycle-supervision

#### The restart budget lives in orchestrator context, not on disk
**Decision**: Exactly one automatic restart per stage dispatch, tracked in the orchestrator's context. No attempt counter is added to `{stage}.yaml`.
**Why**: Last-attempt-only (no per-attempt history) is a shipped design decision; an on-disk counter would be a second source of truth the state machine does not need. The worst case after orchestrator context loss is one extra restart, which is benign.
**Rejected**: A `restarts:` key in `{stage}.yaml` (breaks last-attempt-only, and would need its own reset semantics); an unbounded restart loop (burns tokens against a platform-wide outage).
*Introduced by*: 260806-mnri-dispatch-worker-lifecycle-supervision

#### Mode is re-derived on every restart
**Decision**: A restart resolves its mode through the same ladder from the *current* environment; the prior attempt's mode has no influence.
**Why**: A restart is a fresh attempt under the existing last-attempt-only semantics, and the environment is exactly what changed when a worker died — restarting an orphaned pane dispatch after a tmux server death must correctly soft-fall-back to headless. Inheriting the prior mode would reproduce the failure.
**Rejected**: Persisting and reusing the prior mode (would re-fail on the condition that killed the worker); a `--same-mode` flag (no demonstrated need, and it would need the stored discriminator the record deliberately lacks).
*Introduced by*: 260806-mnri-dispatch-worker-lifecycle-supervision

#### `failed` is orchestrator judgment, not an automatic rule
**Decision**: `orphaned` auto-restarts; `failed` does not, but the orchestrator MAY spend the same single budget on a clearly-transient failure read from the log tail. `failed (no-result)` always escalates.
**Why**: `orphaned` means "no exit code was ever recorded" — a death, which is transient by nature. `failed` carries a real exit code and is usually deterministic (bad config, a genuine test failure, `124`), so an automatic rule would either loop or under-recover. The orchestrator is an agent that can read a log tail, so encoding judgment beats encoding a bad rule. A contract violation needs eyes, not retries.
**Rejected**: Auto-restarting `failed` too (loops on deterministic failures); a provider-5xx pattern matcher in Go (fab would own a rotting signature list for every provider).
*Introduced by*: 260806-mnri-dispatch-worker-lifecycle-supervision

## Tasks

### Phase 1: Go — shared launch path

- [x] T001 Refactor the prompt-acquisition seam in `src/go/fab/cmd/fab/dispatch_start.go`: extract a prompt-source abstraction so `runDispatchStart` takes the prompt bytes (or a source func) rather than reading stdin inline, keeping the prologue/launch/save/report tail single-sourced <!-- R6 -->
- [x] T002 Add `src/go/fab/cmd/fab/dispatch_restart.go` with `dispatchRestartCmd()` — same flag set and exclusions as `start`, `Args: cobra.ExactArgs(2)`, same `dispatch.SelectMode` ladder call, prompt read from `dispatch.PromptPath`, with a clear missing-prompt error <!-- R1 R2 R4 -->
- [x] T003 Register `dispatchRestartCmd()` in `src/go/fab/cmd/fab/dispatch.go`'s `AddCommand` and update the family enumerations in that file's doc comment, `Use`/`Short`/`Long` strings <!-- R1 R13 -->

### Phase 2: Go — tests

- [x] T004 Add `src/go/fab/cmd/fab/dispatch_restart_test.go` covering: restart-over-orphaned relaunches from the persisted prompt (prompt file unchanged, new record, stale exit/result cleared), refuse-while-running, missing-prompt-file (no state written), and output byte-shape parity with `start` <!-- R1 R2 R3 R5 -->
- [x] T005 Add mode-re-derivation tests to `dispatch_restart_test.go`: an orphaned pane record restarted with `$TMUX` unset produces a headless record; `--headless`/`--pane`/`--timeout`/`--server` behave as on `start` (incl. the `--pane`+`--timeout` and `--pane`+`--headless` usage errors persisting nothing) <!-- R4 R5 -->
- [x] T006 Run `go test ./...` in `src/go/fab` and fix failures <!-- R1 R6 -->

### Phase 3: Skill wiring + docs/spec

- [x] T007 Update `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch: replace the unconditional `orphaned` "surface and stop" row with the bounded-recovery policy, add the `failed` / `failed (no-result)` policy, the peek-on-suspicion three-way classification, the no-send-keys rule, the budget-in-context rule, and extend the pane-mode subset bullet <!-- R7 R8 R9 R10 R11 R12 -->
- [x] T008 Update `src/kit/skills/_cli-fab.md` § fab dispatch: add the `### restart` subsection, update the family headline/enumerations, and note the shared prologue <!-- R13 -->
- [x] T009 Add the recovery/supervision paragraph to `docs/specs/harness-adapters.md`'s shared-protocol section (states unchanged; recovery is orchestrator policy) <!-- R14 -->
- [x] T010 Update the SPEC mirrors `docs/specs/skills/SPEC-_cli-fab.md` and `docs/specs/skills/SPEC-_preamble.md` <!-- R15 -->
- [x] T011 Grep-sweep the sibling dispatch-seam surfaces (`src/kit/skills/_pipeline.md`, `fab-continue.md`, `fab-adopt.md` and their SPEC mirrors) for restated five-state / "surface and stop" rows and update or confirm-by-delegation each <!-- R15 -->

### Phase 4: Memory

- [x] T012 Update `docs/memory/runtime/dispatch.md`: the `restart` requirement + scenarios, the shared prompt-acquisition seam, the recovery-policy pointer, and the family enumeration in the frontmatter description/overview <!-- R16 -->
- [x] T013 Update `docs/memory/_shared/context-loading.md`'s CLI-Adapter Dispatch mirror with the bounded-recovery policy <!-- R16 -->
- [x] T014 Grep-verify `docs/memory/pipeline/execution-skills.md` for poll-loop/five-state restatements; update only if present <!-- R16 -->

### Phase 5: Rework (review cycle 1)

- [x] T015 Add `restart` to the `fab dispatch` family enumeration in `docs/site/skill.md` (line 38 — the canonical `shll skill fab-kit` bundle), then run `scripts/sync-skill.sh` to regenerate the embedded copy `src/go/fab/cmd/fab/skill.md`; verify the `skill_test.go` drift guard passes <!-- R13 --> <!-- rework: cycle-1 must-fix — family-enumeration sweep missed the binary-embedded skill bundle twin -->
- [x] T016 Fix the stale present-truth symbol claim at `docs/memory/runtime/dispatch.md:75` — `runDispatchStart` no longer "calls it once in the cobra RunE"; restate against the shipped `runDispatchLaunch` seam. Grep the memory tree for other stale `runDispatchStart` present-truth claims (the Design-Decision "Rejected" line near :437 is as-decided historical wording and may stay) <!-- R16 --> <!-- rework: cycle-1 must-fix — stale pkg.Symbol claim -->
- [x] T017 Extract a shared flag-surface helper (e.g. `addLaunchFlags`) in `src/go/fab/cmd/fab/dispatch_start.go`, used by both `dispatchStartCmd` and `dispatchRestartCmd` — the four flag registrations, the `--pane`+`--timeout` guard, and `MarkFlagsMutuallyExclusive` — removing the ~14-line duplicate in `dispatch_restart.go`; re-run `go test ./...` in `src/go/fab` <!-- R6 --> <!-- rework: cycle-1 should-fix — flag surface was the last drift seam between start and restart -->
- [x] T018 Run `fab memory-index` then `fab memory-index --check` after all rework edits — confirm the regenerated indexes (`docs/memory/runtime/index.md`, `docs/memory/_shared/index.md`, left regenerated in the working tree by review) are current and check-clean <!-- R16 --> <!-- rework: cycle-1 should-fix — generated indexes were stale after hand-edits -->

## Execution Order

- T001 blocks T002 (restart reuses the extracted seam); T002 blocks T003 and T004/T005
- T006 runs after T004/T005
- T007–T011 are independent of the Go phases but T008 depends on the final Go flag set (T002)
- T012–T014 run last (they record shipped behavior)
- T015–T017 are independent; T018 runs after T015/T016 (it verifies the tree after all rework edits)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab dispatch restart <change> <stage>` exists, relaunches from `.fab-dispatch/{id}/{stage}-prompt.md`, and accepts `--pane`/`--headless`/`--timeout`/`--server`
- [x] A-002 R2: A missing prompt file produces an actionable error naming `fab dispatch start`, with nothing launched or persisted — *the error names the full prompt PATH (which embeds the change id + stage) rather than the bare `<change>/<stage>` pair R2 / Assumption 4 sketched; semantically equivalent, and the docs quote the shipped text verbatim*
- [x] A-003 R3: `restart` refuses a genuinely running dispatch with the same mode-aware finished-signal check `start` uses, pointing at `fab dispatch kill`
- [x] A-004 R4: `restart` re-derives its mode from the current environment through the same ladder, with the same explicit-hard / auto-soft asymmetry
- [x] A-005 R5: `restart`'s stdout line and `{stage}.yaml` record are shaped exactly like `start`'s — no new state string, counter, or marker
- [x] A-006 R6: `start` and `restart` share one launch path; the only difference is prompt acquisition
- [x] A-007 R7: `_preamble.md`'s `orphaned` row specifies one automatic restart then escalation (evidence + gated `rk notify` + the stage's failure path)
- [x] A-008 R8: `failed` carries no automatic restart but an orchestrator-judgment carve-out inside the same single budget; `failed (no-result)` always escalates
- [x] A-009 R9: Peek-on-suspicion is documented with its cadence (every 10th result-less poll), its per-mode read-only command, and all three classifications
- [x] A-010 R10: The no-send-keys-from-the-pipeline rule is stated, with the pipeline's verb set enumerated
- [x] A-011 R11: The budget is documented as orchestrator-context-only, with the one-extra-restart worst case named and no `{stage}.yaml` key added
- [x] A-012 R12: The pane-mode subset row carries the same recovery policy and names `fab pane capture` (socket-included) as its evidence surface
- [x] A-013 R13: `_cli-fab.md` documents `restart` and every family enumeration reads `start`/`restart`/`status`/`logs`/`kill`/`clean` — *re-verified after rework T015: the sweep is now repo-wide complete. `_cli-fab.md:513` headline + `### restart` subsection (:580-589), `docs/site/skill.md:38` and its embedded twin `src/go/fab/cmd/fab/skill.md:38` (byte-identical, `skill_test.go` drift guard passes), plus `dispatch.go` doc comment/`Long`, `internal/dispatch/dispatch.go` package doc, `runtime/dispatch.md` frontmatter + :30 ("six subcommands") + `runtime/index.md`, `distribution/kit-architecture.md:312`, and `SPEC-_cli-fab.md`. A grep for any residual `{start,status,logs,kill,clean}` / `start|status|logs|kill|clean` / `start/status/logs/kill/clean` spelling returns zero hits*
- [x] A-014 R14: `harness-adapters.md` carries the recovery/supervision paragraph with the state tables untouched
- [x] A-015 R15: `SPEC-_cli-fab.md` and `SPEC-_preamble.md` are updated, and the sibling dispatch-seam surfaces carry no stale unconditional "surface and stop"
- [x] A-016 R16: `runtime/dispatch.md` and `_shared/context-loading.md` document restart + bounded recovery as present truth; `pipeline/execution-skills.md` grep-verified — *re-verified after rework T016: `runtime/dispatch.md:75` now reads against the shipped shape ("called once per invocation from `launchFlags.resolveMode` … which hands the resolved mode … to `runDispatchLaunch`"), matching `dispatch_start.go:64`/`:166`. A repo-wide grep for `runDispatchStart` leaves exactly one hit — `dispatch.md:437`, inside an l9ng-owned Design Decision's `**Rejected**:` line ("An inline `if` chain in `runDispatchStart`"), which is as-decided historical wording about what l9ng rejected at the time, not a present-truth claim; the FKF rule permits it. The new `### Requirement: fab dispatch restart` block (:232-269) carries four scenarios, and `context-loading.md` § Per-Stage Model Resolution carries the six recovery bullets*

### Behavioral Correctness

- [x] A-017 R6: The refactor is byte-preserving for `start` — every pre-existing `dispatch_start_test.go` case still passes unchanged
- [x] A-018 R5: A restart's `dispatched …` line carries the `auto: …` suffix under auto selection and no suffix under an explicit mode, matching `start` exactly
- [x] A-019 R4: Restarting an orphaned **pane** attempt outside tmux produces a **headless** record — the prior mode is not inherited

### Scenario Coverage

- [x] A-020 R1: A test drives restart over an orphaned attempt and asserts the persisted prompt was the input and the prompt file was not rewritten
- [x] A-021 R3: A test asserts the refuse-while-running path leaves the prior record untouched
- [x] A-022 R2: A test asserts the missing-prompt-file error writes no `{stage}.yaml`
- [x] A-023 R4: A test asserts mode re-derivation across a mode change (pane record → headless restart)

### Edge Cases & Error Handling

- [x] A-024 R2: `restart` on a change/stage with no dispatch dir at all errors on the missing prompt rather than panicking or half-launching
- [x] A-025 R4: `--pane` + `--timeout` and `--pane` + `--headless` are usage errors on `restart` too, persisting nothing
- [x] A-026 R8: `failed (no-result)` is documented as never-restarted, so a contract-violating worker cannot be looped

### Code Quality

- [x] A-027 Pattern consistency: New code follows the naming and structural patterns of `dispatch_start.go` / the `dispatch_*.go` cmd wrappers (cobra command constructor + a `run…` function, doc comments explaining the load-bearing decisions)
- [x] A-028 No unnecessary duplication: `restart` reuses `runDispatchStart`'s prologue/launch/save/report rather than copying it; the refuse-if-running, validatePane, and modeCommand helpers are shared — *the launch path itself is genuinely single-sourced (`runDispatchLaunch`), but the cobra FLAG SURFACE is duplicated verbatim across the two constructors: 4 identical `Flags()` registrations, the `--pane`+`--timeout` guard, `MarkFlagsMutuallyExclusive`, and the `SelectMode` call (~14 lines). See the should-fix finding — a shared `addLaunchFlags` helper would close the same drift seam R6 closed for the tail*
- [x] A-029 No god functions: the shared launch function stays focused; the prompt-source split does not grow it past the file's existing function scale
- [x] A-030 No editing `.claude/skills/` directly: every skill edit lands in `src/kit/skills/*.md`
- [x] A-031 SPEC mirror shipped: every touched `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated in this change
- [x] A-032 CLI change documented + tested: the new command signature is in `src/kit/skills/_cli-fab.md` and covered by `_test.go` cases
- [x] A-033 Sibling & mirror sweep done up front: the whole dispatch-seam mirror class was swept, not reactively patched — *re-verified after rework T015/T018: the class is now complete. 5 skill↔SPEC pairs (`_cli-fab`, `_preamble`, `_pipeline`, `fab-continue`, `fab-adopt`); the `skill` bundle twin `docs/site/skill.md` ↔ `src/go/fab/cmd/fab/skill.md` is byte-identical (`diff` clean, `skill_test.go` drift guard green); the generated memory indexes are regenerated and `fab memory-index --check` exits 0 (remaining output is pre-existing repo-wide size/narration debt on files this change did not create). Recorded as met on the shipped tree — the "up front" clause was satisfied only after a rework cycle, which is the process cost the finding already charged*
- [x] A-034 Test strategy: tests ship in this change (test-alongside) and the affected Go packages were run

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab/cmd/fab/dispatch_start.go:365-374` (`modeCommand`'s `mode == ModePane` arm) — the pane branch's `session_command == ""` error stays unreachable through **both** entry points: `validatePane` (:333) diagnoses a missing `session_command` before composition on every pane path, for `start` and `restart` alike, raising the identical `missingCommandError`. The function's own doc comment (:354-358) already argues for keeping it as a local-contract guard ("a mode never composes an empty command"), which is a defensible call — recorded so the redundancy is a decision rather than an oversight. *(Line numbers refreshed after T017 shifted the file by ~50 lines.)*
- ~~`src/go/fab/cmd/fab/dispatch_restart.go` — the duplicated `--pane`/`--timeout` guard and the four flag registrations~~ — **collapsed in T017**: `addLaunchFlags(cmd) *launchFlags` (:38) + `(*launchFlags).resolveMode` (:64) in `dispatch_start.go` now own the four registrations, the `--pane`+`--timeout` guard, the `MarkFlagsMutuallyExclusive` group, and the `SelectMode` call for both constructors. Per-command `Use`/`Short`/`Long`/`Example` stay local (the prompt-source difference is what they exist to explain); help output and usage-error text verified byte-identical. **Verified deleted** — `dispatch_restart.go` is 80 lines and registers no flags of its own.
- `src/kit/skills/_preamble.md:348` — the parenthetical **"(orphan detection + `fab dispatch kill` cover the failure modes)"** on the Step 1 no-`--timeout` rule is a *rationale this change refutes*: the § Recovery policy three lines below exists precisely because orphan detection and kill do NOT cover a post-retry-exhaustion death or a worker wedged at `running`. Deletion candidate = the parenthetical only (the no-`--timeout` guidance itself may still stand on its own). See the should-fix finding.
- Nothing else — the change is otherwise additive (one new subcommand plus a parameterized seam and a flag-surface helper on an existing file); no existing file, function, branch, config key, dispatch state, or flag became unused. `internal/dispatch` gained comment-only edits and no new record key (`Dispatch` struct unchanged), so last-attempt-only leaves nothing orphaned.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The prompt-acquisition seam is refactored by threading the prompt BYTES into the shared launch function (a `[]byte` parameter), rather than passing an `io.Reader` or a source-strategy interface | The two sources differ only in where the bytes come from and both are already fully read into memory before the launch (`io.ReadAll` on stdin today); bytes are the smallest seam that keeps the prologue single-sourced | S:80 R:90 A:90 D:80 |
| 2 | Certain | `restart` does NOT rewrite `{stage}-prompt.md` — it reads it and leaves it byte-identical, while still clearing the stale exit/result/log files like `start` does | The prompt IS the restart's input, so rewriting it with its own content is a no-op that only risks corruption on a partial write; the stale-file clearing is about the PREVIOUS attempt's output, which must not contaminate the new run's status | S:80 R:85 A:90 D:85 |
| 3 | Confident | `restart` reads the prompt file AFTER the refuse-if-running check (mirroring `start`'s order: prologue → refuse → prompt), so a running dispatch refuses even when the prompt file is missing | Preserves `start`'s ordering exactly, which the shared-path requirement demands; and the running-dispatch refusal is the more actionable of the two errors | S:65 R:85 A:85 D:70 |
| 4 | Confident | The missing-prompt error text is `no persisted prompt for <change>/<stage> at <path> — nothing to relaunch; run \`fab dispatch start <change> <stage>\` with the prompt on stdin` | The intake specifies the semantics ("nothing to relaunch — run `fab dispatch start` with a prompt on stdin"); the exact wording follows the family's actionable-error style (name the thing, name the remedy) | S:70 R:90 A:85 D:70 |
| 5 | Confident | `restart` gets its own cobra `Example` block and `Short` line rather than sharing `start`'s, since the two differ in prompt source and intended use (recovery vs. launch) | Every other `fab dispatch` subcommand carries its own help text; sharing would make the family's help inconsistent and hide the prompt-source difference that is the whole point | S:70 R:90 A:85 D:80 |
| 6 | Confident | The recovery policy is written into `_preamble.md`'s existing five-state numbered list (rows 3.`orphaned`/`failed`/`failed (no-result)`) plus new bullets after it, rather than as a new `###` subsection | § CLI-Adapter Dispatch is declared canonical and the sibling surfaces reference it; keeping the policy inside the existing procedure keeps the single-source structure and avoids a second place to look | S:65 R:85 A:80 D:70 |
| 7 | Confident | The peek's headless command is `fab dispatch logs <change> <stage> --tail 40` (the intake's `--tail 40`), and the pane command is taken from `fab dispatch logs`'s own report (which prints the socket-included `fab pane capture` command) | `--tail 40` is the intake's stated value; routing the pane command through the `logs` report is the shipped copy-pasteable source (`status --json` carries `pane` but not `server`), so the policy should not hand-assemble it | S:70 R:85 A:85 D:75 |
| 8 | Confident | The sibling dispatch-seam surfaces (`_pipeline.md`, `fab-continue.md`, `fab-adopt.md` + mirrors) need NO recovery-policy text of their own — they already delegate the five-state handling to `_preamble.md` § CLI-Adapter Dispatch by reference, which the grep-sweep confirms | § CLI-Adapter Dispatch explicitly states those sites "reference this subsection and do NOT restate the five-state machine"; adding policy text there would violate that single-source rule and create the drift the sweep exists to prevent | S:70 R:80 A:85 D:70 |
| 9 | Confident | `docs/memory/pipeline/execution-skills.md` needs no edit unless the grep finds a five-state/poll-loop restatement; the intake itself makes it conditional ("only if it restates … grep-verify") | The intake grades this file as conditional, and the memory FKF present-truth rule says a file that does not carry the claim should not gain a restatement of it | S:80 R:85 A:85 D:80 |
| 10 | Confident | `restart` accepts the same `--timeout` flag as `start` (with the same `--pane` exclusion), rather than omitting it | The intake lists `--timeout` in restart's flag set explicitly ("same flags (`--pane`/`--headless`/`--timeout`/`--server`, same exclusions"); a restart of a headless attempt is exactly where a timeout bound is wanted | S:80 R:85 A:85 D:80 |

10 assumptions (2 certain, 8 confident, 0 tentative).
