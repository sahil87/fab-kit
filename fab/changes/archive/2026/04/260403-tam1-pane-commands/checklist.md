# Quality Checklist: Fab Pane Command Group

**Change**: 260403-tam1-pane-commands
**Generated**: 2026-04-03
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 `fab pane` parent command: shows help with four subcommands listed
- [x] CHK-002 `fab pane map`: identical output to former `fab pane-map` (table and JSON modes)
- [x] CHK-003 `fab pane capture`: default output shows header with pane metadata + captured content
- [x] CHK-004 `fab pane capture --json`: returns enriched JSON with pane, content, change, stage, agent_state fields
- [x] CHK-005 `fab pane capture --raw`: returns plain captured text only
- [x] CHK-006 `fab pane capture -l N`: respects custom line count
- [x] CHK-007 `fab pane send`: sends keystrokes and appends Enter by default
- [x] CHK-008 `fab pane send --no-enter`: sends without Enter
- [x] CHK-009 `fab pane send --force`: bypasses idle validation
- [x] CHK-010 `fab pane process`: shows process tree in human-readable format
- [x] CHK-011 `fab pane process --json`: returns JSON with pane_pid, processes tree, has_agent
- [x] CHK-012 `internal/pane.ValidatePane`: correctly validates pane existence
- [x] CHK-013 `internal/pane.ResolvePaneContext`: resolves worktree, change, stage, agent state
- [x] CHK-014 `internal/pane.GetPanePID`: returns correct shell PID

## Behavioral Correctness
- [x] CHK-015 `pane map` Use field is `"map"` not `"pane-map"`
- [x] CHK-016 `main.go` registers `paneCmd()` not `paneMapCmd()` at root level
- [x] CHK-017 `pane send` rejects send when agent state is `active`
- [x] CHK-018 `pane send` rejects send when agent state is `unknown`
- [x] CHK-019 `pane capture --json` and `--raw` are mutually exclusive
- [x] CHK-020 Process classification: `claude`/`claude-code` → agent, `node` → node, `git`/`gh` → git, other → other

## Removal Verification
- [x] CHK-021 `fab pane-map` root-level command no longer registered in main.go
- [x] CHK-022 `_cli-external.md` no longer documents `capture-pane` or `send-keys` in tmux table
- [x] CHK-023 Duplicated helper functions removed from `panemap.go` after extraction to `internal/pane`

## Scenario Coverage
- [x] CHK-024 Invalid pane ID returns error for all subcommands (capture, send, process)
- [x] CHK-025 Pane in non-fab directory: capture returns null fields, send rejects (unless --force)
- [x] CHK-026 Linux process discovery via /proc works (build tag `linux`)
- [x] CHK-027 macOS process discovery via ps works (build tag `darwin`)

## Edge Cases & Error Handling
- [x] CHK-028 `pane capture` on pane with no fab context: graceful null fields in JSON
- [x] CHK-029 `pane send` to pane with no `.fab-runtime.yaml`: treated as unknown, rejected
- [x] CHK-030 `pane process` when PID has no children: returns single root process node
- [x] CHK-031 `pane process` when `/proc` read fails (Linux): graceful error

## Code Quality
- [x] CHK-032 Pattern consistency: new commands follow Cobra patterns from existing commands (operator.go, batch_*.go)
- [x] CHK-033 No unnecessary duplication: `internal/pane` package reused by all subcommands, no copy-paste

## Documentation Accuracy
- [x] CHK-034 `_cli-fab.md` documents all four subcommands with correct signatures and flag tables
- [x] CHK-035 `_cli-external.md` tmux section only lists `new-window`

## Cross References
- [x] CHK-036 `_cli-fab.md` Command Reference table updated: `fab pane` replaces `fab pane-map`
- [x] CHK-037 `_cli-external.md` Usage Notes reference `fab pane capture`/`fab pane send` for internalized operations

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
