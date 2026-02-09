# Spec-Kit Constitutional System

## Overview

The constitution is the **architectural DNA** of a Spec-Kit project. It defines immutable principles that govern how specifications become code, ensuring consistency, simplicity, and quality across all generated implementations.

## Purpose

1. **Enforce discipline** - Prevent over-engineering and architectural drift
2. **Ensure consistency** - All code follows the same patterns
3. **Enable accountability** - Violations must be justified
4. **Guide AI agents** - Principles constrain LLM behavior

---

## Constitution Location

```
.specify/memory/constitution.md
```

## Constitution Structure

```markdown
# [PROJECT_NAME] Constitution

## Core Principles

### [PRINCIPLE_1_NAME]
<!-- Example: I. Library-First -->
[PRINCIPLE_1_DESCRIPTION]

### [PRINCIPLE_2_NAME]
<!-- Example: II. CLI Interface -->
[PRINCIPLE_2_DESCRIPTION]

### [PRINCIPLE_3_NAME]
<!-- Example: III. Test-First (NON-NEGOTIABLE) -->
[PRINCIPLE_3_DESCRIPTION]

## [SECTION_2_NAME]
<!-- Example: Additional Constraints -->
[SECTION_2_CONTENT]

## Governance

[GOVERNANCE_RULES]

**Version**: [VERSION] | **Ratified**: [DATE] | **Last Amended**: [DATE]
```

---

## Example Principles

The spec-driven.md document describes nine exemplary constitutional articles:

### Article I: Library-First Principle

```markdown
Every feature in [PROJECT] MUST begin its existence as a standalone library.
No feature shall be implemented directly within application code without
first being abstracted into a reusable library component.
```

**Effect**: Forces modular design from the start.

### Article II: CLI Interface Mandate

```markdown
All CLI interfaces MUST:
- Accept text as input (via stdin, arguments, or files)
- Produce text as output (via stdout)
- Support JSON format for structured data exchange
```

**Effect**: Ensures observability and testability.

### Article III: Test-First Imperative

```markdown
This is NON-NEGOTIABLE: All implementation MUST follow strict Test-Driven Development.
No implementation code shall be written before:
1. Unit tests are written
2. Tests are validated and approved by the user
3. Tests are confirmed to FAIL (Red phase)
```

**Effect**: Completely inverts traditional AI code generation.

### Articles VII & VIII: Simplicity and Anti-Abstraction

```markdown
Section 7.3: Minimal Project Structure
- Maximum 3 projects for initial implementation
- Additional projects require documented justification

Section 8.1: Framework Trust
- Use framework features directly rather than wrapping them
```

**Effect**: Combat over-engineering.

### Article IX: Integration-First Testing

```markdown
Tests MUST use realistic environments:
- Prefer real databases over mocks
- Use actual service instances over stubs
- Contract tests mandatory before implementation
```

**Effect**: Ensures code works in practice, not just theory.

---

## Constitutional Enforcement

### Phase Gates in Plan Template

The implementation plan template operationalizes constitution through gates:

```markdown
## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Simplicity Gate (Article VII)
- [ ] Using ≤3 projects?
- [ ] No future-proofing?

### Anti-Abstraction Gate (Article VIII)
- [ ] Using framework directly?
- [ ] Single model representation?

### Integration-First Gate (Article IX)
- [ ] Contracts defined?
- [ ] Contract tests written?
```

### Complexity Tracking

When gates are violated, justification must be documented:

```markdown
## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| 4th project | Need separate auth service | Single project makes testing harder |
| Repository pattern | Multiple data sources | Direct DB access doesn't support caching |
```

---

## Constitution in `/speckit.analyze`

The analyze command treats constitution as **non-negotiable authority**:

```markdown
**Constitution Authority**: The project constitution is **non-negotiable** within
this analysis scope. Constitution conflicts are automatically CRITICAL and require
adjustment of the spec, plan, or tasks—not dilution, reinterpretation, or silent
ignoring of the principle.

If a principle itself needs to change, that must occur in a separate, explicit
constitution update outside `/speckit.analyze`.
```

### Severity Assignment

| Severity | Constitution Criteria |
|----------|----------------------|
| **CRITICAL** | Violates constitution MUST |
| **HIGH** | Missing mandated sections or quality gates |

---

## Versioning and Amendments

### Semantic Versioning

```markdown
**Version**: 2.1.1 | **Ratified**: 2025-06-13 | **Last Amended**: 2025-07-16
```

| Version Type | When |
|--------------|------|
| **MAJOR** | Backward incompatible governance/principle removals |
| **MINOR** | New principle/section added or materially expanded |
| **PATCH** | Clarifications, wording, typo fixes |

### Amendment Process

```markdown
Section 4.2: Amendment Process
Modifications to this constitution require:
- Explicit documentation of the rationale for change
- Review and approval by project maintainers
- Backwards compatibility assessment
```

---

## Creating a Constitution

Use `/speckit.constitution` to create or update:

```markdown
/speckit.constitution Create principles focused on code quality, testing standards,
user experience consistency, and performance requirements.
```

### Execution Flow

1. **Load template** at `/memory/constitution.md`
2. **Identify placeholders** (`[ALL_CAPS_IDENTIFIER]`)
3. **Collect values** from user input or repo context
4. **Draft content** replacing all placeholders
5. **Propagate** to dependent templates
6. **Generate Sync Impact Report**
7. **Validate** and write

### Sync Impact Report

```markdown
<!--
SYNC IMPACT REPORT
Version change: 1.0.0 → 2.0.0
Modified principles: Library-First (renamed), Test-First (expanded)
Added sections: Security Requirements
Removed sections: None
Templates requiring updates:
  - plan-template.md: ✅ updated
  - spec-template.md: ⚠ pending
Follow-up TODOs: None
-->
```

---

## Consistency Propagation

When constitution changes, the `/speckit.constitution` command updates:

1. **`templates/plan-template.md`** - Constitution Check section
2. **`templates/spec-template.md`** - Scope/requirements alignment
3. **`templates/tasks-template.md`** - Task categorization
4. **`templates/commands/*.md`** - Outdated references
5. **Runtime guidance docs** - README, quickstart

---

## Philosophy Beyond Rules

The constitution isn't just a rulebook—it's a **philosophy**:

| Philosophy | Implication |
|------------|-------------|
| **Observability Over Opacity** | Everything inspectable through CLI |
| **Simplicity Over Cleverness** | Start simple, add complexity when proven necessary |
| **Integration Over Isolation** | Test in real environments |
| **Modularity Over Monoliths** | Every feature is a library with clear boundaries |

By embedding these principles into specification and planning, SDD ensures generated code is:
- **Functional**
- **Maintainable**
- **Testable**
- **Architecturally sound**

The constitution transforms AI from a code generator into an **architectural partner** that respects and reinforces system design principles.
