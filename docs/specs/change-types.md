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

## Expected Minimum Decisions (Tier 1 only)

The `expected_min` thresholds define how many SRAD decisions a Tier 1 change should have at each planning stage. These drive the **coverage factor** in confidence scoring — thin specs with fewer decisions than expected get attenuated scores.

| Type | Intake `expected_min` | Spec `expected_min` |
|------|----------------------|---------------------|
| `fix` | 2 | 4 |
| `feat` | 4 | 6 |
| `refactor` | 3 | 5 |

Tier 2 types (`docs`, `test`, `ci`, `chore`) skip the fab pipeline and do not require design artifacts, so these thresholds do not apply to them.

Thresholds were calibrated from archive analysis of 124 completed changes (84% unaffected at these values). Unknown or null types default to: intake=2, spec=3.

These values are embedded directly in `fab/.kit/scripts/lib/calc-score.sh` since the script ships with `fab/.kit/` to projects.

---

## Gate Thresholds

`/fab-ff` requires the confidence score to meet a type-specific threshold before the fast-forward pipeline can execute.

| Type | Gate Threshold | Rationale |
|------|---------------|-----------|
| `fix` | 2.0 | Low risk, narrow scope — more tolerance for assumptions |
| `feat` | 3.0 | Default — balanced risk tolerance |
| `refactor` | 3.0 | Behavioral preservation important, moderate tolerance |
| `docs` | 2.0 | Low blast radius, documentation-only |
| `test` | 2.0 | Low blast radius, test-only |
| `ci` | 2.0 | Low blast radius, infrastructure-only |
| `chore` | 2.0 | Low blast radius, maintenance |

Unknown types default to 3.0 (the `feat` threshold). The gate check is performed by `calc-score.sh --check-gate`.

---

## Pipeline Requirements by Tier

The change type determines whether the core fab pipeline (intake → spec → tasks → apply → review → hydrate → ship) is required. Post-ship stages like `review-pr` (via `/git-pr-review`) apply to both tiers and are documented separately:

| Tier | Types | Intake Required | Spec Required | Pipeline |
|------|-------|----------------|---------------|----------|
| **Tier 1 — Fab-Linked** | `feat`, `fix`, `refactor` | Yes | Yes | Full fab pipeline via `/fab-new` → `/fab-continue` → `/git-pr` |
| **Tier 2 — Lightweight** | `docs`, `test`, `ci`, `chore` | No | No | Direct commit and PR — no design artifacts needed |

**Tier 1** changes modify application behavior and require design artifacts (intake and spec) to document the problem, approach, and decisions. PRs include blob URL links to intake and spec.

**Tier 2** changes are low-risk changes that do not modify production application behavior. They skip the fab change pipeline (`/fab-new`, `/fab-continue`, `/fab-ff`) — no intake, spec, or design artifacts are generated. PRs are created via `/git-pr` (which auto-detects Tier 2) with "No design artifacts — housekeeping change."

PR titles always use the `{type}: {title}` prefix format regardless of tier.

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

The inferred type is written to `.status.yaml` via `statusman.sh set-change-type`. `/git-pr` reads this value as step 2 in its resolution chain, avoiding re-inference.

---

## Lifecycle

1. **Inference** (`/fab-new`): Type is inferred from intake keywords and stored in `.status.yaml`
2. **Scoring** (`calc-score.sh`): Type determines `expected_min` for coverage-weighted confidence
3. **Gating** (`/fab-ff`): Type determines the confidence threshold for pipeline execution
4. **PR creation** (`/git-pr`): Type determines PR title prefix and body template tier
