scripts := "src/scripts/just"

# Run all tests (bash + rust) with summary
test:
    just test-setup
    just _test-parallel
    # just test-rust  # uncomment when Rust libs exist

# Run test suites in parallel with prefixed live output
_test-parallel:
    {{scripts}}/test-parallel.sh

# Setup test dependencies
test-setup:
    {{scripts}}/test-setup.sh

# Run bash tests (bats)
test-bash:
    {{scripts}}/test-bash.sh

# Run package tests (bats)
test-packages:
    {{scripts}}/test-packages.sh

# Run script tests (bats)
test-scripts:
    {{scripts}}/test-scripts.sh

# Run Rust tests (placeholder)
test-rust:
    @echo "No Rust tests yet."

go_src := "src/fab-go"

# Build fab Go binary for current platform
build-go:
    cd {{go_src}} && CGO_ENABLED=0 go build -o ../../fab/.kit/bin/fab-go ./cmd/fab

# Cross-compile Go binary for a specific target
build-go-target os arch:
    mkdir -p ../../.release-build && cd {{go_src}} && CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}} go build -o ../../.release-build/fab-{{os}}-{{arch}} ./cmd/fab

# Cross-compile Go binary for all release targets
build-go-all:
    just build-go-target darwin arm64
    just build-go-target darwin amd64
    just build-go-target linux arm64
    just build-go-target linux amd64

# Package kit archives for release (generic + per-platform)
package-kit:
    #!/usr/bin/env bash
    set -euo pipefail
    platforms=("darwin/arm64" "darwin/amd64" "linux/arm64" "linux/amd64")
    build_dir=".release-build"
    # Verify cross-compiled binaries exist
    for platform in "${platforms[@]}"; do
      os="${platform%%/*}"
      arch="${platform##*/}"
      binary="$build_dir/fab-${os}-${arch}"
      if [ ! -f "$binary" ]; then
        echo "ERROR: Missing binary $binary — run 'just build-go-all' first."
        exit 1
      fi
    done
    # Generic archive (no binary)
    echo "Packaging kit.tar.gz (generic, no binary)..."
    COPYFILE_DISABLE=1 tar czf kit.tar.gz -C fab --exclude='.kit/bin/fab-go' .kit
    echo "  kit.tar.gz ($(wc -c < kit.tar.gz) bytes)"
    # Per-platform archives (kit + binary)
    for platform in "${platforms[@]}"; do
      os="${platform%%/*}"
      arch="${platform##*/}"
      archive_name="kit-${os}-${arch}.tar.gz"
      binary="$build_dir/fab-${os}-${arch}"
      staging="$build_dir/staging-${os}-${arch}"
      rm -rf "$staging"
      mkdir -p "$staging"
      cp -a fab/.kit "$staging/.kit"
      mkdir -p "$staging/.kit/bin"
      cp "$binary" "$staging/.kit/bin/fab-go"
      chmod +x "$staging/.kit/bin/fab-go"
      COPYFILE_DISABLE=1 tar czf "$archive_name" -C "$staging" .kit
      echo "  $archive_name ($(wc -c < "$archive_name") bytes)"
      rm -rf "$staging"
    done
    echo "Packaging complete: kit.tar.gz + ${#platforms[@]} platform archives"

# Run Go parity tests (bash vs Go binary output)
test-go:
    cd {{go_src}} && go test ./test/parity/... -count=1

# Run Go parity tests with verbose output
test-go-v:
    cd {{go_src}} && go test ./test/parity/... -v -count=1

# Remove build artifacts and kit archives
clean:
    rm -rf .release-build
    rm -f kit.tar.gz kit-*.tar.gz

# Check prerequisites and environment health
doctor:
    fab/.kit/scripts/fab-doctor.sh
    @echo ""
    {{scripts}}/dev-doctor.sh
