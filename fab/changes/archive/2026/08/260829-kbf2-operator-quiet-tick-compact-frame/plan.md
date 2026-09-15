# Plan: Operator Quiet Tick + Compact No-Change Status Frame

**Change**: 260829-kbf2-operator-quiet-tick-compact-frame
**Intake**: `intake.md`

## Requirements

### CLI: `fab operator tick-start --quiet`

#### R1: `--quiet` flag with `--diff`-only validity
`fab operator tick-start` MUST accept a bool `--quiet` flag. `--quiet` without `--diff` MUST fail with a one-line error (`--quiet requires --diff`) before any state read or write — `tick_count` MUST NOT increment.

- **GIVEN** a state file with `tick_count: 5`
- **WHEN** `fab operator tick-start --quiet` runs
- **THEN** it exits non-zero with the error and `tick_count` is still 5

#### R2: Quiet tick replaces `fleet:` with `fleet_summary:`
With `--diff --quiet`, when `deltas` is empty AND the post-increment `tick_count % tickQuietFullEvery != 0` (constant `tickQuietFullEvery = 10`), stdout MUST carry `deltas:`, `candidates:`, then a `fleet_summary:` mapping with the pinned keys `tracked, waiting, idle, active, unknown` (ints) — and MUST NOT carry a `fleet:` key. `tracked` = number of monitored entries; the other four count rows by snapshot agent state (`unknown` = null/empty/em-dash).

- **GIVEN** 6 monitored entries (2 waiting, 1 idle, 2 active, 1 unknown), no deltas, `tick_count` becoming 6
- **WHEN** `--diff --quiet` runs
- **THEN** output has `fleet_summary: {tracked: 6, waiting: 2, idle: 1, active: 2, unknown: 1}`, `candidates:` present, and no `fleet:` key

#### R3: Full document on deltas or every 10th tick
With `--diff --quiet`, when `deltas` is non-empty OR `tick_count % 10 == 0`, the output MUST be byte-identical to plain `--diff` (full `fleet:` block, no `fleet_summary:` key). Never both keys.

- **GIVEN** `tick_count` becoming 10 (or 20) with no deltas, OR any tick with a `stage_advance` delta
- **WHEN** `--diff --quiet` runs
- **THEN** `fleet:` is present and `fleet_summary:` is absent

#### R4: Empty-monitored short-circuit under `--quiet`
With an empty monitored set, the pane-snapshot subprocess MUST still be skipped; `--diff --quiet` emits the all-zero `fleet_summary` on a non-10th tick and `fleet: []` on a 10th tick.

- **GIVEN** no monitored entries and `tick_count` becoming 6
- **WHEN** `--diff --quiet` runs
- **THEN** snapshot is not invoked and output has `fleet_summary: {tracked: 0, waiting: 0, idle: 0, active: 0, unknown: 0}`

#### R5: Flagless and plain `--diff` stay byte-identical
Existing `TestOperatorTickStart_FlaglessByteIdentical`, `TestOperatorTickDiff_EmptyMonitoredSkipsSnapshot`, and `TestOperatorTickDiff_Fleet` MUST pass unchanged.

- **GIVEN** the existing test suite
- **WHEN** `go test ./src/go/fab/cmd/fab/ -run 'TestOperatorTick'` runs
- **THEN** all pre-existing tests pass without modification

#### R6: `_cli-fab.md` documents the flag
`src/kit/skills/_cli-fab.md` § fab operator tick-start MUST show `[--diff [--quiet]]`, the `fleet_summary:` block shape, the quiet/full rule, the always-emitted `candidates:`, the never-both-keys invariant, the empty-monitored case, and that 10 is a built-in constant.

- **GIVEN** a reader of `_cli-fab.md`
- **WHEN** they look up `tick-start`
- **THEN** every `--quiet` behavior in R1–R4 is stated there

### Skill: fab-operator §4

#### R7: Tick runs `--diff --quiet` by default; softer version-skew rung
Tick Behavior step 1 MUST run `fab operator tick-start --diff --quiet`, drop `--quiet` only on a user status request ("status", "any updates?", "show the fleet"), describe `fleet_summary:` as appearing in place of `fleet:`, and add a version-skew rung: `--quiet` unknown → drop `--quiet` only (keep `--diff`) for the session, report once.

- **GIVEN** an older binary that rejects `--quiet`
- **WHEN** the tick's first call errors
- **THEN** the operator re-runs with `--diff` alone for the rest of the session and reports the mismatch once

#### R8: Compact frame rule (single owner: § Status Frame Format)
When the tick document carries `fleet_summary:`, the operator MUST emit exactly one line `🛰️ **Operator** · {HH:MM} · tick #{N} · **{tracked} tracked** · no change`, appending ` · {W} waiting` only when `waiting > 0`; `{tracked}` = `fleet_summary.tracked` + watch count from step 3's state read. The full frame renders when `fleet:` is present. The Watches table renders on a compact tick only when step 3 produced news (new items / `last_error` / auto-disable). The italic action footnote renders on either shape when an action happened. The Element table gains a `Compact frame` row. §9 carries a pointer clause only.

