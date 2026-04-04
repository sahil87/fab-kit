# Spec: Remove Old Operator Skills

**Change**: 260331-eeso-remove-old-operator-skills
**Created**: 2026-03-31
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/index.md`

## Non-Goals

- Modifying `fab-operator7.md`, `fab-operator7.sh`, or `.claude/skills/fab-operator7/` — these are the current operator and remain untouched
- Modifying deployed copies in `.claude/skills/` — user explicitly excluded these from scope
- Performing a broad cleanup of operator4 references — operator4 remains documented as historical context in memory files; only the specific `fab-operator4.sh` references explicitly called out in this spec (e.g., in kit-architecture documentation) may be updated

## Skill Files: Delete Old Operator Sources

### Requirement: Remove operator5 and operator6 source skills

The files `fab/.kit/skills/fab-operator5.md` and `fab/.kit/skills/fab-operator6.md` SHALL be deleted.

#### Scenario: Delete operator5 source skill
- **GIVEN** `fab/.kit/skills/fab-operator5.md` exists
- **WHEN** the change is applied
- **THEN** the file is deleted
- **AND** `fab/.kit/skills/fab-operator7.md` remains unchanged

#### Scenario: Delete operator6 source skill
- **GIVEN** `fab/.kit/skills/fab-operator6.md` exists
- **WHEN** the change is applied
- **THEN** the file is deleted

## Launcher Scripts: Delete Old Operator Launchers

### Requirement: Remove operator5 and operator6 launcher scripts

The files `fab/.kit/scripts/fab-operator5.sh` and `fab/.kit/scripts/fab-operator6.sh` SHALL be deleted. `fab/.kit/scripts/fab-operator7.sh` SHALL remain unchanged.

#### Scenario: Delete operator5 launcher
- **GIVEN** `fab/.kit/scripts/fab-operator5.sh` exists
- **WHEN** the change is applied
- **THEN** the file is deleted

#### Scenario: Delete operator6 launcher
- **GIVEN** `fab/.kit/scripts/fab-operator6.sh` exists
- **WHEN** the change is applied
- **THEN** the file is deleted

#### Scenario: operator7 launcher unchanged
- **GIVEN** `fab/.kit/scripts/fab-operator7.sh` exists
- **WHEN** the change is applied
- **THEN** the file content is identical to before

## Spec Files: Delete Old Operator Spec

### Requirement: Remove operator5 spec file

`docs/specs/skills/SPEC-fab-operator5.md` SHALL be deleted. No SPEC-fab-operator6.md exists.

#### Scenario: Delete operator5 spec
- **GIVEN** `docs/specs/skills/SPEC-fab-operator5.md` exists
- **WHEN** the change is applied
- **THEN** the file is deleted

## Memory Files: Update Execution-Skills

### Requirement: Remove operator5 section from execution-skills.md

The `/fab-operator5` section (starting at "### `/fab-operator5`") in `docs/memory/fab-workflow/execution-skills.md` SHALL be removed entirely. The operator4 section and all operator7 design decisions SHALL remain.

#### Scenario: operator5 section removed
- **GIVEN** `docs/memory/fab-workflow/execution-skills.md` contains a `### /fab-operator5` section
- **WHEN** the change is applied
- **THEN** the section (heading through the end of its content before the next `##` heading) is removed
- **AND** operator7 design decisions referencing "extends operator6" are updated to be self-standing (e.g., "extends the operator" or "adds pre-spawn dependency resolution")

### Requirement: Update operator6 references in design decisions

Design decisions in `docs/memory/fab-workflow/execution-skills.md` that reference operator6 (e.g., "extends operator6") SHALL be updated to remove the operator6 reference while preserving the decision's meaning.

#### Scenario: operator7 dep-aware spawning decision updated
- **GIVEN** the "Dependency-Aware Agent Spawning" design decision says "extends operator6"
- **WHEN** the change is applied
- **THEN** the reference is updated to remove "operator6" (e.g., "adds pre-spawn dependency resolution to the operator")

### Requirement: Update changelog entries

Changelog entries in `docs/memory/fab-workflow/execution-skills.md` that reference operator5 or operator6 as intermediate steps MAY be left as-is (they are historical records). No changelog entries SHALL be deleted.

#### Scenario: Historical changelog preserved
- **GIVEN** changelog entries reference operator5 or operator6
- **WHEN** the change is applied
- **THEN** the changelog entries remain unchanged (they are historical)

