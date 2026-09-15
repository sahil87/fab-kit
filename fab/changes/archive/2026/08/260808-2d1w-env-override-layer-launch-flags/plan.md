# Plan: Per-Session Selection — Env Override Layer + Launch Flags

**Change**: 260808-2d1w-env-override-layer-launch-flags
**Intake**: `intake.md`

## Requirements

### Configuration: Registry-Backed Environment Overrides

#### R1: Forward Registry Enumeration and Scope Gating

`internal/configscope` SHALL expose the ordered dotted keys represented by the `internal/configref` registry, and `internal/config` MUST walk that enumeration forward to derive each environment variable as `FAB_` plus the uppercased dotted key with dots replaced by underscores. The loader MUST honor only rows whose scope is `both` or `system`; a set project-scoped variable MUST warn through `warnw` and be ignored. Unknown `FAB_*` variables MUST NOT be scanned or reverse-parsed. A configref parity test MUST fail if its registry keys and the leaf enumeration diverge.

- **GIVEN** `FAB_AGENT_WORKERS=codex` and `FAB_SOURCE_PATHS=private-src`
- **WHEN** a config is loaded
- **THEN** the `agent.workers` override is included in the environment layer
- **AND** `source_paths` is omitted with a `fab: warning:` naming `FAB_SOURCE_PATHS`

#### R2: YAML Values, Fail-Open Behavior, and Highest Precedence

`internal/config.LoadPath` SHALL resolve effective configuration as env > project > system > built-in defaults. Each non-empty recognized environment value MUST be parsed as YAML and nested beneath its registry row's dotted path before merging through the existing `deepMerge` semantics (maps merge per key; lists, scalars, and map/non-map mismatches replace). A set-but-empty variable MUST be treated as unset. An unparseable value MUST warn through `warnw`, be skipped, and leave lower layers effective.

- **GIVEN** system and project files set different `agent.profiles` leaves and `FAB_AGENT_PROFILES='{review: {provider: codex}}'`
- **WHEN** `LoadPath` resolves the cascade
- **THEN** the environment-provided review provider wins while non-conflicting file-layer leaves survive
- **AND** replacing the environment value with invalid YAML warns and restores the project/system result without returning an error

### Configuration CLI: Effective Values and Provenance

#### R3: Environment Layer Visibility

`config.Layers` MUST expose the environment overlay plus per-registry-key variable provenance, and `LoadLayers` MUST use the same four-layer cascade as `LoadPath`. Plain `fab config show` SHALL include environment-affected effective values without materializing built-in defaults. `fab config show --origin` SHALL render environment leaves at the highest precedence and name the setting variable as `$FAB_…`; project, system, and default origins SHALL remain unchanged for leaves not set by the environment.

- **GIVEN** `FAB_AGENT_WORKERS=codex` over project `agent.workers: claude`
- **WHEN** `fab config show` and `fab config show --origin` run
- **THEN** plain output contains `workers: codex`
- **AND** origin output contains `agent.workers = codex  # $FAB_AGENT_WORKERS`

### Session Launching: Worker Selection Sugar

#### R4: `fab agent --workers`

`fab agent` SHALL accept `--workers <provider>` in both role-addressed and provider-addressed modes. When the command executes a session, the flag MUST append `FAB_AGENT_WORKERS=<provider>` to the exec environment while leaving session-command resolution unchanged. `--print` MUST continue to print only the resolved command. The value MUST pass through verbatim without provider validation, so any unknown provider is reported later by normal worker resolution.

- **GIVEN** `fab agent --workers kimi3`
- **WHEN** the launcher replaces itself with the resolved session command
- **THEN** the child environment contains `FAB_AGENT_WORKERS=kimi3`
- **AND** the composed session command is identical to the no-flag command

#### R5: `fab batch new` and `fab batch switch --workers`

`fab batch new` and `fab batch switch` SHALL each accept `--workers <provider>`. When either creates a tmux worker window, its shell command MUST be prefixed with a portable, single-quoted `FAB_AGENT_WORKERS=<provider>` assignment. Embedded single quotes and shell metacharacters MUST remain data, not executable syntax. The provider name MUST NOT be validated.

- **GIVEN** a workers value containing a single quote
- **WHEN** either batch command composes its `tmux new-window` shell command
- **THEN** the assignment uses the existing `'<part>'\''<part>'` shell-escaping convention
- **AND** the existing `/fab-new` or `/fab-switch` prompt remains the command's argument

#### R6: `fab operator --workers`

`fab operator` SHALL accept `--workers <provider>`. When it creates the singleton operator window, its shell command MUST be prefixed with the same shell-quoted `FAB_AGENT_WORKERS=<provider>` assignment used by batch launchers while leaving operator-role session-command resolution unchanged. Switching to an already-existing operator window MUST remain unchanged and MUST NOT create a new process.

