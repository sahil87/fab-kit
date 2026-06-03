# Intake: Consolidate gating/scoring data into one source of truth

**Change**: 260602-s6q5-consolidate-gating-scoring-source
**Created**: 2026-06-03
**Status**: Draft

## Origin

This change was initiated via `/fab-new` with a detailed natural-language description, then refined through a short conversational clarification before intake generation.

> Consolidate gating/scoring data into one source of truth. Per-change-type values are currently spread across multiple places — `gateThresholds` (flat 3.0) and `expectedMin` (feat:7/refactor:6/fix:5, default 3) in `src/go/fab/internal/score/score.go`, mirrored by hand in `docs/specs/change-types.md` (gate + expected_min tables). No automated check ties code and doc together, so they silently drift (the v1.10.0 spec-merge had to reconcile a pre-existing code/doc drift). Reduce to a single canonical definition (e.g. generate the doc table from the Go maps, or a shared data file) so changing a threshold is a one-place edit.

**Decisions reached in conversation:**

1. **Approach: test-guard, code-canonical.** Rejected the "shared data file" option after analysis: the natural home for project-wide taxonomy data (`src/kit/`) is a *sibling* of the Go module, so `go:embed` cannot reach it without a build-time copy or a runtime file read — and a runtime file dependency brushes against Constitution Principle I ("single-binary, no runtime frameworks"). The only embed-reachable location (`src/go/fab/internal/score/`) would misfile a project-wide taxonomy inside an internal package. `go:generate` was also rejected as more machinery than the goal requires. The agreed approach: the Go maps (`expectedMin`, `gateThresholds`) **stay canonical**, and a new Go test parses the markdown tables in `docs/specs/change-types.md` and asserts they equal the maps. Drift then fails `just test`.
2. **Scope: both tables in `change-types.md`.** Tie both the `expected_min` table and the gate-threshold table. Prose mentions of the same numbers in `docs/specs/srad.md`, `docs/specs/glossary.md`, and `src/kit/skills/_preamble.md` are explicitly **out of scope** — they are explanatory prose, not structured tables, and brittle to parse.

## Why

**The problem.** The per-change-type gating/scoring numbers exist in two hand-maintained copies with no link between them:

- **Code** — `src/go/fab/internal/score/score.go`:
  - `expectedMin = {"feat": 7, "refactor": 6, "fix": 5}` (default 3 for `docs`/`test`/`ci`/`chore` via `getExpectedMin`)
  - `gateThresholds = {fix, feat, refactor, docs, test, ci, chore → 3.0}` (default 3.0 via `getGateThreshold`)
- **Doc** — `docs/specs/change-types.md`:
  - "Expected Minimum Decisions" table (the `expected_min` column)
  - "Gate Thresholds" table

The doc even *claims* to be authoritative ("These values are the source of truth"), while the code comment on `expectedMin` calls itself the seed. Nothing enforces agreement.

**The consequence if unfixed.** They drift silently. This already happened: the v1.10.0 spec-merge (`260601-j6cs`) had to reconcile a pre-existing code/doc drift as a side effect. Every future threshold tweak risks updating one copy and forgetting the other, and the drift is invisible until someone reads both side by side. A stale doc misleads contributors about how scoring actually behaves.

**Why this approach over alternatives.** Three options were considered (see Origin for full reasoning):

| Option | One-place edit? | Drift caught? | New machinery | Verdict |
|--------|-----------------|---------------|---------------|---------|
| **Test-guard (chosen)** | Edit map + fix doc (test points at the stale row) | Yes — `just test` fails | A single test + a small markdown-table parser | **Chosen** — least machinery, fits single-binary constitution |
| go:generate doc | Edit map, run `go generate` | Yes — test diffs generated vs. committed | Generator binary + marker comments in doc | Rejected — more than the goal needs |
| Shared data file | Edit one YAML | Yes | New data file + `go:embed` reach problem + runtime parse | Rejected — embed can't reach `src/kit/`; runtime file dep conflicts with Principle I |

Test-guard keeps the numbers where the compiled binary already carries them (the Go map), adds no runtime dependency and no build step, and converts silent drift into a loud `just test` failure — the existing verification channel for this repo (`just test` runs `go test ./...`, and `internal/score` is already a tested package).

## What Changes

### 1. New consistency test in `internal/score`

Add a test (in `src/go/fab/internal/score/score_test.go`, or a new `changetypes_doc_test.go` in the same package) that:

1. Locates `docs/specs/change-types.md` relative to the test file. The `score` package sits at `src/go/fab/internal/score/`, so the repo root is four directories up; the doc is at `../../../../../docs/specs/change-types.md` from the package dir. The test SHOULD resolve this robustly (e.g., walk up from the test's working directory until it finds `docs/specs/change-types.md`, or use a relative path with a clear failure message if the file is absent) rather than hard-coding a fragile depth count.
2. Parses the two relevant markdown tables:
   - **Expected Minimum Decisions** table → a `map[string]int` of `{type → expected_min}`.
   - **Gate Thresholds** table → a `map[string]float64` of `{type → gate}`.
3. Asserts the parsed tables match the canonical Go maps. The comparison MUST account for the default-value semantics: the doc table lists **all 7 types explicitly** (including `docs`/`test`/`ci`/`chore` at the default), whereas the Go `expectedMin` map only lists the 3 non-default types and resolves the rest via `getExpectedMin`'s fallback of 3. The test SHALL therefore compare the doc's value for each of the 7 types against `getExpectedMin(type)` (and `getGateThreshold(type)`), not against raw map membership — otherwise the explicit doc rows for default types would spuriously fail.

**Comparison shape (illustrative):**

```go
func TestDocTablesMatchScoringMaps(t *testing.T) {
    docPath := findDocFile(t, "docs/specs/change-types.md") // walk up to repo root
    expMinDoc, gateDoc := parseChangeTypeTables(t, docPath)

    allTypes := []string{"feat", "fix", "refactor", "docs", "test", "ci", "chore"}
    for _, ct := range allTypes {
        if got, want := expMinDoc[ct], getExpectedMin(ct); got != want {
            t.Errorf("change-types.md expected_min[%s]=%d, code getExpectedMin=%d (doc drifted)", ct, got, want)
        }
        if got, want := gateDoc[ct], getGateThreshold(ct); got != want {
            t.Errorf("change-types.md gate[%s]=%.1f, code getGateThreshold=%.1f (doc drifted)", ct, got, want)
        }
    }
    // Guard the other direction too: the doc must cover exactly the 7 known types,
    // so a renamed/added/removed type in either place is caught.
}
```

The test SHALL also assert the doc covers **exactly** the 7 canonical types (no missing rows, no stray rows), so adding/removing/renaming a type in code without updating the doc (or vice versa) fails too.

### 2. Markdown table parser (test helper)

A small parser, scoped to the test (unexported, in the `score` package's test files), that:
- Finds a table by its section heading (e.g., `## Expected Minimum Decisions`, `## Gate Thresholds`) — anchoring on the heading is more robust than matching column headers, since both tables share a `Type` first column.
- Reads the pipe-delimited rows under that section until the next heading, skipping the header row and the `|---|` separator.
- Extracts the type (first column, stripping backticks/whitespace) and the numeric value (parsing `int` for expected_min, `float64` for gate).
- This mirrors the existing markdown-parsing idiom already present in this package (`countGrades` in `score.go` scans markdown line-by-line with `bufio.Scanner` and pipe-splitting) — reuse that style for consistency rather than introducing a markdown library (Constitution Principle I: single-binary, no new runtime deps).

### 3. Reconcile any current drift (if present)

Before the test can pass, verify the doc and code currently agree. From the read at intake time they **do** agree (expected_min feat:7/refactor:6/fix:5/default 3; gate flat 3.0). If a discrepancy surfaces when the test is first run, fix the **doc** to match the code (code is canonical per the chosen approach) and note the reconciliation in the change.

### 4. Documentation note (optional, low-priority)

Update the wording in `docs/specs/change-types.md` so it no longer claims to *be* the source of truth in conflict with the code. Today line 37 says "These values are the source of truth and are embedded in `src/go/fab/internal/score/score.go`" — which is contradictory (it can't be both *the* source and *embedded from* somewhere). Reword to make the **code map canonical** and the **doc table a verified mirror** (e.g., "These values mirror the canonical `expectedMin` map in `src/go/fab/internal/score/score.go`; a test in that package fails if the two drift."). Apply the same one-line clarification to the Gate Thresholds section. This keeps the human-facing claim honest about the new direction-of-truth.

### Out of scope

- Prose mentions of the same numbers in `docs/specs/srad.md`, `docs/specs/glossary.md`, and `src/kit/skills/_preamble.md` — explanatory prose, not tables; brittle to parse; explicitly deferred per conversation.
- Changing any threshold *value* — this change only ties existing values together; behavior is identical.
- The change-type **keyword-inference** table (also duplicated between `change-types.md` and the `fab-new` skill) — that inference runs in skill markdown the binary never reads, so there's no Go map to tie it to. Out of scope.
- Migrating to a shared data file or `go:generate` (both considered and rejected).

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) The `internal/score` package gains a doc-consistency test; the "Testing" section lists `internal/score` among tested packages and may note the new code↔doc tie as part of the score package's coverage.
- `fab-workflow/configuration`: (modify) The scoring-data history notes that `expectedMin`/`gateThresholds` in `score.go` are now the canonical source with a test guarding `change-types.md` against drift (supersedes the doc's prior "source of truth" claim).

## Impact

- **`src/go/fab/internal/score/score_test.go`** (or new `changetypes_doc_test.go`) — new test + markdown-table parser helper. Primary change.
- **`src/go/fab/internal/score/score.go`** — no functional change; the maps stay as-is. (Comments may be lightly updated to point at the doc test, optional.)
- **`docs/specs/change-types.md`** — possibly reconcile drift (none expected); reword the "source of truth" lines to reflect code-canonical direction.
- **Test execution** — `just test` (which runs `cd src/go/fab && go test ./...`) is the enforcement channel; CI runs the same. No new build target, no new dependency, no runtime change to the `fab` binary.
- **No API/CLI surface change** — `fab score` behavior is byte-for-byte identical.
- **Edge case — test file path resolution:** Go tests run with the working directory set to the package dir, so a relative path to the repo-root doc must account for that. Walking up to find `docs/specs/change-types.md` is more robust than a fixed `../../../../../` and degrades gracefully (clear skip/fail message) if the layout changes.
- **Edge case — default-type rows:** the doc lists all 7 types; the code map lists only non-defaults. The test must compare against the `getExpectedMin`/`getGateThreshold` *resolved* values, not raw map keys (covered in What Changes §1).

## Open Questions

- None blocking. Both design forks (approach, scope) were resolved in conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Approach = test-guard, Go maps canonical, doc verified against them | Decided in conversation after analyzing all 3 options; user explicitly chose test-guard | S:98 R:75 A:90 D:95 |
| 2 | Certain | Scope = both tables (expected_min + gate) in `change-types.md`; prose elsewhere out of scope | User explicitly selected "Both tables in change-types.md"; prose deferred | S:98 R:80 A:90 D:95 |
| 3 | Certain | Shared-data-file and go:generate approaches are rejected | Discussed — embed can't reach `src/kit/`, runtime file dep conflicts with Principle I; go:generate is more machinery than needed | S:95 R:70 A:85 D:90 |
| 4 | Confident | Test lives in the `score` package (`score_test.go` or new `changetypes_doc_test.go`) | `internal/score` is already a tested package; co-locating with the canonical maps is the natural home and lets the test call unexported `getExpectedMin`/`getGateThreshold` | S:80 R:85 A:85 D:80 |
| 5 | Certain | Reuse the existing line-by-line/`bufio.Scanner` markdown idiom (cf. `countGrades`) rather than add a markdown library | Determined by Constitution Principle I (single-binary, no new runtime deps); matches existing package style — no alternative interpretation | S:80 R:80 A:95 D:90 |
| 6 | Certain | Test compares doc values against resolved `getExpectedMin`/`getGateThreshold`, not raw map membership | Determined by the code's actual structure: the map omits default types, so comparing raw keys would simply be wrong, not a choice | S:85 R:75 A:95 D:90 |
| 7 | Confident | Test asserts the doc covers exactly the 7 canonical types (catches added/removed/renamed types) | Bidirectional guard is the point of a consistency test; one-directional check would miss type-set drift | S:75 R:75 A:80 D:75 |
| 8 | Confident | Resolve the doc-file path by walking up to the repo root rather than a fixed relative depth | Go test CWD is the package dir; a fragile `../../../../../` breaks if layout shifts; walking up degrades gracefully | S:70 R:80 A:80 D:75 |
| 9 | Confident | Reword the "source of truth" lines in `change-types.md` to make the code canonical and the doc a verified mirror | Today's doc text is self-contradictory ("source of truth" yet "embedded from" code); the decision to reword is clear and reversible — only exact phrasing is an apply-time detail | S:75 R:90 A:80 D:70 |
| 10 | Certain | No current drift to reconcile (code and doc agree at intake time) | Verified by reading both files at intake — an observed fact, not a guess; the test's first run merely confirms it | S:90 R:85 A:90 D:85 |

10 assumptions (6 certain, 4 confident, 0 tentative, 0 unresolved).
