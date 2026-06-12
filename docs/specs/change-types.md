# Change Types

Fab uses 7 change types derived from [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/). Change types control confidence scoring thresholds, PR formatting, and pipeline gating.

---

## The 7 Types

| Type | Description | Examples |
|------|-------------|----------|
| `feat` | New feature or capability | Add OAuth support, implement search, new API endpoint |
| `fix` | Bug fix or regression fix | Fix login crash, correct calculation, resolve race condition |
| `refactor` | Code restructuring without behavior change | Extract shared module, rename functions, reorganize directories |
| `docs` | Documentation-only changes | Update README, add API guide, fix typos in docs |
| `test` | Test additions or modifications | Add unit tests, improve coverage, fix flaky test |
| `ci` | CI/CD pipeline changes | Update GitHub Actions, add deployment step, fix build script |
| `chore` | Maintenance, cleanup, housekeeping | Bump dependencies, clean up dead code, update configs |

Types use the short conventional commit prefix form (e.g., `feat`, not `feature`). Consolidated from the full Conventional Commits spec: `style` → `refactor`, `perf` → `feat`/`refactor`, `build` → `ci`.

---

## Expected Minimum Decisions

The `expected_min` threshold defines how many SRAD decisions a change should have at intake (the sole scoring stage). It drives the **coverage factor** in confidence scoring — thin intakes with fewer decisions than expected get attenuated scores. As of 1.10.0 there is a single `expected_min` table (the former per-stage intake/spec split is gone with the spec stage).

| Type | `expected_min` |
|------|----------------|
| `feat` | 7 |
| `refactor` | 6 |
| `fix` | 5 |
| `docs` | 3 |
| `test` | 3 |
| `ci` | 3 |
| `chore` | 3 |

Types without an explicit entry (`docs`, `test`, `ci`, `chore`) use the default of 3. The canonical source is the `expectedMin` map in `src/go/fab/internal/score/score.go`; this table is a verified mirror of it. A test in that package (`TestDocTablesMatchScoringMaps`) fails if the two drift.

---

## Gate Thresholds

`/fab-ff` and `/fab-fff` require the intake confidence score to meet the gate threshold before entering the automated bracket.

| Type | Gate Threshold |
|------|----------------|
| `fix` | 3.0 |
| `feat` | 3.0 |
| `refactor` | 3.0 |
| `docs` | 3.0 |
| `test` | 3.0 |
| `ci` | 3.0 |
| `chore` | 3.0 |

As of 1.10.0 the gate is **flat 3.0 for all types** — a single intake gate replacing the former two-gate (fixed-3.0 intake + per-type spec) model, keeping every type's bar ≥ both old gates. The canonical source is the `gateThresholds` map in `src/go/fab/internal/score/score.go` (resolved via `getGateThreshold`, which keeps future per-type divergence a data-only change); this table is a verified mirror of it, guarded by the same `TestDocTablesMatchScoringMaps` test that fails if the two drift. The gate check is performed by `fab score --check-gate --stage intake`.

---

## PR Template Tiers

| Tier | Types | Template |
|------|-------|----------|
| **Tier 1 — Fab-Linked** | `feat`, `fix`, `refactor` | Summary/Changes/Context with blob URL links to intake and plan |
| **Tier 2 — Lightweight** | `docs`, `test`, `ci`, `chore` | Auto-generated summary with "No design artifacts — housekeeping change" |

PR titles always use the `{type}: {title}` prefix format.

---

## Keyword Heuristics for Inference

`/fab-new` infers the change type from intake content using keyword matching (case-insensitive, evaluated in order, first match wins):

| Priority | Keywords | Inferred Type |
|----------|----------|---------------|
| 1 | fix, bug, broken, regression | `fix` |
| 2 | refactor, restructure, consolidate, split, rename | `refactor` |
| 3 | docs, document, readme, guide | `docs` |
| 4 | test, spec, coverage | `test` |
| 5 | ci, pipeline, deploy, build | `ci` |
| 6 | chore, cleanup, maintenance, housekeeping | `chore` |
| 7 | *(no match)* | `feat` |

The inferred type is written to `.status.yaml` via `fab status set-change-type` (the `artifact-write` hook does this automatically on every `intake.md` write). `/git-pr` reads this value as step 2 in its resolution chain, avoiding re-inference.

---

## Lifecycle

1. **Inference** (`/fab-new`): Type is inferred from intake keywords and stored in `.status.yaml`
2. **Scoring** (`fab score`): Type determines `expected_min` for coverage-weighted confidence (computed from `intake.md`)
3. **Gating** (`/fab-ff`, `/fab-fff`): Type routes through `getGateThreshold` (flat 3.0 today) for the single intake gate
4. **PR creation** (`/git-pr`): Type determines PR title prefix and body template tier
