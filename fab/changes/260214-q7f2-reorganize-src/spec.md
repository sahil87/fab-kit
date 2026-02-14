# Spec: Reorganize src/ and kit script internals

**Change**: 260214-q7f2-reorganize-src
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/kit-architecture.md`, `fab/memory/fab-workflow/distribution.md`, `fab/memory/fab-workflow/preflight.md`, `fab/memory/fab-workflow/context-loading.md`, `fab/memory/fab-workflow/planning-skills.md`

## Non-Goals

- Changing any script logic or behavior — this is a pure structural reorganization
- Modifying the scaffold `.envrc` that ships to end users (only the repo's dev `.envrc` changes)
- Renaming user-facing scripts (`fab-help.sh`, `fab-upgrade.sh`, `fab-release.sh`, batch scripts)

## src/: Test infrastructure to lib/

### Requirement: src/lib/ contains all test infrastructure

All existing test infrastructure directories (`calc-score`, `preflight`, `resolve-change`, `stageman`) SHALL be located under `src/lib/` instead of directly under `src/`.

#### Scenario: Directory structure after move

- **GIVEN** the current `src/` contains `calc-score/`, `preflight/`, `resolve-change/`, `stageman/` at the top level
- **WHEN** the reorganization is applied
- **THEN** all four directories exist under `src/lib/` (`src/lib/calc-score/`, `src/lib/preflight/`, `src/lib/resolve-change/`, `src/lib/stageman/`)
- **AND** `src/` no longer contains these directories at the top level

### Requirement: src/scripts/ for dev-only scripts

A `src/scripts/` directory SHALL be created for dev-only scripts that are not part of the shipped kit.

#### Scenario: fab-release.sh relocated

- **GIVEN** `fab/.kit/scripts/fab-release.sh` exists as a kit script
- **WHEN** the reorganization is applied
- **THEN** `fab-release.sh` exists at `src/scripts/fab-release.sh`
- **AND** `fab/.kit/scripts/fab-release.sh` no longer exists
- **AND** the release tarball no longer includes `fab-release.sh`

### Requirement: Justfile test glob updated

The `justfile` test recipe SHALL glob `src/lib/*/test.sh` instead of `src/*/test.sh`.

#### Scenario: Tests still discovered after move

- **GIVEN** test infrastructure has been moved to `src/lib/`
- **WHEN** `just test` is executed
- **THEN** all four test suites (`calc-score`, `preflight`, `resolve-change`, `stageman`) are discovered and run
- **AND** `src/scripts/` contents are NOT matched by the test glob

### Requirement: src/lib/ symlinks updated

Symlinks in each `src/lib/*/` directory SHALL point to the renamed scripts in `fab/.kit/scripts/lib/` with the correct relative depth.

#### Scenario: Symlink targets after move

- **GIVEN** `src/lib/calc-score/_calc-score.sh` currently symlinks to `../../fab/.kit/scripts/_calc-score.sh`
- **WHEN** both `src/` and `.kit/scripts/` reorganizations are applied
- **THEN** `src/lib/calc-score/calc-score.sh` symlinks to `../../../fab/.kit/scripts/lib/calc-score.sh`
- **AND** `src/lib/preflight/preflight.sh` symlinks to `../../../fab/.kit/scripts/lib/preflight.sh`
- **AND** `src/lib/resolve-change/resolve-change.sh` symlinks to `../../../fab/.kit/scripts/lib/resolve-change.sh`
- **AND** `src/lib/stageman/stageman.sh` symlinks to `../../../fab/.kit/scripts/lib/stageman.sh`

#### Scenario: Symlink names drop underscore prefix

- **GIVEN** symlinks currently use underscore-prefixed names (e.g., `_calc-score.sh`)
- **WHEN** the reorganization is applied
- **THEN** all symlinks in `src/lib/*/` use unprefixed names matching their targets in `fab/.kit/scripts/lib/`

### Requirement: Dev .envrc adds src/scripts to PATH

The repository's `.envrc` (not the scaffold `.envrc`) SHALL add `src/scripts` to PATH via `PATH_add`.

#### Scenario: fab-release.sh accessible after move

- **GIVEN** `fab-release.sh` has been moved to `src/scripts/`
- **WHEN** the developer's shell has loaded the `.envrc`
- **THEN** `fab-release.sh` is accessible on PATH without a full path

#### Scenario: Scaffold .envrc unchanged

- **GIVEN** the scaffold `.envrc` at `fab/.kit/scaffold/envrc`
- **WHEN** the reorganization is applied
- **THEN** the scaffold `.envrc` content is unchanged (still contains only `IDEAS_FILE`, `WORKTREE_INIT_SCRIPT`, and `PATH_add fab/.kit/scripts`)

## .kit/scripts/: Internal scripts to lib/

### Requirement: Internal scripts relocated to lib/ subfolder

All underscore-prefixed internal scripts SHALL be moved from `fab/.kit/scripts/` to `fab/.kit/scripts/lib/` with the underscore prefix dropped:

| Current path | New path |
|---|---|
| `fab/.kit/scripts/_calc-score.sh` | `fab/.kit/scripts/lib/calc-score.sh` |
| `fab/.kit/scripts/_preflight.sh` | `fab/.kit/scripts/lib/preflight.sh` |
| `fab/.kit/scripts/_stageman.sh` | `fab/.kit/scripts/lib/stageman.sh` |
| `fab/.kit/scripts/_resolve-change.sh` | `fab/.kit/scripts/lib/resolve-change.sh` |
| `fab/.kit/scripts/_init_scaffold.sh` | `fab/.kit/scripts/lib/init-scaffold.sh` |

#### Scenario: lib/ folder signals "internal"

- **GIVEN** the `lib/` subfolder has been created
- **WHEN** a developer lists `fab/.kit/scripts/`
- **THEN** only user-facing scripts (`fab-help.sh`, `fab-upgrade.sh`, batch scripts) and the `lib/` directory are visible at the top level
- **AND** internal scripts are grouped inside `lib/`

#### Scenario: No underscore prefix in lib/

- **GIVEN** scripts have been moved to `fab/.kit/scripts/lib/`
- **WHEN** the scripts are listed
- **THEN** none use an underscore prefix — the `lib/` folder replaces the prefix as the "internal" signal

### Requirement: Inter-script source references updated

All scripts that source or call internal scripts SHALL be updated to reference the new `lib/` paths.

#### Scenario: preflight.sh sources from lib/

- **GIVEN** `lib/preflight.sh` currently sources `_stageman.sh` and `_resolve-change.sh` via `$scripts_dir/_stageman.sh`
- **WHEN** the reorganization is applied
- **THEN** `lib/preflight.sh` sources `"$scripts_dir/lib/stageman.sh"` and `"$scripts_dir/lib/resolve-change.sh"` (where `$scripts_dir` resolves to the parent `scripts/` directory)

#### Scenario: calc-score.sh sources stageman from lib/

- **GIVEN** `_calc-score.sh` currently sources `_stageman.sh` via dirname-based resolution
- **WHEN** the reorganization is applied
- **THEN** `lib/calc-score.sh` sources `lib/stageman.sh` via dirname-based resolution (both are now in `lib/`)

#### Scenario: fab-upgrade.sh calls init-scaffold from lib/

- **GIVEN** `fab-upgrade.sh` currently calls `$kit_dir/scripts/_init_scaffold.sh`
- **WHEN** the reorganization is applied
- **THEN** `fab-upgrade.sh` calls `$kit_dir/scripts/lib/init-scaffold.sh`
- **AND** the echo message references `lib/init-scaffold.sh`

#### Scenario: batch-archive-change.sh sources resolve-change from lib/

- **GIVEN** `batch-archive-change.sh` currently sources `${SCRIPT_DIR}/_resolve-change.sh`
- **WHEN** the reorganization is applied
- **THEN** `batch-archive-change.sh` sources `${SCRIPT_DIR}/lib/resolve-change.sh`

#### Scenario: batch-switch-change.sh sources resolve-change from lib/

- **GIVEN** `batch-switch-change.sh` currently sources `${SCRIPT_DIR}/_resolve-change.sh`
- **WHEN** the reorganization is applied
- **THEN** `batch-switch-change.sh` sources `${SCRIPT_DIR}/lib/resolve-change.sh`

### Requirement: Skill file references updated

All skill files that reference internal script paths SHALL be updated to use the new `lib/` paths.

#### Scenario: _context.md preflight reference

- **GIVEN** `_context.md` references `fab/.kit/scripts/_preflight.sh`
- **WHEN** the reorganization is applied
- **THEN** `_context.md` references `fab/.kit/scripts/lib/preflight.sh`
- **AND** the `_calc-score.sh` reference becomes `fab/.kit/scripts/lib/calc-score.sh`

#### Scenario: fab-continue.md calc-score reference

- **GIVEN** `fab-continue.md` references `fab/.kit/scripts/_calc-score.sh`
- **WHEN** the reorganization is applied
- **THEN** `fab-continue.md` references `fab/.kit/scripts/lib/calc-score.sh`

#### Scenario: fab-continue.md stageman CLI references

- **GIVEN** `fab-continue.md` references `_stageman.sh` as a CLI command
- **WHEN** the reorganization is applied
- **THEN** all `_stageman.sh` CLI references in `fab-continue.md` become `lib/stageman.sh`

#### Scenario: fab-init.md init-scaffold reference

- **GIVEN** `fab-init.md` references `fab/.kit/scripts/_init_scaffold.sh`
- **WHEN** the reorganization is applied
- **THEN** `fab-init.md` references `fab/.kit/scripts/lib/init-scaffold.sh`

#### Scenario: fab-status.md preflight reference

- **GIVEN** `fab-status.md` references `fab/.kit/scripts/_preflight.sh`
- **WHEN** the reorganization is applied
- **THEN** `fab-status.md` references `fab/.kit/scripts/lib/preflight.sh`

#### Scenario: fab-archive.md preflight reference

- **GIVEN** `fab-archive.md` references `fab/.kit/scripts/_preflight.sh`
- **WHEN** the reorganization is applied
- **THEN** `fab-archive.md` references `fab/.kit/scripts/lib/preflight.sh`

#### Scenario: fab-clarify.md calc-score reference

- **GIVEN** `fab-clarify.md` references `fab/.kit/scripts/_calc-score.sh`
- **WHEN** the reorganization is applied
- **THEN** `fab-clarify.md` references `fab/.kit/scripts/lib/calc-score.sh`

#### Scenario: fab-ff.md and fab-fff.md stageman CLI references

- **GIVEN** `fab-ff.md` and `fab-fff.md` reference `_stageman.sh` as a CLI command
- **WHEN** the reorganization is applied
- **THEN** all `_stageman.sh` CLI references become `lib/stageman.sh`

#### Scenario: _generation.md stageman CLI references

- **GIVEN** `_generation.md` references `_stageman.sh` CLI commands for checklist updates
- **WHEN** the reorganization is applied
- **THEN** all `_stageman.sh` references become `lib/stageman.sh`

## Deprecated Requirements

### Underscore Prefix Convention for Internal Scripts

**Reason**: Replaced by `lib/` subfolder convention. The `lib/` directory now signals "internal" — scripts within it do not need a naming prefix.
**Migration**: All `_`-prefixed scripts move to `fab/.kit/scripts/lib/` with the prefix dropped.

## Design Decisions

1. **`lib/` over `internal/` or `_/`**: Using `lib/` as the subfolder name.
   - *Why*: `lib/` is a widely understood convention across ecosystems (Node, Python, Ruby) for "support code consumed by other code." It matches the same pattern being applied to `src/lib/`.
   - *Rejected*: `internal/` — too verbose for frequent path references. `_/` — directory names starting with underscore have tooling quirks.

2. **Move fab-release.sh to src/scripts/ (not delete)**: The release script is dev-only but still needed.
   - *Why*: `fab-release.sh` packages and publishes new kit versions — essential for development but never needed by end users. Moving to `src/scripts/` keeps it accessible while removing it from the shipped tarball.
   - *Rejected*: Deleting it — still actively used for releases. Keeping in `.kit/scripts/` with a flag — adds complexity to the tarball build.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Symlink names in `src/lib/*/` drop underscore prefix to match renamed targets | Brief explicitly states "Drop `_` prefix — the `lib/` folder now signals internal". Consistent naming between symlink and target is the obvious default. |
| 2 | Confident | `_stageman.sh` CLI references in skill files also get the `lib/` prefix | The brief lists skill reference updates for `_preflight.sh` and `_calc-score.sh` paths; `_stageman.sh` CLI invocations follow the same pattern. The brief says "all need updating." |
| 3 | Confident | `fab-update-claude-settings.sh` stays in `fab/.kit/scripts/` (not moved to lib/) | It is not underscore-prefixed and is a utility script, not an internal library. Brief's scope is limited to `_`-prefixed scripts. |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.
