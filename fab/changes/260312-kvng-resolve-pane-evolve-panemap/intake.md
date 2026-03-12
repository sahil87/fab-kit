# Intake: Add resolve --pane, drop send-keys, evolve pane-map

**Change**: 260312-kvng-resolve-pane-evolve-panemap
**Created**: 2026-03-12
**Status**: Draft

## Origin

> Add --pane flag to fab resolve, delete fab send-keys subcommand, and evolve fab pane-map to show all tmux panes (not just fab worktrees). The operator's mental model is change-centric ("I have a change, where is it?") but there's no standalone change→pane primitive. resolve --pane fills that gap, making send-keys redundant. pane-map currently silently excludes non-fab panes, but the operator needs to see all tabs since some may not have a change started yet.

Emerged from a `/fab-discuss` session exploring tmux tooling. Key design decisions were made in conversation:
- Rejected a `fab tmux` namespace for raw tmux discovery — the interesting boundary is "change-aware vs raw tmux", and fab's contribution is the change resolution layer
- Agreed that `resolve --pane` is the right primitive since `resolve` already handles all change→handle lookups (`--id`, `--folder`, `--dir`, `--status`)
- `send-keys` becomes redundant: `tmux send-keys -t $(fab resolve --pane r3m7) "/fab-continue" Enter`
- pane-map should show all panes because the operator needs to see tabs where agents haven't started changes yet

## Why

1. **Missing primitive**: The operator's mental model is "I have a change, where is it running?" but there's no standalone change→pane lookup. `send-keys` bundles the lookup with the send action, making it non-composable.
2. **Duplicate mechanisms**: `send-keys` is just `resolve --pane` + `tmux send-keys`. The user explicitly wants no duplicate ways of doing the same thing.
3. **Hidden panes**: `pane-map` silently excludes non-fab panes (non-git dirs, git repos without `fab/`). The operator needs to see all tabs — some may not have a change started yet (the task might be to create the change itself, or the agent might not have been started).

## What Changes

### 1. Add `--pane` flag to `fab resolve`

Add a new output mode to the existing `resolve` command in `src/go/fab/cmd/fab/resolve.go`:

```
fab resolve <change> --pane    → outputs tmux pane ID (e.g., %5)
```

Behavior:
- Guard: check `$TMUX` is set, error if not
- Discover panes via `discoverPanes()` (reuse from panemap.go, same package)
- Match change folder to pane via `matchPanesByFolder()` (move from sendkeys.go)
- Error if no match: `no tmux pane found for change "<folder>"`
- Warning to stderr if multiple matches (same as current send-keys behavior)
- Print first matching pane ID to stdout

Register in `PreRunE` priority chain after `--status`, before default `--id`. Register as `cmd.Flags().Bool("pane", false, "Output tmux pane ID")`.

### 2. Delete `fab send-keys` subcommand

Remove entirely from the Go binary:
- Delete `src/go/fab/cmd/fab/sendkeys.go`
- Delete `src/go/fab/cmd/fab/sendkeys_test.go`
- Remove `sendKeysCmd()` from `main.go` AddCommand list

Before deleting, move two reusable functions from `sendkeys.go` to `panemap.go`:
- `resolvePaneChange(p paneEntry) string` — resolves a pane to its active change folder
- `matchPanesByFolder(panes []paneEntry, folder string, resolveFunc func(paneEntry) string) ([]string, string)` — testable matcher

Move corresponding tests from `sendkeys_test.go` to `panemap_test.go`:
- `TestMatchPanesByFolder`
- `TestResolvePaneChange`

Delete remaining sendkeys-only tests (`TestBuildSendKeysArgs`, `TestBuildSendKeysArgsWithSpaces`, `TestValidateSendKeysInputs`).

Ignore the Rust binary — it is not maintained.

### 3. Evolve `pane-map` to show all panes

Change `resolvePane()` in `panemap.go` to always return `true` instead of silently excluding non-fab panes:

