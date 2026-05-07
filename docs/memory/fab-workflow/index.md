# Fab Workflow Documentation

| File | Description | Last Updated |
|-----|-------------|-------------|
| [hydrate](hydrate.md) | `/docs-hydrate-memory` skill — argument routing, dual-mode (ingest + generate), hydration rules, index maintenance | 2026-02-14 |
| [hydrate-generate](hydrate-generate.md) | `/docs-hydrate-memory` generate mode — codebase scanning, gap detection, interactive scoping, memory file generation | 2026-02-07 |
| [setup](setup.md) | `/fab-setup` skill — structural bootstrap, subcommand architecture (config, constitution, migrations), delegation pattern with `fab-kit sync` | 2026-04-02 |
| [context-loading](context-loading.md) | Smart context loading convention — 7-file always-load layer, standard subagent context, selective domain loading, SRAD protocol, state-keyed Next Steps Convention | 2026-04-02 |
| [planning-skills](planning-skills.md) | `/fab-new`, `/fab-continue`, `/fab-ff`, `/fab-clarify` — the planning pipeline (intake, spec) and the shared `_generation.md` partial (Spec + Plan procedures); plan generation lives at apply entry | 2026-05-06 |
| [clarify](clarify.md) | `/fab-clarify` skill — dual modes (suggest/auto), taxonomy scan over intake/spec/plan targets, structured questions, coverage reports, audit trail, grade reclassification | 2026-05-06 |
| [execution-skills](execution-skills.md) | Apply (with plan-generation entry sub-step), review, hydrate, archive, operator, and orchestrator behavior — `/fab-continue` for pipeline stages, `/fab-archive` for housekeeping, `/fab-proceed` for context-aware pipeline orchestration, `/fab-operator` for multi-agent coordination with dependency-aware spawning | 2026-05-06 |
| [change-lifecycle](change-lifecycle.md) | Change naming, folder structure, `.status.yaml` (7-stage pipeline + `plan:` block), `.fab-status.yaml` symlink, git integration, `/fab-status`, `/fab-switch`, backlog scanning | 2026-05-06 |
| [templates](templates.md) | Artifact templates (intake, spec, plan), skill frontmatter, and memory file format. `plan.md` (`## Tasks` + `## Acceptance`) replaces the prior `tasks.md` + `checklist.md` pair | 2026-05-06 |
| [distribution](distribution.md) | How `src/kit/` is distributed — Homebrew formula (2 binaries direct + 2 via `depends_on`), `fab` router, `fab-kit` lifecycle, `fab init` bootstrap, `fab upgrade-repo`, release workflow (3 binaries, 12 cross-compiled), `wt shell-setup` wrapper | 2026-05-06 |
| [kit-architecture](kit-architecture.md) | `src/kit/` structure (binary-free), three-binary architecture (fab router + fab-kit + fab-go), `fab-kit sync`, agent integration, versioning, monorepos, underscore file ecosystem, `fab pane` command group | 2026-04-06 |
| [pane-commands](pane-commands.md) | `fab pane {map,capture,send,process,window-name}` subcommand reference, persistent `--server`/`-L` flag, `WithServer` argv helper, pane-ID-per-server semantics, motivating multi-socket use case, three-axis model (Change / Agent / Process), window-name primitives for idempotent / guarded tmux window rewrites | 2026-04-23 |
| [runtime-agents](runtime-agents.md) | `.fab-runtime.yaml` schema — `_agents[session_id]` keying, hook write/clear pipeline (stop/session-start/user-prompt), throttled GC via `last_run_gc`, grandparent PID walker, pane-map matching rule | 2026-04-19 |
| [model-tiers](model-tiers.md) | Provider-agnostic model tier system — tier naming, selection criteria, skill audit, config.yaml mapping, copy-with-template deployment | 2026-02-19 |
| [configuration](configuration.md) | `config.yaml` schema (incl. `fab_version`, `review_tools`, `true_impact_exclude`), companion files (`context.md`, `code-quality.md`, `code-review.md`), `constitution.md` governance, 5 Cs of Quality, lifecycle management | 2026-05-07 |
| [preflight](preflight.md) | `lib/preflight.sh` script — validation, accessor-based architecture, structured YAML output, skill integration | 2026-04-02 |
| [migrations](migrations.md) | Migration system — dual-version model, migration file format, `/fab-setup migrations` subcommand, brew-install migration, `1.8.0-to-1.9.0` migration (tasks-stage collapse + plan.md), `1.9.1-to-1.9.2` migration (`true_impact_exclude` config field), version drift detection, `fab/.kit-migration-version` creation | 2026-05-07 |
| [hydrate-specs](hydrate-specs.md) | `/docs-hydrate-specs` skill — structural gap detection between memory and specs, interactive propose-then-apply | 2026-02-14 |
| [specs-index](specs-index.md) | `docs/specs/` directory — pre-implementation specs, distinction from memory, bootstrap and context integration | 2026-02-14 |
| [schemas](schemas.md) | `workflow.yaml` schema — 7-stage pipeline, states, transitions, validation rules, design principles; `.status.yaml` `plan:` block schema | 2026-05-06 |
