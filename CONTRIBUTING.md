# Contributing to Fab Kit

## Documentation Structure

Fab Kit's documentation is split into two categories:

- **[docs/specs/](docs/specs/index.md)** — Pre-implementation design specs. Human-curated, captures the "why" behind features. Organize however makes sense.
- **[docs/memory/](docs/memory/index.md)** — Post-implementation memory files. AI-maintained via hydration, authoritative source of truth for system behavior.

### Reading Paths

#### Contributor — "I want to modify or extend Fab Kit"

1. **[docs/specs/overview.md](docs/specs/overview.md)** — workflow design and principles (prerequisite for everything)
2. **[docs/specs/glossary.md](docs/specs/glossary.md)** — terminology you'll see everywhere
3. **[fab/project/constitution.md](fab/project/constitution.md)** — immutable project principles (MUST/SHOULD rules)
4. **[docs/specs/architecture.md](docs/specs/architecture.md)** — directory structure, config, naming, agent integration
5. **[docs/specs/skills.md](docs/specs/skills.md)** — detailed behavior for each `/fab-*` skill
6. **[docs/memory/fab-workflow/kit-architecture.md](docs/memory/fab-workflow/kit-architecture.md)** — `.kit/` internals, scripts, distribution
7. **[docs/specs/templates.md](docs/specs/templates.md)** — artifact template system
8. **[src/lib/stageman/README.md](src/lib/stageman/README.md)** — Stage Manager development guide and testing

#### Spec Reader — "I want to understand the design rationale"

1. **[docs/specs/glossary.md](docs/specs/glossary.md)** — read this first to understand the vocabulary
2. **[docs/specs/overview.md](docs/specs/overview.md)** — high-level design, principles, stage definitions
3. **[docs/specs/architecture.md](docs/specs/architecture.md)** — structural decisions and conventions
4. **[docs/specs/skills.md](docs/specs/skills.md)** — skill-by-skill behavioral specification
5. **[docs/specs/templates.md](docs/specs/templates.md)** — template design and field semantics
6. **[docs/specs/user-flow.md](docs/specs/user-flow.md)** — visual command flow diagrams

### Document Inventory

#### Getting Started

| Document | Description |
|----------|-------------|
| [docs/specs/overview.md](docs/specs/overview.md) | The Fab workflow specification — design principles, 6 stages, quick command reference |
| [docs/specs/user-flow.md](docs/specs/user-flow.md) | Visual diagrams showing how commands connect and how a typical development session flows |
| [docs/specs/glossary.md](docs/specs/glossary.md) | All Fab terminology — core concepts, stages, skills, files, SRAD, conventions |
| [docs/memory/fab-workflow/init.md](docs/memory/fab-workflow/init.md) | `/fab-init` — structural bootstrap: creates config.yaml, constitution.md, directories |
| [docs/memory/fab-workflow/configuration.md](docs/memory/fab-workflow/configuration.md) | `config.yaml` schema and `constitution.md` governance |

#### Concepts

| Document | Description |
|----------|-------------|
| [fab/project/constitution.md](fab/project/constitution.md) | Project principles and constraints — the MUST/SHOULD rules that govern all skills |
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
| [docs/specs/templates.md](docs/specs/templates.md) | Artifact templates — brief, spec, tasks, checklist |
| [docs/memory/fab-workflow/templates.md](docs/memory/fab-workflow/templates.md) | Template implementation details and centralized doc format |

#### Internals

| Document | Description |
|----------|-------------|
| [docs/specs/architecture.md](docs/specs/architecture.md) | Directory structure, config schema, naming conventions, agent integration |
| [docs/memory/fab-workflow/kit-architecture.md](docs/memory/fab-workflow/kit-architecture.md) | `.kit/` directory structure, shell scripts, agent integration, distribution |
| [docs/memory/fab-workflow/preflight.md](docs/memory/fab-workflow/preflight.md) | `preflight.sh` — validation script, structured YAML output, skill integration |
| [docs/memory/fab-workflow/hydrate-generate.md](docs/memory/fab-workflow/hydrate-generate.md) | `/docs-hydrate-memory` generate mode — codebase scanning, gap detection, doc generation |

### Index Files

| Index | What it covers |
|-------|---------------|
| [docs/specs/index.md](docs/specs/index.md) | Pre-implementation specifications (design intent) |
| [docs/memory/index.md](docs/memory/index.md) | Post-implementation centralized docs (what actually shipped) |
| [docs/memory/fab-workflow/index.md](docs/memory/fab-workflow/index.md) | All fab-workflow domain docs with last-updated dates |

## Stage Manager

The kit includes **Stage Manager** (`stageman.sh`), a bash utility for querying workflow stages and states:

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

## Creating a Release

To publish a new release:

```bash
release.sh [patch|minor|major]
```

- `patch` (default): 0.1.0 → 0.1.1
- `minor`: 0.1.0 → 0.2.0
- `major`: 0.1.0 → 1.0.0

The script will:
1. Bump the version in `src/kit/VERSION`
2. Commit the VERSION bump, tag, and push
3. Commit the VERSION bump
4. Create a GitHub Release with `kit.tar.gz` as an asset

**Requires**: clean working tree, [gh CLI](https://cli.github.com/), and a configured `origin` remote.