- **GIVEN** no existing operator window and `fab operator --workers codex`
- **WHEN** the operator window is launched
- **THEN** the tmux shell command begins with `FAB_AGENT_WORKERS='codex'`
- **AND** the operator session itself still resolves from the Tier-1 `operator` role

### Documentation: Generic Mechanism, Deliberate Advertising

#### R7: CLI and Config Contract Documentation

The canonical `src/kit/skills/_cli-fab.md`, its `docs/specs/skills/SPEC-_cli-fab.md` mirror, and `docs/specs/config.md` MUST document the four-layer cascade, generic forward mapping, scope gating, YAML parsing, fail-open warnings, environment provenance, and all four `--workers` launch surfaces. User-facing examples SHALL advertise only `FAB_AGENT_WORKERS` and `FAB_AGENT_SESSION`, SHALL state that the mechanism applies generically to all `both`/`system` registry rows, and MUST NOT document `FAB_DISPATCH_MODE` or `FAB_KIT_PATH`. The pre-existing fab-kit-only `FAB_AGENTS` variable SHALL be identified as unrelated. No migration SHALL ship because the feature is additive and stores no user data.

- **GIVEN** a reader consults the CLI reference or config spec
- **WHEN** they look for per-session provider selection
- **THEN** they can discover the variables, mapping, precedence, warning behavior, origin display, and `--workers` flags
- **AND** no parallel-change variable or unrelated config surface is advertised as part of this change

### Non-Goals

- Do not rename provider command fields or modify provider field semantics.
- Do not modify `defaults.yaml`, `resolve_agent.go`, `internal/dispatch`, or `internal/configupgrade`.
- Do not add `--session`, persist a session preference, validate provider names, or introduce a migration.
- Do not edit deployed `.claude/skills/` copies or hydrate `docs/memory/` during apply.

### Design Decisions

#### Dotted Keys Stay in the Cycle-Free Leaf

**Decision**: The generic environment walk consumes an ordered dotted-key enumeration from `internal/configscope`; a configref parity test guards equality with the richer registry.
**Why**: `internal/config` cannot import `internal/configref` without closing `configref → agent → config`, while configscope already owns the cycle-free scope taxonomy.
**Rejected**: Copying the keys into `internal/config`, reverse-scanning `FAB_*`, or importing configref despite the cycle.
*Introduced by*: 260808-2d1w-env-override-layer-launch-flags

#### Launch Flags Export One Shared Variable

**Decision**: All four `--workers` surfaces export `FAB_AGENT_WORKERS`; `fab agent` uses its exec environment and tmux launchers use a shell-quoted assignment prefix.
**Why**: The spawned process tree then reaches the same `LoadPath` seam used by both resolver and dispatch re-resolution, with no second selection mechanism.
**Rejected**: New resolution flags on dispatch, tmux-version-specific `new-window -e`, provider validation, or persisted config edits.
*Introduced by*: 260808-2d1w-env-override-layer-launch-flags

## Tasks

### Phase 1: Setup

- [x] T001 Add the ordered dotted registry-key enumeration to `src/go/fab/internal/configscope/configscope.go`, extend `configscope_test.go`, and add a configref parity test under `src/go/fab/internal/configref/`. <!-- R1 -->

### Phase 2: Core Implementation

- [x] T002 Add the exported YAML value parser, generic environment-name/path helpers, scope-gated fail-open environment overlay, and focused parsing/mapping tests in `src/go/fab/internal/config/config.go` and `config_test.go`. <!-- R1, R2 --> <!-- rework(cycle 3): fail-open type hole — loadEnvLayer (config.go:491) accepts any syntactically valid YAML, but a type-incompatible value (FAB_DISPATCH_WATCHABLE=not-a-bool, FAB_AGENT_WORKERS='{bogus: value}') later errors the merged-tree Config unmarshal, so commands EXIT with an error instead of warn+skip. Per-variable, validate the nested fragment is Config-compatible (e.g. trial-unmarshal the single-variable fragment into a throwaway Config) and warn+skip on failure, preserving lower layers. Add regression tests for both probe cases asserting warn+skip+lower-layer-preserved and exit 0. -->
- [x] T003 Merge the environment overlay above project/system in `LoadPath`, covering precedence, per-key map merge, invalid/empty values, scope warnings, and no-environment stability in `src/go/fab/internal/config/config_test.go`. <!-- R2 -->
- [x] T004 Extend `config.Layers`, `LoadLayers`, and `src/go/fab/cmd/fab/config.go` provenance flattening for env values/variable origins, with plain/origin tests in `config_show_init_test.go`. <!-- R3 --> <!-- rework: flattenOrigin uses nil as the absence sentinel, so an explicit YAML-null env override (FAB_AGENT_WORKERS=null) is shown by plain `show` but reported as `default` origin by --origin — track env-layer presence explicitly (per-leaf set-markers or a presence map), and add a regression test for the null-valued override -->

