# Quality Checklist: Scriptify Fab-Archive

**Change**: 260303-hcq9-scriptify-fab-archive
**Generated**: 2026-03-04
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Script location: `archiveman.sh` exists at `fab/.kit/scripts/lib/archiveman.sh` and is executable
- [x] CHK-002 CLI dispatch: `archive`, `restore`, `list`, `--help` subcommands work; unknown subcommands exit 1
- [x] CHK-003 Archive subcommand: clean → move → index → pointer steps execute in sequence
- [x] CHK-004 Archive index format: `# Archive Index` header, `- **{name}** — {desc}` entries, most-recent-first
- [x] CHK-005 Archive backfill: unindexed archived folders get placeholder entries on every invocation
- [x] CHK-006 Archive YAML output: all 6 fields present (action, name, clean, move, index, pointer)
- [x] CHK-007 Restore subcommand: move → index remove → pointer steps execute in sequence
- [x] CHK-008 Restore resolution: case-insensitive substring matching against archive folder names
- [x] CHK-009 Restore YAML output: all 5 fields present (action, name, move, index, pointer)

- [x] CHK-010 List subcommand: one folder name per line, excludes index.md, empty/missing archive returns exit 0
- [x] CHK-011 Skill archive mode: calls `archiveman.sh archive` once, backlog matching still agent-driven
- [x] CHK-012 Skill restore mode: calls `archiveman.sh restore` once with optional `--switch`

## Behavioral Correctness
- [x] CHK-013 Archive pointer: cleared when change is active, skipped otherwise
- [x] CHK-014 Restore `--switch`: activates change via `changeman.sh switch`, skipped without flag
- [x] CHK-015 `--description` required: archive exits 1 without it
- [x] CHK-016 User-facing output unchanged: report format matches existing `/fab-archive` output

## Scenario Coverage
- [x] CHK-017 Archive happy path: active change with .pr-done → full archive with pointer clear
- [x] CHK-018 Archive non-active change: pointer step skipped
- [x] CHK-019 Index creation: new index.md created when absent
- [x] CHK-020 Restore happy path: archived change restored with --switch
- [x] CHK-021 Restore without --switch: pointer skipped
- [x] CHK-022 Restore multiple matches: exits 1 with listing
- [x] CHK-023 Restore no match: exits 1 with error

## Edge Cases & Error Handling
- [x] CHK-024 Archive creates `archive/` directory if missing
- [x] CHK-025 Backfill no-op when all folders already indexed
- [x] CHK-026 Restore when folder already in changes: `already_in_changes`, remaining steps still run
- [x] CHK-027 Index preserved as header-only after last entry removed
- [x] CHK-028 Empty/missing archive dir handled gracefully by list (exit 0)

## Code Quality
- [x] CHK-029 Pattern consistency: follows changeman.sh/resolve.sh conventions (LIB_DIR, FAB_ROOT, set -euo pipefail, CLI dispatch)
- [x] CHK-030 No unnecessary duplication: reuses resolve.sh for active change resolution, changeman.sh for pointer ops
- [x] CHK-031 Readability: functions focused and appropriately sized (no god functions >50 lines)
- [x] CHK-032 No magic strings: error messages and YAML field names use named variables where applicable

## Documentation Accuracy
- [x] CHK-033 `--help` output covers all subcommands with usage examples
- [x] CHK-034 Skill SKILL.md accurately reflects the new orchestrator pattern

## Cross References
- [x] CHK-035 No stale references: skill no longer contains inline file move/index/pointer logic
- [x] CHK-036 Script references in skill use repo-root-relative paths (`fab/.kit/scripts/lib/archiveman.sh`)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
