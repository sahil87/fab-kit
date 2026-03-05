# Spec: Scoring Formula Produces Inflated Scores

**Change**: 260212-29xv-scoring-formula
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`, `fab/docs/fab-workflow/clarify.md`

## Non-Goals

- Changing the SRAD grading criteria or dimension definitions — the framework is conceptually sound; only the formula and recomputation mechanics change
- Changing how `/fab-ff` or `/fab-fff` consume the confidence score (pipeline gate logic stays the same)
- Adding new confidence grade levels beyond the existing four (Certain, Confident, Tentative, Unresolved)

## Empirical Analysis: Historical Score Distribution

Analysis of 26 archived changes with confidence data reveals the formula is inflated:

| Metric | Value |
|--------|-------|
| Score range (actual) | 4.1 – 5.0 |
| Tentative decisions (across all 26) | 0 |
| Unresolved decisions (across all 26) | 0 |
| Confident decisions per change | 0 – 9 |
| Certain decisions per change | 0 – 14 |
| Changes scoring >= 4.5 | 15 / 26 (58%) |
| Changes scoring >= 4.0 | 26 / 26 (100%) |

**Root cause**: The Confident penalty of 0.1 is nearly meaningless. Even 9 Confident decisions (the observed maximum) only reduce the score by 0.9 — from 5.0 to 4.1. The formula provides no meaningful differentiation between a change with 11 Certain/0 Confident and one with 0 Certain/6 Confident.

**Secondary issue**: `/fab-clarify` specifies confidence recomputation (Step 7 in the skill file), but the recount mechanism has no defined way to reclassify resolved assumptions. When a `<!-- assumed: ... -->` marker is replaced with `<!-- clarified: ... -->`, the Assumptions table grade isn't updated, so the recount produces the same totals.

## Confidence Formula: Revised Penalty Weights

### Requirement: Increase Confident Penalty

The confidence formula SHALL use a penalty of **0.3** per Confident decision (increased from 0.1):

```
if unresolved > 0:
  score = 0.0
else:
  score = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)
```

The Tentative penalty (1.0), Unresolved hard-zero, and Certain penalty (0.0) SHALL remain unchanged.
<!-- assumed: 0.3 penalty chosen over 0.2 or 0.5 — 0.3 produces scores in the 2.3–5.0 range for historical data, giving meaningful differentiation while keeping the formula simple -->

#### Scenario: All-Certain Change
- **GIVEN** a change with 11 Certain and 0 Confident decisions
- **WHEN** the confidence score is computed
- **THEN** the score SHALL be 5.0

#### Scenario: Moderate-Confident Change
- **GIVEN** a change with 3 Certain and 4 Confident decisions
- **WHEN** the confidence score is computed
- **THEN** the score SHALL be 3.8

#### Scenario: High-Confident Change
- **GIVEN** a change with 8 Certain and 9 Confident decisions
- **WHEN** the confidence score is computed
- **THEN** the score SHALL be 2.3

#### Scenario: Mixed with Tentative
- **GIVEN** a change with 5 Certain, 2 Confident, and 1 Tentative decision
- **WHEN** the confidence score is computed
- **THEN** the score SHALL be 3.4

### Requirement: Revised Penalty Weight Documentation

The penalty weight table in all documentation locations SHALL be updated to reflect:

| Grade | Penalty | Rationale |
|-------|---------|-----------|
| **Certain** | 0.0 | Deterministic — no ambiguity |
| **Confident** | 0.3 | Moderate — strong signal but still an assumption; accumulates meaningfully |
| **Tentative** | 1.0 | Significant — reasonable guess but could be wrong |
| **Unresolved** | Hard zero | Cannot run autonomously with unresolved decisions |

#### Scenario: Documentation Consistency
- **GIVEN** the formula is updated in `_context.md`
- **WHEN** a reader checks `fab/design/srad.md`, `fab/docs/fab-workflow/planning-skills.md`, or `_context.md`
- **THEN** all three locations SHALL show the same formula with 0.3 Confident penalty

## Confidence Formula: Gate Threshold

### Requirement: Keep Gate at 3.0

The `/fab-fff` gate threshold SHALL remain at **3.0**.
<!-- assumed: Keeping gate at 3.0 rather than lowering it — higher-Confident changes should benefit from clarification before autonomous execution -->

With the revised formula, the 3.0 gate allows:

| Configuration | Score | Gate |
|---------------|-------|------|
| 0 Tentative, 0-6 Confident | 3.2 – 5.0 | Pass |
| 0 Tentative, 7+ Confident | < 3.0 | Fail |
| 1 Tentative, 0-3 Confident | 3.1 – 4.0 | Pass |
| 1 Tentative, 4+ Confident | < 3.0 | Fail |
| 2 Tentative, 0 Confident | 3.0 | Pass (barely) |
| 2 Tentative, 1+ Confident | < 3.0 | Fail |
| 3+ Tentative | < 2.0 | Fail |

#### Scenario: Gate Passes with Moderate Confidence
- **GIVEN** a change with 5 Certain and 5 Confident decisions (score 3.5)
- **WHEN** the user runs `/fab-fff`
- **THEN** the gate check SHALL pass

#### Scenario: Gate Fails with High Assumptions
- **GIVEN** a change with 4 Certain and 8 Confident decisions (score 2.6)
- **WHEN** the user runs `/fab-fff`
- **THEN** the gate check SHALL fail with a message suggesting `/fab-clarify`

## Clarify: Grade Reclassification on Resolution

### Requirement: Update Assumptions Table on Resolution

When `/fab-clarify` resolves a question (in suggest mode), it SHALL update the corresponding entry in the artifact's `## Assumptions` table:

- **Tentative → Certain**: When the user confirms or clarifies a Tentative assumption, the grade in the Assumptions table SHALL change to `Certain` (the decision is now deterministic — the user explicitly confirmed it)
- **Confident → Certain**: When the user confirms a Confident assumption during clarification, the grade SHALL change to `Certain`

The inline marker update (`<!-- assumed: -->` → `<!-- clarified: -->`) already occurs. This requirement adds the Assumptions table update to ensure recomputation reflects the change.

#### Scenario: Tentative Resolved to Certain
- **GIVEN** a spec with 3 Certain, 2 Confident, 2 Tentative decisions (score 1.4)
- **AND** the user runs `/fab-clarify` and confirms both Tentative assumptions
- **WHEN** the skill recomputes the confidence score
- **THEN** the Assumptions table SHALL show both entries as `Certain`
- **AND** the new counts SHALL be 5 Certain, 2 Confident, 0 Tentative
- **AND** the new score SHALL be 4.4

#### Scenario: Confident Confirmed to Certain
- **GIVEN** a spec with 2 Certain, 4 Confident, 0 Tentative decisions (score 3.8)
- **AND** the user runs `/fab-clarify` and confirms 2 of the Confident assumptions
- **WHEN** the skill recomputes the confidence score
- **THEN** the confirmed entries SHALL change to `Certain` in the Assumptions table
- **AND** the new counts SHALL be 4 Certain, 2 Confident, 0 Tentative
- **AND** the new score SHALL be 4.4

#### Scenario: Score Increases After Clarification
- **GIVEN** any change with a confidence score of X
- **AND** the user runs `/fab-clarify` and resolves at least one assumption
- **WHEN** the skill recomputes the confidence score
- **THEN** the new score SHALL be >= X (resolving assumptions can only increase or maintain the score, never decrease it)

### Requirement: Recount Mechanism

The confidence recomputation in `/fab-clarify` Step 7 SHALL recount grades by scanning the `## Assumptions` section of each artifact in the change (brief.md, spec.md, tasks.md — whichever exist). The count is derived from the `Grade` column values in the Assumptions tables, not from inline markers.

#### Scenario: Recount Across Multiple Artifacts
- **GIVEN** a change with brief.md (2 Confident in Assumptions table) and spec.md (1 Tentative, 3 Confident in Assumptions table)
- **WHEN** the confidence score is recomputed
- **THEN** the totals SHALL be: 0 Certain (from Assumptions), 5 Confident, 1 Tentative
- **AND** certain count SHALL include all decisions NOT listed in any Assumptions table (derived from total decision count minus Confident/Tentative/Unresolved)

[NEEDS CLARIFICATION] — The mechanism for counting Certain decisions needs clarification: Certain decisions are omitted from Assumptions tables by convention. The recount either needs a separate count source or should derive Certain count from a known total. Currently, the `certain` count in `.status.yaml` is set by the generating skill and not re-derivable from artifacts alone.

### Requirement: Clarify Recomputation Already Specified

The confidence recomputation behavior after suggest-mode sessions is already specified in the `/fab-clarify` skill (Step 7) and the centralized doc (`clarify.md`). This change does NOT add a new recomputation step — it fixes the grade reclassification mechanism that makes the existing recomputation effective.

#### Scenario: Existing Recomputation Step
- **GIVEN** the `/fab-clarify` skill file at `fab/.kit/skills/fab-clarify.md`
- **WHEN** Step 7 executes after a suggest-mode session
- **THEN** it SHALL recount grades (using the updated Assumptions tables) and write the new `confidence` block to `.status.yaml`

