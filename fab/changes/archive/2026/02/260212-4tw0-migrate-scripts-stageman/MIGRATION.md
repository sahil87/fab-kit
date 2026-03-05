# Migrating to workflow.yaml Schema & Stage Manager

This document explains how to migrate existing scripts and skills to use the canonical `workflow.yaml` schema with the Stage Manager (`stageman.sh`) utility.

## Overview

**Before:** Stage and state knowledge was hardcoded in 7+ locations:
- `fab/.kit/templates/status.yaml` - hardcoded stage list
- `fab/.kit/scripts/fab-status.sh` - hardcoded loops and mappings
- `fab/.kit/scripts/fab-preflight.sh` - hardcoded stage iteration
- `.claude/skills/*/SKILL.md` - documentation and business logic
- `fab/config.yaml` - stage definitions

**After:** Single source of truth:
- `fab/.kit/schemas/workflow.yaml` - canonical definitions
- `fab/.kit/scripts/stageman.sh` - Stage Manager query utility
- All scripts/skills source stageman and query it dynamically

## Benefits

1. **Add/remove stages in one place** - update `workflow.yaml`, all scripts adapt
2. **Consistent state handling** - no more symbol mismatches or forgotten states
3. **Validation** - `validate_status_file()` checks .status.yaml correctness
4. **Self-documenting** - the schema is the documentation

## Migration Examples

### Example 1: fab-status.sh (bash script)

**Before:**
```bash
# Hardcoded stage list
for s in brief spec tasks apply review archive; do
  eval val="\$p_$s"
  if [ "$val" = "active" ]; then
    stage="$s"
    break
  fi
done

# Hardcoded stage numbers
case "${stage:-}" in
  brief)   stage_num=1 ;; spec)    stage_num=2 ;; tasks)   stage_num=3 ;;
  apply)   stage_num=4 ;; review)  stage_num=5 ;; archive) stage_num=6 ;;
  *)       stage_num="?" ;;
esac

# Hardcoded state symbols
symbol() {
  case "$1" in
    done) printf '✓' ;; active)  printf '●' ;; pending) printf '○' ;;
    skipped) printf '—' ;; failed) printf '✗' ;; *)      printf '○' ;;
  esac
}
```

**After:**
```bash
# Source the library
source "$(dirname "$0")/stageman.sh"

# Dynamic stage iteration
for s in $(get_all_stages); do
  eval val="\$p_$s"
  if [ "$val" = "active" ]; then
    stage="$s"
    break
  fi
done

# Dynamic stage numbers
stage_num=$(get_stage_number "$stage")

# Dynamic state symbols
symbol() {
  get_state_symbol "$1"
}
```

### Example 2: fab-preflight.sh (validation)

**Before:**
```bash
# Hardcoded stage list
for s in brief spec tasks apply review archive; do
  eval val="\$p_$s"
  if [ "$val" = "active" ]; then
    stage="$s"
    break
  fi
done
```

**After:**
```bash
source "$(dirname "$0")/stageman.sh"

# Dynamic stage iteration
stage=$(get_current_stage "$status_file")

# Optional: validate the entire status file
if ! validate_status_file "$status_file"; then
  echo "Status file validation failed" >&2
  exit 1
fi
```

### Example 3: status.yaml template generation

**Before:**
```yaml
# Manually maintained template
progress:
  brief: active
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  archive: pending
```

**After (script to generate template):**
```bash
#!/usr/bin/env bash
source "$(dirname "$0")/stageman.sh"

echo "progress:"
for stage in $(get_all_stages); do
  state=$(get_initial_state "$stage")
  echo "  $stage: $state"
done
```

### Example 4: Skills (SKILL.md)

**Before:**
```markdown
The stage progression is:
```
brief → spec → tasks → apply → review → archive
```

Guard: Check the `active` entry to determine whether to allow continuation:
| `active` entry | Action |
|---------------|--------|
| `brief` | Generate `spec.md` → set `brief: done`, `spec: active` |
| `spec` | Generate `tasks.md` + checklist → set `spec: done`, `tasks: active` |
...
```

**After:**
```markdown
The stage progression is read from `fab/.kit/schemas/workflow.yaml`.

Use the preflight script's stage detection:
```bash
stage=$(fab/.kit/scripts/fab-preflight.sh | grep '^stage:' | cut -d' ' -f2)
```

Stage transition rules are defined in `workflow.yaml` under `transitions:`.
```

## Migration Checklist

- [ ] Update `fab-status.sh` to use stageman.sh
- [ ] Update `fab-preflight.sh` to use stageman.sh
- [ ] Generate `status.yaml` template from schema
- [ ] Update skills to reference workflow.yaml instead of hardcoding stages
- [ ] Remove stage hardcoding from `fab-help.sh`
- [ ] Add validation step to `/fab-init` that checks .status.yaml against schema
- [ ] Document workflow.yaml schema in fab/docs/

## Backward Compatibility

The `fab/config.yaml` `stages:` section can remain for now as user-facing configuration (e.g., adding custom stages in the future). The kit's `workflow.yaml` defines the *current* standard workflow, while `config.yaml` allows projects to override or extend it (future enhancement).

For now, `workflow.yaml` is the kit default, and `config.yaml` stages are unused (but preserved for forward compatibility).

## Testing

Test the migration by:

1. Run `stageman.sh` directly to verify all queries work
2. Source it in an existing script and compare output before/after
3. Validate all .status.yaml files in `fab/changes/` using `validate_status_file()`
4. Ensure `/fab-status`, `/fab-continue`, `/fab-apply` all work as expected

## Questions?

- See `workflow.yaml` inline comments for field definitions
- Run `stageman.sh --help` for function reference
- Check `src/stageman/SPEC.md` for complete API documentation
- See `src/stageman/README.md` for development guide
