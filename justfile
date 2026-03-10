scripts := "src/scripts/just"
rust_src := "src/fab-rust"

# Run all tests with summary (excludes Rust)
test:
    just test-scripts
    just test-go

# Run script tests (bats)
test-scripts:
    {{scripts}}/test-scripts.sh

# Run Go unit tests (all modules)
test-go:
    cd src/fab-go && go test ./... -count=1
    cd src/idea-go && go test ./... -count=1
    cd src/wt-go && go test ./... -count=1

# Run Go unit tests (verbose)
test-go-v:
    cd src/fab-go && go test ./... -v -count=1
    cd src/idea-go && go test ./... -v -count=1
    cd src/wt-go && go test ./... -v -count=1

# Build all Go binaries for current platform (fab, wt, idea)
build-go:
    cd src/fab-go && CGO_ENABLED=0 go build -o ../../fab/.kit/bin/fab-go ./cmd/fab
    cd src/idea-go && CGO_ENABLED=0 go build -o ../../fab/.kit/bin/idea ./cmd
    cd src/wt-go && CGO_ENABLED=0 go build -o ../../fab/.kit/bin/wt ./cmd

# Cross-compile a Go binary for a specific target
_build-go-binary src_dir cmd_path name os arch:
    mkdir -p .release-build && cd {{src_dir}} && CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}} go build -o ../../.release-build/{{name}}-{{os}}-{{arch}} {{cmd_path}}

# Cross-compile all Go binaries for a specific target
build-go-target os arch:
    just _build-go-binary src/fab-go ./cmd/fab fab {{os}} {{arch}}
    just _build-go-binary src/idea-go ./cmd idea {{os}} {{arch}}
    just _build-go-binary src/wt-go ./cmd wt {{os}} {{arch}}

# Cross-compile all Go binaries for all release targets
build-go-all:
    just build-go-target darwin arm64
    just build-go-target darwin amd64
    just build-go-target linux arm64
    just build-go-target linux amd64

# Build everything for release (Go binaries only — use build-rust-all for Rust)
build-all:
    mkdir -p .release-build
    just build-go-all

# Run Rust integration tests
test-rust:
    cargo test --manifest-path src/fab-rust/Cargo.toml

# Build Rust binary for the current platform (local dev)
build-rust:
    cargo build --manifest-path {{rust_src}}/Cargo.toml --release
    cp {{rust_src}}/target/release/fab fab/.kit/bin/fab-rust

# Map os/arch pair to Rust target triple
_rust-target os arch:
    {{scripts}}/rust-target.sh {{os}} {{arch}}

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

# Package kit archives for release (generic + per-platform with Go binaries)
package-kit:
    {{scripts}}/package-kit.sh

# Remove build artifacts and kit archives
clean:
    rm -rf .release-build
    rm -f kit.tar.gz kit-*.tar.gz

# Check prerequisites and environment health
doctor:
    fab/.kit/scripts/fab-doctor.sh
    @echo ""
    {{scripts}}/dev-doctor.sh
