# Plan: Config Cascade & Visibility Commands

**Change**: 260708-lpb5-config-cascade-visibility
**Intake**: `intake.md`

## Requirements

### Config Loader: Three-Layer Cascade

#### R1: System-layer resolution at the LoadPath seam
`config.LoadPath` (the single seam every consumer reaches through, incl. `config.Load`) SHALL resolve effective config across three layers, highest precedence first: (1) the requested **project** file, (2) the **system** file `~/.fab-kit/config.yaml`, (3) the built-in defaults already applied at existing point-of-use seams. The system home is resolved via `os.UserHomeDir()`; tests override it with `t.Setenv("HOME", …)`.

- **GIVEN** a project `config.yaml` and a `~/.fab-kit/config.yaml` both present
- **WHEN** any consumer calls `config.Load`/`config.LoadPath`
- **THEN** the returned `*Config` reflects the merged project-over-system result, with built-in fallbacks still applied downstream by `internal/agent`'s tier/provider merge and the nil-safe accessors

#### R2: Absent system file ⇒ byte-identical current behavior
When `~/.fab-kit/config.yaml` does not exist, `LoadPath` SHALL behave byte-identically to today (the system layer is an empty overlay, no error, no warning).

- **GIVEN** no `~/.fab-kit/config.yaml` on the machine
- **WHEN** `LoadPath` reads a project file (or the project file is also absent)
- **THEN** the parsed `*Config` is identical to the pre-change result, exit behavior unchanged

#### R3: Per-field deep merge semantics
The two config **files** SHALL merge at the YAML map level, before unmarshal into `Config`, by generic per-field deep merge: **maps merge per-key recursively**, **lists replace** (never concatenate), **scalars replace**. The project layer wins on every conflicting leaf. A system-only map key survives alongside a project-only sibling key.

- **GIVEN** project `agent.tiers.review.model` and system `agent.tiers.doing.model`
- **WHEN** the layers merge
- **THEN** both tier keys are present in the effective config; a key set in both layers takes the project value; a list set in both layers takes the project list wholesale

#### R4: Malformed system file is fail-open
A malformed or unreadable `~/.fab-kit/config.yaml` SHALL emit a `fab: warning:` line on stderr and skip the system layer (proceed with project-over-defaults). A malformed **project** file SHALL keep today's error behavior (return the parse error). Warnings never change the exit code or any stdout contract.

- **GIVEN** a `~/.fab-kit/config.yaml` that is not valid YAML
- **WHEN** `LoadPath` reads it
- **THEN** a `fab: warning:` line is written to stderr, the system layer is skipped, and the project-over-defaults result is returned with no error

### Config Loader: Scope Enforcement

#### R5: Project-scoped fields in the system file are pruned with a warning
Before merging, each top-level override unit present in the system file SHALL be checked against the field-scope metadata. A field whose scope is `project` SHALL be pruned from the system layer and a `fab: warning:` line emitted naming the field and pointing the user to `fab/project/config.yaml`. Fields with scope `both` or `system` are honored. Unknown keys in the system file are ignored silently (matching project-file behavior). Enforcement fires on every config load (stderr only; stdout contracts and exit codes unaffected).

- **GIVEN** a `~/.fab-kit/config.yaml` containing `source_paths:` (scope `project`)
- **WHEN** `LoadPath` merges the system layer
- **THEN** `source_paths` is not applied from the system layer, and `fab: warning: ignoring project-scoped field "source_paths" in ~/.fab-kit/config.yaml (project-scoped fields belong in fab/project/config.yaml)` is written to stderr

#### R6: Scope metadata reaches the loader cycle-free, single-sourced
The key→scope metadata `internal/config` consumes for R5 SHALL be provided **without** `internal/config` importing `internal/configref` (the cycle `configref → agent → config` forbids it). It SHALL live in a leaf package importable by both `internal/config` and `internal/configref`, so the scope enum and per-key scope values are **single-sourced** (never a second copy).

- **GIVEN** the import chain `configref → agent → config`
- **WHEN** the loader needs a field's scope
- **THEN** it reads the leaf package's key→scope table; `configref`'s `Field.Scope` values derive from the same leaf so no drift is possible; `go build ./...` has no import cycle

### Visibility Commands

