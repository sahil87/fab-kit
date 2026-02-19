# Spec: Move 5 Cs and VERSION into fab/project/

**Change**: 260219-wq0e-move-5cs-to-project-folder
**Created**: 2026-02-19
**Affected memory**: `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/context-loading.md`, `docs/memory/fab-workflow/setup.md`, `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/preflight.md`, `docs/memory/fab-workflow/migrations.md`

## Non-Goals

- Changing the semantics or content of any of the 6 files being moved — this is a pure relocation
- Moving `backlog.md`, `current`, or `changes/` — these stay in `fab/` root
- Moving `fab/sync/` — this is project-specific sync, not project identity
- Changing the 5 Cs loading order or adding/removing files from the "Always Load" layer

## Directory Structure

### Requirement: fab/project/ directory

The 6 project identity files SHALL reside in `fab/project/` instead of `fab/`:

| Old path | New path |
|----------|----------|
| `fab/config.yaml` | `fab/project/config.yaml` |
| `fab/constitution.md` | `fab/project/constitution.md` |
| `fab/context.md` | `fab/project/context.md` |
| `fab/code-quality.md` | `fab/project/code-quality.md` |
| `fab/code-review.md` | `fab/project/code-review.md` |
| `fab/VERSION` | `fab/project/VERSION` |

The resulting `fab/` top-level SHALL be:

```
fab/
├── .kit/           # the tool (engine)
├── project/        # project identity (5 Cs + VERSION)
├── changes/        # active work
├── sync/           # project-specific sync scripts (if any)
├── backlog.md      # idea queue
└── current         # active change pointer
```

#### Scenario: Clean directory hierarchy
- **GIVEN** a project with `fab/project/` containing the 6 files
- **WHEN** a user lists the `fab/` directory
- **THEN** they see a clear triad: `.kit/` (engine), `project/` (identity), `changes/` (work)
- **AND** no configuration files are scattered at the `fab/` root level

#### Scenario: fab/current stays in fab/ root
- **GIVEN** a project using the new layout
- **WHEN** `/fab-switch` writes `fab/current`
- **THEN** the pointer file is at `fab/current` (unchanged from before)

## Shell Script Updates

### Requirement: preflight.sh path updates

`fab/.kit/scripts/lib/preflight.sh` SHALL check for `fab/project/config.yaml` and `fab/project/constitution.md` (instead of `fab/config.yaml` and `fab/constitution.md`) when validating project initialization.

#### Scenario: Preflight validates new paths
- **GIVEN** a project with files at `fab/project/config.yaml` and `fab/project/constitution.md`
- **WHEN** `preflight.sh` runs
- **THEN** it passes the initialization check (exit 0 with valid YAML output)

#### Scenario: Preflight fails on missing project files
- **GIVEN** a project with no `fab/project/config.yaml`
- **WHEN** `preflight.sh` runs
- **THEN** it exits non-zero with a diagnostic message about missing config

### Requirement: changeman.sh path updates

`fab/.kit/scripts/lib/changeman.sh` SHALL read `$FAB_ROOT/project/config.yaml` (instead of `$FAB_ROOT/config.yaml`) for git integration settings in its `switch` subcommand.

#### Scenario: changeman reads config from new path
- **GIVEN** `fab/project/config.yaml` contains `git.enabled: true` and `git.branch_prefix: "feat/"`
- **WHEN** `changeman.sh switch <name>` runs
- **THEN** it reads git settings from `fab/project/config.yaml`
- **AND** creates/checks out a branch with the `feat/` prefix

### Requirement: fab-upgrade.sh path updates

`fab/.kit/scripts/fab-upgrade.sh` SHALL check `fab/project/VERSION` (instead of `fab/VERSION`) for version drift detection after upgrade.

#### Scenario: Upgrade detects version drift at new path
- **GIVEN** `fab/project/VERSION` contains `0.9.0` and `fab/.kit/VERSION` contains `0.10.0`
- **WHEN** `fab-upgrade.sh` completes
- **THEN** it prints a reminder to run `/fab-setup migrations`

### Requirement: fab-help.sh path updates

`fab/.kit/scripts/fab-help.sh` SHALL read `fab/project/VERSION` (instead of `fab/VERSION`) when displaying the version in help output.
<!-- assumed: fab-help.sh reads VERSION for display — confirmed by checking kit-architecture memory showing fab-help.sh reads VERSION -->

#### Scenario: Help shows version from new path
- **GIVEN** `fab/project/VERSION` contains `0.10.0`
- **WHEN** `fab-help.sh` runs
- **THEN** it displays `0.10.0` in the help output

