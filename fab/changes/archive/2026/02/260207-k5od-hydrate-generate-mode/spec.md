# Spec: Add "generate" mode to fab-hydrate

**Change**: 260207-k5od-hydrate-generate-mode
**Created**: 2026-02-07
**Affected docs**: `fab/docs/fab-workflow/hydrate.md`, `fab/docs/fab-workflow/hydrate-generate.md`

## fab-workflow: Argument-Driven Mode Selection

### Requirement: Unified Argument Routing

`/fab:hydrate` SHALL determine its operating mode from the type of arguments provided, with no flags or subcommands:

| Argument type | Detection rule | Mode |
|---|---|---|
| No arguments | Argument list is empty | Generate (scan from project root) |
| URL | Matches `notion.so`, `notion.site`, `linear.app`, or `http(s)://` | Ingest |
| Markdown file | Path ends with `.md` | Ingest |
| Folder | Path resolves to an existing directory | Generate |

When multiple arguments are provided, they MUST all resolve to the same mode. Mixed-mode invocations (e.g., a URL and a folder) SHALL be rejected with an error.

#### Scenario: No arguments triggers generate mode
- **GIVEN** the user runs `/fab:hydrate` with no arguments
- **WHEN** the skill starts
- **THEN** it enters generate mode, scanning from the project root

#### Scenario: URL argument triggers ingest mode
- **GIVEN** the user runs `/fab:hydrate https://notion.so/my-page`
- **WHEN** the skill routes the argument
- **THEN** it enters ingest mode and fetches the URL

#### Scenario: Markdown file triggers ingest mode
- **GIVEN** the user runs `/fab:hydrate ./docs/api-spec.md`
- **WHEN** the skill routes the argument
- **THEN** it enters ingest mode and reads the file

#### Scenario: Folder argument triggers generate mode
- **GIVEN** the user runs `/fab:hydrate ./src/`
- **WHEN** the skill routes the argument
- **THEN** it enters generate mode, scanning the specified folder

#### Scenario: Mixed-mode arguments rejected
- **GIVEN** the user runs `/fab:hydrate https://notion.so/page ./src/`
- **WHEN** the skill classifies arguments
- **THEN** it reports an error: "Cannot mix ingest sources (URLs, .md files) with generate targets (folders). Run separately."
- **AND** no processing occurs

### Requirement: No-Args Replaces Usage Error

When `/fab:hydrate` is invoked with no arguments, it SHALL enter generate mode instead of displaying the current usage error message. The existing "Usage: /fab:hydrate ..." abort behavior is removed.

#### Scenario: No-args behavior change
- **GIVEN** the user runs `/fab:hydrate` with no arguments
- **WHEN** the skill starts
- **THEN** it does NOT display a usage error
- **AND** it enters generate mode scanning from project root

## fab-workflow: Generate Mode — Codebase Scanning

### Requirement: Codebase Gap Detection

In generate mode, the skill SHALL scan source code to identify undocumented areas by comparing codebase structure against existing `fab/docs/`. The scan MUST identify:

- **Public APIs**: Exported functions, classes, endpoints, CLI commands
- **Modules**: Top-level directories and packages with distinct responsibilities
- **Architectural patterns**: Recurring patterns (middleware chains, plugin systems, event buses, etc.)
- **Configuration**: Config files, environment variables, feature flags
- **Conventions**: Naming patterns, file organization, coding standards evident from code

The scan SHOULD use file system exploration (Glob, Grep, Read) to analyze the codebase. It SHALL NOT require external tools or dependencies (Constitution I: Pure Prompt Play).

#### Scenario: Scan identifies undocumented module
- **GIVEN** the project has a `src/auth/` directory with authentication logic
- **AND** `fab/docs/` has no `auth` domain or related docs
- **WHEN** the generate scan runs
- **THEN** `src/auth/` appears in the gap report as an undocumented module

#### Scenario: Scan respects already-documented areas
- **GIVEN** `fab/docs/fab-workflow/hydrate.md` documents the hydrate skill
- **AND** the codebase contains `fab/.kit/skills/fab-hydrate.md`
- **WHEN** the generate scan runs
- **THEN** the hydrate skill does NOT appear as an undocumented gap (or appears with reduced priority)

### Requirement: Scan Scope

When folder paths are provided as arguments, the scan SHALL be limited to those paths. When no arguments are provided, the scan SHALL start from the project root. The scan SHOULD respect common ignore patterns (`.git/`, `node_modules/`, `vendor/`, `__pycache__/`, `dist/`, `build/`).

#### Scenario: Scoped scan to specific folder
- **GIVEN** the user runs `/fab:hydrate ./src/api/`
- **WHEN** the scan executes
- **THEN** only files within `./src/api/` (and subdirectories) are analyzed
- **AND** files outside `./src/api/` are not scanned

#### Scenario: Full project scan
- **GIVEN** the user runs `/fab:hydrate` with no arguments
- **WHEN** the scan executes
- **THEN** the entire project root is scanned
- **AND** standard ignore patterns are applied

## fab-workflow: Generate Mode — Interactive Scoping

### Requirement: Gap Report Presentation

After scanning, the skill SHALL present a gap report — a categorized, prioritized list of discovered documentation gaps. Each gap MUST include:

- **Category**: Module, API, Pattern, Configuration, or Convention
- **Name**: Human-readable identifier (e.g., "auth module", "REST API endpoints")
- **Location**: File paths or directory paths involved
- **Priority suggestion**: High (core functionality, public API), Medium (internal modules, patterns), Low (utilities, config)

