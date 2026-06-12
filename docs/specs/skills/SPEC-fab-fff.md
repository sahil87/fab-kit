# fab-fff

## Summary

Full pipeline gated on the single intake gate (identical to fab-ff). Since 260611-szxd the skill file is a **thin wrapper over the shared pipeline bracket in `_pipeline.md`** (see `SPEC-_pipeline.md`) with the two bracket parameters — `{driver}` = `fab-fff`, `{terminal}` = `review-pr` — plus the fff-only Steps 4–5 (ship, review-pr), its own Output block, and the fff-only error rows. The bracket owns the gate, resumability, Steps 1–3, the auto-rework loop (`{max_cycles}`-cycle cap — the code-review.md Rework Budget knob, default 3 (c5tr); explicit per-cycle choreography, exhaustion terminal state review `failed`), and the shared error rows. Ship and review-pr are self-managed: `/git-pr` and `/git-pr-review` run their own stage transitions internally (the dispatch exception is noted in `fab-fff.md`); `/git-pr-review`'s timeout outcome deliberately leaves `review-pr` `active` and replaces `Pipeline complete.` with a re-run pointer. Since 260612-w7dp, Steps 4–5 dispatch with the **explicit change argument** (`/git-pr {name}`, `/git-pr-review {name}` — `{name}` is the change's folder name from preflight, chosen over the 4-char `{id}` because git-pr classifies any argument spelling a PR type word as a `<type>`, and a 4-char id can collide with `feat`/`docs`/`test`; a folder name never matches a type token) so the subagents resolve the pipeline's target as a transient override instead of self-resolving the active change — and their branch-matches-change guards verify the checked-out branch before mutating anything. Accepts `--force` to bypass the intake gate.

**Helpers**: Declares `helpers: [_generation, _review, _srad, _pipeline]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

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
