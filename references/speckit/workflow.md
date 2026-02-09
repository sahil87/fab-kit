# Spec-Kit Workflow

## The 6-Step Development Process

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ 1. CONSTITUTION │ → │ 2. SPECIFY      │ → │ 3. CLARIFY      │
│   (principles)  │    │   (what/why)    │    │   (refinement)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                      │
        ┌─────────────────────────────────────────────┘
        ↓
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ 4. PLAN         │ → │ 5. TASKS        │ → │ 6. IMPLEMENT    │
│   (how/tech)    │    │   (breakdown)   │    │   (execution)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Step 1: Constitution (`/speckit.constitution`)

**Purpose**: Establish project principles and governance rules.

**Input**: Natural language description of project principles
**Output**: `/memory/constitution.md`

```markdown
/speckit.constitution Create principles focused on code quality, testing standards,
user experience consistency, and performance requirements.
```

The constitution defines **non-negotiable rules** that govern all subsequent development:
- Coding standards
- Testing requirements
- Architectural constraints
- Quality gates

## Step 2: Specify (`/speckit.specify`)

**Purpose**: Create the functional specification from a feature description.

**Input**: Natural language feature description (WHAT and WHY, not HOW)
**Output**:
- New git branch (`###-feature-name`)
- Spec file (`specs/###-feature-name/spec.md`)
- Quality checklist (`specs/###-feature-name/checklists/requirements.md`)

```markdown
/speckit.specify Build an application that can help me organize my photos in
separate photo albums. Albums are grouped by date and can be re-organized by
dragging and dropping on the main page.
```

### Execution Flow

1. **Generate branch name** from description (2-4 words)
2. **Check existing branches** (local, remote, specs directories)
3. **Run script** to create branch and directory structure
4. **Load spec template** and fill with requirements
5. **Generate quality checklist** for validation
6. **Report** branch name and spec file path

### Key Guidelines

- Focus on **WHAT** users need and **WHY**
- Avoid **HOW** to implement (no tech stack, APIs, code structure)
- Mark ambiguities with `[NEEDS CLARIFICATION: specific question]`
- **Maximum 3** clarification markers (prioritize by impact)
- Success criteria must be **measurable and technology-agnostic**

## Step 3: Clarify (`/speckit.clarify`) - Optional but Recommended

**Purpose**: Identify and resolve ambiguities in the specification.

**Input**: Existing spec file
**Output**: Updated spec with clarifications recorded

```markdown
/speckit.clarify Focus on security and performance requirements.
```

### Execution Flow

1. **Load spec** and perform structured ambiguity scan
2. **Categorize** findings (Functional, Domain, UX, Non-Functional, etc.)
3. **Generate questions** (max 5, one at a time)
4. **Present recommendations** with reasoning
5. **Update spec** after each accepted answer
6. **Report** coverage summary and sections touched

### Ambiguity Categories

| Category | Focus Areas |
|----------|-------------|
| Functional Scope | Core goals, success criteria, out-of-scope |
| Domain & Data | Entities, relationships, state transitions |
| Interaction & UX | User journeys, error states, accessibility |
| Non-Functional | Performance, scalability, reliability, security |
| Integration | External services, APIs, failure modes |
| Edge Cases | Negative scenarios, rate limiting, conflicts |

## Step 4: Plan (`/speckit.plan`)

**Purpose**: Create technical implementation plan with architecture decisions.

**Input**: Tech stack requirements and architecture choices
**Output**:
- `plan.md` - Implementation plan
- `research.md` - Technical research findings
- `data-model.md` - Entity definitions
- `contracts/` - API contracts (OpenAPI/GraphQL)
- `quickstart.md` - Validation scenarios
- Updated agent context file

```markdown
/speckit.plan The application uses Vite with minimal libraries. Use vanilla HTML,
CSS, and JavaScript. Metadata stored in local SQLite database.
```

### Execution Phases

#### Phase 0: Research

1. **Extract unknowns** from Technical Context
2. **Dispatch research agents** for each unknown
3. **Consolidate findings** in `research.md`

