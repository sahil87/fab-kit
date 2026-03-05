# Intake: Scaffold Overlay Tree

**Change**: 260218-09fa-scaffold-overlay-tree
**Created**: 2026-02-18
**Status**: Draft

## Origin

> Restructure fab/.kit/scaffold/ as a repo-root overlay tree. Move scaffold files into a directory structure that mirrors their destination paths in the target repo. Files that are merge-fragments (line-ensuring or JSON-merge) get a `fragment-` prefix. Update 3-sync-workspace.sh to walk the scaffold tree generically instead of per-file bespoke blocks, dispatching merge strategy based on the fragment- prefix. Update fab-setup.md skill references to use the new paths.

Discussion-driven: the user and agent analyzed the current scaffold→destination mapping, evaluated overlay vs. manifest approaches, and agreed on the `fragment-` prefix convention over a `.merge-rules` manifest file. The key tradeoff was self-describing filenames (prefix) vs. pure 1:1 overlay (manifest). Prefix won on simplicity — only 3 of 11 files need it, and it avoids a coordination file.

## Why

1. **Implicit mapping**: Currently each scaffold file has a bespoke code block in `3-sync-workspace.sh` that hardcodes its destination. Adding a new scaffold file requires both creating the file and writing a new sync block. With the overlay tree, the file's path *is* its destination — drop it in and it works.

2. **Cognitive load**: The flat `scaffold/` directory with names like `envrc`, `gitignore-entries`, `memory-index.md`, `sync-readme.md` requires a mental lookup table to know where each file goes. The tree structure makes this self-evident: `scaffold/docs/memory/index.md` → `docs/memory/index.md`.

3. **Sync script simplification**: The current `3-sync-workspace.sh` has 9 bespoke sections (one per scaffold file type). A generic tree-walk with strategy dispatch based on the `fragment-` prefix would be shorter, more maintainable, and automatically handle new files.

## What Changes

### Scaffold directory restructure

Move the 11 flat scaffold files into a tree that mirrors the repo root:

| Current path | New path | Merge type |
|---|---|---|
| `scaffold/envrc` | `scaffold/fragment-.envrc` | line-ensure |
| `scaffold/gitignore-entries` | `scaffold/fragment-.gitignore` | line-ensure |
| `scaffold/settings.local.json` | `scaffold/.claude/fragment-settings.local.json` | json-merge |
| `scaffold/memory-index.md` | `scaffold/docs/memory/index.md` | copy-if-absent |
| `scaffold/specs-index.md` | `scaffold/docs/specs/index.md` | copy-if-absent |
| `scaffold/sync-readme.md` | `scaffold/fab/sync/README.md` | copy-if-absent |
| `scaffold/config.yaml` | `scaffold/fab/config.yaml` | template (fab-setup) |
| `scaffold/constitution.md` | `scaffold/fab/constitution.md` | template (fab-setup) |
| `scaffold/context.md` | `scaffold/fab/context.md` | copy-if-absent (fab-setup) |
| `scaffold/code-quality.md` | `scaffold/fab/code-quality.md` | copy-if-absent (fab-setup) |
| `scaffold/code-review.md` | `scaffold/fab/code-review.md` | copy-if-absent (fab-setup) |

### Convention: `fragment-` prefix

Files prefixed with `fragment-` are merge sources, not complete files. The sync script strips the prefix to determine the target filename and applies the appropriate merge strategy:

- **`fragment-` + dotfile** (`.envrc`, `.gitignore`): line-ensuring merge — each non-comment line is appended if not already present
- **`fragment-` + `.json`**: JSON deep-merge — specifically for `settings.local.json`, merge the `permissions.allow` array

Files without the prefix are either copy-if-absent (sync) or templates (fab-setup).

### Sync script rewrite (`3-sync-workspace.sh`)

Replace the per-file bespoke sections (sections 2, 3, 4, 7, 8, 9) with a generic scaffold tree-walk:

1. Walk `scaffold/` recursively, collecting all files
2. For each file, compute the destination path by stripping the `scaffold/` prefix
3. If the filename starts with `fragment-`, strip the prefix from the filename to get the target name, then dispatch to the appropriate merge function based on file type
4. If no `fragment-` prefix, copy-if-absent

The skill/agent sections (5, 6) and directory creation (1) remain as-is — they don't come from scaffold.

Files consumed by `fab-setup.md` (config.yaml, constitution.md, context.md, code-quality.md, code-review.md) should be **skipped** by the sync tree-walk. fab-setup uses them as templates with interactive prompting, not as direct copies. Mechanism: either a skip-list in the script, or a convention like placing them under a `scaffold/fab/` subtree that sync knows to exclude (since fab-setup handles `fab/` files).

### fab-setup.md skill reference updates

Update all references from `fab/.kit/scaffold/config.yaml` → `fab/.kit/scaffold/fab/config.yaml`, etc. for the five files fab-setup consumes.

### Migration reference updates

Update `fab/.kit/migrations/0.7.0-to-0.8.0.md` reference from `fab/.kit/scaffold/code-quality.md` → `fab/.kit/scaffold/fab/code-quality.md`.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the scaffold overlay tree structure and fragment- convention

## Impact

- **`fab/.kit/sync/3-sync-workspace.sh`** — major rewrite of sections 2, 3, 4, 7, 8, 9 into generic tree-walk
- **`fab/.kit/scaffold/`** — all 11 files relocated into tree structure
- **`fab/.kit/skills/fab-setup.md`** — path reference updates (5 files)
- **`fab/.kit/migrations/0.7.0-to-0.8.0.md`** — path reference update (1 file)
- **Existing projects**: No migration needed — the sync script reads from `.kit/scaffold/` which is in the kit itself, not in the target project. Next sync run will just work.

## Open Questions

- Should `fab-setup` files (config.yaml, constitution.md, context.md, code-quality.md, code-review.md) be excluded from the sync tree-walk via an explicit skip-list, or by convention (e.g. sync skips `scaffold/fab/` entirely since fab-setup owns that subtree)?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `fragment-` prefix (not suffix, not manifest) | Explicitly agreed in conversation — prefix chosen over `.merge-rules` manifest and suffix | S:95 R:80 A:90 D:95 |
| 2 | Certain | Three files get fragment- prefix: .envrc, .gitignore, settings.local.json | These are the only line-merge/JSON-merge files; all others are copy-if-absent or templates | S:90 R:85 A:95 D:90 |
| 3 | Confident | Skill/agent sync sections (5, 6) remain unchanged | They operate on `fab/.kit/skills/`, not scaffold — orthogonal to this change | S:80 R:90 A:85 D:85 |
| 4 | Confident | No migration needed for existing projects | Scaffold lives inside `.kit/` which is the engine, not the target repo | S:75 R:90 A:80 D:85 |
| 5 | Tentative | Sync tree-walk skips `scaffold/fab/` subtree (fab-setup owns it) | Clean separation but needs validation — alternative is explicit skip-list | S:60 R:70 A:55 D:50 |

5 assumptions (2 certain, 2 confident, 1 tentative).
