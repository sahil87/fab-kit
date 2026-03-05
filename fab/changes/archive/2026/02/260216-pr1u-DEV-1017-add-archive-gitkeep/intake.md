# Intake: Add .gitkeep to fab/changes/archive/

**Change**: 260216-pr1u-DEV-1017-add-archive-gitkeep
**Created**: 2026-02-16
**Status**: Draft

## Origin

> Fix bug pr1u (DEV-1017): Add .gitkeep to fab/changes/archive/ directory in fab-sync.sh so Git tracks the empty archive folder on fresh projects

Backlog item `[pr1u]` from `fab/backlog.md`, linked to Linear issue DEV-1017. Confirmed the bug still exists — `fab/changes/archive/` has no `.gitkeep` and `fab-sync.sh` doesn't create it.

## Why

1. **Problem**: `fab-sync.sh` creates `fab/changes/` and adds `fab/changes/.gitkeep`, but does not create `fab/changes/archive/` or add a `.gitkeep` to it. On a fresh project with no archived changes, the archive directory either doesn't exist or (if created by other means) won't be tracked by Git because it's empty.

2. **Consequence**: New clones or fresh setups are missing the archive directory. The first `/fab-archive` invocation may need to create it, and the directory structure doesn't match what `/fab-setup` documents (which says "create `fab/changes/`, `fab/changes/archive/`, and `fab/changes/.gitkeep`").

3. **Approach**: Add `fab/changes/archive/` creation and `fab/changes/archive/.gitkeep` to the directory-creation section of `fab-sync.sh`, matching the pattern already used for `fab/changes/.gitkeep`.

## What Changes

### fab-sync.sh — Directory creation section (lines 72-83)

Currently the script creates three directories (`fab/changes`, `docs/memory`, `docs/specs`) and only adds `.gitkeep` to `fab/changes/`. The fix:

1. Add `"$fab_dir/changes/archive"` to the `for dir in ...` loop on line 74, so the archive directory is created alongside the others
2. Add a `.gitkeep` touch for `fab/changes/archive/.gitkeep` after the existing `fab/changes/.gitkeep` block (lines 81-83)

After the fix, the directory creation section should look like:

```bash
for dir in "$fab_dir/changes" "$fab_dir/changes/archive" "$docs_dir/memory" "$docs_dir/specs"; do
  if [ ! -d "$dir" ]; then
    mkdir -p "$dir"
    echo "Created: ${dir#"$repo_root"/}"
  fi
done

if [ ! -f "$fab_dir/changes/.gitkeep" ]; then
  touch "$fab_dir/changes/.gitkeep"
fi

if [ ! -f "$fab_dir/changes/archive/.gitkeep" ]; then
  touch "$fab_dir/changes/archive/.gitkeep"
fi
```

### fab-sync SPEC — Update directory creation docs

Update `src/lib/fab-sync/SPEC-fab-sync.md` section "1. Directory Creation" to mention `fab/changes/archive/` and its `.gitkeep`.

### fab-sync test — Add test case

Add a bats test in `src/lib/fab-sync/test.bats` for `fab/changes/archive/.gitkeep`, following the pattern of the existing `"creates fab/changes/.gitkeep"` test (lines 118-122).

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update directory tree to show `fab/changes/archive/.gitkeep`

## Impact

- `fab/.kit/scripts/fab-sync.sh` — primary fix location
- `src/lib/fab-sync/SPEC-fab-sync.md` — documentation update
- `src/lib/fab-sync/test.bats` — new test case
- No behavior change for projects that already have archived changes (archive/ already exists with content)

## Open Questions

None — the fix is straightforward and mirrors existing patterns.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add archive dir to existing `for` loop | Mirrors existing directory creation pattern at lines 74-79 | S:90 R:95 A:95 D:95 |
| 2 | Certain | Add `.gitkeep` with same conditional pattern | Identical pattern to `fab/changes/.gitkeep` at lines 81-83 | S:90 R:95 A:95 D:95 |
| 3 | Certain | Update SPEC and test to match | Existing SPEC and test file cover the current `.gitkeep`; extend them | S:85 R:95 A:90 D:95 |

3 assumptions (3 certain, 0 confident, 0 tentative, 0 unresolved).
