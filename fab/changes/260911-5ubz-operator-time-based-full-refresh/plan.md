# Plan: Operator tick-start — time-based full-frame refresh (10 minutes) replaces every-10th-tick

**Change**: 260911-5ubz-operator-time-based-full-refresh
**Intake**: `intake.md`

## Requirements

### Runtime: tick-start full/quiet decision

#### R1: The periodic full refresh is a wall-clock age, not a tick count
`fab operator tick-start --diff --quiet` MUST emit the full document (`items:` present) when the state file's `last_full_at` is at least `tickQuietFullAfter = 10 * time.Minute` old, and MAY emit the quiet document (`fleet_summary:` in place of `items:`) only when `deltas:` and `needs_check:` are both empty AND `last_full_at` is younger than the threshold. The count constant `tickQuietFullEvery` and every `tick_count % N` term in the predicate MUST be removed. The threshold MUST be an unexported built-in constant — no flag, no config field.

- **GIVEN** a `--diff --quiet` tick with no deltas and empty `needs_check`
- **WHEN** `last_full_at` is 10 minutes old or older
- **THEN** the full document is emitted (`items:` present, `fleet_summary:` absent)

- **GIVEN** the same tick
- **WHEN** `last_full_at` is younger than 10 minutes
- **THEN** the quiet document is emitted (`fleet_summary:` present, `items:` absent), with the four-block order and `needs_check: []` unchanged

#### R2: A missing, unparseable, non-string, or future `last_full_at` counts as due
The age test MUST treat an absent key, a non-string value, a value that fails `time.Parse(time.RFC3339, …)`, or a timestamp later than `now` as "older than the threshold" (⇒ full document).

- **GIVEN** a fresh or pre-upgrade state file with no `last_full_at`
- **WHEN** a `--diff --quiet` tick runs with no deltas
- **THEN** the full document is emitted and `last_full_at` is written

- **GIVEN** `last_full_at: "not-a-time"` or a stamp one hour in the future
- **WHEN** a `--diff --quiet` tick runs with no deltas
- **THEN** the full document is emitted and the stamp is rewritten to a valid RFC3339 UTC now

#### R3: `last_full_at` is written inside the one tick mutation, on every full document
`tick-start --diff` MUST write `last_full_at` (RFC3339 UTC, the same `nowStr` as `last_tick_at`) inside the same `mutateOperatorStateClock` callback as `tick_count`/`last_tick_at`/the baselines, on every tick whose emitted document is full — whether full because `--quiet` is absent, because of deltas, because of `needs_check`, or because of the age threshold. A quiet tick MUST leave `last_full_at` byte-unchanged. The decision MUST be computed on both callback exits (the `len(items) == 0` short-circuit and the normal path) because it reads the prior value. `emitTickDiffDoc` receives the already-decided `full bool` (plus `tickCount` for the header) and computes nothing itself.

- **GIVEN** a `--diff` tick without `--quiet` and a recent `last_full_at`
- **WHEN** it runs
- **THEN** stdout is the full document exactly as today and the state file's `last_full_at` equals this tick's `last_tick_at`

- **GIVEN** an empty `tracked: []` and a stale `last_full_at`
- **WHEN** a `--diff --quiet` tick runs
- **THEN** `items: []` is emitted, the snapshot subprocess is never invoked, and `last_full_at` is rewritten

- **GIVEN** an empty `tracked: []` and a recent `last_full_at`
- **WHEN** a `--diff --quiet` tick runs
- **THEN** the all-zero `fleet_summary:` is emitted and `last_full_at` is untouched

#### R4: Flagless paths are byte-identical; the field is additive; no migration
Flagless `tick-start` (no `--diff`) MUST NOT read or write `last_full_at` and MUST keep stdout and state byte-identical to today (`TestOperatorTickStart_FlaglessByteIdentical` passes, additionally asserting no `last_full_at` key appears). Flagless `--diff` stdout MUST be byte-identical to today. No migration file is added; `convertLegacyOperatorState` is untouched. The `--quiet` flag help text and the `emitTickDiffDoc` doc comment MUST describe the time rule.

