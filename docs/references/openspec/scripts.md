# OpenSpec Scripts

## NPM Scripts

Located in `package.json`:

| Script | Command | Purpose |
|--------|---------|---------|
| `build` | `node build.js` | Compile TypeScript to dist/ |
| `dev` | `tsc --watch` | Watch mode compilation |
| `dev:cli` | `pnpm build && node bin/openspec.js` | Build and run CLI |
| `lint` | `eslint src/` | Lint source code |
| `test` | `vitest run` | Run test suite |
| `test:watch` | `vitest` | Watch mode tests |
| `test:ui` | `vitest --ui` | Visual test UI |
| `test:coverage` | `vitest --coverage` | Coverage report |
| `prepare` | `pnpm run build` | Pre-publish build |
| `prepublishOnly` | `pnpm run build` | Ensure built before publish |
| `postinstall` | `node scripts/postinstall.js` | Auto-install completions |
| `release` | `pnpm run release:ci` | Full release process |
| `release:ci` | `check:pack-version && changeset publish` | CI release |
| `changeset` | `changeset` | Create changeset |

## Build System

### Main Build Script (`build.js`)

```javascript
// build.js - Node.js build orchestrator

import { execSync } from 'child_process';
import { rmSync } from 'fs';

// 1. Clean dist/
rmSync('dist', { recursive: true, force: true });

// 2. Run TypeScript compiler
try {
  execSync('npx tsc', { stdio: 'inherit' });
  console.log('✅ Build successful');
} catch (error) {
  console.error('❌ Build failed');
  process.exit(1);
}
```

### TypeScript Configuration

```json
// tsconfig.json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "outDir": "dist",
    "rootDir": "src",
    "strict": true,
    "esModuleInterop": true,
    "declaration": true
  }
}
```

## Postinstall Automation

### Shell Completion Auto-Install (`scripts/postinstall.js`)

Automatically installs shell completions after `npm install`:

```javascript
// scripts/postinstall.js

// Skip conditions
if (process.env.CI === 'true' || process.env.CI === '1') {
  process.exit(0); // Skip in CI
}

if (process.env.OPENSPEC_NO_COMPLETIONS === '1') {
  process.exit(0); // User opted out
}

if (!fs.existsSync('dist/')) {
  process.exit(0); // Dev environment
}

// Install completions
const shell = detectShell();
const factory = new CompletionFactory();
const installer = factory.createInstaller(shell);

try {
  await installer.install();
  console.log(`✓ Installed ${shell} completions`);
} catch (error) {
  // Never fail npm install
  console.warn(`Could not install completions: ${error.message}`);
}
```

### Skip Conditions

| Condition | Environment Variable |
|-----------|---------------------|
| CI environment | `CI=true` or `CI=1` |
| User opt-out | `OPENSPEC_NO_COMPLETIONS=1` |
| Dev setup | `dist/` doesn't exist |

## Release Scripts

### Pack Version Check (`scripts/pack-version-check.mjs`)

Guards that packed tarball matches package.json version:

```javascript
// scripts/pack-version-check.mjs

// 1. Create tarball
const packResult = execSync('npm pack --json');
const tarball = JSON.parse(packResult)[0].filename;

// 2. Install in temp directory
const tempDir = mkdtempSync('openspec-test-');
execSync(`npm install ${tarball}`, { cwd: tempDir });

// 3. Run --version
const version = execSync('npx openspec --version', { cwd: tempDir });

// 4. Compare
const expected = require('./package.json').version;
if (version.trim() !== expected) {
  console.error(`Version mismatch: got ${version}, expected ${expected}`);
  process.exit(1);
}

// 5. Cleanup
rmSync(tempDir, { recursive: true });
rmSync(tarball);
```

### Flake.nix Update (`scripts/update-flake.sh`)

Updates Nix dependency hash after lockfile changes:

```bash
#!/bin/bash
# scripts/update-flake.sh

# Detect OS for sed compatibility
if [[ "$OSTYPE" == "darwin"* ]]; then
  SED="sed -i ''"
else
  SED="sed -i"
fi

# Read version
VERSION=$(node -p "require('./package.json').version")

# Set placeholder hash
$SED "s/pnpmDepsHash = .*/pnpmDepsHash = \"sha256-PLACEHOLDER\";/" flake.nix

# Attempt build to capture correct hash
nix build 2>&1 | grep "got:" | awk '{print $2}' > /tmp/hash

# Update with correct hash
HASH=$(cat /tmp/hash)
$SED "s/sha256-PLACEHOLDER/$HASH/" flake.nix

# Verify
nix build || exit 1

echo "✅ flake.nix updated for version $VERSION"
```

