# Spec: Clarify Tentative First

**Change**: 260416-hyl6-clarify-tentative-first
**Created**: 2026-04-16
**Affected memory**: `docs/memory/fab-workflow/clarify.md`

## Clarify Skill: Suggest Mode Step Ordering

### Requirement: Taxonomy Scan Before Bulk Confirm

The `/fab-clarify` suggest mode flow SHALL run the taxonomy scan (detecting `<!-- assumed: -->` markers and other gaps) **before** the bulk confirm flow for Confident assumptions.

The revised step order SHALL be:

1. **Step 1**: Read Target Artifact
2. **Step 1.5**: Taxonomy Scan — scan for gaps, `[NEEDS CLARIFICATION]`, and `<!-- assumed: -->` markers. Build prioritized question queue. Present tentative assumption questions first (from `<!-- assumed: -->` markers), one at a time per existing behavior
3. **Step 2**: Bulk Confirm — if detection conditions are met (`confident >= 3` AND `confident > tentative + unresolved`), present Confident assumptions for bulk confirmation. Counts are evaluated **after** Step 1.5 resolution (tentative items resolved in Step 1.5 no longer count as tentative)
4. **Steps 3–8**: Remaining taxonomy questions (non-tentative gaps from Step 1.5 queue), updates, audit trail, summary, confidence recomputation

#### Scenario: Tentative markers addressed before bulk confirm
- **GIVEN** an artifact with 2 Tentative assumptions (with `<!-- assumed: -->` markers) and 4 Confident assumptions
- **WHEN** the user runs `/fab-clarify`
- **THEN** the taxonomy scan runs first, detecting the 2 `<!-- assumed: -->` markers
- **AND** the tentative markers are presented as questions before bulk confirm triggers
- **AND** after tentative resolution, bulk confirm evaluates remaining counts and triggers if conditions still hold

#### Scenario: Bulk confirm sees updated counts after tentative resolution
- **GIVEN** an artifact with 3 Tentative and 3 Confident assumptions
- **WHEN** the user runs `/fab-clarify` and resolves all 3 Tentative assumptions (upgrading them to Certain)
- **THEN** bulk confirm evaluates counts as: confident=3, tentative=0, unresolved=0
- **AND** bulk confirm triggers (3 >= 3 AND 3 > 0+0)

#### Scenario: Zero tentative, bulk confirm still works
- **GIVEN** an artifact with 0 Tentative and 5 Confident assumptions
- **WHEN** the user runs `/fab-clarify`
- **THEN** the taxonomy scan runs first but finds no `<!-- assumed: -->` markers
- **AND** bulk confirm triggers immediately after (5 >= 3 AND 5 > 0+0)
- **AND** behavior is functionally identical to the previous ordering

### Requirement: Question Budget Spans Both Phases

The 5-question cap per invocation SHALL apply across both the tentative resolution phase (Step 1.5) and the remaining taxonomy questions (Steps 3+). Tentative questions resolved in Step 1.5 count toward the cap.

Bulk confirm does NOT count toward the 5-question cap — it is a batch operation, not individual questions.

#### Scenario: Questions consumed by tentative resolution
- **GIVEN** an artifact with 4 Tentative assumptions and 3 remaining taxonomy gaps
- **WHEN** the user runs `/fab-clarify` and answers all 4 tentative questions
- **THEN** 1 remaining taxonomy question is presented (4 + 1 = 5 cap)
- **AND** 2 taxonomy gaps are reported as Outstanding in the coverage summary

#### Scenario: Bulk confirm does not consume question budget
- **GIVEN** an artifact with 2 Tentative and 4 Confident assumptions plus 3 taxonomy gaps
- **WHEN** the user resolves 2 tentative questions, then bulk-confirms 4 confident assumptions
- **THEN** 3 remaining taxonomy questions are still available (2 tentative consumed from budget, bulk confirm is free, 3 remaining = 5 total)

### Requirement: Bulk Confirm Context Note Reversal

The note in Step 2 (formerly Step 1.5) SHALL state that bulk confirm runs on the artifact already updated by tentative resolution in Step 1.5. This replaces the previous note which stated the taxonomy scan ran on the artifact updated by bulk confirm.

#### Scenario: Bulk confirm sees artifact updated by tentative resolution
- **GIVEN** an artifact where Step 1.5 resolved a Tentative assumption by updating the artifact text and upgrading the grade to Certain
- **WHEN** Step 2 (Bulk Confirm) runs
- **THEN** it operates on the already-updated artifact
- **AND** the resolved tentative assumption is no longer counted in detection conditions

### Requirement: Auto Mode Unaffected

Auto mode SHALL remain unchanged. Auto mode already skips bulk confirm and runs the taxonomy scan autonomously. The reordering applies only to suggest mode.

