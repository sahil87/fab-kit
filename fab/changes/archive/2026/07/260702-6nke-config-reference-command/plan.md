# Plan: `fab config reference` — generated reference config.yaml command

**Change**: 260702-6nke-config-reference-command
**Intake**: `intake.md`

## Requirements

### CLI: `fab config reference` command

#### R1: New `config` command group with `reference` subcommand
The `fab` binary SHALL expose a new command group `config` with a subcommand `reference` (`fab config reference`), registered on the `fab-go` root command. The group naming leaves room for a future `fab config validate` (a non-goal here). `fab config reference` is a pure query — no side effects, no file writes, no flags in v1.

- **GIVEN** the `fab` binary is installed
- **WHEN** a user runs `fab config reference`
- **THEN** a fully-commented reference `config.yaml` is printed to stdout and the process exits 0
- **AND** no file is created or modified anywhere

#### R2: Output is generated from Go constants, never hand-written
The reference text SHALL be produced from a Go template with all default/example values injected from their canonical constants — `spawn.DefaultSpawnCommand` (`internal/spawn`), the default tier profiles via `agent.DefaultTier` and `agent.TierNames`, and the pipeline stage names via `agent.StageNames` (`internal/agent`). No default value that has a canonical Go constant may be hand-typed into the template body.

- **GIVEN** the default `doing` tier profile is `{claude-opus-4-8, high}` in `internal/agent`
- **WHEN** `fab config reference` renders the `agent.tiers` block
- **THEN** the shown `doing` default reads `{ model: claude-opus-4-8, effort: high }`, sourced from `agent.DefaultTier(agent.TierDoing)`, not a literal in the template
- **AND** bumping the default in `internal/agent` changes the rendered output with no template edit

#### R3: Byte-stable output
For a given binary version, `fab config reference` SHALL emit byte-identical output on every invocation (same convention as `fab resolve` / `fab resolve-agent` queries). It reads no project config and depends on no environment state.

- **GIVEN** any working directory (fab project or not)
- **WHEN** `fab config reference` is run twice
- **THEN** the two outputs are byte-identical

#### R4: Full schema coverage — binary-consumed and skill-consumed keys
The reference SHALL cover BOTH the binary-consumed keys (modeled on the `Config` struct in `internal/config`) AND the skill-consumed keys (invisible to Go reflection). Coverage includes, at minimum: `project.name`, `project.description`, `project.linear_workspace`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, `review_tools.claude/codex/copilot`, `agent.spawn_command`, `agent.tiers.{thinking,doing,fast}.{model,effort}`, `stage_hooks.<stage>.{pre,post}`, `branch_prefix`, and `fab_version`.

- **GIVEN** the `Config` struct models `test_paths` (binary-consumed) and skills read `source_paths` (skill-consumed)
- **WHEN** `fab config reference` renders
- **THEN** both `test_paths` and `source_paths` appear as documented keys (live or commented)
- **AND** every yaml-tagged key path reachable from `Config` (recursively, incl. nested structs and map value types) appears in the output