### Requirement: batch script path updates

`fab/.kit/scripts/batch-fab-switch-change.sh` SHALL read `fab/project/config.yaml` (instead of `fab/config.yaml`) for configuration.

#### Scenario: Batch switch reads config from new path
- **GIVEN** `fab/project/config.yaml` exists with git settings
- **WHEN** `batch-fab-switch-change.sh` runs
- **THEN** it reads configuration from `fab/project/config.yaml`

### Requirement: fab-sync.sh and sync scripts path updates

`fab/.kit/sync/2-sync-workspace.sh` SHALL:
- Read `fab/project/VERSION` (instead of `fab/VERSION`) for version creation/preservation
- Detect existing projects by checking for `fab/project/config.yaml` (instead of `fab/config.yaml`)
- Place scaffold files into `fab/project/` (instead of `fab/`)

#### Scenario: Sync creates VERSION at new path for new project
- **GIVEN** a new project with no `fab/project/config.yaml`
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** it creates `fab/project/VERSION` with the engine version from `fab/.kit/VERSION`

#### Scenario: Sync preserves existing VERSION at new path
- **GIVEN** `fab/project/VERSION` already exists with `0.9.0`
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** `fab/project/VERSION` remains `0.9.0`

#### Scenario: Sync detects existing project via new config path
- **GIVEN** `fab/project/config.yaml` exists but `fab/project/VERSION` does not
- **WHEN** `2-sync-workspace.sh` runs
- **THEN** it writes `0.1.0` to `fab/project/VERSION` (base version for migration chain)

## Scaffold Updates

### Requirement: Scaffold overlay tree restructuring

The scaffold overlay tree SHALL place the 5 Cs under `fab/project/` instead of `fab/`:

| Old scaffold path | New scaffold path |
|-------------------|-------------------|
| `fab/.kit/scaffold/fab/config.yaml` | `fab/.kit/scaffold/fab/project/config.yaml` |
| `fab/.kit/scaffold/fab/constitution.md` | `fab/.kit/scaffold/fab/project/constitution.md` |
| `fab/.kit/scaffold/fab/context.md` | `fab/.kit/scaffold/fab/project/context.md` |
| `fab/.kit/scaffold/fab/code-quality.md` | `fab/.kit/scaffold/fab/project/code-quality.md` |
| `fab/.kit/scaffold/fab/code-review.md` | `fab/.kit/scaffold/fab/project/code-review.md` |

The `fab/.kit/scaffold/fab/sync/` directory SHALL remain at its current location (unchanged).

#### Scenario: Scaffold tree-walk places files in fab/project/
- **GIVEN** a new project with no existing `fab/project/` directory
- **WHEN** `2-sync-workspace.sh` runs the scaffold tree-walk
- **THEN** it creates `fab/project/` and copies the 5 Cs into it via copy-if-absent
- **AND** `fab/project/config.yaml` contains the template with placeholders

#### Scenario: Scaffold preserves existing project files
- **GIVEN** `fab/project/config.yaml` already exists with user content
- **WHEN** `2-sync-workspace.sh` runs the scaffold tree-walk
- **THEN** `fab/project/config.yaml` is NOT overwritten (copy-if-absent semantics)

### Requirement: fab-setup.md template detection update

`fab-setup.md` SHALL detect raw templates by checking for placeholder strings in `fab/project/config.yaml` and `fab/project/constitution.md` (instead of `fab/config.yaml` and `fab/constitution.md`).

#### Scenario: Setup detects unfinished template at new path
- **GIVEN** `fab/project/config.yaml` contains `{PROJECT_NAME}` placeholder
- **WHEN** `/fab-setup` runs
- **THEN** it enters interactive config generation mode (overwrites the template)

## Skill Updates

### Requirement: _context.md Always Load layer path updates

`fab/.kit/skills/_context.md` SHALL reference the 5 Cs at their new `fab/project/` paths in the "Always Load" section:

1. `fab/project/config.yaml`
2. `fab/project/constitution.md`
3. `fab/project/context.md` *(optional)*
4. `fab/project/code-quality.md` *(optional)*
5. `fab/project/code-review.md` *(optional)*
6. `docs/memory/index.md`
7. `docs/specs/index.md`

The state derivation rule for **(none)** SHALL check `fab/project/config.yaml` (instead of `fab/config.yaml`).

