# Spec-Kit Analysis

A comprehensive analysis of GitHub's Spec-Kit: a toolkit for Specification-Driven Development (SDD).

## Repository

- **Source**: https://github.com/github/spec-kit
- **Maintainers**: Den Delimarsky (@localden), John Lam (@jflam)
- **License**: MIT

## What is Spec-Kit?

Spec-Kit is an open-source toolkit that implements **Specification-Driven Development (SDD)** - a methodology where specifications are the primary artifact and code is generated from them. It provides:

1. **Templates** - Structured markdown templates for specs, plans, tasks, and checklists
2. **Slash Commands** - AI agent commands that drive the workflow
3. **Scripts** - Automation scripts (bash/PowerShell) for repository management
4. **CLI Tool** - `specify` Python CLI for project initialization
5. **Multi-Agent Support** - Works with 16+ AI coding assistants

## Documentation Index

| Document | Description |
|----------|-------------|
| [philosophy.md](./philosophy.md) | Core philosophy: SDD methodology, power inversion, intent-driven development |
| [workflow.md](./workflow.md) | The 6-step development workflow from constitution to implementation |
| [commands.md](./commands.md) | All slash commands: specify, plan, tasks, implement, clarify, analyze, checklist |
| [templates.md](./templates.md) | Template system: spec, plan, tasks, checklist, constitution templates |
| [scripts.md](./scripts.md) | Script infrastructure: bash/PowerShell automation |
| [agents.md](./agents.md) | AI agent support: 16+ agents, configuration, integration patterns |
| [constitution.md](./constitution.md) | Constitutional system: principles, governance, enforcement |
| [cli.md](./cli.md) | Specify CLI tool: commands, options, installation |
| [artifacts.md](./artifacts.md) | Generated artifacts: directory structure, file relationships |

## Key Concepts Quick Reference

### The Power Inversion

Traditional development: **Code is king, specs serve code**
SDD: **Specs are king, code serves specs**

### Core Workflow

```
/speckit.constitution → /speckit.specify → /speckit.clarify → /speckit.plan → /speckit.tasks → /speckit.implement
```

### Directory Structure

```
project/
├── .specify/
│   ├── memory/
│   │   └── constitution.md      # Project principles
│   ├── scripts/
│   │   ├── bash/                # POSIX scripts
│   │   └── powershell/          # PowerShell scripts
│   ├── templates/
│   │   ├── commands/            # Slash command definitions
│   │   ├── spec-template.md
│   │   ├── plan-template.md
│   │   └── tasks-template.md
│   └── specs/
│       └── 001-feature-name/
│           ├── spec.md          # Feature specification
│           ├── plan.md          # Implementation plan
│           ├── tasks.md         # Task breakdown
│           ├── research.md      # Technical research
│           ├── data-model.md    # Entity definitions
│           ├── quickstart.md    # Validation scenarios
│           ├── contracts/       # API contracts
│           └── checklists/      # Quality checklists
└── .{agent}/                    # Agent-specific commands
    └── commands/
```

## Technology Stack

- **CLI**: Python 3.11+ with Typer, Rich, httpx
- **Templates**: Markdown with YAML frontmatter
- **Scripts**: Bash and PowerShell dual support
- **Distribution**: GitHub Releases, uv package manager
