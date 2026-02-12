# Spec: {CHANGE_NAME}

**Change**: {YYMMDD-XXXX-slug}
**Created**: {DATE}
**Affected docs**: `fab/docs/{domain}/{doc-name}.md`

<!--
  CHANGE SPECIFICATION
  Describes the requirements relevant to this change. No delta markers needed —
  the agent compares against existing centralized docs during hydration to
  determine what's new, changed, or removed.

  Requirements use RFC 2119 keywords: MUST/SHALL (mandatory), SHOULD (recommended), MAY (optional).
  Every requirement MUST have at least one scenario.
  Scenarios use GIVEN/WHEN/THEN format.
  Organize by domain section when the change touches multiple domains.
  Mark unresolved ambiguities with [NEEDS CLARIFICATION] inline. /fab-clarify resolves these.
-->

## Non-Goals

<!--
  OPTIONAL — include when the change has meaningful scope exclusions.
  Omit this section entirely for straightforward changes (no empty headings).

  Each non-goal is a bullet explaining what is explicitly out of scope:
    - {what is excluded} — {brief reason, if not obvious}
-->

- {what is excluded} — {brief reason, if not obvious}

## {Domain}: {Topic}

### Requirement: {Requirement Name}
{Requirement text using SHALL/MUST/SHOULD/MAY}

#### Scenario: {Scenario Name}
- **GIVEN** {precondition}
- **WHEN** {action or event}
- **THEN** {expected outcome}
- **AND** {additional outcome, if needed}

#### Scenario: {Another Scenario}
- **GIVEN** {precondition}
- **WHEN** {action}
- **THEN** {outcome}

### Requirement: {Another Requirement}
{Requirement text}

#### Scenario: {Scenario Name}
- **GIVEN** {precondition}
- **WHEN** {action}
- **THEN** {outcome}

## Design Decisions

<!--
  OPTIONAL — include when the change involves non-trivial choices, architectural
  decisions, or technology selection. Omit entirely for trivial changes.

  Each entry uses the format:
    1. **{Decision}**: {chosen approach}
       - *Why*: {rationale}
       - *Rejected*: {alternative and why it was worse}

  The *Rejected* line MAY be omitted if there were no meaningful alternatives.
-->

1. **{Decision}**: {chosen approach}
   - *Why*: {rationale}
   - *Rejected*: {alternative and why it was worse}

## Deprecated Requirements

<!-- Only include if this change removes existing requirements. -->

### {Requirement Name}
**Reason**: {Why this requirement is being removed}
**Migration**: {What replaces it, or "N/A" if simply deprecated}