## Shell Completion Scripts

### Generated Completion Scripts

The CLI generates completion scripts for multiple shells:

**Zsh (`_openspec`):**
```zsh
#compdef openspec

_openspec() {
  local -a commands
  commands=(
    'init:Initialize OpenSpec in project'
    'list:List changes or specs'
    'validate:Validate changes and specs'
    # ...
  )

  _describe 'command' commands
}

compdef _openspec openspec
```

**Bash (`openspec.bash`):**
```bash
_openspec_completions() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local commands="init list validate archive show"

  COMPREPLY=($(compgen -W "$commands" -- "$cur"))
}

complete -F _openspec_completions openspec
```

**Fish (`openspec.fish`):**
```fish
complete -c openspec -n "__fish_use_subcommand" -a init -d "Initialize OpenSpec"
complete -c openspec -n "__fish_use_subcommand" -a list -d "List changes or specs"
# ...
```

### Completion Installation Locations

| Shell | Location |
|-------|----------|
| Zsh (Oh My Zsh) | `~/.oh-my-zsh/completions/_openspec` |
| Zsh (standard) | `~/.zsh/completions/_openspec` |
| Bash | `~/.bash_completion.d/openspec` |
| Fish | `~/.config/fish/completions/openspec.fish` |

## Test Infrastructure

### CLI Test Helper (`test/helpers/run-cli.ts`)

```typescript
// test/helpers/run-cli.ts

let cliBuilt = false;

export async function ensureCliBuilt(): Promise<void> {
  if (!cliBuilt) {
    execSync('pnpm build', { stdio: 'inherit' });
    cliBuilt = true;
  }
}

export interface RunCLIResult {
  exitCode: number;
  stdout: string;
  stderr: string;
  timedOut: boolean;
}

export async function runCLI(
  args: string[],
  options?: { stdin?: string; timeout?: number; cwd?: string }
): Promise<RunCLIResult> {
  await ensureCliBuilt();

  const child = spawn('node', ['bin/openspec.js', ...args], {
    cwd: options?.cwd ?? process.cwd(),
  });

  // Handle stdin
  if (options?.stdin) {
    child.stdin.write(options.stdin);
    child.stdin.end();
  }

  // Collect output
  let stdout = '';
  let stderr = '';

  child.stdout.on('data', (data) => { stdout += data; });
  child.stderr.on('data', (data) => { stderr += data; });

  // Wait for exit
  return new Promise((resolve) => {
    const timeout = setTimeout(() => {
      child.kill();
      resolve({ exitCode: -1, stdout, stderr, timedOut: true });
    }, options?.timeout ?? 30000);

    child.on('exit', (code) => {
      clearTimeout(timeout);
      resolve({ exitCode: code ?? 0, stdout, stderr, timedOut: false });
    });
  });
}
```

### Test Fixtures

Located in `test/fixtures/`:
- Sample changes
- Valid/invalid specs
- Schema definitions
- Expected outputs

## Spec-Kit Bash Scripts

### Common Utilities (`scripts/bash/common.sh`)

```bash
#!/bin/bash
# Shared utility functions

get_repo_root() {
  if git rev-parse --show-toplevel 2>/dev/null; then
    return
  fi

  # Fallback: look for .specify marker
  local dir="$PWD"
  while [[ "$dir" != "/" ]]; do
    if [[ -f "$dir/.specify" ]]; then
      echo "$dir"
      return
    fi
    dir=$(dirname "$dir")
  done

  echo "$PWD"
}

get_current_branch() {
  git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main"
}

has_git() {
  git rev-parse --is-inside-work-tree &>/dev/null
}

check_feature_branch() {
  local branch="$1"
  [[ "$branch" =~ ^[0-9]+-[a-z0-9-]+$ ]]
}

find_feature_dir_by_prefix() {
  local prefix="$1"
  find specs/ -maxdepth 1 -type d -name "${prefix}-*" 2>/dev/null | head -1
}
```

### Feature Creation (`scripts/bash/create-new-feature.sh`)

