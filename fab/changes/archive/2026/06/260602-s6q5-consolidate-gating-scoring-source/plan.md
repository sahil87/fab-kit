# Plan: Consolidate gating/scoring data into one source of truth

**Change**: 260602-s6q5-consolidate-gating-scoring-source
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md. Code map is canonical; the doc table is a verified
     mirror. A new Go test in the `internal/score` package enforces agreement. -->

### Scoring: Code↔Doc Consistency Guard

#### R1: Consistency test asserts the doc tables match the canonical Go maps
A new Go test in package `internal/score` SHALL parse the "Expected Minimum Decisions"
and "Gate Thresholds" tables in `docs/specs/change-types.md` and assert that each of the
7 canonical change types resolves to the same value in the doc as in the code, comparing
against the RESOLVED values `getExpectedMin(type)` and `getGateThreshold(type)` — not raw
map membership (the maps omit default-valued types).

- **GIVEN** the code maps (`expectedMin`, `gateThresholds`) and the doc tables agree
- **WHEN** `go test ./internal/score/` runs
- **THEN** the consistency test passes
- **AND** if a threshold value is later changed in the code map without updating the doc table (or vice versa), the test FAILS with a message naming the drifted type and the two values

#### R2: Test asserts the doc covers exactly the 7 canonical types
The consistency test SHALL assert the doc's two tables each cover exactly the canonical
type set `{feat, fix, refactor, docs, test, ci, chore}` — no missing rows, no stray rows —
so adding/removing/renaming a type in code without updating the doc (or vice versa) fails.

- **GIVEN** the doc lists exactly the 7 canonical types in both tables
- **WHEN** the consistency test runs
- **THEN** the type-set assertion passes
- **AND** if a type is added/removed/renamed in only one place, the test FAILS naming the offending type

#### R3: Markdown table parser reuses the existing `bufio.Scanner` idiom
The table parser SHALL be an unexported test helper in the `score` package's test files
that scans the markdown line-by-line with `bufio.Scanner` and pipe-splitting (the same idiom
as `countGrades` in `score.go`), anchoring on the section heading (`## Expected Minimum Decisions`,
`## Gate Thresholds`). It MUST NOT introduce any markdown/YAML library (Constitution Principle I:
single-binary, no new runtime deps).

- **GIVEN** the parser is asked for a table by its section heading
- **WHEN** it scans from that heading to the next heading
- **THEN** it skips the column-header row and the `|---|` separator, and returns `{type → value}` with the type's backticks/whitespace stripped and the value parsed as `int`/`float64`
- **AND** no new import of a third-party markdown/YAML package is added

#### R4: Doc-file path resolved by walking up to the repo root
The test SHALL resolve `docs/specs/change-types.md` by walking up from the test's working
directory until it finds the file, rather than hard-coding a fixed relative depth
(`../../../../../`). If the file cannot be located, the test SHALL fail with a clear message.

- **GIVEN** Go runs the test with the working directory set to the package dir
- **WHEN** the test resolves the doc path
- **THEN** it ascends parent directories until `docs/specs/change-types.md` exists and uses that path
- **AND** if no ancestor contains the file, the test fails with a clear, actionable message

#### R5: Reword the self-contradictory "source of truth" lines in the doc
`docs/specs/change-types.md` SHALL be reworded so the CODE map is named canonical and the
doc table is described as a verified mirror, in both the Expected Minimum Decisions section
(line ~37) and the Gate Thresholds section. No threshold VALUE may change.

- **GIVEN** the doc currently claims the doc values "are the source of truth and are embedded in" the code (self-contradictory)
- **WHEN** the wording is updated
- **THEN** the Expected Minimum Decisions section states the doc table mirrors the canonical `expectedMin` map and is guarded by a test that fails on drift
- **AND** the Gate Thresholds section carries the same clarification pointing at the canonical `gateThresholds` map
- **AND** every numeric value in both tables is unchanged

### Non-Goals

- Prose mentions of the same numbers in `docs/specs/srad.md`, `docs/specs/glossary.md`, `src/kit/skills/_preamble.md` — explanatory prose, not structured tables; out of scope.
- The change-type keyword-inference table — runs in skill markdown the binary never reads; no Go map to tie it to.
- Changing any threshold value — behavior is byte-for-byte identical.
- Shared-data-file or `go:generate` approaches — both considered and rejected at intake.

### Design Decisions

