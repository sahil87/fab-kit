# Intake: Split true-impact line count by implementation vs. tests

**Change**: 260602-7t5a-true-impact-test-split
**Created**: 2026-06-02
**Status**: Draft

## Origin

This change reimplements the design captured in PR #361 (`260530-7t5a-true-impact-test-split`), authored against the legacy 7-stage pipeline. The design decisions were settled in a completed design discussion and fully clarified; this intake records them as Certain assumptions so apply does not re-open them. References to the former `spec` stage have been scrubbed to fit the current 6-stage pipeline (`intake → apply → review → hydrate → ship → review-pr`) — requirements are co-generated into `plan.md`'s `## Requirements` section at apply entry, and the final schema verification noted below is performed at apply rather than at a separate spec stage.

> Split the true-impact line count by implementation vs. tests. The `true_impact` block in `.status.yaml` (and the impact line rendered in `/git-pr` PR bodies) reports a single net line-count, conflating implementation code with test code. In a project where tests are first-class deliverables, reviewers want to *see the split* ("200 impl / 400 test") — not erase tests, and not have them vanish into the same noise bucket as `fab/` and `docs/`.

Type: **refactor** (extends the already-shipped `true_impact` infrastructure with a new attribution axis; restructures the impact engine and rendering rather than adding a net-new user-facing feature).

## Why

### The problem

The `true_impact` block (shipped in change `260507-ogf2-restrain-ai-code-bloat`, with the `true_impact_exclude` config from sister change `260507-asvz-git-pr-true-impact-line-count`) reports a single net line-count. It has an optional `excluding` sub-block that strips paths listed in `true_impact_exclude` (default scaffold `[fab/, docs/]`) — workflow/doc scaffolding noise. The metric's purpose is to restrain AI code bloat by surfacing *real* code delta.

But it conflates **implementation code** with **test code** into one number. The constitution makes tests first-class:
- Principle VII (Test Integrity) — tests verify conformance to specs; they are deliverables, not afterthoughts.
- `fab/project/code-quality.md` mandates `test-alongside`.

So when an AI produces "200 lines of impl + 400 lines of tests," a single headline number either (a) hides that the bulk is tests (looks like bloat) or, worse, (b) tempts someone to dump tests into `true_impact_exclude` to "clean up" the number — which destroys the very signal that proves the work is tested.

### Consequence of not fixing

Reviewers cannot distinguish "this PR is mostly disciplined test coverage" from "this PR is 600 lines of implementation bloat." The metric loses its discriminating power exactly where the project cares most. The pressure to hide tests in the exclude list grows, conflating two orthogonal axes (scaffolding-noise vs. test-vs-impl) and permanently losing the split.

### Why this approach (attribution, not exclusion)

The discussion explicitly rejected adding test patterns to `true_impact_exclude`. Excluding tests would (a) conflate two orthogonal axes into one number and (b) lose the ability to show the split. Tests are NOT scaffolding noise like `fab/` and `docs/`. The chosen approach keeps the two axes separate:
- `true_impact_exclude` (existing) — strips scaffolding/doc *noise* from the universe entirely.
- `test_paths` (new) — *attributes* the remaining (already-scaffolding-excluded) lines to tests vs. implementation, showing both.

## What Changes

### 1. New config key `test_paths`

Add a top-level `test_paths` list to `fab/project/config.yaml` — project-defined glob/pathspec patterns identifying test files. Mirrors the existing `source_paths` and `true_impact_exclude` style (top-level list of strings).

**Rationale for project-defined (no kit default)**: fab-kit is language-agnostic (constitution principle V, Portability). There is NO universal test-file pattern — `*_test.go`, `test_*.py`, `*.spec.ts` all differ by language. The KIT MUST NOT ship a default test-pattern list; each project declares its own.

```yaml
# fab/project/config.yaml (this repo would set):
test_paths:
  - "**/*_test.go"
```

**Graceful collapse**: When `test_paths` is absent, null, or empty, behavior collapses to today's single-number display (no impl/tests split) — matching the existing lazy-omit posture of the `excluding` sub-block.

