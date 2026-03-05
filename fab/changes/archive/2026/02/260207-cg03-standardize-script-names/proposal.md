# Proposal: Standardize Script Names and Add fab-help.sh

**Change**: 260207-cg03-standardize-script-names
**Created**: 2026-02-07
**Status**: Draft

## Why

The scripts in `fab/.kit/scripts/` currently use bare names (`setup.sh`, `status.sh`, `update-claude-settings.sh`) with no common prefix. This is inconsistent with the `fab-*` naming convention used for skills, makes scripts harder to identify at a glance, and creates ambiguity when referenced from docs or permissions. Adding a `fab-` prefix unifies the naming across the kit. Additionally, the `/fab:help` skill currently has no shell script backing — creating `fab-help.sh` lets the help output be generated from a single executable source of truth, consistent with how `fab-status` uses `status.sh`.

## What Changes

- **Rename** `fab/.kit/scripts/setup.sh` → `fab-setup.sh`
- **Rename** `fab/.kit/scripts/status.sh` → `fab-status.sh`
- **Rename** `fab/.kit/scripts/update-claude-settings.sh` → `fab-update-claude-settings.sh`
- **Create** `fab/.kit/scripts/fab-help.sh` — shell script that reads `fab/.kit/VERSION` and outputs the help text currently hardcoded in the `fab-help.md` skill definition
- **Update** `fab/.kit/skills/fab-help.md` to invoke `fab-help.sh` instead of inlining the output
- **Update** all internal references to the old script names: <!-- clarified: exhaustive cross-repo grep performed -->
  - `fab/.kit/skills/fab-init.md` (references `setup.sh` on line 176)
  - `fab/.kit/scripts/fab-setup.sh` itself (self-referencing comments on lines 4, 9)
  - `fab/worktree-init/assets/settings.local.json` (permission pattern `Bash(fab/.kit/scripts/setup.sh:*)` on line 72)
  - `doc/fab-spec/ARCHITECTURE.md` (tree listing line 32, prose references on lines 112, 122, 372, 410, 417, 468)
  - `doc/fab-spec/README.md` (references `status.sh` on lines 25, 68)
- **Out of scope** (historical logs, not updated): `.ralph/` run logs/progress, `.agents/tasks/prd-fab-kit.json`, `fab/backlog.md` — these are historical records and should retain original names

## Affected Docs

### New Docs
- None (no centralized docs affected — this is kit-internal)

### Modified Docs
- None

### Removed Docs
- None

## Impact

- **`fab/.kit/scripts/`** — all 3 existing scripts renamed, 1 new script added
- **`fab/.kit/skills/fab-help.md`** — modified to delegate to shell script
- **`fab/.kit/skills/fab-init.md`** — updated reference from `setup.sh` to `fab-setup.sh`
- **`fab/worktree-init/assets/settings.local.json`** — updated permission pattern
- **`doc/fab-spec/`** — spec docs updated to reflect new names (ARCHITECTURE.md, README.md)
- **No external API changes** — scripts are internal kit utilities
- **Idempotency preserved** — all scripts retain their idempotent behavior
- **Historical files untouched** — `.ralph/` logs, `.agents/tasks/`, `fab/backlog.md` retain old names (they are historical records, not active references) <!-- clarified: explicit out-of-scope boundary -->

## Open Questions

- ~~[DEFERRED] Should `fab-setup.sh` internal comment header also reference the new name?~~ Resolved: yes, all self-references in comment headers will be updated to the new names. <!-- clarified: resolved deferred question -->
- [DEFERRED] The ARCHITECTURE.md tree listing (line 32) currently only shows `status.sh` under `scripts/`. After the rename, should we also add `fab-setup.sh`, `fab-help.sh`, and `fab-update-claude-settings.sh` to the tree listing for completeness? (Assumed yes — the tree should reflect actual contents.)
