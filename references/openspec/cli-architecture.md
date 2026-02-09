# OpenSpec CLI Architecture

## Overview

The OpenSpec CLI is a Commander.js-based command-line interface with 30+ subcommands organized into logical groups.

## Entry Points

**Binary:** `bin/openspec.js`
```javascript
#!/usr/bin/env node
import '../dist/cli/index.js';
```

**Main CLI:** `src/cli/index.ts` (~509 lines)
- Initializes Commander program
- Registers all commands
- Sets up global hooks
- Handles telemetry

## Command Structure

### Top-Level Commands

| Command | Description | Key Options |
|---------|-------------|-------------|
| `init [path]` | Initialize OpenSpec in project | `--tools`, `--force` |
| `update [path]` | Update instruction files | `--force` |
| `list` | List changes or specs | `--specs`, `--changes`, `--json` |
| `view` | Interactive dashboard | - |
| `archive [name]` | Archive completed changes | `-y`, `--skip-specs`, `--no-validate` |
| `validate [name]` | Validate changes/specs | `--all`, `--strict`, `--json` |
| `show [name]` | Display change or spec | `--json`, `--deltas-only` |
| `feedback <msg>` | Submit feedback | `--body` |

### Workflow Commands

| Command | Description | Key Options |
|---------|-------------|-------------|
| `status` | Artifact completion status | `--change`, `--schema`, `--json` |
| `instructions [artifact]` | Output enriched instructions | `--change`, `--schema`, `--json` |
| `templates` | Show resolved template paths | `--schema`, `--json` |
| `schemas` | List available schemas | `--json` |
| `new change <name>` | Create new change | `--description`, `--schema` |

### Command Groups

**`change` (deprecated)**
```
openspec change list              # List changes
openspec change show [name]       # Show change
openspec change validate [name]   # Validate change
```

**`completion`**
```
openspec completion generate [shell]   # Output script
openspec completion install [shell]    # Install completion
openspec completion uninstall [shell]  # Remove completion
```

**`spec`**
```
openspec spec show [name]      # Display spec
openspec spec validate [name]  # Validate spec
```

**`config`**
```
openspec config show           # Show configuration
```

**`schema`**
```
openspec schema list           # List schemas
openspec schema show [name]    # Show schema details
```

## Command Implementation Pattern

Each major command follows a class-based pattern:

```typescript
// src/commands/validate.ts
export interface ValidateCommandOptions {
  all?: boolean;
  changes?: boolean;
  specs?: boolean;
  type?: string;
  strict?: boolean;
  json?: boolean;
  concurrency?: number;
  interactive?: boolean;
}

export class ValidateCommand {
  private options: ValidateCommandOptions;

  constructor(options: ValidateCommandOptions) {
    this.options = options;
  }

  async execute(itemName?: string): Promise<void> {
    // Implementation
  }
}
```

**CLI Registration:**
```typescript
// src/cli/index.ts
program
  .command('validate [item-name]')
  .description('Validate changes and specs')
  .option('--all', 'Validate all changes and specs')
  .option('--strict', 'Enable strict validation')
  .action(async (itemName, options) => {
    const command = new ValidateCommand(options);
    await command.execute(itemName);
  });
```

## Global Options

```typescript
program.option('--no-color', 'Disable color output');
```

Applies `NO_COLOR` environment variable when set.

## Hooks System

### Pre-Action Hooks (Global)

```typescript
program.hook('preAction', async () => {
  // Apply NO_COLOR if flag is set
  if (program.opts().color === false) {
    process.env.NO_COLOR = '1';
  }

  // Show telemetry notice on first run
  await maybeShowTelemetryNotice();

  // Track command execution
  await trackCommand(commandPath, version);
});
```

### Post-Action Hooks (Global)

```typescript
program.hook('postAction', async () => {
  // Flush telemetry
  await shutdown();
});
```

### Command-Specific Hooks

```typescript
// Deprecated command warning
changeCommand.hook('preAction', () => {
  console.warn(chalk.yellow('Warning: `openspec change` is deprecated...'));
});
```

## Core Modules

### Init Command (`src/core/init.ts`)

```typescript
export class InitCommand {
  async execute(targetPath?: string): Promise<void> {
    // 1. Detect installed AI tools
    const tools = await detectInstalledTools();

    // 2. Prompt for tool selection (or use --tools flag)
    const selectedTools = await selectTools(tools);

    // 3. Create openspec/ directory structure
    await createDirectoryStructure(targetPath);

    // 4. Generate skill files for each tool
    for (const tool of selectedTools) {
      await generateSkills(tool);
    }

    // 5. Create config.yaml
    await createConfig(targetPath);
  }
}
```

### Archive Command (`src/core/archive.ts`)

```typescript
export class ArchiveCommand {
  async execute(changeName?: string): Promise<void> {
    // 1. Find change to archive
    const change = await findChange(changeName);

    // 2. Validate (unless --no-validate)
    if (!this.options.noValidate) {
      await validateChange(change);
    }

    // 3. Merge delta specs (unless --skip-specs)
    if (!this.options.skipSpecs) {
      await mergeDeltas(change);
    }

    // 4. Move to archive/YYYY-MM-DD-name/
    await moveToArchive(change);
  }
}
```

### Validate Command (`src/commands/validate.ts`)

Validates changes and specs against schemas:
- Checks required sections (Purpose, Requirements, Scenarios)
- Validates RFC 2119 keyword usage
- Reports errors, warnings, and info
- Supports strict mode for enhanced validation

### Show Command (`src/commands/show.ts`)

Displays change or spec content:
- `--deltas-only` - Only show delta sections
- `--requirements-only` - Only show requirements
- `--json` - Machine-readable output

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Validation failed |

## Output Formatting

Uses `chalk` for terminal colors and `ora` for spinners:

```typescript
import chalk from 'chalk';
import ora from 'ora';

const spinner = ora('Validating...').start();
// ...
spinner.succeed('Validation complete');
console.log(chalk.green('✓ All specs valid'));
```

## Error Handling

Graceful degradation pattern:
```typescript
try {
  await riskyOperation();
} catch (error) {
  if (error.code === 'ENOENT') {
    console.error(chalk.red(`File not found: ${error.path}`));
    process.exit(1);
  }
  throw error; // Re-throw unexpected errors
}
```

## Telemetry Integration

Located in `src/telemetry/`:
- PostHog for anonymous usage tracking
- Only tracks: command names, versions
- No PII, arguments, or content
- Auto-disabled in CI
- Opt-out: `OPENSPEC_TELEMETRY=0` or `DO_NOT_TRACK=1`

## Programmatic Usage

Commands can be used programmatically:

```typescript
import { ValidateCommand } from '@fission-ai/openspec';

const command = new ValidateCommand({ all: true, strict: true });
await command.execute();
```

## Command Reference Summary

```
openspec init [path]                    # Initialize project
openspec update [path]                  # Update instructions
openspec new change <name>              # Create change
openspec list [--specs|--changes]       # List items
openspec show [name]                    # Display item
openspec validate [name]                # Validate item
openspec archive [name]                 # Archive change
openspec status                         # Artifact status
openspec instructions [artifact]        # Get instructions
openspec templates                      # Show templates
openspec schemas                        # List schemas
openspec completion generate [shell]    # Shell completion
openspec completion install [shell]     # Install completion
openspec feedback <message>             # Submit feedback
```
