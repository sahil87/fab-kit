# Plan: fab setup check environment doctor

**Change**: 260811-pgbq-setup-check-doctor
**Intake**: `intake.md`

## Requirements

### CLI Surface: `fab setup check`

#### R1: Command surface
A new top-level cobra command `setup` SHALL be registered in `newRootCmd()` (`src/go/fab/cmd/fab/main.go`) with a `check` subcommand, following the existing per-command file pattern (`src/go/fab/cmd/fab/setup.go`). Bare `fab setup` SHALL print a "Yet to be implemented" placeholder message (reserving the seat for the C2 wizard, backlog [stpw]) and exit 0 without running any checks. The router allowlist (`internal.LifecycleCommands` in the fab-kit module) SHALL NOT gain `setup` — the router's default route already forwards it to fab-go.

- **GIVEN** the fab-go binary with this change
- **WHEN** the user runs `fab setup`
- **THEN** it prints a "Yet to be implemented" message and exits 0
- **AND** `fab setup check` runs the doctor; `fab setup bogus` exits 2 (usage error, via the existing `run()` seam)

#### R2: Read-only invariant
`fab setup check` SHALL write nothing: no config mutation, no trust-store seeding, no agent/pane launches, no `.fab-*` state files, no prompts. Running it twice SHALL yield the same report (Constitution III). Help output for the new commands SHALL conform to the shll toolkit standards (layered help: short summary plus examples).

- **GIVEN** any machine state
- **WHEN** `fab setup check` runs once or repeatedly
- **THEN** no file is created or modified by the command and the report is identical across runs

### Probe Layer: internal package

#### R3: Reusable probe package
Probes SHALL live in a new internal package `src/go/fab/internal/setupcheck`, NOT inline in the command function. Each probe SHALL return structured results (per-finding check/subject/severity/detail, plus a provider roster row set) that the command renders; the command file SHALL own only input wiring, rendering, and exit-code mapping. The aggregated report SHALL be consumable by the future C2 wizard without shelling out.

- **GIVEN** the new package
- **WHEN** a caller invokes its aggregation entry point
- **THEN** it receives a structured report value with no rendering or process-exit logic embedded

#### R4: Provider roster probe
For each provider in the effective config (the four built-ins from embedded `defaults.yaml` plus user-defined providers merged via the config cascade), the probe SHALL report: binary presence — resolving the leading executable token of the provider's commands on PATH (unwrapping nested `sh -c '...'` forms so agy/kimi resolve to `agy`/`kimi`, not `sh`) — and declared capabilities (`interactive_command` / `headless_command` / `native`). A provider named by a depth knob (`agent.session` / `agent.workers`) or any `agent.profiles.<role>.provider` whose binary is not on PATH SHALL produce a failure-severity finding; an unconfigured provider's absence SHALL be informational only.

- **GIVEN** effective config naming provider `agy` for `agent.workers`, with no `agy` binary on PATH
- **WHEN** `fab setup check` runs
- **THEN** the report contains a failure finding naming `agy` and the run exits 1
- **AND** an unconfigured built-in provider absent from PATH (e.g. `codex` with default config) yields only an informational row

### Environment & Versions

#### R5: Environment facts
The probe SHALL report `$TMUX` presence using the existing `internal/dispatch` tmux signal classification (`TmuxAvailable` / `TmuxAbsent` from `pane_mode.go`), and the PATH presence of `gh`, `yq`, and `rk`. Absent `gh`/`yq` SHALL be warning-severity; absent `rk` SHALL be informational (fail-silent optional tooling convention).

- **GIVEN** an environment with `$TMUX` unset and no `rk` on PATH
- **WHEN** the doctor runs
- **THEN** it reports tmux as absent (pane rung not viable) and rk's absence as informational, without failing

