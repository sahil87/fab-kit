# Intake: Standardize archiveman.sh Dispatcher Integration

**Change**: 260305-jv4y-standardize-archiveman-dispatcher
**Created**: 2026-03-05
**Status**: Draft

## Origin

> Discovered while investigating a Copilot reviewer bug in the `/fab-ff` pipeline. Examination of the `fab/.kit/bin/fab` shell dispatcher revealed that the `archive` command hardcodes `"archive"` as the first positional argument before forwarding to `archiveman.sh`. This means `fab archive restore <change>` and `fab archive list` are broken in the shell backend — the `restore`/`list` subcommand gets passed as a change name argument to the `archive` subcommand instead. The Go backend already handles this correctly with Cobra subcommands.

## Why

The shell dispatcher's `archive` case does:
```sh
archive)  shift; exec bash "$LIB_DIR/archiveman.sh" "archive" "$@" ;;
```

This hardcoded `"archive"` means:
1. `fab archive restore foo` → `archiveman.sh archive restore foo` — `restore` is treated as a change name, not a subcommand
2. `fab archive list` → `archiveman.sh archive list` — `list` is treated as a change name
3. Only `fab archive <change> --description "..."` works correctly

The Go backend already has the correct interface (`archive`, `archive restore`, `archive list` as separate Cobra commands). The shell backend is the only one broken. Skills (`fab-archive.md`) reference `fab archive restore` and `fab archive list` which silently fail in the shell fallback path.

## What Changes

### 1. archiveman.sh: Default to `archive` when $1 isn't a known subcommand

The current dispatch in archiveman.sh (line ~397):
```bash
case "${1:-}" in
  --help|-h) show_help ;;
  archive)   shift; cmd_archive "$@" ;;
  restore)   shift; cmd_restore "$@" ;;
  list)      shift; cmd_list "$@" ;;
  "")        echo "ERROR: No subcommand provided." >&2; exit 1 ;;
  *)         echo "ERROR: Unknown subcommand '$1'." >&2; exit 1 ;;
esac
```

Change the `""` and `*` cases: when $1 is not a known subcommand, treat all args as `cmd_archive "$@"` (no shift — $1 is the change name). The empty case remains an error.

After:
```bash
case "${1:-}" in
  --help|-h) show_help ;;
  archive)   shift; cmd_archive "$@" ;;
  restore)   shift; cmd_restore "$@" ;;
  list)      shift; cmd_list "$@" ;;
  "")        echo "ERROR: No subcommand provided." >&2; exit 1 ;;
  *)         cmd_archive "$@" ;;
esac
```

### 2. fab dispatcher: Remove hardcoded "archive"

Change:
```sh
archive)  shift; exec bash "$LIB_DIR/archiveman.sh" "archive" "$@" ;;
```
To:
```sh
archive)  shift; exec bash "$LIB_DIR/archiveman.sh" "$@" ;;
```

This standardizes the `archive` entry to match all other dispatcher entries (plain pass-through).

### 3. Update parity tests

The bash side of `src/go/fab/test/parity/archive_test.go` calls `archiveman.sh "archive" ...` explicitly. Update to call via the dispatcher or without the explicit `archive` subcommand, matching the new behavior.

### 4. Update bats tests

`src/lib/archiveman/test.bats` invokes `archiveman.sh` directly with explicit `archive` subcommand. Verify tests still pass (they should — the explicit `archive` subcommand still works, it's just no longer required).

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update dispatcher documentation to note standardized pass-through for archive

## Impact

- `fab/.kit/bin/fab` — dispatcher case statement (1 line change)
- `fab/.kit/scripts/lib/archiveman.sh` — default case in subcommand dispatch (~2 lines)
- `src/go/fab/test/parity/archive_test.go` — bash invocation lines
- `src/lib/archiveman/test.bats` — no change needed (explicit subcommand still works)
- `fab/.kit/skills/fab-archive.md` — no change needed (already uses `fab archive` correctly)
- `fab/.kit/skills/_scripts.md` — no change needed (already documents `fab archive`)

## Open Questions

None — the approach was discussed and agreed upon (Option 3: default-to-archive fallback).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Default to `archive` when $1 isn't a known subcommand | Discussed — user chose Option 3 over top-level commands or double-archive | S:95 R:90 A:95 D:95 |
| 2 | Certain | Keep explicit `archive` subcommand working | Backwards compatibility — existing callers that pass `archive` explicitly still work | S:90 R:95 A:90 D:95 |
| 3 | Certain | Go backend already correct, no changes needed | Verified — Cobra subcommand structure already matches desired interface | S:95 R:95 A:95 D:95 |
| 4 | Confident | Empty args case stays an error | No reason to default to help or list — archiveman needs a change name for archive | S:80 R:90 A:85 D:80 |
| 5 | Certain | Parity tests need bash-side update | Parity tests call `archiveman.sh "archive"` directly — should test through dispatcher or match new convention | S:90 R:85 A:90 D:90 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).
