# Spec-Kit Artifacts

## Artifact Overview

Spec-Kit generates and manages a collection of interconnected artifacts that drive the development process from specification to implementation.

## Complete Directory Structure

```
project/
├── .specify/
│   ├── memory/
│   │   └── constitution.md          # Project principles (v1.0.0)
│   ├── scripts/
│   │   ├── bash/
│   │   │   ├── common.sh
│   │   │   ├── create-new-feature.sh
│   │   │   ├── setup-plan.sh
│   │   │   ├── check-prerequisites.sh
│   │   │   └── update-agent-context.sh
│   │   └── powershell/
│   │       ├── common.ps1
│   │       ├── create-new-feature.ps1
│   │       ├── setup-plan.ps1
│   │       ├── check-prerequisites.ps1
│   │       └── update-agent-context.ps1
│   ├── templates/
│   │   ├── commands/
│   │   │   ├── specify.md
│   │   │   ├── plan.md
│   │   │   ├── tasks.md
│   │   │   ├── implement.md
│   │   │   ├── clarify.md
│   │   │   ├── analyze.md
│   │   │   ├── checklist.md
│   │   │   ├── constitution.md
│   │   │   └── taskstoissues.md
│   │   ├── spec-template.md
│   │   ├── plan-template.md
│   │   ├── tasks-template.md
│   │   ├── checklist-template.md
│   │   └── agent-file-template.md
│   └── specs/
│       ├── 001-user-auth/           # Feature 1
│       │   ├── spec.md
│       │   ├── plan.md
│       │   ├── tasks.md
│       │   ├── research.md
│       │   ├── data-model.md
│       │   ├── quickstart.md
│       │   ├── contracts/
│       │   │   ├── api-spec.json
│       │   │   └── websocket-spec.md
│       │   └── checklists/
│       │       ├── requirements.md
│       │       └── security.md
│       └── 002-notifications/       # Feature 2
│           └── ...
├── .claude/                         # Agent-specific (example)
│   └── commands/
│       ├── speckit.constitution.md
│       ├── speckit.specify.md
│       ├── speckit.clarify.md
│       ├── speckit.plan.md
│       ├── speckit.tasks.md
│       ├── speckit.implement.md
│       ├── speckit.analyze.md
│       └── speckit.checklist.md
└── src/                             # Implementation (generated)
    ├── models/
    ├── services/
    └── api/
```

---

## Artifact Relationships

```
                    ┌─────────────────────┐
                    │    constitution.md   │
                    │    (principles)      │
                    └─────────┬───────────┘
                              │ governs
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       FEATURE ARTIFACTS                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌───────────┐     ┌───────────┐     ┌───────────┐         │
│  │  spec.md  │────▶│  plan.md  │────▶│ tasks.md  │         │
│  │  (what)   │     │   (how)   │     │ (actions) │         │
│  └───────────┘     └─────┬─────┘     └─────┬─────┘         │
│        │                 │                 │                │
│        │                 ▼                 ▼                │
│        │         ┌──────────────┐  ┌──────────────┐        │
│        │         │ research.md  │  │   IMPLEMENT  │        │
│        │         │ data-model.md│  │   (code)     │        │
│        │         │ contracts/   │  └──────────────┘        │
│        │         │ quickstart.md│                           │
│        │         └──────────────┘                           │
│        │                                                    │
│        ▼                                                    │
│  ┌──────────────┐                                           │
│  │ checklists/  │                                           │
│  │ (validation) │                                           │
│  └──────────────┘                                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Core Artifacts

### `constitution.md`

**Created by**: `/speckit.constitution`
**Location**: `.specify/memory/constitution.md`
**Purpose**: Project-wide principles that govern all development

**Content Structure**:
- Core Principles (numbered articles)
- Additional Constraints
- Governance rules
- Version metadata

**Relationships**:
- **Governs**: All feature artifacts
- **Enforced by**: `plan.md` (phase gates), `/speckit.analyze`

---

### `spec.md`

**Created by**: `/speckit.specify`
**Location**: `.specify/specs/###-feature-name/spec.md`
**Purpose**: Functional specification (WHAT and WHY)

**Content Structure**:
- User Scenarios & Testing (prioritized stories)
- Requirements (functional, entities)
- Success Criteria (measurable outcomes)
- Clarifications (if `/speckit.clarify` used)

**Relationships**:
- **Input for**: `plan.md`, `tasks.md`
- **Validated by**: `checklists/requirements.md`
- **Governed by**: `constitution.md`

---

### `plan.md`

**Created by**: `/speckit.plan`
**Location**: `.specify/specs/###-feature-name/plan.md`
**Purpose**: Technical implementation plan (HOW)

**Content Structure**:
- Summary
- Technical Context (stack, dependencies)
- Constitution Check (phase gates)
- Project Structure
- Complexity Tracking (if gates violated)

**Relationships**:
- **Input from**: `spec.md`, `constitution.md`
- **Output to**: `research.md`, `data-model.md`, `contracts/`
- **Input for**: `tasks.md`

---

### `tasks.md`

**Created by**: `/speckit.tasks`
**Location**: `.specify/specs/###-feature-name/tasks.md`
**Purpose**: Actionable task breakdown

