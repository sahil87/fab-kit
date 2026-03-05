# Intake: Unified Fab Dispatcher

**Change**: 260305-qagd-unified-fab-dispatcher
**Created**: 2026-03-05
**Status**: Draft

## Origin

> Discussion session exploring the long-term binary architecture for fab-kit. The current setup has 7 shell scripts in `fab/.kit/scripts/lib/` that each contain a 3-line shim delegating to the Go binary at `fab/.kit/bin/fab` when available. The user proposed inverting this: a single shell dispatcher script named `fab` that checks for compiled binaries (rust > go > shell) and delegates accordingly. This aligns with backlog items gm11 (switchover) and gm12 (remove shim layer).

Key decisions from discussion:
- The shell-script fallback must be fully functional — don't assume Go binary is always available
- The dispatcher is a shell script, not a compiled binary
- Priority chain: fab-rust > fab-go > shell scripts
- Rust takes priority as strict superset (can revisit later)
- `_scripts.md` should be updated to document the new architecture
- Batch scripts should use `fab/.kit/bin/fab` instead of direct `changeman.sh` calls

## Why

1. **Duplicated delegation logic**: Every shell script in `scripts/lib/` has the same 3-line `_fab_bin` shim. This is 7x duplication of the same pattern.
2. **Inverted control flow**: Scripts delegate "up" to the binary, rather than a single entry point delegating "down" to the right backend. This makes the architecture harder to reason about.
3. **No path for multiple compiled backends**: The current shim only knows about `fab/.kit/bin/fab` (the Go binary). Adding Rust would mean updating all 7 scripts again.
4. **Inconsistent calling patterns**: Skills use `fab/.kit/bin/fab <command>`, batch scripts sometimes call `changeman.sh` directly. A single dispatcher makes the entry point unambiguous.

If we don't fix this: every new compiled backend requires touching all 7 scripts, and the delegation pattern keeps getting more convoluted.

## What Changes

### 1. New shell dispatcher at `fab/.kit/bin/fab`

A ~30-line shell script that:
- Resolves its own directory (`SCRIPT_DIR`)
- Checks for `fab-rust` (executable) → `exec "$SCRIPT_DIR/fab-rust" "$@"`
- Checks for `fab-go` (executable) → `exec "$SCRIPT_DIR/fab-go" "$@"`
- Falls back to shell scripts via a `case "$1"` routing table:

```bash
case "$1" in
  resolve)    shift; exec bash "$LIB_DIR/resolve.sh" "$@" ;;
  status)     shift; exec bash "$LIB_DIR/statusman.sh" "$@" ;;
  log)        shift; exec bash "$LIB_DIR/logman.sh" "$@" ;;
  preflight)  shift; exec bash "$LIB_DIR/preflight.sh" "$@" ;;
  change)     shift; exec bash "$LIB_DIR/changeman.sh" "$@" ;;
  score)      shift; exec bash "$LIB_DIR/calc-score.sh" "$@" ;;
  archive)    shift; exec bash "$LIB_DIR/archiveman.sh" "archive" "$@" ;;
  *)          echo "Unknown command: $1" >&2; exit 1 ;;
esac
```

Note: `archive` command injects `"archive"` as the first arg because `archiveman.sh` expects `archiveman.sh archive <change>` while the CLI signature is `fab archive <change>`.

### 2. Rename Go binary to `fab-go`

- Current: `fab/.kit/bin/fab` (Go binary, 7MB)
- New: `fab/.kit/bin/fab-go`
- Update `just build-go` to output to `fab/.kit/bin/fab-go`
- Update parity tests if they reference the binary path

### 3. Remove `_fab_bin` shims from all 7 shell scripts

Remove the 3-line delegation block from each:
- `resolve.sh` (lines 14-16)
- `statusman.sh` (lines 17-19)
- `logman.sh` (lines 17-19)
- `preflight.sh` (lines 5-7)
- `changeman.sh` (lines 18-20)
- `calc-score.sh` (lines 5-7)
- `archiveman.sh` (lines 17-23)

After removal, these scripts are pure shell implementations — only invoked when the dispatcher routes to them.

### 4. Update `_scripts.md`

Reframe from "Go binary is primary, shell scripts are fallback with shims" to:
- `fab/.kit/bin/fab` is a shell dispatcher (always shipped)
- Dispatcher checks `fab-rust` → `fab-go` → shell scripts
- Shell scripts in `scripts/lib/` are pure implementations (no shims)
- Remove "Legacy (shell scripts — still works via shim delegation)" framing
- Command signatures remain identical (no change to the calling convention)

### 5. Update batch scripts

Two batch scripts call `changeman.sh` directly for change resolution:
- `batch-fab-switch-change.sh` — uses `$CHANGEMAN` variable pointing to `changeman.sh`
- `batch-fab-archive-change.sh` — uses `$CHANGEMAN` variable pointing to `changeman.sh`

Update these to use `fab/.kit/bin/fab change resolve` / `fab/.kit/bin/fab change list` instead.

### 6. Update `justfile`

- `build-go` target: output to `fab/.kit/bin/fab-go` instead of `fab/.kit/bin/fab`
- May need a `build` target that builds whichever backend is configured

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the dispatcher pattern, backend priority chain, and `fab/.kit/bin/` layout

## Impact

- **`fab/.kit/bin/`**: New file layout (`fab` script + `fab-go` binary + `.gitkeep`)
- **`fab/.kit/scripts/lib/*.sh`**: All 7 scripts modified (shim removal)
- **`fab/.kit/skills/_scripts.md`**: Reframed documentation
- **`fab/.kit/scripts/batch-fab-switch-change.sh`**: Updated invocation
- **`fab/.kit/scripts/batch-fab-archive-change.sh`**: Updated invocation
- **`justfile`**: Updated build target
- **Parity tests** (`src/fab-go/test/parity/`): May need path updates if they reference `fab/.kit/bin/fab` directly
- **`.gitignore`**: May need update for `fab-go` / `fab-rust` binary names
- **Skills**: No changes — they already call `fab/.kit/bin/fab <command>`, which now points to the dispatcher

## Open Questions

- Should the dispatcher print a diagnostic when falling back to shell scripts (e.g., `[fab] using shell backend`)? Could be useful for debugging but noisy in normal operation.
- Should `fab --version` be handled by the dispatcher itself (reporting which backend is active) or passed through to the backend?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | CLI signature unchanged | Discussed — same `fab <command> [subcommand] [args]` interface, no skill changes needed | S:95 R:90 A:95 D:95 |
| 2 | Certain | Priority: rust > go > shell | Discussed — user explicitly chose this order | S:95 R:85 A:90 D:95 |
| 3 | Certain | Shell fallback must be complete | Discussed — user explicitly said "don't assume Go binary always available" | S:95 R:80 A:90 D:95 |
| 4 | Certain | Rename Go binary to fab-go | Discussed — user confirmed this naming | S:90 R:85 A:90 D:90 |
| 5 | Confident | Archive command needs arg injection | Codebase signal — archiveman.sh expects `archive` as first positional, CLI omits it | S:80 R:90 A:85 D:80 |
| 6 | Confident | No diagnostic output on fallback by default | Silent fallback is less noisy; listed as open question for user input | S:60 R:90 A:70 D:65 |
| 7 | Confident | Batch scripts use `fab change resolve` not direct changeman.sh | Discussed — user said batch scripts should use new signatures | S:85 R:85 A:80 D:75 |
| 8 | Confident | Parity tests may need path updates | Go binary moves from `fab/.kit/bin/fab` to `fab/.kit/bin/fab-go` | S:70 R:90 A:80 D:80 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
