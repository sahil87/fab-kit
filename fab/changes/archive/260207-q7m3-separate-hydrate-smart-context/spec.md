# Spec: Separate Doc Hydration from Init, Add Smart Context Loading, and Index fab/docs

**Change**: 260207-q7m3-separate-hydrate-smart-context
**Created**: 2026-02-07
**Affected docs**: `fab/docs/fab-workflow/hydrate.md`, `fab/docs/fab-workflow/init.md`, `fab/docs/fab-workflow/context-loading.md`

## Fab Workflow: Hydrate Skill

### Requirement: Standalone Hydrate Skill

The system SHALL provide a `/fab:hydrate [sources...]` skill that ingests external documentation into `fab/docs/`. This skill MUST contain all source hydration logic currently in Phase 2 of `fab-init.md`. `/fab:hydrate` MUST be independent of `/fab:init` — it SHALL NOT require init to run first, only that `fab/docs/` exists.

#### Scenario: Hydrate from Notion URL

- **GIVEN** `fab/docs/` exists and contains `index.md`
- **WHEN** the user runs `/fab:hydrate https://notion.so/myteam/API-Spec-abc123`
- **THEN** the Notion page content is fetched via Notion MCP or API
- **AND** the content is analyzed, mapped to domains, and written to `fab/docs/{domain}/{topic}.md`
- **AND** `fab/docs/index.md` is updated with any new domains
- **AND** `fab/docs/{domain}/index.md` is created or updated for each affected domain

#### Scenario: Hydrate from local directory

- **GIVEN** `fab/docs/` exists
- **WHEN** the user runs `/fab:hydrate ./legacy-docs/`
- **THEN** all `.md` files in the directory are read recursively
- **AND** content is analyzed, mapped to domains, and written to `fab/docs/`
- **AND** both top-level and domain-level indexes are updated

#### Scenario: Hydrate without fab/docs/

- **GIVEN** `fab/docs/` does not exist
- **WHEN** the user runs `/fab:hydrate ./some-source`
- **THEN** the skill aborts with: "fab/docs/ not found. Run /fab:init first to create the docs directory."

#### Scenario: Multiple sources in one invocation

- **GIVEN** `fab/docs/` exists
- **WHEN** the user runs `/fab:hydrate https://notion.so/page1 ./local-docs/ https://linear.app/project`
- **THEN** each source is fetched/read independently
- **AND** all sources are merged into `fab/docs/` in a single pass
- **AND** indexes are updated once after all sources are processed

### Requirement: Hydrate Skill File

The skill MUST be defined in `fab/.kit/skills/fab-hydrate.md` following the same conventions as other skill files (frontmatter with name/description, reference to `_context.md`). The skill file MUST be discoverable by `fab-setup.sh`'s existing glob pattern (`fab/.kit/skills/fab-*.md`).

#### Scenario: Symlink auto-discovery

- **GIVEN** `fab/.kit/skills/fab-hydrate.md` exists
- **WHEN** `fab-setup.sh` runs (or `/fab:init` re-runs)
- **THEN** a symlink is created at `.claude/skills/fab-hydrate/SKILL.md` pointing to `../../../fab/.kit/skills/fab-hydrate.md`

### Requirement: Idempotent Hydration

`/fab:hydrate` MUST be safe to run repeatedly with the same sources. Re-hydrating the same content SHALL merge into existing docs without duplicating requirements or overwriting manually-added content.

#### Scenario: Re-hydrate same source

- **GIVEN** `fab/docs/api/endpoints.md` already exists from a previous hydration
- **WHEN** the user runs `/fab:hydrate` with the same source again
- **THEN** new requirements from the source are added
- **AND** existing requirements are updated if the source content changed
- **AND** manually-added content in the doc is preserved

## Fab Workflow: Init Simplification

### Requirement: Init Structural Bootstrap Only

`/fab:init` SHALL perform only structural bootstrap (Phase 1). It MUST NOT accept `[sources...]` arguments. It MUST NOT contain any source hydration logic.

#### Scenario: Init with no arguments

- **GIVEN** `fab/.kit/` exists with a valid VERSION file
- **WHEN** the user runs `/fab:init`
- **THEN** the structural bootstrap runs (config.yaml, constitution.md, docs/index.md, changes/, symlinks, .gitignore)
- **AND** no hydration occurs
- **AND** the output does NOT mention source hydration

