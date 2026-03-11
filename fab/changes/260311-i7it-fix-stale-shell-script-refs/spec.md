# Spec: Fix Stale Shell-Script References After Go Binary Conversion

**Change**: 260311-i7it-fix-stale-shell-script-refs
**Created**: 2026-03-11
**Affected memory**: `docs/memory/fab-workflow/kit-scripts.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Rewriting any Go or Rust source code — purely documentation
- Updating `_scripts.md` — already the canonical CLI reference and correct
- Modifying skill files — no skill content changes needed
- Fixing any issues outside the 4 files identified in the intake

## Specs: Packages Rewrite

### Requirement: Accurate Package Architecture

`docs/specs/packages.md` SHALL be rewritten to accurately describe the current CLI tool architecture:

1. **wt** MUST be described as a Go binary at `fab/.kit/bin/wt`, not as shell scripts in `fab/.kit/packages/wt/`
2. **idea** MUST be described as a Go binary at `fab/.kit/bin/idea` (standalone binary added to PATH by `env-packages.sh`)
3. The `wt` binary SHALL use subcommand style (`wt create`, `wt list`, etc.), NOT hyphenated executables (`wt-create`, `wt-list`)
4. The spec SHALL document that `wt pr` was dropped (replaced by `/git-pr`)
5. References to `lib/wt-common.sh` SHALL be removed (deleted shared library)
6. The Package Architecture section SHALL reflect that `fab/.kit/packages/` no longer exists; both `wt` and `idea` are Go binaries in `fab/.kit/bin/`
7. PATH setup SHALL describe `env-packages.sh` adding `$KIT_DIR/bin` to PATH first (making `fab`, `wt`, `idea` binaries available), then iterating `packages/*/bin` for any future shell packages

#### Scenario: Agent consults packages.md for wt architecture
- **GIVEN** an agent reads `docs/specs/packages.md` for context on the wt tool
- **WHEN** the agent extracts the tool's location, invocation style, and available commands
- **THEN** it finds wt described as a Go binary at `fab/.kit/bin/wt` with subcommands (`wt create`, `wt list`, `wt open`, `wt delete`, `wt init`)
- **AND** no reference to shell scripts, `wt-create`, `wt-list`, `packages/wt/`, or `lib/wt-common.sh` is present

#### Scenario: Agent consults packages.md for idea architecture
- **GIVEN** an agent reads `docs/specs/packages.md` for context on the idea tool
- **WHEN** the agent extracts the tool's location and invocation paths
- **THEN** it finds idea described as a Go binary at `fab/.kit/bin/idea` (added to PATH by `env-packages.sh`)

#### Scenario: Agent consults packages.md for PATH setup
- **GIVEN** an agent reads the PATH setup section of `docs/specs/packages.md`
- **WHEN** it determines how `wt` and `idea` become available on PATH
- **THEN** it finds that `env-packages.sh` adds `$KIT_DIR/bin` to PATH (providing `fab`, `wt`, `idea` binaries)
- **AND** then iterates `$KIT_DIR/packages/*/bin` for any future shell packages

### Requirement: Accurate Packages Directory Structure

The spec SHALL reflect that `fab/.kit/packages/` no longer exists. All tools (`fab`, `wt`, `idea`) are in `fab/.kit/bin/`. The `env-packages.sh` script still iterates `packages/*/bin` for forward compatibility, but the directory is not present in the current repo.

#### Scenario: Agent checks what exists in packages/
- **GIVEN** an agent consults `docs/specs/packages.md` for the directory structure
- **WHEN** it reads the Package Architecture section
- **THEN** it sees all binaries listed under `fab/.kit/bin/` only
- **AND** no `fab/.kit/packages/` directory is referenced as currently existing

## Specs: Naming Convention Fixes

### Requirement: Correct Worktree Naming Source

In `docs/specs/naming.md`, the Worktree section SHALL reference `wt create` (subcommand of `fab/.kit/bin/wt`), not `wt-create` (`fab/.kit/packages/wt/bin/wt-create`).

#### Scenario: Agent reads worktree naming convention
- **GIVEN** an agent reads the Worktree section of `docs/specs/naming.md`
- **WHEN** it checks the "Encoded in" field
- **THEN** it finds `wt create` (`fab/.kit/bin/wt`) as the encoding location

### Requirement: Correct Backlog Entry Encoding Source

In `docs/specs/naming.md`, the Backlog Entry section SHALL reference `idea` command (`fab/.kit/bin/idea` — backlog management), not `idea` command (`fab/.kit/packages/wt/`).

#### Scenario: Agent reads backlog entry naming convention
- **GIVEN** an agent reads the Backlog Entry section of `docs/specs/naming.md`
- **WHEN** it checks the "Encoded in" field
- **THEN** it finds `idea` command (`fab/.kit/bin/idea` — backlog management) as the encoding location

## Memory: kit-scripts.md Deletion

### Requirement: Remove Obsolete Memory File

`docs/memory/fab-workflow/kit-scripts.md` SHALL be deleted entirely. The file documents 7 deleted shell scripts removed in change `260305-u8t9-clean-break-go-only`. The canonical CLI reference is `fab/.kit/skills/_scripts.md`.

#### Scenario: kit-scripts.md no longer exists
- **GIVEN** the change has been applied
- **WHEN** a file listing of `docs/memory/fab-workflow/` is performed
- **THEN** `kit-scripts.md` is not present

### Requirement: Update Memory Index

`docs/memory/fab-workflow/index.md` SHALL be updated to remove the `kit-scripts` entry from the file listing.

#### Scenario: Memory index reflects deletion
- **GIVEN** the change has been applied
- **WHEN** an agent reads `docs/memory/fab-workflow/index.md`
- **THEN** no `kit-scripts` entry appears in the file table

### Requirement: Cross-Reference in kit-architecture.md

`docs/memory/fab-workflow/kit-architecture.md` SHALL include a cross-reference note to `fab/.kit/skills/_scripts.md` as the canonical CLI command reference, placed in the Overview section.

#### Scenario: Agent finds canonical CLI reference from kit-architecture
- **GIVEN** an agent reads `docs/memory/fab-workflow/kit-architecture.md`
- **WHEN** it looks for CLI command documentation
- **THEN** it finds a cross-reference directing to `fab/.kit/skills/_scripts.md`

## Memory: kit-architecture.md Stale Section Removal

### Requirement: Remove Stale lib/ Script Sections

The following subsections in `docs/memory/fab-workflow/kit-architecture.md` under `### Shell Scripts` SHALL be removed:

1. `#### lib/statusman.sh`
2. `#### lib/logman.sh`
3. `#### lib/calc-score.sh`
4. `#### lib/changeman.sh`
5. `#### lib/archiveman.sh`

A brief replacement note SHALL be inserted explaining these operations are now handled by Go binary subcommands, with a cross-reference to `_scripts.md`.

#### Scenario: Agent reads kit-architecture.md Shell Scripts section
- **GIVEN** an agent reads the Shell Scripts section of `docs/memory/fab-workflow/kit-architecture.md`
- **WHEN** it looks for lib/ script documentation
- **THEN** it finds a note explaining the lib/ scripts were removed and replaced by Go subcommands
- **AND** no detailed sections for statusman.sh, logman.sh, calc-score.sh, changeman.sh, or archiveman.sh exist

### Requirement: Clarify env-packages.sh wt Description

The `lib/env-packages.sh` description SHALL explicitly note that `wt` is a Go binary in `$KIT_DIR/bin/` (not a shell package), and the `packages/*/bin` iteration picks up only remaining shell packages (currently: `idea`).

#### Scenario: Agent reads env-packages.sh description
- **GIVEN** an agent reads the `lib/env-packages.sh` section of kit-architecture.md
- **WHEN** it checks how `wt` becomes available on PATH
- **THEN** it learns that `wt` is a Go binary in `$KIT_DIR/bin/`
- **AND** the `packages/*/bin` iteration is described as providing any future shell packages (none currently exist)

## Deprecated Requirements

### Shell Script References in Packages Spec

**Reason**: wt and idea were converted from shell scripts to Go binaries. The shell script architecture description is obsolete.
**Migration**: Full rewrite of `docs/specs/packages.md` reflecting Go binary architecture.

### Detailed lib/ Shell Script Documentation in Memory

**Reason**: The 7 lib/ shell scripts were removed in `260305-u8t9-clean-break-go-only`. Memory file `kit-scripts.md` and detailed sections in `kit-architecture.md` are obsolete.
**Migration**: kit-scripts.md deleted; kit-architecture.md stale sections replaced with cross-reference note.

## Design Decisions

1. **Delete kit-scripts.md rather than rewrite**: `_scripts.md` is already the canonical CLI reference loaded by every skill via `_preamble.md`. A parallel memory file would create a consistency burden with no added value.
   - *Why*: Single source of truth (Constitution §II)
   - *Rejected*: Rewriting kit-scripts.md as Go binary reference — duplicates `_scripts.md`

2. **Brief replacement note rather than full kit-architecture.md rewrite**: The Go Binary section (lines ~332-377) already thoroughly documents the binary. Only stale lib/ sections need removal.
   - *Why*: Minimizes change scope; rest of file is accurate
   - *Rejected*: Full rewrite of kit-architecture.md

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | wt is a Go binary at `fab/.kit/bin/wt` | Confirmed from intake #1 — verified by consistency check | S:95 R:90 A:95 D:95 |
| 2 | Certain | idea is a Go binary at `fab/.kit/bin/idea` | Confirmed from intake #2 — verified by consistency check | S:95 R:90 A:95 D:95 |
| 3 | Certain | 7 lib/ shell scripts were deleted in 260305-u8t9 | Confirmed from intake #3 — changelog confirms | S:95 R:90 A:95 D:95 |
| 4 | Certain | `wt pr` subcommand was dropped | Confirmed from intake #4 — replaced by `/git-pr` | S:90 R:85 A:90 D:95 |
| 5 | Certain | idea is a standalone Go binary; no shell package exists | Confirmed — `fab/.kit/packages/` directory does not exist in repo | S:90 R:85 A:90 D:90 |
| 6 | Certain | wt uses subcommands not hyphenated executables | Confirmed from intake #6 — Go binary verified | S:95 R:90 A:95 D:95 |
| 7 | Confident | packages.md needs full rewrite, not incremental fixes | Confirmed from intake #7 — structural mismatch too deep for line edits | S:80 R:75 A:80 D:75 |
| 8 | Certain | kit-scripts.md should be deleted, not rewritten | Confirmed from intake #8 — user confirmed _scripts.md is canonical | S:95 R:80 A:95 D:95 |
| 9 | Certain | Stale lib/ sections span lines ~126-166 in kit-architecture.md | Verified by reading file — confirmed statusman, logman, calc-score, changeman, archiveman sections | S:90 R:85 A:95 D:95 |
| 10 | Certain | env-packages.sh description needs wt binary clarification | Verified by reading file — current text correct but doesn't distinguish wt binary from packages | S:85 R:85 A:90 D:90 |

10 assumptions (9 certain, 1 confident, 0 tentative, 0 unresolved).