`config.Config` gains a `TestPaths []string` field reading `test_paths` (in `src/go/fab/internal/config/config.go`).

### 2. New `tests` sub-block in the `true_impact` block

Add a `tests` sub-block to the `true_impact` block of `.status.yaml`, computed via an additional `git diff --shortstat` pass with a test-only pathspec, **within the scaffolding-excluded universe** — i.e., test files are counted AFTER `true_impact_exclude` is applied, so a test fixture sitting under `docs/` is not double-counted. The existing raw `added`/`deleted`/`net` base fields stay as the underlying measurement; the `excluding` sub-block stays as today.

Target `.status.yaml` shape after this change:

```yaml
true_impact:
  added: 612            # raw (fab/ + docs/ included) — base measurement, unchanged
  deleted: 38
  net: 574
  excluding:            # only present when true_impact_exclude is non-empty (unchanged)
    added: 540          # raw minus true_impact_exclude  → this is the "total" row
    deleted: 38
    net: 502
  tests:                # NEW — only present when test_paths is non-empty
    added: 400          # test-only lines, within the scaffolding-excluded universe
    deleted: 0
    net: 400
  computed_at: "2026-05-30T..."
  computed_at_stage: apply
```

Note: `impl` is NOT stored as a separate field — it is the residual `total − tests`, derived at render time (see Decision 4 + the guard in Decision 5). Storing it would risk drift between the two diff passes.

### 3. Canonical impact math (the shared engine)

`src/go/fab/internal/impact/impact.go` is the canonical shortstat math shared by the `fab impact` CLI and the status-finish hook. Extend it:
- Add a test-only `git diff --shortstat` pass. The pathspec for this pass must include BOTH the test-path includes AND the `true_impact_exclude` excludes, so the test count lives inside the scaffolding-excluded universe (e.g., `git diff --shortstat <base>...<head> -- <test globs> :(exclude)fab/ :(exclude)docs/`). Reuse the existing `runShortstat` helper (it already supports exclude pathspecs; extend to also accept include pathspecs).
- Extend the `Result` struct with a `Tests *Pair` field (alongside the existing `Excluding *Pair`). `Tests` is nil when `test_paths` is empty.
- `Compute(repoDir, base, head, excludes, testPaths)` — the signature gains `testPaths`. When `testPaths` is empty, skip the test pass and leave `Tests` nil.
- `ComputeForRepo` reads both `cfg.TrueImpactExclude` and `cfg.TestPaths` and passes them through.

### 4. Three-row rendering

**In the `/git-pr` PR body** (`src/kit/skills/git-pr.md`), render the breakdown as three rows when `tests` is present:

```
True impact:
  impl:  +140 / −38  (net +102)
  tests: +400 / −0   (net +400)
  total: +540 / −38  (net +502)   ← excludes fab/, docs/
```

KEY DECISIONS embedded here:
- The `total` row is the SCAFFOLDING-EXCLUDED number — it repurposes/replaces today's headline with what is currently the `excluding` block (raw minus `true_impact_exclude`). The raw-with-fab/docs-included number is NOT displayed in the PR body (it remains in `.status.yaml` as the base measurement).
- The relationship is `impl + tests = total`, where `impl = total − tests` (impl is the residual, not separately measured).
- The `(excludes fab/, docs/)` / `(excluding ...)` annotation MUST reflect the actual `true_impact_exclude` config values, not be hardcoded.

**Compact form in `fab change list --show-stats`** (`impactColumn()` in `src/go/fab/internal/change/change.go`): extend the single net column to show a compact split when `tests` is present, using the explicit-equation form `{impl_net}i+{tests_net}t={total_net}` (e.g., `102i+400t=502`) — all three values visible.
<!-- clarified: compact format chosen as `102i+400t=502` (explicit equation) over `502 (102+400)` and `+102i/+400t` -->
Falls back to today's single `excluding.net` (else `net`, else `—`) when `tests` absent.

### 5. Correctness guard (negative-impl clamp)

