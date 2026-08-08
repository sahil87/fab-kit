# Plan: FAB_KIT_PATH — session-scoped kit-resolution override for kit development

**Change**: 260808-j9rb-kit-path-override
**Intake**: `intake.md`

## Requirements

### fab-go: Kit Directory Resolution

#### R1: Honor `FAB_KIT_PATH` at the fab-go kit-resolution seam

`internal/kitpath.KitDir()` MUST preserve the test-only `SetOverride` as its highest-precedence source, then honor a non-empty `FAB_KIT_PATH`, absolutize the environment value with `filepath.Abs`, and require it to name an existing directory. An invalid environment override MUST return an error that names `FAB_KIT_PATH` and MUST NOT fall back to the executable-sibling kit. With no override, executable and symlink resolution MUST remain unchanged.

- **GIVEN** `SetOverride` and `FAB_KIT_PATH` are both non-empty
- **WHEN** `KitDir()` resolves the kit directory
- **THEN** it returns the test override unchanged

- **GIVEN** only `FAB_KIT_PATH` is set to a relative existing directory
- **WHEN** `KitDir()` resolves the kit directory
- **THEN** it returns that directory as an absolute path

- **GIVEN** `FAB_KIT_PATH` names a missing path or regular file
- **WHEN** `KitDir()` resolves the kit directory
- **THEN** it returns a loud error naming `FAB_KIT_PATH` without trying the executable-sibling path

### fab-kit: Reader Resolution and Sync

#### R2: Resolve fab-kit reader paths through the environment override

The fab-kit internal package MUST expose one override-aware kit-directory resolver that returns a validated absolute `FAB_KIT_PATH` when set and otherwise preserves `CachedKitDir(version)`'s local-cache-before-remote-cache behavior. `fab migrations-status` MUST use this resolver for `VERSION` and migration discovery, failing loudly on an invalid override.

- **GIVEN** a managed repository and an override kit with an unshipped `VERSION` or migration set
- **WHEN** `fab migrations-status` runs
- **THEN** it reads the engine data from the override directory rather than either version cache

- **GIVEN** `FAB_KIT_PATH` is unset
- **WHEN** a fab-kit reader resolves its kit
- **THEN** the existing local-then-remote cache result is unchanged

#### R3: Run sync from the override without cache or version coupling

`Sync()` MUST validate and use `FAB_KIT_PATH` as its kit source when set, print exactly `kit: <absolute-dir> (FAB_KIT_PATH override)`, and skip both `versionGuard` and `EnsureCached`. Prerequisite checks, workspace scaffolding, skill deployment, direnv allow, project sync scripts, shim/project flag behavior, failure propagation, and the final success output MUST otherwise remain unchanged and operate against the override kit where kit content is required.

- **GIVEN** a valid override kit, no usable cached kit content, and a kit version newer than the running binary
- **WHEN** a full sync runs
- **THEN** it succeeds from the override without a cache download or version-guard failure, prints provenance, deploys override content, and runs project scripts

- **GIVEN** an invalid override path
- **WHEN** any sync mode runs
- **THEN** sync fails loudly instead of falling back to cached content

### Lifecycle Safety and Provenance

#### R4: Refuse version-stamping commands under an active override

`fab init` and `fab upgrade-repo` MUST reject a non-empty `FAB_KIT_PATH` before repository discovery, network/cache work, sync, or version stamping. Each error MUST name `FAB_KIT_PATH`, the attempted command, and instruct the user to unset the variable first.

- **GIVEN** `FAB_KIT_PATH` is set
- **WHEN** `fab init` or `fab upgrade-repo` is invoked
- **THEN** the command returns before any stamp, download, cache mutation, or sync work

#### R5: Surface doctor provenance without changing health semantics

In normal output, `fab doctor` MUST print `kit: <absolute-dir> (FAB_KIT_PATH override)` when and only when `FAB_KIT_PATH` is non-empty. The line MUST remain informational, outside the seven prerequisite results, failure count, summary denominator, and exit-code contract. `--porcelain` MUST remain errors-only and omit informational provenance.

- **GIVEN** a valid `FAB_KIT_PATH`
- **WHEN** normal doctor checks run
- **THEN** the provenance line appears while the summary still reports exactly seven checks

- **GIVEN** the variable is unset or doctor runs with `--porcelain`
- **WHEN** doctor checks run
- **THEN** no provenance line is printed

### Development Ergonomics and Documentation

#### R6: Activate the override per fab-kit development worktree

The repository's tracked `.envrc` MUST export `FAB_KIT_PATH=$PWD/src/kit` so each allowed worktree resolves its own kit source. The user-repository scaffold `src/kit/scaffold/fragment-.envrc` MUST remain unchanged.