#### R5: Layout — baseline keys live, opt-in blocks commented with defaults shown
Baseline keys every project sets (`project`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist`, `review_tools`, `agent.spawn_command`, `fab_version`) SHALL appear live with example/default values. Opt-in override blocks (`agent.tiers`, `stage_hooks`, `branch_prefix`) SHALL appear commented-out with fab-kit's defaults shown in comments, mirroring the shipped `2.2.0-to-2.3.0` block style (uncommenting = opting in). `fab_version` SHALL be documented as machine-managed (do not hand-edit).

- **GIVEN** the reference output
- **WHEN** parsed as YAML
- **THEN** `agent.tiers`, `stage_hooks`, and `branch_prefix` are absent from the parsed document (they are commented-out), while `project`, `source_paths`, `agent.spawn_command`, etc. are present as live keys

#### R6: Output round-trips into `Config`
The emitted reference SHALL parse without error via the same `internal/config` loader used for real project configs (its live keys unmarshal into `Config` cleanly).

- **GIVEN** the stdout of `fab config reference`
- **WHEN** it is written to a temp `config.yaml` and loaded via `config.LoadPath`
- **THEN** the load succeeds with no error and the live baseline keys populate their `Config` fields

### Tests: coverage + validity contracts

#### R7: Three test contracts ship alongside the command
Tests SHALL assert: (a) **validity round-trip** — the output parses into `Config` via `config.LoadPath`/`yaml.Unmarshal` without error; (b) **binary-key coverage by reflection** — walking the `Config` struct's yaml tags recursively (nested structs + map value types) confirms every key path appears in the output (commented or live); (c) **skill-key coverage by scaffold superset** — the reference's key set is a superset of the scaffold `src/kit/scaffold/fab/project/config.yaml` key set. Byte-stability is also asserted (R3).

- **GIVEN** a new binary-consumed key is added to `Config`
- **WHEN** the reflection coverage test runs without a matching reference update
- **THEN** the test fails, forcing a reference update
- **AND** injected default values need no drift test (there is no second copy to drift)

### Scaffold: pointer line

#### R8: Scaffold gains one header pointer line
`src/kit/scaffold/fab/project/config.yaml` SHALL gain exactly one header comment line at the top of the file: `# Full reference of all available options: fab config reference`. No commented reference blocks are added; the scaffold stays otherwise minimal.

- **GIVEN** a new project scaffolded from the template
- **WHEN** the user opens `fab/project/config.yaml`
- **THEN** the first line points them to `fab config reference`

### Migration: pointer line for existing configs

#### R9: Sentinel-guarded config-only migration adds the pointer line
A new migration file in `src/kit/migrations/` SHALL append the same one-line pointer comment to existing projects' `fab/project/config.yaml`, following the `2.2.0-to-2.3.0` / `2.7.1-to-2.8.0` precedent (Summary / Pre-check / Changes / Verification). It is config-only: no `.status.yaml` schema change, no binary capability pre-check. The migration is sentinel-guarded (skip when `config.yaml` is absent or the pointer line already present) and idempotent. The file is named for the next version slot after the current `src/kit/VERSION` (`2.9.2`) → `2.9.2-to-2.10.0.md`, and `src/kit/VERSION` SHALL be bumped to `2.10.0` per the dual-version migration model.

- **GIVEN** an existing project whose `config.yaml` lacks the pointer line
- **WHEN** `/fab-setup migrations` applies this migration
- **THEN** the pointer line is prepended as the file's header comment and all other keys/comments are preserved verbatim
- **AND** re-running is a complete no-op (sentinel trips)
- **AND** a project with no `config.yaml` is skipped

### Docs: point at the command, don't copy

#### R10: `_cli-fab.md` documents the new command (+ SPEC mirror sweep)
`src/kit/skills/_cli-fab.md` SHALL gain a `fab config reference` command entry (Contents list + a `## fab config reference` section) documenting purpose, no-flags, byte-stable stdout, exit 0. The SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` SHALL gain the matching Command Inventory row in the same change (constitution SPEC-mirror rule; on a CLI change treat the touched skill's SPEC mirror as in-scope).

- **GIVEN** the constitution requires `_cli-fab.md` + SPEC mirror sync on a CLI change
- **WHEN** this change ships
- **THEN** `_cli-fab.md` has the new section and Contents entry, and `SPEC-_cli-fab.md` has the matching inventory row

#### R11: `architecture.md` and `README.md` point at the command
`docs/specs/architecture.md` (Configuration section) and `README.md` (configuration / CLI reference) SHALL point at `fab config reference` as the canonical full-options reference. No YAML copies are embedded solely to document the schema — the docs point at the command.

- **GIVEN** a reader wants the full config schema
- **WHEN** they read architecture.md's Configuration section or README's config coverage
- **THEN** they find a pointer to `fab config reference`

### Non-Goals

- `fab config validate` (unknown-key/typo linting) — deferred; only the `config` group name is claimed here.
- Multi-agent spawn work (`spawn.WithProfile` placeholders, per-tier `spawn_command`, CLI dispatch adapter) — separate changes; the reference documents today's schema.
- Retro-editing the `2.2.0-to-2.3.0` migration — it stays as shipped history.
- Any config file writing by the new command — stdout only.

### Design Decisions

1. **Template package placement**: put the generator in a new `internal/configref` package (rendering logic + the `text/template` body), with the thin cobra command in `cmd/fab/config.go`. — *Why*: mirrors the codebase convention of `internal/prmeta`/`internal/impact` (rendering logic in `internal/`, thin cobra wrapper in `cmd/fab/`), keeps the template testable without the cobra shell, and isolates the constant-injection. — *Rejected*: colocating the whole template inside `resolve_agent.go`-style single-file command (works but bloats `cmd/fab/` and couples template to the command shell).
2. **`text/template` for the body**: use the stdlib `text/template` with a data struct carrying the injected constants. — *Why*: the reference is a fixed multi-line document with a handful of injected values; `text/template` is the clearest, most maintainable way to keep the prose adjacent to the injection points, and it is stdlib (no new dependency, Constitution I-friendly). — *Rejected*: `fmt.Sprintf`/`strings.Builder` concatenation (harder to read for a large document); `go:embed` (no embed usage exists in the codebase, and embed cannot inject the live constants).
3. **Migration version slot `2.9.2-to-2.10.0`**: the current `VERSION` is `2.9.2` (ahead of the last migration `2.7.1-to-2.8.0`); the new migration targets `2.10.0` and VERSION bumps to `2.10.0`. — *Why*: an additive config-feature migration is a minor bump per the catalog convention (`2.2.0-to-2.3.0`, `2.7.1-to-2.8.0`); the range must start at the current VERSION so the applicability walk chains cleanly. — *Rejected*: `2.8.0-to-...` (would leave a gap/overlap against the real current VERSION).

## Tasks

### Phase 1: Setup

- [x] T001 Create `src/go/fab/internal/configref/` package with `configref.go` holding the `text/template` reference body and a `Render() string` entrypoint that injects `spawn.DefaultSpawnCommand`, `agent.DefaultTier(...)` for each `agent.TierNames()` tier, and `agent.StageNames()` for the `stage_hooks` section <!-- R2 -->

### Phase 2: Core Implementation

- [x] T002 Implement the reference template body in `internal/configref/configref.go`: baseline keys live (`project.name/description/linear_workspace`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist.extra_categories`, `review_tools.claude/codex/copilot`, `agent.spawn_command`, `fab_version`) with example/default values; opt-in blocks (`agent.tiers`, `stage_hooks`, `branch_prefix`) commented-out with injected defaults shown; `fab_version` noted machine-managed. Layout mirrors the 2.2.0-to-2.3.0 comment style <!-- R4 --> <!-- R5 -->
- [x] T003 Ensure `Render()` output is byte-stable — deterministic ordering of injected tier/stage values (sort or use `agent.TierNames()`/`agent.StageNames()` which are already sorted), no map-iteration nondeterminism <!-- R3 -->
- [x] T004 Add `cmd/fab/config.go`: a `configCmd()` parent cobra command (group `config`, `Short` naming the future-validate room) with a `configReferenceCmd()` subcommand (`Use: "reference"`, `Args: cobra.NoArgs`, no flags) whose `RunE` prints `configref.Render()` to `cmd.OutOrStdout()` and returns nil (exit 0) <!-- R1 --> <!-- R2 -->
- [x] T005 Register `configCmd()` on the root command in `cmd/fab/main.go` <!-- R1 -->

