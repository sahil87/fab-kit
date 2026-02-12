# Spec: Clarify fab-setup Responsibilities and Initialize fab/design Folder

**Change**: 260212-emcb-clarify-fab-setup
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/init.md`, `fab/docs/fab-workflow/distribution.md`

## Non-Goals

- Full consolidation of `/fab-init` and `fab-setup.sh` into a single tool — deferred until boundaries are proven clear
- Changing the interactive config/constitution generation in `/fab-init`
- Modifying the release or update scripts

## fab-setup.sh: Structural Bootstrap

### Requirement: fab-setup.sh SHALL create fab/design/ directory and index.md skeleton

`fab-setup.sh` SHALL create `fab/design/` directory and `fab/design/index.md` when they do not exist, following the same idempotent pattern used for `fab/docs/` and `fab/docs/index.md`.

The `fab/design/index.md` skeleton SHALL match the content defined in `/fab-init` step 1d — the full specifications index with blockquote header and empty table. This ensures that running only `fab-setup.sh` (bootstrap path) produces the same `design/index.md` as running `/fab-init`.

#### Scenario: Fresh bootstrap creates fab/design/

- **GIVEN** `fab/design/` directory does not exist
- **WHEN** `fab-setup.sh` is executed
- **THEN** `fab/design/` directory is created
- **AND** `fab/design/index.md` is created with the specifications index skeleton
- **AND** the script outputs `Created: fab/design/index.md`

#### Scenario: Re-run with existing fab/design/index.md

- **GIVEN** `fab/design/index.md` already exists with user-modified content
- **WHEN** `fab-setup.sh` is executed
- **THEN** `fab/design/index.md` is NOT overwritten
- **AND** no output is produced for the design/index.md step

### Requirement: fab-setup.sh SHALL remain a pure structural bootstrap

`fab-setup.sh` SHALL handle only non-interactive structural setup: directories, skeleton files, symlinks, and `.gitignore` entries. It MUST NOT generate project-specific configuration (`config.yaml`, `constitution.md`) or prompt for user input.

The complete responsibility set for `fab-setup.sh`:

1. Create `fab/changes/` directory with `.gitkeep`
2. Create `fab/docs/` directory and `fab/docs/index.md` skeleton
3. Create `fab/design/` directory and `fab/design/index.md` skeleton *(new)*
4. Create `.envrc` symlink
5. Create skill symlinks for all agent platforms (Claude Code, OpenCode, Codex)
6. Append `fab/current` to `.gitignore`

#### Scenario: fab-setup.sh produces complete structural scaffold

- **GIVEN** a fresh project with only `fab/.kit/` present
- **WHEN** `fab-setup.sh` is executed
- **THEN** all structural artifacts are created (changes/, docs/index.md, design/index.md, symlinks, .envrc, .gitignore entry)
- **AND** no interactive prompts are shown
- **AND** no config.yaml or constitution.md is created

## /fab-init: Delegation Pattern Documentation

### Requirement: fab-init skill SHALL document the delegation pattern

The `/fab-init` skill file (`fab/.kit/skills/fab-init.md`) SHALL clearly document that it delegates structural setup to `fab-setup.sh` and only adds interactive/configuration artifacts on top.

The documented responsibility split:

| Responsibility | Owner | Why |
|---|---|---|
| Directories, skeleton files, symlinks, .gitignore, .envrc | `fab-setup.sh` | Scriptable, automatable, no user input needed |
| `config.yaml` (interactive) | `/fab-init` | Requires project-specific user input |
| `constitution.md` (interactive) | `/fab-init` | Requires understanding of project principles |
| Invoking `fab-setup.sh` | `/fab-init` (step 1f) | Ensures structural setup runs as part of init |

#### Scenario: fab-init delegates to fab-setup.sh

- **GIVEN** a fresh project with only `fab/.kit/` present
- **WHEN** `/fab-init` is executed
- **THEN** `fab-setup.sh` is run (step 1f) to create structural artifacts
- **AND** `/fab-init` creates `config.yaml` and `constitution.md` via interactive prompts
- **AND** `/fab-init` skips creating docs/index.md and design/index.md (already created by fab-setup.sh)

### Requirement: fab-init idempotent checks SHALL account for fab-setup.sh

`/fab-init` steps 1c (docs/index.md), 1d (design/index.md), and 1e (changes/) SHALL continue to check existence before creating, ensuring they gracefully skip when `fab-setup.sh` has already created these artifacts. This is already the case — this requirement documents the existing behavior as intentional.

#### Scenario: fab-init re-run after fab-setup.sh

- **GIVEN** `fab-setup.sh` has already been run (structural artifacts exist)
- **WHEN** `/fab-init` is executed
- **THEN** steps 1c, 1d, and 1e report "already exists — skipping"
- **AND** `fab-setup.sh` (step 1f) reports all symlinks valid

## Documentation Updates

### Requirement: init.md SHALL document the fab-setup.sh delegation pattern

`fab/docs/fab-workflow/init.md` SHALL include a section explaining the delegation relationship between `/fab-init` and `fab-setup.sh`. The section SHALL cover:

- What `fab-setup.sh` creates (structural artifacts)
- What `/fab-init` creates on top (interactive configuration)
- That `/fab-init` invokes `fab-setup.sh` as step 1f
- That running `fab-setup.sh` alone is valid (bootstrap path)

#### Scenario: Updated init.md explains delegation

- **GIVEN** a user reads `fab/docs/fab-workflow/init.md`
- **WHEN** they look for the relationship between `/fab-init` and `fab-setup.sh`
- **THEN** they find a clear explanation of which tool creates what
- **AND** they understand that `fab-setup.sh` can be run independently

### Requirement: distribution.md SHALL reflect fab/design/ in bootstrap

`fab/docs/fab-workflow/distribution.md` SHALL update its bootstrap scenarios to reflect that `fab-setup.sh` now creates `fab/design/` directory and `fab/design/index.md`.

#### Scenario: Updated bootstrap scenario

- **GIVEN** a user reads the bootstrap section of distribution.md
- **WHEN** they review what `fab-setup.sh` creates
- **THEN** they see `fab/design/` and `fab/design/index.md` listed alongside other structural artifacts

## Design Decisions

1. **Skeleton content matches fab-init**: `fab-setup.sh` creates the same `fab/design/index.md` content that `/fab-init` step 1d specifies
   - *Why*: Since fab-init checks existence and skips, both paths must produce identical output. Using the same content avoids drift.

2. **Placement in fab-setup.sh**: The new `fab/design/` section goes between the existing docs/index.md section (3) and the skill symlinks section (4)
   - *Why*: Groups all directory/skeleton creation together before the symlink and gitignore steps.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Doc updates add a "Delegation Pattern" section to init.md rather than restructuring the entire doc | The existing init.md structure is clear; an additive section explaining the relationship is sufficient and least disruptive |

1 assumption made (1 confident, 0 tentative). Run `/fab-clarify` to review.
