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

## Deprecated Requirements

<!-- Only include if this change removes existing requirements. -->

### {Requirement Name}
**Reason**: {Why this requirement is being removed}
**Migration**: {What replaces it, or "N/A" if simply deprecated}
