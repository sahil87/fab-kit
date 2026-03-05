# Quality Checklist: Ship fab Go Binary

**Change**: 260305-g0uq-2-ship-fab-go-binary
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Parity test harness: Go test suite at `src/fab-go/test/parity/` runs both bash and Go for each operation
- [ ] CHK-002 Parity fixtures: Fixtures cover all 7 ported scripts with representative inputs
- [ ] CHK-003 Cross-compilation: `fab-release.sh` builds Go binary for 4 platforms (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64)
- [ ] CHK-004 Per-platform archives: Release produces 5 archives (4 platform + 1 generic)
- [ ] CHK-005 Platform detection: `fab-upgrade.sh` detects OS/arch via `uname` with normalization
- [ ] CHK-006 Platform-specific download: `fab-upgrade.sh` tries `kit-{os}-{arch}.tar.gz` first, falls back to `kit.tar.gz`
- [ ] CHK-007 Shim layer: All 7 shell scripts have shim blocks delegating to `fab/.kit/bin/fab`
- [ ] CHK-008 Skill caller update: `_scripts.md`, `_preamble.md`, and all skill files updated with `fab/.kit/bin/fab` invocations
- [ ] CHK-009 Bootstrap one-liner: README updated with platform-aware bootstrap command

## Behavioral Correctness

- [ ] CHK-010 Shim fallback: When binary is absent, shim falls through to bash implementation with identical output
- [ ] CHK-011 Shim delegation: When binary is present, shim `exec`s Go binary and bash code is not reached
- [ ] CHK-012 Upgrade fallback: When platform archive is not available, upgrade falls back to generic `kit.tar.gz`
- [ ] CHK-013 Release cleanup: No `kit*.tar.gz` or binary artifacts remain after release completes

## Scenario Coverage

- [ ] CHK-014 Parity tests pass for all operations (`go test ./test/parity/...`)
- [ ] CHK-015 Cross-compile scenario: 4 platform binaries built with `CGO_ENABLED=0`
- [ ] CHK-016 Archive structure: Platform archives contain `.kit/bin/fab`; generic archive does not
- [ ] CHK-017 Upgrade on supported platform: Downloads platform-specific archive, binary is executable
- [ ] CHK-018 Upgrade on older release: Falls back to generic archive when platform archive missing
- [ ] CHK-019 Missing Go toolchain: `fab-release.sh` exits with clear error message

## Edge Cases & Error Handling

- [ ] CHK-020 Wrong platform binary: `[ -x ]` check fails, shim falls through to bash
- [ ] CHK-021 Missing prerequisites: Parity tests skip (not fail) when `yq`/`jq` not installed
- [ ] CHK-022 Unsupported platform upgrade: Falls back to generic with informative message

## Code Quality

- [ ] CHK-023 Pattern consistency: Shim blocks follow identical pattern across all 7 scripts (only subcommand differs)
- [ ] CHK-024 No unnecessary duplication: Platform detection logic not duplicated between release and upgrade scripts
- [ ] CHK-025 Readability: Release script's new cross-compilation section uses clear variable names and comments
- [ ] CHK-026 No god functions: New release script sections are well-factored (build, package, upload as distinct blocks)
- [ ] CHK-027 No magic strings: Platform targets defined as loop variables, not hardcoded repeatedly

## Documentation Accuracy

- [ ] CHK-028 `_scripts.md` accurately documents both calling conventions with correct Go subcommand mapping
- [ ] CHK-029 README bootstrap one-liner works for all supported platforms

## Cross References

- [ ] CHK-030 All skill files consistently use `fab/.kit/bin/fab` (not a mix of old and new conventions)
- [ ] CHK-031 Skill sync: `fab-sync.sh` run to deploy updated skills to `.claude/skills/`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
