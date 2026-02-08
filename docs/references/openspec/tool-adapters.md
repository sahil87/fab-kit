# OpenSpec Tool Adapters

## Overview

OpenSpec uses an adapter pattern to generate tool-specific skill files from tool-agnostic command definitions. This allows a single source of truth (skill templates) to work with 22+ AI assistants.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Skill Template │ -> │  Tool Adapter   │ -> │  Tool-Specific  │
│  (universal)    │    │  (formatting)   │    │  Skill File     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Core Types

### Command Content (Tool-Agnostic)

```typescript
// src/core/command-generation/types.ts

interface CommandContent {
  id: string;           // e.g., 'explore', 'new', 'apply'
  name: string;         // e.g., 'OpenSpec Explore'
  description: string;  // Human-readable description
  category: string;     // e.g., 'Workflow', 'Planning'
  tags: string[];       // Search/discovery tags
  body: string;         // The instruction content
}
```

### Tool Command Adapter

```typescript
interface ToolCommandAdapter {
  toolId: string;

  // Where to write the skill file
  getFilePath(commandId: string): string;

  // How to format the content
  formatFile(content: CommandContent): string;
}
```

## Adapter Registry

```typescript
// src/core/command-generation/registry.ts

class CommandAdapterRegistry {
  private adapters: Map<string, ToolCommandAdapter> = new Map();

  register(adapter: ToolCommandAdapter): void {
    this.adapters.set(adapter.toolId, adapter);
  }

  get(toolId: string): ToolCommandAdapter | undefined {
    return this.adapters.get(toolId);
  }

  getAll(): ToolCommandAdapter[] {
    return Array.from(this.adapters.values());
  }
}

// Global registry instance
export const adapterRegistry = new CommandAdapterRegistry();
```

## Built-in Adapters

### Claude Code Adapter

```typescript
// src/core/command-generation/adapters/claude.ts

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

**Output Example (.claude/commands/opsx/explore.md):**
```yaml
---
name: OpenSpec Explore
description: Enter thinking partner mode for exploration
category: Workflow
tags: [openspec, explore, thinking]
---

Enter explore mode. Think deeply. Visualize freely.
...
```

### Cursor Adapter

```typescript
// src/core/command-generation/adapters/cursor.ts

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

**Output Example (.cursor/commands/opsx-explore.md):**
```yaml
---
name: /opsx-explore
id: opsx-explore
category: Workflow
description: Enter thinking partner mode for exploration
---

Enter explore mode. Think deeply. Visualize freely.
...
```

### Windsurf Adapter

```typescript
// src/core/command-generation/adapters/windsurf.ts

export const windsurfAdapter: ToolCommandAdapter = {
  toolId: 'windsurf',

  getFilePath(commandId: string): string {
    return path.join('.windsurf', 'workflows', `opsx-${commandId}.json`);
  },

  formatFile(content: CommandContent): string {
    return JSON.stringify({
      name: `opsx-${content.id}`,
      description: content.description,
      category: content.category,
      instructions: content.body,
    }, null, 2);
  },
};
```

**Output Example (.windsurf/workflows/opsx-explore.json):**
```json
{
  "name": "opsx-explore",
  "description": "Enter thinking partner mode for exploration",
  "category": "Workflow",
  "instructions": "Enter explore mode. Think deeply. Visualize freely.\n..."
}
```

### GitHub Copilot Adapter

```typescript
// src/core/command-generation/adapters/github-copilot.ts

export const copilotAdapter: ToolCommandAdapter = {
  toolId: 'github-copilot',

  getFilePath(commandId: string): string {
    return path.join('.github', 'copilot', 'instructions', `opsx-${commandId}.md`);
  },

  formatFile(content: CommandContent): string {
    return `# ${content.name}

> ${content.description}

${content.body}
`;
  },
};
```

## Complete Adapter List

| Tool ID | Config Directory | Skills Path Pattern |
|---------|-----------------|---------------------|
| `claude` | `.claude/` | `.claude/commands/opsx/{id}.md` |
| `cursor` | `.cursor/` | `.cursor/commands/opsx-{id}.md` |
| `windsurf` | `.windsurf/` | `.windsurf/workflows/opsx-{id}.json` |
| `github-copilot` | `.github/copilot/` | `.github/copilot/instructions/opsx-{id}.md` |
| `cline` | `.cline/` | `.cline/skills/opsx-{id}.md` |
| `continue` | `.continue/` | `.continue/skills/opsx-{id}.md` |
| `amazon-q` | `.amazonq/` | `.amazonq/skills/opsx-{id}.md` |
| `gemini` | `.gemini/` | `.gemini/skills/opsx-{id}.md` |
| `qwen` | `.qwen/` | `.qwen/skills/opsx-{id}.md` |
| `codex` | `.codex/` | `.codex/skills/opsx-{id}.md` |
| `codebuddy` | `.codebuddy/` | `.codebuddy/skills/opsx-{id}.md` |
| `auggie` | `.auggie/` | `.auggie/skills/opsx-{id}.md` |
| `antigravity` | `.antigravity/` | `.antigravity/skills/opsx-{id}.md` |
| + 9 more | ... | ... |

## Creating Custom Adapters

### Implementing a New Adapter

```typescript
// my-tool-adapter.ts

import { ToolCommandAdapter } from '@fission-ai/openspec';

export const myToolAdapter: ToolCommandAdapter = {
  toolId: 'my-tool',

  getFilePath(commandId: string): string {
    // Where your tool expects skill files
    return path.join('.my-tool', 'prompts', `${commandId}.txt`);
  },

  formatFile(content: CommandContent): string {
    // How your tool expects skill content
    return `[${content.name}]
Type: ${content.category}
Tags: ${content.tags.join(', ')}

