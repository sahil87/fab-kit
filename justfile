scripts := "scripts/just"
fab_version := `cat src/kit/VERSION`
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

# Build all binaries for native platform → dist/bin/
build:
    #!/usr/bin/env bash
    case "$(uname -s)" in Linux) goos=linux;; Darwin) goos=darwin;; *) echo "unsupported OS"; exit 1;; esac
    case "$(uname -m)" in x86_64) goarch=amd64;; arm64|aarch64) goarch=arm64;; *) echo "unsupported arch"; exit 1;; esac
    suffix="$goos-$goarch"
    just build-target "$goos" "$goarch"
    for bin in fab-go idea wt fab-kit fab; do
        [ -f "dist/bin/${bin}-${suffix}" ] && mv "dist/bin/${bin}-${suffix}" "dist/bin/${bin}"
    done
    echo "Built all binaries → dist/bin/"

# Build + populate ~/.fab-kit/local-versions/{VERSION}/ (shim auto-discovers)
install: build
    mkdir -p {{local_cache}}/kit
    cp dist/bin/fab-go {{local_cache}}/fab-go
    rsync -a --delete src/kit/ {{local_cache}}/kit/
    @echo "Installed fab-go + kit → {{local_cache}}/"

# Check prerequisites and environment health
doctor:
    fab doctor
    @echo ""
    {{scripts}}/dev-doctor.sh

# -- Release -----------------------------------------------------------------------

# Bump version, commit, tag, and push (CI handles the rest)
release bump="patch":
    scripts/release.sh {{bump}}

# Assemble dist/kit/ from src/kit/ (single copy, reused by packaging)
dist-kit:
    rm -rf dist/kit
    mkdir -p dist/kit/bin
    cp -a src/kit/. dist/kit/
    mkdir -p dist/kit/bin
    @echo "Assembled dist/kit/ from src/kit/"

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
