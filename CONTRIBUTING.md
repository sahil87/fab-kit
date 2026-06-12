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
6. **[docs/memory/distribution/kit-architecture.md](docs/memory/distribution/kit-architecture.md)** — kit distribution: the system cache (`~/.fab-kit/versions/<version>/kit/`), `fab sync`, deployment
7. **[docs/specs/templates.md](docs/specs/templates.md)** — artifact template system
8. **[docs/memory/pipeline/schemas.md](docs/memory/pipeline/schemas.md)** — pipeline stage schemas and `.status.yaml` development guide

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
| [docs/memory/distribution/setup.md](docs/memory/distribution/setup.md) | `/fab-setup` — structural bootstrap: creates config.yaml, constitution.md, directories |
| [docs/memory/_shared/configuration.md](docs/memory/_shared/configuration.md) | `config.yaml` schema and `constitution.md` governance |

#### Concepts

| Document | Description |
|----------|-------------|
| [fab/project/constitution.md](fab/project/constitution.md) | Project principles and constraints — the MUST/SHOULD rules that govern all skills |
| [docs/memory/pipeline/change-lifecycle.md](docs/memory/pipeline/change-lifecycle.md) | Change folders, `.status.yaml`, naming conventions, git integration, `/fab-status`, `/fab-switch` |
| [docs/memory/_shared/context-loading.md](docs/memory/_shared/context-loading.md) | How skills load project context — always-load layer, selective domain loading, SRAD protocol |
| [docs/memory/memory-docs/hydrate.md](docs/memory/memory-docs/hydrate.md) | `/docs-hydrate-memory` — dual-mode: ingest external sources or generate docs from codebase scanning |
| [docs/memory/memory-docs/specs-index.md](docs/memory/memory-docs/specs-index.md) | `docs/specs/` directory — pre-implementation specs, distinction from docs |

#### Reference

| Document | Description |
|----------|-------------|
| [docs/specs/skills.md](docs/specs/skills.md) | Detailed behavioral specification for each `/fab-*` skill |
| [docs/memory/pipeline/planning-skills.md](docs/memory/pipeline/planning-skills.md) | `/fab-new`, `/fab-discuss`, `/fab-continue`, `/fab-ff`, `/fab-clarify` — the planning pipeline |
| [docs/memory/pipeline/clarify.md](docs/memory/pipeline/clarify.md) | `/fab-clarify` — dual modes (suggest/auto), taxonomy scan, structured questions |
| [docs/memory/pipeline/execution-skills.md](docs/memory/pipeline/execution-skills.md) | Apply, review, archive behavior — accessed via '/fab-continue' |
| [docs/memory/memory-docs/hydrate-specs.md](docs/memory/memory-docs/hydrate-specs.md) | `/docs-hydrate-specs` — structural gap detection between memory and specs |
| [docs/specs/templates.md](docs/specs/templates.md) | Artifact templates — intake, plan (requirements + tasks + acceptance), status |
| [docs/memory/memory-docs/templates.md](docs/memory/memory-docs/templates.md) | Template implementation details and centralized doc format |

#### Internals

| Document | Description |
|----------|-------------|
| [docs/specs/architecture.md](docs/specs/architecture.md) | Directory structure, config schema, naming conventions, agent integration |
| [docs/memory/distribution/kit-architecture.md](docs/memory/distribution/kit-architecture.md) | Kit distribution — system cache (`~/.fab-kit/versions/<version>/kit/`), `fab sync` deployment, agent integration |
| [docs/memory/pipeline/preflight.md](docs/memory/pipeline/preflight.md) | `fab preflight` — validation, structured YAML output, skill integration |
| [docs/memory/memory-docs/hydrate-generate.md](docs/memory/memory-docs/hydrate-generate.md) | `/docs-hydrate-memory` generate mode — codebase scanning, gap detection, doc generation |

### Index Files

| Index | What it covers |
|-------|---------------|
| [docs/specs/index.md](docs/specs/index.md) | Pre-implementation specifications (design intent) |
| [docs/memory/index.md](docs/memory/index.md) | Post-implementation centralized docs (what actually shipped) |
| [docs/memory/index.md](docs/memory/index.md) | Domain index — links to each domain (pipeline, memory-docs, distribution, runtime, _shared); per-domain indexes list files with last-updated dates |

## Stage State & Queries (fab CLI)

Stage state lives in each change's `.status.yaml` and is owned by the **`fab` Go binary** (source in `src/go/fab/`). The pipeline is six stages: `intake → apply → review → hydrate → ship → review-pr`.

```bash
# Resolve + summarize the active change (YAML: id, name, change_dir, stage, display_stage, display_state, progress, plan, confidence)
fab preflight

# Stage queries
fab status all-stages                     # List stage IDs in order
fab status progress-map <change>          # stage:state pairs
fab status current-stage <change>         # Detect active stage
fab status validate-status-file <change>  # Validate .status.yaml against the schema

# Stage transitions (state machine)
fab status start|advance|finish|reset|skip|fail <change> <stage>
```

**Development & Testing:**

```bash
# Run the Go test suite
cd src/go/fab && go test ./...
```

For complete documentation, see:
- [docs/memory/pipeline/schemas.md](docs/memory/pipeline/schemas.md) — stage/state schemas and `.status.yaml` reference
- [src/kit/skills/_cli-fab.md](src/kit/skills/_cli-fab.md) — the full `fab` CLI reference (updated with every CLI change, per the constitution)

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
3. Create a GitHub Release with `kit.tar.gz` as an asset

**Requires**: clean working tree, [gh CLI](https://cli.github.com/), and a configured `origin` remote.
