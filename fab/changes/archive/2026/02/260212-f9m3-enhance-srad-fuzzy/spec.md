# Spec: Enhance SRAD Confidence Scoring with Fuzzy Dimensions

**Change**: 260212-f9m3-enhance-srad-fuzzy
**Created**: 2026-02-14
**Affected memory**: `docs/memory/fab-workflow/planning-skills.md` (modify), `docs/memory/fab-workflow/context-loading.md` (modify)

## Non-Goals

- Replacing the 4-dimension SRAD framework — the S, R, A, D dimensions are preserved as-is
- Changing the grade taxonomy — Certain, Confident, Tentative, Unresolved remain the four grades
- Auto-tuning weights at runtime — weight validation is a one-time analysis, not a feedback loop
- Changing how agents perform qualitative reasoning — fuzzy scoring formalizes the evaluation, it doesn't replace agent judgment

## SRAD Fuzzy Scoring: Dimension Evaluation

### Requirement: Per-Dimension Fuzzy Scoring

Each SRAD dimension (S, R, A, D) SHALL be evaluated on a continuous 0–100 scale instead of the current binary high/low classification. Planning skills MUST produce a numeric score for each dimension when evaluating a decision point. The score represents the degree to which that dimension favors autonomous assumption (100 = fully safe to assume, 0 = must ask).

<!-- clarified: Trapezoidal boundaries at 0/30/60/85 confirmed — asymmetric bands reflect that Certain should be rare, Tentative is the widest class -->

#### Scenario: Agent Evaluates a Clear Decision Point

- **GIVEN** a planning skill encounters a decision point with detailed user input, high reversibility, strong codebase signals, and one obvious interpretation
- **WHEN** the agent evaluates SRAD dimensions
- **THEN** each dimension score SHALL be in the 75–100 range
- **AND** the resulting grade SHALL be Certain or Confident

#### Scenario: Agent Evaluates an Ambiguous Decision Point

- **GIVEN** a planning skill encounters a decision point with vague user input and multiple valid interpretations
- **WHEN** the agent evaluates SRAD dimensions
- **THEN** the Signal (S) and Disambiguation (D) scores SHALL be in the 0–40 range
- **AND** the resulting grade SHALL be Tentative or Unresolved depending on Reversibility and Agent Competence

#### Scenario: Backward Compatibility with Existing Changes

- **GIVEN** an existing change with binary high/low dimension evaluations in its brief or spec
- **WHEN** `calc-score.sh` processes the change
- **THEN** it SHALL fall back to the current grade-counting logic (no per-dimension scores expected)
- **AND** the confidence score SHALL be computed identically to the current formula

### Requirement: Dimension Score Persistence

Per-dimension scores for each decision point MAY be persisted in the Assumptions table as an optional `Scores` column. The `.status.yaml` confidence block SHALL include aggregate dimension statistics when fuzzy scoring is active.

The `.status.yaml` confidence block SHALL be extended with:

```yaml
confidence:
  certain: 5
  confident: 3
  tentative: 0
  unresolved: 0
  score: 4.1
  fuzzy: true              # flag indicating fuzzy scoring was used
  dimensions:              # aggregate stats across all decision points
    signal: 78.5           # mean S score
    reversibility: 82.0    # mean R score
    competence: 71.2       # mean A score
    disambiguation: 85.0   # mean D score
```

#### Scenario: Status File with Fuzzy Scoring

- **GIVEN** a change where spec generation used fuzzy dimension evaluation
- **WHEN** `calc-score.sh` computes the confidence score
- **THEN** the `fuzzy: true` flag SHALL be set in `.status.yaml`
- **AND** mean dimension scores SHALL be computed and stored under `dimensions:`

#### Scenario: Status File without Fuzzy Scoring (Legacy)

- **GIVEN** a change where spec generation used binary dimension evaluation
- **WHEN** `calc-score.sh` processes the change
- **THEN** the `fuzzy` key SHALL be absent (or false) in `.status.yaml`
- **AND** no `dimensions:` block SHALL be present

