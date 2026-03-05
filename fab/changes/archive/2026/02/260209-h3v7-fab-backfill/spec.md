# Spec: Add `/fab-backfill` Command

**Change**: 260209-h3v7-fab-backfill
**Created**: 2026-02-09
**Affected docs**: `fab/docs/fab-workflow/specs-index.md` (modified), `fab/docs/fab-workflow/backfill.md` (new)

## Fab Workflow: `/fab-backfill` Skill

### Requirement: Gap Detection SHALL Cross-Reference Docs and Specs at Section Level

`/fab-backfill` SHALL read all doc domains via `fab/docs/index.md`, load every doc file, and build a topic inventory from headings/sections. It SHALL then read all spec files via `fab/specs/index.md` and build a spec coverage inventory. It SHALL cross-reference these inventories to find structural gaps — topics that docs cover but specs do not mention at all (not even within a section of a broader spec file).

#### Scenario: Doc topic has no spec coverage
- **GIVEN** `fab/docs/fab-workflow/preflight.md` documents preflight behavior with requirements and scenarios
- **WHEN** no spec file in `fab/specs/` mentions preflight (neither as a file nor within a section)
- **THEN** preflight is identified as a structural gap

#### Scenario: Doc topic has partial spec coverage
- **GIVEN** `fab/docs/fab-workflow/planning-skills.md` documents `/fab-discuss` behavior
- **AND** `fab/specs/skills.md` mentions `/fab-discuss` in a section
- **WHEN** the cross-reference runs
- **THEN** `/fab-discuss` is NOT identified as a gap (coverage exists, even if less detailed)

### Requirement: Output SHALL Be Capped at Top 3 Gaps

The command SHALL present at most 3 gaps, ranked by impact. Impact ranking SHOULD prioritize topics that affect how humans understand the system — core behavioral rules and design decisions over implementation minutiae.

#### Scenario: More than 3 gaps exist
- **GIVEN** 7 doc topics have no spec coverage
- **WHEN** `/fab-backfill` runs
- **THEN** only the top 3 are presented, with a note: "{N} additional gaps found but not shown"

#### Scenario: Fewer than 3 gaps exist
- **GIVEN** 1 doc topic has no spec coverage
- **WHEN** `/fab-backfill` runs
- **THEN** only that 1 gap is presented

#### Scenario: No gaps exist
- **GIVEN** all doc topics have corresponding spec coverage
- **WHEN** `/fab-backfill` runs
- **THEN** output: "No structural gaps found between docs and specs."

### Requirement: Each Gap SHALL Include Exact Markdown Preview

For each gap presented, the command SHALL show:
1. The source doc file and the topic heading identified as uncovered
2. The target spec file where the addition would go
3. The exact markdown that would be inserted

The markdown preview MUST be concise and match the existing spec tone — short declarative statements, no verbose explanations. The user reviews the exact text before any write happens.

#### Scenario: User reviews a gap
- **GIVEN** a gap is identified for "preflight" in `fab/docs/fab-workflow/preflight.md`
- **WHEN** the gap is presented to the user
- **THEN** the output shows the source doc, the target spec file (e.g., `fab/specs/architecture.md`), and the exact markdown snippet to insert

### Requirement: Interactive Per-Gap Confirmation

For each gap, the user SHALL be asked to confirm, reject, or skip. Only confirmed additions are written to the spec file. The command SHALL present gaps one at a time, waiting for user input before proceeding to the next.

#### Scenario: User confirms a gap
- **GIVEN** the user is shown a gap with its markdown preview
- **WHEN** the user confirms (yes)
- **THEN** the markdown is inserted into the target spec file at the appropriate location

#### Scenario: User rejects a gap
- **GIVEN** the user is shown a gap with its markdown preview
- **WHEN** the user rejects (no)
- **THEN** no change is made for that gap, and the next gap is presented

#### Scenario: User skips remaining gaps
- **GIVEN** the user has reviewed 1 of 3 gaps
- **WHEN** the user says "done" or "skip rest"
- **THEN** remaining gaps are not presented and no further changes are made

### Requirement: No Active Change Required

`/fab-backfill` SHALL NOT require `fab/current` to be set. It operates on the project-level `fab/docs/` and `fab/specs/` directories directly, not on a change folder. It SHALL NOT modify `fab/current` or create git branches.

#### Scenario: No active change set
- **GIVEN** `fab/current` does not exist or is empty
- **WHEN** `/fab-backfill` is invoked
- **THEN** the command runs normally

### Requirement: Pre-flight Checks

The command SHALL verify that `fab/docs/index.md` and `fab/specs/index.md` both exist before proceeding. If either is missing, abort with guidance.

#### Scenario: Docs index missing
- **GIVEN** `fab/docs/index.md` does not exist
- **WHEN** `/fab-backfill` is invoked
- **THEN** output: "fab/docs/index.md not found. Run /fab-init first."

#### Scenario: Specs index missing
- **GIVEN** `fab/specs/index.md` does not exist
- **WHEN** `/fab-backfill` is invoked
- **THEN** output: "fab/specs/index.md not found. Run /fab-init first."

## Fab Workflow: Specs Index Update

### Requirement: Specs-Index Doc SHALL Reference `/fab-backfill`

The "Human-Curated Ownership" section in `fab/docs/fab-workflow/specs-index.md` SHALL be updated to replace the "reverse-hydration is a future consideration" note with a reference to `/fab-backfill` as the assisted (not automated) reverse-hydration mechanism.

#### Scenario: Updated docs reflect backfill existence
- **GIVEN** `/fab-backfill` skill has been implemented
- **WHEN** a user reads `fab/docs/fab-workflow/specs-index.md`
- **THEN** the Human-Curated Ownership section mentions `/fab-backfill` as the way to identify and selectively backfill gaps from docs to specs

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Utility command pattern — no `fab/current`, no git | Follows `/fab-status` precedent |
| 2 | Confident | Reads all doc and spec files for cross-referencing | File count is small enough; necessary for section-level matching |

2 assumptions made (2 confident, 0 tentative).
