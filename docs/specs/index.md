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
| [templates](templates.md) | Artifact templates — status, intake, plan (## Requirements + ## Tasks + ## Acceptance), memory files |
| [user-flow](user-flow.md) | Visual diagrams — how development works today, with Fab commands, full command map |
| [srad](srad.md) | SRAD autonomy framework — four dimensions, composite (0.20 S / 0.30 R / 0.30 A / 0.20 D), indicative grades, demerit confidence score (5 − Σ penalty), flat 3.0 gate, lifecycle, Critical Rule, worked examples, autonomy levels |
| [srad-scoring-rationale-v1-to-v2](srad-scoring-rationale-v1-to-v2.md) | Design rationale for the v1→v2 scoring change ([srad-v1](srad-v1.md) → [srad](srad.md)) — why the mean leaked, the four rejected aggregators, the penalty-curve derivation, cross-repo validation (loom/fab-kit/run-kit, 1080 intakes), the promptless-defer decision, and the implementation surface |
| [srad-v1](srad-v1.md) | **Superseded** — the original SRAD scheme (grade-count / Resolution-Average score, hand-written grades, Unresolved & R<25∧A<25 hard-fails). Retained for historical reference and migrated changes only |
| [change-types](change-types.md) | Change type taxonomy — 7 types, expected_min thresholds, gate thresholds, PR tiers, keyword heuristics |
| [stage-models](stage-models.md) | Per-stage model selection via agent roles — two advertised depth knobs (`agent.session`/`agent.workers`) over six fixed roles (`default`/`operator`/`doing`/`review`/`hydrate`/`fast`), a top-level `providers:` table carrying independent `interactive_command`/`headless_command`/`native` capabilities plus per-role fills, the `dispatch.mode` preference ceiling and descending `pane → native → headless` adapter ladder, the fixed fab-owned stage→role mapping and role→depth partition, the sparse `agent.profiles` override, `fab resolve-agent <stage\|role>`, verbatim pass-through (provider-neutral), the effort asymmetry between the native and CLI arms, default role profiles |
| [config](config.md) | Config schema and six-verb surface: `show [key]`, `explain [key]` (`reference` alias), surgical `set`/`unset`, bare-project `init`, and whole-file `upgrade`; per-field metadata (`default`/`description`/`scope`/`advertise`/`renamed_from`/`init-seed`), the four-tier cascade (environment > system `~/.fab-kit/config.yaml` > project > built-in defaults, per-leaf deep merge with empty-skip, fail-open scope enforcement), managed fence, presence=intent, and the `fab/.fab-version` relocation |
| [harness-adapters](harness-adapters.md) | Cross-harness stage-dispatch contract — the three dispatch adapters (native Agent-tool + headless CLI `fab dispatch start` + interactive pane `fab dispatch start --pane`), the shared protocol (dispatch-prompt obligations binding all three incl. the `fab status refresh` epilogue, the five-state machine `running`/`done`/`failed`/`failed (no-result)`/`orphaned` of which the pane adapter reaches only the three-state subset `running`/`done`/`orphaned`, hooks-enhance-never-own), `.fab-dispatch/{id}/` cleanup (archive + `clean`, no auto-GC), and the 3c-authors / 3d-wires split |
| [fkf](fkf.md) | Fab Knowledge Format — design rationale + history companion (OKF lineage, bundle organization, non-scope, adoption/migration, glossary); the normative standard lives at `docs/site/fkf.md`, published at shll.ai/fab-kit/fkf and shipped verbatim as `$(fab kit-path)/reference/fkf.md`, synced by `scripts/sync-fkf.sh` + a CI drift-guard test |
| [hooks](hooks.md) | Claude Code hooks in fab-kit — fab registers, writes, and owns none; the `fab hook` family removed outright in 2.14.0, agent state consumed from run-kit's `@rk_agent_state` pane-option convention, artifact bookkeeping pull-based via `fab status refresh` |
| [companions](companions.md) | Companion CLIs — how fab-kit integrates with wt (worktree isolation) and idea (backlog) |
| [naming](naming.md) | Naming conventions — change folders, branches, worktrees, PRs, backlog entries |
| [glossary](glossary.md) | All Fab terminology — core concepts, stages, skills, files, SRAD, conventions |
| [superpowers-comparison](superpowers-comparison.md) | Comparison with Superpowers — shared ground, key differences, lessons for fab |
| [skills/](skills/) | Per-skill flow diagrams — summary, tool usage, sub-agents, hooks, and bookkeeping candidates |
