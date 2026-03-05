# Quality Checklist: Clean Break — Go Only

**Change**: 260305-u8t9-clean-break-go-only
**Generated**: 2026-03-05
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Shell Fallback Removal: Dispatcher has no shell fallback case block, exits 1 with error when no backend
- [ ] CHK-002 Version Handler: `--version` reports `none` (not `shell`) when no compiled backend present
- [ ] CHK-003 LIB_DIR Removal: Dispatcher script contains no reference to `LIB_DIR`
- [ ] CHK-004 Ported Script Deletion: All 7 ported shell scripts removed from `fab/.kit/scripts/lib/`
- [ ] CHK-005 Retained Scripts: `env-packages.sh` and `frontmatter.sh` remain in `fab/.kit/scripts/lib/`
- [ ] CHK-006 PATH Setup: `env-packages.sh` adds `$KIT_DIR/bin` to PATH before packages loop
- [ ] CHK-007 wt-status Deletion: `fab/.kit/packages/wt/bin/wt-status` removed
- [ ] CHK-008 wt-status Test Deletion: `src/packages/wt/tests/wt-status.bats` removed
- [ ] CHK-009 fab status show: New subcommand exists with `--all`, `--json`, `[<name>]` support
- [ ] CHK-010 dispatch.sh Update: `validate_prerequisites` uses `fab score --check-gate` instead of direct `calc-score.sh`
- [ ] CHK-011 _scripts.md Update: No shell fallback references, backend priority shows `rust > go > error`

## Behavioral Correctness
- [ ] CHK-012 Dispatcher no-backend: Prints error message and exits 1 when no fab-rust/fab-go present
- [ ] CHK-013 fab status show default: Shows current worktree status as single human-readable line
- [ ] CHK-014 fab status show --all: Shows all worktrees with header, formatted table, total count
- [ ] CHK-015 fab status show --json: Outputs valid JSON with correct fields
- [ ] CHK-016 fab status show --all --json: Outputs valid JSON array of worktree objects

## Removal Verification
- [ ] CHK-017 No dead code: No remaining references to deleted shell scripts in dispatcher or other active code
- [ ] CHK-018 No stale imports: No remaining `source` or `bash` calls to deleted scripts outside of test code

## Scenario Coverage
- [ ] CHK-019 Dispatcher with Go backend: `fab-go` present → execs successfully
- [ ] CHK-020 Worktree with no fab dir: Shows `(no fab)` placeholder
- [ ] CHK-021 Worktree with no active change: Shows `(no change)` placeholder
- [ ] CHK-022 Stale fab/current: Shows `(stale)` placeholder
- [ ] CHK-023 Named worktree not found: Exits non-zero with error message

## Edge Cases & Error Handling
- [ ] CHK-024 Parity tests graceful skip: `runBash` skips when bash script not found instead of failing

## Code Quality
- [ ] CHK-025 Pattern consistency: New Go code follows naming and structural patterns of existing `cmd/fab/` and `internal/` packages
- [ ] CHK-026 No unnecessary duplication: Reuses existing `internal/resolve` and `internal/statusfile` packages

## Documentation Accuracy
- [ ] CHK-027 _scripts.md accuracy: All command examples and backend priority description match actual behavior

## Cross References
- [ ] CHK-028 Affected memory files identified: `kit-architecture.md`, `distribution.md`, `execution-skills.md` listed for hydrate update

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
