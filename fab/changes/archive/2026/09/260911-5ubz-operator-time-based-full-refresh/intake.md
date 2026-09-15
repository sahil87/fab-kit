# Intake: Operator tick-start — time-based full-frame refresh (10 minutes) replaces every-10th-tick

**Change**: 260911-5ubz-operator-time-based-full-refresh
**Created**: 2026-09-11

## Origin

Conversational — synthesized from user feedback on the running `/fab-operator` session on 2026-09-11, then dispatched promptless via `/fab-proceed` (create-new). The user's raw question:

> at what frequency does the agent print the table that lists the number of things it is tracking? Right now I think the frequency is a bit too low.

**Decision reached in the conversation** (user called the rule "nice" and said "go ahead with that"): make the periodic full refresh **time-based** — emit the full document whenever the last full frame is older than **10 minutes** — instead of the current *every 10th tick* count rule. Keep it a **built-in constant, not a flag or config knob**, the same posture as the existing constant `10` (the kbf2 Design Decision "Quiet Tick Replaces `items:` With Counts" rejected a knob).

**Explicitly rejected by the user**: writing an HTML fleet table beside the operator state file. That is run-kit's job — run-kit already consumes the operator state YAML at the pinned cross-repo path, so any fleet UI is rendered by run-kit from that file. fab writes no HTML. This is a hard non-goal, not a follow-up.

Interaction mode: the description handed to this intake was a settled decision — no further questions were asked (promptless dispatch; would-be questions, if any, are recorded as deferred Unresolved rows in `## Assumptions`).

## Why

**Problem.** `fab operator tick-start --diff --quiet` emits the full `items:` document (rendered by the skill as the full status table) only when a tick has non-empty `deltas:`, non-empty `needs_check:`, is a user status request run without `--quiet`, or when the post-increment `tick_count` is a multiple of the built-in constant `tickQuietFullEvery = 10` (`src/go/fab/cmd/fab/operator_tick_start.go:114`, consumed in `emitTickDiffDoc` at line 342). Every other tick is a quiet tick: a `fleet_summary:` count block, rendered as a one-line compact frame.

The operator's clock is run-kit's operator-tick cron entry, whose derived schedule is **backoff `1m → 30m`** while only pane items are tracked (`_cli-fab-operator.md` § Clock side effect; `fab-operator.md` §4 The Clock). So "every 10th tick" is a *count* over a *variable* interval: at the 1-minute floor a full table appears every ~10 minutes, but at full backoff it appears roughly every **5 hours** if nothing changes. A count is the wrong unit under a variable cadence — the user's "frequency is too low" is exactly this: the refresh interval silently stretches with backoff.

**Consequence of not fixing.** On a quiet fleet the operator shows only one-line compact frames for hours; the user loses the periodic complete view the kbf2 decision was meant to preserve ("the every-10th-tick full document keeps a periodic complete view"). The user then asks for status manually, which is the interaction the periodic refresh exists to avoid.

