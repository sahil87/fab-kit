# Intake: Move env-packages.sh to lib & Add fab-pipeline.sh Entry Point

**Change**: 260221-i0z6-move-env-packages-add-fab-pipeline
**Created**: 2026-02-21
**Status**: Draft

## Origin

> Move env-packages.sh from fab/.kit/scripts/ to fab/.kit/scripts/lib/ (it's a sourceable helper, not a user command, and scripts/ is on PATH). Update references in scaffold/fragment-.envrc and src/packages/rc-init.sh. Add a thin fab-pipeline.sh wrapper in fab/.kit/scripts/ that delegates to pipeline/run.sh with arg passthrough. Convenience: if the argument has no path separator and no .yaml extension, resolve it as fab/pipelines/{name}.yaml.

Discussion-driven change. User identified two issues with `fab/.kit/scripts/` layout during a `/fab-discuss` session:

1. `env-packages.sh` is a sourceable env helper but sits in a `PATH_add`-ed directory, making it appear as a user command
2. `pipeline/run.sh` is a user-facing orchestrator but has no PATH-accessible entry point — users must type the full path

## Why

1. **env-packages.sh pollution**: `.envrc` adds `fab/.kit/scripts/` to PATH via `PATH_add`. This means `env-packages.sh` shows up as a callable command, but it's a `source`-able helper — running it directly does nothing useful. Moving it to `lib/` (a subdirectory not on PATH) keeps it internal.

2. **Pipeline discoverability**: `pipeline/run.sh` is the main orchestrator script users run to execute a pipeline manifest, but it lives in a subdirectory (`pipeline/`) not directly on PATH. A thin wrapper in `scripts/` gives users a clean `fab-pipeline.sh <manifest>` invocation consistent with `fab-help.sh`, `fab-sync.sh`, and `fab-upgrade.sh`.

3. If we don't fix these: users see a confusing `env-packages.sh` command in their shell tab completion, and they have to remember the full `fab/.kit/scripts/pipeline/run.sh` path every time they want to run a pipeline.

## What Changes

### 1. Move `env-packages.sh` to `lib/`

Move `fab/.kit/scripts/env-packages.sh` → `fab/.kit/scripts/lib/env-packages.sh`.

The file contents remain unchanged:

```bash
#!/usr/bin/env bash
# Add all fab-kit package bin directories to PATH
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KIT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
for d in "$KIT_DIR"/packages/*/bin; do
  [ -d "$d" ] && export PATH="$d:$PATH"
done
```

**After the move**, the `SCRIPT_DIR`/`KIT_DIR` relative path resolution changes — `SCRIPT_DIR` will now be `.../scripts/lib/`, so `KIT_DIR` needs to go up one more level. Update to:

```bash
KIT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
```

### 2. Update references to `env-packages.sh`

Two files source `env-packages.sh` and need path updates:

**`fab/.kit/scaffold/fragment-.envrc`** (line 4):
```bash
# before
source fab/.kit/scripts/env-packages.sh
# after
source fab/.kit/scripts/lib/env-packages.sh
```

**`src/packages/rc-init.sh`** (line 14):
```bash
# before
source "$SCRIPT_DIR/../../fab/.kit/scripts/env-packages.sh"
# after
source "$SCRIPT_DIR/../../fab/.kit/scripts/lib/env-packages.sh"
```

### 3. Add `fab-pipeline.sh` wrapper

Create `fab/.kit/scripts/fab-pipeline.sh` — a thin wrapper that delegates to `pipeline/run.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# fab-pipeline.sh — Entry point for the pipeline orchestrator
#
# Usage: fab-pipeline.sh <manifest>
#
# If the argument contains no path separator and no .yaml extension,
# resolves it as fab/pipelines/{name}.yaml.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ $# -lt 1 ]]; then
  echo "Usage: fab-pipeline.sh <manifest>" >&2
  echo "       fab-pipeline.sh my-feature  →  fab/pipelines/my-feature.yaml" >&2
  exit 1
fi

manifest="$1"
shift

# Convenience: bare name → fab/pipelines/{name}.yaml
if [[ "$manifest" != */* && "$manifest" != *.yaml ]]; then
  manifest="fab/pipelines/${manifest}.yaml"
fi

exec bash "$SCRIPT_DIR/pipeline/run.sh" "$manifest" "$@"
```

Make executable: `chmod +x fab/.kit/scripts/fab-pipeline.sh`.

### 4. Update documentation references

Memory and docs files that reference `env-packages.sh` at its old path need updating:

- `docs/memory/fab-workflow/kit-architecture.md` — directory tree listing and description section
- `docs/memory/fab-workflow/distribution.md` — references to env-packages.sh sourcing path
- `README.md` — delegating-to-env-packages.sh description

Also add `fab-pipeline.sh` to the directory tree in `kit-architecture.md`.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update directory tree to show `env-packages.sh` under `lib/`, add `fab-pipeline.sh` to scripts listing
- `fab-workflow/distribution`: (modify) Update `env-packages.sh` path references

## Impact

- **User PATH**: No change — `fab/.kit/scripts/` remains on PATH. `env-packages.sh` disappears from tab completion (good). `fab-pipeline.sh` appears (good).
- **Existing .envrc files**: Projects that have already synced will have the old `source fab/.kit/scripts/env-packages.sh` line. The next `fab-sync.sh` / `fab-upgrade.sh` run will pick up the new scaffold fragment. Manual fix is trivial if needed.
- **rc-init.sh**: Shell rc users get the fix immediately on next source.
- **Packages PATH setup**: Functionally identical — packages still get added to PATH via the same mechanism, just sourced from a different location.

## Open Questions

- None — scope is well-defined from the discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Move destination is `lib/` not a new subfolder | `lib/` already exists and holds internal sourceable scripts; creating `includes/` would add a second internal folder with no clear distinction | S:95 R:90 A:95 D:90 |
| 2 | Certain | `env-packages.sh` needs `KIT_DIR` path update after move | Moving one directory deeper changes the relative path from `SCRIPT_DIR` to `KIT_DIR` — must go up `../..` instead of `..` | S:95 R:95 A:95 D:95 |
| 3 | Certain | Wrapper uses `exec` delegation, not function copy | Keeps `pipeline/run.sh` as single source of truth; wrapper is a thin entry point only | S:90 R:95 A:90 D:95 |
| 4 | Confident | Bare-name convenience resolves to `fab/pipelines/{name}.yaml` | Matches the convention from `pipeline/run.sh` usage examples; user confirmed the convenience feature | S:85 R:90 A:80 D:75 |
| 5 | Confident | Documentation updates are in-scope | Memory files and README reference the old path; keeping them accurate is part of the change | S:80 R:85 A:85 D:80 |

5 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).
