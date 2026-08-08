# fab-ff

## Summary

**Source organization:** The wrapper binds `fab-ff`/`hydrate`; shared framing, output skeleton, and Steps 1–3 live in `_pipeline`.

Fast-forward apply → review → hydrate in one invocation. The wrapper binds `{driver}=fab-ff` and `{terminal}=hydrate`; `_pipeline.md` owns shared purpose, arguments, output skeleton, gate, context loading, resumability, Steps 1–3, rework, and shared errors. The wrapper accepts `--force`, supplies its header, and points once to the shared Stage Dispatch Procedure and `_preamble.md` dispatch canon.

**Helpers**: Declares `helpers: [_generation, _review, _srad, _pipeline]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

**Dispatch prose contract**: the wrapper points to `_pipeline.md` § Stage Dispatch
Procedure and `_preamble.md` § CLI-Adapter Dispatch; it does not restate them.

## Flow

```
User invokes /fab-ff [change-name] [--force]
│
├─ Read: _preamble.md (always-load layer), helpers incl. _pipeline.md
│
└─ Execute the _pipeline.md bracket with {driver}=fab-ff, {terminal}=hydrate
   │  (see SPEC-_pipeline.md for the full bracket flow: pre-flight gate,
   │   Step 1 apply [plan co-gen + tasks], Step 2 review [single review
   │   sub-agent via _review.md, auto-rework loop], Step 3 hydrate)
   │
   └─ {terminal}=hydrate → pipeline complete after Step 3
      (no ship/review-pr steps — those are /fab-fff's)
```

### Sub-agents

Defined by the bracket — see `SPEC-_pipeline.md`: `/fab-continue` (Apply), `/fab-continue` (Review — dispatches `_review.md`'s single review sub-agent), `/fab-continue` (Hydrate).

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| pre | `fab score --check-gate --stage intake` | Before the bracket (intake gate) |
| 1 | `fab status refresh` recomputes plan counts (`plan.task_count`, `plan.acceptance_count`, `plan.acceptance_completed`); sets `plan.generated=true` | Self-healed at advance/finish/preflight after plan.md write/edit |
