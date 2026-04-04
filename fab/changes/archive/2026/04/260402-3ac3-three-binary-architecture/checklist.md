# Quality Checklist: Three-Binary Architecture

**Change**: 260402-3ac3-three-binary-architecture
**Generated**: 2026-04-02
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Three-binary split: `fab`, `fab-kit`, `fab-go` each exist as independent binaries with their own `--help`
- [x] CHK-002 Router negative-match dispatch: fab-kit commands (init, upgrade, sync) route to fab-kit; all others route to fab-go
- [x] CHK-003 Composed help: `fab help` shows workspace + workflow groups inside a repo, workspace-only outside
- [x] CHK-004 fab-kit sync — directory scaffolding: creates `fab/changes/`, `fab/changes/archive/`, `docs/memory/`, `docs/specs/` with `.gitkeep`
- [x] CHK-005 fab-kit sync — scaffold tree-walk: JSON merge, line-ensure merge, and copy-if-absent all work correctly
- [x] CHK-006 fab-kit sync — multi-agent deployment: skills deployed per agent format (copy for Claude/Codex/Gemini, symlink for OpenCode)
- [x] CHK-007 fab-kit sync — stale cleanup: skill entries not in canonical list are removed from agent directories
- [x] CHK-008 fab-kit sync — version stamp: `fab/.kit-sync-version` written after successful sync
- [x] CHK-009 fab-kit sync — project scripts: `fab/sync/*.sh` discovered and executed in sorted order
- [x] CHK-010 fab-kit sync — prerequisites: validates git, bash, yq v4+, jq, gh, direnv
- [x] CHK-011 Source layout: single Go module at `src/go/fab-kit/` with `cmd/fab/`, `cmd/fab-kit/`, `internal/`
- [x] CHK-012 Build system: `just build-all` produces 20 binaries (5 x 4 platforms)
- [x] CHK-013 Brew packaging: archives contain fab, fab-kit, wt, idea
- [x] CHK-014 CI workflow: release produces correct archives and updates Homebrew formula for 4 binaries

## Behavioral Correctness
- [x] CHK-015 Router dispatch matches shim behavior: `fab resolve`, `fab status`, etc. still reach fab-go correctly
- [x] CHK-016 `fab init` and `fab upgrade` still work via `fab` router (routed to fab-kit)
- [x] CHK-017 Scaffold JSON merge produces same output as shell `json_merge_permissions()` (union, dedup)
- [x] CHK-018 Scaffold line merge produces same output as shell `line_ensure_merge()` (non-duplicate, non-comment)
- [x] CHK-019 Agent detection matches shell behavior (PATH lookup, FAB_AGENTS override)

## Removal Verification
- [x] CHK-020 `fab/.kit/scripts/fab-sync.sh` removed
- [x] CHK-021 `fab/.kit/sync/1-prerequisites.sh` removed
- [x] CHK-022 `fab/.kit/sync/2-sync-workspace.sh` removed
- [x] CHK-023 `fab/.kit/sync/3-direnv.sh` removed
- [x] CHK-024 `src/go/shim/` directory no longer exists
- [x] CHK-025 No references to `fab-sync.sh` remain in skill files or project context

## Scenario Coverage
- [x] CHK-026 Fresh repo sync: all directories and scaffold files created correctly
- [x] CHK-027 Existing repo sync: copy-if-absent skips existing files, merges work correctly
- [x] CHK-028 Agent not installed: deployment skipped with informational message
- [x] CHK-029 Not in fab repo + workflow command: router shows "Not in a fab-managed repo" error
- [x] CHK-030 Version not cached: router auto-fetches via EnsureCached before dispatching

## Edge Cases & Error Handling
- [x] CHK-031 Missing prerequisite halts sync with actionable error
- [x] CHK-032 Project sync script failure halts pipeline
- [x] CHK-033 fab-go not available (no config.yaml, outside repo): router handles gracefully for non-repo commands

## Code Quality
- [x] CHK-034 Pattern consistency: new Go code follows naming and structural patterns of existing fab-go and shim code
- [x] CHK-035 No unnecessary duplication: cache/download/config code shared via `internal/`, not copied
- [x] CHK-036 No god functions: sync logic decomposed into focused functions (scaffolding, tree-walk, skill deploy, cleanup)

## Documentation Accuracy
- [x] CHK-037 `fab/.kit/skills/_cli-fab.md` updated if CLI calling conventions change
- [x] CHK-038 References to `fab-sync.sh` in skill files updated to `fab sync`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
