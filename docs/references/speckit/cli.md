# Specify CLI

## Overview

`specify` is a Python CLI tool that bootstraps Spec-Kit projects. It downloads templates, sets up directory structures, and configures AI agent integrations.

## Installation

### Persistent Installation (Recommended)

```bash
uv tool install specify-cli --from git+https://github.com/github/spec-kit.git
```

### One-time Usage

```bash
uvx --from git+https://github.com/github/spec-kit.git specify init <PROJECT_NAME>
```

### Upgrade

```bash
uv tool install specify-cli --force --from git+https://github.com/github/spec-kit.git
```

---

## Commands

### `specify init`

Initialize a new Spec-Kit project.

**Usage**:
```bash
specify init <project-name> [OPTIONS]
specify init . [OPTIONS]           # Current directory
specify init --here [OPTIONS]      # Current directory (alternative)
```

**Arguments**:
| Argument | Description |
|----------|-------------|
| `<project-name>` | Name for new project directory |
| `.` | Initialize in current directory |

**Options**:
| Option | Type | Description |
|--------|------|-------------|
| `--ai` | String | AI assistant: `claude`, `gemini`, `copilot`, `cursor-agent`, `qwen`, `opencode`, `codex`, `windsurf`, `kilocode`, `auggie`, `codebuddy`, `amp`, `shai`, `q`, `bob`, `qoder` |
| `--script` | String | Script type: `sh` (bash/zsh) or `ps` (PowerShell) |
| `--ignore-agent-tools` | Flag | Skip CLI tool checks |
| `--no-git` | Flag | Skip git repository initialization |
| `--here` | Flag | Initialize in current directory |
| `--force` | Flag | Force merge/overwrite (skip confirmation) |
| `--skip-tls` | Flag | Skip SSL/TLS verification |
| `--debug` | Flag | Enable diagnostic output |
| `--github-token` | String | GitHub token for API requests |

**Examples**:
```bash
# Basic project
specify init my-project

# With specific AI agent
specify init my-project --ai claude

# In current directory with force
specify init . --force --ai copilot

# Skip git initialization
specify init my-project --ai gemini --no-git

# PowerShell scripts on Windows
specify init my-project --ai copilot --script ps

# With GitHub token (for rate limits)
specify init my-project --ai claude --github-token ghp_xxx
```

---

### `specify check`

Check for installed tools (git, AI assistants, VS Code).

**Usage**:
```bash
specify check
```

**Output**:
```
Check Available Tools
├── ● git (available)
├── ● claude (available)
├── ○ gemini (not found)
├── ○ cursor-agent (IDE-based, no CLI check)
├── ● code (available)
└── ○ code-insiders (not found)
```

---

### `specify version`

Display version and system information.

**Usage**:
```bash
specify version
```

**Output**:
```
┌─────────────────────────────────────┐
│ CLI Version       1.2.3            │
│ Template Version  1.2.3            │
│ Released          2025-01-15       │
│                                    │
│ Python            3.11.0           │
│ Platform          Darwin           │
│ Architecture      arm64            │
│ OS Version        14.0             │
└─────────────────────────────────────┘
```

---

## Initialization Process

### Step-by-Step

1. **Show banner** - ASCII art and tagline
2. **Validate project path** - Check for conflicts
3. **Check prerequisites** - Git (optional), AI tool (if CLI-based)
4. **Select AI assistant** - Interactive or via `--ai`
5. **Select script type** - Auto-detect or via `--script`
6. **Download template** - From GitHub releases
7. **Extract template** - To project directory
8. **Set script permissions** - chmod +x on POSIX
9. **Initialize git** - If not `--no-git` and git available
10. **Display next steps** - Slash commands to use

### Template Download

The CLI fetches the latest release from GitHub:

```python
api_url = "https://api.github.com/repos/github/spec-kit/releases/latest"
pattern = f"spec-kit-template-{ai_assistant}-{script_type}"
```

Asset naming: `spec-kit-template-claude-sh-1.2.3.zip`

### Rate Limiting

