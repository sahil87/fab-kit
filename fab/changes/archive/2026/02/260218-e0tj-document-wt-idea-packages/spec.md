# Spec: Document wt and idea packages

**Change**: 260218-e0tj-document-wt-idea-packages
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Per-command reference documentation — inline `help` subcommands already cover this
- Tutorial-style onboarding content — the README handles that
- Dynamic package discovery in fab-help.sh — packages are stable and few

## fab-help.sh: Packages Footer Section

### Requirement: Packages Section in Help Output

`fab-help.sh` SHALL append a `PACKAGES` section after the `TYPICAL FLOW` section that lists bundled packages with one-liner descriptions. The section SHALL be a static block (not dynamically scanned). Each package line SHALL display the command name(s) and a brief description. The section SHALL end with a hint directing users to `<command> help` for details.

#### Scenario: User runs fab-help.sh

- **GIVEN** a user has fab-kit installed with packages in `fab/.kit/packages/`
- **WHEN** the user runs `fab/.kit/scripts/fab-help.sh`
- **THEN** the output includes a `PACKAGES` section after `TYPICAL FLOW`
- **AND** the section lists `wt-create, wt-list, wt-open, wt-delete, wt-init, wt-pr` with description "Git worktree management"
- **AND** the section lists `idea` with description "Per-repo backlog (fab/backlog.md)"
- **AND** the section ends with `Run <command> help for details.`

#### Scenario: Packages section formatting

- **GIVEN** the help output has rendered the `TYPICAL FLOW` section
- **WHEN** the `PACKAGES` section renders
- **THEN** the section header uses the same formatting style as other sections (uppercase, no indent)
- **AND** package entries use the same indentation as skill entries (4-space indent)
- **AND** descriptions are left-aligned with each other

### Requirement: wt-pr Included in Package Listing

The wt package listing SHALL include `wt-pr` alongside the original 5 wt commands. `wt-pr` was added in the 260218-qcqx-harden-wt-resilience change and is a shipped bin/ command.

#### Scenario: Complete wt command listing

- **GIVEN** the wt package contains 6 commands: wt-create, wt-list, wt-open, wt-delete, wt-init, wt-pr
- **WHEN** fab-help.sh renders the PACKAGES section
- **THEN** all 6 commands appear in the wt line

## docs/specs/packages.md: New Spec Page

### Requirement: Packages Spec Page Structure

`docs/specs/packages.md` SHALL be created as a single page covering both the wt and idea packages at the concept/workflow level. The page SHALL contain the following sections in order:

1. **Overview** — what packages are, how they relate to the fab pipeline
2. **wt (Worktree Management)** — concept, commands, integration with fab, common workflows
3. **idea (Backlog Management)** — concept, commands, integration with fab, common workflows
4. **Package Architecture** — directory layout, bin/lib convention, distribution via kit.tar.gz

#### Scenario: New user reads packages.md

- **GIVEN** a developer has access to the docs/specs/ directory
- **WHEN** they read `docs/specs/packages.md`
- **THEN** they understand what packages are and how they relate to the fab pipeline
- **AND** they understand the wt commands and when to use them
- **AND** they understand the idea command and how it feeds `/fab-new`
- **AND** they understand where packages live in the directory structure

### Requirement: wt Section Content

The wt section SHALL explain the worktree management concept, list all 6 commands with one-liner descriptions, describe integration with the fab pipeline (assembly-line pattern, batch scripts), and include common workflows.
<!-- assumed: wt section references assembly-line.md by name — the existing spec is the authoritative source for the assembly-line pattern -->

#### Scenario: wt section covers assembly-line integration

- **GIVEN** the wt section is being read by a user familiar with fab
- **WHEN** they reach the integration subsection
- **THEN** they understand how `wt-create` enables the assembly-line pattern (one worktree per change)
- **AND** they see references to `batch-fab-new-backlog`, `batch-fab-switch-change`, and `batch-fab-archive-change`
- **AND** they are directed to `docs/specs/assembly-line.md` for the full pattern description

#### Scenario: wt section lists all commands

