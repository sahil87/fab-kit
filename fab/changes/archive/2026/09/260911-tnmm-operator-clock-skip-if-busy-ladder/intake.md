# Intake: Operator Clock — Single `skip-if-busy` Deliver Policy, 3m→24m Backoff Ladder, Debounce Doc Fix

**Change**: 260911-tnmm-operator-clock-skip-if-busy-ladder
**Created**: 2026-09-12

## Origin

Decided in a `/fab-discuss` session on 2026-09-12, dispatched through `/fab-proceed` in promptless-defer mode. The user's framing:

> Operator clock — single `skip-if-busy` deliver policy, 3m→24m backoff ladder, debounce doc fix.
>
> The operator-tick cron entry's deliver policy is a two-way table today: fab's schedule reconcile derives `skip-if-busy` for tracked sets containing any shell/agent item, but `immediate` for pane/none-only sets (the backoff branch). Under `immediate` a tick can land mid-turn in the operator's chat (queued as a mid-turn message on Claude; behaviour on codex/gemini under mid-turn injection is less predictable). Simplify — one policy, always `skip-if-busy`.
>
> The `1m→30m` backoff ladder is too aggressive at both ends. For pane/none sets `wake_on: agent-state-change` does the real pickup, so the ladder is a fallback poll; a 1m floor mostly buys token spend. `3m→24m` (a clean doubling ladder 3, 6, 12, 24 *if* rk doubles — unverified).
>
> Separately: the live `rk cron list --json` and run-kit's `operatorTickEntrySpec()` both show `wake_on.debounce = 60s`, while `fab-operator.md` and `docs/memory/runtime/operator.md` quote `debounce: 10s` — a stale doc, fix to `60s`.

The same discussion produced backlog row `[bjrk]` (seed the entry from the fab binary), which is explicitly **out of scope** here and blocked on run-kit backlog row `[ntde]`.

## Why

**Problem 1 — the deliver policy is a two-way table for no benefit.** `deriveOperatorSchedule` (`src/go/fab/cmd/fab/operator_clock.go`) returns `deliver: immediate` on the backoff branch (a tracked set of only `pane`/`none` items) and `deliver: skip-if-busy` on the `every` / `idle-every` branches (any `shell`/`agent` item). `immediate` means rk delivers the `operator tick` text into the operator pane regardless of what the operator is doing; on Claude that queues as a mid-turn message, and on codex/gemini the behaviour under mid-turn injection is less predictable. The split also means the policy the operator runs under changes silently the moment a shell/agent item joins or leaves the tracked set — a behaviour the operator cannot see and nobody asked for. One policy is simpler to reason about, simpler to document (the derive table collapses a column), and removes the mid-turn-injection class entirely.

**Problem 2 — the ladder is mis-tuned at both ends.** The backoff branch is the *fallback poll* for a pane-only fleet: real pickup comes from the entry's `wake_on: { event: agent-state-change }`, which fires the moment any agent's state flips. A 1-minute floor therefore buys almost no latency over the wake — it mostly buys token spend, one full operator turn per minute on an idle fleet. At the other end a 30-minute ceiling is longer than the operator's own 10-minute full-frame refresh window, so the ceiling can starve the periodic full frame. `3m → 24m` moves the floor to a cost that is defensible on an idle fleet and pulls the ceiling under half an hour, while staying a clean doubling sequence (3, 6, 12, 24).

**Problem 3 — a stale documented value.** Both `src/kit/skills/fab-operator.md` §4 and `docs/memory/runtime/operator.md` quote the entry's `wake_on.debounce` as `10s`. The live entry (`rk cron list --json`) and run-kit's `operatorTickEntrySpec()` both carry `60s`. A doc quoting a substrate value it does not own must quote it correctly or it is worse than silent.

**Why this approach over the alternatives.** The alternative deliver policy for the backoff branch is `when-idle`, which *holds* the fire until the operator goes idle rather than dropping it — no tick is ever lost, and no ladder rung is ever paid twice. It was considered and rejected in favour of policy uniformity: a single policy across every branch and every override is the property being bought, and a third policy on one branch would rebuild the two-way table under a different name. Keeping `immediate` was rejected outright (mid-turn injection, plus the two-way table). The accepted cost is stated below and recorded as a design decision.

