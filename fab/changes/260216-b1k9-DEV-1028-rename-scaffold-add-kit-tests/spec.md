# Spec: Rename Scaffold & Add Kit Script Tests

**Change**: 260216-b1k9-DEV-1028-rename-scaffold-add-kit-tests
**Created**: 2026-02-16
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md` (modify), `docs/memory/fab-workflow/distribution.md` (modify)

## Non-Goals

- Migrating existing `test.sh` files to bats — separate follow-up (DEV-1029)
- Updating archived change artifacts (`fab/changes/archive/`) — immutable historical records
- Adding Rust test infrastructure beyond a placeholder recipe

## Script Rename: init-scaffold.sh to sync-workspace.sh

### Requirement: File Rename

The file `fab/.kit/scripts/lib/init-scaffold.sh` SHALL be renamed to `fab/.kit/scripts/lib/sync-workspace.sh`. The script's internal header comment and description line SHALL be updated to reflect the new name.

#### Scenario: Script exists at new path after rename
- **GIVEN** the rename has been applied
- **WHEN** a caller invokes `fab/.kit/scripts/lib/sync-workspace.sh`
- **THEN** the script executes identically to the former `init-scaffold.sh`
- **AND** `fab/.kit/scripts/lib/init-scaffold.sh` no longer exists

### Requirement: Worktree Hook Rename

The worktree hook file `fab/.kit/worktree-init-common/2-rerun-init-scaffold.sh` SHALL be renamed to `fab/.kit/worktree-init-common/2-rerun-sync-workspace.sh`. The call inside the hook SHALL reference `sync-workspace.sh` instead of `init-scaffold.sh`.

#### Scenario: Worktree hook invokes renamed script
- **GIVEN** the worktree hook has been renamed and updated
- **WHEN** the worktree init process runs `2-rerun-sync-workspace.sh`
- **THEN** it invokes `fab/.kit/scripts/lib/sync-workspace.sh`
- **AND** the old hook file `2-rerun-init-scaffold.sh` no longer exists

### Requirement: Codebase-Wide Reference Update

All references to `init-scaffold.sh` across the codebase SHALL be updated to `sync-workspace.sh`. This includes kit scripts, skills, memory files, and README.md. The following locations are affected:

**Kit scripts and config:**
- `fab/.kit/scripts/lib/changeman.sh` (line 128 comment)
- `fab/.kit/scripts/fab-upgrade.sh` (lines 7, 96, 97 — comment + echo + invocation)
- `fab/.kit/model-tiers.yaml` (line 4 comment)

**Skills:**
- `fab/.kit/skills/fab-init.md` (lines 48, 106, 124 — delegation references)

**Memory files (`docs/memory/fab-workflow/`):**
- `distribution.md` — 7 references
- `kit-architecture.md` — 10+ references
- `init.md` — 15+ references
- `model-tiers.md` — 4 references
- `templates.md` — 2 references
- `hydrate.md` — 2 references
- `preflight.md` — 1 reference
- `migrations.md` — 3 references
- `index.md` — 1 reference

**README.md:**
- Lines 32 and 137 — directory tree and setup command

#### Scenario: No stale references remain
- **GIVEN** all reference updates have been applied
- **WHEN** searching the codebase for `init-scaffold` (excluding `fab/changes/archive/`)
- **THEN** zero matches are found
- **AND** searching for `sync-workspace` returns matches in all locations listed above

#### Scenario: Archived changes are not modified
- **GIVEN** archived changes in `fab/changes/archive/` contain historical references to `init-scaffold.sh`
- **WHEN** the rename is applied
- **THEN** files under `fab/changes/archive/` remain untouched

## Bats-Core Adoption

### Requirement: Bats Available for Test Execution

Bats-core SHALL be available on the system PATH for running `.bats` test files. The project SHALL document bats-core as a dev prerequisite (alongside existing prerequisites like `yq` and `gh`).
<!-- assumed: bats expected on PATH rather than vendored — consistent with yq/gh precedent of single-binary dev tools on PATH, aligns with Constitution I (Pure Prompt Play) -->

#### Scenario: Bats is available
- **GIVEN** `bats` is installed on the system (via `brew install bats-core`, `apt install bats`, or equivalent)
- **WHEN** the test runner invokes `bats <file>.bats`
- **THEN** bats executes the test file and reports results

#### Scenario: Bats is not installed
- **GIVEN** `bats` is not on PATH
- **WHEN** the test runner attempts to execute `.bats` files
- **THEN** the runner reports a clear error indicating bats-core is required

## Test Suite: sync-workspace

### Requirement: Directory Structure

A test directory `src/lib/sync-workspace/` SHALL be created following the established `src/lib/*/` pattern:

```
src/lib/sync-workspace/
  sync-workspace.sh → ../../../fab/.kit/scripts/lib/sync-workspace.sh  (symlink)
  SPEC-sync-workspace.md   (spec file)
  test.bats                (bats-core test suite)
```

#### Scenario: Symlink points to distributed script
- **GIVEN** the test directory has been set up
- **WHEN** resolving `src/lib/sync-workspace/sync-workspace.sh`
- **THEN** it points to `fab/.kit/scripts/lib/sync-workspace.sh`
- **AND** the target file exists and is executable

### Requirement: SPEC File

`src/lib/sync-workspace/SPEC-sync-workspace.md` SHALL follow the format of existing SPEC files (e.g., `SPEC-stageman.md`): Sources of Truth, Usage, API/Behavior Reference, Requirements, Testing sections.

#### Scenario: SPEC file covers key behavioral areas
- **GIVEN** the SPEC file has been written
- **WHEN** reading the document
- **THEN** it documents: directory creation, VERSION file logic, .envrc symlink, memory/specs index seeding, skill sync (3 platforms), model-tier agent generation, .gitignore management, and idempotency guarantees

### Requirement: Test Coverage

The `test.bats` suite SHALL cover the following behavioral areas of `sync-workspace.sh`:

1. **Directory creation** — `fab/changes`, `docs/memory`, `docs/specs` are created when missing
2. **VERSION file logic** — new project gets engine version; existing project (with `config.yaml`) gets `0.1.0`; existing `fab/VERSION` is preserved
3. **.envrc symlink** — creation, repair of broken symlink, replacement of regular file
4. **Memory/specs index seeding** — `docs/memory/index.md` and `docs/specs/index.md` created from scaffold templates when missing
5. **Skill sync** — symlinks created for Claude Code (directory-based), OpenCode (flat file), Codex (copy-based)
6. **Model-tier agent generation** — fast-tier skills get generated agent files with translated `model:` field
7. **.gitignore management** — creation of new file, dedup of existing entries, append of missing entries
8. **Idempotency** — running twice produces the same result with no errors

#### Scenario: Tests pass in a clean temporary environment
- **GIVEN** a temporary directory with a minimal `fab/.kit/` structure (VERSION, model-tiers.yaml, scaffold/, skills/, templates/)
- **WHEN** `bats src/lib/sync-workspace/test.bats` is run
- **THEN** all tests pass
- **AND** each test operates in an isolated temporary directory

#### Scenario: Tests verify idempotency
- **GIVEN** `sync-workspace.sh` has been run once in a test environment
- **WHEN** it is run a second time
- **THEN** it produces no errors
- **AND** the resulting file system state is identical to the first run

## Test Suite: changeman

### Requirement: Directory Structure

A test directory `src/lib/changeman/` SHALL be created following the established pattern:

```
src/lib/changeman/
  changeman.sh → ../../../fab/.kit/scripts/lib/changeman.sh  (symlink)
  SPEC-changeman.md   (spec file)
  test.bats            (bats-core test suite)
```

#### Scenario: Symlink points to distributed script
- **GIVEN** the test directory has been set up
- **WHEN** resolving `src/lib/changeman/changeman.sh`
- **THEN** it points to `fab/.kit/scripts/lib/changeman.sh`
- **AND** the target file exists and is executable

### Requirement: SPEC File

`src/lib/changeman/SPEC-changeman.md` SHALL follow the established SPEC format: Sources of Truth, Usage, API/Behavior Reference, Requirements, Testing sections.

#### Scenario: SPEC file covers key behavioral areas
- **GIVEN** the SPEC file has been written
- **WHEN** reading the document
- **THEN** it documents: `new` subcommand interface, slug validation, change-id validation, folder creation, `.status.yaml` initialization, `detect_created_by` fallback chain, collision detection, and error handling

### Requirement: Test Coverage

The `test.bats` suite SHALL cover the following behavioral areas of `changeman.sh`:

1. **`new` subcommand — happy path** — creates folder with correct name format `{YYMMDD}-{XXXX}-{slug}`, initializes `.status.yaml`
2. **Slug validation** — rejects empty slug, leading/trailing hyphens, non-alphanumeric characters
3. **Change-id validation** — rejects non-4-char IDs, uppercase letters, special characters
4. **Random ID generation** — generates a 4-char alphanumeric ID when `--change-id` is not provided
5. **Collision detection** — provided ID colliding with existing folder is fatal; random IDs retry
6. **`--help` flag** — prints usage information
7. **Error cases** — missing slug, unknown flags, missing subcommand, unknown subcommand
8. **`detect_created_by` fallback chain** — `gh api user` → `git config user.name` → `"unknown"`
9. **Stageman integration** — `set-state` and `log-command` called with correct arguments

#### Scenario: Tests pass in a clean temporary environment
- **GIVEN** a temporary directory with the required structure (`fab/changes/`, `fab/.kit/templates/status.yaml`, `fab/.kit/scripts/lib/stageman.sh`)
- **WHEN** `bats src/lib/changeman/test.bats` is run
- **THEN** all tests pass
- **AND** each test operates in an isolated temporary directory

#### Scenario: Tests mock external dependencies
- **GIVEN** `changeman.sh` depends on `stageman.sh`, `gh`, and `git`
- **WHEN** tests run
- **THEN** `stageman.sh` invocations are either stubbed or use the real script against temp files
- **AND** `gh api user` and `git config` are overridden to avoid requiring real credentials

## Test Runner: Justfile Restructuring

### Requirement: Two-Tier Recipe Structure

The `justfile` SHALL be restructured with three recipes:

- **`just test-bash`** — runs both bats `.bats` files and legacy `test.sh` files
- **`just test-rust`** — placeholder (no-op or skip message) until Rust libraries exist
- **`just test`** — runs both `test-bash` and `test-rust`, displays a combined summary

#### Scenario: `just test-bash` runs both bats and legacy tests
- **GIVEN** bats test suites exist at `src/lib/sync-workspace/test.bats` and `src/lib/changeman/test.bats`
- **AND** legacy test suites exist at `src/lib/*/test.sh`
- **WHEN** `just test-bash` is run
- **THEN** all bats test suites are executed via `bats`
- **AND** all legacy test.sh suites are executed via `bash`
- **AND** results from both are reported

#### Scenario: `just test-rust` is a no-op
- **GIVEN** no Rust libraries exist yet
- **WHEN** `just test-rust` is run
- **THEN** it succeeds without error (no-op or informational message)

#### Scenario: `just test` runs the full pipeline
- **GIVEN** both bash and rust test recipes exist
- **WHEN** `just test` is run
- **THEN** it invokes `test-bash` and `test-rust`
- **AND** displays a combined summary after all suites complete

### Requirement: Summary Output

The test runner SHALL display a per-suite pass/fail summary and an overall verdict after all suites complete.

#### Scenario: All suites pass
- **GIVEN** all test suites pass
- **WHEN** the test runner completes
- **THEN** the summary shows each suite name with PASS status
- **AND** the overall verdict shows `N/N suites passed`

#### Scenario: Some suites fail
- **GIVEN** one or more test suites fail
- **WHEN** the test runner completes
- **THEN** the summary shows each suite with its PASS/FAIL status
- **AND** the overall verdict shows `M/N suites passed, K failed (suite-names)`
- **AND** the runner exits non-zero

#### Scenario: Suite names are derived from directory names
- **GIVEN** test suites exist at `src/lib/{name}/test.bats` and `src/lib/{name}/test.sh`
- **WHEN** the summary is displayed
- **THEN** each suite is identified by its directory name (e.g., `sync-workspace`, `changeman`, `stageman`)

## Design Decisions

1. **Bats on PATH, not vendored**: New bats tests require `bats` installed on the system PATH (e.g., via `brew install bats-core`).
   - *Why*: Consistent with existing dev tool precedent — `yq` and `gh` are also expected on PATH. Vendoring a test framework adds repo bulk and maintenance burden for marginal benefit. Aligns with Constitution I (Pure Prompt Play: single-binary utilities).
   - *Rejected*: Git submodule or vendored copy — adds complexity, requires submodule init, and bats-core is a standard tool available from all major package managers.
   <!-- assumed: bats expected on PATH rather than vendored — consistent with yq/gh precedent of single-binary dev tools on PATH, aligns with Constitution I (Pure Prompt Play) -->

2. **Unified `test-bash` recipe for both bats and legacy**: A single recipe handles both `.bats` and `test.sh` files rather than separate recipes per format.
   - *Why*: Users should not need to know which format a test suite uses. The distinction is temporary — DEV-1029 will migrate all to bats. Separate recipes add confusion for no user benefit.
   - *Rejected*: Separate `test-bats` and `test-legacy` recipes — fragments the test surface, adds unnecessary conceptual overhead.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename target is `sync-workspace.sh` | Explicitly discussed and agreed in conversation; confirmed from intake #1 | S:95 R:90 A:95 D:95 |
| 2 | Certain | Test directory follows `src/lib/*/` pattern (symlink + SPEC + test file) | All 4 existing scripts use this layout; confirmed from intake #2 | S:90 R:95 A:95 D:95 |
| 3 | Certain | SPEC files follow existing format (`SPEC-stageman.md` as reference) | Explicit in intake; confirmed from intake #3 | S:85 R:90 A:90 D:95 |
| 4 | Certain | New tests use bats-core (`.bats` files), existing tests unchanged | Explicitly discussed — bats for new, migrate existing in DEV-1029; confirmed from intake #4 | S:95 R:90 A:90 D:95 |
| 5 | Certain | Two-tier justfile: `test-bash` (bats + legacy), `test-rust` (placeholder), `test` (both + summary) | Explicitly discussed and agreed; confirmed from intake #5 | S:95 R:85 A:90 D:90 |
| 6 | Confident | Summary output is suite-level pass/fail with totals and overall verdict | User described the problem (no summary); specific format is reasonable inference; confirmed from intake #6 | S:70 R:90 A:80 D:70 |
| 7 | Confident | Bats expected on PATH rather than vendored | Consistent with `yq`/`gh` precedent of single-binary dev tools on PATH; Constitution I favors no package managers; new assumption at spec level | S:60 R:85 A:75 D:65 |
| 8 | Confident | Unified `test-bash` recipe for both `.bats` and `test.sh` formats | Natural grouping — users shouldn't need to know the format; temporary split until DEV-1029; new assumption at spec level | S:65 R:90 A:80 D:70 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
