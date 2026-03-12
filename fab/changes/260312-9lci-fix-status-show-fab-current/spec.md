# Spec: Remove fab status show and fix stale fab/current references

**Change**: 260312-9lci-fix-status-show-fab-current
**Created**: 2026-03-12
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Modifying any other `fab status` subcommands — they are change-scoped and unaffected
- Removing historical changelog references to `fab/current` or `fab status show` — those document what happened at the time
- Replacing `fab status show` with a new command — its use cases are fully covered by `fab pane-map`, `wt list`, and `fab change list`

## CLI: Remove `fab status show` subcommand

### Requirement: Remove show subcommand registration

The `statusCmd()` function in `src/go/fab/cmd/fab/status.go` SHALL NOT register `statusShowCmd()`. The `statusShowCmd` function, `worktreeInfo` struct, `listWorktrees`, `resolveWorktreeFabState`, `findWorktreeByName`, `currentWorktree`, `formatWorktreeHuman`, and `formatWorktreesHuman` functions SHALL be deleted.

#### Scenario: show subcommand no longer exists
- **GIVEN** the fab binary is compiled after this change
- **WHEN** a user runs `fab status show`
- **THEN** Cobra returns an "unknown command" error

#### Scenario: other status subcommands still work
- **GIVEN** the fab binary is compiled after this change
- **WHEN** a user runs `fab status finish 9lci intake`
- **THEN** the command executes normally (all other subcommands are unaffected)

### Requirement: Remove unused imports

After removing the show-related code, any Go imports that become unused (e.g., `encoding/json`, `os/exec`) SHALL be removed. The `os`, `path/filepath`, `strings`, `fmt` imports shared with other functions SHALL be retained only if still referenced.

#### Scenario: clean compilation
- **GIVEN** show-related code and unused imports are removed
- **WHEN** `go build` runs
- **THEN** compilation succeeds with no errors or warnings

## Documentation: Remove `fab status show` references

### Requirement: Update `_scripts.md` command table

The `fab status show` row SHALL be removed from the Command Reference table in `fab/.kit/skills/_scripts.md`. The `show` entry in the Key subcommands table under `fab status` SHALL also be removed if present.

#### Scenario: _scripts.md no longer references show
- **GIVEN** the updated `_scripts.md`
- **WHEN** searching for "status show"
- **THEN** no matches are found

### Requirement: Update `kit-architecture.md`

The `fab status show` entry SHALL be removed from any subcommand lists in `docs/memory/fab-workflow/kit-architecture.md`. The description text referencing show as a feature SHALL be removed. Historical changelog entries that mention `fab status show` SHALL be left unchanged.

#### Scenario: kit-architecture.md subcommand list updated
- **GIVEN** the updated `kit-architecture.md`
- **WHEN** searching for "status show" outside of changelog entries
- **THEN** no matches are found

### Requirement: Update `execution-skills.md`

References to `fab status show --all` as a fallback in operator skill descriptions SHALL be replaced with `wt list` + `fab change list`. This applies to the "Orientation on Start" and "State Re-derivation" sections.

#### Scenario: execution-skills.md fallback updated
- **GIVEN** the updated `execution-skills.md`
- **WHEN** reading the operator orientation and state re-derivation sections
- **THEN** the outside-tmux fallback references `wt list` + `fab change list` instead of `fab status show --all`

## Documentation: Fix stale `fab/current` references

### Requirement: Update README.md

The directory tree in `README.md` SHALL replace `fab/current/` with the `.fab-status.yaml` symlink representation. The activation text SHALL reference `.fab-status.yaml` instead of `fab/current`.

#### Scenario: README directory tree updated
- **GIVEN** the updated `README.md`
- **WHEN** reading the directory tree example
- **THEN** it shows `.fab-status.yaml` (not `fab/current/`)

#### Scenario: README activation text updated
- **GIVEN** the updated `README.md`
- **WHEN** reading the switch/activation instructions
- **THEN** it says "make it active via .fab-status.yaml" or equivalent

### Requirement: Update `_scripts.md` send-keys description

The `fab send-keys` pane resolution description in `_scripts.md` SHALL reference `.fab-status.yaml` instead of `fab/current`.

#### Scenario: _scripts.md send-keys updated
- **GIVEN** the updated `_scripts.md`
- **WHEN** reading the send-keys pane resolution description
- **THEN** it says "read `.fab-status.yaml`" instead of "read `fab/current`"

### Requirement: Update `kit-architecture.md` send-keys description

The `fab send-keys` pane resolution description in `kit-architecture.md` SHALL reference `.fab-status.yaml` instead of `fab/current`.

#### Scenario: kit-architecture.md send-keys updated
- **GIVEN** the updated `kit-architecture.md`
- **WHEN** reading the send-keys pane resolution description
- **THEN** it says "read `.fab-status.yaml`" instead of "read `fab/current`"

## Deprecated Requirements

### `fab status show` subcommand
**Reason**: Fully superseded by `fab pane-map` (tmux observation), `wt list` (worktree listing), and `fab change list` (pipeline state per change). No remaining consumers.
**Migration**: Use `fab pane-map` for tmux-based observation, `wt list` + `fab change list` for non-tmux contexts.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `.fab-status.yaml` is the active pointer mechanism | Confirmed from intake #1 — config, constitution, migration 0.32.0-to-0.34.0 all confirm | S:95 R:95 A:95 D:95 |
| 2 | Certain | Historical changelog references left as-is | Confirmed from intake #2 — modifying history is wrong | S:90 R:90 A:90 D:95 |
| 3 | Certain | `fab status show` has no remaining consumers | Confirmed from intake #3 — operator skills use `wt list` + `fab change list`; no other skill/script references it | S:95 R:85 A:90 D:90 |
| 4 | Certain | All other `fab status` subcommands are unaffected | Confirmed from intake #4 — change-scoped, explicit args, use `internal/resolve` | S:90 R:90 A:90 D:90 |
| 5 | Certain | `fab pane-map` + `wt list` + `fab change list` fully cover use cases | Confirmed from intake #5 — discussion confirmed | S:90 R:85 A:90 D:85 |
| 6 | Certain | No tests exist for `fab status show` | Verified — grep of `*_test.go` finds no show-related test functions | S:95 R:95 A:95 D:95 |
| 7 | Certain | `resolveWorktreeFabState` reads stale `fab/current` not `.fab-status.yaml` | Verified in source — line 549 reads `filepath.Join(fabDir, "current")`, confirming the code is already broken/stale | S:95 R:90 A:95 D:95 |

7 assumptions (7 certain, 0 confident, 0 tentative, 0 unresolved).