## SRAD Fuzzy Scoring: Grade Derivation

### Requirement: Fuzzy-to-Grade Mapping

The four per-dimension scores (S, R, A, D) SHALL be aggregated into a single composite score using a weighted mean, then mapped to a confidence grade via trapezoidal membership thresholds.
<!-- clarified: Weighted mean aggregation confirmed — standard MCDA approach with R-bias; Critical Rule override catches worst cases -->

**Aggregation formula**:

```
composite = w_S * S + w_R * R + w_A * A + w_D * D
```

where `w_S + w_R + w_A + w_D = 1.0`. Default weights: `w_S = 0.25, w_R = 0.30, w_A = 0.25, w_D = 0.20`.

The higher weight on Reversibility (R) reflects the Critical Rule: low-R decisions have cascading blast radius and deserve extra influence on the grade.

**Grade thresholds** (trapezoidal boundaries on composite score):

| Grade | Composite Range | Interpretation |
|-------|----------------|----------------|
| Certain | 85–100 | All dimensions strongly favor assumption |
| Confident | 60–84 | Most dimensions favor assumption; minor gaps |
| Tentative | 30–59 | Mixed signals; reasonable guess but alternatives exist |
| Unresolved | 0–29 | Too ambiguous to assume safely |

**Critical Rule override**: Regardless of composite score, if R < 25 AND A < 25, the grade MUST be Unresolved. This preserves the existing Critical Rule (low Reversibility + low Agent Competence = always ask).

#### Scenario: High Composite Score Maps to Certain

- **GIVEN** a decision point with S=90, R=85, A=95, D=88
- **WHEN** the composite score is calculated with default weights
- **THEN** composite = 0.25*90 + 0.30*85 + 0.25*95 + 0.20*88 = 89.35
- **AND** grade SHALL be Certain (89.35 >= 85)

#### Scenario: Mixed Scores Map to Tentative

- **GIVEN** a decision point with S=40, R=70, A=55, D=30
- **WHEN** the composite score is calculated with default weights
- **THEN** composite = 0.25*40 + 0.30*70 + 0.25*55 + 0.20*30 = 50.75
- **AND** grade SHALL be Tentative (30 <= 50.75 < 60)

#### Scenario: Critical Rule Override

- **GIVEN** a decision point with S=60, R=20, A=15, D=70
- **WHEN** the composite score is calculated (composite = 0.25*60 + 0.30*20 + 0.25*15 + 0.20*70 = 38.75)
- **THEN** grade SHALL be Unresolved (not Tentative)
- **AND** the reason SHALL note "Critical Rule: R=20 < 25 AND A=15 < 25"

## SRAD Fuzzy Scoring: Weight Validation

### Requirement: Sensitivity Analysis Methodology

The penalty weights in the confidence formula (`0.3` for Confident, `1.0` for Tentative) SHALL be validated via sensitivity analysis using historical change data from completed changes in `fab/changes/archive/`.

The analysis SHALL cover two weight domains:

**Domain 1: Formula penalty weights** (historical data available):
1. Collect grade distributions from all archived `.status.yaml` files
2. Vary penalty weights across a systematic grid: Confident penalty in [0.1, 0.2, 0.3, 0.4, 0.5], Tentative penalty in [0.5, 0.75, 1.0, 1.25, 1.5]
3. For each weight combination, recompute scores for all historical changes
4. Measure correlation between computed scores and actual outcomes (review pass/fail, human intervention during apply)
5. Report the weight combination that maximizes discrimination between changes that needed intervention vs. those that didn't

**Domain 2: Dimension aggregation weights** (theoretical baseline — no historical per-dimension scores):
1. Vary dimension weights across a grid: w_R in [0.20, 0.25, 0.30, 0.35, 0.40], others distributed proportionally to sum to 1.0
2. For synthetic test cases (the worked examples in the spec), compute composite scores under each weight configuration
3. Verify that the Critical Rule override activates correctly under all weight configurations
4. Report the recommended dimension weight distribution and document the sensitivity of grade assignments to weight changes

