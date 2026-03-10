# Quality Checklist: Rust Binary Port

**Change**: 260307-bmp3-3-rust-binary-port
**Generated**: 2026-03-10
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Crate Layout: `src/fab-rust/` exists as a binary crate with clap, serde, serde_yaml, anyhow deps and flat module structure
- [ ] CHK-002 Cargo.toml Configuration: release profile specifies `lto = true` and `strip = true`; `Cargo.lock` is committed
- [ ] CHK-003 Subcommand Parity: All 9 top-level subcommands implemented with identical CLI signatures (flags, positional args, sub-subcommands)
- [ ] CHK-004 Output Parity: Each subcommand produces identical stdout/stderr/exit codes as Go binary (modulo timestamp fields and help text)
- [ ] CHK-005 Change Resolution Parity: Same algorithm — exact match, case-insensitive substring, symlink fallback, single-change guess; identical error messages
- [ ] CHK-006 State Machine Parity: All transitions (start/advance/finish/reset/skip/fail), cascading, auto-activate, metrics, hooks, auto-logging match Go
- [ ] CHK-007 YAML Round-Trip Preservation: `.status.yaml` formatting preserved across read/write cycles
- [ ] CHK-008 JSONL Logging Parity: Same field names, field ordering, ISO 8601 timestamps, optional field omission
- [ ] CHK-009 Archive/Restore Parity: Nested YYYY/MM dirs, index.md format, backfill, pointer cleanup match Go
- [ ] CHK-010 Tmux Integration Parity: pane-map discovery and output, send-keys resolution and execution match Go
- [ ] CHK-011 Backend Override: FAB_BACKEND env var and .fab-backend file override work with correct priority and fallthrough
- [ ] CHK-012 Justfile Recipes: `build-rust` compiles and copies binary; `test-rust` runs Rust tests
- [ ] CHK-013 Gitignore: `.fab-backend` added to `.gitignore`
- [ ] CHK-014 Rust Integration Tests: Test suite at `src/fab-rust/tests/` with fixture-based approach covering all subcommands
- [ ] CHK-015 Go Tests Unmodified: No changes to `src/go/fab/` — existing parity tests still pass

## Behavioral Correctness

- [ ] CHK-016 Resolve output matches Go for all flag combinations (--id, --folder, --dir, --status)
- [ ] CHK-017 Status transitions produce same .status.yaml state as Go (finish with auto-activate, reset with cascading, skip with cascading)
- [ ] CHK-018 Score computation produces same confidence score, grade counts, and dimension means as Go
- [ ] CHK-019 Change new generates valid 4-char IDs with collision detection; switch updates symlink correctly
- [ ] CHK-020 Preflight YAML output is semantically identical to Go (after normalizing timestamps)

## Scenario Coverage

- [ ] CHK-021 Substring resolution: partial match resolves correctly
- [ ] CHK-022 Ambiguous resolution: multiple matches produces error with "Multiple changes match"
- [ ] CHK-023 Cascading reset: downstream stages become pending
- [ ] CHK-024 Stage hooks: pre-hook blocks start on failure, post-hook runs after finish
- [ ] CHK-025 Atomic write safety: temp+rename pattern used for all file writes
- [ ] CHK-026 Pane map outside tmux: exit code 1 with "not inside a tmux session"
- [ ] CHK-027 Backend env var override to Go: FAB_BACKEND=go selects Go binary
- [ ] CHK-028 Backend file override: .fab-backend file content selects backend
- [ ] CHK-029 Env var precedence: FAB_BACKEND takes priority over .fab-backend file
- [ ] CHK-030 Invalid/unavailable override: falls through to default priority

## Edge Cases & Error Handling

- [ ] CHK-031 Missing config.yaml: preflight exits non-zero with clear error
- [ ] CHK-032 Missing constitution.md: preflight exits non-zero with clear error
- [ ] CHK-033 No active change (no .fab-status.yaml): resolve/preflight report appropriate error
- [ ] CHK-034 Invalid state transition: rejected with error (e.g., fail on non-review stage)
- [ ] CHK-035 Archive of active change: .fab-status.yaml symlink removed after archive

## Code Quality

- [ ] CHK-036 Pattern consistency: Rust code follows idiomatic Rust patterns (Result/Option, no unwrap in non-test code, proper error propagation with anyhow)
- [ ] CHK-037 No unnecessary duplication: Shared resolution logic reused across subcommands; shared types used consistently
- [ ] CHK-038 Readability: Flat module structure maintained; no god functions
- [ ] CHK-039 No magic strings: Stage names, state names, and field names use constants

## Documentation Accuracy

- [ ] CHK-040 Affected memory files updated: `docs/memory/fab-workflow/distribution.md` and `docs/memory/fab-workflow/kit-architecture.md` reflect Rust binary

## Cross References

- [ ] CHK-041 `_scripts.md` consistency: Dispatcher and backend priority documentation matches implementation
- [ ] CHK-042 Spec alignment: All 15 spec requirements have corresponding implementation

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
