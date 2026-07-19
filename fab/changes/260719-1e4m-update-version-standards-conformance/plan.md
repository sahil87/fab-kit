# Plan: Update & Version Standards Conformance

**Change**: 260719-1e4m-update-version-standards-conformance
**Intake**: `intake.md`

## Requirements

<!-- Derived from the intake's completed clause-by-clause audit (intake §"What Changes" 2–5).
     The intake is authoritative: the audit is not re-litigated here. -->

### Update Command: Brew-Handling Safety

#### R1: No SIGKILL / no hard timeout on brew subprocesses
`fab-kit update` MUST NOT send SIGKILL to a package-manager subprocess and MUST NOT cap `brew upgrade` with a short hard timeout (shll `update` standard, two MUST clauses). In `src/go/fab-kit/internal/update.go`, `brew update --quiet` and `brew upgrade fab-kit` SHALL run via plain `cmd.Run()` with inherited stdout/stderr — no bound, no kill path. The `runWithTimeout` helper SHALL be deleted outright (its only two callers are these brew invocations).

- **GIVEN** a `fab-kit update` run where `brew upgrade` stalls on a slow GitHub API call
- **WHEN** the stall exceeds what was previously the 120s bound
- **THEN** the brew process keeps running (output streams to the user; Ctrl-C/SIGINT remains available and brew traps it) — no code path kills brew mid-keg-swap

- **GIVEN** the `internal` package source after the change
- **WHEN** grepping for `runWithTimeout` and `Process.Kill` on the brew paths
- **THEN** neither exists — the absence of the kill path is enforced structurally by deletion

### Update Command: Non-Brew Install Degrades With Exit 0

#### R2: `update` exits 0 on non-brew installs; the internal sentinel is preserved
`fab-kit update` (and `fab update` via the router) SHALL exit 0 when the binary is not brew-installed, after printing the existing clear degrade message ("was not installed via Homebrew… Update manually, or reinstall with: brew install sahil87/tap/fab-kit"). The mapping happens at the command layer only — `updateCmd()` in `src/go/fab-kit/cmd/fab-kit/main.go` returns nil when `errors.Is(err, internal.ErrNotBrewInstalled)`. `internal.Update` MUST keep returning the `ErrNotBrewInstalled` sentinel unchanged, because `versionGuard` (`internal/sync.go`) depends on it to compose its "auto-update did not succeed" error and MUST keep treating not-brew as a guard failure (a too-old non-brew binary still blocks sync). Genuine brew failures (`brew update`, `brew upgrade`, `brew info`) still exit non-zero.

- **GIVEN** a go-install/manual/CI (non-Cellar) fab-kit binary
- **WHEN** `fab-kit update` runs
- **THEN** the degrade message prints and the process exits 0 (no false failure in a composed `shll update` run)

- **GIVEN** the same non-brew binary and a project pinned to a newer version
- **WHEN** `fab sync` trips `versionGuard`
- **THEN** the guard still fails the sync with actionable too-old instructions (the sentinel contract is unchanged inside `internal`)

### Tests: Pinning the Fixed Behavior

