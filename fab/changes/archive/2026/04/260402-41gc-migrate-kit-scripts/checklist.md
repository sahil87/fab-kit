# Quality Checklist: Migrate Kit Scripts to Go Binary

**Change**: 260402-41gc-migrate-kit-scripts
**Generated**: 2026-04-02
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 `fab doctor`: Checks all 7 prerequisites (git, fab, bash, yq v4+, jq, gh, direnv+hook) with pass/fail and version info
- [ ] CHK-002 `fab doctor --porcelain`: Only prints error lines, no passes/hints/summary
- [ ] CHK-003 `fab doctor` exit code equals failure count
- [ ] CHK-004 `fab doctor` routes via `fabKitArgs` allowlist (works without config.yaml)
- [ ] CHK-005 `fab fab-help`: Scans `.kit/skills/*.md` frontmatter, groups by category, renders formatted output
- [ ] CHK-006 `fab fab-help`: Excludes `_*` partials and `internal-*` skills
- [ ] CHK-007 `fab fab-help`: Shows batch commands from cobra metadata in "Batch Operations" group
- [ ] CHK-008 `fab operator`: Creates singleton tmux tab named "operator" with spawn command
- [ ] CHK-009 `fab operator`: Switches to existing tab if "operator" already exists
- [ ] CHK-010 `fab batch new`: Parses backlog.md, creates worktrees, opens tmux tabs with `/fab-new`
- [ ] CHK-011 `fab batch switch`: Resolves changes, creates worktrees with correct branch, opens tmux tabs
- [ ] CHK-012 `fab batch archive`: Finds archivable changes (hydrate done|skipped), spawns Claude session

## Behavioral Correctness
- [ ] CHK-013 Spawn command resolution: Reads `agent.spawn_command` from config.yaml, falls back to `claude --dangerously-skip-permissions`
- [ ] CHK-014 `fab batch new --list` / `--all` / no-args-shows-list behavior matches shell predecessor
- [ ] CHK-015 `fab batch switch --list` / `--all` behavior matches shell predecessor
- [ ] CHK-016 `fab batch archive --list` / `--all` / default-all behavior matches shell predecessor

## Removal Verification
- [ ] CHK-017 `fab/.kit/scripts/` directory deleted (all 6 scripts + lib/ with 2 files)
- [ ] CHK-018 No remaining code references `fab/.kit/scripts/` paths

## Scenario Coverage
- [ ] CHK-019 Doctor: all-pass scenario (exit 0, 7/7 summary)
- [ ] CHK-020 Doctor: some-fail scenario (non-zero exit, install hints)
- [ ] CHK-021 Operator: not-in-tmux scenario (error message, exit 1)
- [ ] CHK-022 Batch new: not-in-tmux scenario (error message, exit 1)
- [ ] CHK-023 Frontmatter parser: quoted and unquoted values, inline comments

## Edge Cases & Error Handling
- [ ] CHK-024 Batch new: backlog ID not found in backlog.md — warning and skip
- [ ] CHK-025 Batch switch: change resolution failure — warning and skip
- [ ] CHK-026 Batch archive: change not archivable (hydrate not done/skipped) — warning and skip
- [ ] CHK-027 Batch archive: no eligible changes — error message

## Code Quality
- [ ] CHK-028 Pattern consistency: New Go code follows existing cmd/ file patterns (cobra command structure, error handling, exec.Command usage)
- [ ] CHK-029 No unnecessary duplication: Shared spawn and frontmatter packages reused across all consumers
- [ ] CHK-030 No god functions: Each command handler delegates to focused helper functions

## Documentation Accuracy
- [ ] CHK-031 `/fab-setup` skill references `fab doctor` (not shell script path)
- [ ] CHK-032 `/fab-help` skill references `fab fab-help` (not shell script path)
- [ ] CHK-033 `/fab-operator` skill references `fab operator` (not shell script path)
- [ ] CHK-034 README.md script reference table updated

## Cross References
- [ ] CHK-035 Router allowlist in `src/go/fab-kit/cmd/fab/main.go` includes `"doctor"`
- [ ] CHK-036 `fab-go` main.go registers `fab-help`, `operator`, and `batch` commands

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
