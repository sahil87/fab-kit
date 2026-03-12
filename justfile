scripts := "src/scripts/just"

# Run all tests with summary
test:
    just test-scripts
    just test-go

# Run script tests (bats)
test-scripts:
    {{scripts}}/test-scripts.sh

# Run Go unit tests (all modules)
test-go:
    cd src/go/fab && go test ./... -count=1
    cd src/go/idea && go test ./... -count=1
    cd src/go/wt && go test ./... -count=1

# Run Go unit tests (verbose)
test-go-v:
    cd src/go/fab && go test ./... -v -count=1
    cd src/go/idea && go test ./... -v -count=1
    cd src/go/wt && go test ./... -v -count=1

# Build all Go binaries for current platform (fab, wt, idea)
build-go:
    cd src/go/fab && CGO_ENABLED=0 go build -o ../../../fab/.kit/bin/fab-go ./cmd/fab
    cd src/go/idea && CGO_ENABLED=0 go build -o ../../../fab/.kit/bin/idea ./cmd
    cd src/go/wt && CGO_ENABLED=0 go build -o ../../../fab/.kit/bin/wt ./cmd

# Cross-compile a Go binary for a specific target
_build-go-binary src_dir cmd_path name os arch:
    mkdir -p .release-build && cd {{src_dir}} && CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}} go build -o ../../../.release-build/{{name}}-{{os}}-{{arch}} {{cmd_path}}

# Cross-compile all Go binaries for a specific target
build-go-target os arch:
    just _build-go-binary src/go/fab ./cmd/fab fab {{os}} {{arch}}
    just _build-go-binary src/go/idea ./cmd idea {{os}} {{arch}}
    just _build-go-binary src/go/wt ./cmd wt {{os}} {{arch}}

# Cross-compile all Go binaries for all release targets
build-go-all:
    just build-go-target darwin arm64
    just build-go-target darwin amd64
    just build-go-target linux arm64
    just build-go-target linux amd64

# Build everything for release
build-all:
    mkdir -p .release-build
    just build-go-all

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
