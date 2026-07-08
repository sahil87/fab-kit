# Plan: Config Reference Metadata Table

**Change**: 260708-ff2v-config-reference-metadata-table
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md + fab/plans/sahil/config-upgrade.md Change 1. This is
     Change 1 of 3 in the config-upgrade effort; the schema this introduces is
     consumed by Changes 2-3 (cascade resolver, fence generator, show --origin). -->

### configref: Per-Field Metadata Table

#### R1: Field registry replaces the monolithic template
The `internal/configref` package SHALL model the config schema as an **ordered slice of per-field metadata entries** (a field registry) rather than a single `refData` struct injected into one `text/template` body. Each entry SHALL carry: `Key` (dotted path), `Default` (typed canonical built-in default), `Description`, `Scope` (`project`/`system`/`both`), `Advertise` (bool), and `RenamedFrom` (string, `""` today). Slice order SHALL be the rendering order, giving deterministic byte-stable output.

- **GIVEN** the config schema
- **WHEN** `configref` is loaded
- **THEN** the schema is represented as an ordered `[]Field` (or equivalent) registry, not a monolithic template string
- **AND** each field's metadata (default, description, scope, advertise, renamed_from) is directly queryable

#### R2: Defaults sourced from canonical Go constants — no second copy
Every field default that has a canonical Go constant SHALL be referenced from that constant, never copied as a literal: the claude session command from `agent.DefaultSessionCommand`, per-tier profiles via `agent.DefaultTier` over `agent.TierNames()`, stage names via `agent.StageNames()`. The existing fail-loud invariants (every tier reported by `TierNames()` has a `DefaultTier` profile; every tier has a stage grouping) SHALL carry over — a broken invariant returns an error rather than emitting a degraded reference.

- **GIVEN** a field whose default has a canonical Go constant (e.g. `agent.tiers`)
- **WHEN** the registry is built
- **THEN** the default is read from the constant, not restated as a literal
- **AND** a tier reported by `agent.TierNames()` with no `agent.DefaultTier` profile, or no stage grouping, causes registry construction to return an error (fail-loud)

#### R3: Canonical default vs. rendering example distinction
The registry's `Default` field SHALL hold the **canonical built-in default** (what Change 2's cascade falls back to), NOT a rendering-only example. Where today's output shows an example that is not the binary default (`source_paths: [src/]`, `test_paths: ["**/*_test.go"]` — the binary default for both is empty/nil), the example SHALL live on the description/comment side and the `Default` SHALL be the true canonical default (nil/empty).

- **GIVEN** `source_paths`, whose binary default is empty but whose reference shows `- src/` as an example
- **WHEN** the registry entry is built
- **THEN** `Default` is nil/empty (the canonical default) and the `- src/` example is carried in the rendered comment/description, not in `Default`