${content.body}
`;
  },
};
```

### Registering Custom Adapter

```typescript
import { adapterRegistry } from '@fission-ai/openspec';
import { myToolAdapter } from './my-tool-adapter';

adapterRegistry.register(myToolAdapter);
```

## Command Generation Flow

### Generator Function

```typescript
// src/core/command-generation/generator.ts

export function generateCommand(
  adapter: ToolCommandAdapter,
  content: CommandContent,
  outputDir: string
): string {
  // Get tool-specific path
  const filePath = path.join(outputDir, adapter.getFilePath(content.id));

  // Format content for tool
  const formattedContent = adapter.formatFile(content);

  // Ensure directory exists
  fs.mkdirSync(path.dirname(filePath), { recursive: true });

  // Write file
  fs.writeFileSync(filePath, formattedContent);

  return filePath;
}

export function generateCommands(
  adapter: ToolCommandAdapter,
  contents: CommandContent[],
  outputDir: string
): string[] {
  return contents.map(content =>
    generateCommand(adapter, content, outputDir)
  );
}
```

### Init Command Integration

```typescript
// src/core/init.ts (simplified)

async execute(): Promise<void> {
  // 1. Detect or select tools
  const tools = await this.selectTools();

  // 2. Get skill templates
  const skillTemplates = getSkillTemplates();

  // 3. Generate for each tool
  for (const toolId of tools) {
    const adapter = adapterRegistry.get(toolId);
    if (!adapter) continue;

    // Convert templates to CommandContent
    const contents = skillTemplates.map(template => ({
      id: template.name.replace('openspec-', ''),
      name: formatName(template.name),
      description: template.description,
      category: 'Workflow',
      tags: ['openspec', template.name],
      body: template.instructions,
    }));

    // Generate skill files
    generateCommands(adapter, contents, this.projectRoot);
  }
}
```

## Tool Detection

### Auto-Detection Logic

```typescript
// src/core/shared/tool-detection.ts

const TOOL_DETECTORS: Record<string, () => boolean> = {
  'claude': () => fs.existsSync('.claude'),
  'cursor': () => fs.existsSync('.cursor'),
  'windsurf': () => fs.existsSync('.windsurf'),
  'github-copilot': () => fs.existsSync('.github/copilot'),
  // ... more tools
};

export function detectInstalledTools(): string[] {
  return Object.entries(TOOL_DETECTORS)
    .filter(([_, detect]) => detect())
    .map(([toolId]) => toolId);
}
```

### Manual Selection

```bash
# Auto-detect
$ openspec init

# Specify tools
$ openspec init --tools claude,cursor,windsurf
```

## Version Tracking

### Skill Metadata

Each generated skill includes version metadata:

```yaml
---
name: OpenSpec Explore
metadata:
  author: openspec
  version: "1.0"
  generatedBy: "1.1.1"  # OpenSpec version
---
```

### Upgrade Detection

```typescript
function needsUpgrade(skillPath: string, currentVersion: string): boolean {
  const content = fs.readFileSync(skillPath, 'utf-8');
  const match = content.match(/generatedBy:\s*"([^"]+)"/);

  if (!match) return true;  // No version, needs upgrade

  const skillVersion = match[1];
  return semver.lt(skillVersion, currentVersion);
}
```

## Update Command

Regenerates skills for configured tools:

```bash
# Update all configured tools
$ openspec update

# Force update even if versions match
$ openspec update --force
```

### Update Logic

```typescript
async execute(): Promise<void> {
  // 1. Find configured tools (from .openspec directory)
  const tools = this.findConfiguredTools();

  // 2. For each tool
  for (const toolId of tools) {
    const adapter = adapterRegistry.get(toolId);

    // 3. Check if upgrade needed
    const existingPath = adapter.getFilePath('explore');
    if (!this.options.force && !needsUpgrade(existingPath, VERSION)) {
      continue;
    }

    // 4. Regenerate skills
    generateCommands(adapter, getSkillTemplates(), this.projectRoot);
  }
}
```

## YAML Escaping

Utility for safe YAML values:

```typescript
function escapeYamlValue(value: string): string {
  if (/[:#\[\]{}|>&*!,?'"]/.test(value) || value.includes('\n')) {
    return `"${value.replace(/"/g, '\\"')}"`;
  }
  return value;
}

function formatTagsArray(tags: string[]): string {
  return `[${tags.map(t => escapeYamlValue(t)).join(', ')}]`;
}
```

## Adapter Factory

```typescript
// src/core/command-generation/factory.ts

export function createAdapter(toolId: string): ToolCommandAdapter | null {
  // Check registry first
  const registered = adapterRegistry.get(toolId);
  if (registered) return registered;

  // Check built-in adapters
  switch (toolId) {
    case 'claude': return claudeAdapter;
    case 'cursor': return cursorAdapter;
    case 'windsurf': return windsurfAdapter;
    case 'github-copilot': return copilotAdapter;
    // ... more cases
    default: return null;
  }
}
```

## Testing Adapters

```typescript
// test/core/command-generation/adapters.test.ts

describe('Claude Adapter', () => {
  it('generates correct file path', () => {
    const path = claudeAdapter.getFilePath('explore');
    expect(path).toBe('.claude/commands/opsx/explore.md');
  });

  it('formats content correctly', () => {
    const content: CommandContent = {
      id: 'explore',
      name: 'OpenSpec Explore',
      description: 'Enter exploration mode',
      category: 'Workflow',
      tags: ['openspec', 'explore'],
      body: 'Instructions here...',
    };

    const formatted = claudeAdapter.formatFile(content);

    expect(formatted).toContain('name: OpenSpec Explore');
    expect(formatted).toContain('category: Workflow');
    expect(formatted).toContain('Instructions here...');
  });
});
```
