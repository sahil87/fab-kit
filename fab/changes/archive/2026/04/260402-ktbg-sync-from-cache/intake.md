# Intake: Sync From Cache

**Change**: 260402-ktbg-sync-from-cache
**Created**: 2026-04-02
**Status**: Draft

## Origin

> Redesign `fab sync` to read from `~/.fab-kit/` cache instead of `fab/.kit/` in the repo, as a step toward the larger goal of stopping shipping `fab/.kit/` in repos entirely.

Discussion session preceded this intake. Key decisions were made conversationally about the revised step ordering, flag design, hook absorption, and scope boundaries.

## Why

1. **Redundant copy chain**: Skills currently get copied twice — cache to `fab/.kit/`, then `fab/.kit/` to agent dirs. The repo's `fab/.kit/skills/` is a middleman that adds no value.
2. **Consistency with binary dispatch**: `fab-go` already runs from the cache (`~/.fab-kit/versions/{version}/fab-go`). Having `fab sync` still read from the repo copy is inconsistent with the three-binary architecture established in `260402-3ac3-three-binary-architecture`.
3. **Path to repo-less kit**: This change is the prerequisite for removing `fab/.kit/` from repos entirely (a separate follow-up). Without it, every repo carries ~30+ skill files, templates, scaffold, hooks, and migrations that are already available in the system cache.

## What Changes

### Revised `fab sync` pipeline (6 steps)

The entire sync pipeline is rewritten to source from `~/.fab-kit/versions/{version}/kit/` (resolved via `fab_version` from `config.yaml`) instead of `fab/.kit/`.

#### Step 1: Prerequisites check
Validate required tools: `git`, `bash`, `yq` (v4+), `direnv`. Same as today but sourced from cache metadata if applicable.

#### Step 2: Version guard
Ensure the project's `fab_version` (from `fab/project/config.yaml`) is **<= the system-installed `fab-kit` version**. If the project needs a newer version than what's installed, run `fab update` to upgrade the system binaries first. This prevents version skew where the shim is older than what the project expects.

#### Step 3: Ensure cache
Call `EnsureCached(fab_version)` to guarantee the needed version exists at `~/.fab-kit/versions/{version}/`. Downloads from GitHub releases if missing. This absorbs the old `4-get-fab-binary.sh` functionality.

#### Step 4: Workspace scaffolding
All scaffolding reads from `~/.fab-kit/versions/{version}/kit/` instead of `fab/.kit/`:
- **Directory creation**: `fab/changes/`, `fab/changes/archive/`, `docs/memory/`, `docs/specs/` with `.gitkeep`
- **Scaffold tree-walk**: `{cache}/kit/scaffold/` — fragment merging (`.envrc`, `.gitignore`, JSON permissions) and copy-if-absent
- **Skill deployment**: `{cache}/kit/skills/` — copies to `.claude/skills/`, `.agents/skills/`, `.gemini/skills/`; symlinks for `.opencode/commands/`
- **Hook sync** (absorbed from `5-sync-hooks.sh`): The hook registration logic currently delegated to `fab hook sync` via the shell script is absorbed directly into this step. `fab-kit` calls the hooklib sync function internally rather than shelling out to `fab hook sync`. The `fab/.kit/sync/5-sync-hooks.sh` script is removed.
- **Version stamp**: Writes `fab/.kit-sync-version`
- **Legacy cleanup**: Removes old `.claude/agents/` files

#### Step 5: Direnv allow
If `.envrc` exists, runs `direnv allow`. Best-effort.

#### Step 6: Project sync scripts
Runs `fab/sync/*.sh` scripts in sorted order from repo root. `fab-kit` executes these directly — no dispatch to `fab-go`.

### Command flags

| Flag | Behavior |
|------|----------|
| (none) | Run all steps 1-6 |
| `--shim` | Steps 1-5 only (shim/workspace responsibilities) |
| `--project` | Step 6 only (project-specific scripts) |

### Fix `fragment-.envrc`

In `fab/.kit/scaffold/fragment-.envrc`, change:
```
export WORKTREE_INIT_SCRIPT="fab-kit sync"
```
to:
```
export WORKTREE_INIT_SCRIPT="fab sync"
```

### Migration for existing repos

Add a migration file (version range TBD based on release) that:
1. Finds `.envrc` lines containing `fab-kit sync` and replaces with `fab sync`
2. Verification: `.envrc` contains no `fab-kit sync` references

