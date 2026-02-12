# Fab Workflow Documentation

| Doc | Description | Last Updated |
|-----|-------------|-------------|
| [hydrate](hydrate.md) | `/fab-hydrate` skill — argument routing, dual-mode (ingest + generate), hydration rules, index maintenance | 2026-02-08 |
| [hydrate-generate](hydrate-generate.md) | `/fab-hydrate` generate mode — codebase scanning, gap detection, interactive scoping, doc generation | 2026-02-07 |
| [init](init.md) | `/fab-init` skill — structural bootstrap only, no source hydration | 2026-02-07 |
| [context-loading](context-loading.md) | Smart context loading convention — always-load layer, selective domain loading, SRAD protocol | 2026-02-08 |
| [planning-skills](planning-skills.md) | `/fab-new`, `/fab-continue`, `/fab-ff`, `/fab-clarify` — the planning pipeline from brief through tasks, shared `_generation.md` partial | 2026-02-12 |
| [clarify](clarify.md) | `/fab-clarify` skill — dual modes (suggest/auto), taxonomy scan, structured questions, coverage reports, audit trail | 2026-02-10 |
| [execution-skills](execution-skills.md) | `/fab-apply`, `/fab-review`, `/fab-archive` — implementation, validation, and completion | 2026-02-10 |
| [change-lifecycle](change-lifecycle.md) | Change naming, folder structure, `.status.yaml`, `fab/current`, git integration, `/fab-status`, `/fab-switch` | 2026-02-09 |
| [templates](templates.md) | Artifact templates (brief, spec, tasks, checklist), skill frontmatter, and centralized doc format | 2026-02-12 |
| [distribution](distribution.md) | How `fab/.kit/` is distributed — bootstrap, update, release workflow | 2026-02-10 |
| [kit-architecture](kit-architecture.md) | `.kit/` structure, scripts, agent integration, distribution, versioning, monorepos | 2026-02-12 |
| [model-tiers](model-tiers.md) | Provider-agnostic model tier system — tier naming, selection criteria, skill audit, mapping, dual deployment | 2026-02-12 |
| [configuration](configuration.md) | `config.yaml` schema, `constitution.md` governance, stage graph definition | 2026-02-07 |
| [preflight](preflight.md) | `fab-preflight.sh` script — validation, structured YAML output, skill integration | 2026-02-07 |
| [backfill](backfill.md) | `/fab-backfill` skill — structural gap detection between docs and specs, interactive propose-then-apply | 2026-02-09 |
| [design-index](design-index.md) | `fab/design/` directory — pre-implementation design, distinction from docs, bootstrap and context integration | 2026-02-09 |
