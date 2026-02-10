## What is Fab Kit?

Fab Kit (Fabrication Kit) is a Specification-Driven Development (SDD) workflow kit that runs entirely as AI agent prompts — no CLI installation, no system dependencies. It gives structure to the work developers already do (define, design, build, review, document) by providing named stages, markdown templates, and skill definitions that any AI agent (Claude Code, Cursor, Windsurf, etc.) can execute.

The core ideas:

1. **Pure prompt play** — The entire engine lives in `fab/.kit/` as markdown skill files and templates. You copy the directory into your project and go. No package manager, no binary, no runtime.
2. **Docs as source of truth** — Centralized docs in `fab/docs/` are the authoritative record of what the system does and why. Code changes flow *into* docs (via hydration at archive time), not the other way around.
3. **Change folders as the unit of work** — Each change gets its own folder under `fab/changes/` containing a proposal, spec, plan, tasks, and quality checklist. Git integration is optional and informational — Fab never touches branches, commits, or pushes.
4. **7 stages, 5 user-facing commands** — Internally there are 7 stages (proposal, specs, plan, tasks, apply, review, archive), but the user mostly interacts through `/fab-new`, `/fab-continue` (or `/fab-ff` to fast-forward), `/fab-apply`, `/fab-review`, and `/fab-archive`.
5. **Hybrid lineage** — Cherry-picks from two earlier systems: SpecKit's customizable folder structure, intuitive navigation, and pure-prompt approach, combined with OpenSpec's fast-forward workflow and centralized doc hydration on completion.

The design philosophy leans toward discipline without rigidity — it enforces a structured planning-before-coding workflow with quality checklists and spec validation, but keeps everything lightweight, git-optional, and easy to customize.

## Get Started

Copy the fab/.kit folder to your repo, and run:

```bash
fab-setup.sh #this should already by in your PATH because of .envrc
#Or else, run
fab/.kit/scripts/fab-setup.sh
```



## Repository Structure

```
sddr/
├── references/
│   ├── speckit/      # Analysis of GitHub's Spec-Kit
│   └── openspec/     # Analysis of Fission AI's OpenSpec
├── fab/              # Fab workflow kit
└── README.md
```

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

The `references/` folder contains docs from other libraries and projects, included purely for reference.

### [references/speckit/](references/speckit/)
Comprehensive analysis of **Spec-Kit** (https://github.com/github/spec-kit) - GitHub's SDD toolkit.
- Start with [README.md](references/speckit/README.md) for overview
- Key docs: philosophy, workflow, commands, templates, agents

### [references/openspec/](references/openspec/)
In-depth analysis of **OpenSpec** (https://github.com/Fission-AI/OpenSpec) - an AI-native spec-driven framework.
- Start with [README.md](references/openspec/README.md) for overview
- Key docs: overview, philosophy, cli-architecture, agent-integration
