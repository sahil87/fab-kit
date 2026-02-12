# Fab Kit Schemas

Single source of truth for workflow structure, validation rules, and behavior.

## Files

### `workflow.yaml`

Canonical definition of the Fab workflow: stages, states, transitions, and validation rules.

**What it defines:**

1. **States** - All valid progress values (`pending`, `active`, `done`, `skipped`, `failed`)
   - Each state has: ID, display symbol, description, terminal flag
   - Terminal states (`done`, `skipped`) cannot transition without explicit reset

2. **Stages** - The 6-stage workflow pipeline
   - Each stage has: ID, name, artifact, description, requirements, initial state, allowed states, commands
   - Stages execute in sequence with dependency validation
   - Currently: `brief → spec → tasks → apply → review → archive`

3. **Transitions** - Valid state changes for each stage
   - Default rules apply to all stages
   - Stage-specific overrides (e.g., `review` can go to `failed`)
   - Conditions specify when transitions are allowed

4. **Progression** - How to navigate the workflow
   - Current stage detection: first `active` stage, or `archive` if all done
   - Next stage calculation: first `pending` stage with satisfied dependencies
   - Completion check: `archive` is `done`

5. **Validation** - Rules for `.status.yaml` correctness
   - Exactly 0-1 active stages
   - States must be in `allowed_states` for that stage
   - Prerequisites must be satisfied before activation
   - Terminal states require explicit reset

**Schema versioning:** Includes metadata (version, compatibility, last updated)

### Stage Manager (`stageman.sh`)

Query utility for `workflow.yaml`. Source this in scripts to access workflow data.

**Location:** `fab/.kit/scripts/stageman.sh`

**Key functions:**

```bash
# States
get_all_states                    # List all valid states
validate_state "done"             # Check if state is valid
get_state_symbol "active"         # Get display symbol (●)
is_terminal_state "done"          # Check if state is terminal

# Stages
get_all_stages                    # List all stages in order
validate_stage "spec"             # Check if stage exists
get_stage_number "spec"           # Get position (2)
get_stage_name "spec"             # Get display name (Specification)
get_stage_artifact "spec"         # Get generated file (spec.md)
get_allowed_states "review"       # List allowed states for stage
get_initial_state "brief"         # Get default state (active)

# Progression
get_current_stage "path/.status.yaml"  # Detect active stage
get_next_stage "spec"                  # Get next stage (tasks)

# Validation
validate_status_file "path/.status.yaml"  # Full validation with errors
```

**Documentation:** See `src/stageman/` for API specs, tests, and development guide

## Usage

### In Bash Scripts

```bash
#!/usr/bin/env bash
source "$(dirname "$0")/stageman.sh"

# Example: Print stage progression
current=$(get_current_stage "$status_file")
next=$(get_next_stage "$current" || echo "complete")

echo "Current: $current ($(get_stage_number "$current")/6)"
echo "Next: $next"

# Example: Validate a status file
if ! validate_status_file "$status_file"; then
  echo "Invalid status file" >&2
  exit 1
fi

# Example: Display progress with symbols
for stage in $(get_all_stages); do
  state=$(grep "^ *${stage}:" "$status_file" | sed 's/.*: //')
  symbol=$(get_state_symbol "$state")
  echo "  $symbol $stage"
done
```

### In Skills (Claude prompts)

Reference the schema directly:

```markdown
Before generating artifacts, read `fab/.kit/schemas/workflow.yaml` to:
- Determine allowed states for the current stage
- Check if the stage has `auto_checklist: true`
- Verify prerequisites are satisfied
```

Or use bash scripts that source `stageman.sh`:

```markdown
Run `fab/.kit/scripts/fab-preflight.sh` to get validated stage information.
The script uses `stageman.sh` internally.
```

## Design Principles

1. **Single Source of Truth** - One canonical definition, queried by all consumers
2. **Declarative** - Describe *what* the workflow is, not *how* to execute it
3. **Extensible** - Add stages/states/transitions without breaking existing code
4. **Validated** - Schema enforces correctness at runtime
5. **Versionable** - Metadata tracks compatibility and changes

## Migration Path

See [MIGRATION.md](./MIGRATION.md) for:
- Before/after examples of script refactoring
- Backward compatibility notes
- Testing checklist
- Common migration patterns

## Future Enhancements

Potential extensions to the schema:

1. **Custom workflows** - Allow `fab/config.yaml` to override or extend `workflow.yaml`
2. **Conditional stages** - Skip stages based on change attributes (e.g., docs-only changes skip `apply`)
3. **Parallel stages** - Multiple stages active simultaneously for different artifacts
4. **Stage hooks** - Run scripts before/after stage transitions
5. **State metadata** - Attach timestamps, user info, or exit codes to state transitions

## Questions?

- Read `workflow.yaml` inline comments for field-level documentation
- Run `stageman.sh --help` to see all available functions
- Check [MIGRATION.md](./MIGRATION.md) for refactoring examples
- See `src/stageman/` for complete API documentation
