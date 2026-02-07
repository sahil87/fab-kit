# Init

**Domain**: fab-workflow

## Overview

`/fab:init` is the structural bootstrap skill that creates the `fab/` directory layout. It sets up `config.yaml`, `constitution.md`, `docs/index.md`, the `changes/` directory, skill symlinks, and `.gitignore`. It does not handle doc hydration — that responsibility belongs to `/fab:hydrate`.

## Requirements

### Structural Bootstrap Only

`/fab:init` performs only Phase 1 (structural bootstrap). It does not accept `[sources...]` arguments and contains no source hydration logic.

- Creates `fab/config.yaml` (project configuration)
- Creates `fab/constitution.md` (project principles)
- Creates `fab/docs/index.md` (documentation index skeleton)
- Creates `fab/changes/` directory
- Creates skill symlinks via `fab-setup.sh` glob pattern
- Creates `.gitignore` entries
- Safe to re-run (idempotent — skips existing files)

### Arguments Rejected

When arguments are passed, init outputs a redirect message: "Did you mean /fab:hydrate? /fab:init no longer accepts source arguments." No hydration occurs.

### Output

First-run output lists only structural artifacts created. Next step suggests `/fab:new <description>` or `/fab:hydrate <sources>` to populate docs. The "With Sources" output section has been removed.

## Design Decisions

### Init as Pure Structural Bootstrap
**Decision**: `/fab:init` only creates directory structure and configuration files. Source hydration is delegated to `/fab:hydrate`.
**Why**: Clean separation of concerns — bootstrap runs once per project, hydration runs whenever new sources need ingesting. Using "init" for repeated hydration was confusing.
**Rejected**: Keeping hydration in init with an optional flag — muddled the interface and made init's help text complex.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Redirect Message for Old Interface
**Decision**: When arguments are passed to `/fab:init`, show a helpful redirect to `/fab:hydrate` instead of silently ignoring.
**Why**: Better UX — users who remember the old interface get guided to the new one.
**Rejected**: Silently ignoring arguments — confusing, user would think hydration happened.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

## Deprecated Requirements

### Source Hydration (Phase 2)
**Deprecated by**: 260207-q7m3-separate-hydrate-smart-context (2026-02-07)
**Reason**: Source hydration extracted to dedicated `/fab:hydrate` skill for better separation of concerns.
**Migration**: Use `/fab:hydrate [sources...]` instead of `/fab:init [sources...]`.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Simplified to structural bootstrap only — removed Phase 2 source hydration, added argument redirect |
