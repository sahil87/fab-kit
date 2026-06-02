# Plan: Split true-impact line count by implementation vs. tests

**Change**: 260602-7t5a-true-impact-test-split
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake "What Changes" §1–9, the Impact list, and the 16 Certain
     assumptions. Out of scope (binding): parsimony pass, deletion candidates, any
     other ogf2 bloat-restraint interventions. -->

### Config: `test_paths` key

#### R1: New top-level `test_paths` config field
The project config SHALL gain a top-level `test_paths` list of glob/pathspec strings identifying test files, mirroring the existing `source_paths` / `true_impact_exclude` style. `config.Config` MUST expose it as `TestPaths []string` with `yaml:"test_paths"`. The kit MUST NOT ship a default value (Portability, constitution V) — there is no universal test-file pattern.

- **GIVEN** `fab/project/config.yaml` with a top-level `test_paths: ["**/*_test.go"]`
- **WHEN** `config.Load` parses it
- **THEN** `cfg.TestPaths == ["**/*_test.go"]`
- **AND** when `test_paths` is absent/null/empty, `cfg.TestPaths` is empty and all downstream behavior collapses to today's single-number display

### Impact engine: test attribution pass

#### R2: Test-only shortstat pass within the scaffolding-excluded universe
`internal/impact` SHALL run an additional `git diff --shortstat <base>...<head>` pass whose pathspec includes BOTH the test-path includes AND the `true_impact_exclude` excludes, so test lines are counted AFTER `true_impact_exclude` is applied (no double-counting of test fixtures under excluded paths). `runShortstat` MUST be extended to accept include pathspecs (it already accepts excludes).

- **GIVEN** `testPaths = ["**/*_test.go"]` and `excludes = ["fab/", "docs/"]`
- **WHEN** the test pass runs
- **THEN** it invokes `git diff --shortstat <base>...<head> -- "**/*_test.go" :(exclude)fab/ :(exclude)docs/`
- **AND** the resulting triple is attributed to tests within the scaffolding-excluded universe

#### R3: `Result.Tests` and threaded `testPaths`
`impact.Result` SHALL gain a `Tests *Pair` field (alongside `Excluding *Pair`), nil when `testPaths` is empty. `Compute` MUST gain a trailing `testPaths []string` parameter; when empty it skips the test pass and leaves `Tests` nil. `ComputeForRepo` MUST read both `cfg.TrueImpactExclude` and `cfg.TestPaths` and pass them through.

- **GIVEN** `testPaths` is empty
- **WHEN** `Compute` runs
- **THEN** `res.Tests == nil` and no extra git pass is run
- **AND GIVEN** `testPaths` is non-empty, `res.Tests` holds the measured test triple
- **AND** the engine stores ONLY measured passes (raw, `Excluding`, `Tests`) — never a derived `impl`

### Status file: `tests` sub-block

#### R4: `TrueImpact.Tests` field + serialization
`statusfile.TrueImpact` SHALL gain `Tests *TrueImpactPair` (`yaml:"tests,omitempty"`). `encodeTrueImpact` MUST emit the `tests` mapping AFTER `excluding` and BEFORE `computed_at`. The decode path already covers it via the struct tag. The lazy-omit posture is preserved: `tests` is written only when non-nil.

- **GIVEN** a `TrueImpact` with a non-nil `Tests`
- **WHEN** `.status.yaml` is saved
- **THEN** the `true_impact` block contains a `tests:` mapping with `added`/`deleted`/`net`, positioned after `excluding` and before `computed_at`
- **AND** a round-trip Load→Save→Load preserves the `tests` triple
- **AND** when `Tests` is nil, no `tests:` key is emitted

#### R5: `WriteTrueImpact` populates `Tests`
`status.WriteTrueImpact` SHALL copy `res.Tests` (when non-nil) into a new `sf.TrueImpactPair` on the `TrueImpact` struct, mirroring the existing `Excluding` copy.

- **GIVEN** `impact.ComputeForRepo` returns a result with a non-nil `Tests`
- **WHEN** `WriteTrueImpact` runs at apply/hydrate finish
- **THEN** `statusFile.TrueImpact.Tests` holds the same triple
- **AND** when `res.Tests` is nil, `TrueImpact.Tests` stays nil

### CLI: `fab impact` YAML

#### R6: `renderYAML` emits the `tests` sub-block
`cmd/fab/impact.go`'s `renderYAML` SHALL emit a `tests:` sub-block (added/deleted/net) ONLY when `r.Tests != nil`, positioned after `excluding` so `/git-pr` can parse it via `yq`.