### Phase 3: Integration & Edge Cases

- [x] T005 Add `fab agent --workers` exec-environment injection and a testable exec seam in `src/go/fab/cmd/fab/agent.go` and `agent_test.go`. <!-- R4 -->
- [x] T006 Add the shared workers-assignment quoting helper plus `fab batch new --workers` wiring and tmux-command tests in `src/go/fab/cmd/fab/workers.go`, `workers_test.go`, `batch_new.go`, and `batch_new_test.go`. <!-- R5 --> <!-- rework(cycle 2): the cycle-1 consolidation rewired internal/dispatch (dispatch.go imports internal/shellquote at :44, call sites :213/:457) — that package is reserved for parallel Change 3 (plan Non-Goals). REVERT every internal/dispatch edit to its pre-change state (restore its private quote helper verbatim); keep internal/shellquote consumed ONLY by this change's own call sites (cmd/fab/workers.go). Record the deferred dispatch.go consolidation as a Deletion Candidate/follow-up note instead of doing it here. -->
- [x] T007 Add `fab batch switch --workers` wiring and quoted tmux-command tests in `src/go/fab/cmd/fab/batch_switch.go` and `batch_switch_test.go`. <!-- R5 -->
- [x] T008 Add `fab operator --workers` wiring and singleton-launch shell-command tests in `src/go/fab/cmd/fab/operator.go` and `operator_test.go`. <!-- R6 -->

### Phase 4: Polish

- [x] T009 Update only this change's config/agent/operator/batch sections in `src/kit/skills/_cli-fab.md` and synchronize `docs/specs/skills/SPEC-_cli-fab.md`. <!-- R3, R4, R5, R6, R7 --> <!-- rework(cycle 3): incomplete mirror sweep — current-contract 'three layers' claims remain at src/kit/skills/_cli-fab.md:687 (dispatch.reap_done contract), docs/specs/skills/SPEC-_cli-fab.md:33, and docs/specs/index.md:28. Grep repo-wide for three-layer/'three layers' cascade claims describing the CURRENT contract and update to four layers (env > project > system > defaults); leave historical/landed-change narratives verbatim. -->
- [x] T010 Update only the cascade and visibility sections of `docs/specs/config.md`, including the deliberate advertised-variable set and unrelated-`FAB_AGENTS` caveat. <!-- R1, R2, R3, R7 -->
- [x] T011 Run `gofmt` on every touched Go file and run scoped tests for `./src/go/fab/internal/config/...`, `./src/go/fab/internal/configscope/...`, `./src/go/fab/internal/configref/...`, and `./src/go/fab/cmd/fab/...`; widen if failures reveal cross-package impact. <!-- R1, R2, R3, R4, R5, R6, R7 --> <!-- rework: hermeticity gap — internal/spawn's TestMain isolates HOME but not the registry-derived FAB_* vars, so `FAB_AGENT_SESSION=codex go test ./internal/spawn` fails 4 assertions (spawn.Command reaches config.LoadPath). Sweep EVERY test package whose code path reaches LoadPath (spawn at minimum; grep for other TestMain/HOME-isolating packages) and clear the FAB_* set there, then verify with FAB_AGENT_SESSION=codex FAB_AGENT_WORKERS=x go test ./... under src/go/fab -->

## Execution Order

- T001 blocks T002 because the loader walks configscope's dotted-key enumeration.
- T002 blocks T003 and T004 because both cascade and provenance consume the environment overlay helper.
- T005–T008 are independent of T001–T004 and may proceed after the shared launch-helper shape is established.
- T009–T010 follow implementation so the documented CLI and origin shapes match tested behavior.
- T011 runs after every implementation and documentation task.

## Acceptance

### Functional Completeness

- [x] A-001 R1: Configscope exposes every registry row exactly once in order, and configref parity prevents drift.
- [x] A-002 R2: Recognized non-empty environment variables produce a YAML-parsed overlay that wins above project and system layers without changing built-in fallback behavior.
- [x] A-003 R3: `LoadLayers`, plain `config show`, and `config show --origin` expose effective environment overrides and the setting variable that supplied each leaf.
- [x] A-004 R4: `fab agent --workers` exports the exact pass-through provider value to the executed session environment.
- [x] A-005 R5: Both batch launch commands accept `--workers` and prefix their tmux shell commands with the quoted assignment.
- [x] A-006 R6: `fab operator --workers` prefixes a newly launched operator shell command without changing singleton selection or Tier-1 profile resolution.
- [x] A-007 R7: The canonical CLI reference, its SPEC mirror, and the config spec accurately document the shipped contract and exclusions.

### Behavioral Correctness

