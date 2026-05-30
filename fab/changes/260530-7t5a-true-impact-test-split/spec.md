# Spec: Split true-impact line count by implementation vs. tests

**Change**: 260530-7t5a-true-impact-test-split
**Created**: 2026-05-30
**Affected memory**: `docs/memory/fab-workflow/schemas.md`, `docs/memory/fab-workflow/configuration.md`

## Non-Goals

- The parsimony pass, deletion candidates, or any other `ogf2` bloat-restraint intervention — this change is purely the impl/test attribution split on top of the already-shipped `true_impact` infrastructure.
- Adding test patterns to `true_impact_exclude` — explicitly rejected. Exclusion would conflate two orthogonal axes (scaffolding-noise vs. test-vs-impl) and destroy the test-coverage signal. Tests are attributed, not erased.
- Shipping any default `test_paths` value in the kit — there is no universal test-file pattern across languages (Portability, constitution principle V).
- A migration — the new `tests` field is optional and lazy; existing `.status.yaml` files and existing configs without `test_paths` remain valid and collapse to today's behavior.
- Storing a derived `impl` field anywhere (engine `Result`, `.status.yaml`, or `fab impact` YAML) — `impl` is a render-time residual only.

## Configuration: `test_paths` config key

### Requirement: Top-level `test_paths` config field
`config.Config` SHALL gain a `TestPaths []string` field reading the top-level `test_paths` key (`yaml:"test_paths"`) from `fab/project/config.yaml`. The field holds project-defined glob/pathspec patterns identifying test files, mirroring the existing `source_paths` / `true_impact_exclude` top-level-list style. The kit SHALL NOT ship any default `test_paths` value (Portability — no universal test-file pattern across languages).

#### Scenario: Config with test_paths present
- **GIVEN** `fab/project/config.yaml` contains `test_paths: ["**/*_test.go"]`
- **WHEN** `config.Load` parses the file
- **THEN** `cfg.TestPaths` equals `["**/*_test.go"]`

#### Scenario: Config without test_paths
- **GIVEN** `fab/project/config.yaml` has no `test_paths` key (or it is `null`/`[]`)
- **WHEN** `config.Load` parses the file
- **THEN** `cfg.TestPaths` is empty/nil and the binary proceeds without error

### Requirement: Graceful collapse when `test_paths` is empty
WHEN `test_paths` is absent, `null`, or empty, all behavior SHALL collapse to today's single-number `true_impact` display — no `tests` sub-block is written, and consumers render a single `total` line. This matches the existing lazy-omit posture of the `excluding` sub-block.

#### Scenario: Empty test_paths collapses
- **GIVEN** a project whose config omits `test_paths`
- **WHEN** the `true_impact` block is computed and rendered
- **THEN** no `tests` sub-block is written to `.status.yaml`, the `fab impact` YAML omits `tests`, and the PR body / `--show-stats` column render exactly as they do today (no impl/tests split)

### Requirement: Scaffold config placeholder
`src/kit/scaffold/fab/project/config.yaml` SHALL include a commented-out `test_paths` placeholder (no active default patterns) documenting the knob's purpose (attribution, not exclusion), its project-defined / language-specific nature, the graceful-collapse behavior, and example patterns (e.g., `**/*_test.go`, `test_*.py`, `**/*.spec.ts`).

#### Scenario: Scaffold exposes the knob
- **GIVEN** a freshly scaffolded project from `src/kit/scaffold/fab/project/config.yaml`
- **WHEN** a user reads the config
- **THEN** they find a commented `# test_paths:` block with explanatory text and examples, and the active config behaves as if `test_paths` is empty

## Engine: canonical impact math (`internal/impact`)

### Requirement: `Result` gains a `Tests` field
The `impact.Result` struct SHALL gain a `Tests *Pair` field alongside the existing `Excluding *Pair`. `Tests` SHALL be nil when `test_paths` is empty (no test pass run). The struct SHALL NOT store any derived `impl` field — the engine remains pure measurement.

#### Scenario: Tests nil without test paths
- **GIVEN** `Compute` is called with an empty `testPaths` argument
- **WHEN** it returns
- **THEN** `Result.Tests` is nil and no extra `git diff` pass was run