`impl = total − tests` is arithmetic across two separate `git diff` passes. If a `test_paths` glob accidentally matches a file that is ALSO excluded by `true_impact_exclude`, `tests` could over-count relative to `total` and drive `impl` negative. Guard:
- **Per-component clamp**: `impl.added = max(0, total.added − tests.added)`, `impl.deleted = max(0, total.deleted − tests.deleted)`, `impl.net = max(0, total.net − tests.net)` — each of added/deleted/net clamped independently, because the three-row display shows separate `+X / −Y` components and each must be non-negative on its own.
<!-- clarified: clamp scope chosen as per-component (added/deleted/net independently), not net-only -->
- Emit a one-line stderr warning when any component clamp triggers (consistent with the best-effort stderr posture of `WriteTrueImpact`).
- NEVER render a negative impl line or component.

**Where the residual is computed**: render time only. `.status.yaml` and the `fab impact` YAML store only the *measured* passes (raw, `excluding`, `tests`). The `impl` residual and its clamp are derived in the consumers — `/git-pr` PR body assembly and `impactColumn()` — not in the impact engine. This keeps the engine pure-measurement so no derived field can drift or go stale; the cost is that the clamp logic is implemented at both render sites.
<!-- clarified: residual/clamp computed at render time only (no derived impl field in engine, .status.yaml, or fab impact YAML) -->


### 6. Edge cases

