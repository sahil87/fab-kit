# Change Resolver (_resolve-change.sh)

Bash library for resolving change names from an override argument or `fab/current`. Sourced by `fab-preflight.sh` and `fab-status.sh`.

## Sources of Truth

- **Implementation**: `fab/.kit/scripts/_resolve-change.sh` — main file (distributed with kit)
- **Dev symlink**: `src/resolve-change/_resolve-change.sh` → `../../fab/.kit/scripts/_resolve-change.sh`

## Usage

```bash
source "$(dirname "$0")/_resolve-change.sh"

# Resolve from fab/current
resolve_change "$fab_root" ""
echo "$RESOLVED_CHANGE_NAME"

# Resolve from override (exact or substring)
resolve_change "$fab_root" "puow"
echo "$RESOLVED_CHANGE_NAME"
```

## API Reference

| Function | Input | Output | Exit |
|----------|-------|--------|------|
| `resolve_change <fab_root> [override]` | fab/ directory path + optional name/substring | Sets `RESOLVED_CHANGE_NAME` | 0 success, 1 failure |

### Arguments

- **`fab_root`** (required) — path to the `fab/` directory
- **`override`** (optional) — change name or case-insensitive substring. If empty, reads `fab/current`.

### On Success

Sets `RESOLVED_CHANGE_NAME` to the matched folder name. Returns 0.

### On Failure

Prints a generic diagnostic to stderr (no command suggestions). Returns 1.

| Error | stderr message |
|-------|---------------|
| `fab/changes/` missing | `fab/changes/ not found.` |
| No change folders | `No active changes found.` |
| Multiple matches | `Multiple changes match "X": a, b.` |
| No match | `No change matches "X".` |
| No `fab/current` | `No active change.` |
| Empty `fab/current` | `No active change.` |

Callers add their own context-appropriate guidance (e.g., "Run /fab-new to start one.").

## Requirements

- Bash 4.0+
- GNU coreutils (tr, basename)
- No external dependencies

## Testing

```bash
# Quick smoke test
src/resolve-change/test-simple.sh

# Comprehensive suite
src/resolve-change/test.sh
```

## Changelog

### 1.0.0 (2026-02-14)

- Extracted from `fab-preflight.sh` and `fab-status.sh`
- `resolve_change` function with `RESOLVED_CHANGE_NAME` variable-setting pattern
- Handles: exact match, case-insensitive substring, multiple matches, no match, no fab/current, missing changes dir