- **GIVEN** direnv has loaded this repository's `.envrc`
- **WHEN** a developer enters any fab-kit worktree
- **THEN** `FAB_KIT_PATH` points at that worktree's `src/kit`

#### R7: Document the override without making it configuration

`src/kit/skills/_cli-fab.md` and `docs/specs/skills/SPEC-_cli-fab.md` MUST describe the sync, migrations-status, kit-path, doctor, init, and upgrade-repo override contracts consistently. `docs/specs/config.md` MUST state that `FAB_KIT_PATH` is deliberately environment-only and absent from the registry/config cascade because kit resolution precedes it. Apply MUST NOT edit `.claude/skills/`, memory content, the historical source plan, a config registry row, a migration, or router `cmd/fab/main.go`.

- **GIVEN** a reader consults the canonical CLI reference, its SPEC mirror, or the config spec
- **WHEN** they look for kit source selection
- **THEN** they see one consistent fail-loud, per-process override contract and no persistent config field

### Verification

#### R8: Ship hermetic tests with the Go changes

Tests MUST cover valid, relative/absolute, invalid, unset, precedence, provenance, cache-skip, migrations-status, lifecycle-refusal, and doctor behavior. Existing tests that rely on default kit resolution or lifecycle commands MUST neutralize ambient `FAB_KIT_PATH` with `t.Setenv` so loading the repository `.envrc` cannot change suite results. All edited Go files MUST be formatted with `gofmt`.

- **GIVEN** the test process inherits a developer-shell `FAB_KIT_PATH`
- **WHEN** the affected Go package suites run
- **THEN** default-path tests remain hermetic and explicit override tests exercise only their declared fixture

### Non-Goals

- Changing fab-go binary selection or the router's `syscall.Exec` environment propagation
- Adding a registry/config field, persisted override, migration, or `fab sync --source` flag
- Auto-detecting this repository as a kit source
- Updating memory content during apply; the hydrate stage owns those edits
- Engineering around the first-sync-before-direnv worktree edge; a later idempotent `fab sync` repairs it

### Design Decisions

#### Environment-only resolution at two choke points

**Decision**: Resolve `FAB_KIT_PATH` only at fab-go's `KitDir()` and fab-kit's reader resolver; version-stamping lifecycle commands refuse it.
**Why**: Every content reader follows one per-process source without persisting a teammate-specific path or mixing arbitrary content with a release stamp.
**Rejected**: A sync-only flag, repository autodetection, and a registry/config field because each creates reader skew, invisible environment divergence, or persistent stale-kit state.
*Introduced by*: 260808-j9rb-kit-path-override

## Tasks

### Phase 1: Core Resolution

- [x] T001 Implement and test fab-go `FAB_KIT_PATH` resolution and `SetOverride` precedence in `src/go/fab/internal/kitpath/kitpath.go` and `kitpath_test.go`. <!-- R1 -->
- [x] T002 Implement and test the override-aware fab-kit reader seam in `src/go/fab-kit/internal/cache.go` and `cache_test.go`. <!-- R2 -->

### Phase 2: Lifecycle Integration

- [x] T003 Wire override resolution, provenance, cache/version-guard skipping, and hermetic integration coverage into `src/go/fab-kit/internal/sync.go`, `sync_test.go`, and `sync_integration_test.go`. <!-- R3 -->
- [x] T004 Add early override refusals and tests for `Init` and `Upgrade` in `src/go/fab-kit/internal/init.go`, `init_test.go`, `upgrade.go`, and `upgrade_test.go`. <!-- R4 -->
- [x] T005 Route `fab migrations-status` through the override-aware resolver and add command coverage in `src/go/fab-kit/cmd/fab-kit/migrations_status.go` and its tests. <!-- R2 -->
- [x] T006 Add conditional informational provenance and coverage to `src/go/fab-kit/cmd/fab-kit/doctor.go` and `doctor_test.go` without changing the seven-check contract. <!-- R5 -->

### Phase 3: Development Surface and Documentation

- [x] T007 Add the worktree-local export to `.envrc` and verify `src/kit/scaffold/fragment-.envrc` remains unchanged. <!-- R6 -->
- [x] T008 Update `src/kit/skills/_cli-fab.md`, `docs/specs/skills/SPEC-_cli-fab.md`, and `docs/specs/config.md`; sweep the mirror/claim class without editing memory content or deployed `.claude/skills/`. <!-- R7 -->

### Phase 4: Verification

- [x] T009 Neutralize ambient overrides in default-path tests, run `gofmt`, run affected Go package tests followed by both Go module suites, and perform non-goal/mirror sweeps. <!-- R8 -->

## Execution Order