### Phase 3: Integration & Edge Cases

- [x] T006 Add `cmd/fab/config_test.go` with the three coverage contracts + byte-stability: (a) validity round-trip via `config.LoadPath` on the emitted output written to a temp file; (b) binary-key coverage by recursive reflection over `config.Config` yaml tags (descend nested structs and map value types), asserting each key path appears in the output; (c) skill-key coverage — parse scaffold keys and assert the reference key set is a superset; (d) byte-stability across two `Render()` calls <!-- R7 --> <!-- R6 --> <!-- R3 -->

### Phase 4: Docs, Scaffold, Migration (mirror sweep — apply-owned)

- [x] T007 [P] Add the header pointer line `# Full reference of all available options: fab config reference` at the top of `src/kit/scaffold/fab/project/config.yaml` (single line, no reference block) <!-- R8 -->
- [x] T008 [P] Create `src/kit/migrations/2.9.2-to-2.10.0.md` (Summary / Pre-check / Changes / Verification) appending the same pointer line to existing configs, sentinel-guarded (skip if `config.yaml` absent or pointer line present), config-only, atomic write; preserve all other keys/comments verbatim <!-- R9 -->
- [x] T009 [P] Bump `src/kit/VERSION` from `2.9.2` to `2.10.0` <!-- R9 -->
- [x] T010 Add the `## fab config reference` section to `src/kit/skills/_cli-fab.md` and its Contents entry (purpose, no flags, byte-stable stdout, exit 0, points-not-copies rationale) <!-- R10 -->
- [x] T011 Add the matching `fab config reference` row to the Command Inventory table in `docs/specs/skills/SPEC-_cli-fab.md` (SPEC-mirror rule) <!-- R10 -->
- [x] T012 [P] Update `docs/specs/architecture.md` Configuration (config.yaml) section to point at `fab config reference` as the canonical full-options reference <!-- R11 -->
- [x] T013 [P] Update `README.md` (CLI Subcommands table and/or config coverage) to add a `fab config reference` pointer <!-- R11 -->

