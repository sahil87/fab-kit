# Spec: Rust CI Build

**Change**: 260307-buf0-4-rust-ci-build
**Created**: 2026-03-10
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Dropping the Go binary from releases — that is a future change after Rust stability is confirmed
- Changing the dispatcher or backend override mechanism — already implemented in change 3
- macOS code signing or notarization — out of scope for CLI tools distributed via tar archives

## Distribution: Rust Cross-Compilation via cargo-zigbuild

### Requirement: Rust Cross-Compilation Recipes

The `justfile` SHALL provide Rust cross-compilation recipes using `cargo-zigbuild` that mirror the existing Go cross-compilation pattern. Linux targets SHALL use musl for fully static binaries.

#### Scenario: Cross-compile Rust for a single target
- **GIVEN** `cargo-zigbuild` and Zig are installed, and `src/fab-rust/Cargo.toml` exists
- **WHEN** `just build-rust-target aarch64-apple-darwin` is run
- **THEN** `cargo zigbuild` compiles for the specified target triple with release profile
- **AND** the binary is copied to `.release-build/fab-rust-{target}`

#### Scenario: Cross-compile Rust for all release targets
- **GIVEN** `cargo-zigbuild` and Zig are installed
- **WHEN** `just build-rust-all` is run
- **THEN** four binaries are produced in `.release-build/`: `fab-rust-aarch64-apple-darwin`, `fab-rust-x86_64-apple-darwin`, `fab-rust-aarch64-unknown-linux-musl`, `fab-rust-x86_64-unknown-linux-musl`

#### Scenario: Build everything for release
- **GIVEN** Go and Rust toolchains are available
- **WHEN** `just build-all` is run
- **THEN** both `just build-go-all` and `just build-rust-all` execute, producing 8 binaries total (4 Go + 4 Rust)

### Requirement: Target Triple Mapping

The `justfile` SHALL provide a helper recipe `_rust-target` that maps `{os}-{arch}` pairs (using the existing Go naming convention: `darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`) to Rust target triples.

#### Scenario: Map darwin-arm64 to Rust target
- **GIVEN** the mapping recipe exists
- **WHEN** `just _rust-target darwin arm64` is called
- **THEN** it outputs `aarch64-apple-darwin`

#### Scenario: Map linux-amd64 to Rust target
- **GIVEN** the mapping recipe exists
- **WHEN** `just _rust-target linux amd64` is called
- **THEN** it outputs `x86_64-unknown-linux-musl`

## Distribution: Dual-Binary Release Archives

### Requirement: Package Kit with Both Binaries

The `package-kit` recipe SHALL include both `fab-go` and `fab-rust` binaries in each per-platform archive. The recipe SHALL verify that both Go and Rust cross-compiled binaries exist before packaging.

#### Scenario: Package per-platform archive with dual binaries
- **GIVEN** `.release-build/` contains both `fab-{os}-{arch}` (Go) and `fab-rust-{target}` (Rust) for all platforms
- **WHEN** `just package-kit` is run
- **THEN** each `kit-{os}-{arch}.tar.gz` contains both `.kit/bin/fab-go` and `.kit/bin/fab-rust`
- **AND** the generic `kit.tar.gz` remains binary-free

#### Scenario: Missing Rust binaries blocks packaging
- **GIVEN** Go binaries exist in `.release-build/` but Rust binaries are missing
- **WHEN** `just package-kit` is run
- **THEN** the recipe fails with an error directing to run `just build-rust-all`

#### Scenario: Missing Go binaries blocks packaging
- **GIVEN** Rust binaries exist in `.release-build/` but Go binaries are missing
- **WHEN** `just package-kit` is run
- **THEN** the recipe fails with an error directing to run `just build-go-all`

### Requirement: Generic Archive Unchanged

The generic `kit.tar.gz` SHALL remain binary-free. It SHALL exclude both `fab-go` and `fab-rust` from `.kit/bin/`.

#### Scenario: Generic archive contains no binaries
- **GIVEN** platform binaries exist
- **WHEN** `just package-kit` is run
- **THEN** `kit.tar.gz` contains `.kit/` contents but no `fab-go` or `fab-rust` in `.kit/bin/`