- **GIVEN** a quiet tick with `fleet_summary: {tracked: 8, waiting: 1, …}` and 2 watches with no news
- **WHEN** the frame renders
- **THEN** it is the single line `🛰️ **Operator** · 17:32 · tick #47 · **10 tracked** · no change · 1 waiting` and nothing else besides the idle message

#### R9: Idle message is the only other per-tick output
§ Idle Message MUST state that the idle message is the only other per-tick output — no restated tick document, no echoed `candidates:`, no per-candidate "no question" lines.

- **GIVEN** a quiet tick
- **WHEN** the operator finishes the tick
- **THEN** its message body contains the compact frame, optionally the action footnote, and the idle message — nothing else

### Docs: sibling sweep

#### R10: Memory and specs reflect the quiet tick
`docs/memory/runtime/operator.md` (Monitoring tick ~161, frame-rendering ~159, snapshot bullets ~268/~286, usage ~290, plus one four-field Design Decision), `docs/memory/distribution/kit-architecture.md` (~88 `tick-start` bullet), and `docs/specs/skills.md` (~1095, ~1112) MUST carry `--diff --quiet`, `fleet_summary:`, the compact frame, and the every-10th-tick rule; a repo-wide grep for `tick-start --diff`, `three YAML blocks`, and `the status frame's data source` (excluding `.claude/`, `fab/changes/archive/`, `docs/specs/findings/`, `log*.md`) MUST find no surviving always-full-fleet claim.

- **GIVEN** the sweep grep after apply
- **WHEN** it runs
- **THEN** every hit is consistent with `fleet:` OR `fleet_summary:` carrying the frame data

