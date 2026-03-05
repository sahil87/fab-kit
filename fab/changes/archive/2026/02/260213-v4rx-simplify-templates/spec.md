# Spec: Simplify Brief and Spec Templates

**Change**: 260213-v4rx-simplify-templates
**Created**: 2026-02-13
**Affected docs**: `fab/docs/fab-workflow/templates.md`

## Non-Goals

- Changing how downstream skills (`/fab-new`, `/fab-continue`, `/fab-ff`) consume templates — they fill templates dynamically, not by pattern-matching section headings
- Modifying the tasks, checklist, or status templates — only brief.md and spec.md are in scope

## Templates: Brief Affected Docs

### Requirement: Flat Affected Docs List

The brief template's Affected Docs section SHALL use a single flat list with inline markers instead of three headed subsections (New Docs / Modified Docs / Removed Docs).

Each list entry SHALL use the format: `- \`{domain}/{doc-name}\`: ({marker}) {description}` where `{marker}` is one of `new`, `modify`, or `remove`.

The template SHALL NOT include empty subsection headings.

#### Scenario: Single Modified Doc

- **GIVEN** a change that modifies one centralized doc
- **WHEN** the agent fills in the brief template
- **THEN** the Affected Docs section contains a single bullet: `- \`fab-workflow/templates\`: (modify) template structure rules changing`
- **AND** no subsection headings (New Docs / Modified Docs / Removed Docs) appear

#### Scenario: Mixed New and Modified Docs

- **GIVEN** a change that creates one new doc and modifies two existing docs
- **WHEN** the agent fills in the brief template
- **THEN** the Affected Docs section contains three bullets, each with the appropriate `(new)` or `(modify)` marker
- **AND** all entries are at the same list level with no subsection grouping

## Templates: Brief Open Questions

### Requirement: Plain Open Questions List

The brief template's Open Questions section SHALL use a plain list without `[BLOCKING]` or `[DEFERRED]` priority markers.

The template guidance comment SHALL explain that SRAD handles prioritization at spec generation time, making explicit blocking/deferred labels redundant.

#### Scenario: Open Questions Without Priority Markers

- **GIVEN** a brief with unresolved questions
- **WHEN** the agent fills in the Open Questions section
- **THEN** each question is a plain bullet (`- {question}`) with no `[BLOCKING]` or `[DEFERRED]` prefix
- **AND** the template comment explains that SRAD handles prioritization

## Templates: Spec Optional Sections

### Requirement: No Placeholder Content for Optional Sections

The spec template SHALL NOT include placeholder content (example bullets, example entries) for optional sections. Instead, it SHALL include a single guidance comment block listing the optional sections that can be added when needed, along with their format.

The optional sections covered by this comment are: Non-Goals, Design Decisions, and Deprecated Requirements.

#### Scenario: Spec Template Has No Optional Section Scaffolding

- **GIVEN** the spec.md template file
- **WHEN** an agent reads it
- **THEN** the template contains no `## Non-Goals` heading
- **AND** the template contains no `## Design Decisions` heading
- **AND** the template contains no `## Deprecated Requirements` heading
- **AND** a single guidance comment block describes all three optional sections and when to include them

#### Scenario: Agent Adds Non-Goals When Needed

- **GIVEN** a change with meaningful scope exclusions
- **WHEN** the agent generates a spec using the template
- **THEN** the agent adds a `## Non-Goals` section after metadata and before domain sections
- **AND** the format follows the pattern described in the guidance comment

#### Scenario: Agent Omits Optional Sections for Simple Changes

- **GIVEN** a straightforward change with no scope exclusions, no architectural decisions, and no deprecated requirements
- **WHEN** the agent generates a spec
- **THEN** no Non-Goals, Design Decisions, or Deprecated Requirements sections appear in the output

### Requirement: Deprecated Requirements Documented as Pattern

The Deprecated Requirements section SHALL NOT appear as a standing template section. Instead, the guidance comment for optional sections SHALL document it as an available pattern: include when a change removes existing requirements, with Reason and Migration fields.

#### Scenario: Deprecated Requirements Pattern in Comment

- **GIVEN** the spec.md template file
- **WHEN** an agent reads the optional sections guidance comment
- **THEN** the comment includes Deprecated Requirements with its format (Reason + Migration) and when to use it (only when removing existing requirements)

## Shared Context: Affected Docs Wording

### Requirement: Context Loading Wording Update

The Centralized Doc Lookup section in `_context.md` (Section 3, step 3) SHALL update the parenthetical "(the New, Modified, and Removed entries)" to match the new flat list format.

The updated wording SHALL reference inline markers rather than subsection names.

#### Scenario: Context Wording Matches Flat List

- **GIVEN** the `_context.md` file
- **WHEN** an agent reads Section 3, step 3
- **THEN** the wording references the flat list entries with inline markers (e.g., "entries marked `(new)`, `(modify)`, or `(remove)`")
- **AND** does not reference "New, Modified, and Removed" subsections

## Centralized Docs: Templates Doc Update

### Requirement: Templates Doc Reflects New Structure

The centralized doc `fab/docs/fab-workflow/templates.md` SHALL be updated to reflect the simplified template structures:

1. The `brief.md` section SHALL describe the flat Affected Docs list with inline markers instead of three subsections
2. The `brief.md` section SHALL describe Open Questions as a plain list without BLOCKING/DEFERRED markers, noting that SRAD handles prioritization
3. The `spec.md` section SHALL describe optional sections (Non-Goals, Design Decisions, Deprecated Requirements) as patterns to include when needed, not standing template sections

#### Scenario: Templates Doc Describes Flat Affected Docs

- **GIVEN** the centralized templates doc
- **WHEN** a reader looks at the brief.md requirements
- **THEN** the Affected Docs description references a flat list with `(new)`, `(modify)`, `(remove)` inline markers
- **AND** does not mention three headed subsections

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | `_context.md` Section 3 wording needs updating | The parenthetical explicitly references "New, Modified, and Removed entries" which maps to the old 3-subsection structure; brief's Impact section also flags this |
| 2 | Confident | `_generation.md` spec procedure needs no structural change | Step 5b already says "optional — include when needed / omit entirely" which matches the new approach; only the template itself changes |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
