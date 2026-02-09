# Spec-Kit Commands

## Command Overview

| Command | Purpose | Prerequisites | Output |
|---------|---------|---------------|--------|
| `/speckit.constitution` | Establish project principles | None | `memory/constitution.md` |
| `/speckit.specify` | Create feature specification | Constitution (recommended) | Branch, `spec.md`, checklist |
| `/speckit.clarify` | Resolve spec ambiguities | `spec.md` | Updated `spec.md` |
| `/speckit.plan` | Create technical plan | `spec.md` | `plan.md`, `research.md`, `data-model.md`, `contracts/` |
| `/speckit.tasks` | Generate task breakdown | `plan.md`, `spec.md` | `tasks.md` |
| `/speckit.implement` | Execute implementation | `tasks.md` | Working code |
| `/speckit.analyze` | Consistency analysis | `tasks.md` | Analysis report |
| `/speckit.checklist` | Generate quality checklists | `spec.md` or `plan.md` | `checklists/*.md` |

## Command Anatomy

All commands use YAML frontmatter for metadata:

```yaml
---
description: "What the command does"
handoffs:
  - label: Next Step
    agent: speckit.next-command
    prompt: Suggested prompt
    send: true  # Auto-send on handoff
scripts:
  sh: scripts/bash/script-name.sh --json
  ps: scripts/powershell/script-name.ps1 -Json
agent_scripts:
  sh: scripts/bash/update-agent-context.sh __AGENT__
  ps: scripts/powershell/update-agent-context.ps1 -AgentType __AGENT__
---
```

### Placeholders

| Placeholder | Replaced With |
|-------------|---------------|
| `$ARGUMENTS` | User input after command |
| `{SCRIPT}` | Appropriate script path (bash/powershell) |
| `{AGENT_SCRIPT}` | Agent context update script |
| `{ARGS}` | User arguments (JSON-safe) |
| `__AGENT__` | Current AI agent name |

---

## Core Commands

### `/speckit.constitution`

**Purpose**: Create or update project governing principles.

**Frontmatter**:
```yaml
description: Create or update the project constitution from interactive or provided principle inputs
handoffs:
  - label: Build Specification
    agent: speckit.specify
    prompt: Implement the feature specification based on the updated constitution. I want to build...
```

**Execution Flow**:

1. Load existing constitution template at `/memory/constitution.md`
2. Identify placeholder tokens (`[ALL_CAPS_IDENTIFIER]`)
3. Collect/derive values from user input or repo context
4. Draft updated constitution content
5. Consistency propagation (update dependent templates)
6. Produce Sync Impact Report
7. Validate and write constitution

**Key Rules**:
- No unexplained bracket tokens remain
- Version follows semantic versioning (MAJOR.MINOR.PATCH)
- Dates in ISO format (YYYY-MM-DD)
- Principles must be declarative and testable

---

### `/speckit.specify`

**Purpose**: Create feature specification from natural language description.

**Frontmatter**:
```yaml
description: Create or update the feature specification from a natural language feature description.
handoffs:
  - label: Build Technical Plan
    agent: speckit.plan
    prompt: Create a plan for the spec. I am building with...
  - label: Clarify Spec Requirements
    agent: speckit.clarify
    prompt: Clarify specification requirements
    send: true
scripts:
  sh: scripts/bash/create-new-feature.sh --json "{ARGS}"
  ps: scripts/powershell/create-new-feature.ps1 -Json "{ARGS}"
```

**Execution Flow**:

1. **Generate branch name** (2-4 words, action-noun format)
2. **Check existing branches** (git fetch, check remote/local/specs)
3. **Run script** with calculated number and short-name
4. **Load spec template** (`templates/spec-template.md`)
5. **Fill specification** following guidelines:
   - Parse user description
   - Extract key concepts (actors, actions, data, constraints)
   - Mark ambiguities (max 3 `[NEEDS CLARIFICATION]` markers)
   - Fill User Scenarios & Testing
   - Generate Functional Requirements
   - Define Success Criteria
6. **Quality Validation** - Generate and run checklist
7. **Report** completion with branch name and spec path

**Branch Name Generation**:
```
"Add user authentication" → "user-auth"
"Implement OAuth2 integration for API" → "oauth2-api-integration"
"Create a dashboard for analytics" → "analytics-dashboard"
```

**Clarification Priority**: scope > security/privacy > user experience > technical details

---

### `/speckit.clarify`

**Purpose**: Identify underspecified areas and encode answers back into spec.

**Frontmatter**:
```yaml
description: Identify underspecified areas in the current feature spec by asking up to 5 highly targeted clarification questions
handoffs:
  - label: Build Technical Plan
    agent: speckit.plan
    prompt: Create a plan for the spec. I am building with...
scripts:
  sh: scripts/bash/check-prerequisites.sh --json --paths-only
  ps: scripts/powershell/check-prerequisites.ps1 -Json -PathsOnly
```

**Ambiguity Taxonomy**:

| Category | Focus Areas |
|----------|-------------|
| Functional Scope & Behavior | Core user goals, out-of-scope, user roles |
| Domain & Data Model | Entities, identity rules, state transitions |
| Interaction & UX Flow | Critical journeys, error/empty/loading states |
| Non-Functional Quality | Performance, scalability, reliability, security |
| Integration & Dependencies | External services, data formats, protocols |
| Edge Cases & Failure Handling | Negative scenarios, rate limiting, conflicts |
| Constraints & Tradeoffs | Technical constraints, rejected alternatives |
| Terminology & Consistency | Canonical glossary, avoided synonyms |
| Completion Signals | Acceptance criteria testability |

