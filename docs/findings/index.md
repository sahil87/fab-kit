# Findings

> Investigation notes and design observations that aren't yet (or may never become) changes.
> A finding records something we noticed — a gap, an inconsistency, a limitation — with enough
> evidence and analysis that a future change can act on it without re-deriving the context.
>
> Findings are **human-curated and append-only in spirit**: mark a finding `resolved` (with a
> pointer to the change that closed it) rather than deleting it, so the reasoning trail survives.
> Contrast with `docs/memory/` (what shipped) and `docs/specs/` (what we planned) — findings are
> *what we noticed and haven't acted on yet*.

| Finding | Status | Summary |
|---------|--------|---------|
| [intake-is-the-context-boundary](intake-is-the-context-boundary.md) | open | Intake is the sole context boundary — main context ≤ intake, dispatched artifact-fed blocks > intake. Post-intake stages should have one execution mode (dispatched), collapsing the dual-mode `do NOT run fab status` seam and closing Gap 1a of the model-tier finding |
| [per-stage-model-tier-application](per-stage-model-tier-application.md) | open | Per-stage model tiers are honored only on the subagent-dispatch seam — foreground stages and skipped `resolve-agent` calls inherit the session model, and the Agent tool exposes no per-subagent `effort` knob |
