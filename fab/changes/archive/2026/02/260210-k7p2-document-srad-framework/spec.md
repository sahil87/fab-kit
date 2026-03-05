# Spec: Document the SRAD Framework

**Change**: 260210-k7p2-document-srad-framework
**Created**: 2026-02-10
**Affected docs**: `fab/specs/srad.md` (new), `fab/specs/index.md` (modified)

## Specs: SRAD Specification File

### Requirement: SRAD spec existence

A standalone specification file SHALL exist at `fab/specs/srad.md`. The file SHALL be self-contained — a reader MUST be able to understand the entire SRAD framework from this file alone, without needing to read `fab/.kit/skills/_context.md` or any other internal skill file.

#### Scenario: Reader discovers SRAD via specs index

- **GIVEN** a new contributor reading `fab/specs/index.md`
- **WHEN** they follow the link to `srad.md`
- **THEN** they find a complete explanation of the SRAD autonomy framework
- **AND** they can understand what the acronym stands for, how scoring works, and what the confidence gate does without consulting any other file

### Requirement: Acronym expansion

The spec SHALL define SRAD as an acronym for the four scoring dimensions: **S**ignal Strength, **R**eversibility, **A**gent Competence, **D**isambiguation Type. Each dimension SHALL have a one-sentence definition.

#### Scenario: Acronym lookup

- **GIVEN** a reader unfamiliar with SRAD
- **WHEN** they read the opening section of `fab/specs/srad.md`
- **THEN** they find the full expansion of S, R, A, D with a brief definition for each

### Requirement: Dimension evaluation criteria

Each of the four SRAD dimensions SHALL include a table or description showing what constitutes a "high" score (safe to assume) versus a "low" score (consider asking). The criteria MUST match the definitions in `fab/.kit/skills/_context.md`.

#### Scenario: Evaluating a decision point

- **GIVEN** a skill author who needs to evaluate a decision point against SRAD
- **WHEN** they consult the dimension evaluation criteria in `fab/specs/srad.md`
- **THEN** they find concrete guidance on what makes each dimension high or low
- **AND** the guidance matches what `_context.md` prescribes at runtime

### Requirement: Confidence grades

The spec SHALL define the four confidence grades: **Certain**, **Confident**, **Tentative**, **Unresolved**. For each grade, the spec SHALL document:

1. What it means (when to assign this grade)
2. What artifact marker is used (if any)
3. How it appears in output (visibility to the user)

#### Scenario: Grading a Tentative decision

- **GIVEN** a planning skill encountering a decision with a reasonable guess but multiple valid options
- **WHEN** the skill consults the confidence grades in `fab/specs/srad.md`
- **THEN** it finds that this maps to the Tentative grade
- **AND** it learns that Tentative decisions use `<!-- assumed: ... -->` markers and appear in the Assumptions summary

### Requirement: Confidence scoring formula

The spec SHALL document the confidence scoring formula exactly as implemented:

```
if unresolved > 0:
  score = 0.0
else:
  score = max(0.0, 5.0 - 0.1 * confident - 1.0 * tentative)
```

The spec SHALL explain: the 0.0–5.0 range, why Certain contributes 0 penalty, why Confident contributes 0.1, why Tentative contributes 1.0, and why any Unresolved produces a hard zero.

#### Scenario: Computing a score

- **GIVEN** a change with 10 Certain, 3 Confident, 1 Tentative, 0 Unresolved decisions
- **WHEN** the confidence formula is applied
- **THEN** the score is `max(0.0, 5.0 - 0.3 - 1.0) = 3.7`

#### Scenario: Unresolved hard zero

- **GIVEN** a change with 10 Certain, 2 Confident, 0 Tentative, 1 Unresolved decision
- **WHEN** the confidence formula is applied
- **THEN** the score is `0.0` regardless of other counts

#### Scenario: Floor clamping

- **GIVEN** a change with 5 Certain, 0 Confident, 6 Tentative, 0 Unresolved decisions
- **WHEN** the confidence formula is applied
- **THEN** the raw result is `5.0 - 0.0 - 6.0 = -1.0`
- **AND** the score is clamped to `0.0` by the `max(0.0, ...)` floor

### Requirement: Gate threshold

The spec SHALL document that `/fab-fff` requires `confidence.score >= 3.0` before executing the autonomous pipeline. The spec SHOULD explain what this threshold allows in practice (at most 2 Tentative decisions with some Confident erosion).

#### Scenario: Gate blocks low-confidence change

- **GIVEN** a change with confidence score 2.1
- **WHEN** the user runs `/fab-fff`
- **THEN** `/fab-fff` refuses to execute and reports the score is below the 3.0 threshold

#### Scenario: Gate allows exact threshold

- **GIVEN** a change with confidence score exactly 3.0
- **WHEN** the user runs `/fab-fff`
- **THEN** `/fab-fff` proceeds with the autonomous pipeline (threshold is inclusive)

