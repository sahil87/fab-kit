# Spec: Expand wt name lists and fix wt list output

**Change**: 260316-ez49-expand-wt-name-lists
**Created**: 2026-03-16
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## Worktree Naming: Expanded Word Lists

### Requirement: Adjective list expansion

The `adjectives` slice in `src/go/wt/internal/worktree/names.go` SHALL contain at least 120 entries. Each entry MUST be a lowercase English adjective that is 1–3 syllables (≤7 characters preferred), has a positive or neutral connotation, and is distinct from all other entries (no near-synonyms).

#### Scenario: Expanded adjective count
- **GIVEN** the `adjectives` variable in `names.go`
- **WHEN** a developer inspects the list length
- **THEN** `len(adjectives)` SHALL be >= 120

#### Scenario: Adjective quality constraints
- **GIVEN** any entry in the `adjectives` slice
- **WHEN** evaluated for suitability
- **THEN** it MUST be a real English adjective, lowercase, with no negative or grim connotation

### Requirement: Noun list expansion

The `nouns` slice in `src/go/wt/internal/worktree/names.go` SHALL contain at least 120 entries. Each entry MUST be a real animal name (no mythical creatures), lowercase, ideally ≤8 characters. Entries SHOULD be grouped thematically by animal family.

#### Scenario: Expanded noun count
- **GIVEN** the `nouns` variable in `names.go`
- **WHEN** a developer inspects the list length
- **THEN** `len(nouns)` SHALL be >= 120

#### Scenario: Noun quality constraints
- **GIVEN** any entry in the `nouns` slice
- **WHEN** evaluated for suitability
- **THEN** it MUST be a real animal name, lowercase, and distinct from all other entries

### Requirement: Comment accuracy

The comments above each list variable SHALL reflect the actual count (e.g., `~120 adjectives` instead of `~50 adjectives`).

#### Scenario: Comment matches reality
- **GIVEN** the comment above the `adjectives` variable
- **WHEN** compared to `len(adjectives)`
- **THEN** the stated count SHALL be within ±10 of the actual count

### Requirement: No logic changes

`GenerateRandomName()` and `GenerateUniqueName()` SHALL remain unchanged. They already use `len()` for indexing.

#### Scenario: Functions unchanged
- **GIVEN** the current `GenerateRandomName()` and `GenerateUniqueName()` functions
- **WHEN** this change is applied
- **THEN** neither function body SHALL be modified

### Requirement: Test threshold update

The `TestWordListsNonEmpty` test in `names_test.go` SHALL update its minimum thresholds from 40 to 100 to reflect the expanded lists.

#### Scenario: Test validates expanded lists
- **GIVEN** the `TestWordListsNonEmpty` test
- **WHEN** run against the expanded word lists
- **THEN** it SHALL pass with thresholds of at least 100 for both lists

## Worktree List: Remove separator row

### Requirement: Remove dash separator from formatted output

The `handleFormattedOutput` function in `src/go/wt/cmd/list.go` SHALL remove the dash separator row entirely. The header row SHALL be followed directly by data rows.

#### Scenario: No separator row in output
- **GIVEN** a repository with at least one worktree
- **WHEN** `wt list` is run (formatted output, no `--json`)
- **THEN** the output SHALL contain the header row ("Name", "Branch", "Status", "Path") but SHALL NOT contain a row of dashes (`----`)

#### Scenario: JSON and path output unaffected
- **GIVEN** `wt list --json` or `wt list --path <name>`
- **WHEN** run
- **THEN** output format SHALL be unchanged

### Requirement: Test update for removed separator

The `TestList_HeaderAndSeparator` test in `list_test.go` SHALL be updated to no longer assert the presence of `"----"` in the output.

#### Scenario: Test reflects new format
- **GIVEN** the `TestList_HeaderAndSeparator` test
- **WHEN** run against the updated `wt list`
- **THEN** it SHALL validate headers exist but SHALL NOT check for dash separators

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Expand to ≥120 entries per list | Confirmed from intake #1 — tripling is meaningful without bloat | S:70 R:95 A:80 D:85 |
| 2 | Certain | Keep adjective-noun pattern | Confirmed from intake #2 — no request to change naming scheme | S:75 R:90 A:90 D:95 |
| 3 | Certain | No logic changes to generation functions | Confirmed from intake #3 — functions use len() | S:80 R:95 A:95 D:95 |
| 4 | Confident | Short, positive words only | Confirmed from intake #4 — existing pattern, readability | S:60 R:90 A:85 D:75 |
| 5 | Confident | Real animals only for nouns | Confirmed from intake #5 — consistency with existing list | S:60 R:90 A:85 D:80 |
| 6 | Certain | Remove separator row entirely (not fix alignment) | Upgraded from intake #7 — user explicitly offered removal as option, simpler fix | S:75 R:95 A:80 D:90 |
| 7 | Certain | Update test thresholds to match | Codebase has explicit threshold tests that will fail if not updated | S:85 R:95 A:95 D:95 |
| 8 | Confident | Thematic grouping of animals | Existing list groups by family; new entries should follow same convention | S:55 R:95 A:80 D:70 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
