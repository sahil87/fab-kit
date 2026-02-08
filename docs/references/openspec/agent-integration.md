# OpenSpec Agent Integration

## Overview

OpenSpec provides a sophisticated multi-tool agent integration system supporting 22+ AI coding assistants. The architecture separates tool-agnostic command definitions from tool-specific adapters.

## Supported AI Tools

| Tool | Config Directory | Skills Directory |
|------|-----------------|------------------|
| Claude Code | `.claude/` | `.claude/skills/` |
| Cursor | `.cursor/` | `.cursor/skills/` |
| Windsurf | `.windsurf/` | `.windsurf/workflows.json` |
| GitHub Copilot | `.github/` | `.github/copilot/` |
| Cline | `.cline/` | `.cline/skills/` |
| Continue | `.continue/` | `.continue/skills/` |
| Amazon Q | `.amazonq/` | `.amazonq/skills/` |
| Gemini CLI | `.gemini/` | `.gemini/skills/` |
| Qwen Code | `.qwen/` | `.qwen/skills/` |
| CodeBuddy | `.codebuddy/` | `.codebuddy/skills/` |
| Auggie | `.auggie/` | `.auggie/skills/` |
| + 11 more | ... | ... |

## Skill System

### Skills as Prompts

Each skill is a specialized prompt that guides AI behavior. OpenSpec generates 10 skill templates:

| Skill | Purpose | Behavior |
|-------|---------|----------|
| `openspec-explore` | Thinking partner mode | Read-only exploration, no implementation |
| `openspec-new-change` | Start new workflow | Create change directory, set up artifacts |
| `openspec-continue-change` | Resume work | Continue from current artifact state |
| `openspec-apply-change` | Implement specs | Execute tasks from completed designs |
| `openspec-ff-change` | Fast-forward | Skip to specific workflow phase |
| `openspec-sync-specs` | Sync deltas | Apply changes to main specs |
| `openspec-archive-change` | Archive work | Complete and move to archive |
| `openspec-bulk-archive-change` | Batch archive | Archive multiple changes |
| `openspec-verify-change` | Verify completion | Check artifact status |
| `openspec-onboard` | Project onboarding | Guide new team members |

### Skill Template Structure

```typescript
interface SkillTemplate {
  name: string;           // e.g., 'openspec-explore'
  description: string;    // Human-readable description
  instructions: string;   // The prompt content
}
```

### Example Skill: Explore Mode

```markdown
---
name: openspec-explore
description: Enter thinking partner mode for exploration
metadata:
  author: openspec
  version: "1.0"
  generatedBy: "1.1.1"
---

Enter explore mode. Think deeply. Visualize freely.

**IMPORTANT: Explore mode is for thinking, not implementing.**
You may read files, search code, and investigate the codebase,
but you must NEVER write code or implement features.

## The Stance
- Curious, not prescriptive
- Open threads, not interrogations
- Visual - Use ASCII diagrams liberally
- Adaptive - Follow interesting threads
- Patient - Don't rush to conclusions
- Grounded - Explore the actual codebase

## Exploration Tools
- Read files to understand structure
- Search for patterns and dependencies
- Create mental maps of the codebase
- Ask clarifying questions

## Output
- Observations, not prescriptions
- Questions, not answers
- Possibilities, not decisions
```

## Skill Generation

### Generation Flow

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Skill Template  │ -> │ Tool Adapter    │ -> │ Tool-Specific   │
│ (tool-agnostic) │    │ (formatting)    │    │ Skill File      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Implementation

```typescript
// src/core/shared/skill-generation.ts

export function generateSkillContent(
  template: SkillTemplate,
  generatedByVersion: string,
  transformInstructions?: (instructions: string) => string
): string {
  const instructions = transformInstructions
    ? transformInstructions(template.instructions)
    : template.instructions;

  return `---
name: ${template.name}
description: ${template.description}
metadata:
  author: openspec
  version: "1.0"
  generatedBy: "${generatedByVersion}"
---

${instructions}
`;
}
```

## Context Injection

### Project Configuration

AI context is injected through `openspec/config.yaml`:

```yaml
schema: spec-driven

# Project context - applied to ALL artifacts for ALL agents
context: |
  Tech stack: TypeScript, React, Node.js
  API style: RESTful
  We use conventional commits
  Domain: e-commerce platform

# Per-artifact rules - specific guidance
rules:
  proposal:
    - Keep proposals under 500 words
    - Always include a "Non-goals" section
  tasks:
    - Break tasks into chunks of max 2 hours
    - Include success criteria for each task
```

### Instruction Generation

When an AI requests instructions, context and rules are combined:

```typescript
// src/core/artifact-graph/instruction-loader.ts

export function generateInstructions(
  context: ChangeContext,
  artifactId: string,
  projectRoot?: string
): ArtifactInstructions {
  // Read project config
  const projectConfig = readProjectConfig(projectRoot);

  // Extract context and artifact-specific rules
  const configContext = projectConfig?.context?.trim();
  const configRules = projectConfig?.rules?.[artifactId];

  // Combine with schema template
  return {
    changeName: context.changeName,
    artifactId,
    schemaName: context.schemaName,
    description: artifact.description,
    instruction: artifact.instruction,      // From schema
    context: configContext,                  // From config
    rules: configRules,                      // From config
    template: loadTemplate(artifact),        // Template content
    dependencies: getDependencyInfo(),       // What's blocking
    unlocks: getUnlockedArtifacts(),        // What this enables
  };
}
```

