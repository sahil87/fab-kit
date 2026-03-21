# Intake: Configurable Agent Spawn Command

**Change**: 260320-t13m-configurable-agent-spawn-command
**Created**: 2026-03-20
**Status**: Draft

## Origin

> [t13m] 2026-03-18: Configurable agent spawn command in config.yaml — centralize the agent binary/flags so all spawn scripts and operator read from one place. Default: `claude --dangerously-skip-permissions --effort max -n "$(basename "$(pwd)")"`. Currently hardcoded in 4+ locations.

Backlog item. One-shot description with clear intent and a specific default value.

## Why

The agent spawn command (`claude --dangerously-skip-permissions`) is hardcoded in 5+ shell scripts across `fab/.kit/scripts/`. Each script independently constructs its own invocation string, creating several problems:

1. **Consistency risk**: When the spawn flags need to change (e.g., adding `--effort max`, changing the binary name, adding environment variables), every script must be updated independently. Missing one creates silent divergence.
2. **Portability friction**: Users who run a different agent binary (e.g., a wrapper script, a different path, or additional flags for their environment) have no single place to configure it — they must patch multiple scripts.
3. **Operator/script coupling**: The operator skills (`fab-operator4.sh`, `fab-operator5.sh`) and batch scripts (`batch-fab-new-backlog.sh`, `batch-fab-switch-change.sh`, `batch-fab-archive-change.sh`) all independently hardcode the same pattern, violating DRY.

If not addressed, every new script or operator version will copy-paste the spawn command, compounding the maintenance burden.

## What Changes

### New config key in `config.yaml`

Add an `agent` section to `fab/project/config.yaml` with a `spawn_command` key:

```yaml
agent:
  spawn_command: 'claude --dangerously-skip-permissions --effort max -n "$(basename "$(pwd)")"'
```

This is the **template** for spawning agent processes. Scripts append their specific arguments (skill invocations, prompts) to this base command.

The default value shown above matches the current hardcoded behavior, with the addition of `--effort max` and `-n` (name flag using the worktree basename) as specified in the backlog item.

### Helper function or read mechanism in scripts

Each script currently constructs its own spawn string. After this change, scripts SHALL read the `spawn_command` from `config.yaml` using `yq` (already a project dependency per the constitution — single-binary utility). A shared shell function or inline snippet extracts the value:

```bash
SPAWN_CMD=$(yq -r '.agent.spawn_command' fab/project/config.yaml)
```

Scripts then use `$SPAWN_CMD` wherever they currently hardcode the claude invocation.

### Scripts to update

The following files contain hardcoded spawn commands that SHALL be refactored:

1. **`fab/.kit/scripts/fab-operator4.sh:26`** — `claude --dangerously-skip-permissions '/fab-operator4'`
2. **`fab/.kit/scripts/fab-operator5.sh:26`** — `claude --dangerously-skip-permissions '/fab-operator5'`
3. **`fab/.kit/scripts/batch-fab-new-backlog.sh:144`** — `claude --dangerously-skip-permissions '/fab-new ${safe}'`
4. **`fab/.kit/scripts/batch-fab-switch-change.sh:133`** — `claude --dangerously-skip-permissions '/fab-switch ${safe}'`
5. **`fab/.kit/scripts/batch-fab-archive-change.sh:135`** — `exec claude --dangerously-skip-permissions "$prompt"`

### Documentation updates

- **`fab/.kit/skills/_cli-external.md:59`** — Update the tmux `new-window` example to reference `spawn_command` from config
- **`docs/memory/fab-workflow/kit-architecture.md`** — Update script descriptions to note configurable spawn command
- **`docs/memory/fab-workflow/execution-skills.md`** — Update operator spawn documentation

### Operator skill references

The operator skill files (`fab-operator4.md`, `fab-operator5.md`) instruct the agent to spawn subagents. If they reference a specific command string, those references SHALL also be updated to note that the spawn command comes from config. However, operator skills typically delegate to the shell scripts above, so the skill markdown may not need direct changes — only the scripts they invoke.

## Affected Memory

- `fab-workflow/configuration`: (modify) Add `agent.spawn_command` key documentation
- `fab-workflow/kit-architecture`: (modify) Update script descriptions to note configurable spawn command
- `fab-workflow/execution-skills`: (modify) Update operator spawn documentation

## Impact

- **Config schema**: New `agent` top-level key in `config.yaml`. Existing configs without this key need a sensible default (scripts fall back to the hardcoded value if the key is missing).
- **Shell scripts**: 5 scripts in `fab/.kit/scripts/` modified to read from config.
- **`yq` dependency**: Already established in the project (constitution allows single-binary utilities). Scripts already use `yq` for YAML parsing.
- **Backwards compatibility**: Existing projects without the `agent` section in `config.yaml` MUST still work — scripts fall back to the current hardcoded default.
- **Migration**: A migration file SHALL be created in `fab/.kit/migrations/` to add the `agent` section to existing `config.yaml` files.

## Open Questions

- None — the backlog description is specific about the key, the default value, and the scope.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `yq` to read config value in shell scripts | `yq` is an established project dependency; constitution permits single-binary utilities; scripts already use it | S:90 R:90 A:95 D:95 |
| 2 | Certain | Config key path: `agent.spawn_command` | Backlog specifies "agent spawn command in config.yaml"; `agent.spawn_command` follows existing config naming patterns | S:85 R:85 A:90 D:90 |
| 3 | Certain | Default value: `claude --dangerously-skip-permissions --effort max -n "$(basename "$(pwd)")"` | Explicitly specified in backlog item | S:95 R:90 A:95 D:95 |
| 4 | Confident | Fallback behavior when config key is missing | Scripts should fall back to current hardcoded default for backwards compatibility; standard config pattern in the project | S:70 R:85 A:80 D:85 |
| 5 | Confident | Migration file needed in `fab/.kit/migrations/` | Project context mandates migrations for config restructuring; this adds a new config section | S:65 R:80 A:85 D:80 |
| 6 | Certain | Shell variable expansion in spawn_command happens at invocation time | The `$(basename "$(pwd)")` in the default must expand when the script runs, not when config is read — store as single-quoted string in YAML | S:85 R:70 A:90 D:90 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