Results SHALL be documented in `docs/specs/srad.md` under a new "Weight Validation" section.

#### Scenario: Running Sensitivity Analysis

- **GIVEN** archived changes exist in `fab/changes/archive/` with `.status.yaml` files
- **WHEN** the sensitivity analysis script is executed
- **THEN** it SHALL output a grid of weight combinations with correlation scores
- **AND** it SHALL recommend optimal weights (or confirm current weights are adequate)

#### Scenario: Insufficient Historical Data

- **GIVEN** fewer than 5 archived changes exist
- **WHEN** the sensitivity analysis is attempted
- **THEN** it SHALL report "Insufficient data for reliable sensitivity analysis (N < 5)"
- **AND** it SHALL recommend keeping current weights as defaults

### Requirement: Threshold Calibration

The 3.0 gate threshold for `/fab-fff` SHALL be tested against historical data to measure whether it correlates with actual need for human intervention.

The calibration SHALL compute:
1. For each archived change: confidence score at spec completion, and whether review passed on first attempt
2. The percentage of changes above 3.0 that passed review without intervention
3. The percentage of changes below 3.0 that required intervention
4. Precision/recall at the 3.0 boundary

#### Scenario: Threshold Validation Report

- **GIVEN** historical change data with confidence scores and review outcomes
- **WHEN** threshold calibration is executed
- **THEN** it SHALL report precision and recall at the 3.0 threshold
- **AND** it SHALL suggest an adjusted threshold if the current one shows poor discrimination

## SRAD Fuzzy Scoring: Dynamic Thresholds

### Requirement: Change Type Classification

Each change SHALL be classified into one of four types based on its brief content:

| Type | Indicators | Risk Profile |
|------|-----------|-------------|
| **bugfix** | Fix, bug, patch, regression, error | Low risk — narrow scope, clear criteria |
| **feature** | Add, new, implement, introduce | Medium risk — new behavior, broader scope |
| **refactor** | Refactor, restructure, reorganize, rename, move | Medium risk — existing behavior preserved |
| **architecture** | Architecture, system, foundation, framework, migrate | High risk — cascading changes, broad impact |

Classification SHALL be performed by the planning skill during brief generation and stored in `.status.yaml`:

```yaml
change_type: feature
```

If the brief content does not clearly match a category, the default SHALL be `feature` (medium risk).

#### Scenario: Bugfix Classification

- **GIVEN** a brief with description "Fix broken redirect after login"
- **WHEN** the change type is classified
- **THEN** `change_type` SHALL be `bugfix`

#### Scenario: Ambiguous Classification Defaults to Feature

- **GIVEN** a brief with description "Improve error handling"
- **WHEN** the change type is classified
- **THEN** `change_type` SHALL be `feature` (default)

### Requirement: Per-Type Gate Thresholds

The `/fab-fff` gate threshold SHALL vary by change type:

| Change Type | Gate Threshold | Rationale |
|-------------|---------------|-----------|
| bugfix | 2.0 | Low risk, narrow scope — more tolerance for assumptions |
| feature | 3.0 | Current default — balanced |
| refactor | 3.0 | Behavioral preservation important, moderate tolerance |
| architecture | 4.0 | High blast radius — demand near-certainty |

#### Scenario: Bugfix Passes Lower Gate

- **GIVEN** a bugfix change with confidence score 2.5
- **WHEN** `/fab-fff` checks the gate
- **THEN** the pipeline SHALL proceed (2.5 >= 2.0 bugfix threshold)

#### Scenario: Architecture Requires Higher Confidence

- **GIVEN** an architecture change with confidence score 3.5
- **WHEN** `/fab-fff` checks the gate
- **THEN** the pipeline SHALL refuse (3.5 < 4.0 architecture threshold)
- **AND** the message SHALL include the architecture-specific threshold

