# Spec: README Quick Start Restructure + fab-sync Prerequisites Check

**Change**: 260217-zkah-readme-quickstart-prereqs-check
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## README: Quick Start Restructure

### Requirement: Collapse Initialize into Install

The Quick Start section SHALL fold the current "2. Initialize" step into "1. Install" as a sub-section. The resulting Install section SHALL contain four sub-sections in this order:

1. `#### New project` (existing `gh release download` content)
2. `#### From a local clone` (existing `cp -r` content)
3. `#### Initialize` (content from the current step 2, moved here)
4. `#### Updating from a previous version` (content from the standalone `## Updating` section, moved here)

The standalone `## Updating` section SHALL be removed entirely — its content moves under Install.

#### Scenario: User reads restructured Quick Start

- **GIVEN** a user viewing the README Quick Start section
- **WHEN** they read the Install step
- **THEN** they see sub-sections for New project, From a local clone, Initialize, and Updating from a previous version
- **AND** there is no separate "2. Initialize" top-level step
- **AND** there is no standalone "## Updating" section elsewhere in the README

#### Scenario: Step numbering after restructure

- **GIVEN** the Quick Start section after restructuring
- **WHEN** a user reads the numbered steps
- **THEN** step 1 is "Install" (with sub-sections), step 2 is "Your first change", step 3 is "Going parallel"
- **AND** the Troubleshooting sub-section follows step 3

### Requirement: Update TOC

The Contents line at the top of the README SHALL be updated to reflect the new structure. The `#updating` anchor SHALL be removed from the top-level TOC entries since Updating is now a sub-section under Quick Start > Install.

#### Scenario: TOC reflects new structure

- **GIVEN** the README after restructuring
- **WHEN** a user clicks the Contents links
- **THEN** `Quick Start` links to `#quick-start`
- **AND** there is no top-level `Updating` entry in the Contents line
- **AND** all other Contents links remain functional

### Requirement: Install sub-sections use heading level 4

The sub-sections under Install (New project, From a local clone, Initialize, Updating from a previous version) SHALL use `####` (h4) headings. This maintains the existing heading hierarchy: `##` for top-level sections, `###` for Quick Start steps, `####` for sub-sections within steps.

#### Scenario: Heading hierarchy is consistent

- **GIVEN** the restructured README
- **WHEN** parsed by a markdown renderer
- **THEN** "1. Install" is an `h3` (`###`)
- **AND** "New project", "From a local clone", "Initialize", and "Updating from a previous version" are `h4` (`####`)

## Sync Pipeline: Prerequisites Check

### Requirement: New prerequisites sync step

A new sync step `fab/.kit/sync/1-prerequisites.sh` SHALL validate that all required tools are available before any other sync steps run. The script SHALL check for the following tools: `yq`, `jq`, `gh`, `direnv`, `bats`.

#### Scenario: All prerequisites present

- **GIVEN** a system with yq, jq, gh, direnv, and bats installed
- **WHEN** `fab-sync.sh` runs the sync pipeline
- **THEN** `1-prerequisites.sh` passes silently (or with a brief confirmation)
- **AND** subsequent sync steps (`2-direnv.sh`, `3-sync-workspace.sh`) execute normally

#### Scenario: One or more prerequisites missing

- **GIVEN** a system where `yq` is not installed
- **WHEN** `fab-sync.sh` runs the sync pipeline
- **THEN** `1-prerequisites.sh` exits with code 1
- **AND** the error message names the missing tool(s)
- **AND** the error message points users to the Prerequisites section in the README
- **AND** no subsequent sync steps execute

#### Scenario: Multiple tools missing

- **GIVEN** a system where both `jq` and `bats` are missing
- **WHEN** `1-prerequisites.sh` runs
- **THEN** the error output lists all missing tools (not just the first one found)
- **AND** the script exits with code 1

### Requirement: Existing sync steps renumber

The existing sync steps SHALL be renamed to maintain sort order after the new prerequisites step is inserted:

- `1-direnv.sh` → `2-direnv.sh`
- `2-sync-workspace.sh` → `3-sync-workspace.sh`

#### Scenario: Sync steps execute in correct order

- **GIVEN** a freshly synced `fab/.kit/sync/` directory
- **WHEN** `fab-sync.sh` runs sync steps in sorted order
- **THEN** `1-prerequisites.sh` runs first, then `2-direnv.sh`, then `3-sync-workspace.sh`

### Requirement: Prerequisites check is fatal

Each missing tool SHALL cause the sync pipeline to abort (exit 1). There SHALL be no "soft" or "warn-only" mode — if a prerequisite is listed, it is required.

#### Scenario: Fatal exit prevents downstream sync

- **GIVEN** a system missing `direnv`
- **WHEN** `fab-sync.sh` executes sync steps
- **THEN** `1-prerequisites.sh` exits 1
- **AND** `2-direnv.sh` and `3-sync-workspace.sh` do not execute

### Requirement: Prerequisites script uses `command -v`

The prerequisites check SHALL use `command -v <tool>` to test tool availability. This is POSIX-portable and checks PATH resolution without invoking the tool.

#### Scenario: Tool detection mechanism

- **GIVEN** the prerequisites script
- **WHEN** checking for `yq`
- **THEN** it uses `command -v yq` (not `which yq` or `yq --version`)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All five prerequisites are fatal (yq, jq, gh, direnv, bats) | Confirmed from intake #1 — keep it simple, partial environments cause confusing downstream failures | S:90 R:90 A:95 D:95 |
| 2 | Certain | GitHub auto-generates heading anchors from heading text | Confirmed from intake #2 — standard GitHub markdown behavior | S:95 R:95 A:95 D:95 |
| 3 | Certain | Sync steps run in sorted filename order | `fab-sync.sh` globs `sync/*.sh` in sorted order — this is how the pipeline works | S:95 R:95 A:95 D:95 |
| 4 | Confident | Prerequisites script reports all missing tools, not just the first | Confirmed from intake #3 with upgrade — better UX to show everything missing at once rather than one-at-a-time | S:80 R:90 A:85 D:80 |
| 5 | Confident | Use `command -v` for tool detection | POSIX-portable, no side effects, consistent with shell best practices. Existing sync scripts already use `command -v` (e.g., jq check in `2-sync-workspace.sh`) | S:75 R:95 A:90 D:85 |
| 6 | Certain | Sub-sections under Install use h4 headings | Follows existing README heading hierarchy (h2 → h3 → h4) | S:90 R:95 A:95 D:95 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
