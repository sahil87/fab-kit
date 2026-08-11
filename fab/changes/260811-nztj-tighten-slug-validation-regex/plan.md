# Plan: Tighten Slug Validation Regex

**Change**: 260811-nztj-tighten-slug-validation-regex
**Intake**: `intake.md`

## Requirements

### internal/change: Slug validation matches the documented contract

#### R1: `slugRegex` enforces 2-6 lowercase kebab-case words
The `slugRegex` in `src/go/fab/internal/change/change.go` SHALL be `^[a-z0-9]+(-[a-z0-9]+){1,5}$`, replacing the current `^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`. The new pattern SHALL enforce, in one expression: lowercase letters and digits only; 2-6 hyphen-separated words (one mandatory head word plus 1-5 `-word` groups); no leading or trailing hyphen; and no consecutive hyphens. It SHALL remain the single shared validator for both `change.New` and `change.Rename`. `idRegex` SHALL NOT be touched.

- **GIVEN** the slug `"MyFeature"`, `"my-Feature"`, `"cleanup"`, `"a-b-c-d-e-f-g"`, or `"my--feature"`
- **WHEN** `slugRegex.MatchString` runs
- **THEN** each returns `false`
- **AND** `"my-feature"`, `"one-two-three-four-five-six"`, and `"fix-v2-parser"` each return `true`

#### R2: Both error messages state the enforced rule
The two identical slug-validation error strings (in `New` at the `slugRegex.MatchString(slug)` check and in `Rename` at the `slugRegex.MatchString(newSlug)` check) SHALL read:

```go
fmt.Errorf("Invalid slug format '%s' (expected 2-6 lowercase kebab-case words, e.g. 'add-oauth-support')", slug)
```

(with `newSlug` as the argument in `Rename`). The rejection message MUST match the validation rule so the failure is actionable.

- **GIVEN** a call to `fab change new --slug MyChange` (or `fab change rename --slug MyChange`)
- **WHEN** validation rejects the slug
- **THEN** the error text is `Invalid slug format 'MyChange' (expected 2-6 lowercase kebab-case words, e.g. 'add-oauth-support')`

#### R3: Test coverage matches the enforced contract
`src/go/fab/internal/change/change_test.go` SHALL gain cases, in the file's existing test style, covering: uppercase rejected (`"MyFeature"`, `"my-Feature"`); single word rejected (`"cleanup"`); 7 words rejected (`"a-b-c-d-e-f-g"`); consecutive hyphens rejected (`"my--feature"`); 6-word boundary accepted (`"one-two-three-four-five-six"`); digits accepted (`"fix-v2-parser"`); and `Rename` rejecting an invalid new slug. Existing invalid-slug tests (`"my feature!"`, `"-starts-with-hyphen"`, empty) SHALL keep passing unchanged.

- **GIVEN** the tightened regex in place
- **WHEN** `go test ./internal/change/...` runs
- **THEN** all new and pre-existing slug tests pass

#### R4: `_cli-fab.md` documents the enforced slug format
The `new` and `rename` rows of the `fab change` subcommand table in `src/kit/skills/_cli-fab.md` SHALL note the enforced slug format (`<slug>` must be 2-6 lowercase kebab-case words), per the constitution-mandated CLI-doc sync. The canonical source is edited; `.claude/skills/` deployed copies SHALL NOT be touched.

- **GIVEN** an agent reading `src/kit/skills/_cli-fab.md` to compose a `--slug` value
- **WHEN** they read the `new` or `rename` row
- **THEN** the 2-6 lowercase kebab-case words constraint is visible at the call site

### Non-Goals

- `idRegex` — already strict, untouched.
- Existing folder names — validation runs only at create/rename; pre-existing non-conforming slugs (e.g. archived 7-word slugs) remain valid on disk.
- Spec files (`architecture.md`, `naming.md`) — already state the rule; specs are human-curated and need no edit.

## Tasks

