# Spec: Configurable Agent Spawn Command

**Change**: 260320-t13m-configurable-agent-spawn-command
**Created**: 2026-03-20
**Affected memory**: `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing the operator skill markdown files (`fab-operator4.md`, `fab-operator5.md`) — they instruct the agent, not spawn processes directly
- Adding runtime validation of the spawn command (e.g., checking that the binary exists) — the shell will surface errors naturally
- Supporting per-script command overrides — one global command serves all spawn sites

## Configuration: `agent.spawn_command`

### Requirement: Config Key

`config.yaml` SHALL support an optional `agent` section with a `spawn_command` key:

```yaml
agent:
  spawn_command: 'claude --dangerously-skip-permissions --effort max -n "$(basename "$(pwd)")"'
```

The value is a shell command string. It MAY contain shell expansions (e.g., `$(basename "$(pwd)")`) that are evaluated at invocation time, not at config read time.

#### Scenario: Config has agent.spawn_command

- **GIVEN** `fab/project/config.yaml` contains `agent.spawn_command` with a custom value
- **WHEN** a script reads the spawn command
- **THEN** the custom value is returned verbatim (no shell expansion at read time)

#### Scenario: Config missing agent section entirely

- **GIVEN** `fab/project/config.yaml` has no `agent` section
- **WHEN** a script reads the spawn command
- **THEN** the default `claude --dangerously-skip-permissions` is returned
- **AND** the script operates identically to pre-change behavior

#### Scenario: agent section exists but spawn_command is null

- **GIVEN** `fab/project/config.yaml` has `agent:` but `spawn_command` is null or empty
- **WHEN** a script reads the spawn command
- **THEN** the default `claude --dangerously-skip-permissions` is returned

### Requirement: Default Value in Scaffold

`/fab-setup` SHALL generate `config.yaml` with the `agent` section and default `spawn_command` value. The scaffold template (`fab/.kit/scaffold/config.yaml`) SHALL include:

```yaml
agent:
  # Base command for spawning agent sessions in scripts and operators.
  # Shell expansions (e.g., $(basename "$(pwd)")) expand at invocation time.
  spawn_command: 'claude --dangerously-skip-permissions --effort max -n "$(basename "$(pwd)")"'