- **GIVEN** a `Result` with a non-nil `Tests`
- **WHEN** `fab impact <base> <head>` runs
- **THEN** stdout YAML includes a `tests:` block under `excluding`/`net`
- **AND** when `Tests` is nil, no `tests:` block appears

### Render-time impl/tests split

#### R7: Compact `--show-stats` split with per-component negative clamp
`change.impactColumn()` SHALL render the compact explicit-equation form `{impl_net}i+{tests_net}t={total_net}` (e.g. `102i+400t=502`) when `tests` is present, where `total_net = excluding.net` (else `net`) and `impl_net = max(0, total_net − tests_net)`. When `tests` is absent it MUST fall back to today's single `excluding.net` (else `net`, else `—`). The `impl` residual MUST be derived at render time only — NEVER stored. A one-line stderr warning MUST be emitted when the net clamp triggers (impl would be negative).

- **GIVEN** `true_impact` with `excluding.net=502` and `tests.net=400`
- **WHEN** `fab change list --show-stats` renders the impact column
- **THEN** the column shows `102i+400t=502`
- **AND GIVEN** `tests.net` over-counts `total_net` (e.g. total 100, tests 150)
- **THEN** `impl_net` clamps to `0` (column `0i+150t=100`), a negative is never rendered, and a one-line stderr warning is emitted
- **AND GIVEN** `tests` absent, the column shows the single net (`excluding.net` else `net` else `—`)

#### R8: Three-row PR-body rendering in `/git-pr`
The `/git-pr` PR-body Impact line SHALL render three rows (impl / tests / total) when `tests` is present: `impl` = per-component `max(0, total − tests)`, `tests` = the measured test triple, `total` = the scaffolding-excluded number (today's `excluding`; raw-with-fab/docs NOT shown). The `← excludes …` annotation MUST reflect the ACTUAL `true_impact_exclude` config values (not hardcoded). When any component clamp triggers, never render a negative. When `tests` is absent, collapse to a single `total` line (today's behavior).

- **GIVEN** `fab impact` YAML with `excluding` and `tests`
- **WHEN** `/git-pr` assembles the Impact line
- **THEN** it renders impl / tests / total rows with per-component clamped impl and a config-derived excludes annotation
- **AND GIVEN** no `tests` block, it renders the single `total` line as today

### Documentation

#### R9: CLI + skill docs updated (constitution-mandated)
`src/kit/skills/_cli-fab.md` MUST document the new `tests` sub-block in the `fab impact` output schema. `src/kit/skills/git-pr.md` change MUST be reflected in `docs/specs/skills/SPEC-git-pr.md`. The scaffold config (`src/kit/scaffold/fab/project/config.yaml`) MUST carry a commented-out `test_paths` placeholder (no default patterns). This repo's own `fab/project/config.yaml` MUST set `test_paths: ["**/*_test.go"]` to dogfood the feature.

- **GIVEN** the CLI/skill changes land
- **WHEN** the docs are reviewed
- **THEN** `_cli-fab.md` shows the `tests` block, `SPEC-git-pr.md` reflects the three-row Impact rendering, the scaffold has the commented placeholder, and this repo's config sets `test_paths`

### Edge cases

#### R10: Graceful collapse on empty inputs
Behavior MUST stay graceful per the existing lazy-omit posture. Empty/absent/null `test_paths` → no `tests` sub-block, single-line rendering. Empty `true_impact_exclude` → `total` degenerates to raw `net`; tests can still be split out (computed within the raw universe, since there is nothing to exclude).

- **GIVEN** `test_paths` empty
- **THEN** `Tests` is nil, `.status.yaml` has no `tests`, rendering is a single line
- **AND GIVEN** `true_impact_exclude` empty but `test_paths` set
- **THEN** the test pass runs with only the include pathspec (no `:(exclude)` args), `Excluding` is nil, and `total` falls back to raw `net`

### Non-Goals

- The parsimony pass, deletion candidates, or any other ogf2 bloat-restraint interventions (intake "Out of scope" — binding).
- A migration: the new `tests` field is optional + lazy; existing `.status.yaml` files and configs remain valid. No migration file is shipped.
- Storing an `impl` field anywhere (engine, `.status.yaml`, `fab impact` YAML stay pure-measurement).

### Design Decisions