**Why this approach.** A wall-clock threshold makes the refresh cadence independent of the cron schedule: at the 1-minute cadence ticks stay quiet for ~10 ticks (matching today's behavior), and at any backed-off cadence ≥ 10 m every tick is a full table (which is the right answer — a tick that rare should always show everything). The binary already owns the tick bookkeeping (`tick_count`, `last_tick_at`) and the full/quiet decision, so persisting one more RFC3339 scalar, `last_full_at`, in the same atomic mutation is the smallest correct change. Alternatives rejected: a flag or config knob (kbf2 precedent; the §5 hardcoded-30m idle auto-default is the same posture); a skill-side timer (the skill deliberately keeps no counter — "no skill-side counter is kept", `fab-operator.md` line 309); deriving the interval from the cron schedule (couples fab to run-kit's schedule shape for no gain — a time threshold is cadence-agnostic by construction); an HTML fleet table (run-kit's job — see Origin).

## What Changes

Go behavior change in `fab operator tick-start --diff [--quiet]` plus the CLI reference, the operator skill, and memory. Constitution Additional Constraints: a CLI change MUST update `src/kit/skills/_cli-fab-operator.md` and tests. Canonical skill source is `src/kit/skills/`; `.agents/skills/` and `.claude/skills/` are deployed copies and are never edited (Constitution V).

### 1. Go — `src/go/fab/cmd/fab/operator_tick_start.go`

**1a. Constant.** Replace the count constant with a duration constant. Keep the doc comment's posture sentence (built-in, not a flag or config knob, §5 hardcoded-30m precedent) and update the rule:

```go
// tickQuietFullAfter is the built-in periodic full-refresh interval: under
// --quiet, a tick whose last full document (last_full_at) is at least this
// old emits the full document (items:) even with no deltas, so a complete
// frame still appears periodically regardless of the cron cadence. At the
// 1m backoff floor that is roughly every 10th tick; at any cadence ≥ 10m
// every tick is full. Deliberately a constant — not a flag or config knob
// (matches the §5 hardcoded-30m idle auto-default precedent).
const tickQuietFullAfter = 10 * time.Minute
```

`tickQuietFullEvery` is deleted (no alias, no deprecation — it is unexported and has one consumer).

**1b. New persisted scalar `last_full_at`.** A top-level field in the server-keyed operator state file (`<XDG_STATE_HOME>/fab/operator/<server-slug>.yaml`), RFC3339 UTC — same format and same writer as `last_tick_at` (`now.UTC().Format(time.RFC3339)`, the `nowStr` already captured once per tick). Written by `tick-start --diff` **inside the same `mutateOperatorStateClock` callback** as `tick_count`/`last_tick_at`/the baselines, on every tick whose emitted document is the full one (`items:` present) — whether it is full because `--quiet` is absent, because of deltas, because of `needs_check`, or because of the age threshold. A quiet tick leaves `last_full_at` untouched. The flagless `tick-start` (no `--diff`) never touches it — it emits no document (the field survives that read-modify-write as an unknown top-level key, like every other scalar).

Example state file after this change (reference — the binary owns the schema):

```yaml
tick_count: 48
last_tick_at: "2026-09-11T16:36:00Z"
last_full_at: "2026-09-11T16:30:00Z"
tracked: [...]
branch_map: {}
```

**1c. Full/quiet decision moves into the mutation.** Today `emitTickDiffDoc` computes `full` after the write. It must move inside the callback because it reads the *prior* `last_full_at` and, when full, writes the new one in the same atomic save. The predicate becomes:

```go
full := !quiet || len(out.Deltas) > 0 || len(out.NeedsCheck) > 0 || tickFullDue(data["last_full_at"], now)
if full {
    data["last_full_at"] = nowStr
}
```

where `tickFullDue(raw interface{}, now time.Time) bool` returns **true** when `raw` is missing, not a string, or fails `time.Parse(time.RFC3339, …)` (a missing or unparseable `last_full_at` counts as "older than the threshold", so the first tick after upgrade and a fresh state file render a full frame), and otherwise returns `now.Sub(t) >= tickQuietFullAfter`. A `last_full_at` in the future (clock skew or a corrupt stamp) also returns true — a defensive choice so a bad stamp can never suppress the refresh.

The decision must be computed on **both** exits of the callback: the empty-tracked short-circuit (`len(items) == 0` → `return nil`, where `deltas`/`needs_check` are provably empty so only the age term can make it full) and the normal path after `summarizeItems`. Apply may restructure (e.g., compute `full` once just before `return nil` via a small helper, or wrap the callback) — the contract is: prior value read and new value written inside the one mutation; `emitTickDiffDoc` receives the already-decided `full bool` (it still receives `tickCount` for the `tick: N` header line, which is unchanged). Its doc comment ("post-increment tickCount not a multiple of tickQuietFullEvery") is rewritten to the time rule.

**1d. Flag help text** (`operator_tick_start.go:39`): `"with --diff: on a no-delta tick that is not every 10th, replace the items: block with a fleet_summary: count block"` → e.g. `"with --diff: on a no-delta tick within 10m of the last full document, replace the items: block with a fleet_summary: count block"`.

**1e. Output byte-identity.** Flagless `--diff` output stays byte-identical to today (always the full document; it additionally writes `last_full_at`, which is a state-file side effect, not stdout). Flagless `tick-start` (no `--diff`) is byte-identical in stdout and state (`TestOperatorTickStart_FlaglessByteIdentical` keeps passing unchanged apart from any seed-helper rename). The four-block order and the `fleet_summary:` key order are untouched.

**1f. No migration file.** The binary owns the operator state schema; `last_full_at` is additive with a safe zero-value (absent ⇒ full document), and the file is not user-authored. Precedent: kbf2 added the `tick_count`-based quiet semantics with no migration; `context.md` § Migrations targets user-data restructuring (config, `.status.yaml`, archive layout). Legacy conversion (`convertLegacyOperatorState`, the `monitored`/`watches`/`autopilot`/`notes` sections) is untouched. Note for hydrate: `_cli-fab-operator.md` line 154 enumerates the owned scalars (`tick_count`/`last_tick_at`) — `last_full_at` joins that list (see § 3).

### 2. Tests — `src/go/fab/cmd/fab/operator_tick_diff_test.go`

Replace the count-based cases with time-based ones. Existing scaffolding: `seedDiffStateAt(t, items, tickCount)` (comment "the every-10th-tick cases seed 9/19/10" — update or replace), `withOperatorState(t, yaml)`, `readStateFile(t, path)`, `stubSnapshot`, `stubQuietClock`, `runTickDiffArgs(t, args...)`, `assertDocKeys(t, out, wantSummary)`, `rfc3339Ago(d)`. A seed variant that accepts extra top-level keys (e.g. `last_full_at`) is needed — either a `seedDiffStateWith(t, items, extra map[string]interface{})` or seeding via `withOperatorState` raw YAML as `QuietEmptyTracked` does.

| Test (replace / add) | Seed | Expect |
|---|---|---|
| `TestOperatorTickDiff_QuietEveryTenthTickEmitsFullItems` → **delete**; replace with `TestOperatorTickDiff_QuietFullAfterTenMinutes` (table-driven) | one pane item, no deltas (snapshot stage == baseline, `agent_state: waiting`) | subcases below |
| — subcase "absent last_full_at is full" | no `last_full_at` key | `assertDocKeys(out, false)`; state `last_full_at` == a fresh RFC3339 within a few seconds of now |
| — subcase "unparseable last_full_at is full" | `last_full_at: "not-a-time"` | full; stamp rewritten to a valid RFC3339 |
| — subcase "recent last_full_at is quiet" | `last_full_at: rfc3339Ago(9*time.Minute + 59*time.Second)` (see Assumptions #8 on the margin) | `assertDocKeys(out, true)`; state `last_full_at` **unchanged** (byte-equal to the seed) |
| — subcase "10m-old last_full_at is full" | `rfc3339Ago(10*time.Minute)` | full; `last_full_at` == now |
| — subcase "future last_full_at is full" | `rfc3339Ago(-time.Hour)` | full; stamp rewritten to now |
| `TestOperatorTickDiff_QuietWithDeltaEmitsFullItems` — **extend** | existing (stage_advance delta) + `last_full_at: rfc3339Ago(time.Minute)` | full regardless of age; `last_full_at` == now |
| `TestOperatorTickDiff_NeedsCheckForcesFullDocument` — **extend** | existing + recent `last_full_at` | full; `last_full_at` == now |
| `TestOperatorTickDiff_QuietEmptyTracked` — **rewrite** the two subtests | "recent last_full_at emits all-zero summary" seeds `last_full_at: rfc3339Ago(time.Minute)`, `tracked: []`; "stale last_full_at emits items: []" seeds `rfc3339Ago(10*time.Minute)` | summary / `items: []`; snapshot never invoked in either |
| `TestOperatorTickDiff_QuietNoDeltasEmitsSummary`, `QuietSummaryMixedItems` — **adjust seed** | today they rely on tick 5→6 being non-10th; they must seed a recent `last_full_at` (otherwise the absent-key rule makes them full) | unchanged expectations |
| `TestOperatorTickStart_FlaglessByteIdentical` — **assert** | existing | state file gains no `last_full_at` on the flagless path (add to the "gained key" loop) |
| **Add** `TestOperatorTickDiff_FlaglessDiffWritesLastFullAt` | one pane item, `--diff` without `--quiet`, recent `last_full_at` | full document (byte-identical shape to today); `last_full_at` == now |

Every other existing tick test seeds via `seedDiffState` (tick 5, no `last_full_at`) and asserts on deltas/items, not on the quiet/full shape, so the absent-key ⇒ full rule leaves them passing; apply verifies by running the package.

Run scoped first: `go test ./src/go/fab/cmd/fab/ -run 'TestOperatorTick' -count=1`, then `go test ./src/go/fab/cmd/fab/...`; `gofmt -l src/go/fab/cmd/fab/` must print nothing (CI failed once before on unformatted worker-written Go).

### 3. CLI reference — `src/kit/skills/_cli-fab-operator.md` § fab operator tick-start

Constitution: the CLI change MUST land here. Sites (line numbers as of `aa9d08ee`):

- **Line 44** (intro): "Increments `tick_count`, writes `last_tick_at` (RFC3339 UTC) to the **server-keyed** state file" — add: with `--diff`, also writes `last_full_at` (RFC3339 UTC) on every tick that emits the full document.
- **Line 137** (`--quiet` bullet): "On a **quiet tick** — `deltas:` AND `needs_check:` both empty AND the post-increment `tick_count` not a multiple of the built-in constant 10 (not a flag or config knob)" → "… AND the last full document (`last_full_at`) less than 10 minutes old (a built-in constant, not a flag or config knob; a missing or unparseable `last_full_at` counts as stale, so a fresh state file and the first tick after upgrade emit the full document)".
- **Line 148**: "A tick with non-empty `deltas:` or `needs_check:`, every 10th tick, and any tick without `--quiet` emits the full document." → "…, any tick whose `last_full_at` is at least 10 minutes old, and any tick without `--quiet` emits the full document — and every full document writes `last_full_at` in the same atomic mutation. At the 1m backoff floor that is roughly every 10th tick; at a cadence of 10m or slower every tick is full."
- **Line 154** (Shared state-verb mechanics): "plus the `tick_count`/`last_tick_at` scalars" → add `last_full_at`; the timestamp enumeration "All timestamps (`added_at`, `updated_at`, `checked_at`, `last_tick_at`, the override's `until`)" gains `last_full_at`.
- **Line 152** (State path / cross-repo contract paragraph): add one sentence — adding a top-level field is additive under the tolerant-read contract and needs no coordinated run-kit change (the contract is the file path and slug rule, not the field set).

### 4. Operator skill — `src/kit/skills/fab-operator.md`

Deployed content — MUST NOT cite fab-kit-only paths (Constitution V; the Go guard in `src/go/fab-kit/cmd/fab/` fails on it).

- **Line 309** (§4 Tick Behavior step 1): "the binary's built-in every-10th-tick full document is the periodic full refresh, so no skill-side counter is kept" → "the binary's built-in time-based full document — a full `items:` document whenever the last one is 10 minutes or older — is the periodic full refresh, so no skill-side counter or timer is kept".
- **Line 338** (Status Frame Format, Full frame bullet): "(`items:` present — a delta tick, a non-empty `needs_check:`, every 10th tick, or a user status request run without `--quiet`)" → "(… a tick 10 minutes or more after the last full document, or a user status request run without `--quiet`)".
- **Lines 230–231** (state-file reference block): add `last_full_at: "2026-09-11T16:30:00Z"` under `last_tick_at`, with a trailing comment `# written on every full tick document; the 10m periodic-refresh clock`.

### 5. Memory — `docs/memory/runtime/operator.md` and `docs/memory/distribution/kit-architecture.md`

Hydrate-stage work, listed here so the sweep class is known up front (code-quality.md § Sibling Sweeps):

- `operator.md:257` (Design Constraints, "Binary-internal tick snapshot"): "a no-delta tick whose tick count is not a multiple of 10 replaces `items:` … and every 10th tick emit the full document" → the time rule.
- `operator.md:278` (`fab operator tick-start` description): "on a no-delta, no-needs_check tick whose post-increment `tick_count` is not a multiple of the built-in constant 10 (no flag/config knob); a delta or needs_check tick and every 10th tick emit the full document … (or `items: []` on a 10th tick)" → the time rule; the fields-written list (`tick_count`, `last_tick_at`) gains `last_full_at`; the "same atomic mutation" sentence names it.
- `operator.md:288` (Usage in tick lifecycle): "the binary's every-10th-tick full document is the periodic full refresh, no skill-side counter" → time rule.
- `operator.md:524–527` (DD "Quiet Tick Replaces `items:` With Counts; Compact One-Line Frame on No-Change Ticks"): keep the body as history; append `; *Updated by*: 260911-5ubz-operator-time-based-full-refresh (the periodic full refresh is time-based — `last_full_at` ≥ 10m — not every 10th tick)` to the *Introduced by* line, mirroring the existing `*Updated by*: 260911-4a8m-…` style.
- **New DD** in `operator.md` § Design Decisions (four-field shape), e.g. "Periodic Full Refresh Is Time-Based (`last_full_at` ≥ 10m), Not a Tick Count": **Decision** — the full-document threshold is a wall-clock age persisted as `last_full_at`, a built-in 10-minute constant; **Why** — the cron cadence backs off `1m → 30m`, so a tick count is a variable-length interval (≈5 h at full backoff); a duration is cadence-agnostic and matches today's behavior at the 1m floor; **Rejected** — a flag or config knob (kbf2 posture), a skill-side timer (no skill-side counter), deriving the interval from the cron schedule, and **an HTML fleet table written by fab beside the state file — run-kit owns any fleet UI and already reads the state YAML at the pinned path; fab writes no HTML**; *Introduced by* this change.
- `kit-architecture.md:90` (`fab operator tick-start` bullet — **verified**, it names both the fields written and the 10th tick): "increments `tick_count`, writes `last_tick_at` (RFC3339 UTC)" gains `last_full_at` (with `--diff`, on full-document ticks); "on a no-delta, no-needs_check tick whose tick count is not a multiple of the built-in 10 (full document on deltas, needs_check, and every 10th tick)" → time rule.
- `operator.md:488` (DD Tolerant-Read / Typed-Write) — verify only; update if it enumerates the scalars.
- **Verify-only / leave alone**: `docs/memory/runtime/log.md:36` and `docs/memory/distribution/log.md:20` (dated kbf2 hydrate log lines — history, not present truth), `docs/specs/findings/*`, `src/kit/migrations/2.24.9-to-2.25.0.md:31–32,69` (a historical migration walkthrough naming the scalars preserved by legacy conversion — still true), `docs/specs/operator.md` (no 10th-tick prose). Repo-wide grep for `10th`/`multiple of`/`tickQuietFullEvery` found no other operator sites — `fab-operator.md:201,778` and `_shared/context-loading.md:214` match "every 10m"/"10th poll" and are unrelated.

### 6. Cross-repo contract — explicitly no run-kit change

The run-kit contract is the state file **path and slug rule** (`<XDG_STATE_HOME>/fab/operator/<server-slug>.yaml`, pinned in run-kit's operator-cron spec). Adding a top-level field is additive under the tolerant-read posture (unknown top-level keys survive every read-modify-write; run-kit reads what it needs and ignores the rest) — **no coordinated run-kit change** is needed, and none is proposed. Any fleet UI that renders this file (including an HTML table) is run-kit's to build from the YAML; this change adds `last_full_at` to what such a renderer could read but proposes nothing on the run-kit side.

### 7. Not in scope

- Any HTML/markdown fleet artifact written by fab (user decision — run-kit owns it; not a follow-up).
- A flag or config knob for the interval (kbf2 posture retained).
- Changes to the compact/full frame *format* (`fab-operator.md` Status Frame Format table cells are untouched except the Full-frame trigger list).
- The kp3d §1 principles text (`260911-kp3d-operator-read-plans-never-execute`, merged 2026-09-11, PR #665) — separate; do not touch.
- The backlog daemon idea `[2ne8]` and the tick-completion bug `[nr3a]` — separate.

## Affected Memory

- `runtime/operator`: (modify) Design Constraints line 257, tick-start description line 278, tick-lifecycle usage line 288, kbf2 DD *Updated by* line 527, plus one new Design Decision recording the count→time rule and the run-kit-owns-HTML non-goal
- `distribution/kit-architecture`: (modify) `fab operator tick-start` bullet line 90 — fields written (`last_full_at`) and the quiet-tick rule

## Impact

- **Go**: `src/go/fab/cmd/fab/operator_tick_start.go` (constant, predicate helper, mutation callback, `emitTickDiffDoc` signature/comment, flag help) — one file, ~40 lines net. Tests: `src/go/fab/cmd/fab/operator_tick_diff_test.go` (delete one test, add one table-driven test and one flagless-diff test, extend three, adjust seeds on two–four).
- **Kit skills** (deployed, Constitution V citation rule applies): `src/kit/skills/_cli-fab-operator.md` (5 sites), `src/kit/skills/fab-operator.md` (3 sites).
- **Memory**: `docs/memory/runtime/operator.md` (4 sites + 1 new DD), `docs/memory/distribution/kit-architecture.md` (1 site).
- **State file**: additive top-level scalar `last_full_at`; no migration; legacy conversion untouched.
- **Cross-repo**: none (additive field under the tolerant-read contract).
- **Expected lane**: full (Go + tests + CLI ref + skill + memory ≈ 8–10 tasks).

## Open Questions

None — every design point was settled in the conversation or is determined by the codebase (see `## Assumptions`).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Periodic full refresh is time-based: full document when `last_full_at` is ≥ 10 minutes old; constant `tickQuietFullAfter = 10 * time.Minute`; count constant `tickQuietFullEvery` deleted | Discussed — user agreed ("go ahead with that"); 10 minutes named by the user | S:95 R:85 A:90 D:95 |
| 2 | Certain | Built-in constant, no flag and no config knob | Discussed — same posture as the existing constant; kbf2 DD rejected a knob; §5 hardcoded-30m precedent | S:95 R:90 A:95 D:95 |
| 3 | Certain | fab writes no HTML fleet table; run-kit renders any fleet UI from the state YAML; not a follow-up | Discussed — explicit user decision | S:95 R:90 A:95 D:95 |
| 4 | Certain | `last_full_at` is a top-level RFC3339 UTC scalar written inside the same `mutateOperatorStateClock` callback as `tick_count`/`last_tick_at`; the full/quiet decision is computed inside the mutation (reads the prior value) | Description mandates the same-atomic-mutation contract; the existing callback already owns both bookkeeping scalars | S:90 R:85 A:95 D:90 |
| 5 | Certain | Missing, non-string, or unparseable `last_full_at` ⇒ full document; a full tick always rewrites the stamp | Description states it; safe zero-value is what makes the field additive | S:90 R:90 A:95 D:90 |
| 6 | Confident | A future-dated `last_full_at` (clock skew / corrupt stamp) also counts as due ⇒ full | Defensive default so a bad stamp can never suppress the refresh; trivially reversible; not discussed | S:55 R:90 A:80 D:70 |
| 7 | Certain | No migration file; legacy conversion untouched | Binary-owned schema, additive field, not user-authored; kbf2 precedent; `context.md` § Migrations scopes migrations to user-data restructuring | S:85 R:90 A:95 D:90 |
| 8 | Confident | Boundary tests use real `time.Now()` with seeds via `rfc3339Ago`; the quiet-side case seeds 9m59s only if apply adds a `tickNow` seam, otherwise it seeds with a ≥ 30 s margin (e.g. 9m) to avoid a same-second flake; the full-side 10m case is monotone-safe as-is | The tick path has no clock seam today; a 1 s window between seed and run is a real flake risk; `rfc3339Ago` truncates to seconds | S:65 R:95 A:80 D:70 |
| 9 | Certain | Flagless `--diff` (no `--quiet`) also writes `last_full_at` (it emits the full document); flagless `tick-start` (no `--diff`) never touches it | Description: every full document writes the stamp; the no-`--diff` path emits no document | S:85 R:85 A:90 D:90 |
| 10 | Certain | `emitTickDiffDoc` takes the decided `full bool` (keeps `tickCount` for the header) instead of `quiet, tickCount`; exact helper naming/shape is apply's | Mechanics, not contract; description says apply may refine mechanics | S:75 R:95 A:90 D:85 |
| 11 | Certain | Existing quiet-shape tests (`QuietNoDeltasEmitsSummary`, `QuietSummaryMixedItems`, `QuietEmptyTracked`) must seed a recent `last_full_at`, since the absent-key rule now makes an unseeded tick full | Follows from #5 applied to the current `seedDiffState` (no `last_full_at`) | S:80 R:95 A:90 D:90 |
| 12 | Confident | `change_type` = `fix` | The driver is a user-reported misbehavior (refresh interval stretches with backoff — a count is the wrong unit); no new user-facing capability (no flag, no config); kbf2/dbwg were `feat` because they added flags and frame modes | S:70 R:95 A:75 D:60 |
| 13 | Certain | Cross-repo: no run-kit change; the contract is path + slug, and the field is additive under tolerant-read | `_cli-fab-operator.md` line 152 and `operator.md` line 80 pin the contract as the path/slug rule only | S:85 R:90 A:90 D:90 |
| 14 | Certain | Sweep class is exactly the sites listed in § 3–5; `fab-operator.md:201,778` and `_shared/context-loading.md:214` are unrelated matches; log.md lines, findings, and the 2.25.0 migration doc are history and stay | Repo-wide grep performed at intake (`10th`, `multiple of`, `tickQuietFullEvery`, `last_tick_at`) | S:85 R:90 A:90 D:85 |
| 15 | Certain | Full lane (≈ 8–10 tasks) | Go + tests + CLI ref + skill + two memory files | S:80 R:95 A:85 D:85 |

15 assumptions (12 certain, 3 confident, 0 tentative, 0 unresolved).
