# Quality Checklist: CI Releases with Justfile

**Change**: 260307-ma7o-1-ci-releases-justfile
**Generated**: 2026-03-09
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Local Go Build: `just build-go` compiles Go binary to `fab/.kit/bin/fab-go` with `CGO_ENABLED=0`
- [x] CHK-002 Cross-Compile for Target: `just build-go-target {os} {arch}` produces binary at `.release-build/fab-{os}-{arch}`
- [x] CHK-003 Cross-Compile All Targets: `just build-go-all` produces 4 binaries (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64)
- [x] CHK-004 Package Kit Archives: `just package-kit` creates 5 archives — `kit.tar.gz` (no binary) + 4 per-platform archives (with binary at `.kit/bin/fab-go`)
- [x] CHK-005 Clean Build Artifacts: `just clean` removes `.release-build/` and `kit*.tar.gz`
- [x] CHK-006 Tag-Triggered Workflow: `release.yml` triggers on `v*` tag push only
- [x] CHK-007 Workflow Steps: checkout → setup-go → install just → build-go-all → package-kit → gh release create
- [x] CHK-008 Workflow Permissions: `permissions: contents: write` set for release creation
- [x] CHK-009 Simplified fab-release.sh: retains version bump, migration validation, git commit+tag+push
- [x] CHK-010 fab-release.sh pre-flight: checks clean working tree and VERSION file only (no gh/go checks)

## Behavioral Correctness
- [x] CHK-011 fab-release.sh no longer runs cross-compilation, packaging, or `gh release create`
- [x] CHK-012 fab-release.sh `--no-latest` flag removed (no longer parsed or referenced)
- [x] CHK-013 Archives produced by `package-kit` match existing structure (rooted at `.kit/`, same contents)
- [x] CHK-014 Generic `kit.tar.gz` excludes `.kit/bin/fab-go`; per-platform archives include it

## Removal Verification
- [x] CHK-015 Go toolchain check (`command -v go`) removed from fab-release.sh
- [x] CHK-016 `gh` CLI check (`command -v gh`) removed from fab-release.sh
- [x] CHK-017 Cross-compilation loop removed from fab-release.sh
- [x] CHK-018 Archive packaging logic removed from fab-release.sh
- [x] CHK-019 `gh release create` call removed from fab-release.sh
- [x] CHK-020 `.release-build/` cleanup removed from fab-release.sh
- [x] CHK-021 `--no-latest` flag and `$no_latest` variable removed from fab-release.sh

## Scenario Coverage
- [x] CHK-022 `just build-go-target darwin arm64` produces correct binary
- [x] CHK-023 `just package-kit` without prior build fails with error
- [x] CHK-024 `fab-release.sh patch` with clean tree bumps version, commits, tags, pushes
- [x] CHK-025 `fab-release.sh` with no arguments displays usage and exits 0
- [x] CHK-026 `fab-release.sh` with dirty working tree exits with error

## Edge Cases & Error Handling
- [x] CHK-027 `package-kit` fails gracefully when `.release-build/` is missing
- [x] CHK-028 `fab-release.sh` exits cleanly when VERSION file is missing

## Code Quality
- [x] CHK-029 Pattern consistency: justfile recipes follow `just` conventions (comments, variable naming)
- [x] CHK-030 No unnecessary duplication: packaging logic appears only in justfile, not in fab-release.sh
- [x] CHK-031 Readability: fab-release.sh is significantly shorter and each section is clearly scoped

## Documentation Accuracy
- [x] CHK-032 `release.yml` includes descriptive comments for each step
- [x] CHK-033 `justfile` recipes have descriptive comment headers

## Cross References
- [x] CHK-034 `.gitignore` includes `.release-build/` and `kit-*.tar.gz` patterns

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
