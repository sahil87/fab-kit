# Plan: Add PR URL and Number to `fab pane map` JSON Output

**Change**: 260609-r7ju-pane-map-pr-fields
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### Pane Map JSON: PR Fields

#### R1: Source the PR URL from the already-loaded status file
`resolvePane` SHALL read the PR URL from the `StatusFile` already loaded for stage
derivation (the existing `sf.Load(statusPath)` call), without any second `.status.yaml`
read, subprocess, or network call. The `prURL` SHALL be the LAST entry of
`statusFile.PRs` (most recent), and `""` when the list is absent, empty, or the pane has
no resolvable fab change.

- **GIVEN** a pane whose active change `.status.yaml` has a `prs:` list of two URLs
- **WHEN** `resolvePane` loads the status file (the same load used for `stage`)
- **THEN** the resolved `paneRow.prURL` is the LAST URL in the list
- **AND** no additional file read, `gh`/`git` subprocess, or network call is performed

- **GIVEN** a pane whose `.status.yaml` has an empty or absent `prs:` list (or a pane with
  no fab change / no fab dir / unresolved git)
- **WHEN** `resolvePane` resolves the row
- **THEN** `paneRow.prURL` is `""`

#### R2: `paneRow` carries only the PR URL string
The `paneRow` struct SHALL gain exactly one new field — `prURL string` — keeping it
string-only and consistent with its sibling fields. The PR number SHALL NOT be stored on
`paneRow`; it is derived at the JSON boundary (R4).

- **GIVEN** the `paneRow` struct
- **WHEN** the PR fields are added
- **THEN** only `prURL string` is added (no `*int` PR-number field on `paneRow`)

#### R3: `parsePRNumber` extracts the PR number from a GitHub PR URL
A pure helper `parsePRNumber(url string) (int, bool)` SHALL extract the PR number from the
trailing `/pull/<n>` segment. It SHALL return `(n, true)` when a numeric segment follows
`/pull/`, tolerating a trailing path; otherwise `(0, false)`.

- **GIVEN** the URL `https://github.com/org/repo/pull/42`
- **WHEN** `parsePRNumber` is called
- **THEN** it returns `(42, true)`

- **GIVEN** the URL `https://github.com/org/repo/pull/42/files` (trailing path)
- **WHEN** `parsePRNumber` is called
- **THEN** it returns `(42, true)`

- **GIVEN** a URL with no `/pull/<n>` segment, a non-numeric segment (`/pull/abc`), or an
  empty string
- **WHEN** `parsePRNumber` is called
- **THEN** it returns `(0, false)`

#### R4: `paneJSON` exposes `pr_url` and `pr_number` as nullable fields
The `paneJSON` struct SHALL gain two fields appended after the existing ones —
`PRURL *string \`json:"pr_url"\`` and `PRNumber *int \`json:"pr_number"\`` — following the
existing `*string` + `toNullable` nil convention. In `printPaneJSON`, `prURL` SHALL map via
`toNullable` to `PRURL` (null when empty), and `parsePRNumber(r.prURL)` SHALL set
`PRNumber` to `&n` on success and `nil` otherwise.

- **GIVEN** a row whose `prURL` is `https://github.com/org/repo/pull/42`
- **WHEN** `printPaneJSON` marshals it
- **THEN** `pr_url` is the URL string and `pr_number` is `42`

- **GIVEN** a row whose `prURL` is `""`
- **WHEN** `printPaneJSON` marshals it
- **THEN** both `pr_url` and `pr_number` are JSON `null`

- **GIVEN** a row whose `prURL` is a malformed URL with no `/pull/<n>`
- **WHEN** `printPaneJSON` marshals it
- **THEN** `pr_url` is the URL string and `pr_number` is JSON `null`

#### R5: Table output is unchanged
`printPaneTable` SHALL NOT be modified — no new columns. The PR fields are JSON-only. A
test SHALL assert the table output is byte-identical to the pre-change output (no new
columns appear).

- **GIVEN** the same set of rows as before the change
- **WHEN** `printPaneTable` renders them
- **THEN** the output contains exactly the existing columns (no `pr_url`/`pr_number`)

