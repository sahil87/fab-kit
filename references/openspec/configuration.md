# OpenSpec Configuration

## Configuration Hierarchy

OpenSpec uses a layered configuration system:

1. **CLI flags** - Highest priority, per-command overrides
2. **Change metadata** - Per-change settings (`.openspec.yaml`)
3. **Project config** - Project-wide settings (`openspec/config.yaml`)
4. **Global config** - User-level settings (XDG directories)
5. **Defaults** - Built-in fallbacks

## Project Configuration

### Location

```
project/
└── openspec/
    └── config.yaml
```

### Schema

```yaml
# openspec/config.yaml

# Required: Which workflow schema to use
schema: spec-driven

# Optional: Project-wide context (max 50KB)
# Injected into ALL artifact instructions for ALL agents
context: |
  Tech stack: TypeScript, React, Node.js
  Database: PostgreSQL with Prisma ORM
  API style: RESTful with OpenAPI specs
  Testing: Jest + React Testing Library
  CI: GitHub Actions

  Coding conventions:
  - Use functional components
  - Prefer composition over inheritance
  - All async functions must handle errors

# Optional: Per-artifact rules
# Applied to specific artifacts during instruction generation
rules:
  proposal:
    - Keep proposals under 500 words
    - Always include a "Non-goals" section
    - Focus on user impact, not technical details

  specs:
    - Use RFC 2119 keywords consistently
    - Include at least one scenario per requirement
    - Reference existing specs when modifying

  design:
    - Include data flow diagrams
    - Document all API changes
    - List affected files explicitly

  tasks:
    - Break into chunks of max 2 hours
    - Include success criteria for each task
    - Flag technical unknowns with [SPIKE]
```

### Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `schema` | string | Yes | Workflow schema name |
| `context` | string | No | Project-wide context (max 50KB) |
| `rules` | object | No | Per-artifact rule arrays |

### Context Size Limit

```typescript
const MAX_CONTEXT_SIZE = 50 * 1024; // 50KB

function validateConfig(config: ProjectConfig): ValidationResult {
  if (config.context && config.context.length > MAX_CONTEXT_SIZE) {
    return {
      valid: false,
      error: `Context exceeds ${MAX_CONTEXT_SIZE} bytes`,
    };
  }
  return { valid: true };
}
```

## Change Metadata

### Location

```
openspec/changes/[change-name]/
└── .openspec.yaml
```

### Schema

```yaml
# .openspec.yaml

# Optional: Override schema for this change
schema: custom-workflow

# Optional: Change description
description: |
  This change implements 2FA authentication
  using TOTP with optional SMS fallback.
```

### Creation

Automatically created by `openspec new change <name>`:

```bash
$ openspec new change add-2fa --schema research-first

Created: openspec/changes/add-2fa/
  .openspec.yaml
```

## Global Configuration

### XDG Base Directory Compliance

OpenSpec follows XDG Base Directory specification:

| Platform | Config Location |
|----------|-----------------|
| Linux | `~/.config/openspec/` |
| macOS | `~/Library/Application Support/openspec/` |
| Windows | `%APPDATA%\openspec\` |

### Global Config File

```yaml
# ~/.config/openspec/config.yaml

# Default schema for new projects
defaultSchema: spec-driven

# Telemetry preference
telemetry: false

# Default tools to configure on init
defaultTools:
  - claude
  - cursor
```

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `OPENSPEC_TELEMETRY` | Enable/disable telemetry | `1` (enabled) |
| `DO_NOT_TRACK` | Standard telemetry opt-out | `0` |
| `OPENSPEC_NO_COMPLETIONS` | Skip completion install | `0` |
| `NO_COLOR` | Disable colored output | unset |
| `CI` | Skip interactive features | unset |

### Telemetry Control

```bash
# Disable telemetry
export OPENSPEC_TELEMETRY=0

# Or use standard DO_NOT_TRACK
export DO_NOT_TRACK=1
```

### CI Detection

When `CI=true` or `CI=1`:
- Telemetry is disabled
- Completion installation is skipped
- Interactive prompts use defaults

## CLI Option Precedence

### Schema Resolution

```
1. --schema flag          (highest priority)
2. .openspec.yaml schema
3. config.yaml schema
4. "spec-driven"          (default)
```

### Example

```bash
# Uses CLI flag
$ openspec status --schema minimal

# Uses change metadata (.openspec.yaml)
$ openspec status --change my-change

# Uses project config (config.yaml)
$ openspec status