## Implementation: `calc-score.sh` Changes

### Requirement: Extended Assumptions Table Parsing

`calc-score.sh` SHALL support an optional `Scores` column in Assumptions tables:

```markdown
| # | Grade | Scores | Decision | Rationale |
|---|-------|--------|----------|-----------|
| 1 | Confident | S:75 R:80 A:65 D:70 | Use OAuth2 | Config shows REST API |
```

When the `Scores` column is present, `calc-score.sh` SHALL:
1. Parse per-dimension scores from the `S:nn R:nn A:nn D:nn` format
2. Compute mean dimension scores across all decision points
3. Write the `dimensions:` block and `fuzzy: true` flag to `.status.yaml`

When the `Scores` column is absent, `calc-score.sh` SHALL behave identically to the current implementation (backward compatible).

#### Scenario: Parsing Scores Column

- **GIVEN** a spec with Assumptions table containing a `Scores` column
- **WHEN** `calc-score.sh` processes the spec
- **THEN** it SHALL extract per-dimension scores from each row
- **AND** compute aggregate means for the `dimensions:` block

#### Scenario: No Scores Column (Legacy Fallback)

- **GIVEN** a spec with Assumptions table without a `Scores` column
- **WHEN** `calc-score.sh` processes the spec
- **THEN** it SHALL use the existing grade-counting logic only
- **AND** the `fuzzy` flag SHALL be absent from `.status.yaml`

### Requirement: Dynamic Threshold Lookup

`calc-score.sh` SHALL read `change_type` from `.status.yaml` and apply the corresponding gate threshold when invoked with a `--check-gate` flag.

#### Scenario: Gate Check with Change Type

- **GIVEN** a `.status.yaml` with `change_type: bugfix` and `score: 2.5`
- **WHEN** `calc-score.sh --check-gate` is invoked
- **THEN** it SHALL output `gate: pass` (2.5 >= 2.0)

#### Scenario: Gate Check without Change Type (Default)

- **GIVEN** a `.status.yaml` without `change_type` and `score: 2.5`
- **WHEN** `calc-score.sh --check-gate` is invoked
- **THEN** it SHALL use the default threshold of 3.0
- **AND** output `gate: fail` (2.5 < 3.0)

## Implementation: Test Suite

### Requirement: Comprehensive Test Coverage

`src/lib/calc-score/test.sh` SHALL be extended with test sections covering:

1. **Fuzzy dimension parsing** — Scores column extraction, mean computation, malformed input handling
2. **Grade threshold mapping** — composite scores at each boundary (0, 29, 30, 59, 60, 84, 85, 100)
3. **Critical Rule override** — low R + low A forces Unresolved regardless of composite
4. **Dynamic thresholds** — per-type gate check with each change type
5. **Weight sensitivity** — score variation under different Confident/Tentative penalty weights
6. **Backward compatibility** — legacy tables without Scores column produce identical results
7. **Edge cases** — empty Scores, partial dimension data, zero dimensions, maximum dimensions

### Requirement: Sensitivity Analysis Script

A new script `src/lib/calc-score/sensitivity.sh` SHALL implement the weight validation methodology. It SHALL:
1. Scan `fab/changes/archive/` for `.status.yaml` files
2. Extract grade distributions and review outcomes
3. Run the grid-based sensitivity analysis
4. Output results as a formatted table

#### Scenario: Running Sensitivity Script

- **GIVEN** the archive contains 10+ completed changes
- **WHEN** `sensitivity.sh` is executed
- **THEN** it SHALL output a weight grid with correlation scores
- **AND** exit 0

## Implementation: Documentation Updates

### Requirement: Spec Updates

`docs/specs/srad.md` SHALL be updated with:
1. Fuzzy dimension evaluation methodology (replacing binary evaluation criteria table)
2. Aggregation formula and default weights
3. Grade threshold table
4. Dynamic threshold table by change type
5. Weight validation findings (sensitivity analysis results)
6. Updated worked examples showing numeric dimension scores