#### R3: Conformance behavior is pinned by tests
Tests MUST pin the fixed behavior (shll standards' verify checklists; Constitution VII — tests conform to the spec):
- `cmd/fab-kit/main_test.go`: (a) **version shape** — root command executed with `--version` and an injected release version exits success with first stdout line matching `^fab-kit version v\d+(\.\d+)*$`; (b) **help contract** — `update --help` output contains the literal substring `--skip-brew-update` (substring presence, not regex); (c) **non-brew exit 0** — `updateCmd`'s RunE returns nil on the not-brew path (under `go test` the test binary path never contains `/Cellar/`, so the real guard fires) while the existing `TestUpdate_NotBrewInstalledReturnsSentinel` continues to guard the internal sentinel.
- `internal/update_test.go`: **already-up-to-date exits 0** — fake brew reporting stable == currentVersion → `Update` returns nil and the brew log contains no `upgrade` invocation. Existing `TestUpdateSkipBrewUpdateGating` stays as-is (fake-brew harness unaffected by the wrapper deletion).

- **GIVEN** the fab-kit Go package test suites
- **WHEN** `go test ./...` runs in `src/go/fab-kit`
- **THEN** all tests pass, including the four new pins above

### Docs: `_cli-fab.md` Exit-Contract Row + Behavior-Claim Sweep

#### R4: The `update` exit-contract row states the new semantics; stale claims swept
The `update` row of the Workspace Command Exit Semantics table in `src/kit/skills/_cli-fab.md` MUST be rewritten to the new semantics: exit 0 with the degrade message on non-brew installs; non-zero only on genuine brew/upgrade failure (Constitution: CLI-behavior changes MUST update `_cli-fab.md`). The SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` MUST stay in sync (mirror rule). A repo-wide behavior-claim sweep MUST grep for prose (including user-facing string literals) claiming `update` exits non-zero on non-brew installs, updating every non-memory occurrence; `docs/memory/` occurrences (`distribution/distribution.md` §"`fab update` Exit Semantics", `distribution/kit-architecture.md` versionGuard DD) are hydrate's per the intake's Affected Memory design. Dated historical artifacts (findings reports, `log.md`/`log.seed.md` entries) record what past changes did and are not present-truth claims — out of sweep scope.

- **GIVEN** `src/kit/skills/_cli-fab.md` after the change
- **WHEN** reading the `update` exit-semantics row
- **THEN** it states exit 0 + degrade message on non-brew installs, non-zero only on genuine brew failure
- **AND** no non-memory, non-historical file still claims the old non-zero-on-non-brew behavior

### Non-Goals

- No SIGTERM-escalation / generous-bound machinery — conformance is achieved by deleting the timeout wrapper (intake Assumption 2)
- No `HOMEBREW_NO_GITHUB_API=1` on brew subprocesses (intake Assumption 4)
- No change to router `fab --version` output (intake Assumption 5) or to `--version` behavior (intake Assumption 6 — pinning test only)
- No migration (no user-data restructuring); no skill-behavior changes beyond the `_cli-fab.md` reference row
- `docs/memory/` updates (distribution.md, kit-architecture.md) — hydrate's job per the intake

### Design Decisions

#### Delete the timeout wrapper instead of bounding gracefully
**Decision**: Satisfy the brew-safety MUSTs by deleting `runWithTimeout` and running brew unbounded.
**Why**: The standard's verify checklist ("no code path sends `SIGKILL` to `brew`, and no short hard timeout caps `brew upgrade`") is satisfied cleanest by deletion; the standard explicitly suggests not reaching for a timeout at all. A hung brew is visible (output streams) in both interactive `update` and the `versionGuard` auto-update path; Ctrl-C (SIGINT, brew-trapped) remains available.
**Rejected**: A generous (tens-of-minutes) bound with SIGTERM + grace — conformant too, but adds signal-handling code and a hard-to-test escalation path for no concrete benefit.
*Introduced by*: 260719-1e4m-update-version-standards-conformance

#### Map the sentinel to exit 0 at the command layer only
**Decision**: `updateCmd` RunE returns nil on `errors.Is(err, internal.ErrNotBrewInstalled)`; `internal.Update` keeps returning the sentinel.
**Why**: The standard requires degrading "instead of erroring" and exiting non-zero only on genuine failure; `shll update` delegation would otherwise read false failures. `versionGuard`'s too-old-blocks-sync contract is preserved via the internal sentinel.
**Rejected**: Making `internal.Update` return nil on the not-brew path — would silently re-defeat `versionGuard` for non-brew installs (the exact F19 regression the sentinel was introduced to fix).
*Introduced by*: 260719-1e4m-update-version-standards-conformance

## Tasks

### Phase 2: Core Implementation

- [x] T001 Delete `runWithTimeout` in `src/go/fab-kit/internal/update.go`; run `brew update --quiet` and `brew upgrade fab-kit` via plain `cmd.Run()` (inherited stdout/stderr, no bound, no kill path); drop the now-unused `time` import <!-- R1 -->
- [x] T002 In `src/go/fab-kit/cmd/fab-kit/main.go` `updateCmd()`: map `errors.Is(err, internal.ErrNotBrewInstalled)` → return nil (exit 0, degrade message already printed); add the `errors` import <!-- R2 -->

### Phase 3: Integration & Edge Cases (tests)

- [x] T003 [P] Add `TestUpdate_AlreadyUpToDateExitsZero` to `src/go/fab-kit/internal/update_test.go`: fake brew reports stable == currentVersion → `Update` returns nil, brew log contains no `upgrade` <!-- R3 -->
- [x] T004 [P] Add to `src/go/fab-kit/cmd/fab-kit/main_test.go`: (a) version-shape test (root cmd `--version` with injected version, exit success, first line matches `^fab-kit version v\d+(\.\d+)*$`); (b) help-contract test (`update --help` contains literal `--skip-brew-update`); (c) non-brew-exit-0 test (`update` via rootCmd returns nil under go test's non-Cellar binary path) <!-- R3 -->
- [x] T005 Run the affected Go package tests: `go test ./...` in `src/go/fab-kit` (scope first; widen only if cross-cutting) <!-- R3 -->

### Phase 4: Polish (docs + sweep)

- [x] T006 Rewrite the `update` row of the Workspace Command Exit Semantics table in `src/kit/skills/_cli-fab.md`: exit 0 + degrade message on non-brew installs (shll update standard); non-zero only on genuine brew/upgrade failure; note the internal sentinel/versionGuard contract is unchanged. Sync the mirror `docs/specs/skills/SPEC-_cli-fab.md` (Calling Convention row's Workspace Command Exit Semantics coverage) <!-- R4 -->
- [x] T007 Repo-wide behavior-claim sweep: grep for prose + string literals claiming `update` exits non-zero on non-brew installs (e.g. "not installed via Homebrew", "non-brew", "ErrNotBrewInstalled", "exits non-zero"); update every non-memory, non-historical occurrence; record memory occurrences for hydrate <!-- R4 -->

## Execution Order

- T001 and T002 before T005 (tests run against the fixed code)
- T003/T004 are parallel-safe (different files); both before T005
- T006/T007 independent of the Go tasks

## Acceptance

### Functional Completeness

- [x] A-001 R1: `runWithTimeout` is deleted from `src/go/fab-kit/internal/update.go`; both brew invocations use plain `cmd.Run()` with inherited stdout/stderr; no `Process.Kill` or timeout machinery remains on the brew paths
- [x] A-002 R2: `updateCmd()` RunE returns nil when `internal.Update` returns `ErrNotBrewInstalled`; `internal.Update` still returns the sentinel on the not-brew path
- [x] A-003 R3: The four new test pins exist (already-up-to-date, version shape, help contract, non-brew exit 0) and pass
- [x] A-004 R4: The `_cli-fab.md` `update` exit-contract row states exit 0 on non-brew installs and non-zero only on genuine brew failure

### Behavioral Correctness

- [x] A-005 R2: Behavior change is the exit code only (1 → 0 on non-brew installs); the degrade message text is unchanged; brew failures still exit non-zero
- [x] A-006 R2: `versionGuard`'s contract is untouched — a too-old non-brew binary still fails `fab sync` with actionable instructions (existing `TestUpdate_NotBrewInstalledReturnsSentinel` still passes unchanged)

### Scenario Coverage

- [x] A-007 R3: `TestUpdateSkipBrewUpdateGating` still passes with the wrapper deleted (flag honoring pinned; fake-brew harness unaffected)
- [x] A-008 R1: No test or code path reintroduces a kill/timeout on the brew subprocesses (review greps `Process.Kill` / exec timeouts in `internal/update.go`)

### Edge Cases & Error Handling

- [x] A-009 R3: Already-up-to-date run exits 0 and does not invoke `brew upgrade`
- [x] A-010 R4: No non-memory, non-historical prose still claims `update` exits non-zero when not brew-installed; the memory occurrences (`distribution/distribution.md`, `distribution/kit-architecture.md`) are recorded for hydrate

### Code Quality

- [x] A-011 Pattern consistency: New tests follow the existing fake-brew / package-var-seam / t.Setenv patterns of `update_test.go` and the rootCmd-based patterns of `main_test.go`
- [x] A-012 No unnecessary duplication: Existing test helpers (`fakeBrewScript`, isBrewInstalled seam) reused; no new utilities duplicating them
- [x] A-013 Canonical source only: No edits under `.claude/skills/` (deployed copies); kit docs edited at `src/kit/skills/_cli-fab.md`
- [x] A-014 Go changes ship tests: both touched Go files carry same-change test updates (Constitution VII / code-review rule)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change already removed the only redundant code (`runWithTimeout` in `src/go/fab-kit/internal/update.go`, deleted by T001 along with its sole callers and the now-unused `time` import). No further files, functions, branches, or config are left unused; the `ErrNotBrewInstalled` sentinel and `versionGuard`'s post-state check are deliberately retained (still load-bearing for `sync`'s too-old-blocks-sync contract).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The non-brew-exit-0 command-layer test relies on the natural go-test property (the test binary path never contains `/Cellar/`, so the real `isBrewInstalled` guard fires) instead of exporting a test seam from `internal` | The inverse of this property is already load-bearing in `internal/update_test.go` (its comments document it); exporting a seam would add API surface for no benefit; the sentinel side stays pinned by the existing internal test | S:55 R:90 A:85 D:75 |
| 2 | Confident | The behavior-claim sweep updates non-memory sources only; the two stale `docs/memory/distribution/` claims (distribution.md §"`fab update` Exit Semantics", kit-architecture.md versionGuard DD "so `fab update` exits non-zero") are deferred to hydrate, with kit-architecture.md flagged as an addition to the intake's Affected Memory list | The pipeline assigns memory writes to hydrate (fab-continue Key Properties; intake §5 explicitly assigns distribution.md to hydrate); dated historical artifacts (findings reports, log.md/log.seed.md) are change history, not present-truth claims, and stay verbatim | S:60 R:85 A:85 D:75 |
| 3 | Confident | SPEC-_cli-fab.md mirror sync is a minimal Calling Convention row touch-up (noting update's exit-0 degrade) — the mirror is inventory-granularity and carries no stale claim | Constitution's mirror rule + code-review's strict reading ("treat the whole mirror class as in-scope") is satisfied with the smallest accurate addition; restating the full row would duplicate the reference doc | S:50 R:90 A:85 D:70 |

3 assumptions (0 certain, 3 confident, 0 tentative).
