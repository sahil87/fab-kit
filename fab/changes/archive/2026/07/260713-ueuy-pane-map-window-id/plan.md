# Plan: Stable window_id in pane map JSON

**Change**: 260713-ueuy-pane-map-window-id
**Intake**: `intake.md`

## Requirements

### Pane Map: Stable window identity in JSON

#### R1: `tmuxPaneFormat` carries `#{window_id}` as the trailing field
The `tmuxPaneFormat` string SHALL append `\t#{window_id}` after `#{@rk_agent_state}`, making `window_id` the seventh and trailing field. Both the `tmuxPaneFormat` doc comment and the `parsePaneLines` doc comment SHALL be updated to describe the seven-field format and the new invariant: `#{@rk_agent_state}` is now a possibly-empty MIDDLE field and `#{window_id}` is the never-empty TRAILING field.

- **GIVEN** `fab pane map` runs `tmux list-panes -F <tmuxPaneFormat>`
- **WHEN** tmux emits a pane line
- **THEN** the line carries seven tab-separated fields ending in the window's `#{window_id}` (e.g. `@5`)
- **AND** the doc comments describe seven fields with `#{window_id}` trailing and never empty

#### R2: `parsePaneLines` parses the seventh field with graded legacy tolerance
`parsePaneLines` SHALL use `strings.SplitN(line, "\t", 7)` and populate `windowID` from `parts[6]` when seven fields are present, retaining the existing tolerance for shorter legacy lines. The per-line newline-only trim (`strings.Trim(line, "\r\n")`, never `TrimSpace`) SHALL be preserved.

- **GIVEN** a parsed pane line
- **WHEN** the line has seven fields
- **THEN** `agentState = TrimSpace(parts[5])` and `windowID = parts[6]`
- **AND WHEN** the line has six fields (legacy) **THEN** `agentState = TrimSpace(parts[5])` and `windowID = ""`
- **AND WHEN** the line has five fields (legacy) **THEN** both `agentState` and `windowID` are `""`
- **AND WHEN** the line has fewer than five fields **THEN** the line is skipped
- **AND WHEN** a seven-field line has an empty agent-state middle field (`"...\t3\t\t@5"`) **THEN** `agentState = ""` and `windowID = "@5"`

#### R3: `windowID` is threaded through `paneEntry`, `paneRow`, and `resolvePane`
`paneEntry` and `paneRow` SHALL each gain a `windowID string` field (raw `#{window_id}` value, `""` when absent). `resolvePane` SHALL copy `p.windowID` into the returned `paneRow` in BOTH the non-git early-return branch and the git branch — a window ID exists regardless of git/fab context.

- **GIVEN** a `paneEntry` with a populated `windowID`
- **WHEN** `resolvePane` resolves it in the git branch
- **THEN** the returned `paneRow.windowID` equals the entry's `windowID`
- **AND WHEN** `resolvePane` resolves a non-git pane (`wtRoot == ""`) **THEN** the returned `paneRow.windowID` still equals the entry's `windowID`

