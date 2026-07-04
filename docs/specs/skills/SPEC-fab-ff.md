# fab-ff

## Summary

Fast-forward apply → review → hydrate (everything after intake) in one invocation. Since 260611-szxd the skill file is a **thin wrapper over the shared pipeline bracket in `_pipeline.md`** (see `SPEC-_pipeline.md`): it declares Purpose, Arguments, and the two bracket parameters — `{driver}` = `fab-ff`, `{terminal}` = `hydrate` — plus its own Output block. The bracket owns the single intake confidence gate (flat 3.0, all types), context loading, resumability (incl. the review-`failed` recovery via `fab status start <change> review`), Steps 1–3, the auto-rework loop (`{max_cycles}`-cycle cap — the code-review.md Rework Budget knob, default 3 (c5tr); explicit per-cycle choreography, escalation after 2 consecutive fix-code), the exhaustion stop (terminal state: review `failed` — `/fab-continue`'s review-failed row presents the rework menu from there), and the shared error rows. No `/fab-clarify` runs inside the bracket — clarification is intake-only. All sub-skill invocations are dispatched as sub-agents carrying the refined **block-contract carve-out** (aetz/3d): do NOT run `fab status` *transition* commands (`start`/`advance`/`finish`/`reset`/`fail`/`skip`); return results only — but DO end with a terminal `fab status refresh` (a pull-based recompute, not a transition — the orchestrator still owns those stages' transitions). Each stage dispatch in the bracket first resolves `fab resolve-agent <stage> --alias` (260613-l3ja; `--alias` since 260613-yky7 — emits the Agent-tool-valid short alias on the `model=` line), **surfaces** the resolved `model=/effort=/provider=/dispatch=` (visibility — a skip, or a CLI dispatch, is detectable, not silent; 260613-m3d4, dispatch line added 260702-aetz, renamed `spawn=`→`dispatch=` in 260702-tykw), then **branches on the resolved `dispatch=` line** (aetz): absent ⇒ native Agent-tool dispatch via two seams — model on the Agent `model` param (empty ⇒ omit/inherit) and effort as an imperative instruction in the dispatch prompt (no Agent effort param; omitted when empty; 260613-m3d4); present ⇒ the CLI adapter (`fab dispatch`) where the profile rides the `dispatch=` command so the two seams do not apply — see `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution and § CLI-Adapter Dispatch. Accepts `--force` to bypass the intake gate. The wrapper's `{driver}` parameter row scopes the driver claim to the commands the bracket shows it on — the fail/recovery commands are deliberately driver-less (260612-w7dp).

**Helpers**: Declares `helpers: [_generation, _review, _srad, _pipeline]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

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
