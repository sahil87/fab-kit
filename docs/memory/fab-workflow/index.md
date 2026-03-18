# Fab Workflow Documentation

| File | Description | Last Updated |
|-----|-------------|-------------|
| [hydrate](hydrate.md) | `/docs-hydrate-memory` skill — argument routing, dual-mode (ingest + generate), hydration rules, index maintenance | 2026-02-14 |
| [hydrate-generate](hydrate-generate.md) | `/docs-hydrate-memory` generate mode — codebase scanning, gap detection, interactive scoping, memory file generation | 2026-02-07 |
| [setup](setup.md) | `/fab-setup` skill — structural bootstrap, subcommand architecture (config, constitution, migrations), delegation pattern with `fab-sync.sh` | 2026-02-16 |
| [context-loading](context-loading.md) | Smart context loading convention — 7-file always-load layer, standard subagent context, selective domain loading, SRAD protocol, state-keyed Next Steps Convention | 2026-03-18 |
| [planning-skills](planning-skills.md) | `/fab-new`, `/fab-continue`, `/fab-ff`, `/fab-clarify` — the planning pipeline from intake through tasks, shared `_generation.md` partial | 2026-02-16 |
| [clarify](clarify.md) | `/fab-clarify` skill — dual modes (suggest/auto), taxonomy scan, structured questions, coverage reports, audit trail, grade reclassification | 2026-02-16 |
| [execution-skills](execution-skills.md) | Apply, review, hydrate, archive, and operator behavior — `/fab-continue` for pipeline stages, `/fab-archive` for housekeeping, `/fab-operator4` for cross-agent coordination, `/fab-operator5` for use case registry + branch fallback | 2026-03-18 |
| [change-lifecycle](change-lifecycle.md) | Change naming, folder structure, `.status.yaml`, `.fab-status.yaml` symlink, git integration, `/fab-status`, `/fab-switch`, backlog scanning | 2026-03-07 |
| [templates](templates.md) | Artifact templates (intake, spec, tasks, checklist), skill frontmatter, and memory file format | 2026-02-27 |
| [distribution](distribution.md) | How `fab/.kit/` is distributed — bootstrap, update, release workflow | 2026-03-10 |
| [kit-architecture](kit-architecture.md) | `.kit/` structure, scripts, agent integration, distribution, versioning, monorepos, underscore file ecosystem | 2026-03-15 |
| [model-tiers](model-tiers.md) | Provider-agnostic model tier system — tier naming, selection criteria, skill audit, config.yaml mapping, copy-with-template deployment | 2026-02-19 |
| [configuration](configuration.md) | `config.yaml` schema, companion files (`context.md`, `code-quality.md`, `code-review.md`), `constitution.md` governance, 5 Cs of Quality, lifecycle management | 2026-02-27 |
| [preflight](preflight.md) | `lib/preflight.sh` script — validation, accessor-based architecture, structured YAML output, skill integration | 2026-02-14 |
| [migrations](migrations.md) | Migration system — dual-version model, migration file format, `/fab-setup migrations` subcommand, version drift detection, `fab/.kit-migration-version` creation | 2026-02-16 |
| [hydrate-specs](hydrate-specs.md) | `/docs-hydrate-specs` skill — structural gap detection between memory and specs, interactive propose-then-apply | 2026-02-14 |
| [specs-index](specs-index.md) | `docs/specs/` directory — pre-implementation specs, distinction from memory, bootstrap and context integration | 2026-02-14 |
| [schemas](schemas.md) | `workflow.yaml` schema — stages, states, transitions, validation rules, design principles | 2026-02-12 |
