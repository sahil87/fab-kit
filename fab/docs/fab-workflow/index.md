# Fab Workflow Documentation

| Doc | Description | Last Updated |
|-----|-------------|-------------|
| [hydrate](hydrate.md) | `/fab-hydrate` skill — argument routing, dual-mode (ingest + generate), hydration rules, index maintenance | 2026-02-08 |
| [hydrate-generate](hydrate-generate.md) | `/fab-hydrate` generate mode — codebase scanning, gap detection, interactive scoping, doc generation | 2026-02-07 |
| [init](init.md) | `/fab-init` skill — structural bootstrap only, no source hydration, delegation pattern with `fab-setup.sh` | 2026-02-12 |
| [context-loading](context-loading.md) | Smart context loading convention — always-load layer, selective domain loading, SRAD protocol | 2026-02-08 |
| [planning-skills](planning-skills.md) | `/fab-new`, `/fab-continue`, `/fab-ff`, `/fab-clarify` — the planning pipeline from brief through tasks, shared `_generation.md` partial | 2026-02-12 |
| [clarify](clarify.md) | `/fab-clarify` skill — dual modes (suggest/auto), taxonomy scan, structured questions, coverage reports, audit trail, grade reclassification | 2026-02-12 |
| [execution-skills](execution-skills.md) | Apply, review, and archive behavior — now accessed via `/fab-continue` | 2026-02-12 |
| [change-lifecycle](change-lifecycle.md) | Change naming, folder structure, `.status.yaml`, `fab/current`, git integration, `/fab-status`, `/fab-switch`, backlog scanning | 2026-02-13 |
| [templates](templates.md) | Artifact templates (brief, spec, tasks, checklist), skill frontmatter, and centralized doc format | 2026-02-12 |
| [distribution](distribution.md) | How `fab/.kit/` is distributed — bootstrap, update, release workflow | 2026-02-12 |
| [kit-architecture](kit-architecture.md) | `.kit/` structure, scripts, agent integration, distribution, versioning, monorepos | 2026-02-12 |
| [model-tiers](model-tiers.md) | Provider-agnostic model tier system — tier naming, selection criteria, skill audit, mapping, dual deployment | 2026-02-12 |
| [init-family](init-family.md) | Init command family — `/fab-init-constitution`, `/fab-init-config`, `/fab-init-validate`, delegation from `/fab-init` | 2026-02-12 |
| [config-management](config-management.md) | Maintaining `config.yaml` over project lifecycle — interactive updates, validation, comment preservation | 2026-02-12 |
| [constitution-governance](constitution-governance.md) | Constitutional amendment workflow — semantic versioning, structural rules, audit trail | 2026-02-12 |
| [configuration](configuration.md) | `config.yaml` schema, `constitution.md` governance, stage graph definition, lifecycle management | 2026-02-12 |
| [preflight](preflight.md) | `fab-preflight.sh` script — validation, structured YAML output, skill integration | 2026-02-07 |
| [hydrate-design](hydrate-design.md) | `/fab-hydrate-design` skill — structural gap detection between docs and design, interactive propose-then-apply | 2026-02-09 |
| [design-index](design-index.md) | `fab/design/` directory — pre-implementation design, distinction from docs, bootstrap and context integration | 2026-02-09 |
| [schemas](schemas.md) | `workflow.yaml` schema — stages, states, transitions, validation rules, design principles | 2026-02-12 |
