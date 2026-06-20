# fab-fff

## Summary

Full pipeline gated on the single intake gate (identical to fab-ff). Since 260611-szxd the skill file is a **thin wrapper over the shared pipeline bracket in `_pipeline.md`** (see `SPEC-_pipeline.md`) with the two bracket parameters — `{driver}` = `fab-fff`, `{terminal}` = `review-pr` — plus the fff-only Steps 4–5 (ship, review-pr), its own Output block, and the fff-only error rows. The bracket owns the gate, resumability, Steps 1–3, the auto-rework loop (`{max_cycles}`-cycle cap — the code-review.md Rework Budget knob, default 3 (c5tr); explicit per-cycle choreography, exhaustion terminal state review `failed`), and the shared error rows. Ship and review-pr are self-managed: `/git-pr` and `/git-pr-review` run their own stage transitions internally (the dispatch exception is noted in `fab-fff.md`); `/git-pr-review`'s timeout outcome deliberately leaves `review-pr` `active` and replaces `Pipeline complete.` with a re-run pointer. The Step 5 review-pr dispatch prompt bakes in a **synchronous-poll directive** (260615-qg64): the `/git-pr-review` subagent MUST complete the Copilot poll synchronously and not yield mid-poll, and the poll stays inside `/git-pr-review` (not relocated to the orchestrator) — mirroring `git-pr-review.md` Step 2 Phase 2's own discipline note into the dispatch seam (it stalled mid-poll 4× before). Since 260612-w7dp, Steps 4–5 dispatch with the **explicit change argument** (`/git-pr {name}`, `/git-pr-review {name}` — `{name}` is the change's folder name from preflight, chosen over the 4-char `{id}` because git-pr classifies any argument spelling a PR type word as a `<type>`, and a 4-char id can collide with `feat`/`docs`/`test`; a folder name never matches a type token) so the subagents resolve the pipeline's target as a transient override instead of self-resolving the active change — and their branch-matches-change guards verify the checked-out branch before mutating anything. Accepts `--force` to bypass the intake gate. **Per-stage model** (260613-l3ja, m3d4): the fff-only Steps 4–5 run `fab resolve-agent ship --alias` / `fab resolve-agent review-pr --alias` before dispatching `/git-pr` / `/git-pr-review` (the bracket handles Steps 1–3; `--alias` since 260613-yky7 — emits the Agent-tool-valid short alias on the `model=` line) — each site **surfaces** the resolved `model=/effort=` (visibility — a skip is detectable) and dispatches via two seams: model on the Agent `model` param (empty ⇒ omit/inherit) and effort as an imperative instruction in the dispatch prompt (no Agent effort param; omitted when empty; 260613-m3d4); see `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution.

**Helpers**: Declares `helpers: [_generation, _review, _srad, _pipeline]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

**Prose optimization** (260620-skop): skill content trimmed — the per-stage model recipe duplicated in the Step 4 and Step 5 preambles collapsed to one reference to the top-of-Behavior "Per-stage model" blockquote (matching `fab-ff.md`'s single-reference pattern), the Step 5 timeout outcome stated 3× reduced to one canonical literal in the Error Handling row (Step 5 and Output now reference it), and the synchronous-poll blockquote's rationale recap trimmed to a one-line pointer to `git-pr-review.md` while keeping the dispatch-seam MUST instruction itself; a `## Contents` TOC added. No behavioral change (Flow / Sub-agents / synchronous-poll directive / `{name}`-vs-`{id}` rule unchanged).

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
