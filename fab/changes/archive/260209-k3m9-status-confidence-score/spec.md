# Spec: Show confidence score in fab-status

**Change**: 260209-k3m9-status-confidence-score
**Created**: 2026-02-09
**Affected docs**: `fab/docs/fab-workflow/change-lifecycle.md`

## fab-workflow: Status Display

### Requirement: Confidence Score Parsing

`fab-status.sh` SHALL parse the `confidence` block from `.status.yaml`, extracting the `score`, `certain`, `confident`, `tentative`, and `unresolved` fields.

#### Scenario: Confidence block present

- **GIVEN** a `.status.yaml` with a `confidence` block containing `score: 4.2`, `certain: 3`, `confident: 1`, `tentative: 0`, `unresolved: 0`
- **WHEN** `fab-status.sh` is executed
- **THEN** the confidence fields are parsed and available for rendering

#### Scenario: Confidence block missing

- **GIVEN** a `.status.yaml` without a `confidence` block (legacy change)
- **WHEN** `fab-status.sh` is executed
- **THEN** the script treats confidence as "not yet scored" and does not error

### Requirement: Confidence Score Rendering

`fab-status.sh` SHALL render a `Confidence:` line in its output, placed after the `Checklist:` line and before the `Next:` line.

#### Scenario: Normal display (no unresolved)

- **GIVEN** confidence fields with `score: 4.2`, `certain: 3`, `confident: 1`, `tentative: 0`, `unresolved: 0`
- **WHEN** the output is rendered
- **THEN** the line reads: `Confidence: 4.2/5.0 (3 certain, 1 confident, 0 tentative)`

#### Scenario: Display with unresolved decisions

- **GIVEN** confidence fields with `score: 0.0`, `certain: 2`, `confident: 1`, `tentative: 0`, `unresolved: 1`
- **WHEN** the output is rendered
- **THEN** the line reads: `Confidence: 0.0/5.0 (2 certain, 1 confident, 0 tentative, 1 unresolved)`

#### Scenario: Confidence not yet scored

- **GIVEN** a `.status.yaml` without a `confidence` block
- **WHEN** the output is rendered
- **THEN** the line reads: `Confidence: not yet scored`

### Requirement: Skill Documentation Update

The `/fab-status` skill definition (`fab/.kit/skills/fab-status.md`) SHALL document the Confidence line in its output format description.

#### Scenario: Skill definition reflects new output

- **GIVEN** the updated `fab-status.md`
- **WHEN** a reader checks the output format section
- **THEN** the Confidence line is documented with all three display variants (normal, unresolved, not yet scored)

### Requirement: Centralized Doc Update

The centralized doc `fab/docs/fab-workflow/change-lifecycle.md` SHALL include the Confidence line in its `/fab-status` output description.

#### Scenario: Change lifecycle doc reflects new output

- **GIVEN** the updated `change-lifecycle.md`
- **WHEN** a reader checks the `/fab-status` section
- **THEN** the Confidence line is mentioned as part of the status display

## Deprecated Requirements

(none)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Place confidence line after Checklist, before Next | Logically groups "health of this change" info together |
| 2 | Confident | Show `unresolved` count only when > 0 | Keeps common case clean; surfaces the problem when it matters |

2 assumptions made (2 confident, 0 tentative).
