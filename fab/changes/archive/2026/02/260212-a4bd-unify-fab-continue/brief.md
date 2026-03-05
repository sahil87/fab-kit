# Brief: Unify Pipeline Commands into fab-continue

**Change**: 260212-a4bd-unify-fab-continue
**Created**: 2026-02-12
**Status**: Complete

## Origin

> Backlog: [a4bd], Linear: DEV-1014
> User requested: "/fab-new a4bd"

Rename a few more commands to fab-continue. fab-continue should be able to take it to the end (archive). Absorb fab-apply, fab-review and fab-archive also into fab-continue. Add a form in fab-continue to continue any specific stage — e.g., typing `fab-continue spec` should redo the move from brief to spec stage. This should improve DX as developers only need to remember fewer commands — fab-continue and fab-clarify mainly.

**Key decision from conversation**: Old skills (fab-apply, fab-review, fab-archive) will be **removed entirely**, not kept as aliases.

## Why

Developer experience friction: the current 6-stage pipeline exposes 4+ different commands (`fab-continue`, `fab-apply`, `fab-review`, `fab-archive`) plus `fab-ff`/`fab-fff` for fast-forwarding. Developers need to remember which command to run at each stage, and the "Next:" hints create a sense of command-hopping rather than a unified flow.

Unifying under `fab-continue` means developers primarily need two commands: `fab-continue` (advance) and `fab-clarify` (refine). This matches the mental model of "I'm working on a change; keep going."

## What Changes

- **Extend `fab-continue`** to handle all 6 pipeline stages (brief → spec → tasks → apply → review → archive), removing the guard that currently blocks at `tasks`
- **Absorb apply behavior**: When the active stage is `tasks` (done) or `apply`, `fab-continue` dispatches to the current fab-apply logic (task-by-task implementation with testing)
- **Absorb review behavior**: When the active stage is `apply` (done) or `review`, `fab-continue` dispatches to the current fab-review logic (validate against specs and checklists)
- **Absorb archive behavior**: When the active stage is `review` (done) or `archive`, `fab-continue` dispatches to the current fab-archive logic (hydrate learnings, move to archive)
- **Extend `fab-continue <stage>` reset mode** to accept all stages (not just `spec` and `tasks`), allowing `fab-continue apply` to re-run implementation, `fab-continue review` to re-run review, etc.
<!-- assumed: extending reset mode to all stages — backlog says "typing fab-continue spec should redo the move from brief to spec stage" which implies all stages should be targetable -->
- **Delete skill files**: Remove `fab-apply.md`, `fab-review.md`, `fab-archive.md` from `fab/.kit/skills/`
- **Update fab-ff and fab-fff** to invoke `fab-continue` internally instead of the removed skills
- **Update Next Steps Convention** in `_context.md` — simplify the lookup table since there's now one primary advancement command
- **Update all cross-references** in skills, docs, and design specs that reference the removed commands

## Affected Docs

### New Docs
_(none)_

### Modified Docs
- `fab-workflow/planning-skills`: Expand coverage of fab-continue to include execution stages (or merge with execution-skills)
- `fab-workflow/execution-skills`: Content to be absorbed into planning-skills doc, or renamed/restructured to reflect the unified command
- `fab-workflow/change-lifecycle`: Update stage transition descriptions to reference fab-continue uniformly

### Removed Docs
_(none — docs are restructured, not removed)_

## Impact

### Skills directly modified
- `fab/.kit/skills/fab-continue.md` — primary: absorb all stage logic
- `fab/.kit/skills/fab-ff.md` — update internal invocations
- `fab/.kit/skills/fab-fff.md` — update internal invocations
- `fab/.kit/skills/_context.md` — Next Steps Convention table
- `fab/.kit/skills/fab-new.md` — update "Next:" output lines
- `fab/.kit/skills/fab-clarify.md` — update "Next:" output lines
- `fab/.kit/skills/fab-status.md` — may reference removed commands

### Skills deleted
- `fab/.kit/skills/fab-apply.md`
- `fab/.kit/skills/fab-review.md`
- `fab/.kit/skills/fab-archive.md`

### Other files affected
- `fab/design/skills.md` — design spec references all skills individually
- `fab/design/user-flow.md` — command map diagrams
- `fab/design/overview.md` — quick reference
- `README.md` — command references
- `.claude/settings.local.json` — permission entries for removed commands

### Related Linear issues
- DEV-1014 (this ticket)
- Milestone: M5: Trial Fixes — Correctness & Ergonomics

## Open Questions

_(All resolved — no blocking questions remain)_

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | fab-ff/fff refactored to call fab-continue | They currently chain the individual skills; unified command is the natural replacement |
| 2 | Confident | Stage dispatch approach — fab-continue routes to existing logic per stage | Behavior stays the same, just the entry point changes |
| 3 | Tentative | Reset mode extended to all stages (apply, review, archive) | Backlog says "fab-continue spec should redo..." implying all stages are targetable, but only spec was explicitly named |

3 assumptions made (2 confident, 1 tentative). Run /fab-clarify to review.
