# Spec: Remove Rust Implementation

**Change**: 260312-96nf-remove-rust-implementation
**Created**: 2026-03-12
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Removing the backend override mechanism (`FAB_BACKEND`, `.fab-backend`) entirely — it still works for Go-only use and is low-cost to keep
- Removing the `wt` or `idea` binaries — only the Rust `fab` backend is being removed

## Source Code: Rust Removal

### Requirement: Delete Rust source directory

The `src/rust/` directory tree SHALL be deleted entirely. This includes `src/rust/fab/Cargo.toml`, `src/rust/fab/Cargo.lock`, all Rust source code under `src/rust/fab/src/`, and all Rust tests under `src/rust/fab/tests/`.

#### Scenario: Rust source no longer exists

- **GIVEN** the change has been applied
- **WHEN** an agent or developer lists the `src/` directory
- **THEN** no `rust/` subdirectory exists
- **AND** no `Cargo.toml` or `Cargo.lock` files exist under `src/rust/` (including `src/rust/fab/`)

### Requirement: Delete Rust helper script

The `src/scripts/just/rust-target.sh` helper script SHALL be deleted. It maps Go-style `os/arch` pairs to Rust target triples and is only used by the `_rust-target` justfile recipe.

#### Scenario: Helper script removed

- **GIVEN** the change has been applied
- **WHEN** the `src/scripts/just/` directory is listed
- **THEN** `rust-target.sh` does not exist

## Build System: Justfile Cleanup

### Requirement: Remove Rust recipes from justfile

The `justfile` SHALL have all Rust-related content removed:
- The `rust_src` variable declaration
- The `test-rust` recipe
- The `build-rust` recipe
- The `_rust-target` recipe
- The `build-rust-target` recipe
- The `build-rust-all` recipe
- The comment referencing Rust in `build-all`

The `build-all` recipe SHALL call only `build-go-all` with no Rust references.

#### Scenario: Justfile contains no Rust references

- **GIVEN** the change has been applied
- **WHEN** the justfile is searched for "rust" (case-insensitive)
- **THEN** zero matches are found

#### Scenario: Build-all runs Go only

- **GIVEN** the change has been applied
- **WHEN** `just build-all` is invoked
- **THEN** only Go cross-compilation runs (via `build-go-all`)
- **AND** no Rust compilation is attempted

## Dispatcher: Go-Only Simplification

### Requirement: Remove fab-rust from dispatcher

The `fab/.kit/bin/fab` shell dispatcher SHALL be simplified to remove all `fab-rust` references:
- The `--version` handler SHALL check only for `fab-go` (remove `fab-rust` elif branch)
- The backend override block SHALL remove the `rust` case (keep the `go` case)
- The default priority block SHALL check only for `fab-go` (remove `fab-rust` fallback)
- The error message SHALL reference only `fab-go`

The backend override mechanism (`FAB_BACKEND` env var, `.fab-backend` file) MAY remain for the `go` case — this is low-cost and the infrastructure is already there.

#### Scenario: Dispatcher with fab-go present

- **GIVEN** `fab-go` is executable in `fab/.kit/bin/`
- **WHEN** `fab resolve` is invoked
- **THEN** `fab-go` is executed

#### Scenario: Dispatcher with no backend

- **GIVEN** no `fab-go` exists in `fab/.kit/bin/`
- **WHEN** `fab resolve` is invoked
- **THEN** exit code 1 with error: "no fab backend found (expected fab-go in ...)"

#### Scenario: Version output with Go backend

- **GIVEN** `fab-go` is executable
- **WHEN** `fab --version` is invoked
- **THEN** output is "fab {version} (go backend)"
- **AND** no "rust" appears in the output

#### Scenario: Backend override to go

- **GIVEN** `FAB_BACKEND=go` is set and `fab-go` is executable
- **WHEN** `fab resolve` is invoked
- **THEN** `fab-go` is executed

#### Scenario: Backend override to rust (removed)

- **GIVEN** `FAB_BACKEND=rust` is set
- **WHEN** `fab resolve` is invoked
- **THEN** the override is ignored (fall through to default `fab-go`)

## Skills: _scripts.md Update

### Requirement: Update CLI reference to Go-only

`fab/.kit/skills/_scripts.md` SHALL be updated:
- The "Calling Convention" section SHALL describe the dispatcher as checking only for `fab-go`
- The "Backend Priority" subsection SHALL be removed or rewritten to state Go-only
- Any mention of `fab-rust` SHALL be removed

#### Scenario: _scripts.md contains no Rust references