GitHub API has rate limits:
- **Unauthenticated**: 60 requests/hour
- **Authenticated**: 5,000 requests/hour

Use `--github-token` or set `GH_TOKEN`/`GITHUB_TOKEN` environment variable.

---

## Technical Details

### Dependencies

```python
# requirements
requires-python = ">=3.11"
dependencies = [
    "typer",      # CLI framework
    "rich",       # Terminal formatting
    "platformdirs", # Platform-specific directories
    "readchar",   # Keyboard input
    "httpx",      # HTTP client
]
```

### Agent Configuration

Single source of truth for all agent metadata:

```python
AGENT_CONFIG = {
    "claude": {
        "name": "Claude Code",
        "folder": ".claude/",
        "install_url": "https://docs.anthropic.com/...",
        "requires_cli": True,
    },
    # ... more agents
}
```

### Interactive Selection

Arrow key navigation for agent and script selection:

```
┌─────────────────────────────────────┐
│ Choose your AI assistant:           │
├─────────────────────────────────────┤
│ ▶ claude (Claude Code)              │
│   gemini (Gemini CLI)               │
│   copilot (GitHub Copilot)          │
│   cursor-agent (Cursor)             │
│                                     │
│ Use ↑/↓ to navigate, Enter to select│
└─────────────────────────────────────┘
```

### Step Tracker

Visual progress indicator:

```
Initialize Specify Project
├── ● Check required tools (ok)
├── ● Select AI assistant (claude)
├── ● Select script type (sh)
├── ● Fetch latest release (v1.2.3, 45,678 bytes)
├── ● Download template (spec-kit-template-claude-sh-1.2.3.zip)
├── ● Extract template
├── ● Set script permissions (5 updated)
├── ● Initialize git repository (initialized)
└── ● Finalize (project ready)
```

---

## Output Files

After initialization:

```
project/
├── .specify/
│   ├── memory/
│   │   └── constitution.md
│   ├── scripts/
│   │   ├── bash/
│   │   └── powershell/
│   └── templates/
│       ├── commands/
│       ├── spec-template.md
│       ├── plan-template.md
│       └── tasks-template.md
├── .{agent}/
│   └── commands/          # Slash command files
├── specs/                  # Feature specifications
└── .gitignore
```

---

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `GH_TOKEN` | GitHub API authentication |
| `GITHUB_TOKEN` | GitHub API authentication (alternative) |
| `SPECIFY_FEATURE` | Override feature detection for non-git repos |
| `CODEX_HOME` | Codex CLI configuration directory |

---

## Security Notice

After initialization, the CLI displays:

```
┌────────────────────────────────────────────────┐
│ Agent Folder Security                          │
├────────────────────────────────────────────────┤
│ Some agents may store credentials, auth        │
│ tokens, or other private artifacts in the      │
│ agent folder within your project.              │
│                                                │
│ Consider adding .claude/ (or parts of it)      │
│ to .gitignore to prevent credential leakage.   │
└────────────────────────────────────────────────┘
```

---

## Error Handling

### Directory Conflict

```
┌────────────────────────────────────┐
│ Directory Conflict                 │
├────────────────────────────────────┤
│ Directory 'my-project' already     │
│ exists. Please choose a different  │
│ name or remove the existing        │
│ directory.                         │
└────────────────────────────────────┘
```

### Agent Detection Error

```
┌────────────────────────────────────┐
│ Agent Detection Error              │
├────────────────────────────────────┤
│ claude not found                   │
│ Install from: https://...          │
│                                    │
│ Tip: Use --ignore-agent-tools to   │
│ skip this check                    │
└────────────────────────────────────┘
```

### Rate Limit Error

```
GitHub API returned status 403 for https://...

Rate Limit Information:
  • Rate Limit: 60 requests/hour
  • Remaining: 0
  • Resets at: 2025-01-15 14:30:00 UTC

Troubleshooting Tips:
  • Use --github-token or GH_TOKEN env variable
  • Authenticated requests: 5,000/hour
```