- `test_paths` empty/absent/null → no `tests` sub-block written; rendering collapses to a single `total` line (today's behavior). No impl/tests split.
- `true_impact_exclude` empty → `total` degenerates to raw (nothing to strip); the `excluding` sub-block is absent today, so `total` falls back to raw `net`. Tests can still be split out if `test_paths` is set (computed within the raw universe, since there is nothing to exclude).
- Both stay graceful per the existing lazy-omit posture.

### 7. CLI and YAML surface

- `fab impact <base> <head>` (`src/go/fab/cmd/fab/impact.go`) — `renderYAML` emits the new `tests` sub-block in its YAML output (only when present), so `/git-pr` can parse it via `yq`.
- `src/go/fab/internal/status/true_impact.go` — `WriteTrueImpact` copies `res.Tests` into a new `sf.TrueImpactPair` on the `TrueImpact` struct.
- `src/go/fab/internal/statusfile/statusfile.go` — extend `TrueImpact` with a `Tests *TrueImpactPair` field (`yaml:"tests,omitempty"`); update `encodeTrueImpact` to emit the `tests` mapping (placed after `excluding`, before `computed_at`); update the decode path (`val.Decode(ti)` already covers it via the struct tag). Update `impactColumn()` for the compact `change list --show-stats` form.

### 8. Scaffold config placeholder

`src/kit/scaffold/fab/project/config.yaml` — add a commented-out `test_paths` placeholder (no default patterns) so users discover the knob. Lean toward the placeholder over leaving it out entirely.

```yaml
# Glob/pathspec patterns identifying test files. Used by the true-impact
# breakdown to attribute lines to tests vs. implementation (impl = total − tests).
# Language-specific — no kit default. When absent/empty, the breakdown collapses
# to a single total line. Examples: "**/*_test.go", "test_*.py", "**/*.spec.ts".
# test_paths: []
```

### 9. Documentation (constitution-mandated)

- CLI changes (Go binary) MUST update `src/kit/skills/_cli-fab.md` with the new `tests` sub-block in the `fab impact` output schema, AND MUST include test updates.
- Skill changes (`src/kit/skills/git-pr.md`) MUST update `docs/specs/skills/SPEC-git-pr.md`.
- All canonical edits go under `src/` (per constitution: edit `src/kit/`, never the `.claude/skills/` deployed copies).

## Affected Memory

- `fab-workflow/schemas`: (modify) Document the `tests` sub-block in the `true_impact` schema — its fields (`added`/`deleted`/`net`), that it is computed within the scaffolding-excluded universe, that it is lazily omitted when `test_paths` is empty, and the `impl = max(0, total − tests)` residual relationship.
- `fab-workflow/configuration`: (modify) Document the new top-level `test_paths` config field — purpose (attribution not exclusion), project-defined (no kit default, Portability rationale), glob/pathspec format, and graceful-collapse behavior when absent/empty.

## Impact

Canonical sources under `src/` (per constitution — edit `src/kit/` and `src/go/`, never `.claude/skills/` copies):

- `src/go/fab/internal/impact/impact.go` — add test-only `git diff --shortstat` pass; extend `Result` with `Tests *Pair`; `Compute`/`ComputeForRepo` signatures gain `testPaths`; extend `runShortstat` to accept include pathspecs.
- `src/go/fab/internal/config/config.go` — add `TestPaths []string` field (`yaml:"test_paths"`).
- `src/go/fab/internal/status/true_impact.go` — `WriteTrueImpact` writes the new `tests` sub-block from `res.Tests`.
- `src/go/fab/internal/statusfile/statusfile.go` — extend `TrueImpact` with `Tests *TrueImpactPair`; update `encodeTrueImpact`; update `impactColumn()` for the compact `change list --show-stats` form.
- `src/go/fab/cmd/fab/impact.go` — `renderYAML` emits the `tests` sub-block.
- `src/kit/skills/git-pr.md` — three-row impact rendering in the PR body assembly (Impact line population).
- `src/kit/scaffold/fab/project/config.yaml` — commented `test_paths` placeholder.
- `src/kit/skills/_cli-fab.md` — document the new `tests` sub-block in `fab impact` output (constitution-mandated for CLI changes).
- `docs/specs/skills/SPEC-git-pr.md` — update for the skill change (constitution-mandated).
- Tests: unit tests in `src/go/fab/internal/impact/` for the new test pass, the negative-impl clamp guard, and the empty-`test_paths` / empty-`true_impact_exclude` edges. Plus `internal/statusfile` encode/decode coverage for the `tests` sub-block.

**Out of scope** (do NOT pull in): the parsimony pass, deletion candidates, or any other ogf2 bloat-restraint interventions. This change is purely the impl/test attribution split on top of the already-shipped `true_impact` infrastructure.

**Migration**: NOT needed — the new `tests` field is optional and lazy (absent `test_paths` = today's behavior; existing `.status.yaml` files without `tests` remain valid; existing configs without `test_paths` collapse gracefully). Final schema verification is performed at apply against the in-flight `.status.yaml` schema.

## Open Questions

> Resolved during clarify (see `## Clarifications`):
> - ~~Compact format for `fab change list --show-stats`~~ → `102i+400t=502` (explicit equation).
> - ~~Clamp scope (net-only vs. per-component)~~ → per-component (added/deleted/net independently).
> - ~~Where the residual/clamp is computed~~ → render time only; engine stays pure-measurement.

(none open — migration confirmed not needed; apply performs final schema verification against the in-flight `.status.yaml` schema.)

## Clarifications

### Session 2026-05-30

| # | Action | Detail |
|---|--------|--------|
| 14 | Changed | Compact `--show-stats` column → `102i+400t=502` (explicit equation; chosen over `502 (102+400)` and `+102i/+400t`) |
| 15 | Changed | Clamp scope → per-component (added/deleted/net independently); not net-only |
| 16 | Changed | Residual/clamp → render time only; no derived `impl` in engine, `.status.yaml`, or `fab impact` YAML |

### Session 2026-05-30 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 9 | Confirmed | — |
| 10 | Confirmed | — |
| 11 | Confirmed | — |
| 12 | Confirmed | — |
| 13 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Goal is attribution, not exclusion — do NOT add test patterns to `true_impact_exclude` | Explicit discussion decision; conflating the two axes loses the split and the test-coverage signal. Tests are not scaffolding noise. | S:98 R:80 A:90 D:95 |
| 2 | Certain | New top-level `test_paths` config key (project-defined glob/pathspec list), mirroring `source_paths`/`true_impact_exclude` | Explicit decision; Portability (principle V) forbids a universal kit-shipped test pattern. | S:95 R:75 A:90 D:90 |
| 3 | Certain | Kit ships NO default `test_paths`; absent/null/empty collapses to today's single-number display | Explicit decision + constitution V; matches existing lazy-omit posture of `excluding`. | S:95 R:85 A:92 D:95 |
| 4 | Certain | New `tests` sub-block in `true_impact`, computed via an extra `git diff --shortstat` pass with a test-only pathspec WITHIN the scaffolding-excluded universe (after `true_impact_exclude`) | Explicit decision; counting after excludes prevents double-counting test fixtures under `fab/`/`docs/`. Raw `added/deleted/net` base fields stay. | S:95 R:70 A:85 D:90 |
| 5 | Certain | Three-row rendering (impl / tests / total) in `/git-pr`; `total` = scaffolding-excluded number (repurposes the `excluding` block), raw-with-fab/docs NOT shown in PR body | Explicit decision with the exact three-row layout provided. | S:95 R:78 A:85 D:88 |
| 6 | Certain | `impl + tests = total` where `impl = total − tests` (residual, not separately measured); `(excludes …)` annotation reflects actual config values, never hardcoded | Explicit decision; storing impl separately would risk drift between passes. Annotation-from-config matches existing asvz behavior. | S:95 R:80 A:88 D:90 |
| 7 | Certain | Correctness guard: `impl = max(0, total − tests)`, one-line stderr warning on clamp, never render negative impl | Explicit decision; cross-pass arithmetic can go negative if a test glob overlaps an excluded path. Stderr posture matches `WriteTrueImpact`. | S:96 R:82 A:90 D:92 |
| 8 | Certain | Edge cases: empty `test_paths` → single `total` line; empty `true_impact_exclude` → `total` degenerates to raw; both graceful | Explicit decision; consistent with existing lazy-omit semantics. | S:95 R:85 A:90 D:92 |
| 9 | Certain | Canonical math extension lives in `internal/impact/impact.go` (extend `Result` with `Tests *Pair`, thread `testPaths` through `Compute`/`ComputeForRepo`, extend `runShortstat` for include pathspecs) | Clarified — user confirmed (bulk). Shared engine for both CLI and status hook; obvious extension of existing code. | S:95 R:65 A:85 D:82 |
| 10 | Certain | `statusfile.TrueImpact` gains `Tests *TrueImpactPair` (`yaml:"tests,omitempty"`); `encodeTrueImpact` emits it after `excluding`, before `computed_at`; `impactColumn()` updated for compact split | Clarified — user confirmed (bulk). Follows the existing `Excluding` field pattern exactly. | S:95 R:68 A:88 D:80 |
| 11 | Certain | Scaffold config gets a commented-out `test_paths` placeholder (no default patterns) | Clarified — user confirmed (bulk). Reversible doc-only change; clear front-runner over omitting it. | S:95 R:88 A:80 D:78 |
| 12 | Certain | No migration needed (new field optional + lazy; existing files/configs remain valid) | Clarified — user confirmed (bulk). Strong signal from existing lazy `true_impact` precedent. | S:95 R:70 A:82 D:80 |
| 13 | Certain | Documentation: update `_cli-fab.md` (CLI), `SPEC-git-pr.md` (skill), memory `schemas` + `configuration`; tests in `internal/impact/` (clamp + edges) | Clarified — user confirmed (bulk). Constitution-mandated; direct requirement, low ambiguity. | S:95 R:75 A:90 D:85 |
| 14 | Certain | Compact `--show-stats` column format is `102i+400t=502` (explicit equation: impl + tests = total; bare net when no `test_paths`) | Clarified — user chose explicit-equation form over `502 (102+400)` and `+102i/+400t`; all three values visible. | S:95 R:88 A:65 D:45 |
| 15 | Certain | Clamp applies per-component (added/deleted/net independently): `impl.X = max(0, total.X − tests.X)` for each of added, deleted, net | Clarified — user chose per-component over net-only; the three-row impl display shows separate `+X / −Y`, so each component must be non-negative independently. | S:95 R:80 A:70 D:48 |
| 16 | Certain | Residual/clamp computed at render time only — `.status.yaml` and `fab impact` YAML store only measured passes (raw, excluding, tests); consumers (`/git-pr` body assembly + `impactColumn`) derive and clamp `impl` | Clarified — user chose render-time only; engine stays pure-measurement, no derived field can drift or go stale. Clamp logic lives in the consumer render sites. | S:95 R:62 A:68 D:50 |

16 assumptions (16 certain, 0 confident, 0 tentative, 0 unresolved).
