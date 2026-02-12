# Spec: Add Non-Goals and Design Decisions to Spec Template

**Change**: 260211-r4w8-spec-template-sections
**Created**: 2026-02-11
**Affected docs**: `fab/docs/fab-workflow/templates.md`

## Non-Goals

- Updating existing specs generated before this change — they remain valid without the new sections
- Making the sections required — both are explicitly optional
- Changing the brief template — the brief captures motivation, not design rationale
- Adding other old plan sections (Technical Context, Research, File Changes) — out of scope

## Template Structure: Non-Goals Section

### Requirement: Non-Goals section placement

The `## Non-Goals` section SHALL be placed after the metadata/header block and before the first `## {Domain}: {Topic}` requirements section.

#### Scenario: Spec with Non-Goals
- **GIVEN** a spec being generated for a change with known scope exclusions
- **WHEN** the agent fills in the spec template
- **THEN** the Non-Goals section contains bullet points listing explicit exclusions
- **AND** each bullet explains what is out of scope and optionally why

#### Scenario: Spec without Non-Goals
- **GIVEN** a spec being generated for a straightforward change with no meaningful exclusions
- **WHEN** the agent determines Non-Goals are not needed
- **THEN** the Non-Goals section is omitted entirely from the generated spec

### Requirement: Non-Goals format

Each non-goal SHALL be a bullet point with a concise statement of what is excluded.

#### Scenario: Non-Goal bullet format
- **GIVEN** a non-goal to document
- **WHEN** writing the Non-Goals section
- **THEN** the entry follows the format `- {what is excluded} — {brief reason, if not obvious}`

## Template Structure: Design Decisions Section

### Requirement: Design Decisions section placement

The `## Design Decisions` section SHALL be placed after all `## {Domain}: {Topic}` requirements sections and before `## Deprecated Requirements`.

#### Scenario: Spec with Design Decisions
- **GIVEN** a spec for a change involving non-trivial choices
- **WHEN** the agent fills in the spec template
- **THEN** the Design Decisions section contains numbered entries documenting each key choice

#### Scenario: Spec without Design Decisions
- **GIVEN** a spec for a trivial change with no meaningful design choices
- **WHEN** the agent determines Design Decisions are not needed
- **THEN** the Design Decisions section is omitted entirely from the generated spec

### Requirement: Design Decisions entry format

Each design decision SHALL use a structured format capturing the choice, rationale, and rejected alternatives.

#### Scenario: Design Decision entry
- **GIVEN** a design decision to document
- **WHEN** writing the Design Decisions section
- **THEN** the entry follows the format:
  ```
  1. **{Decision}**: {chosen approach}
     - *Why*: {rationale}
     - *Rejected*: {alternative and why it was worse}
  ```
- **AND** the *Rejected* line MAY be omitted if there were no meaningful alternatives

## Skill Updates: Spec Generation Awareness

### Requirement: fab-continue spec stage awareness

The `fab-continue` skill prompt SHALL include instructions to populate Non-Goals and Design Decisions sections when generating spec content, based on brief context and prior discussion.

#### Scenario: fab-continue generates spec with decisions
- **GIVEN** a change with design decisions captured in the brief or discussion
- **WHEN** fab-continue generates the spec at the spec stage
- **THEN** the generated spec includes a populated Design Decisions section

#### Scenario: fab-continue generates spec for trivial change
- **GIVEN** a change with no non-trivial design choices or scope exclusions
- **WHEN** fab-continue generates the spec at the spec stage
- **THEN** the generated spec omits both optional sections

### Requirement: fab-ff spec stage awareness

The `fab-ff` skill prompt SHALL include the same awareness of Non-Goals and Design Decisions as fab-continue when fast-forwarding through the spec stage.

#### Scenario: fab-ff fast-forwards through spec stage
- **GIVEN** a change being fast-forwarded through planning stages
- **WHEN** fab-ff generates spec content
- **THEN** it follows the same rules as fab-continue for including or omitting optional sections

## Design Decisions

1. **Non-Goals before requirements, Design Decisions after**: Reading flow goes scope boundaries -> requirements -> rationale
   - *Why*: A reader needs to know what's excluded before reading what's included; rationale makes more sense after seeing the requirements it justifies
   - *Rejected*: Both sections at the end — loses the scoping benefit of Non-Goals appearing early

2. **Reuse old plan's Decision/Why/Rejected format**: Structured but lightweight
   - *Why*: Proven useful in the old plan template; captures the three essential pieces (what, why, what else)
   - *Rejected*: Free-form prose — harder to scan, easier to omit key information

3. **Optional with omission, not optional with empty sections**: Skip entirely rather than leaving empty headings
   - *Why*: Empty sections add noise; omission is cleaner and consistent with how Deprecated Requirements already works
   - *Rejected*: Always include with "N/A" — adds boilerplate to simple changes

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | fab-continue and fab-ff are the skills to update | These are the two skills that generate spec content via `_generation.md` |
| 2 | Confident | Guidance via HTML comments in template | Consistent with existing template patterns |