#### R7: `fab config show [--origin]`
`fab config show` SHALL be a pure query (no file writes) in the `config` family that prints the **effective** (post-cascade) config to stdout. With `--origin`, it SHALL additionally annotate each field's provenance (origin ∈ {project path, system path, `default`}), walking the full field registry, with **per-key drill-down for map-valued fields**. Exit 0 on success; extra positional args rejected by `cobra.NoArgs`.

- **GIVEN** an effective config assembled from project + system + defaults
- **WHEN** `fab config show --origin` runs
- **THEN** every registry field is printed with its effective value and origin, map-valued fields drilled down per-key, a field taking its built-in default annotated `default`

#### R8: `fab config init --system`
`fab config init --system` SHALL write a `~/.fab-kit/config.yaml` scaffold generated from the registry: a header explaining the system layer, then **only** `scope: system`/`both` fields (today `agent.tiers`, `providers`), **all commented**. It SHALL refuse to overwrite an existing `~/.fab-kit/config.yaml` (non-zero exit, message naming the path); no `--force` in v1. Bare `fab config init` (no `--system`) SHALL be a usage error.

- **GIVEN** no `~/.fab-kit/config.yaml`
- **WHEN** `fab config init --system` runs
- **THEN** the file is written with only scope system/both fields, all commented, and re-running it exits non-zero without overwriting
- **AND** `fab config init` with no flag exits non-zero as a usage error

### Documentation, Spec & Test Obligations

#### R9: CLI docs, SPEC mirror, and design spec updated
The `_cli-fab.md` § fab config section SHALL document `show [--origin]` and `init --system`; the `docs/specs/skills/SPEC-_cli-fab.md` mirror SHALL be swept in the same change; and `docs/specs/config.md`'s `[Change 2]` forward-looking sections (§ Override cascade, § Visibility commands, the § Scope taxonomy enforcement note) SHALL flip to landed status — the same treatment ff2v gave Change 1.

- **GIVEN** the CLI surface and spec/memory sweep classes
- **WHEN** the change ships
- **THEN** `_cli-fab.md`, `SPEC-_cli-fab.md`, and `docs/specs/config.md` all reflect the landed cascade + commands, with no `[Change 2]`-as-future language for the now-landed pieces

### Non-Goals

- No migration file — nothing restructures existing user data (the system file is net-new + opt-in; `config.yaml` is never written by this change).
- No `fab config upgrade`, no managed fence, no `fab_version` move, no `setFabVersion` deletion (all Change 3).
- The fab-kit binary (`src/go/fab-kit`) is untouched — its own `readFabVersion`/`ResolveConfig` read only the project file and are Change 3's concern.
- No `advertise`/`renamed_from` consumers (Change 3).
- No `fab config validate` / typo linter (recorded future non-goal; `show --origin` surfaces typos in the interim).

### Design Decisions