### Requirement: Test-only shortstat pass within the scaffolding-excluded universe
`Compute(repoDir, base, head, excludes, testPaths)` SHALL accept a new `testPaths` parameter. WHEN `testPaths` is non-empty, it SHALL run an additional `git diff --shortstat <base>...<head>` pass whose pathspec includes BOTH the test-path includes AND the `true_impact_exclude` excludes — so the test count lives strictly inside the scaffolding-excluded universe (a test fixture under an excluded path is not double-counted). The result populates `Result.Tests`. `runShortstat` SHALL be extended to accept include pathspecs in addition to the exclude pathspecs it already supports. `ComputeForRepo` SHALL read both `cfg.TrueImpactExclude` and `cfg.TestPaths` and thread them through.

#### Scenario: Test pass counts only test files inside the excluded universe
- **GIVEN** a diff touching `src/foo.go` (+140/−38), `src/foo_test.go` (+400/−0), and `docs/x.md` (+72/−0), with `true_impact_exclude: [fab/, docs/]` and `test_paths: ["**/*_test.go"]`
- **WHEN** `Compute` runs
- **THEN** `Result.Added/Deleted` are the raw 612/38, `Result.Excluding` is 540/38, and `Result.Tests` is 400/0 (the `docs/` lines never enter any of the three passes that matter for tests/total)

#### Scenario: Test fixture under an excluded path is not double-counted
- **GIVEN** `test_paths: ["**/*_test.*"]` matches a fixture under `docs/` and `true_impact_exclude: [docs/]`
- **WHEN** the test pass runs with both the include and exclude pathspecs
- **THEN** the excluded fixture contributes 0 to `Result.Tests` (the exclude wins)

### Requirement: Engine stores only measured passes
The engine SHALL store only the three measured passes — raw (`Added`/`Deleted`/`Net`), `Excluding`, and `Tests`. It SHALL NOT compute, clamp, or store the `impl` residual. The residual and its clamp are the consumers' responsibility (render time only).

#### Scenario: No impl in engine output
- **GIVEN** any `Compute` invocation
- **WHEN** the `Result` is inspected
- **THEN** there is no `impl`/residual field — only raw, `Excluding`, and `Tests`

## Status file: `tests` sub-block (`internal/statusfile`, `internal/status`)

### Requirement: `TrueImpact` gains a `Tests` sub-block
`statusfile.TrueImpact` SHALL gain a `Tests *TrueImpactPair` field tagged `yaml:"tests,omitempty"`, following the existing `Excluding` pattern exactly. `encodeTrueImpact` SHALL emit the `tests` mapping (with `added`/`deleted`/`net`) positioned AFTER `excluding` and BEFORE `computed_at`. The decode path (`val.Decode(ti)` in `Load`) covers it via the struct tag. When `Tests` is nil, the sub-block SHALL be omitted entirely.

#### Scenario: Encode emits tests after excluding
- **GIVEN** a `TrueImpact` with non-nil `Excluding` and non-nil `Tests`
- **WHEN** `encodeTrueImpact` runs
- **THEN** the emitted mapping order is `added, deleted, net, excluding, tests, computed_at, computed_at_stage`

#### Scenario: Round-trip decode of tests
- **GIVEN** a `.status.yaml` whose `true_impact` block contains a `tests:` mapping
- **WHEN** `Load` parses it and `Save` re-serializes
- **THEN** the `tests` sub-block survives the round-trip with `added`/`deleted`/`net` intact

#### Scenario: Tests omitted when nil
- **GIVEN** a `TrueImpact` with nil `Tests`
- **WHEN** it is encoded
- **THEN** no `tests:` key appears in the output (omitempty)

### Requirement: `WriteTrueImpact` copies the test pass
`WriteTrueImpact` in `internal/status/true_impact.go` SHALL copy `res.Tests` into a new `sf.TrueImpactPair` on the `TrueImpact` struct when `res.Tests` is non-nil, mirroring the existing `res.Excluding` handling. The best-effort stderr posture (warn-and-continue, never fail the stage transition) SHALL be preserved unchanged.

#### Scenario: Apply-finish writes tests
- **GIVEN** a project with `test_paths` set and a resolvable merge-base
- **WHEN** `WriteTrueImpact(..., "apply")` runs
- **THEN** `.status.yaml` `true_impact.tests` is populated from `res.Tests`

#### Scenario: Best-effort on failure preserved
- **GIVEN** the merge-base cannot be resolved
- **WHEN** `WriteTrueImpact` runs
- **THEN** it emits the existing one-line stderr warning and returns nil — the stage transition is unaffected (no new failure mode)

## CLI: `fab impact` YAML surface (`cmd/fab/impact.go`)

