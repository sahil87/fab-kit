# OpenSpec Artifact System

## What are Artifacts?

Artifacts are structured documents created during a change workflow. Each artifact has:
- **ID** - Unique identifier (e.g., `proposal`, `design`)
- **Generated file** - Output filename (e.g., `proposal.md`)
- **Template** - Structure guide
- **Dependencies** - What must exist before creation
- **Description** - What this artifact represents

## Default Artifacts (spec-driven schema)

| Artifact | File | Dependencies | Purpose |
|----------|------|--------------|---------|
| `proposal` | `proposal.md` | None | Intent, scope, approach |
| `specs` | `specs/*.md` | proposal | What's changing |
| `design` | `design.md` | proposal | How to implement |
| `tasks` | `tasks.md` | specs, design | Implementation steps |

## Artifact Workflow

```
                    ┌──────────┐
                    │ proposal │ (root - no dependencies)
                    └────┬─────┘
              ┌─────────┴─────────┐
              v                   v
         ┌────────┐          ┌────────┐
         │  specs │          │ design │
         └────┬───┘          └────┬───┘
              └─────────┬─────────┘
                        v
                   ┌────────┐
                   │ tasks  │
                   └────┬───┘
                        v
                   ┌────────┐
                   │ apply  │ (implementation phase)
                   └────────┘
```

### Parallel Work

Note that `specs` and `design` can be created in parallel - they both only require `proposal`. This enables:
- Technical investigation while requirements are being refined
- Domain experts working on specs while architects work on design

## Artifact Details

### Proposal

**Purpose:** Capture intent, scope, and approach before diving into details.

**Sections:**
```markdown
# [Change Name] Proposal

## Intent
What we're trying to achieve and why this matters.

## Scope

### In Scope
- Feature A
- Modification B

### Out of Scope
- Future feature X
- Related but separate Y

## Approach
High-level approach to solving this.
```

**When to Update:**
- Scope changes
- Intent clarifies
- Approach shifts fundamentally

### Specs (Delta Specs)

**Purpose:** What's changing relative to current system behavior.

**Format:**
```markdown
# [Domain] Specification Changes

## ADDED Requirements

### Requirement: New Feature
The system SHALL [new behavior].

#### Scenario: Happy Path
- GIVEN [context]
- WHEN [action]
- THEN [outcome]

## MODIFIED Requirements

### Requirement: Existing Feature
The system SHALL [updated behavior].

## REMOVED Requirements

### Requirement: Deprecated Feature
<!-- Being removed -->
```

**Key Concepts:**
- ADDED → Appended to main spec on archive
- MODIFIED → Replaces existing requirement
- REMOVED → Deleted from main spec

### Design

**Purpose:** Technical approach and architecture decisions.

**Sections:**
```markdown
# [Change Name] Design

## Technical Approach
Overview of the solution architecture.

## Architecture Decisions
Key decisions and rationale.

## Data Flow
How data moves through the system.

## File Changes
Which files will be created/modified.

## Dependencies
External dependencies or prerequisites.

## Risks and Mitigations
Known risks and how to address them.
```

**When to Update:**
- Implementation reveals approach won't work
- Better solution discovered
- New constraints identified

### Tasks

**Purpose:** Implementation checklist for AI and human executors.

**Format:**
```markdown
# Tasks

## 1. Setup
- [ ] 1.1 Create database migration
- [ ] 1.2 Add configuration entries

## 2. Core Implementation
- [ ] 2.1 Implement service layer
- [ ] 2.2 Add API endpoints
  - [ ] 2.2.1 GET endpoint
  - [ ] 2.2.2 POST endpoint

## 3. Testing
- [ ] 3.1 Unit tests
- [ ] 3.2 Integration tests
```

**Best Practices:**
- Hierarchical numbering (1.1, 1.2, 2.1)
- Checkbox format (`- [ ]` / `- [x]`)
- Group by theme
- Keep tasks small (completable in one session)
- Check off as completed

## Artifact Graph

The artifact graph is the core data structure tracking workflow state.

### Types

```typescript
interface Artifact {
  id: string;              // e.g., 'proposal'
  generates: string;       // e.g., 'proposal.md'
  description: string;     // Human-readable purpose
  template: string;        // Path to template file
  instruction?: string;    // Optional schema-level guidance
  requires: string[];      // Dependency artifact IDs
}

interface Schema {
  artifacts: Artifact[];
  apply?: {
    requires: string[];    // Artifacts needed for apply
    tracks: string;        // File tracking progress
  };
}
```

### State Detection

Artifact state is determined by filesystem existence:

```typescript
function getArtifactState(artifact: Artifact, changePath: string): ArtifactState {
  const filePath = path.join(changePath, artifact.generates);

  if (fs.existsSync(filePath)) {
    return 'complete';
  }

  const depsComplete = artifact.requires.every(dep =>
    getArtifactState(getArtifact(dep), changePath) === 'complete'
  );

  return depsComplete ? 'ready' : 'blocked';
}
```

States:
- **blocked** - Dependencies not met
- **ready** - Can be created
- **complete** - File exists

### Dependency Resolution

```typescript
function getReadyArtifacts(schema: Schema, changePath: string): Artifact[] {
  return schema.artifacts.filter(artifact => {
    // Not already complete
    if (getArtifactState(artifact, changePath) === 'complete') {
      return false;
    }

    // All dependencies complete
    return artifact.requires.every(dep =>
      getArtifactState(getArtifact(dep), changePath) === 'complete'
    );
  });
}
```

## Instruction Generation

When creating an artifact, the system generates enriched instructions:

```typescript
interface ArtifactInstructions {
  changeName: string;
  artifactId: string;
  schemaName: string;
  description: string;
  instruction: string | undefined;   // Schema instruction
  context: string | undefined;       // Project context from config
  rules: string[] | undefined;       // Artifact-specific rules
  template: string;                  // Template structure
  dependencies: DependencyInfo[];    // What was needed
  unlocks: string[];                 // What this enables
}
```

### Three-Layer Composition

1. **Context** (from `config.yaml`) - Project-wide information
2. **Rules** (from `config.yaml`) - Artifact-specific guidance
3. **Template** (from schema) - Structure and format

## Apply Phase

After all planning artifacts are complete, the `apply` phase executes tasks.

### Apply Configuration

```yaml
# In schema.yaml
apply:
  requires: [tasks]           # What must be complete
  tracks: tasks.md           # File tracking progress
```

### Task Progress Tracking

The system parses `tasks.md` to track completion:

```typescript
interface TaskProgress {
  total: number;
  completed: number;
  percentage: number;
  remaining: Task[];
}

function parseTaskProgress(tasksContent: string): TaskProgress {
  const tasks = parseCheckboxes(tasksContent);
  const completed = tasks.filter(t => t.checked);

  return {
    total: tasks.length,
    completed: completed.length,
    percentage: (completed.length / tasks.length) * 100,
    remaining: tasks.filter(t => !t.checked),
  };
}
```

## Archive Process

When a change is complete, archiving:

1. **Validates** - Ensures specs and change are valid
2. **Merges deltas** - Applies ADDED/MODIFIED/REMOVED to main specs
3. **Moves to archive** - Preserves full context

### Delta Merge Algorithm

```typescript
function mergeDeltas(mainSpec: string, deltaSpec: DeltaSpec): string {
  let result = mainSpec;

  // Process ADDED - append to requirements section
  for (const added of deltaSpec.added) {
    result = appendRequirement(result, added);
  }

  // Process MODIFIED - find and replace
  for (const modified of deltaSpec.modified) {
    result = replaceRequirement(result, modified);
  }

  // Process REMOVED - delete
  for (const removed of deltaSpec.removed) {
    result = removeRequirement(result, removed);
  }

  return result;
}
```

### Archive Directory Structure

```
openspec/changes/archive/
├── 2024-01-15-add-auth/
│   ├── .openspec.yaml
│   ├── proposal.md
│   ├── specs/
│   │   └── auth/
│   │       └── login.md
│   ├── design.md
│   └── tasks.md
└── 2024-01-22-fix-checkout/
    └── ...
```

## Custom Artifacts

Schemas can define custom artifacts:

```yaml
# custom-schema/schema.yaml
artifacts:
  - id: research
    generates: research.md
    description: Technical research and spikes
    template: templates/research.md
    requires: []

  - id: proposal
    generates: proposal.md
    description: Change proposal
    template: templates/proposal.md
    requires: [research]  # Research comes first

  - id: specs
    generates: specs/*.md
    description: Delta specifications
    template: templates/spec.md
    requires: [proposal]
```

This allows workflows like:
- Research → Proposal → Specs → Tasks
- RFC → Specs → Implementation
- Spike → Design → Tasks

## Status Command

The `openspec status` command shows artifact state:

```
$ openspec status --change add-auth

Change: add-auth
Schema: spec-driven

Artifacts:
  ✓ proposal     complete
  ✓ specs        complete
  ◯ design       ready (can create now)
  ○ tasks        blocked (needs: design)

Next: Create design.md
```

## Instructions Command

The `openspec instructions` command outputs enriched guidance:

```
$ openspec instructions design --change add-auth

# Design Instructions for add-auth

## Context
Tech stack: TypeScript, React, Node.js
API style: RESTful

## Rules
- Include data flow diagrams
- Document all API changes

## Template
[Design template content...]

## Dependencies
- proposal (complete)

## Unlocks
- tasks
```
