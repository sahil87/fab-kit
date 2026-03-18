# Intake: Standard Subagent Context Template

**Change**: 260318-dzze-standard-subagent-context
**Created**: 2026-03-18
**Status**: Draft

## Origin

> Discussion session (`/fab-discuss`) exploring whether review and execution subagents receive `fab/project/**` files (constitution.md, code-quality.md, code-review.md, context.md, config.yaml). Investigation revealed:
>
> 1. `fab-continue.md` Review Behavior (line 153) explicitly lists these files as context for the review sub-agent
> 2. But the dispatch chain is indirect: `fab-fff` → subagent (fab-continue) → sub-sub-agent (review validation) — project files must flow through two levels of delegation
> 3. No standard template exists in `_preamble.md` to enforce this — it depends on each middle agent faithfully constructing the inner prompt
> 4. User confirmed: all `fab/project/**` files should go to ALL subagents, not selectively — the files are ~150 lines total, negligible context cost, and cross-pollination is beneficial

## Why

Subagent context loading for `fab/project/**` files is currently ad-hoc. `fab-continue.md` Review Behavior lists the files explicitly (line 153), but there's no standard template in `_preamble.md`'s Subagent Dispatch section. This creates two problems:

1. **Silent quality gaps**: If an orchestrator or middle agent forgets to pass a file, the subagent operates without project principles — and nothing flags the omission
2. **Drift risk**: Each skill that dispatches subagents maintains its own context list, which can fall out of sync with `fab/project/**` as new files are added

If unfixed, subagents may generate code or reviews that violate project principles defined in constitution.md, ignore code-quality.md anti-patterns, or skip code-review.md severity definitions — all silently.

## What Changes

### 1. Add "Standard Subagent Context" to `_preamble.md` Subagent Dispatch

Add a new subsection under the existing "Subagent Dispatch (Orchestrator Skills)" section (after the current dispatch pattern list, before SRAD). The template mandates that every subagent prompt includes:

- `fab/project/config.yaml`
- `fab/project/constitution.md`
- `fab/project/context.md` (if present)
- `fab/project/code-quality.md` (if present)
- `fab/project/code-review.md` (if present)

The template should specify that the dispatching agent instructs the subagent to **read these files first** before executing its task. This is item 6 in the existing dispatch pattern numbered list.

### 2. Simplify `fab-continue.md` Review Behavior context list

Replace the ad-hoc file list at line 153 with a reference to the preamble's standard context. The review-specific context (spec.md, tasks.md, checklist.md, relevant source files, target memory files) remains listed explicitly — only the `fab/project/**` portion is replaced with a reference.

### 3. Update `fab-continue.md` Apply Behavior `code-quality.md` reference

Apply Behavior (line 113) currently says "If `fab/project/code-quality.md` exists, load its `## Principles`..." — this is now redundant since the standard context already loads it. Add a note that this file is available via standard subagent context, keeping the specific section reference (`## Principles`, `## Test Strategy`) for what to extract from it.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document standard subagent context template and its effect on review/apply dispatch
- `fab-workflow/context-loading`: (modify) Add standard subagent context as a defined loading layer

## Impact

- `fab/.kit/skills/_preamble.md` — primary change: new subsection in Subagent Dispatch
- `fab/.kit/skills/fab-continue.md` — simplify Review Behavior context list, note in Apply Behavior
- `docs/specs/skills/SPEC-preamble.md` — create new (no existing spec for the internal partial)
- `docs/specs/skills/SPEC-fab-continue.md` — update (exists)

No changes to `fab-ff.md` or `fab-fff.md` — they already reference `_preamble.md § Subagent Dispatch` and will inherit the template automatically.

## Open Questions

None remaining — all resolved via `/fab-clarify`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All 5 `fab/project/**` files loaded in every subagent | Discussed — user explicitly confirmed "safer if all of fab/project/** is read by all sub-agents" | S:95 R:90 A:95 D:95 |
| 2 | Certain | Template goes in `_preamble.md` Subagent Dispatch section | Discussed — user confirmed "A template in preamble makes sense" as the location | S:95 R:85 A:90 D:95 |
| 3 | Certain | `fab-continue.md` Review Behavior context list simplified to reference preamble | Discussed — this was part of the original proposal user agreed to | S:90 R:85 A:90 D:90 |
| 4 | Certain | No changes needed to `fab-ff.md` or `fab-fff.md` | Clarified — verified no ad-hoc context lists exist in either file, only `_preamble.md` references | S:95 R:90 A:95 D:95 |
| 5 | Confident | Template specifies "read these files first" instruction pattern | Subagents need explicit file-read instructions, not just a list — ensures files are actually loaded | S:70 R:80 A:75 D:80 |
| 6 | Certain | `docs/memory/index.md` and `docs/specs/index.md` excluded from standard subagent context | Clarified — user confirmed; standard context = project principles only, not documentation navigation | S:95 R:85 A:95 D:95 |
| 7 | Certain | Create `docs/specs/skills/SPEC-preamble.md` as part of this change | Clarified — user requested; `SPEC-preamble.md` does not exist yet, `SPEC-fab-continue.md` does | S:95 R:85 A:90 D:95 |

7 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-03-18

| # | Action | Detail |
|---|--------|--------|
| 6 | Confirmed | Exclude `docs/memory/index.md` and `docs/specs/index.md` — standard context = project principles only |
| 4 | Confirmed | Verified `fab-ff.md` and `fab-fff.md` have no ad-hoc context lists — only `_preamble.md` references |
| 7 | Added | User requested `docs/specs/skills/SPEC-preamble.md` creation as part of this change |
