# Intake: Fix fab-continue Single-Dispatch Enforcement

**Change**: 260217-elxh-fix-continue-single-dispatch
**Created**: 2026-02-17
**Status**: Draft

## Origin

> Fix fab-continue auto-advancing through multiple stages (apply→review→hydrate) in a single invocation. Added a single-dispatch rule to the Normal Flow section that explicitly constrains execution to one stage per invocation, scoped to fab-continue only (not the shared Apply/Review Behavior sections referenced by fab-ff and fab-fff).

The issue was discovered during normal usage: when running `/fab-continue` from the `apply` stage, the LLM would complete apply, transition to review, execute review, transition to hydrate, and execute hydrate — all in one invocation. The skill's Purpose line said "one step at a time" but nothing in the Normal Flow enforced a hard stop after each stage dispatch.

## Why

`/fab-continue` is the incremental pipeline driver — users expect it to advance exactly one stage, then stop for review. When it auto-advances through apply→review→hydrate, the user loses the ability to inspect intermediate results, intervene between stages, or course-correct before hydration writes to `docs/memory/`. This undermines the deliberate, reviewable nature of the step-by-step workflow.

Without the fix, `/fab-continue` behaves like `/fab-ff` (pipeline execution) rather than its intended role as a single-stage advancer. The distinction between these skills collapses.

## What Changes

### Single-dispatch rule in `fab/.kit/skills/fab-continue.md`

A bolded rule block inserted after the stage dispatch table (after the `| all done |` row, before Step 2):

```markdown
**Single-dispatch rule**: Execute exactly ONE stage per invocation. After the dispatched
stage completes its work and transitions to the next stage (Step 4), proceed directly to
Step 5 (Output) and STOP. Do NOT loop back to re-evaluate the new current stage — the
user will run `/fab-continue` again to advance further.
```

### Scoping: Apply Behavior and Review Behavior sections left unchanged

The Apply Behavior and Review Behavior sections are referenced by `/fab-ff` and `/fab-fff` via "Execute apply behavior per `/fab-continue`". STOP instructions were intentionally NOT added to these shared sections to avoid breaking the pipeline skills, which are designed to auto-advance through all stages.

The single-dispatch rule is positioned in the Normal Flow section (Step 1), which is specific to `/fab-continue`'s own dispatch logic and not referenced by other skills.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document the single-dispatch rule as a key behavioral property of `/fab-continue`

## Impact

- **`fab/.kit/skills/fab-continue.md`** — one insertion (the single-dispatch rule block)
- **No code changes** — this is a prompt-level fix only
- **No impact on `/fab-ff` or `/fab-fff`** — they have their own step-by-step flow control and never reference fab-continue's Normal Flow section

## Open Questions

None — the fix is already implemented and the scoping rationale is clear.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The single-dispatch rule belongs in Normal Flow Step 1, not in Apply/Review Behavior | Apply/Review Behavior sections are shared by fab-ff/fab-fff which need to auto-advance; Normal Flow is fab-continue-specific | S:90 R:85 A:95 D:90 |
| 2 | Certain | No STOP annotations needed in Apply Behavior or Review Behavior | These sections are explicitly referenced by fab-ff (line 69) and fab-fff (line 81) via "per `/fab-continue`" — adding stops there would break pipeline skills | S:95 R:80 A:90 D:95 |
| 3 | Confident | The existing Purpose line ("one step at a time") was insufficient enforcement | LLMs treat purpose statements as soft guidance; a bolded rule with explicit DO NOT instructions is needed for hard behavioral constraint | S:85 R:90 A:70 D:80 |

3 assumptions (2 certain, 1 confident, 0 tentative, 0 unresolved).
