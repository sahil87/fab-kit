# Intake: Pane Map JSON Session Flags

**Change**: 260313-wrt4-pane-map-json-session-flags
**Created**: 2026-03-13
**Status**: Draft

## Origin

> Enhance `fab pane-map` with `--json`, `--session <name>`, and `--all-sessions` flags, plus add `window_index` to the output. Driven by run-kit's need to use pane-map as the per-window fab state data source in its backend poller.

Discussion context: run-kit's sidebar shows tmux sessions and windows. The backend poller enriches each window with fab state (change, stage, agent state). Currently it reads `.fab-status.yaml` and `.fab-runtime.yaml` directly, but this gives session-level state (one active change applied to all windows). Fab worktree windows don't show their own change's stage. Using `pane-map` gives per-pane resolution — each worktree window gets its own change/stage/agent state.

## Why

1. **Per-window fab state**: run-kit currently applies a single `.fab-status.yaml` to all windows in a session. Windows running different fab changes (via git worktrees) all show the same — or no — fab state. `pane-map` already resolves per-pane, making it the natural data source.

2. **Decouple consumers from file format**: Reading `.fab-runtime.yaml` directly is an implementation detail. `pane-map` provides a stable interface that can evolve internally without breaking consumers.

3. **Session targeting**: `pane-map` currently requires `$TMUX` (must be inside a tmux session). A backend process polling all sessions needs to target sessions by name or query all sessions at once.

## What Changes

### 1. `--json` flag

Add a `--json` boolean flag that switches output from the aligned table to a JSON array. Each element represents one pane:

```json
[
  {
    "session": "runK",
    "window_index": 3,
    "pane": "%7",
    "tab": "fab-3brm",
    "worktree": "run-kit.wt/260313-3brm-some-feature/",
    "change": "260313-3brm-some-feature",
    "stage": "apply",
    "agent_state": "active",
    "agent_idle_duration": null
  },
  {
    "session": "runK",
    "window_index": 0,
    "pane": "%1",
    "tab": "zsh",
    "worktree": "(main)",
    "change": null,
    "stage": null,
    "agent_state": null,
    "agent_idle_duration": null
  }
]
```

Null semantics: fields that are `—` (em dash) in table mode become `null` in JSON. `(no change)` also becomes `null` for `change`. Agent idle duration is `null` when agent is not idle.

When `--json` is not set, output is unchanged (current aligned table).

### 2. `--session <name>` flag

Add a `--session` string flag that targets a specific tmux session by name. Replaces the current `$TMUX`-based session discovery with `tmux list-panes -s -t <name>`.

When `--session` is provided, the `$TMUX` environment variable check is skipped — the command works from outside tmux.

### 3. `--all-sessions` flag

Add an `--all-sessions` boolean flag that queries all tmux sessions. Uses `tmux list-sessions` to enumerate sessions, then runs `tmux list-panes -s -t <session>` for each.

When `--all-sessions` is set, the `$TMUX` check is skipped. Output includes panes from all sessions, with the `session` field identifying which session each pane belongs to.

`--session` and `--all-sessions` are mutually exclusive — error if both are provided.

When neither `--session` nor `--all-sessions` is set, current behavior is preserved: the command uses `$TMUX` to discover the current session and only lists panes from that session. This is the default and requires being inside a tmux session (existing behavior, no regression).

### 4. `window_index` field

Add `window_index` (integer) to both table and JSON output. This is the tmux window index (`#{window_index}`). Required by run-kit to join pane-map results back to its `WindowInfo` structs keyed by `session:window_index`.

In table mode, add a `WinIdx` column between `Pane` and `Tab`.

In `discoverPanes()`, extend the tmux format string to include `#{window_index}` and add an `index int` field to `paneEntry`.

### 5. `session` field

Add session name to the output. In the current single-session mode, all rows share the same session name. With `--all-sessions`, it distinguishes which session each pane belongs to.

In `discoverPanes()`, extend the tmux format string to include `#{session_name}` and add a `session string` field to `paneEntry`.

In table mode, add a `Session` column as the first column (only when `--all-sessions` is used, to avoid noise in single-session mode).

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document pane-map flags and JSON output schema

## Impact

- **`src/go/fab/cmd/fab/panemap.go`** — primary implementation file; add flags, modify `discoverPanes()` format string, add JSON output path, extend `paneEntry`/`paneRow` structs
- **`src/go/fab/cmd/fab/panemap_test.go`** — add tests for new flags, JSON output format, session targeting, mutual exclusion
- **`fab/.kit/skills/_scripts.md`** — update `fab pane-map` documentation with new flags and output schema
- **Downstream consumer**: run-kit backend poller (separate repo, not part of this change)

## Open Questions

- None — the discussion in run-kit covered the design thoroughly.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `encoding/json` for `--json` output | Go stdlib, no external dependency needed | S:90 R:95 A:95 D:95 |
| 2 | Certain | `--session` and `--all-sessions` are mutually exclusive | Discussed — clear semantics, standard CLI pattern | S:85 R:90 A:90 D:95 |
| 3 | Certain | Null fields in JSON for em-dash/no-change values | Discussed — clean machine-readable semantics vs display strings | S:85 R:90 A:90 D:90 |
| 4 | Confident | `window_index` from `#{window_index}` tmux format var | Standard tmux variable; matches run-kit's existing usage | S:80 R:85 A:85 D:90 |
| 5 | Confident | Session column only in table mode with `--all-sessions` | Avoids noise in single-session table output; always present in JSON | S:70 R:90 A:75 D:80 |
| 6 | Confident | JSON output uses snake_case field names | Matches run-kit's Go JSON tags and fab-kit's existing YAML conventions | S:75 R:85 A:80 D:80 |
| 7 | Certain | `agent_idle_duration` as separate field in JSON (not embedded in `agent_state`) | Discussed — run-kit needs these as separate fields for its `WindowInfo` struct | S:90 R:85 A:90 D:90 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