1. **Attribution, not exclusion**: add `test_paths` to *attribute* the scaffolding-excluded universe to tests vs. impl — *Why*: tests are first-class deliverables, not scaffolding noise; conflating the two axes loses the split — *Rejected*: adding test patterns to `true_impact_exclude`.
2. **Render-time residual**: `impl = total − tests` derived at render sites only — *Why*: keeps the engine pure-measurement so no derived field drifts between the two diff passes — *Rejected*: storing an `impl` field in the engine/`.status.yaml`/YAML.
3. **Per-component clamp**: clamp added/deleted/net independently to `max(0, total.X − tests.X)` — *Why*: the three-row display shows separate `+X / −Y` components, each must be non-negative on its own — *Rejected*: net-only clamp.
4. **Compact format `102i+400t=502`**: explicit-equation form — *Why*: all three values visible — *Rejected*: `502 (102+400)` and `+102i/+400t`.

## Tasks

### Phase 1: Setup

- [x] T001 Add `TestPaths []string` (`yaml:"test_paths"`) to `Config` in `src/go/fab/internal/config/config.go` <!-- R1 -->

### Phase 2: Core Implementation

- [x] T002 Extend `runShortstat` in `src/go/fab/internal/impact/impact.go` to accept include pathspecs; add `Tests *Pair` to `Result`; add the test-only pass; add trailing `testPaths` param to `Compute`; read `cfg.TestPaths` in `ComputeForRepo` <!-- R2 R3 R10 --> <!-- rework cycle 1: raw pass mistakenly passed `excludes` (collapsed raw==excluding, corrupting the base measurement per R3/intake §2); reverted raw pass to no excludes -->

- [x] T003 Add `Tests *TrueImpactPair` (`yaml:"tests,omitempty"`) to `TrueImpact` and emit it after `excluding`/before `computed_at` in `encodeTrueImpact` (`src/go/fab/internal/statusfile/statusfile.go`) <!-- R4 -->
- [x] T004 Copy `res.Tests` into a `TrueImpactPair` on the `TrueImpact` struct in `WriteTrueImpact` (`src/go/fab/internal/status/true_impact.go`) <!-- R5 -->
- [x] T005 Emit the `tests` sub-block in `renderYAML` when `r.Tests != nil` (`src/go/fab/cmd/fab/impact.go`) <!-- R6 -->

### Phase 3: Integration & Edge Cases

- [x] T006 Implement the compact split `{impl}i+{tests}t={total}` with per-component net clamp + one-line stderr warning in `impactColumn()` (`src/go/fab/internal/change/change.go`) <!-- R7 R10 -->

### Phase 4: Documentation & Tests

- [x] T007 [P] Three-row Impact rendering (impl/tests/total) with config-derived excludes annotation + per-component clamp in `src/kit/skills/git-pr.md` <!-- R8 -->
- [x] T008 [P] Add commented-out `test_paths` placeholder to `src/kit/scaffold/fab/project/config.yaml`; set `test_paths: ["**/*_test.go"]` in this repo's `fab/project/config.yaml` <!-- R9 R1 -->
- [x] T009 [P] Document the `tests` sub-block in `src/kit/skills/_cli-fab.md` `fab impact` output schema <!-- R9 -->
- [x] T010 [P] Update `docs/specs/skills/SPEC-git-pr.md` for the three-row Impact rendering <!-- R9 R8 -->
- [x] T011 Add impact engine tests in `src/go/fab/internal/impact/impact_test.go`: empty `test_paths` → `Tests` nil; empty `true_impact_exclude` → tests within raw universe; include-pathspec behavior <!-- R2 R3 R10 --> <!-- rework cycle 1: strengthened TestCompute_ExcludesEmitsExcluding to assert raw.added > excluding.added (was non-discriminating `>=`), now guards the raw-pass regression -->

- [x] T012 Add statusfile tests in `src/go/fab/internal/statusfile/statusfile_test.go`: encode/decode round-trip of `tests` (impactColumn lives in `internal/change`, covered by T013) <!-- R4 R7 -->
- [x] T013 Add `impactColumn` compact-split + clamp coverage in `src/go/fab/internal/change/change_test.go` <!-- R7 -->

## Execution Order