## Documentation Updates

### Requirement: Update _context.md Confidence Scoring

The confidence formula in `fab/.kit/skills/_context.md` (Confidence Scoring section) SHALL be updated to show the revised formula with 0.3 Confident penalty.

#### Scenario: Context File Updated
- **GIVEN** the `_context.md` file
- **WHEN** a skill reads the Confidence Scoring section
- **THEN** it SHALL see `score = max(0.0, 5.0 - 0.3 * confident - 1.0 * tentative)`

### Requirement: Update srad.md Design Spec

The confidence formula in `fab/design/srad.md` SHALL be updated to reflect the 0.3 penalty, updated penalty weight table, and revised "What 3.0 Allows" section.

#### Scenario: Design Spec Updated
- **GIVEN** the `fab/design/srad.md` file
- **WHEN** a reader checks the Formula and Gate Threshold sections
- **THEN** the formula, penalty table, and gate examples SHALL all reflect the 0.3 Confident penalty

### Requirement: Update Centralized Docs

The affected centralized docs SHALL be updated:
- `fab/docs/fab-workflow/planning-skills.md` — confidence scoring formula references
- `fab/docs/fab-workflow/clarify.md` — add grade reclassification behavior to the Confidence Recomputation section

#### Scenario: Planning Skills Doc Updated
- **GIVEN** `fab/docs/fab-workflow/planning-skills.md`
- **WHEN** a reader checks confidence scoring references
- **THEN** the formula SHALL show 0.3 Confident penalty

#### Scenario: Clarify Doc Updated
- **GIVEN** `fab/docs/fab-workflow/clarify.md`
- **WHEN** a reader checks the Confidence Recomputation section
- **THEN** it SHALL describe grade reclassification: Tentative/Confident → Certain on resolution

## Deprecated Requirements

### Confident Penalty of 0.1
**Reason**: Empirical analysis of 26 archived changes shows the 0.1 penalty provides no meaningful signal — all scores cluster in the 4.1–5.0 range regardless of assumption count.
**Migration**: Replaced by 0.3 penalty in the same formula structure.

## Design Decisions

1. **Penalty increase (0.3) over ratio-based formula**: Chose to increase the Confident penalty from 0.1 to 0.3 rather than switching to a ratio-based formula (e.g., `confident / total_decisions`).
   - *Why*: The penalty-based formula is simpler, well-understood by users, and the brief explicitly favors formula adjustment over structural changes. 0.3 was derived from modeling the penalty against all 26 historical changes to produce a meaningful score range (2.3–5.0).
   - *Rejected*: Ratio-based scoring — more nuanced but adds complexity, changes the mental model, and isn't motivated by the evidence (the issue is penalty magnitude, not formula structure).

2. **Grade reclassification in Assumptions table over separate tracking**: Resolved assumptions are reclassified in-place in the artifact's Assumptions table rather than tracked in a separate structure.
   - *Why*: Keeps the source of truth co-located with the artifact. The recount reads the Assumptions table directly, so updating it in-place makes the recount naturally correct.
   - *Rejected*: Separate resolution tracking file — adds complexity, risks drift between the table and the tracker.

3. **Keep gate at 3.0 rather than lowering**: The `/fab-fff` gate stays at 3.0 even though the new formula produces lower scores for high-Confident changes.
   - *Why*: Changes with 7+ Confident decisions benefit from clarification before autonomous execution. Lowering the gate would negate the value of the penalty increase.
   - *Rejected*: Lowering gate to 2.5 — would allow the same changes through that the old formula allowed, defeating the purpose.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | 0.3 penalty chosen (not 0.2 or 0.5) | Empirical modeling against 26 historical changes shows 0.3 produces meaningful range (2.3–5.0) while keeping formula simple |
| 2 | Confident | Penalty-based formula over ratio-based | Brief explicitly says "formula adjustment will be the primary fix"; penalty structure is well-understood |
| 3 | Confident | Resolved grades become Certain (not just removed) | Logically follows — user confirmation eliminates ambiguity, making the decision deterministic |
| 4 | Tentative | Keep gate threshold at 3.0 | More high-Confident changes will fail the gate; user may prefer lowering to 2.5 to maintain current pass rate |
| 5 | Tentative | Certain count derivation from Assumptions tables | Current convention omits Certain from tables; recount mechanism needs clarification on how to derive this count |

5 assumptions made (3 confident, 2 tentative). Run /fab-clarify to review.
