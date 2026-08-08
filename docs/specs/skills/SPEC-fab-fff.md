# fab-fff

## Summary

**Source organization:** The wrapper binds `fab-fff`/`review-pr`; shared framing lives in `_pipeline`, while ship/review-pr stay local.

Run the full apply → review → hydrate → ship → review-pr pipeline. The wrapper binds `{driver}=fab-fff` and `{terminal}=review-pr`; `_pipeline.md` owns shared framing and Steps 1–3. This skill owns ship/review-pr, their explicit folder-name arguments, the synchronous Copilot-poll directive, timeout behavior, additional output sections, and fff-only errors.

**Helpers**: Declares `helpers: [_generation, _review, _srad, _pipeline]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

**Dispatch prose contract**: Steps 1–3 point to `_pipeline.md` § Stage Dispatch
Procedure and `_preamble.md` § CLI-Adapter Dispatch. The fff-only
ship/review-pr steps dispatch full skill behaviors through the native seams.

**Prose optimization** (260620-skop): shared driver framing and the Steps 1–3 dispatch procedure live in `_pipeline.md`; fab-fff points to those owners and keeps only its ship/review-pr native-dispatch delta. Its local Output section retains the exact `## Assumptions (cumulative)` heading, `Artifact` column requirement, extra Ship/Review-PR sections, and timeout substitution. The synchronous-poll MUST directive and `{name}`-vs-`{id}` rule remain local; no behavior changes.

## Flow

```
User invokes /fab-fff [change-name] [--force]
│
├─ Read: _preamble.md (always-load layer), helpers incl. _pipeline.md
│
├─ Execute the _pipeline.md bracket with {driver}=fab-fff, {terminal}=review-pr
│  (see SPEC-_pipeline.md: pre-flight gate, Step 1 apply, Step 2 review with
│   auto-rework loop, Step 3 hydrate)
│
├─ Step 4: Ship
│  └─ SUB-AGENT: /git-pr {name} (explicit change argument — folder
│     name, never the type-word-collidable 4-char id; transient
│     override + branch guard, 260612-w7dp; commit, push, create PR;
│     manages its own ship-stage transitions)
│
└─ Step 5: Review-PR
   └─ SUB-AGENT: /git-pr-review {name} (explicit change argument,
      same contract; process PR review comments;
      manages its own review-pr stage transitions)
      [dispatch prompt bakes in: complete the Copilot poll
       SYNCHRONOUSLY — do NOT yield mid-poll; poll stays inside
       /git-pr-review, not relocated to the orchestrator (260615-qg64)]
      ├─ [success / no-reviews] stage done
      ├─ [failure] STOP with the error
      └─ [timeout — Copilot review requested, not yet available]
         stage deliberately left active; report "Review-PR pending
         (Copilot review requested, timed out waiting) — re-run
         /git-pr-review {name} when ready" instead of "Pipeline
         complete." (re-run guidance names the change — the run
         may be driving a non-active override, 260612-w7dp)
```

### Sub-agents

Bracket sub-agents per `SPEC-_pipeline.md` (/fab-continue Apply, Review, Hydrate) plus /git-pr and /git-pr-review. No clarify sub-agent (intake-only, runs before the bracket).

### Bookkeeping commands (hook candidates)

Same as fab-ff (see `SPEC-fab-ff.md`); ship/review-pr transitions are run internally by /git-pr and /git-pr-review.