## Distribution: CI Workflow for Rust

### Requirement: Rust Toolchain in CI

The release workflow SHALL install the Rust toolchain with cross-compilation targets (`aarch64-apple-darwin`, `x86_64-apple-darwin`, `aarch64-unknown-linux-musl`, `x86_64-unknown-linux-musl`) using `dtolnay/rust-toolchain@stable`.

#### Scenario: Rust toolchain setup in CI
- **GIVEN** the release workflow is triggered by a `v*` tag push
- **WHEN** the Rust toolchain step runs
- **THEN** `rustc` and `cargo` are available with all four cross-compilation targets installed

### Requirement: Zig and cargo-zigbuild in CI

The release workflow SHALL install Zig via `pip install ziglang` and `cargo-zigbuild` via `cargo install cargo-zigbuild`.

#### Scenario: Zig installed via pip
- **GIVEN** Python is available on the runner (default on `ubuntu-latest`)
- **WHEN** the install step runs `pip install ziglang`
- **THEN** the `zig` binary is available on PATH

#### Scenario: cargo-zigbuild installed
- **GIVEN** Rust toolchain is installed
- **WHEN** the install step runs `cargo install cargo-zigbuild`
- **THEN** the `cargo-zigbuild` binary is available on PATH

### Requirement: CI Builds Both Backends

The release workflow SHALL run `just build-all` (replacing the current `just build-go-all`) to cross-compile both Go and Rust binaries before packaging.

#### Scenario: Full CI release with dual binaries
- **GIVEN** a `v*` tag is pushed
- **WHEN** the release workflow runs
- **THEN** it installs Go, Rust, Zig, cargo-zigbuild, and just
- **AND** runs `just build-all` to produce 8 cross-compiled binaries
- **AND** runs `just package-kit` to create 5 archives (generic + 4 platform, each platform with both binaries)
- **AND** creates a GitHub Release with all 5 archives

### Requirement: CI Caching for Rust Tools

The release workflow SHOULD cache `cargo-zigbuild` and Zig installations to reduce CI build time. The `cargo install` step takes ~30s and `pip install ziglang` takes ~10s; caching avoids this on subsequent runs.

#### Scenario: Cached cargo-zigbuild
- **GIVEN** a previous CI run cached `~/.cargo/bin/cargo-zigbuild`
- **WHEN** a new release workflow runs
- **THEN** the cargo install step is skipped (cache hit)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use cargo-zigbuild for Rust cross-compilation | Confirmed from intake #1 — user explicitly chose Zig | S:95 R:80 A:90 D:95 |
| 2 | Certain | Single Linux runner for both Go and Rust | Confirmed from intake #2 — consistent with change 1 | S:90 R:80 A:90 D:90 |
| 3 | Certain | Ship both Go and Rust binaries in platform archives | Confirmed from intake #3 — dispatcher handles selection | S:90 R:85 A:85 D:90 |
| 4 | Certain | Same 4 platform targets for Rust | Confirmed from intake #4 — matches Go targets | S:95 R:85 A:95 D:95 |
| 5 | Certain | Use musl for Linux Rust targets (static binaries) | Confirmed from intake #5 — upgraded from Confident, standard Rust CLI distribution practice, no alternative considered | S:85 R:85 A:90 D:90 |
| 6 | Confident | Cache cargo-zigbuild and Zig in CI | Confirmed from intake #6 — standard CI optimization, easily adjusted | S:65 R:90 A:80 D:80 |
| 7 | Certain | Generic kit.tar.gz remains binary-free | Confirmed from intake #7 — upgraded from Confident, unchanged from current behavior, serves unsupported platforms | S:85 R:85 A:90 D:85 |
| 8 | Confident | Use `pip install ziglang` for Zig in CI | Python preinstalled on ubuntu-latest, simpler than manual Zig download. Alternatives: direct download, snap, or Zig GitHub Action | S:70 R:90 A:75 D:70 |
| 9 | Confident | Use Rust target triples as build artifact names (e.g. `fab-rust-aarch64-apple-darwin`) | Target triples are canonical identifiers, avoids ambiguity with os/arch mapping | S:70 R:90 A:80 D:75 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).
