# Spec-Kit Templates

## Template System Overview

Spec-Kit uses markdown templates with structured sections to guide AI agents toward high-quality, consistent output. Templates act as **sophisticated prompts** that constrain LLM behavior.

## Template Locations

```
.specify/
├── templates/
│   ├── spec-template.md       # Feature specification
│   ├── plan-template.md       # Implementation plan
│   ├── tasks-template.md      # Task breakdown
│   ├── checklist-template.md  # Quality checklists
│   ├── agent-file-template.md # Agent context files
│   └── commands/              # Slash command definitions
│       ├── specify.md
│       ├── plan.md
│       ├── tasks.md
│       ├── implement.md
│       ├── clarify.md
│       ├── analyze.md
│       ├── checklist.md
│       ├── constitution.md
│       └── taskstoissues.md
└── memory/
    └── constitution.md        # Constitution template
```

---

## Spec Template (`spec-template.md`)

**Purpose**: Captures the WHAT and WHY of a feature - no implementation details.

### Structure

```markdown
# Feature Specification: [FEATURE NAME]

**Feature Branch**: `[###-feature-name]`
**Created**: [DATE]
**Status**: Draft
**Input**: User description: "$ARGUMENTS"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - [Brief Title] (Priority: P1)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value]
**Independent Test**: [How to test independently]

**Acceptance Scenarios**:
1. **Given** [state], **When** [action], **Then** [outcome]
2. **Given** [state], **When** [action], **Then** [outcome]

---

### User Story 2 - [Brief Title] (Priority: P2)
...

### Edge Cases

- What happens when [boundary condition]?
- How does system handle [error scenario]?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST [specific capability]
- **FR-002**: System MUST [specific capability]

### Key Entities *(include if feature involves data)*

- **[Entity 1]**: [What it represents, key attributes]
- **[Entity 2]**: [Relationships to other entities]

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: [Measurable metric]
- **SC-002**: [User satisfaction metric]
```

### Key Design Principles

1. **Priority-ordered user stories** - P1 = most critical, delivers standalone value
2. **Independent testability** - Each story can be implemented, tested, deployed alone
3. **Given-When-Then acceptance** - Gherkin-style scenarios
4. **Technology-agnostic** - No mention of languages, frameworks, APIs
5. **Testable requirements** - Every requirement must be verifiable

### Success Criteria Guidelines

**Good** (measurable, technology-agnostic):
- "Users can complete checkout in under 3 minutes"
- "System supports 10,000 concurrent users"
- "95% of searches return results in under 1 second"

**Bad** (implementation-focused):
- "API response time is under 200ms"
- "Database can handle 1000 TPS"
- "React components render efficiently"

---

## Plan Template (`plan-template.md`)

**Purpose**: Technical implementation plan linking spec requirements to architecture decisions.

### Structure

```markdown
# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

## Summary

[Primary requirement + technical approach from research]

## Technical Context

**Language/Version**: [e.g., Python 3.11]
**Primary Dependencies**: [e.g., FastAPI]
**Storage**: [e.g., PostgreSQL]
**Testing**: [e.g., pytest]
**Target Platform**: [e.g., Linux server]
**Project Type**: [single/web/mobile]
**Performance Goals**: [e.g., 1000 req/s]
**Constraints**: [e.g., <200ms p95]
**Scale/Scope**: [e.g., 10k users]

## Constitution Check

*GATE: Must pass before Phase 0 research.*

[Gates determined based on constitution file]

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # /speckit.tasks output
```

### Source Code (repository root)

```text
# Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/
```

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
```

### Key Design Principles

1. **Constitution gates** - Must pass checks before proceeding
2. **Complexity justification** - Document why violations are necessary
3. **Multiple structure options** - Single project, web app, mobile
4. **Research integration** - Links to `research.md` for decisions

---

## Tasks Template (`tasks-template.md`)

**Purpose**: Actionable, dependency-ordered task breakdown organized by user story.

### Structure

```markdown
# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), spec.md (required), data-model.md, contracts/

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1, US2, US3)

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Create project structure per implementation plan
- [ ] T002 Initialize project with dependencies
- [ ] T003 [P] Configure linting and formatting tools

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure MUST complete before ANY user story

- [ ] T004 Setup database schema and migrations
- [ ] T005 [P] Implement authentication framework
- [ ] T006 [P] Setup API routing and middleware

**Checkpoint**: Foundation ready

---

## Phase 3: User Story 1 - [Title] (Priority: P1)

**Goal**: [What this story delivers]
**Independent Test**: [How to verify independently]

### Tests for User Story 1 (OPTIONAL)

- [ ] T010 [P] [US1] Contract test for [endpoint]
- [ ] T011 [P] [US1] Integration test for [journey]

### Implementation for User Story 1

- [ ] T012 [P] [US1] Create [Entity] model in src/models/
- [ ] T013 [US1] Implement [Service] in src/services/
- [ ] T014 [US1] Implement [endpoint] in src/api/

**Checkpoint**: User Story 1 functional and testable

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup**: No dependencies
- **Foundational**: Depends on Setup - BLOCKS all user stories
- **User Stories**: Depend on Foundational, can run in parallel

### Parallel Opportunities

- All [P] tasks can run in parallel
- Different user stories can be worked on by different team members
```

### Key Design Principles

1. **User story organization** - Each story is a separate phase
2. **Independent testability** - Stories deliver value alone
3. **Clear dependencies** - Explicit blocking and parallel markers
4. **MVP-first strategy** - Complete P1 story first, validate, then continue

---

## Constitution Template (`memory/constitution.md`)

**Purpose**: Project governance principles that all development must follow.

### Structure

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
<!-- Example: Additional Constraints, Security Requirements -->
[SECTION_2_CONTENT]

## Governance

[GOVERNANCE_RULES]

**Version**: [CONSTITUTION_VERSION] | **Ratified**: [RATIFICATION_DATE] | **Last Amended**: [LAST_AMENDED_DATE]
```

### Common Principles (Examples)

| Principle | Description |
|-----------|-------------|
| Library-First | Every feature starts as standalone library |
| CLI Interface | All libraries expose CLI (text in/out) |
| Test-First | TDD mandatory, tests before implementation |
| Integration Testing | Focus on contract tests, real environments |
| Simplicity | Start simple, YAGNI principles |
| Anti-Abstraction | Use frameworks directly, no unnecessary wrappers |

---

## Checklist Template (`checklist-template.md`)

**Purpose**: Quality validation checklists generated for specific domains.

### Structure

```markdown
# [CHECKLIST TYPE] Checklist: [FEATURE NAME]

**Purpose**: [What this checklist covers]
**Created**: [DATE]
**Feature**: [Link to spec.md]

## [Category 1]

- [ ] CHK001 First checklist item
- [ ] CHK002 Second checklist item

## [Category 2]

- [ ] CHK003 Another category item
- [ ] CHK004 Item with specific criteria

## Notes

- Check items off as completed: `[x]`
- Add comments or findings inline
```

---

## Template Effectiveness

Templates constrain LLMs toward quality by:

| Constraint | Effect |
|------------|--------|
| Mandatory sections | Nothing forgotten |
| `[NEEDS CLARIFICATION]` markers | Force explicit uncertainty |
| Checklists | Self-review framework |
| Phase gates | Prevent over-engineering |
| Priority ordering | Clear implementation sequence |
| File path requirements | Concrete, actionable tasks |
| Success criteria guidelines | Technology-agnostic outcomes |

The templates transform LLMs from creative writers into **disciplined specification engineers**.