- If `gitWorktreeRoot()` fails (not a git repo): show `filepath.Base(p.cwd) + "/"` for Worktree, em dashes (`—`) for Change/Stage/Agent
- If git repo exists but no `fab/` directory: show computed worktree path, em dashes for Change/Stage/Agent
- If both exist: unchanged (current fab-aware logic)

Update the empty-rows message from "No fab worktrees found in tmux panes." to "No tmux panes found." (near-impossible edge case).

### 4. Update documentation

**`fab/.kit/skills/_scripts.md`**:
- Remove `fab send-keys` section (lines 306-340) and command summary table entry (line 47)
- Add `--pane` to the `fab resolve` section's flag table
- Update `fab pane-map` section to note it shows all panes

**`fab/.kit/skills/fab-operator1.md`** and **`fab/.kit/skills/fab-operator2.md`**:
- Replace `fab send-keys <change> "<text>"` with `fab resolve <change> --pane` + raw `tmux send-keys -t <pane> "<text>" Enter`
- Update Available Tools tables, use case examples, pre-send validation sections

**`docs/specs/skills/SPEC-fab-operator1.md`** and **`docs/specs/skills/SPEC-fab-operator2.md`**:
- Update primitives sections to replace send-keys with resolve --pane

**`docs/memory/fab-workflow/kit-architecture.md`**:
- Remove send-keys from subcommand list, add resolve --pane, update pane-map behavior description

**`docs/memory/fab-workflow/execution-skills.md`**:
- Update send-keys references to resolve --pane pattern

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Remove send-keys subcommand, add resolve --pane flag, update pane-map behavior
- `fab-workflow/execution-skills`: (modify) Update operator tool references from send-keys to resolve --pane

## Impact

- **Go binary**: `src/go/fab/cmd/fab/` — resolve.go modified, sendkeys.go deleted, panemap.go modified, main.go modified
- **Operator skills**: Both operator1 and operator2 reference send-keys extensively — all references must be updated to the new pattern
- **Shell scripts**: `dispatch.sh` and `run.sh` use raw `tmux send-keys` (not `fab send-keys`) — no changes needed
- **Breaking change**: Any user calling `fab send-keys` directly will get an unknown command error

## Open Questions

None — all design decisions were resolved in conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Extend `fab resolve` with `--pane` flag rather than new subcommand | Discussed — resolve already handles all change→handle lookups, --pane is the natural addition | S:95 R:90 A:95 D:95 |
| 2 | Certain | Delete `fab send-keys` entirely | Discussed — user explicitly chose removal over coexistence, "no duplicate ways of doing the same thing" | S:95 R:85 A:90 D:95 |
| 3 | Certain | `pane-map` shows all tmux panes, not just fab ones | Discussed — operator needs to see tabs where no change started yet | S:90 R:90 A:90 D:90 |
| 4 | Certain | Go binary only, ignore Rust binary | Discussed — user stated "we aren't using or maintaining the rust binary" | S:95 R:95 A:95 D:95 |
| 5 | Certain | Non-fab panes show em dashes for Change/Stage/Agent columns | Codebase convention — existing pane-map already uses em dashes for missing values | S:85 R:95 A:90 D:90 |
| 6 | Certain | Move `matchPanesByFolder` and `resolvePaneChange` to panemap.go | Both depend on types/functions already in panemap.go (paneEntry, gitWorktreeRoot, readFabCurrent) | S:85 R:90 A:90 D:85 |
| 7 | Confident | Non-git panes show `filepath.Base(cwd) + "/"` for Worktree | Consistent with existing fallback pattern in `worktreeDisplayPath` | S:75 R:90 A:80 D:80 |
| 8 | Confident | Shell scripts (dispatch.sh, run.sh) unaffected | They use raw `tmux send-keys` with pane IDs, not `fab send-keys` | S:85 R:90 A:85 D:90 |

8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).
