# Spec: Add Reading Order Guide and Documentation Map

**Change**: 260210-q7m3-reading-order-doc-map
**Created**: 2026-02-10
**Affected docs**: `README.md`, `fab/specs/index.md`, `fab/docs/index.md`

## Documentation: README Documentation Map

### Requirement: Audience-Specific Reading Paths

The README Documentation Map SHALL provide three distinct reading paths for different audiences: **new user** (someone evaluating or adopting Fab Kit), **contributor** (someone making changes to the Fab Kit itself), and **spec reader** (someone studying the design and architecture). Each path SHALL list documents in recommended reading order with brief descriptions.

#### Scenario: New user follows the getting-started path

- **GIVEN** a developer opens the README for the first time
- **WHEN** they locate the Documentation Map section
- **THEN** they find a "New User" reading path listing documents in sequence from overview through first-change workflow
- **AND** each entry includes a one-line description of what the document covers

#### Scenario: Contributor follows the contributor path

- **GIVEN** a developer wants to contribute changes to Fab Kit
- **WHEN** they locate the Documentation Map section
- **THEN** they find a "Contributor" reading path that starts with foundational concepts and progresses to internals (architecture, skills reference, templates)
- **AND** the path makes clear which documents build on which (prerequisite ordering)

#### Scenario: Spec reader follows the design path

- **GIVEN** a developer wants to understand the design rationale and architecture
- **WHEN** they locate the Documentation Map section
- **THEN** they find a "Spec Reader" reading path covering the specs directory in recommended order
- **AND** the path links to the glossary as a prerequisite

### Requirement: Grouped Document Inventory

The README SHALL contain a grouped document inventory organizing all Fab documentation (excluding `references/`) into categories. The categories SHALL be: **Getting Started**, **Concepts**, **Reference**, and **Internals**.

#### Scenario: Reader browses by category

- **GIVEN** a reader wants to find a specific type of documentation
- **WHEN** they browse the Document Inventory section
- **THEN** they see documents organized under four category headings
- **AND** each document entry includes its path and a brief description

#### Scenario: references/ directory is excluded

- **GIVEN** the `references/` directory contains self-contained external analysis docs
- **WHEN** the Document Inventory is rendered
- **THEN** it does NOT include files from `references/speckit/` or `references/openspec/`
- **AND** the existing References section in README remains unchanged for those

### Requirement: Glossary Linking

The README Documentation Map, `fab/specs/index.md`, and `fab/docs/index.md` SHALL each include at least one prominent link to the glossary at `fab/specs/glossary.md`. This SHOULD appear near the top of navigational content so readers encounter it before domain-specific terminology.

#### Scenario: README links to glossary

- **GIVEN** a reader is in the Documentation Map section of README
- **WHEN** they scan the reading paths or inventory
- **THEN** they find a visible link to `fab/specs/glossary.md` with a description indicating it defines all Fab terminology

#### Scenario: Index files link to glossary

- **GIVEN** a reader opens `fab/specs/index.md` or `fab/docs/index.md`
- **WHEN** they read the introductory content
- **THEN** they find a note directing newcomers to the README Documentation Map and the glossary

### Requirement: Index File Orientation Notes

Both `fab/specs/index.md` and `fab/docs/index.md` SHALL include a brief orientation note near the top directing newcomers to the README Documentation Map for reading order guidance. The note SHOULD be a single line or short callout, not a duplication of the full map.

#### Scenario: Specs index has orientation note

- **GIVEN** a reader opens `fab/specs/index.md` directly
- **WHEN** they read the top of the file
- **THEN** they see a note like "New here? See the [Documentation Map](../../README.md#documentation-map) for recommended reading order."

#### Scenario: Docs index has orientation note

- **GIVEN** a reader opens `fab/docs/index.md` directly
- **WHEN** they read the top of the file
- **THEN** they see a note like "New here? See the [Documentation Map](../../README.md#documentation-map) for recommended reading order."

## Deprecated Requirements

(none)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Four categories (Getting Started, Concepts, Reference, Internals) rather than three or five | Maps naturally to the existing doc structure; four is the sweet spot between too few (no value) and too many (confusing) |

1 assumption made (1 confident, 0 tentative).