### Non-Goals
- `_cli-fab.md` load slimming (user rejected).
- Go daemon heartbeat (backlog `[2ne8]`).
- `### Loop Prompt` / `### Post-Compaction Reload` (4q3l, PR #625 — absent on this branch; do not recreate).
- Any change to `candidates:` content or §5 sweep semantics; no state-file schema change.

### Design Decisions

#### Quiet Tick Replaces `fleet:` With Counts; Compact One-Line Frame on No-Change Ticks
**Decision**: `fab operator tick-start --diff --quiet` replaces the `fleet:` block with a five-count `fleet_summary:` on a no-delta tick that is not a multiple of the built-in constant 10; the skill renders a one-line compact frame from it and the full frame only when `fleet:` is present. Flagless `--diff` stays byte-identical and the skill always passes `--quiet`.
**Why**: Every tick appended a full fleet document plus a full multi-table frame even when nothing changed, so context grew linearly with wall-clock time on a quiet fleet and real events were buried in identical frames. The binary already owns "did anything change" (deltas), so it is the right place to decide the payload; the every-10th-tick full document keeps a periodic complete view without a config surface.
**Rejected**: A flag or config knob for the interval (hardcoded 10 mirrors the §5 30m precedent); emitting `fleet_summary:` alongside `fleet:` (no byte saving); a small table for the compact frame (one line is the point); making `--quiet` the default of `--diff` (breaks byte-identity and the version-skew fallback); a Go daemon heartbeat (`[2ne8]`, separate).
*Introduced by*: 260829-kbf2-operator-quiet-tick-compact-frame

## Tasks

### Phase 2: Core Implementation

- [x] T001 `src/go/fab/cmd/fab/operator_tick_start.go`: add `--quiet` flag, `tickQuietFullEvery = 10` constant, the `--quiet requires --diff` guard before any state I/O, `tickFleetSummary` struct + `summarizeFleet([]tickFleetRow)` helper, and an extracted emit step that writes `deltas`/`candidates` then either `fleet` or `fleet_summary` (pinned order, omitted key genuinely absent) <!-- R1 R2 R3 R4 -->
- [x] T002 `src/go/fab/cmd/fab/operator_tick_diff_test.go`: add an args-taking `runTickDiff` variant and a `FleetSummary` field on `tickDiffDoc`; add the six tests (`QuietNoDeltasEmitsSummary`, `QuietWithDeltaEmitsFullFleet`, `QuietEveryTenthTickEmitsFullFleet`, `QuietRequiresDiff`, `QuietSummaryCounts`, `QuietEmptyMonitored`) asserting key presence via raw stdout; run `go test ./src/go/fab/cmd/fab/ -run 'TestOperatorTick'` then the package <!-- R1 R2 R3 R4 R5 -->
- [x] T003 [P] `src/kit/skills/_cli-fab.md` § fab operator tick-start: usage `[--diff [--quiet]]`, `--quiet` bullet, `fleet_summary:` block shape, rules and constant note <!-- R6 -->
- [x] T004 [P] `src/kit/skills/fab-operator.md` §4 Tick Behavior step 1: `--diff --quiet` default, user-status exception, `fleet_summary:` description, softer version-skew rung <!-- R7 -->
- [x] T005 `src/kit/skills/fab-operator.md` §4 Status Frame Format: Compact frame rule (owner), Watches-only-on-news, footnote on either shape, Element-table row; § Idle Message only-other-output sentence; §9 pointer clause <!-- R8 R9 -->

### Phase 3: Integration & Edge Cases

- [x] T006 Sweep `docs/memory/runtime/operator.md` (~159, ~161, ~268, ~286, ~290 + the Design Decision from this plan lifted verbatim), `docs/memory/distribution/kit-architecture.md` ~88, `docs/specs/skills.md` ~1095/~1112 <!-- R10 -->
- [x] T007 Repo-wide grep (excl. `.claude/`, `fab/changes/archive/`, `docs/specs/findings/`, `log*.md`) for `tick-start --diff`, `three YAML blocks`, `the status frame's data source`, `fleet:` prose claims; fix stragglers; `go vet ./src/go/fab/...` and `gofmt -l src/go/fab` clean <!-- R10 R5 -->

### Phase 4: Polish

- [x] T008 `just install && fab sync` so deployed copies + local binary match; verify `fab operator tick-start --quiet` errors and `--diff --quiet` runs against the installed binary <!-- R1 R6 -->

## Execution Order

- T001 blocks T002 (tests exercise the new code)
- T003/T004 are independent of T001 and of each other; T005 after T004 (same file, adjacent sections)
- T006–T008 after all of Phase 2

## Acceptance

### Functional Completeness

- [x] A-001 R1: `--quiet` without `--diff` errors with `--quiet requires --diff` and does not increment `tick_count` (test `QuietRequiresDiff`)
- [x] A-002 R2: quiet no-delta non-10th tick emits `fleet_summary:` with pinned keys/order and no `fleet:` key (tests `QuietNoDeltasEmitsSummary`, `QuietSummaryCounts`)
- [x] A-003 R3: a delta tick or a 10th/20th tick under `--quiet` emits `fleet:` and no `fleet_summary:` (tests `QuietWithDeltaEmitsFullFleet`, `QuietEveryTenthTickEmitsFullFleet`)
- [x] A-004 R4: empty monitored set under `--quiet` skips the snapshot and emits the all-zero summary / `fleet: []` on a 10th tick (test `QuietEmptyMonitored`)
- [x] A-005 R6: `_cli-fab.md` § tick-start documents usage, block shape, rules, constant
- [x] A-006 R7: Tick Behavior step 1 runs `--diff --quiet`, names the user-status exception and the `--quiet`-only fallback rung
- [x] A-007 R8: § Status Frame Format owns the compact-frame rule with the exact one-line literal, Watches-only-on-news, footnote rule, and an Element-table row; §9 only points
- [x] A-008 R9: § Idle Message states it is the only other per-tick output
- [x] A-009 R10: memory/spec sweep files carry `--diff --quiet`, `fleet_summary:`, compact frame, 10th-tick rule; DD present in `docs/memory/runtime/operator.md`

### Behavioral Correctness

- [x] A-010 R5: pre-existing `FlaglessByteIdentical`, `EmptyMonitoredSkipsSnapshot`, `Fleet` tests pass unmodified; full package `go test ./src/go/fab/cmd/fab/` green
- [x] A-011 R2: `tracked == waiting + idle + active + unknown` on every quiet tick (asserted in `QuietSummaryCounts`)

### Scenario Coverage

- [x] A-012 R3: `tick_count` 9→10 and 19→20 produce full fleet; 10→11 produces summary (in `QuietEveryTenthTickEmitsFullFleet`)

### Edge Cases & Error Handling

- [x] A-013 R1: the guard fires before `operatorStatePath()`/`loadOperatorState` — no state file is created by an invalid invocation

### Code Quality

- [x] A-014 Canonical source only: no edits under `.claude/skills/`; `fab sync` regenerates copies
- [x] A-015 CLI ⇒ docs + tests: `_cli-fab.md` updated and tests shipped alongside the Go change
- [x] A-016 No god function: `runOperatorTickStartDiff` stays ≤ 50 lines via the extracted emit step
- [x] A-017 Owner-or-pointer: the compact-frame rule is stated once (§ Status Frame Format); Tick Behavior, §9, memory, and specs point or summarize without a second full restatement
- [x] A-018 `gofmt -l src/go/fab` empty and `go vet` clean
- [x] A-019 Sibling sweep complete: no surviving always-full-`fleet:` claim in skills/memory/specs (excl. archive/findings/logs)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality (the `--quiet` flag, `fleet_summary:` emit path, compact frame) without making existing code redundant; the plain `--diff` full-document path is retained deliberately (byte-identity back-compat and the version-skew fallback).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The quiet path marshals a second output struct (`Deltas, Candidates, FleetSummary`) rather than a `yaml.Node`; yaml.v3 preserves struct field order | Simpler than node building; key absence is guaranteed by the struct having no `Fleet` field | S:80 R:90 A:85 D:85 |
| 2 | Confident | The `--quiet`-requires-`--diff` guard lives at the top of `runOperatorTickStart`, before the `diff` branch | The only place both flags are visible before any I/O | S:85 R:95 A:90 D:90 |
| 3 | Confident | Test seeds for the 10th-tick cases set `tick_count` directly in the seeded state (`9`, `19`, `10`) | `seedDiffState` writes the state map; the counter is a plain top-level int | S:75 R:90 A:85 D:80 |

3 assumptions (0 certain, 3 confident, 0 tentative).