#### Scenario: Init with arguments rejected

- **GIVEN** `fab/.kit/` exists
- **WHEN** the user runs `/fab:init https://notion.so/some-page`
- **THEN** the skill outputs: "Did you mean /fab:hydrate? /fab:init no longer accepts source arguments."
- **AND** no hydration occurs

### Requirement: Init Output Updated

The init skill output MUST reflect the simplified scope. The "With Sources" output section MUST be removed. The re-run output MUST NOT mention hydration.

#### Scenario: First run output

- **GIVEN** a fresh project with `fab/.kit/` in place
- **WHEN** the user runs `/fab:init`
- **THEN** the output lists only structural artifacts created (config, constitution, docs/index.md, changes/, symlinks, .gitignore)
- **AND** the next step suggests `/fab:new <description>` or `/fab:hydrate <sources>` to populate docs

## Fab Workflow: Smart Context Loading

### Requirement: Always Load Docs Index

Every skill (except `/fab:init`, `/fab:switch`, `/fab:status`, `/fab:hydrate`) MUST read `fab/docs/index.md` as part of the "Always Load" context layer. This gives the agent awareness of the documentation landscape before generating any artifact.

#### Scenario: Skill loads docs index

- **GIVEN** `fab/docs/index.md` exists and lists domains `auth`, `payments`, `api`
- **WHEN** any fab skill (e.g., `/fab:new`, `/fab:continue`, `/fab:apply`) starts
- **THEN** the skill reads `fab/docs/index.md` as part of its initial context load
- **AND** the agent is aware of existing domains and their docs

### Requirement: Selective Domain Loading