1. **Cascade lands at `LoadPath`, files merge at the YAML-node level**: `LoadPath` reads the project file's raw bytes, reads the system file (if present + scope-pruned), deep-merges the two `yaml.Node`/`map[string]any` trees (project over system), then unmarshals the merged tree into `Config` — *Why*: a single seam gives every one of the ~12 consumers effective config with zero per-caller change, and node-level merge reuses the per-field deep-merge shape (`internal/agent`'s tykw precedent) generically without teaching the merge about `Config`'s fields — *Rejected*: merging two unmarshalled `*Config` structs (needs per-field reflection/merge code that duplicates the tier merge and can't express "list replaces"), and a per-caller opt-in (contradicts "effective config" — the system layer is user-global by definition).
2. **Built-in defaults stay at existing point-of-use seams**: the file-merge composes project-over-system; the defaults layer remains where it lives today (`internal/agent`'s tier/provider merge, nil-safe accessors) — *Why*: minimal diff, preserves the ye8r single-parser + tykw merge architecture, and file-merge + existing fallbacks compose to identical three-layer semantics — *Rejected*: folding defaults into the loader (would move the built-in tables out of `agent`, a large blast radius for no behavior gain).
3. **Leaf package `internal/configscope` holds the scope enum + key→scope table**: `configscope` imports nothing internal (leaf); `internal/config` imports it for enforcement; `internal/configref` imports it for its `Field.Scope` type/values — *Why*: breaks the `configref → agent → config` cycle while keeping scope single-sourced (the table is the one source; configref's per-row `Scope` values reference the same constants) — *Rejected*: injecting scope as data into `LoadPath` from `cmd/fab` (the loader is called from ~12 sites and internal packages, not all of which can thread scope data; the metadata belongs beside the loader), and duplicating the scope list in `config` (violates the no-drift invariant).
4. **`show --origin` computes provenance by re-reading each layer independently**: walk the registry; for each field, determine the highest layer that sets it (project node → system node → else default) — *Why*: honest per-field/per-key provenance mirrors `git config --show-origin`; per-key drill-down for maps matches the per-key merge granularity — *Rejected*: threading provenance through the merge (couples the merge to a display concern; the merge stays a pure data operation).
5. **`init --system` renders only system/both fields from the registry, all commented**: generated from `configref.Fields()` (segments), filtered to `scope ∈ {system, both}` — *Why*: single source, cannot drift from the schema; commented so the file is inert until the user opts in — *Rejected*: a hand-written scaffold string (drifts from the registry, exactly what ff2v eliminated).

## Tasks

### Phase 1: Setup

- [x] T001 Create leaf package `src/go/fab/internal/configscope/configscope.go`: define `Scope` type + `ScopeProject`/`ScopeSystem`/`ScopeBoth` constants + `Valid(Scope) bool`, and an ordered key→scope table `keyScopes` with a `ScopeFor(topLevelKey string) (Scope, bool)` accessor covering every top-level override unit (`project`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist`, `providers`, `agent`, `stage_hooks`, `branch_prefix`, `fab_version`). Package imports nothing internal (leaf). <!-- R6 -->

### Phase 2: Core Implementation

- [x] T002 Refactor `src/go/fab/internal/configref/configref.go` to source its `Scope` type + constants from `internal/configscope` (type alias `Scope = configscope.Scope`, const re-exports, `validScope` delegates to `configscope.Valid`), so the scope enum is single-sourced. Keep every `Field.Scope` assignment and all existing exported names/behavior byte-identical. <!-- R6 -->
- [x] T003 Implement the three-layer cascade in `src/go/fab/internal/config/config.go`: add `LoadPath` system-layer resolution — read the project bytes, resolve `~/.fab-kit/config.yaml` via `os.UserHomeDir()`, and when present unmarshal it to a `yaml.Node`/`map[string]any`, scope-prune project-scoped top-level keys (T004), deep-merge project-over-system (maps per-key recursive, lists replace, scalars replace), then unmarshal the merged tree into `Config`. Absent system file ⇒ current single-file path unchanged. Malformed/unreadable system file ⇒ `fab: warning:` on stderr + skip layer (fail-open); malformed project file keeps today's error. Add a `homeDir` seam (var indirection over `os.UserHomeDir`) so tests set `HOME`. <!-- R1 R2 R3 R4 -->
- [x] T004 Implement scope pruning + warning inside the T003 system-layer path: for each top-level key in the system map, look up `configscope.ScopeFor`; drop keys whose scope is `project` and emit `fab: warning: ignoring project-scoped field "<key>" in ~/.fab-kit/config.yaml (project-scoped fields belong in fab/project/config.yaml)` to stderr; honor `both`/`system`; ignore unknown keys silently. <!-- R5 -->
- [x] T005 Add the generic YAML deep-merge helper in `internal/config` (maps merge per-key recursively, lists replace, scalars replace) used by T003. Operate on the decoded `map[string]any` trees so the merge is `Config`-agnostic. <!-- R3 -->
- [x] T006 Add `fab config show [--origin]` in `src/go/fab/cmd/fab/config.go`: pure query printing the effective config; with `--origin`, walk `configref.Fields()` and annotate each field's effective value with its origin (project path / system path / `default`), per-key drill-down for map-valued fields (`agent.tiers`, `providers`). Resolve provenance by inspecting each layer's presence for the field/key. Wire the subcommand into `configCmd()`. <!-- R7 -->
- [x] T007 Add `fab config init --system` in `src/go/fab/cmd/fab/config.go`: with `--system`, render a scaffold from `configref.Fields()` filtered to `scope ∈ {system, both}` (all commented, with a system-layer header) and write to `~/.fab-kit/config.yaml`; refuse to overwrite an existing file (non-zero exit + message naming the path); no `--force`. Bare `fab config init` (no `--system`) is a usage error. Wire into `configCmd()`. <!-- R8 -->

### Phase 3: Integration & Edge Cases (tests)

- [x] T008 [P] Tests in `src/go/fab/internal/config/config_test.go`: cascade merge table (maps per-key incl. nested tier fields, lists replace, scalars replace), absent system file = byte-identical, malformed system file = warn+skip (fail-open) with project result intact, malformed project file still errors. Use `t.Setenv("HOME", …)` for the system path. <!-- R1 R2 R3 R4 -->
- [x] T009 [P] Tests in `src/go/fab/internal/config/config_test.go` (or a scope-focused test file): scope pruning drops a project-scoped system key with the exact warning text, honors a `both`-scoped key (`agent.tiers`), ignores unknown keys silently. <!-- R5 -->
- [x] T010 [P] Tests in `src/go/fab/internal/configscope/configscope_test.go`: `ScopeFor` returns the decision-6 taxonomy for every top-level key; `Valid` accepts the three scopes and rejects others. <!-- R6 -->
- [x] T011 [P] Tests in `src/go/fab/cmd/fab/config_test.go`: `show` prints effective config and exits 0; `show --origin` renders per-field provenance incl. per-key map drill-down and a `default`-origin row; `init --system` writes the scaffold (only system/both fields, all commented, parses as absent/inert) and refuses overwrite (non-zero); bare `init` is a usage error. Use `t.Setenv("HOME", …)`. <!-- R7 R8 -->

### Phase 4: Docs, Spec & Mirror Sweep

- [x] T012 Update `src/kit/skills/_cli-fab.md` § fab config: document `fab config show [--origin]` and `fab config init --system` alongside `reference`. <!-- R9 -->
- [x] T013 Update the mirror `docs/specs/skills/SPEC-_cli-fab.md` § fab config in the same change (Sibling & Mirror Sweeps). <!-- R9 -->
- [x] T014 Update `docs/specs/config.md`: flip the `[Change 2]` sections (§ Override cascade, § Visibility commands) and the § Scope taxonomy enforcement note to landed status, same treatment ff2v gave Change 1. <!-- R9 -->

## Execution Order

- T001 blocks T002, T003, T004 (leaf package first)
- T005 blocks T003 (merge helper used by the cascade)
- T003 blocks T004 (pruning lives inside the system-layer path)
- T006, T007 depend on T001–T005 (commands consume the cascade + scope metadata)
- T008–T011 depend on their respective implementation tasks
- T012–T014 are independent of code but should reflect the final command signatures

## Acceptance

### Functional Completeness

- [x] A-001 R1: `config.LoadPath` resolves project > system (`~/.fab-kit/config.yaml`) > built-in defaults, with `HOME` overridable in tests
- [x] A-002 R2: with no system file, `LoadPath` returns a result byte-identical to the pre-change single-file parse (no error, no warning)
- [x] A-003 R3: per-field deep merge — maps merge per-key recursively, lists replace, scalars replace, project wins
- [x] A-004 R5: a project-scoped field in the system file is pruned and a `fab: warning:` line naming the field is emitted
- [x] A-005 R6: scope metadata lives in a leaf `internal/configscope` package consumed by both `config` and `configref`; `go build ./...` has no import cycle and scope values are single-sourced
- [x] A-006 R7: `fab config show` prints effective config; `--origin` adds per-field provenance with per-key map drill-down
- [x] A-007 R8: `fab config init --system` writes the registry-generated commented scaffold (only system/both fields) and refuses overwrite; bare `init` is a usage error
- [x] A-008 R9: `_cli-fab.md`, `SPEC-_cli-fab.md`, and `docs/specs/config.md` document the landed cascade + commands

### Behavioral Correctness

- [x] A-009 R4: malformed system file ⇒ stderr warning + skipped layer (fail-open), project result intact; malformed project file still errors
- [x] A-010 R3: a system-only map key survives alongside a project-only sibling key after merge (per-key, not whole-map, replacement)

### Edge Cases & Error Handling

- [x] A-011 R7: `fab config show`/`show --origin` reject extra positional args (`cobra.NoArgs`) and write no files
- [x] A-012 R8: `fab config init --system` on an existing file exits non-zero with a message naming the path and does not truncate/overwrite it

### Code Quality

- [x] A-013 Pattern consistency: new code follows the surrounding `internal/config`/`cmd/fab` naming, nil-safe accessor, and cobra command patterns
- [x] A-014 No unnecessary duplication: scope enum/values single-sourced via `internal/configscope`; the scaffold + `show` reuse `configref.Fields()` rather than restating the schema
- [x] A-015 Canonical source only: skill edits land in `src/kit/skills/`, never `.claude/skills/`
- [x] A-016 CLI ⇒ docs + tests: the new subcommands ship with `_cli-fab.md` updates and Go tests (constitution VII, test-alongside)
- [x] A-017 Migrations: no user-data restructure, so no migration file is required (system file is net-new + opt-in)

### Documentation Accuracy

- [x] A-018 R9: the docs describe the actual shipped command signatures and cascade semantics (no `[Change 2]`-as-future language for landed pieces)

### Cross References

- [x] A-019 R9: `docs/specs/config.md` and `_cli-fab.md`/`SPEC-_cli-fab.md` cross-references remain consistent (system path, scope taxonomy, command names)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Deletion Candidates

- `src/go/fab/internal/configref/configref.go:84` (`validScope`) — now a one-line delegation to `configscope.Valid`; its single call site (`lintFields`, line 520) could call the leaf package directly, deleting the wrapper.
- `src/go/fab/internal/configref/configref.go` per-row `Scope:` assignments (12 rows) — with the leaf `configscope.ScopeFor` table landed and the construction lint asserting row/table equality, the explicit per-row values are derivable (`Fields()` could populate `Scope` from `scopeFor(f.Key)`), deleting the second enumeration the lint currently guards. Kept deliberately by T002's byte-identical constraint; safe follow-up cleanup.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Cascade lands at `LoadPath` (the single seam `Load` delegates to), giving all ~12 consumers effective config with zero per-caller change | Intake §"What Changes" 1 + assumption 11 name LoadPath explicitly; verified `Load` calls `LoadPath` | S:90 R:75 A:95 D:90 |
| 2 | Certain | Files merge at the YAML map/node level before unmarshal; maps per-key recursive, lists replace, scalars replace; project wins | Intake §"What Changes" 1 merge semantics, user-confirmed decisions restated in docs/specs/config.md § Override cascade | S:90 R:75 A:95 D:90 |
| 3 | Certain | System path is `~/.fab-kit/config.yaml` via `os.UserHomeDir()`; tests override with `t.Setenv("HOME", …)` | Decision 5 (config-upgrade.md), intake §"What Changes" 1 | S:95 R:80 A:95 D:95 |
| 4 | Certain | Malformed system file = `fab: warning:` on stderr + skip layer (fail-open); malformed project file keeps today's error | Intake §"What Changes" 1 + assumption 6; "config must never brick" | S:85 R:80 A:90 D:85 |
| 5 | Confident | Scope metadata extracted into a NEW leaf package `internal/configscope` (holds the enum + key→scope table); both `config` and `configref` consume it, single-sourcing scope | Intake §"What Changes" 2 mandates cycle-free + single-sourced; leaf-package is the named option and the cleanest of the two (vs data injection through ~12 call sites) | S:70 R:70 A:80 D:65 |
| 6 | Confident | Scope enforcement keys on TOP-LEVEL override units in the system map (project, source_paths, checklist, agent, providers, …), matching the registry's override-unit granularity | Intake §"What Changes" 2 checks "top-level override units"; the registry rows are override units, `checklist`/`project`/`agent` are the top-level YAML keys | S:65 R:75 A:75 D:70 |
| 7 | Confident | `show --origin` computes provenance by inspecting each layer's node independently (project → system → default), not by threading provenance through the merge | Keeps the merge a pure data op; honest per-field/per-key provenance is the git-config-show-origin precedent (intake §"What Changes" 3 + assumption 8) | S:55 R:80 A:70 D:60 |
| 8 | Confident | `init --system` renders from `configref.Fields()` filtered to scope∈{system,both}, reusing each row's Segment, all commented, with a system-layer header | Intake §"What Changes" 4 "generated from the same registry so it cannot drift"; Segment is the row's rendered YAML | S:60 R:75 A:80 D:70 |
| 9 | Confident | Both new subcommands are pure/opt-in in the `config` family: `show` writes nothing; `init --system` writes only `~/.fab-kit/config.yaml`, refuses overwrite, no `--force`; bare `init` is a usage error | Intake §"What Changes" 3-4 + assumptions 7, 10; conservative user-owned-file default | S:65 R:80 A:75 D:70 |
| 10 | Confident | The warning fires on every config load (stderr, `fab: warning:` prefix), not only in `show`; stdout contracts + exit codes unaffected | Intake assumption 9; "ignored with a warning" is load-time semantics | S:55 R:80 A:75 D:60 |

10 assumptions (4 certain, 6 confident, 0 tentative).
