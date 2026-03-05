# Intake: Remove Legacy Shell Scripts

**Change**: 260305-socx-5-remove-legacy-shell-scripts
**Created**: 2026-03-05
**Status**: Draft

## Origin

> 5-remove-legacy-shell-scripts: After confidence period with Go binaries in production, remove shim layers from old shell scripts, then remove the shell scripts themselves. Covers both fab lib/ scripts and wt package scripts.

Deferred change — only executed after the Go binaries have been running in production long enough to confirm parity. This is the final cleanup step of the Go migration.

## Why

1. **Dead code removal**: Once the Go binaries are the sole code path, the bash scripts are dead code. Keeping them adds maintenance burden — any future changes would need to be made in both Go and bash.

2. **Distribution size**: Removing ~6,500 lines of bash scripts and the `yq` dependency from the kit reduces archive size and eliminates a prerequisite for new users.

3. **Clean architecture**: The constitution says "no build steps, no runtime frameworks" — Go binaries are pre-built single binaries, fully aligned. Removing the bash scripts eliminates the dual-implementation period.

## What Changes

### Remove Shim Layer from lib/ Scripts

Remove the `if command -v fab` shim block from the top of each lib/ script:
- `statusman.sh`, `resolve.sh`, `logman.sh`, `preflight.sh`, `changeman.sh`, `calc-score.sh`, `archiveman.sh`

At this point the scripts are no longer called by any skill or batch script (switchover already happened in changes 2 and 4).

### Remove lib/ Shell Scripts

Delete from `fab/.kit/scripts/lib/`:
- `statusman.sh` (1,294 lines)
- `changeman.sh` (562 lines)
- `calc-score.sh` (374 lines)
- `archiveman.sh` (359 lines)
- `resolve.sh` (179 lines)
- `logman.sh` (165 lines)
- `preflight.sh` (142 lines)

Keep: `env-packages.sh` (7 lines, still needed for PATH), `frontmatter.sh` (42 lines, markdown parser not ported to Go).

### Remove wt-* Shell Scripts

Delete from `fab/.kit/packages/wt/`:
- `bin/wt-create`, `bin/wt-list`, `bin/wt-open`, `bin/wt-delete`, `bin/wt-pr`, `bin/wt-init`, `bin/wt-status`
- `lib/wt-common.sh`

The `fab/.kit/packages/wt/` directory can be removed entirely if no other files remain.

### Update env-packages.sh

Remove the `fab/.kit/packages/*/bin` PATH entries for packages that no longer have shell scripts. Keep `fab/.kit/bin/` and `fab/.kit/packages/idea/bin/` entries.

### Remove yq Prerequisite

Update `fab/.kit/sync/1-prerequisites.sh` to remove `yq` from the required tools list. Update README prerequisites section. `yq` is no longer needed — the Go binary handles all YAML parsing internally.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Remove shell script references, update architecture to reflect Go-only
- `fab-workflow/distribution`: (modify) Remove yq prerequisite, update kit contents description

## Impact

- **Kit size**: Removes ~6,500 lines of bash, reduces kit.tar.gz
- **Prerequisites**: Removes `yq` dependency for end users
- **`idea` script**: Unaffected — remains as bash in `fab/.kit/packages/idea/bin/`
- **Tests**: Parity test harness can be removed or converted to pure Go integration tests

## Open Questions

- None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Keep `env-packages.sh` and `frontmatter.sh` | env-packages.sh is trivial PATH setup (7 lines); frontmatter.sh is a markdown parser not worth porting | S:85 R:90 A:90 D:90 |
| 2 | Certain | Keep `idea` as bash | Discussed — 353 lines of simple text manipulation, no performance problem | S:90 R:90 A:90 D:95 |
| 3 | Certain | Remove yq prerequisite | Go binary internalizes all YAML parsing — yq is no longer needed | S:90 R:80 A:90 D:95 |
| 4 | Confident | Remove parity test harness after cleanup | Parity tests compare bash vs Go — once bash is removed, they serve no purpose. Go unit/integration tests remain | S:75 R:85 A:80 D:75 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
