# Quality Checklist: Rust CI Build

**Change**: 260307-buf0-4-rust-ci-build
**Generated**: 2026-03-10
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Rust cross-compilation recipes: `build-rust-target`, `build-rust-all`, `build-all` exist and are callable
- [ ] CHK-002 Target triple mapping: `_rust-target` correctly maps all 4 os/arch pairs to Rust target triples
- [ ] CHK-003 Dual-binary packaging: `package-kit` includes both `fab-go` and `fab-rust` in per-platform archives
- [ ] CHK-004 Generic archive: `kit.tar.gz` excludes both `fab-go` and `fab-rust`
- [ ] CHK-005 CI workflow: release.yml installs Rust toolchain, Zig, cargo-zigbuild, and runs `build-all`
- [ ] CHK-006 CI caching: cargo-zigbuild installation is cached between runs

## Behavioral Correctness

- [ ] CHK-007 Build recipe naming: `build-all` runs both `build-go-all` and `build-rust-all`
- [ ] CHK-008 Binary verification: `package-kit` fails if Go or Rust binaries are missing

## Scenario Coverage

- [ ] CHK-009 Single target cross-compile: `build-rust-target aarch64-apple-darwin` produces correct binary
- [ ] CHK-010 All targets: `build-rust-all` produces 4 Rust binaries in `.release-build/`
- [ ] CHK-011 Full CI flow: tag push triggers build-all → package-kit → release create

## Edge Cases & Error Handling

- [ ] CHK-012 Missing Rust binaries: `package-kit` produces clear error directing to `build-rust-all`
- [ ] CHK-013 Missing Go binaries: `package-kit` produces clear error directing to `build-go-all`

## Code Quality

- [ ] CHK-014 Pattern consistency: Rust recipes follow same structure as existing Go recipes
- [ ] CHK-015 No unnecessary duplication: Shared patterns (platform iteration, staging) reused

## Documentation Accuracy

- [ ] CHK-016 Memory file updated: `docs/memory/fab-workflow/distribution.md` reflects dual-binary archives, Rust CI, and cargo-zigbuild

## Cross References

- [ ] CHK-017 Justfile recipes are consistent with CI workflow steps
