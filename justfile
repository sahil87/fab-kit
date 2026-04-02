scripts := "scripts/just"
fab_version := `cat fab/.kit/VERSION`
fab_ldflags := "-X main.version=" + fab_version
fab_kit_ldflags := "-X main.version=" + fab_version

# -- Development ------------------------------------------------------------------

# Run all tests
test:
    cd src/go/fab && go test ./... -count=1
    cd src/go/idea && go test ./... -count=1
    cd src/go/wt && go test ./... -count=1
    cd src/go/fab-kit && go test ./... -count=1

# Run all tests (verbose)
test-v:
    cd src/go/fab && go test ./... -v -count=1
    cd src/go/idea && go test ./... -v -count=1
    cd src/go/wt && go test ./... -v -count=1
    cd src/go/fab-kit && go test ./... -v -count=1

local_cache := env("HOME", "") / ".fab-kit/local-versions" / fab_version

# Build fab-go + copy .kit content to local cache (shim auto-discovers)
build:
    mkdir -p {{local_cache}}/kit
    cd src/go/fab && CGO_ENABLED=0 go build -ldflags '{{fab_ldflags}}' -o {{local_cache}}/fab-go ./cmd/fab
    rsync -a --delete --exclude='bin/' fab/.kit/ {{local_cache}}/kit/
    @echo "Built fab-go → {{local_cache}}/fab-go"

# Build fab-kit and fab router to dist/bin/ (install to PATH or test directly)
build-fab-kit:
    mkdir -p dist/bin
    cd src/go/fab-kit && CGO_ENABLED=0 go build -ldflags '{{fab_kit_ldflags}}' -o ../../../dist/bin/fab-kit ./cmd/fab-kit
    cd src/go/fab-kit && CGO_ENABLED=0 go build -ldflags '{{fab_kit_ldflags}}' -o ../../../dist/bin/fab ./cmd/fab
    @echo "Built fab-kit → dist/bin/fab-kit"
    @echo "Built fab (router) → dist/bin/fab"

# Check prerequisites and environment health
doctor:
    fab/.kit/scripts/fab-doctor.sh
    @echo ""
    {{scripts}}/dev-doctor.sh

# -- Release -----------------------------------------------------------------------

# Bump version, commit, tag, and push (CI handles the rest)
release bump="patch":
    scripts/release.sh {{bump}}

# Assemble dist/kit/ from fab/.kit/ (single copy, reused by packaging)
dist-kit:
    rm -rf dist/kit
    mkdir -p dist/kit/bin
    cp -a fab/.kit/. dist/kit/
    mkdir -p dist/kit/bin
    @echo "Assembled dist/kit/ from fab/.kit/"

# Cross-compile a binary for a specific target into dist/bin/
_build-binary src_dir cmd_path name os arch ldflags="":
    mkdir -p dist/bin && cd {{src_dir}} && CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}} go build -ldflags '{{ldflags}}' -o ../../../dist/bin/{{name}}-{{os}}-{{arch}} {{cmd_path}}

# Cross-compile all binaries for a specific target (5 binaries)
build-target os arch:
    just _build-binary src/go/fab ./cmd/fab fab-go {{os}} {{arch}} '{{fab_ldflags}}'
    just _build-binary src/go/idea ./cmd idea {{os}} {{arch}}
    just _build-binary src/go/wt ./cmd wt {{os}} {{arch}}
    just _build-binary src/go/fab-kit ./cmd/fab-kit fab-kit {{os}} {{arch}} '{{fab_kit_ldflags}}'
    just _build-binary src/go/fab-kit ./cmd/fab fab {{os}} {{arch}} '{{fab_kit_ldflags}}'

# Cross-compile all binaries for all release targets (5 binaries x 4 platforms = 20)
build-all:
    just build-target darwin arm64
    just build-target darwin amd64
    just build-target linux arm64
    just build-target linux amd64

# Package kit archives into dist/ (per-platform: kit content + fab-go)
package-kit:
    {{scripts}}/package-kit.sh

# Package brew archives into dist/ (per-platform: fab, fab-kit, wt, idea)
package-brew:
    {{scripts}}/package-brew.sh

# Generate release notes for the current tag into dist/release-notes.md
release-notes tag="":
    {{scripts}}/release-notes.sh {{tag}}

# Generate Homebrew formula from template into dist/fab-kit.rb
brew-formula tag="":
    {{scripts}}/brew-formula.sh {{tag}}

# Full release pipeline (everything CI runs, minus token-gated uploads)
dist: dist-kit build-all package-kit package-brew

# Remove all build artifacts
clean:
    rm -rf dist
