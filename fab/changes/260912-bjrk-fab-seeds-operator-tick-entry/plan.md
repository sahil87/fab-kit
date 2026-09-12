# Plan: fab Seeds the Operator-Tick Cron Entry — the Reconcile Owns the Whole Entry Lifecycle

**Change**: 260912-bjrk-fab-seeds-operator-tick-entry
**Intake**: `intake.md`

## Requirements

### Runtime: operator clock seed (Go)

#### R1: Zero `role:operator` rows ⇒ exactly one seed, then one re-resolve
`resolveOperatorCronRow()` is split into `listOperatorCronRows() ([]operatorCronRow, bool)` (ok=false only on rk failure or unparseable output; an empty server is `(nil, true)`) and `pickOperatorCronRow(candidates)` (the existing single-candidate / `operatorCronName` tiebreak rule); `resolveOperatorCronRow()` keeps its signature and pure-read behaviour. A new `ensureOperatorCronRow()` MUST: return `(zero,false)` with no seed when the list fails; return the pick with no seed when candidates exist (a tie stays a silent no-op); on zero candidates run exactly one `rk cron add` with the fully explicit argv (every value from the existing constants: `operatorCronName`, `operatorBackoffMin`, `operatorBackoffMax`, `operatorDeliver`; new constants for the wake block) — `cron add "operator tick" --name "operator tick" --backoff --min 3m --max 24m --role operator --deliver skip-if-busy --wake-on agent-state-change --wake-scope server --wake-debounce 60s --if-absent respawn --respawn rk --respawn operator --respawn -L --respawn {server} --pinned` — then list+pick once more; a failed add returns `(zero,false)` silently. `syncOperatorClock`, `muteOperatorClockIfUntracked`, and `reconcileOperatorSchedule` MUST call `ensureOperatorCronRow()` instead of the pure read. `operatorCronRow` gains `ScheduleSummary string \`json:"schedule_summary"\``.

- **GIVEN** `rk cron list --json` returns no `role:operator` row and a tracked set is live
- **WHEN** any clock entry point runs
- **THEN** exactly one `cron add` with the argv above is issued, the seeded row is resolved from the second list, and the normal mute/schedule behaviour follows
- **AND** an rk failure, an `{"ok":false,…}` envelope, or an ambiguous tie issues zero `cron add` calls

#### R2: `fab operator clock sync` heals and reads the entry
A new `operatorClockCmd()` parent with a `sync` subcommand (`cobra.NoArgs`, no flags; file `cmd/fab/operator_clock_sync.go`) MUST: load the operator state (plain non-zero exit on a read failure — it never creates the skeleton); `ensureOperatorCronRow()`; run a new `reconcileOperatorMute(data)` that mutes an unmuted row when untracked and issues `rk cron mute <id> --off` on a muted row when tracked (otherwise no call); `reconcileOperatorSchedule(data)`; re-resolve and print exactly five YAML keys — `id`, `schedule_summary`, `deliver`, `muted`, `muted_until` (`null` unless a lease is live). When the row can neither be resolved nor seeded it MUST exit non-zero with the single stderr line `ERROR: could not resolve or seed the operator-tick cron entry` (fab's root command prints returned errors with the `ERROR:` prefix). `muteOperatorClockIfUntracked` (tick-start) is unchanged.

- **GIVEN** a tracked state and a muted `role:operator` row
- **WHEN** `fab operator clock sync` runs
- **THEN** one `rk cron mute <id> --off` is issued and the five-key document prints with rk's `schedule_summary`/`deliver` verbatim, exit 0
- **AND** with an unresolvable row it exits non-zero with the one stderr line and no panic

### Kit: skill and CLI reference

#### R3: The operator skill uses the verb; the STOP moves to the verb's exit code
`src/kit/skills/fab-operator.md` MUST: replace §2 Init step 4's hand-parse and STOP with `fab operator clock sync` (a non-zero exit STOPs with the verb's error; step 5's ready-line template unchanged, sourced from the YAML); state in §4 The Clock's lead-in and Ownership paragraph that fab's reconcile seeds the entry when the server has none (`rk operator`'s launcher also seeds it until run-kit retires that copy; both key on the `role:operator` row) and that the skill never composes an `rk cron add`; switch the per-tick clock read (§4 Tick Behavior step 1, Status Frame Format header row) to `fab operator clock sync`; update the §9 Cadence key-property row. `src/kit/skills/_cli-fab-operator.md` MUST gain a `### fab operator clock sync` section (signature, five-key output, seed-if-missing, both-direction mute, schedule reconcile, exit contract) and the Clock side effect paragraph MUST state the zero-candidates seed rule with the explicit argv values. Deployed files cite no fab-kit paths.

- **GIVEN** an operator starting on a server with no operator-tick entry
- **WHEN** it follows §2 Init step 4
- **THEN** the entry is seeded by the binary and the ready line renders from the verb's YAML — no `run rk operator to seed it` STOP remains

