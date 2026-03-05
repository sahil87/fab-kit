# Intake: Ship wt Go Binary

**Change**: 260305-k8ds-4-ship-wt-go-binary
**Created**: 2026-03-05
**Status**: Draft

## Origin

> 4-ship-wt-go-binary: Ship the wt Go binary — parity test harness for wt commands, cross-compilation alongside fab binary in fab-release.sh, switchover from bash wt-* scripts to wt binary.

Follows the wt build change (260305-jug9-3-build-wt-go-binary). Same pattern as the fab ship change: parity tests → release integration → switchover.

## Why

1. **Parity verification**: Same rationale as fab ship — verify Go wt binary produces identical behavior to bash wt-* scripts before switching callers.

2. **Distribution**: wt binary ships alongside fab binary in the per-platform kit archives. Users get both binaries pre-built.

3. **Consistent switchover**: Shim layer in wt-* scripts, then direct invocation after confidence period.

## What Changes

### Parity Test Harness for wt

Test suite at `src/fab-go/test/parity/wt/` that:

1. Creates temporary git repositories with worktrees as test fixtures
2. For each wt command, runs both `bash wt-<command> <args>` and `wt <command> <args>`
3. Diffs outputs: stdout, stderr, exit codes, and git state (worktrees created/deleted, branches, stash state)
4. Covers: create, list, open (non-interactive), delete (non-interactive), pr (mocked gh), init, status

Note: Interactive flows (menus, fzf) tested separately via Go unit tests, not parity tests.

### Cross-Compilation Update in fab-release.sh

Extend `src/scripts/fab-release.sh` to also build the `wt` binary:

```bash
for pair in "darwin/arm64" "darwin/amd64" "linux/arm64" "linux/amd64"; do
  GOOS="${pair%/*}" GOARCH="${pair#*/}" go build -o "../../fab/.kit/bin/fab" ./cmd/fab
  GOOS="${pair%/*}" GOARCH="${pair#*/}" go build -o "../../fab/.kit/bin/wt" ./cmd/wt
  # Package archive with both binaries
done
```

Both binaries included in each per-platform archive. Build time increases from ~15s to ~25s.

### Shim Layer in wt-* Scripts

Each `fab/.kit/packages/wt/bin/wt-*` script gets a shim at the top:

```bash
if command -v wt >/dev/null 2>&1; then
  wt <subcommand> "$@"
  exit $?
fi
# ... original bash implementation continues below ...
```

Mapping: `wt-create` → `wt create`, `wt-list` → `wt list`, etc.

### PATH and env-packages.sh Update

Update `fab/.kit/scripts/lib/env-packages.sh` to add `fab/.kit/bin/` to PATH (for both `fab` and `wt` binaries), in addition to the existing `fab/.kit/packages/*/bin` entries.

### Switchover

After shim validation, callers that invoke `wt-create`, `wt-list`, etc. can optionally be updated to `wt create`, `wt list`. However, since wt commands are primarily user-facing (typed in terminal), not called by skills, the shim layer may be sufficient — users continue typing `wt-create` and it transparently delegates.

### Remove Legacy wt Shell Scripts

After switchover and confidence period, remove the bash wt-* scripts entirely (absorbed from archived `socx` change):

- Delete `fab/.kit/packages/wt/bin/wt-create`, `wt-delete`, `wt-init`, `wt-list`, `wt-open`, `wt-pr`
- Delete `fab/.kit/packages/wt/lib/wt-common.sh`
- Remove the `fab/.kit/packages/wt/` directory if empty after deletion
- Update `env-packages.sh` PATH entries to remove `fab/.kit/packages/wt/bin`

## Affected Memory

- `fab-workflow/distribution`: (modify) Document wt binary in per-platform archives, env-packages.sh update

## Impact

- **Release pipeline**: `src/scripts/fab-release.sh` — adds wt binary build (~10s additional)
- **All wt-* scripts**: Shim added at top (non-destructive)
- **env-packages.sh**: Adds `fab/.kit/bin/` to PATH
- **Batch scripts**: `batch-fab-new-backlog.sh`, `batch-fab-switch-change.sh` — may reference wt-* directly

## Open Questions

- None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Same parity testing pattern as fab ship change | Proven pattern from Phase 1 | S:90 R:85 A:90 D:95 |
| 2 | Certain | Both binaries in same per-platform archive | Discussed — single download, same distribution path | S:85 R:85 A:85 D:90 |
| 3 | Confident | Shim may be permanent for wt commands | wt-* commands are user-facing — users type them directly. Changing muscle memory from `wt-create` to `wt create` has cost. Shim makes this transparent | S:70 R:90 A:75 D:65 |
| 4 | Confident | `fab/.kit/bin/` added to PATH via env-packages.sh | Central place for binary PATH management. Alternative: separate mechanism. env-packages.sh is the existing pattern | S:75 R:85 A:80 D:75 |

4 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).