- T001 before T002 (engine reads `cfg.TestPaths`).
- T002 before T003/T004/T005 (downstream consume `Result.Tests`).
- T003 before T006 (impactColumn reads `TrueImpact.Tests`).
- T007–T010 are independent docs/config (`[P]`), may run anytime after their Go counterparts are settled.
- T011–T013 follow their respective implementation tasks.

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `config.Config` has `TestPaths []string` reading `test_paths`; absent/null/empty yields an empty slice.
- [ ] A-002 R2: The impact engine runs a test-only `git diff --shortstat` pass whose pathspec combines test includes with the `true_impact_exclude` excludes.
- [ ] A-003 R3: `Result.Tests` is nil when `testPaths` is empty and holds the measured triple otherwise; `Compute` takes `testPaths`; `ComputeForRepo` reads `cfg.TestPaths`.
- [ ] A-004 R4: `TrueImpact.Tests *TrueImpactPair` (`yaml:"tests,omitempty"`) serializes after `excluding` and before `computed_at`.
- [ ] A-005 R5: `WriteTrueImpact` copies `res.Tests` into `TrueImpact.Tests`.
- [ ] A-006 R6: `renderYAML` emits the `tests` sub-block only when present.
- [ ] A-007 R7: `impactColumn` renders `{impl}i+{tests}t={total}` when `tests` present, single net otherwise.
- [ ] A-008 R8: `/git-pr` renders three Impact rows (impl/tests/total) when `tests` present, single `total` row otherwise.
- [ ] A-009 R9: `_cli-fab.md`, `SPEC-git-pr.md`, scaffold config, and this repo's config are all updated per spec.

### Behavioral Correctness

- [ ] A-010 R3: The engine stores only measured passes — no `impl` field exists in `Result`, `.status.yaml`, or `fab impact` YAML.
- [ ] A-011 R7: `impl_net = max(0, total_net − tests_net)` is computed at render time in `impactColumn`; `total_net = excluding.net` else `net`.
- [ ] A-012 R8: The `/git-pr` excludes annotation is derived from the actual `true_impact_exclude` config values, never hardcoded.
- [ ] A-013 R4: A Load→Save→Load round-trip preserves the `tests` triple and its position.

### Scenario Coverage

- [ ] A-014 R2: A test verifies the include-pathspec pass counts test lines within the scaffolding-excluded universe.
- [ ] A-015 R3: A test verifies empty `test_paths` → `Tests` nil (no extra pass).
- [ ] A-016 R10: A test verifies empty `true_impact_exclude` with non-empty `test_paths` → tests computed within the raw universe (no `:(exclude)` args), `Excluding` nil.
- [ ] A-017 R4: A test verifies the encode/decode round-trip of the `tests` sub-block.
- [ ] A-018 R7: A test verifies `impactColumn` compact form `{impl}i+{tests}t={total}`.

### Edge Cases & Error Handling

- [ ] A-019 R7: A test verifies the per-component negative clamp (tests over-counts total → impl clamps to 0, no negative rendered).
- [ ] A-020 R7: The net clamp emits a one-line stderr warning when triggered.
- [ ] A-021 R10: `test_paths` empty/absent/null collapses to single-line rendering with no `tests` sub-block.

### Code Quality

- [ ] A-022 Pattern consistency: New code follows naming and structural patterns of surrounding code (the `Excluding`/`TrueImpactPair` patterns are followed exactly).
- [ ] A-023 No unnecessary duplication: `runShortstat` is reused/extended rather than duplicated; the existing `formatNet`/encode helpers are reused.
- [ ] A-024 No god functions / magic numbers: render-split helpers stay focused; `impl = max(0, total − tests)` uses named local values, not magic literals.

### Documentation Accuracy

- [ ] A-025 R9: `_cli-fab.md` and `SPEC-git-pr.md` accurately describe the shipped `tests` sub-block and three-row rendering.

### Cross References

- [ ] A-026 R9: Affected memory entries (`fab-workflow/schemas`, `fab-workflow/configuration`) are noted in the intake for the hydrate stage; no dangling references introduced.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

<!-- The intake is fully clarified (16 Certain, score 5.0). Apply needed no new
     under-specified decisions. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All design decisions inherited verbatim from the intake's 16 Certain assumptions; no new SRAD decisions required at apply | Intake fully clarified (score 5.0); every What-Changes section carries exact values, code blocks, and behavior. | S:98 R:80 A:90 D:95 |
| 2 | Confident | Test-path includes are applied as `:(glob)<pattern>` magic pathspecs (not plain pathspecs) so `**` matches across directory boundaries | The intake documents the example `**/*_test.go` and calls these "test globs" (§3). Under plain git pathspec rules `**/*_test.go` is literal and silently misses root-level test files (`*` matches `/`), under-counting tests — verified empirically. `:(glob)` makes the documented pattern behave as a `.gitignore`-style glob, which is the only interpretation consistent with the intake's stated example. Mechanical/low-blast-radius and confirmed by test. | S:80 R:78 A:88 D:82 |

2 assumptions (1 certain, 1 confident, 0 tentative).
