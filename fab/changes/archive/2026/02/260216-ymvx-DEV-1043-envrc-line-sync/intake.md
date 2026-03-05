# Intake: Replace .envrc Symlink with Line-Ensuring Sync

**Change**: 260216-ymvx-DEV-1043-envrc-line-sync
**Created**: 2026-02-16
**Status**: Draft

## Origin

> User requested: "Instead of the dotenvrc file being a symlink, make it a normal file. Change the job of fab-sync.sh to ensure that the lines in our fab/.kit/scaffold/envrc file do exist in the project's .envrc, similar to what we are doing for the .gitignore file. Also create a linear issue for this in the FabKit project (added to the correct milestone) and use that in the change folder name."
>
> Linear issue: [DEV-1043](https://linear.app/weaver-ai/issue/DEV-1043/replace-envrc-symlink-with-line-ensuring-sync-like-gitignore) — created in M4: Operability, Onboarding, Discoverability.

## Why

The current approach creates a symlink from `.envrc` → `fab/.kit/scaffold/envrc`. This prevents projects from adding their own `.envrc` lines alongside fab's entries — the symlink makes `.envrc` read-only from the project's perspective. If a user needs project-specific environment variables or direnv configuration beyond what fab provides, they cannot simply edit `.envrc` without breaking the symlink.

The `.gitignore` handling already solves this exact problem using an additive "ensure lines exist" pattern: read required entries from a scaffold file, append any that are missing. Applying the same pattern to `.envrc` gives projects full ownership of their `.envrc` while guaranteeing fab's required lines are always present.

## What Changes

### `fab/.kit/scripts/fab-sync.sh` — Section 2 (.envrc)

Replace the current symlink logic (lines 101-119) with line-ensuring logic modeled on section 7 (.gitignore, lines 372-401):

1. Read lines from `fab/.kit/scaffold/envrc`
2. Skip comments and empty lines
3. For each line, check if it exists in the project's `.envrc`
4. If missing, append it
5. If `.envrc` doesn't exist yet, create it with the scaffold content

**Migration handling**: If the existing `.envrc` is a symlink (to the scaffold or anything else), resolve its content into a real file first, then apply the line-ensuring logic. This ensures existing projects transition cleanly without losing the scaffold lines.

The current symlink logic:
```bash
# ── 2. .envrc ─────────────────────────────────────────────────────
envrc_link="$repo_root/.envrc"
envrc_target="fab/.kit/scaffold/envrc"

if [ -L "$envrc_link" ] && [ -e "$envrc_link" ]; then
  echo ".envrc: OK (symlink)"
elif [ -L "$envrc_link" ]; then
  rm "$envrc_link"
  ln -s "$envrc_target" "$envrc_link"
  echo ".envrc: repaired broken symlink → $envrc_target"
elif [ -e "$envrc_link" ]; then
  rm "$envrc_link"
  ln -s "$envrc_target" "$envrc_link"
  echo ".envrc: replaced file with symlink → $envrc_target"
else
  ln -s "$envrc_target" "$envrc_link"
  echo ".envrc: created symlink → $envrc_target"
fi
```

Will be replaced with line-ensuring logic similar to:
```bash
# ── 2. .envrc ─────────────────────────────────────────────────────
envrc_file="$repo_root/.envrc"
envrc_entries="$kit_dir/scaffold/envrc"

if [ -f "$envrc_entries" ]; then
  # Migrate: if .envrc is a symlink, replace with real file
  if [ -L "$envrc_file" ]; then
    resolved="$(cat "$envrc_file" 2>/dev/null || true)"
    rm "$envrc_file"
    if [ -n "$resolved" ]; then
      printf '%s\n' "$resolved" > "$envrc_file"
      echo ".envrc: migrated from symlink to file"
    fi
  fi

  envrc_existed=false
  [ -f "$envrc_file" ] && envrc_existed=true
  added=()

  while IFS= read -r entry || [ -n "$entry" ]; do
    [[ -z "$entry" || "$entry" == \#* ]] && continue
    if [ ! -f "$envrc_file" ]; then
      echo "$entry" > "$envrc_file"
      added+=("$entry")
    elif ! grep -qxF "$entry" "$envrc_file"; then
      echo "" >> "$envrc_file"
      echo "$entry" >> "$envrc_file"
      added+=("$entry")
    fi
  done < "$envrc_entries"

  if [ ${#added[@]} -gt 0 ]; then
    if [ "$envrc_existed" = false ]; then
      echo "Created: .envrc (added ${added[*]})"
    else
      echo "Updated: .envrc (added ${added[*]})"
    fi
  else
    echo ".envrc: OK"
  fi
fi
```

### `fab/.kit/scaffold/envrc` — No changes

The file already contains the lines to ensure:
```
export IDEAS_FILE=fab/backlog.md
export WORKTREE_INIT_SCRIPT=fab/.kit/worktree-init.sh
PATH_add fab/.kit/scripts
```

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update .envrc handling description from symlink to line-ensuring
- `fab-workflow/distribution`: (modify) Update sync behavior documentation

## Impact

- **`fab/.kit/scripts/fab-sync.sh`** — Section 2 rewritten; all other sections unchanged
- **Existing projects** — `.envrc` symlinks will be migrated to real files on next `fab-sync.sh` run; `direnv allow` may need to be re-run after migration
- **New projects** — `.envrc` created as a normal file from the start
- **No breaking changes** — the resulting `.envrc` content is identical; only the file type changes

## Open Questions

- None — the approach is well-defined by the existing .gitignore pattern.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use the same line-ensuring pattern as .gitignore | User explicitly requested this approach; section 7 provides the exact template | S:95 R:90 A:95 D:95 |
| 2 | Certain | Skip comments and empty lines from scaffold file | Consistent with .gitignore handling; comments are not functional direnv lines | S:85 R:95 A:90 D:90 |
| 3 | Confident | Migrate existing symlinks by resolving content to a real file | Prevents data loss during transition; the scaffold content is preserved | S:75 R:80 A:85 D:85 |
| 4 | Confident | Users may need to re-run `direnv allow` after migration | direnv tracks file identity; switching from symlink to real file may invalidate the allow | S:70 R:90 A:80 D:85 |

4 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).
