# Intake: fab Seeds the Operator-Tick Cron Entry

**Change**: 260912-bjrk-fab-seeds-operator-tick-entry
**Created**: 2026-09-12

## Origin

Backlog row `[bjrk]` (fab/backlog.md), decided in a `/fab-discuss` session on 2026-09-12 alongside the always-`skip-if-busy` clock change (tnmm, PR #670). Dispatched promptless (`/fab-proceed` create-new, `{questioning-mode} = promptless-defer`) — no questions asked; every decision below was either made by the user in that discussion or graded as an assumption in the table at the end.

> [bjrk] 2026-09-12: Seed the operator-tick cron entry from the fab binary (reconcile in operator_clock.go: `rk cron list --json` shows no `role:operator` row ⇒ `rk cron add 'operator tick' --backoff --min 3m --max 24m --role operator --deliver skip-if-busy --if-absent respawn --respawn rk --respawn operator --respawn -L --respawn '{server}' --pinned --wake-on agent-state-change --wake-scope server --wake-debounce 60s`, then continue editing it as today), so fab owns the whole entry lifecycle and §2 Init step 4 stops STOPping on a missing entry (run the reconcile instead). Skill never composes the add argv by hand. BLOCKED ON run-kit backlog row [ntde] STEP 1: rk cron add/edit have no `--wake-on` flags yet — seeding without `wake_on` would lose the pickup-latency edge. Rollout: fab seed ships first (both sides key on the `role:operator` row — no duplicate in the overlap), then `rk operator` drops `seedOperatorTick` (run-kit STEP 2).

**The blocker is cleared.** run-kit shipped STEP 1 in v3.19.52 (PR #954, change `260911-ntde-cron-wake-on-flags-seed-tuning`): `rk cron add` and `rk cron edit` now carry `--wake-on <event>` (only `agent-state-change`), `--wake-scope <s>` (only `server`, default `server`), and `--wake-debounce <dur>` (default `1m0s`). Verified live against the installed rk:

```
      --wake-debounce duration   Hold a --wake-on fire this long after the entry's own newest delivery (default 1m0s)
      --wake-on string           Also fire on an actionable agent-state edge (only: agent-state-change)
      --wake-scope string        Fingerprint scope for --wake-on (only: server) (default "server")
```

`rk cron add --help`'s own example block already shows the exact operator-tick argv this change will emit:

```
rk cron add "operator tick" --backoff --min 3m --max 24m --wake-on agent-state-change --deliver skip-if-busy --role operator --if-absent respawn --respawn rk --respawn operator --respawn -L --respawn '{server}' --pinned
```

run-kit's backlog row `[pfo3]` is the run-kit half this change unblocks: once fab's seed ships, `rk operator` deletes `seedOperatorTick` / `operatorTickEntrySpec` and its three seed tests and becomes launcher-only. `[pfo3]` is **not** in this change's scope.

## Why

**1. The problem.** The operator-tick cron entry's lifecycle is split across two repos. fab's schedule reconcile (`src/go/fab/cmd/fab/operator_clock.go`) already owns almost all of it — the mute/unmute flip on the tracked predicate, the schedule derive from the tracked set, and the deliver policy — but **seeding** still lives in run-kit's `rk operator` launcher (`seedOperatorTick` → `operatorTickEntrySpec`). Two owners, one entry.

**2. The consequence of not fixing it.** Split ownership means every policy change has to be mirrored across repos in lockstep. Today's own change proves it: tnmm (fab #670) moved the entry to `deliver: skip-if-busy` and the `3m→24m` ladder, and run-kit had to mirror the same values into `operatorTickEntrySpec` (#954) so a freshly seeded entry would not start life on the old policy and wait for fab's first reconcile to converge it. Every future tuning of the ladder, the deliver policy, or the wake block pays that cross-repo tax again, and any skew shows up as an entry that silently ticks on the wrong cadence between seed and first reconcile.

There is a second, sharper failure today: a user who runs `rk cron rm <id>` mid-session (or whose entry is garbage-collected) has **no operator-tick entry and no way back without restarting the operator** — fab's `resolveOperatorCronRow()` finds zero candidates and every clock path degrades to a silent no-op, so the operator simply stops ticking. §2 Init's current answer is a STOP telling the user to run `rk operator`, i.e. relaunch.

**3. Why this approach.** The user's decision: **rk stays generic substrate and launcher; fab owns the operator-tick entry end-to-end.** The seed is the last piece of the entry that is not fab's. Seeding from **fab's reconcile** (rather than from a one-shot `fab operator` init verb) is what buys the mid-run healing: every `track` mutation and every `tick-start --diff` already runs the reconcile, so a `rk cron rm` heals on the next tick with no restart. A one-shot init verb would only heal at startup.

The rollout is safe by construction: both sides key idempotency on the `role:operator` row, so during the overlap window (fab's seed shipped, run-kit's `seedOperatorTick` still present until `[pfo3]`) whichever runs first creates the row and the other finds it — no duplicate. run-kit's interim seed already writes 3m/24m/skip-if-busy, so the convergence edit on first reconcile is usually a no-op.

## What Changes

### 1. `operator_clock.go` — seed-if-missing inside the row resolution

Today `resolveOperatorCronRow()` lists `role:operator` candidates and returns `(row, false)` for **three different reasons**: an rk failure / unparseable output, **zero candidates**, or an unresolved tie (several `role:operator` rows, none or more than one named `operator tick`). Only the **zero-candidates** case may seed — seeding into a tie would add a third row, and seeding when rk itself failed would be a blind write.

The refactor splits the resolution into a list step and a pick step, then adds a seeding wrapper:

```go
// listOperatorCronRows returns every role:operator candidate row.
// ok=false only when rk failed or the output was unparseable — an empty
// server is (nil, true), which is what makes "zero candidates" seedable.
func listOperatorCronRows() ([]operatorCronRow, bool)

// pickOperatorCronRow applies the existing selection: exactly one candidate
// wins outright; otherwise name == operatorCronName is the tiebreak.
func pickOperatorCronRow(candidates []operatorCronRow) (operatorCronRow, bool)

// resolveOperatorCronRow keeps its current signature and behaviour exactly
// (list + pick, no side effect) — it stays the pure read.
func resolveOperatorCronRow() (operatorCronRow, bool)

// ensureOperatorCronRow is resolve + seed-if-zero + re-resolve ONCE.
func ensureOperatorCronRow() (operatorCronRow, bool)
```

`ensureOperatorCronRow()`:

1. `listOperatorCronRows()` — on `ok == false` (rk failed / unparseable), return `(zero, false)`. **No seed.**
2. If the candidate list is **non-empty**, return `pickOperatorCronRow(candidates)` — a tie stays a silent no-op, **no seed**.
3. If the list is **empty**, run exactly one `rk cron add …` (argv below). On error, return `(zero, false)` — a failed add is a silent no-op like every other rk failure, with no retry inside the call.
4. Re-run `listOperatorCronRows()` + `pickOperatorCronRow()` **once** and return its result (this is what picks up the new entry's id, and what makes the `--name` tiebreak matter on a server that already had an unrelated `role:operator` row appear between the two reads).

All four existing clock entry points switch from `resolveOperatorCronRow()` to `ensureOperatorCronRow()`:

| Call site | Today | After |
|---|---|---|
| `syncOperatorClock(before, after)` (edge flip) | resolve | ensure |
| `muteOperatorClockIfUntracked(data)` (tick-start, loud direction) | resolve | ensure |
| `reconcileOperatorSchedule(data)` (B3 schedule) | resolve | ensure |
| `fab operator clock sync` (new, §3 below) | — | ensure |

This is what gives "every reconcile entry point heals". Note the one residual gap, accepted deliberately: `reconcileOperatorSchedule` returns **early** when `deriveOperatorSchedule` reports `ok == false` (an empty/all-done tracked set), before it ever resolves a row — so a `track` mutation that neither flips the predicate nor has anything tracked issues no rk call at all and seeds nothing. That state heals on the next `tick-start --diff` (`muteOperatorClockIfUntracked` ensures on every untracked tick) and on the next `fab operator clock sync`. Do **not** hoist the seed above the derive gate by adding an unconditional pre-reconcile ensure call — that would add a third `rk cron list --json` per mutation for a case two other paths already cover.

**Seeding is not conditioned on tracked-ness** (user decision): whenever a path resolves the row and finds none, it seeds. The normal mute rule then applies in the same invocation — on an untracked state `muteOperatorClockIfUntracked` sees the freshly seeded (unmuted) row and issues exactly one `rk cron mute <id>`, which is precisely what rk's seed + fab's mute do together today. The invariant preserved is "one entry per server always exists once an operator has run".

### 2. The add argv

Everything is explicit — nothing is left to an rk default, so a future change to an rk default cannot silently drift fab's entry. Emitted through the existing `rkCronRunner` seam (argv-only, never a shell string, so `{server}` needs no quoting):

```go
argv := []string{
    "cron", "add", "operator tick",
    "--name", operatorCronName,
    "--backoff", "--min", operatorBackoffMin, "--max", operatorBackoffMax,
    "--role", "operator",
    "--deliver", operatorDeliver,
    "--wake-on", "agent-state-change",
    "--wake-scope", "server",
    "--wake-debounce", "60s",
    "--if-absent", "respawn",
    "--respawn", "rk", "--respawn", "operator", "--respawn", "-L", "--respawn", "{server}",
    "--pinned",
}
```

Rendered (for the docs and the test's expected argv):

```
rk cron add "operator tick" --name "operator tick" --backoff --min 3m --max 24m --role operator --deliver skip-if-busy --wake-on agent-state-change --wake-scope server --wake-debounce 60s --if-absent respawn --respawn rk --respawn operator --respawn -L --respawn {server} --pinned
```

Specific requirements on this argv:

- **Use the existing constants**, never literals: `operatorCronName` (`"operator tick"`), `operatorBackoffMin` (`"3m"`), `operatorBackoffMax` (`"24m"`), `operatorDeliver` (`"skip-if-busy"`). tnmm's policy values stay single-sourced — a future ladder/deliver change must not have to be edited in two places in the same file. A new constant for the wake block (e.g. `operatorWakeEvent`/`operatorWakeScope`/`operatorWakeDebounce`) is welcome but optional.
- **`--name` is passed explicitly.** rk's `--name` "defaults to a prompt prefix", which for the prompt `operator tick` would very likely already be `operator tick` — but `operatorCronName` is the tiebreak fab resolves by, so it is set explicitly rather than inferred from rk's prefix rule.
- **No `-L <server>`.** rk derives the server from `$TMUX`, exactly as fab's existing `cron mute` / `cron edit` calls do. The clock always runs from inside the operator's own pane.
- **The respawn argv is rk's launcher and is unchanged**: `rk operator -L {server}`, one `--respawn` per element, `{server}` substituted by rk at fire time. Only the *seed* moves repos; the launcher does not.
- **Posture unchanged**: `exec.LookPath`-gated, `rkCronTimeout` (5s)-bounded, argv-only, fail-silent. A failed `add` never changes a verb's exit code or stdout.

`operatorCronRow` gains one field so the new verb can print rk's own rendering instead of recomposing it:

```go
ScheduleSummary string `json:"schedule_summary"`
```

### 3. New verb: `fab operator clock sync`

The skill currently hand-parses `rk cron list --json` in two places (§2 Init step 4 and the per-tick read in §4 Tick Behavior step 1) — including the `{"ok":true,"result":[…]}` envelope form. With seeding in the binary, the skill also needs a way to *trigger* the reconcile without advancing `tick_count` (`fab operator track list` is read-only and runs no reconcile; `tick-start --diff --quiet` runs it but increments the tick counter). One verb solves both: heal and read in a single command.

```
fab operator clock sync
```

Registered in `operatorCmd()` alongside `operatorTickStartCmd()` / `operatorTimeCmd()` / `operatorStateCmd()` / `operatorTrackCmd()` / `operatorBranchMapCmd()` — a new `operatorClockCmd()` parent with a `sync` subcommand (`cobra.NoArgs`, no flags). Implementation file: a new `cmd/fab/operator_clock_sync.go` (keeps `operator_clock.go` to the side-effect machinery), tests in `operator_clock_test.go` or a sibling.

**Behaviour**, in order:

1. Load the operator state file (the same loader the other read verbs use; `fab operator state` at Init step 1 has already created the skeleton when missing, and this verb must not be the thing that creates it — a read failure is a plain non-zero exit).
2. `ensureOperatorCronRow()` — seeds when the server has no `role:operator` row.
3. **Mute rule, level-wise in BOTH directions** — a new `reconcileOperatorMute(data)`: untracked and the row is not muted ⇒ one `rk cron mute <id>`; tracked and the row *is* muted (indefinite mute or live lease) ⇒ one `rk cron mute <id> --off`; otherwise no call. This is what lets §2 Init drop its hand-written "issue `rk cron mute <id> --off` when muted while tracked work exists" instruction. `muteOperatorClockIfUntracked` is **left exactly as it is** — `tick-start --diff` keeps its one-direction level reconcile (the unmute direction there stays edge-triggered via `syncOperatorClock`); changing tick-start's mute semantics is out of scope for this change.
4. `reconcileOperatorSchedule(data)` — unchanged, exactly one `rk cron edit` only when the live row differs from the derived schedule.
5. Re-resolve the row (a plain `resolveOperatorCronRow()` — the entry exists by now) and print it as YAML on stdout.

**Output** — exactly five keys, no more (`schedule_summary` and `deliver` are rk's own strings, copied verbatim; `muted_until` is `null` unless a lease is live):

```yaml
id: hk6c
schedule_summary: backoff 3m→24m
deliver: skip-if-busy
muted: false
muted_until: null
```

**Exit contract**: exit 0 with that document when a row resolved. When the row can neither be resolved nor seeded (rk broken, an unresolved tie, a failed `add`), exit **non-zero** with a single stderr line:

```
ERROR: could not resolve or seed the operator-tick cron entry
```

This is the verb's one deliberate departure from the fail-silent posture, and it is correct: fail-silent protects *side effects* riding on other verbs (a `track add` must never fail because rk is down); `clock sync` is an explicit user/agent-facing read whose whole purpose is to answer "what is the clock". The non-zero exit is what preserves §2 Init's STOP gate after the old STOP text is removed. The reconcile calls it makes internally stay fail-silent.

### 4. Skill: `src/kit/skills/fab-operator.md`

**§2 Init step 4** (~line 113) — the STOP `Error: no operator-tick cron entry on this server — run rk operator to seed it` **goes away**, and the hand-parse of `rk cron list --json` (including the envelope-form instruction and the manual unmute) is replaced by the verb. Replacement intent:

> 4. Sync and read the clock — run `fab operator clock sync`: it seeds the operator-tick entry when this server has none, reconciles the mute against the tracked set (muting an untracked set, clearing a standing mute or lease while work is tracked), applies the derived schedule, and prints the resolved row as YAML: `id`, `schedule_summary`, `deliver`, `muted`, `muted_until` (`muted` is the effective state — an indefinite mute or a live lease; `muted_until`, unix seconds, is present only while a lease is live). A non-zero exit STOPs: `Error: the operator-tick cron entry could not be resolved or seeded — check rk cron list`.

**§2 Init step 5** — the ready-line template is unchanged (`Operator ready. Clock: rk cron "operator tick" · {schedule_summary} · {deliver}[ · muted[ until HH:MM]]`); only its source changes to step 4's YAML fields. There is still no `Clock: none` form — the non-zero exit is the only missing-entry path.

**§4 The Clock** lead-in (~line 171) — `seeded idempotently by rk operator` becomes present truth without narrating the transition: the entry is seeded by fab's clock reconcile when the server has none, and `rk operator`'s launcher also seeds it (until run-kit retires its copy) — both key on the `role:operator` row, so there is never a duplicate. The `schedule`/`wake_on`/`target`/`payload`/`deliver`/`if_absent`/`respawn`/`pinned` steady-state YAML block **stays as it is**.

**§4 Ownership** paragraph (~line 186) — rewrite the first sentence: fab's reconcile seeds the entry when absent (one `rk cron add` with the full shape), then manages it as today (mute/unmute on the tracked predicate, derived schedule via one `rk cron edit` on change). `rk cron rm` stays the user's; `rk cron add` is no longer "the user's and `rk operator`'s" — the skill still never composes an `add` by hand, the binary does.

**§4 Tick Behavior step 1** (~line 305) and **§4 Status Frame Format** (~line 359) — the per-tick `rk cron list --json` read becomes `fab operator clock sync`. Rationale for spending the reconcile every tick rather than a bare read: it makes the seed self-healing on **every** tick regardless of tracked-ness (closing the residual gap in §1 above), and it removes the last place the skill hand-parses rk's JSON envelope. Cost is two extra local rk calls per tick and zero extra `rk cron edit`s in steady state (the reconcile is on-change-only). The Status Frame Format table's "copied verbatim from the per-tick `rk cron list --json` read (tick step 1)" note updates to name the verb.

**§9 Key Properties, Cadence row** (~line 821) — `seeded by rk operator` becomes seeded by fab's clock reconcile; `live cadence rendered from rk cron list --json` becomes rendered from `fab operator clock sync`.

Constitution V applies throughout: the skill cites the verb, `_cli-fab-operator.md`, and rk commands — never fab-kit's `docs/`, `src/go/`, or run-kit paths.

### 5. Skill: `src/kit/skills/_cli-fab-operator.md`

- **Contents** list gains `### fab operator clock sync`.
- New `### fab operator clock sync` section documenting the signature, the five-key YAML output, the seed-if-missing step, the both-direction mute reconcile, the schedule reconcile, and the non-zero exit contract.
- The shared **Clock side effect** paragraph (~line 161) gains the seed step: the row is resolved per call from `rk cron list --json`, and **zero `role:operator` rows** (not a tie, not an rk failure) triggers exactly one `rk cron add` with the full steady-state shape before the resolve is retried once; a failed add is a silent no-op. State that the add argv is fully explicit and name the values (backoff 3m→24m, `role:operator`, `skip-if-busy`, `wake_on agent-state-change`/`server`/`60s`, `if_absent respawn` with `rk operator -L {server}`, pinned).
- Constitution's CLI⇒docs rule makes this partial mandatory for the new verb.

### 6. Memory: `docs/memory/runtime/operator.md`

- **The clock paragraph** (~line 132): `rk operator idempotently seeds a pinned operator-tick cron entry` → fab's clock reconcile seeds it when the server carries no `role:operator` row (`rk operator`'s launcher also seeds it until run-kit's row `[pfo3]` retires that copy; both key on the row, so no duplicate). Record the zero-candidates-only rule, the one-retry re-read, the explicit argv, and the new `fab operator clock sync` verb (Init + per-tick read). Update the ready-line sentences: the STOP is replaced by `clock sync`'s non-zero exit, and Init's hand-unmute is replaced by the verb's both-direction mute reconcile. Update the per-tick clock-read sentence to name the verb. Present truth only — no "used to be seeded by rk" narration.
- **Design decision "A Missing Operator-Tick Entry Is a STOP, Not a Clock-Less Ready Line"** (~line 555): superseded/rewritten. The new decision: a missing entry is **seeded, not stopped on**; the STOP survives only as `clock sync`'s non-zero exit when the entry can neither be resolved nor seeded, and the ready line still has no `Clock: none` variant. Its old **Rejected** line ("auto-seeding from the skill — `rk cron add` is the user's and `rk operator`'s") is preserved in spirit and restated precisely: auto-seeding *from the skill* is still rejected (the skill never composes an `add`); the **binary** seeds. Add `*Updated by*: 260912-bjrk-fab-seeds-operator-tick-entry`.
- **Design decision "Binary Derives the Cron Schedule From the Tracked Set"** (~line 601): extend with the seed rule — the reconcile owns the entry's whole lifecycle (seed, mute/unmute, schedule), keyed on the `role:operator` row; seeding fires only on zero candidates. **Rejected**: seeding from `rk operator` only (cross-repo policy drift — tnmm had to be mirrored into run-kit #954); a one-shot `fab operator` init verb only (no mid-run healing after a user `rk cron rm` — the reconcile runs on every `track` mutation and every tick, so healing is free). Add `*Updated by*: 260912-bjrk-fab-seeds-operator-tick-entry`.
- **Design decision "Cadence Is Re-Read Every Tick"** (~line 618): the per-tick read is now `fab operator clock sync` rather than a raw `rk cron list --json`; note the reconcile-plus-read rationale and that the cost stays one cheap local command.

### 7. Spec: `docs/specs/operator.md`

Add a `v12` row to the Version History table naming fab-owned clock seeding:

```
| v12 | fab owns the operator-tick entry end-to-end — the clock reconcile seeds it when absent, `fab operator clock sync` heals and reads it for Init and the per-tick frame (2026-09-12, change 260912-bjrk) |
```

This is human-curated spec content (Constitution VI) — it records intent, not generated state.

### 8. Tests: `src/go/fab/cmd/fab/operator_clock_test.go`

`stubRkCron(t, listJSON, listErr, muteErr)` serves one fixed `listJSON` for every `cron list` call, so it cannot express "empty, then seeded". Add a **sibling** stub rather than changing the existing one (every current case must stay green untouched):

```go
// stubRkCronSeeding serves emptyJSON for the FIRST `cron list --json` and
// seededJSON for every later one, records every argv (including the `cron
// add`), and fails the add when addErr is set.
func stubRkCronSeeding(t *testing.T, seededJSON string, addErr error) *[][]string
```

Cases to add:

| Case | Expectation |
|---|---|
| No `role:operator` row, tracked set live | exactly one `cron add` with the **full expected argv** (asserted element-by-element via a `wantAdds` helper mirroring `wantMutes`/`wantEdits`), then the normal behaviour against the seeded row |
| No row, **untracked** state at `tick-start` | one `cron add`, then exactly one `cron mute <id>` on the seeded row (the decision-3 invariant) |
| Row already present | **zero** `cron add` calls (every existing fixture exercises this implicitly; assert it explicitly on one) |
| `cron add` fails | silent no-op — no panic, no `edit`, no `mute`, verb exit unchanged |
| `cron list` fails / `{"ok":false,…}` envelope | **zero** `cron add` (never seed on an rk failure) |
| Two `role:operator` rows, ambiguous tiebreak | **zero** `cron add` (never seed into a tie) |
| Seeded row resolved by the `--name` tiebreak | a second unrelated `role:operator` row present in the post-seed list; the `operator tick` row wins |
| `fab operator clock sync` happy path | prints the five YAML keys with rk's `schedule_summary`/`deliver` verbatim; exit 0 |
| `fab operator clock sync`, tracked + muted row | one `cron mute <id> --off`, then the printed row |
| `fab operator clock sync`, unresolvable | exit non-zero, the single stderr line, no panic |

Run `gofmt -l` on every touched `.go` file and `go test ./cmd/fab` from `src/go/fab` before finishing (project code-quality: scope to the affected package first).

### 9. Non-goals

- **run-kit changes.** Deleting `seedOperatorTick` / `operatorTickEntrySpec` / `EnsureRoleEntry` and their three tests is run-kit backlog row `[pfo3]`, gated on this change shipping. Nothing in this change touches the run-kit repo.
- **Touching the deliver policy or the ladder bounds** (`skip-if-busy`, 3m→24m) — tnmm's, already shipped.
- **Changing the respawn argv** (`rk operator -L {server}`).
- **Any `rk cron rm`.** fab never removes the entry; removal stays the user's.
- **Changing `muteOperatorClockIfUntracked`'s one-direction semantics at `tick-start`**, or adding a `clock:` block to the `tick-start --diff` document (the alternative to the per-tick `clock sync` read — a possible follow-up, not this change).
- **A `--json` flag or a `seeded:` key on `clock sync`** — the five-key YAML is the whole surface.
- **Marking backlog row `[bjrk]` done** — `/fab-archive` does that.

## Affected Memory

- `runtime/operator`: (modify) — the clock paragraph (seed ownership, the new `fab operator clock sync` verb at Init and per tick, the removed STOP, the both-direction mute reconcile), the design decision "A Missing Operator-Tick Entry Is a STOP, Not a Clock-Less Ready Line" (superseded — a missing entry is seeded), "Binary Derives the Cron Schedule From the Tracked Set" (extended with the seed rule + rejected alternatives), and "Cadence Is Re-Read Every Tick" (the per-tick read is now the verb).

## Impact

**Go (`src/go/fab/cmd/fab/`)**

- `operator_clock.go` — split `resolveOperatorCronRow` into `listOperatorCronRows` + `pickOperatorCronRow`; add `ensureOperatorCronRow` + the `rk cron add` argv builder; add `reconcileOperatorMute`; add `ScheduleSummary` to `operatorCronRow`; repoint `syncOperatorClock`, `muteOperatorClockIfUntracked`, `reconcileOperatorSchedule` at `ensureOperatorCronRow`. Update the file's header comment block (lines 14–25), which currently states the entry is "seeded by `rk operator`".
- `operator_clock_sync.go` *(new)* — `operatorClockCmd()` + `runOperatorClockSync`.
- `operator.go` — register `operatorClockCmd()` in `operatorCmd()`'s `AddCommand` list.
- `operator_clock_test.go` — `stubRkCronSeeding` + `wantAdds` + the cases above.

**Kit skills (`src/kit/skills/`)** — `fab-operator.md` (§2 Init steps 4–5, §4 The Clock lead-in + Ownership, §4 Tick Behavior step 1, §4 Status Frame Format, §9 Key Properties Cadence row); `_cli-fab-operator.md` (Contents, new `### fab operator clock sync`, the shared **Clock side effect** paragraph). Never edit the deployed copies under `.agents/skills/` or `.claude/skills/`.

**Docs** — `docs/memory/runtime/operator.md`; `docs/specs/operator.md`.

**External dependency** — requires rk ≥ v3.19.52 for the `--wake-on` / `--wake-scope` / `--wake-debounce` flags on `cron add`. An older rk rejects the add as an unknown flag, which the fail-silent posture swallows: the operator then behaves exactly as it does today (no entry seeded, the existing `rk operator` seed still covers it). No version probe is added — the fail-silent path is the version-skew arm.

**Scale** — roughly 4 Go files (one new), 2 skill files, 2 doc files. Expected LIGHT lane (≤5 tasks); the `clock sync` verb may push it to the FULL lane, which is fine.

## Open Questions

None. The dispatch description and backlog row `[bjrk]` settled every decision; the remaining design choices are graded below rather than deferred.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Add a `fab operator clock sync` verb (seed-if-missing + mute reconcile + schedule reconcile + print the row) rather than keeping the skill's hand-parsed `rk cron list --json` read | Dispatch instruction asked for this grade explicitly. `track list` is read-only and runs no reconcile; `tick-start --diff --quiet` runs it but advances `tick_count` — neither can be Init's heal-and-read. The verb also removes the skill's last rk-JSON hand-parse (the D5 envelope that just broke in zx5k). CLI change ⇒ `_cli-fab-operator.md` + tests, per the constitution | S:80 R:70 A:75 D:65 |
| 2 | Certain | The exact add argv, with `--name "operator tick"` passed explicitly and every value from the existing `operatorCronName`/`operatorBackoffMin`/`operatorBackoffMax`/`operatorDeliver` constants | Verified live against `rk cron add --help` (its own example block shows this argv); rk's `--name` "defaults to a prompt prefix", and `operatorCronName` is the tiebreak fab resolves by, so it is stated rather than inferred. Constants keep tnmm's policy single-sourced | S:95 R:80 A:90 D:85 |
| 3 | Certain | Seed on **zero** `role:operator` candidates only — never on an rk failure/unparseable output, never on an ambiguous tie | A tie means rows already exist; adding another makes it worse. An rk failure carries no information about what is on the server. Requires splitting the resolver's three no-op reasons apart | S:70 R:80 A:90 D:80 |
| 4 | Certain | Seed regardless of tracked-ness; the normal mute rule then mutes a freshly seeded entry on an untracked set | User decision in the dispatch description; preserves the "one entry per server once an operator has run" invariant `rk operator` guarantees today | S:90 R:75 A:85 D:80 |
| 5 | Confident | Hook the seed into a new `ensureOperatorCronRow()` used by all four resolve call sites, rather than inline in `reconcileOperatorSchedule` or hidden inside `resolveOperatorCronRow` | `reconcileOperatorSchedule` returns early on an empty tracked set before resolving, so an inline seed there would never fire on an untracked state — contradicting row 4. An explicit `ensure*` name keeps the mutation out of a function named `resolve*`, and costs no extra `rk cron list` in the common path | S:60 R:75 A:85 D:65 |
| 6 | Confident | `clock sync` reconciles the mute in **both** directions via a new `reconcileOperatorMute(data)`; `muteOperatorClockIfUntracked` (tick-start) is left one-directional and untouched | Both directions are what lets §2 Init drop its hand-written unmute instruction. Making tick-start symmetric would change existing behaviour and its tests for no stated need — out of scope | S:55 R:70 A:80 D:60 |
| 7 | Confident | `clock sync` exits non-zero with one stderr line when the row can neither be resolved nor seeded; its internal reconcile calls stay fail-silent | Fail-silent protects side effects riding on other verbs; an explicit read verb whose purpose is "what is the clock" must be able to say it failed. The non-zero exit is what preserves Init's STOP gate after the old STOP text is removed | S:50 R:75 A:80 D:60 |
| 8 | Confident | The per-tick clock read (§4 Tick Behavior step 1 + Status Frame Format) switches to `fab operator clock sync`, accepting one redundant reconcile per tick | Dispatch instruction; it makes the seed self-heal on every tick regardless of tracked-ness and removes the last rk-JSON hand-parse. Cost is two extra local rk calls and zero extra `rk cron edit`s in steady state. Rejected alternative (recorded as a follow-up, not this change): emit a `clock:` block from `tick-start --diff` | S:60 R:80 A:75 D:55 |
| 9 | Confident | `clock sync` prints exactly five YAML keys (`id`, `schedule_summary`, `deliver`, `muted`, `muted_until`) — no `seeded:` key, no `--json` flag | Those five are exactly what the ready line and the frame header consume; anything else is surface with no reader. Trivially extendable later | S:70 R:90 A:80 D:70 |
| 10 | Confident | `docs/specs/operator.md` gains a `v12` Version History row naming fab-owned clock seeding | Backlog row `[bjrk]` names the file as a doc site, and the version-history row (each naming its change, e.g. v11 → 260911-4a8m) is the established pattern there. Whether an ownership move rates a version bump is a curation judgment apply may reverse | S:35 R:85 A:45 D:45 |
| 11 | Confident | No rk version probe; an rk older than v3.19.52 rejects the `--wake-on` flags and the fail-silent path swallows it, leaving today's behaviour | The clock's whole posture is fail-silent on rk failures; a version probe would be a second copy of an rk-side contract — exactly the drift the rk-hard-dependency decision rejected. The overlap-window `rk operator` seed still covers those users | S:55 R:85 A:80 D:65 |
| 12 | Certain | run-kit is untouched; `[pfo3]` (delete `seedOperatorTick` + its three tests) stays run-kit's, and the overlap window is left in place | Both sides key idempotency on the `role:operator` row, so the overlap creates no duplicate; run-kit's interim seed already writes 3m/24m/skip-if-busy, so the convergence edit is usually a no-op. Stated as a Non-Goal, and `[pfo3]` is explicitly gated on this change shipping | S:90 R:80 A:85 D:85 |

12 assumptions (4 certain, 8 confident, 0 tentative, 0 unresolved).