- **GIVEN** the wt package ships 6 bin/ commands
- **WHEN** the commands subsection renders
- **THEN** each command (wt-create, wt-list, wt-open, wt-delete, wt-init, wt-pr) appears with a one-liner description
- **AND** the descriptions match what `<command> help` shows (not duplicating full help text)

### Requirement: idea Section Content

The idea section SHALL explain the per-repo backlog concept, list the CRUD subcommands with one-liner descriptions, describe integration with fab (`idea` writes to `fab/backlog.md`, `/fab-new` reads backlog IDs), and include common workflows.

#### Scenario: idea section covers fab integration

- **GIVEN** the idea section is being read by a user
- **WHEN** they reach the integration subsection
- **THEN** they understand that `idea` manages `fab/backlog.md`
- **AND** they understand that `/fab-new` can accept a backlog ID to start a change from an idea
- **AND** they understand the capture-triage-implement lifecycle

### Requirement: Package Architecture Section

The architecture section SHALL document where packages live (`fab/.kit/packages/{name}/`), the bin/ and lib/ convention, and how packages are distributed via `kit.tar.gz`. This section SHOULD be concise — it supplements the kit-architecture memory file, not replaces it.

#### Scenario: Contributor reads architecture section

- **GIVEN** a contributor wants to understand how packages are structured
- **WHEN** they read the architecture section
- **THEN** they see the `fab/.kit/packages/{name}/` directory structure
- **AND** they understand the bin/ (executable commands) and lib/ (shared libraries) convention
- **AND** they understand that packages ship inside `kit.tar.gz` and are added to PATH via `env-packages.sh`

## docs/specs/index.md: New Entry

### Requirement: Packages Entry in Specs Index

`docs/specs/index.md` SHALL include a new row for `packages.md` in the existing table. The description SHALL be concise and follow the existing style.

#### Scenario: User browses specs index

- **GIVEN** the specs index table exists with entries for overview, architecture, skills, etc.
- **WHEN** the user reads `docs/specs/index.md`
- **THEN** they see a `[packages](packages.md)` entry in the table
- **AND** the description reads something like "Bundled packages — wt (worktree management) and idea (backlog management)"

## Design Decisions

1. **Static block over dynamic scanning for packages in fab-help.sh**
   - *Why*: Packages are stable (2 packages, unlikely to change frequently). Dynamic scanning adds complexity (directory iteration, description extraction) for negligible benefit. The skill catalog is dynamic because skills change often; packages don't.
   - *Rejected*: Scanning `fab/.kit/packages/*/` at runtime — over-engineering for 2 stable entries.

2. **Single packages.md over per-package spec pages**
   - *Why*: Two packages don't justify separate files. A single page keeps related concepts together and avoids navigation overhead. Inline `help` subcommands remain the per-command reference.
   - *Rejected*: `wt.md` + `idea.md` — fragmentation for no benefit, sync burden increases.

3. **Concept/workflow focus over command reference**
   - *Why*: Each package command already has comprehensive inline `help`. The spec page fills the gap that inline help can't: the "why", integration with fab, and cross-cutting workflows. Duplicating command details would create drift.
   - *Rejected*: Full command reference in packages.md — duplicates inline help, creates maintenance burden.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Static block in fab-help.sh, not dynamic scanning | Confirmed from intake #1 — packages are stable; dynamic scanning adds complexity for no benefit | S:90 R:90 A:85 D:90 |
| 2 | Certain | Packages section placed after TYPICAL FLOW | Confirmed from intake #2 — visually separates from skills, follows top-down structure | S:85 R:95 A:90 D:85 |
| 3 | Certain | Single packages.md, not per-package spec pages | Confirmed from intake #3 — agreed in discussion; inline help is the reference | S:95 R:85 A:90 D:90 |
| 4 | Confident | Include package architecture section in packages.md | Confirmed from intake #4 — useful for contributors, natural fit in a packages overview | S:60 R:90 A:75 D:80 |
| 5 | Certain | Include wt-pr in wt command listings | wt-pr ships in fab/.kit/packages/wt/bin/ — omitting it would be inaccurate | S:95 R:95 A:95 D:95 |
| 6 | Confident | Reference assembly-line.md rather than duplicating pattern description | assembly-line.md is the authoritative spec for the pattern; packages.md should link, not repeat | S:70 R:85 A:80 D:75 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
