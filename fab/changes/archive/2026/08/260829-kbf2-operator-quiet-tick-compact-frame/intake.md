# Intake: Operator Quiet Tick + Compact No-Change Status Frame

**Change**: 260829-kbf2-operator-quiet-tick-compact-frame
**Created**: 2026-08-29

## Origin

> Shrink the operator's per-tick context footprint — quiet tick output + compact no-change status frame.
>
> The operator is a long-lived `/loop` heartbeat (3m normal / 90s tightened). Every tick appends to the session context: the full `fab operator tick-start --diff` stdout document (a `fleet:` row per monitored entry every tick, even when nothing changed), the full multi-table status frame the skill renders from it, and the idle message. Context therefore grows linearly with wall-clock time even on a perfectly quiet fleet, which shortens the time-to-compaction and buries real events in identical repeated frames.

Interaction mode: promptless dispatch from the user's operator (no questions asked). A sibling change just shipped — PR #625, `260829-4q3l-operator-loop-prompt-hygiene` — fixing the loop PROMPT (bare `operator tick`, never a slash command) and adding a Post-Compaction Reload procedure; that change explicitly deferred THIS one (the steady-state tick payload) as a separate FULL change. Decisions (a)–(e) under § What Changes → Decisions were delegated by the user's operator and are recorded as Confident. The user rejected slimming the `_cli-fab.md` load (out of scope); the Go-side daemon heartbeat is backlog `[2ne8]` (out of scope).

## Why

**Problem.** The operator tick has three per-tick outputs, all of which scale with fleet size and repeat verbatim when nothing changed:

1. `fab operator tick-start --diff` stdout — `deltas: []`, `candidates: [...]`, and a `fleet:` block with one 9-field YAML row per monitored entry. On a quiet 8-agent fleet that is ~8 × 9 lines of identical YAML every 3 minutes.
2. The status frame the skill renders from `fleet:` — header + a `📂` anchor and a 5-column table per repo + the Watches table.
3. The idle message.

