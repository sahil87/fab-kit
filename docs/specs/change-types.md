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

The `expected_min` threshold records how many SRAD decisions a change is typically expected to have at intake (the sole scoring stage). It is **documentation-only** — it feeds no formula. The v2 demerit score dropped the former coverage factor, so a thin intake is no longer attenuated for being short (see [srad.md](srad.md) § Gate Threshold, "no coverage factor"); the table below survives as the canonical mirror the doc-drift guard checks, and as a rough authoring cue for how much decision surface a type usually has. As of 1.10.0 there is a single `expected_min` table (the former per-stage intake/spec split is gone with the spec stage).

| Type | `expected_min` |
|------|----------------|
| `feat` | 7 |
| `refactor` | 6 |
| `fix` | 3 |
| `docs` | 3 |
| `test` | 3 |
| `ci` | 3 |
| `chore` | 3 |

Types without an explicit entry (`docs`, `test`, `ci`, `chore`) use the default of 3. The canonical source is the `expectedMin` map in `src/go/fab/internal/score/score.go`; this table is a verified mirror of it. A test in that package (`TestDocTablesMatchScoringMaps`) fails if the two drift — which is now the map's *only* consumer: `computeScore` never reads it.

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

## PR Formatting

PR titles always use the `{type}: {title}` prefix format — the one place the change type shapes a PR.

The **body does not vary by type.** `/git-pr` renders a single unified template for every change; the fab-linked fields (the mechanically-rendered `## Meta` block) are populated from **artifact availability** — whether the change resolves and its artifacts exist — not from the type. The former two-tier model (Fab-Linked for `feat`/`fix`/`refactor`, Lightweight for the rest) is retired: it hid real planning work on `docs`/`test`/`chore` changes that had gone through the full pipeline. See `docs/memory/pipeline/execution-skills.md` § Unified PR Template with Conditional Field Population.

---

## Keyword Heuristics for Inference

`/fab-new` infers the change type from intake content using keyword matching (case-insensitive, evaluated in order, first match wins):

| Priority | Keywords | Inferred Type |
|----------|----------|---------------|
| 1 | fix, bug, broken, regression — but see the `must-fix` discount below | `fix` |
| 2 | refactor, restructure, consolidate, split, rename, redesign | `refactor` |
| 3 | docs, document, readme, guide | `docs` |
| 4 | test, spec, coverage | `test` |
| 5 | ci, pipeline, deploy, build | `ci` |
| 6 | chore, cleanup, maintenance, housekeeping | `chore` |
| 7 | *(no match)* | `feat` |

**The `must-fix` discount.** Priority 1 is not a plain keyword match. Before testing for a fix signal, `must-fix` / `must fix` occurrences are blanked out of the content — that phrase is a *review directive*, the one fix-adjacent token a feature intake legitimately carries. So an otherwise-feature intake that says "address must-fix findings" does **not** infer `fix`. Everything else still counts: `bug`, `broken`, `regression`, a standalone `fix`, and fix-describing compounds like `bug-fix` / `hot-fix` (hyphens are word boundaries in Go's RE2, so `fix` inside a compound matches). Implemented as `fixSignal` in `src/go/fab/internal/artifact/artifact.go`; the other six rows match their regex directly.

The inferred type is recomputed by `fab status refresh` (self-healed at the transition seams — `fab status advance`/`finish`, `fab preflight`); a manual `fab status set-change-type` marks it sticky `explicit`, which `refresh` never overwrites. `/git-pr` reads this value as step 2 in its resolution chain, avoiding re-inference.

---

## Lifecycle

1. **Inference** (`/fab-new`): Type is inferred from intake keywords and stored in `.status.yaml`
2. **Scoring** (`fab score`): Type does not enter the confidence formula — the score is computed per decision from `intake.md` alone. `expected_min` is documentation-only (§ Expected Minimum Decisions)
3. **Gating** (`/fab-ff`, `/fab-fff`): Type routes through `getGateThreshold` (flat 3.0 today) for the single intake gate
4. **PR creation** (`/git-pr`): Type determines the PR title prefix; the body is unified and artifact-driven
