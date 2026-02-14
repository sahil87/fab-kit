# Init

**Domain**: fab-workflow

## Overview

`/fab-init` is the structural bootstrap skill that creates the `fab/` directory layout. It also provides subcommands for managing `config.yaml` and `constitution.md`, and for validating their structure. It does not handle memory hydration — that responsibility belongs to `/docs-hydrate-memory`.

## Requirements

### Structural Bootstrap Only

`/fab-init` performs only Phase 1 (structural bootstrap). It does not accept `[sources...]` arguments and contains no source hydration logic.

- Creates `fab/config.yaml` (project configuration)
- Creates `fab/constitution.md` (project principles)
- Creates `fab/VERSION` (local project version — via `lib/init-scaffold.sh`)
- Creates `docs/memory/index.md` (memory index skeleton)
- Creates `docs/specs/index.md` (specifications index skeleton — pre-implementation, human-curated)
- Creates `fab/changes/` directory
- Creates skill symlinks via `lib/init-scaffold.sh` glob pattern
- Creates `.gitignore` entries
- Safe to re-run (idempotent — skips existing files)

### Subcommands

`/fab-init` accepts three subcommands: `config [section]`, `constitution`, and `validate`. These provide ongoing management of initialization artifacts without requiring separate commands.

### Unrecognized Arguments Rejected

When arguments other than recognized subcommands are passed, init outputs a redirect message: "Did you mean /fab-hydrate? /fab-init no longer accepts source arguments." No hydration occurs.

### Output

First-run output lists only structural artifacts created. Next step suggests `/fab-new <description>` or `/fab-hydrate <sources>` to populate memory. The "With Sources" output section has been removed.

### Bootstrap Alternative

As an alternative to manual `cp -r`, new projects can use the one-liner bootstrap:

```
curl -sL https://github.com/wvrdz/fab-kit/releases/latest/download/kit.tar.gz | tar xz -C fab/
```

After extraction, run `fab/.kit/scripts/lib/init-scaffold.sh` then `/fab-init` as usual.

## Subcommand Architecture

The three subcommands manage the lifecycle of Fab's initialization artifacts:

| Subcommand | Purpose |
|---------|---------|
| `/fab-init constitution` | Create or amend `constitution.md` with semantic versioning (see [configuration](configuration.md#amending-constitution)) |
| `/fab-init config` | Create or update `config.yaml` interactively, preserving comments (see [configuration](configuration.md#updating-config)) |
| `/fab-init validate` | Validate structural correctness of both files (see [configuration](configuration.md#validation)) |

`/fab-init` delegates artifact creation to the subcommands:

- Step 1a: If `config.yaml` doesn't exist → invokes `/fab-init config` in create mode
- Step 1b: If `constitution.md` doesn't exist → invokes `/fab-init constitution` in create mode

This ensures each subcommand is the single source of truth for its artifact's generation logic. `/fab-init` retains ownership of structural orchestration (directories, symlinks, `.gitignore`).

Each subcommand operates independently — they can be invoked directly without going through `/fab-init`. This supports two workflows:

1. **Initial setup**: `/fab-init` orchestrates everything (delegates to subcommands internally)
2. **Ongoing management**: User invokes subcommands directly as project evolves

## Delegation Pattern

`/fab-init` delegates structural setup to `fab/.kit/scripts/lib/init-scaffold.sh` and adds interactive configuration on top. This means `lib/init-scaffold.sh` can be run independently (e.g., in CI or after a bootstrap download) without requiring `/fab-init`.

| Responsibility | Owner | Notes |
|---|---|---|
| Directories (`changes/`, `memory/`, `specs/`) | `lib/init-scaffold.sh` | Non-interactive, scriptable |
| `fab/VERSION` | `lib/init-scaffold.sh` | New project → engine version; existing project (has `config.yaml`) → `0.1.0`; existing file → preserved |
| Skeleton files (`memory/index.md`, `specs/index.md`) | `lib/init-scaffold.sh` | Copies from `scaffold/memory-index.md` and `scaffold/specs-index.md`; idempotent — skips if file exists |
| Skill symlinks (Claude Code, OpenCode, Codex) | `lib/init-scaffold.sh` | Discovers skills via glob pattern |
| `.envrc` symlink | `lib/init-scaffold.sh` | Links to `fab/.kit/scaffold/envrc` |
| `.gitignore` entries | `lib/init-scaffold.sh` | Appends entries from `scaffold/gitignore-entries` if not present |
| `config.yaml` | `/fab-init config` (delegated by `/fab-init`) | Single source of truth for config generation and updates |
| `constitution.md` | `/fab-init constitution` (delegated by `/fab-init`) | Single source of truth for constitution generation and amendments |

`/fab-init` invokes `lib/init-scaffold.sh` as step 1g of its bootstrap sequence. Steps 1c–1f in `/fab-init` have idempotent guards so they gracefully skip artifacts already created by `lib/init-scaffold.sh`.

**Bootstrap path** (without `/fab-init`): After downloading `fab/.kit/` via curl or `cp -r`, running `lib/init-scaffold.sh` alone creates a complete structural scaffold. `/fab-init` is only needed to generate `config.yaml` and `constitution.md`.

## Design Decisions

### Init as Pure Structural Bootstrap
**Decision**: `/fab-init` only creates directory structure and configuration files. Source hydration is delegated to `/docs-hydrate-memory`.
**Why**: Clean separation of concerns — bootstrap runs once per project, hydration runs whenever new sources need ingesting. Using "init" for repeated hydration was confusing.
**Rejected**: Keeping hydration in init with an optional flag — muddled the interface and made init's help text complex.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Redirect Message for Old Interface
**Decision**: When arguments are passed to `/fab-init`, show a helpful redirect to `/docs-hydrate-memory` instead of silently ignoring.
**Why**: Better UX — users who remember the old interface get guided to the new one.
**Rejected**: Silently ignoring arguments — confusing, user would think hydration happened.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Consolidated Skill with Subcommands
**Decision**: All three commands are subcommands within a single `fab-init.md` skill file. Each subcommand has its own behavior section, sharing the same `model_tier` and frontmatter.
*Introduced by*: 260213-3tyk-merge-fab-init-subcommands

### Config Updates Use String Replacement
**Decision**: `/fab-init config` uses targeted string replacement rather than full YAML parse-and-rewrite. This preserves the heavily-commented `config.yaml` format at the cost of slightly less structural safety.
*Introduced by*: 260212-h9k3-fab-init-family

### Validate Is Read-Only
**Decision**: `/fab-init validate` only checks and reports — it never modifies files. Fix suggestions are provided but the user applies them (directly or via the other subcommands).
*Introduced by*: 260212-h9k3-fab-init-family

## Deprecated Requirements

### Source Hydration (Phase 2)
**Deprecated by**: 260207-q7m3-separate-hydrate-smart-context (2026-02-07)
**Reason**: Source hydration extracted to dedicated `/docs-hydrate-memory` skill for better separation of concerns.
**Migration**: Use `/fab-hydrate [sources...]` instead of `/fab-init [sources...]`.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260214-m3v8-relocate-docs-dev-scripts | 2026-02-14 | Updated `lib/init-scaffold.sh` delegation to create `docs/memory/` and `docs/specs/` instead of `fab/memory/` and `fab/specs/` |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_init_scaffold.sh` → `lib/init-scaffold.sh` throughout (directory structure, delegation table, bootstrap references, design decisions, changelog entries) |
| — | 2026-02-14 | Absorbed init-family.md — subcommand architecture, delegation details, and design decisions (memory reorganization) |
| 260213-k7m2-kit-version-migrations | 2026-02-14 | Added `fab/VERSION` to bootstrap steps and delegation table; updated step numbering (1e = VERSION, 1f = changes, 1g = symlinks, 1h = gitignore) |
| 260213-3njv-scaffold-dir | 2026-02-13 | Updated delegation table: `.envrc` → `scaffold/envrc`, `.gitignore` → `scaffold/gitignore-entries`, skeleton files → scaffold sources |
| 260213-3tyk-merge-fab-init-subcommands | 2026-02-13 | Consolidated init family into subcommands — `/fab-init config`, `/fab-init constitution`, `/fab-init validate` are now subcommands of `/fab-init` |
| 260213-iq2l-rename-setup-scripts | 2026-02-13 | Renamed `fab-setup.sh` → `lib/init-scaffold.sh` in delegation pattern and all references |
| 260212-h9k3-fab-init-family | 2026-02-12 | Added Related Commands section, updated Delegation Pattern to reflect `/fab-init` delegating to `/fab-init config` and `/fab-init constitution` |
| 260212-emcb-clarify-fab-setup | 2026-02-12 | Added Delegation Pattern section documenting responsibility split between `/fab-init` and `lib/init-scaffold.sh` |
| 260210-h7r3-kit-distribution-update | 2026-02-10 | Added Bootstrap Alternative section with curl one-liner as alternative to manual `cp -r` |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-bb1q-add-specs-index | 2026-02-07 | Added `docs/specs/index.md` creation as step 1d in bootstrap sequence |
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Simplified to structural bootstrap only — removed Phase 2 source hydration, added argument redirect |