- **GIVEN** the flagless `fab operator tick-start`
- **WHEN** it runs on a state file lacking `last_full_at`
- **THEN** the written file gains only `tick_count`/`last_tick_at` changes and no `last_full_at`

### Runtime: tests

#### R5: Time-based tests replace the count-based ones
`src/go/fab/cmd/fab/operator_tick_diff_test.go` MUST: delete `TestOperatorTickDiff_QuietEveryTenthTickEmitsFullItems`; add a table-driven `TestOperatorTickDiff_QuietFullAfterTenMinutes` covering absent / unparseable / recent (quiet, stamp unchanged) / 10m-old (full, stamp rewritten) / future (full); extend `QuietWithDeltaEmitsFullItems` and `NeedsCheckForcesFullDocument` to seed a recent `last_full_at` and assert the stamp is rewritten; rewrite `QuietEmptyTracked`'s two subtests around recent vs stale stamps; adjust the seeds of `QuietNoDeltasEmitsSummary` and `QuietSummaryMixedItems` to a recent `last_full_at`; extend `FlaglessByteIdentical`'s gained-key check with `last_full_at`; add `TestOperatorTickDiff_FlaglessDiffWritesLastFullAt`. The recent-side boundary seed MUST carry a ≥ 30 s margin (e.g. 9m) unless a clock seam is introduced. `seedDiffStateAt`'s "9/19/10" comment is updated or the helper replaced by one accepting extra top-level keys. `go test ./src/go/fab/cmd/fab/...` passes and `gofmt -l src/go/fab/cmd/fab/` prints nothing.

- **GIVEN** the test package after apply
- **WHEN** `go test ./src/go/fab/cmd/fab/ -run 'TestOperatorTick' -count=1` runs
- **THEN** every test passes, none references `tickQuietFullEvery` or "10th"

### Memory-docs: CLI reference and operator skill (deployed content)

#### R6: `_cli-fab-operator.md` § fab operator tick-start documents the time rule and the new scalar
The five sites from intake § What Changes 3 MUST be updated: the intro (line ~44 — fields written gain `last_full_at` with `--diff` on full ticks), the `--quiet` bullet (~137 — the quiet condition becomes "`last_full_at` less than 10 minutes old (built-in constant, not a flag or config knob; missing/unparseable counts as stale)"), the full-document sentence (~148 — adds the age term, that every full document writes `last_full_at` in the same mutation, and the 1m-floor ≈ every-10th-tick / ≥10m ⇒ every-tick consequence), the state-path contract paragraph (~152 — one sentence: an added top-level field is additive under tolerant-read and needs no coordinated run-kit change), and the shared state-verb mechanics (~154 — `last_full_at` joins the owned scalars and the timestamp enumeration). No fab-kit-only path is cited.

- **GIVEN** the reference after apply
- **WHEN** grepped for "10th", "multiple of", "constant 10"
- **THEN** zero matches in § fab operator tick-start
- **AND** `last_full_at` appears in the intro, the `--quiet` bullet, the full-document sentence, and the scalar/timestamp enumerations

#### R7: `fab-operator.md` mirrors the rule and the state-file reference block
Line ~309 (§4 step 1: "no skill-side counter or timer is kept" with the time-based full document), line ~338 (Full-frame trigger list: "a tick 10 minutes or more after the last full document" replaces "every 10th tick"), and the state-file reference YAML (~230–231: `last_full_at` line with the trailing comment) MUST be updated. No other prose changes; no fab-kit-only path cited.

- **GIVEN** the skill after apply
- **WHEN** grepped for "10th tick"
- **THEN** zero matches
- **AND** the reference block lists `last_full_at` under `last_tick_at`

### Memory-docs: memory files (hydrate-visible, swept up front)

