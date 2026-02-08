# Proposal: Add fab-discuss for Conversational Proposal Development

**Change**: 260208-r4w1-add-fab-discuss
**Created**: 2026-02-08
**Status**: Draft

## Why

The current workflow has a gap between "I have an idea" and "I have a solid proposal." `/fab-new` captures a description and generates a proposal in one shot — good for clear changes, but it doesn't help you think through whether the change is even needed, explore alternatives, or iteratively refine scope. This session (discussing `fab-fff`) demonstrated the pattern: a back-and-forth conversation that sharpened a vague idea into a concrete, low-ambiguity proposal ready for `/fab-fff`. That pattern should be a first-class skill.

## What Changes

- **New `/fab-discuss` command**: An interactive, conversational skill for developing proposals. Works in two modes:
  - **From scratch**: Start with a vague idea, explore whether the change is needed (gap analysis), walk through clarifying questions, and output a solid `proposal.md` in a new change folder.
  - **On existing proposal**: Take an existing change's `proposal.md` and refine it through discussion — improving clarity, resolving tentative/unresolved decisions, sharpening scope.
- **Does not switch active change**: Unlike `/fab-new`, `fab-discuss` creates the change folder but does NOT update `fab/current`. The user must `/fab-switch` to it explicitly. This keeps the current work context undisturbed — you can discuss a future change while mid-implementation on another.
- **Computes confidence score**: Both `/fab-discuss` and `/fab-new` fill the `confidence` section in `.status.yaml` using the SRAD framework. The goal: after a thorough discussion, the confidence score should be high enough (>= 3.0) to go straight to `/fab-fff`.
- **Gap analysis step**: Before committing to a proposal, `fab-discuss` helps evaluate whether the change is needed — is there an existing mechanism, is the scope right, are there simpler alternatives? This is the "should we even do this?" phase that `/fab-new` skips.

## Affected Docs

### New Docs
- (none — fab-discuss will be documented via updates to existing docs)

### Modified Docs
- `fab-workflow/planning-skills.md`: Add `/fab-discuss` documentation alongside `/fab-new`
- `fab-workflow/change-lifecycle.md`: Add the discuss → fff path as an alternative entry point

### Removed Docs
(none)

## Impact

### Skill files
- `fab/.kit/skills/fab-discuss.md` — **New**: conversational proposal development skill
- `fab/.kit/skills/fab-new.md` — **Modified**: add confidence score computation (shared with `fab-discuss`; may already be added by the `fab-fff` change)
- `fab/.kit/skills/_context.md` — **Modified**: add `fab-discuss` to SRAD skill autonomy table, update Next Steps table

### Skill registration
- `.claude/skills/fab-discuss/prompt.md` — **New**: Claude Code skill registration

### Relationship to fab-fff change
This change depends on the confidence scoring design from `260208-k3m7-add-fab-fff`. The `confidence` field in `.status.yaml` and the SRAD-based scoring formula (start at 5.0, penalties per grade, unresolved = instant 0) are defined there. This change consumes that design — it does not redefine it.

## Key Design: fab-discuss vs fab-new vs fab-clarify

| Aspect | fab-discuss | fab-new | fab-clarify |
|--------|-------------|---------|-------------|
| **Purpose** | Explore & develop a proposal through conversation | Capture a clear description as a proposal | Refine an existing artifact's gaps |
| **Input** | Vague idea or existing proposal | Clear change description | Existing artifact with gaps |
| **Gap analysis** | Yes — "is this change even needed?" | No — assumes the change is needed | No — assumes the artifact exists |
| **Interaction style** | Free-form conversation with clarifying questions | One-shot generation with max 3 SRAD questions | Structured Q&A (max 5 per session) |
| **Sets active change** | No — must `/fab-switch` | Yes | N/A (operates on active change) |
| **Creates change folder** | Yes (if from scratch) | Yes | No |
| **Confidence goal** | Drive score high enough for `/fab-fff` | Compute initial score | Recompute score after refinements |

## Open Questions

(none)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | fab-discuss creates the change folder + .status.yaml but not fab/current | User explicitly said "doesn't switch to the change" |
| 2 | Confident | Gap analysis is a conversational step, not a structured checklist | The value is in free-form exploration ("should we even do this?"), not box-checking |
| 3 | Tentative | fab-discuss can be invoked with no arguments (starts from scratch) or with a change name (refines existing proposal) | User said "discuss anything (even an existing proposal)" — argument syntax is a reasonable design choice |
<!-- assumed: fab-discuss takes an optional change name argument to target an existing proposal -->
| 4 | Tentative | Git branch creation is deferred — fab-discuss doesn't create a branch since it doesn't switch context | Branching belongs to the "I'm working on this now" moment (fab-switch or fab-new), not the "let's think about this" moment |
<!-- assumed: fab-discuss skips git integration, branch created later when user switches to the change -->

4 assumptions made (2 confident, 2 tentative).
