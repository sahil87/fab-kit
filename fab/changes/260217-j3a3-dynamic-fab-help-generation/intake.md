# Intake: Dynamic Fab Help Generation

**Change**: 260217-j3a3-dynamic-fab-help-generation
**Created**: 2026-02-17
**Status**: Draft

## Origin

> Rewrite fab-help.sh to dynamically generate command list from skill file frontmatter instead of hardcoding it. Delete the redundant .claude/agents/fab-help.md agent file. The skill files in fab/.kit/skills/ already have name and description in YAML frontmatter — use frontmatter_field() (from 2-sync-workspace.sh) to read them at runtime. This eliminates the main duplication point: fab-help.sh will always reflect the actual skill files. Keep the workflow diagram and grouping structure, just derive the command names and descriptions dynamically.

Preceded by a discussion identifying that the complete command list is duplicated across ~6 locations. The help script is the designated "single source of truth" for runtime help, but it hardcodes every command name and description. The skill files themselves already carry canonical `name` and `description` in YAML frontmatter — the help script should read from them instead of maintaining a parallel copy.

Additionally, `.claude/agents/fab-help.md` is a near-identical duplicate of `fab/.kit/skills/fab-help.md` with an outdated `model: haiku` frontmatter key (should be `model_tier: fast`). It causes the agent framework to spawn a subprocess instead of running the skill inline, which breaks output display.

## Why

1. **Problem**: The command list in `fab-help.sh` is hardcoded and already drifting from reality — `/fab-fff` is missing, `/fab-apply` and `/fab-review` are listed as separate commands but don't exist as skill files (they're sub-behaviors of `/fab-continue`). Any new skill added to `fab/.kit/skills/` requires a manual update to `fab-help.sh`.

2. **Consequence**: Users see stale or incorrect help output. The "single source of truth" claim is undermined when the help content is itself a stale copy. Adding a new skill requires updating ~6 files, of which `fab-help.sh` is the most likely to be forgotten.

3. **Approach**: The skill files already have structured frontmatter with `name` and `description`. The `frontmatter_field()` function already exists in `fab/.kit/sync/2-sync-workspace.sh` for parsing this format. Rather than maintaining a parallel list, `fab-help.sh` should scan skill files at runtime and extract names/descriptions dynamically. The workflow diagram and group headings remain hardcoded (they're layout, not data), but the command entries within each group come from the skill files.

## What Changes

### 1. Rewrite `fab/.kit/scripts/fab-help.sh`

Replace the hardcoded `cat <<EOF` block with dynamic generation:

- **Keep static**: Version header, workflow diagram, group headings ("Start & Navigate", "Planning", etc.), and "Typical Flow" footer
- **Make dynamic**: Command names and descriptions within each group are read from skill file frontmatter using `frontmatter_field()`
- **Grouping strategy**: The script maintains a hardcoded mapping of skill names to groups. This is a display concern — which group a command appears under — and doesn't belong in the skill frontmatter. The mapping is a short associative array or case statement in the script itself
- **Excluded files**: Files prefixed with `_` (partials like `_context.md`, `_generation.md`) and files prefixed with `internal-` are excluded from help output — they're not user-facing commands
- **Non-skill entries**: `fab-sync.sh` appears in the current help but isn't a skill — it remains a hardcoded line in the "Setup" group since it has no frontmatter to read

### 2. Extract `frontmatter_field()` to shared lib

Move `frontmatter_field()` from `fab/.kit/sync/2-sync-workspace.sh` to a new shared file `fab/.kit/scripts/lib/frontmatter.sh` that both `2-sync-workspace.sh` and `fab-help.sh` can source. This avoids duplicating the function. `2-sync-workspace.sh` replaces its inline copy with `source "$scripts_dir/lib/frontmatter.sh"`.

### 3. Delete `.claude/agents/fab-help.md`

Remove the redundant agent file. The skill at `fab/.kit/skills/fab-help.md` already exists and works correctly when invoked inline. The agent version uses the outdated `model: haiku` frontmatter key and causes the output to render incorrectly because it spawns a subprocess.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document `lib/frontmatter.sh` as a shared library and the dynamic help generation approach

## Impact

- **`fab/.kit/scripts/fab-help.sh`** — complete rewrite of command listing logic
- **`fab/.kit/sync/2-sync-workspace.sh`** — refactor to source `frontmatter_field()` from shared lib instead of inlining it
- **`fab/.kit/scripts/lib/frontmatter.sh`** — new shared library file
- **`.claude/agents/fab-help.md`** — deleted
- **No changes to skill files** — frontmatter format is already correct and sufficient

## Open Questions

- None — the input is clear and all design decisions are well-grounded by existing patterns.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Exclude `_*` and `internal-*` prefixed skill files from help output | Convention is clear: `_` = shared partial, `internal-` = internal tooling. Neither is user-invocable | S:70 R:95 A:90 D:85 |
| 2 | Certain | Delete `.claude/agents/fab-help.md` | Explicitly requested. Duplicate of skill file with outdated frontmatter | S:95 R:90 A:90 D:95 |
| 3 | Confident | Keep group headings hardcoded in the script, not in skill frontmatter | Groups are a display concern. Adding a `group` field to every skill file would be over-engineering for a static layout that rarely changes | S:60 R:90 A:70 D:50 |
| 4 | Confident | Extract `frontmatter_field()` to `lib/frontmatter.sh` rather than duplicating it | Constitution principle I (Pure Prompt Play) favors shared utilities. `2-sync-workspace.sh` already has the function; both scripts need it | S:50 R:95 A:80 D:60 |
| 5 | Confident | Keep `fab-sync.sh` as a hardcoded line in help (it's not a skill) | No frontmatter to read. It's a shell script, not a skill. One hardcoded entry is simpler than creating a pseudo-skill file | S:40 R:90 A:75 D:60 |

5 assumptions (2 certain, 3 confident, 0 tentative, 0 unresolved).