#### Scenario: Auto mode behavior unchanged
- **GIVEN** `/fab-clarify` is invoked with `[AUTO-MODE]` prefix
- **WHEN** the skill runs
- **THEN** it performs autonomous gap resolution (scan + resolve) without bulk confirm
- **AND** the behavior is identical to pre-change auto mode

## Preamble Update: Bulk Confirm Reference

### Requirement: Update Preamble Bulk Confirm Description

The `_preamble.md` Bulk Confirm subsection (under Confidence Scoring) SHALL be updated to reflect that bulk confirm runs **after** the taxonomy scan, not before it. The sentence "This flow runs as Step 1.5 in Suggest Mode, before the standard taxonomy scan (Step 2)" SHALL be replaced with "This flow runs as Step 2 in Suggest Mode, after the taxonomy scan and tentative resolution (Step 1.5)." <!-- clarified: auto-resolved — exact replacement text specified for implementability -->

#### Scenario: Preamble reflects new ordering
- **GIVEN** a reader consulting `_preamble.md` for bulk confirm behavior
- **WHEN** they read the Bulk Confirm subsection
- **THEN** it describes bulk confirm running after the taxonomy scan (Step 2), not before it (Step 1.5)

## Spec File Update: SPEC-fab-clarify.md

### Requirement: Update Spec Flow Diagram

The `docs/specs/skills/SPEC-fab-clarify.md` flow diagram SHALL be updated to reflect the new step ordering. The suggest mode flow SHALL show Step 1.5 as the taxonomy scan and Step 2 as bulk confirm, reversing the current order. <!-- clarified: auto-resolved from cross-reference — SPEC-fab-clarify.md lines 24/28 document the step ordering that this change reverses -->

## Memory File Update: clarify.md

### Requirement: Update Memory Ordering Text

The `docs/memory/fab-workflow/clarify.md` Bulk Confirm section SHALL be updated to reflect that bulk confirm runs **after** the taxonomy scan, not before. The text "suggest mode SHALL offer a bulk confirm flow before the taxonomy scan (Step 1.5)" SHALL be replaced to indicate bulk confirm runs as Step 2, after the taxonomy scan at Step 1.5. The text "After bulk confirm completes, proceed to Step 2 (taxonomy scan) on the updated artifact" SHALL be replaced to indicate that after bulk confirm completes, remaining taxonomy questions (from Step 1.5's queue) are presented. <!-- clarified: auto-resolved — intake explicitly lists this file as affected, and the memory file contains the exact pre-change ordering text at lines 88 and 106 -->

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Taxonomy scan and tentative resolution run before bulk confirm | Confirmed from intake #1 — directly stated in user request | S:95 R:85 A:90 D:90 |
| 2 | Certain | Step renumbering: taxonomy scan becomes Step 1.5, bulk confirm becomes Step 2 | Confirmed from intake #2 — logical consequence of reorder | S:90 R:90 A:95 D:95 |
| 3 | Certain | 5-question cap unchanged | Confirmed from intake #3 — user only asked to reorder | S:85 R:90 A:95 D:95 |
| 4 | Certain | Auto mode unaffected | Confirmed from intake #4 — auto mode has no bulk confirm | S:90 R:95 A:95 D:95 |
| 5 | Certain | Bulk confirm context note flips to reference tentative resolution preceding it | Confirmed from intake #5 — upgraded from Confident, logical consequence | S:90 R:85 A:85 D:85 |
| 6 | Certain | Taxonomy scan runs in full before presenting questions; tentative markers presented first from the queue | Confirmed from intake #6 — upgraded from Confident, preserves scan-then-ask pattern | S:85 R:80 A:80 D:80 |
| 7 | Certain | Bulk confirm detection conditions evaluated after tentative resolution, using updated counts | <!-- clarified: auto-resolved — explicitly stated in spec requirement and scenario "Bulk confirm sees updated counts after tentative resolution" --> | S:95 R:80 A:85 D:70 |
| 8 | Certain | Bulk confirm does not count toward the 5-question cap | <!-- clarified: auto-resolved — explicitly stated in spec requirement "Question Budget Spans Both Phases" and scenario "Bulk confirm does not consume question budget" --> | S:95 R:85 A:80 D:70 |
| 9 | Certain | SPEC-fab-clarify.md flow diagram must be updated to reflect new step ordering | <!-- clarified: auto-resolved — cross-reference gap; file documents the step order at lines 24/28 --> | S:90 R:90 A:90 D:90 |
| 10 | Certain | Memory file clarify.md Bulk Confirm section text must be updated to match new ordering | <!-- clarified: auto-resolved — intake lists this file as affected; memory contains pre-change text --> | S:90 R:90 A:90 D:90 |

10 assumptions (10 certain, 0 confident, 0 tentative, 0 unresolved).