### Absorb hook sync into `fab-kit`

The `hooklib` package currently lives in `src/go/fab/internal/hooklib/` (the `fab` binary). To call it directly from `fab-kit` without shelling out:
- Either move `hooklib` to a shared internal package accessible by both binaries
- Or import it directly if the module structure allows

The `5-sync-hooks.sh` script in `fab/.kit/sync/` is removed after this change.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Remove `sync/5-sync-hooks.sh` from directory listing, document that sync reads from cache
- `fab-workflow/distribution`: (modify) Update sync description — reads from cache, version guard step, hook absorption, flag options
- `fab-workflow/setup`: (modify) Update sync delegation description if `/fab-setup` calls sync

## Impact

- **`src/go/fab-kit/internal/sync.go`** — major rewrite: all source paths change from repo-relative to cache-resolved, hook sync absorbed, symlink creation added, flag handling added
- **`src/go/fab-kit/internal/cache.go`** — may need `CachedKitDir()` exposed or new helpers for version comparison
- **`src/go/fab-kit/cmd/fab-kit/main.go`** — add `--shim` and `--project` flags to sync command
- **`src/go/fab/internal/hooklib/sync.go`** — either moved to shared package or duplicated in fab-kit
- **`fab/.kit/scaffold/fragment-.envrc`** — fix `fab-kit sync` → `fab sync`
- **`fab/.kit/sync/5-sync-hooks.sh`** — deleted
- **`fab/.kit/migrations/`** — new migration file for `.envrc` fix
- **`src/go/fab-kit/internal/sync_test.go`** — tests updated for cache-based resolution

## Open Questions

- What version range should the migration file use? Depends on the release version this ships in.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Sync reads from `~/.fab-kit/` cache | Discussed — user explicitly decided this is the new source | S:95 R:60 A:90 D:95 |
| 2 | Certain | 6-step pipeline: prerequisites, version guard, ensure cache, scaffolding, direnv, project scripts | Discussed — user provided the exact step ordering | S:95 R:70 A:90 D:95 |
| 3 | Certain | `--shim` (steps 1-5) and `--project` (step 6) flags | Discussed — user specified the split | S:90 R:80 A:85 D:90 |
| 4 | Certain | Hook sync absorbed into step 4, `5-sync-hooks.sh` removed | Discussed — user said hooks are part of scaffolding | S:90 R:65 A:85 D:90 |
| 5 | Certain | `fab-kit` executes project sync scripts directly (no fab-go dispatch) | Discussed — user confirmed fab-kit handles step 6 | S:95 R:80 A:90 D:95 |
| 6 | Certain | Fix `fragment-.envrc` from `fab-kit sync` to `fab sync` | Discussed — user identified this as a bug | S:95 R:85 A:95 D:95 |
| 7 | Certain | Migration to fix `.envrc` in existing repos | Discussed — user requested this explicitly | S:90 R:75 A:85 D:90 |
| 8 | Certain | No symlink — `fab/.kit/` remains a copied directory, sync just reads from cache | Discussed — user clarified symlink is deferred to follow-up change | S:95 R:85 A:90 D:95 |
| 9 | Certain | Agent dirs are gitignored, recreated per worktree | Discussed — user confirmed | S:90 R:85 A:90 D:95 |
| 10 | Certain | Offline clone support is not a concern | Discussed — user explicitly dismissed this | S:95 R:70 A:85 D:95 |
| 11 | Certain | Removing `fab/.kit/` from git is a separate follow-up | Discussed — user scoped this out explicitly | S:95 R:90 A:90 D:95 |
| 12 | Confident | `hooklib` needs to be shared or moved for direct `fab-kit` access | Inferred from architecture — hooklib is in `fab` binary, fab-kit needs it | S:70 R:55 A:80 D:70 |
| 13 | Certain | `fab init` continues copying kit to `fab/.kit/` — removal of `.kit` is a later change | Discussed — user confirmed this is follow-up scope | S:95 R:85 A:90 D:95 |
| 14 | Confident | Version guard compares semver of `fab_version` against embedded `fab-kit` binary version | Inferred from architecture — both use semver from same VERSION file at build time | S:75 R:75 A:85 D:80 |

14 assumptions (12 certain, 2 confident, 0 tentative, 0 unresolved).
