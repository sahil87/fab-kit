# Fab Kit

Fab Kit is a Specification-Driven Development (SDD) workflow kit that runs entirely as AI agent prompts — no CLI installation, no system dependencies. It provides named stages, markdown templates, and skill definitions that any AI agent (Claude Code, Cursor, Windsurf, etc.) can execute.

The core engine lives in `fab/.kit/` as markdown skill files, templates, and shell scripts. You copy it into your project and go.

## Quick Start

### Bootstrap a new project

**For private repository access** (requires [gh CLI](https://cli.github.com/) with authentication):

```bash
mkdir -p fab
gh release download --repo wvrdz/fab-kit --pattern 'kit.tar.gz' --output - | tar xz -C fab/
```

Alternatively, if you have a local clone:

```bash
cp -r /path/to/fab-kit/fab/.kit ./fab/
```

Then run setup and init:

```bash
fab/.kit/scripts/lib/init-scaffold.sh   # creates directories, symlinks, .gitignore
direnv allow # To approve .envrc content, used to add scripts to path
# Once setup completes, use your AI agent to run:
#> /fab-init     # generates config.yaml and constitution.md
```

## Working with FabKit

To start a new change, open your AI Agent and run:

```bash
#> /fab-new      # starts your first change
```

From there, the agent will inform you the following instructions.

## Updating

Upgrading is a two-step process: first update the engine (`fab/.kit/`), then apply any migrations to your project files.

### Step 1: Upgrade the engine

```bash
fab-upgrade.sh
# bash fab/.kit/scripts/fab-upgrade.sh
```

This will:
1. Download the latest `kit.tar.gz` from GitHub Releases
2. Atomically replace `fab/.kit/` (your `config.yaml`, `docs/`, `changes/`, etc. are never touched)
3. Display the version change (e.g., "0.1.0 → 0.2.0")
4. Re-run `lib/init-scaffold.sh` to repair symlinks

**Requires**: [gh CLI](https://cli.github.com/) installed and authenticated.

### Step 2: Apply migrations

If `fab-upgrade.sh` reports that `fab/VERSION` is behind the engine version, run:

```bash
#> /fab-update
```

This will:
1. Compare `fab/VERSION` (your project version) to `fab/.kit/VERSION` (engine version)
2. Apply any migration scripts found in `fab/.kit/migrations/` sequentially
3. Update `fab/VERSION` to match the engine

Safe to re-run — it only applies migrations that haven't been applied yet.

### Check your versions

```bash
cat fab/.kit/VERSION   # engine version
cat fab/VERSION        # project version (should match after /fab-update)
```

## Creating a Release

For maintainers of this repo — to publish a new release:

```bash
fab-release.sh [patch|minor|major]
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
├── scripts/         # Shell utilities (setup, status, update, release, stageman)
└── schemas/         # Workflow schema and documentation
```

The kit provides a 6-stage workflow: **brief → spec → tasks → apply → review → hydrate**. See [docs/specs/index.md](docs/specs/index.md) for the full specification.

### Stage Manager (`stageman.sh`)

The kit includes **Stage Manager**, a bash utility for querying workflow stages and states:

```bash
# Query utility
fab/.kit/scripts/lib/stageman.sh --help     # Show all available functions
fab/.kit/scripts/lib/stageman.sh --version  # Show version
fab/.kit/scripts/lib/stageman.sh --test     # Run self-tests

# Use in scripts
source fab/.kit/scripts/lib/stageman.sh
get_all_stages                          # List stages
get_stage_number "spec"                 # Get position (2)
validate_status_file .status.yaml       # Validate change status
```

**Development & Testing:**

```bash
# Run basic tests
src/lib/stageman/test-simple.sh

# View API documentation and development guide
cat src/lib/stageman/README.md
```

For complete documentation, see:
- [docs/memory/fab-workflow/schemas.md](docs/memory/fab-workflow/schemas.md) - Schema overview
- [src/lib/stageman/README.md](src/lib/stageman/README.md) - API reference and development guide

## Documentation Map

> **New to Fab Kit?** Start with the reading path for your role below, then use the inventory to find specific docs. For terminology, see the **[Glossary](docs/specs/glossary.md)**.

### Reading Paths

#### New User — "I want to use Fab Kit in my project"

1. **[This README](#what-is-fab-kit)** — what Fab Kit is, core ideas, setup
2. **[docs/specs/overview.md](docs/specs/overview.md)** — the 7-stage workflow, design principles, quick command reference
3. **[docs/specs/user-flow.md](docs/specs/user-flow.md)** — visual diagrams of how commands connect
4. **[docs/specs/glossary.md](docs/specs/glossary.md)** — all terminology defined in one place
5. **[docs/memory/fab-workflow/init.md](docs/memory/fab-workflow/init.md)** — how `/fab-init` bootstraps your project
6. **[docs/memory/fab-workflow/change-lifecycle.md](docs/memory/fab-workflow/change-lifecycle.md)** — how changes work: folders, naming, status tracking

#### Contributor — "I want to modify or extend Fab Kit"

1. **[docs/specs/overview.md](docs/specs/overview.md)** — workflow design and principles (prerequisite for everything)
2. **[docs/specs/glossary.md](docs/specs/glossary.md)** — terminology you'll see everywhere
3. **[fab/constitution.md](fab/constitution.md)** — immutable project principles (MUST/SHOULD rules)
4. **[docs/specs/architecture.md](docs/specs/architecture.md)** — directory structure, config, naming, agent integration
5. **[docs/specs/skills.md](docs/specs/skills.md)** — detailed behavior for each `/fab-*` skill
6. **[docs/memory/fab-workflow/kit-architecture.md](docs/memory/fab-workflow/kit-architecture.md)** — `.kit/` internals, scripts, distribution
7. **[docs/specs/templates.md](docs/specs/templates.md)** — artifact template system
8. **[src/lib/stageman/README.md](src/lib/stageman/README.md)** — Stage Manager development guide and testing

#### Spec Reader — "I want to understand the design rationale"

1. **[docs/specs/glossary.md](docs/specs/glossary.md)** — read this first to understand the vocabulary
2. **[docs/specs/overview.md](docs/specs/overview.md)** — high-level design, principles, stage definitions
3. **[docs/specs/proposal.md](docs/specs/proposal.md)** — original SpecKit vs OpenSpec comparison and design rationale
4. **[docs/specs/architecture.md](docs/specs/architecture.md)** — structural decisions and conventions
5. **[docs/specs/skills.md](docs/specs/skills.md)** — skill-by-skill behavioral specification
6. **[docs/specs/templates.md](docs/specs/templates.md)** — template design and field semantics
7. **[docs/specs/user-flow.md](docs/specs/user-flow.md)** — visual command flow diagrams

### Document Inventory

#### Getting Started

| Document | Description |
|----------|-------------|
| [docs/specs/overview.md](docs/specs/overview.md) | The Fab workflow specification — design principles, 7 stages, quick command reference |
| [docs/specs/user-flow.md](docs/specs/user-flow.md) | Visual diagrams showing how commands connect and how a typical development session flows |
| [docs/specs/glossary.md](docs/specs/glossary.md) | All Fab terminology — core concepts, stages, skills, files, SRAD, conventions |
| [docs/memory/fab-workflow/init.md](docs/memory/fab-workflow/init.md) | `/fab-init` — structural bootstrap: creates config.yaml, constitution.md, directories |
| [docs/memory/fab-workflow/configuration.md](docs/memory/fab-workflow/configuration.md) | `config.yaml` schema and `constitution.md` governance |

#### Concepts

| Document | Description |
|----------|-------------|
| [fab/constitution.md](fab/constitution.md) | Project principles and constraints — the MUST/SHOULD rules that govern all skills |
| [docs/memory/fab-workflow/change-lifecycle.md](docs/memory/fab-workflow/change-lifecycle.md) | Change folders, `.status.yaml`, naming conventions, git integration, `/fab-status`, `/fab-switch` |
| [docs/memory/fab-workflow/context-loading.md](docs/memory/fab-workflow/context-loading.md) | How skills load project context — always-load layer, selective domain loading, SRAD protocol |
| [docs/memory/fab-workflow/hydrate.md](docs/memory/fab-workflow/hydrate.md) | `/docs-hydrate-memory` — dual-mode: ingest external sources or generate docs from codebase scanning |
| [docs/memory/fab-workflow/specs-index.md](docs/memory/fab-workflow/specs-index.md) | `docs/specs/` directory — pre-implementation specs, distinction from docs |

#### Reference

| Document | Description |
|----------|-------------|
| [docs/specs/skills.md](docs/specs/skills.md) | Detailed behavioral specification for each `/fab-*` skill |
| [docs/memory/fab-workflow/planning-skills.md](docs/memory/fab-workflow/planning-skills.md) | `/fab-new`, `/fab-discuss`, `/fab-continue`, `/fab-ff`, `/fab-clarify` — the planning pipeline |
| [docs/memory/fab-workflow/clarify.md](docs/memory/fab-workflow/clarify.md) | `/fab-clarify` — dual modes (suggest/auto), taxonomy scan, structured questions |
| [docs/memory/fab-workflow/execution-skills.md](docs/memory/fab-workflow/execution-skills.md) | Apply, review, archive behavior — accessed via '/fab-continue' |
| [docs/memory/fab-workflow/hydrate-specs.md](docs/memory/fab-workflow/hydrate-specs.md) | `/docs-hydrate-specs` — structural gap detection between memory and specs |
| [docs/specs/templates.md](docs/specs/templates.md) | Artifact templates — proposal, spec, plan, tasks, checklist |
| [docs/memory/fab-workflow/templates.md](docs/memory/fab-workflow/templates.md) | Template implementation details and centralized doc format |

#### Internals

| Document | Description |
|----------|-------------|
| [docs/specs/architecture.md](docs/specs/architecture.md) | Directory structure, config schema, naming conventions, agent integration |
| [docs/memory/fab-workflow/kit-architecture.md](docs/memory/fab-workflow/kit-architecture.md) | `.kit/` directory structure, shell scripts, agent integration, distribution |
| [docs/memory/fab-workflow/preflight.md](docs/memory/fab-workflow/preflight.md) | `preflight.sh` — validation script, structured YAML output, skill integration |
| [docs/memory/fab-workflow/hydrate-generate.md](docs/memory/fab-workflow/hydrate-generate.md) | `/docs-hydrate-memory` generate mode — codebase scanning, gap detection, doc generation |
| [docs/specs/proposal.md](docs/specs/proposal.md) | Original SpecKit vs OpenSpec comparison and design rationale |

### Index Files

These are the structural indexes for navigating within each documentation area:

| Index | What it covers |
|-------|---------------|
| [docs/specs/index.md](docs/specs/index.md) | Pre-implementation specifications (design intent) |
| [docs/memory/index.md](docs/memory/index.md) | Post-implementation centralized docs (what actually shipped) |
| [docs/memory/fab-workflow/index.md](docs/memory/fab-workflow/index.md) | All fab-workflow domain docs with last-updated dates |

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

### Why Fab Kit?

- **Resumable by design** — Every stage produces a persistent artifact. Walk away mid-change, come back tomorrow, and pick up exactly where you left off.
- **Stages that don't get skipped** — Brief, spec, plan, tasks, apply, review, hydrate. The pipeline encodes the discipline so the agent (and you) can't quietly skip straight to code.
- **Fast-forward when confidence is high** — `/fab-ff` and `/fab-fff` let you blast through multiple stages in one shot when the change is well-understood, without sacrificing the structure when it isn't.
- **Deterministic progress tracking** — `.status.yaml` and stage checklists give you a single source of truth for where a change stands — no guessing, no stale mental models.