1. **Test-guard, code-canonical**: Go maps stay canonical; a test parses the doc and asserts equality. — *Why*: least machinery, no new runtime dep, fits single-binary Constitution; converts silent drift into a loud `just test` failure. — *Rejected*: shared data file (embed can't reach `src/kit/`, runtime file dep conflicts with Principle I); `go:generate` (more machinery than the goal needs).
2. **Compare against resolved getters, not raw map keys**: the doc lists all 7 types; the maps omit default types. — *Why*: comparing raw keys would spuriously fail on default rows.
3. **Anchor the parser on section headings, not column headers**: both tables share a `Type` first column. — *Why*: heading anchoring is unambiguous.

## Tasks

<!-- Each item carries an <!-- R# --> trace annotation. -->

### Phase 2: Core Implementation

- [x] T001 Add `changetypes_doc_test.go` in `src/go/fab/internal/score/` with: (a) `findDocFile` helper that walks up from CWD to locate `docs/specs/change-types.md` (fail clearly if absent); (b) `parseChangeTypeTable` helper using `bufio.Scanner`/pipe-split anchored on a section heading, returning a `map[string]string` of `{type → raw value}`; (c) `TestDocTablesMatchScoringMaps` that parses both tables, asserts each of the 7 types matches `getExpectedMin`/`getGateThreshold`, and asserts the doc covers exactly the 7 canonical types in both tables <!-- R1 R2 R3 R4 -->

### Phase 4: Polish

- [x] T002 Reword the "source of truth" lines in `docs/specs/change-types.md`: the Expected Minimum Decisions section (line ~37) and the Gate Thresholds section, making the code maps canonical and the doc tables a verified mirror guarded by the new test; change no numeric value <!-- R5 -->

## Execution Order

- T001 must pass before T002 is meaningful, but T002 (doc wording) does not change values, so it does not affect T001's assertions. Run T001's test after both for final verification.

## Acceptance

### Functional Completeness

- [ ] A-001 R1: A test in `internal/score` parses both doc tables and asserts each of the 7 types matches `getExpectedMin`/`getGateThreshold`; `go test ./internal/score/` passes.
- [ ] A-002 R2: The test asserts the doc covers exactly the 7 canonical types in both tables (no missing/stray rows).
- [ ] A-003 R3: The parser is an unexported test helper using `bufio.Scanner`/pipe-split anchored on the section heading; no markdown/YAML library is imported.
- [ ] A-004 R4: The doc path is resolved by walking up to the repo root; a missing file produces a clear failure message.
- [ ] A-005 R5: Both "source of truth" lines in `change-types.md` are reworded to code-canonical / doc-mirror; no numeric value changed.

### Behavioral Correctness

- [ ] A-006 R1: Inducing a code/doc value mismatch makes the test fail with a message naming the drifted type and both values (verified by reasoning about the assertion, not a permanent test).
- [ ] A-007 R2: Adding/removing/renaming a type in only one place makes the type-set assertion fail.

### Edge Cases & Error Handling

- [ ] A-008 R4: When `docs/specs/change-types.md` cannot be located by walking up, the test fails with a clear message rather than panicking.
- [ ] A-009 R1: The comparison uses resolved getter values, so the explicit default-valued doc rows (docs/test/ci/chore) do not spuriously fail.

### Code Quality

- [ ] A-010 Pattern consistency: New test follows `score_test.go` conventions — standard `testing` package only, table-driven `t.Run` subtests where natural, `t.Fatalf`/`t.Errorf` style.
- [ ] A-011 No unnecessary duplication: The parser reuses the existing line-by-line `bufio.Scanner`/pipe-split idiom rather than adding a markdown library.
- [ ] A-012 No magic numbers: The canonical type list and table heading strings are named (constants or clearly-named locals), not scattered literals.

### Documentation Accuracy

- [ ] A-013 R5: The reworded doc is internally consistent — it no longer claims to be both "the source of truth" and "embedded from" the code.

### Cross References

- [ ] A-014 R5: The doc's cross-reference to `src/go/fab/internal/score/score.go` correctly names the `expectedMin` and `gateThresholds` maps and the guarding test.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

<!-- Apply-time decisions, SRAD-graded. Code is canonical; no values change. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New test lives in a dedicated `changetypes_doc_test.go` (not appended to `score_test.go`) | Intake offers either; a separate file keeps the consistency concern isolated and matches the package's one-concern-per-file leaning while still in-package (can call unexported getters) | S:85 R:90 A:85 D:80 |
| 2 | Certain | Parser returns raw string values keyed by type; the test converts (`int`/`float64`) at comparison time | Keeping the parser value-type-agnostic lets one helper serve both tables; conversion lives next to the per-type assertion | S:80 R:85 A:90 D:80 |
| 3 | Confident | `findDocFile` walks up using `os.Getwd()` then `filepath.Dir` until root, checking `docs/specs/change-types.md` at each level | Standard, dependency-free walk-up; Go test CWD is the package dir so the file is 5 levels up but the loop is depth-agnostic | S:80 R:85 A:85 D:80 |
| 4 | Confident | Type-set assertion compares a sorted slice of parsed doc types against the canonical 7-type list for each table | Simplest unambiguous bidirectional set check; reuses the same canonical list used for value comparison | S:78 R:85 A:85 D:78 |
| 5 | Certain | No current drift to reconcile (code and doc agree) | Verified by reading both files at apply time: expected_min feat:7/refactor:6/fix:5/default 3; gate flat 3.0 — confirmed by the test's first green run | S:90 R:85 A:90 D:85 |
| 6 | Confident | Doc rewording phrasing names the canonical map and the guarding test explicitly | Intake fixed the direction-of-truth and called exact phrasing an apply-time detail; chosen wording is honest and reversible | S:78 R:90 A:80 D:75 |

6 assumptions (3 certain, 3 confident, 0 tentative).
