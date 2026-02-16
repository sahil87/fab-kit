# Change Manager (changeman)

CLI utility for change lifecycle operations. Currently supports the `new` subcommand for creating change directories with initialized `.status.yaml`.

## Sources of Truth

- **Implementation**: `fab/.kit/scripts/lib/changeman.sh` — main file (distributed with kit)
- **Dev symlink**: `src/lib/changeman/changeman.sh` → `../../../fab/.kit/scripts/lib/changeman.sh`
- **Architecture docs**: `docs/memory/fab-workflow/kit-architecture.md` — script description

## Usage

```bash
CHANGEMAN="path/to/changeman.sh"

# Create a new change
"$CHANGEMAN" new --slug add-oauth
"$CHANGEMAN" new --slug DEV-988-add-oauth --change-id a7k2 --log-args "Add OAuth"

# Help
"$CHANGEMAN" --help
```

## API Reference

### Subcommands

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `new --slug <slug> [--change-id <4char>] [--log-args <desc>]` | slug (required), optional id and log args | folder name to stdout | 0 success, 1 error |
| `--help` | — | usage text | 0 |

### `new` Subcommand

**Arguments:**
- `--slug <slug>` (required) — Folder name suffix. Alphanumeric + hyphens, no leading/trailing hyphens.
- `--change-id <4char>` (optional) — Explicit 4-char lowercase alphanumeric ID. Random if omitted.
- `--log-args <description>` (optional) — Description logged via `stageman log-command`.

**Behavior:**
1. Validates slug format (`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
2. Validates change-id if provided (`^[a-z0-9]{4}$`)
3. Generates date prefix (`YYMMDD`)
4. Generates or uses provided 4-char ID with collision detection
5. Constructs folder name: `{YYMMDD}-{XXXX}-{slug}`
6. Creates directory via `mkdir` (not `-p` — parent guaranteed by sync-workspace.sh)
7. Detects `created_by`: `gh api user` → `git config user.name` → `"unknown"`
8. Initializes `.status.yaml` from template via `sed`
9. Calls `stageman set-state <file> intake active fab-new`
10. Optionally calls `stageman log-command` if `--log-args` provided

**Collision detection:**
- Provided ID collision → fatal error with existing folder name
- Random ID collision → retry (up to 10 attempts)

**Error cases:**
- Missing `--slug` → error
- Invalid slug format → error
- Invalid change-id format → error
- Unknown flags → error
- No subcommand → error
- Unknown subcommand → error

## Requirements

- Bash 4.0+
- `fab/.kit/scripts/lib/stageman.sh` on the same relative path
- `fab/.kit/templates/status.yaml` for template
- `fab/changes/` directory must exist
- GNU coreutils (sed, head, mkdir, date)
- Optional: `gh` CLI, `git` (for `detect_created_by`)

## Testing

```bash
# Run bats test suite
bats src/lib/changeman/test.bats
```