#### R6: Version triplet and bottle-skew detection
The doctor SHALL report the version triplet — the running binary's version, the kit cache version (`<kit-dir>/VERSION`, resolved via `internal/kitpath`), and the project pin (`fab/.fab-version`) — and flag mismatches as warnings. Additionally it SHALL run the override-masking skew check: for each user-set capability-bearing override (`interactive_command`, `headless_command`, `native`) under `providers.<name>` in the system or project tier, introspect the binary's own embedded defaults (`internal/agent` embedded `defaults.yaml`, via a new exported lookup) and flag any override on a provider+key the embedded defaults do not define as "load-bearing against your installed binary — unsetting it changes behavior" (warning-severity, the #573 incident shape). Providers absent from the embedded table entirely (user-defined) SHALL be skipped — a definition is not masking.

- **GIVEN** a system config setting `providers.agy.interactive_command` against a binary whose embedded defaults lack agy interactive capability
- **WHEN** the doctor runs
- **THEN** it reports the override as load-bearing (warning) and still exits 0 when nothing else fails

### Config Sanity & Exit Contract

#### R7: Config-sanity findings
The doctor SHALL analyze the effective config read-only: when `dispatch.mode`'s preference would descend at dispatch time, it SHALL report the descent and why, reusing the exact reason strings `internal/dispatch.SelectMode` produces (`pane unavailable: no tmux`, `pane unavailable: no interactive_command`, `native unavailable`); a preference with NO reachable rung SHALL be failure-severity, an available descent SHALL be warning-severity. The depth-knob/provider-binary cross-check is R4's failure finding.

- **GIVEN** `dispatch.mode: pane` with `$TMUX` unset
- **WHEN** the doctor runs
- **THEN** it warns that pane would descend, naming `pane unavailable: no tmux`, and exits 0 (warnings-only)

#### R8: Exit-code contract
Exit 0 SHALL mean healthy or warnings-only; exit 1 SHALL mean at least one failure-severity finding; usage errors SHALL exit 2 via the existing `markRunReached`/`run()` seam (no new in-handler exit tier). Informational findings SHALL never affect the exit code.

- **GIVEN** a report with only warnings and informational findings
- **WHEN** the command completes
- **THEN** exit is 0; with any failure finding, exit is 1 with a stderr summary line

### Documentation & Tests

#### R9: Same-change obligations
The change SHALL update `src/kit/skills/_cli-fab.md` with the new command's signature and behavior (Constitution Additional Constraints), ship Go tests for the command and probe package in the same change (test-alongside), keep the help-dump/router contract tests in `cmd/fab` green, and leave `fab doctor` (fab-kit binary) untouched. No SPEC-mirror, migration, or skill-flow changes.

- **GIVEN** the completed change
- **WHEN** `go test` runs on the touched packages and the `cmd/fab` contract tests
- **THEN** all pass, and `_cli-fab.md` documents `fab setup` / `fab setup check`

### Non-Goals

- The interactive `fab setup` wizard (question flow, `fab config set` writes) — C2 / backlog [stpw]
- Trust-store seeding, pane warm-up — C3 candidates
- Network-dependent checks (release lookup, update checks)
- Provider auth/quota probing — deferred, revisit at C2/C3
- Artifact-based defaults comparison (shipping a comparable `defaults.yaml` counterpart into the kit tarball) — deferred; v1 is override-masking introspection
- Any change to fab-kit's `fab doctor` — coexist with distinct jobs

### Design Decisions

#### Subcommand form over a `--check` flag
**Decision**: The doctor is the `fab setup check` subcommand; bare `fab setup` prints "Yet to be implemented".
**Why**: Reserves the `fab setup` seat for the C2 wizard without shipping it; a subcommand reads as a distinct read-only operation (toolkit principle №5).
**Rejected**: `fab setup --check` flag — couples the doctor to the wizard's eventual flag surface.
*Introduced by*: 260811-pgbq-setup-check-doctor (clarify session 2026-08-11, question 11)

#### Override-masking via embedded-defaults introspection
**Decision**: Detect bottle skew by comparing user-tier capability overrides against the binary's OWN embedded defaults — an override on a provider+key the embedded defaults lack is load-bearing.
**Why**: Needs no reference artifact and reproduces the #573 incident shape exactly; env-var overrides cannot test this (top-tier shadowing only, empty leaves fall through).
**Rejected**: Artifact-based defaults comparison (no comparable counterpart in the kit cache today — deferred); env-var-based detection (structurally incapable).
*Introduced by*: 260811-pgbq-setup-check-doctor (clarify session 2026-08-11, question 12)

#### Warnings-only exits 0
**Decision**: Only failure-severity findings exit 1; warnings and informational findings exit 0; usage errors stay exit 2 via the existing seam.
**Why**: Keeps the doctor CI-able as "real problems" gating without crying wolf on advisory findings.
**Rejected**: A distinct warnings exit tier — an undocumented third runtime code would complicate the CI contract.
*Introduced by*: 260811-pgbq-setup-check-doctor (clarify session 2026-08-11, question 14)

## Tasks

### Phase 1: Setup

- [x] T001 Export the embedded-defaults provider lookup `BuiltinProvider(name) (config.ProviderConfig, bool)` from `src/go/fab/internal/agent/agent.go` (unmerged built-in table entry) + pin test in `src/go/fab/internal/agent/agent_test.go` <!-- R6 -->

### Phase 2: Core Implementation

- [x] T002 Create `src/go/fab/internal/setupcheck/setupcheck.go`: `Severity` enum (OK/Info/Warn/Fail), `Finding`, `ProviderProbe`, `Report` (with `HasFailures`), `LeadingExecutable` (nested `sh -c` unwrap), and the provider roster probe (`ProbeProviders` with injectable lookPath seam) — plus `setupcheck_test.go` covering token unwrapping and configured-vs-unconfigured missing-binary severities <!-- R3, R4 -->
- [x] T003 Add environment, version-triplet, and override-masking probes (`ProbeEnvironment`, `ProbeVersions`, `ProbeOverrideMasking`) in `src/go/fab/internal/setupcheck/` + tests (PATH/env seams, fixture kit dir, fixture system/project layer maps) <!-- R5, R6 -->
- [x] T004 Add the dispatch-mode viability probe (`ProbeDispatchMode`, reusing `dispatch.SelectMode` + its descent-reason strings) and the `Run` aggregation entry point in `src/go/fab/internal/setupcheck/` + tests <!-- R7, R3 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Create `src/go/fab/cmd/fab/setup.go`: `setupCmd()` (bare → "Yet to be implemented", exit 0) and `setupCheckCmd()` (wire inputs: `resolve.FabRoot` tolerate-absent, `config.LoadPath`/`LoadLayers`, `kitpath.KitDir`, `$TMUX`, `exec.LookPath`, binary `version`; render report; return operational error when `Report.HasFailures()`), register in `newRootCmd()` (`src/go/fab/cmd/fab/main.go`) <!-- R1, R2, R8 -->
- [x] T006 Create `src/go/fab/cmd/fab/setup_test.go`: bare-setup placeholder output/exit, `check` on a fixture repo (exit 0 warnings-only vs exit 1 on failure finding, usage error → 2), read-only assertion <!-- R1, R2, R8 -->

### Phase 4: Polish

- [x] T007 Add the `fab setup` section to `src/kit/skills/_cli-fab.md` (signature, probe set, severities, exit contract, coexistence with `fab doctor`) + Contents entry; grep-sweep aggregate specs (`docs/specs/`) for command-list restatements that must name the new command <!-- R9 -->
- [x] T008 Verify: `gofmt -l` clean on touched files, `go test` on `internal/agent`, `internal/setupcheck`, `cmd/fab` (incl. `helpdump_test.go`, `lifecycle_collision_test.go`, `examples_test.go`), wider `go test ./...` in `src/go/fab` if cross-cutting fallout appears <!-- R9 -->

## Execution Order

- T001 blocks T003 (override-masking introspects the embedded table through the new export)
- T002 blocks T003 and T004 (shared types)
- T002–T004 block T005 (command renders the package's report)
- T006 follows T005; T007–T008 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab setup` prints "Yet to be implemented" and exits 0; `fab setup check` runs the doctor; both are registered in `newRootCmd()`; `setup` is absent from the router allowlist
- [x] A-002 R2: Running `fab setup check` twice produces identical output and creates/modifies no files
- [x] A-003 R3: Probe logic lives in `internal/setupcheck` returning structured results; `cmd/fab/setup.go` holds only wiring, rendering, and exit mapping
- [x] A-004 R4: The provider roster reports binary presence (unwrapping `sh -c` wrappers to the provider executable) and interactive/headless/native capabilities for built-in + user-defined providers
- [x] A-005 R5: The report covers `$TMUX` (via the `internal/dispatch` classification), `gh`, `yq` (warn when absent), and `rk` (info when absent)
- [x] A-006 R6: The report shows the binary/kit-cache/project-pin version triplet, warns on mismatch, and flags user-tier capability overrides with no embedded default beneath them as load-bearing
- [x] A-007 R7: An unviable `dispatch.mode` is reported with the exact descent-reason strings from `internal/dispatch`; no-reachable-rung is failure-severity, a working descent is warning-severity
- [x] A-008 R8: Warnings-only report exits 0; any failure finding exits 1; usage errors exit 2
- [x] A-009 R9: `_cli-fab.md` documents the new command; `fab doctor` is untouched; no migrations or SPEC-mirror changes ship

### Behavioral Correctness

- [x] A-010 R4: A depth-knob/`agent.profiles` provider whose binary is missing produces a failure finding and exit 1; an unconfigured provider's missing binary is informational only
- [x] A-011 R6: The #573 shape is detected: a system-tier `providers.<name>.interactive_command` override against a binary lacking that embedded default is flagged load-bearing (the pinned test simulates it via fixture layers)

### Scenario Coverage

- [x] A-012 R1: `fab setup bogus` exits 2 through the existing `run()` classification (no new exit plumbing)
- [x] A-013 R8: Exit-code mapping is covered by command tests (0 healthy, 0 warnings-only, 1 failure, 2 usage)

### Edge Cases & Error Handling

- [x] A-014 R5: Outside a fab repo the doctor still runs (no project pin, system-tier config only) instead of erroring
- [x] A-015 R6: Unresolvable kit cache dir degrades to an informational finding, not a command failure
- [x] A-016 R4: Nested-shell headless commands (`sh -c 'agy ...'`, `sh -c 'kimi ...'`) resolve `agy`/`kimi`, never `sh`

### Code Quality

- [x] A-017 Pattern consistency: New files follow the existing per-command file pattern (`kitpath.go`/`resolve_agent.go` shape) and the internal-package doc-comment conventions
- [x] A-018 No unnecessary duplication: tmux classification, descent reasons, provider resolution, and config cascade are reused from `internal/dispatch` / `internal/agent` / `internal/config`, not re-implemented
- [x] A-019 Readability: probe functions stay focused (<50 lines) with named constants for severity strings; no magic strings
- [x] A-020 Composition: probes compose into `Report` via small functions; no god-function aggregator

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality (the `fab setup` command family + the `internal/setupcheck` probe package) without making existing code redundant; it reuses `internal/dispatch` (SelectMode, TmuxAvailability), `internal/agent` (ResolveProvider/ResolveRole, new BuiltinProvider export), `internal/config` (cascade loaders), and `internal/kitpath` rather than superseding any of them.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Package name is `internal/setupcheck` (intake's example name) | Follows the existing `internal/` package map; intake leaves final name to apply | S:70 R:80 A:75 D:70 |
| 2 | Confident | Probe input seams: injectable `LookPath` func, `$TMUX` passed as string, kit dir passed as path — mirroring the `dispatchLineFor` / `SelectMode` purity precedent | Keeps probes table-testable without PATH/tmux fixtures beyond stub funcs | S:70 R:75 A:80 D:75 |
| 3 | Confident | gh/yq absence is warning-severity, rk informational, tmux-absent informational unless it makes configured `dispatch.mode: pane` descend (then warning via R7) | Intake fixes rk as informational and warnings-only→exit 0; gh/yq are pipeline tooling so warn, but real problems are reserved for dispatch-breaking conditions | S:60 R:70 A:65 D:60 |
| 4 | Confident | Override-masking checks the three capability keys (`interactive_command`, `headless_command`, `native`) and skips providers absent from the embedded table | The #573 shape is capability-masking; a user-defined provider is a definition, not a mask | S:70 R:70 A:70 D:65 |
| 5 | Confident | Version-triplet mismatches are warnings (never failures); a `dev` binary version is reported but not compared | A mismatch is a skew signal, not proof of breakage; dev builds cannot compare meaningfully | S:60 R:70 A:65 D:60 |
| 6 | Certain | Failure exit rides the existing operational-error path (return error from RunE → exit 1 with stderr line); no `os.Exit` in the handler | Matches fab-go's `run()` convention; the pane/memory-index `os.Exit` pattern exists only for non-1 domain codes, which this command does not need | S:70 R:80 A:85 D:80 |

6 assumptions (1 certain, 5 confident, 0 tentative).