#### R4: `paneJSON` emits a nullable `window_id` immediately after `window_index`
`paneJSON` SHALL gain a `WindowID *string ` + "`json:\"window_id\"`" + ` field positioned immediately after `WindowIndex`. `printPaneJSON` SHALL populate it via `toNullable(r.windowID)` — a non-empty raw window ID surfaces as a string; an empty value surfaces as JSON `null`. No existing JSON key SHALL be renamed, removed, or reordered.

- **GIVEN** a `paneRow` with `windowID == "@5"`
- **WHEN** `printPaneJSON` marshals it
- **THEN** the JSON object carries `"window_id": "@5"` immediately after `"window_index"`
- **AND WHEN** `windowID == ""` **THEN** the JSON object carries `"window_id": null`
- **AND** every pre-existing JSON key (`session`, `window_index`, `pane`, `tab`, `worktree`, `repo`, `change`, `stage`, `display_state`, `agent_state`, `agent_idle_duration`, `pr_url`, `pr_number`) is unchanged in name and order

#### R5: Human table output is byte-identical
`printPaneTable` SHALL NOT gain a `window_id` column. The table column set SHALL remain `Session` (all-sessions only), `Pane`, `WinIdx`, `Tab`, `Worktree`, `Change`, `Stage`, `Agent`. Rendering identical rows with `windowID` set versus cleared SHALL produce byte-identical table output.

- **GIVEN** two `paneRow` slices identical except one has `windowID` populated and the other cleared
- **WHEN** `printPaneTable` renders each
- **THEN** the two outputs are byte-identical
- **AND** the header contains no `window_id`/`WinID` column

#### R6: CLI reference documents `window_id`
`src/kit/skills/_cli-fab.md` (the `fab pane map` `--json` flag row) SHALL add `window_id` to the snake_case JSON field list plus a one-clause contract note: `string|null`, the tmux `@N` window ID, stable across `swap-window`/`move-window`, `null` when unavailable, `--json` only with no table column.

- **GIVEN** the `_cli-fab.md` `fab pane map --json` row
- **WHEN** a reader consults the documented JSON field list
- **THEN** `window_id` appears in the field list with its `string|null` contract note

### Non-Goals

- No CLI flag changes (`--json`, `--session`, `--all-sessions`, `--server` unchanged).
- No new table column — this is a JSON-contract change only.
- No `@N` validation or integer parse of the window ID — it is stored/emitted as an opaque raw string; "unparsed" means empty.
- No CHANGELOG edit or version-bump commit — releases are minted separately via `just release`.
- `docs/memory/runtime/pane-commands.md` and `docs/memory/distribution/kit-architecture.md` are hydrate-stage artifacts, NOT edited during apply.
- No edits under `.claude/skills/` (gitignored deployed copies).

### Design Decisions

1. **`#{window_id}` trailing, `#{@rk_agent_state}` demoted to a middle field**: append the never-empty `window_id` last — *Why*: `@rk_agent_state` can be empty and its trailing position required a newline-only-trim invariant to preserve a trailing empty field; a never-empty trailing field simplifies the invariant while keeping additive positional compatibility for legacy 5/6-field lines — *Rejected*: inserting `window_id` mid-format, which would reorder existing field indices and break the graded legacy tolerance.
2. **Nullable `*string` via `toNullable`**: emit `window_id` as `*string` populated through the existing `toNullable` helper — *Why*: matches the file's established nullable convention (`repo`/`change`/`stage`/`display_state`/`pr_url`) and the additive `--json`-only precedent — *Rejected*: a non-nullable `string` that emits `""`, which would diverge from the file's null-for-unresolved contract.

## Tasks

### Phase 2: Core Implementation

