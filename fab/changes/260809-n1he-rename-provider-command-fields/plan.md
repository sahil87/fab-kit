# Plan: Rename Provider Command Fields

**Change**: 260809-n1he-rename-provider-command-fields
**Intake**: `intake.md`

## Requirements

### Config: ProviderConfig read-time alias

#### R1: ProviderConfig Field Rename and Alias
The `ProviderConfig` struct MUST expose `InteractiveCommand` and `HeadlessCommand` as its primary fields, while retaining `SessionCommand` and `DispatchCommand` as deprecated aliases.

- **GIVEN** a config YAML containing `session_command` or `dispatch_command`
- **WHEN** it is decoded into `ProviderConfig`
- **THEN** it SHALL resolve using the new spelling if present, falling back to the old spelling per field if the new is absent
- **AND** it SHALL NOT emit a deprecation warning for the old spelling

### Core: Built-in data and Go constants

#### R2: Defaults and Identifiers
Built-in provider defaults and their corresponding exported Go identifiers MUST use the new naming convention.

- **GIVEN** `src/go/fab/internal/agent/defaults.yaml` and `agent.go`
- **WHEN** the built-in defaults are compiled and tested
- **THEN** `defaults.yaml` SHALL use `interactive_command` and `headless_command`
- **AND** exported identifiers in `agent.go` SHALL be renamed (e.g. `DefaultInteractiveCommand`, `DefaultHeadlessCommand`)
- **AND** all associated pinned tests MUST pass with the new names

### CLI: Registry and verb surface

#### R3: Config CLI Surface
The `fab config` command's dynamic dotted-key matcher MUST accept only the new spellings for provider commands.

- **GIVEN** an invocation of `fab config set providers.codex.dispatch_command ...`
- **WHEN** the dynamic dotted-key matcher processes the key
- **THEN** it SHALL reject it as an unknown key and suggest `explain`
- **AND** it SHALL accept `providers.codex.headless_command`
- **AND** `renamed_from` metadata MUST be set on the new fields for `--json` consumers

### Toolkit: Migration

#### R4: On-Disk Migration
A migration file MUST rewrite the old keys to the new keys in both project and system scopes.

- **GIVEN** an existing `fab/project/config.yaml` with `session_command`
- **WHEN** `/fab-setup migrations` runs the `2.18.1-to-2.19.0.md` migration
- **THEN** it SHALL rewrite it to `interactive_command` preserving values and comments
- **AND** historical migration files and `fab/plans/sahil/config-overhaul.md` MUST stay verbatim

### Docs & Memory: Kit-text sweep

#### R5: Comprehensive Kit-Text Sweep
All documentation, skill files, and memory files MUST be updated to use the new command names, leaving out-of-scope usages untouched.

- **GIVEN** the repository documentation and skills
- **WHEN** a full sweep is performed
- **THEN** `session_command` and `dispatch_command` MUST be replaced with `interactive_command` and `headless_command` respectively in the affected files
- **AND** `dispatch.mode` values and `fab dispatch` verb mentions MUST remain unchanged
- **AND** any edited memory files missing `type: memory` MUST have it stamped

### Design Decisions
#### Read-Time Alias for ProviderConfig
**Decision**: Keep deprecated fields on `ProviderConfig` with new-spelling-wins resolution.
**Why**: Ensures half-migrated configs resolve correctly without breaking users.
**Rejected**: Hard break (removing old fields), which would break existing configurations instantly.
*Introduced by*: 260809-n1he-rename-provider-command-fields

#### Dotted-Key Matcher Rejects Old Spellings
**Decision**: The `fab config` dotted-key matcher for `set`/`unset`/`explain`/`show` accepts only new spellings.
**Why**: The alias is a file-read affordance, not a write-surface one; migration fixes disk.
**Rejected**: Accepting old spellings for writes, which would prolong the use of deprecated names.
*Introduced by*: 260809-n1he-rename-provider-command-fields

## Tasks

### Phase 1: Setup
- [x] T001 Update `ProviderConfig` in `src/go/fab/internal/config/config.go` with aliases and resolve methods in `agent.go` (and usages in consumers) <!-- R1 -->

### Phase 2: Core Implementation
- [x] T002 Update `src/go/fab/internal/agent/defaults.yaml` keys and prose <!-- R2 -->
- [x] T003 Rename exported constants in `src/go/fab/internal/agent/agent.go` and update `TestDefaultRoleProfilesArePinned` / `TestNonClaudeProviderFillsArePinned` in `agent_test.go` <!-- R2 -->
- [x] T004 Update `TestDefaultsFileIsWellFormed` in `src/go/fab/internal/agent/defaults_test.go` <!-- R2 -->
- [x] T005 Update registry row in `src/go/fab/internal/configref/configref.go` and matcher logic <!-- R3 -->
- [x] T006 Update `src/go/fab/internal/configref/defaultsmap.go` and `configref_test.go` <!-- R3 -->
- [x] T007 Fix references in `src/go/fab/internal/config/config_test.go` and `src/go/fab/internal/dispatch/dispatch_test.go` <!-- R1 -->
- [x] T008 Update occurrences in `src/go/fab/internal/dispatch/dispatch.go`, `pane_mode.go`, `src/go/fab/internal/spawn/spawn.go` and tests <!-- R1 -->
- [x] T009 Update `cmd/fab/agent.go`, `batch_new.go`, `batch_switch.go`, `config.go`, `dispatch_start.go`, `operator.go`, `resolve_agent.go` and their tests <!-- R1 -->
- [x] T010 Update `fab/project/config.yaml` fence <!-- R3 -->

