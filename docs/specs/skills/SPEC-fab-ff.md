# fab-ff

## Summary

Fast-forward apply → review → hydrate (everything after intake) in one invocation. Since 260611-szxd the skill file is a **thin wrapper over the shared pipeline bracket in `_pipeline.md`** (see `SPEC-_pipeline.md`): it declares Purpose, Arguments, and the two bracket parameters — `{driver}` = `fab-ff`, `{terminal}` = `hydrate` — plus its own Output block. The bracket owns the single intake confidence gate (flat 3.0, all types), context loading, resumability (incl. the review-`failed` recovery via `fab status start <change> review`), Steps 1–3, the auto-rework loop (`{max_cycles}`-cycle cap — the code-review.md Rework Budget knob, default 3 (c5tr); explicit per-cycle choreography, escalation after 2 consecutive fix-code), the exhaustion stop (terminal state: review `failed` — `/fab-continue`'s review-failed row presents the rework menu from there), and the shared error rows. No `/fab-clarify` runs inside the bracket — clarification is intake-only. All sub-skill invocations are dispatched as sub-agents with "do NOT run `fab status` commands; return results only" — the orchestrator owns those stages' transitions. Accepts `--force` to bypass the intake gate.

**Helpers**: Declares `helpers: [_generation, _review, _srad, _pipeline]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

## Flow

```
User invokes /fab-ff [change-name] [--force]
│
├─ Read: _preamble.md (always-load layer), helpers incl. _pipeline.md
│
└─ Execute the _pipeline.md bracket with {driver}=fab-ff, {terminal}=hydrate
   │  (see SPEC-_pipeline.md for the full bracket flow: pre-flight gate,
   │   Step 1 apply [plan co-gen + tasks], Step 2 review [inward + outward
   │   sub-agents via _review.md, auto-rework loop], Step 3 hydrate)
   │
   └─ {terminal}=hydrate → pipeline complete after Step 3
      (no ship/review-pr steps — those are /fab-fff's)
```

### Sub-agents

Defined by the bracket — see `SPEC-_pipeline.md`: `/fab-continue` (Apply), `/fab-continue` (Review — dispatches `_review.md`'s inward + outward sub-agents in parallel), `/fab-continue` (Hydrate).

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| pre | `fab score --check-gate --stage intake` | Before the bracket (intake gate) |
| 1 | PostToolUse hook recomputes plan counts (`plan.task_count`, `plan.acceptance_count`, `plan.acceptance_completed`); sets `plan.generated=true` | After plan.md write/edit |