### Phase 5: Validation

- [x] T014 Run `cd src/go/fab && go build ./... && go test ./cmd/fab/ ./internal/configref/` (widen to `go test ./...` if cross-cutting); fix failures <!-- R7 -->

## Execution Order

- T001 → T002 → T003 → T004 → T005 (package before command before registration)
- T006 depends on T002–T005 (tests exercise the built command and package)
- T007–T009 are independent `[P]` (scaffold, migration, VERSION)
- T010 before T011 (skill edit precedes its SPEC-mirror row, though both must land together)
- T012, T013 independent `[P]`
- T014 last (validation gate over Go changes)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab config reference` exists as `config` group + `reference` subcommand, prints a commented reference config.yaml to stdout, exits 0, has no flags, and writes no file
- [x] A-002 R2: the rendered defaults are injected from `spawn.DefaultSpawnCommand`, `agent.DefaultTier(...)`, `agent.TierNames()`, `agent.StageNames()` — no canonical default value is a hand-typed literal in the template body. (Verified: tier profiles injected via `gatherData`/`DefaultTier`; `DefaultSpawnCommand` injected in the fallback comment; `StageNames()` injected for valid stage keys. The live `spawn_command:` example matches the scaffold example, not the canonical `DefaultSpawnCommand` constant — it is an illustrative example, not a duplicated constant, so R2 holds.)
- [x] A-003 R3: output is byte-identical across repeated invocations (asserted by `TestConfigReferenceByteStable`, passes)
- [x] A-004 R4: every yaml-tagged key path reachable from `Config` (recursively) AND every scaffold key appears in the reference (asserted by `TestConfigReferenceCoversBinaryKeys` + `TestConfigReferenceSupersetsScaffoldKeys`, both pass)
- [x] A-005 R5: baseline keys render live; `agent.tiers`, `stage_hooks`, `branch_prefix` render commented-out with injected defaults; `fab_version` noted machine-managed (verified in rendered output)
- [x] A-006 R6: the emitted output parses into `Config` via `config.LoadPath` with no error (asserted by `TestConfigReferenceRoundTrips`, passes)
- [x] A-007 R7: three coverage tests (validity round-trip, binary-key reflection, scaffold-superset) plus a byte-stability test exist and pass
- [x] A-008 R8: `src/kit/scaffold/fab/project/config.yaml` has exactly the one header pointer line and no added reference blocks (diff = 1 added line)
- [x] A-009 R9: `src/kit/migrations/2.9.2-to-2.10.0.md` exists in the standard 4-section shape (Summary / Pre-check / Changes / Verification), sentinel-guarded and config-only, and `src/kit/VERSION` reads `2.10.0`
- [x] A-010 R10: `_cli-fab.md` has the new section + Contents entry and `SPEC-_cli-fab.md` has the matching inventory row
- [x] A-011 R11: architecture.md and README.md point at `fab config reference` (no schema-documenting YAML copies added — architecture's existing YAML excerpt is retained with a pointer added above it, per Assumption 8)

### Behavioral Correctness

- [x] A-012 R2: changing a default in `internal/agent` (e.g. the `doing` model) would change the rendered output with no template body edit (verified by construction — `gatherData` reads `agent.DefaultTier(name)`; no tier profile literal in the template body)

### Scenario Coverage

- [x] A-013 R9: re-running the migration on an already-migrated config is a no-op (sentinel), and a project with no config.yaml is skipped (both asserted in the migration's Pre-check + Verification sections)

### Edge Cases & Error Handling

- [x] A-014 R1: `fab config reference extra-arg` is rejected (`cobra.NoArgs` + `TestConfigReferenceCommandPrintsAndExitsZero`); `fab config` with no subcommand shows the group help without error (cobra default, no `RunE` on the group)
- [x] A-015 R4: the reflection coverage walk descends into nested structs (`AgentConfig`, `StageHook`, `TierProfile`, `ProjectConfig`) and the `map[string]...` value types (`stage_hooks`, `agent.tiers`) so a nested key addition is caught (`yamlKeySegments` walks Struct/Map/Slice recursively; verified the walk visits `TierProfile.model/effort` and `StageHook.pre/post`)

### Code Quality

- [x] A-016 Pattern consistency: new code follows the `internal/prmeta`-style split (rendering package + thin cobra command) and the existing `resolve_agent.go` command conventions
- [x] A-017 No unnecessary duplication: default values are sourced from existing `internal/spawn` / `internal/agent` constants and accessors, not re-declared (see nice-to-have note on `tierStages` display-grouping map)
- [x] A-018 Readability over cleverness: the template body is readable, no God function; the reflection walk is a focused helper
- [x] A-019 Canonical source only: kit edits are under `src/kit/` (scaffold, migration, `_cli-fab.md`), never `.claude/skills/`
- [x] A-020 Migration for user-data restructuring: the pointer line reaches existing configs via a `src/kit/migrations/` file, not an ad-hoc script
- [x] A-021 CLI ⇒ docs + tests: the CLI change updates `_cli-fab.md` and ships tests (Constitution Additional Constraints)
- [x] A-022 Markdown-only artifacts: migration and doc edits are standard CommonMark; no binary formats

### Documentation Accuracy (checklist.extra_categories)

- [x] A-023 Documentation accuracy: `_cli-fab.md`, SPEC-_cli-fab.md, architecture.md, and README.md describe the command exactly as implemented (no flags, stdout-only, exit 0); the migration Summary/Verification matches its actual Changes

### Cross-References (checklist.extra_categories)

- [x] A-024 Cross-references: the SPEC mirror row matches the `_cli-fab.md` section; the migration file name/version matches `src/kit/VERSION` (`2.10.0`); docs pointers name the exact command `fab config reference`; repo-wide grep for a stale "no config reference" claim surfaces no contradiction

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- Memory-file updates (`_shared/configuration.md`, `distribution/kit-architecture.md`, `distribution/migrations.md`) belong to the hydrate stage, not apply.

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (The `2.2.0-to-2.3.0` migration's hand-written `agent.tiers` comment block is now conceptually superseded by the generated reference, but per the plan's Non-Goals it stays as shipped history and is intentionally NOT deleted.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Command is `fab config reference` under a new `config` group; pure query, no flags, stdout only | User chose the name explicitly in intake; group leaves room for future `config validate` (intake assumption 1) | S:90 R:70 A:95 D:90 |
| 2 | Certain | Reference generated from a Go `text/template` with values injected from real constants (`spawn.DefaultSpawnCommand`, `agent.DefaultTier`/`TierNames`/`StageNames`), never hand-written | User approved "generated, not hand-written" (intake assumption 2); strictly stronger than drift tests on copies | S:95 R:80 A:95 D:95 |
| 3 | Confident | Template lives in new `internal/configref` package; thin cobra command in `cmd/fab/config.go` | Mirrors `internal/prmeta`/`internal/impact` convention; intake assumption 8 left placement to apply; easily moved | S:60 R:90 A:85 D:75 |
| 4 | Confident | Three coverage tests = validity round-trip + Config-struct recursive yaml-tag reflection + scaffold key-superset, plus byte-stability | Round-trip + reflection discussed in intake; scaffold-superset covers skill-consumed keys reflection can't see (intake assumption 3) | S:60 R:85 A:80 D:70 |
| 5 | Confident | Layout: baseline keys live with example values; opt-in blocks (`agent.tiers`, `stage_hooks`, `branch_prefix`) commented with defaults shown, mirroring the 2.2.0-to-2.3.0 style | Intake assumption 4; uncommenting = opting in; prevents copy-paste pinning of defaults | S:55 R:85 A:75 D:65 |
| 6 | Certain | Migration slot is `2.9.2-to-2.10.0` and VERSION bumps to `2.10.0` | Current VERSION is 2.9.2 (ahead of last migration 2.7.1-to-2.8.0); additive config-feature = minor bump per catalog convention; range must start at current VERSION to chain | S:80 R:75 A:95 D:80 |
| 7 | Confident | Sentinel = the exact pointer line `# Full reference of all available options: fab config reference`; migration prepends it as the file header (matching the scaffold placement) | Simplest idempotency guard consistent with the 2.2.0-to-2.3.0 sentinel-comment precedent; header placement matches the scaffold so migrated + new configs converge | S:55 R:85 A:80 D:65 |
| 8 | Confident | Docs point at the command; no schema-documenting YAML copies added to architecture.md/README | User approved "docs point, don't copy" (intake assumption 7). The existing architecture.md config YAML block is retained (it predates this change and illustrates key relationships) but gains a pointer to the command | S:75 R:90 A:85 D:70 |

8 assumptions (3 certain, 5 confident, 0 tentative).
