# Spec-Kit Agent Support

## Supported AI Agents

Spec-Kit supports 16+ AI coding assistants through a unified command system.

### Agent Support Matrix

| Agent | Directory | Format | CLI Tool | Type |
|-------|-----------|--------|----------|------|
| **Claude Code** | `.claude/commands/` | Markdown | `claude` | CLI |
| **Gemini CLI** | `.gemini/commands/` | TOML | `gemini` | CLI |
| **GitHub Copilot** | `.github/agents/` | Markdown | N/A | IDE |
| **Cursor** | `.cursor/commands/` | Markdown | `cursor-agent` | IDE |
| **Qwen Code** | `.qwen/commands/` | TOML | `qwen` | CLI |
| **opencode** | `.opencode/command/` | Markdown | `opencode` | CLI |
| **Codex CLI** | `.codex/commands/` | Markdown | `codex` | CLI |
| **Windsurf** | `.windsurf/workflows/` | Markdown | N/A | IDE |
| **Kilo Code** | `.kilocode/rules/` | Markdown | N/A | IDE |
| **Auggie CLI** | `.augment/rules/` | Markdown | `auggie` | CLI |
| **Roo Code** | `.roo/rules/` | Markdown | N/A | IDE |
| **CodeBuddy CLI** | `.codebuddy/commands/` | Markdown | `codebuddy` | CLI |
| **Qoder CLI** | `.qoder/commands/` | Markdown | `qoder` | CLI |
| **Amazon Q** | `.amazonq/prompts/` | Markdown | `q` | CLI |
| **Amp** | `.agents/commands/` | Markdown | `amp` | CLI |
| **SHAI** | `.shai/commands/` | Markdown | `shai` | CLI |
| **IBM Bob** | `.bob/commands/` | Markdown | N/A | IDE |

---

## Agent Categories

### CLI-Based Agents

Require a command-line tool to be installed:

```python
AGENT_CONFIG = {
    "claude": {
        "name": "Claude Code",
        "folder": ".claude/",
        "install_url": "https://docs.anthropic.com/en/docs/claude-code/setup",
        "requires_cli": True,
    },
    "gemini": {
        "name": "Gemini CLI",
        "folder": ".gemini/",
        "install_url": "https://github.com/google-gemini/gemini-cli",
        "requires_cli": True,
    },
    # ... etc
}
```

### IDE-Based Agents

Work within integrated development environments, no CLI check needed:

```python
"copilot": {
    "name": "GitHub Copilot",
    "folder": ".github/",
    "install_url": None,  # IDE-based
    "requires_cli": False,
},
```

---

## Command File Formats

### Markdown Format

Used by: Claude, Cursor, opencode, Windsurf, Amazon Q, Amp, SHAI, IBM Bob, Codex, CodeBuddy, Qoder

```markdown
---
description: "Command description"
---

Command content with {SCRIPT} and $ARGUMENTS placeholders.
```

**GitHub Copilot Chat Mode**:
```markdown
---
description: "Command description"
mode: speckit.command-name
---

Command content...
```

### TOML Format

Used by: Gemini, Qwen

```toml
description = "Command description"

prompt = """
Command content with {SCRIPT} and {{args}} placeholders.
"""
```

---

## Argument Placeholders

| Agent Type | Arguments Placeholder | Script Placeholder |
|------------|----------------------|-------------------|
| Markdown-based | `$ARGUMENTS` | `{SCRIPT}` |
| TOML-based | `{{args}}` | `{SCRIPT}` |

---

## Agent Configuration (AGENT_CONFIG)

The Python CLI maintains a single source of truth for agent metadata:

```python
AGENT_CONFIG = {
    "claude": {
        "name": "Claude Code",           # Display name
        "folder": ".claude/",            # Agent files directory
        "install_url": "https://...",    # Installation docs
        "requires_cli": True,            # Needs CLI tool check
    },
}
```

### Design Principle: Actual CLI Tool Names as Keys

**Critical**: Always use the **actual executable name** as the dictionary key:

