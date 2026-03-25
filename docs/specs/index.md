# Specs Index

> **Specs are pre-implementation artifacts** — what you *planned*. They capture conceptual
> design intent, high-level decisions, and the "why" behind features. Specs are human-curated,
> flat in structure, and deliberately size-controlled for quick reading.
>
> Contrast with [`docs/memory/index.md`](../memory/index.md): memory files are *post-implementation* —
> what actually happened. Memory is the authoritative source of truth for system behavior,
> maintained by `/fab-continue` (hydrate).
>
> **Ownership**: Specs are written and maintained by humans. No automated tooling creates or
> enforces structure here — organize files however makes sense for your project.

> **New here?** Start with the [README](../../README.md) for setup and a walkthrough. For terminology, see the [Glossary](glossary.md).

| Spec | Description |
|------|-------------|
| [overview](overview.md) | Fab workflow specification — background, design principles, 6 stages, quick reference |
| [architecture](architecture.md) | Directory structure, config, naming conventions, git integration, agent integration |
| [skills](skills.md) | Detailed behavior for each `/fab-*` skill |
| [templates](templates.md) | Artifact templates — status, intake, spec, tasks, checklist, memory files |
| [user-flow](user-flow.md) | Visual diagrams — how development works today, with Fab commands, full command map |
| [srad](srad.md) | SRAD autonomy framework — scoring dimensions, confidence grades, confidence scoring, gating, worked examples |
| [change-types](change-types.md) | Change type taxonomy — 7 types, expected_min thresholds, gate thresholds, PR tiers, keyword heuristics |
| [packages](packages.md) | Bundled packages — wt (worktree management) and idea (backlog management) |
| [naming](naming.md) | Naming conventions — change folders, branches, worktrees, PRs, backlog entries |
| [glossary](glossary.md) | All Fab terminology — core concepts, stages, skills, files, SRAD, conventions |
| [superpowers-comparison](superpowers-comparison.md) | Comparison with Superpowers — shared ground, key differences, lessons for fab |
| [skills/](skills/) | Per-skill flow diagrams — summary, tool usage, sub-agents, hooks, and bookkeeping candidates |
