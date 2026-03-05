# Quality Checklist: Unified Fab Dispatcher

**Change**: 260305-qagd-unified-fab-dispatcher
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Single Entry Point: `fab/.kit/bin/fab` is a shell script (not a binary) that dispatches to backends
- [ ] CHK-002 Backend Priority: dispatcher checks `fab-rust` → `fab-go` → shell in that order
- [ ] CHK-003 Shell Routing Table: all 7 commands route to correct scripts (resolve→resolve.sh, status→statusman.sh, log→logman.sh, preflight→preflight.sh, change→changeman.sh, score→calc-score.sh, archive→archiveman.sh)
- [ ] CHK-004 Archive Arg Injection: `fab archive <args>` forwards as `archiveman.sh archive <args>`
- [ ] CHK-005 Version Handling: `fab --version` prints `fab {version} ({backend} backend)` without delegating
- [ ] CHK-006 Shell Fallback Diagnostic: `[fab] using shell backend` printed to stderr when no compiled backend found
- [ ] CHK-007 Go Binary Renamed: built to `fab/.kit/bin/fab-go`, not `fab/.kit/bin/fab`
- [ ] CHK-008 Shim Removal: all 7 shell scripts have `_fab_bin` shim blocks removed
- [ ] CHK-009 Batch Scripts Updated: both batch scripts use `fab/.kit/bin/fab change resolve` instead of direct changeman.sh
- [ ] CHK-010 Parity Tests Updated: `fabBinary()` references `fab-go`
- [ ] CHK-011 _scripts.md Updated: documents dispatcher architecture, no legacy shim framing

## Behavioral Correctness
- [ ] CHK-012 CLI Signatures Unchanged: existing `fab <command> [subcommand] [args]` calls work identically
- [ ] CHK-013 Shell Scripts Standalone: each lib script works when invoked directly (shebang, set -euo pipefail, own path resolution preserved)

## Scenario Coverage
- [ ] CHK-014 Fallback Scenario: with no compiled binary, `fab preflight` routes to `preflight.sh` and works
- [ ] CHK-015 Go Backend Scenario: with `fab-go` present, `fab preflight` delegates to `fab-go`
- [ ] CHK-016 Unknown Command Scenario: `fab frobnicate` prints error to stderr and exits 1
- [ ] CHK-017 Version Scenario: `fab --version` outputs correct format with backend name

## Edge Cases & Error Handling
- [ ] CHK-018 No Diagnostic for --version: `fab --version` does not print fallback diagnostic even when using shell backend
- [ ] CHK-019 No Diagnostic for Compiled Backend: stderr clean when `fab-go` or `fab-rust` handles the command

## Code Quality
- [ ] CHK-020 Pattern consistency: dispatcher follows existing shell script patterns (shebang, set -euo pipefail)
- [ ] CHK-021 No unnecessary duplication: shim removal eliminates the 7x duplicated delegation pattern

## Documentation Accuracy
- [ ] CHK-022 _scripts.md: command mapping table matches actual dispatcher routing
- [ ] CHK-023 _scripts.md: calling convention section reflects dispatcher-first architecture

## Cross References
- [ ] CHK-024 .gitignore: `fab/.kit/bin/fab-go` and `fab/.kit/bin/fab-rust` are ignored; `fab/.kit/bin/fab` is not
- [ ] CHK-025 justfile: `build-go` target outputs to `fab-go`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
