# Plan: Operator Drives the rk Cron Clock Explicitly — Mute/Unmute on Tracked-Set Transitions, Lease as Bounded Snooze

**Change**: 260909-qvek-operator-cron-mute-lease
**Intake**: `intake.md`

## Requirements

> Grounding (verified 2026-09-09, see intake § Clarifications): run-kit `260909-upt2` is merged (sahil87/run-kit#889); installed rk v3.19.37 has `rk cron mute <id> [--for <dur>] [--off]`; `rk cron list --json` rows are `{id, name, schedule, target, deliver, pinned, muted, muted_until?, last_fired, orphaned_since, expires_at}` — `target` is a string (`"role:operator"`), `muted` is the *effective* state (indefinite mute OR live lease), `muted_until` (unix seconds) is present only while a lease is live.

### Go: Tracked-Set Predicate and Clock Sync (`src/go/fab/cmd/fab/`)

#### R1: Tracked predicate
The operator state SHALL be **tracked** when ANY holds: `monitored` is non-empty; the `autopilot` block is present with a non-null `state` (an exhausted block with `state: null` counts as empty); `watches` is non-empty (enabled **or** disabled entries both count); any `notes` entry has `kind: coordination` and `resolved: false`. Otherwise it is **untracked**.

- **GIVEN** a state with empty `monitored`, `autopilot: null`, empty `watches`, and one unresolved `kind: coordination` note
- **WHEN** the predicate is evaluated
- **THEN** it reports tracked
- **AND** the same state with that note `resolved: true` reports untracked

#### R2: Edge-triggered clock sync in `mutateOperatorState`
`mutateOperatorState` (`operator_state.go`) SHALL evaluate R1 on the state before and after `fn`, and — only after the mutated state has been saved successfully — issue exactly one rk call when the boolean flips: tracked→untracked ⇒ `rk cron mute <id>`; untracked→tracked ⇒ `rk cron mute <id> --off`. A mutation that does not flip the predicate SHALL issue no rk call.

- **GIVEN** an untracked state
- **WHEN** `fab operator enroll` adds the first monitored entry
- **THEN** exactly one `rk cron mute <id> --off` is issued after the save
- **AND** a second `enroll` issues no rk call
- **GIVEN** a state tracked only by one monitored entry
- **WHEN** `fab operator remove` deletes it
- **THEN** exactly one `rk cron mute <id>` is issued
- **GIVEN** a state tracked only by an open coordination note
- **WHEN** `fab operator autopilot advance` exhausts the queue
- **THEN** no mute is issued (the note keeps it tracked)

#### R3: Entry resolution
The clock sync SHALL resolve the entry id per call by running `rk cron list --json` and selecting the row whose `target` equals `"role:operator"`, using `name == "operator tick"` as the tiebreak when several match. Zero candidates, an unresolved tie, or unparseable output SHALL be a silent no-op.

- **GIVEN** a `rk cron list --json` document with one `target: "role:operator"` row
- **WHEN** a flip occurs
- **THEN** the mute/`--off` call carries that row's `id`
- **GIVEN** an empty array or malformed JSON
- **WHEN** a flip occurs
- **THEN** no mute call is made and the verb succeeds normally

#### R4: Fail-silent, gated, bounded, argv-only rk calls
Every rk call SHALL be gated on `exec.LookPath("rk")`, executed as argv (never a shell string), bounded by a per-call timeout constant (5s), and fail-silent: rk absent, a non-zero exit (older rk without `--off`, `no entry`), a timeout, or a parse failure SHALL never change the invoking verb's exit code, stdout, or the already-saved state. No `-L <server>` is passed — rk's own `$TMUX` derivation addresses the server.

- **GIVEN** rk absent from PATH
- **WHEN** any tracked-set verb flips the predicate
- **THEN** the verb exits 0 with its usual output and no error is printed
- **GIVEN** an rk that exits non-zero on `mute --off`
- **WHEN** a flip occurs
- **THEN** the verb's exit code and stdout are unchanged

#### R5: Injectable runner seam
The rk invocation SHALL go through a package-level function variable (the `rkOperatorPath` / `rkPanesRunner` precedent) so tests stub `rk cron list --json` output and record the argv of every mute call without a live rk.

- **GIVEN** the test stubs the runner
- **WHEN** a verb flips the predicate
- **THEN** the test observes the exact argv (`cron mute <id>` / `cron mute <id> --off`) and count

#### R6: Tick-start reconcile (loud direction)
`fab operator tick-start --diff` (`operator_tick_start.go`, `runOperatorTickStartDiff`) SHALL, after its baseline write, issue exactly one `rk cron mute <id>` when the resulting state is untracked, and none when tracked. A tick arriving on an untracked state is the drift signal, so the call fires only while drifted. The reconcile SHALL NOT double-issue when R2's flip logic already muted in the same mutation.

- **GIVEN** an untracked state
- **WHEN** `fab operator tick-start --diff --quiet` runs
- **THEN** exactly one indefinite mute is issued and the tick document is emitted unchanged
- **GIVEN** a tracked state
- **WHEN** tick-start runs
- **THEN** no rk call is made

#### R7: Context-bound subprocess helper
`internal/pane` SHALL gain a context-bound sibling of `RunCmd` (e.g. `RunCmdContext(ctx, name, args...)`) with the same return shape, used by the clock sync with a `context.WithTimeout`. Existing `RunCmd` behavior is unchanged.

- **GIVEN** a command that sleeps past the deadline
- **WHEN** invoked through the context-bound helper
- **THEN** it returns a non-nil error within the deadline

#### R8: Cross-repo contract comments
`slugify`, `serverSlug`, and `StatePath` in `operator.go` SHALL carry a comment stating that run-kit mirrors this exact slug rule to locate the operator state file for display (◉ watched rows, `⚠ operator stale`), pinned in run-kit's `docs/specs/cron.md`; fab-kit owns the file and the rule; renaming either requires a coordinated run-kit change. The `slugify` tests in `operator_test.go` SHALL carry a one-line comment naming the cross-repo consumer.

- **GIVEN** a future edit to `slugify`
- **WHEN** the editor reads the function or its test
- **THEN** both name run-kit as a consumer of the rule

### Skill: `src/kit/skills/fab-operator.md`

#### R9: §2 Init steps 4–5 — clock verification reads JSON and reconciles the silent direction
Step 4 SHALL read `rk cron list --json` (still `command -v rk`-gated, fail-silent) to find the operator-tick entry and its mute state, and SHALL issue `rk cron mute <id> --off` when the entry is `muted` while `fab operator state` shows tracked work (R1's predicate, evaluated by the agent from the state read in step 1). Step 5 SHALL list four literal ready-line forms — the existing first form, ` · muted` (row `muted: true`, no `muted_until`), ` · muted until <HH:MM>` (`muted_until` present, rendered local time), and the existing `Clock: none` form — under the copy-never-compose rule.

- **GIVEN** `rk cron list --json` shows the entry with `muted: true` and no `muted_until`, and the state file has a monitored entry
- **WHEN** the operator starts or reloads
- **THEN** it issues `rk cron mute <id> --off` and prints the first (unmuted) ready-line form
- **GIVEN** the entry has `muted_until` live and the state is untracked
- **WHEN** the operator starts
- **THEN** it prints the ` · muted until <HH:MM>` form and issues nothing

#### R10: §4 The Clock — guard-free entry, verb ownership, Mute and Lease
§4 SHALL quote the seeded entry WITHOUT `anchor` and WITHOUT `suppress_while`, with `respawn: ["rk", "operator", "-L", "{server}"]` (caller-supplied argv; `{server}` substituted by rk at fire time). The Ownership paragraph SHALL state that `rk operator` seeds the entry idempotently, the tracked-set verbs (`fab operator enroll`/`remove`, `watch add`/`rm`, `autopilot start`/`stop`/`advance`-to-exhaustion, `note add --kind coordination`/`resolve`) mute/unmute it via `rk cron mute <id>` / `--off`, and `rk cron add`/`rm` and the schedule stay the user's; the "never creates or mutates it" and anchor-join sentences are deleted. The union-predicate bullets SHALL replace the `nothing-tracked` bullet with the tracked-set-verb rule, delete the `operator-loop-fresh` bullet, and drop "anchored on operator idle" from the backoff bullet. A new `### Mute and Lease` subsection SHALL carry: MAY use `rk cron mute <id> --for <dur>` for a user-requested bounded quiet window (auto-expires; no unmute needed); MUST never leave an indefinite mute behind while work is tracked (the Go verbs enforce the tracked-set half; the prose covers a manual mute); `--off` clears both; the lease is a bounded snooze, not a heartbeat — nothing renews it since the in-session loop is retired.

- **GIVEN** the rewritten §4
- **WHEN** grepped for `anchor`, `suppress_while`, `nothing-tracked`, `operator-loop-fresh`
- **THEN** no match remains in the skill except historical mention inside the Design Decisions of memory (not this file)

#### R11: Tick Behavior step 7, §6 gap paragraph, §9 Cadence row
Tick Behavior step 7 SHALL read: clock lifecycle — none to manage in the tick; the tracked-set verbs mute/unmute the entry as a side effect of their state mutation (§4 Mute and Lease); cadence adaptation is the entry's backoff + `wake_on` union predicate, evaluated by rk. §6's "Known gap (run-kit follow-up)" paragraph SHALL be replaced by: an open merge-sequence `coordination` note is tracked state to the mute logic (`note add --kind coordination` unmutes, `note resolve` may mute), so ticks keep coming for the whole armed sequence — no gap, no follow-up. §9's Cadence row SHALL drop the suppress-guard and "never mutated by the skill" claims and add "tracked-set verbs mute/unmute via `rk cron mute`; lease = bounded snooze". The Tick Payload rule, Degraded Fallback paragraph, and Post-Compaction Reload SHALL be unchanged.

- **GIVEN** the rewritten skill
- **WHEN** a reader follows step 7 or §6 rule 4
- **THEN** neither names a run-kit guard or a pending follow-up

### Docs: CLI Reference, External Reference, Spec

#### R12: `_cli-fab.md` § fab operator — clock side effect documented once, State path contract sentence
`src/kit/skills/_cli-fab.md` § fab operator SHALL gain one shared paragraph describing the clock side effect (R1–R6: predicate, edge-triggered mute/`--off`, entry resolution, fail-silent/gated/bounded, tick-start reconcile), referenced by one sentence from each of `### fab operator tick-start`, `### fab operator enroll / update / remove`, `### fab operator note`, `### fab operator watch`, and `### fab operator autopilot`. The **State path** paragraph SHALL gain the cross-repo-contract sentence (R8's wording, sibling of the memory sentence).

- **GIVEN** the updated reference
- **WHEN** a reader looks up any tracked-set verb
- **THEN** it points at the shared clock-side-effect paragraph, which is stated exactly once

#### R13: `_cli-external.md` § rk pointer
`src/kit/skills/_cli-external.md` § rk (run-kit) SHALL add a fab-owned pointer "Operator clock mute/lease" → `fab-operator.md` §4 Mute and Lease (skill-side lease) and `_cli-fab.md` § fab operator (Go-side mute/unmute). § /loop is unchanged.

- **GIVEN** the § rk section
- **WHEN** read
- **THEN** it points at the two owners and restates neither rule (owner-or-pointer)

#### R14: `docs/specs/skills.md` § /fab-operator
The Flow header line (~1114) and the Tools line (~1120) SHALL add the mute/lease posture: cadence delivered by the guard-free entry; the tracked-set verbs mute/unmute it; `rk cron list --json` / `rk cron mute --for` are the skill's own rk uses. Authored human edit (Constitution VI permits it).

- **GIVEN** the spec section
- **WHEN** read against the skill
- **THEN** both describe the same clock ownership

### Non-Goals
- Any run-kit change — shipped in `260909-upt2-cron-decoupling-mute-lease` (sahil87/run-kit#889)
- Changing the operator state file's name or schema — no migration ships (no user-data restructuring)
- Re-introducing any rk-side inference of operator state
- Passing `-L <server>` to rk from the operator verbs — fab has no server name there
- Memory realignment of `docs/memory/runtime/operator.md` — hydrate's step, per intake § 4 and § Affected Memory (the Design Decisions below are shaped for hydrate to lift)

### Design Decisions

#### Explicit Mute Over Inferred Quiescence
**Decision**: The operator tells the clock when to be quiet through rk's own verbs (`rk cron mute <id>` / `--off`), issued from the Go verbs that mutate the tracked set; rk never infers operator intent from fab's private state file.
**Why**: rk's parsing of fab's operator state file resolved the wrong filename and silently suppressed every operator tick on every server for days; a verb the operator issues cannot fail that way, and the Go placement cannot be skipped by an LLM step or lost to compaction.
**Rejected**: rk-side inference (the incident class); mute from the skill's tick prose (skippable, compaction-lossy); a fab-side heartbeat (nothing to renew it — the in-session loop is retired).
*Introduced by*: 260909-qvek-operator-cron-mute-lease

#### Edge-Triggered Flip Plus Lightweight Reconcile
**Decision**: `mutateOperatorState` issues an rk call only when tracked-ness flips (empty↔non-empty), and two cheap self-heals close the drift window a silently failed flip leaves: `tick-start --diff` mutes when it finds the state untracked (Go), and §2 Init unmutes when the entry is muted while work is tracked (prose, on every startup/reload).
**Why**: Edge-triggering costs zero rk calls per tick and leaves a user's lease alone across unrelated mutations; the reconcile covers both drift directions, and muted-while-tracked is exactly the incident class this change exists to prevent, so leaving it unhealed was the wrong default.
**Rejected**: Level-triggered mute per mutation (shells out every tick, clears user leases); no reconcile (loud drift persists until the next flip, silent drift until noticed); Init-only reconcile (loud drift between reloads).
*Introduced by*: 260909-qvek-operator-cron-mute-lease

#### Lease Is a Bounded Snooze, Not a Heartbeat
**Decision**: `rk cron mute <id> --for <dur>` is used only for a user-requested bounded quiet window; it auto-expires and nothing renews it.
**Why**: 260909-6t88 retired the in-session `/loop`, so no loop exists to renew a lease; `operator-loop-fresh` has no successor.
**Rejected**: Renewing leases from a loop (no loop); treating the lease as the quiescence mechanism (the verbs' indefinite mute is).
*Introduced by*: 260909-qvek-operator-cron-mute-lease

#### Any Open Coordination Note Counts as Tracked
**Decision**: The tracked predicate treats any unresolved `kind: coordination` note as "merge sequence open"; no reserved marker is introduced.
**Why**: The note verbs carry no sub-kind today; over-inclusion (a non-merge coordination note keeping ticks on) is the harmless direction, while a forgotten marker would mute mid-sequence — the bad direction.
**Rejected**: A `--ref merge-sequence` marker convention the §6 rule-4 note must always carry.
*Introduced by*: 260909-qvek-operator-cron-mute-lease

## Tasks

### Phase 1: Setup

- [x] T001 Add a context-bound sibling of `RunCmd` to `src/go/fab/internal/pane/pane.go` (`RunCmdContext(ctx context.Context, name string, args ...string) (string, []byte, error)` — same capture shape via `exec.CommandContext`), leaving `RunCmd` unchanged; add a deadline test in `src/go/fab/internal/pane/pane_test.go` (or the package's existing test file) using a sleeping command <!-- R7 -->

### Phase 2: Core Implementation

- [x] T002 Create `src/go/fab/cmd/fab/operator_clock.go`: `operatorTracked(data map[string]interface{}) bool` per R1 (decode `monitored`/`autopilot`/`watches`/`notes` via `operatorSection`); the package-level runner seam `var rkCronRunner = func(args ...string) (string, error)` (LookPath-gated, argv exec through `pane.RunCmdContext` with a 5s `context.WithTimeout` constant); `resolveOperatorCronID() (string, bool)` parsing `rk cron list --json` per R3 (a minimal struct with `id`, `name`, `target`, `muted`, `muted_until`); `syncOperatorClock(before, after bool)` issuing `mute <id>` / `mute <id> --off` on a flip only, fail-silent per R4 (all errors swallowed, nothing written to stdout/stderr); `muteOperatorClockIfUntracked(data)` for R6 <!-- R1 R2 R3 R4 R5 -->
- [x] T003 Wire the edge trigger into `mutateOperatorState` in `src/go/fab/cmd/fab/operator_state.go`: compute `before := operatorTracked(data)` after load, `after := operatorTracked(data)` after `fn`, save, then `syncOperatorClock(before, after)` only when the save succeeded; add a doc comment naming the clock side effect and that failures never surface <!-- R2 R4 -->
- [x] T004 Add the tick-start reconcile in `src/go/fab/cmd/fab/operator_tick_start.go` `runOperatorTickStartDiff`: capture tracked-ness of the post-diff state inside the mutation closure and, after the mutation returns without error, issue one indefinite mute when untracked (skip when the closure's own flip already muted — e.g. by having the tick-start path bypass the edge trigger and mute level-wise, or by threading a flag; pick the simplest that satisfies the exactly-one test) <!-- R6 -->
- [x] T005 Write `src/go/fab/cmd/fab/operator_clock_test.go` (plus additions to `operator_monitored_test.go` / `operator_watch_test.go` / `operator_autopilot_test.go` / `operator_note_test.go` / `operator_tick_diff_test.go` where the verb harnesses live) stubbing `rkCronRunner` to serve a fixture `rk cron list --json` document and record argv: (a) predicate table for R1 incl. disabled watch, exhausted autopilot, resolved vs open coordination note; (b) untracked→tracked on `enroll`, `watch add`, `autopilot start`, `note add --kind coordination` each issue exactly one `--off`; (c) tracked→untracked on the last `remove`, `watch rm`, `autopilot stop`, `note resolve` each issue exactly one indefinite mute; (d) `autopilot advance` to exhaustion mutes with everything else empty but not with an open coordination note; (e) non-flipping mutations (second `enroll`, `update`, `note add --kind correction`, tick-start on a tracked state) issue no call; (f) tick-start on an untracked state issues exactly one mute; (g) rk absent / non-zero / timeout / malformed JSON / zero or two candidate rows leave the verb's exit code and stdout unchanged and issue no mute; (h) tiebreak picks `name == "operator tick"` among two `role:operator` rows <!-- R1 R2 R3 R4 R5 R6 -->
- [x] T006 [P] Add the cross-repo contract comment above `slugify`, `serverSlug`, and `StatePath` in `src/go/fab/cmd/fab/operator.go` and a one-line consumer comment on the `slugify` tests in `src/go/fab/cmd/fab/operator_test.go` (~line 668) <!-- R8 -->

### Phase 3: Integration & Edge Cases

- [x] T007 Rewrite `src/kit/skills/fab-operator.md` §2 Init step 4 (read `rk cron list --json`; reconcile muted-while-tracked with `rk cron mute <id> --off`) and step 5 (four literal ready-line forms incl. ` · muted` and ` · muted until <HH:MM>`, copy-never-compose) <!-- R9 -->
- [x] T008 Rewrite `src/kit/skills/fab-operator.md` §4 The Clock: entry YAML without `anchor`/`suppress_while` plus the `respawn` argv line; Ownership paragraph (seeds / verbs mute-unmute / user owns add-rm-schedule; delete the never-mutates and anchor-join sentences); union-predicate bullets (replace `nothing-tracked` bullet, delete `operator-loop-fresh` bullet, drop "anchored on operator idle"); add `### Mute and Lease` before `### Tick Payload`; leave Tick Payload, Degraded Fallback, Post-Compaction Reload untouched; add `Mute and Lease` to the `## Contents` list <!-- R10 -->
- [x] T009 [P] Rewrite `src/kit/skills/fab-operator.md` Tick Behavior step 7, replace the §6 "Known gap (run-kit follow-up)" paragraph (line ~795) with the closed-by-construction sentence, and update the §9 Key Properties `Cadence` row <!-- R11 -->
- [x] T010 [P] Update `src/kit/skills/_cli-fab.md` § fab operator: add one shared "Clock side effect" paragraph (after the Shared state-verb mechanics paragraph), one referencing sentence in each of `### fab operator tick-start` (the reconcile), `### fab operator enroll / update / remove`, `### fab operator note`, `### fab operator watch`, `### fab operator autopilot`; append the cross-repo-contract sentence to the **State path** paragraph (~line 1266) <!-- R12 -->
- [x] T011 [P] Add the "Operator clock mute/lease" pointer to `src/kit/skills/_cli-external.md` § rk (run-kit) — pointer only, no rule restatement <!-- R13 -->
- [x] T012 [P] Update `docs/specs/skills.md` § /fab-operator Flow header line (~1114) and Tools line (~1120) with the mute/lease posture <!-- R14 -->

### Phase 4: Polish

- [x] T013 Sibling sweep + verification: grep `src/ docs/specs/ README.md` (excluding `fab/changes/`, `docs/memory/`, `.agents/`, `.claude/`, `.opencode/`) for `suppress_while`, `nothing-tracked`, `operator-loop-fresh`, `anchor: operator-idle`, `never creates or mutates` and fix every remaining hit in the class; run `gofmt -l src/go` (must be empty) and `go test ./src/go/fab/cmd/fab/... ./src/go/fab/internal/pane/...` from the module root (all green); confirm no edit landed under `.agents/skills/` or `.claude/skills/` <!-- R10 R12 -->

## Execution Order

- T001 blocks T002 (the runner uses `RunCmdContext`)
- T002 blocks T003, T004, T005
- T006 is independent of T001–T005
- T007–T012 are independent of the Go tasks and of each other (all edit different sections/files; T007–T009 touch the same file — run sequentially or coordinate edits)
- T013 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `operatorTracked` returns true for each of: one monitored entry; autopilot with `state: running` or `paused`; one watch (enabled or disabled); one unresolved `kind: coordination` note — and false for the empty skeleton, an exhausted autopilot (`state: null`), and a resolved coordination note
- [x] A-002 R2: `mutateOperatorState` issues `rk cron mute <id> --off` exactly once on an untracked→tracked flip and `rk cron mute <id>` exactly once on a tracked→untracked flip, after the save, and nothing on a non-flipping mutation
- [x] A-003 R3: The entry id is taken from the `rk cron list --json` row with `target == "role:operator"` (tiebreak `name == "operator tick"`); zero, ambiguous, or malformed input yields no mute call
- [x] A-004 R4: All rk calls are `exec.LookPath`-gated argv executions bounded by a 5s context timeout; no `-L` flag; no shell string
- [x] A-005 R5: A package-level runner variable is the only path to rk, and the tests stub it
- [x] A-006 R6: `tick-start --diff` issues exactly one indefinite mute on an untracked state and none on a tracked state
- [x] A-007 R7: `pane.RunCmdContext` exists with `RunCmd`'s return shape; `RunCmd` is unchanged
- [x] A-008 R8: `slugify`/`serverSlug`/`StatePath` and the `slugify` tests carry the cross-repo consumer comment
- [x] A-009 R9: §2 Init step 4 reads `rk cron list --json` and reconciles muted-while-tracked with `--off`; step 5 lists the four literal ready-line forms
- [x] A-010 R10: §4 quotes the entry without `anchor`/`suppress_while` and with `respawn`; Ownership names the tracked-set verbs as the mute/unmute owners; `### Mute and Lease` exists with the MAY/MUST/`--off`/not-a-heartbeat rules
- [x] A-011 R11: Step 7, the §6 paragraph, and the §9 Cadence row carry the new posture; Tick Payload, Degraded Fallback, Post-Compaction Reload are byte-identical to before
- [x] A-012 R12: `_cli-fab.md` § fab operator states the clock side effect once with five per-verb references, and the State path paragraph carries the cross-repo sentence
- [x] A-013 R13: `_cli-external.md` § rk carries the pointer and no restated rule
- [x] A-014 R14: `docs/specs/skills.md` § /fab-operator Flow and Tools lines mention the mute/lease posture

### Behavioral Correctness

- [x] A-015 R4: With rk absent, non-zero, timing out, or returning malformed JSON, every tracked-set verb and tick-start exit with the same code and stdout as before this change
- [x] A-016 R2: A user-set lease (`--for`) survives non-flipping mutations (no rk call touches it); a flip to tracked clears it via `--off`
- [x] A-017 R6: The tick-start reconcile never issues two mutes in one invocation

### Removal Verification

- [x] A-018 R10: No occurrence of `suppress_while`, `nothing-tracked`, `operator-loop-fresh`, or `anchor: operator-idle` remains under `src/kit/skills/` or `docs/specs/`
- [x] A-019 R11: The "Known gap (run-kit follow-up)" paragraph is gone from `fab-operator.md`

### Scenario Coverage

- [x] A-020 R2: Tests cover the untracked→tracked flip for `enroll`, `watch add`, `autopilot start`, `note add --kind coordination`, and the tracked→untracked flip for the last `remove`, `watch rm`, `autopilot stop`, `note resolve`
- [x] A-021 R1: A test shows `autopilot advance` to exhaustion mutes when nothing else is tracked and does not mute while a coordination note is open
- [x] A-022 R3: A test exercises the tiebreak over two `role:operator` rows

### Edge Cases & Error Handling

- [x] A-023 R4: A stubbed runner that blocks past the deadline returns within the timeout and leaves the verb's output unchanged
- [x] A-024 R2: When `saveOperatorState` fails, no rk call is issued (the clock never diverges from a state that was not persisted)

### Code Quality

- [x] A-025 Pattern consistency: New Go follows the `rkOperatorPath`/`rkPanesRunner` seam style, `operatorSection` decoding, and the file's error-handling conventions
- [x] A-026 No unnecessary duplication: The tracked predicate and rk invocation exist once (`operator_clock.go`) and are reused by `mutateOperatorState` and tick-start
- [x] A-027 Readability over cleverness: `syncOperatorClock` and `operatorTracked` are short, single-purpose functions (no god function)
- [x] A-028 Magic values named: the 5s timeout, `"role:operator"`, and `"operator tick"` are named constants
- [x] A-029 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`
- [x] A-030 CLI ⇒ docs + tests: the Go behavior change is reflected in `_cli-fab.md` and covered by tests
- [x] A-031 Owner-or-pointer: `_cli-external.md` and the per-verb `_cli-fab.md` sentences point at the owner paragraph rather than restating it; `fab-operator.md` §4 states the skill-side rule once
- [x] A-032 Sibling sweep: the T013 grep class is clean; no stale twin claim remains in `skills.md`
- [x] A-033 Migrations: no user-data restructuring occurred, so no migration file is required (state schema unchanged)
- [x] A-034 gofmt clean and `go test` green for `cmd/fab` and `internal/pane`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Memory (`docs/memory/runtime/operator.md`) is realigned at hydrate per intake § 4: clock block, monitoring-tick quiescence sentence, server-keyed-state-file cross-repo sentence, rewrite "Cron Entry as the Sole Cadence, Provider-Neutral", supersede "Document the `nothing-tracked`/Merge-Sequence Gap, No Workaround", lift the four Design Decisions above

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (the retired `suppress_while` guards lived in run-kit, removed there by 260909-upt2; nothing in this repo's Go or skill text became dead code)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | New Go lives in one file `operator_clock.go` (predicate + runner seam + entry resolution + sync) rather than inside `operator_state.go` | Keeps `operator_state.go` to state IO; mirrors the per-concern file split (`operator_note.go`, `operator_watch.go`) | S:70 R:90 A:85 D:70 |
| 2 | Confident | The context-bound helper is named `RunCmdContext` and takes a caller-supplied `context.Context` | Go stdlib naming (`exec.CommandContext`); the 5s deadline stays in the operator package so `pane` stays policy-free | S:65 R:90 A:85 D:75 |
| 3 | Confident | Ready-line lease expiry renders as local `HH:MM` | User confirmed at clarify (intake row 11); the exact copy is a literal the agent copies | S:85 R:90 A:80 D:75 |
| 4 | Confident | The tick-start reconcile is a level-wise mute-if-untracked after the mutation, de-duplicated against the closure's own flip so exactly one call is made | Intake row 19; the exactly-one test pins it; implementation choice left to the worker | S:75 R:85 A:75 D:70 |
| 5 | Confident | `note add --kind correction` (and other non-coordination kinds) never affect tracked-ness | Only coordination notes model the merge sequence (intake row 7) | S:80 R:85 A:85 D:80 |

5 assumptions (0 certain, 5 confident, 0 tentative).