```

#### Scenario: New project setup

- **GIVEN** a user runs `/fab-setup` on a fresh project
- **WHEN** `config.yaml` is generated
- **THEN** it includes the `agent.spawn_command` key with the default value

## Helper: `lib/spawn.sh`

### Requirement: Shared Helper Function

A shared shell library `fab/.kit/scripts/lib/spawn.sh` SHALL provide a function `fab_spawn_cmd` that reads the spawn command from config with fallback:

```bash
fab_spawn_cmd() {
  local config="$1"
  local cmd
  cmd=$(yq -r '.agent.spawn_command // empty' "$config" 2>/dev/null)
  if [[ -z "$cmd" ]]; then
    cmd="claude --dangerously-skip-permissions"
  fi
  printf '%s' "$cmd"
}
```

Scripts source this helper via `source "$SCRIPT_DIR/lib/spawn.sh"` (or equivalent path resolution) and call `fab_spawn_cmd "$CONFIG_FILE"` to get the command string.

#### Scenario: Helper reads from config

- **GIVEN** `config.yaml` has `agent.spawn_command: 'custom-agent --flag'`
- **WHEN** `fab_spawn_cmd` is called with the config path
- **THEN** it returns `custom-agent --flag`

#### Scenario: Helper falls back on missing key

- **GIVEN** `config.yaml` has no `agent` section
- **WHEN** `fab_spawn_cmd` is called
- **THEN** it returns `claude --dangerously-skip-permissions`

#### Scenario: yq not available

- **GIVEN** `yq` is not in PATH
- **WHEN** `fab_spawn_cmd` is called
- **THEN** it returns the fallback default (stderr from yq is suppressed)

## Script Updates

### Requirement: Operator Scripts

`fab/.kit/scripts/fab-operator4.sh` and `fab/.kit/scripts/fab-operator5.sh` SHALL read the spawn command from config and use it in the tmux invocation.

Before:
```bash
tmux new-window -c "$REPO_ROOT" -n "$TAB_NAME" "claude --dangerously-skip-permissions '/fab-operator4'"
```

After:
```bash
SPAWN_CMD=$(fab_spawn_cmd "$CONFIG_FILE")
tmux new-window -c "$REPO_ROOT" -n "$TAB_NAME" "$SPAWN_CMD '/fab-operator4'"
```

The spawn command is interpolated into the tmux command string. tmux creates a subshell to execute it, so shell expansions in the command (e.g., `$(basename "$(pwd)")`) expand naturally at runtime.

#### Scenario: Operator with default config

- **GIVEN** `config.yaml` has the default `spawn_command`
- **WHEN** `fab-operator4.sh` runs
- **THEN** tmux receives `claude --dangerously-skip-permissions --effort max -n "$(basename "$(pwd)")" '/fab-operator4'`
- **AND** the subshell expands `$(basename "$(pwd)")` to the repo directory name

#### Scenario: Operator with custom agent binary

- **GIVEN** `config.yaml` has `agent.spawn_command: 'my-agent --custom-flag'`
- **WHEN** `fab-operator5.sh` runs
- **THEN** tmux receives `my-agent --custom-flag '/fab-operator5'`

### Requirement: Batch New Backlog Script

`fab/.kit/scripts/batch-fab-new-backlog.sh` SHALL read the spawn command from config and use it in the tmux invocation.

Before:
```bash
tmux new-window -n "fab-$id" -c "$wt_path" \
  "claude --dangerously-skip-permissions '/fab-new ${safe}'"
```

After:
```bash
SPAWN_CMD=$(fab_spawn_cmd "$CONFIG_FILE")
# ...
tmux new-window -n "fab-$id" -c "$wt_path" \
  "$SPAWN_CMD '/fab-new ${safe}'"
```

#### Scenario: Batch new with config

- **GIVEN** `config.yaml` has `agent.spawn_command` set
- **WHEN** `batch-fab-new-backlog.sh` opens a tab for backlog item `a1b2`
- **THEN** the tmux command uses the configured spawn command followed by `'/fab-new ...'`

### Requirement: Batch Switch Script

`fab/.kit/scripts/batch-fab-switch-change.sh` SHALL read the spawn command from config and use it in the tmux invocation.

Before:
```bash
tmux new-window -n "${match}" -c "$wt_path" \
  "claude --dangerously-skip-permissions '/fab-switch ${safe}'"
```

After:
```bash
SPAWN_CMD=$(fab_spawn_cmd "$CONFIG_FILE")
# ...
tmux new-window -n "${match}" -c "$wt_path" \
  "$SPAWN_CMD '/fab-switch ${safe}'"
