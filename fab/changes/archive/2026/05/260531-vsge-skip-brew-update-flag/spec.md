# Spec: Add --skip-brew-update flag to update command

**Change**: 260531-vsge-skip-brew-update-flag
**Created**: 2026-05-31
**Affected memory**: `(none)`

<!--
  Implementation-only, behavior-preserving additive flag. Default (flag absent)
  exactly preserves current `fab-kit update` behavior, so no memory file's
  documented contract changes. The `update` command's internal `brew update`
  step is not documented as spec-level behavior in docs/memory/. See the intake's
  Affected Memory section. Do NOT invent a memory file for this change.
-->

## Non-Goals

- **NOT** refactoring the exec/subprocess convention in `update.go` — commands stay built with `exec.Command(...)` + `runWithTimeout(...)` exactly as today. The contract explicitly forbids an exec-style refactor.
- **NOT** changing default behavior — with the flag absent, the full sequence (`brew update` + `brew info` + up-to-date short-circuit + `brew upgrade`) runs byte-for-byte as it does today.
- **NOT** touching the `brew info` version check (`brewLatestVersion()`), the up-to-date short-circuit (`latest == currentVersion`), or the `brew upgrade` invocation — the flag gates *only* the `brew update --quiet` tap-metadata refresh.
- **NOT** changing, aliasing, or aliasing-away the flag name — it is fixed at exactly `--skip-brew-update` by the cross-toolkit contract shared across 6 sibling tools. fab-kit MUST NOT introduce local naming variation.
- **NOT** adding memory or spec-doc updates — additive, behavior-preserving flag; no documented contract changes. (`_cli-fab.md` documents the `fab` workflow CLI surface, not the `fab-kit` workspace binary's `update` subcommand, so it is out of scope here.)

## CLI: The `--skip-brew-update` Flag

### Requirement: Flag declaration and wiring in `updateCmd()`

The `update` command MUST expose a boolean flag named EXACTLY `--skip-brew-update`. The flag SHALL be registered in `updateCmd()` (in `src/go/fab-kit/cmd/fab-kit/main.go`) via `cmd.Flags().BoolVar`, following the same construction pattern already used by `syncCmd()` in the same file: declare a local `bool`, build the command, register the flag, and return the command. The flag's default value MUST be `false`. The flag's help text SHOULD make clear that `brew info` and `brew upgrade` still run when it is set.

#### Scenario: Flag is registered with the exact name and default

- **GIVEN** the `fab-kit` binary
- **WHEN** the user runs `fab-kit update --help`
- **THEN** the output MUST list a flag named `--skip-brew-update`
- **AND** the flag's default value MUST be `false`

#### Scenario: Construction follows the syncCmd pattern

- **GIVEN** `updateCmd()` in `main.go`
- **WHEN** the flag is wired
- **THEN** a local `var skipBrewUpdate bool` MUST be declared, the command built, `cmd.Flags().BoolVar(&skipBrewUpdate, "skip-brew-update", false, <help>)` registered, and `cmd` returned
- **AND** the construction MUST mirror the existing `syncCmd()` flag-wiring pattern, not introduce a new style

#### Scenario: Unknown flag still rejected

- **GIVEN** the `fab-kit` binary
- **WHEN** the user runs `fab-kit update --skip-brew-updates` (typo, trailing `s`)
- **THEN** cobra MUST reject the unknown flag and the command MUST NOT run the update sequence

### Requirement: `RunE` passes the flag value to `Update()`

The `update` command's `RunE` MUST invoke `internal.Update(version, skipBrewUpdate)`, passing the resolved flag value through unmodified. `updateCmd()` SHALL remain the sole caller of `internal.Update()`.

#### Scenario: Flag value threaded to the update logic

- **GIVEN** the user runs `fab-kit update --skip-brew-update`
- **WHEN** `RunE` executes
- **THEN** `internal.Update` MUST be called with `skipBrewUpdate == true`

#### Scenario: Default value threaded when flag is absent

- **GIVEN** the user runs `fab-kit update` with no flags
- **WHEN** `RunE` executes
- **THEN** `internal.Update` MUST be called with `skipBrewUpdate == false`

## Update Logic: Gating the tap-metadata refresh

### Requirement: `Update()` accepts the skip parameter

`internal.Update` MUST change its signature from `Update(currentVersion string) error` to `Update(currentVersion string, skipBrewUpdate bool) error`. The new parameter SHALL be a plain `bool` (not an options struct), preserving the existing positional-parameter style of `Update(currentVersion string)`.

#### Scenario: Signature threads a plain bool

- **GIVEN** the `internal` package
- **WHEN** `Update` is defined
- **THEN** its signature MUST be `func Update(currentVersion string, skipBrewUpdate bool) error`
- **AND** the package MUST compile and the sole caller (`updateCmd()`) MUST pass the bool

### Requirement: Skip gates ONLY the `brew update --quiet` block

When `skipBrewUpdate` is `true`, `Update()` MUST skip the `brew update --quiet` tap-metadata refresh and the surrounding `fmt.Println("Checking for updates...")` block. All other steps MUST run unchanged: the `isBrewInstalled()` guard, the `brewLatestVersion()` / `brew info --json=v2` version check, the up-to-date short-circuit (`if latest == currentVersion`), and the `brew upgrade fab-kit` invocation. The existing exec construction MUST NOT be refactored; the only structural change is wrapping the `brew update` block in `if !skipBrewUpdate { ... }` (with the `brew upgrade` command consequently declared via `:=` rather than reassigned via `=`).

#### Scenario: Skip omits brew update but still upgrades

- **GIVEN** `fab-kit` is installed via Homebrew and a newer version is available
- **WHEN** `Update(currentVersion, true)` runs
- **THEN** `brew update` MUST NOT be invoked
- **AND** the `brew info` version check MUST still run
- **AND** `brew upgrade fab-kit` MUST still run

#### Scenario: Skip still honors the up-to-date short-circuit

- **GIVEN** `fab-kit` is installed via Homebrew and the latest version equals the current version
- **WHEN** `Update(currentVersion, true)` runs
- **THEN** `brew update` MUST NOT be invoked
- **AND** the `brew info` version check MUST still run
- **AND** the command MUST report "Already up to date" and MUST NOT run `brew upgrade`

#### Scenario: Skip still honors the brew-install guard

- **GIVEN** `fab-kit` was NOT installed via Homebrew (`isBrewInstalled()` is false)
- **WHEN** `Update(currentVersion, true)` runs
- **THEN** the manual-install guidance MUST be printed
- **AND** no brew subcommand (`update`, `info`, or `upgrade`) MUST be invoked

### Requirement: Default behavior is exactly preserved

When `skipBrewUpdate` is `false` (the default, flag absent), `Update()` MUST behave byte-for-byte as it does today: it MUST run `brew update --quiet`, then `brew info`, then the up-to-date short-circuit, then `brew upgrade fab-kit`.

#### Scenario: Default runs the full sequence

- **GIVEN** `fab-kit` is installed via Homebrew and a newer version is available
- **WHEN** `Update(currentVersion, false)` runs
- **THEN** `brew update` MUST run
- **AND** the `brew info` version check MUST run
- **AND** `brew upgrade fab-kit` MUST run

#### Scenario: brew update failure still aborts in default mode

- **GIVEN** `skipBrewUpdate == false` and `brew update --quiet` exits non-zero
- **WHEN** `Update(currentVersion, false)` runs
- **THEN** `Update` MUST return the wrapped "could not check for updates (brew update failed)" error
- **AND** `brew upgrade` MUST NOT run (current behavior unchanged)

## Testing: Update package coverage

### Requirement: New test asserts the flag-gating invariant

A new test file `src/go/fab-kit/internal/update_test.go` MUST be added (none exists today). It MUST follow the repo's existing convention seen in `download_test.go` and `sync_test.go`: use `t.TempDir()` for fixtures and `t.Setenv()` for environment, and exercise real subprocess invocation by placing a fake `brew` executable on `PATH` rather than mocking the exec layer. The fake `brew` MUST log each invocation's subcommand (`$1` → `update` / `info` / `upgrade`) to a file, and MUST emit valid `--json=v2` output for `brew info` reporting a stable version DIFFERENT from `currentVersion` (so the up-to-date short-circuit does not fire and `brew upgrade` is reached). The test MUST assert: with skip enabled, the log contains `info` and `upgrade` but NOT `update`; with skip disabled (default), the log contains `update`, `info`, AND `upgrade`.

#### Scenario: Test asserts skip omits update but keeps upgrade

- **GIVEN** a fake `brew` on `PATH` logging subcommands and returning a newer version from `brew info`
- **WHEN** the test drives the update logic with `skipBrewUpdate == true`
- **THEN** the brew log MUST NOT contain `update`
- **AND** the brew log MUST contain `info`
- **AND** the brew log MUST contain `upgrade`

#### Scenario: Test asserts default runs all three

- **GIVEN** a fake `brew` on `PATH` logging subcommands and returning a newer version from `brew info`
- **WHEN** the test drives the update logic with `skipBrewUpdate == false`
- **THEN** the brew log MUST contain `update`, `info`, AND `upgrade`

### Requirement: Build and update-package tests pass before PR

`go build ./...` and the update-package tests (`go test ./internal/...`, or scoped `go test ./internal/ -run Update`) MUST pass from `src/go/fab-kit/` before the PR is opened.

#### Scenario: Build and tests green pre-PR

- **GIVEN** the implemented change on the branch
- **WHEN** `go build ./...` and `go test ./internal/...` run from `src/go/fab-kit/`
- **THEN** both MUST exit zero
- **AND** the new update-package test MUST be among those that pass

## Design Decisions

1. **Plain `bool` parameter vs. options struct**: thread a plain `skipBrewUpdate bool` through `Update()`.
   - *Why*: matches the cross-toolkit contract wording ("thread `skipBrewUpdate bool` through `Update()`") and the existing positional `Update(currentVersion string)` signature; minimal, least-surprising surface for a single boolean.
   - *Rejected*: an options/config struct — over-engineered for one flag, diverges from the existing signature style and from the sibling tools' shared contract.

2. **`BoolVar` following the `syncCmd()` pattern**: register the flag with `cmd.Flags().BoolVar(&skipBrewUpdate, "skip-brew-update", false, <help>)` after building the command, then return it.
   - *Why*: `syncCmd()` in the same `main.go` already uses exactly this idiom (`var x bool` → build cmd → `BoolVar` → return); reusing it keeps the file consistent and satisfies the contract's "wire a real cobra BoolVar".
   - *Rejected*: persistent flags or a separate flag-set abstraction — unnecessary for a command-local flag and inconsistent with the established local pattern.

3. **Test seam for `isBrewInstalled()`** *(design note — Tentative, resolved at apply; NOT a blocking clarification)*: under `go test`, `os.Executable()` resolves to the test binary, whose path does not contain `/Cellar/`, so `isBrewInstalled()` returns `false` and `Update()` short-circuits at the guard before reaching any brew call. To exercise the flag-gating invariant the test must reach the brew sequence. The apply stage SHALL pick the lowest-touch seam that does NOT alter production exec behavior — the two viable candidates are (a) extracting the brew-sequence body into a small internal helper the test calls directly, or (b) making `isBrewInstalled` an overridable package-level func var stubbed in the test. Both preserve the exec/subprocess convention and the default runtime behavior; the flag-gating assertions are the invariant regardless of seam.
   - *Why*: refactoring the exec style is prohibited by the contract, but adding a narrow test seam is permitted; deferring the exact seam keeps the spec from over-constraining a trivially reversible apply-time choice.
   - *Rejected*: asserting on `Update()` with the guard returning early — defeats the test's purpose (never reaches the brew sequence). Also rejected: refactoring how commands are constructed/run to inject a fake exec — explicitly forbidden by the contract.
   <!-- clarified: auto-mode verified the cross-toolkit contract against the live codebase (update.go L16/L26-53, main.go syncCmd L82-96 / updateCmd L99-106, sole internal.Update caller, download_test.go + sync_test.go convention, no update_test.go yet). Contract is fully specified; the only open item is this test seam, which is NON-blocking — both candidate seams preserve the exec convention and default runtime behavior, the flag-gating assertions are seam-invariant, and the choice is trivially reversible at apply. Not escalated to blocking. -->

## Assumptions

<!-- SCORING SOURCE: fab score reads only this table. Carried forward from intake.md's
     8 assumptions, each confirmed/upgraded based on spec-level analysis (grounded in
     reading update.go, main.go, download_test.go, sync_test.go). The single Tentative
     (test seam) is a deferred-to-apply design note, not a blocking clarification. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Flag name is EXACTLY `--skip-brew-update` | Confirmed from intake #1 — fixed by cross-toolkit contract, stated verbatim, zero discretion | S:100 R:90 A:100 D:100 |
| 2 | Certain | Flag default is `false`; absent = current behavior preserved byte-for-byte | Confirmed from intake #2 — explicit in contract; cobra `BoolVar` default `false` is the idiomatic encoding and is captured by the default-preservation requirement | S:100 R:90 A:100 D:100 |
| 3 | Certain | Skip gates ONLY the `brew update --quiet` block; `brew info`, up-to-date short-circuit, and `brew upgrade` all run unchanged | Confirmed from intake #3 and verified against update.go (L26–53) — the only structural edit is wrapping L27–33 in `if !skipBrewUpdate` | S:100 R:90 A:100 D:100 |
| 4 | Certain | Thread a plain `skipBrewUpdate bool` param through `Update()` | Confirmed from intake #4 — contract wording plus the existing `Update(currentVersion string)` signature in update.go L16 make this unambiguous | S:95 R:85 A:95 D:95 |
| 5 | Certain | Wire flag via `cmd.Flags().BoolVar` following the `syncCmd()` pattern; `updateCmd()` is the sole `Update()` caller | Upgraded from intake #5 (Confident → Certain) — verified `syncCmd()` (main.go L82–97) uses this exact idiom and grep confirms `updateCmd()` is the only `internal.Update(` caller | S:95 R:85 A:95 D:90 |
| 6 | Confident | No memory/spec updates required — additive opt-in flag, behavior-preserving | Confirmed from intake #6 — `update` internals are not documented as spec-level behavior in docs/memory/; default preserves current behavior; affected memory recorded as `(none)` | S:85 R:80 A:90 D:85 |
| 7 | Confident | Test uses a fake `brew` on `PATH` logging subcommands; asserts update-omitted-but-upgrade-present (skip) and all-three-present (default) | Upgraded from intake #7 — verified repo convention in download_test.go and sync_test.go (real exec + fixtures via `t.TempDir`/`t.Setenv`); honors "do NOT refactor exec style" | S:85 R:80 A:85 D:80 |
| 8 | Tentative | Lowest-touch test seam for `isBrewInstalled()` (helper extraction vs. overridable func var) chosen at apply | Carried from intake #8 — `go test` binary path lacks `/Cellar/`, so the guard short-circuits; both seams preserve exec convention and runtime behavior, easily reversed; deferred to apply by design (not blocking) | S:60 R:70 A:65 D:50 |

8 assumptions (5 certain, 2 confident, 1 tentative, 0 unresolved).