Because the harness appends every tool result and every assistant message to the session, context grows linearly with wall-clock time on a fleet where nothing is happening. That shortens time-to-compaction (the operator loses its working state on compaction — PR #625 added a Post-Compaction Reload precisely because this hurts) and buries the rare real event (a completion, a review_fail, a waiting agent) inside a stack of byte-identical frames.

**Consequence of not fixing.** Every long operator session compacts sooner than it needs to, and the compact→reload cycle is the operator's biggest reliability risk. The daemon heartbeat (`[2ne8]`) is the structural fix but is a large Go change; this change gets most of the steady-state win now with a small, back-compatible flag.

**Why this approach.** Keep the binary the owner of "did anything change" (same doctrine as `fab score`/`tick-start --diff`: agents never compute what the binary can own). A `--quiet` flag lets the binary replace the `fleet:` block with a five-count `fleet_summary:` when `deltas:` is empty, while a hardcoded every-10th-tick full emission guarantees the operator (and the user scrolling back) still sees a complete frame periodically without any new config surface. The skill then renders a one-line compact frame from `fleet_summary:`. The flagless `--diff` stays byte-identical, so an older binary or a user status request still gets today's full document.

## What Changes

### 1. Go — `fab operator tick-start --diff --quiet`

File: `src/go/fab/cmd/fab/operator_tick_start.go`.

- New bool flag `--quiet` on `operatorTickStartCmd()`: help text along the lines of `"with --diff: on a no-delta tick that is not every 10th, replace the fleet: block with a fleet_summary: count block"`.
- **Validity**: `--quiet` without `--diff` is an error (exit non-zero, one-line message, e.g. `--quiet requires --diff`), checked in `runOperatorTickStart` before any state read/write — the flagless path must remain byte-identical and must not tick when the flag combination is invalid.
- **Constant**: `const tickQuietFullEvery = 10` (package-level, doc-commented as the built-in periodic full-refresh interval; deliberately not a flag or config knob).
- **Semantics** (evaluated after `diffMonitored` and the state write, i.e. against the final `out` and the post-increment `tickCount`):
  - **Quiet tick** = `--quiet` AND `len(out.Deltas) == 0` AND `tickCount % tickQuietFullEvery != 0` → emit `deltas:`, `candidates:`, then `fleet_summary:` — NO `fleet:` key.
  - **Otherwise** (deltas non-empty, OR tick is a multiple of 10, OR no `--quiet`) → today's document exactly: `deltas:`, `candidates:`, `fleet:` — NO `fleet_summary:` key.
  - `candidates:` is ALWAYS emitted (the §5 sweep consumes it every tick).
  - Never both `fleet` and `fleet_summary`. Pinned key order: `deltas`, `candidates`, then one of the two.
- **`fleet_summary:` shape** — a mapping of five ints, in this pinned order:

  ```yaml
  fleet_summary:
      tracked: 8      # len(monitored) — one per monitored entry
      waiting: 1      # rows whose snapshot agent_state == waiting
      idle: 3         # rows whose snapshot agent_state == idle
      active: 3       # rows whose snapshot agent_state == active
      unknown: 1      # rows whose agent_state is null (— / empty)
  ```

  Counts are taken over the same rows `fleet:` would have carried (`tickFleetRow.AgentState`). `tracked == waiting + idle + active + unknown` always holds on a quiet tick: dead and mismatched panes cannot appear here because they emit level-triggered deltas, which force the full document.
- **Empty-monitored short-circuit** stays (no pane-snapshot subprocess). Under `--quiet` with an empty monitored set, the tick is a no-delta tick, so it emits `fleet_summary: {tracked: 0, waiting: 0, idle: 0, active: 0, unknown: 0}` — unless `tickCount % 10 == 0`, in which case it emits `fleet: []` as today.
- **Implementation shape** (guidance, not binding): keep `tickDiffOutput{Deltas, Candidates, Fleet}` as-is for the full path; add a `tickFleetSummary` struct (`Tracked, Waiting, Idle, Active, Unknown int` with yaml tags) and a `summarizeFleet([]tickFleetRow) tickFleetSummary` helper; marshal via a second output struct (e.g. `tickDiffQuietOutput{Deltas, Candidates, FleetSummary}`) or a `yaml.Node` so the key order stays pinned and the omitted key is genuinely absent, not `fleet: []`. Keep `runOperatorTickStartDiff` under the 50-line god-function bar by extracting the emit step.

### 2. Tests — `src/go/fab/cmd/fab/operator_tick_diff_test.go`

Extend in the existing style (`seedDiffState` / `diffEntry` / `snapRow` / `stubSnapshot` / `parseTickDiff`). `runTickDiff` hardcodes `--diff`; add an args-taking variant (e.g. `runTickDiffArgs(t, "--diff", "--quiet")`) rather than duplicating the harness. `tickDiffDoc` needs a `FleetSummary *tickFleetSummary \`yaml:"fleet_summary"\`` field, and the assertions must check key PRESENCE (`strings.Contains(out, "fleet:")` / `"fleet_summary:"`) not just parsed emptiness — the contract is "the key is absent", not "the key is empty". Required cases:

1. `TestOperatorTickDiff_QuietNoDeltasEmitsSummary` — quiet + no deltas (tick 6, not a multiple of 10) → `fleet_summary:` present with correct counts, `fleet:` absent, `candidates:` present, `deltas: []`.
2. `TestOperatorTickDiff_QuietWithDeltaEmitsFullFleet` — quiet + a delta (any kind; use a `stage_advance`) → `fleet:` present, `fleet_summary:` absent.
3. `TestOperatorTickDiff_QuietEveryTenthTickEmitsFullFleet` — seed `tick_count: 9` (→ tick 10) and `tick_count: 19` (→ 20) with no deltas → `fleet:` present, `fleet_summary:` absent; seed `tick_count: 10` (→ 11) → summary.
4. `TestOperatorTickStart_QuietRequiresDiff` — `--quiet` alone → error, no tick increment (state `tick_count` unchanged).
5. `TestOperatorTickDiff_QuietSummaryCounts` — a fleet with 2 waiting, 1 idle, 2 active, 1 unknown (`agentState: ""`) → `tracked: 6, waiting: 2, idle: 1, active: 2, unknown: 1`.
6. `TestOperatorTickDiff_QuietEmptyMonitored` — empty monitored set + quiet on tick 6 → `fleet_summary` all zeros and snapshot NOT invoked; on tick 10 → `fleet: []`.

Existing tests (`TestOperatorTickDiff_EmptyMonitoredSkipsSnapshot`, `TestOperatorTickStart_FlaglessByteIdentical`, and the `fleet:` assertions in `TestOperatorTickDiff_Fleet`) must keep passing unchanged — flagless `--diff` is byte-identical.

### 3. `src/kit/skills/_cli-fab.md` § fab operator tick-start

- Usage line → `fab operator tick-start [--diff [--quiet]]`.
- Add a `--quiet` bullet: valid only with `--diff` (error otherwise); on a no-delta tick whose `tick_count` is not a multiple of 10, `fleet:` is replaced by `fleet_summary:` (show the five-key block); deltas non-empty or every 10th tick → identical to plain `--diff`; `candidates:` always emitted; never both keys; key order `deltas`, `candidates`, then `fleet` | `fleet_summary`; the empty-monitored case under `--quiet` emits the all-zero summary (or `fleet: []` on a 10th tick). State that the 10 is a built-in constant, not configurable.

### 4. Skill — `src/kit/skills/fab-operator.md` §4 (canonical source only; never `.claude/skills/`)

**Tick Behavior step 1** — the tick runs `fab operator tick-start --diff --quiet` by default. Run WITHOUT `--quiet` only when the user asks for status ("status", "any updates?", "show the fleet") — the built-in every-10th-tick full document is the periodic full refresh, so no skill-side counter is kept. Describe the fourth block: `fleet_summary:` (five counts) appears IN PLACE OF `fleet:` on a quiet tick; the skill branches its frame on which key is present. Extend the existing **Version-skew fallback** sentence: if `--quiet` errors as an unknown flag (older binary), drop `--quiet` for the session (keep `--diff`) and report once — this is a separate, softer rung than the existing `--diff`→flagless fallback.

**Status Frame Format** — add a **Compact frame** rule (this subsection is the single owner of the rule; §9 points here):

- When the tick document carries `fleet_summary:` (no deltas, not a 10th tick), emit exactly ONE line:

  ```
  🛰️ **Operator** · {HH:MM} · tick #{N} · **{tracked} tracked** · no change
  ```

  Append ` · {waiting} waiting` only when `waiting > 0` (e.g. `… · **8 tracked** · no change · 1 waiting`). No anchors, no tables. `{tracked}` keeps the full header's definition (changes + watches): changes = `fleet_summary.tracked`, watches = the count from step 3's `fab operator state` read.
- The **full frame** (header + repo tables + Watches table) renders when `fleet:` is present — a delta tick, a 10th tick, or a user status request run without `--quiet`.
- The **Watches table** renders on a compact tick ONLY if the watch pass (step 3) produced news this tick — new items, a `last_error`, or an auto-disable; otherwise it is omitted.
- The *italic* action-footnote line still renders whenever an action happened (nudge / answer / removal / autopilot), on either frame shape.
- Add the compact line to the Element table (one row: `Compact frame | 🛰️ **Operator** · {HH:MM} · tick #{N} · **{N} tracked** · no change[ · {W} waiting] | Rendered from fleet_summary:; replaces header+tables on a quiet tick`).

**Idle Message** — shape unchanged. State explicitly that the idle message is the ONLY other per-tick output: nothing else — no restating the tick document, no echoing `candidates:`, no per-candidate "no question detected" lines.

**§9 Key Properties** — amend the `Uses /loop?` row minimally with a pointer clause (e.g. `… quiet ticks render the one-line compact frame (§4 Status Frame Format)`), or add a `Per-tick output?` pointer row. Owner-or-pointer: the rule is stated once, in § Status Frame Format.

Do NOT touch `### Loop Prompt` / `### Post-Compaction Reload` — those sections belong to 4q3l (PR #625) and are absent on this branch (origin/main does not yet contain #625); do not recreate them. If this change lands after #625, the apply worker rebases and leaves those sections alone.

### 5. Sibling sweep (`fab/project/code-quality.md` § Sibling Sweeps)

- `docs/memory/runtime/operator.md`:
  - § Requirements "Monitoring tick" paragraph (~line 161): tick opens with `fab operator tick-start --diff --quiet`; `fleet:` or `fleet_summary:` carries the frame data.
  - "Binary-internal tick snapshot" bullet (~268) and the `tick-start` bullet (~286): add `--quiet` semantics (summary in place of fleet, every-10th full, candidates always).
  - "Usage in tick lifecycle" (~290): default `--diff --quiet`; user status request drops `--quiet`; add the `--quiet`-unknown fallback rung; the idle message is the only other per-tick output.
  - "Frame rendering (markdown-native)" paragraph (~159): mention the one-line compact frame on quiet ticks.
  - Add a Design Decision in the four-field shape (**Decision / Why / Rejected / *Introduced by***): "Quiet Tick Replaces `fleet:` With Counts; Compact One-Line Frame on No-Change Ticks (kbf2)" — Rejected: a config knob or flag for the interval (hardcoded 10, matching the §5 hardcoded-30m precedent); emitting `fleet_summary:` alongside `fleet:` (defeats the byte-saving); a small table for the compact frame (one line is the point); making `--quiet` the default of `--diff` (breaks byte-identity/back-compat and the version-skew fallback); a Go daemon heartbeat (`[2ne8]`, separate).
- `docs/memory/distribution/kit-architecture.md` ~line 88 (`tick-start` bullet): add `--diff [--quiet]` and the summary block in one clause.
- `docs/specs/skills.md` ~line 1112 Flow line: `fab operator tick-start --diff --quiet (snapshot + deltas + fleet or fleet_summary; …)`; ~1095 Purpose line mentions `tick-start --diff` — add `--quiet`.
- `docs/specs/harness-adapters.md` — grep confirmed it does not mention the tick document (only "sub-poll tick" of `fab dispatch wait`); no edit expected.
- Grep sweep before finishing apply: `tick-start --diff` (all occurrences in `src/kit/skills/`, `docs/memory/`, `docs/specs/`), `three YAML blocks`, `the status frame's data source`, `fleet:` (skill/memory prose), and `Status Frame Format` — update every occurrence that states the always-full-fleet claim.

### Decisions (delegated by the user's operator — Confident, not Unresolved)

- (a) The periodic full-frame interval is a hardcoded binary constant `10`, not a flag or config knob (no new config surface; matches the §5 hardcoded-30m idle auto-default precedent).
- (b) `fleet_summary:` REPLACES `fleet:` on a quiet tick rather than being added alongside — fewer bytes per tick is the whole point.
- (c) The compact frame is one line, not a small table.
- (d) `--quiet` is opt-in on the binary but the skill always passes it; flagless `--diff` stays byte-identical for back-compat and the version-skew fallback.
- (e) `change_type` = `feat` (new flag + new frame mode).

### Out of scope

- `_cli-fab.md` load slimming (user rejected).
- Go-side daemon heartbeat (backlog `[2ne8]`).
- Any change to the loop prompt / cadence (`### Loop Prompt`, `### Post-Compaction Reload` — 4q3l, PR #625).
- Changing what `candidates:` contains or how §5 sweeps it.

## Affected Memory

- `runtime/operator`: (modify) tick lifecycle (`--diff --quiet` default, `fleet_summary:` block, compact one-line frame, watches-only-on-news rule, idle message as the only other per-tick output, `--quiet`-unknown fallback rung); new Design Decision for the quiet/compact model
- `distribution/kit-architecture`: (modify) `fab operator tick-start` bullet — `--diff [--quiet]` and the `fleet_summary:` block

## Impact

- **Go**: `src/go/fab/cmd/fab/operator_tick_start.go` (flag, constant, summary struct + helper, quiet emit path, `--quiet`-requires-`--diff` guard); `src/go/fab/cmd/fab/operator_tick_diff_test.go` (6 new tests + harness variant). Run `go test ./src/go/fab/cmd/fab/ -run 'TestOperatorTick'` first, then the package.
- **Skills**: `src/kit/skills/fab-operator.md` §4 (Tick Behavior step 1, Status Frame Format, Idle Message) + §9 pointer; `src/kit/skills/_cli-fab.md` § fab operator tick-start. Constitution: CLI change ⇒ `_cli-fab.md` + tests.
- **Docs**: `docs/memory/runtime/operator.md`, `docs/memory/distribution/kit-architecture.md`, `docs/specs/skills.md`. `fab memory-index` regen after memory edits.
- **Runtime behavior**: the operator's per-tick output on a quiet fleet drops from ~(9×N + frame) lines to ~(3 + 5 + 1) lines; a full document every 10th tick (30 min at 3m cadence, 15 min at 90s). No state-file schema change (no migration). No change to deltas, candidates, or baseline-write semantics.
- **Back-compat**: flagless `--diff` byte-identical (existing test guards it); an older binary rejects `--quiet` → the skill drops it for the session.

## Open Questions

- None blocking. Noted for the apply worker: the full-frame header's `{N} tracked` "includes changes + watches"; the compact line follows the same definition (changes from `fleet_summary.tracked`, watches from the step-3 state read) — see Assumption 8.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Full-frame interval is a hardcoded `tickQuietFullEvery = 10` constant, no flag/config | Delegated decision (a); matches §5 hardcoded-30m precedent; trivially changeable later | S:90 R:85 A:80 D:85 |
| 2 | Confident | `fleet_summary:` replaces `fleet:` on quiet ticks; never both keys | Delegated decision (b); byte-saving is the goal | S:90 R:75 A:80 D:85 |
| 3 | Confident | Compact frame is one line, not a table | Delegated decision (c) | S:90 R:90 A:80 D:80 |
| 4 | Confident | `--quiet` opt-in on the binary, always passed by the skill; flagless `--diff` byte-identical | Delegated decision (d); existing `FlaglessByteIdentical` test and version-skew fallback depend on it | S:90 R:80 A:85 D:85 |
| 5 | Confident | `change_type` = `feat` | Delegated decision (e); new flag + new frame mode | S:90 R:95 A:90 D:90 |
| 6 | Certain | `--quiet` without `--diff` is a usage error checked before any state read/write (no tick increment) | Description mandates the error; a failed flag combo must not consume a tick | S:85 R:90 A:90 D:90 |
| 7 | Certain | Quiet decision uses the post-increment `tick_count` (tick #10, #20 … are full) | Only tick counter available; `tick: N` header already prints it | S:80 R:90 A:95 D:90 |
| 8 | Confident | Compact-line `{tracked}` keeps the full header's definition (changes + watches) — changes from `fleet_summary.tracked`, watches from step 3's state read | Keeps the two frame shapes consistent; the tick already reads state in step 3, so no extra call; description's `{tracked}` placeholder does not define it | S:60 R:85 A:70 D:65 |
| 9 | Confident | Empty monitored set under `--quiet` emits the all-zero `fleet_summary` (or `fleet: []` on a 10th tick), snapshot still skipped | Description states it; the short-circuit stays as the no-op tick | S:85 R:85 A:85 D:85 |
| 10 | Confident | `unknown` counts rows with null `agent_state` (empty/`—` snapshot state); dead/mismatched rows never reach the summary because they emit deltas | Follows from `diffMonitored`: pane_death/pane_mismatch are level-triggered deltas → full document | S:80 R:85 A:90 D:85 |
| 11 | Confident | Watches table on a compact tick renders only when step 3 produced news (new items / `last_error` / auto-disable) | Description states it; keeps the quiet tick quiet unless a watch changed | S:85 R:85 A:80 D:80 |
| 12 | Confident | Version-skew: `--quiet` unknown-flag error drops `--quiet` only (keeps `--diff`) for the session, reported once — a separate rung from the existing `--diff`→flagless fallback | An older-but-post-dbwg binary still supports `--diff`; falling all the way to flagless would lose the diff | S:75 R:85 A:80 D:80 |
| 13 | Confident | Test harness: add an args-taking `runTickDiff` variant and a `FleetSummary` field on `tickDiffDoc`; assert key presence via raw stdout | Existing harness hardcodes `--diff`; contract is key absence, not emptiness | S:70 R:95 A:90 D:85 |
| 14 | Confident | `### Loop Prompt` / `### Post-Compaction Reload` are not recreated on this branch; rebase leaves them alone if #625 lands first | Explicit instruction; sections belong to 4q3l | S:90 R:90 A:90 D:90 |
| 15 | Confident | `fleet_summary` key order pinned `tracked, waiting, idle, active, unknown` | Description lists it in this order; yaml.v3 struct marshal preserves field order so no `yaml.Node` needed for the mapping itself | S:65 R:90 A:80 D:70 |

15 assumptions (2 certain, 13 confident, 0 tentative, 0 unresolved).
