# Init

**Domain**: fab-workflow

## Overview

`/fab-init` is the structural bootstrap skill that creates the `fab/` directory layout. It sets up `config.yaml`, `constitution.md`, `docs/index.md`, `design/index.md`, the `changes/` directory, skill symlinks, and `.gitignore`. It does not handle doc hydration — that responsibility belongs to `/fab-hydrate`.

## Requirements

### Structural Bootstrap Only

`/fab-init` performs only Phase 1 (structural bootstrap). It does not accept `[sources...]` arguments and contains no source hydration logic.

- Creates `fab/config.yaml` (project configuration)
- Creates `fab/constitution.md` (project principles)
- Creates `fab/docs/index.md` (documentation index skeleton)
- Creates `fab/design/index.md` (specifications index skeleton — pre-implementation, human-curated)
- Creates `fab/changes/` directory
- Creates skill symlinks via `fab-setup.sh` glob pattern
- Creates `.gitignore` entries
- Safe to re-run (idempotent — skips existing files)

### Arguments Rejected

When arguments are passed, init outputs a redirect message: "Did you mean /fab-hydrate? /fab-init no longer accepts source arguments." No hydration occurs.

### Output

First-run output lists only structural artifacts created. Next step suggests `/fab-new <description>` or `/fab-hydrate <sources>` to populate docs. The "With Sources" output section has been removed.

### Bootstrap Alternative

As an alternative to manual `cp -r`, new projects can use the one-liner bootstrap:

```
curl -sL https://github.com/wvrdz/fab-kit/releases/latest/download/kit.tar.gz | tar xz -C fab/
```

After extraction, run `fab/.kit/scripts/fab-setup.sh` then `/fab-init` as usual.

## Related Commands

For post-initialization management of config and constitution files, see the [init family](init-family.md):

- `/fab-init-constitution` — create or amend the constitution with semantic versioning (see [constitution governance](constitution-governance.md))
- `/fab-init-config` — interactive config.yaml updates preserving comments (see [config management](config-management.md))
- `/fab-init-validate` — structural validation for both files

## Delegation Pattern

`/fab-init` delegates structural setup to `fab/.kit/scripts/fab-setup.sh` and adds interactive configuration on top. This means `fab-setup.sh` can be run independently (e.g., in CI or after a bootstrap download) without requiring `/fab-init`.

| Responsibility | Owner | Notes |
|---|---|---|
| Directories (`changes/`, `docs/`, `design/`) | `fab-setup.sh` | Non-interactive, scriptable |
| Skeleton files (`docs/index.md`, `design/index.md`) | `fab-setup.sh` | Idempotent — skips if file exists |
| Skill symlinks (Claude Code, OpenCode, Codex) | `fab-setup.sh` | Discovers skills via glob pattern |
| `.envrc` symlink | `fab-setup.sh` | Links to `fab/.kit/envrc` |
| `.gitignore` (`fab/current` entry) | `fab-setup.sh` | Appends if not present |
| `config.yaml` | `/fab-init-config` (delegated by `/fab-init`) | Single source of truth for config generation and updates |
| `constitution.md` | `/fab-init-constitution` (delegated by `/fab-init`) | Single source of truth for constitution generation and amendments |

`/fab-init` invokes `fab-setup.sh` as step 1f of its bootstrap sequence. Steps 1c–1e in `/fab-init` have idempotent guards so they gracefully skip artifacts already created by `fab-setup.sh`.

**Bootstrap path** (without `/fab-init`): After downloading `fab/.kit/` via curl or `cp -r`, running `fab-setup.sh` alone creates a complete structural scaffold. `/fab-init` is only needed to generate `config.yaml` and `constitution.md`.

## Design Decisions

### Init as Pure Structural Bootstrap
**Decision**: `/fab-init` only creates directory structure and configuration files. Source hydration is delegated to `/fab-hydrate`.
**Why**: Clean separation of concerns — bootstrap runs once per project, hydration runs whenever new sources need ingesting. Using "init" for repeated hydration was confusing.
**Rejected**: Keeping hydration in init with an optional flag — muddled the interface and made init's help text complex.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Redirect Message for Old Interface
**Decision**: When arguments are passed to `/fab-init`, show a helpful redirect to `/fab-hydrate` instead of silently ignoring.
**Why**: Better UX — users who remember the old interface get guided to the new one.
**Rejected**: Silently ignoring arguments — confusing, user would think hydration happened.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

## Deprecated Requirements

### Source Hydration (Phase 2)
**Deprecated by**: 260207-q7m3-separate-hydrate-smart-context (2026-02-07)
**Reason**: Source hydration extracted to dedicated `/fab-hydrate` skill for better separation of concerns.
**Migration**: Use `/fab-hydrate [sources...]` instead of `/fab-init [sources...]`.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260212-h9k3-fab-init-family | 2026-02-12 | Added Related Commands section, updated Delegation Pattern to reflect `/fab-init` delegating to `/fab-init-config` and `/fab-init-constitution` |
| 260212-emcb-clarify-fab-setup | 2026-02-12 | Added Delegation Pattern section documenting responsibility split between `/fab-init` and `fab-setup.sh` |
| 260210-h7r3-kit-distribution-update | 2026-02-10 | Added Bootstrap Alternative section with curl one-liner as alternative to manual `cp -r` |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-bb1q-add-specs-index | 2026-02-07 | Added `fab/design/index.md` creation as step 1d in bootstrap sequence |
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Simplified to structural bootstrap only — removed Phase 2 source hydration, added argument redirect |
