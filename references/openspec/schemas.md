# OpenSpec Schema System

## What are Schemas?

Schemas define the workflow structure for changes. They specify:
- Which artifacts exist
- Dependencies between artifacts
- Templates for each artifact
- Apply phase configuration

## Built-in Schema: spec-driven

Location: `schemas/spec-driven/`

```yaml
# schema.yaml
artifacts:
  - id: proposal
    generates: proposal.md
    description: Change proposal with intent, scope, and approach
    template: templates/proposal.md
    requires: []

  - id: specs
    generates: specs/*.md
    description: Delta specifications showing what's changing
    template: templates/spec.md
    requires: [proposal]

  - id: design
    generates: design.md
    description: Technical design and architecture decisions
    template: templates/design.md
    requires: [proposal]

  - id: tasks
    generates: tasks.md
    description: Implementation task checklist
    template: templates/tasks.md
    requires: [specs, design]

apply:
  requires: [tasks]
  tracks: tasks.md
```

### Directory Structure

```
schemas/spec-driven/
├── schema.yaml
└── templates/
    ├── proposal.md
    ├── spec.md
    ├── design.md
    └── tasks.md
```

## Schema Definition

### Artifact Properties

```yaml
artifacts:
  - id: string           # Unique identifier
    generates: string    # Output file path (supports globs like specs/*.md)
    description: string  # Human-readable description
    template: string     # Path to template file (relative to schema dir)
    instruction: string  # Optional schema-level guidance (optional)
    requires: string[]   # Dependency artifact IDs
```

### Apply Phase

```yaml
apply:
  requires: string[]    # Artifacts that must be complete
  tracks: string        # File used to track progress (checkbox parsing)
```

### Full Example

```yaml
# schema.yaml
artifacts:
  - id: proposal
    generates: proposal.md
    description: "Intent, scope, and high-level approach"
    template: templates/proposal.md
    instruction: |
      Keep proposals focused and concise.
      Always include non-goals to clarify boundaries.
    requires: []

  - id: specs
    generates: specs/*.md
    description: "Delta specifications for this change"
    template: templates/spec.md
    instruction: |
      Use RFC 2119 keywords consistently.
      Include at least one scenario per requirement.
    requires: [proposal]

  - id: design
    generates: design.md
    description: "Technical approach and architecture"
    template: templates/design.md
    requires: [proposal]

  - id: tasks
    generates: tasks.md
    description: "Implementation checklist"
    template: templates/tasks.md
    instruction: |
      Break tasks into chunks completable in one session.
      Use hierarchical numbering (1.1, 1.2, etc.)
    requires: [specs, design]

apply:
  requires: [tasks]
  tracks: tasks.md
```

## Templates

Templates provide structure guidance for each artifact.

### Proposal Template

```markdown
# [Change Name] Proposal

## Intent

[What we're trying to achieve and why]

## Scope

### In Scope

- [Feature/change 1]
- [Feature/change 2]

### Out of Scope

- [Explicitly excluded item 1]
- [Explicitly excluded item 2]

## Approach

[High-level approach to solving this]
```

### Spec Template

```markdown
# [Domain] Specification Changes

## ADDED Requirements

### Requirement: [Name]

The system SHALL [behavior].

#### Scenario: [Name]
- GIVEN [context]
- WHEN [action]
- THEN [outcome]

## MODIFIED Requirements

### Requirement: [Name]

The system SHALL [updated behavior].

## REMOVED Requirements

### Requirement: [Name]

[This requirement is being removed]
```

### Design Template

```markdown
# [Change Name] Design

## Technical Approach

[Overview of the solution]

## Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| [Decision 1] | [Why] |

## Data Flow

[How data moves through the system]

## File Changes

| File | Change |
|------|--------|
| [path] | [create/modify/delete] |

## Dependencies

[External dependencies or prerequisites]

## Risks

| Risk | Mitigation |
|------|------------|
| [Risk 1] | [How to address] |
```

### Tasks Template

```markdown
# Tasks

## 1. Setup

- [ ] 1.1 [First setup task]
- [ ] 1.2 [Second setup task]

## 2. Implementation

- [ ] 2.1 [Core implementation task]
- [ ] 2.2 [Additional implementation]

## 3. Testing

- [ ] 3.1 [Unit tests]
- [ ] 3.2 [Integration tests]

## 4. Documentation

- [ ] 4.1 [Update relevant docs]
```

## Custom Schemas

### Creating a Custom Schema