The report SHALL be grouped by category and sorted by priority within each category.

#### Scenario: Gap report for a medium codebase
- **GIVEN** a codebase with 5 undocumented areas identified
- **WHEN** the gap report is presented
- **THEN** gaps are grouped by category (Modules, APIs, Patterns, etc.)
- **AND** each gap shows name, location, and priority
- **AND** the user can see all gaps before selecting

### Requirement: Interactive Selection

After presenting the gap report, the skill SHALL offer the user batch selection of which gaps to document. The skill MUST support:

- **Select all**: Document everything found
- **Select by category**: e.g., "all APIs"
- **Select individually**: Pick specific gaps by number/name
- **Select by priority**: e.g., "all High priority"

The selection interface SHOULD use the AskUserQuestion tool for structured input. If only 1-3 gaps are found, the skill MAY skip the interactive prompt and proceed to document all of them (with a brief confirmation).

#### Scenario: User selects subset of gaps
- **GIVEN** 8 documentation gaps were identified
- **WHEN** the gap report is presented
- **AND** the user selects 3 specific gaps
- **THEN** only those 3 gaps are documented
- **AND** the remaining 5 are not processed

#### Scenario: Small number of gaps skips selection
- **GIVEN** 2 documentation gaps were identified
- **WHEN** the gap report is presented
- **THEN** the skill confirms: "Found 2 undocumented areas. Document both?"
- **AND** on confirmation, documents both without full selection UI

#### Scenario: User selects by priority
- **GIVEN** gaps include 3 High, 4 Medium, 2 Low priority items
- **WHEN** the user selects "all High priority"
- **THEN** the 3 High priority gaps are documented
- **AND** Medium and Low priority gaps are skipped

## fab-workflow: Generate Mode — Doc Generation

### Requirement: Structured Doc Output

For each selected gap, the skill SHALL generate a documentation file in `fab/docs/{domain}/{topic}.md` following the centralized doc format:

- **Overview**: What the module/API/pattern does, inferred from code analysis
- **Requirements**: Key behaviors documented as requirements using RFC 2119 keywords, derived from code behavior (not invented)
- **Design Decisions**: Architectural choices evident from the code, with rationale where inferable
- **Changelog**: Initial entry noting the doc was generated from code analysis

Generated docs SHOULD be accurate to what the code actually does, not aspirational. When behavior is ambiguous from code alone, the doc SHOULD note this with `[INFERRED]` markers.

#### Scenario: Generate doc for a module
- **GIVEN** the user selected the `auth` module gap
- **AND** `src/auth/` contains middleware, token validation, and session management
- **WHEN** the doc is generated
- **THEN** `fab/docs/auth/index.md` is created (domain index)
- **AND** one or more docs are created under `fab/docs/auth/` covering the module's behavior
- **AND** each doc includes Overview, Requirements, Design Decisions, Changelog sections

#### Scenario: Ambiguous behavior marked
- **GIVEN** a function has unclear side effects or implicit behavior
- **WHEN** the doc is generated
- **THEN** the requirement includes an `[INFERRED]` marker
- **AND** a note explains what was inferred and suggests verification

### Requirement: Index Maintenance

Generate mode SHALL reuse the same index maintenance logic as ingest mode. After generating docs:

1. Create or update `fab/docs/{domain}/index.md` for each domain touched
2. Update `fab/docs/index.md` with new domains and doc lists
3. All links SHALL be relative
4. Existing entries SHALL NOT be removed

#### Scenario: New domain created
- **GIVEN** no `fab/docs/api/` domain exists
- **WHEN** generate mode creates `fab/docs/api/endpoints.md`
- **THEN** `fab/docs/api/index.md` is created with an entry for `endpoints`
- **AND** `fab/docs/index.md` is updated with a row for the `api` domain

### Requirement: Idempotent Generation

Generate mode SHALL be safe to re-run. On re-generation:

- Existing generated docs SHALL be updated (merged), not overwritten
- Manually-added content in docs SHALL be preserved
- New gaps discovered since last run SHALL appear in the gap report
- Previously documented areas SHALL NOT appear as gaps (or appear with lower priority)

#### Scenario: Re-run after manual edits
- **GIVEN** `/fab:hydrate` generated `fab/docs/api/endpoints.md`
- **AND** a user manually added a Design Decision to that doc
- **WHEN** `/fab:hydrate` is run again (generate mode)
- **THEN** `fab/docs/api/endpoints.md` is updated with any new code findings
- **AND** the manually-added Design Decision is preserved

## fab-workflow: Ingest Mode Preservation

### Requirement: Ingest Behavior Unchanged

All existing ingest mode behavior (URLs, markdown files) SHALL remain identical. The only change to ingest mode is the removal of the no-args usage error (replaced by generate mode).

#### Scenario: URL ingest unchanged
- **GIVEN** the user runs `/fab:hydrate https://notion.so/api-spec`
- **WHEN** the skill processes the argument
- **THEN** it fetches and ingests the Notion page exactly as before this change

#### Scenario: Markdown file ingest unchanged
- **GIVEN** the user runs `/fab:hydrate ./legacy-docs/api.md`
- **WHEN** the skill processes the argument
- **THEN** it reads and ingests the markdown file exactly as before this change

## Deprecated Requirements

(none — no existing requirements are removed, only the no-args error behavior is replaced)
