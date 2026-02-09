# OpenSpec Conventions

## Naming Conventions

### Change Names

Use kebab-case with descriptive intent:

**Good:**
```
add-dark-mode
fix-login-redirect
optimize-product-query
implement-2fa
refactor-auth-flow
```

**Avoid:**
```
feature-1          # Non-descriptive
update             # Too vague
changes            # Generic
wip                # Temporary-sounding
my-feature         # Personal prefix
```

### Spec Organization

Organize specs by domain, component, or bounded context:

**By Feature Area:**
```
specs/
├── auth/
│   ├── login.md
│   └── permissions.md
├── payments/
│   ├── checkout.md
│   └── refunds.md
└── search/
    └── product-search.md
```

**By Component:**
```
specs/
├── api/
├── frontend/
└── workers/
```

**By Bounded Context:**
```
specs/
├── ordering/
├── fulfillment/
└── inventory/
```

### Artifact Filenames

Standard artifact names (from default schema):
- `proposal.md` - Change proposal
- `design.md` - Technical design
- `tasks.md` - Implementation tasks
- `specs/*.md` - Delta specifications

## Spec Document Format

### Required Sections

Every spec must have:

```markdown
# [Domain] Specification

## Purpose
High-level description of what this spec covers.

## Requirements
The actual requirements with scenarios.
```

### Requirement Structure

```markdown
### Requirement: [Descriptive Name]

The system SHALL/MUST/SHOULD [behavior description].

#### Scenario: [Scenario Name]
- GIVEN [initial context/state]
- WHEN [action/trigger occurs]
- THEN [expected outcome]
- AND [additional outcomes if needed]
```

### RFC 2119 Keywords

Use standard requirement keywords:

| Keyword | Meaning | Usage |
|---------|---------|-------|
| `MUST` / `SHALL` | Absolute requirement | Core functionality |
| `MUST NOT` / `SHALL NOT` | Absolute prohibition | Security constraints |
| `SHOULD` | Recommended | Best practices |
| `SHOULD NOT` | Not recommended | Discouraged patterns |
| `MAY` | Optional | Nice-to-haves |

### Delta Spec Sections

For changes to existing specs:

```markdown
## ADDED Requirements

### Requirement: New Feature
The system SHALL [new behavior].

## MODIFIED Requirements

### Requirement: Existing Feature
The system SHALL [updated behavior].

## REMOVED Requirements

### Requirement: Deprecated Feature
<!-- This requirement is being removed -->
```

## Task Format

### Structure

```markdown
# Tasks

## 1. Setup
- [ ] 1.1 Create database migration
- [ ] 1.2 Add configuration entries

## 2. Implementation
- [ ] 2.1 Implement service layer
- [ ] 2.2 Add API endpoints
  - [ ] 2.2.1 GET endpoint
  - [ ] 2.2.2 POST endpoint

## 3. Testing
- [ ] 3.1 Unit tests
- [ ] 3.2 Integration tests
```

### Best Practices

1. **Hierarchical numbering** - Use 1.1, 1.2, 2.1 format
2. **Checkbox format** - Use `- [ ]` for tracking
3. **Group by theme** - Logical sections (Setup, Core, Testing)
4. **Small tasks** - Each task completable in one session
5. **Check as completed** - Mark `- [x]` when done

## Proposal Format

### Sections

```markdown
# [Change Name] Proposal

## Intent
What we're trying to achieve and why.

## Scope

### In Scope
- Feature A
- Feature B

### Out of Scope
- Future feature X
- Related but separate concern Y

## Approach
High-level approach to solving this.

## Non-Goals (optional)
What this change explicitly doesn't address.
```

## Design Format

### Sections

```markdown
# [Change Name] Design

## Technical Approach
Overview of the solution architecture.

## Architecture Decisions
Key decisions and their rationale.

## Data Flow
How data moves through the system.

## File Changes
Which files will be created/modified.

## Dependencies
External dependencies or prerequisites.

## Risks and Mitigations
Known risks and how to address them.
```

## Configuration Conventions

### Project Config (`openspec/config.yaml`)

```yaml
# Schema selection
schema: spec-driven

# Project-wide context (max 50KB)
context: |
  Tech stack: TypeScript, React, Node.js
  API style: RESTful
  Testing: Jest + React Testing Library

# Per-artifact rules
rules:
  proposal:
    - Keep under 500 words
    - Include non-goals section
  tasks:
    - Break into 2-hour chunks
    - Include success criteria
```

### Change Metadata (`.openspec.yaml`)

```yaml
schema: spec-driven
description: Optional description of the change
```

## Directory Structure Conventions

### Standard Layout

```
project/
├── openspec/
│   ├── config.yaml              # Project config
│   ├── specs/                   # Source of truth
│   │   └── [domain]/
│   │       └── [spec].md
│   └── changes/
│       ├── [active-change]/     # In-progress
│       │   ├── .openspec.yaml
│       │   ├── proposal.md
│       │   ├── specs/
│       │   │   └── [domain]/
│       │   │       └── [spec].md
│       │   ├── design.md
│       │   └── tasks.md
│       └── archive/             # Completed
│           └── YYYY-MM-DD-[change]/
└── .claude/                     # Tool-specific (auto-generated)
    └── skills/
```

### Archive Naming

Archived changes use date prefix:
```
archive/
├── 2024-01-15-add-auth/
├── 2024-01-22-fix-checkout/
└── 2024-02-03-implement-2fa/
```

## Code Style in Specs

### Examples in Specs

Use fenced code blocks with language hints:

```markdown
#### Scenario: Valid Login
- GIVEN a registered user with email "user@example.com"
- WHEN they submit valid credentials:
  ```json
  {
    "email": "user@example.com",
    "password": "valid-password"
  }
  ```
- THEN they receive a 200 response with:
  ```json
  {
    "token": "<jwt-token>",
    "expiresIn": 3600
  }
  ```
```

### Pseudocode

Use pseudocode for algorithm descriptions:

```markdown
## Algorithm

```
1. Validate input parameters
2. Check user permissions
3. If authorized:
   a. Execute operation
   b. Log action
   c. Return success
4. Else:
   a. Log failed attempt
   b. Return 403
```
```

## When to Update vs. Start New

### Update Existing Change When:
- Same intent, refined execution
- Scope narrows (not expands)
- Learning-driven corrections
- Fixing issues found during implementation

### Start New Change When:
- Intent fundamentally changes
- Scope significantly expands
- Original change can be marked complete
- Different area of the codebase
