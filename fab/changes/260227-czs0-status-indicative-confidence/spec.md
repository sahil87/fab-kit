# Spec: Status Indicative Confidence

**Change**: 260227-czs0-status-indicative-confidence
**Created**: 2026-02-27
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## calc-score.sh: Intake Gate Count Emission

### Requirement: Emit assumption counts in intake --check-gate output

`calc-score.sh --check-gate --stage intake` SHALL emit `certain`, `confident`, `tentative`, and `unresolved` count fields in its YAML output, in addition to the existing `gate`, `score`, `threshold`, and `change_type` fields.

The output format SHALL be:

```yaml
gate: pass
score: 4.2
threshold: 3.0
change_type: feat
certain: 3
confident: 1
tentative: 0
unresolved: 0
```

This requirement applies ONLY to the intake branch of `--check-gate` mode. The spec branch of `--check-gate` reads counts from `.status.yaml` where they are already persisted and SHALL continue to emit them as before.

#### Scenario: Intake gate with all-certain assumptions
- **GIVEN** a change with an `intake.md` containing 4 Certain and 1 Confident assumptions
- **WHEN** `calc-score.sh --check-gate --stage intake <change-dir>` is invoked
- **THEN** the output SHALL include `certain: 4`, `confident: 1`, `tentative: 0`, `unresolved: 0`
- **AND** the output SHALL include `gate: pass` and the computed `score`

#### Scenario: Intake gate with unresolved assumptions
- **GIVEN** a change with an `intake.md` containing 2 Certain and 1 Unresolved assumptions
- **WHEN** `calc-score.sh --check-gate --stage intake <change-dir>` is invoked
- **THEN** the output SHALL include `unresolved: 1`
- **AND** `score` SHALL be `0.0` and `gate` SHALL be `fail`

### Requirement: No side effects in --check-gate mode

`calc-score.sh --check-gate` SHALL NOT write to `.status.yaml` regardless of the `--stage` argument. This is already the case and MUST be preserved.

#### Scenario: Verify read-only behavior
- **GIVEN** a change at the intake stage
- **WHEN** `calc-score.sh --check-gate --stage intake <change-dir>` is invoked
- **THEN** `.status.yaml` SHALL NOT be modified (no timestamp update, no confidence block write)

### Requirement: Spec branch of --check-gate emits counts from .status.yaml

When `--check-gate` is invoked without `--stage intake` (i.e., spec gate), the script SHALL emit `certain`, `confident`, `tentative`, and `unresolved` fields read from the persisted confidence block in `.status.yaml`.

#### Scenario: Spec gate emits persisted counts
- **GIVEN** a change with `.status.yaml` containing `confidence: { certain: 6, confident: 2, tentative: 0, unresolved: 0, score: 4.4 }`
- **WHEN** `calc-score.sh --check-gate <change-dir>` is invoked (no `--stage` argument)
- **THEN** the output SHALL include `certain: 6`, `confident: 2`, `tentative: 0`, `unresolved: 0`

## fab-status: Stage-Aware Confidence Display

### Requirement: Indicative confidence at intake stage

When the active stage is `intake`, `/fab-status` SHALL compute the indicative confidence on the fly by running `calc-score.sh --check-gate --stage intake <change-dir>` and display:

```
Indicative confidence: {score} (fab-ff gate: {threshold}) — {total} assumptions ({N} certain, {N} confident, {N} tentative)
```

The `, {N} unresolved` suffix SHALL be appended ONLY when `unresolved > 0`.

#### Scenario: Intake stage with passing confidence
- **GIVEN** a change at the intake stage with 4 Certain and 1 Confident assumptions
- **WHEN** `/fab-status` is invoked
- **THEN** the confidence line SHALL display `Indicative confidence: 4.7 (fab-ff gate: 3.0) — 5 assumptions (4 certain, 1 confident, 0 tentative)`

#### Scenario: Intake stage with unresolved assumptions
- **GIVEN** a change at the intake stage with 2 Certain and 1 Unresolved assumptions
- **WHEN** `/fab-status` is invoked
- **THEN** the confidence line SHALL display `Indicative confidence: 0.0 (fab-ff gate: 3.0) — 3 assumptions (2 certain, 0 confident, 0 tentative, 1 unresolved)`

### Requirement: Persisted confidence at spec stage or later

When the active stage is `spec` or later and `.status.yaml` contains a confidence block, `/fab-status` SHALL read the persisted confidence and display:

```
Confidence: {score} of 5.0 ({N} certain, {N} confident, {N} tentative)
```

The `, {N} unresolved` suffix SHALL be appended ONLY when `unresolved > 0`.

#### Scenario: Spec stage with persisted confidence
- **GIVEN** a change at the spec stage with persisted confidence `{ score: 4.4, certain: 6, confident: 2, tentative: 0, unresolved: 0 }`
- **WHEN** `/fab-status` is invoked
- **THEN** the confidence line SHALL display `Confidence: 4.4 of 5.0 (6 certain, 2 confident, 0 tentative)`

### Requirement: Fallback for missing confidence data

When the active stage is NOT `intake` and `.status.yaml` has no confidence block (or score is `0.0` with all zero counts), `/fab-status` SHALL display `Confidence: not yet scored`.

#### Scenario: Spec stage with no confidence data
- **GIVEN** a change at the spec stage with no confidence block in `.status.yaml`
- **WHEN** `/fab-status` is invoked
- **THEN** the confidence line SHALL display `Confidence: not yet scored`

### Requirement: Skill files kept in sync

The behavior specified in `fab/.kit/skills/fab-status.md` SHALL be mirrored in `.claude/skills/fab-status/SKILL.md`. Both files MUST describe identical confidence display behavior.

#### Scenario: Both skill files match
- **GIVEN** the updated `fab-status.md` and `SKILL.md`
- **WHEN** a diff is performed between the two files
- **THEN** the content SHALL be identical (both files describe the same confidence display logic)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Indicative scores are ephemeral, not persisted | Confirmed from intake #1 — user explicitly stated this in discussion | S:95 R:90 A:95 D:95 |
| 2 | Certain | calc-score.sh --check-gate is the right mechanism | Confirmed from intake #2 — already read-only, no .status.yaml writes | S:90 R:95 A:90 D:90 |
| 3 | Certain | Display format includes gate threshold for indicative | Confirmed from intake #3 — user approved format | S:90 R:90 A:85 D:90 |
| 4 | Certain | Adding counts to --check-gate output is additive/non-breaking | Confirmed from intake #4 — existing consumers ignore extra YAML fields | S:85 R:95 A:90 D:95 |
| 5 | Confident | Spec branch of --check-gate reads counts from .status.yaml | Confirmed from intake #5 — counts already persisted, spec branch just emits them | S:80 R:90 A:85 D:85 |
| 6 | Certain | Both fab-status.md and SKILL.md must be updated in sync | Constitution requires portable .kit/; SKILL.md is the Claude Code wrapper | S:90 R:85 A:90 D:95 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
