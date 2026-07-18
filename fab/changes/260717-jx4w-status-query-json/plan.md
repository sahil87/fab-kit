# Plan: Status Query `--json`

**Change**: 260717-jx4w-status-query-json
**Intake**: `intake.md`

## Requirements

### fab status: `--json` on the read-only query surface

#### R1: `--json` flag on the nine query subcommands
Each of the nine read-only `fab status` query subcommands — `confidence`, `plan`, `progress-map`, `get-issues`, `get-prs`, `get-summary`, `current-stage`, `display-stage`, `all-stages` — MUST accept a `--json` boolean flag registered as `cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")`, following the `fab dispatch status --json` precedent.

- **GIVEN** any of the nine query subcommands
- **WHEN** invoked with `--json`
- **THEN** it emits a machine-readable JSON document to stdout instead of the bespoke text lines
- **AND** the flag help text is exactly `Output as JSON`

#### R2: Stable per-subcommand JSON shapes
The `--json` output of each subcommand MUST match the intake's per-subcommand shape, with snake_case keys matching the `.status.yaml` field names, ordered lists rendered as JSON arrays (never alphabetized maps), and empty lists as `[]` (never `null`).

- **GIVEN** a change with known status data
- **WHEN** each subcommand is invoked with `--json`
- **THEN** the emitted shapes are:
  - `confidence` → `{"certain":N,"confident":N,"tentative":N,"unresolved":N,"score":F}`
  - `plan` → `{"generated":bool,"task_count":N,"acceptance_count":N,"acceptance_completed":N}`
  - `progress-map` → `[{"stage":"intake","state":"done"},…]` (array, stage order preserved)
  - `display-stage` → `{"stage":"apply","state":"active"}`
  - `current-stage` → `{"stage":"apply"}`
  - `all-stages` → `["intake","apply","review","hydrate","ship","review-pr"]`
  - `get-issues` → `["DEV-988"]`, empty → `[]`
  - `get-prs` → `["https://…/pull/42"]`, empty → `[]`
  - `get-summary` → `{"summary":"…"}`, empty → `{"summary":""}`
- **AND** `progress-map` preserves pipeline stage order (an array, because a Go map would marshal alphabetically)
- **AND** `get-issues`/`get-prs`/`all-stages` emit `[]` (not `null`) when the underlying slice is empty

#### R3: JSON emit mechanics follow the in-repo precedent
Each subcommand MUST emit JSON via `json.NewEncoder(cmd.OutOrStdout())` with `enc.SetIndent("", "  ")` (two-space indent, trailing newline from `Encode`), using named `xxxJSON` struct types with `json:` tags for object shapes (mirroring `dispatchStatusJSON`). No `schema_version` field is emitted — stability is guaranteed by additive-only evolution.

- **GIVEN** an object-shaped subcommand (`confidence`, `plan`, `display-stage`, `current-stage`, `get-summary`)
- **WHEN** rendered with `--json`
- **THEN** it uses a named `xxxJSON` struct with `json:` tags encoded through an indented `json.NewEncoder`
- **AND** no subcommand emits a `schema_version` key

#### R4: `plan --json` preserves the live-acceptance read path
`plan --json` MUST use the same computed values as the text path — including the live-acceptance preference (`status.LiveAcceptance` over the cached `plan.acceptance_*` counter). The acceptance values MUST be computed once and only the rendering branches on the flag.

- **GIVEN** a change whose `plan.md` `## Acceptance` checkbox count differs from the cached `.status.yaml` counter
- **WHEN** `plan` is invoked with and without `--json`
- **THEN** both report the same live-derived `acceptance_count`/`acceptance_completed`
- **AND** the `LiveAcceptance` computation happens once, before the render branch