### Instruction Structure

```typescript
interface ArtifactInstructions {
  changeName: string;
  artifactId: string;
  schemaName: string;
  description: string;
  instruction: string | undefined;   // Schema-level guidance
  context: string | undefined;       // Project context
  rules: string[] | undefined;       // Artifact-specific rules
  template: string;                  // Template structure
  dependencies: DependencyInfo[];    // Blocking artifacts
  unlocks: string[];                 // What this enables
}
```

## Dependency Tracking

Agents receive information about artifact dependencies:

```
Dependencies for 'tasks':
  ✓ proposal (complete)
  ✓ specs (complete)
  ✓ design (complete)

Unlocks:
  → apply (implementation phase)
```

This helps agents understand workflow state and guide users appropriately.

## Conversational Patterns

### Interactive Prompting

Skills use tools like `AskUserQuestion` for interaction:

```markdown
Use the **AskUserQuestion tool** (open-ended, no preset options) to ask:
> "What change do you want to work on? Describe what you want to build or fix."
```

### Workflow Guidance

Skills provide step-by-step guidance:

```markdown
**Steps**
1. If no clear input provided, ask what they want to build
2. Determine the workflow schema (or use default)
3. Create the change directory with: `openspec new change "<name>"`
4. Explain the artifact workflow
5. Generate first artifact (proposal)
```

## OPSX Slash Commands

OpenSpec provides `/opsx:*` commands that map to skills:

| Command | Skill | Description |
|---------|-------|-------------|
| `/opsx:explore` | `openspec-explore` | Investigation mode |
| `/opsx:new` | `openspec-new-change` | Start change |
| `/opsx:continue` | `openspec-continue-change` | Resume work |
| `/opsx:ff` | `openspec-ff-change` | Fast-forward |
| `/opsx:apply` | `openspec-apply-change` | Implement |
| `/opsx:verify` | `openspec-verify-change` | Validate |
| `/opsx:archive` | `openspec-archive-change` | Complete |

## Tool-Specific Adapters

Each AI tool has a custom adapter handling:
1. **File paths** - Where skills are stored
2. **Metadata format** - Tool-specific frontmatter
3. **Content structure** - Any tool-specific formatting

### Example: Claude vs Cursor

**Claude Code adapter:**
```typescript
export const claudeAdapter: ToolCommandAdapter = {
  toolId: 'claude',

  getFilePath(commandId: string): string {
    return path.join('.claude', 'commands', 'opsx', `${commandId}.md`);
  },

  formatFile(content: CommandContent): string {
    return `---
name: ${escapeYamlValue(content.name)}
description: ${escapeYamlValue(content.description)}
category: ${escapeYamlValue(content.category)}
tags: ${formatTagsArray(content.tags)}
---

${content.body}
`;
  },
};
```

**Cursor adapter:**
```typescript
export const cursorAdapter: ToolCommandAdapter = {
  toolId: 'cursor',

  getFilePath(commandId: string): string {
    return path.join('.cursor', 'commands', `opsx-${commandId}.md`);
  },

  formatFile(content: CommandContent): string {
    return `---
name: /opsx-${content.id}
id: opsx-${content.id}
category: ${escapeYamlValue(content.category)}
description: ${escapeYamlValue(content.description)}
---

${content.body}
`;
  },
};
```

## Agent Workflow

```
┌─────────────────────────────────────────────────┐
│          Complete Agent Workflow               │
├─────────────────────────────────────────────────┤
│                                                 │
│  1. INIT PHASE                                  │
│  ├─ User: openspec init --tools claude         │
│  ├─ System: Detect AI tools                    │
│  └─ System: Generate .claude/skills/           │
│                                                 │
│  2. SKILL GENERATION                           │
│  ├─ Generate 10 skill templates                │
│  ├─ Apply tool-specific adapter                │
│  └─ Write to tool directory                    │
│                                                 │
│  3. PROJECT CONFIGURATION                      │
│  ├─ Create openspec/config.yaml                │
│  ├─ Store project context (50KB max)           │
│  └─ Define per-artifact rules                  │
│                                                 │
│  4. CHANGE CREATION                            │
│  ├─ User: /opsx:new add-feature                │
│  ├─ Agent: openspec new change "add-feature"   │
│  └─ System: Create change directory            │
│                                                 │
│  5. INSTRUCTION GENERATION                     │
│  ├─ Load schema & templates                    │
│  ├─ Inject project context                     │
│  ├─ Apply artifact-specific rules              │
│  └─ Provide dependency status                  │
│                                                 │
│  6. AGENT EXECUTION                            │
│  ├─ Agent reads instructions                   │
│  ├─ Agent creates artifact file                │
│  └─ Progress tracked via file existence        │
│                                                 │
│  7. WORKFLOW PROGRESSION                       │
│  ├─ Dependencies enable/block artifacts        │
│  ├─ Commands transition phases                 │
│  ├─ Archive moves to done                      │
│  └─ Specs sync to main                         │
│                                                 │
└─────────────────────────────────────────────────┘
```

## MCP Compatibility

While OpenSpec doesn't explicitly implement MCP (Model Context Protocol), the architecture is compatible:

1. **Tool-agnostic patterns** - Separation of "what" from "how"
2. **Structured metadata** - Formal frontmatter with name, description
3. **Context injection** - Project context as structured data (similar to MCP resources)
4. **Namespace standardization** - Consistent `/opsx:*` naming

This design would make MCP integration straightforward if needed.