1. Create schema directory:
```bash
mkdir -p openspec/schemas/my-workflow
```

2. Create `schema.yaml`:
```yaml
# openspec/schemas/my-workflow/schema.yaml
artifacts:
  - id: research
    generates: research.md
    description: Technical research and spikes
    template: templates/research.md
    requires: []

  - id: rfc
    generates: rfc.md
    description: Request for comments
    template: templates/rfc.md
    requires: [research]

  - id: implementation
    generates: implementation.md
    description: Implementation plan
    template: templates/implementation.md
    requires: [rfc]

apply:
  requires: [implementation]
  tracks: implementation.md
```

3. Create templates:
```bash
mkdir -p openspec/schemas/my-workflow/templates
```

4. Reference in config:
```yaml
# openspec/config.yaml
schema: my-workflow
```

### Schema Locations

Schemas are resolved in order:

1. **CLI flag:** `--schema custom-name`
2. **Change metadata:** `.openspec.yaml` in change directory
3. **Project config:** `openspec/config.yaml`
4. **Default:** `spec-driven`

### Schema Search Paths

1. `openspec/schemas/[name]/` - Project-local schemas
2. `schemas/[name]/` - Built-in schemas (in OpenSpec package)

## Schema Resolution

```typescript
function resolveSchema(options: {
  cliSchema?: string;
  changeMetadata?: ChangeMetadata;
  projectConfig?: ProjectConfig;
}): string {
  // 1. CLI flag takes precedence
  if (options.cliSchema) {
    return options.cliSchema;
  }

  // 2. Change metadata
  if (options.changeMetadata?.schema) {
    return options.changeMetadata.schema;
  }

  // 3. Project config
  if (options.projectConfig?.schema) {
    return options.projectConfig.schema;
  }

  // 4. Default
  return 'spec-driven';
}
```

## Schema Validation

Schemas are validated with Zod:

```typescript
const ArtifactSchema = z.object({
  id: z.string(),
  generates: z.string(),
  description: z.string(),
  template: z.string(),
  instruction: z.string().optional(),
  requires: z.array(z.string()),
});

const WorkflowSchema = z.object({
  artifacts: z.array(ArtifactSchema),
  apply: z.object({
    requires: z.array(z.string()),
    tracks: z.string(),
  }).optional(),
});
```

Validation ensures:
- All artifact IDs are unique
- All `requires` references exist
- Apply phase references valid artifacts
- Templates exist at specified paths

## Example Schemas

### Research-First Workflow

```yaml
artifacts:
  - id: research
    generates: research.md
    description: Technical investigation
    template: templates/research.md
    requires: []

  - id: proposal
    generates: proposal.md
    description: Change proposal based on research
    template: templates/proposal.md
    requires: [research]

  - id: specs
    generates: specs/*.md
    description: Specifications
    template: templates/spec.md
    requires: [proposal]

  - id: tasks
    generates: tasks.md
    description: Tasks
    template: templates/tasks.md
    requires: [specs]

apply:
  requires: [tasks]
  tracks: tasks.md
```

### Minimal Workflow

```yaml
artifacts:
  - id: spec
    generates: spec.md
    description: Simple specification
    template: templates/spec.md
    requires: []

  - id: tasks
    generates: tasks.md
    description: Implementation tasks
    template: templates/tasks.md
    requires: [spec]

apply:
  requires: [tasks]
  tracks: tasks.md
```

### RFC-Style Workflow

```yaml
artifacts:
  - id: rfc
    generates: rfc.md
    description: Request for comments
    template: templates/rfc.md
    requires: []

  - id: decision
    generates: decision.md
    description: Decision record
    template: templates/decision.md
    requires: [rfc]

  - id: implementation
    generates: implementation.md
    description: Implementation plan
    template: templates/implementation.md
    requires: [decision]

apply:
  requires: [implementation]
  tracks: implementation.md
```

## Listing Schemas

```bash
$ openspec schemas

Available schemas:

  spec-driven (default)
    proposal → specs → design → tasks

  my-workflow (project)
    research → rfc → implementation
```

## Schema Introspection

```bash
$ openspec schema show spec-driven

Schema: spec-driven

Artifacts:
  proposal
    generates: proposal.md
    requires: (none)

  specs
    generates: specs/*.md
    requires: proposal

  design
    generates: design.md
    requires: proposal

  tasks
    generates: tasks.md
    requires: specs, design

Apply:
  requires: tasks
  tracks: tasks.md
```
