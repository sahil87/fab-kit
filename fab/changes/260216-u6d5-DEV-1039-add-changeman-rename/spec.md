# Spec: Add Rename Subcommand to changeman.sh

**Change**: 260216-u6d5-DEV-1039-add-changeman-rename
**Created**: 2026-02-16
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Git branch rename — complex (local + remote), out of scope for this change; can be added as a future enhancement
- Renaming archived changes — rename operates only on active (non-archived) changes under `fab/changes/`

## Change Lifecycle: Rename Subcommand

### Requirement: Rename Interface

`changeman.sh` SHALL support a `rename` subcommand with the interface: `changeman.sh rename --folder <current-folder> --slug <new-slug>`.

Both `--folder` and `--slug` are required flags. `--folder` accepts the full current change folder name. `--slug` accepts the new slug portion that replaces everything after the `{YYMMDD}-{XXXX}-` prefix.

#### Scenario: Basic rename

- **GIVEN** a change folder `260216-u6d5-DEV-1039-add-changeman-rename` exists under `fab/changes/`
- **WHEN** `changeman.sh rename --folder 260216-u6d5-DEV-1039-add-changeman-rename --slug DEV-1039-changeman-rename-cmd` is executed
- **THEN** the folder is renamed to `260216-u6d5-DEV-1039-changeman-rename-cmd`
- **AND** the new folder name is printed to stdout

#### Scenario: Missing --folder flag

- **GIVEN** the rename subcommand is invoked
- **WHEN** `--folder` is not provided
- **THEN** the command exits non-zero with an error message to stderr indicating `--folder` is required

#### Scenario: Missing --slug flag

- **GIVEN** the rename subcommand is invoked
- **WHEN** `--slug` is not provided
- **THEN** the command exits non-zero with an error message to stderr indicating `--slug` is required

### Requirement: Prefix Preservation

The rename operation SHALL preserve the `{YYMMDD}-{XXXX}` prefix (first two hyphen-separated segments) from the current folder name. Only the slug portion (everything after the prefix) is replaced.

#### Scenario: Prefix extracted correctly

- **GIVEN** a change folder `260216-u6d5-DEV-1039-add-changeman-rename` exists
- **WHEN** `changeman.sh rename --folder 260216-u6d5-DEV-1039-add-changeman-rename --slug new-slug` is executed
- **THEN** the new folder name is `260216-u6d5-new-slug`
- **AND** the date prefix `260216` and ID `u6d5` are preserved

### Requirement: Slug Validation

The `--slug` argument SHALL be validated using the same regex as the `new` subcommand: `^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`. This permits alphanumeric characters and hyphens, with no leading or trailing hyphens. Uppercase is allowed (for Linear issue IDs like `DEV-1039-new-name`).

#### Scenario: Valid slug accepted

- **GIVEN** a valid source folder exists
- **WHEN** `--slug add-new-feature` is provided
- **THEN** the rename succeeds

#### Scenario: Slug with leading hyphen rejected

- **GIVEN** a valid source folder exists
- **WHEN** `--slug "-bad-slug"` is provided
- **THEN** the command exits non-zero with `Invalid slug format` error to stderr

#### Scenario: Slug with trailing hyphen rejected

- **GIVEN** a valid source folder exists
- **WHEN** `--slug "bad-slug-"` is provided
- **THEN** the command exits non-zero with `Invalid slug format` error to stderr

#### Scenario: Slug with uppercase (Linear ID) accepted

- **GIVEN** a valid source folder exists
- **WHEN** `--slug DEV-1039-new-name` is provided
- **THEN** the rename succeeds with the new slug included as-is

### Requirement: Source Folder Validation

The rename command SHALL verify the source folder exists under `fab/changes/` before proceeding.

#### Scenario: Source folder does not exist

- **GIVEN** no folder named `260216-xxxx-nonexistent` exists under `fab/changes/`
- **WHEN** `changeman.sh rename --folder 260216-xxxx-nonexistent --slug new-slug` is executed
- **THEN** the command exits non-zero with `ERROR: Change folder '260216-xxxx-nonexistent' not found` to stderr

### Requirement: Destination Collision Detection

The rename command SHALL verify the destination folder name does not already exist under `fab/changes/` before renaming.

#### Scenario: Destination folder already exists

- **GIVEN** folders `260216-u6d5-old-name` and `260216-u6d5-new-name` both exist under `fab/changes/`
- **WHEN** `changeman.sh rename --folder 260216-u6d5-old-name --slug new-name` is executed
- **THEN** the command exits non-zero with `ERROR: Folder '260216-u6d5-new-name' already exists` to stderr

### Requirement: Same-Name Detection

The rename command SHALL detect when the new slug produces the same folder name as the current one and exit with an error.

#### Scenario: New name equals current name

- **GIVEN** a folder `260216-u6d5-same-slug` exists
- **WHEN** `changeman.sh rename --folder 260216-u6d5-same-slug --slug same-slug` is executed
- **THEN** the command exits non-zero with `ERROR: New name is the same as current name` to stderr

### Requirement: Status File Update

