# Spec: Status Symlink Pointer

**Change**: 260307-x2tx-status-symlink-pointer
**Created**: 2026-03-07
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/schemas.md`, `docs/memory/fab-workflow/migrations.md`

## Non-Goals

- Windows compatibility — explicitly scoped out per user decision
- Changing `.fab-runtime.yaml` format or location — remains unchanged
- Modifying how `fab resolve` override mode works — only default-mode (no override) changes
- Atomic symlink updates — accepted tradeoff per user decision (all changes go through `fab` subcommands)

## Go Binary: Symlink Pointer

### Requirement: Symlink Replaces fab/current

The active change pointer SHALL be a symlink at `<repo_root>/.fab-status.yaml` instead of the text file `fab/current`. The symlink target SHALL be a relative path from the repo root to the active change's `.status.yaml`: `fab/changes/{name}/.status.yaml`.

#### Scenario: Reading active change via symlink
- **GIVEN** `.fab-status.yaml` is a symlink pointing to `fab/changes/260307-x2tx-example/.status.yaml`
- **WHEN** `resolveFromCurrent()` is called with no override
- **THEN** it SHALL call `os.Readlink()` on the symlink path
- **AND** extract the folder name (`260307-x2tx-example`) from the target path by parsing the path components between `fab/changes/` and `/.status.yaml`
- **AND** return the folder name for downstream resolution

#### Scenario: No symlink present (no active change)
- **GIVEN** `.fab-status.yaml` does not exist (neither as file nor symlink)
- **WHEN** `resolveFromCurrent()` is called
- **THEN** it SHALL fall through to the single-change guess logic (unchanged behavior)

#### Scenario: Broken symlink (archived/deleted change)
- **GIVEN** `.fab-status.yaml` is a symlink but the target file does not exist
- **WHEN** `resolveFromCurrent()` is called
- **THEN** `os.Readlink()` SHALL succeed (reads the link, not the target)
- **AND** the folder name SHALL be extracted from the target path
- **AND** if the change folder does not exist, resolution SHALL fail with the standard "change not found" error

#### Scenario: Symlink path construction
- **GIVEN** `fabRoot` is the path to the `fab/` directory
- **WHEN** constructing the symlink path
- **THEN** the symlink path SHALL be `filepath.Join(filepath.Dir(fabRoot), ".fab-status.yaml")` (repo root, not `fab/`)

### Requirement: Switch Creates Symlink

The `Switch()` function in `internal/change/change.go` SHALL create a symlink instead of writing a text file.

#### Scenario: Switching to a change
- **GIVEN** a valid change folder `260307-x2tx-example` exists
- **WHEN** `Switch()` is called with that change name
- **THEN** it SHALL remove any existing `.fab-status.yaml` (file or symlink) via `os.Remove()`
- **AND** create a new symlink: `os.Symlink("fab/changes/260307-x2tx-example/.status.yaml", symlinkPath)`
- **AND** return the structured switch output (name, stage, confidence, next command)

#### Scenario: Switch --blank (deactivate)
- **GIVEN** `.fab-status.yaml` exists as a symlink
- **WHEN** `SwitchBlank()` is called
- **THEN** it SHALL remove the symlink via `os.Remove()`
- **AND** return deactivation confirmation
- **AND** if the symlink does not exist, return "(already blank)" without error

#### Scenario: Switch overwrites stale file
- **GIVEN** `.fab-status.yaml` exists as a regular file (legacy or corruption)
- **WHEN** `Switch()` is called
- **THEN** `os.Remove()` SHALL remove the regular file
- **AND** create the symlink normally

### Requirement: Rename Updates Symlink

The `Rename()` function SHALL update the symlink target when the active change is renamed.

#### Scenario: Renaming the active change
- **GIVEN** `.fab-status.yaml` points to `fab/changes/260307-x2tx-old-slug/.status.yaml`
- **AND** the active change is being renamed to `260307-x2tx-new-slug`
- **WHEN** `Rename()` updates the `.fab-status.yaml` symlink
- **THEN** it SHALL read the current symlink target via `os.Readlink()`
- **AND** extract the folder name from the target
- **AND** if the folder name matches the old name, remove and recreate the symlink with the new target path
- **AND** if the symlink does not exist or points to a different change, skip symlink update

### Requirement: Pane Map Uses Symlink

The `readFabCurrent()` function in `panemap.go` SHALL resolve the symlink instead of reading a text file.

#### Scenario: Reading active change for pane display
- **GIVEN** `.fab-status.yaml` is a symlink in the worktree root
- **WHEN** `readFabCurrent()` is called with the worktree root path
- **THEN** it SHALL call `os.Readlink()` on `filepath.Join(wtRoot, ".fab-status.yaml")`
- **AND** extract the folder name from the target path
- **AND** return `(folderName, folderName)` as `(displayName, folderName)`

#### Scenario: No symlink in worktree
- **GIVEN** `.fab-status.yaml` does not exist in the worktree
- **WHEN** `readFabCurrent()` is called
- **THEN** it SHALL return `("(no change)", "")`

## Go Binary: ID Field in .status.yaml

### Requirement: Status File Includes Change ID

The `.status.yaml` file SHALL include an `id` field as the first top-level field, containing the 4-char change ID.

#### Scenario: New change creation
- **GIVEN** a new change is being created via `fab change new`
- **WHEN** the `.status.yaml` is initialized from the template
- **THEN** it SHALL include `id: {XXXX}` as the first field (before `name:`)
- **AND** the ID SHALL be the same 4-char token used in the folder name

#### Scenario: Reading the ID from status file
- **GIVEN** a `.status.yaml` with `id: x2tx`
- **WHEN** the status file is parsed
- **THEN** the `StatusFile` struct SHALL expose the `id` field
- **AND** the ID SHALL be available without parsing the folder name

### Requirement: Template Updated

The `fab/.kit/templates/status.yaml` SHALL include an `id` placeholder as the first field.

#### Scenario: Template structure
- **GIVEN** the status template file
- **WHEN** it is used to initialize a new change
- **THEN** the template SHALL contain `id: {ID}` as the first line (before `name: {NAME}`)

### Requirement: StatusFile Struct Updated

The `StatusFile` struct in `statusfile.go` SHALL include an `ID` field.

#### Scenario: Parsing status file with ID
- **GIVEN** a `.status.yaml` containing `id: x2tx`
- **WHEN** `statusfile.Load()` parses the file
- **THEN** `sf.ID` SHALL equal `"x2tx"`

#### Scenario: Writing status file preserves ID
- **GIVEN** a loaded `StatusFile` with `ID: "x2tx"`
- **WHEN** `sf.Save()` writes the file
- **THEN** `id: x2tx` SHALL appear as the first field in the output

## Configuration: .gitignore

### Requirement: Gitignore Updated

The `.gitignore` SHALL reference `.fab-status.yaml` instead of `fab/current`.

#### Scenario: Gitignore content
- **GIVEN** the repo `.gitignore`
- **WHEN** the change is applied
- **THEN** the line `fab/current` SHALL be replaced with `.fab-status.yaml`

## Configuration: Template

### Requirement: Status Template ID Placeholder

The template at `fab/.kit/templates/status.yaml` SHALL include `{ID}` placeholder.

#### Scenario: Placeholder replacement in fab change new
- **GIVEN** the template contains `id: {ID}`
- **WHEN** `New()` in `change.go` initializes a status file
- **THEN** `{ID}` SHALL be replaced with the generated 4-char change ID

## Skills and Documentation

### Requirement: Preamble References Updated

All references to `fab/current` in `_preamble.md` SHALL be updated to reference `.fab-status.yaml` symlink.

#### Scenario: Preamble change context section
- **GIVEN** `_preamble.md` Section 2 (Change Context)
- **WHEN** describing the active change pointer
- **THEN** it SHALL reference `.fab-status.yaml` symlink at repo root instead of `fab/current` text file

### Requirement: Scripts Reference Updated

All references to `fab/current` in `_scripts.md` SHALL be updated.

#### Scenario: Change command documentation
- **GIVEN** `_scripts.md` documents `fab change switch`
- **WHEN** describing what switch writes
- **THEN** it SHALL reference creating/removing `.fab-status.yaml` symlink

### Requirement: Skill Files Updated

Skills that reference `fab/current` SHALL be updated to reference `.fab-status.yaml`.

#### Scenario: fab-switch skill
- **GIVEN** `fab-switch.md` references `fab/current`
- **WHEN** describing switch behavior
- **THEN** it SHALL reference `.fab-status.yaml` symlink

#### Scenario: fab-archive skill
- **GIVEN** `fab-archive.md` references clearing `fab/current`
- **WHEN** describing pointer clearing
- **THEN** it SHALL reference symlink removal via `fab change switch --blank`

### Requirement: Spec and Memory References Updated

All references to `fab/current` in specs and memory files SHALL be updated.

#### Scenario: Glossary update
- **GIVEN** `docs/specs/glossary.md` defines "Pointer file" as `fab/current`
- **WHEN** the change is applied
- **THEN** the definition SHALL reference `.fab-status.yaml` symlink

## Migration

### Requirement: Migration File

A migration file SHALL be created at `fab/.kit/migrations/` to convert existing `fab/current` files to `.fab-status.yaml` symlinks and backfill `id` fields.

#### Scenario: Migrate existing fab/current
- **GIVEN** `fab/current` exists as a two-line text file with ID on line 1 and folder name on line 2
- **WHEN** the migration is applied
- **THEN** the folder name SHALL be read from line 2
- **AND** a symlink SHALL be created: `.fab-status.yaml` → `fab/changes/{folder}/.status.yaml`
- **AND** `fab/current` SHALL be removed
- **AND** `.gitignore` SHALL be updated (remove `fab/current`, add `.fab-status.yaml`)

#### Scenario: Migrate with empty/missing fab/current
- **GIVEN** `fab/current` does not exist or is empty
- **WHEN** the migration is applied
- **THEN** the symlink step SHALL be skipped (no-op)
- **AND** `.gitignore` SHALL still be updated

#### Scenario: Backfill id field in existing .status.yaml files
- **GIVEN** existing `.status.yaml` files without an `id` field
- **WHEN** the migration is applied
- **THEN** the `id` field SHALL be extracted from the folder name (second hyphen-separated component)
- **AND** `id: {extracted_id}` SHALL be added as the first line of each `.status.yaml`

## Go Tests

### Requirement: Test Coverage for Symlink Resolution

Tests SHALL verify symlink-based resolution replaces file-based resolution.

#### Scenario: resolve tests
- **GIVEN** the resolve test suite
- **WHEN** tests set up the active change
- **THEN** they SHALL create a `.fab-status.yaml` symlink (not write `fab/current`)
- **AND** verify `resolveFromCurrent()` correctly extracts the folder name from the symlink target

#### Scenario: change switch tests
- **GIVEN** the change switch test suite
- **WHEN** testing switch behavior
- **THEN** tests SHALL verify `.fab-status.yaml` is a symlink after switch
- **AND** the symlink target SHALL be the correct relative path
- **AND** tests SHALL verify `fab/current` is NOT created

#### Scenario: panemap tests
- **GIVEN** the panemap test suite
- **WHEN** testing `readFabCurrent()`
- **THEN** tests SHALL use symlinks instead of text files
- **AND** verify correct folder name extraction

#### Scenario: New test cases
- **GIVEN** the test suites
- **WHEN** the change is complete
- **THEN** there SHALL be tests for:
  - Symlink resolution returns correct folder name
  - Broken symlink treated as missing (falls through to guess)
  - Switch creates symlink with correct relative target
  - Switch --blank removes symlink
  - Rename updates symlink target
  - ID field round-trips through parse/save

## Deprecated Requirements

### fab/current Text File
**Reason**: Replaced by `.fab-status.yaml` symlink. The two-line text file format (ID on line 1, folder name on line 2) is superseded by a symlink whose target path encodes both the change identity and the status file location.
**Migration**: `fab/.kit/migrations/{VERSION}.md` converts existing `fab/current` to symlink.

## Design Decisions

1. **Symlink target is relative path, not absolute**
   - *Why*: Relative paths work when the repo is moved or across worktrees. Absolute paths would break on any path change.
   - *Rejected*: Absolute symlink — breaks on repo move, different on each machine.

2. **Broken symlink = fall through to guess, not hard error**
   - *Why*: A broken symlink means the target change folder was deleted or archived. Falling through to single-change guess preserves the existing graceful degradation. The folder name can still be extracted from the symlink target for error messages.
   - *Rejected*: Hard error on broken symlink — would require manual cleanup after every archive.

3. **os.Remove + os.Symlink (non-atomic)**
   - *Why*: All symlink lifecycle goes through `fab` subcommands (switch, rename, archive). No concurrent writers. Atomic rename-over-symlink requires a temp file dance that adds complexity for no practical benefit.
   - *Rejected*: Atomic symlink swap via temp file — over-engineering for single-writer scenario.

4. **ID field in .status.yaml rather than parsing folder name**
   - *Why*: Makes the change ID directly available from a single file read. The conductor can `readlink` + `cat` to get both identity and full status without folder name parsing.
   - *Rejected*: Parse-only approach — requires string manipulation on every read, fragile if naming conventions change.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `.fab-status.yaml` symlink at repo root replaces `fab/current` | Confirmed from intake #1 — user explicitly chose this | S:95 R:75 A:90 D:95 |
| 2 | Certain | Symlink target is relative path: `fab/changes/{name}/.status.yaml` | Confirmed from intake #2 — relative paths work across worktrees | S:90 R:85 A:90 D:90 |
| 3 | Certain | Naming: `.fab-status.yaml` (not `.status.yaml`) | Confirmed from intake #3 — consistency with `.fab-runtime.yaml` | S:95 R:90 A:95 D:95 |
| 4 | Certain | Location: repo root (not `fab/`) | Confirmed from intake #4 — groups ephemeral state | S:90 R:85 A:85 D:90 |
| 5 | Certain | Broken symlink = no active change (fall through to guess) | Confirmed from intake #5 — same semantics as missing fab/current | S:90 R:90 A:90 D:95 |
| 6 | Certain | Add `id` field to `.status.yaml` | Confirmed from intake #6 — direct availability without path parsing | S:90 R:90 A:85 D:90 |
| 7 | Certain | Windows compatibility not a concern | Confirmed from intake #7 — user stated explicitly | S:95 R:95 A:95 D:95 |
| 8 | Certain | All symlink lifecycle through `fab` subcommands | Confirmed from intake #8 — no direct symlink manipulation from skills | S:90 R:80 A:90 D:90 |
| 9 | Confident | Migration ships in `fab/.kit/migrations/` per project convention | Confirmed from intake #9 — consistent with context.md policy | S:80 R:80 A:85 D:85 |
| 10 | Confident | Non-atomic symlink update via os.Remove + os.Symlink | Confirmed from intake #10 — single-writer through fab subcommands | S:80 R:85 A:80 D:85 |
| 11 | Certain | Existing StatusFile struct gains `ID string` field with yaml tag `id` | Codebase confirms no id field exists yet; struct uses standard yaml tags | S:85 R:90 A:90 D:95 |
| 12 | Confident | `resolveFromCurrent` extracts folder name by splitting symlink target path | Target format is fixed (`fab/changes/{name}/.status.yaml`), path splitting is reliable | S:80 R:85 A:85 D:80 |

12 assumptions (9 certain, 3 confident, 0 tentative, 0 unresolved).