## Memory Files: Update Kit-Architecture

### Requirement: Remove operator5.sh from scripts listing

In `docs/memory/fab-workflow/kit-architecture.md`, the directory tree and launcher script descriptions SHALL reference only `fab-operator7.sh`. References to `fab-operator4.sh` and `fab-operator5.sh` SHALL be removed from the directory tree and description sections.

#### Scenario: Directory tree updated
- **GIVEN** the directory tree shows `fab-operator4.sh` and `fab-operator5.sh`
- **WHEN** the change is applied
- **THEN** only `fab-operator7.sh` appears in the scripts directory tree

#### Scenario: Launcher descriptions updated
- **GIVEN** launcher descriptions exist for `fab-operator4.sh` and `fab-operator5.sh`
- **WHEN** the change is applied
- **THEN** only `fab-operator7.sh` launcher description remains
- **AND** the description accurately reflects operator7's behavior

### Requirement: Update lib/spawn.sh sourcer list

The `lib/spawn.sh` description SHALL reference `fab-operator7.sh` instead of `fab-operator4.sh` and `fab-operator5.sh` as sourcers.

#### Scenario: spawn.sh sourcer list updated
- **GIVEN** the spawn.sh description says "Sourced by operator launchers (`fab-operator4.sh`, `fab-operator5.sh`)"
- **WHEN** the change is applied
- **THEN** it says "Sourced by operator launcher (`fab-operator7.sh`)"

## Memory Files: Update Index

### Requirement: Update execution-skills description in index

The execution-skills row in `docs/memory/fab-workflow/index.md` SHALL reference `/fab-operator7` instead of `/fab-operator4` and `/fab-operator5`.

#### Scenario: Index description updated
- **GIVEN** the execution-skills row references `/fab-operator4` and `/fab-operator5`
- **WHEN** the change is applied
- **THEN** the row references `/fab-operator7` as the current operator skill

## Specs: Update Superpowers Comparison

### Requirement: Update operator references in superpowers-comparison.md

References to `operator5/6/7` in `docs/specs/superpowers-comparison.md` SHALL be simplified to reference only the current operator (`operator7` or just "operator").

#### Scenario: Multi-agent coordination row updated
- **GIVEN** the comparison table references "operator5/6/7"
- **WHEN** the change is applied
- **THEN** the reference says "operator" or "operator7" (no version range)

## Deprecated Requirements

### SPEC-fab-operator5.md
**Reason**: operator5 has been superseded by operator7; the spec is no longer needed.
**Migration**: N/A — operator7 spec (if it exists) or the operator7 skill file itself is the authoritative reference.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only operator7 is current | User explicit — "keep only fab-operator7" | S:95 R:90 A:95 D:95 |
| 2 | Certain | fab-operator7.sh launcher unchanged | User explicit | S:95 R:95 A:95 D:95 |
| 3 | Certain | Delete source skill files for operator5 and operator6 | User explicit; deployed copies excluded per user correction | S:95 R:85 A:95 D:95 |
| 4 | Confident | Remove full operator5 section from execution-skills.md | Dead versions should be fully removed — consistent with cleanup intent | S:80 R:80 A:75 D:80 |
| 5 | Confident | No SPEC-fab-operator6.md exists to delete | Only SPEC-fab-operator5.md found in docs/specs/skills/ | S:70 R:95 A:85 D:90 |
| 6 | Confident | No migration needed | Internal repo cleanup — no user-facing data structures change | S:75 R:90 A:80 D:85 |
| 7 | Certain | Delete operator5.sh and operator6.sh launcher scripts | Launchers for deleted skills serve no purpose; user said only operator7.sh remains | S:90 R:85 A:90 D:90 |
| 8 | Confident | operator4 artifacts (including fab-operator4.sh) are already absent on main | operator4 cleanup is out of scope per Non-Goals; only specific references in kit-architecture docs are updated | S:70 R:80 A:75 D:70 |
| 9 | Confident | Preserve changelog entries referencing old operators | Changelogs are historical records — deleting entries loses provenance | S:75 R:85 A:85 D:85 |
| 10 | Confident | Update "extends operator6" to standalone phrasing | Referencing a deleted entity creates a dangling reference | S:80 R:85 A:80 D:85 |

10 assumptions (4 certain, 6 confident, 0 tentative, 0 unresolved).
