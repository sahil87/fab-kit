# Plan: CLI Exit-Code Contract Conformance

**Change**: 260612-k4ge-cli-exit-contract-conformance
**Intake**: `intake.md`

## Requirements

### CLI: Confidence Gate Exit Code

#### R1: `fab score --check-gate` SHALL exit non-zero on gate fail
`fab score --check-gate` MUST exit non-zero when the gate result is `fail`, keeping the gate YAML on stdout. This makes the binary conform to `_preamble.md` § Common fab Commands, `_cli-fab.md` § fab score, and `_pipeline.md` Pre-flight 3 — those doc rows stay untouched and become true.

- **GIVEN** a change whose `intake.md` Assumptions score below the 3.0 intake gate
- **WHEN** `fab score --check-gate --stage intake <change>` runs
- **THEN** the gate YAML (`gate: fail`, score, threshold, counts) is printed on stdout
- **AND** the command exits non-zero (error on stderr)

- **GIVEN** a change whose score meets the gate
- **WHEN** the same command runs
- **THEN** the YAML (`gate: pass`) is printed and the exit code is 0

### State Machine: AllowedStates Enforcement

#### R2: Transitions MUST reject schema-forbidden target states
`lookupTransition` (`src/go/fab/internal/status/status.go`) MUST validate the resolved target state against `AllowedStates` for the target stage and error cleanly instead of writing a state that permanently bricks `fab preflight`. Known violations to close: `advance ship` / `advance review-pr` (would write `ready`, forbidden) and `skip intake` (would write `skipped`, forbidden).

- **GIVEN** a change with `ship: active`
- **WHEN** `fab status advance <change> ship` runs
- **THEN** the command errors (non-zero) without modifying `.status.yaml`
- **AND** `fab preflight` continues to succeed afterward

- **GIVEN** a change with `intake: active`
- **WHEN** `fab status skip <change> intake` runs
- **THEN** the command errors (non-zero) without writing `skipped`

- **GIVEN** any stage/event whose target state is allowed (e.g., `start ship`, `skip apply`, `fail review`)
- **WHEN** the event command runs from a valid source state
- **THEN** the transition succeeds exactly as before

### CLI: `fab change switch` Next Routing

#### R3: Switch SHALL derive Next: like fab-status
`Switch` (`src/go/fab/internal/change/change.go`) MUST print `Next: {routing_stage} (via {command})` where the command is the one that drives the routing stage (`intake`/`apply`/`review`/`hydrate` → `/fab-continue`, `ship` → `/git-pr`, `review-pr` → `/git-pr-review`), and print bare `Next: /fab-archive` only when every stage is `done` or `skipped` — fixing the post-review off-by-one where `/fab-archive` was printed while review-pr work remained.

- **GIVEN** a change with `ship: done` and `review-pr: active`
- **WHEN** `fab change switch <change>` runs
- **THEN** the output ends `Next: review-pr (via /git-pr-review)` (not `/fab-archive`)

- **GIVEN** a change with all six stages `done` (or trailing stages `skipped`)
- **WHEN** switch runs
- **THEN** the output ends `Next: /fab-archive`

- **GIVEN** a change with `hydrate: active`
- **WHEN** switch runs
- **THEN** the output ends `Next: hydrate (via /fab-continue)` (not `/git-pr`)

### CLI: Archive Exit Semantics

#### R4: `fab change archive` with no argument SHALL be a usage error
The command MUST use `cobra.ExactArgs(1)` so a missing argument exits non-zero with a usage error, instead of exiting 0 with help text.

- **GIVEN** a fab project
- **WHEN** `fab change archive` runs with no argument
- **THEN** the command exits non-zero with an args error

#### R5: Re-archiving a genuinely archived change SHALL soft-skip (exit 0)
`Archive()` MUST detect that an unresolvable change argument matches an archived change (flat or nested entry) and return `ErrAlreadyArchived`, so `fab change archive <archived>` prints the explicit `already archived: {change}` notice and exits 0 — making the documented soft skip reachable (it previously required the source folder to still exist).

- **GIVEN** a change that was archived to `fab/changes/archive/yyyy/mm/{name}/`
- **WHEN** `fab change archive <name>` runs again
- **THEN** stdout shows `already archived: {name}` and the exit code is 0