- [x] T001 In `src/go/fab/cmd/fab/panemap.go`, append `\t#{window_id}` to the end of `tmuxPaneFormat` and update the `tmuxPaneFormat` doc comment to describe the seven-field format with `#{window_id}` trailing/never-empty and `#{@rk_agent_state}` as a middle field <!-- R1 -->
- [x] T002 In `src/go/fab/cmd/fab/panemap.go`, add `windowID string` to `paneEntry` (with doc-comment update) and to `paneRow` <!-- R3 -->
- [x] T003 In `src/go/fab/cmd/fab/panemap.go`, change `parsePaneLines` to `SplitN(line, "\t", 7)`, populate `windowID = parts[6]` when `len(parts) == 7`, keep `agentState = TrimSpace(parts[5])` for `len(parts) >= 6`, retain the `<5` skip and the newline-only trim, and update the `parsePaneLines` doc comment to the seven-field narrative <!-- R2 -->
- [x] T004 In `src/go/fab/cmd/fab/panemap.go`, thread `windowID: p.windowID` into the `paneRow` returned by both the non-git early-return branch and the git branch of `resolvePane` <!-- R3 -->
- [x] T005 In `src/go/fab/cmd/fab/panemap.go`, add `WindowID *string ` + "`json:\"window_id\"`" + ` to `paneJSON` immediately after `WindowIndex`, and populate it in `printPaneJSON` via `toNullable(r.windowID)` <!-- R4 -->

### Phase 3: Tests

- [x] T006 In `src/go/fab/cmd/fab/panemap_test.go`, extend `TestParsePaneLines` with: a seven-field line (windowID populated); a seven-field line with an empty agent-state middle field (`"...\t3\t\t@5"` → `agentState == ""`, `windowID == "@5"`); a legacy six-field line (`windowID == ""`, agentState present); confirm the existing five-field / `<5`-field cases still hold with `windowID == ""` <!-- R2 -->
- [x] T007 In `src/go/fab/cmd/fab/panemap_test.go`, add JSON-output coverage: `window_id` present and correct when `windowID` is set; `null` when `windowID` is empty; `window_id` key positioned immediately after `window_index` <!-- R4 -->
- [x] T008 In `src/go/fab/cmd/fab/panemap_test.go`, add `resolvePane` coverage asserting `windowID` is threaded through in BOTH the git and non-git branches <!-- R3 -->
- [x] T009 In `src/go/fab/cmd/fab/panemap_test.go`, add a table-unchanged assertion: identical rows with `windowID` set vs cleared render byte-identical tables and the header carries no `window_id` column <!-- R5 -->

### Phase 4: Docs

- [x] T010 In `src/kit/skills/_cli-fab.md` (the `fab pane map` `--json` flag row, ~line 420), add `window_id` to the snake_case field list and a one-clause contract note (`string|null`; tmux `@N` window ID; stable across swap-window/move-window; `null` when unavailable; `--json` only, no table column) <!-- R6 -->

## Execution Order

- T001–T005 (source) precede the tests that exercise them; within source, T002 (struct fields) precedes T004 (which sets them) and T005 (which reads `r.windowID`), and T003 (parser) depends on T002's `paneEntry.windowID` field.
- T006–T009 (tests) run after the source changes compile.
- T010 (docs) is independent of the Go changes.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `tmuxPaneFormat` ends in `\t#{window_id}` (seven fields) and both the `tmuxPaneFormat` and `parsePaneLines` doc comments describe the seven-field / trailing-never-empty invariant
- [x] A-002 R2: `parsePaneLines` uses `SplitN(..., 7)` and populates `windowID` from `parts[6]` on seven-field lines while retaining the 6/5/`<5` graded tolerance and the newline-only trim
- [x] A-003 R3: `paneEntry` and `paneRow` carry a `windowID string` field and `resolvePane` threads `p.windowID` through both branches
- [x] A-004 R4: `paneJSON` emits `window_id` (`*string`, `json:"window_id"`) immediately after `window_index`, populated via `toNullable`
- [x] A-005 R6: `src/kit/skills/_cli-fab.md` documents `window_id` in the `fab pane map --json` field list with its `string|null` contract note

### Behavioral Correctness

- [x] A-006 R4: JSON output carries `"window_id": "@N"` for a populated window ID and `"window_id": null` for an empty one, with no other key renamed/removed/reordered
- [x] A-007 R5: `printPaneTable` output is byte-identical whether `windowID` is set or cleared, and the header gains no column

### Scenario Coverage

- [x] A-008 R2: a test exercises a seven-field line, a seven-field line with an empty agent-state middle field, and legacy six/five-field lines (all with correct `windowID`)
- [x] A-009 R3: a test asserts `windowID` is threaded through `resolvePane` in both the git and non-git branches
- [x] A-010 R4: a test asserts `window_id` is present/correct when set, `null` when empty, and positioned immediately after `window_index`

### Edge Cases & Error Handling

- [x] A-011 R2: a legacy line with an empty trailing agent-state field is still preserved by the newline-only trim (no regression of the existing invariant)

### Code Quality

