# Plan: Add `.status.yaml` `summary:` field + migration

**Change**: 260615-5943-status-summary-field
**Intake**: `intake.md`

## Requirements

### Statusfile: `summary` optional-string field

#### R1: `StatusFile` struct carries a `summary` field
The `StatusFile` struct in `src/go/fab/internal/statusfile/statusfile.go` SHALL carry a
`Summary string \`yaml:"summary,omitempty"\`` field, modeled on the existing
`ChangeTypeSource` optional-string field.

- **GIVEN** a `.status.yaml` with a `summary: "some text"` key
- **WHEN** it is loaded via `Load()`
- **THEN** `sf.Summary == "some text"` (struct-tag decode picks it up; no manual decode edit)

#### R2: empty `summary` round-trips to absent (drop-when-empty)
`syncToRaw()` SHALL drop the `summary` key when `sf.Summary == ""` (mirroring the
`change_type_source` empty case), so an absent/empty summary serializes to nothing per `omitempty`.

- **GIVEN** a loaded `.status.yaml` whose `summary` is empty (or has no `summary` key)
- **WHEN** it is saved via `Save()`
- **THEN** the serialized file contains no `summary:` line

#### R3: a non-empty `summary` persists and round-trips
`syncToRaw()` SHALL write the value when `sf.Summary != ""`, and SHALL insert the key
(before `last_updated`, via `insertKey`) when absent from a sparse document (mirroring the
`change_type_source` insert-when-absent clause).

- **GIVEN** a loaded `.status.yaml` (including a sparse legacy doc with no `summary` key)
- **WHEN** `sf.Summary` is set to `"the change did X"` and `Save()` is called
- **THEN** reloading yields `Summary == "the change did X"`, and the key lands between the
  existing keys and `last_updated`

### Status package: `SetSummary` helper

#### R4: `status.SetSummary(st, statusPath, text)` sets and persists the summary
A new `SetSummary` function in `src/go/fab/internal/status/status.go` SHALL set
`st.Summary = text` then `Save`, mirroring `SetChangeType`'s save/`last_updated`-touch pattern
(but WITHOUT the `change_type_source: explicit` side effect — that is specific to change-type).

- **GIVEN** a loaded `StatusFile`
- **WHEN** `SetSummary(st, path, "did X")` is called
- **THEN** `st.Summary == "did X"`, the file persists it, and `last_updated` is refreshed

### CLI: `set-summary` + `get-summary` verbs

#### R5: `fab status set-summary <change> <text>` writes the summary
A `statusSetSummaryCmd()` in `src/go/fab/cmd/fab/status.go` SHALL accept `cobra.ExactArgs(2)`,
route through `withStatusLock`, and delegate to `status.SetSummary(st, statusPath, args[1])`
(mirroring `statusSetChangeTypeCmd`). It SHALL be registered in `statusCmd()`'s `AddCommand(...)`.

- **GIVEN** an existing change
- **WHEN** `fab status set-summary <change> "did X"` runs
- **THEN** the change's `.status.yaml` carries `summary: "did X"`

#### R6: `fab status get-summary <change>` prints the summary
A `statusGetSummaryCmd()` SHALL accept `cobra.ExactArgs(1)`, load via the lock-free `loadStatus`
reader, and print `st.Summary` to stdout (mirroring `statusGetIssuesCmd`). An empty summary
prints an empty line (graceful absence — the generator falls back to the slug). It SHALL be
registered in `statusCmd()`'s `AddCommand(...)`.

- **GIVEN** a change whose `.status.yaml` has `summary: "did X"`
- **WHEN** `fab status get-summary <change>` runs
- **THEN** it prints `did X`
- **AND GIVEN** a change with no summary, **THEN** it prints an empty line (exit 0)

### Template + version + migration

#### R7: template seeds `summary: ""`
`src/kit/templates/status.yaml` SHALL carry `summary: ""` between `prs: []` and the
`# true_impact` comment / `last_updated`, documenting the field for humans reading a fresh change.

