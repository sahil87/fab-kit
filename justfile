scripts := "src/scripts/just"
rust_src := "src/fab-rust"
go_src := "src/fab-go"

# Run all tests with summary
test:
    just test-setup
    just test-hooks
    just test-packages
    just test-scripts
    just test-rust
    just test-go

# Setup test dependencies
test-setup:
    {{scripts}}/test-setup.sh

# Run hook tests (bats)
test-hooks:
    {{scripts}}/test-hooks.sh

# Run package tests (bats)
test-packages:
    {{scripts}}/test-packages.sh

# Run script tests (bats)
test-scripts:
    {{scripts}}/test-scripts.sh

# Run Go unit tests
test-go:
    cd {{go_src}} && go test ./... -count=1

# Run Go unit tests (verbose)
test-go-v:
    cd {{go_src}} && go test ./... -v -count=1

# Run Rust integration tests
test-rust:
    cargo test --manifest-path src/fab-rust/Cargo.toml

# Build Rust binary for the current platform (local dev)
build-rust:
    cargo build --manifest-path {{rust_src}}/Cargo.toml --release
    cp {{rust_src}}/target/release/fab fab/.kit/bin/fab-rust

# Map os/arch pair to Rust target triple
_rust-target os arch:
    #!/usr/bin/env sh
    case "{{os}}-{{arch}}" in
      darwin-arm64) echo "aarch64-apple-darwin" ;;
      darwin-amd64) echo "x86_64-apple-darwin" ;;
      linux-arm64)  echo "aarch64-unknown-linux-musl" ;;
      linux-amd64)  echo "x86_64-unknown-linux-musl" ;;
      *) echo "ERROR: unknown platform {{os}}-{{arch}}" >&2; exit 1 ;;
    esac

# Cross-compile Rust binary for a specific target triple
build-rust-target target:
    cargo zigbuild --manifest-path {{rust_src}}/Cargo.toml --release --target {{target}}
    mkdir -p .release-build
    cp {{rust_src}}/target/{{target}}/release/fab .release-build/fab-rust-{{target}}

# Cross-compile Rust binary for all release targets
build-rust-all:
    just build-rust-target aarch64-apple-darwin
    just build-rust-target x86_64-apple-darwin
    just build-rust-target aarch64-unknown-linux-musl
    just build-rust-target x86_64-unknown-linux-musl

# Build everything for release (Go + Rust + wt)
build-all:
    mkdir -p .release-build
    just build-go-all
    just build-wt-all
    just build-rust-all

# Build fab Go binary for current platform
build-go:
    cd {{go_src}} && CGO_ENABLED=0 go build -o ../../fab/.kit/bin/fab-go ./cmd/fab

# Cross-compile Go binary for a specific target
build-go-target os arch:
    mkdir -p .release-build && cd {{go_src}} && CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}} go build -o ../../.release-build/fab-{{os}}-{{arch}} ./cmd/fab

# Cross-compile Go binary for all release targets
build-go-all:
    just build-go-target darwin arm64
    just build-go-target darwin amd64
    just build-go-target linux arm64
    just build-go-target linux amd64

# Build wt Go binary for current platform
build-wt:
    cd {{go_src}} && CGO_ENABLED=0 go build -o ../../fab/.kit/bin/wt ./cmd/wt

# Cross-compile wt binary for a specific target
build-wt-target os arch:
    mkdir -p .release-build && cd {{go_src}} && CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}} go build -o ../../.release-build/wt-{{os}}-{{arch}} ./cmd/wt

# Cross-compile wt binary for all release targets
build-wt-all:
    just build-wt-target darwin arm64
    just build-wt-target darwin amd64
    just build-wt-target linux arm64
    just build-wt-target linux amd64

# Package kit archives for release (generic + per-platform with Go binary)
package-kit:
    #!/usr/bin/env bash
    set -euo pipefail
    platforms=("darwin/arm64" "darwin/amd64" "linux/arm64" "linux/amd64")
    build_dir=".release-build"
    # Verify cross-compiled Go and wt binaries exist
    for platform in "${platforms[@]}"; do
      os="${platform%%/*}"
      arch="${platform##*/}"
      binary="$build_dir/fab-${os}-${arch}"
      if [ ! -f "$binary" ]; then
        echo "ERROR: Missing Go binary $binary — run 'just build-go-all' first."
        exit 1
      fi
      wt_binary="$build_dir/wt-${os}-${arch}"
      if [ ! -f "$wt_binary" ]; then
        echo "ERROR: Missing wt binary $wt_binary — run 'just build-wt-all' first."
        exit 1
      fi
    done
    # Generic archive (no binary)
    echo "Packaging kit.tar.gz (generic, no binary)..."
    COPYFILE_DISABLE=1 tar czf kit.tar.gz -C fab --exclude='.kit/bin/fab-go' --exclude='.kit/bin/fab-rust' --exclude='.kit/bin/wt' .kit
    echo "  kit.tar.gz ($(wc -c < kit.tar.gz) bytes)"
    # Per-platform archives (kit + Go binary + wt binary)
    for platform in "${platforms[@]}"; do
      os="${platform%%/*}"
      arch="${platform##*/}"
      archive_name="kit-${os}-${arch}.tar.gz"
      go_binary="$build_dir/fab-${os}-${arch}"
      wt_binary="$build_dir/wt-${os}-${arch}"
      staging="$build_dir/staging-${os}-${arch}"
      rm -rf "$staging"
      mkdir -p "$staging"
      cp -a fab/.kit "$staging/.kit"
      mkdir -p "$staging/.kit/bin"
      cp "$go_binary" "$staging/.kit/bin/fab-go"
      chmod +x "$staging/.kit/bin/fab-go"
      cp "$wt_binary" "$staging/.kit/bin/wt"
      chmod +x "$staging/.kit/bin/wt"
      COPYFILE_DISABLE=1 tar czf "$archive_name" -C "$staging" .kit
      echo "  $archive_name ($(wc -c < "$archive_name") bytes)"
      rm -rf "$staging"
    done
    echo "Packaging complete: kit.tar.gz + ${#platforms[@]} platform archives"

# Remove build artifacts and kit archives
clean:
    rm -rf .release-build
    rm -f kit.tar.gz kit-*.tar.gz

# Check prerequisites and environment health
doctor:
    fab/.kit/scripts/fab-doctor.sh
    @echo ""
    {{scripts}}/dev-doctor.sh
