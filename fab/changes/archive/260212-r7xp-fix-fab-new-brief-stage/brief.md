# Brief: Fix fab-new premature brief completion

**Change**: 260212-r7xp-fix-fab-new-brief-stage
**Created**: 2026-02-12
**Status**: Draft

## Origin

> fab-new frequently causes stage brief to be done. Step 4 says to initialize with brief: active and all others pending. Step 8 says to transition to brief: done / spec: active only "once the user is satisfied". Remove Step 8 from /fab-new — /fab-new always ends with brief: active. Ensure /fab-continue handles the brief → spec transition — which it likely already does, since that's its job.

## Why

`/fab-new` Step 8 ("Mark Brief Complete") transitions `progress.brief` to `done` and `progress.spec` to `active` within the same skill invocation that generates the brief. In practice, the agent executes Step 8 immediately after Step 7 without waiting for explicit user satisfaction, causing the brief stage to be marked complete prematurely. This bypasses the intended workflow where the user reviews the brief and uses `/fab-continue` or `/fab-clarify` to advance or refine.

## What Changes

- **Remove Step 8** ("Mark Brief Complete") from `fab/.kit/skills/fab-new.md` entirely — the stage transition logic (`brief: done`, `spec: active`) and `last_updated` write belong in `/fab-continue`
- **Renumber Step 9 → Step 8** (Activate Change via `/fab-switch`)
- **Update the _context.md Next Steps table** — change `/fab-new` stage reached from "brief done" to "brief active"
- **Update output examples** in `fab-new.md` if any imply brief is marked done

## Affected Docs

### New Docs
_(none)_

### Modified Docs
- `fab-workflow/planning-skills`: The description of `/fab-new`'s behavior will change (brief stays active, no longer transitions)
- `fab-workflow/context-loading`: The Next Steps lookup table entry for `/fab-new` shows "brief done" — needs updating to "brief active"

### Removed Docs
_(none)_

## Impact

- **`fab/.kit/skills/fab-new.md`** — primary change: remove Step 8, renumber Step 9
- **`fab/.kit/skills/_context.md`** — update Next Steps lookup table entry for `/fab-new`
- **`/fab-continue`** — already handles `brief → spec` transition (line 54: "Generate `spec.md` → set `brief: done`, `spec: active`") — no changes needed
- **`/fab-ff` and `/fab-fff`** — these invoke `/fab-continue` internally, so they inherit the fix

## Open Questions

_(none — all decisions resolved from context)_

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Step 7 (confidence write) stays unchanged | Step 7 writes the confidence block to `.status.yaml` independent of stage transitions — it doesn't touch `progress.*` fields |
| 2 | Confident | Output examples in fab-new.md don't need content changes | The "Brief complete." text and Next lines already point to `/fab-continue`, which is correct regardless of whether brief is active or done |

2 assumptions made (2 confident, 0 tentative).