#### R6: Docs reflect the new JSON fields
The `src/kit/skills/_cli-fab.md` `fab pane map --json` reference SHALL document the two
new fields, their types (`pr_url: string|null`, `pr_number: number|null`), null semantics,
and the explicit non-goal (no PR status, no network). The kit SOURCE is edited, never the
deployed `.claude/skills/` copy.

- **GIVEN** the `_cli-fab.md` `fab pane map` `--json` row
- **WHEN** the docs are updated
- **THEN** it lists `pr_url` and `pr_number` with types, null semantics, and the
  filesystem-only / no-network boundary

### Non-Goals

- PR *status* (open/merged/closed, CI state) — out of scope; run-kit fetches live status.
- Any `gh`/`git` subprocess or network call — `fab pane map` stays filesystem-only.
- A second `.status.yaml` read — the PR URL is sourced from the existing load.
- Storing the PR number on `paneRow` — it is parsed at the JSON boundary.
- New statusfile/status package accessors — `StatusFile.PRs` is used directly.
- Table output columns for the PR fields.

### Design Decisions

1. **Parse the PR number at the JSON boundary, not on `paneRow`**: `paneRow` carries only
   `prURL string`; `parsePRNumber` runs in `printPaneJSON` producing a `*int`. — *Why*:
   keeps `paneRow` string-only and consistent with its siblings; a nil `*int` cleanly
   represents both "no URL" and "unparseable URL" with no sentinel. — *Rejected*: a `*int`
   field on `paneRow` (identical JSON, but breaks the string-only convention).
2. **Source `prURL` from the existing `sf.Load` block**: read `statusFile.PRs` inside the
   same success block that derives `stage`. — *Why*: zero-cost addition at a seam already
   paid for; no new I/O. — *Rejected*: a second load or a new statusfile accessor.
3. **Use `StatusFile.PRs` directly**: it is already a public field. — *Why*: no new accessor
   warranted. — *Rejected*: a convenience accessor in the statusfile package.

## Tasks

### Phase 1: Core Implementation

- [x] T001 Add `prURL string` field to the `paneRow` struct in `src/go/fab/cmd/fab/panemap.go` (with a comment matching sibling field style). <!-- R2 -->
- [x] T002 Add `PRURL *string \`json:"pr_url"\`` and `PRNumber *int \`json:"pr_number"\`` to the `paneJSON` struct in `src/go/fab/cmd/fab/panemap.go`, appended after the existing fields. <!-- R4 -->
- [x] T003 Add the `parsePRNumber(url string) (int, bool)` helper to `src/go/fab/cmd/fab/panemap.go` (parses the trailing `/pull/<n>` segment; tolerates trailing path; returns `(0, false)` for no/empty/non-numeric segment). <!-- R3 -->

### Phase 2: Integration

- [x] T004 In `resolvePane` (`src/go/fab/cmd/fab/panemap.go`), inside the existing `if statusFile, err := sf.Load(statusPath); err == nil { ... }` block, set `prURL = statusFile.PRs[n-1]` when `len(statusFile.PRs) > 0`; populate `paneRow.prURL` in the returned row. No second read; do not parse the number here. <!-- R1 -->
- [x] T005 In `printPaneJSON` (`src/go/fab/cmd/fab/panemap.go`), map `prURL` via `toNullable` to `PRURL`, run `parsePRNumber(r.prURL)` and set `PRNumber` to `&n` on success / `nil` otherwise. <!-- R4 -->

### Phase 3: Tests & Docs