### Requirement: Context File Updates

`fab/.kit/skills/_context.md` SHALL note that SRAD dimensions use fuzzy 0–100 scoring in the SRAD Scoring section, updating the evaluation criteria table from binary to continuous.

## Design Decisions

1. **Trapezoidal grade thresholds over continuous mapping**: Discrete grade boundaries (0–29, 30–59, 60–84, 85–100) preserve the existing 4-grade taxonomy and keep the confidence formula unchanged. A fully continuous mapping would require reworking the entire scoring pipeline.
   - *Why*: Minimizes blast radius — only dimension evaluation changes, not the downstream grade→score pipeline
   - *Rejected*: Continuous composite-to-score mapping — would bypass grades entirely and require rewriting calc-score.sh, /fab-fff gate logic, and all planning skill SRAD sections

2. **Weighted mean with R-bias over equal weights**: Giving Reversibility 0.30 weight (vs. 0.25 for others) encodes the Critical Rule's intent directly into the aggregation, reducing reliance on the override.
   - *Why*: Aligns quantitative scoring with the existing qualitative principle that low-R decisions are highest risk
   - *Rejected*: Equal weights (0.25 each) — doesn't reflect that R has disproportionate impact on rework cost

3. **Optional Scores column over mandatory migration**: Making the Scores column optional in Assumptions tables allows gradual adoption without breaking existing changes or requiring artifact migration.
   - *Why*: Constitution Principle III (Idempotent Operations) — existing changes must continue to work
   - *Rejected*: Mandatory migration of all existing artifacts — violates backward compatibility and risks corrupting in-progress changes

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Certain | Trapezoidal membership boundaries at 0/30/60/85/100 | Clarified: user confirmed asymmetric bands |
| 2 | Certain | Weighted mean for dimension aggregation (w_S=0.25, w_R=0.30, w_A=0.25, w_D=0.20) | Clarified: user confirmed weighted mean with R-bias |
| 3 | Confident | Per-dimension scores persisted in `.status.yaml` dimensions block | Brief suggests "may need additional fields"; natural storage location alongside existing confidence data |
| 4 | Confident | Change type categories: bugfix/feature/refactor/architecture | Brief explicitly lists these four; matches common software change taxonomy |
| 5 | Confident | Sensitivity analysis covers both formula penalties and dimension weights | Clarified: user chose both domains; historical data for penalties, theoretical baselines for dimension weights |
| 6 | Confident | Dynamic thresholds: bugfix=2.0, feature/refactor=3.0, architecture=4.0 | Proportional to risk profile; preserves 3.0 as default for medium-risk changes |
| 7 | Confident | Backward compatibility via optional Scores column | Constitution Principle III requires idempotent operations; existing artifacts must continue working |

7 assumptions made (5 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-14

1. **Grade boundaries**: Confirmed 0/30/60/85 trapezoidal thresholds. Asymmetric bands make Certain hard to achieve (narrow 85–100 band), which matches the design intent that truly Certain decisions should be rare. → Assumption #1 reclassified Tentative → Certain.

2. **Aggregation method**: Confirmed weighted mean with R-bias (w_R=0.30). Critical Rule override catches worst-case low-R+low-A scenarios. Min-operator and OWA rejected as overly conservative or complex. → Assumption #2 reclassified Tentative → Certain.

3. **Planning skill files**: User confirmed `_context.md` is the single source for SRAD agent behavior. No updates needed for `fab-new.md` or `fab-continue.md` — agents inherit SRAD instructions from `_context.md`. → No spec change; gap closed.

4. **Sensitivity analysis scope**: Expanded to cover both formula penalty weights (historical data) and dimension aggregation weights (theoretical baselines from synthetic test cases). → Assumption #5 updated, requirement text expanded with Domain 1 / Domain 2 structure.