**Question Format**:

For multiple-choice:
```markdown
**Recommended:** Option [X] - <reasoning>

| Option | Description |
|--------|-------------|
| A | Option A description |
| B | Option B description |
| Short | Provide your own short answer |

You can reply with the option letter, accept the recommendation by saying "yes", or provide your own short answer.
```

**Rules**:
- Maximum 5 questions total
- One question at a time
- Update spec after each accepted answer
- Stop if user signals "done", "good", "no more"

---

### `/speckit.plan`

**Purpose**: Execute implementation planning workflow.

**Frontmatter**:
```yaml
description: Execute the implementation planning workflow using the plan template
handoffs:
  - label: Create Tasks
    agent: speckit.tasks
    prompt: Break the plan into tasks
    send: true
  - label: Create Checklist
    agent: speckit.checklist
    prompt: Create a checklist for the following domain...
scripts:
  sh: scripts/bash/setup-plan.sh --json
  ps: scripts/powershell/setup-plan.ps1 -Json
agent_scripts:
  sh: scripts/bash/update-agent-context.sh __AGENT__
  ps: scripts/powershell/update-agent-context.ps1 -AgentType __AGENT__
```

**Execution Phases**:

**Phase 0: Research**
1. Extract unknowns from Technical Context
2. Generate and dispatch research agents
3. Consolidate findings in `research.md`

**Phase 1: Design & Contracts**
1. Extract entities → `data-model.md`
2. Generate API contracts → `contracts/`
3. Update agent context with new technology

**Outputs**:
- `plan.md` - Implementation plan
- `research.md` - Research findings
- `data-model.md` - Entity definitions
- `contracts/` - API specifications
- `quickstart.md` - Validation scenarios

---

### `/speckit.tasks`

**Purpose**: Generate actionable, dependency-ordered task list.

**Frontmatter**:
```yaml
description: Generate an actionable, dependency-ordered tasks.md
handoffs:
  - label: Analyze For Consistency
    agent: speckit.analyze
    prompt: Run a project analysis for consistency
    send: true
  - label: Implement Project
    agent: speckit.implement
    prompt: Start the implementation in phases
    send: true
scripts:
  sh: scripts/bash/check-prerequisites.sh --json
  ps: scripts/powershell/check-prerequisites.ps1 -Json
```

**Task Format**:
```
- [ ] [TaskID] [P?] [Story?] Description with file path
```

**Task Sources**:
| Source | Tasks Generated |
|--------|-----------------|
| `spec.md` | User story phases (P1, P2, P3...) |
| `plan.md` | Tech stack, libraries, structure |
| `data-model.md` | Entity model tasks |
| `contracts/` | API endpoint tasks |
| `research.md` | Setup tasks from decisions |

**Phase Structure**:
1. **Phase 1**: Setup (project initialization)
2. **Phase 2**: Foundational (blocking prerequisites)
3. **Phase 3+**: User Stories (in priority order)
4. **Final Phase**: Polish & Cross-Cutting

---

### `/speckit.implement`

**Purpose**: Execute all tasks defined in tasks.md.

**Frontmatter**:
```yaml
description: Execute the implementation plan by processing and executing all tasks
scripts:
  sh: scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks
  ps: scripts/powershell/check-prerequisites.ps1 -Json -RequireTasks -IncludeTasks
```

**Execution Flow**:

1. **Check checklists** - Warn if any incomplete
2. **Load context** - tasks.md, plan.md, data-model.md, contracts/
3. **Project setup** - Generate ignore files for detected technologies
4. **Parse tasks** - Extract phases, dependencies, parallel markers
5. **Execute tasks** - Phase-by-phase, respecting dependencies
6. **Track progress** - Mark completed tasks with `[X]`
7. **Validate** - Verify features match specification

**Ignore Files Generated**:

| Detection | File Created |
|-----------|--------------|
| Git repo | `.gitignore` |
| Dockerfile | `.dockerignore` |
| .eslintrc* | `.eslintignore` |
| .prettierrc* | `.prettierignore` |
| terraform files | `.terraformignore` |

---

## Enhancement Commands

### `/speckit.analyze`

**Purpose**: Non-destructive cross-artifact consistency analysis.

**Key Constraint**: **STRICTLY READ-ONLY** - never modifies files.

**Detection Passes**:
- Duplication Detection
- Ambiguity Detection
- Underspecification
- Constitution Alignment
- Coverage Gaps
- Inconsistency

**Severity Levels**:
| Level | Criteria |
|-------|----------|
| CRITICAL | Constitution violation, missing core artifact, blocking functionality |
| HIGH | Duplicate/conflicting requirement, ambiguous security/performance |
| MEDIUM | Terminology drift, missing non-functional coverage |
| LOW | Style improvements, minor redundancy |

---

### `/speckit.checklist`

**Purpose**: Generate custom quality checklists.

**Checklist Format**:
```markdown
# [CHECKLIST TYPE] Checklist: [FEATURE NAME]

**Purpose**: [What this checklist covers]
**Created**: [DATE]
**Feature**: [Link to spec.md]

## [Category 1]

- [ ] CHK001 First checklist item
- [ ] CHK002 Second checklist item

## Notes

- Check items off as completed: `[x]`
```

---

### `/speckit.taskstoissues`

**Purpose**: Convert tasks.md to GitHub issues (optional command).

Used for teams that want to track implementation in GitHub Issues.
