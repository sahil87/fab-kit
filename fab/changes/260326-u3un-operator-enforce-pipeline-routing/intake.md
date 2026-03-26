# Intake: Operator Enforce Pipeline Routing

**Change**: 260326-u3un-operator-enforce-pipeline-routing
**Created**: 2026-03-26
**Status**: Draft

## Origin

> Update fab-operator7.md skill: add guidance that the operator must always route new work through the fab pipeline (fab-new then fab-fff) when spawning agents, never dispatch raw inline task instructions

One-shot request. No prior conversation context.

## Why

The operator7 skill's §6 "Working a Change" describes three paths for handling work (backlog/Linear, raw text, existing change), and each path correctly shows routing through `/fab-new`. However, there is no explicit principle-level prohibition against dispatching raw inline task instructions to agents. Without this explicit guidance, an operator (especially one recovering from a `/clear` or operating under time pressure) could shortcut by sending freeform implementation instructions directly to an agent pane — bypassing intake generation, confidence scoring, and the full pipeline. This would violate the fab workflow's core value proposition: specification-driven development with traceability.

The fix is a small but important addition: an explicit rule in §1 (Principles) or §6 that the operator MUST always route new work through the fab pipeline (`/fab-new` → `/fab-fff` or equivalent), and MUST NOT dispatch raw inline task instructions that bypass the pipeline.

## What Changes

### Addition to `fab/.kit/skills/fab-operator7.md`

Add explicit guidance in the skill file. Two placement options:

1. **§1 Principles** — add a new principle (e.g., "Pipeline-first routing") that states the operator must always route new work through `/fab-new` then `/fab-fff` (or `/fab-ff`, `/fab-continue`), never send raw implementation instructions to agent panes.

2. **§6 Coordination Patterns** — add a highlighted rule at the top of "Working a Change" subsection reinforcing that all three work paths must go through the pipeline.

The guidance should cover:
- The operator MUST route all new work through `/fab-new` (to generate intake) then a pipeline command (`/fab-fff`, `/fab-ff`, or `/fab-continue`)
- The operator MUST NOT send raw implementation instructions (e.g., "fix the login bug by changing line 42 in auth.ts") directly to agent panes
- The operator MUST NOT send `/fab-continue` to skip intake — `/fab-new` is always the entry point for new work
- Exception: operational maintenance commands (merge, archive, rebase, `/git-branch`, `/fab-switch`) are coordination-level and remain direct

### Constitution alignment

This reinforces Constitution §II (Docs Are Source of Truth) — the intake artifact is the source of truth for what a change does. Bypassing it means no traceability.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update operator7 section to document the pipeline-first routing principle

## Impact

- `fab/.kit/skills/fab-operator7.md` — primary change target
- `docs/specs/skills/` — may need a corresponding spec update if one exists for operator7 (none found; operator4 and operator5 have specs)
- No code changes — this is a docs/skill-file-only change

## Open Questions

(none — scope is clear)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add to §1 Principles as a new principle | Principles section is the authoritative place for operator behavioral rules — §6 describes mechanics, §1 describes invariants | S:80 R:90 A:85 D:85 |
| 2 | Confident | Also add a reinforcing note in §6 "Working a Change" | Both sections are read by the operator; belt-and-suspenders prevents drift. Could be §1-only but dual placement is low-cost | S:70 R:90 A:75 D:65 |
| 3 | Certain | Operational maintenance commands remain exempt | Already established in §1 "Coordinate, don't execute" — maintenance is explicitly carved out | S:85 R:95 A:90 D:90 |
| 4 | Certain | No new spec file needed for operator7 | No existing SPEC-fab-operator7.md; operators 6 and 7 lack dedicated specs. Constitution says skill changes update corresponding specs — but none exists to update | S:80 R:95 A:85 D:85 |
| 5 | Confident | Memory file update covers the change | execution-skills.md already has operator7 section; this adds a behavioral note | S:75 R:90 A:80 D:70 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