When operating on an active change, skills MUST selectively load relevant domain docs based on the change's scope. The agent SHALL read the proposal's Affected Docs section (or spec's Affected docs metadata) to identify which domains are relevant, then load only those domain indexes and individual doc files.

#### Scenario: Load relevant domain docs for a change

- **GIVEN** an active change whose proposal lists `auth/authentication` under Affected Docs
- **AND** `fab/docs/auth/index.md` exists
- **WHEN** `/fab:continue` runs to generate spec.md
- **THEN** the skill reads `fab/docs/auth/index.md` to understand the domain
- **AND** reads `fab/docs/auth/authentication.md` for the specific doc
- **AND** does NOT read `fab/docs/payments/` or other unrelated domains

#### Scenario: No relevant docs exist yet

- **GIVEN** an active change that references `fab-workflow/hydrate` under New Docs
- **AND** `fab/docs/fab-workflow/` does not exist
- **WHEN** any skill loads context
- **THEN** the skill notes that the referenced doc doesn't exist yet (it will be created by archive)
- **AND** proceeds without error

### Requirement: Context Loading in _context.md

The `_context.md` shared preamble MUST be updated to define the smart loading behavior. The "Always Load" layer MUST include `fab/docs/index.md`. The "Centralized Doc Lookup" layer MUST describe selective loading based on change scope (not just for spec-writing skills, but for all skills operating on an active change).

#### Scenario: Updated _context.md structure

- **GIVEN** the updated `_context.md`
- **WHEN** an agent reads the context loading instructions
- **THEN** section 1 "Always Load" lists `fab/docs/index.md` alongside config.yaml and constitution.md
- **AND** section 3 "Centralized Doc Lookup" applies to all skills operating on an active change, not just spec-writing skills

## Fab Workflow: Doc Indexing

### Requirement: Top-Level Index Maintenance

`fab/docs/index.md` MUST always be a navigable table linking to every domain's `index.md`. Every operation that creates or removes a domain (hydration, archive) MUST update this index. Each row SHALL contain the domain name as a relative link, a description, and a comma-separated list of docs.

#### Scenario: Domain added during hydration

- **GIVEN** `fab/docs/index.md` has no `api` domain entry
- **WHEN** `/fab:hydrate` creates `fab/docs/api/endpoints.md`
- **THEN** `fab/docs/api/index.md` is created with an entry for `endpoints`
- **AND** `fab/docs/index.md` is updated with a row: `| [api](api/index.md) | {description} | endpoints |`

#### Scenario: Domain added during archive

- **GIVEN** `fab/docs/index.md` has no `auth` domain entry
- **WHEN** `/fab:archive` hydrates a spec that introduces `auth/authentication`
- **THEN** `fab/docs/auth/index.md` is created
- **AND** `fab/docs/index.md` is updated with the new domain row

### Requirement: Domain Index Maintenance

Every domain directory MUST contain an `index.md` file. The domain index MUST be a table listing each doc in the domain with its name (as a relative link), description, and last-updated date. Every operation that creates or modifies a doc within a domain MUST update that domain's `index.md`.

#### Scenario: New doc added to existing domain

- **GIVEN** `fab/docs/auth/index.md` exists with entry for `authentication`
- **WHEN** `/fab:hydrate` creates `fab/docs/auth/authorization.md`
- **THEN** `fab/docs/auth/index.md` is updated with a new row for `authorization`

#### Scenario: Doc updated during archive

- **GIVEN** `fab/docs/auth/index.md` lists `authentication` with last-updated 2026-01-15
- **WHEN** `/fab:archive` hydrates updated requirements into `authentication.md`
- **THEN** the `Last Updated` column for `authentication` in `fab/docs/auth/index.md` is updated to today's date

### Requirement: Index Format Consistency

All index files MUST follow the formats defined in `doc/fab-spec/TEMPLATES.md`:
- Top-level: `| [domain](domain/index.md) | description | doc-list |`
- Domain-level: `| [doc-name](doc-name.md) | description | last-updated |`

Relative links MUST be used (not absolute paths) so indexes work regardless of where the repo is cloned.

#### Scenario: Index uses relative links

- **GIVEN** `fab/docs/index.md` is generated or updated
- **WHEN** an agent reads the index
- **THEN** all domain links use relative paths (e.g., `[auth](auth/index.md)`, not `[auth](/fab/docs/auth/index.md)`)

## Fab Spec Documentation: Updates

### Requirement: Spec Docs Reflect New Skill

`doc/fab-spec/SKILLS.md` MUST include a section for `/fab:hydrate` with purpose, arguments, behavior, and examples — following the same format as other skill sections. The existing `/fab:init` section MUST be updated to remove source hydration references.

#### Scenario: SKILLS.md updated

- **GIVEN** the current `doc/fab-spec/SKILLS.md`
- **WHEN** this change is applied
- **THEN** a new `/fab:hydrate [sources...]` section exists between `/fab:init` and `/fab:new`
- **AND** the `/fab:init` section no longer mentions `[sources...]`, source types, or hydration behavior
- **AND** the `/fab:init` next-step suggestion includes `/fab:hydrate`

### Requirement: README Quick Reference Updated

`doc/fab-spec/README.md` quick reference table MUST include `/fab:hydrate` as a new row. The `/fab:init` row MUST be updated to reflect its simplified scope.

#### Scenario: README updated

- **GIVEN** the current `doc/fab-spec/README.md`
- **WHEN** this change is applied
- **THEN** the Quick Reference table has a row: `| /fab:hydrate [sources...] | Ingest external docs into fab/docs/ | Updated fab/docs/ with indexes |`
- **AND** the `/fab:init` row reads: `| /fab:init | Bootstrap fab/ structure | config.yaml, constitution.md, docs/, skill symlinks (idempotent) |`

### Requirement: ARCHITECTURE Bootstrapping Updated

`doc/fab-spec/ARCHITECTURE.md` bootstrap sequence and "Hydrating Docs" section MUST reference `/fab:hydrate` instead of `/fab:init` for source ingestion.

#### Scenario: Bootstrap sequence updated

- **GIVEN** the current `doc/fab-spec/ARCHITECTURE.md`
- **WHEN** this change is applied
- **THEN** the bootstrap sequence reads:
  1. User obtains .kit/
  2. User runs fab-setup.sh
  3. User runs /fab:init (structural bootstrap)
  4. User optionally runs /fab:hydrate (doc ingestion)
  5. User runs /fab:new

## Deprecated Requirements

### Init Source Hydration

**Reason**: Source hydration is being extracted to a dedicated `/fab:hydrate` skill for better separation of concerns. `/fab:init` retains structural bootstrap only.
**Migration**: Use `/fab:hydrate [sources...]` instead of `/fab:init [sources...]` for doc ingestion.
