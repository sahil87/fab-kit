# Change Manager (changeman)

CLI utility for change lifecycle operations. Supports the `new` subcommand for creating change directories and the `rename` subcommand for renaming an existing change folder's slug.

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

# Rename an existing change
"$CHANGEMAN" rename --folder 260216-u6d5-old-slug --slug new-slug

# Help
"$CHANGEMAN" --help
```

## API Reference

### Subcommands

| Subcommand | Input | Output | Exit |
|------------|-------|--------|------|
| `new --slug <slug> [--change-id <4char>] [--log-args <desc>]` | slug (required), optional id and log args | folder name to stdout | 0 success, 1 error |
| `rename --folder <current-folder> --slug <new-slug>` | folder (required), slug (required) | new folder name to stdout | 0 success, 1 error |
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

### `rename` Subcommand

**Arguments:**
- `--folder <current-folder>` (required) — Full current change folder name (e.g., `260216-u6d5-old-slug`).
- `--slug <new-slug>` (required) — New slug to replace the current slug portion. Same validation as `new`.

**Behavior:**
1. Validates slug format (`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
2. Verifies source folder exists under `fab/changes/`
3. Extracts `{YYMMDD}-{XXXX}` prefix (first two hyphen-separated segments)
4. Constructs new folder name: `{prefix}-{new-slug}`
5. Checks new name differs from current name
6. Checks destination folder does not already exist
7. Renames folder via `mv`
8. Updates `.status.yaml` `name` field via `sed`
9. Updates `fab/current` if it points to the old folder name
10. Calls `stageman log-command` with the new change directory

**Error cases:**
- Missing `--folder` → error
- Missing `--slug` → error
- Invalid slug format → error
- Source folder not found → `ERROR: Change folder '...' not found`
- Destination already exists → `ERROR: Folder '...' already exists`
- New name same as current → `ERROR: New name is the same as current name`
- Unknown flags → error

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
