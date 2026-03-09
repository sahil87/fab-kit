# Tasks: CI Releases with Justfile

**Change**: 260307-ma7o-1-ci-releases-justfile
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Verify existing `build-go` recipe in `justfile` at repo root — it already compiles Go binary for current platform to `fab/.kit/bin/fab-go` using `CGO_ENABLED=0`. No changes needed unless recipe signature diverges from spec <!-- clarified: justfile and build-go recipe already exist; confirmed via repo scan -->
- [x] T002 [P] Create `.github/workflows/` directory and `release.yml` skeleton with tag trigger (`on: push: tags: ['v*']`), `ubuntu-latest` runner, and `permissions: contents: write`

## Phase 2: Core Implementation

- [x] T003 Add `build-go-target os arch` recipe to `justfile` that cross-compiles to `.release-build/fab-{os}-{arch}` using `CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}}`
- [x] T004 Add `build-go-all` recipe to `justfile` that invokes `build-go-target` for all 4 platforms (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64)
- [x] T005 Add `package-kit` recipe to `justfile` — creates `kit.tar.gz` (generic, no binary) and 4 `kit-{os}-{arch}.tar.gz` (with binary at `.kit/bin/fab-go`). Extract packaging logic from `src/scripts/fab-release.sh` lines 182-206
- [x] T006 Add `clean` recipe to `justfile` — removes `.release-build/` and `kit*.tar.gz` from repo root
- [x] T007 Complete `.github/workflows/release.yml` — add steps: checkout, `actions/setup-go`, install `just`, run `just build-go-all`, run `just package-kit`, `gh release create` with `--generate-notes` and all 5 archives

## Phase 3: Integration & Edge Cases

- [x] T008 Refactor `src/scripts/fab-release.sh` — remove Go toolchain check, cross-compilation loop, archive packaging, `gh release create`, `.release-build/` cleanup, `--no-latest` flag. Keep: version bump, migration validation, git commit+tag+push, argument parsing for bump type, clean working tree check
- [x] T009 Add `.release-build/` and `kit-*.tar.gz` to `.gitignore` — `kit.tar.gz` is already present (line 18); only `.release-build/` and the platform-specific `kit-*.tar.gz` pattern need adding <!-- clarified: confirmed .gitignore already has kit.tar.gz but lacks .release-build/ and kit-*.tar.gz -->

---

## Execution Order

- T001 and T002 are independent (parallel)
- T003 depends on T001 (extends justfile)
- T004 depends on T003
- T005 depends on T004 (needs build artifacts to exist)
- T006 is independent of T003-T005, can run after T001
- T007 depends on T002 (extends workflow) and T004-T005 (references just recipes)
- T008 is independent of T001-T007 (modifies different file)
- T009 is independent, can run anytime