#### Phase 1: Design & Contracts

1. **Extract entities** from feature spec → `data-model.md`
2. **Generate API contracts** from functional requirements
3. **Update agent context** with new technology

### Constitution Check

The plan must pass constitutional gates before proceeding:

```markdown
## Constitution Check

### Simplicity Gate (Article VII)
- [ ] Using ≤3 projects?
- [ ] No future-proofing?

### Anti-Abstraction Gate (Article VIII)
- [ ] Using framework directly?
- [ ] Single model representation?
```

## Step 5: Tasks (`/speckit.tasks`)

**Purpose**: Generate actionable, dependency-ordered task list.

**Input**: Design documents (plan.md, spec.md, data-model.md, contracts/)
**Output**: `tasks.md` with executable task breakdown

```markdown
/speckit.tasks
```

### Task Organization

Tasks are organized by **user story** to enable independent implementation:

```markdown
## Phase 1: Setup (Shared Infrastructure)
- [ ] T001 Create project structure per implementation plan
- [ ] T002 Initialize project with dependencies

## Phase 2: Foundational (Blocking Prerequisites)
- [ ] T003 Setup database schema
- [ ] T004 [P] Implement authentication framework

## Phase 3: User Story 1 - [Title] (Priority: P1)
- [ ] T005 [P] [US1] Create User model in src/models/user.py
- [ ] T006 [US1] Implement UserService in src/services/user_service.py
```

### Task Format

```
- [ ] [TaskID] [P?] [Story?] Description with file path
```

- **[P]** = Parallelizable (different files, no dependencies)
- **[USn]** = User story label (required for story phases)

### Implementation Strategy

**MVP First**:
1. Complete Setup + Foundational
2. Complete User Story 1
3. **STOP and VALIDATE**
4. Deploy/demo if ready

**Incremental Delivery**:
- Each story adds value without breaking previous stories

## Step 6: Implement (`/speckit.implement`)

**Purpose**: Execute the implementation plan by processing all tasks.

**Input**: Complete `tasks.md` and design documents
**Output**: Working implementation

```markdown
/speckit.implement
```

### Execution Flow

1. **Check checklist status** (if exists, warn on incomplete)
2. **Load implementation context** (tasks.md, plan.md, data-model.md, contracts/)
3. **Project setup verification** (create ignore files, configure tooling)
4. **Parse task structure** (phases, dependencies, parallel markers)
5. **Execute tasks** (phase-by-phase, respecting dependencies)
6. **Progress tracking** (mark completed tasks with `[X]`)
7. **Completion validation** (verify features match spec)

### Ignore File Generation

The implement command auto-generates appropriate ignore files:

| Technology | Patterns |
|------------|----------|
| Node.js | `node_modules/`, `dist/`, `build/`, `*.log`, `.env*` |
| Python | `__pycache__/`, `*.pyc`, `.venv/`, `venv/` |
| Go | `*.exe`, `*.test`, `vendor/` |
| Rust | `target/`, `debug/`, `release/` |
| .NET | `bin/`, `obj/`, `*.user`, `packages/` |

## Optional Enhancement Commands

### `/speckit.analyze`

**Purpose**: Cross-artifact consistency and quality analysis.

**When**: After `/speckit.tasks`, before `/speckit.implement`

```markdown
/speckit.analyze
```

Detects:
- Duplications
- Ambiguities
- Underspecification
- Constitution alignment issues
- Coverage gaps
- Inconsistencies

### `/speckit.checklist`

**Purpose**: Generate custom quality checklists.

**When**: After `/speckit.plan` for domain-specific validation

```markdown
/speckit.checklist Create a security review checklist for the authentication flow.
```

## Branch and Feature Detection

Spec-Kit automatically detects the active feature based on:

1. **Git branch name** (e.g., `001-photo-albums`)
2. **SPECIFY_FEATURE** environment variable (for non-git repos)

Scripts handle feature number generation:
- Check remote branches
- Check local branches
- Check specs directories
- Use maximum + 1 for new feature
