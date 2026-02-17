# Workspace Sync (fab-sync)

Structural bootstrap script that syncs kit assets into the workspace. Creates directories, skill symlinks, agent files, skeleton docs, `.envrc`, `fab/VERSION`, and `.gitignore` entries. Cleans up stale artifacts from deleted skills. Idempotent — safe to re-run.

## Sources of Truth

- **Implementation**: `fab/.kit/scripts/fab-sync.sh` — main file (distributed with kit)
- **Dev symlink**: `src/lib/fab-sync/fab-sync.sh` → `../../../fab/.kit/scripts/fab-sync.sh`
- **Architecture docs**: `docs/memory/fab-workflow/kit-architecture.md` — directory structure, script descriptions

## Usage

```bash
# Run directly (from anywhere in the repo)
fab/.kit/scripts/fab-sync.sh

# Or via dev symlink
src/lib/fab-sync/fab-sync.sh
```

No arguments. No flags. The script resolves paths relative to its own location.

## Behavior Reference

### Pre-flight

- Checks `fab/.kit/VERSION` exists (exits with error if missing)

### 1. Directory Creation

Creates `fab/changes/`, `fab/changes/archive/`, `docs/memory/`, `docs/specs/` (with `mkdir -p`). Creates `fab/changes/.gitkeep` and `fab/changes/archive/.gitkeep`. Skips existing directories and files.

### 2. fab/VERSION

- **New project** (no `fab/config.yaml`): copies `fab/.kit/VERSION` value
- **Existing project** (has `config.yaml`, no `fab/VERSION`): writes `0.1.0`
- **Already exists**: preserves existing file

### 3. .envrc Symlink

Creates symlink `.envrc` → `fab/.kit/scaffold/envrc`. Repairs broken symlinks. Replaces regular files with symlinks.

### 4. Memory/Specs Index Seeding

Copies `fab/.kit/scaffold/memory-index.md` → `docs/memory/index.md` and `fab/.kit/scaffold/specs-index.md` → `docs/specs/index.md`. Skips if target exists.

### 5. Skill Sync (3 Platforms)

Discovers skills by globbing `fab/.kit/skills/*.md` (excluding `_*.md` partials). For each skill:

- **Claude Code**: `.claude/skills/<name>/SKILL.md` → symlink to `../../../fab/.kit/skills/<name>.md`
- **OpenCode**: `.opencode/commands/<name>.md` → symlink to `../../fab/.kit/skills/<name>.md`
- **Codex**: `.agents/skills/<name>/SKILL.md` → file copy (Codex ignores symlinks)

After syncing each platform, removes stale entries (skill directories/files not in the current skills list).

Reports created/repaired/valid counts per platform.

### 6. Model-Tier Agent Generation

Identifies `fast`-tier skills (via `model_tier: fast` in YAML frontmatter). For each:

- Reads the Claude model mapping from `fab/config.yaml` `model_tiers.fast.claude` (falls back to `haiku` if absent or no config.yaml)
- Generates `.claude/agents/<name>.md` with `model_tier:` replaced by `model:` platform-specific value
- Updates existing agent files if content changed
- Removes stale agent files for skills that no longer exist in `.kit/skills/`

### 7. .gitignore Management

Reads entries from `fab/.kit/scaffold/gitignore-entries`. For each entry: creates `.gitignore` if missing, appends entry if not already present. Skips comments and empty lines.

## Requirements

- Bash 4.0+
- GNU coreutils (grep, sed, head, cmp, ln, cp, mkdir)
- No `yq` dependency

## Testing

```bash
# Run bats test suite
bats src/lib/fab-sync/test.bats
```