- [x] T006 [P] Add `TestParsePRNumber` to `src/go/fab/cmd/fab/panemap_test.go` covering: canonical URL → `(42, true)`, trailing path → `(42, true)`, no `/pull/` → `(0, false)`, non-numeric (`/pull/abc`) → `(0, false)`, empty string → `(0, false)`. <!-- R3 -->
- [x] T007 [P] Add `printPaneJSON` test cases to `src/go/fab/cmd/fab/panemap_test.go`: row with a valid PR URL → `pr_url` set + `pr_number` parsed; empty `prURL` → both null; malformed URL → `pr_url` set + `pr_number` null. Add `pr_url`/`pr_number` to the snake_case-field assertion. <!-- R4 -->
- [x] T008 [P] Add a `resolvePane` test to `src/go/fab/cmd/fab/panemap_test.go` using a real git repo + fab dir + `.status.yaml` fixture (2-URL `prs:` list) asserting `paneRow.prURL` is the LAST URL; plus an empty/absent `prs:` fixture asserting `prURL == ""`. <!-- R1 -->
- [x] T009 [P] Add a table byte-identity assertion to `src/go/fab/cmd/fab/panemap_test.go`: render rows with `prURL` populated through `printPaneTable` and assert the output contains no `pr_url`/`pr_number` and exactly the existing columns. <!-- R5 -->
- [x] T010 Update `src/kit/skills/_cli-fab.md` `fab pane map --json` row to document `pr_url`/`pr_number`, their types, null semantics, and the no-PR-status / no-network non-goal. <!-- R6 -->

## Execution Order

- T001, T002, T003 (Phase 1) precede T004, T005 (Phase 2) — the structs and helper must exist first.
- T004 and T005 both depend on Phase 1; T005 also depends on T003.
- Phase 3 tests (T006-T009) depend on Phase 1-2 implementation; they are mutually `[P]`.
- T010 (docs) is independent and may run any time after Phase 2.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `resolvePane` sets `prURL` to the LAST entry of `statusFile.PRs` from the already-loaded status file, with no second read / subprocess / network call. (panemap.go:326-328, inside the existing `sf.Load` block)
- [x] A-002 R2: `paneRow` gains exactly one new field, `prURL string` (no PR-number field). (panemap.go:54)
- [x] A-003 R3: `parsePRNumber` extracts the number from `/pull/<n>` and returns `(0, false)` for no/empty/non-numeric segments. (panemap.go:383-398)
- [x] A-004 R4: `paneJSON` exposes `pr_url` (`*string`) and `pr_number` (`*int`); `printPaneJSON` maps via `toNullable` + `parsePRNumber`. (panemap.go:364-365, 420-436)
- [x] A-005 R5: `printPaneTable` is unchanged — no new columns. (panemap.go:445-500 untouched; verified by TestPrintPaneTablePRFieldsUnchanged)
- [x] A-006 R6: `src/kit/skills/_cli-fab.md` documents `pr_url`/`pr_number`, types, null semantics, and the non-goal. (_cli-fab.md:161)

### Behavioral Correctness

- [x] A-007 R1: With a 2-URL `prs:` fixture, `pr_url` resolves to the most recent (last) URL. (TestResolvePanePRURL/prURL_is_the_last_entry — PASS)
- [x] A-008 R4: An empty/absent `prs:` list yields both `pr_url` and `pr_number` as JSON null. (TestPrintPaneJSONPRFields/empty_PR_URL_yields_both_null + TestResolvePanePRURL absent/empty — PASS)
- [x] A-009 R4: A malformed PR URL yields `pr_url` set but `pr_number` null. (TestPrintPaneJSONPRFields/malformed_PR_URL — PASS)

### Scenario Coverage

- [x] A-010 R3: `TestParsePRNumber` covers canonical, trailing-path, no-`/pull/`, non-numeric, and empty-string cases. (panemap_test.go:801-824; also adds trailing-slash-only — all PASS)
- [x] A-011 R4: `printPaneJSON` tests cover valid-URL, empty-URL, and malformed-URL rows, plus snake_case field presence for `pr_url`/`pr_number`. (TestPrintPaneJSONPRFields — PASS)
- [x] A-012 R1: A `resolvePane` test using a git+fab+`.status.yaml` fixture asserts `prURL` is the last URL (and `""` for empty/absent `prs:`). (TestResolvePanePRURL — PASS)

### Edge Cases & Error Handling

- [x] A-013 R1: Panes with no fab change / no fab dir / unresolved git leave `prURL` empty (both JSON fields null). (prURL initialized to "" at panemap.go:317; only set inside the fabDir+folderName+load success path)
- [x] A-014 R3: Empty-string URL into `parsePRNumber` returns `(0, false)` — empty URL naturally yields both-null without a special case. (panemap.go:385-388; TestParsePRNumber/empty_string — PASS)

### Code Quality