After renaming the folder, the `name` field in `.status.yaml` SHALL be updated to reflect the new folder name.

#### Scenario: .status.yaml name field updated

- **GIVEN** a change folder is successfully renamed from `260216-u6d5-old-slug` to `260216-u6d5-new-slug`
- **WHEN** the rename completes
- **THEN** `.status.yaml` inside the renamed folder contains `name: 260216-u6d5-new-slug`

### Requirement: Active Change Pointer Update

If `fab/current` contains the old folder name, the rename command SHALL update it to the new folder name. If `fab/current` points to a different change (or does not exist), it SHALL NOT be modified.

#### Scenario: fab/current points to renamed change

- **GIVEN** `fab/current` contains `260216-u6d5-old-slug`
- **WHEN** `changeman.sh rename --folder 260216-u6d5-old-slug --slug new-slug` succeeds
- **THEN** `fab/current` now contains `260216-u6d5-new-slug`

#### Scenario: fab/current points to different change

- **GIVEN** `fab/current` contains `260216-abcd-other-change`
- **WHEN** `changeman.sh rename --folder 260216-u6d5-old-slug --slug new-slug` succeeds
- **THEN** `fab/current` still contains `260216-abcd-other-change`

#### Scenario: fab/current does not exist

- **GIVEN** `fab/current` does not exist
- **WHEN** the rename succeeds
- **THEN** `fab/current` is not created

### Requirement: Stageman Logging

The rename command SHALL log the operation via `stageman log-command` with the change directory and rename arguments.

#### Scenario: Rename logged to history

- **GIVEN** a rename operation succeeds
- **WHEN** the rename completes
- **THEN** `stageman log-command` is called with the new change directory path and `"changeman-rename"` as the command name

### Requirement: Help Text

The `show_help()` function SHALL document the `rename` subcommand with its flags, behavior, and examples.

#### Scenario: Help includes rename

- **GIVEN** the user runs `changeman.sh --help`
- **WHEN** the help text is displayed
- **THEN** the output includes the `rename` subcommand with `--folder` and `--slug` flags

### Requirement: CLI Dispatch

The top-level `case` dispatch SHALL route the `rename` subcommand to `cmd_rename`.

#### Scenario: rename subcommand dispatched

- **GIVEN** the user runs `changeman.sh rename --folder ... --slug ...`
- **WHEN** the CLI dispatcher processes the input
- **THEN** `cmd_rename` is invoked with the remaining arguments

## Change Lifecycle: Spec and Test Updates

### Requirement: SPEC-changeman.md API Reference

`src/lib/changeman/SPEC-changeman.md` SHALL be updated to include the `rename` subcommand in the API Reference table and with a dedicated `### rename Subcommand` section documenting arguments, behavior, collision detection, and error cases.

#### Scenario: Spec includes rename

- **GIVEN** the SPEC file is updated
- **WHEN** a reader views the API Reference
- **THEN** the `rename` subcommand appears in the Subcommands table alongside `new`
- **AND** a `rename` subsection documents the full interface

### Requirement: Test Coverage

`src/lib/changeman/test.bats` SHALL include a `# -- rename: ... --` section covering all scenarios: happy path, `.status.yaml` update, `fab/current` update (active and non-active), slug validation, missing source folder, destination collision, same-name detection, missing flags, and stageman logging.

#### Scenario: All rename test cases present

- **GIVEN** the test file includes a rename section
- **WHEN** `bats src/lib/changeman/test.bats` is run
- **THEN** all rename tests pass

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Slug validation uses same regex as `new` | Confirmed from intake #1 — reuse existing `^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$` | S:90 R:95 A:95 D:95 |
| 2 | Certain | Rename preserves `{YYMMDD}-{XXXX}` prefix, replaces only slug | Confirmed from intake #2 — date and 4-char ID are immutable identifiers per naming convention | S:85 R:90 A:95 D:90 |
| 3 | Certain | Update `.status.yaml` `name` field on rename | Confirmed from intake #3 — name must match folder name, structural invariant | S:90 R:95 A:95 D:95 |
| 4 | Certain | Update `fab/current` if it references the renamed change | Upgraded from intake #4 (Confident → Certain) — preflight resolution depends on exact name match; not updating breaks the active change pointer | S:85 R:90 A:95 D:90 |
| 5 | Confident | No git branch rename in this scope | Confirmed from intake #5 — git branch rename is complex (local + remote), listed as non-goal | S:60 R:80 A:70 D:65 |
| 6 | Certain | Use `--folder` and `--slug` flags (not positional args) | Upgraded from intake #6 (Confident → Certain) — consistent with `new`'s flag-based style, verified against `cmd_new` implementation | S:80 R:90 A:90 D:85 |
| 7 | Certain | Use `sed` for `.status.yaml` name update (not yq) | Replacing the `name:` field is a simple string substitution; no structural YAML manipulation needed; matches changeman.sh's existing yq-free pattern | S:80 R:90 A:90 D:90 |
| 8 | Confident | Log via `stageman log-command` with `"changeman-rename"` as command name | Follows the `new` subcommand's logging pattern; command name identifies the operation for history | S:70 R:85 A:80 D:75 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