```python
# CORRECT - matches actual CLI tool
"cursor-agent": { ... }

# WRONG - requires special-case mapping
"cursor": { ... }  # CLI is actually "cursor-agent"
```

This eliminates special-case mappings throughout the codebase.

---

## Directory Conventions

| Agent Type | Pattern | Example |
|------------|---------|---------|
| CLI agents | `.<agent>/commands/` | `.claude/commands/` |
| IDE agents | Varies by IDE | `.github/agents/`, `.windsurf/workflows/` |

---

## Adding New Agent Support

### 1. Add to AGENT_CONFIG

```python
AGENT_CONFIG = {
    "new-agent-cli": {
        "name": "New Agent Display Name",
        "folder": ".newagent/",
        "install_url": "https://example.com/install",
        "requires_cli": True,
    },
}
```

### 2. Update CLI Help Text

```python
ai_assistant: str = typer.Option(
    None, "--ai",
    help="AI assistant to use: claude, gemini, ..., new-agent-cli"
)
```

### 3. Update README Documentation

Add to the Supported AI Agents table.

### 4. Update Release Package Script

In `.github/workflows/scripts/create-release-packages.sh`:

```bash
ALL_AGENTS=(claude gemini copilot ... new-agent-cli)

case $agent in
  new-agent-cli)
    mkdir -p "$base_dir/.newagent/commands"
    generate_commands new-agent-cli md "\$ARGUMENTS" "$base_dir/.newagent/commands" "$script"
    ;;
esac
```

### 5. Update Agent Context Scripts

**Bash** (`scripts/bash/update-agent-context.sh`):
```bash
NEW_AGENT_FILE="$REPO_ROOT/.newagent/context.md"

case "$AGENT_TYPE" in
  new-agent-cli) update_agent_file "$NEW_AGENT_FILE" "New Agent" ;;
esac
```

**PowerShell** (`scripts/powershell/update-agent-context.ps1`):
```powershell
$newAgentFile = Join-Path $repoRoot '.newagent/context.md'

switch ($AgentType) {
    'new-agent-cli' { Update-AgentFile $newAgentFile 'New Agent' }
}
```

---

## Special Handling: Claude Migration

Claude CLI has a special case for post-migration installations:

```python
CLAUDE_LOCAL_PATH = Path.home() / ".claude" / "local" / "claude"

if tool == "claude":
    if CLAUDE_LOCAL_PATH.exists() and CLAUDE_LOCAL_PATH.is_file():
        return True  # Found migrated Claude
```

The `claude migrate-installer` command moves the executable from PATH to `~/.claude/local/claude`.

---

## Agent Context Updates

During `/speckit.plan`, the agent context file is updated with technology information:

```markdown
<!-- SPECIFY_CONTEXT_START -->
## Technology Stack

- **Language**: Python 3.11
- **Framework**: FastAPI
- **Database**: PostgreSQL

## Current Feature

- Branch: 001-user-auth
- Spec: specs/001-user-auth/spec.md
<!-- SPECIFY_CONTEXT_END -->
```

Content between markers is auto-updated. Content outside markers is preserved.

---

## Handoffs Between Commands

Commands define suggested next steps via handoffs:

```yaml
handoffs:
  - label: Build Technical Plan
    agent: speckit.plan
    prompt: Create a plan for the spec. I am building with...
  - label: Clarify Spec Requirements
    agent: speckit.clarify
    prompt: Clarify specification requirements
    send: true  # Auto-send on selection
```

This enables a guided workflow where agents suggest (or auto-execute) the next command.

---

## Security Considerations

The CLI warns users about agent folder security:

```
Some agents may store credentials, auth tokens, or other identifying
and private artifacts in the agent folder within your project.

Consider adding .claude/ (or parts of it) to .gitignore to prevent
accidental credential leakage.
```

---

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `SPECIFY_FEATURE` | Override feature detection for non-git repos |
| `CODEX_HOME` | Required for Codex CLI - set to `.codex` path |
| `GH_TOKEN` / `GITHUB_TOKEN` | GitHub API authentication |