### Phase 3: Integration & Edge Cases
- [x] T011 Create migration `src/kit/migrations/2.18.1-to-2.19.0.md` <!-- R4 -->
- [x] T012 Sweep `src/kit/skills/` (`_cli-fab.md`, `_cli-agents.md`, `_preamble.md`, `fab-operator.md`) <!-- R5 -->
- [x] T013 Sweep `docs/specs/skills/` (`SPEC-_cli-fab.md`, `SPEC-_cli-agents.md`, `SPEC-_preamble.md`, `SPEC-fab-operator.md`) <!-- R5 -->
- [x] T014 Sweep specs: `docs/specs/stage-models.md`, `docs/specs/architecture.md`, `docs/specs/harness-adapters.md`, `docs/specs/config.md`, `docs/specs/glossary.md`, `docs/specs/skills.md`, `docs/specs/index.md` <!-- R5 -->
- [x] T015 Sweep memory: `docs/memory/_shared/configuration.md`, `docs/memory/runtime/providers-and-profiles.md`, `docs/memory/runtime/dispatch.md`, `docs/memory/_shared/context-loading.md`, `docs/memory/distribution/kit-architecture.md`, `docs/memory/distribution/migrations.md`, `docs/memory/pipeline/execution-skills.md`, `docs/memory/runtime/agent-primitives.md`, `docs/memory/runtime/operator.md` <!-- R5 -->
- [x] T016 Run `fab memory-index` to regenerate indexes <!-- R5 -->

### Phase 4: Polish
- [x] T017 Run relevant tests (`cd src/go/fab && go test ./...`) <!-- R2 -->

## Execution Order
- T001 is a prerequisite for most tasks.
- Tests (T017) should be run continuously alongside changes but fully verified at the end.

## Acceptance

### Functional Completeness
- [x] A-001 R1: `ProviderConfig` resolves old and new spellings correctly with new-spelling-wins logic. (verified: `ResolveProvider` agent.go:568-578 + end-to-end `fab resolve-agent` spot-checks)
- [x] A-002 R2: `defaults.yaml` and `agent.go` constants use new spellings and compile successfully. (`go build ./...` + full uncached `go test ./...` green)
- [x] A-003 R3: `fab config` set/unset/explain/show rejects old spellings and accepts new ones. (verified: `config set providers.codex.dispatch_command` refuses with explain pointer; `headless_command` accepted)
- [x] A-004 R4: `2.18.1-to-2.19.0.md` migration exists and correctly rewrites configs on disk. (both scopes, live-keys-only, atomic write, comment/value preservation, idempotency sentinel)
- [x] A-005 R5: `session_command` and `dispatch_command` are replaced in skills, specs, memory, and config fence, while excluding out-of-scope mentions. (repo-wide grep: remaining hits only in alias code/tests, migration machinery, historical migrations, change artifacts, deliberate renamed-from notes)

### Behavioral Correctness
- [x] A-006 R1: Resolving a config with only `session_command` correctly maps it to `InteractiveCommand` with no deprecation warning. (verified end-to-end incl. env `FAB_PROVIDERS` layer; silent)
- [x] A-007 R3: Dotted-key matchers in `configref.go` enforce new spelling whitelist. (configref.go:1016; covered by renamed TestMutation_* tests)

### Scenario Coverage
- [x] A-008 R1: Tests pass for fallback logic when old spelling is present. (`TestResolveProvider_CommandFieldsAlias` in `internal/agent/agent_test.go` — YAML-driven, so it covers the deprecated `yaml:` tags too: old-only, both-spellings new-wins per field, half-migrated, and silence on stderr; verified to FAIL when the fallback is removed)
- [x] A-009 R3: Tests pass for `--json` showing `renamed_from`. (`wantRenamedFrom` pin in config_test.go; verified `config explain providers --json`)

### Edge Cases & Error Handling
- [x] A-010 R1: Config correctly resolves mixed use of new and old spellings on different providers. (verified: half-migrated and both-spellings configs resolve per-field new-wins)

### Code Quality
- [x] A-011 Pattern consistency: New code follows naming and structural patterns of surrounding code. (mirrors the `Tiers`/`Profiles` alias precedent)
- [x] A-012 No unnecessary duplication: Existing utilities reused where applicable.
- [x] A-013 Sibling & Mirror Sweeps: Entire sibling/mirror classes swept up front (all 4 touched skills carry their SPEC-*.md mirrors; user-facing literals incl. cobra Example and error strings swept).
- [x] A-014 Editing `.claude/skills/` directly: Changes made to canonical `src/kit/skills/*.md` only.
- [x] A-015 Changing a CLI command without updating `_cli-fab.md` + tests: Handled implicitly in sweeps and test runs.
- [x] A-016 Stating an owned rule AND pointing at its owner: Not violated during memory sweeps.

## Notes
- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Deletion Candidates

None — this change renames fields, constants, and keys in place; the deprecated `SessionCommand`/`DispatchCommand` struct fields it retains (src/go/fab/internal/config/config.go:126-127) are deliberate read-time aliases per R1, not redundancy, and the old exported identifiers were renamed rather than left dead.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | All tasks will be implemented directly without further prompts. | The intake was very thorough on what needs to change. | S:90 R:70 A:90 D:90 |

1 assumptions (0 certain, 1 confident, 0 tentative).
