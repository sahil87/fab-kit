# Setup

**Domain**: fab-workflow

## Overview

`/fab-setup` is the structural bootstrap skill that creates the `fab/` directory layout. It also provides subcommands for managing `config.yaml` and `constitution.md` (with built-in validation), and for running version migrations (absorbed from the former `/fab-update`). It delegates structural setup to `fab-sync.sh`. It does not handle memory hydration — that responsibility belongs to `/docs-hydrate-memory`.

## Requirements

### Prerequisite Check (Phase 0)

`/fab-setup` (bare bootstrap only) runs `fab/.kit/scripts/fab-doctor.sh` as an early gate before creating any project artifacts. If doctor exits non-zero, setup stops immediately and surfaces the doctor output with fix hints. This gate does not apply to subcommands (`config`, `constitution`, `migrations`).

### Structural Bootstrap Only

`/fab-setup` performs only Phase 1 (structural bootstrap). It does not accept `[sources...]` arguments and contains no source hydration logic.

- Creates `fab/project/config.yaml` (project configuration)
- Creates `fab/project/constitution.md` (project principles)
- Creates `fab/.kit-migration-version` (migration version — via `fab-sync.sh`)
- Creates `docs/memory/index.md` (memory index skeleton)
- Creates `docs/specs/index.md` (specifications index skeleton — pre-implementation, human-curated)
- Creates `fab/changes/` directory
- Creates skill symlinks via `fab-sync.sh` glob pattern
- Creates `.gitignore` entries
- Safe to re-run (idempotent — skips existing files)

### Subcommands

`/fab-setup` accepts three subcommands: `config [section]`, `constitution`, and `migrations [file]`. These provide ongoing management of initialization artifacts and version migrations without requiring separate commands. Validation is built into the `config` and `constitution` flows rather than exposed as a standalone subcommand.

### Unrecognized Arguments Rejected

When arguments other than recognized subcommands are passed, setup outputs a redirect message listing the valid subcommands: `config`, `constitution`, `migrations`. No hydration occurs.

### Output

First-run output lists only structural artifacts created. Next step suggests `/fab-new <description>` or `/fab-hydrate <sources>` to populate memory. The "With Sources" output section has been removed.

### Bootstrap Alternative

As an alternative to manual `cp -r`, new projects can use the one-liner bootstrap:

```
curl -sL https://github.com/{repo}/releases/latest/download/kit.tar.gz | tar xz -C fab/
```

Where `{repo}` is the `repo` value from `fab/.kit/kit.conf`.

After extraction, run `fab/.kit/scripts/fab-sync.sh` then `/fab-setup` as usual.

## Subcommand Architecture

The subcommands manage the lifecycle of Fab's setup artifacts and migrations:

| Subcommand | Purpose |
|---------|---------|
| `/fab-setup constitution` | Create or amend `constitution.md` with semantic versioning (see [configuration](configuration.md#amending-constitution)) |
| `/fab-setup config` | Create or update `config.yaml` interactively, preserving comments (see [configuration](configuration.md#updating-config)) |
| `/fab-setup migrations [file]` | Run version migrations against the current project (see [migrations](migrations.md)) |

`/fab-setup` delegates artifact creation to the subcommands:

- Step 1a: If `config.yaml` doesn't exist → invokes `/fab-setup config` in create mode
- Step 1b: If `constitution.md` doesn't exist → invokes `/fab-setup constitution` in create mode

This ensures each subcommand is the single source of truth for its artifact's generation logic. `/fab-setup` retains ownership of structural orchestration (directories, symlinks, `.gitignore`).

Each subcommand operates independently — they can be invoked directly without going through `/fab-setup`. This supports two workflows:

1. **Initial setup**: `/fab-setup` orchestrates everything (delegates to subcommands internally)
2. **Ongoing management**: User invokes subcommands directly as project evolves

## Delegation Pattern

`/fab-setup` delegates structural setup to `fab/.kit/scripts/fab-sync.sh` and adds interactive configuration on top. This means `fab-sync.sh` can be run independently (e.g., in CI or after a bootstrap download) without requiring `/fab-setup`.

| Responsibility | Owner | Notes |
|---|---|---|
| Directories (`changes/`, `memory/`, `specs/`) | `fab-sync.sh` | Non-interactive, scriptable |
| `fab/.kit-migration-version` | `fab-sync.sh` | New project → engine version; existing project (has `config.yaml`) → `0.1.0`; existing file → preserved |
| Skeleton files (`memory/index.md`, `specs/index.md`) | `fab-sync.sh` | Copies from `scaffold/memory-index.md` and `scaffold/specs-index.md`; idempotent — skips if file exists |
| Skill symlinks (Claude Code, OpenCode, Codex) | `fab-sync.sh` | Discovers skills via glob pattern |
| `.envrc` symlink | `fab-sync.sh` | Links to `fab/.kit/scaffold/envrc` |
| `.gitignore` entries | `fab-sync.sh` | Appends entries from `scaffold/gitignore-entries` if not present |
| Hook registration | `fab-sync.sh` via `5-sync-hooks.sh` | Registers `fab/.kit/hooks/on-*.sh` into `.claude/settings.local.json` hooks; supports tool-name matchers for PostToolUse events; idempotent jq merge |
| `config.yaml` | `/fab-setup config` (delegated by `/fab-setup`) | Reads `scaffold/config.yaml` template, substitutes placeholders with user-provided values |
| `constitution.md` | `/fab-setup constitution` (delegated by `/fab-setup`) | Reads `scaffold/constitution.md` skeleton, generates principles from project context |

`/fab-setup` invokes `fab-sync.sh` as step 1g of its bootstrap sequence. Steps 1c–1f in `/fab-setup` have idempotent guards so they gracefully skip artifacts already created by `fab-sync.sh`.

**Bootstrap path** (without `/fab-setup`): After downloading `fab/.kit/` via curl or `cp -r`, running `fab-sync.sh` alone creates a complete structural scaffold. `/fab-setup` is only needed to generate `config.yaml` and `constitution.md`.

## Design Decisions

### Init as Pure Structural Bootstrap
**Decision**: `/fab-setup` only creates directory structure and configuration files. Source hydration is delegated to `/docs-hydrate-memory`.
**Why**: Clean separation of concerns — bootstrap runs once per project, hydration runs whenever new sources need ingesting. Using "init" for repeated hydration was confusing.
**Rejected**: Keeping hydration in init with an optional flag — muddled the interface and made init's help text complex.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Redirect Message for Old Interface
**Decision**: When arguments are passed to `/fab-setup`, show a helpful redirect to `/docs-hydrate-memory` instead of silently ignoring.
**Why**: Better UX — users who remember the old interface get guided to the new one.
**Rejected**: Silently ignoring arguments — confusing, user would think hydration happened.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Consolidated Skill with Subcommands
**Decision**: All four commands are subcommands within a single `fab-setup.md` skill file — `config`, `constitution`, `migrations`, and a validate-redirect for backward compatibility. Each subcommand has its own behavior section, sharing the same `model_tier` and frontmatter.
*Introduced by*: 260213-3tyk-merge-fab-init-subcommands

### Config Updates Use String Replacement
**Decision**: `/fab-setup config` uses targeted string replacement rather than full YAML parse-and-rewrite. This preserves the heavily-commented `config.yaml` format at the cost of slightly less structural safety.
*Introduced by*: 260212-h9k3-fab-init-family

### Validate Is Read-Only (deprecated)
**Decision**: `/fab-init validate` only checked and reported — it never modified files. Fix suggestions were provided but the user applied them (directly or via the other subcommands).
**Deprecated**: Validation is now folded into the `config` and `constitution` subcommand flows, removing the need for a standalone validate step.
*Introduced by*: 260212-h9k3-fab-init-family
*Deprecated by*: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow

### Templates in Scaffold Files
**Decision**: `config.yaml` and `constitution.md` templates live as standalone files in `fab/.kit/scaffold/` rather than as inline code blocks in `fab-setup.md`. `/fab-setup` reads from these files and substitutes placeholders. Index templates (`memory-index.md`, `specs-index.md`) are also referenced from scaffold files, eliminating duplicated inline copies.
**Why**: Prevents drift between inline templates and actual schema expectations. Aligns with Constitution V (Portability) — `.kit/` owns its templates as inspectable, diffable files. Single source of truth for both `fab-sync.sh` and `/fab-setup`.
**Rejected**: Keeping inline templates — two sources of truth that can diverge when the config schema evolves.
*Introduced by*: 260217-17pe-DEV-1046-scaffold-setup-templates

### Agent-Inferred Conventions Replace Templates (superseded)
**Decision**: Step 1b-lang uses agent inference (Detection → Inference → Write) instead of bundled language templates. The agent reads project marker files (`Cargo.toml`, `tsconfig.json`, `package.json`, `go.mod`, `pyproject.toml`, etc.) and linter/formatter configs, then derives conventions from its training knowledge grounded in actual config values. Conventions are routed to the appropriate `fab/project/*` file by content type (enforcement rules → constitution, stack info → context, coding standards → code-quality, review policy → code-review, source paths → config). The skill describes the *process*, not hard-coded convention content.
**Why**: Bundling language-specific templates in `fab/.kit/templates/constitutions/` and `fab/.kit/templates/configs/` violated Constitution §V (portability — no assumptions about host project's language/toolchain). Templates created maintenance burden and encoded opinions that may not match the project's actual setup.
**Rejected**: Keeping language templates — violates neutrality, creates maintenance burden, makes judgment calls on behalf of users.
*Introduced by*: 260306-143f-setup-language-inference
*Superseded by*: 260306-6bba-redesign-hooks-strategy — language-specific customization rejected entirely; fab-kit stays language-neutral. Step 1b-lang removed from bootstrap flow.

### Absorbed /fab-update into /fab-setup migrations
**Decision**: `/fab-update` functionality is now available as `/fab-setup migrations`. Version migrations live under the same command namespace as the rest of project setup.
**Why**: Reduces the dropped-ball two-step flow where users had to remember a separate `/fab-update` command after upgrading the kit. Makes migrations discoverable from the same command namespace as config and constitution management.
**Rejected**: Keeping `/fab-update` as a separate top-level skill — created a discoverability gap and a two-step flow that was easy to forget.
*Introduced by*: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow

## Deprecated Requirements

### Source Hydration (Phase 2)
**Deprecated by**: 260207-q7m3-separate-hydrate-smart-context (2026-02-07)
**Reason**: Source hydration extracted to dedicated `/docs-hydrate-memory` skill for better separation of concerns.
**Migration**: Use `/fab-hydrate [sources...]` instead of `/fab-setup [sources...]`.

### /fab-init validate Subcommand
**Deprecated by**: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow (2026-02-16)
**Reason**: Validation folded into the `config` and `constitution` subcommand flows. A standalone validate step was redundant — each subcommand now validates its own artifact as part of the create/update workflow.
**Migration**: Use `/fab-setup config` or `/fab-setup constitution` which include built-in validation.

### /fab-update
**Deprecated by**: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow (2026-02-16)
**Reason**: Absorbed into `/fab-setup migrations` to reduce the two-step upgrade flow and make migrations discoverable from the same command namespace.
**Migration**: Use `/fab-setup migrations [file]` instead of `/fab-update`.

### Template-Driven Language Detection (Step 1b-lang)
**Deprecated by**: 260306-143f-setup-language-inference (2026-03-06)
**Reason**: Replaced by agent-inferred conventions. Template files (`fab/.kit/templates/constitutions/`, `fab/.kit/templates/configs/`) deleted. Language template advisory in `fab/.kit/sync/2-sync-workspace.sh` (invoked by `fab-sync.sh`, section 2b) removed.
**Migration**: Step 1b-lang now uses agent inference — no user action required.

### Agent-Inferred Language Conventions (Step 1b-lang)
**Deprecated by**: 260306-6bba-redesign-hooks-strategy (2026-03-06)
**Reason**: Language-specific customization rejected entirely — fab-kit stays language-neutral per Constitution §V. Detection logic has no purpose without templates or language-specific content to produce. Agent inference (260306-143f) was a stepping stone that this change supersedes. Step 1b-lang removed from `fab-setup.md` bootstrap flow.
**Migration**: Projects that want language-specific conventions can add them manually to `fab/project/*` files.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260306-6bba-redesign-hooks-strategy | 2026-03-06 | Full removal of Phase 1b-lang (language detection and convention inference) from bootstrap flow — fab-kit stays language-neutral. Supersedes 260306-143f agent-inferred conventions. Added deprecated requirement for agent-inferred conventions. Updated "Agent-Inferred Conventions" design decision as superseded. Updated hook registration delegation table to reflect matcher support. |
| 260306-143f-setup-language-inference | 2026-03-06 | Replaced template-driven language detection (step 1b-lang) with agent-inferred conventions. Deleted `fab/.kit/templates/constitutions/` and `fab/.kit/templates/configs/` (6 template files). Removed language template advisory (section 2b) from `2-sync-workspace.sh`. Added "Agent-Inferred Conventions Replace Templates" design decision. |
| 260305-bs5x-orchestrator-idle-hooks | 2026-03-05 | Added hook registration to delegation table: `fab-sync.sh` via `5-sync-hooks.sh` registers `fab/.kit/hooks/on-*.sh` into `.claude/settings.local.json` hooks (idempotent jq merge). |
| 260223-sr3u-add-fab-doctor | 2026-02-23 | Added Phase 0 prerequisite check: `/fab-setup` (bare bootstrap) runs `fab-doctor.sh` as early gate before creating any project artifacts. Non-zero exit stops bootstrap. Subcommands (config, constitution, migrations) skip this check. |
| 260217-17pe-DEV-1046-scaffold-setup-templates | 2026-02-17 | Extracted inline config.yaml and constitution.md templates from `fab-setup.md` into `scaffold/config.yaml` and `scaffold/constitution.md`. Replaced inline memory-index and specs-index templates with scaffold file references. Updated delegation table notes. Added "Templates in Scaffold Files" design decision. |
| 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow | 2026-02-16 | Renamed `/fab-init` → `/fab-setup`; absorbed `/fab-update` as `migrations` subcommand; promoted `lib/sync-workspace.sh` → `fab-sync.sh`; validate subcommand folded into config/constitution flows; file renamed from init.md to setup.md |
| 260214-m3v8-relocate-docs-dev-scripts | 2026-02-14 | Updated `fab-sync.sh` delegation to create `docs/memory/` and `docs/specs/` instead of `fab/memory/` and `fab/specs/` |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_init_scaffold.sh` → `fab-sync.sh` throughout (directory structure, delegation table, bootstrap references, design decisions, changelog entries) |
| — | 2026-02-14 | Absorbed init-family.md — subcommand architecture, delegation details, and design decisions (memory reorganization) |
| 260213-k7m2-kit-version-migrations | 2026-02-14 | Added `fab/VERSION` to bootstrap steps and delegation table; updated step numbering (1e = VERSION, 1f = changes, 1g = symlinks, 1h = gitignore) |
| 260213-3njv-scaffold-dir | 2026-02-13 | Updated delegation table: `.envrc` → `scaffold/envrc`, `.gitignore` → `scaffold/gitignore-entries`, skeleton files → scaffold sources |
| 260213-3tyk-merge-fab-init-subcommands | 2026-02-13 | Consolidated init family into subcommands — `/fab-setup config`, `/fab-setup constitution`, `/fab-setup validate` are now subcommands of `/fab-setup` |
| 260213-iq2l-rename-setup-scripts | 2026-02-13 | Renamed `fab-setup.sh` → `fab-sync.sh` in delegation pattern and all references |
| 260212-h9k3-fab-init-family | 2026-02-12 | Added Related Commands section, updated Delegation Pattern to reflect `/fab-setup` delegating to `/fab-setup config` and `/fab-setup constitution` |
| 260212-emcb-clarify-fab-setup | 2026-02-12 | Added Delegation Pattern section documenting responsibility split between `/fab-setup` and `fab-sync.sh` |
| 260210-h7r3-kit-distribution-update | 2026-02-10 | Added Bootstrap Alternative section with curl one-liner as alternative to manual `cp -r` |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-bb1q-add-specs-index | 2026-02-07 | Added `docs/specs/index.md` creation as step 1d in bootstrap sequence |
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Simplified to structural bootstrap only — removed Phase 2 source hydration, added argument redirect |