<!-- Sequential work items for the apply stage. Checked off [x] as completed.
     Each item MUST carry a <!-- R# --> trace annotation. -->

### Phase 1: Core Implementation

- [x] T001 Tighten `slugRegex` in `src/go/fab/internal/change/change.go` to `^[a-z0-9]+(-[a-z0-9]+){1,5}$` <!-- R1 -->
- [x] T002 Update both slug-validation error strings in `src/go/fab/internal/change/change.go` (`New` and `Rename`) to `Invalid slug format '%s' (expected 2-6 lowercase kebab-case words, e.g. 'add-oauth-support')` <!-- R2 -->

### Phase 2: Tests

- [x] T003 Add the intake-listed slug test cases (uppercase, single word, 7 words, consecutive hyphens, 6-word boundary, digits, Rename-side rejection) to `src/go/fab/internal/change/change_test.go` in the file's existing style, then run `go test ./internal/change/...` and `go test ./cmd/...` from `src/go/fab` <!-- R3 -->

### Phase 3: Docs

- [x] T004 Add the slug-format note ("`<slug>` must be 2-6 lowercase kebab-case words") to the `new`/`rename` rows in `src/kit/skills/_cli-fab.md` <!-- R4 -->

## Acceptance

<!-- Declarative acceptance criteria used by the review stage. -->

### Functional Completeness

- [x] A-001 R1: `slugRegex` is `^[a-z0-9]+(-[a-z0-9]+){1,5}$` and remains the single validator shared by `New` and `Rename`; `idRegex` is unchanged
- [x] A-002 R2: Both error strings read exactly `Invalid slug format '%s' (expected 2-6 lowercase kebab-case words, e.g. 'add-oauth-support')`
- [x] A-003 R3: All intake-listed test cases exist in `change_test.go` and `go test ./internal/change/...` passes
- [x] A-004 R4: The `new`/`rename` rows in `src/kit/skills/_cli-fab.md` note the enforced slug format; no file under `.claude/skills/` was edited

### Behavioral Correctness

- [x] A-005 R1: `fab change new`/`rename` reject uppercase, 1-word, 7+-word, and `--`-containing slugs and accept 2-6-word lowercase kebab-case slugs with digits

### Scenario Coverage

- [x] A-006 R3: Tests exercise both boundaries (2 words and 6 words accepted; 1 word and 7 words rejected) and the `Rename` call site

### Edge Cases & Error Handling

- [x] A-007 R1: Pre-existing invalid-slug tests (`"my feature!"`, `"-starts-with-hyphen"`, empty) still pass; rejection error text matches the enforced rule

### Code Quality

- [x] A-008 Pattern consistency: New tests follow the existing `change_test.go` naming/style (`TestNew_*` / `TestRename_*` helpers and fixture reuse)
- [x] A-009 No unnecessary duplication: One shared `slugRegex` for both call sites; no parallel validation introduced
- [x] A-010 Test-alongside: Test updates ship in the same change as the code they cover

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Requirements, regex, error text, and test cases reproduce the intake's What Changes verbatim | The intake carries exact code blocks and an enumerated test list; no design gap to resolve at apply | S:95 R:95 A:95 D:95 |
| 2 | Confident | New tests are added as focused `TestNew_*`/`TestRename_*` functions reusing `setupChangeFixture`, matching the file's existing per-case style | The file's convention is one function per case (e.g. `TestNew_InvalidSlug`, `TestNew_InvalidSlugLeadingHyphen`); the intake says "following the file's existing test style" without naming the shape | S:80 R:95 A:85 D:75 |
| 3 | Confident | The `_cli-fab.md` note is appended to the `new`/`rename` Purpose cells as "`<slug>` must be 2-6 lowercase kebab-case words" | Intake names this exact phrasing as the example for the `new` row and requires both rows noted; placement in the table's Purpose column is the only visible spot in those rows | S:75 R:90 A:80 D:70 |

3 assumptions (1 certain, 2 confident, 0 tentative).