#### Scenario: Skills load context from new paths
- **GIVEN** a skill reads `_context.md` and follows the Always Load instructions
- **WHEN** it loads the baseline context files
- **THEN** it reads from `fab/project/config.yaml`, `fab/project/constitution.md`, etc.

#### Scenario: State derivation uses new config path
- **GIVEN** `fab/project/config.yaml` does not exist
- **WHEN** a skill derives state per the state table
- **THEN** the state is `(none)` and the next step is `/fab-setup`

### Requirement: All skill path references updated

Every skill file in `fab/.kit/skills/` that references `fab/config.yaml`, `fab/constitution.md`, `fab/context.md`, `fab/code-quality.md`, `fab/code-review.md`, or `fab/VERSION` SHALL be updated to use the `fab/project/` prefix. This includes:

- `fab-setup.md` — bootstrap paths, migration references, config/constitution creation paths
- `fab-continue.md` — apply/review context loading references
- `fab-new.md` — pre-flight references
- `fab-status.md` — VERSION and config.yaml reads
- `fab-switch.md` — config.yaml loading reference
- `fab-ff.md`, `fab-fff.md`, `fab-clarify.md`, `fab-archive.md` — via `_context.md` (transitively correct) plus any direct references
- `_generation.md` — `fab/code-quality.md`, `fab/config.yaml` references
- `internal-consistency-check.md` — `fab/config.yaml` reference

#### Scenario: Skill references are consistent
- **GIVEN** all skill files have been updated
- **WHEN** searching for `fab/config.yaml` (without `project/` prefix) across `fab/.kit/skills/`
- **THEN** no matches are found (excluding changelog/history references in generated artifacts)

### Requirement: Scaffold path references in skills updated

Any skill that references scaffold paths (e.g., `fab/.kit/scaffold/fab/config.yaml`) SHALL be updated to reference `fab/.kit/scaffold/fab/project/config.yaml`.

#### Scenario: Setup skill references new scaffold paths
- **GIVEN** `fab-setup.md` references scaffold templates
- **WHEN** it reads the config.yaml template
- **THEN** it reads from `fab/.kit/scaffold/fab/project/config.yaml`

## Agent File Updates

### Requirement: Agent files regenerated

After skill updates, agent files in `.claude/agents/` SHALL be regenerated by `2-sync-workspace.sh` to reflect the new paths. During this change, both skills (source) and agents (generated) SHALL be updated to maintain consistency.

#### Scenario: Agent files reflect new paths
- **GIVEN** skill files reference `fab/project/config.yaml`
- **WHEN** `2-sync-workspace.sh` regenerates agent files
- **THEN** agent files also reference `fab/project/config.yaml`

## Documentation Updates

### Requirement: README and CONTRIBUTING updates

`README.md` and `CONTRIBUTING.md` (if present) SHALL be updated to reference the new `fab/project/` paths in:
- Directory structure examples
- Quick start instructions
- File reference tables

#### Scenario: README shows new directory structure
- **GIVEN** the README describes the `fab/` directory layout
- **WHEN** a user reads the directory structure section
- **THEN** it shows `fab/project/` containing the 5 Cs and VERSION

### Requirement: Memory file path updates

All memory files in `docs/memory/fab-workflow/` that reference the old paths SHALL be updated:
- `configuration.md` — all `fab/{file}` → `fab/project/{file}` paths; document `fab/project/` purpose
- `context-loading.md` — Always Load file paths
- `setup.md` — bootstrap file creation paths
- `kit-architecture.md` — directory structure diagrams, preserved file list
- `distribution.md` — preserved file paths
- `preflight.md` — existence check paths
- `migrations.md` — VERSION path references

#### Scenario: Memory files reference new paths
- **GIVEN** all memory files have been updated
- **WHEN** searching for `fab/config.yaml` (without `project/` prefix) across `docs/memory/`
- **THEN** no matches are found outside of changelog entries

### Requirement: Spec file updates

Spec files in `docs/specs/` that reference the old paths SHALL be updated.

#### Scenario: Spec references updated
- **GIVEN** `docs/specs/architecture.md` references `fab/config.yaml`
- **WHEN** the change is applied
- **THEN** it references `fab/project/config.yaml`

## Migration

### Requirement: Migration file 0.9.0-to-0.10.0.md

A migration file `fab/.kit/migrations/0.9.0-to-0.10.0.md` SHALL be created following the established migration pattern. It SHALL handle existing installs that have files at the old `fab/` locations.

**Pre-check**: At least one of the 6 files exists at the old `fab/` location. If all are already at `fab/project/`, skip (already migrated).