- [x] A-008 R1: Project-scoped recognized variables warn and are ignored, while unknown `FAB_*` names are never scanned or warned about.
- [x] A-009 R2: Environment maps deep-merge per key with lower layers, while scalars/lists replace and malformed values fail open.
- [x] A-010 R3: Environment values take precedence in origin output without changing project/system/default attribution for unaffected leaves.
- [x] A-011 R4: `--workers` neither validates provider names nor alters the resolved `fab agent` session command or `--print` data contract.
- [x] A-012 R5: Batch launch prompts and worktree behavior remain unchanged apart from the optional environment assignment prefix.
- [x] A-013 R6: An existing `operator` window is still selected rather than relaunched, regardless of `--workers`.
- [x] A-014 R7: Docs mention neither `FAB_DISPATCH_MODE` nor `FAB_KIT_PATH` and do not conflate config overrides with fab-kit's unrelated `FAB_AGENTS` variable.

### Scenario Coverage

- [x] A-015 R2: Tests cover env > project > system precedence, YAML flow-map parsing, per-key map merge, empty values, malformed values, and a no-env baseline.
- [x] A-016 R3: Tests cover plain show output plus `--origin` output naming `$FAB_AGENT_WORKERS`.
- [x] A-017 R4: Tests capture the environment passed to the `fab agent` exec seam.
- [x] A-018 R5: Tests capture both batch tmux shell commands with a provider value containing a single quote.
- [x] A-019 R6: Tests capture the operator tmux shell command with the environment prefix.

### Edge Cases & Error Handling

- [x] A-020 R1: Set project-scoped variables produce stderr warnings through `warnw` but never change the loader's return error or stdout contracts.
- [x] A-021 R2: A set-but-empty variable behaves as unset, and an invalid YAML value restores the lower-layer value after warning.
- [x] A-022 R2: A whole map-valued environment row can merge with non-conflicting file-layer leaves.
- [x] A-023 R3: Different variables under one top-level map retain their own origin names at leaf level.
- [x] A-024 R5: Embedded quotes and shell metacharacters in `--workers` remain quoted data in both batch launchers.
- [x] A-025 R6: Embedded quotes and shell metacharacters in the operator flag remain quoted data.

### Code Quality

- [x] A-026 Pattern consistency: New loader and command code follows surrounding naming, error-handling, import, and test-seam conventions.
- [x] A-027 No unnecessary duplication: The YAML parser, dotted-key enumeration, env overlay builder, and workers shell-assignment quoting are each single-sourced and reused.
- [x] A-028 Readability and maintainability: The four-layer cascade and provenance ordering are explicit and understandable without clever control flow.
- [x] A-029 Existing patterns: Fail-open warnings use `warnw`, maps use `deepMerge`, commands remain non-interactive, and stdout remains data.
- [x] A-030 Focused functions: New helpers keep `LoadPath`, provenance flattening, and launcher functions focused rather than growing god functions.
- [x] A-031 Named constants: `FAB_AGENT_WORKERS` is represented by one named command-layer constant rather than repeated magic strings.
- [x] A-032 Canonical source: No deployed `.claude/skills/` file is edited.
- [x] A-033 SPEC mirror sync: The `_cli-fab.md` change and `SPEC-_cli-fab.md` mirror describe the same flags and environment contract.
- [x] A-034 CLI docs and tests: Every changed command signature is reflected in `_cli-fab.md`, live Cobra help, and focused Go tests.
- [x] A-035 Go test integrity: Every touched Go behavior ships with tests derived from these requirements, and scoped suites are green after `gofmt`.
- [x] A-036 No migration: The implementation adds no persisted state or user-data restructure and ships no migration.

### Security

- [x] A-037 R5: Batch `--workers` values cannot break out of the environment assignment to inject shell syntax.
- [x] A-038 R6: Operator `--workers` values cannot break out of the environment assignment to inject shell syntax.

## Notes

- Review owns the acceptance checkboxes; apply ignores them.
- Memory updates remain deferred to hydrate.
- Shared docs are edited only in Change 1's env-layer and launch-flag sections.

## Deletion Candidates

- `internal/dispatch.shellQuote` — duplicates `internal/shellquote.Single`; consolidate only after parallel Change 3 lands so this change does not edit Change 3's reserved surface.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Expose the leaf enumeration as ordered dotted keys and derive scope from the existing top-level taxonomy | The intake requires dotted enumeration without duplicating scope; this keeps configscope dependency-free and makes the parity assertion direct | S:75 R:85 A:90 D:80 |
| 2 | Confident | Single-source the tmux assignment quoting and advertised workers variable in `cmd/fab` | All three tmux launchers share one package and the exact same shell contract; one helper prevents quoting drift without expanding package API | S:70 R:90 A:85 D:80 |

2 assumptions (0 certain, 2 confident, 0 tentative).
