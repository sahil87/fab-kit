# Intake: fab dispatch pane-mode done-worker reaping

**Change**: 260807-zfl7-dispatch-reap-done-panes
**Created**: 2026-08-08

## Origin

Backlog **[zfl7]** (2026-08-07), **part 1 only** — created via `/fab-proceed`'s promptless create-intake dispatch on 2026-08-08, from a synthesized description carrying the decisions of a `/fab-discuss` scoping session (2026-08-08). The entry was originally deferred from the deck-v layout discussion of 2026-08-07.

> fab dispatch pane-mode worker UX polish: done-worker pane hygiene (reap/zoom-park after done result read) + richer pane names/colors for stage workers — welcome but cuts into understanding of the worker grammar; deferred from deck-v layout discussion 2026-08-07

**Part 2 of the entry (richer pane names/colors for stage workers) was explicitly split out and stays deferred.** The discussion rejected it for v1 on three grounds: richer titles lose to harness title-clobbering (the documented reason split-shape placement keys on record pane-IDs, never titles — see `docs/memory/runtime/dispatch.md` § Worker placement keys on the record's pane ID); state-reactive colors have no honest eventing home (no timers/daemons — the no-magic-background-work posture; `status`/`wait` are pure observation verbs); and border-status display reaches into user tmux config. Run-kit's instrumentation/dashboard side may cover display needs instead. On ship, the backlog entry is amended: part 1 marked covered by this change, part 2 remains as the residual deferred entry.

## Why

**Problem**: A pane-mode stage worker never exits on completion — it writes `{stage}-result.yaml` and sits at its interactive prompt (this is by design: the result file is the sole completion signal, and requiring exit was explicitly rejected to keep the steer-after-finish property). Over a multi-stage pipeline the carved worker column accumulates finished panes that hold their space indefinitely: the orchestrator reads the `done` result and moves on, and the pane just lingers. Each finished worker permanently occupies a slice of the stacked column that the two-tier hierarchy (260806/260807 work) carved to keep the dispatching agent readable.

**If not fixed**: a 3-stage pipeline leaves 3 dead panes stacked in the 35% column; the panes the user actually needs to watch (the live worker, the dispatcher) shrink with every completed stage, and the visual hygiene the pane-worker-column work bought erodes across a run.

**Why this approach**: reap (kill) the done worker's pane at the one deterministic orchestrator-invoked moment that already exists — immediately after the orchestrator reads a `done` result from a CLI dispatch — behind a default-true config knob. The policy guard lives in Go (a new `fab dispatch reap` subcommand), where the three-layer config cascade is available (a skill reading `fab/project/config.yaml` directly would miss the system layer `~/.fab-kit/config.yaml`); the skill wiring stays unconditional and dumb. Zoom/park alternatives (shrink-in-place `resize-pane -y 2`, or `break-pane` to a parked window, preserving scrollback) were considered and rejected for v1 — parking either keeps a stub in the column or re-clutters the window list the two-tier hierarchy just cleaned up. Users who want done-worker scrollback set `dispatch.reap_done: false` and keep today's leave-the-pane-alone behavior.

**Safety (verified against code + memory)**: reaping a done pane changes no state machine. `DerivePaneState(resultPresent, paneAlive)` gives result-file presence precedence over pane liveness, so a reaped `done` dispatch reads `done` forever; tmux `kill-pane` semantics leave the dispatcher's pane, its window, and sibling worker panes intact; kill is idempotent (already-gone pane → benign no-op); and `kill` is already in the pipeline's sanctioned verb set (peek/kill/restart/notify/stop). No new dispatch state, no record schema change, no migration.

## What Changes

### 1. New subcommand: `fab dispatch reap <change> <stage>`

The `fab dispatch` family grows from seven subcommands to eight: `start`, `restart`, `status`, `wait`, `logs`, `kill`, **`reap`**, `clean`. `reap` owns the **whole guard** — it kills the worker pane **only when all three** hold:

1. the dispatch record is **pane-mode** (`Dispatch.IsPane()` — a `pane:`-bearing record), and
2. the **derived state is `done`** (`DerivePaneState`: `{stage}-result.yaml` present — pane liveness is irrelevant to the state), and
3. **`dispatch.reap_done` resolves `true`** through the config cascade.

In every other case it is a **no-op with a clear one-line report** (exit 0), naming the reason:

| Case | Behavior |
|------|----------|
| record is headless | no-op — the process already exited; there is nothing visual to reclaim |
| derived state is not `done` (`running` / `orphaned` — or any headless state) | no-op — reap is NOT kill; it must never terminate a live or failed dispatch |
| `dispatch.reap_done` resolves `false` | no-op — the user opted to keep done-worker scrollback |
| pane already gone (killed by hand, tmux server died) | benign no-op / already-gone report — mirrors `kill`'s idempotence |
| all three conditions hold | `KillPane(paneID, server)` — the record's persisted `server` is used, so a `--server`-started dispatch is reaped on the right socket with no flag |

Only **real errors** exit non-zero: no dispatch record for the pair, or an unresolvable change — mirroring `status`/`kill`'s message surface. Exact report wording is refinable at plan time; the success line should name the pane (e.g. `reaped pane %N for <id>/<stage>`), and each no-op should name its reason in one line.

**Reap kills the pane only — `.fab-dispatch/` state files are untouched.** The record (`{stage}.yaml`), result (`{stage}-result.yaml`), and prompt file all remain; that is exactly why a reaped dispatch still reads `done` forever, and why reap is pane **hygiene**, not state cleanup — the no-automatic-GC posture (archive-time deletion + explicit `fab dispatch clean` as the only two cleanup moments) is unchanged. `kill-pane` also covers both pane shapes uniformly: killing the only pane of a new-window-shape worker takes the window with it (plain tmux semantics), and killing a split-shape worker's pane leaves the dispatcher's window and siblings intact.

**Reap is NOT kill** — it must never terminate a `running`/`orphaned`/`failed`/`failed (no-result)` dispatch. The Recovery policy (peek-on-suspicion, single restart budget, escalation, classification (c) never-kill) is untouched. `restart` after a reaped attempt needs no special handling: last-attempt-only overwrite already covers completed attempts (a `start`/`restart` over a `done` prior attempt overwrites its files).

**Implementation split** (matching the family's existing pattern): the testable guard core in `internal/dispatch` (composing the existing `Load`, `IsPane()`, `ResultPresent`, `DerivePaneState`, `PaneAlive`, `KillPane` — the reap decision itself should be a pure, table-testable function like `SelectMode`/`DerivePaneState`), thin cobra wiring in `cmd/fab/dispatch_reap.go` registered on the `dispatch.go` parent. Paths are relative to `src/go/fab/`. No flags in v1 (no `--json`, no `--server` — the socket comes from the record).

### 2. New config field: `dispatch.reap_done`

```yaml
dispatch:
  watchable: false      # existing
  column_width: 35      # existing
  reap_done: true       # NEW — default true (reap); false preserves today's leave-the-pane-alone behavior
```

- **Type/default/scope**: bool, default **`true`**, scope **`both`** (settable once machine-wide in `~/.fab-kit/config.yaml`, sitting beside `dispatch.watchable` and `dispatch.column_width`). Advertise `true` (a surface users are expected to reach for, like its two siblings).
- **Struct/accessor** (`internal/config/config.go`): the field must be `ReapDone *bool` (pointer), **not** plain `bool` — a default-**true** bool cannot ride the Go zero value the way `Watchable` (default false) does; `nil` = unset = `true`, and an explicit `false` must be distinguishable from absent. Accessor `GetDispatchReapDone() bool` returns `true` when nil, else the pointed value — following the existing `GetDispatchWatchable`/`GetDispatchColumnWidth` accessor pattern (nil-receiver safe).
- **Registry** (`internal/configref/configref.go`): a new row in the ordered `[]Field` slice (16 → **17 rows**), `{Key: "dispatch.reap_done", Default: true, Description: …, Scope: both, Advertise: true}`. Per the "Several Registry Rows Under One YAML Block Share a Single Segment" rule, the `dispatch:` block's single `Segment` lives on the `dispatch.watchable` row — **extend that shared Segment** to document and scaffold `reap_done`, and give the new row an **empty Segment** exactly like `dispatch.column_width`'s. The registry's non-null-default count goes 6 → **7** (`reap_done` carries a real built-in `true` — same boundary-case rationale as the two existing `dispatch` rows: a bool has no "absent" state in JSON output). The `--json` key-parity guard and registry lint pick the row up automatically; update their fixtures/counts where pinned (e.g. `internal/configupgrade/freeze_test.go` if it pins fence bytes).
- **No `configscope` change**: scope enforcement is keyed on the top-level `dispatch` parent, already `ScopeBoth` in `internal/configscope/configscope.go`.
- **No migration**: additive field with a built-in default under the presence=intent fence model — the reference fence is regenerated by `fab config upgrade` on every upgrade, so unmodified projects pick up the new commented documentation automatically; absent key = default `true`.

### 3. Skill wiring: `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch

- **Step 3, `done` bullet**: after reading `.fab-dispatch/{id}/{stage}-result.yaml` from a CLI dispatch, run `fab dispatch reap <change> <stage>`. The wiring is **unconditional and dumb** — no mode check, no config check in the skill: headless records no-op inside the command (the process already exited), and the knob/state guards live in Go where the cascade is readable. The pane-mode subsection's `done` handling (step 3 subset bullet, "Handle `done` and `orphaned` exactly as above") inherits the call by reference — verify its wording still reads correctly.
- **Step 4 ("No cleanup after `done`") reconciliation**: the adjacent claim "the wiring adds no cleanup call after a `done` dispatch" must be narrowed, not deleted — reap is a **pane-hygiene** call (visual space reclaim), not `.fab-dispatch/` state cleanup; the state dir still has no automatic GC and keeps its two deterministic cleanup moments. Reword so the two statements cannot be read as contradicting.
- **Recovery-policy verb set** (the "verb set is exactly **peek**, **kill**, **restart**, **notify**, **stop**" line): gains **reap**, with the reap-is-not-kill distinction stated (reap fires only on `done`; recovery verbs are untouched).
- No edits needed at the other dispatch sites (`_pipeline.md`, `fab-continue.md`, `fab-adopt.md`) — they reference § CLI-Adapter Dispatch and do not restate the five-state machine; grep-verify during apply anyway (Sibling & Mirror Sweeps).

### 4. Docs swept in the same change (constitution obligations)

- **`src/kit/skills/_cli-fab.md` § fab dispatch**: the family line (`fab dispatch <start|restart|status|wait|logs|kill|clean>` → add `reap`), a new `### reap` subsection (guard, no-op table, exit codes), and the § kill section's cross-reference distinguishing kill (recovery, any state) from reap (hygiene, done-only, knob-gated). Also `_cli-fab.md`'s config-surface notes if they enumerate `dispatch.*` keys.
- **SPEC mirrors** (`docs/specs/skills/`): `SPEC-_preamble.md` and `SPEC-_cli-fab.md` — constitution-required in the same change.
- **`docs/specs/harness-adapters.md`**: § 3 Interactive-pane adapter (done-worker lifecycle now ends in an optional orchestrator-invoked reap), the pipeline-verbs line (~line 350: "read-only-peek, kill, restart, notify, stop" gains reap), and any cleanup/no-GC statements that would now over-claim (e.g. line ~49's "status/kill/clean surfaces").
- **`docs/specs/config.md`**: the 16-row registry count → 17, the "exactly six rows emit a non-null `default`" claim → seven, the scope-taxonomy `both` list, and the Advertise list.
- **Aggregate-spec grep** during apply: `glossary.md`, `skills.md`, `architecture.md` for any dispatch-verb-family enumeration (none found at intake time, but the sweep rule requires the grep, not the memory of it).

### 5. Tests (same change, constitution)

- `internal/dispatch`: table tests for the pure reap decision (mode × state × knob matrix), plus integration coverage in the real-tmux tests following the established private-socket discipline (verified-socket isolation, hard-fail if `$TMUX` set) — reap a done pane worker, assert dispatcher/sibling panes survive and `status` still reads `done`; assert the running/orphaned/knob-false/headless no-ops.
- `cmd/fab`: `dispatch_reap.go` wiring tests (missing record error, no-op reports, exit codes).
- `internal/config`: `GetDispatchReapDone` accessor tests (nil config, unset, explicit true, explicit false; cascade merge of the `dispatch` block with a system-layer `reap_done`).
- `internal/configref` (+ `configupgrade` freeze test if it pins fence bytes): registry row present, key-parity guard passes, Segment renders the third key.

### 6. Ship-time backlog amendment

Amend `fab/backlog.md` [zfl7]: mark part 1 (done-worker pane hygiene) as covered by this change; part 2 (richer pane names/colors) remains as the residual deferred entry with its deferral rationale.

## Affected Memory

- `runtime/dispatch`: (modify) the command family grows to eight subcommands (`reap` requirement section: guard, no-op table, idempotence, exit codes); kill-vs-reap distinction; the cleanup section gains the reap-is-not-state-cleanup clarification (two deterministic state-cleanup moments unchanged); Design Decision entry for the reap posture (space-reclaimed default, knob, rejected zoom/park)
- `_shared/configuration`: (modify) § `dispatch` — third key `reap_done` (default true, scope both, advertise true, `*bool` boundary-case note); the "two dispatch rows are the convention's boundary cases" and shared-Segment claims updated to three keys; scope/advertise enumerations
- `_shared/context-loading`: (modify) § Per-Stage Model Resolution / CLI-adapter wiring mirror — the post-done-result-read reap call, the verb-set enumeration (peek/kill/restart/notify/stop + reap), and the pane-mode subsection's done handling
- `distribution/kit-architecture`: (modify) the fab-go command inventory line for the `fab dispatch` family, if it enumerates subcommands (grep hit at intake time)

## Impact

- **Go** (`src/go/fab/`): `internal/dispatch` (reap core + tests), `cmd/fab/dispatch_reap.go` (new) + `cmd/fab/dispatch.go` (register), `internal/config/config.go` (`ReapDone *bool` + accessor + tests), `internal/configref/configref.go` (17th `[]Field` row + shared Segment + tests), possibly `internal/configupgrade` freeze fixtures
- **Kit skills** (`src/kit/skills/`): `_preamble.md` (§ CLI-Adapter Dispatch step 3/4, recovery verb set, pane subsection), `_cli-fab.md` (§ fab dispatch)
- **Specs** (`docs/specs/`): `skills/SPEC-_preamble.md`, `skills/SPEC-_cli-fab.md`, `harness-adapters.md`, `config.md`
- **Memory** (hydrate): the four files above
- **Backlog**: `fab/backlog.md` [zfl7] amendment at ship
- **Not touched**: dispatch record schema (no new fields), state strings (no new states), Recovery policy, `fab dispatch clean`/archive cleanup, the resolver (`fab resolve-agent` output unchanged), migrations (none needed)

## Open Questions

None — all substantive decisions (posture, knob name/default/scope, mechanism direction, wiring placement, part-2 split) were made and approved in the originating discussion; the remaining details recorded below as Confident assumptions are refinable at plan time without changing scope.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Default posture is space-reclaimed (reap the done pane), not evidence-preserved; zoom/shrink-in-place and break-pane parking rejected for v1 | Discussed — explicit user decision; parking keeps a stub in the column or re-clutters the window list; the knob preserves the opt-out | S:90 R:80 A:90 D:85 |
| 2 | Certain | Policy knob is `dispatch.reap_done`, bool, default `true`, scope `both`, advertise `true`, beside `watchable`/`column_width` | Discussed — user approved name, default, and scope; sibling keys establish the exact pattern | S:90 R:85 A:95 D:90 |
| 3 | Confident | Mechanism is a new `fab dispatch reap <change> <stage>` subcommand owning the whole three-condition guard, with unconditional skill wiring | User approved as the recommended mechanism, explicitly refinable at plan time; Go owns the cascade read, the skill stays dumb | S:85 R:70 A:80 D:75 |
| 4 | Certain | `DispatchConfig.ReapDone` is `*bool` (nil = unset = true) rather than plain `bool` | Codebase-grounded: a default-true bool cannot ride the Go zero value the way default-false `Watchable` does; explicit `false` must be distinguishable from absent | S:65 R:85 A:90 D:85 |
| 5 | Confident | No-op/report surface: exit 0 for all no-op shapes with a one-line reason; non-zero only for missing record / unresolvable change | Mirrors `kill`'s idempotent exit-0 and `wait`'s expiry-exit-0 precedents; exact wording refinable at plan time | S:60 R:85 A:80 D:70 |
| 6 | Confident | `KillPane` (pane-ID keyed, record's `server` socket) covers both pane shapes with no shape branch — a one-pane window dies with its pane | Plain tmux semantics; everything downstream of shape selection is already pane-ID keyed and shape-blind | S:65 R:80 A:85 D:80 |
| 7 | Certain | Registry row joins `dispatch.watchable`'s shared Segment (own Segment empty, like `column_width`); no `configscope` change (parent `dispatch` already `both`); no migration (additive default under presence=intent fence) | Codebase/spec-grounded: the shared-Segment rule and the fence regeneration model directly answer all three | S:70 R:85 A:90 D:85 |
| 8 | Confident | `_preamble.md` step 4's "No cleanup after `done`" is narrowed, not deleted: reap = pane hygiene, `.fab-dispatch/` state cleanup posture unchanged; recovery verb set gains `reap` with the reap-is-not-kill distinction | The two claims must coexist without reading as contradiction; wording is a plan-time editorial call within an approved semantic | S:70 R:80 A:80 D:70 |
| 9 | Certain | Reap changes no state machine: `DerivePaneState` result-precedence keeps a reaped dispatch `done` forever; restart-after-reap is covered by last-attempt-only overwrite; no record schema change, no new state strings | Verified against `internal/dispatch` code and `runtime/dispatch.md` requirements | S:85 R:85 A:95 D:90 |
| 10 | Certain | Part 2 of [zfl7] (richer pane names/colors) is out of scope and stays deferred; backlog amended at ship to mark part 1 covered | Discussed — explicit user split with three documented deferral reasons (title clobbering, no eventing home, user tmux config reach) | S:95 R:90 A:90 D:95 |

10 assumptions (6 certain, 4 confident, 0 tentative, 0 unresolved).