#### R4: Spec version row
`docs/specs/operator.md` Version History MUST gain `v12` naming fab-owned clock seeding and `fab operator clock sync` (2026-09-12, change 260912-bjrk).

- **GIVEN** the spec's version table
- **WHEN** read
- **THEN** a `v12` row follows `v11`

### Non-Goals
- run-kit changes — deleting `seedOperatorTick`/`operatorTickEntrySpec` is run-kit row `[pfo3]`; the overlap window is safe (both key on the `role:operator` row)
- The deliver policy, ladder bounds, or respawn argv (`rk operator -L {server}`)
- Any `rk cron rm`; a `--json` flag or `seeded:` key on `clock sync`; a `clock:` block in the tick document
- Changing `muteOperatorClockIfUntracked`'s one-direction tick-start semantics
- An rk version probe — an older rk rejects `--wake-on` and the fail-silent path swallows it
- Marking backlog `[bjrk]` done — `/fab-archive`'s job

### Design Decisions

#### The Clock Reconcile Owns the Entry End-to-End, Seeding on Zero Candidates Only
**Decision**: fab's clock reconcile seeds the operator-tick entry when `rk cron list --json` shows no `role:operator` row — one fully explicit `rk cron add` from the existing policy constants, then one re-resolve — and keeps managing it as before (tracked-predicate mute/unmute, on-change `rk cron edit`). Seeding fires only on zero candidates, never on an rk failure or a tie, regardless of tracked-ness (an untracked set mutes the fresh entry in the same pass). `fab operator clock sync` is the heal-and-read verb Init and the per-tick frame use; its non-zero exit is the only missing-entry STOP.
**Why**: fab already owned mute, unmute, schedule, and deliver; leaving the seed in `rk operator` meant every policy change was mirrored across two repos (tnmm → run-kit #954). Hooking the seed into every reconcile entry point makes a user `rk cron rm` heal on the next tick without a restart. The verb removes the skill's last hand-parse of rk's JSON envelope and gives Init a single command that both heals and answers "what is the clock".
**Rejected**: seeding from `rk operator` only (cross-repo drift); a one-shot Init verb only (no mid-run healing); hoisting the seed above `reconcileOperatorSchedule`'s empty-set early return (the tick-start and `clock sync` paths already heal that state); a version probe for the `--wake-on` flags (fail-silent already covers an old rk); auto-seeding from the skill (the skill never composes an `add` — the binary does).
*Introduced by*: 260912-bjrk-fab-seeds-operator-tick-entry

## Tasks

### Phase 2: Core Implementation

- [x] T001 `src/go/fab/cmd/fab/operator_clock.go`: split `resolveOperatorCronRow` into `listOperatorCronRows` + `pickOperatorCronRow` (pure read preserved), add wake-block constants and `operatorCronAddArgv()`, add `ensureOperatorCronRow()` (list → seed on zero candidates → list+pick once), add `ScheduleSummary` to `operatorCronRow`, switch `syncOperatorClock`/`muteOperatorClockIfUntracked`/`reconcileOperatorSchedule` to `ensureOperatorCronRow`, add `reconcileOperatorMute(data)` (both directions). <!-- R1, R2 -->
- [x] T002 New `src/go/fab/cmd/fab/operator_clock_sync.go` (rework cycle 1: missing state file exits non-zero with a stat check + test; `operator_clock.go` header comment states fab-owned seeding; spec intro bumped to v12/twelve): `operatorClockCmd()` + `sync` (`cobra.NoArgs`), registered in `operatorCmd()`; load state via `operatorStatePath()`+`loadOperatorState` (non-zero on read failure, no skeleton creation), `ensureOperatorCronRow`, `reconcileOperatorMute`, `reconcileOperatorSchedule`, re-resolve, print the five YAML keys; non-zero + `ERROR: could not resolve or seed the operator-tick cron entry` when unresolvable. <!-- R2 -->
- [x] T003 Tests in `src/go/fab/cmd/fab/operator_clock_test.go`: sibling `stubRkCronSeeding(t, seededJSON, addErr)` (empty list first, seeded list after, records every argv) + `wantAdds` helper; cases: seed with full argv on a live set; seed then one mute on an untracked tick-start; row present ⇒ zero adds; add fails ⇒ silent no-op; list fails / `{"ok":false}` ⇒ zero adds; two-row tie ⇒ zero adds; `--name` tiebreak after seeding; `clock sync` happy path (five keys, verbatim values); `clock sync` tracked+muted ⇒ one `mute --off`; `clock sync` unresolvable ⇒ non-zero + stderr line. `gofmt -l` clean; `go test ./cmd/fab` from `src/go/fab`. <!-- R1, R2 -->

### Phase 4: Polish

- [x] T004 [P] `src/kit/skills/fab-operator.md`: §2 Init step 4 → `fab operator clock sync` (STOP = non-zero exit); step 5 source note; §4 The Clock lead-in + Ownership paragraph (fab seeds when absent; rk's launcher also seeds until run-kit retires it; skill never composes an add); §4 Tick step 1 + Status Frame Format header row → `clock sync`; §9 Cadence row. `src/kit/skills/_cli-fab-operator.md`: new `### fab operator clock sync` section + Clock side effect seed rule with the explicit argv values. <!-- R3 -->
- [x] T005 [P] `docs/specs/operator.md`: add the `v12` Version History row. Then grep `src/kit/skills` for `run rk operator to seed it` and `rk cron list --json` (skill-side reads) to confirm the STOP text is gone and remaining mentions are the rk Gate probe (`>/dev/null`) or pointers to covered steps. <!-- R4, R3 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `ensureOperatorCronRow` seeds on zero candidates with the exact argv (test asserts element-by-element) and re-resolves once
- [x] A-002 R1: the three clock entry points call `ensureOperatorCronRow`; `resolveOperatorCronRow` remains the pure read with unchanged behaviour
- [x] A-003 R2: `fab operator clock sync` exists under `fab operator clock`, prints exactly the five keys, exit 0 on a resolved row
- [x] A-004 R3: §2 Init step 4 uses the verb; the `run rk operator to seed it` STOP text no longer exists in the skill
- [x] A-005 R3: `_cli-fab-operator.md` documents `clock sync` and the seed rule in the Clock side effect paragraph
- [x] A-006 R4: `docs/specs/operator.md` has the `v12` row

### Behavioral Correctness

- [x] A-007 R1: a freshly seeded entry on an untracked tick-start receives exactly one `rk cron mute <id>` (test)
- [x] A-008 R1: rk failure, `ok:false` envelope, and a two-row tie each issue zero `cron add` calls (tests)
- [x] A-009 R2: `clock sync` on tracked+muted issues one `rk cron mute <id> --off`; on untracked+unmuted one `rk cron mute <id>`; otherwise none (tests)
- [x] A-010 R2: an unresolvable row makes `clock sync` exit non-zero with the single stderr line and no stdout document (test)

### Scenario Coverage

- [x] A-011 R1: after seeding, a second unrelated `role:operator` row present in the post-seed list is resolved by the `operator tick` name tiebreak (test)
- [x] A-012 R1: a failed `cron add` leaves no mute/edit calls and no panic (test)

### Edge Cases & Error Handling

- [x] A-013 R2: `clock sync` never creates the state skeleton — a missing state file is a plain non-zero exit (rework cycle 1: `os.Stat` gate in `operator_clock_sync.go` before `loadOperatorState`, covered by the "missing state file is a plain error, no skeleton created" test)
- [x] A-014 R1: `muteOperatorClockIfUntracked` semantics at tick-start are unchanged (existing tests still pass)
- [x] A-015 R1: policy values in the add argv come from the shared constants — no literal `"3m"`, `"24m"`, `"skip-if-busy"`, `"operator tick"` outside the const block

### Code Quality

- [x] A-016 Pattern consistency: the new verb mirrors `operatorStateCmd`/`operatorTimeCmd` cobra shape and the package's `rkCronRunner`/stub seams
- [x] A-017 No unnecessary duplication: one argv builder, one pick function reused by resolve and ensure
- [x] A-018 No magic strings: wake event/scope/debounce and the error line are named constants
- [x] A-019 Canonical sources only: kit edits in `src/kit/skills/*.md`
- [x] A-020 CLI change ⇒ reference + tests: `_cli-fab-operator.md` gains the verb section in the same change as the Go
- [x] A-021 Constitution V: no deployed file cites `docs/specs/`, `docs/memory/`, `docs/site/`, `src/go/`, or run-kit paths
- [x] A-022 **N/A**: hydrate-owned, verified post-hydrate
- [x] A-023 Test strategy: `gofmt -l` clean; `go test ./cmd/fab` green before apply finishes

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant (the skill-side `rk cron list --json` hand-parse it replaces was deleted in the same diff; `resolveOperatorCronRow` and `stubRkCron` remain in use).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Five tasks ⇒ LIGHT lane; memory edits are hydrate's (A-022) | `_pipeline.md` fork rule | S:90 R:95 A:90 D:90 |
| 2 | Confident | `clock sync` prints via the package's YAML marshaller into a fixed-order struct so key order is stable (`id, schedule_summary, deliver, muted, muted_until`) | A struct with yaml tags gives deterministic order; a map would not | S:70 R:95 A:90 D:80 |
| 3 | Confident | The stderr error line is emitted by returning an error from `RunE` (cobra prints it) with `SilenceUsage` so no usage text follows | Matches the package's other verbs' error posture | S:65 R:95 A:85 D:75 |

3 assumptions (1 certain, 2 confident, 0 tentative).
