# Quality Checklist: Merge Claude Code Hooks Into Go Binary

**Change**: 260310-bvc6-merge-hooks-into-go
**Generated**: 2026-03-10
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 `fab hook session-start`: Clears agent idle state for active change, exits 0 always
- [ ] CHK-002 `fab hook stop`: Sets agent idle timestamp for active change, exits 0 always
- [ ] CHK-003 `fab hook user-prompt`: Clears agent idle state for active change, exits 0 always
- [ ] CHK-004 `fab hook artifact-write`: Parses PostToolUse JSON from stdin, performs per-artifact bookkeeping, outputs additionalContext JSON
- [ ] CHK-005 `fab hook sync`: Discovers hook scripts, maps to events, merges into settings.local.json, idempotent
- [ ] CHK-006 Runtime package extraction: `internal/runtime/` package contains exported helpers, used by both `fab runtime` and `fab hook` commands
- [ ] CHK-007 Shell wrappers: All four hook scripts are thin `exec` one-liners delegating to binary
- [ ] CHK-008 `5-sync-hooks.sh`: Rewritten to delegate to `fab hook sync`
- [ ] CHK-009 jq removed from `fab-doctor.sh` prerequisites
- [ ] CHK-010 `_scripts.md` updated with `fab hook` command group documentation

## Behavioral Correctness
- [ ] CHK-011 `fab hook session-start` behavior matches original `on-session-start.sh` (clears idle, exits 0 on all errors)
- [ ] CHK-012 `fab hook stop` behavior matches original `on-stop.sh` (sets idle, exits 0 on all errors)
- [ ] CHK-013 `fab hook artifact-write` behavior matches original `on-artifact-write.sh` (keyword matching, score, checklist bookkeeping)
- [ ] CHK-014 `fab hook sync` behavior matches original `5-sync-hooks.sh` (event mapping, dedup, JSON merge, status messages)
- [ ] CHK-015 `fab runtime set-idle/clear-idle/is-idle` still work correctly after runtime package extraction

## Scenario Coverage
- [ ] CHK-016 No active change: all hook subcommands exit 0 silently
- [ ] CHK-017 Runtime file missing: session-start/user-prompt exit silently, stop creates file
- [ ] CHK-018 Invalid stdin JSON: artifact-write exits 0 silently
- [ ] CHK-019 Non-fab file path: artifact-write exits 0 silently
- [ ] CHK-020 Absolute and relative artifact paths both matched correctly
- [ ] CHK-021 All keyword categories map to correct change type (fix, refactor, docs, test, ci, chore, feat)
- [ ] CHK-022 Hook sync deduplication: running twice produces no changes
- [ ] CHK-023 Hook sync preserves existing non-hook settings in settings.local.json
- [ ] CHK-024 Binary missing: shell wrappers exit 0 gracefully

## Edge Cases & Error Handling
- [ ] CHK-025 Broken `.fab-status.yaml` symlink: all hooks exit 0 silently
- [ ] CHK-026 Change folder doesn't resolve: artifact-write exits 0 silently
- [ ] CHK-027 Empty stdin: artifact-write exits 0 silently
- [ ] CHK-028 `settings.local.json` doesn't exist: hook sync creates it
- [ ] CHK-029 `settings.local.json` is empty `{}`: hook sync creates hooks section

## Code Quality
- [ ] CHK-030 Pattern consistency: New Go code follows naming and structural patterns of existing cmd/fab and internal packages
- [ ] CHK-031 No unnecessary duplication: Shared runtime logic in internal/runtime, shared hook logic in internal/hooklib
- [ ] CHK-032 Readability: Functions <50 lines, clear naming, no magic strings

## Documentation Accuracy
- [ ] CHK-033 `_scripts.md` command reference table includes `fab hook` entry
- [ ] CHK-034 `_scripts.md` detailed section documents all five hook subcommands with usage and purpose

## Cross References
- [ ] CHK-035 `fab hook sync` mapping table matches shell wrapper filenames in `fab/.kit/hooks/`
- [ ] CHK-036 Keyword matching rules in Go match the rules documented in intake.md and spec.md

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
