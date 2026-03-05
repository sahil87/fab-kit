# Intake: Consolidate worktree-init into fab-sync

**Change**: 260218-bx4d-consolidate-worktree-init-into-sync
**Created**: 2026-02-18
**Status**: Draft

## Origin

> User-initiated refactoring discussion. The current worktree bootstrap flow has a circular call chain: `worktree-init.sh` iterates `worktree-init-common/*.sh`, one of which (`2-rerun-sync-workspace.sh`) just calls `fab-sync.sh`. The user proposes collapsing this into a single entry point (`fab-sync.sh`) that orchestrates everything. Decided through iterative discussion covering file structure, naming, path resolution, env var updates, and impact on the external `wt-init` tool.

## Why

The current bootstrap architecture has an unnecessary indirection layer. `worktree-init.sh` exists only to iterate two directories of scripts, but one of those scripts (`2-rerun-sync-workspace.sh`) is just a trampoline back into `fab-sync.sh`. This creates:

1. **A circular call chain** — `worktree-init.sh` → `2-rerun-sync-workspace.sh` → `fab-sync.sh`, where the orchestrator delegates to a script that calls the real work
2. **Two entry points for one concept** — users must understand both `worktree-init.sh` (for new worktrees) and `fab-sync.sh` (for re-syncing), when they're effectively the same operation
3. **Unnecessary file count** — `worktree-init.sh` is a thin loop that adds no logic beyond what `fab-sync.sh` could do directly

Making `fab-sync.sh` the single entry point simplifies the mental model and removes the trampoline.

## What Changes

### 1. New orchestrator: `fab/.kit/scripts/fab-sync.sh`

Rewrite `fab-sync.sh` as a thin orchestrator that:
1. Iterates `fab/.kit/sync/*.sh` in sorted order (kit-level scripts)
2. Then iterates `fab/sync/*.sh` in sorted order (project-specific scripts), if the directory exists

The current 470-line content of `fab-sync.sh` moves into `fab/.kit/sync/2-sync-workspace.sh`.

### 2. Rename `fab/.kit/worktree-init-common/` → `fab/.kit/sync/`

Contents after rename:
- `1-direnv.sh` — unchanged (`direnv allow`, idempotent)
- `2-sync-workspace.sh` — renamed from `2-rerun-sync-workspace.sh`, now contains the full workspace sync logic (current `fab-sync.sh` content) with path resolution adjusted (one directory deeper: `sync_dir → kit_dir` goes up one level instead of `scripts_dir → kit_dir`)

### 3. Rename `fab/worktree-init/` → `fab/sync/`

- Delete `1-claude-settings.sh` and `assets/settings.local.json` — this functionality is already handled by section 8 of the workspace sync script
- Renumber `2-symlink-backlog.sh` → `1-symlink-backlog.sh`
- Add `README.md` scaffolded from `fab/.kit/scaffold/sync-readme.md` (created if not present by `2-sync-workspace.sh`)

### 4. New scaffold: `fab/.kit/scaffold/sync-readme.md`

Content: brief explanation that project-specific sync scripts go in this folder, with naming convention guidance (numbered `*.sh` files, executed in sorted order).

### 5. Delete `fab/.kit/worktree-init.sh`

No longer needed — `fab-sync.sh` is the single entry point.

### 6. Update `fab/.kit/scaffold/envrc`

Change `WORKTREE_INIT_SCRIPT=fab/.kit/worktree-init.sh` → `WORKTREE_INIT_SCRIPT=fab/.kit/scripts/fab-sync.sh`

### 7. Update README.md

Update any references to `worktree-init.sh`, the old directory names, and the bootstrap flow description.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update bootstrap flow, directory structure, and `fab-sync.sh` description to reflect single entry point

## Impact

- **`fab/.kit/scripts/fab-sync.sh`** — rewritten from 470-line sync script to thin orchestrator
- **`fab/.kit/sync/2-sync-workspace.sh`** — new file containing the moved sync logic
- **`fab/.kit/sync/1-direnv.sh`** — moved (unchanged content)
- **`fab/.kit/scaffold/envrc`** — env var path update
- **`fab/.kit/scaffold/sync-readme.md`** — new scaffold file
- **`fab/sync/`** — renamed from `fab/worktree-init/`, files removed/renumbered
- **`fab/.kit/worktree-init.sh`** — deleted
- **`fab/.kit/worktree-init-common/`** — deleted (moved to `fab/.kit/sync/`)
- **`fab/worktree-init/`** — deleted (moved to `fab/sync/`)
- **External**: `wt-init` binary (in prompt-pantry) reads `$WORKTREE_INIT_SCRIPT` — no code change needed there, just the env var value changes

## Open Questions

- None — all decisions resolved during discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `1-direnv.sh` runs on every sync without guard | User explicitly chose "just let it be" — `direnv allow` is idempotent | S:95 R:95 A:90 D:95 |
| 2 | Certain | Delete `1-claude-settings.sh` and `assets/` | User confirmed — section 8 of sync-workspace already handles `settings.local.json` | S:95 R:85 A:95 D:95 |
| 3 | Certain | Keep `WORKTREE_INIT_SCRIPT` env var (don't remove) | User explicitly said not to touch external `wt-init`, work with the env var | S:95 R:90 A:90 D:95 |
| 4 | Certain | Orchestrator runs `.kit/sync/` first, then `fab/sync/` | User confirmed ordering — project scripts depend on workspace being set up first | S:90 R:80 A:90 D:90 |
| 5 | Confident | Path resolution in `2-sync-workspace.sh` needs adjustment | Moving from `scripts/` to `sync/` changes relative depth; `kit_dir` derivation goes from `dirname "$scripts_dir"` to `dirname "$sync_dir"` | S:85 R:90 A:80 D:75 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).