**The one accepted cost.** Under `skip-if-busy`, a tick that fires while the operator is mid-turn is **dropped**, not queued. On the `every`/`idle-every` branches that costs at most one interval. On the *backoff* branch a drop at a long rung costs up to that rung — now bounded at **≤ 24 minutes** (it would have been ≤ 30m under the old ceiling). `wake_on: agent-state-change` still fires independently of the ladder, so a real fleet event does not wait for the next rung; the bound applies only to the fallback poll on a fleet where nothing is changing state. The user was shown this and held the decision.

## What Changes

### 1. Go — `src/go/fab/cmd/fab/operator_clock.go`

**Constants** (current lines 37–44):

```go
	// operatorBackoffMin/Max are the derived backoff bounds for a tracked set
	// of only pane/none items (B3).
	operatorBackoffMin = "1m"
	operatorBackoffMax = "30m"
	// operatorDeliverImmediate is the derived deliver policy for pane/none
	// sets; skip-if-busy covers shell/agent sets (and clock overrides).
	operatorDeliverImmediate  = "immediate"
	operatorDeliverSkipIfBusy = "skip-if-busy"
```

becomes:

```go
	// operatorBackoffMin/Max are the derived backoff bounds for a tracked set
	// of only pane/none items (B3) — a fallback poll behind
	// wake_on: agent-state-change, not the primary pickup path.
	operatorBackoffMin = "3m"
	operatorBackoffMax = "24m"
	// operatorDeliver is the deliver policy for EVERY derived branch and for
	// clock overrides: one policy, never a per-branch table. skip-if-busy
	// drops a tick that would land mid-turn rather than injecting into it.
	operatorDeliver = "skip-if-busy"
```

`operatorDeliverImmediate` is **deleted**. `operatorDeliverSkipIfBusy` is **renamed** to `operatorDeliver` (a single-policy constant should not read as one of a set) — this is a mechanical rename with three call sites in this file plus one in `operator_track.go`.

**`deriveOperatorSchedule`** (current lines ~235–278): the doc comment's B3 table description changes from "only pane/none items → backoff 1m→30m, deliver immediate; any shell/agent item → … both deliver skip-if-busy" to a statement that the schedule *kind* varies by tracked set while the deliver policy is uniform `skip-if-busy`. The three `return operatorSchedule{…}` statements all carry `deliver: operatorDeliver`:

```go
	if minEvery == "" {
		return operatorSchedule{kind: "backoff", deliver: operatorDeliver}, true
	}
	if epoch {
		return operatorSchedule{kind: "idle-every", every: minEvery, deliver: operatorDeliver}, true
	}
	return operatorSchedule{kind: "every", every: minEvery, deliver: operatorDeliver}, true
```

**`cronRowMatches`** and **`reconcileOperatorSchedule`** need no structural change — they already read the constants, so the emitted argv on the backoff branch becomes `rk cron edit <id> --backoff --min 3m --max 24m --deliver skip-if-busy`.

**Clock-override path is untouched behaviourally**: the override already writes `deliver: skip-if-busy` (`operator_track.go:1012`), and the derive's override short-circuit passes `override.Deliver` through verbatim. Only the constant's *name* changes at that call site.

### 2. Go tests — `src/go/fab/cmd/fab/operator_clock_test.go`

Grep-verified: this is the only test file carrying the affected literals.

- **Fixtures, lines 16 and 20** — `cronListBackoffJSON` and `cronListBackoffMutedJSON`: `"min":"1m0s","max":"30m0s"` → `"min":"3m0s","max":"24m0s"`, and `"deliver":"immediate"` → `"deliver":"skip-if-busy"`. These two fixtures exist to represent *the entry already on the derived pane/none schedule*, so they must track the new derived values or the "equal row issues nothing" cases invert.
- **Line ~259** (`TestClockSync_NonFlippingMutationsStayQuiet`): the trailing comment `// derived backoff/immediate equals the row` → `// derived backoff/skip-if-busy equals the row`. The assertion (`wantEdits(t, *calls)` — no edits) is unchanged and still passes once the fixture moves.
- **Line ~273** (`TestReconcile_DerivedSchedule`, case "pane-only set derives backoff (A-031 idle-every→backoff)"): expected argv `{"cron-op", "--backoff", "--min", "1m", "--max", "30m", "--deliver", "immediate"}` → `{"cron-op", "--backoff", "--min", "3m", "--max", "24m", "--deliver", "skip-if-busy"}`.
- **Lines ~404–406** (tick-start `--diff` reconcile case): the same argv update, and the explanatory comment `// The pane-only set derives backoff/immediate;` → `backoff/skip-if-busy`.
- Table cases "backoff row equal to derived backoff issues nothing" and "R12: equal row (2m0s == 2m) issues nothing" must still hold after the fixture move; the shell-item cases that read `cronListBackoffJSON` as the *drifted* row still differ on `kind` and still emit their edits.