**Content Structure**:
- Phase 1: Setup
- Phase 2: Foundational (blocking)
- Phase 3+: User Stories (in priority order)
- Final Phase: Polish
- Dependencies & Execution Order

**Relationships**:
- **Input from**: `plan.md`, `spec.md`, `data-model.md`, `contracts/`
- **Input for**: `/speckit.implement`
- **Validated by**: `/speckit.analyze`

---

## Supporting Artifacts

### `research.md`

**Created by**: `/speckit.plan` (Phase 0)
**Purpose**: Technical research and decisions

**Content Structure**:
```markdown
## [Topic]

**Decision**: [What was chosen]
**Rationale**: [Why chosen]
**Alternatives Considered**: [What else evaluated]
```

---

### `data-model.md`

**Created by**: `/speckit.plan` (Phase 1)
**Purpose**: Entity definitions

**Content Structure**:
```markdown
## Entity: [Name]

**Description**: [What it represents]

### Fields

| Field | Type | Description | Constraints |
|-------|------|-------------|-------------|
| id | UUID | Primary key | Required |
| name | String | Display name | Max 100 chars |

### Relationships

- Has many: [Related Entity]
- Belongs to: [Parent Entity]

### State Transitions

[State diagram if applicable]
```

---

### `contracts/`

**Created by**: `/speckit.plan` (Phase 1)
**Location**: `.specify/specs/###-feature-name/contracts/`
**Purpose**: API and protocol specifications

**Common Files**:
- `api-spec.json` - OpenAPI specification
- `api-spec.yaml` - OpenAPI (YAML format)
- `graphql-schema.graphql` - GraphQL schema
- `websocket-spec.md` - WebSocket event definitions
- `signalr-spec.md` - SignalR hub definitions

---

### `quickstart.md`

**Created by**: `/speckit.plan` (Phase 1)
**Purpose**: Key validation scenarios

**Content Structure**:
```markdown
# Quickstart: [Feature Name]

## Prerequisites

[What must be installed/configured]

## Validation Scenarios

### Scenario 1: [Happy Path]

1. [Step 1]
2. [Step 2]
3. **Expected**: [Outcome]

### Scenario 2: [Edge Case]

1. [Step 1]
2. **Expected**: [Error handling]
```

---

### `checklists/`

**Created by**: `/speckit.specify`, `/speckit.checklist`
**Location**: `.specify/specs/###-feature-name/checklists/`
**Purpose**: Quality validation

**Common Files**:
- `requirements.md` - Spec quality checklist (auto-generated)
- `security.md` - Security review checklist
- `ux.md` - User experience checklist
- `performance.md` - Performance checklist

---

## Agent-Specific Artifacts

### Command Files

**Location**: `.{agent}/commands/` (varies by agent)
**Purpose**: Slash command definitions for AI agents

**Format**: Markdown with YAML frontmatter (or TOML for Gemini/Qwen)

**Files**:
- `speckit.constitution.md`
- `speckit.specify.md`
- `speckit.clarify.md`
- `speckit.plan.md`
- `speckit.tasks.md`
- `speckit.implement.md`
- `speckit.analyze.md`
- `speckit.checklist.md`

### Agent Context Files

**Purpose**: Technology context for AI agent

**Locations by Agent**:
| Agent | Context File |
|-------|--------------|
| Claude | `CLAUDE.md` or `.claude/CLAUDE.md` |
| Gemini | `.gemini/context.md` |
| Copilot | `.github/copilot-instructions.md` |
| Cursor | `.cursor/rules/specify-rules.md` |

**Auto-updated Section**:
```markdown
<!-- SPECIFY_CONTEXT_START -->
## Technology Stack

- **Language**: Python 3.11
- **Framework**: FastAPI
- **Database**: PostgreSQL

## Current Feature

- Branch: 001-user-auth
- Spec: specs/001-user-auth/spec.md
<!-- SPECIFY_CONTEXT_END -->
```

---

## Artifact Lifecycle

### Creation Order

```
1. constitution.md        ← /speckit.constitution
2. spec.md               ← /speckit.specify
   └── checklists/requirements.md
3. (clarifications)      ← /speckit.clarify (optional)
4. plan.md               ← /speckit.plan
   ├── research.md
   ├── data-model.md
   ├── contracts/
   └── quickstart.md
5. tasks.md              ← /speckit.tasks
6. (analysis report)     ← /speckit.analyze (optional)
7. (implementation)      ← /speckit.implement
```

### Status Updates

`spec.md` tracks status:
```markdown
**Status**: Draft | Review | Approved | Implemented
```

`tasks.md` tracks completion:
```markdown
- [ ] T001 Pending task
- [X] T002 Completed task
```

---

## Branch Naming Convention

```
###-short-description

Examples:
001-user-auth
002-notifications
003-payment-integration
```

- **###** - Zero-padded feature number
- **short-description** - 2-4 word slug from feature description

---

## Git Integration

### Feature Branches

Each feature creates a dedicated branch:
```
main
├── 001-user-auth
├── 002-notifications
└── 003-payment-integration
```

### Recommended .gitignore

```gitignore
# Agent credentials (varies by agent)
.claude/settings.json
.gemini/credentials/

# Generated files (optional)
.specify/specs/*/research.md

# Local environment
.env
.env.local
```