- **GIVEN** the change has been applied
- **WHEN** `_scripts.md` is searched for "rust" (case-insensitive)
- **THEN** zero matches are found

## Packaging: Clean Up fab-rust Exclusion

### Requirement: Remove fab-rust exclusion from package-kit.sh

`src/scripts/just/package-kit.sh` SHALL remove the `--exclude='.kit/bin/fab-rust'` flag from the generic archive tar command. The file no longer exists, so the exclusion is dead code.

#### Scenario: No fab-rust exclusion in packaging

- **GIVEN** the change has been applied
- **WHEN** `package-kit.sh` is searched for "fab-rust"
- **THEN** zero matches are found

## Documentation: Memory and Spec Updates

### Requirement: Update kit-architecture memory

`docs/memory/fab-workflow/kit-architecture.md` SHALL be updated:
- Remove `fab-rust` from the directory tree listing in `bin/`
- Remove or update the "Rust Binary (`fab-rust`)" subsection (if present)
- Update the dispatcher description to reference only `fab-go`
- Remove benchmark data comparing Rust and Go (if present)
- Update the overview to describe Go-only backend

#### Scenario: kit-architecture contains no Rust references

- **GIVEN** the change has been applied
- **WHEN** `kit-architecture.md` is searched for "rust" (case-insensitive)
- **THEN** zero matches are found (except in the changelog, which records the removal)

### Requirement: Update distribution memory

`docs/memory/fab-workflow/distribution.md` SHALL be updated:
- Remove `fab-rust` from release archive descriptions
- Remove Rust build recipes from justfile documentation
- Remove Rust cross-compilation from CI workflow description
- Remove `cargo-zigbuild`, Zig, and Rust toolchain references from CI steps
- Update the backend override mechanism to document Go-only
- Remove the "Transition Period: Dual Backends" section
- Update bootstrap one-liners to remove Rust dispatcher preference mention
- Update the `build-all` and `package-kit` descriptions

#### Scenario: distribution.md contains no Rust references

- **GIVEN** the change has been applied
- **WHEN** `distribution.md` is searched for "rust" (case-insensitive)
- **THEN** zero matches are found (except in the changelog, which records the removal)

### Requirement: Update packages spec

`docs/specs/packages.md` SHALL remove `fab-rust` from the `fab/.kit/bin/` directory tree listing.

#### Scenario: packages.md directory tree has no fab-rust

- **GIVEN** the change has been applied
- **WHEN** `packages.md` is searched for "fab-rust"
- **THEN** zero matches are found

## Deprecated Requirements

### Rust Binary (`fab-rust`)

**Reason**: The Rust backend was a parallel implementation that is no longer maintained. Only Go is actively developed.
**Migration**: N/A — `fab-go` provides all functionality.

### Rust Cross-Compilation in CI

**Reason**: CI no longer needs Rust toolchain, Zig, or cargo-zigbuild for cross-compilation.
**Migration**: N/A — CI already builds Go-only (the Rust steps were already removed from release.yml).

### Rust Build Recipes in Justfile

**Reason**: `test-rust`, `build-rust`, `_rust-target`, `build-rust-target`, `build-rust-all` recipes serve no purpose without the Rust source.
**Migration**: N/A — Go recipes remain.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete entire `src/rust/` directory | User explicitly requested removal; confirmed from intake #1 | S:95 R:60 A:95 D:95 |
| 2 | Certain | Remove all Rust recipes from justfile | User explicitly requested; confirmed from intake #2 | S:90 R:85 A:90 D:95 |
| 3 | Certain | Simplify dispatcher to Go-only | Direct consequence of removing Rust — confirmed from intake #3 | S:85 R:80 A:90 D:90 |
| 4 | Confident | Keep backend override mechanism for Go-only use | The `FAB_BACKEND` / `.fab-backend` mechanism still works for Go; removing it entirely is out of scope. Confirmed from intake #4 | S:60 R:85 A:70 D:70 |
| 5 | Certain | Update `_scripts.md` and memory docs | Constitution requires `_scripts.md` updates for CLI changes; memory should reflect current state. Confirmed from intake #5 | S:70 R:90 A:85 D:85 |
| 6 | Certain | No CI workflow changes needed | `release.yml` already builds Go-only; no Rust steps present. Confirmed from intake #6 | S:95 R:95 A:95 D:95 |
| 7 | Confident | Remove `package-kit.sh` fab-rust exclusion | Dead exclusion — cleaning it keeps the script accurate. Confirmed from intake #7 | S:65 R:95 A:80 D:85 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
