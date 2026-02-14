# Spec: Enhance SRAD Confidence Scoring

**Change**: 260212-f9m3-enhance-srad-fuzzy
**Created**: 2026-02-14
**Affected memory**: `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/context-loading.md`

## Non-Goals

- Replacing SRAD's four dimensions (Signal, Reversibility, Agent Competence, Disambiguation) with a different framework.
- Introducing non-linear or opaque scoring models that make score movement hard to explain.
- Turning `/fab-fff` into a multi-factor gate beyond confidence score and change type.

## SRAD Scoring: Fuzzy Dimension Inputs

### Requirement: Per-decision 0-100 dimension scoring

Planning-stage SRAD evaluation SHALL score each decision point on all four SRAD dimensions using integer values in the inclusive range 0-100, rather than binary high/low labels.

Each decision SHALL persist:
- `signal_score`
- `reversibility_score`
- `agent_competence_score`
- `disambiguation_score`

#### Scenario: Decision scored during spec generation

- **GIVEN** `/fab-continue` evaluates a decision while generating `spec.md`
- **WHEN** SRAD scoring is applied
- **THEN** the decision SHALL receive four integer scores from 0-100
- **AND** the decision SHALL NOT be stored as binary high/low only

### Requirement: Fuzzy membership-derived penalty contributions

Confidence penalties SHALL be derived from fuzzy memberships computed from a decision's composite SRAD value, so each decision can contribute fractional amounts to `confident` and `tentative` totals.

The implementation SHALL:
- Compute a composite SRAD decision value using weighted mean:

  `composite = (0.2 * signal_score) + (0.3 * reversibility_score) + (0.3 * agent_competence_score) + (0.2 * disambiguation_score)`
<!-- clarified: user confirmed weighted composite during /fab-clarify with S/R/A/D weights = 0.2/0.3/0.3/0.2 -->
- Derive `confident_membership` and `tentative_membership` as continuous values in [0.0, 1.0].
- Aggregate these memberships across decisions to produce effective totals:
  - `effective_confident`
  - `effective_tentative`

`effective_confident` and `effective_tentative` SHALL be used by the final score formula in place of integer-only grade counts.

#### Scenario: Mid-ambiguity decision contributes partially

- **GIVEN** a decision with a composite SRAD value in the middle band (neither clearly Certain nor clearly Tentative)
- **WHEN** fuzzy membership values are computed
- **THEN** both `confident_membership` and `tentative_membership` SHALL be non-zero
- **AND** the decision SHALL contribute partially to each effective penalty total

### Requirement: Interpretability-preserving confidence output

The confidence block in `.status.yaml` SHALL retain the existing top-level shape (`certain`, `confident`, `tentative`, `unresolved`, `score`) while allowing `confident` and `tentative` to be decimal values derived from fuzzy aggregation.

#### Scenario: Status output remains readable

- **GIVEN** fuzzy scoring is enabled
- **WHEN** `calc-score.sh` writes confidence to `.status.yaml`
- **THEN** the block SHALL still include `certain`, `confident`, `tentative`, `unresolved`, and `score`
- **AND** users SHALL be able to read it without learning a new schema

## Confidence Formula and Weight Calibration

### Requirement: Preserve linear formula structure

The final confidence score SHALL preserve the current linear structure:

```
if unresolved > 0:
  score = 0.0
else:
  score = max(0.0, 5.0 - (w_confident * effective_confident) - (w_tentative * effective_tentative))
```

This change SHALL update weight values and effective totals, but SHALL NOT replace the formula with a non-linear or black-box model.

#### Scenario: No unresolved decisions

- **GIVEN** `unresolved = 0`
- **WHEN** score is computed
- **THEN** score SHALL be derived by subtracting weighted effective penalties from 5.0
- **AND** score SHALL be clamped at 0.0 minimum

### Requirement: Sensitivity-analysis-driven weight validation

Penalty weights SHALL be selected via reproducible sensitivity analysis over historical change data from `fab/changes/**/.status.yaml` and corresponding artifacts.

The analysis SHALL:
- Evaluate multiple candidate pairs for `(w_confident, w_tentative)`.
- Measure stability of score ordering under small weight perturbations.
- Measure correlation between low scores and observed human-intervention proxies.
- Output the chosen weights and rationale.

