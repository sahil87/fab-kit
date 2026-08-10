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
| [kimi-vs-codex-rvza-ttff](kimi-vs-codex-rvza-ttff.html) | open | Head-to-head experiment (2026-08-10): rvza + ttff each piped to a draft PR on all-kimi (k3) vs all-codex (sol@xhigh) workers — kimi ~2.5× faster on both, codex higher-quality on both (decisively on the sweep-heavy ttff: sweep 10 vs 6); sweep-verification greps should target phrase classes, not tokens |
| [kimi-codex-claude-rvza-ttff](kimi-codex-claude-rvza-ttff.html) | open | Three-provider extension (2026-08-10, PRs #570–573/#575–576): claude (opus-5@high) added — wins rvza outright (9.20), near-ties codex on ttff (9.10 vs 9.45); kimi still ~2.6× faster than both; new diff-size metric: codex leanest, claude largest (+51–55% vs codex, substance not churn); contrastive-phrase sweep-miss class found (`unlike X` carries a retired claim no literal grep sees) |