#### Scenario: Gate allows high-confidence change

- **GIVEN** a change with confidence score 4.5
- **WHEN** the user runs `/fab-fff`
- **THEN** `/fab-fff` proceeds with the autonomous pipeline

### Requirement: Confidence lifecycle

The spec SHALL document which skills compute, recompute, and consume the confidence score:

| Event | Skill | Action |
|-------|-------|--------|
| Initial computation | `/fab-new` | Compute and write to `.status.yaml` |
| Initial computation | `/fab-discuss` | Compute and write to `.status.yaml` (same as `/fab-new`) |
| Recomputation | `/fab-continue`, `/fab-clarify` | Re-count across all artifacts, update `.status.yaml` |
| No recomputation | `/fab-ff`, `/fab-fff` | Use score from last manual step |
| Consumption | `/fab-fff` | Pre-flight gate check |

#### Scenario: Score recomputed after clarify

- **GIVEN** a change with 2 Tentative decisions and confidence score 3.0
- **WHEN** the user runs `/fab-clarify` and resolves 1 Tentative → Certain
- **THEN** the confidence score is recomputed to `max(0.0, 5.0 - 0.1*C - 1.0*1) = 4.0` (approximately)
- **AND** `.status.yaml` is updated with the new score

### Requirement: Critical Rule

The spec SHALL document the Critical Rule: Unresolved decisions with low Reversibility AND low Agent Competence MUST always be asked as questions — even when the question budget is otherwise exhausted. The spec SHALL explain that `/fab-clarify` is NOT a valid escape valve for Unresolved decisions (it is for Tentative ones).

#### Scenario: Critical Rule enforcement

- **GIVEN** a planning skill has already asked 3 questions (budget exhausted)
- **AND** a 4th decision point scores Unresolved with low R and low A
- **WHEN** the skill evaluates this decision
- **THEN** the decision MUST still be asked as a question (Critical Rule overrides budget)

### Requirement: Worked examples

The spec SHALL include at least two worked examples at the proposal level:

1. **High-ambiguity proposal** — a vague or underspecified description that produces many Tentative/Unresolved decisions and a low confidence score (near 0.0)
2. **Low-ambiguity proposal** — a detailed, well-specified description that produces mostly Certain/Confident decisions and a high confidence score (near 5.0)

Each example SHALL show: the input description, the SRAD evaluation of 2-3 representative decisions, the resulting confidence counts, and the computed score.

#### Scenario: Reader understands high vs low ambiguity

- **GIVEN** a reader unfamiliar with confidence scoring
- **WHEN** they read the two worked examples
- **THEN** they understand what makes a proposal high-ambiguity vs low-ambiguity
- **AND** they can see how SRAD grades aggregate into the confidence score

### Requirement: Skill-specific autonomy levels

The spec SHALL include the skill-specific autonomy table documenting how SRAD manifests across different skills (`fab-discuss`, `fab-new`, `fab-continue`, `fab-ff`, `fab-fff`). This SHOULD cover: posture, interruption budget, output format, and escape valve for each skill.
#### Scenario: Skill author checks interruption budget

- **GIVEN** a contributor writing a new planning skill
- **WHEN** they check the skill-specific autonomy table
- **THEN** they find guidance on how many questions their skill can ask and what output format to use

## Specs: Index Update

### Requirement: Specs index entry

`fab/specs/index.md` SHALL include an entry for `srad.md` in the specs table with a description matching the document's purpose (e.g., "SRAD autonomy framework — scoring dimensions, confidence grades, gating, worked examples").

#### Scenario: Index lists new spec

- **GIVEN** the current `fab/specs/index.md`
- **WHEN** this change is applied
- **THEN** the index table contains a row for `[srad](srad.md)` with an appropriate description

## Deprecated Requirements

(none)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Include skill-specific autonomy table in the spec | Shows how SRAD operates differently per skill, essential for holistic understanding of the framework |
| 2 | Confident | Self-contained document rather than referencing `_context.md` | The whole point of this change is making SRAD accessible; requiring readers to cross-reference internal skill files defeats the purpose |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-10

- **Q**: Confidence lifecycle table is missing `/fab-discuss`, which also recomputes confidence. Should it be added?
  **A**: Yes — added `/fab-discuss` as a separate initial computation row in the lifecycle table.
- **Q**: Should a boundary scenario be added for a score of exactly 3.0 to make the `>=` gate unambiguous?
  **A**: Accepted recommendation — added scenario showing 3.0 passes the gate (inclusive).
- **Q**: Should a floor-clamping scenario be added showing `max(0.0, ...)` in action when penalties exceed 5.0?
  **A**: Accepted recommendation — added scenario with 6 Tentative decisions clamping to 0.0.