- **GIVEN** an argument matching no change anywhere (active or archived)
- **WHEN** `fab change archive <bogus>` runs
- **THEN** the original resolution error propagates (non-zero exit)

#### R6: `fab batch archive` SHALL exit 0 on an empty `--all` set and soft-skip archived names
The `--all`/no-args path with zero archivable changes MUST be a benign no-op (notice + `Archived 0, skipped 0, failed 0.` footer, exit 0). Explicitly named targets that are genuinely archived MUST route to the loop's `already archived, skipping` path (counted skipped, exit 0) instead of falling through `could not resolve` into the exit-1 `No valid changes` path. The explicit-args path where nothing resolves anywhere keeps exit 1 (genuine error) and is documented.

- **GIVEN** a repo with no changes at `hydrate: done|skipped`
- **WHEN** `fab batch archive` (or `--all`) runs
- **THEN** it prints `No archivable changes found.` plus the zero footer and exits 0

- **GIVEN** an explicitly named change that is already archived
- **WHEN** `fab batch archive <name>` runs
- **THEN** the output shows `{name} — already archived, skipping` and the exit code is 0

#### R7: `fab change restore --switch` SHALL surface activation failure
`Restore()` MUST record `pointer: failed` in the YAML output when the post-restore `change.Switch` call errors, instead of rendering the failure as `pointer: skipped` ("not requested").

- **GIVEN** a restore where symlink creation fails (e.g., `.fab-status.yaml` path is blocked)
- **WHEN** `fab change restore <name> --switch` runs
- **THEN** the YAML reports `pointer: failed` (move/index results still reported)

### State Machine: Review Iterations Survive Fail+Reset

#### R8: `stage_metrics` iterations MUST survive the reset/skip cascade
`applyMetricsSideEffect`'s `pending`/`skipped` cascade case MUST preserve a non-zero `Iterations` counter (keeping the entry with only `iterations`, clearing timing fields) instead of deleting the entry — so the rework choreography's `fail review` + `reset apply` no longer zeroes the cycle counter that `fab pr-meta` reports, making `SPEC-_pipeline.md`'s PR-meta rationale true. Entries with zero iterations are still deleted.

- **GIVEN** a change where review was activated once (`stage_metrics.review.iterations: 1`)
- **WHEN** `fab status fail <change> review` then `fab status reset <change> apply` run
- **THEN** `stage_metrics.review` retains `iterations: 1` (timing fields cleared)
- **AND** the next review activation increments it to 2 (logged as `re-entry`)

### CLI: Hook Sync Exit Code

#### R9: `fab hook sync` SHALL exit 0
`fab hook sync` MUST exit 0 even when it cannot resolve the fab root or write settings, surfacing the error on stderr instead of via exit code — conforming to `_cli-fab.md`'s "All hook subcommands exit 0" contract.

- **GIVEN** an environment where `.claude/settings.local.json` cannot be written
- **WHEN** `fab hook sync` runs
- **THEN** the error is printed to stderr and the exit code is 0

### Docs: Exit-Contract Doc Pass

#### R10: Doc rows MUST match the post-fix binary
After the Go fixes, the skill docs (canonical `src/kit/skills/`, never `.claude/skills/`) MUST be corrected where they were wrong about the binary, and left untouched where the Go pass makes them true:
- `_preamble.md` Common fab Commands: replace the invalid canonical form `fab change resolve --folder` with `fab resolve --folder`. Mirror: `docs/specs/skills/SPEC-_preamble.md`.
- `_cli-fab.md`: document archive partial failure (YAML + non-zero when the move succeeded but the backlog mark failed); update `fab batch archive` exit semantics (empty `--all` set exits 0; named-targets-all-invalid still exits 1); note the AllowedStates target-state enforcement on `advance`/`skip`; clarify `fab hook sync` surfaces errors on stderr while still exiting 0. (No `SPEC-_cli-fab.md` mirror exists.)
- `fab-archive.md`: add the `pointer: failed` row to the restore report table. Mirror: `docs/specs/skills/SPEC-fab-archive.md`.
- `fab-switch.md`: align the Next-line explanation with the fixed derivation if its wording references the old behavior. Mirror: `docs/specs/skills/SPEC-fab-switch.md` (only if the skill file is edited).
- `_preamble.md` score gate row, `_cli-fab.md` § fab score gate row, `_pipeline.md` Pre-flight 3, `_cli-fab.md` re-archive soft-skip rows: **no edit** — R1/R5 make them true.

