# Intake: Operator Observation Fixes

**Change**: 260310-b8ff-operator-observation-fixes
**Created**: 2026-03-10
**Status**: Draft

## Origin

> Fix pane-map session scoping, add tab name column, add `fab runtime is-idle` read subcommand, and remove `status show --all` from operator skill.

Identified during a deep-dive discussion of `/fab-operator1` and the commands it uses to observe state. Four issues surfaced from reading the Go implementation (`src/go/fab/cmd/fab/panemap.go`, `src/go/fab/cmd/fab/runtime.go`) and the hook scripts.

## Why

1. **pane-map shows noise from other sessions**: `tmux list-panes -a` spans all tmux sessions on the server. An operator running in one session sees unrelated panes from other sessions, making coordination harder.
2. **No tab context in pane-map**: The operator sees pane IDs (`%3`, `%7`) but not which tmux tab (window) they belong to — losing spatial orientation.
3. **`fab runtime` is write-only**: The CLI has `set-idle` and `clear-idle` but no read subcommand. The operator skill's pre-send validation (UC6) needs to check if an agent is idle before sending keys, but there's no clean way to do this.
4. **`status show --all` duplicates pane-map**: The `--all` flag is consumed exclusively by the operator skill. With pane-map already showing Stage per change, plus Agent state and Pane IDs that `status show --all` lacks, the flag is redundant overhead.

If not fixed: the operator skill sees noise from unrelated sessions, has no clean programmatic way to check idle state before sending commands to agents, and uses a redundant observation command.

**Note**: Agent idle tracking accuracy (agents showing idle while actively processing) is addressed by change `bvc6` which adds a `UserPromptSubmit` hook via `fab hook on-user-prompt`.

## What Changes

### 1. Scope `pane-map` to current tmux session

In `src/go/fab/cmd/fab/panemap.go`, change:

```
tmux list-panes -a -F "#{pane_id} #{pane_current_path}"
```

to:

```
tmux list-panes -s -F "#{pane_id} #{window_name} #{pane_current_path}"
```

`-s` lists all panes in the current session only (vs `-a` which lists all sessions). Add `#{window_name}` to capture the tmux tab name.

Add a **Tab** column to the output table between Pane and Worktree:

```
Pane   Tab        Worktree                       Change                              Stage     Agent
%3     alpha      myrepo.worktrees/alpha/        260306-r3m7-add-retry-logic         apply     active
%7     bravo      myrepo.worktrees/bravo/        260306-k8ds-ship-wt-binary          review    idle (2m)
%12    main       (main)                         260306-ab12-refactor-auth           hydrate   idle (8m)
```

### 2. Update `send-keys` pane discovery

`src/go/fab/cmd/fab/sendkeys.go` uses `tmux list-panes -a` for pane resolution. Switch to `-s` for consistency with pane-map — send-keys should only target panes in the current session.

### 3. Add `fab runtime is-idle <change>`

Add a new subcommand to the runtime module that reads `.fab-runtime.yaml` and outputs the idle state:

```
fab/.kit/bin/fab runtime is-idle <change>
```

**Behavior**:
- If the change has an `agent.idle_since` entry: exit 0, print `idle {duration}` to stdout (using the same duration format as pane-map: `Ns`, `Nm`, `Nh`)
- If the change has no `agent` block or no `idle_since`: exit 0, print `active` to stdout
- If `.fab-runtime.yaml` doesn't exist: exit 0, print `unknown` to stdout

This gives the operator a clean single-command check for pre-send validation.

### 4. Remove `status show --all` from operator skill

Update `fab/.kit/skills/fab-operator1.md` to replace all `fab status show --all` references with `fab pane-map`. The operator's orientation, state re-derivation, and UC5 dashboard all switch to using pane-map as the sole observation mechanism.

The `--all` flag itself stays in the CLI (it has legitimate non-operator use for scripting), but the operator skill no longer references it.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update operator observation section — pane-map session scoping, tab column, runtime is-idle, remove status show --all references
- `fab-workflow/kit-architecture`: (modify) Update pane-map column list (add Tab), add `runtime is-idle` to subcommand list, note send-keys session scoping

## Impact

- **Go binary** (`src/go/fab/`): `panemap.go` (session scoping + tab column), `runtime.go` (add is-idle), `sendkeys.go` (session scoping), `main.go` (register is-idle subcommand)
- **Operator skill** (`fab/.kit/skills/fab-operator1.md`): Replace `status show --all` with `pane-map`
- **Operator spec** (`docs/specs/skills/SPEC-fab-operator1.md`): Update references
- **Specs/memory**: Update docs to reflect new behavior
- **Go parity tests**: Update `src/go/fab/test/parity/`

## Open Questions

- Should `send-keys` also display which tab it's sending to (e.g., "Sending to %3 (tab: alpha)")? Would help operator confirm spatial targeting.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `-s` not `-a` for tmux pane listing | Discussed — operator should only see panes in its own session, not all sessions on the server | S:95 R:90 A:90 D:95 |
| 2 | Certain | Add Tab column using `#{window_name}` | Discussed — operator needs spatial orientation beyond pane IDs | S:90 R:90 A:85 D:90 |
| 3 | Certain | `is-idle` prints `idle {duration}` / `active` / `unknown` | Discussed — operator needs clean single-command check for pre-send validation | S:85 R:85 A:85 D:85 |
| 4 | Certain | Replace `status show --all` with `pane-map` in operator skill only | Discussed — pane-map is strictly more informative; `--all` flag stays in CLI for other consumers | S:90 R:90 A:90 D:90 |
| 5 | Confident | Apply same `-s` scoping to `send-keys` pane discovery | Consistency with pane-map; sending to panes in other sessions would be surprising | S:80 R:80 A:85 D:85 |
| 6 | Confident | Tab column placed between Pane and Worktree | Natural reading order: pane ID → tab name → worktree path → change → stage → agent | S:75 R:90 A:80 D:75 |
| 7 | Confident | Go-only — no Rust binary changes | Go is the distributed binary; Rust is local-dev only | S:80 R:85 A:80 D:85 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
