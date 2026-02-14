# Fab Workflow Documentation

| File | Description | Last Updated |
|-----|-------------|-------------|
| [hydrate](hydrate.md) | `/docs-hydrate-memory` skill — argument routing, dual-mode (ingest + generate), hydration rules, index maintenance | 2026-02-14 |
| [hydrate-generate](hydrate-generate.md) | `/docs-hydrate-memory` generate mode — codebase scanning, gap detection, interactive scoping, memory file generation | 2026-02-07 |
| [init](init.md) | `/fab-init` skill — structural bootstrap, subcommand architecture (config, constitution, validate), delegation pattern with `lib/init-scaffold.sh` | 2026-02-14 |
| [context-loading](context-loading.md) | Smart context loading convention — always-load layer, selective domain loading, SRAD protocol | 2026-02-14 |
| [planning-skills](planning-skills.md) | `/fab-new`, `/fab-continue`, `/fab-ff`, `/fab-clarify` — the planning pipeline from brief through tasks, shared `_generation.md` partial | 2026-02-14 |
| [clarify](clarify.md) | `/fab-clarify` skill — dual modes (suggest/auto), taxonomy scan, structured questions, coverage reports, audit trail, grade reclassification | 2026-02-12 |
| [execution-skills](execution-skills.md) | Apply, review, hydrate, and archive behavior — `/fab-continue` for pipeline stages, `/fab-archive` for housekeeping | 2026-02-13 |
| [change-lifecycle](change-lifecycle.md) | Change naming, folder structure, `.status.yaml`, `fab/current`, git integration, `/fab-status`, `/fab-switch`, backlog scanning | 2026-02-14 |
| [templates](templates.md) | Artifact templates (brief, spec, tasks, checklist), skill frontmatter, and memory file format | 2026-02-14 |
| [distribution](distribution.md) | How `fab/.kit/` is distributed — bootstrap, update, release workflow | 2026-02-12 |
| [kit-architecture](kit-architecture.md) | `.kit/` structure, scripts, agent integration, distribution, versioning, monorepos | 2026-02-14 |
| [model-tiers](model-tiers.md) | Provider-agnostic model tier system — tier naming, selection criteria, skill audit, mapping, dual deployment | 2026-02-12 |
| [configuration](configuration.md) | `config.yaml` schema, `constitution.md` governance, stage graph, lifecycle management (updates, amendments, validation) | 2026-02-14 |
| [preflight](preflight.md) | `lib/preflight.sh` script — validation, accessor-based architecture, structured YAML output, skill integration | 2026-02-14 |
| [migrations](migrations.md) | Migration system — dual-version model, migration file format, `/fab-update` skill, version drift detection, `fab/VERSION` creation | 2026-02-14 |
| [hydrate-specs](hydrate-specs.md) | `/docs-hydrate-specs` skill — structural gap detection between memory and specs, interactive propose-then-apply | 2026-02-14 |
| [specs-index](specs-index.md) | `docs/specs/` directory — pre-implementation specs, distinction from memory, bootstrap and context integration | 2026-02-14 |
| [schemas](schemas.md) | `workflow.yaml` schema — stages, states, transitions, validation rules, design principles | 2026-02-12 |