- **GIVEN** the post-fix binary
- **WHEN** an agent copies the canonical form from `_preamble.md` § Common fab Commands
- **THEN** `fab resolve --folder` executes successfully (no `unknown flag` error)

### Non-Goals

- No fix for `CurrentStage`'s routing of a `review: failed` resting state (routes to `hydrate`) — pre-existing behavior, owned by the review-pr-failed-recovery batch (w7dp scope).
- No `/fab-archive` skill-flow changes beyond the restore report row — g8st/c5tr own the skill-side archive seam; k4ge owns the Go/doc exit-semantics side.
- No history-derivation of review cycles from `.history.jsonl` (intake Assumption #3 resolved toward in-Go preservation).
- No changes to `workflow.yaml` or scaffold config (Theme 5 scope).

### Design Decisions

1. **Gate fail exits via returned error**: print the YAML, then return an error so main's handler emits `ERROR: ...` on stderr and exits 1 — *Why*: stdout YAML stays parseable, cobra/test-friendly — *Rejected*: bare `os.Exit(1)` (skips error plumbing, less testable).
2. **Target-state validation inside `lookupTransition`**: one `validateTarget` helper applied to both the stage-override and default resolution paths — *Why*: schema (AllowedStates) is the single constraint source; closes all current and future forbidden combos at once — *Rejected*: deleting the `ready` rows from ship/review-pr transition tables (fixes only the known combos, leaves the schema unenforced).
3. **Uniform iterations preservation**: preserve `Iterations > 0` for every stage in the cascade, not just review — *Why*: matches the documented "incremented, not reset — tracks rework cycles" semantics; omit-empty encoding keeps the YAML minimal (`{iterations: N}`) — *Rejected*: review-only special case (asymmetric, surprises later consumers).
4. **Soft-skip via archive-scan fallback in `Archive()`** plus an exported `IsArchived` for batch pass-through — *Why*: keeps the cmd layer's existing `ErrAlreadyArchived` handling as the single soft-skip rendering — *Rejected*: duplicating archive detection in the cmd layer.

## Tasks

### Phase 1: Core Go Conformance (tests alongside each task)

- [x] T001 `src/go/fab/cmd/fab/score.go`: return a non-zero-exit error after printing gate YAML when `result.Gate == "fail"`; add cmd-level test (gate-fail fixture → error returned, stdout still has `gate: fail`; gate-pass → nil) in `src/go/fab/cmd/fab/score_test.go` (new) <!-- R1 -->
- [x] T002 `src/go/fab/internal/status/status.go`: add `validateTarget` AllowedStates check to `lookupTransition`; tests in `src/go/fab/internal/status/status_test.go` for `advance ship`/`advance review-pr`/`skip intake` rejected (file unchanged) and `start ship`/`skip apply`/`fail review` still allowed <!-- R2 -->
- [x] T003 `src/go/fab/internal/status/status.go` `applyMetricsSideEffect`: preserve `Iterations > 0` on `pending`/`skipped` (clear timing fields, keep entry); delete zero-iteration entries; tests covering fail review + reset apply cascade preserving `iterations` and re-activation incrementing to 2 <!-- R8 -->
- [x] T004 `src/go/fab/internal/change/change.go`: fix `defaultCommand` mapping (`hydrate → /fab-continue`, `ship → /git-pr`, `review-pr → /git-pr-review`) and `Switch`'s Next line (routing stage + its command; bare `/fab-archive` only when all stages done/skipped); table-driven tests in `change_test.go` <!-- R3 -->
- [x] T005 `src/go/fab/internal/archive/archive.go`: in `Archive()`, on `resolve.ToFolder` failure fall back to `resolveArchive` and return `ErrAlreadyArchived` when found; add exported `IsArchived(fabRoot, changeArg) bool`; tests: genuinely-archived re-archive (no source folder recreation) → `ErrAlreadyArchived`; bogus name → original error <!-- R5 -->
- [x] T006 `src/go/fab/internal/archive/archive.go` `Restore()`: set `pointer: failed` when `change.Switch` errors; test with `.fab-status.yaml` blocked by a directory <!-- R7 -->
- [x] T007 `src/go/fab/cmd/fab/archive.go`: change `changeArchiveCmd` to `cobra.ExactArgs(1)` and drop the help-on-zero-args branch; test asserting non-zero error on zero args <!-- R4 -->
- [x] T008 `src/go/fab/cmd/fab/batch_archive.go`: empty `--all`/no-args set → notice + zero footer + exit 0; in the resolve loop, pass archived names through to `archiveLoop` (via `archivePkg.IsArchived`) so they soft-skip; tests in `batch_archive_test.go` <!-- R6 -->
- [x] T009 `src/go/fab/cmd/fab/hook.go` `hookSyncCmd`: surface `FabRoot`/`Sync` errors on stderr and return nil (exit 0); test in `hook_test.go` <!-- R9 -->

### Phase 2: Doc Pass (after Go fixes)

- [x] T010 `src/kit/skills/_preamble.md`: replace canonical form `fab change resolve --folder` with `fab resolve --folder`; update `docs/specs/skills/SPEC-_preamble.md` mirror <!-- R10 -->
- [x] T011 `src/kit/skills/_cli-fab.md`: document archive partial-failure semantics (YAML + non-zero); fix `fab batch archive` exit rows (empty `--all` → 0; all-named-invalid → 1); note AllowedStates target-state enforcement on `advance`/`skip` rows; clarify `fab hook sync` error surfacing (stderr, exit 0); verify (no-edit) the score-gate and re-archive soft-skip rows <!-- R10 -->
- [x] T012 `src/kit/skills/fab-archive.md`: add `pointer: failed` row to the restore report table + output line; update `docs/specs/skills/SPEC-fab-archive.md` mirror <!-- R10 -->
- [x] T013 `src/kit/skills/fab-switch.md`: align Next-line wording with the fixed routing-stage derivation; update `docs/specs/skills/SPEC-fab-switch.md` mirror if edited <!-- R3 -->

### Phase 3: Verification

- [x] T014 Run scoped tests (`go test ./internal/status/... ./internal/change/... ./internal/archive/... ./internal/score/... ./cmd/fab/...` from `src/go/fab/`), then the full module test suite (`go test ./...`); fix any regressions <!-- R1 R2 R3 R4 R5 R6 R7 R8 R9 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab score --check-gate` returns a non-zero exit on `gate: fail` with the YAML intact on stdout; exit 0 on pass
- [x] A-002 R2: `lookupTransition` rejects `advance ship`, `advance review-pr`, and `skip intake` with a clean error; `.status.yaml` is not modified on rejection
- [x] A-003 R3: `Switch` prints `Next: {routing_stage} (via {command})` with the command that drives the routing stage, and bare `Next: /fab-archive` only when all stages are done/skipped
- [x] A-004 R4: `fab change archive` with no argument exits non-zero via `ExactArgs(1)`
- [x] A-005 R5: `fab change archive` on a genuinely archived change (source folder gone) prints `already archived: {change}` and exits 0
- [x] A-006 R6: `fab batch archive --all` on an empty set exits 0 with the notice and zero footer; a named archived change is counted skipped, not failed
- [x] A-007 R7: `fab change restore --switch` reports `pointer: failed` when activation fails
- [x] A-008 R8: `stage_metrics.review.iterations` survives `fail review` + `reset apply`; next activation increments it (re-entry)
- [x] A-009 R9: `fab hook sync` exits 0 on failure, error surfaced on stderr
- [x] A-010 R10: `_preamble.md` canonical column shows `fab resolve --folder`; `_cli-fab.md` documents archive partial failure, batch-archive exit semantics, transition target-state enforcement, and hook-sync error surfacing; `fab-archive.md` restore table has a `pointer: failed` row

### Behavioral Correctness

- [x] A-011 R2: all previously-valid transitions (`start`, `finish`, `reset`, `skip apply..review-pr`, `fail review|review-pr`) still succeed — only schema-forbidden targets are newly rejected
- [x] A-012 R8: a stage entry with zero iterations is still deleted on cascade (no empty `{}` entries linger)
- [x] A-013 R6: the explicit-args path where no target resolves anywhere still exits 1 (`No valid changes to archive.`)

### Scenario Coverage

- [x] A-014 R1: Go tests cover both gate-fail (error) and gate-pass (nil) command paths
- [x] A-015 R2 R8: Go tests cover the brick scenarios (advance ship/review-pr, skip intake) and the fail+reset iterations-preservation choreography
- [x] A-016 R3 R5 R6 R7: Go tests cover the post-review Next derivation, the genuinely-archived soft skip, the empty batch set, and the restore activation failure

### Edge Cases & Error Handling

- [x] A-017 R5: an argument matching no change anywhere still propagates the resolution error (non-zero); ambiguous archive matches do not soft-skip
- [x] A-018 R9: hook sync success output is unchanged (`Created`/`Updated`/`... hooks: OK` on stdout)

### Code Quality

- [x] A-019 Pattern consistency: new code follows naming and structural patterns of surrounding code (error message style, cobra command structure, test fixtures)
- [x] A-020 No unnecessary duplication: archive detection reuses `resolveArchive`; target validation reuses `AllowedStates`/`contains`
- [x] A-021 Readability over cleverness: transition validation extracted as a named helper with a comment explaining the brick scenario
- [x] A-022 No magic strings: commands/states reference existing constants and maps where available

### Documentation Accuracy

- [x] A-023 R10: every doc row touched matches the post-fix binary behavior exactly (verified against the implementation, not the intake)
- [x] A-024 R10: rows the intake marks "no edit" (`_preamble.md` gate row, `_cli-fab.md` gate + soft-skip rows, `_pipeline.md` Pre-flight 3) are byte-unchanged

### Cross References

- [x] A-025 R10: each touched skill file with an existing `docs/specs/skills/SPEC-*.md` mirror has the mirror updated in the same change (`_preamble`, `fab-archive`, `fab-switch` if edited)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `stageTransitions["review-pr"]["advance"]` (src/go/fab/internal/status/status.go:56) — with `validateTarget` enforcing AllowedStates, this override row can never succeed (`ready` is forbidden for review-pr); deleting it yields byte-identical behavior because the default `advance` row produces the same rejection message. Complementary cleanup to Design Decision 2 (which rejected row-deletion *instead of* schema enforcement, not alongside it).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Switch Next prints the routing stage (`CurrentStage`) with the command that drives it; bare `/fab-archive` only when every stage is done/skipped | Matches fab-switch.md's documented `Next: {routing_stage} (via {default_command})` and fab-status's "routing stage with the default command"; fixes the named off-by-one. Early-stage label shifts from `NextStage(routing)` to the routing stage itself — the doc-conformant reading | S:70 R:85 A:75 D:65 |
| 2 | Confident | Iterations preserved uniformly for all stages in the `pending`/`skipped` cascade (entry kept with only `iterations` when > 0), not review-only | Honors documented "incremented, not reset — tracks rework cycles" semantics; omit-empty YAML encoding keeps entries minimal; intake Assumption #3 (preserve-in-Go) honored — no flip | S:65 R:80 A:75 D:60 |
| 3 | Confident | Re-archive detection as a `resolveArchive` fallback inside `Archive()` + exported `IsArchived` for batch pass-through | Keeps cmd-layer `ErrAlreadyArchived` handling as the single soft-skip rendering; ambiguous matches fall through to the original error | S:75 R:85 A:80 D:75 |
| 4 | Confident | Only the `--all`/no-args empty batch set becomes exit 0; explicit-args all-invalid keeps exit 1 and gets documented | Finding a052 offers either/or; empty `--all` is the benign clean-repo no-op, named-targets-all-invalid is a genuine caller error | S:70 R:85 A:75 D:70 |
| 5 | Certain | No `SPEC-_cli-fab.md` mirror exists, so the constitution's mirror rule binds only `_preamble.md`, `fab-archive.md`, `fab-switch.md` edits | Verified `docs/specs/skills/` listing — `_cli-fab` has never had a mirror; creating one is out of scope | S:90 R:90 A:95 D:90 |
| 6 | Confident | Gate-fail exit implemented by returning an error after printing the YAML (stderr gets `ERROR: intake gate failed…` via main) | stdout YAML stays parseable for orchestrators; avoids bare `os.Exit` in RunE; consistent with main.go's SilenceErrors handler | S:75 R:90 A:85 D:80 |
| 7 | Confident | `fab hook sync` failures print to stderr with exit 0 (not silently swallowed like session hooks) | Sync is user/setup-invoked — silent failure would hide a broken hook install; exit 0 honors the documented contract that hook subcommands never block agent flows | S:70 R:85 A:80 D:75 |

7 assumptions (1 certain, 6 confident, 0 tentative).