### 3. Kit skill — `src/kit/skills/fab-operator.md`

**§2 Init step 5, line 120** — the ready-line example:

> Renders today as `Operator ready. Clock: rk cron "operator tick" · backoff 1m→30m · immediate`, or `… · immediate · muted until 14:30`.

becomes `… · backoff 3m→24m · skip-if-busy`, and `… · skip-if-busy · muted until 14:30`. (The rendering *rule* — the values are copied verbatim from the JSON — is unchanged; only the illustrative values move.)

**§4 The Clock, the entry-shape YAML block (lines ~175–183)** — currently introduced as "The seeded entry's shape (reference summary …)":

```yaml
schedule: { kind: backoff, min: 60s, max: 30m }
wake_on: { event: agent-state-change, scope: server, debounce: 10s }
…
deliver: immediate
```

becomes:

```yaml
schedule: { kind: backoff, min: 3m, max: 24m }
wake_on: { event: agent-state-change, scope: server, debounce: 60s }
target: { kind: role, role: operator }
payload: "operator tick"
deliver: skip-if-busy
if_absent: respawn
respawn: ["rk", "operator", "-L", "{server}"]   # caller-supplied argv; {server} substituted by rk at fire time
pinned: true
```

**The block's lead-in sentence must be reframed at the same time.** Today it reads as *what `rk operator` seeds*. After this change the `schedule` and `deliver` lines are what **fab derives and keeps** — rk's own seed default still writes `1m`/`30m`/`immediate` until run-kit backlog row `[ntde]` flips it, and fab's first reconcile then converges the live entry. Rewrite the lead-in to describe the entry's **steady-state shape** (seeded by `rk operator`, with `schedule` and `deliver` converged by fab's reconcile on the first `track` mutation or tick), so the block does not assert something false about rk's seeding. `wake_on`, `target`, `payload`, `if_absent`, `respawn`, and `pinned` remain purely rk's, quoted as reference.

**§4 The Clock, Ownership paragraph (line ~186)** — currently:

> … the **schedule reconcile** derives the cadence from the tracked set (backoff `1m`→`30m` while only pane/none items are tracked; `--idle-every`/`--every <min check_every>` once any shell/agent item exists, deliver `skip-if-busy`) …

becomes: backoff `3m`→`24m` while only pane/none items are tracked; `--idle-every`/`--every <min check_every>` once any shell/agent item exists — **deliver is `skip-if-busy` on every branch**.

**Verify-only, no value edit expected**: §6 Key Properties' "Cadence" row (line ~821) and §5's settings paragraph (line ~798) describe the mechanism, not the values; the compact-frame and status-frame examples (lines ~344, ~359–360) already use `idle-every 2m · skip-if-busy`. Confirm during apply that no literal slipped in.

### 4. Kit CLI reference — `src/kit/skills/_cli-fab-operator.md`

**The derive table in the "Clock side effect" paragraph (lines ~164–168)**:

| Tracked set (items not `done`) | Derived schedule | Derived deliver |
|---|---|---|
| empty | *(muted — no edit)* | — |
| only `pane`/`none` items | `--backoff --min 1m --max 30m` | `immediate` |
| any `shell`/`agent` item, and the operator pane has an agent-state epoch | `--idle-every <min(check_every) over the shell+agent items>` | `skip-if-busy` |
| any `shell`/`agent` item, no epoch | `--every <min(check_every)>` | `skip-if-busy` |

The pane/none row becomes `--backoff --min 3m --max 24m` / `skip-if-busy`. With the Derived-deliver column now uniform, state the policy once in the surrounding prose ("every derived branch and every override delivers `skip-if-busy`") — the column MAY stay for row-level legibility but MUST NOT reintroduce a per-branch reading. This satisfies the Constitution's CLI⇒docs obligation: the emitted argv changed, so this partial (the owner of the `fab operator` family) changes with it.

**Line ~153 — a *derived* claim that is now arithmetically wrong**:

> A tick with non-empty `deltas:` or `needs_check:`, any tick whose `last_full_at` is at least 10 minutes old, and any tick without `--quiet` emits the full document … **At the 1m backoff floor that is roughly every 10th tick**; at a cadence of 10m or slower every tick is full.

At a 3-minute floor, a 10-minute full-frame age is reached roughly every **4th** tick. Update the sentence to match; the rule itself (`last_full_at` ≥ 10m) is unchanged.

**Do NOT touch** the `1m` values at lines ~220 and ~227 — those are `--check-every`'s validation **floor** and the override's floor, unrelated to the backoff ladder.

### 5. Memory — `docs/memory/runtime/operator.md` (FKF present truth)

**The clock paragraph (line ~132)** — three edits, written as present truth with no changelog narration:

1. The quoted entry shape: `schedule: { kind: backoff, min: 60s, max: 30m }` → `{ kind: backoff, min: 3m, max: 24m }`; `debounce: 10s` → `debounce: 60s`; `deliver: immediate` → `deliver: skip-if-busy`. Apply the same steady-state reframing as the skill (§3 above): the `schedule`/`deliver` values are fab's derived steady state, not rk's seed default.
2. The derive sentence: "only `pane`/`none` items ⇒ `--backoff --min 1m --max 30m --deliver immediate`; any `shell`/`agent` item ⇒ `--idle-every <min(check_every)> --deliver skip-if-busy`" → `--backoff --min 3m --max 24m --deliver skip-if-busy` for the first arm, with the deliver policy stated once as uniform across every branch and the override.

**The "Binary Derives the Cron Schedule From the Tracked Set" design decision (lines ~601–605)** — extend the existing four-field entry rather than adding a new one (the decision it records is unchanged; its *content* gains the uniform-deliver rule and the accepted cost):

- **Decision**: add — the deliver policy is `skip-if-busy` for every derived branch (backoff, every, idle-every) and for the bounded `clock_override`; the backoff branch is a fallback poll behind `wake_on: agent-state-change` and runs `3m → 24m`.
- **Why**: add — `immediate` injects a tick mid-turn (queued as a mid-turn message on Claude; less predictable on other providers), and a per-branch policy table meant the operator's delivery semantics changed silently when a shell/agent item joined or left the tracked set. A 1-minute floor buys almost no latency over the wake event and mostly buys token spend; a 30-minute ceiling exceeds the operator's own 10-minute full-frame window.
- **Rejected**: add — `when-idle` on the backoff branch (it holds the fire until idle rather than dropping it, so no tick is lost and no rung is paid twice; rejected for policy uniformity — a third policy on one branch rebuilds the two-way table under another name). Keeping `immediate` on the backoff branch (mid-turn injection; the two-way table).
- Record the accepted cost inside the decision: under `skip-if-busy` a tick that lands mid-turn is dropped, so on the backoff branch a drop at a long rung delays the fallback poll by up to that rung (**≤ 24m**); `wake_on` fires independently, so a real fleet event does not wait for the rung.
- *Updated by*: this change's folder name.

**The 5ubz decision's Why (line ~633)** quotes the ladder as context: "The operator's clock is run-kit's cron entry with backoff `1m → 30m`, so 'every 10th tick' is a count over a variable interval — roughly every 10 minutes at the floor but about 5 hours at full backoff …". Update the quoted ladder to `3m → 24m` and recompute the two illustrative figures (≈ 30 minutes at the floor, ≈ 4 hours at full backoff) — the argument is unchanged and in fact strengthened, so only the numbers move. Do **not** rewrite the decision itself.

### 6. Backlog — `fab/backlog.md`, row `[bjrk]`

The row's quoted future seed argv reads `rk cron add 'operator tick' --backoff --min 1m --max 30m … --deliver skip-if-busy … --wake-debounce 60s`. Update `--min 1m --max 30m` → `--min 3m --max 24m` so the blocked row does not re-introduce the retired ladder when it is eventually picked up. No other edit to the row; `[bjrk]` stays blocked and out of scope.

### 7. Migration behaviour on upgrade (no artifact, verify only)

On the first `fab operator track …` mutation or `tick-start --diff` after this ships, a live entry sitting on `backoff 1m→30m / immediate` will not match the derived value, so the reconcile issues **exactly one** `rk cron edit <id> --backoff --min 3m --max 24m --deliver skip-if-busy`. rk logs a `rescheduled` line and resets the backoff anchor (the ladder restarts at the 3m floor). This is expected and acceptable — it is the reconcile working — and needs no migration file: no user data is restructured, only a live rk cron row is converged.

## Affected Memory

- `runtime/operator`: (modify) the clock paragraph's quoted entry shape and derive sentence; the "Binary Derives the Cron Schedule From the Tracked Set" design decision (uniform deliver + new bounds + the accepted ≤24m drop cost + the `when-idle` rejected alternative); the ladder figures quoted in the "Periodic Full Refresh Is Time-Based" decision's Why

## Impact

**Code**

- `src/go/fab/cmd/fab/operator_clock.go` — constants, `deriveOperatorSchedule` and its doc comment (3 return sites)
- `src/go/fab/cmd/fab/operator_track.go` — one line (1012), the `operatorDeliverSkipIfBusy` → `operatorDeliver` rename at the `clock_override` writer
- `src/go/fab/cmd/fab/operator_clock_test.go` — 2 fixtures, 2 expected-argv assertions, 2 comments

**Deployed kit content**

- `src/kit/skills/fab-operator.md` — §2 step 5 ready-line example, §4 entry-shape block + its lead-in, §4 Ownership paragraph
- `src/kit/skills/_cli-fab-operator.md` — the derive table + surrounding prose, the full-frame cadence sentence at line ~153

**Docs**

- `docs/memory/runtime/operator.md` — clock paragraph, two design decisions
- `fab/backlog.md` — row `[bjrk]`'s quoted argv

**Not touched** (verified by grep, with reasons)

- `docs/specs/operator.md` — carries no clock values (it is a version-history spec only). No new version row: this is a cadence-value change, not a change to the skill's shape.
- `docs/wiki/operator-tick-anatomy.html` — a **dated design study** ("Operator tick anatomy — design study, 2026-09-11"), a point-in-time artifact like a plan doc. Four `backoff 1m→30m` occurrences stay as they were written.
- `fab/plans/sahil/26-09-11-operator-generic-tracking.md` — the same class (a dated plan document).
- `fab/changes/**` (active and archived) — historical intakes/plans are never swept.
- Any run-kit / `rk` source — no cross-repo change is in scope here.

**Verification**

- `gofmt -l` on every touched `.go` file (a worker-written Go file left unformatted has failed CI on this repo before).
- Scoped first: `go test ./cmd/fab -run 'Operator.*Clock|Clock|Schedule'` from `src/go/fab`, then the whole `./cmd/fab` package.
- Repo-wide grep for `1m→30m`, `--min 1m --max 30m`, `min: 60s`, `debounce: 10s`, `deliver: immediate`, and `deliver immediate` returning only the four deliberately-excluded historical locations above.

## Open Questions

- Does rk's backoff implementation actually **double** each rung? `3, 6, 12, 24` is a clean doubling ladder if it does, but this was not verified against run-kit's source. It does not block: fab asserts only the `--min`/`--max` bounds, and the growth curve between them is rk's.
- Two sites carrying the old ladder were found by grep outside the scope the discussion enumerated: `docs/wiki/operator-tick-anatomy.html` (4 occurrences) and `fab/plans/sahil/26-09-11-operator-generic-tracking.md`. Both are treated here as dated point-in-time artifacts and deliberately left alone — flag if that is wrong.
- `fab/backlog.md` row `[bjrk]`'s quoted seed argv was not part of the enumerated sweep either; it is updated here so the blocked row does not carry the retired ladder forward.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Deliver is `skip-if-busy` for every derived branch (backoff, every, idle-every) and for `clock_override` — no per-branch policy table remains | Discussed — user chose one uniform policy over the current two-way table, explicitly to kill mid-turn injection | S:95 R:75 A:90 D:95 |
| 2 | Certain | Backoff bounds become `operatorBackoffMin = "3m"` / `operatorBackoffMax = "24m"`; the reconcile emits `--backoff --min 3m --max 24m --deliver skip-if-busy` | Discussed — user gave the exact values and the reasoning (wake_on does the real pickup; a 1m floor buys token spend; 30m exceeds the 10m full-frame window) | S:95 R:80 A:85 D:95 |
| 3 | Confident | `operatorDeliverImmediate` is deleted and `operatorDeliverSkipIfBusy` is renamed to `operatorDeliver` (one call site in `operator_track.go` moves with it) | User said "drop `operatorDeliverImmediate`; a single `operatorDeliver` (or equivalent)" — the rename is the "or equivalent" read; a single-policy constant should not read as one of a set. Trivially reversible | S:65 R:90 A:85 D:65 |
| 4 | Certain | The one-time `rk cron edit` on the first post-upgrade reconcile (rk logs `rescheduled`, backoff anchor resets) is expected and acceptable; no migration file | Discussed — user called this out and accepted it. No user data is restructured, so `context.md` § Migrations does not bite | S:90 R:85 A:85 D:90 |
| 5 | Certain | `wake_on.debounce` is documented as `60s` (was `10s`) in `fab-operator.md` §4 and `docs/memory/runtime/operator.md` | Discussed — the live `rk cron list --json` and run-kit's `operatorTickEntrySpec()` both show 60s; the docs were simply stale | S:85 R:90 A:85 D:90 |
| 6 | Confident | The entry-shape block's lead-in is reframed from "the seeded entry's shape" to the entry's steady-state shape (rk seeds; fab's reconcile converges `schedule`/`deliver`) | Not stated by the user, but required for truth: rk's seed default still writes `1m`/`30m`/`immediate` until run-kit row `[ntde]` lands, so quoting the new values as rk's seed would be false | S:60 R:85 A:90 D:70 |
| 7 | Confident | `_cli-fab-operator.md` line ~153's "At the 1m backoff floor that is roughly every 10th tick" is recomputed to the 3m floor (≈ every 4th tick) | A derived arithmetic claim about the ladder, so it is part of the ladder's sweep class even though the discussion did not enumerate it; the underlying 10-minute rule is unchanged | S:60 R:90 A:85 D:75 |
| 8 | Confident | The ladder figures quoted in the "Periodic Full Refresh Is Time-Based" decision's Why (memory line ~633) are updated to `3m → 24m` (≈30 min at the floor, ≈4 h at full backoff); the decision itself is untouched | FKF present truth: a design decision may keep its historical argument, but a quoted live value that is now wrong is drift. The argument holds a fortiori at the new bounds | S:55 R:85 A:80 D:70 |
| 9 | Confident | `docs/specs/operator.md` is not touched and gains no version-history row | Grep-verified: it carries no clock values, only a version table whose rows mark skill-shape versions (v11 = the generic tracked-items rewrite). A cadence-value change is not a shape version | S:70 R:85 A:85 D:70 |
| 10 | Confident | `docs/wiki/operator-tick-anatomy.html` (4 hits) and `fab/plans/sahil/26-09-11-operator-generic-tracking.md` are NOT swept | Both are dated point-in-time artifacts — the HTML's own header says "design study, 2026-09-11" — in the same class as `fab/changes/**` intakes and plans, which are never swept | S:45 R:90 A:80 D:65 |
| 11 | Confident | `fab/backlog.md` row `[bjrk]`'s quoted seed argv is updated to `--min 3m --max 24m` | Outside the enumerated sweep, but the row is a forward instruction, not a historical record — leaving the retired ladder in it would re-introduce the bug when `[bjrk]` is picked up. One-line, trivially reversible | S:55 R:90 A:85 D:70 |
| 12 | Certain | Non-goals hold: no run-kit change; no move of the entry's seeding into fab (backlog `[bjrk]`, blocked on run-kit `[ntde]`); §2 Init step 4's "run rk operator to seed it" STOP is untouched; no `rk cron add` call is added; the skill never composes cron argv by hand | Discussed — user enumerated each of these explicitly as out of scope | S:95 R:80 A:90 D:95 |
| 13 | Certain | The accepted cost — a `skipped-busy` drop at a long backoff rung delays the fallback poll by up to that rung (≤24m) — is recorded in the memory design decision alongside the rejected `when-idle` alternative | Discussed — the assistant raised it, the user held the decision and asked for it to be recorded | S:90 R:90 A:85 D:90 |
| 14 | Confident | rk's backoff growth curve between `--min` and `--max` is not verified (the "clean doubling 3, 6, 12, 24" reading is an assumption); the change ships regardless | The user flagged it unverified. fab asserts only the bounds — the curve is rk's, so nothing in this change depends on the answer | S:70 R:95 A:70 D:75 |
| 15 | Certain | The `1m` values at `_cli-fab-operator.md` lines ~220 and ~227 are left alone | Grep disambiguation: those are `--check-every`'s validation floor and the override's floor, not the backoff ladder — changing them would be a real behaviour regression | S:75 R:80 A:95 D:85 |

15 assumptions (7 certain, 8 confident, 0 tentative, 0 unresolved).