```bash
#!/bin/bash
# Create new feature with numbered prefix

source "$(dirname "$0")/common.sh"

# Parse options
JSON_OUTPUT=false
SHORT_NAME=""
MANUAL_NUMBER=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --json) JSON_OUTPUT=true ;;
    --short-name) SHORT_NAME="$2"; shift ;;
    --number) MANUAL_NUMBER="$2"; shift ;;
    --help) show_help; exit 0 ;;
    *) DESCRIPTION="$1" ;;
  esac
  shift
done

# Find next number
REPO_ROOT=$(get_repo_root)
HIGHEST=$(ls -d "$REPO_ROOT/specs/"[0-9]*-* 2>/dev/null | \
  sed 's/.*\/\([0-9]*\)-.*/\1/' | sort -n | tail -1)
NEXT_NUMBER=$((${HIGHEST:-0} + 1))
NEXT_NUMBER=$(printf "%03d" "$NEXT_NUMBER")

# Generate feature name
if [[ -n "$SHORT_NAME" ]]; then
  FEATURE_NAME="${NEXT_NUMBER}-${SHORT_NAME}"
else
  FEATURE_NAME="${NEXT_NUMBER}-$(echo "$DESCRIPTION" | tr ' ' '-' | tr '[:upper:]' '[:lower:]')"
fi

# Output
if $JSON_OUTPUT; then
  echo "{\"feature\": \"$FEATURE_NAME\", \"number\": $NEXT_NUMBER}"
else
  echo "$FEATURE_NAME"
fi
```

### Prerequisite Checker (`scripts/bash/check-prerequisites.sh`)

```bash
#!/bin/bash
# Check feature prerequisites before operations

source "$(dirname "$0")/common.sh"

JSON_OUTPUT=false
REQUIRE_TASKS=false
INCLUDE_TASKS=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --json) JSON_OUTPUT=true ;;
    --require-tasks) REQUIRE_TASKS=true ;;
    --include-tasks) INCLUDE_TASKS=true ;;
    --paths-only) PATHS_ONLY=true ;;
    *) FEATURE="$1" ;;
  esac
  shift
done

# Find feature directory
REPO_ROOT=$(get_repo_root)
FEATURE_DIR=$(find_feature_dir_by_prefix "${FEATURE%%-*}")

# Check required files
PLAN_EXISTS=false
TASKS_EXISTS=false

[[ -f "$FEATURE_DIR/plan.md" ]] && PLAN_EXISTS=true
[[ -f "$FEATURE_DIR/tasks.md" ]] && TASKS_EXISTS=true

# Validate
if ! $PLAN_EXISTS; then
  echo "ERROR: plan.md not found" >&2
  exit 1
fi

if $REQUIRE_TASKS && ! $TASKS_EXISTS; then
  echo "ERROR: tasks.md required but not found" >&2
  exit 1
fi

# Output paths
echo "FEATURE_DIR=$FEATURE_DIR"
echo "PLAN_PATH=$FEATURE_DIR/plan.md"
$INCLUDE_TASKS && echo "TASKS_PATH=$FEATURE_DIR/tasks.md"
```

## GitHub Actions Scripts

### Release Package Builder (`.github/workflows/scripts/create-release-packages.sh`)

```bash
#!/bin/bash
# Build release packages for all AI assistants

AGENTS=${AGENTS:-"claude cursor windsurf copilot cline"}
SCRIPTS=${SCRIPTS:-"sh"}

for agent in $AGENTS; do
  echo "Building $agent package..."

  # Create package directory
  mkdir -p "dist/$agent"

  # Generate commands from templates
  for template in templates/*.md; do
    output="dist/$agent/$(basename "$template")"

    # Replace placeholders
    sed -e "s/{SCRIPT}/openspec/g" \
        -e "s/{AGENT_SCRIPT}/$agent/g" \
        "$template" > "$output"

    # Remove YAML frontmatter
    sed -i '1{/^---$/,/^---$/d}' "$output"

    # Normalize paths to .specify/
    sed -i 's|openspec/|.specify/|g' "$output"
  done

  # Create archive
  tar -czf "openspec-$agent.tar.gz" -C dist "$agent"
done

echo "✅ Release packages created"
```

## Utility Scripts

### Command Reference Generator

Generates markdown documentation from CLI structure:

```typescript
// src/utils/command-references.ts

export function generateCommandReference(program: Command): string {
  let output = '# Command Reference\n\n';

  for (const command of program.commands) {
    output += `## ${command.name()}\n\n`;
    output += `${command.description()}\n\n`;
    output += '```bash\n';
    output += `openspec ${command.name()} ${command.usage()}\n`;
    output += '```\n\n';

    if (command.options.length) {
      output += '**Options:**\n\n';
      for (const option of command.options) {
        output += `- \`${option.flags}\` - ${option.description}\n`;
      }
      output += '\n';
    }
  }

  return output;
}
```
