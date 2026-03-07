# Intake: Rust CI Build

**Change**: 260307-buf0-4-rust-ci-build
**Created**: 2026-03-07
**Status**: Draft

## Origin

> Add `just build-rust-all` (cross-compilation via `cargo-zigbuild`) to the justfile and CI workflow. Releases ship both Go and Rust binaries during the transition period, with Rust as the preferred backend. This is step 4 of the 4-part plan.

Discussion context: User chose `cargo-zigbuild` (Zig as cross-compiler linker) over Docker-based `cross` or native runners. It works from a single Linux runner, produces fully static musl binaries for Linux, and handles macOS targets. The user confirmed "Zig" when asked about cross-compilation approach.

Depends on: `260307-bmp3-3-rust-binary-port` (Rust binary must exist before CI can build it).

## Why

Change 3 creates the Rust binary but only builds it locally. For releases to ship `fab-rust`, CI needs to cross-compile it for all 4 platform targets. Using `cargo-zigbuild` keeps the single-Linux-runner model from change 1 — no Docker, no macOS runners, no complex toolchain setup.

During the transition period, releases ship both binaries. The dispatcher prefers Rust. Users can fall back to Go via `.fab-backend` or `FAB_BACKEND=go`. After confidence is established, a future change drops the Go binary from releases.

## What Changes

### Modified: `justfile`

Add Rust cross-compilation recipes:

```just
# Rust target triples for each platform
rust_targets := "aarch64-apple-darwin x86_64-apple-darwin aarch64-unknown-linux-musl x86_64-unknown-linux-musl"

# Map platform names to Rust target triples
_rust-target os arch:
    #!/usr/bin/env sh
    case "{{os}}-{{arch}}" in
      darwin-arm64) echo "aarch64-apple-darwin" ;;
      darwin-amd64) echo "x86_64-apple-darwin" ;;
      linux-arm64)  echo "aarch64-unknown-linux-musl" ;;
      linux-amd64)  echo "x86_64-unknown-linux-musl" ;;
    esac

# Cross-compile Rust binary for a specific target
build-rust-target target:
    cargo zigbuild --manifest-path src/fab-rust/Cargo.toml --release --target {{target}}
    mkdir -p .release-build
    cp target/{{target}}/release/fab-rust .release-build/fab-rust-{{target}}

# Cross-compile Rust binary for all release targets
build-rust-all:
    just build-rust-target aarch64-apple-darwin
    just build-rust-target x86_64-apple-darwin
    just build-rust-target aarch64-unknown-linux-musl
    just build-rust-target x86_64-unknown-linux-musl

# Build everything for release (Go + Rust)
build-all:
    just build-go-all
    just build-rust-all
```

### Modified: `justfile` `package-kit` recipe

Update packaging to include both binaries in platform archives:

```just
# Package kit archives — now includes both fab-go and fab-rust per platform
package-kit:
    # For each platform:
    #   1. Create staging/.kit/ from fab/.kit/
    #   2. Copy fab-go binary to staging/.kit/bin/fab-go
    #   3. Copy fab-rust binary to staging/.kit/bin/fab-rust
    #   4. Tar into kit-{os}-{arch}.tar.gz
    # Generic kit.tar.gz remains binary-free
```

### Modified: `.github/workflows/release.yml`

Add Rust build step alongside Go:

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - uses: dtolnay/rust-toolchain@stable
        with:
          targets: aarch64-apple-darwin,x86_64-apple-darwin,aarch64-unknown-linux-musl,x86_64-unknown-linux-musl
      - name: Install tools
        run: |
          pip install ziglang
          cargo install cargo-zigbuild
          curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin
      - name: Build all
        run: just build-all
      - name: Package
        run: just package-kit
      - name: Release
        run: gh release create ${{ github.ref_name }} ...
```

### Modified: Release archive contents

Platform archives now contain two binaries:

```
kit-darwin-arm64.tar.gz
  .kit/
    bin/
      fab          # dispatcher (shell script)
      fab-go       # Go binary
      fab-rust     # Rust binary (preferred by dispatcher)
    ...
```

The generic `kit.tar.gz` remains binary-free (unchanged).

### CI toolchain requirements

The single Linux runner needs:
- Go toolchain (already in change 1)
- Rust toolchain with cross-compile targets (`dtolnay/rust-toolchain` action)
- Zig (`pip install ziglang`)
- `cargo-zigbuild` (`cargo install cargo-zigbuild` — can be cached)
- `just` (single binary install)

## Affected Memory

- `fab-workflow/distribution`: (modify) Document Rust binary in release archives, dual-binary transition period, cargo-zigbuild cross-compilation, and CI toolchain requirements (Rust + Zig additions)

## Impact

- **`justfile`**: Add `build-rust-target`, `build-rust-all`, `build-all` recipes; update `package-kit`
- **`.github/workflows/release.yml`**: Add Rust toolchain setup, Zig install, change `build-go-all` → `build-all`
- **Release archives**: Each platform archive grows by ~2-3MB (the Rust binary). Total archive size: ~8-11MB per platform (from ~6-8MB)
- **End users**: Transparent — dispatcher handles backend selection automatically
- **Go binary**: Still shipped during transition. Future change (not in this plan) removes it

## Open Questions

- When should the Go binary be dropped from releases? Probably after 1-2 release cycles with Rust shipping without issues. Not in scope for this change.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use cargo-zigbuild for Rust cross-compilation | Discussed — user explicitly chose Zig over Docker-based cross or native runners | S:95 R:80 A:90 D:95 |
| 2 | Certain | Single Linux runner for both Go and Rust cross-compilation | Discussed — consistent with change 1's approach | S:90 R:80 A:90 D:90 |
| 3 | Certain | Ship both Go and Rust binaries during transition | Discussed — dispatcher prefers Rust, users can fall back to Go | S:90 R:85 A:85 D:90 |
| 4 | Certain | Same 4 platform targets for Rust (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64) | Consistent with Go targets, user confirmed | S:95 R:85 A:95 D:95 |
| 5 | Confident | Use musl for Linux targets (fully static binaries) | Standard practice for Rust CLI distribution — no glibc version issues | S:75 R:85 A:85 D:80 |
| 6 | Confident | Cache cargo-zigbuild and Zig in CI for faster builds | Standard CI optimization — install step is slow (~30s) | S:65 R:90 A:80 D:80 |
| 7 | Confident | Generic kit.tar.gz remains binary-free | Unchanged from current behavior — serves as fallback for unsupported platforms | S:80 R:85 A:85 D:85 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