- T001 and T002 are independent; T002 blocks T003 and T005.
- T003, T004, T005, and T006 can proceed after T002; T007 and T008 are independent of those code integrations.
- T009 runs after every implementation and documentation task.

## Acceptance

### Functional Completeness

- [x] A-001 R1: fab-go resolves a valid relative or absolute `FAB_KIT_PATH`, returns an absolute directory, preserves `SetOverride` precedence, and preserves unset fallback behavior.
- [x] A-002 R2: fab-kit reader resolution and migrations-status use a valid override and preserve local-then-remote cache behavior when unset.
- [x] A-003 R3: sync uses override content, prints the exact provenance line, and does not require cache content or a compatible pinned/system version under the override.
- [x] A-004 R4: init and upgrade-repo refuse an active override before version-stamping or other lifecycle work.
- [x] A-005 R5: normal doctor output conditionally reports override provenance without altering the seven-check result or exit-count semantics.
- [x] A-006 R6: the dev `.envrc` exports the worktree's `src/kit`, while the user scaffold fragment is byte-unchanged.
- [x] A-007 R7: canonical CLI/config docs and the SPEC mirror consistently describe an environment-only override and its safety boundaries.
- [x] A-008 R8: affected Go behavior is covered by hermetic tests and edited Go files are formatted.

### Behavioral Correctness

- [x] A-009 R1: an invalid fab-go override errors with `FAB_KIT_PATH` in the message and never silently falls back.
- [x] A-010 R2: an invalid fab-kit override errors with `FAB_KIT_PATH` in the message and never silently uses cached kit content.
- [x] A-011 R3: sync still performs prerequisites, scaffolding, deployment, direnv, and project scripts from the override as selected by its existing flags.
- [x] A-012 R4: lifecycle refusal errors identify the command and tell the user to unset `FAB_KIT_PATH`.
- [x] A-013 R5: doctor's override line is not counted as an eighth check and is absent from errors-only porcelain output.

### Scenario Coverage

- [x] A-014 R1: tests cover override honored, relative-path absolutization, missing/file rejection, unset fallback, and `SetOverride` precedence.
- [x] A-015 R2: tests cover valid/invalid/unset fab-kit resolution and migrations-status reading override engine data.
- [x] A-016 R3: an integration test proves sync can deploy override content with no usable cached kit and with a version that would otherwise trip the guard.
- [x] A-017 R4: tests prove both init and upgrade-repo refuse before their normal preconditions or mutation paths.
- [x] A-018 R5: doctor tests cover set, unset, and porcelain provenance behavior.

### Edge Cases & Error Handling

- [x] A-019 R1: a regular file supplied as `FAB_KIT_PATH` is rejected as not a directory.
- [x] A-020 R2: relative fab-kit override values are normalized against the process working directory before use.
- [x] A-021 R3: project-only sync still surfaces an active override and invalid overrides fail loudly rather than disappearing behind flag-specific control flow.
- [x] A-022 R8: default-resolution tests explicitly clear ambient `FAB_KIT_PATH` so the tracked `.envrc` cannot contaminate them.

### Code Quality

- [x] A-023 Pattern consistency: New code follows existing Go naming, error-wrapping, package-boundary, and test-fixture patterns.
- [x] A-024 No unnecessary duplication: fab-kit override parsing/validation is centralized and reused by sync, migrations-status, doctor display, and lifecycle guards where applicable.
- [x] A-025 Readability and maintainability: override branches are explicit, focused, and preserve the existing sync pipeline structure.
- [x] A-026 Composition over duplication: existing cache resolution, lifecycle seams, and command writers are retained instead of reimplemented.
- [x] A-027 No magic strings: the `FAB_KIT_PATH` name is a package constant in each Go module that consumes it.
- [x] A-028 Mirror discipline: `_cli-fab.md` and `SPEC-_cli-fab.md` are updated together, and a repo-wide claim sweep finds no contradictory apply-owned documentation.
- [x] A-029 Test integrity: tests assert the intake/plan contracts without bending production behavior around fixtures.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- Memory changes identified by the sibling sweep remain deferred to hydrate.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Doctor provenance is normal-output information and remains absent under `--porcelain`. | The existing flag contract is explicitly errors-only, while the intake calls the new line informational and outside the seven checks; preserving both has one clear interpretation. | S:70 R:90 A:85 D:80 |
| 2 | Confident | Doctor displays the absolutized configured override even when directory validation would fail for reader commands, without adding an eighth health check. | Mandatory provenance exists to expose lingering exports, and the intake forbids changing the seven-check/failure-count contract; showing the selected path is more informative than suppressing it. | S:60 R:90 A:80 D:75 |

2 assumptions (0 certain, 2 confident, 0 tentative).
