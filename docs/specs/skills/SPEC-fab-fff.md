# fab-fff

## Summary

Full pipeline gated on the single intake gate (identical to fab-ff). Extends through ship and review-pr (fab-ff stops at hydrate). No `/fab-clarify` runs inside the bracket — clarification is intake-only. Max 3 rework cycles on review failure with escalation rule. Like fab-ff, the orchestrator owns the `fab status` transitions for the `/fab-continue`-behavior subagents (apply/review/hydrate): their prompts include "do NOT run `fab status` commands; return results only" (incl. the hydrate finish), the intake finish runs when `progress.intake` is not `done`, and a review-`failed` state is recovered via `fab status start <change> review` before resuming. Ship and review-pr are self-managed: `/git-pr` and `/git-pr-review` run their own stage transitions internally. Accepts `--force` to bypass the intake gate.

**Helpers**: Declares `helpers: [_generation, _review, _srad]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

## Flow

```
User invokes /fab-fff [change-name] [--force]
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ Gate: Intake Gate (skip if --force)
│  └─ Bash: fab score --check-gate --stage intake <change>
│     └─ STOP if < 3.0
│
├─ Steps 1-3: Same as /fab-ff Steps 1-3 (apply [plan.md co-gen + tasks], review, hydrate)
│  └─ Driver argument is "fab-fff" instead of "fab-ff". No in-bracket clarify.
│
├─ Step 4: Ship
│  └─ SUB-AGENT: /git-pr (commit, push, create PR)
│
└─ Step 5: Review-PR
   └─ SUB-AGENT: /git-pr-review (process PR review comments;
      manages its own review-pr stage transitions)
      ├─ [success / no-reviews] stage done
      ├─ [failure] STOP with the error
      └─ [timeout — Copilot review requested, not yet available]
         stage deliberately left active; report "Review-PR pending
         (Copilot review requested, timed out waiting) — re-run
         /git-pr-review when ready" instead of "Pipeline complete."
```

### Sub-agents

Same as fab-ff: /fab-continue (Apply, Review, Hydrate), /git-pr, /git-pr-review. No clarify sub-agent (intake-only, runs before the bracket).

> Step 2 review behavior (inward requirements + acceptance validation and outward holistic diff review) is defined in `_review.md`. `/fab-continue` Review Behavior delegates to `_review.md` — the authoritative source for inward + outward sub-agent dispatch and findings merge.

### Bookkeeping commands (hook candidates)

Same as fab-ff.