#### R4: Scope assignments per decision 6
Scope SHALL be assigned: `agent.tiers` and `providers` = `both`; `project.*`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories` = `project`. Fields the plan's taxonomy does not enumerate (`stage_hooks`, `branch_prefix`, `fab_version`) SHALL default to `project`.

- **GIVEN** the field registry
- **WHEN** scope is assigned per field
- **THEN** `agent.tiers`/`providers` are `both`; all other enumerated fields and the three unenumerated fields are `project`

#### R5: Advertise assignments
`Advertise` SHALL be `true` for the optional override surfaces a project typically has not set live (`agent.tiers`, `providers`, `checklist.extra_categories`, `true_impact_exclude`, `stage_hooks`, `branch_prefix`, `test_paths`) and `false` for scaffold-seeded identity fields (`project.*`, `source_paths`) and machine-managed `fab_version`. `Advertise` has NO behavioral consumer in this change — it is data + `--json` exposure only.

- **GIVEN** the field registry
- **WHEN** advertise is assigned per field
- **THEN** the optional-override surfaces are `true` and the identity/machine-managed fields are `false`
- **AND** no code branches on `Advertise` in this change (the fence generator is Change 3)

#### R6: renamed_from plumbed empty
`RenamedFrom` SHALL be present on every field entry with value `""` today. Historical renames (`agent.spawn_command` → `providers.claude.session_command`) SHALL NOT be backfilled — the field serves future renames only.

- **GIVEN** the field registry
- **WHEN** built today
- **THEN** every entry's `RenamedFrom` is `""`
- **AND** the `--json` output omits `renamed_from` when empty (`omitempty`)

### configref: YAML Renderer From Table

#### R7: Render() walks the table, output contract-equivalent to today
`Render()` SHALL keep its signature (`() (string, error)`) and byte-stability contract but generate the commented YAML by walking the field registry instead of executing one monolithic template. "Output equivalent to today" means **contract-equivalent, not byte-identical**: same key coverage, same live/commented split, same documented semantics that the existing tests assert verbatim. All nine existing tests in `config_test.go` SHALL keep passing.

- **GIVEN** the restructured renderer
- **WHEN** `Render()` is called
- **THEN** it returns commented YAML with the same key coverage and live/commented split as today
- **AND** all nine existing `TestConfigReference*` tests pass unchanged in intent
- **AND** two successive `Render()` calls return byte-identical output

#### R8: Section-level narrative prose survives
The registry representation SHALL accommodate multi-line/block commentary beyond one-line field descriptions, so today's narrative comment blocks survive verbatim where tests assert them: the providers explanation, the per-provider notes, the three-provider starter template (claude live / codex+gemini commented), the "fallback from dispatch_command to session_command" phrase, the `{model}`/`{effort}` placeholders, and the fixed stage→tier mapping comment.

- **GIVEN** the existing tests assert verbatim prose (`fallback from dispatch_command to session_command`, `{model}`, `codex -m {model} -c model_reasoning_effort={effort}`, etc.)
- **WHEN** the reference is rendered from the table
- **THEN** every asserted prose string is still present in the output
- **AND** retired keys (`review_tools`, `spawn_command`) are absent

### configref + CLI: --json Flag

#### R9: --json emits the field table as machine-readable JSON
`fab config reference --json` SHALL emit the field registry as a flat JSON array in table (rendering) order to stdout, using stdlib `encoding/json` only. Each element SHALL be an object `{key, default, description, scope, advertise, renamed_from}` with `renamed_from` omitted when empty. Output SHALL be deterministic/byte-stable. Without `--json`, output SHALL be the commented YAML exactly as before. The command SHALL stay a pure query (no file writes, exit 0 on success), and an extra positional arg SHALL still be rejected.

- **GIVEN** `fab config reference --json`
- **WHEN** run
- **THEN** it prints a valid, deterministic JSON array of per-field objects to stdout and exits 0
- **AND** `renamed_from` is absent from every object (empty today)
- **GIVEN** `fab config reference` (no flag)
- **WHEN** run
- **THEN** output is the commented YAML, contract-identical to before
- **GIVEN** `fab config reference --json extra`
- **WHEN** run
- **THEN** the extra positional arg is rejected (non-zero exit)

#### R10: Registry lint + JSON/YAML key parity
The package SHALL fail-loud (like `gatherData` today) if any registry row has an empty `Description` or a `Scope` not in {`project`, `system`, `both`}. The `--json` key set SHALL be verifiable against the YAML reference's documented key set so the two renderings cannot drift apart.

- **GIVEN** a registry row with an empty description or invalid scope
- **WHEN** the reference is built/rendered
- **THEN** an error is returned (fail-loud), not a degraded reference
- **GIVEN** both the YAML and JSON renderings
- **WHEN** their documented key sets are compared
- **THEN** every JSON `key` appears as a documented key in the YAML reference (no drift)

### Docs: Spec + CLI Reference + SPEC Mirror + Index

#### R11: New spec docs/specs/config.md
A new human-curated spec `docs/specs/config.md` SHALL record the config-system design intent: the per-field metadata schema (fields, granularity rule, defaults-from-constants invariant, defaults-vs-examples distinction), the scope taxonomy with decision-6 assignments and rationale, `advertise` semantics (the A/B/C field-category model), `renamed_from` carry-forward, and the `--json` output shape — all in authoritative detail for Change 1. Forward-looking effort context (override cascade, presence=intent, managed-fence contract, system config path) SHALL be recorded clearly marked as landing in Changes 2-3.

- **GIVEN** the config-upgrade effort design
- **WHEN** `docs/specs/config.md` is written
- **THEN** it authoritatively records Change 1's schema and marks Changes 2-3 forward-looking intent as such

#### R12: _cli-fab.md + SPEC-_cli-fab.md updated
`src/kit/skills/_cli-fab.md` § fab config reference SHALL be updated: document `--json` (shape, determinism, pure-query unchanged), and refresh the "Generated, not hand-written" paragraph to describe the per-field metadata table as the generation source (correcting the stale `spawn.DefaultSpawnCommand` reference to the canonical `agent.DefaultSessionCommand`). The corresponding `docs/specs/skills/SPEC-_cli-fab.md` row SHALL be updated in the same change (constitution SPEC-mirror rule). The canonical source `src/kit/skills/_cli-fab.md` is edited — never the deployed `.claude/skills/` copy.

- **GIVEN** the CLI gains a `--json` flag and a new generation source
- **WHEN** the docs are updated
- **THEN** `_cli-fab.md` documents `--json` and the table-based generation, and `SPEC-_cli-fab.md`'s fab config reference row reflects the same
- **AND** no file under `.claude/skills/` is edited directly

#### R13: Specs index row
`docs/specs/index.md` SHALL gain a `config` row (hand-edited — the specs index is human-curated).

- **GIVEN** the new `docs/specs/config.md`
- **WHEN** the specs index is updated
- **THEN** a `| [config](config.md) | ... |` row is present

### Non-Goals

- Cascade resolution / three-layer merge in `internal/config` — Change 2.
- `fab config show --origin` and `fab config init --system` — Change 2.
- `fab config upgrade`, the managed fence generator, and `fab_version` relocation — Change 3.
- Any behavioral consumer of `Scope`/`Advertise`/`RenamedFrom` — Changes 2-3 (this change is data + `--json` exposure only).
- No migration ships — this change writes no user data (`fab config reference` stays a pure query; `config.yaml` untouched).
- No change to `fab/project/config.yaml` semantics, the scaffold, or `internal/config` (the loader).

### Design Decisions

1. **Ordered slice registry, not a map**: rendering order must be deterministic and byte-stable; an ordered `[]Field` gives that for free — *Why*: matches today's byte-stability contract and Change 2's per-field iteration; *Rejected*: `map[string]Field` (non-deterministic iteration would break byte-stability).
2. **Block prose carried as a per-field `Comment`/`Doc` string plus interleaved section entries**: today's narrative blocks (providers explanation, per-provider notes) are long-form and not one-line descriptions; the registry entry carries a multi-line rendered-comment field, and pure-prose section headers are registry entries with no live key — *Why*: reproduces today's output quality and keeps the verbatim-string tests passing; *Rejected*: a single one-line `Description` per field (cannot reproduce the narrative blocks the tests assert).
3. **`Default any` (typed) with a separate rendered representation**: `Default` holds the canonical typed default for `--json`/Change 2; the YAML renderer emits the live/commented example text separately — *Why*: keeps the defaults-vs-examples distinction load-bearing for Change 2 clean; *Rejected*: storing the rendered YAML string as the default (conflates example with canonical default).
4. **YAML renderer keeps deterministic hand-emitted output, not a per-field template loop that changes bytes**: the safest path to "all nine tests pass" is to preserve the exact current output text while sourcing every value/segment from the registry — *Why*: contract-equivalence with the lowest regression risk; the tests are the executable contract.

## Tasks

### Phase 1: Core registry + renderer (single package, single commit unit)

- [x] T001 Define the `Field` struct and `Scope` type in `src/go/fab/internal/configref/configref.go` — fields `Key string`, `Default any`, `Description string`, `Scope Scope`, `Advertise bool`, `RenamedFrom string`, plus a `Comment`/block-prose representation for section narrative. Add `Scope` constants (`ScopeProject`/`ScopeSystem`/`ScopeBoth` with string values `project`/`system`/`both`). <!-- R1 --> <!-- done (rework 1): Field now carries `Segment string` — the rendered commented-YAML block for that field. Design Decision 2 is now real: the registry rows carry the block prose, not a monolithic template. Rows that render inside another block (project.description/linear_workspace live in the project.name segment) carry an empty Segment. -->
- [x] T002 Build the ordered field registry in `configref.go`: a constructor (e.g. `fields() ([]Field, error)`) that assembles the ordered slice, sourcing every constant-backed default from `agent.DefaultSessionCommand`, `agent.DefaultTier`/`agent.TierNames()`, `agent.StageNames()` (no literal copies), carrying over the fail-loud tier/stage-grouping invariants from `gatherData`. Assign `Scope` per R4 and `Advertise` per R5; set `RenamedFrom: ""` on every row. Set `Default` to the canonical default (nil/empty for `source_paths`/`test_paths`), keeping example values on the comment side. <!-- R2 R3 R4 R5 R6 --> <!-- done (rework 1): empty-default convention unified — every "no built-in default" field now carries nil Default (JSON null), never a typed empty. checklist.extra_categories/stage_hooks/branch_prefix/fab_version moved from []string{}/map{}/"" → nil, so --json emits null uniformly (only providers + agent.tiers carry real defaults). Documented in docs/specs/config.md new § "Default semantics — the uniform empty convention" + the `default` table row; pinned by new test TestConfigReferenceJSONEmptyDefaultConvention. -->
- [x] T003 Add a registry lint in `configref.go` (invoked from the registry constructor / `Render`): return an error if any row has an empty `Description` or a `Scope` not in {project, system, both} — fail-loud like the existing `gatherData` invariants. <!-- R10 --> <!-- done: lintFields() called from Fields(); Render() calls Fields() so the lint gates Render too. -->
- [x] T004 Rewrite `Render()` in `configref.go` to walk the field registry and emit the commented YAML, preserving today's exact output contract (key coverage, live/commented split, all verbatim narrative prose blocks: providers explanation, per-provider notes, three-provider template, no-fallback phrase, `{model}`/`{effort}` placeholders, fixed stage→tier mapping comment). Keep the signature `() (string, error)` and byte-stability. Remove `refData`/`referenceTemplate`/`gatherData`/`tmpl` once the registry path replaces them (retaining `tierStages` if still the stage-grouping source). Update the package doc comment to describe the metadata-table generation source. <!-- R7 R8 --> <!-- done (rework 1): Render() now genuinely walks the registry — emits referenceHeader then concatenates each row's Segment in table order (blank line between). Output BYTE-IDENTICAL to the pre-change binary (verified: `fab-pre config reference` == `fab-final config reference`, sha256 match); all 9 pre-existing TestConfigReference* tests green. Deleted render.go wholesale (referenceTemplate/tmpl/renderYAML gone) and removed renderData/tierRenderRow/buildRenderData from configref.go. The two TierNames+DefaultTier loops collapsed to ONE: new tierRows() is the single walk that both the agent.tiers Default and the agent.tiers Segment consume. Dynamic segments (providers/agent.tiers/stage_hooks) interpolate agent.DefaultSessionCommand/DefaultTier/StageNames at build time — no literal value copy. Nice-to-haves fixed: the old unordered-map "ordered" comment is gone (tierRows returns an ordered slice, correctly documented); test_paths comment no longer uses backslash-escaped quotes. -->
- [x] T005 Add `RenderJSON()` (or equivalent) in `configref.go`: marshal the field registry to a deterministic flat JSON array via stdlib `encoding/json`, per-field objects `{key, default, description, scope, advertise, renamed_from(omitempty)}`. Use a JSON-tagged view struct (or json tags on `Field`) with `renamed_from,omitempty`. <!-- R9 --> <!-- done: RenderJSON() via jsonField view struct + SetIndent(2)/SetEscapeHTML(false). -->

### Phase 2: CLI wiring

- [x] T006 Add the `--json` flag to `configReferenceCmd()` in `src/go/fab/cmd/fab/config.go`: when set, print `configref.RenderJSON()`; otherwise print `configref.Render()` unchanged. Keep `cobra.NoArgs` (extra positional args still rejected), pure-query semantics (no file writes, exit 0 on success). Update the command's `Long`/doc comment to mention `--json`. <!-- R9 --> <!-- done: BoolVar --json + renderReference(asJSON) helper; NoArgs retained. -->

### Phase 3: Tests

- [x] T007 Verify the nine existing `TestConfigReference*` tests in `src/go/fab/cmd/fab/config_test.go` still pass against the restructured renderer; adapt only where a test references a now-removed internal symbol (expected: none — tests call `configref.Render()` and the cobra command). <!-- R7 R8 --> <!-- done: all 9 pass unchanged (no internal-symbol references). -->
- [x] T008 [P] Add new tests in `config_test.go`: (a) `--json` output parses as valid JSON and is byte-stable across two renders; (b) the JSON `key` set is a subset of / matches the YAML reference's documented key set (no drift); (c) registry lint — every row has a non-empty description and a valid scope ∈ {project, system, both}; (d) scope assignments match decision 6 for the enumerated fields; (e) `fab config reference --json extra` is rejected and plain `reference` output is contract-unchanged. <!-- R9 R10 R4 --> <!-- done: 5 new tests (JSONIsValidAndByteStable, JSONKeysMatchYAML, RegistryLint, ScopeAssignments, CommandJSONFlag). -->

### Phase 4: Docs (SPEC-mirror + spec + index)

- [x] T009 [P] Create `docs/specs/config.md` per R11: the per-field metadata schema, scope taxonomy + decision-6 assignments + rationale, `advertise` A/B/C semantics, `renamed_from` carry-forward, `--json` shape (Change 1, authoritative); the cascade / presence=intent / managed-fence / system-path forward context clearly marked as Changes 2-3. <!-- R11 --> <!-- done (rework 1): resolved the self-contradiction — § Section-level prose rewritten (now "Section-level prose lives on the row — the segment") to describe the registry-carried Segment as one projection of one row ("two projections of ONE row, not a second copy of the schema to drift"), agreeing with the "One source, no second copy" framing. Added § "Default semantics — the uniform empty convention" pinning T002's null-not-typed-empty resolution; updated the `default` table-row wording to match. No monolithic-template claim remains. -->
- [x] T010 [P] Add the `config` row to `docs/specs/index.md` (hand-edited human-curated index). <!-- R13 --> <!-- done: config row added after stage-models. -->
- [x] T011 [P] Update `src/kit/skills/_cli-fab.md` § fab config reference: document `--json` (shape, determinism, pure-query unchanged); refresh the "Generated, not hand-written" paragraph to describe the per-field metadata table as the generation source and correct `spawn.DefaultSpawnCommand` → `agent.DefaultSessionCommand`. <!-- R12 --> <!-- done: §fab config reference updated (usage block, metadata-table paragraph, --json paragraph). -->
- [x] T012 [P] Update `docs/specs/skills/SPEC-_cli-fab.md` § fab config reference row to reflect the `--json` flag and the metadata-table generation source (SPEC-mirror rule). <!-- R12 --> <!-- done: inventory row rewritten. -->

## Execution Order

- T001 → T002 → T003 → T004 → T005 (registry types before the constructor before the renderers; all in one file, sequential).
- T006 depends on T005 (`RenderJSON` must exist).
- T007, T008 depend on the Phase 1/2 code; T008 is `[P]` alongside T007.
- Phase 4 docs (T009-T012) are all `[P]` and independent of code; may run alongside Phase 3.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `internal/configref` models the schema as an ordered per-field metadata registry (`[]Field` with Key/Default/Description/Scope/Advertise/RenamedFrom), not a monolithic `refData`+template. <!-- MET (rework 1): the registry is now the sole YAML source — Field gained a Segment member, Render() walks the slice concatenating segments, and render.go (referenceTemplate/tmpl/renderYAML) plus renderData/buildRenderData are deleted. No monolithic template remains. -->
- [x] A-002 R2: every constant-backed default is sourced from its Go constant (`agent.DefaultSessionCommand`, `agent.DefaultTier`/`TierNames`, `StageNames`); no literal duplicates; fail-loud tier/stage-grouping invariants preserved.
- [x] A-003 R3: `Default` holds the canonical default (nil/empty for `source_paths`/`test_paths`); example values live on the comment/description side.
- [x] A-004 R4: scope assignments match decision 6 (`agent.tiers`/`providers`=both; all others incl. `stage_hooks`/`branch_prefix`/`fab_version`=project).
- [x] A-005 R5: advertise=true for the optional-override surfaces and false for identity/machine-managed fields; no code branches on `Advertise`.
- [x] A-006 R6: every registry row has `RenamedFrom==""`; `--json` omits `renamed_from`; no historical rename backfilled.
- [x] A-007 R7: `Render()` keeps its signature and byte-stability, walks the registry, and all nine existing tests pass. <!-- MET (rework 1): Render() = referenceHeader + each row's Segment joined in table order; the returned slice from Fields() is what it iterates (no lint-only call). Byte-identical to the pre-change binary (sha256-verified); all nine TestConfigReference* pass. -->
- [x] A-008 R8: all verbatim narrative prose (providers explanation, per-provider notes, three-provider template, no-fallback phrase, placeholders, stage→tier mapping) survives; retired keys absent.
- [x] A-009 R9: `fab config reference --json` emits a deterministic per-field JSON array; plain `reference` output is contract-unchanged; extra positional arg rejected; exit 0 on success.
- [x] A-010 R10: registry lint fails loud on empty description / invalid scope; JSON key set has no drift from the YAML documented key set.
- [x] A-011 R11: `docs/specs/config.md` records Change 1's schema authoritatively and marks Changes 2-3 forward context.
- [x] A-012 R12: `_cli-fab.md` and `SPEC-_cli-fab.md` both document `--json` and the metadata-table generation source; no `.claude/skills/` edits.
- [x] A-013 R13: `docs/specs/index.md` has a `config` row.

### Behavioral Correctness

- [x] A-014 R7: after the restructure, `fab config reference` (no flag) produces output that round-trips via `config.LoadPath` with the same live/commented key split as before (existing `TestConfigReferenceRoundTrips` passes).
- [x] A-015 R9: `--json` and no-flag are mutually exclusive output modes on the same pure-query command; neither writes a file.

### Scenario Coverage

- [x] A-016 R9: a test exercises `--json` valid-JSON + byte-stability and JSON/YAML key parity.
- [x] A-017 R10: a test exercises the registry lint (non-empty description, valid scope) and the decision-6 scope assignments.

### Edge Cases & Error Handling

- [x] A-018 R2: a tier from `agent.TierNames()` lacking a `DefaultTier` profile or stage grouping causes an error, not a degraded reference (fail-loud invariant retained).
- [x] A-019 R9: `fab config reference --json extra` returns a non-zero exit (cobra.NoArgs) and writes nothing.

### Code Quality

- [x] A-020 Pattern consistency: new code follows the existing `configref`/`config` naming, error-wrapping (`fmt.Errorf("configref: …%w")`), and doc-comment style; functions stay focused (no god function in `Render`).
- [x] A-021 No unnecessary duplication: defaults reuse the `agent` constants rather than restating values (the 6nke no-drift invariant); no second schema copy introduced. <!-- MET (rework 1): the field documentation is now authored once per row — the one-line Description and the long-form Segment are two projections of ONE row, not two hand-maintained copies. The duplicated TierNames/DefaultTier loop is gone: tierRows() is the single walk feeding both the agent.tiers Default and its Segment. -->
- [x] A-022 Canonical source only: kit edits are in `src/kit/skills/_cli-fab.md`, never the gitignored `.claude/skills/` deployed copy.
- [x] A-023 SPEC-mirror sync: the `_cli-fab.md` edit carries its `SPEC-_cli-fab.md` update in the same change.
- [x] A-024 CLI ⇒ docs + tests: the `--json` command-signature change updates `_cli-fab.md` and ships test coverage.
- [x] A-025 Markdown-only artifacts: `docs/specs/config.md` and index/CLI edits are standard CommonMark; no binary formats.

### documentation_accuracy

- [x] A-026 R12: the updated `_cli-fab.md` generation-source description matches the actual code (`agent.DefaultSessionCommand`, per-field metadata table) — no stale `spawn.DefaultSpawnCommand` or "Go template" claim left. <!-- MET (rework 1): "generated by walking an ordered per-field metadata table" (_cli-fab.md:315, SPEC-_cli-fab.md:25, config.go doc, configref.go package doc) is now TRUE as written — Render() walks the table. The code doc comments (config.go, configref.go package doc) were rewritten to match the segment-walking implementation. Verified none of the five doc locations retains a monolithic-template claim. (docs/memory/_shared/configuration.md still says "text/template"/spawn.DefaultSpawnCommand — that is a memory file, updated at hydrate per intake § Affected Memory, out of apply scope.) -->
- [x] A-027 R11: `docs/specs/config.md` scope/advertise/renamed_from claims match the shipped registry values exactly.

### cross_references

- [x] A-028 R13: the `docs/specs/index.md` `config` row links resolve to `config.md`; `SPEC-_cli-fab.md` row wording is consistent with `_cli-fab.md`.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- Affected memory (for hydrate, not apply): `docs/memory/_shared/configuration.md` and `docs/memory/distribution/kit-architecture.md` per intake § Affected Memory.

## Deletion Candidates

<!-- Replaced in place at review (rework cycle 1) — the rework-0 candidates were executed during rework. -->

- None remaining — the rework-1 restructure deleted everything this change made redundant: `render.go` (the monolithic `referenceTemplate`/`tmpl`/`renderYAML`) is gone, and `configref.go`'s parallel render view (`renderData`/`tierRenderRow`/`buildRenderData`, the duplicate `TierNames`/`DefaultTier` loop) is removed; the registry is the sole source. No other existing code is made redundant: `tierStages` remains the sole stage-grouping source, and `spawn.DefaultSpawnCommand` (alias of `agent.DefaultSessionCommand`) keeps its live consumers in `cmd/fab/operator.go` and `cmd/fab/batch.go`.

## Assumptions

<!-- Carried from intake.md § Assumptions (all user-confirmed via the 2026-07-08
     /fab-discuss session) plus apply-entry generation decisions. Three grades
     only (Certain/Confident/Tentative); Scores required per row. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Field table is the single source; defaults referenced from canonical Go constants (`agent.DefaultSessionCommand`, `agent.DefaultTier`, `agent.StageNames`) — no second copy | Mandated by the 6nke no-drift invariant and the plan doc ("the single source") | S:90 R:85 A:95 D:90 |
| 2 | Confident | "Output equivalent to today" = contract equivalence (same keys, live/commented split, tested prose strings, all nine existing tests pass), not byte-identical | Plan says "should stay equivalent"; the test suite is the executable contract; byte-identity would defeat the restructure | S:70 R:80 A:75 D:65 |
| 3 | Confident | `--json` shape: flat JSON array in table order, per-field objects `{key, default, description, scope, advertise, renamed_from(omitempty)}`, deterministic | Plan says only "add `--json` for tooling"; array-of-fields is the obvious dump of an ordered table; easily revised before Change 2 consumes it | S:60 R:85 A:80 D:60 |
| 4 | Confident | Unenumerated fields (`stage_hooks`, `branch_prefix`, `fab_version`) get scope=project | Decision-6 rationale + conservative default; enforcement lands in Change 2, so re-classification is a one-line data change | S:65 R:85 A:70 D:60 |
| 5 | Confident | advertise=true for optional override surfaces (`agent.tiers`, `providers`, `checklist.extra_categories`, `true_impact_exclude`, `stage_hooks`, `branch_prefix`, `test_paths`); false for `project.*`, `source_paths`, `fab_version` | Plan's fence example is illustrative; no behavioral consumer until Change 3; final set recorded in the spec | S:45 R:88 A:50 D:35 |
| 6 | Certain | `renamed_from` ships empty on every row; historical renames NOT backfilled | Plan frames it as "future field renames"; historical renames already shipped as migrations | S:70 R:90 A:85 D:75 |
| 7 | Confident | `docs/specs/config.md` covers the full effort's design intent (cascade, presence=intent, fence, system path) marked as Changes 2-3, with Change 1's table schema in authoritative detail | Plan: "this is where the schema decisions are recorded"; specs are pre-implementation intent (constitution VI); trivially editable | S:60 R:90 A:75 D:55 |
| 8 | Confident | The registry carries block-level/section prose (a multi-line comment field + pure-prose section entries) in addition to per-field descriptions, so today's narrative blocks survive | Existing tests assert verbatim prose; a one-line description per field cannot reproduce today's output; exact representation is apply's call | S:50 R:80 A:60 D:45 |
| 9 | Certain | No migration ships with this change | Constitution migration rule triggers on user-data restructure; this change writes no user data | S:85 R:80 A:95 D:90 |
| 10 | Confident | `Default` is typed `any` holding the canonical default; the YAML live/commented example text is rendered separately (not stored as the default) | Keeps the defaults-vs-examples distinction (R3) clean for Change 2's cascade; the JSON `default` must be the canonical value, not example prose | S:65 R:80 A:75 D:60 |
| 11 | Confident | `_cli-fab.md` generation-source description is corrected to `agent.DefaultSessionCommand` (was stale `spawn.DefaultSpawnCommand`) | The code sources from `agent.DefaultSessionCommand` (`spawn.DefaultSpawnCommand` is merely an alias); documentation_accuracy checklist category requires the doc match the code | S:80 R:90 A:85 D:75 |

11 assumptions (3 certain, 8 confident, 0 tentative).
