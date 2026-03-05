# Brief: Scoring Formula Produces Inflated Scores

**Change**: 260212-29xv-scoring-formula
**Created**: 2026-02-12
**Status**: Draft

## Origin

**Backlog**: [29xv]
**Linear**: DEV-994
**User requested**: `/fab-new 29xv`

> Scoring formula needs to be relooked at — scores are generally too high. Check historical scores from archive. Do an analysis of the reason, the methodology, and the best way get relevant scores that give a strong signal. Also feels strange: Scores don't change after clarify - clarification should ideally increase score.

## Why

The current confidence scoring formula produces inflated scores that don't provide a strong signal for whether a change is well-understood. High scores are being assigned to changes that may still have ambiguity. Additionally, running `/fab-clarify` doesn't update the confidence score, even though clarification resolves uncertainties and should increase confidence.

This undermines the confidence gate for `/fab-fff` (requires score >= 3.0) and reduces trust in the scoring system as a quality signal.

## What Changes

- Analyze historical confidence scores from archived changes to identify patterns and establish baseline expectations
- Evaluate the current scoring formula (`score = max(0.0, 5.0 - 0.1 * confident - 1.0 * tentative)`) against empirical data
- Propose a revised formula that produces more meaningful scores with better signal strength
<!-- assumed: Formula adjustment will be the primary fix, rather than changing SRAD grading criteria — formula is more directly tunable -->
- Update `/fab-clarify` to recompute and update confidence scores after resolution sessions
- Document the new scoring methodology and expected score distributions

## Affected Docs

### New Docs
- None

### Modified Docs
- `fab-workflow/clarify`: Add confidence score recomputation behavior
- `fab-workflow/planning-skills`: Update confidence scoring formula documentation

### Removed Docs
- None

## Impact

**Affected Code**:
- `/fab-clarify` skill — needs to recompute confidence after suggest-mode sessions
- Confidence scoring logic (likely in `/fab-new`, `/fab-continue`, or shared context)
- Documentation of the SRAD framework and confidence formula

**Affected Workflows**:
- Changes confidence gate threshold or formula for `/fab-fff`
- Improves trust in confidence scores across all planning skills
- Makes `/fab-clarify` more impactful by reflecting reduced ambiguity

## Open Questions

- [DEFERRED] Should the formula penalize Confident decisions more heavily (current: 0.1 per decision)?
<!-- assumed: Analysis of archive data will inform this — deferring until we see empirical score distributions -->
- [DEFERRED] What is the target score distribution — should most changes score 2-4 instead of 3-5?
<!-- assumed: Will determine target range based on archive analysis and user feedback on what scores feel "right" -->

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Analyze historical archive data | Backlog explicitly mentions "check historical scores from archive" |
| 2 | Confident | `/fab-clarify` should update confidence scores | Backlog explicitly states "Scores don't change after clarify - clarification should ideally increase score" |
| 3 | Tentative | Formula adjustment over SRAD criteria changes | Formula is more directly tunable; SRAD framework is conceptually sound |
| 4 | Tentative | Defer target score range until after analysis | Need empirical data to determine what "inflated" means quantitatively |

4 assumptions made (2 confident, 2 tentative). Run /fab-clarify to review.