#### R5: Default (no-flag) text output stays byte-identical
Adding `--json` MUST NOT change the default text output of any touched subcommand. Every existing text emit line stays byte-for-byte identical so existing consumers (e.g. `git-pr.md`'s `get-issues` line parse, any hand parsers) keep working unchanged.

- **GIVEN** any of the nine subcommands invoked without `--json`
- **WHEN** compared against the pre-change output
- **THEN** the emitted bytes are identical (same `key:value\n` / `stage:state\n` / one-item-per-line formats)

### Documentation & Tests

#### R6: `_cli-fab.md` and its SPEC mirror document `--json`
`src/kit/skills/_cli-fab.md` § fab status MUST note the `--json` flag (and its shape) on the nine query rows, and the SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` MUST be updated in the same change (per the constitution's SPEC-mirror rule and the mirror-sweep discipline).

- **GIVEN** the CLI signature gains a `--json` flag on nine subcommands
- **WHEN** the change is reviewed
- **THEN** `_cli-fab.md` § fab status documents `--json` on each query row with its JSON shape
- **AND** `docs/specs/skills/SPEC-_cli-fab.md`'s fab status inventory row reflects the added `--json` query surface

#### R7: Test coverage for every `--json` subcommand
`src/go/fab/cmd/fab/status_test.go` MUST add a `--json` test case per subcommand asserting the emitted shape, the empty-list `[]` behavior (`get-issues`/`get-prs`), the empty-summary `{"summary":""}` behavior, and live-acceptance parity between text and JSON `plan` paths.

- **GIVEN** the nine `--json` subcommands
- **WHEN** the test suite runs
- **THEN** each subcommand has a test exercising its `--json` shape
- **AND** empty-list, empty-summary, and text/JSON live-acceptance parity are covered

### Non-Goals

- Migrating existing consumers (e.g. `git-pr.md`) to `--json` — out of scope; line output keeps working, migration is a later independent change.
- `--json` on `progress-line` (visual decoration, not programmatic data) and `validate-status-file` (its contract is the exit code; it emits no data).
- Any `.status.yaml` schema change or migration — none is needed (no user-data restructuring).

### Design Decisions

1. **Object vs. array per shape**: ordered/list subcommands (`progress-map`, `get-issues`, `get-prs`, `all-stages`) emit JSON arrays; scalar-field subcommands emit named-struct objects — *Why*: a Go `map` marshals keys alphabetically, destroying pipeline stage order, so arrays are the only order-preserving option; objects keep field-set additively extensible — *Rejected*: emitting maps (would alphabetize `progress-map`); emitting a bare string for `get-summary` (blocks additive fields).
2. **No `schema_version` field**: stability is the additive-evolution rule (new fields optional) — *Why*: matches both existing fab `--json` surfaces (`dispatch status`, `config reference`), neither of which carries a version field — *Rejected*: a version field (premature; can be added later without breaking).
3. **Compute-once, branch-on-render for `plan`**: `LiveAcceptance` is resolved before the `if jsonFlag` branch so text and JSON share one source of truth — *Why*: prevents the two rendering paths from drifting — *Rejected*: duplicating the read in each branch.

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add per-subcommand `--json` bool flag + JSON render branch to the five object-shaped query subcommands in `src/go/fab/cmd/fab/status.go` (`confidence`, `plan`, `display-stage`, `current-stage`, `get-summary`), defining named `xxxJSON` struct types with `json:` tags and emitting via `json.NewEncoder(cmd.OutOrStdout())` + `SetIndent("", "  ")`; for `plan`, compute the live-acceptance values once before the render branch. Add the `encoding/json` import. <!-- R1 R2 R3 R4 R5 -->
- [x] T002 Add the `--json` bool flag + JSON render branch to the four list-shaped query subcommands in `src/go/fab/cmd/fab/status.go` (`progress-map` → array of `{stage,state}` objects, `get-issues`/`get-prs` → string arrays with `[]`-not-`null` empty handling, `all-stages` → string array), preserving stage order for `progress-map`. <!-- R1 R2 R3 R5 -->

### Phase 3: Tests

- [x] T003 Add `--json` test cases per subcommand in `src/go/fab/cmd/fab/status_test.go` — assert each shape, empty-list `[]` for `get-issues`/`get-prs`, empty-summary `{"summary":""}` for `get-summary`, and text↔JSON live-acceptance parity for `plan`; run `go test ./src/go/fab/cmd/fab/...`. <!-- R7 -->

### Phase 4: Docs

- [x] T004 [P] Update `src/kit/skills/_cli-fab.md` § fab status — note `--json` and its JSON shape on each of the nine query rows (`current-stage`, `all-stages`, `progress-map`, `display-stage`, `plan`, `confidence`, `get-issues`, `get-prs`, `get-summary`). <!-- R6 -->
- [x] T005 [P] Update the SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` — reflect the added `--json` query surface in the fab status inventory row (mirror-sweep class). <!-- R6 -->

## Execution Order

- T001 and T002 both edit `status.go` (T001 adds the `encoding/json` import) — run T001 before T002 to avoid an import-line conflict.
- T003 depends on T001+T002 (tests exercise the new flags).
- T004, T005 are docs-only and independent (`[P]`), but the whole mirror class must land together.

## Acceptance

### Functional Completeness

- [x] A-001 R1: All nine query subcommands (`confidence`, `plan`, `progress-map`, `get-issues`, `get-prs`, `get-summary`, `current-stage`, `display-stage`, `all-stages`) register a `--json` bool flag with help `Output as JSON`.
- [x] A-002 R2: Each subcommand's `--json` output matches the intake's declared shape (snake_case keys, arrays for ordered/list subcommands, objects for scalar-field subcommands).
- [x] A-003 R3: Object-shaped subcommands use named `xxxJSON` structs with `json:` tags; all emit via indented `json.NewEncoder`; no `schema_version` key is present.
- [x] A-004 R4: `plan --json` reports live-acceptance-derived counts identical to the text path, computed once before the render branch.
- [x] A-005 R6: `_cli-fab.md` § fab status and `SPEC-_cli-fab.md` both document the `--json` query surface.
- [x] A-006 R7: `status_test.go` has a `--json` case per subcommand; `go test ./src/go/fab/cmd/fab/...` passes. *(Review note: the cases live in the sibling `status_json_test.go` — same package, full per-subcommand coverage; the whole `cmd/fab` suite and full module suite pass with `-count=1`.)*

### Behavioral Correctness

- [x] A-007 R5: Default (no-flag) text output of every touched subcommand is byte-identical to before this change. *(Review note: verified empirically — built a baseline binary from HEAD's `status.go` and byte-compared all nine subcommands' no-flag output + exit codes against the changed binary: identical.)*

### Edge Cases & Error Handling

- [x] A-008 R2: `get-issues`/`get-prs` emit `[]` (not `null`) when empty; `get-summary` emits `{"summary":""}` when the summary is empty/absent.
- [x] A-009 R2: `progress-map --json` preserves pipeline stage order (intake→apply→review→hydrate→ship→review-pr), not alphabetized.

### Code Quality

- [x] A-010 Pattern consistency: New code follows the `dispatchStatusJSON` precedent (flag registration, struct naming, encoder mechanics) and surrounding `status.go` conventions.
- [x] A-011 No unnecessary duplication: The `plan` acceptance-value computation is shared between text and JSON paths (not duplicated); existing helpers (`status.ProgressMap`, `status.DisplayStage`, `status.CurrentStage`, `status.AllStages`, `status.LiveAcceptance`) are reused rather than reimplemented.
- [x] A-012 No magic strings: the `--json` flag name/help and JSON key literals match the intake spec and the `.status.yaml` field names.

### Documentation Accuracy

- [x] A-013 R6: The documented JSON shapes in `_cli-fab.md` exactly match what the code emits (keys, array-vs-object, empty-value behavior). *(Review note: verified against live output of the built binary for all nine shapes.)*
- [x] A-014 R6: No edits were made under `.claude/skills/` (canonical source `src/kit/skills/` only). *(Review note: the deployed `_cli-fab` copy contains none of the new `--json` query text.)*

### Cross-References

- [x] A-015 R6: The SPEC mirror `SPEC-_cli-fab.md` is updated in the same change as `_cli-fab.md` (constitution SPEC-mirror rule); the mirror-sweep grep confirms no other doc file restates the changed output line formats (text output is unchanged, so consumer references need no update). *(Review note: re-ran the sweep grep — `naming.md`, `git-pr.md`/`SPEC-git-pr.md`, `fab-continue.md`/`SPEC-fab-continue.md` reference the subcommands only as consumers of the unchanged text output; findings docs are historical records.)*

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- Mirror-sweep: the affected line formats are documented only in `_cli-fab.md` § fab status; `SPEC-_cli-fab.md` mirrors it at inventory granularity. Other files (`git-pr.md`, `naming.md`, `SPEC-git-pr.md`, `docs/memory/**`) reference the subcommands as consumers of the unchanged text output and need no update.

## Deletion Candidates

- `src/go/fab/cmd/fab/dispatch_status.go:69-71`, `helpdump.go:49`, `memory_index.go:234`, `panemap.go:518`, `pane_capture.go:90`, `pane_process.go:132` — six pre-existing inline `json.NewEncoder` + `SetIndent("", "  ")` blocks in package `main` are now byte-duplicates of the new shared `encodeJSON` helper (`status.go`); consolidation candidates (not unused code — each still runs), a follow-up cleanup outside this change's diff.

Otherwise: None — this change adds new functionality without making existing code redundant (no existing consumer, branch, or config became unused; consumer migration to `--json` is an explicit Non-Goal).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Cover nine subcommands (backlog's six + `current-stage`/`display-stage`/`all-stages`); exclude `progress-line` and `validate-status-file` | Backlog says "across the status query surface" and its line refs include `display-stage`; principle №2 scopes the MUST to programmatic output; later additions are additive | S:70 R:85 A:80 D:65 |
| 2 | Confident | JSON shapes: snake_case keys matching `.status.yaml` fields; `progress-map` as ordered array of `{stage,state}`; bare arrays for `get-issues`/`get-prs`/`all-stages`; `get-summary` object-wrapped; empty lists `[]` never `null` | Field names copied from the status file schema; Go map marshaling alphabetizes, so arrays preserve stage order; object wrapper keeps summary additively extensible | S:60 R:70 A:85 D:60 |
| 3 | Certain | Emit mechanics: per-subcommand `--json` bool flag, named `xxxJSON` structs, `json.NewEncoder` + two-space `SetIndent` | Determined by the shipped `fab dispatch status --json` precedent (`dispatchStatusJSON`) — follow existing project patterns | S:65 R:90 A:95 D:90 |
| 4 | Confident | No `schema_version` field; stability = additive-only evolution | Matches both existing fab `--json` surfaces; principle №2's versioning obligation is satisfied by the additive rule; a version field can be added later without breaking | S:55 R:60 A:80 D:70 |
| 5 | Certain | Default text output stays byte-identical; `--json` is purely additive | Back-compat with `git-pr.md`'s `get-issues` consumer and any hand parsers; the backlog asks to *add* `--json`, not to change existing output | S:80 R:70 A:95 D:95 |
| 6 | Confident | Mirror-sweep class = `_cli-fab.md` § fab status + `SPEC-_cli-fab.md` only; other subcommand references are consumers of unchanged text and need no edit | Grep of `src/`/`docs/` shows no other file restates the emitted output line formats; the SPEC mirror documents fab status at inventory granularity, not per-line, so the row-level `--json` note satisfies the constitution's strict CLI SPEC-mirror reading | S:70 R:80 A:75 D:65 |

6 assumptions (2 certain, 4 confident, 0 tentative).