```

#### Scenario: Batch switch with config

- **GIVEN** `config.yaml` has `agent.spawn_command` set
- **WHEN** `batch-fab-switch-change.sh` opens a tab for a change
- **THEN** the tmux command uses the configured spawn command

### Requirement: Batch Archive Script

`fab/.kit/scripts/batch-fab-archive-change.sh` SHALL read the spawn command from config and use it in the exec invocation.

Before:
```bash
exec claude --dangerously-skip-permissions "$prompt"
```

After:
```bash
SPAWN_CMD=$(fab_spawn_cmd "$CONFIG_FILE")
eval "exec $SPAWN_CMD \"\$prompt\""
```

The `eval` is required because the spawn command is a string that may contain shell expansions, and `exec` needs to expand them. This is safe because the config file is user-controlled (not untrusted input).

#### Scenario: Archive exec with default config

- **GIVEN** `config.yaml` has the default `spawn_command`
- **WHEN** `batch-fab-archive-change.sh` runs
- **THEN** the shell expands the spawn command and execs the agent with the prompt

#### Scenario: Archive exec with no config key

- **GIVEN** `config.yaml` has no `agent` section
- **WHEN** `batch-fab-archive-change.sh` runs
- **THEN** it falls back to `exec claude --dangerously-skip-permissions "$prompt"` (pre-change behavior, no eval needed since fallback has no expansions)

## Documentation: `_cli-external.md`

### Requirement: Update Spawn Example

The tmux `new-window` usage note in `fab/.kit/skills/_cli-external.md` SHALL reference the configurable spawn command instead of the hardcoded string.

Before:
```
- **`new-window`** is used for spawning new agent sessions: `tmux new-window -n "fab-<id>" -c <worktree> "claude --dangerously-skip-permissions '<command>'"`
```

After:
```
- **`new-window`** is used for spawning new agent sessions: `tmux new-window -n "fab-<id>" -c <worktree> "$SPAWN_CMD '<command>'"` where `$SPAWN_CMD` is read from `config.yaml` `agent.spawn_command` (see `lib/spawn.sh`)
```

#### Scenario: Documentation accuracy

- **GIVEN** a user reads the tmux section of `_cli-external.md`
- **WHEN** they look at the `new-window` example
- **THEN** it references the configurable spawn command, not a hardcoded string

## Migration

### Requirement: Migration File

A migration file SHALL be created to add the `agent` section to existing `config.yaml` files that lack it.

The migration SHALL:
1. Check if `config.yaml` already has an `agent` section — if so, skip
2. Append the `agent` section with the default value and comment

#### Scenario: Existing config without agent section

- **GIVEN** an existing `config.yaml` without `agent:`
- **WHEN** the migration runs
- **THEN** it appends the `agent` section with default spawn_command and explanatory comment

#### Scenario: Existing config with agent section

- **GIVEN** an existing `config.yaml` that already has `agent:`
- **WHEN** the migration runs
- **THEN** it skips with a message that the section already exists

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `yq` to read config value | Confirmed from intake #1 — `yq` is established project dependency, scripts already use it | S:90 R:90 A:95 D:95 |
| 2 | Certain | Config key path: `agent.spawn_command` | Confirmed from intake #2 — follows existing config naming patterns (`project.name`, `model_tiers.fast`) | S:85 R:85 A:90 D:90 |
| 3 | Certain | Default: `claude --dangerously-skip-permissions --effort max -n "$(basename "$(pwd)")"` | Confirmed from intake #3 — explicitly specified in backlog item | S:95 R:90 A:95 D:95 |
| 4 | Certain | Fallback to `claude --dangerously-skip-permissions` when key missing | Upgraded from intake Confident #4 — examined all 5 scripts, fallback is trivially reversible and standard config pattern | S:80 R:90 A:85 D:90 |
| 5 | Certain | Migration file for existing configs | Upgraded from intake Confident #5 — confirmed migration conventions from `docs/memory/fab-workflow/migrations.md` and existing migration files | S:75 R:85 A:90 D:85 |
| 6 | Certain | Shell expansions expand at invocation time via tmux subshell or eval | Confirmed from intake #6 — tmux `new-window` spawns a subshell that handles expansion naturally; exec uses eval for the same effect | S:85 R:75 A:90 D:90 |
| 7 | Certain | Shared helper in `fab/.kit/scripts/lib/spawn.sh` | Scripts already source from `lib/` (env-packages.sh exists); DRY principle from constitution | S:80 R:90 A:90 D:85 |
| 8 | Certain | `eval` is acceptable for exec-context spawn | Config file is user-controlled, not untrusted input — same trust model as shell aliases, EDITOR, etc. | S:75 R:70 A:85 D:85 |

8 assumptions (8 certain, 0 confident, 0 tentative, 0 unresolved).
