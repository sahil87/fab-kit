# Spec: Update Specs References from Stageman to Statusman

**Change**: 260228-hqv5-update-specs-statusman-refs
**Created**: 2026-02-28
**Affected memory**: None — specs are independent of memory files

## Non-Goals

- Changing any behavioral logic or script functionality — this is a pure text replacement
- Updating files outside `docs/specs/` — skills, memory, and scripts were already updated in PR #177

## Specs: Script Name Consistency

### Requirement: Stageman-to-Statusman Text Replacement

All references to `stageman.sh` in `docs/specs/` SHALL be replaced with `statusman.sh` to match the rename completed in PR #177.

The replacement MUST be a literal text substitution with no semantic or structural changes to the surrounding content.

#### Scenario: Simple Reference Replacement
- **GIVEN** a spec file containing `stageman.sh` in a code block, table, or prose
- **WHEN** the replacement is applied
- **THEN** `stageman.sh` is replaced with `statusman.sh`
- **AND** no other text in the line or surrounding context is modified

#### Scenario: Directory Tree Listing
- **GIVEN** `architecture.md` contains a directory tree showing `stageman.sh` as a file entry
- **WHEN** the replacement is applied
- **THEN** the tree entry reads `statusman.sh` with the same indentation and comment

#### Scenario: Script Examples Table
- **GIVEN** `architecture.md` contains a table row listing `lib/stageman.sh` as an example
- **WHEN** the replacement is applied
- **THEN** the table row reads `lib/statusman.sh`

### Requirement: Glossary Script Enumeration Update

The `glossary.md` script enumeration in the Files and Structure table SHALL list the current 6-script architecture: `resolve.sh`, `statusman.sh`, `logman.sh`, `changeman.sh`, `calc-score.sh`, `preflight.sh`.

#### Scenario: Glossary Lists Correct Scripts
- **GIVEN** `glossary.md` has a row describing `fab/.kit/scripts/`
- **WHEN** a reader checks the internal script listing
- **THEN** the listing includes `resolve.sh`, `statusman.sh`, `logman.sh`, `changeman.sh`, `calc-score.sh`, `preflight.sh`
- **AND** `stageman.sh` does not appear

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Pure text replacement: stageman → statusman | Confirmed from intake #1 — no semantic changes, only updating stale script names to match codebase | S:95 R:95 A:95 D:95 |
| 2 | Certain | Update glossary script enumeration to reflect current architecture | Confirmed from intake #2 — glossary should list the actual set of scripts | S:90 R:90 A:90 D:95 |
| 3 | Certain | All 7 spec files affected as identified in intake | Verified by grep — architecture.md, change-types.md, glossary.md, naming.md, skills.md, templates.md, user-flow.md | S:95 R:95 A:95 D:95 |

3 assumptions (3 certain, 0 confident, 0 tentative, 0 unresolved).
