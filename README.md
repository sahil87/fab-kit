# Fab Kit

Fab Kit is a Specification-Driven Development (SDD) workflow kit that runs entirely as AI agent prompts — no CLI installation, no system dependencies. It provides named stages, markdown templates, and skill definitions that any AI agent (Claude Code, Cursor, Windsurf, etc.) can execute.

The core engine lives in `fab/.kit/` as markdown skill files, templates, and shell scripts. You copy it into your project and go.

## Quick Start

### Bootstrap a new project

```bash
mkdir -p fab
curl -sL https://github.com/wvrdz/fab-kit/releases/latest/download/kit.tar.gz | tar xz -C fab/
```

Then run setup and init:

```bash
fab/.kit/scripts/fab-setup.sh   # creates directories, symlinks, .gitignore
```

Once setup completes, use your AI agent to run:

```
/fab-init     # generates config.yaml and constitution.md
/fab-new      # starts your first change
```

### Alternative: manual copy

If you have a local clone of this repo:

```bash
cp -r fab/.kit /path/to/your-project/fab/.kit
```

Then run `fab-setup.sh` and `/fab-init` as above.

## Updating

To update `fab/.kit/` to the latest release:

```bash
bash fab/.kit/scripts/fab-update.sh
```

This will:
1. Download the latest `kit.tar.gz` from GitHub Releases
2. Atomically replace `fab/.kit/` (your `config.yaml`, `docs/`, `changes/`, etc. are never touched)
3. Display the version change (e.g., "0.1.0 → 0.2.0")
4. Re-run `fab-setup.sh` to repair symlinks