**Changes**:
1. Create `fab/project/` directory if it doesn't exist
2. For each of the 6 files (`config.yaml`, `constitution.md`, `context.md`, `code-quality.md`, `code-review.md`, `VERSION`): move from `fab/{file}` to `fab/project/{file}`, skipping any that don't exist at the old location or already exist at the new location

**Verification**:
- `fab/project/config.yaml` exists
- `fab/project/constitution.md` exists
- None of the 6 files exist at the old `fab/` location (unless absent to begin with)

#### Scenario: Migration moves files from old to new location
- **GIVEN** an existing project with `fab/config.yaml`, `fab/constitution.md`, `fab/VERSION`, and optionally `fab/context.md`
- **WHEN** `/fab-setup migrations` applies `0.9.0-to-0.10.0.md`
- **THEN** `fab/project/` is created
- **AND** each file is moved to `fab/project/{file}`
- **AND** no files remain at the old `fab/` location

#### Scenario: Migration is idempotent
- **GIVEN** a project already migrated (files at `fab/project/`)
- **WHEN** the migration pre-check runs
- **THEN** it skips (no old-path files found)

#### Scenario: Migration handles partial state
- **GIVEN** a project where `fab/context.md` doesn't exist (optional file was never created)
- **WHEN** the migration runs
- **THEN** it moves the files that exist and skips `context.md` without error

## Version Bump

### Requirement: Engine VERSION update

`fab/.kit/VERSION` SHALL be bumped to `0.10.0` to reflect this breaking change. The migration filename (`0.9.0-to-0.10.0.md`) SHALL match the version transition.

#### Scenario: VERSION reflects new version
- **GIVEN** `fab/.kit/VERSION` is updated
- **WHEN** a user reads the file
- **THEN** it contains `0.10.0`

## Design Decisions

1. **`fab/project/` as the subfolder name**: Chosen explicitly by the user over alternatives (`guide/`, `profile/`, `rails/`, `config/`). `project/` honestly describes the contents (project-level definitions), is short, and creates a clean triad with `.kit/` (engine) and `changes/` (work). `config/` was rejected because `config/config.yaml` is redundant.
   - *Rejected*: `guide/`, `profile/`, `rails/` — less descriptive or misleading; `config/` — redundant nesting

2. **VERSION moves with the 5 Cs**: VERSION is a project-level file (tracks the project's migration state), not engine state. It belongs with the other project identity files.
   - *Rejected*: Keeping VERSION in `fab/` root — inconsistent grouping, VERSION is conceptually part of project identity

3. **Migration handles the move for existing installs**: Rather than requiring manual file moves, the migration automates it. Follows the established `0.7.0-to-0.8.0.md` pattern.
   - *Rejected*: Manual migration — error-prone, especially with 6 files across many projects

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Folder name is `project/` | Explicitly discussed and chosen by user over alternatives — confirmed from intake #1 | S:95 R:90 A:95 D:95 |
| 2 | Certain | All 6 files move (5 Cs + VERSION) | User explicitly stated VERSION should move too — confirmed from intake #2 | S:95 R:85 A:90 D:95 |
| 3 | Certain | `backlog.md` stays in `fab/` root | User confirmed; not one of the 5 Cs — confirmed from intake #3 | S:95 R:90 A:90 D:95 |
| 4 | Certain | `fab/current` pointer stays in `fab/` root | Workflow state, not project identity; no script references it via config paths — upgraded from intake Confident #4 | S:70 R:90 A:90 D:90 |
| 5 | Confident | Agent files updated alongside skills | Sync generates agents from skills, both need to be in sync for this commit — confirmed from intake #5 | S:75 R:80 A:75 D:80 |
| 6 | Certain | Migration `0.9.0-to-0.10.0.md` moves files | User confirmed; follows established migration pattern — confirmed from intake #6 | S:90 R:85 A:90 D:90 |
| 7 | Confident | Scaffold overlay mirrors new structure | `fab/.kit/scaffold/fab/` gains a `project/` subdirectory; tree-walk handles it generically — confirmed from intake #7 | S:70 R:80 A:80 D:80 |
| 8 | Certain | `fab/sync/` directory stays in `fab/` root | Not project identity; it's project-specific sync scripts. No mention of moving it | S:85 R:90 A:90 D:90 |
| 9 | Confident | Config header comment updated | `fab/.kit/scaffold/fab/project/config.yaml` header says `# fab/project/config.yaml` | S:65 R:90 A:80 D:85 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).