- [x] A-012 Pattern consistency: new code follows the naming and structural patterns of the surrounding `panemap.go` (nullable `*string`+`toNullable`, table-driven tests, field-threading style)
- [x] A-013 No unnecessary duplication: the existing `toNullable` helper is reused rather than reimplemented; no new helper introduced
- [x] A-014 Readability over cleverness: the parser's graded tolerance stays explicit and legible (Code Quality § Principles)
- [x] A-015 No god functions / magic strings: `parsePaneLines` remains focused; `#{window_id}` lives in the format constant, not scattered literals (Code Quality § Anti-Patterns)
- [x] A-016 CLI ⇒ docs + tests: the Go command-surface change ships with the `_cli-fab.md` update and test updates (Code Quality § Anti-Patterns, constitution CLI constraint)

### documentation_accuracy

- [x] A-017 The `_cli-fab.md` `window_id` note accurately describes the emitted contract (`string|null`, raw tmux `@N`, `--json` only, no table column) and matches the implementation
- [x] A-018 The `tmuxPaneFormat` and `parsePaneLines` doc comments accurately describe the seven-field format after the change (no stale "six fields" / "trailing @rk_agent_state" text)

### cross_references

- [x] A-019 `docs/specs/skills/SPEC-_cli-fab.md` is verified: its `fab pane` row summarizes the command without restating JSON field names, so no SPEC edit is required (mirror class confirmed in sync)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- `docs/memory/runtime/pane-commands.md` and `docs/memory/distribution/kit-architecture.md` updates land at hydrate, not apply.

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (The parser's legacy 6/5-field tolerance and the newline-only trim were explicitly required to be preserved by the intake, and the never-empty trailing `#{window_id}` does not obsolete either — legacy six-field lines still need the trailing-tab protection.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `#{window_id}` appended as the trailing field of `tmuxPaneFormat`; both doc comments rewritten to the seven-field / never-empty-trailing narrative | Specified verbatim in the intake with rationale, verified against panemap.go:165-171 | S:95 R:85 A:95 D:95 |
| 2 | Certain | `parsePaneLines` uses `SplitN(line, "\t", 7)` with graded tolerance (7→both, 6→agentState only, 5→neither, <5→skip); newline-only trim kept | Specified verbatim; mirrors the existing len==6/len==5 tolerance at panemap.go:243-251 | S:95 R:85 A:90 D:90 |
| 3 | Certain | `window_id` emitted as `WindowID *string` immediately after `WindowIndex` via `toNullable`; no table column; `window_index` retained | Specified verbatim; matches the file's nullable convention and the repo/display_state/pr_url `--json`-only precedent | S:95 R:90 A:95 D:95 |
| 4 | Certain | `resolvePane` threads `p.windowID` through BOTH branches (non-git panes too) | Specified verbatim; consistent with the existing axis-independence comment in the non-git branch | S:95 R:90 A:95 D:95 |
| 5 | Certain | `windowID` stored/emitted verbatim as a raw string (e.g. `@5`) — no validation/int-parse; "unparsed" means empty | Intake says "null when empty/unparsed"; the field is an opaque tmux identifier, same treatment as the raw agentState option | S:75 R:90 A:90 D:85 |
| 6 | Certain | Tests extend the existing table-driven suites per the intake's enumerated matrix; run scoped `go test ./cmd/fab/` first, then `just test` | Test matrix specified verbatim; runner convention verified in code-quality.md § Test Strategy | S:90 R:95 A:95 D:95 |
| 7 | Confident | `docs/specs/skills/SPEC-_cli-fab.md` needs no content change; apply verifies the mirror class rather than assuming | Verified by grep — the SPEC `fab pane` row (line 27) does not restate JSON field names and contains no `window_index`/`window_id`; residual uncertainty only because reviewers read the mirror rule strictly | S:60 R:95 A:80 D:80 |

7 assumptions (6 certain, 1 confident, 0 tentative).