- [x] A-015 Pattern consistency: New fields/helper follow `panemap.go` naming (snake_case JSON tags, camelCase Go fields), the `toNullable` idiom, and comment density of surrounding code. (verified against `Repo`/`Change`/`Stage` siblings)
- [x] A-016 No unnecessary duplication: `StatusFile.PRs` and `toNullable` are reused; no new statusfile accessor and no second `.status.yaml` read are introduced. (verified: `statusFile.PRs` used directly; no new `internal/statusfile` change)
- [x] A-017 Readability: `parsePRNumber` is a small focused function (well under the 50-line god-function threshold) with no magic strings beyond the documented `/pull/` segment marker. (16 lines; `marker` is a named const)

### Documentation Accuracy

- [x] A-018 R6: The `_cli-fab.md` JSON field list and types match the actual `paneJSON` struct tags and null semantics. (`pr_url: string|null` ↔ `PRURL *string`; `pr_number: number|null` ↔ `PRNumber *int`)

### Cross-References

- [x] A-019 R6: The `_cli-fab.md` edit targets the kit SOURCE (`src/kit/skills/_cli-fab.md`), not the deployed `.claude/skills/` copy. (git diff confirms `src/kit/skills/_cli-fab.md` modified)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- `docs/memory/runtime/pane-commands.md` is updated at hydrate, not during apply.

## Deletion Candidates

- None — this change adds new functionality (two JSON fields, one pure helper, one read at an existing seam) without making any existing code redundant or unused. The `toNullable` extension (adding the `""` case) widens an existing helper rather than replacing anything; all prior callers (`repo`/`change`/`stage`) keep using it unchanged.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Source `pr_url`/`pr_number` from the `statusFile` already loaded in `resolvePane`; no second read. | Explicit hard requirement in the intake; `sf.Load` at the stage seam is the verified source. | S:98 R:90 A:95 D:98 |
| 2 | Certain | Use `StatusFile.PRs []string` directly — no new accessor. | Verified public at `statusfile.go:105`; intake's "add accessor if absent" resolves to absent-of-need. | S:95 R:85 A:98 D:95 |
| 3 | Certain | `pr_url` = LAST entry of `prs:`; null when absent/empty. | Explicit requirement; matches `AddPR` append-order semantics. | S:98 R:88 A:95 D:98 |
| 4 | Certain | Reuse `toNullable` / `*string` / `*int` nil convention for JSON null. | Explicit requirement; `paneJSON` already uses it for `repo`/`change`/`stage`. | S:98 R:90 A:98 D:98 |
| 5 | Certain | Table output unchanged — JSON-only fields; assert byte-identity. | Explicit requirement; `printPaneTable` is independent of `paneJSON`. | S:98 R:92 A:95 D:98 |
| 6 | Certain | No `gh`/`git`/network; filesystem-only and poll-free. | Explicit, emphatic intake requirement; current code reads only the local status file. | S:99 R:85 A:98 D:99 |
| 7 | Confident | `pr_number` parsed from trailing `/pull/<n>`; null when no URL or unparseable; `pr_url` still set on malformed URL. | Explicit requirement; `…/pull/<n>` is the repo convention. Trailing-path / non-numeric handling is the obvious robust default. | S:90 R:80 A:88 D:85 |
| 8 | Confident | Document new fields + non-goal in `src/kit/skills/_cli-fab.md` (kit source). | Constitution requires CLI output changes update `_cli-fab.md`; the `fab pane map --json` row is the confirmed location. | S:88 R:85 A:88 D:82 |
| 9 | Certain | Parse `pr_number` in `printPaneJSON` (Q1 option b), not a `*int` on `paneRow`. | Clarified in intake — user chose parse-at-JSON-boundary; identical JSON, keeps `paneRow` string-only. | S:95 R:88 A:80 D:95 |
| 10 | Confident | `resolvePane` test uses a real git repo + fab dir + `.status.yaml` fixture (mirrors `initGitRepo` / `TestMainRootForPane` style). | `resolvePane` reads the filesystem; the existing test helpers establish the fixture pattern to follow. | S:80 R:88 A:85 D:80 |

10 assumptions (7 certain, 3 confident, 0 tentative).
