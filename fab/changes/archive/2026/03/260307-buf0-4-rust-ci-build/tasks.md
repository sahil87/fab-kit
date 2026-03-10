# Tasks: Rust CI Build

**Change**: 260307-buf0-4-rust-ci-build
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add `_rust-target` mapping recipe and `build-rust-target` cross-compilation recipe to `justfile`
- [x] T002 Add `build-rust-all` and `build-all` recipes to `justfile`

## Phase 2: Core Implementation

- [x] T003 Update `package-kit` recipe in `justfile` to verify and include both Go and Rust binaries in per-platform archives, and exclude both from generic archive
- [x] T004 Update `.github/workflows/release.yml` to install Rust toolchain, Zig, cargo-zigbuild, and run `just build-all` instead of `just build-go-all`

## Phase 3: Integration & Edge Cases

- [x] T005 Verify `just build-rust-all` cross-compiles all 4 targets locally, and `just package-kit` produces correct dual-binary archives (partial — verified syntax, recipe listing, target mapping; full cross-compile requires cargo-zigbuild + zig)

---

## Execution Order

- T001 → T002 (build-rust-all depends on build-rust-target)
- T002 → T003 (package-kit depends on build-all existing)
- T004 is independent of T001-T003 (CI workflow can be updated in parallel)
- T005 depends on T001-T003 (verification requires all justfile recipes)