Human-intervention proxies SHALL include:
- review-stage failure and rework loops
- clarify-session volume
- explicit task rework markers (`<!-- rework: ... -->`)

#### Scenario: Weight selection report is reproducible

- **GIVEN** a repository with archived changes
- **WHEN** the sensitivity-analysis workflow is run
- **THEN** the same input dataset and candidate grid SHALL produce the same selected weights
- **AND** the report SHALL include tested weight pairs, metrics, and selected pair

### Requirement: Baseline fallback when historical signal is sparse

If historical data is insufficient to produce stable calibrated weights, the system SHALL fall back to baseline weights `w_confident=0.3` and `w_tentative=1.0` until enough data is available.

#### Scenario: Small dataset fallback

- **GIVEN** fewer than the minimum required completed changes for stable sensitivity analysis
- **WHEN** calibration is attempted
- **THEN** calibration SHALL skip custom weight adoption
- **AND** the scoring engine SHALL use baseline weights 0.3 and 1.0

## Dynamic Thresholds by Change Type

### Requirement: Change-type classification for gating

Each change SHALL be classified into one of:
- `bugfix`
- `feature`
- `refactor`
- `architecture`

Classification SHALL be derived from change metadata and artifact cues.
If no class can be assigned confidently, class SHALL default to `feature`.
<!-- clarified: user confirmed default unknown type = feature during /fab-clarify -->

#### Scenario: Type inferred from brief and tasks

- **GIVEN** a change whose brief and tasks focus on defect correction with no new capability
- **WHEN** classification runs
- **THEN** the change type SHALL be `bugfix`

### Requirement: Per-type confidence gate thresholds

`/fab-fff` SHALL evaluate `confidence.score` against a threshold table keyed by change type:

- `bugfix`: 2.7
- `refactor`: 3.0
- `feature`: 3.3
- `architecture`: 3.6
<!-- clarified: user confirmed v1 thresholds (2.7/3.0/3.3/3.6) during /fab-clarify; retune only after post-rollout sensitivity results -->

#### Scenario: Lower-risk change passes lower threshold

- **GIVEN** a change classified as `bugfix` with `confidence.score=2.9`
- **WHEN** `/fab-fff` gate check runs
- **THEN** the gate SHALL pass

#### Scenario: Higher-risk change requires higher confidence

- **GIVEN** a change classified as `architecture` with `confidence.score=3.4`
- **WHEN** `/fab-fff` gate check runs
- **THEN** the gate SHALL fail
- **AND** output SHALL recommend `/fab-clarify`

### Requirement: Backward-compatible gate behavior

When change type is absent (older changes), `/fab-fff` SHALL use threshold 3.0 to preserve legacy behavior.

#### Scenario: Legacy change without type metadata

- **GIVEN** an older change with no type classification metadata
- **WHEN** `/fab-fff` evaluates gate eligibility
- **THEN** it SHALL apply threshold 3.0

## Rollout and Compatibility

### Requirement: Opt-in rollout control

Fuzzy SRAD scoring and dynamic thresholds SHALL be introduced behind an explicit scoring-mode control with at least two modes:
- `legacy` (existing integer-count scoring and fixed 3.0 threshold)
- `fuzzy` (new fuzzy scoring and dynamic threshold table)

Default mode for existing repositories SHALL be `legacy`.

#### Scenario: Existing repo upgrades kit version

- **GIVEN** a repository using current SRAD behavior
- **WHEN** it upgrades to the new kit version without changing scoring mode
- **THEN** confidence calculations and `/fab-fff` gate outcomes SHALL remain legacy-compatible

### Requirement: Mixed-history compatibility

Scoring logic SHALL support repositories where archived changes were produced under legacy scoring and new changes under fuzzy scoring.

#### Scenario: Historical analysis includes mixed modes

- **GIVEN** archive history containing both legacy and fuzzy-scored changes
- **WHEN** sensitivity analysis reads historical data
- **THEN** it SHALL process both without schema errors
- **AND** mode-specific assumptions SHALL be documented in the analysis output

## Tests and Documentation

### Requirement: Expand scoring test coverage

