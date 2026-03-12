# Intake: Remove fab status show and fix stale fab/current references

**Change**: 260312-9lci-fix-status-show-fab-current
**Created**: 2026-03-12
**Status**: Draft

## Origin

> fix: remove `fab status show` subcommand (superseded by `fab pane-map`, `wt list`, and `fab change list`) and update stale `fab/current` references to `.fab-status.yaml`

Discussion sessions via `/fab-discuss` identified the scope expansion.

## Why

### 1. `fab status show` is redundant

`fab status show` was built as a replacement for the shell `wt-status` script. It shows worktree name, path, branch, and active change — but **not** pipeline stage, confidence, or agent state. Every consumer that needs status already uses better tools:

| Need | Better tool | What `fab status show` lacks |
|------|------------|---------------------------|
| Operator observation (tmux) | `fab pane-map` | Stage, agent state, pane ID |
| Operator observation (no tmux) | `wt list` + `fab change list` | Stage, confidence |
| Worktree existence check | `wt list --path <name>` | Clean exit code semantics |
| Pipeline state queries | `fab change list` | Stage, state, confidence per change |

No skill, script, or workflow depends on `fab status show` — it was retained only as a "fallback" in operator skills, which has now been replaced with `wt list` + `fab change list`.

### 2. `fab/current` references are stale

The `fab/current` pointer file was replaced by `.fab-status.yaml` symlink in change `260307-x2tx-status-symlink-pointer` (migration `0.32.0-to-0.34.0`). Several documentation files still describe `fab/current` as the active mechanism.

## What Changes

### 1. Remove `fab status show` subcommand

Delete the `show` subcommand from `src/go/fab/cmd/fab/status.go`:
- Remove the `showCmd` cobra command and its `runShow` function
- Remove `resolveWorktreeFabState` helper (only used by show)
- Remove `worktreeInfo` struct and related types (only used by show)
- Keep all other `fab status` subcommands (`confidence`, `get-prs`, `advance`, `finish`, etc.) — these are change-scoped and work correctly

### 2. Documentation: remove `fab status show` references

| File | Action |
|------|--------|
| `fab/.kit/skills/_scripts.md` | Remove `fab status show` entry from the command table and any description sections |
| `docs/memory/fab-workflow/kit-architecture.md` | Remove `fab status show` from the subcommand list; update any text describing it |
| `docs/memory/fab-workflow/execution-skills.md` | Replace `fab status show --all` fallback references with `wt list` + `fab change list` |

### 3. Documentation: fix stale `fab/current` references

Update these files to replace `fab/current` with `.fab-status.yaml` where the text describes the *current* mechanism (not historical context):

| File | Lines | Nature |
|------|-------|--------|
| `README.md` | 58, 158 | Directory tree shows `fab/current/`, text says "make it active via fab/current" |
| `fab/.kit/skills/_scripts.md` | 319 | `fab send-keys` pane resolution says "read `fab/current`" |
| `docs/memory/fab-workflow/kit-architecture.md` | 414 | `fab send-keys` pane resolution says "read `fab/current`" |

Files with historical/changelog references (e.g., "replaced the former `fab/current`") are left as-is — those are accurate historical context.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Remove `fab status show` from subcommand list, update `fab send-keys` pane resolution description
- `fab-workflow/execution-skills`: (modify) Replace `fab status show --all` fallback references with `wt list` + `fab change list`

## Impact

- `src/go/fab/cmd/fab/status.go` — remove `show` subcommand, `resolveWorktreeFabState`, and `worktreeInfo`
- `fab/.kit/skills/_scripts.md` — remove `fab status show` entry, fix `fab send-keys` `fab/current` reference
- `docs/memory/fab-workflow/kit-architecture.md` — remove `fab status show` from list, fix `fab send-keys` text
- `docs/memory/fab-workflow/execution-skills.md` — replace fallback references
- `README.md` — fix directory tree and activation text
- Tests for `fab status show` (if any) should be removed

## Open Questions

None — scope is well-defined.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `.fab-status.yaml` is the active pointer mechanism | Config, constitution, migration 0.32.0-to-0.34.0 all confirm this | S:95 R:95 A:95 D:95 |
| 2 | Certain | Historical `fab/current` references in changelogs are left as-is | These accurately describe what happened at the time — modifying history is wrong | S:90 R:90 A:90 D:95 |
| 3 | Certain | `fab status show` has no remaining consumers | Operator skills now use `wt list` + `fab change list`; no other skill or script references it | S:95 R:85 A:90 D:90 |
| 4 | Certain | All other `fab status` subcommands are unaffected | They are change-scoped, take explicit `<change>` args, and use `internal/resolve` | S:90 R:90 A:90 D:90 |
| 5 | Certain | `fab pane-map` + `wt list` + `fab change list` fully cover all use cases | Discussion confirmed — pane-map for tmux, wt list for worktrees, change list for pipeline state | S:90 R:85 A:90 D:85 |

5 assumptions (5 certain, 0 confident, 0 tentative, 0 unresolved).
