# Spec: Consistency Fixes from 260214 Audit

**Change**: 260214-eikh-consistency-fixes
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/kit-architecture.md` (modify), `fab/memory/fab-workflow/execution-skills.md` (verify only — cf13 resolved)

## Non-Goals

- Behavioral code changes — no skill files, shell scripts, or templates are modified
- Restructuring spec files — only in-place corrections to existing content
- Rewriting sections for style — minimal edits scoped to fixing incorrect references

---

## Specs: Stage Terminology Alignment (cf01, cf06)

### Requirement: Stage 6 SHALL be named "hydrate" in all spec files

All references to stage 6 as "archive" or "Archive" in spec files SHALL be replaced with "hydrate" or "Hydrate" (matching case context). The stage name "hydrate" is the canonical name defined in `workflow.yaml`. `/fab-archive` is a standalone housekeeping command, not a pipeline stage.

Affected locations:
- `overview.md:124` — stage table row "6 | **Archive**" → "6 | **Hydrate**"
- `glossary.md:23` — stage list "brief, spec, tasks, apply, review, archive" → "..., review, hydrate"
- `glossary.md:36` — "**Archive** | Stage 6." → "**Hydrate** | Stage 6."
- `skills.md:273` — reset guard listing "archive" → "hydrate"
- `skills.md:339` — "Step 4 — Archive (fab-archive)" → "Step 4 — Hydrate (fab-continue)"
- `skills.md:341` — "execution stages (apply, review, archive)" → "execution stages (apply, review, hydrate)"
- `glossary.md:13` — "from brief through archive" → "from brief through hydrate"
- `glossary.md:19` — "Cleared by `/fab-continue` (archive)" → "Cleared by `/fab-archive`"

#### Scenario: Stage 6 reference in overview stage table

- **GIVEN** a reader views the stage table in `overview.md`
- **WHEN** they read stage 6
- **THEN** the row shows "Hydrate" as the stage name (not "Archive")

#### Scenario: Glossary stage list

- **GIVEN** a reader views the Stage definition in `glossary.md`
- **WHEN** they read the list of stages
- **THEN** the sixth stage is "hydrate" (not "archive")

### Requirement: "Archive Behavior" section in skills.md SHALL be retitled to "Hydrate Behavior"

The section at `skills.md:443` titled "Archive Behavior (via `/fab-continue`)" SHALL be retitled to "Hydrate Behavior (via `/fab-continue`)" to match the canonical stage name. The section content already describes hydration behavior, so only the heading changes.

#### Scenario: skills.md section heading

- **GIVEN** a reader navigates to the execution behavior sections in `skills.md`
- **WHEN** they find the stage-6 behavior section
- **THEN** the heading reads "Hydrate Behavior (via `/fab-continue`)"

---

## Specs: Skill Rename Alignment (cf02, cf05)

### Requirement: All `/fab-hydrate` references SHALL be replaced with `/docs-hydrate-memory`

The skill `/fab-hydrate` was renamed to `/docs-hydrate-memory`. All occurrences in spec files SHALL be updated. This affects approximately 24 occurrences across `overview.md`, `skills.md`, `glossary.md`, `architecture.md`, and `user-flow.md`.

Key locations include:
- `overview.md:71,75,78,81,162` — usage examples and quick reference
- `skills.md:13,24,58,59,78,95,103,108,154,174,180,185` — skill descriptions, context loading, next-step table, redirect message, and the skill's own section heading
- `glossary.md:12,17,45` — Memory files definition, Hydration definition, skill entry
- `architecture.md:392,400,419` — bootstrap sequence and re-run guidance
- `user-flow.md:78` — flowchart node label

#### Scenario: Skill reference in overview quick start

- **GIVEN** a reader follows the quick-start guide in `overview.md`
- **WHEN** they see the command for ingesting external documentation
- **THEN** the command reads `/docs-hydrate-memory` (not `/fab-hydrate`)

#### Scenario: Skill section heading in skills.md

- **GIVEN** a reader navigates to the hydration skill section in `skills.md`
- **WHEN** they find the section heading
- **THEN** it reads `/docs-hydrate-memory [sources...]`

### Requirement: Symlink references to `fab-hydrate.md` SHALL be updated to `docs-hydrate-memory.md`

In `architecture.md`, the directory listing and symlink examples referencing `fab-hydrate.md` SHALL be updated to `docs-hydrate-memory.md`, matching the actual filename in `.kit/skills/`.

Affected locations:
- `architecture.md:24` — directory tree listing `fab-hydrate.md`
- `architecture.md:361-362` — symlink example `fab-hydrate/` → `fab-hydrate.md`

#### Scenario: Symlink path in architecture directory listing

- **GIVEN** a reader views the `.kit/skills/` directory listing in `architecture.md`
- **WHEN** they see the hydration skill entry
- **THEN** the filename reads `docs-hydrate-memory.md`

---

## Schema: Command Reference Fixes (cf03)

### Requirement: workflow.yaml SHALL reference `fab-continue` for apply and review stages

The `commands` field for the apply stage (line 90) and review stage (line 100) in `fab/.kit/schemas/workflow.yaml` SHALL reference `fab-continue` instead of `fab-apply` and `fab-review`, since the standalone apply/review skills were consolidated into `/fab-continue`.

#### Scenario: Apply stage command in workflow.yaml

- **GIVEN** a tool reads the apply stage definition from `workflow.yaml`
- **WHEN** it inspects the `commands` field
- **THEN** the value is `[fab-continue]` (not `[fab-apply]`)

#### Scenario: Review stage command in workflow.yaml

- **GIVEN** a tool reads the review stage definition from `workflow.yaml`
- **WHEN** it inspects the `commands` field
- **THEN** the value is `[fab-continue]` (not `[fab-review]`)

---

## Specs: File Reference Case Fixes (cf04)

### Requirement: Cross-references SHALL use correct lowercase filenames

References to `SKILLS.md` and `TEMPLATES.md` (uppercase) SHALL be corrected to `skills.md` and `templates.md` (lowercase) to match the actual filenames.

Affected locations:
- `architecture.md:400` — `[Skills Reference](SKILLS.md#fabhydrate-sources)` → `[Skills Reference](skills.md#...)`
- `skills.md:199` — `[Memory File Format](TEMPLATES.md#memory-file-format-fabmemory)` and `[Hydration Rules](TEMPLATES.md#hydration-rules)` → lowercase `templates.md`

Note: The anchor fragment in architecture.md:400 (`#fabhydrate-sources`) will also need updating since the section was renamed (cf02). The corrected anchor SHALL match the new section heading for `/docs-hydrate-memory`.
<!-- assumed: Anchor fragment will be updated to match renamed section heading — clear dependency on cf02 -->

#### Scenario: Architecture cross-reference to skills spec

- **GIVEN** a reader follows the skills reference link in `architecture.md`
- **WHEN** the link is resolved
- **THEN** it points to `skills.md` (lowercase) with a valid anchor

---

## Specs: Missing Coverage Additions (cf07, cf08, cf09, cf14, cf15)

### Requirement: `/docs-reorg-memory` and `/docs-reorg-specs` SHALL have behavioral spec sections in skills.md

Both skills exist in `.kit/skills/` and are user-facing but have no spec coverage. A new section SHALL be added to `skills.md` for each, documenting: purpose, arguments, behavior summary, and output. The sections SHOULD follow the same format as existing skill sections in the file.

#### Scenario: Reader looks up docs-reorg-memory in skills.md

- **GIVEN** a reader searches for `/docs-reorg-memory` in `skills.md`
- **WHEN** the search completes
- **THEN** a section exists with purpose, arguments, behavior, and output

#### Scenario: Reader looks up docs-reorg-specs in skills.md

- **GIVEN** a reader searches for `/docs-reorg-specs` in `skills.md`
- **WHEN** the search completes
- **THEN** a section exists with purpose, arguments, behavior, and output

### Requirement: Batch scripts SHALL be documented in architecture.md

The batch scripts (`batch-archive-change.sh`, `batch-new-backlog.sh`, `batch-switch-change.sh`) are only mentioned in a naming convention table (`architecture.md:94`). A dedicated subsection SHALL be added describing each script's purpose, arguments, and behavior. Content SHOULD be derived from the existing documentation in `fab/memory/fab-workflow/kit-architecture.md` (Batch Scripts section).

#### Scenario: Reader looks up batch scripts in architecture.md

- **GIVEN** a reader searches for batch script documentation in `architecture.md`
- **WHEN** they find the batch scripts section
- **THEN** each script has its purpose, argument handling, and tmux integration documented

### Requirement: Internal skills SHALL be listed in architecture.md directory tree

The `.kit/skills/` directory listing in `architecture.md` SHALL include the three internal skills: `internal-consistency-check.md`, `internal-retrospect.md`, `internal-skill-optimize.md`. These exist in the implementation but are absent from the spec's directory listing.

#### Scenario: Directory listing completeness

- **GIVEN** a reader views the `.kit/skills/` directory tree in `architecture.md`
- **WHEN** they scan for internal skills
- **THEN** entries exist for `internal-consistency-check.md`, `internal-retrospect.md`, and `internal-skill-optimize.md`

### Requirement: `docs-reorg-*` skills SHALL be added to glossary.md

`/docs-reorg-memory` and `/docs-reorg-specs` SHALL be added to the skills section of `glossary.md` as user-facing skills with brief descriptions.

#### Scenario: Glossary skill lookup

- **GIVEN** a reader searches the glossary for `/docs-reorg-memory`
- **WHEN** they find the skills table
- **THEN** an entry exists describing it as a user-facing reorganization skill for memory files

### Requirement: Glossary "Hydration" definition SHALL cover dual-mode behavior

The glossary entry for "Hydration" (`glossary.md:17`) SHALL be expanded to describe both modes: (1) pipeline hydration — merging change artifacts into memory via `/fab-continue`, and (2) source ingestion/generation — ingesting external docs or generating from codebase via `/docs-hydrate-memory`. The current definition only covers the merging aspect.

#### Scenario: Reader looks up Hydration in glossary

- **GIVEN** a reader reads the Hydration glossary entry
- **WHEN** they review the definition
- **THEN** it describes both pipeline hydration (via `/fab-continue`) and source hydration with dual modes (ingest + generate) via `/docs-hydrate-memory`

---

## Specs: Stale Content Removal (cf11, cf12)

### Requirement: `/fab-init-config`, `/fab-init-constitution`, `/fab-init-validate` documentation SHALL be removed from skills.md

The sub-skill sections at `skills.md:119-150` document `/fab-init-config`, `/fab-init-constitution`, and `/fab-init-validate` as standalone skills. These sub-skills do not exist — only monolithic `/fab-init` exists. The three sections SHALL be removed entirely.

#### Scenario: Sub-skill section absent after fix

- **GIVEN** a reader searches `skills.md` for `/fab-init-config`
- **WHEN** the search completes
- **THEN** no section exists for this sub-skill (or the other two)

### Requirement: Orphaned memory file entries SHALL be removed from memory/index.md

The entries `hydrate-design` and `design-index` in the `fab-workflow` domain's memory file list (`fab/memory/index.md:14`) SHALL be removed. These files do not exist on disk.

#### Scenario: Memory index accuracy

- **GIVEN** a reader views the `fab-workflow` domain in `memory/index.md`
- **WHEN** they cross-reference the listed files against the filesystem
- **THEN** every listed file exists (no orphaned entries)

---

## Deprecated Requirements

### cf13: Contradictory changelog entries about fab-status.sh/stageman.sh

**Reason**: Verified during spec generation — the `260214-r8kv-docs-skills-housekeeping` change already cleaned up `fab-status.sh` references in both `execution-skills.md` and `kit-architecture.md`. No contradictions remain.
**Migration**: N/A — already resolved.

---

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | cf13 is already resolved — skip during apply | Grep confirmed no contradictory references remain; changelog entries are consistent |
| 2 | Confident | cf04 anchor fragment `#fabhydrate-sources` needs updating alongside filename fix | Clear dependency: the section being linked was renamed (cf02), so the anchor must track |
| 3 | Confident | cf07 section content derived from reading actual `.kit/skills/docs-reorg-*.md` files at apply time | Spec sections should reflect actual implemented behavior, not guessed behavior |
| 4 | Confident | cf08 batch script docs derived from memory file `kit-architecture.md` (Batch Scripts section) | Memory file already has thorough batch script documentation to draw from |

4 assumptions made (4 confident, 0 tentative). Run /fab-clarify to review.