**Requires**: [gh CLI](https://cli.github.com/) installed and authenticated.

### Check your version

```bash
cat fab/.kit/VERSION
```

## Creating a Release

For maintainers of this repo — to publish a new release:

```bash
bash fab/.kit/scripts/fab-release.sh [patch|minor|major]
```

- `patch` (default): 0.1.0 → 0.1.1
- `minor`: 0.1.0 → 0.2.0
- `major`: 0.1.0 → 1.0.0

The script will:
1. Bump the version in `fab/.kit/VERSION`
2. Package `fab/.kit/` into `kit.tar.gz`
3. Commit the VERSION bump
4. Create a GitHub Release with `kit.tar.gz` as an asset

**Requires**: clean working tree, [gh CLI](https://cli.github.com/), and a configured `origin` remote.

## What's in the Box

```
fab/.kit/
├── VERSION          # Semver version string
├── skills/          # Markdown skill definitions for AI agents
├── templates/       # Artifact templates (proposal, spec, plan, tasks, checklist)
└── scripts/         # Shell utilities (setup, status, update, release)
```

The kit provides a 7-stage workflow: **proposal → specs → plan → tasks → apply → review → archive**. See [fab/specs/index.md](fab/specs/index.md) for the full specification.

## Documentation Map

> **New to Fab Kit?** Start with the reading path for your role below, then use the inventory to find specific docs. For terminology, see the **[Glossary](fab/specs/glossary.md)**.

### Reading Paths

#### New User — "I want to use Fab Kit in my project"

1. **[This README](#what-is-fab-kit)** — what Fab Kit is, core ideas, setup
2. **[fab/specs/overview.md](fab/specs/overview.md)** — the 7-stage workflow, design principles, quick command reference
3. **[fab/specs/user-flow.md](fab/specs/user-flow.md)** — visual diagrams of how commands connect
4. **[fab/specs/glossary.md](fab/specs/glossary.md)** — all terminology defined in one place
5. **[fab/docs/fab-workflow/init.md](fab/docs/fab-workflow/init.md)** — how `/fab-init` bootstraps your project
6. **[fab/docs/fab-workflow/change-lifecycle.md](fab/docs/fab-workflow/change-lifecycle.md)** — how changes work: folders, naming, status tracking

#### Contributor — "I want to modify or extend Fab Kit"

1. **[fab/specs/overview.md](fab/specs/overview.md)** — workflow design and principles (prerequisite for everything)
2. **[fab/specs/glossary.md](fab/specs/glossary.md)** — terminology you'll see everywhere
3. **[fab/constitution.md](fab/constitution.md)** — immutable project principles (MUST/SHOULD rules)
4. **[fab/specs/architecture.md](fab/specs/architecture.md)** — directory structure, config, naming, agent integration
5. **[fab/specs/skills.md](fab/specs/skills.md)** — detailed behavior for each `/fab-*` skill
6. **[fab/docs/fab-workflow/kit-architecture.md](fab/docs/fab-workflow/kit-architecture.md)** — `.kit/` internals, scripts, distribution
7. **[fab/specs/templates.md](fab/specs/templates.md)** — artifact template system

#### Spec Reader — "I want to understand the design rationale"

1. **[fab/specs/glossary.md](fab/specs/glossary.md)** — read this first to understand the vocabulary
2. **[fab/specs/overview.md](fab/specs/overview.md)** — high-level design, principles, stage definitions
3. **[fab/specs/proposal.md](fab/specs/proposal.md)** — original SpecKit vs OpenSpec comparison and design rationale
4. **[fab/specs/architecture.md](fab/specs/architecture.md)** — structural decisions and conventions
5. **[fab/specs/skills.md](fab/specs/skills.md)** — skill-by-skill behavioral specification
6. **[fab/specs/templates.md](fab/specs/templates.md)** — template design and field semantics
7. **[fab/specs/user-flow.md](fab/specs/user-flow.md)** — visual command flow diagrams

### Document Inventory

#### Getting Started

| Document | Description |
|----------|-------------|
| [fab/specs/overview.md](fab/specs/overview.md) | The Fab workflow specification — design principles, 7 stages, quick command reference |
| [fab/specs/user-flow.md](fab/specs/user-flow.md) | Visual diagrams showing how commands connect and how a typical development session flows |
| [fab/specs/glossary.md](fab/specs/glossary.md) | All Fab terminology — core concepts, stages, skills, files, SRAD, conventions |
| [fab/docs/fab-workflow/init.md](fab/docs/fab-workflow/init.md) | `/fab-init` — structural bootstrap: creates config.yaml, constitution.md, directories |
| [fab/docs/fab-workflow/configuration.md](fab/docs/fab-workflow/configuration.md) | `config.yaml` schema and `constitution.md` governance |

#### Concepts

| Document | Description |
|----------|-------------|
| [fab/constitution.md](fab/constitution.md) | Project principles and constraints — the MUST/SHOULD rules that govern all skills |
| [fab/docs/fab-workflow/change-lifecycle.md](fab/docs/fab-workflow/change-lifecycle.md) | Change folders, `.status.yaml`, naming conventions, git integration, `/fab-status`, `/fab-switch` |
| [fab/docs/fab-workflow/context-loading.md](fab/docs/fab-workflow/context-loading.md) | How skills load project context — always-load layer, selective domain loading, SRAD protocol |
| [fab/docs/fab-workflow/hydrate.md](fab/docs/fab-workflow/hydrate.md) | `/fab-hydrate` — dual-mode: ingest external sources or generate docs from codebase scanning |
| [fab/docs/fab-workflow/specs-index.md](fab/docs/fab-workflow/specs-index.md) | `fab/specs/` directory — pre-implementation specs, distinction from docs |

#### Reference

| Document | Description |
|----------|-------------|
| [fab/specs/skills.md](fab/specs/skills.md) | Detailed behavioral specification for each `/fab-*` skill |
| [fab/docs/fab-workflow/planning-skills.md](fab/docs/fab-workflow/planning-skills.md) | `/fab-new`, `/fab-discuss`, `/fab-continue`, `/fab-ff`, `/fab-clarify` — the planning pipeline |
| [fab/docs/fab-workflow/clarify.md](fab/docs/fab-workflow/clarify.md) | `/fab-clarify` — dual modes (suggest/auto), taxonomy scan, structured questions |
| [fab/docs/fab-workflow/execution-skills.md](fab/docs/fab-workflow/execution-skills.md) | `/fab-apply`, `/fab-review`, `/fab-archive` — implementation, validation, completion |
| [fab/docs/fab-workflow/backfill.md](fab/docs/fab-workflow/backfill.md) | `/fab-backfill` — structural gap detection between docs and specs |
| [fab/specs/templates.md](fab/specs/templates.md) | Artifact templates — proposal, spec, plan, tasks, checklist |
| [fab/docs/fab-workflow/templates.md](fab/docs/fab-workflow/templates.md) | Template implementation details and centralized doc format |

#### Internals

| Document | Description |
|----------|-------------|
| [fab/specs/architecture.md](fab/specs/architecture.md) | Directory structure, config schema, naming conventions, agent integration |
| [fab/docs/fab-workflow/kit-architecture.md](fab/docs/fab-workflow/kit-architecture.md) | `.kit/` directory structure, shell scripts, agent integration, distribution |
| [fab/docs/fab-workflow/preflight.md](fab/docs/fab-workflow/preflight.md) | `fab-preflight.sh` — validation script, structured YAML output, skill integration |
| [fab/docs/fab-workflow/hydrate-generate.md](fab/docs/fab-workflow/hydrate-generate.md) | `/fab-hydrate` generate mode — codebase scanning, gap detection, doc generation |
| [fab/specs/proposal.md](fab/specs/proposal.md) | Original SpecKit vs OpenSpec comparison and design rationale |

### Index Files

These are the structural indexes for navigating within each documentation area:

| Index | What it covers |
|-------|---------------|
| [fab/specs/index.md](fab/specs/index.md) | Pre-implementation specifications (design intent) |
| [fab/docs/index.md](fab/docs/index.md) | Post-implementation centralized docs (what actually shipped) |
| [fab/docs/fab-workflow/index.md](fab/docs/fab-workflow/index.md) | All fab-workflow domain docs with last-updated dates |

## References

The `references/` folder contains docs from other projects, included for reference:

### [references/speckit/](references/speckit/)
Comprehensive analysis of **Spec-Kit** (https://github.com/github/spec-kit) - GitHub's SDD toolkit.
- Start with [README.md](references/speckit/README.md) for overview
- Key docs: philosophy, workflow, commands, templates, agents

### [references/openspec/](references/openspec/)
In-depth analysis of **OpenSpec** (https://github.com/Fission-AI/OpenSpec) - an AI-native spec-driven framework.
- Start with [README.md](references/openspec/README.md) for overview
- Key docs: overview, philosophy, cli-architecture, agent-integration

### Advantages

- Resumability from any stage
- Everyone remembers that there is a spec stage, a planning stage, and a task stage. And they don't get skipped. 
- FF is cool
- Determinism is calculated