`src/lib/calc-score/test.sh` SHALL include test cases for:
- fuzzy membership computation boundaries
- fractional effective penalty aggregation
- calibrated weight application
- dynamic threshold gating by change type
- legacy-mode compatibility behavior
- sparse-data fallback to baseline weights

#### Scenario: Boundary coverage for fuzzy memberships

- **GIVEN** decision scores at exact boundary values for fuzzy bands
- **WHEN** tests run
- **THEN** expected membership outputs and resulting score contributions SHALL match specification

### Requirement: Update SRAD and skill documentation

The following docs SHALL be updated for behavioral consistency:
- `docs/specs/srad.md`
- `docs/specs/skills.md`
- `fab/.kit/skills/_context.md`

Documentation updates SHALL include:
- 0-100 per-dimension scoring model
- fuzzy membership aggregation concept
- sensitivity-analysis calibration method
- per-type threshold table and fallback behavior

#### Scenario: Documentation consistency check

- **GIVEN** a reader compares SRAD scoring sections across all affected docs
- **WHEN** the change is complete
- **THEN** formula structure, threshold behavior, and terminology SHALL be consistent

## Deprecated Requirements

### Binary-only dimension evaluation
**Reason**: Binary high/low scoring discards middle-range uncertainty and cannot represent partial confidence shifts.
**Migration**: Replace with 0-100 dimension scores and fuzzy membership-derived effective penalties.

### Single fixed `/fab-fff` threshold (3.0 for all change types)
**Reason**: Uniform threshold does not reflect different risk profiles across bugfix, feature, refactor, and architecture changes.
**Migration**: Use type-aware threshold table, with legacy fallback to 3.0 when type metadata is missing.

## Design Decisions

1. **Fuzzy membership on top of linear penalties**: Keep the linear confidence formula but feed it fractional effective penalty totals.
   - *Why*: Preserves user mental model and interpretability while adding gradation.
   - *Rejected*: Replacing SRAD with a non-linear MCDA model (less transparent in shell-script workflows).

2. **Empirical calibration over static retuning**: Select penalty weights via sensitivity analysis on historical data.
   - *Why*: Directly addresses research finding that small weight shifts can materially alter outcomes.
   - *Rejected*: One-time manual weight tuning without reproducible analysis.

3. **Type-aware thresholds with legacy fallback**: Use per-type gate thresholds but keep 3.0 for untyped legacy changes.
   - *Why*: Balances better risk calibration with smooth migration.
   - *Rejected*: Hard switch to dynamic thresholds with no fallback (would destabilize existing workflows).

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Human-intervention proxies can be inferred from review failures, clarify sessions, and rework markers | These signals are present in workflow artifacts and align with autonomy escalation behavior |
| 2 | Certain | Use initial dynamic thresholds: bugfix 2.7, refactor 3.0, feature 3.3, architecture 3.6 | Confirmed by user during `/fab-clarify` as the v1 baseline, with retuning deferred to post-rollout analysis |
| 3 | Confident | Keep formula linear while introducing fuzzy effective penalties | Satisfies requirement for granularity without sacrificing explainability |
| 4 | Certain | Default unknown change type to `feature` | Confirmed by user during `/fab-clarify`; keeps unclassified changes in a middle-risk tier |
| 5 | Confident | Default rollout mode to `legacy` for existing repositories | Prevents behavior regressions during upgrade and supports gradual adoption |
| 6 | Certain | Compute composite SRAD score via weighted mean: S=0.2, R=0.3, A=0.3, D=0.2 | Confirmed by user during `/fab-clarify`; weights prioritize Reversibility and Agent Competence per SRAD risk model |

6 assumptions tracked (3 certain, 3 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-14

- **Q**: Should we keep the initial dynamic `/fab-fff` thresholds as currently specified?
  **A**: Yes. Keep v1 thresholds as `bugfix=2.7`, `refactor=3.0`, `feature=3.3`, `architecture=3.6`.
- **Q**: How should composite SRAD value be computed from 0-100 dimension scores?
  **A**: Use weighted mean with `S/R/A/D = 0.2/0.3/0.3/0.2`.
- **Q**: For unclassifiable changes, should default type remain `feature`?
  **A**: Yes. Keep default as `feature`.