### Requirement: `renderYAML` emits the `tests` sub-block
`renderYAML` SHALL emit a `tests:` sub-block (with `added`/`deleted`/`net`) when `r.Tests` is non-nil, placed after the `excluding` sub-block and before `computed_at`, matching the `.status.yaml` field order so `/git-pr` can parse it via `yq`. When `r.Tests` is nil, no `tests:` block is emitted.

#### Scenario: CLI emits tests when present
- **GIVEN** `fab impact <base> HEAD` run in a project with `test_paths` set
- **WHEN** the test pass yields non-nil `Tests`
- **THEN** stdout includes a `tests:` mapping after `excluding:` and before `computed_at:`

#### Scenario: CLI omits tests when absent
- **GIVEN** `fab impact <base> HEAD` run in a project with no `test_paths`
- **WHEN** the result has nil `Tests`
- **THEN** stdout omits the `tests:` block and is byte-compatible with today's output

## Rendering: PR body three-row impact (`src/kit/skills/git-pr.md`)

### Requirement: Three-row impact block when `tests` is present
WHEN the parsed `fab impact` YAML contains a `tests` sub-block, the `/git-pr` PR body SHALL render the true-impact breakdown as a three-row block:
```
True impact:
  impl:  +140 / −38  (net +102)
  tests: +400 / −0   (net +400)
  total: +540 / −38  (net +502)   ← excludes fab/, docs/
```
The `total` row SHALL be the scaffolding-excluded number (today's `excluding` block, falling back to raw when `true_impact_exclude` is empty). The raw-with-`fab/`/`docs/`-included number SHALL NOT be displayed in the PR body. The `(excludes …)` annotation SHALL reflect the actual `true_impact_exclude` config values verbatim, never hardcoded. The Unicode minus `−` (U+2212) SHALL be used, consistent with the existing line.

#### Scenario: Three-row render with excludes
- **GIVEN** parsed YAML with `excluding` = 540/38, `tests` = 400/0, and `true_impact_exclude: [fab/, docs/]`
- **WHEN** the PR body is assembled
- **THEN** it shows `impl: +140 / −38 (net +102)`, `tests: +400 / −0 (net +400)`, `total: +540 / −38 (net +502)` with an `excludes fab/, docs/` annotation derived from config

### Requirement: `impl` is the render-time residual `total − tests`
The `impl` row SHALL be computed at render time as `total − tests` (NOT read from any stored field). The relationship displayed SHALL be `impl + tests = total`.

#### Scenario: impl derived, not stored
- **GIVEN** `total` = 540/38 and `tests` = 400/0
- **WHEN** the PR body renders
- **THEN** `impl` = 140/38 (net 102), computed from the two measured passes, with no `impl` field read from YAML

### Requirement: Per-component negative-impl clamp with stderr warning
The render-time residual SHALL be clamped per component: `impl.added = max(0, total.added − tests.added)`, `impl.deleted = max(0, total.deleted − tests.deleted)`, `impl.net = max(0, total.net − tests.net)` — each of added/deleted/net clamped independently (the three-row display shows separate `+X / −Y` components, each of which must be non-negative on its own). WHEN any component clamp triggers, a one-line stderr warning SHALL be emitted (consistent with the best-effort stderr posture). A negative `impl` line or component SHALL NEVER be rendered.

#### Scenario: Over-counting tests clamps impl to zero per component
- **GIVEN** `total` = 400/0 and `tests` = 450/0 (a `test_paths` glob overlapped a path also counted in tests, over-counting relative to total)
- **WHEN** the residual is computed
- **THEN** `impl.added` = `max(0, 400 − 450)` = 0, `impl.net` = `max(0, 400 − 450)` = 0, a one-line stderr warning is emitted, and no negative value is rendered

#### Scenario: Single total line when tests absent
- **GIVEN** parsed YAML with `excluding` present but no `tests` sub-block (empty `test_paths`)
- **WHEN** the PR body renders
- **THEN** it renders today's single inline `**Impact**` line (no three-row block, no impl/tests split) and the existing omit rules (no merge-base, `+0/−0`, missing `excluding`, no fab context) still apply unchanged

## Rendering: `--show-stats` compact column (`impactColumn` in `internal/change`)

### Requirement: Compact explicit-equation split column
WHEN the loaded `TrueImpact` has a non-nil `Tests`, `impactColumn` SHALL render the explicit-equation form `{impl_net}i+{tests_net}t={total_net}` (e.g., `102i+400t=502`), where `total_net` is `Excluding.Net` (else raw `Net`), `tests_net` is `Tests.Net`, and `impl_net` is the per-net clamped residual `max(0, total_net − tests_net)`. WHEN `Tests` is nil, it SHALL fall back to today's behavior (`Excluding.Net`, else `Net`, else `—`).

#### Scenario: Compact split rendered
- **GIVEN** a `.status.yaml` with `excluding.net` = 502 and `tests.net` = 400
- **WHEN** `fab change list --show-stats` renders the column
- **THEN** the column shows `102i+400t=502`

#### Scenario: Compact net-clamp guard
- **GIVEN** `excluding.net` = 400 and `tests.net` = 450
- **WHEN** the compact column renders
- **THEN** `impl_net` clamps to 0, rendering `0i+450t=400` — no negative value

#### Scenario: Bare-net fallback without tests
- **GIVEN** a `.status.yaml` whose `true_impact` has no `tests` sub-block
- **WHEN** the column renders
- **THEN** it shows the single net value as today (`+502`, else raw net, else `—`)

## Documentation

### Requirement: Constitution-mandated documentation updates
This change SHALL update: `src/kit/skills/_cli-fab.md` (the `fab impact` output schema gains the `tests` sub-block; CLI change is constitution-mandated), `docs/specs/skills/SPEC-git-pr.md` (the skill change to the Impact-line rendering is constitution-mandated), and the two affected memory files (`schemas.md` for the `tests` sub-block; `configuration.md` for the `test_paths` config field). Canonical edits SHALL go under `src/` (never the deployed `.claude/skills/` copies).

#### Scenario: Docs reflect the new surface
- **GIVEN** the implementation is complete
- **WHEN** docs are reviewed
- **THEN** `_cli-fab.md` documents the `tests` sub-block in `fab impact` output, `SPEC-git-pr.md` describes the three-row Impact rendering, and the memory files document the `tests` schema and `test_paths` config

### Requirement: Test coverage
Unit tests SHALL cover, in `internal/impact/`: the test-only pass (counts test lines within the excluded universe), the empty-`test_paths` edge (no test pass, `Tests` nil), and the empty-`true_impact_exclude` edge (total degenerates to raw, tests still splittable within the raw universe). The per-component clamp is exercised at the render sites; impact-engine tests SHALL assert the engine itself does NOT clamp (pure measurement). In `internal/statusfile`: encode/decode (round-trip) coverage for the `tests` sub-block, including the omitempty path. Test strategy is `test-alongside` per `code-quality.md`.

#### Scenario: Edge tests pass
- **GIVEN** the test suite
- **WHEN** `go test ./internal/impact/... ./internal/statusfile/...` runs
- **THEN** the test pass, empty-`test_paths`, empty-`true_impact_exclude`, and `tests` encode/decode cases all pass

## Edge Cases

### Requirement: Empty `true_impact_exclude` degenerates total to raw
WHEN `true_impact_exclude` is empty (nothing to strip), the `excluding` sub-block is absent (as today), so the `total` row SHALL fall back to the raw `net`. Tests SHALL still be splittable when `test_paths` is set, computed within the raw universe (the test pass simply has no exclude pathspecs to add).

#### Scenario: Tests split within raw universe
- **GIVEN** `true_impact_exclude` empty and `test_paths: ["**/*_test.go"]`, with a diff of `src/foo.go` (+140/−0) and `src/foo_test.go` (+400/−0)
- **WHEN** the impact is computed and rendered
- **THEN** `total` = raw 540/0, `tests` = 400/0, `impl` = 140/0 (net 102i+400t=540 compact); no `excluding` sub-block is written

## Assumptions

<!-- SCORING SOURCE: fab score reads only this table. All 16 intake decisions are locked
     (Certain) and carried forward verbatim. No new spec-level micro-decisions surfaced
     that warrant a lower grade — the grounding pass confirmed every intake assumption
     against the real code. Zero [NEEDS CLARIFICATION] markers. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Goal is attribution, not exclusion — do NOT add test patterns to `true_impact_exclude` | Confirmed from intake #1. Explicit locked decision; conflating axes loses the split. | S:98 R:80 A:90 D:95 |
| 2 | Certain | New top-level `test_paths` config key (project-defined glob/pathspec list, `yaml:"test_paths"`), mirroring `source_paths`/`true_impact_exclude` | Confirmed from intake #2. Portability (principle V) forbids a universal kit-shipped pattern. | S:95 R:75 A:90 D:90 |
| 3 | Certain | Kit ships NO default `test_paths`; absent/null/empty collapses to today's single-number display | Confirmed from intake #3 + constitution V; matches existing `excluding` lazy-omit posture. | S:95 R:85 A:92 D:95 |
| 4 | Certain | New `tests` sub-block via an extra `git diff --shortstat` pass with test-only pathspec WITHIN the scaffolding-excluded universe (test includes AND `true_impact_exclude` excludes both applied) | Confirmed from intake #4. Grounded against `runShortstat` (already supports excludes; extend for includes). Counting after excludes prevents double-counting. | S:95 R:70 A:88 D:90 |
| 5 | Certain | Three-row rendering (impl/tests/total) in `/git-pr`; `total` = scaffolding-excluded number (repurposes `excluding`), raw-with-fab/docs NOT shown in PR body | Confirmed from intake #5. NOTE: replaces today's single inline `**Impact**: …code…·…total` line (grounding surprise, see notes) — intentional repurpose. | S:95 R:78 A:85 D:88 |
| 6 | Certain | `impl + tests = total` where `impl = total − tests` (residual, not measured); `(excludes …)` annotation reflects actual config values, never hardcoded | Confirmed from intake #6. Storing impl separately risks cross-pass drift; annotation-from-config matches existing asvz behavior. | S:95 R:80 A:88 D:90 |
| 7 | Certain | Correctness guard: `impl = max(0, total − tests)`, one-line stderr warning on clamp, never render negative | Confirmed from intake #7. Cross-pass arithmetic can go negative on glob/exclude overlap; stderr posture matches `WriteTrueImpact`. | S:96 R:82 A:90 D:92 |
| 8 | Certain | Edge cases: empty `test_paths` → single `total` line; empty `true_impact_exclude` → `total` degenerates to raw; both graceful | Confirmed from intake #8; consistent with existing lazy-omit semantics. | S:95 R:85 A:90 D:92 |
| 9 | Certain | Canonical math extension in `internal/impact/impact.go` (`Result.Tests *Pair`, thread `testPaths` through `Compute`/`ComputeForRepo`, extend `runShortstat` for includes) | Confirmed from intake #9. Grounded: `Compute`/`ComputeForRepo`/`runShortstat`/`Result`/`Pair` all match the intake's description exactly. | S:96 R:65 A:88 D:82 |
| 10 | Certain | `statusfile.TrueImpact` gains `Tests *TrueImpactPair` (`yaml:"tests,omitempty"`); `encodeTrueImpact` emits it after `excluding`, before `computed_at`; `impactColumn()` updated for compact split | Confirmed from intake #10. Grounded: `TrueImpact`/`TrueImpactPair`/`encodeTrueImpact`/`impactColumn` match the `Excluding` pattern exactly; decode is tag-driven. | S:96 R:68 A:90 D:80 |
| 11 | Certain | Scaffold config gets a commented-out `test_paths` placeholder (no default patterns) | Confirmed from intake #11. Reversible doc-only change. | S:95 R:88 A:80 D:78 |
| 12 | Certain | No migration needed (new field optional + lazy; existing files/configs remain valid) | Confirmed from intake #12. Verified against the live `TrueImpact` struct and `Load` tolerance — `tests,omitempty` + tag-driven decode mean old files parse unchanged. | S:96 R:72 A:85 D:82 |
| 13 | Certain | Documentation: update `_cli-fab.md` (CLI), `SPEC-git-pr.md` (skill), memory `schemas` + `configuration`; tests in `internal/impact/` (clamp + edges) and `internal/statusfile` (encode/decode) | Confirmed from intake #13. Constitution-mandated (CLI + skill change rules); direct requirement. | S:95 R:75 A:90 D:85 |
| 14 | Certain | Compact `--show-stats` column format is `102i+400t=502` (explicit equation); bare net when no `test_paths` | Confirmed from intake #14 (clarified — user chose explicit-equation form). Grounded against `impactColumn`/`formatNet`. | S:95 R:88 A:65 D:45 |
| 15 | Certain | Clamp applies per-component (added/deleted/net independently): `impl.X = max(0, total.X − tests.X)` | Confirmed from intake #15 (clarified — per-component over net-only). Three-row display shows separate `+X / −Y`, so each component must be non-negative. | S:95 R:80 A:70 D:48 |
| 16 | Certain | Residual/clamp computed at render time only — `.status.yaml`, engine `Result`, and `fab impact` YAML store only measured passes (raw, excluding, tests); consumers (`/git-pr` body + `impactColumn`) derive and clamp | Confirmed from intake #16 (clarified — render-time only). Engine stays pure-measurement; clamp logic lives at both render sites. | S:95 R:62 A:68 D:50 |

16 assumptions (16 certain, 0 confident, 0 tentative, 0 unresolved).