- **GIVEN** a new change created from the template
- **WHEN** its `.status.yaml` is inspected
- **THEN** a `summary: ""` line sits between `prs: []` and `# true_impact`/`last_updated`

#### R8: VERSION bumps to 2.5.0
`src/kit/VERSION` SHALL be `2.5.0` (a `.status.yaml` schema change is a minor version bump).

- **GIVEN** the current VERSION `2.4.2`
- **WHEN** the change ships
- **THEN** `src/kit/VERSION` reads `2.5.0`

#### R9: migration `2.4.2-to-2.5.0.md` adds `summary: ""` to in-flight changes
A new `src/kit/migrations/2.4.2-to-2.5.0.md` (named for the live VERSION `2.4.2` → target
`2.5.0`, NOT the backlog's stale `2.4.1`) SHALL add `summary: ""` to in-flight
`fab/changes/*/.status.yaml`, skipping `fab/changes/archive/**`, idempotent (skip files already
having a `summary:` key), inserting before `last_updated`. It SHALL follow the existing migration
format (Summary / Pre-check / Changes / Verification).

- **GIVEN** an in-flight change with no `summary:` key
- **WHEN** the migration runs
- **THEN** its `.status.yaml` gains `summary: ""` before `last_updated`
- **AND** re-running is a no-op; archived files are untouched

### Documentation / spec / memory mirrors

#### R10: doc, spec, and memory surfaces document the new field/verbs
The schema-change ripple SHALL update every place the `.status.yaml` schema or `fab status` verbs
are documented: `src/kit/skills/_cli-fab.md` (§ fab status verb table — add `set-summary` /
`get-summary`), `docs/specs/skills/SPEC-fab-status.md` (mirror the verbs), `docs/specs/templates.md`
(document the `summary:` template field), `docs/memory/pipeline/change-lifecycle.md` (note the
field + where it is authored), and `docs/memory/pipeline/schemas.md` (add `summary` to the
`.status.yaml` schema doc).

- **GIVEN** the change is complete
- **WHEN** the documentation is read
- **THEN** each surface above describes the `summary` field / verbs consistently

### Non-Goals

- **No authoring wiring.** This change creates the field + read/write verbs only; no stage
  (hydrate, intake, or any other) auto-populates `summary`. That wiring is a later FKF change.
- **No `log.md` generator.** That is FKF Change 2, which consumes this field.
- **No `fab memory-index` changes.** The generator read happens in Change 2.

### Design Decisions

1. **Model `summary` on `change_type_source`, not a bespoke shape**: optional string,
   `omitempty`, drop-when-empty in `syncToRaw`, insert-when-absent before `last_updated` — *Why*:
   the codebase has an exact existing sibling pattern (statusfile.go lines 112, 435–443, 474–476);
   constitution Principle "follow existing project patterns". — *Rejected*: a required field (would
   break back-compat with existing files) or a separate `log:` block (over-engineered for one line).
2. **`SetSummary` omits the `change_type_source: explicit` side effect**: it sets `Summary` then
   `Save`, period — *Why*: that side effect is specific to the change-type re-inference race;
   `summary` has no inferring hook to guard against. — *Rejected*: blindly copying `SetChangeType`
   wholesale would introduce a meaningless mutation.
3. **Migration named `2.4.2-to-2.5.0.md`** (not the backlog's `2.4.1-to-2.5.0.md`) — *Why*: live
   VERSION is `2.4.2`; migrations are named current→target. — *Rejected*: honoring the stale
   backlog text would orphan the migration (the setup runner keys on the current version).

## Tasks

### Phase 1: Core statusfile field + round-trip

- [x] T001 Add `Summary string \`yaml:"summary,omitempty"\`` to the `StatusFile` struct in `src/go/fab/internal/statusfile/statusfile.go` (after `TrueImpact`, before `LastUpdated`), with a comment mirroring the `ChangeTypeSource` doc-comment style <!-- R1 -->
- [x] T002 In `syncToRaw()` (`src/go/fab/internal/statusfile/statusfile.go`), add a `case "summary":` in the `switch key` loop that drops the key via `dropKeyAt(root, i); i -= 2` when `sf.Summary == ""`, else sets `val.Value = sf.Summary` — mirroring the `change_type_source` case at lines ~435–443 <!-- R2 -->
- [x] T003 In `syncToRaw()`'s post-loop insert-when-absent block, add `if !seen["summary"] && sf.Summary != "" { insertKey(root, "summary", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: sf.Summary}) }` — mirroring the `change_type_source` insert at lines ~474–476 (places the key before `last_updated`) <!-- R3 -->
- [x] T004 [P] Add round-trip tests in `src/go/fab/internal/statusfile/statusfile_test.go` mirroring `TestChangeTypeSource_AbsentDefaultsInferred`, `TestChangeTypeSource_ExplicitRoundTrips`, and `TestSparseFile_ChangeTypeSourceInserts`: (a) absent/empty `summary` does not serialize, (b) a non-empty `summary` round-trips, (c) a sparse-file `summary` is inserted on Save <!-- R1 R2 R3 -->

### Phase 2: Status-package helper

- [x] T005 Add `SetSummary(statusFile *sf.StatusFile, statusPath, text string) error` to `src/go/fab/internal/status/status.go` — sets `statusFile.Summary = text` then `return statusFile.Save(statusPath)` (mirror `SetChangeType`'s save pattern, WITHOUT the `change_type_source` side effect) with a doc-comment <!-- R4 -->
- [x] T006 [P] Add `TestSetSummary_PersistsAndRoundTrips` to `src/go/fab/internal/status/mutators_test.go` (mirror `TestSetChangeType_PersistsValidType`): set a summary, assert in-memory + reloaded value, assert `last_updated` refreshed <!-- R4 -->

### Phase 3: CLI verbs

- [x] T007 Add `statusSetSummaryCmd()` to `src/go/fab/cmd/fab/status.go` (`Use: "set-summary <change> <text>"`, `cobra.ExactArgs(2)`, `withStatusLock` → `status.SetSummary(st, statusPath, args[1])`) — mirror `statusSetChangeTypeCmd` <!-- R5 -->
- [x] T008 Add `statusGetSummaryCmd()` to `src/go/fab/cmd/fab/status.go` (`Use: "get-summary <change>"`, `cobra.ExactArgs(1)`, `loadStatus` reader, `fmt.Println(st.Summary)`) — mirror `statusGetIssuesCmd` but print the single scalar (one `Println`, prints empty line when empty) <!-- R6 -->
- [x] T009 Register `statusSetSummaryCmd()` and `statusGetSummaryCmd()` in the `cmd.AddCommand(...)` list in `statusCmd()` (`src/go/fab/cmd/fab/status.go`) <!-- R5 R6 -->
- [x] T010 [P] Add CLI registration tests to `src/go/fab/cmd/fab/status_test.go` (mirror `TestStatusSetAcceptanceCmd_RegisteredWithExpectedUse` / `TestStatusCmd_RegistersBothChecklistRemovedAndSetAcceptance`): assert `statusSetSummaryCmd().Use` and `statusGetSummaryCmd().Use` prefixes, and that `statusCmd()` registers both `set-summary` and `get-summary` subcommands <!-- R5 R6 -->

### Phase 4: Template, version, migration

- [x] T011 [P] Add `summary: ""` to `src/kit/templates/status.yaml` between `prs: []` and the `# true_impact` comment line <!-- R7 -->
- [x] T012 [P] Bump `src/kit/VERSION` from `2.4.2` to `2.5.0` <!-- R8 -->
- [x] T013 [P] Create `src/kit/migrations/2.4.2-to-2.5.0.md` following the existing migration format (Summary / Pre-check / Changes / Verification): adds `summary: ""` before `last_updated` to in-flight `fab/changes/*/.status.yaml`, skips `fab/changes/archive/**`, idempotent (skip files already having a `summary:` key) <!-- R9 -->

### Phase 5: Documentation / spec / memory mirrors

- [x] T014 [P] Add `set-summary` and `get-summary` rows to the `## fab status` verb table in `src/kit/skills/_cli-fab.md` (near the `set-change-type` / `get-issues` rows) <!-- R10 -->
- [x] T015 [P] Mirror the two new verbs into `docs/specs/skills/SPEC-fab-status.md` <!-- R10 -->
- [x] T016 [P] Document the `summary: ""` template field in `docs/specs/templates.md` (add to the `.status.yaml` template block + a field note) <!-- R10 -->
- [x] T017 [P] Note the new `summary` field and where it is authored (hydrate / carried from intake) in `docs/memory/pipeline/change-lifecycle.md` § Status Tracking <!-- R10 -->
- [x] T018 [P] Add `summary` (optional string) to the `.status.yaml` schema doc in `docs/memory/pipeline/schemas.md` <!-- R10 -->

### Phase 6: Build + verify

- [x] T019 Build (`cd src/go/fab && go build ./...`) and run the scoped test packages (`go test ./internal/statusfile/... ./internal/status/... ./cmd/fab/...`); fix any failures <!-- R1 R2 R3 R4 R5 R6 -->

## Execution Order

- T001 → T002, T003 (struct field before the syncToRaw cases)
- T002, T003 → T004 (round-trip tests need the writer)
- T001 → T005 (SetSummary uses the struct field) → T006
- T005 → T007, T008 → T009 → T010 (CLI delegates to SetSummary; registration after the commands)
- Phases 4 and 5 ([P] tasks) are independent of the Go code and of each other
- T019 runs last (needs all Go code + tests in place)

## Acceptance

### Functional Completeness

- [ ] A-001 R1: The `StatusFile` struct has a `Summary string \`yaml:"summary,omitempty"\`` field and `Load()` decodes a present `summary:` key into it
- [ ] A-002 R2: `syncToRaw()` drops the `summary` key when empty — an empty/absent summary produces no `summary:` line on Save
- [ ] A-003 R3: A non-empty summary persists and round-trips, including insert-when-absent on a sparse document (key lands before `last_updated`)
- [ ] A-004 R4: `status.SetSummary` sets `Summary` and persists with a refreshed `last_updated`, and does NOT touch `change_type_source`
- [ ] A-005 R5: `fab status set-summary <change> <text>` (ExactArgs(2), withStatusLock → SetSummary) is registered and writes the summary
- [ ] A-006 R6: `fab status get-summary <change>` (ExactArgs(1), loadStatus reader) prints `st.Summary`, printing an empty line when absent
- [ ] A-007 R7: `src/kit/templates/status.yaml` carries `summary: ""` between `prs: []` and `# true_impact`/`last_updated`
- [ ] A-008 R8: `src/kit/VERSION` reads `2.5.0`
- [ ] A-009 R9: `src/kit/migrations/2.4.2-to-2.5.0.md` exists, is named for current→target, adds `summary: ""` before `last_updated`, skips `archive/**`, and is idempotent
- [ ] A-010 R10: `_cli-fab.md`, `SPEC-fab-status.md`, `templates.md`, `change-lifecycle.md`, and `schemas.md` all document the new field/verbs consistently

### Behavioral Correctness

- [ ] A-011 R2: An existing `.status.yaml` with no `summary` key, loaded and re-saved, still has no `summary` key (back-compat round-trip preserved)
- [ ] A-012 R6: `get-summary` on a change with no summary exits 0 and emits an empty line (graceful absence, generator falls back to slug)

### Scenario Coverage

- [ ] A-013 R3: A test exercises the sparse-document insert path (summary inserted before `last_updated` on a doc that lacked the key)
- [ ] A-014 R5 R6: Tests assert both new CLI subcommands are registered with the expected `Use` strings

### Edge Cases & Error Handling

- [ ] A-015 R9: Re-running the migration on an already-migrated tree is a complete no-op; `fab/changes/archive/**` files are unchanged

### Code Quality

- [ ] A-016 Pattern consistency: New code follows the `change_type_source` / `SetChangeType` / `get-issues` patterns (naming, error handling, doc-comment style)
- [ ] A-017 No unnecessary duplication: `SetSummary` reuses `Save`; `syncToRaw` reuses `dropKeyAt`/`insertKey`; the CLI reuses `withStatusLock`/`loadStatus` — no reimplementation

### documentation_accuracy

- [ ] A-018 R10: The documented verb signatures and template field match the actual implementation (ExactArgs counts, key placement, `omitempty` behavior)

### cross_references

- [ ] A-019 R10: Cross-references between `_cli-fab.md` and `SPEC-fab-status.md` stay in sync (same two verbs, same signatures); schema docs reference the same field semantics

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Model `summary` on `change_type_source` (omitempty optional string, drop-when-empty in `syncToRaw`, insert-when-absent before `last_updated`) | Exact existing sibling pattern in statusfile.go (lines 112, 435–443, 474–476); template/constitution rule says follow existing patterns. Deterministic. | S:90 R:90 A:100 D:95 |
| 2 | Certain | `set-summary` mirrors `set-change-type` (withStatusLock → status.SetSummary); `get-summary` mirrors `get-issues` (loadStatus reader, single scalar Println) | Direct sibling commands exist (status.go lines 327, 477); copying their shape is the obvious deterministic choice. | S:90 R:90 A:100 D:95 |
| 3 | Confident | `SetSummary` omits the `change_type_source: explicit` side effect that `SetChangeType` performs — it only sets `Summary` + `Save` | That side effect guards the change-type re-inference hook race; `summary` has no inferring hook, so copying it would introduce a meaningless mutation. Clear single interpretation. | S:80 R:85 A:90 D:80 |
| 4 | Confident | Migration named `2.4.2-to-2.5.0.md` (not the backlog's `2.4.1-to-2.5.0.md`); VERSION 2.4.2 → 2.5.0 | Live VERSION is 2.4.2; the backlog line predates the 2.4.2 release; migrations are named current→target. Strong codebase signal, trivially reversible. | S:80 R:90 A:95 D:85 |
| 5 | Confident | Migration writes `summary: ""` into in-flight `fab/changes/*/.status.yaml` (rather than relying solely on omitempty graceful-absence) | Backlog explicitly says "adding the field to in-flight changes under fab/changes/"; idempotent + skips archive/**. omitempty would make a no-write defensible, but the ticket is explicit. | S:80 R:85 A:85 D:70 |
| 6 | Confident | Apply updates the two `docs/memory/` mirrors (`change-lifecycle.md`, `schemas.md`) directly, even though memory is normally hydrate's domain | The intake §6 + the dispatched task summary list both files as explicit apply deliverables; hydrate's merge-without-duplication contract reconciles any later overlap, so doing them now is safe and matches the instruction. | S:75 R:80 A:75 D:70 |
| 7 | Confident | Mirror the new verbs into `SPEC-fab-status.md` (the intake-named SPEC) despite it documenting the `/fab-status` display skill rather than the `fab status` CLI; there is no `SPEC-_cli-fab.md` | The intake explicitly names this file as the mirror target; no dedicated CLI-skill SPEC exists. Add a contextually-appropriate note rather than force a foreign table. Easily revised. | S:70 R:85 A:75 D:65 |

7 assumptions (2 certain, 5 confident, 0 tentative).