#### R8: `docs/memory/runtime/operator.md` and `kit-architecture.md` state the present truth
`operator.md` lines ~257 (Design Constraints), ~278 (tick-start description — rule and fields written), ~288 (tick-lifecycle usage) MUST state the time rule and name `last_full_at`; the kbf2 DD's *Introduced by* line (~527) MUST gain `; *Updated by*: 260911-5ubz-operator-time-based-full-refresh (…)` in the existing style, body untouched; a new four-field DD "Periodic Full Refresh Is Time-Based (`last_full_at` ≥ 10m), Not a Tick Count" MUST be appended per this plan's § Design Decisions (its **Rejected** names the fab-written HTML fleet table as run-kit's job). `kit-architecture.md:90` MUST name `last_full_at` among the fields written and state the time rule. `operator.md` ~488 (tolerant-read DD) is verify-only. `docs/memory/*/log.md`, `docs/specs/findings/*`, `src/kit/migrations/2.24.9-to-2.25.0.md`, `docs/specs/operator.md` MUST NOT be edited.

- **GIVEN** the two memory files after apply
- **WHEN** grepped for "multiple of 10", "every 10th", "constant 10"
- **THEN** the only matches are inside the kbf2 DD body (history) and `log.md` files
- **AND** the new DD exists with all four fields and the full change name

### Non-Goals

- An HTML or markdown fleet artifact written by fab beside the state file — user decision: run-kit owns any fleet UI and already reads the state YAML at the pinned path. Not a follow-up.
- A flag or config knob for the interval (kbf2 posture).
- Changes to the compact/full frame format cells; changes to the kp3d §1 principles text (merged separately in PR #665).
- Any run-kit change (the cross-repo contract is path + slug; the field is additive).
- Backlog `[2ne8]` daemon heartbeat and `[nr3a]` tick-completion — separate.

### Design Decisions

#### Periodic Full Refresh Is Time-Based (`last_full_at` ≥ 10m), Not a Tick Count
**Decision**: `fab operator tick-start --diff --quiet` emits the full `items:` document whenever the state file's `last_full_at` is at least 10 minutes old (built-in constant `tickQuietFullAfter`, no flag or config knob), and every full document — quiet-mode-or-not, delta- or needs_check-forced or age-forced — rewrites `last_full_at` (RFC3339 UTC) inside the same atomic tick mutation as `tick_count`/`last_tick_at`. A missing, unparseable, or future stamp counts as due. The every-10th-tick constant is deleted.
**Why**: The operator's clock is run-kit's cron entry with backoff `1m → 30m`, so "every 10th tick" is a count over a variable interval — roughly every 10 minutes at the floor but about 5 hours at full backoff, which is the "frequency is too low" the user observed on 2026-09-11. A wall-clock age is cadence-agnostic: identical to today at the 1m floor, and every tick is full once the cadence reaches 10 minutes, which is the right answer for a tick that rare. The binary already owns the tick bookkeeping and the full/quiet decision, so one more scalar in the same mutation is the smallest correct change.
**Rejected**: A flag or config knob (kbf2's rejection stands — the §5 hardcoded-30m idle default is the same posture). A skill-side timer (the skill keeps no counter or timer by design). Deriving the interval from the cron schedule (couples fab to run-kit's schedule shape for nothing a duration does not already give). An HTML fleet table written by fab beside the state file — run-kit already consumes the operator state YAML at the pinned cross-repo path, so any fleet UI is run-kit's to render; fab writes no HTML.
*Introduced by*: 260911-5ubz-operator-time-based-full-refresh

### Deprecated Requirements

#### Every-10th-tick full document
**Reason**: A tick count is the wrong unit under the variable backoff cadence; at full backoff the periodic full frame degraded to roughly every five hours.
**Migration**: Replaced by the `last_full_at` ≥ 10m rule. State files lacking the field render a full frame on their first tick; no migration file.

## Tasks

### Phase 1: Setup

- [x] T001 Baseline: run `go test ./src/go/fab/cmd/fab/ -run 'TestOperatorTick' -count=1` to confirm green before edits; record `grep -rn "10th\|multiple of\|tickQuietFullEvery\|constant 10" src/kit/skills/_cli-fab-operator.md src/kit/skills/fab-operator.md docs/memory/runtime/operator.md docs/memory/distribution/kit-architecture.md src/go/fab/cmd/fab/operator_tick_start.go src/go/fab/cmd/fab/operator_tick_diff_test.go` as the sweep checklist for T006–T009 <!-- R5 R6 R7 R8 -->

### Phase 2: Core Implementation

- [x] T002 In `src/go/fab/cmd/fab/operator_tick_start.go`: replace `const tickQuietFullEvery = 10` with `const tickQuietFullAfter = 10 * time.Minute` and its doc comment (intake § 1a wording); add `func tickFullDue(raw interface{}, now time.Time) bool` returning true for missing / non-string / unparseable / future stamps, else `now.Sub(t) >= tickQuietFullAfter` <!-- R1 R2 -->
- [x] T003 In `runOperatorTickStartDiff`: compute `full := !quiet || len(out.Deltas) > 0 || len(out.NeedsCheck) > 0 || tickFullDue(data["last_full_at"], now)` inside the `mutateOperatorStateClock` callback on BOTH exits (the `len(items) == 0` short-circuit and after `summarizeItems`), write `data["last_full_at"] = nowStr` when full, capture `full` for the emit; change `emitTickDiffDoc(w, out, quiet bool, tickCount int, now)` to take the decided `full bool` (keep `tickCount` for the header) and rewrite its doc comment to the time rule <!-- R1 R3 -->
- [x] T004 Update the `--quiet` flag help at `operator_tick_start.go:39` to the time rule ("…on a no-delta tick within 10m of the last full document…"); confirm the flagless `runOperatorTickStart` path neither reads nor writes `last_full_at`; run `gofmt -l src/go/fab/cmd/fab/` <!-- R4 -->
- [x] T005 In `src/go/fab/cmd/fab/operator_tick_diff_test.go`: add a seed helper accepting extra top-level keys (or use `withOperatorState` raw YAML); delete `TestOperatorTickDiff_QuietEveryTenthTickEmitsFullItems`; add table-driven `TestOperatorTickDiff_QuietFullAfterTenMinutes` (absent ⇒ full + stamp written; `"not-a-time"` ⇒ full + valid stamp; `rfc3339Ago(9*time.Minute)` ⇒ quiet + stamp byte-unchanged; `rfc3339Ago(10*time.Minute)` ⇒ full + stamp == now; `rfc3339Ago(-time.Hour)` ⇒ full); extend `QuietWithDeltaEmitsFullItems` and `NeedsCheckForcesFullDocument` with a recent stamp and assert it is rewritten; rewrite `QuietEmptyTracked` subtests (recent ⇒ all-zero summary, stale ⇒ `items: []`, snapshot never invoked); seed a recent stamp in `QuietNoDeltasEmitsSummary` and `QuietSummaryMixedItems`; add `last_full_at` to `FlaglessByteIdentical`'s gained-key loop; add `TestOperatorTickDiff_FlaglessDiffWritesLastFullAt`; update the `seedDiffStateAt` "9/19/10" comment; run `go test ./src/go/fab/cmd/fab/ -run 'TestOperatorTick' -count=1` then `go test ./src/go/fab/cmd/fab/...` <!-- R5 -->

### Phase 3: Integration & Edge Cases

- [x] T006 [P] `src/kit/skills/_cli-fab-operator.md` § fab operator tick-start: update the intro (~44), the `--quiet` bullet (~137), the full-document sentence (~148), the state-path contract paragraph (~152, additive-field sentence), and the shared state-verb scalar + timestamp enumerations (~154) per intake § 3; cite no fab-kit-only paths <!-- R6 -->
- [x] T007 [P] `src/kit/skills/fab-operator.md`: update §4 step 1 (~309), the Full-frame trigger list (~338), and add `last_full_at: "2026-09-11T16:30:00Z"  # written on every full tick document; the 10m periodic-refresh clock` to the state-file reference block (~230–231) <!-- R7 -->
- [x] T008 [P] `docs/memory/runtime/operator.md`: rewrite lines ~257, ~278 (rule + fields written + same-mutation sentence), ~288 to the time rule; append `; *Updated by*: 260911-5ubz-operator-time-based-full-refresh (the periodic full refresh is time-based — \`last_full_at\` ≥ 10m — not every 10th tick)` to the kbf2 DD's *Introduced by* line; append the new DD from this plan's § Design Decisions; verify ~488 tolerant-read DD needs no change <!-- R8 -->
- [x] T009 [P] `docs/memory/distribution/kit-architecture.md:90`: add `last_full_at` (with `--diff`, on full-document ticks) to the fields written and replace the "not a multiple of the built-in 10 … every 10th tick" clause with the time rule <!-- R8 -->

### Phase 4: Polish

- [x] T010 Final sweep: re-run the T001 grep — permitted remaining hits are only the kbf2 DD body in `operator.md` and the `log.md` files; confirm `git diff --name-only` lists only the two Go files, the two skill files, the two memory files, and `fab/changes/260911-5ubz-*/`; confirm `go vet ./src/go/fab/cmd/fab/` and `gofmt -l src/go/fab/cmd/fab/` are clean; run the deployed-content path guard test in `src/go/fab-kit/cmd/fab/` (`go test ./src/go/fab-kit/cmd/fab/ -run 'Deploy|Path|Cite' -count=1`, or the whole package if the name is unknown) <!-- R4 R6 R7 R8 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `tickQuietFullAfter = 10 * time.Minute` exists, `tickQuietFullEvery` and every `tick_count %` term are gone, and the predicate uses `tickFullDue(data["last_full_at"], now)`
- [x] A-002 R2: `tickFullDue` returns true for a missing key, a non-string, an unparseable string, and a future timestamp; false only for a valid stamp younger than 10 minutes
- [x] A-003 R3: `last_full_at` is written inside the `mutateOperatorStateClock` callback on both exits whenever the document is full, and `emitTickDiffDoc` takes the decided `full bool`
- [x] A-004 R4: `--quiet` help text and the `emitTickDiffDoc` doc comment describe the time rule; flagless `tick-start` never touches `last_full_at`; no migration file added; `convertLegacyOperatorState` untouched
- [x] A-005 R5: The test table in intake § 2 is implemented — `QuietEveryTenthTick…` deleted, `QuietFullAfterTenMinutes` and `FlaglessDiffWritesLastFullAt` added, the five listed tests extended or re-seeded, `FlaglessByteIdentical` checks `last_full_at`
- [x] A-006 R6: All five `_cli-fab-operator.md` sites updated; `last_full_at` named in the intro, `--quiet` bullet, full-document sentence, and the scalar and timestamp enumerations; the additive-field / no-run-kit-change sentence present
- [x] A-007 R7: `fab-operator.md` ~309, ~338, and the state-file reference block updated; no other prose changed
- [x] A-008 R8: `operator.md` ~257/~278/~288 state the time rule and name `last_full_at`; kbf2 DD gains the *Updated by* clause with body untouched; the new DD exists with four fields and names the run-kit-owns-HTML rejection; `kit-architecture.md:90` updated

### Behavioral Correctness

- [x] A-009 R1: A `--diff --quiet` tick with no deltas and `last_full_at` 10m old emits `items:`; with `last_full_at` 9m old emits `fleet_summary:` and leaves the stamp byte-unchanged
- [x] A-010 R3: A `--diff` tick without `--quiet` writes `last_full_at == last_tick_at`; a delta-forced or needs_check-forced full tick with a recent stamp also rewrites it
- [x] A-011 R4: Flagless `--diff` stdout is byte-identical to before (four-block order, `needs_check: []` explicit, no new stdout key)

### Removal Verification

- [x] A-012 R1: `grep -rn "tickQuietFullEvery" src/go` returns nothing; `grep -n "10th\|multiple of" src/go/fab/cmd/fab/operator_tick_start.go src/go/fab/cmd/fab/operator_tick_diff_test.go` returns nothing
- [x] A-013 R6 R7: `grep -n "10th tick\|multiple of the built-in constant 10\|constant 10" src/kit/skills/_cli-fab-operator.md src/kit/skills/fab-operator.md` returns nothing

### Scenario Coverage

- [x] A-014 R2: A state file with no `last_full_at` (fresh, or pre-upgrade) produces a full document on its first `--diff --quiet` tick and gains the key
- [x] A-015 R3: `tracked: []` + stale stamp ⇒ `items: []` with the snapshot subprocess never invoked and the stamp rewritten; `tracked: []` + recent stamp ⇒ all-zero `fleet_summary:` with the stamp untouched
- [x] A-016 R5: `go test ./src/go/fab/cmd/fab/...` passes; the recent-side boundary seed carries a ≥ 30 s margin or a clock seam exists

### Edge Cases & Error Handling

- [x] A-017 R2: `last_full_at: "not-a-time"` and a future stamp both yield a full document and a rewritten valid RFC3339 UTC stamp — never an error exit
- [x] A-018 R4: The run-kit cross-repo contract paragraph in `_cli-fab-operator.md` states the field is additive and needs no coordinated run-kit change; nothing under `docs/specs/findings/`, `docs/memory/*/log.md`, or `src/kit/migrations/` is edited

### Code Quality

- [x] A-019 Pattern consistency: `tickFullDue` follows the tolerant-read style of `nextTickCount` (type-switch / parse, never panics); the new DD matches the file's four-field entries; test names follow `TestOperatorTickDiff_*`
- [x] A-020 No unnecessary duplication: one `nowStr` is reused for both stamps; the seed helper is shared across the re-seeded tests rather than copy-pasted YAML
- [x] A-021 CLI ⇒ docs + tests: the Go behavior change lands with `_cli-fab-operator.md` updates and test updates in the same change (Constitution Additional Constraints; code-quality.md anti-pattern)
- [x] A-022 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`
- [x] A-023 Deployed content cites no fab-kit-only paths: the two skill files reference no `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` path (Constitution V; the Go guard passes)
- [x] A-024 No migration for a binary-owned additive field; `gofmt -l` prints nothing (code-quality.md § Test Strategy; prior CI gofmt failure)
- [x] A-025 Sibling sweep done up front: every site in intake § 3–5 is edited or recorded verify-only in the apply summary

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Hydrate note: T008/T009 land the memory edits and the new DD during apply; hydrate verifies present-truth wording and regenerates indexes with the **dev** binary's semantics in mind (the released 2.25.0 `fab docs-index` header is current with main — a header-only index diff is legitimate, not a regression).

## Deletion Candidates

- None — this change removed its own redundancies in-diff (the `tickQuietFullEvery` constant and the `seedDiffStateAt` helper were deleted, not left behind); review found no surviving code made redundant

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `full` is computed inside the mutation callback and passed to `emitTickDiffDoc` as a bool; exact helper shape is apply's | Intake assumption 10; the contract is prior-read + same-mutation write | S:85 R:95 A:90 D:90 |
| 2 | Confident | The recent-side boundary test seeds 9m (≥ 30 s margin) rather than 9m59s; no clock seam is added unless apply finds it trivial | Intake assumption 8; a same-second flake is a real CI risk and a seam is more code than the change needs | S:65 R:95 A:80 D:70 |
| 3 | Certain | Memory files (`operator.md`, `kit-architecture.md`) are edited during apply (T008/T009) and verified by hydrate | Sibling-sweep rule: the memory file documenting the behavior is in the sweep class; same choreography as kp3d | S:80 R:95 A:90 D:90 |
| 4 | Certain | `change_type` stays `fix` as inferred | Intake assumption 12 | S:70 R:95 A:80 D:70 |
| 5 | Certain | The deployed-content path guard in `src/go/fab-kit/cmd/fab/` is run as part of T010 even though the skill edits are small | Constitution V is a must-fix review rule; running the guard is cheap | S:80 R:95 A:90 D:90 |
| 6 | Confident | T006–T009 are `[P]` (four distinct files) | Informational for a sequential worker | S:60 R:95 A:85 D:75 |

6 assumptions (4 certain, 2 confident, 0 tentative).
