# Intake: Add Rename Subcommand to changeman.sh

**Change**: 260216-u6d5-DEV-1039-add-changeman-rename
**Created**: 2026-02-16
**Status**: Draft

## Origin

> Add a rename option to changeman.sh, also add relevant test cases

Linear ticket DEV-1039 in M4: Operability, Onboarding, Discoverability. Related to DEV-1022 (create changeman.sh) and DEV-1023 (batch script rename). The request came as a one-shot natural language description.

## Why

`changeman.sh` currently only supports the `new` subcommand. Once a change folder is created, there is no programmatic way to rename it. If a slug is wrong, the scope evolves, or a typo is discovered, the user must manually:

1. Rename the folder on disk
2. Update the `name` field in `.status.yaml`
3. Update `fab/current` if this is the active change
4. Rename the git branch if one was created

This is error-prone (easy to forget a step) and tedious. A `rename` subcommand makes this a single atomic operation with proper validation.

## What Changes

### 1. New `rename` subcommand in `changeman.sh`

Add `changeman.sh rename --folder <current-folder> --slug <new-slug>` to `fab/.kit/scripts/lib/changeman.sh`.

**Behavior:**
1. Parse `--folder` (required) — the current change folder name (full name, e.g., `260216-u6d5-DEV-1039-add-changeman-rename`)
2. Parse `--slug` (required) — the new slug to replace the current slug portion
3. Validate the new slug using the same regex as `new`: `^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`
4. Verify the source folder exists under `fab/changes/`
5. Extract the `{YYMMDD}-{XXXX}` prefix from the current folder name (first two segments)
6. Construct new folder name: `{YYMMDD}-{XXXX}-{new-slug}`
7. Verify the destination folder does not already exist (collision check)
8. Rename the folder via `mv`
9. Update `.status.yaml` `name` field to reflect the new folder name (via `sed` or `yq`)
10. Check if `fab/current` points to the old folder name — if so, update it to the new name
11. Log the rename via `stageman log-command`
12. Print the new folder name to stdout

**Example:**
```bash
changeman.sh rename --folder 260216-u6d5-DEV-1039-add-changeman-rename --slug DEV-1039-changeman-rename-cmd
# Output: 260216-u6d5-DEV-1039-changeman-rename-cmd
```

**Error cases:**
- Missing `--folder` or `--slug` → error with usage hint
- Source folder doesn't exist → `ERROR: Change folder '{name}' not found`
- Destination folder already exists → `ERROR: Folder '{new-name}' already exists`
- Invalid slug format → same error as `new`
- New name same as old name → `ERROR: New name is the same as current name`

### 2. Test cases in `src/lib/changeman/test.bats`

Add a new test section `# -- rename: ... --` covering:

- **Happy path**: rename changes the folder, updates `.status.yaml` name field, outputs new name
- **Active change update**: when `fab/current` contains the old name, it gets updated
- **fab/current not affected**: when `fab/current` points to a different change, it's left alone
- **Slug validation**: rejects invalid slugs (same rules as `new`)
- **Missing source folder**: errors when folder doesn't exist
- **Destination collision**: errors when target folder name already exists
- **Same name**: errors when new slug produces the same folder name
- **Missing flags**: errors for missing `--folder` and `--slug`
- **Stageman logging**: verifies `log-command` is called with rename args

### 3. Update help text and spec

- Update `show_help()` in `changeman.sh` to document the `rename` subcommand
- Update `src/lib/changeman/SPEC-changeman.md` with the new API reference

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) Document the rename capability as part of change lifecycle operations
- `fab-workflow/kit-architecture`: (modify) Update changeman.sh description to list both `new` and `rename` subcommands

## Impact

- **`fab/.kit/scripts/lib/changeman.sh`** — primary implementation file
- **`src/lib/changeman/test.bats`** — test suite (add rename section)
- **`src/lib/changeman/SPEC-changeman.md`** — API documentation
- **`fab/current`** — read/written during rename if active change is renamed
- **`.status.yaml`** — `name` field updated during rename

## Open Questions

None — the scope is well-defined by the existing `new` subcommand patterns and the folder naming convention.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Slug validation uses same regex as `new` | Reuse existing validation — `^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$` | S:90 R:95 A:95 D:95 |
| 2 | Certain | Rename preserves date-ID prefix, only changes slug | The date and 4-char ID are immutable identifiers per naming convention in config.yaml | S:85 R:90 A:95 D:90 |
| 3 | Certain | Update `.status.yaml` name field on rename | The name field must match the folder name — this is a structural invariant | S:90 R:95 A:95 D:95 |
| 4 | Confident | Update `fab/current` if it references the renamed change | Active change pointer must stay valid; not updating would break preflight | S:80 R:85 A:90 D:85 |
| 5 | Confident | No git branch rename in this scope | Git branch rename is complex (local + remote), and the rename subcommand focuses on fab internals; git branch rename can be added later as an enhancement | S:60 R:80 A:70 D:65 |
| 6 | Confident | Use `--folder` flag (not positional) for the source folder | Consistent with `new`'s flag-based argument style (`--slug`, `--change-id`) | S:70 R:90 A:85 D:75 |

6 assumptions (2 certain, 4 confident, 0 tentative, 0 unresolved).