# Uses default if nothing configured
$ openspec status  # -> spec-driven
```

## Tool Configuration

### Auto-Detection

On `openspec init`, tools are auto-detected by checking for:

| Tool | Detection |
|------|-----------|
| Claude Code | `.claude/` directory |
| Cursor | `.cursor/` directory |
| Windsurf | `.windsurf/` directory |
| GitHub Copilot | `.github/copilot/` directory |

### Manual Tool Selection

```bash
# Specify tools explicitly
$ openspec init --tools claude,cursor

# Update tools later
$ openspec update --tools windsurf
```

### Tool Configuration File

Each tool gets skill files in its config directory:

```
.claude/
└── skills/
    ├── openspec-explore.md
    ├── openspec-new-change.md
    ├── openspec-continue-change.md
    ├── openspec-apply-change.md
    ├── openspec-ff-change.md
    ├── openspec-sync-specs.md
    ├── openspec-archive-change.md
    ├── openspec-bulk-archive-change.md
    ├── openspec-verify-change.md
    └── openspec-onboard.md
```

## Configuration Loading

### Implementation

```typescript
// src/core/project-config.ts

export interface ProjectConfig {
  schema: string;
  context?: string;
  rules?: Record<string, string[]>;
}

export function readProjectConfig(projectRoot?: string): ProjectConfig | null {
  const configPath = path.join(
    projectRoot ?? process.cwd(),
    'openspec',
    'config.yaml'
  );

  if (!fs.existsSync(configPath)) {
    return null;
  }

  const content = fs.readFileSync(configPath, 'utf-8');
  const config = yaml.parse(content);

  // Validate
  return ProjectConfigSchema.parse(config);
}
```

### No Caching

Configuration is read directly from filesystem on each access:
- Ensures immediate reflection of changes
- No stale state issues
- Simplifies implementation

```typescript
// Direct read, no cache
const config = readProjectConfig();
```

## Validation

### Zod Schema

```typescript
import { z } from 'zod';

const ProjectConfigSchema = z.object({
  schema: z.string(),
  context: z.string().max(50 * 1024).optional(),
  rules: z.record(z.string(), z.array(z.string())).optional(),
});

const ChangeMetadataSchema = z.object({
  schema: z.string().optional(),
  description: z.string().optional(),
});
```

### Error Messages

```
Error: Invalid project configuration

  - context: String must contain at most 51200 characters
  - rules.proposal: Expected array, received string
```

## Creating Configuration

### Interactive Init

```bash
$ openspec init

? Select AI tools to configure: (Press space to select)
❯ ◉ Claude Code
  ◉ Cursor
  ◯ Windsurf
  ◯ GitHub Copilot

Creating openspec/config.yaml...
Creating .claude/skills/...
Creating .cursor/skills/...

✓ OpenSpec initialized
```

### Programmatic Creation

```typescript
import { InitCommand } from '@fission-ai/openspec';

const init = new InitCommand({
  tools: ['claude', 'cursor'],
  force: false,
});

await init.execute('/path/to/project');
```

## Configuration Examples

### Minimal Configuration

```yaml
# openspec/config.yaml
schema: spec-driven
```

### Full Configuration

```yaml
# openspec/config.yaml
schema: spec-driven

context: |
  ## Project: E-Commerce Platform

  ### Tech Stack
  - Frontend: React 18 + TypeScript
  - Backend: Node.js + Express
  - Database: PostgreSQL + Prisma
  - Cache: Redis

  ### Conventions
  - REST API with versioning (/api/v1/)
  - Conventional commits
  - Feature branches from main
  - PR required for all changes

  ### Testing
  - Unit: Jest
  - E2E: Playwright
  - Coverage: 80% minimum

rules:
  proposal:
    - Include user story format
    - Estimate complexity (S/M/L/XL)
    - List affected services

  specs:
    - Follow Given/When/Then format
    - Include error scenarios
    - Reference API contracts

  design:
    - Include sequence diagrams for complex flows
    - Document database migrations
    - List breaking changes

  tasks:
    - Maximum 4 hours per task
    - Include PR checklist items
    - Tag with [TEST], [DOC], [DEPLOY] as needed
```

### Custom Schema Configuration

```yaml
# openspec/config.yaml
schema: research-first  # Custom schema in openspec/schemas/

context: |
  This project uses a research-first workflow.
  All changes require a spike document before proposal.

rules:
  research:
    - Time-box to 2 hours
    - Document findings even if negative
    - Include code samples where helpful

  proposal:
    - Reference research findings
    - Include confidence level
```
